#!/usr/bin/env bats
# sources.bats - packages from sources beyond the release: pinned by digest, fetched once, and looked up last first
# ----------------------------------------------------------------------------------------------------------------------
# Each source here is a tarball on disk, fetched by a file:// URL, so the tests need no network. A git: URL is checked
# by the address it names.

bats_require_minimum_version 1.5.0

load helper

setup_file() {
  world-repo installed
}

setup() {
  world-copy-repo installed
}

kit() { "${kit_checkout}/bin/kit" "$@"; }

# source-tarball <name> <package> <note> makes a tarball whose packages/<package> delivers .kit/notes/<note>.md and
# bash/<name>.sh, and prints its path and its digest, tab-separated.
source-tarball() {
  local dir="${BATS_TEST_TMPDIR}/src-$1" digest
  mkdir -p "${dir}/top/packages/$2/files/repo/.kit/notes" "${dir}/top/bash"
  printf 'description: "the %s package from %s"\ndepends:\n  - kit\n' "$2" "$1" > "${dir}/top/packages/$2/package.yml"
  printf '%s\n' "# kit:generated -- a note, from the $2 package" "# see: kit packages remove $2" '' "$3" \
    > "${dir}/top/packages/$2/files/repo/.kit/notes/$3.md"
  printf '%s:hello() { printf "hello from %s\\n"; }\n' "$1" "$1" > "${dir}/top/bash/$1.sh"
  tar --create --gzip --file "${dir}/source.tar.gz" --directory "${dir}" top
  digest=$(shasum -a 256 "${dir}/source.tar.gz" | cut -d ' ' -f 1)
  printf '%s\tsha256:%s\n' "${dir}/source.tar.gz" "${digest}"
}

@test "a source's package installs, comes from its source, and its bash directory joins the module path" {
  IFS=$'\t' read -r file digest < <(source-tarball team guides first)
  printf 'sources:\n  - name: team\n    url: file://%s\n    digest: %s\nbash_path:\n  - kit:modules/bash/\n  - team:bash\n' \
    "${file}" "${digest}" >> kit.yml
  run kit packages add guides
  [ "$status" -eq 0 ]
  [ -f .kit/notes/first.md ]
  run kit packages guides
  [[ "$output" == *"guides"*"team"*"the guides package from team"* ]]
  run bash -c "source '${kit_checkout}/modules/bash/1st.sh' && use team && team:hello"
  [ "$output" = "hello from team" ]
  run kit check
  [ "$status" -eq 0 ]
}

@test "a source whose bytes have another digest is refused before anything of it is read" {
  IFS=$'\t' read -r file _ < <(source-tarball team guides first)
  printf 'sources:\n  - name: team\n    url: file://%s\n    digest: sha256:%064d\n' "${file}" 0 >> kit.yml
  run kit packages add guides
  [ "$status" -eq 2 ]
  [[ "$output" == *"the source team has digest sha256:"*", not the sha256:0000"* ]]
  [ ! -e "${XDG_CACHE_HOME}/kit/sources" ] || [ -z "$(ls "${XDG_CACHE_HOME}/kit/sources")" ]
  printf 'sources:\n  - name: bare\n    url: file://%s\n' "${file}" > kit.yml.extra
  sed -i.bak '/^sources:/,$d' kit.yml && rm kit.yml.bak && cat kit.yml.extra >> kit.yml
  run kit packages
  [[ "$output" == *"the source bare has no digest"* ]]
}

@test "of two sources with a package of one name, the later one's is the package" {
  IFS=$'\t' read -r one one_digest < <(source-tarball one guides first)
  IFS=$'\t' read -r two two_digest < <(source-tarball two guides second)
  printf 'sources:\n  - name: one\n    url: file://%s\n    digest: %s\n  - name: two\n    url: file://%s\n    digest: %s\n' \
    "${one}" "${one_digest}" "${two}" "${two_digest}" >> kit.yml
  run kit packages add guides
  [ "$status" -eq 0 ]
  [ -f .kit/notes/second.md ]
  [ ! -e .kit/notes/first.md ]
  run kit packages guides
  [[ "$output" == *"guides"*"two"* ]]
}

@test "a git: source names its forge's tarball of the tag" {
  run bash -c "source '${kit_checkout}/modules/bash/1st.sh' && use sources && sources:tarball git:acme/rules@v1.2.0 \
    && sources:tarball git:gitlab.com/acme/rules@v1.2.0 && sources:tarball https://example.com/r.tar.gz"
  [ "${lines[0]}" = "https://codeload.github.com/acme/rules/tar.gz/refs/tags/v1.2.0" ]
  [ "${lines[1]}" = "https://gitlab.com/acme/rules/-/archive/v1.2.0/rules-v1.2.0.tar.gz" ]
  [ "${lines[2]}" = "https://example.com/r.tar.gz" ]
}

# ----------------------------------------------------------------------------------------------------------------------
