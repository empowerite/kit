#!/usr/bin/env bats
# terminal.bats - the terminal kit runs in: detected, read, probed, judged, and what the person keeps
# ----------------------------------------------------------------------------------------------------------------------
# Each test runs with every variable that names a terminal unset, as no terminal at all, then sets the ones it needs.
# The tests run in a terminal of their own at a desk, so without this they would judge it.

bats_require_minimum_version 1.5.0

load helper

setup() {
  fresh-repo >/dev/null
  kit_bash_modules=terminal
  export KIT_HOME="${BATS_TEST_TMPDIR}/home"
  mkdir -p "${KIT_HOME}"
  local name
  for name in TMUX STY ZELLIJ SSH_CONNECTION SSH_TTY TERM_PROGRAM ITERM_PROFILE KITTY_PID ALACRITTY_WINDOW_ID \
    WT_SESSION WT_PROFILE_ID WSL_DISTRO_NAME KIT_TERMINAL_PROBES GHOSTTY_BIN_DIR WEZTERM_CONFIG_FILE COLORTERM
  do
    unset "${name}"
  done
  export TERM=xterm
}

@test "a multiplexer comes before the terminal outside it, ssh before the terminal's own variables, else unknown" {
  run kit-bash 'TMUX=/tmp/t TERM_PROGRAM=iTerm.app terminal:detect'
  [ "$output" = tmux ]
  run kit-bash 'SSH_TTY=/dev/ttys1 TERM_PROGRAM=iTerm.app terminal:detect'
  [ "$output" = ssh ]
  run kit-bash 'STY=1.x terminal:detect; ZELLIJ=0 terminal:detect; terminal:detect'
  [ "$output" = $'screen\nzellij\nunknown' ]
}

@test "over ssh the checks are the heading alone, which holds and says kit cannot read the settings from here" {
  run kit-bash 'SSH_TTY=/dev/ttys1 terminal:checks'
  [ "$status" -eq 0 ]
  [ "$(jq -c '[.id, .ok]' <<<"$output")" = '["terminal-ssh",true]' ]
  [[ "$(jq -r .line <<<"$output")" == "The terminal over ssh: kit cannot read its settings from this machine"* ]]
}

@test "terminal:timed gives what the command prints, and 124 when it outlasts the seconds, without waiting it out" {
  run kit-bash 'terminal:timed 1 echo hi'
  [ "$status" -eq 0 ]
  [ "$output" = hi ]
  SECONDS=0
  run kit-bash 'terminal:timed 1 sleep 5'
  [ "$status" -eq 124 ]
  (( SECONDS < 3 ))
}

@test "every finding's bad in terminals.json is a jq expression that compiles" {
  local bad
  while IFS= read -r bad
  do
    run jq --null-input --argjson s '{}' "\$s | (${bad})"
    [ "$status" -ne 3 ] || { echo "does not compile: ${bad}"; return 1; }
  done < <(jq -r '.terminals[].findings[]?.bad' "${kit_root}/modules/jq/terminals.json")
}

@test "terminal:bad answers each finding of a terminal from the reader's object" {
  run kit-bash 'terminal:bad iterm "{\"triggers\": 2, \"triggers_interactive\": true}"'
  [ "$status" -eq 0 ]
  [ "$(jq -c .triggers <<<"$output")" = true ]
  run kit-bash 'terminal:bad iterm "{\"triggers\": 0, \"triggers_interactive\": true}"'
  [ "$(jq -c .triggers <<<"$output")" = false ]
}

@test "a finding whose bad answers null, as for a setting the reader did not give, holds" {
  run kit-bash 'terminal:bad iterm "{\"triggers\": 2, \"triggers_interactive\": true}" | jq -c ".\"alternate-screen\""'
  [ "$status" -eq 0 ]
  [ "$output" = false ]
}

@test "kept is empty with no settings file, a keep is written once however often, and an unkeep of none is fine" {
  run kit-bash 'terminal:kept'
  [ "$output" = '[]' ]
  export SSH_TTY=/dev/ttys1
  run kit-bash 'terminal:keep terminal-probe-color; terminal:keep terminal-probe-color; terminal:kept'
  [ "$status" -eq 0 ]
  [ "$(jq -c . <<<"$output")" = '["terminal-probe-color"]' ]
  run kit-bash 'terminal:unkeep terminal-probe-color; terminal:unkeep terminal-probe-color; terminal:kept'
  [ "$status" -eq 0 ]
  [ "$(jq -c . <<<"$output")" = '[]' ]
}

