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
  [[ "$output" == *"run: cd "*" && gh auth status (the pull requests on 8-idea could not be read)"* ]]
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

# waiting <log> starts kit tidy --wait=7 in the background, asking every tenth of a second, its output in <log>, and
# puts its process in the variable pid. Job control is on while it starts, so it leads a process group of its own,
# as a terminal's foreground job does. A suite started in the background gives every process an interrupt that is
# ignored, which no bash can take back: perl takes it back before it execs, as bots.bats does.
waiting() {
  set -m
  KIT_TIDY_WAIT_SECONDS=0.1 perl -e '$SIG{INT} = "DEFAULT"; exec @ARGV or die "exec: $!"' \
    "${kit_root}/bin/kit" tidy --wait=7 3>&- > "$1" 2>&1 &
  pid=$!
  set +m
}

# ended <pid> waits, thirty seconds at most, for the process to end, and kills its group when it does not.
ended() {
  local i
  for (( i = 0; i < 300; i++ ))
  do
    kill -0 "$1" 2>/dev/null || return 0
    sleep 0.1
  done
  kill -KILL -- "-$1" 2>/dev/null
  return 1
}

# asked <n> waits, ten seconds at most, until the forge has been asked for the change on 7-fix-the-thing <n> times.
asked() {
  local i n
  for (( i = 0; i < 100; i++ ))
  do
    n=$(grep -c '^pr list --head 7-fix-the-thing --state all' "${FAKE_DIR}/gh.calls" 2>/dev/null) || true
    (( ${n:-0} >= $1 )) && return 0
    sleep 0.1
  done
  return 1
}

@test "kit tidy --wait asks until the change merges, then sweeps that worktree and its branch alone" {
  local status=0
  world
  worktree-on 7-fix-the-thing && gone 7-fix-the-thing
  worktree-on 9-other && gone 9-other
  change-on 7-fix-the-thing OPEN
  change-on 9-other MERGED
  waiting "${BATS_TEST_TMPDIR}/wait.log"
  asked 2
  [ -d worktrees/7-fix-the-thing ]
  change-on 7-fix-the-thing MERGED
  ended "${pid}"
  wait "${pid}" || status=$?
  [ "${status}" -eq 0 ]
  output=$(cat "${BATS_TEST_TMPDIR}/wait.log")
  [[ "$output" == *"waiting for the change on 7-fix-the-thing to merge"* ]]
  [[ "$output" == *"the change on 7-fix-the-thing merged; sweeping its worktree"* ]]
  [[ "$output" == *"removing "*"/worktrees/7-fix-the-thing (7-fix-the-thing, MERGED)"* ]]
  [[ "$output" == *"deleting 7-fix-the-thing"* ]]
  [ ! -e worktrees/7-fix-the-thing ]
  run ! git show-ref --verify --quiet refs/heads/7-fix-the-thing
  # another worktree, though its change merged, is not this wait's to sweep
  [ -d worktrees/9-other ]
  git show-ref --verify --quiet refs/heads/9-other
  [[ "$output" != *"9-other"* ]]
}

@test "kit tidy --wait says a change closed unmerged, keeps the worktree, and exits 1" {
  world
  worktree-on 7-fix-the-thing && gone 7-fix-the-thing
  change-on 7-fix-the-thing CLOSED
  KIT_TIDY_WAIT_SECONDS=0.1 run "${kit_root}/bin/kit" tidy --wait=#7
  [ "$status" -eq 1 ]
  [[ "$output" == *"the change on 7-fix-the-thing closed unmerged; "*"/worktrees/7-fix-the-thing is kept"* ]]
  [[ "$output" != *"removing"* ]]
  [ -d worktrees/7-fix-the-thing ]
  git show-ref --verify --quiet refs/heads/7-fix-the-thing
}

