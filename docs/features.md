# Features

kit is a versioned release of house rules for agents, and of the checks that hold agents and people to them. This
page lists what it does, grouped by purpose, with a pointer to the page that describes each part in full.

## Rules where agents look

- `.kit/1st.md` is the file every agent reads first. It sets the integrity rules for reporting, the desk that no
  agent changes, the kit commands that replace raw git commands, and the rules by kind of file. The `workflow`
  package adds `.kit/workflow.md` beside it: the workflow, plans as prose, and the worker contract.
- The `agents/generic` package writes a region into `AGENTS.md`, a heading and two sentences that send every agent
  to `.kit/1st.md`, and every `agents/<harness>` package depends on it. Claude Code reads `AGENTS.md` on its own
  only when the project has no `CLAUDE.md`, so the `agents/claude` package imports it from `.claude/CLAUDE.md`.
- Each kind of file is defined once, by its `name:`, in a kinds file under `.kit/kinds/`: the paths and first lines
  that make a file of that kind, and the kinds found inside its files. A kind may have a rule,
  `.kit/rules/<kind>.md`, that names the kind and the linter it runs. `any`, `text`, and `yaml` have none of their
  own; `kit context` prints the rules of every kind a file is of. A style package adds `.kit/styles/<kind>.md`, how
  a file of that kind looks, with the same frontmatter.
- `kit context` takes one or more paths and prints the rules that govern them, each rule once. `--json` prints the
  rules per path. Every call appends the paths to `.git/kit/context.log`.
- The `agents/claude` package calls `kit context` through a hook before each edit, so the agent is shown the rules
  without asking.
- A repository's own rules sit beside the delivered ones, in `AGENTS.md` or a package of its own. Updates leave them
  alone.

see: [context.md](context.md)

## Checks

`kit ci` runs four checks and reports each one. The `github/ci` package delivers a GitHub Actions job that runs it on
every pull request, and any other pipeline can run it too. On a desk it passes only once the branch has a pull
request: before one exists, `kit linked` fails, and the other three run without one.

| Check | Fails when |
| --- | --- |
| `kit gate` | the repository's gate fails, or it has none |
| `kit check` | a delivered file differs from the release, or a generated region differs from its command's output |
| `kit linked` | the pull request is not linked to an open issue |
| `kit policy` | a branch name, commit subject, pull request title, or file in the tree breaks the policy |

`kit lint` runs each rule's linter over the staged files, or any files, before a commit or at any time. It does not
need a hook manager; the `hooks/lefthook` package runs it from lefthook.

The four checks are fixed. What each one runs is found at run time, from the repository's files:

| What | Found from |
| --- | --- |
| the gate | the `gate` setting, else a `ci` recipe in the root justfile, a `ci` target in a Makefile, or a `ci` script in `package.json` |
| the linters | the `lint:` command in the frontmatter of each rule and style that governs a file |
| the kit's own checks | the names under `checks:` in that frontmatter; the workflows rule names `pinned`, which refuses an action not pinned to a full commit SHA |
| the templates | the `templates:` list in that frontmatter |
| the policy engine | the `policy_engine` setting, else Open Policy Agent when `.kit/policy/` holds a `.rego` file, else the patterns in `kit.yml` |
| the toolchain | the one active or usable where kit runs, such as mise with a `mise.toml`, else the person's choice |

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
root checkout on the human's machine. The human at the desk is not bound by these limits. The `workflow` package
delivers these steps to agents as `.kit/workflow.md`; a repository that leaves it out writes its own in `AGENTS.md`.

1. `kit work <N>` creates the worktree `worktrees/<N>-<slug>`, a branch linked to the issue, and a draft pull
   request. It moves any uncommitted work on the desk into the new worktree, and never switches the desk's branch.
   It refuses an issue whose branch the desk has checked out, before it assigns, stashes, or commits anything. When
   it stops before the worktree has the swept work, the work goes back to the desk, staged work staged; when the
   work conflicts in the worktree, it stays in the stash, and `kit work` names the command that applies it.
