# argocd

`argocd` is the project's own CLI. It authenticates with `argocd login "${server}"`, or `ARGOCD_SERVER` and
`ARGOCD_AUTH_TOKEN`; `--core` reads the cluster's Kubernetes API instead of the Argo CD server. `-o json` prints JSON
from `app list`, `app get`, `app sync`, and `app rollback`; `app history` prints text alone, and the same history is
`.status.history` in `app get -o json`. A list is one response, with no pages.

This page says what `argocd` reads and writes where it meets kit's scope, for each service it reaches, and the
machine-readable descriptions each publishes. The index of these pages is [1st.md](1st.md).

| Service | Category | kit status |
| --- | --- | --- |
| [Argo CD](#argo-cd) | [CD](cd.md) | none |

## Argo CD

An Application tracks a path at a revision of a repository and syncs a cluster to it. An environment is an
Application, or a set of them from an ApplicationSet, one per cluster or namespace. `.status.sync.revision` is the
commit that is live, `.status.sync.status` is `Synced` or `OutOfSync`, and `.status.health.status` is the health.
Argo CD has no approvals; promotion is a commit to the tracked branch, reviewed as any other change. Its
notifications controller can write a GitHub deployment, commit status, check run, or pull request comment when a sync
ends.

### Read from Argo CD

| Capability | Command |
| --- | --- |
| List environments (Applications) | `argocd app list -o json`; `-p "${project}"`, `-l "${selector}"`, `-r "${repo_url}"` |
| What is live where, and health | `argocd app get "${app}" -o json`; `--refresh` reads the repository again first |
| Deployments of an environment | `argocd app get "${app}" -o json`, then `.status.history` |
| What a sync would change | `argocd app diff "${app}"` |

### Write to Argo CD

| Capability | Command |
| --- | --- |
| Sync | `argocd app sync "${app}"`; `--revision "${sha}"` syncs to that revision; `--prune`, `--dry-run`, `--async` |
| Roll back | `argocd app rollback "${app}" "${history_id}"`; refused while automated sync is on |
| Stop a sync | `argocd app terminate-op "${app}"` |
| Promote | a commit to the branch the Application tracks; no `argocd` command |
| Act on a resource, such as resuming a Rollout | `argocd app actions run "${app}" "${action}" --kind "${kind}" --resource-name "${name}"` |

Gaps: `app history` has no JSON output. Argo CD keeps no notion of a release; it knows revisions.

### Argo CD schemas

- [Argo CD REST Swagger 2.0 document](https://raw.githubusercontent.com/argoproj/argo-cd/master/assets/swagger.json),
  the API `argocd` and the web interface use; it holds `/api/v1/applications`.
- [Argo CD Application CRD](https://raw.githubusercontent.com/argoproj/argo-cd/master/manifests/crds/application-crd.yaml)
  and [ApplicationSet CRD](https://raw.githubusercontent.com/argoproj/argo-cd/master/manifests/crds/applicationset-crd.yaml),
  each a custom resource definition (CRD) with its OpenAPI v3 schema.
- [Argo CD resources JSON Schema](https://raw.githubusercontent.com/argoproj/argo-schema-generator/main/schema/argo_cd_kustomize_schema.json),
  SchemaStore's "Argo CD" entry ("Argo CD base resources").
- [Application JSON Schema in the CRDs catalog](https://raw.githubusercontent.com/datreeio/CRDs-catalog/main/argoproj.io/application_v1alpha1.json),
  community (datreeio/CRDs-catalog, MIT), converted from the CRD.
