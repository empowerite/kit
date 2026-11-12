# shellcheck shell=bash
# trust.sh - the clones you trust: what one asks kit to run, and the file that records your answer (sourced module)
# ----------------------------------------------------------------------------------------------------------------------
# kit runs nothing in a clone until you trust it. modules/bash/1st.sh holds the check, kit:trust-here, which every
# verb's kit:enter makes before anything else, and bin/kit, the loader, makes the same check in POSIX sh before it
# runs a release the clone names. This module holds what kit trust adds: what a clone asks kit to run, and the
# records it writes.
#
# A record is one line of the trust file, kit:trust-file. For a clone, which trusts it and every worktree of it, it is
# the sha256 of what the clone asks kit to run, kit:trust-digest's, two spaces, and the clone's absolute path, as
# sha256sum prints a file's: the trust holds while the clone asks the same, and a clone whose kit.yml, rules, or
# styles come to ask something new is asked about again. A clone may have a line for each thing it asked that you
# trusted, as two worktrees on different branches do. For a folder it is the folder's absolute path alone, ending in
# /, which trusts every clone under it, whatever it asks. A line that starts with # is a comment, and ~/ at the start
# of a path is the home directory. kit trust writes physical paths, with no link in them, as the check compares them.
#
# see: bin/kit-trust, modules/bash/1st.sh, bin/kit

use yaml file

# trust:asks <clone> prints what the clone at <clone> asks kit to run, one line each: the release it pins, and whether
# that is the clone's own kit/; each bash_path entry that loads bash from outside the release; its gate; each source
# it brings packages from; the tools it has kit download and run; and each lint: command a rule or a style names, of
# the kit's packages or the clone's own, once each. It prints nothing for a clone that asks for none of them. It reads
# kit.yml and the frontmatter with yq, and runs nothing the clone names.
trust:asks() {
  local clone=$1 version entry gate file lint
  local -a entries=() sources=() tools=() files=()
  local -A said=()
  if [[ -f "${clone}/kit.yml" ]]
  then
    version=$(yaml:value "${clone}/kit.yml" version) || return 1
    if [[ "${version}" == unreleased && -x "${clone}/kit/bin/kit" && -f "${clone}/kit/modules/bash/1st.sh" ]]
    then
      printf 'run its own kit, kit/ in the clone, for every kit command there (version: unreleased)\n'
    elif [[ -n "${version}" && "${version}" != unreleased ]]
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
    list:read tools yaml:eval "${clone}/kit.yml" tool-names '(.tools // {}) | keys | .[]' || return 1
    if (( ${#tools[@]} ))
    then
      printf 'download and run the tools it pins, from the URLs it names: %s (tools)\n' "$(trust:joined "${tools[@]}")"
    fi
  fi
  for file in "${clone}"/.kit/{rules,styles}/*.md "${clone}"/.kit/packages/{,*/}*/files/repo/.kit/{rules,styles}/*.md
  do
    [[ ! -f "${file}" ]] || files+=("${file}")
  done
  for file in "${files[@]}"
  do
    lint=$(yaml:value "${file}" lint) || return 1
    [[ -n "${lint}" && -z "${said[${lint}]+x}" ]] || continue
    said[${lint}]=1
    printf 'run %s for kit lint (lint: in %s)\n' "${lint}" "${file#"${clone}"/}"
  done
}

# trust:joined <word>… prints the words joined by a comma and a space.
trust:joined() {
  local IFS=,
  local joined="$*"
  printf '%s\n' "${joined//,/, }"
}

# trust:line <record> [<digest>] prints the line that records <record>: <digest>, two spaces, and <record> for a
# clone, and <record> alone for a folder.
trust:line() {
  if [[ -n "${2:-}" ]]
  then printf '%s  %s\n' "$2" "$1"
  else printf '%s\n' "$1"
  fi
}

# trust:names <name> <line> puts in the variable <name> the path a line of the trust file names, without the digest
# a clone's line starts with.
trust:names() {
  if [[ "$2" =~ ^[0-9a-f]{64}\ \ (.*)$ ]]
  then printf -v "$1" '%s' "${BASH_REMATCH[1]}"
  else printf -v "$1" '%s' "$2"
  fi
}

# trust:recorded <record> [<digest>] succeeds when the trust file holds trust:line's line for them, as a line of its
# own.
trust:recorded() {
  local file
  file=$(kit:trust-file)
  [[ -f "${file}" ]] && grep --quiet --fixed-strings --line-regexp -- "$(trust:line "$@")" "${file}"
}

# trust:record <record> [<digest>] adds trust:line's line to the trust file, once, writing the file whole or not at
# all. A line of the clone's path alone, which kit trust wrote before it kept the digest, goes.
trust:record() {
  local file
  file=$(kit:trust-file)
  trust:recorded "$@" && return 0
  file:replace "${file}" trust:with "${file}" "$1" "$(trust:line "$@")"
}

# trust:revoke <record> takes every line for <record> out of the trust file, writing the file whole or not at all. It
# fails when the file holds none.
trust:revoke() {
  local file line name found=false
  file=$(kit:trust-file)
  [[ -f "${file}" ]] || return 1
  while IFS= read -r line || [[ -n "${line}" ]]
  do
    trust:names name "${line}"
    [[ "${name}" != "$1" ]] || found=true
  done < "${file}"
  [[ "${found}" == true ]] || return 1
  file:replace "${file}" trust:without "${file}" "$1"
}

# trust:with <file> <record> <line> prints <file>'s lines, if it is there, but a line of <record> alone, then <line>.
trust:with() {
  local line
  if [[ -f "$1" ]]
  then
    while IFS= read -r line || [[ -n "${line}" ]]
    do
      [[ "${line}" == "$2" && "$2" != */ ]] || printf '%s\n' "${line}"
    done < "$1"
  fi
  printf '%s\n' "$3"
}

