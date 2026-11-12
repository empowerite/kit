# shellcheck shell=bash
# generate.sh - fill each kit:generate region with its command's output, until nothing changes (sourced module)
# ----------------------------------------------------------------------------------------------------------------------
# shellcheck disable=SC2154  # kit_root is exported by modules/bash/1st.sh
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
# skipped.
#
# The directive and end lines stay. The command's standard output replaces the lines between them. With no
# command, the command is `kit region`, which reads the region's name as the path of a region an active package
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
# see: bin/kit-generate, bin/kit-region, docs/generate.md

use file settings render place syntax

# The directive patterns. The name is any run of characters without whitespace or a backtick. The command is
# everything between the first backtick after the name and the last backtick on the line.
declare -g generate_open_re='^[^[:alnum:]]*kit:generate[[:space:]]+([^[:space:]`]+)([[:space:]]+`(.+)`)?[^[:alnum:]]*$'
declare -g generate_end_re='^[^[:alnum:]]*kit:end[[:space:]]+([^[:space:]`]+)[^[:alnum:]]*$'
# A line that starts with a marker is a directive or an error, never prose.
declare -g generate_marker_re='^[^[:alnum:]]*kit:(generate|end)([^[:alnum:]]|$)'
declare -g generate_fence_re='^[[:space:]]*(```|~~~)'

# An unresolved merge conflict: a line of seven < or seven >, alone or followed by git's label. The middle marker
# (seven =) is not matched: it is also a markdown heading underline, and git never writes it without the opening one.
declare -g generate_conflict_re='^(<{7}|>{7})([[:space:]].*)?$'

