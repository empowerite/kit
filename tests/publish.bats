#!/usr/bin/env bats
# publish.bats - kit publish pushes a release's artifacts to their targets from a runner; pull behind push is watched
# ----------------------------------------------------------------------------------------------------------------------
# origin is a bare repository beside the test's, so the tags are real git. gh, jf, and npm are tests/fakes/fake-tool,
# each answering from its routes, so what is proven is what kit asks each connector. tests/fakes/fake-curl serves the
# pull path: ${FAKE_RELEASES}/mirror.example/<file> is what the mirror serves.

bats_require_minimum_version 1.5.0

load helper

setup() {
  fake-tools gh jf npm
  export FAKE_RELEASES="${BATS_TEST_TMPDIR}/releases"
  mkdir -p "${FAKE_RELEASES}/mirror.example"
  ln -sf "${BATS_TEST_DIRNAME}/fakes/fake-curl" "${FAKE_DIR}/bin/curl"
  unset GITHUB_ACTIONS GITLAB_CI TF_BUILD KIT_SOURCES
  : > "${FAKE_DIR}/empty"
  printf '[]\n' > "${FAKE_DIR}/none.json"
  repo=$(mktemp -d)
  git init --quiet --bare --initial-branch=main "${repo}/origin.git"
  git clone --quiet "${repo}/origin.git" "${repo}/desk" 2>/dev/null
  cd "${repo}/desk"
  printf 'one\n' > one && git add one && git commit --quiet --message one
  git push --quiet origin main && git remote set-head origin main
  settings
  mkdir -p dist
  printf 'kit 1.0.0\n' > dist/kit-1.0.0.tar.gz
  printf 'kit 1.1.0-rc.1\n' > dist/kit-1.1.0-rc.1.tar.gz
}

# settings [<targets>] writes kit.yml: a mirror on the pull path, and the kit artifact, whose release goes to the
# targets, a YAML flow list, github-release and vcs-tag by default, and whose pre-release goes to github-release.
settings() {
  cat > kit.yml <<YAML
forge: github
sources:
  - name: mirror
    kind: github-releases
    url: https://mirror.example
    expect: 1h
  - off
publish:
  kit:
    files: ["dist/kit-{version}.tar.gz"]
    release: acme/kit
    tag: release/{version}
    asset: kit-{version}.tar.gz
    targets:
      release: ${1:-[github-release, vcs-tag]}
      prerelease: [github-release]
YAML
}

# tagged <version> tags HEAD as the release and pushes the tag, as kit release does.
tagged() { git tag --annotate --message "release $1" "release/$1" && git push --quiet origin "refs/tags/release/$1"; }

# routes-free makes gh answer that no release holds a version, and take every create.
routes-free() {
  route gh 'release view *' '!1'
  route gh 'release create *' empty
}

@test "a release is published from a runner alone; a desk may publish a pre-release, at its own risk" {
  tagged 1.0.0
  routes-free
  run "${kit_root}/bin/kit" publish 1.0.0
  [ "$status" -eq 1 ]
  [[ "$output" == *"a release is published from a runner, never at a desk; push its tag with kit release 1.0.0"* ]]
  run ! grep --quiet 'release create' "${FAKE_DIR}/gh.calls"
  tagged 1.1.0-rc.1
  run "${kit_root}/bin/kit" publish 1.1.0-rc.1
  [ "$status" -eq 0 ]
  [[ "$output" == *"publishing the pre-release 1.1.0-rc.1 from a desk, at your own risk"* ]]
  grep --quiet --line-regexp 'release create release/1.1.0-rc.1 dist/kit-1.1.0-rc.1.tar.gz --verify-tag --title kit 1.1.0-rc.1 --notes kit 1.1.0-rc.1 --prerelease' \
    "${FAKE_DIR}/gh.calls"
}

