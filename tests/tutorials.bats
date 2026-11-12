#!/usr/bin/env bats
# tutorials.bats - kit tutorials: the base per version, the JSON, the audience filter, opening one, the hand-off
# ----------------------------------------------------------------------------------------------------------------------
# Each test runs in a fresh repository, with KIT_TUTORIALS_BASE unset unless it names it. The catalogue is a fixture
# the test points KIT_TUTORIALS_CATALOGUE at, three tutorials online and one that ships beside it; the version comes from a copy of the kit with a version file, since a
# checkout has none. A browser is a fake open and xdg-open, first on PATH, that record the URL they are given.

bats_require_minimum_version 1.5.0

load helper

site=https://empowerite.github.io/kit/tutorials

setup() {
  hide-system-toolchains
  fresh-repo >/dev/null
  unset KIT_TUTORIALS_BASE KIT_TUTORIALS_CATALOGUE
  catalogue="${BATS_TEST_TMPDIR}/tutorials.yml"
  cat > "${catalogue}" <<'YML'
audiences:
  - id: newcomer
    label: Just starting
    color: "#15803d"
  - id: ui
    label: UI
    color: "#1d4ed8"
  - id: maintainer
    label: Maintainer
    color: "#7c3aed"
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
  - id: tour
    file: tutorials/tour.md
    title: The tour
    summary: Read here, not online.
    glyph: "4"
    audience:
      - newcomer
    order: 4
YML
  mkdir -p "${BATS_TEST_TMPDIR}/tutorials"
  printf -- '---\ntitle: The tour\n---\n\nThe first screen is Flow.\n' > "${BATS_TEST_TMPDIR}/tutorials/tour.md"
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

@test "the line and the base follow the version, on one site" {
  run bash -c "source '${kit_root}/modules/bash/1st.sh'; use tutorials
    for v in unreleased 0.1.0-rc.7 0.4.12; do tutorials:line \"\${v}\"; tutorials:base \"\${v}\"; done"
  [ "$status" -eq 0 ]
  [ "$output" = "$(printf '%s\n' main "${site}/main/" 0.1.0-rc.7 "${site}/0.1.0-rc.7/" 0.4 "${site}/0.4/")" ]
}

@test "a missing catalogue is 1, whatever is named" {
  local word
  for word in --text --json first
  do
    KIT_TUTORIALS_CATALOGUE=/nonesuch run --separate-stderr "${kit_root}/bin/kit" tutorials "${word}"
    [ "$status" -eq 1 ]
    [[ "$stderr" == *"no catalogue of tutorials at /nonesuch"* ]]
  done
}

@test "the grammar gives a named tutorial no --json" {
  [ "$("${kit_root}/bin/kit" syntax tutorials --match -- --json first | jq .status)" -eq 2 ]
  [ "$("${kit_root}/bin/kit" syntax tutorials --match -- --text --tag=ui first | jq .status)" -eq 0 ]
}

@test "--json gives an audience the catalogue gives no color an empty one" {
  grep -v '^    color: ' "${catalogue}" > "${BATS_TEST_TMPDIR}/colorless.yml"
  export KIT_TUTORIALS_CATALOGUE="${BATS_TEST_TMPDIR}/colorless.yml"
  run --separate-stderr "${kit_root}/bin/kit" tutorials --json
  [ "$status" -eq 0 ]
  [ "$(jq --compact-output '.audiences[0]' <<<"$output")" = '{"id":"newcomer","label":"Just starting","color":""}' ]
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
  run --separate-stderr "${kit_root}/bin/kit" tutorials --json --tag=ui
  [ "$status" -eq 0 ]
  [ "$(jq --raw-output '.tutorials[].id' <<<"$output")" = "screens" ]
  # the audiences stay whole: the screen shows every filter, whichever are on
  [ "$(jq '.audiences | length' <<<"$output")" -eq 3 ]
  run --separate-stderr "${kit_root}/bin/kit" tutorials --json --tag=ui --tag=maintainer
  [ "$(jq --raw-output '.tutorials[].id' <<<"$output" | paste -s -d ' ' -)" = "screens releasing" ]
  # the audiences may come joined by :, each one checked
  run --separate-stderr "${kit_root}/bin/kit" tutorials --json --tag=ui:maintainer
  [ "$(jq --raw-output '.tutorials[].id' <<<"$output" | paste -s -d ' ' -)" = "screens releasing" ]
  run --separate-stderr "${kit_root}/bin/kit" tutorials --json --tag=ui:nonesuch
  [ "$status" -eq 2 ]
}

@test "an unknown tutorial or audience is a usage error" {
  run --separate-stderr "${kit_root}/bin/kit" tutorials nonesuch
  [ "$status" -eq 2 ]
  run --separate-stderr "${kit_root}/bin/kit" tutorials --json --tag=nonesuch
  [ "$status" -eq 2 ]
}

@test "a tutorial named opens in the browser" {
  fake-openers
  versioned 0.4.7
  run --separate-stderr "${kit}" tutorials screens
  [ "$status" -eq 0 ]
  [[ "$output" == *"The screens"* ]]
  [[ "$output" == *"${site}/0.4/screens/?kit=0.4.7"* ]]
  [ "$(cat "${BATS_TEST_TMPDIR}/opened")" = "${site}/0.4/screens/?kit=0.4.7" ]
}

@test "without an opener, the link prints" {
  versioned 0.4.7
  # a uname that says Linux makes xdg-open the opener; every PATH directory that holds open or xdg-open is replaced by
  # links to everything else in it, so no machine has an opener here
  local dir out="${BATS_TEST_TMPDIR}/linux" n=0 copy
  mkdir -p "${out}"
  printf '#!/usr/bin/env bash\necho Linux\n' > "${out}/uname"
  chmod +x "${out}/uname"
  while IFS= read -r dir
  do
    [[ -d "${dir}" ]] || continue
    if [[ -e "${dir}/open" || -e "${dir}/xdg-open" ]]
    then
      copy="${BATS_TEST_TMPDIR}/noopener-$((n++))"
      mkdir -p "${copy}"
      ln -s "${dir}"/* "${copy}/" 2>/dev/null || true
      rm -f "${copy}/open" "${copy}/xdg-open"
      dir=${copy}
    fi
    out+=":${dir}"
  done < <(tr ':' '\n' <<<"${PATH}")
  PATH=${out} run --separate-stderr "${kit}" tutorials first
  [ "$status" -eq 0 ]
  [[ "$output" == *"${site}/0.4/first/?kit=0.4.7"* ]]
  [[ "$stderr" == *"open the URL above"* ]]
}

@test "an audience no tutorial has is one dim line, and exit 0" {
  grep -v '^      - maintainer$' "${catalogue}" | sed '/id: releasing/,$d' > "${BATS_TEST_TMPDIR}/none.yml"
  export KIT_TUTORIALS_CATALOGUE="${BATS_TEST_TMPDIR}/none.yml"
  run --separate-stderr "${kit_root}/bin/kit" tutorials --text --tag=maintainer
  [ "$status" -eq 0 ]
  [ -z "$output" ]
  [ "$(wc -l <<<"$stderr")" -eq 1 ]
  [[ "$stderr" == *"no tutorials"*"maintainer"* ]]
}

@test "the text lists glyph, title, summary, audiences, and link" {
  versioned 0.4.7
  run --separate-stderr "${kit}" tutorials --text
  [ "$status" -eq 0 ]
  in-order "$output" '2' 'The screens' 'What kittui shows.' 'ui' 'newcomer' \
    "${site}/0.4/screens/?kit=0.4.7" '3' 'Cutting a release'
  [ "$(grep -c 'kit=0.4.7' <<<"$output")" -eq 3 ]
  [[ "$output" != *"#15803d"* ]]
  # a tutorial that ships with the kit says so, and how to read it, in place of a link
  in-order "$output" 'Cutting a release' '4 The tour' 'Read here, not online.' 'in the kit: kit tutorials tour'
}

@test "--json says where each tutorial is: in the kit, with its file, or online, with its link" {
  versioned 0.4.7
  run --separate-stderr "${kit}" tutorials --json
  [ "$status" -eq 0 ]
  [ "$(jq --compact-output '.tutorials[0] | {where, url, file}' <<<"$output")" = \
    "{\"where\":\"online\",\"url\":\"${site}/0.4/first/?kit=0.4.7\",\"file\":\"\"}" ]
  [ "$(jq --compact-output '.tutorials[3] | {where, url, file}' <<<"$output")" = \
    "{\"where\":\"kit\",\"url\":\"\",\"file\":\"${BATS_TEST_TMPDIR}/tutorials/tour.md\"}" ]
}

@test "a tutorial named that ships with the kit prints its page, its front matter left out, and opens no browser" {
  fake-openers
  run --separate-stderr "${kit_root}/bin/kit" tutorials tour
  [ "$status" -eq 0 ]
  [ "$output" = "$(printf '4 The tour\n\nThe first screen is Flow.')" ]
  [ ! -e "${BATS_TEST_TMPDIR}/opened" ]
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
  run --separate-stderr "${kit_root}/bin/kit" syntax tutorials --choices='<tutorial>'
  [ "$status" -eq 0 ]
  [[ "$output" == *"first"* && "$output" == *"Where to begin."* ]] || [[ "$output" == *"The first tutorial"* ]]
  run --separate-stderr "${kit_root}/bin/kit" syntax tutorials --choices='<audience>'
  [[ "$output" == *"newcomer"* && "$output" == *"Just starting"* ]]
}

@test "on a terminal, it opens kittui's screen" {
  mkdir -p "${BATS_TEST_TMPDIR}/fake"
  ln -s "${BATS_TEST_DIRNAME}/fakes/fake-kittui" "${BATS_TEST_TMPDIR}/fake/kittui"
  ln -s "${BATS_TEST_DIRNAME}/fakes/fake-go" "${BATS_TEST_TMPDIR}/fake/go"
  export FAKES="${BATS_TEST_TMPDIR}/fake" PATH="${BATS_TEST_TMPDIR}/fake:${PATH}"
  run pty "'${kit_root}/bin/kit' tutorials --tag=newcomer --tag=ui"
  [[ "$output" == *"kittui native-native tutorials --tag=newcomer --tag=ui"* ]]
  run pty "'${kit_root}/bin/kit' tutorials --text"
  [[ "$output" == *"The first tutorial"* && "$output" != *"native-native"* ]]
  # a tutorial named opens the screen on its card, the browser left alone
  fake-openers
  run pty "'${kit_root}/bin/kit' tutorials --tag=ui screens"
  [[ "$output" == *"kittui native-native tutorials --tag=ui screens"* ]]
  [ ! -e "${BATS_TEST_TMPDIR}/opened" ]
}
