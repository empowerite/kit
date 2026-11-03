# shellcheck shell=bash
# helper.bash - what every test file loads: the checkout's kit, an environment the desk cannot reach, and shared helpers
# ----------------------------------------------------------------------------------------------------------------------
# A file loads this with `load helper`, after bats_require_minimum_version where it has one. Loading moves everything a
# test could leak under BATS_TEST_TMPDIR, which bats removes after the test: HOME and git's global configuration, the
# kit's home and its cache, and TMPDIR, so a mktemp in the kit or in a test lands there too; and it moves the test
# there, so a file whose setup() goes nowhere runs its verbs outside the checkout. It pins the locale, so a width
# measured with ${#text} counts characters, and turns color off; a test that asserts colors forces them on for its own
# command, with CLICOLOR_FORCE=1 or env -u NO_COLOR. kit_root and kit_checkout both name the checkout's kit/.
#
# A file whose every test starts from an installed repository builds it once, in setup_file, with world-repo or
# world-make, and each test's setup() copies it with world-copy-repo or world-copy. A kit install takes seconds; a copy
# takes a few milliseconds.
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
  [[ -n "${BATS_TEST_TMPDIR:-}" ]] || return 0
  helper:isolate-in "${BATS_TEST_TMPDIR}"
}

# helper:isolate-in <dir> moves what a command could leak under <dir>, as helper:isolate does for a test, and moves
# there. world-make runs it in a subshell, to build a world before any test's directory exists.
helper:isolate-in() {
  local dir=$1 go_cache go_env
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
  export HOME="${dir}/home" XDG_CONFIG_HOME="${dir}/config"
  export KIT_HOME="${dir}/kit-home" XDG_CACHE_HOME="${dir}/cache"
  export TMPDIR="${dir}/tmp"
  mkdir -p "${HOME}" "${XDG_CONFIG_HOME}" "${KIT_HOME}" "${XDG_CACHE_HOME}" "${TMPDIR}"
  # git reads this file as its global configuration, and no system one: a desk's hooks path, signing, aliases, and
  # global ignores stay out, and every commit a test makes has an author
  export GIT_CONFIG_GLOBAL="${HOME}/.gitconfig" GIT_CONFIG_NOSYSTEM=1
  printf '[user]\n\tname = t\n\temail = t@example.com\n[commit]\n\tgpgsign = false\n[tag]\n\tgpgsign = false\n' \
    > "${GIT_CONFIG_GLOBAL}"
  printf '[init]\n\tdefaultBranch = main\n' >> "${GIT_CONFIG_GLOBAL}"
  export LC_ALL=C.UTF-8 NO_COLOR=1
  # one time zone on every desk and runner, so a date the kit writes reads the same in every test
  export TZ=UTC
  # what a CI runner's environment would hand the verbs that read it: the workflow commands an error adds, the step
  # summary, the toolchain the job names, and the pull request
  unset GITHUB_ACTIONS GITHUB_STEP_SUMMARY KIT_TOOLCHAIN
  unset GITHUB_EVENT_PATH GITHUB_HEAD_REF CI_MERGE_REQUEST_IID CI_MERGE_REQUEST_SOURCE_BRANCH_NAME
  # the depth of the kit call that ran bats, kit gate's under kit ci, which every call a test makes would add to
  unset kit_depth
  # the forge tools' login files, which name the hosts each forge serves: a test writes its own, under the new HOME
  unset GH_CONFIG_DIR GLAB_CONFIG_DIR
  # the marker of a toolchain the desk's shell activated, which would make that toolchain active in every test, and
  # mise's environment, the desk's or the runner's, which a test names for itself where it counts
  unset MISE_SHELL MISE_ENV
  # a file whose setup() goes nowhere runs its verbs here, outside the checkout, rather than where bats was started
  cd "${dir}" || return 1
}

# world-make <name> <function> builds a world once for the file, from its setup_file: <function> runs in an empty
# directory under BATS_FILE_TMPDIR, isolated there as a test is isolated in its own. A world costs one kit install per
# file, where setup() would cost one per test. It fails when the function fails.
world-make() {
  local dir=${BATS_FILE_TMPDIR}/world/$1
  mkdir -p "${dir}/root" "${dir}/env"
  # What the build prints is kept, and shown only when it fails.
  if ! ( helper:isolate-in "${dir}/env" && cd "${dir}/root" && "$2" ) > "${dir}/make.log" 2>&1
  then
    cat "${dir}/make.log" >&2
    return 1
  fi
}

