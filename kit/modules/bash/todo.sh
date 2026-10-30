# shellcheck shell=bash
# todo.sh - the forge's open issues, oldest first, each with what is already in flight for it (sourced module)
# ----------------------------------------------------------------------------------------------------------------------
# shellcheck disable=SC2154  # modules/bash/status.sh sets status_sep, status_issue_sgr, and status_change_sgr
#
# kit todo is the pick: the open issues of the repository on its forge, as forge:open-issues folds them, and for each
# whether work has started, which is when an open change closes it or a branch for it is on origin. An issue with
# nothing in flight is one to take up with kit issue <N>. The flags are kit status's: --mine is the issues assigned to
# the caller, --others the rest, unassigned ones included, --owner=<login> the named assignees', --all every open
# issue; --today keeps only the issues opened in the last 24 hours, whichever of those is chosen.
#
# A row is drawn as kit status draws a pull request, its cells separated by status_sep and sized across the rows:
#
#    #7 · 👤 bob · 🛫 in flight 40d · 🔀 #3 · 🌿 7-fix-the-thing · 🎫 Fix the thing
#   #11 · 👤 ada · ⏳ idle      30d ·       ·                    · 🎫 Third thing
#
# that is the issue's number, right-aligned, its assignees (dropped under --mine, as kit status drops the author),
# its state with the issue's age, right-aligned, the change and the branch in flight, and last the title, clipped to
# what the terminal's width leaves after the other cells. A cell a row has nothing for is blank, glyph included, so
# every title starts at the same column. The state's color shows its family, as kit status's does: yellow when a
# human owes the next move, blue when the next move is someone else's. The issue's number is cyan and the change's
# magenta, as in kit status. Detail is dim: the assignees, the age, and the branch.
#
#   state       glyph   color
#   idle        ⏳      yellow   nothing is in flight; the next move is a pick, kit issue <N>
#   in flight   🛫      blue     an open change closes it, or a branch for it is on origin
#
# see: kit/bin/kit-todo, modules/bash/forge.sh, modules/jq/forge.jq

use forge status

# todo:collect [--except] <login>… prints the open issues as one JSON array, oldest first, each with inflight:
# {change, branch} when an open change closes it or a branch on origin starts with its number, and null otherwise.
# With logins, only the issues assigned to one of them, or with --except, those assigned to none of them.
todo:collect() {
  local issues changes heads except=false
  [[ "${1:-}" == --except ]] && { except=true; shift; }
  issues=$(forge:open-issues) || return 1
  changes=$(forge:open-prs) || return 1
  heads=$(git ls-remote --heads origin 2>/dev/null | sed 's@.*refs/heads/@@' || true)
  jq --compact-output --slurp --argjson changes "$(jq --slurp . <<<"${changes}")" --arg heads "${heads}" \
    --arg logins "$*" --argjson except "${except}" '
    ($heads | split("\n") | map(select(. != ""))) as $heads
    | ($logins | split(" ") | map(select(. != ""))) as $logins
    | map(. as $issue
        | ($changes | map(select(.closes | any(.number == $issue.number))) | first | .number) as $change
        | ($heads | map(select(test("^\($issue.number)-"))) | first) as $branch
        | . + {inflight: (if $change == null and $branch == null then null
                          else {change: $change, branch: $branch} end)})
    | if ($logins | length) == 0 then .
      else map(select((.assignees | any(. as $a | $logins | index($a))) != $except)) end' \
    <<<"${issues}"
}

# todo:state-sgr <state> prints the color code of a state's family.
todo:state-sgr() {
  case "$1" in
    (idle)        printf '33' ;;
    (in\ flight)  printf '34' ;;
    (*)           printf '90' ;;
  esac
}

# todo:state-emoji <state> prints a state's glyph: one code point drawn two columns wide, so the next column stays
# aligned.
todo:state-emoji() {
  case "$1" in
    (idle)        printf '⏳' ;;
    (in\ flight)  printf '🛫' ;;
    (*)           printf '🔳' ;;
  esac
}

