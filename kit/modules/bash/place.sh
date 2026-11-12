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

# place_delivered maps a key, "<root>|<base>|<package> …" and a tab, to the rows place:delivered prints for it, and
# place_source maps that key and a delivered path to the path's source. place:delivered fills both in the
# shell that calls it, and every subshell under it inherits them, so a verb asks once and its subshells answer with no
# fork: render:packages asks for each base in the verb's own shell. A release's layers do not change while a process
# runs, since render:package renders each once.
declare -gA place_delivered=() place_source=()

# place:key <name> <root> <base> <package>… sets the variable <name> to the key place_delivered keeps the rows under.
place:key() {
  local IFS=' '
  printf -v "$1" '%s|%s|%s\t' "$2" "$3" "${*:4}"
}

# place:kind <source> <path> [<name>] prints how the content is placed, or, given a <name>, sets that variable to it,
# with no fork:
#
#   region   the source is under a package's regions/<base>/
#   merge    the destination is a merged .json file (see place:merged)
#   copy     anything else
#
# The region test looks only at the source's last packages/ segment, so a checkout under a directory named packages
# works the same as one anywhere else.
place:kind() {
  local place_kind_of=copy
  if [[ "${1##*/packages/}" =~ ^[^/]+(/[^/]+)?/regions/(repo|home)/ ]]
  then place_kind_of=region
  elif place:merged "$2"
  then place_kind_of=merge
  fi
  if [[ -n "${3:-}" ]]
  then printf -v "$3" '%s' "${place_kind_of}"
  else printf '%s\n' "${place_kind_of}"
  fi
}

