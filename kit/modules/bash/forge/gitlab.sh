# shellcheck shell=bash
# gitlab.sh - the GitLab forge provider, through the glab CLI (sourced module)
# ----------------------------------------------------------------------------------------------------------------------
# shellcheck disable=SC2154  # kit_jq is exported by modules/bash/1st.sh, forge_list_limit set by modules/bash/forge.sh
# shellcheck disable=SC2016  # jq filters name their variables with $, in single quotes on purpose
#
# modules/bash/forge.sh loads this provider when the repository's forge is gitlab:
#
#   - gitlab.com, detected from the primary remote's host
#   - a self-hosted GitLab: a host glab has logged in to, or `forge: gitlab` in kit.yml; glab finds the host from
#     the remote
#
# It implements the calls listed in the header of modules/bash/forge.sh, with the same output as the GitHub
# provider. The GitLab equivalents:
#
#   pull request    a merge request
#   number          the merge request's iid
#   checks          the jobs of its pipeline
#   review threads  its resolvable discussions
#
# Only this file requires glab. It reads the REST API through `glab api`, a list's every page, and shapes every result
# with jq, so a fake glab can drive it in a test.
#
# see: modules/bash/forge.sh

forge/gitlab:branch-for-issue() {
  local open
  forge/gitlab:pages open 'the open merge requests' 'merge_requests?state=opened&per_page=100' || return 1
  forge/gitlab:fold 'the open merge requests' "${open}" --raw-output --arg n "$1" \
    'import "forge" as forge; forge::gitlab_branch_for_issue($n)'
}

# forge/gitlab:issue-add-sub links the second issue to the first as a related item: GitLab gives an issue no
# sub-issues.
forge/gitlab:issue-add-sub() {
  local project
  project=$(forge/gitlab:project) || return 1
  forge/gitlab:api "issues/$1/links" --method POST --raw-field "target_project_id=${project}" \
    --raw-field "target_issue_iid=$2" >/dev/null 2>&1
}

forge/gitlab:issue-assign() {
  glab issue update "$1" --assignee "+$2" >/dev/null 2>&1
}

forge/gitlab:issue-assignees() {
  local issue
  forge/gitlab:ask issue "the assignees of #$1" "issues/$1" || return 1
  forge/gitlab:fold "the assignees of #$1" "${issue}" --raw-output '[.assignees[].username] | join(" ")'
}

# forge/gitlab:issue-close closes the issue. GitLab keeps no reason, so completed and not planned close it alike; a
# duplicate is closed by the /duplicate quick action, which links it to the original.
forge/gitlab:issue-close() {
  if [[ "$2" == duplicate ]]
  then
    glab issue note "$1" --message "/duplicate #$3" >/dev/null 2>&1
  else
    glab issue close "$1" >/dev/null 2>&1
  fi
}

forge/gitlab:issue-comment() {
  glab issue note "$1" --message "$2" >/dev/null 2>&1
}

forge/gitlab:issue-create() {
  local number
  number=$(glab issue create --title "$1" --description "${2:-}" --yes 2>/dev/null \
    | sed -n 's@.*/-/issues/\([0-9]*\).*@\1@p' | tail -n 1)
  if [[ ! "${number}" =~ ^[0-9]+$ ]]
  then
    error "run: glab auth status (the issue could not be created)"
    return 1
  fi
  printf '%s\n' "${number}"
}

# forge/gitlab:issue-state says GitLab's opened and closed as OPEN and CLOSED; an answer with neither fails.
forge/gitlab:issue-state() {
  local issue
  forge/gitlab:ask issue "the state of #$1" "issues/$1" || return 1
  forge/gitlab:fold "the state of #$1" "${issue}" --raw-output '{opened: "OPEN", closed: "CLOSED"}[.state] // error'
}

forge/gitlab:issue-title() {
  local issue
  forge/gitlab:ask issue "the title of #$1" "issues/$1" || return 1
  forge/gitlab:fold "the title of #$1" "${issue}" --raw-output '.title | strings // error'
}

forge/gitlab:issue-unassign() {
  glab issue update "$1" --assignee "-$2" >/dev/null 2>&1
}

