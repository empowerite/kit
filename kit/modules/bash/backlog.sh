# shellcheck shell=bash
# backlog.sh - open issues not closed by any open pull request, oldest first, with the branch for each (sourced module)
# ----------------------------------------------------------------------------------------------------------------------
# shellcheck disable=SC2154  # kit_jq is exported by modules/bash/1st.sh
#
# The backlog stage of kit flow is the pick: the open issues of the repository on its forge, as forge:open-issues
# folds them, less the ones an open pull request closes. The underway stage lists those. An issue with a branch for it
# on the primary remote is claimed, and its row says so; one with neither is for the next kit work <N>. The flags are
# kit flow's: --mine is the issues assigned to the caller, --others the rest, unassigned ones included,
# --owner=<login> the named assignees', --all every one. --today keeps only the issues opened in the last 24 hours,
# whichever of those is chosen.
#
# A row is drawn by flow:issue-row, as the issues under a change are, without the state cell:
#
#    #7  👤 bob  40d  🌿 7-fix-the-thing  🎫 Fix the thing
#   #11  👤 ada  30d                      🎫 Third thing
#
# see: kit/bin/kit-flow, modules/bash/flow.sh, modules/bash/underway.sh, modules/bash/forge.sh,
# modules/jq/forge.jq

use forge flow workspaces

# backlog:collect [--except] <login>… prints the open issues not closed by any open change, as one JSON array, oldest
# first, each with branch: the branch on the primary remote that starts with its number, or null. The branches are the
# primary's as of the last fetch, which the picture's incoming stage makes first, not asked of the remote, which takes
# seconds a stage cannot spare. With logins, only
# the issues assigned to one of them, or with --except, those assigned to none of them.
backlog:collect() {
  local issues changes heads primary except=false
  [[ "${1:-}" == --except ]] && { except=true; shift; }
  primary=$(forge:primary) || return $?
  # A forge without issues of its own has none open.
  issues=$(forge:open-issues) || (( $? == 3 )) || return 1
  changes=$(forge:open-prs) || return 1
  heads=$(git for-each-ref --format='%(refname:lstrip=3)' -- "refs/remotes/${primary}/") || return 1
  jq --compact-output --slurp -L "${kit_jq}" --argjson changes "$(jq --slurp . <<<"${changes}")" \
    --arg heads "${heads}" --arg logins "$*" --argjson except "${except}" '
    import "backlog" as backlog;
    backlog::unclaimed($changes; $heads | split("\n") | map(select(. != "" and . != "HEAD"));
      $logins | split(" ") | map(select(. != "")); $except)' <<<"${issues}"
}

# backlog:documents [--today] [--all | [--mine] [--others] [--owner=<login>…]] prints the issues the flags choose as
# one JSON array, oldest first. The owner flags combine as flow:filter says.
backlog:documents() {
  local filter doc
  local -a selection=()
  filter=$(flow:filter-of "$@") || return 1
  read -r -a selection <<<"${filter}"
  doc=$(backlog:collect "${selection[@]}") || return 1
  [[ " $* " != *" --today "* ]] || doc=$(jq --compact-output 'map(select(.age < 86400))' <<<"${doc}")
  printf '%s\n' "${doc}"
}

# backlog:draw <issues-json> <choice> prints the issues, one row each, to stderr. With none it says so in one dim
# line. <choice> mine leaves the assignee column out, as a change's row drops the author column under --mine.
backlog:draw() {
  local doc=$1 choice=$2 widths ref_w owner_w age_w branch_w issue
  if [[ "$(jq 'length' <<<"${doc}")" == 0 ]]
  then
    flow:none 'no open issue without a pull request'
    return 0
  fi
  read -r ref_w owner_w age_w branch_w <<<"$(flow:issue-widths "${doc}")"
  [[ "${choice}" != mine ]] || owner_w=0
  widths="${ref_w} ${owner_w} ${age_w} ${branch_w}"
  while IFS= read -r issue
  do
    [[ -n "${issue}" ]] || continue
    flow:issue-row "${issue}" "${widths}" '' ''
  done < <(jq --compact-output '.[]' <<<"${doc}")
}

# backlog:render [--json] [--today] [--all | [--mine] [--others] [--owner=<login>…]] prints the issues as
# backlog:draw does, or with --json as backlog:documents does, to stdout, each saying where it is from, as
# workspaces:placed says, the forge its source where the tracker names none.
backlog:render() {
  local doc
  doc=$(backlog:documents "$@") || return 1
  if [[ " $* " == *" --json "* ]]
  then
    workspaces:placed forge "${doc}"
    return
  fi
  backlog:draw "${doc}" "$(flow:choice-of "$@")"
}

# ----------------------------------------------------------------------------------------------------------------------
