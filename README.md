# empowerite/kit: **k**eep **i**t **t**rustworthy

Do you trust your agents? They were trained on the whole of what people have written, and that includes every
shortcut anyone ever took: the "done" that was not, the test that was skipped and reported as passing, the rebase
that quietly rewrote a colleague's branch. An agent is a tireless new colleague who has read everything and holds
none of your house rules, unless you put them where it looks.

kit is the house rules, put where agents look, and the checks that hold agents to them, as a versioned release any
repository can take and keep taking. The rules are the smaller part. The point is verification that does not depend on
anyone's word: a delivered CI job that refuses a pull request without a linked open issue, `kit check` holding every
delivered file to its release, a pre-commit hook that runs the linters over what is staged, a deny list that refuses
the spellings the rules forbid, and `kit status` showing what every pull request's checks said. The text is what those
checks hold the work against: the entry file an agent reads first, a rule file per kind of file, the root checkout
kept as the human's desk, and the one command no harness provides, which turns an issue into a worktree with a draft
pull request. A tag on `main` publishes a release. A repository pins it, installs it, and keeps rules of its own
beside it. The design is [#1](https://github.com/empowerite/kit/issues/1), and the entry file every agent reads is
[.kit/1st.md](.kit/1st.md).

## Why a release, and not a copy

A copy drifts. Six repositories with six copies of the same rules have six sets of rules within a month, and no
one can say which is current. A release is pinned, so a repository knows which rules bind it; updated, so a
change to the rules reaches every repository as a pull request that bumps a version; and checked, so `kit check`
fails the build when a delivered file was edited in place. The rules a repository adds live beside the delivered
ones, in its own files, and survive every update.

## Install

The `kit` command is a release: a tarball holding `bin/`, `tree/`, and `targets/`. Take it whichever way the
repository takes its other tools.

With mise, pin it in `mise.toml` and the version is part of the repository:

```toml
[tools]
"github:empowerite/kit" = "0.1.0"
```

With Homebrew, from the empowerite tap:

```sh
brew install empowerite/tap/kit
```

Or by hand, with `gh`, which the kit needs anyway. Without a tag this takes the latest release, never a
pre-release; `gh release download "v${version}" ...` takes a particular one. Unpack it and put its `bin/` on `PATH`:

```sh
gh release download --repo empowerite/kit --pattern 'kit-*.tar.gz' --output kit.tar.gz
mkdir kit && tar --extract --gzip --file kit.tar.gz --directory kit --strip-components 1
export PATH="${PWD}/kit/bin:${PATH}"
```

The kit needs bash 4.4 or later, git, `gh`, and `jq`. macOS ships bash 3.2, and `brew install bash` gives a
current one. With mise, an exact pin installs the minute a release exists; a range pin sees a new kit a day after
it ships unless `[settings]` carries `minimum_release_age_excludes = ["github:empowerite/kit"]`.

## Take the kit

In the repository, install it and point `AGENTS.md` at it, in prose, the same sentence for every agent:

```sh
kit install claude
```

```markdown
Read .kit/1st.md at the start of every session. It defines how any agent uses empowerite/kit to keep this
repository trustworthy.
```

The rest of `AGENTS.md` is yours. Everything the kit delivers is a module, and `install` activates the default ones,
`kit`, `lefthook`, and `ci`, plus those named: `claude` is the adapter for Claude Code, which reads `AGENTS.md` only
as a fallback, when no `CLAUDE.md` is present and its configuration allows, so the module places `.claude/CLAUDE.md`
importing `AGENTS.md` and Claude reads what everyone else reads in every case. A repository with no adapter still
binds every agent that reads `AGENTS.md`. Commit what it wrote, keep a `ci` recipe in the root justfile for the
delivered `CI` job to run, and take the pre-commit linters by pinning `lefthook`, putting `extends:
[.kit/lefthook.yml]` in `lefthook.yml`, and running `lefthook install` once per clone. `kit module list`, `kit module
add`, and `kit module remove` show and change the set later, and [docs/modules.md](docs/modules.md) describes a
module. An agent asks `kit context <path>` which rules govern a file before it creates or changes it;
[docs/context.md](docs/context.md) describes the call and the Claude Code hook that makes it for the agent.

To move to a new release, bump the pin or reinstall, then run `kit update`: a delivered file you left alone is
replaced, one you edited is three-way merged with the previous release as the base, which `update` fetches from
GitHub if it is not already on the machine, and your own deny entries and settings keys stay. `kit check` fails,
locally and in `CI`, when a delivered file differs from the pinned release.

## Setup

This repository uses its own kit, and its tools come from mise. Every commit is signed, and the branch ruleset
refuses one that is not. If you have no signing key GitHub verifies, make one and register it, following
[generating a new GPG key](https://docs.github.com/en/authentication/managing-commit-signature-verification/generating-a-new-gpg-key)
and [adding a GPG key to your GitHub account](https://docs.github.com/en/authentication/managing-commit-signature-verification/adding-a-gpg-key-to-your-github-account).
Then, once per clone, with the key's id in `signing_key`:

```sh
git config gpg.format openpgp
git config user.signingkey "${signing_key}"
git config commit.gpgsign true
export MISE_ENV=desk
mise install
lefthook install
```

`MISE_ENV=desk` belongs in your shell profile: it makes mise read `mise.desk.toml`, the tools only a desk runs,
on top of `mise.toml`; the delivered `CI` job sets `MISE_ENV=ci` the same way for `mise.ci.toml`.

## Getting started

Work here moves by the loop in [.kit/1st.md](.kit/1st.md):

- An issue exists. A change without an issue has not started.
- `kit issue <N>` makes `worktrees/<N>-<slug>`, links the branch to the issue under Development, and opens a
  draft pull request. It never switches the checkout it runs from.
- Work happens in that worktree. Commits are signed, and their subject is `#<N> <area>: <subject>`. `kit rebase`
  brings the branch up to date with `main` and pushes it, and `kit status` shows every pull request in flight
  and the worktree it is written in.
- `just ci` is the gate; the justfile says what it runs.
- `gh pr ready` says the work is done, and the pull request body is the report.
- A human merges. `kit tidy` then removes the worktree and its branch.

## Maintaining

This repository is both the source of the kit and a consumer of it. The release is the checkout: `modules/<name>/`
is what a consumer activating that module gets, its `files/repo/` mirrored at the consumer's root and its
`templates/repo/` rendered from the consumer's answers first; `bin/` is the machinery, `bin/kit` dispatching to one executable
`bin/kit-<verb>.sh` per verb, each sourcing the bash modules beside it; and `docs/` with each module's `docs/` is
the documentation. [docs/modules.md](docs/modules.md) is the module format.

[docs/architecture.md](docs/architecture.md) is the shape of the whole, with its figures.

The installed copies are written by `bin/kit install` and not edited by hand:

| Installed copy | Written from |
| --- | --- |
| `.kit/1st.md`, `.kit/rules/` | `modules/kit/files/repo/.kit/` |
| `.kit/lefthook.yml` | `modules/lefthook/files/repo/.kit/lefthook.yml` |
| `.github/workflows/kit-ci.yml` | `modules/ci/templates/repo/.github/workflows/kit-ci.yml`, with `ci_runner` answered |
| `.claude/settings.json` | `modules/claude/files/repo/.claude/settings.json`, merged by key; here only the kit's entries |
| `.kit/manifest` | the release version (`unreleased` here), the active modules, and a digest per delivered path |

A commit that changes a file under `modules/` rewrites the installed copies in the same commit: the
pre-commit hook runs `just refresh`, which regenerates them from the index and stages the result. On a desk
without the hook, stage the change, run `just refresh`, and commit. `bin/kit check`, which `just ci` runs, names
any installed copy that no longer matches its source. Each delivered text file says in its header that it is
delivered by kit; `settings.json` cannot, so this table is its notice.

## Release

A release is the tag `v<version>` on a commit on `main`, not only its head. A pre-release is `v<version>-<suffix>`,
say `v0.1.0-rc.1`, on a commit that is not on `main`, and it is published as a GitHub pre-release. The tag is the
version: the repository carries no version file, the workflow stamps the tag's version into the packed kit, and a
published release is immutable, so a tag is never reused. The workflow packs `bin/`, `tree/`, and `targets/` into
`kit-<version>.tar.gz` and publishes the release; it refuses a tag that is not a version, a release tag off
`main`, and a pre-release tag on `main`. A release is permanent. A pre-release is retired, tag and all, when the
branch it was tagged on is deleted and no other branch reaches its commit.

Repository policy is managed by Terraform; see: empowerite/infrastructure/terraform/projects/kit/main.tf
