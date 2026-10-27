#!/usr/bin/env bash
# kit-update - move the repository, and the home directory, to the pinned release (kit verb)
# ----------------------------------------------------------------------------------------------------------------------
#
# The previous release is the base of every merge. KIT_BASE names its root; the same release as the current
# one is its own base; otherwise release:fetch finds or fetches it. A path left for lack of a base keeps the
# manifest at the previous version, so the next run starts where this one stopped.
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

conflicts=0
skipped=0
for base in $(release:bases)
do
  while IFS=$'\t' read -r path src
  do
    [[ -n "${path}" ]] || continue
    dest=$(release:dest "${base}" "${path}")
    recorded=$(release:recorded "${base}" "${path}")
    if [[ -z "${recorded}" ]]
    then
      # New in this release: written where there is nothing, or the same thing, and refused over anything else.
      release:place-new "${src}" "${dest}" "new in ${current}" || conflicts=$(( conflicts + 1 ))
      continue
    fi
    previous_src=""
    [[ -n "${base_root}" ]] && previous_src=$(release:source-of "${base_root}" "${base}" "${path}" "${modules[@]}")
    if [[ -z "${previous_src}" ]]
    then
      error "${dest} cannot be updated without the base release ${previous}; set KIT_BASE to a copy of it"
      skipped=$(( skipped + 1 ))
      continue
    fi
    kind=$(release:kind "${src}" "${dest}")
    if [[ "${kind}" == merge ]]
    then
      step "${dest}: merged by key"
      release:place "${src}" "${dest}" "${previous_src}"
    elif [[ "${kind}" == region ]]
    then
      # A region is what its command makes it, so it is regenerated, never merged.
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

if (( skipped > 0 ))
then
  error "${skipped} path(s) stay at ${previous}, and so does the manifest; set KIT_BASE, then run: kit update"
  exit 1
fi
release:write-manifests "${modules[@]}"
if (( conflicts > 0 ))
then
  error "${conflicts} file(s) need a hand; see above."
  exit 1
fi
release:finish "${modules[@]}"

# ----------------------------------------------------------------------------------------------------------------------
