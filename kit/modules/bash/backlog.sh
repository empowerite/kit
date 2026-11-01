# shellcheck shell=bash
# backlog.sh - the open issues nothing is in flight for, oldest first, with the branch that claims one (sourced module)
# ----------------------------------------------------------------------------------------------------------------------
# shellcheck disable=SC2154  # kit_jq is exported by modules/bash/1st.sh
#
# The backlog stage of kit status is the pick: the open issues of the repository on its forge, as forge:open-issues
# folds them, less the ones an open change closes, which the changes stage draws. An issue with a branch for it on
# origin is claimed, and its row says so; one with neither is for the next kit issue <N>. The flags are kit status's:
# --mine is the issues assigned to the caller, --others the rest, unassigned ones included, --owner=<login> the named
# assignees', --all every one; --today keeps only the issues opened in the last 24 hours, whichever of those is chosen.
#
# A row is drawn by status:issue-row, as the issues under a change are, without the state cell:
#
#    #7  👤 bob  40d  🌿 7-fix-the-thing  🎫 Fix the thing
#   #11  👤 ada  30d                      🎫 Third thing
#
# see: kit/bin/kit-status, modules/bash/status.sh, modules/bash/forge.sh, modules/jq/forge.jq

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
  jq --compact-output --slurp -L "${kit_jq}" --argjson changes "$(jq --slurp . <<<"${changes}")" \
    --arg heads "${heads}" --arg logins "$*" --argjson except "${except}" '
    import "backlog" as backlog;
    backlog::unclaimed($changes; $heads | split("\n") | map(select(. != ""));
      $logins | split(" ") | map(select(. != "")); $except)' <<<"${issues}"
}

# backlog:documents [--today] [--all | [--mine] [--others] [--owner=<login>…]] prints the issues the flags choose as
# one JSON array, oldest first. The owner flags combine as status:filter says.
backlog:documents() {
  local filter doc
  local -a selection=()
  filter=$(status:filter-of "$@") || return 1
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
    status:none 'no open issue without a pull request'
    return 0
  fi
  read -r ref_w owner_w age_w branch_w <<<"$(status:issue-widths "${doc}")"
  [[ "${choice}" != mine ]] || owner_w=0
  widths="${ref_w} ${owner_w} ${age_w} ${branch_w}"
  while IFS= read -r issue
  do
    [[ -n "${issue}" ]] || continue
    status:issue-row "${issue}" "${widths}" '' ''
  done < <(jq --compact-output '.[]' <<<"${doc}")
}

# backlog:render [--json] [--today] [--all | [--mine] [--others] [--owner=<login>…]] prints the issues as
# backlog:draw does, or with --json as backlog:documents does, to stdout.
backlog:render() {
  local doc
  doc=$(backlog:documents "$@") || return 1
  if [[ " $* " == *" --json "* ]]
  then
    printf '%s\n' "${doc}"
    return 0
  fi
  backlog:draw "${doc}" "$(status:choice-of "$@")"
}

# ----------------------------------------------------------------------------------------------------------------------
