# Workspaces: design

Status: proposed, 2026-11-08.

## Intent

kit and kittui see one repository at a time: the one they were started in. Work rarely stays in one. The empowerite
repositories (kit, edge, enchiridion, infrastructure, jig) move together, and today the only way to see their tickets,
changes, and builds side by side is to open each one. A workspace is what kit is looking at: either one repository, or
a named collection of repositories treated as a group. The first thing a group workspace is for is a view of all the
tickets, repositories, builds, and the rest across every repository in it, on the Flow screen.

Much of what kit needs to know about a repository it can infer from the repository: its remotes and their forges from
git, and, when kit is installed there, its trackers, forges, and settings from `kit.yml` and `.kit/`. A workspace adds
only what cannot be inferred: which repositories belong together.

## Terms

- **Workspace**: one repository, or a named group of repositories.
- **Member**: a repository in a group workspace, named by its host and path, `github.com/empowerite/edge`. A path can
  have more than two segments on GitLab, `gitlab.com/group/subgroup/repo`.
- **Checkout**: a clone of a member on this machine. A member with none is shown as ☁️ (not cloned).
- **Definition**: how a group names its members: a fixed list, queries, a GitHub Project, and exclusions.

## Goals

- Pick the workspace from the 📚 control on the title line of every kittui screen: a favorite, anything on the machine,
  or a new one.
- See the whole workspace on Flow: a 📚 repos tab, and every other tab gathering its rows from every member.
- Define a group by a fixed list, by queries that keep matching as repositories appear, by a GitHub Project, or by a
  mix of them.
- Infer each member's settings from the member itself; the workspace file holds only membership.
- One startup rule, the same for kit and every kit application that needs a workspace.

## Non-goals

- Acting on many repositories at once (a commit, a release, or a rebase across the group). A group is for seeing; an
  action still runs in one member.
- Sharing workspace definitions with a team through kit. A definition is personal. A team can still share membership
  through forge topics, which a definition can query.
- Cloning a whole workspace in bulk.

## Anti-goals

- A repository joining a group with no rule its owner wrote. Topics, git config labels, and folder layout can
  feed a definition, or be offered as suggestions, but never add a member by themselves. Without this, people end up
  asking why a repository is in a group.
- Counting a ☁️ member's missing local data as zero. A total over the workspace must never look complete when it is
  not.
- A workspace that empties the forge's rate limit. A 40-member workspace has to stay usable all day.
- Changing what a single-repository user sees, except for the 📚 repos tab.

## What exists now

- The title line draws `📚 owner/repo` in the repository's magenta, separately in each screen:
  `projects/tui/internal/welcome/view.go` and `projects/tui/internal/flow/view.go`. The repository comes from
  `git remote get-url origin`, in `projects/tui/internal/flow/origin.go`, which keeps only the last two path segments.
- kittui keeps its per-user state in `~/.config/kit/` (`kittui.json`, `presets.json`, `search-history.json`), under
  `XDG_CONFIG_HOME` when it is set.
- `screens.Model` gives every screen made a `shell.SettingsMsg` when a setting changes. A workspace change can travel
  the same way.
- Flow's rows already carry `organization` and `repository` (`projects/tui/internal/flow/rows.go`), and its tables
  group by a field of several values, a row standing under each value.
- `kit flow` already spans checkouts in one case: run above desks, outside any repository, worktrees and changes draw
  each desk up to four levels below. It reads the forge at most once a minute and keeps the snapshot in the
  repository's common git directory.
- The kittui components design (`2026-11-07-kittui-components-design.md`) has moved menus to `shell.Dropdown`, popups
  to `shell.Layers`, dialogs to `shell.Placed`, and screen ends to `shell.View` and `shell.Notice`. Its next moves are
  `source`, a shared read of kit, and `table`.

## Decided

These were decided in the brainstorm, as stated.

1. A workspace is a repository or a named collection of repositories treated as a group.
2. The 📚 on the title line, when clicked, opens a popup to pick another workspace: from a list of favorites, from
   everything on the machine, or by creating a new one.
3. Started inside a repository, kit's workspace is that repository. Started outside one, it is the last group
   workspace used.
4. Settings are inferred from each repository: git config for remotes and forges, and `kit.yml` and `.kit/` where kit
   is installed.
