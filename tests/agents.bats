#!/usr/bin/env bats
# agents.bats - the agents/cursor, agents/copilot, and agents/gemini packages deliver each harness's native shapes
# ----------------------------------------------------------------------------------------------------------------------
#
# Each test gets an empty git repository and runs the kit from this checkout against it. A rule mirror names, in the
# harness's own frontmatter, the paths a rule governs, and its body points at the rule under .kit/rules/, so what is
# proven is that each mirror follows its rule and that the merged settings keep the repository's own entries.

bats_require_minimum_version 1.5.0

setup() {
  kit_checkout=$(cd "${BATS_TEST_DIRNAME}/../kit" && pwd)
  repo=$(mktemp -d)
  git -C "${repo}" init --quiet --initial-branch=main
  git -C "${repo}" config user.name t
  git -C "${repo}" config user.email t@example.com
  git -C "${repo}" config commit.gpgsign false
  export KIT_HOME="${repo}.home" XDG_CACHE_HOME="${repo}.cache"
  mkdir "${KIT_HOME}" "${XDG_CACHE_HOME}"
  cd "${repo}"
}

teardown() {
  rm -rf "${repo}" "${repo}.home" "${repo}.cache"
}

kit() { "${kit_checkout}/bin/kit" "$@"; }

# kinds prints the kind of every rule the kit package delivers.
kinds() {
  local rule
  for rule in "${kit_checkout}"/packages/kit/files/repo/.kit/rules/*.md
  do
    rule=${rule##*/}
    printf '%s\n' "${rule%.md}"
  done
}

# rule-paths <kind> prints the rule's paths: entries, comma-joined, as a harness's glob list.
rule-paths() {
  awk '/^---$/ { n++; next }
       n == 2 { exit }
       n == 1 && /^[a-z]+:/ { p = ($0 == "paths:") }
       n == 1 && p && /^  - / { gsub(/^  - "|"$/, ""); printf "%s%s", sep, $0; sep = "," }
       END { printf "\n" }' "${kit_checkout}/packages/kit/files/repo/.kit/rules/$1.md"
}

# claude-prefixes prints the command prefixes of the Claude deny list, with one that another covers left out.
claude-prefixes() {
  jq --raw-output '.permissions.deny[] | ltrimstr("Bash(") | rtrimstr(" *)")' \
    "${kit_checkout}/packages/agents/claude/files/repo/.claude/settings.json" \
    | awk '{ all[NR] = $0 } END { for (i in all) { covered = 0; for (j in all) if (i != j && index(all[i], all[j] " ") == 1) covered = 1; if (!covered) print all[i] } }' \
    | sort
}

@test "agents/cursor places one rule mirror per kind, attached by the rule's paths, whose body points at the rule" {
  run kit install agents/cursor
  [ "$status" -eq 0 ]
  local kind
  for kind in $(kinds)
  do
    [ -f ".cursor/rules/kit-${kind}.mdc" ]
    [ "$(head -n 1 ".cursor/rules/kit-${kind}.mdc")" = "---" ]
    grep --quiet "^description: ." ".cursor/rules/kit-${kind}.mdc"
    grep --quiet --fixed-strings "globs: $(rule-paths "${kind}")" ".cursor/rules/kit-${kind}.mdc"
    grep --quiet --line-regexp "alwaysApply: false" ".cursor/rules/kit-${kind}.mdc"
    grep --quiet --fixed-strings "@.kit/rules/${kind}.md" ".cursor/rules/kit-${kind}.mdc"
    grep --quiet --fixed-strings "  .cursor/rules/kit-${kind}.mdc" .kit/manifest
  done
  [ "$(ls .cursor/rules | wc -l | tr -d ' ')" = "$(kinds | wc -l | tr -d ' ')" ]
  printf 'edited\n' >> .cursor/rules/kit-bash.mdc
  run kit check
  [ "$status" -ne 0 ]
}

@test "agents/copilot places one instruction file per kind, applied to the rule's paths, whose body points at the rule" {
  run kit install agents/copilot
  [ "$status" -eq 0 ]
  local kind
  for kind in $(kinds)
  do
    [ -f ".github/instructions/kit-${kind}.instructions.md" ]
    [ "$(head -n 1 ".github/instructions/kit-${kind}.instructions.md")" = "---" ]
    grep --quiet --line-regexp --fixed-strings "applyTo: \"$(rule-paths "${kind}")\"" \
      ".github/instructions/kit-${kind}.instructions.md"
    grep --quiet --fixed-strings ".kit/rules/${kind}.md" ".github/instructions/kit-${kind}.instructions.md"
    grep --quiet --fixed-strings "  .github/instructions/kit-${kind}.instructions.md" .kit/manifest
  done
  [ "$(ls .github/instructions | wc -l | tr -d ' ')" = "$(kinds | wc -l | tr -d ' ')" ]
}

@test "agents/gemini merges the context file names and the deny list into .gemini/settings.json, keeps the repository's entries, and remove takes the kit's out" {
  mkdir .gemini
  printf '%s\n' '{"context": {"fileName": ["TEAM.md"]}, "tools": {"exclude": ["run_shell_command(rm -rf)"]}, "ui": {"theme": "Dark"}}' \
    > .gemini/settings.json
  run kit install agents/gemini
  [ "$status" -eq 0 ]
  [ "$(jq --compact-output '.context.fileName' .gemini/settings.json)" = '["TEAM.md","GEMINI.md","AGENTS.md"]' ]
  [ "$(jq --raw-output '.tools.exclude[0]' .gemini/settings.json)" = 'run_shell_command(rm -rf)' ]
  [ "$(jq --raw-output '.ui.theme' .gemini/settings.json)" = 'Dark' ]
  run kit check
  [ "$status" -eq 0 ]
  run kit package remove agents/gemini
  [ "$status" -eq 0 ]
  [ "$(jq --compact-output '.context.fileName' .gemini/settings.json)" = '["TEAM.md"]' ]
  [ "$(jq --compact-output '.tools.exclude' .gemini/settings.json)" = '["run_shell_command(rm -rf)"]' ]
  [ "$(jq --raw-output '.ui.theme' .gemini/settings.json)" = 'Dark' ]
}

@test "agents/gemini denies the commands agents/claude denies, as prefixes, which is how Gemini matches a shell command" {
  kit install agents/gemini
  expected=$(claude-prefixes | sed 's/.*/run_shell_command(&)/')
  [ "$(jq --raw-output '.tools.exclude[]' .gemini/settings.json | sort)" = "${expected}" ]
}

# ----------------------------------------------------------------------------------------------------------------------
