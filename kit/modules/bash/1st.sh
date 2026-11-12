# shellcheck shell=bash
# 1st.sh - the kit's bash loader: the module path, messages, guards, and run (sourced module)
# ----------------------------------------------------------------------------------------------------------------------
# Every verb, bin/kit-<verb>, sources this file and then calls kit:enter. A verb runs the same through bin/kit and
# called directly. This file sets no shell options, so the caller's options stay in effect.
#
#   - A verb is an executable bin/kit-<verb>, with no extension.
#   - A module is a sourced <module>.sh in a directory on the module path, or below one: toolchain/brew is
#     toolchain/brew.sh. The verb or module that needs it loads it with `use <module>`, which sources every
#     <module>.sh on the path, in order, once per process.
#   - The module path is the bash_path setting in kit.yml. Its first entry is kit:modules/bash/, this
#     file's directory. A later file can replace a function an earlier one defined; super.sh lets it call the one
#     it replaced.
#   - Within the modules, work is a function call. Across a process boundary it is `run kit <verb>`, which runs
#     this release's kit and carries the trace depth.
#
# Every message goes to stderr, so stdout carries only data. Color is on when stderr is a terminal or CLICOLOR_FORCE
# is set to other than 0, and off when NO_COLOR is set, which wins. Under GitHub Actions, warn and error also print
# the matching workflow command (::warning::, ::error::).

if (( BASH_VERSINFO[0] < 4 || (BASH_VERSINFO[0] == 4 && BASH_VERSINFO[1] < 4) ))
then
  echo "kit needs bash 4.4 or later; this is ${BASH_VERSION}" >&2
  return 1
fi

# Sourcing the loader again in the same process is a no-op: a verb sources it after BASH_ENV already has, and the
# readonly declarations below would fail a second time. kit_modules is not exported, so a new process loads afresh.
# It is tested by declaration: before it is an associative array, kit_modules[1st] is read as an arithmetic index.
declare -p kit_modules &>/dev/null && return 0

# kit_root is the release: two directories above this file's, modules/bash/. kit_jq is the release's jq module
# directory, for jq -L "${kit_jq}".
kit_root=$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)
kit_jq=${kit_root}/modules/jq
export kit_root kit_jq
declare -Ag kit_modules=([1st]=1)
declare -ag kit_bash_path=()

# kit:enter [--any-clone] starts a verb's process: it turns POSIX mode off, which POSIXLY_CORRECT, or a SHELLOPTS
# exported by the sh the loader runs in, would turn on, as the kit's modules are bash, each function named
# <module>:<name>, which declare -F refuses in POSIX mode; it exports the directory the command was typed in as
# kit_invoked_in, for verbs that take relative paths, refuses a clone you have not trusted, as kit:trust-here says,
# and changes to the repository root, so every verb reads and writes the same paths wherever it was typed. A
# repository from a release that kept its settings in .kit/settings.yml has them moved to kit.yml first, before
# anything reads them. With --any-clone, a verb that runs nothing the clone names, as kit version and kit trust,
# runs in a clone you have not trusted too, and moves nothing there; every git command it runs there is hardened, as
# kit:harden-git says.
kit:enter() {
  set +o posix
  local any_clone=false
  [[ "${1:-}" != --any-clone ]] || any_clone=true
  kit_invoked_in=$(pwd -P)
  export kit_invoked_in
  if [[ "${any_clone}" == false ]]
  then
    kit:trust-here || exit 1
  elif ! kit:trusted-at .
  then
    kit:harden-git
  fi
  local repository
  if repository=$(git rev-parse --show-toplevel 2>/dev/null)
  then
    cd "${repository}" || return 1
  fi
  [[ "${any_clone}" == true ]] || kit:move-settings
}

