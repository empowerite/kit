# shellcheck shell=bash
# github.sh - the GitHub forge provider, through the gh CLI (sourced module)
# ----------------------------------------------------------------------------------------------------------------------
# shellcheck disable=SC2154  # kit_jq is exported by modules/bash/1st.sh, forge_list_limit set by modules/bash/forge.sh
# shellcheck disable=SC2016  # jq filters and GraphQL queries name their variables with $, in single quotes on purpose
#
# modules/bash/forge.sh loads this provider when the repository's forge is github. It implements the calls listed in the
# header of modules/bash/forge.sh. Only this file requires gh.
#
# A read that fails because GitHub's rate limit is spent says so, and when it resets, as forge/github:limited tells
# it; any other failure names gh auth status.
#
# see: modules/bash/forge.sh

# forge/github:artifacts prints one document per artifact of the repository's runs, newest first, as the seam's header
# says: one REST request for the newest 100, and {"limited": 100} after them when GitHub holds more.
forge/github:artifacts() {
  local answer url
  url=$(forge/github:repo-url) || return 1
  forge/github:ask answer 'the artifacts' api 'repos/{owner}/{repo}/actions/artifacts?per_page=100' || return 1
  forge/github:fold 'the artifacts' "${answer}" --exit-status '.artifacts | arrays' >/dev/null || return 1
  forge/github:fold 'the artifacts' "${answer}" --compact-output --argjson now "$(date +%s)" --arg url "${url}" \
    'import "forge" as forge; forge::github_artifacts($now; $url)'
}