# GitLab has no command that creates a branch for an issue. This creates the branch the issue page would:
# <iid>-<slug of the title>, from the default branch. GitLab links a branch with that name to the issue.
forge/gitlab:link-branch() {
  local title slug
  title=$(forge/gitlab:issue-title "$1") || return 1
  slug=$(tr '[:upper:]' '[:lower:]' <<<"${title}" | sed -E 's/[^a-z0-9]+/-/g; s/^-+//; s/-+$//' | cut -c 1-60)
  slug=${slug%-}
  forge/gitlab:api repository/branches --method POST --raw-field "branch=$1-${slug}" --raw-field "ref=$2" >/dev/null
}

# forge/gitlab:marks [<dir>] reads, for each kind of the project's data, a mark that changes when it does: issues, of
# the issue and the merge request updated last; runs, of the ten newest pipelines; releases, of the newest release.
# GitLab sends no ETag for these, so each mark is what the answer says of when its items changed, and costs a request
# each read, four in all.
forge/gitlab:marks() {
  local issues mrs runs releases
  forge/gitlab:ask issues 'the marks' 'issues?order_by=updated_at&sort=desc&per_page=1' || return 1
  forge/gitlab:ask mrs 'the marks' 'merge_requests?order_by=updated_at&sort=desc&per_page=1' || return 1
  forge/gitlab:ask runs 'the marks' 'pipelines?per_page=10' || return 1
  forge/gitlab:ask releases 'the marks' 'releases?per_page=1' || return 1
  forge/gitlab:fold 'the marks' '' --null-input --compact-output --argjson issues "${issues}" --argjson mrs "${mrs}" \
    --argjson runs "${runs}" --argjson releases "${releases}" \
    'import "forge" as forge; forge::gitlab_marks($issues; $mrs; $runs; $releases)'
}

forge/gitlab:me() {
  local user
  user=$(glab api user 2>/dev/null) && jq --exit-status --raw-output '.username | strings' <<<"${user}" 2>/dev/null \
    && return 0
  error "run: glab auth login (the caller's login could not be read)"
  return 1
}

forge/gitlab:noun() { printf '!\tmerge request\n'; }

# forge/gitlab:open-issues prints one document per open issue, oldest first, as the seam's header says. GitLab gives
# each label's color only with its details.
forge/gitlab:open-issues() {
  local list
  forge/gitlab:pages list 'the issues' \
    'issues?state=opened&per_page=100&order_by=created_at&sort=asc&with_labels_details=true' || return 1
  forge/gitlab:fold 'the issues' "${list}" --compact-output --argjson now "$(date +%s)" \
    'import "forge" as forge; forge::gitlab_issues($now)'
}

