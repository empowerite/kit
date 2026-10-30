#!/usr/bin/env bats
# commit.bats - kit commit and kit submit: the checks a commit's rule requires, then git, in a worktree only
# ----------------------------------------------------------------------------------------------------------------------
#
# The world is a consumer: origin is a bare repository beside the desk, the desk has the kit installed and its start
# commit on main, and the work happens in worktrees/7-fix-the-thing, the branch kit issue would make for issue 7.
# No forge is involved: the policy's defaults and the branch name are what judge a subject.

bats_require_minimum_version 1.5.0

setup() {
  export NO_COLOR=1
  kit_checkout=$(cd "${BATS_TEST_DIRNAME}/../kit" && pwd)
  export KIT_HOME="${BATS_TEST_TMPDIR}/home" XDG_CACHE_HOME="${BATS_TEST_TMPDIR}/cache"
  mkdir -p "${KIT_HOME}" "${XDG_CACHE_HOME}"
  work=$(mktemp -d)
  cd "${work}"
  git init --quiet --bare --initial-branch=main origin.git
  git clone --quiet origin.git desk 2>/dev/null
  cd desk
  git config user.name t && git config user.email t@example.com && git config commit.gpgsign false
  "${kit_checkout}/bin/kit" install >/dev/null 2>&1
  printf 'notes\n' > notes.txt
  git add --all && git commit --quiet --message '#1 start: the first' && git push --quiet origin main
  git remote set-head origin main
  git worktree add --quiet -b 7-fix-the-thing worktrees/7-fix-the-thing main
  cd worktrees/7-fix-the-thing
}

teardown() {
  rm -rf "${work}"
}

kit() { "${kit_checkout}/bin/kit" "$@"; }

@test "kit commit refuses the desk, and the default branch in a worktree, before it looks at anything else" {
  cd "${work}/desk"
  printf 'x\n' > x && git add x
  run kit commit --message '#1 x: a file'
  [ "$status" -eq 1 ]
  [[ "$output" == *"the root checkout, the human's desk"* ]]
  git reset --quiet -- x
  git switch --quiet --create holding
  git worktree add --quiet worktrees/main-too main
  cd worktrees/main-too
  printf 'y\n' > y && git add y
  run kit commit --message '#1 y: a file'
  [ "$status" -eq 1 ]
  [[ "$output" == *"it is the default branch"* ]]
  [ "$(git log --oneline | wc -l | tr -d ' ')" = 1 ]
}

@test "kit commit refuses when nothing is staged, and when the message is missing" {
  run kit commit --message '#7 x: nothing'
  [ "$status" -eq 1 ]
  [[ "$output" == *"nothing is staged"* ]]
  printf 'x\n' > x && git add x
  run kit commit
  [ "$status" -eq 2 ]
  [[ "$output" == *"--message"* ]]
  [ "$(git log --oneline | wc -l | tr -d ' ')" = 1 ]
}

@test "kit commit refuses a staged path the kit placed under .kit/" {
  kit hydrate >/dev/null 2>&1
  git add --force .kit/1st.md
  run kit commit --message '#7 kit: a delivered file'
  [ "$status" -eq 1 ]
  [[ "$output" == *".kit/1st.md is kit's"* ]]
  [[ "$output" == *"git rm --cached"* ]]
}

@test "kit commit runs the linters over the staged paths and refuses a file that fails one, so no commit is made" {
  printf '#!/usr/bin/env bash\nset -o errexit -o nounset -o pipefail\necho $1\n' > bad.sh
  git add bad.sh
  run kit commit --message '#7 bad: a script'
  [ "$status" -eq 1 ]
  [[ "$output" == *"bad.sh"* ]]
  [[ "$output" == *"linter"* ]]
  [ "$(git log --oneline | wc -l | tr -d ' ')" = 1 ]
}

@test "kit commit supplies the issue's reference from the branch, judges the subject by the policy, and commits" {
  printf 'x\n' > x && git add x
  run kit commit --message 'x: a file' --message 'The body.'
  [ "$status" -eq 0 ]
  [[ "$output" == *"#7 supplied from the branch"* ]]
  [[ "$output" == *"committed "*"#7 x: a file"* ]]
  [ "$(git log -1 --format=%s)" = "#7 x: a file" ]
  [ "$(git log -1 --format=%b)" = "The body." ]
  [ -z "$(git status --porcelain)" ]
}

@test "kit commit refuses a subject the policy refuses, and names the pattern" {
  git switch --quiet --create no-issue
  printf 'x\n' > x && git add x
  run kit commit --message 'plain words'
  [ "$status" -eq 1 ]
  [[ "$output" == *"policy.commit_subject"* ]]
  [ "$(git log --oneline | wc -l | tr -d ' ')" = 1 ]
}

@test "kit commit --all takes every change to a tracked file, --file reads the message, and paths commit those alone" {
  printf 'more\n' >> notes.txt
  printf '#7 notes: more\n\nFrom a file.\n' > "${work}/message"
  run kit commit --all --file "${work}/message"
  [ "$status" -eq 0 ]
  [ "$(git log -1 --format=%s)" = "#7 notes: more" ]
  [ "$(git log -1 --format=%b)" = "From a file." ]
  [ -z "$(git status --porcelain)" ]
  printf 'a\n' > a && printf 'b\n' > b && git add a b
  run kit commit --message '#7 a: only a' a
  [ "$status" -eq 0 ]
  [ "$(git show --name-only --format= HEAD)" = a ]
  [ "$(git status --porcelain)" = "A  b" ]
}

@test "kit submit commits what is staged and pushes, setting the upstream the first time; with nothing new it says so" {
  printf 'x\n' > x && git add x
  run kit submit --message 'x: a file'
  [ "$status" -eq 0 ]
  [[ "$output" == *"committed "* ]]
  [[ "$output" == *"pushed 7-fix-the-thing"* ]]
  [ "$(git rev-parse --abbrev-ref '@{upstream}')" = origin/7-fix-the-thing ]
  [ "$(git rev-parse origin/7-fix-the-thing)" = "$(git rev-parse HEAD)" ]
  run kit submit
  [ "$status" -eq 0 ]
  [[ "$output" == *"nothing to submit"* ]]
}

@test "kit submit refuses the desk; when origin has moved it names kit rebase, never forcing" {
  cd "${work}/desk"
  run kit submit
  [ "$status" -eq 1 ]
  [[ "$output" == *"the root checkout, the human's desk"* ]]
  cd "${work}/desk/worktrees/7-fix-the-thing"
  git push --quiet --set-upstream origin 7-fix-the-thing
  git clone --quiet --branch 7-fix-the-thing "${work}/origin.git" "${work}/other" 2>/dev/null
  git -C "${work}/other" config user.name o && git -C "${work}/other" config user.email o@example.com
  git -C "${work}/other" commit --quiet --allow-empty --message '#7 other: theirs'
  git -C "${work}/other" push --quiet
  printf 'x\n' > x && git add x
  run kit submit --message '#7 x: mine'
  [ "$status" -eq 1 ]
  [[ "$output" == *"run: kit rebase"* ]]
  [ "$(git log -1 --format=%s)" = "#7 x: mine" ]
}

# ----------------------------------------------------------------------------------------------------------------------
