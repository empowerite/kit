# bk

`bk` is Buildkite's CLI. It signs in with `bk auth login` or reads `BUILDKITE_API_TOKEN`. Its commands take `-o json`
(or `--json`), and `bk api` calls the REST API, or the GraphQL API with `-f`. The REST API pages with `page` and
`per_page`. Neither filters by what started a build; a build's `source` is `webhook`, `api`, `ui`, `trigger_job`, or
`schedule`, and its `pull_request` is empty for a branch build.

This page says what `bk` reads and writes where it meets kit's scope, for each service it reaches, and the
machine-readable descriptions each publishes. The index of these pages is [1st.md](1st.md).

| Service | Category | kit status |
| --- | --- | --- |
| [Buildkite](#buildkite) | [CI](ci.md) | `none` |

## Buildkite

### Read from Buildkite

| Capability | Command |
| --- | --- |
| Recent runs | `bk build list -p "${org}/${pipeline}" --branch "${branch}" --limit 20 -o json` |
| A run's jobs and failed job | `bk build view "${build}" -o json`; `bk job list --pipeline "${pipeline}" --build "${build}" --state failed` |
| A job's log | `bk job log "${job_id}"` |
| Artifacts | `bk artifacts list "${build}"`; `bk artifacts download "${artifact_id}" --build "${build}"` |

### Write to Buildkite

| Capability | Command |
| --- | --- |
| Rerun | `bk build rebuild "${build}"`; `bk build retry-failed-jobs "${build}"`; `bk job retry "${job_id}"` |
| Cancel | `bk build cancel "${build}" -p "${pipeline}"` |
| Start a run | `bk build create -b "${branch}" -c "${sha}" -m "${message}"` |
| Report a status | Buildkite posts it; see [who reports to the code forge](ci.md#who-reports-to-the-code-forge) |

A Buildkite job is a command step; the steps inside a script are its log's lines.

### Buildkite schemas

- [Buildkite pipeline JSON Schema](https://raw.githubusercontent.com/buildkite/pipeline-schema/main/schema.json),
  SchemaStore's "Buildkite" entry
- [Buildkite GraphQL schema](https://raw.githubusercontent.com/buildkite/cli/main/schema.graphql), the copy the CLI
  is built from
- No OpenAPI document for the REST API was found.
