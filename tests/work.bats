#!/usr/bin/env bats
# work.bats - kit work through a fake of the forge's tool: the worktree it makes is hydrated before the start commit
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

@test "kit work makes the worktree, hydrates it, and the start commit carries nothing kit placed" {
  "${kit_checkout}/bin/kit" install >/dev/null 2>&1
  printf 'forge: github\n' >> kit.yml
  git add --all && git commit --quiet --message init
  world
  run "${kit_checkout}/bin/kit" work 7
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

@test "kit work with nothing is refused: a reference or a title, and no prompt remains" {
  run "${kit_checkout}/bin/kit" work </dev/null
  [ "$status" -eq 2 ]
  [[ "$output" == *"kit work needs <reference> or --title or --pr; run: kit help work"* ]]
  [ "$(grep -c 'read -r -p' "${kit_checkout}/modules/bash/issue.sh")" -eq 0 ]
  # the module takes the title as an argument, and reads nothing the argument check left behind
  [ "$(grep -c syntax_ "${kit_checkout}/modules/bash/issue.sh")" -eq 0 ]
  run "${kit_checkout}/bin/kit" work --pr 3 --title 'Fix the thing' 7
  [ "$status" -eq 2 ]
  [[ "$output" == *"--title cannot be combined with --pr; run: kit help work"* ]]
  run "${kit_checkout}/bin/kit" work --pr 3
  [ "$status" -eq 2 ]
}

@test "kit work --title with issues gathers them under a new parent and starts it, the one change closing them all" {
  "${kit_checkout}/bin/kit" install >/dev/null 2>&1
  printf 'forge: github\n' >> kit.yml
  git add --all && git commit --quiet --message init
  world
  git push --quiet origin main:12-the-parent
  fixture created.txt <<<'https://github.com/empowerite/example/issues/12'
  fixture parent.json <<<'{"title": "The parent", "state": "OPEN", "assignees": []}'
  fixture other.json <<<'{"title": "Other", "state": "OPEN", "assignees": []}'
  fixture heads.json <<<'[{"headRefName": "12-the-parent"}]'
  fixture pr-open-12.json <<<'[{"number": 5}]'
  fixture pr-5.json <<<'{"title": "The parent", "body": ""}'
  route 'issue create --title The parent --body *' created.txt
  route 'issue edit 12 --add-sub-issue *' '!0'
  route 'issue view 12 *' parent.json
  route 'issue view 9 *' other.json
  route 'issue edit 12 --add-assignee ada' '!0'
  route 'issue edit 9 --add-assignee ada' '!0'
  sed -i.bak '/open-heads.json/d' "${FAKE_DIR}/gh.routes"
  route 'pr list --state open --json headRefName*' heads.json
  route 'pr list --head 12-the-parent --state open*' pr-open-12.json
  route 'pr view 5 --json *' pr-5.json
  route 'pr edit 5 --body *' '!0'
  run "${kit_checkout}/bin/kit" work --title 'The parent' 7 9
  [ "$status" -eq 0 ]
  [[ "$output" == *"Created #12: The parent"* ]]
  [[ "$output" == *"Gathered #7 #9 under #12"* ]]
  grep -q '^issue edit 12 --add-sub-issue 7$' "${FAKE_DIR}/gh.calls"
  grep -q '^issue edit 12 --add-sub-issue 9$' "${FAKE_DIR}/gh.calls"
  grep -q '^issue edit 7 --add-assignee ada$' "${FAKE_DIR}/gh.calls"
  grep -q '^pr edit 5 --body' "${FAKE_DIR}/gh.calls"
  [ -d worktrees/12-the-parent ]
}

@test "kit work --pr adds issues to the caller's own open change, and refuses someone else's with the way to hand them over" {
  git commit --quiet --allow-empty --message init
  world
  fixture pr-state.json <<<'{"state": "OPEN", "mergeCommit": null}'
  fixture pull-bob.json <<<'{"user": {"login": "bob"}}'
  fixture pull-ada.json <<<'{"user": {"login": "ada"}}'
  fixture pr-4.json <<<'{"title": "Mine", "body": "Closes #2"}'
  printf 'forge: github\n' >> kit.yml
  route 'pr view 4 --json state,mergeCommit*' pr-state.json
  route 'pr view 6 --json state,mergeCommit*' pr-state.json
  route 'api repos/empowerite/example/pulls/4 *' pull-ada.json
  route 'api repos/empowerite/example/pulls/6 *' pull-bob.json
  route 'pr view 4 --json *' pr-4.json
  route 'pr edit 4 --body *' '!0'
  run "${kit_checkout}/bin/kit" work --pr 6 7
  [ "$status" -eq 1 ]
  [[ "$output" == *"pull request #6 is bob's; hand the issues over instead: kit assign --to bob #7"* ]]
  run "${kit_checkout}/bin/kit" work --pr 4 7
  [ "$status" -eq 0 ]
  [[ "$output" == *"Added #7 to pull request #4"* ]]
  grep -q '^issue edit 7 --add-assignee ada$' "${FAKE_DIR}/gh.calls"
  run grep -A1 '^pr edit 4 --body' "${FAKE_DIR}/gh.calls"
  [[ "$output" == *"Closes #7"* ]]
}

@test "kit assign adds the caller or a login, kit unassign takes the caller or everyone off, and each says what it did" {
  git commit --quiet --allow-empty --message init
  world
  printf 'forge: github\n' >> kit.yml
  route 'issue edit 7 --add-assignee carol' '!0'
  route 'issue edit 7 --remove-assignee *' '!0'
  run "${kit_checkout}/bin/kit" assign 7
  [ "$status" -eq 0 ]
  [[ "$output" == *"#7 is already assigned to bob; adding you as well"* ]]
  run "${kit_checkout}/bin/kit" assign --to carol 7
  [ "$status" -eq 0 ]
  [[ "$output" == *"#7 assigned to carol"* ]]
  run "${kit_checkout}/bin/kit" unassign 7
  [ "$status" -eq 0 ]
  [[ "$output" == *"#7 is not assigned to you"* ]]
  run "${kit_checkout}/bin/kit" unassign --all 7
  [ "$status" -eq 0 ]
  [[ "$output" == *"#7 no longer assigned to bob"* ]]
  grep -q '^issue edit 7 --remove-assignee bob$' "${FAKE_DIR}/gh.calls"
}

@test "kit close closes as not planned, or as a duplicate, never an issue as its own, and leaves a closed one be" {
  git commit --quiet --allow-empty --message init
  world
  printf 'forge: github\n' >> kit.yml
  fixture closed.json <<<'{"title": "Old", "state": "CLOSED", "assignees": []}'
  route 'issue view 8 *' closed.json
  route 'issue close 7 *' '!0'
  run "${kit_checkout}/bin/kit" close --not-planned 7 8
  [ "$status" -eq 0 ]
  [[ "$output" == *"#7 closed as not planned"* ]]
  [[ "$output" == *"#8 is closed already"* ]]
  run "${kit_checkout}/bin/kit" close --duplicate-of 9 7
  [ "$status" -eq 0 ]
  [[ "$output" == *"#7 closed as duplicate of #9"* ]]
  grep -q '^issue close 7 --reason duplicate --duplicate-of 9$' "${FAKE_DIR}/gh.calls"
  run "${kit_checkout}/bin/kit" close --duplicate-of 7 7
  [ "$status" -eq 1 ]
  [[ "$output" == *"#7 cannot be a duplicate of itself"* ]]
  run "${kit_checkout}/bin/kit" close 7
  [ "$status" -eq 2 ]
}

@test "kit gather creates a parent with the issues under it, and starts nothing" {
  git commit --quiet --allow-empty --message init
  world
  printf 'forge: github\n' >> kit.yml
  fixture created.txt <<<'https://github.com/empowerite/example/issues/12'
  route 'issue create --title Plan --body *' created.txt
  route 'issue edit 12 --add-sub-issue *' '!0'
  run --separate-stderr "${kit_checkout}/bin/kit" gather --title Plan 7 9
  [ "$status" -eq 0 ]
  [ "$output" = "#12" ]
  [[ "$stderr" == *"Gathered #7 #9 under #12"* ]]
  [ ! -d worktrees ]
}

@test "kit work --title creates the issue and starts it in one step" {
  "${kit_checkout}/bin/kit" install >/dev/null 2>&1
  printf 'forge: github\n' >> kit.yml
  git add --all && git commit --quiet --message init
  world
  fixture created.txt <<<'https://github.com/empowerite/example/issues/7'
  route 'issue create --title Fix the thing *' created.txt
  run "${kit_checkout}/bin/kit" work --title 'Fix the thing'
  [ "$status" -eq 0 ]
  [[ "$output" == *"Created #7: Fix the thing"* ]]
  [ -d worktrees/7-fix-the-thing ]
  [ "$(git -C worktrees/7-fix-the-thing log --format=%s -1)" = "#7 start: Fix the thing" ]
}

@test "kit work sweeps the desk's uncommitted work into the worktree, and leaves the desk and the stash clean" {
  "${kit_checkout}/bin/kit" install >/dev/null 2>&1
  printf 'forge: github\n' >> kit.yml
  printf 'notes\n' > notes.txt
  git add --all && git commit --quiet --message init
  world
  printf 'mine\n' > notes.txt
  printf 'scratch\n' > scratch.txt
  run "${kit_checkout}/bin/kit" work 7
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
  run "${kit_checkout}/bin/kit" work 7
  [ "$status" -eq 1 ]
  stash=$(git rev-parse 'stash@{0}')
  [[ "$output" == *"the swept work conflicts in "*"worktrees/7-fix-the-thing. It is kept in the stash: "* ]]
  [[ "$output" == *"It is kept in the stash: git stash apply ${stash}"* ]]
  [ "$(git stash list | wc -l | tr -d ' ')" -eq 1 ]
  [ "$(cat notes.txt)" = notes ]
  [ -z "$(git status --porcelain)" ]
}

# ----------------------------------------------------------------------------------------------------------------------
