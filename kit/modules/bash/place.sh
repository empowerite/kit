# shellcheck shell=bash
# place.sh - a delivered path: written, merged by key, compared with the release, and taken out again (sourced module)
# ----------------------------------------------------------------------------------------------------------------------
# shellcheck disable=SC2016  # jq filters name jq variables with $, in single quotes on purpose
# shellcheck disable=SC2154  # kit_root and kit_jq are exported by modules/bash/1st.sh
#
# How a delivered file is placed depends on its destination:
#
#   - a destination ending in .json (except *.schema.json) is merged by key; the kit owns only the entries it
#     writes
#   - a region is regenerated, never merged
#   - a path under .kit/ is materialized: a link to its source, or a read-only copy (materialize.sh)
#   - every other file is copied
#
# place:delivered lists what the active packages deliver under a base, "<path>\t<source>", the last layer winning
# where two deliver the same path. place:remove and place:retire take the kit's content out again.
#
# see: docs/packages.md

use release render region materialize manifest file

# place:kind <source> <path> prints how the content is placed:
#
#   region   the source is under a package's regions/<base>/
#   merge    the destination is a merged .json file (see place:merged)
#   copy     anything else
#
# The region test looks only at the source's last packages/ segment, so a checkout under a directory named packages
# works the same as one anywhere else.
place:kind() {
  if [[ "${1##*/packages/}" =~ ^[^/]+(/[^/]+)?/regions/(repo|home)/ ]]
  then printf 'region\n'
  elif place:merged "$2"
  then printf 'merge\n'
  else printf 'copy\n'
  fi
}

# place:delivered <root> <base> <package>… prints "<path>\t<source>" for every path under the base's root that
# the release at <root> delivers for those packages, sorted by path. When layers deliver the same path, the last
# layer wins. A symlink counts as a source only when it points at a file. A dangling symlink fails here, where it
# is found.
place:delivered() {
  local layer path src
  while IFS= read -r layer
  do
    (cd "${layer}" && find . \( -type f -o -type l \) | sed 's@^\./@@' \
      | awk -v l="${layer}" '{ print $0 "\t" l "/" $0 }')
  done < <(render:layers "$@") | awk -F '\t' '{ src[$1] = $2 } END { for (p in src) print p "\t" src[p] }' \
    | LC_ALL=C sort \
    | while IFS=$'\t' read -r path src
  do
    if [[ -L "${src}" && ! -e "${src}" ]]
    then
      error "${src} is a link to nothing; the release is broken"
      return 1
    fi
    printf '%s\t%s\n' "${path}" "${src}"
  done
}

# place:source-of <root> <base> <path> <package>… prints the release file behind one delivered path, or nothing.
place:source-of() {
  local root=$1 base=$2 path=$3
  shift 3
  place:delivered "${root}" "${base}" "$@" | awk -F '\t' -v p="${path}" '$1 == p { print $2 }'
}

# place:beyond <root> <base> <before> <after> prints the paths under the base that the package set <after>
# delivers and <before> does not. Each set is passed as one space-separated word.
place:beyond() {
  local root=$1 base=$2
  local -a before after
  read -r -a before <<<"$3"
  read -r -a after <<<"$4"
  LC_ALL=C comm -13 <(place:delivered "${root}" "${base}" "${before[@]}" | cut -f 1) \
    <(place:delivered "${root}" "${base}" "${after[@]}" | cut -f 1)
}

# place:merged <path> succeeds for a destination that is merged by key instead of copied: a .json settings
# file. Every .json file the kit delivers is a settings file, except a schema (*.schema.json).
place:merged() { [[ "$1" == *.json && "$1" != *.schema.json ]]; }

# place:json-write <path> <filter> [<jq option>…] rewrites the JSON at <path> through <filter>, which can call
# modules/jq/merge.jq as merge::. It creates the file, as {}, when it is missing. A file jq cannot read, such as one
# with comments, is named, left as it was, and fails.
place:json-write() {
  local path filter=$2
  path=$(release:target "$1")
  shift 2
  mkdir -p "$(dirname "${path}")"
  [[ -f "${path}" ]] || printf '{}\n' > "${path}"
  if ! file:replace "${path}" jq -L "${kit_jq}" --indent 2 "import \"merge\" as merge; ${filter}" "$@" "${path}"
  then
    error "${path}: not JSON jq can read, so the kit's entries in it were not changed; it is left as it was"
    return 1
  fi
}

