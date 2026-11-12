# bugzilla

This page says what `bugzilla` reads and writes where it meets kit's scope, for each service it reaches, and the
machine-readable descriptions each publishes. The index of these pages is [1st.md](1st.md).

| Service | Category | kit status |
| --- | --- | --- |
| [Bugzilla](#bugzilla) | [tracker](trackers.md) | none |

## Bugzilla

Bugzilla is an open-source bug tracker (MPL-2.0) that a project hosts. A bug has one assignee, a status and a
resolution (`FIXED`, `WONTFIX`, `DUPLICATE`, and others), dependencies, and a target milestone.

Bugzilla has no CLI of its own. `bugzilla` is a community CLI from the python-bugzilla project
([python-bugzilla/python-bugzilla](https://github.com/python-bugzilla/python-bugzilla)). `bugzilla login` stores an
API key, and `--json` prints JSON from `query`. The REST API answers under `/rest`.

### Read from Bugzilla

| Capability | Command |
| --- | --- |
| Title, status, and assignee | `bugzilla query --bug_id "${n}" --json` |
| Caller | `GET ${bugzilla}/rest/whoami` (raw REST) |
| Issue page | none; `${bugzilla}/show_bug.cgi?id=${n}` |
| New-issue page | none; `${bugzilla}/enter_bug.cgi?product=${product}&short_desc=${title}` |
| Open issues | `bugzilla query --product "${product}" --status NEW,ASSIGNED,REOPENED --json` |
| Milestones | `GET ${bugzilla}/rest/product?names=${product}` (raw REST), its `milestones` |

### Write to Bugzilla

| Capability | Command |
| --- | --- |
| Assign | `bugzilla modify "${n}" --assignee "${login}"` |
| Unassign | `bugzilla modify "${n}" --reset-assignee`, back to the component's default |
| Close as completed | `bugzilla modify "${n}" --close FIXED` |
| Close as not planned | `bugzilla modify "${n}" --close WONTFIX` |
| Close as a duplicate | `bugzilla modify "${n}" --dupeid "${m}"` |
| Comment | `bugzilla modify "${n}" --comment "${text}"` |
| Create | `bugzilla new --product "${product}" --component "${component}" --summary "${title}" --comment "${body}"` |
| Put under a parent | `bugzilla modify "${parent}" --dependson "${n}"`, a tracking bug that depends on it |
| Link a change | none |

Gaps: no parent, no iterations, and no link from a pull request; the open statuses are an installation's own.

### Bugzilla schemas

- None. [The Bugzilla REST API](https://bugzilla.readthedocs.io/en/latest/api/) is documented in prose.
