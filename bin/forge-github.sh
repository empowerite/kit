# shellcheck shell=bash
# shellcheck disable=SC2016  # jq filters and GraphQL queries name their variables with $, in single quotes on purpose
# forge-github.sh - GitHub, through gh, as the forge the loop runs on (sourced module)
# ----------------------------------------------------------------------------------------------------------------------
#
# The provider bin/forge.sh loads for a repository whose forge is github. Every call is one of the names the
# seam's header lists; gh is this file's requirement and nobody else's.

# forge-github:require refuses where gh is absent or not authenticated. A revoked token passes `gh auth status`,
# so the probe is a real call.
forge-github:require() {
  guard:require tool:gh || return 1
  if ! gh api user --jq '.login' >/dev/null 2>&1
  then
    error "run: gh auth login (the GitHub CLI is not authenticated)"
    return 1
  fi
}

forge-github:repo() {
  gh repo view --json nameWithOwner --jq .nameWithOwner 2>/dev/null \
    || { error "run: gh repo view (this checkout does not resolve to a GitHub repository)"; return 1; }
}

forge-github:repo-url() {
  gh repo view --json url --jq .url 2>/dev/null \
    || { error "run: gh repo view (this checkout does not resolve to a GitHub repository)"; return 1; }
}

forge-github:noun() { printf '#\tpull request\n'; }

forge-github:pr-url() { printf '%s/pull/%s\n' "$(forge-github:repo-url)" "$1"; }

forge-github:me() {
  gh api user --jq '.login' 2>/dev/null && return 0
  error "run: gh auth login (the GitHub CLI is not authenticated)"
  return 1
}

forge-github:issue-create() {
  local number
  number=$(gh issue create --title "$1" --body "" 2>/dev/null | sed -n 's@.*/issues/@@p')
  if [[ ! "${number}" =~ ^[0-9]+$ ]]
  then
    error "run: gh auth status (the issue could not be created)"
    return 1
  fi
  printf '%s\n' "${number}"
}

forge-github:issue-title() {
  gh issue view "$1" --json title --jq '.title' 2>/dev/null \
    || { error "run: gh auth status (the title of #$1 could not be read)"; return 1; }
}

forge-github:issue-state() {
  gh issue view "$1" --json state --jq .state 2>/dev/null
}

forge-github:issue-assignees() {
  gh issue view "$1" --json assignees --jq '[.assignees[].login] | join(" ")' 2>/dev/null
}

forge-github:issue-assign() {
  gh issue edit "$1" --add-assignee "$2" >/dev/null 2>&1
}

# The branch is found by an open pull request's head, so a second run never forks a <N>-<slug>-1 duplicate.
forge-github:branch-for-issue() {
  gh pr list --state open --json headRefName \
    --jq "[.[] | select(.headRefName | test(\"^$1-\"))] | .[0].headRefName // empty" 2>/dev/null || true
}

# gh issue develop names the branch <N>-<slug> itself, links it under Development, and pushes it.
forge-github:link-branch() {
  gh issue develop "$1" --base "$2" >/dev/null
}

forge-github:pr-for-branch() {
  gh pr list --head "$1" --state open --json number --jq '.[0].number // empty' 2>/dev/null || true
}

forge-github:pr-create() {
  local url
  url=$(gh pr create --draft --base "$2" --head "$1" --title "$3" --body "") || return 1
  printf '%s\n' "${url##*/}"
}

forge-github:pr-title() {
  gh pr view "$1" --json title --jq '.title'
}

forge-github:pr-body() {
  gh pr view "$1" --json body --jq '.body // ""'
}

forge-github:pr-set-body() {
  gh pr edit "$1" --body "$2" >/dev/null
}

forge-github:pr-state-for-branch() {
  gh pr list --head "$1" --state all --json state --jq '.[0].state // empty' 2>/dev/null || true
}

# On a runner the event names the pull request; on a desk the current branch does.
forge-github:pr-here() {
  local number=''
  if [[ -n "${GITHUB_EVENT_PATH:-}" ]]
  then
    number=$(jq --raw-output '.pull_request.number // empty' "${GITHUB_EVENT_PATH}" 2>/dev/null || true)
  fi
  [[ -n "${number}" ]] || number=$(gh pr view --json number --jq .number 2>/dev/null || true)
  printf '%s\n' "${number}"
}

