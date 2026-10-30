# shellcheck shell=bash
# forge-github.sh - the GitHub forge provider, through the gh CLI (sourced module)
# ----------------------------------------------------------------------------------------------------------------------
# shellcheck disable=SC2016  # jq filters and GraphQL queries name their variables with $, in single quotes on purpose
#
# modules/bash/forge.sh loads this provider when the repository's forge is github. It implements the calls listed in the
# header of modules/bash/forge.sh. Only this file requires gh.
#
# see: modules/bash/forge.sh

# forge-github:require fails when gh is missing or not authenticated. It makes a real API call, because
# `gh auth status` passes with a revoked token.
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

# forge-github:branch-for-issue reads the branch from an open pull request's head, so a second run never creates
# a duplicate <N>-<slug>-1 branch.
forge-github:branch-for-issue() {
  gh pr list --state open --json headRefName \
    --jq "[.[] | select(.headRefName | test(\"^$1-\"))] | .[0].headRefName // empty" 2>/dev/null || true
}

# gh issue develop names the branch <N>-<slug>, links it to the issue under Development, and pushes it.
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

# The REST API names a bot by the login the pull request page shows, dependabot[bot]. gh's own JSON names the same
# author app/dependabot, which is not a login anyone would write in a setting.
forge-github:pr-author() {
  local repo
  repo=$(forge-github:repo) || return 1
  gh api "repos/${repo}/pulls/$1" --jq '.user.login' 2>/dev/null \
    || { error "run: gh auth status (the author of #$1 could not be read)"; return 1; }
}

forge-github:pr-state-for-branch() {
  gh pr list --head "$1" --state all --json state --jq '.[0].state // empty' 2>/dev/null || true
}

# On a GitHub Actions runner, the event payload gives the pull request. Otherwise gh finds the pull request for
# the current branch.
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

# forge-github:open-prs makes one GraphQL request for every open pull request, with the issues it closes, its
# review threads, and the checks on its head commit. It prints one JSON object per pull request: the document kit
# status renders, which modules/jq/forge.jq folds the answer into. Of several runs of one name on the head commit,
# as after a rerun, only the latest counts.
forge-github:open-prs() {
  local repo query answer
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
  answer=$(gh api graphql --field owner="${repo%%/*}" --field name="${repo#*/}" --raw-field query="${query}" \
    2>/dev/null) || { error "run: gh auth status (the pull requests could not be read)"; return 1; }
  # shellcheck disable=SC2154  # kit_jq is exported by modules/bash/1st.sh
  jq -L "${kit_jq}" --compact-output --argjson now "$(date +%s)" 'import "forge" as forge; forge::github_prs($now)' \
    <<<"${answer}"
}

# forge-github:open-issues prints one document per open issue, oldest first, as the seam's header says. gh issue list
# answers with issues alone, never pull requests.
forge-github:open-issues() {
  local answer
  answer=$(gh issue list --state open --limit 1000 --json number,title,assignees,createdAt 2>/dev/null) \
    || { error "run: gh auth status (the issues could not be read)"; return 1; }
  # shellcheck disable=SC2154  # kit_jq is exported by modules/bash/1st.sh
  jq -L "${kit_jq}" --compact-output --argjson now "$(date +%s)" 'import "forge" as forge; forge::github_issues($now)' \
    <<<"${answer}"
}

# ----------------------------------------------------------------------------------------------------------------------