@test "on a runner each target's connector takes the release, and release-hashes records what was pushed" {
  settings '[github-release, vcs-tag, {kind: artifactory-local, repository: libs-local}]'
  tagged 1.0.0
  routes-free
  route jf 'rt search *' none.json
  route jf 'rt upload *' empty
  GITHUB_ACTIONS=true run "${kit_root}/bin/kit" publish 1.0.0
  [ "$status" -eq 0 ]
  [[ "$output" == *"published 1.0.0 from GitHub Actions"* ]]
  grep --quiet --line-regexp 'release create release/1.0.0 dist/kit-1.0.0.tar.gz --verify-tag --title kit 1.0.0 --notes kit 1.0.0' \
    "${FAKE_DIR}/gh.calls"
  grep --quiet --line-regexp 'rt upload dist/kit-1.0.0.tar.gz libs-local/kit/1.0.0/' "${FAKE_DIR}/jf.calls"
  # vcs-tag pushed v1.0.0 on the commit release/1.0.0 tags
  [ "$(git ls-remote origin 'refs/tags/v1.0.0^{}' | cut -f 1)" = "$(git rev-parse HEAD)" ]
  hash=$(shasum -a 256 dist/kit-1.0.0.tar.gz | cut -d ' ' -f 1)
  git fetch --quiet origin 'refs/tags/release-hashes/1.0.0:refs/tags/fetched'
  [ "$(git cat-file -p fetched)" = "kit"$'\t'"sha256:${hash}" ]
}

@test "a version a target holds already is refused before any target is pushed to: a version is never overwritten" {
  tagged 1.0.0
  route gh 'release view *' empty
  route gh 'release create *' empty
  GITHUB_ACTIONS=true run "${kit_root}/bin/kit" publish 1.0.0
  [ "$status" -eq 1 ]
  [[ "$output" == *"kit: the github-release target holds 1.0.0 already; a version is never overwritten"* ]]
  run ! grep --quiet 'release create' "${FAKE_DIR}/gh.calls"
  [ -z "$(git ls-remote origin refs/tags/v1.0.0)" ]
  [ -z "$(git ls-remote origin refs/tags/release-hashes/1.0.0)" ]
}

@test "a version not tagged on the primary, a file not built, a connector not on PATH, or an unknown target is refused" {
  routes-free
  GITHUB_ACTIONS=true run "${kit_root}/bin/kit" publish 1.0.0
  [ "$status" -eq 1 ]
  [[ "$output" == *"release/1.0.0 is not on origin; tag and push it first: kit release 1.0.0"* ]]
  tagged 1.0.0
  rm dist/kit-1.0.0.tar.gz
  GITHUB_ACTIONS=true run "${kit_root}/bin/kit" publish 1.0.0
  [[ "$output" == *"kit: dist/kit-1.0.0.tar.gz is not here; build it before kit publish 1.0.0"* ]]
  printf 'kit 1.0.0\n' > dist/kit-1.0.0.tar.gz
  settings '[{kind: artifactory-local, repository: libs-local}]'
  rm "${FAKE_DIR}/bin/jf"
  GITHUB_ACTIONS=true run "${kit_root}/bin/kit" publish 1.0.0
  [[ "$output" == *"kit: the artifactory-local target runs jf, which is not on PATH; install it on the runner"* ]]
  settings '[{kind: ftp, host: x}]'
  GITHUB_ACTIONS=true run "${kit_root}/bin/kit" publish 1.0.0
  [ "$status" -eq 1 ]
  [[ "$output" == *"kit.yml: the artifact kit has a target of the kind ftp, which no provider answers"* ]]
  settings '[{kind: github-release, branch: x}]'
  GITHUB_ACTIONS=true run "${kit_root}/bin/kit" publish 1.0.0
  [[ "$output" == *"has a github-release target with the key branch, which a github-release target does not take"* ]]
  settings
  run "${kit_root}/bin/kit" publish 1.0
  [ "$status" -eq 2 ]
}

