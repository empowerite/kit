# Workspaces, step 2: the 📚 control Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or
> superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** The 📚 on kittui's title line becomes one control in `shell`: it names the workspace, drops down the groups
the repository belongs to and the favorites, and switching between them reloads the screens; over a group, Welcome
shows every member's checks together.

**Architecture:** kit stays the only reader of `workspaces.yml`: `kit workspaces --current --json` grows the fields
the control needs (the group's members with their checkouts, the groups a repository belongs to, the favorites), in
one call. kittui's root (`internal/screens`) loads that once at start, and again on a pick, and tells every screen made
by a `shell.WorkspaceMsg`, as it tells settings. A pick that changes the place the screens read is a fresh start: the
root closes and remakes the screens made, as they would be made there. Welcome over a group runs `kit welcome --checks
--json` in every member's checkout and composes the answers.

**Tech Stack:** bash + jq + bats (kit), Go + Bubble Tea v2 + lipgloss v2 (kittui).

**Spec:** `docs/superpowers/specs/2026-11-08-workspaces-design.md` (§ The 📚 control, § Welcome and Tutorials over a
workspace, § Which workspace kit starts on). Step 1's plan:
`docs/superpowers/plans/2026-11-08-workspaces-1-kit-workspaces.md`.

## Global Constraints

- Issue #89; every commit subject starts `#89 kit workspaces: …` or `#89 kittui: …`. `kit commit` only, never submit
  or push. `export PATH=$PWD/kit/bin:$PATH` first.
- Peers share the worktree and its index: commit only named paths, check the index holds exactly them, leave nothing
  staged. `internal/flow`, `internal/grid`, and `internal/table` are kit-e7's until it says otherwise (Task 7 waits).
- kittui reads workspaces through `kit workspaces --json` alone; no YAML in Go.
- A single repository draws `📚 owner/repo`; a group `📚 <name> <count>`, the count dim; both in the repository's
  magenta (`style.Repo`). No workspace: 📚 alone, dim.
- Menus: every action a plain click; a line that cannot act now stays, dim; a line's keys never on the keys bar; the
  dropdown's glyphs two-cell emoji with no variation selector.
- Over a single repository, Welcome is exactly as now (its goldens do not change). Tutorials and Help do not change
  with the workspace.
- `kit welcome` gains no `--workspace`: kittui runs it in each member's checkout.

## Review Focus

1. A kit without `kit workspaces`, or one that refuses (a bad `workspaces.yml`): every screen still opens, the title
   keeps `📚 owner/repo` read from origin, and the dropdown says kit's line, dim. (Task 5 test.)
2. A group with a member that is not cloned, or whose `kit welcome` refuses: the other members' checks still show; the
   uncloned one is named as not cloned, the refusing one's line shows as its failed check. (Task 6 tests.)
3. The same check id in two members with different results: each keeps its own cursor place, later, and note; a fix
   runs in that member's checkout, not where kittui started. (Task 6 test.)
4. A pick answered after another pick: only the latest pick lands. (Task 5 test.)
5. The first load at start, inside a repository: the screen already open is not remade, so the checks already running
   are not run twice. (Task 5 test.)

---

### Task 1: `kit workspaces --current --json` carries what the control needs

**Files:**

- Modify: `kit/modules/bash/workspaces.sh` (add `workspaces:menu-json`)
- Modify: `kit/bin/kit-workspaces` (the `current` action; the header's JSON paragraph)
- Modify: `docs/workspaces.md` (the JSON paragraph)
- Test: `tests/workspaces.bats`

**Interfaces:**

- Consumes: `workspaces:current`, `workspaces:fixed`, `workspaces:names`, `workspaces:favorites`, `workspaces:kind`,
  `workspaces:checkouts`, `workspaces:member-json` (step 1).
- Produces: `--current --json` (and `--jq`) prints `{kind, name, member, checkout, why, members, groups, favorites}`:
  - `members`: a group's fixed members, each as `--show` prints a member (`repository`, `checkout`, …); `[]` otherwise.
  - `groups`: for a repository with a member, every group whose fixed members hold it, `{name, members}` (a count), in
    the file's order; `[]` otherwise.
  - `favorites`: each favorite that names a group or a repository, `{name, kind, members, checkout}`: `members` a
    group's count, else null; `checkout` a repository's, else null. A favorite naming a group no longer in the file is
    left out.
  - The text form is unchanged and walks no roots. The JSON forms walk the roots at most once, and only when a member
    or a repository favorite needs a checkout.

- [ ] **Step 1: Write the failing tests** in `tests/workspaces.bats`, using step 1's helpers (`clone-at`,
  `workspaces-file`):
  - `@test "--current --json gives a group's members with their checkouts, and the favorites"`: a file with group `w`
    of `github.com/o/a` (cloned under the root) and `github.com/o/b` (not), favorites `[w, github.com/o/a]`; run outside
    a repository after `--use w`. Assert with jq: `.kind == "group"`, `.members | map(.repository) ==
    ["github.com/o/a","github.com/o/b"]`, `.members[1].checkout == null`, `.favorites == [{name:"w", kind:"group",
    members:2, checkout:null}, {name:"github.com/o/a", kind:"repository", members:null, checkout:<a's path>}]`.
  - `@test "--current --json inside a member names the groups it belongs to"`: groups `w` (a, b) and `v` (a); run in
    a's checkout. Assert `.groups == [{name:"w",members:2},{name:"v",members:1}]` and `.members == []`.
  - `@test "a favorite naming a group no longer in the file is left out of --current"`: favorites `[gone]`; assert
    `.favorites == []` and exit 0.
  - `@test "--current as text walks no roots"`: a root holding a directory that is not readable
    (`chmod 000`) under which a walk would print an error; assert the text output's stderr is empty. (Use the same
    pattern as step 1's "naming a workspace walks no roots" test.)

- [ ] **Step 2: Run them to see them fail**

Run: `KIT_TOOLCHAIN=mise kit tools run bats tests/workspaces.bats --filter 'current'`
Expected: the four new tests FAIL (`members` null / absent), step 1's `--current` tests still pass.

- [ ] **Step 3: Implement `workspaces:menu-json <current-json>`** in `kit/modules/bash/workspaces.sh`: reads the
  current JSON, prints it with the three fields added, as Produces says; calls `workspaces:checkouts` once, into a
  local, only when `.kind == "group"` or a favorite is a repository. In `kit-workspaces`'s `current` action, pipe
  through it when `form` is `--json` or `--jq`. Update the header's `--current's is …` sentence and `docs/workspaces.md`
  to name the three fields and say the text form leaves them out.

- [ ] **Step 4: Run the file and the verb-wide tests**

Run: `KIT_TOOLCHAIN=mise kit tools run bats tests/workspaces.bats tests/verbs.bats`
Expected: all pass. (`tests/verbs.bats` checks the header grammar.)

- [ ] **Step 5: Commit** `kit/modules/bash/workspaces.sh kit/bin/kit-workspaces docs/workspaces.md
  tests/workspaces.bats`
  as `#89 kit workspaces: --current's JSON gives a group's members, the groups a repository belongs to, and the
  favorites`.

---

### Task 2: kittui reads the workspace from kit

**Files:**

- Modify: `projects/tui/internal/kit/kit.go`
- Modify: `projects/tui/internal/kit/testdata/fake-kit` (a `workspaces` branch)
- Test: `projects/tui/internal/kit/kit_test.go`

**Interfaces:**

- Produces, in package `kit`:

```go
type Workspace struct {
    Kind      string              `json:"kind"`     // repository, group, or none; "" while unknown
    Name      string              `json:"name"`
    Member    string              `json:"member"`   // host/path, "" for a repository with no remote
    Checkout  string              `json:"checkout"`
    Why       string              `json:"why"`
    Members   []WorkspaceMember   `json:"members"`
    Groups    []WorkspaceGroup    `json:"groups"`
    Favorites []WorkspaceFavorite `json:"favorites"`
}
type WorkspaceMember struct {
    Repository string `json:"repository"`
    Checkout   string `json:"checkout"`
}
type WorkspaceGroup struct {
    Name    string `json:"name"`
    Members int    `json:"members"`
}
type WorkspaceFavorite struct {
    Name     string `json:"name"`
    Kind     string `json:"kind"`
    Members  int    `json:"members"`
    Checkout string `json:"checkout"`
}
// Workspace runs kit workspaces --current [--workspace given] --json, in Dir; given "" names none, as kit starts.
func (r Runner) Workspace(ctx context.Context, given string) (Workspace, error)
// UseWorkspace runs kit workspaces --use name, which keeps a group as the last used.
func (r Runner) UseWorkspace(ctx context.Context, name string) error
```

  Both use `within(ctx, callLimit, …)` with `NO_COLOR=1`, as `WelcomeChecks` does; a refusal is the `*ExitError`.

- The fake's `workspaces` branch: `--current` prints the file `$FAKE_WORKSPACES/<given>.json`, `/` in the name as `_`,
  `current.json` when none is given; exits 2 with a `❌` line when the file is missing or `FAKE_WORKSPACES` is unset.
  `--use <name>` appends `use <name>` to `$FAKE_KIT_RECORD` when set, and exits 0.

- [ ] **Step 1: Write the failing tests** `TestWorkspaceIsWhatKitWorkspacesCurrentSays` (a `current.json` fixture
  written by the test with a group of two members, one `checkout: null`, and a repository favorite; assert the decoded
  struct field by field, the null checkout as `""`), `TestWorkspaceGivenNamesItToKit` (given
  `github.com/o/a` reads `github.com_o_a.json`), `TestWorkspaceRefusedIsKitsLine` (`FAKE_WORKSPACES` unset →
  `*ExitError`
  with code 2), and `TestUseWorkspaceTellsKit` (the record holds `use w`).

- [ ] **Step 2: Run them to see them fail**

Run: `cd projects/tui && go test ./internal/kit -run 'Workspace' -count=1`
Expected: FAIL, `r.Workspace undefined`.

- [ ] **Step 3: Implement** the types, the two methods, and the fake's branch.

- [ ] **Step 4: Run the package**

Run: `cd projects/tui && go test ./internal/kit -count=1`
Expected: PASS.

- [ ] **Step 5: Commit** `projects/tui/internal/kit/kit.go projects/tui/internal/kit/kit_test.go
  projects/tui/internal/kit/testdata/fake-kit` as `#89 kittui: kit.Workspace is what kit workspaces --current says`.

---

### Task 3: `shell.BooksMenu`, the control

**Files:**

- Create: `projects/tui/internal/shell/books.go`
- Test: `projects/tui/internal/shell/books_test.go`

**Interfaces:**

- Consumes: `kit.Workspace` (Task 2); `shell.Dropdown`, `shell.Box`, `shell.Over`, `shell.Item`, `style.Repo`,
  `style.Off`.
- Produces, in package `shell`:

```go
const Books = "📚" // the workspace's glyph, on the title line and beside each group in its dropdown

// WorkspaceMsg tells a screen its workspace, when it is made and whenever kit says it anew.
type WorkspaceMsg struct {
    Workspace kit.Workspace
    Refusal   string // kit's line when it could not say, the workspace then as it was
}
// PickMsg asks kittui to switch to a workspace: a group's name or a repository's host/path, as kit names it.
type PickMsg struct{ Workspace string }
func Pick(workspace string) tea.Cmd

// BooksMenu is the 📚 control on a screen's title line: the workspace it names, the origin's owner/repo it names until
// kit says, kit's line when kit could not, and its dropdown.
type BooksMenu struct {
    Workspace kit.Workspace
    Fallback  string
    Refusal   string
    Dropdown
}
func (b BooksMenu) Told(msg WorkspaceMsg) BooksMenu       // keeps the workspace, or kit's line with the old one
func (b BooksMenu) Label() string                          // what the title draws after " · "
func BooksAt(title string) int                             // the 📚's column after a title line drawn so: width+3
func (b BooksMenu) On(x, y, at int) bool                   // a click on the control's cells, on the title line
func (b BooksMenu) Items() []Item
func (b BooksMenu) Opened() BooksMenu
func (b BooksMenu) Key(k tea.KeyPressMsg) (BooksMenu, string)            // the workspace picked, "" for none
func (b BooksMenu) Click(x, y, at, room int) (BooksMenu, string)
func (b BooksMenu) Overlay(lines []string, at, room int) []string
```

- What it draws and holds:
  - `Label`: a repository `📚 owner/repo` (the member's last two segments; with no member, the checkout's base name);
    a group `📚 <name> <count>` (`len(Members)`), the count dim; both name parts `style.Repo`. Kind `""` with a
    fallback: `📚 <fallback>`, as today's title. `none`, or `""` with no fallback: `style.Off("📚")`.
  - `Items`, top to bottom: when the workspace is a repository with groups, a line per group, glyph `📚`, name
    `<name> <members>`, then a rule; the favorites, glyph `⭐`, name a group's `<name> <members>` or a repository's
    owner/repo followed by ☁️ when it has no checkout, then a rule when there were any; then `🔎 Browse…`,
    `➕ New workspace…`, `🧰 Manage workspaces…`. With a refusal, the first line is kit's line, glyph `❌`, then a rule.
  - Enabled: a group line and a favorite, but not the workspace shown now, nor a repository with no checkout. Kit's
    line, Browse…, New workspace…, and Manage workspaces… are never enabled in this step (step 7 makes them act).
  - Picking gives the line's workspace as kit names it: the group's name, the favorite's `name`.
  - The dropdown hangs from the 📚 at column `at` on `Top`, as `Menu.Overlay` hangs from `App`; Key closes on esc.

- [ ] **Step 1: Write the failing tests** in `books_test.go`:
  - `TestBooksNamesARepositoryAGroupAndNone`: strip escapes; repository member `github.com/empowerite/kit` →
    `📚 empowerite/kit`; group `empowerite` of 5 members → `📚 empowerite 5`; kind `""`, fallback `empowerite/kit` →
    `📚 empowerite/kit`; none → `📚`. Assert `Label()` of the group holds `style.Off("5")` unstripped.
  - `TestBooksDropsDownTheGroupsThenTheFavorites`: a repository in groups `w` (2) and `v` (1), favorites a group `w`
    and a repository with no checkout. Assert the names, in order: `w 2`, `v 1`, a rule, `w 2`, `o/b ☁️`, a rule,
    `Browse…`, `New workspace…`, `Manage workspaces…`; enabled: only indexes 0, 1, 3.
  - `TestBooksPicksAGroupByClickAndByKey`: `Opened()` puts `At` on 0; a Key `down` then `enter` gives `"v"`; a click
    on the box's line for `w 2` gives `"w"` and closes it; a click on the dim `Browse…` gives `""` and keeps it open; a
    click off the box gives `""` and closes it.
  - `TestBooksSaysKitsLineWhenKitCouldNot`: `Told(WorkspaceMsg{Refusal: "no kit verb 'workspaces'"})` keeps the
    fallback label, and its first item is `❌ no kit verb 'workspaces'`, not enabled.
  - `TestBooksIsClickedOnItsCells`: with the title `shell.Title("kit welcome", false)`, `On` is true at
    `BooksAt(title)` and at the label's last cell, false a cell after it and on line 1.

- [ ] **Step 2: Run them to see them fail**

Run: `cd projects/tui && go test ./internal/shell -run Books -count=1`
Expected: FAIL, `undefined: BooksMenu`.

- [ ] **Step 3: Implement** `books.go`, with a package comment paragraph in the header style of `dropdown.go`.

- [ ] **Step 4: Run the package**

Run: `cd projects/tui && go test ./internal/shell -count=1`
Expected: PASS.

- [ ] **Step 5: Commit** `projects/tui/internal/shell/books.go projects/tui/internal/shell/books_test.go` as
  `#89 kittui: shell.BooksMenu, the 📚 control: it names the workspace and drops down its groups and the favorites`.

---

### Task 4: Welcome's title is the 📚 control

**Files:**

- Modify: `projects/tui/internal/welcome/welcome.go`, `projects/tui/internal/welcome/view.go`
- Modify: `projects/tui/internal/screens/screens.go`, `projects/tui/cmd/kittui/main.go` (the new `Open` signature)
- Test: `projects/tui/internal/welcome/welcome_test.go`

**Interfaces:**

- Consumes: `shell.BooksMenu`, `shell.BooksAt`, `shell.WorkspaceMsg`, `shell.Pick` (Task 3).
- Produces: `welcome.Model` gains `books shell.BooksMenu` (its `Fallback` the origin's owner/repo `New` is given) and
  `workspace kit.Workspace`; `Open(r kit.Runner, ws kit.Workspace) Model` (`ws` zero for unknown; `screens.open` and
  `cmd/kittui` pass the zero value until Task 5); `func (m Model) Books() shell.BooksMenu`.
  `title()` is `shell.Title(…) + " · " + m.books.Label()`; the `Repository` constant goes, `shell.Books` in its place.
- Clicks and keys: a left click on the control (`m.books.On(x, y, m.booksAt())`) opens it; while open it takes every
  key and every left click, as the app's menu does, and a pick returns `shell.Pick(workspace)`; the keys bar while open
  is `shell.Path{shell.MenuKeys()}`; it draws over the screen after the app's menu.

- [ ] **Step 1: Write the failing tests**: `TestTheTitleIsTheBooksControl` (the goldens are unchanged: run
  `go test ./internal/welcome -run Golden` and expect PASS before and after; plus a group workspace told by
  `WorkspaceMsg` draws `📚 w 2`), `TestAClickOnTheBooksDropsItDownAndAPickAsksKittui` (click at `booksAt()`, the keys
  bar is the menu's, a click on the first group's line returns a command whose message is `shell.PickMsg{"w"}`),
  `TestEscClosesTheBooks`.

- [ ] **Step 2: Run them to see them fail**

Run: `cd projects/tui && go test ./internal/welcome -run 'Books' -count=1`
Expected: FAIL.

- [ ] **Step 3: Implement.**

- [ ] **Step 4: Run the package, goldens included**

Run: `cd projects/tui && go test ./internal/welcome ./internal/screens ./cmd/... -count=1`
Expected: PASS, no golden changed (`git status --short projects/tui/internal/welcome/testdata` empty).

- [ ] **Step 5: Commit** the two source files and the test as `#89 kittui: Welcome's title is the 📚 control`.

---

### Task 5: the root loads the workspace, tells the screens, and switches

**Files:**

- Modify: `projects/tui/internal/screens/screens.go`
- Test: `projects/tui/internal/screens/screens_test.go`

**Interfaces:**

- Consumes: `kit.Runner.Workspace`, `kit.Runner.UseWorkspace` (Task 2); `shell.WorkspaceMsg`, `shell.PickMsg` (Task 3);
  `welcome.Open(r, ws)` (Task 4).
- Produces: `screens.Model` gains `home string` (the runner's first `Dir`), `workspace kit.Workspace`, and `asked int`
  (the latest load's number). Messages of its own: `workspaceMsg{gen int; ws kit.Workspace; err error}`.
- Behavior:
  - `Init` batches the shown screen's `Init` with the first load, `Workspace(ctx, "")`.
  - A `workspaceMsg` whose `gen` is not `asked` is dropped. An error: every screen made gets
    `WorkspaceMsg{Workspace: m.workspace, Refusal: <kit's line, as app.Refusal gives it, else err.Error()>}`.
  - Else, when the place changed, the screens made are remade: on the first load (gen 0), only when the workspace is a
    group; on a later one, when `Kind` or `Name` differ from `m.workspace`'s. Remaking closes every screen made but
    Tutorials and Help, drops them from `made`, and shows `here` again as `show` makes a screen. Then every screen made
    is told `WorkspaceMsg{Workspace: ws}`.
  - The runner's `Dir` follows the workspace: a repository with a checkout, its checkout; anything else, `home`.
  - `shell.PickMsg`: `asked++`, then a command that runs `UseWorkspace` when the pick is a group (no `/`, no `:`) and
    then `Workspace(ctx, pick)`, giving a `workspaceMsg` of that gen; `UseWorkspace`'s refusal is the message's err.
  - `show` tells a screen it makes the settings and the workspace before calling its `Init`, on the model those
    messages gave back.
  - `tag` passes `shell.PickMsg` through untagged, as it does `shell.SwitchMsg`.

- [ ] **Step 1: Write the failing tests** in `screens_test.go`, on the kit package's fake kit with `FAKE_WORKSPACES`
  pointed at a directory the test fills:
  - `TestTheWorkspaceReachesEveryScreenMade`: `current.json` a repository; start on Help, show Welcome; assert
    Welcome's title holds `📚 o/a` (from kit, not origin: the fake's directory has no origin).
  - `TestAKitWithoutWorkspacesLeavesEveryScreenOpen` (Review Focus 1): `FAKE_WORKSPACES` unset; Welcome opens, its
    last line is its keys bar (no refusal there), and its control's first item is kit's line.
  - `TestPickingAGroupRemembersItAndRemakesTheScreens`: Welcome shown, `PickMsg{"w"}`; assert the record holds `use w`,
    the Welcome made now is a new one (its `Init` ran again: the fake's welcome calls file has a second `--checks`), and
    Help, made before, is the same value.
  - `TestOnlyTheLatestPickLands` (Review Focus 4): deliver `workspaceMsg{gen: 1, …w…}` after `asked` is 2; assert
    `m.workspace.Name` is unchanged.
  - `TestTheFirstLoadInsideARepositoryRemakesNothing` (Review Focus 5): `current.json` a repository; after the first
    load, the welcome calls file holds one `--checks`.
  - `TestARepositoryPickedIsWhereLaterScreensRead`: pick `github.com/o/a` whose JSON has `checkout: <dir>`; assert
    `m.runner.Dir == dir`; then a group pick puts it back to `home`.

- [ ] **Step 2: Run them to see them fail**

Run: `cd projects/tui && go test ./internal/screens -run 'Workspace|Pick|Latest|FirstLoad|ARepository' -count=1`
Expected: FAIL (no workspace load; Welcome's title from origin).

- [ ] **Step 3: Implement** as Behavior says; extend the package comment with a sentence on the workspace.

- [ ] **Step 4: Run the package and the binary's tests**

Run: `cd projects/tui && go test ./internal/screens ./cmd/... -count=1`
Expected: PASS.

- [ ] **Step 5: Commit** `projects/tui/internal/screens/screens.go projects/tui/internal/screens/screens_test.go` as
  `#89 kittui: the root reads the workspace from kit, tells every screen, and a pick that moves it remakes them`.

---

### Task 6: Welcome over a group shows every member's checks together

**Files:**

- Modify: `projects/tui/internal/kit/kit.go` (`Check` gains `Member`, `Dir`, both `json:"-"`; `Key()`)
- Create: `projects/tui/internal/welcome/members.go` (`compose`, the group's read)
- Modify: `projects/tui/internal/welcome/welcome.go`, `view.go` (keys by `Key()`, the member on a row, the uncloned)
- Modify: `projects/tui/internal/welcome/testdata/fake-kit` (prints `./checks.json` from its working directory when one
  is there, the fixture otherwise; `--fix` records its working directory with the call)
- Test: `projects/tui/internal/welcome/members_test.go`

**Interfaces:**

- Produces:

```go
// in kit: the check's member, host/path, "" for one said once or over one repository; and the checkout its fix runs in.
func (c Check) Key() string // c.ID alone without a member, else member + " " + id

// in welcome:
type run struct {
    member, dir string
    checks      []kit.Check
    err         error
}
// compose gives the checks of every member's run as one list, in the order each id first appears: a check every
// member that ran it reports alike (OK, Line, Fix the same), once, with no member and the first such member's dir; one
// that differs, once per member, in the members' order, each with its member and dir. A run kit refused gives one
// failed check of its own: id "kit", its member, kit's line, no fix.
func compose(runs []run) []kit.Check
```

- Behavior:
  - `Open`/`New` take the workspace; over a group, the `checking` load's read runs `WelcomeChecks` in every member
    with a checkout, at once (one goroutine each, results in members' order), and composes them; over anything else
    it is today's read.
  - `uncloned`: the group's members with no checkout, by owner/repo. The intro, after its summary line, adds
    `☁️ not cloned: o/b, o/c`, dim, when there are any; a step's checks are unchanged by them.
  - Every map and lookup now keyed by a check's id (`deferred`, `notes`, `places`, `find`, `cursorID`, `fixedMsg.id`)
    uses `Key()`; with no member, `Key()` is the id, so a single repository behaves as now.
  - A row of a check with a member ends with a dim `·` and the member's last segment, before `· later`.
  - The fix runs `WelcomeFix(c.ID)` on a runner whose `Dir` is `c.Dir` when it is set.

- [ ] **Step 1: Write the failing tests** in `members_test.go`:
  - `TestComposeSaysACheckAlikeInEveryMemberOnce`: runs a, b each `{jq ok=false "jq is not installed"}` →
    one check, `Member ""`, `Dir` a's.
  - `TestComposeShowsACheckThatDiffersPerMember`: a `{toolchain-project ok}`, b `{toolchain-project fail}` → two
    checks, members a then b, at the place the id first appeared.
  - `TestComposeTurnsARefusalIntoItsMembersFailedCheck` (Review Focus 2): b refused → a's checks and `{id:"kit",
    member:b, ok:false, line:kit's line}`.
  - `TestAGroupsChecksDrawTogetherWithTheUnclonedNamed`: a group of two checkouts (temp dirs, one with its own
    `checks.json` where `toolchain-project` passes) and one member with no checkout; the intro holds
    `☁️ not cloned: o/c`; the toolchains step shows `toolchain-project` twice, each row ending `· a` / `· b`.
  - `TestAFixRunsInItsMembersCheckout` (Review Focus 3): on b's failing `toolchain-project`, `n`; the exec stub saw
    `cmd.Dir == b's dir`; deferring a's same id with `l` leaves b's row without `· later`.
  - `TestOverOneRepositoryTheChecksAreAsBefore`: the goldens pass unchanged (run them in Step 4).

- [ ] **Step 2: Run them to see them fail**

Run: `cd projects/tui && go test ./internal/welcome -run 'Compose|Group|Member|Fix' -count=1`
Expected: FAIL, `undefined: compose`.

- [ ] **Step 3: Implement.**

- [ ] **Step 4: Run kittui's whole suite**

Run: `cd projects/tui && go test ./... -count=1`
Expected: PASS, no golden changed.

- [ ] **Step 5: Commit** the files above as `#89 kittui: over a group, Welcome shows every member's checks together,
  a check alike in every member once, and a fix runs in its member's checkout`.

---

### Task 7: Flow's title is the 📚 control (waits for kit-e7)

**Not started until kit-e7 says `internal/flow` is free.** Then, in `internal/flow` (`view.go`'s `title`, the click
and key routing beside the app's menu, a `WorkspaceMsg` case): Flow draws `m.books.Label()` in place of
`Repository + style.Repo(m.origin.Repo)`, `Fallback` the origin's owner/repo, opens and picks as Welcome does (Task 4's
tests, ported). Over a group, Flow's rows stay as they are today until step 3 (`kit flow --workspace`). Commit as
`#89 kittui: Flow's title is the 📚 control`. If kit-e7 would rather fold it into its own move, send it this task.

---

## Finish

`kit gate` on a clean copy of HEAD (lint rules, bats, Go, the coverage floor). Then a whole-branch review of the
step's commits, and `kit commit` only.
