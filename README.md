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

`kit install` copies `.kit/1st.md` and `.kit/rules/` into the repository, places `.github/workflows/kit-ci.yml`,
ignores `worktrees/`, points this clone's `core.hooksPath` at the release's hooks, and writes `.kit/manifest`. Each
target named adds what that harness insists on having outside `.kit/`; `claude` places `.claude/rules/kit-*.md` and
merges the kit's deny rules into `.claude/settings.json`. A repository with no target installed still binds every
agent that reads `AGENTS.md`, through the rules table in `.kit/1st.md`. `kit target add` and `kit target remove`
change the set later, and `kit target list` prints it. Commit what it wrote. Then:

- Open `AGENTS.md` with `@.kit/1st.md` and the sentence "Read .kit/1st.md first." The rest of the file is yours.
- Keep a `ci` recipe in the root justfile. The delivered `CI` job runs it, then `kit check`, then refuses a pull
  request not linked to an open issue.
- For `just issue 42`, add a veneer recipe over `kit issue` to the root justfile; `kit issue 42` works as it is.
- In every other clone, run `kit setup` once to wire the hooks.

The machinery, the `kit` command, the bash behind `kit issue`, `kit tidy`, and `kit status`, and the hooks, stays
in the release where mise installed it; the repository carries the text an agent reads and the record of what it
took.

To move to a new release, bump the pin, run `mise install`, then `kit update`: a delivered file you left alone is
replaced, one you edited is three-way merged with the previous release as the base, and your own deny entries
and settings keys stay. `kit check` fails, locally and in `CI`, when a delivered file differs from the pinned
release.

The kit needs bash 4 or later, git, `gh`, and `jq`. macOS ships bash 3.2; `brew install bash` gives a current one.

## Setup

Every commit is signed, and the branch ruleset refuses one that is not. If you have no signing key GitHub
verifies, make one and register it, following [generating a new GPG key](https://docs.github.com/en/authentication/managing-commit-signature-verification/generating-a-new-gpg-key)
and [adding a GPG key to your GitHub account](https://docs.github.com/en/authentication/managing-commit-signature-verification/adding-a-gpg-key-to-your-github-account).
Then, once per clone, with the key's id in `signing_key`:

```sh
git config gpg.format openpgp
git config user.signingkey "${signing_key}"
git config commit.gpgsign true
mise install
bin/kit setup
```

This repository carries the installed copy of its own kit, and `just ci` checks it.

## Getting started

Work here moves by the loop in [.kit/1st.md](.kit/1st.md):

- An issue exists. A change without an issue has not started.
- `just issue <N>`, a veneer over `kit issue`, makes `worktrees/<N>-<slug>`, links the branch to the issue under
  Development, and opens a draft pull request. It never switches the checkout it runs from.
- Work happens in that worktree. Commits are signed, and their subject is `#<N> <area>: <subject>`.
- `just ci` is the gate: shellcheck over the bash, actionlint over the workflows, the bats suite in `tests/`, and
  `bin/kit check`.
- `gh pr ready` says the work is done, and the pull request body is the report.
- A human merges. `just tidy` then removes the worktree and its branch.

## Maintaining

This repository is both the source of the kit and a consumer of it, so some committed files are written by
`bin/kit install` from others and are not edited by hand:

| Installed copy | Written from |
| --- | --- |
| `.claude/rules/kit-<kind>.md` | `.kit/rules/<kind>.md`, header and all, with a one-line notice |
| `.claude/settings.json` | `.kit/settings.json`, merged by key; here it carries only the kit's entries |
| `.github/workflows/kit-ci.yml` | `.kit/ci.yml` |
| `.kit/manifest` | the release version, the targets taken, and a digest per delivered path |

Everything else is source, all of `.kit/` included: `1st.md`, `rules/`, `bash/`, `hooks/`, `ci.yml`,
`settings.json`, and `version`, together with `bin/kit`, the two workflows, and `tests/`. After changing a source
file under `.kit/`, run `just refresh`, which reinstalls the kit into this repository, and commit the result with
the change. `bin/kit check`, which `just ci` runs, names any installed copy that no longer matches its source. The
copies that can carry a comment say they are placed; `settings.json` cannot, so this table is its notice.

## Release

`.kit/version` carries the version under development, without a suffix. A release is the tag `v<version>` on a
commit on `main`, not only its head. A pre-release is `v<version>-<suffix>`, say `v0.1.0-rc.1`, on a commit that is
not on `main`, and it is published as a GitHub pre-release. In both cases the workflow packs `bin/` and `.kit/` into
`kit-<version>.tar.gz` with the tag's full version stamped into the packed `.kit/version`, and publishes the release.
It refuses a tag that is not a version, a release tag off `main`, a pre-release tag on `main`, and a tag whose base
version is not `.kit/version`.

Repository policy is managed by Terraform; see: empowerite/infrastructure/terraform/projects/kit/main.tf
