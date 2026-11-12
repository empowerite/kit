#!/usr/bin/env bats
# uninstall.bats - kit uninstall takes everything the kit placed out of the repository, and nothing else
# ----------------------------------------------------------------------------------------------------------------------
# Each test gets an empty git repository under a temporary directory, with a few files of its own, installs the kit
# from this checkout into it, and takes it out again.

bats_require_minimum_version 1.5.0

load helper

setup() {
  fresh-repo
  mkdir .claude
  printf '{"model":"x"}\n' > .claude/settings.json
  printf '# Mine\n\nText of my own.\n' > AGENTS.md
  printf 'gate: just ci\n' > kit.yml
}

# tree prints every path under the repository but .git, one per line, sorted.
tree() {
  find . -path ./.git -prune -o -print | LC_ALL=C sort
}

@test "uninstall takes out every file, entry, and region the kit placed, keeps the repository's own, and drops the declaration" {
  "${kit_checkout}/bin/kit" install agents/claude github/ci >/dev/null 2>&1
  [ -f .github/workflows/kit-ci.yml ] && [ -f .kit/1st.md ] && grep --quiet 'kit:generate' AGENTS.md
  run "${kit_checkout}/bin/kit" uninstall
  [ "$status" -eq 0 ]
  [[ "$output" == *"kit is uninstalled;"* ]]
  [ ! -e .github/workflows/kit-ci.yml ]
  [ ! -e .kit ]
  [ ! -e .claude/CLAUDE.md ]
  [ "$(jq --compact-output . .claude/settings.json)" = '{"model":"x"}' ]
  run grep 'kit:' AGENTS.md
  [ "$status" -eq 1 ]
  grep --quiet '^Text of my own.$' AGENTS.md
  [ "$(cat kit.yml)" = "gate: just ci" ]
  [ ! -e .github ]
  run "${kit_checkout}/bin/kit" packages
  [ "$status" -eq 1 ]
  [[ "$output" == *"run: cd "*" && kit install"* ]]
}

@test "uninstall says a region is taken out only of a file that held one" {
  "${kit_checkout}/bin/kit" install agents/claude >/dev/null 2>&1
  # the repository took the kit's region out of .gitignore by hand, and kept a line of its own
  printf 'mine\n' > .gitignore
  run "${kit_checkout}/bin/kit" uninstall --dry-run
  [ "$status" -eq 0 ]
  [[ "$output" == *"AGENTS.md: the kit's region would be taken out"* ]]
  [[ "$output" != *".gitignore: the kit's region would be taken out"* ]]
  run "${kit_checkout}/bin/kit" uninstall
  [ "$status" -eq 0 ]
  [[ "$output" == *"AGENTS.md: the kit's region taken out"* ]]
  [[ "$output" != *".gitignore: the kit's region taken out"* ]]
  [ "$(cat .gitignore)" = mine ]
}

@test "uninstall removes a JSON file of the kit's entries alone, and names what is left in lefthook.yml and the hook" {
  rm .claude/settings.json
  "${kit_checkout}/bin/kit" install agents/claude hooks/lefthook >/dev/null 2>&1
  [ -f .claude/settings.json ] && [ -f .mcp.json ]
  printf '# lefthook.yml - the git hooks\nextends: [.kit/lefthook.yml]\n' > lefthook.yml
  mkdir -p .git/hooks
  printf '#!/bin/sh\nlefthook run pre-commit\n' > .git/hooks/pre-commit
  run "${kit_checkout}/bin/kit" uninstall
  [ "$status" -eq 0 ]
  # the kit made both files, and nothing else was ever in them
  [ ! -e .claude/settings.json ]
  [ ! -e .mcp.json ]
  [ ! -e .claude ]
  # lefthook.yml and the hook are the repository's; kit names them and leaves them
  [ "$(sed -n 2p lefthook.yml)" = "extends: [.kit/lefthook.yml]" ]
  [[ "$output" == *"left to do in lefthook.yml: take out the line that names .kit/lefthook.yml"* ]]
  [[ "$output" == *"left to do: the pre-commit hook still runs lefthook; run: cd "*" && lefthook uninstall"* ]]
}

