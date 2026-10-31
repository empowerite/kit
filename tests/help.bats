#!/usr/bin/env bats
# help.bats - kit help reads every verb's usage from its header: one verb gets its page, several get their usage
# ----------------------------------------------------------------------------------------------------------------------

setup() {
  export kit_root NO_COLOR=1
  kit_root=$(cd "${BATS_TEST_DIRNAME}/../kit" && pwd)
}

@test "kit help lists every verb beside bin/kit with its purpose, under its group; -h and --help are unknown verbs that point to it" {
  run "${kit_root}/bin/kit" help
  [ "$status" -eq 0 ]
  local verb
  for verb in "${kit_root}"/bin/kit-*
  do
    [[ -x "${verb}" ]] || continue
    verb=${verb##*/kit-}
    [[ "$output" == *$'\n'"  ${verb}: "* ]] || { echo "missing: ${verb}"; return 1; }
  done
  [[ "$output" == *$'WORKFLOW\n'*"  tidy:        removes merged worktrees and gone branches"$'\n\nFILES\n'* ]]
  [[ "$output" == *$'INSTALLATION\n'*"  install:     mirrors the release"* ]]
  [[ "$output" == *$'USAGE\n  kit <verb> [<argument>…]\n'* ]]
  [[ "$output" == *$'\nEXAMPLES\n'* && "$output" == *$'\nLEARN MORE\n'* ]]
  [[ "$output" != *$'\nOTHER\n'* ]]
  run "${kit_root}/bin/kit" --help
  [ "$status" -eq 2 ]
  [ "$output" = "kit: unknown verb '--help'; run: kit help" ]
  run "${kit_root}/bin/kit" -h
  [ "$status" -eq 2 ]
  [ "$output" = "kit: unknown verb '-h'; run: kit help" ]
}

@test "every verb names one of the groups in its banner, and its page names the group and the verbs beside it" {
  local verb group
  for verb in "${kit_root}"/bin/kit-*
  do
    [[ -x "${verb}" ]] || continue
    group=$(sed -n '2s/^.* (kit verb: \([a-z]*\))$/\1/p' "${verb}")
    [[ " workflow files checks installation interfaces help " == *" ${group} "* && -n "${group}" ]] \
      || { echo "no known group: ${verb##*/}"; return 1; }
  done
  run "${kit_root}/bin/kit" help kind
  [[ "$output" == *$'\nGroup\n  files: also kit context, kit generate, kit new, kit region'* ]]
}

@test "kit usage prints the usage lines alone, of the named verbs or of every verb, and refuses a name that is no verb" {
  run "${kit_root}/bin/kit" usage context check
  [ "$status" -eq 0 ]
  [ "$output" = $'kit context <path>… [--json]\nkit context --claude\nkit check' ]
  run "${kit_root}/bin/kit" usage
  [ "$status" -eq 0 ]
  [ "${#lines[@]}" -ge "$(find "${kit_root}/bin" -name 'kit-*' -perm -u+x | wc -l)" ]
  [[ "$output" != *"Usage"* && "$output" != *"Description"* ]]
  run "${kit_root}/bin/kit" usage che
  [ "$status" -eq 2 ]
  [[ "$output" == *"no kit verb 'che'; run: kit help"* ]]
}

@test "no verb takes --help: each refuses it with a pointer to its page, and kit help knows no verb by that name" {
  cd "${BATS_TEST_TMPDIR}" && git init --quiet --initial-branch=main
  local verb
  for verb in "${kit_root}"/bin/kit-*
  do
    [[ -x "${verb}" ]] || continue
    verb=${verb##*/kit-}
    [[ "${verb}" != help ]] || continue
    run "${kit_root}/bin/kit" "${verb}" --help
    [ "$status" -eq 2 ] || { echo "${verb}: exit ${status}: ${output}"; return 1; }
    [[ "$output" == *"; run: kit help ${verb}"* ]] || { echo "${verb}: ${output}"; return 1; }
  done
  run "${kit_root}/bin/kit" help --help
  [ "$status" -eq 2 ]
  [[ "$output" == *"unknown flag '--help'; run: kit help help"* ]]
}

@test "kit help <verb> prints the verb's page: purpose, usage, and description, without its source pointers" {
  run "${kit_root}/bin/kit" help tidy
  [ "$status" -eq 0 ]
  [ "${lines[0]}" = "kit tidy - removes merged worktrees and gone branches" ]
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
  [[ "$output" == *"  rebase [<path>]"* && "$output" == *"  region [--home]"* ]]
  [[ "$output" == *"  release <version> [--lower]"* && "$output" == *"  releases [--json]"* ]]
  [[ "$output" != *"  status"* ]]
  # release is a prefix of releases, and its own name wins
  run "${kit_root}/bin/kit" help release
  [ "${lines[0]}" = "kit release - tags HEAD as a release and pushes the tag, after the rules a release keeps" ]
  # and with a copy of the kit that has kit-gates beside kit-gate
  local root="${BATS_TEST_TMPDIR}/kit"
  mkdir -p "${root}"
  cp -R "${kit_root}/bin" "${kit_root}/modules" "${root}/"
  cp "${root}/bin/kit-gate" "${root}/bin/kit-gates"
  run "${root}/bin/kit" help gat
  [[ "$output" == *"  gate "* && "$output" == *"  gates "* ]]
  run "${root}/bin/kit" help gate
  [ "${lines[0]}" = "kit gate - runs the repository's gate, and refuses when there is none" ]
}

@test "a verb with no line of dashes under its banner keeps the line that follows the banner" {
  local root="${BATS_TEST_TMPDIR}/kit"
  mkdir -p "${root}"
  cp -R "${kit_root}/bin" "${kit_root}/modules" "${root}/"
  printf '%s\n' '#!/usr/bin/env bash' '# kit-plain - a verb in no house style (kit verb)' '# kit plain <thing>' '#' \
    '# Does it.' > "${root}/bin/kit-plain"
  chmod +x "${root}/bin/kit-plain"
  run "${root}/bin/kit" help plain
  [ "$status" -eq 0 ]
  [[ "$output" == *$'Usage\n  kit plain <thing>\n\nDescription\n  Does it.'* ]]
}

@test "kit help refuses a name no verb starts with, and more than one name" {
  run "${kit_root}/bin/kit" help nonesuch
  [ "$status" -eq 2 ]
  [[ "$output" == *"no kit verb matches 'nonesuch'; run: kit help"* ]]
  run "${kit_root}/bin/kit" help tidy gate
  [ "$status" -eq 2 ]
}

# ----------------------------------------------------------------------------------------------------------------------
