# shellcheck shell=bash
# changes.sh - the open changes, each with the issues it closes, drawn as kit flow draws them (sourced module)
# ----------------------------------------------------------------------------------------------------------------------
# shellcheck disable=SC2154  # kit_jq is exported by modules/bash/1st.sh
#
# The changes stage of kit flow is the reviewer's picture: every open change of the repository on its forge, as
# forge:open-prs folds them, each drawn as a pull request's row, and under it the issues it closes, each drawn as
# the backlog stage draws an issue, in flight. The flags are kit flow's: --mine is the changes the caller opened,
# --others the rest, --owner=<login> the named authors', --all every open change. Every change's row ends with its
# branch. In a worktree the stage is the change its branch holds, whoever opened it.
#
# see: kit/bin/kit-flow, modules/bash/flow.sh, modules/bash/forge.sh

use forge flow workspaces

# changes:documents [--all | [--mine] [--others] [--owner=<login>…]] [<number>…] prints the open changes the flags
# choose as one JSON array, oldest first, each with the issues it closes under closes. The owner flags combine as
# flow:filter says. Numbers given keep only the changes numbered so, or that close an issue numbered so. In a
# worktree it is the change its branch holds, and the owner flags change nothing. It returns 2 on a word that is not
# a number.
changes:documents() {
  local arg prs filter branch
  local -a selection=() issues=()
  for arg in "$@"
  do
    case "${arg}" in
      (--json | --mine | --others | --all | --owner=?* | --) ;;
      (*)
        if [[ ! "${arg}" =~ ^#?[0-9]+$ ]]
        then
          error "'${arg}' is not a number: 7 or #7; run: kit help flow"
          return 2
        fi
        issues+=("${arg#\#}")
    esac
  done
  prs=$(forge:open-prs) || return 1
  if flow:in-worktree
  then
    branch=$(git symbolic-ref --quiet --short HEAD 2>/dev/null || true)  # a detached HEAD has no change
    prs=$(jq --compact-output --arg branch "${branch}" 'select(.branch == $branch)' <<<"${prs}")
  else
    filter=$(flow:filter-of "$@") || return 1
    read -r -a selection <<<"${filter}"
    prs=$(flow:select "${selection[@]}" <<<"${prs}")
  fi
  if (( ${#issues[@]} > 0 ))
  then
    prs=$(jq --compact-output -L "${kit_jq}" --argjson refs "[$(IFS=,; printf '%s' "${issues[*]}")]" \
      'import "changes" as changes; changes::concerning($refs)' <<<"${prs}")
  fi
  jq --slurp --compact-output . <<<"${prs}"
}

# changes:draw <changes-json> <choice> prints the changes, one block each, to stderr: the change's row, and under it
# the issues it closes. With none it says so in one dim line. <choice> mine leaves the author column out.
changes:draw() {
  local doc=$1 choice=$2
  if [[ "$(jq 'length' <<<"${doc}")" == 0 ]]
  then
    flow:none 'no open pull request'
    return 0
  fi
  flow:pr-table "${choice}" "$(jq --compact-output '.[]' <<<"${doc}")"
}

# changes:render [--json] [--all | [--mine] [--others] [--owner=<login>…]] [<number>…] prints the changes as
# changes:draw does, or with --json as changes:documents does, to stdout, each saying where it is from, the forge, as
# workspaces:placed says.
changes:render() {
  local doc
  doc=$(changes:documents "$@") || return $?
  if [[ " $* " == *" --json "* ]]
  then
    workspaces:placed forge "${doc}"
    return
  fi
  changes:draw "${doc}" "$(flow:choice-of "$@")"
}

# ----------------------------------------------------------------------------------------------------------------------
