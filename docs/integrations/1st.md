# Integrations

An integration is a service kit talks to besides git: the code forge that hosts the repository, the tracker that
holds its issues, the CI system that runs its checks, and the CD system that ships its releases. kit is tied to git
and needs none of them to install, update, check, lint, or run `kit context`. Each one sits behind a seam, the
contract plus the point that picks a provider; [architecture.md](../architecture.md#code-forges-seams-and-providers)
says what a seam is.

These pages come in two kinds. A category's page lists its top providers, open-source ones included, with the
capabilities kit needs from the category and which provider answers each. A tool's page says, for every service that
command line tool reaches, what it reads and writes where it meets kit's scope, and which machine-readable
descriptions of its API and its configuration files the provider publishes.

| Page | Category | Seam | What in kit uses it |
| --- | --- | --- | --- |
| [forges.md](forges.md) | code forges: the repository, its branches, and its changes | `kit/modules/bash/forge.sh` | `kit work`, `kit submit`, `kit linked`, `kit policy`, `kit tidy`, and the changes and underway stages of `kit flow` |
| [trackers.md](trackers.md) | trackers: issues, their parents, iterations, and milestones | `kit/modules/bash/tracker.sh`; see [../trackers.md](../trackers.md) | `kit work`, `kit gather`, `kit assign`, `kit close`, `kit settle`, `kit new issue`, and the backlog stage |
| [ci.md](ci.md) | continuous integration: runs, their jobs, and a change's checks | the forge seam's `runs` and a change's checks | `kit ci`, which a pipeline runs, and the builds and changes stages |
| [cd.md](cd.md) | continuous delivery: releases, environments, and deployments | the forge seam's `releases` | `kit release`, and the releases stage |

## Providers in kit

| Category | Providers | Selected by |
| --- | --- | --- |
| code forge | `github` over `gh`, `gitlab` over `glab`, `ado` over `az` | the `forge` setting, else the primary remote's host |
| tracker | `forge`, which is Azure Boards on an `ado` forge, `ado` over `az boards`, `web` | the reference's prefix |
| CI | the code forge's own, through the forge seam | the code forge |
| CD | the code forge's releases, through the forge seam | the code forge |

A CI or CD system that is not the code forge's own reaches kit today only through the code forge: it reports a
commit status or a check run there, and kit reads it with the change's checks.

A forge provider declares what its forge lacks, in the forge seam's contract version 3, and `kit flow
--capabilities` prints the declaration; [architecture.md](../architecture.md#code-forges-seams-and-providers) says how
a verb answers a call of a feature the forge lacks. [aws.md](aws.md#a-plan-for-an-aws-provider) holds the plan for
AWS's developer services, which kit has no provider for: what each lacks, and the stand-ins a repository on AWS would
configure.

## Tools

git is the one integration kit cannot work without. Every other tool reaches a provider in one category or more; kit's
own providers' tools come first, then the rest by name. A provider with no command line tool of its own has a page
under its own name, which says what its REST API answers.

| Page | The services it reaches |
| --- | --- |
| [git](git.md) | the repository itself: its checkouts, worktrees, branches, remotes, commits, tags, and index |
| [gh](gh.md) | GitHub pull requests (code forge), GitHub Issues (tracker), GitHub Projects (tracker), GitHub Actions (CI), GitHub releases and deployments (CD) |
| [glab](glab.md) | GitLab merge requests (code forge), GitLab issues and work items (tracker), GitLab CI/CD (CI), GitLab releases and deployments (CD) |
| [az](az.md) | Azure DevOps Repos (code forge), Azure Boards (tracker), Azure Pipelines runs (CI), Azure Pipelines environments and releases (CD) |
| [aws](aws.md) | AWS CodeCommit (code forge), AWS CodeBuild and CodePipeline (CI), AWS CodeDeploy and CodePipeline (CD) |
| [argo](argo.md) | Argo Workflows (CI) |
| [argocd](argocd.md) | Argo CD (CD) |
| [bk](bk.md) | Buildkite (CI) |
| [bkt](bkt.md) | Bitbucket Cloud (code forge), Bitbucket Data Center (code forge), Bitbucket Pipelines (CI) |
| [bugzilla](bugzilla.md) | Bugzilla (tracker) |
| [circleci](circleci.md) | CircleCI (CI) |
| [drone](drone.md) | Drone (CI) |
| [fj](fj.md) | Forgejo and Codeberg (code forge) |
| [flux](flux.md) | Flux (CD) |
| [fly](fly.md) | Concourse (CI) |
| [gerrit](gerrit.md) | Gerrit (code forge) |
| [harness](harness.md) | Harness CD (CD) |
| [hut](hut.md) | SourceHut (code forge) |
| [jenkins](jenkins.md) | Jenkins (CI) |
| [jira](jira.md) | Jira (tracker) |
| [kargo](kargo.md) | Kargo (CD) |
| [kubectl](kubectl.md) | Argo Rollouts (CD), Kubernetes custom resources (CD) |
| [linear](linear.md) | Linear (tracker) |
| [octopus](octopus.md) | Octopus Deploy (CD) |
| [op](op.md) | OpenProject (tracker) |
| [plane](plane.md) | Plane (tracker) |
| [rad](rad.md) | Radicle (code forge) |
| [redmine](redmine.md) | Redmine (tracker) |
| [short](short.md) | Shortcut (tracker) |
| [spin](spin.md) | Spinnaker (CD) |
| [taiga](taiga.md) | Taiga (tracker) |
| [tea](tea.md) | Gitea pull requests (code forge), Gitea and Forgejo issues (tracker), Forgejo and Gitea Actions (CI) |
| [tkn](tkn.md) | Tekton (CI) |
| [woodpecker-cli](woodpecker-cli.md) | Woodpecker CI (CI) |
| [yt](yt.md) | YouTrack (tracker) |

## How a page reads

A category's page has these sections:

1. Scope: the capabilities kit needs from the category, each read or write, each tied to the seam call, verb, or
   stage that uses it.
2. Providers: one table of every provider, its hosting and license, its command line tool, kit's status for it,
   how the tool prints JSON, and the descriptions it publishes. Each provider's name leads to its section on its
   tool's page.
3. Coverage: a matrix of capability by provider, each cell `cli` when the tool has a command, `api` when only the raw
   API answers, and `—` when neither does.

A tool's page opens with the tool, how it signs in and prints JSON, and a table of the services it reaches, each with
its category and kit's status for it. One section per service follows: a Read table and a Write table from kit's
capability to the exact command, the gaps, and the schemas.

A provider's status is `provider` when kit has one, `planned` when a seam for it is planned, and `none` otherwise.

## Schemas

A provider can publish three kinds of machine-readable description, and each page links every one it found, to the
raw document:

- its API: an OpenAPI or Swagger document for REST, or a GraphQL schema
- its configuration files: a JSON Schema for a workflow or pipeline file, or the OpenAPI schema of a Kubernetes custom
  resource definition (CRD), often also listed in [SchemaStore](https://www.schemastore.org/)
- its command line tool's output: the JSON fields a command prints, which few tools publish beyond their help text

A tool's page links every published description for the services it reaches. Where kit has a provider, the tool's
output gets a schema of its own: beside the page, in a directory named for the tool, one JSON Schema per command kit
runs whose JSON it reads, typed from the provider's API description, holding only the fields kit reads, with an
example of the output. The gate validates each example against its schema. [gh/](gh/), [glab/](glab/), and [az/](az/)
hold them; [git.md](git.md#schemas) says why git has none.

A schema of a tool's output is what a provider's test fixtures can be checked against. `tests/forge-conformance.bats`
and `tests/tracker.bats` run each provider against a fake of its tool, and a fake is only as true as the documents it
returns.
