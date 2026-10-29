#!/usr/bin/env bats
# context.bats - kit context names a file's rules by path or by first line, logs the ask, and answers Claude's hook
# ----------------------------------------------------------------------------------------------------------------------

setup() {
  kit_checkout=$(cd "${BATS_TEST_DIRNAME}/../kit" && pwd)
  export KIT_HOME="${BATS_TEST_TMPDIR}/home" XDG_CACHE_HOME="${BATS_TEST_TMPDIR}/cache"
  mkdir -p "${KIT_HOME}" "${XDG_CACHE_HOME}"
  repo=$(mktemp -d)
  git -C "${repo}" init --quiet --initial-branch=main
  cd "${repo}"
  "${kit_checkout}/bin/kit" install >/dev/null
}

teardown() {
  rm -rf "${repo}"
}

@test "a file is governed by its path, by its first line, or by nothing" {
  # a real file the rule's glob would expand to, were the glob ever word-split
  mkdir docs && printf '# x\n' > docs/x.md
  run "${kit_checkout}/bin/kit" context README.md
  [ "$output" = ".kit/rules/markdown.md" ]
  run "${kit_checkout}/bin/kit" context justfile
  [ "$status" -eq 0 ]
  [ "$output" = ".kit/rules/just.md" ]
  mkdir bin && printf '#!/usr/bin/env bash\necho\n' > bin/tool
  run "${kit_checkout}/bin/kit" context bin/tool
  [ "$output" = ".kit/rules/bash.md" ]
  run "${kit_checkout}/bin/kit" context bin/other
  [ "$status" -eq 0 ]
  [ -z "$output" ]
  run "${kit_checkout}/bin/kit" context docs/a.md "${PWD}/.github/workflows/x.yml" lib/x.md
  [ "${lines[0]}" = ".kit/rules/markdown.md" ]
  [ "${lines[1]}" = ".kit/rules/workflows.md" ]
  [ "${#lines[@]}" -eq 2 ]
}

@test "the prose style governs every file, after the rule and the style for the file's kind" {
  "${kit_checkout}/bin/kit" package add style/bash style/prose >/dev/null 2>&1
  mkdir bin && printf '#!/usr/bin/env bash\necho\n' > bin/tool
  run "${kit_checkout}/bin/kit" context bin/tool
  [ "${lines[0]}" = ".kit/rules/bash.md" ]
  [ "${lines[1]}" = ".kit/style/bash.md" ]
  [ "${lines[2]}" = ".kit/style/prose.md" ]
  [ "${#lines[@]}" -eq 3 ]
  run "${kit_checkout}/bin/kit" context bin/other
  [ "$output" = ".kit/style/prose.md" ]
  grep --quiet 'indicative' .kit/style/prose.md
}

@test "a relative path is taken from the directory the command was typed in" {
  mkdir docs && cd docs
  run "${kit_checkout}/bin/kit" context x.md
  [ "$output" = ".kit/rules/markdown.md" ]
  [ "$(cut -f 2 ../.git/kit/context.log)" = "docs/x.md" ]
}

@test "the json answer is one object per path, and every ask is logged" {
  run "${kit_checkout}/bin/kit" context --json justfile README.md
  [ "$status" -eq 0 ]
  [ "$(jq -c .rules <<<"${lines[0]}")" = '[".kit/rules/just.md"]' ]
  [ "$(jq -r .path <<<"${lines[1]}")" = "README.md" ]
  [ "$(cut -f 2 .git/kit/context.log | paste -s -d ' ' -)" = "justfile README.md" ]
}

@test "a style the repository takes is printed after the rule, and the hook asks for each to be read" {
  "${kit_checkout}/bin/kit" package add style/bash >/dev/null 2>&1
  run "${kit_checkout}/bin/kit" context lib/run.sh
  [ "${lines[0]}" = ".kit/rules/bash.md" ]
  [ "${lines[1]}" = ".kit/style/bash.md" ]
  [ "${#lines[@]}" -eq 2 ]
  input=$(printf '{"session_id":"s1","tool_name":"Write","tool_input":{"file_path":"%s/lib/run.sh"}}' "${PWD}")
  run "${kit_checkout}/bin/kit" context --claude <<<"${input}"
  [ "$(jq -r .hookSpecificOutput.additionalContext <<<"$output")" \
    = "kit: lib/run.sh is governed by .kit/rules/bash.md .kit/style/bash.md; read each before this change." ]
}

@test "the claude hook names the rules of the file about to be written, and nothing for a file none governs" {
  input=$(printf '{"session_id":"s1","tool_name":"Write","tool_input":{"file_path":"%s/lib/run.sh"}}' "${PWD}")
  run "${kit_checkout}/bin/kit" context --claude <<<"${input}"
  [ "$status" -eq 0 ]
  [ "$(jq -r .hookSpecificOutput.hookEventName <<<"$output")" = "PreToolUse" ]
  [ "$(jq -r .hookSpecificOutput.additionalContext <<<"$output")" \
    = "kit: lib/run.sh is governed by .kit/rules/bash.md; read it before this change." ]
  [ "$(cut -f 2 .git/kit/context.log)" = "lib/run.sh" ]
  input='{"session_id":"s1","tool_name":"Write","tool_input":{"file_path":"x.bin"}}'
  run "${kit_checkout}/bin/kit" context --claude <<<"${input}"
  [ "$status" -eq 0 ]
  [ -z "$output" ]
}

# ----------------------------------------------------------------------------------------------------------------------
