# shellcheck shell=bash
# generate.sh - fill each kit:generate region with its command's output, until nothing changes (sourced module)
# ----------------------------------------------------------------------------------------------------------------------
# shellcheck disable=SC2154  # kit_root is exported by bin/kit
# shellcheck disable=SC2016  # the $ in a pattern anchors a line; nothing is meant to expand
#
# A region starts at a directive line and ends at an end line:
#
#   - opened by a line that is `kit:generate <name>`, or `kit:generate <name> `<command>``
#   - closed by a line that is `kit:end <name>`
#
# Only non-alphanumeric characters may come before or after the marker on its line. So a directive works inside
# any comment syntax without the kit knowing that syntax: `#`, `//`, `--`, `;`, `%`, `'`, `<!-- -->`, `/* */`,
# and any other, since comment delimiters are punctuation. A line of prose has letters before the marker, so it is
# not read at all.
#
# The directive and end lines stay. The command's standard output replaces the lines between them. With no
# command, the command is `kit region`, which reads the region's name as the path of a region an active module
# delivers.
#
# The command is a bash command line, pipes included. It runs:
#
#   - from the repository root
#   - with the running kit's bin/ first on PATH
#   - with stdin closed
#   - with errexit and pipefail on
#   - with KIT_GENERATE_FILE and KIT_GENERATE_REGION set to the file and the region's name
#
# A command cannot contain a literal backtick. Use $() instead.
#
# The file and each command's output are expanded as separate units, and each must be balanced on its own. An
# output may open regions of its own if it also closes them. Those regions are expanded in turn, depth first, so
# regions can nest and one pass settles a file. In markdown, directives inside a fenced code block are ignored, so
# a page can show a directive as an example.
#
# The file is rewritten only when the whole expansion succeeds. Otherwise the file is left unchanged and the run
# fails, naming the file and the line. The run fails when:
#
#   - a command fails
#   - a unit closes a region it did not open, or opens one it does not close
#   - a line starts with the marker but is not a valid directive, so a misspelled directive is never silently
#     skipped
#   - a region name is opened twice
#   - in markdown, a fenced code block opens and never closes (in the file or in an output), since the directives
#     after it could not be read
#   - the file contains an unresolved conflict marker
#
# see: bin/kit-generate.sh, bin/kit-region.sh, docs/generate.md

# The directive patterns. A line is a directive only when nothing but non-alphanumeric characters (the comment
# opener and closer) come before and after the marker. The name is any run of characters without whitespace or a
# backtick. The command is everything between the first backtick after the name and the last backtick on the line.
declare -g generate_open_re='^[^[:alnum:]]*kit:generate[[:space:]]+([^[:space:]`]+)([[:space:]]+`(.+)`)?[^[:alnum:]]*$'
declare -g generate_end_re='^[^[:alnum:]]*kit:end[[:space:]]+([^[:space:]`]+)[^[:alnum:]]*$'
# A line that starts with a marker must be a valid directive, or it is an error. It is never read as prose.
declare -g generate_marker_re='^[^[:alnum:]]*kit:(generate|end)([^[:alnum:]]|$)'
declare -g generate_fence_re='^[[:space:]]*(```|~~~)'

# An unresolved merge conflict: a line of seven < or seven >, alone or followed by git's label. The middle marker
# (seven =) is not matched: it is also a markdown heading underline, and git never writes it without the opening one.
declare -g generate_conflict_re='^(<{7}|>{7})([[:space:]].*)?$'

# generate:carries <file> is true when the file has a line that starts with a marker. Files under .kit/templates/
# never count: a template is the skeleton of a file, and its directive belongs to the file made from it.
generate:carries() {
  [[ -f "$1" && ! "$1" =~ (^|/)\.kit/templates/ ]] && grep --quiet --extended-regexp "${generate_marker_re}" "$1"
}

# generate:markdown <file> is true for a markdown file (.md or .markdown), the only kind where fenced code blocks
# are skipped.
generate:markdown() {
  [[ "$1" == *.md || "$1" == *.markdown ]]
}