5. Flow over a workspace has a 📚 repos tab, first, with a row per member and a count from each other tab. A count's
   click opens that tab showing only that member.
6. The 📚 repos tab is always present, a single repository's workspace included. It is where information about the
   members collects, and other tabs link back to it.
7. The 📚 repos tab's remotes are a field of several values, drawn as pills, as an issue's labels are.
8. The title shows `📚 <name> <count>` for a group, `📚 empowerite 5`, and nothing more than `📚 owner/repo` for a
   single repository.
9. A group's members can be a fixed list, queries with filters (visibility, a name pattern such as `homebrew-*`, and
   the like), or a GitHub Project, together.
10. Every repository a query matches is a member, cloned or not.
11. Over a group, Welcome shows every member's checks together, as Flow composes every member's rows, rather than
    following one member.

## Design

### The workspace file

Definitions are kept in `~/.config/kit/workspaces.yml` (under `XDG_CONFIG_HOME` when set), beside kittui's files. It
is YAML because people edit it by hand. Only kit reads and writes it, its bash through `yaml.sh` and `yq`, which keeps
a hand-written comment where it stands; kittui reads workspaces through `kit workspaces --json` alone and takes no YAML
dependency. The field names below are the design's; the exact set of query filters is the builder's to extend.

```yaml
# where checkouts are looked for, and where a clone goes
roots: [~/Code]
favorites: [empowerite, github.com/empowerite/kit]
workspaces:
  empowerite:
    repos:                                  # a fixed list
      - github.com/empowerite/kit
      - github.com/empowerite/edge
    match:                                  # queries; a repository matching any one is a member
      - owner: github.com/empowerite
        visibility: private                 # public | private | internal | all (the default)
      - owner: github.com/empowerite
        name: homebrew-*                    # a glob on the repository's name
      - topic: workspace-empowerite         # a forge topic
      - label: empowerite                   # a git config label on a checkout
    exclude: [github.com/empowerite/attic]
  roadmap:
    project: github.com/orgs/empowerite/projects/3
```

A group's members are its `repos`, together with every repository its `match` entries match and every repository its
`project`'s open items live in, whatever their owner, without its `exclude`. A repository whose only items in the
Project are closed is not a member through the Project. Within one `match` entry the filters must all hold; across entries,
any one is enough. Queries that name an owner ask the forge, so they match repositories that are not cloned. `label:`
reads checkouts alone.

