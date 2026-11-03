#!/usr/bin/env bats
# issue.bats - kit issue through a fake of the forge's tool: the worktree it makes is hydrated before the start commit
# ----------------------------------------------------------------------------------------------------------------------
# The world is forge-conformance.bats's: issue 7 "Fix the thing", assigned to bob, with the branch 7-fix-the-thing
# already on origin, as the forge makes it, and change 3 open on it. origin is a bare repository beside the test's,
# so git's part is real; gh is tests/fakes/fake-tool, so the forge's part is the provider's folding of fixtures.

bats_require_minimum_version 1.5.0

load helper

setup() {
  fake-tools gh
  fresh-repo
}

# world makes origin a bare repository holding main and the branch the forge made for issue 7, and routes gh.
world() {
  git init --quiet --bare "${repo}.origin.git"
  git remote add origin "${repo}.origin.git"
  git push --quiet origin main main:7-fix-the-thing
  : > "${FAKE_DIR}/gh.routes"
  fixture user.json <<<'{"login": "ada"}'
  fixture repo.json <<<'{"nameWithOwner": "empowerite/example", "url": "https://github.com/empowerite/example"}'
  fixture issue.json <<<'{"title": "Fix the thing", "state": "OPEN", "assignees": [{"login": "bob"}]}'
  fixture open-heads.json <<<'[{"headRefName": "9-other"}, {"headRefName": "7-fix-the-thing"}]'
  fixture pr-open.json <<<'[{"number": 3}]'
  fixture pr.json <<<'{"title": "Fix the thing", "body": "Closes #7"}'
  route 'api user*' user.json
  route 'repo view*' repo.json
  route 'issue view 7 *' issue.json
  route 'issue edit 7 --add-assignee ada' '!0'
  route 'pr list --state open --json headRefName*' open-heads.json
  route 'pr list --head 7-fix-the-thing --state open*' pr-open.json
  route 'pr view 3 --json *' pr.json
}

@test "kit issue makes the worktree, hydrates it, and the start commit carries nothing kit placed" {
  "${kit_checkout}/bin/kit" install >/dev/null 2>&1
  printf 'forge: github\n' >> kit.yml
  git add --all && git commit --quiet --message init
  world
  run "${kit_checkout}/bin/kit" issue 7
  [ "$status" -eq 0 ]
  [[ "$output" == *"Hydrating "*"worktrees/7-fix-the-thing"* ]]
  [ -d worktrees/7-fix-the-thing ]
  [ -f worktrees/7-fix-the-thing/.kit/1st.md ]
  [ ! -w worktrees/7-fix-the-thing/.kit/1st.md ]
  [ -f worktrees/7-fix-the-thing/.kit/.gitignore ]
  [ -z "$(git -C worktrees/7-fix-the-thing status --porcelain)" ]
  [ "$(git -C worktrees/7-fix-the-thing log --format=%s -1)" = "#7 start: Fix the thing" ]
  run ! git -C worktrees/7-fix-the-thing ls-files --error-unmatch .kit/1st.md
}

@test "a worktree that declares unreleased is hydrated by its own kit, the version read as YAML: spaces and a comment beside it do not matter" {
  # the worktree is the kit: kit/bin/kit, a stand-in that records what it was asked, and 1st.sh beside it
  mkdir -p wt/kit/bin wt/kit/modules/bash
  printf 'version:   unreleased   # built from this checkout\n' > wt/kit.yml
  printf '#!/usr/bin/env bash\nprintf "%%s\\n" "$*" > "$(dirname "$0")/../../called"\n' > wt/kit/bin/kit
  chmod +x wt/kit/bin/kit
  : > wt/kit/modules/bash/1st.sh
  run kit-bash "use issue && issue:hydrate '${PWD}/wt'"
  [ "$status" -eq 0 ]
  [[ "$output" == *"Hydrating ${PWD}/wt"* ]]
  [ "$(cat wt/called)" = hydrate ]
  # a kit.yml that declares no version, or no kit.yml at all, leaves the worktree alone
  rm wt/called
  printf 'forge: github\n' > wt/kit.yml
  run kit-bash "use issue && issue:hydrate '${PWD}/wt'"
  [ "$status" -eq 0 ]
  [ -z "$output" ]
  [ ! -e wt/called ]
  rm wt/kit.yml
  run kit-bash "use issue && issue:hydrate '${PWD}/wt'"
  [ "$status" -eq 0 ]
  [ -z "$output" ]
  # the module reads kit.yml as YAML, never with grep
  [ "$(grep -c 'grep.*kit\.yml' "${kit_checkout}/modules/bash/issue.sh")" -eq 0 ]
}