@test "kit tidy --wait keeps a merged change's worktree with uncommitted work, as kit tidy does, and exits 1" {
  world
  worktree-on 7-fix-the-thing && gone 7-fix-the-thing
  change-on 7-fix-the-thing MERGED
  printf 'x\n' > worktrees/7-fix-the-thing/x
  KIT_TIDY_WAIT_SECONDS=0.1 run "${kit_root}/bin/kit" tidy --wait=7
  [ "$status" -eq 1 ]
  [[ "$output" == *"keeping "*"/worktrees/7-fix-the-thing (7-fix-the-thing, it has uncommitted changes)"* ]]
  [[ "$output" == *"7-fix-the-thing merged, and kit tidy kept "*"/worktrees/7-fix-the-thing; see above"* ]]
  [ -f worktrees/7-fix-the-thing/x ]
  git show-ref --verify --quiet refs/heads/7-fix-the-thing
  # it never forces: git is never asked to remove a worktree by force
  run ! grep -rq -- 'worktree remove --force\|worktree remove -f' "${kit_root}/modules/bash/git.sh"
}

@test "ctrl-c stops kit tidy --wait, which removes nothing and exits 130" {
  local status=0
  world
  worktree-on 7-fix-the-thing && gone 7-fix-the-thing
  change-on 7-fix-the-thing OPEN
  waiting "${BATS_TEST_TMPDIR}/wait.log"
  asked 1
  # ctrl-c signals the terminal's whole foreground process group: the loader, and the verb it runs
  kill -INT -- "-${pid}"
  ended "${pid}"
  wait "${pid}" || status=$?
  [ "${status}" -eq 130 ]
  output=$(cat "${BATS_TEST_TMPDIR}/wait.log")
  [[ "$output" == *"stopped waiting for the change on 7-fix-the-thing; nothing was removed, and "* ]]
  [[ "$output" == *"/worktrees/7-fix-the-thing is kept"* ]]
  [ -d worktrees/7-fix-the-thing ]
  git show-ref --verify --quiet refs/heads/7-fix-the-thing
}

@test "kit tidy --wait refuses no worktree, the worktree it runs in, a branch with no change, and a bad interval" {
  world
  run "${kit_root}/bin/kit" tidy --wait=7
  [ "$status" -eq 1 ]
  [[ "$output" == *"no worktree under worktrees/ is on a branch for #7, so there is nothing to wait for;"* ]]
  [[ "$output" == *"run: cd "*" && kit work #7"* ]]
  worktree-on 7-fix-the-thing
  # in the worktree itself, the remedy is to run it from the desk
  (cd worktrees/7-fix-the-thing && "${kit_root}/bin/kit" tidy --wait=7) > "${BATS_TEST_TMPDIR}/here.log" 2>&1 && false
  grep -q "run: cd .* && kit tidy --wait=#7 (kit tidy keeps the worktree it runs in, " "${BATS_TEST_TMPDIR}/here.log"
  fixture pr-none.json <<<'[]'
  route 'pr list --head 7-fix-the-thing --state all*' pr-none.json
  KIT_TIDY_WAIT_SECONDS=0.1 run "${kit_root}/bin/kit" tidy --wait=7
  [ "$status" -eq 1 ]
  [[ "$output" == *"7-fix-the-thing has no change to wait for; "*"/worktrees/7-fix-the-thing is kept"* ]]
  KIT_TIDY_WAIT_SECONDS=0 run "${kit_root}/bin/kit" tidy --wait=7
  [ "$status" -eq 2 ]
  [[ "$output" == *"KIT_TIDY_WAIT_SECONDS must be a number of seconds above 0, not '0'"* ]]
  run "${kit_root}/bin/kit" tidy --wait=7 --dry-run
  [ "$status" -eq 2 ]
  [ -d worktrees/7-fix-the-thing ]
}

@test "kit tidy --wait asks again when a read fails, and stops after three failures in a row, keeping the worktree" {
  world
  worktree-on 7-fix-the-thing
  route 'pr list --head 7-fix-the-thing --state all*' '!1'
  KIT_TIDY_WAIT_SECONDS=0.1 run "${kit_root}/bin/kit" tidy --wait=7
  [ "$status" -eq 1 ]
  [ "$(grep -c '^pr list --head 7-fix-the-thing --state all' "${FAKE_DIR}/gh.calls")" -eq 3 ]
  [[ "$output" == *"the change on 7-fix-the-thing could not be read; asking again in 0.1s"* ]]
  [[ "$output" == *"read three times in a row, so the wait stops; "*"/worktrees/7-fix-the-thing is kept"* ]]
  [ -d worktrees/7-fix-the-thing ]
}
