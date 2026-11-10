# shellcheck shell=bash
# github.sh - the GitHub forge provider, through the gh CLI (sourced module)
# ----------------------------------------------------------------------------------------------------------------------
# shellcheck disable=SC2016  # jq filters and GraphQL queries name their variables with $, in single quotes on purpose
#
# modules/bash/forge.sh loads this provider when the repository's forge is github. It implements the calls listed in the
# header of modules/bash/forge.sh. Only this file requires gh.
#
# A read that fails because GitHub's rate limit is spent says so, and when it resets, as forge/github:limited tells
# it; any other failure names gh auth status.
#
# see: modules/bash/forge.sh

# forge/github:branch-for-issue reads the branch from an open pull request's head, so a second run never creates
# a duplicate <N>-<slug>-1 branch.
forge/github:branch-for-issue() {
  # A failed read is read as none; the forge seam's contract decides this.
  gh pr list --state open --json headRefName \
    --jq "[.[] | select(.headRefName | test(\"^$1-\"))] | .[0].headRefName // empty" 2>/dev/null || true
}

# forge/github:issue-add-sub makes the second issue a sub-issue of the first, as the issue page's Relationships do.
forge/github:issue-add-sub() {
  gh issue edit "$1" --add-sub-issue "$2" >/dev/null 2>&1
}

forge/github:issue-assign() {
  gh issue edit "$1" --add-assignee "$2" >/dev/null 2>&1
}

forge/github:issue-assignees() {
  gh issue view "$1" --json assignees --jq '[.assignees[].login] | join(" ")' 2>/dev/null
}

# forge/github:issue-close closes the issue with GitHub's reason: completed, not planned, or duplicate of the third
# argument.
forge/github:issue-close() {
  local -a how=(--reason "${2//-/ }")
  [[ "$2" != duplicate ]] || how+=(--duplicate-of "$3")
  gh issue close "$1" "${how[@]}" >/dev/null 2>&1
}

forge/github:issue-comment() {
  gh issue comment "$1" --body "$2" >/dev/null 2>&1
}

forge/github:issue-create() {
  local number
  number=$(gh issue create --title "$1" --body "${2:-}" 2>/dev/null | sed -n 's@.*/issues/@@p')
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
  local title
  forge/github:ask title "the title of #$1" issue view "$1" --json title --jq '.title' || return 1
  printf '%s\n' "${title}"
}

forge/github:issue-unassign() {
  gh issue edit "$1" --remove-assignee "$2" >/dev/null 2>&1
}

# gh issue develop names the branch <N>-<slug>, links it to the issue under Development, and pushes it.
forge/github:link-branch() {
  gh issue develop "$1" --base "$2" >/dev/null
}

# forge/github:marks [<dir>] reads, for each kind of the repository's data, a mark that changes when it does: issues,
# of the issue or pull request updated last, as GitHub lists them together; runs, of the ten newest workflow runs;
# releases, of the newest release. A mark is GitHub's ETag for the answer. With <dir>, each ETag is kept there and the
# next read sends it, so GitHub answers 304 Not Modified while nothing changed, which counts against no rate limit.
# gh names the repository from the checkout's remote, as {owner}/{repo}, which costs no request.
forge/github:marks() {
  local dir=${1:-} issues runs releases
  issues=$(forge/github:mark 'repos/{owner}/{repo}/issues?state=all&sort=updated&direction=desc&per_page=1' \
    "${dir:+${dir}/marks-issues.etag}") || return 1
  runs=$(forge/github:mark 'repos/{owner}/{repo}/actions/runs?per_page=10' "${dir:+${dir}/marks-runs.etag}") \
    || return 1
  releases=$(forge/github:mark 'repos/{owner}/{repo}/releases?per_page=1' "${dir:+${dir}/marks-releases.etag}") \
    || return 1
  jq --null-input --compact-output --arg issues "${issues}" --arg runs "${runs}" --arg releases "${releases}" \
    '{issues: $issues, runs: $runs, releases: $releases}'
}

