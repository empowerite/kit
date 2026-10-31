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

@test "verify holds a tag to its shape, and a release tag to the default branch" {
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
  [ "$status" -eq 0 ]
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
  [ -f "unpacked/kit-${version}/packages/agents/claude/package.yml" ]
  mkdir fresh && git -C fresh init --quiet --initial-branch=main
  cd fresh
  run "${work}/unpacked/kit-${version}/bin/kit" install agents/claude
  [ "$status" -eq 0 ]
  run "${work}/unpacked/kit-${version}/bin/kit" check
  [ "$status" -eq 0 ]
  [ "$("${work}/unpacked/kit-${version}/bin/kit" version)" = "${version}" ]
  grep --quiet "^version: ${version}$" .kit/settings.yml
}

@test "publish builds kittui for the four platforms, named as mise's http backend asks, and uploads each with its digest" {
  local fakes="${BATS_TEST_TMPDIR}/fakes"
  mkdir -p "${fakes}"
  # go records each build's platform and flags and writes the file it was asked for; gh records its arguments
  printf '%s\n' '#!/usr/bin/env bash' \
    'out=""; for (( i = 1; i <= $#; i++ )); do [[ "${!i}" == -o ]] && { j=$(( i + 1 )); out=${!j}; }; done' \
    'printf "%s %s %s %s\n" "${GOOS}" "${GOARCH}" "${CGO_ENABLED}" "$*" >> "${FAKES}/go.calls"' \
    'printf "binary\n" > "${out}"' > "${fakes}/go"
  printf '%s\n' '#!/usr/bin/env bash' 'printf "%s\n" "$*" >> "${FAKES}/gh.calls"' > "${fakes}/gh"
  chmod +x "${fakes}/go" "${fakes}/gh"
  run env GITHUB_ACTIONS=true FAKES="${fakes}" PATH="${fakes}:${PATH}" "${repo_bin}/release" publish release/9.9.9
  [ "$status" -eq 0 ]
  [ "$(wc -l < "${fakes}/go.calls" | tr -d ' ')" -eq 4 ]
  grep -q '^linux amd64 0 build -trimpath -ldflags -s -w -X main.version=9.9.9 -o .*/kittui-9.9.9-linux-x64 ./cmd/kittui$' "${fakes}/go.calls"
  grep -q '^linux arm64 0 .*/kittui-9.9.9-linux-arm64 ' "${fakes}/go.calls"
  grep -q '^darwin arm64 0 .*/kittui-9.9.9-macos-arm64 ' "${fakes}/go.calls"
  grep -q '^darwin amd64 0 .*/kittui-9.9.9-macos-x64 ' "${fakes}/go.calls"
  local asset
  for asset in kit-9.9.9.tar.gz kittui-9.9.9-linux-x64 kittui-9.9.9-linux-arm64 kittui-9.9.9-macos-arm64 kittui-9.9.9-macos-x64
  do
    grep -q "/${asset} " "${fakes}/gh.calls" && grep -q "/${asset}.sha256" "${fakes}/gh.calls" \
      || { echo "not uploaded: ${asset}"; cat "${fakes}/gh.calls"; return 1; }
  done
}

# ----------------------------------------------------------------------------------------------------------------------
