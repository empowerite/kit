# shellcheck shell=bash
# ghostty.sh - Ghostty's reader: the settings Ghostty prints with +show-config, and the keyboard layout (sourced module)
# ----------------------------------------------------------------------------------------------------------------------
# ghostty +show-config prints, as key = value lines, only what differs from Ghostty's defaults, every config file and
# include read; the last line of a key wins, and keybind may come many times. The ghostty is the one in GHOSTTY_BIN_DIR,
# which Ghostty sets, else the one on PATH. On macOS, Ghostty's default for ⌥ depends on the keyboard layout: Alt on
# the U.S. layouts, a plain key on every other, so the reader also reads the layout.
#
#   option              macos-option-as-alt: true, false, left, or right; null where it is not set, or off macOS
#   layout_us           the layout is U.S. or U.S. International; true off macOS
#   alt_arrows_legacy   on macOS, Ghostty binds ⌥← and ⌥→ to esc:b and esc:f unless both are bound otherwise
#   mouse               mouse-reporting, true
#   grapheme            grapheme-width-method, unicode
#
# see: modules/bash/terminal.sh, https://github.com/ghostty-org/ghostty/blob/main/src/config/Config.zig

# terminal/ghostty:settings prints the settings Ghostty runs with, or why it cannot read them, and fails.
terminal/ghostty:settings() {
  local ghostty=ghostty config mac=false layout=''
  [[ -z "${GHOSTTY_BIN_DIR:-}" ]] || ghostty=${GHOSTTY_BIN_DIR}/ghostty
  if ! config=$("${ghostty}" +show-config 2>/dev/null)
  then
    printf 'ghostty +show-config did not print its config\n'
    return 1
  fi
  if [[ "${OSTYPE:-}" == darwin* ]]
  then
    mac=true
    layout=$(defaults read com.apple.HIToolbox AppleCurrentKeyboardLayoutInputSourceID 2>/dev/null) || layout=''
  fi
  jq --raw-input --slurp --compact-output --argjson mac "${mac}" --arg layout "${layout}" '
    [split("\n")[] | capture("^(?<key>[a-z-]+) = (?<value>.*)$")] as $lines
    | def last($key; $default): ([$lines[] | select(.key == $key) | .value] | last) // $default;
    [$lines[] | select(.key == "keybind") | .value] as $binds
    | {option: (if $mac then last("macos-option-as-alt"; null) else null end),
       layout_us: (if $mac then $layout | IN("com.apple.keylayout.US", "com.apple.keylayout.USInternational-PC")
                   else true end),
       alt_arrows_legacy: ($mac and ((any($binds[]; startswith("alt+left="))
                                      and any($binds[]; startswith("alt+right="))) | not)),
       mouse: (last("mouse-reporting"; "true") == "true"),
       grapheme: last("grapheme-width-method"; "unicode")}' <<<"${config}"
}

# ----------------------------------------------------------------------------------------------------------------------