# kit:clone-root <dir> prints the root of the clone <dir> is in, by its physical path, as bin/kit, the loader, finds
# it: the nearest directory at or above it that holds .git, a directory in a desk and a file in a worktree. It fails
# outside a clone. It reads the file system alone and runs no git, so nothing a clone configures runs before it is
# trusted.
kit:clone-root() {
  local dir
  dir=$(cd "$1" 2>/dev/null && pwd -P) || return 1
  until [[ -e "${dir}/.git" ]]
  do
    [[ -n "${dir}" ]] || return 1
    dir=${dir%/*}
  done
  printf '%s\n' "${dir:-/}"
}

# kit:trust-name <clone-root> prints the name a clone is trusted by: its root, and for a worktree, its desk, but only
# where the desk confirms the worktree, as bin/kit, the loader, reads it. The worktree's .git file names an admin
# directory, which must be in the .git/worktrees/ of the desk its commondir file names, and whose gitdir file must
# name this .git back by its absolute path; a .git file of any other kind, as a submodule's or a crafted one, leaves
# the root as the name, so a clone cannot borrow the trust of a repository it only names. It runs no git, as
# kit:clone-root does not.
kit:trust-name() {
  local name=$1 line='' admin back='' common='' dir
  if [[ -f "$1/.git" ]] && { IFS= read -r line < "$1/.git" || [[ -n "${line}" ]]; } && [[ "${line}" == 'gitdir: '* ]]
  then
    admin=${line#gitdir: }
    [[ "${admin}" == /* ]] || admin=$1/${admin}
    if { IFS= read -r back < "${admin}/gitdir" || [[ -n "${back}" ]]; } 2>/dev/null \
      && { IFS= read -r common < "${admin}/commondir" || [[ -n "${common}" ]]; } 2>/dev/null \
      && [[ "${back}" == /*/.git ]] && dir=$(cd "${back%/.git}" 2>/dev/null && pwd -P) && [[ "${dir}" == "$1" ]]
    then
      [[ "${common}" == /* ]] || common=${admin}/${common}
      # The admin directory must be one the desk made, in its own .git/worktrees/: one the clone made elsewhere could
      # name any desk's .git as its common dir.
      if common=$(cd "${common}" 2>/dev/null && pwd -P) && admin=$(cd "${admin}" 2>/dev/null && pwd -P) \
        && [[ "${admin%/*}" == "${common}/worktrees" ]]
      then
        name=${common%/.git}
      fi
    fi
  fi
  printf '%s\n' "${name}"
}

# kit:trust-file prints the file that records the clones you trust: kit/trusted under XDG_CONFIG_HOME, else
# ~/.config. modules/bash/trust.sh says what a line holds.
kit:trust-file() {
  printf '%s\n' "${XDG_CONFIG_HOME:-${HOME}/.config}/kit/trusted"
}

# kit_trust_text is the awk program that prints what a clone asks kit to run, the text whose sha256 a trust record
# holds, from the files kit:trust-digest hands it: kit.yml but its keys that name no command (answers, excludes,
# facts, flow, forge, forge_settings, forges, linked_exempt, log_days, packages, policy, primary, require_signing,
# trackers), so a line kit cannot place under one of those is in it, and a key written in quotes too; .kit/manifest's
# version line, which the loader reads where kit.yml has none; and the frontmatter of each rule and style, where a
# lint: command is, read as modules/bash/yaml.sh reads one, or the whole file where its first line is not ---. A
# file's text starts with a line that names it, where it has any. bin/kit, the loader, holds the same program, and
# every release keeps it, as any installed kit checks the records any release wrote.
# shellcheck disable=SC2016  # the program is awk's, whose $0 and fields are its own, not the shell's
kit_trust_text='
BEGIN {
  data = "^(answers|excludes|facts|flow|forge|forge_settings|forges|linked_exempt|log_days|packages|policy|primary|"
  data = data "require_signing|trackers)[[:space:]]*:"
}
function emit() {
  if (!named) printf "== %s\n", FILENAME
  named = 1
  print
}
FNR == 1 {
  named = 0
  mode = "whole"
  keep = 1
  first = $0
  sub(/\r$/, "", first)
  if (FILENAME == "kit.yml") mode = "settings"
  else if (FILENAME == ".kit/manifest") mode = "manifest"
  else if (first == "---") { mode = "front"; next }
}
mode == "settings" {
  if ($0 ~ /^[^[:space:]#-]/)
    keep = $0 !~ data
  if (keep) emit()
  next
}
mode == "manifest" { if ($0 ~ /^version /) emit(); next }
mode == "front" { if ($0 == "---") mode = "done"; else emit(); next }
mode == "whole" { emit() }
'

# kit:trust-digest <clone> prints the sha256 of what the clone at <clone> asks kit to run: kit_trust_text over its
# kit.yml, its .kit/manifest, its own packages' rules and styles, and each rule and style under .kit/ that a person or
# a pull put there, in byte order, as bin/kit, the loader, digests it. What kit placed under .kit/rules/ and
# .kit/styles/ is left out: a copy without its write bit, and a link kit's own .kit/.gitignore names, one without its
# write bit, as kit writes it. Those come from the release and the packages, whose names the digest holds, and are
# there in one worktree and not yet in a new one; a file a pull brings has its write bit, as git checks every file out
# with it, and a pull that brings a .kit/.gitignore brings one with its write bit, which names nothing here. It runs
# find, awk, and sha256sum, or shasum, and nothing the clone names.
kit:trust-digest() {
  local line
  line=$(
    LC_ALL=C
    cd "$1" 2>/dev/null || exit 1
    files=()
    for file in kit.yml .kit/manifest .kit/packages/*/files/repo/.kit/rules/*.md \
      .kit/packages/*/files/repo/.kit/styles/*.md .kit/packages/*/*/files/repo/.kit/rules/*.md \
      .kit/packages/*/*/files/repo/.kit/styles/*.md
    do
      [[ ! -f "${file}" ]] || files+=("${file}")
    done
    # a directory that is not there, as .kit/styles/ in a clone with no style, lists nothing, and find says so
    writable=$'\n'$(find .kit/.gitignore .kit/rules .kit/styles -maxdepth 1 \( -name '*.md' -o -name .gitignore \) \
      -type f -perm -0200 2>/dev/null || true)$'\n'
    ignore=''
    if [[ -f .kit/.gitignore && ! -L .kit/.gitignore && "${writable}" != *$'\n'.kit/.gitignore$'\n'* ]]
    then
      IFS= read -r -d '' ignore < .kit/.gitignore || true  # read meets the end before any NUL, so it returns 1
      ignore=$'\n'${ignore}
    fi
    for file in .kit/rules/*.md .kit/styles/*.md
    do
      if [[ -L "${file}" ]]
      then
        [[ "${ignore}" != *$'\n'/${file#.kit/}$'\n'* && -f "${file}" ]] || continue
      else
        [[ "${writable}" == *$'\n'${file}$'\n'* ]] || continue
      fi
      files+=("${file}")
    done
    { (( ${#files[@]} == 0 )) || awk "${kit_trust_text}" "${files[@]}"; } \
      | if hash sha256sum 2>/dev/null; then sha256sum; else shasum -a 256; fi
  ) || return 1
  printf '%s\n' "${line%% *}"
}

# kit:trusted <name> [<clone>] succeeds when the trust file trusts <name>: a folder above it, a line that ends in /,
# or a line for <name> itself that holds the digest of what <clone> asks kit to run now, kit:trust-digest's. It
# returns 1 when no line names <name> or a folder above it, and 3 when a line names <name> with another digest, or
# none, as kit trust wrote before it kept one: what the clone asks has changed since you trusted it. Without <clone>,
# a folder alone trusts it. A line that starts with # is a comment, and ~/ at the start of a path is the home
# directory, by its physical path. The digest is worked out once, and only when a line names <name>.
kit:trusted() {
  local file line want have='' status=1
  file=$(kit:trust-file)
  [[ -f "${file}" ]] || return 1
  while IFS= read -r line || [[ -n "${line}" ]]
  do
    [[ -n "${line}" && "${line}" != '#'* ]] || continue
    want=''
    if [[ "${line}" =~ ^[0-9a-f]{64}\ \ ([/~].*)$ ]]
    then
      want=${line%%  *}
      line=${BASH_REMATCH[1]}
    fi
    # The home directory by its physical path, as every name is compared: on macOS, /var is a link to /private/var.
    [[ "${line}" != \~/* ]] || line=$(cd "${HOME}" 2>/dev/null && pwd -P || printf '%s' "${HOME}")/${line#\~/}
    if [[ "${line}" == */ ]]
    then
      [[ -z "${want}" && "$1/" == "${line}"* ]] && return 0
    elif [[ "$1" == "${line}" ]]
    then
      status=3
      [[ -n "${want}" && -n "${2:-}" ]] || continue
      [[ -n "${have}" ]] || have=$(kit:trust-digest "$2") || have=none
      [[ "${want}" != "${have}" ]] || return 0
    fi
  done < "${file}"
  return "${status}"
}

