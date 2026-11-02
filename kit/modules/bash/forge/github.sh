# shellcheck shell=bash
# github.sh - the GitHub forge provider, through the gh CLI (sourced module)
# ----------------------------------------------------------------------------------------------------------------------
# shellcheck disable=SC2016  # jq filters and GraphQL queries name their variables with $, in single quotes on purpose
#
# modules/bash/forge.sh loads this provider when the repository's forge is github. It implements the calls listed in the
# header of modules/bash/forge.sh. Only this file requires gh.
#
# see: modules/bash/forge.sh

# forge/github:branch-for-issue reads the branch from an open pull request's head, so a second run never creates
# a duplicate <N>-<slug>-1 branch.
forge/github:branch-for-issue() {
  gh pr list --state open --json headRefName \
    --jq "[.[] | select(.headRefName | test(\"^$1-\"))] | .[0].headRefName // empty" 2>/dev/null || true
}

forge/github:issue-assign() {
  gh issue edit "$1" --add-assignee "$2" >/dev/null 2>&1
}

forge/github:issue-assignees() {
  gh issue view "$1" --json assignees --jq '[.assignees[].login] | join(" ")' 2>/dev/null
}

forge/github:issue-create() {
  local number
  number=$(gh issue create --title "$1" --body "" 2>/dev/null | sed -n 's@.*/issues/@@p')
  if [[ ! "${number}" =~ ^[0-9]+$ ]]
  then
    error "run: gh auth status (the issue could not be created)"
    return 1
  fi
  printf '%s\n' "${number}"
}

forge/github:issue-state() {
  gh issue view "$1" --json state --jq .state 2>/dev/null
}

forge/github:issue-title() {
  gh issue view "$1" --json title --jq '.title' 2>/dev/null \
    || { error "run: gh auth status (the title of #$1 could not be read)"; return 1; }
}

# gh issue develop names the branch <N>-<slug>, links it to the issue under Development, and pushes it.
forge/github:link-branch() {
  gh issue develop "$1" --base "$2" >/dev/null
}

forge/github:me() {
  gh api user --jq '.login' 2>/dev/null && return 0
  error "run: gh auth login (the GitHub CLI is not authenticated)"
  return 1
}

forge/github:noun() { printf '#\tpull request\n'; }

# forge/github:open-issues prints one document per open issue, oldest first, as the seam's header says. gh issue list
# answers with issues alone, never pull requests.
forge/github:open-issues() {
  local answer
  answer=$(gh issue list --state open --limit 1000 --json number,title,assignees,createdAt 2>/dev/null) \
    || { error "run: gh auth status (the issues could not be read)"; return 1; }
  # shellcheck disable=SC2154  # kit_jq is exported by modules/bash/1st.sh
  jq -L "${kit_jq}" --compact-output --argjson now "$(date +%s)" 'import "forge" as forge; forge::github_issues($now)' \
    <<<"${answer}"
}

