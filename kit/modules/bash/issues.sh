# shellcheck shell=bash
# issues.sh - every open issue, in the backlog or underway: who works on each, and where (sourced module)
# ----------------------------------------------------------------------------------------------------------------------
# shellcheck disable=SC2154  # kit_jq is exported by modules/bash/1st.sh; flow_gap is set by modules/bash/flow.sh
#
# The issues stage of kit flow lists every open issue of the repository on its forge, as forge:open-issues folds them,
# each with its status:
#
#   - backlog: no open pull request closes it. One with a branch for it on the primary remote is claimed, and its row
#     names the branch; one with neither is for the next kit work <N>.
#   - underway: an open pull request closes it, whoever opened it. The row names the pull request, and where its work
#     is. An issue two pull requests close has a row for each.
#
# The rows come from forge:open-issues and forge:open-prs, the call the changes stage makes, and the stages of one load
# share their snapshot, as modules/bash/forge.sh says, so the stage asks the forge nothing new. The oldest issue first:
#
#    #7  👤 bob  45d  🛫  🔀 #3 🔨 draft  🌿 here          📤  3d          ⛳ v0.1  🎫 Fix the thing
#    #9  👤 ada  34d  🛫  🔀 #4 🏁 ready  🌱 local                        ⛳ v0.1  🎫 Nine
#   #10  👤 bob  32d  💤                  🌿 10-ten                                🎫 Ten
#   #11  👤 ada  30d  🛫  🔀 #5 🏁 ready  👤 cy     🍴 cy  📤  2h  👤 dee          🎫 Third thing
#   #12          20d  💤                                                           🎫 Twelve
#
# 💤 says the issue is in the backlog, and 🛫 that it is underway. 💻 here means the desk, the root checkout, has the
# pull request's branch, and 🌿 here that a linked worktree has it. 🌱 local means the desk has the branch and no
# checkout has it. Otherwise the cell names the pull request's author, whose desk the work is on, or for an issue in
# the backlog, the branch that claims it. 🍴 names the owner of the fork the branch lives in, 📤 the age of the
# branch's last push, and the 👤 after it who pushed, when that is not the author; each is blank when the forge does
# not tell it. ⛳ names the issue's milestone, 📅 its iteration, and 🏊 its swimlane, each the forge's own or the
# stand-in kit.yml chooses, as modules/bash/forge.sh says. The owner flags are kit flow's, and match the issue's
# assignees; who opened the change does not count. --today keeps only the issues opened in the last 24 hours.
#
# see: kit/bin/kit-flow, modules/bash/flow.sh, modules/bash/changes.sh, modules/bash/forge.sh, modules/jq/issues.jq

use forge flow workspaces

# issues:checkouts prints one JSON object that maps each branch checked out on this desk to its checkout's path:
# the desk's branch, and each linked worktree's.
issues:checkouts() {
  git worktree list --porcelain | awk '
    /^worktree / { path = substr($0, 10) }
    /^branch refs\/heads\// { print substr($0, 19) "\t" path }' \
    | jq --raw-input --slurp --compact-output \
      'split("\n") | map(select(. != "") | split("\t") | {key: .[0], value: (.[1:] | join("\t"))}) | from_entries'
}

# issues:branches prints the local branches on this desk, refs/heads/<branch>, as one JSON array of their names.
# The desk and its linked worktrees share them.
issues:branches() {
  git for-each-ref --format='%(refname:lstrip=2)' refs/heads/ \
    | jq --raw-input --slurp --compact-output 'split("\n") | map(select(. != ""))'
}