@test "--retire takes the version back from each target that can: a pre-release's release and its tag" {
  tagged 1.1.0-rc.1
  route gh 'release delete *' empty
  run "${kit_root}/bin/kit" publish 1.1.0-rc.1 --retire
  [ "$status" -eq 0 ]
  grep --quiet --line-regexp 'release delete release/1.1.0-rc.1 --yes --cleanup-tag' "${FAKE_DIR}/gh.calls"
  run "${kit_root}/bin/kit" publish 1.0.0 --retire
  [ "$status" -eq 1 ]
  [[ "$output" == *"a release is published from a runner, never at a desk"* ]]
}

@test "npm-publish publishes the package's tarball, a pre-release under next, and asks the registry what it holds" {
  settings '[npm-publish]'
  sed -i.bak 's|    release: acme/kit|    npm: "@acme/kit"|; /    tag: /d; /    asset: /d' kit.yml && rm kit.yml.bak
  sed -i.bak 's|      prerelease: \[github-release\]|      prerelease: [{kind: npm-publish, registry: "https://npm.example"}]|' kit.yml
  rm kit.yml.bak
  route npm 'view *' '!1'
  route npm 'publish *' empty
  tagged 1.0.0
  GITHUB_ACTIONS=true run "${kit_root}/bin/kit" publish 1.0.0
  [ "$status" -eq 0 ]
  grep --quiet --line-regexp 'view @acme/kit@1.0.0 version' "${FAKE_DIR}/npm.calls"
  grep --quiet --line-regexp 'publish dist/kit-1.0.0.tar.gz' "${FAKE_DIR}/npm.calls"
  # the hash recorded is npm's integrity of the tarball
  git fetch --quiet origin 'refs/tags/release-hashes/1.0.0:refs/tags/fetched'
  [[ "$(git cat-file -p fetched)" == "kit"$'\t'"sha512-"* ]]
  tagged 1.1.0-rc.1
  run "${kit_root}/bin/kit" publish 1.1.0-rc.1
  grep --quiet --line-regexp 'publish dist/kit-1.1.0-rc.1.tar.gz --registry=https://npm.example --tag=next' \
    "${FAKE_DIR}/npm.calls"
}

# published <version> publishes the version from a runner, as the release workflow does.
published() {
  routes-free
  tagged "$1"
  GITHUB_ACTIONS=true "${kit_root}/bin/kit" publish "$1" >/dev/null 2>&1
}

@test "kit release --wait waits until the mirror serves the release with the hash release-hashes records" {
  published 1.0.0
  export KIT_RELEASE_POLL=1
  run "${kit_root}/bin/kit" release 1.0.0 --wait=2s
  [ "$status" -eq 3 ]
  [[ "$output" == *"release/1.0.0 is on origin already, so kit release only waits"* ]]
  [[ "$output" == *"1.0.0 is not served everywhere after 2s: kit pending on mirror"* ]]
  cp dist/kit-1.0.0.tar.gz "${FAKE_RELEASES}/mirror.example/"
  run "${kit_root}/bin/kit" release 1.0.0 --wait=5s
  [ "$status" -eq 0 ]
  [[ "$output" == *"1.0.0 is visible on mirror"* ]]
  grep --quiet --line-regexp 'https://mirror.example/acme/kit/releases/download/release/1.0.0/kit-1.0.0.tar.gz' \
    "${FAKE_RELEASES}/curl.calls"
  run "${kit_root}/bin/kit" release 1.0.0 --wait=soon
  [ "$status" -eq 2 ]
}