forge/github:me() {
  local login said
  file:temp said gh || return 1
  if login=$(gh api user --jq '.login' 2>"${said}")
  then
    printf '%s\n' "${login}"
    return 0
  fi
  forge/github:limited "the caller's login" "$(<"${said}")" \
    || error "run: gh auth login (the GitHub CLI is not authenticated)"
  return 1
}

forge/github:noun() { printf '#\tpull request\n'; }

# forge/github:open-issues prints one document per open issue, oldest first, as the seam's header says. gh issue list
# answers with issues alone, never pull requests.
forge/github:open-issues() {
  local answer
  forge/github:ask answer 'the issues' issue list --state open --limit 1000 \
    --json number,title,assignees,labels,createdAt,url \
    || return 1
  # shellcheck disable=SC2154  # kit_jq is exported by modules/bash/1st.sh
  jq -L "${kit_jq}" --compact-output --argjson now "$(date +%s)" 'import "forge" as forge; forge::github_issues($now)' \
    <<<"${answer}"
}

# forge/github:open-prs makes one GraphQL request for every open pull request, with its labels, the issues it closes
# and theirs, its review threads, the owner of the repository its branch lives in, and its head commit: when it was
# committed, by whom, and the checks on it. It prints one JSON object per pull request: the document kit status
# renders, which modules/jq/forge.jq folds the answer into. Of several runs of one name on the head commit, as after a
# rerun, only the latest counts.
#
# GitHub charges a request by the nodes it may return, so assignees asked of each issue each pull request closes cost
# a hundred times as much as the rest of the request. The same request asks instead for the assignees of the hundred
# open issues with an assignee most recently updated, and forge.jq gives each closed issue its assignees from them.
# Labels are asked under each closed issue all the same, as no other list holds them all, and a pull request is
# asked for the first 10 issues it closes, not 100, which keeps them cheap. GitHub's dry run, rateLimit(dryRun: true),
# puts the request at 16 points of the GraphQL rate limit; without any labels, it costs 5, and with the labels under
# the first 100 issues each closes, 106. A pull request that closes more than 10 says so, as its list's next page, and
# forge/github:rest-of-closes reads the rest of it, 100 at a time, a point each.
forge/github:open-prs() {
  local repo query answer number cursor rest
  repo=$(forge/github:repo) || return 1
  query='query($owner: String!, $name: String!) {
    repository(owner: $owner, name: $name) {
      nameWithOwner
      issues(states: OPEN, first: 100, filterBy: {assignee: "*"}, orderBy: {field: UPDATED_AT, direction: DESC}) {
        nodes { number assignees(first: 10) { nodes { login } } }
      }
      pullRequests(states: OPEN, first: 100, orderBy: {field: CREATED_AT, direction: ASC}) {
        nodes {
          number title isDraft headRefName createdAt mergeStateStatus reviewDecision isCrossRepository
          author { login }
          headRepositoryOwner { login }
          labels(first: 20) { nodes { name color } }
          closingIssuesReferences(first: 10) { pageInfo { hasNextPage endCursor } nodes {
            number title createdAt repository { nameWithOwner } labels(first: 20) { nodes { name color } }
          } }
          reviewThreads(first: 100) { nodes { isResolved } }
          commits(last: 1) { nodes { commit {
            committedDate author { user { login } }
            statusCheckRollup { contexts(first: 100) { nodes {
              __typename
              ... on CheckRun { name status conclusion startedAt completedAt }
              ... on StatusContext { context state createdAt }
            } } }
          } } }
        }
      }
    }
  }'
  forge/github:ask answer 'the pull requests' api graphql --field owner="${repo%%/*}" --field name="${repo#*/}" \
    --raw-field query="${query}" || return 1
  while IFS=$'\t' read -r number cursor
  do
    [[ -n "${number}" ]] || continue
    rest=$(forge/github:rest-of-closes "${repo}" "${number}" "${cursor}") || return 1
    answer=$(jq --compact-output --argjson number "${number}" --argjson rest "${rest}" \
      '.data.repository.pullRequests.nodes |= map(if .number == $number then .closingIssuesReferences.nodes += $rest
                                                  else . end)' <<<"${answer}")
  done < <(jq --raw-output '.data.repository.pullRequests.nodes[]
    | select(.closingIssuesReferences.pageInfo.hasNextPage) | "\(.number)\t\(.closingIssuesReferences.pageInfo.endCursor)"' \
    <<<"${answer}")
  # shellcheck disable=SC2154  # kit_jq is exported by modules/bash/1st.sh
  jq -L "${kit_jq}" --compact-output --argjson now "$(date +%s)" 'import "forge" as forge; forge::github_prs($now)' \
    <<<"${answer}"
}

