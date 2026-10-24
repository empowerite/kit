#!/usr/bin/env bash
# kit-check - fail when a delivered file differs from the pinned release (kit verb, run through bin/kit)
# ----------------------------------------------------------------------------------------------------------------------
#
# see: bash/kit.sh

set -o errexit -o nounset -o pipefail
# shellcheck source=bash/1st.sh
source "${kit_root:?run through bin/kit}/bash/1st.sh"
use kit

if [[ "${1:-}" == --help ]]
then
  echo "usage: kit check" >&2
  exit 0
fi

kit:require-manifest
read -r -a targets <<<"$(kit:manifest-targets)"
version=$(kit:manifest-version)
findings=0
if [[ "${version}" != "$(cat "${kit_root}/version")" ]]
then
  error ".kit/manifest records kit ${version}, and the pinned release is $(cat "${kit_root}/version"); run: kit update"
  findings=$(( findings + 1 ))
fi
while IFS= read -r path
do
  src=$(kit:source-of "${kit_root}" "${path}" "${targets[@]}")
  if [[ ! -f "${path}" ]]
  then
    error "${path} is missing"
    findings=$(( findings + 1 ))
  elif kit:merged "${path}"
  then
    if ! command -v jq >/dev/null 2>&1
    then
      error "run: mise install (jq is not on PATH, so ${path} could not be read)"
      findings=$(( findings + 1 ))
    elif ! kit:json-holds "${path}" "${src}"
    then
      error "${path} is missing an entry the kit writes; run: kit update"
      findings=$(( findings + 1 ))
    fi
  elif ! cmp --silent "${src}" "${path}"
  then
    error "${path} differs from the pinned release"
    findings=$(( findings + 1 ))
  fi
done < <(kit:delivered "${kit_root}" "${targets[@]}")
if (( findings > 0 ))
then
  error "${findings} finding(s). A delivered file is changed by pinning a new release and running kit update, not in place."
  exit 1
fi
ok "kit ${version}, targets ${targets[*]:-none}: every delivered file matches the pinned release."

# ----------------------------------------------------------------------------------------------------------------------
