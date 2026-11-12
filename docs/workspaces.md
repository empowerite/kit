# Workspaces

A workspace is what kit looks at: one repository, or a named group of repositories that move together. Started inside
a repository, kit looks at that repository. A group lets you see several at once, without opening each one.

## The file

Groups are yours alone. kit keeps them in `kit/workspaces.yml` under `XDG_CONFIG_HOME`, else `~/.config`. You can edit
the file by hand; when kit edits it, your comments stay where they were.

```yaml
# where your clones are; kit looks for checkouts here, four levels down
roots: [~/Code]
favorites: [empowerite]
workspaces:
  empowerite:
    repos:
      - github.com/empowerite/kit
      - github.com/empowerite/edge
      - git@github.com:empowerite/jig.git
    exclude: [github.com/empowerite/attic]
```

- `roots` are the directories kit looks under for checkouts. A leading `~` is your home directory, and a root
  written twice, as `~/Code` and `~/Code/`, counts once.
- `favorites` names workspaces to keep at hand: a group's name, or a repository.
- Each group has `repos`, its members, and may have `exclude`, which takes members back out.

A group's name starts with a lowercase letter or a digit, and holds lowercase letters, digits, and `.`, `_`, or `-`,
and never `/`. That is how kit tells a name from a repository.

kit checks the file wherever it reads it. It refuses a file that does not parse or is not a map, a group with a key
other than `repos` and `exclude` (`match` and `project` are not built yet), a group whose name breaks that rule, an
entry that names no repository, and a group named `none`. `kit workspaces`, `kit --workspace=<name> flow`, and `kit
--workspace=<name> workspace` each say what is wrong and exit 1, printing nothing, as `kit flow` and `kit workspace`
do outside a repository with a group used last, where they read the file for that group. Inside a repository, or
with no group used, `kit flow` and `kit workspace` with no `--workspace=` read no file.

## Members and checkouts

A member is named by its host and its whole path: `github.com/empowerite/edge`, or `gitlab.com/group/sub/repo` on
GitLab. A git URL names the same member, so `git@github.com:empowerite/jig.git` is `github.com/empowerite/jig`. kit
spells a member one way: the host in lower case, and on GitHub and GitLab the path too, so `github.com/NixOS/nixpkgs`
is `github.com/nixos/nixpkgs`.

A member's checkout is a clone under one of the roots whose primary remote points at it: the `primary` setting in its
`kit.yml`, else `origin`. A clone's other remotes never make it a checkout of another member, so a fork's clone, with
the original as `upstream`, counts once, under the fork. A member cloned twice has both as checkouts, and the first,
under the earliest root, is the one kit uses. A member with no checkout is still a member. A clone whose `kit.yml`
does not parse names no primary, so it is filed under `origin` and the walk of the roots goes on; kit says so where
its member is drawn: a warning for the member's first checkout, and its repos row's `error`.

A checkout is a directory that holds a `.git` directory. A bare repository, a linked worktree, and a clone behind a
symbolic link are no checkouts, and kit does not look inside a hidden folder.

## Which workspace kit starts on

1. A workspace given with `--workspace=`, the kit option before the verb, wins: `kit --workspace=empowerite flow`.
   `--workspace=none` names no workspace.
2. Inside a repository, it is that repository, even when the repository belongs to a group. In a linked worktree, it
   is the repository the worktree belongs to.
3. Outside any repository, it is the last group used.
4. With no group used yet, there is none.

A directory is inside a repository when it is in the repository's working tree, where `git rev-parse --show-toplevel`
answers. A bare repository has no working tree, and a `.git` directory is not in one, so both are outside a
repository: there, too, kit starts on the last group used, and with none, `kit flow` draws the desks below and `kit
workspace` refuses.

