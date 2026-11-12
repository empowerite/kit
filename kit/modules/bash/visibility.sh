# shellcheck shell=bash
# visibility.sh - pull behind push: whether each pull source serves a pushed release yet, and with its hash
# ----------------------------------------------------------------------------------------------------------------------
# shellcheck disable=SC2154  # kit_jq is exported by modules/bash/1st.sh
#
# Visibility is a state kit watches, not a step a push waits for. The release tag, release/<version>, is what was
# pushed, and kit publish tags what it pushed as release-hashes/<version>: a tag of a blob of one line per artifact,
# "<artifact>\t<hash>", the hash its pull kind takes. Anyone who can read the forge reads both. For each artifact
# under publish: in kit.yml that names what it is, as a tool pin does, with release:, npm:, module:, or url:, and each
# pull source that serves that form, or only the sources its verify: names, a release is:
#
#   visible     served, with the hash the release-hashes tag records, or with any where none is recorded
#   pending     tagged, not served yet; stuck once older than the source's expect:
#   blocked     the source says it never will serve it, as a scan's block or a refused approval
#   mismatch    served, with another hash
#
# A source that cannot be reached is unknown, and asked again on the next read. Each answer is kept under
# ${XDG_CACHE_HOME:-$HOME/.cache}/kit/visibility/: visible, blocked, and mismatch for good, pending for five minutes,
# so a view that reads it asks the network once in a while. A module's zip is the proxy's own, so it has no hash
# recorded, and is visible once served.
#
# see: modules/bash/targets.sh, modules/bash/sources.sh, bin/kit-publish, bin/kit-release

use sources targets file forge

# visibility_pending_for is how long a pending answer is kept, in seconds, before the source is asked again.
declare -g visibility_pending_for=${KIT_VISIBILITY_PENDING_FOR:-300}

# visibility:seconds <duration> prints the duration in seconds: a whole number and s, m, h, or d, as 30m or 1d, and a
# bare number as seconds. It returns 1 for anything else.
visibility:seconds() {
  [[ "$1" =~ ^([0-9]+)([smhd]?)$ ]] || return 1
  case "${BASH_REMATCH[2]}" in
    ('' | s) printf '%s\n' "${BASH_REMATCH[1]}" ;;
    (m) printf '%s\n' $(( BASH_REMATCH[1] * 60 )) ;;
    (h) printf '%s\n' $(( BASH_REMATCH[1] * 3600 )) ;;
    (d) printf '%s\n' $(( BASH_REMATCH[1] * 86400 ))
  esac
}

# visibility:coordinate <artifact> <version> prints what the artifact is at the version, as kit_coordinate gives a
# tool's, and nothing for an artifact that names none.
visibility:coordinate() {
  jq -L "${kit_jq}" --raw-output --arg v "$2" 'import "toolchain" as toolchain;
    if [.url, .release, .module, .npm] | map(select(. != "")) | length == 0 then empty
    else toolchain::kit_coordinate(.; "linux-x64"; $v) end' <<<"$1"
}

# visibility:hash-kind <coordinate> prints the pull kind whose hash a pushed file of the coordinate's form is recorded
# with, and nothing for a module, whose zip the proxy makes.
visibility:hash-kind() {
  case "$1" in
    (release:* | url:*) printf 'github-releases\n' ;;
    (npm:*) printf 'npm\n' ;;
    (*) ;;
  esac
}

