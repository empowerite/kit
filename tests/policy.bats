#!/usr/bin/env bats
# policy.bats - the facts, the patterns and their defaults, and two engines that refuse the same things the same way
# ----------------------------------------------------------------------------------------------------------------------

bats_require_minimum_version 1.5.0

load helper

setup() {
  fake-tools gh
  work=$(mktemp -d)
  cd "${work}"
  git init --quiet --bare --initial-branch=main origin.git
  git clone --quiet origin.git desk 2>/dev/null
  cd desk
  git commit --quiet --allow-empty --message "#1 start: the first" && git push --quiet origin main
  git remote set-head origin main
  mkdir .kit
}

# branch <name> <subject>… makes a branch past main with one empty commit per subject.
branch() {
  git switch --quiet --create "$1"
  shift
  for subject in "$@"
  do
    git commit --quiet --allow-empty --message "${subject}"
  done
}

# change <title> makes the forge, GitHub by the setting since origin is a local path, answer a change on the
# branch with that title.
change() {
  grep --quiet '^forge:' kit.yml 2>/dev/null || printf 'forge: github\n' >> kit.yml
  fixture pr.json <<<"{\"number\": 3, \"title\": \"$1\", \"body\": \"Closes #7\"}"
  fixture repo.json <<<'{"nameWithOwner": "empowerite/example"}'
  fixture pull.json <<<"{\"user\": {\"login\": \"${2:-ada}\"}}"
  fixture open.json <<<'[{"number": 3}]'
  route gh 'pr list --head * --state open --json number*' open.json
  route gh 'pr view 3 --json *' pr.json
  route gh 'repo view*' repo.json
  route gh 'api repos/empowerite/example/pulls/3 *' pull.json
}

@test "the facts are the branch, the commits past the fork, the change, and the rules with the kit's defaults" {
  branch 7-fix-the-thing "#7 cli: fix the thing" "AB#4521 docs: say so"
  change "Fix the thing"
  run "${kit_root}/bin/kit" policy --input
  [ "$status" -eq 0 ]
  [ "$(jq --raw-output .branch <<<"$output")" = "7-fix-the-thing" ]
  subjects=$(jq --raw-output '[.commits[].subject] | join("|")' <<<"$output")
  [ "${subjects}" = "AB#4521 docs: say so|#7 cli: fix the thing" ]
  [ "$(jq --raw-output .change.title <<<"$output")" = "Fix the thing" ]
  [ "$(jq --raw-output .change.author <<<"$output")" = "ada" ]
  [ "$(jq --raw-output '.policy | keys | join(" ")' <<<"$output")" = "branch change_title commit_subject" ]
  run "${kit_root}/bin/kit" policy
  [ "$status" -eq 0 ]
  [[ "$output" == *"the patterns policy allows the branch, 2 commit(s), the change title"* ]]
  # the facts were written to scratch space, swept at exit
  [ -z "$(find "${TMPDIR}" -mindepth 1 -maxdepth 1 -name 'kit.*')" ]
}

@test "a change whose title cannot be read is an error in the facts, and the policy judges nothing" {
  branch 7-fix-the-thing "#7 cli: fix the thing"
  change "Fix the thing"
  # the read of the title fails; the first route that matches answers, so it goes first
  printf '%s\t%s\n' 'pr view 3 --json title*' '!1' | cat - "${FAKE_DIR}/gh.routes" > "${FAKE_DIR}/gh.routes.new"
  mv "${FAKE_DIR}/gh.routes.new" "${FAKE_DIR}/gh.routes"
  run --separate-stderr "${kit_root}/bin/kit" policy --input
  [ "$status" -eq 0 ]
  [ "$(jq --raw-output '.change | keys[]' <<<"$output")" = error ]
  [[ "$(jq --raw-output .change.error <<<"$output")" == *"could not be read"* ]]
  run "${kit_root}/bin/kit" policy
  [ "$status" -eq 1 ]
  refusal='the change could not be read, so its title is unchecked: '
  [[ "$output" == *"${refusal}"*"could not be read"*"; then run: kit policy"* ]]
  [[ "$output" != *"policy allows"* ]]
}

