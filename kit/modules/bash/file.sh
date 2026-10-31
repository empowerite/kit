# shellcheck shell=bash
# file.sh - writes a file whole or not at all: made beside the path, moved over it only when its producer succeeds (sourced module)
# ----------------------------------------------------------------------------------------------------------------------
#
# Every file the kit writes in a repository, or under the home directory, goes through here. The content is produced
# into <path>.kit-new; when the producer fails, that temporary is removed and <path> is left exactly as it was, so a
# jq that refuses a file, or an awk that stops, never leaves an empty file where the repository's was. The move is a
# rename, so a reader sees the old file or the new one, never a partial one.
#
# see: modules/bash/release.sh

# file:replace [--read-only] <path> <command> [<argument>…] runs the command with its stdout to <path>.kit-new, and
# moves that over <path> when the command succeeds, making the directory when it is missing. The new file keeps the
# mode of the file it replaces; with --read-only it has no write bit, as every file kit places under .kit/ has none.
# When the command fails, the temporary is removed, <path> is left as it was, and the command's status is returned.
file:replace() {
  local strip_write=false path status=0
  if [[ "$1" == --read-only ]]
  then
    strip_write=true
    shift
  fi
  path=$1
  shift
  mkdir -p "$(dirname "${path}")" || return 1
  # A read-only temporary that an interrupted run left behind would refuse the write.
  rm -f "${path}.kit-new"
  # The copy carries the mode; the redirection below replaces the content.
  [[ ! -f "${path}" ]] || { cp -p "${path}" "${path}.kit-new" && chmod u+w "${path}.kit-new"; } || return 1
  "$@" > "${path}.kit-new" || status=$?
  if (( status != 0 ))
  then
    rm -f "${path}.kit-new"
    return "${status}"
  fi
  [[ "${strip_write}" == false ]] || chmod a-w "${path}.kit-new"
  mv -f "${path}.kit-new" "${path}"
}

# file:copy [--read-only] <source> <path> copies the source over <path> the same way, keeping the source's mode, or
# with the write bit taken off.
file:copy() {
  local strip_write=false src path status=0
  if [[ "$1" == --read-only ]]
  then
    strip_write=true
    shift
  fi
  src=$1 path=$2
  mkdir -p "$(dirname "${path}")" || return 1
  rm -f "${path}.kit-new"
  cp "${src}" "${path}.kit-new" || status=$?
  if (( status != 0 ))
  then
    rm -f "${path}.kit-new"
    return "${status}"
  fi
  [[ "${strip_write}" == false ]] || chmod a-w "${path}.kit-new"
  mv -f "${path}.kit-new" "${path}"
}

# ----------------------------------------------------------------------------------------------------------------------
