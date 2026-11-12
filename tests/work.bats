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

# desk-world installs the kit on the desk, declares the forge, commits notes.txt with it, and makes the world.
desk-world() {
  "${kit_checkout}/bin/kit" install >/dev/null 2>&1
  printf 'forge: github\n' >> kit.yml
  printf 'notes\n' > notes.txt
  git add --all && git commit --quiet --message init
  world
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
  # the worktree is the kit: kit/bin/kit, a stand-in that records what it was asked, and 1st.sh beside it; its .git
  # file makes it a clone of its own, whose kit.yml the loader reads
  mkdir -p wt/kit/bin wt/kit/modules/bash
  printf 'gitdir: %s/.git/worktrees/wt\n' "${PWD}" > wt/.git
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
  [[ "$output" == *"kit work needs <reference> or --title or --pr; run: cd "*" && kit help work"* ]]
  [ "$(grep -c 'read -r -p' "${kit_checkout}/modules/bash/issue.sh")" -eq 0 ]
  # the module takes the title as an argument, and reads nothing the argument check left behind
  [ "$(grep -c syntax_ "${kit_checkout}/modules/bash/issue.sh")" -eq 0 ]
  run "${kit_checkout}/bin/kit" work --pr 3 --title 'Fix the thing' 7
  [ "$status" -eq 2 ]
  [[ "$output" == *"--title cannot be combined with --pr; run: cd "*" && kit help work"* ]]
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

@test "kit assign exits 1 when an issue could not be assigned, and still assigns the next" {
  git commit --quiet --allow-empty --message init
  world
  printf 'forge: github\n' >> kit.yml
  fixture other.json <<<'{"title": "Other", "state": "OPEN", "assignees": []}'
  sed -i.bak '/^issue edit 7 --add-assignee/d' "${FAKE_DIR}/gh.routes"
  route 'issue edit 7 --add-assignee *' '!1'
  route 'issue view 9 *' other.json
  route 'issue edit 9 --add-assignee ada' '!0'
  run "${kit_checkout}/bin/kit" assign 7 9
  [ "$status" -eq 1 ]
  [[ "$output" == *"#7 could not be assigned"* ]]
  grep -q '^issue edit 7 --add-assignee ada$' "${FAKE_DIR}/gh.calls"
  grep -q '^issue edit 9 --add-assignee ada$' "${FAKE_DIR}/gh.calls"
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
  desk-world
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
  desk-world
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

@test "kit work refuses when the desk has the issue's branch checked out, before it assigns, sweeps, or commits" {
  local head origin_head
  desk-world
  git fetch --quiet origin
  git switch --quiet 7-fix-the-thing
  printf 'mine\n' > notes.txt
  head=$(git rev-parse HEAD)
  origin_head=$(git -C "${repo}.origin.git" rev-parse 7-fix-the-thing)
  run "${kit_checkout}/bin/kit" work 7
  [ "$status" -eq 1 ]
  [ "$(git status --porcelain)" = " M notes.txt" ]
  [ -z "$(git stash list)" ]
  [[ "$output" == *"refusing to start #7: its branch 7-fix-the-thing is checked out on the desk"* ]]
  run ! grep --quiet 'add-assignee' "${FAKE_DIR}/gh.calls"
  [ "$(git rev-parse HEAD)" = "${head}" ]
  [ "$(git -C "${repo}.origin.git" rev-parse 7-fix-the-thing)" = "${origin_head}" ]
}

@test "kit work that stops after the sweep puts the desk's work back, staged work staged, and leaves no stash" {
  desk-world
  printf 'mine\n' > notes.txt && git add notes.txt
  printf 'scratch\n' > scratch.txt
  printf 'staged\n' > staged.txt && git add staged.txt
  # a leftover directory where the worktree goes, which is not a worktree, stops kit work after the sweep
  mkdir -p worktrees/7-fix-the-thing
  run "${kit_checkout}/bin/kit" work 7
  [ "$status" -eq 1 ]
  [[ "$output" == *"The swept work is back on main."* ]]
  [ "$(cat notes.txt)" = mine ]
  [ "$(cat scratch.txt)" = scratch ]
  [ "$(cat staged.txt)" = staged ]
  [ "$(git status --porcelain)" = "$(printf 'M  notes.txt\nA  staged.txt\n?? scratch.txt')" ]
  [ -z "$(git stash list)" ]
}

@test "a desk whose only uncommitted work is inside a submodule sweeps nothing, and the human's own stash stays" {
  git init --quiet --initial-branch=main "${BATS_TEST_TMPDIR}/lib"
  printf 'lib\n' > "${BATS_TEST_TMPDIR}/lib/l"
  git -C "${BATS_TEST_TMPDIR}/lib" add l
  git -C "${BATS_TEST_TMPDIR}/lib" commit --quiet --message lib
  git -c protocol.file.allow=always submodule --quiet add "${BATS_TEST_TMPDIR}/lib" lib
  desk-world
  # the human has an old, unrelated stash, and work inside the submodule
  printf 'half an idea\n' > idea.txt
  git add idea.txt
  git stash push --quiet --message 'an old idea'
  printf 'wip\n' >> lib/l
  run "${kit_checkout}/bin/kit" work 7
  [ "$status" -eq 0 ]
  [[ "$output" == *"Nothing to sweep"* ]]
  [ "$(git stash list --format=%gs)" = "On main: an old idea" ]
  [ ! -e worktrees/7-fix-the-thing/idea.txt ]
  [ "$(cat lib/l)" = "$(printf 'lib\nwip')" ]
}

@test "kit work pushes the branch to its own name on origin, whatever push.default and its upstream say" {
  local main
  desk-world
  # the branch was made by hand from origin/main, which git sets as its upstream, and the desk pushes by upstream
  git fetch --quiet origin
  git branch --quiet 7-fix-the-thing origin/main
  git config push.default upstream
  main=$(git -C "${repo}.origin.git" rev-parse main)
  run "${kit_checkout}/bin/kit" work 7
  [ "$status" -eq 0 ]
  [ "$(git -C "${repo}.origin.git" rev-parse main)" = "${main}" ]
  [ "$(git -C "${repo}.origin.git" log -1 --format=%s 7-fix-the-thing)" = "#7 start: Fix the thing" ]
  [ "$(git -C "${repo}.origin.git" rev-parse 7-fix-the-thing)" = "$(git -C worktrees/7-fix-the-thing rev-parse HEAD)" ]
}

@test "kit work's start commit keeps its subject under commit.cleanup=strip, and is signed when signing is required" {
  printf 'require_signing: true\n' > kit.yml
  desk-world
  git config --global commit.cleanup strip
  signer
  run "${kit_checkout}/bin/kit" work 7
  [ "$status" -eq 0 ]
  git -C worktrees/7-fix-the-thing cat-file commit HEAD | grep --quiet '^gpgsig '
  [ "$(git -C worktrees/7-fix-the-thing log -1 --format=%s)" = "#7 start: Fix the thing" ]
}

@test "with require_signing, a start commit a hook refuses is said to have failed, never to be one git could not sign" {
  printf 'require_signing: true\n' > kit.yml
  desk-world
  signer
  printf '#!/bin/sh\necho "the hook says no" >&2\nexit 1\n' > .git/hooks/pre-commit
  chmod +x .git/hooks/pre-commit
  run "${kit_checkout}/bin/kit" work 7
  [ "$status" -eq 1 ]
  [[ "$output" == *"the hook says no"* ]]
  [[ "$output" == *"the start commit failed; see above"* ]]
  [[ "$output" != *"could not sign"* ]]
  [ "$(git -C "${repo}.origin.git" rev-parse 7-fix-the-thing)" = "$(git -C "${repo}.origin.git" rev-parse main)" ]
}

@test "kit work refuses the desk's branch though a tag has its name, which makes git's short name of it ambiguous" {
  local head
  desk-world
  git fetch --quiet origin
  git switch --quiet 7-fix-the-thing
  git tag 7-fix-the-thing
  head=$(git rev-parse HEAD)
  run "${kit_checkout}/bin/kit" work 7
  [ "$status" -eq 1 ]
  [[ "$output" == *"refusing to start #7: its branch 7-fix-the-thing is checked out on the desk"* ]]
  [ "$(git symbolic-ref HEAD)" = refs/heads/7-fix-the-thing ]
  [ "$(git rev-parse HEAD)" = "${head}" ]
  [ ! -e worktrees/7-fix-the-thing ]
}

@test "a stash another session pushes on top of the sweep's is not taken for it: nothing is swept, and both stay" {
  local mine
  desk-world
  printf 'mine\n' > notes.txt
  # git stands in for the real one, and pushes another session's stash right after the sweep's
  run kit-bash "use issue
    git() {
      local status=0
      command git \"\$@\" || status=\$?
      if [[ \"\$*\" == *'stash push'*'kit work'* ]]
      then
        printf 'theirs\n' > theirs.txt
        command git stash push --quiet --include-untracked --message theirs
      fi
      return \${status}
    }
    issue:sweep-push '${PWD}' 'kit work #7 → 7-fix-the-thing'"
  [ "$status" -eq 0 ]
  [[ "$output" == *"is not the one kit work pushed, so nothing is swept"* ]]
  [[ "$output" != *"$(git rev-parse 'stash@{0}')"* ]]
  [ "$(git stash list --format=%gs)" = "$(printf 'On main: theirs\nOn main: kit work #7 → 7-fix-the-thing')" ]
  mine=$(git rev-parse 'stash@{1}')
  [ "$(git show "${mine}:notes.txt")" = mine ]
}

@test "a start commit git cannot sign stops kit work, which pushes nothing; a rerun with a signer finishes" {
  local origin_head
  printf 'require_signing: true\n' > kit.yml
  desk-world
  signer 2
  origin_head=$(git -C "${repo}.origin.git" rev-parse 7-fix-the-thing)
  run "${kit_checkout}/bin/kit" work 7
  [ "$status" -eq 1 ]
  [[ "$output" == *"the start commit is refused: require_signing is true and git could not sign it; see above"* ]]
  [ "$(git -C "${repo}.origin.git" rev-parse 7-fix-the-thing)" = "${origin_head}" ]
  signer
  run "${kit_checkout}/bin/kit" work 7
  [ "$status" -eq 0 ]
  [ "$(git -C worktrees/7-fix-the-thing log -1 --format=%s)" = "#7 start: Fix the thing" ]
  git -C worktrees/7-fix-the-thing cat-file commit HEAD | grep --quiet '^gpgsig '
  [ "$(git -C "${repo}.origin.git" rev-parse 7-fix-the-thing)" = "$(git -C worktrees/7-fix-the-thing rev-parse HEAD)" ]
}

# issue-call <function> <argument>… runs a function of the issue module here, with the forge's reads stubbed: no open
# change names a branch, and the issue is no forge's, so kit names its branch; a draft opened is change 5.
issue-call() {
  bash -c 'source "${kit_root}/modules/bash/1st.sh" && use issue
    forge:branch-for-issue() { :; }
    forge:pr-for-branch() { :; }
    forge:pr-create() { printf "5\n"; }
    tracker:is-forge() { return 1; }
    tracker:key() { printf "7\n"; }
    "$@"' _ "$@"
}

@test "a branch kit work names is held to policy.branch: one it refuses is never made, and the pattern is named" {
  git commit --quiet --allow-empty --message init
  world
  # the issue has no branch yet, so kit names one
  git push --quiet origin --delete 7-fix-the-thing
  printf "policy:\n  branch: '^feature/'\n" > kit.yml
  run issue-call issue:branch '#7' main 'Fix the thing'
  [ "$status" -eq 1 ]
  [[ "$output" == *"refusing to start #7: the branch kit work names, branch '7-fix-the-thing' does not match"* ]]
  [[ "$output" == *"policy.branch: ^feature/; set policy.branch in kit.yml to allow <key>-<slug>"* ]]
  [ -z "$(git ls-remote --heads origin '7-*')" ]
  # a title with nothing for the slug still names a branch the default allows
  rm kit.yml
  run issue-call issue:branch '#7' main '!!!'
  [ "$status" -eq 0 ]
  [ "${lines[-1]}" = 7-issue ]
  [ -n "$(git ls-remote --heads origin 7-issue)" ]
}

@test "kit work refuses a worktree policy.worktree refuses before it assigns, and warns of a branch that exists" {
  desk-world
  printf "policy:\n  worktree: '^wt/'\n  branch: '^x'\n" >> kit.yml
  run "${kit_checkout}/bin/kit" work 7
  [ "$status" -eq 1 ]
  [[ "$output" == *"branch '7-fix-the-thing' does not match policy.branch: ^x; kit policy refuses it in CI"* ]]
  [[ "$output" == *"refusing to check 7-fix-the-thing out: the worktree kit work makes, worktree"* ]]
  [[ "$output" == *"'worktrees/7-fix-the-thing' does not match policy.worktree: ^wt/; set policy.worktree"* ]]
  [ ! -e worktrees/7-fix-the-thing ]
  run ! grep --quiet 'add-assignee' "${FAKE_DIR}/gh.calls"
}

@test "a draft whose title policy.change_title refuses is opened, with a warning that names the pattern" {
  git commit --quiet --allow-empty --message init
  world
  printf "policy:\n  change_title:\n    pattern: '^[a-z]+: '\n    remedy: start with a type\n" > kit.yml
  run issue-call issue:pull-request 7-fix-the-thing main 'Fix the thing'
  [ "$status" -eq 0 ]
  [[ "$output" == *"change title 'Fix the thing' does not match policy.change_title: ^[a-z]+:  (start with a type);"* ]]
  [[ "$output" == *"retitle it before it is ready"* ]]
  [ "${lines[-1]}" = 5 ]
}

# ----------------------------------------------------------------------------------------------------------------------
