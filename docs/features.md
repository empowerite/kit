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
| `kit policy` | a branch name, worktree, commit subject, pull request title, or file in the tree breaks the policy |

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
   request closes, and assigns it to the caller when no one is assigned. A commit from an index of its own
   (`GIT_INDEX_FILE`) brings the shared index up to it, for the paths it changed alone.
3. `kit submit` pushes the branch, setting the upstream the first time, and commits nothing: what is staged stays
   staged, for `kit commit`. It never forces: when the push remote has moved, the remedy is `kit rebase`. It pushes
   the branch's lease too, to the same remote, so CI holds the branch to it.
4. `kit rebase` rebases the worktree's branch onto the default branch at the primary remote and pushes it to its own
   name, never where `push.default` points. When the remote's branch has commits the worktree lacks, it takes them
   first, and it forces only over the tip it read, so a commit pushed meanwhile is never overwritten. It refuses the
   desk, the default branch, and a worktree with uncommitted changes.
5. `kit gate` runs the gate locally, with the same tools CI uses.
6. A human merges, and `kit tidy` removes the finished worktrees and the branches whose upstream is gone.

Sessions that work at once say which paths each will change with leases, so kit can tell them who holds what.
A lease is a ref every worktree of the clone sees, `refs/kit/leases/<N>`, holding the paths an issue's work claims,
who took it, on which branch, and until when: an hour after it is taken or renewed, by default. `kit work <N>` takes
one for the paths the issue lists under a `Paths` heading, or `**`. `kit leases` lists them, and `kit leases take`,
`renew`, and `release` change them, each as a compare and swap, so two claims at once are judged one after the
other. A claim that may overlap another issue's live lease is refused, naming who holds it; the leases on one branch
nest in the branch's own lease, the one of the issue its name starts with. `kit commit` refuses a staged path another
issue's lease on the branch holds, unless the commit names that issue, and renews the leases of the issues it names.
`kit policy`, in CI, refuses a path the branch changed outside its own lease. A lease is not a lock: it only says who
holds what.

see: [the leases design](superpowers/specs/2026-11-12-leases-design.md)

`kit flow` draws where the work stands. Work moves left to right through five stages:

- `issues`: every open issue of every tracker, the code forge's own and each `kit.yml` declares, by its reference,
  `#7` or `AB#4521`, oldest first, each with its assignees, its age, its milestone, iteration, and swimlane,
  and its status. One underway has a row per pull request that closes it: the pull request and its state, and where
  the work is, here or on the author's desk. One no pull request closes is underway too while a branch is named for
  it, `<N>-<slug>`, on the desk or the primary remote: its row names the branch. One with neither is in the backlog.
  A forge that lacks milestones, iterations, or swimlanes can take each from a label, or a default, as
  `forge_settings.<forge>.stand_ins` in `kit.yml` says.
- `changes`: every open pull request with its state, age, and checks, and under it the issues it closes
- `builds`: the runs not owned by any pull request, newest first: pushes to the default branch or a tag, and scheduled
  or manual runs. Each shows its state, number, commit, and title, with its age and duration; a failed job names the
  step that failed
- `incoming`: what fetching every remote brought, the primary first: commits on each default branch, branches new,
  updated, force-pushed, and gone, new tags, and each local branch now behind its upstream. It never rebases, merges,
  or pulls. Its JSON holds every one as a flat row, which kittui draws as a table that sorts, groups, filters, and
  keeps presets, as every stage does.
- `releases`: the releases on the code forge, newest first

The desk and its worktrees are no stage of `kit flow`: `kit workspace worktrees` draws them, each with its commits
and its uncommitted paths.

On a terminal the picture is interactive. Elsewhere, or with `--text`, each stage prints under a heading with its
glyph and its count. `kit flow <stage>` prints one stage, with that stage's flags: `kit flow issues --today`
keeps the issues opened in the last 24 hours. `--json` prints the picture as one object keyed by stage, or one stage's
document. The `flow` setting in `kit.yml` names the stages drawn, in order.

