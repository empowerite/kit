# Workspaces 3b: kittui's Flow across a workspace Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or
> superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** kittui's Flow draws a group workspace: every tab gathers its rows from every cloned member, each row naming
its member, and a 📚 repos tab, first, lists the members with a count per tab that opens that tab filtered to the
member.

**Architecture:** `kit.Runner` gains the group its `kit flow` reads span, and every read carries `--workspace <group>`;
the root sets it whenever kit says the workspace is a group, and `kit flow --workspace` on a terminal hands it to
kittui. Over a group, every stage's JSON is 3a's `{rows, errors}`: Flow decodes it into the slices it already holds,
desk, worktrees, and incoming becoming one entry per member, every row carrying `member`, its key made unique by it,
and a member that failed drawn as its line. `repos` is a stage like the others, with its fields in `fields.json`, its
counts drawn from the other stages' rows.

**Tech Stack:** Go, Bubble Tea v2, kittui's `grid`/`table`/`tabs`; bash and bats for kit-flow's hand-off; jq for
`glyphs.json`.

**Spec:** `docs/superpowers/specs/2026-11-08-workspaces-design.md`: step 3 of § The order of the work, § Flow over a
workspace, decisions 5–8. Plan 3a (`docs/superpowers/plans/2026-11-09-workspaces-3a-kit-flow-across-a-workspace.md`)
built the data: its Interfaces blocks give every JSON shape this plan reads.

## Global Constraints

- kit commit only, never kit submit or push; every subject starts with `#89`. `export PATH=$PWD/kit/bin:$PATH` first.
- Peers share the worktree: stage named paths or hunks only, guard that the index holds only them, never leave anything
  staged. kittui tests run in a clean copy of HEAD with the task's patch applied and `mise*.toml` removed; compare the
  staged index with the tested copy before committing.
- Before touching `cmd/kittui/main.go`, `internal/flow/view.go`, or `kit/bin/kit-flow`, ask the TUI fork
  (`uds:/tmp/cc-socks/77485.sock`, issues #96, #97, #123) whether it holds them.
- Over a single repository, nothing Flow draws changes but the 📚 repos tab, first: every existing golden and test
  built with explicit `Options.Stages` passes unchanged.
- A member is host and path, `github.com/empowerite/kit`, as kit names it. Its short name, drawn wherever a member is
  drawn, is the path past the owner when every member of the workspace shares a host and owner, else host and path.
- Step 3 reads only cloned members. A member not cloned has no rows in any tab, and a dim `-` for every count; ☁️,
  forge reads without a checkout, and an issue shared by trackers counted once are step 4's.
- Plain clicks only; a line or count that cannot act now stays, dim. Two-cell emoji in menus; no circled glyphs.
- Lines are 120 cells at most.

## Review Focus

1. A member whose read of a stage failed, in a group: that tab shows the other members' rows and one dim-red line for
   it, `❌ <short name>: <kit's line>`, not a refusal of the whole tab; the repos tab's count for it is ❌, leading to
   that tab filtered to it.
2. Two members whose rows share a key, the same issue number, tag, or build id: two rows, each one selectable, opened,
   and jumped to on its own; a pull request's link from underway or desk lands on its own member's row.
3. Switching from a single repository to a group and back: every read after the switch to the group carries
   `--workspace <group>`, and none after the switch back.
4. A group with no member cloned: every tab shows its empty line, the repos tab lists each member, `not cloned`, with
   a dim `-` for each count, and nothing is a refusal.
5. Backlog rows of two members selected: Assign to Me runs `kit assign` once in each member's checkout, with only that
   member's issues; the actions that need one member (Start Local Work Together…, Add to Pull Request…, Gather Under a
   Parent…, Close as Duplicate of…) are dim.

## Calls this plan makes

- The field is `member`, not the spec's `repository`: an issue's `repository` is already its tracker's, and 3a tags
  every row `member`.
- kittui, not kit, puts `repos` first over a single repository, so `kit flow --stages` prints what it prints today.
  The tabs' numbers move up one: repos is 1.
- A member not cloned shows dim `not cloned`, as `kit flow repos` prints, not ☁️: ☁️ draws one cell or two by
  terminal, and ☁️ members arrive with step 4.
