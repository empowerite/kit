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

A group's name holds lowercase letters, digits, and `.`, `_`, or `-`, and never `/`. That is how kit tells a name from
a repository.

kit checks the file wherever it reads it. It refuses a file that does not parse or is not a map, a group with a key
other than `repos` and `exclude` (`match` and `project` are not built yet), a group whose name breaks that rule, an
entry that names no repository, and a group named `none`. `kit workspaces`, `kit flow --workspace`, and `kit workspace
--workspace` each say what is wrong and exit 1, printing nothing, as `kit flow` and `kit workspace` do outside a
repository with a group used last, where they read the file for that group. Inside a repository, or with no group
used, `kit flow` and `kit workspace` with no `--workspace` read no file.

## Members and checkouts

A member is named by its host and its whole path: `github.com/empowerite/edge`, or `gitlab.com/group/sub/repo` on
GitLab. A git URL names the same member, so `git@github.com:empowerite/jig.git` is `github.com/empowerite/jig`. kit
spells a member one way: the host in lower case, and on GitHub and GitLab the path too, so `github.com/NixOS/nixpkgs`
is `github.com/nixos/nixpkgs`.

A member's checkout is a clone under one of the roots whose primary remote points at it: the `primary` setting in its
`kit.yml`, else `origin`. A clone's other remotes never make it a checkout of another member, so a fork's clone, with
the original as `upstream`, counts once, under the fork. A member cloned twice has both as checkouts, and the first,
under the earliest root, is the one kit uses. A member with no checkout is still a member.

## Which workspace kit starts on

1. A workspace given with `--workspace` wins. `--workspace none` names no workspace.
2. Inside a repository, it is that repository, even when the repository belongs to a group. In a linked worktree, it
   is the repository the worktree belongs to.
3. Outside any repository, it is the last group used.
4. With no group used yet, there is none.

`kit workspaces --current` says which one, and why. With `--json` it also gives what kittui's workspace menu draws: a
group's members with their checkouts, the groups a repository belongs to, and the favorites.

`kit flow` and `kit workspace` start where `kit workspaces --current` says. Outside a repository, they draw the last
group used, as `--workspace` with its name would. With no group used, or with `--workspace none`, `kit flow` draws
each desk below, as it does above desks, and `kit workspace` refuses with 2, as it reads a repository.

## Commands

```sh
kit workspaces                                    # the groups and the favorites
kit workspaces --show=empowerite                  # each member, with its checkout or ☁️
kit workspaces --current                          # the workspace kit starts on here, and why
kit workspaces --use=empowerite                   # start on empowerite outside a repository
kit workspaces --new=empowerite . github.com/empowerite/kit
kit workspaces --add=empowerite github.com/empowerite/edge
kit workspaces --remove=empowerite github.com/empowerite/edge
kit workspaces --rename=empowerite core           # the group keeps its place, its comments, and its favorite
kit workspaces --delete=core                      # the group goes; its checkouts are as they were
kit workspaces --favorite=empowerite
kit workspaces --unfavorite=empowerite
```

`--new` and `--add` take a member, a git URL, or a checkout's directory, as `.` for the repository you are in. The
forms that read print JSON with `--json`; `kit help workspaces` gives the shapes. The forms that edit print one line,
which says what changed, or that nothing needed to, as `📚 empowerite holds them already`, and take no `--json`. Each
takes the file's lock first, so two commands that edit the file at once each land.

`kit workspaces` exits 1 when the file cannot be read or edited, as when another command holds its lock past the
wait of 30 seconds, and 2 for a usage error, an unknown workspace, or a new name that is refused or taken.

## kit flow over a workspace

```sh
kit flow --workspace empowerite                   # each cloned member's picture under its name
kit flow backlog --workspace empowerite --json    # every member's backlog, each row tagged with its member
```

`--workspace` runs `kit flow` in every member's checkout at once, each under that checkout's own `kit.yml`, so a
GitLab member beside GitHub ones keeps its forges and trackers. With `--json`, a stage prints `{rows, errors}`: each
row, or each desk, worktrees, or incoming object, gains `member`, its host and path; a member that could not be read
is `{member, error}` under `errors`, and the others still draw. kit flow exits 1 when a member cannot be read: with
`--json`, what was read is still printed. `--marks` names such a member under `errors` beside the marks, and
`--quota` leaves its rate limits out; both say its line on stderr. A member with no checkout draws no stage; `kit
workspace repos` lists it. Over a workspace, the stages are kit's eight, whatever the `kit.yml` where you run it says.
`--stages`, `--marks`, and `--quota` span the members too; a pull request's number, and `--me`, name one repository's,
and are refused.

## Flow over a workspace in kittui

On a terminal, `kit flow --workspace empowerite` opens kittui's Issues app over the group, as picking the group from
the 📚 workspace menu does, and every other app follows the same group. Every tab gathers its rows from every cloned
member, grouped by member as they load, and a member whose read failed is one red line among them.

## kit workspace

```sh
kit workspace                                     # the git of the repository here: every stage
kit workspace repos --workspace empowerite        # each member: its forge, branch, and checkout, or not cloned
kit workspace branches                            # its branches, local and on each remote
kit workspace commits --workspace empowerite      # the commits not on the default branch, in every member
kit workspace worktrees --json                    # {rows, errors}, each row a worktree
```

`kit workspace` draws what git holds across a workspace: its members, then a stage for each of git's entities:
remotes, branches, worktrees, commits, and tags. It reads git alone, from each repository's main checkout, and asks
no remote, so a remote's side is what the last fetch left. HEAD is no stage: a remote's is its default branch, ⚓ on
its row and on the branch's, and a worktree's is its branch, or its commit where it is detached.

- repos: each member, cloned or not, with its forge, remotes, clone or `not cloned`, its desk's branch, its linked
  worktrees by branch, and the local branches no checkout has; with no `--workspace`, the repository here alone.
- remotes: each remote, its forge, the repository its URL names, its default branch, and how many branches it has.
- branches: each branch on a remote, with the local branches that pull from it and what the first is ahead and
  behind; and each local branch whose upstream is gone, or that has none. A branch checked out names its worktree.
- worktrees: each worktree, the desk first: its branch, what it is ahead and behind its upstream and the default
  branch, and its paths changed and untracked.
- commits: the commits the local branches hold and the default branch does not, the newest first, each with its
  branches, whether a remote has it, and the issue its subject starts with.
- tags: each tag, the newest first, annotated or lightweight, with its commit and its tagger.

`--workspace` reads every cloned member's checkout at once, as `kit flow --workspace` does, and every row gains its
member. kit workspace exits 1 when a member, or a member's desk, cannot be read: with `--json`, what was read is
still printed, the member under `errors`, or the desk's line as its repos row's `error`. On a terminal, `kit
workspace` opens kittui's Repositories app, a tab per stage, the 📚 repos tab first and kit flow's incoming last.
On the repos tab, a clone reads `~` for the home folder, and a closed row shows `…` and its last two folders; open,
the row shows the whole path. A member's name on any tab leads to its row there.

## Still to come

Groups made by queries (an owner, public or private, a name pattern such as
`homebrew-*`, forge topics), members read from the forge without a checkout, and groups made from a GitHub Project are
designed in [the workspaces design](superpowers/specs/2026-11-08-workspaces-design.md), and are not built yet.
