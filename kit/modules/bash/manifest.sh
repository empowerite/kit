# shellcheck shell=bash
# manifest.sh - .kit/manifest and ~/.kit/manifest: the digest of every delivered path, per base (sourced module)
# ----------------------------------------------------------------------------------------------------------------------
# shellcheck disable=SC2154  # kit_root is exported by modules/bash/1st.sh
#
# Each base has a manifest, written beside what kit placed and ignored like it: one line per delivered path, the
# release's digest of its content and the path. kit update compares a file against its digest to tell whether the
# repository edited it. Home's manifest, with no settings file beside it, carries the version and the packages as
# well, and is shared by every repository on the machine.
#
# see: docs/packages.md

use release place settings file

# manifest:path <base> prints the base's manifest: .kit/manifest in the repository, ~/.kit/manifest for home.
manifest:path() { release:dest "$1" .kit/manifest; }

# manifest:write <base> <excluded> <package>… writes the base's manifest: the digest of the release's file
# for every path delivered under the base, and under home, which has no settings file, the version and the active
# packages as well. The repository's manifest is always written. Home's manifest is shared by every repository on
# the machine. It is written when these packages deliver to home. When they do not, it drops the lines whose paths
# no longer exist, and removes the file once no line is left.
manifest:write() {
  local base=$1 excluded=$2 manifest path src delivered kept
  local -a paths
  shift 2
  manifest=$(manifest:path "${base}")
  delivered=$(place:delivered "${kit_root}" "${base}" "$@") || return 1
  if [[ "${base}" != repo && -z "${delivered}" ]]
  then
    [[ -f "${manifest}" ]] || return 0
    list:read paths manifest:paths "${base}" || return 1
    # A path that is gone, or that grep does not find, keeps no line.
    kept=$(for path in "${paths[@]}"
      do
        [[ -n "${path}" && -e "$(release:dest "${base}" "${path}")" ]] && grep -F "  ${path}" "${manifest}"
      done; true)
    if [[ -z "${kept}" ]]
    then
      rm -f "${manifest}"
    else
      file:replace --read-only "${manifest}" manifest:kept-text "${manifest}" "${kept}"
    fi
    return 0
  fi
  # The manifest has no write bit, as every file kit places under .kit/ has none.
  file:replace --read-only "${manifest}" manifest:text "${base}" "${excluded}" "${delivered}" "$@"
}

# manifest:kept-text <manifest> <kept> prints home's manifest with its header lines and the record lines in <kept>.
manifest:kept-text() {
  awk '!(length($1) == 64 && $0 ~ /^[0-9a-f]+  /)' "$1"
  printf '%s\n' "$2"
}

# manifest:text <base> <excluded> <delivered> <package>… prints a manifest: its header; under home, the
# version, the packages, and the excludes; and one record per "<path>\t<source>" line of <delivered>, the source's
# digest and the path.
manifest:text() {
  local base=$1 excluded=$2 delivered=$3 path src
  shift 3
  printf '# .kit/manifest - written by kit, not by hand\n'
  if [[ "${base}" != repo ]]
  then
    # Home has no settings file, so its manifest carries the version and the packages that wrote it.
    printf 'version %s\n' "$(release:version)"
    if (( $# > 0 ))
    then printf 'packages %s\n' "$*"
    else printf 'packages\n'
    fi
    [[ -z "${excluded}" ]] || printf 'excludes %s\n' "${excluded}"
  fi
  while IFS=$'\t' read -r path src
  do
    # Use if, not &&: as the last command in the loop body, a false && list would end the script under errexit.
    if [[ -n "${path}" ]]
    then
      printf '%s  %s\n' "$(file:digest < "${src}")" "${path}"
    fi
  done <<<"${delivered}"
}

# manifest:write-all <excluded> <package>… writes every base's manifest, then the repository's declaration in
# kit.yml. <excluded> is the excluded packages as a space-separated list, empty when there are none.
manifest:write-all() {
  local excluded=$1 base
  shift
  for base in $(release:bases)
  do
    manifest:write "${base}" "${excluded}" "$@"
  done
  settings:declare "$(release:version)" "${excluded}" "$@"
}

# manifest:paths <base> prints every path the base's manifest records. A record line is a 64-character
# hex digest, two spaces, and the path, which can contain spaces. A record whose path climbs out of the base's root,
# by a leading /, or a . or .. segment, is skipped and named: a manifest a clone carries is not kit's to trust, and
# every path read here is one a verb may remove.
manifest:paths() {
  local manifest path
  manifest=$(manifest:path "$1")
  [[ -f "${manifest}" ]] || return 0
  while IFS= read -r path
  do
    if [[ "${path}" == /* || "/${path}/" == */../* || "/${path}/" == */./* ]]
    then
      warn "${manifest}: the record for '${path}' climbs out of its root; skipped"
      continue
    fi
    printf '%s\n' "${path}"
  done < <(awk 'length($1) == 64 && $0 ~ /^[0-9a-f]+  / { print substr($0, 67) }' "${manifest}")
}

# manifest:recorded <base> <path> prints the digest the base's manifest records for <path>, or nothing.
manifest:recorded() {
  local manifest
  manifest=$(manifest:path "$1")
  [[ -f "${manifest}" ]] || return 0
  awk -v p="$2" 'length($1) == 64 && substr($0, 67) == p { print $1 }' "${manifest}"
}

# ----------------------------------------------------------------------------------------------------------------------
