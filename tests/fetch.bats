#!/usr/bin/env bats
# fetch.bats - a release is fetched with the digest published beside it, and unpacked only when the bytes match it
# ----------------------------------------------------------------------------------------------------------------------
# release:fetch brings the previous release for kit update's three-way merge, and the delivered CI job brings the
# pinned release onto a runner. Both read kit-<version>.tar.gz and its .sha256 from the release, through a fake curl
# that serves the files a test puts in ${FAKE_RELEASES}.

bats_require_minimum_version 1.5.0

load helper

setup() {
  kit_bash_modules=release
  fresh-repo
  export FAKE_RELEASES="${repo}.releases"
  mkdir -p "${FAKE_RELEASES}" "${repo}.bin"
  ln -s "${BATS_TEST_DIRNAME}/fakes/fake-curl" "${repo}.bin/curl"
  # a mise that records each call, so a test proves no toolchain is asked; the fake curl first, so no real one answers
  printf '#!/usr/bin/env bash\necho "$*" >> "%s"\n' "${repo}.mise.calls" > "${repo}.bin/mise"
  chmod +x "${repo}.bin/mise"
  export PATH="${repo}.bin:${PATH}"
  publish 9.9.9
}

# publish <version> puts kit-<version>.tar.gz, one directory kit-<version>/ holding this checkout's bin/, and its
# .sha256 in the fake release, as bin/release pack publishes them.
publish() {
  local stage
  stage=$(mktemp -d)
  mkdir "${stage}/kit-$1"
  cp -R "${kit_root}/bin" "${stage}/kit-$1/"
  printf '%s\n' "$1" > "${stage}/kit-$1/version"
  tar --create --gzip --file "${FAKE_RELEASES}/kit-$1.tar.gz" --directory "${stage}" "kit-$1"
  printf '%s  kit-%s.tar.gz\n' "$(shasum -a 256 "${FAKE_RELEASES}/kit-$1.tar.gz" | cut -d ' ' -f 1)" "$1" \
    > "${FAKE_RELEASES}/kit-$1.tar.gz.sha256"
  rm -rf "${stage}"
}

# ci-step prints the kit-on-path step of the delivered CI job, as bash.
ci-step() {
  awk '
    /name: kit-on-path/ { on = 1; next }
    on && /run: \|/ { body = 1; next }
    on && body && /^      - name:/ { exit }
    on && body { sub(/^          /, ""); print }
  ' "${kit_root}/packages/github/ci/templates/repo/.github/workflows/kit-ci.yml"
}

@test "release:fetch checks the tarball against the digest published beside it, caches it, reads the cache next time, and asks no toolchain" {
  run kit-bash 'release:fetch 9.9.9'
  [ "$status" -eq 0 ]
  [ "${lines[-1]}" = "${XDG_CACHE_HOME}/kit/kit-9.9.9" ]
  [ -x "${XDG_CACHE_HOME}/kit/kit-9.9.9/bin/kit" ]
  [ "$(paste -s -d ' ' - < "${FAKE_RELEASES}/curl.calls")" = \
    "https://github.com/empowerite/kit/releases/download/release/9.9.9/kit-9.9.9.tar.gz https://github.com/empowerite/kit/releases/download/release/9.9.9/kit-9.9.9.tar.gz.sha256" ]
  run kit-bash 'release:fetch 9.9.9'
  [ "$status" -eq 0 ]
  [ "$(wc -l < "${FAKE_RELEASES}/curl.calls")" -eq 2 ]
  [ ! -e "${repo}.mise.calls" ]
}

@test "release:fetch refuses a tarball whose bytes have another digest, names both digests, and caches nothing" {
  printf 'x' >> "${FAKE_RELEASES}/kit-9.9.9.tar.gz"
  have=$(shasum -a 256 "${FAKE_RELEASES}/kit-9.9.9.tar.gz" | cut -d ' ' -f 1)
  published=$(cut -d ' ' -f 1 "${FAKE_RELEASES}/kit-9.9.9.tar.gz.sha256")
  run kit-bash 'release:fetch 9.9.9'
  [ "$status" -eq 1 ]
  [[ "$output" == *"${have}"* && "$output" == *"${published}"* ]]
  [ ! -e "${XDG_CACHE_HOME}/kit/kit-9.9.9" ]
}

@test "release:fetch refuses a release that publishes no digest, and one it cannot fetch, each in its own words" {
  rm "${FAKE_RELEASES}/kit-9.9.9.tar.gz.sha256"
  run kit-bash 'release:fetch 9.9.9'
  [ "$status" -eq 1 ]
  [[ "$output" == *"publishes no digest"* ]]
  [ ! -e "${XDG_CACHE_HOME}/kit/kit-9.9.9" ]
  run kit-bash 'release:fetch 9.9.8'
  [ "$status" -eq 1 ]
  [[ "$output" == *"kit 9.9.8 could not be fetched from its release"* ]]
}

@test "the delivered CI job puts the pinned release on PATH only after its bytes match the published digest" {
  printf 'version: 9.9.9\n' > kit.yml
  export RUNNER_TEMP="${repo}/runner" GITHUB_PATH="${repo}/github-path"
  mkdir "${RUNNER_TEMP}" && : > "${GITHUB_PATH}"
  # no kit on PATH: the step must download one
  run env PATH="${repo}.bin:/usr/bin:/bin" bash -c "$(ci-step)"
  [ "$status" -eq 0 ]
  [ -x "${RUNNER_TEMP}/kit/bin/kit" ]
  [ "$(cat "${GITHUB_PATH}")" = "${RUNNER_TEMP}/kit/bin" ]
  rm -rf "${RUNNER_TEMP}/kit" && : > "${GITHUB_PATH}"
  printf 'x' >> "${FAKE_RELEASES}/kit-9.9.9.tar.gz"
  run env PATH="${repo}.bin:/usr/bin:/bin" bash -c "$(ci-step)"
  [ "$status" -eq 1 ]
  [[ "$output" == *"digest"* ]]
  [ ! -e "${RUNNER_TEMP}/kit/bin/kit" ]
  [ ! -s "${GITHUB_PATH}" ]
}
