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
    WT_SESSION WT_PROFILE_ID WSL_DISTRO_NAME KIT_TERMINAL_PROBES GHOSTTY_BIN_DIR WEZTERM_CONFIG_FILE
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

@test "a reader's object is judged: a profile with triggers fails the triggers finding" {
  mkdir -p bash/terminal
  printf 'bash_path:\n  - kit:modules/bash/\n  - bash/\n' > kit.yml
  cat > bash/terminal/iterm.sh <<'EOF'
terminal/iterm:settings() { echo '{"profile": "Work", "triggers": 1, "triggers_interactive": true}'; }
EOF
  run kit-bash 'TERM_PROGRAM=iTerm.app terminal:checks'
  [ "$status" -eq 0 ]
  [ "$(jq -sc 'map([.id, .ok])' <<<"$output")" = '[["terminal-iterm",true],["terminal-iterm-triggers",false]]' ]
}

# ----------------------------------------------------------------------------------------------------------------------
