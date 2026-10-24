#!/usr/bin/env bash
# kit-issue - start or resume an issue: linked branch, worktree, start commit, draft pull request (kit verb, run through bin/kit)
# ----------------------------------------------------------------------------------------------------------------------
#
# see: bin/gh.sh

set -o errexit -o nounset -o pipefail
# shellcheck source=bin/1st.sh
source "${kit_root:?run through bin/kit}/bin/1st.sh"
use gh

if [[ "${1:-}" == --help ]]
then
  echo "usage: kit issue [<number>…]    (no number, at a terminal: ask for a title and create the issue)" >&2
  exit 0
fi

gh:issue "$@"

# ----------------------------------------------------------------------------------------------------------------------
