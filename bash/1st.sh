# shellcheck shell=bash
# 1st.sh - the kit's bash loader: the module path, output verbs, guards, and run (sourced module)
# ----------------------------------------------------------------------------------------------------------------------
#
# Every verb, bin/kit-<verb>, sources this file and then calls kit:enter. A verb runs the same through bin/kit and
# called directly. This file sets no shell options, so the caller's options stay in effect.
#
#   - A verb is an executable bin/kit-<verb>, with no extension.
#   - A module is a sourced <module>.sh in a directory on the module path. The verb or module that needs it loads
#     it with `use <module>`, which sources every <module>.sh on the path, in order, once per process.
#   - The module path is the bash_path setting in .kit/settings.yml. Its first entry is kit:bash/, this file's
#     directory. A later file can replace a function an earlier one defined; super.sh lets it call the one it
#     replaced.
#   - Within the modules, work is a function call. Across a process boundary it is `run kit <verb>`, which runs
#     this release's kit and carries the trace depth.
#
# Every message goes to stderr, so stdout carries only data. Color is off when stderr is not a terminal or NO_COLOR
# is set. Under GitHub Actions, warn and error also print the matching workflow command (::warning::, ::error::).

if (( BASH_VERSINFO[0] < 4 || (BASH_VERSINFO[0] == 4 && BASH_VERSINFO[1] < 4) ))
then
  echo "kit needs bash 4.4 or later; this is ${BASH_VERSION}" >&2
  return 1
fi

# kit_root is the release: the directory above this file's.
kit_root=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
export kit_root
declare -Ag kit_modules=([1st]=1)
declare -ag kit_bash_path=()

# kit:enter starts a verb's process: it exports the directory the command was typed in as kit_invoked_in, for verbs
# that take relative paths, and changes to the repository root, so every verb reads and writes the same paths
# wherever it was typed.
kit:enter() {
  kit_invoked_in=$(pwd -P)
  export kit_invoked_in
  local repository
  if repository=$(git rev-parse --show-toplevel 2>/dev/null)
  then
    cd "${repository}" || return 1
  fi
}

# use:path fills kit_bash_path with the module path's directories, once per process: each entry of the bash_path
# setting, else kit:bash/ alone. An entry is kit:<dir> (in the release), an absolute directory, or a directory
# relative to the repository root. It returns 2 on a path that does not start with kit:bash/, an entry that is not
# a directory, or a root other than kit:.
use:path() {
  (( ${#kit_bash_path[@]} )) && return 0
  local entry dir
  local -a entries
  # yaml.sh is read in a subshell, so this does not count as loading it.
  readarray -t entries < <(source "${kit_root}/bash/yaml.sh" && yaml:list .kit/settings.yml bash_path)
  (( ${#entries[@]} )) || entries=(kit:bash/)
  if [[ "${entries[0]}" != kit:bash/ ]]
  then
    error "bash_path must start with kit:bash/, the kit's own modules (bash_path in .kit/settings.yml)"
    return 2
  fi
  for entry in "${entries[@]}"
  do
    case "${entry}" in
      (kit:*) dir=${kit_root}/${entry#kit:} ;;
      (/*) dir=${entry} ;;
      ([A-Za-z0-9_-]*:*)
        error "bash_path entry '${entry}' names a root the kit does not know; the root is kit: (bash_path in .kit/settings.yml)"
        return 2
        ;;
      (*) dir=$(pwd -P)/${entry}
    esac
    if [[ ! -d "${dir}" ]]
    then
      error "bash_path entry '${entry}' is not a directory (bash_path in .kit/settings.yml)"
      return 2
    fi
    kit_bash_path+=("$(cd "${dir}" && pwd -P)")
  done
}

# use:find <module> prints the path of every <module>.sh on the module path, in order. Call use:path first.
use:find() {
  local dir
  for dir in "${kit_bash_path[@]}"
  do
    [[ -f "${dir}/$1.sh" ]] && printf '%s\n' "${dir}/$1.sh"
  done
  return 0
}

# use:modules <pattern> prints the name of every module on the module path that matches the glob <pattern>, once
# each, sorted: use:modules 'forge-*' lists the forge providers.
use:modules() {
  use:path || return 2
  local dir file
  for dir in "${kit_bash_path[@]}"
  do
    for file in "${dir}"/$1.sh
    do
      [[ -f "${file}" ]] && basename "${file}" .sh
    done
  done | LC_ALL=C sort --unique
}

# use <module>… sources every <module>.sh on the module path, in order, once per module. An unknown module is a bug
# in the caller: use prints an error and returns 2.
use() {
  local module file
  local -a files
  use:path || return 2
  for module in "$@"
  do
    [[ ${kit_modules[${module}]+x} ]] && continue
    readarray -t files < <(use:find "${module}")
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
elif [[ -t 2 ]]
then
  declare -gr out_color=1
else
  declare -gr out_color=0
fi

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

# out:emit <icon> <sgr> <message>… prints one line to stderr.
out:emit() {
  local icon=$1 sgr=$2
  shift 2
  printf '%s%s %s%s\n' "$(out:sgr "${sgr}")" "${icon}" "$*" "$(out:sgr 0)" >&2
}

info()    { out:emit '•' '' "$@"; }
step()    { out:emit '→' '36' "$@"; }
ok()      { out:emit '✅' '32' "$@"; }
section() { out:emit '' '1' "── $* ──"; }

warn() {
  out:emit '⚠️' '33' "$@"
  [[ -n "${GITHUB_ACTIONS:-}" ]] && echo "::warning::$*" >&2
  return 0
}

error() {
  out:emit '❌' '31' "$@"
  [[ -n "${GITHUB_ACTIONS:-}" ]] && echo "::error::$*" >&2
  return 0
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
          error "run: mise install (${tool} is not on PATH)"
          return 1
        fi
        ;;
      (*)
        error "guard:require: unknown predicate '${predicate}'"
        return 1
    esac
  done
}

# ----------------------------------------------------------------------------------------------------------------------
