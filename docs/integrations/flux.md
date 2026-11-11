# flux

`flux` is the project's own CLI and talks to the cluster through the kubeconfig. `flux get` prints tables alone; it
has no JSON or YAML output. `flux export` prints YAML of the objects' specs, without status. JSON comes from
`kubectl get` on Flux's custom resources, which is the API kit would read.

This page says what `flux` reads and writes where it meets kit's scope, for each service it reaches, and the
machine-readable descriptions each publishes. The index of these pages is [1st.md](1st.md).

| Service | Category | kit status |
| --- | --- | --- |
| [Flux](#flux) | [CD](cd.md) | none |

## Flux

A Kustomization or a HelmRelease applies what a source (a GitRepository, an OCIRepository) holds at a revision. An
environment is one of them, usually per cluster. `.status.lastAppliedRevision` is the revision that is live, and the
`Ready` condition is its health; a HelmRelease also keeps `.status.history`. Flux has no promotion or approval; a
promotion is a commit, and a rollback is a revert. Its notification controller writes a commit status to GitHub,
GitLab, Gitea, Bitbucket, or Azure DevOps when a Kustomization applies a commit, and can send a GitHub
`repository_dispatch`.

### Read from Flux

| Capability | Command |
| --- | --- |
| List environments | `flux get kustomizations -A`, `flux get helmreleases -A` (tables); `kubectl get kustomizations.kustomize.toolkit.fluxcd.io -A -o json` |
| What is live where, and health | `kubectl get kustomizations.kustomize.toolkit.fluxcd.io -A -o json`; `.status.lastAppliedRevision`, `.status.conditions` |
| Deployments of a Helm release | `kubectl get helmreleases.helm.toolkit.fluxcd.io "${name}" -n "${namespace}" -o json`; `.status.history` |
| What happened | `flux events --for "Kustomization/${name}"` |
| What an apply would change | `flux diff kustomization "${name}" --path "${path}"` |

### Write to Flux

| Capability | Command |
| --- | --- |
| Sync | `flux reconcile kustomization "${name}" --with-source` |
| Hold or release an environment | `flux suspend kustomization "${name}"`, `flux resume kustomization "${name}"` |
| Promote or roll back | a commit, or a revert, to the branch the source tracks; no `flux` command |

Gaps: no JSON from `flux` itself, and a Kustomization keeps no deployment history; past revisions are the git log.

### Flux schemas

- CRDs with their OpenAPI v3 schemas, per controller:
  [Kustomization CRD](https://raw.githubusercontent.com/fluxcd/kustomize-controller/main/config/crd/bases/kustomize.toolkit.fluxcd.io_kustomizations.yaml),
  [HelmRelease CRD](https://raw.githubusercontent.com/fluxcd/helm-controller/main/config/crd/bases/helm.toolkit.fluxcd.io_helmreleases.yaml),
  [GitRepository CRD](https://raw.githubusercontent.com/fluxcd/source-controller/main/config/crd/bases/source.toolkit.fluxcd.io_gitrepositories.yaml),
  and the notification controller's
  [Provider CRD](https://raw.githubusercontent.com/fluxcd/notification-controller/main/config/crd/bases/notification.toolkit.fluxcd.io_providers.yaml).
- Every fluxcd/flux2 release publishes all of them: the
  [Flux install manifest](https://github.com/fluxcd/flux2/releases/latest/download/install.yaml) holds the CRDs, and
  the [Flux CRD JSON Schema bundle](https://github.com/fluxcd/flux2/releases/latest/download/crd-schemas.tar.gz) holds
  them as JSON Schema; a release's own copy is under `releases/download/v2.10.0/`.
- [Kustomization JSON Schema in the CRDs catalog](https://raw.githubusercontent.com/datreeio/CRDs-catalog/main/kustomize.toolkit.fluxcd.io/kustomization_v1.json)
  and [HelmRelease JSON Schema in the CRDs catalog](https://raw.githubusercontent.com/datreeio/CRDs-catalog/main/helm.toolkit.fluxcd.io/helmrelease_v2.json),
  community. SchemaStore has no Flux entry.
