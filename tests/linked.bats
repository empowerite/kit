#!/usr/bin/env bats
# linked.bats - kit linked names the pull request it needs when none can be found; the forge fakes test the rest
# ----------------------------------------------------------------------------------------------------------------------

load helper

setup() {
  fresh-repo
  git remote add origin https://github.com/empowerite/example.git
  export GITHUB_STEP_SUMMARY="${BATS_TEST_TMPDIR}/summary.md"
}

@test "linked names the pull request it needs when none can be found" {
  # gh answers that the branch has no pull request: its filter of an empty list prints nothing
  mkdir -p "${BATS_TEST_TMPDIR}/bin"
  printf '#!/usr/bin/env bash\nexit 0\n' > "${BATS_TEST_TMPDIR}/bin/gh" && chmod +x "${BATS_TEST_TMPDIR}/bin/gh"
  PATH="${BATS_TEST_TMPDIR}/bin:${PATH}" run env -u GITHUB_EVENT_PATH "${kit_checkout}/bin/kit" linked
  [ "$status" -eq 1 ]
  [[ "$output" == *"kit linked <number>"* ]]
}

@test "linked fails, naming the read, when the forge cannot say which pull request this checkout is working on" {
  PATH=$(shims gh) run env -u GITHUB_EVENT_PATH "${kit_checkout}/bin/kit" linked
  [ "$status" -eq 1 ]
  [[ "$output" == *"run: gh auth status (the open pull request on main could not be read)"* ]]
  [[ "$output" != *"kit linked <number>"* ]]
}

@test "on a branch, kit linked exempts the logins its base's kit.yml lists, and on the default branch the disk's" {
  fake-tools gh
  fixture repo.json <<<'{"nameWithOwner": "empowerite/example", "url": "https://github.com/empowerite/example"}'
  fixture pull.json <<<'{"user": {"login": "ada"}}'
  fixture pr.json <<<'{"body": ""}'
  fixture closing.json <<<'{"data": {"repository": {"pullRequest": {"closingIssuesReferences": {"nodes": []}}}}}'
  route gh 'repo view*' repo.json
  route gh 'api repos/empowerite/example/pulls/3 *' pull.json
  route gh 'pr view 3 --json *' pr.json
  route gh 'api graphql*' closing.json
  printf 'linked_exempt:\n  - bob\n' > kit.yml
  git add kit.yml
  git commit --quiet --message "#1 kit: bob is exempt"
  git update-ref refs/remotes/origin/main HEAD
  git switch --quiet --create 7-fix-the-thing
  printf 'linked_exempt:\n  - ada\n' > kit.yml
  run "${kit_checkout}/bin/kit" linked 3
  [ "$status" -eq 1 ]
  [[ "$output" == *"pull request #3 is not linked to any issue"* ]]
  [[ "$output" != *"exempt"* ]]
  git switch --quiet main
  run "${kit_checkout}/bin/kit" linked 3
  [ "$status" -eq 0 ]
  [[ "$output" == *"pull request #3 is ada's, exempt from the linked-issue check by kit.yml"* ]]
}

@test "an issue whose state its tracker cannot read is named, and never called closed" {
  fake-tools gh az
  printf 'trackers:\n  - prefix: "AB#"\n    provider: ado\n    organization: https://dev.azure.com/example\n' > kit.yml
  printf '    project: Claims\n' >> kit.yml
  fixture repo.json <<<'{"nameWithOwner": "empowerite/example", "url": "https://github.com/empowerite/example"}'
  fixture closing.json <<<'{"data": {"repository": {"pullRequest": {"closingIssuesReferences": {"nodes": []}}}}}'
  fixture pr.json <<<'{"body": "Closes AB#4521"}'
  route gh 'repo view*' repo.json
  route gh 'api graphql*' closing.json
  route gh 'pr view 3 --json *' pr.json
  route az 'boards work-item show --id 4521 *' '!1'
  run "${kit_checkout}/bin/kit" linked 3
  [ "$status" -eq 1 ]
  [[ "$output" == *"AB#4521's state could not be read, so it does not count"* ]]
  [[ "$output" == *"the state of 1 linked issue(s) could not be read"* ]]
  [[ "$output" != *"closed"* ]]
  grep --quiet 'boards work-item show --id 4521' "${FAKE_DIR}/az.calls"
  grep --quiet 'could not be read' "${GITHUB_STEP_SUMMARY}"
}

@test "on a branch, kit linked fails when the base is not here, since the logins it exempts cannot be read" {
  fake-tools gh
  git commit --quiet --allow-empty --message "#1 start: the first"
  git switch --quiet --create 7-fix-the-thing
  run "${kit_checkout}/bin/kit" linked 3
  [ "$status" -eq 1 ]
  # origin's HEAD is not recorded, so the default branch is taken to be main, which is not here either
  [[ "$output" == *"run: git remote set-head origin --auto (origin/HEAD is not recorded"* ]]
  [ ! -s "${FAKE_DIR}/gh.calls" ]
  git symbolic-ref refs/remotes/origin/HEAD refs/remotes/origin/main
  run "${kit_checkout}/bin/kit" linked 3
  [ "$status" -eq 1 ]
  [[ "$output" == *"the base, origin/main, is not here, so its policy cannot be read"* ]]
  [ ! -s "${FAKE_DIR}/gh.calls" ]
}

# ----------------------------------------------------------------------------------------------------------------------
