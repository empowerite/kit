# Continuous delivery and deployment

Continuous delivery and deployment (CD) is the part of the work that turns a commit into something running: a release
made from a tag, deployments of that release to environments, and the record of which revision is live where. kit
starts a release and reads what came of it; the rest belongs to the repository's own pipeline or GitOps tool. The
index of these pages is [1st.md](1st.md).

`kit release ${version}` tags HEAD as `release/${version}` and pushes that tag alone, with git and no code forge
call. The repository's release workflow, if it has one, builds and publishes the release. The releases stage of
`kit flow` lists the code forge's releases, newest first, through `forge:releases` in `kit/modules/bash/forge.sh`, so
GitHub and GitLab answer it today. kittui's Delivery screen has environments and deployments tabs with no data source
yet. No seam has calls for environments, deployments, or approvals.
see: [architecture.md](../architecture.md#code-forges-seams-and-providers), `kit/modules/bash/releases.sh`

GitOps tools (Argo CD, Flux, Kargo) read the repository rather than being called by it. They watch a branch or a tag
and make a cluster match it, so the write that deploys is a commit, and kit already makes commits through
`kit work` and `kit submit`. What kit would show from them is their read side: which revision each application or
stage has synced, and whether it is healthy. Several of them also write that state back to the code forge as a
deployment or a commit status, which kit could read from the code forge with no GitOps tool installed.

The builds that produce a release are on [ci.md](ci.md); the code forges themselves are on [forges.md](forges.md).

## Scope

| Capability | Kind | Used by |
| --- | --- | --- |
| List releases, newest first, with kind (draft, pre-release, release) and latest | read | `forge:releases`, the releases stage of `kit flow` |
| Create a release from a tag and publish its notes | write | the repository's release workflow, after `kit release` pushes `release/${version}`; kit makes no call |
| List environments | read | planned: kittui's environments tab |
| List deployments per environment, with their state | read | planned: kittui's deployments tab |
| Show the revision live in each environment, and its health | read | planned: kittui's environments tab |
| Promote, sync, or roll back | write | none; for a GitOps tool the write is a commit, made by `kit work` and `kit submit` |
| Read the approvals a deployment waits on | read | planned: kittui's deployments tab |
| Approve or reject a deployment | write | planned |
| Deployment status on the change, so the pull request or merge request shows it | write by the CD tool, read by kit | planned: read from the code forge, as checks are |

## Providers

| Provider | Hosting | CLI | kit status | JSON output | Machine-readable descriptions |
| --- | --- | --- | --- | --- | --- |
| [GitHub Releases, Environments, and Deployments](gh.md#github-releases-and-deployments) | SaaS; GitHub Enterprise Server self-hosted | `gh`, official ([cli/cli](https://github.com/cli/cli), MIT) | `provider` for releases; environments and deployments planned | `--json "${fields}"`, `--jq`; `gh api` | REST OpenAPI, GraphQL schema, workflow JSON Schema |
| [GitLab Releases, environments, and deployments](glab.md#gitlab-releases-and-deployments) | SaaS; self-hosted (Community Edition MIT, Enterprise Edition proprietary) | `glab`, official ([gitlab-org/cli](https://gitlab.com/gitlab-org/cli), MIT) | `provider` for releases; environments and deployments planned | `--output json`, `--jq`; `glab api` | REST OpenAPI 3.0, CI JSON Schema |
| [Argo CD](argocd.md#argo-cd) | self-hosted, open source (Apache-2.0); hosted by vendors | `argocd`, official ([argoproj/argo-cd](https://github.com/argoproj/argo-cd)) | none | `-o json` | REST Swagger 2.0, Application CRD |
| [Azure Pipelines (YAML environments, classic releases)](az.md#azure-pipelines-environments-and-releases) | SaaS (Azure DevOps Services); Azure DevOps Server self-hosted; proprietary | `az` with the `azure-devops` extension, official ([Azure/azure-devops-cli-extension](https://github.com/Azure/azure-devops-cli-extension), MIT) | none; an `ado` code forge is planned | `--output json`, `--query` | YAML JSON Schema; REST: none published; a REST reference in prose |
| [AWS CodeDeploy and CodePipeline](aws.md#aws-codedeploy-and-codepipeline) | SaaS, proprietary | `aws`, official ([aws/aws-cli](https://github.com/aws/aws-cli), Apache-2.0) | none; an `aws` code forge is planned | JSON by default, `--query` | Smithy API models; no AppSpec schema |
| [Flux](flux.md#flux) | self-hosted, open source (Apache-2.0) | `flux`, official ([fluxcd/flux2](https://github.com/fluxcd/flux2)) | none | none from `flux get`; `kubectl get -o json` | CRDs and their JSON Schemas per release |
| [Octopus Deploy](octopus.md#octopus-deploy) | SaaS (Octopus Cloud); self-hosted; proprietary | `octopus`, official ([OctopusDeploy/cli](https://github.com/OctopusDeploy/cli), Apache-2.0) | none | `--output-format json`; `octopus api` (GET only) | Swagger 2.0 per instance |
| [Harness CD](harness.md#harness-cd) | SaaS; self-managed enterprise; proprietary | `harness`, official ([harness/cli](https://github.com/harness/cli), Apache-2.0) | none | `--format json` | OpenAPI (not verified) |
| [Spinnaker](spin.md#spinnaker) | self-hosted, open source (Apache-2.0) | `spin`, official ([spinnaker/spinnaker, `spin/`](https://github.com/spinnaker/spinnaker/tree/main/spin)) | none | `-o json` (the default), `-o jsonpath=` | Gate OpenAPI, generated at build, not published |
| [Argo Rollouts](kubectl.md#argo-rollouts) | self-hosted, open source (Apache-2.0) | `kubectl argo rollouts`, official ([argoproj/argo-rollouts](https://github.com/argoproj/argo-rollouts)) | none | none from the plugin; `kubectl get -o json` | Rollout CRD, Swagger 2.0 |
| [Kargo](kargo.md#kargo) | self-hosted, open source (Apache-2.0); Akuity Platform SaaS | `kargo`, official ([akuity/kargo](https://github.com/akuity/kargo)) | none | `-o json` | CRDs, Swagger 2.0 |

Each provider's name leads to its section on its tool's page: what the tool reads and writes
where it meets kit's scope, its gaps, and the descriptions the provider publishes.

The order is a judgment of how widely each is used, kit's own providers first; no survey backs it.

## Coverage

`cli` means the CLI has a command, `api` means only the raw API answers, and `—` means the provider has no such
thing. For a GitOps tool, promote and roll back are a commit to the repository, which kit makes; `—` there means the
tool has no command of its own. Flux, Argo Rollouts, and Kargo read through `kubectl` count as `cli`.

| Capability | GitHub | GitLab | Argo CD | Azure Pipelines | CodeDeploy | CodePipeline | Flux | Octopus | Harness | Spinnaker | Argo Rollouts | Kargo |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| List releases | cli | cli | — | cli | — | — | — | cli | — | — | — | — |
| Create a release from a tag, notes | cli | cli | — | cli | — | — | — | cli | — | — | — | — |
| List environments | api | api | cli | api | cli | cli | cli | cli | cli | — | — | cli |
| Deployments per environment, state | api | api | cli | api | cli | cli | cli | api | cli | cli | cli | cli |
| What is live where | api | api | cli | api | cli | cli | cli | api | api | api | cli | cli |
| Promote or deploy | cli | cli | — | cli | cli | cli | — | cli | cli | cli | cli | cli |
| Sync | — | — | cli | — | — | — | cli | — | cli | — | — | cli |
| Roll back | cli | cli | cli | api | cli | cli | — | cli | cli | cli | cli | cli |
| Read approvals | api | api | — | api | — | cli | — | api | cli | cli | cli | cli |
| Approve or reject | api | api | — | api | — | cli | — | api | cli | api | cli | cli |
| Deployment status on the change | api | api | — | — | — | — | — | — | — | — | — | — |

Notes on the cells:

- Azure Pipelines' and Octopus's releases are their own numbered objects, not git tags.
- Flux keeps deployment history for a HelmRelease alone; a Kustomization's history is the git log.
- Harness's instance routes, for what is live where, were not checked.
- For Argo Rollouts, the approval is a paused step, and `promote` answers it.
- Deployment status on the change is written by the code forge's own pipeline, or by a tool's notifications once
  they are configured in the cluster: Argo CD and Argo Rollouts write a GitHub deployment or commit status, and Flux
  a commit status on GitHub, GitLab, Gitea, Bitbucket, or Azure DevOps. No CLI command does it, so those cells are
  `—`; kit would read the result from the code forge.
