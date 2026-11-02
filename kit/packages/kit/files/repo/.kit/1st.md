<!-- kit:generated -- the entry every agent reads first, from the kit package -->
<!-- see: "Changing what kit adds" below, to replace it -->
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

An agent never checks out, resets, rebases, or otherwise changes the desk in place. An agent works in a worktree,
which `kit issue` makes. Work reaches the default branch only through a merged pull request.

## The workflow

How work moves from an issue to a merged pull request is the repository's to say. With the `workflow` package active,
`.kit/workflow.md` says it, and an agent follows it as it follows this file. Without it, `AGENTS.md` says it.

## Kit commands instead of git commands

Some git commands have a kit replacement that adds the check its rule requires. An agent uses only the kit
command. A harness with a deny list refuses the raw git command.

- `kit rebase`, never `git rebase`. `kit rebase` refuses to run in the root checkout or on the default branch.
  Otherwise it rebases the worktree's branch onto the default branch at origin and pushes it.
- `kit commit`, never `git commit`. `kit commit` refuses the desk and the default branch, runs the linters over the
  staged paths, refuses a path the kit placed, and holds the subject to the policy before the commit exists.
- `kit submit`, never `git push`. `kit submit` commits what is staged the same way, then pushes the branch. It never
  forces: when origin has moved, it names `kit rebase`.
- `kit issue`, never `git worktree add`, `git checkout -b`, or `git switch -c`.
- `kit tidy`, never `git worktree remove` or `git branch -D`.
- An agent never switches or resets the desk. `git checkout main`, `git switch main`, `git reset --hard`, and
  `git push --force` are refused as well.

These limits do not apply to the human at the desk.

## Rules by file type

Each kind of file is defined under `.kit/kinds/`, by its `name:`: the path globs and first-line patterns that make a
file of that kind, and the kinds found inside its files, such as the bash in a justfile. A kind may have a rule file
under `.kit/rules/`; `any`, `text`, and `yaml` have none of their own. A rule says what the work must do to be
trusted. A style file under `.kit/styles/`, from a style package, says how the file looks. Rule files and style
files have the same frontmatter, with these keys:

| Key | Contains |
| --- | --- |
| `kind:` | The kinds of file it governs, by name. |
| `lint:` | The linter that `kit lint` runs over the files it governs. |
| `checks:` | The kit's own checks that `kit lint` runs over the files it governs, by name. |
| `templates:` | The shapes a file of this kind can take. The templates are under `.kit/templates/`. |

A rule or style governs every file of a kind it names, and `kit kinds <path>` prints the kinds a file is of.
`kit context` prints the rules of every kind a file is of, then the rules of the kinds those kinds embed, so a
justfile brings the bash rule with it. `kit lint` does not follow embeds, so no linter runs over a file it cannot
parse.

Before you change a file:

1. Run `kit context <path>`. It prints the rules, then the style, that govern the file, and records that you asked.
2. Read every file it prints.

No harness keeps a copy of the rules. `kit context` is how every harness, and every person, finds them. Where the
harness has the kit's MCP server, `kit serve`, the read-only verbs are tools, `context` and `lint` among them, and
the rules are resources, starting at `kit://1st.md`.

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

`kit new` also makes a kind, a rule, a style, a template, a package, a bash module, or a seam's provider, in a
package or module directory of the repository's own. `kit help new` lists them.

| Kind | Rule |
| --- | --- |
| Markdown | [rules/markdown.md](rules/markdown.md) |
| Bash | [rules/bash.md](rules/bash.md) |
| Just | [rules/just.md](rules/just.md) |
| Workflows | [rules/workflows.md](rules/workflows.md) |
| JSON | [rules/json.md](rules/json.md) |
| jq modules | [rules/jq.md](rules/jq.md) |
| Ignore lists | [rules/gitignore.md](rules/gitignore.md) |

## What the kit delivers

The repository commits the declaration, and every worktree holds the text:

- `kit.yml`, committed: its `version:` and `packages:` declare the release and the active packages.
- `.kit/packages/`, committed: the repository's own packages.
- Everything else under `.kit/`, placed by `kit hydrate`: this file, the kinds, the rules, the styles from the
  style packages the repository takes, and the rest of what the packages deliver. Each is a link to its source
  when the source is inside the worktree, and otherwise a copy without the write bit. `.kit/.gitignore` names every
  one, so git never sees them.
- `.kit/manifest`, the digest of each delivered file, and `.kit/.gitignore` itself: written by kit, and ignored
  and without the write bit like the rest.

The installed release contains the code. You run it through the kit CLI, `kit` on `PATH`. The workflow commands
are `kit issue`, `kit commit`, `kit submit`, `kit rebase`, `kit tidy`, `kit status`,
`kit context`, `kit new file`, `kit lint`, `kit gate`, and `kit hydrate`. Run them exactly as written here, and in
no other way.

Everything the release delivers belongs to a package. A repository activates the packages it wants:

- `kit install github/ci hooks/lefthook agents/claude` activates packages when the kit is installed.
- `kit packages add agents/claude` activates a package later.
- `kit packages remove` deactivates a package.
- `kit uninstall` takes everything the kit placed out of the repository.

A package of the repository's own lives under `.kit/packages/` and is activated the same way.

How a delivered file is placed depends on its destination:

- A `.json` file, other than a schema, is merged by key. The kit owns only the entries it writes.
- A region inside a file the repository owns starts at a `kit:generate` directive and ends at a `kit:end` line.
  `kit generate` fills it with the output of the directive's command, or of `kit region` when it names none.
- Every other file under `.kit/` is a link to its source when the source is inside the worktree, and otherwise a
  copy without the write bit. Every other file outside `.kit/` is copied as it is.

The packages:

- `kit`: this file, `.kit/README.md` for the person at the desk, the pages of `kit welcome`, the kinds, the rules,
  `.kit/kit.schema.json`, and a region that `kit region` fills: the `/worktrees/` entry in the repository's
  `.gitignore`.
- `workflow`: `.kit/workflow.md`, the workflow from an issue to a merged pull request: a worktree per issue, a
  draft pull request, the report in its body, plans as prose, and the worker contract. "The workflow" above points
  to it.
- `style/bash`, `style/markdown`, `style/just`, `style/workflows`, `style/yaml`, `style/json`, and
  `style/gitignore`: the style for one kind of file each, under `.kit/styles/`, and the kind's templates (JSON has
  none). `style/markdown` also delivers `.kit/markdownlint.jsonc`, the settings its linter reads. A repository takes
  the ones it agrees with, or writes its own.
- `style/prose`: `.kit/styles/prose.md`, the style of prose in every text file, documents and comments alike.
  `kit context` prints it after the rule and the style for the file's kind.
- `hooks/lefthook`: `.kit/lefthook.yml`, a pre-commit hook that runs `kit lint` and `kit generate --check` over the
  staged paths. A repository's `lefthook.yml` includes it with `extends` and adds its own commands. `kit lint`
  does not need a hook manager: it runs each rule's linter over the paths the rule governs.
- `github/ci`: `.github/workflows/kit-ci.yml`, one GitHub Actions job. The job installs the gate's tools with
  `kit toolchains install`, then runs `kit ci`. `kit ci` runs four checks, on the runner or on any machine; before
  the branch has a pull request, `kit linked` fails and the other three run without one:
  - the repository's gate
  - `kit check`
  - `kit linked`, which refuses a pull request not linked to an open issue
  - `kit policy`, which refuses a branch name, commit subject, or title that breaks the repository's policy
- `policy/opa`: `.kit/policy/kit.rego`, the naming policy in Rego. With it, `kit policy` runs on Open Policy
  Agent, and a rule can express what a pattern cannot.
- `agents/generic`: support for any agent that reads `AGENTS.md`. Every `agents/<harness>` package below depends
  on it. It delivers a region in the repository's `AGENTS.md`, filled by `kit region`, that points to this file.
- `agents/claude`: support for Claude Code, which reads `AGENTS.md` only when it finds no `CLAUDE.md`. It delivers:
  - a region in `.claude/CLAUDE.md` that imports `AGENTS.md`, so Claude Code reads `AGENTS.md` in every case
  - the deny list above, merged into `.claude/settings.json`
  - a hook that runs `kit context` before every write, so the agent is told the file's rules without asking
- `agents/cursor`: support for Cursor, which reads `AGENTS.md` on its own. It delivers a rule under `.cursor/rules/`
  for each kit rule, attached by the rule's paths, whose body sends the agent to `.kit/rules/<kind>.md`.
- `agents/copilot`: support for GitHub Copilot, which reads `AGENTS.md` on its own. It delivers an instruction file
  under `.github/instructions/` for each kit rule, applied to the rule's paths, whose body sends the agent to the rule.
- `agents/gemini`: support for Gemini CLI, which reads `GEMINI.md` unless told otherwise. It merges `AGENTS.md` into
  the context files and the deny list above into `.gemini/settings.json`.
- `empowerite`: a bundle, with no file of its own. It includes every style package, `github/ci`,
  `agents/claude`, `hooks/lefthook`, and `workflow`, which is what Empowerite's repositories take.

Any file may also contain a `kit:generate` directive of the repository's own, naming any command. `kit check` fails
when any region differs from what its command produces.

## Changing what kit adds

Every file kit adds belongs to a package. A file under `.kit/` opens with a `kit:generated` line that names what it is
and its package, and a `see:` line that says how to stop using it or replace it. It cannot be edited where it lies:
a link leads to its source, a copy has no write bit, and git never sees either. `kit check` fails when any
delivered file differs from the pinned release. Instead:

- To stop using a file, remove its package: `kit packages remove <package>`. It takes out everything the package
  added. A package that another active package depends on is refused until that one is removed. A member of a
  bundle such as `empowerite` can be removed on its own: kit records it as excluded, and `kit update` leaves it
  out.
- To replace a file, write a package of the repository's own under `.kit/packages/<name>/`. Its `package.yml`
  depends on the package that adds the file, and it places the replacement at the same path under `files/repo/`.
  Activate it with `kit packages add <name>`. The later package in dependency order wins, so the repository's file
  is used, and `kit update` still brings everything else from the next release.
- To fill in a value that a template asks for, set it under `answers:` in `kit.yml`.
- In a file kit shares with the repository, such as `AGENTS.md`, `.gitignore`, or `.claude/settings.json`, only
  the kit's region or the kit's entries are kit's. The rest of the file is the repository's to edit.
