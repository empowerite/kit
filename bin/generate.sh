# shellcheck shell=bash
# shellcheck disable=SC2154  # kit_root is exported by bin/kit
# shellcheck disable=SC2016  # the $ in a pattern anchors a line; nothing is meant to expand
# generate.sh - a directive opens a named region a command fills: discovery, expansion, and the fixed point (sourced module)
# ----------------------------------------------------------------------------------------------------------------------
#
# A line that is `kit:generate <name>`, or `kit:generate <name> `<command>``, with nothing but non-alphanumeric
# characters before and after it, opens a region; a line that is `kit:end <name>` framed the same way closes it.
# So the directive sits in any comment syntax, and is read without knowing one: `#`, `//`, `--`, `;`, `%`, `'`,
# `<!-- -->`, `/* */`, and whatever else a language frames a comment with, since every such frame is punctuation,
# and a line of prose has letters before the marker. The directive and the end marker stay; the command's
# standard output replaces the lines between them. With no command, the command is `kit region`, which reads the
# region's name as the path of a region an active module delivers.
#
# The command is a bash command line, pipes and all, run from the repository root with the running kit's bin/
# first on PATH, stdin closed, errexit and pipefail on, and the file and the region's name exported as
# KIT_GENERATE_FILE and KIT_GENERATE_REGION. A literal backtick is the one thing a command cannot carry; $() does.
#
# Every unit is balanced on its own: the file as written, and each command's output. An output may open regions
# of its own, each closed in that same output, and they are expanded in turn, depth first, so a region nests and
# one pass settles a file. An output that closes a region it did not open, or opens one it does not close, holds
# the file. A line that opens with the marker and is not a directive holds the file too, so a misspelled one is
# never silently skipped; a line of prose, with letters before the marker, is not read at all. In markdown, a
# fenced code block is not live, so a page may show a directive whole.
#
# A file is rewritten only on a clean expansion. A command that fails, an unbalanced unit, a malformed marker, a
# name opened twice, or a conflict marker in the file leaves the file as it was and fails the run, naming the
# file and the line.
#
# see: bin/kit-generate.sh, bin/kit-region.sh, docs/generate.md

# The marker's shapes. A line is read as a directive only when nothing but non-alphanumeric characters frames it:
# the comment opener before, the comment closer after. The name is any run without whitespace or a backtick,
# the command everything between the first backtick after the name and the last one before the closer.
declare -g generate_open_re='^[^[:alnum:]]*kit:generate[[:space:]]+([^[:space:]`]+)([[:space:]]+`(.+)`)?[^[:alnum:]]*$'
declare -g generate_end_re='^[^[:alnum:]]*kit:end[[:space:]]+([^[:space:]`]+)[^[:alnum:]]*$'
# A line that opens with a marker is a directive or an error, never prose.
declare -g generate_marker_re='^[^[:alnum:]]*kit:(generate|end)([^[:alnum:]]|$)'
declare -g generate_fence_re='^[[:space:]]*(```|~~~)'

# An unresolved conflict's shape: a line of seven < or > alone or followed by git's label. The middle marker,
# seven =, is also a markdown heading underline and is never written without the opening one, so it is not read.
declare -g generate_conflict_re='^(<{7}|>{7})([[:space:]].*)?$'

# generate:carries <file> is true when the file has a line that opens with a marker.
generate:carries() {
  [[ -f "$1" ]] && grep --quiet --extended-regexp "${generate_marker_re}" "$1"
}

# generate:markdown <file> is true for a markdown file, the one kind where a fenced code block is not live.
generate:markdown() {
  [[ "$1" == *.md || "$1" == *.markdown ]]
}

# generate:discover [<path>…] prints every file under the paths, or the whole repository, that carries a
# directive: tracked files and untracked ones git does not ignore, so a file install just wrote counts before
# it is added. A named file is printed when it carries one, wherever it is.
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

# generate:target <file> prints the file a write lands in: the file, or what a symlink there points at.
generate:target() {
  if [[ -L "$1" ]]
  then readlink -f "$1"
  else printf '%s\n' "$1"
  fi
}

