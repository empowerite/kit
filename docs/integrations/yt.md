# yt

This page says what `yt` reads and writes where it meets kit's scope, for each service it reaches, and the
machine-readable descriptions each publishes. The index of these pages is [1st.md](1st.md).

| Service | Category | kit status |
| --- | --- | --- |
| [YouTrack](#youtrack) | [tracker](trackers.md) | none |

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
