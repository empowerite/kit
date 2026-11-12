# Code review, 2026-11-10

This page is a review of the whole tree at `2dad3552`, the head of `1-the-first-release-v0-1-0`: the bash CLI under
`kit/`, kittui under `projects/tui/`, the tutorials site, the packages, the tests, the release scripts, and the
documents. Ten reviewers read one area each, reproduced what they could in throwaway repositories and scratch copies
of the tree, and changed nothing. [The findings](2026-11-10-code-review-findings.md) holds all 260 of them, numbered
by area, so "Workflow 1" is the first item under its Workflow heading. This page holds the findings that matter most,
the state of the specs, the plans, and the issues, and the plans written from the review.

## What ran

- `gofmt`, `go vet`, and `go test` pass over `projects/tui/`, and `go test -race` reports no race in `cmd/kittui`,
  `flow`, `screens`, `app`, `kit`, `source`, `welcome`, and `tutorials`.
- `kit syntax --check`, `kit check`, and `kit generate --check` pass. Every verb's usage line parses, and the 34
  syntax snapshots under `projects/tui/internal/syntax/testdata/` equal what `kit syntax` prints.
- The bats suite, 736 tests, run whole in a scratch clone, fails one test, `tests/underway.bats:199`
  (Infrastructure 2), so `kit gate` fails. No run of CI has passed on the pushed head, `6edaa6b6`; this branch is
  165 commits ahead of it, and the five tests that fail there pass at this commit.
- Everything ran on macOS, with bash 5.3, git 2.56, and jq 1.8.2. Nothing ran on Linux.
- A finding is confirmed when a reviewer reproduced it or the code leaves no doubt, and plausible otherwise. The
  reproductions were scratch scripts and are not kept: each finding's scenario says how to repeat it.

## A verb destroys work

- `kit rebase` force-pushes over commits that origin has and the branch lacks. It fetches, then pushes with a bare
  `--force-with-lease`, and the fetch has just moved the ref the lease compares with (Workflow 1).
- `kit tidy` deletes a branch whose commits were never pushed, once its change is merged or closed (Workflow 2).
- `kit commit` replaces the whole body of the pull request with `Closes #N` when the read of the body fails
  (Workflow 4).
- `kit work` leaves the desk's work in the stash when it fails after the sweep, and takes a stash that is not its
  own when its push saved nothing (Workflow 6, Workflow 7).
- `kit uninstall` and `kit packages remove` delete every line after a region's directive when the file has lost
  its `kit:end` line (Delivery 1). `kit packages remove` deletes a delivered copy the repository edited
  (Delivery 13).
- `kit update` drops an active package from `kit.yml`, removes its files, and exits 0, when the package's
  dependencies form a cycle (Delivery 5).
- `kit packages add` leaves a `kit.yml` that yq refuses, when the file's lists are not indented (Delivery 4).

## A guarantee does not hold

- `kit serve` is described as read-only in `1st.md`, the README, the Claude adapter's page, and `kit help serve`.
  Its `packages` tool runs `kit packages add`, its `check` tool runs every `kit:generate` command in the tree, and
  six of its seventeen tools write to the forge (Serve 1, Serve 2, Serve 7, Documents 1). Three kinds of message
  stop the server without an answer (Serve 3, Serve 4).
- The refusal to commit on the default branch fails open when the default branch cannot be read (Workflow 5). The
  guard against undoing a commit the index did not see misses a path that two commits changed (Workflow 3).
  `require_signing: true` warns and commits unsigned (Workflow 13).
- One invalid regular expression in a kind stops the classifier, every path has no kind, and `kit lint` and
  `kit check` pass (Delivery 7). `kit lint` on a path that does not exist says every governed path passes
  (Documents 7). A `kit.yml` that yq cannot read is taken for a repository with no kit.
- `kit lint` is not one of the four checks of `kit ci`, so a consumer whose gate does not call it has no CI check of
  pinned actions or of a file's shape, and the documents imply otherwise (Documents 2).
