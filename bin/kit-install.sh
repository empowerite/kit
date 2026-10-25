#!/usr/bin/env bash
# kit-install - mirror the release into the repository, and into the home directory (kit verb)
# ----------------------------------------------------------------------------------------------------------------------
#
# Activates the default modules and the named ones, with the modules they depend on, places what they deliver
# under each base, and writes the manifests. A different file already at a delivered path is refused, named, and
# left alone, and no manifest is written until every path is clear.
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
readarray -t defaults < <(release:default-modules "${kit_root}")
readarray -t modules < <(release:resolve "${kit_root}" "${defaults[@]}" "$@")
release:render-all "${kit_root}" "${modules[@]}"
section "kit install $(release:version), modules: ${modules[*]}"
refused=0
for base in $(release:bases)
do
  release:place-all "${kit_root}" "${base}" "${modules[@]}" || refused=$(( refused + $? ))
done
if (( refused > 0 ))
then
  error "${refused} path(s) already hold a different file; move each aside, then run: kit install"
  exit 1
fi
release:write-manifests "${modules[@]}"
release:finish "${modules[@]}"

# ----------------------------------------------------------------------------------------------------------------------
