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
  [ "$(printf '%s\n' "${output}" | yq -o json -I 0 '.')" = '{"audiences":[{"id":"newcomer","label":"Just starting"},{"id":"ui","label":"UI"},{"id":"maintainer","label":"Maintainer"}],"tutorials":[{"id":"beta","path":"beta/","title":"Beta","summary":"The first one.","glyph":"🅱","audience":["maintainer"],"order":10},{"id":"alpha","path":"alpha/","title":"Alpha","summary":"The second one.","glyph":"🅰","audience":["newcomer","ui"],"order":20}]}' ]
  [[ "${output}" != *Deeper* ]]
}

@test "an unknown audience is refused, naming the file" {
  catalogue unknown-audience
  [ "$status" -eq 1 ]
  [[ "${stderr}" == *"gamma/index.mdx"* ]]
  [[ "${stderr}" == *"wizard"* ]]
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
  before=$(cat kit/modules/tutorials.yml)
  run "${kit_root}/bin/kit" generate --check kit/modules/tutorials.yml
  [ "$status" -eq 0 ]
  [ "$(cat kit/modules/tutorials.yml)" = "${before}" ]
  run "${kit_root}/bin/kit" generate kit/modules/tutorials.yml
  [ "$status" -eq 0 ]
  [ "$(cat kit/modules/tutorials.yml)" = "${before}" ]
}
