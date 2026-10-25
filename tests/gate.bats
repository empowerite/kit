#!/usr/bin/env bats
# gate.bats - kit gate runs the root justfile's ci recipe, and reports a missing one as a skip and an unreadable one as a failure
# ----------------------------------------------------------------------------------------------------------------------

setup() {
  kit_checkout=$(cd "${BATS_TEST_DIRNAME}/.." && pwd)
  repo=$(mktemp -d)
  cd "${repo}"
  export GITHUB_STEP_SUMMARY="${repo}/summary.md"
}

teardown() {
  rm -rf "${repo}"
}

@test "no justfile is a skip, said so, and not a pass" {
  run "${kit_checkout}/bin/kit" gate
  [ "$status" -eq 0 ]
  [[ "$output" == *"no root justfile, so the gate was skipped"* ]]
  grep --quiet 'skipped' summary.md
}

@test "a justfile without a ci recipe is a skip" {
  printf 'build:\n  echo built\n' > justfile
  run "${kit_checkout}/bin/kit" gate
  [ "$status" -eq 0 ]
  [[ "$output" == *"no ci recipe, so the gate was skipped"* ]]
}

@test "a justfile that cannot be read is a failure" {
  printf 'ci:\n  echo\nci:\n  echo twice\n' > justfile
  run "${kit_checkout}/bin/kit" gate
  [ "$status" -eq 1 ]
  [[ "$output" == *"the root justfile could not be read"* ]]
  grep --quiet 'could not be read' summary.md
}

@test "a ci recipe runs, and its failure is the gate's" {
  printf 'ci:\n  echo gate ran\n' > justfile
  run "${kit_checkout}/bin/kit" gate
  [ "$status" -eq 0 ]
  [[ "$output" == *"gate ran"* ]]
  grep --quiet 'just ci passed' summary.md
  printf 'ci:\n  exit 3\n' > justfile
  run "${kit_checkout}/bin/kit" gate
  [ "$status" -ne 0 ]
}

@test "linked names the pull request it needs when none can be found" {
  run env -u GITHUB_EVENT_PATH "${kit_checkout}/bin/kit" linked
  [ "$status" -eq 2 ]
  [[ "$output" == *"kit linked <number>"* ]]
}

# ----------------------------------------------------------------------------------------------------------------------
