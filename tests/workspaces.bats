#!/usr/bin/env bats
# workspaces.bats - kit workspaces: members and checkouts, the file, the startup rule, and editing the file
# ----------------------------------------------------------------------------------------------------------------------
# Each test runs with HOME, XDG_CONFIG_HOME, and XDG_CACHE_HOME of its own, as helper:isolate sets them, so the
# workspace file and the last group used start missing. A clone is a git init with an origin added; nothing fetches.

bats_require_minimum_version 1.5.0

load helper

setup() {
  hide-system-toolchains
  kit_bash_modules=workspaces
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

@test "git:desks stops at a desk, skips hidden directories and links, and goes four levels down" {
  root="${BATS_TEST_TMPDIR}/code"
  mkdir -p "${root}/a/.git" "${root}/a/inner/.git" "${root}/.hidden/h/.git"
  mkdir -p "${root}/1/2/3/4/.git" "${root}/1/2/3/4/5/.git"
  mkdir -p "${root}/x/y/z/w/v/.git"
  ln -s "${root}/a" "${root}/link"
  run kit-bash "git:desks '${root}' 4"
  [ "$status" -eq 0 ]
  [ "$output" = "${root}/1/2/3/4
${root}/a" ]
}

@test "a remote's URL gives its member, every segment of its path kept" {
  run kit-bash 'for u in git@github.com:empowerite/edge.git https://github.com/empowerite/kit \
      ssh://git@gitlab.com/g/sub/repo.git /srv/repo.git file:///srv/r https://host/one
    do printf "[%s]\n" "$(workspaces:url-member "$u")"; done'
  [ "$output" = "[github.com/empowerite/edge]
[github.com/empowerite/kit]
[gitlab.com/g/sub/repo]
[]
[]
[]" ]
}

@test "a checkout's member is its primary remote's: kit.yml's primary, else origin" {
  clone-at "${BATS_TEST_TMPDIR}/fork" git@github.com:me/edge.git
  git -C "${BATS_TEST_TMPDIR}/fork" remote add upstream git@github.com:empowerite/edge.git
  run kit-bash "workspaces:member-of '${BATS_TEST_TMPDIR}/fork'"
  [ "$output" = "github.com/me/edge" ]
  printf 'primary: upstream\n' > "${BATS_TEST_TMPDIR}/fork/kit.yml"
  run kit-bash "workspaces:member-of '${BATS_TEST_TMPDIR}/fork'"
  [ "$output" = "github.com/empowerite/edge" ]
}

@test "the checkouts are every desk under the roots with a member, the earliest root first" {
  clone-at "${BATS_TEST_TMPDIR}/b/edge" git@github.com:empowerite/edge.git
  clone-at "${BATS_TEST_TMPDIR}/a/github.com/empowerite/kit" git@github.com:empowerite/kit.git
  clone-at "${BATS_TEST_TMPDIR}/a/kit-again" https://github.com/empowerite/kit.git
  git init --quiet "${BATS_TEST_TMPDIR}/a/no-remote"
  clone-at "${BATS_TEST_TMPDIR}/a/local" /srv/repo.git
  t=${BATS_TEST_TMPDIR}
  printf 'roots: [%s/b, %s/a, %s/a]\n' "$t" "$t" "$t" | workspaces-file
  run --separate-stderr kit-bash workspaces:checkouts
  [ "$status" -eq 0 ]
  [ -z "$stderr" ]
  [ "$output" = "$(printf '%s\t%s\n' \
    github.com/empowerite/edge "${BATS_TEST_TMPDIR}/b/edge" \
    github.com/empowerite/kit "${BATS_TEST_TMPDIR}/a/github.com/empowerite/kit" \
    github.com/empowerite/kit "${BATS_TEST_TMPDIR}/a/kit-again")" ]
}

@test "a root given with ~ is under HOME" {
  printf 'roots: ["~/Code"]\n' | workspaces-file
  run kit-bash workspaces:roots
  [ "$output" = "${HOME}/Code" ]
}
