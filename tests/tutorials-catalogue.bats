#!/usr/bin/env bats
# tutorials-catalogue.bats - bin/tutorials catalogue lists the top-level tutorials from their front matter, or refuses
# ----------------------------------------------------------------------------------------------------------------------

bats_require_minimum_version 1.5.0

load helper

repo_root=$(cd "${BATS_TEST_DIRNAME}/.." && pwd)

# catalogue <fixture> runs bin/tutorials catalogue over a fixture project under tests/fixtures/tutorials/.
catalogue() {
  TUTORIALS_PROJECT="${repo_root}/tests/fixtures/tutorials/$1" run --separate-stderr "${repo_root}/bin/tutorials" catalogue
}

@test "the catalogue lists the top-level tutorials in order" {
  catalogue ok
  [ "$status" -eq 0 ]
  [ "$(printf '%s\n' "${output}" | yq -o json -I 0 '.')" = '{"audiences":[{"id":"newcomer","label":"Just starting","color":"#15803d"},{"id":"ui","label":"UI","color":"#1d4ed8"},{"id":"maintainer","label":"Maintainer","color":"#7c3aed"}],"tutorials":[{"id":"beta","path":"beta/","title":"Beta","summary":"The first one.","glyph":"🅱","audience":["maintainer"],"order":10},{"id":"alpha","path":"alpha/","title":"Alpha","summary":"The second one.","glyph":"🅰","audience":["newcomer","ui"],"order":20}]}' ]
  [[ "${output}" != *Deeper* ]]
}

@test "an unknown audience is refused, naming the file" {
  catalogue unknown-audience
  [ "$status" -eq 1 ]
  [[ "${stderr}" == *"gamma/index.mdx"* ]]
  [[ "${stderr}" == *"wizard"* ]]
}

@test "an audience needs a color" {
  catalogue no-color
  [ "$status" -eq 1 ]
  [ "$output" = "" ]
  local file="${repo_root}/tests/fixtures/tutorials/no-color/audiences.yml"
  [ "${stderr}" = "❌ ${file}: the audience 'ui' needs a color, as #rrggbb" ]
}

@test "a color is #rrggbb" {
  catalogue bad-color
  [ "$status" -eq 1 ]
  local file="${repo_root}/tests/fixtures/tutorials/bad-color/audiences.yml"
  [ "${stderr}" = "❌ ${file}: the audience 'ui' has the color '#1d4ed', not #rrggbb" ]
  # each color is set as a YAML value, in a copy of the ok project
  local color
  for color in '"blue"' '"#1d4ed8f"' '"1d4ed8"' '"#gggggg"' '"#1d4ed8 "' 12 '[]' '"#1D4ED8"'
  do
    rm -rf "${BATS_TEST_TMPDIR}/project"
    cp -R "${repo_root}/tests/fixtures/tutorials/ok" "${BATS_TEST_TMPDIR}/project"
    yq -i ".[1].color = ${color}" "${BATS_TEST_TMPDIR}/project/audiences.yml"
    TUTORIALS_PROJECT="${BATS_TEST_TMPDIR}/project" run --separate-stderr "${repo_root}/bin/tutorials" catalogue
    if [[ "${color}" == '"#1D4ED8"' ]]
    then
      [ "$status" -eq 0 ]
    else
      [ "$status" -eq 1 ]
      [[ "${stderr}" == *"/project/audiences.yml: the audience 'ui' has the color "*", not #rrggbb" ]]
    fi
  done
}

@test "a top-level page needs a summary" {
  catalogue no-summary
  [ "$status" -eq 1 ]
  [[ "${stderr}" == *"gamma/index.mdx"* ]]
  [[ "${stderr}" == *"summary"* ]]
}

@test "a top-level page needs a glyph" {
  catalogue no-glyph
  [ "$status" -eq 1 ]
  [[ "${stderr}" == *"gamma/index.mdx"* ]]
  [[ "${stderr}" == *"glyph"* ]]
}

@test "a top-level page lives at <id>/index.mdx" {
  catalogue misplaced
  [ "$status" -eq 1 ]
  [[ "${stderr}" == *"a/b.mdx"* ]]
}

@test "the catalogue in kit is current" {
  # A test ends outside the checkout, so the kit runs in a subshell that goes there.
  run bash -c 'cd "$1" && "$2/bin/kit" generate --check kit/modules/tutorials.yml' _ "${repo_root}" "${kit_checkout}"
  [ "$status" -eq 0 ]
}

@test "where kit's source is not, the catalogue's region reprints as it stands, so generate --check holds" {
  fresh-repo >/dev/null
  cp -R "${kit_checkout}" kit
  [ ! -e bin/tutorials ]
  cp kit/modules/tutorials.yml "${BATS_TEST_TMPDIR}/before"
  run "${kit_root}/bin/kit" generate --check kit/modules/tutorials.yml
  [ "$status" -eq 0 ]
  cmp kit/modules/tutorials.yml "${BATS_TEST_TMPDIR}/before"
  run "${kit_root}/bin/kit" generate kit/modules/tutorials.yml
  [ "$status" -eq 0 ]
  cmp kit/modules/tutorials.yml "${BATS_TEST_TMPDIR}/before"
}

@test "a repository's own bin/tutorials is never run by the region, but kit's source fails loudly without its execute bit" {
  fresh-repo >/dev/null
  cp -R "${kit_checkout}" kit
  mkdir bin
  printf '#!/bin/sh\ntouch ran\n' > bin/tutorials
  chmod +x bin/tutorials
  cp kit/modules/tutorials.yml "${BATS_TEST_TMPDIR}/before"
  run "${kit_root}/bin/kit" generate kit/modules/tutorials.yml
  [ "$status" -eq 0 ]
  [ ! -e ran ]
  cmp kit/modules/tutorials.yml "${BATS_TEST_TMPDIR}/before"
  # with a projects/tutorials, as in kit's source, the command runs bin/tutorials, and a lost execute bit fails
  mkdir -p projects/tutorials
  chmod -x bin/tutorials
  run "${kit_root}/bin/kit" generate kit/modules/tutorials.yml
  [ "$status" -ne 0 ]
}
