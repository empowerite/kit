# Leases: design

Status: designed, 2026-11-12. see: issue #73

## Intent

Several sessions, people or agents, change one repository at once. On 2026-11-02 and 11-03, six Claude sessions
worked in one worktree of this repository at once. Each held files the others waited on: a sweep of every banner line,
a move of `tui/`, the splits of the docs. One session swept another's uncommitted README edits into its own commit.
The sessions sorted it out by asking each other by message, since nothing in the repository said who held what.

A lease says it. A session claims the paths it will change, for an issue, in a git ref every worktree of the clone
sees. A claim that overlaps another issue's live lease is refused, naming who holds what, so the session asks before
it takes a path someone else holds. `kit commit` refuses a staged path that another issue's lease on the same branch
holds, which is where this repository's sessions collided. CI holds the branch to its own lease, so a branch that
strays outside what it claimed fails.

A lease is not a lock, not an identity, and not proof that its holder is alive. It is a note every session reads, and
it lapses when nobody renews it.

## Decided

From the issue, and from Aegis (`aegis_core/team.py`, `lane_gate.py`), which does this for its lanes:

- A lease is a JSON blob at `refs/kit/leases/<key>`, written with `git hash-object -w` and moved by a compare and
  swap through `git update-ref --stdin`, so every worktree of the clone sees it at once.
- A claim that may overlap a live lease of another holder is refused. A lease that does not read counts as
  overlapping everything.
- A lease expires, one hour after it is taken or renewed by default.
- The backstop is CI: every path a branch changed since its merge base with its base must be inside its claim.

The issue's two questions:

1. **Where the paths come from.** From the words of the claim: `kit leases take <reference> <path>…`. `kit work`
   takes the paths its issue names under a `Paths` heading, and `**` when the issue names none or its tracker keeps
   no body. Paths the branch has changed so far are not a claim: a claim says what a session will change.
2. **What a lease binds.** Both. Leases of different branches compete when they are taken. Inside one branch,
   `kit commit` refuses a path another issue's lease holds. A branch's own lease, the one of the issue its name
   starts with, is the edge of the branch: the leases that sessions take inside it nest in it and never compete with
   it.

## Terms

- **Key**: an issue's reference as a branch name starts with it, `tracker:key`: `73` for `#73`, `ab4521` for
  `AB#4521`. The lease of issue `#73` is `refs/kit/leases/73`.
- **Holder**: the issue a lease is for. Two sessions working one issue share its lease. The lease also records who
  took it, a name of the session's choice, which `kit leases` prints.
- **Live**: a lease whose `expires` is later than now, or one that does not read.
- **A branch's own lease**: the lease whose key the branch's name starts with, `<key>-`, as `kit work` names a
  branch. CI holds the branch to it.
- **Compete**: two leases compete when they are of different issues, unless they are on the same branch and one is
  that branch's own lease. Only competing leases are judged for overlap.

## The verb

A noun lists, and the noun with a verb acts, as `kit packages` does:

```text
kit leases [--json | --jq]
kit leases take <reference> [--for=<duration>] [--holder=<name>] [<path>…]
kit leases renew <reference>… [--for=<duration>]
kit leases release <reference>…
kit leases --check
```

- `kit leases` lists every lease in the clone, the live first, each with its issue, its holder, its branch, when it
  expires, and its paths. A lease that does not read is named with the command that releases it.
- `take` claims the paths for the issue, `**` without any. A live lease of the same issue keeps its paths, and the
  new ones join them; an expired one is replaced. The claim is refused, and nothing is written, when it overlaps a
  live lease it competes with: the refusal names that lease's issue, holder, branch, paths, and expiry.
- `renew` moves a lease's expiry to now plus its term, or `--for`. A lease that expired is judged again first, as a
  new claim is, since another may have claimed its paths meanwhile.
- `release` deletes the lease, here and on the primary remote when it holds a copy.
- `--check` is the CI rule below, as `kit policy` runs it.

`<duration>` is a number and a unit: `90s`, `45m`, `2h`, or `1d`. The default is `1h`. `--holder` defaults to git's
`user.name`, else the login.

