# shellcheck shell=bash
# todo.sh - the forge's open issues, oldest first, each with what is already in flight for it (sourced module)
# ----------------------------------------------------------------------------------------------------------------------
# shellcheck disable=SC2154  # status_sep is set by modules/bash/status.sh
#
# kit todo is the pick: the open issues of the repository on its forge, as forge:open-issues folds them, and for each
# whether work has started, which is when an open change closes it or a branch for it is on origin. An issue with
# nothing in flight is one to take up with kit issue <N>. The flags are kit status's: --mine is the issues assigned to
# the caller, --owner=<login> the named assignees', --all every open issue.
#
# A row is drawn as kit status draws a pull request, its cells separated by status_sep and sized across the rows:
#
#   🎫 #7 Fix the thing · 👤 bob · 🛫 in flight 40d · 🔀 #3 · 🌿 7-fix-the-thing
#
# that is the issue with its title, its assignees (dropped under --mine, as kit status drops the author, and without
# the glyph when it has none), its state with the issue's age, right-aligned, then the change and the branch in
# flight. The title is clipped to what the terminal's width leaves after the other cells. The state's color shows its
# family, as kit status's does: yellow when a human owes the next move, blue when the next move is someone else's.
# Detail is dim: the assignees, the age, the change, and the branch.
#
#   state       glyph   color
#   idle        ⏳      yellow   nothing is in flight; the next move is a pick, kit issue <N>
#   in flight   🛫      blue     an open change closes it, or a branch for it is on origin
#
# see: kit/bin/kit-todo, modules/bash/forge.sh, modules/jq/forge.jq

use forge status

# todo:collect <login>… prints the open issues as one JSON array, oldest first, each with inflight: {change, branch}
# when an open change closes it or a branch on origin starts with its number, and null otherwise. With logins, only
# the issues assigned to one of them.
todo:collect() {
  local issues changes heads
  issues=$(forge:open-issues) || return 1
  changes=$(forge:open-prs) || return 1
  heads=$(git ls-remote --heads origin 2>/dev/null | sed 's@.*refs/heads/@@' || true)
  jq --compact-output --slurp --argjson changes "$(jq --slurp . <<<"${changes}")" --arg heads "${heads}" \
    --arg logins "$*" '
    ($heads | split("\n") | map(select(. != ""))) as $heads
    | ($logins | split(" ") | map(select(. != ""))) as $logins
    | map(. as $issue
        | ($changes | map(select(.closes | any(.number == $issue.number))) | first | .number) as $change
        | ($heads | map(select(test("^\($issue.number)-"))) | first) as $branch
        | . + {inflight: (if $change == null and $branch == null then null
                          else {change: $change, branch: $branch} end)})
    | if ($logins | length) == 0 then . else map(select(.assignees | any(. as $a | $logins | index($a)))) end' \
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

# todo:rows <issues-json> prints one line per issue with its cells joined by \x1f: the title cell's text, the
# assignees, the state, the age in seconds, then the change and the branch in flight, each empty when there is none.
# The separator is \x1f, not a tab: a tab is IFS whitespace, and read would drop an empty assignees cell.
todo:rows() {
  jq --raw-output \
    '.[] | ["\(.ref) \(.title)", (.assignees | join(" ")), (if .inflight == null then "idle" else "in flight" end),
            .age, (.inflight.change // "" | tostring), (.inflight.branch // "")]
      | join("\u001f")' <<<"$1"
}

# todo:render [--json] [--mine | --all | --owner=<login>…] prints the issues, one row each as the file header draws
# them, to stderr. With --json it prints todo:collect's array to stdout.
todo:render() {
  local json=false choice=all arg doc
  local -a owners=()
  for arg in "$@"
  do
    case "${arg}" in
      (--json) json=true ;;
      (--mine) choice=mine ;;
      (--all) choice=all ;;
      (--owner=?*) choice=owner; owners+=("${arg#--owner=}") ;;
      (*) error "unknown flag '${arg}'; run: kit help todo"; return 2 ;;
    esac
  done
  if [[ "${choice}" == mine ]]
  then
    owners=("$(forge:me)") || return 1
  fi
  doc=$(todo:collect "${owners[@]}") || return 1
  if [[ "${json}" == true ]]
  then
    printf '%s\n' "${doc}"
    return 0
  fi
  if [[ "$(jq 'length' <<<"${doc}")" == 0 ]]
  then
    step "nothing to pick up: no open issue${owners:+ assigned to ${owners[*]}}"
    return 0
  fi

  # Size the cells across the rows. The title gets what the terminal's width leaves after the others, 20 at least.
  local text assignees state age change branch tail row glyph
  local title_w=0 owner_w=0 state_w=0 age_w=0 tail_w=0 width
  while IFS=$'\x1f' read -r text assignees state age change branch
  do
    (( ${#text} > title_w )) && title_w=${#text}
    [[ "${choice}" != mine ]] && (( ${#assignees} > owner_w )) && owner_w=${#assignees}
    (( ${#state} > state_w )) && state_w=${#state}
    text=$(status:age "${age}")
    (( ${#text} > age_w )) && age_w=${#text}
    tail=0
    [[ -n "${change}" ]] && tail=$(( tail + 3 + 3 + 1 + ${#change} ))
    [[ -n "${branch}" ]] && tail=$(( tail + 3 + 3 + ${#branch} ))
    (( tail > tail_w )) && tail_w=${tail}
  done < <(todo:rows "${doc}")
  width=$(( $(status:width) - 3 - (owner_w > 0 ? owner_w + 6 : 0) - 3 - 3 - state_w - 1 - age_w - tail_w ))
  (( width < 20 )) && width=20
  (( title_w > width )) && title_w=${width}

  while IFS=$'\x1f' read -r text assignees state age change branch
  do
    row="🎫 $(status:cell "${text}" "${title_w}")"
    if (( owner_w > 0 ))
    then
      glyph='   '
      [[ -n "${assignees}" ]] && glyph='👤 '
      row+="${status_sep}${glyph}$(status:paint 90 "$(status:cell "${assignees}" "${owner_w}")")"
    fi
    row+="${status_sep}$(todo:state-emoji "${state}")"
    row+=" $(status:paint "$(todo:state-sgr "${state}")" "$(status:cell "${state}" "${state_w}")")"
    row+=" $(status:paint 90 "$(status:cell "$(status:age "${age}")" "${age_w}" right)")"
    [[ -n "${change}" ]] && row+="${status_sep}$(status:paint 90 "🔀 #${change}")"
    [[ -n "${branch}" ]] && row+="${status_sep}$(status:paint 90 "🌿 ${branch}")"
    printf '%s\n' "${row}" >&2
  done < <(todo:rows "${doc}")
}

# ----------------------------------------------------------------------------------------------------------------------
