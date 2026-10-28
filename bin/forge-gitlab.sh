# shellcheck shell=bash
# shellcheck disable=SC2016  # jq filters name their variables with $, in single quotes on purpose
# forge-gitlab.sh - GitLab, through glab, as the forge the loop runs on (sourced module)
# ----------------------------------------------------------------------------------------------------------------------
#
# The provider bin/forge.sh loads for a repository whose forge is gitlab: gitlab.com by origin's host, or any
# self-hosted GitLab by `forge: gitlab` in .kit/settings.yml, glab finding the host from origin as it does. Every
# call is one of the names the seam's header lists, and answers in the same shape the GitHub provider does: a
# merge request is the change, its iid the number, its pipeline's jobs the checks, its resolvable discussions the
# review threads. glab is this file's requirement and nobody else's; the REST API is read through `glab api`, and
# every document is folded here with jq, so the provider's logic is this file's and a test can drive it.

# forge-gitlab:project prints the project's path with namespace, from origin: group/subgroup/name.
forge-gitlab:project() {
  local url
  url=$(git remote get-url origin 2>/dev/null) || { error "this checkout has no origin; add one"; return 1; }
  sed -E 's#^[a-z]+://[^/]+/##; s#^[^@/]+@[^:]+:##; s#\.git$##' <<<"${url}"
}

# forge-gitlab:api <path-under-the-project> [<glab api option>…] calls the REST API under the project, its path
# encoded the way GitLab takes it in a URL.
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

# GitLab has no command that makes a branch for an issue, so the branch is made the way its issue page would make
# it, <iid>-<slug of the title>, from the default branch; GitLab links a branch named so to the issue.
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

# On a runner the pipeline names the merge request; on a desk the current branch does.
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

# Every open merge request folded into the document kit status draws, the GitHub provider's shape: number, ref,
# branch, title, author, createdAt, age, state, checks {state, passed, total, failing, took}, unresolved, closes.
# GitLab's REST API spreads these over four reads per merge request: the request, its pipeline's jobs, its
# discussions, and the issues it closes.
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
    jq --compact-output --null-input --argjson mr "${mr}" --argjson jobs "${jobs}" \
      --argjson discussions "${discussions}" --argjson closes "${closes}" '
      def pending: IN("created", "pending", "running", "preparing", "waiting_for_resource", "scheduled");
      def passed: IN("success", "skipped", "manual");
      (now | floor) as $now
      | ($jobs | length) as $n
      | ([$jobs[] | select(.status | pending)] | length) as $running
      | ([$jobs[] | select(.status | passed)] | length) as $ok
      | ([$jobs[] | select((.status | pending or passed) | not) | .name] | first // "") as $failing
      | (if $n == 0 then "NONE" elif $failing != "" then "FAILURE" elif $running > 0 then "PENDING" else "SUCCESS" end)
        as $check
      | ([$jobs[] | .started_at // empty | sub("\\.[0-9]+"; "") | fromdateiso8601] | min) as $started
      | ([$jobs[] | .finished_at // empty | sub("\\.[0-9]+"; "") | fromdateiso8601] | max) as $completed
      | (if $started == null then null
         elif $check == "PENDING" then $now - $started
         elif $completed == null then null
         else $completed - $started end) as $took
      | ([$discussions[] | .notes[]? | select(.resolvable and (.resolved | not))] | length) as $unresolved
      | (if $mr.draft then "draft"
         elif $check == "PENDING" then "queued"
         elif $mr.has_conflicts then "conflicted"
         elif $check == "FAILURE" then "failing:gate"
         elif $check == "NONE" then "no-check"
         elif $unresolved > 0 then "comments:open"
         elif ($mr.detailed_merge_status // "") == "not_approved" then "blocked:approval"
         else "ready" end) as $state
      | { number: $mr.iid, ref: "!\($mr.iid)", branch: $mr.source_branch, title: $mr.title,
          author: ($mr.author.username // ""), createdAt: $mr.created_at,
          age: ($now - ($mr.created_at | sub("\\.[0-9]+"; "") | fromdateiso8601)), state: $state,
          checks: { state: $check, passed: $ok, total: $n, failing: $failing, took: $took },
          unresolved: $unresolved,
          closes: [$closes[] | { number: .iid, title }] }'
  done < <(jq --raw-output '.[].iid' <<<"${list}")
}

# ----------------------------------------------------------------------------------------------------------------------
