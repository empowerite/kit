#!/usr/bin/env bats
# syntax.bats - kit syntax prints a verb's grammar as JSON, parsed from its header, and judges the header's discipline
# ----------------------------------------------------------------------------------------------------------------------

load helper

setup() {
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
  jq -e '.purpose == "commits what is staged, after the kit'"'"'s checks"' <<<"$output"
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
  [ "$(jq -r .error <<<"$output")" = "'[' is not closed in: kit broken [<a>" ]
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
  [[ "$output" == *"unknown verb 'nonesuch'; run: kit help"* ]]
}

@test "a word that starts with - is no verb: kit syntax refuses it as the guard refuses an unknown flag, and says its output is JSON already" {
  run "${kit_root}/bin/kit" syntax commit --json
  [ "$status" -eq 2 ]
  [ "$output" = "❌ kit syntax: unknown option '--json' (its output is JSON already); run: kit help syntax" ]
  run "${kit_root}/bin/kit" syntax commit --jq
  [ "$status" -eq 2 ]
  [ "$output" = "❌ kit syntax: unknown option '--jq' (its output is JSON already); run: kit help syntax" ]
  run "${kit_root}/bin/kit" syntax commit --nope
  [ "$status" -eq 2 ]
  [ "$output" = "❌ kit syntax: unknown option '--nope'; run: kit help syntax" ]
  # before any verb, the guard refuses the word itself, in the same words
  run "${kit_root}/bin/kit" syntax --jq
  [ "$status" -eq 2 ]
  [ "$output" = "❌ unknown flag '--jq'; run: kit help syntax" ]
  # after a --, a dash word is an answer or a word to judge, which the usage lines name
  run "${kit_root}/bin/kit" syntax commit --match -- --json
  [ "$status" -eq 0 ]
  [ "$(jq -r '.status, .refusal' <<<"$output" | paste -sd' ' -)" = "2 unknown flag '--json'; run: kit help commit" ]
}

@test "kit syntax --check is clean for every verb" {
  run "${kit_root}/bin/kit" syntax --check
  [ "$status" -eq 0 ]
  [[ "$output" == *"every verb's usage parses, and every argument is described"* ]]
}

@test "the argument block is read by key, in either spelling, with continuation lines joined" {
  run "${kit_root}/bin/kit" syntax commit
  [ "$(jq -r '.arguments["--message"].description' <<<"$output")" = "the message; the first is the subject, the rest are paragraphs" ]
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
  [[ "$output" == *"kit broken: usage does not parse: '[' is not closed in: kit broken [<a>"* ]]
  [[ "$output" != *"kit plain: --b"* ]]
}

@test "a usage line is made of flags, options, operands, and lowercase words; any other word is refused by name with its line" {
  # three verbs whose one-line paragraphs start with kit <verb> but are not usage: a sentence with a dot, alternatives
  # inside angle brackets, and a capital
  printf '%s\n' '#!/usr/bin/env bash' '# kit-prose - a verb whose description reads as a usage line (kit verb: help)' '#' \
    '# kit prose never touches kit.yml' > "${root}/bin/kit-prose"
  printf '%s\n' '#!/usr/bin/env bash' '# kit-pick - a verb with alternatives inside angle brackets (kit verb: help)' '#' \
    '# kit pick [--format=<json|text>] [<path>]' > "${root}/bin/kit-pick"
  printf '%s\n' '#!/usr/bin/env bash' '# kit-caps - a verb with a capital in an operand (kit verb: help)' '#' \
    '# kit caps <Path>…' > "${root}/bin/kit-caps"
  chmod +x "${root}/bin/kit-prose" "${root}/bin/kit-pick" "${root}/bin/kit-caps"
  local word="is not --name, --name=<value>, <name>, or a lowercase word"
  run "${root}/bin/kit" syntax prose
  [ "$status" -eq 1 ]
  [ "$(jq -r .error <<<"$output")" = "'kit.yml' ${word} in: kit prose never touches kit.yml" ]
  jq -e '.verb == "prose" and (has("syntax") | not)' <<<"$output"
  run "${root}/bin/kit" syntax pick
  [ "$status" -eq 1 ]
  [ "$(jq -r .error <<<"$output")" = "'--format=<json|text>' ${word} in: kit pick [--format=<json|text>] [<path>]" ]
  run "${root}/bin/kit" syntax caps
  [ "$status" -eq 1 ]
  [ "$(jq -r .error <<<"$output")" = "'<Path>' ${word} in: kit caps <Path>…" ]
  run "${root}/bin/kit" syntax --check prose pick caps
  [ "$status" -eq 1 ]
  [[ "$output" == *"kit prose: usage does not parse: 'kit.yml' ${word} in: kit prose never touches kit.yml"* ]]
  [[ "$output" == *"kit pick: usage does not parse: '--format=<json|text>' ${word} in: kit pick"* ]]
  [[ "$output" == *"kit caps: usage does not parse: '<Path>' ${word} in: kit caps <Path>…"* ]]
  # and every real verb's lines are made of those words alone
  run "${kit_root}/bin/kit" syntax --check
  [ "$status" -eq 0 ]
}

