# short

This page says what `short` reads and writes where it meets kit's scope, for each service it reaches, and the
machine-readable descriptions each publishes. The index of these pages is [1st.md](1st.md).

| Service | Category | kit status |
| --- | --- | --- |
| [Shortcut](#shortcut) | [tracker](trackers.md) | none |

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
