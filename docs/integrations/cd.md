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
| GitHub Releases, Environments, and Deployments | SaaS; GitHub Enterprise Server self-hosted | `gh`, official ([cli/cli](https://github.com/cli/cli), MIT) | `provider` for releases; environments and deployments planned | `--json "${fields}"`, `--jq`; `gh api` | REST OpenAPI, GraphQL schema, workflow JSON Schema |
| GitLab Releases, environments, and deployments | SaaS; self-hosted (Community Edition MIT, Enterprise Edition proprietary) | `glab`, official ([gitlab-org/cli](https://gitlab.com/gitlab-org/cli), MIT) | `provider` for releases; environments and deployments planned | `--output json`, `--jq`; `glab api` | REST OpenAPI 3.0, CI JSON Schema |
| Argo CD | self-hosted, open source (Apache-2.0); hosted by vendors | `argocd`, official ([argoproj/argo-cd](https://github.com/argoproj/argo-cd)) | none | `-o json` | REST Swagger 2.0, Application CRD |
| Azure Pipelines (YAML environments, classic releases) | SaaS (Azure DevOps Services); Azure DevOps Server self-hosted; proprietary | `az` with the `azure-devops` extension, official ([Azure/azure-devops-cli-extension](https://github.com/Azure/azure-devops-cli-extension), MIT) | none; an `ado` code forge is planned | `--output json`, `--query` | YAML JSON Schema; REST: none published; a REST reference in prose |
| AWS CodeDeploy and CodePipeline | SaaS, proprietary | `aws`, official ([aws/aws-cli](https://github.com/aws/aws-cli), Apache-2.0) | none; an `aws` code forge is planned | JSON by default, `--query` | Smithy API models; no AppSpec schema |
| Flux | self-hosted, open source (Apache-2.0) | `flux`, official ([fluxcd/flux2](https://github.com/fluxcd/flux2)) | none | none from `flux get`; `kubectl get -o json` | CRDs and their JSON Schemas per release |
| Octopus Deploy | SaaS (Octopus Cloud); self-hosted; proprietary | `octopus`, official ([OctopusDeploy/cli](https://github.com/OctopusDeploy/cli), Apache-2.0) | none | `--output-format json`; `octopus api` (GET only) | Swagger 2.0 per instance |
| Harness CD | SaaS; self-managed enterprise; proprietary | `harness`, official ([harness/cli](https://github.com/harness/cli), Apache-2.0) | none | `--format json` | OpenAPI (not verified) |
| Spinnaker | self-hosted, open source (Apache-2.0) | `spin`, official ([spinnaker/spinnaker, `spin/`](https://github.com/spinnaker/spinnaker/tree/main/spin)) | none | `-o json` (the default), `-o jsonpath=` | Gate OpenAPI, generated at build, not published |
| Argo Rollouts | self-hosted, open source (Apache-2.0) | `kubectl argo rollouts`, official ([argoproj/argo-rollouts](https://github.com/argoproj/argo-rollouts)) | none | none from the plugin; `kubectl get -o json` | Rollout CRD, Swagger 2.0 |
| Kargo | self-hosted, open source (Apache-2.0); Akuity Platform SaaS | `kargo`, official ([akuity/kargo](https://github.com/akuity/kargo)) | none | `-o json` | CRDs, Swagger 2.0 |

The order is a judgment of how widely each is used, kit's own providers first; no survey backs it.

## GitHub

`gh` is GitHub's own CLI. It authenticates with `gh auth login`, or `GH_TOKEN` in CI. `--json "${fields}"` picks the
fields, `--jq` filters them, and `--limit` sets how many items a list reads. `gh` has release commands and none for
environments or deployments, so those go through `gh api`, which fills `{owner}/{repo}` from the checkout, reads
every page with `--paginate`, and runs GraphQL with `gh api graphql`. kit reads releases with `gh release list`.

A job in GitHub Actions that names an `environment:` creates a deployment for that environment, and its run waits
there for the environment's protection rules, such as required reviewers. A deployment whose ref is a pull request's
head branch shows on that pull request. Releases can be immutable: once published, their tag and assets stay fixed.

### Read from GitHub

| Capability | Command |
| --- | --- |
| List releases | `gh release list --limit "${limit}" --json tagName,name,isPrerelease,isDraft,isLatest,isImmutable,publishedAt,createdAt` |
| A release's notes | `gh release view "${tag}" --json body,url` |
| List environments | `gh api --paginate 'repos/{owner}/{repo}/environments?per_page=100'` (raw API) |
| Deployments in an environment | `gh api --paginate "repos/{owner}/{repo}/deployments?environment=${environment}&per_page=100"` (raw API) |
| A deployment's state | `gh api --paginate "repos/{owner}/{repo}/deployments/${deployment_id}/statuses"` (raw API); the first status is the newest |
| What is live where | `gh api graphql` on `repository.environments.nodes { name latestCompletedDeployment { sha ref { name } latestStatus { state } } }` (raw API) |
| Approvals a run waits on | `gh api "repos/{owner}/{repo}/actions/runs/${run_id}/pending_deployments"` (raw API) |
| Approvals given | `gh api "repos/{owner}/{repo}/actions/runs/${run_id}/approvals"` (raw API) |

### Write to GitHub

| Capability | Command |
| --- | --- |
| Create a release from a tag with notes | `gh release create "${tag}" --verify-tag --generate-notes` or `--notes-file "${file}"`; `--prerelease`, `--draft` |
| Publish or change notes | `gh release edit "${tag}" --notes-file "${file}"` |
| Generate notes without a release | `gh api 'repos/{owner}/{repo}/releases/generate-notes' -f tag_name="${tag}"` (raw API) |
| Deploy or promote | `gh workflow run "${workflow}" --ref "${ref}" -f environment="${environment}"`, when the workflow takes that input |
| Roll back | `gh run rerun "${run_id}"` on the run that deployed the earlier commit; GitHub re-runs a run up to 30 days after it first ran |
| Approve or reject | `gh api -X POST "repos/{owner}/{repo}/actions/runs/${run_id}/pending_deployments" -F 'environment_ids[]'="${environment_id}" -f state=approved -f comment="${comment}"` (raw API) |
| Record a deployment made elsewhere | `gh api 'repos/{owner}/{repo}/deployments' -f ref="${sha}" -f environment="${environment}"` (raw API) |
| Report its status | `gh api "repos/{owner}/{repo}/deployments/${deployment_id}/statuses" -f state=success -f environment_url="${url}"` (raw API) |

Gaps: `gh` has no environment or deployment command, so every read except releases goes through `gh api`. GitHub has
no rollback primitive; a rollback is a re-run of an older run, or a new deployment of an older commit.

### GitHub schemas

- [GitHub REST API OpenAPI description](https://raw.githubusercontent.com/github/rest-api-description/main/descriptions/api.github.com/api.github.com.json),
  from github/rest-api-description; it holds `/repos/{owner}/{repo}/releases`, `/environments`, `/deployments`,
  `/deployments/{deployment_id}/statuses`, and `/actions/runs/{run_id}/pending_deployments`, and the webhook
  payloads, `deployment` and `deployment_status` among them.
- [GitHub GraphQL schema](https://docs.github.com/public/fpt/schema.docs.graphql): `Environment`,
  `Deployment`, `DeploymentStatus`, and `Release`.
- [GitHub Actions workflow JSON Schema](https://www.schemastore.org/github-workflow.json), SchemaStore's
  "GitHub Workflow" entry, for a job's `environment:`.
- [Release notes configuration JSON Schema](https://www.schemastore.org/github-release-config.json), SchemaStore's
  entry for `.github/release.yml`, which `--generate-notes` reads.
- [Octokit webhook JSON Schemas](https://unpkg.com/@octokit/webhooks-schemas/schema.json), community, covering the
  same events.

## GitLab

`glab` is GitLab's own CLI. It authenticates with `glab auth login`, or `GITLAB_TOKEN`. `--output json` (`-F json`)
prints JSON, `--jq` filters it, and `--page` and `--per-page` page a list. It has release commands and none for
environments or deployments, which go through `glab api`. `glab api` fills `:id` with the project, and reads every
page with `--paginate`. kit reads releases with `glab api` and pages them itself.

A job with `environment:` in `.gitlab-ci.yml` creates a deployment when it runs. A protected environment can require
approvals before its deployment jobs run, on GitLab Premium and Ultimate. GitLab links a deployment to the merge
requests it shipped, and a merge request shows the environments its pipeline deployed to.

### Read from GitLab

| Capability | Command |
| --- | --- |
| List releases | `glab release list --output json --per-page 100 --page "${page}"` |
| A release's notes | `glab release view "${tag}" --output json` |
| List environments | `glab api --paginate 'projects/:id/environments?per_page=100'` (raw API) |
| Deployments in an environment | `glab api --paginate "projects/:id/deployments?environment=${environment}&order_by=finished_at&sort=desc"` (raw API) |
| A deployment's state and approvals | `glab api "projects/:id/deployments/${deployment_id}"` (raw API); `status`, `approvals`, `pending_approval_count` |
| What is live where | `glab api "projects/:id/environments/${environment_id}"` (raw API); `last_deployment` |
| Merge requests a deployment shipped | `glab api "projects/:id/deployments/${deployment_id}/merge_requests"` (raw API) |

### Write to GitLab

| Capability | Command |
| --- | --- |
| Create a release from a tag with notes | `glab release create "${tag}" --notes-file "${file}"`; `--ref "${sha}"` makes the tag when it does not exist |
| Generate notes | `glab changelog generate --version "${version}"` |
| Deploy or promote | `glab ci trigger "${job_id}"`, for a manual deployment job |
| Roll back | `glab ci retry "${job_id}"` on the job of the earlier deployment |
| Stop an environment | `glab api -X POST "projects/:id/environments/${environment_id}/stop"` (raw API) |
| Approve or reject | `glab api -X POST "projects/:id/deployments/${deployment_id}/approval" -f status=approved -f comment="${comment}"` (raw API) |
| Record a deployment made elsewhere | `glab api -X POST 'projects/:id/deployments' -f environment="${environment}" -f sha="${sha}" -f ref="${ref}" -F tag=false -f status=running` (raw API) |
| Report its status | `glab api -X PUT "projects/:id/deployments/${deployment_id}" -f status=success` (raw API) |

Gaps: environments, deployments, and approvals have no `glab` command. GitLab marks no release as a pre-release, so
kit treats a tag with a suffix as one.

### GitLab schemas

- [GitLab REST OpenAPI 3.0 description](https://gitlab.com/gitlab-org/gitlab/-/raw/master/doc/api/openapi/openapi_v3.yaml),
  generated from the API's definitions; it holds `/api/v4/projects/{id}/releases`, `/environments`, and
  `/deployments`. The OpenAPI 2.0 file beside it, `openapi_v2.yaml`, says it is deprecated.
- [GitLab CI JSON Schema](https://gitlab.com/gitlab-org/gitlab/-/raw/master/app/assets/javascripts/editor/schema/ci.json),
  for `environment:` and `release:`. SchemaStore's "gitlab-ci" entry points at the
  [GitLab FOSS copy of the CI schema](https://gitlab.com/gitlab-org/gitlab-foss/-/raw/master/app/assets/javascripts/editor/schema/ci.json).
- [GitLab agent configuration JSON Schema](https://gitlab.com/gitlab-org/cluster-integration/gitlab-agent/-/raw/master/pkg/agentcfg/agentcfg_schemas/gitlab.agent.agentcfg.ConfigurationFile.jsonschema.bundle.json),
  SchemaStore's entry for `.gitlab/agents/*/config.yaml`, the agent that connects a cluster and its Flux to GitLab.
- GraphQL: GitLab publishes no schema file; a GitLab instance answers introspection at `/api/graphql`.

## Argo CD

`argocd` is the project's own CLI. It authenticates with `argocd login "${server}"`, or `ARGOCD_SERVER` and
`ARGOCD_AUTH_TOKEN`; `--core` reads the cluster's Kubernetes API instead of the Argo CD server. `-o json` prints JSON
from `app list`, `app get`, `app sync`, and `app rollback`; `app history` prints text alone, and the same history is
`.status.history` in `app get -o json`. A list is one response, with no pages.

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

## Azure Pipelines

`az` with the `azure-devops` extension is Microsoft's own CLI, the one kit's `ado` tracker uses. It authenticates
with `az login` or `AZURE_DEVOPS_EXT_PAT`. `--output json` prints JSON and `--query` filters it with JMESPath.
`az pipelines release` covers classic releases. YAML environments, their deployment records, and pipeline approvals
have no command, so they go through `az rest` with `--resource 499b84ac-1321-427f-aa17-267ca6975798`, the Azure
DevOps resource, or `az devops invoke`. A list that has more pages returns a continuation token.

Azure Pipelines has two models. A YAML pipeline's deployment job targets an environment, which records each
deployment and holds approvals and checks. A classic release pipeline makes a release, a numbered object unrelated
to a git tag, and deploys it through stages. Microsoft recommends YAML, and an organization setting can disable the
creation of classic release pipelines; no retirement date for classic releases was found.

### Read from Azure Pipelines

| Capability | Command |
| --- | --- |
| List classic releases | `az pipelines release list --definition-id "${definition_id}" --top "${n}" --output json` |
| A classic release and its stages | `az pipelines release show --id "${release_id}" --output json` |
| List environments | `az rest --resource 499b84ac-1321-427f-aa17-267ca6975798 --url "https://dev.azure.com/${org}/${project}/_apis/distributedtask/environments?api-version=7.1"` (raw API) |
| Deployments in an environment | `az rest ... --url "https://dev.azure.com/${org}/${project}/_apis/distributedtask/environments/${environment_id}/environmentdeploymentrecords?api-version=7.1"` (raw API) |
| Classic deployments, what is live where | `az rest ... --url "https://vsrm.dev.azure.com/${org}/${project}/_apis/release/deployments?definitionEnvironmentId=${stage_id}&latestAttemptsOnly=true&api-version=7.1"` (raw API) |
| Pending approvals, YAML | `az rest ... --url "https://dev.azure.com/${org}/${project}/_apis/pipelines/approvals?state=pending&api-version=7.1"` (raw API) |
| Pending approvals, classic | `az rest ... --url "https://vsrm.dev.azure.com/${org}/${project}/_apis/release/approvals?api-version=7.1"` (raw API) |

### Write to Azure Pipelines

| Capability | Command |
| --- | --- |
| Create a classic release | `az pipelines release create --definition-id "${definition_id}" --artifact-metadata-list "${alias}=${version_id}"` |
| Deploy or promote, YAML | `az pipelines run --name "${pipeline}" --branch "${ref}" --parameters "environment=${environment}"`, when the pipeline takes that parameter |
| Deploy or promote, classic | `az rest --method patch ... --url "https://vsrm.dev.azure.com/${org}/${project}/_apis/Release/releases/${release_id}/environments/${stage_id}?api-version=7.1" --body '{"status":"inProgress"}'` (raw API) |
| Roll back | the same deploy on an earlier release or commit |
| Approve, YAML | `az rest --method patch ... --url "https://dev.azure.com/${org}/${project}/_apis/pipelines/approvals?api-version=7.1" --body "[{\"approvalId\":\"${approval_id}\",\"status\":\"approved\"}]"` (raw API) |
| Approve, classic | `az rest --method patch ... --url "https://vsrm.dev.azure.com/${org}/${project}/_apis/release/approvals/${approval_id}?api-version=7.1" --body '{"status":"approved"}'` (raw API) |

Here `...` stands for `--resource 499b84ac-1321-427f-aa17-267ca6975798`. Gaps: no command reads environments,
deployment records, or approvals. Azure Pipelines has no built-in way to write a GitHub deployment; its GitHub app
reports checks.

### Azure Pipelines schemas

- [Azure Pipelines YAML JSON Schema](https://raw.githubusercontent.com/microsoft/azure-pipelines-vscode/master/service-schema.json),
  SchemaStore's "Azure Pipelines" entry.
- Microsoft publishes no OpenAPI document for the Azure DevOps REST API. The Swagger documents per area were in
  MicrosoftDocs/vsts-rest-api-specs, which answers 404 on 2026-11-10, and no other copy was found. The
  [Azure DevOps REST API reference](https://learn.microsoft.com/en-us/rest/api/azure/devops/) is the readable source
  of the routes above.

## AWS CodeDeploy and CodePipeline

`aws` is AWS's own CLI. It authenticates with `aws configure`, `aws sso login`, or the standard environment
variables. It prints JSON by default (`--output json`), `--query` filters with JMESPath, and it reads every page
unless `--max-items` or `--no-paginate` says otherwise. Every command below is real; none needs a raw API route.

CodeDeploy deploys a revision, from S3 or a GitHub commit, to a deployment group: EC2 or on-premises instances, an
ECS service, or a Lambda function. A deployment group is the nearest thing to an environment, and its
`targetRevision` and `lastSuccessfulDeployment` say what is live. CodePipeline runs stages in order, each of which
can deploy; a stage is the nearest thing to an environment, and a manual approval action is its approval. Both
services are current: AWS added regions to each in 2026, and neither has an end-of-support notice.

### Read from AWS CodeDeploy and CodePipeline

| Capability | Command |
| --- | --- |
| List environments (deployment groups) | `aws deploy list-deployment-groups --application-name "${app}"` |
| What is live in a group | `aws deploy get-deployment-group --application-name "${app}" --deployment-group-name "${group}"` |
| Deployments of a group | `aws deploy list-deployments --application-name "${app}" --deployment-group-name "${group}"`, then `aws deploy batch-get-deployments --deployment-ids ${ids}` |
| A deployment's state | `aws deploy get-deployment --deployment-id "${deployment_id}"` |
| List environments (pipeline stages) | `aws codepipeline get-pipeline --name "${pipeline}"` |
| What is live in each stage, and pending approvals | `aws codepipeline get-pipeline-state --name "${pipeline}"`; each action's `currentRevision` and `latestExecution` |
| Deployments (executions) with their source commits | `aws codepipeline list-pipeline-executions --pipeline-name "${pipeline}"`; `sourceRevisions` |

### Write to AWS CodeDeploy and CodePipeline

| Capability | Command |
| --- | --- |
| Deploy a revision | `aws deploy create-deployment --application-name "${app}" --deployment-group-name "${group}" --github-location repository="${owner}/${repo}",commitId="${sha}"` |
| Stop and roll back | `aws deploy stop-deployment --deployment-id "${deployment_id}" --auto-rollback-enabled` |
| Continue a blue/green deployment | `aws deploy continue-deployment --deployment-id "${deployment_id}"` |
| Promote (run the pipeline) | `aws codepipeline start-pipeline-execution --name "${pipeline}"` |
| Let a stage through | `aws codepipeline enable-stage-transition --pipeline-name "${pipeline}" --stage-name "${stage}" --transition-type Inbound` |
| Roll back a stage | `aws codepipeline rollback-stage --pipeline-name "${pipeline}" --stage-name "${stage}" --target-pipeline-execution-id "${execution_id}"` |
| Approve or reject | `aws codepipeline put-approval-result --pipeline-name "${pipeline}" --stage-name "${stage}" --action-name "${action}" --result summary="${text}",status=Approved --token "${token}"` |

Gaps: neither service has releases or writes a deployment to a code forge. CodeDeploy has no approvals of its own.

### AWS CodeDeploy and CodePipeline schemas

- [CodeDeploy Smithy API model](https://raw.githubusercontent.com/aws/api-models-aws/main/models/codedeploy/service/2014-10-06/codedeploy-2014-10-06.json)
  and [CodePipeline Smithy API model](https://raw.githubusercontent.com/aws/api-models-aws/main/models/codepipeline/service/2015-07-09/codepipeline-2015-07-09.json),
  from aws/api-models-aws, the models the SDKs and the CLI are generated from.
- AppSpec, CodeDeploy's `appspec.yml`: AWS publishes no JSON Schema and SchemaStore has no entry; the
  [AppSpec file reference](https://docs.aws.amazon.com/codedeploy/latest/userguide/reference-appspec-file.html) is
  prose.

## Flux

`flux` is the project's own CLI and talks to the cluster through the kubeconfig. `flux get` prints tables alone; it
has no JSON or YAML output. `flux export` prints YAML of the objects' specs, without status. JSON comes from
`kubectl get` on Flux's custom resources, which is the API kit would read.

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

## Octopus Deploy

`octopus` is Octopus Deploy's own CLI, written in Go. It authenticates with `octopus login`, or `OCTOPUS_URL` and
`OCTOPUS_API_KEY`. `--output-format json` prints JSON. `octopus api "${path}"` makes a raw GET request and prints the
JSON response; it sends nothing else, so a write with no command is a plain REST call with the `X-Octopus-ApiKey`
header.

Octopus has its own releases, a numbered snapshot of a project's packages and process, separate from git tags, and
deploys a release to environments in the order a lifecycle sets. Manual intervention steps are its approvals, kept as
interruptions. The dashboard says which release is in each environment.

### Read from Octopus Deploy

| Capability | Command |
| --- | --- |
| List releases | `octopus release list --project "${project}" --output-format json` |
| List environments | `octopus environment list --output-format json` |
| Deployments | `octopus api "/api/${space_id}/deployments?environments=${environment_id}"` (raw API) |
| What is live where | `octopus api "/api/${space_id}/dashboard"` (raw API) |
| Pending approvals | `octopus api "/api/${space_id}/interruptions?pendingOnly=true"` (raw API) |
| Wait on a deployment's task | `octopus task wait "${task_id}"` |

### Write to Octopus Deploy

| Capability | Command |
| --- | --- |
| Create a release | `octopus release create --project "${project}" --version "${version}"` |
| Deploy | `octopus release deploy --project "${project}" --version "${version}" --environment "${environment}"` |
| Promote | `octopus release promote --project "${project}" --from "${environment}" --to "${environment_next}"` |
| Roll back | `octopus release deploy` with the earlier `--version` |
| Approve | `POST /api/${space_id}/interruptions/${interruption_id}/responsible`, then `POST .../submit` with `{"Result":"Proceed"}` (plain REST) |

Gaps: the CLI reads deployments, the dashboard, and interruptions through `octopus api` alone, and cannot write them.
The `pendingOnly` query parameter is from the Swagger document and was not tried against a server.

### Octopus Deploy schemas

- Every Octopus server serves its own Swagger 2.0 document at `/api/swagger.json`; the
  [Octopus samples instance's Swagger document](https://samples.octopus.app/api/swagger.json) is one, version
  2026.4. It holds `/{spaceId}/dashboard`, `/{spaceId}/deployments`, and `/{spaceId}/interruptions`.
- No JSON Schema for its configuration-as-code files was found, and SchemaStore has no entry.

## Harness CD

`harness` is Harness's own CLI ([harness/cli](https://github.com/harness/cli), version 3); an older one,
harness/harness-cli, still exists and takes YAML files. It authenticates with `harness auth login`, or
`HARNESS_API_KEY`, and `harness auth setscope` sets the organization and project. `--format json` prints JSON;
`--limit` and `--all` set how much a list reads. Its commands are a verb and a resource, such as
`harness list environment`.

A Harness pipeline deploys a service to an environment's infrastructure; an execution is a deployment, and an approval
step makes an approval instance. Harness also runs a managed Argo CD for GitOps.

### Read from Harness CD

| Capability | Command |
| --- | --- |
| List environments | `harness list environment --format json --all` |
| Deployments (executions) | `harness list execution "${pipeline_id}" --format json --limit "${n}"` |
| A deployment's state | `harness get execution "${execution_id}" --format json` |
| Pending approvals | `harness list approval_instance "${execution_id}"` |
| GitOps applications | `harness list gitops_application` |

### Write to Harness CD

| Capability | Command |
| --- | --- |
| Deploy or promote | `harness execute pipeline "${pipeline_id}" --input environment="${environment}"`; `--follow` |
| Roll back | `harness execute execution:abort "${execution_id}" --interrupt-type StageRollback` |
| Retry | `harness execute execution:retry "${pipeline_id}/${execution_id}" --only-failed-stages` |
| Approve or reject | `harness execute approval_instance:approve "${approval_instance_id}" --comment "${comment}"`, `approval_instance:reject` |
| Sync a GitOps application | `harness execute gitops_application:sync "${app}"` |

Gaps: no command says which version of a service runs in each environment; the Harness API has instance routes, not
checked here. These commands are from Harness's documentation; `harness` was not run.

### Harness CD schemas

- Harness publishes its API at [the Harness API reference](https://apidocs.harness.io/), which offers an OpenAPI
  download; the download answered 500 when tried, so it is (not verified).
- No JSON Schema for pipeline YAML was found on SchemaStore.

## Spinnaker

`spin` is Spinnaker's own CLI. Its repository, spinnaker/spin, is archived; the CLI lives in the
[spinnaker/spinnaker monorepo's `spin/`](https://github.com/spinnaker/spinnaker/tree/main/spin). It talks to Gate,
Spinnaker's API server, set with `--gate-endpoint` or `~/.spin/config`, which also holds the authentication. It
prints JSON by default; `-o yaml` and `-o jsonpath=${path}` are the others.

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

## Argo Rollouts

`kubectl argo rollouts` is the project's own kubectl plugin and talks to the cluster through the kubeconfig. Its
`get`, `list`, and `status` print text alone; JSON comes from `kubectl get rollouts.argoproj.io -o json`.

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

## Kargo

`kargo` is the project's own CLI. It authenticates with `kargo login "${server}"` and one of `--sso`, `--admin`, or
`--kubeconfig`. `-o json` prints JSON, as kubectl does. Kargo keeps its state in custom resources, so
`kubectl get -o json` reads the same objects.

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

## Kubernetes CRDs in general

Argo CD, Flux, Argo Rollouts, and Kargo keep their state in Kubernetes custom resources. Every CRD carries an
OpenAPI v3 schema under `spec.versions[].schema.openAPIV3Schema`, and a cluster serves the schemas of what it has
installed at `/openapi/v3`, which `kubectl get --raw /openapi/v3` reads and `kubectl explain` renders. The
[Kubernetes OpenAPI document](https://raw.githubusercontent.com/kubernetes/kubernetes/master/api/openapi-spec/swagger.json)
and the [Kubernetes apiextensions OpenAPI v3 document](https://raw.githubusercontent.com/kubernetes/kubernetes/master/api/openapi-spec/v3/apis__apiextensions.k8s.io__v1_openapi.json)
describe the API that serves them and the CRD object itself. The
[CRDs catalog](https://github.com/datreeio/CRDs-catalog) converts many CRDs to JSON Schema.

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
