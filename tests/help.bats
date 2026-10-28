#!/usr/bin/env bats
# help.bats - kit help reads every verb's usage from its header: one verb gets its page, several get their usage
# ----------------------------------------------------------------------------------------------------------------------

setup() {
  export kit_root NO_COLOR=1
  kit_root=$(cd "${BATS_TEST_DIRNAME}/../kit" && pwd)
}

@test "kit help lists every verb beside bin/kit, with its purpose, and kit alone or --help does the same" {
  run "${kit_root}/bin/kit" help
  [ "$status" -eq 0 ]
  local verb
  for verb in "${kit_root}"/bin/kit-*
  do
    verb=${verb##*/kit-}
    [[ "$output" == *$'\n'"  ${verb}"[\ $'\n']* ]] || { echo "missing: ${verb}"; return 1; }
  done
  [[ "$output" == *"  tidy [--dry-run]      remove merged worktrees and gone branches"* ]]
  [[ "$output" == *"  install [<module>…]   mirror the release"* ]]
  local listed=$output
  run "${kit_root}/bin/kit"
  [ "$output" = "${listed}" ]
  run "${kit_root}/bin/kit" --help
  [ "$output" = "${listed}" ]
}

@test "kit help <verb> prints the verb's page: purpose, usage, and description, without its source pointers" {
  run "${kit_root}/bin/kit" help tidy
  [ "$status" -eq 0 ]
  [ "${lines[0]}" = "kit tidy - remove merged worktrees and gone branches" ]
  [[ "$output" == *$'Usage\n  kit tidy [--dry-run]\n\nDescription\n  Removes finished worktrees'* ]]
  [[ "$output" != *"# "* ]]
  [[ "$output" != *"see:"* ]]
}

@test "a verb with two usage lines shows both, and one that takes no arguments shows kit <verb>" {
  run "${kit_root}/bin/kit" help context
  [[ "$output" == *$'Usage\n  kit context <path>… [--json]\n  kit context --claude\n'* ]]
  run "${kit_root}/bin/kit" help check
  [[ "$output" == *$'Usage\n  kit check\n\nDescription\n  kit check takes no arguments.'* ]]
}

@test "kit help <prefix> lists every verb that starts with it, and a verb's own name wins over a longer one" {
  run "${kit_root}/bin/kit" help re
  [ "$status" -eq 0 ]
  [[ "$output" == *"  rebase [<path>]"* && "$output" == *"  region [--home]"* && "$output" == *"  release verify"* ]]
  [[ "$output" != *"  status"* ]]
  run "${kit_root}/bin/kit" help gate
  [ "${lines[0]}" = "kit gate - run the repository's gate, and refuse when there is none" ]
}

@test "kit help refuses a name no verb starts with, and more than one name" {
  run "${kit_root}/bin/kit" help nonesuch
  [ "$status" -eq 2 ]
  [[ "$output" == *"no kit verb matches 'nonesuch'; run: kit help"* ]]
  run "${kit_root}/bin/kit" help tidy gate
  [ "$status" -eq 2 ]
}

# ----------------------------------------------------------------------------------------------------------------------
