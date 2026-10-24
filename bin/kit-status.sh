#!/usr/bin/env bash
# kit-status - show every worktree with its branch, pull request, and state (kit verb, run through bin/kit)
# ----------------------------------------------------------------------------------------------------------------------
#
# see: bin/git.sh

set -o errexit -o nounset -o pipefail
# shellcheck source=bin/1st.sh
source "${kit_root:?run through bin/kit}/bin/1st.sh"
use git

if [[ "${1:-}" == --help ]]
then
  echo "usage: kit status [--no-fetch]" >&2
  exit 0
fi

git:status "$@"

# ----------------------------------------------------------------------------------------------------------------------