- A repo row's error is in its opened row, not a popover, as every other row's detail is.
- desk counts a cloned member's desk, 1; incoming, the commits, branches, and tags it brought.
- Outside every repository with no workspace, the repos tab shows kit's line, as any stage kit refuses does.
- A repository given by `--workspace <host/path>` keeps reading through `--workspace`, as kit draws it, rather than
  moving to its checkout.

---

## File structure

- Modify `projects/tui/internal/kit/kit.go`: `Runner.Group`, carried by `Stages`, `Status`, `Marks`, `Quota`.
- Modify `projects/tui/cmd/kittui/main.go`, `internal/screens/screens.go`: the group from `--workspace`, and the root
  keeping `Runner.Group` as kit says the workspace.
- Modify `kit/bin/kit-flow`: on a terminal, `--workspace` hands off to kittui; `tests/across.bats`.
- Modify `projects/tui/internal/flow/flow.go` (decode, shape, the stage struct), `rows.go` (`member`, keys, links,
  failures), `fields.go` and `fields.json` (the `member` field, the repos stage), `host.go` as needed.
- Create `projects/tui/internal/flow/repos.go`: the repos stage's JSON, rows, and counts; `repos_test.go`;
  `testdata/repos.json`, `testdata/repos.golden`.
- Create `projects/tui/internal/flow/group_test.go`: the stages over a group.
- Modify `projects/tui/internal/grid/panel.go`: export `GroupedBy`.
- Modify `kit/modules/jq/glyphs.json`: the repos stage and its group; regenerate with `kit generate`.
- Modify `projects/tui/internal/flow/act.go`: acting in a row's member.
- Modify `docs/workspaces.md`, `projects/tui/README.md`.

## Task 1: Flow's reads span the workspace kit says

**Files:**

- Modify: `projects/tui/internal/kit/kit.go`, `projects/tui/internal/kit/kit_test.go`
- Modify: `projects/tui/internal/screens/screens.go`, `projects/tui/internal/screens/workspace_test.go`
- Modify: `projects/tui/cmd/kittui/main.go` and its test
- Modify: `projects/tui/internal/flow/flow.go` (Open, Model), `projects/tui/internal/flow/flow_test.go`
- Modify: `kit/bin/kit-flow`, `tests/across.bats`

**Interfaces:**