`kit workspaces --use=<name>` keeps the last group used in `kit/workspaces/last` under `XDG_CACHE_HOME`, else
`~/.cache`: kit's own to rewrite, apart from your file. A last group the file does not hold is none, and
`kit workspaces --delete` of the last group used forgets it, so a group made later under its name is not the last
used.

`kit workspaces --use=<folder>`, with a folder written as a path, as `.`, `~/Code/edge`, or `/srv/edge`, keeps the
repository whose working tree holds it first among the recent repositories, in `kit/workspaces/recent` under
`XDG_CACHE_HOME`: ten at most, each once, the newest first. A linked worktree is no workspace of its own: its folder
is kept as its repository's desk, and one kept before stands as that desk. A recent repository is no group: kit starts
on it only inside it, as on any other. The workspace menu offers them, and its **Open…** keeps the one it opens there.
`kit --workspace=<path> workspaces --current`, with an absolute path, names the repository whose working tree holds
it, and refuses with 2 where none does.

`kit flow`, `kit workspace`, and `kit workspaces --current` follow this rule. Every other verb runs in its folder's
repository, or refuses outside one, and takes no `--workspace=`. `--workspace=` is an option of `kit` itself, which the
loader, `kit/bin/kit`, reads and hands to the verb, and a verb refuses it among its own words. A call across a
workspace runs on the kit you installed, and each member's own call, in its checkout, runs the release that member
pins, so `--workspace=` takes no `--version=`.

`kit workspaces --current` says which one, and why. With `--json` it also gives what kittui's workspace menu draws: a
group's members with their checkouts, the groups a repository belongs to, every group in the file, the favorites, and
the recent repositories whose working trees are still there.

`kit flow` and `kit workspace` start where `kit workspaces --current` says. Outside a repository, they draw the last
group used, as `--workspace=` with its name would. With no group used, or with `--workspace=none`, `kit flow` draws
each desk below, as it does above desks, and `kit workspace` refuses with 2, as it reads a repository.

## Commands

```sh
kit workspaces                                    # the groups and the favorites
kit workspaces --show=empowerite                  # each member, with its checkout or ☁️
kit workspaces --current                          # the workspace kit starts on here, and why
kit workspaces --use=empowerite                   # start on empowerite outside a repository
kit workspaces --use=.                            # keep the repository here among the recent ones
kit workspaces --new=empowerite . github.com/empowerite/kit
kit workspaces --add=empowerite github.com/empowerite/edge
kit workspaces --remove=empowerite github.com/empowerite/edge
kit workspaces --rename=empowerite core           # the group keeps its place, its comments, and its favorite
kit workspaces --delete=core                      # the group goes; its checkouts are as they were
kit workspaces --favorite=empowerite
kit workspaces --unfavorite=empowerite
```

`--new`, `--add`, and `--remove` take a member, a git URL, or a checkout's directory, as `.` for the repository you
are in. Each also takes a pattern, a member holding `*`, `?`, or `[`, as `github.com/empowerite/homebrew-*`, and
`--new` and `--add` take `--from=<command>`, a command kit runs by bash whose every line is a repository:

```sh
kit workspaces --new=brew 'github.com/empowerite/homebrew-*'      # the checkouts under the roots it matches
kit workspaces --add=org --from="gh repo list empowerite --limit 1000 --json url --jq '.[].url'"
kit workspaces --new=tools --from="gh repo list empowerite --json url --jq '.[].url'" 'github.com/empowerite/*-tool'
kit workspaces --remove=org 'github.com/empowerite/attic-*'       # the group's members it matches
```

A pattern matches the repositories `--from`'s command printed, else, for `--remove`, the group's members, else the
members of the checkouts under the roots, and one that matches none is refused with 2. With no pattern, every
repository the command printed joins. Either way, kit writes the members it found under `repos`, as if you had named
each: a group does not follow a pattern or a command afterward, which the queries of `match` will, once they are
built. The forms that read print JSON with `--json`; `kit help workspaces` gives the shapes. The forms that edit
print one line, which says what changed, or that nothing needed to, as `📚 empowerite holds them already`, and take
no `--json`. Each, and `--use`, takes the file's lock first, so two commands that edit the file at once each land.

