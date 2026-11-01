# shellcheck shell=bash
# tui.sh - finds kittui, the binary kit tui runs: the release's own, one on PATH, or one built from tui/ in a checkout
# ----------------------------------------------------------------------------------------------------------------------
# shellcheck disable=SC2154  # kit_root is exported by modules/bash/1st.sh
#
# A release carries kittui for each platform it supports in libexec/, built by bin/release pack, so whatever installs
# the kit installs kittui at the same version. A checkout of the kit has tui/ beside kit/ instead, and kittui is
# built from it, once per change to its sources or to go, into the cache; a kittui on PATH is the last resort.

use file

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

# tui:checkout prints the tui/ directory of a checkout of the kit, and fails for a release, which has a version file
# and no tui/ beside it.
tui:checkout() {
  [[ ! -f "${kit_root}/version" && -f "${kit_root}/../tui/go.mod" ]] || return 1
  (cd "${kit_root}/../tui" && pwd)
}

# tui:cached <tui> prints where the build of the sources under <tui>, with the go on PATH, lives in the cache: named by
# a digest of the Go sources and of go version, so a change to either is a new build.
tui:cached() {
  local digest
  digest=$( { find "$1" -type f \( -name '*.go' -o -name go.mod -o -name go.sum \) | LC_ALL=C sort | xargs cat
    go version; } | file:digest)
  printf '%s\n' "${XDG_CACHE_HOME:-${HOME}/.cache}/kit/kittui-${digest:0:16}"
}

# tui:binary [--probe] prints the path of the kittui to run. In a checkout with go, it is the build of tui/'s sources,
# built first when they or go changed, under a temporary name and then renamed into place, so two first runs never
# run a half-written file. Otherwise it is the release's own for this platform, else kittui on PATH. With --probe it
# only says whether one can be had, and builds nothing, as the hand-off asks. It says why there is none, with the
# remedy, and returns 1.
tui:binary() {
  local probe=false platform tui out tmp
  [[ "${1:-}" == --probe ]] && probe=true
  if tui=$(tui:checkout) && command -v go >/dev/null 2>&1
  then
    out=$(tui:cached "${tui}")
    if [[ ! -x "${out}" && "${probe}" == false ]]
    then
      step "building kittui from tui/, once for these sources and this go"
      tmp="${out}.tmp.$$"
      mkdir -p "$(dirname "${out}")"
      if ! (cd "${tui}" && CGO_ENABLED=0 go build -o "${tmp}" ./cmd/kittui)
      then
        rm -f "${tmp}"
        error "kittui could not be built from ${tui}; see above"
        return 1
      fi
      mv -f "${tmp}" "${out}"
    fi
    printf '%s\n' "${out}"
    return 0
  fi
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
  if tui:checkout >/dev/null
  then
    guard:require tool:go
  elif [[ -f "${kit_root}/version" ]]
  then
    error "this release of kit carries no kittui for ${platform:-$(uname -sm)}; run: kit help tui"
  else
    error "run: mise install (kittui is not on PATH)"
  fi
  return 1
}

# ----------------------------------------------------------------------------------------------------------------------