# forge/github:open-prs makes one GraphQL request for every open pull request, with the issues it closes, its
# review threads, and the checks on its head commit. It prints one JSON object per pull request: the document kit
# status renders, which modules/jq/forge.jq folds the answer into. Of several runs of one name on the head commit,
# as after a rerun, only the latest counts.
forge/github:open-prs() {
  local repo query answer
  repo=$(forge/github:repo) || return 1
  query='query($owner: String!, $name: String!) {
    repository(owner: $owner, name: $name) {
      pullRequests(states: OPEN, first: 100, orderBy: {field: CREATED_AT, direction: ASC}) {
        nodes {
          number title isDraft headRefName createdAt mergeStateStatus reviewDecision
          author { login }
          closingIssuesReferences(first: 100) {
            nodes { number title createdAt assignees(first: 10) { nodes { login } } }
          }
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

# The REST API names a bot by the login the pull request page shows, dependabot[bot]. gh's own JSON names the same
# author app/dependabot, which is not a login anyone would write in a setting.
forge/github:pr-author() {
  local repo
  repo=$(forge/github:repo) || return 1
  gh api "repos/${repo}/pulls/$1" --jq '.user.login' 2>/dev/null \
    || { error "run: gh auth status (the author of #$1 could not be read)"; return 1; }
}

forge/github:pr-body() {
  gh pr view "$1" --json body --jq '.body // ""'
}

forge/github:pr-closing-issues() {
  local repo query
  repo=$(forge/github:repo) || return 1
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

forge/github:pr-create() {
  local url
  url=$(gh pr create --draft --base "$2" --head "$1" --title "$3" --body "") || return 1
  printf '%s\n' "${url##*/}"
}

forge/github:pr-for-branch() {
  gh pr list --head "$1" --state open --json number --jq '.[0].number // empty' 2>/dev/null || true
}

# On a GitHub Actions runner, the event payload gives the pull request. Otherwise gh finds the pull request for
# the current branch.
forge/github:pr-here() {
  local number=''
  if [[ -n "${GITHUB_EVENT_PATH:-}" ]]
  then
    number=$(jq --raw-output '.pull_request.number // empty' "${GITHUB_EVENT_PATH}" 2>/dev/null || true)
  fi
  [[ -n "${number}" ]] || number=$(gh pr view --json number --jq .number 2>/dev/null || true)
  printf '%s\n' "${number}"
}

forge/github:pr-set-body() {
  gh pr edit "$1" --body "$2" >/dev/null
}

forge/github:pr-state-for-branch() {
  gh pr list --head "$1" --state all --json state --jq '.[0].state // empty' 2>/dev/null || true
}

forge/github:pr-title() {
  gh pr view "$1" --json title --jq '.title'
}

forge/github:pr-url() { printf '%s/pull/%s\n' "$(forge/github:repo-url)" "$1"; }

# forge/github:releases prints one document per release, newest first, as the seam's header says.
forge/github:releases() {
  local answer
  answer=$(gh release list --limit 1000 --json tagName,name,isPrerelease,isDraft,isLatest,publishedAt,createdAt \
    2>/dev/null) || { error "run: gh auth status (the releases could not be read)"; return 1; }
  # shellcheck disable=SC2154  # kit_jq is exported by modules/bash/1st.sh
  jq -L "${kit_jq}" --compact-output --argjson now "$(date +%s)" \
    'import "forge" as forge; forge::github_releases($now)' \
    <<<"${answer}"
}

# forge/github:runs prints the builds stage's runs as one JSON array, as the seam's header says. It reads the 50
# newest runs and keeps those forge.jq's builds keeps. A run's jobs cost a call each, so only the 5 newest runs carry
# them; a run whose jobs cannot be read keeps none.
forge/github:runs() {
  local answer runs id jobs index=0
  answer=$(gh run list --limit 50 \
    --json databaseId,workflowName,headBranch,event,status,conclusion,createdAt,startedAt,updatedAt,url \
    2>/dev/null) || { error "run: gh auth status (the runs could not be read)"; return 1; }
  # shellcheck disable=SC2154  # kit_jq is exported by modules/bash/1st.sh
  runs=$(jq -L "${kit_jq}" --compact-output --argjson now "$(date +%s)" --arg default "$(git:default-branch)" \
    --argjson tags "$(forge:tags)" 'import "forge" as forge; forge::github_builds($now; $default; $tags)' \
    <<<"${answer}") || return 1
  while IFS= read -r id
  do
    jobs=$(gh run view "${id}" --json jobs 2>/dev/null | jq -L "${kit_jq}" --compact-output \
      'import "forge" as forge; forge::github_jobs' 2>/dev/null) || jobs='[]'
    runs=$(jq --compact-output --argjson index "${index}" --argjson jobs "${jobs:-[]}" '.[$index].jobs = $jobs' \
      <<<"${runs}")
    index=$(( index + 1 ))
  done < <(jq --raw-output '.[:5][].id' <<<"${runs}")
  printf '%s\n' "${runs}"
}

forge/github:repo() {
  gh repo view --json nameWithOwner --jq .nameWithOwner 2>/dev/null \
    || { error "run: gh repo view (this checkout does not resolve to a GitHub repository)"; return 1; }
}

forge/github:repo-url() {
  gh repo view --json url --jq .url 2>/dev/null \
    || { error "run: gh repo view (this checkout does not resolve to a GitHub repository)"; return 1; }
}

# forge/github:require fails when gh is missing or not authenticated. It makes a real API call, because
# `gh auth status` passes with a revoked token.
forge/github:require() {
  guard:require tool:gh || return 1
  if ! gh api user --jq '.login' >/dev/null 2>&1
  then
    error "run: gh auth login (the GitHub CLI is not authenticated)"
    return 1
  fi
}

# The seam's calls end here; what follows is this provider's own.

# ----------------------------------------------------------------------------------------------------------------------
