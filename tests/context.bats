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

@test "the prose style governs every text file, the kind made of the kit's kinds, after the rule and the style for its kind" {
  "${kit_checkout}/bin/kit" package add style/bash style/prose >/dev/null 2>&1
  mkdir bin && printf '#!/usr/bin/env bash\necho\n' > bin/tool
  run "${kit_checkout}/bin/kit" context bin/tool
  [ "${lines[0]}" = ".kit/rules/bash.md" ]
  [ "${lines[1]}" = ".kit/styles/bash.md" ]
  [ "${lines[2]}" = ".kit/styles/prose.md" ]
  [ "${#lines[@]}" -eq 3 ]
  run "${kit_checkout}/bin/kit" context notes.txt
  [ "$output" = ".kit/styles/prose.md" ]
  run "${kit_checkout}/bin/kit" context bin/other image.png
  [ -z "$output" ]
  run "${kit_checkout}/bin/kit" context --json bin/tool
  [ "$(jq -c .kinds <<<"$output")" = '["any","bash","text"]' ]
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
  [ "$(jq -c .kinds <<<"${lines[0]}")" = '["any","just","text"]' ]
  [ "$(jq -c .rules <<<"${lines[0]}")" = '[".kit/rules/just.md",".kit/rules/bash.md"]' ]
  [ "$(jq -r .path <<<"${lines[1]}")" = "README.md" ]
  [ "$(jq -c .kinds <<<"${lines[1]}")" = '["any","markdown","text"]' ]
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
  [ "$(jq -c .kinds <<<"$output")" = '["any","text","workflows","yaml"]' ]
  [ "$(jq -c .rules <<<"$output")" = '[".kit/rules/workflows.md",".kit/rules/bash.md"]' ]
}

@test "a repository package adds a kind with its rule, and replaces a delivered kind to widen it" {
  mkdir -p .kit/packages/toml/files/repo/.kit/kinds .kit/packages/toml/files/repo/.kit/rules
  printf 'description: "The TOML kind and its rule"\ndepends:\n  - kit\n' > .kit/packages/toml/package.yml
  printf '# The TOML kind\nname: toml\npaths:\n  - "**/*.toml"\n' > .kit/packages/toml/files/repo/.kit/kinds/toml.yml
  printf -- '---\nkind:\n  - toml\n---\n# TOML\n\nQuote every key.\n' > .kit/packages/toml/files/repo/.kit/rules/toml.md
  run "${kit_checkout}/bin/kit" package add toml
  [ "$status" -eq 0 ]
  run "${kit_checkout}/bin/kit" context Cargo.toml
  [ "$output" = ".kit/rules/toml.md" ]
  mkdir -p .kit/packages/mdx/files/repo/.kit/kinds
  printf 'description: "Markdown with components counts as markdown"\ndepends:\n  - kit\n' > .kit/packages/mdx/package.yml
  printf '# The markdown kind, with .mdx\nname: markdown\npaths:\n  - "**/*.md"\n  - "**/*.mdx"\n' > .kit/packages/mdx/files/repo/.kit/kinds/markdown.yml
  run "${kit_checkout}/bin/kit" package add mdx
  [ "$status" -eq 0 ]
  run "${kit_checkout}/bin/kit" context docs/page.mdx
  [ "$output" = ".kit/rules/markdown.md" ]
  run "${kit_checkout}/bin/kit" check
  [ "$status" -eq 0 ]
}

@test "a kind is named by its name:, never by its file, one file may define several, a kind may be made of others, and a nameless or doubled kind is refused" {
  mkdir -p .kit/packages/conf/files/repo/.kit/kinds .kit/packages/conf/files/repo/.kit/rules
  printf 'description: "Two configuration kinds in one file"\ndepends:\n  - kit\n' > .kit/packages/conf/package.yml
  printf '# configuration\nname: toml\npaths:\n  - "**/*.toml"\n---\nname: ini\npaths:\n  - "**/*.ini"\n' \
    > .kit/packages/conf/files/repo/.kit/kinds/config.yml
  printf -- '---\nkind:\n  - toml\n---\n# TOML\n' > .kit/packages/conf/files/repo/.kit/rules/toml.md
  printf -- '---\nkind:\n  - ini\n---\n# INI\n' > .kit/packages/conf/files/repo/.kit/rules/ini.md
  run "${kit_checkout}/bin/kit" package add conf
  [ "$status" -eq 0 ]
  run "${kit_checkout}/bin/kit" context --json Cargo.toml setup.ini
  [ "$(jq -sc 'map(.kinds)' <<<"$output")" = '[["any","toml"],["any","ini"]]' ]
  [ "$(jq -sc 'map(.rules)' <<<"$output")" = '[[".kit/rules/toml.md"],[".kit/rules/ini.md"]]' ]
  printf -- '---\nname: config\nkinds:\n  - toml\n  - ini\n' >> .kit/packages/conf/files/repo/.kit/kinds/config.yml
  run "${kit_checkout}/bin/kit" context --json setup.ini
  [ "$(jq -c .kinds <<<"$output")" = '["any","config","ini"]' ]
  printf '# configuration\nname: toml\npaths:\n  - "**/*.toml"\n---\npaths:\n  - "**/*.ini"\n' \
    > .kit/packages/conf/files/repo/.kit/kinds/config.yml
  run "${kit_checkout}/bin/kit" context Cargo.toml
  [ "$status" -ne 0 ]
  [[ "$output" == *".kit/kinds/config.yml: a kind has no name:"* ]]
  printf 'name: toml\npaths:\n  - "**/*.toml"\n---\nname: toml\npaths:\n  - "**/*.tml"\n' \
    > .kit/packages/conf/files/repo/.kit/kinds/config.yml
  run "${kit_checkout}/bin/kit" context Cargo.toml
  [ "$status" -ne 0 ]
  [[ "$output" == *".kit/kinds/config.yml: the kind toml is defined twice"* ]]
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
