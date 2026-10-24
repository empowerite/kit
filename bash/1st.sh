# shellcheck shell=bash
# 1st.sh - the kit's bash loader: output verbs, guards, and lazy module loading (sourced module)
# ----------------------------------------------------------------------------------------------------------------------
#
# Sourced by every verb, bin/kit-<verb>.bash, which bin/kit dispatches to with kit_root exported. It defines functions only and sets no shell option, so the caller's options stand. A module under
# bash/ is loaded with `use <module>`, once, by the verb that needs it; there is no other way in.
#
# Every message goes to stderr, so stdout stays clean for data. Color is off when stderr is not a terminal or
# NO_COLOR is set. Under GitHub Actions, warn and error also emit the matching workflow command.

kit_bash_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
declare -gr kit_bash_dir
declare -Ag kit_modules=([1st]=1)

# use <module>… loads bash/<module>.sh once. An unknown module is a bug in the caller, so it is fatal.
use() {
  local module
  for module in "$@"
  do
    [[ ${kit_modules[${module}]+x} ]] && continue
    if [[ ! -f "${kit_bash_dir}/${module}.sh" ]]
    then
      error "no such kit bash module: ${module} (looked in ${kit_bash_dir})"
      return 2
    fi
    kit_modules[${module}]=1
    # shellcheck disable=SC1090  # the path is computed from the module name
    source "${kit_bash_dir}/${module}.sh"
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

# guard:require <predicate>… returns non-zero on the first predicate that does not hold, with the remedy.
#   gh           gh is usably authenticated (a real API call, since a revoked token still passes gh auth status)
#   signing-key  git user.signingkey is set
#   tool:<name>  <name> is on PATH
guard:require() {
  local predicate
  for predicate in "$@"
  do
    case "${predicate}" in
      (gh)
        if ! gh api user --jq '.login' >/dev/null 2>&1
        then
          error "run: gh auth login (the GitHub CLI is not authenticated)"
          return 1
        fi
        ;;
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

# git:default-branch prints the default branch: origin/HEAD, with main as the fallback.
git:default-branch() {
  local branch
  branch=$(git symbolic-ref --quiet --short refs/remotes/origin/HEAD 2>/dev/null | sed 's@^origin/@@' || true)
  [[ -n "${branch}" ]] || branch=main
  printf '%s\n' "${branch}"
}

# git:main-root prints the root of the main checkout, the first entry of the worktree list. worktrees/ lives
# under it whether the caller stands in the main checkout or in a linked worktree.
git:main-root() {
  git worktree list --porcelain | sed -n 's@^worktree @@p' | head -n 1
}

# ----------------------------------------------------------------------------------------------------------------------
