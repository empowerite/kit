# shellcheck shell=bash
# secrets.sh - a key kept in the macOS keychain or the Linux Secret Service, never in argv or a file (sourced module)
# ----------------------------------------------------------------------------------------------------------------------
# An item is named by a service and an account. A secret goes in on stdin and is stored base64-encoded on one line:
# the keychain prints a value that holds a newline as hex, and a PEM holds several, so the encoding keeps what comes
# back what went in. On macOS the secret reaches security through `security -i`, which reads its commands from stdin, so
# it is never in argv, where any process can read it. On Linux it reaches secret-tool on stdin.
#
# The backend is what is on PATH: security, else secret-tool. With neither there is none, and kit keeps no key in a
# file instead.
#
# see: modules/bash/forge.sh

# secrets:backend prints keychain where security is on PATH, else secret-service where secret-tool is, and succeeds.
# With neither it says to install secret-tool, prints nothing, and returns 1.
secrets:backend() {
  if command -v security >/dev/null
  then
    printf 'keychain\n'
  elif command -v secret-tool >/dev/null
  then
    printf 'secret-service\n'
  else
    error 'install secret-tool (libsecret) so kit can keep a key; kit keeps none in a file'
    return 1
  fi
}

# secrets:put <service> <account> stores stdin as the item, base64-encoded, replacing any item of that name, then reads
# the item back and compares it: a backend that exits 0 and keeps nothing is caught here, not at the first use of the
# key. It says which backend failed and returns 1 where there is none, or it refused, or it kept something else.
secrets:put() {
  local backend encoded kept
  backend=$(secrets:backend) || return 1
  encoded=$(base64) || return 1
  encoded=${encoded//$'\n'/}
  case "${backend}" in
    (keychain)
      # base64 holds no space or quote, so the one line security -i reads is safe
      printf 'add-generic-password -U -s %s -a %s -w %s\n' "$1" "$2" "${encoded}" | security -i
      ;;
    (secret-service)
      printf '%s' "${encoded}" | secret-tool store --label="$1 $2" service "$1" account "$2"
  esac || { error "the ${backend} did not store $1 $2"; return 1; }
  kept=$(secrets:raw "${backend}" "$1" "$2") && [[ "${kept}" == "${encoded}" ]] \
    || { error "the ${backend} did not keep $1 $2: it gave back something else"; return 1; }
}

# secrets:raw <backend> <service> <account> prints the item as the backend answers, and returns 1, silent, where there
# is none.
secrets:raw() {
  case "$1" in
    (keychain) security find-generic-password -s "$2" -a "$3" -w 2>/dev/null ;;
    (secret-service) secret-tool lookup service "$2" account "$3" 2>/dev/null
  esac
}

# secrets:get <service> <account> prints the secret secrets:put stored, as it was given. It prints nothing and returns
# 1, silent, where there is no item. The value is base64, which the keychain never hex-encodes, so it is not un-hexed.
secrets:get() {
  local backend stored
  backend=$(secrets:backend) || return 1
  stored=$(secrets:raw "${backend}" "$1" "$2") || return 1
  [[ -n "${stored}" ]] || return 1
  printf '%s\n' "${stored}" | base64 -d
}

# secrets:delete <service> <account> removes the item, and succeeds where there was none. Any other failure is an error
# naming the backend, and returns 1.
secrets:delete() {
  local backend status=0
  backend=$(secrets:backend) || return 1
  case "${backend}" in
    (keychain)
      security delete-generic-password -s "$1" -a "$2" >/dev/null 2>&1 || status=$?
      # 44 is security's answer for an item that is not there
      (( status == 0 || status == 44 )) && return 0
      ;;
    (secret-service)
      # secret-tool clear succeeds where there is nothing to clear
      secret-tool clear service "$1" account "$2" >/dev/null 2>&1 && return 0
  esac
  error "the ${backend} did not delete $1 $2"
  return 1
}

# secrets:foreign <service> <account> prints an item another tool stored, as it stored it: the keychain answers hex
# where the value holds a newline, so a value of hex digits alone is turned back into its bytes. It returns 1,
# silent, where there is no item.
secrets:foreign() {
  local backend value
  backend=$(secrets:backend) || return 1
  value=$(secrets:raw "${backend}" "$1" "$2") || return 1
  if [[ "${backend}" == keychain && "${value}" =~ ^([0-9A-Fa-f]{2})+$ ]]
  then
    printf '%s' "${value}" | xxd -r -p
  else
    printf '%s\n' "${value}"
  fi
}
# ----------------------------------------------------------------------------------------------------------------------