- A failed check is hidden by a check of the same name that started later, so a change with a failing job reads as
  ready (Flow 1). The GitLab provider drops a merge request whose read failed and keeps the short list as the
  snapshot (Flow 2). Both providers read first pages only and say nothing (Flow 4).
- The Claude deny list does not match `git -C x commit`, `bash -c 'git push'`, `git checkout -B`, and other forms
  of the commands that `1st.md` forbids (Infrastructure 11).

## The release cannot ship

- `bin/release pack` writes a `.sha256` with no digest and exits 0: it calls `release:digest`, and the function is
  `file:digest`. Every consumer's checksum step and `kit update` refuse such a release (Infrastructure 1).
- `pack` copies the working tree, not the tag. `publish` does not check that the tag exists. Merging this branch
  retires the pre-releases that consumers pin. A release has no signature and no attestation, and no ruleset
  guards `release/*` tags (Infrastructure 7, Infrastructure 8, Infrastructure 9).
- The README's `brew install empowerite/tap/kit` names a tap that is empty, and every link `kit tutorials` prints
  answers 404, because nothing builds or publishes the site (Infrastructure 3, Infrastructure 4).
- On jq 1.7, the minimum the kit accepts and the one macOS ships, `kit check` fails a file that passes on jq 1.8
  (Delivery 6).
- `kit tools run prettier --check .` is refused, because `--check` is one of `kit tools`' own flags. The first
  `kit tools install` on a fresh machine skips npm and uv and exits 0. The uv check fails on a lock over 1 MB and
  expects packages that a platform marker excludes (Command line 1 to 4).
- No repository but this one takes the kit, and `kit update` has never run across two real releases (Gaps).

## kittui shows the wrong thing, or stops

- A reload decodes into the rows the tab already holds, so a JSON `null` keeps the value of the row that stood at
  that index: after issue 5 closes, issue 7 is drawn with issue 5's branch (Apps 1).
- One remote whose fetch fails blanks the whole incoming tab, because kit prints its document and exits 1, and the
  bridge discards the output of a failed call (Apps 2, Flow 6).
- A verb's form runs in the repository kittui started in, not the one picked from the workspace menu (Apps 3).
  `--no-fetch`, `--mine`, and `--owner` reach the first app only (Apps 6).
- A form keeps a choice by its row's position, so it lands on another row: the form panics, or builds
  `kit tools --jq`, which nobody chose (Components 1). An open column menu panics when the table empties
  (Components 2). `kit welcome` panics on a page key when a search hides every check, and in that state ⏎ runs the
  fix of a check that is not shown (Apps 4, Apps 5).
- The tab menu costs 0.9 seconds a frame at 500 rows and 13.6 seconds at 2,000 (Components 3). Paste does nothing
  in a field or the search (Components 4).
- Twenty of the thirty-two tabs of the seven apps have no data source, and the `desk` and `worktrees` stages, Add to
  Pull Request…, and most of the flow tests belong to the one picture that no app opens (Apps 7, Apps 14, Gaps).

## The repository against its own rules

- `.kit/workflow.md` says a plan is prose in the body of an issue and is never committed as a file. This repository
  takes the `workflow` package and commits its specs and plans under `docs/superpowers/`, this review's plans
  among them (Infrastructure 27).
- The gate's floor for the TUI's tests is 98% of statements. The tests cover 8,512 of 8,689, which is 97.96%;
  `go tool cover` prints 98.0, and `bin/ci` compares the printed figure.
- Nine `see:` lines under `kit/` write their path from the repository root, `kit/bin/kit-workspaces` names
  `github.com/empowerite/edge`, and two delivered files point at `docs/`, which a consumer does not have
  (Infrastructure 25).
- kittui draws circled glyphs (the radios, the filter mark, and the tab numbers), a form and the palette take a
  click on the menu bar only, and a column moves by ⇧ and an arrow only (Components 10, 11, and 19, Apps 18).
- `AGENTS.md` and the README say `libexec/kittui` is built from `tui/`; the source is `projects/tui/`, and the files
  are `libexec/kittui-<os>-<arch>`. `projects/tui/README.md` describes six screens, and kittui opens seven apps
  (Documents 12, Documents 14).