@test "a forge that cannot answer is an error in the facts, and kit policy refuses it" {
  branch 7-fix-the-thing "#7 cli: fix the thing"
  change "Fix the thing"
  : > "${FAKE_DIR}/gh.routes"
  run "${kit_root}/bin/kit" policy
  [ "$status" -eq 1 ]
  [[ "$output" == *"the change could not be read, so its title is unchecked"* ]]
  [[ "$output" != *"policy allows"* ]]
  [ "$("${kit_root}/bin/kit" policy --input | jq --raw-output '.change | keys[]')" = error ]
  # the error is the provider's line, less its ❌ and its color
  error=$(CLICOLOR_FORCE=1 "${kit_root}/bin/kit" policy --input 2>/dev/null | jq --raw-output .change.error)
  [[ "${error}" == "run: gh auth status ("*"could not be read)" ]]
  [[ "${error}" != *$'\x1b'* ]]
  # a document that holds the error is refused as kit policy refuses the one it gathers, before either engine runs
  "${kit_root}/bin/kit" policy --input > "${BATS_TEST_TMPDIR}/facts.json" 2>/dev/null
  run "${kit_root}/bin/kit" policy --facts="${BATS_TEST_TMPDIR}/facts.json"
  [ "$status" -eq 1 ]
  [[ "$output" == *"the change could not be read, so its title is unchecked: ${error}; then run: kit policy"* ]]
  [[ "$output" != *"refusal(s) by the"* ]]
}

@test "with no forge, and with a forge that says there is no change, the change is null" {
  branch 7-fix-the-thing "#7 cli: fix the thing"
  run "${kit_root}/bin/kit" policy
  [ "$status" -eq 0 ]
  [ "$("${kit_root}/bin/kit" policy --input | jq --compact-output .change)" = null ]
  # no forge is asked: origin is a local path, and no setting names one
  [ ! -s "${FAKE_DIR}/gh.calls" ]
  printf 'forge: github\n' > kit.yml
  fixture none.json <<<'[]'
  route gh 'pr list --head * --state open --json number*' none.json
  run "${kit_root}/bin/kit" policy
  [ "$status" -eq 0 ]
  [ "$("${kit_root}/bin/kit" policy --input | jq --compact-output .change)" = null ]
  grep --quiet 'pr list --head 7-fix-the-thing --state open' "${FAKE_DIR}/gh.calls"
}

@test "the defaults refuse a branch, a commit subject, and a title off the loop's shape, each named" {
  branch topic "#7 cli: fine" "wip"
  change ""
  run "${kit_root}/bin/kit" policy
  [ "$status" -eq 1 ]
  [[ "$output" == *"branch 'topic' does not match policy.branch"* ]]
  [[ "$output" == *"'wip' does not match policy.commit_subject"* ]]
  [[ "$output" == *"change title '' does not match policy.change_title"* ]]
  [[ "$output" == *"3 refusal(s) by the patterns policy"* ]]
}

@test "a repository's patterns replace the defaults, an empty one turns a rule off, and an unknown subject is refused" {
  branch feature/login "wip"
  printf "policy:\n  branch: '^feature/[a-z-]+\$'\n  commit_subject: ''\n" > kit.yml
  run "${kit_root}/bin/kit" policy
  [ "$status" -eq 0 ]
  printf "policy:\n  branche: '^x\$'\n" > kit.yml
  run "${kit_root}/bin/kit" policy
  [ "$status" -eq 1 ]
  [[ "$output" == *"policy has a subject the kit does not know: branche"* ]]
}

