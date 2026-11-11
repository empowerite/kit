#!/usr/bin/env bats
# trust.bats - kit runs nothing in a clone until you trust it: the loader, kit:enter, the module path, and kit trust
# ----------------------------------------------------------------------------------------------------------------------
# Each test gets an empty git repository and a trust file of its own, empty, so the repository starts untrusted; the
# test helper trusts every clone for every other file. A clone that would run code of its own holds a kit/bin/kit
# that leaves a mark when it runs.

bats_require_minimum_version 1.5.0

load helper

setup() {
  fresh-repo >/dev/null
  kit="${kit_checkout}/bin/kit"
  trusted="${XDG_CONFIG_HOME}/kit/trusted"
  : > "${trusted}"
}

# own-kit makes the clone declare unreleased and hold a kit/ of its own whose bin/kit leaves ${BATS_TEST_TMPDIR}/ran
# when it runs.
own-kit() {
  printf 'version: unreleased\n' > kit.yml
  mkdir -p kit/bin kit/modules/bash
  printf '#!/bin/sh\n: > "%s/ran"\n' "${BATS_TEST_TMPDIR}" > kit/bin/kit
  chmod +x kit/bin/kit
  : > kit/modules/bash/1st.sh
}

# answering <answer> <command> runs the command on a terminal of its own, and types the answer there once the question
# is up; script gives the command the terminal, as tests/helper.bash's pty does.
answering() {
  local answer=$1
  shift
  { sleep 2; printf '%s\n' "${answer}"; sleep 2; } | if script --version >/dev/null 2>&1
  then
    script --quiet --return --command "$*" /dev/null
  else
    script -q /dev/null bash -c "$*"
  fi
}

@test "in a clone you have not trusted, every verb but help, version, and trust is refused, by the loader and by the verb" {
  run "${kit}" check
  [ "$status" -eq 1 ]
  [[ "$output" == *"kit runs nothing in $(pwd -P) until you trust it; see what it asks kit to run, and trust it: kit trust"* ]]
  run "${kit}"
  [ "$status" -eq 1 ]
  run "${kit_checkout}/bin/kit-check"
  [ "$status" -eq 1 ]
  [[ "$output" == *"kit runs nothing in $(pwd -P) until you trust it"* ]]
  run "${kit}" version
  [ "$status" -eq 0 ]
  [ "$output" = unreleased ]
  run "${kit}" help
  [ "$status" -eq 0 ]
  run "${kit}" -h
  [ "$status" -eq 0 ]
}

@test "on a terminal, a verb in a clone you have not trusted asks first, and after a yes runs as it was typed" {
  run answering y "'${kit}' root"
  [ "$status" -eq 0 ]
  [[ "$output" == *"kit runs nothing in $(pwd -P) until you trust it"* ]]
  [[ "$output" == *"Trust $(pwd -P)? [y/N]"* ]]
  [[ "$output" == *"kit trusts $(pwd -P), and every worktree of it"* ]]
  [[ "$output" == *"${kit_checkout}"* ]]
  [ "$(cat "${trusted}")" = "$(pwd -P)" ]
}

@test "on a terminal, a no keeps the clone untrusted, and the verb does not run" {
  run answering n "'${kit}' root"
  [ "$status" -eq 1 ]
  [[ "$output" == *"is not trusted; kit runs nothing there"* ]]
  [[ "$output" != *"${kit_checkout}"* ]]
  [ ! -s "${trusted}" ]
}

@test "outside a clone, kit needs no trust" {
  mkdir "${BATS_TEST_TMPDIR}/plain"
  run bash -c 'cd "$1" && "$0" kinds x' "${kit}" "${BATS_TEST_TMPDIR}/plain"
  [[ "$output" != *"until you trust it"* ]]
}

@test "a clone you have not trusted runs none of its own code: not its kit/, not a release it names, not its bash_path" {
  own-kit
  printf 'version: unreleased\nbash_path:\n  - kit:modules/bash/\n  - lib/bash/\n' > kit.yml
  mkdir -p lib/bash
  printf ': > "%s/sourced"\n' "${BATS_TEST_TMPDIR}" > lib/bash/help.sh
  run "${kit}" version
  [ "$output" = unreleased ]
  run "${kit}" help
  [ "$status" -eq 0 ]
  run "${kit}" --version=9.9.9 version
  [ "$output" = unreleased ]
  [ ! -e "${BATS_TEST_TMPDIR}/ran" ]
  [ ! -e "${BATS_TEST_TMPDIR}/sourced" ]
}

