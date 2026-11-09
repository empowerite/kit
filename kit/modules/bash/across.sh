# shellcheck shell=bash
# across.sh - kit flow across a workspace: its members, their checkouts, and their stages merged (sourced module)
# ----------------------------------------------------------------------------------------------------------------------
# shellcheck disable=SC2154  # flow_glyphs and out_color are set by modules/bash/flow.sh and 1st.sh
#
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

# across:flow <workspace> <word>… draws what kit flow's words ask for across the workspace: --stages, repos first;
# --marks and --quota, spanning the members; repos; a stage with --json or --jq as across:stage gives it; the picture
# as JSON, as across:picture gives it; and a stage or the picture as text, each member's under it. No workspace is the
# repository here, for repos alone. A pull request's number names one repository's, so it is refused, with 2.
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
    (--marks)
      across:marks "${workspace}"
      return
      ;;
    (--quota)
      across:quota "${workspace}"
      return
  esac
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
  if [[ -n "${form}" ]]
  then
    across:picture "${workspace}" "$@" | if [[ "${form}" == --jq ]]; then jq .; else cat; fi
    return "${PIPESTATUS[0]}"
  fi
  across:text "${workspace}" "${stage}" "$@"
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

# across:start <workspace> <dir> <word>… runs kit flow with the words, --jq as --json, in every cloned member's
# checkout at once, as across:run runs one, the n-th's in <dir>/<n>, and waits for them. It prints each cloned member's
# number and member, tab-separated, in the workspace's order; it returns 2 for a workspace across:members refuses.
across:start() {
  local workspace=$1 dir=$2 members member checkout n=0 arg
  local -a args=()
  shift 2
  for arg in "$@"
  do
    [[ "${arg}" == --jq ]] && arg=--json
    args+=("${arg}")
  done
  members=$(across:members "${workspace}") || return 2
  while IFS=$'\t' read -r member checkout
  do
    [[ -n "${checkout}" ]] || continue
    printf '%s' "${member}" > "${dir}/${n}.member"
    printf '%s' "${checkout}" > "${dir}/${n}.checkout"
    across:run "${checkout}" "${dir}/${n}" "${workspace}" "${args[@]}" &
    printf '%s\t%s\n' "${n}" "${member}"
    n=$(( n + 1 ))
  done <<<"${members}"
  wait || true
}

# across:picture <workspace> <word>… prints the picture across the workspace as one JSON object: stages, repos and then
# the stages where kit runs; repos, as across:repos gives it; and each stage's document, {rows, errors}, cut from the
# picture each member's kit flow --json printed. A stage a member's flow setting leaves out adds nothing; one it could
# not draw, null in its picture, is its last line under errors.
across:picture() {
  local workspace=$1 dir stages repos stage kind
  local -a named=()
  shift
  file:scratch dir
  stages=$(across:flow "${workspace}" --stages) || return $?
  across:start "${workspace}" "${dir}" "$@" >/dev/null || return $?
  repos=$(across:repos "${workspace}") || return $?
  while IFS= read -r stage
  do
    [[ "${stage}" != repos ]] || continue
    kind=array
    case "${stage}" in
      (desk | worktrees | incoming) kind=object
    esac
    named+=(--argjson "${stage}" "$(across:cut "${dir}" "${stage}" "${kind}")")
  done <<<"${stages}"
  jq --null-input --compact-output --argjson stages "$(workspaces:lines-json <<<"${stages}")" \
    --argjson repos "${repos}" "${named[@]}" '{stages: $stages, repos: $repos} + ($ARGS.named | del(.repos))'
}

# across:cut <dir> <stage> <kind> prints a stage's document from the pictures the members printed in <dir>, as
# across:picture says.
across:cut() {
  local dir=$1 stage=$2 kind=$3 n=0 member value
  local -a rows=() errors=()
  while [[ -f "${dir}/${n}.member" ]]
  do
    member=$(<"${dir}/${n}.member")
    if value=$(jq --compact-output --arg stage "${stage}" 'if has($stage) then .[$stage] else "absent" end' \
      "${dir}/${n}.out" 2>/dev/null) && [[ -n "${value}" && "${value}" != null ]]
    then
      [[ "${value}" == '"absent"' ]] || rows+=("$(jq --compact-output --arg member "${member}" --arg kind "${kind}" \
        'if $kind == "array" then map(. + {member: $member})[] else . + {member: $member} end' <<<"${value}")")
    else
      errors+=("$(jq --null-input --compact-output --arg member "${member}" \
        --arg error "$(across:line "${dir}/${n}.err")" '{member: $member, error: $error}')")
    fi
    n=$(( n + 1 ))
  done
  jq --null-input --compact-output \
    --argjson rows "$(printf '%s\n' "${rows[@]}" | grep -v '^$' | jq --slurp --compact-output .)" \
    --argjson errors "$(printf '%s\n' "${errors[@]}" | jq --slurp --compact-output .)" '{rows: $rows, errors: $errors}'
}

