#!/usr/bin/env bats
# context.bats - kit context names a file's rules by path or by first line, logs the ask, and answers Claude's hook
# ----------------------------------------------------------------------------------------------------------------------

setup() {
  kit_checkout=$(cd "${BATS_TEST_DIRNAME}/.." && pwd)
  export KIT_HOME="${BATS_TEST_TMPDIR}/home"
  mkdir -p "${KIT_HOME}"
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
  [ "${lines[1]}" = ".kit/rules/yaml.md" ]
  [ "${#lines[@]}" -eq 2 ]
}

@test "the json answer is one object per path, and every ask is logged" {
  run "${kit_checkout}/bin/kit" context --json justfile README.md
  [ "$status" -eq 0 ]
  [ "$(jq -c .rules <<<"${lines[0]}")" = '[".kit/rules/just.md"]' ]
  [ "$(jq -r .path <<<"${lines[1]}")" = "README.md" ]
  [ "$(cut -f 2 .git/kit/context.log | paste -s -d ' ' -)" = "justfile README.md" ]
}

@test "the claude hook names the rules of the file about to be written, and nothing for a file none governs" {
  input='{"session_id":"s1","tool_name":"Write","tool_input":{"file_path":"'"${PWD}"'/lib/run.sh","content":""}}'
  run "${kit_checkout}/bin/kit" context --claude <<<"${input}"
  [ "$status" -eq 0 ]
  [ "$(jq -r .hookSpecificOutput.hookEventName <<<"$output")" = "PreToolUse" ]
  [ "$(jq -r .hookSpecificOutput.additionalContext <<<"$output")" = "kit: lib/run.sh is governed by .kit/rules/bash.md; read it before this change." ]
  [ "$(cut -f 2 .git/kit/context.log)" = "lib/run.sh" ]
  run "${kit_checkout}/bin/kit" context --claude <<<'{"session_id":"s1","tool_name":"Write","tool_input":{"file_path":"x.bin"}}'
  [ "$status" -eq 0 ]
  [ -z "$output" ]
}

# ----------------------------------------------------------------------------------------------------------------------
