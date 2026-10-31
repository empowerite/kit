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

# context:glob-ere <glob> prints the extended regular expression that matches what context:glob-matches matches,
# anchored at both ends: **/ is any run of directories, * and ** any text, slashes included, and ? one character. A
# bracket expression is kept as it is; every other metacharacter is escaped.
context:glob-ere() {
  local glob=$1 out='' c
  local -i i=0
  while (( i < ${#glob} ))
  do
    c=${glob:i:1}
    if [[ "${glob:i:3}" == '**/' ]]
    then
      out+='(.*/)?'
      i+=3
      continue
    fi
    case "${c}" in
      ('*') out+='.*' ;;
      ('?') out+='.' ;;
      ('[')
        out+=${glob:i:$(( $(expr "${glob:i}" : '\[[^]]*\]') ))}
        i+=$(( $(expr "${glob:i}" : '\[[^]]*\]') - 1 ))
        ;;
      ('.' | '+' | '(' | ')' | '{' | '}' | '|' | '^' | '$' | "\\") out+="\\${c}" ;;
      (*) out+=${c}
    esac
    i+=1
  done
  printf '^%s$\n' "${out}"
}

# context:awk-string <text> prints the text as an awk string literal, quoted, with its backslashes and quotes escaped.
context:awk-string() {
  local text=${1//\\/\\\\}
  printf '"%s"' "${text//\"/\\\"}"
}

# context:compile prints an awk program that classifies paths by every kind context:load read. Its input is one
# "<path>\t<1 when the path is a regular file, else 0>" line per path, and for each it prints "<path>\t<kinds>", the
# kinds space-separated in name order. Each kind is one test of the path against its globs, joined into one regular
# expression, then of the file's first line against its patterns; the kinds made of others follow until none grows.
# shellcheck disable=SC2016  # the $1 and $2 it prints are awk's fields, not the shell's
context:compile() {
  local kind glob pattern member ere test member_test
  local -a kinds globs patterns
  context:load || return 1
  readarray -t kinds < <(printf '%s\n' "${!context_kind_paths[@]}" | LC_ALL=C sort)
  printf 'BEGIN { FS = "\\t" }\n{\n  path = $1; split("", is); first = ""; has = 0\n'
  printf '  if ($2 == 1 && (getline first < path) > 0) has = 1\n  if ($2 == 1) close(path)\n'
  for kind in "${kinds[@]}"
  do
    readarray -t globs < <(awk 'NF' <<<"${context_kind_paths[${kind}]}")
    readarray -t patterns < <(awk 'NF' <<<"${context_kind_content[${kind}]}")
    test=''
    if (( ${#globs[@]} ))
    then
      ere=''
      for glob in "${globs[@]}"
      do
        ere+="${ere:+|}($(context:glob-ere "${glob}"))"
      done
      test="path ~ $(context:awk-string "${ere}")"
    fi
    if (( ${#patterns[@]} ))
    then
      ere=''
      for pattern in "${patterns[@]}"
      do
        ere+="${ere:+|}(${pattern})"
      done
      test+="${test:+ || }(has && first ~ $(context:awk-string "${ere}"))"
    fi
    [[ -n "${test}" ]] && printf '  if (%s) is[%s] = 1\n' "${test}" "$(context:awk-string "${kind}")"
  done
  printf '  do {\n    grew = 0\n'
  for kind in "${kinds[@]}"
  do
    member_test=''
    while IFS= read -r member
    do
      [[ -n "${member}" ]] && member_test+="${member_test:+ || }($(context:awk-string "${member}") in is)"
    done <<<"${context_kind_members[${kind}]}"
    [[ -n "${member_test}" ]] || continue
    printf '    if (!(%s in is) && (%s)) { is[%s] = 1; grew = 1 }\n' "$(context:awk-string "${kind}")" \
      "${member_test}" "$(context:awk-string "${kind}")"
  done
  printf '  } while (grew)\n  out = ""\n'
  for kind in "${kinds[@]}"
  do
    printf '  if (%s in is) out = out (out == "" ? "" : " ") %s\n' "$(context:awk-string "${kind}")" \
      "$(context:awk-string "${kind}")"
  done
  printf '  print path "\\t" out\n}\n'
}

# context:matcher prints the path of the compiled classifier for the kinds under .kit/kinds/, compiling it first when
# the kinds have changed. It is kept in the repository's common git directory, named by a checksum of the kinds
# files' names and text, so every worktree shares it and an edit to a kind compiles afresh.
context:matcher() {
  local dir key file
  dir="$(git rev-parse --path-format=absolute --git-common-dir)/kit"
  key=$( { printf '%s\n' .kit/kinds/*.yml; cat .kit/kinds/*.yml 2>/dev/null; } | cksum | cut -d ' ' -f 1)
  file="${dir}/kinds-${key}.awk"
  if [[ ! -f "${file}" ]]
  then
    # Two processes may compile at once, so each writes a file of its own, by BASHPID, which a subshell does not
    # share, and moves it into place whole; a matcher of other kinds is removed, never the one being placed.
    mkdir -p "${dir}"
    context:compile > "${file}.${BASHPID}" || { rm -f "${file}.${BASHPID}"; return 1; }
    find "${dir}" -name 'kinds-*.awk' ! -name "kinds-${key}.awk" -delete
    mv "${file}.${BASHPID}" "${file}"
  fi
  printf '%s\n' "${file}"
}

# context:classify reads repository-relative paths on stdin, one per line, and prints "<path>\t<kinds>" for each, in
# one process however many there are: every kind with a glob that matches the path, or a pattern that matches the
# first line of a file that exists, and every kind made of one of those under kinds:, space-separated in name order.
context:classify() {
  local matcher path
  matcher=$(context:matcher) || return 1
  while IFS= read -r path
  do
    if [[ -f "${path}" ]]
    then
      printf '%s\t1\n' "${path}"
    else
      printf '%s\t0\n' "${path}"
    fi
  done | awk -f "${matcher}"
}

# context:kinds <path> prints the kinds <path> is of, one per line in name order, as context:classify finds them.
context:kinds() {
  printf '%s\n' "$1" | context:classify | cut -f 2 | tr ' ' '\n' | awk 'NF'
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
