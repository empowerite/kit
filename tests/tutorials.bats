#!/usr/bin/env bats
# tutorials.bats - kit tutorials: the base per version, the JSON, the audience filter, opening one, the hand-off
# ----------------------------------------------------------------------------------------------------------------------
# Each test runs in a fresh repository, with KIT_TUTORIALS_BASE unset unless it names it. The catalogue is a fixture
# the test points KIT_TUTORIALS_CATALOGUE at; the version comes from a copy of the kit with a version file, since a
# checkout has none. A browser is a fake open and xdg-open, first on PATH, that record the URL they are given.

bats_require_minimum_version 1.5.0

load helper

setup() {
  hide-system-toolchains
  fresh-repo >/dev/null
  unset KIT_TUTORIALS_BASE KIT_TUTORIALS_CATALOGUE
  catalogue="${BATS_TEST_TMPDIR}/tutorials.yml"
  cat > "${catalogue}" <<'YML'
audiences:
  - id: newcomer
    label: Just starting
  - id: ui
    label: UI
  - id: maintainer
    label: Maintainer
tutorials:
  - id: first
    path: first/
    title: The first tutorial
    summary: Where to begin.
    glyph: "1"
    audience:
      - newcomer
    order: 1
  - id: screens
    path: screens/
    title: The screens
    summary: What kittui shows.
    glyph: "2"
    audience:
      - ui
      - newcomer
    order: 2
  - id: releasing
    path: releasing/
    title: Cutting a release
    summary: From tag to tarball.
    glyph: "3"
    audience:
      - maintainer
    order: 3
YML
  export KIT_TUTORIALS_CATALOGUE="${catalogue}"
}

# versioned <version> makes a copy of the kit with a version file, and sets kit to it.
versioned() {
  cp -R "${kit_root}" "${BATS_TEST_TMPDIR}/kit-copy"
  printf '%s\n' "$1" > "${BATS_TEST_TMPDIR}/kit-copy/version"
  kit="${BATS_TEST_TMPDIR}/kit-copy/bin/kit"
}

# fake-openers writes an open and an xdg-open that record the URL they are given in opened, first on PATH.
fake-openers() {
  local opener
  mkdir -p "${BATS_TEST_TMPDIR}/openers"
  for opener in open xdg-open
  do
    printf '#!/usr/bin/env bash\nprintf "%%s\\n" "$1" >> "%s/opened"\n' "${BATS_TEST_TMPDIR}" \
      > "${BATS_TEST_TMPDIR}/openers/${opener}"
    chmod +x "${BATS_TEST_TMPDIR}/openers/${opener}"
  done
  export PATH="${BATS_TEST_TMPDIR}/openers:${PATH}"
}

