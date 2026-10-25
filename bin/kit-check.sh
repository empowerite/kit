#!/usr/bin/env bash
# kit-check - fail when a delivered file differs from the pinned release (kit verb)
# ----------------------------------------------------------------------------------------------------------------------
#
# Every path under the repository is checked. A path under home is checked where ~/.kit/manifest exists, that is,
# on a desk that installed; a machine with none, such as a runner, is told so and the home base is skipped.
#
# see: bin/release.sh

set -o errexit -o nounset -o pipefail
# shellcheck source=bin/1st.sh
source "${kit_root:?run through bin/kit}/bin/1st.sh"
use release

if [[ "${1:-}" == --help ]]
then
  echo "usage: kit check" >&2
  exit 0
fi

release:require-manifest
read -r -a modules <<<"$(release:manifest-modules)"
version=$(release:manifest-version)
release:render-all "${kit_root}" "${modules[@]}"
findings=0
if [[ "${version}" != "$(release:version)" ]]
then
  error ".kit/manifest records kit ${version}, and the pinned release is $(release:version); run: kit update"
  findings=$(( findings + 1 ))
fi
# shellcheck disable=SC2154  # release_bases is declared by bin/release.sh, which use loaded above
for base in "${release_bases[@]}"
do
  delivered=$(release:delivered "${kit_root}" "${base}" "${modules[@]}")
  [[ -n "${delivered}" ]] || continue
  if [[ "${base}" != repo && ! -f "$(release:manifest "${base}")" ]]
  then
    info "${base}: nothing is installed under $(release:base-root "${base}") on this machine, so its paths were not checked"
    continue
  fi
  while IFS=$'\t' read -r path src
  do
    dest=$(release:dest "${base}" "${path}")
    if [[ ! -f "${dest}" ]]
    then
      error "${dest} is missing"
      findings=$(( findings + 1 ))
    elif release:merged "${dest}"
    then
      if ! command -v jq >/dev/null 2>&1
      then
        error "run: mise install (jq is not on PATH, so ${dest} could not be read)"
        findings=$(( findings + 1 ))
      elif ! release:json-holds "${dest}" "${src}"
      then
        error "${dest} is missing an entry the kit writes; run: kit update"
        findings=$(( findings + 1 ))
      fi
    elif ! cmp --silent "${src}" "${dest}"
    then
      error "${dest} differs from the pinned release"
      findings=$(( findings + 1 ))
    fi
  done <<<"${delivered}"
done
if (( findings > 0 ))
then
  error "${findings} finding(s). A delivered file is changed by pinning a new release and running kit update, not in place."
  exit 1
fi
ok "kit ${version}, modules ${modules[*]:-none}: every delivered file matches the pinned release."

# ----------------------------------------------------------------------------------------------------------------------
