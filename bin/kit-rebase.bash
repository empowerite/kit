#!/usr/bin/env bash
# kit-rebase - rebase a worktree's branch onto the default branch and push it; never the default branch (kit verb, run through bin/kit)
# ----------------------------------------------------------------------------------------------------------------------
#
# The default branch is the human's desk and is never rebased by an agent: this verb refuses it, and 1st.md says an
# agent spells a rebase this way and no other. Any other branch is rebased onto origin/<default> and pushed with
# --force-with-lease, which fails rather than overwrite a push that landed since. On a conflict the rebase is
# aborted and the worktree left as it was.

set -o errexit -o nounset -o pipefail
# shellcheck source=bash/1st.sh
source "${kit_root:?run through bin/kit}/bash/1st.sh"

if [[ "${1:-}" == --help ]]
then
  echo "usage: kit rebase [<path>]    (no path: the checkout this runs in)" >&2
  exit 0
fi

guard:require tool:git
worktree=${1:-.}
branch=$(git -C "${worktree}" symbolic-ref --quiet --short HEAD 2>/dev/null || true)
if [[ -z "${branch}" ]]
then
  error "${worktree} has no branch checked out, so there is nothing to rebase"
  exit 1
fi
default_branch=$(git -C "${worktree}" symbolic-ref --quiet --short refs/remotes/origin/HEAD 2>/dev/null | sed 's@^origin/@@' || true)
[[ -n "${default_branch}" ]] || default_branch=main
if [[ "${branch}" == "${default_branch}" ]]
then
  error "refusing to rebase ${branch}: it is the default branch, the human's desk. It advances by fast-forward: git pull --ff-only"
  exit 1
fi

section "kit rebase ${branch} onto origin/${default_branch}"
git -C "${worktree}" fetch --quiet --prune origin
if ! git -C "${worktree}" rebase --quiet "origin/${default_branch}"
then
  git -C "${worktree}" rebase --abort
  error "${branch} conflicts with origin/${default_branch}; the rebase was aborted and the worktree is as it was. Resolve by hand."
  exit 1
fi
git -C "${worktree}" push --quiet --force-with-lease
ok "${branch} is rebased onto origin/${default_branch} and pushed."

# ----------------------------------------------------------------------------------------------------------------------
