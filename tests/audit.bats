#!/usr/bin/env bats
# audit.bats - kit audit: the forge's attestations, alerts, controls, and their waivers, and the policy's rules
# ----------------------------------------------------------------------------------------------------------------------
# Each test runs with HOME, XDG_CONFIG_HOME, and XDG_CACHE_HOME of its own, as helper:isolate sets them. A repository's
# origin is a GitHub URL that nothing fetches: the fake gh answers for it.

bats_require_minimum_version 1.5.0

load helper

setup() {
  hide-system-toolchains
}

# github-world makes a repository whose origin is GitHub's, and the fake gh's answers: the release v0.1.0 offers
# kit-0.1.0.tar.gz, digest sha256:ab12, which the release workflow attested as SLSA provenance; code scanning's alert
# 4 is open; ada dismissed Dependabot's alert 1 as a tolerable risk; secret scanning is off; and the ruleset main
# guards the default branch with two rules, which the team release may bypass.
github-world() {
  local statement alerts='sort=updated&direction=desc&per_page=100'
  mkdir -p "${BATS_TEST_TMPDIR}/r"
  git init --quiet --initial-branch=main "${BATS_TEST_TMPDIR}/r"
  cd "${BATS_TEST_TMPDIR}/r"
  git remote add origin https://github.com/o/r.git
  fake-tools gh
  fixture user.json <<<'{"login": "ada"}'
  fixture repo.json <<<'{"nameWithOwner": "o/r", "url": "https://github.com/o/r"}'
  fixture releases.json <<'JSON'
[{"tag_name": "v0.1.0", "assets": [{"name": "kit-0.1.0.tar.gz", "digest": "sha256:ab12",
  "created_at": "2026-10-01T10:00:00Z"}]}]
JSON
  fixture artifacts.json <<<'{"total_count": 0, "artifacts": []}'
  statement=$(jq --null-input --raw-output '{predicateType: "https://slsa.dev/provenance/v1",
    predicate: {buildDefinition: {externalParameters: {workflow: {ref: "refs/tags/v0.1.0",
      repository: "https://github.com/o/r", path: ".github/workflows/release.yml"}}}}} | tojson | @base64')
  jq --null-input --arg payload "${statement}" \
    '[{attestations: [{initiator: "user", bundle: {dsseEnvelope: {payload: $payload}}}]}]' \
    | fixture attestations.json
  fixture code-open.json <<'JSON'
[[{"number": 4, "state": "open", "created_at": "2026-10-01T09:00:00Z",
   "html_url": "https://github.com/o/r/security/code-scanning/4",
   "rule": {"id": "js/xss", "severity": "error", "security_severity_level": "high"},
   "most_recent_instance": {"location": {"path": "src/app.js"}}}]]
JSON
  fixture none.json <<<'[[]]'
  fixture dependabot.json <<'JSON'
[[{"number": 1, "state": "dismissed", "created_at": "2026-09-15T09:00:00Z",
   "html_url": "https://github.com/o/r/security/dependabot/1",
   "dependency": {"package": {"name": "lodash"}, "manifest_path": "package-lock.json"},
   "security_advisory": {"ghsa_id": "GHSA-35jh-r3h4-6jhm", "severity": "high"},
   "security_vulnerability": {"severity": "high"},
   "dismissed_by": {"login": "ada"}, "dismissed_reason": "tolerable_risk", "dismissed_comment": null}]]
JSON
  fixture secret-off.txt <<<'gh: Secret scanning is disabled on this repository. (HTTP 404)'
  fixture rulesets.json <<'JSON'
{"data": {"repository": {"url": "https://github.com/o/r", "defaultBranchRef": {"name": "main"},
  "rulesets": {"totalCount": 1, "nodes": [
    {"databaseId": 7, "name": "main", "enforcement": "ACTIVE",
     "conditions": {"refName": {"include": ["~DEFAULT_BRANCH"], "exclude": []}},
     "rules": {"totalCount": 2, "nodes": [{"type": "DELETION"}, {"type": "PULL_REQUEST"}]},
     "bypassActors": {"totalCount": 1, "nodes": [{"bypassMode": "ALWAYS", "actor": {"__typename": "Team",
       "slug": "release"}}]}}]}}}}
JSON
  fixture branch.json <<<'{"name": "main", "protection": {"enabled": false}}'
  printf 'HTTP/2.0 200 OK\r\nETag: W/"m1"\r\n\r\n[]\n' > "${FAKE_DIR}/marks-200.txt"
  route gh 'api user*' user.json
  route gh 'repo view*' repo.json
  route gh 'api --include repos/{owner}/{repo}/dependabot/alerts?state=*' "${DEPENDABOT_HEAD:-marks-200.txt}"
  route gh 'api --include*' marks-200.txt
  route gh 'api repos/{owner}/{repo}/releases?per_page=6' releases.json
  route gh 'api repos/{owner}/{repo}/actions/artifacts?per_page=20' artifacts.json
  route gh 'api --paginate --slurp repos/{owner}/{repo}/attestations/sha256:ab12?per_page=100' attestations.json
  route gh "api --paginate --slurp repos/{owner}/{repo}/code-scanning/alerts?state=open&${alerts}" code-open.json
  route gh "api --paginate --slurp repos/{owner}/{repo}/code-scanning/alerts?state=dismissed&${alerts}" \
    "${CODE_DISMISSED:-none.json}"
  route gh "api --paginate --slurp repos/{owner}/{repo}/dependabot/alerts?state=*" "${DEPENDABOT:-dependabot.json}"
  route gh 'api --paginate --slurp repos/{owner}/{repo}/secret-scanning/alerts?state=*' '!1 secret-off.txt'
  route gh 'api graphql*rulesets(first: 100*' rulesets.json
  route gh 'api repos/{owner}/{repo}/branches/main' branch.json
}

