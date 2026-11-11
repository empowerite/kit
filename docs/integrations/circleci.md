# circleci

`circleci` is CircleCI's CLI. It signs in with `circleci auth login` or reads `CIRCLE_TOKEN`, and `CIRCLE_HOST` points
it at a CircleCI server. Its data commands take `--json` and `--jq`, and `circleci api` calls any route of the API it
is built on, which CircleCI calls v3. In this CLI a run is one pipeline's execution, with workflows, and jobs in them.
The CLI has no filter for runs no change owns; `--branch` comes closest.

This page says what `circleci` reads and writes where it meets kit's scope, for each service it reaches, and the
machine-readable descriptions each publishes. The index of these pages is [1st.md](1st.md).

| Service | Category | kit status |
| --- | --- | --- |
| [CircleCI](#circleci) | [CI](ci.md) | `none` |

## CircleCI

### Read from CircleCI

| Capability | Command |
| --- | --- |
| Recent runs | `circleci run list --project "${project_slug}" --branch "${branch}" --limit 20 --json` |
| A run's jobs and failed step | `circleci run get "${run_id}" --json`; `circleci job get "${job_id}" --json` gives each step's status and exit code; `--failure-report` prints the failed steps |
| A job's log | `circleci job output get "${job_id}" --step-num "${step}"` |
| Artifacts | `circleci job artifact "${job_id}" --json` lists; `-o "${dir}"` downloads |

### Write to CircleCI

| Capability | Command |
| --- | --- |
| Rerun | `circleci workflow rerun "${workflow_id}" --from-failed` |
| Cancel | `circleci run cancel "${run_id}"`; `circleci workflow cancel "${workflow_id}"` |
| Start a run | `circleci run trigger --branch "${branch}" --parameter "${key}=${value}"` |
| Report a status | CircleCI posts it; see the table above |

The published OpenAPI describes API v2, not the v3 routes the CLI calls; no v3 description was found. `circleci
config validate` checks `.circleci/config.yml`.

### CircleCI schemas

- [CircleCI API v2 OpenAPI](https://circleci.com/api/v2/openapi.json)
- [CircleCI config JSON Schema](https://raw.githubusercontent.com/CircleCI-Public/circleci-yaml-language-server/main/schema.json),
  from CircleCI's language server; SchemaStore keeps
  [its own "CircleCI config.yml" entry](https://www.schemastore.org/circleciconfig.json)
