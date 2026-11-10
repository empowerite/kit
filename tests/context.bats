#!/usr/bin/env bats
# context.bats - kit context classifies a file into its kinds, names the rules by kind and by what a kind embeds, logs the ask, and answers Claude's hook
# ----------------------------------------------------------------------------------------------------------------------

bats_require_minimum_version 1.5.0

load helper

setup_file() {
  world-repo installed
}

setup() {
  world-copy-repo installed
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
  "${kit_checkout}/bin/kit" packages add style/bash style/prose >/dev/null 2>&1
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
  "${kit_checkout}/bin/kit" packages add style/bash >/dev/null 2>&1
  [ -f .kit/kinds/bash.yml ]
  grep --quiet --line-regexp '  - "\*\*/\*.sh"' .kit/kinds/bash.yml
  for file in .kit/rules/*.md .kit/styles/*.md
  do
    run ! grep --quiet --line-regexp 'paths:' "${file}"
    run ! grep --quiet --line-regexp 'content:' "${file}"
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
  run "${kit_checkout}/bin/kit" packages add toml
  [ "$status" -eq 0 ]
  run "${kit_checkout}/bin/kit" context Cargo.toml
  [ "$output" = ".kit/rules/toml.md" ]
  mkdir -p .kit/packages/mdx/files/repo/.kit/kinds
  printf 'description: "Markdown with components counts as markdown"\ndepends:\n  - kit\n' > .kit/packages/mdx/package.yml
  printf '# The markdown kind, with .mdx\nname: markdown\npaths:\n  - "**/*.md"\n  - "**/*.mdx"\n' > .kit/packages/mdx/files/repo/.kit/kinds/markdown.yml
  run "${kit_checkout}/bin/kit" packages add mdx
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
  run "${kit_checkout}/bin/kit" packages add conf
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
  [ "$status" -eq 1 ]
  [[ "$output" == *".kit/kinds/config.yml: a kind has no name:"* ]]
  printf 'name: toml\npaths:\n  - "**/*.toml"\n---\nname: toml\npaths:\n  - "**/*.tml"\n' \
    > .kit/packages/conf/files/repo/.kit/kinds/config.yml
  run "${kit_checkout}/bin/kit" context Cargo.toml
  [ "$status" -eq 1 ]
  [[ "$output" == *".kit/kinds/config.yml: the kind toml is defined twice"* ]]
}

@test "a style the repository takes is printed after the rule, and the hook asks for each to be read" {
  "${kit_checkout}/bin/kit" packages add style/bash >/dev/null 2>&1
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

@test "a kinds file is YAML: CRLF endings, a flow list, a quoted name, a comment after an item, and a leading --- all read" {
  mkdir -p .kit/packages/conf/files/repo/.kit/kinds .kit/packages/conf/files/repo/.kit/rules
  printf 'description: "Two configuration kinds in one file"\ndepends:\n  - kit\n' > .kit/packages/conf/package.yml
  printf -- '---\r\n# configuration\r\nname: "toml"\r\npaths: ["**/*.toml", "**/*.tml"] # both spellings\r\n---\r\nname: ini\r\npaths:\r\n  - "**/*.ini" # windows\r\n---\r\n' \
    > .kit/packages/conf/files/repo/.kit/kinds/config.yml
  printf -- '---\r\nkind: [toml, ini]\r\n---\r\n# Configuration\r\n' > .kit/packages/conf/files/repo/.kit/rules/config.md
  run "${kit_checkout}/bin/kit" packages add conf
  [ "$status" -eq 0 ]
  run "${kit_checkout}/bin/kit" context --json Cargo.toml build.tml setup.ini
  [ "$status" -eq 0 ]
  [ "$(jq -sc 'map(.kinds)' <<<"$output")" = '[["any","toml"],["any","toml"],["any","ini"]]' ]
  [ "$(jq -sc 'map(.rules)' <<<"$output")" = '[[".kit/rules/config.md"],[".kit/rules/config.md"],[".kit/rules/config.md"]]' ]
  run "${kit_checkout}/bin/kit" kinds --json toml ini
  [ "$status" -eq 0 ]
  [ "$(jq -sc 'map([.name, .paths])' <<<"$output")" = '[["ini",["**/*.ini"]],["toml",["**/*.toml","**/*.tml"]]]' ]
}

