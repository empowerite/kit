#!/usr/bin/env bats
# tidy.bats - the one decision in tidy that needs no GitHub: a clean detached worktree is residue only when the
# default branch already has its commit
# ----------------------------------------------------------------------------------------------------------------------

setup() {
  export kit_root
  kit_root=$(cd "${BATS_TEST_DIRNAME}/.." && pwd)
  export KIT_HOME="${BATS_TEST_TMPDIR}/home" XDG_CACHE_HOME="${BATS_TEST_TMPDIR}/cache"
  mkdir -p "${KIT_HOME}" "${XDG_CACHE_HOME}"
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

# decide <path> runs tidy's decision on <path>, a detached checkout, with main as the default branch and no dry
# run.
decide() {
  bash -c 'source "${kit_root}/bash/1st.sh" && use git \
    && git:tidy-worktree "$1" "" true main "$(git rev-parse --show-toplevel)" false' _ "$1"
}

# fields prints every linked worktree as "<dir> <detached> [<branch>]", read the way the verbs read git:worktrees.
fields() {
  bash -c 'source "${kit_root}/bash/1st.sh" && use git && git:worktrees' \
    | while IFS=$'\t' read -r path detached branch
  do
    printf '%s %s [%s]\n' "${path##*/}" "${detached}" "${branch}"
  done
}

@test "the worktree list keeps a detached checkout's empty branch field where a tab read can keep it" {
  git worktree add --quiet --detach worktrees/loose HEAD
  git worktree add --quiet -b topic worktrees/topic HEAD
  run fields
  [ "$status" -eq 0 ]
  [[ "$output" == *"loose true []"* ]]
  [[ "$output" == *"topic false [topic]"* ]]
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
  [ "$status" -eq 1 ]
  [[ "$output" == *"keeping"* ]]
  [ -e worktrees/only ]
}

# ----------------------------------------------------------------------------------------------------------------------
