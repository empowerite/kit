#!/usr/bin/env bats
# across.bats - kit flow across a workspace: its members and their checkouts, each member's kit flow merged into one
# document per stage, every row tagged with its member, and the repos stage
# ----------------------------------------------------------------------------------------------------------------------
# Each test runs with HOME, XDG_CONFIG_HOME, and XDG_CACHE_HOME of its own, as helper:isolate sets them, so the
# workspace file starts missing.

bats_require_minimum_version 1.5.0

load helper

setup() {
  hide-system-toolchains
  kit_bash_modules=across
}

# clone-at <path> <url> makes a repository at <path> whose origin is <url>; nothing is fetched.
clone-at() {
  git init --quiet "$1"
  git -C "$1" remote add origin "$2"
}

# workspaces-file writes what it reads to the workspace file.
workspaces-file() {
  mkdir -p "${XDG_CONFIG_HOME}/kit"
  cat > "${XDG_CONFIG_HOME}/kit/workspaces.yml"
}

@test "across:members gives each member and its checkout, in the group's order, empty for one not cloned" {
  clone-at "${BATS_TEST_TMPDIR}/code/b" https://github.com/o/b.git
  clone-at "${BATS_TEST_TMPDIR}/code/a dir/a" https://github.com/o/a.git
  workspaces-file <<EOF
roots: [${BATS_TEST_TMPDIR}/code]
workspaces:
  w: {repos: [github.com/o/a, github.com/o/c, github.com/o/b]}
EOF
  run kit-bash "across:members w"
  [ "$status" -eq 0 ]
  [ "$output" = "$(printf '%s\t%s\n' github.com/o/a "${BATS_TEST_TMPDIR}/code/a dir/a" github.com/o/c '' \
    github.com/o/b "${BATS_TEST_TMPDIR}/code/b")" ]
}

@test "across:members refuses a workspace that is neither a group nor a repository, and none, with 2" {
  run kit-bash "across:members nosuch"
  [ "$status" -eq 2 ]
  [[ "$output" == *"'nosuch' is no workspace"* ]]
  run kit-bash "across:members none"
  [ "$status" -eq 2 ]
  [[ "$output" == *"'none' is no workspace"* ]]
}

@test "across:merge tags each row with its member, in order, and turns a failed member into its line" {
  d=${BATS_TEST_TMPDIR}/m && mkdir -p "$d"
  printf 'github.com/o/a' > "$d/0.member"; printf '[{"n":1},{"n":2}]' > "$d/0.out"; printf 0 > "$d/0.status"
  : > "$d/0.err"
  printf 'github.com/o/b' > "$d/1.member"; : > "$d/1.out"; printf 1 > "$d/1.status"
  printf 'a warning\n❌ gh cannot read o/b\n' > "$d/1.err"
  printf 'github.com/o/c' > "$d/2.member"; printf '[{"n":3}]' > "$d/2.out"; printf 0 > "$d/2.status"; : > "$d/2.err"
  run kit-bash "across:merge '$d' array"
  [ "$status" -eq 0 ]
  [ "$output" = '{"rows":[{"n":1,"member":"github.com/o/a"},{"n":2,"member":"github.com/o/a"},{"n":3,"member":"github.com/o/c"}],"errors":[{"member":"github.com/o/b","error":"gh cannot read o/b"}]}' ]
}

@test "across:merge tags an object stage's object with its member" {
  d=${BATS_TEST_TMPDIR}/m && mkdir -p "$d"
  printf 'github.com/o/a' > "$d/0.member"; printf '{"branch":"main"}' > "$d/0.out"; printf 0 > "$d/0.status"
  : > "$d/0.err"
  run kit-bash "across:merge '$d' object"
  [ "$output" = '{"rows":[{"branch":"main","member":"github.com/o/a"}],"errors":[]}' ]
}