@test "kit audit --stages prints its five stages, and --marks the forge's for its reads, the policy's hash for it" {
  run "${kit_root}/bin/kit" audit --stages
  [ "$status" -eq 0 ]
  [ "$output" = $'evidence\nfindings\ncontrols\nwaivers\npolicy' ]
  github-world
  run --separate-stderr "${kit_root}/bin/kit" audit --marks
  [ "$status" -eq 0 ] || { echo "${stderr}"; return 1; }
  # evidence moves with the runs and the releases, findings with the runs and the alerts; the controls and their
  # waivers are read again only when asked; and a repository with no kit.yml nor policy of its own has no policy mark
  [ "$(jq -c '[.evidence, .findings, .controls, .waivers, .policy]' <<<"$output")" \
    = '["W/\"m1\" W/\"m1\"","W/\"m1\" W/\"m1\" W/\"m1\" W/\"m1\"",null,null,null]' ]
  printf 'policy:\n  branch: "^[0-9]+-"\n' > kit.yml
  first=$("${kit_root}/bin/kit" audit --marks | jq -r .policy)
  [ -n "${first}" ] && [ "${first}" != null ]
  printf 'policy:\n  branch: "^x-"\n' > kit.yml
  [ "$("${kit_root}/bin/kit" audit --marks | jq -r .policy)" != "${first}" ]
}

@test "evidence is a row per attestation, keyed by its digest and kind: what it attests, and who signed it" {
  github-world
  run --separate-stderr "${kit_root}/bin/kit" audit evidence --json
  [ "$status" -eq 0 ] || { echo "${stderr}"; return 1; }
  expected='[{"key":"sha256:ab12 https://slsa.dev/provenance/v1","subject":"kit-0.1.0.tar.gz","digest":"sha256:ab12",'
  expected+='"kind":"https://slsa.dev/provenance/v1","signer":"o/r/.github/workflows/release.yml@refs/tags/v0.1.0",'
  expected+='"links":{"subject":"https://github.com/o/r/attestations"}}]'
  [ "$(jq -c '.rows | map(del(.age, .organization, .repository, .source))' <<<"$output")" = "${expected}" ]
  [ "$(jq -c '.errors' <<<"$output")" = '[]' ]
}

