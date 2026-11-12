# op

This page says what `op` reads and writes where it meets kit's scope, for each service it reaches, and the
machine-readable descriptions each publishes. The index of these pages is [1st.md](1st.md).

| Service | Category | kit status |
| --- | --- | --- |
| [OpenProject](#openproject) | [tracker](trackers.md) | none |

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
