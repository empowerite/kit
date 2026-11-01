#!/usr/bin/env bats
# linked.bats - kit linked names the pull request it needs when none can be found; the forge fakes test the rest
# ----------------------------------------------------------------------------------------------------------------------

load helper

setup() {
  PATH=$(shims gh)
  fresh-repo
  git remote add origin https://github.com/empowerite/example.git
  export GITHUB_STEP_SUMMARY="${BATS_TEST_TMPDIR}/summary.md"
}

@test "linked names the pull request it needs when none can be found" {
  run env -u GITHUB_EVENT_PATH "${kit_checkout}/bin/kit" linked
  [ "$status" -eq 1 ]
  [[ "$output" == *"kit linked <number>"* ]]
}

# ----------------------------------------------------------------------------------------------------------------------
