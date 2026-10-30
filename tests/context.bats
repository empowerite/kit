#!/usr/bin/env bats
# context.bats - kit context classifies a file into its kinds, names the rules by kind and by what a kind embeds, logs the ask, and answers Claude's hook
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
  [ "${lines[0]}" = ".kit/rules/just.md" ]
  [ "${lines[1]}" = ".kit/rules/bash.md" ]
  [ "${#lines[@]}" -eq 2 ]
  mkdir bin && printf '#!/usr/bin/env bash\necho\n' > bin/tool
  run "${kit_checkout}/bin/kit" context bin/tool
  [ "$output" = ".kit/rules/bash.md" ]
  run "${kit_checkout}/bin/kit" context bin/other
  [ "$status" -eq 0 ]
  [ -z "$output" ]
  run "${kit_checkout}/bin/kit" context docs/a.md "${PWD}/.github/workflows/x.yml" lib/x.md
  [ "${lines[0]}" = ".kit/rules/markdown.md" ]
  [ "${lines[1]}" = ".kit/rules/workflows.md" ]
  [ "${lines[2]}" = ".kit/rules/bash.md" ]
  [ "${#lines[@]}" -eq 3 ]
}

@test "the prose style governs every file, after the rule and the style for the file's kind" {
  "${kit_checkout}/bin/kit" package add style/bash style/prose >/dev/null 2>&1
  mkdir bin && printf '#!/usr/bin/env bash\necho\n' > bin/tool
  run "${kit_checkout}/bin/kit" context bin/tool
  [ "${lines[0]}" = ".kit/rules/bash.md" ]
  [ "${lines[1]}" = ".kit/styles/bash.md" ]
  [ "${lines[2]}" = ".kit/styles/prose.md" ]
  [ "${#lines[@]}" -eq 3 ]
  run "${kit_checkout}/bin/kit" context bin/other
  [ "$output" = ".kit/styles/prose.md" ]
  grep --quiet 'indicative' .kit/styles/prose.md
}

@test "a relative path is taken from the directory the command was typed in" {
  mkdir docs && cd docs
  run "${kit_checkout}/bin/kit" context x.md
  [ "$output" = ".kit/rules/markdown.md" ]
  [ "$(cut -f 2 ../.git/kit/context.log)" = "docs/x.md" ]
}

@test "the json answer is one object per path with its kinds and its rules, and every ask is logged" {
  run "${kit_checkout}/bin/kit" context --json justfile README.md
  [ "$status" -eq 0 ]
  [ "$(jq -c .kinds <<<"${lines[0]}")" = '["any","just"]' ]
  [ "$(jq -c .rules <<<"${lines[0]}")" = '[".kit/rules/just.md",".kit/rules/bash.md"]' ]
  [ "$(jq -r .path <<<"${lines[1]}")" = "README.md" ]
  [ "$(jq -c .kinds <<<"${lines[1]}")" = '["any","markdown"]' ]
  [ "$(cut -f 2 .git/kit/context.log | paste -s -d ' ' -)" = "justfile README.md" ]
}

@test "a kind is defined once under .kit/kinds/, and the rule and the style of a kind name it, so neither carries a glob" {
  "${kit_checkout}/bin/kit" package add style/bash >/dev/null 2>&1
  [ -f .kit/kinds/bash.yml ]
  grep --quiet --line-regexp '  - "\*\*/\*.sh"' .kit/kinds/bash.yml
  for file in .kit/rules/*.md .kit/styles/*.md
  do
    ! grep --quiet --line-regexp 'paths:' "${file}"
    ! grep --quiet --line-regexp 'content:' "${file}"
    grep --quiet --line-regexp 'kind:' "${file}"
  done
  grep --quiet --line-regexp '  - bash' .kit/kinds/just.yml
  grep --quiet --line-regexp 'embeds:' .kit/kinds/workflows.yml
  mkdir -p .github/workflows && printf 'on: push\n' > .github/workflows/w.yml
  run "${kit_checkout}/bin/kit" context --json .github/workflows/w.yml
  [ "$(jq -c .kinds <<<"$output")" = '["any","workflows","yaml"]' ]
  [ "$(jq -c .rules <<<"$output")" = '[".kit/rules/workflows.md",".kit/rules/bash.md"]' ]
}

@test "a repository package adds a kind with its rule, and replaces a delivered kind to widen it" {
  mkdir -p .kit/packages/toml/files/repo/.kit/kinds .kit/packages/toml/files/repo/.kit/rules
  printf 'description: "The TOML kind and its rule"\ndepends:\n  - kit\n' > .kit/packages/toml/package.yml
  printf '# The TOML kind\npaths:\n  - "**/*.toml"\n' > .kit/packages/toml/files/repo/.kit/kinds/toml.yml
  printf -- '---\nkind:\n  - toml\n---\n# TOML\n\nQuote every key.\n' > .kit/packages/toml/files/repo/.kit/rules/toml.md
  run "${kit_checkout}/bin/kit" package add toml
  [ "$status" -eq 0 ]
  run "${kit_checkout}/bin/kit" context Cargo.toml
  [ "$output" = ".kit/rules/toml.md" ]
  mkdir -p .kit/packages/mdx/files/repo/.kit/kinds
  printf 'description: "Markdown with components counts as markdown"\ndepends:\n  - kit\n' > .kit/packages/mdx/package.yml
  printf '# The markdown kind, with .mdx\npaths:\n  - "**/*.md"\n  - "**/*.mdx"\n' > .kit/packages/mdx/files/repo/.kit/kinds/markdown.yml
  run "${kit_checkout}/bin/kit" package add mdx
  [ "$status" -eq 0 ]
  run "${kit_checkout}/bin/kit" context docs/page.mdx
  [ "$output" = ".kit/rules/markdown.md" ]
  run "${kit_checkout}/bin/kit" check
  [ "$status" -eq 0 ]
}

@test "a style the repository takes is printed after the rule, and the hook asks for each to be read" {
  "${kit_checkout}/bin/kit" package add style/bash >/dev/null 2>&1
  run "${kit_checkout}/bin/kit" context lib/run.sh
  [ "${lines[0]}" = ".kit/rules/bash.md" ]
  [ "${lines[1]}" = ".kit/styles/bash.md" ]
  [ "${#lines[@]}" -eq 2 ]
  input=$(printf '{"session_id":"s1","tool_name":"Write","tool_input":{"file_path":"%s/lib/run.sh"}}' "${PWD}")
  run "${kit_checkout}/bin/kit" context --claude <<<"${input}"
  [ "$(jq -r .hookSpecificOutput.additionalContext <<<"$output")" \
    = "kit: lib/run.sh is governed by .kit/rules/bash.md .kit/styles/bash.md; read each before this change." ]
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