@test "a change by a login in linked_exempt passes the policy by name, since its branch and subjects are the bot's" {
  branch dependabot/github-actions/actions-1a2b3c "actions: bump actions/checkout from 4 to 5"
  change "actions: bump actions/checkout from 4 to 5" 'dependabot[bot]'
  printf 'linked_exempt:\n  - dependabot[bot]\n' >> kit.yml
  run "${kit_root}/bin/kit" policy
  [ "$status" -eq 0 ]
  [[ "$output" == *"pull request #3 is dependabot[bot]'s, exempt from the policy by kit.yml"* ]]
  printf 'forge: github\n' > kit.yml
  run "${kit_root}/bin/kit" policy
  [ "$status" -eq 1 ]
  [[ "$output" == *"branch 'dependabot/github-actions/actions-1a2b3c' does not match policy.branch"* ]]
}

@test "a branch is judged by the policy on its base: a pattern it loosens, and a login it exempts, change nothing" {
  printf 'policy:\n  commit_subject: "^#[0-9]+ "\n' > kit.yml
  git add kit.yml
  git commit --quiet --message "#2 kit: the policy" && git push --quiet origin main
  branch 7-fix-the-thing "no reference"
  printf 'policy:\n  commit_subject: ""\nlinked_exempt:\n  - ada\n' > kit.yml
  change "Fix the thing" ada
  run "${kit_root}/bin/kit" policy
  [ "$status" -eq 1 ]
  [[ "$output" == *"does not match policy.commit_subject"* ]]
  [[ "$output" != *"exempt"* ]]
  # the forge and the facts stay the checkout's: the base names no forge, and the change was read
  grep --quiet 'pr view 3 --json' "${FAKE_DIR}/gh.calls"
  # the facts kit policy --input prints are the ones it judges
  [ "$("${kit_root}/bin/kit" policy --input | jq --raw-output .policy.commit_subject)" = '^#[0-9]+ ' ]
}

@test "on a runner, the pull request's detached checkout is judged by the policy on its base" {
  printf 'policy:\n  commit_subject: "^#[0-9]+ "\n' > kit.yml
  git add kit.yml
  git commit --quiet --message "#2 kit: the policy" && git push --quiet origin main
  branch 7-fix-the-thing "no reference"
  printf 'policy:\n  commit_subject: ""\nlinked_exempt:\n  - ada\n' > kit.yml
  change "Fix the thing" ada
  git switch --quiet --detach
  printf '{"pull_request": {"number": 3}}\n' > "${BATS_TEST_TMPDIR}/event.json"
  GITHUB_HEAD_REF=7-fix-the-thing GITHUB_EVENT_PATH="${BATS_TEST_TMPDIR}/event.json" run "${kit_root}/bin/kit" policy
  [ "$status" -eq 1 ]
  [[ "$output" == *"'no reference' does not match policy.commit_subject"* ]]
  [[ "$output" != *"exempt"* ]]
}

@test "a base with no kit.yml, as before the kit is installed there, leaves the branch's own policy to judge it" {
  branch 7-install-the-kit "no reference"
  printf 'policy:\n  commit_subject: "^no "\n' > kit.yml
  git add kit.yml
  git commit --quiet --message "no kit until this one"
  [ -z "$(git ls-tree origin/main -- kit.yml)" ]
  run "${kit_root}/bin/kit" policy
  [ "$status" -eq 0 ]
  [[ "$output" == *"the patterns policy allows the branch, 2 commit(s)"* ]]
  printf 'policy:\n  commit_subject: "^#"\n' > kit.yml
  run "${kit_root}/bin/kit" policy
  [ "$status" -eq 1 ]
  [[ "$output" == *"'no reference' does not match policy.commit_subject"* ]]
}