# The REST API names a bot by the login the pull request page shows, dependabot[bot]. gh's own JSON names the same
# author app/dependabot, which is not a login anyone would write in a setting.
forge/github:pr-author() {
  local repo author
  repo=$(forge/github:repo) || return 1
  forge/github:ask author "the author of #$1" api "repos/${repo}/pulls/$1" --jq '.user.login' || return 1
  printf '%s\n' "${author}"
}

forge/github:pr-body() {
  gh pr view "$1" --json body --jq '.body // ""'
}

forge/github:pr-closing-issues() {
  local repo query issues
  repo=$(forge/github:repo) || return 1
  query='query($owner: String!, $name: String!, $number: Int!) {
    repository(owner: $owner, name: $name) {
      pullRequest(number: $number) {
        closingIssuesReferences(first: 100) { nodes { number state repository { nameWithOwner } } }
      }
    }
  }'
  forge/github:ask issues "pull request #$1" api graphql --field owner="${repo%%/*}" --field name="${repo#*/}" \
    --field number="$1" --raw-field query="${query}" \
    --jq '.data.repository.pullRequest.closingIssuesReferences.nodes[]
          | "\(.number) \(.state) \(.repository.nameWithOwner)"' || return 1
  [[ -z "${issues}" ]] || printf '%s\n' "${issues}"
}

forge/github:pr-create() {
  local url
  url=$(gh pr create --draft --base "$2" --head "$1" --title "$3" --body "") || return 1
  printf '%s\n' "${url##*/}"
}

forge/github:pr-for-branch() {
  # A failed read is read as none; the forge seam's contract decides this.
  gh pr list --head "$1" --state open --json number --jq '.[0].number // empty' 2>/dev/null || true
}

forge/github:pr-head-for-branch() {
  # A failed read is read as none; the forge seam's contract decides this.
  gh pr list --head "$1" --state all --json headRefOid --jq '.[0].headRefOid // empty' 2>/dev/null || true
}

# On a GitHub Actions runner, the event payload gives the pull request. Otherwise gh finds the pull request for
# the current branch.
forge/github:pr-here() {
  local number=''
  if [[ -n "${GITHUB_EVENT_PATH:-}" ]]
  then
    # A payload with no pull request gives none, and gh is asked.
    number=$(jq --raw-output '.pull_request.number // empty' "${GITHUB_EVENT_PATH}" 2>/dev/null || true)
  fi
  # gh's exit 1 is a branch with no pull request, and also a read that failed: both read as none, as the forge seam's
  # contract decides.
  [[ -n "${number}" ]] || number=$(gh pr view --json number --jq .number 2>/dev/null || true)
  printf '%s\n' "${number}"
}

forge/github:pr-last-for-branch() {
  # A failed read is read as none; the forge seam's contract decides this.
  gh pr list --head "$1" --state all --json number --jq '.[0].number // empty' 2>/dev/null || true
}

forge/github:pr-set-body() {
  gh pr edit "$1" --body "$2" >/dev/null
}

forge/github:pr-state() {
  gh pr view "$1" --json state,mergeCommit --jq '"\(.state)\t\(.mergeCommit.oid // "")"' 2>/dev/null \
    || { error "run: gh auth status (pull request #$1 could not be read)"; return 1; }
}

