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

@test "a primary on a host nothing names is a change that cannot be read, never one that passes unchecked" {
  unset GITHUB_SERVER_URL CI_SERVER_HOST SYSTEM_COLLECTIONURI
  branch 7-fix-the-thing "#7 cli: fix the thing"
  git remote set-url origin https://ghe.example.com/o/r.git
  run "${kit_root}/bin/kit" policy
  [ "$status" -eq 1 ]
  [[ "$output" == *"the change could not be read, so its title is unchecked: set forge in kit.yml; origin's host"* ]]
  [[ "$output" != *"policy allows"* ]]
  [ "$("${kit_root}/bin/kit" policy --input 2>/dev/null | jq --raw-output '.change | keys[]')" = error ]
  [ ! -s "${FAKE_DIR}/gh.calls" ]
  # a local path stays a primary with no forge
  git remote set-url origin "${work}/origin.git"
  [ "$("${kit_root}/bin/kit" policy --input | jq --compact-output .change)" = null ]
}

@test "a primary on a host with no forge the kit knows says so with forge: none, and its change is null" {
  unset GITHUB_SERVER_URL CI_SERVER_HOST SYSTEM_COLLECTIONURI
  branch 7-fix-the-thing "#7 cli: fix the thing"
  git remote set-url origin git@bitbucket.org:o/r.git
  run "${kit_root}/bin/kit" policy
  [ "$status" -eq 1 ]
  [[ "$output" == *"or none)"* ]]
  printf 'forge: none\n' > kit.yml
  git add kit.yml
  git commit --quiet --message "#7 cli: no forge"
  run "${kit_root}/bin/kit" policy
  [ "$status" -eq 0 ]
  [ "$("${kit_root}/bin/kit" policy --input | jq --compact-output .change)" = null ]
  [ ! -s "${FAKE_DIR}/gh.calls" ]
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
  printf 'forge: github\npolicy:\n  commit_subject: "^#[0-9]+ "\n' > kit.yml
  git add kit.yml
  git commit --quiet --message "#2 kit: the policy" && git push --quiet origin main
  branch 7-fix-the-thing "no reference"
  printf 'policy:\n  commit_subject: ""\nlinked_exempt:\n  - ada\n' > kit.yml
  change "Fix the thing" ada
  run "${kit_root}/bin/kit" policy
  [ "$status" -eq 1 ]
  [[ "$output" == *"does not match policy.commit_subject"* ]]
  [[ "$output" != *"exempt"* ]]
  # the forge is the base's, and the change was read through it
  grep --quiet 'pr view 3 --json' "${FAKE_DIR}/gh.calls"
  # the facts kit policy --input prints are the ones it judges
  [ "$("${kit_root}/bin/kit" policy --input | jq --raw-output .policy.commit_subject)" = '^#[0-9]+ ' ]
}