@test "Rego a branch removes still judges it" {
  mkdir -p .kit/packages/mine/files/repo/.kit/policy
  printf 'description: "rules of our own"\n' > .kit/packages/mine/package.yml
  printf 'package kit.policy\n\nimport rego.v1\n\ndeny contains "x" if false\n' \
    > .kit/packages/mine/files/repo/.kit/policy/x.rego
  printf 'packages:\n  - mine\n' > kit.yml
  git add kit.yml .kit/packages
  git commit --quiet --message "#2 kit: a rule of our own" && git push --quiet origin main
  branch 7-fix-the-thing
  git rm --quiet -r .kit/packages/mine
  printf 'packages: []\n' > kit.yml
  git commit --quiet --all --message "#7 kit: no rule of our own"
  dir=${BATS_TEST_TMPDIR}/judged
  kit_bash_modules=policy run kit-bash "policy:from-base '${dir}'"
  [ "$status" -eq 0 ]
  [ -f "${dir}/rego/x.rego" ]
  cmp "${dir}/rego/x.rego" <(git show origin/main:.kit/packages/mine/files/repo/.kit/policy/x.rego)
}

@test "a rule in Rego the base holds judges a branch that deletes it" {
  command -v opa >/dev/null || skip "opa, the opa policy engine, is not on PATH"
  mkdir -p .kit/policy
  cat > .kit/policy/ours.rego <<'REGO'
package kit.policy

import rego.v1

deny contains "a commit with no reference" if {
	some commit in input.commits
	commit.subject == "no reference"
}
REGO
  printf 'policy:\n  commit_subject: ""\n' > kit.yml
  git add kit.yml .kit/policy/ours.rego
  git commit --quiet --message "#2 kit: a rule of our own" && git push --quiet origin main
  branch 7-fix-the-thing "no reference"
  git rm --quiet .kit/policy/ours.rego
  git commit --quiet --message "#7 kit: no rule of our own"
  run "${kit_root}/bin/kit" policy
  [ "$status" -eq 1 ]
  [[ "$output" == *"❌ a commit with no reference"* ]]
  [[ "$output" == *"1 refusal(s) by the opa policy"* ]]
}

@test "Rego a branch adds does not choose the engine that judges it" {
  printf 'policy:\n  commit_subject: "^#[0-9]+ "\n' > kit.yml
  git add kit.yml
  git commit --quiet --message "#2 kit: the policy" && git push --quiet origin main
  branch 7-fix-the-thing "no reference"
  mkdir -p .kit/policy
  printf 'package kit.policy\n\nimport rego.v1\n\ndeny contains "never" if false\n' > .kit/policy/lax.rego
  run "${kit_root}/bin/kit" policy
  [ "$status" -eq 1 ]
  [[ "$output" == *"'no reference' does not match policy.commit_subject"* ]]
  [[ "$output" == *"1 refusal(s) by the patterns policy"* ]]
}

@test "a change whose head branch is named main, as a fork's can be, is judged by the policy on its base" {
  printf 'policy:\n  commit_subject: "^#[0-9]+ "\n' > kit.yml
  git add kit.yml
  git commit --quiet --message "#2 kit: the policy" && git push --quiet origin main
  git switch --quiet --create fork-main
  printf 'policy:\n  commit_subject: ""\nlinked_exempt:\n  - ada\n' > kit.yml
  git commit --quiet --all --message "no reference"
  change "Fix the thing" ada
  fixture closing.json <<<'{"data": {"repository": {"pullRequest": {"closingIssuesReferences": {"nodes": []}}}}}'
  route gh 'api graphql*' closing.json
  git switch --quiet --detach
  printf '{"pull_request": {"number": 3}}\n' > "${BATS_TEST_TMPDIR}/event.json"
  export GITHUB_HEAD_REF=main GITHUB_EVENT_PATH="${BATS_TEST_TMPDIR}/event.json"
  run "${kit_root}/bin/kit" policy
  [ "$status" -eq 1 ]
  [[ "$output" == *"'no reference' does not match policy.commit_subject"* ]]
  [[ "$output" != *"exempt"* ]]
  run "${kit_root}/bin/kit" linked 3
  [ "$status" -eq 1 ]
  [[ "$output" == *"pull request #3 is not linked to any issue"* ]]
  [[ "$output" != *"exempt"* ]]
}