forge/github:pr-state-for-branch() {
  # A failed read is read as none; the forge seam's contract decides this.
  gh pr list --head "$1" --state all --json state --jq '.[0].state // empty' 2>/dev/null || true
}

forge/github:pr-title() {
  gh pr view "$1" --json title --jq '.title'
}

forge/github:pr-url() { printf '%s/pull/%s\n' "$(forge/github:repo-url)" "$1"; }

# forge/github:quota reads the rate_limit endpoint, which no limit counts, and keeps the two kit spends: GraphQL's
# points and REST's requests.
forge/github:quota() {
  local limits
  forge/github:ask limits 'the rate limits' api rate_limit || return 1
  # shellcheck disable=SC2154  # kit_jq is exported by modules/bash/1st.sh
  jq -L "${kit_jq}" --compact-output 'import "forge" as forge; forge::github_quota' <<<"${limits}"
}

# forge/github:releases prints one document per release, newest first, as the seam's header says.
forge/github:releases() {
  local answer
  forge/github:ask answer 'the releases' release list --limit 1000 \
    --json tagName,name,isPrerelease,isDraft,isLatest,publishedAt,createdAt || return 1
  # shellcheck disable=SC2154  # kit_jq is exported by modules/bash/1st.sh
  jq -L "${kit_jq}" --compact-output --argjson now "$(date +%s)" \
    'import "forge" as forge; forge::github_releases($now)' \
    <<<"${answer}"
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
  forge/github:me >/dev/null
}