@test "uninstall takes out what toolchain detection cached in .kit/cache/" {
  "${kit_checkout}/bin/kit" install >/dev/null 2>&1
  kit-bash 'use toolchain && toolchain:detect' >/dev/null
  [ -f .kit/cache/toolchains.tsv ] && [ -f .kit/cache/toolchains.key ]
  run "${kit_checkout}/bin/kit" uninstall
  [ "$status" -eq 0 ]
  [ ! -e .kit ]
}

@test "uninstall at the desk takes out the project toolchain choice kept in the git directory" {
  "${kit_checkout}/bin/kit" install >/dev/null 2>&1
  kit-bash 'use toolchain && toolchain:choose project ambient'
  [ "$(cat .git/kit/toolchain)" = ambient ]
  run "${kit_checkout}/bin/kit" uninstall --dry-run
  [[ "$output" == *".git/kit/toolchain: would be removed"* ]]
  [ -f .git/kit/toolchain ]
  run "${kit_checkout}/bin/kit" uninstall
  [ "$status" -eq 0 ]
  [ ! -e .git/kit/toolchain ]
}

@test "uninstall in a linked worktree takes out that worktree's own toolchain choice, and leaves the clone's" {
  "${kit_checkout}/bin/kit" install >/dev/null 2>&1
  git add --all && git commit --quiet --message init
  kit-bash 'use toolchain && toolchain:choose project ambient'
  git worktree add --quiet "${BATS_TEST_TMPDIR}/feature" -b feature
  cd "${BATS_TEST_TMPDIR}/feature"
  "${kit_checkout}/bin/kit" hydrate >/dev/null 2>&1
  kit-bash 'use toolchain && toolchain:choose project ambient'
  [ "$(cat "${repo}/.git/worktrees/feature/kit/toolchain")" = ambient ]
  run "${kit_checkout}/bin/kit" uninstall --dry-run
  [[ "$output" == *"worktrees/feature/kit/toolchain: would be removed"* ]]
  run "${kit_checkout}/bin/kit" uninstall
  [ "$status" -eq 0 ]
  [ ! -e "${repo}/.git/worktrees/feature/kit" ]
  [ "$(cat "${repo}/.git/kit/toolchain")" = ambient ]
  cd "${repo}"
}

@test "uninstall leaves a copied file the repository edited, names it, and takes out the rest" {
  "${kit_checkout}/bin/kit" install github/ci >/dev/null 2>&1
  printf '# edited\n' >> .github/workflows/kit-ci.yml
  run "${kit_checkout}/bin/kit" uninstall
  [ "$status" -eq 0 ]
  [ -f .github/workflows/kit-ci.yml ]
  [[ "$output" == *".github/workflows/kit-ci.yml: edited since the kit placed it; it is left as it is"* ]]
  [ ! -e .kit ]
  [ "$(cat kit.yml)" = "gate: just ci" ]
}

@test "uninstall --dry-run names each path it would take out and changes nothing" {
  "${kit_checkout}/bin/kit" install agents/claude github/ci >/dev/null 2>&1
  before=$(tree; cat kit.yml AGENTS.md .claude/settings.json)
  run "${kit_checkout}/bin/kit" uninstall --dry-run
  [ "$status" -eq 0 ]
  [[ "$output" == *".github/workflows/kit-ci.yml: would be removed"* ]]
  [[ "$output" == *".claude/settings.json: the kit's entries would be taken out"* ]]
  [[ "$output" == *"AGENTS.md: the kit's region would be taken out"* ]]
  [[ "$output" == *"kit.yml: version and packages would be dropped"* ]]
  [ "$(tree; cat kit.yml AGENTS.md .claude/settings.json)" = "${before}" ]
}

