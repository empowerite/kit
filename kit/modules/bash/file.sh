# shellcheck shell=bash
# file.sh - writes a file whole or not at all: made beside the path, moved over it only when its producer succeeds (sourced module)
# ----------------------------------------------------------------------------------------------------------------------
# Every file the kit writes in a repository, or under the home directory, goes through here. The content is produced
# into <path>.kit-new.<pid>; when the producer fails, that temporary is removed and <path> is left exactly as it was,
# so a jq that refuses a file, or an awk that stops, never leaves an empty file where the repository's was. The move
# is a rename, so a reader sees the old file or the new one, never a partial one. The temporary is named for the
# process, so two kit commands writing one path at once never remove each other's, and the sweep at exit removes one
# a stopped command left.
#
# Scratch space goes through here too. file:scratch makes a directory under TMPDIR, and the shell's one EXIT trap,
# file:sweep, removes it at exit, with every other directory made so and every command file:at-exit was given. No
# module sets a trap of its own: a second trap … EXIT replaces the first, and what the first swept leaks. A trap a
# verb set before the first call is kept, and runs at the sweep. A subshell sweeps what it made itself, when it ends.
#
# see: modules/bash/place.sh, modules/bash/manifest.sh

# file_scratch holds every directory file:scratch made, file_at_exit every command file:at-exit was given, its words
# quoted for eval, and file_partial every temporary a write made, for the sweep to remove if the write never ended. file_trapped is the id of the shell whose EXIT trap file:sweep is, or empty before the
# first call. A subshell inherits the three, but not the trap's run, so it installs its own, over lists emptied of
# what the parent made, which stays the parent's to sweep.
declare -ag file_scratch=() file_at_exit=() file_partial=()
declare -g file_trapped=''

# file:scratch <name> makes a fresh directory under TMPDIR and puts its path in the variable <name>, registered for
# file:sweep to remove at exit. The first call installs the one EXIT trap. It assigns rather than prints: a command
# substitution runs in a subshell, which sweeps the directory when the substitution ends, so $(…) would hand back a
# path that is already gone. Call it in the main shell.
file:scratch() {
  local made
  made=$(mktemp -d "${TMPDIR:-/tmp}/kit.XXXXXX") || return 1
  file:trap
  file_scratch+=("${made}")
  printf -v "$1" '%s' "${made}"
}

# file:at-exit <command> [<argument>…] registers a command for file:sweep to run at exit, its words expanded now. A
# verb with a cleanup of its own registers it here, rather than setting a trap that would replace file:sweep.
file:at-exit() {
  file:trap
  file_at_exit+=("$(printf '%q ' "$@")")
}

# file:trap installs file:sweep as this shell's EXIT trap, once per shell. An EXIT trap set before it, by a verb for
# a cleanup of its own, is kept: its command is the first the sweep runs. trap -p prints the trap quoted for the
# shell, as "trap -- '<command>' EXIT", so the shell reads it back as words.
file:trap() {
  local -a previous=()
  [[ "${file_trapped}" != "${BASHPID}" ]] || return 0
  [[ -z "${file_trapped}" ]] || file_scratch=() file_at_exit=() file_partial=()
  file_trapped=${BASHPID}
  eval "previous=($(trap -p EXIT))"
  [[ -z "${previous[2]:-}" || "${previous[2]}" == file:sweep ]] || file_at_exit+=("${previous[2]}")
  trap file:sweep EXIT
}

# file:sweep runs every command file:at-exit was given, in order, then removes every directory file:scratch made.
# Each command runs in a subshell of its own: one that fails, or exits, stops nothing, and leaves the status the
# process exits with as it was.
file:sweep() {
  local command
  for command in "${file_at_exit[@]}"
  do
    ( eval "${command}" ) || true
  done
  (( ${#file_partial[@]} == 0 )) || rm -f "${file_partial[@]}" || true
  (( ${#file_scratch[@]} == 0 )) || rm -rf "${file_scratch[@]}" || true
}

# file:partial <name> <path> puts in the variable <name> the temporary a write of <path> uses in this process, and
# registers it for the sweep. A read-only temporary of the old fixed name, which an interrupted run of an older kit
# left, is removed, as it would otherwise stay beside the path. It assigns rather than prints, as file:scratch does,
# so the registration happens in the shell that writes.
file:partial() {
  local made=$2.kit-new.${BASHPID}
  file:trap
  rm -f "$2.kit-new" "${made}"
  file_partial+=("${made}")
  printf -v "$1" '%s' "${made}"
}

# file:replace [--read-only] <path> <command> [<argument>…] runs the command with its stdout to <path>.kit-new, and
# moves that over <path> when the command succeeds, making the directory when it is missing. The new file keeps the
# mode of the file it replaces; with --read-only it has no write bit, as every file kit places under .kit/ has none.
# When the command fails, the temporary is removed, <path> is left as it was, and the command's status is returned.
file:replace() {
  local strip_write=false path partial status=0
  if [[ "$1" == --read-only ]]
  then
    strip_write=true
    shift
  fi
  path=$1
  shift
  mkdir -p "$(dirname "${path}")" || return 1
  file:partial partial "${path}"
  # The copy carries the mode; the redirection below replaces the content.
  [[ ! -f "${path}" ]] || { cp -p "${path}" "${partial}" && chmod u+w "${partial}"; } || return 1
  "$@" > "${partial}" || status=$?
  if (( status != 0 ))
  then
    rm -f "${partial}"
    return "${status}"
  fi
  [[ "${strip_write}" == false ]] || chmod a-w "${partial}"
  mv -f "${partial}" "${path}"
}

# file:copy [--read-only] <source> <path> copies the source over <path> the same way, keeping the source's mode, or
# with the write bit taken off.
file:copy() {
  local strip_write=false src path partial status=0
  if [[ "$1" == --read-only ]]
  then
    strip_write=true
    shift
  fi
  src=$1 path=$2
  mkdir -p "$(dirname "${path}")" || return 1
  file:partial partial "${path}"
  cp "${src}" "${partial}" || status=$?
  if (( status != 0 ))
  then
    rm -f "${partial}"
    return "${status}"
  fi
  [[ "${strip_write}" == false ]] || chmod a-w "${partial}"
  mv -f "${partial}" "${path}"
}

# file:digest [<path>] prints the sha256 of the file, or of stdin without a path, as 64 hex characters, using
# sha256sum or shasum, whichever the machine has.
file:digest() {
  if command -v sha256sum >/dev/null 2>&1
  then sha256sum "$@" | cut -d ' ' -f 1
  else shasum -a 256 "$@" | cut -d ' ' -f 1
  fi
}

# ----------------------------------------------------------------------------------------------------------------------
