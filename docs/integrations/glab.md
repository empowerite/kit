# glab

`glab` is GitLab's own CLI. It authenticates with `glab auth login`, or `GITLAB_TOKEN`, and a CI job token inside
GitLab CI. `--output json` (`-F json`) prints JSON, `--jq` filters it, and `--page` and `--per-page` page a list.
`glab api` is the escape hatch: it fills `:id`, `:fullpath`, `:repo`, and `:branch` from the checkout, reads every
page with `--paginate`, prints headers with `--include`, and runs GraphQL with `glab api graphql`. kit reads the REST
API through `glab api` for every call but the draft, and pages each list itself.

This page says what `glab` reads and writes where it meets kit's scope, for each service it reaches, and the
machine-readable descriptions each publishes. The index of these pages is [1st.md](1st.md).

| Service | Category | kit status |
| --- | --- | --- |
| [GitLab merge requests](#gitlab-merge-requests) | [code forge](forges.md) | `provider` |
| [GitLab issues and work items](#gitlab-issues-and-work-items) | [tracker](trackers.md) | `provider` (`forge`) |
| [GitLab CI/CD](#gitlab-cicd) | [CI](ci.md) | `provider`, through `forge/gitlab.sh` |
| [GitLab releases and deployments](#gitlab-releases-and-deployments) | [CD](cd.md) | `provider`: releases, environments, and deployments |

## GitLab merge requests

A self-managed GitLab answers at `https://${host}/api/v4` and `https://${host}/api/graphql`. `glab` reaches it with
`glab auth login --hostname "${host}"`, or `GITLAB_HOST`, and kit finds such a host in `glab`'s `config.yml`. Its
administrator sets its rate limits, so its `RateLimit-*` headers may be absent. Its version decides which endpoints
and GraphQL fields exist, and features of the paid tiers are absent from Community Edition.

### Read from GitLab merge requests

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
| Reviewers | each open merge request's `reviewers`, from `glab api "projects/:id/merge_requests/${n}"`, and who approved it and when, from `glab api "projects/:id/merge_requests/${n}/approvals"` (raw API), which every tier answers: a request per open merge request, beside its discussions, made by every read of the open changes, so `kit flow changes`, `underway`, and `backlog` pay it as `kit changes reviews` does |
| Merged changes | `glab api 'projects/:id/merge_requests?state=merged&order_by=merged_at&per_page=51'` (raw API): the 50 merged last, and a 51st that says there are more; `order_by=merged_at` is GitLab 17.2's, and an older GitLab, which refuses it naming `order_by`, is asked once more with `order_by=updated_at&sort=desc&per_page=51`, of whose first 50 kit keeps those merged since the oldest `updated_at`, saying `limited` when there are more; `kit changes merged` draws them |

### Write to GitLab merge requests

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

### GitLab merge requests schemas

- [GitLab REST API OpenAPI 3.0 description](https://gitlab.com/gitlab-org/gitlab/-/raw/master/doc/api/openapi/openapi_v3.yaml),
  generated from the source, with `/api/v4/projects/{id}/merge_requests` and its `closes_issues` and `discussions`.
  [GitLab REST API OpenAPI 2.0 description](https://gitlab.com/gitlab-org/gitlab/-/raw/master/doc/api/openapi/openapi_v2.yaml)
  is deprecated and no longer updated.
- [GitLab GraphQL schema](https://gitlab.com/gitlab-org/gitlab/-/jobs/artifacts/master/raw/tmp/tests/graphql/gitlab_schema.graphql?job=graphql-schema-dump),
  the `graphql-schema-dump` job's artifact on the default branch, so it describes GitLab.com's next release, not a
  self-managed instance's. Its `MergeRequest` has `discussions`, `headPipeline`, and `draft`.
- No JSON Schema for webhook payloads or for `glab` output is published. The CI file's schema is under
  [GitLab CI/CD schemas](#gitlab-cicd-schemas).

## GitLab issues and work items

`glab` is GitLab's own CLI. `glab auth login` stores a token per host; `GITLAB_TOKEN` overrides it. `--output json`
(`-O` on `issue list`, `-F` elsewhere) prints JSON and `--jq` filters it. `glab api` reaches the REST API, with `:id`
and `:fullpath` standing for the current project and `--paginate` reading every page, and `glab api graphql` reaches
the GraphQL API; `glab api` has no `--jq`, so its answer goes to `jq`. `glab issue list` reads one page per call
(`--page`, `--per-page`), and kit's `forge/gitlab.sh` reads issues through `glab api` instead. Epics, iterations, and
work item statuses need GitLab Premium or Ultimate.

### Read from GitLab issues and work items

| Capability | Command |
| --- | --- |
| Title | `glab issue view "${n}" --output json --jq .title` |
| State | `glab issue view "${n}" --output json --jq .state` (`opened` or `closed`) |
| Assignees | `glab issue view "${n}" --output json --jq '.assignees[].username'` |
| Caller | `glab api user`, its `username` |
| Issue page | `glab issue view "${n}" --output json --jq .web_url` |
| New-issue page | none; kit builds `${project_url}/-/issues/new?issue[title]=${title}` |
| Open issues | `glab api --paginate 'projects/:id/issues?state=opened&order_by=created_at&sort=asc&with_labels_details=true'`, as kit reads them; or `glab issue list --output json --per-page 100 --page "${page}"` |
| Linked items | `glab api "projects/:id/issues/${n}/links"` (raw REST) |
| Merge requests that close it | `glab api "projects/:id/issues/${n}/closed_by"` (raw REST) |
| Epics and other work items | `glab work-items list --group "${group}" --type epic --output json` (experimental) |
| Milestones | `glab api graphql -f path="${project}" -f query="${query}"`, `project(fullPath:) { milestones(first: 100) { nodes { iid title state dueDate webPath stats { totalIssuesCount closedIssuesCount } } } }`, as kit reads them, page after page by `endCursor`, since the REST list counts no issues; or `glab milestone list --output json`; `kit issues milestones` draws them, and its mark is the newest milestone's `updated_at` with the newest issue's, from `glab api 'projects/:id/milestones?sort=updated_desc&per_page=1'` |
| Iterations | `glab api --paginate 'projects/:id/iterations?include_ancestors=true&per_page=100'`, as kit reads them, where a tier without iterations answers 403 or 404, which kit reads as off; or `glab iteration list --output json`; `kit issues iterations` draws them |
| Work item status | `glab api graphql -f query="${query}"`, the `status` widget of `workItem` (raw GraphQL) |

### Write to GitLab issues and work items

| Capability | Command |
| --- | --- |
| Assign | `glab issue update "${n}" --assignee "+${login}"` |
| Unassign | `glab issue update "${n}" --assignee "-${login}"` |
| Close as completed or not planned | `glab issue close "${n}"`; GitLab keeps no reason |
| Close as a duplicate | `glab issue note "${n}" --message "/duplicate #${m}"`, a quick action that links and closes it |
| Comment | `glab issue note "${n}" --message "${text}"` |
| Create | `glab issue create --title "${title}" --description "${body}" --yes` |
| Put under a parent | kit links it: `glab api --method POST "projects/:id/issues/${n}/links" -f target_project_id="${project_id}" -f target_issue_iid="${m}"`; a work item takes a parent with `glab work-items update "${m}" --parent "${n}"` (experimental) |
| Link a branch | `glab api --method POST "projects/:id/repository/branches" -f branch="${n}-${slug}" -f ref="${default_branch}"`; GitLab links a branch named `${n}-${slug}` to the issue |
| Link a change | `Closes #${n}` in the merge request's description |
| Set the milestone | `glab issue update "${n}" --milestone "${milestone}"` |
| Set the iteration | `glab work-items update "${n}" --iteration "${iteration_id}"` (experimental) |

Gaps: an issue cannot be the child of another issue. GitLab's hierarchy puts issues under epics and tasks under
issues, so kit links gathered issues as related items. The Epics REST API is deprecated in favor of work items, and
`glab work-items` is experimental. GitLab's default closing pattern accepts `Close`, `Fix`, `Resolve`, and `Implement`
in their usual forms; see [managing issues](https://docs.gitlab.com/user/project/issues/managing_issues/).

### GitLab issues and work items schemas

- [GitLab REST API OpenAPI 3.0](https://gitlab.com/gitlab-org/gitlab/-/raw/master/doc/api/openapi/openapi_v3.yaml),
  generated from the API's own definitions; the OpenAPI 2.0 file beside it says it is deprecated
- [GitLab GraphQL introspection result](https://gitlab.com/-/graphql/introspection_result.json) and
  [the same without deprecated fields](https://gitlab.com/-/graphql/introspection_result_no_deprecated.json), both
  served by every instance at the same path
- No JSON Schema for issue templates (`.gitlab/issue_templates/` holds Markdown) or for webhook payloads.

## GitLab CI/CD

`glab` is GitLab's CLI. It signs in with `glab auth login` or reads `GITLAB_TOKEN`. `glab ci list`, `glab ci get`, and
`glab ci status` take `-F json` and `--jq`; lists page with `-p` and `-P`. `glab api` calls any REST route, fills
`:id` with the project, and reads every page with `--paginate`. A GitLab job has no steps, so the failed step is the
failed job and its `failure_reason`.

### Read from GitLab CI/CD

| Capability | Command |
| --- | --- |
| Recent runs | `glab ci list -F json -P 50`; filter with `--source push`, `--source schedule`, `--source web`, `--scope tags`, `-r "${branch}"`; kit reads `glab api 'projects/:id/pipelines?per_page=50'` |
| A run's jobs | `glab ci get -p "${pipeline_id}" -d -F json`, or `glab api projects/:id/pipelines/${pipeline_id}/jobs` |
| A change's checks | `glab ci get --merge-request "${n}" -F json`; kit reads the merge request's head pipeline and its jobs |
| A job's log | `glab ci trace "${job_id}"`, which follows a running job; `glab api projects/:id/jobs/${job_id}/trace` for the raw text |
| Artifacts | `glab job artifact "${ref}" "${job_name}"` downloads the latest; `--list-paths` lists them; `glab api projects/:id/jobs/${job_id}/artifacts` for one job; kit reads `glab api 'projects/:id/jobs?scope[]=success&per_page=100'` (raw API), the 100 newest successful jobs, each with an `artifacts_file` an archive, with no digest GitLab shows, and `kit delivery artifacts` draws them, each named `<job>: <file>` |

### Write to GitLab CI/CD

| Capability | Command |
| --- | --- |
| Rerun | a job: `glab ci retry "${job_id}"`; a pipeline's failed jobs: `glab api -X POST projects/:id/pipelines/${pipeline_id}/retry` |
| Cancel | `glab ci cancel pipeline "${pipeline_id}"`; `glab ci cancel job "${job_id}"` |
| Start a run | `glab ci run -b "${branch}"`, with `--variables` or `--input`; a manual job: `glab ci trigger "${job_id}"` |
| Report a status | `glab api -X POST projects/:id/statuses/${sha} -f state=success -f name=kit -f target_url="${url}"` |

`glab ci lint` validates `.gitlab-ci.yml` on the server, and `glab ci config compile` prints it with its includes
merged. Retrying a whole pipeline has no `glab` command.

### GitLab CI/CD schemas

- [GitLab CI JSON Schema](https://gitlab.com/gitlab-org/gitlab/-/raw/master/app/assets/javascripts/editor/schema/ci.json),
  for `.gitlab-ci.yml`; SchemaStore's "gitlab-ci" entry points at the
  [same file in gitlab-foss](https://gitlab.com/gitlab-org/gitlab-foss/-/raw/master/app/assets/javascripts/editor/schema/ci.json)
- [GitLab REST OpenAPI 3](https://gitlab.com/gitlab-org/gitlab/-/raw/master/doc/api/openapi/openapi_v3.yaml), with the
  pipelines, jobs, and commit statuses routes, and the
  [Swagger 2 form](https://gitlab.com/gitlab-org/gitlab/-/raw/master/doc/api/openapi/openapi_v2.yaml)
- The GraphQL schema is read by introspection at `/api/graphql`; no static file was verified.

## GitLab releases and deployments

`glab` is GitLab's own CLI. It authenticates with `glab auth login`, or `GITLAB_TOKEN`. `--output json` (`-F json`)
prints JSON, `--jq` filters it, and `--page` and `--per-page` page a list. It has release commands and none for
environments or deployments, which go through `glab api`. `glab api` fills `:id` with the project, and reads every
page with `--paginate`. kit reads releases with `glab api` and pages them itself. Each release in that list carries
its notes, `description`; its page, `_links.self`; and its assets: `assets.links`, the files a release offers, which
kit lists with each one's `direct_asset_url`, and `assets.sources`, the archives GitLab makes of every tag, which kit
leaves out, as GitHub's list leaves out its own.

A job with `environment:` in `.gitlab-ci.yml` creates a deployment when it runs. A protected environment can require
approvals before its deployment jobs run, on GitLab Premium and Ultimate. GitLab links a deployment to the merge
requests it shipped, and a merge request shows the environments its pipeline deployed to.

### Read from GitLab releases and deployments

| Capability | Command |
| --- | --- |
| List releases, with their notes, pages, and assets | `glab api --paginate 'projects/:id/releases?per_page=100'` (raw API), or `glab release list --output json --per-page 100 --page "${page}"` |
| A release's notes | `glab release view "${tag}" --output json` |
| List environments | `glab api --paginate 'projects/:id/environments?per_page=100'` (raw API); `kit delivery environments` draws them |
| Who may deploy to and approve an environment | `glab api --paginate 'projects/:id/protected_environments?per_page=100'` (raw API): `deploy_access_levels`, `required_approval_count`, and `approval_rules`, on Premium and Ultimate; another tier answers 403 or 404, and kit leaves the protection unknown |
| Deployments in an environment | `glab api --paginate "projects/:id/deployments?environment=${environment}&order_by=finished_at&sort=desc"` (raw API) |
| Whether deployments or environments changed | `glab api 'projects/:id/deployments?order_by=updated_at&sort=desc&per_page=1'` and `glab api 'projects/:id/environments?per_page=100'` (raw API): what they say of when they changed, as kit's deployments and environments marks do; a protected environment's edit moves neither |
| The newest deployments | `glab api 'projects/:id/deployments?order_by=created_at&sort=desc&per_page=51'` (raw API): the 50 newest, and a 51st that says there are more; `kit delivery deployments` draws them |
| A deployment's state and approvals | `glab api "projects/:id/deployments/${deployment_id}"` (raw API); `status`, `approvals`, `pending_approval_count` |
| What is live where | `glab api "projects/:id/environments/${environment_id}"` (raw API); `last_deployment` |
| Merge requests a deployment shipped | `glab api "projects/:id/deployments/${deployment_id}/merge_requests"` (raw API) |

### Write to GitLab releases and deployments

| Capability | Command |
| --- | --- |
| Create a release from a tag with notes | `glab release create "${tag}" --notes-file "${file}"`; `--ref "${sha}"` makes the tag when it does not exist |
| Generate notes | `glab changelog generate --version "${version}"` |
| Deploy or promote | `glab ci trigger "${job_id}"`, for a manual deployment job |
| Roll back | `glab ci retry "${job_id}"` on the job of the earlier deployment |
| Stop an environment | `glab api -X POST "projects/:id/environments/${environment_id}/stop"` (raw API) |
| Approve or reject | `glab api -X POST "projects/:id/deployments/${deployment_id}/approval" -f status=approved -f comment="${comment}"` (raw API) |
| Record a deployment made elsewhere | `glab api -X POST 'projects/:id/deployments' -f environment="${environment}" -f sha="${sha}" -f ref="${ref}" -F tag=false -f status=running` (raw API) |
| Report its status | `glab api -X PUT "projects/:id/deployments/${deployment_id}" -f status=success` (raw API) |

Gaps: environments, deployments, and approvals have no `glab` command. GitLab marks no release as a pre-release, so
kit treats a tag with a suffix as one. An environment restricts who deploys, not from which branch, and the REST API
shows no digest of a job's artifacts.

### GitLab releases and deployments schemas

- [GitLab REST OpenAPI 3.0 description](https://gitlab.com/gitlab-org/gitlab/-/raw/master/doc/api/openapi/openapi_v3.yaml),
  generated from the API's definitions; it holds `/api/v4/projects/{id}/releases`, `/environments`, and
  `/deployments`. The OpenAPI 2.0 file beside it, `openapi_v2.yaml`, says it is deprecated.
- [GitLab CI JSON Schema](https://gitlab.com/gitlab-org/gitlab/-/raw/master/app/assets/javascripts/editor/schema/ci.json),
  for `environment:` and `release:`. SchemaStore's "gitlab-ci" entry points at the
  [GitLab FOSS copy of the CI schema](https://gitlab.com/gitlab-org/gitlab-foss/-/raw/master/app/assets/javascripts/editor/schema/ci.json).
- [GitLab agent configuration JSON Schema](https://gitlab.com/gitlab-org/cluster-integration/gitlab-agent/-/raw/master/pkg/agentcfg/agentcfg_schemas/gitlab.agent.agentcfg.ConfigurationFile.jsonschema.bundle.json),
  SchemaStore's entry for `.gitlab/agents/*/config.yaml`, the agent that connects a cluster and its Flux to GitLab.
- GraphQL: GitLab publishes no schema file; a GitLab instance answers introspection at `/api/graphql`.

## Schemas of its output

`glab` publishes no schema of what its commands print, so each command kit runs whose JSON kit reads has one
here, in [glab/](glab/): a JSON Schema of the fields kit reads, typed from the provider's API description,
with an example the gate validates against it.

| Schema | The command as kit runs it |
| --- | --- |
| [api-iterations.schema.json](glab/api-iterations.schema.json) | `glab api projects/${project}/iterations?include_ancestors=true&per_page=100 --paginate` |
| [api-deployments.schema.json](glab/api-deployments.schema.json) | `glab api projects/${project}/deployments?order_by=${order}&sort=desc&per_page=${n}` |
| [api-environments.schema.json](glab/api-environments.schema.json) | `glab api projects/${project}/environments?per_page=100 [--paginate]` |
| [api-graphql-milestones.schema.json](glab/api-graphql-milestones.schema.json) | `glab api graphql --field path=${project} [--field after=${cursor}] --raw-field query=${query}` |
| [api-issue.schema.json](glab/api-issue.schema.json) | `glab api projects/${project}/issues/${n}` |
| [api-issues.schema.json](glab/api-issues.schema.json) | `glab api projects/${project}/issues?${query} [--paginate]` |
| [api-jobs.schema.json](glab/api-jobs.schema.json) | `glab api projects/${project}/jobs?scope[]=success&per_page=100` |
| [api-labels.schema.json](glab/api-labels.schema.json) | `glab api projects/${project}/labels?per_page=100 --paginate` |
| [api-merge-request-approvals.schema.json](glab/api-merge-request-approvals.schema.json) | `glab api projects/${project}/merge_requests/${n}/approvals` |
| [api-merge-request-closes-issues.schema.json](glab/api-merge-request-closes-issues.schema.json) | `glab api projects/${project}/merge_requests/${n}/closes_issues?per_page=100 --paginate` |
| [api-merge-request-discussions.schema.json](glab/api-merge-request-discussions.schema.json) | `glab api projects/${project}/merge_requests/${n}/discussions?per_page=100 --paginate` |
| [api-merge-request.schema.json](glab/api-merge-request.schema.json) | `glab api projects/${project}/merge_requests/${n}` |
| [api-merge-requests.schema.json](glab/api-merge-requests.schema.json) | `glab api projects/${project}/merge_requests?${query} [--paginate]` |
| [api-milestones.schema.json](glab/api-milestones.schema.json) | `glab api projects/${project}/milestones?sort=updated_desc&per_page=1` |
| [api-pipeline-jobs.schema.json](glab/api-pipeline-jobs.schema.json) | `glab api projects/${home}/pipelines/${pipeline}/jobs?per_page=100 --paginate` |
| [api-pipelines.schema.json](glab/api-pipelines.schema.json) | `glab api projects/${project}/pipelines?per_page=${n}` |
| [api-project.schema.json](glab/api-project.schema.json) | `glab api projects/${source}` |
| [api-protected-environments.schema.json](glab/api-protected-environments.schema.json) | `glab api projects/${project}/protected_environments?per_page=100 --paginate` |
| [api-releases.schema.json](glab/api-releases.schema.json) | `glab api projects/${project}/releases?per_page=${n} [--paginate]` |
| [api-user.schema.json](glab/api-user.schema.json) | `glab api user` |