# generate:command <file> <line> <name> <command> <out> runs one region's command with its stdout in <out>.
generate:command() {
  local file=$1 number=$2 name=$3 command=$4 out=$5
  if ! KIT_GENERATE_FILE="${file}" KIT_GENERATE_REGION="${name}" PATH="${kit_root}/bin:${PATH}" \
    bash -o errexit -o pipefail -c "${command}" </dev/null >"${out}"
  then
    error "${file}:${number}: the command failed: ${command}"
    return 2
  fi
}

# generate:unit <in> <out> <file> <where> <markdown> expands one unit, the file or a command's output, from <in>
# into <out>. <where> opens every message: the file's path for the file, and for an output the file, the line
# of the directive, and the command. Two passes: the first holds the unit to its shape, every marker line a
# directive and every region opened and closed here, in order; the second runs each top-level region's command,
# expands the output as a unit of its own, and writes the result between the markers.
generate:unit() {
  local in=$1 out=$2 file=$3 where=$4 markdown=$5
  local -a lines stack opened
  local n i j line fence=false depth name command body result
  mapfile -t lines < "${in}"
  n=${#lines[@]}
  stack=()
  opened=()
  for (( i = 0; i < n; i++ ))
  do
    line=${lines[i]}
    if [[ "${markdown}" == true && "${line}" =~ ${generate_fence_re} ]]
    then
      if [[ "${fence}" == true ]]; then fence=false; else fence=true; fi
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
  : > "${out}"
  fence=false
  i=0
  while (( i < n ))
  do
    line=${lines[i]}
    printf '%s\n' "${line}" >> "${out}"
    if [[ "${markdown}" == true && "${line}" =~ ${generate_fence_re} ]]
    then
      if [[ "${fence}" == true ]]; then fence=false; else fence=true; fi
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
    if ! generate:command "${file}" "$(( i + 1 ))" "${name}" "${command}" "${body}" \
      || ! generate:unit "${body}" "${result}" "${file}" "${where}:$(( i + 1 )) (the output of ${command})" "${markdown}"
    then
      rm -f "${body}" "${result}"
      return 2
    fi
    cat "${result}" >> "${out}"
    rm -f "${body}" "${result}"
    # The old body is skipped to this region's end, by depth, since the first pass proved the nesting.
    depth=1
    for (( j = i + 1; depth > 0; j++ ))
    do
      line=${lines[j]}
      if [[ "${markdown}" == true && "${line}" =~ ${generate_fence_re} ]]
      then
        if [[ "${fence}" == true ]]; then fence=false; else fence=true; fi
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

# generate:twice <file> prints the first region name the file opens twice, outside a markdown fence, or nothing.
generate:twice() {
  local file=$1 line fence=false markdown
  local -A seen=()
  markdown=false
  generate:markdown "$2" && markdown=true
  while IFS= read -r line || [[ -n "${line}" ]]
  do
    if [[ "${markdown}" == true && "${line}" =~ ${generate_fence_re} ]]
    then
      if [[ "${fence}" == true ]]; then fence=false; else fence=true; fi
      continue
    fi
    [[ "${fence}" == false && "${line}" =~ ${generate_open_re} ]] || continue
    if [[ -n "${seen[${BASH_REMATCH[1]}]+x}" ]]
    then
      printf '%s\n' "${BASH_REMATCH[1]}"
      return 0
    fi
    seen[${BASH_REMATCH[1]}]=1
  done < "${file}"
}

# generate:expand <file> <mode> expands the file's regions into a sibling temp file and compares. It returns 0
# when nothing changed, 1 when the file was rewritten (mode write) or is stale (mode check), and 2 when it could
# not be cleanly expanded, in which case the file is left as it was. Only mode write ever writes.
generate:expand() {
  local file=$1 mode=${2:-write} target temp markdown=false twice
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
  twice=$(generate:twice "${temp}" "${file}")
  if [[ -n "${twice}" ]]
  then
    error "${file}: the region '${twice}' is opened twice"
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

# generate:run [--check] [--iterations=<n>] [<path>…] is the driver. Named files are expanded for one pass, or
# <n>; with none named every file carrying a directive is expanded until a pass changes nothing, at most twenty
# passes or <n>, since a command may read another file's region. --check runs one pass without writing and fails
# naming every stale file. A file that could not be cleanly expanded fails the run whatever the mode.
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
  if (( limit == 0 ))
  then
    if [[ "${explicit}" == true ]]; then limit=1; else limit=20; fi
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
