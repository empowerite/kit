#!/usr/bin/env bats
# commit.bats - kit commit and kit submit: the checks a commit's rule requires, then git, in a worktree only
# ----------------------------------------------------------------------------------------------------------------------
# The world is a consumer: origin is a bare repository beside the desk, the desk has the kit installed and its start
# commit on main, and the work happens in worktrees/7-fix-the-thing, the branch kit work would make for issue 7.
# No forge is involved: the policy's defaults and the branch name are what judge a subject.

bats_require_minimum_version 1.5.0

load helper

# desk-world makes what every test starts from, once for the file: origin, a bare repository beside the desk, and the
# desk with the kit installed and its start commit on main, pushed.
desk-world() {
  git init --quiet --bare --initial-branch=main origin.git
  git clone --quiet origin.git desk 2>/dev/null
  cd desk || return 1
  "${kit_checkout}/bin/kit" install
  printf 'notes\n' > notes.txt
  git add --all && git commit --quiet --message '#1 start: the first' && git push --quiet origin main
  git remote set-head origin main
}

setup_file() {
  world-make desk desk-world
}

setup() {
  work=$(mktemp -d)
  world-copy desk "${work}"
  cd "${work}/desk"
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

@test "kit submit pushes what kit commit committed, setting the upstream the first time; with nothing new it says so" {
  printf 'x\n' > x && git add x
  kit commit --message 'x: a file' >/dev/null 2>&1
  run kit submit
  [ "$status" -eq 0 ]
  [[ "$output" != *"committed "* ]]
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
  kit commit --message '#7 x: mine' >/dev/null 2>&1
  run kit submit
  [ "$status" -eq 1 ]
  [[ "$output" == *"run: kit rebase"* ]]
  [ "$(git log -1 --format=%s)" = "#7 x: mine" ]
}

@test "kit submit pushes the branch to its own name on origin, whatever push.default and its upstream say" {
  git branch --quiet --set-upstream-to=origin/main
  git config push.default upstream
  main=$(git -C "${work}/origin.git" rev-parse main)
  printf 'x\n' > x && git add x
  kit commit --message 'x: a file' >/dev/null 2>&1
  run kit submit
  [ "$status" -eq 0 ]
  [ "$(git rev-parse --abbrev-ref '@{upstream}')" = origin/7-fix-the-thing ]
  [ "$(git -C "${work}/origin.git" rev-parse 7-fix-the-thing)" = "$(git rev-parse HEAD)" ]
  printf 'y\n' > y && git add y
  kit commit --message 'y: a file' >/dev/null 2>&1
  run kit submit
  [ "$status" -eq 0 ]
  [[ "$output" == *"pushed 7-fix-the-thing: 1 commit(s)"* ]]
  [ "$(git -C "${work}/origin.git" rev-parse 7-fix-the-thing)" = "$(git rev-parse HEAD)" ]
  [ "$(git -C "${work}/origin.git" rev-parse main)" = "${main}" ]
}

@test "kit submit pushes a branch whose remote-tracking ref is gone, and never says it is at origin" {
  printf 'x\n' > x && git add x
  kit commit --message 'x: a file' >/dev/null 2>&1
  run kit submit
  [ "$status" -eq 0 ]
  # the forge deletes the branch, and a fetch with --prune takes origin/7-fix-the-thing away
  git -C "${work}/origin.git" branch --quiet --delete --force 7-fix-the-thing
  git fetch --quiet --prune origin
  run ! git rev-parse --quiet --verify refs/remotes/origin/7-fix-the-thing
  printf 'y\n' > y && git add y
  kit commit --message 'y: a file' >/dev/null 2>&1
  run kit submit
  [ "$status" -eq 0 ]
  [[ "$output" != *"nothing to submit"* ]]
  [ "$(git -C "${work}/origin.git" rev-parse 7-fix-the-thing)" = "$(git rev-parse HEAD)" ]
  [ "$(git rev-parse refs/remotes/origin/7-fix-the-thing)" = "$(git rev-parse HEAD)" ]
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
  run ! grep --quiet '^pr edit' "${FAKE_DIR}/gh.calls"
  run ! grep --quiet '^issue edit' "${FAKE_DIR}/gh.calls"
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

@test "a body that cannot be read is never replaced: the commit stands, and kit says to add the line by hand" {
  forge-world
  sed -i.bak '/^pr view 3 /d' "${FAKE_DIR}/gh.routes"
  printf 'x\n' > x && git add x
  run kit commit --message '#9 x: a file' x
  [ "$status" -eq 0 ]
  [[ "$output" == *"could not be added to change 3's body"* ]]
  run ! grep --quiet '^pr edit' "${FAKE_DIR}/gh.calls"
}

@test "with a primary that names no remote, kit commit refuses, since the default branch could not be read" {
  local head
  head=$(git rev-parse HEAD)
  printf 'primary: upstream\n' >> kit.yml
  printf 'x\n' > x && git add x
  run kit commit --message '#9 x: a file' x
  [ "$status" -eq 1 ]
  [[ "$output" == *"is not a remote"* ]]
  [[ "$output" != *"✅ committed"* ]]
  [ "$(git rev-parse HEAD)" = "${head}" ]
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

@test "a staged deletion never reaches the checks as a named path: git rm commits, alone and beside an added file" {
  printf 'x\n' > x.txt && git add x.txt && git commit --quiet --message '#7 x: a file' -- x.txt
  git rm --quiet notes.txt
  printf 'y\n' > y.txt && git add y.txt
  run kit commit --message '#7 files: notes go, y comes'
  [ "$status" -eq 0 ]
  [[ "$output" != *"is not a file or a directory here"* ]]
  [ "$(git show --name-status --format= HEAD | sort)" = $'A\ty.txt\nD\tnotes.txt' ]
  git rm --quiet x.txt
  run kit commit --message '#7 x: gone'
  [ "$status" -eq 0 ]
  [ "$(git show --name-status --format= HEAD)" = $'D\tx.txt' ]
}

@test "a path staged and then taken off the disk is refused by name, and HEAD stays" {
  printf '#!/usr/bin/env bash\nset -o errexit -o nounset -o pipefail\necho $1\n' > bad.sh
  printf 'changed\n' > notes.txt
  git add bad.sh notes.txt
  rm bad.sh notes.txt
  head=$(git rev-parse HEAD)
  run kit commit --message '#7 bad: a script'
  [ "$status" -eq 1 ]
  for path in bad.sh notes.txt
  do
    said="'${path}' is staged but is not on the disk; run: git restore -- '${path}', or git rm --cached -- '${path}'"
    [[ "$output" == *"${said}"* ]]
  done
  [[ "$output" != *"committed "* ]]
  [ "$(git rev-parse HEAD)" = "${head}" ]
  [ "$(git diff --cached --name-only | sort | paste -s -d ' ' -)" = "bad.sh notes.txt" ]
}

@test "a named path deleted from the disk is committed as a deletion" {
  # notes.txt is in HEAD from the start commit
  rm notes.txt
  run kit commit --message '#1 a: b' notes.txt
  [ "$status" -eq 0 ]
  [ -z "$(git ls-tree HEAD -- notes.txt)" ]
}

@test "a named folder the disk now holds as a file is committed as that file" {
  mkdir d && printf 'c\n' > d/c.md && git add d && git commit --quiet --message '#7 d: a folder' -- d
  rm -r d && printf 'd\n' > d && git add d
  run kit commit --message '#1 a: b' d
  [ "$status" -eq 0 ]
  [ "$(git cat-file -t HEAD:d)" = blob ]
  [ -z "$(git ls-tree -r HEAD -- d/c.md)" ]
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

@test "kit commit prints a warning and commits" {
  opa-rule 'warn contains "x is new" if input.tree.x'
  printf 'x\n' > x && git add x
  run kit commit --message '#7 x: a file' x
  [ "$status" -eq 0 ]
  [[ "$output" == *"⚠️ x is new"* ]]
  [ "$(git log -1 --format=%s)" = "#7 x: a file" ]
}

# peer-script writes ${work}/peer. peer <message> <path> <content> is another session in this worktree: it writes
# <path> on the disk and commits it on the branch from an index of its own, so the shared index never sees the commit.
# It prints the commit.
peer-script() {
  cat > "${work}/peer" <<'EOF'
#!/usr/bin/env bash
set -o errexit -o nounset
own=$(mktemp -d)
printf '%s\n' "$3" > "$2"
GIT_INDEX_FILE=${own}/index git read-tree HEAD
GIT_INDEX_FILE=${own}/index git update-index --add -- "$2"
commit=$(git commit-tree "$(GIT_INDEX_FILE=${own}/index git write-tree)" -p HEAD -m "$1")
git update-ref HEAD "${commit}"
rm -rf "${own}"
printf '%s\n' "${commit}"
EOF
  chmod +x "${work}/peer"
}

@test "an index left behind by another session's commit is refused, naming the commit, and HEAD stays" {
  local index=${work}/private-index peer
  peer-script
  # a changed path: the index holds notes.txt as it was before the peer changed it
  GIT_INDEX_FILE=${index} git read-tree HEAD
  printf 'mine\n' > mine.txt && GIT_INDEX_FILE=${index} git update-index --add mine.txt
  peer=$("${work}/peer" '#7 notes: theirs' notes.txt theirs)
  run env GIT_INDEX_FILE="${index}" "${kit_checkout}/bin/kit" commit --message '#7 mine: a file'
  [ "$status" -eq 1 ]
  [[ "$output" == *"notes.txt"*"${peer:0:7}"* ]]
  [ "$(git rev-parse HEAD)" = "${peer}" ]
  # an added path: the index lacks peer.txt, which the peer added
  GIT_INDEX_FILE=${index} git read-tree HEAD
  GIT_INDEX_FILE=${index} git update-index --add mine.txt
  peer=$("${work}/peer" '#7 peer: a file' peer.txt peer)
  run env GIT_INDEX_FILE="${index}" "${kit_checkout}/bin/kit" commit --message '#7 mine: a file'
  [ "$status" -eq 1 ]
  [[ "$output" == *"peer.txt"*"${peer:0:7}"* ]]
  [ "$(git rev-parse HEAD)" = "${peer}" ]
}

@test "a commit that lands while kit commit checks is refused, naming it, and HEAD stays the other session's" {
  opa-rule 'deny contains "never" if input.never'
  peer-script
  # the engine runs during the checks: the other session commits as it is called the first time
  mkdir "${work}/bin"
  printf '#!/usr/bin/env bash\n[[ -e %q ]] || { touch %q; %q "#7 notes: theirs" notes.txt theirs >/dev/null; }\nexec %q "$@"\n' \
    "${work}/landed" "${work}/landed" "${work}/peer" "$(command -v opa)" > "${work}/bin/opa"
  chmod +x "${work}/bin/opa"
  printf 'mine\n' > mine.txt && git add mine.txt
  PATH="${work}/bin:${PATH}" run kit commit --message '#7 mine: a file'
  [ -e "${work}/landed" ]
  [ "$status" -eq 1 ]
  [[ "$output" == *"$(git rev-parse --short HEAD)"* ]]
  [ "$(git log -1 --format=%s)" = "#7 notes: theirs" ]
  [ "$(git show HEAD:notes.txt)" = theirs ]
  [ -z "$(git ls-tree HEAD -- mine.txt)" ]
}

@test "a commit that lands while kit commit checks named paths is refused too, and HEAD stays the other session's" {
  opa-rule 'deny contains "never" if input.never'
  peer-script
  mkdir "${work}/bin"
  printf '#!/usr/bin/env bash\n[[ -e %q ]] || { touch %q; %q "#7 notes: theirs" notes.txt theirs >/dev/null; }\nexec %q "$@"\n' \
    "${work}/landed" "${work}/landed" "${work}/peer" "$(command -v opa)" > "${work}/bin/opa"
  chmod +x "${work}/bin/opa"
  printf 'mine\n' > mine.txt && git add mine.txt
  PATH="${work}/bin:${PATH}" run kit commit --message '#7 mine: a file' mine.txt
  [ -e "${work}/landed" ]
  [ "$status" -eq 1 ]
  [[ "$output" == *"$(git rev-parse --short HEAD)"* ]]
  [ "$(git log -1 --format=%s)" = "#7 notes: theirs" ]
  [ -z "$(git ls-tree HEAD -- mine.txt)" ]
}

@test "the shared index commits as before: after a commit by name, a change put back on purpose, and a file removed" {
  printf 'mine\n' > mine.txt && git add mine.txt
  printf 'theirs\n' > notes.txt && git commit --quiet --message '#7 notes: theirs' -- notes.txt
  run kit commit --message '#7 mine: a file'
  [ "$status" -eq 0 ]
  [ "$(git show --name-only --format= HEAD)" = mine.txt ]
  [ "$(git show HEAD:notes.txt)" = theirs ]
  printf 'notes\n' > notes.txt && git add notes.txt
  run kit commit --message '#7 notes: back as they were'
  [ "$status" -eq 0 ]
  [ "$(git show HEAD:notes.txt)" = notes ]
  rm mine.txt
  run kit commit --all --message '#7 mine: gone'
  [ "$status" -eq 0 ]
  [ "$(git show --name-status --format= HEAD)" = $'D\tmine.txt' ]
}

@test "kit commit refuses a file under a directory kit placed, such as the logs and the cache" {
  kit hydrate >/dev/null 2>&1
  mkdir -p .kit/logs .kit/cache
  printf 'a line\n' > .kit/logs/kit-test.log
  printf 'a probe\n' > .kit/cache/probe
  git add --force .kit/logs/kit-test.log .kit/cache/probe
  run kit commit --message '#7 logs: the day'
  [ "$status" -eq 1 ]
  [[ "$output" == *".kit/logs/kit-test.log is kit's"* ]]
  [[ "$output" == *".kit/cache/probe is kit's"* ]]
  [ "$(git log --oneline | wc -l | tr -d ' ')" = 1 ]
}

@test "kit commit refuses a directory named by path when it holds a file kit placed" {
  kit hydrate >/dev/null 2>&1
  git add --force .kit/1st.md
  run kit commit --message '#7 kit: the directory' -- .kit
  [ "$status" -eq 1 ]
  [[ "$output" == *".kit/1st.md is kit's"* ]]
  [ "$(git log --oneline | wc -l | tr -d ' ')" = 1 ]
}

@test "a named path is a path, never a pattern: a glob matches no file, and a name with [ ] is that file" {
  printf 'a\n' > a.md && printf 'b\n' > b.md && git add a.md b.md
  run kit commit --message '#7 md: a glob' -- '*.md'
  [ "$status" -eq 1 ]
  [[ "$output" == *"*.md is not known to git"* ]]
  [ "$(git log --oneline | wc -l | tr -d ' ')" = 1 ]
  printf 'x\n' > x.txt && printf 'bracket\n' > '[x].txt' && git add x.txt '[x].txt'
  run kit commit --message '#7 txt: the bracket file' -- '[x].txt'
  [ "$status" -eq 0 ]
  [ "$(git show --name-only --format= HEAD)" = '[x].txt' ]
}

@test "a named directory commits the files under it, each linted" {
  mkdir scripts
  printf '#!/usr/bin/env bash\nset -o errexit -o nounset -o pipefail\necho $1\n' > scripts/bad.sh
  git add scripts
  run kit commit --message '#7 scripts: a bad one' -- scripts
  [ "$status" -eq 1 ]
  [[ "$output" == *"linter"* ]]
  [ "$(git log --oneline | wc -l | tr -d ' ')" = 1 ]
}

@test "the policy is the committed tree's: an unstaged kit.yml that weakens it changes nothing about the commit" {
  printf 'policy:\n  commit_subject: "^#[0-9]+ strict: "\n' >> kit.yml
  git add kit.yml && git commit --quiet --message '#7 policy: strict' -- kit.yml
  git show HEAD:kit.yml | sed '/^policy:/,$d' > kit.yml
  printf 'x\n' > x && git add x
  run kit commit --message '#7 x: loose'
  [ "$status" -eq 1 ]
  [[ "$output" == *"#7 x: loose"* ]]
  run kit commit --message '#7 x: loose' -- x
  [ "$status" -eq 1 ]
  [ "$(git log --oneline | wc -l | tr -d ' ')" = 2 ]
}

@test "the engine is the committed tree's: an unstaged policy_engine cannot swap the Rego out" {
  opa-rule 'deny contains "never" if true'
  printf 'policy_engine: patterns\n' >> kit.yml
  printf 'x\n' > x && git add x
  run kit commit --message '#7 x: a file' -- x
  [ "$status" -eq 1 ]
  [[ "$output" == *"never"* ]]
  [ "$(git log --oneline | wc -l | tr -d ' ')" = 1 ]
}

@test "a path staged by another session while kit commit checks is refused, and nothing is committed" {
  opa-rule 'deny contains "never" if input.never'
  mkdir "${work}/bin"
  printf '#!/usr/bin/env bash\n[[ -e %q ]] || { touch %q; printf late > late.txt; git add late.txt; }\nexec %q "$@"\n' \
    "${work}/staged" "${work}/staged" "$(command -v opa)" > "${work}/bin/opa"
  chmod +x "${work}/bin/opa"
  printf 'mine\n' > mine.txt && git add mine.txt
  PATH="${work}/bin:${PATH}" run kit commit --message '#7 mine: a file'
  [ -e "${work}/staged" ]
  [ "$status" -eq 1 ]
  [[ "$output" == *"the index changed while kit commit checked"* ]]
  [ "$(git log --oneline | wc -l | tr -d ' ')" = 1 ]
}

@test "kit commit stopped while it checks leaves no scratch behind, and keeps its scratch under TMPDIR" {
  opa-rule 'deny contains "never" if input.never'
  mkdir "${work}/bin" "${work}/tmp"
  # the engine's input file is the seventh word; the stand-in notes it, then stalls, so the commit is stopped mid-judge
  printf '#!/usr/bin/env bash\n[[ "$1" != eval ]] || { printf "%%s\\n" "$7" > %q; sleep 2; }\nexec %q "$@"\n' \
    "${work}/judging" "$(command -v opa)" > "${work}/bin/opa"
  chmod +x "${work}/bin/opa"
  printf 'mine\n' > mine.txt && git add mine.txt
  PATH="${work}/bin:${PATH}" TMPDIR="${work}/tmp" "${kit_checkout}/bin/kit-commit" --message '#7 mine: a file' \
    >/dev/null 2>&1 &
  local pid=$! tries=0 input
  until [[ -s "${work}/judging" ]] || (( tries++ > 300 ))
  do
    sleep 0.1
  done
  input=$(cat "${work}/judging")
  [[ "${input}" == "${work}/tmp/"* ]]
  kill -TERM "${pid}"
  wait "${pid}" || true
  [ ! -e "${input}" ]
  [ -z "$(find "${work}/tmp" -mindepth 1 -maxdepth 1)" ]
}

@test "kit commit refuses --all beside named paths, as git does" {
  printf 'x\n' > x && git add x
  run kit commit --all --message '#7 x: a file' -- x
  [ "$status" -eq 2 ]
  [[ "$output" == *"--all"* ]]
  [ "$(git log --oneline | wc -l | tr -d ' ')" = 1 ]
}

@test "a named path that matches HEAD is nothing to commit, said plainly" {
  run kit commit --message '#7 notes: the same' -- notes.txt
  [ "$status" -eq 1 ]
  [[ "$output" == *"nothing to commit: notes.txt is as HEAD has it"* ]]
}

@test "kit submit commits nothing: what is staged stays staged and out of the push, and every word given is refused" {
  printf 'x\n' > x && git add x
  kit commit --message '#7 x: a file' >/dev/null 2>&1
  head=$(git rev-parse HEAD)
  printf 'y\n' > y && git add y
  run kit submit
  [ "$status" -eq 0 ]
  [[ "$output" == *"1 staged path(s) stay staged, and are no part of the push; kit commit commits them"* ]]
  [ "$(git rev-parse HEAD)" = "${head}" ]
  [ "$(git rev-parse origin/7-fix-the-thing)" = "${head}" ]
  [ "$(git diff --cached --name-only)" = y ]
  for words in "--all" "--message=#7 y: a file" "--file=m" "y"
  do
    run kit submit "${words}"
    [ "$status" -eq 2 ]
    [[ "$output" == *"kit submit takes no arguments: it pushes, and commits nothing; commit first: kit commit"* ]]
  done
  [ "$(git rev-parse HEAD)" = "${head}" ]
  [ "$(git diff --cached --name-only)" = y ]
}

@test "kit commit refuses a stale region, and a detached HEAD" {
  printf '# notes\n\n<!-- kit:generate echo fresh -->\nstale\n<!-- kit:end -->\n' > regions.md
  git add regions.md
  run kit commit --message '#7 regions: a stale one'
  [ "$status" -eq 1 ]
  [[ "$output" == *"a region is stale; run: kit generate"* ]]
  git reset --quiet -- regions.md
  git switch --quiet --detach
  printf 'x\n' > x && git add x
  run kit commit --message '#7 x: a file'
  [ "$status" -eq 1 ]
  [[ "$output" == *"no branch is checked out here"* ]]
}

@test "kit commit warns when commits are not signed, and commits" {
  git config commit.gpgsign false
  printf 'x\n' > x && git add x
  run kit commit --message '#7 x: a file'
  [ "$status" -eq 0 ]
  [[ "$output" == *"commits here are not signed"* ]]
}

@test "with require_signing, kit commit signs or refuses; commit.cleanup=strip cannot drop the subject it judged" {
  printf 'require_signing: true\n' >> kit.yml
  git config commit.cleanup strip
  printf 'x\n' > x && git add x
  head=$(git rev-parse HEAD)
  run kit commit --message 'x: a file'
  [ "$status" -eq 1 ]
  [[ "$output" == *"git config user.signingkey"* ]]
  [ "$(git rev-parse HEAD)" = "${head}" ]
  [ "$(git diff --cached --name-only)" = x ]
  signer
  run kit commit --message 'x: a file' --message 'WIP'
  [ "$status" -eq 0 ]
  [[ "$output" != *"commits here are not signed"* ]]
  [ "$(git log -1 --format=%s)" = "#7 x: a file" ]
  git cat-file commit HEAD | grep --quiet '^gpgsig '
  signer 2
  head=$(git rev-parse HEAD)
  printf 'y\n' > y && git add y
  run kit commit --message 'y: a file'
  [ "$status" -eq 1 ]
  [[ "$output" == *"require_signing is true and git could not sign it"* ]]
  [ "$(git rev-parse HEAD)" = "${head}" ]
  [ "$(git diff --cached --name-only)" = y ]
}

@test "with require_signing, a commit a hook refuses is said to have failed, never to be one git could not sign" {
  local head
  printf 'require_signing: true\n' >> kit.yml
  signer
  printf '#!/bin/sh\necho "the hook says no" >&2\nexit 1\n' > "$(git rev-parse --git-common-dir)/hooks/pre-commit"
  chmod +x "$(git rev-parse --git-common-dir)/hooks/pre-commit"
  head=$(git rev-parse HEAD)
  printf 'x\n' > x && git add x
  run kit commit --message 'x: a file'
  [ "$status" -eq 1 ]
  [[ "$output" == *"the hook says no"* ]]
  [[ "$output" == *"the commit failed; see above"* ]]
  [[ "$output" != *"could not sign"* ]]
  [ "$(git rev-parse HEAD)" = "${head}" ]
  [ "$(git diff --cached --name-only)" = x ]
}

# On Linux a tee to /dev/stderr truncated a regular file standing there, so this fails on Linux before the fix and
# passes on macOS (which shares the file offset) before and after it.
@test "kit commit leaves what was already on its standard error where it is, when that is a file" {
  printf '#!/bin/sh\necho "the hook says no" >&2\nexit 1\n' > "$(git rev-parse --git-common-dir)/hooks/pre-commit"
  chmod +x "$(git rev-parse --git-common-dir)/hooks/pre-commit"
  printf 'the first line\n' > "${BATS_TEST_TMPDIR}/err"
  printf 'x\n' > x && git add x
  status=0
  kit commit --message 'x: a file' 2>>"${BATS_TEST_TMPDIR}/err" >/dev/null || status=$?
  [ "$status" -eq 1 ]
  [ "$(head -n 1 "${BATS_TEST_TMPDIR}/err")" = 'the first line' ]
  grep --quiet --text 'no rule governs the 1 path(s); nothing was linted' "${BATS_TEST_TMPDIR}/err"
  grep --quiet --text 'the hook says no' "${BATS_TEST_TMPDIR}/err"
  tr -d '\0' < "${BATS_TEST_TMPDIR}/err" | cmp --silent - "${BATS_TEST_TMPDIR}/err"
}

@test "facts that cannot be gathered refuse the commit" {
  printf 'policy:\n  colour: blue\n' >> kit.yml
  git add kit.yml
  run kit commit --message '#7 kit: a broken policy'
  [ "$status" -eq 1 ]
  [[ "$output" == *"the commit is refused"* ]]
  [ "$(git log --oneline | wc -l | tr -d ' ')" = 1 ]
}

# ----------------------------------------------------------------------------------------------------------------------
