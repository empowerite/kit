# shellcheck shell=bash
# builds.sh - the forge's runs no pull request owns, drawn as kit status draws them (sourced module)
# ----------------------------------------------------------------------------------------------------------------------
# shellcheck disable=SC2154  # status_gap is set by modules/bash/status.sh
#
# The builds stage of kit status sits between changes and releases. It lists the recent runs that no pull request
# owns, as forge:runs keeps them:
#
#   - pushes to the default branch
#   - pushes to a tag
#   - scheduled and manually started runs, on any branch
#
# A run for a pull request stays under changes, so no run shows twice.
#
# see: kit/bin/kit-status, modules/bash/status.sh, modules/bash/forge.sh, modules/jq/forge.jq

use forge status

# builds:documents prints the forge's runs, newest first, as one JSON array: id, workflow, branch, event, state, age,
# took, url, and jobs, each.
builds:documents() {
  forge:require || return 1
  forge:runs
}

# builds:glyph <state> prints a run's state glyph: the check glyphs the changes stage draws, and 🛑 for cancelled.
builds:glyph() {
  case "$1" in
    (running) printf '🏃' ;;
    (passed) printf '✅' ;;
    (failed) printf '❌' ;;
    (*) printf '🛑' ;;
  esac
}

# builds:took <seconds> prints how long a run took: 42s, 3m 36s, 1h 05m. It prints … for none, while it runs.
builds:took() {
  local secs=$1
  if [[ -z "${secs}" || "${secs}" == null ]]
  then printf '…'
  elif (( secs < 60 ))
  then printf '%ds' "${secs}"
  elif (( secs < 3600 ))
  then printf '%dm %02ds' $(( secs / 60 )) $(( secs % 60 ))
  else printf '%dh %02dm' $(( secs / 3600 )) $(( secs % 3600 / 60 ))
  fi
}

# builds:draw <runs-json> prints the runs, one row each, to stderr: the state glyph, the workflow, the branch or tag,
# the age, and how long the run took. The age and the time taken are numbers, so they are right-aligned and dim. The
# workflow and the branch are clipped to what the terminal's width leaves. With none it says so in one dim line.
builds:draw() {
  local doc=$1 row state workflow branch age took room floor
  local workflow_w=0 branch_w=0 age_w=0 took_w=0
  local -a rows=()
  if [[ "$(jq 'length' <<<"${doc}")" == 0 ]]
  then
    status:none 'no build since the last 20 runs'
    return 0
  fi
  readarray -t rows < <(jq --raw-output '.[] | [.state, .workflow, .branch, .age, .took] | map(tostring)
    | join("\u001f")' <<<"${doc}")
  for row in "${rows[@]}"
  do
    IFS=$'\x1f' read -r state workflow branch age took <<<"${row}"
    age=$(status:age "${age}")
    took=$(builds:took "${took}")
    (( ${#workflow} > workflow_w )) && workflow_w=${#workflow}
    (( ${#branch} > branch_w )) && branch_w=${#branch}
    (( ${#age} > age_w )) && age_w=${#age}
    (( ${#took} > took_w )) && took_w=${#took}
  done
  # The glyph and its gap, and the gaps before the branch, the age, and the time taken, are fixed. The branch gives
  # way first, then the workflow, neither below 10.
  room=$(( $(status:width) - 4 - 2 - 2 - age_w - 2 - took_w ))
  if (( workflow_w + branch_w > room ))
  then
    floor=$(( branch_w < 10 ? branch_w : 10 ))
    branch_w=$(( room - workflow_w < floor ? floor : room - workflow_w ))
    floor=$(( workflow_w < 10 ? workflow_w : 10 ))
    (( workflow_w + branch_w > room )) && workflow_w=$(( room - branch_w < floor ? floor : room - branch_w ))
  fi
  for row in "${rows[@]}"
  do
    IFS=$'\x1f' read -r state workflow branch age took <<<"${row}"
    printf '%s  %s%s%s%s%s%s%s\n' "$(builds:glyph "${state}")" "$(status:cell "${workflow}" "${workflow_w}")" \
      "${status_gap}" "$(status:cell "${branch}" "${branch_w}")" \
      "${status_gap}" "$(status:paint 90 "$(status:cell "$(status:age "${age}")" "${age_w}" right)")" \
      "${status_gap}" "$(status:paint 90 "$(status:cell "$(builds:took "${took}")" "${took_w}" right)")" >&2
  done
}

# builds:render [--json] prints the runs as builds:draw does, or with --json as builds:documents does, to stdout.
builds:render() {
  local doc
  doc=$(builds:documents) || return 1
  if [[ " $* " == *" --json "* ]]
  then
    printf '%s\n' "${doc}"
    return 0
  fi
  builds:draw "${doc}"
}

# ----------------------------------------------------------------------------------------------------------------------
