# shellcheck shell=bash
# alacritty.sh - Alacritty's reader: how ⌥ is sent, read from its config file (sourced module)
# ----------------------------------------------------------------------------------------------------------------------
# Alacritty reads the first of these that is there: alacritty/alacritty.toml, then alacritty.toml, under
# XDG_CONFIG_HOME; then the same under ~/.config; ~/.alacritty.toml; and /etc/alacritty/alacritty.toml. Nothing prints
# what it settles on, and alacritty msg config changes it, so the reader reads the one key it needs from the file:
# option_as_alt under [window], or window.option_as_alt at the top. With no file, it is Alacritty's default.
#
#   option    window.option_as_alt: None, OnlyLeft, OnlyRight, or Both; None; null off macOS, where it does nothing
#   config    the file read, else where Alacritty looks first
#
# see: modules/bash/terminal.sh, https://alacritty.org/config-alacritty.html

# terminal/alacritty:settings prints how Alacritty sends ⌥.
terminal/alacritty:settings() {
  local config='' candidate option=None
  for candidate in "${XDG_CONFIG_HOME:-${HOME}/.config}/alacritty/alacritty.toml" \
    "${XDG_CONFIG_HOME:-${HOME}/.config}/alacritty.toml" "${HOME}/.config/alacritty/alacritty.toml" \
    "${HOME}/.alacritty.toml" /etc/alacritty/alacritty.toml
  do
    if [[ -r "${candidate}" ]]
    then
      config=${candidate}
      break
    fi
  done
  if [[ -n "${config}" ]]
  then
    option=$(awk '
      /^[[:space:]]*\[/ { table = $0; gsub(/[][[:space:]]/, "", table); next }
      (table == "window" && /^[[:space:]]*option_as_alt[[:space:]]*=/) \
        || (table == "" && /^[[:space:]]*window\.option_as_alt[[:space:]]*=/) { value = $0 }
      END {
        sub(/^[^=]*=[[:space:]]*/, "", value)
        sub(/[[:space:]]*#.*$/, "", value)
        gsub(/["'\'']/, "", value)
        print value
      }' "${config}")
    [[ -n "${option}" ]] || option=None
  fi
  [[ "${OSTYPE:-}" == darwin* ]] || option=''
  jq --null-input --compact-output --arg option "${option}" \
    --arg config "${config:-${XDG_CONFIG_HOME:-${HOME}/.config}/alacritty/alacritty.toml}" \
    '{option: (if $option == "" then null else $option end), config: $config}'
}

# ----------------------------------------------------------------------------------------------------------------------