# forge/gitlab:open-prs prints one JSON object per open merge request: the document kit flow renders, which
# modules/jq/forge.jq folds the answers into. Of several jobs of one name in the pipeline, as after a retry, only
# the latest counts. GitLab's REST API needs one read for the project's labels, whose colors the merge requests and
# the issues they close name only by name; then four reads per merge request, and one per fork:
#
#   - the merge request, whose head pipeline also says when the branch was last pushed, and by whom
#   - its pipeline's jobs, read under the project the pipeline names, a fork's for a merge request from one; a fork's
#     jobs that cannot be read are one check, pipeline, of the pipeline's own status
#   - its discussions
#   - the issues it closes
#   - the project its branch lives in, when that is not this project, once for each such project: its namespace's
#     path names the fork's owner, and a project the caller cannot read names none
#
# Any other read that fails fails the call, which then prints none of the merge requests.
forge/gitlab:open-prs() {
  local list labels colors iid mr jobs discussions closes pipeline home target source fork doc docs=''
  local -a iids
  local -A owners=()
  forge/gitlab:pages list 'the merge requests' 'merge_requests?state=opened&per_page=100&order_by=created_at&sort=asc' \
    || return 1
  # A label whose color cannot be read is drawn without it, so the labels' read never fails the merge requests'.
  if ! forge/gitlab:pages labels 'the labels' 'labels?per_page=100' 2>/dev/null \
    || ! colors=$(jq --compact-output 'map({key: .name, value: .color}) | from_entries' <<<"${labels}" 2>/dev/null)
  then
    colors='{}'
  fi
  list:read iids forge/gitlab:fold 'the merge requests' "${list}" --raw-output '.[].iid' || return 1
  for iid in "${iids[@]}"
  do
    [[ -n "${iid}" ]] || continue
    forge/gitlab:ask mr "merge request !${iid}" "merge_requests/${iid}" || return 1
    pipeline=$(forge/gitlab:fold "merge request !${iid}" "${mr}" --raw-output '.head_pipeline.id // empty') || return 1
    home=$(forge/gitlab:fold "merge request !${iid}" "${mr}" --raw-output '.head_pipeline.project_id // empty') \
      || return 1
    target=$(forge/gitlab:fold "merge request !${iid}" "${mr}" --raw-output '.target_project_id // empty') || return 1
    jobs='[]'
    if [[ -n "${pipeline}" && -n "${home}" && "${home}" != "${target}" ]]
    then
      # A fork's pipeline lives in the fork's project, which the caller may not be able to read; its jobs are then
      # the pipeline's own status, so that read's failure is said by the check it gives, not on stderr.
      if ! forge/gitlab:pages jobs "the jobs of pipeline ${pipeline}" \
        "/projects/${home}/pipelines/${pipeline}/jobs?per_page=100" 2>/dev/null
      then
        jobs=$(forge/gitlab:fold "merge request !${iid}" "${mr}" --compact-output \
          'import "forge" as forge; forge::gitlab_pipeline_jobs') || return 1
      fi
    elif [[ -n "${pipeline}" ]]
    then
      forge/gitlab:pages jobs "the jobs of pipeline ${pipeline}" \
        "${home:+/projects/${home}/}pipelines/${pipeline}/jobs?per_page=100" || return 1
    fi
    forge/gitlab:pages discussions "the discussions of merge request !${iid}" \
      "merge_requests/${iid}/discussions?per_page=100" || return 1
    forge/gitlab:pages closes "the issues merge request !${iid} closes" \
      "merge_requests/${iid}/closes_issues?per_page=100" || return 1
    source=$(forge/gitlab:fold "merge request !${iid}" "${mr}" --raw-output \
      'select(.source_project_id != .target_project_id) | .source_project_id // empty') || return 1
    # The fork's project is not under this one, so glab reads it by its id, past forge/gitlab:api. A fork the caller
    # cannot read names no owner, as the list above says.
    if [[ -n "${source}" && -z "${owners[${source}]+x}" ]]
    then
      owners[${source}]=$(glab api "projects/${source}" 2>/dev/null \
        | jq --compact-output '.namespace.full_path // null' 2>/dev/null) || owners[${source}]=null
    fi
    fork=null
    [[ -n "${source}" ]] && fork=${owners[${source}]:-null}
    doc=$(forge/gitlab:fold "merge request !${iid}" "${mr}" --compact-output --argjson now "$(date +%s)" \
      --argjson jobs "${jobs}" --argjson discussions "${discussions}" --argjson closes "${closes}" \
      --argjson fork "${fork}" --argjson colors "${colors}" \
      'import "forge" as forge; forge::gitlab_pr($now; $jobs; $discussions; $closes; $fork)
       | forge::colored($colors)') || return 1
    docs+=${doc}$'\n'
  done
  printf '%s' "${docs}"
}

forge/gitlab:pr-author() {
  local mr
  forge/gitlab:ask mr "the author of merge request !$1" "merge_requests/$1" || return 1
  forge/gitlab:fold "the author of merge request !$1" "${mr}" --raw-output '.author.username | strings // error'
}

forge/gitlab:pr-body() {
  local mr body
  forge/gitlab:ask mr "the body of merge request !$1" "merge_requests/$1" || return 1
  body=$(forge/gitlab:fold "the body of merge request !$1" "${mr}" --raw-output \
    '.description // "" | strings // error') || return 1
  printf '%s\n' "${body}"
}

forge/gitlab:pr-closing-issues() {
  local closes
  forge/gitlab:pages closes "merge request !$1" "merge_requests/$1/closes_issues?per_page=100" || return 1
  forge/gitlab:fold "merge request !$1" "${closes}" --raw-output '.[] | "\(.iid) \(if .state == "closed" then "CLOSED"
    else "OPEN" end) \(.references.full | sub("#[0-9]+$"; ""))"'
}

forge/gitlab:pr-create() {
  local number
  number=$(glab mr create --draft --source-branch "$1" --target-branch "$2" --title "$3" --description "" --yes \
    | sed -n 's@.*/-/merge_requests/\([0-9]*\).*@\1@p' | tail -n 1) || return 1
  [[ "${number}" =~ ^[0-9]+$ ]] || { error "the merge request could not be opened; see above"; return 1; }
  printf '%s\n' "${number}"
}

