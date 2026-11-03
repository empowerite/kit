# shellcheck shell=bash
# builds.sh - the forge's runs not owned by any pull request, drawn as kit flow draws them (sourced module)
# ----------------------------------------------------------------------------------------------------------------------
# shellcheck disable=SC2154  # flow_gap is set by modules/bash/flow.sh
#
# The builds stage of kit flow sits between changes and releases. It lists the recent runs that no pull request
# owns, as forge:runs keeps them:
#
#   - pushes to the default branch
#   - pushes to a tag
#   - scheduled and manually started runs, on any branch
#
# A run for a pull request stays under changes, so no run shows twice.
#
# see: kit/bin/kit-flow, modules/bash/flow.sh, modules/bash/forge.sh, modules/jq/forge.jq

use forge flow

# builds:documents prints the forge's runs, newest first, as one JSON array: id, number, attempt, workflow, branch,
# tag, event, sha, title, state, age, took, url, and jobs, each.
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

# builds:event <event> <tag> prints how a run began, in a word: tag for a push to a tag, push, schedule, or manual.
builds:event() {
  if [[ "$2" == true ]]
  then printf 'tag'
  else
    case "$1" in
      (workflow_dispatch) printf 'manual' ;;
      (*) printf '%s' "$1" ;;
    esac
  fi
}

# builds:draw <runs-json> prints the runs, one row each, to stderr: the state glyph, the workflow, the run's number,
# how it began, the branch or tag, the commit's short hash, the run's title, the age, and how long the run took. The
# number, how it began, and the hash are dim; the age and the time taken are numbers, so they are right-aligned and
# dim. A title that only repeats the workflow is left out, and so is a column no run fills. The title gives way first
# to what the terminal's width leaves, then the branch, then the workflow, none below 10, each clipped with an
# ellipsis. With none it says so in one dim line.
builds:draw() {
  local doc=$1 row state workflow number event tag branch sha title age took room excess name floor give line
  local workflow_w=0 number_w=0 event_w=0 branch_w=0 sha_w=0 title_w=0 age_w=0 took_w=0
  local -a rows=()
  if [[ "$(jq 'length' <<<"${doc}")" == 0 ]]
  then
    flow:none 'no build since the last 20 runs'
    return 0
  fi
  readarray -t rows < <(jq --raw-output '.[] | [.state, .workflow, (if .number then "#\(.number)" else "" end),
    .event, .tag, .branch, .sha, (if .title == .workflow then "" else .title end), .age, .took]
    | map(. // "" | tostring) | join("\u001f")' <<<"${doc}")
  for row in "${rows[@]}"
  do
    IFS=$'\x1f' read -r state workflow number event tag branch sha title age took <<<"${row}"
    event=$(builds:event "${event}" "${tag}")
    age=$(flow:age "${age}")
    took=$(builds:took "${took}")
    (( ${#workflow} > workflow_w )) && workflow_w=${#workflow}
    (( ${#number} > number_w )) && number_w=${#number}
    (( ${#event} > event_w )) && event_w=${#event}
    (( ${#branch} > branch_w )) && branch_w=${#branch}
    (( ${#sha} > sha_w )) && sha_w=${#sha}
    (( ${#title} > title_w )) && title_w=${#title}
    (( ${#age} > age_w )) && age_w=${#age}
    (( ${#took} > took_w )) && took_w=${#took}
  done
  # The glyph and its gap, and the dim and numeric columns with their gaps, are fixed; the workflow, the branch, and
  # the title share the rest.
  room=$(( $(flow:width) - 4 - (2 + event_w) - (2 + age_w) - (2 + took_w) - 2 ))
  (( number_w > 0 )) && room=$(( room - 2 - number_w ))
  (( sha_w > 0 )) && room=$(( room - 2 - sha_w ))
  (( title_w > 0 )) && room=$(( room - 2 ))
  excess=$(( workflow_w + branch_w + title_w - room ))
  for name in title_w branch_w workflow_w
  do
    (( excess > 0 )) || break
    floor=$(( ${!name} < 10 ? ${!name} : 10 ))
    give=$(( ${!name} - floor < excess ? ${!name} - floor : excess ))
    printf -v "${name}" '%d' $(( ${!name} - give ))
    excess=$(( excess - give ))
  done
  for row in "${rows[@]}"
  do
    IFS=$'\x1f' read -r state workflow number event tag branch sha title age took <<<"${row}"
    line="$(builds:glyph "${state}")  $(flow:cell "${workflow}" "${workflow_w}")"
    (( number_w > 0 )) && line+="${flow_gap}$(flow:paint 90 "$(flow:cell "${number}" "${number_w}" right)")"
    line+="${flow_gap}$(flow:paint 90 "$(flow:cell "$(builds:event "${event}" "${tag}")" "${event_w}")")"
    line+="${flow_gap}$(flow:cell "${branch}" "${branch_w}")"
    (( sha_w > 0 )) && line+="${flow_gap}$(flow:paint 90 "$(flow:cell "${sha}" "${sha_w}")")"
    (( title_w > 0 )) && line+="${flow_gap}$(flow:cell "${title}" "${title_w}")"
    line+="${flow_gap}$(flow:paint 90 "$(flow:cell "$(flow:age "${age}")" "${age_w}" right)")"
    line+="${flow_gap}$(flow:paint 90 "$(flow:cell "$(builds:took "${took}")" "${took_w}" right)")"
    printf '%s\n' "${line}" >&2
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
