# shellcheck shell=bash
# context.sh - which kinds a file is of, and which rules govern it (sourced module)
# ----------------------------------------------------------------------------------------------------------------------
#
# A kind of file is defined once, in .kit/kinds/<kind>.yml, and read by everything that asks what a file is:
#
#   - `paths:` globs matched against the repository-relative path, where ** matches any depth
#   - `content:` patterns matched against the first line of a file that exists, so a bash script with no extension
#     is matched by its shebang
#   - `embeds:` the kinds whose text appears inside this kind's files: a justfile's recipe bodies are bash
#
# A path is of every kind whose glob or pattern matches it. A rule under .kit/rules/ and a style under .kit/styles/
# name the kinds they govern under `kind:` in their frontmatter, and carry no glob of their own.
#
# context:render <path>… [--json] prints the files that govern each path: the rules, then the styles, of the path's
# kinds, and then the rules of the kinds those kinds embed, so opening a justfile names the bash rule too. It also
# logs each request in the repository's context log. kit lint and kit new ask for the direct rules alone, so no
# linter runs over a file it cannot parse.
#
# The rules are found where they are and copied nowhere, so every harness and every person finds them the same way.
#
# context:claude gives the same answer in Claude Code's hook protocol. It names the rules but does not quote them,
# so the agent still reads each rule itself, and a hook on Read can record that.
#
# see: docs/context.md

use yaml
shopt -s extglob

# The kinds and the frontmatter, read once per process by context:load: each kind's globs, patterns, and embedded
# kinds, newline-separated, by kind; and each rule's and style's kinds, space-separated, by file.
declare -gA context_kind_paths=() context_kind_content=() context_kind_embeds=() context_file_kinds=()
declare -g context_loaded=false

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