@test "findings merges the three scanners' alerts, each row its tool's, and a scanner off here says so, exit 0" {
  github-world
  run --separate-stderr "${kit_root}/bin/kit" audit findings --json
  [ "$status" -eq 0 ] || { echo "${stderr}"; return 1; }
  expected='[{"key":"code-scanning 4","tool":"code-scanning","number":4,"rule":"js/xss","severity":"high",'
  expected+='"state":"open","path":"src/app.js","links":{"rule":"https://github.com/o/r/security/code-scanning/4"}},'
  expected+='{"key":"dependabot 1","tool":"dependabot","number":1,"rule":"GHSA-35jh-r3h4-6jhm","severity":"high",'
  expected+='"state":"dismissed","path":"package-lock.json",'
  expected+='"links":{"rule":"https://github.com/o/r/security/dependabot/1"}}]'
  [ "$(jq -c '.rows | map(del(.age, .organization, .repository, .source))' <<<"$output")" = "${expected}" ]
  [ "$(jq -c '.errors' <<<"$output")" = '[{"feature":"secret-scanning","off":true}]' ]
  run --separate-stderr "${kit_root}/bin/kit" audit findings --text
  [ "$status" -eq 0 ]
  [ "${lines[-1]}" = 'secret-scanning is off here' ]
}

@test "a scanner that refuses the caller is its line under findings' and waivers' errors, the rest drawn, exit 1" {
  DEPENDABOT='!1 refused.txt' DEPENDABOT_HEAD='!1 >scopes.txt' CODE_DISMISSED=code-dismissed.json github-world
  fixture code-dismissed.json <<'JSON'
[[{"number": 2, "state": "dismissed", "created_at": "2026-09-01T09:00:00Z",
   "html_url": "https://github.com/o/r/security/code-scanning/2", "rule": {"id": "js/unused-local-variable"},
   "dismissed_by": {"login": "bob"}, "dismissed_reason": "false positive", "dismissed_comment": null}]]
JSON
  fixture refused.txt <<<'gh: Resource not accessible by integration (HTTP 403)'
  printf 'HTTP/2.0 403 Forbidden\r\nX-Oauth-Scopes: read:org\r\n%s\r\n\r\n{}\n' \
    'X-Accepted-Oauth-Scopes: repo, security_events' > "${FAKE_DIR}/scopes.txt"
  run --separate-stderr "${kit_root}/bin/kit" audit findings --json
  [ "$status" -eq 1 ]
  [ "$(jq -c '[.rows[].key]' <<<"$output")" = '["code-scanning 4","code-scanning 2"]' ]
  [[ "$(jq -c '.errors' <<<"$output")" == '[{"feature":"dependabot","error":"run: cd '*' && gh auth refresh --scopes'\
' security_events (the Dependabot alerts could not be read)"},{"feature":"secret-scanning","off":true}]' ]]
  # the picture of every stage still draws the others: the waivers hold the ruleset's bypass and code scanning's
  # dismissal, Dependabot's line under their errors too, each scanner read once for the findings and the waivers
  : > "${FAKE_DIR}/gh.calls"
  run --separate-stderr "${kit_root}/bin/kit" audit --json
  [ "$status" -eq 1 ]
  [ "$(jq -c '[(.findings.rows | length), (.controls.rows | length)]' <<<"$output")" = '[2,1]' ]
  [ "$(jq -c '[.waivers.rows[].key]' <<<"$output")" = '["7 release","code-scanning 2"]' ]
  [[ "$(jq -c '[.waivers.errors[] | [.feature, (.error // .off)]]' <<<"$output")" == '[["dependabot","run: cd '*' && gh auth'\
' refresh --scopes security_events (the Dependabot alerts could not be read)"],["secret-scanning",true]]' ]]
  [ "$(grep -c 'paginate --slurp repos/{owner}/{repo}/dependabot/alerts' "${FAKE_DIR}/gh.calls")" -eq 1 ]
  [ "$(grep -c 'code-scanning/alerts?state=dismissed' "${FAKE_DIR}/gh.calls")" -eq 1 ]
  # waivers alone read the scanners themselves, and still draw the rest
  run --separate-stderr "${kit_root}/bin/kit" audit waivers --json
  [ "$status" -eq 1 ]
  [ "$(jq -c '[.rows[].key]' <<<"$output")" = '["7 release","code-scanning 2"]' ]
}