# place:json-holds <path> <source> succeeds when the JSON at <path> contains every entry that the release's
# <source> writes.
place:json-holds() {
  jq -L "${kit_jq}" --exit-status 'import "merge" as merge; merge::holds(.; $new[0])' --slurpfile new "$2" "$1" \
    >/dev/null
}

# place:content <source> <path> [<base-source>] writes the release's content to <path>. It merges a .json file,
# writes the region for a region source, brings a materialized path under .kit/ to its state (materialize:owned), and
# otherwise copies the file, keeping the source's mode.
place:content() {
  local src=$1 path=$2 base=${3:-/dev/null}
  case "$(place:kind "${src}" "${path}")" in
    (merge)
      place:json-write "${path}" 'merge::apply(.; $base[0]; $new[0])' --slurpfile new "${src}" --slurpfile base "${base}"
      ;;
    (region)
      region:write "${path}" "${src}"
      ;;
    (*)
      if materialize:owned repo "${path}"
      then
        materialize:one "${src}" "${path}" >/dev/null
      else
        file:copy "${src}" "$(release:target "${path}")"
      fi
  esac
}

# place:installed <source> <path> prints the repository's copy of the release's content at <path>: the file,
# or the region's content. It prints nothing and fails when there is none.
place:installed() {
  local src=$1 path=$2
  case "$(place:kind "${src}" "${path}")" in
    (region) region:read "${path}" ;;
    (*) [[ -f "${path}" ]] && cat "${path}" ;;
  esac
}

# place:holds <source> <path> succeeds when the repository has the release's content at <path>: the same file,
# every merged entry, or the same region.
place:holds() {
  local src=$1 path=$2
  case "$(place:kind "${src}" "${path}")" in
    (merge) [[ -f "${path}" ]] && place:json-holds "${path}" "${src}" ;;
    (*) cmp --silent "${src}" <(place:installed "${src}" "${path}") ;;
  esac
}

# place:new <source> <path> [<label>] places content the repository has not recorded. It writes the
# content when:
#
#   - nothing is at <path>
#   - the destination is a .json file to merge into
#   - the file exists but has no region yet, for a region source
#
# When the same content is already there, it reports that and writes nothing. A region already there is
# regenerated. A different file fails, and is named and left as it is.
place:new() {
  local src=$1 path=$2 label=${3:-} kind
  kind=$(place:kind "${src}" "${path}")
  if [[ -e "${path}" && ! -f "${path}" ]]
  then
    error "${path}: a directory, or something that is not a file, is there; move it aside"
    return 1
  fi
  if [[ "${kind}" == merge || ! -f "${path}" ]] \
    || { [[ "${kind}" == region ]] && ! place:installed "${src}" "${path}" >/dev/null; }
  then
    step "${path}${label:+: ${label}}"
    place:holds "${src}" "${path}" 2>/dev/null || place_placed+=("${path}")
    place:content "${src}" "${path}"
  elif place:holds "${src}" "${path}"
  then
    step "${path}${label:+: ${label}}, already as the release has it"
  elif [[ "${kind}" == region ]]
  then
    step "${path}${label:+: ${label}}, the region regenerated"
    place:content "${src}" "${path}"
  else
    error "${path}: a different file is there already; move it aside, or remove it if it is not yours"
    return 1
  fi
}

# place_placed holds each path this process placed afresh, or merged an entry into that was not there, so that
# release:say-notices tells the person only about what changed.
declare -ga place_placed=()

# place:all <root> <base> <package>… places every path the packages deliver under the base. A path where a
# different file exists fails. It returns the number of failures.
place:all() {
  local root=$1 base=$2 path src refused=0
  shift 2
  while IFS=$'\t' read -r path src
  do
    [[ -n "${path}" ]] || continue
    place:new "${src}" "$(release:dest "${base}" "${path}")" || refused=$(( refused + 1 ))
  done < <(place:delivered "${root}" "${base}" "$@")
  return "${refused}"
}

# place:remove <source> <path> removes the release's content from <path>: the kit's entries from a .json
# file, the kit's region from a region file, and otherwise the file itself. A region file that holds no region is
# left as it is, and nothing is said of it. It fails when it cannot take the content out, as for a region whose
# markers region:has refuses, and the file is left as it is.
place:remove() {
  local src=$1 path=$2 status=0
  case "$(place:kind "${src}" "${path}")" in
    (merge)
      [[ -f "${path}" ]] || return 0
      step "${path}: the kit's entries taken out"
      place:json-write "${path}" 'merge::remove(.; $base[0])' --slurpfile base "${src}"
      ;;
    (region)
      region:has "${path}" "$(release:target "${path}")" || status=$?
      (( status != 2 )) || return 1
      (( status == 0 )) || return 0
      region:remove "${path}" "${src}" || return $?
      step "${path}: the kit's region taken out"
      ;;
    (*)
      step "removing ${path}"
      rm -f "${path}" || return 1
      if materialize:owned repo "${path}"
      then
        materialize:prune "${path}"
      fi
  esac
}

