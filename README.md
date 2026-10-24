# kit

kit is the agentic guardrails a repository takes as a versioned mixin: the policy text an agent reads first, the
rules that bind each kind of file, the guard that keeps `main` as the human's desk, and the one command no harness
provides, the issue-to-worktree on-ramp. A tag on `main` publishes a release. A consumer pins it in `mise.toml`,
installs it with `kit install`, and keeps rules of its own beside it. The design is
[#1](https://github.com/empowerite/kit/issues/1), and the entry file every agent reads is [.kit/1st.md](.kit/1st.md).

## Take the kit

Pin a release in the repository's `mise.toml`, then install it, naming the harnesses the team uses:

```toml
[tools]
"github:empowerite/kit" = "0.1.0"
```

```sh
mise install
kit install claude
```

`kit install` copies `.kit/1st.md`, `.kit/rules/`, and `.kit/lefthook.yml` into the repository, places
`.github/workflows/kit-ci.yml`, and writes `.kit/manifest`. Each target named adds what that harness insists on
having outside `.kit/`; `claude` places `.claude/rules/kit-*.md` and merges the kit's deny rules into
`.claude/settings.json`. A repository with no target installed still binds every agent that reads `AGENTS.md`,
through the rules table in `.kit/1st.md`. `kit target add` and `kit target remove` change the set later, and
`kit target list` prints it. Commit what it wrote. Then:

- Open `AGENTS.md` with `@.kit/1st.md` and the sentence "Read .kit/1st.md first." The rest of the file is yours.
- Keep a `ci` recipe in the root justfile. The delivered `CI` job runs it, then `kit check`, then refuses a pull
  request not linked to an open issue.
- Pin `lefthook` in `mise.desk.toml`, which mise reads when `MISE_ENV=desk`, put `extends: [.kit/lefthook.yml]`
  in `lefthook.yml`, and run `lefthook install` once per clone. The fragment says which linter runs over which
  paths before a commit, and your `lefthook.yml` adds commands of its own. `kit lint` runs the same hook by hand.
  The delivered `CI` job sets `MISE_ENV=ci`, so a tool only the runner needs is pinned in `mise.ci.toml`.

The machinery, the `kit` command and the bash behind its verbs, stays in the release where mise installed it; the
repository carries the text an agent reads and the record of what it took. The kit needs bash 4.4 or later, git,
`gh`, and `jq`.

To move to a new release, bump the pin, run `mise install`, then `kit update`: a delivered file you left alone is
replaced, one you edited is three-way merged with the previous release as the base, and your own deny entries
and settings keys stay. `kit check` fails, locally and in `CI`, when a delivered file differs from the pinned
release.

macOS ships bash 3.2; `brew install bash` gives a current one.

## Setup

Every commit is signed, and the branch ruleset refuses one that is not. If you have no signing key GitHub
verifies, make one and register it, following [generating a new GPG key](https://docs.github.com/en/authentication/managing-commit-signature-verification/generating-a-new-gpg-key)
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

`MISE_ENV=desk` belongs in your shell profile. mise reads `mise.toml` always, `mise.desk.toml` on top of it when
`MISE_ENV=desk`, and `mise.ci.toml` when `MISE_ENV=ci`, which the CI workflow sets: a tool goes in the file of
whoever runs it. Lefthook is a desk's; actionlint is the runner's.

This repository carries the installed copy of its own kit, and `just ci` checks it.

## Getting started

Work here moves by the loop in [.kit/1st.md](.kit/1st.md):

- An issue exists. A change without an issue has not started.
- `kit issue <N>` makes `worktrees/<N>-<slug>`, links the branch to the issue under Development, and opens a
  draft pull request. It never switches the checkout it runs from.
- Work happens in that worktree. Commits are signed, and their subject is `#<N> <area>: <subject>`. `kit rebase`
  brings the branch up to date with `main` and pushes it.
- `just ci` is the gate: shellcheck over the bash, actionlint over the workflows, the bats suite in `tests/`, and
  `bin/kit check`.
- `gh pr ready` says the work is done, and the pull request body is the report.
- A human merges. `kit tidy` then removes the worktree and its branch.

## Maintaining

This repository is both the source of the kit and a consumer of it. The release is the checkout: `tree/` is what
every consumer gets, mirrored at its root; `targets/<name>/` is what a consumer taking that target gets, mirrored
the same way; `bin/` is the machinery: `bin/kit` dispatches to one executable `bin/kit-<verb>.sh`
per verb, and the verbs source the modules beside them, `1st.sh`, `gh.sh`, `git.sh`, and `release.sh`. A symlink
under `targets/` is delivered as the file it points at, so a rule lives once, in `tree/.kit/rules/`.

The installed copies are written by `bin/kit install` and not edited by hand:

| Installed copy | Written from |
| --- | --- |
| `.kit/1st.md`, `.kit/rules/`, `.kit/lefthook.yml` | `tree/.kit/` |
| `.github/workflows/kit-ci.yml` | `tree/.github/workflows/kit-ci.yml` |
| `.claude/rules/kit-<kind>.md` | `targets/claude/.claude/rules/`, links into `tree/.kit/rules/` |
| `.claude/settings.json` | `targets/claude/.claude/settings.json`, merged by key; here only the kit's entries |
| `.kit/manifest` | the release version (`unreleased` here), the targets taken, and a digest per delivered path |

After changing a file under `tree/` or `targets/`, run `just refresh`, which reinstalls the kit into this
repository, and commit the result with the change. `bin/kit check`, which `just ci` runs, names any installed copy
that no longer matches its source. Each delivered text file says in its header that it is delivered by kit;
`settings.json` cannot, so this table is its notice.

## Release

A release is the tag `v<version>` on a commit on `main`, not only its head. A pre-release is `v<version>-<suffix>`,
say `v0.1.0-rc.1`, on a commit that is not on `main`, and it is published as a GitHub pre-release. The tag is the
version: the repository carries no version file, the workflow stamps the tag's version into the packed kit, and a
published release is immutable, so a tag is never reused. The workflow packs `bin/`, `tree/`, and `targets/` into
`kit-<version>.tar.gz` and publishes the release; it refuses a tag that is not a version, a release tag off
`main`, and a pre-release tag on `main`. A release retires its pre-releases, `v<version>-*`, tags and all.

Repository policy is managed by Terraform; see: empowerite/infrastructure/terraform/projects/kit/main.tf
