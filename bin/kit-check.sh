#!/usr/bin/env bash
# kit-check - fail when a delivered file differs from the pinned release (kit verb)
# ----------------------------------------------------------------------------------------------------------------------
#
# Every path under the repository is compared with the release's file. A path under home is compared where
# ~/.kit/manifest exists, that is, on a desk that installed; a machine with none, such as a runner, is told so.
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
readarray -t modules < <(release:manifest-modules)
version=$(release:manifest-version)
release:render-all "${kit_root}" "${modules[@]}"
findings=0
if [[ "${version}" != "$(release:version)" ]]
then
  error ".kit/manifest records kit ${version}, and the pinned release is $(release:version); run: kit update"
  findings=$(( findings + 1 ))
fi
for base in $(release:bases)
do
  delivered=$(release:delivered "${kit_root}" "${base}" "${modules[@]}")
  [[ -n "${delivered}" ]] || continue
  if [[ "${base}" != repo && ! -f "$(release:manifest "${base}")" ]]
  then
    info "${base}: nothing is installed under $(release:base-root "${base}") on this machine; not checked"
    continue
  fi
  while IFS=$'\t' read -r path src
  do
    dest=$(release:dest "${base}" "${path}")
    kind=$(release:kind "${src}" "${dest}")
    if [[ ! -f "${dest}" ]]
    then
      error "${dest} is missing"
      findings=$(( findings + 1 ))
    elif [[ "${kind}" == merge ]] && ! command -v jq >/dev/null 2>&1
    then
      error "run: mise install (jq is not on PATH, so ${dest} could not be read)"
      findings=$(( findings + 1 ))
    elif [[ "${kind}" == region ]] && ! release:installed "${src}" "${dest}" >/dev/null
    then
      error "${dest} has no kit region; run: kit update"
      findings=$(( findings + 1 ))
    elif ! release:holds "${src}" "${dest}"
    then
      case "${kind}" in
        (merge) error "${dest} is missing an entry the kit writes; run: kit update" ;;
        (region) error "${dest}: the kit's region differs from the pinned release" ;;
        (*) error "${dest} differs from the pinned release" ;;
      esac
      findings=$(( findings + 1 ))
    fi
  done <<<"${delivered}"
done
if (( findings > 0 ))
then
  error "${findings} finding(s). A delivered file changes by a new release and kit update, never in place."
  exit 1
fi
ok "kit ${version}, modules ${modules[*]:-none}: every delivered file matches the pinned release."

# ----------------------------------------------------------------------------------------------------------------------