A favorite is a group's name or a single repository's host and path. What kit keeps for itself, the last group used
and the query results, is not in this file; see [State and cost](#state-and-cost).

### Labels on repositories

There are two kinds, and a definition asks for each by name:

- **Topics**, the forges' public labels on a repository (GitHub and GitLab both have them), queried with `topic:`.
  They are shared with everyone who can see the repository, so they suit a group a team agrees on. A topic allows only
  lowercase letters, digits, and hyphens, so the convention is `workspace-<name>`, not `workspace:<name>`.
- **Git config labels**, `git config --add kit.workspace <name>`, in a checkout's `.git/config`, queried with
  `label:`. They are never committed, are personal, take several values, and cost nothing to read.

### Members and checkouts

A member is named by host and path, never by a folder: queries and Projects answer in host and path, and a folder
moves. kit finds a member's checkout by scanning `roots` (four levels down, as `kit flow` already looks for desks) for
repositories whose primary remote points at the member: the `primary` setting in its `kit.yml`, else origin, as
`kit flow` already chooses. Its other remotes never make it a checkout of another member, so a fork's clone, with the
original as upstream, counts once, under the fork. One member can have several checkouts; the first under the earliest
root is its checkout, and the 📚 repos tab names the others.

### Which workspace kit starts on

- Inside a repository, the workspace is that repository, even when it is a member of a group.
- Outside any repository, it is the last group workspace used.
- Outside any repository with no group ever used, there is no workspace. kittui's title shows 📚 alone, dim, and its
  dropdown; `kit flow` draws what it draws above desks today, unchanged.
- `--workspace <name>` overrides all three. `--workspace none` asks for no workspace, which gives back today's
  above-desks picture in a folder where a group would otherwise be drawn.

The rule lives in kit's bash, in one function the verbs share, so every kit application resolves the same workspace.
kittui asks kit for it rather than deciding itself.

Only `kit flow` and `kit workspaces` take `--workspace` and follow the rule in this design. The verbs that act
(`commit`, `work`, `submit`, and the rest) are unchanged: they run in the repository of the folder they are run in, or
refuse outside one, as now. Other verbs that read can adopt the shared function later, one at a time.

### `kit workspaces`

A verb owns workspaces, so the CLI and kittui agree and kittui reads them as it reads everything else, through kit
with `--json`. What it must do:

- list the workspaces and favorites;
- resolve a workspace: its members, each with its host and path, forge, remotes, checkout (or none), and the source
  that made it a member (fixed, a query, a Project);
- say which workspace kit would start on here, and why;
- record the last group used;
- create a group, add or remove a fixed member, and add or remove a favorite, writing the file.

On a terminal, `kit workspaces` with no arguments opens the Workspaces screen in kittui, as `kit flow` opens the picture.
The subcommands' names and the JSON's exact shape are the builder's.

### The 📚 control

The 📚 and what follows it become one control in `shell`, which every screen puts on its title line, as menus became
`shell.Dropdown`. The rest of each screen's title line, and its search, stays the screen's own, as the components
design decided.

- A single repository draws `📚 owner/repo`. A group draws `📚 <name> <count>`, the count dim. Both in the
  repository's magenta.
- A plain click on 📚 opens a dropdown, a `shell.Dropdown`:
  - when the workspace is a single repository that belongs to groups, a line per group first, "📚 empowerite 5",
    so the group is one click away;
  - the favorites;
  - **Browse…**, which opens a picker of every checkout under `roots` and every group, with groups suggested from
    owners (every local repository under one owner is offered as a group to save, never applied unasked);
  - **New workspace…**;
  - **Manage workspaces…**, which opens the Workspaces screen.
- Picking one tells every screen made, as settings are told now, and each loads again for the new workspace. For a
  group, the pick is recorded as the last group used.

kittui's rules for menus hold: every action takes a plain click, since iTerm swallows the others; a line that cannot
act now stays, dim; a line's keys stay out of the keys bar; the dropdown's glyphs are two-cell emoji.

### Flow over a workspace

**The 📚 repos tab** comes first, always. One row per member. Its fields, at the least:

| Field | What it shows |
| --- | --- |
| repository | host and path; the name alone when every member shares a host and owner |
| forge | the primary remote's forge |
| remotes | every remote, a pill each, as labels are; ☁️ and the forge's URL for a member with no checkout |
| checkout | the checkout's path, or ☁️ |
| branch | the desk's branch, with ahead and behind |
| desk, backlog, underway, worktrees, changes, builds, incoming, releases | each tab's count for this member |
| member by | fixed, a query, or a Project |

A count is a link: a click opens its tab filtered to that member. A count a ☁️ member cannot have (desk, worktrees,
incoming) is a dim dash, not zero. A member whose read failed shows ❌ in its row, with git's or the forge's line in
the popover, and every other member still draws.

**Every other tab** gathers its rows from every member. `repository` is a field of every stage, and an Auto-grouped
view groups by it, as it already groups a column that differs from row to row. A row's repository links back to that
member's row on the 📚 repos tab. The tab bar's counts are totals over the workspace, ☁️ members contributing only
to the tabs the forge answers.

**A GitHub Project** narrows backlog and underway to its items, for the members it brought in: their issues outside
the Project are not shown. A member that `repos` or `match` brought in shows all its issues, as in any group, even when
the Project also names it. The other tabs span every member as in any group.

**An issue appears once.** Several members can share a tracker outside the forge, such as an Azure DevOps project
their `kit.yml` files all declare. An issue is keyed by its tracker and its id, not by the member it was read
through, and is read once per tracker. Its `repository` field holds every member that declares the tracker, a value
each, so grouping by repository shows it under each member, counted once in every total, as a field of several values
already behaves.

**Acting from a row** runs in that row's member: Watch, opening a branch, starting work. An action that needs a
checkout, on a ☁️ member, asks first, then clones into the first root, under `host/owner/repo`, and goes on.

### Welcome and Tutorials over a workspace

Welcome composes the way Flow does. Over a group, each step's checks are every member's, from `kit welcome --checks
--json` run in each member's checkout, and each check carries its member's host and path, as a Flow row does. A check
that holds or fails alike in every member, such as a tool on the machine, is said once; one that differs is shown per
member. A step's state counts its failed checks across the members, and `--fix` runs in the failed check's member.
A ☁️ member has no checks, and Welcome names it as not cloned. Over a single repository, Welcome is as it is now.
Tutorials is kit's catalogue, not a repository's, and does not change with the workspace.

### Where Flow's data comes from

`kit flow` gains `--workspace <name>`. For each member with a checkout, the local stages read that checkout as they do
now, under its own `kit.yml`, so a GitLab member beside GitHub ones keeps its forges and trackers. The forge stages
(backlog, underway, changes, builds, releases) are read per forge and owner in batches where the forge allows it
(GitHub's GraphQL reads many repositories' issues and pull requests in one request), not one request per member per
stage. Batching by forge and owner covers the issues of members whose `#` is the forge's own; a member whose
`kit.yml` gives `#` to another tracker, and every declared tracker, is read through `tracker.sh` per tracker. A member
with no `kit.yml` uses what its remotes say. Each row gains the member's host and path.

### State and cost

- The last group used, and each group's resolved members with the time they were resolved, are kept under
  `XDG_CACHE_HOME` (else `~/.cache`) as `kit/workspaces/`. They are kit's to rewrite, unlike the definitions.
- A group opens at once from its last resolved members and its last kept rows, then resolves and reads again behind
  them, as the screens keep a copy shown while they load.
- The forge is read for a workspace at most once a minute, as for one repository today. Query resolution, which
  changes rarely, is read at most once an hour, and on demand.
- `kit flow --quota` reports each forge the workspace touches.

## The order of the work

Each step is usable on its own.

1. The workspace file, `kit workspaces` with fixed members, local checkout discovery, and the startup rule.
2. The 📚 control in `shell`, its dropdown with favorites and groups, and switching between workspaces; Welcome over
   a group.
3. The 📚 repos tab, `kit flow --workspace`, and every tab across the members with checkouts.
4. Queries, ☁️ members, batched forge reads, and the query cache.
5. Topics and git config labels as filters.
6. GitHub Project workspaces.
7. The Workspaces screen, and **New workspace…** and **Browse…** in full.

The 📚 repos tab is a stage like the others, with its fields in Flow's catalog, so step 3 does not wait on the
components design's `table` move. It would gain from `source` (step 5 of that design) for its per-member loads.

## My calls

These were not decided in the brainstorm. Each is a default the design takes until someone changes it.

- The file's name, place, format, and field names; state and caches kept apart from it, under `XDG_CACHE_HOME`.
- A member named by host and path; checkouts found under `roots`, four levels down.
- Outside any repository with no group ever used, there is no workspace, and `kit flow` behaves as today;
  `--workspace none` brings that back where a group would be drawn.
- Only `kit flow` and `kit workspaces` take `--workspace` for now; the verbs that act are unchanged.
- Only kit reads and writes `workspaces.yml`; kittui goes through `kit workspaces --json`.
- A checkout belongs only to the member its primary remote points at, so a fork's clone is counted once.
- An issue from a tracker several members share appears once, keyed by tracker and id, under each member's group.
- A Project narrows backlog and underway only for the members it brought in; its members are the repositories of its
  open items, any owner.
- Inside a member, kit starts on the member, and the 📚 dropdown offers its groups first.
- Welcome says a check that is the same in every member once, and per member one that differs.
- `kit welcome` gains no `--workspace` now: kittui runs it in each member's checkout and composes the answers.
- Owners suggested as groups in **Browse…**, never applied unasked.
- A ☁️ member is cloned only when an action needs a checkout, after asking, into the first root as `host/owner/repo`.
- A GitHub Project workspace narrows backlog and underway to the Project's items.
- Dim dashes, not zeros, for what a ☁️ member cannot have.
- The topic convention `workspace-<name>`.

## Left to the builder

- The subcommands of `kit workspaces` and its JSON's shape.
- The query filters beyond owner, visibility, name, topic, and label (fork, archived, language, and so on).
- How the Workspaces screen edits queries: in kittui, or by opening the file in `$EDITOR`.
- The batching per forge, and what GitLab and Azure DevOps members do where GitHub's GraphQL would batch.
- The 📚 repos tab's fields beyond those listed, and its default preset.
