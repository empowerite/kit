# shellcheck shell=bash
# forge-gitlab.sh - the GitLab forge provider, through the glab CLI (sourced module)
# ----------------------------------------------------------------------------------------------------------------------
# shellcheck disable=SC2016  # jq filters name their variables with $, in single quotes on purpose
#
# modules/bash/forge.sh loads this provider when the repository's forge is gitlab:
#
#   - gitlab.com, detected from origin's host
#   - a self-hosted GitLab, set with `forge: gitlab` in .kit/settings.yml; glab finds the host from origin
#
# It implements the calls listed in the header of modules/bash/forge.sh, with the same output as the GitHub provider. The
# GitLab equivalents:
#
#   pull request    a merge request
#   number          the merge request's iid
#   checks          the jobs of its pipeline
#   review threads  its resolvable discussions
#
# Only this file requires glab. It reads the REST API through `glab api` and shapes every result with jq, so a
# fake glab can drive it in a test.
#
# see: modules/bash/forge.sh

# forge-gitlab:project prints the project's full path, read from origin: group/subgroup/name.
forge-gitlab:project() {
  local url
  url=$(git remote get-url origin 2>/dev/null) || { error "this checkout has no origin; add one"; return 1; }
  sed -E 's#^[a-z]+://[^/]+/##; s#^[^@/]+@[^:]+:##; s#\.git$##' <<<"${url}"
}

# forge-gitlab:api <path-under-the-project> [<glab api option>…] calls the REST API under the project, with the
# project's path URL-encoded as GitLab expects.
forge-gitlab:api() {
  local path=$1 project
  shift
  project=$(forge-gitlab:project) || return 1
  glab api "projects/${project//\//%2F}/${path}" "$@"
}

forge-gitlab:require() {
  guard:require tool:glab || return 1
  if ! glab api user >/dev/null 2>&1
  then
    error "run: glab auth login (the GitLab CLI is not authenticated)"
    return 1
  fi
}

forge-gitlab:repo() { forge-gitlab:project; }

forge-gitlab:repo-url() {
  local host
  host=$(git remote get-url origin 2>/dev/null | sed -E 's#^([a-z]+://)?([^@/]+@)?([^/:]+).*#\3#')
  printf 'https://%s/%s\n' "${host}" "$(forge-gitlab:project)"
}

forge-gitlab:noun() { printf '!\tmerge request\n'; }

forge-gitlab:pr-url() { printf '%s/-/merge_requests/%s\n' "$(forge-gitlab:repo-url)" "$1"; }

forge-gitlab:me() {
  glab api user 2>/dev/null | jq --raw-output '.username' && return 0
  error "run: glab auth login (the GitLab CLI is not authenticated)"
  return 1
}

forge-gitlab:issue-create() {
  local number
  number=$(glab issue create --title "$1" --description "" --yes 2>/dev/null \
    | sed -n 's@.*/-/issues/\([0-9]*\).*@\1@p' | tail -n 1)
  if [[ ! "${number}" =~ ^[0-9]+$ ]]
  then
    error "run: glab auth status (the issue could not be created)"
    return 1
  fi
  printf '%s\n' "${number}"
}

forge-gitlab:issue-title() {
  forge-gitlab:api "issues/$1" 2>/dev/null | jq --raw-output '.title' \
    || { error "run: glab auth status (the title of #$1 could not be read)"; return 1; }
}

forge-gitlab:issue-state() {
  forge-gitlab:api "issues/$1" 2>/dev/null | jq --raw-output 'if .state == "closed" then "CLOSED" else "OPEN" end'
}

forge-gitlab:issue-assignees() {
  forge-gitlab:api "issues/$1" 2>/dev/null | jq --raw-output '[.assignees[]?.username] | join(" ")'
}

forge-gitlab:issue-assign() {
  glab issue update "$1" --assignee "+$2" >/dev/null 2>&1
}

forge-gitlab:branch-for-issue() {
  forge-gitlab:api "merge_requests?state=opened&per_page=100" 2>/dev/null \
    | jq --raw-output --arg n "$1" \
      '[.[] | select(.source_branch | startswith($n + "-"))] | .[0].source_branch // empty' \
    || true
}

# GitLab has no command that creates a branch for an issue. This creates the branch the issue page would:
# <iid>-<slug of the title>, from the default branch. GitLab links a branch with that name to the issue.
forge-gitlab:link-branch() {
  local title slug
  title=$(forge-gitlab:issue-title "$1") || return 1
  slug=$(tr '[:upper:]' '[:lower:]' <<<"${title}" | sed -E 's/[^a-z0-9]+/-/g; s/^-+//; s/-+$//' | cut -c 1-60)
  slug=${slug%-}
  forge-gitlab:api repository/branches --method POST --raw-field "branch=$1-${slug}" --raw-field "ref=$2" >/dev/null
}

