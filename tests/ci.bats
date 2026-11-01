#!/usr/bin/env bats
# ci.bats - kit ci runs the gate, kit check, kit linked, and kit policy, and reports every one
# ----------------------------------------------------------------------------------------------------------------------

load helper

setup() {
  PATH=$(shims gh)
  fresh-repo
  git remote add origin https://github.com/empowerite/example.git
  export GITHUB_STEP_SUMMARY="${BATS_TEST_TMPDIR}/summary.md"
  "${kit_checkout}/bin/kit" install >/dev/null
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
  [ "$status" -eq 1 ]
  [[ "$output" == *"3 of 4 checks refused"* ]]
}

# ----------------------------------------------------------------------------------------------------------------------
