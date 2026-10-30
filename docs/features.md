# Features

kit is a versioned release of house rules for agents, and of the checks that hold agents and people to them. This
page lists what it does, grouped by purpose, with a pointer to the page that describes each part in full.

## Rules where agents look

- `.kit/1st.md` is the file every agent reads first. It sets the integrity rules for reporting, the workflow, the
  worker contract, and the kit commands that replace raw git commands.
- `install` writes a sentence into `AGENTS.md` that sends every agent to `.kit/1st.md`. Claude Code reads
  `AGENTS.md` on its own only when the project has no `CLAUDE.md`, so the `agents/claude` package imports
  `AGENTS.md` from `.claude/CLAUDE.md`.
- Each kind of file has one rule, `.kit/rules/<kind>.md`, that names the paths it governs and the linter it runs. A
  style package adds `.kit/styles/<kind>.md`, how a file of that kind looks, with the same frontmatter.
- `kit context` takes one or more paths and prints the rules that govern them, each rule once. `--json` prints the
  rules per path. Every call appends the paths to `.git/kit/context.log`.
- The `agents/claude` package calls `kit context` through a hook before each edit, so the agent is shown the rules
  without asking.
- A repository's own rules sit beside the delivered ones, in `AGENTS.md` or a package of its own. Updates leave them
  alone.

see: [context.md](context.md)

## Checks

`kit ci` runs four checks and reports each one. The `github/ci` package delivers a GitHub Actions job that runs it on
every pull request, and any other pipeline can run it too.

| Check | Fails when |
| --- | --- |
| `kit gate` | the repository's gate fails, or it has none |
| `kit check` | a delivered file differs from the release, or a generated region differs from its command's output |
| `kit linked` | the pull request is not linked to an open issue |
| `kit policy` | a branch name, commit subject, or pull request title breaks the naming policy |

`kit lint` runs each rule's linter over the staged files, or any files, before a commit or at any time. It needs no
hook manager; the `hooks/lefthook` package runs it from lefthook.

The four checks are fixed. What each one runs is found at run time, from the repository's files:

| What | Found from |
| --- | --- |
| the gate | the `gate` setting, else a `ci` recipe in the root justfile, a `ci` target in a Makefile, or a `ci` script in `package.json` |
| the linters | the `lint:` command in the frontmatter of each rule and style that governs a file |
| the kit's own checks | the names under `checks:` in that frontmatter; the workflows rule names `pinned`, which refuses an action not pinned to a full commit SHA |
| the templates | the `templates:` list in that frontmatter |
| the policy engine | Open Policy Agent when `.kit/policy/` holds a `.rego` file, else the patterns in `.kit/settings.yml` |
| the toolchains | the `toolchains` setting, else the files present, such as `mise.toml` or a `Brewfile` |

A repository that adds a rule or a style adds its linter, its checks, and its templates with it.

