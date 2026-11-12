#!/usr/bin/env bats
# call.bats - libexec/kit-call runs the verb with the terminal, Ctrl-C, a TERM or HUP sent to kit, and its status
# ----------------------------------------------------------------------------------------------------------------------
# The loader execs libexec/kit-call, so the kit process a test starts is the one timeout, a supervisor, or a closed
# terminal signals. Each test runs a release made once for the file, with tests/fakes/fake-verb as its kit fake, in an
# empty git repository, where kit-call logs the call.

bats_require_minimum_version 1.5.0

load helper

setup_file() {
  release="${BATS_FILE_TMPDIR}/release"
  later-release "${release}"
  cp "${BATS_TEST_DIRNAME}/fakes/fake-verb" "${release}/bin/kit-fake"
  export release
}

setup() {
  fresh-repo >/dev/null
  # a clone's .kit/.gitignore that ignores /logs/ is where kit logs
  mkdir .kit && printf '/logs/\n' > .kit/.gitignore
  kit="${release}/bin/kit"
  ended="${BATS_TEST_TMPDIR}/ended"
}

# logged prints the kit log's lines, oldest first.
logged() {
  cat .kit/logs/kit-*.log
}

# at-terminal <input> <command> runs the command in a pseudo-terminal, as tests/helper.bash's pty does, and types
# <input> there a second after it starts, as at a desk. printf reads <input>, so \003 is Ctrl-C. It keeps typing
# nothing for three seconds more, so the command runs with the terminal still open. bats --jobs runs a test in the
# background, where INT is ignored, and a bash started so cannot take it back; perl gives the command the INT a shell at
# a desk has.
at-terminal() {
  local input=$1 command
  printf -v command 'perl -e %q %s' '$SIG{INT} = "DEFAULT"; exec @ARGV or die' "$2"
  if script --version >/dev/null 2>&1
  then
    { sleep 1; printf "${input}"; sleep 3; } | script --quiet --return --command "${command}" /dev/null
  else
    { sleep 1; printf "${input}"; sleep 3; } | script -q /dev/null bash -c "${command}"
  fi
}

# sent <signal> starts kit fake wait in the background, sends <signal> to the kit process once the verb has started,
# and prints kit's status and how many seconds it took to end.
sent() {
  local pid status=0 started
  "${kit}" fake wait "${ended}" > "${BATS_TEST_TMPDIR}/out" &
  pid=$!
  until [[ -s "${BATS_TEST_TMPDIR}/out" ]]
  do
    sleep 0.1
  done
  started=${SECONDS}
  kill -s "$1" "${pid}"
  wait "${pid}" || status=$?
  printf '%s %s\n' "${status}" "$(( SECONDS - started ))"
}

@test "a TERM sent to kit, as timeout sends it, reaches the verb, which ends with its EXIT trap, logged with 143" {
  run sent TERM
  [ "${lines[0]%% *}" -eq 143 ]
  (( ${lines[0]#* } <= 2 ))
  [ "$(cat "${ended}")" = ended ]
  run logged
  [[ "${lines[0]}" == *" status=143 in=. kit fake wait ${ended}" ]]
}

@test "a HUP sent to kit, as a closed terminal sends it, reaches the verb, which ends, logged with 129" {
  run sent HUP
  [ "${lines[0]%% *}" -eq 129 ]
  (( ${lines[0]#* } <= 2 ))
  [ "$(cat "${ended}")" = ended ]
  run logged
  [[ "${lines[0]}" == *" status=129 in=. kit fake wait ${ended}" ]]
}

@test "timeout stops a verb kit runs, sending to kit alone or to its group: it answers 124, and the call logs 143" {
  command -v timeout >/dev/null || skip 'timeout is not on PATH'
  # --foreground sends the TERM to kit alone, as a supervisor does; without it, timeout sends it to the whole group
  started=${SECONDS}
  run timeout --foreground -s TERM 1 "${kit}" fake wait "${ended}"
  [ "$status" -eq 124 ]
  (( SECONDS - started <= 3 ))
  started=${SECONDS}
  run timeout -s TERM 1 "${kit}" fake wait "${ended}"
  [ "$status" -eq 124 ]
  (( SECONDS - started <= 3 ))
  run timeout --foreground --preserve-status -s TERM 1 "${kit}" fake wait "${ended}"
  [ "$status" -eq 143 ]
  run logged
  [ "${#lines[@]}" -eq 3 ]
  [[ "${lines[0]}" == *" status=143 in=. kit fake wait ${ended}" ]]
  [[ "${lines[1]}" == *" status=143 in=. kit fake wait ${ended}" ]]
  [[ "${lines[2]}" == *" status=143 in=. kit fake wait ${ended}" ]]
}

@test "at a terminal, the verb reads it as stdin, from the terminal's foreground process group" {
  run at-terminal 'typed\n' "'${kit}' fake read"
  [ "$status" -eq 0 ]
  [[ "$output" == *"terminal=yes foreground=yes"* ]]
  [[ "$output" == *"read=typed"* ]]
  run logged
  [[ "${lines[0]}" == *" status=0 in=. kit fake read" ]]
}

@test "Ctrl-C at a terminal ends the verb, and kit lives to log the call with 130" {
  started=${SECONDS}
  run at-terminal '\003' "'${kit}' fake wait '${ended}'"
  [ "$status" -eq 130 ]
  [[ "$output" == *waiting* ]]
  # the verb ended at the Ctrl-C, not when its 30 seconds were up; at-terminal itself takes four
  (( SECONDS - started <= 6 ))
  [ "$(cat "${ended}")" = ended ]
  run logged
  [[ "${lines[0]}" == *" status=130 in=. kit fake wait ${ended}" ]]
}

@test "the verb's own status is kit's, and the logged one, a status above 128 too" {
  run "${kit}" fake exit 7
  [ "$status" -eq 7 ]
  run "${kit}" fake exit 143
  [ "$status" -eq 143 ]
  run "${kit}" fake exit 0
  [ "$status" -eq 0 ]
  run logged
  [[ "${lines[0]}" == *" status=7 in=. kit fake exit 7" ]]
  [[ "${lines[1]}" == *" status=143 in=. kit fake exit 143" ]]
  [[ "${lines[2]}" == *" status=0 in=. kit fake exit 0" ]]
}

@test "stdin from a pipe reaches the verb" {
  run bash -c "printf 'piped\n' | '${kit}' fake read"
  [ "$status" -eq 0 ]
  [[ "$output" == *"terminal=no"* ]]
  [[ "$output" == *"read=piped"* ]]
}

# ----------------------------------------------------------------------------------------------------------------------
