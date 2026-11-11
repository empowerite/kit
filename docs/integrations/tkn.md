# tkn

`tkn` is Tekton's CLI. It signs in through the kubeconfig and takes `-n` for the namespace. `list` and `describe` take
`-o json`; lists have `--limit` and no pages. A run is a `PipelineRun` of `TaskRun`s, each with steps. Tekton keeps
no history once runs are pruned; Tekton Results, with its `tkn-results` CLI, stores it. Pipelines-as-Code ties
Tekton to a code forge: it starts runs from `.tekton/` on each event and posts their results back.

This page says what `tkn` reads and writes where it meets kit's scope, for each service it reaches, and the
machine-readable descriptions each publishes. The index of these pages is [1st.md](1st.md).

| Service | Category | kit status |
| --- | --- | --- |
| [Tekton](#tekton) | [CI](ci.md) | `none` |

## Tekton

### Read from Tekton

| Capability | Command |
| --- | --- |
| Recent runs | `tkn pipelinerun list --limit 20 -o json`; `--label` selects |
| A run's tasks and failed step | `tkn pipelinerun describe "${run}" -o json` |
| A run's log | `tkn pipelinerun logs "${run}" --log-failed` |
| Artifacts | none in the CLI |

### Write to Tekton

| Capability | Command |
| --- | --- |
| Rerun | `tkn pipeline start "${pipeline}" --use-pipelinerun "${run}"`; with Pipelines-as-Code, a `/retest` comment on the change |
| Cancel | `tkn pipelinerun cancel "${run}"` |
| Start a run | `tkn pipeline start "${pipeline}" -p "${key}=${value}"` |
| Report a status | Pipelines-as-Code posts it; see [who reports to the code forge](ci.md#who-reports-to-the-code-forge) |

### Tekton schemas

- Tekton's custom resource definitions (CRDs), such as [the PipelineRun CRD](https://raw.githubusercontent.com/tektoncd/pipeline/main/config/300-crds/300-pipelinerun.yaml);
  their `openAPIV3Schema` leaves much of each spec open with `x-kubernetes-preserve-unknown-fields`
- [Tekton Pipelines v1 Swagger](https://raw.githubusercontent.com/tektoncd/pipeline/main/pkg/apis/pipeline/v1/swagger.json),
  the full type definitions
- [Tekton Results OpenAPI](https://raw.githubusercontent.com/tektoncd/results/main/docs/api/openapi.yaml)
- SchemaStore's Tekton entries, from Red Hat's editor extension: [Pipeline](https://raw.githubusercontent.com/redhat-developer/vscode-tekton/refs/heads/main/scheme/tekton.dev/v1_Pipeline.json),
  [PipelineRun](https://raw.githubusercontent.com/redhat-developer/vscode-tekton/refs/heads/main/scheme/tekton.dev/v1_PipelineRun.json),
  [Task](https://raw.githubusercontent.com/redhat-developer/vscode-tekton/refs/heads/main/scheme/tekton.dev/v1_Task.json),
  and [TaskRun](https://raw.githubusercontent.com/redhat-developer/vscode-tekton/refs/heads/main/scheme/tekton.dev/v1_TaskRun.json)
