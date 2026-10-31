#!/usr/bin/env bats
# just.bats - a justfile made from the just style's template loads the kit's loader in every bash recipe
# ----------------------------------------------------------------------------------------------------------------------
#
# The template exports BASH_ENV as kit root's loader, so a [script] recipe calls use, the output verbs, and run kit
# <verb> as a verb does, and finds the repository's own modules, from the root justfile and from a module justfile in
# a directory of its own.

bats_require_minimum_version 1.5.0

setup() {
  command -v just >/dev/null || skip "just is not on PATH"
  export kit_checkout NO_COLOR=1
  kit_checkout=$(cd "${BATS_TEST_DIRNAME}/../kit" && pwd)
  export KIT_HOME="${BATS_TEST_TMPDIR}/home" XDG_CACHE_HOME="${BATS_TEST_TMPDIR}/cache"
  export PATH="${kit_checkout}/bin:${PATH}"
  mkdir -p "${KIT_HOME}" "${XDG_CACHE_HOME}"
  repo="${BATS_TEST_TMPDIR}/repo"
  git init --quiet --initial-branch=main "${repo}"
  cd "${repo}"
  kit install style/just >/dev/null 2>&1
  printf 'bash_path:\n  - kit:modules/bash/\n  - bash/\n' >> kit.yml
  mkdir bash
  printf 'local:greet() { info "hello from $1"; }\n' > bash/local.sh
}

@test "a recipe of a justfile made from the template calls use, the output verbs, and run kit" {
  kit new file justfile >/dev/null 2>&1
  grep --quiet --line-regexp 'export BASH_ENV := `kit root` / "modules/bash/1st.sh"' justfile
  printf '%s\n' '' '[script]' 'greet:' '  set -o errexit -o nounset -o pipefail' '  use local' '  local:greet root' \
    '  run kit version' >> justfile
  run just greet
  [ "$status" -eq 0 ]
  [[ "$output" == *"• hello from root"* ]]
  [[ "$output" == *"unreleased"* ]]
  run kit lint justfile
  [ "$status" -eq 0 ]
}

@test "a recipe of a module justfile in its own directory finds the repository's modules" {
  kit new file justfile >/dev/null 2>&1
  printf '\nmod tools "tools/tools.just"\n' >> justfile
  mkdir tools
  kit new file tools/tools.just >/dev/null 2>&1
  printf '%s\n' '' '[script]' 'greet:' '  set -o errexit -o nounset -o pipefail' '  use local' \
    '  local:greet "${PWD##*/}"' >> tools/tools.just
  run just tools greet
  [ "$status" -eq 0 ]
  [[ "$output" == *"• hello from tools"* ]]
}

# ----------------------------------------------------------------------------------------------------------------------
