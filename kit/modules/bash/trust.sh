# shellcheck shell=bash
# trust.sh - the clones you trust: what one asks kit to run, and the file that records your answer (sourced module)
# ----------------------------------------------------------------------------------------------------------------------
# kit runs nothing in a clone until you trust it. modules/bash/1st.sh holds the check, kit:trust-here, which every
# verb's kit:enter makes before anything else, and bin/kit, the loader, makes the same check in POSIX sh before it
# runs a release the clone names. This module holds what kit trust adds: what a clone asks kit to run, and the
# records it writes.
#
# A record is one line of the trust file, kit:trust-file: the absolute path of a clone, which trusts it and every
# worktree of it, or of a folder, ending in /, which trusts every clone under it. A line that starts with # is a
# comment, and ~/ at the start of a line is the home directory. kit trust writes physical paths, with no link in
# them, as the check compares them.
#
# see: bin/kit-trust, modules/bash/1st.sh, bin/kit

use yaml file

# trust:asks <clone> prints what the clone at <clone> asks kit to run, one line each: the release it pins, and whether
# that is the clone's own kit/; each bash_path entry that loads bash from outside the release; its gate; and each
# source it brings packages from. It prints nothing for a clone that asks for none of them. It reads kit.yml with yq,
# and runs nothing the clone names.
trust:asks() {
  local clone=$1 version entry gate
  local -a entries=() sources=()
  [[ -f "${clone}/kit.yml" ]] || return 0
  version=$(yaml:value "${clone}/kit.yml" version) || return 1
  if [[ "${version}" == unreleased && -x "${clone}/kit/bin/kit" && -f "${clone}/kit/modules/bash/1st.sh" ]]
  then
    printf 'run its own kit, kit/ in the clone, for every kit command there (version: unreleased)\n'
  elif [[ -n "${version}" ]]
  then
    printf 'run kit %s, downloaded from its release and checked against its digest (version: %s)\n' \
      "${version}" "${version}"
  fi
  list:read entries yaml:list "${clone}/kit.yml" bash_path || return 1
  for entry in "${entries[@]}"
  do
    [[ "${entry}" == kit:* ]] || printf 'load bash into every kit command from %s (bash_path)\n' "${entry}"
  done
  gate=$(yaml:value "${clone}/kit.yml" gate) || return 1
  [[ -z "${gate}" ]] || printf 'run %s for kit gate (gate)\n' "${gate}"
  list:read sources yaml:records "${clone}/kit.yml" sources name || return 1
  for entry in "${sources[@]}"
  do
    printf 'bring packages from the source %s (sources)\n' "${entry}"
  done
}

# trust:recorded <record> succeeds when the trust file holds <record> as a line of its own.
trust:recorded() {
  local file
  file=$(kit:trust-file)
  [[ -f "${file}" ]] && grep --quiet --fixed-strings --line-regexp -- "$1" "${file}"
}

# trust:record <record> adds <record> to the trust file, once, writing the file whole or not at all.
trust:record() {
  local file
  file=$(kit:trust-file)
  trust:recorded "$1" && return 0
  file:replace "${file}" trust:with "${file}" "$1"
}

# trust:revoke <record> takes <record> out of the trust file, writing the file whole or not at all. It fails when the
# file does not hold it.
trust:revoke() {
  local file
  file=$(kit:trust-file)
  trust:recorded "$1" || return 1
  file:replace "${file}" trust:without "${file}" "$1"
}

# trust:with <file> <record> prints <file>'s lines, if it is there, then <record>.
trust:with() {
  [[ ! -f "$1" ]] || cat "$1"
  printf '%s\n' "$2"
}

# trust:without <file> <record> prints <file>'s lines but <record>.
trust:without() {
  grep --invert-match --fixed-strings --line-regexp -- "$2" "$1" || (( $? == 1 ))
}

# trust:list prints the records, one per line, comments and blank lines left out.
trust:list() {
  local file
  file=$(kit:trust-file)
  [[ -f "${file}" ]] || return 0
  grep --invert-match --extended-regexp '^(#|[[:space:]]*$)' "${file}" || true  # a file of comments alone lists none
}

# ----------------------------------------------------------------------------------------------------------------------