# across:text <workspace> <stage> <word>… draws a stage, or with none the picture, as text across the workspace: for the
# picture, repos first; then each cloned member's, as kit flow draws it in that checkout, under its member, as
# flow:above draws each desk's under its path. With none cloned, one dim line says so. It returns 1 when a member's
# kit flow did.
across:text() {
  local workspace=$1 stage=$2 dir n member line failed=false first=true color=''
  local -a started=()
  shift 2
  file:scratch dir
  [[ -z "${stage}" ]] || set -- "${stage}" "$@"
  if [[ -z "${stage}" ]]
  then
    across:repos-form "${workspace}" '' || return $?
    first=false
  fi
  [[ "${out_color}" == 1 ]] && color=1
  readarray -t started < <(COLUMNS=$(flow:width) CLICOLOR_FORCE=${color} across:start "${workspace}" "${dir}" "$@")
  if (( ${#started[@]} == 0 ))
  then
    [[ "${first}" == true ]] || printf '\n' >&2
    printf '%s\n' "$(flow:paint 90 "none of ${workspace}'s members is cloned under the roots; run: kit workspaces \
--show ${workspace}")" >&2
    return 0
  fi
  for line in "${started[@]}"
  do
    IFS=$'\t' read -r n member <<<"${line}"
    [[ "${first}" == true ]] || printf '\n' >&2
    first=false
    printf '%s %s\n' "${flow_glyphs[Workspace]}" "${member}" >&2
    cat "${dir}/${n}.err" "${dir}/${n}.out" >&2
    [[ "$(<"${dir}/${n}.status")" == 0 ]] || failed=true
  done
  [[ "${failed}" == false ]]
}

# across:marks <workspace> prints a mark per stage across the workspace, as one JSON object: each the members' marks
# for the stage, in their order, so it changes when any member's does; and repos', each member's checkout and branch.
across:marks() {
  local workspace=$1 dir members member checkout branch
  file:scratch dir
  across:start "${workspace}" "${dir}" --marks >/dev/null || return $?
  members=$(across:members "${workspace}") || return 2
  while IFS=$'\t' read -r member checkout
  do
    branch=''
    [[ -z "${checkout}" ]] || branch=$(git -C "${checkout}" rev-parse --abbrev-ref HEAD 2>/dev/null || true)
    printf '%s\t%s\t%s\n' "${member}" "${checkout}" "${branch}"
  done <<<"${members}" > "${dir}/repos.mark"
  across:marks-merge "${dir}" "$(git hash-object --stdin < "${dir}/repos.mark")"
}

# across:marks-merge <dir> <repos> joins the marks each member printed in <dir>, in the members' order, under each
# stage any of them names, and adds repos' mark.
across:marks-merge() {
  local dir=$1 n=0
  local -a marks=()
  while [[ -f "${dir}/${n}.member" ]]
  do
    marks+=("$(jq --compact-output 'if type == "object" then . else {} end' "${dir}/${n}.out" 2>/dev/null || echo '{}')")
    n=$(( n + 1 ))
  done
  printf '%s\n' "${marks[@]}" | grep -v '^$' | jq --slurp --compact-output --arg repos "$2" '
    . as $all | ([$all[] | keys[]] | unique) as $stages
    | reduce $stages[] as $s ({}; .[$s] = ([$all[] | .[$s] // "null" | tostring] | join(" ")))
    | . + {repos: $repos}'
}

# across:quota <workspace> prints every forge the workspace's members touch, each once, as kit flow --quota prints a
# repository's.
across:quota() {
  local dir
  file:scratch dir
  across:start "$1" "${dir}" --quota >/dev/null || return $?
  across:quota-merge "${dir}"
}

# across:quota-merge <dir> joins the arrays kit flow --quota printed in <dir>, each forge once, in the order first met.
across:quota-merge() {
  local dir=$1 out
  for out in "${dir}"/*.out
  do
    [[ -e "${out}" && "$(<"${out%.out}.status")" == 0 ]] || continue
    jq --compact-output 'if type == "array" then . else [] end' "${out}" 2>/dev/null || true
  done | jq --slurp --compact-output 'add // [] | reduce .[] as $q ([]; if any(.[]; .forge == $q.forge) then . else
    . + [$q] end)'
}

# ----------------------------------------------------------------------------------------------------------------------
