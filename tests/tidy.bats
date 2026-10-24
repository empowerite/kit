#!/usr/bin/env bats
# tidy.bats - the one decision in tidy that needs no GitHub: a clean detached worktree is residue only when the
# default branch already has its commit
# ----------------------------------------------------------------------------------------------------------------------

setup() {
  export kit_root
  kit_root=$(cd "${BATS_TEST_DIRNAME}/.." && pwd)
  work=$(mktemp -d)
  cd "${work}"
  git init --quiet --bare --initial-branch=main origin.git
  git clone --quiet origin.git desk 2>/dev/null
  cd desk
  git config user.name t && git config user.email t@example.com && git config commit.gpgsign false
  printf 'one\n' > a && git add a && git commit --quiet --message one && git push --quiet origin main
  git remote set-head origin main
  mkdir worktrees
}

teardown() {
  rm -rf "${work}"
}

# decide <path> runs tidy's per-worktree decision on <path>, a detached checkout, with tidy's globals in place.
decide() {
  bash -c '
    source "${kit_root}/bin/1st.sh" && use git
    found=false; dry_run=false; removed_worktrees=0; declare -A freed=()
    self_root=$(git rev-parse --show-toplevel); default_branch=main
    git:tidy-worktree "$1" "" true
  ' _ "$1"
}

@test "a clean detached worktree at a commit on main is removed" {
  git worktree add --quiet --detach worktrees/old HEAD
  run decide "${PWD}/worktrees/old"
  [ "$status" -eq 0 ]
  [[ "$output" == *"removing"* ]]
  [ ! -e worktrees/old ]
}

@test "a clean detached worktree at a commit main does not reach is kept" {
  git worktree add --quiet --detach worktrees/only HEAD
  printf 'two\n' > worktrees/only/b && git -C worktrees/only add b && git -C worktrees/only commit --quiet --message two
  run decide "${PWD}/worktrees/only"
  [ "$status" -eq 0 ]
  [[ "$output" == *"keeping"* ]]
  [ -e worktrees/only ]
}

# ----------------------------------------------------------------------------------------------------------------------
