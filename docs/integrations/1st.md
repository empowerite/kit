# Integrations

An integration is a service kit talks to besides git: the code forge that hosts the repository, the tracker that
holds its issues, the CI system that runs its checks, and the CD system that ships its releases. kit is tied to git
and needs none of them to install, update, check, lint, or run `kit context`. Each one sits behind a seam, the
contract plus the point that picks a provider; [architecture.md](../architecture.md#code-forges-seams-and-providers)
says what a seam is.

These pages list the top providers in each category, open-source ones included. For each provider they say which
command line tool reaches it, what that tool reads and writes where it meets kit's scope, and which machine-readable
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
| code forge | `github` over `gh`, `gitlab` over `glab` | the `forge` setting, else the primary remote's host |
| tracker | `forge`, `ado` over `az boards`, `web` | the reference's prefix |
| CI | the code forge's own, through the forge seam | the code forge |
| CD | the code forge's releases, through the forge seam | the code forge |

A CI or CD system that is not the code forge's own reaches kit today only through the code forge: it reports a
commit status or a check run there, and kit reads it with the change's checks.

## How a page reads

Each page has the same sections:

1. Scope: the capabilities kit needs from the category, each read or write, each tied to the seam call, verb, or
   stage that uses it.
2. Providers: one table of every provider, its hosting and license, its command line tool, kit's status for it,
   how the tool prints JSON, and the descriptions it publishes.
3. One section per provider: the tool, a Read table and a Write table from kit's capability to the exact command,
   the gaps, and the schemas.
4. Coverage: a matrix of capability by provider, each cell `cli` when the tool has a command, `api` when only the raw
   API answers, and `—` when neither does.

A provider's status is `provider` when kit has one, `planned` when a seam for it is planned, and `none` otherwise.

## Schemas

A provider can publish three kinds of machine-readable description, and each page links every one it found, to the
raw document:

- its API: an OpenAPI or Swagger document for REST, or a GraphQL schema
- its configuration files: a JSON Schema for a workflow or pipeline file, or the OpenAPI schema of a Kubernetes custom
  resource definition (CRD), often also listed in [SchemaStore](https://www.schemastore.org/)
- its command line tool's output: the JSON fields a command prints, which few tools publish beyond their help text

A published schema is what a provider's test fixtures can be checked against. `tests/forge-conformance.bats` and
`tests/tracker.bats` run each provider against a fake of its tool, and a fake is only as true as the documents it
returns.
