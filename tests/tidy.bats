#!/usr/bin/env bats
# tidy.bats - kit tidy removes the worktrees whose change is done and the branches origin dropped, and keeps the rest
# ----------------------------------------------------------------------------------------------------------------------
# origin is a bare repository beside the desk, so git's part is real: a branch pushed there and checked out under
# worktrees/ is what kit issue leaves, and deleting it on origin is what the forge does when the change merges. gh is
# tests/fakes/fake-tool, so a change's state is a fixture. The decision on a detached worktree needs no forge at all.

bats_require_minimum_version 1.5.0

load helper

setup() {
  work=$(mktemp -d)
  cd "${work}"
  git init --quiet --bare --initial-branch=main origin.git
  git clone --quiet origin.git desk 2>/dev/null
  cd desk
  printf 'one\n' > a && git add a && git commit --quiet --message one && git push --quiet origin main
  git remote set-head origin main
  mkdir worktrees
}

# decide <path> runs tidy's decision on <path>, a detached checkout, with main as the default branch and no dry
# run.
decide() {
  bash -c 'source "${kit_root}/modules/bash/1st.sh" && use git \
    && git:tidy-worktree "$1" "" true main "$(git rev-parse --show-toplevel)" false' _ "$1"
}

# fields prints every linked worktree as "<dir> <detached> [<branch>]", read the way the verbs read git:worktrees.
fields() {
  bash -c 'source "${kit_root}/modules/bash/1st.sh" && use git && git:worktrees' \
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

# world declares the forge, and routes a fake gh for the repository empowerite/example and the caller ada, so kit tidy
# can ask about a branch's change.
world() {
  fake-tools gh
  mkdir .kit && printf 'forge: github\n' > kit.yml
  git add kit.yml && git commit --quiet --message forge && git push --quiet origin main
  fixture user.json <<<'{"login": "ada"}'
  fixture repo.json <<<'{"nameWithOwner": "empowerite/example", "url": "https://github.com/empowerite/example"}'
  route 'api user*' user.json
  route 'repo view*' repo.json
}

# change-on <branch> <state> routes the latest change on <branch> as <state>: MERGED, CLOSED, or OPEN.
change-on() {
  fixture "pr-$1.json" <<<"[{\"state\": \"$2\"}]"
  route "pr list --head $1 --state all*" "pr-$1.json"
}

# worktree-on <branch> pushes <branch> from main to origin and checks it out in worktrees/<branch>, tracking origin's,
# as kit issue leaves it. gone <branch> deletes it on origin, as the forge does when its change merges.
worktree-on() {
  git push --quiet origin "main:$1"
  git worktree add --quiet -b "$1" "worktrees/$1" "origin/$1"
}
gone() { git push --quiet origin --delete "$1"; }

@test "kit tidy removes a clean worktree whose change is merged, and deletes its branch once origin has dropped it" {
  world
  worktree-on 7-fix-the-thing && gone 7-fix-the-thing
  change-on 7-fix-the-thing MERGED
  run "${kit_root}/bin/kit" tidy
  [ "$status" -eq 0 ]
  [[ "$output" == *"removing "*"/worktrees/7-fix-the-thing (7-fix-the-thing, MERGED)"* ]]
  [[ "$output" == *"deleting 7-fix-the-thing"* ]]
  [[ "$output" == *"Tidied empowerite/example: removed 1 worktree(s) and 1 branch(es)."* ]]
  [ ! -e worktrees/7-fix-the-thing ]
  run ! git show-ref --verify --quiet refs/heads/7-fix-the-thing
  [ "$(git worktree list | wc -l | tr -d ' ')" -eq 1 ]
}

@test "kit tidy --dry-run says what it would remove and delete, and changes nothing" {
  world
  worktree-on 7-fix-the-thing && gone 7-fix-the-thing
  change-on 7-fix-the-thing MERGED
  # a dry run fetches nothing, so what origin dropped is known from an earlier fetch
  git fetch --quiet --prune origin
  before=$(git worktree list --porcelain; git for-each-ref)
  run "${kit_root}/bin/kit" tidy --dry-run
  [ "$status" -eq 0 ]
  [[ "$output" == *"would remove "*"/worktrees/7-fix-the-thing (7-fix-the-thing, MERGED)"* ]]
  [[ "$output" == *"would delete 7-fix-the-thing"* ]]
  [[ "$output" == *"Dry run: would remove 1 worktree(s) and 1 branch(es)."* ]]
  [ "$(git worktree list --porcelain; git for-each-ref)" = "${before}" ]
  [ -z "$(git -C worktrees/7-fix-the-thing status --porcelain)" ]
}

@test "in a fork, kit tidy fetches every remote, the primary first, so a branch the fork dropped is deleted" {
  world
  git init --quiet --bare --initial-branch=main ../upstream.git
  git remote add upstream ../upstream.git
  git push --quiet upstream main
  printf 'forge: github\nprimary: upstream\n' > kit.yml
  worktree-on 7-fix-the-thing && gone 7-fix-the-thing
  change-on 7-fix-the-thing MERGED
  run "${kit_root}/bin/kit" tidy --dry-run
  [[ "$output" == *"would run: git fetch --prune upstream"*"would run: git fetch --prune origin"* ]]
  run "${kit_root}/bin/kit" tidy
  [ "$status" -eq 0 ]
  [[ "$output" == *"git fetch --prune upstream"*"git fetch --prune origin"* ]]
  [[ "$output" == *"deleting 7-fix-the-thing"* ]]
  run ! git show-ref --verify --quiet refs/heads/7-fix-the-thing
}

@test "kit tidy keeps a branch checked out in a worktree it keeps, though origin has dropped the branch" {
  world
  worktree-on 9-other && gone 9-other
  change-on 9-other OPEN
  run "${kit_root}/bin/kit" tidy
  [ "$status" -eq 0 ]
  [[ "$output" == *"keeping "*"/worktrees/9-other (9-other, change OPEN)"* ]]
  [[ "$output" == *"keeping 9-other (checked out in a worktree)"* ]]
  [[ "$output" == *"removed 0 worktree(s) and 0 branch(es)."* ]]
  [ -d worktrees/9-other ]
  git show-ref --verify --quiet refs/heads/9-other
}

@test "kit tidy never deletes the default branch, even when origin has dropped it and no checkout has it" {
  world
  # the desk moves off main, so nothing but the rule for the default branch keeps it
  git switch --quiet --create elsewhere
  git -C ../origin.git config receive.denyDeleteCurrent ignore
  gone main
  run "${kit_root}/bin/kit" tidy
  [ "$status" -eq 0 ]
  git show-ref --verify --quiet refs/heads/main
  [ "$(git for-each-ref --format='%(upstream:track)' refs/heads/main)" = "[gone]" ]
  [[ "$output" != *"deleting main"* ]]
  [[ "$output" == *"removed 0 worktree(s) and 0 branch(es)."* ]]
}

@test "kit tidy keeps a worktree with uncommitted changes, and names it, whatever its change's state" {
  world
  worktree-on 7-fix-the-thing && gone 7-fix-the-thing
  change-on 7-fix-the-thing MERGED
  printf 'x\n' > worktrees/7-fix-the-thing/x
  run "${kit_root}/bin/kit" tidy
  [ "$status" -eq 0 ]
  [[ "$output" == *"keeping "*"/worktrees/7-fix-the-thing (7-fix-the-thing, it has uncommitted changes)"* ]]
  [[ "$output" == *"keeping 7-fix-the-thing (checked out in a worktree)"* ]]
  [ -f worktrees/7-fix-the-thing/x ]
  git show-ref --verify --quiet refs/heads/7-fix-the-thing
}

# ----------------------------------------------------------------------------------------------------------------------