# kit_trusted_at holds, by clone root, what kit:trusted said of the clone in this process, so a verb asks once.
declare -gA kit_trusted_at=()

# kit:trusted-at <dir> succeeds where kit may run what a clone asks for: outside a clone, and in a clone you trust,
# as it asks now. It returns kit:trusted's 1 or 3 otherwise.
kit:trusted-at() {
  local root status=0
  root=$(kit:clone-root "$1") || return 0
  if [[ -z "${kit_trusted_at[${root}]+x}" ]]
  then
    kit:trusted "$(kit:trust-name "${root}")" "${root}" || status=$?
    kit_trusted_at[${root}]=${status}
  fi
  return "${kit_trusted_at[${root}]}"
}

# kit:trust-here succeeds where kit:trusted-at the working directory does. Elsewhere it says how to trust the clone,
# in the words bin/kit uses, and fails.
kit:trust-here() {
  local status=0 name
  kit:trusted-at . || status=$?
  (( status != 0 )) || return 0
  name=$(kit:trust-name "$(kit:clone-root .)")
  if (( status == 3 ))
  then
    error "${name} asks kit to run something new since you trusted it; see what it asks, and trust it again: kit trust"
  else
    error "kit runs nothing in ${name} until you trust it; see what it asks kit to run, and trust it: kit trust"
  fi
  return 1
}

# kit:harden-git makes every git command this process and its children run ignore what a clone's own git config
# could make git run: no hooks (core.hooksPath=/dev/null), no fsmonitor (core.fsmonitor=false), no transport at all
# (protocol.allow=never), and no prompt for a password (GIT_TERMINAL_PROMPT=0). It sets them through
# GIT_CONFIG_COUNT, after any the environment holds, as git -c would, so they win over every config file. kit:enter
# calls it in a clone you have not trusted, where kit help, kit version, and kit trust run.
kit:harden-git() {
  local i=${GIT_CONFIG_COUNT:-0} pair
  [[ "${i}" =~ ^[0-9]+$ ]] || i=0
  for pair in core.hooksPath=/dev/null core.fsmonitor=false protocol.allow=never
  do
    export "GIT_CONFIG_KEY_${i}=${pair%%=*}" "GIT_CONFIG_VALUE_${i}=${pair#*=}"
    i=$(( i + 1 ))
  done
  export GIT_CONFIG_COUNT=${i} GIT_TERMINAL_PROMPT=0
}

# kit:move-settings moves .kit/settings.yml to kit.yml at the root, where a release before kit.yml left it, and
# points its schema modeline at .kit/kit.schema.json. Where kit.yml is there already, it leaves both alone.
kit:move-settings() {
  [[ -f .kit/settings.yml && ! -e kit.yml ]] || return 0
  # shellcheck disable=SC2016  # $schema is the modeline's own text, not the shell's
  sed 's#\$schema=settings\.schema\.json#$schema=.kit/kit.schema.json#' .kit/settings.yml > kit.yml || return 1
  rm -f .kit/settings.yml
  step "kit.yml: the settings moved here from .kit/settings.yml; commit both paths"
}

