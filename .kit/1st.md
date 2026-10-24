# Read this first

This file is the kit's entry. It binds every agent and every session in a repository that carries `.kit/`, and
the repository's own rules in `AGENTS.md` bind beside it. Where the two differ, the repository's rule wins.

## Integrity

1. Report only what happened. Write "done", "passing", or "verified" only for what ran and held.
2. A skip is a skip. A check not run, not implemented, or not implementable is reported as exactly that. It
   clears no gate and earns no credit.
3. A claim carries its evidence, or says it has none. Name the command that ran and what it printed.
4. Being wrong is recoverable. Misreporting is not.
5. A rule lives in the repository, never in a session's memory. A convention learned or a correction given is
   written into a rule file or `AGENTS.md` through the loop, where every agent and every person reads it.

## `main` is the human's desk

The checkout at the repository root stays on the default branch. It is where the human types, reviews, and
leaves work in progress. Never check out, reset, rebase, or otherwise change it in place. Work happens in a
worktree under `worktrees/`, one per issue, and reaches `main` only through a merged pull request.

## The loop

1. An issue exists. A change without an issue has not started.
2. `kit issue <N>` makes `worktrees/<N>-<slug>`, a branch linked to the issue under Development, and a draft
   pull request. It never switches the checkout it is run from.
3. Work happens in that worktree. Every commit is signed, and its subject is `#<N> <area>: <subject>`.
4. `gh pr ready` says the work is done. The pull request body is the report: what changed, what was checked
   and how, and what was left out and why.
5. A human merges. Nothing reaches `main` any other way, and nothing is called merged until the merge is
   observed.

## Plans are prose

A design, a spec, or a plan is the body of an issue, written in prose. It is never a committed file. When
the design is written, the issue is the deliverable, and the loop above starts from it.

## The worker contract

A worker is whoever implements one issue in the worktree that issue names.

1. `cd` into the worktree named in the task and confirm `git rev-parse --abbrev-ref HEAD` prints `<N>-`.
2. Do every command from there. Never touch the repository root checkout.
3. Never run `kit issue`, make a branch, or make a worktree. Those were done before the handoff.
4. Implement what the issue asks and nothing beside it.
5. Push as you go.
6. Run the repository's gate before calling anything done, and run it through the pinned toolchain.
7. End with `gh pr ready` and the report in the pull request body.
8. If the gate fails or the work is partial, leave the pull request a draft and say why.
9. Never merge.
10. Report what ran and what it printed, nothing more.

## Rules by file type

Each file kind has a rule file under `.kit/rules/`, and each rule file names the paths it binds in its `paths:`
header. A harness that loads rules by path finds a copy under its own rules directory when the repository takes
that target, `.claude/rules/kit-<kind>.md` for Claude Code. Read the rule for a kind before writing a file of
that kind.

| Kind | Paths | Rule |
| --- | --- | --- |
| Markdown | `*.md` | [rules/markdown.md](rules/markdown.md) |
| Bash | `*.sh`, `*.bash`, `.kit/hooks/*` | [rules/bash.md](rules/bash.md) |
| Just | `justfile`, `*.just` | [rules/just.md](rules/just.md) |
| YAML | `*.yml`, `*.yaml` | [rules/yaml.md](rules/yaml.md) |
| JSON | `*.json`, `*.jsonc` | [rules/json.md](rules/json.md) |
| Ignore lists | `.gitignore` | [rules/gitignore.md](rules/gitignore.md) |

## What the kit carries

The repository carries the text: this file, the rules under `.kit/rules/`, and `.kit/manifest`, which records the
release version, the targets taken, and a checksum per delivered file. The release, where mise installed it,
carries the machinery, reached through the `kit` command on `PATH`:

- `kit issue`, `kit tidy`, and `kit status`, the loop's commands. A root justfile that wants `just issue 42` adds
  a veneer recipe over `kit issue`.
- Two hooks. `pre-rebase` refuses a rebase of the default branch. `pre-commit` lints the staged paths. `kit install`
  points a clone's `core.hooksPath` at them, and every other clone runs `kit setup` once.
- Targets. A harness that insists on files outside `.kit/` is a target, and a repository takes only the targets
  it names: `kit install claude`, or `kit target add claude` later. The `claude` target places
  `.claude/rules/kit-<kind>.md` and merges six deny rules into `.claude/settings.json`. The kit owns only the
  entries it writes; the repository may add its own. `kit target remove` takes a target back out.
- `.kit/ci.yml`, placed as `.github/workflows/kit-ci.yml`: one `CI` job that runs the consumer's `just ci`,
  runs `kit check`, and refuses a pull request not linked to an open issue.

`kit check` fails when a delivered file differs from the pinned release. Change the kit's files by pinning a
new release and running `kit update`, never by editing them in place.
