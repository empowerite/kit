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
  run "${kit_root}/bin/kit" tools install --toolchain=kit
  [ "$status" -eq 0 ]
  [[ "$output" == *"tools.lock: locked alpha, beta"* ]]
  # a row per tool and platform, each with what the file is and its hash, and never where it came from
  [ "$(grep -c -v '^#' tools.lock)" -eq 8 ]
  while IFS=$'\t' read -r _ _ _ _ artifact hash _
  do
    [ "${hash}" = "sha256:$(shasum -a 256 "${FAKE_RELEASES}/${artifact#file:}" | cut -d ' ' -f 1)" ]
  done < <(grep -v '^#' tools.lock)
  t=$'\t'
  grep --quiet -E "^beta${t}2\.0\.0${t}linux-x64${t}desk${t}file:beta-2\.0\.0-linux-x64\.tar\.gz${t}sha256:[0-9a-f]{64}\
${t}beta-2\.0\.0/bin$" tools.lock
  run ! grep --quiet 'https://' tools.lock
  run "${kit_root}/bin/kit" tools path --toolchain=kit
  [ "$status" -eq 0 ]
  [ "$(wc -l <<<"$output" | tr -d ' ')" -eq 2 ]
  [ "$(PATH="$(paste -s -d ':' - <<<"$output"):${PATH}" alpha)" = "alpha 1.0.0 ${platform}" ]
  [ "$(PATH="$(paste -s -d ':' - <<<"$output"):${PATH}" beta)" = "beta 2.0.0 ${platform}" ]
  # the cache keeps this platform's downloads alone; the others were fetched only to be hashed
  [ "$(find "${XDG_CACHE_HOME}/kit/downloads" -type f | wc -l | tr -d ' ')" -eq 2 ]
}

@test "a url every platform shares is downloaded once while locking" {
  printf '#!/bin/sh\necho omega\n' > "${FAKE_RELEASES}/omega"
  cat >> kit.yml <<'YAML'
  omega:
    version: "1"
    url: https://example.com/omega/{version}/omega
YAML
  run "${kit_root}/bin/kit" tools install --toolchain=kit
  [ "$status" -eq 0 ]
  [ "$(grep -c '/omega$' "${FAKE_RELEASES}/curl.calls")" -eq 1 ]
}

@test "an install with the lock up to date and the tools in place downloads nothing" {
  "${kit_root}/bin/kit" tools install --toolchain=kit >/dev/null 2>&1
  before=$(downloads)
  run "${kit_root}/bin/kit" tools install --toolchain=kit
  [ "$status" -eq 0 ]
  [ "$(downloads)" -eq "${before}" ]
  [[ "$output" != *"locked"* ]]
}

@test "a pin that moves is locked again at a desk, and the tools of other pins are left as they were" {
  "${kit_root}/bin/kit" tools install --toolchain=kit >/dev/null 2>&1
  beta_rows=$(grep '^beta' tools.lock)
  releases 1.1.0 2.0.0
  pins 1.1.0 2.0.0
  run "${kit_root}/bin/kit" tools install --toolchain=kit
  [ "$status" -eq 0 ]
  [[ "$output" == *"tools.lock: locked alpha"* ]]
  [ "$(grep '^beta' tools.lock)" = "${beta_rows}" ]
  [ "$(grep -c $'^alpha\t1\.1\.0\t' tools.lock)" -eq 4 ]
  [ "$(grep -c $'^alpha\t1\.0\.0\t' tools.lock)" -eq 0 ]
}

@test "a runner refuses a lock behind kit.yml, naming the tool, and installs none of a desk's tools" {
  "${kit_root}/bin/kit" tools install --toolchain=kit >/dev/null 2>&1
  releases 1.1.0 2.0.0
  pins 1.1.0 2.0.0
  CI=true run "${kit_root}/bin/kit" tools install --toolchain=kit
  [ "$status" -eq 1 ]
  [[ "$output" == *"tools.lock is behind tools: in kit.yml for alpha; run kit tools install at a desk"* ]]
  # with the lock up to date, a runner installs alpha alone: beta is a desk's
  pins 1.0.0 2.0.0
  rm -rf "${KIT_HOME}/.kit/tools"
  CI=true run "${kit_root}/bin/kit" tools install --toolchain=kit
  [ "$status" -eq 0 ]
  [ -d "${KIT_HOME}/.kit/tools/alpha/1.0.0/${platform}" ]
  [ ! -d "${KIT_HOME}/.kit/tools/beta" ]
  CI=true run "${kit_root}/bin/kit" tools path --toolchain=kit
  [ "$output" = "${KIT_HOME}/.kit/tools/alpha/1.0.0/${platform}" ]
}

@test "a download whose bytes are not the ones the lock pins is refused, naming both digests, and nothing is installed" {
  "${kit_root}/bin/kit" tools install --toolchain=kit >/dev/null 2>&1
  rm -rf "${KIT_HOME}/.kit/tools" "${XDG_CACHE_HOME}/kit/downloads"
  os=${platform%-*} arch=${platform#*-}
  [[ "${os}" == macos ]] && os=darwin
  printf '#!/bin/sh\necho tampered\n' > "${FAKE_RELEASES}/alpha-1.0.0-${os}-${arch}"
  run "${kit_root}/bin/kit" tools install --toolchain=kit
  [ "$status" -eq 1 ]
  [[ "$output" =~ direct\ \(https://example\.com/alpha/alpha-1\.0\.0-[a-z0-9-]+\)\ serves\ alpha-1\.0\.0-[a-z0-9-]+\ with\ the\ hash\ sha256:[0-9a-f]{64},\ not\ the\ sha256:[0-9a-f]{64}\ the\ lock\ pins ]]
  [[ "$output" == *"alpha: alpha-1.0.0-${os}-${arch} could not be downloaded; nothing was installed"* ]]
  [ ! -d "${KIT_HOME}/.kit/tools/alpha" ]
}

@test "a bin: that is not in what the download unpacks to is refused, naming the tool and its bin:" {
  sed -i.bak 's|bin: beta-{version}/bin|bin: nowhere|' kit.yml
  run "${kit_root}/bin/kit" tools install --toolchain=kit
  [ "$status" -eq 1 ]
  [[ "$output" == *"beta: nowhere is not in what beta-2.0.0-${platform}.tar.gz unpacks to; set bin: under tools: beta in kit.yml"* ]]
}

@test "a bin: that holds no command named as the tool is refused, and nothing is installed" {
  sed -i.bak 's|bin: beta-{version}/bin|bin: beta-{version}|' kit.yml
  run "${kit_root}/bin/kit" tools install --toolchain=kit
  [ "$status" -eq 1 ]
  [[ "$output" == *"beta: beta-2.0.0 in what beta-2.0.0-${platform}.tar.gz unpacks to holds no beta command; set bin: under tools: beta in kit.yml"* ]]
  [ ! -d "${KIT_HOME}/.kit/tools/beta" ]
}

@test "kit tools lists each pin and whether it is installed, with no network, and a runner leaves a desk's out" {
  "${kit_root}/bin/kit" tools install --toolchain=kit >/dev/null 2>&1
  rm -rf "${KIT_HOME}/.kit/tools/beta"
  before=$(downloads)
  run "${kit_root}/bin/kit" tools --toolchain=kit --json
  [ "$status" -eq 0 ]
  [ "$(jq --compact-output --slurp 'map([.name, .pinned, .installed])' <<<"$output")" \
    = '[["alpha","1.0.0","1.0.0"],["beta","2.0.0",null]]' ]
  CI=true run "${kit_root}/bin/kit" tools --toolchain=kit --json
  [ "$(jq --compact-output --slurp 'map(.name)' <<<"$output")" = '["alpha"]' ]
  [ "$(downloads)" -eq "${before}" ]
}

@test "a verb that runs the repository's tools outside kit gate finds them on the chosen toolchain's path" {
  "${kit_root}/bin/kit" tools install --toolchain=kit >/dev/null 2>&1
  KIT_TOOLCHAIN=kit run kit-bash 'use dependencies && dependencies:onto-path && command -v alpha'
  [ "$status" -eq 0 ]
  [ "$output" = "${KIT_HOME}/.kit/tools/alpha/1.0.0/${platform}/alpha" ]
}

@test "the toolchain is active once each directory it names is on PATH" {
  "${kit_root}/bin/kit" tools install --toolchain=kit >/dev/null 2>&1
  dirs=$("${kit_root}/bin/kit" tools path --toolchain=kit | paste -s -d ':' -)
  run kit-bash 'use toolchain && toolchain:detect'
  grep --quiet --line-regexp $'kit\tproject\tyes\tno' <<<"$output"
  PATH="${dirs}:${PATH}" run kit-bash 'use toolchain && toolchain:detect'
  grep --quiet --line-regexp $'kit\tproject\tyes\tyes' <<<"$output"
}

@test "outdated asks each GitHub release's latest tag, and says which tool's source it cannot read" {
  route gh 'api repos/acme/beta/releases/latest --jq .tag_name' 'beta-tag'
  printf '{"tag_name": "v2.1.0"}\n' | fixture beta-tag
  run "${kit_root}/bin/kit" tools --outdated --toolchain=kit
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
  run "${kit_root}/bin/kit" tools update --toolchain=kit
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
  run "${kit_root}/bin/kit" tools install --toolchain=kit
  [ "$status" -eq 0 ]
  grep --quiet $'^gamma\t3\.0\.0\tmacos-arm64\tboth\tfile:gamma-macos\.zip\t' tools.lock
  grep --quiet $'^gamma\t3\.0\.0\tlinux-x64\tboth\tfile:gamma-linux\.tar\.gz\t' tools.lock
  [ "$(PATH="$("${kit_root}/bin/kit" tools path --toolchain=kit | paste -s -d ':' -):${PATH}" gamma)" = gamma ]
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
  run "${kit_root}/bin/kit" tools --outdated --toolchain=kit
  [ "$status" -eq 0 ]
  grep --quiet -E '^🔺 delta +· 1\.0\.0 +· +· 1\.2\.0$' <<<"$output"
  grep --quiet -E '^🔺 par +· 20260101 +· +· 20260301$' <<<"$output"
}

@test "a lock written with each url and a bare sha256 is read as the artifact and its hash, so a runner takes it" {
  "${kit_root}/bin/kit" tools install --toolchain=kit >/dev/null 2>&1
  awk -F '\t' -v OFS='\t' '/^#/ { print; next } $1 == "alpha" { $5 = "https://example.com/alpha/" substr($5, 6) }
    $1 == "beta" { $5 = "https://github.com/acme/beta/releases/download/v2.0.0/" substr($5, 6) }
    { sub(/^sha256:/, "", $6); print }' tools.lock > old.lock && mv old.lock tools.lock
  grep --quiet $'\thttps://example.com/alpha/alpha-1.0.0-' tools.lock
  rm -rf "${KIT_HOME}/.kit/tools"
  CI=true run "${kit_root}/bin/kit" tools install --toolchain=kit
  [ "$status" -eq 0 ]
  [[ "$output" != *"behind"* ]]
  [ "$(PATH="$(CI=true "${kit_root}/bin/kit" tools path --toolchain=kit):${PATH}" alpha)" = "alpha 1.0.0 ${platform}" ]
  run "${kit_root}/bin/kit" tools install --toolchain=kit
  [ "$status" -eq 0 ]
  grep --quiet 'https://' tools.lock
}

@test "a download is found on the pull path: a path that ends at off before direct installs no url, and says so" {
  KIT_SOURCES=off run "${kit_root}/bin/kit" tools install --toolchain=kit
  [ "$status" -eq 1 ]
  [[ "$output" == *"no source on the pull path served alpha-1.0.0-"*": the path ends at off; see: kit tools sources"* ]]
  [ ! -e tools.lock ]
  [ "$(downloads)" -eq 0 ]
  KIT_SOURCES=direct,off run "${kit_root}/bin/kit" tools install --toolchain=kit
  [ "$status" -eq 0 ]
}

@test "a download direct cannot find is named, with what direct said" {
  rm "${FAKE_RELEASES}/alpha-1.0.0-"*
  run "${kit_root}/bin/kit" tools install --toolchain=kit
  [ "$status" -eq 1 ]
  [[ "$output" == *"no source on the pull path served alpha-1.0.0-"*": direct (https://example.com/alpha/alpha-1.0.0-"*") does not have it"* ]]
}

# ----------------------------------------------------------------------------------------------------------------------
