#!/usr/bin/env bats
# tui.bats - kit with no verb runs kit tui, which execs kittui; a verb short of its arguments hands off to it at a terminal
# ----------------------------------------------------------------------------------------------------------------------

setup() {
  export kit_root NO_COLOR=1
  kit_root=$(cd "${BATS_TEST_DIRNAME}/../kit" && pwd)
  repo=$(mktemp -d)
  cd "${repo}"
  git init --quiet --initial-branch=main
  : > .gitignore
  git add .gitignore
  # kittui is the fake, first on PATH
  mkdir -p "${BATS_TEST_TMPDIR}/fake"
  ln -s "${BATS_TEST_DIRNAME}/fakes/fake-kittui" "${BATS_TEST_TMPDIR}/fake/kittui"
  export PATH="${BATS_TEST_TMPDIR}/fake:${PATH}"
}

teardown() {
  rm -rf "${repo}"
}

# without-kittui prints a PATH with bash, git, and jq, the system's tools, and no kittui.
without-kittui() {
  local dir="${BATS_TEST_TMPDIR}/nokittui" tool
  mkdir -p "${dir}"
  for tool in bash git jq
  do
    ln -sf "$(command -v "${tool}")" "${dir}/${tool}"
  done
  printf '%s:/usr/bin:/bin\n' "${dir}"
}

# pty <command> runs the command in a pseudo-terminal, so its stdin, stdout, and stderr are terminals, as at a desk.
pty() {
  if script --version >/dev/null 2>&1
  then
    script --quiet --return --command "$1" /dev/null </dev/null
  else
    script -q /dev/null bash -c "$1" </dev/null
  fi
}

@test "kit with no verb runs kit tui, and kit tui execs kittui with the words after --" {
  run "${kit_root}/bin/kit"
  [ "$status" -eq 0 ]
  [ "$output" = "kittui" ]
  run "${kit_root}/bin/kit" tui -- commit --all
  [ "$status" -eq 0 ]
  [ "$output" = "kittui -- commit --all" ]
}

@test "a checkout with neither kittui nor go refuses with the line that installs go, which builds it" {
  run env PATH="$(without-kittui)" "${kit_root}/bin/kit" tui
  [ "$status" -eq 1 ]
  [[ "$output" == *"run: mise install (go is not on PATH)"* ]]
  run env PATH="$(without-kittui)" "${kit_root}/bin/kit"
  [ "$status" -eq 1 ]
  [[ "$output" == *"run: mise install (go is not on PATH)"* ]]
}

@test "a verb short of its arguments hands off to kit tui only when stdin, stdout, and stderr are terminals" {
  run pty "'${kit_root}/bin/kit' commit .gitignore"
  [[ "$output" == *"kittui -- commit .gitignore"* ]]
  run "${kit_root}/bin/kit" commit .gitignore </dev/null
  [ "$status" -eq 2 ]
  [[ "$output" == *"kit commit needs --message or --file; run: kit help commit"* ]]
  run pty "'${kit_root}/bin/kit' context | cat"
  [[ "$output" == *"kit context needs <path> or --claude; run: kit help context"* ]]
  [[ "$output" != *"kittui"* ]]
}

@test "a verb that has what it needs, or refuses a word, never hands off" {
  run pty "'${kit_root}/bin/kit' version"
  [[ "$output" != *"kittui"* ]]
  run pty "'${kit_root}/bin/kit' status --bogus"
  [[ "$output" == *"unknown flag '--bogus'; run: kit help status"* ]]
  [[ "$output" != *"kittui"* ]]
}

# platform prints this machine's platform as Go names it, as a release's libexec/ names its kittui.
platform() {
  local os arch
  os=$(uname -s | tr '[:upper:]' '[:lower:]')
  arch=$(uname -m)
  case "${arch}" in
    (x86_64 | amd64) arch=amd64 ;;
    (aarch64 | arm64) arch=arm64 ;;
  esac
  printf '%s-%s\n' "${os}" "${arch}"
}

@test "a release runs the kittui it carries for this machine, with no kittui on PATH" {
  export FAKES="${BATS_TEST_TMPDIR}/gofakes"
  mkdir -p "${FAKES}"
  ln -s "${BATS_TEST_DIRNAME}/fakes/fake-go" "${FAKES}/go"
  PATH="${FAKES}:${PATH}" "${BATS_TEST_DIRNAME}/../bin/release" pack release/9.9.9 "${BATS_TEST_TMPDIR}/rel" >/dev/null 2>&1
  tar -xzf "${BATS_TEST_TMPDIR}/rel/kit-9.9.9.tar.gz" -C "${BATS_TEST_TMPDIR}/rel"
  run env PATH="$(without-kittui)" "${BATS_TEST_TMPDIR}/rel/kit-9.9.9/bin/kit" tui -- commit
  [ "$status" -eq 0 ]
  [ "$output" = "kittui $(platform) -- commit" ]
}

@test "a checkout with no kittui builds it from tui/ with go, once, and runs it" {
  export FAKES="${BATS_TEST_TMPDIR}/gofakes" XDG_CACHE_HOME="${BATS_TEST_TMPDIR}/cache"
  mkdir -p "${FAKES}"
  ln -s "${BATS_TEST_DIRNAME}/fakes/fake-go" "${FAKES}/go"
  run env PATH="${FAKES}:$(without-kittui)" "${kit_root}/bin/kit" tui -- version
  [ "$status" -eq 0 ]
  [[ "$output" == *"kittui native-native -- version" ]]
  run env PATH="${FAKES}:$(without-kittui)" "${kit_root}/bin/kit" tui
  [ "$status" -eq 0 ]
  [ "$output" = "kittui native-native" ]
  [ "$(wc -l < "${FAKES}/go.calls" | tr -d ' ')" -eq 1 ]
}

# ----------------------------------------------------------------------------------------------------------------------
