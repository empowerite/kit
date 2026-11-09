# shellcheck shell=bash
# across.sh - kit flow across a workspace: its members, their checkouts, and their stages merged (sourced module)
# ----------------------------------------------------------------------------------------------------------------------
# shellcheck disable=SC2154  # flow_glyphs is set by modules/bash/flow.sh
# A workspace's stage is the stage as kit flow draws it in each member's checkout, each under that checkout's own
# kit.yml, run at once, and merged into one document, {rows, errors}: every row, or every object of a stage that draws
# one, tagged with its member, host and path; and a member that could not be read as {member, error}, its line. The
# members keep the workspace's order. A member with no checkout here draws no stage.
#
# see: docs/workspaces.md, modules/bash/workspaces.sh
use workspaces flow

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
      line=$(across:line "${dir}/${n}.err")
      errors+=("$(jq --null-input --compact-output --arg member "${member}" --arg error "${line}" \
        '{member: $member, error: $error}')")
    fi
    n=$(( n + 1 ))
  done
  jq --null-input --compact-output --argjson rows "$(printf '%s\n' "${rows[@]}" | jq --slurp --compact-output .)" \
    --argjson errors "$(printf '%s\n' "${errors[@]}" | jq --slurp --compact-output .)" '{rows: $rows, errors: $errors}'
}

# across:line <file> prints the last line of what a member said on stderr, less a leading ❌.
across:line() {
  local line
  line=$(grep -v '^[[:space:]]*$' "$1" | tail -n 1 || true)
  printf '%s\n' "${line#❌ }"
}

# across:flow <workspace> <word>… draws what kit flow's words ask for across the workspace: --stages, repos first; repos;
# and a stage with --json or --jq as across:stage gives it. No workspace is the repository here, for repos alone. A
# pull request's number names one repository's, so it is refused, with 2.
across:flow() {
  local workspace=$1 stage='' word form=''
  shift
  if [[ "${1:-}" == --stages ]]
  then
    workspaces:kind "${workspace}" >/dev/null || return 2
    printf 'repos\n'
    settings:stages
    return
  fi
  case "${1:-}" in
    (repos | backlog | underway | desk | worktrees | changes | incoming | builds | releases)
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
  if [[ "${stage}" == repos ]]
  then
    across:repos-form "${workspace}" "${form}"
    return
  fi
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

# across:repos-form <workspace> <form> prints the repos stage, as JSON for --json, formatted for --jq, else drawn.
across:repos-form() {
  local doc
  doc=$(across:repos "$1") || return $?
  case "$2" in
    (--json) printf '%s\n' "${doc}" ;;
    (--jq) jq . <<<"${doc}" ;;
    (*) across:repos-text "${doc}"
  esac
}

# across:repos <workspace> prints the repos stage's document, {rows, errors}: a row per member, in the workspace's
# order, cloned or not, as workspaces:member-json gives it, with its desk's branch, ahead, and behind, each null for a
# member not cloned, and error, the desk's line where it could not be read, else null. No workspace is the repository
# here, by here; outside one it returns 2.
across:repos() {
  local resolved top member row
  if [[ -n "$1" ]]
  then
    resolved=$(workspaces:resolve "$1") || return 2
  else
    if ! top=$(git rev-parse --show-toplevel 2>/dev/null)
    then
      error "repos lists a workspace's repositories: run it in one, or give --workspace"
      return 2
    fi
    member=$(workspaces:member-of "${top}")
    if [[ -n "${member}" ]]
    then
      row=$(workspaces:member-json "${member}" here "$(printf '%s\t%s' "${member}" "${top}")")
    else
      row=$(jq --null-input --compact-output --arg top "${top}" --argjson remotes "$(workspaces:remotes-json)" \
        '{repository: $top, host: null, path: null, forge: null, remotes: $remotes, checkout: $top,
          checkouts: [$top], by: "here"}')
    fi
    resolved=$(jq --compact-output '{members: [.]}' <<<"${row}")
  fi
  across:repos-of "${resolved}" "${1:-${member:-${top}}}"
}

# across:repos-of <resolved> <workspace> reads each cloned member's desk at once and prints the repos document of the
# members <resolved> holds, as across:repos says.
across:repos-of() {
  local resolved=$1 workspace=$2 dir checkout n=0 desk error
  local -a checkouts=() rows=()
  file:scratch dir
  readarray -t checkouts < <(jq --raw-output '.members[] | .checkout // ""' <<<"${resolved}")
  for checkout in "${checkouts[@]}"
  do
    [[ -z "${checkout}" ]] || across:run "${checkout}" "${dir}/${n}" "${workspace}" desk --json &
    n=$(( n + 1 ))
  done
  wait || true
  for n in "${!checkouts[@]}"
  do
    desk=null error=''
    if [[ -f "${dir}/${n}.status" ]]
    then
      if [[ "$(<"${dir}/${n}.status")" == 0 ]] && jq --exit-status 'type == "object"' "${dir}/${n}.out" >/dev/null 2>&1
      then
        desk=$(<"${dir}/${n}.out")
      else
        error=$(across:line "${dir}/${n}.err")
      fi
    fi
    rows+=("$(jq --compact-output --argjson n "${n}" --argjson desk "${desk}" --arg error "${error}" \
      '.members[$n] + {branch: ($desk.branch // null), ahead: ($desk.ahead // null), behind: ($desk.behind // null),
        error: (if $error == "" then null else $error end)}' <<<"${resolved}")")
  done
  printf '%s\n' "${rows[@]}" | jq --slurp --compact-output '{rows: ., errors: []}'
}

# across:repos-text <doc> draws the repos stage, a line per member: the workspace's glyph and the member; its forge;
# its branch, with ↑ the commits it is ahead and ↓ those it is behind where either is not 0; and its checkout, or not
# cloned, dim. A member whose desk could not be read says why, after ❌.
across:repos-text() {
  local repository forge branch ahead behind checkout error where
  while IFS=$'\t' read -r repository forge branch ahead behind checkout error
  do
    [[ "${ahead}" == 0 ]] || branch+=" ↑${ahead}"
    [[ "${behind}" == 0 ]] || branch+=" ↓${behind}"
    where=${checkout}
    [[ -n "${checkout}" ]] || where=$(flow:paint 90 "not cloned")
    [[ -z "${error}" ]] || where+="  ${flow_glyphs[Failed]} ${error}"
    printf '%s %s  %s  %s  %s\n' "${flow_glyphs[Workspace]}" "${repository}" "${forge:-no forge}" "${branch}" \
      "${where}" >&2
  done < <(jq --raw-output '.rows[] | [.repository, (.forge // ""), (.branch // ""), (.ahead // 0), (.behind // 0),
    (.checkout // ""), (.error // "")] | @tsv' <<<"$1")
}

# ----------------------------------------------------------------------------------------------------------------------
