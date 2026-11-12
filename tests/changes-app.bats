#!/usr/bin/env bats
# changes-app.bats - kit changes: each open change's reviews and checks, and the changes merged last, each with a mark
# ----------------------------------------------------------------------------------------------------------------------
# Each test runs with HOME, XDG_CONFIG_HOME, and XDG_CACHE_HOME of its own, as helper:isolate sets them. A repository's
# origin is a GitHub URL that nothing fetches: the fake gh answers for it. tests/changes.bats is kit flow changes'.

bats_require_minimum_version 1.5.0

load helper

setup() {
  hide-system-toolchains
}

# github-world makes a repository whose origin is GitHub's, and the fake gh's answers: change 3, opened by ada, which
# asks bob for a review, which carol approved and ada commented on, with one unresolved thread, and the runs a and b
# on its head commit, a's first run failed and superseded by a rerun that passed; and change 2, merged by ada.
github-world() {
  mkdir -p "${BATS_TEST_TMPDIR}/r"
  git init --quiet --initial-branch=main "${BATS_TEST_TMPDIR}/r"
  cd "${BATS_TEST_TMPDIR}/r"
  git remote add origin https://github.com/o/r.git
  fake-tools gh
  fixture user.json <<<'{"login": "ada"}'
  fixture repo.json <<<'{"nameWithOwner": "o/r"}'
  fixture open-prs.json <<'JSON'
{"data": {"repository": {"nameWithOwner": "o/r", "issues": {"nodes": []}, "pullRequests": {"nodes": [{
  "number": 3, "title": "Fix the thing", "isDraft": false, "headRefName": "7-fix-the-thing",
  "createdAt": "2026-10-01T09:00:00Z", "mergeStateStatus": "CLEAN", "reviewDecision": null,
  "author": {"login": "ada"}, "labels": {"nodes": []}, "closingIssuesReferences": {"nodes": []},
  "reviewRequests": {"totalCount": 1, "nodes": [{"requestedReviewer": {"login": "bob"}}]},
  "latestReviews": {"totalCount": 2, "nodes": [
    {"author": {"login": "ada"}, "state": "COMMENTED", "submittedAt": "2026-10-01T11:00:00Z"},
    {"author": {"login": "carol"}, "state": "APPROVED", "submittedAt": "2026-10-02T09:00:00Z"}]},
  "reviewThreads": {"nodes": [{"isResolved": true}, {"isResolved": false}]},
  "commits": {"nodes": [{"commit": {"statusCheckRollup": {"contexts": {"nodes": [
    {"__typename": "CheckRun", "name": "a", "status": "COMPLETED", "conclusion": "FAILURE",
     "startedAt": "2026-09-30T10:00:00Z", "completedAt": "2026-09-30T10:05:00Z"},
    {"__typename": "CheckRun", "name": "a", "status": "COMPLETED", "conclusion": "SUCCESS",
     "startedAt": "2026-10-01T10:00:00Z", "completedAt": "2026-10-01T10:00:30Z"},
    {"__typename": "CheckRun", "name": "b", "status": "COMPLETED", "conclusion": "SUCCESS",
     "startedAt": "2026-10-01T10:00:10Z", "completedAt": "2026-10-01T10:01:00Z"}]}}}}]}}]}}}}
JSON
  fixture merged.json <<'JSON'
{"data": {"repository": {"pullRequests": {"totalCount": 1, "nodes": [
  {"number": 2, "title": "Do the other thing", "url": "https://github.com/o/r/pull/2", "headRefName": "9-other",
   "mergedAt": "2026-10-03T09:00:00Z", "updatedAt": "2026-10-03T09:00:00Z", "author": {"login": "bob"},
   "mergedBy": {"login": "ada"}}]}}}}
JSON
  printf 'HTTP/2.0 200 OK\r\nETag: W/"m1"\r\n\r\n[]\n' > "${FAKE_DIR}/marks-200.txt"
  route gh 'api user*' user.json
  route gh 'repo view*' repo.json
  route gh 'api --include*' marks-200.txt
  route gh 'api graphql*states: MERGED*' merged.json
  route gh 'api graphql*' open-prs.json
}

@test "kit changes --stages prints reviews, checks, then merged, and --marks gives each the issues and runs marks" {
  run "${kit_root}/bin/kit" changes --stages
  [ "$status" -eq 0 ]
  [ "$output" = $'reviews\nchecks\nmerged' ]
  github-world
  run --separate-stderr "${kit_root}/bin/kit" changes --marks
  [ "$status" -eq 0 ]
  [ "$output" = '{"reviews":"W/\"m1\" W/\"m1\"","checks":"W/\"m1\" W/\"m1\"","merged":"W/\"m1\" W/\"m1\""}' ]
}

