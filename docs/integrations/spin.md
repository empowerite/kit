# spin

`spin` is Spinnaker's own CLI. Its repository, spinnaker/spin, is archived; the CLI lives in the
[spinnaker/spinnaker monorepo's `spin/`](https://github.com/spinnaker/spinnaker/tree/main/spin). It talks to Gate,
Spinnaker's API server, set with `--gate-endpoint` or `~/.spin/config`, which also holds the authentication. It
prints JSON by default; `-o yaml` and `-o jsonpath=${path}` are the others.

This page says what `spin` reads and writes where it meets kit's scope, for each service it reaches, and the
machine-readable descriptions each publishes. The index of these pages is [1st.md](1st.md).

| Service | Category | kit status |
| --- | --- | --- |
| [Spinnaker](#spinnaker) | [CD](cd.md) | none |

## Spinnaker

Spinnaker has applications, pipelines, and their executions; it has no environment object, and a stage or a cloud
account stands for one. A manual judgment stage is its approval. Server groups say what runs where.

### Read from Spinnaker

| Capability | Command |
| --- | --- |
| List applications | `spin application list` |
| List pipelines | `spin pipeline list --application "${app}"` |
| Deployments (executions) | `spin pipeline execution list --pipeline-id "${pipeline_id}" --limit "${n}"` |
| A deployment's state and its manual judgments | `spin pipeline execution get "${execution_id}"` |
| What is live where | `GET /applications/${app}/serverGroups` on Gate (raw API) |

### Write to Spinnaker

| Capability | Command |
| --- | --- |
| Deploy or promote | `spin pipeline execute --application "${app}" --name "${pipeline}" --parameter "${key}=${value}"`; `--wait` |
| Cancel | `spin pipeline execution cancel "${execution_id}"` |
| Roll back | a pipeline with a rollback stage, run with `spin pipeline execute` |
| Approve | `PATCH /pipelines/${execution_id}/stages/${stage_id}` on Gate with `{"judgmentStatus":"continue"}` (raw API) |

Gaps: no command reads server groups or answers a manual judgment.

### Spinnaker schemas

- Gate's OpenAPI document is generated at build time by `gate/swagger/generate_swagger.sh` and is not committed or
  published. A running Gate with `swagger.enabled: true` serves it; no raw document URL is given here.
- No JSON Schema for pipeline JSON was found, and SchemaStore has no entry.
