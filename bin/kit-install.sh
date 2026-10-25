#!/usr/bin/env bash
# kit-install - mirror the release into the repository, and into the home directory (kit verb)
# ----------------------------------------------------------------------------------------------------------------------
#
# Activates the default modules and the named ones, with the modules they depend on, places what they deliver
# under each base, and writes the manifests: .kit/manifest, and ~/.kit/manifest when a module delivers to home.
#
# see: bin/release.sh

set -o errexit -o nounset -o pipefail
# shellcheck source=bin/1st.sh
source "${kit_root:?run through bin/kit}/bin/1st.sh"
use release

if [[ "${1:-}" == --help ]]
then
  echo "usage: kit install [<module>…]" >&2
  exit 0
fi

guard:require tool:jq
release:require-modules "${kit_root}" "$@"
if [[ -f .kit/manifest ]]
then
  error "run: kit update (this repository already carries kit $(release:manifest-version))"
  exit 1
fi
read -r -a defaults <<<"$(release:default-modules "${kit_root}" | paste -s -d ' ' -)"
read -r -a modules <<<"$(release:resolve "${kit_root}" "${defaults[@]}" "$@" | paste -s -d ' ' -)"
release:render-all "${kit_root}" "${modules[@]}"
section "kit install $(release:version), modules: ${modules[*]}"
# shellcheck disable=SC2154  # release_bases is declared by bin/release.sh, which use loaded above
for base in "${release_bases[@]}"
do
  while IFS=$'\t' read -r path src
  do
    [[ -n "${path}" ]] || continue
    dest=$(release:dest "${base}" "${path}")
    step "${dest}"
    release:place "${src}" "${dest}"
  done < <(release:delivered "${kit_root}" "${base}" "${modules[@]}")
done
release:write-manifests "${modules[@]}"
release:finish "${modules[@]}"

# ----------------------------------------------------------------------------------------------------------------------
