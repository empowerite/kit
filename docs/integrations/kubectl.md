# kubectl

`kubectl argo rollouts` is the project's own kubectl plugin and talks to the cluster through the kubeconfig. Its
`get`, `list`, and `status` print text alone; JSON comes from `kubectl get rollouts.argoproj.io -o json`.

This page says what `kubectl` reads and writes where it meets kit's scope, for each service it reaches, and the
machine-readable descriptions each publishes. The index of these pages is [1st.md](1st.md).

| Service | Category | kit status |
| --- | --- | --- |
| [Argo Rollouts](#argo-rollouts) | [CD](cd.md) | none |
| [Kubernetes custom resources](#kubernetes-custom-resources) | [CD](cd.md) | — |

## Argo Rollouts

A Rollout replaces a Deployment and moves a new version in by canary or blue-green steps, pausing where the steps
say. A pause is its manual gate: promote moves past it, abort goes back to the stable version. A Rollout is usually
managed by Argo CD, so this is the step inside one environment, not the list of environments. It shares Argo CD's
notifications, so it can write to GitHub the same way.

### Read from Argo Rollouts

| Capability | Command |
| --- | --- |
| List rollouts | `kubectl argo rollouts list rollouts -A` (text); `kubectl get rollouts.argoproj.io -A -o json` |
| What is live, its steps and revisions | `kubectl argo rollouts get rollout "${name}"` (text); `kubectl get rollouts.argoproj.io "${name}" -o json` |
| Whether it is paused, progressing, or healthy | `kubectl argo rollouts status "${name}" --watch=false` |

### Write to Argo Rollouts

| Capability | Command |
| --- | --- |
| Promote past a pause | `kubectl argo rollouts promote "${name}"`; `--full` skips the remaining steps |
| Abort | `kubectl argo rollouts abort "${name}"` |
| Roll back | `kubectl argo rollouts undo "${name}" --to-revision "${revision}"` |
| Retry an aborted rollout | `kubectl argo rollouts retry rollout "${name}"` |

Gaps: no JSON from the plugin.

### Argo Rollouts schemas

- [Rollout CRD](https://raw.githubusercontent.com/argoproj/argo-rollouts/master/manifests/crds/rollout-crd.yaml),
  with its OpenAPI v3 schema.
- [Argo Rollouts Swagger 2.0 document](https://raw.githubusercontent.com/argoproj/argo-rollouts/master/pkg/apiclient/rollout/rollout.swagger.json),
  the API of `kubectl argo rollouts dashboard`.
- [Rollout JSON Schema in the CRDs catalog](https://raw.githubusercontent.com/datreeio/CRDs-catalog/main/argoproj.io/rollout_v1alpha1.json),
  community.

## Kubernetes custom resources

Argo CD, Flux, Argo Rollouts, and Kargo keep their state in Kubernetes custom resources. Every CRD carries an
OpenAPI v3 schema under `spec.versions[].schema.openAPIV3Schema`, and a cluster serves the schemas of what it has
installed at `/openapi/v3`, which `kubectl get --raw /openapi/v3` reads and `kubectl explain` renders. The
[Kubernetes OpenAPI document](https://raw.githubusercontent.com/kubernetes/kubernetes/master/api/openapi-spec/swagger.json)
and the [Kubernetes apiextensions OpenAPI v3 document](https://raw.githubusercontent.com/kubernetes/kubernetes/master/api/openapi-spec/v3/apis__apiextensions.k8s.io__v1_openapi.json)
describe the API that serves them and the CRD object itself. The
[CRDs catalog](https://github.com/datreeio/CRDs-catalog) converts many CRDs to JSON Schema.