# world-copy <name> [<dir>] copies what the world <name> holds into <dir>, the test's directory by default, so a
# repository the world made at repo lands where fresh-repo would make it. The kit's home the world made goes into the
# test's KIT_HOME, and its cache into the test's cache.
#
# A git remote is an absolute path, so each repository's config that names the world's directory names the copy
# instead. An installed repository does not name any other path of its own: the kit places copies under .kit/, and
# links only to sources inside the repository. A worktree records absolute paths too, so a test adds its worktrees
# after the copy.
world-copy() {
  local from=${BATS_FILE_TMPDIR}/world/$1 to=${2:-${BATS_TEST_TMPDIR}} config
  mkdir -p "${to}"
  cp -Rp "${from}/root/." "${to}/"
  [[ ! -d "${from}/env/kit-home" ]] || cp -Rp "${from}/env/kit-home/." "${KIT_HOME}/"
  # The cache holds what yq read from the packages, as a test's own install would have left it.
  [[ ! -d "${from}/env/cache" ]] || cp -Rp "${from}/env/cache/." "${XDG_CACHE_HOME}/"
  while IFS= read -r -d '' config
  do
    sed -e "s|${from}/root|${to}|g" "${config}" > "${config}.copy" && mv -f "${config}.copy" "${config}"
  done < <(find "${to}" -maxdepth 4 -path '*.git/config' -type f -print0)
}

# world-repo <name> [<package>…] is setup_file's most common world: a repository at repo with the kit installed and the
# packages named, or the defaults. world-copy-repo <name> copies it and moves into it, as fresh-repo would.
world-repo() {
  helper_world_packages=("${@:2}")
  world-make "$1" helper:world-install
}

# helper:world-install makes the repository world-repo describes, in the world's directory.
helper:world-install() {
  git init --quiet --initial-branch=main repo && cd repo \
    && "${kit_checkout}/bin/kit" install "${helper_world_packages[@]}"
}

world-copy-repo() {
  world-copy "$1" || return 1
  # shellcheck disable=SC2034  # the test files read repo
  repo=${BATS_TEST_TMPDIR}/repo
  cd "${repo}" || return 1
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

# hide-system-toolchains hides a real apt-get, dnf, and pacman from PATH, as a Linux runner has, so a test finds only
# the system toolchains it fakes. Each directory that holds one is replaced by links to everything else in it, made by
# one ln, as /usr/bin is too large to link one file at a time.
hide-system-toolchains() {
  local dir out='' copy n=0 tool IFS=:
  for dir in ${PATH}
  do
    for tool in apt-get dnf pacman
    do
      [[ -x "${dir}/${tool}" ]] || continue
      copy="${BATS_TEST_TMPDIR}/system-hidden-$((n++))"
      mkdir -p "${copy}"
      ln -s "${dir}"/* "${copy}/" 2>/dev/null || true
      rm -f "${copy}/apt-get" "${copy}/dnf" "${copy}/pacman"
      dir=${copy}
      break
    done
    out+="${out:+:}${dir}"
  done
  PATH=${out}
}

# path-without <tool> hides the tool from PATH, such as a real brew on this machine. Each directory that holds it is
# replaced by links to everything else in it, so the other tools there still run. One ln makes them, as
# hide-system-toolchains does: Homebrew's bin is too large to link one file at a time.
path-without() {
  local dir out='' copy n=0 IFS=:
  for dir in ${PATH}
  do
    if [[ -x "${dir}/$1" ]]
    then
      copy="${BATS_TEST_TMPDIR}/without-$1-$((n++))"
      mkdir -p "${copy}"
      ln -s "${dir}"/* "${copy}/" 2>/dev/null || true
      rm -f "${copy}/$1"
      dir=${copy}
    fi
    out+="${out:+:}${dir}"
  done
  PATH=${out}
}

# without-kittui prints a PATH with bash, git, jq, and yq, then the system's /usr/bin and /bin, and neither go nor
# kittui: with go, a checkout builds kittui, and GitHub's Ubuntu images link their go into /usr/bin. A system directory
# that holds either is replaced by links to everything else in it, made by one ln, as hide-system-toolchains does.
without-kittui() {
  local dir="${BATS_TEST_TMPDIR}/nokittui" system out='' copy n=0 tool
  mkdir -p "${dir}"
  for tool in bash git jq yq
  do
    ln -sf "$(command -v "${tool}")" "${dir}/${tool}"
  done
  for system in /usr/bin /bin
  do
    if [[ -x "${system}/go" || -x "${system}/kittui" ]]
    then
      copy="${BATS_TEST_TMPDIR}/nokittui-system-$((n++))"
      mkdir -p "${copy}"
      ln -s "${system}"/* "${copy}/" 2>/dev/null || true
      rm -f "${copy}/go" "${copy}/kittui"
      system=${copy}
    fi
    out+=":${system}"
  done
  printf '%s%s\n' "${dir}" "${out}"
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

# in-order <text> <word>… succeeds when the text holds each word, each after the one before, and otherwise names the
# first word it lacks. A test of what a picture holds checks its words this way, so a change to a glyph or a column's
# padding breaks only the test that pins that drawing.
in-order() {
  local text=$1 rest=$1 word
  shift
  for word in "$@"
  do
    if [[ "${rest}" != *"${word}"* ]]
    then
      printf "'%s' is not where it belongs in:\n%s\n" "${word}" "${text}" >&2
      return 1
    fi
    rest=${rest#*"${word}"}
  done
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
