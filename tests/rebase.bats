#!/usr/bin/env bats
# rebase.bats - kit rebase refuses the desk and the default branch, and brings a worktree's branch up to date
# ----------------------------------------------------------------------------------------------------------------------

load helper

setup() {
  work=$(mktemp -d)
  cd "${work}"
  git init --quiet --bare --initial-branch=main origin.git
  git clone --quiet origin.git desk 2>/dev/null
  cd desk
  printf 'one\n' > a && git add a && git commit --quiet --message one && git push --quiet origin main
  git remote set-head origin main
}

# worktree makes a worktree on branch 1-x beside the desk, configured to commit.
worktree() {
  git worktree add --quiet ../wt -b 1-x
  git -C ../wt config commit.gpgsign false
}

# commit-in <checkout> <file> <content> writes the file and commits it there.
commit-in() {
  printf '%s\n' "$3" > "$1/$2"
  git -C "$1" add "$2"
  git -C "$1" commit --quiet --message "$2"
}

@test "rebase refuses the root checkout, whatever branch it has" {
  run "${kit_checkout}/bin/kit" rebase
  [ "$status" -eq 1 ]
  [[ "$output" == *"refusing to rebase main: it is the root checkout"* ]]
  git switch --quiet --create desk-topic
  run "${kit_checkout}/bin/kit" rebase
  [ "$status" -eq 1 ]
  [[ "$output" == *"refusing to rebase desk-topic: it is the root checkout"* ]]
}

@test "rebase refuses the default branch in a worktree" {
  git switch --quiet --create desk-topic
  git worktree add --quiet ../wt main
  run "${kit_checkout}/bin/kit" rebase ../wt
  [ "$status" -eq 1 ]
  [[ "$output" == *"refusing to rebase main: it is the default branch"* ]]
}

@test "rebase moves a worktree's branch onto origin/main and pushes it" {
  worktree
  commit-in ../wt b two && git -C ../wt push --quiet --set-upstream origin 1-x
  commit-in . c three && git push --quiet origin main
  run "${kit_checkout}/bin/kit" rebase ../wt
  [ "$status" -eq 0 ]
  [ "$(git -C ../wt rev-parse HEAD~1)" = "$(git rev-parse origin/main)" ]
  [ "$(git -C ../wt rev-parse HEAD)" = "$(git -C ../wt rev-parse origin/1-x)" ]
}

@test "in a fork, rebase moves the branch onto the primary's default branch and pushes it to its own upstream" {
  git init --quiet --bare --initial-branch=main ../upstream.git
  git remote add upstream ../upstream.git
  git push --quiet upstream main
  printf 'primary: upstream\n' > kit.yml
  worktree
  commit-in ../wt b two && git -C ../wt push --quiet --set-upstream origin 1-x
  git clone --quiet ../upstream.git ../shared 2>/dev/null
  commit-in ../shared c three && git -C ../shared push --quiet origin main
  run "${kit_checkout}/bin/kit" rebase ../wt
  [ "$status" -eq 0 ]
  [[ "$output" == *"kit rebase 1-x onto upstream/main"* ]]
  [ "$(git -C ../wt rev-parse HEAD~1)" = "$(git rev-parse upstream/main)" ]
  [ "$(git -C ../wt rev-parse HEAD)" = "$(git -C ../wt rev-parse origin/1-x)" ]
}

@test "rebase aborts on a conflict and leaves the worktree as it was" {
  worktree
  commit-in ../wt a theirs && git -C ../wt push --quiet --set-upstream origin 1-x
  commit-in . a ours && git push --quiet origin main
  before=$(git -C ../wt rev-parse HEAD)
  run "${kit_checkout}/bin/kit" rebase ../wt
  [ "$status" -eq 1 ]
  [[ "$output" == *"conflicts with origin/main"* ]]
  [ "$(git -C ../wt rev-parse HEAD)" = "${before}" ]
  [ -z "$(git -C ../wt status --porcelain)" ]
}