## The specs, the plans, and the issues

- The five plans under `docs/superpowers/plans/` dated before this review are built, except task 4 of plan 3a, which
  is partial. Their checkboxes were never ticked.
- The workspaces spec is built through step 3 of its order of work. Steps 4 to 7 have no issue.
- The tutorials spec is built inside the repository. Publishing the site, the changelog gate, and a first real
  tutorial are not.
- The entities spec is a design. None of the four decisions under its "What the first release settles" is met.
- Twelve issues have no code: #62, #63, #64, #66, #67, #68, #69, #72, #73, #76, and #77, and #74 in part. They are
  the issues that the closing list of pull request #2 leaves out.
- Issue #71 is done. The three cases of #70 are done, and six verb headers still omit a status the verb exits
  with: `check`, `gate`, `hydrate`, and `update` omit 2, and `help` and `serve` omit 1. Issue #3's closing
  comment says the Cursor, Copilot, and Gemini adapters are still open, and `kit/packages/agents/` has all three.

## The plans written from this review

Each plan is in the form of the plans before it: test first, a task at a time. Each lists the decisions it leaves
to the owner, with the default it is written on.

| Plan | Issue | Holds |
| --- | --- | --- |
| [Refusals that hold](../plans/2026-11-10-refusals-that-hold.md) | #141 | the verbs that destroy or write: rebase, tidy, work, commit, serve, regions |
| [Fail closed](../plans/2026-11-10-fail-closed.md) | #69, #67 | a read or a check that fails is never empty, absent, or passing |
| [Workspaces 4](../plans/2026-11-10-workspaces-4-says-what-it-does.md) | #89 | what is built of workspaces says what it does |
| [Contracts](../plans/2026-11-10-contracts.md) | #74 | snapshots of every verb's syntax and the MCP tools, and the documents' claims checked |
| [The apps' tabs read kit](../plans/2026-11-10-apps-tabs-read-kit.md) | #138 | a tab is a catalog entry naming a kit command; what the one Flow picture left goes |
| [What the first release settles](../plans/2026-11-10-entities-first-release-decisions.md) | #135 | references, owner and reach, provider contracts, a signable verdict |
| [Release readiness](../plans/2026-11-10-release-readiness.md) | #1 | a tarball that verifies, a green gate, an update across releases, and what needs the owner |
| [Tutorials 2](../plans/2026-11-10-tutorials-2-published.md) | #86 | the site published, its links resolving, and a build that refuses a broken page |

No plan covers kittui's crashes and wrong rows beyond those the apps plan fixes on its way (Apps 1, 6, 7, 19, 20,
and 22), the cost of a large table (#92), or issues #62, #63, #64, #66, #68, #72, #73, #76, and #77.

## Corrections and additions

The writers of the plans opened every place a finding names. Where a finding is wrong, or its fix would break
something else, the plan says so and takes another way, and the findings page is as the reviewers wrote it.

- Workflow 1: a lease on the commit read before the fetch refuses in the very state `kit submit` sends a person to
  `kit rebase` for. The plan takes origin's commits first, and leases on the tip it fetched.
- Workflow 2: a test by git alone keeps every squash-merged branch. The plan asks the forge for the change's head.
- Delivery 13: a guard inside `place:remove` stops `place:retire`, which calls it. The plan shares a predicate.
- Serve 8 names `kit/bin/kit-flow:286`; the unchecked call is at `kit/modules/bash/across.sh:109`. Serve 23 is
  marked plausible, and it reproduces.
- Command line 5 misses a fifth `| grep --quiet` under `pipefail`, at `kit/modules/bash/1st.sh:437-438`. Command
  line 12 and 13 hold in a source checkout only: a release has a `version` file and never builds kittui.
- Infrastructure 18: `projects/tutorials/README.md:59-61` states that a link resolves against the base, so the code
  follows its document, and the plan changes both.
- Apps 20: the legend's title is at `projects/tui/internal/flow/legend.go:214`. Apps 24 holds when kittui runs by
  itself: through kit, an unknown tutorial is refused before kittui starts.
- Flow 27: the changes row has no `url` field in Go; only the underway row's `pr` has one.
- Gaps names two of CI's failing tests by the wrong file: they are `tests/exits.bats:14` and
  `tests/template.bats:291`. Gaps and issue #74 say `tests/help.bats` snapshots the help pages; it holds their
  structure only.

The writers also found what no reviewer did:

- A `kit.yml` that yq cannot read is taken for a repository with no kit: `kit hydrate` exits 0 and `kit lint`
  says every path passes (`kit/modules/bash/settings.sh:202`, `kit/modules/bash/materialize.sh:314`).
- `kit workspace branches` exits 0 where `kit workspace remotes` exits 1 on the same failure
  (`kit/modules/bash/workspace.sh:60`).
- `tests/linked.bats:14` holds a failing `gh` to mean "no pull request", and `tests/contract.bats:17` fails, where
  it means to skip, when mise has nothing installed.
- kittui's fixture `changes.json` lacks `fork`, `pushed`, and four keys under `closes[]` that kit prints, and
  `incoming-no-fetch.json` has a key that kit prints only after a fetch.
- `kit toolchains --json` and `kit tools --json` print one object a line, not an array, and `kit tools --json`
  exits 1 in this repository, where mise and nix are both usable and neither is chosen.

## The plans, as they are built

### Refusals that hold

The plan is built, under #141, in the 23 commits from `5861f1fd` to `99dad853`. The suite holds 779 tests, 43 of
them this plan's, and one fails, `tests/underway.bats:199`, as it did before the plan. Everything ran on macOS, and
one fix was also shown in a Linux container.

Its four decisions stand on their defaults. Where the plan was silent, or one part of it spoke against another, these
were ruled:

- `kit serve` has seven tools that read only. `toolchains` and `tools` join `check` and `lint` among those that run
  what the repository's files name: `tools` evaluates a flake, and `toolchains` runs `mise bin-paths` over the
  repository's `mise.toml`.
- A refusal changes nothing, so `kit uninstall`, `kit packages remove`, and `kit update` judge every region before
  the first write. The plan's task said to count the refusals during the sweep, which left a kit half removed.
- `kit packages remove` leaves an edited copy under the home base too, as `kit uninstall` does. An older test
  expected the copy gone, and puts the delivered content back before it removes the package.
- `kit serve` judges a path as the kernel reads it and as a shell's `cd` does, and refuses unless both are inside.
  The verbs' own reading of a path, in `context:relative`, is as it was.
- The whole-plan review found what no task had: `kit work`'s first push followed `push.default`, `kit rebase`
  unsigned the commits it rewrote, and `kit submit` called a branch at its remote when the remote had none. All
  three are fixed.

These are left as they are:

- A signer that fails during `kit rebase` is reported as a conflict, and a full disk during a signed commit as a
  failure to sign.
- A take over history that another session rewrote keeps the empty start commit twice.
- `kit rebase` given a path in another clone reads `require_signing` and the signing key from the directory it runs
  in.
- `kit work` can make or push the issue's branch before it refuses a desk that has that branch checked out, when the
  desk's branch was never pushed.
- An interrupt in the moment between `kit work`'s stash and its record of it, or between that record's end and the
  apply, leaves the work in the stash without a word.
- An unreadable `kit.yml`, or an unstaged one without the key, turns signing off for a commit; the plan Fail Closed
  owns that read.
- The length limit of a path given to `kit serve` counts the repository's root, so a real file deep under a long
  root is refused.
- Three commits name their co-author as `Claude Sonnet 5.5`, and the others as `Claude`.

### Fail closed

The plan is built in 42 commits, 35 under #69 and 7 under #67, from `5c0353dd` to `59e506f2`. The whole suite, run
after its last task, held 877 tests and failed one, `tests/underway.bats:199`, as before the plan. One commit,
`82eca1e8`, holds #67's change to `forges` beside #69's work: two sessions committed from one index at once.

Its five decisions stand on their defaults: `kit policy` fails at a desk too when the forge cannot answer; a JSON verb
that read a part prints it and exits 1; a list is read whole, to 10000 items; the default branch falls back to `main`;
and nothing approves a weakening, so the base's rule judges the pull request that loosens it. These were ruled:

- `list:read` carries a command's status in band and runs the command under errexit whoever calls it. `wait "$!"`
  after a process substitution lost the status under load, and bash hung in `lastpipe` under load, once in about
  500000 calls.
- A base with no `kit.yml` lets the branch's own settings judge it, so the pull request that installs the kit sets
  the first policy, as #2 does.
- On a branch, `policy`, `policy_engine`, `linked_exempt`, `facts`, `forge`, and `forges` are the base's, and so are
  its Rego, its whole `.kit/policy/`, and its kinds. `kit ci` judges the issue link and the policy before it runs any
  of the branch's commands. On a runner the base is the commit the event names; on GitLab, the target branch's tip,
  fetched each time with no tags.
- A pull request with more than 100 checks, review threads, or closing issues is read page by page. The plan's
  task refused it, and #2 has 108 checks and 130 closing issues: that is #142.
- A workspace member that exits 0 with a stage of `null` has nothing to read there, and is not a failure.
- `forge: none` says a primary has no forge the kit knows; a primary on a real host that nothing names has a change
  that cannot be read, not one that passes unchecked.

These are left as they are, and the first must be decided before a release:

- CI runs the branch's own kit: the workflow takes its version and `bash_path` from the branch's `kit.yml`, runs
  `kit tools install` first, and trusts the event file. A branch can set `version: unreleased`, or put its own
  modules on the path, and judge itself.
- `glab api --paginate` was not run against a GitLab server, and nothing was run on bash 4.4, the kit's floor.
- `kit facts` prints the facts with the disk's settings, and `kit policy --input` with the base's.
- On GitHub, a pull request's labels past the first 20, and the assignees of a closed issue past the 100 most recently
  updated, are not read.
- `tracker:is-forge` reads trackers it cannot read as another tracker's; its callers read the reference first.
- kittui does not name a workspace member whose rate limit could not be read, and does not draw the marks' `errors`.

On the way, the owner asked for four changes to kittui, each its own issue: `kit` with no verb opens the welcome
guide (#143); `=` opens the app menu while help's search has focus (#144); the welcome guide's terms are a table,
which moved the table out of the flow screen into `internal/table` (#145); and its "make it yours" step is one too,
read by the same code (#150).

### Workspaces 4

The plan is built under #89 in 13 commits, from `6079a0a3` to `0fdc2c2f`. The whole suite, run against an export of
its last task, failed one test of its own, the lock's, under load; the lock waits by the clock since then, and five
runs of its file held 40 of 40.

Its three decisions stand on their defaults: a group's `match` and `project` are refused by name until they are built;
a member is spelled one way, the host in lower case, and the path too on GitHub and GitLab; and a clone whose
`kit.yml` does not parse is filed under origin and said where its member is drawn. These were ruled:

- The plan's "JSON exits 0" for a member that failed yields to the contract Fail closed built: the document, and
  exit 1. kittui reads such a document as what was read.
- The owner's rule replaced the plan's usage lines: an option with a value is `--option=value`, so `kit workspaces`
  takes `--add=<name>`, `--rename=<name> <new-name>`, `--use=<workspace>`, and the like.
- Inside a repository is a working tree, for every verb that chooses where to start: a bare repository and a `.git`
  directory are outside. Outside one, `kit flow` and `kit workspace` draw the last group used; the workspace file is
  read only for a group, so a broken file never stops a verb that does not need it.
- Every edit of the workspace file takes a lock that names its process; a dead holder's lock is taken over in one
  step, and a live one's is waited for by the clock.
- kittui's workspace menu draws no line that opens nothing, and asks kit for no group when what it shows is none.

These are left as they are:

- `kit workspaces --current` inside a repository still reads the file, so a broken file shows as kittui's refusal
  there.
- A third process can take the lock in the instant a live holder's link is moved aside, and a process number reused
  by a live process is waited for as a holder.
- `--workspace <workspace>` keeps the space form until #153 moves it before the verb as `--workspace=<workspace>`;
  every other option is #160's.