## The lease

```json
{
  "lease": 1,
  "issue": "#73",
  "paths": ["kit/bin/kit-leases", "kit/modules/bash/leases.sh", "docs/**"],
  "holder": "kit-40/i73",
  "host": "desk",
  "branch": "1-the-first-release-v0-1-0",
  "worktree": "/Users/ada/kit/worktrees/1-the-first-release-v0-1-0",
  "taken": 1762950000,
  "renewed": 1762950000,
  "expires": 1762953600
}
```

`lease` is the shape's version. Times are seconds since the epoch. The term is `expires - renewed`. `branch` is null
on a detached HEAD, where a lease competes with every other. A blob that is not a JSON object with `issue`, `paths`
as an array of strings, and `expires` as a number does not read.

## Paths

A path is relative to the repository's root, as `git ls-files` prints it. It holds itself and everything under it, so
`docs` holds `docs/policy.md`. `*` and `?` match within one segment, `[…]` one character of a set, and `**` any number
of segments. A path that starts with `/` or holds a `..` segment is refused.

Two claims may overlap unless kit can prove they cannot: segment by segment, a pair of literal segments that differ,
or a literal segment the other's pattern does not match, proves them apart. Two patterns in one segment are apart only
when their literal starts or ends differ. `**`, or one claim running out of segments first, overlaps. So the judgement
errs toward refusing.

An issue names its paths under a heading whose text is `Paths`, one per list item, until the next heading. A path in
backticks is taken as written; otherwise the item's first word is.

## The compare and swap

Every change to a lease also moves `refs/kit/leases-epoch`, a blob that names the change and the epoch before it. A take
reads the epoch, then every lease, judges the claim, writes its blob, and then, in one `git update-ref --stdin`
transaction, moves its lease from the value it read and the epoch from the value it read. A session that changed any
lease meanwhile moved the epoch, so the transaction fails and the take reads and judges again. The epoch is read before
the leases: read after them, it could already hold a lease the take never judged, and the swap would land. Two sessions
that claim overlapping paths at once: one transaction lands, and the other, judged again, is refused, naming the first.
A take tries 50 times before it gives up.

## Inside one worktree

`kit commit` reads the issues the commit names, the reference in front of its subject and each `Refs:` trailer's, then
refuses a staged path held by a live lease on the same branch of an issue it does not name, unless it is the branch's
own lease, which holds every path of the branch. The refusal names the lease's issue, holder, and expiry, and the
remedy: name that issue in the commit, or ask its holder to release it. A lease
that does not read has no branch, so a commit is warned of it rather than refused. After the commit, the lease of each
issue the commit names, when it is live and on this branch, is renewed by its term, so a lease lives while its work
goes on.

## CI

`kit policy`, which `kit ci` runs, judges a branch past the default one by its own lease. The changed paths are
`git diff --name-only --no-renames` from the merge base of the base, as `kit policy` names it, and HEAD: a move counts
as both its paths. Each one outside the lease is refused, with the remedy `kit leases take <reference> <path>`, then
`kit submit`. A branch whose name starts with no issue's key, or whose issue holds no lease, is judged by none, and the
output says so. A lease that does not read fails the check.

The lease is read from the clone. On a CI runner, whose clone has none, it is read from the primary remote:
`git ls-remote`, then `git fetch` of that one object, into no ref and no `FETCH_HEAD`. `kit submit` and `kit work` push
the branch's own lease to the primary, forced, since a blob has no history to fast-forward; `kit leases release` deletes
the primary's copy. A CI runner's clone has no lease of its own, so it reads the one the branch's last push carried. The
claim lives in a ref, never in the branch's tree, so no commit on the branch can widen the claim that judges it.

## Not now

- A stage of `kit flow`, or a tab in kittui, that shows who holds what. `kit leases` prints it, and its JSON is the
  document a stage would draw.
- `kit serve` offers its verbs by a list, `serve_tools`, not by default; `leases` joins it when the agents that call
  kit through the Model Context Protocol need to claim.
- Leases shared between clones. A clone's leases are its own; the primary's copy of a branch's lease is for CI.