- Consumes: 3a's `kit flow … --workspace <workspace>` on every form; `kit workspaces --current --json [--workspace
  <w>]` through `Runner.Workspace`.
- Produces:
  - `kit.Runner.Group string`: the group every `kit flow` read spans, none for the place `Dir` is. `Stages`, `Status`,
    `Marks`, and `Quota` append `--workspace <Group>` after their words when it is set; `Me`, `Act`, and the rest never
    do.
  - `flow.Model.group string`, from the runner at `New`, and `flow.Model.ws kit.Workspace`, kept from every
    `shell.WorkspaceMsg` (its `Members` give the checkouts later tasks run in and the short names).
  - `screens.New` asks kit for `r.Group` at Init in place of `""`; at the first answer, the screen open is made anew
    only when kit's group differs from `r.Group`. Every answer sets `m.runner.Group` to the group's name, or `""` for a
    repository or none, before any screen is made.
  - `main` takes `--workspace <w>` and `--workspace=<w>` out of flow's words into `r.Group`.
  - `kit flow --workspace <w>` on a terminal, with no stage, form, or `--text`, execs `kittui flow --workspace <w>
    <words>…`.

- [ ] **Step 1: Write the failing tests.**
  - `kit_test.go`, `TestAGroupsReadsCarryTheWorkspace`: a runner over the fake kit with `Group: "w"`; `Status(ctx,
    "backlog", []string{"--mine"})` prints `{"stage": "backlog", "args": "--mine --workspace w"}`; with `Group: ""`,
    `"args": "--mine"`. `Stages()` with `Group: "w"` still gives `["backlog", "incoming"]`.
  - `flow_test.go`, `TestOverAGroupEveryLoadCarriesTheWorkspace`: `New` with `runner(t)` whose `Group` is `"w"`, and
    `FAKE_FLOW_CALLS` set; after Init, every line `calls` reads ends in `--workspace w`.
  - `workspace_test.go` (screens), `TestTheRunnerFollowsTheGroupKitSays`: a root over a fake whose current workspace
    is a repository; a pick of group `w` remakes Flow with `runner.Group == "w"`; a pick back of the repository remakes
    it with `runner.Group == ""` and `Dir` its checkout.
  - `workspace_test.go`, `TestAGroupGivenAtStartIsAskedAndNotMadeTwice`: `New` over a runner with `Group: "w"`; Init's
    ask runs `workspaces --current --json --workspace w`; the answer, group `w`, leaves the first screen as it was made
    (the same `flow.Model`, by its loads not run again).
  - main's test, `TestFlowTakesTheWorkspaceFromItsWords`: `--workspace w --no-fetch` and `--workspace=w --no-fetch`
    each give `r.Group == "w"` and flow's words `["--no-fetch"]`.
  - `tests/across.bats`, `on a terminal, kit flow --workspace hands off to kittui; --json stays across the members`:
    with `fake-tui` (copied from `flow.bats`) and `helper.bash`'s `pty`, `kit flow --workspace w --no-fetch` prints
    `kittui native-native flow --workspace w --no-fetch`; `kit flow --workspace w --json` prints 3a's picture JSON and
    nothing of kittui.

- [ ] **Step 2: Run them.** `cd projects/tui && go test ./internal/kit/ ./internal/flow/ ./internal/screens/
  ./cmd/kittui/ -run 'Group|Workspace'` and `KIT_TOOLCHAIN=mise kit/bin/kit tools run bats tests/across.bats`.
  Expected: each new test fails: no `Group` field, no `--workspace` in the calls, kittui not exec'd.

- [ ] **Step 3: Implement.** `Runner.Group` and one helper that appends it, used by the four reads. `main` parses
  flow's words. `screens`: keep the group asked at start; set `runner.Group` in `answered`. `flow.Model` keeps `group`
  and `ws`. In `kit-flow`, inside the `--workspace` branch before `across:flow`, exec kittui under the same condition
  the exec inside a repository has when this task starts (the TUI fork's #123 may have changed it), without its `git
  rev-parse`, passing `--workspace "${workspace}"` before the words.

- [ ] **Step 4: Run the tests again, then `go test ./...` and `tests/flow.bats`, `tests/across.bats`.** Expected: all
  pass; every flow golden unchanged.

- [ ] **Step 5: Commit**, subject `#89 kittui: Flow's reads carry --workspace while kit says the workspace is a group,
  and kit flow --workspace on a terminal opens kittui on it`.

## Task 2: Every stage's rows across the members

**Files:**

- Modify: `projects/tui/internal/flow/flow.go` (stage, decode, shape), `projects/tui/internal/flow/rows.go`
- Create: `projects/tui/internal/flow/group_test.go`

**Interfaces:**

- Consumes: Task 1's `Model.group`, `Model.ws`; 3a's stage document over a workspace: `{"rows": [...], "errors":
  [{"member", "error"}]}`, each row its stage's object with `"member"`; desk, worktrees, and incoming one object a member.
- Produces:
  - ``Member string `json:"member"` `` on `issue`, `underway`, `pull`, `run`, `release`, `checkout`, `worktrees`,
    `incoming`; empty over a single repository.
  - `stage.desks []checkout`, `stage.trees []worktrees`, `stage.ins []incoming` in place of `desk`, `trees`, `in`: one
    entry over a single repository, one a cloned member over a group. `stage.failed []failure`, `failure{Member,
    Error string}`, from `errors`.
  - `(s *stage) decode(data []byte, over bool) error`: over a group, the envelope, then its rows; else as now, an
    object stage's object as a one-entry slice. `shape` over a group refuses anything but an object with `rows`:
    `kit flow <stage> --workspace --json printed <what> where {rows, errors} was expected`.
  - `(m Model) keyOf(member, key string) string`: `key` for no member, else `member + " " + key`. Every row key and
    every `jumpTo` key goes through it; `pullKey(member string, number int)`, `pullOf(member, branch string, desk
    bool)` match only that member's pull requests.
  - `(m Model) originOf(member string) Origin`: `m.origin` for none, else the member's host and path, as `parseRemote`
    reads `https://<member>`; every link a row builds uses its row's.
  - `(m Model) short(member string) string`: the member's short name, as Global Constraints say, from `m.ws.Members`;
    host and path until kit has said them.
  - Each failed member is one aside row after the stage's rows, key `keyOf(member, "failed")`, cells
    `span(style.Refusal(Failed + " " + short + ": " + error))`. Over a group, incoming's remote headings start with
    the member's short name, `kit · 📡 origin · github · github.com`.

