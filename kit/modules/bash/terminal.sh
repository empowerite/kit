# shellcheck shell=bash
# terminal.sh - the terminal kit runs in: detected, read, probed, and judged as kit welcome's checks (sourced module)
# ----------------------------------------------------------------------------------------------------------------------
# shellcheck disable=SC2154  # kit_jq is exported by modules/bash/1st.sh
#
# kittui is a full-screen program, and the terminal around it can make it slower or worse without anything in kit
# being wrong. This module names the terminal, reads its settings, asks it what it does, and judges both with
# modules/jq/terminals.json, the one table of every terminal, finding, and probe. It never writes a terminal's
# settings: every fix is the person's to make.
#
#   terminal:detect       the terminal's key, from the environment, the first of these that answers: a multiplexer
#                         (TMUX, STY, ZELLIJ); ssh (SSH_CONNECTION, SSH_TTY); the terminal's own variables; inside WSL,
#                         the Windows host; else unknown
#   terminal:key          terminal:detect's key, or unknown for one terminals.json does not have
#   terminal:timed        runs a command for at most so many seconds
#   terminal:settings     the reader's object for a key: modules/bash/terminal/<key>.sh, terminal/<key>:settings
#   terminal:bad          each finding's answer, from the reader's object
#   terminal:probes       what the terminal answered, or null where kit did not ask
#   terminal:kept         the ids the person keeps
#   terminal:keep         keeps one, in the profile it is judged in
#   terminal:unkeep       undoes a keep
#   terminal:checks       the checks, one JSON object a line, as modules/jq/terminal.jq gives them
#
# A reader prints one JSON object, the settings its findings read, every default filled in, within a second. Where it
# cannot read them it prints why on stdout and fails; terminal:settings says so as {"unreadable": "<why>"}.
#
# The person's keeps are terminal: kept: in ${KIT_HOME:-$HOME}/.kit/settings.yml, each id after @ and its profile where
# the reader gave one, so a keep in one profile never silences another.
#
# see: modules/jq/terminal.jq, modules/bash/welcome.sh, docs/superpowers/specs/2026-11-11-terminal-checks-design.md

use file

# terminal:detect prints the key of the terminal kit runs in.
terminal:detect() {
  if [[ -n "${TMUX:-}" ]]
  then
    printf 'tmux\n'
  elif [[ -n "${STY:-}" ]]
  then
    printf 'screen\n'
  elif [[ -n "${ZELLIJ:-}" ]]
  then
    printf 'zellij\n'
  elif [[ -n "${SSH_CONNECTION:-}" || -n "${SSH_TTY:-}" ]]
  then
    printf 'ssh\n'
  else
    case "${TERM_PROGRAM:-}" in
      (iTerm.app) printf 'iterm\n' ;;
      (Apple_Terminal) printf 'apple\n' ;;
      (ghostty) printf 'ghostty\n' ;;
      (WezTerm) printf 'wezterm\n' ;;
      (WarpTerminal) printf 'warp\n' ;;
      (Tabby) printf 'tabby\n' ;;
      (*) terminal:detect-rest
    esac
  fi
}

# terminal:detect-rest prints the key of a terminal that sets no TERM_PROGRAM kit knows.
terminal:detect-rest() {
  if [[ -n "${KITTY_PID:-}" || "${TERM:-}" == xterm-kitty ]]
  then
    printf 'kitty\n'
  elif [[ -n "${ALACRITTY_WINDOW_ID:-}" || "${TERM:-}" == alacritty ]]
  then
    printf 'alacritty\n'
  elif [[ -n "${WT_SESSION:-}" ]]
  then
    printf 'wt\n'
  else
    printf 'unknown\n'
  fi
}

# terminal:key prints terminal:detect's key, or unknown where terminals.json has no such terminal.
terminal:key() {
  local key
  key=$(terminal:detect)
  jq --exit-status --arg key "${key}" '.terminals | has($key)' "${kit_jq}/terminals.json" >/dev/null \
    || key=unknown
  printf '%s\n' "${key}"
}

# terminal:timed <seconds> <command> [<argument>…] runs the command, prints what it printed, and returns its status;
# when it outlasts the seconds, it is stopped, with its children, and terminal:timed returns 124 and prints nothing.
terminal:timed() {
  local seconds=$1 dir pid watcher status=0
  shift
  file:scratch dir || return 1
  "$@" > "${dir}/out" &
  pid=$!
  # the watcher marks the run as outlasted before it stops it, so a command that ends as the watcher wakes is never
  # taken for one that was stopped
  ( sleep "${seconds}"; : > "${dir}/outlasted"; pkill -TERM -P "${pid}"; kill -TERM "${pid}" ) >/dev/null 2>&1 &
  watcher=$!
  wait "${pid}" || status=$?
  [[ ! -e "${dir}/outlasted" ]] || return 124
  pkill -TERM -P "${watcher}" 2>/dev/null
  kill -TERM "${watcher}" 2>/dev/null
  wait "${watcher}" 2>/dev/null
  cat "${dir}/out"
  return "${status}"
}

