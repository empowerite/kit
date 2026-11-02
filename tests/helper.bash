# shellcheck shell=bash
# helper.bash - what every test file loads: the checkout's kit, an environment the desk cannot reach, and shared helpers
# ----------------------------------------------------------------------------------------------------------------------
#
# A file loads this with `load helper`, after bats_require_minimum_version where it has one. Loading moves everything a
# test could leak under BATS_TEST_TMPDIR, which bats removes after the test: HOME and git's global configuration, the
# kit's home and its cache, and TMPDIR, so a mktemp in the kit or in a test lands there too; and it moves the test
# there, so a file whose setup() goes nowhere runs its verbs outside the checkout. It pins the locale, so a width
# measured with ${#text} counts characters, and turns color off; a test that asserts colors forces them on for its own
# command, with CLICOLOR_FORCE=1 or env -u NO_COLOR. kit_root and kit_checkout both name the checkout's kit/.
#
# The file's own setup() adds what its tests need. Its teardown is this file's: a test that ends inside the checkout
# fails, because a verb run there writes into the checkout's own .git/kit and .kit/. A file that defines teardown() of
# its own calls helper:outside-checkout from it.

kit_root=$(cd "${BATS_TEST_DIRNAME}/../kit" && pwd)
kit_checkout=${kit_root}
export kit_root kit_checkout

# helper:isolate moves what a test could leak under BATS_TEST_TMPDIR. bats sources a file once more for setup_file,
# before any test's directory exists; that pass changes nothing.
helper:isolate() {
  local go_cache go_env
  [[ -n "${BATS_TEST_TMPDIR:-}" ]] || return 0
  # The tools on PATH may be mise's shims, which find their installs under the desk's directories. Named before HOME
  # moves, so every tool resolves as it does at the desk.
  export MISE_DATA_DIR="${MISE_DATA_DIR:-${XDG_DATA_HOME:-${HOME}/.local/share}/mise}"
  export MISE_CONFIG_DIR="${MISE_CONFIG_DIR:-${XDG_CONFIG_HOME:-${HOME}/.config}/mise}"
  export MISE_CACHE_DIR="${MISE_CACHE_DIR:-${XDG_CACHE_HOME:-${HOME}/.cache}/mise}"
  export MISE_STATE_DIR="${MISE_STATE_DIR:-${XDG_STATE_HOME:-${HOME}/.local/state}/mise}"
  # go builds the TUI for the tests that reach it; its caches sit under the desk's HOME too, and a fresh one would
  # fetch every module again, build from nothing, and leave read-only directories bats cannot remove
  case "${OSTYPE}" in
    (darwin*)
      go_cache="${HOME}/Library/Caches/go-build"
      go_env="${HOME}/Library/Application Support/go/env"
      ;;
    (*)
      go_cache="${XDG_CACHE_HOME:-${HOME}/.cache}/go-build"
      go_env="${XDG_CONFIG_HOME:-${HOME}/.config}/go/env"
  esac
  export GOPATH="${GOPATH:-${HOME}/go}"
  export GOMODCACHE="${GOMODCACHE:-${GOPATH}/pkg/mod}" GOCACHE="${GOCACHE:-${go_cache}}" GOENV="${GOENV:-${go_env}}"
  export HOME="${BATS_TEST_TMPDIR}/home" XDG_CONFIG_HOME="${BATS_TEST_TMPDIR}/config"
  export KIT_HOME="${BATS_TEST_TMPDIR}/kit-home" XDG_CACHE_HOME="${BATS_TEST_TMPDIR}/cache"
  export TMPDIR="${BATS_TEST_TMPDIR}/tmp"
  mkdir -p "${HOME}" "${XDG_CONFIG_HOME}" "${KIT_HOME}" "${XDG_CACHE_HOME}" "${TMPDIR}"
  # git reads this file as its global configuration, and no system one: a desk's hooks path, signing, aliases, and
  # global ignores stay out, and every commit a test makes has an author
  export GIT_CONFIG_GLOBAL="${HOME}/.gitconfig" GIT_CONFIG_NOSYSTEM=1
  printf '[user]\n\tname = t\n\temail = t@example.com\n[commit]\n\tgpgsign = false\n[tag]\n\tgpgsign = false\n' \
    > "${GIT_CONFIG_GLOBAL}"
  printf '[init]\n\tdefaultBranch = main\n' >> "${GIT_CONFIG_GLOBAL}"
  export LC_ALL=C.UTF-8 NO_COLOR=1
  # what a CI runner's environment would hand the verbs that read it
  unset GITHUB_EVENT_PATH GITHUB_HEAD_REF CI_MERGE_REQUEST_IID CI_MERGE_REQUEST_SOURCE_BRANCH_NAME
  # the forge tools' login files, which name the hosts each forge serves: a test writes its own, under the new HOME
  unset GH_CONFIG_DIR GLAB_CONFIG_DIR
  # the marker of a toolchain the desk's shell activated, which would make that toolchain active in every test
  unset MISE_SHELL
  # a file whose setup() goes nowhere runs its verbs here, outside the checkout, rather than where bats was started
  cd "${BATS_TEST_TMPDIR}" || return 1
}