@test "the files the base's rules read are the base's facts.read: a branch that drops it blinds no rule" {
  command -v opa >/dev/null || skip "opa, the opa policy engine, is not on PATH"
  mkdir -p .kit/policy
  cat > .kit/policy/ours.rego <<'REGO'
package kit.policy

import rego.v1

deny contains "README.md is a template" if contains(input.tree["README.md"].text, "TEMPLATE")
REGO
  printf 'TEMPLATE\n' > README.md
  printf 'policy:\n  commit_subject: ""\nfacts:\n  read:\n    - README.md\n' > kit.yml
  git add kit.yml README.md .kit/policy/ours.rego
  git commit --quiet --message "#2 kit: a rule over the text" && git push --quiet origin main
  branch 7-fix-the-thing
  printf 'policy:\n  commit_subject: ""\n' > kit.yml
  git commit --quiet --all --message "#7 kit: read no file"
  run "${kit_root}/bin/kit" policy
  [ "$status" -eq 1 ]
  [[ "$output" == *"❌ README.md is a template"* ]]
  [ "$("${kit_root}/bin/kit" policy --input | jq --raw-output '.tree["README.md"].text')" = TEMPLATE ]
}

@test "Rego from a package of a source the base names still judges a branch" {
  command -v opa >/dev/null || skip "opa, the opa policy engine, is not on PATH"
  local src=${BATS_TEST_TMPDIR}/src digest
  mkdir -p "${src}/top/packages/rules/files/repo/.kit/policy"
  printf 'description: "rules from a source"\n' > "${src}/top/packages/rules/package.yml"
  cat > "${src}/top/packages/rules/files/repo/.kit/policy/rules.rego" <<'REGO'
package kit.policy

import rego.v1

deny contains "a commit with no reference" if {
	some commit in input.commits
	commit.subject == "no reference"
}
REGO
  tar --create --gzip --file "${src}/source.tar.gz" --directory "${src}" top
  digest=$(shasum -a 256 "${src}/source.tar.gz" | cut -d ' ' -f 1)
  printf 'policy:\n  commit_subject: ""\nsources:\n  - name: team\n    url: file://%s\n    digest: sha256:%s\n' \
    "${src}/source.tar.gz" "${digest}" > kit.yml
  printf 'packages:\n  - rules\n' >> kit.yml
  git add kit.yml
  git commit --quiet --message "#2 kit: rules from a source" && git push --quiet origin main
  branch 7-fix-the-thing "no reference"
  run "${kit_root}/bin/kit" policy
  [ "$status" -eq 1 ]
  [[ "$output" == *"❌ a commit with no reference"* ]]
  [[ "$output" == *"by the opa policy"* ]]
}

@test "a package the base lists that is in no source, the release, or the base's own packages fails, named" {
  printf 'policy:\n  commit_subject: ""\npackages:\n  - nowhere\n' > kit.yml
  git add kit.yml
  git commit --quiet --message "#2 kit: a package that is nowhere" && git push --quiet origin main
  branch 7-fix-the-thing "#7 cli: fix the thing"
  run "${kit_root}/bin/kit" policy
  [ "$status" -eq 1 ]
  [[ "$output" == *"the package nowhere, which its kit.yml lists, is in none of its sources"* ]]
  [[ "$output" != *"policy allows"* ]]
}

@test "a file on the disk under a name a package of the base delivers is judged as the base delivers it" {
  mkdir -p .kit/packages/mine/files/repo/.kit/policy
  printf 'description: "rules of our own"\n' > .kit/packages/mine/package.yml
  printf 'package kit.policy\n\nimport rego.v1\n\ndeny contains "x" if true\n' \
    > .kit/packages/mine/files/repo/.kit/policy/x.rego
  printf 'packages:\n  - mine\n' > kit.yml
  git add kit.yml .kit/packages
  git commit --quiet --message "#2 kit: a rule of our own" && git push --quiet origin main
  branch 7-fix-the-thing
  mkdir -p .kit/policy
  printf 'package kit.policy\n\nimport rego.v1\n\ndeny contains "x" if false\n' > .kit/policy/x.rego
  dir=${BATS_TEST_TMPDIR}/judged
  kit_bash_modules=policy run kit-bash "policy:from-base '${dir}'"
  [ "$status" -eq 0 ]
  cmp "${dir}/rego/x.rego" .kit/packages/mine/files/repo/.kit/policy/x.rego
}

