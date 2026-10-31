# shellcheck shell=bash
# changes.sh - the open changes, each with the issues it closes, drawn as kit status draws them (sourced module)
# ----------------------------------------------------------------------------------------------------------------------
# shellcheck disable=SC2154  # kit_jq is exported by modules/bash/1st.sh
#
# kit changes is the reviewer's picture: every open change of the repository on its forge, as forge:open-prs folds
# them, each drawn as kit status draws a pull request, and under it the issues it closes, each drawn as kit backlog
# draws an issue, in flight. The flags are kit status's: --mine is the changes the caller opened, --others the rest,
# --owner=<login> the named authors', --all every open change. Every change's row ends with its branch, since there
# is no worktree in the picture to say which are checked out here.
#
# see: kit/bin/kit-changes, modules/bash/status.sh, modules/bash/forge.sh

use forge status

# changes:render [--json] [--all | [--mine] [--others] [--owner=<login>…]] [<number>…] prints the changes, one block
# each, to stderr. With --json it prints the change documents as one array to stdout. The owner flags combine as
# status:filter says; numbers given keep only the changes numbered so, or that close an issue numbered so. With no
# change to list it prints nothing, or [] with --json. It returns 2 on a word that is not a number.
changes:render() {
  local json=false mine=false others=false choice arg prs filter
  local -a owners=() selection=() issues=()
  for arg in "$@"
  do
    case "${arg}" in
      (--json) json=true ;;
      (--mine) mine=true ;;
      (--others) others=true ;;
      (--all) ;;
      (--owner=?*) owners+=("${arg#--owner=}") ;;
      (--) ;;
      (*)
        if [[ ! "${arg}" =~ ^#?[0-9]+$ ]]
        then
          error "'${arg}' is not a number: 7 or #7; run: kit help changes"
          return 2
        fi
        issues+=("${arg#\#}")
    esac
  done
  choice=$(status:choice "${mine}" "${others}" "${#owners[@]}")
  filter=$(status:filter "${mine}" "${others}" "${owners[@]}") || return 1
  read -r -a selection <<<"${filter}"
  prs=$(forge:open-prs) || return 1
  prs=$(status:select "${selection[@]}" <<<"${prs}")
  if (( ${#issues[@]} > 0 ))
  then
    prs=$(jq --compact-output -L "${kit_jq}" --argjson refs "[$(IFS=,; printf '%s' "${issues[*]}")]" \
      'import "changes" as changes; changes::concerning($refs)' <<<"${prs}")
  fi
  if [[ "${json}" == true ]]
  then
    jq --slurp --compact-output . <<<"${prs}"
    return 0
  fi
  # Nothing to list prints nothing, so a script reads an empty answer as none.
  [[ -n "${prs}" ]] || return 0
  status:pr-table "${choice}" "${prs}" always between
}

# ----------------------------------------------------------------------------------------------------------------------