# use:yq exports kit_yq, the path of the yq on PATH, once it has seen that yq is mikefarah's, the one yaml.sh reads
# through. yaml.sh trusts kit_yq while the yq on PATH is still that one, so neither its load in the loader's subshell,
# nor its load in the main shell, nor any load in a child process starts yq --version again. bash's own command hash
# finds the yq on PATH, so looking costs no process. A yq that is not mikefarah's leaves kit_yq as it was, for yaml.sh
# to say so.
use:yq() {
  local yq=''
  hash yq 2>/dev/null && yq=${BASH_CMDS[yq]:-}
  [[ -n "${yq}" && "${kit_yq:-}" != "${yq}" ]] || return 0
  [[ "$("${yq}" --version 2>/dev/null)" == *mikefarah* ]] || return 0
  kit_yq=${yq}
  export kit_yq
}

# use:load-path fills kit_bash_path with the module path's directories, once per process: each entry of the bash_path
# setting, else kit:modules/bash/ alone. An entry is kit:<dir> (in the release), an absolute directory, ~/<dir> (in
# the home directory), <source>:<dir> (in a source the settings name), or a directory relative to the repository root.
# A directory that does not exist is skipped, as PATH skips one. Away from the root, it reads the root's settings. It
# returns 2 on a path that does not start with kit:modules/bash/, a root other than kit: and the sources, or a kit.yml
# yq cannot read.
#
# Reading bash_path starts yq, so the entries it found are exported for child processes: kit_bash_path_dirs, their
# directories, one per line, and kit_bash_path_from, all they were resolved from: the release, the home directory, the
# cache directory, the root, and the text of the kit.yml there. A child for which all of them are the same uses the
# directories as they are, and reads kit.yml again otherwise, so an edit to it, another repository, or another release
# is never read stale. Whether each directory exists is asked again every time.
use:load-path() {
  (( ${#kit_bash_path[@]} )) && return 0
  local entry dir root digest settings='' from status=0
  local -a entries=() dirs=()
  # A verb runs at the repository root, where kit.yml is. A just recipe, or any script that sources this
  # file, may run in a directory below it, so the settings and the relative entries are read from the root then.
  root=$(pwd -P)
  if [[ ! -f kit.yml ]]
  then
    root=$(git rev-parse --show-toplevel 2>/dev/null || pwd -P)
  fi
  if [[ -f "${root}/kit.yml" ]]
  then
    # read, a builtin, takes the file's text with no process; at the end of a file with no NUL it returns 1
    IFS= read -r -d '' settings < "${root}/kit.yml" || true
  fi
  printf -v from '%s\n' "${kit_root}" "${HOME}" "${XDG_CACHE_HOME:-}" "${XDG_CONFIG_HOME:-}" "${root}" "${settings}"
  if [[ -n "${kit_bash_path_dirs:-}" && "${kit_bash_path_from:-}" == "${from}" ]]
  then
    readarray -t dirs <<<"${kit_bash_path_dirs}"
    use:keep-path "${dirs[@]}"
    return 0
  fi
  use:yq
  # Only a kit.yml whose text names bash_path is read with yq: one that does not cannot set it, and a kit run without
  # yq goes on to say yq is missing. yaml.sh says nothing in a subshell, so a yq that is missing is named here; a file
  # it cannot parse has had its line.
  # A clone you have not trusted names no bash: the verbs that run there, kit help among them, load the release's
  # own modules alone.
  if [[ "${settings}" == *bash_path* ]] && kit:trusted-at "${root}"
  then
    list:read entries use:path-setting "${root}" || status=$?
    if (( status == 2 ))
    then
      guard:missing yq "bash_path in kit.yml is read through yq 4, mikefarah/yq"
    fi
    (( status == 0 )) || return 2
  fi
  (( ${#entries[@]} )) || entries=(kit:modules/bash/)
  if [[ "${entries[0]}" != kit:modules/bash/ ]]
  then
    error "bash_path must start with kit:modules/bash/, the kit's own modules (bash_path in kit.yml)"
    return 2
  fi
  for entry in "${entries[@]}"
  do
    case "${entry}" in
      (kit:*) dir=${kit_root}/${entry#kit:} ;;
      (/*) dir=${entry} ;;
      (\~ | \~/*) dir=${HOME}${entry#\~} ;;
      ([A-Za-z0-9_-]*:*)
        # <source>:<dir> is a directory in a source kit.yml names, in the cache under its digest.
        digest=$(source "${kit_root}/modules/bash/yaml.sh" && yaml:records "${root}/kit.yml" sources name digest \
          | awk -F '\037' -v n="${entry%%:*}" '$1 == n { print $2; exit }')
        if [[ -z "${digest}" ]]
        then
          error "bash_path entry '${entry}' names a root the kit does not know; the roots are kit: and the sources" \
            "(bash_path and sources in kit.yml)"
          return 2
        fi
        dir=${XDG_CACHE_HOME:-${HOME}/.cache}/kit/sources/${digest#sha256:}/${entry#*:}
        ;;
      (*) dir=${root}/${entry}
    esac
    dirs+=("${dir}")
  done
  kit_bash_path_from=${from}
  printf -v kit_bash_path_dirs '%s\n' "${dirs[@]}"
  export kit_bash_path_from kit_bash_path_dirs
  use:keep-path "${dirs[@]}"
}

# use:path-setting <root> prints the entries of the bash_path setting in <root>/kit.yml, one per line. It fails, after
# yq's line, on a kit.yml yq cannot read. yaml.sh is read in a subshell, so this does not count as loading it.
use:path-setting() {
  # shellcheck source=SCRIPTDIR/yaml.sh
  (source "${kit_root}/modules/bash/yaml.sh" && yaml:list "$1/kit.yml" bash_path)
}

# use:keep-path <dir>… fills kit_bash_path with each directory that exists, by its physical path.
use:keep-path() {
  local dir
  for dir in "$@"
  do
    [[ -d "${dir}" ]] && kit_bash_path+=("$(cd "${dir}" && pwd -P)")
  done
  return 0
}

# use:path prints the module path's directories, one per line, in order: the entries of bash_path that exist.
use:path() {
  use:load-path || return 2
  printf '%s\n' "${kit_bash_path[@]}"
}

# use:find <module> [<array>] prints the path of every <module>.sh on the module path, in order, or, with <array>,
# fills that array with them, in this shell, so use starts no subshell for each module it loads. Call use:load-path
# first.
use:find() {
  local use_find_dir
  local -a use_find_files=()
  for use_find_dir in "${kit_bash_path[@]}"
  do
    [[ ! -f "${use_find_dir}/$1.sh" ]] || use_find_files+=("${use_find_dir}/$1.sh")
  done
  if [[ -n "${2:-}" ]]
  then
    local -n use_find_into=$2
    # shellcheck disable=SC2034  # the caller's array, through the name
    use_find_into=("${use_find_files[@]}")
    return 0
  fi
  for use_find_dir in "${use_find_files[@]}"
  do
    printf '%s\n' "${use_find_dir}"
  done
}

# use:modules <pattern> prints the name of every module on the module path that matches the glob <pattern>, once
# each, sorted, named from its directory on the path: use:modules 'forge/*' lists the forge providers, forge/github
# among them.
use:modules() {
  use:load-path || return 2
  local dir file name
  for dir in "${kit_bash_path[@]}"
  do
    for file in "${dir}"/$1.sh
    do
      [[ -f "${file}" ]] || continue
      name=${file#"${dir}"/}
      printf '%s\n' "${name%.sh}"
    done
  done | LC_ALL=C sort --unique
}

# use <module>… sources every <module>.sh on the module path, in order, once per module. An unknown module is a bug
# in the caller: use prints an error and returns 2. So is a name that could reach outside the module path: one that
# starts with /, or has an empty, ., or .. part.
use() {
  local module file
  local -a files
  use:load-path || return 2
  for module in "$@"
  do
    if [[ "${module}" == /* || "/${module}/" == *//* || "/${module}/" == */./* || "/${module}/" == */../* ]]
    then
      error "no such kit bash module: ${module} (a module's name has no empty, ., or .. part, and no leading /)"
      return 2
    fi
    [[ ${kit_modules[${module}]+x} ]] && continue
    use:find "${module}" files
    if (( ${#files[@]} == 0 ))
    then
      error "no such kit bash module: ${module} (looked in ${kit_bash_path[*]})"
      return 2
    fi
    kit_modules[${module}]=1
    for file in "${files[@]}"
    do
      # shellcheck disable=SC1090  # the path is computed from the module name
      source "${file}"
    done
  done
}

# run <command> <argument>… runs a command across a process boundary. kit and kit-<verb> name this release's own
# commands, not whatever is on PATH. The command runs with kit_depth one deeper than this process's, so a trace can
# nest it. It returns the command's status, or 2 for a kit-<verb> this release does not have.
run() {
  local command=$1
  shift
  case "${command}" in
    (kit | kit-*)
      command=${kit_root}/bin/${command}
      if [[ ! -x "${command}" ]]
      then
        error "no such kit command: ${command##*/}; run: kit help"
        return 2
      fi
  esac
  kit_depth=$(( ${kit_depth:-0} + 1 )) "${command}" "$@"
}

if [[ -n "${NO_COLOR:-}" ]]
then
  declare -gr out_color=0
elif [[ ( -n "${CLICOLOR_FORCE:-}" && "${CLICOLOR_FORCE}" != 0 ) || -t 2 ]]
then
  declare -gr out_color=1
else
  declare -gr out_color=0
fi

# list:among <name> <name>… succeeds when the first name is one of the rest.
list:among() {
  local needle=$1 item
  shift
  for item in "$@"
  do
    [[ "${item}" == "${needle}" ]] && return 0
  done
  return 1
}

# list:without <name> <name>… prints the rest without the first name, one per line.
list:without() {
  local needle=$1 item
  shift
  for item in "$@"
  do
    [[ "${item}" == "${needle}" ]] || printf '%s\n' "${item}"
  done
}

# list:read <array> <command> [<argument>…] empties the named array, runs the command, and fills the array with its
# lines, a trailing empty line dropped. It returns the command's status, the array left empty when that is not 0.
# stdin reaches the command. A read through `< <(…)` alone loses that status.
list:read() {
  list:fill $'\n' "$@"
}

# list:read0 <array> <command> [<argument>…] does what list:read does, for items that each end in a NUL.
list:read0() {
  list:fill '' "$@"
}

# list:fill <delimiter> <array> <command> [<argument>…] is list:read's work, for items that end in <delimiter>. The
# command runs in a process substitution, as in `< <(…)`, under errexit whoever calls it, so a failure inside it ends
# it there. A command that returns 0 has its status travel in band: the subshell prints a mark unique to the call and
# the status as one last item, and the mark is taken off the list before the trailing empty item is. A command that
# fails, exits, or execs prints no mark, and its status is wait "$!"'s, which under load can lose it: 127, still a
# failure. Where wait answers 0, output written after the command ended, as by an EXIT trap, can hold the mark, and
# the read fails. lastpipe, with PIPESTATUS, is not used: bash can hang in a lastpipe pipeline under load. It returns 1
# where readarray fails, and 2, after a line that says so, for a name that is not an identifier or starts list_fill_.
list:fill() {
  local list_fill_delimiter=$1 list_fill_status=0 list_fill_format='\n%s\n' list_fill_mark list_fill_item
  if [[ ! "$2" =~ ^[A-Za-z_][A-Za-z0-9_]*$ || "$2" == list_fill_* ]]
  then
    error "list:read: '$2' is not a name of an array it may fill: an identifier, not list_fill_…"
    return 2
  fi
  local -n list_fill_items=$2
  shift 2
  [[ -n "${list_fill_delimiter}" ]] || list_fill_format='\0%s\0'
  list_fill_mark="list:fill ${BASHPID} ${RANDOM}${RANDOM} "
  list_fill_items=()
  # shellcheck disable=SC2059  # the format is one of the two above, chosen by the delimiter
  if ! readarray -t -d "${list_fill_delimiter}" list_fill_items < <(set -o errexit; "$@"; list_fill_status=$?
    printf "${list_fill_format}" "${list_fill_mark}${list_fill_status}")
  then
    list_fill_items=()
    return 1
  fi
  if (( ${#list_fill_items[@]} )) && [[ "${list_fill_items[-1]}" == "${list_fill_mark}"* ]]
  then
    list_fill_status=${list_fill_items[-1]#"${list_fill_mark}"}
    [[ "${list_fill_status}" =~ ^[0-9]+$ ]] || list_fill_status=1
    unset 'list_fill_items[-1]'
    # The delimiter printed in front of the mark ends the command's last item, or, after one that ended, adds an
    # empty one, which is not the command's.
    if (( ${#list_fill_items[@]} )) && [[ -z "${list_fill_items[-1]}" ]]
    then
      unset 'list_fill_items[-1]'
    fi
  else
    wait "$!" 2>/dev/null || list_fill_status=$?
    if (( list_fill_status == 0 ))
    then
      for list_fill_item in "${list_fill_items[@]}"
      do
        if [[ "${list_fill_item}" =~ ^"${list_fill_mark}"[0-9]+$ ]]
        then
          list_fill_status=1
          break
        fi
      done
    fi
  fi
  if (( list_fill_status != 0 ))
  then
    list_fill_items=()
    return "${list_fill_status}"
  fi
  if (( ${#list_fill_items[@]} )) && [[ -z "${list_fill_items[-1]}" ]]
  then
    unset 'list_fill_items[-1]'
  fi
  return 0
}

# ere:quote <text> prints the text with every extended-regular-expression metacharacter escaped, so it matches
# literally in a pattern. It is pure bash, so BSD and GNU sed differences in bracket expressions do not matter.
ere:quote() {
  local text=$1 out='' c i
  for (( i = 0; i < ${#text}; i++ ))
  do
    c=${text:i:1}
    case "${c}" in
      ('.' | '[' | ']' | '*' | '^' | '$' | "\\" | '+' | '?' | '(' | ')' | '{' | '}' | '|') out+="\\${c}" ;;
      (*) out+=${c} ;;
    esac
  done
  printf '%s\n' "${out}"
}

# out:sgr <code> prints the escape sequence for one SGR code when color is on, and nothing otherwise.
out:sgr() {
  (( out_color )) && printf '\x1b[%sm' "$1"
  return 0
}

# out:emit <icon> <sgr> <message>… prints one line to stderr, painted as out:sgr paints, and forks nothing.
out:emit() {
  local icon=$1 on='' off=''
  (( out_color )) && on=$'\x1b['$2m off=$'\x1b[0m'
  shift 2
  printf '%s%s %s%s\n' "${on}" "${icon}" "$*" "${off}" >&2
}

info()    { out:emit '•' '' "$@"; }
step()    { out:emit '→' '36' "$@"; }
ok()      { out:emit '✅' '32' "$@"; }
section() { out:emit '' '1' "── $* ──"; }

# warn and error print a message whose remedy, the command after run:, runs from anywhere, as out:remedy writes it.
warn() {
  local message
  message=$(out:remedy "$*")
  out:emit '⚠️' '33' "${message}"
  [[ -n "${GITHUB_ACTIONS:-}" ]] && echo "::warning::${message}" >&2
  return 0
}

error() {
  local message
  message=$(out:remedy "$*")
  out:emit '❌' '31' "${message}"
  [[ -n "${GITHUB_ACTIONS:-}" ]] && echo "::error::${message}" >&2
  return 0
}

# out:remedy <message> prints the message with each remedy in it, the command after run:, made to run from anywhere:
# cd, the folder kit runs in, as out:folder writes it, and && in front of the command, as in run: cd ~/src/app && kit
# hydrate. A remedy kit wrote a cd for itself keeps it. Pasted into a shell in another folder, or run by kittui in
# the folder it names, the remedy does what it says where it should.
out:remedy() {
  local message=$1 out='' rest folder
  [[ "${message}" == *'run: '* ]] || { printf '%s\n' "${message}"; return 0; }
  folder=$(out:folder "${PWD}")
  while [[ "${message}" == *'run: '* ]]
  do
    out+=${message%%'run: '*}'run: '
    rest=${message#*'run: '}
    [[ "${rest}" == 'cd '* ]] || out+="cd ${folder} && "
    message=${rest}
  done
  printf '%s\n' "${out}${message}"
}

# out:folder <path> prints a folder as a shell reads it back: ~ for the home folder, and ~/ before a folder under it;
# the rest single-quoted where it holds more than letters, digits, and . _ - / + @ : ,.
out:folder() {
  local path=$1 lead=''
  if [[ -n "${HOME:-}" && "${path}" == "${HOME}" ]]
  then
    printf '~\n'
    return 0
  elif [[ -n "${HOME:-}" && "${path}" == "${HOME}"/* ]]
  then
    # shellcheck disable=SC2088  # the ~ is written for the person's shell to expand, not this one
    lead='~/'
    path=${path#"${HOME}"/}
  fi
  if [[ "${path}" =~ ^[A-Za-z0-9._/+@:,-]+$ ]]
  then
    printf '%s%s\n' "${lead}" "${path}"
  else
    printf "%s'%s'\n" "${lead}" "${path//\'/\'\\\'\'}"
  fi
}

# summary <line>… appends lines to the GitHub Actions step summary when GITHUB_STEP_SUMMARY is set, and does
# nothing otherwise.
summary() {
  [[ -n "${GITHUB_STEP_SUMMARY:-}" ]] || return 0
  printf '%s\n' "$@" >> "${GITHUB_STEP_SUMMARY}"
}

# guard:require <predicate>… checks each predicate in order. At the first one that fails, it prints the remedy and
# returns non-zero. The predicates:
#   signing-key  git user.signingkey is set
#   tool:<name>  <name> is on PATH
guard:require() {
  local predicate
  for predicate in "$@"
  do
    case "${predicate}" in
      (signing-key)
        if [[ -z "$(git config user.signingkey 2>/dev/null)" ]]
        then
          error "run: git config user.signingkey <key> (every commit here is signed, and none is configured)"
          return 1
        fi
        ;;
      (tool:*)
        local tool=${predicate#tool:}
        if ! command -v "${tool}" >/dev/null 2>&1
        then
          guard:missing "${tool}" "${tool} is not on PATH"
          return 1
        fi
        ;;
      (*)
        error "guard:require: unknown predicate '${predicate}'"
        return 1
    esac
  done
}

# guard:missing <tool> <reason> prints, in one line, the remedy for a tool kit cannot run, then the reason in
# parentheses. Every message about a missing or unusable tool goes through it, so the same condition gives the same
# remedy everywhere.
guard:missing() {
  error "$(guard:remedy "$1") ($2)"
}

# guard:remedy <tool> prints what installs the tool. bash, git, jq, and yq, the tools kit runs on, are the system
# toolchain's to install, so theirs is that toolchain's command, as run: brew install jq; where kit can name no
# command, it is to install the tool with the machine's package manager. Any other tool is the project toolchain's,
# and its remedy is run: kit tools install. No remedy names a tool manager kit did not find.
guard:remedy() {
  local fix
  case "$1" in
    (bash | git | jq | yq)
      if fix=$(guard:system-command "$1")
      then
        printf 'run: %s\n' "${fix}"
      else
        printf "install %s with this machine's package manager\n" "$1"
      fi
      ;;
    (*) printf 'run: kit tools install\n'
  esac
}

# guard:system-command <tool> prints the system toolchain's command that installs the tool. It fails when there is
# none: no toolchain fills the system role, several do and none is chosen, or the one found names no command. It never
# asks which to use. It runs in a subshell, so the modules it loads stay out of the caller's process. While the module
# path is still being read, as when bash_path in kit.yml needs yq, it uses the release's own modules.
guard:system-command() (
  local provider
  local -a candidates=()
  (( ${#kit_bash_path[@]} )) || kit_bash_path=("${kit_root}/modules/bash")
  use toolchain || return 1
  # The choice kept for the system is read through yq; without it, the only usable toolchain is the one kit uses.
  if ! provider=$(toolchain:chosen system list)
  then
    list:read candidates toolchain:usable system || return 1
    (( ${#candidates[@]} == 1 )) || return 1
    provider=${candidates[0]}
  fi
  toolchain:remedy "${provider}" "$1"
) 2>/dev/null

# guard:jq <verb> <argument>… replaces the process with the verb run again, --jq given as --json, from the directory
# the command was typed in, its output formatted by jq ., and the verb's status kept: --jq for every verb at once.
guard:jq() {
  local verb=$1 arg replaced=false
  local -a args=()
  shift
  for arg in "$@"
  do
    if [[ "${replaced}" == false && "${arg}" == --jq ]]
    then
      arg=--json replaced=true
    fi
    args+=("${arg}")
  done
  # shellcheck disable=SC2016  # the script is single-quoted for bash -c, its arguments after it
  exec bash -o pipefail -c 'cd "$1" && shift && "$@" | jq .' guard:jq "${kit_invoked_in:-.}" "${kit_root}/bin/kit" \
    "${verb}" "${args[@]}"
}

# guard:unknown-verb <word> refuses a word no verb answers, in the words every refusal of one uses. When a verb is one
# or two edits from it, a letter added, dropped, changed, or two swapped, and the edits are fewer than half the word,
# it names the nearest verb and its page. It always returns 2.
guard:unknown-verb() {
  local word=$1 nearest
  nearest=$(find "${kit_root}/bin" -maxdepth 1 -name 'kit-*' -perm -u+x | sed 's@.*/kit-@@' | LC_ALL=C sort \
    | awk -v word="${word}" '
      # the optimal string alignment distance: insertions, deletions, substitutions, and adjacent transpositions
      function distance(a, b,    i, j, la, lb, cost, d) {
        la = length(a); lb = length(b)
        for (i = 0; i <= la; i++) d[i, 0] = i
        for (j = 0; j <= lb; j++) d[0, j] = j
        for (i = 1; i <= la; i++)
          for (j = 1; j <= lb; j++) {
            cost = substr(a, i, 1) != substr(b, j, 1)
            d[i, j] = d[i - 1, j] + 1
            if (d[i, j - 1] + 1 < d[i, j]) d[i, j] = d[i, j - 1] + 1
            if (d[i - 1, j - 1] + cost < d[i, j]) d[i, j] = d[i - 1, j - 1] + cost
            if (i > 1 && j > 1 && substr(a, i, 1) == substr(b, j - 1, 1) && substr(a, i - 1, 1) == substr(b, j, 1) \
              && d[i - 2, j - 2] + 1 < d[i, j]) d[i, j] = d[i - 2, j - 2] + 1
          }
        return d[la, lb]
      }
      { n = distance(word, $0); if (best == "" || n < least) { best = $0; least = n } }
      END { if (best != "" && least <= 2 && least * 2 < length(word)) print best }')
  if [[ -n "${nearest}" ]]
  then
    error "unknown verb '${word}'; did you mean ${nearest}? run: kit help ${nearest}"
  else
    error "unknown verb '${word}'; run: kit help"
  fi
  return 2
}

# guard:arguments <verb> <argument>… judges a verb's arguments against its syntax, read from its header, so every verb
# refuses in one place and in the same words: an unknown flag or argument, an option without its value, two
# alternatives of one group together, or a required argument missing, each in one line that points to kit help <verb>.
# A --help or -h among the arguments, before any --, is answered with that page in this process's place, so every
# verb has the --help a person types first. A verb short of a required argument hands off instead, when stdin,
# stdout, and stderr are all terminals and kittui can be found (modules/bash/tui.sh): it execs kit tui -- <verb>
# <argument>…, whose form asks for the rest. It returns 2 on a refusal, 1 when the verb's usage cannot be read to
# judge a --jq, and 0 when the arguments complete the syntax, which the verb then parses itself; with --jq beside a
# --json the verb takes, guard:jq runs it in this process's place.
# shellcheck disable=SC2154  # syntax_literals, syntax_missing, and syntax_refusal are set by modules/bash/syntax.sh
guard:arguments() {
  local verb=$1 status=0 argument usage form line
  shift
  for argument in "$@"
  do
    [[ "${argument}" != -- ]] || break
    if [[ "${argument}" == --help || "${argument}" == -h ]]
    then
      exec "${kit_root}/bin/kit-help" "${verb}"
    fi
  done
  guard:require tool:jq || return 2
  use syntax || return 2
  syntax:match "${verb}" "$@" || status=$?
  if (( status == 0 ))
  then
    # --jq beside --json is --json read through jq ., where the form the words take, by its literals, offers --json;
    # kit new module's --jq, whose form offers none, means its own thing
    if [[ -n "${syntax_options[--jq]+x}" ]]
    then
      usage=$(help:usage "${verb}") || return 1
      form="kit ${verb}${syntax_literals[*]:+ ${syntax_literals[*]}}"
      while IFS= read -r line
      do
        if [[ "${line}" == *"${form}"* && "${line}" == *--json* ]]
        then
          guard:jq "${verb}" "$@"
        fi
      done <<<"${usage}"
    fi
    return 0
  fi
  if (( status == 1 ))
  then
    if [[ -t 0 && -t 1 && -t 2 ]] && use tui && tui:binary --probe >/dev/null 2>&1
    then
      exec "${kit_root}/bin/kit" tui -- "${verb}" "$@"
    fi
    error "kit ${verb}${syntax_literals[*]:+ ${syntax_literals[*]}} needs ${syntax_missing}; run: kit help ${verb}"
  else
    error "${syntax_refusal}"
  fi
  return 2
}

# ----------------------------------------------------------------------------------------------------------------------
