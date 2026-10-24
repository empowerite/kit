#!/usr/bin/env bash
# kit-update - after the pin is bumped: replace unchanged files, three-way merge changed ones (kit verb, run through bin/kit)
# ----------------------------------------------------------------------------------------------------------------------
#
# An untouched file is one whose digest still matches the manifest, which records the previous release's own
# content; it is replaced. An edited one is merged with the previous release as the base: KIT_BASE=<dir> names
# that release's root, otherwise mise knows where it installed it.
#
# see: bash/kit.sh

set -o errexit -o nounset -o pipefail
# shellcheck source=bash/1st.sh
source "${kit_root:?run through bin/kit}/bash/1st.sh"
use kit

if [[ "${1:-}" == --help ]]
then
  echo "usage: kit update" >&2
  exit 0
fi

guard:require tool:jq tool:git
kit:require-manifest
read -r -a targets <<<"$(kit:manifest-targets)"
previous=$(kit:manifest-version)
current=$(cat "${kit_root}/version")
section "kit update ${previous} → ${current}${targets[*]:+, targets: ${targets[*]}}"

base_root=""
if [[ -n "${KIT_BASE:-}" ]]
then
  base_root=${KIT_BASE}
elif [[ "${previous}" != "${current}" ]] && command -v mise >/dev/null 2>&1
then
  base_root=$(mise where "github:empowerite/kit@${previous}" 2>/dev/null || true)
  if [[ -z "${base_root}" ]] && mise install --quiet "github:empowerite/kit@${previous}" 2>/dev/null
  then
    base_root=$(mise where "github:empowerite/kit@${previous}" 2>/dev/null || true)
  fi
fi

conflicts=0
while IFS= read -r path
do
  src=$(kit:source-of "${kit_root}" "${path}" "${targets[@]}")
  base=""
  [[ -n "${base_root}" ]] && base=$(kit:source-of "${base_root}" "${path}" "${targets[@]}" || true)
  if kit:merged "${path}"
  then
    step "${path}: merged by key"
    kit:place "${src}" "${path}" "${base}"
    continue
  fi
  if [[ ! -f "${path}" ]]
  then
    step "${path}: missing, written"
    kit:place "${src}" "${path}"
    continue
  fi
  recorded=$(awk -v p="${path}" '$2==p{print $1}' .kit/manifest)
  if [[ -z "${recorded}" || "${recorded}" == "$(kit:digest < "${path}")" ]]
  then
    step "${path}: unchanged since ${previous}, replaced"
    kit:place "${src}" "${path}"
    continue
  fi
  if [[ -z "${base}" ]]
  then
    error "${path} was edited and the base release ${previous} is not installed; run: mise install github:empowerite/kit@${previous}"
    conflicts=$(( conflicts + 1 ))
    continue
  fi
  step "${path}: edited since ${previous}, three-way merged"
  if ! git merge-file --marker-size 7 "${path}" "${base}" "${src}"
  then
    warn "${path}: conflict markers are left in the file; resolve them, then run: kit check"
    conflicts=$(( conflicts + 1 ))
  fi
done < <(kit:delivered "${kit_root}" "${targets[@]}")

kit:write-manifest "${targets[@]}"
if (( conflicts > 0 ))
then
  error "${conflicts} file(s) need a hand; see above."
  exit 1
fi
kit:finish "${targets[@]}"

# ----------------------------------------------------------------------------------------------------------------------