@test "a group that repeats takes a word for each turn: kit help names verbs and groups in any order" {
  run "${kit_root}/bin/kit" syntax help
  [ "$status" -eq 0 ]
  jq -e '.usage == ["kit help [<verb> | <group>]…"] and (.syntax[0][0] | .kind == "group" and .optional and .repeat and (.alternatives | length == 2))' <<<"$output"
  run "${kit_root}/bin/kit" syntax help --match -- tidy gate
  [ "$status" -eq 0 ]
  [ "$(jq -c '[.status, .operands]' <<<"$output")" = '[0,["tidy","gate"]]' ]
  run "${kit_root}/bin/kit" syntax help --match -- plumbing --bogus
  [ "$(jq -r '.status, .refusal' <<<"$output" | paste -sd' ' -)" = "2 unknown flag '--bogus'; run: kit help help" ]
}

@test "kit syntax packages --choices lists every package with its state, closed, and a planned set says what it keeps" {
  checkout-copy
  run "${kit_root}/bin/kit" syntax packages --choices '<package>'
  [ "$status" -eq 0 ]
  grep -q $'^agents/claude\t.*\tactive\t$' <<<"$output"
  grep -q $'^agents/copilot\t.*\t\t$' <<<"$output"
  run "${kit_root}/bin/kit" syntax packages
  [ "$(jq -r '.arguments["<package>"].choices' <<<"$output")" = closed ]
  [ "$(jq -r '.arguments["list"].choices' <<<"$output")" = null ]
  run "${kit_root}/bin/kit" syntax packages --choices '<package>' '<package>=empowerite' '<package>=agents/claude'
  [ "$status" -eq 0 ]
  grep -q $'^hooks/lefthook\t.*\tactive\thooks/lefthook: kept, a member of empowerite$' <<<"$output"
  grep -q $'^kit\t.*\tactive\tkit: kept, .* depend(s) on it$' <<<"$output"
  grep -q $'^agents/copilot\t.*\t\t$' <<<"$output"
}

@test "kit syntax new --choices --variant lists the rule's variants, and --rule the rules with templates; both closed" {
  checkout-copy
  run "${kit_root}/bin/kit" syntax new --choices --variant -- --rule=markdown
  [ "$status" -eq 0 ]
  [ "$(cut -f1 <<<"$output" | paste -sd' ' -)" = "document notice frontmatter" ]
  run "${kit_root}/bin/kit" syntax new --choices --rule
  [ "$status" -eq 0 ]
  grep -q $'^bash\t' <<<"$output"
  grep -q $'^markdown\t' <<<"$output"
  run "${kit_root}/bin/kit" syntax new
  [ "$(jq -r '[.arguments["--rule"].choices, .arguments["--variant"].choices, (.arguments["<path>"].choices // "none")] | join(" ")' <<<"$output")" = "closed closed none" ]
}

