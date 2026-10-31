#!/usr/bin/env bats
# completion.bats - kit completion prints the zsh script, and the candidates for each word, read from the verbs
# ----------------------------------------------------------------------------------------------------------------------

bats_require_minimum_version 1.5.0

setup() {
  export kit_checkout NO_COLOR=1
  kit_checkout=$(cd "${BATS_TEST_DIRNAME}/../kit" && pwd)
  export KIT_HOME="${BATS_TEST_TMPDIR}/home" XDG_CACHE_HOME="${BATS_TEST_TMPDIR}/cache"
  mkdir -p "${KIT_HOME}" "${XDG_CACHE_HOME}"
  repo="${BATS_TEST_TMPDIR}/repo"
  git init --quiet --initial-branch=main "${repo}"
  cd "${repo}"
  "${kit_checkout}/bin/kit" install >/dev/null 2>&1
}

# candidates <word>… prints the candidates' words alone, for the words after kit.
candidates() {
  "${kit_checkout}/bin/kit" completion candidates kit "$@" | cut -f 1
}

@test "the first word is a verb, each with its purpose, and every verb is offered" {
  run "${kit_checkout}/bin/kit" completion candidates kit ''
  [ "$status" -eq 0 ]
  [[ "$output" == *$'commit\tcommits what is staged, after the checks a commit\'s rule requires'* ]]
  [ "$(cut -f 1 <<<"$output" | wc -l)" -eq "$(find "${kit_checkout}/bin" -name 'kit-*' -perm -u+x | wc -l)" ]
}

@test "a dash offers the flags and options not given yet, and an inline option ends in =" {
  [ "$(candidates commit -)" = $'--all\n--file\n--message' ]
  [ "$(candidates commit --all --)" = $'--file\n--message' ]
  [ "$(candidates commit --message x --message y -)" = $'--all\n--file\n--message' ]
  [[ "$(candidates new file x -)" == *"--rule="* ]]
}

@test "the literal words come first, then the values a verb lists for its operand" {
  [ "$(candidates package '')" = $'add\nremove\nselect' ]
  run candidates package add ''
  [[ "$output" == *"agents/claude"* && "$output" == *"style/bash"* ]]
  [ "$(candidates help '' | head -n 1)" = backlog ]
}

@test "an option's value comes from its choices, after --name= or apart, and a path asks for the files" {
  "${kit_checkout}/bin/kit" package add style/bash style/markdown >/dev/null 2>&1
  run candidates new file x --rule=
  [[ "$output" == *"--rule=bash"* && "$output" == *"--rule=markdown"* ]]
  [ "$(candidates commit --file '')" = ':files' ]
  [ "$(candidates new file '')" = ':files' ]
  [ -z "$(candidates nosuchverb '')" ]
}

@test "the zsh script parses, registers for kit and for every kit-<verb>, and knows no verb by name" {
  command -v zsh >/dev/null || skip "zsh is not on PATH"
  run "${kit_checkout}/bin/kit" completion zsh
  [ "$status" -eq 0 ]
  [[ "$output" == "#compdef kit -p kit-*"* ]]
  [[ "$output" != *"commit"* && "$output" != *"package"* ]]
  zsh -n <<<"$output"
  run zsh -fc "autoload -Uz compinit && compinit -u -d '${BATS_TEST_TMPDIR}/zcompdump'
    eval \"\$('${kit_checkout}/bin/kit' completion zsh)\"; print -r -- \"\${_comps[kit]} \${_patcomps[kit-*]}\""
  [ "$output" = "_kit _kit" ]
}

# ----------------------------------------------------------------------------------------------------------------------