forge/gitlab:pr-for-branch() {
  forge/gitlab:on-branch "the open merge request on $1" "merge_requests?state=opened&source_branch=$1" \
    '.[0].iid // empty'
}

forge/gitlab:pr-head-for-branch() {
  forge/gitlab:on-branch "the merge requests on $1" "$(forge/gitlab:latest-on "$1")" '.[0].sha // empty'
}

# On a GitLab CI runner, CI_MERGE_REQUEST_IID gives the merge request. Otherwise it is the open merge request for
# the current branch; a detached HEAD has none.
forge/gitlab:pr-here() {
  local number=${CI_MERGE_REQUEST_IID:-} branch
  if [[ -z "${number}" ]]
  then
    branch=$(git symbolic-ref --quiet --short HEAD 2>/dev/null || true)  # a detached HEAD has no merge request
    [[ -z "${branch}" ]] || number=$(forge/gitlab:pr-for-branch "${branch}") || return 1
  fi
  [[ -z "${number}" ]] || printf '%s\n' "${number}"
}

forge/gitlab:pr-last-for-branch() {
  forge/gitlab:on-branch "the merge requests on $1" "$(forge/gitlab:latest-on "$1")" '.[0].iid // empty'
}

forge/gitlab:pr-set-body() {
  forge/gitlab:api "merge_requests/$1" --method PUT --raw-field "description=$2" >/dev/null
}

forge/gitlab:pr-state() {
  local mr
  forge/gitlab:ask mr "merge request !$1" "merge_requests/$1" || return 1
  forge/gitlab:fold "merge request !$1" "${mr}" --raw-output '"\(.state | strings // error
    | if . == "merged" then "MERGED" elif . == "closed" then "CLOSED" else "OPEN" end
    )\t\(.merge_commit_sha // .squash_commit_sha // "")"'
}

forge/gitlab:pr-state-for-branch() {
  forge/gitlab:on-branch "the merge requests on $1" "$(forge/gitlab:latest-on "$1")" \
    'import "forge" as forge; forge::gitlab_pr_state'
}

forge/gitlab:pr-title() {
  local mr
  forge/gitlab:ask mr "the title of merge request !$1" "merge_requests/$1" || return 1
  forge/gitlab:fold "the title of merge request !$1" "${mr}" --raw-output '.title | strings // error'
}

forge/gitlab:pr-url() {
  local url
  url=$(forge/gitlab:repo-url) || return 1
  printf '%s/-/merge_requests/%s\n' "${url}" "$1"
}

# forge/gitlab:quota reads the caller's user, one request, for the RateLimit headers GitLab sends with every answer:
# the limit that throttles the caller, by name, what is left of it, and when it resets. A GitLab that sends none has
# no limit to show.
forge/gitlab:quota() {
  local answer
  answer=$(glab api --include user 2>/dev/null) \
    || { error "run: glab auth status (the rate limits could not be read)"; return 1; }
  forge/gitlab:fold 'the rate limits' "${answer}" --raw-input --slurp --compact-output \
    'import "forge" as forge; forge::gitlab_quota'
}

# forge/gitlab:releases prints one document per release, newest first, as the seam's header says.
forge/gitlab:releases() {
  local list
  forge/gitlab:pages list 'the releases' 'releases?per_page=100' || return 1
  forge/gitlab:fold 'the releases' "${list}" --compact-output --argjson now "$(date +%s)" \
    'import "forge" as forge; forge::gitlab_releases($now)'
}

forge/gitlab:repo() { forge/gitlab:project; }

forge/gitlab:repo-url() {
  local host primary project
  primary=$(forge:primary) || return $?
  host=$(forge:host "${primary}")
  project=$(forge/gitlab:project) || return 1
  printf 'https://%s/%s\n' "${host}" "${project}"
}

forge/gitlab:require() {
  guard:require tool:glab || return 1
  forge/gitlab:me >/dev/null
}

