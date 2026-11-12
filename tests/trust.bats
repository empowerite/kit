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

# recorded <path> succeeds when the trust file holds one line, a clone's record of <path>: a sha256, two spaces, and
# the path.
recorded() {
  local line
  line=$(cat "${trusted}")
  [[ "${line}" =~ ^[0-9a-f]{64}\ \ (.*)$ && "${BASH_REMATCH[1]}" == "$1" ]]
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
  recorded "$(pwd -P)"
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
  [[ "$output" == *"there is no terminal to ask; to trust it without asking, run: cd "*" && kit trust --yes $(pwd -P)"* ]]
  [ ! -s "${trusted}" ]
  [ ! -e "${BATS_TEST_TMPDIR}/ran" ]
}

@test "kit trust --yes records the clone, and kit runs there then, its own kit/ included" {
  run "${kit}" trust --yes
  [ "$status" -eq 0 ]
  [[ "$output" == *"asks kit to run nothing beyond the kit itself"* ]]
  [[ "$output" == *"kit trusts $(pwd -P), and every worktree of it"* ]]
  recorded "$(pwd -P)"
  run "${kit}" trust --yes
  [[ "$output" == *"kit trusts $(pwd -P) already"* ]]
  [ "$(wc -l < "${trusted}")" -eq 1 ]
  own-kit
  "${kit}" trust --yes
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
  later-release "${BATS_TEST_TMPDIR}/elsewhere" 9.9.7
  printf 'version: %s\n' "${BATS_TEST_TMPDIR}/elsewhere" > kit.yml
  "${kit}" trust --yes
  run "${kit}" version
  [ "$status" -eq 1 ]
  [[ "$output" == *"kit.yml names a release by its version, and a path only --version= or KIT_VERSION names"* ]]
  run "${kit}" --version="${BATS_TEST_TMPDIR}/elsewhere" version
  [ "$output" = 9.9.7 ]
}

@test "a clone that comes to ask kit to run something new is asked again; a change to anything else keeps its trust" {
  printf 'gate: bin/ci\npackages:\n  - kit\n' > kit.yml
  "${kit}" trust --yes
  run "${kit}" trust --check
  [ "$status" -eq 0 ]
  # what names no command: the packages, a README, the gate's own script
  printf 'gate: bin/ci\npackages:\n  - kit\n  - workflow\nanswers:\n  x: "1"\n' > kit.yml
  mkdir bin && printf '#!/bin/sh\n' > bin/ci && printf 'hello\n' > README.md
  run "${kit}" trust --check
  [ "$status" -eq 0 ]
  run "${kit}" kinds x
  [[ "$output" != *"since you trusted it"* ]]
  # a new gate, by a key written in quotes too
  printf '"gate": bin/other\npackages:\n  - kit\n' > kit.yml
  run "${kit}" trust --check
  [ "$status" -eq 1 ]
  [ "$output" = "$(pwd -P)" ]
  run "${kit}" kinds x
  [ "$status" -eq 1 ]
  [[ "$output" == *"$(pwd -P) asks kit to run something new since you trusted it; see what it asks, and trust it again: kit trust"* ]]
  run "${kit_checkout}/bin/kit-kinds" x
  [ "$status" -eq 1 ]
  [[ "$output" == *"asks kit to run something new since you trusted it"* ]]
  run "${kit}" trust --yes
  [[ "$output" == *"asks kit to run something new since you trusted it"* ]]
  run "${kit}" trust --check
  [ "$status" -eq 0 ]
  # a lint: command in a rule, and in a rule of the clone's own package
  mkdir -p .kit/rules
  printf -- '---\nkind:\n  - text\nlint: touch %s/linted\n---\n# Text\n' "${BATS_TEST_TMPDIR}" > .kit/rules/text.md
  run "${kit}" trust --check
  [ "$status" -eq 1 ]
  "${kit}" trust --yes
  printf '# Text\n\nAnother line of prose.\n' >> .kit/rules/text.md
  run "${kit}" trust --check
  [ "$status" -eq 0 ]
  mkdir -p .kit/packages/local/files/repo/.kit/styles
  printf -- '---\nkind:\n  - text\nlint: rm -rf /\n---\n' > .kit/packages/local/files/repo/.kit/styles/text.md
  run "${kit}" trust --check
  [ "$status" -eq 1 ]
  # a version kit.yml no longer names, which .kit/manifest names in its place
  rm -rf .kit/packages
  "${kit}" trust --yes
  printf 'version 9.9.9\n' > .kit/manifest
  run "${kit}" trust --check
  [ "$status" -eq 1 ]
}

