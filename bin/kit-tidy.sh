#!/usr/bin/env bash
# kit-tidy - remove merged worktrees and gone branches (kit verb)
# ----------------------------------------------------------------------------------------------------------------------
#
# kit tidy [--dry-run]
#
# Removes finished worktrees under worktrees/, and deletes each local branch whose upstream is gone. A worktree is
# finished when its pull request is merged or closed, or when it is detached, clean, and at a commit the default
# branch already has. kit tidy also prunes git's worktree records. It never touches:
#
#   - the default branch
#   - a branch checked out in a live worktree
#   - a worktree with uncommitted changes
#   - a detached worktree whose commit nothing else reaches
#   - the worktree it runs in
#
# --dry-run prints what kit tidy would do, and changes nothing.
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
