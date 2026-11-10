# Trackers

A tracker is the system that holds a repository's issues: the code forge's own issues, or a separate system such as
Azure Boards, Jira, or Linear. This page catalogues the trackers kit reaches or could reach: what each one's command
line and API answer for kit's calls, and the machine-readable descriptions each one publishes. The seam, its
references, and its three providers are described in [the tracker seam](../trackers.md); every category of integration
is listed in [the integrations index](1st.md).

kit reaches a tracker through `kit/modules/bash/tracker.sh`. `kit work`, `kit assign`, `kit unassign`, `kit close`,
`kit gather`, `kit settle`, `kit new issue`, `kit linked`, and `kit tidy` call it. The backlog and underway stages of
`kit flow` draw the code forge's open issues, through `open-issues` in `kit/modules/bash/forge.sh`. kittui has
iterations and milestones tabs with no data source yet, and this page records where each tracker keeps both.

see: [the code forges](forges.md), whose APIs GitHub Issues, GitLab, Azure Boards, Gitea, and Forgejo share

## Scope

| Capability | Kind | Seam call or kit verb |
| --- | --- | --- |
| An issue's title | read | `title`: `kit work` names the branch from it, `kit gather` |
| An issue's state, open or closed | read | `state`: `kit work`, `kit linked`, `kit tidy`, `kit close` |
| An issue's assignees | read | `assignees`: `kit work`, `kit assign`, `kit unassign` |
| The caller's login in the tracker | read | `me`: `kit work` assigns the caller, `kit assign` |
| An issue's page | read | `url` |
| The page a new issue is filed at | read | `new-url`: `kit new issue` |
| The open issues, each with assignees, age, and labels | read | `open-issues` (forge seam): the backlog and underway stages |
| Adds or removes an assignee | write | `assign`, `unassign`: `kit work`, `kit assign`, `kit unassign`, `kit settle` |
| Closes an issue as completed, not planned, or a duplicate | write | `close`: `kit close`, `kit close --duplicate-of`, `kit settle` |
| Adds a comment | write | `comment`: `kit settle`, `kit close --duplicate-of` across trackers |
| Creates an issue | write | `create`: `kit gather` and `kit work --title` create the parent |
| Puts an issue under a parent | write | `issue-add-sub` (forge seam): `kit gather`, `kit work --title` |
| Links a change to an issue | write | `kit work`: a linked branch, and a `Closes` line in the change's body; `kit linked` reads it |
| Iterations: sprints or cycles, with dates and their issues | read | planned: kittui's iterations tab |
| Milestones or versions, with due dates and their issues | read | planned: kittui's milestones tab |
| A project board's status field | read | planned: a column beside an issue in the backlog |

The calls are the seam's, and a provider answers each one. kit's providers call a tracker's command line, which holds
the sign-in; a call that goes only through a tracker's raw API needs a token of its own.

## Providers