@test "the base follows the version" {
  run bash -c "source '${kit_root}/modules/bash/1st.sh'; use tutorials
    for v in unreleased 0.1.0-rc.7 0.4.12 1.0.0; do tutorials:base \"\${v}\"; done"
  [ "$status" -eq 0 ]
  [ "$output" = "https://empowerite.github.io/kit/tutorials/main/
https://empowerite.github.io/kit/tutorials/0.1.0-rc.7/
https://kit.empowerite.dev/tutorials/0.4/
https://kit.empowerite.dev/tutorials/1.0/" ]
}

@test "--json carries each tutorial's link" {
  versioned 0.4.7
  run --separate-stderr "${kit}" tutorials --json
  [ "$status" -eq 0 ]
  [ "$(jq --raw-output '.version, .base' <<<"$output")" = "0.4.7
https://kit.empowerite.dev/tutorials/0.4/" ]
  [ "$(jq --raw-output '.tutorials[].url' <<<"$output")" = "https://kit.empowerite.dev/tutorials/0.4/first/?kit=0.4.7
https://kit.empowerite.dev/tutorials/0.4/screens/?kit=0.4.7
https://kit.empowerite.dev/tutorials/0.4/releasing/?kit=0.4.7" ]
  [ "$(jq --raw-output '.audiences[].id' <<<"$output" | paste -s -d ' ' -)" = "newcomer ui maintainer" ]
  [ "$(jq --raw-output '.tutorials[0] | keys | join(",")' <<<"$output")" = "audience,glyph,id,summary,title,url" ]
  [ "$(wc -l <<<"$output")" -eq 1 ]
}

@test "KIT_TUTORIALS_BASE replaces the base" {
  versioned 0.4.7
  export KIT_TUTORIALS_BASE=http://localhost:4321/tutorials/
  run --separate-stderr "${kit}" tutorials --json
  [ "$status" -eq 0 ]
  [ "$(jq --raw-output '.base' <<<"$output")" = "http://localhost:4321/tutorials/" ]
  [ "$(jq --raw-output '.tutorials[0].url' <<<"$output")" = "http://localhost:4321/tutorials/first/?kit=0.4.7" ]
}

@test "--tag keeps the tutorials for that audience, and two --tags keep either's" {
  run --separate-stderr "${kit_root}/bin/kit" tutorials --json --tag ui
  [ "$status" -eq 0 ]
  [ "$(jq --raw-output '.tutorials[].id' <<<"$output")" = "screens" ]
  # the audiences stay whole: the screen shows every filter, whichever are on
  [ "$(jq '.audiences | length' <<<"$output")" -eq 3 ]
  run --separate-stderr "${kit_root}/bin/kit" tutorials --json --tag ui --tag maintainer
  [ "$(jq --raw-output '.tutorials[].id' <<<"$output" | paste -s -d ' ' -)" = "screens releasing" ]
}

@test "an unknown tutorial or audience is a usage error" {
  run --separate-stderr "${kit_root}/bin/kit" tutorials nonesuch
  [ "$status" -eq 2 ]
  run --separate-stderr "${kit_root}/bin/kit" tutorials --json --tag nonesuch
  [ "$status" -eq 2 ]
}

@test "a tutorial named opens in the browser" {
  fake-openers
  versioned 0.4.7
  run --separate-stderr "${kit}" tutorials screens
  [ "$status" -eq 0 ]
  [[ "$output" == *"The screens"* ]]
  [[ "$output" == *"https://kit.empowerite.dev/tutorials/0.4/screens/?kit=0.4.7"* ]]
  [ "$(cat "${BATS_TEST_TMPDIR}/opened")" = "https://kit.empowerite.dev/tutorials/0.4/screens/?kit=0.4.7" ]
}

@test "without an opener, the link prints" {
  # a uname that says Linux makes xdg-open the opener, which a machine without one lacks
  command -v xdg-open >/dev/null && skip "this machine has xdg-open"
  versioned 0.4.7
  mkdir -p "${BATS_TEST_TMPDIR}/linux"
  printf '#!/usr/bin/env bash\necho Linux\n' > "${BATS_TEST_TMPDIR}/linux/uname"
  chmod +x "${BATS_TEST_TMPDIR}/linux/uname"
  PATH="${BATS_TEST_TMPDIR}/linux:${PATH}" run --separate-stderr "${kit}" tutorials first
  [ "$status" -eq 0 ]
  [[ "$output" == *"https://kit.empowerite.dev/tutorials/0.4/first/?kit=0.4.7"* ]]
  [[ "$stderr" == *"open the URL above"* ]]
}

@test "the text lists glyph, title, summary, audiences, and link" {
  versioned 0.4.7
  run --separate-stderr "${kit}" tutorials --text
  [ "$status" -eq 0 ]
  in-order "$output" '2' 'The screens' 'What kittui shows.' 'ui' 'newcomer' \
    'https://kit.empowerite.dev/tutorials/0.4/screens/?kit=0.4.7' '3' 'Cutting a release'
  [ "$(grep -c 'kit=0.4.7' <<<"$output")" -eq 3 ]
}

@test "no tutorials" {
  printf 'audiences: []\ntutorials: []\n' > "${catalogue}"
  run --separate-stderr "${kit_root}/bin/kit" tutorials --text
  [ "$status" -eq 0 ]
  [ -z "$output" ]
  [ "$(wc -l <<<"$stderr")" -eq 1 ]
  [[ "$stderr" == *"no tutorials"* ]]
}

@test "the choices list the tutorials and the audiences, closed" {
  run --separate-stderr "${kit_root}/bin/kit" syntax tutorials --choices '<tutorial>'
  [ "$status" -eq 0 ]
  [[ "$output" == *"first"* && "$output" == *"Where to begin."* ]] || [[ "$output" == *"The first tutorial"* ]]
  run --separate-stderr "${kit_root}/bin/kit" syntax tutorials --choices '<audience>'
  [[ "$output" == *"newcomer"* && "$output" == *"Just starting"* ]]
}

@test "on a terminal, it opens kittui's screen" {
  mkdir -p "${BATS_TEST_TMPDIR}/fake"
  ln -s "${BATS_TEST_DIRNAME}/fakes/fake-kittui" "${BATS_TEST_TMPDIR}/fake/kittui"
  ln -s "${BATS_TEST_DIRNAME}/fakes/fake-go" "${BATS_TEST_TMPDIR}/fake/go"
  export FAKES="${BATS_TEST_TMPDIR}/fake" PATH="${BATS_TEST_TMPDIR}/fake:${PATH}"
  run pty "'${kit_root}/bin/kit' tutorials --tag newcomer --tag ui"
  [[ "$output" == *"kittui native-native tutorials --tag newcomer --tag ui"* ]]
  run pty "'${kit_root}/bin/kit' tutorials --text"
  [[ "$output" == *"The first tutorial"* && "$output" != *"native-native"* ]]
}