@test "controls is a row per control on the default branch, keyed by its id, its kind, its page the control's link" {
  github-world
  run --separate-stderr "${kit_root}/bin/kit" audit controls --json
  [ "$status" -eq 0 ] || { echo "${stderr}"; return 1; }
  expected='[{"key":7,"id":7,"control":"main","target":["~DEFAULT_BRANCH"],"enforcement":"active",'
  expected+='"rules":["deletion","pull_request"],"kind":"ruleset",'
  expected+='"links":{"control":"https://github.com/o/r/rules/7"}}]'
  [ "$(jq -c '.rows | map(del(.organization, .repository, .source))' <<<"$output")" = "${expected}" ]
  # a ruleset or a protection is its kind, so source says the forge, as every forge row's does
  [ "$(jq -c '[.rows[] | [.organization, .repository, .source]]' <<<"$output")" = '[["o","r","github"]]' ]
}

@test "waivers is a row per bypass and dismissal, keyed by what it waives and for whom, each linking its row" {
  github-world
  run --separate-stderr "${kit_root}/bin/kit" audit waivers --json
  [ "$status" -eq 0 ] || { echo "${stderr}"; return 1; }
  expected='[{"key":"7 release","waives":7,"actor":"release","kind":"bypass","reason":"always","by":null,"until":null,'
  expected+='"hidden":null,"links":{"waives":"row:controls/7","reason":"https://github.com/o/r/rules/7"}},'
  expected+='{"key":"dependabot 1","waives":"dependabot 1","actor":null,"kind":"dismissal","reason":"tolerable_risk",'
  expected+='"by":"ada","until":null,"hidden":null,"links":{"waives":"row:findings/dependabot 1",'
  expected+='"reason":"https://github.com/o/r/security/dependabot/1"}}]'
  [ "$(jq -c '.rows | map(del(.organization, .repository, .source))' <<<"$output")" = "${expected}" ]
  # the scanners off here say so on the waivers too
  [ "$(jq -c '.errors' <<<"$output")" = '[{"feature":"secret-scanning","off":true}]' ]
}

@test "policy is a row per rule the policy judges by, where it is in force: its pattern, where it is set, the engine" {
  github-world
  printf 'policy:\n  branch: "^[0-9]+-"\n' > kit.yml
  run --separate-stderr "${kit_root}/bin/kit" audit policy --json
  [ "$status" -eq 0 ] || { echo "${stderr}"; return 1; }
  [ "$(jq -c '.rows[0]' <<<"$output")" = "$(printf '%s' '{"key":"branch github o r main",' \
    '"source":"github","organization":"o","repository":"r","branch":"main","scope":"branch","rule":"branch",' \
    '"pattern":"^[0-9]+-","from":"kit.yml","engine":"patterns","links":{}}')" ]
  [ "$(jq -c '[.rows[] | [.rule, .from]]' <<<"$output")" \
    = '[["branch","kit.yml"],["worktree","default"],["commit_subject","default"],["change_title","default"]]' ]
}

@test "policy reads each checked-out branch, merges a rule its branches agree on, and keeps apart those they do not" {
  github-world
  printf 'policy:\n  branch: "^[0-9]+-"\n' > kit.yml
  git add kit.yml
  git commit --quiet --message one
  git update-ref refs/remotes/origin/main HEAD
  git worktree add --quiet -b 2-fix "${BATS_TEST_TMPDIR}/w"
  git worktree add --quiet --detach "${BATS_TEST_TMPDIR}/d"
  # the branch past the default is judged by the base's kit.yml, which main's own agrees with: one row a rule
  run --separate-stderr "${kit_root}/bin/kit" audit policy --json
  [ "$status" -eq 0 ] || { echo "S${status} E${stderr} O${output}"; return 1; }
  [ "$(jq -c '[.rows[] | [.key, .scope, .branch]]' <<<"$output")" = "$(printf '%s' \
    '[["branch github o r","repository",null],["worktree github o r","repository",null],' \
    '["commit_subject github o r","repository",null],["change_title github o r","repository",null]]')" ]
  # main's own kit.yml now differs from what it holds, which still judges 2-fix: the rule is a row a branch
  printf 'policy:\n  branch: "^x-"\n' > kit.yml
  run --separate-stderr "${kit_root}/bin/kit" audit policy --json
  [ "$status" -eq 0 ] || { echo "${stderr}"; return 1; }
  [ "$(jq -c '[.rows[] | [.rule, .scope, .branch, .pattern]]' <<<"$output")" = "$(printf '%s' \
    '[["branch","branch","2-fix","^[0-9]+-"],["branch","branch","main","^x-"],' \
    '["worktree","repository",null,"^worktrees/[a-z0-9]+-[a-z0-9][a-z0-9-]*$"],' \
    '["commit_subject","repository",null,"^([A-Z]+)?#[0-9]+ [^ ]"],' \
    '["change_title","repository",null,"^[^[:cntrl:]]{1,120}$"]]')" ]
  # nothing is placed in the other checkout
  [ ! -e "${BATS_TEST_TMPDIR}/w/.kit" ]
  # the marks follow each checkout's policy
  first=$("${kit_root}/bin/kit" audit --marks | jq -r .policy)
  [[ "${first}" == "main="*" 2-fix="* ]]
  printf 'policy:\n  branch: "^y-"\n' > "${BATS_TEST_TMPDIR}/w/kit.yml"
  [ "$("${kit_root}/bin/kit" audit --marks | jq -r .policy)" != "${first}" ]
}

