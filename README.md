# empowerite/kit: **k**eep **i**t **t**rustworthy

Do you trust your agents? They were trained on the whole of what people have written, and that includes every shortcut
anyone ever took: the "done" that was not, the test that was skipped and reported as passing, the rebase that quietly
rewrote a colleague's branch. An agent is a tireless new colleague who has read everything and holds none of your house
rules, unless you put them where it looks. kit is the house rules, put where agents look, and the checks that hold
agents to them, as a versioned release any repository can take and keep taking.

What kit checks:

- `kit ci` refuses a pull request when:
  - it is not linked to an open issue,
  - its branch name, commit subjects, or title break the repository's naming policy, or
  - the repository's gate did not run.

  It runs in the delivered GitHub Actions job or in any other pipeline.
- `kit check` fails when a delivered file differs from the release, or a generated region differs from the output
  of its command.
- `kit lint` runs each rule's linter over the files the rule covers, before a commit or at any time.
- A deny list blocks the raw git commands the rules forbid, such as `git rebase`.

What else kit provides:

- `.kit/1st.md`, the file every agent reads first, and one rule file per kind of file.
- `kit issue`, which turns an issue into a worktree, a linked branch, and a draft pull request.
- `kit toolchain`, which installs, lists, and updates the gate's tools (mise, Homebrew, or a container image) and
  runs the gate with them.
- `kit status`, which shows every open pull request and the result of its checks.
- A rule that agents never change the desk: the root checkout on your machine, on whatever branch it currently has.

