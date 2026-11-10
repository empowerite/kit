#!/usr/bin/env bats
# tidy.bats - kit tidy removes the worktrees whose change is done and the branches origin dropped, and keeps the rest
# ----------------------------------------------------------------------------------------------------------------------
# origin is a bare repository beside the desk, so git's part is real: a branch pushed there and checked out under
# worktrees/ is what kit work leaves, and deleting it on origin is what the forge does when the change merges. gh is
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

# change-on <branch> <state> [<head>] routes the latest change on <branch> as <state>, MERGED, CLOSED, or OPEN, with
# <head> as the commit the forge saw last on the branch.
change-on() {
  fixture "pr-$1.json" <<<"[{\"state\": \"$2\", \"headRefOid\": \"${3:-}\"}]"
  route "pr list --head $1 --state all*" "pr-$1.json"
}

# worktree-on <branch> pushes <branch> from main to origin and checks it out in worktrees/<branch>, tracking origin's,
# as kit work leaves it. gone <branch> deletes it on origin, as the forge does when its change merges.
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

@test "a worktree prune that fails is warned of, and kit tidy does not say it pruned" {
  world
  # shellcheck disable=SC2016  # the script is single-quoted for bash -c
  run bash -c 'source "${kit_root}/modules/bash/1st.sh" && use git
    git() { [[ "$1 $2" != "worktree prune" ]] || { echo "fatal: no prune" >&2; return 1; }; command git "$@"; }
    git:tidy'
  [ "$status" -eq 0 ]
  [[ "$output" == *"fatal: no prune"* ]]
  [[ "$output" == *"git worktree prune failed, so the records of removed worktrees stay; see above"* ]]
  [[ "$output" != *"pruned"* ]]
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

# commit-in <dir> <file> commits a new file in the checkout at <dir> and prints the commit.
commit-in() {
  printf '%s\n' "$2" > "$1/$2"
  git -C "$1" add "$2"
  git -C "$1" commit --quiet --message "$2"
  git -C "$1" rev-parse HEAD
}

# idea-on <branch> makes <branch> track origin's, as a branch of the human's does, and moves it to a commit no remote
# has, made with git commit-tree.
idea-on() {
  git push --quiet origin "main:$1"
  git branch --quiet --track "$1" "origin/$1"
  git update-ref "refs/heads/$1" "$(git commit-tree -p main -m "$1" 'main^{tree}')"
}

@test "kit tidy keeps a worktree, and a gone branch, whose commits no remote ever had, though the change merged" {
  world
  worktree-on 7-fix-the-thing
  lost=$(commit-in worktrees/7-fix-the-thing work.txt)
  gone 7-fix-the-thing
  change-on 7-fix-the-thing MERGED "$(git rev-parse main)"
  idea-on 8-idea
  idea=$(git rev-parse 8-idea)
  gone 8-idea
  # the forge has no change on 8-idea
  fixture none.json <<<'[]'
  route 'pr list --head 8-idea --state all*' none.json
  run "${kit_root}/bin/kit" tidy
  [ "$status" -eq 0 ]
  [[ "$output" == *"(7-fix-the-thing, change MERGED, 1 commit(s) no remote ever had)"* ]]
  [[ "$output" == *"keeping 8-idea (1 commit(s) no remote ever had)"* ]]
  [[ "$output" == *"removed 0 worktree(s) and 0 branch(es)."* ]]
  [ -d worktrees/7-fix-the-thing ]
  [ "$(git rev-parse refs/heads/7-fix-the-thing)" = "${lost}" ]
  [ "$(git rev-parse refs/heads/8-idea)" = "${idea}" ]
}

@test "kit tidy keeps a branch whose change's head this clone lacks, and names the fetch that brings the head" {
  local missing=0123456789abcdef0123456789abcdef01234567
  world
  worktree-on 7-fix-the-thing
  lost=$(commit-in worktrees/7-fix-the-thing work.txt)
  gone 7-fix-the-thing
  change-on 7-fix-the-thing MERGED "${missing}"
  idea-on 8-idea
  idea=$(git rev-parse 8-idea)
  gone 8-idea
  change-on 8-idea MERGED "${missing}"
  run "${kit_root}/bin/kit" tidy
  [ "$status" -eq 0 ]
  [[ "$output" == *"(7-fix-the-thing, change MERGED, 1 commit(s) no remote-tracking branch here reaches;"* ]]
  [[ "$output" == *"keeping 8-idea (1 commit(s) no remote-tracking branch here reaches;"* ]]
  [[ "$output" == *"this clone lacks the change's head ${missing}; fetch it, then run kit tidy again:"* ]]
  [[ "$output" == *"git fetch origin ${missing}"* ]]
  [[ "$output" != *"no remote ever had"* ]]
  [ -d worktrees/7-fix-the-thing ]
  [ "$(git rev-parse refs/heads/7-fix-the-thing)" = "${lost}" ]
  [ "$(git rev-parse refs/heads/8-idea)" = "${idea}" ]
}

@test "kit tidy removes a worktree, and deletes a gone branch, whose commits were pushed before the branch went" {
  world
  worktree-on 7-fix-the-thing
  pushed=$(commit-in worktrees/7-fix-the-thing work.txt)
  git -C worktrees/7-fix-the-thing push --quiet origin 7-fix-the-thing
  gone 7-fix-the-thing
  change-on 7-fix-the-thing MERGED "${pushed}"
  idea-on 8-idea
  git push --quiet origin 8-idea
  change-on 8-idea CLOSED "$(git rev-parse 8-idea)"
  gone 8-idea
  run "${kit_root}/bin/kit" tidy
  [ "$status" -eq 0 ]
  [[ "$output" == *"removing "*"/worktrees/7-fix-the-thing (7-fix-the-thing, MERGED)"* ]]
  [[ "$output" == *"deleting 7-fix-the-thing"* ]]
  [[ "$output" == *"deleting 8-idea"* ]]
  [[ "$output" == *"removed 1 worktree(s) and 2 branch(es)."* ]]
  [[ "$output" != *"no remote ever had"* ]]
  [ ! -e worktrees/7-fix-the-thing ]
  run ! git show-ref --verify --quiet refs/heads/7-fix-the-thing
  run ! git show-ref --verify --quiet refs/heads/8-idea
}

@test "kit tidy removes a squash-merged worktree, its commits on no branch of origin's, its tip the change's head" {
  world
  worktree-on 7-fix-the-thing
  commit-in worktrees/7-fix-the-thing one.txt >/dev/null
  head=$(commit-in worktrees/7-fix-the-thing two.txt)
  git -C worktrees/7-fix-the-thing push --quiet origin 7-fix-the-thing
  # the forge squashes the change's two commits into one new commit on main, then deletes the branch
  squash=$(git commit-tree -p origin/main -m '#7 Fix the thing (#3)' "${head}^{tree}")
  git push --quiet origin "${squash}:refs/heads/main"
  gone 7-fix-the-thing
  change-on 7-fix-the-thing MERGED "${head}"
  run "${kit_root}/bin/kit" tidy
  [ "$status" -eq 0 ]
  [[ "$output" == *"removing "*"/worktrees/7-fix-the-thing (7-fix-the-thing, MERGED)"* ]]
  [[ "$output" == *"deleting 7-fix-the-thing"* ]]
  [[ "$output" == *"removed 1 worktree(s) and 1 branch(es)."* ]]
  [ ! -e worktrees/7-fix-the-thing ]
  run ! git show-ref --verify --quiet refs/heads/7-fix-the-thing
}

@test "kit tidy keeps a gone branch whose change's head the forge cannot be read for, though the head was pushed" {
  local idea
  world
  idea-on 8-idea
  git push --quiet origin 8-idea
  idea=$(git rev-parse 8-idea)
  gone 8-idea
  # a read of the change that worked would give the branch's tip as its head, and the branch would go
  route 'pr list --head 8-idea --state all*' '!1'
  run "${kit_root}/bin/kit" tidy
  [ "$status" -eq 0 ]
  [[ "$output" == *"run: gh auth status (the pull requests on 8-idea could not be read)"* ]]
  [[ "$output" == *"keeping 8-idea (its commits could not be counted)"* ]]
  [[ "$output" == *"removed 0 worktree(s) and 0 branch(es)."* ]]
  [ "$(git rev-parse refs/heads/8-idea)" = "${idea}" ]
}

@test "kit tidy keeps a worktree git will not remove, says so, and goes on to the next" {
  world
  worktree-on 7-fix-the-thing && gone 7-fix-the-thing
  change-on 7-fix-the-thing MERGED
  worktree-on 8-next && gone 8-next
  change-on 8-next MERGED
  git worktree lock worktrees/7-fix-the-thing
  run "${kit_root}/bin/kit" tidy
  [ "$status" -eq 0 ]
  [[ "$output" == *"(7-fix-the-thing, MERGED, git would not remove it)"* ]]
  [[ "$output" == *"keeping 7-fix-the-thing (checked out in a worktree)"* ]]
  [[ "$output" == *"removed 1 worktree(s) and 1 branch(es)."* ]]
  [ -d worktrees/7-fix-the-thing ]
  git show-ref --verify --quiet refs/heads/7-fix-the-thing
  [ ! -e worktrees/8-next ]
  run ! git show-ref --verify --quiet refs/heads/8-next
}

# ----------------------------------------------------------------------------------------------------------------------
