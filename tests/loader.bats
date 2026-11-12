#!/usr/bin/env bats
# loader.bats - bin/kit, the loader: which release a call runs, the kit options, and what it hands the release
# ----------------------------------------------------------------------------------------------------------------------
# Each test gets an empty git repository and runs this checkout's bin/kit in it. A release the test pins is a copy of
# this checkout with a version file, made by later-release, in the kit's cache, at another path, or as the clone's own
# kit/; one that must be downloaded is served by the fake curl, as tests/fetch.bats serves one.

bats_require_minimum_version 1.5.0

load helper

setup() {
  fresh-repo >/dev/null
  kit="${kit_checkout}/bin/kit"
  cache="${XDG_CACHE_HOME}/kit"
}

# cached <version> puts a release of that version in the kit's cache.
cached() {
  later-release "${cache}/kit-$1" "$1"
}

# telling <dir> makes a release at <dir> whose every call prints what the loader handed it: KIT_VERSION, then
# KIT_VERSION_ROOT, one per line, then the words of the call, then KIT_WORKSPACE, or unset.
telling() {
  later-release "$1" 9.9.9
  printf '#!/usr/bin/env bash\nprintf "%%s\\n" "${KIT_VERSION}" "${KIT_VERSION_ROOT}" "$*" "${KIT_WORKSPACE-unset}"\n' \
    > "$1/libexec/kit-call"
}

# publish <version> puts kit-<version>.tar.gz, a release of that version, and its .sha256 where the fake curl serves
# them, as bin/release pack publishes them, and puts the fake curl first on PATH.
publish() {
  local stage
  export FAKE_RELEASES="${BATS_TEST_TMPDIR}/releases"
  mkdir -p "${FAKE_RELEASES}" "${BATS_TEST_TMPDIR}/bin"
  ln -sf "${BATS_TEST_DIRNAME}/fakes/fake-curl" "${BATS_TEST_TMPDIR}/bin/curl"
  export PATH="${BATS_TEST_TMPDIR}/bin:${PATH}"
  stage=$(mktemp -d)
  later-release "${stage}/kit-$1" "$1"
  tar --create --gzip --file "${FAKE_RELEASES}/kit-$1.tar.gz" --directory "${stage}" "kit-$1"
  printf '%s  kit-%s.tar.gz\n' "$(shasum -a 256 "${FAKE_RELEASES}/kit-$1.tar.gz" | cut -d ' ' -f 1)" "$1" \
    > "${FAKE_RELEASES}/kit-$1.tar.gz.sha256"
  rm -rf "${stage}"
}

@test "with no kit.yml, kit runs this release, and -v and --version run kit version in it" {
  run "${kit}" version
  [ "$status" -eq 0 ]
  [ "$output" = unreleased ]
  run "${kit}" -v
  [ "$output" = unreleased ]
  run "${kit}" --version
  [ "$output" = unreleased ]
}

@test "kit.yml's version: runs that release from the cache, quoted or not, with a comment beside it or not" {
  cached 9.9.9
  printf 'version: 9.9.9\n' > kit.yml
  run "${kit}" version
  [ "$status" -eq 0 ]
  [ "$output" = 9.9.9 ]
  printf 'facts: {}\nversion: "9.9.9"  # pinned\n' > kit.yml
  run "${kit}" version
  [ "$output" = 9.9.9 ]
  printf "version: '9.9.9'\n" > kit.yml
  run "${kit}" version
  [ "$output" = 9.9.9 ]
}

@test "a kit.yml with no version: runs the release its .kit/manifest records, and one with neither runs this one" {
  cached 9.9.9
  mkdir .kit && printf 'version 9.9.9\n' > .kit/manifest && printf 'packages:\n  - kit\n' > kit.yml
  run "${kit}" version
  [ "$output" = 9.9.9 ]
  rm .kit/manifest
  run "${kit}" version
  [ "$output" = unreleased ]
}

