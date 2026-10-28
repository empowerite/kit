#!/usr/bin/env bash
# kit-update - move the repository, and the home directory, to the pinned release (kit verb)
# ----------------------------------------------------------------------------------------------------------------------
#
# kit update takes no arguments. It moves the repository and the home directory from the release .kit/manifest
# records (the previous release) to the pinned release. For each delivered file:
#
#   - a file new in this release is written when nothing, or the same content, is there, and refused otherwise
#   - a merged JSON file is merged by key
#   - a region is regenerated
#   - a missing file is written
#   - a file unchanged since the previous release is replaced
#   - a file edited since the previous release gets a three-way merge, which leaves conflict markers on a conflict
#
# A merge by key, or a three-way merge, needs the previous release as its base. The base is the first of:
#
#   1. the directory KIT_BASE names
#   2. this release, when the previous release is the same version
#   3. the previous release, which release:fetch finds or fetches (never for an unreleased previous version)
#
# kit update checks that every merge has its base before it writes anything. When a base is missing, it writes
# nothing and the manifest stays at the previous version, so the next run starts from the same place.
#
# Exits 0 when the update is complete, and 1 when a base is missing or a file needs to be resolved by hand.
#
# see: bin/release.sh

set -o errexit -o nounset -o pipefail
# shellcheck source=bin/1st.sh
source "${kit_root:?run through bin/kit}/bin/1st.sh"
use release

if [[ "${1:-}" == --help ]]
then
  echo "usage: kit update" >&2
  exit 0
fi

guard:require tool:jq
release:require-manifest
readarray -t modules < <(release:manifest-modules)
previous=$(release:manifest-version)
current=$(release:version)
release:render-all "${kit_root}" "${modules[@]}"
section "kit update ${previous} → ${current}, modules: ${modules[*]}"

base_root=""
if [[ -n "${KIT_BASE:-}" ]]
then
  base_root=${KIT_BASE}
elif [[ "${previous}" == "${current}" ]]
then
  base_root=${kit_root}
elif [[ "${previous}" != "$(release:unreleased)" ]]
then
  base_root=$(release:fetch "${previous}" || true)
fi
if [[ -n "${base_root}" ]] && ! release:render-all "${base_root}" "${modules[@]}"
then
  error "the base release at ${base_root} could not be read; see above"
  exit 1
fi

# lacks-base <base> <path> <src> is true when updating the path needs the base release, and the base release is
# missing.
lacks-base() {
  local base=$1 path=$2 src=$3 dest recorded kind
  dest=$(release:dest "${base}" "${path}")
  recorded=$(release:recorded "${base}" "${path}")
  [[ -n "${recorded}" ]] || return 1
  [[ -z "${base_root}" || -z "$(release:source-of "${base_root}" "${base}" "${path}" "${modules[@]}")" ]] || return 1
  kind=$(release:kind "${src}" "${dest}")
  [[ "${kind}" == merge ]] && return 0
  [[ "${kind}" != region ]] && release:installed "${src}" "${dest}" >/dev/null 2>&1 \
    && [[ "${recorded}" != "$(release:installed "${src}" "${dest}" | release:digest)" ]]
}

# Find every path that lacks its base before writing anything, so an update is all or nothing.
skipped=0
for base in $(release:bases)
do
  while IFS=$'\t' read -r path src
  do
    [[ -n "${path}" ]] || continue
    if lacks-base "${base}" "${path}" "${src}"
    then
      error "$(release:dest "${base}" "${path}") cannot be updated without the base release ${previous};" \
        "set KIT_BASE to a copy of it"
      skipped=$(( skipped + 1 ))
    fi
  done < <(release:delivered "${kit_root}" "${base}" "${modules[@]}")
done
if (( skipped > 0 ))
then
  error "${skipped} path(s) stay at ${previous}, and so does the manifest; set KIT_BASE, then run: kit update"
  exit 1
fi

conflicts=0
for base in $(release:bases)
do
  while IFS=$'\t' read -r path src
  do
    [[ -n "${path}" ]] || continue
    dest=$(release:dest "${base}" "${path}")
    recorded=$(release:recorded "${base}" "${path}")
    if [[ -z "${recorded}" ]]
    then
      # New in this release: written when nothing, or the same content, is there, and refused otherwise.
      release:place-new "${src}" "${dest}" "new in ${current}" || conflicts=$(( conflicts + 1 ))
      continue
    fi
    previous_src=""
    [[ -n "${base_root}" ]] && previous_src=$(release:source-of "${base_root}" "${base}" "${path}" "${modules[@]}")
    kind=$(release:kind "${src}" "${dest}")
    if [[ "${kind}" == merge ]]
    then
      step "${dest}: merged by key"
      release:place "${src}" "${dest}" "${previous_src}"
    elif [[ "${kind}" == region ]]
    then
      # A region's content comes from its command, so it is regenerated, never merged.
      step "${dest}: the region regenerated"
      release:place "${src}" "${dest}"
    elif ! release:installed "${src}" "${dest}" >/dev/null 2>&1
    then
      step "${dest}: missing, written"
      release:place "${src}" "${dest}"
    elif [[ "${recorded}" == "$(release:installed "${src}" "${dest}" | release:digest)" ]]
    then
      step "${dest}: unchanged since ${previous}, replaced"
      release:place "${src}" "${dest}"
    else
      step "${dest}: edited since ${previous}, three-way merged"
      if ! git merge-file --marker-size 7 "${dest}" "${previous_src}" "${src}"
      then
        warn "${dest}: conflict markers are left in the file; resolve them, then run: kit check"
        conflicts=$(( conflicts + 1 ))
      fi
    fi
  done < <(release:delivered "${kit_root}" "${base}" "${modules[@]}")
  release:retire "${base_root}" "${base}" "${previous}" "${current}" "${modules[@]}"
done

release:write-manifests "${modules[@]}"
if (( conflicts > 0 ))
then
  error "${conflicts} file(s) need a hand; see above."
  exit 1
fi
release:finish "${modules[@]}"

# ----------------------------------------------------------------------------------------------------------------------