A repository keeps its own rules next to the delivered ones. The design is in
[#1](https://github.com/empowerite/kit/issues/1) and [#4](https://github.com/empowerite/kit/issues/4).

## Why a release, and not a copy

A copied file drifts. A release does not:

- It is pinned, so a repository knows which version of the rules applies.
- It is updated by a pull request that bumps the version, so every repository gets the change the same way.
- It is checked: `kit check` fails the build when someone edits a delivered file in place.

A repository's own rules live in its own files, and updates leave them alone.

## Install

A release is a tarball containing `bin/`, `bash/`, `packages/`, and a `version` file. Install it the same way the
repository installs its other tools.

With mise, pin it in `mise.toml`:

```toml
[tools]
"github:empowerite/kit" = { version = "0.1.0", version_prefix = "release/" }
```

With Homebrew:

```sh
brew install empowerite/tap/kit
```

By hand, with `gh`. This downloads the latest release, never a pre-release. For a specific version, run
`gh release download "release/${version}" ...` instead. Unpack it and put its `bin/` on `PATH`:

```sh
gh release download --repo empowerite/kit --pattern 'kit-*.tar.gz' --output kit.tar.gz
mkdir kit && tar --extract --gzip --file kit.tar.gz --directory kit --strip-components 1
export PATH="${PWD}/kit/bin:${PATH}"
```

Requirements:

- bash 4.4 or later. macOS ships bash 3.2; run `brew install bash`.
- git, `jq`, and `curl`, which `kit update` uses to fetch the release it updates from.
- For `kit issue`, `kit status`, `kit tidy`, and `kit linked`: the command-line tool of the forge, the service that
  hosts the repository. That is `gh` for GitHub or `glab` for GitLab.
- For an Azure DevOps tracker: `az`, the Azure CLI.

With mise, an exact version pin installs as soon as a release exists. A range pin waits one day after a release
unless `[settings]` contains `minimum_release_age_excludes = ["github:empowerite/kit"]`.

## Take the kit

1. Run `kit install` with the packages you want. The `kit` package (the entry file and the rules) is always
   installed. The optional packages are:

   | Package | Delivers |
   | --- | --- |
   | `ci-github` | the GitHub Actions workflow that runs `kit ci` |
   | `lefthook` | a pre-commit hook that runs `kit lint` and `kit generate --check` |
   | `claude` | the Claude Code adapter |
   | `policy-opa` | the naming policy as Open Policy Agent rules |

   `kit install ci-github lefthook claude` takes three of them. `kit package add` adds one later.

   `install` also writes a line into `AGENTS.md` telling every agent to read `.kit/1st.md`, and adds `/worktrees/`
   to `.gitignore`. Each sits in a region between a `kit:generate` line and a `kit:end` line. The rest of each file
   is yours, and you can move the region anywhere in it. `kit generate` rewrites a region from its command, and
   `kit check` fails when a region is out of date. Your own files can contain regions too; see
   [docs/generate.md](docs/generate.md).
2. Commit what `install` wrote.
3. Give the repository a gate, the command that CI runs. kit takes the first of these:
   - any command, set as `gate` in `.kit/settings.yml`
   - a `ci` recipe in the root justfile
   - a `ci` target in a Makefile
   - a `ci` script in `package.json`

   `kit ci` runs the gate together with `kit check`, `kit linked`, and `kit policy`, and fails when there is no gate.
   The gate runs with the tools your toolchain manages; see [docs/toolchains.md](docs/toolchains.md). The naming
   policy is described in [docs/policy.md](docs/policy.md).
4. Run the linters before each commit. `kit lint` lints the staged files and needs no hook manager. To run it from
   lefthook instead:
   - pin `lefthook`,
   - add `extends: [.kit/lefthook.yml]` to `lefthook.yml`, and
   - run `lefthook install` once per clone.

   With the `lefthook` package installed, `kit check` fails until `lefthook.yml` has that `extends` line. kit never
   writes it for you.

Claude Code reads `AGENTS.md` only when there is no `CLAUDE.md`. The `claude` package adds a region to
`.claude/CLAUDE.md` that imports `AGENTS.md`, so Claude always reads the same instructions as every other agent. The
rest of `.claude/CLAUDE.md` is yours. Without the package, any agent that reads `AGENTS.md` still gets the rules.

`kit package list`, `kit package add`, and `kit package remove` show and change the installed packages. The package format
is in [docs/packages.md](docs/packages.md).

### Working with the rules

- Before an agent changes a file, it runs `kit context "${path}"` to find the rules that apply. The `claude` package
  makes this call automatically through a Claude Code hook; see [docs/context.md](docs/context.md).
- `kit new file "${path}"` creates a file from its rule's template, and `kit lint` checks that every file keeps the
  shape of a template; see [docs/templates.md](docs/templates.md).

### Updating

Bump the pin or reinstall, then run `kit update`:

- A delivered file you never edited is replaced.
- A delivered file you edited is merged three ways, with the previous release as the base. `kit update` downloads
  the previous release from GitHub when it is not on the machine. Where it cannot, set `KIT_BASE` to a copy of it.
- Your own deny list entries and settings keys are kept.

`kit check` fails, locally and in CI, when a delivered file differs from the pinned release.

## Setup

The rest of this README is about working on this repository, which is the kit's own source and also uses the kit.

Tools come from mise. Every commit must be signed; the branch ruleset rejects unsigned commits. If you have no signing
key that GitHub verifies, create and register one:

- [generating a new GPG key](https://docs.github.com/en/authentication/managing-commit-signature-verification/generating-a-new-gpg-key)
- [adding a GPG key to your GitHub account](https://docs.github.com/en/authentication/managing-commit-signature-verification/adding-a-gpg-key-to-your-github-account)

Then, once per clone, with the key's id in `signing_key`:

```sh
git config gpg.format openpgp
git config user.signingkey "${signing_key}"
git config commit.gpgsign true
export MISE_ENV=desk
mise install
lefthook install
```

Put `export MISE_ENV=desk` in your shell profile. It makes mise read `mise.desk.toml`, the tools only a developer
machine needs, on top of `mise.toml`. The delivered CI job sets `MISE_ENV=ci` the same way, for repositories that have
a `mise.ci.toml`.

## Getting started

Work here follows the workflow in [.kit/1st.md](.kit/1st.md):

1. Start from an issue. A change without an issue has not started.
2. Run `kit issue <N>`. It creates `worktrees/<N>-<slug>`, links the branch to the issue under Development, and opens
   a draft pull request. It never switches the branch of the checkout you run it from. For an issue in another
   tracker, use that tracker's reference, such as `kit issue AB#4521` for an Azure DevOps work item; see
   [docs/trackers.md](docs/trackers.md).
3. Work in that worktree. Commits are signed, with the subject `#<N> <area>: <subject>`.
   - `kit rebase` rebases the branch onto `main` and pushes it.
   - `kit status` shows every open pull request and the worktree it lives in.
4. Run the gate, `kit gate`. It runs `bin/ci`, which lists what it checks.
5. Run `gh pr ready` when the work is done. The pull request body is the report.
6. A human merges. Then `kit tidy` removes the worktree and its branch.

## Maintaining

| Directory | Contents |
| --- | --- |
| `kit/` | the release, packed as it is: everything in it ships, and nothing outside it does |
| `kit/bin/` | the `kit` command and its verbs |
| `kit/bash/` | the bash modules the verbs share |
| `kit/packages/` | everything a release delivers |
| `bin/` | this repository's own commands, not on `PATH`: its gate, `bin/ci`; `bin/refresh`; and `bin/release` |
| `docs/` | the documentation, with a page per package under `docs/packages/` |

[docs/architecture.md](docs/architecture.md) describes how the pieces fit, with diagrams.
[docs/packages.md](docs/packages.md) describes the package format.
[docs/bash.md](docs/bash.md) describes the bash module system the verbs are built on, and how a repository extends it.

This repository installs its own kit. The installed copies are written by `bin/refresh`, which runs `kit update`
from the index. Do not edit them by hand:

| Installed copy | Written from |
| --- | --- |
| `.kit/1st.md`, `.kit/rules/`, `.kit/templates/`, `.kit/settings.schema.json`, `.kit/markdownlint.jsonc` | `kit/packages/kit/files/repo/.kit/` |
| `.kit/lefthook.yml` | `kit/packages/lefthook/files/repo/.kit/lefthook.yml` |
| `.github/workflows/kit-ci.yml` | `kit/packages/ci-github/templates/repo/.github/workflows/kit-ci.yml`, with `ci_runner` filled in |
| `.claude/CLAUDE.md` | `kit/packages/claude/regions/repo/.claude/CLAUDE.md`, a region; the rest of the file belongs to this repository |
| `.claude/settings.json` | `kit/packages/claude/files/repo/.claude/settings.json`, merged by key; here only the kit's entries |
| `AGENTS.md`, `.gitignore` | `kit/packages/kit/regions/repo/`, regions; the rest of each file belongs to this repository |
| `.kit/manifest` | the release version (`unreleased` here), the active packages, and a digest per delivered path |

`.kit/settings.yml` belongs to this repository and is not delivered.

When a commit changes a file under `kit/` or `.kit/settings.yml`, the installed copies must change in the same commit:

- With the pre-commit hook installed, this happens automatically: the hook runs `bin/refresh`, which regenerates
  the copies from the index and stages them.
- Without the hook, stage your change, run `bin/refresh`, and commit.

`kit/bin/kit check`, which the gate runs, names any installed copy that does not match its source. Each delivered file
says in its header that kit delivered it. A region's `kit:generate` line serves as its notice, and the table above
serves as the notice for `settings.json`.

## Release

- A release is the tag `release/<version>` on any commit on `main`, not only the latest one.
- A pre-release is `release/<version>-<suffix>`, such as `release/0.1.0-rc.1`, on a commit that is not on `main`.
  It is published as a GitHub pre-release.
- The tag is the version. The repository has no version file. The release workflow writes the part after `release/`
  into the packed kit and names the release after it.
- Published releases are immutable, so a tag is never reused.

The release workflow runs two commands:

1. `bin/release verify` rejects a tag that is not `release/<version>`, a release tag on a commit not on `main`, and a
   pre-release tag on a commit on `main`.
2. `bin/release publish` packs `kit/` (`bin/`, `bash/`, and `packages/`) and `version` into
   `kit-<version>.tar.gz` and publishes it.

`bin/release pack` builds the same tarball locally. `bin/release publish` refuses to publish a release anywhere but
the CI runner, so only the workflow publishes releases. A pre-release can be published from a developer machine.

A release is permanent. A pre-release is deleted, tag and all, once the branch it was tagged on is deleted and no
other branch contains its commit.

The branch ruleset is managed in [empowerite/infrastructure](https://github.com/empowerite/infrastructure).
