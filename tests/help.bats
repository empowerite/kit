#!/usr/bin/env bats
# help.bats - kit help reads every verb's usage from its header: one verb gets its page, several get their usage
# ----------------------------------------------------------------------------------------------------------------------

load helper

# group-has <GROUP> <verb> <purpose> passes when kit help's output lists <verb> with a purpose that starts with
# <purpose> under <GROUP>, whatever the padding and the verbs beside it.
group-has() {
  local group=${output#*$'\n'"$1"$'\n'} line=$'\n'"  $2 +$3"
  group=${group%%$'\n\n'*}
  [[ $'\n'"${group}" =~ ${line} ]] || { echo "$2 is not under $1"; return 1; }
}

@test "kit help lists every verb beside bin/kit with its purpose, under its group; kit --help and kit -h print the same page" {
  run "${kit_root}/bin/kit" help
  [ "$status" -eq 0 ]
  local verb group
  for verb in "${kit_root}"/bin/kit-*
  do
    [[ -x "${verb}" ]] || continue
    group=$(sed -n '2s/^.* (kit verb: \([a-z]*\))$/\1/p' "${verb}")
    verb=${verb##*/kit-}
    if [[ "${group}" == plumbing ]]
    then
      [[ "$output" != *$'\n'"  ${verb} "* ]] || { echo "listed, though plumbing: ${verb}"; return 1; }
      continue
    fi
    [[ "$output" == *$'\n'"  ${verb} "* ]] || { echo "missing: ${verb}"; return 1; }
  done
  group-has WORKFLOW tidy 'removes merged worktrees and gone branches'
  group-has INSTALLATION install 'mirrors the release'
  [[ "$output" == *$'USAGE\n  kit <verb> [<argument>…]\n'* ]]
  [[ "$output" == *$'\nEXAMPLES\n'* && "$output" == *$'\nLEARN MORE\n'* ]]
  [[ "$output" == *$'\nLEARN MORE\n  • kit help <verb>… '* ]]
  [[ "$output" != *$'\nOTHER\n'* ]]
  page=$output
  run "${kit_root}/bin/kit" --help
  [ "$status" -eq 0 ]
  [ "$output" = "${page}" ]
  run "${kit_root}/bin/kit" -h
  [ "$status" -eq 0 ]
  [ "$output" = "${page}" ]
}

@test "every verb names one of the groups in its banner, and its page names the group and the verbs beside it" {
  local verb group
  for verb in "${kit_root}"/bin/kit-*
  do
    [[ -x "${verb}" ]] || continue
    group=$(sed -n '2s/^.* (kit verb: \([a-z]*\))$/\1/p' "${verb}")
    [[ " workflow files checks installation interfaces help plumbing " == *" ${group} "* && -n "${group}" ]] \
      || { echo "no known group: ${verb##*/}"; return 1; }
  done
  run "${kit_root}/bin/kit" help kinds
  [[ "$output" == *$'\nGroup\n  files: also kit context, kit generate, kit new'* ]]
  [[ "$output" != *"kit region"* ]]
}

@test "kit help sums the plumbing up in one line with its count, and kit help <group> lists a group's verbs as the overview does" {
  local verb count=0
  for verb in "${kit_root}"/bin/kit-*
  do
    [[ -x "${verb}" && "$(sed -n '2p' "${verb}")" == *'(kit verb: plumbing)' ]] && count=$(( count + 1 ))
  done
  [ "${count}" -ge 6 ]
  run "${kit_root}/bin/kit" help
  [ "$status" -eq 0 ]
  [[ "$output" == *$'\nPLUMBING\n  '"${count}"$' verbs that scripts, the hooks, and the TUI call; kit help plumbing lists them\n\nEXAMPLES\n'* ]]
  [[ "$output" != *$'\n  region '* && "$output" != *$'\n  usage '* ]]
  run "${kit_root}/bin/kit" help plumbing
  [ "$status" -eq 0 ]
  [ "${lines[0]}" = PLUMBING ]
  [ "${#lines[@]}" -eq $(( count + 1 )) ]
  [[ "$output" == *$'\n  region       prints the content the active packages deliver'* ]]
  [[ "$output" == *$'\n  root         prints the directory of the release that runs'* ]]
  [[ "$output" != *"  tidy "* ]]
  run "${kit_root}/bin/kit" help workflow
  [ "$status" -eq 0 ]
  [ "${lines[0]}" = WORKFLOW ]
  [[ "$output" == *$'\n  tidy         removes merged worktrees and gone branches'* ]]
  [[ "$output" != *"  region "* ]]
  # a verb's name wins over a group's, the start of a name still lists the verbs, and the answers follow in order
  run "${kit_root}/bin/kit" help help
  [[ "${lines[0]}" == "kit help - "* ]]
  run "${kit_root}/bin/kit" help re
  [[ "$output" == *"  rebase [<path>]"* && "$output" == *"  region [--home]"* ]]
  run "${kit_root}/bin/kit" help plumbing tidy
  [ "$status" -eq 0 ]
  [[ "$output" == *$'\n  usage        prints the usage lines'*$'\n\nkit tidy - removes merged worktrees and gone branches\n'* ]]
}

@test "kit help status holds the legend of the glyphs every stage draws" {
  run "${kit_root}/bin/kit" help status
  [ "$status" -eq 0 ]
  [[ "$output" == *$'\n  Legend\n'* ]]
  [[ "$output" == *$'\n    🔀  a pull request'* ]]
  [[ "$output" == *$'\n    🛬  ready'* ]]
  [[ "$output" == *"NO_COLOR removes the colors"* ]]
}

@test "kit usage prints the usage lines alone, of the named verbs or of every verb, and refuses a name that is no verb" {
  run "${kit_root}/bin/kit" usage context check
  [ "$status" -eq 0 ]
  [ "$output" = $'kit context <path>… [--json | --jq]\nkit context --claude\nkit check' ]
  run "${kit_root}/bin/kit" usage
  [ "$status" -eq 0 ]
  [ "${#lines[@]}" -ge "$(find "${kit_root}/bin" -name 'kit-*' -perm -u+x | wc -l)" ]
  [[ "$output" != *"Usage"* && "$output" != *"Description"* ]]
  run "${kit_root}/bin/kit" usage che
  [ "$status" -eq 2 ]
  [ "$output" = "❌ unknown verb 'che'; run: kit help" ]
}

@test "every verb prints its page on --help or -h, before any --, as kit help <verb> does, through bin/kit or called directly" {
  cd "${BATS_TEST_TMPDIR}" && git init --quiet --initial-branch=main
  local verb page
  for verb in "${kit_root}"/bin/kit-*
  do
    [[ -x "${verb}" ]] || continue
    verb=${verb##*/kit-}
    page=$("${kit_root}/bin/kit" help "${verb}")
    run "${kit_root}/bin/kit" "${verb}" --help
    [ "$status" -eq 0 ] || { echo "${verb}: exit ${status}: ${output}"; return 1; }
    [ "$output" = "${page}" ] || { echo "${verb}: ${output}"; return 1; }
  done
  run "${kit_root}/bin/kit-status" -h
  [ "$status" -eq 0 ]
  [[ "${lines[0]}" == "kit status - "* ]]
  run "${kit_root}/bin/kit" status --no-fetch --help
  [ "$status" -eq 0 ]
  [[ "${lines[0]}" == "kit status - "* ]]
  # after --, the words belong to the verb named there, so tui gets them and refuses as it does without a terminal
  run "${kit_root}/bin/kit" tui -- status --help
  [ "$status" -eq 2 ]
  [[ "$output" != "kit status - "* ]]
}

@test "an unknown verb is refused in the same words by kit, kit help, kit usage, and kit syntax" {
  local command
  for command in '' help usage syntax
  do
    # shellcheck disable=SC2086  # an empty command is bin/kit alone
    run "${kit_root}/bin/kit" ${command} nonesuch
    [ "$status" -eq 2 ] || { echo "kit ${command} nonesuch: exit ${status}"; return 1; }
    [ "$output" = "❌ unknown verb 'nonesuch'; run: kit help" ] || { echo "kit ${command} nonesuch: ${output}"; return 1; }
  done
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
  [[ "$output" == *$'Usage\n  kit context <path>… [--json | --jq]\n  kit context --claude\n'* ]]
  run "${kit_root}/bin/kit" help check
  [[ "$output" == *$'Usage\n  kit check\n\nDescription\n  kit check takes no arguments.'* ]]
}

@test "kit help <prefix> lists every verb that starts with it, and a verb's own name wins over a longer one" {
  run "${kit_root}/bin/kit" help re
  [ "$status" -eq 0 ]
  [[ "$output" == *"  rebase [<path>]"* && "$output" == *"  region [--home]"* ]]
  [[ "$output" == *"  release <version> [--lower]"* ]]
  [[ "$output" != *"  status"* ]]
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

@test "kit help <verb> rewraps the description's prose to 80 columns, counted in characters, and keeps an indented block as written" {
  local root="${BATS_TEST_TMPDIR}/kit" prose block line joined=''
  mkdir -p "${root}"
  cp -R "${kit_root}/bin" "${kit_root}/modules" "${root}/"
  # a paragraph at the header's full width, written twice, with multibyte characters a byte count would wrap early
  prose='Does the thing… and says so in a paragraph → written out to the header'"'"'s full width, one hundred and eighteen columns…'
  block='  --wide=<value>   an argument whose line is left exactly as the header writes it, however wide, since it is a block'
  [ "${#prose}" -eq 118 ]
  printf '%s\n' '#!/usr/bin/env bash' '# kit-wide - a verb written wide (kit verb: help)' '#' '# kit wide [--wide=<value>]' '#' \
    "# ${prose}" "# ${prose}" '#' "# ${block}" '#' '# A second paragraph, short.' > "${root}/bin/kit-wide"
  chmod +x "${root}/bin/kit-wide"
  run "${root}/bin/kit" help wide
  [ "$status" -eq 0 ]
  # the first line is filled to 77 characters, which a byte count would have broken before "full"
  [[ "$output" == *$'\nDescription\n  Does the thing… and says so in a paragraph → written out to the header\'s full\n'* ]]
  # every prose line fits 80 columns, and the paragraph's words are all there, in order
  while IFS= read -r line
  do
    [[ "${line}" == '  '[!' ']* ]] || continue
    [ "${#line}" -le 80 ] || { echo "over 80: ${line}"; return 1; }
    joined+="${joined:+ }${line#  }"
  done < <(sed -n '/^Description$/,/^$/p' <<<"$output")
  [ "${joined}" = "${prose} ${prose}" ]
  # the block is there exactly as written, under the page's indent, and the second paragraph after it
  [[ "$output" == *$'\n\n  '"${block}"$'\n\n  A second paragraph, short.\n'* ]]
  [ "$(grep -c '^    --wide=<value>   an argument whose line' <<<"$output")" -eq 1 ]
  # the overview's LEARN MORE bullets fit 80 columns too
  run "${kit_root}/bin/kit" help
  while IFS= read -r line
  do
    [ "${#line}" -le 80 ] || { echo "over 80: ${line}"; return 1; }
  done < <(sed -n '/^LEARN MORE$/,$p' <<<"$output")
}

@test "kit help prints a page for each verb named, in order, and refuses a name no verb starts with before printing any" {
  run "${kit_root}/bin/kit" help nonesuch
  [ "$status" -eq 2 ]
  [[ "$output" == *"unknown verb 'nonesuch'; run: kit help"* ]]
  run "${kit_root}/bin/kit" help tidy gate
  [ "$status" -eq 0 ]
  [ "$(grep -E '^kit [a-z]+ - ' <<<"$output" | cut -d ' ' -f 2 | paste -s -d ' ' -)" = "tidy gate" ]
  run "${kit_root}/bin/kit" help tidy nonesuch
  [ "$status" -eq 2 ]
  [[ "$output" != *"kit tidy -"* ]]
}

# ----------------------------------------------------------------------------------------------------------------------
