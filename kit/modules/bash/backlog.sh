# shellcheck shell=bash
# backlog.sh - the open issues nothing is in flight for, oldest first, with the branch that claims one (sourced module)
# ----------------------------------------------------------------------------------------------------------------------
#
# kit backlog is the pick: the open issues of the repository on its forge, as forge:open-issues folds them, less the
# ones an open change closes, which are kit changes's. An issue with a branch for it on origin is claimed, and its
# row says so; one with neither is for the next kit issue <N>. The flags are kit status's: --mine is the issues
# assigned to the caller, --others the rest, unassigned ones included, --owner=<login> the named assignees', --all
# every one; --today keeps only the issues opened in the last 24 hours, whichever of those is chosen.
#
# A row is drawn by status:issue-row, as the issues under a change in kit changes are, without the state cell:
#
#    #7  👤 bob  40d  🌿 7-fix-the-thing  🎫 Fix the thing
#   #11  👤 ada  30d                      🎫 Third thing
#
# see: kit/bin/kit-backlog, modules/bash/status.sh, modules/bash/forge.sh, modules/jq/forge.jq

use forge status

# backlog:collect [--except] <login>… prints the open issues no open change closes, as one JSON array, oldest
# first, each with branch: the branch on origin that starts with its number, or null. With logins, only the issues
# assigned to one of them, or with --except, those assigned to none of them.
backlog:collect() {
  local issues changes heads except=false
  [[ "${1:-}" == --except ]] && { except=true; shift; }
  issues=$(forge:open-issues) || return 1
  changes=$(forge:open-prs) || return 1
  heads=$(git ls-remote --heads origin 2>/dev/null | sed 's@.*refs/heads/@@' || true)
  jq --compact-output --slurp --argjson changes "$(jq --slurp . <<<"${changes}")" --arg heads "${heads}" \
    --arg logins "$*" --argjson except "${except}" '
    ($heads | split("\n") | map(select(. != ""))) as $heads
    | ($logins | split(" ") | map(select(. != ""))) as $logins
    | ($changes | map(.closes[].number)) as $closed
    | map(select(.number as $n | $closed | index($n) | not)
          | . + {branch: (. as $issue | $heads | map(select(test("^\($issue.number)-"))) | first)})
    | if ($logins | length) == 0 then .
      else map(select((.assignees | any(. as $a | $logins | index($a))) != $except)) end' \
    <<<"${issues}"
}

# backlog:render [--json] [--today] [--mine | --others | --all | --owner=<login>…] prints the issues, one row each,
# to stderr. With --json it prints backlog:collect's array to stdout.
backlog:render() {
  local json=false today=false choice=all arg doc when='' widths ref_w owner_w age_w branch_w issue
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
      (*) error "unknown flag '${arg}'; run: kit help backlog"; return 2 ;;
    esac
  done
  if [[ "${choice}" == mine || "${choice}" == others ]]
  then
    owners=("$(forge:me)") || return 1
    [[ "${choice}" == others ]] && except=(--except)
  fi
  doc=$(backlog:collect "${except[@]}" "${owners[@]}") || return 1
  [[ "${today}" == false ]] || doc=$(jq --compact-output 'map(select(.age < 86400))' <<<"${doc}")
  if [[ "${json}" == true ]]
  then
    printf '%s\n' "${doc}"
    return 0
  fi
  if [[ "$(jq 'length' <<<"${doc}")" == 0 ]]
  then
    [[ "${today}" == false ]] || when=' opened in the last 24 hours'
    step "nothing in the backlog: no open issue${when}${owners:+ assigned to ${owners[*]}}"
    return 0
  fi
  # The assignee column goes under --mine, as the author column goes under kit status --mine.
  read -r ref_w owner_w age_w branch_w <<<"$(status:issue-widths "${doc}")"
  [[ "${choice}" != mine ]] || owner_w=0
  widths="${ref_w} ${owner_w} ${age_w} ${branch_w}"
  while IFS= read -r issue
  do
    [[ -n "${issue}" ]] || continue
    status:issue-row "${issue}" "${widths}" '' ''
  done < <(jq --compact-output '.[]' <<<"${doc}")
}

# ----------------------------------------------------------------------------------------------------------------------
