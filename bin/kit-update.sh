#!/usr/bin/env bash
# kit-update - move the repository, and the home directory, to the pinned release (kit verb)
# ----------------------------------------------------------------------------------------------------------------------
#
# A file whose digest still matches its manifest, which records the previous release's own content, is replaced.
# An edited one is three-way merged with the previous release as the base, which release:fetch finds or fetches.
# A path the previous release did not deliver is written only where there is nothing, or the same thing, there.
# A template is rendered from the repository's answers in both releases before either is compared. The home base
# is updated against ~/.kit/manifest the same way.
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
read -r -a modules <<<"$(release:manifest-modules)"
previous=$(release:manifest-version)
current=$(release:version)
release:render-all "${kit_root}" "${modules[@]}"
section "kit update ${previous} → ${current}, modules: ${modules[*]}"

# The previous release is the base of every merge, needed only when the release changed. KIT_BASE names its
# root; otherwise release:fetch finds or fetches it.
base_root=""
# shellcheck disable=SC2154  # release_unreleased is declared by bin/release.sh, which use loaded above
if [[ -n "${KIT_BASE:-}" ]]
then
  base_root=${KIT_BASE}
elif [[ "${previous}" != "${current}" && "${previous}" != "${release_unreleased}" ]]
then
  base_root=$(release:fetch "${previous}" || true)
fi
[[ -n "${base_root}" ]] && release:render-all "${base_root}" "${modules[@]}"
needs_base=false
[[ "${previous}" != "${current}" ]] && needs_base=true

conflicts=0
# shellcheck disable=SC2154  # release_bases is declared by bin/release.sh, which use loaded above
for base in "${release_bases[@]}"
do
  while IFS=$'\t' read -r path src
  do
    [[ -n "${path}" ]] || continue
    dest=$(release:dest "${base}" "${path}")
    recorded=$(release:recorded "${base}" "${path}")
    if [[ -z "${recorded}" ]]
    then
      # A path the previous release did not deliver: written where there is nothing, or the same thing.
      if release:merged "${dest}" || [[ ! -f "${dest}" ]]
      then
        step "${dest}: new in ${current}, written"
        release:place "${src}" "${dest}"
      elif cmp --silent "${src}" "${dest}"
      then
        step "${dest}: new in ${current}, and already as the release has it"
      else
        error "${dest} is new in ${current} and a file is there already; move yours aside, then run: kit update"
        conflicts=$(( conflicts + 1 ))
      fi
      continue
    fi
    previous_src=""
    [[ -n "${base_root}" ]] && previous_src=$(release:source-of "${base_root}" "${base}" "${path}" "${modules[@]}")
    if [[ "${needs_base}" == true && -z "${previous_src}" ]]
    then
      # Without the base, neither a merge nor the removal of the previous release's entries is possible.
      error "${dest} cannot be updated without the base release ${previous}; set KIT_BASE to a copy of it"
      conflicts=$(( conflicts + 1 ))
      continue
    fi
    if release:merged "${dest}"
    then
      step "${dest}: merged by key"
      release:place "${src}" "${dest}" "${previous_src}"
    elif [[ ! -f "${dest}" ]]
    then
      step "${dest}: missing, written"
      release:place "${src}" "${dest}"
    elif [[ "${recorded}" == "$(release:digest < "${dest}")" ]]
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
done

release:write-manifests "${modules[@]}"
if (( conflicts > 0 ))
then
  error "${conflicts} file(s) need a hand; see above."
  exit 1
fi
release:finish "${modules[@]}"

# ----------------------------------------------------------------------------------------------------------------------