forge-github:pr-closing-issues() {
  local repo query
  repo=$(forge-github:repo) || return 1
  query='query($owner: String!, $name: String!, $number: Int!) {
    repository(owner: $owner, name: $name) {
      pullRequest(number: $number) {
        closingIssuesReferences(first: 100) { nodes { number state repository { nameWithOwner } } }
      }
    }
  }'
  gh api graphql --field owner="${repo%%/*}" --field name="${repo#*/}" --field number="$1" \
    --raw-field query="${query}" \
    --jq '.data.repository.pullRequest.closingIssuesReferences.nodes[]
          | "\(.number) \(.state) \(.repository.nameWithOwner)"' 2>/dev/null \
    || { error "run: gh auth status (pull request #$1 could not be read)"; return 1; }
}

# One request: every open pull request with the issues it closes, its review threads, and the checks on its
# head, folded into the document kit status draws: number, ref, branch, title, author, createdAt, age in seconds,
# state, checks {state, passed, total, failing, took in seconds or null}, unresolved, and closes [{number, title}].
forge-github:open-prs() {
  local repo query filter
  repo=$(forge-github:repo) || return 1
  query='query($owner: String!, $name: String!) {
    repository(owner: $owner, name: $name) {
      pullRequests(states: OPEN, first: 100, orderBy: {field: CREATED_AT, direction: ASC}) {
        nodes {
          number title isDraft headRefName createdAt mergeStateStatus reviewDecision
          author { login }
          closingIssuesReferences(first: 10) { nodes { number title } }
          reviewThreads(first: 100) { nodes { isResolved } }
          commits(last: 1) { nodes { commit { statusCheckRollup { contexts(first: 100) { nodes {
            __typename
            ... on CheckRun { name status conclusion startedAt completedAt }
            ... on StatusContext { context state createdAt }
          } } } } } }
        }
      }
    }
  }'
  filter='
    def verdict: if .__typename == "CheckRun" then (if .status == "COMPLETED" then .conclusion else "PENDING" end)
                 else .state end;
    def pending: IN("PENDING", "EXPECTED");
    def passed: IN("SUCCESS", "NEUTRAL", "SKIPPED");
    (now | floor) as $now
    | .data.repository.pullRequests.nodes[]
    | [.commits.nodes[0].commit.statusCheckRollup.contexts.nodes[]?] as $checks
    | ($checks | length) as $n
    | ([$checks[] | verdict | select(pending)] | length) as $running
    | ([$checks[] | verdict | select(passed)] | length) as $ok
    | ([$checks[] | select((verdict | pending or passed) | not) | (.name // .context)] | first // "") as $failing
    | (if $n == 0 then "NONE" elif $failing != "" then "FAILURE" elif $running > 0 then "PENDING" else "SUCCESS" end)
      as $check
    | ([$checks[] | .startedAt // .createdAt // empty | fromdateiso8601] | min) as $started
    | ([$checks[] | .completedAt // empty | fromdateiso8601] | max) as $completed
    | (if $started == null then null
       elif $check == "PENDING" then $now - $started
       elif $completed == null then null
       else $completed - $started end) as $took
    | ([.reviewThreads.nodes[]? | select(.isResolved | not)] | length) as $unresolved
    | (if .isDraft then "draft"
       elif $check == "PENDING" then "queued"
       elif .mergeStateStatus == "DIRTY" then "conflicted"
       elif $check == "FAILURE" then "failing:gate"
       elif $check == "NONE" then "no-check"
       elif $unresolved > 0 then "comments:open"
       elif (.reviewDecision // "") | IN("REVIEW_REQUIRED", "CHANGES_REQUESTED") then "blocked:approval"
       else "ready" end) as $state
    | { number, ref: "#\(.number)", branch: .headRefName, title, author: (.author.login // ""), createdAt,
        age: ($now - (.createdAt | fromdateiso8601)), state: $state,
        checks: { state: $check, passed: $ok, total: $n, failing: $failing, took: $took },
        unresolved: $unresolved,
        closes: [.closingIssuesReferences.nodes[] | { number, title }] }
    | tojson'
  gh api graphql --field owner="${repo%%/*}" --field name="${repo#*/}" --raw-field query="${query}" --jq "${filter}" \
    2>/dev/null || { error "run: gh auth status (the pull requests could not be read)"; return 1; }
}

# ----------------------------------------------------------------------------------------------------------------------