see: [architecture.md](architecture.md#the-checks), [policy.md](policy.md)

## Templates

A template is a file written in kit's template language, which has one construct: the hole, `«name»`. The same
template both generates files and validates them.

- `kit new file "${path}"` generates a file from the template of the style that governs it. It fills the holes from
  the path and from flags, and prints the rules to read before writing the rest.
- `kit lint` validates every governed file against its style's templates, and names the line where a file leaves the
  closest one.
- A style package delivers the templates for its kind of file, such as a bash script, a justfile, or a workflow. A
  repository replaces one through a package of its own, as it replaces any delivered file.

see: [templates.md](templates.md)

## The workflow

An agent works from an issue to a merged pull request, in a worktree of its own. It never changes the desk, the
root checkout on the human's machine. The human at the desk is not bound by these limits.

1. `kit issue <N>` creates the worktree `worktrees/<N>-<slug>`, a branch linked to the issue, and a draft pull
   request. It moves any uncommitted work on the desk into the new worktree, and never switches the desk's branch.
2. `kit commit` commits what is staged, after the checks a commit's rule requires: the linters over the staged paths,
   no path the kit placed, and a subject the policy allows, with the issue's reference supplied from the branch. It
   refuses the desk and the default branch.
3. `kit submit` commits the same way when something is staged, then pushes the branch, setting the upstream the
   first time. It never forces: when origin has moved, the remedy is `kit rebase`.
4. `kit rebase` rebases the worktree's branch onto the default branch at origin and pushes it. It refuses the desk,
   the default branch, and a worktree with uncommitted changes.
5. `kit gate` runs the gate locally, with the same tools CI uses.
6. A human merges, and `kit tidy` removes the finished worktrees and the branches whose upstream is gone.

`kit status` shows everything in flight: what the last fetch brought in, the desk and every worktree with its
uncommitted paths and commits, every open pull request with its state, age, and checks, and the gone branches.

`kit todo` lists the open issues, oldest first, each with its assignees and what is already in flight for it: the open
change that closes it, or the branch for it on origin. The next issue to take up is one with nothing in flight.

The `agents/claude` package delivers a deny list that refuses the raw git commands these verbs replace, such as
`git commit`, `git push`, `git rebase`, `git worktree add`, and `git branch -D`.

see: [architecture.md](architecture.md#the-workflow-and-the-gate)

## Forges and trackers

- A forge is the service that hosts the repository. kit works with GitHub through `gh` and with GitLab through
  `glab`, and shows the same picture on either.
- A tracker holds the issues. The reference picks it: `#7` for the forge's issues, `AB#4521` for an Azure DevOps
  work item. A repository can use more than one at once.
- Only the workflow verbs call a forge. A repository with none can still install, update, check, and run
  `kit context`.

see: [trackers.md](trackers.md), [architecture.md](architecture.md#forges-seams-and-providers)

## Toolchains

- `kit toolchain` installs, lists, and updates the tools the gate runs, whether mise, Homebrew, or a container image
  manages them.
- The gate runs with those tools on the desk, in a worktree, and on a CI runner alike.

see: [toolchains.md](toolchains.md)

## Packages

Everything kit delivers comes from a package. The `kit` package, the entry file and the rules, is always active;
the others are optional, such as one style package per kind of file, the CI job, the pre-commit hook, and the
Claude Code adapter.

- `kit install` and `kit package add` activate packages, with the packages they depend on. `kit package remove`
  deactivates one and takes out everything it placed. `kit package list` shows the active set.
- A bundle, such as `empowerite`, activates its members. Any member can be removed on its own, and stays out
  through later updates.
- A repository defines packages of its own under `.kit/packages/<name>/`, in the same format. One can add rules,
  styles, or templates the release does not have.
- To replace a delivered file, a repository's package depends on the package that delivers it and places its own
  file at the same path. Where two active packages deliver the same path, the later in dependency order wins, so
  the repository's file is used and every other file still comes from the release.
- A value a delivered template asks for, such as the CI runner, is set under `answers:` in `.kit/settings.yml`.

A file kit shares with the repository is changed in the least intrusive way its format allows:

| When the file | kit uses |
| --- | --- |
| has a format that supports includes, such as `lefthook.yml` | an include line the repository adds, which `kit check` requires |
| has a format with comments, such as `AGENTS.md` | a region |
| supports neither, such as `.claude/settings.json` | a merge by key, which keeps the repository's own keys |
| belongs entirely to kit | a link or a read-only copy under `.kit/` |

see: [packages.md](packages.md), the "Changing what kit adds" section of `.kit/1st.md`

## Releases and updates

- A kit release is one tarball, `kit-<version>.tar.gz`, published from the tag `release/<version>` on this
  repository. A repository pins it like any other tool: with mise, with Homebrew, or by hand.
- `kit update` moves a repository to the release it pins. It replaces a delivered file left as placed, merges an
  edited one three ways against the previous release, and keeps the repository's own deny list entries and settings
  keys. When it lacks the previous release for a file it must merge, it refuses before writing anything.
- What kit places under `.kit/` is ignored by git, and the active packages are declared in `.kit/settings.yml`.
  `kit hydrate` places the files from that declaration in a fresh clone or worktree, and every verb that reads a
  delivered file hydrates first. With mise, a hook can hydrate as a shell enters the checkout.

see: [packages.md](packages.md), the Release section of [README.md](../README.md#release)

## Regions

- A region is a part of a file that a command writes, between a `kit:generate` line and a `kit:end` line. The rest
  of the file stays the repository's own.
- `kit generate` rewrites every region from its command. `kit check` fails on a stale one.
- The kit's own sentence in `AGENTS.md` and its `/worktrees/` line in `.gitignore` are regions, and a repository can
  add regions of its own, such as a table built from its manifests.

see: [generate.md](generate.md)

## Extending

- Modules extend how kit works, as packages extend what it can do in a repository. The verbs share modules, one
  directory per language under `modules/`: bash modules, loaded with `use`, and jq modules, imported by the jq
  programs the bash modules run and tested with `jq --run-tests`.
- A repository can add bash modules, or override a function, in its own directory on the module path.
- Each forge, tracker, toolchain, and policy engine sits behind a seam: one file whose header lists the calls a
  provider answers, and whose code picks the provider at run time. A new provider is a module that answers those
  calls.

see: [modules.md](modules.md), [architecture.md](architecture.md#forges-seams-and-providers)

## What no check covers

No check verifies that an agent followed a rule after reading it. The linters and templates cover the mechanical
part for the kinds of file they handle. The rest is the report in the pull request body, which the integrity rules
in `.kit/1st.md` govern.

see: [architecture.md](architecture.md#what-verifies-what)
