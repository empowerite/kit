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
  run "${kit_checkout}/bin/kit" package list
  [ "$status" -eq 0 ]
  [[ "$output" == *"✅ agents/cursor"* && "$output" == *"✅ agents/claude"* && "$output" == *"✅ empowerite"* ]]
  [[ "$output" != *"✅ agents/copilot"* ]]
}

@test "select of the current set changes nothing and exits 0" {
  "${kit_checkout}/bin/kit" install agents/claude
  before=$(cat .kit/settings.yml .kit/manifest)
  run "${kit_checkout}/bin/kit" package select kit agents/claude
  [ "$status" -eq 0 ]
  [[ "$output" == *"packages: kit agents/claude"* ]]
  [ "$(cat .kit/settings.yml .kit/manifest)" = "${before}" ]
}

@test "select refuses a package that is not in reach before it changes anything, and a glob names its matches" {
  "${kit_checkout}/bin/kit" install
  before=$(cat .kit/settings.yml)
  run "${kit_checkout}/bin/kit" package select 'style/*' nonesuch
  [ "$status" -eq 2 ]
  [ "$(cat .kit/settings.yml)" = "${before}" ]
  run "${kit_checkout}/bin/kit" package select 'style/*'
  [ "$status" -eq 0 ]
  run "${kit_checkout}/bin/kit" package list
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
  grep -q 'hooks/lefthook' .kit/settings.yml
  run "${kit_checkout}/bin/kit" package select empowerite hooks/lefthook
  [ "$status" -eq 0 ]
  [ -e .kit/lefthook.yml ]
}

# ----------------------------------------------------------------------------------------------------------------------
