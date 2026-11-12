# shellcheck shell=bash
# kitty.sh - kitty's reader: the effective config the running kitty wrote for its window (sourced module)
# ----------------------------------------------------------------------------------------------------------------------
# Each running kitty writes the config it runs with, every include read, to effective-config/<its pid> in its cache:
# KITTY_CACHE_DIRECTORY, else ~/Library/Caches/kitty on macOS, else kitty under XDG_CACHE_HOME or ~/.cache. A later
# line wins. kitty puts its pid in KITTY_PID.
#
#   option          macos_option_as_alt: no, left, right, or both; no where it is not set; null off macOS
#   grabbed_left    a mouse_map that grabs a plain left press for kitty, so it never reaches the program
#
# see: modules/bash/terminal.sh, https://github.com/kovidgoyal/kitty/blob/master/kitty/options/definitions.py

# terminal/kitty:settings prints the settings the running kitty wrote, or why it cannot read them, and fails.
terminal/kitty:settings() {
  local cache file mac=false
  if [[ -n "${KITTY_CACHE_DIRECTORY:-}" ]]
  then
    cache=${KITTY_CACHE_DIRECTORY}
  elif [[ "${OSTYPE:-}" == darwin* ]]
  then
    cache=${HOME}/Library/Caches/kitty
  else
    cache=${XDG_CACHE_HOME:-${HOME}/.cache}/kitty
  fi
  [[ "${OSTYPE:-}" != darwin* ]] || mac=true
  file=${cache}/effective-config/${KITTY_PID:-none}
  if [[ -z "${KITTY_PID:-}" || ! -r "${file}" ]]
  then
    printf 'kitty wrote no effective config for this window\n'
    return 1
  fi
  jq --raw-input --slurp --compact-output --argjson mac "${mac}" '
    [split("\n")[] | capture("^(?<key>[a-z_]+)\\s+(?<value>.*)$")] as $lines
    | {option: (if $mac then ([$lines[] | select(.key == "macos_option_as_alt") | .value] | last) // "no"
                else null end),
       grabbed_left: any($lines[]; .key == "mouse_map" and (.value | test("^left\\s+press\\s+grabbed\\s")))}' \
    < "${file}"
}

# ----------------------------------------------------------------------------------------------------------------------
