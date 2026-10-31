# shellcheck shell=bash
# context.sh - which kinds a file is of, and which rules govern it (sourced module)
# ----------------------------------------------------------------------------------------------------------------------
#
# A kind of file is defined once, in a kinds file under .kit/kinds/, and read by everything that asks what a file is.
# A kinds file is a YAML stream of one document per kind, separated by `---` lines, so one file may define several
# kinds. Each document has:
#
#   - `name:` the kind's name, which rules and styles use; the file's own name plays no part
#   - `paths:` globs matched against the repository-relative path, where ** matches any depth
#   - `content:` patterns matched against the first line of a file that exists, so a bash script with no extension
#     is matched by its shebang
#   - `embeds:` the kinds whose text appears inside this kind's files: a justfile's recipe bodies are bash
#   - `kinds:` other kinds this kind is made of: a path of any of them is of this kind too, so text names bash,
#     markdown, and the rest, and a style that governs text governs them all
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
declare -gA context_kind_paths=() context_kind_content=() context_kind_embeds=() context_kind_members=()
declare -gA context_file_kinds=()
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
  local file kind field value
  local -A defined=()
  while IFS=$'\x1f' read -r file kind field value
  do
    case "${field}" in
      (error)
        error "${file}: ${value}"
        return 1
        ;;
      (name)
        if [[ -n "${defined[${kind}]:-}" ]]
        then
          error "${file}: the kind ${kind} is defined twice, here and in ${defined[${kind}]}"
          return 1
        fi
        defined[${kind}]=${file}
        context_kind_paths[${kind}]='' context_kind_content[${kind}]='' context_kind_embeds[${kind}]=''
        context_kind_members[${kind}]=''
        ;;
      (paths) context_kind_paths[${kind}]+=${value}$'\n' ;;
      (content) context_kind_content[${kind}]+=${value}$'\n' ;;
      (embeds) context_kind_embeds[${kind}]+=${value}$'\n' ;;
      (kinds) context_kind_members[${kind}]+=${value}$'\n'
    esac
  done < <(context:read-kinds .kit/kinds/*.yml)
  for file in .kit/rules/*.md .kit/styles/*.md
  do
    [[ -f "${file}" ]] || continue
    context_file_kinds[${file}]=$(yaml:list "${file}" kind | paste -s -d ' ' -)
  done
  context_loaded=true
}

# context:read-kinds <file>… prints every kind the kinds files define, one field to a line: "<file>", "<kind>",
# "<field>", and "<value>", joined by the unit separator \x1f. A name line opens each kind, then one line follows per
# item of its paths, content, embeds, and kinds lists, unquoted as yaml:list unquotes. A document without a name prints an
# error line instead, with an empty kind and the field error. The separator is not a tab, because read collapses
# consecutive tabs and the empty kind would vanish.
context:read-kinds() {
  local file
  for file in "$@"
  do
    [[ -f "${file}" ]] || continue
    awk -v file="${file}" '
      function close_doc() {
        if (lines && name == "") print file "\037\037error\037a kind has no name: (see packages/kind.schema.json)"
        else for (i = 1; i <= n; i++) print file "\037" name "\037" out[i]
        name = ""; n = 0; lines = 0; key = ""
      }
      /^---[ \t]*$/ { close_doc(); next }
      /^#/ || /^[ \t]*$/ { next }
      /^name: / {
        v = $0; sub(/^name: */, "", v)
        if (v ~ /^".*"$/ || v ~ /^\x27.*\x27$/) v = substr(v, 2, length(v) - 2)
        name = v; out[++n] = "name\037" v; lines++; next
      }
      /^[A-Za-z0-9_-]+:/ { key = $0; sub(/:.*$/, "", key); lines++; next }
      /^  - / && (key == "paths" || key == "content" || key == "embeds" || key == "kinds") {
        v = $0; sub(/^  - /, "", v)
        if (v ~ /^"/) gsub(/^"|"$/, "", v); else sub(/[ \t]+#.*$/, "", v)
        out[++n] = key "\037" v; lines++
      }
      END { close_doc() }' "${file}"
  done
}

# context:kinds <path> prints the kinds <path> is of, one per line in name order: every kind with a glob that
# matches the path, or a pattern that matches its first line when the file exists, and then every kind made of one
# of those under kinds:, and of those in turn.
context:kinds() {
  local path=$1 kind glob pattern member first='' grew=true
  local -A is=()
  context:load
  [[ -f "${path}" ]] && first=$(head -n 1 "${path}" 2>/dev/null || true)
  for kind in "${!context_kind_paths[@]}"
  do
    # Read each glob with read, not word splitting: an unquoted glob would expand against the working tree first.
    while IFS= read -r glob
    do
      [[ -n "${glob}" ]] || continue
      if context:glob-matches "${path}" "${glob}"
      then
        is[${kind}]=1
        continue 2
      fi
    done <<<"${context_kind_paths[${kind}]}"
    [[ -n "${first}" ]] || continue
    while IFS= read -r pattern
    do
      [[ -n "${pattern}" ]] || continue
      if [[ "${first}" =~ ${pattern} ]]
      then
        is[${kind}]=1
        continue 2
      fi
    done <<<"${context_kind_content[${kind}]}"
  done
  while [[ "${grew}" == true ]]
  do
    grew=false
    for kind in "${!context_kind_members[@]}"
    do
      [[ -z "${is[${kind}]:-}" ]] || continue
      while IFS= read -r member
      do
        if [[ -n "${member}" && -n "${is[${member}]:-}" ]]
        then
          is[${kind}]=1
          grew=true
          break
        fi
      done <<<"${context_kind_members[${kind}]}"
    done
  done
  (( ${#is[@]} )) || return 0
  printf '%s\n' "${!is[@]}" | LC_ALL=C sort
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
      (--) ;;
      (--json) json=true ;;
      (*) paths+=("${arg}")
    esac
  done
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

# context:choices <key> [<answer>…] answers kit syntax --choices for <path>: an open set, every tracked file, since a
# path kit context takes need not be among them.
context:choices() {
  [[ "$1" == '<path>' ]] || return 1
  if [[ "${2:-}" == --kind ]]
  then
    printf 'open\n'
    return 0
  fi
  git ls-files | awk '{ printf "%s\t\t\t\n", $0 }'
}

# ----------------------------------------------------------------------------------------------------------------------
