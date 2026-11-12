# shellcheck shell=bash
# console.sh - the Windows console's reader: how it measures text, read from the registry (sourced module)
# ----------------------------------------------------------------------------------------------------------------------
# The Windows console keeps its settings under HKCU\Console, and a shortcut's own properties, or an executable's key
# under it, override them where kit cannot see; the reader reads the root key alone, as reg.exe gives it inside WSL.
#
#   text_measurement    TextMeasurement: 0 graphemes, 1 wcswidth, 2 console; 0
#
# see: modules/bash/terminal.sh, https://github.com/microsoft/terminal/blob/main/src/propslib/RegistrySerialization.cpp

# terminal/console:settings prints how the Windows console measures text.
terminal/console:settings() {
  local value
  value=$(reg.exe query 'HKCU\Console' /v TextMeasurement 2>/dev/null | tr -d '\r' \
    | awk '$1 == "TextMeasurement" { print $3 }') || value=''
  jq --null-input --compact-output --argjson value "$(( ${value:-0} ))" '{text_measurement: $value}'
}

# ----------------------------------------------------------------------------------------------------------------------
