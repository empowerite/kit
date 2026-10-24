#!/usr/bin/env bash
# kit-install - mirror the release into the repository, take the named targets, and write .kit/manifest (kit verb, run through bin/kit)
# ----------------------------------------------------------------------------------------------------------------------
#
# see: bin/release.sh

set -o errexit -o nounset -o pipefail
# shellcheck source=bin/1st.sh
source "${kit_root:?run through bin/kit}/bin/1st.sh"
use release

if [[ "${1:-}" == --help ]]
then
  echo "usage: kit install [<target>…]" >&2
  exit 0
fi

guard:require tool:jq tool:git
release:require-targets "$@"
if [[ -f .kit/manifest ]]
then
  error "run: kit update (this repository already carries kit $(release:manifest-version))"
  exit 1
fi
section "kit install $(release:version)${*:+, targets: $*}"
while IFS= read -r path
do
  step "${path}"
  release:place "$(release:source-of "${kit_root}" "${path}" "$@")" "${path}"
done < <(release:delivered "${kit_root}" "$@")
release:write-manifest "$@"
release:finish "$@"

# ----------------------------------------------------------------------------------------------------------------------
