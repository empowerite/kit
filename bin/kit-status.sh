#!/usr/bin/env bash
# kit-status - show every worktree with its branch, pull request, and state (kit verb)
# ----------------------------------------------------------------------------------------------------------------------
#
# see: bin/status.sh

set -o errexit -o nounset -o pipefail
# shellcheck source=bin/1st.sh
source "${kit_root:?run through bin/kit}/bin/1st.sh"
use status

if [[ "${1:-}" == --help ]]
then
  echo "usage: kit status [--no-fetch] [--json] [--mine | --all | --owner=<login>…]" >&2
  exit 0
fi

status:render "$@"

# ----------------------------------------------------------------------------------------------------------------------
