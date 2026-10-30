#!/usr/bin/env bats
# ci.bats - kit ci runs the gate, kit check, kit linked, and kit policy, and reports every one
# ----------------------------------------------------------------------------------------------------------------------

setup() {
  kit_checkout=$(cd "${BATS_TEST_DIRNAME}/../kit" && pwd)
  export KIT_HOME="${BATS_TEST_TMPDIR}/home" XDG_CACHE_HOME="${BATS_TEST_TMPDIR}/cache"
  mkdir -p "${KIT_HOME}" "${XDG_CACHE_HOME}" "${BATS_TEST_TMPDIR}/shims"
  printf '#!/usr/bin/env bash\nexit 1\n' > "${BATS_TEST_TMPDIR}/shims/gh" && chmod +x "${BATS_TEST_TMPDIR}/shims/gh"
  export PATH="${BATS_TEST_TMPDIR}/shims:${PATH}"
  repo=$(mktemp -d)
  cd "${repo}"
  git init --quiet --initial-branch=main
  git remote add origin https://github.com/empowerite/example.git
  export GITHUB_STEP_SUMMARY="${BATS_TEST_TMPDIR}/summary.md"
  "${kit_checkout}/bin/kit" install >/dev/null
}

teardown() {
  rm -rf "${repo}"
}

@test "every check runs and the refusals are counted" {
  printf 'ci:\n  echo gate ran\n' > justfile
  run env -u GITHUB_EVENT_PATH "${kit_checkout}/bin/kit" ci
  [ "$status" -eq 1 ]
  [[ "$output" == *"gate ran"* ]]
  [[ "$output" == *"every delivered file matches"* ]]
  [[ "$output" == *"kit linked <number>"* ]]
  [[ "$output" == *"the patterns policy allows"* ]]
  [[ "$output" == *"1 of 4 checks refused"* ]]
  # a stray file under a delivered directory: check reports it, and no verb heals it
  printf 'rogue\n' > .kit/rules/rogue.md
  rm justfile
  run env -u GITHUB_EVENT_PATH "${kit_checkout}/bin/kit" ci
  [ "$status" -eq 3 ]
  [[ "$output" == *"3 of 4 checks refused"* ]]
}

# ----------------------------------------------------------------------------------------------------------------------
