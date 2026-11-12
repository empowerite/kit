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

# secrets:put <service> <account> stores stdin as the item, base64-encoded, replacing any item of that name. It returns 1
# where there is no backend or the backend refuses.
secrets:put() {
  local backend encoded
  backend=$(secrets:backend) || return 1
  encoded=$(base64 | tr -d '\n') || return 1
  case "${backend}" in
    (keychain)
      # base64 holds no space or quote, so the one line security -i reads is safe
      printf 'add-generic-password -U -s %s -a %s -w %s\n' "$1" "$2" "${encoded}" | security -i
      ;;
    (secret-service)
      printf '%s' "${encoded}" | secret-tool store --label="$1 $2" service "$1" account "$2"
  esac
}

# secrets:get <service> <account> prints the secret secrets:put stored, as it was given. It prints nothing and returns
# 1, silent, where there is no item.
secrets:get() {
  local stored
  stored=$(secrets:foreign "$1" "$2") || return 1
  [[ -n "${stored}" ]] || return 1
  printf '%s\n' "${stored}" | base64 -d
}

# secrets:delete <service> <account> removes the item, and succeeds where there was none.
secrets:delete() {
  local backend
  backend=$(secrets:backend) || return 1
  case "${backend}" in
    (keychain) security delete-generic-password -s "$1" -a "$2" >/dev/null 2>&1 || true ;;
    (secret-service) secret-tool clear service "$1" account "$2" >/dev/null 2>&1 || true
  esac
}

# secrets:foreign <service> <account> prints an item another tool stored, as it stored it: the keychain answers hex
# where the value holds a newline, so a value of hex digits alone is turned back into its bytes. It returns 1,
# silent, where there is no item.
secrets:foreign() {
  local backend value
  backend=$(secrets:backend) || return 1
  case "${backend}" in
    (keychain) value=$(security find-generic-password -s "$1" -a "$2" -w 2>/dev/null) || return 1 ;;
    (secret-service) value=$(secret-tool lookup service "$1" account "$2" 2>/dev/null) || return 1
  esac
  if [[ "${backend}" == keychain && "${value}" =~ ^([0-9A-Fa-f]{2})+$ ]]
  then
    printf '%s' "${value}" | xxd -r -p
  else
    printf '%s\n' "${value}"
  fi
}
# ----------------------------------------------------------------------------------------------------------------------
