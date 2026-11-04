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
#    #7  👤 bob  45d  🔀 #3 🔨 draft  🌿 here          📤  3d          🎫 Fix the thing
#    #8  👤 bob  37d  🔀 #6 🔨 draft  💻 here          📤 10m          🎫 Eight
#    #9  👤 ada  34d  🔀 #4 🏁 ready  🌱 local                         🎫 Nine
#   #11  👤 ada  30d  🔀 #5 🏁 ready  👤 cy     🍴 cy  📤  2h  👤 dee  🎫 Third thing
#
# 💻 here means the desk, the root checkout, has the pull request's branch, and 🌿 here that a linked worktree has it.
# 🌱 local means the desk has the branch and no checkout has it. Otherwise the cell names the pull request's author,
# whose desk the work is on. 🍴 names the owner of the fork the branch lives in, 📤 the age of the branch's last
# push, and the 👤 after it who pushed, when that is not the author; each is blank when the forge does not tell it.
# The owner flags are kit flow's, and match the pull request's author or the issue's assignees.
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

# underway:branches prints the local branches on this desk, refs/heads/<branch>, as one JSON array of their names.
# The desk and its linked worktrees share them.
underway:branches() {
  git for-each-ref --format='%(refname:lstrip=2)' refs/heads/ \
    | jq --raw-input --slurp --compact-output 'split("\n") | map(select(. != ""))'
}

# underway:documents [--all | [--mine] [--others] [--owner=<login>…]] prints the rows the flags choose as one JSON
# array, the oldest issue first: {number, title, assignees, age, pr, here, local, worktree} each. The owner flags
# combine as flow:filter says, and match the pull request's author or the issue's assignees.
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
    --argjson branches "$(underway:branches)" --argjson except "${except}" \
    'import "underway" as underway;
     underway::rows($worktrees; $branches) | underway::by_owners($ARGS.positional; $except)' \
    --args "${selection[@]}" <<<"${prs}"
}

# underway:draw <rows-json> <choice> prints the rows, one each, to stderr. With none it says so in one dim line.
# <choice> mine leaves the assignee column out, as the backlog does. The cells, in columns:
#
#   1. the issue's number, cyan and right-aligned
#   2. 👤 and its assignees, dim
#   3. the issue's age, dim and right-aligned
#   4. 🔀, the pull request's number, magenta and right-aligned, then its state glyph and state, as changes draws it
#   5. 💻 here for the desk, 🌿 here for a linked worktree, 🌱 local, or 👤 and the pull request's author, dim
#   6. 🍴 and the owner of the fork the branch lives in, dim
#   7. 📤 and the last push's age, dim and right-aligned
#   8. 👤 and who pushed last, dim, when that is not the pull request's author
#   9. 🎫 and the issue's title, clipped to what the terminal's width leaves
#
# Cells 6, 7, and 8 are blank in a row that has none, and take no room when no row has one.
underway:draw() {
  local doc=$1 choice=$2 sigil desk row ref assignees age number state where author fork pushed by title width glyph
  local ref_w=0 owner_w=0 age_w=0 pr_w=0 state_w=0 where_w=4 fork_w=0 pushed_w=0 by_w=0
  local -a rows=()
  if [[ "$(jq 'length' <<<"${doc}")" == 0 ]]
  then
    flow:none 'no issue is underway'
    return 0
  fi
  sigil=$(forge:noun 2>/dev/null | cut -f 1) || true
  # The desk is the first checkout git lists; underway:checkouts names each checkout's path as git lists it.
  desk=$(git worktree list --porcelain 2>/dev/null | sed -n '1s/^worktree //p') || true
  readarray -t rows < <(jq --raw-output --arg desk "${desk}" '.[] | ["#\(.number)", (.assignees | join(" ")),
    (.age // "" | tostring), (.pr.number | tostring), .pr.state,
    (if .here then (if .worktree == $desk then "desk" else "linked" end) elif .local then "local" else "" end),
    .pr.author,
    (.pr.fork // ""), (.pr.pushed.age // "" | tostring),
    (.pr as $pr | $pr.pushed.by // "" | if . == $pr.author then "" else . end), .title] | join("\u001f")' \
    <<<"${doc}")
  for row in "${rows[@]}"
  do
    IFS=$'\x1f' read -r ref assignees age number state where author fork pushed by title <<<"${row}"
    age=$(flow:age "${age}")
    pushed=$(flow:age "${pushed}")
    number="${sigil:-#}${number}"
    case "${where}" in
      (desk | linked) where=here ;;
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
  done
  [[ "${choice}" != mine ]] || owner_w=0
  # Each glyph is two columns and is followed by a space; flow_gap, two columns, comes before every cell but the
  # first. The age has no glyph. The title takes what is left, and never less than 20.
  width=$(( $(flow:width) - ref_w - (owner_w > 0 ? 5 + owner_w : 0) - 2 - age_w - 5 - pr_w - 4 - state_w \
    - 5 - where_w - (fork_w > 0 ? 5 + fork_w : 0) - (pushed_w > 0 ? 5 + pushed_w : 0) - (by_w > 0 ? 5 + by_w : 0) \
    - 5 ))
  (( width < 20 )) && width=20
  for row in "${rows[@]}"
  do
    IFS=$'\x1f' read -r ref assignees age number state where author fork pushed by title <<<"${row}"
    row=$(flow:paint "${flow_issue_sgr}" "$(flow:cell "${ref}" "${ref_w}" right)")
    if (( owner_w > 0 ))
    then
      glyph='   '
      [[ -n "${assignees}" ]] && glyph='👤 '
      row+="${flow_gap}${glyph}$(flow:paint 90 "$(flow:cell "${assignees}" "${owner_w}")")"
    fi
    row+="${flow_gap}$(flow:paint 90 "$(flow:cell "$(flow:age "${age}")" "${age_w}" right)")"
    number=$(flow:cell "${sigil:-#}${number}" "${pr_w}" right)
    row+="${flow_gap}🔀 $(flow:paint "${flow_change_sgr}" "${number}")"
    row+=" $(flow:state-emoji "${state}") $(flow:paint "$(flow:state-sgr "${state}")" \
      "$(flow:cell "${state}" "${state_w}")")"
    case "${where}" in
      (desk) row+="${flow_gap}💻 $(flow:cell here "${where_w}")" ;;
      (linked) row+="${flow_gap}🌿 $(flow:cell here "${where_w}")" ;;
      (local) row+="${flow_gap}🌱 $(flow:cell local "${where_w}")" ;;
      (*) row+="${flow_gap}👤 $(flow:paint 90 "$(flow:cell "${author}" "${where_w}")")"
    esac
    if (( fork_w > 0 ))
    then
      glyph='   '
      [[ -n "${fork}" ]] && glyph='🍴 '
      row+="${flow_gap}${glyph}$(flow:paint 90 "$(flow:cell "${fork}" "${fork_w}")")"
    fi
    if (( pushed_w > 0 ))
    then
      glyph='   '
      [[ -n "${pushed}" ]] && glyph='📤 '
      row+="${flow_gap}${glyph}$(flow:paint 90 "$(flow:cell "$(flow:age "${pushed}")" "${pushed_w}" right)")"
    fi
    if (( by_w > 0 ))
    then
      glyph='   '
      [[ -n "${by}" ]] && glyph='👤 '
      row+="${flow_gap}${glyph}$(flow:paint 90 "$(flow:cell "${by}" "${by_w}")")"
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