# place:delivered <root> <base> <package>… prints "<path>\t<source>" for every path under the base's root that
# the release at <root> delivers for those packages, sorted by path. When layers deliver the same path, the last
# layer wins. A symlink counts as a source only when it points at a file. A dangling symlink fails here, where it
# is found, and nothing is printed. It reads the release once per key (place_delivered): one find over every layer
# and one sort.
place:delivered() {
  local key line path src rows='' i=0
  local -a layers found paths
  local -A source_of=()
  place:key key "$@"
  if [[ -z "${place_delivered[${key}]+x}" ]]
  then
    list:read layers render:layers "$@" || return 1
    if (( ${#layers[@]} ))
    then
      list:read found find -H "${layers[@]}" \( -type f -o -type l \) || return 1
    fi
    # find reads the layers in order, so each line is under the layer it is at, or one after it.
    for line in "${found[@]}"
    do
      until [[ "${line}" == "${layers[i]}/"* ]]
      do
        if (( ++i == ${#layers[@]} ))
        then
          error "${line}: find printed a path under none of the release's layers"
          return 1
        fi
      done
      source_of[${line#"${layers[i]}/"}]=${line}
    done
    if (( ${#source_of[@]} ))
    then
      list:read paths place:sorted "${!source_of[@]}" || return 1
    fi
    for path in "${paths[@]}"
    do
      src=${source_of[${path}]}
      if [[ -L "${src}" && ! -e "${src}" ]]
      then
        error "${src} is a link to nothing; the release is broken"
        return 1
      fi
      rows+="${path}"$'\t'"${src}"$'\n'
      place_source[${key}${path}]=${src}
    done
    place_delivered[${key}]=${rows}
  fi
  printf '%s' "${place_delivered[${key}]}"
}

# place:sorted <line>… prints the lines, sorted bytewise.
place:sorted() {
  local lines
  printf -v lines '%s\n' "$@"
  LC_ALL=C sort <<<"${lines%$'\n'}"
}

# place:source-of <root> <base> <path> <package>… prints the release file behind one delivered path, or nothing.
place:source-of() {
  local root=$1 base=$2 path=$3 key
  shift 3
  place:delivered "${root}" "${base}" "$@" >/dev/null || return 1
  place:key key "${root}" "${base}" "$@"
  [[ -z "${place_source[${key}${path}]+x}" ]] || printf '%s\n' "${place_source[${key}${path}]}"
}

# place:beyond <root> <base> <before> <after> prints the paths under the base that the package set <after>
# delivers and <before> does not, sorted. Each set is passed as one space-separated word.
place:beyond() {
  local root=$1 base=$2 had has path
  local -a before after
  read -r -a before <<<"$3"
  read -r -a after <<<"$4"
  place:delivered "${root}" "${base}" "${before[@]}" >/dev/null || return 1
  place:delivered "${root}" "${base}" "${after[@]}" >/dev/null || return 1
  place:key had "${root}" "${base}" "${before[@]}"
  place:key has "${root}" "${base}" "${after[@]}"
  while IFS=$'\t' read -r path _
  do
    [[ -z "${path}" || -n "${place_source[${had}${path}]+x}" ]] || printf '%s\n' "${path}"
  done <<<"${place_delivered[${has}]}"
}

# place:merged <path> succeeds for a destination that is merged by key instead of copied: a .json settings
# file. Every .json file the kit delivers is a settings file, except a schema (*.schema.json).
place:merged() { [[ "$1" == *.json && "$1" != *.schema.json ]]; }

# place:json-write <path> <filter> [<jq option>…] rewrites the JSON at <path> through <filter>, which can call
# modules/jq/merge.jq as merge::. It creates the file, as {}, when it is missing. A file jq cannot read, such as one
# with comments, is named, left as it was, and fails.
place:json-write() {
  local path filter=$2 dir=.
  path=$(release:target "$1")
  shift 2
  [[ "${path}" != */* ]] || dir=${path%/*}
  [[ -d "${dir:-/}" ]] || mkdir -p "${dir}"
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
  local src=$1 path=$2 base=${3:-/dev/null} kind
  place:kind "${src}" "${path}" kind
  case "${kind}" in
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
  local src=$1 path=$2 kind
  place:kind "${src}" "${path}" kind
  case "${kind}" in
    (region) region:read "${path}" ;;
    (*) [[ -f "${path}" ]] && cat "${path}" ;;
  esac
}

# place:holds <source> <path> succeeds when the repository has the release's content at <path>: the same file,
# every merged entry, or the same region. A file that is not there holds only an empty source.
place:holds() {
  local src=$1 path=$2 kind
  place:kind "${src}" "${path}" kind
  case "${kind}" in
    (merge) [[ -f "${path}" ]] && place:json-holds "${path}" "${src}" ;;
    (region) cmp --silent "${src}" <(region:read "${path}") ;;
    (*)
      if [[ -f "${path}" ]]
      then cmp --silent "${src}" "${path}"
      else [[ -e "${src}" && ! -s "${src}" ]]
      fi
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
  place:kind "${src}" "${path}" kind
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
# different file exists fails. It returns the number of failures, or 1 when the delivered paths cannot be read.
place:all() {
  local root=$1 base=$2 key root_of row path src refused=0
  local -a rows
  shift 2
  place:delivered "${root}" "${base}" "$@" >/dev/null || return 1
  place:key key "${root}" "${base}" "$@"
  # release:dest joins the base's root and the path, so the root is read once, as the empty path's destination.
  root_of=$(release:dest "${base}" '') || return 1
  readarray -t rows <<<"${place_delivered[${key}]}"
  for row in "${rows[@]}"
  do
    IFS=$'\t' read -r path src <<<"${row}"
    [[ -n "${path}" ]] || continue
    place:new "${src}" "${root_of}${path}" || refused=$(( refused + 1 ))
  done
  return "${refused}"
}

# place:remove <source> <path> removes the release's content from <path>: the kit's entries from a .json
# file, the kit's region from a region file, and otherwise the file itself. A region file that holds no region is
# left as it is, and nothing is said of it. It fails when it cannot take the content out, as for a region whose
# markers region:has refuses, and the file is left as it is.
place:remove() {
  local src=$1 path=$2 status=0 kind
  place:kind "${src}" "${path}" kind
  case "${kind}" in
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
  local base=$1 path=$2 src=$3 dest kind
  dest=$(release:dest "${base}" "${path}")
  [[ -e "${dest}" || -L "${dest}" ]] || return 1
  place:kind "${src}" "${dest}" kind
  [[ "${kind}" == copy ]] || return 1
  ! materialize:owned "${base}" "${path}" || return 1
  ! cmp --silent "${src}" "${dest}"
}

# place:refusals <base> <package>… reads the paths under the base that a removal would take out, one per line, and
# judges the markers of each the packages deliver as a region, as region:remove judges them, so a removal refuses
# before it takes anything out. region:has names each file it refuses. It prints how many it refuses, and fails,
# printing nothing, when the paths the release delivers cannot be read.
place:refusals() {
  local base=$1 key root_of line row path src kind dest status refused=0
  local -a lines rows
  local -A wanted=()
  shift
  place:delivered "${kit_root}" "${base}" "$@" >/dev/null || return 1
  place:key key "${kit_root}" "${base}" "$@"
  readarray -t lines
  for line in "${lines[@]}"
  do
    [[ -z "${line}" ]] || wanted[${line}]=1
  done
  # release:dest joins the base's root and the path, so the root is read once, as the empty path's destination.
  root_of=$(release:dest "${base}" '') || return 1
  readarray -t rows <<<"${place_delivered[${key}]}"
  for row in "${rows[@]}"
  do
    IFS=$'\t' read -r path src <<<"${row}"
    [[ -n "${path}" && -n "${wanted[${path}]+x}" ]] || continue
    place:kind "${src}" "${path}" kind
    [[ "${kind}" == region ]] || continue
    dest=${root_of}${path}
    status=0
    region:has "${dest}" "$(release:target "${dest}")" || status=$?
    (( status != 2 )) || refused=$(( refused + 1 ))
  done
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
# these packages, and nothing without the base release. It fails, and touches nothing, when the manifest or a
# release's paths cannot be read.
place:retire() {
  local base_root=$1 base=$2 previous=$3 current=$4 path dest previous_src recorded now was='' root_of
  local -a paths candidates=()
  shift 4
  # Each list is read before any path is touched, so one that cannot be read touches nothing.
  recorded=$(manifest:paths "${base}" | LC_ALL=C sort) || return 1
  place:delivered "${kit_root}" "${base}" "$@" >/dev/null || return 1
  place:key now "${kit_root}" "${base}" "$@"
  [[ "${base}" == repo || -n "${base_root}" ]] || return 0
  if [[ -n "${base_root}" ]]
  then
    place:delivered "${base_root}" "${base}" "$@" >/dev/null || return 1
    place:key was "${base_root}" "${base}" "$@"
  fi
  readarray -t paths <<<"${recorded}"
  for path in "${paths[@]}"
  do
    [[ -n "${path}" && -z "${place_source[${now}${path}]+x}" ]] || continue
    # Under home, a candidate is a path the previous release delivered for these packages.
    [[ "${base}" == repo || -n "${place_source[${was}${path}]+x}" ]] || continue
    candidates+=("${path}")
  done
  # release:dest joins the base's root and the path, so the root is read once, as the empty path's destination.
  root_of=$(release:dest "${base}" '') || return 1
  for path in "${candidates[@]}"
  do
    dest=${root_of}${path}
    [[ -e "${dest}" ]] || continue
    previous_src=""
    [[ -z "${was}" ]] || previous_src=${place_source[${was}${path}]:-}
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
  done
}

# ----------------------------------------------------------------------------------------------------------------------
