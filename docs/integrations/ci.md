# Continuous integration

Continuous integration (CI) is the system that runs a repository's pipeline on each push and each change and reports
the result. To kit it is two things. It is a source of data: the builds stage of `kit flow` lists the runs no change
owns, and the changes stage shows each change's checks. It is also where `kit ci` runs: the delivered GitHub Actions
job runs it on every pull request into the default branch, and any other pipeline can run it too. The index of these
pages is [1st.md](1st.md).

kit reads CI through the code forge seam, `kit/modules/bash/forge.sh`, and two of its calls carry CI data; what a
seam is, [architecture.md](../architecture.md#code-forges-seams-and-providers) says.

- `forge:runs` returns the 20 newest runs that no change owns, each with its jobs and the failed step. The builds
  stage draws them.
- `open-prs` returns each open change with its checks: on GitHub the commit's `statusCheckRollup`, both check runs
  and commit statuses; on GitLab the jobs of the merge request's head pipeline. The changes stage draws them.

`forge:marks` also keeps a runs mark, a text that changes when the runs do, so a picture reloads only when it must.

see: [forges.md](forges.md), `kit/modules/bash/builds.sh`

## CI that is not the code forge's own

The seam reads the code forge's own CI: GitHub Actions on GitHub, GitLab CI/CD on GitLab. A repository on GitHub can
run its pipeline elsewhere, on Jenkins, CircleCI, Buildkite, or Azure Pipelines. Two cases follow.

- A change's checks still work. External CI reports into the code forge: it posts a commit status or, as a GitHub
  App, a check run on the commit it built. `open-prs` reads both, so the changes stage shows an external system's
  checks beside the code forge's own with no new code.
- The builds stage does not. `forge:runs` lists the code forge's own runs, and a status on a commit of the default
  branch is not a run. To list an external system's runs, their jobs, and their logs, kit needs a seam of its own,
  a `ci.sh` beside `forge.sh` whose providers answer `runs` in the shape `forge:runs` has today: `{id, number,
  attempt, workflow, branch, tag, event, sha, title, state, age, took, url, jobs}`, each job `{name, state, took,
  step}`. Its enabling point would be a `ci` setting in `kit.yml`, else the code forge's own.

`kit ci` itself runs on any CI system, but `kit linked` needs the change's number. `forge:pr-here` reads it from
GitHub's event payload or GitLab's `CI_MERGE_REQUEST_IID`, else asks for the open change on the checked-out branch.
Most external systems check out a detached HEAD, so a provider would read the system's own variable:
`BUILDKITE_PULL_REQUEST`, `CIRCLE_PULL_REQUEST`, `SYSTEM_PULLREQUEST_PULLREQUESTNUMBER` (Azure Pipelines),
`CODEBUILD_WEBHOOK_TRIGGER`, `CHANGE_ID` (Jenkins multibranch), `BITBUCKET_PR_ID`, `CI_COMMIT_PULL_REQUEST`
(Woodpecker), or `DRONE_PULL_REQUEST`.

## Scope

| Capability | Read or write | Where kit uses it |
| --- | --- | --- |
| Recent runs no change owns: pushes to the default branch or a tag, scheduled, and manual runs | read | `forge:runs`, the builds stage |
| A run's jobs and the step that failed | read | `forge:runs` (`jobs[].step`), the builds stage |
| Whether the runs changed, read cheaply | read | `forge:marks`, the runs mark |
| A change's checks | read | `open-prs`, the changes stage; kittui's checks tab (planned) |
| A run's or a job's log | read | planned |
| A run's artifacts: list | read | `forge:artifacts`, `kit delivery artifacts`: the newest, with digest, size, run, and expiry |
| A run's artifacts: download | read | planned |
| Rerun a run, or its failed jobs | write | planned |
| Cancel a run | write | planned |
| Start a run: a manual run of a workflow on a ref | write | planned |
| Report a status on a commit | write | planned; `kit ci` reports through its job's own result today |
| The pipeline file that runs `kit ci` | write | the `github/ci` package, kept by `kit check` |

### The pipeline file

kit delivers `kit ci` as a file in the repository. The `github/ci` package delivers
`kit/packages/github/ci/templates/repo/.github/workflows/kit-ci.yml` as `.github/workflows/kit-ci.yml`, and `kit
check` fails when it differs from the release. A system that reads a directory of pipeline files can take kit's file
beside the repository's own. A system that reads one root file needs the repository's file to include kit's.

| Provider | Pipeline file | kit's job would live |
| --- | --- | --- |
| GitHub Actions | `.github/workflows/*.yml` | `.github/workflows/kit-ci.yml`, delivered today |
| GitLab CI/CD | `.gitlab-ci.yml`, one root file | a file of kit's that the root file names under `include: local:` |
| Jenkins | `Jenkinsfile`, path set per job | a stage the repository's `Jenkinsfile` calls; no separate file |
| Azure Pipelines | `azure-pipelines.yml` by default; each pipeline names its own YAML path | a file of kit's, with a pipeline of its own or as a `template:` the root file uses |
| CircleCI | `.circleci/config.yml`, one root file | a job in that file |
| AWS CodeBuild | `buildspec.yml` by default; each project names its own path | a buildspec of kit's, with a project of its own |
| Bitbucket Pipelines | `bitbucket-pipelines.yml`, one root file | a step in that file |
| Buildkite | `.buildkite/pipeline.yml`, uploaded by a step | a file of kit's that a step uploads |
| Tekton (Pipelines-as-Code) | `.tekton/*.yaml` | `.tekton/kit-ci.yaml` |
| Argo Workflows | manifests applied to a cluster | a `WorkflowTemplate`; no file in the repository is read on its own |
| Forgejo and Gitea Actions | `.forgejo/workflows/` and `.gitea/workflows/`; Gitea falls back to `.github/workflows/` | `.forgejo/workflows/kit-ci.yml` or `.gitea/workflows/kit-ci.yml`; whether the GitHub file runs there as it is, its actions resolved from the instance's actions source, was not verified |
| Woodpecker CI | `.woodpecker.yml` or `.woodpecker/*.yml` | `.woodpecker/kit-ci.yml` |
| Drone | `.drone.yml`, path set per repository | a pipeline in that file |
| Concourse | any file, set with `fly set-pipeline` | a pipeline of kit's, set from its file |

## Providers

| Provider | Hosting | CLI | kit status | JSON from the CLI | Machine-readable descriptions |
| --- | --- | --- | --- | --- | --- |
| [GitHub Actions](gh.md#github-actions) | SaaS; GitHub Enterprise Server self-hosted; proprietary | `gh`, official, [cli/cli](https://github.com/cli/cli) | `provider`, through `forge/github.sh` | `--json "${fields}"`, `--jq`; `gh api` | REST OpenAPI, GraphQL schema, workflow JSON Schema, webhook JSON Schema |
| [GitLab CI/CD](glab.md#gitlab-cicd) | SaaS; self-managed; open core, the community edition MIT | `glab`, official, [gitlab-org/cli](https://gitlab.com/gitlab-org/cli) | `provider`, through `forge/gitlab.sh` | `-F json`, `--jq`; `glab api` | REST OpenAPI, CI JSON Schema |
| [Jenkins](jenkins.md#jenkins) | self-hosted; open source, MIT | `jenkins-cli.jar`, official, shipped by each controller | `none` | none; REST `/api/json` | none |
| [Azure Pipelines](az.md#azure-pipelines-runs) | SaaS; Azure DevOps Server self-hosted; proprietary | `az` with the `azure-devops` extension, official, [Azure/azure-devops-cli-extension](https://github.com/Azure/azure-devops-cli-extension) | `provider`, through `forge/ado.sh` | `-o json` (the default), `--query` JMESPath; `az devops invoke` | pipeline YAML JSON Schema; REST: none published; a REST reference in prose |
| [CircleCI](circleci.md#circleci) | SaaS; CircleCI server self-hosted; proprietary | `circleci`, official, [CircleCI-Public/circleci-cli](https://github.com/CircleCI-Public/circleci-cli) | `none` | `--json`, `--jq`; `circleci api` | API v2 OpenAPI, config JSON Schema |
| [AWS CodeBuild and CodePipeline](aws.md#aws-codebuild-and-codepipeline) | SaaS; proprietary | `aws`, official, [aws/aws-cli](https://github.com/aws/aws-cli) | `planned`, with an `aws` code forge | `--output json`, `--query` JMESPath | service models (JSON); no buildspec schema |
| [Bitbucket Pipelines](bkt.md#bitbucket-pipelines) | SaaS; proprietary | Atlassian's Teamwork Graph CLI, `twg`, official; status unclear | `none` | not verified | REST OpenAPI, pipeline JSON Schema |
| [Buildkite](bk.md#buildkite) | SaaS control plane, self-hosted agents | `bk`, official, [buildkite/cli](https://github.com/buildkite/cli) | `none` | `-o json`; `bk api` | pipeline JSON Schema, GraphQL schema |
| [Tekton](tkn.md#tekton) | self-hosted on Kubernetes; open source, Apache-2.0 | `tkn`, official, [tektoncd/cli](https://github.com/tektoncd/cli) | `none` | `-o json` | custom resource definitions, Swagger, Tekton Results OpenAPI |
| [Argo Workflows](argo.md#argo-workflows) | self-hosted on Kubernetes; open source, Apache-2.0 | `argo`, official, [argoproj/argo-workflows](https://github.com/argoproj/argo-workflows) | `none` | `-o json` | JSON Schema, Swagger |
| [Forgejo and Gitea Actions](tea.md#forgejo-and-gitea-actions) | self-hosted, or a hosted instance such as Codeberg; Forgejo GPL-3.0, Gitea MIT | `tea`, official to Gitea, [gitea/tea](https://gitea.com/gitea/tea); `fj`, community, [forgejo-contrib/forgejo-cli](https://codeberg.org/forgejo-contrib/forgejo-cli) | `none` | `tea -o json`; `fj` none | REST Swagger; the GitHub workflow JSON Schema |
| [Woodpecker CI](woodpecker-cli.md#woodpecker-ci) | self-hosted; open source, Apache-2.0 | `woodpecker-cli`, official, [woodpecker-ci/woodpecker](https://github.com/woodpecker-ci/woodpecker) | `none` | none; `--output go-template=` | pipeline JSON Schema, Swagger served by the server |
| [Drone](drone.md#drone) | self-hosted; community edition Apache-2.0 | `drone`, official, [harness/drone-cli](https://github.com/harness/drone-cli) | `none` | `--format '{{ jsonify . }}'` | pipeline JSON Schema (SchemaStore) |
| [Concourse](fly.md#concourse) | self-hosted; open source, Apache-2.0 | `fly`, official, [concourse/concourse](https://github.com/concourse/concourse) | `none` | `--json` on list commands; `fly curl` | none |

The table orders providers by how widely they are used, as far as kit can tell; there is no single count to cite.

### Who reports to the code forge

| Provider | What it posts on the commit |
| --- | --- |
| GitHub Actions, GitLab CI/CD, Forgejo and Gitea Actions, Bitbucket Pipelines | nothing to post; each is its code forge's own CI |
| Azure Pipelines | check runs on GitHub, through its GitHub App; on Azure Repos, its own status |
| CircleCI | commit statuses per job; check runs per workflow on GitHub, through its GitHub App |
| Buildkite | commit statuses on GitHub, per build or per job |
| AWS CodeBuild | commit statuses on GitHub, GitLab, and Bitbucket, when the project reports build status |
| AWS CodePipeline | none found; its CodeBuild actions report as CodeBuild does |
| Jenkins | through plugins: commit statuses (GitHub Branch Source), check runs (GitHub Checks) |
| Woodpecker CI, Drone | commit statuses |
| Tekton | through Pipelines-as-Code: check runs on GitHub, statuses on GitLab, Gitea, and Bitbucket |
| Argo Workflows, Concourse | nothing built in; a step or a resource posts it |

Each provider's name leads to its section on its tool's page: what the tool reads and writes
where it meets kit's scope, its gaps, and the descriptions the provider publishes.

## Coverage

`cli` is a command of the provider's CLI, `api` a route reached only through the raw API, and `—` not available.
Bitbucket's `cli` cells are the Teamwork Graph CLI, whose flags were not verified. Reporting a status on the commit
is the code forge's call, not the CI system's; see "Who reports to the code forge".

| Capability | GitHub Actions | GitLab CI/CD | Jenkins | Azure Pipelines | CircleCI | CodeBuild | CodePipeline | Bitbucket | Buildkite | Tekton | Argo | Gitea | Forgejo | Woodpecker | Drone | Concourse |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| Recent runs | cli | cli | api | cli | cli | cli | cli | cli | cli | cli | cli | cli | api | cli | cli | cli |
| Runs no change owns, filtered by the server | cli | cli | api | cli | — | — | — | api | — | — | — | cli | api | cli | — | — |
| A run's jobs and failed step | cli | cli | api | api | cli | cli | cli | api | cli | cli | cli | cli | api | cli | cli | api |
| A run's log | cli | cli | cli | api | cli | cli | — | cli | cli | cli | cli | cli | api | cli | cli | cli |
| Artifacts | cli | cli | api | cli | cli | cli | cli | — | cli | — | cli | api | api | — | — | — |
| Rerun | cli | cli | cli | api | cli | cli | cli | cli | cli | cli | cli | api | api | cli | cli | cli |
| Cancel | cli | cli | cli | cli | cli | cli | cli | api | cli | cli | cli | cli | api | cli | cli | cli |
| Start a run | cli | cli | cli | cli | cli | cli | cli | cli | cli | cli | cli | cli | cli | cli | cli | cli |
