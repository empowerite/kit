#!/usr/bin/env bats
# package.bats - kit package select makes exactly the named packages, and what they need, the active set
# ----------------------------------------------------------------------------------------------------------------------
#
# Each test gets an empty git repository under a temporary directory and runs the kit from this checkout against it.

bats_require_minimum_version 1.5.0

setup() {
  kit_checkout=$(cd "${BATS_TEST_DIRNAME}/../kit" && pwd)
  repo=$(mktemp -d)
  git -C "${repo}" init --quiet --initial-branch=main
  # a home directory of the test's own, so no test ever writes under the real one
  export KIT_HOME="${repo}.home" XDG_CACHE_HOME="${repo}.cache" NO_COLOR=1
  mkdir "${KIT_HOME}" "${XDG_CACHE_HOME}"
  cd "${repo}"
}

teardown() {
  rm -rf "${repo}" "${repo}.home" "${repo}.cache"
}

@test "select adds and removes to reach exactly the named set, keeps what the set needs, and says so" {
  "${kit_checkout}/bin/kit" install agents/claude agents/copilot
  [ -f .github/instructions/kit-bash.instructions.md ]
  run "${kit_checkout}/bin/kit" package select agents/cursor empowerite
  [ "$status" -eq 0 ]
  [ -f .cursor/rules/kit-bash.mdc ]
  [ ! -e .github/instructions/kit-bash.instructions.md ]
  [[ "$output" == *"agents/claude: kept, a member of empowerite"* ]]
  [[ "$output" == *"kit: kept, "*" depend(s) on it"* ]]
  run "${kit_checkout}/bin/kit" packages
  [ "$status" -eq 0 ]
  [[ "$output" == *"✅ agents/cursor"* && "$output" == *"✅ agents/claude"* && "$output" == *"✅ empowerite"* ]]
  [[ "$output" != *"✅ agents/copilot"* ]]
}

@test "select of the current set changes nothing and exits 0" {
  "${kit_checkout}/bin/kit" install agents/claude
  before=$(cat kit.yml .kit/manifest)
  run "${kit_checkout}/bin/kit" package select kit agents/claude
  [ "$status" -eq 0 ]
  [[ "$output" == *"packages: kit agents/generic agents/claude"* ]]
  [ "$(cat kit.yml .kit/manifest)" = "${before}" ]
}

@test "select refuses a package that is not in reach before it changes anything, and a glob names its matches" {
  "${kit_checkout}/bin/kit" install
  before=$(cat kit.yml)
  run "${kit_checkout}/bin/kit" package select 'style/*' nonesuch
  [ "$status" -eq 2 ]
  [ "$(cat kit.yml)" = "${before}" ]
  run "${kit_checkout}/bin/kit" package select 'style/*'
  [ "$status" -eq 0 ]
  run "${kit_checkout}/bin/kit" packages
  [[ "$output" == *"✅ style/bash"* && "$output" == *"✅ style/yaml"* ]]
}

@test "add and remove still work, from the module" {
  "${kit_checkout}/bin/kit" install
  run "${kit_checkout}/bin/kit" package add agents/cursor
  [ "$status" -eq 0 ]
  [ -f .cursor/rules/kit-bash.mdc ]
  run "${kit_checkout}/bin/kit" package remove agents/cursor
  [ "$status" -eq 0 ]
  [ ! -e .cursor/rules/kit-bash.mdc ]
}

@test "select keeps a member removed earlier out of its bundle unless the member is named" {
  "${kit_checkout}/bin/kit" install empowerite
  "${kit_checkout}/bin/kit" package remove hooks/lefthook
  [ ! -e .kit/lefthook.yml ]
  run "${kit_checkout}/bin/kit" package select empowerite agents/cursor
  [ "$status" -eq 0 ]
  [ ! -e .kit/lefthook.yml ]
  grep -q 'hooks/lefthook' kit.yml
  run "${kit_checkout}/bin/kit" package select empowerite hooks/lefthook
  [ "$status" -eq 0 ]
  [ -e .kit/lefthook.yml ]
}

@test "kit packages lists the packages, filtered by state and by name or glob, or as JSON, and kit package wants a change" {
  "${kit_checkout}/bin/kit" install agents/claude >/dev/null 2>&1
  run "${kit_checkout}/bin/kit" packages --active
  [ "$(awk '{ print $2 }' <<<"$output" | paste -s -d ' ' -)" = "agents/claude agents/generic kit" ]
  run "${kit_checkout}/bin/kit" packages --inactive 'agents/*'
  [ "$(awk '{ print $1 }' <<<"$output" | paste -s -d ' ' -)" = "agents/copilot agents/cursor agents/gemini" ]
  run "${kit_checkout}/bin/kit" packages --json kit style/bash
  [ "$(jq -sc 'map({name, state, origin})' <<<"$output")" = \
    '[{"name":"kit","state":"active","origin":"release"},{"name":"style/bash","state":"inactive","origin":"release"}]' ]
  run "${kit_checkout}/bin/kit" package
  [ "$status" -eq 2 ]
  run "${kit_checkout}/bin/kit" package list
  [ "$status" -eq 2 ]
}

@test "kit new kind writes a kind into the repository's own package, which kit kinds then lists and kit kind applies" {
  "${kit_checkout}/bin/kit" install >/dev/null 2>&1
  run "${kit_checkout}/bin/kit" new kind toml --path='**/*.toml' --embeds=bash
  [ "$status" -eq 0 ]
  [[ "$output" == *"run: kit package add local"* ]]
  [ -f .kit/packages/local/package.yml ]
  [ "$(sed -n 3p .kit/packages/local/files/repo/.kit/kinds/toml.yml)" = "name: toml" ]
  "${kit_checkout}/bin/kit" package add local >/dev/null 2>&1
  run "${kit_checkout}/bin/kit" kind Cargo.toml
  [ "$output" = $'any\ntoml' ]
  run "${kit_checkout}/bin/kit" new kind conf --kinds=toml --kinds=json
  [ "$status" -eq 0 ]
  [[ "$output" == *"the conf kind is in place"* ]]
  run "${kit_checkout}/bin/kit" kinds --json conf
  [ "$(jq -c '{file, kinds}' <<<"$output")" = '{"file":".kit/kinds/conf.yml","kinds":["toml","json"]}' ]
  run "${kit_checkout}/bin/kit" kind Cargo.toml
  [ "$output" = $'any\nconf\ntoml' ]
  run "${kit_checkout}/bin/kit" new kind toml --path='*.tml'
  [ "$status" -eq 1 ]
  [[ "$output" == *"the kind toml is defined already"* ]]
  run "${kit_checkout}/bin/kit" new kind Bad --path='*.x'
  [ "$status" -eq 2 ]
  run "${kit_checkout}/bin/kit" new kind empty
  [ "$status" -eq 2 ]
  [[ "$output" == *"needs --path, --content, or --kinds"* ]]
  run "${kit_checkout}/bin/kit" kinds
  [[ "$output" == *"toml "*".kit/kinds/toml.yml"*"**/*.toml · embeds bash"* ]]
}

# ----------------------------------------------------------------------------------------------------------------------