@test "a keep leaves the rest of the person's settings file as it was" {
  mkdir -p "${KIT_HOME}/.kit"
  printf 'toolchain: brew\n' > "${KIT_HOME}/.kit/settings.yml"
  run kit-bash 'SSH_TTY=/dev/ttys1 terminal:keep terminal-probe-color'
  [ "$status" -eq 0 ]
  [ "$(yq '.toolchain' "${KIT_HOME}/.kit/settings.yml")" = brew ]
  [ "$(yq -o=json -I=0 '.terminal.kept' "${KIT_HOME}/.kit/settings.yml")" = '["terminal-probe-color"]' ]
}

@test "a reader that outlasts a second gives a heading that says so, and welcome goes on" {
  mkdir -p bash/terminal
  printf 'bash_path:\n  - kit:modules/bash/\n  - bash/\n' > kit.yml
  printf 'terminal/iterm:settings() { sleep 5; }\n' > bash/terminal/iterm.sh
  SECONDS=0
  run kit-bash 'TERM_PROGRAM=iTerm.app terminal:checks'
  [ "$status" -eq 0 ]
  (( SECONDS < 3 ))
  [ "$(jq -r 'select(.id == "terminal-iterm") | .line' <<<"$output")" \
    = "iTerm2: kit could not read its settings: it took more than a second" ]
}

@test "a reader that fails gives its reason in the heading, and no finding" {
  mkdir -p bash/terminal
  printf 'bash_path:\n  - kit:modules/bash/\n  - bash/\n' > kit.yml
  printf 'terminal/iterm:settings() { echo "no preferences"; return 1; }\n' > bash/terminal/iterm.sh
  run kit-bash 'TERM_PROGRAM=iTerm.app terminal:checks'
  [ "$status" -eq 0 ]
  [ "$(jq -sc 'map(.id)' <<<"$output")" = '["terminal-iterm"]' ]
  [ "$(jq -r .line <<<"$output")" = "iTerm2: kit could not read its settings: no preferences" ]
}

@test "a terminal's answers are read: the kitty protocol, mode 2027, both widths, and the colors from the variables" {
  run kit-bash $'COLORTERM=truecolor terminal:parse \'\e[?1u\e[?2027;2$y\e[5;3R\e[5;2R\e[?62;22c\''
  [ "$status" -eq 0 ]
  [ "$(jq -c '[.answered, .kitty_keyboard, .grapheme_width, .emoji_width, .ambiguous_width, .color]' <<<"$output")" \
    = '[true,true,true,2,1,"truecolor"]' ]
}

@test "a terminal that answers neither kitty nor 2027 and draws 📚 in one cell is read so, with 256 colors from TERM" {
  run kit-bash $'TERM=xterm-256color terminal:parse \'\e[5;2R\e[5;2R\e[?1;2c\''
  [ "$(jq -c '[.answered, .kitty_keyboard, .grapheme_width, .emoji_width, .ambiguous_width, .color]' <<<"$output")" \
    = '[true,false,false,1,1,"256"]' ]
}

@test "mode 2027 counts as measured by grapheme when set, reset, or set for good, not when unknown or off for good" {
  local reply=$'"\e[?2027;${p}\\$y\e[5;3R\e[5;2R\e[?1c"'
  run kit-bash "for p in 0 1 2 3 4; do terminal:parse ${reply} | jq -c .grapheme_width; done"
  [ "$output" = $'false\ntrue\ntrue\ntrue\nfalse' ]
}

@test "no answer is read as a terminal that did not answer" {
  run kit-bash "terminal:parse ''"
  [ "$output" = '{"answered":false}' ]
}

@test "KIT_TERMINAL_PROBES is what was answered when it is a JSON object, and is ignored otherwise" {
  run kit-bash 'KIT_TERMINAL_PROBES="{\"answered\":false}" terminal:probes'
  [ "$output" = '{"answered":false}' ]
  run kit-bash 'for p in nope "" "[]"; do KIT_TERMINAL_PROBES=${p} terminal:probes; done'
  [ "$output" = $'null\nnull\nnull' ]
}

@test "with no terminal, kit asks nothing and exports nothing" {
  run kit-bash 'terminal:export-probes </dev/null; echo "${KIT_TERMINAL_PROBES-unset}"'
  [ "$status" -eq 0 ]
  [ "$output" = unset ]
}

@test "the answers in KIT_TERMINAL_PROBES are judged: one cell for 📚 fails the emoji width" {
  export SSH_TTY=/dev/ttys1
  export KIT_TERMINAL_PROBES='{"answered":true,"kitty_keyboard":true,"grapheme_width":true,"emoji_width":1}'
  run kit-bash 'terminal:checks'
  [ "$status" -eq 0 ]
  [ "$(jq -sc 'map(select(.id == "terminal-probe-emoji-width") | .ok)' <<<"$output")" = '[false]' ]
}

