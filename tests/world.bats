#!/usr/bin/env bats
# world.bats - a world built once per file and copied into each test is the repository a fresh install makes
# ----------------------------------------------------------------------------------------------------------------------
# world-make and world-copy, in tests/helper.bash, let a file pay for one kit install instead of one per test. These
# tests hold the copy to what setup() made before: the same files with the same modes, the kit's own check green, a
# remote that follows the copy, and a copy no other test shares.

load helper

# helper:desk is a world with a remote: a bare origin beside a desk cloned from it, with the kit installed.
helper:desk() {
  git init --quiet --bare --initial-branch=main origin.git
  git clone --quiet origin.git desk 2>/dev/null
  cd desk && "${kit_checkout}/bin/kit" install
}

setup_file() {
  world-repo installed
  world-make desk helper:desk
}

# tree <dir> prints every path under <dir> but .git, with its type, and whether it is executable or read-only, one
# per line, sorted. A log names the directory and the time it was written in, so the logs are left out.
tree() {
  local test
  (
    cd "$1" || exit 1
    for test in '-type d' '-type f' '-type l' '-type f -perm -u+x' '-type f ! -perm -u+w'
    do
      # shellcheck disable=SC2086  # the test splits into find's words on purpose
      find . -path ./.git -prune -o -path ./.kit/logs -prune -o ${test} -print | sed "s|^|${test}: |"
    done
  ) | LC_ALL=C sort
}

@test "a copied world holds the paths, types, and modes a fresh install makes, and the kit's check passes there" {
  world-copy-repo installed
  [ "${PWD}" = "${BATS_TEST_TMPDIR}/repo" ]
  fresh=${BATS_TEST_TMPDIR}/fresh
  git init --quiet --initial-branch=main "${fresh}"
  (cd "${fresh}" && "${kit_checkout}/bin/kit" install >/dev/null 2>&1)
  [ "$(tree "${repo}")" = "$(tree "${fresh}")" ]
  [ "$(git status --porcelain)" = "$(git -C "${fresh}" status --porcelain)" ]
  run "${kit_checkout}/bin/kit" check
  [ "$status" -eq 0 ]
}

@test "a remote in the world names the copy, and a worktree added after the copy works" {
  world-copy desk
  cd "${BATS_TEST_TMPDIR}/desk"
  [ "$(git remote get-url origin)" = "${BATS_TEST_TMPDIR}/origin.git" ]
  git commit --quiet --allow-empty --message '#1 start: the first'
  git push --quiet origin main
  [ "$(git -C "${BATS_TEST_TMPDIR}/origin.git" rev-parse main)" = "$(git rev-parse HEAD)" ]
  git worktree add --quiet -b 7-x worktrees/7-x main
  [ "$(git -C worktrees/7-x rev-parse --abbrev-ref HEAD)" = 7-x ]
}

@test "each test gets a copy of its own: what one test writes, the next does not see" {
  world-copy-repo installed
  [ ! -e left-by-a-test ]
  touch left-by-a-test
  [ ! -e "${BATS_FILE_TMPDIR}/world/installed/root/repo/left-by-a-test" ]
}

@test "and the next test starts from the world as it was built" {
  world-copy-repo installed
  [ ! -e left-by-a-test ]
}

@test "a world whose build fails fails setup, showing what the build printed" {
  helper:broken() { echo "the build said this"; return 3; }
  run world-make broken helper:broken
  [ "$status" -eq 1 ]
  [[ "$output" == *"the build said this"* ]]
}

# ----------------------------------------------------------------------------------------------------------------------