# todo:rows <issues-json> prints one line per issue with its cells joined by \x1f: the issue's ref, its title, the
# assignees, the state, the age in seconds, then the change and the branch in flight, each empty when there is none.
# The separator is \x1f, not a tab: a tab is IFS whitespace, and read would drop an empty assignees cell.
todo:rows() {
  jq --raw-output \
    '.[] | [.ref, .title, (.assignees | join(" ")), (if .inflight == null then "idle" else "in flight" end),
            .age, (.inflight.change // "" | tostring), (.inflight.branch // "")]
      | join("\u001f")' <<<"$1"
}

# todo:render [--json] [--today] [--mine | --others | --all | --owner=<login>…] prints the issues, one row each as
# the file header draws them, to stderr. With --json it prints todo:collect's array to stdout.
todo:render() {
  local json=false today=false choice=all arg doc when=''
  local -a owners=() except=()
  for arg in "$@"
  do
    case "${arg}" in
      (--json) json=true ;;
      (--today) today=true ;;
      (--mine) choice=mine ;;
      (--others) choice=others ;;
      (--all) choice=all ;;
      (--owner=?*) choice=owner; owners+=("${arg#--owner=}") ;;
      (*) error "unknown flag '${arg}'; run: kit help todo"; return 2 ;;
    esac
  done
  if [[ "${choice}" == mine || "${choice}" == others ]]
  then
    owners=("$(forge:me)") || return 1
    [[ "${choice}" == others ]] && except=(--except)
  fi
  doc=$(todo:collect "${except[@]}" "${owners[@]}") || return 1
  [[ "${today}" == false ]] || doc=$(jq --compact-output 'map(select(.age < 86400))' <<<"${doc}")
  if [[ "${json}" == true ]]
  then
    printf '%s\n' "${doc}"
    return 0
  fi
  if [[ "$(jq 'length' <<<"${doc}")" == 0 ]]
  then
    [[ "${today}" == false ]] || when=' opened in the last 24 hours'
    step "nothing to pick up: no open issue${when}${owners:+ assigned to ${owners[*]}}"
    return 0
  fi

  # Size the cells across the rows. A column no row fills is left out, and a row that has nothing for a column gets
  # blanks, so every title starts at the same column.
  local ref title assignees state age change branch text row glyph
  local ref_w=0 owner_w=0 state_w=0 age_w=0 change_w=0 branch_w=0 width
  while IFS=$'\x1f' read -r ref title assignees state age change branch
  do
    (( ${#ref} > ref_w )) && ref_w=${#ref}
    [[ "${choice}" != mine ]] && (( ${#assignees} > owner_w )) && owner_w=${#assignees}
    (( ${#state} > state_w )) && state_w=${#state}
    text=$(status:age "${age}")
    (( ${#text} > age_w )) && age_w=${#text}
    [[ -n "${change}" ]] && (( ${#change} + 1 > change_w )) && change_w=$(( ${#change} + 1 ))
    (( ${#branch} > branch_w )) && branch_w=${#branch}
  done < <(todo:rows "${doc}")
  # The title is last, and gets what the terminal's width leaves after the other cells, 20 at least. A cell with a
  # glyph takes six columns beyond its text: the separator, the glyph, which is drawn two wide, and a space.
  width=$(( $(status:width) - ref_w - (owner_w > 0 ? 6 + owner_w : 0) - 6 - state_w - 1 - age_w
            - (change_w > 0 ? 6 + change_w : 0) - (branch_w > 0 ? 6 + branch_w : 0) - 6 ))
  (( width < 20 )) && width=20

  while IFS=$'\x1f' read -r ref title assignees state age change branch
  do
    row=$(status:paint "${status_issue_sgr}" "$(status:cell "${ref}" "${ref_w}" right)")
    if (( owner_w > 0 ))
    then
      glyph='   '
      [[ -n "${assignees}" ]] && glyph='👤 '
      row+="${status_sep}${glyph}$(status:paint 90 "$(status:cell "${assignees}" "${owner_w}")")"
    fi
    row+="${status_sep}$(todo:state-emoji "${state}")"
    row+=" $(status:paint "$(todo:state-sgr "${state}")" "$(status:cell "${state}" "${state_w}")")"
    row+=" $(status:paint 90 "$(status:cell "$(status:age "${age}")" "${age_w}" right)")"
    if (( change_w > 0 ))
    then
      if [[ -n "${change}" ]]
      then
        row+="${status_sep}🔀 $(status:paint "${status_change_sgr}" "$(status:cell "#${change}" "${change_w}" right)")"
      else row+="${status_sep}   $(status:cell '' "${change_w}")"
      fi
    fi
    if (( branch_w > 0 ))
    then
      if [[ -n "${branch}" ]]
      then row+="${status_sep}$(status:paint 90 "🌿 $(status:cell "${branch}" "${branch_w}")")"
      else row+="${status_sep}   $(status:cell '' "${branch_w}")"
      fi
    fi
    (( ${#title} > width )) && title="${title:0:width-1}…"
    printf '%s%s🎫 %s\n' "${row}" "${status_sep}" "${title}" >&2
  done < <(todo:rows "${doc}")
}

# ----------------------------------------------------------------------------------------------------------------------
