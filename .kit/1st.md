<!-- Added by the kit package. Do not edit it here; to change it, see "Changing what kit adds" below. -->
# Read this first

This file is the kit's entry point. Every agent follows it in every session, in any repository that has a `.kit/`
directory. The repository's own rules in `AGENTS.md` apply as well. Where the two differ, `AGENTS.md` wins.

## Integrity

1. Report only what happened. Write "done", "passing", or "verified" only for something that ran and succeeded.
2. Report a skip as a skip. A check that was not run, is not implemented, or cannot be implemented is reported as
   exactly that. It passes no gate and earns no credit.
3. Give the evidence for every claim, or say there is none. Name the command that ran and what it printed.
4. Being wrong is recoverable. Misreporting is not.
5. Rules live in the repository, never in a session's memory. When you learn a convention or receive a correction,
   write it down, through the workflow, in one of these places, where every agent and every person reads it:
   - `AGENTS.md`
   - a package of the repository's own under `.kit/packages/`
   - a kit release

## The root checkout is the human's desk

The desk is the root checkout on the human's machine, on whatever branch it currently has. It belongs to the human.
They type, review, and leave work in progress there. It usually has `main` checked out, and may have any branch that
no worktree has checked out.

An agent never checks out, resets, rebases, or otherwise changes the desk in place. An agent works in a worktree
under `worktrees/`, one per issue. Work reaches `main` only through a merged pull request.

## The workflow

1. Start from an issue. A change without an issue has not started.
2. `kit issue <N>` creates the following, and never switches the checkout you run it from:
   - the worktree `worktrees/<N>-<slug>`
   - a branch, linked to the issue under Development
   - a draft pull request
3. Work in that worktree. Sign every commit. Write each commit subject as `#<N> <area>: <subject>`. Run
   `kit rebase` to bring the branch up to date with the default branch and push it.
4. Run `gh pr ready` when the work is done. The pull request body is the report. It states:
   - what changed
   - what was checked, and how
   - what was left out, and why
5. A human merges. Nothing reaches `main` any other way. Do not call anything merged until you see the merge.

## Kit commands instead of git commands

Some git commands have a kit replacement that adds the check its rule requires. An agent uses only the kit
command. A harness with a deny list refuses the raw git command.

- `kit rebase`, never `git rebase`. `kit rebase` refuses to run in the root checkout or on the default branch.
  Otherwise it rebases the worktree's branch onto the default branch at origin and pushes it.
- `kit issue`, never `git worktree add`, `git checkout -b`, or `git switch -c`.
- `kit tidy`, never `git worktree remove` or `git branch -D`.
- An agent never switches or resets the desk. `git checkout main`, `git switch main`, `git reset --hard`, and
  `git push --force` are refused as well.

These limits do not apply to the human at the desk.

## Plans are prose

Write a design, a spec, or a plan as prose in the body of an issue. Never commit it as a file. When the design is
written, the issue is the deliverable, and the workflow above starts from it.

## The worker contract

A worker is whoever implements one issue, in the worktree named for that issue.

1. `cd` into the worktree named in the task. Confirm that `git rev-parse --abbrev-ref HEAD` prints a branch name
   that begins with `<N>-`.
2. Run every command from there. Never touch the root checkout.
3. Never run `kit issue`, create a branch, or create a worktree. These were done before the handoff.
4. Implement what the issue asks and nothing else.
5. Push as you go.
6. Before you call anything done, run the repository's gate (its own CI command, such as `just ci`) through
   the pinned toolchain.
7. Finish with `gh pr ready`, with the report in the pull request body.
8. If the gate fails or the work is partial, leave the pull request as a draft and say why.
9. Never merge.
10. Report what ran and what it printed. Nothing more.

## Rules by file type

Each kind of file has a rule file under `.kit/rules/`. A rule says what the work must do to be trusted. A
repository that takes a style package also has a style file for that kind under `.kit/style/`, which says how the
file looks. Rule files and style files have the same frontmatter, with these keys:

| Key | Contains |
| --- | --- |
| `paths:` | Globs matched against a file's path from the repository root. The file governs a path that matches. |
| `content:` | Patterns matched against a file's first line. The file governs a file that matches. |
| `lint:` | The linter that `kit lint` runs over the files it governs. |
| `templates:` | The shapes a file of this kind can take. The templates are under `.kit/templates/`. |

Before you change a file:

1. Run `kit context <path>`. It prints the rules, then the style, that govern the file, and records that you asked.
2. Read every file it prints.

No harness keeps a copy of the rules. `kit context` is how every harness, and every person, finds them.

Where a template governs a new file, create it with `kit new file <path>`, giving the path from the repository
root, and never write it from nothing. `kit new file`:

