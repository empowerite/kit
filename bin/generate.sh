# shellcheck shell=bash
# shellcheck disable=SC2154  # kit_root is exported by bin/kit
# generate.sh - a directive opens a region a command fills: discovery, expansion, and the fixed point (sourced module)
# ----------------------------------------------------------------------------------------------------------------------
#
# A comment line in a file's own syntax that is `:generate:<command>:until:"<pattern>"`, or `:while:"<pattern>"`,
# and nothing else opens a region. The directive line stays. The command runs from the repository root, with the running kit's
# bin/ first on PATH, and its stdout replaces the lines after the directive: up to the first line matching the
# pattern for until, which stays, or every line matching it for while. So a region is a command's output living
# inside a file a person owns, and the file itself says how it is made.
#
# The command is one command and its arguments, with no shell operator, so what makes a region is always a verb,
# a recipe, a target, or a script in the tree, where it can be read, listed, and tested, never a one-liner in a
# comment. A directive inside a fenced code block is not live, so a page may show one.
#
# A file is rewritten only on a clean expansion. A command that fails, an until pattern no later line matches, a
# conflict marker in the file, an empty result from a file that was not, or a command outside its shape leaves
# the file as it was and fails the run, naming the file and the line.
#
# see: bin/kit-generate.sh, bin/kit-region.sh, docs/generate.md

# The directive's shape: a line that is a comment, opened by one of the comment openers the kit knows, carrying
# the directive and nothing before it, so a mention of the syntax inside prose, a code span, or a string in a
# script is not one. Captures: the opener, the command, until or while, and the pattern.
declare -g generate_directive_re='^[[:space:]]*(#|//|--|<!--|/\*|%|;)[[:space:]]*:generate:(.+):(until|while):[[:space:]]*"([^"]+)"'

# The command's shape: words of letters, digits, and the characters a name, a flag, a path, or a key=value
# pair is made of, and no shell operator.
declare -g generate_command_re='^[A-Za-z0-9_./=:@~%,+-]+([[:space:]]+[A-Za-z0-9_./=:@~%,+-]+)*$'

# An unresolved conflict's shape: a line of seven < or > alone or followed by git's label. The middle marker,
# seven =, is also a markdown heading underline and is never written without the opening one, so it is not read.
declare -g generate_conflict_re='^(<{7}|>{7})([[:space:]].*)?$'

# generate:carries <file> is true when the file carries a directive.
generate:carries() {
  [[ -f "$1" ]] && grep --quiet --extended-regexp "${generate_directive_re}" "$1"
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

# generate:expand <file> <mode> expands the file's directives into a sibling temp file and compares. It returns 0
# when nothing changed, 1 when the file was rewritten (mode write) or is stale (mode check), and 2 when it could
# not be cleanly expanded, in which case the file is left as it was. Only mode write ever writes.
generate:expand() {
  local file=$1 mode=${2:-write} target temp line number=0 fence=false seeking='' pattern='' opened=0 command rc
  local -a words
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
  rc=0
  {
    while IFS= read -r line || [[ -n "${line}" ]]
    do
      number=$(( number + 1 ))
      if [[ "${seeking}" == until ]]
      then
        [[ "${line}" =~ ${pattern} ]] || continue
        seeking=''
      elif [[ "${seeking}" == while ]]
      then
        [[ "${line}" =~ ${pattern} ]] && continue
        seeking=''
      fi
      printf '%s\n' "${line}"
      if [[ "${line}" =~ ^[[:space:]]*(\`\`\`|~~~) ]]
      then
        if [[ "${fence}" == true ]]; then fence=false; else fence=true; fi
        continue
      fi
      [[ "${fence}" == false && "${line}" =~ ${generate_directive_re} ]] || continue
      command=${BASH_REMATCH[2]}
      seeking=${BASH_REMATCH[3]}
      pattern=${BASH_REMATCH[4]}
      opened=${number}
      command=${command#"${command%%[![:space:]]*}"}
      command=${command%"${command##*[![:space:]]}"}
      if [[ ! "${command}" =~ ${generate_command_re} ]]
      then
        error "${file}:${number}: a :generate: command is one command and its arguments, with no shell operator:" \
          "${command}"
        rc=2
        break
      fi
      read -r -a words <<<"${command}"
      if ! PATH="${kit_root}/bin:${PATH}" "${words[@]}"
      then
        error "${file}:${number}: the command failed: ${command}"
        rc=2
        break
      fi
    done < "${file}"
    if (( rc == 0 )) && [[ "${seeking}" == until ]]
    then
      error "${file}:${opened}: the region never closes; no later line matches \"${pattern}\""
      rc=2
    fi
  } > "${temp}"
  if (( rc != 0 ))
  then
    rm -f "${temp}"
    return 2
  fi
  if [[ -s "${file}" && ! -s "${temp}" ]]
  then
    error "generate: expanding ${file} produced nothing; left as it was"
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
# passes or <n>. --check runs one pass without writing and fails naming every stale file. A file that could not
# be cleanly expanded fails the run whatever the mode.
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
    ok "generate: no :generate: directive to expand"
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
