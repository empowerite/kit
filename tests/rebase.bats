#!/usr/bin/env bats
# rebase.bats - kit rebase refuses the default branch and brings a worktree's branch up to date
# ----------------------------------------------------------------------------------------------------------------------

setup() {
  kit_checkout=$(cd "${BATS_TEST_DIRNAME}/.." && pwd)
  work=$(mktemp -d)
  cd "${work}"
  git init --quiet --bare origin.git
  git clone --quiet origin.git desk 2>/dev/null
  cd desk
  git config user.name t && git config user.email t@example.com && git config commit.gpgsign false
  printf 'one\n' > a && git add a && git commit --quiet --message one && git push --quiet origin main
  git remote set-head origin main
}

teardown() {
  rm -rf "${work}"
}

worktree() {  # a worktree on branch 1-x, configured to commit
  git worktree add --quiet ../wt -b 1-x
  git -C ../wt config user.name t && git -C ../wt config user.email t@example.com && git -C ../wt config commit.gpgsign false
}

@test "rebase refuses the default branch" {
  run "${kit_checkout}/bin/kit" rebase
  [ "$status" -eq 1 ]
  [[ "$output" == *"refusing to rebase main"* ]]
}

@test "rebase moves a worktree's branch onto origin/main and pushes it" {
  worktree
  printf 'two\n' > ../wt/b && git -C ../wt add b && git -C ../wt commit --quiet --message two && git -C ../wt push --quiet --set-upstream origin 1-x
  printf 'three\n' > c && git add c && git commit --quiet --message three && git push --quiet origin main
  run "${kit_checkout}/bin/kit" rebase ../wt
  [ "$status" -eq 0 ]
  [ "$(git -C ../wt rev-parse HEAD~1)" = "$(git rev-parse origin/main)" ]
  [ "$(git -C ../wt rev-parse HEAD)" = "$(git -C ../wt rev-parse origin/1-x)" ]
}

@test "rebase aborts on a conflict and leaves the worktree as it was" {
  worktree
  printf 'theirs\n' > ../wt/a && git -C ../wt commit --quiet --all --message theirs && git -C ../wt push --quiet --set-upstream origin 1-x
  printf 'ours\n' > a && git commit --quiet --all --message ours && git push --quiet origin main
  before=$(git -C ../wt rev-parse HEAD)
  run "${kit_checkout}/bin/kit" rebase ../wt
  [ "$status" -eq 1 ]
  [[ "$output" == *"conflicts with origin/main"* ]]
  [ "$(git -C ../wt rev-parse HEAD)" = "${before}" ]
  [ -z "$(git -C ../wt status --porcelain)" ]
}

# ----------------------------------------------------------------------------------------------------------------------
