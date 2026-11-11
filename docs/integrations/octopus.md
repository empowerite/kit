# octopus

`octopus` is Octopus Deploy's own CLI, written in Go. It authenticates with `octopus login`, or `OCTOPUS_URL` and
`OCTOPUS_API_KEY`. `--output-format json` prints JSON. `octopus api "${path}"` makes a raw GET request and prints the
JSON response; it sends nothing else, so a write with no command is a plain REST call with the `X-Octopus-ApiKey`
header.

This page says what `octopus` reads and writes where it meets kit's scope, for each service it reaches, and the
machine-readable descriptions each publishes. The index of these pages is [1st.md](1st.md).

| Service | Category | kit status |
| --- | --- | --- |
| [Octopus Deploy](#octopus-deploy) | [CD](cd.md) | none |

## Octopus Deploy

Octopus has its own releases, a numbered snapshot of a project's packages and process, separate from git tags, and
deploys a release to environments in the order a lifecycle sets. Manual intervention steps are its approvals, kept as
interruptions. The dashboard says which release is in each environment.

### Read from Octopus Deploy

| Capability | Command |
| --- | --- |
| List releases | `octopus release list --project "${project}" --output-format json` |
| List environments | `octopus environment list --output-format json` |
| Deployments | `octopus api "/api/${space_id}/deployments?environments=${environment_id}"` (raw API) |
| What is live where | `octopus api "/api/${space_id}/dashboard"` (raw API) |
| Pending approvals | `octopus api "/api/${space_id}/interruptions?pendingOnly=true"` (raw API) |
| Wait on a deployment's task | `octopus task wait "${task_id}"` |

### Write to Octopus Deploy

| Capability | Command |
| --- | --- |
| Create a release | `octopus release create --project "${project}" --version "${version}"` |
| Deploy | `octopus release deploy --project "${project}" --version "${version}" --environment "${environment}"` |
| Promote | `octopus release promote --project "${project}" --from "${environment}" --to "${environment_next}"` |
| Roll back | `octopus release deploy` with the earlier `--version` |
| Approve | `POST /api/${space_id}/interruptions/${interruption_id}/responsible`, then `POST .../submit` with `{"Result":"Proceed"}` (plain REST) |

Gaps: the CLI reads deployments, the dashboard, and interruptions through `octopus api` alone, and cannot write them.
The `pendingOnly` query parameter is from the Swagger document and was not tried against a server.

### Octopus Deploy schemas

- Every Octopus server serves its own Swagger 2.0 document at `/api/swagger.json`; the
  [Octopus samples instance's Swagger document](https://samples.octopus.app/api/swagger.json) is one, version
  2026.4. It holds `/{spaceId}/dashboard`, `/{spaceId}/deployments`, and `/{spaceId}/interruptions`.
- No JSON Schema for its configuration-as-code files was found, and SchemaStore has no entry.