@test "a mirror that serves other bytes is a mismatch, and one that says it never will is blocked" {
  published 1.0.0
  printf 'other bytes\n' > "${FAKE_RELEASES}/mirror.example/kit-1.0.0.tar.gz"
  KIT_RELEASE_POLL=1 run "${kit_root}/bin/kit" release 1.0.0 --wait=5s
  [ "$status" -eq 1 ]
  [[ "$output" == *"1.0.0: kit mismatch on mirror (sha256:"* ]]
  # a source of the repository's own whose fetch says it never will serve it
  mkdir -p bash/sources
  printf '%s\n' '# shellcheck shell=bash' 'sources/scanner:fetch() { return 5; }' 'sources/scanner:hash() { :; }' \
    'sources/scanner:home() { :; }' 'sources/scanner:keys() { :; }' \
    'sources/scanner:serves() { [[ "$1" == release:* ]]; }' 'sources/scanner:settings() { :; }' \
    'sources/scanner:versions() { return 4; }' > bash/sources/scanner.sh
  printf 'bash_path:\n  - kit:modules/bash/\n  - bash/\n' >> kit.yml
  KIT_SOURCES='scan=scanner:https://scan.example,off' KIT_RELEASE_POLL=1 \
    run "${kit_root}/bin/kit" release 1.0.0 --wait=5s
  [ "$status" -eq 1 ]
  [[ "$output" == *"1.0.0: kit blocked on scan (scan says it will not serve it)"* ]]
}

@test "kit flow releases shows each release's pull state, says a pending one past its source's expect is stuck, and --json carries it" {
  published 1.0.0
  : > "${FAKE_DIR}/gh.routes"
  printf '{"login": "ada"}\n' > "${FAKE_DIR}/user.json"
  # 1.0.0 was published two hours ago, past the mirror's expect: of an hour, and the mirror does not serve it
  at=$(date -u -r $(( $(date +%s) - 7200 )) +%Y-%m-%dT%H:%M:%SZ 2>/dev/null || date -u -d "@$(( $(date +%s) - 7200 ))" +%Y-%m-%dT%H:%M:%SZ)
  printf '[{"tag_name": "release/1.0.0", "name": "kit 1.0.0", "prerelease": false, "draft": false, "published_at": "%s", "created_at": "%s", "html_url": "https://github.com/acme/kit/releases/tag/release/1.0.0", "body": null, "assets": []}]\n' \
    "${at}" "${at}" > "${FAKE_DIR}/releases.json"
  route gh 'api user*' user.json
  route gh 'api --paginate repos/{owner}/{repo}/releases?per_page=100' releases.json
  run "${kit_root}/bin/kit" flow releases
  [ "$status" -eq 0 ]
  [[ "$output" == *"1.0.0  📦 release "*"pull: stuck on mirror"* ]]
  [[ "$output" == *"1.0.0: kit is stuck on mirror, 2h past 1h"* ]]
  forge-forget
  rm -rf "${XDG_CACHE_HOME}/kit/visibility"
  cp dist/kit-1.0.0.tar.gz "${FAKE_RELEASES}/mirror.example/"
  run --separate-stderr "${kit_root}/bin/kit" flow releases --json
  [ "$status" -eq 0 ]
  [ "$(jq --compact-output '.[0].visibility | map([.artifact, .source, .state])' <<<"$output")" \
    = '[["kit","mirror","visible"]]' ]
  # a visible answer is kept, so the mirror is not asked again
  calls=$(grep -c '' "${FAKE_RELEASES}/curl.calls")
  forge-forget
  "${kit_root}/bin/kit" flow releases --json >/dev/null 2>&1
  [ "$(grep -c '' "${FAKE_RELEASES}/curl.calls")" -eq "${calls}" ]
}

@test "a repository with nothing under publish: has no visibility to wait for, and its releases carry none" {
  printf 'forge: github\n' > kit.yml
  tagged 1.0.0
  run "${kit_root}/bin/kit" release 1.0.0 --wait=1s
  [ "$status" -eq 1 ]
  [[ "$output" == *"kit.yml names no artifact under publish: that the pull path can serve, so there is nothing to wait for"* ]]
}

# ----------------------------------------------------------------------------------------------------------------------
