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

# across:flow <workspace> <word>… draws what kit flow's words ask for across the workspace: a stage with --json or --jq
# as across:stage gives it. A pull request's number names one repository's, so it is refused, with 2.
across:flow() {
  local workspace=$1 stage='' word form=''
  shift
  case "${1:-}" in
    (backlog | underway | desk | worktrees | changes | incoming | builds | releases)
      stage=$1
      shift
  esac
  for word in "$@"
  do
    case "${word}" in
      (--json | --jq) form=${word} ;;
      ([0-9]* | '#'[0-9]*)
        error "a number names a pull request in one repository; run kit flow changes <number> in it"
        return 2
    esac
  done
  if [[ -n "${stage}" && -n "${form}" ]]
  then
    across:stage "${workspace}" "${stage}" "$@" | if [[ "${form}" == --jq ]]; then jq .; else cat; fi
    return "${PIPESTATUS[0]}"
  fi
  error "--workspace draws with --json for now"
  return 2
}

# across:stage <workspace> <stage> <argument>… runs kit flow <stage> with the arguments, --jq as --json, in every
# cloned member's checkout at once, and prints the document across:merge makes of them. It returns 0 whatever a member
# did, and 2 for a workspace across:members refuses.
across:stage() {
  local workspace=$1 stage=$2 members dir member checkout n=0 kind=array arg
  local -a args=()
  shift 2
  for arg in "$@"
  do
    [[ "${arg}" == --jq ]] && arg=--json
    args+=("${arg}")
  done
  members=$(across:members "${workspace}") || return 2
  file:scratch dir
  while IFS=$'\t' read -r member checkout
  do
    [[ -n "${checkout}" ]] || continue
    printf '%s' "${member}" > "${dir}/${n}.member"
    across:run "${checkout}" "${dir}/${n}" "${workspace}" "${stage}" "${args[@]}" &
    n=$(( n + 1 ))
  done <<<"${members}"
  wait || true
  case "${stage}" in
    (desk | worktrees | incoming) kind=object
  esac
  across:merge "${dir}" "${kind}"
}

# across:run <checkout> <prefix> <workspace> <word>… runs kit flow with the words in the checkout, its output in
# <prefix>.out and <prefix>.err and its exit in <prefix>.status. A checkout gone since the roots were walked is its
# line.
across:run() {
  local checkout=$1 prefix=$2 workspace=$3 code
  shift 3
  if [[ ! -d "${checkout}" ]]
  then
    printf '❌ %s is gone; run: kit workspaces --show %s\n' "${checkout}" "${workspace}" > "${prefix}.err"
    : > "${prefix}.out"
    printf 1 > "${prefix}.status"
    return 0
  fi
  ( cd "${checkout}" && run kit flow "$@" ) > "${prefix}.out" 2> "${prefix}.err" && code=0 || code=$?
  printf '%s' "${code}" > "${prefix}.status"
}

# ----------------------------------------------------------------------------------------------------------------------
