# empowerite/kit: **k**eep **i**t **t**rustworthy

[![kit CI][ci-badge]][ci] [![release][release-badge]][releases] [![license: MIT][license-badge]](LICENSE)

[ci-badge]: https://github.com/empowerite/kit/actions/workflows/kit-ci.yml/badge.svg?event=pull_request
[ci]: https://github.com/empowerite/kit/actions/workflows/kit-ci.yml
[release-badge]: https://img.shields.io/github/v/release/empowerite/kit?include_prereleases&display_name=release
[releases]: https://github.com/empowerite/kit/releases
[license-badge]: https://img.shields.io/badge/license-MIT-blue

Do you trust your agents? They were trained on the whole of what people have written, and that includes every shortcut
anyone ever took: the "done" that was not, the test that was skipped and reported as passing, the rebase that quietly
rewrote a colleague's branch. An agent is a tireless new colleague who has read everything and holds none of your house
rules, unless you put them where it looks. kit is the house rules, put where agents look, and the checks that hold
agents to them, as a versioned release any repository can take and keep taking.

What kit checks:

- `kit ci` runs four checks and refuses a pull request when any fails:
  - `kit gate`: the repository's own gate fails, or it has none
  - `kit check`: a delivered file differs from the release, or a generated region from its command's output
  - `kit linked`: the pull request is not linked to an open issue
  - `kit policy`: the branch name, a commit subject, the pull request title, or a file in the tree breaks the
    policy

  The delivered GitHub Actions job runs it. Any other pipeline can.
- `kit lint` runs each rule's linter, and the kit's own checks, over the files the rule governs. One of those checks
  refuses a workflow action not pinned to a full commit SHA.

What else kit provides:

- `.kit/1st.md`, the file every agent reads first, and a rule file for each kind of file that has one. Style, how a
  file looks, is optional: one style package per kind of file.
- `kit work`, which turns an issue into a worktree, a linked branch, and a draft pull request.
- `kit toolchains`, which lists the toolchains kit finds where it runs, and `kit tools`, which installs and
  updates the tools the chosen one pins, and the packages npm or uv pins on top, and runs commands with them.
- `kit flow`, which draws where the work stands in eight stages: backlog, underway, desk, worktrees,
  changes, builds, incoming, releases.
- `kit workspace`, which draws what git holds across a workspace: its remotes, branches, worktrees, commits, and tags.
- Adapters for Claude Code, Cursor, GitHub Copilot, and Gemini CLI, which put the rules where each harness looks.
  The Claude Code and Gemini CLI adapters also add a deny list that refuses the raw git commands the rules forbid,
  such as `git rebase`, by literal pattern. `git -C <dir> commit` and `git merge` pass it.
- A rule that agents never change the desk: the root checkout on your machine, on whatever branch it has.

CI, through `kit ci`, is the only enforcement every harness feels. It runs on pull requests into the default
branch. It blocks a merge only when a branch ruleset requires its check, and kit does not ship a ruleset. An
adapter's deny list and hooks stop an agent only in that harness.

