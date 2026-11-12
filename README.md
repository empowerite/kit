# empowerite/kit: **k**eep **i**t **t**rustworthy

[![kit CI][ci-badge]][ci] [![release][release-badge]][releases] [![license: MIT][license-badge]](LICENSE)

[ci-badge]: https://github.com/empowerite/kit/actions/workflows/kit-ci.yml/badge.svg?event=pull_request
[ci]: https://github.com/empowerite/kit/actions/workflows/kit-ci.yml
[release-badge]: https://img.shields.io/github/v/release/empowerite/kit?include_prereleases&display_name=release
[releases]: https://github.com/empowerite/kit/releases
[license-badge]: https://img.shields.io/badge/license-MIT-blue

AI coding agents take the same shortcuts people do: they call work done when it is not, skip tests, and rewrite
other people's branches. kit writes down the rules an agent must follow, puts them where each agent reads, and checks
them in CI. It ships as a versioned release that any repository can install and keep up to date.

## What kit does

Checks:

- `kit ci` runs four checks and fails a pull request when any of them fails:
  - `kit gate`: the repository's own gate fails, or it has none.
  - `kit check`: a delivered file differs from the release, or a generated region differs from its command's output.
  - `kit linked`: the pull request is not linked to an open issue.
  - `kit policy`: the branch name, a commit subject, the pull request title, or a file in the tree breaks the
    naming policy.

  The delivered GitHub Actions job runs `kit ci`. Any other pipeline can run it too.
- `kit lint` runs each rule's linter over the files the rule covers. It also refuses a workflow action that is not
  pinned to a full commit SHA.

Everything else:

- `.kit/1st.md`, the file every agent reads first, and a rule file for each kind of file that has rules. Style, how a
  file looks, is optional: one style package per kind of file.
