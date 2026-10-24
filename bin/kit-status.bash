#!/usr/bin/env bash
# kit-status - show every worktree with its branch, pull request, and state (kit verb, run through bin/kit)
# ----------------------------------------------------------------------------------------------------------------------
#
# see: bash/git.sh

set -o errexit -o nounset -o pipefail
# shellcheck source=bash/1st.sh
source "${kit_root:?run through bin/kit}/bash/1st.sh"
use git

if [[ "${1:-}" == --help ]]
then
  sed -n '2s/^# kit-[a-z]* - \(.*\) (kit verb, run through bin/kit)$/\1/p' "${BASH_SOURCE[0]}" >&2
  exit 0
fi

git:status "$@"

# ----------------------------------------------------------------------------------------------------------------------
