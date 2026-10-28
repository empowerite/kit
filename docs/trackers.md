# Trackers

An issue lives in a tracker, and the tracker is not always the forge. GitHub issues and GitLab issues live in their
forges; Azure DevOps work items do not, and a repository may use more than one tracker at once, as one that files bugs
in GitHub issues and plans its work in Azure DevOps does. The loop's verbs ask about an issue through one seam,
`tracker:` in `bin/tracker.sh`, and the reference says which tracker answers.

## References

A reference names its tracker in its syntax, the way the trackers already write it:

| Reference | Tracker |
| --- | --- |
| `7` or `#7` | the forge's own issues, GitHub's or GitLab's |
| `AB#4521` | an Azure DevOps work item, under a prefix the repository declares |

`.kit/settings.yml` declares every tracker but the forge's own, one record each: the prefix a reference opens with,
the provider that answers it, and the provider's own fields.

```yaml
trackers:
  - prefix: "AB#"
    provider: ado
    organization: https://dev.azure.com/example
    project: Claims
    type: Task
```

`#` is the forge's own tracker unless a record claims it. A reference no declared tracker answers is refused, naming
the prefixes there are.

## The verbs

- `kit issue 7`, `kit issue AB#4521`, or both, primary first. Each issue is assigned to the caller as its tracker
  knows the caller, which for Azure DevOps is the identity `az` is signed in as. The branch opens with the
  reference's key, `7-<slug>` or `ab4521-<slug>`. The forge links a branch to one of its own issues and closes it on
  merge; for any other, and for every secondary, `kit issue` adds a `Closes <reference>` line to the change's body,
  which is what GitHub's Azure Boards integration reads as a link.
- `kit linked` accepts a change that the forge says closes an open issue of the repository, or whose body names an
  open issue of a declared tracker with a closing keyword: `Closes AB#4521`, `Fixes AB#4521`, `Resolves AB#4521`.
- `kit tidy` reads a worktree's issue back from its branch's key and removes the worktree when the issue is closed
  and the change is gone.

## Providers

| Provider | Tool | Fields | Closed in |
| --- | --- | --- | --- |
| `forge` | the forge's own, through `bin/forge.sh` | none | the forge's closed state |
| `ado` | `az` with the `azure-devops` extension | `organization`, `project`, `type` | Closed, Done, Removed |

A provider is `bin/tracker-<name>.sh`, answering `title`, `state`, `assignees`, `assign`, `me`, `create`, and `url`.
`tests/tracker.bats` drives the Azure DevOps provider through a fake of `az`; what only a real organization can
prove, that `az` answers as the fixtures do, is not proven there.
