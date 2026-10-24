#!/usr/bin/env bats
# release.bats - a release packed the way the workflow packs it installs from its unpacked directory
# ----------------------------------------------------------------------------------------------------------------------

setup() {
  kit_checkout=$(cd "${BATS_TEST_DIRNAME}/.." && pwd)
  work=$(mktemp -d)
  cd "${work}"
}

teardown() {
  rm -rf "${work}"
}

@test "the tarball holds one directory with bin/ and .kit/, no manifest, and installs a fresh repository" {
  version=$(cat "${kit_checkout}/.kit/version")
  mkdir "kit-${version}"
  cp -R "${kit_checkout}/bin" "${kit_checkout}/.kit" "kit-${version}/"
  rm -f "kit-${version}/.kit/manifest"
  tar --create --gzip --file "kit-${version}.tar.gz" "kit-${version}"
  rm -rf "kit-${version}"
  mkdir unpacked && tar --extract --gzip --file "kit-${version}.tar.gz" --directory unpacked
  [ "$(ls unpacked)" = "kit-${version}" ]
  [ -x "unpacked/kit-${version}/bin/kit" ]
  [ ! -e "unpacked/kit-${version}/.kit/manifest" ]
  mkdir fresh && git -C fresh init --quiet --initial-branch=main
  cd fresh
  run "${work}/unpacked/kit-${version}/bin/kit" install
  [ "$status" -eq 0 ]
  run "${work}/unpacked/kit-${version}/bin/kit" check
  [ "$status" -eq 0 ]
  [ "$("${work}/unpacked/kit-${version}/bin/kit" version)" = "${version}" ]
}

# ----------------------------------------------------------------------------------------------------------------------