@test "a trusted clone that comes to pin its own kit/ is asked again before the loader runs it" {
  "${kit}" trust --yes
  own-kit
  run "${kit}" kinds x
  [ "$status" -eq 1 ]
  [[ "$output" == *"asks kit to run something new since you trusted it"* ]]
  [ ! -e "${BATS_TEST_TMPDIR}/ran" ]
  run answering y "'${kit}' kinds x"
  [[ "$output" == *"asks kit to run something new since you trusted it"* ]]
  [[ "$output" == *"Trust $(pwd -P)? [y/N]"* ]]
  [ -e "${BATS_TEST_TMPDIR}/ran" ]
}

@test "a line of a clone's path alone, as kit trust wrote before it kept the digest, trusts it no more; kit trust --yes writes the digest in its place" {
  printf '%s\n' "$(pwd -P)" > "${trusted}"
  run "${kit}" trust --check
  [ "$status" -eq 1 ]
  run "${kit}" kinds x
  [[ "$output" == *"asks kit to run something new since you trusted it"* ]]
  "${kit}" trust --yes
  recorded "$(pwd -P)"
  run "${kit}" trust --list
  [ "$output" = "$(pwd -P)" ]
}

@test "two worktrees that ask different things each keep a line of their own, and --revoke takes every line back" {
  git commit --quiet --allow-empty --message init
  git worktree add --quiet "${BATS_TEST_TMPDIR}/away" -b away
  printf 'gate: bin/ci\n' > "${BATS_TEST_TMPDIR}/away/kit.yml"
  "${kit}" trust --yes
  run bash -c 'cd "$1" && "$0" trust --check' "${kit}" "${BATS_TEST_TMPDIR}/away"
  [ "$status" -eq 1 ]
  run bash -c 'cd "$1" && "$0" trust --yes' "${kit}" "${BATS_TEST_TMPDIR}/away"
  [ "$status" -eq 0 ]
  [ "$(wc -l < "${trusted}")" -eq 2 ]
  run "${kit}" trust --check
  [ "$status" -eq 0 ]
  run bash -c 'cd "$1" && "$0" trust --check' "${kit}" "${BATS_TEST_TMPDIR}/away"
  [ "$status" -eq 0 ]
  run "${kit}" trust --list
  [ "$output" = "$(pwd -P)" ]
  run "${kit}" trust --revoke
  [ "$status" -eq 0 ]
  [ ! -s "${trusted}" ]
}

@test "kit trust --asks prints what the clone asks, its tools and every lint: command once, and asks nothing" {
  printf 'gate: bin/ci\ntools:\n  jq:\n    version: "1.8.2"\n  yq:\n    version: "4.54.1"\n' > kit.yml
  mkdir -p .kit/rules .kit/styles .kit/packages/local/files/repo/.kit/rules
  printf -- '---\nkind:\n  - bash\nlint: shellcheck --shell=bash\n---\n' > .kit/rules/bash.md
  printf -- '---\nkind:\n  - bash\nlint: shellcheck --shell=bash\n---\n' > .kit/packages/local/files/repo/.kit/rules/bash.md
  printf -- '---\nkind:\n  - text\n---\n' > .kit/styles/text.md
  run "${kit}" trust --asks </dev/null
  [ "$status" -eq 0 ]
  [ "${lines[0]}" = 'run bin/ci for kit gate (gate)' ]
  [ "${lines[1]}" = 'download and run the tools it pins, from the URLs it names: jq, yq (tools)' ]
  [ "${lines[2]}" = 'run shellcheck --shell=bash for kit lint (lint: in .kit/rules/bash.md)' ]
  [ "${#lines[@]}" -eq 3 ]
  [ ! -s "${trusted}" ]
  run "${kit}" trust --asks "${BATS_TEST_TMPDIR}/"
  [ "$status" -eq 2 ]
}