# issues:documents [--today] [--all | [--mine] [--others] [--owner=<login>…]] prints the rows the flags choose as one
# JSON array, the oldest issue first, as issues.jq's rows gives them. The owner flags combine as flow:filter says, and
# match the issue's assignees. The branches that claim an issue are the primary's as of the last fetch, which the
# picture's incoming stage makes first, not asked of the remote, which takes seconds a stage cannot spare.
issues:documents() {
  local filter issues prs heads primary doc except=false
  local -a selection=()
  filter=$(flow:filter-of "$@") || return 1
  read -r -a selection <<<"${filter}"
  if [[ "${selection[0]:-}" == --except ]]
  then
    except=true
    selection=("${selection[@]:1}")
  fi
  primary=$(forge:primary) || return $?
  # A forge without issues of its own has none open.
  issues=$(forge:open-issues) || (( $? == 3 )) || return 1
  prs=$(forge:open-prs) || return 1
  heads=$(git for-each-ref --format='%(refname:lstrip=3)' -- "refs/remotes/${primary}/") || return 1
  doc=$(jq --compact-output --slurp -L "${kit_jq}" --argjson changes "$(jq --slurp . <<<"${prs}")" \
    --arg heads "${heads}" --argjson worktrees "$(issues:checkouts)" --argjson branches "$(issues:branches)" \
    --argjson except "${except}" '
    import "issues" as issues;
    issues::rows($changes; $heads | split("\n") | map(select(. != "" and . != "HEAD")); $worktrees; $branches)
    | issues::by_owners($ARGS.positional; $except)' --args "${selection[@]}" <<<"${issues}") || return 1
  [[ " $* " != *" --today "* ]] || doc=$(jq --compact-output 'map(select(.age != null and .age < 86400))' <<<"${doc}")
  printf '%s\n' "${doc}"
}

# issues:draw <rows-json> <choice> prints the rows, one each, to stderr. With none it says so in one dim line.
# <choice> mine leaves the assignee column out, as a change's row drops the author column under --mine. The cells, in
# columns:
#
#   1. the issue's number, cyan and right-aligned
#   2. 👤 and its assignees, dim
#   3. the issue's age, dim and right-aligned
#   4. 💤 for an issue in the backlog, 🛫 for one underway
#   5. 🔀, the pull request's number, magenta and right-aligned, then its state glyph and state, as changes draws it
#   6. 💻 here for the desk, 🌿 here for a linked worktree, 🌱 local, or 👤 and the pull request's author, dim; for an
#      issue in the backlog, 🌿 and the branch that claims it
#   7. 🍴 and the owner of the fork the branch lives in, dim
#   8. 📤 and the last push's age, dim and right-aligned
#   9. 👤 and who pushed last, dim, when that is not the pull request's author
#  10. ⛳ and the milestone, 📅 and the iteration, 🏊 and the swimlane, dim
#  11. 🎫 and the issue's title, clipped to what the terminal's width leaves
#
# A cell is blank in a row that has none, and a column takes no room when no row has one: column 5 when no issue is
# underway, and column 6 when none is underway or claimed.
issues:draw() {
  local doc=$1 choice=$2 desk row ref assignees age status number state where author fork pushed by branch
  local milestone iteration swimlane title width glyph
  local ref_w=0 owner_w=0 age_w=0 pr_w=0 state_w=0 where_w=0 fork_w=0 pushed_w=0 by_w=0 milestone_w=0
  local iteration_w=0 swimlane_w=0
  local -a rows=()
  if [[ "$(jq 'length' <<<"${doc}")" == 0 ]]
  then
    flow:none 'no open issue'
    return 0
  fi
  # The desk is the first checkout git lists; issues:checkouts names each checkout's path as git lists it. A change is
  # drawn as the forge it is on names one: !<number> on GitLab and Azure DevOps, #<number> elsewhere.
  desk=$(git worktree list --porcelain 2>/dev/null | sed -n '1s/^worktree //p') || true
  readarray -t rows < <(jq --raw-output --arg desk "${desk}" '.[] | ["#\(.number)", ((.assignees // []) | join(" ")),
    (.age // "" | tostring), .status,
    (.pr // null | if . == null then "" else (if .source | IN("gitlab", "ado") then "!" else "#" end) + "\(.number)"
                   end),
    (.pr.state // ""),
    (if .status == "backlog" then (if .branch then "branch" else "none" end)
     elif .here then (if .worktree == $desk then "desk" else "linked" end) elif .local then "local" else "" end),
    (.pr.author // ""), (.pr.fork // ""), (.pr.pushed.age // "" | tostring),
    (.pr as $pr | $pr.pushed.by // "" | if . == $pr.author then "" else . end), (.branch // ""),
    (.milestone // ""), (.iteration // ""), (.swimlane // ""), .title] | join("\u001f")' <<<"${doc}")
  for row in "${rows[@]}"
  do
    IFS=$'\x1f' read -r ref assignees age status number state where author fork pushed by branch milestone \
      iteration swimlane title <<<"${row}"
    age=$(flow:age "${age}")
    pushed=$(flow:age "${pushed}")
    case "${where}" in
      (desk | linked) where=here ;;
      (branch) where=${branch} ;;
      (none) where='' ;;
      ('') where=${author}
    esac
    (( ${#ref} > ref_w )) && ref_w=${#ref}
    (( ${#assignees} > owner_w )) && owner_w=${#assignees}
    (( ${#age} > age_w )) && age_w=${#age}
    (( ${#number} > pr_w )) && pr_w=${#number}
    (( ${#state} > state_w )) && state_w=${#state}
    (( ${#where} > where_w )) && where_w=${#where}
    (( ${#fork} > fork_w )) && fork_w=${#fork}
    (( ${#pushed} > pushed_w )) && pushed_w=${#pushed}
    (( ${#by} > by_w )) && by_w=${#by}
    (( ${#milestone} > milestone_w )) && milestone_w=${#milestone}
    (( ${#iteration} > iteration_w )) && iteration_w=${#iteration}
    (( ${#swimlane} > swimlane_w )) && swimlane_w=${#swimlane}
  done
  [[ "${choice}" != mine ]] || owner_w=0
  (( where_w == 0 || where_w >= 4 )) || where_w=4
  # Each glyph is two columns and is followed by a space; flow_gap, two columns, comes before every cell but the
  # first. The age and the status have no text after their glyph. The title takes what is left, and never less
  # than 20.
  width=$(( $(flow:width) - ref_w - (owner_w > 0 ? 5 + owner_w : 0) - 2 - age_w - 4 \
    - (pr_w > 0 ? 5 + pr_w + 4 + state_w : 0) - (where_w > 0 ? 5 + where_w : 0) - (fork_w > 0 ? 5 + fork_w : 0) \
    - (pushed_w > 0 ? 5 + pushed_w : 0) - (by_w > 0 ? 5 + by_w : 0) - (milestone_w > 0 ? 5 + milestone_w : 0) \
    - (iteration_w > 0 ? 5 + iteration_w : 0) - (swimlane_w > 0 ? 5 + swimlane_w : 0) - 5 ))
  (( width < 20 )) && width=20
  for row in "${rows[@]}"
  do
    IFS=$'\x1f' read -r ref assignees age status number state where author fork pushed by branch milestone \
      iteration swimlane title <<<"${row}"
    row=$(flow:paint "${flow_issue_sgr}" "$(flow:cell "${ref}" "${ref_w}" right)")
    if (( owner_w > 0 ))
    then
      glyph='   '
      [[ -n "${assignees}" ]] && glyph="${flow_glyphs[Owner]} "
      row+="${flow_gap}${glyph}$(flow:paint 90 "$(flow:cell "${assignees}" "${owner_w}")")"
    fi
    row+="${flow_gap}$(flow:paint 90 "$(flow:cell "$(flow:age "${age}")" "${age_w}" right)")"
    if [[ "${status}" == underway ]]
    then row+="${flow_gap}${flow_glyphs[InFlight]}"
    else row+="${flow_gap}${flow_glyphs[Waiting]}"
    fi
    if (( pr_w > 0 ))
    then
      if [[ -n "${number}" ]]
      then
        row+="${flow_gap}${flow_glyphs[Change]} $(flow:paint "${flow_change_sgr}" \
          "$(flow:cell "${number}" "${pr_w}" right)")"
        row+=" $(flow:state-emoji "${state}") $(flow:paint "$(flow:state-sgr "${state}")" \
          "$(flow:cell "${state}" "${state_w}")")"
      else
        row+="${flow_gap}   $(flow:cell '' "${pr_w}") $(flow:cell '' $(( 2 + 1 + state_w )))"
      fi
    fi
    if (( where_w > 0 ))
    then
      case "${where}" in
        (desk) row+="${flow_gap}${flow_glyphs[Desk]} $(flow:cell here "${where_w}")" ;;
        (linked) row+="${flow_gap}${flow_glyphs[Branch]} $(flow:cell here "${where_w}")" ;;
        (local) row+="${flow_gap}${flow_glyphs[Sprout]} $(flow:cell local "${where_w}")" ;;
        (branch) row+="${flow_gap}${flow_glyphs[Branch]} $(flow:paint 90 "$(flow:cell "${branch}" "${where_w}")")" ;;
        (none) row+="${flow_gap}   $(flow:cell '' "${where_w}")" ;;
        (*) row+="${flow_gap}${flow_glyphs[Owner]} $(flow:paint 90 "$(flow:cell "${author}" "${where_w}")")"
      esac
    fi
    row+=$(issues:cell "${fork_w}" "${fork}" Fork)
    if (( pushed_w > 0 ))
    then
      glyph='   '
      [[ -n "${pushed}" ]] && glyph="${flow_glyphs[Pushed]} "
      row+="${flow_gap}${glyph}$(flow:paint 90 "$(flow:cell "$(flow:age "${pushed}")" "${pushed_w}" right)")"
    fi
    row+=$(issues:cell "${by_w}" "${by}" Owner)
    row+=$(issues:cell "${milestone_w}" "${milestone}" Milestone)
    row+=$(issues:cell "${iteration_w}" "${iteration}" Iteration)
    row+=$(issues:cell "${swimlane_w}" "${swimlane}" Swimlane)
    (( ${#title} > width )) && title="${title:0:width-1}…"
    printf '%s%s%s %s\n' "${row}" "${flow_gap}" "${flow_glyphs[Issue]}" "${title}" >&2
  done
}

# issues:cell <width> <text> <glyph> prints one dim cell of issues:draw's, after flow_gap: the glyph named and the text,
# padded to the width, or blank, glyph included, for no text; nothing for a width of 0, a column no row has.
issues:cell() {
  local width=$1 text=$2 glyph='   '
  (( width > 0 )) || return 0
  [[ -z "${text}" ]] || glyph="${flow_glyphs[$3]} "
  printf '%s%s%s' "${flow_gap}" "${glyph}" "$(flow:paint 90 "$(flow:cell "${text}" "${width}")")"
}

# issues:render [--json] [--today] [--all | [--mine] [--others] [--owner=<login>…]] prints the rows as issues:draw
# does, or with --json as issues:documents does, to stdout, each saying where it is from, as workspaces:placed says,
# the forge its source where the tracker names none.
issues:render() {
  local doc
  doc=$(issues:documents "$@") || return 1
  if [[ " $* " == *" --json "* ]]
  then
    workspaces:placed forge "${doc}"
    return
  fi
  issues:draw "${doc}" "$(flow:choice-of "$@")"
}

# ----------------------------------------------------------------------------------------------------------------------