A repository can have many remotes, such as a fork as `origin` and the shared repository as `upstream`. One is the
primary: the `primary` setting in `kit.yml`, else `origin`. Ahead and behind, the default branch, pull requests,
builds, releases, and issues are read from it, and `kit rebase` and `kit release` read it. A branch goes to the push
remote: the `push` setting in `kit.yml`, as `push: origin` for a fork, else the primary. `kit work` and `kit submit`
fetch from it, look a branch up on it, fast-forward from it, and push to it, and the pull request opens on the
primary from the branch there. A `push` that names no remote is refused, as a `primary` that names none is. Each remote
has a code forge, or none:

1. the `forges` setting names it: `forges: {upstream: gitlab}`; for the primary, the `forge` setting wins
2. else its host: `github.com` is GitHub, `gitlab.com` is GitLab, `dev.azure.com` is Azure DevOps
3. else, on a CI runner, the runner's own server when its host is the remote's: `GITHUB_SERVER_URL` names GitHub,
   `CI_SERVER_HOST` GitLab, and `SYSTEM_COLLECTIONURI` Azure DevOps
4. else a host `gh` or `glab` has logged in to, which tells a self-hosted code forge apart
5. else it has no code forge: kit fetches it and counts its branches, and asks no code forge about it

The primary is the exception to the last: on a host nothing names, its code forge is one kit cannot tell, never
none, so a read of its pull request fails and names the `forge` setting. A primary that is a path on this machine has
no code forge. `forge: none`, or `none` under `forges`, says a remote has no code forge the kit knows, as on Bitbucket
or a plain git server; it wins over the host and a runner's server.

`kit release <version>` tags HEAD as `release/<version>` and pushes that tag alone, which starts the repository's
release workflow. It refuses:

- a version of the wrong shape
- a tag that exists
- a release, one with no suffix, whose commit is not on the default branch at the primary remote
- a version not higher than every release on the primary remote, unless `--lower` confirms it

### Publishing, and pull behind push

The repository names its own artifacts under `publish:` in `kit.yml`: what the runner built, what each is on the
pull path, as a tool pin says it, and the push targets a release and a pre-release go to. A target is never searched;
the pull path is what consumers search.

```yaml
publish:
  kit:
    files: ["dist/kit-{version}.tar.gz"]
    release: empowerite/kit
    tag: release/{version}
    asset: kit-{version}.tar.gz
    targets:
      release: [github-release, vcs-tag]
      prerelease: [github-release]
    verify: [corp-files]
```

- A target is a provider of the `targets` seam, `modules/bash/targets/<kind>.sh`, written as its kind or as a map
  of its kind and its keys: `vcs-tag` pushes the tag `v<version>` (`prefix:` to change it), as a Go module is
  published; `github-release` makes the release with gh (`repo:`); `artifactory-local` uploads with jf
  (`repository:`, `path:`, `url:`); and `npm-publish` publishes with npm (`registry:`), a pre-release under `next`.
  Each runs its connector, the tool its service ships; kit never speaks a service's upload API, and the runner
  carries the credentials.
- `kit publish <version>` pushes each artifact to its targets. It runs on a runner alone, where `GITHUB_ACTIONS`,
  `GITLAB_CI`, or `TF_BUILD` is set; a pre-release may be published from a desk, at its own risk. It refuses a
  version whose tag `kit release` has not pushed, a file not built, a connector not on `PATH`, and, before it pushes
  to any, a target that holds the version already: a version is never overwritten. `--retire` takes a version back
  where a target can: delete is retire.
- The push is done when the targets take it. `kit publish` then tags what it pushed as `release-hashes/<version>`, a
  tag of a blob of one line per artifact and its hash, so anyone who can read the forge can tell the same bytes from
  others.