@test "a repository with no kinds at all governs nothing: kit context prints nothing and exits 0, as does the hook" {
  fresh-repo bare >/dev/null
  run "${kit_checkout}/bin/kit" context foo.sh
  [ "$status" -eq 0 ]
  [ -z "$output" ]
  run "${kit_checkout}/bin/kit" context --json foo.sh
  [ "$status" -eq 0 ]
  [ "$(jq -c '[.kinds, .rules]' <<<"$output")" = '[[],[]]' ]
  input='{"session_id":"s1","tool_name":"Write","tool_input":{"file_path":"foo.sh"}}'
  run "${kit_checkout}/bin/kit" context --claude <<<"${input}"
  [ "$status" -eq 0 ]
  [ -z "$output" ]
}

@test "two lookups that compile changed kinds at once both succeed and say nothing, though both prune the old matcher" {
  # kit context --json asks for a path's kinds twice at once; after the kinds change, both compile them and both
  # prune the old matcher, so the one that finds it gone already must say nothing, or its words land in the JSON
  "${kit_checkout}/bin/kit" hydrate >/dev/null 2>&1
  kit_bash_modules=context run kit-bash '
    dir="$(git rev-parse --path-format=absolute --git-common-dir)/kit"
    mkdir -p "${dir}"
    for round in $(seq 40)
    do
      : > "${dir}/kinds-0.awk"
      rm -f "${dir}"/kinds-[1-9]*.awk
      context:matcher >/dev/null & context:matcher >/dev/null & wait
    done'
  [ "$status" -eq 0 ]
  [ -z "$output" ]
  # the matcher of other kinds is younger than a day, so another worktree may be using it
  [ -f "$(git rev-parse --path-format=absolute --git-common-dir)/kit/kinds-0.awk" ]
}

@test "a compile prunes a matcher of other kinds older than a day, and only that one" {
  local dir
  dir="$(git rev-parse --path-format=absolute --git-common-dir)/kit"
  mkdir -p "${dir}"
  rm -f "${dir}"/kinds-*.awk
  : > "${dir}/kinds-1.awk"
  : > "${dir}/kinds-2.awk"
  touch -t 202001010000 "${dir}/kinds-1.awk"
  run "${kit_checkout}/bin/kit" context README.md
  [ "$status" -eq 0 ]
  [ "$output" = ".kit/rules/markdown.md" ]
  [ ! -e "${dir}/kinds-1.awk" ]
  [ -f "${dir}/kinds-2.awk" ]
}

@test "a matcher another compile prunes between its lookup and its read is compiled again, never read as no kinds" {
  # the first lookup's matcher is taken away before classify reads it, as another worktree's prune would
  kit_bash_modules=context run kit-bash '
    eval "$(declare -f context:matcher | sed "1s/^context:matcher /context:matcher-real /")"
    context:matcher() {
      local file
      file=$(context:matcher-real) || return 1
      if [[ ! -e "${file}.taken" ]]
      then
        : > "${file}.taken"
        rm -f "${file}"
      fi
      printf "%s\n" "${file}"
    }
    printf "README.md\n" | context:classify'
  [ "$status" -eq 0 ]
  [[ "$output" == "README.md"$'\t'*"markdown"* ]]
}

@test "a compiled matcher awk cannot run fails the lookup, and is never read as no kinds" {
  local dir file
  "${kit_checkout}/bin/kit" context README.md >/dev/null
  dir="$(git rev-parse --path-format=absolute --git-common-dir)/kit"
  for file in "${dir}"/kinds-*.awk
  do
    printf 'BEGIN {\n' > "${file}"
  done
  run "${kit_checkout}/bin/kit" context README.md
  [ "$status" -eq 1 ]
  [[ "$output" == *"awk cannot run the compiled kinds"* ]]
  [[ "$output" != *".kit/rules/"* ]]
}

@test "a repository that declares a kit version and has no kind says so, and names kit hydrate" {
  # a verb hydrates first and puts the kinds back, so the lookup runs alone, as a hook's would under kit serve
  rm -f .kit/kinds/*.yml
  kit_bash_modules=context run kit-bash 'context:load'
  [ "$status" -eq 1 ]
  [[ "$output" == *"no kind is defined under .kit/kinds; run: kit hydrate"* ]]
  kit_bash_modules=context run kit-bash 'printf "README.md\n" | context:classify'
  [ "$status" -eq 1 ]
  [[ "$output" != *"README.md"$'\t'* ]]
  # a verb puts the kinds back, and finds them
  run --separate-stderr "${kit_checkout}/bin/kit" context README.md
  [ "$status" -eq 0 ]
  [ "$output" = ".kit/rules/markdown.md" ]
  [ -f .kit/kinds/markdown.yml ]
}

# ----------------------------------------------------------------------------------------------------------------------
