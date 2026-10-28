#!/usr/bin/env bash
# kit-issue - start or resume an issue (kit verb)
# ----------------------------------------------------------------------------------------------------------------------
#
# A linked branch, a worktree, a start commit, and a draft change, for an issue in any declared tracker: `7` or
# `#7` for the forge's own, `AB#4521` for an Azure DevOps work item.
#
# see: bin/issue.sh, bin/forge.sh, bin/tracker.sh

set -o errexit -o nounset -o pipefail
# shellcheck source=bin/1st.sh
source "${kit_root:?run through bin/kit}/bin/1st.sh"
use issue

if [[ "${1:-}" == --help ]]
then
  echo "usage: kit issue [<reference>…]    (none, at a terminal: ask for a title and create the issue)" >&2
  exit 0
fi

issue:start "$@"

# ----------------------------------------------------------------------------------------------------------------------