@test "kit trust shows what the clone asks kit to run, and with no terminal and no --yes records nothing" {
  own-kit
  printf 'version: unreleased\ngate: bin/ci\nbash_path:\n  - kit:modules/bash/\n  - lib/bash/\nsources:\n  - name: house\n    url: https://example.com/house.git\n' > kit.yml
  run "${kit}" trust </dev/null
  [ "$status" -eq 1 ]
  [[ "$output" == *"$(pwd -P) asks kit to:"* ]]
  [[ "$output" == *"  - run its own kit, kit/ in the clone, for every kit command there (version: unreleased)"* ]]
  [[ "$output" == *"  - load bash into every kit command from lib/bash/ (bash_path)"* ]]
  [[ "$output" == *"  - run bin/ci for kit gate (gate)"* ]]
  [[ "$output" == *"  - bring packages from the source house (sources)"* ]]
  [[ "$output" == *"there is no terminal to ask; to trust it without asking, run: kit trust --yes $(pwd -P)"* ]]
  [ ! -s "${trusted}" ]
  [ ! -e "${BATS_TEST_TMPDIR}/ran" ]
}

@test "kit trust --yes records the clone, and kit runs there then, its own kit/ included" {
  run "${kit}" trust --yes
  [ "$status" -eq 0 ]
  [[ "$output" == *"asks kit to run nothing beyond the kit itself"* ]]
  [[ "$output" == *"kit trusts $(pwd -P), and every worktree of it"* ]]
  [ "$(cat "${trusted}")" = "$(pwd -P)" ]
  run "${kit}" trust --yes
  [[ "$output" == *"kit trusts $(pwd -P) already"* ]]
  [ "$(wc -l < "${trusted}")" -eq 1 ]
  own-kit
  run "${kit}" check
  [ -e "${BATS_TEST_TMPDIR}/ran" ]
}

@test "a worktree is trusted with its desk, wherever the worktree is" {
  git commit --quiet --allow-empty --message init
  git worktree add --quiet "${BATS_TEST_TMPDIR}/away" -b away
  run bash -c 'cd "$1" && "$0" trust --check' "${kit}" "${BATS_TEST_TMPDIR}/away"
  [ "$status" -eq 1 ]
  [ "$output" = "$(pwd -P)" ]
  "${kit}" trust --yes
  run bash -c 'cd "$1" && "$0" trust --check' "${kit}" "${BATS_TEST_TMPDIR}/away"
  [ "$status" -eq 0 ]
  run bash -c 'cd "$1" && "$0" kinds x' "${kit}" "${BATS_TEST_TMPDIR}/away"
  [[ "$output" != *"until you trust it"* ]]
}

@test "a .git file the desk does not confirm borrows none of its trust: a worktree of its own, or a name made up" {
  git commit --quiet --allow-empty --message init
  git worktree add --quiet "${BATS_TEST_TMPDIR}/real" -b real
  "${kit}" trust --yes
  desk=$(pwd -P)
  mkdir "${BATS_TEST_TMPDIR}/spoof"
  printf 'gitdir: %s/.git/worktrees/real\n' "${desk}" > "${BATS_TEST_TMPDIR}/spoof/.git"
  run "${kit}" trust --check "${BATS_TEST_TMPDIR}/spoof"
  [ "$status" -eq 1 ]
  [ "$output" = "$(cd "${BATS_TEST_TMPDIR}/spoof" && pwd -P)" ]
  run bash -c 'cd "$1" && "$0" check' "${kit}" "${BATS_TEST_TMPDIR}/spoof"
  [ "$status" -eq 1 ]
  [[ "$output" == *"kit runs nothing in $(cd "${BATS_TEST_TMPDIR}/spoof" && pwd -P) until you trust it"* ]]
  printf 'gitdir: %s/.git/worktrees/made-up\n' "${desk}" > "${BATS_TEST_TMPDIR}/spoof/.git"
  run "${kit}" trust --check "${BATS_TEST_TMPDIR}/spoof"
  [ "$status" -eq 1 ]
  # an admin directory of the clone's own making, which names the clone back and the desk as its common dir
  mkdir -p "${BATS_TEST_TMPDIR}/spoof/admin/x"
  printf '%s/.git\n' "$(cd "${BATS_TEST_TMPDIR}/spoof" && pwd -P)" > "${BATS_TEST_TMPDIR}/spoof/admin/x/gitdir"
  printf '%s/.git\n' "${desk}" > "${BATS_TEST_TMPDIR}/spoof/admin/x/commondir"
  printf 'gitdir: admin/x\n' > "${BATS_TEST_TMPDIR}/spoof/.git"
  run "${kit}" trust --check "${BATS_TEST_TMPDIR}/spoof"
  [ "$status" -eq 1 ]
  [ "$output" = "$(cd "${BATS_TEST_TMPDIR}/spoof" && pwd -P)" ]
  run bash -c 'cd "$1" && "$0" check' "${kit}" "${BATS_TEST_TMPDIR}/spoof"
  [ "$status" -eq 1 ]
  run "${kit}" trust --check "${BATS_TEST_TMPDIR}/real"
  [ "$status" -eq 0 ]
  [ "$output" = "${desk}" ]
}

