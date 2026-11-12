# aws

`aws` is Amazon's CLI, and `aws codecommit` holds every CodeCommit call. It authenticates with `aws configure`,
`aws sso login`, or the standard `AWS_*` variables, per profile and region. It prints JSON by default, `--query`
filters with JMESPath, and a list reads every page unless `--no-paginate` or `--max-items` stops it. The API is the
CLI: there is no raw-route escape hatch, and none is needed.

This page says what `aws` reads and writes where it meets kit's scope, for each service it reaches, and the
machine-readable descriptions each publishes. The index of these pages is [1st.md](1st.md).

| Service | Category | kit status |
| --- | --- | --- |
| [AWS CodeCommit](#aws-codecommit) | [code forge](forges.md) | `planned` |
| [AWS CodeBuild and CodePipeline](#aws-codebuild-and-codepipeline) | [CI](ci.md) | `planned`, with an `aws` code forge |
| [AWS CodeDeploy and CodePipeline](#aws-codedeploy-and-codepipeline) | [CD](cd.md) | none; an `aws` code forge is planned |

## AWS CodeCommit

AWS closed CodeCommit to new customers in July 2024 and returned it to general availability on 24 November 2025; new
customers can create repositories in 29 regions. See
[AWS CodeCommit returns to general availability](https://aws.amazon.com/blogs/devops/aws-codecommit-returns-to-general-availability/)
and the [CodeCommit document history](https://docs.aws.amazon.com/codecommit/latest/userguide/history.html).
kit's plan for CodeCommit and AWS's other developer services is [below](#a-plan-for-an-aws-provider).

### Read from AWS CodeCommit

| Capability | Command |
| --- | --- |
| Signed in | `aws sts get-caller-identity` |
| Owner and name | `aws codecommit get-repository --repository-name "${repo}" --query repositoryMetadata.[accountId,repositoryName]` |
| Browser URL | none; the console's is `https://${region}.console.aws.amazon.com/codesuite/codecommit/repositories/${repo}/pull-requests/${n}` |
| Caller's login | `aws sts get-caller-identity --query Arn` |
| Rate limits | none; a throttled call fails with `ThrottlingException`, and the quotas are documented, not read |
| Marks | none cheap; `aws codecommit list-pull-requests` returns IDs alone, and a change's history is `describe-pull-request-events --pull-request-id "${n}"`. Repository triggers to SNS or Lambda (`put-repository-triggers`) and EventBridge push changes instead |
| Branch for an issue | `aws codecommit list-pull-requests --repository-name "${repo}" --pull-request-status OPEN`, then `get-pull-request` on each, for a `sourceReference` of `refs/heads/${n}-*` |
| Title, body | `aws codecommit get-pull-request --pull-request-id "${n}" --query pullRequest.[title,description]` |
| Author | `aws codecommit get-pull-request --pull-request-id "${n}" --query pullRequest.authorArn` |
| State and merge commit | `aws codecommit get-pull-request --pull-request-id "${n}" --query 'pullRequest.[pullRequestStatus, pullRequestTargets[0].mergeMetadata]'`: `OPEN` or `CLOSED`, and `isMerged` with `mergeCommitId` |
| Change on a branch | `list-pull-requests`, then `get-pull-request` on each, matching `pullRequestTargets[].sourceReference` |
| Closing issues | none |
| Review threads | `aws codecommit get-comments-for-pull-request --pull-request-id "${n}"`: comments and replies, with no resolved state |
| Checks rollup | none; `get-pull-request-approval-states` and `evaluate-pull-request-approval-rules` give the approval rules alone |
| Open changes | `aws codecommit list-pull-requests --repository-name "${repo}" --pull-request-status OPEN`, then `get-pull-request` on each |

### Write to AWS CodeCommit

| Capability | Command |
| --- | --- |
| Branch linked to an issue | `aws codecommit create-branch --repository-name "${repo}" --branch-name "${n}-${slug}" --commit-id "${sha}"`, with no link to an issue |
| Open a draft | none; `aws codecommit create-pull-request --title "${title}" --description "" --targets repositoryName="${repo}",sourceReference="${branch}",destinationReference="${default}"` opens one ready for review |
| Mark ready | none |
| Set the title | `aws codecommit update-pull-request-title --pull-request-id "${n}" --title "${title}"` |
| Set the body | `aws codecommit update-pull-request-description --pull-request-id "${n}" --description "${body}"` |

Gaps: CodeCommit has no draft, no issues, no checks on a pull request, and no resolved state on a comment, so the
contract's draft, closing issues, threads, and checks have no answer there. A change for a branch takes a read per
open pull request, as the list returns IDs alone.

### AWS CodeCommit schemas

- [AWS CodeCommit Smithy model](https://raw.githubusercontent.com/aws/api-models-aws/main/models/codecommit/service/2015-04-13/codecommit-2015-04-13.json),
  from aws/api-models-aws, the source the SDKs are generated from.
- [AWS CodeCommit botocore model](https://raw.githubusercontent.com/boto/botocore/develop/botocore/data/codecommit/2015-04-13/service-2.json),
  the shape `aws` itself reads, with its pagination in the same directory.
- No JSON Schema for trigger or EventBridge payloads was found.

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
| Report a status | CodeBuild posts it when the project reports build status; see [who reports to the code forge](ci.md#who-reports-to-the-code-forge) |

AWS CodeCommit, the code forge an `aws` provider would read, is open to new customers: AWS closed it to them in July
2024 and made it generally available again on 24 November 2025. No retirement of CodeBuild or CodePipeline was found.

### AWS CodeBuild and CodePipeline schemas

- [CodeBuild service model](https://raw.githubusercontent.com/boto/botocore/develop/botocore/data/codebuild/2016-10-06/service-2.json)
  and [CodePipeline service model](https://raw.githubusercontent.com/boto/botocore/develop/botocore/data/codepipeline/2015-07-09/service-2.json),
  the JSON the CLI is generated from
- The same APIs as Smithy models: [CodeBuild](https://raw.githubusercontent.com/aws/api-models-aws/main/models/codebuild/service/2016-10-06/codebuild-2016-10-06.json)
  and [CodePipeline](https://raw.githubusercontent.com/aws/api-models-aws/main/models/codepipeline/service/2015-07-09/codepipeline-2015-07-09.json)
- AWS publishes no JSON Schema for `buildspec.yml`, and SchemaStore has none.

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

## A plan for an AWS provider

AWS's developer services meet kit's scope in four places: AWS CodeCommit hosts a repository, AWS CodeBuild and AWS
CodePipeline run its builds, AWS CodeDeploy deploys its releases, and IAM names the people who do each. kit has no
provider for any of them. This section is the plan for one: which seam each service sits behind, what the forge seam's
declaration says each lacks, the stand-ins a repository on AWS configures, and what must be true to start. AWS
CodeCommit closed to new customers in July 2024 and returned to general availability on 24 November 2025.

### The services and their seams

| Service | Seam | What it answers | Command line |
| --- | --- | --- | --- |
| CodeCommit | the forge seam, a provider `forge/aws.sh` | the repository, branches, pull requests, their comments and approval rules | `aws codecommit` |
| CodePipeline | the forge seam's `runs`, from the same provider | a pipeline's executions whose source is the repository | `aws codepipeline` |
| CodeBuild | the forge seam's checks, through a stand-in | the builds of a pull request's source commit | `aws codebuild` |
| CodeDeploy | none yet; kittui's deployments tab, see [cd.md](cd.md) | deployments and their state per deployment group | `aws deploy` |
| IAM and STS | the forge seam's `me` and a change's author | the caller's and an author's ARN | `aws sts get-caller-identity` |
| EventBridge | the forge seam's `marks`, as a later step | the repository's events, pushed rather than read | `aws events` |

CodeArtifact is a package registry, so it meets the dependency providers rather than a forge: `aws codeartifact
login --tool npm` or `--tool pip` points npm or uv at it, and kit's dependency seam needs nothing of its own for that.

### The forge provider

`forge/aws.sh` would read the repository from the primary remote's URL, in any of its three forms:

- `https://git-codecommit.${region}.amazonaws.com/v1/repos/${repo}`
- `ssh://git-codecommit.${region}.amazonaws.com/v1/repos/${repo}`
- `codecommit::${region}://${profile}@${repo}`, the git-remote-codecommit helper's form

A host under `amazonaws.com` that starts with `git-codecommit.` names the forge, as `dev.azure.com` names Azure DevOps.
`forge_settings.aws` gives what the URL cannot: the profile, when the URL names none, and the stand-ins below.

It would implement contract 3 and declare that CodeCommit lacks these features:

| Feature | Why CodeCommit lacks it | What kit does without it |
| --- | --- | --- |
| `issues`, `sub-issues` | CodeCommit has no issues | a record in `trackers` claims `#`, or the forge's own tracker refuses with that remedy |
| `linked-branches` | a branch links to nothing | `kit work` pushes the branch it names |
| `drafts` | a pull request opens ready for review | the change opens ready, unless the `drafts` stand-in marks it |
| `closing-issues` | a pull request closes nothing | `kit linked` and `kit settle` take the body's word, or the `closing-issues` stand-in |
| `review-threads` | a comment has no resolved state | the changes stage draws no thread count |
| `checks` | no build posts a status on a pull request | the `checks` stand-in reads CodeBuild |
| `quota` | throttling is an error, `ThrottlingException`, never a count | the status bar draws no quota |
| `marks` | no read is cheap: a pull request list gives identifiers alone | the `marks` stand-in reloads on a timer |
| `releases` | CodeCommit has no releases | the releases stage lists none; a release is the tag `kit release` pushes |

`runs` stays native: the provider answers it from CodePipeline, the executions of each pipeline whose source action
names the repository, newest first, with each execution's stages as its jobs.

### Stand-ins

A stand-in is what the repository chooses in place of a feature its forge lacks, in its shared `kit.yml`, under the
forge it is for. On CodeCommit:

```yaml
forge_settings:
  aws:
    profile: dev
    stand_ins:
      drafts: {title_prefix: "[draft] "}
      closing-issues: body
      checks: {codebuild: [example-ci]}
      marks: {every: 60}
```

- `drafts: {title_prefix: …}` opens a change with the prefix on its title and draws it as a draft while the title has
  it; `kit ready` takes the prefix off. CodeCommit has no labels, so `{label: …}`, the stand-in a forge with labels
  takes, is refused here.
- `closing-issues: body` reads the issues a change closes from its body's closing keywords, `Closes #7`, and asks the
  tracker each one's state.
- `checks: {codebuild: […]}`, the one stand-in this plan adds, would read the named CodeBuild projects' builds of
  the change's source commit as its checks: `aws codebuild list-builds-for-project`, then
  `aws codebuild batch-get-builds`.
- `marks: {every: 60}` gives the forge's stages a mark that changes once a minute, so kittui reloads them on that
  timer rather than never.

`kit flow --capabilities` then names each of these features a stand-in, not none, so kittui draws a change's draft
and checks and leaves its threads blank.

### Identity

CodeCommit names the caller and every author by an IAM ARN, where the tracker names a login. An ARN of an assumed
role carries the session's name as its last segment, `arn:aws:sts::123456789012:assumed-role/Developer/ada`, and IAM
Identity Center sets that to the person's user name. The provider would answer `me` and a change's author with that
segment, and `forge_settings.aws.logins` would map any other ARN to a login, so `kit work` assigns the right person and
`linked_exempt` names a bot by the login it has.

### Costs

`list-pull-requests` returns identifiers alone, so the changes stage costs one `get-pull-request` per open pull
request, and the `checks` stand-in one more CodeBuild read each. The forge seam's snapshot shares one such read between
the stages of a load, for a minute. A repository with many open pull requests would want the EventBridge step: a
rule that writes each pull request event where `marks` can read it, so the stages reload only on a change.

### What must be true to start

- A repository that needs it: a team on CodeCommit that uses kit, and names the tracker it keeps its issues in.
- The `checks` stand-in: the forge seam reads `drafts`, `closing-issues`, and `marks` stand-ins under
  `forge_settings.<forge>.stand_ins`, with `kit ready` to end a draft, and `checks` is the one this plan adds.
- The tests: a fake `aws` in `tests/forge-conformance.bats`, answering from the botocore model's shapes, as the fakes
  of `gh`, `glab`, and `az` answer from their tools'.
- CodeDeploy waits for a seam of its own, with kittui's environments and deployments tabs; see [cd.md](cd.md).

see: issue #149, [AWS CodeCommit](#aws-codecommit), [AWS CodeBuild and CodePipeline](#aws-codebuild-and-codepipeline),
[AWS CodeDeploy and CodePipeline](#aws-codedeploy-and-codepipeline)
