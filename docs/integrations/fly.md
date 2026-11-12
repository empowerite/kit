# fly

`fly` is Concourse's CLI. It signs in with `fly -t "${target}" login -c "${concourse_url}"` and keeps the token in
`~/.flyrc`. List commands take `--json`; `watch`, `trigger-job`, `rerun-build`, and `abort-build` do not. `fly curl`
calls any API route. A run is a build of a job, and Concourse has no notion of a change: what starts a job is a
resource, such as a git branch or a pull request resource.

This page says what `fly` reads and writes where it meets kit's scope, for each service it reaches, and the
machine-readable descriptions each publishes. The index of these pages is [1st.md](1st.md).

| Service | Category | kit status |
| --- | --- | --- |
| [Concourse](#concourse) | [CI](ci.md) | `none` |

## Concourse

### Read from Concourse

| Capability | Command |
| --- | --- |
| Recent runs | `fly -t "${target}" builds -p "${pipeline}" -c 20 --json` |
| A run's steps | `GET /api/v1/builds/${build_id}/plan` and `/events` through `fly curl`; no command names the failed step |
| A run's log | `fly -t "${target}" watch -j "${pipeline}/${job}" -b "${build}"` |
| Artifacts | none for a job's build; outputs go to resources |

### Write to Concourse

| Capability | Command |
| --- | --- |
| Rerun | `fly -t "${target}" rerun-build -j "${pipeline}/${job}" -b "${build}"` |
| Cancel | `fly -t "${target}" abort-build -j "${pipeline}/${job}" -b "${build}"` |
| Start a run | `fly -t "${target}" trigger-job -j "${pipeline}/${job}"` |
| Report a status | a resource in the pipeline posts it |

### Concourse schemas

Concourse publishes none: no OpenAPI for its API, whose routes are in `atc/routes.go`, and no JSON Schema for its
pipeline files.