# other-commits <file> <content> commits the file on 1-x in a second clone, ../other, and pushes it, as another
# session does.
other-commits() {
  git clone --quiet ../origin.git ../other 2>/dev/null
  git -C ../other switch --quiet 1-x
  commit-in ../other "$1" "$2" && git -C ../other push --quiet
}

@test "rebase takes the commits origin's branch has and this checkout lacks, and pushes both" {
  worktree
  git -C ../wt push --quiet --set-upstream origin 1-x
  other-commits theirs.txt theirs
  commit-in ../wt mine.txt mine
  run "${kit_checkout}/bin/kit" rebase ../wt
  [ "$status" -eq 0 ]
  [[ "$output" == *"taking 1 commit(s) origin/1-x has and this checkout lacks"* ]]
  git -C ../origin.git merge-base --is-ancestor "$(git -C ../other rev-parse HEAD)" 1-x
  [ "$(git -C ../origin.git rev-parse 1-x)" = "$(git -C ../wt rev-parse HEAD)" ]
}

@test "rebase aborts on a conflict with origin's branch and leaves the worktree and origin's branch as they were" {
  worktree
  git -C ../wt push --quiet --set-upstream origin 1-x
  other-commits a theirs
  commit-in ../wt a mine
  before=$(git -C ../wt rev-parse HEAD)
  theirs=$(git -C ../other rev-parse HEAD)
  run "${kit_checkout}/bin/kit" rebase ../wt
  [ "$status" -eq 1 ]
  [[ "$output" == *"1-x conflicts with origin/1-x"* ]]
  [ "$(git -C ../wt rev-parse HEAD)" = "${before}" ]
  [ -z "$(git -C ../wt status --porcelain)" ]
  [ "$(git -C ../origin.git rev-parse 1-x)" = "${theirs}" ]
}

@test "rebase takes origin's branch, then conflicts with origin/main: the worktree and origin's branch stay" {
  local before theirs
  worktree
  git -C ../wt push --quiet --set-upstream origin 1-x
  other-commits theirs.txt theirs
  commit-in ../wt a mine
  commit-in . a ours && git push --quiet origin main
  before=$(git -C ../wt rev-parse HEAD)
  theirs=$(git -C ../other rev-parse HEAD)
  run "${kit_checkout}/bin/kit" rebase ../wt
  [ "$status" -eq 1 ]
  [[ "$output" == *"taking 1 commit(s) origin/1-x has and this checkout lacks"* ]]
  [[ "$output" == *"1-x conflicts with origin/main; the rebase was aborted and the worktree is as it was"* ]]
  [ "$(git -C ../wt rev-parse HEAD)" = "${before}" ]
  [ -z "$(git -C ../wt status --porcelain)" ]
  [ ! -e ../wt/theirs.txt ]
  [ "$(git -C ../origin.git rev-parse 1-x)" = "${theirs}" ]
}