2. `kit commit` commits what is staged, after the checks a commit's rule requires: the linters over the staged paths,
   no path the kit placed, and a subject the policy allows. It supplies the issue's reference from the branch, and
   refuses the desk and the default branch. A subject that names another issue adds that issue to the ones the pull
   request closes, and assigns it to the caller when no one is assigned.
3. `kit submit` commits the same way when something is staged, then pushes the branch, setting the upstream the
   first time. It never forces: when origin has moved, the remedy is `kit rebase`.
4. `kit rebase` rebases the worktree's branch onto the default branch at the primary remote and pushes it to its own
   name, never where `push.default` points. When origin's branch has commits the worktree lacks, it takes them
   first, and it forces only over the tip it read, so a commit pushed meanwhile is never overwritten. It refuses the
   desk, the default branch, and a worktree with uncommitted changes.
5. `kit gate` runs the gate locally, with the same tools CI uses.
6. A human merges, and `kit tidy` removes the finished worktrees and the branches whose upstream is gone.

`kit flow` draws where the work stands. Work moves left to right through eight stages:

- `backlog`: the open issues not closed by any open pull request, oldest first, each with its assignees, its age, and
  the branch started for it on the primary remote
- `underway`: the open issues closed by an open pull request, one row per issue and pull request: the assignees, the
  pull request and its state, and where the work is, here or on the author's desk
- `incoming`: what fetching every remote brought, the primary first: commits on each default branch, branches new
  and gone, and new tags. It never rebases, merges, or pulls.
- `desk`: the root checkout, wherever kit runs in the repository, with its commits and its uncommitted paths
- `worktrees`: every linked worktree, each with its commits and its uncommitted paths
- `changes`: every open pull request with its state, age, and checks, and under it the issues it closes
- `builds`: the runs not owned by any pull request, newest first: pushes to the default branch or a tag, and scheduled
  or manual runs. Each shows its state, number, commit, and title, with its age and duration; a failed job names the
  step that failed
- `releases`: the releases on the code forge, newest first

On a terminal the picture is interactive. Elsewhere, or with `--text`, each stage prints under a heading with its
glyph and its count. `kit flow <stage>` prints one stage, with that stage's flags: `kit flow backlog --today`
keeps the issues opened in the last 24 hours. `--json` prints the picture as one object keyed by stage, or one stage's
document. The `flow` setting in `kit.yml` names the stages drawn, in order.

A repository can have many remotes, such as a fork as `origin` and the shared repository as `upstream`. One is the
primary: the `primary` setting in `kit.yml`, else `origin`. Ahead and behind, the default branch, pull requests,
builds, releases, and issues are read from it. `kit work` and `kit submit` still fetch from and push to `origin`,
whatever the primary is. Each remote has a code forge, or none:

1. the `forges` setting names it: `forges: {upstream: gitlab}`; for the primary, the `forge` setting wins
2. else its host: `github.com` is GitHub, `gitlab.com` is GitLab
3. else a host `gh` or `glab` has logged in to, which tells a self-hosted code forge apart
4. else it has no code forge: kit fetches it and counts its branches, and asks no code forge about it

`kit release <version>` tags HEAD as `release/<version>` and pushes that tag alone, which starts the repository's
release workflow. It refuses:

- a version of the wrong shape
- a tag that exists
- a release, one with no suffix, whose commit is not on the default branch at the primary remote
- a version not higher than every release on the primary remote, unless `--lower` confirms it

`kit flow releases` lists the releases on the code forge, newest first, with each one's kind and age.

`kit workspace` draws what git holds across a workspace, a stage for each of git's entities: remotes, branches,
worktrees, commits, and tags. It reads git alone and asks no remote, so a remote's side is what the last fetch left.
Branches pair each remote's branch with the local ones that pull from it, worktrees list the desk first, commits are
the ones the local branches hold and the default branch does not, and tags tell an annotated tag from a lightweight
one. `kit workspace <stage>` prints one, and `--workspace` spans a group's members.

The `agents/claude` and `agents/gemini` packages each deliver a deny list that refuses the raw git commands these
verbs replace, such as `git commit`, `git push`, `git rebase`, `git worktree add`, and `git branch -D`. It matches
literal patterns, so a form it does not list, such as `git -C <dir> commit`, passes. Other harnesses get none.

