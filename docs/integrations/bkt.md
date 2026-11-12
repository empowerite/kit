# bkt

This page says what `bkt` reads and writes where it meets kit's scope, for each service it reaches, and the
machine-readable descriptions each publishes. The index of these pages is [1st.md](1st.md).

| Service | Category | kit status |
| --- | --- | --- |
| [Bitbucket Cloud](#bitbucket-cloud) | [code forge](forges.md) | `none` |
| [Bitbucket Data Center](#bitbucket-data-center) | [code forge](forges.md) | `none` |
| [Bitbucket Pipelines](#bitbucket-pipelines) | [CI](ci.md) | `none` |

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
- No JSON Schema for webhook payloads or CLI output is published. The pipelines file's schema is under
  [Bitbucket Pipelines schemas](#bitbucket-pipelines-schemas).

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

## Bitbucket Pipelines

Bitbucket Pipelines is Bitbucket Cloud's own CI; Bitbucket Data Center has none and points to Bamboo. Its REST API
lives under `https://api.bitbucket.org/2.0/repositories/${workspace}/${repo}`, signs in with an API token (app
passwords are retired), and pages with `page` and `pagelen`. A run's `trigger_type` is `PUSH`, `MANUAL`, `SCHEDULED`,
or `PARENT_STEP`, and its `target.selector.type` tells a pull request from a branch or a tag. Atlassian's `acli` has
no Bitbucket commands. Its Teamwork Graph CLI, `twg`, lists `twg bitbucket pipeline query`, `get`, `run`, `trigger`,
`rerun-failed`, `tail`, `grep`, and `latest-failure`; their flags, their JSON, and whether the CLI is past beta were
not verified, so the tables give REST.

### Read from Bitbucket Pipelines

| Capability | Command |
| --- | --- |
| Recent runs | `GET /pipelines?sort=-created_on&trigger_type=PUSH` (REST); `twg bitbucket pipeline query` |
| A run's steps | `GET /pipelines/${pipeline_uuid}/steps` (REST) |
| A step's log | `GET /pipelines/${pipeline_uuid}/steps/${step_uuid}/log` (REST); `twg bitbucket pipeline tail` |
| Artifacts | none in the public API |

### Write to Bitbucket Pipelines

| Capability | Command |
| --- | --- |
| Rerun | none in the public API; `twg bitbucket pipeline rerun-failed` |
| Cancel | `POST /pipelines/${pipeline_uuid}/stopPipeline` (REST) |
| Start a run | `POST /pipelines` with a `target` of a branch, a tag, or a commit, and a `selector` for a custom pipeline (REST); `twg bitbucket pipeline run` |
| Report a status | `POST /commit/${sha}/statuses/build` with `key`, `state`, and `url` (REST), for another CI system on Bitbucket |

### Bitbucket Pipelines schemas

- [Bitbucket Pipelines configuration JSON Schema](https://api.bitbucket.org/schemas/pipelines-configuration),
  SchemaStore's "bitbucket-pipelines" entry
- [Bitbucket REST OpenAPI 3](https://dac-static.atlassian.com/cloud/bitbucket/swagger.v3.json), with the pipelines and
  commit statuses routes, and the [Swagger 2 form](https://api.bitbucket.org/swagger.json)