| Provider | Hosting | CLI | kit status | JSON from the CLI | Machine-readable descriptions |
| --- | --- | --- | --- | --- | --- |
| GitHub Issues | SaaS; GitHub Enterprise Server self-hosted; proprietary | `gh`, official ([cli/cli](https://github.com/cli/cli), MIT) | `provider` (`forge`) | `--json "${fields}"`, `--jq`; `gh api` | REST OpenAPI, GraphQL schema, webhook payloads in OpenAPI 3.1, issue form JSON Schema |
| GitLab Issues and work items | SaaS; self-managed; open core (MIT core) | `glab`, official ([gitlab-org/cli](https://gitlab.com/gitlab-org/cli), MIT) | `provider` (`forge`) | `--output json`, `--jq`; `glab api` | REST OpenAPI 3.0, GraphQL introspection result |
| Azure Boards | SaaS (Azure DevOps Services); Azure DevOps Server self-hosted; proprietary | `az` with the `azure-devops` extension, official ([Azure/azure-devops-cli-extension](https://github.com/Azure/azure-devops-cli-extension), MIT) | `provider` (`ado`) | `--output json`, `--query` (JMESPath); `az devops invoke`, `az rest` | none published; a REST reference in prose |
| GitHub Projects | SaaS; GitHub Enterprise Server; proprietary | `gh project`, official | none | `--format json`, `--jq`; `gh api graphql` | GraphQL schema (`ProjectV2`), `projects_v2_item` webhooks |
| Jira Cloud and Data Center | SaaS; Data Center self-hosted until March 28, 2029; proprietary | `acli`, official, Cloud ([acli reference](https://developer.atlassian.com/cloud/acli/reference/commands/), proprietary); `jira`, community ([ankitpokhrel/jira-cli](https://github.com/ankitpokhrel/jira-cli), MIT) | none | `acli`: `--json`; `jira`: `--raw`, `--plain` | Cloud platform and Jira Software OpenAPI 3; Data Center OpenAPI 3 per release |
| Linear | SaaS; proprietary | `linear`, community ([schpet/linear-cli](https://github.com/schpet/linear-cli), ISC) | none | `--json`; `linear api` | GraphQL schema |
| YouTrack | SaaS; YouTrack Server self-hosted; proprietary | `yt`, community ([ryancheley/yt-cli](https://github.com/ryancheley/yt-cli), MIT) | none | `--format json` | OpenAPI 3 per instance |
| Shortcut | SaaS; proprietary | `short`, community ([shortcut-cli/shortcut-cli](https://github.com/shortcut-cli/shortcut-cli), MIT) | none | none; `short api` | REST Swagger 2.0 |
| Plane | SaaS; self-hosted; open source (AGPL-3.0) | none | none | — | OpenAPI 3 per self-hosted instance, off by default |
| Redmine | self-hosted; open source (GPL-2.0) | none | none | — | none official; an unofficial OpenAPI 3 |
| Gitea and Forgejo | self-hosted; Gitea Cloud, Codeberg; open source (Gitea MIT, Forgejo GPL-3.0) | `tea`, official for Gitea ([gitea/tea](https://gitea.com/gitea/tea), MIT); `fj`, community ([forgejo-contrib/forgejo-cli](https://codeberg.org/forgejo-contrib/forgejo-cli), Apache-2.0 or MIT) | none | `tea`: `--output json`, `tea api`; `fj`: none | REST Swagger 2.0 per instance, issue form JSON Schema |
| OpenProject | self-hosted; SaaS; open source (GPL-3.0) | `op`, from OpenProject's own organization ([opf/openproject-cli](https://github.com/opf/openproject-cli), GPL-3.0) | none | none found | OpenAPI 3 |
| Bugzilla | self-hosted; open source (MPL-2.0) | `bugzilla`, community ([python-bugzilla/python-bugzilla](https://github.com/python-bugzilla/python-bugzilla), GPL-2.0) | none | `--json` | none |
| Taiga | SaaS; self-hosted; open source (MPL-2.0) | none | none | — | none |

The order is roughly by use among development teams, kit's own providers first. Any tracker here can be a `web`
tracker today: kit opens its new-issue page and an issue's page, and refuses every other call. Asana and ClickUp are
left out: development teams use them less as the tracker a branch and a change name, and kit has no reference syntax
for either.

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
| Milestones | `gh api --paginate 'repos/{owner}/{repo}/milestones?state=all'` (raw REST) |
| Iterations | none in Issues; see [GitHub Projects](#github-projects) |

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

## GitLab

`glab` is GitLab's own CLI. `glab auth login` stores a token per host; `GITLAB_TOKEN` overrides it. `--output json`
(`-O` on `issue list`, `-F` elsewhere) prints JSON and `--jq` filters it. `glab api` reaches the REST API, with `:id`
and `:fullpath` standing for the current project and `--paginate` reading every page, and `glab api graphql` reaches
the GraphQL API; `glab api` has no `--jq`, so its answer goes to `jq`. `glab issue list` reads one page per call
(`--page`, `--per-page`), and kit's `forge/gitlab.sh` reads issues through `glab api` instead. Epics, iterations, and
work item statuses need GitLab Premium or Ultimate.

### Read from GitLab

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
| Milestones | `glab milestone list --output json` |
| Iterations | `glab iteration list --output json` |
| Work item status | `glab api graphql -f query="${query}"`, the `status` widget of `workItem` (raw GraphQL) |

### Write to GitLab

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

### GitLab schemas

- [GitLab REST API OpenAPI 3.0](https://gitlab.com/gitlab-org/gitlab/-/raw/master/doc/api/openapi/openapi_v3.yaml),
  generated from the API's own definitions; the OpenAPI 2.0 file beside it says it is deprecated
- [GitLab GraphQL introspection result](https://gitlab.com/-/graphql/introspection_result.json) and
  [the same without deprecated fields](https://gitlab.com/-/graphql/introspection_result_no_deprecated.json), both
  served by every instance at the same path
- No JSON Schema for issue templates (`.gitlab/issue_templates/` holds Markdown) or for webhook payloads.

## Azure Boards

`az` is Microsoft's own CLI; its `azure-devops` extension adds `az boards`, `az repos`, and `az pipelines`. `az login`
signs in with Microsoft Entra ID; `az devops login` or `AZURE_DEVOPS_EXT_PAT` takes a personal access token instead.
`--output json` prints JSON and `--query` filters it with JMESPath. `az devops invoke --area --resource
--route-parameters` reaches any REST route by its area, and `az rest` reaches any URL. `az boards query` runs a flat
Work Item Query Language (WIQL) query and answers every item it matches. kit's `tracker/ado.sh` uses these commands.

### Read from Azure Boards

| Capability | Command |
| --- | --- |
| Title | `az boards work-item show --id "${n}" --org "${org}" --output json`, `.fields["System.Title"]` |
| State | the same, `.fields["System.State"]`; kit reads Closed, Done, and Removed as closed |
| Assignees | the same, `.fields["System.AssignedTo"].uniqueName` |
| Caller | `az account show --query user.name --output tsv` |
| Issue page | none; kit builds `${org}/${project}/_workitems/edit/${n}` |
| New-issue page | none; kit builds `${org}/${project}/_workitems/create/${type}?[System.Title]=${title}` |
| Open issues | `az boards query --org "${org}" --project "${project}" --wiql "${wiql}"`, selecting `System.Title`, `System.AssignedTo`, `System.Tags`, and `System.CreatedDate` where `System.State` is not closed |
| Parent, children, and linked pull requests | `az boards work-item show --id "${n}" --expand relations --org "${org}"`, or `az boards work-item relation show --id "${n}"` |
| Iterations | `az boards iteration team list --team "${team}" --project "${project}" --org "${org}" --output json` |
| An iteration's work items | `az boards iteration team list-work-items --id "${iteration_id}" --team "${team}" --project "${project}"` |
| Board status | `az boards work-item show --id "${n}"`, `.fields["System.BoardColumn"]` |

### Write to Azure Boards

| Capability | Command |
| --- | --- |
| Assign | `az boards work-item update --id "${n}" --assigned-to "${login}" --org "${org}"` |
| Unassign | `az boards work-item update --id "${n}" --assigned-to "" --org "${org}"` |
| Close as completed | `az boards work-item update --id "${n}" --state Closed --org "${org}"` (`closed_state`) |
| Close as not planned | `az boards work-item update --id "${n}" --state Removed --org "${org}"` (`removed_state`); `--reason` sets the process's reason |
| Close as a duplicate | the comment `Duplicate of AB#${m}`, then the not-planned close; `az boards work-item relation add --relation-type "${type}"` can add a duplicate link, its name from `relation list-type` |
| Comment | `az boards work-item update --id "${n}" --discussion "${text}" --org "${org}"` |
| Create | `az boards work-item create --title "${title}" --type "${type}" --description "${body}" --project "${project}" --org "${org}"` |
| Put under a parent | `az boards work-item relation add --id "${m}" --relation-type parent --target-id "${n}" --org "${org}"`; kit writes a task line in the parent's body instead |
| Link a change | `AB#${n}` in a GitHub pull request's title, description, or commits; `az repos pr work-item add --id "${pr}" --work-items "${n}"` for an Azure Repos pull request |
| Set the iteration | `az boards work-item update --id "${n}" --iteration "${iteration_path}"` |

Gaps: a work item has one assignee, and no milestone object; an iteration path or a tag stands in for one. A process
template defines the states, so the closed states are settings (`closed_state`, `removed_state`). The
[Azure Boards app for GitHub](https://learn.microsoft.com/en-us/azure/devops/boards/github/link-to-from-github) links
`AB#` references, and `Fixes AB#${n}` moves the work item to its done state when the pull request merges.

### Azure Boards schemas

- Microsoft publishes no OpenAPI document for the Azure DevOps REST API. The Swagger documents per area were in
  MicrosoftDocs/vsts-rest-api-specs, which answers 404 on 2026-11-10, and no other copy was found.
  [The work item tracking REST reference](https://learn.microsoft.com/en-us/rest/api/azure/devops/wit/) and the
  `work` area's, for iterations and boards, describe the routes in prose.
- No JSON Schema for work item types or service hook payloads. The SchemaStore entries for Azure DevOps cover
  `azure-pipelines.yml` and extension manifests; see [continuous integration](ci.md).

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
| Iterations, with dates | `gh api graphql -f query="${query}" -F project="${project_id}"` (raw GraphQL; below) |

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

## Linear

Linear is a SaaS tracker reached through one GraphQL API at `https://api.linear.app/graphql`. An issue's identifier,
such as `ENG-123`, is a team key and a number, so the record's `prefix` is `ENG-`. An issue has one assignee, a
workflow state whose type is `triage`, `backlog`, `unstarted`, `started`, `completed`, or `canceled`, an optional
parent, a cycle, and a project milestone.

Linear publishes no CLI of its own; it publishes the GraphQL schema and a TypeScript SDK. `linear` is a community
CLI ([schpet/linear-cli](https://github.com/schpet/linear-cli)). `linear auth login` stores an API key per workspace,
and `LINEAR_API_KEY` overrides it. `--json` prints JSON, `--limit all` reads every page, and `linear api` sends a
GraphQL document.

### Read from Linear

| Capability | Command |
| --- | --- |
| Title, state, assignee, and parent | `linear issue view "${key}" --json` |
| Title alone | `linear issue title "${key}"` |
| Caller | `linear auth whoami`, or `linear api '{ viewer { id name email } }'` |
| Issue page | `linear issue url "${key}"` |
| New-issue page | none; `https://linear.app/${workspace}/team/${team}/new?title=${title}` |
| Open issues | `linear issue query --team "${team}" --state backlog --state unstarted --state started --json --limit all` |
| Cycles | `linear cycle list --team "${team}" --json` |
| A cycle's issues | `linear cycle view "${cycle}" --team "${team}" --json` |
| Project milestones | `linear milestone list --project "${project}" --json` |
| A milestone's issues | `linear milestone view "${milestone}" --json` |

### Write to Linear

| Capability | Command |
| --- | --- |
| Assign | `linear issue update "${key}" --assignee "${user}"` |
| Unassign | `linear issue update "${key}" --unassign` |
| Close as completed | `linear issue update "${key}" --state completed` |
| Close as not planned | `linear issue update "${key}" --state canceled` |
| Close as a duplicate | `linear issue relation add "${key}" duplicate "${other}"`, then `linear issue update "${key}" --state canceled` |
| Comment | `linear issue comment add "${key}" --body "${text}"` |
| Create | `linear issue create --team "${team}" --title "${title}" --description "${body}" --no-input` |
| Put under a parent | `linear issue update "${key}" --parent "${parent}"` |
| Set the cycle or milestone | `linear issue update "${key}" --cycle "${cycle}"`, `--milestone "${milestone}"` |
| Link a change | the identifier in the branch name, or `Fixes ENG-123` in the pull request's description |

Gaps: `--state` takes a state's name or its type, so the types `completed` and `canceled` map closed and not planned
without a setting. Linear's
[GitHub integration](https://linear.app/docs/github) links a pull request whose branch, title, or description names
the identifier, and closes the issue on merge when a closing word precedes it.

### Linear schemas

- [Linear GraphQL schema](https://raw.githubusercontent.com/linear/linear/master/packages/sdk/src/schema.graphql),
  the SDK's copy in [the linear/linear repository](https://github.com/linear/linear); it also types the webhook
  payloads (`*WebhookPayload`) and the `IssueRelationType` enum with `duplicate`
- No OpenAPI, no JSON Schema for webhooks apart from those GraphQL types, and no SchemaStore entry.

## YouTrack

YouTrack is JetBrains' tracker, as YouTrack Cloud and as self-hosted YouTrack Server. An issue's ID, such as
`ENG-123`, is a project's short name and a number. State, Assignee, and Fix versions are custom fields a project
configures; a project's agile boards hold its sprints.

JetBrains publishes no CLI. `yt` is a community CLI ([ryancheley/yt-cli](https://github.com/ryancheley/yt-cli)) that
signs in with `yt auth login` and a permanent token; `--format json` prints JSON on its list and search commands, and
`--all` reads every page. A call it does not make goes to the REST API under `${instance}/api`, with
`Authorization: Bearer ${token}` and a `fields` parameter naming what to return.

### Read from YouTrack

| Capability | Command |
| --- | --- |
| Title, state, and assignee | `GET ${instance}/api/issues/${key}?fields=summary,resolved,customFields(name,value(login,name))` (raw REST) |
| Caller | `GET ${instance}/api/users/me?fields=login` (raw REST) |
| Issue page | none; `${instance}/issue/${key}` |
| New-issue page | none; `${instance}/newIssue?project=${project}&summary=${title}` |
| Open issues | `yt issues search "project: ${project} #Unresolved sort by: created asc" --format json --all` |
| Links, among them parent and subtasks | `yt issues links list "${key}"` |
| Sprints | `GET ${instance}/api/agiles/${board}/sprints?fields=name,start,finish,issues(idReadable)` (raw REST) |

### Write to YouTrack

| Capability | Command |
| --- | --- |
| Assign | `yt issues assign "${key}" "${login}"` |
| Unassign | `POST ${instance}/api/commands` with the query `Assignee Unassigned` (raw REST) |
| Close as completed | `yt issues move "${key}" --state Fixed` |
| Close as not planned | `yt issues move "${key}" --state "Won't fix"` |
| Close as a duplicate | `yt issues links create "${key}" "${other}" duplicates`, then `yt issues move "${key}" --state Duplicate` |
| Comment | `yt issues comments add "${key}" "${text}"` |
| Create | `yt issues create "${project}" "${title}" -d "${body}"` |
| Put under a parent | `yt issues links create "${key}" "${parent}" "subtask of"` |
| Link a change | the issue ID in a commit message or pull request, through a VCS integration the project configures |

Gaps: the state names come from the project's State field (Fixed, Won't fix, and Duplicate are the defaults), so a
provider needs them as settings. YouTrack has no milestone object; a Fix versions field stands in for one. The
commands API (`/api/commands`) applies the same commands as YouTrack's command window.

### YouTrack schemas

- [YouTrack OpenAPI 3 from JetBrains' instance](https://youtrack.jetbrains.com/api/openapi.json); every instance
  serves its own at `${instance}/api/openapi.json`, which matches its version
- [The YouTrack REST API documentation](https://www.jetbrains.com/help/youtrack/devportal/youtrack-rest-api.html)
- The SchemaStore entry `YouTrack App` describes app manifests, not the tracker's data.

## Shortcut

Shortcut is a SaaS tracker whose issues are stories. A story has owners, a workflow state of type `unstarted`,
`started`, or `done`, an iteration, an epic, and an optional parent story; objectives group epics. A story's
reference in a branch is `sc-123`.

Shortcut publishes no CLI of its own. `short` is a community CLI
([shortcut-cli/shortcut-cli](https://github.com/shortcut-cli/shortcut-cli)) that reads `SHORTCUT_API_TOKEN` or the
token `short install` stores. It prints text, shaped by `--format` templates; `short api` sends any REST request and
prints the JSON answer.

### Read from Shortcut

| Capability | Command |
| --- | --- |
| Title, state, and owners | `short story "${id}"`; `short api "/stories/${id}"` for JSON |
| Caller | `short api /member` |
| Issue page | `short api "/stories/${id}"`, its `app_url` |
| New-issue page | none (not verified) |
| Open issues | `short api /search/stories -f query='!is:done' -f page_size=25`, following `next` |
| Sub-tasks | `short story sub-tasks "${id}"` |
| Iterations | `short iterations`; `short api /iterations` for JSON |
| Objectives | `short objectives` |

### Write to Shortcut

| Capability | Command |
| --- | --- |
| Assign or unassign | `short story "${id}" --owners "${owners}"`, which sets the whole list |
| Close as completed | `short story "${id}" --state Done` |
| Close as not planned | none; `short story "${id}" --archived` archives it |
| Close as a duplicate | `short story relation add "${id}" "${other}" --type duplicates`, then archive it |
| Comment | `short story "${id}" --comment "${text}"` |
| Create | `short create --title "${title}" --description "${body}" --project "${project}" --idonly` |
| Put under a parent | `short api "/stories/${id}" -X PUT -f parent_story_id="${parent}"` (raw REST) |
| Set the iteration | `short story "${id}" --iteration "${iteration}"` |
| Link a change | `sc-${id}` in the branch name, or `[sc-${id}]` in the pull request's title or description |

Gaps: no JSON outside `short api`, and no state for not planned: a team archives the story or keeps a done state for
it. The state names are a workspace's own.

### Shortcut schemas

- [Shortcut REST API v3 Swagger 2.0](https://developer.shortcut.com/api/rest/v3/shortcut.swagger.json), described in
  [the Shortcut REST API reference](https://developer.shortcut.com/api/rest/v3)
- No JSON Schema for webhook payloads and no SchemaStore entry.

## Plane

Plane is an open-source tracker (AGPL-3.0), self-hosted or on Plane's SaaS. Its issues are work items; a work item's
reference is a project identifier and a number, such as `ENG-123`. A work item has assignees, a state in a state
group (`backlog`, `unstarted`, `started`, `completed`, `cancelled`), a parent, cycles (sprints), and modules.

Plane has no CLI. Its REST API answers under `/api/v1/workspaces/${workspace}/` with an `X-API-Key` header, and pages
by cursor: `per_page` at most 100, then `cursor` from `next_cursor`. The routes below follow
[the Plane API reference](https://developers.plane.so/api-reference/introduction) and were not run.

### Read from Plane

| Capability | Command |
| --- | --- |
| A work item by its reference | `GET /api/v1/workspaces/${workspace}/work-items/${identifier}-${sequence}/` (raw REST) |
| Caller | `GET /api/v1/users/me/` (raw REST) |
| Open issues | `GET /api/v1/workspaces/${workspace}/projects/${project_id}/work-items/?per_page=100` (raw REST), filtered by state group |
| Cycles | `GET /api/v1/workspaces/${workspace}/projects/${project_id}/cycles/` (raw REST) |
| Modules | `GET /api/v1/workspaces/${workspace}/projects/${project_id}/modules/` (raw REST) |

### Write to Plane

| Capability | Command |
| --- | --- |
| Assign, unassign, close, or put under a parent | `PATCH /api/v1/workspaces/${workspace}/projects/${project_id}/work-items/${id}/` with `assignees`, `state`, or `parent` (raw REST) |
| Comment | `POST /api/v1/workspaces/${workspace}/projects/${project_id}/work-items/${id}/comments/` (raw REST) |
| Create | `POST /api/v1/workspaces/${workspace}/projects/${project_id}/work-items/` (raw REST) |
| Link a change | the work item's reference in a pull request, through Plane's GitHub or GitLab integration |

Gaps: no CLI, so every call is a raw request with a stored key. A close sets a state, which is a UUID a provider looks
up by its group. Whether Plane has a milestone object beside modules is not settled here.

### Plane schemas

- An OpenAPI 3 document at `${instance}/api/schema/?format=openapi-json` on a self-hosted instance that sets
  `ENABLE_DRF_SPECTACULAR=1`; it is off by default, and Plane's SaaS answered 404 (not verified). See
  [Plane's OpenAPI specification page](https://developers.plane.so/dev-tools/openapi-specification).

## Redmine

Redmine is an open-source tracker (GPL-2.0) that a team hosts. An issue has one assignee, a status that is closed or
not by its definition, a parent issue, relations (among them `duplicates`), and a target version.

Redmine has no CLI of its own, and no maintained community one stood out. Its REST API answers JSON at `.json`
routes, with an `X-Redmine-API-Key` header, and pages with `offset` and `limit` (at most 100).

### Read from Redmine

| Capability | Command |
| --- | --- |
| Title, status, and assignee | `GET ${redmine}/issues/${n}.json` (raw REST) |
| Caller | `GET ${redmine}/users/current.json` (raw REST) |
| Issue page | none; `${redmine}/issues/${n}` |
| New-issue page | none; `${redmine}/projects/${project}/issues/new?issue[subject]=${title}` |
| Open issues | `GET ${redmine}/issues.json?project_id=${project}&status_id=open&sort=created_on&limit=100&offset=${offset}` (raw REST) |
| Children and relations | `GET ${redmine}/issues/${n}.json?include=children,relations` (raw REST) |
| Versions | `GET ${redmine}/projects/${project}/versions.json` (raw REST) |

### Write to Redmine

| Capability | Command |
| --- | --- |
| Assign, unassign, close, or comment | `PUT ${redmine}/issues/${n}.json` with `assigned_to_id`, `status_id`, or `notes` (raw REST) |
| Close as a duplicate | `POST ${redmine}/issues/${n}/relations.json` with `relation_type: duplicates`, then a closed status (raw REST) |
| Create | `POST ${redmine}/issues.json` (raw REST) |
| Put under a parent | `PUT ${redmine}/issues/${n}.json` with `parent_issue_id` (raw REST) |
| Link a change | `refs #${n}` or `fixes #${n}` in a commit of a repository Redmine reads; the keywords are settings |

Gaps: the API must be enabled by an administrator. Redmine writes `#123` for its own issues, which kit reads as the
code forge's, so its record needs another prefix. Statuses are IDs a provider looks up.

### Redmine schemas

- No official description; see [the Redmine REST API](https://www.redmine.org/projects/redmine/wiki/Rest_api).
- [An unofficial Redmine OpenAPI 3](https://raw.githubusercontent.com/d-yoshi/redmine-openapi/main/openapi.yaml),
  tested by its author against Redmine 7.0.

## Gitea and Forgejo

Gitea and Forgejo are self-hosted code forges with issues; Forgejo is a fork of Gitea and keeps its API. Gitea is
MIT and Forgejo GPL-3.0; Gitea Cloud hosts Gitea and Codeberg hosts Forgejo. An issue has assignees, labels, a
milestone, and dependencies; it has no parent and no close reason.

`tea` is Gitea's own CLI. `tea login add` stores a token per instance. `--output json` prints JSON from lists, and
`tea api` reaches any REST route, with `{owner}` and `{repo}` standing for the current repository. `--page` and
`--limit` page a list. It talks to Forgejo too, which keeps Gitea's routes, though a Forgejo-only route may differ.
`fj` is a community CLI for Forgejo ([forgejo-contrib/forgejo-cli](https://codeberg.org/forgejo-contrib/forgejo-cli))
that prints text only.

### Read from Gitea and Forgejo

| Capability | Command |
| --- | --- |
| Title, state, and assignees | `tea api "/repos/{owner}/{repo}/issues/${n}"` |
| Caller | `tea whoami` |
| Issue page | `tea api "/repos/{owner}/{repo}/issues/${n}"`, its `html_url` |
| New-issue page | none; `${repo_url}/issues/new?title=${title}` |
| Open issues | `tea issues list --state open --fields index,title,assignees,labels,created,url --output json --limit 50 --page "${page}"` |
| Dependencies | `tea api "/repos/{owner}/{repo}/issues/${n}/dependencies"` |
| Milestones | `tea milestones list --output json` |

### Write to Gitea and Forgejo

| Capability | Command |
| --- | --- |
| Assign | `tea issues edit "${n}" --add-assignees "${login}"` |
| Unassign | `tea issues edit "${n}" --remove-assignees "${login}"` |
| Close | `tea issues close "${n}"`; no reason is kept |
| Comment | `tea comments add "${n}" "${text}"` |
| Create | `tea issues create --title "${title}" --description "${body}"` |
| Put under a parent | none; `tea api -X POST "/repos/{owner}/{repo}/issues/${n}/blocks"` makes a dependency instead |
| Link a change | `Closes #${n}` in the pull request's description |
| Set the milestone | `tea issues edit "${n}" --milestone "${milestone}"` |

Gaps: no sub-issues, no duplicate state, and no iterations.

see: [the code forges](forges.md#gitea) for `tea` and `fj` against pull requests

### Gitea and Forgejo schemas

- [Gitea's Swagger 2.0 from gitea.com](https://gitea.com/swagger.v1.json) and
  [Forgejo's Swagger 2.0 from Codeberg](https://codeberg.org/swagger.v1.json); every instance serves its own at
  `/swagger.v1.json`
- [Gitea Issue Template forms](https://www.schemastore.org/gitea-issue-forms.json) and
  [Gitea Issue Template configuration](https://www.schemastore.org/gitea-issue-config.json), SchemaStore entries
  for `.gitea/ISSUE_TEMPLATE/` and `.forgejo/ISSUE_TEMPLATE/`

## OpenProject

OpenProject is an open-source project management system (GPL-3.0), self-hosted or on OpenProject's SaaS. Its issues
are work packages, each with one assignee, a status, a parent, relations (among them `duplicates`), and a version.
Its API v3 answers HAL+JSON under `/api/v3` with an API key.

`op` is a CLI from OpenProject's own GitHub organization
([opf/openproject-cli](https://github.com/opf/openproject-cli)). `op login` stores a token per profile. Its README
shows text output only.

### Read from OpenProject

| Capability | Command |
| --- | --- |
| Title, status, and assignee | `op work-package inspect "${n}"`; `GET ${instance}/api/v3/work_packages/${n}` for JSON |
| Caller | `op whoami` |
| Open issues | `GET ${instance}/api/v3/projects/${project}/work_packages` with a status filter (raw REST) |
| Relations | `GET ${instance}/api/v3/work_packages/${n}/relations` (raw REST) |
| Versions | `GET ${instance}/api/v3/versions` (raw REST) |

### Write to OpenProject

| Capability | Command |
| --- | --- |
| Assign | `op work-package update "${n}" --assignee "${user}"` |
| Unassign, close, or put under a parent | `PATCH ${instance}/api/v3/work_packages/${n}` with `assignee`, `status`, or `parent` links and the `lockVersion` (raw REST) |
| Comment | `POST ${instance}/api/v3/work_packages/${n}/activities` (raw REST) |
| Close as a duplicate | `POST ${instance}/api/v3/work_packages/${n}/relations` with type `duplicates` (raw REST) |
| Create | `op work-package create --project "${project}" "${title}"` |
| Link a change | `OP#${n}` in the pull request's description, through OpenProject's GitHub or GitLab integration |

Gaps: every update carries the work package's `lockVersion`, so a write reads first. Statuses and custom actions are
an instance's own.

### OpenProject schemas

- [OpenProject API v3 OpenAPI 3 from community.openproject.org](https://community.openproject.org/api/v3/spec.json);
  every instance serves its own at `/api/v3/spec.json`
- [OpenProject API v3 OpenAPI source](https://raw.githubusercontent.com/opf/openproject/dev/docs/api/apiv3/openapi-spec.yml),
  the YAML in the repository

## Bugzilla

Bugzilla is an open-source bug tracker (MPL-2.0) that a project hosts. A bug has one assignee, a status and a
resolution (`FIXED`, `WONTFIX`, `DUPLICATE`, and others), dependencies, and a target milestone.

Bugzilla has no CLI of its own. `bugzilla` is a community CLI from the python-bugzilla project
([python-bugzilla/python-bugzilla](https://github.com/python-bugzilla/python-bugzilla)). `bugzilla login` stores an
API key, and `--json` prints JSON from `query`. The REST API answers under `/rest`.

### Read from Bugzilla

| Capability | Command |
| --- | --- |
| Title, status, and assignee | `bugzilla query --bug_id "${n}" --json` |
| Caller | `GET ${bugzilla}/rest/whoami` (raw REST) |
| Issue page | none; `${bugzilla}/show_bug.cgi?id=${n}` |
| New-issue page | none; `${bugzilla}/enter_bug.cgi?product=${product}&short_desc=${title}` |
| Open issues | `bugzilla query --product "${product}" --status NEW,ASSIGNED,REOPENED --json` |
| Milestones | `GET ${bugzilla}/rest/product?names=${product}` (raw REST), its `milestones` |

### Write to Bugzilla

| Capability | Command |
| --- | --- |
| Assign | `bugzilla modify "${n}" --assignee "${login}"` |
| Unassign | `bugzilla modify "${n}" --reset-assignee`, back to the component's default |
| Close as completed | `bugzilla modify "${n}" --close FIXED` |
| Close as not planned | `bugzilla modify "${n}" --close WONTFIX` |
| Close as a duplicate | `bugzilla modify "${n}" --dupeid "${m}"` |
| Comment | `bugzilla modify "${n}" --comment "${text}"` |
| Create | `bugzilla new --product "${product}" --component "${component}" --summary "${title}" --comment "${body}"` |
| Put under a parent | `bugzilla modify "${parent}" --dependson "${n}"`, a tracking bug that depends on it |
| Link a change | none |

Gaps: no parent, no iterations, and no link from a pull request; the open statuses are an installation's own.

### Bugzilla schemas

- None. [The Bugzilla REST API](https://bugzilla.readthedocs.io/en/latest/api/) is documented in prose.

## Taiga

Taiga is an open-source agile tracker (MPL-2.0), self-hosted or on Taiga's SaaS. It has user stories, tasks, and
issues; a sprint is a milestone in its API; a task belongs to a user story. A reference in a commit is `TG-123`.

Taiga has no CLI. Its REST API answers under `/api/v1` with a token from `POST /api/v1/auth`, and pages with
`page` and `x-pagination-*` headers.

### Read from Taiga

| Capability | Command |
| --- | --- |
| A user story by its reference | `GET ${taiga}/api/v1/userstories/by_ref?ref=${n}&project=${project_id}` (raw REST) |
| Caller | `GET ${taiga}/api/v1/users/me` (raw REST) |
| Open user stories | `GET ${taiga}/api/v1/userstories?project=${project_id}&status__is_closed=false` (raw REST) |
| Sprints | `GET ${taiga}/api/v1/milestones?project=${project_id}` (raw REST) |

### Write to Taiga

| Capability | Command |
| --- | --- |
| Assign, close, or comment | `PATCH ${taiga}/api/v1/userstories/${id}` with `assigned_to`, `status`, or `comment`, and its `version` (raw REST) |
| Create | `POST ${taiga}/api/v1/userstories` (raw REST) |
| Link a change | `TG-${n} #${status}` in a commit message, through Taiga's GitHub or GitLab webhook |

Gaps: no CLI, three kinds of item with separate routes, and every update carries the item's `version`.

### Taiga schemas

- None. [The Taiga REST API](https://docs.taiga.io/api.html) is documented in prose.

## Coverage

`cli` means the CLI makes the call, `api` that only the raw API does, through the CLI's escape hatch or `curl`, and
`—` that the tracker cannot. kit builds `new-url`, and `url` where no `cli` or `api` gives it, from a page pattern,
which needs neither. GitHub Projects answers the planning rows only; its items are GitHub issues.

| Capability | GitHub | Projects | GitLab | Azure Boards | Jira `acli` | Jira `jira` | Linear | YouTrack | Shortcut | Plane | Redmine | Gitea/Forgejo | OpenProject | Bugzilla | Taiga |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| `title` | cli | — | cli | cli | cli | cli | cli | api | cli | api | api | api | cli | cli | api |
| `state` | cli | — | cli | cli | cli | cli | cli | api | cli | api | api | api | cli | cli | api |
| `assignees` | cli | — | cli | cli | cli | cli | cli | api | cli | api | api | api | cli | cli | api |
| `assign` | cli | — | cli | cli | cli | cli | cli | cli | cli | api | api | cli | cli | cli | api |
| `unassign` | cli | — | cli | cli | cli | cli | cli | api | cli | api | api | cli | api | cli | api |
| `close` completed | cli | — | cli | cli | cli | cli | cli | cli | cli | api | api | cli | api | cli | api |
| `close` not planned | cli | — | — | cli | api | cli | cli | cli | — | api | api | — | api | cli | api |
| `close` duplicate | cli | — | cli | cli | cli | cli | cli | cli | cli | api | api | — | api | cli | — |
| `comment` | cli | — | cli | cli | cli | cli | cli | cli | cli | api | api | cli | api | cli | api |
| `me` | api | — | api | cli | api | cli | cli | api | api | api | api | cli | cli | api | api |
| `new-url` | — | — | — | — | — | — | — | — | — | — | — | — | — | — | — |
| `create` | cli | — | cli | cli | cli | cli | cli | cli | cli | api | api | cli | cli | cli | api |
| `url` | cli | — | cli | — | — | cli | cli | — | api | — | — | api | — | — | — |
| Open issues with assignees, age, and labels | cli | cli | cli | cli | cli | cli | cli | cli | api | api | api | cli | api | cli | api |
| Put under a parent | cli | — | api | cli | api | cli | cli | cli | api | api | api | — | api | cli | — |
| Iterations | — | api | cli | cli | cli | cli | cli | api | cli | api | — | — | — | — | api |
| Milestones or versions | api | — | cli | — | api | cli | cli | — | cli | api | api | cli | api | api | — |
| Status field | — | cli | api | cli | cli | cli | cli | cli | cli | api | api | — | api | cli | api |

How a change reaches an issue in each tracker:

| Tracker | What links a change | What closes the issue on merge |
| --- | --- | --- |
| GitHub Issues | a branch from `gh issue develop`, a closing keyword | `Closes #${n}` in the pull request's body |
| GitLab | a branch named `${n}-${slug}`, a closing pattern | `Closes #${n}` in the merge request's description |
| Azure Boards | `AB#${n}` in a GitHub pull request, the Azure Repos pull request's work items | `Fixes AB#${n}`; Azure Repos' option to complete linked work items |
| Jira | the key in a branch, commit, or pull request title | a smart commit's transition, or `kit settle` |
| Linear | the identifier in the branch, title, or description | a closing word before the identifier |
| YouTrack | the ID in a commit or pull request | a command in a commit message, where the integration allows it |
| Shortcut | `sc-${id}` in the branch, `[sc-${id}]` in the title or description | the workspace's workflow automation |
| Plane | the reference in a pull request | the integration's state mapping |
| Redmine | `refs #${n}` in a commit | `fixes #${n}` in a commit |
| Gitea and Forgejo | a closing keyword | `Closes #${n}` in the pull request's description |
| OpenProject | `OP#${n}` in the pull request's description | none; `kit settle` would close it |
| Bugzilla | none | none; `kit settle` would close it |
| Taiga | `TG-${n}` in a commit message | `TG-${n} #closed` in a commit message |
