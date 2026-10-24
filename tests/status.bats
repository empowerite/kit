#!/usr/bin/env bats
# status.bats - the parts of the picture that need no GitHub: a cell, the commit list, and a worktree's row
# ----------------------------------------------------------------------------------------------------------------------

setup() {
  export kit_root NO_COLOR=1
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

# in_status <command>… runs a command with the status module loaded.
in_status() {
  bash -c 'source "${kit_root}/bin/1st.sh" && use status && "$@"' _ "$@"
}

@test "a cell is padded to its width, or clipped with an ellipsis" {
  run in_status status:cell abc 5
  [ "$output" = "abc  " ]
  run in_status status:cell abcdefgh 5
  [ "$output" = "abcd…" ]
}

@test "the commit list shows the newest, says how many are hidden, and ends with the oldest" {
  git worktree add --quiet -b topic worktrees/topic main
  for i in $(seq 1 15)
  do
    printf '%s\n' "${i}" > "worktrees/topic/f${i}" && git -C worktrees/topic add . && git -C worktrees/topic commit --quiet --message "c${i}"
  done
  run in_status status:commits "${PWD}/worktrees/topic" main..topic
  [ "$status" -eq 0 ]
  [ "${#lines[@]}" -eq 11 ]
  [[ "${lines[0]}" == *" c15" ]]
  [[ "${lines[8]}" == *" c7" ]]
  [[ "${lines[9]}" == *"… and 5 more" ]]
  [[ "${lines[10]}" == *" c1" ]]
}

@test "a worktree with no pull request shows its distance from main and the size of its diff" {
  git worktree add --quiet -b topic worktrees/topic main
  printf 'two\nthree\n' > worktrees/topic/b && git -C worktrees/topic add b && git -C worktrees/topic commit --quiet --message two
  printf 'four\n' > c && git add c && git commit --quiet --message four && git push --quiet origin main
  run in_status status:worktree-alone "${PWD}/worktrees/topic" topic 20 "${PWD}" main
  [ "$status" -eq 0 ]
  [ "$output" = "🌿 worktrees/topic        ↑1 ↓1 · +2 -0" ]
}

@test "a detached worktree is marked, and a dirty one counts its uncommitted paths" {
  git worktree add --quiet --detach worktrees/loose main
  printf 'x\n' > worktrees/loose/x
  run in_status status:worktree-alone "${PWD}/worktrees/loose" '' 16 "${PWD}" main
  [ "$status" -eq 0 ]
  [ "$output" = "🌿 worktrees/loose    ❗ detached HEAD   ↑0 ↓0 · +0 -0 · 1 uncommitted" ]
}