@test "rebase never overwrites a push that lands after its fetch: the push fails, and a second run takes it" {
  local theirs
  worktree
  commit-in ../wt mine.txt mine && git -C ../wt push --quiet --set-upstream origin 1-x
  git clone --quiet ../origin.git ../other 2>/dev/null
  git -C ../other switch --quiet 1-x
  commit-in ../other theirs.txt theirs
  theirs=$(git -C ../other rev-parse HEAD)
  commit-in . c three && git push --quiet origin main
  # another session's push lands between kit rebase's fetch and its push, once: git runs the pre-rebase hook after
  # the fetch, and a push that git began would see the new tip, so only the lease can refuse to overwrite it
  cat > "$(git rev-parse --git-common-dir)/hooks/pre-rebase" <<HOOK
#!/bin/sh
unset \$(git rev-parse --local-env-vars)
[ -e "${work}/raced" ] || { : > "${work}/raced"; git -C "${work}/other" push --quiet; }
HOOK
  chmod +x "$(git rev-parse --git-common-dir)/hooks/pre-rebase"
  run "${kit_checkout}/bin/kit" rebase ../wt
  [ "$status" -eq 1 ]
  [[ "$output" == *"1-x is rebased here and was not pushed; see above, then run: kit rebase"* ]]
  [ "$(git -C ../origin.git rev-parse 1-x)" = "${theirs}" ]
  run "${kit_checkout}/bin/kit" rebase ../wt
  [ "$status" -eq 0 ]
  [[ "$output" == *"commit(s) origin/1-x has and this checkout lacks"* ]]
  # main moved, so their commit is rewritten onto it, and kept
  [ "$(git -C ../origin.git show 1-x:theirs.txt)" = theirs ]
  [ "$(git -C ../origin.git show 1-x:mine.txt)" = mine ]
  [ "$(git -C ../origin.git rev-parse 1-x)" = "$(git -C ../wt rev-parse HEAD)" ]
  [ "$(git -C ../wt rev-parse HEAD~2)" = "$(git rev-parse origin/main)" ]
}

@test "rebase pushes the branch to its own name on origin, never where push.default points" {
  git worktree add --quiet ../wt -b 1-x origin/main
  git -C ../wt config push.default upstream
  commit-in ../wt b two
  main=$(git -C ../origin.git rev-parse main)
  run "${kit_checkout}/bin/kit" rebase ../wt
  [ "$status" -eq 0 ]
  [ "$(git -C ../origin.git rev-parse main)" = "${main}" ]
  [ "$(git -C ../wt rev-parse --abbrev-ref '@{upstream}')" = origin/1-x ]
  [ "$(git -C ../origin.git rev-parse 1-x)" = "$(git -C ../wt rev-parse HEAD)" ]
}

@test "with require_signing true, rebase signs every commit it rewrites, here and on origin" {
  printf 'require_signing: true\n' > kit.yml
  signer
  worktree
  printf 'two\n' > ../wt/b && git -C ../wt add b && git -C ../wt commit --quiet --gpg-sign --message b
  git -C ../wt push --quiet --set-upstream origin 1-x
  git -C ../wt cat-file commit HEAD | grep --quiet '^gpgsig '
  commit-in . c three && git push --quiet origin main
  run "${kit_checkout}/bin/kit" rebase ../wt
  [ "$status" -eq 0 ]
  [ "$(git -C ../wt rev-parse HEAD~1)" = "$(git rev-parse origin/main)" ]
  git -C ../wt cat-file commit HEAD | grep --quiet '^gpgsig '
  [ "$(git -C ../origin.git rev-parse 1-x)" = "$(git -C ../wt rev-parse HEAD)" ]
  git -C ../origin.git cat-file commit 1-x | grep --quiet '^gpgsig '
}

@test "with require_signing true and no signing key, rebase refuses before it fetches or moves anything" {
  local head origin_main
  printf 'require_signing: true\n' > kit.yml
  worktree
  commit-in ../wt b two && git -C ../wt push --quiet --set-upstream origin 1-x
  # origin's main moves on in another clone, so a fetch here would show
  git clone --quiet ../origin.git ../other 2>/dev/null
  commit-in ../other c three && git -C ../other push --quiet origin main
  head=$(git -C ../wt rev-parse HEAD)
  origin_main=$(git rev-parse origin/main)
  run "${kit_checkout}/bin/kit" rebase ../wt
  [ "$status" -eq 1 ]
  [[ "$output" == *"run: git config user.signingkey <key> (every commit here is signed, and none is configured)"* ]]
  [ "$(git -C ../wt rev-parse HEAD)" = "${head}" ]
  [ "$(git rev-parse origin/main)" = "${origin_main}" ]
  [ "$(git -C ../origin.git rev-parse 1-x)" = "${head}" ]
  [ -z "$(git -C ../wt status --porcelain)" ]
}

# ----------------------------------------------------------------------------------------------------------------------