@test "in a clone you have not trusted, every git command a verb runs there ignores the clone's hooks and fsmonitor" {
  git commit --quiet --allow-empty --message init
  printf '#!/bin/sh\n: > "%s/fsmonitor"\n' "${BATS_TEST_TMPDIR}" > "${BATS_TEST_TMPDIR}/monitor"
  chmod +x "${BATS_TEST_TMPDIR}/monitor"
  git config core.fsmonitor "${BATS_TEST_TMPDIR}/monitor"
  touch file
  run bash -c 'source "$1/modules/bash/1st.sh" && kit:enter --any-clone && git config core.hooksPath &&
    git config protocol.allow && printf "%s\n" "${GIT_TERMINAL_PROMPT}" && git status --porcelain' _ "${kit_checkout}"
  [ "$status" -eq 0 ]
  [ "${lines[0]}" = /dev/null ]
  [ "${lines[1]}" = never ]
  [ "${lines[2]}" = 0 ]
  [ ! -e "${BATS_TEST_TMPDIR}/fsmonitor" ]
  "${kit}" trust --yes
  run bash -c 'source "$1/modules/bash/1st.sh" && kit:enter --any-clone && git config core.hooksPath' _ "${kit_checkout}"
  [ "$status" -eq 1 ]
}

@test "what kit places keeps the trust: an install that writes the version, a worktree that has none of it yet, a clean" {
  "${kit}" trust --yes
  run "${kit}" install agents/claude
  [ "$status" -eq 0 ]
  [ -f .kit/rules/bash.md ]
  run "${kit}" trust --check
  [ "$status" -eq 0 ]
  git add kit.yml && git commit --quiet --message init
  git worktree add --quiet "${BATS_TEST_TMPDIR}/away" -b away
  [ ! -e "${BATS_TEST_TMPDIR}/away/.kit/rules" ]
  run bash -c 'cd "$1" && "$0" trust --check' "${kit}" "${BATS_TEST_TMPDIR}/away"
  [ "$status" -eq 0 ]
  run bash -c 'cd "$1" && "$0" hydrate && "$0" trust --check' "${kit}" "${BATS_TEST_TMPDIR}/away"
  [ "$status" -eq 0 ]
  [ -f "${BATS_TEST_TMPDIR}/away/.kit/rules/bash.md" ]
  rm -rf .kit/rules
  run "${kit}" trust --check
  [ "$status" -eq 0 ]
  "${kit}" hydrate >/dev/null 2>&1
  # a rule a pull brings, with its write bit, as git checks it out, is the clone's
  printf -- '---\nkind:\n  - bash\nlint: touch %s/linted\n---\n' "${BATS_TEST_TMPDIR}" > .kit/rules/stray.md
  run "${kit}" trust --check
  [ "$status" -eq 1 ]
  rm .kit/rules/stray.md
  # and so is a link a pull brings, which a .kit/.gitignore of kit's own does not name
  ln -s ../../README.md .kit/rules/stray.md
  printf -- '---\nlint: touch x\n---\n' > README.md
  run "${kit}" trust --check
  [ "$status" -eq 1 ]
  # a link kit's own .kit/.gitignore names, without its write bit, is kit's; one with its write bit, as a pull brings,
  # names nothing kit placed
  chmod u+w .kit/.gitignore && printf '/rules/stray.md\n' >> .kit/.gitignore && chmod a-w .kit/.gitignore
  run "${kit}" trust --check
  [ "$status" -eq 0 ]
  chmod u+w .kit/.gitignore
  run "${kit}" trust --check
  [ "$status" -eq 1 ]
}

# ----------------------------------------------------------------------------------------------------------------------
