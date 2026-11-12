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
#                         the Windows host, terminal:windows-host; else unknown
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
  elif [[ -n "${WSL_DISTRO_NAME:-}" ]]
  then
    terminal:windows-host
  else
    printf 'unknown\n'
  fi
}

# terminal:windows-host prints the key of the window WSL runs in, from Windows' default terminal: Windows hands a
# console session to it without setting WT_SESSION. The Windows console's GUID names the console; none, or zeros, is
# Windows' own choice, Windows Terminal from Windows 11 (build 22000) and the console before; any other GUID is a
# terminal Windows hands sessions to, Windows Terminal. Where reg.exe cannot run, as with interop off, or does not
# answer within a second, it is unknown.
terminal:windows-host() {
  local answer delegation build status=0
  if ! command -v reg.exe >/dev/null 2>&1
  then
    printf 'unknown\n'
    return 0
  fi
  answer=$(terminal:timed 1 reg.exe query 'HKCU\Console\%%Startup' /v DelegationTerminal) || status=$?
  if (( status == 124 ))
  then
    printf 'unknown\n'
    return 0
  fi
  delegation=$(tr -d '\r' <<<"${answer}" | awk '$1 == "DelegationTerminal" { print $3 }')
  case "${delegation}" in
    ('{B23D10C0-E52E-411E-9D5B-C09FDF709C7D}') printf 'console\n' ;;
    ('' | '{00000000-0000-0000-0000-000000000000}')
      build=$(terminal:timed 1 terminal:windows-version | tr -d '\r' \
        | sed -n 's/.*Version [0-9]*\.[0-9]*\.\([0-9]*\).*/\1/p')
      if (( ${build:-0} >= 22000 ))
      then
        printf 'wt\n'
      else
        printf 'console\n'
      fi
      ;;
    (*) printf 'wt\n'
  esac
}