# generate:carries <file>… prints each named file that has a line that starts with a marker, once, in the order named.
# One grep reads them all, where a grep per file would start a process for each. A template never counts, under
# .kit/templates/ or a package's hosts/: it is the skeleton of a file, and its directive belongs to the file made from
# it. Nor does a path that is not a file, or one named before. It fails only where grep could not run.
generate:carries() {
  local path rc=0
  local -a files=()
  local -A seen=()
  for path in "$@"
  do
    [[ -f "${path}" && -z "${seen[${path}]+x}" ]] || continue
    seen[${path}]=1
    # A pattern costs far more than a glob, so only a path the glob finds near a template is matched against one.
    if [[ "${path}" == *.kit/templates/* || "${path}" == *packages/*/hosts/* ]] \
      && [[ "${path}" =~ (^|/)\.kit/templates/ || "${path}" =~ (^|/)packages/[^/]+(/[^/]+)?/hosts/ ]]
    then
      continue
    fi
    files+=("${path}")
  done
  (( ${#files[@]} )) || return 0
  # A grep that found nothing in its share of the files, or could not read one, exits 1 or 2, which xargs passes on
  # as 1 (BSD) or 123 (GNU). From 125 up, grep was stopped by a signal, or could not run at all.
  printf '%s\0' "${files[@]}" | xargs -0 grep --files-with-matches --extended-regexp -- "${generate_marker_re}" \
    || rc=$?
  (( rc < 125 ))
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
  local -a paths
  if (( $# == 0 ))
  then
    list:read0 paths git ls-files -z --cached --others --exclude-standard || return 1
    generate:carries "${paths[@]}"
    return
  fi
  generate:carries "$@"
}

# generate_body and generate_expanded are the scratch files every region's command and every file's expansion write
# to, made once per process (generate_scratch_pid): a command's output is read whole before the next command runs,
# and an expansion is done with before the next file's starts, so one file serves each.
declare -g generate_body='' generate_expanded='' generate_scratch_pid=''

# generate:scratch makes generate_body and generate_expanded, once per process, and empties generate_expanded.
generate:scratch() {
  if [[ "${generate_scratch_pid}" != "${BASHPID}" ]]
  then
    file:temp generate_body region || return 1
    file:temp generate_expanded expanded || return 1
    generate_scratch_pid=${BASHPID}
  fi
  : > "${generate_expanded}"
}

# generate_regions maps the path of every repository region the active packages deliver to its rendered source, as
# generate:region reads them once per process; generate_regions_read says whether it has.
declare -gA generate_regions=()
declare -g generate_regions_read=false

# generate:region <name> prints the content the active packages deliver as the repository region at <name>, as kit
# region prints it, but in this process: it renders the packages once, on its first call, where a kit region process
# per region would render them again for each. It prints the same refusal as kit region, and returns 1, when no
# active package delivers a region there.
generate:region() {
  local path src row kind text
  local -a packages rows
  if [[ "${generate_regions_read}" == false ]]
  then
    use release
    settings:require || return 1
    list:read packages settings:packages || return 1
    render:all "${kit_root}" "${packages[@]}" || return 1
    list:read rows place:delivered "${kit_root}" repo "${packages[@]}" || return 1
    for row in "${rows[@]}"
    do
      IFS=$'\t' read -r path src <<<"${row}"
      [[ -n "${path}" ]] || continue
      place:kind "${src}" "${path}" kind
      [[ "${kind}" != region ]] || generate_regions[${path}]=${src}
    done
    generate_regions_read=true
  fi
  if [[ -z "${generate_regions[$1]+x}" ]]
  then
    list:read packages settings:packages || return 1
    error "no active package delivers a region at $1 (packages: ${packages[*]}); see kit.yml"
    return 1
  fi
  # The whole file, read in this process: read looks for a NUL, which a region never holds, so it fails at the
  # file's end, having read all of it.
  src=${generate_regions[$1]}
  text=''
  IFS= read -r -d '' text < "${src}" || [[ -r "${src}" ]] || return 1
  printf '%s' "${text}"
}

# generate:command <file> <where> <line> <name> <command> <out> runs one region's command and writes its stdout
# to <out>. On failure it prints an error naming <where> and <line> (the unit and line of the directive). The bare
# command, kit region, runs in this process through generate:region.
generate:command() {
  local file=$1 where=$2 number=$3 name=$4 command=$5 out=$6
  if [[ "${command}" == 'kit region' ]]
  then
    if ! generate:region "${name}" >"${out}"
    then
      error "${where}:${number}: the command failed: ${command}"
      return 2
    fi
    return 0
  fi
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

# generate:unit <in> <out> <file> <where> <markdown> expands one unit (the file, or a command's output) from <in>,
# appending it to <out>. Every message starts with <where>: the file's path for the file, or, for an output, the
# file, the directive's line, and the command. It makes two passes:
#
#   1. check the unit's shape: every marker line is a directive, and every region opens and closes in this unit,
#      in order; and note where each top-level region opens and closes
#   2. run each top-level region's command into generate_body, and append its output, expanded as a unit of its own,
#      between the directive and end lines
#
# A line is matched against a pattern only when it holds the text the pattern needs, and the lines around the regions
# are written a run at a time, so a long file costs a few writes, not one per line. Call generate:scratch first.
generate:unit() {
  local in=$1 out=$2 file=$3 where=$4 markdown=$5
  local -a lines stack opened tops ends names commands
  local n i k line fence=false fenced=0 name command from
  mapfile -t lines < "${in}"
  n=${#lines[@]}
  stack=()
  opened=()
  tops=()
  ends=()
  names=()
  commands=()
  for (( i = 0; i < n; i++ ))
  do
    line=${lines[i]}
    if [[ "${markdown}" == true && ( "${line}" == *'```'* || "${line}" == *'~~~'* ) \
      && "${line}" =~ ${generate_fence_re} ]]
    then
      generate:flip
      [[ "${fence}" == true ]] && fenced=$(( i + 1 ))
      continue
    fi
    [[ "${fence}" == false && "${line}" == *kit:* && "${line}" =~ ${generate_marker_re} ]] || continue
    if [[ "${line}" =~ ${generate_open_re} ]]
    then
      if (( ${#stack[@]} == 0 ))
      then
        tops+=("${i}")
        names+=("${BASH_REMATCH[1]}")
        commands+=("${BASH_REMATCH[3]:-kit region}")
      fi
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
      (( ${#stack[@]} )) || ends+=("${i}")
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
  # Each top-level region: the lines up to its directive as they are, then its command's output, expanded. Its end
  # line starts the next run.
  from=0
  for (( k = 0; k < ${#tops[@]}; k++ ))
  do
    i=${tops[k]}
    printf '%s\n' "${lines[@]:from:i + 1 - from}" >> "${out}"
    command=${commands[k]}
    if ! generate:command "${file}" "${where}" "$(( i + 1 ))" "${names[k]}" "${command}" "${generate_body}" \
      || ! generate:unit "${generate_body}" "${out}" "${file}" "${where}:$(( i + 1 )) (the output of ${command})" \
        "${markdown}"
    then
      return 2
    fi
    from=${ends[k]}
  done
  if (( from < n ))
  then
    printf '%s\n' "${lines[@]:from}" >> "${out}"
  fi
}

# generate:twice <expanded> <file> <name> <line> puts the name of the first region the expanded text opens a second
# time in the variable <name>, and the line it does so at in <line>, or empties both. When <file> is markdown, fenced
# code blocks are skipped. The variables are the caller's, named other than its own locals.
generate:twice() {
  local line fence=false markdown=false at=0
  local -a lines
  local -A seen=()
  generate:markdown "$2" && markdown=true
  printf -v "$3" '%s' ''
  printf -v "$4" '%s' ''
  mapfile -t lines < "$1"
  for line in "${lines[@]}"
  do
    at=$(( at + 1 ))
    if [[ "${markdown}" == true && ( "${line}" == *'```'* || "${line}" == *'~~~'* ) \
      && "${line}" =~ ${generate_fence_re} ]]
    then
      generate:flip
      continue
    fi
    [[ "${fence}" == false && "${line}" == *kit:generate* && "${line}" =~ ${generate_open_re} ]] || continue
    if [[ -n "${seen[${BASH_REMATCH[1]}]+x}" ]]
    then
      printf -v "$3" '%s' "${BASH_REMATCH[1]}"
      printf -v "$4" '%s' "${at}"
      return 0
    fi
    seen[${BASH_REMATCH[1]}]=1
  done
}

# generate:expand <file> <mode> expands the file's regions into a temporary file and compares the two. Only mode
# write writes the file, whole, through file:replace. It returns:
#
#   0  nothing changed
#   1  the file was rewritten (mode write) or is stale (mode check)
#   2  the file could not be expanded cleanly, and is left unchanged
generate:expand() {
  local file=$1 mode=${2:-write} target line markdown=false twice number
  local -a lines
  if [[ ! -f "${file}" ]]
  then
    error "generate: no such file: ${file}"
    return 2
  fi
  mapfile -t lines < "${file}"
  for line in "${lines[@]}"
  do
    if [[ ( "${line}" == '<<<<<<<'* || "${line}" == '>>>>>>>'* ) && "${line}" =~ ${generate_conflict_re} ]]
    then
      error "generate: ${file} carries an unresolved conflict; resolve it, then run: kit generate"
      return 2
    fi
  done
  # A write goes to the file itself, or to the target of a symlink there.
  target=${file}
  if [[ -L "${file}" ]]
  then
    target=$(readlink -f -- "${file}")
  fi
  generate:scratch || return 2
  generate:markdown "${file}" && markdown=true
  generate:unit "${file}" "${generate_expanded}" "${file}" "${file}" "${markdown}" || return 2
  generate:twice "${generate_expanded}" "${file}" twice number
  if [[ -n "${twice}" ]]
  then
    error "${file}: the region '${twice}' is opened twice, the second time at line ${number} of the expanded file"
    return 2
  fi
  if cmp --silent -- "${file}" "${generate_expanded}"
  then
    return 0
  fi
  if [[ "${mode}" == check ]]
  then
    error "${file}: stale; run: kit generate"
    return 1
  fi
  file:replace "${target}" cat "${generate_expanded}" || return 2
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
# A file that cannot be expanded cleanly fails the run in every mode. A named path that is neither a file nor a
# directory returns 2, before anything is expanded, as kit lint refuses it: a typo never passes as a file with no
# directive. kit generate refuses it first, naming it as it was typed, and hands a named directory over as the
# tracked files under it.
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
  for file in "${files[@]}"
  do
    if [[ ! -e "${file}" && ! -L "${file}" ]]
    then
      error "'${file}' is not a file or a directory here"
      return 2
    fi
  done
  if (( ${#files[@]} ))
  then list:read targets generate:discover "${files[@]}" || return 1
  else list:read targets generate:discover || return 1
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

# generate:choices <key> [<answer>…] answers kit syntax --choices for <path>: an open set, every tracked file, named from
# where the command was typed, as kit generate resolves a path; a path it takes need not be among them.
generate:choices() {
  syntax:tracked "$@"
}

# ----------------------------------------------------------------------------------------------------------------------
