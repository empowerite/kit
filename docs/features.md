# Features

kit is a versioned release of house rules for agents, and of the checks that hold agents and people to them. This
page lists what it does, grouped by purpose, with a pointer to the page that describes each part in full.

## Rules where agents look

- `.kit/1st.md` is the file every agent reads first. It sets the integrity rules for reporting, the workflow, the
  worker contract, and the kit commands that replace raw git commands.
- `install` writes a sentence into `AGENTS.md` that sends every agent to `.kit/1st.md`. The `agents/claude` package
  makes Claude Code read `AGENTS.md` as well.
- Each kind of file has one rule, `.kit/rules/<kind>.md`, that names the paths it governs and the linter it runs. A
  style package adds `.kit/styles/<kind>.md`, how a file of that kind looks, with the same frontmatter.
- `kit context "${path}"` prints the rules that govern a path, and appends the request to `.git/kit/context.log`. The
  `agents/claude` package calls it through a hook before each edit, so the agent is shown the rules without asking.
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

- The gate is the repository's own CI command: the `gate` setting, a `ci` recipe in the root justfile, a `ci` target
  in a Makefile, or a `ci` script in `package.json`.
- The naming policy is a set of patterns, or an Open Policy Agent policy with the `policy/opa` package.
- `kit lint` runs each rule's linter over the staged files, or any files, before a commit or at any time. It needs no
  hook manager; the `hooks/lefthook` package runs it from lefthook.
- `kit lint` also refuses a workflow action not pinned to a full commit SHA.

see: [architecture.md](architecture.md#the-checks), [policy.md](policy.md)

## Templates

- A style package delivers templates for its kind of file, such as a bash script, a justfile, or a workflow.
- `kit new file "${path}"` creates a file from the template of the style that governs it, fills its holes from the
  path and from flags, and prints the rules to read before writing the rest.
- `kit lint` checks that every governed file still has the shape of one of its templates, and names the line where a
  file leaves it.
- A repository changes a template by placing its own at the same path through a package of its own.

see: [templates.md](templates.md)

## The workflow

Work runs from an issue to a merged pull request, and never through the desk: the root checkout on the human's
machine, which no agent changes in place.

1. `kit issue <N>` creates the worktree `worktrees/<N>-<slug>`, a branch linked to the issue, and a draft pull
   request. It moves any uncommitted work on the desk into the new worktree, and never switches the desk's branch.
2. `kit rebase` rebases the worktree's branch onto the default branch at origin and pushes it. It refuses the desk,
   the default branch, and a worktree with uncommitted changes.
3. `kit gate` runs the gate locally, with the same tools CI uses.
4. A human merges, and `kit tidy` removes the finished worktrees and the branches whose upstream is gone.

`kit status` shows everything in flight: what the last fetch brought in, the desk and every worktree with its
uncommitted paths and commits, every open pull request with its state, age, and checks, and the gone branches.

The `agents/claude` package delivers a deny list that refuses the raw git commands these verbs replace, such as
`git rebase`, `git worktree add`, `git branch -D`, and `git push --force`.

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

## Delivery

- A release is one tarball, pinned like any other tool: with mise, with Homebrew, or by hand.
- Everything a release delivers comes from a package. `kit install` and `kit package add` activate packages, and
  `kit package remove` deactivates them. A bundle, such as `empowerite`, activates its members, and any member can be
  removed on its own.
- `kit update` moves a repository to the pinned release. It replaces a delivered file left as placed, merges an
  edited one three ways against the previous release, and keeps the repository's own deny list entries and settings
  keys. When it lacks the previous release for a file it must merge, it refuses before writing anything.
- What kit places under `.kit/` is ignored by git and declared in `.kit/settings.yml`. `kit hydrate` places it from
  that declaration in a fresh clone or worktree, and every verb that reads a delivered file hydrates first. With
  mise, a hook can hydrate as a shell enters the checkout.
- A package of the repository's own, under `.kit/packages/`, works like a delivered one. It can add rules, styles, or
  templates, or replace a delivered file.

see: [packages.md](packages.md)

## Regions

- A region is a part of a file that a command writes, between a `kit:generate` line and a `kit:end` line. The rest
  of the file stays the repository's own.
- `kit generate` rewrites every region from its command. `kit check` fails on a stale one.
- The kit's own sentence in `AGENTS.md` and its `/worktrees/` line in `.gitignore` are regions, and a repository can
  add regions of its own, such as a table built from its manifests.

see: [generate.md](generate.md)

## Extending

- The verbs share bash modules loaded with `use`. A repository can add modules or override a function in its own.
- Each forge, tracker, toolchain, and policy engine sits behind a seam: one file that defines the calls and picks
  the provider at run time. A new provider implements the same calls.

see: [bash.md](bash.md), [architecture.md](architecture.md#forges-seams-and-providers)

## What no check covers

No check verifies that an agent followed a rule after reading it. The linters and templates cover the mechanical
part for the kinds of file they handle. The rest is the report in the pull request body, which the integrity rules
in `.kit/1st.md` govern.

see: [architecture.md](architecture.md#what-verifies-what)
