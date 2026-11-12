# shellcheck shell=bash
# wezterm.sh - WezTerm's reader: its key bindings, as wezterm show-keys prints them (sourced module)
# ----------------------------------------------------------------------------------------------------------------------
# WezTerm's config is Lua, and nothing prints the values it settles on, so the reader reads only what wezterm show-keys
# prints: each binding, one line each, modifiers, key, ->, and the action. Whether WezTerm speaks the kitty keyboard
# protocol is the probe's to find; its fix names the config file, WEZTERM_CONFIG_FILE, which WezTerm sets.
#
#   alt_enter_fullscreen    ⌥⏎ is bound to ToggleFullScreen, as WezTerm's defaults bind it
#   config                  WEZTERM_CONFIG_FILE, else wezterm.lua
#
# see: modules/bash/terminal.sh, https://wezterm.org/cli/show-keys.html

# terminal/wezterm:settings prints WezTerm's bindings kit judges, or why it cannot read them, and fails.
terminal/wezterm:settings() {
  local keys
  if ! keys=$(wezterm show-keys 2>/dev/null)
  then
    printf 'wezterm show-keys did not print its bindings\n'
    return 1
  fi
  jq --raw-input --slurp --compact-output --arg config "${WEZTERM_CONFIG_FILE:-wezterm.lua}" '
    {alt_enter_fullscreen: any(split("\n")[]; test("^\\s+ALT\\s+Enter\\s+->\\s+ToggleFullScreen\\s*$")),
     config: $config}' <<<"${keys}"
}

# ----------------------------------------------------------------------------------------------------------------------
