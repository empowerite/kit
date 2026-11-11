# redmine

Redmine has no command line tool of its own, so this page says what its REST API reads and writes where it meets
kit's scope, and the machine-readable descriptions it publishes. The index of these pages is [1st.md](1st.md).

| Service | Category | kit status |
| --- | --- | --- |
| [Redmine](#redmine) | [tracker](trackers.md) | none |

## Redmine

Redmine is an open-source tracker (GPL-2.0) that a team hosts. An issue has one assignee, a status that is closed or
not by its definition, a parent issue, relations (among them `duplicates`), and a target version.

Redmine has no CLI of its own, and no maintained community one stood out. Its REST API answers JSON at `.json`
routes, with an `X-Redmine-API-Key` header, and pages with `offset` and `limit` (at most 100).

### Read from Redmine

| Capability | Command |
| --- | --- |
| Title, status, and assignee | `GET ${redmine}/issues/${n}.json` (raw REST) |
| Caller | `GET ${redmine}/users/current.json` (raw REST) |
| Issue page | none; `${redmine}/issues/${n}` |
| New-issue page | none; `${redmine}/projects/${project}/issues/new?issue[subject]=${title}` |
| Open issues | `GET ${redmine}/issues.json?project_id=${project}&status_id=open&sort=created_on&limit=100&offset=${offset}` (raw REST) |
| Children and relations | `GET ${redmine}/issues/${n}.json?include=children,relations` (raw REST) |
| Versions | `GET ${redmine}/projects/${project}/versions.json` (raw REST) |

### Write to Redmine

| Capability | Command |
| --- | --- |
| Assign, unassign, close, or comment | `PUT ${redmine}/issues/${n}.json` with `assigned_to_id`, `status_id`, or `notes` (raw REST) |
| Close as a duplicate | `POST ${redmine}/issues/${n}/relations.json` with `relation_type: duplicates`, then a closed status (raw REST) |
| Create | `POST ${redmine}/issues.json` (raw REST) |
| Put under a parent | `PUT ${redmine}/issues/${n}.json` with `parent_issue_id` (raw REST) |
| Link a change | `refs #${n}` or `fixes #${n}` in a commit of a repository Redmine reads; the keywords are settings |

Gaps: the API must be enabled by an administrator. Redmine writes `#123` for its own issues, which kit reads as the
code forge's, so its record needs another prefix. Statuses are IDs a provider looks up.

### Redmine schemas

- No official description; see [the Redmine REST API](https://www.redmine.org/projects/redmine/wiki/Rest_api).
- [An unofficial Redmine OpenAPI 3](https://raw.githubusercontent.com/d-yoshi/redmine-openapi/main/openapi.yaml),
  tested by its author against Redmine 7.0.
