# shellcheck shell=bash
# context.sh - which rules govern a file (sourced module)
# ----------------------------------------------------------------------------------------------------------------------
#
# context:render <path>… [--json] prints the files that govern each path: the rules under .kit/rules/, then the
# style under .kit/styles/ that the repository takes. It also logs each request in the repository's context log.
# Each file's frontmatter declares what it governs:
#
#   - `paths:` globs matched against the repository-relative path, where ** matches any depth
#   - `content:` patterns matched against the first line of a file that exists, so a bash script with no extension
#     is matched by its shebang
#
# The rules are found where they are and copied nowhere, so every harness and every person finds them the same way.
#
# context:claude gives the same answer in Claude Code's hook protocol. It names the rules but does not quote them,
# so the agent still reads each rule itself, and a hook on Read can record that.
#
# see: docs/context.md

use yaml
shopt -s extglob

# context:log-path prints the path of the context log. It is under the repository's common git directory, so every
# worktree shares it.
context:log-path() {
  printf '%s/kit/context.log\n' "$(git rev-parse --path-format=absolute --git-common-dir)"
}

# context:relative <path> prints <path> relative to the repository root. A relative <path> starts from the
# directory the command was typed in, which kit:enter exports as kit_invoked_in before it changes to the root. The
# nearest existing directory in the path is resolved physically (pwd -P), the way git reports the root. So a path
# through a symlinked directory, or to a file not yet created, still comes out relative to the root.
context:relative() {
  local path=$1 root dir rest
  root=$(git rev-parse --show-toplevel)
  [[ "${path}" == /* ]] || path="${kit_invoked_in:-$(pwd -P)}/${path}"
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

# context:glob-matches <path> <glob> is true when the repository-relative path matches the glob. ** matches any
# depth. A lone * also matches across directories, as in bash's [[ == ]], so *.md matches at every depth. A glob
# with no wildcard, such as justfile, matches one path.
context:glob-matches() {
  local path=$1 glob=$2
  glob=${glob//\*\*\//*(*\/)}
  glob=${glob//\*\*/*}
  # shellcheck disable=SC2053  # the glob is the point
  [[ "${path}" == ${glob} ]]
}

# context:rules <path> prints each rule file, then each style file, that governs <path>, by a `paths:` glob or a
# `content:` pattern.
context:rules() {
  local path=$1 rule glob pattern first=''
  [[ -f "${path}" ]] && first=$(head -n 1 "${path}" 2>/dev/null || true)
  for rule in .kit/rules/*.md .kit/styles/*.md
  do
    [[ -f "${rule}" ]] || continue
    # Read each glob with read, not word splitting: an unquoted glob would expand against the working tree first.
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

# context:log <path> appends the time and <path> to the context log, recording that its rules were requested.
context:log() {
  local log
  log=$(context:log-path)
  mkdir -p "$(dirname "${log}")"
  printf '%s\t%s\n' "$(date +%s)" "$1" >> "${log}"
}

# context:render <path>… [--json] logs each path and prints the rules that govern the paths. Without --json, it
# prints each rule's path once. With --json, it prints one object per path with its rules.
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

# context:claude reads Claude Code's PreToolUse hook input on stdin and logs the file the tool is about to write.
# It prints, as additional context, the paths of the rules that govern that file. It prints nothing when the tool
# input has no file path or no rule governs the file.
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
  if [[ "${rules}" == *" "* ]]
  then
    text="kit: ${path} is governed by ${rules}; read each before this change."
  else
    text="kit: ${path} is governed by ${rules}; read it before this change."
  fi
  jq --null-input --compact-output --arg text "${text}" \
    '{ hookSpecificOutput: { hookEventName: "PreToolUse", additionalContext: $text } }'
}

# ----------------------------------------------------------------------------------------------------------------------