`kit workspaces` exits 1 when the file cannot be read or edited, as when another command holds its lock past the
wait of 30 seconds by the clock, or `kit/workspaces/` under `XDG_CACHE_HOME` cannot be written, and 2 for a usage
error, an unknown workspace, or a new name that is refused or taken.

## kit flow over a workspace

```sh
kit --workspace=empowerite flow                   # each cloned member's picture under its name
kit --workspace=empowerite flow backlog --json    # every member's backlog, each row tagged with its member
```

`--workspace=` runs `kit flow` in every member's checkout at once, each under that checkout's own `kit.yml`, so a
GitLab member beside GitHub ones keeps its forges and trackers. With `--json`, a stage prints `{rows, errors}`: each
row, or each desk, worktrees, or incoming object, gains `member`, its host and path; a member that could not be read
is `{member, error}` under `errors`, and the others still draw. Every row already says where it is from, as each
member's `kit flow` printed it: `organization` and `repository`, as the member's primary remote names them, and
`source`, the forge, or `git` for incoming's; see [apps.md](apps.md). kit flow exits 1 when a member cannot be read: with
`--json`, what was read is still printed. `--marks` names such a member under `errors` beside the marks, and
`--quota` leaves its rate limits out; both say its line on stderr. A member with no checkout draws no stage; `kit
workspace repos` lists it. Over a workspace, the stages are kit's eight, whatever the `kit.yml` where you run it says.
`--stages`, `--marks`, and `--quota` span the members too; a pull request's number, and `--me`, name one repository's,
and are refused.

## The workspace menu in kittui

The `-` on every screen's title line drops down the workspaces to switch to: the groups the repository here belongs to;
the favorites, however many; every other group in the file; and the recent repositories, but the one shown, or the one a
worktree shown belongs to. The one shown is dim, as is a favorite repository with no checkout. **Open…**, last, opens a
dialog to type a repository's folder in: tab completes a folder, the folders it can complete to stand under it, and a
plain click on one opens it. Enter, or a click on Open, asks kit; a folder in no git working tree is refused with one
line in the dialog, which stays open.

**Manage…**, after it, opens the workspaces' screen: every group a row, with whether it is a favorite, how many
members it has, and the members. The app's menu, `=`, holds what it does to the group under the cursor, each a line
that a plain click does and a key does with the menu closed, and each a `kit workspaces` command kittui runs: New
Workspace… (`n`), a name and its repositories; Switch to It (`⏎`); Rename… (`r`); Add Members… (`a`), repositories and
patterns; Add an Owner's Repositories… (`o`), a GitHub owner and a pattern, which runs `gh repo list` through
`--from`; Add from a Command… (`c`); Remove Members… (`x`); Favorite or Unfavorite (`f`); and Delete… (`d`), which
asks first. kit's line, when it refuses, stands in the action's dialog; when it is done, every screen's menu lists
the groups as they are now, and a group renamed or deleted while it is shown is followed. Its title line has a crumb,
↩ and the screen Manage… was chosen on, which a plain click or ⌫ goes back to, as a link's crumb does. A switch to
another workspace drops every crumb, as the apps are made anew there.

## Flow over a workspace in kittui

On a terminal, `kit --workspace=empowerite flow` opens kittui's Issues app over the group, as picking the group from
the 📚 workspace menu does, and every other app follows the same group. Every tab gathers its rows from every cloned
member, grouped by member as they load, and a member whose read failed is one red line among them.

## kit workspace

```sh
kit workspace                                     # the git of the repository here: every stage
kit --workspace=empowerite workspace repos        # each member: its forge, branch, and checkout, or not cloned
kit workspace branches                            # its branches, local and on each remote
kit --workspace=empowerite workspace commits      # the commits not on the default branch, in every member
kit workspace worktrees --json                    # {rows, errors}, each row a worktree
```