@test "reviews is a row per change and reviewer, the author's own comment none, with the change's unresolved threads" {
  github-world
  run --separate-stderr "${kit_root}/bin/kit" changes reviews --json
  [ "$status" -eq 0 ]
  expected='[{"key":"3 bob","change":3,"title":"Fix the thing","login":"bob","state":"requested","at":null,'
  expected+='"unresolved":1,"links":{}},'
  expected+='{"key":"3 carol","change":3,"title":"Fix the thing","login":"carol","state":"approved","at":1790931600,'
  expected+='"unresolved":1,"links":{}}]'
  [ "$(jq -c '.rows | map(del(.organization, .repository, .source))' <<<"$output")" = "${expected}" ]
  [ "$(jq -c '.errors' <<<"$output")" = '[]' ]
}

@test "checks is a row per change and check, the latest run of each name, so a rerun supersedes a failed run" {
  github-world
  run --separate-stderr "${kit_root}/bin/kit" changes checks --json
  [ "$status" -eq 0 ]
  [ "$(jq -c '[.rows[] | [.key, .change, .name, .verdict, .at, .took]]' <<<"$output")" \
    = '[["3 a",3,"a","passed",1790848800,30],["3 b",3,"b","passed",1790848810,50]]' ]
}

@test "reviews and checks read the open changes once, the one GraphQL request kit flow changes makes" {
  github-world
  run --separate-stderr "${kit_root}/bin/kit" changes --json
  [ "$status" -eq 0 ]
  [ "$(jq -c '[.reviews.rows, .checks.rows, .merged.rows] | map(length)' <<<"$output")" = '[2,2,1]' ]
  [ "$(grep -c '^api graphql' "${FAKE_DIR}/gh.calls")" -eq 2 ]
  [ "$(grep -c 'pullRequests(states: OPEN' "${FAKE_DIR}/gh.calls")" -eq 1 ]
  [ "$(grep -c 'pullRequests(states: MERGED' "${FAKE_DIR}/gh.calls")" -eq 1 ]
}

@test "reviews and checks given --fresh read the open changes afresh, not the snapshot of the last minute" {
  github-world
  reads() { grep -c 'pullRequests(states: OPEN' "${FAKE_DIR}/gh.calls"; }
  "${kit_root}/bin/kit" changes reviews --json > /dev/null
  "${kit_root}/bin/kit" changes checks --json > /dev/null
  [ "$(reads)" -eq 1 ]
  # a snapshot read in the second the stage began counts as fresh, so each fresh read begins a second later
  sleep 1
  run --separate-stderr "${kit_root}/bin/kit" changes reviews --json --fresh
  [ "$status" -eq 0 ]
  [ "$(jq '.rows | length' <<<"$output")" -eq 2 ]
  [ "$(reads)" -eq 2 ]
  sleep 1
  run --separate-stderr "${kit_root}/bin/kit" changes checks --json --fresh
  [ "$status" -eq 0 ]
  [ "$(jq '.rows | length' <<<"$output")" -eq 2 ]
  [ "$(reads)" -eq 3 ]
  # merged keeps no snapshot, so it takes no --fresh
  run --separate-stderr "${kit_root}/bin/kit" changes merged --json --fresh
  [ "$status" -eq 2 ]
  # over a workspace, each member reads afresh
  mkdir -p "${BATS_TEST_TMPDIR}/code/o"
  git -C "${BATS_TEST_TMPDIR}/r" commit --quiet --allow-empty --message one
  git clone --quiet "${BATS_TEST_TMPDIR}/r" "${BATS_TEST_TMPDIR}/code/o/a"
  git -C "${BATS_TEST_TMPDIR}/code/o/a" remote set-url origin https://github.com/o/a.git
  mkdir -p "${XDG_CONFIG_HOME}/kit"
  printf 'roots: [%s]\nworkspaces: {w: {repos: [github.com/o/a]}}\n' "${BATS_TEST_TMPDIR}/code" \
    > "${XDG_CONFIG_HOME}/kit/workspaces.yml"
  "${kit_root}/bin/kit" --workspace=w changes reviews --json > /dev/null
  "${kit_root}/bin/kit" --workspace=w changes reviews --json > /dev/null
  [ "$(reads)" -eq 4 ]
  sleep 1
  run --separate-stderr "${kit_root}/bin/kit" --workspace=w changes reviews --json --fresh
  [ "$status" -eq 0 ]
  [ "$(jq -c '[.rows[].member] | unique' <<<"$output")" = '["github.com/o/a"]' ]
  [ "$(reads)" -eq 5 ]
}