@test "on the default branch, or detached, only the tree is under check, and kit policy says so" {
  run "${kit_root}/bin/kit" policy
  [ "$status" -eq 0 ]
  [[ "$output" == *"the patterns policy allows the tree at HEAD; on main, no branch or commit is judged"* ]]
  git commit --quiet --allow-empty --message "not checked"
  git switch --quiet --detach
  run "${kit_root}/bin/kit" policy
  [ "$status" -eq 0 ]
}

@test "the opa engine, given the policy/opa package's Rego, refuses exactly what the patterns engine does" {
  command -v opa >/dev/null || skip "opa, the opa policy engine, is not on PATH"
  "${kit_root}/bin/kit" install policy/opa >/dev/null 2>&1
  [ -f .kit/policy/kit.rego ]
  branch topic "#7 cli: fine" "wip" "Also wrong"
  change "A title that is fine"
  run "${kit_root}/bin/kit" policy
  [ "$status" -eq 1 ]
  [[ "$output" == *"by the opa policy"* ]]
  by_opa=$(grep '^❌ ' <<<"$output" | grep --invert-match 'refusal(s)' | LC_ALL=C sort)
  printf 'forge: github\npolicy_engine: patterns\n' > kit.yml
  run "${kit_root}/bin/kit" policy
  [ "$status" -eq 1 ]
  by_patterns=$(grep '^❌ ' <<<"$output" | grep --invert-match 'refusal(s)' | LC_ALL=C sort)
  [ "$(wc -l <<<"${by_opa}")" -eq 3 ]
  [ "${by_opa}" = "${by_patterns}" ]
}

@test "a rule of the repository's own in Rego adds to the kit's, and a policy engine the kit does not know is refused" {
  command -v opa >/dev/null || skip "opa, the opa policy engine, is not on PATH"
  "${kit_root}/bin/kit" install policy/opa >/dev/null 2>&1
  cat > .kit/policy/ours.rego <<'REGO'
package kit.policy

import rego.v1

deny contains msg if {
	count(input.commits) > 2
	msg := sprintf("%d commits; split the change", [count(input.commits)])
}
REGO
  branch 7-many "#7 a: one" "#7 a: two" "#7 a: three"
  run "${kit_root}/bin/kit" policy
  [ "$status" -eq 1 ]
  [[ "$output" == *"3 commits; split the change"* ]]
  printf 'policy_engine: cue\n' > kit.yml
  run "${kit_root}/bin/kit" policy
  [ "$status" -eq 1 ]
  [[ "$output" == *"'cue' is not a policy engine the kit knows"* ]]
}

@test "a warning is printed and passes; a refusal beside it still fails" {
  command -v opa >/dev/null || skip "opa, the opa policy engine, is not on PATH"
  "${kit_root}/bin/kit" install policy/opa >/dev/null 2>&1
  cat > .kit/policy/ours.rego <<'REGO'
package kit.policy

import rego.v1

warn contains "consider fewer commits" if count(input.commits) > 1
REGO
  branch 7-many "#7 a: one" "#7 a: two"
  run "${kit_root}/bin/kit" policy
  [ "$status" -eq 0 ]
  [[ "$output" == *"⚠️ consider fewer commits"* ]]
  [[ "$output" == *"allows"*"1 warning"* ]]
  cat >> .kit/policy/ours.rego <<'REGO'

deny contains "never two commits" if count(input.commits) > 1
REGO
  run "${kit_root}/bin/kit" policy
  [ "$status" -eq 1 ]
  [[ "$output" == *"⚠️ consider fewer commits"* ]]
  [[ "$output" == *"❌ never two commits"* ]]
}