- writes the file from its template
- fills the template's holes that the path supplies
- refuses a hole you left empty, and names the flag that fills it, such as `--purpose=<…>` or `--title=<…>`
- prints the rules to read before you write the rest of the file

Two flags choose the template:

- `--variant=<…>` picks another of the shapes, such as a sourced bash module.
- `--rule=<…>` names the rule or style for a path with no extension.

`kit lint` fails any file, including one written by hand, whose shape matches none of its templates.

| Kind | Rule |
| --- | --- |
| Markdown | [rules/markdown.md](rules/markdown.md) |
| Bash | [rules/bash.md](rules/bash.md) |
| Just | [rules/just.md](rules/just.md) |
| Workflows | [rules/workflows.md](rules/workflows.md) |
| JSON | [rules/json.md](rules/json.md) |
| Ignore lists | [rules/gitignore.md](rules/gitignore.md) |

## What the kit delivers

The repository contains the text:

- this file
- the rules under `.kit/rules/`
- the style under `.kit/style/`, from the style packages the repository takes
- `.kit/manifest`, which records the release version, the active packages, and a checksum for each delivered file

The installed release contains the code. You run it through the `kit` command on `PATH`. The workflow commands
are `kit issue`, `kit rebase`, `kit tidy`, `kit status`, `kit context`, `kit new file`, and `kit lint`. Run them
exactly as written here, and in no other way.

Everything the release delivers belongs to a package. A repository activates the packages it wants:

- `kit install ci-github lefthook claude` activates packages when the kit is installed.
- `kit package add claude` activates a package later.
- `kit package remove` deactivates a package.

A package of the repository's own lives under `.kit/packages/` and is activated the same way.

How a delivered file is placed depends on its destination:

- A `.json` file, other than a schema, is merged by key. The kit owns only the entries it writes.
- A region inside a file the repository owns starts at a `kit:generate` directive. `kit generate` fills the region
  by running the command the directive names.
- Every other file is copied as it is.

The packages:

- `kit`: this file, the rules, and two regions that `kit region` fills:
  - the sentence in the repository's `AGENTS.md` that points to this file
  - the `/worktrees/` entry in the repository's `.gitignore`
- `style-bash`, `style-markdown`, `style-just`, `style-workflows`, `style-yaml`, `style-json`, and
  `style-gitignore`: the style for one kind of file each, under `.kit/style/`, with its templates. `style-markdown`
  also delivers `.kit/markdownlint.jsonc`, the settings its linter reads. A repository takes the ones it agrees
  with, or writes its own.
- `lefthook`: `.kit/lefthook.yml`, a pre-commit hook that runs `kit lint` and `kit generate --check` over the
  staged paths. A repository's `lefthook.yml` includes it with `extends` and adds its own commands. `kit lint`
  does not need a hook manager: it runs each rule's linter over the paths the rule governs.
- `ci-github`: `.github/workflows/kit-ci.yml`, one GitHub Actions job. The job installs the gate's tools with
  `kit toolchain install`, then runs `kit ci`. `kit ci` runs four checks, and runs the same four on any machine:
  - the repository's gate
  - `kit check`
  - `kit linked`, which refuses a change that is not linked to an open issue
  - `kit policy`, which refuses a branch name, commit subject, or title that breaks the repository's policy
- `policy-opa`: `.kit/policy/kit.rego`, the naming policy in Rego. With it, `kit policy` runs on Open Policy
  Agent, and a rule can express what a pattern cannot.
- `claude`: support for Claude Code, which reads `AGENTS.md` only when it finds no `CLAUDE.md`. It delivers:
  - a region in `.claude/CLAUDE.md` that imports `AGENTS.md`, so Claude Code reads `AGENTS.md` in every case
  - the deny list above, merged into `.claude/settings.json`
  - a hook that runs `kit context` before every write, so the agent sees the rules without asking

Any file may also contain a `kit:generate` directive of the repository's own, naming any command. `kit check` fails
when any region differs from what its command produces.

## Changing what kit adds

Every file kit adds belongs to a package, and the file's first lines name it. `kit check` fails when the file
differs from that package's copy in the pinned release, so do not edit it in place. Instead:

- To stop using a file, remove its package: `kit package remove <package>`. It takes out everything the package
  added. A package that another active package depends on is refused until that one is removed.
- To replace a file, write a package of the repository's own under `.kit/packages/<name>/`. Its `package.yml`
  depends on the package that adds the file, and it places the replacement at the same path under `files/repo/`.
  Activate it with `kit package add <name>`. The later package in dependency order wins, so the repository's file
  is used, and `kit update` still brings everything else from the next release.
- To fill in a value that a template asks for, set it under `answers:` in `.kit/settings.yml`.
- In a file kit shares with the repository, such as `AGENTS.md`, `.gitignore`, or `.claude/settings.json`, only
  the kit's region or the kit's entries are kit's. The rest of the file is the repository's to edit.
