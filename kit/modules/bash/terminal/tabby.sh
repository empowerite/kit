# shellcheck shell=bash
# tabby.sh - Tabby's reader: how ⌥ is sent and how it draws, read from its config.yaml (sourced module)
# ----------------------------------------------------------------------------------------------------------------------
# Tabby keeps config.yaml in TABBY_CONFIG_DIRECTORY, else its folder of application data: ~/Library/Application
# Support/tabby on macOS, tabby under XDG_CONFIG_HOME or ~/.config elsewhere. It writes only what differs from its
# defaults. With its vault on, the file is encrypted, and kit cannot read it. Tabby sets TERM_PROGRAM on macOS and Linux
# alone: inside WSL it forwards nothing, so kit never knows it there.
#
#   alt_meta      terminal.altIsMeta, false; null off macOS
#   frontend      terminal.frontend: xterm-webgl or xterm, the DOM; xterm-webgl
#   disable_gpu   hacks.disableGPU, false
#
# see: modules/bash/terminal.sh, https://github.com/Eugeny/tabby/blob/master/tabby-terminal/src/config.ts

# terminal/tabby:settings prints how Tabby sends ⌥ and draws, or why it cannot read them, and fails.
terminal/tabby:settings() {
  local dir file mac=false
  if [[ -n "${TABBY_CONFIG_DIRECTORY:-}" ]]
  then
    dir=${TABBY_CONFIG_DIRECTORY}
  elif [[ "${OSTYPE:-}" == darwin* ]]
  then
    dir="${HOME}/Library/Application Support/tabby"
  else
    dir=${XDG_CONFIG_HOME:-${HOME}/.config}/tabby
  fi
  [[ "${OSTYPE:-}" != darwin* ]] || mac=true
  file=${dir}/config.yaml
  [[ -r "${file}" ]] || file=/dev/null
  if [[ "$(yq '.encrypted // false' "${file}" 2>/dev/null)" == true ]]
  then
    printf "Tabby's config is encrypted\n"
    return 1
  fi
  yq --output-format=json --indent=0 '{
      "alt_meta": (.terminal.altIsMeta // false),
      "frontend": (.terminal.frontend // "xterm-webgl"),
      "disable_gpu": (.hacks.disableGPU // false)}' "${file}" \
    | jq --compact-output --argjson mac "${mac}" 'if $mac then . else .alt_meta = null end'
}

# ----------------------------------------------------------------------------------------------------------------------