# forge/gitlab:runs prints the builds stage's runs as one JSON array, as the seam's header says: the 50 newest
# pipelines, those forge.jq's builds keeps. A pipeline's jobs would cost a call each, so every run carries none.
forge/gitlab:runs() {
  local list default tags
  forge/gitlab:ask list 'the pipelines' 'pipelines?per_page=50' || return 1
  default=$(git:default-branch) || return 1
  tags=$(forge:tags) || return 1
  forge/gitlab:fold 'the pipelines' "${list}" --compact-output --argjson now "$(date +%s)" --arg default "${default}" \
    --argjson tags "${tags}" 'import "forge" as forge; forge::gitlab_builds($now; $default; $tags)'
}

# The provider implements contract 2 and its forge lacks none of its features.
forge/gitlab:contract() { printf '2\n'; }

# The seam's calls end here; what follows is this provider's own.

# forge/gitlab:project prints the project's full path, read from the primary remote: group/subgroup/name.
forge/gitlab:project() {
  local url primary
  primary=$(forge:primary) || return $?
  if ! url=$(git remote get-url --end-of-options "${primary}" 2>/dev/null)
  then
    error "run: git remote add ${primary} <url>, as this checkout has no ${primary} (the project could not be read)"
    return 1
  fi
  sed -E 's#^[a-z]+://[^/]+/##; s#^[^@/]+@[^:]+:##; s#\.git$##' <<<"${url}"
}

# forge/gitlab:api <path> [<glab api option>…] calls the REST API at <path> under the project, with the project's
# path URL-encoded as GitLab expects; a <path> that starts with / is read from the API's root, as a fork's project is.
forge/gitlab:api() {
  local path=$1 project
  shift
  if [[ "${path}" == /* ]]
  then
    glab api "${path#/}" "$@"
    return
  fi
  project=$(forge/gitlab:project) || return 1
  glab api "projects/${project//\//%2F}/${path}" "$@"
}

# forge/gitlab:ask <name> <what> <path> [<glab api option>…] reads <path> as forge/gitlab:api does and puts the answer
# in the variable <name>. When glab fails, or answers with anything but JSON, it names glab auth status for <what>, and
# returns 1.
forge/gitlab:ask() {
  local ask_into=$1 ask_what=$2 ask_path=$3 ask_out
  shift 3
  # The project is read first, so a checkout with no remote says so, not glab.
  [[ "${ask_path}" == /* ]] || forge/gitlab:project >/dev/null || return 1
  if ! ask_out=$(forge/gitlab:api "${ask_path}" "$@" 2>/dev/null) \
    || ! forge:json "${ask_out}"
  then
    error "run: glab auth status (${ask_what} could not be read)"
    return 1
  fi
  printf -v "${ask_into}" '%s' "${ask_out}"
}

# forge/gitlab:fold <what> <answer> <jq argument>… is forge:fold, naming glab auth status.
forge/gitlab:fold() { forge:fold 'run: glab auth status' "$@"; }

# forge/gitlab:latest-on <branch> prints the path of the merge requests on the branch, in any state, newest first.
forge/gitlab:latest-on() {
  printf 'merge_requests?state=all&source_branch=%s&order_by=created_at&sort=desc\n' "$1"
}

# forge/gitlab:on-branch <what> <path> <jq filter> reads the merge requests at <path> and prints what the filter makes
# of them, or nothing when it makes nothing of them. An answer that is not a list fails, so it never reads as none.
forge/gitlab:on-branch() {
  local mrs
  forge/gitlab:ask mrs "$1" "$2" || return 1
  forge/gitlab:fold "$1" "${mrs}" --exit-status 'type == "array"' >/dev/null || return 1
  forge/gitlab:fold "$1" "${mrs}" --raw-output "$3"
}

# forge/gitlab:pages <name> <what> <path> reads every page of the list at <path>, as glab api --paginate asks for
# them, and puts the list, one JSON array, in the variable <name>. glab prints each page's array after the one before,
# or one array of every page's items; either is joined into one. A list that is not whole fails, as forge:whole says.
forge/gitlab:pages() {
  local pages_into=$1 pages_what=$2 pages_answer pages_joined
  forge/gitlab:ask pages_answer "${pages_what}" "$3" --paginate || return 1
  pages_joined=$(forge/gitlab:fold "${pages_what}" "${pages_answer}" --slurp --compact-output \
    'if all(.[]; type == "array") then add // [] else error end') || return 1
  forge:whole "${pages_what}" "$(jq length <<<"${pages_joined}")" || return 1
  printf -v "${pages_into}" '%s' "${pages_joined}"
}

# ----------------------------------------------------------------------------------------------------------------------