@test "kit audit --json draws every stage, and a repository with no forge draws only the policy's rules" {
  mkdir -p "${BATS_TEST_TMPDIR}/r"
  git init --quiet --initial-branch=main "${BATS_TEST_TMPDIR}/r"
  cd "${BATS_TEST_TMPDIR}/r"
  run --separate-stderr "${kit_root}/bin/kit" audit --json
  [ "$status" -eq 0 ] || { echo "${stderr}"; return 1; }
  [ "$(jq -c '.stages' <<<"$output")" = '["evidence","findings","controls","waivers","policy"]' ]
  [ "$(jq -c '[.evidence, .findings, .controls, .waivers]' <<<"$output")" = "$(printf '[%s]' \
    '{"rows":[],"errors":[]},{"rows":[],"errors":[]},{"rows":[],"errors":[]},{"rows":[],"errors":[]}')" ]
  [ "$(jq -c '.policy.rows | length' <<<"$output")" -eq 4 ]
  run --separate-stderr "${kit_root}/bin/kit" audit --marks
  [ "$status" -eq 0 ]
  [ "$output" = '{"evidence":null,"findings":null,"controls":null,"waivers":null,"policy":null}' ]
}

@test "policy over a clone with no remote on a forge is its main checkout's directory, of no organization or source" {
  mkdir -p "${BATS_TEST_TMPDIR}/r"
  git init --quiet --initial-branch=main "${BATS_TEST_TMPDIR}/r"
  cd "${BATS_TEST_TMPDIR}/r"
  git commit --quiet --allow-empty --message one
  # with no remote, the organization and the source it has none of are "", as null is only the place merged rows span
  run --separate-stderr "${kit_root}/bin/kit" audit policy --json
  [ "$status" -eq 0 ] || { echo "${stderr}"; return 1; }
  [ "$(jq -c '[.rows[] | [.key, .scope, .source, .organization, .repository, .branch]]' <<<"$output")" \
    = "$(printf '%s' '[["branch   r main","branch","","","r","main"],["worktree   r main","branch","","","r","main"],' \
    '["commit_subject   r main","branch","","","r","main"],["change_title   r main","branch","","","r","main"]]')" ]
  # a remote on this machine is on no forge: the branches merge, and the place is still none of either
  git init --quiet --bare "${BATS_TEST_TMPDIR}/origin.git"
  git remote add origin "${BATS_TEST_TMPDIR}/origin.git"
  git push --quiet origin main
  git fetch --quiet origin
  git worktree add --quiet -b 2-fix "${BATS_TEST_TMPDIR}/w"
  run --separate-stderr "${kit_root}/bin/kit" audit policy --json
  [ "$status" -eq 0 ] || { echo "${stderr}"; return 1; }
  [ "$(jq -c '[.rows[] | [.key, .scope, .source, .organization, .repository, .branch]]' <<<"$output")" \
    = "$(printf '%s' '[["branch   r","repository","","","r",null],["worktree   r","repository","","","r",null],' \
    '["commit_subject   r","repository","","","r",null],["change_title   r","repository","","","r",null]]')" ]
  # read from the other worktree, the repository is still the main checkout's directory
  cd "${BATS_TEST_TMPDIR}/w"
  run --separate-stderr "${kit_root}/bin/kit" audit policy --json
  [ "$status" -eq 0 ] || { echo "${stderr}"; return 1; }
  [ "$(jq -c '[.rows[] | [.scope, .organization, .repository]] | unique' <<<"$output")" = '[["repository","","r"]]' ]
}

