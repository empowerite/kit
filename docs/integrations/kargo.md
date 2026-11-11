# kargo

`kargo` is the project's own CLI. It authenticates with `kargo login "${server}"` and one of `--sso`, `--admin`, or
`--kubeconfig`. `-o json` prints JSON, as kubectl does. Kargo keeps its state in custom resources, so
`kubectl get -o json` reads the same objects.

This page says what `kargo` reads and writes where it meets kit's scope, for each service it reaches, and the
machine-readable descriptions each publishes. The index of these pages is [1st.md](1st.md).

| Service | Category | kit status |
| --- | --- | --- |
| [Kargo](#kargo) | [CD](cd.md) | none |

## Kargo

Kargo promotes between environments that Argo CD syncs. A Warehouse finds new freight: a set of commits, images, and
charts. A Stage is an environment, and a Promotion moves freight into it by steps that commit to the GitOps
repository, can open and wait on a pull request there (`git-open-pr`, `git-wait-for-pr`), and update Argo CD
(`argocd-update`). Freight approved for a stage can be promoted to it without passing through the stages before.

### Read from Kargo

| Capability | Command |
| --- | --- |
| List environments (stages) and what is live | `kargo get stages --project "${project}" -o json` |
| Freight, and the stages it is approved for | `kargo get freight --project "${project}" -o json` |
| Deployments (promotions) | `kargo get promotions --project "${project}" --stage "${stage}" -o json` |

### Write to Kargo

| Capability | Command |
| --- | --- |
| Promote | `kargo promote --project "${project}" --freight "${freight}" --stage "${stage}"`; `--downstream-from "${stage}"` promotes to the stages after it |
| Roll back | `kargo promote` with earlier freight |
| Stop a promotion | `kargo promote --project "${project}" --name "${promotion}" --abort` |
| Approve | `kargo approve --project "${project}" --freight "${freight}" --stage "${stage}"` |
| Sync | `kargo refresh stage "${stage}" --project "${project}"` |

Gaps: Kargo's pull requests go to the GitOps repository, which may not be the repository kit works in.

### Kargo schemas

- CRDs with their OpenAPI v3 schemas, among them the
  [Kargo Stage CRD](https://raw.githubusercontent.com/akuity/kargo/main/charts/kargo/resources/crds/kargo.akuity.io_stages.yaml),
  [Kargo Freight CRD](https://raw.githubusercontent.com/akuity/kargo/main/charts/kargo/resources/crds/kargo.akuity.io_freights.yaml),
  and [Kargo Promotion CRD](https://raw.githubusercontent.com/akuity/kargo/main/charts/kargo/resources/crds/kargo.akuity.io_promotions.yaml).
- [Kargo API Swagger 2.0 document](https://raw.githubusercontent.com/akuity/kargo/main/swagger.json).
- [Stage JSON Schema in the CRDs catalog](https://raw.githubusercontent.com/datreeio/CRDs-catalog/main/kargo.akuity.io/stage_v1alpha1.json),
  community.