# forge/github:attestations prints one document per attestation of the newest release assets and artifacts, as the
# seam's header says: one REST request for the 6 newest releases, the 6th saying there are more than the 5 read, one
# for the 20 newest artifacts, then one for the attestations of each digest among the 5 releases' assets and the
# artifacts, newest first, each once, 50 at most. A digest GitHub holds none of answers 404, and has none.
# {"limited": N} follows the N drawn where there are more releases, artifacts, or digests than those read.
forge/github:attestations() {
  local releases artifacts subjects url now row name digest made answer said more=false out=
  local -i drawn=0
  local -a rows
  url=$(forge/github:repo-url) || return 1
  now=$(date +%s)
  forge/github:ask releases 'the attestations' api 'repos/{owner}/{repo}/releases?per_page=6' || return 1
  forge/github:ask artifacts 'the attestations' api 'repos/{owner}/{repo}/actions/artifacts?per_page=20' || return 1
  subjects=$(forge/github:fold 'the attestations' "${releases}"$'\n'"${artifacts}" --slurp --compact-output '
    (.[0] | arrays // error) as $releases | (.[1].artifacts | arrays // error) as $artifacts
    | (.[1].total_count // 0) as $total
    | ([$releases[:5][] | .assets[]? | {name, digest, made: .created_at}]
       + [$artifacts[] | {name, digest, made: .created_at}])
    | map(select((.digest | type) == "string")) | reduce .[] as $s ([]; if any(.[]; .digest == $s.digest) then .
                                                                         else . + [$s] end)
    | {subjects: .[:50],
       more: (($releases | length) > 5 or $total > ($artifacts | length) or length > 50)}') \
    || return 1
  [[ "$(jq --raw-output '.more' <<<"${subjects}")" == false ]] || more=true
  file:temp said gh || return 1
  list:read rows jq --raw-output '.subjects[] | [.name, .digest, (.made // "")] | @tsv' <<<"${subjects}" || return 1
  for row in "${rows[@]}"
  do
    [[ -n "${row}" ]] || continue
    IFS=$'\t' read -r name digest made <<<"${row}"
    if ! answer=$(gh api --paginate --slurp "repos/{owner}/{repo}/attestations/${digest}?per_page=100" 2>"${said}")
    then
      # A digest GitHub keeps no attestation of answers 404.
      [[ "$(<"${said}")" == *'(HTTP 404)'* ]] && continue
      forge/github:limited 'the attestations' "${answer}"$'\n'"$(<"${said}")" \
        || forge/github:scoped 'the attestations' "$(<"${said}")" \
        || error "run: gh auth status (the attestations of ${name} could not be read)"
      return 1
    fi
    answer=$(forge/github:fold "the attestations of ${name}" "${answer}" --compact-output --arg subject "${name}" \
      --arg digest "${digest}" --arg made "${made}" --argjson now "${now}" --arg url "${url}/attestations" \
      'import "forge" as forge;
       if all(.[]; type == "object") then . else error end
       | forge::github_attestations($subject; $digest; (if $made == "" then null else $made end); $now; $url)') \
      || return 1
    [[ -z "${answer}" ]] || { out+=${answer}$'\n'; drawn+=$(wc -l <<<"${answer}"); }
  done
  # Nothing is printed until every digest is read, so a read that fails prints nothing.
  [[ "${more}" == false ]] || out+=$(printf '{"limited":%d}' "${drawn}")$'\n'
  printf '%s' "${out}"
}

# forge/github:bots prints one document per bot that acted in the repository, the last to act first, as the seam's
# header says: one GraphQL request, two points, for the authors of the 100 newest commits on the default branch, and
# of the 100 open pull requests and the 50 merged pull requests updated last, with each one's latest review by each
# reviewer. A bot is a Bot account, or a login that ends [bot]; a commit is a bot's also when its author names itself
# one, as an agent's commit names the account of the person it works for. A commit by no account GitHub knows is no
# one's. The bots are those that acted in what it reads, and no more: one that acted only before is not among them.
# More than 100 open pull requests, or more than 20 reviewers of one, end the list with {"limited": N}, N those drawn.
forge/github:bots() {
  local repo query answer
  repo=$(forge/github:repo) || return 1
  query='query($owner: String!, $name: String!) {
    repository(owner: $owner, name: $name) {
      url
      defaultBranchRef { target { ... on Commit { history(first: 100) { nodes {
        committedDate author { name user { login __typename } }
      } } } } }
      open: pullRequests(states: OPEN, first: 100, orderBy: {field: UPDATED_AT, direction: DESC}) {
        totalCount nodes { ...acts }
      }
      merged: pullRequests(states: MERGED, first: 50, orderBy: {field: UPDATED_AT, direction: DESC}) {
        nodes { ...acts }
      }
    }
  }
  fragment acts on PullRequest {
    createdAt author { __typename login url }
    latestReviews(first: 20) { totalCount nodes { submittedAt author { __typename login url } } }
  }'
  forge/github:ask answer 'the bots' api graphql --field owner="${repo%%/*}" --field name="${repo#*/}" \
    --raw-field query="${query}" || return 1
  forge/github:fold 'the bots' "${answer}" --exit-status '.data.repository | objects' >/dev/null || return 1
  forge/github:fold 'the bots' "${answer}" --compact-output 'import "forge" as forge; forge::github_bots'
}

# forge/github:branch-for-issue reads the branch from an open pull request's head, so a second run never creates
# a duplicate <N>-<slug>-1 branch.
forge/github:branch-for-issue() {
  local heads
  forge/github:ask heads 'the open pull requests' pr list --state open --json headRefName \
    --limit "${forge_list_limit}" || return 1
  forge:whole 'the open pull requests' "$(jq length <<<"${heads}")" || return 1
  forge/github:fold 'the open pull requests' "${heads}" --raw-output --arg n "$1" \
    '[.[] | select(.headRefName | startswith("\($n)-"))] | .[0].headRefName // empty'
}

# forge/github:bypasses prints one document per bypass of the default branch's controls, as the seam's header says:
# the bypass actors of each ruleset that guards it, and those its own protection lets bypass a pull request, read as
# forge/github:rulesets reads them. Those GitHub hides, or that kit cannot name, are one document a ruleset.
forge/github:bypasses() {
  local answer protection
  forge/github:controls answer protection || return 1
  forge/github:fold 'the bypasses' "${answer}" --compact-output --argjson protection "${protection}" \
    'import "forge" as forge; forge::github_bypasses($protection | if type == "object" then .full else null end)'
}

# forge/github:code-scanning-alerts prints one document per code scanning alert, open or dismissed, as the seam's
# header says: GitHub takes one state a read, so two REST reads, every page of each, each by its last update. A
# repository with no analysis, or without code scanning, has it off, and answers 3.
forge/github:code-scanning-alerts() {
  local pages alerts='sort=updated&direction=desc&per_page=100'
  forge/github:alert-pages pages 'the code scanning alerts' \
    "repos/{owner}/{repo}/code-scanning/alerts?state=open&${alerts}" \
    "repos/{owner}/{repo}/code-scanning/alerts?state=dismissed&${alerts}" || return $?
  forge/github:fold 'the code scanning alerts' "${pages}" --compact-output --argjson now "$(date +%s)" \
    'import "forge" as forge; forge::github_code_scanning($now)'
}

# forge/github:collaborators prints one document per person or bot with access to the repository, as the seam's header
# says: every page of the REST list, which holds those with access of their own, through a team, and as the
# organization's members. GitHub shows it only to a caller who may push, and answers any other 403, "Must have push
# access", which the failure says.
forge/github:collaborators() {
  local repo answer
  repo=$(forge/github:repo) || return 1
  forge/github:refused answer 'the collaborators' '*push access*' \
    "ask an admin of ${repo} for write access: GitHub shows who has access only to who may push" \
    api --paginate --slurp 'repos/{owner}/{repo}/collaborators?per_page=100' || return 1
  forge:whole 'the collaborators' "$(jq 'if all(.[]; type == "array") then map(length) | add // 0 else error end' \
    <<<"${answer}" 2>/dev/null)" || return 1
  forge/github:fold 'the collaborators' "${answer}" --compact-output \
    'import "forge" as forge; forge::github_collaborators'
}

# forge/github:dependabot-alerts prints one document per Dependabot alert, open or dismissed, as the seam's header
# says: one REST read of both states, every page, by its last update. A repository with Dependabot alerts disabled
# has them off, and answers 3.
forge/github:dependabot-alerts() {
  local pages
  forge/github:alert-pages pages 'the Dependabot alerts' \
    'repos/{owner}/{repo}/dependabot/alerts?state=open,dismissed&sort=updated&direction=desc&per_page=100' \
    || return $?
  forge/github:fold 'the Dependabot alerts' "${pages}" --compact-output --argjson now "$(date +%s)" \
    'import "forge" as forge; forge::github_dependabot($now)'
}

# forge/github:deployments prints one document per deployment, newest first, as the seam's header says: one GraphQL
# request, a point, for the 50 created last, each with its latest status, and {"limited": 50} after them when GitHub
# holds more.
forge/github:deployments() {
  local repo query answer
  repo=$(forge/github:repo) || return 1
  query='query($owner: String!, $name: String!) {
    repository(owner: $owner, name: $name) {
      url
      deployments(first: 50, orderBy: {field: CREATED_AT, direction: DESC}) { totalCount nodes {
        databaseId environment ref { name } commitOid state createdAt creator { login } commit { url }
        latestStatus { state logUrl environmentUrl }
      } }
    }
  }'
  forge/github:ask answer 'the deployments' api graphql --field owner="${repo%%/*}" --field name="${repo#*/}" \
    --raw-field query="${query}" || return 1
  forge/github:fold 'the deployments' "${answer}" --exit-status '.data.repository.deployments.nodes | arrays' \
    >/dev/null || return 1
  forge/github:fold 'the deployments' "${answer}" --compact-output --argjson now "$(date +%s)" \
    'import "forge" as forge; forge::github_deployments($now; 50)'
}

# forge/github:environments prints one document per deployment environment, in GitHub's order, as the seam's header
# says: every page of the REST list, and for an environment with branch policies of its own, a request for them, as
# no list holds them. Those requests are 20 at most: the list stops before the 21st such environment, and
# {"limited": N} follows the N before it. An environment with more than 100 branch policies fails the read.
forge/github:environments() {
  local answer name policies branches='{}' encoded
  local -a custom=()
  forge/github:ask answer 'the environments' api --paginate --slurp 'repos/{owner}/{repo}/environments?per_page=100' \
    || return 1
  forge:whole 'the environments' "$(jq 'if all(.[]; type == "object") then map(.environments | length) | add // 0
    else error end' <<<"${answer}" 2>/dev/null)" || return 1
  list:read custom forge/github:fold 'the environments' "${answer}" --raw-output \
    '[.[].environments[]? | select(.deployment_branch_policy.custom_branch_policies == true) | .name][:20][]' \
    || return 1
  for name in "${custom[@]}"
  do
    [[ -n "${name}" ]] || continue
    encoded=$(jq --null-input --raw-output --arg s "${name}" '$s | @uri')
    forge/github:ask policies 'the environments' api \
      "repos/{owner}/{repo}/environments/${encoded}/deployment-branch-policies?per_page=100" || return 1
    if ! jq --exit-status '(.total_count // 0) <= (.branch_policies // [] | length)' <<<"${policies}" >/dev/null 2>&1
    then
      error "kit reads at most 100 branch policies of an environment (the environments could not be read)"
      return 1
    fi
    branches=$(forge/github:fold 'the environments' "${policies}" --compact-output --arg name "${name}" \
      --argjson branches "${branches}" \
      'import "forge" as forge; $branches + {($name): forge::github_branch_policies}') || return 1
  done
  forge/github:fold 'the environments' "${answer}" --compact-output --argjson branches "${branches}" \
    'import "forge" as forge; forge::github_environments($branches; 20)'
}

# forge/github:issue-add-sub makes the second issue a sub-issue of the first, as the issue page's Relationships do.
forge/github:issue-add-sub() {
  gh issue edit "$1" --add-sub-issue "$2" >/dev/null 2>&1
}

forge/github:issue-assign() {
  gh issue edit "$1" --add-assignee "$2" >/dev/null 2>&1
}

forge/github:issue-assignees() {
  local assignees
  forge/github:ask assignees "the assignees of #$1" issue view "$1" --json assignees \
    --jq '[.assignees[].login] | join(" ")' || return 1
  printf '%s\n' "${assignees}"
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
  local state
  forge/github:ask state "the state of #$1" issue view "$1" --json state --jq .state || return 1
  printf '%s\n' "${state}"
}

forge/github:issue-title() {
  local title
  forge/github:ask title "the title of #$1" issue view "$1" --json title --jq '.title' || return 1
  printf '%s\n' "${title}"
}

forge/github:issue-unassign() {
  gh issue edit "$1" --remove-assignee "$2" >/dev/null 2>&1
}

# forge/github:iterations prints one document per iteration, by start, as the seam's header says: those of the iteration
# fields of the projects linked to the repository, which GitHub keeps in its Projects, not in its Issues. A repository
# with no project linked has none, and prints nothing. One GraphQL request reads the first 20 projects and the first 50
# fields of each; more than that, and the read fails, as it would be partial. Projects need the read:project scope,
# which gh's default login lacks: GitHub answers INSUFFICIENT_SCOPES, and the failure names the command that adds it.
forge/github:iterations() {
  local repo answer query said out
  repo=$(forge/github:repo) || return 1
  query='query($owner: String!, $name: String!) {
    repository(owner: $owner, name: $name) {
      projectsV2(first: 20) { totalCount nodes { url fields(first: 50) { totalCount nodes {
        ... on ProjectV2IterationField {
          configuration {
            iterations { id title startDate duration }
            completedIterations { id title startDate duration }
          }
        }
      } } } }
    }
  }'
  file:temp said gh || return 1
  if ! answer=$(gh api graphql --field owner="${repo%%/*}" --field name="${repo#*/}" --raw-field query="${query}" \
    2>"${said}")
  then
    out=${answer}$'\n'$(<"${said}")
    if [[ "${out,,}" == *insufficient_scopes* || "${out,,}" == *'required scopes'* ]]
    then
      error "run: gh auth refresh --scopes read:project (the iterations could not be read)"
    else
      forge/github:limited 'the iterations' "${out}" || error "run: gh auth status (the iterations could not be read)"
    fi
    return 1
  fi
  forge:json "${answer}" || { error "run: gh auth status (the iterations could not be read)"; return 1; }
  forge/github:fold 'the iterations' "${answer}" --exit-status '.data.repository.projectsV2.nodes | arrays' \
    >/dev/null || return 1
  jq --exit-status '
    .data.repository.projectsV2 | (.totalCount // 0) <= (.nodes | length)
      and all(.nodes[]; (.fields.totalCount // 0) <= ((.fields.nodes // []) | length))' <<<"${answer}" >/dev/null \
    || { error "kit reads at most 20 projects and 50 fields of each (the iterations could not be read)"; return 1; }
  forge/github:fold 'the iterations' "${answer}" --compact-output 'import "forge" as forge; forge::github_iterations'
}

# gh issue develop names the branch <N>-<slug>, links it to the issue under Development, and pushes it.
forge/github:link-branch() {
  gh issue develop "$1" --base "$2" >/dev/null
}

# forge/github:marks [<dir>] reads, for each kind of the repository's data, a mark that changes when it does: issues,
# of the issue or pull request updated last, as GitHub lists them together; runs, of the ten newest workflow runs;
# releases, of the newest release; milestones, of the first 100 milestones, the page the read takes first, so an edit
# to a milestone past the first 100 goes unseen; deployments, of the 50 newest deployments, each of which a new status
# updates, whoever made it; environments, of the first 100 environments, with their protection; alerts, of the alert
# each scanner, code scanning, Dependabot, and secret scanning, updated last, in any state, which a new alert, a fix,
# and a dismissal each update, off for a scanner off here, and refused for one that refuses the caller. A mark is GitHub's
# ETag for the answer. With <dir>, each ETag is kept there and the next read sends it, so GitHub answers 304 Not
# Modified while nothing changed, which counts against no rate limit.
# gh names the repository from the checkout's remote, as {owner}/{repo}, which costs no request.
forge/github:marks() {
  local dir=${1:-} issues runs releases milestones deployments environments alerts='' kind mark
  issues=$(forge/github:mark 'repos/{owner}/{repo}/issues?state=all&sort=updated&direction=desc&per_page=1' \
    "${dir:+${dir}/marks-issues.etag}") || return 1
  runs=$(forge/github:mark 'repos/{owner}/{repo}/actions/runs?per_page=10' "${dir:+${dir}/marks-runs.etag}") \
    || return 1
  releases=$(forge/github:mark 'repos/{owner}/{repo}/releases?per_page=1' "${dir:+${dir}/marks-releases.etag}") \
    || return 1
  milestones=$(forge/github:mark 'repos/{owner}/{repo}/milestones?state=all&per_page=100' \
    "${dir:+${dir}/marks-milestones.etag}") || return 1
  deployments=$(forge/github:mark 'repos/{owner}/{repo}/deployments?per_page=50' \
    "${dir:+${dir}/marks-deployments.etag}") || return 1
  environments=$(forge/github:mark 'repos/{owner}/{repo}/environments?per_page=100' \
    "${dir:+${dir}/marks-environments.etag}") || return 1
  # A scanner's mark is of the alert it updated last, in any state; one off here is off, and one that refuses the
  # caller is refused, so the mark stays live for the scanners that answer.
  for kind in code-scanning dependabot secret-scanning
  do
    mark=$(forge/github:mark "repos/{owner}/{repo}/${kind}/alerts?sort=updated&direction=desc&per_page=1" \
      "${dir:+${dir}/marks-${kind}.etag}" forge/github:scanner-mark) || return 1
    alerts+=${alerts:+ }${mark}
  done
  jq --null-input --compact-output --arg alerts "${alerts}" --arg issues "${issues}" --arg runs "${runs}" \
    --arg releases "${releases}" \
    --arg milestones "${milestones}" --arg deployments "${deployments}" --arg environments "${environments}" \
    '{issues: $issues, runs: $runs, releases: $releases, milestones: $milestones, deployments: $deployments,
      environments: $environments, alerts: $alerts}'
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
    || error "run: gh auth login (the caller's login could not be read)"
  return 1
}

# forge/github:merged-prs prints one document per merged pull request, as the seam's header says: one GraphQL request,
# a point of the rate limit, for the 50 updated last. GitHub orders a list by its creation or its update, never by
# its merge, and a merge updates a pull request, so every pull request merged since the oldest update among them is
# there; forge.jq orders them by merge. When GitHub holds more, one merged before that update may be missing, so
# forge.jq keeps only those merged since, and ends the list with {"limited": N}, N those it kept.
forge/github:merged-prs() {
  local repo query answer
  repo=$(forge/github:repo) || return 1
  query='query($owner: String!, $name: String!) {
    repository(owner: $owner, name: $name) {
      pullRequests(states: MERGED, first: 50, orderBy: {field: UPDATED_AT, direction: DESC}) {
        totalCount nodes { number title url headRefName mergedAt updatedAt author { login } mergedBy { login } }
      }
    }
  }'
  forge/github:ask answer 'the merged pull requests' api graphql --field owner="${repo%%/*}" \
    --field name="${repo#*/}" --raw-field query="${query}" || return 1
  forge/github:fold 'the merged pull requests' "${answer}" --exit-status \
    '.data.repository.pullRequests.nodes | arrays' >/dev/null || return 1
  forge/github:fold 'the merged pull requests' "${answer}" --compact-output \
    'import "forge" as forge; forge::github_merged'
}

# forge/github:milestones prints one document per milestone, open and closed, by number, as the seam's header says,
# every page of the REST list: gh api --paginate --slurp joins the pages into one array of them.
forge/github:milestones() {
  local answer
  forge/github:ask answer 'the milestones' api --paginate --slurp \
    'repos/{owner}/{repo}/milestones?state=all&per_page=100' || return 1
  forge:whole 'the milestones' "$(jq 'if all(.[]; type == "array") then map(length) | add // 0 else error end' \
    <<<"${answer}" 2>/dev/null)" || return 1
  forge/github:fold 'the milestones' "${answer}" --compact-output 'import "forge" as forge; forge::github_milestones'
}

forge/github:noun() { printf '#\tpull request\n'; }

# forge/github:open-issues prints one document per open issue, oldest first, as the seam's header says. gh issue list
# answers with issues alone, never pull requests, and reads every page up to its --limit.
forge/github:open-issues() {
  local answer
  forge/github:ask answer 'the issues' issue list --state open --limit "${forge_list_limit}" \
    --json number,title,assignees,labels,createdAt,url \
    || return 1
  forge:whole 'the issues' "$(jq length <<<"${answer}")" || return 1
  forge/github:fold 'the issues' "${answer}" --compact-output --argjson now "$(date +%s)" \
    'import "forge" as forge; forge::github_issues($now)'
}

# forge/github:open-prs makes one GraphQL request for every open pull request, with its labels, the issues it closes
# and theirs, the reviewers it asks and each reviewer's latest review, its review threads, the owner of the repository
# its branch lives in, and its head commit: when it was committed, by whom, and the checks on it. It prints one JSON
# object per pull request: the document kit status renders, which modules/jq/forge.jq folds the answer into. Of
# several runs of one name on the head commit, as after a rerun, only the latest counts.
#
# GitHub charges a request by the nodes it may return, so assignees asked of each issue each pull request closes cost
# a hundred times as much as the rest of the request. The same request asks instead for the assignees of the hundred
# open issues with an assignee most recently updated, and forge.jq gives each closed issue its assignees from them.
# Labels are asked under each closed issue all the same, as no other list holds them all, and a pull request is
# asked for the first 10 issues it closes, not 100, which keeps them cheap. GitHub's dry run, rateLimit(dryRun: true),
# puts the request at 18 points of the GraphQL rate limit, the reviewers asked and their reviews a point each of
# them; without any labels, it costs 7, and with the labels under the first 100 issues each closes, 108. A pull
# request that closes more than 10 says so, as its list's next page, and forge/github:rest-of-closes reads the rest of
# it, 100 at a time, a point each. A pull request is asked for its first 20 reviewers asked and 20 reviews, and one
# with more fails the read, as it would be partial. A team asked is named by its slug, which may need the read:org
# scope that gh's default login has and a GH_TOKEN may lack. GitHub then refuses the field, and the read asks again
# without it, once: the open changes are drawn, and a team asked is not among their reviewers.
#
# The pull requests come 100 a request, each page after the cursor the one before ended on, until the list ends. A
# pull request with more than 100 review threads, or more than 100 checks on its head commit, has the rest of that
# list read by forge/github:rest-of-pr, 100 a request, as the issues it closes are read by forge/github:rest-of-closes.
forge/github:open-prs() {
  local repo query answer='' page number cursor rest list head row said team='... on Team { slug } '
  local -a after=() rests=()
  repo=$(forge/github:repo) || return 1
  query='query($owner: String!, $name: String!, $after: String) {
    repository(owner: $owner, name: $name) {
      nameWithOwner
      issues(states: OPEN, first: 100, filterBy: {assignee: "*"}, orderBy: {field: UPDATED_AT, direction: DESC}) {
        nodes { number assignees(first: 10) { nodes { login } } }
      }
      pullRequests(states: OPEN, first: 100, after: $after, orderBy: {field: CREATED_AT, direction: ASC}) {
        pageInfo { hasNextPage endCursor }
        nodes {
          number title isDraft headRefName createdAt mergeStateStatus reviewDecision isCrossRepository
          author { login }
          headRepositoryOwner { login }
          labels(first: 20) { nodes { name color } }
          closingIssuesReferences(first: 10) { pageInfo { hasNextPage endCursor } nodes {
            number title createdAt repository { nameWithOwner } labels(first: 20) { nodes { name color } }
          } }
          reviewRequests(first: 20) { totalCount nodes { requestedReviewer {
            ... on User { login } @team@... on Bot { login } ... on Mannequin { login }
          } } }
          latestReviews(first: 20) { totalCount nodes { author { login } state submittedAt } }
          reviewThreads(first: 100) { pageInfo { hasNextPage endCursor } nodes { isResolved } }
          commits(last: 1) { nodes { commit {
            oid committedDate author { user { login } }
            statusCheckRollup { contexts(first: 100) { pageInfo { hasNextPage endCursor } nodes {
              __typename
              ... on CheckRun { name status conclusion startedAt completedAt }
              ... on StatusContext { context state createdAt }
            } } }
          } } }
        }
      }
    }
  }'
  file:temp said gh || return 1
  while :
  do
    if ! forge/github:ask page 'the pull requests' api graphql --field owner="${repo%%/*}" --field name="${repo#*/}" \
      "${after[@]}" --raw-field query="${query//@team@/${team}}" 2>"${said}"
    then
      if [[ -n "${team}" ]] && forge/github:teamless "${forge_github_said:-}"
      then
        team=''
        continue
      fi
      cat "${said}" >&2
      return 1
    fi
    if [[ -z "${answer}" ]]
    then
      answer=${page}
    else
      answer=$(forge/github:fold 'the pull requests' "${answer}"$'\n'"${page}" --slurp --compact-output \
        '.[0].data.repository.pullRequests.nodes += .[1].data.repository.pullRequests.nodes | .[0]') || return 1
    fi
    forge:whole 'the pull requests' "$(jq '.data.repository.pullRequests.nodes | length' <<<"${answer}")" || return 1
    cursor=$(forge/github:fold 'the pull requests' "${page}" --raw-output \
      '.data.repository.pullRequests.pageInfo // {} | select(.hasNextPage == true) | .endCursor') || return 1
    [[ -n "${cursor}" ]] || break
    after=(--raw-field after="${cursor}")
  done
  # An answer with no pull requests to read, as one with no repository, is no cap reached: it could not be read.
  forge/github:fold 'the pull requests' "${answer}" --exit-status '.data.repository.pullRequests.nodes | arrays' \
    >/dev/null || return 1
  jq --exit-status 'all(.data.repository.pullRequests.nodes[]?;
      (.reviewRequests.totalCount // 0) <= (.reviewRequests.nodes // [] | length)
      and (.latestReviews.totalCount // 0) <= (.latestReviews.nodes // [] | length))' <<<"${answer}" >/dev/null \
    || { error "kit reads at most 20 review requests and 20 reviews of a change (the pull requests could not be read)"
         return 1; }
  list:read rests forge/github:fold 'the pull requests' "${answer}" --raw-output \
    'import "forge" as forge; forge::github_partial' || return 1
  for row in "${rests[@]}"
  do
    IFS=$'\t' read -r number list cursor head <<<"${row}"
    rest=$(forge/github:rest-of-pr "${repo}" "${number}" "${list}" "${cursor}" "${head}") || return 1
    answer=$(forge/github:fold 'the pull requests' "${answer}" --compact-output --argjson number "${number}" \
      --arg list "${list}" --argjson rest "${rest}" '.data.repository.pullRequests.nodes
        |= map(if .number != $number then .
               elif $list == "threads" then .reviewThreads.nodes += $rest
               else .commits.nodes[0].commit.statusCheckRollup.contexts.nodes += $rest end)') || return 1
  done
  while IFS=$'\t' read -r number cursor
  do
    [[ -n "${number}" ]] || continue
    rest=$(forge/github:rest-of-closes "${repo}" "${number}" "${cursor}") || return 1
    answer=$(forge/github:fold 'the pull requests' "${answer}" --compact-output --argjson number "${number}" \
      --argjson rest "${rest}" '.data.repository.pullRequests.nodes
        |= map(if .number == $number then .closingIssuesReferences.nodes += $rest else . end)') || return 1
  done < <(jq --raw-output '.data.repository.pullRequests.nodes[]
    | select(.closingIssuesReferences.pageInfo.hasNextPage) | "\(.number)\t\(.closingIssuesReferences.pageInfo.endCursor)"' \
    <<<"${answer}")
  forge/github:fold 'the pull requests' "${answer}" --compact-output --argjson now "$(date +%s)" \
    'import "forge" as forge; forge::github_prs($now)'
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
  local body
  forge/github:ask body "the body of pull request #$1" pr view "$1" --json body --jq '.body // ""' || return 1
  printf '%s\n' "${body}"
}

# forge/github:pr-closing-issues reads every issue the pull request closes, 100 a request, each page after the cursor
# the one before ended on, until the list ends. A pull request GitHub does not have fails the read.
forge/github:pr-closing-issues() {
  local repo query page all='[]' more=true cursor
  local -a after=()
  repo=$(forge/github:repo) || return 1
  query='query($owner: String!, $name: String!, $number: Int!, $after: String) {
    repository(owner: $owner, name: $name) {
      pullRequest(number: $number) {
        closingIssuesReferences(first: 100, after: $after) { pageInfo { hasNextPage endCursor } nodes {
          number state repository { nameWithOwner }
        } }
      }
    }
  }'
  while [[ "${more}" == true ]]
  do
    forge/github:ask page "pull request #$1" api graphql --field owner="${repo%%/*}" --field name="${repo#*/}" \
      --field number="$1" "${after[@]}" --raw-field query="${query}" || return 1
    all=$(forge/github:fold "pull request #$1" "${all}"$'\n'"${page}" --slurp --compact-output \
      '.[0] + (.[1].data.repository.pullRequest // error("no such pull request")).closingIssuesReferences.nodes') \
      || return 1
    forge:whole "the issues pull request #$1 closes" "$(jq length <<<"${all}")" || return 1
    more=$(forge/github:fold "pull request #$1" "${page}" --raw-output \
      '.data.repository.pullRequest.closingIssuesReferences.pageInfo.hasNextPage == true') || return 1
    cursor=$(forge/github:fold "pull request #$1" "${page}" --raw-output \
      '.data.repository.pullRequest.closingIssuesReferences.pageInfo.endCursor // ""') || return 1
    after=(--raw-field after="${cursor}")
  done
  forge/github:fold "pull request #$1" "${all}" --raw-output '.[] | "\(.number) \(.state) \(.repository.nameWithOwner)"'
}

forge/github:pr-create() {
  local url
  local -a draft=(--draft)
  [[ "${4:-true}" != false ]] || draft=()
  url=$(gh pr create "${draft[@]}" --base "$2" --head "$1" --title "$3" --body "") || return 1
  printf '%s\n' "${url##*/}"
}

forge/github:pr-for-branch() {
  forge/github:on-branch "the open pull request on $1" "$1" open number
}

forge/github:pr-head-for-branch() {
  forge/github:on-branch "the pull requests on $1" "$1" all headRefOid
}

# On a GitHub Actions runner, the event payload gives the pull request. Otherwise it is the open pull request for the
# current branch; a detached HEAD has none.
forge/github:pr-here() {
  local number='' branch
  if [[ -n "${GITHUB_EVENT_PATH:-}" ]]
  then
    # A payload with no pull request, or none that can be read, gives none, and the branch is asked about.
    number=$(jq --raw-output '.pull_request.number // empty' "${GITHUB_EVENT_PATH}" 2>/dev/null || true)
  fi
  if [[ -z "${number}" ]]
  then
    branch=$(git symbolic-ref --quiet --short HEAD 2>/dev/null || true)  # a detached HEAD has no pull request
    [[ -z "${branch}" ]] || number=$(forge/github:pr-for-branch "${branch}") || return 1
  fi
  [[ -z "${number}" ]] || printf '%s\n' "${number}"
}

forge/github:pr-label-add() {
  gh pr edit "$1" --add-label "$2" >/dev/null 2>&1
}

forge/github:pr-label-remove() {
  gh pr edit "$1" --remove-label "$2" >/dev/null 2>&1
}

forge/github:pr-last-for-branch() {
  forge/github:on-branch "the pull requests on $1" "$1" all number
}

forge/github:pr-ready() {
  gh pr ready "$1" >/dev/null 2>&1
}

forge/github:pr-set-body() {
  gh pr edit "$1" --body "$2" >/dev/null
}

forge/github:pr-set-title() {
  gh pr edit "$1" --title "$2" >/dev/null 2>&1
}

forge/github:pr-state() {
  local state
  forge/github:ask state "pull request #$1" pr view "$1" --json state,mergeCommit \
    --jq '"\(.state)\t\(.mergeCommit.oid // "")"' || return 1
  printf '%s\n' "${state}"
}

forge/github:pr-state-for-branch() {
  forge/github:on-branch "the pull requests on $1" "$1" all state
}

forge/github:pr-title() {
  local title
  forge/github:ask title "the title of pull request #$1" pr view "$1" --json title --jq '.title' || return 1
  printf '%s\n' "${title}"
}

forge/github:pr-url() {
  local url
  url=$(forge/github:repo-url) || return 1
  printf '%s/pull/%s\n' "${url}" "$1"
}

# forge/github:quota reads the rate_limit endpoint, which no limit counts, and keeps the two kit spends: GraphQL's
# points and REST's requests.
forge/github:quota() {
  local limits
  forge/github:ask limits 'the rate limits' api rate_limit || return 1
  forge/github:fold 'the rate limits' "${limits}" --compact-output 'import "forge" as forge; forge::github_quota'
}

# forge/github:releases prints one document per release, newest first, as the seam's header says. gh release list
# gives no release's notes or assets, so the releases are read from the REST list, which has both, every page of it:
# gh api --paginate prints each page's array after the one before.
forge/github:releases() {
  local answer
  forge/github:ask answer 'the releases' api --paginate 'repos/{owner}/{repo}/releases?per_page=100' || return 1
  forge:whole 'the releases' "$(jq --slurp 'map(length) | add // 0' <<<"${answer}" 2>/dev/null)" || return 1
  forge/github:fold 'the releases' "${answer}" --slurp --compact-output --argjson now "$(date +%s)" \
    'import "forge" as forge; forge::github_releases($now)'
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

# forge/github:rulesets prints one document per control on the default branch, as the seam's header says: the
# rulesets that guard it, its own and those of its organization, read as forge/github:controls reads them, then the
# branch's own protection, where it has one.
forge/github:rulesets() {
  local answer protection
  forge/github:controls answer protection || return 1
  forge/github:fold 'the rulesets' "${answer}" --compact-output --argjson protection "${protection}" \
    'import "forge" as forge; forge::github_rulesets($protection | if type == "object" then {rules} else null end)'
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
  runs=$(forge/github:fold 'the runs' "${answer}" --compact-output --argjson now "$(date +%s)" \
    --arg default "${default}" --argjson tags "${tags}" \
    'import "forge" as forge; forge::github_builds($now; $default; $tags)') || return 1
  while IFS= read -r id
  do
    # The run's own state stands whatever its jobs say, and the jobs are a detail of the picture, so a run whose jobs
    # cannot be read is drawn with none.
    jobs=$(gh run view "${id}" --json jobs 2>/dev/null | jq -L "${kit_jq}" --compact-output \
      'import "forge" as forge; forge::github_jobs' 2>/dev/null) || jobs='[]'
    runs=$(forge/github:fold 'the runs' "${runs}" --compact-output --argjson index "${index}" \
      --argjson jobs "${jobs:-[]}" '.[$index].jobs = $jobs') || return 1
    index=$(( index + 1 ))
  done < <(jq --raw-output '.[:5][].id' <<<"${runs}")
  printf '%s\n' "${runs}"
}

# forge/github:secret-scanning-alerts prints one document per secret scanning alert, open or dismissed, as the seam's
# header says: two REST reads, every page of each, by its last update, the open alerts and those resolved as a false
# positive, as won't fix, or as used in tests, which a person dismissed; one revoked, or whose pattern went, is fixed.
# A repository with secret scanning disabled has it off, and answers 3.
forge/github:secret-scanning-alerts() {
  local pages alerts='sort=updated&direction=desc&per_page=100'
  local waived='resolution=false_positive,wont_fix,used_in_tests'
  forge/github:alert-pages pages 'the secret scanning alerts' \
    "repos/{owner}/{repo}/secret-scanning/alerts?state=open&${alerts}" \
    "repos/{owner}/{repo}/secret-scanning/alerts?state=resolved&${waived}&${alerts}" || return $?
  forge/github:fold 'the secret scanning alerts' "${pages}" --compact-output --argjson now "$(date +%s)" \
    'import "forge" as forge; forge::github_secret_scanning($now)'
}

# forge/github:teams prints one document per team with access to the repository, in GitHub's order, as the seam's
# header says: a REST request for the first 21, then one GraphQL request, a point, for the first 100 members of each of
# the first 20, of the organization that owns the repository. A 21st team, or a team past its first 100 members, ends
# the list with {"limited": N}. A repository a user owns has no teams, and asks no more. GitHub answers some who can
# reach the repository 404 for its teams, and its reference says not whom, so the failure names an admin and no role.
# It shows the members only to a login with the read:org scope, which a GH_TOKEN may lack: GitHub answers
# INSUFFICIENT_SCOPES, and the failure names the command that adds it. A team with no slug fails the read, as the
# members of the teams after it could not be told apart.
forge/github:teams() {
  local repo answer slug page members='{}' variables='' fields=''
  local -i index=0
  local -a slugs named=()
  repo=$(forge/github:repo) || return 1
  forge/github:refused answer 'the teams' '*(HTTP 404)*' \
    "GitHub shows a repository's teams only to some who can reach it; ask an admin of ${repo}" \
    api 'repos/{owner}/{repo}/teams?per_page=21' || return 1
  list:read slugs forge/github:fold 'the teams' "${answer}" --raw-output \
    '.[:20][] | .slug | if type == "string" and . != "" then . else error("a team with no slug") end' || return 1
  for slug in "${slugs[@]}"
  do
    variables+=", \$t${index}: String!"
    fields+=" t${index}: team(slug: \$t${index}) { members(first: 100) { totalCount nodes { login } } }"
    named+=(--raw-field "t${index}=${slug}")
    index+=1
  done
  if (( index > 0 ))
  then
    forge/github:ask page 'the teams' api graphql --raw-field org="${repo%%/*}" "${named[@]}" \
      --raw-field query="query(\$org: String!${variables}) { organization(login: \$org) {${fields} } }" || return 1
    members=$(forge/github:fold 'the teams' "${page}" --compact-output \
      '(.data | objects // error) | .organization // {}') || return 1
  fi
  forge/github:fold 'the teams' "${answer}" --compact-output --argjson members "${members}" \
    'import "forge" as forge; forge::github_teams($members; 20)'
}

# The provider implements contract 4, and its forge lacks none of its features.
forge/github:contract() { printf '4\n'; }

forge/github:lacks() { :; }

# The seam's calls end here; what follows is this provider's own.

# forge/github:on-branch <what> <branch> <state> <field> prints the field of the newest pull request on the branch,
# among those open or all, or nothing when there is none.
forge/github:on-branch() {
  local found
  forge/github:ask found "$1" pr list --head "$2" --state "$3" --json "$4" --jq ".[0].$4 // empty" || return 1
  [[ -z "${found}" ]] || printf '%s\n' "${found}"
}

# forge/github:mark <path> [<file>] [<refused>] prints the ETag of GitHub's answer for <path>: the one kept in <file>
# when GitHub answers 304 Not Modified to it, else the new one, which it keeps there. gh exits 1 on a 304, so the
# answer's status line says what came back. With <refused>, a function, a 403 or 404 is no failure: the mark is what
# <refused> prints of the answer. An answer that is none of those fails.
forge/github:mark() {
  local path=$1 file=${2:-} refused=${3:-} etag='' answer status
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
        error "run: gh auth status; GitHub gave no ETag for ${path} (the marks could not be read)"
        return 1
      fi
      # An ETag that cannot be kept costs only the next read, which asks GitHub for the whole answer.
      [[ -z "${file}" ]] || file:replace "${file}" printf '%s\n' "${etag}" || true
      printf '%s\n' "${etag}"
      ;;
    (403|404)
      if [[ -n "${refused}" ]]
      then
        "${refused}" "${answer}"
        return
      fi
      error "run: gh auth status (the marks could not be read)"
      return 1
      ;;
    (*)
      error "run: gh auth status (the marks could not be read)"
      return 1
  esac
}

# forge/github:scanner-mark <answer> is forge/github:mark's mark for a scanner's alerts that GitHub refused: off where
# it says the scanner is off for the repository, as forge/github:off tells it, else refused, as the caller may not
# see them; either holds while nothing changes.
forge/github:scanner-mark() {
  if forge/github:off "$1"
  then
    printf 'off\n'
  else
    printf 'refused\n'
  fi
}

# forge/github:off <said> succeeds when what GitHub said of a scanner's alerts is that the scanner is off for the
# repository: code scanning with no analysis, no default branch, or Code Security, once Advanced Security, not
# enabled; Dependabot alerts or secret scanning disabled; or a repository archived.
forge/github:off() {
  local said=${1,,}
  [[ "${said}" == *'no analysis found'* || "${said}" == *'no default branch found'* \
    || "${said}" == *'must be enabled'* || "${said}" == *' disabled '* || "${said}" == *'disabled for'* \
    || "${said}" == *'archived'* ]]
}

# forge/github:alert-pages <name> <what> <path>… reads every page of each path, a scanner's alerts, and puts them, one
# JSON array of the pages, each an array, in the variable <name>. It returns 3, printing nothing, where GitHub says
# the scanner is off for the repository, and 1 after the line forge/github:alerts-refused says where GitHub refuses
# the read.
forge/github:alert-pages() {
  local pages_into=$1 pages_what=$2 pages_path pages_said pages_out pages_all='[]'
  shift 2
  file:temp pages_said gh || return 1
  for pages_path in "$@"
  do
    if ! pages_out=$(gh api --paginate --slurp "${pages_path}" 2>"${pages_said}")
    then
      forge/github:alerts-refused "${pages_what}" "${pages_path}" "${pages_out}"$'\n'"$(<"${pages_said}")"
      return
    fi
    pages_all=$(forge/github:fold "${pages_what}" "${pages_all}"$'\n'"${pages_out}" --slurp --compact-output \
      'if all(.[1][]; type == "array") then .[0] + .[1] else error end') || return 1
  done
  forge:whole "${pages_what}" "$(jq 'map(length) | add // 0' <<<"${pages_all}")" || return 1
  printf -v "${pages_into}" '%s' "${pages_all}"
}

# forge/github:alerts-refused <what> <path> <said> says why GitHub refused the read of a scanner's alerts at <path>,
# and returns 1, or returns 3, saying nothing, where it says the scanner is off for the repository. A rate limit says
# which, as forge/github:limited does. Any other refusal, 403 or 404, is asked once more for its headers: a login whose scopes, X-OAuth-Scopes, hold none of those the alerts
# accept, X-Accepted-OAuth-Scopes, is told the scope to add; a token whose scopes GitHub does not say, as a
# fine-grained one, that GitHub says cannot reach the alerts, is told to grant it; any other caller may not see them,
# and is told to ask the repository's admin.
forge/github:alerts-refused() {
  local what=$1 path=$2 said=$3 head granted accepted scope repo
  if forge/github:off "${said}"
  then
    return 3
  fi
  # A rate limit, primary or secondary, answers 403 too, and says so.
  if forge/github:limited "${what}" "${said}"
  then
    return 1
  fi
  if [[ "${said}" != *'(HTTP 403)'* && "${said}" != *'(HTTP 404)'* ]]
  then
    forge/github:scoped "${what}" "${said}" || error "run: gh auth status (${what} could not be read)"
    return 1
  fi
  head=$(gh api --include "${path}" 2>/dev/null | tr -d '\r' | sed '/^$/q') || true  # a refusal exits 1
  granted=$(sed -n 's/^[Xx]-[Oo][Aa]uth-[Ss]copes: *//p' <<<"${head}")
  accepted=$(sed -n 's/^[Xx]-[Aa]ccepted-[Oo][Aa]uth-[Ss]copes: *//p' <<<"${head}")
  if grep --quiet --ignore-case '^x-oauth-scopes:' <<<"${head}" && [[ -n "${accepted}" ]]
  then
    for scope in ${accepted//,/ }
    do
      [[ ", ${granted}, " != *", ${scope}, "* ]] || accepted=''
    done
    if [[ -n "${accepted}" ]]
    then
      error "run: gh auth refresh --scopes security_events (${what} could not be read)"
      return 1
    fi
  fi
  if [[ "${said,,}" == *'not accessible by'* ]]
  then
    error "grant the token GitHub was given read access to the repository's security alerts (${what} could not be" \
      "read)"
    return 1
  fi
  repo=$(forge/github:repo) || return 1
  error "ask an admin of ${repo} to let you see its security alerts (${what} could not be read)"
  return 1
}

# forge/github:controls <answer> <protection> reads what guards the default branch into the two variables: <answer>,
# one GraphQL request, a point, for the repository's rulesets of branches, its own and its organization's, 100 at
# most, each with its conditions, its first 100 rules, and its first 100 bypass actors, and the default branch, asked
# once more without a team's slug where the login lacks read:org, as forge/github:teamless tells it; and
# <protection>, null where the branch has no protection of its own, as the branch's REST answer says, which every
# reader may have, else {rules, full}: the protection's REST answer and its kinds of rule, or rules and full null
# where GitHub shows the protection only to an admin, answering 404. A list past what the request holds fails.
forge/github:controls() {
  local controls_into=$1 protection_into=$2 controls_repo controls_query controls_answer controls_branch
  local controls_page controls_said controls_protection=null encoded team='... on Team { slug }'
  controls_repo=$(forge/github:repo) || return 1
  controls_query='query($owner: String!, $name: String!) {
    repository(owner: $owner, name: $name) {
      url defaultBranchRef { name }
      rulesets(first: 100, includeParents: true, targets: [BRANCH]) { totalCount nodes {
        databaseId name enforcement conditions { refName { include exclude } }
        rules(first: 100) { totalCount nodes { type } }
        bypassActors(first: 100) { totalCount nodes {
          bypassMode organizationAdmin repositoryRoleName deployKey enterpriseOwner
          actor { __typename @team@ ... on User { login } ... on App { slug } }
        } }
      } }
    }
  }'
  file:temp controls_said gh || return 1
  # A login without read:org is refused a team's slug, and is asked once more without it: those teams go unnamed.
  if ! forge/github:ask controls_answer 'the rulesets' api graphql --field owner="${controls_repo%%/*}" \
    --field name="${controls_repo#*/}" --raw-field query="${controls_query//@team@/${team}}" 2>"${controls_said}"
  then
    if ! forge/github:teamless "${forge_github_said:-}"
    then
      cat "${controls_said}" >&2
      return 1
    fi
    forge/github:ask controls_answer 'the rulesets' api graphql --field owner="${controls_repo%%/*}" \
      --field name="${controls_repo#*/}" --raw-field query="${controls_query//@team@/}" || return 1
  fi
  forge/github:fold 'the rulesets' "${controls_answer}" --exit-status '.data.repository.rulesets.nodes | arrays' \
    >/dev/null || return 1
  if ! jq --exit-status -L "${kit_jq}" 'import "forge" as forge; forge::github_whole' <<<"${controls_answer}" \
    >/dev/null 2>&1
  then
    error "kit reads at most 100 rulesets, and 100 rules and bypass actors of each (the rulesets could not be read)"
    return 1
  fi
  controls_branch=$(jq --raw-output '.data.repository.defaultBranchRef.name // empty' <<<"${controls_answer}")
  if [[ -n "${controls_branch}" ]]
  then
    encoded=$(jq --null-input --raw-output --arg s "${controls_branch}" '$s | @uri')
    forge/github:ask controls_page 'the protection of the default branch' api \
      "repos/{owner}/{repo}/branches/${encoded}" || return 1
    if [[ "$(jq --raw-output '.protection.enabled == true' <<<"${controls_page}")" == true ]]
    then
      file:temp controls_said gh || return 1
      if controls_page=$(gh api "repos/{owner}/{repo}/branches/${encoded}/protection" 2>"${controls_said}")
      then
        controls_protection=$(forge/github:fold 'the protection of the default branch' "${controls_page}" \
          --compact-output 'import "forge" as forge; {rules: forge::github_protection_rules, full: .}') || return 1
      elif [[ "$(<"${controls_said}")" == *'(HTTP 404)'* ]]
      then
        controls_protection='{"rules": null, "full": null}'
      else
        forge/github:limited 'the protection of the default branch' "$(<"${controls_said}")" \
          || error "run: gh auth status (the protection of the default branch could not be read)"
        return 1
      fi
    fi
  fi
  printf -v "${controls_into}" '%s' "${controls_answer}"
  printf -v "${protection_into}" '%s' "${controls_protection}"
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
    all=$(forge/github:fold "the rest of the issues #${number} closes" "${all}"$'\n'"${page}" --slurp \
      --compact-output '.[0] + .[1].data.repository.pullRequest.closingIssuesReferences.nodes') || return 1
    more=$(jq --raw-output '.data.repository.pullRequest.closingIssuesReferences.pageInfo.hasNextPage' <<<"${page}")
    cursor=$(jq --raw-output '.data.repository.pullRequest.closingIssuesReferences.pageInfo.endCursor // ""' <<<"${page}")
  done
  printf '%s\n' "${all}"
}

# forge/github:rest-of-pr <owner/name> <number> <list> <cursor> <commit> prints, as one JSON array, the rest of a
# pull request's review threads (<list> threads) or of the checks on its head commit (<list> checks) after the cursor
# its first page ended on, each as forge/github:open-prs asks for one, 100 a request, until the list ends. <commit> is
# the head commit's oid as that first page gave it: a head commit that moved since fails the read, since its checks
# are another commit's.
forge/github:rest-of-pr() {
  local repo=$1 number=$2 list=$3 cursor=$4 head=$5 query path what page moved all='[]' more=true
  if [[ "${list}" == threads ]]
  then
    what="the review threads of #${number}"
    path='.data.repository.pullRequest.reviewThreads'
    query='query($owner: String!, $name: String!, $pr: Int!, $after: String) {
      repository(owner: $owner, name: $name) { pullRequest(number: $pr) {
        reviewThreads(first: 100, after: $after) { pageInfo { hasNextPage endCursor } nodes { isResolved } }
      } }
    }'
  else
    what="the checks of #${number}"
    path='.data.repository.pullRequest.commits.nodes[0].commit.statusCheckRollup.contexts'
    query='query($owner: String!, $name: String!, $pr: Int!, $after: String) {
      repository(owner: $owner, name: $name) { pullRequest(number: $pr) {
        commits(last: 1) { nodes { commit { oid statusCheckRollup {
          contexts(first: 100, after: $after) { pageInfo { hasNextPage endCursor } nodes {
            __typename
            ... on CheckRun { name status conclusion startedAt completedAt }
            ... on StatusContext { context state createdAt }
          } }
        } } } }
      } }
    }'
  fi
  while [[ "${more}" == true ]]
  do
    forge/github:ask page "${what}" api graphql --field owner="${repo%%/*}" --field name="${repo#*/}" \
      --field pr="${number}" --raw-field after="${cursor}" --raw-field query="${query}" || return 1
    if [[ "${list}" == checks ]]
    then
      moved=$(forge/github:fold "${what}" "${page}" --raw-output --arg head "${head}" \
        '.data.repository.pullRequest.commits.nodes[0].commit.oid != $head') || return 1
      if [[ "${moved}" != false ]]
      then
        error "read them again: #${number}'s head commit moved while kit read its checks (${what} could not be read)"
        return 1
      fi
    fi
    all=$(forge/github:fold "${what}" "${all}"$'\n'"${page}" --slurp --compact-output \
      ".[0] + (.[1] | ${path}.nodes)") || return 1
    forge:whole "${what}" "$(jq length <<<"${all}")" || return 1
    more=$(forge/github:fold "${what}" "${page}" --raw-output "${path}.pageInfo.hasNextPage == true") || return 1
    cursor=$(forge/github:fold "${what}" "${page}" --raw-output "${path}.pageInfo.endCursor // \"\"") || return 1
  done
  printf '%s\n' "${all}"
}

# forge/github:ask <name> <what> <gh argument>… runs gh and puts what it printed in the variable <name>. When gh
# fails, it keeps what gh said in forge_github_said, and says why, as forge/github:limited or forge/github:scoped
# does, or else names gh auth status, for <what>, and returns 1. An answer gh is not asked to filter with --jq must be
# JSON, or the read fails; gh parses the one it filters itself.
forge/github:ask() {
  local ask_into=$1 ask_what=$2 ask_said ask_out ask_arg ask_filtered=false
  shift 2
  file:temp ask_said gh || return 1
  if ! ask_out=$(gh "$@" 2>"${ask_said}")
  then
    forge_github_said=$(<"${ask_said}")
    forge/github:limited "${ask_what}" "${ask_out}"$'\n'"$(<"${ask_said}")" \
      || forge/github:scoped "${ask_what}" "$(<"${ask_said}")" \
      || error "run: gh auth status (${ask_what} could not be read)"
    return 1
  fi
  for ask_arg in "$@"
  do
    [[ "${ask_arg}" != --jq ]] || ask_filtered=true
  done
  if [[ "${ask_filtered}" == false ]] && ! forge:json "${ask_out}"
  then
    error "run: gh auth status (${ask_what} could not be read)"
    return 1
  fi
  printf -v "${ask_into}" '%s' "${ask_out}"
}

# forge/github:refused <name> <what> <said> <line> <gh argument>… is forge/github:ask for a list, an array, that
# GitHub may refuse a caller in words of its own: when what gh said matches the glob <said>, in any case, the failure
# says <line>, then that <what> could not be read.
forge/github:refused() {
  local refused_into=$1 refused_what=$2 refused_glob=$3 refused_line=$4 refused_said refused_out
  shift 4
  file:temp refused_said gh || return 1
  if ! refused_out=$(gh "$@" 2>"${refused_said}")
  then
    refused_out=${refused_out}$'\n'$(<"${refused_said}")
    # shellcheck disable=SC2053  # the glob is the caller's, matched as a glob on purpose
    if [[ "${refused_out,,}" == ${refused_glob,,} ]]
    then
      error "${refused_line} (${refused_what} could not be read)"
    else
      forge/github:limited "${refused_what}" "${refused_out}" \
        || forge/github:scoped "${refused_what}" "${refused_out}" \
        || error "run: gh auth status (${refused_what} could not be read)"
    fi
    return 1
  fi
  if ! forge:json "${refused_out}" || ! jq --exit-status 'arrays' <<<"${refused_out}" >/dev/null 2>&1
  then
    error "run: gh auth status (${refused_what} could not be read)"
    return 1
  fi
  printf -v "${refused_into}" '%s' "${refused_out}"
}

# forge/github:fold <what> <answer> <jq argument>… is forge:fold, naming gh auth status.
forge/github:fold() { forge:fold 'run: gh auth status' "$@"; }

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
  wait=$(jq -L "${kit_jq}" --raw-output --argjson now "$(date +%s)" \
    'import "forge" as forge; forge::github_spent($now)' <<<"${limits}" 2>/dev/null) || wait=''
  [[ -n "${wait}" ]] || wait="wait for GitHub's rate limit to reset; run: gh api rate_limit"
  error "${wait} (${what} could not be read)"
}

# forge/github:scoped <what> <said> says, when what gh said is that its login lacks a scope the read needs, the command
# that adds the scope GitHub names, and succeeds; for anything else, or a refusal that names no scope, it says nothing
# and fails. GitHub refuses a GraphQL field so with INSUFFICIENT_SCOPES, naming the field and the scopes that would do,
# as a team asked to review needs read:org.
forge/github:scoped() {
  local what=$1 said=$2 scope
  [[ "${said,,}" == *insufficient_scopes* || "${said,,}" == *'required scopes'* ]] || return 1
  scope=$(sed -n "s/.*requires one of the following scopes: \['\([^']*\)'.*/\1/p" <<<"${said}" | head -n 1)
  [[ -n "${scope}" ]] || return 1
  error "run: gh auth refresh --scopes ${scope} (${what} could not be read)"
}

# forge/github:teamless <said> succeeds when what gh said is GitHub refusing a team's field for a scope the login
# lacks, read:org, as a token without it is refused a team asked to review: the read can be asked again without it.
forge/github:teamless() {
  local said=$1
  [[ "${said,,}" == *insufficient_scopes* || "${said,,}" == *'required scopes'* ]] || return 1
  [[ "${said}" == *"'slug'"* || "${said}" == *read:org* || "${said}" == *Team* ]]
}

# ----------------------------------------------------------------------------------------------------------------------