`kit workspace` draws what git holds across a workspace: its members, then a stage for each of git's entities:
remotes, branches, worktrees, commits, and tags. It reads git alone, from each repository's main checkout, and asks
no remote, so a remote's side is what the last fetch left. HEAD is no stage: a remote's is its default branch, ⚓ on
its row and on the branch's, and a worktree's is its branch, or its commit where it is detached.

- repos: each member, cloned or not, with its forge, remotes, clone or `not cloned`, its desk's branch, its linked
  worktrees by branch, and the local branches no checkout has; with no `--workspace=`, the repository here alone.
- remotes: each remote, its forge, the repository its URL names, its default branch, and how many branches it has.
- branches: each branch on a remote, with the local branches that pull from it and what the first is ahead and
  behind; and each local branch whose upstream is gone, or that has none. A branch checked out names its worktree.
- worktrees: each worktree, the desk first: its branch, what it is ahead and behind its upstream and the default
  branch, and its paths changed and untracked.
- commits: the commits the local branches hold and the default branch does not, the newest first, each with its
  branches, whether a remote has it, and the issue its subject starts with.
- tags: each tag, the newest first, annotated or lightweight, with its commit and its tagger.

`--workspace=` reads every cloned member's checkout at once, as it does for `kit flow`, and every row gains its
member. kit workspace exits 1 when a member, or a member's desk, cannot be read: what was read is still drawn, as
text or with `--json`, the member under `errors`, or the desk's line as its repos row's `error`. A bare repository
is outside a repository: without `--workspace=`, kit workspace draws the last group used there, and with none
refuses it with 2. On a terminal, `kit workspace` opens kittui's Repositories app, a tab per stage, the 📚 repos tab
first and kit flow's incoming last.
On the repos tab, a clone reads `~` for the home folder, and a closed row shows `…` and its last two folders; open,
the row shows the whole path. A member's name on any tab leads to its row there.

## kit welcome

```sh
kit welcome --checks --json                       # the machine's checks, then each folder of the repository here
kit --workspace=empowerite welcome --checks       # each member's desk and worktrees, in the group's order
kit --workspace=none welcome --checks             # no group: the repository here, else the machine's checks alone
```

`kit welcome`'s setup checks each folder of the workspace on its own, since each branch can have a `kit.yml` or none,
and pin a kit of its own: for a repository, its desk, then each worktree `git worktree list` names; for a group, each
member's, for every checkout of it under the roots. Each folder's check names its folder, its repository, whether it
is the desk, its branch, and the kit its `kit.yml` declares, and `kit check` runs there on the release that folder
pins. The machine's checks, the tools kit runs on, the system toolchain, and each forge host's tool and login, run
once, for every host among the folders' remotes. The folders are judged at once. `kit welcome --fix <check>
--folder=<path>` runs a folder's check's fix in that folder.

On a terminal, kittui's setup step draws the checks as one table: the machine's first, then each repository, a row
per folder with its branch, its kit, and a column for each kind of check, its failed checks under it, each offering
its fix. The introduction names each folder above its failed checks.

## Still to come

What is built is steps 1 to 3 of [the workspaces design](superpowers/specs/2026-11-08-workspaces-design.md): the file,
`kit workspaces` with fixed members, checkout discovery, the startup rule, the 📚 control, and Flow across the members
with checkouts. Steps 4 to 7 are designed and not built:

- Queries, ☁️ members, batched forge reads, and the query cache. A group's `match` is refused by name until then.
- Topics and git config labels as filters.
- GitHub Project workspaces. A group's `project` is refused by name until then.
- **Browse…**, a picker of every checkout under the roots with groups suggested from owners. The workspaces' screen,
  **Open…**, and **Manage…** are built; a pattern or a command given to `--new` and `--add` is read once, when it is
  given.