@test "a reader's object is judged: a profile with triggers fails the triggers finding" {
  mkdir -p bash/terminal
  printf 'bash_path:\n  - kit:modules/bash/\n  - bash/\n' > kit.yml
  cat > bash/terminal/iterm.sh <<'EOF'
terminal/iterm:settings() { echo '{"profile": "Work", "triggers": 1, "triggers_interactive": true}'; }
EOF
  run kit-bash 'TERM_PROGRAM=iTerm.app terminal:checks'
  [ "$status" -eq 0 ]
  [ "$(jq -sc 'map(select(.id == "terminal-iterm" or .id == "terminal-iterm-triggers") | [.id, .ok])' <<<"$output")" \
    = '[["terminal-iterm",true],["terminal-iterm-triggers",false]]' ]
}

# fakes [<tool> <body>]… writes each tool as a bash script with that body into a directory first on PATH.
fakes() {
  local dir="${BATS_TEST_TMPDIR}/fakes"
  mkdir -p "${dir}"
  while (( $# ))
  do
    printf '#!/usr/bin/env bash\n%s\n' "$2" > "${dir}/$1"
    chmod +x "${dir}/$1"
    shift 2
  done
  [[ ":${PATH}:" == *":${dir}:"* ]] || export PATH="${dir}:${PATH}"
}

# iterm-fakes stands in defaults and plutil for iTerm2's preferences: its profiles are the fixture's, GUID-1 the
# default.
iterm-fakes() {
  export ITERM_FIXTURE="${BATS_TEST_DIRNAME}/fixtures/terminal/iterm-bookmarks.json"
  # shellcheck disable=SC2016  # the fakes' bodies expand when they run
  fakes defaults 'exit 0' plutil 'case "$2" in
  ("New Bookmarks") cat "${ITERM_FIXTURE}" ;;
  ("Default Bookmark Guid") printf GUID-1 ;;
  (*) exit 1
esac'
}

@test "iTerm2's profile named by ITERM_PROFILE is read, and one with every setting as kittui needs has no finding" {
  iterm-fakes
  run kit-bash 'ITERM_PROFILE="Work (copy)" terminal:settings iterm'
  [ "$status" -eq 0 ]
  [ "$(jq -r .profile <<<"$output")" = "Work (copy)" ]
  run kit-bash 'export ITERM_PROFILE="Work (copy)"; terminal:bad iterm "$(terminal:settings iterm)"'
  [ "$(jq -c '[to_entries[] | select(.value) | .key]' <<<"$output")" = '[]' ]
}

@test "with no ITERM_PROFILE, iTerm2's default profile is read, and its defaults fill what it does not set" {
  local want='{"ambiguous_wide":false,"apps_change_keys":true,"encoding":4,"mouse":false,"mouse_clicks":true,'
  want+='"no_alternate_screen":false,"option_left":0,"option_right":0,"profile":"Default","triggers":1,'
  want+='"triggers_interactive":true,"unicode_version":9}'
  iterm-fakes
  run kit-bash 'terminal:settings iterm'
  [ "$(jq -cS . <<<"$output")" = "${want}" ]
  run kit-bash 'terminal:bad iterm "$(terminal:settings iterm)"'
  [ "$(jq -c '[to_entries[] | select(.value) | .key]' <<<"$output")" = '["triggers","option","mouse"]' ]
}

@test "a dynamic profile takes what it does not set from its parent" {
  iterm-fakes
  run kit-bash 'ITERM_PROFILE=Dyn terminal:settings iterm'
  [ "$(jq -c '[.profile, .option_left, .mouse, .triggers_interactive]' <<<"$output")" = '["Dyn",2,true,false]' ]
}

@test "a profile ITERM_PROFILE names that is not there falls back to the default" {
  iterm-fakes
  run kit-bash 'ITERM_PROFILE=Gone terminal:settings iterm'
  [ "$(jq -r .profile <<<"$output")" = Default ]
}

@test "in iTerm2, the checks name the profile, and its triggers fail" {
  iterm-fakes
  run kit-bash 'TERM_PROGRAM=iTerm.app terminal:checks'
  [ "$status" -eq 0 ]
  [ "$(jq -sr '.[0].line' <<<"$output")" = "iTerm2, profile Default" ]
  [ "$(jq -sr 'map(select(.id == "terminal-iterm-triggers"))[0] | [.ok, .fix] | @tsv' <<<"$output")" = \
    "false"$'\t'"iTerm2 → Settings → Profiles → Default → Advanced → Triggers: untick Enabled in interactive apps" ]
}

@test "iTerm2's preferences that cannot be read say so" {
  fakes defaults 'exit 1'
  run kit-bash 'TERM_PROGRAM=iTerm.app terminal:checks'
  [ "$(jq -r .line <<<"$output")" = "iTerm2: kit could not read its settings: kit cannot read iTerm2's preferences" ]
}

# apple-fakes <default-profile> [<key>=<value>]… stands in defaults and plutil for Terminal's preferences: the default
# profile named, and each key given set in it; any other key is not there.
apple-fakes() {
  local profile=$1 pair body
  shift
  body='case "$2" in'$'\n'"  (\"Default Window Settings\") printf '%s' '${profile}' ;;"$'\n'
  for pair in "$@"
  do
    body+="  (\"Window Settings.${profile}.${pair%%=*}\") printf '%s' '${pair#*=}' ;;"$'\n'
  done
  body+='  (*) exit 1'$'\n''esac'
  fakes defaults 'exit 0' plutil "${body}"
}

@test "Terminal's default profile is read, Terminal's own defaults filling what it does not set" {
  apple-fakes Basic
  run kit-bash 'terminal:settings apple'
  [ "$(jq -cS . <<<"$output")" = '{"ambiguous_wide":false,"encoding":4,"option_meta":false,"profile":"Basic"}' ]
  apple-fakes Basic useOptionAsMetaKey=true EastAsianAmbiguousWide=true CharacterEncoding=30
  run kit-bash 'terminal:settings apple'
  [ "$(jq -cS . <<<"$output")" = '{"ambiguous_wide":true,"encoding":30,"option_meta":true,"profile":"Basic"}' ]
}

@test "in Terminal, ⌥ not Meta fails, the heading says it judged the default profile, and no kitty protocol holds" {
  apple-fakes Basic
  export KIT_TERMINAL_PROBES='{"answered":true,"kitty_keyboard":false,"emoji_width":2,"ambiguous_width":1}'
  run kit-bash 'TERM_PROGRAM=Apple_Terminal terminal:checks'
  [ "$status" -eq 0 ]
  [ "$(jq -sr '.[0].line' <<<"$output")" = "Terminal, its default profile Basic" ]
  [ "$(jq -sc 'map(select(.id == "terminal-apple-option") | .ok)' <<<"$output")" = '[false]' ]
  # with ⌥ as Meta, no finding explains the keys, so the probe says what kittui does instead
  apple-fakes Basic useOptionAsMetaKey=true
  run kit-bash 'TERM_PROGRAM=Apple_Terminal terminal:checks'
  [ "$(jq -sr 'map(select(.id == "terminal-probe-kitty-keyboard"))[0] | [.ok, .line] | @tsv' <<<"$output")" \
    = "true"$'\t'"Terminal has no kitty keyboard protocol, so kittui offers > and < for ⌥ with an arrow" ]
}

@test "a default profile with a dot in its name cannot be read, and the heading says so and holds" {
  apple-fakes Pro.Dark
  run kit-bash 'TERM_PROGRAM=Apple_Terminal terminal:checks'
  [ "$(jq -sc 'map([.id, .ok])' <<<"$output")" = '[["terminal-apple",true]]' ]
  local want="Terminal: kit could not read its settings: its default profile Pro.Dark has a dot in its name,"
  want+=" which kit cannot read"
  [ "$(jq -r .line <<<"$output")" = "${want}" ]
}

# ghostty-fakes <layout> <config-line>… stands in ghostty, whose +show-config prints the lines, and defaults, which
# names the keyboard layout.
ghostty-fakes() {
  local layout=$1
  shift
  printf '%s\n' "$@" > "${BATS_TEST_TMPDIR}/ghostty.config"
  # shellcheck disable=SC2016  # the fakes' bodies expand when they run
  fakes ghostty '[[ "$1" == +show-config ]] && cat "${BATS_TEST_TMPDIR}/ghostty.config"' \
    defaults "printf '%s\n' 'com.apple.keylayout.${layout}'"
}

# found <checks> prints the ids of the findings that failed among the checks on stdin.
found() {
  jq -sc 'map(select((.ok | not) and (.id | startswith("terminal-probe-") | not)) | .id)'
}

@test "Ghostty on a French layout with mouse reporting off fails ⌥, ⌥ with the arrows, and the mouse" {
  ghostty-fakes French 'mouse-reporting = false'
  run kit-bash 'OSTYPE=darwin24 TERM_PROGRAM=ghostty terminal:checks'
  [ "$status" -eq 0 ]
  [ "$(found <<<"$output")" = '["terminal-ghostty-option","terminal-ghostty-alt-arrows","terminal-ghostty-mouse"]' ]
}

@test "Ghostty with ⌥ as Alt, both arrows unbound, and the defaults otherwise has nothing to find" {
  ghostty-fakes French 'macos-option-as-alt = left' 'keybind = alt+left=unbind' 'keybind = alt+right=unbind'
  run kit-bash 'OSTYPE=darwin24 TERM_PROGRAM=ghostty terminal:checks'
  [ "$(found <<<"$output")" = '[]' ]
  [ "$(jq -sr '.[0].line' <<<"$output")" = "Ghostty: nothing in it slows or breaks kittui" ]
}

@test "Ghostty on a U.S. layout sends ⌥ as Alt by default, and legacy widths fail" {
  ghostty-fakes US 'grapheme-width-method = legacy'
  run kit-bash 'OSTYPE=darwin24 TERM_PROGRAM=ghostty terminal:checks'
  [ "$(found <<<"$output")" = '["terminal-ghostty-alt-arrows","terminal-ghostty-grapheme-width"]' ]
}

@test "a Ghostty that does not print its config within a second says so" {
  fakes ghostty 'sleep 5'
  run kit-bash 'TERM_PROGRAM=ghostty terminal:checks'
  [ "$(jq -sr '.[0].line' <<<"$output")" = "Ghostty: kit could not read its settings: it took more than a second" ]
}

# kitty-config <line>… writes the effective config the running kitty, KITTY_PID 42, wrote, in a cache of its own.
kitty-config() {
  export KITTY_CACHE_DIRECTORY="${BATS_TEST_TMPDIR}/kitty" KITTY_PID=42
  mkdir -p "${KITTY_CACHE_DIRECTORY}/effective-config"
  printf '%s\n' "$@" > "${KITTY_CACHE_DIRECTORY}/effective-config/42"
}

@test "kitty's effective config: the last macos_option_as_alt wins, and only a plain left click grabbed fails" {
  kitty-config 'macos_option_as_alt no' 'macos_option_as_alt left' \
    'mouse_map shift+left press grabbed mouse_selection normal'
  run kit-bash 'OSTYPE=darwin24 terminal:checks'
  [ "$(jq -sr '.[0].line' <<<"$output")" = "kitty: nothing in it slows or breaks kittui" ]
  kitty-config 'mouse_map left press grabbed mouse_selection normal'
  run kit-bash 'OSTYPE=darwin24 terminal:checks'
  [ "$(found <<<"$output")" = '["terminal-kitty-option","terminal-kitty-mouse"]' ]
}

@test "a kitty that wrote no effective config for its window says so" {
  export KITTY_CACHE_DIRECTORY="${BATS_TEST_TMPDIR}/kitty" KITTY_PID=7
  run kit-bash 'terminal:checks'
  [ "$(jq -sr '.[0].line' <<<"$output")" = \
    "kitty: kit could not read its settings: kitty wrote no effective config for this window" ]
}

# The show-keys fixture is the example on https://wezterm.org/cli/show-keys.html, with ⌥⏎ added in its shape, as
# WezTerm's default keys bind it.
@test "WezTerm's ⌥⏎ bound to full screen fails, and its fix names the config file" {
  export WEZTERM_FIXTURE="${BATS_TEST_DIRNAME}/fixtures/terminal/wezterm-show-keys.txt"
  # shellcheck disable=SC2016  # the fake's body expands when it runs
  fakes wezterm '[[ "$1" == show-keys ]] && cat "${WEZTERM_FIXTURE}"'
  export WEZTERM_CONFIG_FILE=/home/me/.wezterm.lua
  run kit-bash 'TERM_PROGRAM=WezTerm terminal:checks'
  [ "$status" -eq 0 ]
  [ "$(found <<<"$output")" = '["terminal-wezterm-alt-enter"]' ]
  fakes wezterm 'printf "Default key table\n-----------------\n"'
  export KIT_TERMINAL_PROBES='{"answered":true,"kitty_keyboard":false}'
  run kit-bash 'TERM_PROGRAM=WezTerm terminal:checks'
  [ "$(jq -sr 'map(select(.id == "terminal-probe-kitty-keyboard"))[0].fix' <<<"$output")" = \
    "add config.enable_kitty_keyboard = true to /home/me/.wezterm.lua" ]
}

# alacritty-config <toml>… writes Alacritty's config where it looks first.
alacritty-config() {
  export XDG_CONFIG_HOME="${BATS_TEST_TMPDIR}/config"
  mkdir -p "${XDG_CONFIG_HOME}/alacritty"
  printf '%s\n' "$@" > "${XDG_CONFIG_HOME}/alacritty/alacritty.toml"
}

@test "Alacritty's ⌥ under [window] or as window.option_as_alt is read, and None or no config fails on macOS" {
  alacritty-config '[window]' 'option_as_alt = "OnlyLeft" # left only'
  run kit-bash 'OSTYPE=darwin24 ALACRITTY_WINDOW_ID=1 terminal:checks'
  [ "$(found <<<"$output")" = '[]' ]
  alacritty-config 'window.option_as_alt = "Both"' '[font]' 'size = 12'
  run kit-bash 'OSTYPE=darwin24 ALACRITTY_WINDOW_ID=1 terminal:checks'
  [ "$(found <<<"$output")" = '[]' ]
  alacritty-config '[font]' 'option_as_alt = "Both"'
  run kit-bash 'OSTYPE=darwin24 ALACRITTY_WINDOW_ID=1 terminal:checks'
  [ "$(found <<<"$output")" = '["terminal-alacritty-option"]' ]
  run kit-bash 'OSTYPE=linux-gnu ALACRITTY_WINDOW_ID=1 terminal:checks'
  [ "$(found <<<"$output")" = '[]' ]
}

# warp-settings <toml>… writes Warp's settings where it keeps them on macOS.
warp-settings() {
  mkdir -p "${HOME}/.warp"
  printf '%s\n' "$@" > "${HOME}/.warp/settings.toml"
}

@test "Warp's Meta keys and mouse reporting are read from its settings, and the defaults fail ⌥" {
  warp-settings '[terminal.input]' 'extra_meta_keys = { left_alt = true, right_alt = false }'
  run kit-bash 'OSTYPE=darwin24 TERM_PROGRAM=WarpTerminal terminal:checks'
  [ "$(found <<<"$output")" = '[]' ]
  warp-settings '[terminal]' 'mouse_reporting_enabled = false'
  run kit-bash 'OSTYPE=darwin24 TERM_PROGRAM=WarpTerminal terminal:checks'
  [ "$(found <<<"$output")" = '["terminal-warp-option","terminal-warp-mouse"]' ]
}

# windows-fakes <delegation> <build> stands in reg.exe, which names the default terminal (none for none), and
# cmd.exe, which says the Windows build and where LOCALAPPDATA is, and wslpath, which maps it under BATS_TEST_TMPDIR.
windows-fakes() {
  export WIN_DELEGATION=$1 WIN_BUILD=$2 WIN_LOCAL="${BATS_TEST_TMPDIR}/Local"
  # shellcheck disable=SC2016  # the fakes' bodies expand when they run
  fakes reg.exe 'case "$*" in
  (*Startup*DelegationTerminal*)
    [[ "${WIN_DELEGATION}" != none ]] || exit 1
    printf "\r\nHKEY_CURRENT_USER\\Console\\%%%%Startup\r\n"
    printf "    DelegationTerminal    REG_SZ    %s\r\n" "${WIN_DELEGATION}" ;;
  (*TextMeasurement*)
    [[ -n "${WIN_MEASUREMENT:-}" ]] || exit 1
    printf "\r\nHKEY_CURRENT_USER\\Console\r\n    TextMeasurement    REG_DWORD    %s\r\n" "${WIN_MEASUREMENT}" ;;
  (*) exit 1
esac' cmd.exe 'case "$*" in
  (*ver*) printf "\r\nMicrosoft Windows [Version 10.0.%s.4317]\r\n" "${WIN_BUILD}" ;;
  (*LOCALAPPDATA*) printf "C:\\Users\\me\\AppData\\Local\r\n" ;;
esac' wslpath 'printf "%s\n" "${WIN_LOCAL}"'
}

@test "inside WSL, the default terminal names the window: the console's GUID, else Windows Terminal from build 22000" {
  export WSL_DISTRO_NAME=Ubuntu
  windows-fakes '{B23D10C0-E52E-411E-9D5B-C09FDF709C7D}' 22631
  run kit-bash 'terminal:detect'
  [ "$output" = console ]
  windows-fakes '{00000000-0000-0000-0000-000000000000}' 22631
  run kit-bash 'terminal:detect'
  [ "$output" = wt ]
  windows-fakes none 19045
  run kit-bash 'terminal:detect'
  [ "$output" = console ]
  windows-fakes '{E12CFF52-A866-4C77-9A90-F570A7AA2C6B}' 19045
  run kit-bash 'terminal:detect'
  [ "$output" = wt ]
  path-without reg.exe
  run kit-bash 'terminal:detect'
  [ "$output" = unknown ]
}

@test "Windows Terminal's settings: comments and trailing commas read, the profile found in any case, defaults used" {
  windows-fakes none 22631
  mkdir -p "${WIN_LOCAL}/Packages/Microsoft.WindowsTerminal_8wekyb3d8bbwe/LocalState"
  cp "${BATS_TEST_DIRNAME}/fixtures/terminal/wt-settings.jsonc" \
    "${WIN_LOCAL}/Packages/Microsoft.WindowsTerminal_8wekyb3d8bbwe/LocalState/settings.json"
  run kit-bash 'WT_SESSION=1 WT_PROFILE_ID="{2c4de342-38b7-51cf-b940-2309a097f518}" terminal:checks'
  [ "$status" -eq 0 ]
  [ "$(jq -sr '.[0].line' <<<"$output")" = "Windows Terminal, profile Ubuntu" ]
  [ "$(found <<<"$output")" = '["terminal-wt-kitty-keyboard","terminal-wt-force-vt"]' ]
  run kit-bash 'WT_SESSION=1 terminal:settings wt'
  [ "$(jq -c '[.profile, .kitty_keyboard, .force_vt, .text_measurement]' <<<"$output")" = \
    '["Windows PowerShell",null,true,"graphemes"]' ]
}

@test "the JSONC reader takes out a comma that only comments and blank lines keep from its closing brace" {
  run kit-bash 'use terminal/wt; printf "{\"a\": [1, 2, /* two */\n\n],  // a\n  // b\n\n}\n" | terminal/wt:jsonc | jq -c .'
  [ "$status" -eq 0 ]
  [ "$output" = '{"a":[1,2]}' ]
}

@test "the JSONC reader reads a settings.json of half a megabyte in well under its second" {
  local big="${BATS_TEST_TMPDIR}/big.jsonc"
  {
    printf '{"list": [\n'
    for (( i = 0; i < 5000; i++ ))
    do
      printf '  {"guid": "{%08d-0000-0000-0000-000000000000}", "name": "a // profile, not a comment", }, // %d\n' \
        "${i}" "${i}"
    done
    printf '],\n}\n'
  } > "${big}"
  (( $(wc -c < "${big}") > 500000 ))
  run kit-bash "use terminal/wt; start=\${EPOCHREALTIME/./}; terminal/wt:jsonc < '${big}' | jq '.list | length';
    printf '%d\n' \$(( (\${EPOCHREALTIME/./} - start) / 1000 ))"
  [ "$status" -eq 0 ]
  [ "${lines[0]}" = 5000 ]
  (( lines[1] < 2000 ))
}

@test "the Windows console always fails, and its text measurement of 2 fails too" {
  export WSL_DISTRO_NAME=Ubuntu WIN_MEASUREMENT=0x2
  windows-fakes '{B23D10C0-E52E-411E-9D5B-C09FDF709C7D}' 22631
  run kit-bash 'terminal:checks'
  [ "$(found <<<"$output")" = '["terminal-console-console","terminal-console-text-measurement"]' ]
}

# tabby-config <yaml>… writes Tabby's config.yaml where TABBY_CONFIG_DIRECTORY names.
tabby-config() {
  export TABBY_CONFIG_DIRECTORY="${BATS_TEST_TMPDIR}/tabby"
  mkdir -p "${TABBY_CONFIG_DIRECTORY}"
  printf '%s\n' "$@" > "${TABBY_CONFIG_DIRECTORY}/config.yaml"
}

@test "Tabby: an encrypted config holds with its reason, and the DOM renderer and ⌥ not Meta fail" {
  tabby-config 'encrypted: true'
  run kit-bash 'OSTYPE=darwin24 TERM_PROGRAM=Tabby terminal:checks'
  [ "$(jq -sc 'map([.id, .ok])' <<<"$output")" = '[["terminal-tabby",true]]' ]
  [ "$(jq -r .line <<<"$output")" = "Tabby: kit could not read its settings: Tabby's config is encrypted" ]
  tabby-config 'terminal:' '  frontend: xterm'
  run kit-bash 'OSTYPE=darwin24 TERM_PROGRAM=Tabby terminal:checks'
  [ "$(found <<<"$output")" = '["terminal-tabby-option","terminal-tabby-renderer"]' ]
}

@test "every terminal in terminals.json has its section in docs/terminals.md, which names each of its findings" {
  local doc="${kit_root}/../docs/terminals.md" key anchor id section missing=''
  [ -f "${doc}" ]
  while IFS=$'\t' read -r key anchor
  do
    # a section's anchor is its heading as GitHub makes one: lower case, punctuation dropped, spaces to dashes
    section=$(awk -v anchor="${anchor}" '
      /^## / {
        slug = tolower(substr($0, 4)); gsub(/[^a-z0-9 -]/, "", slug); gsub(/ /, "-", slug); on = (slug == anchor)
      }
      on' "${doc}")
    if [[ -z "${section}" ]]
    then
      missing+=" ${key}:#${anchor}"
      continue
    fi
    while IFS= read -r id
    do
      [[ "${section}" == *"\`${id}\`"* ]] || missing+=" ${key}-${id}"
    done < <(jq -r --arg key "${key}" '.terminals[$key].findings[]?.id' "${kit_root}/modules/jq/terminals.json")
  done < <(jq -r '.terminals | to_entries[] | [.key, .value.anchor] | @tsv' "${kit_root}/modules/jq/terminals.json")
  [ -z "${missing}" ] || { echo "missing:${missing}"; return 1; }
}

# answering <seconds> <text> stands a terminal on a fifo that writes the text after the seconds, then stays open a
# while, as a terminal does; the kit-bash code reads it on fd 3.
answering() {
  mkfifo "${BATS_TEST_TMPDIR}/tty"
  { sleep "$1"; printf '%b' "$2"; sleep 3; } > "${BATS_TEST_TMPDIR}/tty" &
}

@test "an answer within the half second is all read, typed text before it changing nothing" {
  answering 0.2 'cd src\e[?1u\e[5;3R\e[5;2R\e[?62;22c'
  SECONDS=0
  run kit-bash 'exec 3<"${BATS_TEST_TMPDIR}/tty"
    terminal:parse "$(terminal:collect 3 500 2000)" | jq -c "[.answered, .kitty_keyboard, .emoji_width]"'
  [ "$output" = '[true,true,2]' ]
  (( SECONDS < 2 ))
}

@test "an answer after the half second is not judged, and is read to its end, so none of it is left to the shell" {
  answering 0.8 '\e[?1u\e[5;3R\e[5;2R\e[?62;22c'
  run kit-bash 'exec 3<"${BATS_TEST_TMPDIR}/tty"; got=$(terminal:collect 3 500 2000); left=""
    IFS= read -r -d "" -t 0.2 -u 3 left || true; printf "%s|%s" "${#got}" "${#left}"'
  [ "$output" = '0|0' ]
}

@test "a terminal that never answers is read for the drain's seconds, and no longer" {
  answering 5 ''
  SECONDS=0
  run kit-bash 'exec 3<"${BATS_TEST_TMPDIR}/tty"; terminal:collect 3 500 1000 | wc -c'
  [ "$(tr -d ' ' <<<"$output")" = 0 ]
  (( SECONDS < 3 ))
}

@test "a reader cut at a second leaves nothing behind that holds kit's output, and nothing in the person's TMPDIR" {
  mkdir -p bash/terminal
  printf 'bash_path:\n  - kit:modules/bash/\n  - bash/\n' > kit.yml
  cat > bash/terminal/iterm.sh <<'EOF'
terminal/iterm:settings() { touch "${TMPDIR:-/tmp}/kit-reader-left"; local x; x=$(sleep 5 | cat); }
EOF
  export TMPDIR="${BATS_TEST_TMPDIR}/tmp"
  mkdir -p "${TMPDIR}"
  SECONDS=0
  run kit-bash 'TERM_PROGRAM=iTerm.app terminal:checks 2>&1'
  (( SECONDS < 3 ))
  [[ "$output" != *Terminated* ]]
  [ -z "$(find "${TMPDIR}" -name kit-reader-left)" ]
}

@test "the colors are read as kittui's library reads them: the terminal's name, Windows Terminal, COLORTERM, terminfo" {
  run kit-bash 'for t in xterm-ghostty xterm-kitty screen tmux-256color xterm-direct dumb; do
      TERM=${t} terminal:color; done; TERM=xterm WT_SESSION=1 terminal:color
    TERM=screen COLORTERM=truecolor terminal:color; TERM=xterm-256color COLORTERM=yes terminal:color'
  [ "$output" = $'truecolor\ntruecolor\n256\n256\ntruecolor\nfewer\ntruecolor\n256\ntruecolor' ]
}

@test "every verb that hands the terminal to kittui asks it first" {
  local file missing=''
  for file in "${kit_root}"/bin/* "${kit_root}"/modules/bash/*.sh
  do
    grep -q '=\$(tui:binary)' "${file}" || continue
    grep -q 'terminal:export-probes' "${file}" || missing+=" ${file#"${kit_root}"/}"
  done
  [ -z "${missing}" ] || { echo "hands kittui an unasked terminal:${missing}"; return 1; }
}

@test "inside WSL, a reg.exe that hangs makes the window unknown within a second or so" {
  export WSL_DISTRO_NAME=Ubuntu
  fakes reg.exe 'sleep 5'
  SECONDS=0
  run kit-bash 'terminal:detect'
  [ "$output" = unknown ]
  (( SECONDS < 3 ))
}

# ----------------------------------------------------------------------------------------------------------------------
