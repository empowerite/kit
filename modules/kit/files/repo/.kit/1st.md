<!-- Delivered by kit, not edited by hand. A change is a new release, then kit update. -->
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
   written, through the loop, into `AGENTS.md`, a module of the repository's own under `.kit/modules/`, or a kit
   release, where every agent and every person reads it.

## The root checkout is the human's desk

The checkout at the repository root is the human's. It is where they type, review, and leave work in progress.
It usually has `main` checked out, and may have any branch no worktree holds. An agent never checks out, resets,
rebases, or otherwise changes it in place. Work happens in a worktree under `worktrees/`, one per issue, and
reaches `main` only through a merged pull request.

## The loop

1. An issue exists. A change without an issue has not started.
2. `kit issue <N>` makes `worktrees/<N>-<slug>`, a branch linked to the issue under Development, and a draft
   pull request. It never switches the checkout it is run from.
3. Work happens in that worktree. Every commit is signed, and its subject is `#<N> <area>: <subject>`. `kit rebase`
   brings the branch up to date with the default branch and pushes it.
4. `gh pr ready` says the work is done. The pull request body is the report: what changed, what was checked
   and how, and what was left out and why.
5. A human merges. Nothing reaches `main` any other way, and nothing is called merged until the merge is
   observed.

## Spellings an agent uses

Where the kit adds a rule to a git verb, the kit's spelling is the only one an agent uses. Each kit spelling
carries the check its rule requires, and a harness with a deny list refuses the raw git spelling.

- `kit rebase`, never `git rebase`. It refuses the root checkout and the default branch, and otherwise rebases the
  worktree's branch onto the default branch at origin and pushes it.
- `kit issue`, never `git worktree add`, `git checkout -b`, or `git switch -c`.
- `kit tidy`, never `git worktree remove` or `git branch -D`.
- The desk is never switched or reset by an agent: `git checkout main`, `git switch main`, `git reset --hard`,
  and `git push --force` are refused as well.

The human at the desk is not bound by this; the root checkout is theirs.

## Plans are prose

A design, a spec, or a plan is the body of an issue, written in prose. It is never a committed file. When
the design is written, the issue is the deliverable, and the loop above starts from it.

## The worker contract

A worker is whoever implements one issue in the worktree that issue names.

1. `cd` into the worktree named in the task and confirm `git rev-parse --abbrev-ref HEAD` prints a branch name
   beginning `<N>-`.
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

Each file kind has a rule file under `.kit/rules/`, and each rule file's header names what it governs: `paths:`,
globs on the repository path, and `content:`, patterns on the first line; `lint:` names the linter `kit lint`
runs over the paths it governs. Before you create or change a file, run
`kit context <path>`. It names the rules that govern the file and records that you asked. Read what it names. No
harness holds a copy of a rule; this call is how every harness, and every person, finds it.

| Kind | Rule |
| --- | --- |
| Markdown | [rules/markdown.md](rules/markdown.md) |
| Bash | [rules/bash.md](rules/bash.md) |
| Just | [rules/just.md](rules/just.md) |
| YAML | [rules/yaml.md](rules/yaml.md) |
| Workflows | [rules/workflows.md](rules/workflows.md) |
| JSON | [rules/json.md](rules/json.md) |
| Ignore lists | [rules/gitignore.md](rules/gitignore.md) |

## What the kit carries

The repository carries the text: this file, the rules under `.kit/rules/`, and `.kit/manifest`, which records the
release version, the active modules, and a checksum per delivered file. The release, where it is installed,
carries the machinery, reached through the `kit` command on `PATH`: the loop's commands, `kit issue`, `kit rebase`,
`kit tidy`, `kit status`, `kit context`, and `kit lint`, spelled that way and no other. Everything the release
delivers is a module, and a repository activates the modules it takes: `kit install ci-github lefthook claude`, or
`kit module add claude` later, and `kit module remove` to take one back out; a module of the repository's own
lives under `.kit/modules/` and is taken the same way. A `.json` destination, other than a schema, is merged by
key, the kit owning only the entries it writes; a region inside a file the repository owns is opened by a
directive, `kit:generate`, and made by the command it names, through `kit generate`; everything else is copied
as it is.

- `kit`: this file and the rules; the sentence in `AGENTS.md` that points here and the `/worktrees/` entry in
  `.gitignore`, each a region `kit region` fills inside a file the repository owns. Any file may carry a directive
  of the repository's own, `kit:generate` naming any command, and `kit check` holds every region to its command.
- `lefthook`: `.kit/lefthook.yml`, a pre-commit hook that runs `kit lint` over the staged paths. A repository's
  `lefthook.yml` takes it with `extends` and adds commands of its own. `kit lint` itself needs no hook manager:
  it runs each rule's linter over the paths the rule governs.
- `ci-github`: `.github/workflows/kit-ci.yml`, one GitHub Actions job that runs `kit ci`: the repository's gate,
  `kit check`, and the refusal of a pull request not linked to an open issue. `kit ci` runs the same three
  anywhere.
- `claude`: for Claude Code, which reads `AGENTS.md` only when it finds no `CLAUDE.md`: a region in
  `.claude/CLAUDE.md` importing `AGENTS.md`, so it is read in every case; the deny list above merged into
  `.claude/settings.json`; and a hook that runs `kit context` before every write so the rules are named to the agent
  without being asked for.

`kit check` fails when a delivered file differs from the pinned release. Change the kit's files by pinning a
new release and running `kit update`, never by editing them in place.
