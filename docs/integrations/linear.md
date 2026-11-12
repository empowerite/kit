# linear

This page says what `linear` reads and writes where it meets kit's scope, for each service it reaches, and the
machine-readable descriptions each publishes. The index of these pages is [1st.md](1st.md).

| Service | Category | kit status |
| --- | --- | --- |
| [Linear](#linear) | [tracker](trackers.md) | none |

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
