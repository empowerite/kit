# Trackers

A tracker is the system that holds a repository's issues. It can be the forge (the service that hosts
the repository, GitHub or GitLab) or a separate system such as Azure DevOps. The syntax of an issue
reference selects its tracker:

```sh
kit issue AB#4521
```

The workflow commands look up issues through one seam, `kit/modules/bash/tracker.sh`;
[architecture.md](architecture.md#forges-seams-and-providers) says what a seam is. A repository can use more than one
tracker at once, for example GitHub issues for bugs and Azure DevOps for planned work.

## References

A reference names its tracker in its syntax, the way the trackers already write it:

| Reference | Tracker |
| --- | --- |
| `7` or `#7` | the forge's own issues, GitHub's or GitLab's |
| `AB#4521` | an Azure DevOps work item, under a prefix the repository declares |

`.kit/settings.yml` declares every tracker except the forge's own. Each record has:

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

`#` is the forge's own tracker unless a record claims it. A reference that matches no declared tracker is refused,
and the error lists the known prefixes.

## Commands

- `kit issue 7`, `kit issue AB#4521`, or both, primary first:
  - Each issue is assigned to the caller as its tracker knows the caller. For Azure DevOps, that is the
    identity `az` is signed in as.
  - The branch name starts with the reference's key: `7-<slug>` or `ab4521-<slug>`.
  - The forge links a branch to one of its own issues and closes the issue on merge. For any other issue, and
    for every secondary issue, `kit issue` adds a `Closes <reference>` line to the body of the change (the
    pull request or merge request). GitHub's Azure Boards integration reads that line as a link.
- `kit linked` passes when the forge reports that the change closes an open issue of the repository. It also
  passes when the change's body names an open issue of a declared tracker with a closing keyword:
  `Closes AB#4521`, `Fixes AB#4521`, or `Resolves AB#4521`.
- `kit tidy` reads a worktree's issue from its branch's key. It removes the worktree when that issue is closed and
  the branch has no change.

## Providers

| Provider | Tool | Fields | Closed when the state is |
| --- | --- | --- | --- |
| `forge` | the forge's own, through `kit/modules/bash/forge.sh` | none | the forge's closed state |
| `ado` | `az` with the `azure-devops` extension | `organization`, `project`, `type` (default `Task`) | Closed, Done, or Removed |

A provider is `kit/modules/bash/tracker-<name>.sh`. It implements these calls:

| Call | Prints |
| --- | --- |
| `title <id>` | the issue's title |
| `state <id>` | `OPEN` or `CLOSED` |
| `assignees <id>` | the assignees' logins, space-separated |
| `assign <id> <login>` | nothing; adds the assignee |
| `me` | the caller's login in this tracker, which can differ from the forge's |
| `create <title>` | the number of the issue it creates |
| `url <id>` | the issue's browser URL |

`tests/tracker.bats` tests the Azure DevOps provider against a fake `az`. It cannot prove that a real organization's
`az` responds the way the fixtures do.