# terminal:settings <key> prints the reader's object for the terminal <key>, {"unreadable": "<why>"} where it cannot
# read it, or null for a terminal terminals.json marks unread.
terminal:settings() {
  local key=$1 out status=0
  if jq --exit-status --arg key "${key}" '.terminals[$key].unread' "${kit_jq}/terminals.json" >/dev/null
  then
    printf 'null\n'
    return 0
  fi
  if ! use "terminal/${key}" 2>/dev/null
  then
    terminal:unreadable "kit has no reader for it"
    return 0
  fi
  out=$(terminal:timed 1 "terminal/${key}:settings") || status=$?
  if (( status == 124 ))
  then
    terminal:unreadable "it took more than a second"
  elif (( status != 0 ))
  then
    terminal:unreadable "${out:-its reader failed}"
  elif ! jq --exit-status 'type == "object"' <<<"${out}" >/dev/null 2>&1
  then
    terminal:unreadable "its reader gave no settings"
  else
    printf '%s\n' "${out}"
  fi
}

# terminal:unreadable <why> prints the object that says kit could not read a terminal's settings.
terminal:unreadable() {
  jq --null-input --compact-output --arg why "$1" '{unreadable: $why}'
}

# terminal:bad <key> <settings> prints each finding of the terminal <key> by its id, true where its bad holds of the
# reader's object <settings>. A bad that does not compile, or answers other than true or false, is kit's bug: it
# says so and returns 2.
terminal:bad() {
  local key=$1 settings=$2 id bad answer out='{}'
  while IFS=$'\t' read -r id bad
  do
    answer=$(jq --null-input --argjson s "${settings}" "\$s | (${bad})" 2>/dev/null) || answer=''
    if [[ "${answer}" != true && "${answer}" != false ]]
    then
      error "terminals.json: ${key}'s ${id} has a bad that does not answer true or false: ${bad}"
      return 2
    fi
    out=$(jq --compact-output --arg id "${id}" --argjson answer "${answer}" '.[$id] = $answer' <<<"${out}")
  done < <(jq --raw-output --arg key "${key}" '.terminals[$key].findings[]? | [.id, .bad] | @tsv' \
    "${kit_jq}/terminals.json")
  printf '%s\n' "${out}"
}

# terminal:probes prints what the terminal answered, or null where kit did not ask it.
terminal:probes() {
  printf 'null\n'
}

# terminal:settings-file prints the person's own settings file, where the keeps are.
terminal:settings-file() {
  printf '%s/.kit/settings.yml\n' "${KIT_HOME:-${HOME}}"
}

# terminal:kept prints the ids the person keeps, as a JSON array, empty with no settings file or no keeps in it.
terminal:kept() {
  local file
  file=$(terminal:settings-file)
  if [[ ! -f "${file}" ]]
  then
    printf '[]\n'
    return 0
  fi
  yq --output-format=json --indent=0 '.terminal.kept // []' "${file}"
}

# terminal:kept-id <id> prints the id a keep of <id> is written as: after @ and the profile the terminal is judged in,
# where its reader gives one.
terminal:kept-id() {
  local profile
  profile=$(terminal:settings "$(terminal:key)" \
    | jq --raw-output 'if type == "object" then .profile // empty else empty end')
  printf '%s%s\n' "$1" "${profile:+@${profile}}"
}

# terminal:keep <id> keeps the check <id> in the profile it is judged in, once.
terminal:keep() {
  local file kit_kept
  file=$(terminal:settings-file)
  kit_kept=$(terminal:kept-id "$1")
  export kit_kept
  if [[ -f "${file}" ]]
  then
    file:replace "${file}" yq '.terminal.kept = ((.terminal.kept // []) + [strenv(kit_kept)] | unique)' "${file}"
  else
    file:replace "${file}" yq --null-input '.terminal.kept = [strenv(kit_kept)]'
  fi
}

# terminal:unkeep <id> undoes the keep of <id> in the profile it is judged in; with none, it changes nothing.
terminal:unkeep() {
  local file kit_kept
  file=$(terminal:settings-file)
  [[ -f "${file}" ]] || return 0
  kit_kept=$(terminal:kept-id "$1")
  export kit_kept
  file:replace "${file}" yq '.terminal.kept = ((.terminal.kept // []) - [strenv(kit_kept)])' "${file}"
}

# terminal:checks prints the terminal's checks, one JSON object a line: its heading, each finding, and each probe.
terminal:checks() {
  local key read bad='{}' probes kept
  key=$(terminal:key)
  read=$(terminal:settings "${key}")
  if jq --exit-status 'type == "object" and (has("unreadable") | not)' <<<"${read}" >/dev/null
  then
    bad=$(terminal:bad "${key}" "${read}") || return 2
  fi
  probes=$(terminal:probes)
  kept=$(terminal:kept)
  jq --null-input --compact-output -L "${kit_jq}" --arg key "${key}" --argjson read "${read}" --argjson bad "${bad}" \
    --argjson probes "${probes}" --argjson kept "${kept}" \
    'import "terminal" as terminal; terminal::checks($key; $read; $bad; $probes; $kept)[]'
}

# ----------------------------------------------------------------------------------------------------------------------