@test "checks on a provider that lacks check-runs, as one written to contract 3, draws none and says it lacks them" {
  mkdir -p "${BATS_TEST_TMPDIR}/r/bash/forge"
  git init --quiet --initial-branch=main "${BATS_TEST_TMPDIR}/r"
  cd "${BATS_TEST_TMPDIR}/r"
  git remote add origin https://code.example/team/example.git
  printf 'forge: plain\nbash_path:\n  - kit:modules/bash/\n  - bash/\n' > kit.yml
  printf '%s\n' 'forge/plain:contract() { printf "3\n"; }' 'forge/plain:require() { :; }' \
    'forge/plain:open-prs() { printf "%s\n" "{\"number\": 3, \"title\": \"x\"}"; }' > bash/forge/plain.sh
  run --separate-stderr "${kit_root}/bin/kit" changes checks --json
  [ "$status" -eq 0 ] || { echo "${stderr}"; return 1; }
  [ "$output" = '{"rows":[],"errors":[{"feature":"check-runs","off":true,"lacked":true}]}' ]
  run --separate-stderr "${kit_root}/bin/kit" changes checks --text
  [ "$status" -eq 0 ]
  [[ "$output" == *"check-runs is lacked by the provider of the forge here"* ]]
}

@test "merged is the changes merged last, each keyed by its number, its page the title's link" {
  github-world
  run --separate-stderr "${kit_root}/bin/kit" changes merged --json
  [ "$status" -eq 0 ]
  expected='[{"key":2,"number":2,"title":"Do the other thing","author":"bob","branch":"9-other","merged":1791018000,'
  expected+='"by":"ada","named":[],"links":{"title":"https://github.com/o/r/pull/2"}}]'
  [ "$(jq -c '.rows | map(del(.organization, .repository, .source))' <<<"$output")" = "${expected}" ]
  [ "$(jq -c 'has("limited")' <<<"$output")" = false ]
  # more merged than GitHub's read held makes the document limited, at the count it drew
  jq '.data.repository.pullRequests.totalCount = 70' "${FAKE_DIR}/merged.json" > "${FAKE_DIR}/merged.new" \
    && mv "${FAKE_DIR}/merged.new" "${FAKE_DIR}/merged.json"
  run --separate-stderr "${kit_root}/bin/kit" changes merged --json
  [ "$status" -eq 0 ]
  [ "$(jq -c '.limited' <<<"$output")" = 1 ]
}

@test "kit changes with --text prints a stage as text" {
  github-world
  run --separate-stderr "${kit_root}/bin/kit" changes reviews --text
  [ "$status" -eq 0 ]
  [[ "${lines[0]}" == "change  title  "*"login  state  "*"at  "*"unresolved" ]]
  [[ "${lines[1]}" == "3  "*"Fix the thing  "*"bob  "*"requested"* ]]
}

@test "the other Changes tab is kit flow's: kit changes changes --json refuses with 2, and names the command" {
  github-world
  run --separate-stderr "${kit_root}/bin/kit" changes changes --json
  [ "$status" -eq 2 ]
  [ -z "$output" ]
  [[ "$stderr" == *"run: cd "*" && kit flow changes --json"* ]]
}

@test "kit changes --json draws every stage, and a repository with no forge draws none" {
  mkdir -p "${BATS_TEST_TMPDIR}/r"
  git init --quiet --initial-branch=main "${BATS_TEST_TMPDIR}/r"
  cd "${BATS_TEST_TMPDIR}/r"
  run --separate-stderr "${kit_root}/bin/kit" changes --json
  [ "$status" -eq 0 ]
  expected='{"stages":["reviews","checks","merged"],"reviews":{"rows":[],"errors":[]},'
  expected+='"checks":{"rows":[],"errors":[]},"merged":{"rows":[],"errors":[]}}'
  [ "$output" = "${expected}" ]
  run --separate-stderr "${kit_root}/bin/kit" changes --marks
  [ "$status" -eq 0 ]
  [ "$output" = '{"reviews":null,"checks":null,"merged":null}' ]
}

# ----------------------------------------------------------------------------------------------------------------------
