# shellcheck shell=bash
# context.sh - which rules govern a file (sourced module)
# ----------------------------------------------------------------------------------------------------------------------
#
# context:render <path>… [--json] names the rule files under .kit/rules/ that govern each path, and records each
# ask in the repository's context log. A rule's frontmatter declares what it governs: `paths:`, globs matched
# against the repository-relative path, where ** stands for any depth; and `content:`, patterns matched against
# the first line of a file that exists, so a bash script with no extension is bash by its shebang. The rules are
# discovered here and copied nowhere, which is what lets every harness, and every person, find them the same way.
#
# context:claude is the same answer in Claude Code's hook protocol: the hook input on stdin, the rules' paths as
# additional context on stdout. It names, never quotes: reading the rule stays the agent's own act, one a hook on
# Read can record. docs/context.md is the description.

use yaml
shopt -s extglob

# context:log-path prints where every ask is recorded: under the repository's git directory, shared by its worktrees.
context:log-path() {
  printf '%s/kit/context.log\n' "$(git rev-parse --path-format=absolute --git-common-dir)"
}

# context:relative <path> prints <path> relative to the repository root. A relative path is taken from the current
# directory. The nearest directory of the path that exists is resolved physically, as git reports the root, so a
# path through a symlinked directory, or to a file not yet created, still cuts at the root.
context:relative() {
  local path=$1 root dir rest
  root=$(git rev-parse --show-toplevel)
  [[ "${path}" == /* ]] || path="${PWD}/${path}"
  dir=${path%/*}
  rest=${path##*/}
  while [[ -n "${dir}" && ! -d "${dir}" ]]
  do
    rest="${dir##*/}/${rest}"
    dir=${dir%/*}
  done
  dir=$(cd "${dir:-/}" && pwd -P)
  path="${dir}/${rest}"
  printf '%s\n' "${path#"${root}"/}"
}

# context:glob-matches <path> <glob> is true when the repository-relative path matches the glob; ** is any depth.
context:glob-matches() {
  local path=$1 glob=$2
  glob=${glob//\*\*\//*(*\/)}
  glob=${glob//\*\*/*}
  # shellcheck disable=SC2053  # the glob is the point
  [[ "${path}" == ${glob} ]]
}

# context:rules <path> prints the rule files governing <path>, by its path or by its first line.
context:rules() {
  local path=$1 rule glob pattern first=''
  [[ -f "${path}" ]] && first=$(head -n 1 "${path}" 2>/dev/null || true)
  for rule in .kit/rules/*.md
  do
    [[ -f "${rule}" ]] || continue
    # Read, never word-split: an unquoted glob would be expanded against the working tree before it is matched.
    while IFS= read -r glob
    do
      [[ -n "${glob}" ]] || continue
      if context:glob-matches "${path}" "${glob}"
      then
        printf '%s\n' "${rule}"
        continue 2
      fi
    done < <(yaml:list "${rule}" paths)
    [[ -n "${first}" ]] || continue
    while IFS= read -r pattern
    do
      [[ -n "${pattern}" ]] || continue
      if [[ "${first}" =~ ${pattern} ]]
      then
        printf '%s\n' "${rule}"
        continue 2
      fi
    done < <(yaml:list "${rule}" content)
  done
}

# context:log <path> records that the context of <path> was asked for.
context:log() {
  local log
  log=$(context:log-path)
  mkdir -p "$(dirname "${log}")"
  printf '%s\t%s\n' "$(date +%s)" "$1" >> "${log}"
}

# context:render <path>… [--json] prints the rules governing the paths: once each, as paths to read; with --json,
# one object per path with its rules.
context:render() {
  local json=false arg path rule
  local -a paths=()
  local -A seen=()
  for arg in "$@"
  do
    case "${arg}" in
      (--json) json=true ;;
      (-*)
        error "unknown flag '${arg}' (usage: kit context <path>… [--json] | kit context --claude)"
        return 2
        ;;
      (*) paths+=("${arg}")
    esac
  done
  if (( ${#paths[@]} == 0 ))
  then
    error "usage: kit context <path>… [--json] | kit context --claude"
    return 2
  fi
  cd "$(git rev-parse --show-toplevel)" || return 1
  for path in "${paths[@]}"
  do
    path=$(context:relative "${path}")
    context:log "${path}"
    if [[ "${json}" == true ]]
    then
      context:rules "${path}" | jq --raw-input --slurp --compact-output --arg path "${path}" \
        '{ path: $path, rules: [split("\n")[] | select(. != "")] }'
      continue
    fi
    while IFS= read -r rule
    do
      [[ -n "${rule}" && -z "${seen[${rule}]:-}" ]] || continue
      seen[${rule}]=1
      printf '%s\n' "${rule}"
    done < <(context:rules "${path}")
  done
}

# context:claude reads Claude Code's PreToolUse hook input on stdin and answers with the rules governing the file
# the tool is about to write, as additional context naming each rule's path. A tool input with no file path, or
# a file no rule governs, gets no answer.
context:claude() {
  local input path rules text=''
  input=$(cat)
  path=$(jq --raw-output '.tool_input.file_path // empty' <<<"${input}")
  [[ -n "${path}" ]] || return 0
  cd "$(git rev-parse --show-toplevel)" || return 0
  path=$(context:relative "${path}")
  context:log "${path}"
  rules=$(context:rules "${path}" | paste -s -d ' ' -)
  [[ -n "${rules}" ]] || return 0
  text="kit: ${path} is governed by ${rules}; read it before this change."
  jq --null-input --compact-output --arg text "${text}" \
    '{ hookSpecificOutput: { hookEventName: "PreToolUse", additionalContext: $text } }'
}

# ----------------------------------------------------------------------------------------------------------------------
