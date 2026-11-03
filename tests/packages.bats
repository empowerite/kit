#!/usr/bin/env bats
# packages.bats - kit packages lists the packages, and select, add, and remove change which are active
# ----------------------------------------------------------------------------------------------------------------------
# Each test gets an empty git repository under a temporary directory and runs the kit from this checkout against it.

bats_require_minimum_version 1.5.0

load helper

setup() {
  fresh-repo
}

@test "select adds and removes to reach exactly the named set, keeps what the set needs, and says so" {
  "${kit_checkout}/bin/kit" install agents/claude agents/copilot
  [ -f .github/instructions/kit-bash.instructions.md ]
  run "${kit_checkout}/bin/kit" packages select agents/cursor empowerite
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
  run "${kit_checkout}/bin/kit" packages select kit agents/claude
  [ "$status" -eq 0 ]
  [[ "$output" == *"packages: kit agents/generic agents/claude"* ]]
  [ "$(cat kit.yml .kit/manifest)" = "${before}" ]
}

@test "select refuses a package that is not in reach before it changes anything, and a glob names its matches" {
  "${kit_checkout}/bin/kit" install
  before=$(cat kit.yml)
  run "${kit_checkout}/bin/kit" packages select 'style/*' nonesuch
  [ "$status" -eq 2 ]
  [ "$(cat kit.yml)" = "${before}" ]
  run "${kit_checkout}/bin/kit" packages select 'style/*'
  [ "$status" -eq 0 ]
  run "${kit_checkout}/bin/kit" packages
  [[ "$output" == *"✅ style/bash"* && "$output" == *"✅ style/yaml"* ]]
}

@test "add and remove still work, from the module" {
  "${kit_checkout}/bin/kit" install
  run "${kit_checkout}/bin/kit" packages add agents/cursor
  [ "$status" -eq 0 ]
  [ -f .cursor/rules/kit-bash.mdc ]
  run "${kit_checkout}/bin/kit" packages remove agents/cursor
  [ "$status" -eq 0 ]
  [ ! -e .cursor/rules/kit-bash.mdc ]
}

@test "select keeps a member removed earlier out of its bundle unless the member is named" {
  "${kit_checkout}/bin/kit" install empowerite
  "${kit_checkout}/bin/kit" packages remove hooks/lefthook
  [ ! -e .kit/lefthook.yml ]
  run "${kit_checkout}/bin/kit" packages select empowerite agents/cursor
  [ "$status" -eq 0 ]
  [ ! -e .kit/lefthook.yml ]
  grep -q 'hooks/lefthook' kit.yml
  run "${kit_checkout}/bin/kit" packages select empowerite hooks/lefthook
  [ "$status" -eq 0 ]
  [ -e .kit/lefthook.yml ]
}

@test "kit packages lists the packages, filtered by state and by name or glob, or as JSON, and a verb wants a package" {
  "${kit_checkout}/bin/kit" install agents/claude >/dev/null 2>&1
  run "${kit_checkout}/bin/kit" packages --active
  [ "$(awk '{ print $2 }' <<<"$output" | paste -s -d ' ' -)" = "agents/claude agents/generic kit" ]
  run "${kit_checkout}/bin/kit" packages --inactive 'agents/*'
  [ "$(awk '{ print $1 }' <<<"$output" | paste -s -d ' ' -)" = "agents/copilot agents/cursor agents/gemini" ]
  run "${kit_checkout}/bin/kit" packages --json kit style/bash
  [ "$(jq -sc 'map({name, state, origin})' <<<"$output")" = \
    '[{"name":"kit","state":"active","origin":"release"},{"name":"style/bash","state":"inactive","origin":"release"}]' ]
  run "${kit_checkout}/bin/kit" packages add
  [ "$status" -eq 2 ]
  [[ "$output" == *"kit packages add needs <package>; run: kit help packages"* ]]
  # a word that is no verb of kit packages is a package name, and one the kit does not know is refused
  run "${kit_checkout}/bin/kit" packages list
  [ "$status" -eq 2 ]
  [[ "$output" == *"<package> 'list' is not one of the choices; run: kit help packages"* ]]
}

@test "kit new kind writes a kind into the repository's own package, which kit kinds then lists and applies to a path" {
  "${kit_checkout}/bin/kit" install >/dev/null 2>&1
  run "${kit_checkout}/bin/kit" new kind toml --path='**/*.toml' --embeds=bash
  [ "$status" -eq 0 ]
  [[ "$output" == *"run: kit packages add local"* ]]
  [ -f .kit/packages/local/package.yml ]
  [ "$(sed -n 3p .kit/packages/local/files/repo/.kit/kinds/toml.yml)" = "name: toml" ]
  "${kit_checkout}/bin/kit" packages add local >/dev/null 2>&1
  run "${kit_checkout}/bin/kit" kinds Cargo.toml
  [ "$output" = $'any\ntoml' ]
  run "${kit_checkout}/bin/kit" new kind conf --kinds=toml --kinds=json
  [ "$status" -eq 0 ]
  [[ "$output" == *"the conf kind is in place"* ]]
  run "${kit_checkout}/bin/kit" kinds --json conf
  [ "$(jq -c '{file, kinds}' <<<"$output")" = '{"file":".kit/kinds/conf.yml","kinds":["toml","json"]}' ]
  run "${kit_checkout}/bin/kit" kinds Cargo.toml
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