# place:edited <base> <path> <source> succeeds when the delivered <path> exists as a copy the repository owns and
# differs from the release's <source>, which means the repository edited it since the kit placed it. A path under
# .kit/ is the kit's own, so it is never edited.
place:edited() {
  local base=$1 path=$2 src=$3 dest
  dest=$(release:dest "${base}" "${path}")
  [[ -e "${dest}" || -L "${dest}" ]] || return 1
  [[ "$(place:kind "${src}" "${dest}")" == copy ]] || return 1
  ! materialize:owned "${base}" "${path}" || return 1
  ! cmp --silent "${src}" "${dest}"
}

# place:refusals <base> <package>… reads the paths under the base that a removal would take out, one per line, and
# judges the markers of each the packages deliver as a region, as region:remove judges them, so a removal refuses
# before it takes anything out. region:has names each file it refuses. It prints how many it refuses.
place:refusals() {
  local base=$1 path src dest status refused=0
  shift
  while IFS=$'\t' read -r path src
  do
    [[ "$(place:kind "${src}" "${path}")" == region ]] || continue
    dest=$(release:dest "${base}" "${path}")
    status=0
    region:has "${dest}" "$(release:target "${dest}")" || status=$?
    (( status != 2 )) || refused=$(( refused + 1 ))
  done < <(awk -F '\t' 'NR == FNR { wanted[$0]; next } $1 in wanted' - <(place:delivered "${kit_root}" "${base}" "$@"))
  printf '%s\n' "${refused}"
}

# place:retire <base-root> <base> <previous> <current> <package>… handles the paths that the base's manifest
# records and the current release no longer delivers. <base-root> is the base release (the previous version), or
# empty when it is not available. For each such path that still exists:
#
#   - a merged .json file: the previous release's entries are removed when the base release is available;
#     otherwise it warns and leaves the file as it is
#   - any other file: it is removed when its content still matches the digest the manifest records; otherwise it
#     warns that the file was edited and leaves it as it is
#
# Under home, whose manifest every repository shares, it touches only paths the previous release delivered for
# these packages, and nothing without the base release.
place:retire() {
  local base_root=$1 base=$2 previous=$3 current=$4 path dest previous_src candidates
  shift 4
  candidates=$(LC_ALL=C comm -23 <(manifest:paths "${base}" | LC_ALL=C sort) \
    <(place:delivered "${kit_root}" "${base}" "$@" | cut -f 1 | LC_ALL=C sort))
  if [[ "${base}" != repo ]]
  then
    [[ -n "${base_root}" ]] || return 0
    candidates=$(LC_ALL=C comm -12 <(printf '%s\n' "${candidates}" | LC_ALL=C sort) \
      <(place:delivered "${base_root}" "${base}" "$@" | cut -f 1 | LC_ALL=C sort))
  fi
  while IFS= read -r path
  do
    [[ -n "${path}" ]] || continue
    dest=$(release:dest "${base}" "${path}")
    [[ -e "${dest}" ]] || continue
    previous_src=""
    [[ -n "${base_root}" ]] && previous_src=$(place:source-of "${base_root}" "${base}" "${path}" "$@")
    if place:merged "${dest}"
    then
      if [[ -n "${previous_src}" ]]
      then
        step "${dest}: retired in ${current}, the previous release's entries taken out"
        place:json-write "${dest}" 'merge::remove(.; $base[0])' --slurpfile base "${previous_src}"
      else
        warn "${dest}: retired in ${current}; without the base release its entries stay, and it is left as it was"
      fi
    elif materialize:owned "${base}" "${path}"
    then
      # A materialized path is kit's link or copy, never the repository's edit, so it goes whatever it holds.
      step "${dest}: retired in ${current}, removed"
      rm -f "${dest}"
    elif [[ "$(manifest:recorded "${base}" "${path}")" \
      == "$(place:installed "${previous_src:-/dev/null}" "${dest}" | file:digest)" ]]
    then
      step "${dest}: retired in ${current}, removed"
      place:remove "${previous_src:-/dev/null}" "${dest}"
    else
      warn "${dest}: retired in ${current}, but edited since ${previous}; it is left as it was"
    fi
  done <<<"${candidates}"
}

# ----------------------------------------------------------------------------------------------------------------------
