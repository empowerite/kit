#!/usr/bin/env bats
# arguments.bats - guard:arguments judges every verb's arguments from its syntax, and refuses in one generic place
# ----------------------------------------------------------------------------------------------------------------------

load helper

setup() {
  kit_bash_modules=syntax
  fresh-repo
}

@test "an unknown flag, an unknown argument, and an argument to a verb that takes none are each refused in one line, exit 2" {
  run "${kit_root}/bin/kit" flow --bogus
  [ "$status" -eq 2 ]
  [[ "$output" == *"unknown flag '--bogus'; run: kit help flow"* ]]
  run "${kit_root}/bin/kit" flow foo
  [ "$status" -eq 2 ]
  [[ "$output" == *"unknown argument 'foo'; run: kit help flow"* ]]
  run "${kit_root}/bin/kit" version --json
  [ "$status" -eq 2 ]
  [[ "$output" == *"kit version takes no arguments; run: kit help version"* ]]
  run "${kit_root}/bin/kit" new nonesuch
  [ "$status" -eq 2 ]
  [[ "$output" == *"unknown argument 'nonesuch'; run: kit help new"* ]]
  run "${kit_root}/bin/kit" toolchains nope
  [ "$status" -eq 2 ]
  [[ "$output" == *"unknown argument 'nope'; run: kit help toolchains"* ]]
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
  [[ "$output" == *"kit new needs file or kind or rule or style or package or template or module or provider or verb or issue; run: kit help new"* ]]
  run "${kit_root}/bin/kit" new file
  [ "$status" -eq 2 ]
  [[ "$output" == *"kit new file needs <path>; run: kit help new"* ]]
  run "${kit_root}/bin/kit" packages add
  [ "$status" -eq 2 ]
  [[ "$output" == *"kit packages add needs <package>; run: kit help packages"* ]]
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
  run kit-bash 'syntax:match tui run bash -c "echo hi"; echo "$? ${syntax_literals[*]} | ${#syntax_operands[@]} | ${syntax_operands[*]}"'
  [ "$output" = "0  | 4 | run bash -c echo hi" ]
  run kit-bash 'syntax:match commit --message "a b" -- --weird; echo "$? ${syntax_options[--message]} | ${syntax_operands[*]}"'
  [ "$output" = "0 a b | --weird" ]
  run kit-bash 'syntax:match flow --owner=ada --owner=bob; echo "$? ${syntax_options[--owner]}"'
  [ "$output" = $'0 ada\nbob' ]
  run kit-bash 'syntax:match context a.md --json; echo "$? ${syntax_operands[*]} ${syntax_options[--json]}"'
  [ "$output" = "0 a.md 1" ]
  run kit-bash 'syntax:match commit --all; echo "$? ${syntax_missing} | ${syntax_options[--all]}"'
  [ "$output" = "1 --message or --file | 1" ]
  run kit-bash 'syntax:match flow --bogus; echo "$? ${syntax_refusal}"'
  [ "$output" = "2 unknown flag '--bogus'; run: kit help flow" ]
  run kit-bash 'syntax:match flow; echo "$?"'
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
  run "${kit_root}/bin/kit" packages add nonesuch
  [ "$status" -eq 2 ]
  [[ "$output" == *"<package> 'nonesuch' is not one of the choices; run: kit help packages"* ]]
  run kit-bash 'syntax:match packages select "style/*" kit; echo "$?"'
  [ "$output" = 0 ]
  run kit-bash 'syntax:match packages select "nonesuch/*"; echo "$? ${syntax_refusal}"'
  [ "$output" = "2 <package> 'nonesuch/*' is not one of the choices; run: kit help packages" ]
  # a rule's variants are its templates in this repository
  "${kit_root}/bin/kit" install style/markdown >/dev/null 2>&1
  run kit-bash 'syntax:match new file x --rule=markdown --variant=document; echo "$?"'
  [ "$output" = 0 ]
  run kit-bash 'syntax:match new file x --rule=markdown --variant=nonesuch; echo "$? ${syntax_refusal}"'
  [ "$output" = "2 --variant 'nonesuch' is not one of the choices; run: kit help new" ]
  run kit-bash 'syntax:match context nonesuch.md; echo "$?"'
  [ "$output" = 0 ]
}

