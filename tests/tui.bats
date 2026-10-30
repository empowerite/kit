#!/usr/bin/env bats
# tui.bats - kit with no verb runs kit tui, which this release refuses with a pointer to kit help
# ----------------------------------------------------------------------------------------------------------------------

setup() {
  export kit_root NO_COLOR=1
  kit_root=$(cd "${BATS_TEST_DIRNAME}/../kit" && pwd)
}

@test "kit with no verb runs kit tui, and this release's kit tui refuses with a pointer to kit help" {
  run "${kit_root}/bin/kit" tui
  [ "$status" -eq 2 ]
  [[ "$output" == *"no interactive terminal UI; run: kit help"* ]]
  local tui=$output
  run "${kit_root}/bin/kit"
  [ "$status" -eq 2 ]
  [ "$output" = "${tui}" ]
}

@test "kit tui takes no arguments" {
  run "${kit_root}/bin/kit" tui --json
  [ "$status" -eq 2 ]
  [[ "$output" == *"kit tui takes no arguments; run: kit help tui"* ]]
}

# ----------------------------------------------------------------------------------------------------------------------
