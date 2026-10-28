#!/usr/bin/env bats
# policy.bats - the facts, the patterns and their defaults, and two engines that refuse the same things the same way
# ----------------------------------------------------------------------------------------------------------------------

setup() {
  export kit_root NO_COLOR=1
  kit_root=$(cd "${BATS_TEST_DIRNAME}/.." && pwd)
  export KIT_HOME="${BATS_TEST_TMPDIR}/home" XDG_CACHE_HOME="${BATS_TEST_TMPDIR}/cache"
  export FAKE_DIR="${BATS_TEST_TMPDIR}/fake"
  mkdir -p "${KIT_HOME}" "${XDG_CACHE_HOME}" "${FAKE_DIR}/bin"
  ln -s "${BATS_TEST_DIRNAME}/fakes/fake-tool" "${FAKE_DIR}/bin/gh"
  : > "${FAKE_DIR}/gh.routes"
  export PATH="${FAKE_DIR}/bin:${PATH}"
  unset GITHUB_HEAD_REF CI_MERGE_REQUEST_SOURCE_BRANCH_NAME GITHUB_EVENT_PATH
  work=$(mktemp -d)
  cd "${work}"
  git init --quiet --bare --initial-branch=main origin.git
  git clone --quiet origin.git desk 2>/dev/null
  cd desk
  git config user.name t && git config user.email t@example.com && git config commit.gpgsign false
  git commit --quiet --allow-empty --message "#1 start: the first" && git push --quiet origin main
  git remote set-head origin main
  mkdir .kit
}

teardown() {
  rm -rf "${work}"
}

fixture() { cat > "${FAKE_DIR}/$1"; }
route() { printf '%s\t%s\n' "$2" "$3" >> "${FAKE_DIR}/$1.routes"; }

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
  grep --quiet '^forge:' .kit/settings.yml 2>/dev/null || printf 'forge: github\n' >> .kit/settings.yml
  fixture pr.json <<<"{\"number\": 3, \"title\": \"$1\", \"body\": \"Closes #7\"}"
  route gh 'pr view --json number*' pr.json
  route gh 'pr view 3 --json *' pr.json
}

@test "the facts are the branch, the commits past the fork, the change, and the rules with the kit's defaults" {
  branch 7-fix-the-thing "#7 cli: fix the thing" "AB#4521 docs: say so"
  change "Fix the thing"
  run "${kit_root}/bin/kit" policy --input
  [ "$status" -eq 0 ]
  [ "$(jq --raw-output .branch <<<"$output")" = "7-fix-the-thing" ]
  [ "$(jq --raw-output '[.commits[].subject] | join("|")' <<<"$output")" = "AB#4521 docs: say so|#7 cli: fix the thing" ]
  [ "$(jq --raw-output .change.title <<<"$output")" = "Fix the thing" ]
  [ "$(jq --raw-output '.policy | keys | join(" ")' <<<"$output")" = "branch change_title commit_subject" ]
  run "${kit_root}/bin/kit" policy
  [ "$status" -eq 0 ]
  [[ "$output" == *"the patterns policy allows the branch, 2 commit(s), the change title"* ]]
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

@test "the repository's patterns override the defaults, an empty one turns a rule off, and an unknown subject is refused" {
  branch feature/login "wip"
  printf "policy:\n  branch: '^feature/[a-z-]+\$'\n  commit_subject: ''\n" > .kit/settings.yml
  run "${kit_root}/bin/kit" policy
  [ "$status" -eq 0 ]
  printf "policy:\n  branche: '^x\$'\n" > .kit/settings.yml
  run "${kit_root}/bin/kit" policy
  [ "$status" -ne 0 ]
  [[ "$output" == *"policy has a subject the kit does not know: branche"* ]]
}

@test "on the default branch, or detached, nothing is under check" {
  run "${kit_root}/bin/kit" policy
  [ "$status" -eq 0 ]
  [[ "$output" == *"allows 0 commit(s)"* ]]
  git commit --quiet --allow-empty --message "not checked"
  git switch --quiet --detach
  run "${kit_root}/bin/kit" policy
  [ "$status" -eq 0 ]
}

@test "the opa engine, given the policy-opa module's Rego, refuses exactly what the patterns engine does" {
  command -v opa >/dev/null || skip "opa, the opa policy engine, is not on PATH"
  "${kit_root}/bin/kit" install policy-opa >/dev/null 2>&1
  [ -f .kit/policy/kit.rego ]
  branch topic "#7 cli: fine" "wip" "Also wrong"
  change "A title that is fine"
  run "${kit_root}/bin/kit" policy
  [ "$status" -eq 1 ]
  [[ "$output" == *"by the opa policy"* ]]
  by_opa=$(grep '^❌ ' <<<"$output" | grep --invert-match 'refusal(s)' | LC_ALL=C sort)
  printf 'forge: github\npolicy_engine: patterns\n' > .kit/settings.yml
  run "${kit_root}/bin/kit" policy
  [ "$status" -eq 1 ]
  by_patterns=$(grep '^❌ ' <<<"$output" | grep --invert-match 'refusal(s)' | LC_ALL=C sort)
  [ "$(wc -l <<<"${by_opa}")" -eq 3 ]
  [ "${by_opa}" = "${by_patterns}" ]
}

@test "a rule of the repository's own in Rego adds to the kit's, and a policy engine the kit does not know is refused" {
  command -v opa >/dev/null || skip "opa, the opa policy engine, is not on PATH"
  "${kit_root}/bin/kit" install policy-opa >/dev/null 2>&1
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
  printf 'policy_engine: cue\n' > .kit/settings.yml
  run "${kit_root}/bin/kit" policy
  [ "$status" -ne 0 ]
  [[ "$output" == *"'cue' is not a policy engine the kit knows"* ]]
}

# ----------------------------------------------------------------------------------------------------------------------