# ----------------------------------------------------------------------------------------------------------------------

@test "kit policy --facts judges a saved document, and gathers nothing" {
  branch 7-fix-the-thing "#7 cli: fix the thing"
  change "Fix the thing"
  "${kit_root}/bin/kit" policy --input | jq '.branch = "BAD"' > facts.json
  : > "${FAKE_DIR}/gh.calls"
  run "${kit_root}/bin/kit" policy --facts=facts.json
  [ "$status" -eq 1 ]
  [[ "$output" == *"branch 'BAD' does not match policy.branch"* ]]
  [[ "$output" != *"7-fix-the-thing"* ]]
  [ ! -s "${FAKE_DIR}/gh.calls" ]
}

@test "kit policy --facts with a file that is not JSON is refused" {
  printf 'not json\n' > facts.json
  run "${kit_root}/bin/kit" policy --facts=facts.json
  [ "$status" -eq 2 ]
  [[ "$output" == *"facts.json"* ]]
  run "${kit_root}/bin/kit" policy --facts=missing.json
  [ "$status" -eq 2 ]
  [[ "$output" == *"missing.json"* ]]
}

@test "kit policy --facts refuses an empty file, and an object with no policy, as no facts document" {
  : > empty.json
  run "${kit_root}/bin/kit" policy --facts=empty.json
  [ "$status" -eq 2 ]
  [[ "$output" == *"empty.json"*"is not a facts document"* ]]
  printf '{}\n' > bare.json
  run "${kit_root}/bin/kit" policy --facts=bare.json
  [ "$status" -eq 2 ]
  [[ "$output" == *"bare.json"*"is not a facts document"* ]]
}

@test "kit policy --facts reads a relative file from the directory it was typed in" {
  branch 7-fix-the-thing "#7 cli: fix the thing"
  mkdir sub
  "${kit_root}/bin/kit" policy --input | jq '.branch = "BAD"' > sub/fx.json
  "${kit_root}/bin/kit" policy --input > fx.json
  cd sub
  run "${kit_root}/bin/kit" policy --facts=fx.json
  [ "$status" -eq 1 ]
  [[ "$output" == *"branch 'BAD' does not match policy.branch"* ]]
}

@test "Rego that defines no deny in package kit.policy refuses, since unjudged is never allowed" {
  command -v opa >/dev/null || skip "opa, the opa policy engine, is not on PATH"
  mkdir -p .kit/policy
  printf 'package kit.polcy\n\nimport rego.v1\n\ndeny contains "a typo hides me" if true\n' > .kit/policy/typo.rego
  branch 7-fix-the-thing "#7 cli: fix the thing"
  run "${kit_root}/bin/kit" policy
  [ "$status" -eq 1 ]
  [[ "$output" == *"defines no data.kit.policy.deny"* ]]
}

@test "at ci, a branch whose fork from the default branch is not here fails, since no commit could be checked" {
  branch 7-fix-the-thing "#7 cli: fix the thing"
  # origin/main is here, as in a shallow clone, and shares no commit with the branch
  git update-ref refs/remotes/origin/main "$(git commit-tree "$(git mktree </dev/null)" -m unrelated)"
  run "${kit_root}/bin/kit" policy
  [ "$status" -eq 1 ]
  [[ "$output" == *"no commit was checked"*"fetch"* ]]
}

@test "a base that is not here fails" {
  branch 7-fix-the-thing "#7 cli: fix the thing"
  git update-ref -d refs/remotes/origin/main
  run "${kit_root}/bin/kit" policy
  [ "$status" -eq 1 ]
  [[ "$output" == *"its policy cannot be read"* ]]
  [[ "$output" == *"the base, origin/main, is not here"* ]]
  [[ "$output" != *"policy allows"* ]]
}