@test "on a runner, the pull request's detached checkout is judged by the policy on its base" {
  printf 'forge: github\npolicy:\n  commit_subject: "^#[0-9]+ "\n' > kit.yml
  git add kit.yml
  git commit --quiet --message "#2 kit: the policy" && git push --quiet origin main
  branch 7-fix-the-thing "no reference"
  printf 'policy:\n  commit_subject: ""\nlinked_exempt:\n  - ada\n' > kit.yml
  change "Fix the thing" ada
  git switch --quiet --detach
  printf '{"pull_request": {"number": 3, "base": {"sha": "%s"}}}\n' "$(git rev-parse origin/main)" \
    > "${BATS_TEST_TMPDIR}/event.json"
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
  printf 'forge: github\npolicy:\n  commit_subject: "^#[0-9]+ "\n' > kit.yml
  git add kit.yml
  git commit --quiet --message "#2 kit: the policy" && git push --quiet origin main
  git switch --quiet --create fork-main
  printf 'policy:\n  commit_subject: ""\nlinked_exempt:\n  - ada\n' > kit.yml
  git commit --quiet --all --message "no reference"
  change "Fix the thing" ada
  fixture closing.json <<<'{"data": {"repository": {"pullRequest": {"closingIssuesReferences": {"nodes": []}}}}}'
  route gh 'api graphql*' closing.json
  git switch --quiet --detach
  printf '{"pull_request": {"number": 3, "base": {"sha": "%s"}}}\n' "$(git rev-parse origin/main)" \
    > "${BATS_TEST_TMPDIR}/event.json"
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

@test "a link a branch adds under .kit/policy/, to Rego the base holds elsewhere, does not judge it" {
  mkdir -p docs
  printf 'package kit.policy\n\nimport rego.v1\n\ndeny contains "never" if false\n' > docs/lax.rego
  printf 'policy:\n  commit_subject: "^#[0-9]+ "\n' > kit.yml
  git add kit.yml docs/lax.rego
  git commit --quiet --message "#2 kit: the policy" && git push --quiet origin main
  branch 7-fix-the-thing
  mkdir -p .kit/policy
  ln -s ../../docs/lax.rego .kit/policy/lax.rego
  git add --force .kit/policy/lax.rego
  git commit --quiet --message "no reference"
  run "${kit_root}/bin/kit" policy
  [ "$status" -eq 1 ]
  [[ "$output" == *"'no reference' does not match policy.commit_subject"* ]]
  [[ "$output" == *"1 refusal(s) by the patterns policy"* ]]
}

@test "a link the base holds under .kit/policy/ judges by its target as the base holds it" {
  mkdir -p docs .kit/policy
  printf 'package kit.policy\n\nimport rego.v1\n\ndeny contains "x" if true\n' > docs/ours.rego
  ln -s ../../docs/ours.rego .kit/policy/ours.rego
  printf 'policy:\n  commit_subject: ""\n' > kit.yml
  git add kit.yml docs/ours.rego .kit/policy/ours.rego
  git commit --quiet --message "#2 kit: a rule of our own" && git push --quiet origin main
  branch 7-fix-the-thing
  printf 'package kit.policy\n\nimport rego.v1\n\ndeny contains "x" if false\n' > docs/ours.rego
  git rm --quiet .kit/policy/ours.rego
  git commit --quiet --all --message "#7 kit: a lax rule, and no link to it"
  dir=${BATS_TEST_TMPDIR}/judged
  kit_bash_modules=policy run kit-bash "policy:from-base '${dir}'"
  [ "$status" -eq 0 ]
  cmp "${dir}/rego/ours.rego" <(git show origin/main:docs/ours.rego)
}

@test "a branch of the kit itself that edits the Rego its release delivers is judged by the base's" {
  command -v opa >/dev/null || skip "opa, the opa policy engine, is not on PATH"
  # the release is inside the repository, as in a checkout of the kit, and delivers a package the base lists
  (cd "${kit_root}/.." && git ls-files -z kit | tar --null --create --files-from - --file -) | tar --extract --file -
  mkdir -p kit/packages/rules/files/repo/.kit/policy
  printf 'description: "rules of the release"\n' > kit/packages/rules/package.yml
  cat > kit/packages/rules/files/repo/.kit/policy/rules.rego <<'REGO'
package kit.policy

import rego.v1

deny contains "a commit with no reference" if {
	some commit in input.commits
	commit.subject == "no reference"
}
REGO
  printf 'policy:\n  commit_subject: ""\npackages:\n  - rules\n' > kit.yml
  git add kit.yml kit
  git commit --quiet --message "#2 kit: the release's rules" && git push --quiet origin main
  branch 7-fix-the-thing
  printf 'package kit.policy\n\nimport rego.v1\n\ndeny contains "never" if false\n' \
    > kit/packages/rules/files/repo/.kit/policy/rules.rego
  git commit --quiet --all --message "no reference"
  run kit/bin/kit policy
  [ "$status" -eq 1 ]
  [[ "$output" == *"❌ a commit with no reference"* ]]
}

@test "a branch that drops the forge its base names still has its change judged" {
  printf 'forge: github\npolicy:\n  change_title: "^Fix "\n' > kit.yml
  change "bad title" ada
  git add kit.yml
  git commit --quiet --message "#2 kit: the policy" && git push --quiet origin main
  branch 7-fix-the-thing "#7 cli: fix the thing"
  printf 'policy:\n  change_title: "^Fix "\n' > kit.yml
  git commit --quiet --all --message "#7 cli: drop the forge"
  run "${kit_root}/bin/kit" policy
  [ "$status" -eq 1 ]
  [[ "$output" == *"change title 'bad title' does not match policy.change_title"* ]]
  grep --quiet 'pr view 3 --json' "${FAKE_DIR}/gh.calls"
}

@test "a branch that points the primary at another forge in forges still has its change read through the base's" {
  change "bad title" ada
  printf 'forges:\n  origin: github\npolicy:\n  change_title: "^Fix "\n' > kit.yml
  git add kit.yml
  git commit --quiet --message "#2 kit: the policy" && git push --quiet origin main
  branch 7-fix-the-thing "#7 cli: fix the thing"
  printf 'forges:\n  origin: gitlab\npolicy:\n  change_title: "^Fix "\n' > kit.yml
  git commit --quiet --all --message "#7 cli: another forge"
  run "${kit_root}/bin/kit" policy
  [ "$status" -eq 1 ]
  [[ "$output" == *"change title 'bad title' does not match policy.change_title"* ]]
  grep --quiet 'pr view 3 --json' "${FAKE_DIR}/gh.calls"
}

@test "on a runner, the change's base commit in the event judges, so a gate that moves origin/main loosens nothing" {
  printf 'policy:\n  commit_subject: "^#[0-9]+ "\n' > kit.yml
  git add kit.yml
  git commit --quiet --message "#2 kit: the policy" && git push --quiet origin main
  base=$(git rev-parse origin/main)
  branch 7-fix-the-thing
  printf 'gate: git update-ref refs/remotes/origin/main HEAD\npolicy:\n  commit_subject: ""\n' > kit.yml
  git commit --quiet --all --message "no reference"
  printf '{"pull_request": {"number": 3, "base": {"sha": "%s"}}}\n' "${base}" > "${BATS_TEST_TMPDIR}/event.json"
  export GITHUB_HEAD_REF=7-fix-the-thing GITHUB_EVENT_PATH="${BATS_TEST_TMPDIR}/event.json"
  run "${kit_root}/bin/kit" gate
  [ "$status" -eq 0 ]
  [ "$(git rev-parse origin/main)" = "$(git rev-parse HEAD)" ]
  run "${kit_root}/bin/kit" policy
  [ "$status" -eq 1 ]
  [[ "$output" == *"'no reference' does not match policy.commit_subject"* ]]
  [ "$("${kit_root}/bin/kit" policy --input | jq --raw-output '.commits | length')" -eq 1 ]
  # a merged-results pipeline on GitLab names the target's tip the same way, and the moved ref counts for nothing
  run env -u GITHUB_HEAD_REF -u GITHUB_EVENT_PATH CI_MERGE_REQUEST_SOURCE_BRANCH_NAME=7-fix-the-thing \
    CI_MERGE_REQUEST_TARGET_BRANCH_NAME=main CI_MERGE_REQUEST_TARGET_BRANCH_SHA="${base}" "${kit_root}/bin/kit" policy
  [ "$status" -eq 1 ]
  [[ "$output" == *"'no reference' does not match policy.commit_subject"* ]]
  # the head ref names a change; without its event the base cannot be read, and nothing is judged
  run env -u GITHUB_EVENT_PATH "${kit_root}/bin/kit" policy
  [ "$status" -eq 1 ]
  [[ "$output" == *"so its policy cannot be read"* ]]
  [[ "$output" != *"policy allows"* ]]
  # a base commit the event names that is not here is named
  printf '{"pull_request": {"number": 3, "base": {"sha": "%040d"}}}\n' 0 > "${BATS_TEST_TMPDIR}/event.json"
  run "${kit_root}/bin/kit" policy
  [ "$status" -eq 1 ]
  [[ "$output" == *"the base, $(printf '%040d' 0), is not here, so its policy cannot be read"* ]]
}

@test "a kind the base defines, which a rule of the base keys on, judges a branch that redefines it" {
  command -v opa >/dev/null || skip "opa, the opa policy engine, is not on PATH"
  mkdir -p .kit/policy .kit/kinds
  printf 'name: secret\npaths:\n  - "**/*.pem"\n' > .kit/kinds/secret.yml
  cat > .kit/policy/ours.rego <<'REGO'
package kit.policy

import rego.v1

deny contains sprintf("%s is a secret", [path]) if {
	some path, f in input.tree
	"secret" in f.kinds
}
REGO
  printf 'policy:\n  commit_subject: ""\n' > kit.yml
  git add --force kit.yml .kit/kinds/secret.yml .kit/policy/ours.rego
  git commit --quiet --message "#2 kit: no secret in the tree" && git push --quiet origin main
  branch 7-fix-the-thing
  printf 'key\n' > key.pem
  printf 'name: secret\npaths:\n  - "**/*.never"\n' > .kit/kinds/secret.yml
  git add key.pem .kit/kinds/secret.yml
  git commit --quiet --message "#7 kit: a key, and a kind that matches nothing"
  run "${kit_root}/bin/kit" policy
  [ "$status" -eq 1 ]
  [[ "$output" == *"❌ key.pem is a secret"* ]]
  [ "$("${kit_root}/bin/kit" policy --input | jq --compact-output '.tree["key.pem"].kinds')" = '["secret"]' ]
  # the disk's .kit/ is as it was, and the matcher of the base's kinds is not kept among the disk's
  [ -z "$(git status --porcelain)" ]
  common=$(git rev-parse --path-format=absolute --git-common-dir)
  [ -z "$(find "${common}/kit" -name 'kinds-*.awk' 2>/dev/null)" ]
}

@test "a kind and a rule a package of the base delivers judge a branch that drops the package" {
  command -v opa >/dev/null || skip "opa, the opa policy engine, is not on PATH"
  mkdir -p .kit/packages/mine/files/repo/.kit/policy .kit/packages/mine/files/repo/.kit/kinds
  printf 'description: "rules of our own"\n' > .kit/packages/mine/package.yml
  printf 'name: secret\npaths:\n  - "**/*.pem"\n' > .kit/packages/mine/files/repo/.kit/kinds/secret.yml
  cat > .kit/packages/mine/files/repo/.kit/policy/ours.rego <<'REGO'
package kit.policy

import rego.v1

deny contains sprintf("%s is a secret", [path]) if {
	some path, f in input.tree
	"secret" in f.kinds
}
REGO
  printf 'policy:\n  commit_subject: ""\npackages:\n  - mine\n' > kit.yml
  git add kit.yml .kit/packages
  git commit --quiet --message "#2 kit: a kind and a rule of our own" && git push --quiet origin main
  branch 7-fix-the-thing
  printf 'key\n' > key.pem
  git rm --quiet -r .kit/packages/mine
  printf 'policy:\n  commit_subject: ""\npackages: []\n' > kit.yml
  git add key.pem kit.yml
  git commit --quiet --message "#7 kit: a key, and no package"
  run "${kit_root}/bin/kit" policy
  [ "$status" -eq 1 ]
  [[ "$output" == *"❌ key.pem is a secret"* ]]
}

@test "every file the base holds under .kit/policy/, in a subdirectory or as data, judges a branch" {
  command -v opa >/dev/null || skip "opa, the opa policy engine, is not on PATH"
  mkdir -p .kit/policy/team
  printf 'package kit.policy\n\nimport rego.v1\n\ndeny contains "never" if false\n' > .kit/policy/kit.rego
  cat > .kit/policy/team/strict.rego <<'REGO'
package kit.policy

import rego.v1

deny contains sprintf("a commit with the subject '%s'", [commit.subject]) if {
	some commit in input.commits
	commit.subject in data.team.banned
}
REGO
  printf '{"banned": ["no reference"]}\n' > .kit/policy/team/data.json
  printf 'policy:\n  commit_subject: ""\n' > kit.yml
  git add --force kit.yml .kit/policy
  git commit --quiet --message "#2 kit: the team's rule and its data" && git push --quiet origin main
  branch 7-fix-the-thing "no reference"
  run "${kit_root}/bin/kit" policy
  [ "$status" -eq 1 ]
  [[ "$output" == *"❌ a commit with the subject 'no reference'"* ]]
  printf '{"banned": []}\n' > .kit/policy/team/data.json
  git rm --quiet .kit/policy/team/strict.rego
  git commit --quiet --all --message "#7 kit: nothing banned, and no rule"
  run "${kit_root}/bin/kit" policy
  [ "$status" -eq 1 ]
  [[ "$output" == *"❌ a commit with the subject 'no reference'"* ]]
}

@test "a template a package of the base delivers under .kit/policy/ judges, rendered with the base's answers" {
  command -v opa >/dev/null || skip "opa, the opa policy engine, is not on PATH"
  mkdir -p .kit/packages/mine/templates/repo/.kit/policy
  printf 'description: "rules of our own"\n' > .kit/packages/mine/package.yml
  cat > .kit/packages/mine/templates/repo/.kit/policy/banned.rego <<'REGO'
package kit.policy

import rego.v1

deny contains "a commit with the banned subject" if {
	some commit in input.commits
	commit.subject == "@banned@"
}
REGO
  printf 'policy:\n  commit_subject: ""\npackages:\n  - mine\nanswers:\n  banned: "no reference"\n' > kit.yml
  git add kit.yml .kit/packages
  git commit --quiet --message "#2 kit: a rule from a template" && git push --quiet origin main
  branch 7-fix-the-thing "no reference"
  printf 'policy:\n  commit_subject: ""\npackages:\n  - mine\nanswers:\n  banned: "nothing"\n' > kit.yml
  git commit --quiet --all --message "#7 kit: another answer"
  run "${kit_root}/bin/kit" policy
  [ "$status" -eq 1 ]
  [[ "$output" == *"❌ a commit with the banned subject"* ]]
}

@test "on GitLab, the target branch's tip judges a merge request, not the merge base the branch chose by its fork" {
  printf 'policy:\n  commit_subject: ""\n' > kit.yml
  git add kit.yml
  git commit --quiet --message "#2 kit: a lax policy" && git push --quiet origin main
  old=$(git rev-parse HEAD)
  printf 'policy:\n  commit_subject: "^#[0-9]+ "\n' > kit.yml
  git commit --quiet --all --message "#3 kit: the strict policy" && git push --quiet origin main
  tip=$(git rev-parse origin/main)
  # the branch forks from the commit before the strict policy, so the merge base is the lax commit
  git switch --quiet --create 7-fix-the-thing "${old}"
  git commit --quiet --allow-empty --message "no reference"
  [ "$(git merge-base origin/main HEAD)" = "${old}" ]
  export CI_MERGE_REQUEST_SOURCE_BRANCH_NAME=7-fix-the-thing CI_MERGE_REQUEST_TARGET_BRANCH_NAME=main
  run "${kit_root}/bin/kit" policy
  [ "$status" -eq 1 ]
  [[ "$output" == *"'no reference' does not match policy.commit_subject"* ]]
  [ "$("${kit_root}/bin/kit" policy --input | jq --raw-output '.commits | length')" -eq 1 ]
  # the merge base alone, which the pipeline also names, names no base; the target is what counts
  run env -u CI_MERGE_REQUEST_TARGET_BRANCH_NAME CI_MERGE_REQUEST_DIFF_BASE_SHA="${old}" "${kit_root}/bin/kit" policy
  [ "$status" -eq 1 ]
  [[ "$output" == *"CI_MERGE_REQUEST_TARGET_BRANCH_NAME is not set"*"so its policy cannot be read"* ]]
  [[ "$output" != *"policy allows"* ]]
  # a target that is not here is fetched from origin, and the moved ref counts for nothing
  git update-ref -d refs/remotes/origin/main
  run "${kit_root}/bin/kit" policy
  [ "$status" -eq 1 ]
  [[ "$output" == *"'no reference' does not match policy.commit_subject"* ]]
  [ "$(git rev-parse refs/remotes/origin/main)" = "${tip}" ]
  # a target ref an earlier job left at the lax commit is fetched again, and a tag named origin/main, on origin too,
  # counts for nothing
  git update-ref refs/remotes/origin/main "${old}"
  git tag origin/main "${old}" && git push --quiet origin refs/tags/origin/main
  run "${kit_root}/bin/kit" policy
  [ "$status" -eq 1 ]
  [[ "$output" == *"'no reference' does not match policy.commit_subject"* ]]
  [ "$(git rev-parse refs/remotes/origin/main)" = "${tip}" ]
  # a target origin does not have, and one that is no branch name, fail closed, named
  run env CI_MERGE_REQUEST_TARGET_BRANCH_NAME=nowhere "${kit_root}/bin/kit" policy
  [ "$status" -eq 1 ]
  [[ "$output" == *"origin/nowhere, the merge request's target, could not be fetched"* ]]
  run env CI_MERGE_REQUEST_TARGET_BRANCH_NAME='-x' "${kit_root}/bin/kit" policy
  [ "$status" -eq 1 ]
  [[ "$output" == *"'-x' in CI_MERGE_REQUEST_TARGET_BRANCH_NAME is not a branch name"* ]]
  [ -z "$(git for-each-ref 'refs/remotes/origin/nowhere' 'refs/remotes/origin/-x')" ]
}

@test "at a desk, a tag or a branch named origin/main does not stand for the base" {
  printf 'policy:\n  commit_subject: ""\n' > kit.yml
  git add kit.yml
  git commit --quiet --message "#2 kit: a lax policy" && git push --quiet origin main
  old=$(git rev-parse HEAD)
  printf 'policy:\n  commit_subject: "^#[0-9]+ "\n' > kit.yml
  git commit --quiet --all --message "#3 kit: the strict policy" && git push --quiet origin main
  git switch --quiet --create 7-fix-the-thing
  git commit --quiet --allow-empty --message "no reference"
  for ref in refs/tags/origin/main refs/heads/origin/main
  do
    git update-ref "${ref}" "${old}"
    run "${kit_root}/bin/kit" policy
    [ "$status" -eq 1 ]
    [[ "$output" == *"'no reference' does not match policy.commit_subject"* ]]
    git update-ref -d "${ref}"
  done
}

@test "at a desk the base is the primary's default branch, and on GitLab the target is fetched from the primary" {
  printf 'policy:\n  commit_subject: ""\n' > kit.yml
  git add kit.yml
  git commit --quiet --message "#2 kit: a lax policy" && git push --quiet origin main
  printf 'policy:\n  commit_subject: "^#[0-9]+ "\n' > kit.yml
  git commit --quiet --all --message "#3 kit: the strict policy"
  git init --quiet --bare --initial-branch=main ../upstream.git
  git remote add upstream ../upstream.git
  git push --quiet upstream main && git fetch --quiet upstream && git remote set-head upstream main
  tip=$(git rev-parse upstream/main)
  # origin/main holds the lax policy and upstream/main the strict one; the primary is the checkout's
  git switch --quiet --create 7-fix-the-thing
  git commit --quiet --allow-empty --message "no reference"
  printf 'primary: upstream\npolicy:\n  commit_subject: ""\n' > kit.yml
  run "${kit_root}/bin/kit" policy
  [ "$status" -eq 1 ]
  [[ "$output" == *"'no reference' does not match policy.commit_subject"* ]]
  [ "$("${kit_root}/bin/kit" policy --input | jq --raw-output '.commits | length')" -eq 1 ]
  git update-ref -d refs/remotes/upstream/main
  run "${kit_root}/bin/kit" policy
  [ "$status" -eq 1 ]
  [[ "$output" == *"the base, upstream/main, is not here"* ]]
  export CI_MERGE_REQUEST_SOURCE_BRANCH_NAME=7-fix-the-thing CI_MERGE_REQUEST_TARGET_BRANCH_NAME=main
  run "${kit_root}/bin/kit" policy
  [ "$status" -eq 1 ]
  [[ "$output" == *"'no reference' does not match policy.commit_subject"* ]]
  [ "$(git rev-parse refs/remotes/upstream/main)" = "${tip}" ]
  run env CI_MERGE_REQUEST_TARGET_BRANCH_NAME=nowhere "${kit_root}/bin/kit" policy
  [ "$status" -eq 1 ]
  [[ "$output" == *"upstream/nowhere, the merge request's target, could not be fetched"* ]]
}

@test "a base the primary's unrecorded HEAD names says to record it, and a repository without the remote says so" {
  branch 7-fix-the-thing "#7 cli: fix the thing"
  git update-ref -d refs/remotes/origin/main
  git remote set-head origin --delete
  run "${kit_root}/bin/kit" policy
  [ "$status" -eq 1 ]
  said='run: git remote set-head origin --auto (origin/HEAD is not recorded, so the default branch is taken to be main)'
  [[ "$output" == *"${said}"* ]]
  [[ "$output" != *"policy allows"* ]]
  git remote remove origin
  run "${kit_root}/bin/kit" policy
  [ "$status" -eq 1 ]
  said='run: git remote add origin <url> (there is no remote origin, so the base, origin/main, and its policy cannot'
  [[ "$output" == *"${said} be read)"* ]]
  [[ "$output" != *"policy allows"* ]]
}

@test "a merge group is judged by the base commit its event names, as the change on the merge group's head" {
  printf 'policy:\n  branch: ""\n  commit_subject: "^#[0-9]+ "\n' > kit.yml
  git add kit.yml
  git commit --quiet --message "#2 kit: the policy" && git push --quiet origin main
  base=$(git rev-parse origin/main)
  branch gh-readonly-queue/main/pr-3-abc
  printf 'policy:\n  branch: ""\n  commit_subject: ""\n' > kit.yml
  git commit --quiet --all --message "no reference"
  printf '{"merge_group": {"base_sha": "%s", "head_ref": "refs/heads/gh-readonly-queue/main/pr-3-abc"}}\n' \
    "${base}" > "${BATS_TEST_TMPDIR}/event.json"
  export GITHUB_EVENT_NAME=merge_group GITHUB_EVENT_PATH="${BATS_TEST_TMPDIR}/event.json"
  git update-ref refs/remotes/origin/main HEAD
  run "${kit_root}/bin/kit" policy
  [ "$status" -eq 1 ]
  [[ "$output" == *"'no reference' does not match policy.commit_subject"* ]]
  # the head is the merge group's, from the event, on a detached checkout too
  git switch --quiet --detach
  input=$("${kit_root}/bin/kit" policy --input)
  [ "$(jq --raw-output .branch <<<"${input}")" = gh-readonly-queue/main/pr-3-abc ]
  [ "$(jq --raw-output '.commits | length' <<<"${input}")" -eq 1 ]
  run "${kit_root}/bin/kit" policy
  [ "$status" -eq 1 ]
  [[ "$output" == *"'no reference' does not match policy.commit_subject"* ]]
  # an event with no base commit fails closed
  printf '{"merge_group": {"head_ref": "refs/heads/gh-readonly-queue/main/pr-3-abc"}}\n' \
    > "${BATS_TEST_TMPDIR}/event.json"
  run "${kit_root}/bin/kit" policy
  [ "$status" -eq 1 ]
  [[ "$output" == *"merge_group.base_sha is not in the event"*"so its policy cannot be read"* ]]
  [[ "$output" != *"policy allows"* ]]
}

@test "nothing on the disk under .kit/policy/ is read while a branch is judged" {
  printf 'policy:\n  commit_subject: "^#[0-9]+ "\n' > kit.yml
  git add kit.yml
  git commit --quiet --message "#2 kit: the policy" && git push --quiet origin main
  branch 7-fix-the-thing "no reference"
  # a directory under the name a Rego file has: a copy of it fails, as a read of a FIFO there would hang
  mkdir -p .kit/policy/lax.rego
  run "${kit_root}/bin/kit" policy
  [ "$status" -eq 1 ]
  [[ "$output" == *"'no reference' does not match policy.commit_subject"* ]]
  [[ "$output" == *"1 refusal(s) by the patterns policy"* ]]
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

@test "kit policy --rules --json lists each rule, its pattern, where it is set, and the engine" {
  printf "policy:\n  branch: '^feature/[a-z-]+\$'\n  change_title: ''\n" > kit.yml
  run "${kit_root}/bin/kit" policy --rules --json
  [[ "$status" -eq 0 && "${#lines[@]}" -eq 3 && "$(jq --raw-output .from <<<"${lines[1]}")" == default ]]
  [ "${lines[0]}" = '{"rule":"branch","pattern":"^feature/[a-z-]+$","from":"kit.yml","engine":"patterns"}' ]
  [ "${lines[2]}" = '{"rule":"change_title","pattern":"","from":"kit.yml","engine":"patterns"}' ]
}

@test "kit policy --rules on a branch past the default lists the base's rules, not the branch's own" {
  printf "forge: github\npolicy:\n  branch: '^base-[a-z]+\$'\n" > kit.yml
  git add kit.yml
  git commit --quiet --message "#2 kit: the policy" && git push --quiet origin main
  branch 7-fix-the-thing "#7 cli: fix the thing"
  printf "policy:\n  branch: ''\n" > kit.yml
  run "${kit_root}/bin/kit" policy --rules --json
  [ "$status" -eq 0 ]
  [ "${lines[0]}" = '{"rule":"branch","pattern":"^base-[a-z]+$","from":"kit.yml","engine":"patterns"}' ]
}

@test "kit policy --rules prints a table of the three rules, (off) for an empty pattern, and --jq formats the JSON" {
  printf "policy:\n  change_title: ''\n" > kit.yml
  run "${kit_root}/bin/kit" policy --rules
  [ "$status" -eq 0 ]
  [[ "${lines[0]}" == *rule*from*engine*pattern* && "${#lines[@]}" -eq 4 ]]
  [[ "${lines[1]}" == branch*default*patterns*'^[a-z0-9]+-[a-z0-9][a-z0-9-]*$' ]]
  [[ "${lines[3]}" == change_title*kit.yml*patterns*'(off)' ]]
  run "${kit_root}/bin/kit" policy --rules --jq
  [ "$status" -eq 0 ]
  [ "${lines[0]}" = '{' ]
  [ "$(jq --slurp --raw-output '[.[].rule] | join(",")' <<<"${output}")" = branch,commit_subject,change_title ]
}