@test "kit issue with nothing is refused: a reference or a title, and no prompt remains" {
  run "${kit_checkout}/bin/kit" issue </dev/null
  [ "$status" -eq 2 ]
  [[ "$output" == *"kit issue needs <reference> or --title; run: kit help issue"* ]]
  [ "$(grep -c 'read -r -p' "${kit_checkout}/modules/bash/issue.sh")" -eq 0 ]
  # the module takes the title as an argument, and reads nothing the argument check left behind
  [ "$(grep -c syntax_ "${kit_checkout}/modules/bash/issue.sh")" -eq 0 ]
  run "${kit_checkout}/bin/kit" issue 7 --title 'Fix the thing'
  [ "$status" -eq 2 ]
  [[ "$output" == *"<reference> cannot be combined with --title; run: kit help issue"* ]]
}

@test "kit issue --title creates the issue and starts it in one step" {
  "${kit_checkout}/bin/kit" install >/dev/null 2>&1
  printf 'forge: github\n' >> kit.yml
  git add --all && git commit --quiet --message init
  world
  fixture created.txt <<<'https://github.com/empowerite/example/issues/7'
  route 'issue create --title Fix the thing *' created.txt
  run "${kit_checkout}/bin/kit" issue --title 'Fix the thing'
  [ "$status" -eq 0 ]
  [[ "$output" == *"Created #7: Fix the thing"* ]]
  [ -d worktrees/7-fix-the-thing ]
  [ "$(git -C worktrees/7-fix-the-thing log --format=%s -1)" = "#7 start: Fix the thing" ]
}

@test "kit issue sweeps the desk's uncommitted work into the worktree, and leaves the desk and the stash clean" {
  "${kit_checkout}/bin/kit" install >/dev/null 2>&1
  printf 'forge: github\n' >> kit.yml
  printf 'notes\n' > notes.txt
  git add --all && git commit --quiet --message init
  world
  printf 'mine\n' > notes.txt
  printf 'scratch\n' > scratch.txt
  run "${kit_checkout}/bin/kit" issue 7
  [ "$status" -eq 0 ]
  [[ "$output" == *"Sweeping the uncommitted work on main into the worktree"* ]]
  [[ "$output" == *"Swept the work into "*"worktrees/7-fix-the-thing, uncommitted"* ]]
  [ "$(cat worktrees/7-fix-the-thing/notes.txt)" = mine ]
  [ "$(cat worktrees/7-fix-the-thing/scratch.txt)" = scratch ]
  [ "$(git -C worktrees/7-fix-the-thing status --porcelain | sort)" = "$(printf ' M notes.txt\n?? scratch.txt')" ]
  [ "$(cat notes.txt)" = notes ]
  [ -z "$(git status --porcelain)" ]
  [ -z "$(git stash list)" ]
}

@test "a sweep that conflicts in the worktree keeps the stash, names the command that applies it, and stops" {
  "${kit_checkout}/bin/kit" install >/dev/null 2>&1
  printf 'forge: github\n' >> kit.yml
  printf 'notes\n' > notes.txt
  git add --all && git commit --quiet --message init
  world
  # the branch on origin changes notes.txt one way, and the desk another
  git switch --quiet --create theirs && printf 'theirs\n' > notes.txt && git commit --quiet --all --message theirs
  git push --quiet --force origin theirs:7-fix-the-thing
  git switch --quiet main && git branch --quiet --delete --force theirs
  printf 'mine\n' > notes.txt
  run "${kit_checkout}/bin/kit" issue 7
  [ "$status" -eq 1 ]
  stash=$(git rev-parse 'stash@{0}')
  [[ "$output" == *"the swept work conflicts in "*"worktrees/7-fix-the-thing. It is kept in the stash: "* ]]
  [[ "$output" == *"It is kept in the stash: git stash apply ${stash}"* ]]
  [ "$(git stash list | wc -l | tr -d ' ')" -eq 1 ]
  [ "$(cat notes.txt)" = notes ]
  [ -z "$(git status --porcelain)" ]
}

# ----------------------------------------------------------------------------------------------------------------------
