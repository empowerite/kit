#!/usr/bin/env bash
# kit-install - mirror the release into the repository, take the named targets, and write .kit/manifest (kit verb, run through bin/kit)
# ----------------------------------------------------------------------------------------------------------------------
#
# see: bash/kit.sh

set -o errexit -o nounset -o pipefail
# shellcheck source=bash/1st.sh
source "${kit_root:?run through bin/kit}/bash/1st.sh"
use kit

if [[ "${1:-}" == --help ]]
then
  echo "usage: kit install [<target>…]" >&2
  exit 0
fi

guard:require tool:jq tool:git
kit:require-targets "$@"
if [[ -f .kit/manifest ]]
then
  error "run: kit update (this repository already carries kit $(kit:manifest-version))"
  exit 1
fi
section "kit install $(cat "${kit_root}/version")${*:+, targets: $*}"
while IFS= read -r path
do
  step "${path}"
  kit:place "$(kit:source-of "${kit_root}" "${path}" "$@")" "${path}"
done < <(kit:delivered "${kit_root}" "$@")
kit:write-manifest "$@"
kit:finish "$@"

# ----------------------------------------------------------------------------------------------------------------------
