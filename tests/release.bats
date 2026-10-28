#!/usr/bin/env bats
# release.bats - bin/release verifies a tag and packs a checkout as a release that installs from its unpacked directory
# ----------------------------------------------------------------------------------------------------------------------

setup() {
  kit_checkout=$(cd "${BATS_TEST_DIRNAME}/../kit" && pwd)
  repo_bin=$(cd "${BATS_TEST_DIRNAME}/../bin" && pwd)
  export KIT_HOME="${BATS_TEST_TMPDIR}/home" XDG_CACHE_HOME="${BATS_TEST_TMPDIR}/cache"
  mkdir -p "${KIT_HOME}" "${XDG_CACHE_HOME}"
  work=$(mktemp -d)
  cd "${work}"
}

teardown() {
  rm -rf "${work}"
}

@test "verify holds a tag to its shape and its side of the default branch" {
  git init --quiet --bare --initial-branch=main origin.git && git clone --quiet origin.git desk 2>/dev/null && cd desk
  git config user.name t && git config user.email t@example.com && git config commit.gpgsign false
  printf 'one\n' > a && git add a && git commit --quiet --message one && git push --quiet origin main
  git remote set-head origin main
  git tag release/1.2.3 && git tag release/1.2.3-rc.1 && git tag release/nope && git tag v1.2.3
  git switch --quiet --create topic && printf 'two\n' > b && git add b && git commit --quiet --message two
  git tag release/1.2.4 && git tag release/1.2.4-rc.1
  run "${repo_bin}/release" verify release/1.2.3
  [ "$status" -eq 0 ]
  run "${repo_bin}/release" verify release/1.2.4-rc.1
  [ "$status" -eq 0 ]
  run "${repo_bin}/release" verify release/1.2.3-rc.1
  [ "$status" -eq 1 ]
  [[ "$output" == *"is a pre-release tag on main"* ]]
  run "${repo_bin}/release" verify release/1.2.4
  [ "$status" -eq 1 ]
  [[ "$output" == *"is not on main"* ]]
  run "${repo_bin}/release" verify release/nope
  [ "$status" -eq 1 ]
  [[ "$output" == *"is not a release tag"* ]]
  run "${repo_bin}/release" verify v1.2.3
  [ "$status" -eq 1 ]
  [[ "$output" == *"is not a release tag"* ]]
}

@test "publish refuses a release from a desk, and names the push that has the runner publish it" {
  run env -u GITHUB_ACTIONS "${repo_bin}/release" publish release/1.2.3
  [ "$status" -eq 1 ]
  [[ "$output" == *"a release is published by the release workflow, not from a desk"* ]]
  [[ "$output" == *"git push origin release/1.2.3"* ]]
}

@test "pack makes one directory with the release's layout and the version stamped in, which installs a repository" {
  version=9.9.9
  run "${repo_bin}/release" pack "release/${version}" "${work}"
  [ "$status" -eq 0 ]
  [ -f "kit-${version}.tar.gz.sha256" ]
  grep --quiet "  kit-${version}.tar.gz$" "kit-${version}.tar.gz.sha256"
  mkdir unpacked && tar --extract --gzip --file "kit-${version}.tar.gz" --directory unpacked
  [ "$(ls unpacked)" = "kit-${version}" ]
  [ -x "unpacked/kit-${version}/bin/kit" ]
  [ -x "unpacked/kit-${version}/bin/kit-install" ]
  [ -f "unpacked/kit-${version}/packages/claude/package.yml" ]
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
