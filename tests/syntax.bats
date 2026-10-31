#!/usr/bin/env bats
# syntax.bats - kit syntax prints a verb's grammar as JSON, parsed from its header, and judges the header's discipline
# ----------------------------------------------------------------------------------------------------------------------

setup() {
  export kit_root NO_COLOR=1
  kit_root=$(cd "${BATS_TEST_DIRNAME}/../kit" && pwd)
  # a copy of the kit with two verbs whose headers are written for the tests, as help.bats makes one: plain has two
  # usage lines, three dots, a wildcard, and a block that describes --b in the other spelling and <a> over two lines
  # but says nothing of --<hole>; broken has a usage line that does not parse
  root="${BATS_TEST_TMPDIR}/kit"
  mkdir -p "${root}"
  cp -R "${kit_root}/bin" "${kit_root}/modules" "${root}/"
  printf '%s\n' '#!/usr/bin/env bash' '# kit-plain - a verb written for the tests (kit verb)' '#' \
    '# kit plain <a>... [--<hole>=<value>]' '#' '# kit plain --b <v>' '#' '# Does it.' '#' \
    '#   <a>…      first line' '#       second line' '#   --b=<v>   bee' > "${root}/bin/kit-plain"
  printf '%s\n' '#!/usr/bin/env bash' '# kit-broken - a verb whose usage line does not parse (kit verb)' '#' \
    '# kit broken [<a>' > "${root}/bin/kit-broken"
  chmod +x "${root}/bin/kit-plain" "${root}/bin/kit-broken"
}

@test "every verb's usage line parses, and kit syntax with no verb prints one object per verb, in order" {
  run "${kit_root}/bin/kit" syntax
  [ "$status" -eq 0 ]
  local verb expected=''
  for verb in "${kit_root}"/bin/kit-*
  do
    [[ -x "${verb}" ]] && expected+="${verb##*/kit-} "
  done
  [ "$(jq -r 'map(.verb) | join(" ")' <<<"$output")" = "${expected% }" ]
  [ "$(jq 'map(select(.error)) | length' <<<"$output")" -eq 0 ]
}

@test "kit commit's syntax is one alternative of three nodes: an optional flag, a required choice, an optional repeated operand" {
  run "${kit_root}/bin/kit" syntax commit
  [ "$status" -eq 0 ]
  jq -e '.verb == "commit" and (.usage | length == 1) and (.syntax | length == 1) and (.syntax[0] | length == 3)' <<<"$output"
  jq -e '.purpose == "commits what is staged, after the checks a commit'"'"'s rule requires"' <<<"$output"
  jq -e '.syntax[0][0] | .kind == "group" and .optional and (.alternatives[0][0] | .kind == "flag" and .name == "all")' <<<"$output"
  jq -e '.syntax[0][1] | .kind == "group" and (.optional | not) and (.alternatives | length == 2)' <<<"$output"
  jq -e '.syntax[0][1].alternatives[0][0] | .kind == "option" and .name == "message" and .value == "text" and .repeat and (.inline | not)' <<<"$output"
  jq -e '.syntax[0][1].alternatives[1][0] | .kind == "option" and .name == "file" and .value == "path" and (.repeat | not)' <<<"$output"
  jq -e '.syntax[0][2].alternatives[0][0] | .kind == "operand" and .name == "path" and .repeat' <<<"$output"
}

@test "two usage lines are two alternatives, three dots are an ellipsis, a wildcard option is marked, and an unclosed bracket is an error" {
  run "${root}/bin/kit" syntax plain
  [ "$status" -eq 0 ]
  jq -e '.usage == ["kit plain <a>... [--<hole>=<value>]", "kit plain --b <v>"]' <<<"$output"
  jq -e '.syntax | length == 2' <<<"$output"
  jq -e '.syntax[0][0] | .kind == "operand" and .name == "a" and .repeat' <<<"$output"
  jq -e '.syntax[0][1] | .kind == "group" and .optional and (.alternatives[0][0] | .kind == "option" and .wildcard and .name == "hole" and .value == "value" and .inline)' <<<"$output"
  jq -e '.syntax[1][0] | .kind == "option" and .name == "b" and .value == "v" and (.inline | not)' <<<"$output"
  run "${root}/bin/kit" syntax broken
  [ "$status" -eq 1 ]
  [ "$(jq -r .error <<<"$output")" = "'[' is not closed" ]
  run "${root}/bin/kit" syntax
  [ "$status" -eq 1 ]
  [ "$(jq -r 'map(select(.error)) | .[0].verb' <<<"$output")" = broken ]
}

@test "a verb with no usage line has one empty alternative, and kit syntax refuses a name that is not a verb" {
  run "${kit_root}/bin/kit" syntax version
  [ "$status" -eq 0 ]
  jq -e '.syntax == [[]] and .usage == ["kit version"]' <<<"$output"
  run "${kit_root}/bin/kit" syntax nonesuch
  [ "$status" -eq 2 ]
  [[ "$output" == *"no kit verb 'nonesuch'; run: kit help"* ]]
}

@test "kit syntax --check is clean for every verb" {
  run "${kit_root}/bin/kit" syntax --check
  [ "$status" -eq 0 ]
  [[ "$output" == *"every verb's usage parses, and every argument is described"* ]]
}

@test "the argument block is read by key, in either spelling, with continuation lines joined" {
  run "${kit_root}/bin/kit" syntax commit
  [ "$(jq -r '.arguments["--message"].description' <<<"$output")" = "the message; given twice or more, the first is the subject and the rest are paragraphs" ]
  jq -e '.arguments | has("--all") and has("--file") and has("<path>")' <<<"$output"
  run "${root}/bin/kit" syntax plain
  [ "$(jq -r '.arguments["--b"].description' <<<"$output")" = "bee" ]
  [ "$(jq -r '.arguments["<a>"].description' <<<"$output")" = "first line second line" ]
  jq -e '.arguments | has("--<hole>") | not' <<<"$output"
}

@test "--check names each undescribed argument and each unparsable line, and fails" {
  run "${root}/bin/kit" syntax --check plain broken
  [ "$status" -eq 1 ]
  [[ "$output" == *"kit plain: --<hole> has no description in the header"* ]]
  [[ "$output" == *"kit broken: usage does not parse: '[' is not closed"* ]]
  [[ "$output" != *"kit plain: --b"* ]]
}

# ----------------------------------------------------------------------------------------------------------------------
