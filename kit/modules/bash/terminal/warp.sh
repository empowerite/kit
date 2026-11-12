# shellcheck shell=bash
# warp.sh - Warp's reader: its Meta keys and mouse reporting, read from its settings file (sourced module)
# ----------------------------------------------------------------------------------------------------------------------
# Warp keeps its settings in settings.toml: ~/.warp on macOS, warp-terminal under XDG_CONFIG_HOME or ~/.config
# elsewhere. Warp's source is closed, and its docs do not say whether the file holds every setting or only the changed
# ones, so a key that is not there is Warp's default. View → Toggle Mouse Reporting changes reporting for the session
# without the file, which no reader sees.
#
#   left_meta     [terminal.input] extra_meta_keys' left_alt, false
#   right_meta    its right_alt, false
#   mouse         [terminal] mouse_reporting_enabled, true
#
# see: modules/bash/terminal.sh, https://docs.warp.dev/terminal/settings/all-settings/

# terminal/warp:settings prints Warp's Meta keys and mouse reporting.
terminal/warp:settings() {
  local file
  if [[ "${OSTYPE:-}" == darwin* ]]
  then
    file=${HOME}/.warp/settings.toml
  else
    file=${XDG_CONFIG_HOME:-${HOME}/.config}/warp-terminal/settings.toml
  fi
  [[ -r "${file}" ]] || file=/dev/null
  awk '
    /^[[:space:]]*\[/ { table = $0; gsub(/[][[:space:]]/, "", table); next }
    table == "terminal.input" && /^[[:space:]]*extra_meta_keys[[:space:]]*=/ {
      if (match($0, /left_alt[[:space:]]*=[[:space:]]*(true|false)/)) left = substr($0, RSTART, RLENGTH)
      if (match($0, /right_alt[[:space:]]*=[[:space:]]*(true|false)/)) right = substr($0, RSTART, RLENGTH)
    }
    table == "terminal" && /^[[:space:]]*mouse_reporting_enabled[[:space:]]*=/ { mouse = $0 }
    END {
      printf "{\"left_meta\": %s, \"right_meta\": %s, \"mouse\": %s}\n",
        (left ~ /true/ ? "true" : "false"), (right ~ /true/ ? "true" : "false"), (mouse ~ /false/ ? "false" : "true")
    }' "${file}"
}

# ----------------------------------------------------------------------------------------------------------------------