# forge/github:runs prints the builds stage's runs as one JSON array, as the seam's header says. It reads the 50
# newest runs and keeps those forge.jq's builds keeps. A run's jobs cost a call each, so only the 5 newest runs carry
# them; a run whose jobs cannot be read keeps none.
forge/github:runs() {
  local answer runs id jobs index=0 default tags
  local fields=databaseId,number,attempt,workflowName,headBranch,headSha,displayTitle,event,status,conclusion
  fields+=,createdAt,startedAt,updatedAt,url
  forge/github:ask answer 'the runs' run list --limit 50 --json "${fields}" || return 1
  default=$(git:default-branch) || return 1
  tags=$(forge:tags) || return 1
  # shellcheck disable=SC2154  # kit_jq is exported by modules/bash/1st.sh
  runs=$(jq -L "${kit_jq}" --compact-output --argjson now "$(date +%s)" --arg default "${default}" \
    --argjson tags "${tags}" 'import "forge" as forge; forge::github_builds($now; $default; $tags)' \
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

# The seam's calls end here; what follows is this provider's own.

# forge/github:mark <path> [<file>] prints the ETag of GitHub's answer for <path>: the one kept in <file> when GitHub
# answers 304 Not Modified to it, else the new one, which it keeps there. gh exits 1 on a 304, so the answer's status
# line says what came back. An answer that is neither fails.
forge/github:mark() {
  local path=$1 file=${2:-} etag='' answer status
  local -a header=()
  [[ -z "${file}" || ! -f "${file}" ]] || etag=$(<"${file}")
  [[ -z "${etag}" ]] || header=(--header "If-None-Match: ${etag}")
  answer=$(gh api --include "${header[@]}" "${path}" 2>/dev/null) || true  # the status line decides, below
  status=$(sed -n '1s@^HTTP/[0-9.]* \([0-9]*\).*@\1@p' <<<"${answer}")
  case "${status}" in
    (304) printf '%s\n' "${etag}" ;;
    (200)
      etag=$(tr -d '\r' <<<"${answer}" | sed -n 's/^[Ee][Tt][Aa][Gg]: *//p' | head -n 1)
      if [[ -z "${etag}" ]]
      then
        error "run: gh auth status (GitHub gave no ETag for ${path})"
        return 1
      fi
      # An ETag that cannot be kept costs only the next read, which asks GitHub for the whole answer.
      [[ -z "${file}" ]] || file:replace "${file}" printf '%s\n' "${etag}" || true
      printf '%s\n' "${etag}"
      ;;
    (*)
      error "run: gh auth status (the marks could not be read)"
      return 1
  esac
}

# forge/github:rest-of-closes <owner/name> <number> <cursor> prints, as one JSON array, the issues the pull request
# closes after the cursor its first page ended on, each as forge/github:open-prs asks for one, reading 100 a request,
# a point of the GraphQL rate limit each, until the list ends.
forge/github:rest-of-closes() {
  local repo=$1 number=$2 cursor=$3 query page all='[]' more=true
  query='query($owner: String!, $name: String!, $pr: Int!, $after: String) {
    repository(owner: $owner, name: $name) {
      pullRequest(number: $pr) {
        closingIssuesReferences(first: 100, after: $after) { pageInfo { hasNextPage endCursor } nodes {
          number title createdAt repository { nameWithOwner } labels(first: 20) { nodes { name color } }
        } }
      }
    }
  }'
  while [[ "${more}" == true ]]
  do
    forge/github:ask page "the rest of the issues #${number} closes" api graphql --field owner="${repo%%/*}" \
      --field name="${repo#*/}" --field pr="${number}" --raw-field after="${cursor}" --raw-field query="${query}" \
      || return 1
    all=$(jq --compact-output --argjson all "${all}" '$all + .data.repository.pullRequest.closingIssuesReferences.nodes' \
      <<<"${page}")
    more=$(jq --raw-output '.data.repository.pullRequest.closingIssuesReferences.pageInfo.hasNextPage' <<<"${page}")
    cursor=$(jq --raw-output '.data.repository.pullRequest.closingIssuesReferences.pageInfo.endCursor // ""' <<<"${page}")
  done
  printf '%s\n' "${all}"
}

# forge/github:ask <name> <what> <gh argument>… runs gh and puts what it printed in the variable <name>. When gh
# fails, it says why, as forge/github:limited does, or else names gh auth status, for <what>, and returns 1.
forge/github:ask() {
  local into=$1 what=$2 said out
  shift 2
  file:temp said gh || return 1
  if ! out=$(gh "$@" 2>"${said}")
  then
    forge/github:limited "${what}" "${out}"$'\n'"$(<"${said}")" \
      || error "run: gh auth status (${what} could not be read)"
    return 1
  fi
  printf -v "${into}" '%s' "${out}"
}

# forge/github:limited <what> <said> says, when what gh said is that a rate limit stopped the read of <what>, which
# limit it is and when it resets, and succeeds; for anything else it says nothing and fails. GitHub answers a
# GraphQL request over its limit with RATE_LIMITED, and a REST request with API rate limit exceeded; the reset comes
# from the rate_limit endpoint, which no limit counts. A secondary rate limit, which GitHub sets on bursts of
# requests, names no reset: GitHub asks for a wait of a few minutes.
forge/github:limited() {
  local what=$1 said=${2,,} limits wait
  if [[ "${said}" == *'secondary rate limit'* || "${said}" == *'abuse detection'* ]]
  then
    error "wait a few minutes: GitHub's secondary rate limit refused the read (${what} could not be read)"
    return 0
  fi
  [[ "${said}" == *rate_limited* || "${said}" == *'rate limit'* ]] || return 1
  limits=$(gh api rate_limit 2>/dev/null) || limits='{}'
  # shellcheck disable=SC2154  # kit_jq is exported by modules/bash/1st.sh
  wait=$(jq -L "${kit_jq}" --raw-output --argjson now "$(date +%s)" \
    'import "forge" as forge; forge::github_spent($now)' <<<"${limits}" 2>/dev/null) || wait=''
  [[ -n "${wait}" ]] || wait="wait for GitHub's rate limit to reset; run: gh api rate_limit"
  error "${wait} (${what} could not be read)"
}

# ----------------------------------------------------------------------------------------------------------------------