see: [architecture.md](architecture.md#the-workflow-and-the-gate)

## Welcome

`kit welcome` is where a newcomer starts. It is a guide in seven steps:

1. Intro: what the guide does.
2. Before toolchains: bash, git, jq, and yq, which kit needs before it can look for a toolchain.
3. Toolchains: the project toolchain and the system toolchain, one check each. Several usable means you choose.
4. Setup: one check per line for the linters, each code forge's login, and the repository.
5. The lay of the land: how work moves from an issue to a release, through the stages of `kit flow`.
6. Terms: desk, worktree, code forge, package, rule, and the rest, one or two sentences each.
7. Make it yours: the settings teams change first, packages, a house package, and the docs.

- On a terminal, it opens in kittui, on the first step with a failed check. A failed check offers its fix: now runs
  it, later moves on.
- Without a terminal, or with `--text`, it prints each page with its checks, and does not run anything. It exits 1
  when a check fails.
- A missing tool's fix is `kit tools install`, or, when ambient, the system toolchain's command. No tool manager is
  assumed.
- `kit welcome --checks --json` prints the checks alone; `kit welcome --fix <check>` runs one fix and checks again.
- Each step is a page under `.kit/welcome/`, from the `kit` package. A house package replaces or adds a page.

## The terminal UI

`kit` typed with no verb opens a palette of every verb with its one-line purpose, as a tree under the groups
`kit help` lists them in, filtered as you type.

- The up and down arrows move over the verbs, never onto a heading. Shift with either jumps to the first verb of
  the previous or the next group.
- The right arrow shows the highlighted verb's help page beside the names; the left arrow shows the purposes again.
- Enter opens that verb's form. A verb run at a terminal without an argument it needs opens the same form instead
  of refusing, filled with what was given, as `kit commit` with no message does.
- The form has one field per part of the verb's usage line, and offers the values the repository has: changed
  paths, packages, templates. It shows the command line it builds, and runs it from Enter on that line, leaving
  that command line in the scrollback above the verb's output. A field the command line needs carries a dim
  `required` tag until it has an answer.
- `kit packages select` shows the whole set of packages, each marked by what will happen to it.

The form reads everything from `kit syntax <verb>`, the verb's grammar as JSON, parsed from its header, so a new
verb needs only its header. `kit tui` runs `kittui`, a Go binary each release carries for linux and darwin
on amd64 and arm64; in a checkout of the kit it is built from `projects/tui/`. Without a terminal on stdin, stdout,
and stderr, nothing opens: a verb short of an argument refuses in one line, as before.

see: [modules.md](modules.md#writing-a-verb)

kittui's app menu, under `=` on every screen's title line, lists the guides, then the apps. Each app is built around
one entity: its first tab is that entity's home, and its other tabs the entities around it. A tab kit has no data
source for yet says so, and loads nothing.

| App | Its entity | Its tabs |
| --- | --- | --- |
| Issues | the issue | backlog, underway, iterations, milestones |
| Changes | the pull or merge request | changes, reviews, checks, merged |
| Repositories | the repository | repos, worktrees, branches, commits, tags, remotes, incoming |
| Delivery | the release | releases, builds, artifacts, environments, deployments |
| People | the person | people, teams, agents |
| Trust | the evidence | evidence, findings, controls, waivers, policy |
| Platform | the service | services, toolchains, tools, settings |

A link to a row of another app's tab opens that app, on the tab, with the row under the cursor once it loads: an
underway issue's pull request leads to Changes, the checkout it is worked in to Repositories' worktrees, and a
member's name, over a group, to Repositories' repos.

`kit flow` opens Issues, and `kit flow <stage>` the app with that tab, on the tab; `kit flow desk` is Repositories'
worktrees, the desk first. `kit workspace` opens Repositories. Iterations, milestones, reviews, checks, merged,
artifacts, environments, deployments, and every tab of People, Trust, and Platform have no data source yet.

## Code forges and trackers

- A code forge is the service that hosts the repository. kit works with GitHub through `gh` and with GitLab through
  `glab`, and shows the same picture on either.
- A tracker holds the issues. The reference picks it: `#7` for the code forge's issues, `AB#4521` for an Azure DevOps
  work item. A repository can use more than one at once.
- Only the workflow verbs and two checks, `kit linked` and `kit policy`, call a code forge. A repository with none
  can still install, update, check, and run `kit context`.

see: [trackers.md](trackers.md), [architecture.md](architecture.md#code-forges-seams-and-providers)

## Workspaces

- A workspace is one repository, or a named group of them. `kit workspaces` lists your groups and favorites, shows a
  group's members with their checkouts, and edits `kit/workspaces.yml`, your comments kept.
- Inside a repository, kit starts on that repository; outside one, on the last group used.

see: [workspaces.md](workspaces.md)

## Toolchains

- `kit toolchains` lists the toolchains kit found where it runs, and records which one to use.
- `kit tools` lists, installs, and updates the tools that toolchain pins, whether mise, nix, Homebrew, docker, or a
  dev container manages them, and the packages npm or uv pins on top. apt, dnf, and pacman can fill the system
  role: they supply what kit itself runs on.
- The gate runs with those tools on the desk, in a worktree, and on a CI runner alike.

see: [toolchains.md](toolchains.md)

## Packages

Everything kit delivers comes from a package. The `kit` package, the entry file and the rules, is the default:
`kit install` with no names brings it. `kit packages remove kit` takes it out once no other active package depends
on it, and lists it under `excludes:`, so `kit update` leaves it out. The others are optional: the
`workflow` package, one style package per kind of file, the CI job, the pre-commit hook, and an adapter for each of
Claude Code, Cursor, GitHub Copilot, and Gemini CLI.

- `kit install` and `kit packages add` activate packages, with the packages they depend on. `kit packages remove`
  deactivates one and takes out everything it placed. `kit packages` lists them, the active set
  among them.
- A bundle, such as `empowerite`, activates its members. Any member can be removed on its own, and stays out
  through later updates.
- A repository defines packages of its own under `.kit/packages/<name>/`, in the same format. One can add rules,
  styles, or templates the release does not have.
- To replace a delivered file, a repository's package depends on the package that delivers it and places its own
  file at the same path. Where two active packages deliver the same path, the later in dependency order wins, so
  the repository's file is used and every other file still comes from the release.
- A value a delivered template asks for, such as the CI runner, is set under `answers:` in `kit.yml`.

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
- What kit places under `.kit/` is ignored by git, and the active packages are declared in `kit.yml`.
  `kit hydrate` places the files from that declaration in a fresh clone or worktree, and every verb that reads a
  delivered file hydrates first. With mise, a hook can hydrate as a shell enters the checkout.

see: [packages.md](packages.md), the Release section of [README.md](../README.md#release)

## Regions

- A region is a part of a file that a command writes, between a `kit:generate` line and a `kit:end` line. The rest
  of the file stays the repository's own.
- `kit generate` rewrites every region from its command. `kit check` fails on a stale one.
- The kit's block in `AGENTS.md`, a heading and two sentences, is a region, and so is its `/worktrees/` line in
  `.gitignore`. A repository can add regions of its own, such as a table built from its manifests.

see: [generate.md](generate.md)

## Extending

- Modules extend how kit works, as packages extend what it can do in a repository. The verbs share modules, one
  directory per language under `modules/`: bash modules, loaded with `use`, and jq modules, imported by the jq
  programs the bash modules run and tested with `jq --run-tests`.
- A repository can add bash modules, or override a function, in its own directory on the module path.
- Each code forge, tracker, toolchain, dependency provider, and policy engine sits behind a seam: one file whose
  header lists the calls a provider answers, and whose code picks the provider at run time. A new provider is a
  module that answers those calls.

see: [modules.md](modules.md), [architecture.md](architecture.md#code-forges-seams-and-providers)

## What no check covers

No check verifies that an agent followed a rule after reading it. The linters and templates cover the mechanical
part for the kinds of file they handle. The rest is the report in the pull request body, which the integrity rules
in `.kit/1st.md` govern.

see: [architecture.md](architecture.md#what-verifies-what)