A repository keeps its own rules next to the delivered ones. The design is in
[#1](https://github.com/empowerite/kit/issues/1) and [#4](https://github.com/empowerite/kit/issues/4).

## Why a release, and not a copy

A copied file drifts. A release does not:

- It is pinned, so a repository knows which version of the rules applies.
- It is updated by a pull request that bumps the version, so every repository gets the change the same way.
- It is checked: `kit check` fails the build when someone edits a delivered file in place.

A repository's own rules live in its own files, and updates leave them alone. To change a file the kit adds, remove
the package that adds it, or replace the file through a package of your own. `.kit/1st.md` says how, under
"Changing what kit adds".

## Install

A release is a tarball: `bin/`, `modules/`, `packages/`, `libexec/` with the loader's call and the `kittui` binaries
`kit tui` runs, and a `version` file. Install any one release on your machine, the way you install your other
tools. The `kit` you install is a loader: it runs the release each repository asks for, so one installed kit serves
every repository, whatever release each one pins.

The first of these chooses the release a call runs:

1. `kit --version=<release> …`, for this call and every kit call it makes in the same clone.
2. `KIT_VERSION=<release>`, the same thing, from the environment.
3. `version:` in the `kit.yml` at the root of the clone you are in. Each worktree is its own root.
4. The kit you installed, outside a clone, and in a clone that pins none.

A release is named by its version, `0.1.0`; `--version=` and `KIT_VERSION` also take the path of a copy of one. The
loader keeps every release it runs in `~/.cache/kit/`, and downloads a version the first time a repository asks for
it, checked against the `.sha256` the release publishes beside it. A kit option it does not know is refused: a later
kit may know it, so update the kit you installed.

kit runs nothing in a clone until you trust it. `kit trust` shows what the clone asks kit to run, its pinned release,
the bash it loads, and its gate, then asks you, and remembers your answer for the clone and all its worktrees. On a
terminal, the first kit command in a clone you have not trusted asks the same question, then runs after a yes.
`kit trust ~/Code/` trusts every clone under that folder. Trusting a clone means you accept code from whoever can
push to it: kit does not check each pull; review and signed commits do. The records are plain lines in
`~/.config/kit/trusted`, and `kit trust --list` and `kit trust --revoke` read and change them.

With Homebrew:

```sh
brew install empowerite/tap/kit
```

With mise, as a tool in your own `~/.config/mise/config.toml`:

```toml
[tools]
"github:empowerite/kit" = { version = "0.1.0", version_prefix = "release/" }
```

By hand, with `gh`. This downloads the latest release, never a pre-release; for one version, run
`gh release download "release/${version}" ...` instead. Unpack it and put its `bin/` on `PATH`:

```sh
gh release download --repo empowerite/kit --pattern 'kit-*.tar.gz' --output kit.tar.gz
mkdir kit && tar --extract --gzip --file kit.tar.gz --directory kit --strip-components 1
export PATH="${PWD}/kit/bin:${PATH}"
```

Requirements:

- bash 4.4 or later. macOS ships bash 3.2; run `brew install bash`.
- git.
- `jq` 1.7 or later: every verb reads its arguments through it.
- `yq` 4, mikefarah's: it reads `kit.yml` and every other YAML file. GitHub's hosted runners have `jq` and `yq`
  preinstalled.
- `curl` and `tar`: the loader fetches a pinned release through them, and `kit update` the previous release, as the
  base of a three-way merge, when that release is not in the cache already. Each checks the bytes against the
  `.sha256` published beside the release before reading anything in it.
- The linters the delivered rules and styles run. `kit lint` fails, not skips, when a linter it needs is missing,
  so a team's first commit needs them:
  - `shellcheck`, which the bash rule runs
  - `actionlint`, which the workflows rule runs
  - `markdownlint-cli2`, which the markdown style runs, with the `style/markdown` package
- For `kit work`, `kit assign`, `kit unassign`, `kit close`, `kit gather`, `kit settle`, `kit flow`, `kit tidy`,
  `kit linked`, and `kit release`: the code forge's command-line tool, `gh` for GitHub or `glab` for GitLab. The
  code forge is the service that hosts the repository. `kit commit` uses it too, to close an issue its subject
  names, and `kit policy` uses it in CI to read the pull request.
- For an Azure DevOps tracker: `az`, the Azure CLI.

kit runs on Linux and macOS. On Windows, use WSL, which runs it as Linux.

Then check the install: `kit version` prints the release, and `kit help` lists the verbs. Once a repository has
taken the kit, `kit check` says every delivered file matches the release.

New to a repository that uses kit? Run `kit trust`, then `kit welcome`. It checks your tools, your code forge logins,
and the repository, and shows the fix for each problem. Then it walks you through how work flows, the terms, and
what a team can change.

`kit tutorials` lists kit's tutorials, by who each is for, and opens one, in kittui or the browser; see
[docs/tutorials.md](docs/tutorials.md).

## Take the kit

1. Run `kit install` with the packages you want. The `kit` package, the entry file and the rules, is the default:
   `kit install` with no names brings it. `kit packages remove kit` takes it out once no other active package
   depends on it, and lists it under `excludes:`, so `kit update` leaves it out. The optional packages are:

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
   | `style/<kind>` | the style for one kind of file, with its templates: `style/bash`, `style/markdown`, `style/just`, `style/workflows`, `style/yaml`, `style/json`, `style/gitignore` |
   | `style/prose` | the style of prose in every text file, documents and comments alike |
   | `empowerite` | the bundle Empowerite's repositories take: every `style/<kind>`, `github/ci`, `agents/claude`, `hooks/lefthook`, and `workflow` |

   `kit install github/ci hooks/lefthook agents/claude` takes three of them; a team on Cursor adds `agents/cursor`.
   `kit packages add` adds one later. `kit install empowerite` takes the bundle, and a member of it can be removed
   on its own.

   `kit install` adds `/worktrees/` to `.gitignore`. Every `agents/<harness>` package, through `agents/generic`,
   writes a heading and two sentences into `AGENTS.md` that send every agent to `.kit/1st.md`. Each is a region: it
   runs from a `kit:generate` line to a `kit:end` line, and the rest of the file is yours. The `.gitignore` region
   can sit anywhere; the `AGENTS.md` and `.claude/CLAUDE.md` regions sit where their host template puts them.
   `kit generate` rewrites a region from its command, and `kit check` fails when one is stale. Your own files can
   hold regions too; see [docs/generate.md](docs/generate.md).
2. Commit what `install` wrote. A person makes this commit by hand: `kit commit` refuses to run at the desk, and
   the Claude Code and Gemini CLI adapters deny an agent `git commit`. Or start an issue with `kit work`, which
   sweeps the uncommitted work into the new worktree, and run `kit commit` there.
3. Give the repository a gate, the command that CI runs. kit takes the first of these:
   - any command, set as `gate` in `kit.yml`
   - a `ci` recipe in the root justfile
   - a `ci` target in a Makefile
   - a `ci` script in `package.json`

   `kit ci` fails when there is no gate. The gate runs with the tools your toolchain manages; see
   [docs/toolchains.md](docs/toolchains.md). The naming policy is in [docs/policy.md](docs/policy.md).
4. Run the linters before each commit. `kit lint` lints the staged files and does not need a hook manager. To run it
   from lefthook instead:
   - pin `lefthook`,
   - add `extends: [.kit/lefthook.yml]` to `lefthook.yml`, and
   - run `kit hydrate && lefthook install` once per clone; lefthook skips a missing `.kit/lefthook.yml` without a
     word, and `git commit --no-verify` skips the hook.

   With the `hooks/lefthook` package installed, `kit check` fails until `lefthook.yml` has that `extends` line. kit
   never writes it for you.

Claude Code reads `AGENTS.md` only when there is no `CLAUDE.md`. So the `agents/claude` package adds a region to
`.claude/CLAUDE.md` that imports `AGENTS.md`, and Claude reads what every other agent reads. The rest of
`.claude/CLAUDE.md` is yours.

`kit packages` lists the packages, and `kit packages add` and `kit packages remove` change which are active. The
package format is in [docs/packages.md](docs/packages.md).

kit works with no package at all. The workflow verbs (`kit flow`, `kit work`, `kit assign`, `kit unassign`,
`kit close`, `kit gather`, `kit commit`, `kit submit`, `kit rebase`, `kit settle`, `kit tidy`, `kit release`)
and the checks that read `kit.yml` (`kit gate`, `kit policy`, `kit check`) run in a repository that has none.
What needs the `kit` package is its kinds, rules, and templates: without them, `kit context` does not name any
rule, `kit kinds` does not list any kind, `kit lint` does not govern any file, and `kit new file` refuses, because
there is no template.

In a repository kit has hydrated, every kit call, and every message `kit serve` answers, is a line in a daily log
under `.kit/logs/`; see [docs/logs.md](docs/logs.md).

`kit completion zsh` prints the zsh completion, which completes the verbs, their flags, and the values they take,
read from the verbs themselves; see [docs/completion.md](docs/completion.md).

### Working with the rules

- Before an agent changes a file, it runs `kit context "${path}"` to find the rules that apply. The `agents/claude`
  package makes this call automatically through a Claude Code hook; see [docs/context.md](docs/context.md).
- `kit serve` offers the rules, and the verbs in three groups, to any agent over the Model Context Protocol:
  `context`, `flow`, `help`, `kinds`, `packages`, `syntax`, and `version` read only; `check`, `lint`, `toolchains`,
  and `tools` run what the repository's files name; and `assign`, `close`, `gather`, `settle`, `unassign`, and
  `work` act on issues and pull requests. Each `agents/<harness>` package registers it with its harness; see
  [docs/serve.md](docs/serve.md).
- `kit new file "${path}"` creates a file from its style's template, and `kit lint` checks that every file keeps the
  shape of a template; see [docs/templates.md](docs/templates.md).

### Updating

Run the release to move to, and it moves the repository to itself and writes its version into `kit.yml`:

```sh
kit --version=0.2.0 update
```

What it does to each file:

- A delivered file you never edited is replaced.
- A delivered file you edited is merged three ways, with the previous release as the base. `kit update` downloads
  the previous release from GitHub when it is not on the machine. Where it cannot, set `KIT_BASE` to a copy of it.
- Your own deny list entries and settings keys are kept.

`kit uninstall` takes everything the kit placed out again, with `--dry-run` to see what would go first. Your own files,
your own packages under `.kit/packages/`, and any delivered copy you edited stay.

## Setup

The rest of this README is about working on this repository, which is the kit's own source and also uses the kit.

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
kit/bin/kit toolchains choose kit
kit tools install
kit hydrate && lefthook install
kit welcome
```

`kit tools install` puts each tool `tools.lock` pins for this platform under `~/.kit/tools/`, shared by every clone
and worktree. On a desk it installs lefthook too, which `kit.yml` keeps to a desk; a runner, where `CI` is set, leaves
it out. See [using kit as the toolchain](docs/toolchains/using-kit.md).

## Getting started

Work here follows the workflow in [.kit/workflow.md](.kit/workflow.md), from the `workflow` package:

1. Start from an issue. A change without an issue has not started.
2. Run `kit work <N>`. It creates `worktrees/<N>-<slug>`, links the branch to the issue under Development, and opens
   a draft pull request. It never switches the branch of the checkout you run it from. For an issue in another
   tracker, use that tracker's reference, such as `kit work AB#4521` for an Azure DevOps work item; see
   [docs/trackers.md](docs/trackers.md).
3. Work in that worktree. Commits are signed. A commit's subject starts with its issue's reference, as in
   `#<N> <subject>`, and the policy requires it.
   - `kit commit` commits what is staged, after the linters, the policy, and a check that nothing kit placed is in it.
   - `kit submit` pushes the branch, and commits nothing; `kit rebase` rebases it onto `main` and pushes it.
   - `kit flow` draws where the work stands in eight stages: backlog, underway, desk, worktrees,
     changes, builds, incoming, releases. `kit flow <stage>` prints one.
   - `kit`, with no verb, opens the welcome guide. `kit help` opens a palette of the verbs and a form for the one you
     pick. A verb run without an argument it needs opens the same form.
4. Run the gate, `kit gate`. It runs `bin/ci`, which lists what it checks.
5. Mark the pull request ready for review when the work is done: `gh pr ready` on GitHub, `glab mr update --ready`
   on GitLab. The pull request body is the report.
6. A human merges. Then `kit tidy` removes the worktree and its branch.

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
[docs/modules.md](docs/modules.md) describes the modules the verbs are built on, in bash and jq, and how a
repository extends them.

This repository installs its own kit. What kit places under `.kit/` is a link into the package that delivers it,
live and ignored by git: an edit through the link is an edit of the source. The installed copies outside `.kit/`
are written by `bin/refresh`, which runs `kit update` from the index. Do not edit those by hand:

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

`kit.yml` belongs to this repository and is not delivered; kit writes only its `version`, `packages`, and
`excludes` keys there. A `.kit/settings.yml` from an earlier release is moved into `kit.yml` whole, once. A fresh
clone or worktree has nothing else under `.kit/` until a kit verb runs, or `cd` hydrates it through the `.envrc`.

When a commit changes a file under `kit/` or `.kit/packages/`, or `kit.yml`, the installed copies must change
in the same commit:

- With the pre-commit hook installed, this happens automatically: the hook runs `bin/refresh`, which regenerates
  the copies from the index and stages them.
- Without the hook, stage your change, run `bin/refresh`, and commit.

`kit/bin/kit check`, which the gate runs, names any installed copy that does not match its source. Each delivered file
but a template says in its header that kit delivered it. A region's `kit:generate` line serves as its notice, and
the table above serves as the notice for `settings.json`.

### The test suite

`kit gate` runs the suite as `bin/ci` does: bats, one job per CPU, on GNU parallel from `kit.yml`. To run one
file, use `kit tools run bats tests/<file>.bats`, which puts the pinned tools first on `PATH`.

Nearly all of the suite's time goes to starting short processes: bash, jq, yq, and git. macOS starts a process more
slowly than Linux, so a test takes 1.5 to 2.4 times as long at a Mac desk as on a runner. On a Mac, list the terminal
and the editor you run the tests from as developer tools; that saves up to about 5 percent of a run.
[docs/testing.md](docs/testing.md) has the numbers, the time limits, and the steps.

## Release

- A release is the tag `release/<version>` on any commit on `main`, not only the latest one.
- A pre-release is `release/<version>-<suffix>`, such as `release/0.1.0-rc.1`, on any commit, on `main` or off it. It
  is published as a GitHub pre-release.
- The tag is the version. The repository has no version file. The release workflow writes the part after `release/`
  into the packed kit and names the release after it.
- Published releases are immutable, so a tag is never reused.

`kit release <version>` makes the tag and pushes it alone, after these rules and one more: a version not higher
than every release on the primary remote needs `--lower`. `kit flow releases` lists what the code forge published.

The release workflow runs two commands:

1. `bin/release verify` rejects a tag that is not `release/<version>`, and a release tag on a commit not on `main`.
2. `bin/release publish` packs `kit/` (`bin/`, `modules/`, and `packages/`), `version`, and `libexec/`, which holds
   `kittui` built from `projects/tui/` for linux and darwin on amd64 and arm64, into `kit-<version>.tar.gz` and
   publishes it.

`bin/release pack` builds the same tarball locally. `bin/release publish` refuses to publish a release anywhere but
the CI runner, so only the workflow publishes releases. A pre-release can be published from a developer machine.

A release is permanent. A pre-release is deleted, tag and all, once no branch contains its commit, which is what
deleting the branch it was tagged on does. A pre-release on `main` stays.

The branch ruleset is managed in [empowerite/infrastructure](https://github.com/empowerite/infrastructure).
