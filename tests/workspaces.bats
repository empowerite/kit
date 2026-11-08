#!/usr/bin/env bats
# workspaces.bats - kit workspaces: members and checkouts, the file, the startup rule, and editing the file
# ----------------------------------------------------------------------------------------------------------------------
# Each test runs with HOME, XDG_CONFIG_HOME, and XDG_CACHE_HOME of its own, as helper:isolate sets them, so the
# workspace file and the last group used start missing. A clone is a git init with an origin added; nothing fetches.

bats_require_minimum_version 1.5.0

load helper

setup() {
  hide-system-toolchains
  kit_bash_modules=git
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
