#!/usr/bin/env bats
# write.bats - what kit writes in the repository is written whole or not at all, and around the repository's own lines
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

@test "install refuses a settings file jq cannot read, names it, and leaves it byte for byte as it was" {
  mkdir -p .claude
  printf '{\n  // Claude Code reads this comment; jq does not\n  "model": "x"\n}\n' > .claude/settings.json
  before=$(cat .claude/settings.json)
  run "${kit_checkout}/bin/kit" install agents/claude
  [ "$status" -eq 1 ]
  [[ "$output" == *".claude/settings.json: not JSON jq can read"* ]]
  [ "$(cat .claude/settings.json)" = "${before}" ]
  [ ! -e .claude/settings.json.kit-new ]
  # whole or not at all: no manifest, no declaration
  [ ! -e .kit/manifest ]
  [ ! -e kit.yml ]
}

@test "package add refuses a settings file jq cannot read the same way, and the active set stays as it was" {
  "${kit_checkout}/bin/kit" install
  mkdir -p .gemini
  printf '{"ui": {"theme": "Dark"},}\n' > .gemini/settings.json
  before=$(cat .gemini/settings.json)
  run "${kit_checkout}/bin/kit" package add agents/gemini
  [ "$status" -eq 1 ]
  [[ "$output" == *".gemini/settings.json: not JSON jq can read"* ]]
  [ "$(cat .gemini/settings.json)" = "${before}" ]
  run "${kit_checkout}/bin/kit" packages
  [[ "$output" != *"✅ agents/gemini"* ]]
}

@test "the declaration keeps every item after a comment or a blank line inside its block, and the repository's comments after it" {
  mkdir .kit
  printf 'version: unreleased\npackages:\n  - kit\n  - style/bash\n# markdown is next\n  - style/markdown\n\n  - style/json\nexcludes:\n  - style/yaml\n\n# the gate\ngate: bin/ci\n' > kit.yml
  run "${kit_checkout}/bin/kit" package add hooks/lefthook
  [ "$status" -eq 0 ]
  expected='version: unreleased|packages:|  - kit|  - hooks/lefthook|  - style/bash|  - style/json|  - style/markdown|excludes:|  - style/yaml||# the gate|gate: bin/ci'
  [ "$(paste -s -d '|' - < kit.yml)" = "${expected}" ]
}
