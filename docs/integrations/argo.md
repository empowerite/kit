# argo

`argo` is Argo Workflows' CLI. It signs in through the kubeconfig, or to an Argo Server with `ARGO_SERVER` and
`ARGO_TOKEN`. It prints `-o json`, and lists page with `--chunk-size`. A run is a `Workflow` of nodes; runs that are
gone from the cluster stay in the archive, `argo archive list`. Argo Workflows has no code forge integration of its
own: Argo Events receives the code forge's webhooks, and a step posts the status. Argo CD, its sibling, is on
[argocd.md](argocd.md).

This page says what `argo` reads and writes where it meets kit's scope, for each service it reaches, and the
machine-readable descriptions each publishes. The index of these pages is [1st.md](1st.md).

| Service | Category | kit status |
| --- | --- | --- |
| [Argo Workflows](#argo-workflows) | [CI](ci.md) | `none` |

## Argo Workflows

### Read from Argo Workflows

| Capability | Command |
| --- | --- |
| Recent runs | `argo list --completed --since 24h -o json`; `-l` selects by label |
| A run's nodes and the failed one | `argo get "${workflow}" --node-field-selector phase=Failed -o json` |
| A run's log | `argo logs "${workflow}"` |
| Artifacts | `argo cp "${workflow}" "${dir}" --artifact-name "${name}"` |

### Write to Argo Workflows

| Capability | Command |
| --- | --- |
| Rerun | `argo retry "${workflow}"` resets the failed nodes; `argo resubmit "${workflow}"` starts a copy |
| Cancel | `argo stop "${workflow}"` runs exit handlers; `argo terminate "${workflow}"` does not |
| Start a run | `argo submit --from "workflowtemplate/${template}" -p "${key}=${value}"`; `argo submit --from "cronwf/${name}"` |
| Report a status | a step of the workflow posts it to the code forge |

### Argo Workflows schemas

- [Argo Workflows JSON Schema](https://raw.githubusercontent.com/argoproj/argo-workflows/main/api/jsonschema/schema.json),
  SchemaStore's "Argo Workflows" entry, which points at the `master` branch
- [Argo Workflows Swagger](https://raw.githubusercontent.com/argoproj/argo-workflows/main/api/openapi-spec/swagger.json),
  the Argo Server's API
