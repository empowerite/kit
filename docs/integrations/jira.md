# jira

This page says what `jira` reads and writes where it meets kit's scope, for each service it reaches, and the
machine-readable descriptions each publishes. The index of these pages is [1st.md](1st.md).

| Service | Category | kit status |
| --- | --- | --- |
| [Jira](#jira) | [tracker](trackers.md) | none |

## Jira

Jira is Atlassian's tracker, as Jira Cloud and as Jira Data Center, which Atlassian supports until March 28, 2029
([Data Center end of life](https://www.atlassian.com/licensing/data-center-end-of-life)). An issue's key, such as
`ENG-123`, is a project key and a number, so the record's `prefix` is `ENG-`. A Jira issue has one assignee, and
closing it is a workflow transition, with a resolution where the transition's screen asks for one.

Two CLIs reach it:

- `acli` is Atlassian's own CLI ([acli reference](https://developer.atlassian.com/cloud/acli/reference/commands/)). It
  signs in to a Cloud site with `acli jira auth login`; its reference names no Data Center support. `--json` prints
  JSON; `search --paginate` reads every page. It has no raw API command. Appfire sells an unrelated product also named
  ACLI.
- `jira` is a community CLI ([ankitpokhrel/jira-cli](https://github.com/ankitpokhrel/jira-cli)) for Cloud and Data
  Center, with an API token, a personal access token (`JIRA_AUTH_TYPE=bearer`), or client certificates. `--raw`
  prints the API's JSON, and `--plain` prints tab-separated text. `--paginate "${from}:${limit}"` reads at most
  100 items a call.

A call neither CLI makes goes to the REST API, `/rest/api/3` on Cloud and `/rest/api/2` on Data Center, with `curl`
and a token.

### Read from Jira

| Capability | `acli` | `jira` |
| --- | --- | --- |
| Title, state, and assignee | `acli jira workitem view "${key}" --fields summary,status,assignee --json` | `jira issue view "${key}" --raw` |
| Caller | `acli jira auth status` (text); the account ID from `GET /rest/api/3/myself` | `jira me` |
| Issue page | none; `${site}/browse/${key}`, which `acli jira workitem view "${key}" --web` opens | `jira open "${key}" --no-browser` |
| New-issue page | none; `${site}/secure/CreateIssueDetails!init.jspa?pid=${project_id}&issuetype=${type_id}&summary=${title}` | the same |
| Open issues | `acli jira workitem search --jql "project = ${project} AND statusCategory != Done ORDER BY created ASC" --fields key,summary,assignee,labels,created --json --paginate` | `jira issue list -q "statusCategory != Done" --order-by created --reverse --raw --paginate "${from}:100"` |
| Parent and subtasks | `acli jira workitem view "${key}" --fields parent,subtasks --json` | `jira issue list --parent "${key}" --raw` |
| Sprints | `acli jira board list-sprints --id "${board}" --state active,future --json` | `jira sprint list --table --plain` |
| A sprint's issues | `acli jira sprint list-workitems --board "${board}" --sprint "${sprint}" --json --paginate` | `jira sprint list "${sprint}" --plain` |
| Versions | `GET /rest/api/3/project/${project}/versions` (raw REST) | `jira release list --project "${project}"` |
| Status | the `status` field of `view` | the same |

### Write to Jira

| Capability | `acli` | `jira` |
| --- | --- | --- |
| Assign | `acli jira workitem assign --key "${key}" --assignee "${email}" --yes` | `jira issue assign "${key}" "${user}"` |
| Unassign | `acli jira workitem assign --key "${key}" --remove-assignee --yes` | `jira issue assign "${key}" x` |
| Close as completed | `acli jira workitem transition --key "${key}" --status Done --yes` | `jira issue move "${key}" Done -R Done` |
| Close as not planned | the same transition; the resolution through `POST /rest/api/3/issue/${key}/transitions` (raw REST) | `jira issue move "${key}" Done -R "Won't Do"` |
| Close as a duplicate | `acli jira workitem link create --out "${key}" --in "${other}" --type duplicates --yes`, then the transition | `jira issue link "${key}" "${other}" Duplicate`, then `jira issue move "${key}" Done -R Duplicate` |
| Comment | `acli jira workitem comment create --key "${key}" --body "${text}"` | `jira issue comment add "${key}" "${text}"` |
| Create | `acli jira workitem create --project "${project}" --type Task --summary "${title}" --description "${body}" --json` | `jira issue create -p "${project}" -t Task -s "${title}" -b "${body}" --no-input --raw` |
| Put under a parent | `--parent "${parent}"` on `create` only; an existing issue through `PUT /rest/api/3/issue/${key}` with `fields.parent` (raw REST) | `jira issue edit "${key}" --parent "${parent}" --no-input`; `jira epic add "${epic}" "${key}"` |
| Add to a sprint | `POST /rest/agile/1.0/sprint/${sprint}/issue` (raw REST) | `jira sprint add "${sprint}" "${key}"` |
| Link a change | the key in the branch name, a commit message, or the pull request's title | the same |

Gaps: Jira has no closing keyword.
[GitHub for Jira](https://support.atlassian.com/jira-cloud-administration/docs/integrate-with-github/) and the GitLab
integration show the branches, commits, and pull requests that name the key in the issue's Development panel, and
[smart commits](https://support.atlassian.com/jira-software-cloud/docs/process-issues-with-smart-commits/) such as
`ENG-123 #done` transition it; kit would close it itself, as `kit settle` does for Azure Boards. The Development
panel's data has no documented read route. Resolution names (`Done`, `Won't Do`, `Duplicate`) are a site's own.
`acli` names a user by email or account ID, and `jira` by display name or email.

### Jira schemas

- [Jira Cloud platform OpenAPI 3](https://developer.atlassian.com/cloud/jira/platform/swagger-v3.v3.json): issues,
  transitions, comments, links, versions, `myself`, and `search/jql`; `search` is deprecated
- [Jira Software Cloud OpenAPI 3](https://developer.atlassian.com/cloud/jira/software/swagger.v3.json): boards,
  sprints, and epics under `/rest/agile/1.0`
- [Jira Data Center 11.3 OpenAPI 3](https://dac-static.atlassian.com/server/jira/platform/jira_software_dc_11003_swagger.v3.json):
  the platform's `/api/2` and Jira Software's `/agile/1.0`; each release has its own file, linked from
  [the Data Center REST reference](https://developer.atlassian.com/server/jira/platform/rest/v11003/)
- No JSON Schema for webhook payloads or for either CLI's output. SchemaStore has no Jira entry.