@test "open choices suggest and never refuse: commit's paths are the changed files with their state, help's verbs carry their purpose" {
  cd "${BATS_TEST_TMPDIR}" && git init --quiet --initial-branch=main
  printf 'x\n' > new.txt
  printf 'y\n' > staged.txt
  git add staged.txt
  run "${kit_root}/bin/kit" syntax commit --choices '<path>'
  [ "$status" -eq 0 ]
  grep -q $'^new.txt\tuntracked\t\t$' <<<"$output"
  grep -q $'^staged.txt\tadded\tactive\t$' <<<"$output"
  run "${kit_root}/bin/kit" syntax commit
  [ "$(jq -r '.arguments["<path>"].choices' <<<"$output")" = open ]
  run "${kit_root}/bin/kit" syntax help --choices '<verb>'
  grep -q $'^tidy\tremoves merged worktrees and gone branches\t\t$' <<<"$output"
  run "${kit_root}/bin/kit" syntax help --choices '--bogus'
  [ "$status" -eq 1 ]
  [[ "$output" == *"kit help has no choices for '--bogus'; run: kit help syntax"* ]]
}

@test "kit syntax new --extend lists the template's holes as options once the rule and variant are known, and new's syntax says it extends" {
  checkout-copy
  run "${kit_root}/bin/kit" syntax new
  jq -e '.extends == true' <<<"$output"
  run "${kit_root}/bin/kit" syntax commit
  jq -e '.extends | not' <<<"$output"
  run "${kit_root}/bin/kit" syntax new --extend '<path>=docs/x.md' -- --rule=markdown --variant=document
  [ "$status" -eq 0 ]
  jq -e '.["--title"].extended == true and (has("--body") | not) and (has("--path") | not)' <<<"$output"
  run "${kit_root}/bin/kit" syntax new --extend
  [ "$status" -eq 0 ]
  [ "$output" = '{}' ]
  run "${kit_root}/bin/kit" syntax commit --extend
  [ "$status" -eq 1 ]
  [[ "$output" == *"kit commit does not extend its arguments; run: kit help syntax"* ]]
}

@test "only kit new extends its arguments: the verb syntax does not answer with the API's own functions" {
  run "${kit_root}/bin/kit" syntax
  [ "$(jq -r '[.[] | select(.extends) | .verb] | join(" ")' <<<"$output")" = "new" ]
  [ "$(jq -r '[.[] | .arguments[] | select(.choices) | .choices] | unique | join(" ")' <<<"$output")" = "closed open" ]
  run "${kit_root}/bin/kit" syntax syntax --extend
  [ "$status" -eq 1 ]
  [[ "$output" == *"kit syntax does not extend its arguments; run: kit help syntax"* ]]
  run "${kit_root}/bin/kit" syntax syntax --choices '<verb>'
  [ "$status" -eq 1 ]
  [[ "$output" == *"kit syntax has no choices for '<verb>'; run: kit help syntax"* ]]
}

