#!/usr/bin/env bats
# gate.bats - kit gate runs the root justfile's ci recipe, and refuses a missing one and an unreadable one alike
# ----------------------------------------------------------------------------------------------------------------------

setup() {
  kit_checkout=$(cd "${BATS_TEST_DIRNAME}/.." && pwd)
  export KIT_HOME="${BATS_TEST_TMPDIR}/home" XDG_CACHE_HOME="${BATS_TEST_TMPDIR}/cache"
  mkdir -p "${KIT_HOME}" "${XDG_CACHE_HOME}"
  repo=$(mktemp -d)
  cd "${repo}"
  export GITHUB_STEP_SUMMARY="${repo}/summary.md"
}

teardown() {
  rm -rf "${repo}"
}

@test "no justfile is refused, naming the remedy" {
  run "${kit_checkout}/bin/kit" gate
  [ "$status" -eq 1 ]
  [[ "$output" == *"add a root justfile with a ci recipe"* ]]
  grep --quiet 'cannot run' summary.md
}

@test "a justfile without a ci recipe is refused" {
  printf 'build:\n  echo built\n' > justfile
  run "${kit_checkout}/bin/kit" gate
  [ "$status" -eq 1 ]
  [[ "$output" == *"add a ci recipe to the root justfile"* ]]
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

# ----------------------------------------------------------------------------------------------------------------------
