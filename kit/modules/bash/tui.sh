# shellcheck shell=bash
# tui.sh - finds kittui, the binary kit tui runs: the release's own, one on PATH, or one built from tui/ in a checkout
# ----------------------------------------------------------------------------------------------------------------------
# shellcheck disable=SC2154  # kit_root is exported by modules/bash/1st.sh
#
# A release carries kittui for each platform it supports in libexec/, built by bin/release pack, so whatever installs
# the kit installs kittui at the same version. A checkout of the kit has tui/ beside kit/ instead, and kittui is
# built from it, once per change to its sources, into the cache.

use release

# tui:platform prints this machine's platform as Go names it: linux-amd64, darwin-arm64. It returns 1 on an
# operating system or architecture no release builds for.
tui:platform() {
  local os arch
  os=$(uname -s | tr '[:upper:]' '[:lower:]')
  arch=$(uname -m)
  case "${arch}" in
    (x86_64 | amd64) arch=amd64 ;;
    (aarch64 | arm64) arch=arm64 ;;
    (*) return 1
  esac
  case "${os}" in
    (linux | darwin) ;;
    (*) return 1
  esac
  printf '%s-%s\n' "${os}" "${arch}"
}

# tui:binary prints the path of the kittui to run: the release's own for this platform, else kittui on PATH, else, in
# a checkout, the one built from tui/, building it first when its sources changed. It says why there is none, with
# the remedy, and returns 1.
tui:binary() {
  local platform tui digest out
  platform=$(tui:platform || true)
  if [[ -n "${platform}" && -x "${kit_root}/libexec/kittui-${platform}" ]]
  then
    printf '%s\n' "${kit_root}/libexec/kittui-${platform}"
    return 0
  fi
  if command -v kittui >/dev/null 2>&1
  then
    command -v kittui
    return 0
  fi
  tui=${kit_root}/../tui
  if [[ -f "${tui}/go.mod" ]]
  then
    guard:require tool:go || return 1
    digest=$(find "${tui}" -type f \( -name '*.go' -o -name go.mod -o -name go.sum \) | LC_ALL=C sort \
      | xargs cat | release:digest)
    out="${XDG_CACHE_HOME:-${HOME}/.cache}/kit/kittui-${digest:0:16}"
    if [[ ! -x "${out}" ]]
    then
      step "building kittui from tui/, once for these sources"
      if ! (cd "${tui}" && CGO_ENABLED=0 go build -o "${out}" ./cmd/kittui)
      then
        error "kittui could not be built from ${tui}; see above"
        return 1
      fi
    fi
    printf '%s\n' "${out}"
    return 0
  fi
  if [[ -f "${kit_root}/version" ]]
  then
    error "this release of kit carries no kittui for ${platform:-$(uname -sm)}; run: kit help tui"
  else
    error "run: mise install (kittui is not on PATH)"
  fi
  return 1
}

# ----------------------------------------------------------------------------------------------------------------------
