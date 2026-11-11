# taiga

Taiga has no command line tool of its own, so this page says what its REST API reads and writes where it meets
kit's scope, and the machine-readable descriptions it publishes. The index of these pages is [1st.md](1st.md).

| Service | Category | kit status |
| --- | --- | --- |
| [Taiga](#taiga) | [tracker](trackers.md) | none |

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