# context:load reads every kind under .kit/kinds/ and the kind: list of every rule and style, once. It runs from
# the repository root. A caller that loops over paths calls it first, so the subshells of the loop inherit the
# tables instead of each reading the files again.
context:load() {
  [[ "${context_loaded}" == true ]] && return 0
  local file kind
  for file in .kit/kinds/*.yml
  do
    [[ -f "${file}" ]] || continue
    kind=${file##*/}
    kind=${kind%.yml}
    context_kind_paths[${kind}]=$(yaml:list "${file}" paths)
    context_kind_content[${kind}]=$(yaml:list "${file}" content)
    context_kind_embeds[${kind}]=$(yaml:list "${file}" embeds)
  done
  for file in .kit/rules/*.md .kit/styles/*.md
  do
    [[ -f "${file}" ]] || continue
    context_file_kinds[${file}]=$(yaml:list "${file}" kind | paste -s -d ' ' -)
  done
  context_loaded=true
}

# context:kinds <path> prints the kinds <path> is of, one per line in name order: every kind with a glob that
# matches the path, or a pattern that matches its first line when the file exists.
context:kinds() {
  local path=$1 kind glob pattern first=''
  context:load
  [[ -f "${path}" ]] && first=$(head -n 1 "${path}" 2>/dev/null || true)
  for kind in $(printf '%s\n' "${!context_kind_paths[@]}" | LC_ALL=C sort)
  do
    # Read each glob with read, not word splitting: an unquoted glob would expand against the working tree first.
    while IFS= read -r glob
    do
      [[ -n "${glob}" ]] || continue
      if context:glob-matches "${path}" "${glob}"
      then
        printf '%s\n' "${kind}"
        continue 2
      fi
    done <<<"${context_kind_paths[${kind}]}"
    [[ -n "${first}" ]] || continue
    while IFS= read -r pattern
    do
      [[ -n "${pattern}" ]] || continue
      if [[ "${first}" =~ ${pattern} ]]
      then
        printf '%s\n' "${kind}"
        continue 2
      fi
    done <<<"${context_kind_content[${kind}]}"
  done
}

# context:embedded <kind>… prints the kinds the given kinds embed, and what those embed in turn, leaving out the
# given kinds themselves, each once.
context:embedded() {
  local kind embedded
  local -A given=() seen=()
  local -a queue=("$@")
  for kind in "$@"
  do
    given[${kind}]=1
  done
  while (( ${#queue[@]} > 0 ))
  do
    kind=${queue[0]}
    queue=("${queue[@]:1}")
    while IFS= read -r embedded
    do
      [[ -n "${embedded}" && -z "${given[${embedded}]-}" && -z "${seen[${embedded}]-}" ]] || continue
      seen[${embedded}]=1
      printf '%s\n' "${embedded}"
      queue+=("${embedded}")
    done <<<"${context_kind_embeds[${kind}]-}"
  done
}

# context:governed-by <rules|styles> <kind>… prints the files under .kit/<rules|styles>/ whose kind: names any of
# the kinds, in name order.
context:governed-by() {
  local dir=$1 file kinds kind
  shift
  for file in .kit/"${dir}"/*.md
  do
    [[ -f "${file}" ]] || continue
    kinds=" ${context_file_kinds[${file}]-} "
    for kind in "$@"
    do
      if [[ "${kinds}" == *" ${kind} "* ]]
      then
        printf '%s\n' "${file}"
        break
      fi
    done
  done
}

# context:rules <path> [--direct] prints each rule file, then each style file, of the kinds <path> is of, and then,
# unless --direct is given, each rule file of the kinds those kinds embed. kit context prints all three; kit lint
# and kit new ask with --direct.
context:rules() {
  local path=$1 mode=${2:-} file
  local -a kinds=() embedded=()
  local -A seen=()
  context:load
  readarray -t kinds < <(context:kinds "${path}")
  (( ${#kinds[@]} > 0 )) || return 0
  while IFS= read -r file
  do
    [[ -n "${file}" ]] || continue
    seen[${file}]=1
    printf '%s\n' "${file}"
  done < <(context:governed-by rules "${kinds[@]}"; context:governed-by styles "${kinds[@]}")
  [[ "${mode}" == --direct ]] && return 0
  readarray -t embedded < <(context:embedded "${kinds[@]}")
  (( ${#embedded[@]} > 0 )) || return 0
  while IFS= read -r file
  do
    [[ -n "${file}" && -z "${seen[${file}]-}" ]] || continue
    printf '%s\n' "${file}"
  done < <(context:governed-by rules "${embedded[@]}")
}

# context:log <path> appends the time and <path> to the context log, recording that its rules were requested.
context:log() {
  local log
  log=$(context:log-path)
  mkdir -p "$(dirname "${log}")"
  printf '%s\t%s\n' "$(date +%s)" "$1" >> "${log}"
}

# context:render <path>… [--json] logs each path and prints the rules that govern the paths. Without --json, it
# prints each rule's path once. With --json, it prints one object per path with its kinds and its rules.
context:render() {
  local json=false arg path rule
  local -a paths=()
  local -A seen=()
  for arg in "$@"
  do
    case "${arg}" in
      (--json) json=true ;;
      (-*)
        error "unknown flag '${arg}'; run: kit help context"
        return 2
        ;;
      (*) paths+=("${arg}")
    esac
  done
  if (( ${#paths[@]} == 0 ))
  then
    error "kit context needs a path; run: kit help context"
    return 2
  fi
  cd "$(git rev-parse --show-toplevel)" || return 1
  context:load
  for path in "${paths[@]}"
  do
    path=$(context:relative "${path}")
    context:log "${path}"
    if [[ "${json}" == true ]]
    then
      context:rules "${path}" | jq --raw-input --slurp --compact-output --arg path "${path}" \
        --arg kinds "$(context:kinds "${path}" | paste -s -d ' ' -)" \
        '{ path: $path, kinds: ($kinds | split(" ") | map(select(. != ""))),
           rules: [split("\n")[] | select(. != "")] }'
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
