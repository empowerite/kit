#!/usr/bin/env bats
# super.bats - a module overrides a function another defined, and calls the one it replaced
# ----------------------------------------------------------------------------------------------------------------------

setup() {
  export kit_root NO_COLOR=1
  kit_root=$(cd "${BATS_TEST_DIRNAME}/.." && pwd)
  repo=$(mktemp -d)
  cd "${repo}"
  git init --quiet --initial-branch=main
  mkdir -p .kit a b c
  printf 'bash_path:\n  - kit:bash/\n  - a/\n  - b/\n  - c/\n' > .kit/settings.yml
  printf 'foo:bar() { echo "A($*)"; }\nfoo:baz() { echo "baz $(foo:bar x)"; }\n' > a/foo.sh
  printf 'use super\nsuper:override foo:bar\nfoo:bar() { echo "B[$(super "$@")]"; }\n' > b/foo.sh
  printf 'use super\nsuper:override foo:bar\nfoo:bar() { echo "C{$(super "$@")}"; }\n' > c/foo.sh
}

teardown() {
  rm -rf "${repo}"
}

# kit-bash <code> runs <code> in a fresh bash that has sourced the loader.
kit-bash() {
  bash -c 'source "${kit_root}/bash/1st.sh" && eval "$1"' kit-bash "$1"
}

@test "each override reaches the one it replaced, and every caller gets the last" {
  run kit-bash 'use foo && foo:bar 1 && foo:baz'
  [ "$status" -eq 0 ]
  [ "$output" = $'C{B[A(1)]}\nbaz C{B[A(x)]}' ]
}

@test "the origin of each replaced definition is kept, and the live one is not recorded" {
  run kit-bash 'use foo && printf "%s=%s\n" "foo:bar^^" "${super_origin[foo:bar^^]}" "foo:bar^" "${super_origin[foo:bar^]}" && [[ -z ${super_origin[foo:bar]+x} ]]'
  [ "$status" -eq 0 ]
  [ "$output" = "foo:bar^^=1 $(pwd -P)/a/foo.sh"$'\n'"foo:bar^=3 $(pwd -P)/b/foo.sh" ]
}

@test "extdebug is left as it was" {
  run kit-bash 'use foo && shopt -q extdebug && echo on || echo off'
  [ "$output" = off ]
  run kit-bash 'shopt -s extdebug && use foo && shopt -q extdebug && echo on || echo off'
  [ "$output" = on ]
}

@test "overriding a function no module defined is refused" {
  run kit-bash 'use super && super:override nope:nothing'
  [ "$status" -eq 1 ]
  [[ "$output" == *"super:override: nope:nothing is not defined"* ]]
}

# ----------------------------------------------------------------------------------------------------------------------
