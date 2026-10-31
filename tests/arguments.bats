#!/usr/bin/env bats
# arguments.bats - guard:arguments judges every verb's arguments from its syntax, and refuses in one generic place
# ----------------------------------------------------------------------------------------------------------------------

setup() {
  export kit_root NO_COLOR=1
  kit_root=$(cd "${BATS_TEST_DIRNAME}/../kit" && pwd)
  repo=$(mktemp -d)
  cd "${repo}"
  git init --quiet --initial-branch=main
}

teardown() {
  rm -rf "${repo}"
}

# kit-bash <code> runs <code> in a fresh bash that has sourced the loader and the syntax module, as a verb does.
kit-bash() {
  bash -c 'source "${kit_root}/modules/bash/1st.sh" && use syntax && eval "$1"' kit-bash "$1"
}

@test "an unknown flag, an unknown argument, and an argument to a verb that takes none are each refused in one line, exit 2" {
  run "${kit_root}/bin/kit" status --bogus
  [ "$status" -eq 2 ]
  [[ "$output" == *"unknown flag '--bogus'; run: kit help status"* ]]
  run "${kit_root}/bin/kit" status foo
  [ "$status" -eq 2 ]
  [[ "$output" == *"unknown argument 'foo'; run: kit help status"* ]]
  run "${kit_root}/bin/kit" version --json
  [ "$status" -eq 2 ]
  [[ "$output" == *"kit version takes no arguments; run: kit help version"* ]]
  run "${kit_root}/bin/kit" package nonesuch
  [ "$status" -eq 2 ]
  [[ "$output" == *"unknown argument 'nonesuch'; run: kit help package"* ]]
  run "${kit_root}/bin/kit" toolchain nope
  [ "$status" -eq 2 ]
  [[ "$output" == *"unknown argument 'nope'; run: kit help toolchain"* ]]
  run "${kit_root}/bin/kit" hydrate --quick=1
  [ "$status" -eq 2 ]
  [[ "$output" == *"unknown flag '--quick=1'; run: kit help hydrate"* ]]
}

@test "a missing required group is refused naming its alternatives, and two alternatives together are refused" {
  run "${kit_root}/bin/kit" commit
  [ "$status" -eq 2 ]
  [[ "$output" == *"kit commit needs --message or --file; run: kit help commit"* ]]
  run "${kit_root}/bin/kit" new
  [ "$status" -eq 2 ]
  [[ "$output" == *"kit new needs file; run: kit help new"* ]]
  run "${kit_root}/bin/kit" new file
  [ "$status" -eq 2 ]
  [[ "$output" == *"kit new file needs <path>; run: kit help new"* ]]
  run "${kit_root}/bin/kit" package add
  [ "$status" -eq 2 ]
  [[ "$output" == *"kit package add needs <package>; run: kit help package"* ]]
  run "${kit_root}/bin/kit" context
  [ "$status" -eq 2 ]
  [[ "$output" == *"kit context needs <path> or --claude; run: kit help context"* ]]
  run "${kit_root}/bin/kit" commit --message m --file f
  [ "$status" -eq 2 ]
  [[ "$output" == *"--message cannot be combined with --file; run: kit help commit"* ]]
  run "${kit_root}/bin/kit" commit --message
  [ "$status" -eq 2 ]
  [[ "$output" == *"--message needs a value; run: kit help commit"* ]]
  run "${kit_root}/bin/kit" new file x --rule=
  [ "$status" -eq 2 ]
  [[ "$output" == *"--rule needs a value; run: kit help new"* ]]
}

@test "syntax:match sorts what is given: a trailing operand takes the dash-words, -- ends the flags, a value with spaces is whole" {
  run kit-bash 'syntax:match toolchain run bash -c "echo hi"; echo "$? ${syntax_literals[*]} | ${#syntax_operands[@]} | ${syntax_operands[*]}"'
  [ "$output" = "0 run | 3 | bash -c echo hi" ]
  run kit-bash 'syntax:match commit --message "a b" -- --weird; echo "$? ${syntax_options[--message]} | ${syntax_operands[*]}"'
  [ "$output" = "0 a b | --weird" ]
  run kit-bash 'syntax:match status --owner=ada --owner=bob; echo "$? ${syntax_options[--owner]}"'
  [ "$output" = $'0 ada\nbob' ]
  run kit-bash 'syntax:match context a.md --json; echo "$? ${syntax_operands[*]} ${syntax_options[--json]}"'
  [ "$output" = "0 a.md 1" ]
  run kit-bash 'syntax:match commit --all; echo "$? ${syntax_missing} | ${syntax_options[--all]}"'
  [ "$output" = "1 --message or --file | 1" ]
  run kit-bash 'syntax:match status --bogus; echo "$? ${syntax_refusal}"'
  [ "$output" = "2 unknown flag '--bogus'; run: kit help status" ]
  run kit-bash 'syntax:match status; echo "$?"'
  [ "$output" = 0 ]
}

@test "complete arguments return 0 and the verb runs as before" {
  run "${kit_root}/bin/kit" help tidy
  [ "$status" -eq 0 ]
  [ "${lines[0]}" = "kit tidy - removes merged worktrees and gone branches" ]
  run "${kit_root}/bin/kit" version
  [ "$status" -eq 0 ]
  run "${kit_root}/bin/kit" syntax version
  [ "$status" -eq 0 ]
}

@test "no verb calls guard:no-arguments or guard:no-flags, and none keeps an unknown-flag branch of its own" {
  [ "$(grep -l 'guard:no-arguments\|guard:no-flags' "${kit_root}"/bin/kit-* "${kit_root}"/modules/bash/*.sh | wc -l)" -eq 0 ]
  [ "$(grep -l "unknown flag '" "${kit_root}"/bin/kit-* | wc -l)" -eq 0 ]
  [ "$(grep -L 'guard:arguments' "${kit_root}"/bin/kit-* | wc -l)" -eq 0 ]
}

@test "a value off a closed set is refused in one line by the verb itself, a glob that matches is not, and an open set never refuses" {
  run "${kit_root}/bin/kit" package add nonesuch
  [ "$status" -eq 2 ]
  [[ "$output" == *"<package> 'nonesuch' is not one of the choices; run: kit help package"* ]]
  run kit-bash 'syntax:match package select "style/*" kit; echo "$?"'
  [ "$output" = 0 ]
  run kit-bash 'syntax:match package select "nonesuch/*"; echo "$? ${syntax_refusal}"'
  [ "$output" = "2 <package> 'nonesuch/*' is not one of the choices; run: kit help package" ]
  run kit-bash 'cd "${kit_root}/.." && syntax:match new file x --rule=markdown --variant=document; echo "$?"'
  [ "$output" = 0 ]
  run kit-bash 'cd "${kit_root}/.." && syntax:match new file x --rule=markdown --variant=nonesuch; echo "$? ${syntax_refusal}"'
  [ "$output" = "2 --variant 'nonesuch' is not one of the choices; run: kit help new" ]
  run kit-bash 'syntax:match context nonesuch.md; echo "$?"'
  [ "$output" = 0 ]
}

# ----------------------------------------------------------------------------------------------------------------------