# visibility:file <artifact> <version> <file>… prints the file of those pushed that the pull path serves: the one
# named as the artifact's asset, or its url's last part, and the first where it names neither.
visibility:file() {
  local artifact=$1 version=$2 name file
  shift 2
  name=$(jq --raw-output --arg v "${version}" '(if .asset != "" then .asset else (.url | sub(".*/"; "")) end)
    | gsub("\\{version\\}"; $v)' <<<"${artifact}")
  for file in "$@"
  do
    [[ "${file##*/}" != "${name}" ]] || { printf '%s\n' "${file}"; return 0; }
  done
  printf '%s\n' "$1"
}

# visibility:recorded <version> prints the lines release-hashes/<version> records, "<artifact>\t<hash>", read from the
# primary remote, and nothing where it has none yet.
visibility:recorded() {
  local primary tag=release-hashes/$1
  primary=$(forge:primary 2>/dev/null) || return 0
  if ! git rev-parse --quiet --verify "refs/tags/${tag}" >/dev/null
  then
    # A tag the primary does not have yet is no hash recorded yet.
    git fetch --quiet --no-tags --end-of-options "${primary}" "refs/tags/${tag}:refs/tags/${tag}" 2>/dev/null \
      || return 0
  fi
  git cat-file -p "refs/tags/${tag}" 2>/dev/null || true  # a tag of anything but a blob records nothing
}

# visibility:kept <key> prints "<state>\t<detail>" kept for the key while it holds: visible, blocked, and mismatch for
# good, pending for visibility_pending_for seconds. It returns 1 where nothing is kept that holds.
visibility:kept() {
  local file state detail at
  file="${XDG_CACHE_HOME:-${HOME}/.cache}/kit/visibility/$1"
  [[ -f "${file}" ]] || return 1
  IFS=$'\t' read -r state detail at < "${file}" || return 1
  case "${state}" in
    (visible | blocked | mismatch) ;;
    (pending | stuck) (( $(date +%s) - at < visibility_pending_for )) || return 1 ;;
    (*) return 1
  esac
  printf '%s\t%s\n' "${state}" "${detail}"
}

# visibility:keep <key> <state> <detail> keeps the answer for the key, unless it is unknown.
visibility:keep() {
  local dir="${XDG_CACHE_HOME:-${HOME}/.cache}/kit/visibility"
  [[ "$2" != unknown ]] || return 0
  mkdir -p "${dir}" 2>/dev/null || return 0
  # A cache that cannot be written is asked again next time.
  printf '%s\t%s\t%s\n' "$2" "$3" "$(date +%s)" > "${dir}/$1" 2>/dev/null || true
}

# visibility:ask <coordinate> <recorded> <age> <name> <kind> <url> <entry> asks one pull source for the coordinate,
# and prints "<state>\t<detail>": the detail is the hash served for visible and mismatch, and the age and expect for
# pending.
visibility:ask() {
  local coordinate=$1 recorded=$2 age=$3 name=$4 kind=$5 url=$6 entry=$7 status=0 have expect seconds file
  # shellcheck disable=SC2034  # read by sources:get and by the provider, as npm reads its scope
  local sources_name=${name} sources_entry=${entry}
  file:temp file visibility || return 1
  if [[ -z "${kind}" ]]
  then sources:get "${url}" "${file}" || status=$?
  else sources:call "${kind}" fetch "${url}" "${coordinate}" "${file}" || status=$?
  fi
  case "${status}" in
    (0)
      if [[ -n "${kind}" ]]
      then have=$(sources:call "${kind}" hash "${file}" "${coordinate}") || have=''
      else have="sha256:$(file:digest "${file}")"
      fi
      rm -f "${file}"
      if [[ -z "${recorded}" || "${have}" == "${recorded}" ]]
      then printf 'visible\t%s\n' "${have}"
      else printf 'mismatch\t%s\n' "${have}"
      fi
      ;;
    (4)
      rm -f "${file}"
      expect=$(jq --raw-output '.expect // ""' <<<"${entry:-{\}}")
      seconds=$(visibility:seconds "${expect}" 2>/dev/null) || seconds=''
      if [[ -n "${seconds}" && "${age}" -gt "${seconds}" ]]
      then printf 'stuck\t%s past %s\n' "$(visibility:age "${age}")" "${expect}"
      else printf 'pending\t%s\n' "$(visibility:age "${age}")"
      fi
      ;;
    (5) rm -f "${file}"; printf 'blocked\t%s says it will not serve it\n' "${name}" ;;
    (*) rm -f "${file}"; printf 'unknown\t%s could not be reached\n' "${name}" ;;
  esac
}

# visibility:age <seconds> prints an age as a person reads it: 45s, 12m, 3h, or 2d.
visibility:age() {
  if (( $1 < 60 )); then printf '%ss\n' "$1"
  elif (( $1 < 3600 )); then printf '%sm\n' $(( $1 / 60 ))
  elif (( $1 < 86400 )); then printf '%sh\n' $(( $1 / 3600 ))
  else printf '%sd\n' $(( $1 / 86400 ))
  fi
}

# visibility:release <version> <age> [<path>] prints, as JSON lines, the state of the release on each pull source that
# checks it, for each artifact under publish: that names what it is: {artifact, source, state, detail}. <age> is how
# long ago the release was tagged, in seconds.
visibility:release() {
  local version=$1 age=$2 path=${3:-} artifact name coordinate recorded hash verify line source kind url entry key
  local answer state detail
  local -a artifacts
  list:read artifacts targets:artifacts || return 1
  (( ${#artifacts[@]} )) || return 0
  [[ -n "${path}" ]] || path=$(sources:path) || return 1
  recorded=$(visibility:recorded "${version}")
  for artifact in "${artifacts[@]}"
  do
    [[ -n "${artifact}" ]] || continue
    name=$(jq --raw-output .name <<<"${artifact}")
    coordinate=$(visibility:coordinate "${artifact}" "${version}") || return 1
    [[ -n "${coordinate}" ]] || continue
    hash=$(awk -F '\t' -v n="${name}" '$1 == n { print $2; exit }' <<<"${recorded}")
    verify=$(jq --raw-output '.verify | join(" ")' <<<"${artifact}")
    while IFS=$'\x1f' read -r source kind url _ entry
    do
      [[ -n "${source}" && "${source}" != off ]] || continue
      [[ -z "${verify}" || " ${verify} " == *" ${source} "* ]] || continue
      if [[ "${source}" == direct ]]
      then
        line=$(sources:direct "${coordinate}") || continue
        IFS=$'\x1f' read -r kind url <<<"${line//$'\t'/$'\x1f'}"
        entry='{}'
      else
        sources:call "${kind}" serves "${coordinate}" || continue
      fi
      key=$(printf '%s@%s@%s' "${version}" "${name}" "${source}" | tr '/:' '__')
      if ! answer=$(visibility:kept "${key}")
      then
        answer=$(visibility:ask "${coordinate}" "${hash}" "${age}" "${source}" "${kind}" "${url}" "${entry}") \
          || return 1
        visibility:keep "${key}" "${answer%%$'\t'*}" "${answer#*$'\t'}"
      fi
      state=${answer%%$'\t'*} detail=${answer#*$'\t'}
      jq --null-input --compact-output --arg artifact "${name}" --arg source "${source}" --arg state "${state}" \
        --arg detail "${detail}" '{artifact: $artifact, source: $source, state: $state, detail: $detail}'
    done <<<"${path//$'\t'/$'\x1f'}"
  done
}

# visibility:summary <states> prints one line for a release's states, JSON lines as visibility:release prints them:
# "visible" where every source serves it, and otherwise each source that does not, with its state.
visibility:summary() {
  jq --slurp --raw-output 'if length == 0 then empty
    elif all(.state == "visible") then "visible on \(map(.source) | unique | join(", "))"
    else map(select(.state != "visible") | "\(.artifact) \(.state) on \(.source) (\(.detail))") | join("; ") end' \
    <<<"$1"
}

# ----------------------------------------------------------------------------------------------------------------------
