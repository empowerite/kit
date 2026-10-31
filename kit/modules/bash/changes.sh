# shellcheck shell=bash
# changes.sh - the open changes, each with the issues it closes, drawn as kit status draws them (sourced module)
# ----------------------------------------------------------------------------------------------------------------------
#
# kit changes is the reviewer's picture: every open change of the repository on its forge, as forge:open-prs folds
# them, each drawn as kit status draws a pull request, and under it the issues it closes, each drawn as kit backlog
# draws an issue, in flight. The flags are kit status's: --mine is the changes the caller opened, --others the rest,
# --owner=<login> the named authors', --all every open change. Every change's row ends with its branch, since there
# is no worktree in the picture to say which are checked out here.
#
# see: kit/bin/kit-changes, modules/bash/status.sh, modules/bash/forge.sh

use forge status

# changes:render [--json] [--mine | --others | --all | --owner=<login>…] prints the changes, one block each, to
# stderr. With --json it prints the change documents as one array to stdout.
changes:render() {
  local json=false choice=all arg prs me by=''
  local -a owners=() except=()
  for arg in "$@"
  do
    case "${arg}" in
      (--json) json=true ;;
      (--mine) choice=mine ;;
      (--others) choice=others ;;
      (--all) choice=all ;;
      (--owner=?*) choice=owner; owners+=("${arg#--owner=}") ;;
      (*) error "unknown flag '${arg}'; run: kit help changes"; return 2 ;;
    esac
  done
  if [[ "${choice}" == mine || "${choice}" == others ]]
  then
    me=$(forge:me) || return 1
    owners=("${me}")
    [[ "${choice}" == others ]] && except=(--except)
  fi
  prs=$(forge:open-prs) || return 1
  prs=$(status:select "${except[@]}" "${owners[@]}" <<<"${prs}")
  if [[ "${json}" == true ]]
  then
    jq --slurp --compact-output . <<<"${prs}"
    return 0
  fi
  if [[ -z "${prs}" ]]
  then
    case "${choice}" in
      (mine) by=" by ${me}" ;;
      (others) by=" by anyone but ${me}" ;;
      (owner) by=" by ${owners[*]}" ;;
    esac
    step "no open change${by}"
    return 0
  fi
  status:pr-table "${choice}" "${prs}" always between
}

# ----------------------------------------------------------------------------------------------------------------------
