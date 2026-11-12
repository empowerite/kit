# Trackers

A tracker is the system that holds a repository's issues: the code forge's own issues, or a separate system such as
Azure Boards, Jira, or Linear. This page catalogues the trackers kit reaches or could reach: what each one's command
line and API answer for kit's calls, and the machine-readable descriptions each one publishes. The seam, its
references, and its three providers are described in [the tracker seam](../trackers.md); every category of integration
is listed in [the integrations index](1st.md).

kit reaches a tracker through `kit/modules/bash/tracker.sh`. `kit work`, `kit assign`, `kit unassign`, `kit close`,
`kit gather`, `kit settle`, `kit new issue`, `kit linked`, and `kit tidy` call it. The issues stage of
`kit flow` draws the code forge's open issues, through `open-issues` in `kit/modules/bash/forge.sh`. kittui has
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
| The open issues, each with assignees, age, labels, milestone, iteration, and swimlane | read | `open-issues` (forge seam): the issues stage |
| Adds or removes an assignee | write | `assign`, `unassign`: `kit work`, `kit assign`, `kit unassign`, `kit settle` |
| Closes an issue as completed, not planned, or a duplicate | write | `close`: `kit close`, `kit close --duplicate-of`, `kit settle` |
| Adds a comment | write | `comment`: `kit settle`, `kit close --duplicate-of` across trackers |
| Creates an issue | write | `create`: `kit gather` and `kit work --title` create the parent |
| Puts an issue under a parent | write | `issue-add-sub` (forge seam): `kit gather`, `kit work --title` |
| Links a change to an issue | write | `kit work`: a linked branch, and a `Closes` line in the change's body; `kit linked` reads it |
| Iterations: sprints or cycles, with dates and their issues | read | planned: kittui's iterations tab |
| Milestones or versions, with due dates and their issues | read | planned: kittui's milestones tab |
| A project board's status field | read | planned: a column beside an issue in the issues stage |

The calls are the seam's, and a provider answers each one. kit's providers call a tracker's command line, which holds
the sign-in; a call that goes only through a tracker's raw API needs a token of its own.

## Providers

| Provider | Hosting | CLI | kit status | JSON from the CLI | Machine-readable descriptions |
| --- | --- | --- | --- | --- | --- |
| [GitHub Issues](gh.md#github-issues) | SaaS; GitHub Enterprise Server self-hosted; proprietary | `gh`, official ([cli/cli](https://github.com/cli/cli), MIT) | `provider` (`forge`) | `--json "${fields}"`, `--jq`; `gh api` | REST OpenAPI, GraphQL schema, webhook payloads in OpenAPI 3.1, issue form JSON Schema |
| [GitLab Issues and work items](glab.md#gitlab-issues-and-work-items) | SaaS; self-managed; open core (MIT core) | `glab`, official ([gitlab-org/cli](https://gitlab.com/gitlab-org/cli), MIT) | `provider` (`forge`) | `--output json`, `--jq`; `glab api` | REST OpenAPI 3.0, GraphQL introspection result |
| [Azure Boards](az.md#azure-boards) | SaaS (Azure DevOps Services); Azure DevOps Server self-hosted; proprietary | `az` with the `azure-devops` extension, official ([Azure/azure-devops-cli-extension](https://github.com/Azure/azure-devops-cli-extension), MIT) | `provider` (`ado`) | `--output json`, `--query` (JMESPath); `az devops invoke`, `az rest` | none published; a REST reference in prose |
| [GitHub Projects](gh.md#github-projects) | SaaS; GitHub Enterprise Server; proprietary | `gh project`, official | none | `--format json`, `--jq`; `gh api graphql` | GraphQL schema (`ProjectV2`), `projects_v2_item` webhooks |
| [Jira Cloud and Data Center](jira.md#jira) | SaaS; Data Center self-hosted until March 28, 2029; proprietary | `acli`, official, Cloud ([acli reference](https://developer.atlassian.com/cloud/acli/reference/commands/), proprietary); `jira`, community ([ankitpokhrel/jira-cli](https://github.com/ankitpokhrel/jira-cli), MIT) | none | `acli`: `--json`; `jira`: `--raw`, `--plain` | Cloud platform and Jira Software OpenAPI 3; Data Center OpenAPI 3 per release |
| [Linear](linear.md#linear) | SaaS; proprietary | `linear`, community ([schpet/linear-cli](https://github.com/schpet/linear-cli), ISC) | none | `--json`; `linear api` | GraphQL schema |
| [YouTrack](yt.md#youtrack) | SaaS; YouTrack Server self-hosted; proprietary | `yt`, community ([ryancheley/yt-cli](https://github.com/ryancheley/yt-cli), MIT) | none | `--format json` | OpenAPI 3 per instance |
| [Shortcut](short.md#shortcut) | SaaS; proprietary | `short`, community ([shortcut-cli/shortcut-cli](https://github.com/shortcut-cli/shortcut-cli), MIT) | none | none; `short api` | REST Swagger 2.0 |
| [Plane](plane.md#plane) | SaaS; self-hosted; open source (AGPL-3.0) | none | none | — | OpenAPI 3 per self-hosted instance, off by default |
| [Redmine](redmine.md#redmine) | self-hosted; open source (GPL-2.0) | none | none | — | none official; an unofficial OpenAPI 3 |
| [Gitea and Forgejo](tea.md#gitea-and-forgejo-issues) | self-hosted; Gitea Cloud, Codeberg; open source (Gitea MIT, Forgejo GPL-3.0) | `tea`, official for Gitea ([gitea/tea](https://gitea.com/gitea/tea), MIT); `fj`, community ([forgejo-contrib/forgejo-cli](https://codeberg.org/forgejo-contrib/forgejo-cli), Apache-2.0 or MIT) | none | `tea`: `--output json`, `tea api`; `fj`: none | REST Swagger 2.0 per instance, issue form JSON Schema |
| [OpenProject](op.md#openproject) | self-hosted; SaaS; open source (GPL-3.0) | `op`, from OpenProject's own organization ([opf/openproject-cli](https://github.com/opf/openproject-cli), GPL-3.0) | none | none found | OpenAPI 3 |
| [Bugzilla](bugzilla.md#bugzilla) | self-hosted; open source (MPL-2.0) | `bugzilla`, community ([python-bugzilla/python-bugzilla](https://github.com/python-bugzilla/python-bugzilla), GPL-2.0) | none | `--json` | none |
| [Taiga](taiga.md#taiga) | SaaS; self-hosted; open source (MPL-2.0) | none | none | — | none |

Each provider's name leads to its section on its tool's page: what the tool reads and writes
where it meets kit's scope, its gaps, and the descriptions the provider publishes.

The order is roughly by use among development teams, kit's own providers first. Any tracker here can be a `web`
tracker today: kit opens its new-issue page and an issue's page, and refuses every other call. Asana and ClickUp are
left out: development teams use them less as the tracker a branch and a change name, and kit has no reference syntax
for either.

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
