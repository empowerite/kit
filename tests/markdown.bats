#!/usr/bin/env bats
# markdown.bats - markdown:unlink: a link to kittui is its label, a web link its label and address, in the text pages
# ----------------------------------------------------------------------------------------------------------------------
# The pages kittui draws links on are also printed as text, by kit welcome --text and kit tutorials <id>, which must
# not show the markdown only kittui follows, however the page is laid out.

bats_require_minimum_version 1.5.0

load helper

unlink() {
  bash -c 'source "$1/modules/bash/1st.sh"; use markdown; markdown:unlink' _ "${kit_root}" <<<"$1"
}

@test "a link to kittui is its label alone, with no address" {
  run unlink 'See [pull request](kittui:welcome/terms/pull%20request) and [Flow](kittui:flow) or [the menu bar](kittui:tutorials/menu-bar).'
  [ "$status" -eq 0 ]
  [ "${output}" = 'See pull request and Flow or the menu bar.' ]
}

@test "a web link is its label and its address once" {
  run unlink 'Read [the docs](https://example.com/docs/x.md), then [more](http://example.com).'
  [ "${output}" = 'Read the docs (https://example.com/docs/x.md), then more (http://example.com).' ]
}

@test "any other link, code, and text without links are left as written" {
  run unlink 'Edit [kit.yml](kit.yml), run `kit flow`; a [bracket] (paren) stays.'
  [ "${output}" = 'Edit [kit.yml](kit.yml), run `kit flow`; a [bracket] (paren) stays.' ]
}

@test "no page kit ships, printed as text, shows a kittui link" {
  cd "${BATS_TEST_TMPDIR}"
  for page in "${kit_root}"/modules/tutorials/*.md "${kit_root}"/packages/kit/files/repo/.kit/welcome/*.md
  do
    run bash -c 'source "$1/modules/bash/1st.sh"; use tutorials welcome; if [[ $2 == */tutorials/* ]]; then tutorials:page "$2"; else welcome:page "$2"; fi' _ "${kit_root}" "${page}"
    [ "$status" -eq 0 ]
    ! grep -q 'kittui:' <<<"${output}"
    ! grep -Eq '\]\(https?://' <<<"${output}"
  done
}
