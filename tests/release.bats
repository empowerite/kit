#!/usr/bin/env bats
# release.bats - a release packed the way the workflow packs it installs from its unpacked directory
# ----------------------------------------------------------------------------------------------------------------------

setup() {
  kit_checkout=$(cd "${BATS_TEST_DIRNAME}/.." && pwd)
  export KIT_HOME="${BATS_TEST_TMPDIR}/home"
  mkdir -p "${KIT_HOME}"
  work=$(mktemp -d)
  cd "${work}"
}

teardown() {
  rm -rf "${work}"
}

@test "the tarball holds one directory with the release's layout, and installs a fresh repository" {
  # packed the way the workflow packs it, with the tag's version stamped in
  version=9.9.9
  mkdir "kit-${version}"
  cp -R "${kit_checkout}/bin" "${kit_checkout}/modules" "kit-${version}/"
  printf '%s\n' "${version}" > "kit-${version}/version"
  tar --create --gzip --file "kit-${version}.tar.gz" "kit-${version}"
  rm -rf "kit-${version}"
  mkdir unpacked && tar --extract --gzip --file "kit-${version}.tar.gz" --directory unpacked
  [ "$(ls unpacked)" = "kit-${version}" ]
  [ -x "unpacked/kit-${version}/bin/kit" ]
  [ -x "unpacked/kit-${version}/bin/kit-install.sh" ]
  [ -f "unpacked/kit-${version}/modules/claude/module.yml" ]
  mkdir fresh && git -C fresh init --quiet --initial-branch=main
  cd fresh
  run "${work}/unpacked/kit-${version}/bin/kit" install claude
  [ "$status" -eq 0 ]
  run "${work}/unpacked/kit-${version}/bin/kit" check
  [ "$status" -eq 0 ]
  [ "$("${work}/unpacked/kit-${version}/bin/kit" version)" = "${version}" ]
  grep --quiet "^version ${version}$" .kit/manifest
}

# ----------------------------------------------------------------------------------------------------------------------