@test "a folder ending in / trusts every clone under it, and none beside it" {
  mkdir -p "${BATS_TEST_TMPDIR}/code/a" "${BATS_TEST_TMPDIR}/codex"
  git init --quiet "${BATS_TEST_TMPDIR}/code/a/one"
  git init --quiet "${BATS_TEST_TMPDIR}/codex/two"
  run "${kit}" trust --yes "${BATS_TEST_TMPDIR}/code/"
  [ "$status" -eq 0 ]
  [[ "$output" == *"trusts every clone under it"* ]]
  [ "$(cat "${trusted}")" = "$(cd "${BATS_TEST_TMPDIR}/code" && pwd -P)/" ]
  run "${kit}" trust --check "${BATS_TEST_TMPDIR}/code/a/one"
  [ "$status" -eq 0 ]
  run "${kit}" trust --check "${BATS_TEST_TMPDIR}/codex/two"
  [ "$status" -eq 1 ]
}

@test "the trust file takes comments, blank lines, and ~/ for the home directory" {
  mkdir -p "${HOME}/src"
  git init --quiet "${HOME}/src/mine"
  printf '# mine\n\n~/src/\n' > "${trusted}"
  run "${kit}" trust --check "${HOME}/src/mine"
  [ "$status" -eq 0 ]
  run "${kit}" trust --list
  [ "$output" = "~/src/" ]
}

@test "kit trust --revoke takes the trust back, says when there is none, and warns when a folder still holds the clone" {
  "${kit}" trust --yes
  run "${kit}" trust --revoke
  [ "$status" -eq 0 ]
  [[ "$output" == *"kit no longer trusts $(pwd -P)"* ]]
  run "${kit}" check
  [ "$status" -eq 1 ]
  run "${kit}" trust --revoke
  [ "$status" -eq 1 ]
  [[ "$output" == *"kit trust holds no record of $(pwd -P)"* ]]
  printf '%s/\n%s\n' "$(cd .. && pwd -P)" "$(pwd -P)" > "${trusted}"
  run "${kit}" trust --revoke
  [[ "$output" == *"a folder you trust still holds $(pwd -P)"* ]]
}

@test "kit trust runs on the installed kit in a trusted clone too, so a clone of its own kit cannot keep its trust" {
  own-kit
  "${kit}" trust --yes
  rm -f "${BATS_TEST_TMPDIR}/ran"
  run "${kit}" trust --list
  [ "$output" = "$(pwd -P)" ]
  run "${kit}" trust --revoke
  [ "$status" -eq 0 ]
  [ ! -e "${BATS_TEST_TMPDIR}/ran" ]
}

@test "kit.yml names a release by its version alone, even in a clone you trust; a path only --version= names" {
  "${kit}" trust --yes
  later-release "${BATS_TEST_TMPDIR}/elsewhere" 9.9.7
  printf 'version: %s\n' "${BATS_TEST_TMPDIR}/elsewhere" > kit.yml
  run "${kit}" version
  [ "$status" -eq 1 ]
  [[ "$output" == *"kit.yml names a release by its version, and a path only --version= or KIT_VERSION names"* ]]
  run "${kit}" --version="${BATS_TEST_TMPDIR}/elsewhere" version
  [ "$output" = 9.9.7 ]
}

# ----------------------------------------------------------------------------------------------------------------------