@test "the Go fixtures under tui/ are what kit syntax prints today" {
  local fixture verb
  for fixture in "${kit_root}"/../tui/internal/syntax/testdata/*.json
  do
    verb=$(basename "${fixture}" .json)
    diff <("${kit_root}/bin/kit" syntax "${verb}") "${fixture}" >/dev/null \
      || { echo "stale: ${verb}; rerun: kit syntax ${verb} > tui/internal/syntax/testdata/${verb}.json"; return 1; }
  done
}

@test "kit syntax <verb> --match judges a command line and prints what it sorted, as the form reads it" {
  run "${kit_root}/bin/kit" syntax commit --match -- --message hi x.txt
  [ "$status" -eq 0 ]
  [ "$(jq -c '[.status, .operands, .options["--message"], .values["<path>"]]' <<<"$output")" = '[0,["x.txt"],["hi"],["x.txt"]]' ]
  run "${kit_root}/bin/kit" syntax commit --match -- --all
  [ "$status" -eq 0 ]
  [ "$(jq -r '.status, .missing' <<<"$output" | paste -sd' ' -)" = "1 --message or --file" ]
  run "${kit_root}/bin/kit" syntax commit --match -- --bogus
  [ "$status" -eq 0 ]
  [ "$(jq -r '.status, .refusal' <<<"$output" | paste -sd' ' -)" = "2 unknown flag '--bogus'; run: kit help commit" ]
  # a word of the group the acting line opens with opens that line, over the listing line's operand
  run "${kit_root}/bin/kit" syntax packages --match -- add
  [ "$(jq -c '[.status, .literals, .missing]' <<<"$output")" = '[1,["add"],"<package>"]' ]
  run "${kit_root}/bin/kit" syntax packages --match -- add style/bash
  [ "$(jq -c '[.status, .literals, .operands]' <<<"$output")" = '[0,["add"],["style/bash"]]' ]
  run "${kit_root}/bin/kit" syntax packages --match -- style/bash
  [ "$(jq -c '[.status, .literals, .operands]' <<<"$output")" = '[0,[],["style/bash"]]' ]
}

@test "kit syntax --list prints each verb, its purpose, and its group, one per line, for a palette to open on" {
  run "${kit_root}/bin/kit" syntax --list
  [ "$status" -eq 0 ]
  [ "${#lines[@]}" -eq "$(compgen -G "${kit_root}/bin/kit-*" | wc -l | tr -d ' ')" ]
  grep -q $'^tidy\tremoves merged worktrees and gone branches\tworkflow$' <<<"$output"
  # every line has the three columns, and the group is one word
  [ "$(grep -c $'^[a-z-]*\t[^\t]*\t[a-z]*$' <<<"$output")" -eq "${#lines[@]}" ]
  run "${kit_root}/bin/kit" syntax --list --check
  [ "$status" -eq 2 ]
  [[ "$output" == *"--check cannot be combined with --list; run: kit help syntax"* ]]
}

@test "a list of choices says it is a set only when the answers name the declarative subcommand" {
  run "${kit_root}/bin/kit" syntax packages --choices '<package>' -- --kind select
  [ "$output" = set ]
  run "${kit_root}/bin/kit" syntax packages --choices '<package>' -- --kind add
  [ "$output" = closed ]
  run "${kit_root}/bin/kit" syntax packages --choices '<package>' -- --kind
  [ "$output" = closed ]
  run "${kit_root}/bin/kit" packages add nonesuch
  [[ "$output" == *"<package> 'nonesuch' is not one of the choices"* ]]
}

@test "the tracked files a path is picked from are relative to where the command was typed" {
  fresh-repo >/dev/null
  mkdir docs && printf '# Features\n' > docs/features.md && git add docs/features.md
  cd docs
  run "${kit_root}/bin/kit" syntax lint --choices '<path>'
  [ "$status" -eq 0 ]
  grep -q $'^features.md\t' <<<"$output"
  ! grep -q $'^docs/features.md\t' <<<"$output"
}

@test "a verb whose usage line does not parse refuses with the check that names why" {
  printf '%s\n' '#!/usr/bin/env bash' '# kit-askew - a verb whose usage line does not parse (kit verb: help)' '#' '# kit askew [<a>' \
    '#' '# Does it.' '' 'source "$(dirname "$0")/../modules/bash/1st.sh"' 'guard:arguments askew "$@" || exit $?' \
    > "${root}/bin/kit-askew"
  chmod +x "${root}/bin/kit-askew"
  run "${root}/bin/kit" askew x
  [ "$status" -eq 2 ]
  [[ "$output" == *"kit askew: usage does not parse: '[' is not closed in: kit askew [<a>; run: kit syntax --check askew"* ]]
}

@test "--choices tells a verb with no choices for an argument from one that cannot list them for these answers" {
  run "${kit_root}/bin/kit" syntax syntax --choices '<verb>'
  [ "$status" -eq 1 ]
  [[ "$output" == *"kit syntax has no choices for '<verb>'; run: kit help syntax"* ]]
  run "${kit_root}/bin/kit" syntax new --choices --variant -- --rule=nonesuch
  [ "$status" -eq 1 ]
  [[ "$output" == *"kit new lists no --variant for these answers: --rule=nonesuch; run: kit help new"* ]]
}

# ----------------------------------------------------------------------------------------------------------------------
