# shellcheck shell=bash
# log.sh - the worktree's logs under .kit/logs/, a line per kit call and per MCP request, kept log_days days (sourced module)
# ----------------------------------------------------------------------------------------------------------------------
# kit keeps two logs, one file a day each, under .kit/logs/ at the root of the worktree it runs in:
#
#   - kit-YYYY-MMDD.log has a line for each call of the kit CLI, written when the call ends: when it started, its
#     process, its depth (a call another kit call made is one deeper), its exit status, its seconds, the directory it
#     was typed in relative to the root, and the command line, quoted so it can be pasted back.
#   - mcp-YYYY-MMDD.log has a line for each message kit serve answers, and one when it starts and stops: the request's
#     id and method, the tool and its arguments or the resource, the outcome, and its seconds.
#
# A log is written only where .kit/.gitignore lists /logs/, as kit hydrate writes it, so git never sees one, and a
# directory that is not a kit worktree gets none. log_days in kit.yml is how many days a log is kept, 14 when it is
# not set, and 0 writes none. A day's first line removes each log last written more than log_days days before.
#
# bin/kit sources this file alone, before a verb loads the loader, so it uses nothing from the loader. Writing a line
# starts no process but on a day's first line. A log that cannot be written is skipped, and the call goes on.
#
# see: bin/kit, modules/bash/serve.sh, docs/logs.md

# log:root <var> sets <var> to the root of the worktree the working directory is in, the nearest directory at or
# above it that holds .git. It fails outside a worktree. It assigns rather than prints, so it starts no subshell, and
# each function here that assigns names its own locals log_*, so a caller's <var> is never one of them.
log:root() {
  local log_dir=${PWD}
  until [[ -e "${log_dir}/.git" ]]
  do
    [[ -n "${log_dir}" ]] || return 1
    log_dir=${log_dir%/*}
  done
  printf -v "$1" '%s' "${log_dir:-/}"
}

# log:days <root> <var> sets <var> to how many days a log is kept: log_days in <root>/kit.yml, else 14.
log:days() {
  local log_line log_days=14
  if [[ -f "$1/kit.yml" ]]
  then
    while IFS= read -r log_line || [[ -n "${log_line}" ]]
    do
      if [[ "${log_line}" =~ ^log_days:[[:space:]]*([0-9]+)[[:space:]]*(#.*)?$ ]]
      then
        log_days=$(( 10#${BASH_REMATCH[1]} ))
        break
      fi
    done < "$1/kit.yml"
  fi
  printf -v "$2" '%s' "${log_days}"
}

# log:prune <dir> <days> removes each log in <dir> last written more than <days> whole days ago.
log:prune() {
  find "$1" -maxdepth 1 -type f -name '*.log' -mtime "+$2" -exec rm -f {} + 2>/dev/null || true
}

# log:write <name> <line> appends <line> to today's <name> log, .kit/logs/<name>-YYYY-MMDD.log at the root, where
# .kit/.gitignore lists /logs/ and log_days is not 0. The day's first line prunes the logs first. It always succeeds.
log:write() {
  local root ignore='' days file
  log:root root || return 0
  [[ -f "${root}/.kit/.gitignore" ]] || return 0
  IFS= read -r -d '' ignore < "${root}/.kit/.gitignore" || true
  [[ $'\n'"${ignore}"$'\n' == *$'\n/logs/\n'* ]] || return 0
  printf -v file '%s/.kit/logs/%s-%(%Y-%m%d)T.log' "${root}" "$1" -1
  if [[ ! -f "${file}" ]]
  then
    log:days "${root}" days
    (( days > 0 )) || return 0
    mkdir -p "${root}/.kit/logs" 2>/dev/null || return 0
    log:prune "${root}/.kit/logs" "${days}"
  fi
  { printf '%s\n' "$2" >> "${file}"; } 2>/dev/null || true
}

# log:now <var> sets <var> to the time now in microseconds, or in whole seconds times a million where the shell has
# no EPOCHREALTIME (bash before 5).
log:now() {
  if [[ -n "${EPOCHREALTIME:-}" ]]
  then
    printf -v "$1" '%s' "${EPOCHREALTIME/[.,]/}"
  else
    printf -v "$1" '%s' "$(( $(printf '%(%s)T' -1) * 1000000 ))"
  fi
}

# log:stamp <var> <started> sets <var> to the time <started>, in microseconds, then a tab, then the seconds since
# then: 2026-11-02T10:15:03-0800 and 1.234.
log:stamp() {
  local log_now log_elapsed
  log:now log_now
  log_elapsed=$(( (log_now - $2) / 1000 ))
  printf -v "$1" '%(%Y-%m-%dT%H:%M:%S%z)T\t%d.%03d' "$(( $2 / 1000000 ))" "$(( log_elapsed / 1000 ))" \
    "$(( log_elapsed % 1000 ))"
}

# log:call <name> <started> <field>… writes one line to the <name> log: when <started> was, the seconds since then,
# and each field, separated by spaces. A field with spaces in it goes last, as a command line does.
log:call() {
  local name=$1 stamp
  log:stamp stamp "$2"
  shift 2
  log:write "${name}" "${stamp%%$'\t'*} seconds=${stamp#*$'\t'}${*:+ $*}"
}

# ----------------------------------------------------------------------------------------------------------------------
