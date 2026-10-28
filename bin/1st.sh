# shellcheck shell=bash
# 1st.sh - the kit's bash loader: output verbs, guards, and lazy module loading (sourced module)
# ----------------------------------------------------------------------------------------------------------------------
#
# Every verb, bin/kit-<verb>.sh, sources this file. bin/kit runs the verb with kit_root exported. This file only
# defines functions and sets no shell options, so the caller's options stay in effect.
#
#   - A verb is an executable kit-<verb>.sh in this directory.
#   - A module is a sourced <module>.sh in this directory. The verb or module that needs it loads it with
#     `use <module>`, which sources it at most once.
#
# Every message goes to stderr, so stdout carries only data. Color is off when stderr is not a terminal or NO_COLOR
# is set. Under GitHub Actions, warn and error also print the matching workflow command (::warning::, ::error::).

kit_bin_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
declare -gr kit_bin_dir
declare -Ag kit_modules=([1st]=1)

# use <module>… sources bin/<module>.sh, once per module. An unknown module is a bug in the caller: use prints an
# error and returns 2.
use() {
  local module
  for module in "$@"
  do
    [[ ${kit_modules[${module}]+x} ]] && continue
    if [[ ! -f "${kit_bin_dir}/${module}.sh" ]]
    then
      error "no such kit bash module: ${module} (looked in ${kit_bin_dir})"
      return 2
    fi
    kit_modules[${module}]=1
    # shellcheck disable=SC1090  # the path is computed from the module name
    source "${kit_bin_dir}/${module}.sh"
  done
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