- `kit work` turns an issue into a worktree, a linked branch, and a draft pull request.
- `kit toolchains` lists the toolchains kit finds. A toolchain supplies the tools a repository's work runs with, such
  as its linters and the tools its gate runs. `kit tools` installs and updates the tools the chosen toolchain pins,
  along with the packages npm or uv pins, and runs commands with them. kit supports these toolchains; see
  [docs/toolchains.md](docs/toolchains.md):
  - [kit's own](docs/toolchains/using-kit.md), with `tools:` in `kit.yml`
  - [mise](docs/toolchains/using-mise.md)
  - [Nix](docs/toolchains/using-nix.md), with flakes
  - [Homebrew](docs/toolchains/using-brew.md), with a Brewfile
  - [a Docker image](docs/toolchains/using-docker.md)
  - [a dev container](docs/toolchains/using-devcontainer.md)
  - [apt, dnf, or pacman](docs/toolchains/using-system-packages.md)
  - [none](docs/toolchains/using-no-toolchain.md): the tools already on the machine
- `kit flow` shows where work stands in seven stages: issues, desk, worktrees, changes, builds, incoming, and
  releases.
- `kit workspace` shows what git holds across a workspace: remotes, branches, worktrees, commits, and tags.
- Adapters for Claude Code, Cursor, GitHub Copilot, and Gemini CLI put the rules where each one looks. The Claude
  Code and Gemini CLI adapters also deny raw git commands the rules forbid, such as `git rebase`. The deny list
  matches literal patterns, so `git -C <dir> commit` and `git merge` get through.
- A rule that agents never change the desk: the main checkout on your machine, whatever branch it is on.

Only CI, through `kit ci`, holds every agent to the rules. It runs on pull requests into the default branch, and it
blocks a merge only when a branch ruleset requires its check. kit does not ship a ruleset. An adapter's deny list
and hooks work only in that agent.

A repository keeps its own rules next to the ones kit delivers. The design is in
[#1](https://github.com/empowerite/kit/issues/1) and [#4](https://github.com/empowerite/kit/issues/4).

## Why a release, and not a copy

A copied file drifts. A release does not:

- It is pinned, so a repository knows which version of the rules applies.
- A pull request that bumps the version updates it, so every repository gets a change the same way.
- `kit check` fails the build when someone edits a delivered file in place.

Updates leave a repository's own files alone. To change a file kit adds, remove the package that adds it, or replace
the file with a package of your own. `.kit/1st.md` explains how, under "Changing what kit adds".

## Install

macOS and Linux, with [Homebrew](https://brew.sh):

```sh
brew install empowerite/tap/kit
```

Windows: kit runs in Windows Subsystem for Linux (WSL). Run `wsl --install` in PowerShell, install Homebrew inside
WSL, then run the line above.

Check it with `kit version` and `kit help`.

Other ways to install, such as mise or a download with `gh`, are in [docs/install.md](docs/install.md). That page
also lists what kit needs on the machine (bash 4.4 or later, git, `jq`, `yq`, and the linters), how kit picks which
release to run, and how `kit trust` works.

New to a repository that uses kit? Run `kit trust`, then `kit welcome`. `kit welcome` checks your tools, your code
forge logins, and the repository, and shows how to fix each problem. Then it explains how work flows, the terms, and
what a team can change. It also names terminal settings that make kittui slower or worse; see
[docs/terminals.md](docs/terminals.md). `kit tutorials` lists the tutorials and opens one; see
[docs/tutorials.md](docs/tutorials.md).

## Take the kit

1. Run `kit install` with the packages you want. With no names, it installs the `kit` package: the entry file and
   the rules. `kit packages remove kit` removes it once no other active package depends on it, and lists it under
   `excludes:` so `kit update` leaves it out. The optional packages are:

   | Package | Delivers |
   | --- | --- |
   | `workflow` | `.kit/workflow.md`: the workflow from an issue to a merged pull request, plans as prose, and the worker contract |
   | `github/ci` | the GitHub Actions workflow that runs `kit ci` |
   | `hooks/lefthook` | a pre-commit hook that runs `kit lint` and `kit generate --check` |
   | `agents/generic` | for any agent that reads `AGENTS.md`: a region there that sends it to `.kit/1st.md` |
   | `agents/claude` | the Claude Code adapter |
   | `agents/cursor` | the Cursor adapter: one rule under `.cursor/rules/` per kit rule, attached by that rule's paths |
   | `agents/copilot` | the GitHub Copilot adapter: one instruction file under `.github/instructions/` per kit rule |
   | `agents/gemini` | the Gemini CLI adapter: `AGENTS.md` among its context files, and the deny list, in `.gemini/settings.json` |
   | `policy/opa` | the naming policy as Open Policy Agent rules |
   | `policy/conventional` | commit subjects and pull request titles held to Conventional Commits, the issue in a `Refs:` trailer |
   | `style/<kind>` | the style for one kind of file, with its templates: `style/bash`, `style/markdown`, `style/just`, `style/workflows`, `style/yaml`, `style/json`, `style/gitignore` |
   | `style/prose` | the style of prose in every text file, documents and comments alike |
   | `empowerite` | the bundle Empowerite's repositories take: every `style/<kind>`, `github/ci`, `agents/claude`, `hooks/lefthook`, and `workflow` |

   For example, `kit install github/ci hooks/lefthook agents/claude`. A team on Cursor adds `agents/cursor`.
   `kit packages add` adds one later. `kit install empowerite` takes the bundle, and you can remove any member of
   it on its own.

   `kit install` adds `/worktrees/` to `.gitignore`. Every `agents/<harness>` package, through `agents/generic`,
   writes a short region into `AGENTS.md` that sends every agent to `.kit/1st.md`. Each region starts at a line
   with `kit:generate` and ends at a line with `kit:end`, and the rest of the file is yours. The `.gitignore` region
   can sit anywhere; the `AGENTS.md` and `.claude/CLAUDE.md` regions sit where their host template puts them.
   `kit generate` rewrites a region from its command, and `kit check` fails when one is out of date. Your own files
   can hold regions too; see [docs/generate.md](docs/generate.md).
2. Commit what `kit install` wrote. A person makes this commit by hand: `kit commit` refuses to run at the desk, and
   the Claude Code and Gemini CLI adapters deny an agent `git commit`. Or start an issue with `kit work`, which
   moves the uncommitted work into the new worktree, and run `kit commit` there.
3. Give the repository a gate, the command CI runs. kit takes the first of these it finds:
   - a `gate` command in `kit.yml`
   - a `ci` recipe in the root justfile
   - a `ci` target in a Makefile
   - a `ci` script in `package.json`

   `kit ci` fails when there is no gate. The gate runs with the tools your toolchain manages; see
   [docs/toolchains.md](docs/toolchains.md). The naming policy is in [docs/policy.md](docs/policy.md).
4. Run the linters before each commit. `kit lint` lints the staged files and needs no hook manager. To run it from
   lefthook instead:
   - pin `lefthook`,
   - add `extends: [.kit/lefthook.yml]` to `lefthook.yml`, and
   - run `kit hydrate && lefthook install` once per clone. lefthook silently skips a missing `.kit/lefthook.yml`,
     and `git commit --no-verify` skips the hook.

   With the `hooks/lefthook` package installed, `kit check` fails until `lefthook.yml` has that `extends` line. kit
   never writes it for you.

Claude Code reads `AGENTS.md` only when there is no `CLAUDE.md`. So the `agents/claude` package adds a region to
`.claude/CLAUDE.md` that imports `AGENTS.md`, and Claude reads what every other agent reads. The rest of
`.claude/CLAUDE.md` is yours.

`kit packages` lists the packages. `kit packages add` and `kit packages remove` change which are active. The package
format is in [docs/packages.md](docs/packages.md).

kit works with no packages at all. The workflow verbs (`kit flow`, `kit work`, `kit assign`, `kit unassign`,
`kit close`, `kit gather`, `kit commit`, `kit submit`, `kit rebase`, `kit settle`, `kit tidy`, and `kit release`)
and the checks that read `kit.yml` (`kit gate`, `kit policy`, and `kit check`) run without any. Kinds, rules, and
templates come from the `kit` package. Without it, `kit context` names no rule, `kit kinds` lists no kind,
`kit lint` covers no file, and `kit new file` refuses because there is no template.

Once kit has hydrated a repository, every kit call and every message `kit serve` answers is a line in a daily log
under `.kit/logs/`; see [docs/logs.md](docs/logs.md).

`kit completion zsh` prints zsh completion for the verbs, their flags, and their values; see
[docs/completion.md](docs/completion.md).

### Working with the rules

- Before an agent changes a file, it runs `kit context "${path}"` to find the rules that apply. The `agents/claude`
  package makes this call automatically through a Claude Code hook; see [docs/context.md](docs/context.md).
- `kit serve` offers the rules and the verbs to any agent over the Model Context Protocol (MCP), in three groups:
  - read only: `context`, `flow`, `help`, `kinds`, `packages`, `syntax`, and `version`
  - run what the repository's files name: `check`, `lint`, `toolchains`, and `tools`
  - act on issues and pull requests: `assign`, `close`, `gather`, `settle`, `unassign`, and `work`

  Each `agents/<harness>` package registers it with its agent; see [docs/serve.md](docs/serve.md).
- `kit new file "${path}"` creates a file from its style's template, and `kit lint` checks that every file keeps the
  shape of its template; see [docs/templates.md](docs/templates.md).

### Updating

Run the release you want to move to. It updates the repository and writes its version into `kit.yml`:

```sh
kit --version=0.2.0 update
```

For each file:

- A delivered file you never edited is replaced.
- A delivered file you edited is merged three ways, with the previous release as the base. `kit update` downloads
  the previous release from GitHub when it is not on the machine. Where it cannot, set `KIT_BASE` to a copy of it.
- Your own deny list entries and settings keys are kept.

`kit uninstall` removes everything kit placed. Add `--dry-run` to see what it would remove first. Your own files,
your own packages under `.kit/packages/`, and any delivered file you edited stay.

## Setup

The rest of this README is for working on this repository. It is kit's own source, and it also uses kit.

Tools come from the kit toolchain: `tools:` in `kit.yml` pins them, and direnv puts them on `PATH` through the
`.envrc`. Every commit must be signed; the branch ruleset rejects unsigned commits. If you have no signing key that
GitHub verifies, create and register one:

- [generating a new GPG key](https://docs.github.com/en/authentication/managing-commit-signature-verification/generating-a-new-gpg-key)
- [adding a GPG key to your GitHub account](https://docs.github.com/en/authentication/managing-commit-signature-verification/adding-a-gpg-key-to-your-github-account)

Then, once per clone, with the key's id in `signing_key`:

```sh
git config gpg.format openpgp
git config user.signingkey "${signing_key}"
git config commit.gpgsign true
brew bundle install
direnv allow
kit/bin/kit toolchains use kit
kit tools install
kit hydrate && lefthook install
kit welcome
```

`kit tools install` puts each tool `tools.lock` pins for this platform under `~/.kit/tools/`, shared by every clone
and worktree. On a desk it also installs lefthook, which `kit.yml` limits to desks. A runner, where `CI` is set,
skips it. See [using kit as the toolchain](docs/toolchains/using-kit.md).

## Getting started

Work here follows [.kit/workflow.md](.kit/workflow.md), from the `workflow` package:

1. Start from an issue. A change without an issue has not started.
2. Run `kit work <N>`. It creates `worktrees/<N>-<slug>`, links the branch to the issue under Development, and opens
   a draft pull request. It never switches the branch of the checkout you run it from. For an issue in another
   tracker, use that tracker's reference, such as `kit work AB#4521` for an Azure DevOps work item; see
   [docs/trackers.md](docs/trackers.md).
3. Work in that worktree. Commits are signed. Each commit subject starts with its issue's reference, as in
   `#<N> <subject>`, and the policy checks it.
   - `kit commit` commits what is staged, after it runs the linters, the policy, and a check that nothing kit placed
     is in the commit.
   - `kit submit` pushes the branch and commits nothing. `kit rebase` rebases it onto `main` and pushes it.
   - `kit flow` shows where work stands. `kit flow <stage>` prints one stage.
   - `kit` with no verb opens the welcome guide. `kit help` opens a list of the verbs and a form for the one you
     pick. A verb run without an argument it needs opens the same form.
4. Run the gate, `kit gate`. It runs `bin/ci`, which lists what it checks.
5. Mark the pull request ready for review when the work is done: `gh pr ready` on GitHub, `glab mr update --ready`
   on GitLab. The pull request body is the report.
6. A person merges it. Then `kit tidy` removes the worktree and its branch.

## Maintaining

| Directory | Contents |
| --- | --- |
| `kit/` | the release, packed as it is: everything in it ships. The tarball adds only a generated `version` file and `libexec/kittui`, built from `tui/` |
| `kit/bin/` | the kit CLI, `kit`, and its verbs |
| `kit/modules/` | the modules the verbs share: `bash/` and `jq/`, one directory per language |
| `kit/packages/` | everything a release delivers |
| `bin/` | this repository's own commands, not on `PATH`: its gate, `bin/ci`; `bin/refresh`; and `bin/release` |
| `docs/` | the documentation, with the packages' pages under `docs/packages/` |

[docs/architecture.md](docs/architecture.md) describes how the pieces fit, with diagrams.
[docs/packages.md](docs/packages.md) describes the package format.
[docs/modules.md](docs/modules.md) describes the bash and jq modules the verbs are built on, and how a repository
extends them.

This repository installs its own kit. What kit places under `.kit/` is a link into the package that delivers it. The
link is live and ignored by git, so editing through it edits the source. `bin/refresh` writes the installed copies
outside `.kit/` by running `kit update` from the index. Do not edit those by hand:

| Installed copy | Written from |
| --- | --- |
| `.kit/1st.md`, `.kit/README.md`, `.kit/kinds/`, `.kit/rules/`, `.kit/kit.schema.json` | links into `kit/packages/kit/files/repo/.kit/` |
| `.kit/styles/<kind>.md`, `.kit/templates/<kind>/`, `.kit/markdownlint.jsonc` | links into `kit/packages/style/<kind>/files/repo/.kit/` |
| `.kit/templates/bash/`, `.kit/templates/gitignore/file`, `.kit/templates/just/` | links into `.kit/packages/house/files/repo/.kit/templates/`, this repository's own style, which replaces the style packages' |
| `.kit/styles/house.md` | a link into `.kit/packages/house/files/repo/.kit/styles/`, this repository's own style beyond the style packages' |
| `.kit/lefthook.yml` | a link into `kit/packages/hooks/lefthook/files/repo/.kit/` |
| `.github/workflows/kit-ci.yml` | `kit/packages/github/ci/templates/repo/.github/workflows/kit-ci.yml`, with `ci_runner`, `ci_timeout_minutes`, and `ci_toolchain` filled in |
| `.claude/CLAUDE.md` | `kit/packages/agents/claude/regions/repo/.claude/CLAUDE.md`, a region; the rest of the file belongs to this repository |
| `.claude/settings.json` | `kit/packages/agents/claude/files/repo/.claude/settings.json`, merged by key; here only the kit's entries |
| `.mcp.json` | `kit/packages/agents/claude/files/repo/.mcp.json` and `.kit/packages/house/files/repo/.mcp.json`, merged by key |
| `.kit/policy/kit.rego`, `.kit/policy/house.rego` | links into `kit/packages/policy/opa/files/repo/.kit/policy/` and `.kit/packages/house/files/repo/.kit/policy/` |
| `AGENTS.md` | `kit/packages/agents/generic/regions/repo/AGENTS.md`, a region; the rest of the file belongs to this repository |
| `.gitignore` | `kit/packages/kit/regions/repo/.gitignore`, a region; the rest of the file belongs to this repository |
| `.kit/.gitignore`, `.kit/manifest` | written by kit beside what it placed under `.kit/`, and ignored like it: the paths git never sees, and a digest per delivered path. The release version (`unreleased` here) and the active packages are declared in `kit.yml` |

`kit.yml` belongs to this repository and is not delivered. kit writes only its `version`, `packages`, and `excludes`
keys. A `.kit/settings.yml` from an earlier release is moved into `kit.yml` whole, once. A fresh clone or worktree has
nothing else under `.kit/` until a kit verb runs, or until `cd` hydrates it through the `.envrc`.

When a commit changes a file under `kit/` or `.kit/packages/`, or changes `kit.yml`, the installed copies must change
in the same commit:

- With the pre-commit hook installed, this happens on its own: the hook runs `bin/refresh`, which regenerates the
  copies from the index and stages them.
- Without the hook, stage your change, run `bin/refresh`, and commit.

`kit/bin/kit check`, which the gate runs, names any installed copy that does not match its source. Every delivered
file except a template says in its header that kit delivered it. A region's `kit:generate` line is its notice, and
the table above is the notice for `settings.json`.

### The test suite

`kit gate` runs the suite the way `bin/ci` does: bats, one job per CPU, on GNU parallel from `kit.yml`. To run one
file, use `kit tools run bats tests/<file>.bats`, which puts the pinned tools first on `PATH`.

Most of the suite's time goes to starting short processes: bash, jq, yq, and git. macOS starts processes more slowly
than Linux, so a test takes 1.5 to 2.4 times as long on a Mac as on a runner. On a Mac, list the terminal and the
editor you run tests from as developer tools; that saves up to about 5 percent of a run.
[docs/testing.md](docs/testing.md) has the numbers, the time limits, and the steps.

## Release

- A release is the tag `release/<version>` on any commit on `main`, not only the latest one.
- A pre-release is `release/<version>-<suffix>`, such as `release/0.1.0-rc.1`, on any commit, on `main` or off it.
  It is published as a GitHub pre-release.
- The tag is the version. The repository has no version file. The release workflow writes the part after `release/`
  into the packed kit and names the release after it.
- Published releases cannot change, so a tag is never reused.

`kit release <version>` checks these rules, then makes the tag and pushes only the tag. One more rule: a version that
is not higher than every release on the primary remote needs `--lower`. `kit flow releases` lists what the code forge
published.

The release workflow runs two commands:

1. `bin/release verify` rejects a tag that is not `release/<version>`, and a release tag on a commit not on `main`.
2. `bin/release publish` packs `kit/` (`bin/`, `modules/`, and `packages/`), `version`, and `libexec/` into
   `kit-<version>.tar.gz` and publishes it. `libexec/` holds `kittui`, built from `projects/tui/` for Linux and
   macOS on amd64 and arm64.

`bin/release pack` builds the same tarball locally. `bin/release publish` refuses to publish a release anywhere but
the CI runner, so only the workflow publishes releases. A pre-release can be published from a developer machine.

A release is permanent. A pre-release is deleted, tag and all, once no branch contains its commit, which happens when
the branch it was tagged on is deleted. A pre-release on `main` stays.

The branch ruleset is managed in [empowerite/infrastructure](https://github.com/empowerite/infrastructure).
