# Code forges

A code forge is the service that hosts a repository with its branches and its changes: pull requests on GitHub,
merge requests on GitLab, patches on Radicle. kit reaches it for the repository's name and URL, the caller's login and
rate limits, the branch linked to an issue, and a change's draft, title, body, author, state, closing issues, review
threads, and checks. The index of these pages is [1st.md](1st.md).

Every such call goes through one seam, `kit/modules/bash/forge.sh`, whose header is the contract. The `forge` setting
or the primary remote's host picks the provider: `forge/github.sh` over `gh`, `forge/gitlab.sh` over `glab`, or
`forge/ado.sh` over `az`. `kit work` links a branch to an issue and opens a draft change. `kit linked` and
`kit policy` read the change in CI. `kit settle` and `kit tidy` read its state when it closes, and the changes and
underway stages of `kit flow` draw every open change with its checks and threads. The forge seam's issue calls are
on [trackers.md](trackers.md), its `runs` on [ci.md](ci.md), and its `releases` on [cd.md](cd.md).
see: [architecture.md](../architecture.md#code-forges-seams-and-providers), `kit/modules/bash/forge.sh`

## Scope

| Capability | Kind | Used by |
| --- | --- | --- |
| The tool is present and signed in | read | `forge:require`, before every verb that calls the code forge |
| The repository as owner and name | read | `forge:repo`: `kit work`, `kit linked` |
| The repository's and a change's browser URL | read | `forge:repo-url`, `forge:pr-url`: the messages of `kit work` and `kit linked` |
| The change's sigil and name, `#` and pull request | none | `forge:noun`, a constant of the provider |
| The caller's login | read | `forge:me`: `kit work` assigns the caller; the underway stage marks the caller's issues |
| The caller's rate limits, at most one request | read | `forge:quota`: `kit flow` |
| A cheap mark that changes when issues, changes, runs, or releases do | read | `forge:marks`: `kit flow` reloads a stage when its mark moves |
| Make a branch linked to an issue, from the default branch, pushed | write | `forge:link-branch`: `kit work` |
| The branch an open change for an issue is on | read | `forge:branch-for-issue`: `kit work`, so a second run reuses the branch |
| Open a change as a draft | write | `forge:pr-create`: `kit work` |
| Mark the change ready for review | write | the workflow's last step, with the forge's own command; no seam call |
| The change's title | read | `forge:pr-title`: `kit policy` |
| The change's body | read | `forge:pr-body`: `kit linked`, `kit policy`, `kit work` |
| Replace the change's body | write | `forge:pr-set-body`: `kit work` adds its `Closes` lines |
| The change's author, as the forge shows it | read | `forge:pr-author`: `kit linked` (`linked_exempt`), `kit policy` |
| The change's state, and its merge commit once merged | read | `forge:pr-state`: `kit settle`, `kit tidy` |
| The change on a branch: open, latest in any state, its state, its head commit | read | `forge:pr-for-branch`, `pr-last-for-branch`, `pr-state-for-branch`, `pr-head-for-branch`, `pr-here`: `kit commit`, `kit tidy`, `kit linked`, the worktrees stage |
| The issues the change closes, with their state | read | `forge:pr-closing-issues`: `kit linked`, `kit settle` |
| The change's review threads, resolved or not | read | `forge:open-prs`: the changes stage; planned: kittui's reviews tab |
| The change's checks rollup on its head commit | read | `forge:open-prs`: the changes stage; planned: kittui's checks tab |
| Every open change, whole, with the above | read | `forge:open-prs`: the changes, backlog, and underway stages |
| Merged changes | read | planned: kittui's merged tab |

## Providers

| Provider | Hosting | CLI | kit status | JSON output | Machine-readable descriptions |
| --- | --- | --- | --- | --- | --- |
| GitHub | SaaS; GitHub Enterprise Server self-hosted; proprietary | `gh`, official ([cli/cli](https://github.com/cli/cli), MIT) | `provider` | `--json "${fields}"`, `--jq`; `gh api` | REST OpenAPI per product and version, GraphQL schema, webhook schemas |
| GitLab | SaaS; self-managed (Community Edition MIT, Enterprise Edition proprietary) | `glab`, official ([gitlab-org/cli](https://gitlab.com/gitlab-org/cli), MIT) | `provider` | `--output json` (`-F json`), `--jq`; `glab api` | REST OpenAPI 3.0, GraphQL schema |
| Azure DevOps Repos | SaaS (Azure DevOps Services); Azure DevOps Server self-hosted; proprietary | `az` with the `azure-devops` extension, official ([Azure/azure-devops-cli-extension](https://github.com/Azure/azure-devops-cli-extension), MIT) | `provider` | `--output json`, `--query`; `az devops invoke`, `az rest` | none published; a REST reference in prose |
| Bitbucket Cloud | SaaS; proprietary | `bkt`, community ([avivsinai/bitbucket-cli](https://github.com/avivsinai/bitbucket-cli), MIT); `twg`, Atlassian's, beta ([atlassian/twg-cli](https://github.com/atlassian/twg-cli)) | `none` | `bkt`: `--json`, `--jq`; `bkt api`. `twg`: `-o json` | REST OpenAPI 3.0 and Swagger 2.0 |
| Bitbucket Data Center | self-hosted; proprietary | `bkt`, community ([avivsinai/bitbucket-cli](https://github.com/avivsinai/bitbucket-cli), MIT) | `none` | `--json`, `--jq`; `bkt api` | REST OpenAPI 3.0 per version |
| AWS CodeCommit | SaaS; proprietary | `aws`, official ([aws/aws-cli](https://github.com/aws/aws-cli), Apache-2.0) | `planned` | JSON by default, `--query` | Smithy model, botocore model |
| Gitea | self-hosted, open source (MIT); Gitea Cloud SaaS | `tea`, official ([gitea/tea](https://gitea.com/gitea/tea), MIT) | `none` | `--output json` with `--fields`; `tea api` | Swagger 2.0 and OpenAPI 3.0, served by every instance |
| Forgejo and Codeberg | self-hosted, open source (GPL-3.0-or-later); Codeberg is a hosted Forgejo | `fj`, community ([forgejo-contrib/forgejo-cli](https://codeberg.org/forgejo-contrib/forgejo-cli), Apache-2.0 or MIT); `tea` for the API Forgejo shares with Gitea | `none` | `fj`: none; `tea` as above | Swagger 2.0, served by every instance |
| Gerrit | self-hosted, open source (Apache-2.0) | `git-review`, community ([opendev/git-review](https://opendev.org/opendev/git-review)); `ssh -p 29418 "${host}" gerrit`, built in | `none` | `gerrit query --format=JSON`; REST answers JSON after a `)]}'` line | none |
| SourceHut | SaaS (sr.ht); self-hosted, open source (AGPL-3.0) | `hut`, community ([~xenrox/hut](https://git.sr.ht/~xenrox/hut), AGPL-3.0) | `none` | none; `hut graphql` prints the API's JSON | GraphQL schema per service |
| Radicle | peer-to-peer, open source (MIT or Apache-2.0) | `rad`, official ([radicle-dev/heartwood](https://github.com/radicle-dev/heartwood), a mirror) | `none` | `rad cob show` prints JSON; `rad patch` prints text | none |

The order is a judgment of how widely each is used, kit's own providers first; no survey backs it. Bitbucket Cloud
and Bitbucket Data Center are two products with two APIs, so they have two rows. Bitbucket Cloud has no issues since
Atlassian removed them on 20 August 2026, so a Bitbucket repository's tracker is Jira or another; see
[trackers.md](trackers.md).

## GitHub

`gh` is GitHub's own CLI. It authenticates with `gh auth login`, or `GH_TOKEN` in CI. `--json "${fields}"` picks the
fields, `--jq` filters them, and `--limit` sets how many items a list reads. `gh api` is the escape hatch: it fills
`{owner}/{repo}` from the checkout, reads every page with `--paginate` (`--slurp` joins them), sends headers with
`--header`, prints the status line and headers with `--include`, and runs GraphQL with `gh api graphql`. kit reads
`gh` for a single change and GraphQL for the open changes, 100 a page, following each list's cursor to its end.

GitHub Enterprise Server answers the same calls at `https://${host}/api/v3` and `https://${host}/api/graphql`. `gh`
reaches it with `gh auth login --hostname "${host}"`, or `GH_HOST`, and kit finds such a host in `gh`'s `hosts.yml`.
Each Server release has its own REST description and GraphQL schema, and a field newer than the release is absent.
Rate limits on Server are off unless its administrator turns them on.

### Read from GitHub

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
| Open changes | `gh api graphql` on `repository.pullRequests(states: OPEN, first: 100)` with threads, checks, and closing issues (raw API); `gh pr list --state open --json "${fields}" --limit "${limit}"` for fewer fields |

### Write to GitHub

| Capability | Command |
| --- | --- |
| Branch linked to an issue | `gh issue develop "${n}" --base "${default}"`: GitHub names the branch `${n}-${slug}`, links it under the issue's Development, and pushes it |
| Open a draft | `gh pr create --draft --base "${default}" --head "${branch}" --title "${title}" --body ""`; it prints the URL |
| Mark ready | `gh pr ready "${n}"`; `--undo` makes it a draft again |
| Set the title | `gh pr edit "${n}" --title "${title}"` |
| Set the body | `gh pr edit "${n}" --body "${body}"` |

Gaps: review threads have no `gh` command, so they are GraphQL alone. `gh` caps a JSON list at what one request
returns for nested connections, such as checks and closing issues, so a long list needs GraphQL paging.

### GitHub schemas

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

## GitLab

`glab` is GitLab's own CLI. It authenticates with `glab auth login`, or `GITLAB_TOKEN`, and a CI job token inside
GitLab CI. `--output json` (`-F json`) prints JSON, `--jq` filters it, and `--page` and `--per-page` page a list.
`glab api` is the escape hatch: it fills `:id`, `:fullpath`, `:repo`, and `:branch` from the checkout, reads every
page with `--paginate`, prints headers with `--include`, and runs GraphQL with `glab api graphql`. kit reads the REST
API through `glab api` for every call but the draft, and pages each list itself.

A self-managed GitLab answers at `https://${host}/api/v4` and `https://${host}/api/graphql`. `glab` reaches it with
`glab auth login --hostname "${host}"`, or `GITLAB_HOST`, and kit finds such a host in `glab`'s `config.yml`. Its
administrator sets its rate limits, so its `RateLimit-*` headers may be absent. Its version decides which endpoints
and GraphQL fields exist, and features of the paid tiers are absent from Community Edition.

### Read from GitLab

| Capability | Command |
| --- | --- |
| Signed in | `glab api user --jq .username`; `glab auth status` for the person |
| Owner and name | `glab repo view -F json --jq .path_with_namespace`; kit reads it from the primary remote's URL |
| Browser URL | `glab repo view -F json --jq .web_url`; a merge request's is `${url}/-/merge_requests/${n}` |
| Caller's login | `glab api user --jq .username` (raw API) |
| Rate limits | `glab api --include user` (raw API): the `RateLimit-Name`, `RateLimit-Limit`, `RateLimit-Remaining`, and `RateLimit-Reset` headers |
| Marks | `glab api 'projects/:id/merge_requests?order_by=updated_at&sort=desc&per_page=1'` and the same for `issues`, `pipelines`, and `releases` (raw API); GitLab sends no ETag for these, so a mark is the newest `updated_at`, and each costs a request |
| Branch for an issue | `glab mr list -F json --per-page 100` and the open merge request whose `source_branch` starts with `${n}-` |
| Title, body | `glab mr view "${n}" -F json --jq '{title, description}'` |
| Author | `glab mr view "${n}" -F json --jq .author.username` |
| State and merge commit | `glab mr view "${n}" -F json --jq '"\(.state)\t\(.merge_commit_sha // .squash_commit_sha // "")"'` |
| Change on a branch | `glab mr list --source-branch "${branch}" -F json`; `--all` adds the closed and merged ones |
| Closing issues | `glab api --paginate "projects/:id/merge_requests/${n}/closes_issues"` (raw API); `glab mr issues "${n}"` prints them as text |
| Review threads | `glab mr note list "${n}" --state unresolved -F json` (experimental); kit reads `glab api --paginate "projects/:id/merge_requests/${n}/discussions"` and their `resolvable` and `resolved` notes |
| Checks rollup | `glab ci status --branch "${branch}" -F json`; kit reads `glab api --paginate "projects/:id/pipelines/${pipeline}/jobs"` for the head pipeline's jobs |
| Open changes | `glab mr list -F json --per-page 100`, or `glab api --paginate 'projects/:id/merge_requests?state=opened'` |

### Write to GitLab

| Capability | Command |
| --- | --- |
| Branch linked to an issue | `glab api -X POST 'projects/:id/repository/branches' -f branch="${n}-${slug}" -f ref="${default}"` (raw API); GitLab links a branch named `${n}-…` to issue `${n}`. `glab mr create --related-issue "${n}"` makes the branch and a merge request at once |
| Open a draft | `glab mr create --draft --source-branch "${branch}" --target-branch "${default}" --title "${title}" --description "" --yes` |
| Mark ready | `glab mr update "${n}" --ready`; `--draft` makes it a draft again |
| Set the title | `glab mr update "${n}" --title "${title}"` |
| Set the body | `glab mr update "${n}" --description "${body}"`, or `--description-file -` from standard input |

Gaps: `glab mr issues` has no JSON output, so closing issues come from the API. A merge request's checks are its
pipeline's jobs, which take a read per merge request, and a fork's pipeline lives in the fork's project, which the
caller may not be able to read.

### GitLab schemas

- [GitLab REST API OpenAPI 3.0 description](https://gitlab.com/gitlab-org/gitlab/-/raw/master/doc/api/openapi/openapi_v3.yaml),
  generated from the source, with `/api/v4/projects/{id}/merge_requests` and its `closes_issues` and `discussions`.
  [GitLab REST API OpenAPI 2.0 description](https://gitlab.com/gitlab-org/gitlab/-/raw/master/doc/api/openapi/openapi_v2.yaml)
  is deprecated and no longer updated.
- [GitLab GraphQL schema](https://gitlab.com/gitlab-org/gitlab/-/jobs/artifacts/master/raw/tmp/tests/graphql/gitlab_schema.graphql?job=graphql-schema-dump),
  the `graphql-schema-dump` job's artifact on the default branch, so it describes GitLab.com's next release, not a
  self-managed instance's. Its `MergeRequest` has `discussions`, `headPipeline`, and `draft`.
- No JSON Schema for webhook payloads or for `glab` output is published. The CI file's schema is on [ci.md](ci.md).

## Azure DevOps Repos

`az` is Microsoft's Azure CLI, and its `azure-devops` extension, `az extension add --name azure-devops`, adds
`az repos`. It authenticates with `az login`, or a personal access token in `AZURE_DEVOPS_EXT_PAT`, and takes the
organization and project from `az devops configure --defaults` or from `--org` and `--project`. `--output json` is
the default output, and `--query` filters it with JMESPath. A list takes `--top` and `--skip`. `az devops invoke
--area "${area}" --resource "${resource}" --route-parameters …` calls any REST resource by its area, and `az rest`
any URL. kit's `ado` tracker already uses `az boards`; see [trackers.md](trackers.md).

kit's provider, `forge/ado.sh`, reads the organization, the project, and the repository from the primary remote's
URL. It implements contract 2 and declares that Azure DevOps lacks rate limits it shows, cheap marks, releases, and
branches linked to a work item: `kit work` pushes the branch it names, and the pull request it opens links the work
item. Azure Boards is the forge's own tracker, as `#7` and `AB#7` alike, and `forge_settings.ado` in `kit.yml` names
the work item type and the closing states of the project's process.

Azure DevOps Server, the self-hosted product, answers at `https://${host}/${collection}` with the same REST API at an
older version. `--org` names the collection URL; which Server releases the extension supports was not checked.

### Read from Azure DevOps Repos

| Capability | Command |
| --- | --- |
| Signed in | `az account show --query user.name --output tsv`, as the `ado` tracker does |
| Owner and name | `az repos show --repository "${repo}" --query '[project.name, name]'` |
| Browser URL | `az repos show --repository "${repo}" --query webUrl`; a pull request's is `${webUrl}/pullrequest/${n}` |
| Caller's login | `az account show --query user.name` |
| Rate limits | no call; Azure DevOps sends `X-RateLimit-Resource`, `X-RateLimit-Delay`, `X-RateLimit-Limit`, `X-RateLimit-Remaining`, and `X-RateLimit-Reset` only on a request it delays, which `az` does not show |
| Marks | none cheap found; `az repos pr list --status all --top 1` gives one pull request, in an order not verified, and misses an update to an older one; no ETag was checked |
| Branch for an issue | `az boards work-item show --id "${n}" --expand relations` and its `ArtifactLink` relations named `Branch` or `Pull Request` |
| Title, body | `az repos pr show --id "${n}" --query '[title, description]'` |
| Author | `az repos pr show --id "${n}" --query createdBy.uniqueName` |
| State and merge commit | `az repos pr show --id "${n}" --query '[status, lastMergeCommit.commitId]'`: `active`, `abandoned`, or `completed` |
| Change on a branch | `az repos pr list --source-branch "${branch}" --status all` |
| Closing issues | `az repos pr work-item list --id "${n}"`: the linked work items, which completion can transition |
| Review threads | `az devops invoke --area git --resource pullRequestThreads --route-parameters project="${project}" repositoryId="${repo}" pullRequestId="${n}"` (raw API); each thread's `status` is `active`, `fixed`, `closed`, or another |
| Checks rollup | `az repos pr policy list --id "${n}"`: the policy evaluations, build validation among them; other statuses through `--resource pullRequestStatuses` (raw API) |
| Open changes | `az repos pr list --status active --top "${limit}"` |

### Write to Azure DevOps Repos

| Capability | Command |
| --- | --- |
| Branch linked to an issue | `az repos ref create --name "refs/heads/${branch}" --object-id "${sha}"`, then `az boards work-item relation add --id "${n}" --relation-type Branch --target-url "vstfs:///Git/Ref/${project_id}%2F${repo_id}%2FGB${branch}"` (the link not verified) |
| Open a draft | `az repos pr create --draft true --source-branch "${branch}" --target-branch "${default}" --title "${title}" --work-items "${n}"` |
| Mark ready | `az repos pr update --id "${n}" --draft false` |
| Set the title | `az repos pr update --id "${n}" --title "${title}"` |
| Set the body | `az repos pr update --id "${n}" --description "${body}"`; each value is a line |

Gaps: review threads and their state have no `az repos` command. Azure DevOps closes no work item from a pull
request's body by itself; work items link with `--work-items` and move on completion with `--transition-work-items`.
The `az devops invoke` area and resource names here were not tried against a live organization.

### Azure DevOps Repos schemas

- Microsoft publishes no OpenAPI document for the Azure DevOps REST API. The Swagger documents per area were in
  MicrosoftDocs/vsts-rest-api-specs, which answers 404 on 2026-11-10, and no other copy was found.
- The REST reference is [Azure DevOps Git REST API](https://learn.microsoft.com/en-us/rest/api/azure/devops/git/),
  a page, not a document; the rate-limit headers are on
  [Azure DevOps rate limits](https://learn.microsoft.com/en-us/azure/devops/integrate/concepts/rate-limits).
- No JSON Schema for service hook payloads or for `az` output is published. The pipeline file's schema is on
  [ci.md](ci.md).

## Bitbucket Cloud

Atlassian publishes no general Bitbucket CLI. `acli`, its CLI for Jira and admin, has no Bitbucket commands. `twg`,
its Teamwork Graph CLI, has `twg bitbucket pull-requests` with `create`, `get`, `update`, `merge`, `query`, and
`comment` and `task` subcommands, prints `-o json`, reaches Bitbucket Cloud alone, and ships on a beta channel; its
source is not public, and its flags were not checked. `bkt` is a community CLI for Bitbucket Cloud and Bitbucket Data
Center, modeled on `gh` and maintained: `bkt auth login https://bitbucket.org --kind cloud --web`, or an API token,
since Atlassian removed app passwords on 28 July 2026. Its commands take `--json`, `--yaml`, `--jq`, and
`--template`, and `bkt api "${path}"` calls any REST endpoint. The tables use `bkt`.

### Read from Bitbucket Cloud

| Capability | Command |
| --- | --- |
| Signed in | `bkt auth status` |
| Owner and name | `bkt repo view "${repo}" --json`: the workspace and the repository slug |
| Browser URL | `bkt repo view "${repo}" --json` and its `links.html.href`; a pull request's is `${url}/pull-requests/${n}` |
| Caller's login | `bkt api /user --json` (raw API) |
| Rate limits | `bkt status rate-limit` |
| Marks | `bkt api "/repositories/${workspace}/${repo}/pullrequests" --param sort=-updated_on --param pagelen=1` (raw API) |
| Branch for an issue | `bkt api "/repositories/${workspace}/${repo}/pullrequests" --param q="source.branch.name ~ \"${key}-\" AND state = \"OPEN\""` (raw API) |
| Title, body | `bkt pr view "${n}" --json`: `title`, `description` |
| Author | `bkt pr view "${n}" --json`: `author` |
| State and merge commit | `bkt pr view "${n}" --json`: `state` (`OPEN`, `MERGED`, `DECLINED`, `SUPERSEDED`) and `merge_commit.hash` |
| Change on a branch | `bkt api "/repositories/${workspace}/${repo}/pullrequests" --param q="source.branch.name = \"${branch}\""` (raw API); `bkt pr list` has no branch filter |
| Closing issues | none; a Jira key in the branch, title, or commits links the Jira issue, which Jira reads |
| Review threads | `bkt pr comments "${n}" --details --json`; a comment's `resolution` says it is resolved, and tasks are `bkt pr` tasks |
| Checks rollup | `bkt pr checks "${n}" --json`, the commit statuses of `/pullrequests/${n}/statuses`; `bkt pr mergeability "${n}" --json` for the merge checks |
| Open changes | `bkt pr list --state OPEN --json --limit "${limit}"` |

### Write to Bitbucket Cloud

| Capability | Command |
| --- | --- |
| Branch linked to an issue | `bkt branch create "${key}-${slug}" --from "${default}"`; Jira links it by the key in its name |
| Open a draft | `bkt pr create --draft --source "${branch}" --target "${default}" --title "${title}" --description ""` |
| Mark ready | `bkt pr publish "${n}"`; `--undo` makes it a draft again |
| Set the title | `bkt pr edit "${n}" --title "${title}"` |
| Set the body | `bkt pr edit "${n}" --description "${body}"` |

Gaps: a pull request closes no Bitbucket issue, as Bitbucket Cloud has none; closing is Jira's, through smart
commits or its integration. `bkt` is one maintainer's project, so a kit provider over it carries that risk.

### Bitbucket Cloud schemas

- [Bitbucket Cloud REST API OpenAPI 3.0 description](https://dac-static.atlassian.com/cloud/bitbucket/swagger.v3.json),
  with `/repositories/{workspace}/{repo_slug}/pullrequests`, its `draft` and `merge_commit`, `/statuses`,
  `/comments` (with `resolution`), and `/tasks`, and
  [Bitbucket Cloud REST API Swagger 2.0 description](https://api.bitbucket.org/swagger.json).
- No JSON Schema for webhook payloads or CLI output is published. The pipelines file's schema is on [ci.md](ci.md).

## Bitbucket Data Center

Bitbucket Data Center is the self-hosted product, with its own REST API under `https://${host}/rest/api/latest`.
`bkt` reaches it with `bkt auth login https://${host} --web-token`, a personal access token, and the same commands as
for Cloud, under a project key in place of a workspace. Its pull requests have a `draft` field, in the description
below; which release added it was not checked.

### Read from Bitbucket Data Center

| Capability | Command |
| --- | --- |
| Signed in | `bkt auth status` |
| Owner and name | `bkt repo view "${repo}" --project "${project}" --json` |
| Browser URL | `bkt repo view "${repo}" --project "${project}" --json` and its `links.self`; a pull request's is `${url}/pull-requests/${n}` |
| Caller's login | `bkt auth status`; the REST answer's `X-AUSERNAME` header names the caller (not verified) |
| Rate limits | `bkt status rate-limit` (on Data Center not verified); the administrator sets the limits, under `/rest/api/latest/admin/rate-limit` |
| Marks | `bkt api "/rest/api/latest/projects/${project}/repos/${repo}/pull-requests" --param state=ALL --param order=NEWEST --param limit=1` (raw API); whether `NEWEST` follows updates was not verified |
| Branch for an issue | `bkt api "/rest/api/latest/projects/${project}/repos/${repo}/pull-requests" --param filterText="${key}"` (raw API) |
| Title, body | `bkt pr view "${n}" --json`: `title`, `description` |
| Author | `bkt pr view "${n}" --json`: `author.user.name` |
| State and merge commit | `bkt pr view "${n}" --json`: `state` (`OPEN`, `MERGED`, `DECLINED`); the merge commit is in the `MERGED` activity under `/pull-requests/${n}/activities` (not verified) |
| Change on a branch | `bkt api "/rest/api/latest/projects/${project}/repos/${repo}/pull-requests" --param at="refs/heads/${branch}" --param direction=OUTGOING --param state=ALL` (raw API) |
| Closing issues | none; Jira keys link Jira issues, as on Cloud |
| Review threads | `bkt pr comments "${n}" --details --json`; a comment's `threadResolved`, and blocker comments, the tasks, under `/blocker-comments` |
| Checks rollup | `bkt pr checks "${n}" --json`: the builds of `/rest/api/latest/projects/${project}/repos/${repo}/commits/${sha}/builds` |
| Open changes | `bkt pr list --state OPEN --json` |

### Write to Bitbucket Data Center

| Capability | Command |
| --- | --- |
| Branch linked to an issue | `bkt branch create "${key}-${slug}" --from "${default}"`; Jira links it by the key |
| Open a draft | `bkt pr create --draft --source "${branch}" --target "${default}" --title "${title}"` |
| Mark ready | `bkt pr publish "${n}"` |
| Set the title | `bkt pr edit "${n}" --title "${title}"` |
| Set the body | `bkt pr edit "${n}" --description "${body}"` |

Gaps: the published description has no field for a pull request's merge commit, and no call names the caller
outright. The `bkt` commands were read from its source, not run against a Data Center instance.

### Bitbucket Data Center schemas

- [Bitbucket Data Center 10.5 REST API OpenAPI 3.0 description](https://dac-static.atlassian.com/server/bitbucket/10.5.swagger.v3.json),
  the newest found; `https://dac-static.atlassian.com/server/bitbucket/${version}.swagger.v3.json` names one per
  release, though not every release has one. Its paths are under `/rest`, and `RestPullRequest` has `draft`.
- No JSON Schema for webhook payloads or CLI output is published.

## AWS CodeCommit

`aws` is Amazon's CLI, and `aws codecommit` holds every CodeCommit call. It authenticates with `aws configure`,
`aws sso login`, or the standard `AWS_*` variables, per profile and region. It prints JSON by default, `--query`
filters with JMESPath, and a list reads every page unless `--no-paginate` or `--max-items` stops it. The API is the
CLI: there is no raw-route escape hatch, and none is needed.

AWS closed CodeCommit to new customers in July 2024 and returned it to general availability on 24 November 2025; new
customers can create repositories in 29 regions. See
[AWS CodeCommit returns to general availability](https://aws.amazon.com/blogs/devops/aws-codecommit-returns-to-general-availability/)
and the [CodeCommit document history](https://docs.aws.amazon.com/codecommit/latest/userguide/history.html).
kit's plan for CodeCommit and AWS's other developer services is [aws.md](aws.md).

### Read from AWS CodeCommit

| Capability | Command |
| --- | --- |
| Signed in | `aws sts get-caller-identity` |
| Owner and name | `aws codecommit get-repository --repository-name "${repo}" --query repositoryMetadata.[accountId,repositoryName]` |
| Browser URL | none; the console's is `https://${region}.console.aws.amazon.com/codesuite/codecommit/repositories/${repo}/pull-requests/${n}` |
| Caller's login | `aws sts get-caller-identity --query Arn` |
| Rate limits | none; a throttled call fails with `ThrottlingException`, and the quotas are documented, not read |
| Marks | none cheap; `aws codecommit list-pull-requests` returns IDs alone, and a change's history is `describe-pull-request-events --pull-request-id "${n}"`. Repository triggers to SNS or Lambda (`put-repository-triggers`) and EventBridge push changes instead |
| Branch for an issue | `aws codecommit list-pull-requests --repository-name "${repo}" --pull-request-status OPEN`, then `get-pull-request` on each, for a `sourceReference` of `refs/heads/${n}-*` |
| Title, body | `aws codecommit get-pull-request --pull-request-id "${n}" --query pullRequest.[title,description]` |
| Author | `aws codecommit get-pull-request --pull-request-id "${n}" --query pullRequest.authorArn` |
| State and merge commit | `aws codecommit get-pull-request --pull-request-id "${n}" --query 'pullRequest.[pullRequestStatus, pullRequestTargets[0].mergeMetadata]'`: `OPEN` or `CLOSED`, and `isMerged` with `mergeCommitId` |
| Change on a branch | `list-pull-requests`, then `get-pull-request` on each, matching `pullRequestTargets[].sourceReference` |
| Closing issues | none |
| Review threads | `aws codecommit get-comments-for-pull-request --pull-request-id "${n}"`: comments and replies, with no resolved state |
| Checks rollup | none; `get-pull-request-approval-states` and `evaluate-pull-request-approval-rules` give the approval rules alone |
| Open changes | `aws codecommit list-pull-requests --repository-name "${repo}" --pull-request-status OPEN`, then `get-pull-request` on each |

### Write to AWS CodeCommit

| Capability | Command |
| --- | --- |
| Branch linked to an issue | `aws codecommit create-branch --repository-name "${repo}" --branch-name "${n}-${slug}" --commit-id "${sha}"`, with no link to an issue |
| Open a draft | none; `aws codecommit create-pull-request --title "${title}" --description "" --targets repositoryName="${repo}",sourceReference="${branch}",destinationReference="${default}"` opens one ready for review |
| Mark ready | none |
| Set the title | `aws codecommit update-pull-request-title --pull-request-id "${n}" --title "${title}"` |
| Set the body | `aws codecommit update-pull-request-description --pull-request-id "${n}" --description "${body}"` |

Gaps: CodeCommit has no draft, no issues, no checks on a pull request, and no resolved state on a comment, so the
contract's draft, closing issues, threads, and checks have no answer there. A change for a branch takes a read per
open pull request, as the list returns IDs alone.

### AWS CodeCommit schemas

- [AWS CodeCommit Smithy model](https://raw.githubusercontent.com/aws/api-models-aws/main/models/codecommit/service/2015-04-13/codecommit-2015-04-13.json),
  from aws/api-models-aws, the source the SDKs are generated from.
- [AWS CodeCommit botocore model](https://raw.githubusercontent.com/boto/botocore/develop/botocore/data/codecommit/2015-04-13/service-2.json),
  the shape `aws` itself reads, with its pagination in the same directory.
- No JSON Schema for trigger or EventBridge payloads was found.

## Gitea

`tea` is Gitea's own CLI. It authenticates with `tea login add`, a token per instance, and finds the login from the
checkout's remote. `--output json` (`-o json`) prints JSON, and `--fields` picks the fields of a list. A list takes
`--page` and `--limit`, and reads one page. `tea api "${endpoint}"` calls any endpoint under `/api/v1/`, fills
`{owner}` and `{repo}` from the checkout, and prints headers on standard error with `--include`; it has no
`--paginate`. Gitea marks a pull request as a draft by a `WIP:` prefix on its title.

### Read from Gitea

| Capability | Command |
| --- | --- |
| Signed in | `tea whoami` |
| Owner and name | `tea api 'repos/{owner}/{repo}'`: `full_name` (raw API) |
| Browser URL | `tea api 'repos/{owner}/{repo}'`: `html_url` (raw API); a pull request's is `${url}/pulls/${n}` |
| Caller's login | `tea whoami`, or `tea api user`: `login` |
| Rate limits | none found; gitea.com sent no rate-limit header |
| Marks | `tea api 'repos/{owner}/{repo}/pulls?state=all&sort=recentupdate&limit=1'` (raw API); no ETag was seen |
| Branch for an issue | `tea pulls list --state open --fields index,head --output json` and the head that starts with `${n}-` |
| Title, body | `tea pulls list --state all --fields index,title,body --output json`, or `tea api "repos/{owner}/{repo}/pulls/${n}"` |
| Author | the `author` field of `tea pulls list --fields` |
| State and merge commit | `tea api "repos/{owner}/{repo}/pulls/${n}"` (raw API): `state`, `merged`, and `merge_commit_sha` |
| Change on a branch | `tea pulls list --state all --fields index,state,head,base-commit --output json`, or `tea api "repos/{owner}/{repo}/pulls/${default}/${branch}"` (raw API) |
| Closing issues | none; Gitea closes the issues a merged pull request's body or commits name with a closing keyword, and lists no such link |
| Review threads | `tea pulls review-comments "${n}" --fields id,path,line,resolver --output json`; a comment with a `resolver` is resolved |
| Checks rollup | the `ci` field of `tea pulls list --fields`, or `tea api "repos/{owner}/{repo}/commits/${sha}/status"` (raw API) |
| Open changes | `tea pulls list --state open --output json --limit "${limit}"` |

### Write to Gitea

| Capability | Command |
| --- | --- |
| Branch linked to an issue | `tea api -X POST 'repos/{owner}/{repo}/branches' -f new_branch_name="${n}-${slug}" -f old_ref_name="${default}"` (raw API), with no link to the issue |
| Open a draft | `tea pulls create --draft --head "${branch}" --base "${default}" --title "${title}" --description ""` |
| Mark ready | `tea pulls edit "${n}" --ready`, which strips the `WIP:` prefix |
| Set the title | `tea pulls edit "${n}" --title "${title}"` |
| Set the body | `tea pulls edit "${n}" --description "${body}"`, or `--description-file -` |

Gaps: `tea api` reads one page, so a whole list is read page by page. Gitea keeps no link from a pull request to the
issues it closes until the merge closes them.

### Gitea schemas

- Every Gitea instance serves its own description at `https://${host}/swagger.v1.json` (Swagger 2.0) and
  `https://${host}/openapi3.v1.json` (OpenAPI 3.0); gitea.com's are
  [gitea.com Swagger 2.0 description](https://gitea.com/swagger.v1.json) and
  [gitea.com OpenAPI 3.0 description](https://gitea.com/openapi3.v1.json).
- The source's are
  [Gitea Swagger 2.0 description in the repository](https://raw.githubusercontent.com/go-gitea/gitea/main/templates/swagger/v1-swagger.generated.json)
  and
  [Gitea OpenAPI 3.0 description in the repository](https://raw.githubusercontent.com/go-gitea/gitea/main/templates/swagger/v1-openapi3.generated.json),
  for the next release.
- No JSON Schema for webhook payloads or `tea` output is published. SchemaStore's Gitea entries are for issue
  templates, on [trackers.md](trackers.md).

## Forgejo and Codeberg

Forgejo is a hard fork of Gitea, and Codeberg is the largest public instance of it. Its API is Gitea's, with
additions under `/api/forgejo/v1`, and it drifts from Gitea's release by release. `fj` is the community CLI: it
authenticates with `fj auth login` or `fj auth add-token`, per instance, and has `fj pr create`, `view`, `status`,
`edit`, `search`, and `merge`, all printing text; it has no JSON output and no raw API command. `tea` reaches a
Forgejo instance for the calls the two still share, which Forgejo does not test (not verified). The tables use `fj`
where it has the command, and the raw API through `tea api` or `curl` elsewhere.

### Read from Forgejo and Codeberg

| Capability | Command |
| --- | --- |
| Signed in | `fj whoami` |
| Owner and name | `fj repo view`, as text; `tea api 'repos/{owner}/{repo}'` for JSON (raw API) |
| Browser URL | `html_url` of `tea api 'repos/{owner}/{repo}'` (raw API) |
| Caller's login | `fj whoami` |
| Rate limits | the `RateLimit` and `RateLimit-Policy` headers Codeberg sends with every answer, read by `tea api --include` (raw API); a self-hosted Forgejo sets its own |
| Marks | `tea api 'repos/{owner}/{repo}/pulls?state=all&sort=recentupdate&limit=1'` (raw API) |
| Branch for an issue | `tea api 'repos/{owner}/{repo}/pulls?state=open'` and the head that starts with `${n}-` (raw API) |
| Title, body | `fj pr view "${n}"`, and `fj pr view "${n}" body` |
| Author | `fj pr view "${n}"`, as text |
| State and merge commit | `tea api "repos/{owner}/{repo}/pulls/${n}"` (raw API): `merged`, `merge_commit_sha` |
| Change on a branch | `tea api "repos/{owner}/{repo}/pulls/${default}/${branch}"` (raw API) |
| Closing issues | none, as on Gitea |
| Review threads | `tea api "repos/{owner}/{repo}/pulls/${n}/reviews"` and each review's `comments`, with `resolver` (raw API) |
| Checks rollup | `fj pr status "${n}"`, as text |
| Open changes | `fj pr search --state open`, as text |

### Write to Forgejo and Codeberg

| Capability | Command |
| --- | --- |
| Branch linked to an issue | `tea api -X POST 'repos/{owner}/{repo}/branches' -f new_branch_name="${n}-${slug}" -f old_ref_name="${default}"` (raw API), with no link to the issue |
| Open a draft | `fj pr create --base "${default}" --head "${branch}" "WIP: ${title}"`; Forgejo, like Gitea, marks a draft by the prefix |
| Mark ready | `fj pr edit "${n}" title "${title}"`, without the prefix |
| Set the title | `fj pr edit "${n}" title "${title}"` |
| Set the body | `fj pr edit "${n}" body "${body}"` |

Gaps: `fj` prints no JSON, so a kit provider would read the API for every read. A provider over `tea` covers more,
but rests on an API the two projects no longer keep the same.

### Forgejo and Codeberg schemas

- Every Forgejo instance serves `https://${host}/swagger.v1.json` (Swagger 2.0); Codeberg's is
  [Codeberg Swagger 2.0 description](https://codeberg.org/swagger.v1.json). Forgejo serves no OpenAPI 3.0 document.
- The source's is
  [Forgejo Swagger 2.0 description in the repository](https://codeberg.org/forgejo/forgejo/raw/branch/forgejo/templates/swagger/v1_json.tmpl),
  a template the server fills with its URL.
- No JSON Schema for webhook payloads or `fj` output is published.

## Gerrit

Gerrit reviews changes rather than branches: each commit is a change, named by the `Change-Id` trailer in its
message, and a new push of the commit is a new patch set of the same change. A change's title is its commit's
subject, and its body the rest of the message. There is no draft; a change pushed as work in progress waits there.
Gerrit has two command lines. Its SSH daemon answers `ssh -p 29418 "${host}" gerrit …`, and
`gerrit query --format=JSON` prints one JSON object per line. `git-review`, the OpenStack project's community tool,
pushes a branch to `refs/for/${branch}`. Everything else is the REST API at `https://${host}/a/`, with HTTP
credentials from the user's settings; each JSON answer starts with the line `)]}'`, which a reader drops.

### Read from Gerrit

| Capability | Command |
| --- | --- |
| Signed in | `ssh -p 29418 "${host}" gerrit version`; `curl --user "${user}:${password}" "https://${host}/a/accounts/self"` (raw API) |
| Owner and name | the project's name, from the remote's path; `GET /a/projects/${project}` (raw API) |
| Browser URL | none; a change's is `https://${host}/c/${project}/+/${n}` |
| Caller's login | `GET /a/accounts/self` (raw API): `username` |
| Rate limits | none in Gerrit itself; the quota plugin adds some |
| Marks | `GET /a/changes/?q=project:${project}&n=1` (raw API); Gerrit lists the change updated last first. No ETag was seen |
| Branch for an issue | `ssh -p 29418 "${host}" gerrit query --format=JSON "status:open project:${project} topic:${n}"`, when the change's topic names the issue |
| Title, body | `gerrit query --format=JSON "change:${n}"`: `subject`; `GET /a/changes/${n}/message` (raw API) for the whole message |
| Author | `gerrit query --format=JSON "change:${n}"`: `owner.username` |
| State and merge commit | `gerrit query --format=JSON --current-patch-set "change:${n}"`: `status` (`NEW`, `MERGED`, `ABANDONED`) and the current patch set's `revision`, which is the merged commit unless the submit rebased it |
| Change on a branch | `gerrit query --format=JSON "change:${change_id}"`, with the `Change-Id` of the branch's head commit |
| Closing issues | none; an its-* plugin links an issue tracker by a footer in the message |
| Review threads | `GET /a/changes/${n}/comments` (raw API): each comment's `unresolved`, a thread's state being its last comment's |
| Checks rollup | `gerrit query --format=JSON --submit-records "change:${n}"`, the labels and submit requirements; `GET /a/changes/${n}/detail` (raw API). CI results are votes on a label such as `Verified` |
| Open changes | `gerrit query --format=JSON "status:open project:${project}"` |

### Write to Gerrit

| Capability | Command |
| --- | --- |
| Branch linked to an issue | none; a local branch, and `git review --topic "${n}"` |
| Open a draft | `git review --wip "${default}"`, or `git push origin "HEAD:refs/for/${default}%wip"` |
| Mark ready | `git review --ready` on the next push, or `POST /a/changes/${n}/ready` (raw API) |
| Set the title | `PUT /a/changes/${n}/message` (raw API), which makes a new patch set; or amend the commit and push again |
| Set the body | `PUT /a/changes/${n}/message` (raw API) |

Gaps: Gerrit's change is a commit, not a branch, so the contract's branch lookups go through the `Change-Id`, and
`kit work`'s `Closes` lines would go in the commit message. `gerrit query` prints no unresolved comment state.

### Gerrit schemas

- Gerrit publishes no OpenAPI document or GraphQL schema. The REST API is described in prose at
  [Gerrit REST API](https://gerrit-review.googlesource.com/Documentation/rest-api.html), and the query command at
  [gerrit query](https://gerrit-review.googlesource.com/Documentation/cmd-query.html).

## SourceHut

SourceHut reviews changes by email: a contributor sends a patch series to a mailing list with `git send-email`, and
lists.sr.ht keeps it as a patchset with a status, one of `PROPOSED`, `NEEDS_REVISION`, `SUPERSEDED`, `APPROVED`,
`REJECTED`, or `APPLIED`. A patchset is not tied to a branch, has no draft, and has no review threads beyond the
replies on the list. `hut` is the community CLI: `hut init` stores an OAuth token per instance. Its commands print
text, and `hut graphql "${service}"` runs a GraphQL query on one service, `git`, `lists`, or `meta`, and prints the
JSON answer.

### Read from SourceHut

| Capability | Command |
| --- | --- |
| Signed in | `hut meta show` |
| Owner and name | `hut git show`, as text |
| Browser URL | none; a repository's is `https://git.sr.ht/~${owner}/${repo}` |
| Caller's login | `hut meta show`, or `hut graphql meta <<<'query { me { canonicalName } }'` |
| Rate limits | none documented |
| Marks | `hut graphql lists` on a list's `patches` and their `updated` (raw API) |
| Branch for an issue | none |
| Title, body | `hut lists patchset show "${id}"`: the subject and the cover letter |
| Author | `hut lists patchset show "${id}"`: the submitter |
| State and merge commit | `hut lists patchset show "${id}"`: the status; an applied patchset names no commit |
| Change on a branch | none |
| Closing issues | none; todo.sr.ht reads `Fixes:` and `Implements:` trailers in commits pushed to git.sr.ht (not verified) |
| Review threads | none; replies on the list have no resolved state |
| Checks rollup | none from `hut`; builds.sr.ht can test a patchset sent to a list set up for it (not verified) |
| Open changes | `hut lists patchset list "${list}"`, which shows the `PROPOSED` ones by default |

### Write to SourceHut

| Capability | Command |
| --- | --- |
| Branch linked to an issue | none |
| Open a draft | none; `git send-email --to="${list_address}" "${default}"` sends the series ready for review, and `--rfc` marks it a request for comments |
| Mark ready | none |
| Set the title | none; a new version of the series |
| Set the body | none; a new version of the series |
| Set the status | `hut lists patchset update "${id}" --status APPROVED` |

Gaps: the contract's change is a branch with a body that can change, and a SourceHut patchset is neither, so most of
the change calls have no answer. A provider would read patchsets alone, through `hut graphql`.

### SourceHut schemas

- [git.sr.ht GraphQL schema](https://git.sr.ht/~sircmpwn/git.sr.ht/blob/master/api/graph/schema.graphqls),
  [lists.sr.ht GraphQL schema](https://git.sr.ht/~sircmpwn/lists.sr.ht/blob/master/api/graph/schema.graphqls), and
  [meta.sr.ht GraphQL schema](https://git.sr.ht/~sircmpwn/meta.sr.ht/blob/master/api/graph/schema.graphqls), each in
  the public domain or CC0. todo.sr.ht's is on [trackers.md](trackers.md), and builds.sr.ht's manifest schema on
  [ci.md](ci.md).

## Radicle

Radicle is a peer-to-peer code forge: each node keeps the repositories it follows, and issues and patches are
collaborative objects stored in git and spread by seeds. `rad` is its own CLI, and the identity is a key, created with
`rad auth`. A patch is opened by a push to the `rad` remote, `git push rad HEAD:refs/patches`, with push options for
draft, base, and message. `rad patch` prints text; `rad cob show` prints a patch as JSON, by default. Reads are local,
so they cost no request and count against no limit, and `rad sync` fetches what the seeds have.

### Read from Radicle

| Capability | Command |
| --- | --- |
| Signed in | `rad self` |
| Owner and name | `rad inspect`: the repository's ID, `rad:…` |
| Browser URL | none; a seed's web view, such as [Radicle's web view](https://app.radicle.xyz), shows the repository |
| Caller's login | `rad self --did` |
| Rate limits | none |
| Marks | none; reads are local, after `rad sync` |
| Branch for an issue | none; a patch records its head commit, not a branch name |
| Title, body | `rad patch show "${id}"`, or `rad cob show --repo "${rid}" --type xyz.radicle.patch --object "${id}"` for JSON |
| Author | `rad cob show --repo "${rid}" --type xyz.radicle.patch --object "${id}"`: the author's DID |
| State and merge commit | `rad cob show` on the patch: its state (`draft`, `open`, `archived`, `merged`) and the merge's commit |
| Change on a branch | `rad patch list --all`, and the patch whose head is the branch's commit |
| Closing issues | none |
| Review threads | `rad cob show` on the patch: its reviews' comments and their resolution |
| Checks rollup | none in `rad`; a CI broker writes its results elsewhere (not verified) |
| Open changes | `rad patch list --open`, as text |

### Write to Radicle

| Capability | Command |
| --- | --- |
| Branch linked to an issue | none |
| Open a draft | `git push -o patch.draft -o patch.message="${title}" rad HEAD:refs/patches` |
| Mark ready | `rad patch ready "${id}"`; `--undo` makes it a draft again |
| Set the title | `rad patch edit "${id}" --message "${title}"` |
| Set the body | `rad patch edit "${id}" --message "${title}" --message "${body}"`; the messages are joined with newlines |

Gaps: a patch has no branch and no closing issues, and `rad patch` prints no JSON. A patch's ID is a hash, where the
contract's change is a number.

### Radicle schemas

- Radicle publishes no OpenAPI document for its node's HTTP API, and no JSON Schema for its collaborative objects; the
  types are in the Rust source of [radicle-dev/heartwood](https://github.com/radicle-dev/heartwood).

## Coverage

| Capability | GitHub | GitLab | Azure DevOps | Bitbucket Cloud | Bitbucket DC | CodeCommit | Gitea | Forgejo | Gerrit | SourceHut | Radicle |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| Signed in | cli | cli | cli | cli | cli | cli | cli | cli | cli | cli | cli |
| Owner and name | cli | cli | cli | cli | cli | cli | api | cli | api | cli | cli |
| Browser URL | cli | cli | cli | cli | cli | — | api | api | — | — | — |
| Caller's login | api | api | cli | api | cli | cli | cli | cli | api | cli | cli |
| Rate limits | api | api | — | cli | api | — | — | api | — | — | — |
| Marks | api | api | — | api | api | — | api | api | api | api | — |
| Branch linked to an issue | cli | api | cli | cli | cli | — | — | — | — | — | — |
| Branch for an issue | cli | cli | cli | api | api | cli | cli | api | cli | — | — |
| Open a draft | cli | cli | cli | cli | cli | — | cli | cli | cli | — | cli |
| Mark ready | cli | cli | cli | cli | cli | — | cli | cli | cli | — | cli |
| Title, body | cli | cli | cli | cli | cli | cli | cli | cli | cli | cli | cli |
| Set the body | cli | cli | cli | cli | cli | cli | cli | cli | api | — | cli |
| Author | api | cli | cli | cli | cli | cli | cli | cli | cli | cli | cli |
| State and merge commit | cli | cli | cli | cli | api | cli | api | api | cli | — | cli |
| Change on a branch | cli | cli | cli | api | api | cli | cli | api | cli | — | cli |
| Closing issues | cli | api | cli | — | — | — | — | — | — | — | — |
| Review threads, resolved | api | cli | api | cli | cli | — | cli | api | api | — | cli |
| Checks rollup | cli | cli | cli | cli | cli | — | cli | cli | cli | — | — |
| Open changes | cli | cli | cli | cli | cli | cli | cli | cli | cli | cli | cli |

Notes on the cells:

- GitHub's author is `api` because `gh pr view` names a bot by its app, not by its login.
- Azure DevOps' marks are `—`: no read was found that changes on every update to a pull request.
- Bitbucket's branch link and closing issues are Jira's: Bitbucket Cloud has no issues, and Data Center never did.
- Bitbucket Data Center's merge commit is `api` from the activities, and its rate limits `api` (both not verified).
- Gitea's and Forgejo's branch is made through the API with no link to the issue; a closing keyword in the body
  closes it at merge.
- Forgejo's `cli` cells are text alone: `fj` prints no JSON.
- Gerrit's branch for an issue is a topic that names it, a convention rather than a link. Its draft is a change
  marked work in progress.
- SourceHut's open changes are patchsets on a list, and its marks are the patchsets' `updated` in GraphQL.
- Radicle's `cli` reads are `rad cob show` JSON, except open changes, which `rad patch list` prints as text.
