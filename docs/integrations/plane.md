# plane

Plane has no command line tool of its own, so this page says what its REST API reads and writes where it meets
kit's scope, and the machine-readable descriptions it publishes. The index of these pages is [1st.md](1st.md).

| Service | Category | kit status |
| --- | --- | --- |
| [Plane](#plane) | [tracker](trackers.md) | none |

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
