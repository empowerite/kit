#!/usr/bin/env bash
# kit-tidy - remove merged worktrees and gone branches (kit verb)
# ----------------------------------------------------------------------------------------------------------------------
#
# see: bin/git.sh

set -o errexit -o nounset -o pipefail
# shellcheck source=bin/1st.sh
source "${kit_root:?run through bin/kit}/bin/1st.sh"
use git

if [[ "${1:-}" == --help ]]
then
  echo "usage: kit tidy [--dry-run]" >&2
  exit 0
fi

git:tidy "$@"

# ----------------------------------------------------------------------------------------------------------------------