@test "uninstall removes a kit.yml that held nothing but the declaration, and refuses when there is nothing to take out" {
  rm kit.yml
  "${kit_checkout}/bin/kit" install >/dev/null 2>&1
  [ -f kit.yml ]
  run "${kit_checkout}/bin/kit" uninstall
  [ "$status" -eq 0 ]
  [ ! -e kit.yml ]
  run "${kit_checkout}/bin/kit" uninstall
  [ "$status" -eq 1 ]
  [[ "$output" == *"nothing to uninstall"* ]]
}

@test "uninstall keeps the repository's own packages under .kit/packages/" {
  mkdir -p .kit/packages/house/files/repo/.kit/rules
  printf 'description: "ours"\ndepends:\n  - kit\n' > .kit/packages/house/package.yml
  printf -- '---\nkind:\n  - any\n---\n# Ours\n' > .kit/packages/house/files/repo/.kit/rules/ours.md
  "${kit_checkout}/bin/kit" install house >/dev/null 2>&1
  [ -e .kit/rules/ours.md ]
  run "${kit_checkout}/bin/kit" uninstall
  [ "$status" -eq 0 ]
  [ -f .kit/packages/house/package.yml ]
  [ ! -e .kit/rules ]
  [ ! -e .kit/manifest ]
  [ "$(ls -A .kit)" = "packages" ]
}

@test "uninstall refuses a region that never closes, keeps every line after it, and keeps the declaration" {
  local before sound
  printf 'mine-before\n' > .gitignore
  "${kit_checkout}/bin/kit" install github/ci >/dev/null 2>&1
  printf 'mine-after-1\nmine-after-2\n' >> .gitignore
  sound=$(cat .gitignore)
  grep --invert-match --line-regexp --fixed-strings '# kit:end .gitignore' .gitignore > gitignore.new
  mv gitignore.new .gitignore
  before=$(cat .gitignore)
  run "${kit_checkout}/bin/kit" uninstall --dry-run
  [ "$status" -eq 1 ]
  [[ "$output" == *".gitignore: the kit's region opens with '# kit:generate .gitignore' and never closes"* ]]
  run "${kit_checkout}/bin/kit" uninstall
  [ "$status" -eq 1 ]
  [[ "$output" == *"1 path(s) could not be taken out"* ]]
  [[ "$output" == *".gitignore: the kit's region opens with '# kit:generate .gitignore' and never closes"* ]]
  [ "$(cat .gitignore)" = "${before}" ]
  grep --quiet '^version: ' kit.yml
  # the refusal comes before anything is taken out
  [ -f .kit/1st.md ]
  [ -f .kit/manifest ]
  [ -f .github/workflows/kit-ci.yml ]
  [ "$(declared packages)" = "kit github/ci" ]
  run "${kit_checkout}/bin/kit" packages remove github/ci
  [ "$status" -eq 0 ]
  run "${kit_checkout}/bin/kit" packages remove kit
  [ "$status" -eq 1 ]
  [[ "$output" == *"1 path(s) could not be taken out"* ]]
  [[ "$output" == *"then run: cd "*" && kit packages remove kit"* ]]
  [ "$(cat .gitignore)" = "${before}" ]
  [ -f .kit/1st.md ]
  [ -f .kit/rules/bash.md ]
  [ "$(declared packages)" = "kit" ]
  # with the end line back, the same uninstall takes everything out and keeps the repository's lines
  printf '%s\n' "${sound}" > .gitignore
  run "${kit_checkout}/bin/kit" uninstall
  [ "$status" -eq 0 ]
  [ "$(cat .gitignore)" = "$(printf 'mine-before\nmine-after-1\nmine-after-2')" ]
  [ ! -e .kit/1st.md ]
  [ ! -e .github/workflows/kit-ci.yml ]
}
