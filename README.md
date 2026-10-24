# kit

kit is the agentic guardrails a repository takes as a versioned mixin: the policy text an agent reads first, the
rules that bind each kind of file, the guard that keeps `main` as the human's desk, and the one command no harness
provides, the issue-to-worktree on-ramp. A tag on `main` publishes a release. A consumer pins it in `mise.toml`,
installs it with `kit install`, and keeps rules of its own beside it. What the first release delivers, and what the
kit is not, is [#1](https://github.com/empowerite/kit/issues/1).

## Setup

Every commit is signed, and the branch ruleset refuses one that is not. Once per clone:

```sh
git config gpg.format openpgp
git config user.signingkey <a key GitHub verifies for you>
git config commit.gpgsign true
```

## Getting started

Work here moves by the loop the kit itself delivers. An issue exists. A branch is checked out in a worktree under
`worktrees/`, linked to the issue under Development, with a draft pull request. Commits are signed, and their
subject is `#<N> <area>: <subject>`. `gh pr ready` says the work is done, and the pull request body is the report.
A human merges. Nothing reaches `main` any other way.

Until the kit can run itself, those steps are taken by hand. The first thing #1 delivers is the command that takes
them.

## Status

Pre-alpha. There is no code yet, so nothing here builds, tests, or runs. The repository will use its own kit as soon
as the kit exists.

Repository policy is managed by Terraform; see: empowerite/infrastructure/terraform/projects/kit/main.tf