# helper:outside-checkout fails when the working directory is inside the checkout, naming where the test ended.
helper:outside-checkout() {
  local here checkout
  here=$(pwd -P 2>/dev/null || true)
  checkout=$(cd "${kit_root}/.." && pwd -P)
  if [[ "${here}" == "${checkout}" || "${here}" == "${checkout}/"* ]]
  then
    echo "the test ended in ${here}, inside the checkout; run the kit in a repository of its own: fresh-repo" >&2
    return 1
  fi
}

teardown() {
  helper:outside-checkout
}

# kit-bash <code> runs <code> in a fresh bash that has sourced the loader, as a verb does, with the modules
# ${kit_bash_modules} names loaded first; a file whose tests need one sets it in setup(), as kit_bash_modules=syntax.
kit-bash() {
  kit_bash_modules="${kit_bash_modules:-}" \
    bash -c 'source "${kit_root}/modules/bash/1st.sh" && use ${kit_bash_modules} && eval "$1"' kit-bash "$1"
}

# fresh-repo [<name>] makes an empty git repository, <name> or repo under BATS_TEST_TMPDIR, moves into it, sets repo
# to its path, and prints the path.
fresh-repo() {
  # shellcheck disable=SC2034  # the test files read repo
  repo="${BATS_TEST_TMPDIR}/${1:-repo}"
  git init --quiet --initial-branch=main "${repo}"
  cd "${repo}" || return 1
  printf '%s\n' "${repo}"
}

# checkout-copy makes a repository of its own from the checkout's tracked files and its .kit/, under BATS_TEST_TMPDIR,
# and moves into it. A test that checks this repository's own files runs the kit there, so whatever a verb compiles or
# logs lands in the copy's .git, never in the checkout's.
checkout-copy() {
  local root copy="${BATS_TEST_TMPDIR}/checkout"
  root=$(cd "${kit_root}/.." && pwd)
  mkdir -p "${copy}"
  (cd "${root}" && git ls-files -z | tar --null --create --files-from - --file -) | tar --extract --file - -C "${copy}"
  cp -R "${root}/.kit" "${copy}/"
  git init --quiet --initial-branch=main "${copy}" && git -C "${copy}" add --all
  cd "${copy}" || return 1
}

# shims <tool>… puts a failing stand-in for each tool first on a PATH it prints, so a test proves a path is not taken:
# run the command with PATH="$(shims mise curl)".
shims() {
  local tool dir="${BATS_TEST_TMPDIR}/shims"
  mkdir -p "${dir}"
  for tool in "$@"
  do
    printf '#!/usr/bin/env bash\nexit 1\n' > "${dir}/${tool}" && chmod +x "${dir}/${tool}"
  done
  printf '%s:%s\n' "${dir}" "${PATH}"
}

# fake-tools <tool>… stands tests/fakes/fake-tool in for each tool, first on PATH, with an empty route file each, and
# exports FAKE_DIR, where fixture and route write and the fake records its calls.
fake-tools() {
  local tool
  export FAKE_DIR="${BATS_TEST_TMPDIR}/fake"
  mkdir -p "${FAKE_DIR}/bin"
  for tool in "$@"
  do
    ln -sf "${BATS_TEST_DIRNAME}/fakes/fake-tool" "${FAKE_DIR}/bin/${tool}"
    : > "${FAKE_DIR}/${tool}.routes"
  done
  [[ ":${PATH}:" == *":${FAKE_DIR}/bin:"* ]] || export PATH="${FAKE_DIR}/bin:${PATH}"
}

# fixture <name> writes stdin to a response file the routes name.
fixture() { cat > "${FAKE_DIR}/$1"; }

# route [<tool>] <glob> <response> adds a route for <tool>, gh when none is named, the more particular before the more
# general: "<glob>\t<response>", as tests/fakes/fake-tool reads it.
route() {
  local tool=gh
  if (( $# == 3 ))
  then
    tool=$1
    shift
  fi
  printf '%s\t%s\n' "$1" "$2" >> "${FAKE_DIR}/${tool}.routes"
}

# declared <key> prints a top-level key of kit.yml: a scalar's value, or a list's items on one line.
declared() {
  awk -v key="$1" '
    $0 ~ "^" key ": " { v = $0; sub("^" key ": *", "", v); print v; exit }
    $0 == key ":" { on = 1; next }
    /^[A-Za-z]/ { on = 0 }
    on && /^  - / { sub(/^  - /, ""); printf "%s%s", (n++ ? " " : ""), $0 }
    END { if (n) print "" }' kit.yml 2>/dev/null
}

# later-release <dir> [<version>] copies this checkout's release to <dir> as <version>, 9.9.9 unless given, with one
# rule line added in that version.
later-release() {
  local version=${2:-9.9.9}
  mkdir -p "$1"
  cp -R "${kit_checkout}/bin" "${kit_checkout}/modules" "${kit_checkout}/packages" "$1/"
  printf '%s\n' "${version}" > "$1/version"
  printf '\n## Added in %s\n\nA new rule.\n' "${version}" >> "$1/packages/kit/files/repo/.kit/rules/json.md"
}

helper:isolate

# ----------------------------------------------------------------------------------------------------------------------