@test "--version= and -v= run the release they name over kit.yml's, and so does KIT_VERSION" {
  cached 9.9.9
  cached 9.9.8
  printf 'version: 9.9.8\n' > kit.yml
  run "${kit}" --version=9.9.9 version
  [ "$output" = 9.9.9 ]
  run "${kit}" -v=9.9.9 version
  [ "$output" = 9.9.9 ]
  run env KIT_VERSION=9.9.9 "${kit}" version
  [ "$output" = 9.9.9 ]
  run "${kit}" version
  [ "$output" = 9.9.8 ]
}

@test "a release is named by its path too, and a path that holds no kit is refused" {
  later-release "${BATS_TEST_TMPDIR}/elsewhere" 9.9.7
  run "${kit}" --version="${BATS_TEST_TMPDIR}/elsewhere" version
  [ "$status" -eq 0 ]
  [ "$output" = 9.9.7 ]
  run "${kit}" --version="${BATS_TEST_TMPDIR}" version
  [ "$status" -eq 1 ]
  [[ "$output" == *"--version names ${BATS_TEST_TMPDIR}, which is not a kit release: it has no bin/kit"* ]]
}

@test "the loader hands the release the release it chose and the clone it chose it for, and the verb's words as typed" {
  telling "${BATS_TEST_TMPDIR}/telling"
  run "${kit}" --version="${BATS_TEST_TMPDIR}/telling" flow --stages 'two words'
  [ "$status" -eq 0 ]
  [ "${lines[0]}" = "$(cd "${BATS_TEST_TMPDIR}/telling" && pwd -P)" ]
  [ "${lines[1]}" = "$(pwd -P)" ]
  [ "${lines[2]}" = "flow --stages two words" ]
}

@test "KIT_VERSION holds in the clone it was chosen for, and another clone chooses again from its own kit.yml" {
  telling "${BATS_TEST_TMPDIR}/telling"
  cached 9.9.8
  printf 'version: 9.9.8\n' > kit.yml
  run env KIT_VERSION="${BATS_TEST_TMPDIR}/telling" KIT_VERSION_ROOT="$(pwd -P)" "${kit}" version
  [ "${lines[2]}" = version ]
  run env KIT_VERSION="${BATS_TEST_TMPDIR}/telling" KIT_VERSION_ROOT="${BATS_TEST_TMPDIR}/another" "${kit}" version
  [ "$output" = 9.9.8 ]
}

@test "the loader hands on the workspace --workspace= names, and an empty one without it, whatever the environment holds" {
  telling "${BATS_TEST_TMPDIR}/telling"
  run env KIT_WORKSPACE=stale "${kit}" --version="${BATS_TEST_TMPDIR}/telling" flow
  [ "${lines[2]}" = flow ]
  [ "${#lines[@]}" -eq 3 ] || [ "${lines[3]}" = '' ]
  run "${kit}" --version="${BATS_TEST_TMPDIR}/telling" --workspace=none flow --stages
  [ "${lines[2]}" = "flow --stages" ]
  [ "${lines[3]}" = none ]
}

@test "a call across a workspace runs on the installed kit, whatever the clone pins, and a hand-off carries the workspace" {
  cached 9.9.9
  printf 'version: 9.9.9\n' > kit.yml
  run "${kit}" --workspace=w version
  [ "$output" = unreleased ]
  run "${kit}" --workspace=none version
  [ "$output" = 9.9.9 ]
  telling "${cache}/kit-9.9.8"
  printf 'version: 9.9.8\n' > kit.yml
  run "${kit}" --workspace=none flow
  [ "${lines[2]}" = flow ]
  [ "${lines[3]}" = none ]
}

@test "unreleased is the clone's own kit/ where it holds bin/kit beside modules/bash/1st.sh, and this release elsewhere" {
  printf 'version: unreleased\n' > kit.yml
  later-release kit 9.9.6
  run "${kit}" version
  [ "$output" = 9.9.6 ]
  rm kit/modules/bash/1st.sh
  run "${kit}" version
  [ "$output" = unreleased ]
}