- [ ] **Step 1: Write the failing tests** in `group_test.go`, each a model with `group: "w"` fed `answer(stage, …)`:
  - `TestOverAGroupEachStageDrawsEveryMembersRows`: backlog `{"rows": [{"number": 1, "title": "a1", "member":
    "github.com/o/a"}, {"number": 1, "title": "b1", "member": "github.com/o/b"}], "errors": []}` gives two rows, keys
    `github.com/o/a #1` and `github.com/o/b #1`, each opened alone by right.
  - `TestOverAGroupTheDeskIsAPerMemberRow`: desk with two members' objects gives two rows, each branch drawn.
  - `TestAMemberThatFailedIsItsLineAndTheOthersDraw` (Review Focus 1): changes `{"rows": [<a's pull>], "errors":
    [{"member": "github.com/o/b", "error": "gh cannot read o/b"}]}`: a's row, then `❌ b: gh cannot read o/b`; the
    stage's `refusal` is empty.
  - `TestAPullRequestsLinkLandsOnItsMembersRow` (Review Focus 2): underway rows of a and b closing their own `#7`; a
    click on b's change glyph selects changes with the cursor on `github.com/o/b #7`.
  - `TestOverAGroupAnArrayIsNoEnvelope`: backlog `[]` over a group is the one refusal line naming `{rows, errors}`.
  - `TestOverOneRepositoryTheStagesDecodeAsBefore`: `group: ""`, desk's fixture: one row, key its path.