# two-members makes the workspace w of two members under code/o: a, on GitHub, as github-world answers for it, and b,
# on GitLab, which the fake glab, answering nothing, refuses.
two-members() {
  github-world
  fake-tools glab
  git commit --quiet --allow-empty --message one
  mkdir -p "${BATS_TEST_TMPDIR}/code/o"
  mv "${BATS_TEST_TMPDIR}/r" "${BATS_TEST_TMPDIR}/code/o/a"
  git clone --quiet "${BATS_TEST_TMPDIR}/code/o/a" "${BATS_TEST_TMPDIR}/code/o/b"
  git -C "${BATS_TEST_TMPDIR}/code/o/a" remote set-url origin https://github.com/o/a.git
  git -C "${BATS_TEST_TMPDIR}/code/o/b" remote set-url origin https://gitlab.com/o/b.git
  mkdir -p "${XDG_CONFIG_HOME}/kit"
  printf 'roots: [%s]\nworkspaces: {w: {repos: [github.com/o/a, gitlab.com/o/b]}}\n' "${BATS_TEST_TMPDIR}/code" \
    > "${XDG_CONFIG_HOME}/kit/workspaces.yml"
  cd "${BATS_TEST_TMPDIR}"
}

@test "over a group, a member the forge refuses is its scanners' lines under errors, each with its member" {
  two-members
  run --separate-stderr "${kit_root}/bin/kit" --workspace=w audit findings --json
  [ "$status" -eq 1 ]
  # the member that answers draws its rows, each its own
  [ "$(jq --compact-output '[.rows[] | [.member, .key]]' <<<"$output")" \
    = '[["github.com/o/a","code-scanning 4"],["github.com/o/a","dependabot 1"]]' ]
  # its scanner off is said as off; the refused member's three scanners each say their line
  [[ "$(jq --compact-output '[.errors[] | [.member, .feature, (.off // .error)]]' <<<"$output")" == \
    '[["github.com/o/a","secret-scanning",true],'\
'["gitlab.com/o/b","code-scanning","run: cd '*' && glab auth status (the code scanning alerts could not be read)"],'\
'["gitlab.com/o/b","dependabot","run: cd '*' && glab auth status (the Dependabot alerts could not be read)"],'\
'["gitlab.com/o/b","secret-scanning","run: cd '*' && glab auth status (the secret scanning alerts could not be read)"]]' ]]
}

@test "over a group, the policy's rules every member agrees on are one row each, wider than any one repository" {
  two-members
  printf 'policy:\n  branch: "^b-"\n' > code/o/b/kit.yml
  run --separate-stderr "${kit_root}/bin/kit" --workspace=w audit policy --json
  [ "$status" -eq 0 ] || { echo "${stderr}"; return 1; }
  [ "$(jq -c '[.rows[] | [.rule, .scope, .source, .repository, .member]]' <<<"$output")" = "$(printf '%s' \
    '[["branch","branch","github","a","github.com/o/a"],["branch","branch","gitlab","b","gitlab.com/o/b"],' \
    '["worktree","workspace",null,null,null],["commit_subject","workspace",null,null,null],' \
    '["change_title","workspace",null,null,null]]')" ]
}

@test "an app verb's marks over a group join each member's, and name the member they could not be read of" {
  two-members
  run --separate-stderr "${kit_root}/bin/kit" --workspace=w issues --marks
  [ "$status" -eq 1 ]
  [ "$(jq --compact-output .milestones <<<"$output")" = '"W/\"m1\" null"' ]
  [[ "$(jq --compact-output '.errors' <<<"$output")" \
    == '[{"member":"gitlab.com/o/b","error":"run: cd '*' && glab auth status (the milestones mark could not be read)"}]' ]]
  [[ "$stderr" == *"gitlab.com/o/b: run: cd "*" && glab auth status (the milestones mark could not be read)"* ]]
}

# ----------------------------------------------------------------------------------------------------------------------
