#!/usr/bin/env bats
# log.bats - kit logs each call and each MCP request under .kit/logs/, a file a day, kept log_days days, never seen by git
# ----------------------------------------------------------------------------------------------------------------------

bats_require_minimum_version 1.5.0

load helper

setup() {
  fresh-repo
  "${kit_checkout}/bin/kit" install >/dev/null 2>&1
  rm -rf .kit/logs
  printf -v today '%(%Y-%m%d)T' -1
}

# today_log <name> prints the path of today's <name> log.
today_log() { printf '.kit/logs/%s-%s.log\n' "$1" "${today}"; }

@test "a call is logged when it ends: its time, seconds, process, depth, status, directory, and command line" {
  mkdir -p 'a dir'
  (cd 'a dir' && "${kit_checkout}/bin/kit" version >/dev/null)
  run "${kit_checkout}/bin/kit" nosuchverb 'two words'
  [ "$status" -eq 2 ]
  run cat "$(today_log kit)"
  [ "${#lines[@]}" -eq 2 ]
  [[ "${lines[0]}" =~ ^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9:]{8}[-+][0-9]{4}\ seconds=[0-9]+\.[0-9]{3}\ pid=[0-9]+\ depth=0\ status=0\ in=a\\\ dir\ kit\ version$ ]]
  [[ "${lines[1]}" == *" depth=0 status=2 in=. kit nosuchverb two\\ words" ]]
}

@test "kit serve logs its start, each message with its outcome, and its stop; a tool's verb is a call one deeper" {
  printf '%s\n' \
    '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18","capabilities":{}}}' \
    '{"jsonrpc":"2.0","method":"notifications/initialized"}' \
    '{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"lint","arguments":{"args":["--nope","a b"]}}}' \
    '{"jsonrpc":"2.0","id":3,"method":"resources/read","params":{"uri":"kit://1st.md"}}' \
    '{"jsonrpc":"2.0","id":4,"method":"nope"}' \
    | "${kit_checkout}/bin/kit" serve >/dev/null 2>&1
  run cat "$(today_log mcp)"
  [ "${#lines[@]}" -eq 7 ]
  [[ "${lines[0]}" == *" event=start" ]]
  [[ "${lines[1]}" == *" id=1 method=initialize outcome=ok" ]]
  [[ "${lines[2]}" == *" method=notifications/initialized outcome=none" ]]
  [[ "${lines[3]}" == *" id=2 method=tools/call outcome=status=2 kit lint --nope a\\ b" ]]
  [[ "${lines[4]}" == *" id=3 method=resources/read outcome=ok uri=kit://1st.md" ]]
  [[ "${lines[5]}" == *" id=4 method=nope outcome=error=-32601" ]]
  [[ "${lines[6]}" == *" event=stop" ]]
  grep --quiet ' depth=1 status=2 in=. kit lint --nope a\\ b$' "$(today_log kit)"
  grep --quiet ' depth=0 status=0 in=. kit serve$' "$(today_log kit)"
}

@test "git never sees a log, and a repository kit has not hydrated gets none" {
  "${kit_checkout}/bin/kit" version >/dev/null
  [ -f "$(today_log kit)" ]
  [ -z "$(git status --porcelain --untracked-files=all -- .kit/logs)" ]
  fresh-repo other
  printf 'version: unreleased\npackages:\n  - kit\n' > kit.yml
  "${kit_checkout}/bin/kit" version >/dev/null
  [ ! -e .kit/logs ]
}

@test "a day's first line removes the logs older than log_days, and log_days 0 writes none" {
  printf 'log_days: 3\n' >> kit.yml
  mkdir -p .kit/logs
  touch -t 202001010000 .kit/logs/kit-2020-0101.log .kit/logs/mcp-2020-0101.log
  touch .kit/logs/kit-recent.log
  "${kit_checkout}/bin/kit" version >/dev/null
  [ ! -e .kit/logs/kit-2020-0101.log ]
  [ ! -e .kit/logs/mcp-2020-0101.log ]
  [ -f .kit/logs/kit-recent.log ]
  [ -f "$(today_log kit)" ]
  rm -rf .kit/logs
  sed -i.bak 's/^log_days: 3$/log_days: 0/' kit.yml && rm kit.yml.bak
  "${kit_checkout}/bin/kit" version >/dev/null
  [ ! -e .kit/logs ]
}

@test "a log_days that is not a whole number is refused" {
  printf 'log_days: soon\n' >> kit.yml
  run "${kit_checkout}/bin/kit" check
  [ "$status" -eq 1 ]
  [[ "$output" == *"log_days must be a whole number of days, 0 for no logs, not 'soon'"* ]]
}

@test "uninstall takes the logs out with the rest of what kit wrote" {
  "${kit_checkout}/bin/kit" version >/dev/null
  run "${kit_checkout}/bin/kit" uninstall --dry-run
  [[ "$output" == *".kit/logs/: would be removed"* ]]
  run "${kit_checkout}/bin/kit" uninstall
  [ "$status" -eq 0 ]
  [ ! -e .kit/logs ]
}

# ----------------------------------------------------------------------------------------------------------------------
