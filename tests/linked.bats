#!/usr/bin/env bats
# linked.bats - kit linked names the pull request it needs when none can be found; the rest needs GitHub
# ----------------------------------------------------------------------------------------------------------------------

setup() {
  kit_checkout=$(cd "${BATS_TEST_DIRNAME}/.." && pwd)
  export KIT_HOME="${BATS_TEST_TMPDIR}/home" XDG_CACHE_HOME="${BATS_TEST_TMPDIR}/cache"
  mkdir -p "${KIT_HOME}" "${XDG_CACHE_HOME}"
  repo=$(mktemp -d)
  cd "${repo}"
  git init --quiet --initial-branch=main
  git remote add origin https://github.com/empowerite/example.git
}

teardown() {
  rm -rf "${repo}"
}

@test "linked names the pull request it needs when none can be found" {
  run env -u GITHUB_EVENT_PATH "${kit_checkout}/bin/kit" linked
  [ "$status" -eq 2 ]
  [[ "$output" == *"kit linked <number>"* ]]
}

# ----------------------------------------------------------------------------------------------------------------------
