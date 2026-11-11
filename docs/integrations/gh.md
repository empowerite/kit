# gh

`gh` is GitHub's own CLI. It authenticates with `gh auth login`, or `GH_TOKEN` in CI. `--json "${fields}"` picks the
fields, `--jq` filters them, and `--limit` sets how many items a list reads. `gh api` is the escape hatch: it fills
`{owner}/{repo}` from the checkout, reads every page with `--paginate` (`--slurp` joins them), sends headers with
`--header`, prints the status line and headers with `--include`, and runs GraphQL with `gh api graphql`. kit reads
`gh` for a single change and GraphQL for the open changes, 100 a page, following each list's cursor to its end.

This page says what `gh` reads and writes where it meets kit's scope, for each service it reaches, and the
machine-readable descriptions each publishes. The index of these pages is [1st.md](1st.md).

| Service | Category | kit status |
| --- | --- | --- |
| [GitHub pull requests](#github-pull-requests) | [code forge](forges.md) | `provider` |
| [GitHub Issues](#github-issues) | [tracker](trackers.md) | `provider` (`forge`) |
| [GitHub Projects](#github-projects) | [tracker](trackers.md) | none |
| [GitHub Actions](#github-actions) | [CI](ci.md) | `provider`, through `forge/github.sh` |
| [GitHub releases and deployments](#github-releases-and-deployments) | [CD](cd.md) | `provider`: releases, environments, and deployments |

## GitHub pull requests

GitHub Enterprise Server answers the same calls at `https://${host}/api/v3` and `https://${host}/api/graphql`. `gh`
reaches it with `gh auth login --hostname "${host}"`, or `GH_HOST`, and kit finds such a host in `gh`'s `hosts.yml`.
Each Server release has its own REST description and GraphQL schema, and a field newer than the release is absent.
Rate limits on Server are off unless its administrator turns them on.

### Read from GitHub pull requests

| Capability | Command |
| --- | --- |
| Signed in | `gh api user --jq .login`; `gh auth status` passes with a revoked token, so kit makes a real call |
| Owner and name | `gh repo view --json nameWithOwner --jq .nameWithOwner` |
| Browser URL | `gh repo view --json url --jq .url`; a change's is `${url}/pull/${n}` |
| Caller's login | `gh api user --jq .login` (raw API) |
| Rate limits | `gh api rate_limit` (raw API); it counts against no limit, and holds `core` and `graphql` among its resources |
| Marks | `gh api --include --header "If-None-Match: ${etag}" 'repos/{owner}/{repo}/issues?state=all&sort=updated&direction=desc&per_page=1'` (raw API); a `304 Not Modified` counts against no limit. `repos/{owner}/{repo}/events` also takes an ETag and sends `X-Poll-Interval` |
| Branch for an issue | `gh issue develop --list "${n}"` lists the linked branches; kit reads `gh pr list --state open --json headRefName --limit "${limit}"` for the open change on `${n}-*` |
| Title, body | `gh pr view "${n}" --json title,body` |
| Author | `gh api "repos/{owner}/{repo}/pulls/${n}" --jq .user.login` (raw API); `gh pr view --json author` names a bot `app/dependabot`, not `dependabot[bot]` |
| State and merge commit | `gh pr view "${n}" --json state,mergeCommit --jq '"\(.state)\t\(.mergeCommit.oid // "")"'` |
| Change on a branch | `gh pr list --head "${branch}" --state all --json number,state,headRefOid --jq '.[0]'`; `--state open` for the open one |
| Closing issues | `gh pr view "${n}" --json closingIssuesReferences`; kit pages `closingIssuesReferences(first: 100, after: $after)` with `gh api graphql`, as `gh` reads at most 100 |
| Review threads | `gh api graphql` on `pullRequest(number: $n) { reviewThreads(first: 100) { nodes { isResolved } } }` (raw API) |
| Checks rollup | `gh pr checks "${n}" --json name,state,bucket,startedAt,completedAt`, or `gh pr view "${n}" --json statusCheckRollup` |
| Open changes | `gh api graphql` on `repository.pullRequests(states: OPEN, first: 100)` with threads, checks, closing issues, and reviewers (raw API), 18 points a page; `gh pr list --state open --json "${fields}" --limit "${limit}"` for fewer fields |
| Reviewers | in the open changes' query, `reviewRequests(first: 20) { totalCount nodes { requestedReviewer { ... on User { login } ... on Team { slug } } } }` and `latestReviews(first: 20) { totalCount nodes { author { login } state submittedAt } }`, failing past 20; a team's slug may need `read:org`, which `gh auth login` grants and a `GH_TOKEN` may lack: GitHub then refuses the field, and kit asks once more without `... on Team`, so the open changes still draw, a team asked among none of their reviewers; `gh pr view "${n}" --json reviewRequests,latestReviews` for one; `kit changes reviews` draws them |
| Merged changes | `gh api graphql` on `repository.pullRequests(states: MERGED, first: 50, orderBy: {field: UPDATED_AT, direction: DESC}) { totalCount nodes { number title url headRefName mergedAt updatedAt author { login } mergedBy { login } } }` (raw API), a point: GitHub orders by creation or update, never by merge, and with `DESC`, `last: 50` would be the oldest; past `totalCount`, kit keeps those merged since the oldest `updatedAt` among the 50, as one merged before it may be missing, and says `limited`; `kit changes merged` draws them by `mergedAt` |
| Who has access | `gh api --paginate --slurp 'repos/{owner}/{repo}/collaborators?per_page=100'` (raw API): each one's `login`, `type`, `Bot` for an app's account, and `role_name`, the highest role any source gives, those of the organization and its teams among them; GitHub shows them only to a caller who may push, and answers any other 403 `Must have push access`, which kit says names who can give it; `name` is mostly absent; `kit people people` draws them |
| Teams with access | `gh api 'repos/{owner}/{repo}/teams?per_page=21'` (raw API): each one's `slug`, `name`, `permission`, and `html_url`, a 21st saying there are more than the 20 kit draws, none for a repository a user owns, and 404 for some who can reach the repository, whom GitHub's reference does not say, so kit names an admin and no role; then one `gh api graphql` on `organization(login: $org) { t0: team(slug: $t0) { members(first: 100) { totalCount nodes { login } } } … }`, an alias a team, a point, which needs `read:org`: without it GitHub answers `INSUFFICIENT_SCOPES` and kit names `gh auth refresh --scopes read:org`; a team past 100 members says `limited`; `kit people teams` draws them |
| Bots that acted | one `gh api graphql` on `repository { defaultBranchRef { target { ... on Commit { history(first: 100) { nodes { committedDate author { name user { login __typename } } } } } } } open: pullRequests(states: OPEN, first: 100, …) { totalCount … } merged: pullRequests(states: MERGED, first: 50, …) }`, each change's `createdAt author { __typename login url }` and `latestReviews(first: 20) { totalCount … }`'s authors (raw API), two points, `limited` past 100 open changes or 20 reviewers of one: a `Bot`, whose GraphQL login lacks the `[bot]` REST gives, or a login that ends `[bot]`; a commit's author is a `GitActor`, its `user` null for an address GitHub does not know, and an agent's commit names itself, `copilot-swe-agent[bot]`, under a person's account; installed apps need admin or app credentials, so kit lists those that acted; `kit people agents` draws them |
| Attestations | `gh api 'repos/{owner}/{repo}/releases?per_page=6'` and `gh api 'repos/{owner}/{repo}/actions/artifacts?per_page=20'` (raw API), whose assets and artifacts carry a `digest`, `sha256:…`, then `gh api --paginate --slurp "repos/{owner}/{repo}/attestations/${digest}?per_page=100"` a request each, the 5 newest releases' assets first, each digest once, 50 at most, `limited` past those; a digest of none answers 404; `gh release view --json assets` carries the digest too, a request a release; a bundle's `dsseEnvelope.payload` is the in-toto statement in base64, its `predicateType` the kind and SLSA provenance's `buildDefinition.externalParameters.workflow` the signer, the certificate's SAN in DER, which kit does not parse; GitHub may leave the bundle out, behind `bundle_url`, snappy-compressed, as cli/cli's were, checked live, and kit draws its kind null; `kit audit evidence` draws them |
| Security alerts | `gh api --paginate --slurp 'repos/{owner}/{repo}/code-scanning/alerts?state=open&sort=updated&direction=desc&per_page=100'`, and `state=dismissed`, as code scanning takes one state a read; `dependabot/alerts?state=open,dismissed&…`, whose state is a list; `secret-scanning/alerts?state=open&…` and `state=resolved&resolution=false_positive,wont_fix,used_in_tests&…`, as secret scanning takes open or resolved alone, checked live (raw API); off for the repository, each checked live: 404 `no analysis found` or `no default branch found`, 403 `Code Security must be enabled…`, 403 `Dependabot alerts are disabled for this repository.`, 404 `Secret scanning is disabled on this repository.`; another 403 or 404 is asked once more with `--include`, and a classic token whose `X-OAuth-Scopes` hold none of `X-Accepted-OAuth-Scopes` is told `gh auth refresh --scopes security_events`, which every scanner accepts, else the caller is told to ask an admin; gh's own scope hint on such a refusal names the first scope GitHub accepts, `admin:repo_hook`, even when the login holds another; `kit audit findings` draws them |
| Whether alerts changed | `gh api --include 'repos/{owner}/{repo}/${scanner}/alerts?sort=updated&direction=desc&per_page=1'` (raw API) for each scanner, the alert updated last in any state, whose ETag a new alert, a fix, and a dismissal move; a scanner off is `off`, and one that refuses the caller leaves the mark null |
| Rulesets and protection | one `gh api graphql` on `repository { url defaultBranchRef { name } rulesets(first: 100, includeParents: true, targets: [BRANCH]) { totalCount nodes { databaseId name enforcement conditions { refName { include exclude } } rules(first: 100) { totalCount nodes { type } } bypassActors(first: 100) { totalCount nodes { bypassMode organizationAdmin repositoryRoleName deployKey enterpriseOwner actor { … on Team { slug } … on User { login } … on App { slug } } } } } } }` (raw API), a point, the REST list holding no conditions, rules, or bypass actors, checked live; a caller who may not edit a ruleset gets its bypass actors as null; then `gh api "repos/{owner}/{repo}/branches/${default}"`, whose `protection.enabled` says the branch has a protection of its own, and only then `…/protection`, which answers an admin alone, 404 to any other, checked live; `kit audit controls` and `kit audit waivers` draw them |

### Write to GitHub pull requests

| Capability | Command |
| --- | --- |
| Branch linked to an issue | `gh issue develop "${n}" --base "${default}"`: GitHub names the branch `${n}-${slug}`, links it under the issue's Development, and pushes it |
| Open a draft | `gh pr create --draft --base "${default}" --head "${branch}" --title "${title}" --body ""`; it prints the URL |
| Mark ready | `gh pr ready "${n}"`; `--undo` makes it a draft again |
| Set the title | `gh pr edit "${n}" --title "${title}"` |
| Set the body | `gh pr edit "${n}" --body "${body}"` |

Gaps: review threads have no `gh` command, so they are GraphQL alone. `gh` caps a JSON list at what one request
returns for nested connections, such as checks and closing issues, so a long list needs GraphQL paging.

### GitHub pull requests schemas

- [GitHub REST API OpenAPI description](https://raw.githubusercontent.com/github/rest-api-description/main/descriptions/api.github.com/api.github.com.json),
  from github/rest-api-description, with `/repos/{owner}/{repo}/pulls`, `/rate_limit`, and the webhook payloads.
  The same directory holds one per product, `ghec/` for Enterprise Cloud and one per Server release, the newest being
  [GitHub Enterprise Server 3.22](https://raw.githubusercontent.com/github/rest-api-description/main/descriptions/ghes-3.22/ghes-3.22.json).
- [GitHub GraphQL schema](https://docs.github.com/public/fpt/schema.docs.graphql): `PullRequest`, `reviewThreads`,
  `statusCheckRollup`, `closingIssuesReferences`, and `RateLimit`. Enterprise Cloud has its own,
  [GitHub Enterprise Cloud GraphQL schema](https://docs.github.com/public/ghec/schema.docs.graphql), and each Server
  release one, the newest being
  [GitHub Enterprise Server 3.22 GraphQL schema](https://docs.github.com/public/ghes-3.22/schema.docs-enterprise.graphql).
- [Octokit webhook JSON Schemas](https://unpkg.com/@octokit/webhooks-schemas/schema.json), community, for the
  `pull_request` and `pull_request_review_thread` events.
- [GitHub CLI hosts JSON Schema](https://www.schemastore.org/github-cli-hosts.json), SchemaStore's "GitHub CLI
  hosts" entry, for the `hosts.yml` kit reads to tell a GitHub Enterprise Server host, and
  [GitHub CLI configuration JSON Schema](https://www.schemastore.org/github-cli-config.json) for `config.yml`.
- `gh` publishes no schema of its own output; `gh pr view --json` with no field lists the fields.

## GitHub Issues

`gh` is GitHub's own CLI. `gh auth login` stores a token; `GH_TOKEN` overrides it. `--json "${fields}"` picks the fields
of a list or a view, `--jq` filters them, and `gh api` reaches every REST route and `gh api graphql` the GraphQL API.
`gh issue list --limit` reads every page up to the limit; `gh api --paginate --slurp` reads every page of a raw route.
kit's `forge/github.sh` uses these commands.

### Read from GitHub Issues

| Capability | Command |
| --- | --- |
| Title | `gh issue view "${n}" --json title --jq .title` |
| State | `gh issue view "${n}" --json state --jq .state` (`OPEN` or `CLOSED`; `stateReason` says why) |
| Assignees | `gh issue view "${n}" --json assignees --jq '.assignees[].login'` |
| Caller | `gh api user --jq .login` |
| Issue page | `gh issue view "${n}" --json url --jq .url` |
| New-issue page | none; kit builds `${repo_url}/issues/new?title=${title}` |
| Open issues | `gh issue list --state open --limit 10000 --json number,title,assignees,labels,createdAt,url` |
| Parent and sub-issues | `gh issue view "${n}" --json parent,subIssues,subIssuesSummary` |
| Changes that close it | `gh issue view "${n}" --json closedByPullRequestsReferences` |
| Linked branches | `gh issue develop --list "${n}"` |
| Milestones | `gh api --paginate --slurp 'repos/{owner}/{repo}/milestones?state=all&per_page=100'` (raw REST); `kit issues milestones` draws them, and its mark is the ETag of the same path's first page, so an edit to a milestone past the first 100 goes unseen |
| Iterations | none in Issues; `kit issues iterations` reads them from [GitHub Projects](#github-projects) |

### Write to GitHub Issues

| Capability | Command |
| --- | --- |
| Assign | `gh issue edit "${n}" --add-assignee "${login}"` |
| Unassign | `gh issue edit "${n}" --remove-assignee "${login}"` |
| Close as completed | `gh issue close "${n}" --reason completed` |
| Close as not planned | `gh issue close "${n}" --reason "not planned"` |
| Close as a duplicate | `gh issue close "${n}" --reason duplicate --duplicate-of "${m}"` |
| Comment | `gh issue comment "${n}" --body "${text}"` |
| Create | `gh issue create --title "${title}" --body "${body}"` (prints the issue's URL) |
| Put under a parent | `gh issue edit "${n}" --add-sub-issue "${m}"`, or `gh issue edit "${m}" --parent "${n}"` |
| Link a branch | `gh issue develop "${n}" --base "${default_branch}"` (listed under Development) |
| Link a change | `Closes #${n}` in the pull request's body |
| Set the milestone | `gh issue edit "${n}" --milestone "${milestone}"` |

Gaps: an issue has no status beyond open and closed, and no iteration; both live in a GitHub Projects field. `gh` has
no command that lists milestones. `kit linked` relies on GitHub's
[closing keywords](https://docs.github.com/en/issues/tracking-your-work-with-issues/using-issues/linking-a-pull-request-to-an-issue):
`close`, `closes`, `closed`, `fix`, `fixes`, `fixed`, `resolve`, `resolves`, and `resolved`, which close the issue
when the pull request merges into the default branch.

### GitHub Issues schemas

- [GitHub REST API OpenAPI 3.0](https://raw.githubusercontent.com/github/rest-api-description/main/descriptions/api.github.com/api.github.com.json),
  which holds the issue, sub-issue, and milestone routes
- [GitHub REST API OpenAPI 3.1 with webhooks](https://raw.githubusercontent.com/github/rest-api-description/main/descriptions-next/api.github.com/api.github.com.json):
  the `issues-*`, `milestone-*`, and `projects-v2-*` webhook payloads under `webhooks`
- [GitHub GraphQL schema](https://docs.github.com/public/fpt/schema.docs.graphql), also mirrored as
  [Octokit's GraphQL schema](https://raw.githubusercontent.com/octokit/graphql-schema/main/schema.graphql)
- [GitHub Issue Template forms](https://www.schemastore.org/github-issue-forms.json) and
  [GitHub Issue Template configuration](https://www.schemastore.org/github-issue-config.json), both SchemaStore
  entries, for `.github/ISSUE_TEMPLATE/`
- No JSON Schema for `gh` output; its `--json` fields are listed by `gh issue view --help`.

## GitHub Projects

GitHub Projects is the planning layer over GitHub issues and pull requests: a project holds items and the fields on
them, among them a single-select Status and an Iteration field. It holds no issues of its own, draft issues aside, so
the seam's calls stay with GitHub Issues; Projects answers kittui's iterations tab and the status field. `gh project`
needs the `project` scope (`gh auth refresh -s project`). `--format json` prints JSON, `--jq` filters it, and
`--limit` sets how many items it reads. The iterations of a field, with their start dates and durations, are read
through `gh api graphql`, paged with `--paginate` and `$endCursor`.

### Read from GitHub Projects

| Capability | Command |
| --- | --- |
| Projects | `gh project list --owner "${owner}" --format json` |
| Fields, with Status options | `gh project field-list "${number}" --owner "${owner}" --format json` |
| Items, with their Status and Iteration | `gh project item-list "${number}" --owner "${owner}" --format json --limit "${limit}"` |
| Items that match a filter | `gh project item-list "${number}" --owner "${owner}" --query "${filter}" --format json` |
| An issue's projects and field values | `gh issue view "${n}" --json projectItems` |
| Iterations, with dates | `gh api graphql -f query="${query}" -F project="${project_id}"` (raw GraphQL; below); `kit issues iterations` reads those of the first 20 projects linked to the repository, the first 50 fields of each, failing where there are more, and needing the `read:project` scope (`gh auth refresh --scopes read:project`), with one `repository.projectsV2` query, `-f owner=… -f name=…`, and flattens each iteration field's `iterations` and `completedIterations` |

```graphql
query($project: ID!) {
  node(id: $project) {
    ... on ProjectV2 {
      fields(first: 50) {
        nodes {
          ... on ProjectV2IterationField {
            name
            configuration {
              iterations { id title startDate duration }
              completedIterations { id title startDate duration }
            }
          }
        }
      }
    }
  }
}
```

### Write to GitHub Projects

| Capability | Command |
| --- | --- |
| Add an issue to a project | `gh project item-add "${number}" --owner "${owner}" --url "${issue_url}"`, or `gh issue edit "${n}" --add-project "${title}"` |
| Set the Status | `gh project item-edit --id "${item_id}" --project-id "${project_id}" --field-id "${field_id}" --single-select-option-id "${option_id}"` |
| Set the Iteration | `gh project item-edit --id "${item_id}" --project-id "${project_id}" --field-id "${field_id}" --iteration-id "${iteration_id}"` |

Gaps: `gh project` names projects, fields, and options by their node IDs, so a write first reads `field-list`. No
`gh project` command lists an iteration field's iterations with their dates.

### GitHub Projects schemas

- [GitHub GraphQL schema](https://docs.github.com/public/fpt/schema.docs.graphql): `ProjectV2`, `ProjectV2Item`,
  `ProjectV2IterationField`, and `ProjectV2SingleSelectField`
- [GitHub REST API OpenAPI 3.1 with webhooks](https://raw.githubusercontent.com/github/rest-api-description/main/descriptions-next/api.github.com/api.github.com.json):
  the `projects-v2-item-*` payloads; see also
  [using the API to manage projects](https://docs.github.com/en/issues/planning-and-tracking-with-projects/automating-your-project/using-the-api-to-manage-projects)

## GitHub Actions

`gh` is GitHub's CLI. It signs in with `gh auth login` or reads `GH_TOKEN`. Every list command takes `--json
<fields>`, with `--jq` and `--template` to shape it, and `--limit` instead of pages. `gh api` calls any REST route,
fills `{owner}` and `{repo}` from the checkout, and reads every page with `--paginate`; with `--slurp` it joins them.

### Read from GitHub Actions

| Capability | Command |
| --- | --- |
| Recent runs | `gh run list --limit 50 --json databaseId,number,attempt,workflowName,headBranch,headSha,displayTitle,event,status,conclusion,createdAt,startedAt,updatedAt,url`; filter with `--branch "${branch}"`, `--event push`, `--event schedule`, `--event workflow_dispatch` |
| A run's jobs and failed step | `gh run view "${run_id}" --json jobs`; each job has `steps[]` with `conclusion` |
| Whether runs changed | `gh api --include -H "If-None-Match: ${etag}" 'repos/{owner}/{repo}/actions/runs?per_page=10'`, as kit's runs mark does |
| A change's checks | `gh pr checks "${n}" --json name,state,bucket,link,workflow`; kit reads `statusCheckRollup` through `gh api graphql` |
| A run's log | `gh run view "${run_id}" --log-failed`, or `--log`; one job with `--job "${job_id}"` |
| Artifacts | list: `gh api 'repos/{owner}/{repo}/actions/artifacts?per_page=100'` (raw API; no `gh` command lists them), the newest 100, each with its `digest`, `size_in_bytes`, `workflow_run`, and `expires_at`, and `total_count` past the page saying there are more; `kit delivery artifacts` draws those not `expired`; one run's: `gh api repos/{owner}/{repo}/actions/runs/${run_id}/artifacts`; download: `gh run download "${run_id}" --name "${name}"` |

### Write to GitHub Actions

| Capability | Command |
| --- | --- |
| Rerun | `gh run rerun "${run_id}"`; `--failed` for the failed jobs and what they need; `--job "${job_id}"` for one |
| Cancel | `gh run cancel "${run_id}"`; `--force` |
| Start a run | `gh workflow run "${workflow}" --ref "${ref}" -f "${key}=${value}"`; the workflow needs a `workflow_dispatch` trigger |
| Report a status | `gh api repos/{owner}/{repo}/statuses/${sha} -f state=success -f context=kit -f target_url="${url}"` |

A check run, `POST repos/{owner}/{repo}/check-runs`, takes a GitHub App's token, so a person's token or `gh` alone
posts only a commit status. `gh` has no command that validates a workflow file.

### GitHub Actions schemas

- [GitHub workflow JSON Schema](https://www.schemastore.org/github-workflow.json), SchemaStore's "GitHub Workflow"
  entry, which also matches `.gitea/workflows/` and `.forgejo/workflows/`
- [GitHub action metadata JSON Schema](https://www.schemastore.org/github-action.json), for `action.yml`
- [GitHub REST OpenAPI](https://raw.githubusercontent.com/github/rest-api-description/main/descriptions/api.github.com/api.github.com.json),
  with the `actions/runs`, `actions/jobs`, `check-runs`, and `statuses` routes, and its
  [YAML form](https://raw.githubusercontent.com/github/rest-api-description/main/descriptions/api.github.com/api.github.com.yaml)
- [GitHub GraphQL schema](https://docs.github.com/public/fpt/schema.docs.graphql), with `CheckRun`, `StatusContext`,
  and `WorkflowRun`
- [Octokit webhook JSON Schema](https://unpkg.com/@octokit/webhooks-schemas/schema.json), for `workflow_run`,
  `check_run`, and `status` payloads

## GitHub releases and deployments

`gh` is GitHub's own CLI. It authenticates with `gh auth login`, or `GH_TOKEN` in CI. `--json "${fields}"` picks the
fields, `--jq` filters them, and `--limit` sets how many items a list reads. `gh` has release commands and none for
environments or deployments, so those go through `gh api`, which fills `{owner}/{repo}` from the checkout, reads
every page with `--paginate`, and runs GraphQL with `gh api graphql`. kit reads releases with `gh api`, as
`gh release list` gives no release's notes or assets; the REST list does not mark the latest release, so kit names it
as GitHub does by default: the newest release, by its creation, that is neither a draft nor a pre-release.

A job in GitHub Actions that names an `environment:` creates a deployment for that environment, and its run waits
there for the environment's protection rules, such as required reviewers. A deployment whose ref is a pull request's
head branch shows on that pull request. Releases can be immutable: once published, their tag and assets stay fixed.

### Read from GitHub releases and deployments

| Capability | Command |
| --- | --- |
| List releases, with their notes, pages, and assets | `gh api --paginate 'repos/{owner}/{repo}/releases?per_page=100'` (raw API) |
| Which release is the latest | `gh release list --limit "${limit}" --json tagName,isLatest`, or `gh api 'repos/{owner}/{repo}/releases/latest'` (raw API) |
| A release's notes | `gh release view "${tag}" --json body,url` |
| List environments | `gh api --paginate --slurp 'repos/{owner}/{repo}/environments?per_page=100'` (raw API): each one's `protection_rules`, `required_reviewers` with its `reviewers`, `wait_timer`, or `branch_policy`, and `deployment_branch_policy`; `kit delivery environments` draws them |
| An environment's branch policies | `gh api "repos/{owner}/{repo}/environments/${environment}/deployment-branch-policies?per_page=100"` (raw API), for one whose `deployment_branch_policy` has `custom_branch_policies`, each a `branch` or a `tag` pattern; kit reads 20 environments' at most |
| Deployments in an environment | `gh api --paginate "repos/{owner}/{repo}/deployments?environment=${environment}&per_page=100"` (raw API) |
| Whether deployments or environments changed | the ETags of `gh api --include 'repos/{owner}/{repo}/deployments?per_page=50'` and `'repos/{owner}/{repo}/environments?per_page=100'`, sent back with `If-None-Match`, as kit's deployments and environments marks do; a new status updates a deployment |
| A deployment's state | `gh api --paginate "repos/{owner}/{repo}/deployments/${deployment_id}/statuses"` (raw API); the first status is the newest |
| The newest deployments, each with its latest status | `gh api graphql` on `repository.deployments(first: 50, orderBy: {field: CREATED_AT, direction: DESC}) { totalCount nodes { databaseId environment ref { name } commitOid state createdAt creator { login } commit { url } latestStatus { state logUrl environmentUrl } } }` (raw API), a point; `kit delivery deployments` draws them |
| What is live where | `gh api graphql` on `repository.environments.nodes { name latestCompletedDeployment { sha ref { name } latestStatus { state } } }` (raw API) |
| Approvals a run waits on | `gh api "repos/{owner}/{repo}/actions/runs/${run_id}/pending_deployments"` (raw API) |
| Approvals given | `gh api "repos/{owner}/{repo}/actions/runs/${run_id}/approvals"` (raw API) |

### Write to GitHub releases and deployments

| Capability | Command |
| --- | --- |
| Create a release from a tag with notes | `gh release create "${tag}" --verify-tag --generate-notes` or `--notes-file "${file}"`; `--prerelease`, `--draft` |
| Publish or change notes | `gh release edit "${tag}" --notes-file "${file}"` |
| Generate notes without a release | `gh api 'repos/{owner}/{repo}/releases/generate-notes' -f tag_name="${tag}"` (raw API) |
| Deploy or promote | `gh workflow run "${workflow}" --ref "${ref}" -f environment="${environment}"`, when the workflow takes that input |
| Roll back | `gh run rerun "${run_id}"` on the run that deployed the earlier commit; GitHub re-runs a run up to 30 days after it first ran |
| Approve or reject | `gh api -X POST "repos/{owner}/{repo}/actions/runs/${run_id}/pending_deployments" -F 'environment_ids[]'="${environment_id}" -f state=approved -f comment="${comment}"` (raw API) |
| Record a deployment made elsewhere | `gh api 'repos/{owner}/{repo}/deployments' -f ref="${sha}" -f environment="${environment}"` (raw API) |
| Report its status | `gh api "repos/{owner}/{repo}/deployments/${deployment_id}/statuses" -f state=success -f environment_url="${url}"` (raw API) |

Gaps: `gh` has no environment or deployment command, so every read except releases goes through `gh api`. GraphQL's
`Environment` holds no branch policies, so they cost a request each. GitHub has no rollback primitive; a rollback is a
re-run of an older run, or a new deployment of an older commit.

### GitHub releases and deployments schemas

- [GitHub REST API OpenAPI description](https://raw.githubusercontent.com/github/rest-api-description/main/descriptions/api.github.com/api.github.com.json),
  from github/rest-api-description; it holds `/repos/{owner}/{repo}/releases`, `/environments`, `/deployments`,
  `/deployments/{deployment_id}/statuses`, and `/actions/runs/{run_id}/pending_deployments`, and the webhook
  payloads, `deployment` and `deployment_status` among them.
- [GitHub GraphQL schema](https://docs.github.com/public/fpt/schema.docs.graphql): `Environment`,
  `Deployment`, `DeploymentStatus`, and `Release`.
- [GitHub Actions workflow JSON Schema](https://www.schemastore.org/github-workflow.json), SchemaStore's
  "GitHub Workflow" entry, for a job's `environment:`.
- [Release notes configuration JSON Schema](https://www.schemastore.org/github-release-config.json), SchemaStore's
  entry for `.github/release.yml`, which `--generate-notes` reads.
- [Octokit webhook JSON Schemas](https://unpkg.com/@octokit/webhooks-schemas/schema.json), community, covering the
  same events.

## Schemas of its output

`gh` publishes no schema of what its commands print, so each command kit runs whose JSON kit reads has one
here, in [gh/](gh/): a JSON Schema of the fields kit reads, typed from the provider's API description,
with an example the gate validates against it.

| Schema | The command as kit runs it |
| --- | --- |
| [api-graphql-iterations.schema.json](gh/api-graphql-iterations.schema.json) | `gh api graphql --field owner=${owner} --field name=${name} --raw-field query=${query}` |
| [api-graphql-deployments.schema.json](gh/api-graphql-deployments.schema.json) | `gh api graphql --field owner=${owner} --field name=${name} --raw-field query=${query}` |
| [api-graphql-bots.schema.json](gh/api-graphql-bots.schema.json) | `gh api graphql --field owner=${owner} --field name=${name} --raw-field query=${query}` |
| [api-graphql-team-members.schema.json](gh/api-graphql-team-members.schema.json) | `gh api graphql --raw-field org=${owner} --raw-field t0=${slug}… --raw-field query=${query}` |
| [api-graphql-rulesets.schema.json](gh/api-graphql-rulesets.schema.json) | `gh api graphql --field owner=${owner} --field name=${name} --raw-field query=${query}` |
| [api-graphql-merged-prs.schema.json](gh/api-graphql-merged-prs.schema.json) | `gh api graphql --field owner=${owner} --field name=${name} --raw-field query=${query}` |
| [api-graphql-more-checks.schema.json](gh/api-graphql-more-checks.schema.json) | `gh api graphql --field owner=${owner} --field name=${name} --field pr=${number} --raw-field after=${cursor} --raw-field query=${query}` |
| [api-graphql-more-closing-issues.schema.json](gh/api-graphql-more-closing-issues.schema.json) | `gh api graphql --field owner=${owner} --field name=${name} --field pr=${number} --raw-field after=${cursor} --raw-field query=${query}` |
| [api-graphql-more-review-threads.schema.json](gh/api-graphql-more-review-threads.schema.json) | `gh api graphql --field owner=${owner} --field name=${name} --field pr=${number} --raw-field after=${cursor} --raw-field query=${query}` |
| [api-graphql-open-prs.schema.json](gh/api-graphql-open-prs.schema.json) | `gh api graphql --field owner=${owner} --field name=${name} --raw-field after=${cursor} --raw-field query=${query}` |
| [api-graphql-pr-closing-issues.schema.json](gh/api-graphql-pr-closing-issues.schema.json) | `gh api graphql --field owner=${owner} --field name=${name} --field number=${number} --raw-field after=${cursor} --raw-field query=${query}` |
| [api-rate-limit.schema.json](gh/api-rate-limit.schema.json) | `gh api rate_limit` |
| [api-repos-actions-artifacts.schema.json](gh/api-repos-actions-artifacts.schema.json) | `gh api repos/{owner}/{repo}/actions/artifacts?per_page=100` |
| [api-repos-attestations.schema.json](gh/api-repos-attestations.schema.json) | `gh api --paginate --slurp repos/{owner}/{repo}/attestations/${digest}?per_page=100` |
| [api-repos-branch.schema.json](gh/api-repos-branch.schema.json) | `gh api repos/{owner}/{repo}/branches/${branch}` |
| [api-repos-branch-protection.schema.json](gh/api-repos-branch-protection.schema.json) | `gh api repos/{owner}/{repo}/branches/${branch}/protection` |
| [api-repos-code-scanning-alerts.schema.json](gh/api-repos-code-scanning-alerts.schema.json) | `gh api --paginate --slurp repos/{owner}/{repo}/code-scanning/alerts?state=${state}&sort=updated&direction=desc&per_page=100` |
| [api-repos-collaborators.schema.json](gh/api-repos-collaborators.schema.json) | `gh api --paginate --slurp repos/{owner}/{repo}/collaborators?per_page=100` |
| [api-repos-dependabot-alerts.schema.json](gh/api-repos-dependabot-alerts.schema.json) | `gh api --paginate --slurp repos/{owner}/{repo}/dependabot/alerts?state=${states}&sort=updated&direction=desc&per_page=100` |
| [api-repos-environment-branch-policies.schema.json](gh/api-repos-environment-branch-policies.schema.json) | `gh api repos/{owner}/{repo}/environments/${environment}/deployment-branch-policies?per_page=100` |
| [api-repos-environments.schema.json](gh/api-repos-environments.schema.json) | `gh api --paginate --slurp repos/{owner}/{repo}/environments?per_page=100` |
| [api-repos-milestones.schema.json](gh/api-repos-milestones.schema.json) | `gh api --paginate --slurp repos/{owner}/{repo}/milestones?state=all&per_page=100` |
| [api-repos-pulls.schema.json](gh/api-repos-pulls.schema.json) | `gh api repos/${repo}/pulls/${number} --jq .user.login` |
| [api-repos-releases.schema.json](gh/api-repos-releases.schema.json) | `gh api --paginate repos/{owner}/{repo}/releases?per_page=100` |
| [api-repos-secret-scanning-alerts.schema.json](gh/api-repos-secret-scanning-alerts.schema.json) | `gh api --paginate --slurp repos/{owner}/{repo}/secret-scanning/alerts?state=${state}[&resolution=…]&sort=updated&direction=desc&per_page=100` |
| [api-repos-teams.schema.json](gh/api-repos-teams.schema.json) | `gh api repos/{owner}/{repo}/teams?per_page=21` |
| [api-user.schema.json](gh/api-user.schema.json) | `gh api user --jq .login` |
| [issue-list.schema.json](gh/issue-list.schema.json) | `gh issue list --state open --limit ${forge_list_limit} --json number,title,assignees,labels,createdAt,url` |
| [issue-view.schema.json](gh/issue-view.schema.json) | `gh issue view ${number} --json ${fields} --jq ${filter}` |
| [pr-list.schema.json](gh/pr-list.schema.json) | `gh pr list --head ${branch} --state ${state} --json ${field} --limit ${forge_list_limit} --jq ${filter}` |
| [pr-view.schema.json](gh/pr-view.schema.json) | `gh pr view ${number} --json ${fields} --jq ${filter}` |
| [repo-view.schema.json](gh/repo-view.schema.json) | `gh repo view --json ${field} --jq .${field}` |
| [run-list.schema.json](gh/run-list.schema.json) | `gh run list --limit 50 --json databaseId,number,attempt,workflowName,headBranch,headSha,displayTitle,event,status,conclusion,createdAt,startedAt,updatedAt,url` |
| [run-view.schema.json](gh/run-view.schema.json) | `gh run view ${id} --json jobs` |
