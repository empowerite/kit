#!/usr/bin/env bats
# issue.bats - kit issue through a fake of the forge's tool: the worktree it makes is hydrated before the start commit
# ----------------------------------------------------------------------------------------------------------------------
#
# The world is forge-conformance.bats's: issue 7 "Fix the thing", assigned to bob, with the branch 7-fix-the-thing
# already on origin, as the forge makes it, and change 3 open on it. origin is a bare repository beside the test's,
# so git's part is real; gh is tests/fakes/fake-tool, so the forge's part is the provider's folding of fixtures.

bats_require_minimum_version 1.5.0

setup() {
  export NO_COLOR=1
  kit_checkout=$(cd "${BATS_TEST_DIRNAME}/../kit" && pwd)
  export KIT_HOME="${BATS_TEST_TMPDIR}/home" XDG_CACHE_HOME="${BATS_TEST_TMPDIR}/cache"
  export FAKE_DIR="${BATS_TEST_TMPDIR}/fake"
  mkdir -p "${KIT_HOME}" "${XDG_CACHE_HOME}" "${FAKE_DIR}/bin"
  ln -s "${BATS_TEST_DIRNAME}/fakes/fake-tool" "${FAKE_DIR}/bin/gh"
  export PATH="${FAKE_DIR}/bin:${PATH}"
  unset GITHUB_EVENT_PATH CI_MERGE_REQUEST_IID
  repo=$(mktemp -d)
  cd "${repo}"
  git init --quiet --initial-branch=main
  git config user.name t
  git config user.email t@example.com
  git config commit.gpgsign false
}

teardown() {
  rm -rf "${repo}" "${repo}.origin.git"
}

# fixture <name> writes stdin to a response file the routes name.
fixture() { cat > "${FAKE_DIR}/$1"; }

# route <glob> <response> adds a gh route, the more particular before the more general.
route() { printf '%s\t%s\n' "$1" "$2" >> "${FAKE_DIR}/gh.routes"; }

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
  printf 'forge: github\n' >> .kit/settings.yml
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

@test "kit issue with nothing is refused: a reference or a title, and no prompt remains" {
  run "${kit_checkout}/bin/kit" issue </dev/null
  [ "$status" -eq 2 ]
  [[ "$output" == *"kit issue needs <reference> or --title; run: kit help issue"* ]]
  [ "$(grep -c 'read -r -p' "${kit_checkout}/modules/bash/issue.sh")" -eq 0 ]
  run "${kit_checkout}/bin/kit" issue 7 --title 'Fix the thing'
  [ "$status" -eq 2 ]
  [[ "$output" == *"<reference> cannot be combined with --title; run: kit help issue"* ]]
}

@test "kit issue --title creates the issue and starts it in one step" {
  "${kit_checkout}/bin/kit" install >/dev/null 2>&1
  printf 'forge: github\n' >> .kit/settings.yml
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

# ----------------------------------------------------------------------------------------------------------------------
