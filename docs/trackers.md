# Trackers

A tracker is the system that holds a repository's issues. It can be the code forge (the service that hosts
the repository, GitHub or GitLab) or a separate system such as Azure DevOps. The syntax of an issue
reference selects its tracker:

```sh
kit work AB#4521
```

The workflow commands look up issues through one seam, `kit/modules/bash/tracker.sh`;
[architecture.md](architecture.md#code-forges-seams-and-providers) says what a seam is. A repository can use more than one
tracker at once, for example GitHub issues for bugs and Azure DevOps for planned work.

## References

A reference names its tracker in its syntax, the way the trackers already write it:

| Reference | Tracker |
| --- | --- |
| `7` or `#7` | the code forge's own issues, GitHub's or GitLab's |
| `AB#4521` | an Azure DevOps work item, under a prefix the repository declares |
| `LIN#12` | an issue in a ticket system such as Linear or Bugzilla, which kit reaches through its pages alone |

`kit.yml` declares every tracker except the code forge's own. Each record has:

- `prefix`, the text a reference starts with
- `provider`, the provider that handles the reference
- the provider's own fields

```yaml
trackers:
  - prefix: "AB#"
    provider: ado
    organization: https://dev.azure.com/example
    project: Claims
    type: Task
```

`#` is the code forge's own tracker unless a record claims it. On an Azure DevOps code forge, whose own tracker is
Azure Boards, `AB#` is the forge's own too, unless a record claims it: `#7` and `AB#7` name one work item. A reference
that matches no declared tracker is refused, and the error lists the known prefixes.

## Commands

- `kit work 7`, `kit work AB#4521`, or both, primary first:
  - Each issue is assigned to the caller as its tracker knows the caller. For Azure DevOps, that is the
    identity `az` is signed in as.
  - The branch name starts with the reference's key: `7-<slug>` or `ab4521-<slug>`.
  - The code forge links a branch to one of its own issues and closes the issue on merge. For any other issue, and
    for every secondary issue, `kit work` adds a `Closes <reference>` line to the body of the change (the
    pull request or merge request). GitHub's Azure Boards integration reads that line as a link.
- `kit linked` passes when the code forge reports that the change closes an open issue of the repository. It also
  passes when the change's body names an open issue of a declared tracker with a closing keyword:
  `Closes AB#4521`, `Fixes AB#4521`, or `Resolves AB#4521`. A change by a login under `linked_exempt` in
  `kit.yml` passes without an issue, and the output says so.
- `kit work --title=<text> 7 AB#4521` gathers the issues under a new parent issue and starts the parent, as
  `kit gather` does without starting it: a forge issue is a sub-issue on GitHub, a linked item on GitLab; a work
  item is a task line in the parent's body.
- `kit assign`, `kit unassign`, and `kit close` take references from any tracker. `kit close --duplicate-of` a
  reference in another tracker comments with the original, and closes the issue as not planned.
- `kit settle` does for a closed change what the code forge does not: when it merged, it closes the work items its
  body closes, with a comment naming the merge commit; when it closed without merging, it puts every issue it names
  back in the backlog. The delivered CI runs it when a change closes, and `kit tidy` runs it once it has removed a
  finished worktree.
- `kit new issue AB#` opens the page a person files a new issue at in that tracker, and prints its URL; with no
  prefix, the code forge's own. `--print` prints the URL alone. `kit new issue --list` lists every tracker, the
  code forge's own first, with its page, and `--json` gives the list as JSON for kittui's New… in the issues tab's menu.
- `kit tidy` reads a worktree's issue from its branch's key. It removes the worktree when that issue is closed and
  the branch has no change.

## Providers

| Provider | Tool | Fields | Closed when the state is |
| --- | --- | --- | --- |
| `forge` | the code forge's own, through `kit/modules/bash/forge.sh` | none; on Azure DevOps, `forge_settings.ado` gives `type`, `closed_state`, and `removed_state` | the code forge's closed state |
| `ado` | `az` with the `azure-devops` extension | `organization`, `project`, `type` (default `Task`), `closed_state` (default `Closed`), `removed_state` (default `Removed`) | Closed, Done, or Removed |
| `web` | none: kit opens its pages | `new`, the page a new issue is filed at, `{title}` standing for the title; `url`, an issue's page, `{id}` standing for its number | never read |

A `web` tracker is a ticket system kit has no command line for. kit opens its pages and does nothing else there: the
calls that read or change an issue refuse, naming the issue's page.

```yaml
trackers:
  - prefix: "LIN#"
    provider: web
    new: https://linear.app/example/team/ENG/new?title={title}
    url: https://linear.app/example/issue/ENG-{id}
```

[integrations/trackers.md](integrations/trackers.md) lists the other trackers and which of these calls each answers;
each one's name leads to its tool's page, with the commands and the schemas it publishes.

A provider is `kit/modules/bash/tracker/<name>.sh`. It implements these calls:

| Call | Prints |
| --- | --- |
| `title <id>` | the issue's title |
| `state <id>` | `OPEN` or `CLOSED` |
| `assignees <id>` | the assignees' logins, space-separated |
| `assign <id> <login>` | nothing; adds the assignee |
| `unassign <id> <login>` | nothing; removes the assignee |
| `close <id> <reason> [<other-id>]` | nothing; closes the issue as `completed`, `not-planned`, or `duplicate` of the other issue in the same tracker |
| `comment <id> <text>` | nothing; adds a comment |
| `me` | the caller's login in this tracker, which can differ from the code forge's |
| `new-url [<title>]` | the browser URL a person files a new issue at, the title filled in where the page takes one |
| `create <title> [<body>]` | the number of the issue it creates |
| `url <id>` | the issue's browser URL |

`tests/tracker.bats` tests the Azure DevOps provider against a fake `az`, and the `web` provider and every new-issue
page against `kit.yml` alone. It cannot prove that a real organization's
`az` responds the way the fixtures do.
