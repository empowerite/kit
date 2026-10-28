#!/usr/bin/env bash
# kit-issue - start or resume an issue (kit verb)
# ----------------------------------------------------------------------------------------------------------------------
#
# kit issue [<reference>…]
#
# Starts work on an issue, or resumes it. kit issue makes each of these, and reuses any that already exists:
#
#   - a branch linked to the issue
#   - a worktree for the branch
#   - a start commit
#   - a draft pull request (a merge request on GitLab)
#
# A reference names the issue and its tracker:
#
#   - `7` or `#7` for an issue on the forge (the service that hosts the repository, GitHub or GitLab)
#   - `AB#4521` for an Azure DevOps work item, when that tracker is declared in .kit/settings.yml
#
# With no reference, at a terminal, kit issue asks for a title and creates the issue.
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
