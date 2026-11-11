# AWS

AWS's developer services meet kit's scope in four places: AWS CodeCommit hosts a repository, AWS CodeBuild and AWS
CodePipeline run its builds, AWS CodeDeploy deploys its releases, and IAM names the people who do each. kit has no
provider for any of them. This page is the plan for one: which seam each service sits behind, what the forge seam's
declaration says each lacks, the stand-ins a repository on AWS configures, and what must be true to start. The index
of these pages is [1st.md](1st.md).

AWS CodeCommit closed to new customers in July 2024 and returned to general availability on 24 November 2025; see
[forges.md](forges.md#aws-codecommit). What `aws codecommit` reads and writes, call by call, is on that page.

## The services and their seams

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

## The forge provider

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

## Stand-ins

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

## Identity

CodeCommit names the caller and every author by an IAM ARN, where the tracker names a login. An ARN of an assumed
role carries the session's name as its last segment, `arn:aws:sts::123456789012:assumed-role/Developer/ada`, and IAM
Identity Center sets that to the person's user name. The provider would answer `me` and a change's author with that
segment, and `forge_settings.aws.logins` would map any other ARN to a login, so `kit work` assigns the right person and
`linked_exempt` names a bot by the login it has.

## Costs

`list-pull-requests` returns identifiers alone, so the changes stage costs one `get-pull-request` per open pull
request, and the `checks` stand-in one more CodeBuild read each. The forge seam's snapshot shares one such read between
the stages of a load, for a minute. A repository with many open pull requests would want the EventBridge step: a
rule that writes each pull request event where `marks` can read it, so the stages reload only on a change.

## What must be true to start

- A repository that needs it: a team on CodeCommit that uses kit, and names the tracker it keeps its issues in.
- The `checks` stand-in: the forge seam reads `drafts`, `closing-issues`, and `marks` stand-ins under
  `forge_settings.<forge>.stand_ins`, with `kit ready` to end a draft, and `checks` is the one this plan adds.
- The tests: a fake `aws` in `tests/forge-conformance.bats`, answering from the botocore model's shapes, as the fakes
  of `gh`, `glab`, and `az` answer from their tools'.
- CodeDeploy waits for a seam of its own, with kittui's environments and deployments tabs; see [cd.md](cd.md).

see: issue #149, [forges.md](forges.md#aws-codecommit), [ci.md](ci.md#aws-codebuild-and-codepipeline),
[cd.md](cd.md#aws-codedeploy-and-codepipeline)
