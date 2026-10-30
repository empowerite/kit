# shellcheck shell=bash
# todo.sh - the forge's open issues, oldest first, each with what is already in flight for it (sourced module)
# ----------------------------------------------------------------------------------------------------------------------
#
# kit todo is the pick: the open issues of the repository on its forge, as forge:open-issues folds them, and for each
# whether work has started, which is when an open change closes it or a branch for it is on origin. An issue with
# nothing in flight is one to take up with kit issue <N>. The flags are kit status's: --mine is the issues assigned to
# the caller, --owner=<login> the named assignees', --all every open issue.
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

# todo:render [--json] [--mine | --all | --owner=<login>…] prints the issues, one line each: reference, title,
# assignees, age, and what is in flight. With --json it prints todo:collect's array.
todo:render() {
  local json=false choice=all arg doc ref title assignees age change branch flight ref_width title_width who_width
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
  read -r ref_width title_width who_width < <(jq --raw-output \
    '[([.[].ref | length] | max), ([.[].title | length] | max | if . > 60 then 60 else . end),
      ([.[].assignees | join(" ") | length] | max)] | @tsv' <<<"${doc}")
  # Fields are separated by \x1f, not a tab: a tab is IFS whitespace, and read would drop an empty assignees field.
  while IFS=$'\x1f' read -r ref title assignees age change branch
  do
    flight=''
    if [[ -n "${change}${branch}" ]]
    then
      flight="  · in flight:${change:+ #${change}}${branch:+ ${branch}}"
    fi
    printf '%s  %s  %s  %s%s\n' "$(status:cell "${ref}" "${ref_width}")" "$(status:cell "${title}" "${title_width}")" \
      "$(status:cell "${assignees}" "${who_width}")" "$(status:age "${age}")" "${flight}"
  done < <(jq --raw-output \
    '.[] | [.ref, .title, (.assignees | join(" ")), .age, (.inflight.change // "" | tostring), (.inflight.branch // "")]
      | join("\u001f")' <<<"${doc}")
}

# ----------------------------------------------------------------------------------------------------------------------
