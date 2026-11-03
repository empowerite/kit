# shellcheck shell=bash
# underway.sh - the open issues closed by an open pull request, who works on each, and where (sourced module)
# ----------------------------------------------------------------------------------------------------------------------
# shellcheck disable=SC2154  # kit_jq is exported by modules/bash/1st.sh; flow_gap is set by modules/bash/flow.sh
#
# The underway stage of kit flow sits beside the backlog. Every open issue is in one of the two:
#
#   - backlog: the open issues not closed by any open pull request
#   - underway: the open issues closed by an open pull request, whoever opened it
#
# The rows come from forge:open-prs, the call the changes stage makes, so the stage asks the forge nothing new. One
# row per issue and pull request, the oldest issue first:
#
#    #7  👤 bob  🛫 40d  🔀 #3 🔨 draft  💻 here  🎫 Fix the thing
#   #11  👤 ada  🛫 30d  🔀 #5 🛬 ready  👤 cy    🎫 Third thing
#
# 💻 here means a checkout on this desk has the pull request's branch: the desk or a linked worktree. Otherwise the
# cell names the pull request's author, whose desk the work is on. The owner flags are kit flow's, and match the
# pull request's author or the issue's assignees.
#
# see: kit/bin/kit-flow, modules/bash/flow.sh, modules/bash/changes.sh, modules/jq/underway.jq

use forge flow

# underway:checkouts prints one JSON object that maps each branch checked out on this desk to its checkout's path:
# the desk's branch, and each linked worktree's.
underway:checkouts() {
  git worktree list --porcelain | awk '
    /^worktree / { path = substr($0, 10) }
    /^branch refs\/heads\// { print substr($0, 19) "\t" path }' \
    | jq --raw-input --slurp --compact-output \
      'split("\n") | map(select(. != "") | split("\t") | {key: .[0], value: (.[1:] | join("\t"))}) | from_entries'
}

# underway:documents [--all | [--mine] [--others] [--owner=<login>…]] prints the rows the flags choose as one JSON
# array, the oldest issue first: {number, title, assignees, age, pr, here, worktree} each. The owner flags combine as
# flow:filter says, and match the pull request's author or the issue's assignees.
underway:documents() {
  local filter prs except=false
  local -a selection=()
  filter=$(flow:filter-of "$@") || return 1
  read -r -a selection <<<"${filter}"
  if [[ "${selection[0]:-}" == --except ]]
  then
    except=true
    selection=("${selection[@]:1}")
  fi
  prs=$(forge:open-prs) || return 1
  jq --slurp --compact-output -L "${kit_jq}" --argjson worktrees "$(underway:checkouts)" \
    --argjson except "${except}" \
    'import "underway" as underway; underway::rows($worktrees) | underway::by_owners($ARGS.positional; $except)' \
    --args "${selection[@]}" <<<"${prs}"
}

# underway:draw <rows-json> <choice> prints the rows, one each, to stderr. With none it says so in one dim line.
# <choice> mine leaves the assignee column out, as the backlog does. The cells, in columns:
#
#   1. the issue's number, cyan and right-aligned
#   2. 👤 and its assignees, dim
#   3. 🛫 and its age, dim and right-aligned
#   4. 🔀, the pull request's number, magenta and right-aligned, then its state glyph and state, as changes draws it
#   5. 💻 here, or 👤 and the pull request's author, dim
#   6. 🎫 and the issue's title, clipped to what the terminal's width leaves
underway:draw() {
  local doc=$1 choice=$2 sigil row ref assignees age number state here author title width glyph
  local ref_w=0 owner_w=0 age_w=0 pr_w=0 state_w=0 where_w=4
  local -a rows=()
  if [[ "$(jq 'length' <<<"${doc}")" == 0 ]]
  then
    flow:none 'no issue is underway'
    return 0
  fi
  sigil=$(forge:noun 2>/dev/null | cut -f 1) || true
  readarray -t rows < <(jq --raw-output '.[] | ["#\(.number)", (.assignees | join(" ")), (.age // "" | tostring),
    (.pr.number | tostring), .pr.state, (.here | tostring), .pr.author, .title] | join("\u001f")' <<<"${doc}")
  for row in "${rows[@]}"
  do
    IFS=$'\x1f' read -r ref assignees age number state here author title <<<"${row}"
    age=$(flow:age "${age}")
    number="${sigil:-#}${number}"
    (( ${#ref} > ref_w )) && ref_w=${#ref}
    (( ${#assignees} > owner_w )) && owner_w=${#assignees}
    (( ${#age} > age_w )) && age_w=${#age}
    (( ${#number} > pr_w )) && pr_w=${#number}
    (( ${#state} > state_w )) && state_w=${#state}
    [[ "${here}" == true ]] || (( ${#author} <= where_w )) || where_w=${#author}
  done
  [[ "${choice}" != mine ]] || owner_w=0
  # Each glyph is two columns and is followed by a space; flow_gap, two columns, comes before every cell but the
  # first. The title takes what is left, and never less than 20.
  width=$(( $(flow:width) - ref_w - (owner_w > 0 ? 5 + owner_w : 0) - 5 - age_w - 5 - pr_w - 4 - state_w \
    - 5 - where_w - 5 ))
  (( width < 20 )) && width=20
  for row in "${rows[@]}"
  do
    IFS=$'\x1f' read -r ref assignees age number state here author title <<<"${row}"
    row=$(flow:paint "${flow_issue_sgr}" "$(flow:cell "${ref}" "${ref_w}" right)")
    if (( owner_w > 0 ))
    then
      glyph='   '
      [[ -n "${assignees}" ]] && glyph='👤 '
      row+="${flow_gap}${glyph}$(flow:paint 90 "$(flow:cell "${assignees}" "${owner_w}")")"
    fi
    row+="${flow_gap}🛫 $(flow:paint 90 "$(flow:cell "$(flow:age "${age}")" "${age_w}" right)")"
    number=$(flow:cell "${sigil:-#}${number}" "${pr_w}" right)
    row+="${flow_gap}🔀 $(flow:paint "${flow_change_sgr}" "${number}")"
    row+=" $(flow:state-emoji "${state}") $(flow:paint "$(flow:state-sgr "${state}")" \
      "$(flow:cell "${state}" "${state_w}")")"
    if [[ "${here}" == true ]]
    then row+="${flow_gap}💻 $(flow:cell here "${where_w}")"
    else row+="${flow_gap}👤 $(flow:paint 90 "$(flow:cell "${author}" "${where_w}")")"
    fi
    (( ${#title} > width )) && title="${title:0:width-1}…"
    printf '%s%s🎫 %s\n' "${row}" "${flow_gap}" "${title}" >&2
  done
}

# underway:render [--json] [--all | [--mine] [--others] [--owner=<login>…]] prints the rows as underway:draw does, or
# with --json as underway:documents does, to stdout.
underway:render() {
  local doc
  doc=$(underway:documents "$@") || return 1
  if [[ " $* " == *" --json "* ]]
  then
    printf '%s\n' "${doc}"
    return 0
  fi
  underway:draw "${doc}" "$(flow:choice-of "$@")"
}

# ----------------------------------------------------------------------------------------------------------------------