@test "a verb's own flag after a trailing operand's first word belongs to the command, and an option meant once is refused twice" {
  run kit-bash 'syntax:match tui flow kit generate --check; echo "$? ${syntax_operands[*]} | ${syntax_options[--check]:-none}"'
  [ "$output" = "0 flow kit generate --check | none" ]
  run kit-bash 'syntax:match flow issues --no-fetch; echo "$? ${syntax_refusal}"'
  [ "$output" = "2 --no-fetch cannot be combined with issues; run: kit help flow" ]
  run kit-bash 'syntax:match packages add --json x; echo "$? ${syntax_refusal}"'
  [ "$output" = "2 --json cannot be combined with add; run: kit help packages" ]
  run kit-bash 'syntax:match commit a.txt --all; echo "$? ${syntax_operands[*]} | ${syntax_options[--all]}"'
  [ "$output" = "1  | 1" ]
  run kit-bash 'syntax:match commit --message m a.txt --all; echo "$? ${syntax_operands[*]} | ${syntax_options[--all]}"'
  [ "$output" = "0 a.txt | 1" ]
  run kit-bash 'syntax:match work --title a --title b; echo "$? ${syntax_refusal}"'
  [ "$output" = "2 --title is given twice; run: kit help work" ]
  run kit-bash 'syntax:match commit --message a --message b; echo "$? ${syntax_options[--message]}"'
  [ "$output" = $'0 a\nb' ]
}

@test "a wildcard option may be empty, -- never reaches a verb as a path, and a missing jq is refused with the system toolchain's remedy" {
  run kit-bash 'syntax:match new file x --purpose=; echo "$? [${syntax_options[--purpose]}]"'
  [ "$output" = "0 []" ]
  run kit-bash 'syntax:match new file x --rule=; echo "$? ${syntax_refusal}"'
  [ "$output" = "2 --rule needs a value; run: kit help new" ]
  "${kit_root}/bin/kit" install >/dev/null 2>&1
  printf '# Mine\n' > README.md && git add README.md
  run "${kit_root}/bin/kit" context README.md
  local plain=$output
  run "${kit_root}/bin/kit" context -- README.md
  [ "$status" -eq 0 ]
  [ "$output" = "${plain}" ]
  # a PATH with every tool but jq, and this bash rather than the system's 3.2
  mkdir -p "${BATS_TEST_TMPDIR}/nojq"
  for tool in /usr/bin/* /bin/*
  do
    [[ "${tool##*/}" != jq ]] && ln -sf "${tool}" "${BATS_TEST_TMPDIR}/nojq/"
  done
  ln -sf "$(command -v bash)" "${BATS_TEST_TMPDIR}/nojq/bash"
  run env PATH="${BATS_TEST_TMPDIR}/nojq" "${kit_root}/bin/kit" version
  [ "$status" -eq 2 ]
  [[ "$output" == *" jq "*"(jq is not on PATH)"* && "$output" != *"kit tools install"* ]]
  [[ "$output" != *"command not found"* ]]
}

@test "a missing tool kit runs on is the system toolchain's to install, any other kit tools install's, and none is assumed" {
  hide-system-toolchains
  fake-tools brew
  run kit-bash 'guard:remedy jq; guard:remedy yq; guard:remedy go; guard:missing git "git is not on PATH"'
  [ "$output" = "run: brew install jq
run: brew install yq
run: kit tools install
❌ run: brew install git (git is not on PATH)" ]
  # beside apt, brew is one of two, and with neither chosen, the remedy names neither, and asks nothing
  fake-tools apt-get
  run kit-bash 'guard:remedy jq' </dev/null
  [ "$output" = "install jq with this machine's package manager" ]
  # the system's choice names its command
  local sudo='sudo '
  (( EUID != 0 )) || sudo=''
  kit-bash 'use toolchain && toolchain:choose system apt'
  run kit-bash 'guard:remedy jq'
  [ "$output" = "run: ${sudo}apt-get install jq" ]
}

@test "every verb with --json takes --jq too, described, and the two are one or the other" {
  run "${kit_root}/bin/kit" syntax
  [ "$(jq -r '[.[] | select(.arguments | has("--json")) | select(.arguments["--jq"].description | not) | .verb] | join(" ")' <<<"$output")" = "" ]
  [ "$(jq '[.[] | select(.arguments | has("--json"))] | length' <<<"$output")" -gt 0 ]
  run "${kit_root}/bin/kit" context README.md --json --jq
  [ "$status" -eq 2 ]
  [[ "$output" == *"--json cannot be combined with --jq; run: kit help context"* ]]
}

@test "--jq prints what --json prints, formatted by jq, from where the command was typed, with the verb's status" {
  "${kit_root}/bin/kit" install >/dev/null 2>&1
  mkdir docs && cp "${kit_root}/../docs/features.md" docs/ && git add docs/features.md
  cd docs
  local json
  json=$("${kit_root}/bin/kit" context features.md --json | jq .)
  run "${kit_root}/bin/kit" context features.md --jq
  [ "$status" -eq 0 ]
  [ "$output" = "${json}" ]
  [ "$(wc -l <<<"$output")" -gt 1 ]
}

# ----------------------------------------------------------------------------------------------------------------------
