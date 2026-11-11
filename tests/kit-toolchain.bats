#!/usr/bin/env bats
# kit-toolchain.bats - the kit toolchain: tools: in kit.yml pins each tool, and tools.lock its download for every platform
# ----------------------------------------------------------------------------------------------------------------------
# tests/fakes/fake-curl stands in for curl, serving each URL's last part from ${FAKE_RELEASES}, and tests/fakes/fake-tool
# stands in for gh. Each tool is a script, so what is proven is what kit downloads, checks, unpacks, and puts on PATH.

bats_require_minimum_version 1.5.0

load helper

setup() {
  hide-system-toolchains
  fake-tools gh
  export FAKE_RELEASES="${BATS_TEST_TMPDIR}/releases"
  mkdir -p "${FAKE_RELEASES}"
  ln -sf "${BATS_TEST_DIRNAME}/fakes/fake-curl" "${FAKE_DIR}/bin/curl"
  # a runner sets CI, which makes the run a runner's; each test that wants one says so
  unset CI
  fresh-repo
  platform=$(kit-bash 'use toolchain/kit && toolchain/kit:platform')
  releases 1.0.0 2.0.0
  pins 1.0.0 2.0.0
}

# releases <alpha> <beta> serves alpha, a bare command, and beta, a tarball whose command is under beta-<v>/bin/, at the
# two versions, for every platform. Each command prints its name, version, and platform.
releases() {
  local plat os arch stage
  for plat in linux-x64 linux-arm64 macos-x64 macos-arm64
  do
    os=${plat%-*} arch=${plat#*-}
    [[ "${os}" == macos ]] && os=darwin
    printf '#!/bin/sh\necho alpha %s %s\n' "$1" "${plat}" > "${FAKE_RELEASES}/alpha-$1-${os}-${arch}"
    stage="${BATS_TEST_TMPDIR}/stage-$2-${plat}"
    mkdir -p "${stage}/beta-$2/bin"
    printf '#!/bin/sh\necho beta %s %s\n' "$2" "${plat}" > "${stage}/beta-$2/bin/beta"
    chmod +x "${stage}/beta-$2/bin/beta"
    tar --create --gzip --file "${FAKE_RELEASES}/beta-$2-${plat}.tar.gz" --directory "${stage}" "beta-$2"
  done
}

# pins <alpha> <beta> writes kit.yml's tools: with the two versions; beta is a desk's alone.
pins() {
  cat > kit.yml <<YAML
tools:
  alpha:
    version: "$1"
    url: https://example.com/alpha/alpha-{version}-{os}-{arch}
    os: {macos: darwin}
  beta:
    version: "$2"
    url: https://github.com/acme/beta/releases/download/v{version}/beta-{version}-{platform}.tar.gz
    bin: beta-{version}/bin
    profile: desk
YAML
}

# downloads prints how many URLs the fake curl has been asked for.
downloads() {
  if [[ -f "${FAKE_RELEASES}/curl.calls" ]]
  then wc -l < "${FAKE_RELEASES}/curl.calls" | tr -d ' '
  else echo 0
  fi
}

@test "the kit toolchain is usable where kit.yml has tools: or tools.lock is here, and nowhere else" {
  printf 'version: unreleased\n' > kit.yml
  run kit-bash 'use toolchain && toolchain:detect'
  grep --quiet --line-regexp $'kit\tproject\tno\tno' <<<"$output"
  pins 1.0.0 2.0.0
  run kit-bash 'use toolchain && toolchain:detect'
  grep --quiet --line-regexp $'kit\tproject\tyes\tno' <<<"$output"
}

@test "install locks every tool for every platform, installs this platform's, and kit tools path names them" {
  run "${kit_root}/bin/kit" tools install --toolchain kit
  [ "$status" -eq 0 ]
  [[ "$output" == *"tools.lock: locked alpha, beta"* ]]
  # a row per tool and platform, each with the digest of the file it names
  [ "$(grep -c -v '^#' tools.lock)" -eq 8 ]
  while IFS=$'\t' read -r _ _ _ _ url digest _
  do
    [ "${digest}" = "$(shasum -a 256 "${FAKE_RELEASES}/${url##*/}" | cut -d ' ' -f 1)" ]
  done < <(grep -v '^#' tools.lock)
  t=$'\t'
  grep --quiet -E "^beta${t}2\.0\.0${t}linux-x64${t}desk${t}https://github\.com/acme/beta/releases/download/v2\.0\.0/\
beta-2\.0\.0-linux-x64\.tar\.gz${t}[0-9a-f]{64}${t}beta-2\.0\.0/bin$" tools.lock
  run "${kit_root}/bin/kit" tools path --toolchain kit
  [ "$status" -eq 0 ]
  [ "$(wc -l <<<"$output" | tr -d ' ')" -eq 2 ]
  [ "$(PATH="$(paste -s -d ':' - <<<"$output"):${PATH}" alpha)" = "alpha 1.0.0 ${platform}" ]
  [ "$(PATH="$(paste -s -d ':' - <<<"$output"):${PATH}" beta)" = "beta 2.0.0 ${platform}" ]
}

@test "an install with the lock up to date and the tools in place downloads nothing" {
  "${kit_root}/bin/kit" tools install --toolchain kit >/dev/null 2>&1
  before=$(downloads)
  run "${kit_root}/bin/kit" tools install --toolchain kit
  [ "$status" -eq 0 ]
  [ "$(downloads)" -eq "${before}" ]
  [[ "$output" != *"locked"* ]]
}

@test "a pin that moves is locked again at a desk, and the tools of other pins are left as they were" {
  "${kit_root}/bin/kit" tools install --toolchain kit >/dev/null 2>&1
  beta_rows=$(grep '^beta' tools.lock)
  releases 1.1.0 2.0.0
  pins 1.1.0 2.0.0
  run "${kit_root}/bin/kit" tools install --toolchain kit
  [ "$status" -eq 0 ]
  [[ "$output" == *"tools.lock: locked alpha"* ]]
  [ "$(grep '^beta' tools.lock)" = "${beta_rows}" ]
  [ "$(grep -c $'^alpha\t1\.1\.0\t' tools.lock)" -eq 4 ]
  [ "$(grep -c $'^alpha\t1\.0\.0\t' tools.lock)" -eq 0 ]
}

@test "a runner refuses a lock behind kit.yml, naming the tool, and installs none of a desk's tools" {
  "${kit_root}/bin/kit" tools install --toolchain kit >/dev/null 2>&1
  releases 1.1.0 2.0.0
  pins 1.1.0 2.0.0
  CI=true run "${kit_root}/bin/kit" tools install --toolchain kit
  [ "$status" -eq 1 ]
  [[ "$output" == *"tools.lock is behind tools: in kit.yml for alpha; run kit tools install at a desk"* ]]
  # with the lock up to date, a runner installs alpha alone: beta is a desk's
  pins 1.0.0 2.0.0
  rm -rf "${KIT_HOME}/.kit/tools"
  CI=true run "${kit_root}/bin/kit" tools install --toolchain kit
  [ "$status" -eq 0 ]
  [ -d "${KIT_HOME}/.kit/tools/alpha/1.0.0/${platform}" ]
  [ ! -d "${KIT_HOME}/.kit/tools/beta" ]
  CI=true run "${kit_root}/bin/kit" tools path --toolchain kit
  [ "$output" = "${KIT_HOME}/.kit/tools/alpha/1.0.0/${platform}" ]
}

@test "a download whose bytes are not the ones the lock pins is refused, naming both digests, and nothing is installed" {
  "${kit_root}/bin/kit" tools install --toolchain kit >/dev/null 2>&1
  rm -rf "${KIT_HOME}/.kit/tools" "${XDG_CACHE_HOME}/kit/downloads"
  os=${platform%-*} arch=${platform#*-}
  [[ "${os}" == macos ]] && os=darwin
  printf '#!/bin/sh\necho tampered\n' > "${FAKE_RELEASES}/alpha-1.0.0-${os}-${arch}"
  run "${kit_root}/bin/kit" tools install --toolchain kit
  [ "$status" -eq 1 ]
  [[ "$output" =~ alpha:\ https://example\.com/alpha/alpha-1\.0\.0-[a-z0-9-]+\ has\ digest\ sha256:[0-9a-f]{64},\ not\ the\ sha256:[0-9a-f]{64}\ tools\.lock\ pins ]]
  [ ! -d "${KIT_HOME}/.kit/tools/alpha" ]
}

@test "a bin: that is not in what the URL unpacks to is refused, naming the tool and its bin:" {
  sed -i.bak 's|bin: beta-{version}/bin|bin: nowhere|' kit.yml
  run "${kit_root}/bin/kit" tools install --toolchain kit
  [ "$status" -eq 1 ]
  [[ "$output" == *"beta: nowhere is not in what https://github.com/acme/beta/releases/download/v2.0.0/beta-2.0.0-${platform}.tar.gz unpacks to; set bin: under tools: beta in kit.yml"* ]]
}

@test "kit tools lists each pin and whether it is installed, with no network, and a runner leaves a desk's out" {
  "${kit_root}/bin/kit" tools install --toolchain kit >/dev/null 2>&1
  rm -rf "${KIT_HOME}/.kit/tools/beta"
  before=$(downloads)
  run "${kit_root}/bin/kit" tools --toolchain kit --json
  [ "$status" -eq 0 ]
  [ "$(jq --compact-output --slurp 'map([.name, .pinned, .installed])' <<<"$output")" \
    = '[["alpha","1.0.0","1.0.0"],["beta","2.0.0",null]]' ]
  CI=true run "${kit_root}/bin/kit" tools --toolchain kit --json
  [ "$(jq --compact-output --slurp 'map(.name)' <<<"$output")" = '["alpha"]' ]
  [ "$(downloads)" -eq "${before}" ]
}

@test "the toolchain is active once each directory it names is on PATH" {
  "${kit_root}/bin/kit" tools install --toolchain kit >/dev/null 2>&1
  dirs=$("${kit_root}/bin/kit" tools path --toolchain kit | paste -s -d ':' -)
  run kit-bash 'use toolchain && toolchain:detect'
  grep --quiet --line-regexp $'kit\tproject\tyes\tno' <<<"$output"
  PATH="${dirs}:${PATH}" run kit-bash 'use toolchain && toolchain:detect'
  grep --quiet --line-regexp $'kit\tproject\tyes\tyes' <<<"$output"
}

@test "outdated asks each GitHub release's latest tag, and says which tool's source it cannot read" {
  route gh 'api repos/acme/beta/releases/latest --jq .tag_name' 'beta-tag'
  printf '{"tag_name": "v2.1.0"}\n' | fixture beta-tag
  run "${kit_root}/bin/kit" tools --outdated --toolchain kit
  [ "$status" -eq 1 ]
  # beta is behind, and not installed here
  grep --quiet -E '^🔺 beta +· 2\.0\.0 +· +· 2\.1\.0$' <<<"$output"
  [[ "$output" == *"alpha: kit cannot tell the latest version from https://example.com/alpha/alpha-{version}-{os}-{arch}"* ]]
}

@test "update moves each tool behind to its latest in kit.yml, locks it again, and installs it" {
  sed -i.bak '/^  alpha:/,/^  beta:/{/^  beta:/!d;}' kit.yml
  route gh 'api repos/acme/beta/releases/latest --jq .tag_name' 'beta-tag'
  printf '{"tag_name": "v2.1.0"}\n' | fixture beta-tag
  releases 1.0.0 2.1.0
  run "${kit_root}/bin/kit" tools update --toolchain kit
  [ "$status" -eq 0 ]
  [ "$(yq '.tools.beta.version' kit.yml)" = 2.1.0 ]
  [ "$(grep -c $'^beta\t2\.1\.0\t' tools.lock)" -eq 4 ]
  [ -d "${KIT_HOME}/.kit/tools/beta/2.1.0/${platform}" ]
}

@test "{ext} is filled in from ext:, by the platform's os, so a tool can ship a zip for one and a tarball for another" {
  stage="${BATS_TEST_TMPDIR}/stage-gamma"
  mkdir -p "${stage}/gamma"
  printf '#!/bin/sh\necho gamma\n' > "${stage}/gamma/gamma" && chmod +x "${stage}/gamma/gamma"
  tar --create --gzip --file "${FAKE_RELEASES}/gamma-linux.tar.gz" --directory "${stage}" gamma
  (cd "${stage}" && zip --quiet --recurse-paths "${FAKE_RELEASES}/gamma-macos.zip" gamma)
  cat >> kit.yml <<'YAML'
  gamma:
    version: "3.0.0"
    url: https://example.com/gamma/gamma-{os}.{ext}
    ext: {linux: tar.gz, macos: zip}
    bin: gamma
YAML
  run "${kit_root}/bin/kit" tools install --toolchain kit
  [ "$status" -eq 0 ]
  grep --quiet $'^gamma\t3\.0\.0\tmacos-arm64\tboth\thttps://example\.com/gamma/gamma-macos\.zip\t' tools.lock
  grep --quiet $'^gamma\t3\.0\.0\tlinux-x64\tboth\thttps://example\.com/gamma/gamma-linux\.tar\.gz\t' tools.lock
  [ "$(PATH="$("${kit_root}/bin/kit" tools path --toolchain kit | paste -s -d ':' -):${PATH}" gamma)" = gamma ]
}

@test "outdated reads a GitHub source archive's repository, and a GNU mirror's directory, for the latest version" {
  cat > kit.yml <<'YAML'
tools:
  delta:
    version: "1.0.0"
    url: https://github.com/acme/delta/archive/refs/tags/v{version}.tar.gz
  par:
    version: "20260101"
    url: https://ftp.gnu.org/gnu/par/par-{version}.tar.bz2
YAML
  route gh 'api repos/acme/delta/releases/latest --jq .tag_name' 'delta-tag'
  printf '{"tag_name": "v1.2.0"}\n' | fixture delta-tag
  printf '<a href="par-20251201.tar.bz2">x</a> <a href="par-20260301.tar.bz2">x</a>\n<a href="par-20260301.tar.bz2.sig">s</a>\n' \
    > "${FAKE_RELEASES}/par"
  run "${kit_root}/bin/kit" tools --outdated --toolchain kit
  [ "$status" -eq 0 ]
  grep --quiet -E '^🔺 delta +· 1\.0\.0 +· +· 1\.2\.0$' <<<"$output"
  grep --quiet -E '^🔺 par +· 20260101 +· +· 20260301$' <<<"$output"
}

# ----------------------------------------------------------------------------------------------------------------------