forge-gitlab:pr-for-branch() {
  forge-gitlab:api "merge_requests?state=opened&source_branch=$1" 2>/dev/null \
    | jq --raw-output '.[0].iid // empty' || true
}

forge-gitlab:pr-create() {
  local number
  number=$(glab mr create --draft --source-branch "$1" --target-branch "$2" --title "$3" --description "" --yes \
    | sed -n 's@.*/-/merge_requests/\([0-9]*\).*@\1@p' | tail -n 1) || return 1
  [[ "${number}" =~ ^[0-9]+$ ]] || { error "the merge request could not be opened; see above"; return 1; }
  printf '%s\n' "${number}"
}

forge-gitlab:pr-title() {
  forge-gitlab:api "merge_requests/$1" | jq --raw-output '.title'
}

forge-gitlab:pr-body() {
  forge-gitlab:api "merge_requests/$1" | jq --raw-output '.description // ""'
}

forge-gitlab:pr-set-body() {
  forge-gitlab:api "merge_requests/$1" --method PUT --raw-field "description=$2" >/dev/null
}

forge-gitlab:pr-state-for-branch() {
  forge-gitlab:api "merge_requests?state=all&source_branch=$1&order_by=created_at&sort=desc" 2>/dev/null \
    | jq --raw-output '.[0].state // empty | {merged: "MERGED", closed: "CLOSED", locked: "CLOSED"}[.] // "OPEN"' \
    || true
}

# On a GitLab CI runner, CI_MERGE_REQUEST_IID gives the merge request. Otherwise it is the open merge request for
# the current branch.
forge-gitlab:pr-here() {
  local number=${CI_MERGE_REQUEST_IID:-} branch
  if [[ -z "${number}" ]]
  then
    branch=$(git symbolic-ref --quiet --short HEAD 2>/dev/null || true)
    [[ -n "${branch}" ]] && number=$(forge-gitlab:pr-for-branch "${branch}")
  fi
  printf '%s\n' "${number}"
}

forge-gitlab:pr-closing-issues() {
  forge-gitlab:api "merge_requests/$1/closes_issues" 2>/dev/null \
    | jq --raw-output '.[] | "\(.iid) \(if .state == "closed" then "CLOSED" else "OPEN" end) \(.references.full
        | sub("#[0-9]+$"; ""))"' \
    || { error "run: glab auth status (merge request !$1 could not be read)"; return 1; }
}

# forge-gitlab:open-prs prints one JSON object per open merge request: the document kit status renders, which
# modules/jq/forge.jq folds the answers into. Of several jobs of one name in the pipeline, as after a retry, only
# the latest counts. GitLab's REST API needs four reads per merge request:
#
#   - the merge request
#   - its pipeline's jobs
#   - its discussions
#   - the issues it closes
forge-gitlab:open-prs() {
  local list iid mr jobs discussions closes pipeline
  list=$(forge-gitlab:api "merge_requests?state=opened&per_page=100&order_by=created_at&sort=asc" 2>/dev/null) \
    || { error "run: glab auth status (the merge requests could not be read)"; return 1; }
  while IFS= read -r iid
  do
    [[ -n "${iid}" ]] || continue
    mr=$(forge-gitlab:api "merge_requests/${iid}") || return 1
    pipeline=$(jq --raw-output '.head_pipeline.id // empty' <<<"${mr}")
    jobs='[]'
    [[ -n "${pipeline}" ]] && jobs=$(forge-gitlab:api "pipelines/${pipeline}/jobs?per_page=100")
    discussions=$(forge-gitlab:api "merge_requests/${iid}/discussions?per_page=100")
    closes=$(forge-gitlab:api "merge_requests/${iid}/closes_issues")
    # shellcheck disable=SC2154  # kit_jq is exported by modules/bash/1st.sh
    jq -L "${kit_jq}" --compact-output --argjson now "$(date +%s)" --argjson jobs "${jobs}" \
      --argjson discussions "${discussions}" --argjson closes "${closes}" \
      'import "forge" as forge; forge::gitlab_pr($now; $jobs; $discussions; $closes)' <<<"${mr}"
  done < <(jq --raw-output '.[].iid' <<<"${list}")
}

# forge-gitlab:open-issues prints one document per open issue, oldest first, as the seam's header says.
forge-gitlab:open-issues() {
  local list
  list=$(forge-gitlab:api "issues?state=opened&per_page=100&order_by=created_at&sort=asc" 2>/dev/null) \
    || { error "run: glab auth status (the issues could not be read)"; return 1; }
  # shellcheck disable=SC2154  # kit_jq is exported by modules/bash/1st.sh
  jq -L "${kit_jq}" --compact-output --argjson now "$(date +%s)" 'import "forge" as forge; forge::gitlab_issues($now)' \
    <<<"${list}"
}

# ----------------------------------------------------------------------------------------------------------------------
