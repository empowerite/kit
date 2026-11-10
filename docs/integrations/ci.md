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
| A run's artifacts: list and download | read | planned, kittui's artifacts tab |
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
| GitHub Actions | SaaS; GitHub Enterprise Server self-hosted; proprietary | `gh`, official, [cli/cli](https://github.com/cli/cli) | `provider`, through `forge/github.sh` | `--json "${fields}"`, `--jq`; `gh api` | REST OpenAPI, GraphQL schema, workflow JSON Schema, webhook JSON Schema |
| GitLab CI/CD | SaaS; self-managed; open core, the community edition MIT | `glab`, official, [gitlab-org/cli](https://gitlab.com/gitlab-org/cli) | `provider`, through `forge/gitlab.sh` | `-F json`, `--jq`; `glab api` | REST OpenAPI, CI JSON Schema |
| Jenkins | self-hosted; open source, MIT | `jenkins-cli.jar`, official, shipped by each controller | `none` | none; REST `/api/json` | none |
| Azure Pipelines | SaaS; Azure DevOps Server self-hosted; proprietary | `az` with the `azure-devops` extension, official, [Azure/azure-devops-cli-extension](https://github.com/Azure/azure-devops-cli-extension) | `planned`, with an `ado` code forge | `-o json` (the default), `--query` JMESPath; `az devops invoke` | pipeline YAML JSON Schema; REST: none published; a REST reference in prose |
| CircleCI | SaaS; CircleCI server self-hosted; proprietary | `circleci`, official, [CircleCI-Public/circleci-cli](https://github.com/CircleCI-Public/circleci-cli) | `none` | `--json`, `--jq`; `circleci api` | API v2 OpenAPI, config JSON Schema |
| AWS CodeBuild and CodePipeline | SaaS; proprietary | `aws`, official, [aws/aws-cli](https://github.com/aws/aws-cli) | `planned`, with an `aws` code forge | `--output json`, `--query` JMESPath | service models (JSON); no buildspec schema |
| Bitbucket Pipelines | SaaS; proprietary | Atlassian's Teamwork Graph CLI, `twg`, official; status unclear | `none` | not verified | REST OpenAPI, pipeline JSON Schema |
| Buildkite | SaaS control plane, self-hosted agents | `bk`, official, [buildkite/cli](https://github.com/buildkite/cli) | `none` | `-o json`; `bk api` | pipeline JSON Schema, GraphQL schema |
| Tekton | self-hosted on Kubernetes; open source, Apache-2.0 | `tkn`, official, [tektoncd/cli](https://github.com/tektoncd/cli) | `none` | `-o json` | custom resource definitions, Swagger, Tekton Results OpenAPI |
| Argo Workflows | self-hosted on Kubernetes; open source, Apache-2.0 | `argo`, official, [argoproj/argo-workflows](https://github.com/argoproj/argo-workflows) | `none` | `-o json` | JSON Schema, Swagger |
| Forgejo and Gitea Actions | self-hosted, or a hosted instance such as Codeberg; Forgejo GPL-3.0, Gitea MIT | `tea`, official to Gitea, [gitea/tea](https://gitea.com/gitea/tea); `fj`, community, [forgejo-contrib/forgejo-cli](https://codeberg.org/forgejo-contrib/forgejo-cli) | `none` | `tea -o json`; `fj` none | REST Swagger; the GitHub workflow JSON Schema |
| Woodpecker CI | self-hosted; open source, Apache-2.0 | `woodpecker-cli`, official, [woodpecker-ci/woodpecker](https://github.com/woodpecker-ci/woodpecker) | `none` | none; `--output go-template=` | pipeline JSON Schema, Swagger served by the server |
| Drone | self-hosted; community edition Apache-2.0 | `drone`, official, [harness/drone-cli](https://github.com/harness/drone-cli) | `none` | `--format '{{ jsonify . }}'` | pipeline JSON Schema (SchemaStore) |
| Concourse | self-hosted; open source, Apache-2.0 | `fly`, official, [concourse/concourse](https://github.com/concourse/concourse) | `none` | `--json` on list commands; `fly curl` | none |

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

## GitHub Actions

`gh` is GitHub's CLI. It signs in with `gh auth login` or reads `GH_TOKEN`. Every list command takes `--json
<fields>`, with `--jq` and `--template` to shape it, and `--limit` instead of pages. `gh api` calls any REST route,
fills `{owner}` and `{repo}` from the checkout, and reads every page with `--paginate`; with `--slurp` it joins them.

### Read from GitHub Actions

| Capability | Command |
| --- | --- |
| Recent runs | `gh run list --limit 50 --json databaseId,number,attempt,workflowName,headBranch,headSha,displayTitle,event,status,conclusion,createdAt,startedAt,updatedAt,url`; filter with `--branch "${branch}"`, `--event push`, `--event schedule`, `--event workflow_dispatch` |
| A run's jobs and failed step | `gh run view "${run_id}" --json jobs`; each job has `steps[]` with `conclusion` |
| Whether runs changed | `gh api --include -H "If-None-Match: ${etag}" 'repos/{owner}/{repo}/actions/runs?per_page=10'`, as kit's runs mark does |
| A change's checks | `gh pr checks "${n}" --json name,state,bucket,link,workflow`; kit reads `statusCheckRollup` through `gh api graphql` |
| A run's log | `gh run view "${run_id}" --log-failed`, or `--log`; one job with `--job "${job_id}"` |
| Artifacts | list: `gh api repos/{owner}/{repo}/actions/runs/${run_id}/artifacts` (no `gh` command lists them); download: `gh run download "${run_id}" --name "${name}"` |

### Write to GitHub Actions

| Capability | Command |
| --- | --- |
| Rerun | `gh run rerun "${run_id}"`; `--failed` for the failed jobs and what they need; `--job "${job_id}"` for one |
| Cancel | `gh run cancel "${run_id}"`; `--force` |
| Start a run | `gh workflow run "${workflow}" --ref "${ref}" -f "${key}=${value}"`; the workflow needs a `workflow_dispatch` trigger |
| Report a status | `gh api repos/{owner}/{repo}/statuses/${sha} -f state=success -f context=kit -f target_url="${url}"` |

A check run, `POST repos/{owner}/{repo}/check-runs`, takes a GitHub App's token, so a person's token or `gh` alone
posts only a commit status. `gh` has no command that validates a workflow file.

### GitHub Actions schemas

- [GitHub workflow JSON Schema](https://www.schemastore.org/github-workflow.json), SchemaStore's "GitHub Workflow"
  entry, which also matches `.gitea/workflows/` and `.forgejo/workflows/`
- [GitHub action metadata JSON Schema](https://www.schemastore.org/github-action.json), for `action.yml`
- [GitHub REST OpenAPI](https://raw.githubusercontent.com/github/rest-api-description/main/descriptions/api.github.com/api.github.com.json),
  with the `actions/runs`, `actions/jobs`, `check-runs`, and `statuses` routes, and its
  [YAML form](https://raw.githubusercontent.com/github/rest-api-description/main/descriptions/api.github.com/api.github.com.yaml)
- [GitHub GraphQL schema](https://docs.github.com/public/fpt/schema.docs.graphql), with `CheckRun`, `StatusContext`,
  and `WorkflowRun`
- [Octokit webhook JSON Schema](https://unpkg.com/@octokit/webhooks-schemas/schema.json), for `workflow_run`,
  `check_run`, and `status` payloads

## GitLab CI/CD

`glab` is GitLab's CLI. It signs in with `glab auth login` or reads `GITLAB_TOKEN`. `glab ci list`, `glab ci get`, and
`glab ci status` take `-F json` and `--jq`; lists page with `-p` and `-P`. `glab api` calls any REST route, fills
`:id` with the project, and reads every page with `--paginate`. A GitLab job has no steps, so the failed step is the
failed job and its `failure_reason`.

### Read from GitLab CI/CD

| Capability | Command |
| --- | --- |
| Recent runs | `glab ci list -F json -P 50`; filter with `--source push`, `--source schedule`, `--source web`, `--scope tags`, `-r "${branch}"`; kit reads `glab api 'projects/:id/pipelines?per_page=50'` |
| A run's jobs | `glab ci get -p "${pipeline_id}" -d -F json`, or `glab api projects/:id/pipelines/${pipeline_id}/jobs` |
| A change's checks | `glab ci get --merge-request "${n}" -F json`; kit reads the merge request's head pipeline and its jobs |
| A job's log | `glab ci trace "${job_id}"`, which follows a running job; `glab api projects/:id/jobs/${job_id}/trace` for the raw text |
| Artifacts | `glab job artifact "${ref}" "${job_name}"` downloads the latest; `--list-paths` lists them; `glab api projects/:id/jobs/${job_id}/artifacts` for one job |

### Write to GitLab CI/CD

| Capability | Command |
| --- | --- |
| Rerun | a job: `glab ci retry "${job_id}"`; a pipeline's failed jobs: `glab api -X POST projects/:id/pipelines/${pipeline_id}/retry` |
| Cancel | `glab ci cancel pipeline "${pipeline_id}"`; `glab ci cancel job "${job_id}"` |
| Start a run | `glab ci run -b "${branch}"`, with `--variables` or `--input`; a manual job: `glab ci trigger "${job_id}"` |
| Report a status | `glab api -X POST projects/:id/statuses/${sha} -f state=success -f name=kit -f target_url="${url}"` |

`glab ci lint` validates `.gitlab-ci.yml` on the server, and `glab ci config compile` prints it with its includes
merged. Retrying a whole pipeline has no `glab` command.

### GitLab CI/CD schemas

- [GitLab CI JSON Schema](https://gitlab.com/gitlab-org/gitlab/-/raw/master/app/assets/javascripts/editor/schema/ci.json),
  for `.gitlab-ci.yml`; SchemaStore's "gitlab-ci" entry points at the
  [same file in gitlab-foss](https://gitlab.com/gitlab-org/gitlab-foss/-/raw/master/app/assets/javascripts/editor/schema/ci.json)
- [GitLab REST OpenAPI 3](https://gitlab.com/gitlab-org/gitlab/-/raw/master/doc/api/openapi/openapi_v3.yaml), with the
  pipelines, jobs, and commit statuses routes, and the
  [Swagger 2 form](https://gitlab.com/gitlab-org/gitlab/-/raw/master/doc/api/openapi/openapi_v2.yaml)
- The GraphQL schema is read by introspection at `/api/graphql`; no static file was verified.

## Jenkins

Jenkins ships its CLI as `jenkins-cli.jar`, run as `java -jar jenkins-cli.jar -s "${jenkins_url}" -auth
"${user}:${token}" "${command}"`, or with `JENKINS_USER_ID` and `JENKINS_API_TOKEN`. It prints text. The REST API is
where JSON comes from: append `/api/json` to any page, and trim it with `tree=`. There is no paging. Stages come from
the Pipeline Stage View plugin's `wfapi` routes. A multibranch job keeps each branch and each pull request as a job of
its own (`main`, `PR-7`), which is how a run no change owns is told apart.

### Read from Jenkins

| Capability | Command |
| --- | --- |
| Recent runs | `GET /job/${job}/api/json?tree=builds[number,result,timestamp,duration,url]` (REST; `list-jobs` names jobs only) |
| A run's stages and failed step | `GET /job/${job}/${build}/wfapi/describe`, then `GET /job/${job}/${build}/execution/node/${node}/wfapi/describe` (REST) |
| A run's log | `java -jar jenkins-cli.jar console "${job}" "${build}"`; REST `GET /job/${job}/${build}/consoleText` |
| Artifacts | `artifacts[]` in `GET /job/${job}/${build}/api/json`; download `GET /job/${job}/${build}/artifact/${path}` (REST) |

### Write to Jenkins

| Capability | Command |
| --- | --- |
| Rerun | `java -jar jenkins-cli.jar replay-pipeline "${job}" -n "${build}"` |
| Cancel | `java -jar jenkins-cli.jar stop-builds "${job}"` stops every running build of the job; one build: `POST /job/${job}/${build}/stop` (REST) |
| Start a run | `java -jar jenkins-cli.jar build "${job}" -p "${key}=${value}" -s`; REST `POST /job/${job}/buildWithParameters` |
| Report a status | plugins post it: GitHub Branch Source a commit status, GitHub Checks a check run |

Jenkins lists no runs in JSON from its CLI, and the `wfapi` routes belong to a plugin. Blue Ocean is deprecated, and
Pipeline Graph View replaces it.

### Jenkins schemas

Jenkins publishes none. A `Jenkinsfile` is Groovy, and the REST API describes itself page by page at `/api`.

## Azure Pipelines

The `azure-devops` extension to `az` covers Azure Pipelines; it installs itself on first use. It signs in with `az
login` or `AZURE_DEVOPS_EXT_PAT`, as the `ado` tracker does, and takes `--organization` and `--project` or their
defaults from `az devops configure`. Output is JSON by default; `--query` filters it with JMESPath, and `--top` limits
a list. `az devops invoke` calls any REST route with the extension's sign-in.

### Read from Azure Pipelines

| Capability | Command |
| --- | --- |
| Recent runs | `az pipelines runs list --branch "${branch}" --top 20 --query-order QueueTimeDesc --reason individualCI`; `--reason` takes one value per call, `schedule` or `manual` the others |
| A run's jobs and failed step | REST `GET https://dev.azure.com/${organization}/${project}/_apis/build/builds/${build_id}/timeline?api-version=7.1`: records of type `Stage`, `Job`, and `Task`, each with `result` |
| A change's checks | `az pipelines runs list --reason pullRequest --branch "refs/pull/${n}/merge"` on GitHub; an Azure Repos pull request's policies belong to the code forge |
| A run's log | REST `GET https://dev.azure.com/${organization}/${project}/_apis/build/builds/${build_id}/logs/${log_id}?api-version=7.1`; the timeline names each record's `log.id` |
| Artifacts | `az pipelines runs artifact list --run-id "${run_id}"`; `az pipelines runs artifact download --run-id "${run_id}" --artifact-name "${name}" --path "${dir}"` |

### Write to Azure Pipelines

| Capability | Command |
| --- | --- |
| Rerun | REST `PATCH https://dev.azure.com/${organization}/${project}/_apis/build/builds/${build_id}?retry=true&api-version=7.1` |
| Cancel | `az pipelines build cancel --build-id "${build_id}"` |
| Start a run | `az pipelines run --name "${pipeline}" --branch "${branch}"`, with `--parameters` or `--variables` |
| Report a status | on Azure Repos, REST `POST https://dev.azure.com/${organization}/${project}/_apis/git/repositories/${repository}/commits/${sha}/statuses?api-version=7.1` |

The CLI has no command for a run's jobs, its log, or a retry. The REST routes above go through `az devops invoke
--area build --resource timeline --route-parameters project="${project}" buildId="${build_id}"`; the area and
resource names were not checked against a live organization.

### Azure Pipelines schemas

- [Azure Pipelines YAML JSON Schema](https://raw.githubusercontent.com/microsoft/azure-pipelines-vscode/main/service-schema.json),
  SchemaStore's "Azure Pipelines" entry
- Microsoft publishes no OpenAPI document for the Azure DevOps REST API. The Swagger documents per area were in
  MicrosoftDocs/vsts-rest-api-specs, which answers 404 on 2026-11-10, and no other copy was found.
  The REST reference on Microsoft Learn describes the routes in prose.

## CircleCI

`circleci` is CircleCI's CLI. It signs in with `circleci auth login` or reads `CIRCLE_TOKEN`, and `CIRCLE_HOST` points
it at a CircleCI server. Its data commands take `--json` and `--jq`, and `circleci api` calls any route of the API it
is built on, which CircleCI calls v3. In this CLI a run is one pipeline's execution, with workflows, and jobs in them.
The CLI has no filter for runs no change owns; `--branch` comes closest.

### Read from CircleCI

| Capability | Command |
| --- | --- |
| Recent runs | `circleci run list --project "${project_slug}" --branch "${branch}" --limit 20 --json` |
| A run's jobs and failed step | `circleci run get "${run_id}" --json`; `circleci job get "${job_id}" --json` gives each step's status and exit code; `--failure-report` prints the failed steps |
| A job's log | `circleci job output get "${job_id}" --step-num "${step}"` |
| Artifacts | `circleci job artifact "${job_id}" --json` lists; `-o "${dir}"` downloads |

### Write to CircleCI

| Capability | Command |
| --- | --- |
| Rerun | `circleci workflow rerun "${workflow_id}" --from-failed` |
| Cancel | `circleci run cancel "${run_id}"`; `circleci workflow cancel "${workflow_id}"` |
| Start a run | `circleci run trigger --branch "${branch}" --parameter "${key}=${value}"` |
| Report a status | CircleCI posts it; see the table above |

The published OpenAPI describes API v2, not the v3 routes the CLI calls; no v3 description was found. `circleci
config validate` checks `.circleci/config.yml`.

### CircleCI schemas

- [CircleCI API v2 OpenAPI](https://circleci.com/api/v2/openapi.json)
- [CircleCI config JSON Schema](https://raw.githubusercontent.com/CircleCI-Public/circleci-yaml-language-server/main/schema.json),
  from CircleCI's language server; SchemaStore keeps
  [its own "CircleCI config.yml" entry](https://www.schemastore.org/circleciconfig.json)

## AWS CodeBuild and CodePipeline

The `aws` CLI covers both. It signs in with an AWS profile, single sign-on, or environment variables, prints JSON with
`--output json`, filters with `--query` (JMESPath), and reads every page unless `--no-paginate` or `--max-items` says
otherwise. CodeBuild runs one build of one project, in phases; CodePipeline runs stages of actions, and a CodeBuild
action's build is a CodeBuild build. Neither filters runs by what started them on the server: a build's
`sourceVersion` is `pr/${n}` for a pull request, and an execution's `trigger.triggerType` says how it began, so kit
would filter as it does today.

### Read from AWS CodeBuild and CodePipeline

| Capability | Command |
| --- | --- |
| Recent runs | `aws codebuild list-builds-for-project --project-name "${project}" --sort-order DESCENDING` (identifiers only); `aws codepipeline list-pipeline-executions --pipeline-name "${pipeline}"` |
| A run's phases and the failed one | `aws codebuild batch-get-builds --ids "${build_id}"`: `phases[]` with `phaseStatus`; `aws codepipeline list-action-executions --pipeline-name "${pipeline}" --filter pipelineExecutionId="${execution_id}"` |
| A run's log | `aws logs tail "${log_group}" --log-stream-names "${log_stream}"`, the group and stream from the build's `logs`; CodePipeline keeps none of its own |
| Artifacts | the build's `artifacts.location` in `batch-get-builds`, an S3 location, then `aws s3 cp`; an action's `output.outputArtifacts[].s3location` |

### Write to AWS CodeBuild and CodePipeline

| Capability | Command |
| --- | --- |
| Rerun | `aws codebuild retry-build --id "${build_id}"`; `aws codepipeline retry-stage-execution --pipeline-name "${pipeline}" --stage-name "${stage}" --pipeline-execution-id "${execution_id}" --retry-mode FAILED_ACTIONS` |
| Cancel | `aws codebuild stop-build --id "${build_id}"`; `aws codepipeline stop-pipeline-execution --pipeline-name "${pipeline}" --pipeline-execution-id "${execution_id}"` |
| Start a run | `aws codebuild start-build --project-name "${project}" --source-version "${ref}"`; `aws codepipeline start-pipeline-execution --name "${pipeline}"` |
| Report a status | CodeBuild posts it when the project reports build status; see the table above |

AWS CodeCommit, the code forge an `aws` provider would read, is open to new customers: AWS closed it to them in July
2024 and made it generally available again on 24 November 2025. No retirement of CodeBuild or CodePipeline was found.

### AWS CodeBuild and CodePipeline schemas

- [CodeBuild service model](https://raw.githubusercontent.com/boto/botocore/develop/botocore/data/codebuild/2016-10-06/service-2.json)
  and [CodePipeline service model](https://raw.githubusercontent.com/boto/botocore/develop/botocore/data/codepipeline/2015-07-09/service-2.json),
  the JSON the CLI is generated from
- The same APIs as Smithy models: [CodeBuild](https://raw.githubusercontent.com/aws/api-models-aws/main/models/codebuild/service/2016-10-06/codebuild-2016-10-06.json)
  and [CodePipeline](https://raw.githubusercontent.com/aws/api-models-aws/main/models/codepipeline/service/2015-07-09/codepipeline-2015-07-09.json)
- AWS publishes no JSON Schema for `buildspec.yml`, and SchemaStore has none.

## Bitbucket Pipelines

Bitbucket Pipelines is Bitbucket Cloud's own CI; Bitbucket Data Center has none and points to Bamboo. Its REST API
lives under `https://api.bitbucket.org/2.0/repositories/${workspace}/${repo}`, signs in with an API token (app
passwords are retired), and pages with `page` and `pagelen`. A run's `trigger_type` is `PUSH`, `MANUAL`, `SCHEDULED`,
or `PARENT_STEP`, and its `target.selector.type` tells a pull request from a branch or a tag. Atlassian's `acli` has
no Bitbucket commands. Its Teamwork Graph CLI, `twg`, lists `twg bitbucket pipeline query`, `get`, `run`, `trigger`,
`rerun-failed`, `tail`, `grep`, and `latest-failure`; their flags, their JSON, and whether the CLI is past beta were
not verified, so the tables give REST.

### Read from Bitbucket Pipelines

| Capability | Command |
| --- | --- |
| Recent runs | `GET /pipelines?sort=-created_on&trigger_type=PUSH` (REST); `twg bitbucket pipeline query` |
| A run's steps | `GET /pipelines/${pipeline_uuid}/steps` (REST) |
| A step's log | `GET /pipelines/${pipeline_uuid}/steps/${step_uuid}/log` (REST); `twg bitbucket pipeline tail` |
| Artifacts | none in the public API |

### Write to Bitbucket Pipelines

| Capability | Command |
| --- | --- |
| Rerun | none in the public API; `twg bitbucket pipeline rerun-failed` |
| Cancel | `POST /pipelines/${pipeline_uuid}/stopPipeline` (REST) |
| Start a run | `POST /pipelines` with a `target` of a branch, a tag, or a commit, and a `selector` for a custom pipeline (REST); `twg bitbucket pipeline run` |
| Report a status | `POST /commit/${sha}/statuses/build` with `key`, `state`, and `url` (REST), for another CI system on Bitbucket |

### Bitbucket Pipelines schemas

- [Bitbucket Pipelines configuration JSON Schema](https://api.bitbucket.org/schemas/pipelines-configuration),
  SchemaStore's "bitbucket-pipelines" entry
- [Bitbucket REST OpenAPI 3](https://dac-static.atlassian.com/cloud/bitbucket/swagger.v3.json), with the pipelines and
  commit statuses routes, and the [Swagger 2 form](https://api.bitbucket.org/swagger.json)

## Buildkite

`bk` is Buildkite's CLI. It signs in with `bk auth login` or reads `BUILDKITE_API_TOKEN`. Its commands take `-o json`
(or `--json`), and `bk api` calls the REST API, or the GraphQL API with `-f`. The REST API pages with `page` and
`per_page`. Neither filters by what started a build; a build's `source` is `webhook`, `api`, `ui`, `trigger_job`, or
`schedule`, and its `pull_request` is empty for a branch build.

### Read from Buildkite

| Capability | Command |
| --- | --- |
| Recent runs | `bk build list -p "${org}/${pipeline}" --branch "${branch}" --limit 20 -o json` |
| A run's jobs and failed job | `bk build view "${build}" -o json`; `bk job list --pipeline "${pipeline}" --build "${build}" --state failed` |
| A job's log | `bk job log "${job_id}"` |
| Artifacts | `bk artifacts list "${build}"`; `bk artifacts download "${artifact_id}" --build "${build}"` |

### Write to Buildkite

| Capability | Command |
| --- | --- |
| Rerun | `bk build rebuild "${build}"`; `bk build retry-failed-jobs "${build}"`; `bk job retry "${job_id}"` |
| Cancel | `bk build cancel "${build}" -p "${pipeline}"` |
| Start a run | `bk build create -b "${branch}" -c "${sha}" -m "${message}"` |
| Report a status | Buildkite posts it; see the table above |

A Buildkite job is a command step; the steps inside a script are its log's lines.

### Buildkite schemas

- [Buildkite pipeline JSON Schema](https://raw.githubusercontent.com/buildkite/pipeline-schema/main/schema.json),
  SchemaStore's "Buildkite" entry
- [Buildkite GraphQL schema](https://raw.githubusercontent.com/buildkite/cli/main/schema.graphql), the copy the CLI
  is built from
- No OpenAPI document for the REST API was found.

## Tekton

`tkn` is Tekton's CLI. It signs in through the kubeconfig and takes `-n` for the namespace. `list` and `describe` take
`-o json`; lists have `--limit` and no pages. A run is a `PipelineRun` of `TaskRun`s, each with steps. Tekton keeps
no history once runs are pruned; Tekton Results, with its `tkn-results` CLI, stores it. Pipelines-as-Code ties
Tekton to a code forge: it starts runs from `.tekton/` on each event and posts their results back.

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
| Report a status | Pipelines-as-Code posts it; see the table above |

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

## Argo Workflows

`argo` is Argo Workflows' CLI. It signs in through the kubeconfig, or to an Argo Server with `ARGO_SERVER` and
`ARGO_TOKEN`. It prints `-o json`, and lists page with `--chunk-size`. A run is a `Workflow` of nodes; runs that are
gone from the cluster stay in the archive, `argo archive list`. Argo Workflows has no code forge integration of its
own: Argo Events receives the code forge's webhooks, and a step posts the status. Argo CD, its sibling, belongs to
[cd.md](cd.md).

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

## Forgejo and Gitea Actions

Forgejo and Gitea run GitHub Actions' workflow syntax on runners of their own, and post commit statuses as their
own CI. Both are also code forges, so a provider for either would come with a forge provider; see
[forges.md](forges.md). Gitea's `tea` signs in with `tea login add` and prints `-o json`. Forgejo's `fj` is a
community CLI with no JSON output and only `fj actions tasks` and `fj actions dispatch` for Actions, so the Forgejo
column below is mostly REST. Both REST APIs live under `/api/v1/repos/${owner}/${repo}/actions/` and page with `page`
and `limit`.

### Read from Forgejo and Gitea Actions

| Capability | Gitea | Forgejo |
| --- | --- | --- |
| Recent runs | `tea actions runs ls --event push --branch "${branch}" -o json` | `GET runs?event=push&ref=${ref}` (REST); `fj actions tasks` lists jobs |
| A run's jobs | `tea actions runs view "${run_id}" --jobs` | `GET runs/${run_id}/jobs` (REST) |
| A job's log | `tea actions runs logs "${run_id}" --job "${job_id}"` | `GET jobs/${job_id}/logs` (REST) |
| Artifacts | `GET runs/${run_id}/artifacts`, then `GET artifacts/${artifact_id}/zip` (REST) | the same routes (REST) |

### Write to Forgejo and Gitea Actions

| Capability | Gitea | Forgejo |
| --- | --- | --- |
| Rerun | `POST runs/${run_id}/rerun` or `rerun-failed-jobs` (REST) | `POST runs/${run_id}/rerun` (REST) |
| Cancel | `tea actions runs cancel "${run_id}"` | `POST runs/${run_id}/cancel` (REST) |
| Start a run | `tea actions workflows dispatch "${workflow}" --ref "${ref}"` | `fj actions dispatch "${workflow}" "${ref}"` |
| Report a status | the code forge's `POST /api/v1/repos/${owner}/${repo}/statuses/${sha}` (REST) | the same route (REST) |

### Forgejo and Gitea Actions schemas

- The [GitHub workflow JSON Schema](https://www.schemastore.org/github-workflow.json) covers both; SchemaStore matches
  it to `.gitea/workflows/` and `.forgejo/workflows/`.
- [Gitea's Swagger](https://gitea.com/swagger.v1.json), served by gitea.com, and
  [Forgejo's Swagger](https://codeberg.org/swagger.v1.json), served by Codeberg; Forgejo keeps its source in
  [the Forgejo repository](https://codeberg.org/forgejo/forgejo/raw/branch/forgejo/templates/swagger/v1_json.tmpl)

## Woodpecker CI

`woodpecker-cli` ships with Woodpecker. It reads `WOODPECKER_SERVER` and `WOODPECKER_TOKEN`. It prints no JSON:
`ls`, `show`, and `last` take `--output table` or `--output go-template=`, and `ps` takes `--format`. Lists page up to
`--limit`. A run is a pipeline of workflows and steps; the server filters runs by `event`, `branch`, and `status`.

### Read from Woodpecker CI

| Capability | Command |
| --- | --- |
| Recent runs | `woodpecker-cli pipeline ls "${repo}" --event push --branch "${branch}" --limit 20` |
| A run's steps and failed step | `woodpecker-cli pipeline ps "${repo}" "${number}"` |
| A step's log | `woodpecker-cli pipeline log show "${repo}" "${number}" "${step}"` |
| Artifacts | none |

### Write to Woodpecker CI

| Capability | Command |
| --- | --- |
| Rerun | `woodpecker-cli pipeline start "${repo}" "${number}"` |
| Cancel | `woodpecker-cli pipeline stop "${repo}" "${number}"`; `pipeline kill` forces it |
| Start a run | `woodpecker-cli pipeline create "${repo}" --branch "${branch}"` |
| Report a status | Woodpecker posts it; see the table above |

### Woodpecker CI schemas

- [Woodpecker pipeline JSON Schema](https://raw.githubusercontent.com/woodpecker-ci/woodpecker/main/pipeline/frontend/yaml/linter/schema/schema.json),
  SchemaStore's "Woodpecker pipeline config" entry
- The server generates its Swagger 2 document and serves it at `/swagger/doc.json`, as
  [Woodpecker's own instance](https://ci.woodpecker-ci.org/swagger/doc.json) does; the repository keeps no static
  copy.

## Drone

`drone` reads `DRONE_SERVER` and `DRONE_TOKEN`. It has no JSON flag; `--format '{{ jsonify . }}'` prints JSON through
a Go template. `drone build ls` filters its page by branch, event, and status on the client. Drone 2 releases come
from the `drone` branch of `harness/harness`, whose default branch is Harness Open Source.

### Read from Drone

| Capability | Command |
| --- | --- |
| Recent runs | `drone build ls "${repo}" --event push --branch "${branch}" --limit 20 --format '{{ jsonify . }}'` |
| A run's stages and steps | `drone build info "${repo}" "${number}" --format '{{ jsonify . }}'`: `stages[].steps[]` with `exit_code` |
| A step's log | `drone log view "${repo}" "${number}" "${stage}" "${step}"` |
| Artifacts | none |

### Write to Drone

| Capability | Command |
| --- | --- |
| Rerun | `drone build restart "${repo}" "${number}"` |
| Cancel | `drone build stop "${repo}" "${number}"` |
| Start a run | `drone build create "${repo}" --branch "${branch}"` |
| Report a status | Drone posts it; see the table above |

### Drone schemas

- [Drone pipeline JSON Schema](https://www.schemastore.org/drone.json), SchemaStore's own "drone.json"; Drone
  publishes none.

## Concourse

`fly` is Concourse's CLI. It signs in with `fly -t "${target}" login -c "${concourse_url}"` and keeps the token in
`~/.flyrc`. List commands take `--json`; `watch`, `trigger-job`, `rerun-build`, and `abort-build` do not. `fly curl`
calls any API route. A run is a build of a job, and Concourse has no notion of a change: what starts a job is a
resource, such as a git branch or a pull request resource.

### Read from Concourse

| Capability | Command |
| --- | --- |
| Recent runs | `fly -t "${target}" builds -p "${pipeline}" -c 20 --json` |
| A run's steps | `GET /api/v1/builds/${build_id}/plan` and `/events` through `fly curl`; no command names the failed step |
| A run's log | `fly -t "${target}" watch -j "${pipeline}/${job}" -b "${build}"` |
| Artifacts | none for a job's build; outputs go to resources |

### Write to Concourse

| Capability | Command |
| --- | --- |
| Rerun | `fly -t "${target}" rerun-build -j "${pipeline}/${job}" -b "${build}"` |
| Cancel | `fly -t "${target}" abort-build -j "${pipeline}/${job}" -b "${build}"` |
| Start a run | `fly -t "${target}" trigger-job -j "${pipeline}/${job}"` |
| Report a status | a resource in the pipeline posts it |

### Concourse schemas

Concourse publishes none: no OpenAPI for its API, whose routes are in `atc/routes.go`, and no JSON Schema for its
pipeline files.

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
