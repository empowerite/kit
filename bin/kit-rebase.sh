#!/usr/bin/env bash
# kit-rebase - rebase a worktree's branch onto the default branch and push it; never the default branch (kit verb, run through bin/kit)
# ----------------------------------------------------------------------------------------------------------------------
#
# see: bin/git.sh

set -o errexit -o nounset -o pipefail
# shellcheck source=bin/1st.sh
source "${kit_root:?run through bin/kit}/bin/1st.sh"
use git

if [[ "${1:-}" == --help ]]
then
  echo "usage: kit rebase [<path>]    (no path: the checkout this runs in)" >&2
  exit 0
fi

guard:require tool:git
git:rebase "$@"

# ----------------------------------------------------------------------------------------------------------------------