@test "the loader reads the kit.yml of the clone it runs in alone: a clone inside another never reads the outer one" {
  cached 9.9.9
  printf 'version: 9.9.9\n' > kit.yml
  mkdir -p deep/er
  run bash -c 'cd deep/er && "$0" version' "${kit}"
  [ "$output" = 9.9.9 ]
  git init --quiet inner
  run bash -c 'cd inner && "$0" version' "${kit}"
  [ "$output" = unreleased ]
  mkdir wt && printf 'gitdir: %s/.git/worktrees/wt\n' "$(pwd -P)" > wt/.git
  run bash -c 'cd wt && "$0" version' "${kit}"
  [ "$output" = unreleased ]
}

@test "a release not in the cache is downloaded, checked against its digest, cached, and read from the cache next time" {
  publish 9.9.5
  printf 'version: 9.9.5\n' > kit.yml
  run "${kit}" version
  [ "$status" -eq 0 ]
  [ "${lines[-1]}" = 9.9.5 ]
  [[ "$output" == *"Fetching kit 9.9.5 from its release"* ]]
  [ -x "${cache}/kit-9.9.5/bin/kit" ]
  [ "$(wc -l < "${FAKE_RELEASES}/curl.calls")" -eq 2 ]
  run "${kit}" version
  [ "$output" = 9.9.5 ]
  [ "$(wc -l < "${FAKE_RELEASES}/curl.calls")" -eq 2 ]
}

@test "a downloaded release whose bytes have another digest is refused and not cached, and so is one with no digest" {
  publish 9.9.5
  printf 'x' >> "${FAKE_RELEASES}/kit-9.9.5.tar.gz"
  printf 'version: 9.9.5\n' > kit.yml
  run "${kit}" version
  [ "$status" -eq 1 ]
  [[ "$output" == *"kit 9.9.5's tarball has digest"*"it is not unpacked"* ]]
  [ ! -e "${cache}/kit-9.9.5" ]
  rm "${FAKE_RELEASES}/kit-9.9.5.tar.gz.sha256"
  run "${kit}" version
  [ "$status" -eq 1 ]
  [[ "$output" == *"kit 9.9.5 publishes no digest beside its tarball, so it is not trusted"* ]]
  run "${kit}" --version=9.9.4 version
  [ "$status" -eq 1 ]
  [[ "$output" == *"kit 9.9.4 could not be fetched from its release"* ]]
}

@test "a version that is not one is refused before it is used in a path" {
  printf 'version: ../../elsewhere\n' > kit.yml
  run "${kit}" version
  [ "$status" -eq 1 ]
  [[ "$output" == *"/kit.yml names '../../elsewhere', which is not a kit version"* ]]
  run "${kit}" --version=nine version
  [ "$status" -eq 1 ]
  [[ "$output" == *"--version names 'nine', which is not a kit version"* ]]
}

@test "an option the loader does not know is refused with exit 2, and so is --version= with no release" {
  run "${kit}" --workspaces=x version
  [ "$status" -eq 2 ]
  [[ "$output" == *"kit unreleased does not know the option --workspaces; update the kit you installed"* ]]
  run "${kit}" --version= version
  [ "$status" -eq 2 ]
  [[ "$output" == *"--version= needs a release"* ]]
}

@test "-h and --help run kit help, -- ends the options, and the first other word is the verb" {
  run "${kit}" help
  expected=$output
  run "${kit}" -h
  [ "$status" -eq 0 ]
  [ "$output" = "${expected}" ]
  run "${kit}" --help
  [ "$output" = "${expected}" ]
  run "${kit}" -- version
  [ "$output" = unreleased ]
}

@test "a bash older than 4.4 first on PATH is refused before anything runs, with the command that installs one" {
  /bin/bash -c '(( BASH_VERSINFO[0] < 4 ))' || skip "/bin/bash here is bash 4 or later"
  mkdir old && ln -s /bin/bash old/bash
  run env PATH="$(pwd)/old:/usr/bin:/bin" "${kit}" version
  [ "$status" -eq 1 ]
  [[ "$output" == *"kit needs bash 4.4 or later; the bash on PATH"*"is 3."* ]]
}

# ----------------------------------------------------------------------------------------------------------------------
