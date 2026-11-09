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

- `roots` are the directories kit looks under for checkouts. A leading `~` is your home directory.
- `favorites` names workspaces to keep at hand: a group's name, or a repository.
- Each group has `repos`, its members, and may have `exclude`, which takes members back out.

A group's name holds lowercase letters, digits, and `.`, `_`, or `-`, and never `/`. That is how kit tells a name from
a repository.

## Members and checkouts

A member is named by its host and its whole path: `github.com/empowerite/edge`, or `gitlab.com/group/sub/repo` on
GitLab. A git URL names the same member, so `git@github.com:empowerite/jig.git` is `github.com/empowerite/jig`.

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

## Commands

```sh
kit workspaces                                    # the groups and the favorites
kit workspaces --show empowerite                  # each member, with its checkout or ☁️
kit workspaces --current                          # the workspace kit starts on here, and why
kit workspaces --use empowerite                   # start on empowerite outside a repository
kit workspaces --new empowerite . github.com/empowerite/kit
kit workspaces --add empowerite github.com/empowerite/edge
kit workspaces --remove empowerite github.com/empowerite/edge
kit workspaces --favorite empowerite
kit workspaces --unfavorite empowerite
```

`--new` and `--add` take a member, a git URL, or a checkout's directory, as `.` for the repository you are in. Each
form prints JSON with `--json`; `kit help workspaces` gives the shapes.

## kit flow over a workspace

```sh
kit flow --workspace empowerite                   # repos, then each cloned member's picture under it
kit flow backlog --workspace empowerite --json    # every member's backlog, each row tagged with its member
kit flow repos --workspace empowerite             # each member: its forge, branch, and checkout, or not cloned
kit flow repos                                    # the repository you are in, alone
```

`--workspace` runs `kit flow` in every member's checkout at once, each under that checkout's own `kit.yml`, so a
GitLab member beside GitHub ones keeps its forges and trackers. With `--json`, a stage prints `{rows, errors}`: each
row, or each desk, worktrees, or incoming object, gains `member`, its host and path; a member that could not be read
is `{member, error}` under `errors`, and the others still draw. A member with no checkout draws no stage but `repos`.
Over a workspace, the stages are `repos` and kit's eight, whatever the `kit.yml` where you run it says.
`--stages`, `--marks`, and `--quota` span the members too; a pull request's number, and `--me`, name one repository's,
and are refused.

## Flow over a workspace in kittui

On a terminal, `kit flow --workspace empowerite` opens kittui's picture over the group, as picking the group from the
📚 workspace menu does. Every tab gathers its rows from every cloned member, grouped by member as they load, and a
member whose read failed is one red line among them. The 📚 repos tab comes first, always, a single repository's
picture included: a row per member, with its forge, remotes, checkout or `not cloned`, its desk's branch, and a count
of each tab's rows of the member. A click on a count opens that tab filtered to the member. A member that is not
cloned counts a dim `-`, never 0.

## Still to come

Groups made by queries (an owner, public or private, a name pattern such as
`homebrew-*`, forge topics), members read from the forge without a checkout, and groups made from a GitHub Project are
designed in [the workspaces design](superpowers/specs/2026-11-08-workspaces-design.md), and are not built yet.
