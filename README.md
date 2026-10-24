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

`kit install` copies `.kit/` into the repository, places `.github/workflows/kit-ci.yml`, ignores `worktrees/`,
points `core.hooksPath` at `.kit/hooks`, and writes `.kit/manifest`. Each target named adds what that harness
insists on having outside `.kit/`; `claude` places `.claude/rules/kit-*.md` and merges the kit's deny rules into
`.claude/settings.json`. A repository with no target installed still binds every agent that reads `AGENTS.md`,
through the rules table in `.kit/1st.md`. `kit target add` and `kit target remove` change the set later, and
`kit target list` prints it. Commit what it wrote. Then:

- Open `AGENTS.md` with `@.kit/1st.md` and the sentence "Read .kit/1st.md first." The rest of the file is yours.
- Add `mod kit ".kit/kit.just"` to the root justfile for `just kit issue`, `just kit tidy`, and `just kit status`.
- Keep a `ci` recipe in the root justfile. The delivered `CI` job runs it, then `kit check`, then refuses a pull
  request not linked to an open issue.

To move to a new release, bump the pin, run `mise install`, then `kit update`: a delivered file you left alone is
replaced, one you edited is three-way merged with the previous release as the base, and your own deny entries
and settings keys stay. `kit check` fails, locally and in `CI`, when a delivered file differs from the pinned
release.

The kit needs bash 4 or later, git, `gh`, and `jq`. macOS ships bash 3.2; `brew install bash` gives a current one.

## Setup

Every commit is signed, and the branch ruleset refuses one that is not. Once per clone:

```sh
git config gpg.format openpgp
git config user.signingkey <a key GitHub verifies for you>
git config commit.gpgsign true
```

Then `mise install` for the pinned tools. `bin/kit install claude` is already done: this repository carries the
installed copy of its own kit, which `just ci` checks.

## Getting started

Work here moves by the loop in [.kit/1st.md](.kit/1st.md). An issue exists. `just kit issue <N>` makes
`worktrees/<N>-<slug>`, links the branch to the issue under Development, and opens a draft pull request. Commits
are signed, and their subject is `#<N> <area>: <subject>`. `gh pr ready` says the work is done, and the pull
request body is the report. A human merges.

`just ci` is the gate: shellcheck over the bash, actionlint over the workflows, the bats suite in `tests/`, and
`bin/kit check`.

## Release

`.kit/version` carries the version under development, without a suffix. A release is the tag `v<version>` on a
commit on `main`, not only its head. A pre-release is `v<version>-<suffix>`, say `v0.1.0-rc.1`, on a commit that is
not on `main`, and it is published as a GitHub pre-release. In both cases the workflow packs `bin/` and `.kit/` into
`kit-<version>.tar.gz` with the tag's full version stamped into the packed `.kit/version`, and publishes the release.
It refuses a tag that is not a version, a release tag off `main`, a pre-release tag on `main`, and a tag whose base
version is not `.kit/version`.

Repository policy is managed by Terraform; see: empowerite/infrastructure/terraform/projects/kit/main.tf