# trust:without <file> <record> prints <file>'s lines but those for <record>.
trust:without() {
  local line name
  while IFS= read -r line || [[ -n "${line}" ]]
  do
    trust:names name "${line}"
    [[ "${name}" == "$2" ]] || printf '%s\n' "${line}"
  done < "$1"
}

# trust:list prints the clones and folders the records name, one per line, once each, comments and blank lines left
# out.
trust:list() {
  local file line name
  local -A listed=()
  file=$(kit:trust-file)
  [[ -f "${file}" ]] || return 0
  while IFS= read -r line || [[ -n "${line}" ]]
  do
    [[ -n "${line//[[:space:]]/}" && "${line}" != '#'* ]] || continue
    trust:names name "${line}"
    [[ -z "${listed[${name}]+x}" ]] || continue
    listed[${name}]=1
    printf '%s\n' "${name}"
  done < "${file}"
}

# trust:keeping <command>… runs a command that writes what kit itself keeps, its declaration in kit.yml, .kit/manifest,
# and what it places under .kit/, and keeps the clone's trust across it: where a line of the trust file trusts the
# clone as it asks just before, it records the digest of what it asks after, so a fresh clone that kit hydrates, or
# an update that moves its version, is not asked about what kit wrote. A clone a folder trusts, or none, it leaves
# as it is. The command runs in this shell, with trust_keeping set, so one it calls in turn is not kept twice. It
# returns what the command returns.
trust:keeping() {
  # shellcheck disable=SC2034  # read by the command it runs, through bash's dynamic scope
  local trust_keeping=1
  local root name before after status=0
  if ! root=$(kit:clone-root .)
  then
    "$@"
    return
  fi
  name=$(kit:trust-name "${root}")
  before=$(kit:trust-digest "${root}") && trust:recorded "${name}" "${before}" || before=''
  "$@" || status=$?
  if [[ -n "${before}" ]] && after=$(kit:trust-digest "${root}") && [[ "${after}" != "${before}" ]]
  then
    trust:record "${name}" "${after}" || status=1
  fi
  return "${status}"
}

# ----------------------------------------------------------------------------------------------------------------------