- [ ] **Step 2: Run them.** `go test ./internal/flow/ -run 'OverAGroup|AMember|APullRequests|OverOneRepository'`.
  Expected: fail to compile (`decode`'s arguments, `desks`), then fail on two rows not drawn.

- [ ] **Step 3: Implement** the Interfaces above in `flow.go` and `rows.go`. `here`, `deskRows`, `worktreeRows`,
  `incomingRows`, `empty`, and every reader of `desk`, `trees`, `in` walk the slices. A stage whose only rows are
  failures shows them, not its empty line.

- [ ] **Step 4: Run the tests, then `go test ./...`.** Expected: all pass; every golden unchanged.

- [ ] **Step 5: Commit**, subject `#89 kittui: over a group every tab draws each member's rows, a row's key and links
  its member's, and a member whose read failed is one line among the others`.

## Task 3: `member`, a field of every stage

**Files:**

- Modify: `projects/tui/internal/flow/fields.json`, `projects/tui/internal/flow/fields.go`, `rows.go`, `flow.go`
  (`autoGroup`)
- Modify: `projects/tui/internal/grid/panel.go` (export `GroupedBy`)
- Test: `projects/tui/internal/flow/group_test.go`, `projects/tui/internal/flow/fields_test.go`

**Interfaces:**

- Consumes: Task 2's `Member`, `short`, `keyOf`.
- Produces:
  - In `fields.json`, on every stage but incoming, whose rows are drawn by hand: `{"id": "member", "label": "member",
    "type": "enum", "path": ".member", "sort": true, "group": true, "filter": true, "renderings": ["member"], "dim":
    true, "placement": "hidden"}`, after `id` (or the stage's first field).
  - Every record's `"member"` value: none over a single repository; else `table.Value{Raw: []string{member}, Of:
    short(member), Link: jumpTo("repos", member)}`. The rendering `member`, registered in `fields.go`'s `init` as
    `plainName` is, draws `Of`.
  - `grid.Panel.GroupedBy(ids []string) Panel`, today's `groupedBy`, exported.
  - `autoGroup` over a group groups by `member` outermost, then by what `AutoGrouped` gives, once, as it does now.

- [ ] **Step 1: Write the failing tests.**
  - `TestOverAGroupTheRowsGroupByMemberAsTheyLoad`: backlog with rows of a and b loads grouped by `member`, headings
    `a` and `b`, the short names, since both share `github.com/o`.
  - `TestAMembersNameLeadsToItsReposRow`: a click on a backlog heading's member, over a group with repos in the
    picture, selects repos with the cursor on `github.com/o/a`.
  - `TestMembersOfTwoOwnersAreNamedInFull`: `ws.Members` `github.com/o/a`, `github.com/p/b`: headings
    `github.com/o/a`, `github.com/p/b`.
  - `fields_test.go`: every stage but incoming and repos has `member`, hidden, groupable, filterable.

- [ ] **Step 2: Run them.** `go test ./internal/flow/ ./internal/grid/ -run 'Member|Fields'`. Expected: fail, no
  `member` field.

- [ ] **Step 3: Implement** the field, the rendering, `GroupedBy`, and `autoGroup`'s member first.

- [ ] **Step 4: Run the tests, then `go test ./...`.** Expected: all pass; single-repository goldens unchanged, since
  the field is hidden and holds nothing there.

- [ ] **Step 5: Commit**, subject `#89 kittui: member is a field of every tab, over a group the rows group by it as they
  load, and a member's name leads to its row on the repos tab`.

## Task 4: The 📚 repos tab

**Files:**

- Modify: `kit/modules/jq/glyphs.json`; regenerate (`kit generate`) `projects/tui/internal/flow/glyph.go` and what else
  it writes
- Create: `projects/tui/internal/flow/repos.go`, `projects/tui/internal/flow/repos_test.go`
- Create: `projects/tui/internal/flow/testdata/repos.json`, `projects/tui/internal/flow/testdata/repos.golden`
- Modify: `projects/tui/internal/flow/fields.json`, `flow.go` (`known`, `decode`, `records`, `Open`, `jump`),
  `rows.go` (`empty`)
- Modify: `docs/workspaces.md`

**Interfaces:**

- Consumes: 3a's `kit flow repos --json [--workspace <w>]`, `{rows, errors}` in every case, a row per member:
  `{repository, host, path, forge, remotes: [{name, url, forge}], checkout, checkouts, by, branch, ahead, behind,
  error}`, the last four null for a member not cloned. Tasks 2–3's `member`, `short`, `keyOf`.
- Produces:
  - `glyphs.json`: the group `{"name": "workspace", "color": "13", "stages": ["repos"]}`, first, and the stage
    `{"stage": "repos", "glyph": "Workspace", "legend": [Workspace, GitHub, GitLab, Desk, Failed]}`.
  - `repo` (repos.go), the row's JSON; `stage.repos []repo`; `known["repos"]`; `expected["repos"]`, the envelope.
  - `fields.json`'s repos stage: `repository` (the short name, sort, the row's key the full member), `forge` (enum,
    🐙 github, 🦊 gitlab), `remotes` (label pills, a remote's name each), `checkout` (its path, or dim `not cloned`),
    `branch` and `ahead` as desk defines them, then a count field per stage, `backlog`, `underway`, `desk`, `worktrees`,
    `changes`, `builds`, `incoming`, `releases`, right-aligned and sortable, then `by`, hidden. Presets: Default.
  - `(m Model) countOf(stage, member string) table.Value`: none for a stage not in the picture; `Else: "-"` for a
    member not cloned; `Else: "…"` while the stage loads; `Raw: [Failed]` where the stage refused or the member is in
    its `failed`; else the member's rows in it, asides apart, incoming's the commits, branches, and tags. Each but the
    first has `Link: onlyTo(stage, member)`.
  - `onlyTo(stage, member string) string`: `jumps + stage + "?" + member`. `jump` reads it: the stage selected, its
    panel `FilteredBy("member", grid.Filter{Values: []string{member}})`; for no member, the stage alone.
  - A repo row whose `error` is set draws `❌` before its branch, and its line in the row's detail.
  - `Open` puts `repos` first where kit's `--stages` does not name it.

- [ ] **Step 1: Write the failing tests** in `repos_test.go`:
  - `TestReposDrawsAMemberARow`: the fixture `repos.json`, two members of `github.com/o`, a cloned with a remote and
    `main ↑1`, b not cloned; golden `repos.golden` at 120 wide.
  - `TestACountOpensItsTabFilteredToTheMember`: over a group, backlog with a's two rows and b's one; repos' backlog
    counts `2` and `1`; a click on b's `1` selects backlog showing b's row alone.
  - `TestAMemberNotClonedCountsADimDashNeverZero` (Review Focus 4): with no member cloned, every count of every row is
    `-`, no stage is a refusal, and each tab shows its empty line.
  - `TestAMemberThatFailedCountsAFailure` (Review Focus 1): changes failing for b: b's changes count is ❌, its click
    opens changes filtered to b, showing b's line.
  - `TestOpenPutsReposFirst`: `Open` over the fake with `FAKE_FLOW_STAGES="backlog desk"` gives `repos backlog desk`;
    with `FAKE_FLOW_STAGES="repos backlog"`, `repos backlog`.
  - `TestOverOneRepositoryACountOpensItsTab`: group `""`, one row, `by` `here`; a click on its desk count selects desk,
    no filter set.

- [ ] **Step 2: Run them.** `go test ./internal/flow/ -run 'Repos|Count|Open'`. Expected: fail, `repos` not drawn.

- [ ] **Step 3: Implement.** Edit `glyphs.json`, run `kit generate`, then `kit generate --check`. Write `repos.go`,
  the catalog entry, `countOf`, `onlyTo`, and `jump`'s filter. Add a section to `docs/workspaces.md`, "Flow over a
  workspace in kittui", and take the 📚 repos tab off its "Still to come".

- [ ] **Step 4: Run the tests with `-update` once for the new golden, read it, then `go test ./...`, `kit lint`,
  `tests/flow.bats`.** Expected: all pass; `kit help flow`'s legend gains the repos tab's lines and nothing else.

- [ ] **Step 5: Commit**, subject `#89 kittui: the 📚 repos tab, first, lists the workspace's members, each with its
  forge, remotes, checkout, and branch, and a count per tab that opens it filtered to the member`.

## Task 5: Acting from a row runs in its member

**Files:**

- Modify: `projects/tui/internal/flow/act.go`, `flow.go` (`whoami`)
- Test: `projects/tui/internal/flow/act_test.go`
- Modify: `projects/tui/README.md`

**Interfaces:**

- Consumes: Task 2's `Member` and keys; Task 1's `ws.Members` (each `Repository`, `Checkout`).
- Produces:
  - `call{dir string, words []string}`; `run(calls []call) tea.Cmd` runs each with the runner's `Dir` set to `dir`,
    none keeping the runner's own.
  - `(m Model) refsByMember() []memberRefs`, `memberRefs{member, dir string, refs []string}`: the selected backlog
    rows' issue references, `#45`, grouped by member in the rows' order; over a single repository one entry, no dir.
  - Over a group, each action that takes refs runs once per entry, in its checkout. Start Local Work Together…, Add to
    Pull Request…, Gather Under a Parent…, and Close as Duplicate of… are dim while the selection spans two members.
    New… runs in the member of the row under the cursor, dim with none. Copy Links uses each row's `originOf`.
  - `whoami` over a group runs `kit flow --me` in the first cloned member's checkout; with none, `@me` matches no one.

- [ ] **Step 1: Write the failing tests** in `act_test.go`, over a group whose fake records each call's directory:
  - `TestAssignToMeRunsOnceInEachMember` (Review Focus 5): rows a `#1`, b `#2`, both selected; Assign to Me runs
    `assign #1` in a's checkout, then `assign #2` in b's.
  - `TestTheActionsOfOneMemberAreDimAcrossTwo`: the four named above are dim; with only a's rows selected, each acts.
  - `TestOverAGroupMeIsAskedInAMember`: `--me` runs in the first cloned member's checkout.

- [ ] **Step 2: Run them.** `go test ./internal/flow/ -run 'AssignToMe|OneMember|MeIsAsked'`. Expected: fail: refs
  carry the member, one call in the root's directory.

- [ ] **Step 3: Implement** the Interfaces above. Note in `projects/tui/README.md`'s Flow section what a group draws:
  the repos tab, the member field, and acting in the row's member.

- [ ] **Step 4: Run the tests, then `go test ./...`.** Expected: all pass.

- [ ] **Step 5: Commit**, subject `#89 kittui: over a group an action runs in each selected row's member, those that
  need one member dim across two, and @me is read in a member`.
