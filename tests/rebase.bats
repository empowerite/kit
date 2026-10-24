#!/usr/bin/env bats
# rebase.bats - kit rebase refuses the desk and the default branch, and brings a worktree's branch up to date
# ----------------------------------------------------------------------------------------------------------------------

setup() {
  kit_checkout=$(cd "${BATS_TEST_DIRNAME}/.." && pwd)
  work=$(mktemp -d)
  cd "${work}"
  git init --quiet --bare --initial-branch=main origin.git
  git clone --quiet origin.git desk 2>/dev/null
  cd desk
  git config user.name t && git config user.email t@example.com && git config commit.gpgsign false
  printf 'one\n' > a && git add a && git commit --quiet --message one && git push --quiet origin main
  git remote set-head origin main
}

teardown() {
  rm -rf "${work}"
}

# worktree makes a worktree on branch 1-x beside the desk, configured to commit.
worktree() {
  git worktree add --quiet ../wt -b 1-x
  git -C ../wt config user.name t
  git -C ../wt config user.email t@example.com
  git -C ../wt config commit.gpgsign false
}

# commit-in <checkout> <file> <content> writes the file and commits it there.
commit-in() {
  printf '%s\n' "$3" > "$1/$2"
  git -C "$1" add "$2"
  git -C "$1" commit --quiet --message "$2"
}

@test "rebase refuses the root checkout, whatever branch it has" {
  run "${kit_checkout}/bin/kit" rebase
  [ "$status" -eq 1 ]
  [[ "$output" == *"refusing to rebase main: it is checked out at the root"* ]]
  git switch --quiet --create desk-topic
  run "${kit_checkout}/bin/kit" rebase
  [ "$status" -eq 1 ]
  [[ "$output" == *"refusing to rebase desk-topic: it is checked out at the root"* ]]
}

@test "rebase refuses the default branch in a worktree" {
  git switch --quiet --create desk-topic
  git worktree add --quiet ../wt main
  run "${kit_checkout}/bin/kit" rebase ../wt
  [ "$status" -eq 1 ]
  [[ "$output" == *"refusing to rebase main: it is the default branch"* ]]
}

@test "rebase moves a worktree's branch onto origin/main and pushes it" {
  worktree
  commit-in ../wt b two && git -C ../wt push --quiet --set-upstream origin 1-x
  commit-in . c three && git push --quiet origin main
  run "${kit_checkout}/bin/kit" rebase ../wt
  [ "$status" -eq 0 ]
  [ "$(git -C ../wt rev-parse HEAD~1)" = "$(git rev-parse origin/main)" ]
  [ "$(git -C ../wt rev-parse HEAD)" = "$(git -C ../wt rev-parse origin/1-x)" ]
}

@test "rebase aborts on a conflict and leaves the worktree as it was" {
  worktree
  commit-in ../wt a theirs && git -C ../wt push --quiet --set-upstream origin 1-x
  commit-in . a ours && git push --quiet origin main
  before=$(git -C ../wt rev-parse HEAD)
  run "${kit_checkout}/bin/kit" rebase ../wt
  [ "$status" -eq 1 ]
  [[ "$output" == *"conflicts with origin/main"* ]]
  [ "$(git -C ../wt rev-parse HEAD)" = "${before}" ]
  [ -z "$(git -C ../wt status --porcelain)" ]
}

# ----------------------------------------------------------------------------------------------------------------------
