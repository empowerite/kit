#!/usr/bin/env bash
# kit-update - after the pin is bumped: replace unchanged files, three-way merge changed ones (kit verb, run through bin/kit)
# ----------------------------------------------------------------------------------------------------------------------
#
# A file whose digest still matches the manifest, which records the previous release's own content, is replaced.
# An edited one is three-way merged with the previous release as the base. A path the previous release did not
# deliver is written only where the repository has nothing, or the same thing, there.
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

guard:require tool:jq tool:git
release:require-manifest
read -r -a targets <<<"$(release:manifest-targets)"
previous=$(release:manifest-version)
current=$(release:version)
section "kit update ${previous} → ${current}${targets[*]:+, targets: ${targets[*]}}"

# The previous release is the base of every merge, and a merge is needed only when the release changed. KIT_BASE
# names the base's root; otherwise mise knows where it installed it.
base_root=""
# shellcheck disable=SC2154  # release_unreleased is declared by bin/release.sh, which use loaded above
if [[ -n "${KIT_BASE:-}" ]]
then
  base_root=${KIT_BASE}
elif [[ "${previous}" != "${current}" && "${previous}" != "${release_unreleased}" ]] && command -v mise >/dev/null 2>&1
then
  base_root=$(mise where "github:empowerite/kit@${previous}" 2>/dev/null || true)
  if [[ -z "${base_root}" ]] && mise install --quiet "github:empowerite/kit@${previous}" 2>/dev/null
  then
    base_root=$(mise where "github:empowerite/kit@${previous}" 2>/dev/null || true)
  fi
fi
needs_base=false
[[ "${previous}" != "${current}" ]] && needs_base=true

conflicts=0
while IFS=$'\t' read -r path src
do
  recorded=$(awk -v p="${path}" '$2 == p { print $1 }' .kit/manifest)
  if [[ -z "${recorded}" ]]
  then
    # A path the previous release did not deliver: written where the repository has nothing, or the same thing.
    if release:merged "${path}" || [[ ! -f "${path}" ]]
    then
      step "${path}: new in ${current}, written"
      release:place "${src}" "${path}"
    elif cmp --silent "${src}" "${path}"
    then
      step "${path}: new in ${current}, and already as the release has it"
    else
      error "${path} is new in ${current} and the repository has a file there; move yours aside, then run: kit update"
      conflicts=$(( conflicts + 1 ))
    fi
    continue
  fi
  base=""
  [[ -n "${base_root}" ]] && base=$(release:source-of "${base_root}" "${path}" "${targets[@]}")
  if [[ "${needs_base}" == true && -z "${base}" ]]
  then
    # Without the base, neither a merge nor the removal of the previous release's entries is possible.
    error "${path} cannot be updated without the base release ${previous}; run: mise install github:empowerite/kit@${previous}"
    conflicts=$(( conflicts + 1 ))
    continue
  fi
  if release:merged "${path}"
  then
    step "${path}: merged by key"
    release:place "${src}" "${path}" "${base}"
  elif [[ ! -f "${path}" ]]
  then
    step "${path}: missing, written"
    release:place "${src}" "${path}"
  elif [[ "${recorded}" == "$(release:digest < "${path}")" ]]
  then
    step "${path}: unchanged since ${previous}, replaced"
    release:place "${src}" "${path}"
  else
    step "${path}: edited since ${previous}, three-way merged"
    if ! git merge-file --marker-size 7 "${path}" "${base}" "${src}"
    then
      warn "${path}: conflict markers are left in the file; resolve them, then run: kit check"
      conflicts=$(( conflicts + 1 ))
    fi
  fi
done < <(release:delivered "${kit_root}" "${targets[@]}")

release:write-manifest "${targets[@]}"
if (( conflicts > 0 ))
then
  error "${conflicts} file(s) need a hand; see above."
  exit 1
fi
release:finish "${targets[@]}"

# ----------------------------------------------------------------------------------------------------------------------
