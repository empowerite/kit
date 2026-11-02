#!/usr/bin/env bats
# commit.bats - kit commit and kit submit: the checks a commit's rule requires, then git, in a worktree only
# ----------------------------------------------------------------------------------------------------------------------
#
# The world is a consumer: origin is a bare repository beside the desk, the desk has the kit installed and its start
# commit on main, and the work happens in worktrees/7-fix-the-thing, the branch kit issue would make for issue 7.
# No forge is involved: the policy's defaults and the branch name are what judge a subject.

bats_require_minimum_version 1.5.0

load helper

setup() {
  work=$(mktemp -d)
  cd "${work}"
  git init --quiet --bare --initial-branch=main origin.git
  git clone --quiet origin.git desk 2>/dev/null
  cd desk
  "${kit_checkout}/bin/kit" install >/dev/null 2>&1
  printf 'notes\n' > notes.txt
  git add --all && git commit --quiet --message '#1 start: the first' && git push --quiet origin main
  git remote set-head origin main
  git worktree add --quiet -b 7-fix-the-thing worktrees/7-fix-the-thing main
  cd worktrees/7-fix-the-thing
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

# forge-world routes a fake gh for change 3, open on this branch with the body "Closes #7", and issue 9 with the
# assignees given, if any; ada is the caller. Every gh call lands in ${FAKE_DIR}/gh.calls.
forge-world() {
  fake-tools gh
  printf 'forge: github\n' >> kit.yml
  printf '{"login": "ada"}\n' > "${FAKE_DIR}/user.json"
  printf '[{"number": 3}]\n' > "${FAKE_DIR}/pr-open.json"
  printf '{"body": "Closes #7"}\n' > "${FAKE_DIR}/pr.json"
  jq --null-input --compact-output '{assignees: [$ARGS.positional[] | {login: .}]}' --args "$@" \
    > "${FAKE_DIR}/issue.json"
  printf '%s\t%s\n' \
    'api user*' user.json \
    'pr list --head 7-fix-the-thing --state open*' pr-open.json \
    'pr view 3 --json body*' pr.json \
    'pr edit 3 --body *' '!0' \
    'issue view 9 --json assignees*' issue.json \
    'issue edit 9 --add-assignee ada' '!0' > "${FAKE_DIR}/gh.routes"
}

@test "kit commit for another issue closes it in the branch's change, and assigns the caller when no one is" {
  forge-world
  printf 'x\n' > x && git add x
  run kit commit --message '#9 x: a file' x
  [ "$status" -eq 0 ]
  grep --quiet --line-regexp 'Closes #9' "${FAKE_DIR}/gh.calls"
  grep --quiet --line-regexp 'issue edit 9 --add-assignee ada' "${FAKE_DIR}/gh.calls"
}

@test "kit commit for another issue leaves its assignees, and a body that closes it already, as they are" {
  forge-world bob
  printf '{"body": "Closes #7\\nfixes #9"}\n' > "${FAKE_DIR}/pr.json"
  printf 'x\n' > x && git add x
  run kit commit --message '#9 x: a file' x
  [ "$status" -eq 0 ]
  ! grep --quiet '^pr edit' "${FAKE_DIR}/gh.calls"
  ! grep --quiet '^issue edit' "${FAKE_DIR}/gh.calls"
}

@test "kit commit for the branch's own issue asks the forge nothing" {
  forge-world
  printf 'x\n' > x && git add x
  run kit commit --message '#7 x: a file' x
  [ "$status" -eq 0 ]
  printf 'y\n' > y && git add y
  run kit commit --message 'y: no reference' y
  [ "$status" -eq 0 ]
  [ ! -e "${FAKE_DIR}/gh.calls" ]
}

@test "kit commit for another issue on a branch with no open change keeps the commit and says the issue is not closed" {
  forge-world
  printf '[]\n' > "${FAKE_DIR}/pr-open.json"
  printf 'x\n' > x && git add x
  run kit commit --message '#9 x: a file' x
  [ "$status" -eq 0 ]
  [[ "$output" == *"committed "* && "$output" == *"#9"*"no open change"* ]]
}

# opa-rule <rego> adds the policy/opa package and writes <rego> as a rule of package kit.policy beside its kit.rego,
# so the opa engine judges the commit. It skips the test when opa is not on PATH.
opa-rule() {
  command -v opa >/dev/null || skip "opa, the opa policy engine, is not on PATH"
  kit packages add policy/opa >/dev/null 2>&1
  printf 'package kit.policy\n\nimport rego.v1\n\n%s\n' "$1" > .kit/policy/test.rego
}

@test "kit commit takes a path git mv, git rm, or rm took away, commits it alone, and still refuses one git never knew" {
  printf 'old\n' > old.txt && git add old.txt && git commit --quiet --message '#7 files: an old file'
  git mv old.txt new.txt
  printf 'other\n' > other.txt && git add other.txt
  run kit commit --message '#7 files: old is new' old.txt new.txt
  [ "$status" -eq 0 ]
  [ "$(git show --no-renames --name-status --format= HEAD | sort)" = $'A\tnew.txt\nD\told.txt' ]
  [ "$(git diff --cached --name-only)" = other.txt ]
  git rm --quiet new.txt
  run kit commit --message '#7 files: new is gone' new.txt
  [ "$status" -eq 0 ]
  [ "$(git show --name-status --format= HEAD)" = $'D\tnew.txt' ]
  rm notes.txt
  run kit commit --message '#7 files: the notes are gone' notes.txt
  [ "$status" -eq 0 ]
  [ "$(git show --name-status --format= HEAD)" = $'D\tnotes.txt' ]
  [ "$(git diff --cached --name-only)" = other.txt ]
  run kit commit --message '#7 files: nothing' nowhere.txt
  [ "$status" -eq 1 ]
  [[ "$output" == *"nowhere.txt is not known to git; run: git add 'nowhere.txt'"* ]]
}

@test "the tree kit commit judges is HEAD and the named paths, never another session's staging" {
  opa-rule 'deny contains "theirs.md is in the tree" if input.tree["theirs.md"]

deny contains "mine.md is not in the tree" if not input.tree["mine.md"]'
  printf 'mine\n' > mine.md && printf 'theirs\n' > theirs.md && git add mine.md theirs.md
  run kit commit --message '#1 a: b' mine.md
  [ "$status" -eq 0 ]
  [ "$(git show --name-only --format= HEAD)" = mine.md ]
  [ "$(git status --porcelain -- theirs.md)" = "A  theirs.md" ]
}

@test "the first commit is judged against the named paths alone" {
  opa-rule 'deny contains "README.md is missing" if not input.tree["README.md"]

deny contains sprintf("%s is in the tree", [path]) if {
  some path, _ in input.tree
  path != "README.md"
}'
  # the orphan branch starts with no HEAD, and its switch wants no change to a tracked file
  git commit --quiet --message '#7 kit: the opa engine' kit.yml
  git switch --quiet --orphan 7-the-first
  printf '# Hello\n' > README.md && printf 'other\n' > other.md && git add README.md other.md
  run kit commit --message '#7 readme: the first' README.md
  [ "$status" -eq 0 ]
  [ "$(git show --name-only --format= HEAD)" = README.md ]
  [ "$(git status --porcelain -- other.md)" = "A  other.md" ]
}

@test "a policy the engine cannot evaluate refuses the commit, naming the engine's error" {
  opa-rule 'deny contains "never" if {'
  printf 'x\n' > x && git add x
  run kit commit --message '#7 x: a file' x
  [ "$status" -eq 1 ]
  [[ "$output" == *"opa could not evaluate .kit/policy"* ]]
  [ "$(git log --oneline | wc -l | tr -d ' ')" = 1 ]
}

@test "a named path deleted from the disk is committed as a deletion" {
  # notes.txt is in HEAD from the start commit
  rm notes.txt
  run kit commit --message '#1 a: b' notes.txt
  [ "$status" -eq 0 ]
  [ -z "$(git ls-tree HEAD -- notes.txt)" ]
}

@test "a file rule refuses the commit at the commit moment" {
  opa-rule 'deny contains sprintf("%s is not executable", [path]) if {
  input.moment == "commit"
  some path, f in input.tree
  startswith(path, "bin/")
  not f.executable
}'
  mkdir bin && printf 'x\n' > bin/x && git add bin/x
  run kit commit --message '#7 bin: x'
  [ "$status" -eq 1 ]
  [[ "$output" == *"bin/x is not executable"* ]]
  [ "$(git log --oneline | wc -l | tr -d ' ')" = 1 ]
}

# ----------------------------------------------------------------------------------------------------------------------
