# shellcheck shell=bash
# across.sh - kit flow across a workspace: its members, their checkouts, and their stages merged (sourced module)
# ----------------------------------------------------------------------------------------------------------------------
# A workspace's stage is the stage as kit flow draws it in each member's checkout, each under that checkout's own
# kit.yml, run at once, and merged into one document, {rows, errors}: every row, or every object of a stage that draws
# one, tagged with its member, host and path; and a member that could not be read as {member, error}, its line. The
# members keep the workspace's order. A member with no checkout here draws no stage.
#
# see: docs/workspaces.md, modules/bash/workspaces.sh
use workspaces

# across:members <workspace> prints "<member>\t<checkout>" for each of the workspace's members, in its order, the
# checkout empty for a member not cloned under the roots. The roots are walked once. It returns 2, after workspaces'
# line, for a workspace that is neither a group nor a repository.
across:members() {
  local resolved
  resolved=$(workspaces:resolve "$1") || return 2
  jq --raw-output '.members[] | [.repository, (.checkout // "")] | @tsv' <<<"${resolved}"
}

# across:merge <dir> <kind> prints the document of what each member printed, from <dir>'s <n>.member, <n>.out, <n>.err,
# and <n>.status, for n from 0 up. For an array stage, each row a member printed gains its member; for an object
# stage, the object does. A member that exited other than 0, or printed what does not parse, is its stderr's last
# line, less a leading ❌, under errors.
across:merge() {
  local dir=$1 kind=$2 n=0 member out line
  local -a rows=() errors=()
  while [[ -f "${dir}/${n}.member" ]]
  do
    member=$(<"${dir}/${n}.member")
    if [[ "$(<"${dir}/${n}.status")" == 0 ]] && out=$(jq --compact-output --arg member "${member}" --arg kind "${kind}" \
      'if $kind == "array" then map(. + {member: $member})[] else . + {member: $member} end' "${dir}/${n}.out" \
      2>/dev/null)
    then
      [[ -z "${out}" ]] || rows+=("${out}")
    else
      line=$(grep -v '^[[:space:]]*$' "${dir}/${n}.err" | tail -n 1 || true)
      line=${line#❌ }
      errors+=("$(jq --null-input --compact-output --arg member "${member}" --arg error "${line}" \
        '{member: $member, error: $error}')")
    fi
    n=$(( n + 1 ))
  done
  jq --null-input --compact-output --argjson rows "$(printf '%s\n' "${rows[@]}" | jq --slurp --compact-output .)" \
    --argjson errors "$(printf '%s\n' "${errors[@]}" | jq --slurp --compact-output .)" '{rows: $rows, errors: $errors}'
}

# ----------------------------------------------------------------------------------------------------------------------
