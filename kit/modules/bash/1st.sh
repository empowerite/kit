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

# kit:enter starts a verb's process: it exports the directory the command was typed in as kit_invoked_in, for verbs
# that take relative paths, and changes to the repository root, so every verb reads and writes the same paths
# wherever it was typed. A repository from a release that kept its settings in .kit/settings.yml has them moved to
# kit.yml first, before anything reads them.
kit:enter() {
  kit_invoked_in=$(pwd -P)
  export kit_invoked_in
  local repository
  if repository=$(git rev-parse --show-toplevel 2>/dev/null)
  then
    cd "${repository}" || return 1
  fi
  kit:move-settings
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
# returns 2 on a path that does not start with kit:modules/bash/, or a root other than kit: and the sources.
#
# Reading bash_path starts yq, so the entries it found are exported for child processes: kit_bash_path_dirs, their
# directories, one per line, and kit_bash_path_from, all they were resolved from: the release, the home directory, the
# cache directory, the root, and the text of the kit.yml there. A child for which all of them are the same uses the
# directories as they are, and reads kit.yml again otherwise, so an edit to it, another repository, or another release
# is never read stale. Whether each directory exists is asked again every time.
use:load-path() {
  (( ${#kit_bash_path[@]} )) && return 0
  local entry dir root digest settings='' from
  local -a entries dirs=()
  # A verb runs at the repository root, where kit.yml is. A just recipe, or any script that sources this
  # file, may run in a directory below it, so the settings and the relative entries are read from the root then.
  root=$(pwd -P)
  if [[ ! -f kit.yml ]]
  then
    root=$(git rev-parse --show-toplevel 2>/dev/null || pwd -P)
  fi
  # read, a builtin, takes the file's text with no process; at the end of a file with no NUL it returns 1
  if [[ -f "${root}/kit.yml" ]]
  then
    IFS= read -r -d '' settings < "${root}/kit.yml" || true
  fi
  printf -v from '%s\n' "${kit_root}" "${HOME}" "${XDG_CACHE_HOME:-}" "${root}" "${settings}"
  if [[ -n "${kit_bash_path_dirs:-}" && "${kit_bash_path_from:-}" == "${from}" ]]
  then
    readarray -t dirs <<<"${kit_bash_path_dirs}"
    use:keep-path "${dirs[@]}"
    return 0
  fi
  use:yq
  # yaml.sh is read in a subshell, so this does not count as loading it.
  # shellcheck source=SCRIPTDIR/yaml.sh
  readarray -t entries < <(source "${kit_root}/modules/bash/yaml.sh" && yaml:list "${root}/kit.yml" bash_path)
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

# use:find <module> prints the path of every <module>.sh on the module path, in order. Call use:load-path first.
use:find() {
  local dir
  for dir in "${kit_bash_path[@]}"
  do
    [[ -f "${dir}/$1.sh" ]] && printf '%s\n' "${dir}/$1.sh"
  done
  return 0
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
# verb has the --help a person types first. A verb short of a required argument hands off instead, when stdin, stdout, and stderr are all terminals and kittui
# can be found (modules/bash/tui.sh): it execs kit tui -- <verb> <argument>…, whose form asks for the rest. It
# returns 2 on a refusal, and 0 when the arguments complete the syntax, which the verb then parses as it does today;
# with --jq beside a --json the verb takes, guard:jq runs it in this process's place.
# shellcheck disable=SC2154  # syntax_literals, syntax_missing, and syntax_refusal are set by modules/bash/syntax.sh
guard:arguments() {
  local verb=$1 status=0 argument
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
    # --jq beside --json is --json read through jq .; kit new's --jq, with no --json beside it, means its own thing
    if [[ -n "${syntax_options[--jq]+x}" ]] && help:usage "${verb}" | grep --quiet -- '--json'
    then
      guard:jq "${verb}" "$@"
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