- Visibility is a state kit watches. For each artifact and each pull source that serves its form, or the sources its
  `verify:` names, a release is visible, pending (stuck once older than the source's `expect:`), blocked (the source
  says it never will serve it, as a scan's block), or mismatch (served with other bytes). Answers are kept under
  `XDG_CACHE_HOME/kit/visibility/`: visible, blocked, and mismatch for good, pending for five minutes.
- `kit flow releases` shows the state of the five newest releases on each row, says when one is stuck, and with
  `--json` each row carries `visibility`, a state per artifact and source. In a consumer, `kit tools --outdated`
  adds what is tagged upstream that a mirror does not serve yet, and a pin to such a version fails, naming it pending.
- `kit release <version> --wait=<duration>` waits for visibility, for whoever wants that: it exits 0 when every source
  serves the release, 1 when one blocks it or serves other bytes, and 3 when the time passes with one pending. With
  the tag on the primary already, it only waits.

`kit flow releases` lists the releases on the code forge, newest first, with each one's kind and age. With `--json`,
each release is a row of its fields, flat: its version, tag, name, kind, whether it is the latest, when it was
published, its page, its notes, and its assets, each file's name and where it downloads. kittui draws the rows from
its catalog, so a preset shows or hides each field, and the table sorts, groups, and filters by it; a release's kind
leads to its page, and an asset to its download.

`kit workspace` draws what git holds across a workspace, a stage for each of git's entities: remotes, branches,
worktrees, commits, and tags. It reads git alone and asks no remote, so a remote's side is what the last fetch left.
Branches pair each remote's branch with the local ones that pull from it, worktrees list the desk first, commits are
the ones the local branches hold and the default branch does not, and tags tell an annotated tag from a lightweight
one. `kit workspace <stage>` prints one, and `kit --workspace=<group> workspace` spans a group's members.

The `agents/claude` and `agents/gemini` packages each deliver a deny list that refuses the raw git commands these
verbs replace, such as `git commit`, `git push`, `git rebase`, `git worktree add`, and `git branch -D`. It matches
literal patterns, so a form it does not list, such as `git -C <dir> commit`, passes. Other harnesses get none.
`kit audit enforcement` shows, for this repository and each harness, what binds an agent and what is advice; see
[architecture.md](architecture.md#enforcement-tiers).

see: [architecture.md](architecture.md#the-workflow-and-the-gate)

## Welcome

`kit welcome` is where a newcomer starts. It is a guide in five steps:

1. Introduction: what kit is, where it runs, and who it is for.
2. Toolchains: bash, git, jq, and yq, which kit runs on; then the machine's package manager. Several usable means you
   choose.
3. Setup: each code forge's login, once for the machine; then each folder of the workspace on its own, the desk then
   each worktree, by repository: kit installed, with the version its `kit.yml` declares, `.kit/` in place, the hooks,
   its tool manager, its linters, and `kit check`.
4. Customize: the settings teams change first, packages, a house package, and the docs.
5. Terms: the words kit uses, by topic, from workspace and tracker to CI, CD, and toolchain.

kit's screens and what each shows, Flow first, how work moves from an issue to a release, are a tutorial that ships
with the kit, `kit tutorials applications`; see [tutorials.md](tutorials.md).

- On a terminal, it opens in kittui, on the introduction. A step's tab counts its failed checks in red. A failed
  check offers its fix on the row under the cursor: ⧉, or `c`, copies it, and ▶️, or `r`, runs it where kit can. A
  folder's fix is shown and copied with `cd <folder> &&` in front, as every remedy kit prints is, so it runs wherever
  it is pasted.
- Without a terminal, or with `--text`, it prints each page with its checks, and does not run anything. It exits 1
  when a check fails.
- A check that holds can still offer a fix: an offer, drawn 💡, which needs no attention and fails nothing. kit runs in
  a folder whose `kit.yml` declares no kit version, on what it infers, so there kit installed holds and offers
  `kit install`.
- A missing tool's fix is `kit tools install`, or, when ambient, the package manager's command. No tool manager is
  assumed.
- `kit welcome --checks --json` prints the checks alone, a folder's with its folder, repository, branch, and kit;
  `kit welcome --fix=<check>` runs one fix and checks again, and `--folder=<path>` names a folder's check.
- The workspace is the one kit starts on, or `kit --workspace=<workspace> welcome`'s: a repository's desk and
  worktrees, or every member's. With none, only the machine is checked; see [workspaces.md](workspaces.md).
- Each step is a page under `.kit/welcome/`, from the `kit` package. A house package replaces or adds a page.

## The terminal UI

`kit` typed with no verb opens the welcome guide, as `kit welcome` does. `kit help` at a terminal opens a palette of
every verb, a table of its group, its command, and its description, filtered as you type.

- By default the rows are grouped by group and sorted by command, and the description shows its first line, the
  verb's one-line purpose. The table's presets, sorting, grouping, and filtering are those of every table in kittui.
- The right arrow opens the row under the cursor to the rest of the description: the verb's help page, fetched as
  the row opens. The left arrow closes it again.
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

kittui's app menu, under `=` on every screen's title line, lists the guides, then the apps. The first, Flow, is the
picture `kit flow` draws: every stage the `flow` setting in `kit.yml` names, in its order, from the issues to releases.
Each other app is built around one entity: its first tab is that entity's home, and its other tabs the entities around
it. Each app has a verb, `kit issues`, `kit changes`, `kit workspace`, `kit delivery`, `kit people`, `kit audit`
(Trust), `kit platform`, and `kit packages`, and Flow has `kit flow`; on a terminal one opens its app, and
`kit <verb> <tab>` opens it on that tab, maximized.

| App | Its entity | Its tabs |
| --- | --- | --- |
| Flow | the work, from the issues to releases | the `flow` setting's stages: issues, changes, builds, incoming, releases |
| Issues | the issue | issues, milestones, iterations |
| Changes | the pull or merge request | changes, reviews, checks, merged |
| Repositories | the repository | repos, worktrees, branches, commits, tags, remotes, incoming |
| Delivery | the release | releases, builds, artifacts, environments, deployments |
| People | the person | people, teams, agents |
| Trust | the evidence | evidence, findings, controls, waivers, policy, enforcement |
| Platform | the service | services, toolchains, tools, settings |
| Packages | the package | packages |

A link to a row of a tab the app does not have opens the app built around that tab's entity, on the tab, with the row
under the cursor once it loads: from Issues, the pull request of an issue underway leads to Changes, and the checkout
it is worked in to Repositories' worktrees; from any app but Repositories, a member's name, over a group, leads to
Repositories' repos.

A link to another app leaves a crumb on the title line of the app it opens: after the workspace menu, and after the
maximized tab's name when one is, a dim · and ↩ with the name of where the link was followed, the app and its tab.
A plain click on it, or ⌫, which the keys bar names while it shows, goes back there, the view as it was left: the
tab, maximized or not, the preset, sort, filters, and groups, the row under the cursor, and the scroll. That place
keeps its own crumb, so crumbs chain, each one deep. The workspace menu's Manage… leaves a crumb as a link does. Only
the app menu, `=`, opens an app with no crumb, and drops the chain; so does a switch to another workspace, which makes
the apps anew there. Switching tabs within an app keeps its crumb. Crumbs last for the session.

`kit flow` opens Flow, and `kit flow <stage>` opens it on that stage, maximized. `kit workspace` opens Repositories,
and `kit workspace <stage>` the app with that tab, on the tab. `kit platform` prints Platform's tabs, services,
toolchains, tools, and settings, each a stage, as [apps.md](apps.md) says; on a terminal it opens Platform in kittui.

`kit packages` alone, on a terminal, opens Packages, a row per package in reach, and `kit packages packages` opens it
on its one tab. Its tab menu adds, removes, and makes the only active ones the packages selected, each by running
`kit packages`; off a terminal, `kit packages` still lists them.

## Code forges and trackers

- A code forge is the service that hosts the repository. kit works with GitHub through `gh`, with GitLab through
  `glab`, and with Azure DevOps through `az`, and shows the same picture on each.
- A tracker holds the issues. The reference picks it: `#7` for the code forge's issues, `AB#4521` for an Azure DevOps
  work item. A repository can use more than one at once.
- Only the workflow verbs and two checks, `kit linked` and `kit policy`, call a code forge. A repository with none
  can still install, update, check, and run `kit context`.

see: [trackers.md](trackers.md), [architecture.md](architecture.md#code-forges-seams-and-providers),
[integrations/1st.md](integrations/1st.md)

## Workspaces

- A workspace is one repository, or a named group of them. `kit workspaces` lists your groups and favorites, shows a
  group's members with their checkouts, and edits `${XDG_CONFIG_HOME:-~/.config}/kit/workspaces.yml`, your comments
  kept.
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