# generate:discover [<path>…] prints every file that contains a directive. With no paths, it searches tracked
# files and untracked files git does not ignore, so a file kit install just wrote counts before it is added. With
# paths, it prints each named file that contains a directive, wherever it is.
generate:discover() {
  local path
  if (( $# == 0 ))
  then
    while IFS= read -r -d '' path
    do
      generate:carries "${path}" && printf '%s\n' "${path}"
    done < <(git ls-files -z --cached --others --exclude-standard)
    return 0
  fi
  for path in "$@"
  do
    generate:carries "${path}" && printf '%s\n' "${path}"
  done
  return 0
}

# generate:target <file> prints the file a write goes to: <file> itself, or the target of a symlink at <file>.
generate:target() {
  if [[ -L "$1" ]]
  then readlink -f "$1"
  else printf '%s\n' "$1"
  fi
}

# generate:command <file> <where> <line> <name> <command> <out> runs one region's command and writes its stdout
# to <out>. On failure it prints an error naming <where> and <line> (the unit and line of the directive).
generate:command() {
  local file=$1 where=$2 number=$3 name=$4 command=$5 out=$6
  if ! KIT_GENERATE_FILE="${file}" KIT_GENERATE_REGION="${name}" PATH="${kit_root}/bin:${PATH}" \
    bash -o errexit -o pipefail -c "${command}" </dev/null >"${out}"
  then
    error "${where}:${number}: the command failed: ${command}"
    return 2
  fi
}

# generate:flip toggles the caller's fence variable (true inside a fenced code block) at a fence line.
generate:flip() {
  if [[ "${fence}" == true ]]
  then fence=false
  else fence=true
  fi
}

# generate:unit <in> <out> <file> <where> <markdown> expands one unit (the file, or a command's output) from <in>
# into <out>. Every message starts with <where>: the file's path for the file, or, for an output, the file, the
# directive's line, and the command. It makes two passes:
#
#   1. check the unit's shape: every marker line is a directive, and every region opens and closes in this unit,
#      in order
#   2. run each top-level region's command, expand its output as a unit of its own, and write the result between
#      the directive and end lines
generate:unit() {
  local in=$1 out=$2 file=$3 where=$4 markdown=$5
  local -a lines stack opened
  local n i j line fence=false fenced=0 depth name command body result
  mapfile -t lines < "${in}"
  n=${#lines[@]}
  stack=()
  opened=()
  for (( i = 0; i < n; i++ ))
  do
    line=${lines[i]}
    if [[ "${markdown}" == true && "${line}" =~ ${generate_fence_re} ]]
    then
      generate:flip
      [[ "${fence}" == true ]] && fenced=$(( i + 1 ))
      continue
    fi
    [[ "${fence}" == false && "${line}" =~ ${generate_marker_re} ]] || continue
    if [[ "${line}" =~ ${generate_open_re} ]]
    then
      stack+=("${BASH_REMATCH[1]}")
      opened+=("$(( i + 1 ))")
    elif [[ "${line}" =~ ${generate_end_re} ]]
    then
      name=${BASH_REMATCH[1]}
      if (( ${#stack[@]} == 0 ))
      then
        error "${where}:$(( i + 1 )): closes '${name}', which is not open"
        return 2
      fi
      if [[ "${stack[-1]}" != "${name}" ]]
      then
        error "${where}:$(( i + 1 )): closes '${name}' while '${stack[-1]}' is open"
        return 2
      fi
      unset 'stack[-1]' 'opened[-1]'
    else
      error "${where}:$(( i + 1 )): not a directive: ${line}"
      return 2
    fi
  done
  if (( ${#stack[@]} > 0 ))
  then
    error "${where}:${opened[-1]}: the region '${stack[-1]}' never closes"
    return 2
  fi
  if [[ "${fence}" == true ]]
  then
    error "${where}: a fenced code block opens at line ${fenced} and never closes"
    return 2
  fi
  : > "${out}"
  fence=false
  i=0
  while (( i < n ))
  do
    line=${lines[i]}
    printf '%s\n' "${line}" >> "${out}"
    if [[ "${markdown}" == true && "${line}" =~ ${generate_fence_re} ]]
    then
      generate:flip
      i=$(( i + 1 ))
      continue
    fi
    if [[ "${fence}" == true ]] || ! [[ "${line}" =~ ${generate_open_re} ]]
    then
      i=$(( i + 1 ))
      continue
    fi
    name=${BASH_REMATCH[1]}
    command=${BASH_REMATCH[3]:-kit region}
    body=$(mktemp)
    result=$(mktemp)
    if ! generate:command "${file}" "${where}" "$(( i + 1 ))" "${name}" "${command}" "${body}" \
      || ! generate:unit "${body}" "${result}" "${file}" "${where}:$(( i + 1 )) (the output of ${command})" "${markdown}"
    then
      rm -f "${body}" "${result}"
      return 2
    fi
    cat "${result}" >> "${out}"
    rm -f "${body}" "${result}"
    # Skip the old body up to this region's end line, counting depth. The first pass already checked the nesting.
    depth=1
    for (( j = i + 1; depth > 0; j++ ))
    do
      line=${lines[j]}
      if [[ "${markdown}" == true && "${line}" =~ ${generate_fence_re} ]]
      then
        generate:flip
        continue
      fi
      [[ "${fence}" == false ]] || continue
      if [[ "${line}" =~ ${generate_open_re} ]]
      then depth=$(( depth + 1 ))
      elif [[ "${line}" =~ ${generate_end_re} ]]
      then depth=$(( depth - 1 ))
      fi
    done
    printf '%s\n' "${lines[j - 1]}" >> "${out}"
    i=${j}
  done
}

# generate:twice <expanded> <file> prints "<name>\t<line>" for the first region the expanded text opens a second
# time, or nothing. When <file> is markdown, fenced code blocks are skipped.
generate:twice() {
  local expanded=$1 line fence=false markdown=false number=0
  local -A seen=()
  generate:markdown "$2" && markdown=true
  while IFS= read -r line || [[ -n "${line}" ]]
  do
    number=$(( number + 1 ))
    if [[ "${markdown}" == true && "${line}" =~ ${generate_fence_re} ]]
    then
      generate:flip
      continue
    fi
    [[ "${fence}" == false && "${line}" =~ ${generate_open_re} ]] || continue
    if [[ -n "${seen[${BASH_REMATCH[1]}]+x}" ]]
    then
      printf '%s\t%s\n' "${BASH_REMATCH[1]}" "${number}"
      return 0
    fi
    seen[${BASH_REMATCH[1]}]=1
  done < "${expanded}"
}

# generate:expand <file> <mode> expands the file's regions into a temp file beside it and compares the two. Only
# mode write writes the file. It returns:
#
#   0  nothing changed
#   1  the file was rewritten (mode write) or is stale (mode check)
#   2  the file could not be expanded cleanly, and is left unchanged
generate:expand() {
  local file=$1 mode=${2:-write} target temp markdown=false twice number
  if [[ ! -f "${file}" ]]
  then
    error "generate: no such file: ${file}"
    return 2
  fi
  if grep --quiet --extended-regexp "${generate_conflict_re}" "${file}"
  then
    error "generate: ${file} carries an unresolved conflict; resolve it, then run: kit generate"
    return 2
  fi
  target=$(generate:target "${file}")
  temp="${target}.kit-new"
  cp -p "${target}" "${temp}"
  generate:markdown "${file}" && markdown=true
  if ! generate:unit "${file}" "${temp}" "${file}" "${file}" "${markdown}"
  then
    rm -f "${temp}"
    return 2
  fi
  IFS=$'\t' read -r twice number < <(generate:twice "${temp}" "${file}")
  if [[ -n "${twice}" ]]
  then
    error "${file}: the region '${twice}' is opened twice, the second time at line ${number} of the expanded file"
    rm -f "${temp}"
    return 2
  fi
  if cmp --silent "${file}" "${temp}"
  then
    rm -f "${temp}"
    return 0
  fi
  if [[ "${mode}" == check ]]
  then
    error "${file}: stale; run: kit generate"
    rm -f "${temp}"
    return 1
  fi
  mv "${temp}" "${target}"
  step "${file}: regenerated"
  return 1
}

# generate:run [--check] [--iterations=<n>] [<path>…] is the driver:
#
#   - Named files get one pass, or up to <n>.
#   - With no paths, every file containing a directive is expanded until a pass changes nothing, for at most 20
#     passes or <n>. More than one pass can be needed because a command may read another file's region.
#   - --check runs one pass without writing and fails naming every stale file.
#
# A file that cannot be expanded cleanly fails the run in every mode.
generate:run() {
  local check=false limit=0 explicit=false arg file rc pass=0 changed failed stale
  local -a files=() targets=()
  for arg in "$@"
  do
    case "${arg}" in
      (--check) check=true ;;
      (--iterations=*) limit=${arg#*=} ;;
      (*) files+=("${arg}"); explicit=true ;;
    esac
  done
  if (( ${#files[@]} ))
  then readarray -t targets < <(generate:discover "${files[@]}")
  else readarray -t targets < <(generate:discover)
  fi
  if (( ${#targets[@]} == 0 ))
  then
    ok "generate: no kit:generate directive to expand"
    return 0
  fi
  if [[ "${check}" == true ]]
  then
    stale=0
    failed=0
    for file in "${targets[@]}"
    do
      generate:expand "${file}" check && rc=0 || rc=$?
      (( rc == 2 )) && failed=$(( failed + 1 ))
      (( rc == 1 )) && stale=$(( stale + 1 ))
    done
    if (( failed > 0 ))
    then
      error "generate: ${failed} file(s) could not be expanded; see above"
      return 1
    fi
    if (( stale > 0 ))
    then
      error "generate: ${stale} of ${#targets[@]} generated file(s) are stale; run: kit generate"
      return 1
    fi
    ok "generate: ${#targets[@]} generated file(s), every region as its command makes it"
    return 0
  fi
  if (( limit == 0 )) && [[ "${explicit}" == true ]]
  then limit=1
  elif (( limit == 0 ))
  then limit=20
  fi
  while (( pass < limit ))
  do
    pass=$(( pass + 1 ))
    changed=0
    failed=0
    for file in "${targets[@]}"
    do
      generate:expand "${file}" write && rc=0 || rc=$?
      (( rc == 2 )) && failed=$(( failed + 1 ))
      (( rc == 1 )) && changed=$(( changed + 1 ))
    done
    if (( failed > 0 ))
    then
      error "generate: ${failed} file(s) could not be expanded; see above"
      return 1
    fi
    if (( changed == 0 ))
    then
      ok "generate: ${#targets[@]} generated file(s), settled in ${pass} pass(es)"
      return 0
    fi
  done
  if [[ "${explicit}" == true ]]
  then
    ok "generate: ${#targets[@]} file(s) expanded in ${pass} pass(es)"
    return 0
  fi
  error "generate: still changing after ${limit} passes; a command's output depends on its own output." \
    "Raise the bound with --iterations=<n>, or fix the command."
  return 1
}

# ----------------------------------------------------------------------------------------------------------------------
