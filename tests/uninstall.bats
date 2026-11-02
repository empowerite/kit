#!/usr/bin/env bats
# uninstall.bats - kit uninstall takes everything the kit placed out of the repository, and nothing else
# ----------------------------------------------------------------------------------------------------------------------
#
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
  [ "$status" -ne 0 ]
  grep --quiet '^Text of my own.$' AGENTS.md
  [ "$(cat kit.yml)" = "gate: just ci" ]
  [ ! -e .github ]
  run "${kit_checkout}/bin/kit" packages
  [ "$status" -ne 0 ]
  [[ "$output" == *"run: kit install"* ]]
}

@test "uninstall takes out what toolchain detection cached in .kit/cache/" {
  "${kit_checkout}/bin/kit" install >/dev/null 2>&1
  kit-bash 'use toolchain && toolchain:detect' >/dev/null
  [ -f .kit/cache/toolchains.tsv ] && [ -f .kit/cache/toolchains.key ]
  run "${kit_checkout}/bin/kit" uninstall
  [ "$status" -eq 0 ]
  [ ! -e .kit ]
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