# terminal:windows-version prints what cmd.exe /c ver says, run from /mnt/c so cmd.exe does not warn of a path it
# cannot use.
terminal:windows-version() {
  cd /mnt/c 2>/dev/null || true  # outside WSL's /mnt/c, cmd.exe runs where it is, and only warns
  cmd.exe /c ver
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
# when it outlasts the seconds, it is stopped, with every process under it, and terminal:timed returns 124 and prints
# nothing. The command reads no stdin, its stderr is dropped, and its TMPDIR is inside terminal:timed's own scratch
# directory, so nothing it leaves holds kit's output open, or stays behind in the person's TMPDIR, once it is stopped.
terminal:timed() {
  local seconds=$1 dir pid watcher status=0
  shift
  file:scratch dir || return 1
  mkdir -p "${dir}/tmp"
  TMPDIR=${dir}/tmp "$@" > "${dir}/out" 2>/dev/null </dev/null &
  pid=$!
  # the watcher marks the run as outlasted before it stops it, so a command that ends as the watcher wakes is never
  # taken for one that was stopped
  ( sleep "${seconds}"; : > "${dir}/outlasted"; terminal:stop "${pid}" ) >/dev/null 2>&1 </dev/null &
  watcher=$!
  wait "${pid}" 2>/dev/null || status=$?
  if [[ -e "${dir}/outlasted" ]]
  then
    wait "${watcher}" 2>/dev/null
    return 124
  fi
  terminal:stop "${watcher}"
  wait "${watcher}" 2>/dev/null
  cat "${dir}/out"
  return "${status}"
}

# terminal:stop <pid> stops the process and every process under it, deepest first, as ps lists them.
terminal:stop() {
  local child
  for child in $(ps -A -o pid= -o ppid= 2>/dev/null | awk -v parent="$1" '$2 == parent { print $1 }')
  do
    terminal:stop "${child}"
  done
  kill -TERM "$1" 2>/dev/null || true  # a process that ended already is stopped
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
# reader's object <settings>. A bad that answers null reads a setting the reader did not give, so it is false: kit
# fails a finding only on what it read. A bad that does not compile, or answers anything else, is kit's bug: it says
# so and returns 2.
terminal:bad() {
  local key=$1 settings=$2 id bad answer out='{}'
  while IFS=$'\t' read -r id bad
  do
    answer=$(jq --null-input --argjson s "${settings}" "\$s | (${bad})" 2>/dev/null) || answer=''
    [[ "${answer}" != null ]] || answer=false
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

# terminal:ask asks the terminal, on /dev/tty, in one write: does it speak the kitty keyboard protocol (CSI ? u); does
# it measure by grapheme (DECRQM 2027); where is the cursor after 📚, and after ↑ (CSI 6n), each printed at the line's
# start; then it clears the line, and asks for the primary attributes (CSI c), which every terminal answers, so their
# answer ends the exchange. It prints what the terminal answered within half a second, as terminal:collect reads it,
# and puts the terminal back as it was.
terminal:ask() {
  local tty saved reply
  exec {tty}<>/dev/tty 2>/dev/null || return 0
  if ! saved=$(stty -g <&"${tty}" 2>/dev/null)
  then
    exec {tty}>&-
    return 0
  fi
  stty raw -echo <&"${tty}"
  # shellcheck disable=SC2016  # $p is DECRQM's final byte, not an expansion
  printf '\e[?u\e[?2027$p\r📚\e[6n\r↑\e[6n\r\e[2K\e[c' >&"${tty}"
  reply=$(terminal:collect "${tty}" 500 2000)
  stty "${saved}" <&"${tty}"
  exec {tty}>&-
  printf '%s' "${reply}"
}

# terminal:collect <fd> <judge-ms> <drain-ms> reads the terminal's answers from <fd> until the answer to the primary
# attributes, ESC [ ? … c, comes. Where it came within <judge-ms> milliseconds, it prints all that came, what the
# person typed before it among it; later, it prints nothing, as a terminal that did not answer, and reads on until that
# answer or <drain-ms>, so a late answer never lands in the person's shell.
terminal:collect() {
  local fd=$1 judge=$2 drain=$3 got='' chunk start elapsed status
  local end_re=$'\e''\[\?[0-9;]*c'
  start=${EPOCHREALTIME/[.,]/}
  while true
  do
    chunk=''
    status=0
    IFS= read -r -s -d '' -n 64 -t 0.05 -u "${fd}" chunk || status=$?
    got+=${chunk}
    elapsed=$(( (${EPOCHREALTIME/[.,]/} - start) / 1000 ))
    if [[ "${got}" =~ ${end_re} ]]
    then
      (( elapsed > judge )) || printf '%s' "${got}"
      return 0
    fi
    # the end of the input, or the drain's last moment: nothing more comes
    if (( elapsed > drain )) || { (( status == 1 )) && [[ -z "${chunk}" ]]; }
    then
      return 0
    fi
  done
}

# terminal:parse <reply> prints what the terminal answered, as terminal:ask gives it, as JSON: answered; kitty_keyboard;
# grapheme_width, true where mode 2027 is set, reset, or set for good; emoji_width and ambiguous_width, the columns the
# cursor moved after 📚 and after ↑, null where it did not say; and color, as terminal:color reads it. An empty reply is
# a terminal that did not answer.
terminal:parse() {
  local reply=$1 rest kitty=false grapheme=false
  local -a columns=()
  # shellcheck disable=SC2016  # \$y is DECRPM's final bytes, matched as they are
  local kitty_re=$'\e''\[\?[0-9]+u' mode_re=$'\e''\[\?2027;([0-9])\$y' position_re=$'\e''\[[0-9]+;([0-9]+)R'
  if [[ -z "${reply}" ]]
  then
    printf '{"answered":false}\n'
    return 0
  fi
  [[ ! "${reply}" =~ ${kitty_re} ]] || kitty=true
  if [[ "${reply}" =~ ${mode_re} ]]
  then
    case "${BASH_REMATCH[1]}" in
      (1 | 2 | 3) grapheme=true
    esac
  fi
  rest=${reply}
  while [[ "${rest}" =~ ${position_re} ]]
  do
    columns+=("$(( BASH_REMATCH[1] - 1 ))")
    rest=${rest#*"${BASH_REMATCH[0]}"}
  done
  jq --null-input --compact-output --argjson kitty "${kitty}" --argjson grapheme "${grapheme}" \
    --argjson emoji "${columns[0]:-null}" --argjson ambiguous "${columns[1]:-null}" --arg color "$(terminal:color)" \
    '{answered: true, kitty_keyboard: $kitty, grapheme_width: $grapheme, emoji_width: $emoji,
      ambiguous_width: $ambiguous, color: $color}'
}

# terminal:color prints the colors kittui gets, truecolor, 256, or fewer, as its library, colorprofile 0.4.3, reads
# them: from the environment as its envColorProfile does, then terminfo's colors where those say more.
terminal:color() {
  local term=${TERM:-} colorterm=${COLORTERM:-} profile=fewer colors
  colorterm=${colorterm,,}
  case "${term}" in
    (*alacritty* | *contour* | *foot* | *ghostty* | *kitty* | *rio* | *st* | *wezterm*)
      printf 'truecolor\n'
      return 0
      ;;
    (tmux* | screen*) profile=256
  esac
  case "${WT_SESSION:+set}:${GOOGLE_CLOUD_SHELL:-}" in
    (set:* | *:1 | *:t | *:T | *:true | *:TRUE | *:True)
      printf 'truecolor\n'
      return 0
  esac
  if [[ "${colorterm}" =~ ^(truecolor|24bit|yes|true)$ && "${term}" != screen* && "${term}" != tmux* \
    || "${term}" == *direct ]]
  then
    printf 'truecolor\n'
    return 0
  fi
  [[ "${term}" != *256color ]] || profile=256
  if [[ "${profile}" == fewer && -n "${term}" && "${term}" != dumb ]]
  then
    colors=$(tput -T "${term}" colors 2>/dev/null) || colors=0
    (( ${colors:-0} < 256 )) || profile=256
  fi
  printf '%s\n' "${profile}"
}

# terminal:probes prints what the terminal answered, as a verb that asked it left it in KIT_TERMINAL_PROBES, or null
# where KIT_TERMINAL_PROBES holds no JSON object, as kit did not ask.
terminal:probes() {
  if [[ -n "${KIT_TERMINAL_PROBES:-}" ]] \
    && jq --exit-status 'type == "object"' <<<"${KIT_TERMINAL_PROBES}" >/dev/null 2>&1
  then
    jq --compact-output . <<<"${KIT_TERMINAL_PROBES}"
  else
    printf 'null\n'
  fi
}

# terminal:export-probes asks the terminal, where stdin and stdout are one and KIT_TERMINAL_PROBES holds no answers
# yet, and exports what it answered in KIT_TERMINAL_PROBES, so kittui and every kit welcome --checks it runs read them
# and never touch the terminal kittui holds. A KIT_TERMINAL_PROBES that is no JSON object is dropped, as if unset. A
# verb calls it in its own shell, where the terminal is still stdin and stdout, before it hands the terminal to kittui.
terminal:export-probes() {
  local probes
  [[ "$(terminal:probes)" == null ]] || return 0
  unset KIT_TERMINAL_PROBES
  [[ -t 0 && -t 1 ]] || return 0
  probes=$(terminal:parse "$(terminal:ask)")
  export KIT_TERMINAL_PROBES="${probes}"
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
