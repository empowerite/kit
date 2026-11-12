# The Apps' Tabs Read kit Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or
> superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** A kittui tab is one catalog entry naming the kit command that prints its rows, so Platform's toolchains and
tools and Trust's policy draw kit's data, a tab or an app with nothing says so honestly, and what #138 left of the
one Flow picture is gone.

**Architecture:** A stage's entry in `fields.json` gains its command, the words the command takes, and how its output
reads: `record`, `key`, and each field's `path`, which `internal/table` now evaluates, into fresh values.
`internal/flow` reads every stage that one way, with one registry for the seven stages whose rows Go still builds
and no switch by a stage's name. kittui's root keeps kit flow's words and gives them to every app it opens.

**Tech Stack:** Go 1.27 (Bubble Tea v2), bash, jq, bats.

**Spec:** `../specs/2026-11-09-entities-design.md` (§ The domains, § Trust, § Tools),
`../specs/2026-11-07-kittui-components-design.md` § Table, and issues #138 and #123.

Today 20 of 32 tabs are planned (`internal/flow/apps.go:26-35`), and `Schema.Record` and `Field.Path` are never read
(Apps 17, verified). Paths start in `projects/tui/` unless they start with `kit/`, `tests/`, or `docs/`.

**Review:** [the findings](../reviews/2026-11-10-code-review-findings.md) of the review of 2026-11-10, cited
here by area and number: Serve 8 is item 8 under its Serve heading.

## Decisions for the owner

1. The word that opens an app: `kit tui --app <app> [<tab>]`, or seven verbs, `kit issues` to `kit platform`. Written
   on `--app`: `kit trust` is #66's remedy, `kit platform` the entities spec's listing, and kittui's half is the same.
2. An app none of whose tabs reads kit, People after this plan: left out of the app menu and its word refused, or
   kept, opening on planned tabs. Written on leaving it out, as "never for this item" is omitted.
3. The flag that lists a policy's rules: `kit policy --rules [--json | --jq]`, or `kit policy --json`. Written on
   `--rules`: `--json` alone reads as the verdict, which the entities spec gives to `kit ci --json`.
4. kit flow's `desk` and `worktrees`, which no app draws since #138: deleted from kittui, or re-homed as a tab of
   Repositories. Written on deleting: `kit flow desk` already opens kit workspace's worktrees (`apps.go:53`).

## Global Constraints

- kit commit only, never kit submit or push; every subject starts with `#138`, as `#138 kittui: …` or `#138 kit: …`.
- `export PATH=$PWD/kit/bin:$PATH` before `kit commit`.
- Peers share the worktree: stage named paths only, guard that the index holds only them, never leave anything staged.
  A Commit step is `git add <its paths>`, `git diff --cached --name-only` showing only those, then `kit commit
  --message "<subject>" --message "Co-Authored-By: Claude <noreply@anthropic.com>"`.
- Everything under `kit/` is written for a consumer, and a path in it is written from `kit/`: `modules/bash/x.sh`.
- Lines are 120 cells at most.
- A verb's header comment is its help page and its usage line is its grammar (`kit syntax --check` holds them): a new
  flag is added to the header's usage line and described there, and `docs/` says the same. `tests/syntax.bats` then
  prints the command that refreshes `internal/syntax/testdata/<verb>.json`.
- kittui: every action by a plain left click and by a key; a menu line's shortcut at its right; no circled glyphs;
  two-cell emoji in menus; esc only cancels or goes back; a line that never applies is omitted, a not-now one is dim.
- Go tests run without `-update`. A step that regenerates goldens runs `go test ./internal/flow -update` on a tree
  with no peer work in progress under `projects/tui/`, and reviews the diff of `internal/flow/testdata`.
- Coverage, after every task, in `projects/tui/`: `go test -coverprofile=$TMPDIR/c ./... && go tool cover
  -func=$TMPDIR/c | tail -1`; `bin/ci:67` fails under 98.0. At `2dad3552` it is 8,512 of 8,689 statements, 97.96%,
  flow 98.5%. The 61 Task 2 deletes are all covered, which alone leaves 8,451 of 8,628, printed 97.9: deleting
  covered code lowers it, so each task covers what it adds, and Task 2 two statements more.

## Review Focus

1. A reload whose answer holds `null` where the row before held a value, and a fresh answer over the kept copy: the
   row shows nothing there (Apps 1). Task 3.
2. Empty output, `null`, or `{"rows": null}` with exit 0 is a refusal naming the command; `[]`, and no line from a
   command read a line at a time, are the empty stage (Apps 22). Tasks 1 and 3.
3. `kit tools --json` refusing in five lines, as with no toolchain chosen: tools shows them, toolchains draws. Task 6.
4. Two rows that share a name, as brew is a system and a project toolchain: their keys differ. Task 5.
5. `kit flow --mine`, another app, then another workspace: each app made later takes the same words. Task 4.

---

## Task 1: A schema names its command, and table reads its records

**Files:** Modify `internal/table/catalog.go:14-19,105-119`. Create `internal/table/read.go`, `read_test.go`.

**Interfaces:** Produces, in package `table`:

| Name | Contract |
| --- | --- |
| `Schema`'s new keys | `Command []string` (`command`, kit's words: `["toolchains", "--json"]`); `Takes []string` (`takes`, of `owner`, `no-fetch`, `fresh`, `cached`, `workspace`); `Staged bool` (`staged`, a stage its verb's `--stages` and `--marks` name); `Slurp bool` (`slurp`, a document a line, read as `jq --slurp` reads them); `Key string` (`key`); `Empty string` (`empty`, the line an empty table shows). |
| `func Records(data []byte, record string, slurp bool) ([]json.RawMessage, error)` | Fresh each call. `record` is `.` (the document, an object), `.[]` (each element of the document, an array), or `.<key>[]` (each element of the array the object holds under `<key>`). |
| `type ShapeError struct{ Printed, Wanted string }` | What `Records` gives for another shape: `printed <Printed> where <Wanted> was expected`, `Wanted` being `an object`, `an array`, or `an object with "<key>"`. |
| `func (s Schema) KeyOf(record json.RawMessage) (key string, ok bool)` | The paths of `s.Key`, or `.key`; `ok` when each is present and not null; the key is each scalar's text, a space between. |
| `func (f Field) Read(record json.RawMessage) Value` | `f.Path` is one path or several after commas, each `.a.b`, a `[]` after a key allowed and ignored. An array gives a value an element, and a path through one maps over it; a string is itself, a number its digits, `true` the field's `ID`, and `false`, `null`, or an object no value. |

- [ ] **Step 1: Write the failing tests** in `read_test.go`: the two below, and (Components 15)
  `TestTheCatalogRefusesWhatATableCannotIndexBy`, in which `ReadCatalog`'s error holds `declared twice`, `no preset
  named Default` (for fields without one), `has no rendering`, `record "..x"`, `takes "bogus"`, and `names no stage`.

```go
// TestRecordsReadsEachShapeFreshAndRefusesTheRest: data, record, slurp, how many records, ShapeError.Printed
{`[{"n":1},{"n":2}]`, ".[]", false, 2, ""}, {`[]`, ".[]", false, 0, ""}, {`{"n":1}`, ".", false, 1, ""},
{`{"rows":[{"n":1}],"errors":[]}`, ".rows[]", false, 1, ""}, {"{\"n\":1}\n{\"n\":2}\n", ".[]", true, 2, ""},
{"", ".[]", true, 0, ""}, {"", ".[]", false, 0, "nothing"}, {"null", ".", false, 0, "null"},
{`{"n":1}`, ".[]", false, 0, "an object"}, {`[]`, ".", false, 0, "an array"},
{`{"errors":[]}`, ".rows[]", false, 0, `an object without "rows"`}, {`{"r":null}`, ".r[]", false, 0, `null "r"`},
// TestAFieldReadsItsPath: Field{ID: "id", Path: path}.Read(rec).Raw for each path, then KeyOf
rec := json.RawMessage(`{"a":{"b":"x"},"n":3,"t":true,"f":false,"z":null,"l":[{"name":"p"},{"name":"q"}],"u":[1,2]}`)
map[string][]string{".a.b": {"x"}, ".n": {"3"}, ".t": {"id"}, ".f": nil, ".z": nil, ".nope": nil, ".a": nil,
  ".l[].name": {"p", "q"}, ".u": {"1", "2"}, ".u[]": {"1", "2"}, ".n, .a.b": {"3", "x"}}
// Schema{Key: ".n, .a.b"}.KeyOf(rec) is "3 x", true; Schema{}.KeyOf(`{"key":"v"}`) is "v", true; Key ".z" is "", false
```

- [ ] **Step 2: Run** `go test ./internal/table -run 'TestRecords|TestAFieldReads|TestTheCatalogRefuses' -count=1`.
  Expected: it does not compile, `undefined: Records`.
- [ ] **Step 3: Implement**; `Records` reads with a `json.Decoder`; `staged` wants a command of three words or more.
- [ ] **Step 4: Run** the same, then `go test ./internal/table ./internal/flow -count=1`. Expected: pass.
- [ ] **Step 5: Commit** `internal/table` as `#138 kittui: a table's schema names the command that prints it, and
  table reads its records and each field's path, fresh each time`.

## Task 2: What #138 left of the one picture goes

**Files:** Modify `internal/flow/{flow.go,rows.go,fields.go,fields.json,apps.go,view.go}`, `testdata/fake-kit`, and
the tests below. Delete `testdata/{desk,worktrees}.json` and their four goldens.

Each item of Apps 14, decided; `mine()` is Task 8's:

| What #138 left | Decision |
| --- | --- |
| `desk` and `worktrees` | Deleted from kittui: their catalog entries; their names in `known` (`flow.go:88`), `expected` (`:1138`), `decodeRows` (`:1063`), `records` (`rows.go:454`), and `empty` (`rows.go:981`); `stage.desks`, `stage.trees`, and the type `worktrees`; `deskRows`, `worktreeRows`, `checkoutRow`, `pullOf`, `diffstat`, and `uncommitted` (`rows.go:742-863`); `uncommittedCells` and the renderers `diffstat` and `uncommitted` (`fields.go:120`). `checkout` and its other renderers stay, as repos draws with them; so does `glyph.go`'s generated region, as kit's glyph table keeps both stages for `kit flow --text`. |
| `moved`, `reloads` (`flow.go:957-969`) | Deleted, with the branch of `update` that calls `reloads`. Repositories' tabs go stale by `kit workspace --marks`, which reads the refs a fetch moves. |
| `here` asking the desk stage (`rows.go:613`) | It asks a new `deskOf(member string) string`: `checkoutOf(member)`, and `m.ws.Checkout` for a row of no member. A path equal to it is `desk`; either way the link is to `workspace/worktrees`. |
| `Options.App == ""` | Gone: `appLabel` (`view.go:27`) and `reaches` (`apps.go:117`) lose its branch, and the tests' `options()` names `shell.Issues`. |
| Tests of what is deleted | These go: `TestTheDeskIsOneRowThatOpensToItsPathsThenItsCommits`, `TestWorktreesShowTheirPullRequestsAndTheirUncommittedCountButNotTheDesk`, `TestAnOpenCheckoutFillsThePaneAndPagesThroughEveryCommit`, `TestWithoutIncomingEveryStageLoadsAtOnce`, `TestTheFetchReloadsTheStagesItMovesKeepingTheirRows`, `TestOverAGroupTheDeskIsAPerMemberRow`, `TestOverAGroupAMembersGoneBranchesSayWhose`, `TestRefreshingIncomingReloadsTheStagesTheFetchMoves`. |
| Tests that used a checkout's tall open row | These move to builds, whose open row lists its jobs, answered with `answer("builds", tallBuild(n))`, a new helper giving one run of n jobs: `TestThePaneScrollsToKeepTheCursorWhileTheRestStays`, `TestPageDownScrollsThroughAnOpenRowTallerThanThePaneBeforeMovingOn`, `TestAClickOnARowSelectsItAndAgainOpensOrClosesIt`, `TestPageUpInsideATallOpenRowScrollsBackThroughIt`, `TestTheWheelScrollsTheRowsAndLeavesTheCursor`, `TestPagingThroughAWorktreeOfSevenHundredCommitsIsWellUnderAFramesTime`. |
| The other tests | `eight` (`flow_test.go:32`) becomes `six`; a test that counted tabs to a stage presses `slices.Index(six, name)` of them; one that named `desk` or `worktrees` as any stage that reads git names `incoming` (`grep -n 'desk\|worktrees' internal/flow/*_test.go \| grep -v workspace` lists them). |

- [ ] **Step 1: Write the failing tests.** `TestTheDeskIsHereInIssues` (`rows_test.go`, new): Issues over backlog
  and underway, told `shell.WorkspaceMsg{Workspace: kit.Workspace{Kind: "repository", Checkout: p}}`, p being
  `/home/kit/worktrees/1-the-first-release-v0-1-0`; the record `#1 #2` has `values["where"].Of == "desk"` and
  `Link == jumpTo("workspace/worktrees", p)`. `TestAnAnswerForAStageTheScreenDoesNotHaveIsDropped`: an answer named
  `nonesuch`, and the same with `Kept: true`, leave `view(m)` as it was; they cover `flow.go:976` and `kept.go:33`.
- [ ] **Step 2: Run** `go test ./internal/flow -run 'DeskIsHere|AnAnswerFor' -count=1`. Expected: `Of` is `here`.
- [ ] **Step 3: Delete and re-point** as decided.
- [ ] **Step 4: Regenerate the goldens**, as Global Constraints say: the bar has six tabs. Run `go test
  ./internal/flow -count=1` and the coverage command. Expected: pass, 98.0 or more.
- [ ] **Step 5: Commit** `internal/flow` as `#138 kittui: the desk and worktrees stages, the fetch's reloads, and the
  picture without an app go, and the desk's checkout draws 💻 here again`.

## Task 3: Every stage is read as its catalog entry says

**Files:** Modify `internal/flow/{fields.json,flow.go,rows.go,workspace.go,kept.go,apps.go,repos.go,view.go}`,
`internal/kit/kit.go:306`, and the tests below.

**Interfaces:** Consumes Task 1's. Produces the entries below, every stage `staged`, kit workspace's other five
taking `workspace`, reading `.rows[]`, and saying no `key` and no `empty`; and, in Go, what the second table names:

| stage | command | takes | record | key | empty |
| --- | --- | --- | --- | --- | --- |
| `backlog` | `flow backlog --json` | owner fresh cached workspace | `.[]` | `.number` | no open issue without a pull request |
| `underway` | `flow underway --json` | owner fresh cached workspace | `.[]` | `.number, .pr.number` | no issue is underway |
| `changes` | `flow changes --json` | owner fresh cached workspace | `.[]` | `.number` | no open pull request |
| `builds` | `flow builds --json` | cached workspace | `.[]` | `.id` | no build since the last 20 runs |
| `releases` | `flow releases --json` | cached workspace | `.[]` | `.version` | no release |
| `incoming`, new, no fields | `flow incoming --json` | no-fetch cached workspace | `.` | `.remotes` | no remote to fetch |
| `workspace/repos` | `workspace repos --json` | workspace | `.rows[]` | `.repository` | no repository |

| Goes | In its place |
| --- | --- |
| `stage.issues`, `under`, `ins`, `pulls`, `runs`, `rels`, `repos` | `stage.rows []json.RawMessage` and `stage.typed any`; `func of[T any](s stage) []T` gives `typed` as the slice it is. |
| `decodeRows`, `across`, `expected`, `shape`, `lacking` | `func (s *stage) decode(data []byte, over bool) error` alone: `Records` with the entry's record, or `.rows[]` over a group for a stage that takes `workspace`; a `ShapeError` worded `<command(name)> <error>`; a record `KeyOf` refuses, `<command(name)> printed a record without <key>`; then `s.rows`, `s.failed`, and `s.typed`, each replaced whole. |
| the switch in `records` | `var builders map[string]builder` in `rows.go`, else `catalogRows`: `type builder struct{ decode func([]json.RawMessage) (any, error); rows func(Model, stage) []record }`, and `func decoding[T any](records []json.RawMessage) (any, error)`, each record into a fresh `T`. It holds the seven above; incoming's `rows` is nil, as `build` and `autoGroup` still call `incomingRows`, and its entry, without fields, has no presets, which `New` (`flow.go:251`) and `builtins` (`views.go:123`) allow for. |
| `workspaceRows`, `rawValue` | `func (m Model) catalogRows(s stage) []record` in `workspace.go`: the key is `m.keyOf(member, key)` from `KeyOf`, the member `.member`, a link `.links.<id>`, and every other field `Read`. |
| `known`, `shared`, the switches in `args` and `empty`, `command`'s two forms | `func sourced(name string) bool` (the entry has a command), `staged(name)`, `takes(name, word)`, the entry's `Empty`, else `nothing`, and `kit` with the entry's words. Incoming's not-fetched line goes: a repository without remotes has none to fetch either way. |
| `Status`'s verb from the stage's name (`kit.go:306`) | `func (r Runner) Rows(ctx context.Context, words []string, over bool) ([]byte, error)`: `kit <words…>`, then `--workspace <Group>` when `over` and the runner spans a group. `Status` calls it; `OpenApp` and `verbsOf` ask `Command[0]` for `--stages` and `--marks`; the quota is read after a load of a `flow` command. |
| The old refusals' and empty lines' words, in tests | Re-worded: `TestADocumentOfAnotherShapeIsOneRefusalLine`, `TestBuildsRefuseADocumentWithoutIDs`, `TestUnderwayRefusesADocumentWithoutPullRequests`, `TestTheTitleCarriesOriginAndTheFetchIsTheIncomingStage`, `TestADocumentKitFlowCannotDecodeIsARefusal`, `TestEveryEmptyStageSaysWhatItLacks`, the refusal at `group_test.go:121`, `TestTheCatalogNamesOnlyWhatIsRegistered`, and `TestEveryStageHasAMemberField`, which holds for a stage that takes `workspace`. |

- [ ] **Step 1: Write the failing tests** in `flow_test.go`:

```go
// TestAReloadKeepsNoFieldOfTheRowThatStoodThere, over Stages: []string{"backlog"}
m := drive(New(runner(t), o), answer("backlog",
  `[{"number":5,"title":"five","branch":"5-five","source":"github"},{"number":7,"title":"seven","branch":null}]`))
m.restart("backlog")
m = drive(m, source.Answer[[]byte]{Name: "backlog", Gen: m.find("backlog").src.Gen(),
  Value: []byte(`[{"number":7,"title":"seven","branch":null,"source":null}]`)})
got := m.records(*m.find("backlog")) // one, its Key "#7", values["branch"].Raw and values["source"].Raw empty
// TestAFreshAnswerKeepsNoFieldOfTheKeptCopy: the same, the first answer with Kept: true and no restart
// TestANullOrEmptyDocumentIsARefusal: stage{name: name}.decode(doc, false) is an error starting with
// command(name)+" printed " for each of these, and for backlog's "[]" no error and no rows
{"backlog": "null", "changes": "", "incoming": "null", "workspace/tags": `{"rows":null}`, "builds": `[{"number":3}]`}
```

- [ ] **Step 2: Run** `go test ./internal/flow -run 'TestAReloadKeeps|TestANullOrEmpty|TestAFreshAnswer' -count=1`.
  Expected: `#7` shows `5-five`; backlog's `null` decodes with no error.
- [ ] **Step 3: Implement.** A test that poked a typed slice answers the stage instead: `drive(m, answer(name, doc))`.
- [ ] **Step 4: Re-word the tests that pin the old lines**, which the table's last row names. No golden changes.
- [ ] **Step 5: Run** `go test ./internal/flow ./internal/kit ./internal/screens -count=1` and the coverage command.
- [ ] **Step 6: Commit** `internal/flow internal/kit` as `#138 kittui: a stage is read as its catalog entry says,
  into fresh rows, and null or nothing is a refusal`.

## Task 4: kit flow's words reach every app

**Files:** Modify `internal/flow/{flow.go:191-205,apps.go:76}`, `internal/screens/screens.go:309-313`,
`cmd/kittui/main.go:83-87`, and their tests.

**Interfaces:** Produces, in `flow`: `type Words struct{ Tab string; NoFetch, Fresh bool; Owner []string }`;
`func ParseWords(words []string) Words`, taking `--no-fetch`, `--fresh`, the owner's flags (`--all`, `--mine`,
`--others`, `--owner=<login>`), the first other word as the tab, and no other word (Apps 16, kittui's
half); `func (w Words) Rest() Words`, without the tab; `OpenApp(r kit.Runner, app shell.Screen, w Words) (Model,
error)`; `func (m Model) Words() Words`. In `screens`: `func (m Model) Taking(w flow.Words) Model`, which `main`
calls with `Rest()`, and `open` gives every app it makes. `Fresh` makes each stage's first read fresh.

- [ ] **Step 1: Write the failing tests.** `TestTheWordsReachEachCommandThatTakesThem` (`apps_test.go`): under
  `FAKE_FLOW_CALLS`, Repositories opened with `Words{NoFetch: true, Owner: []string{"--mine"}}` and loaded calls
  `flow incoming --json --no-fetch`, and no `workspace` call holds `--mine`; Changes, with `Fresh: true` too, calls
  `flow changes --json --mine --fresh`; `ParseWords(strings.Fields("underway --mine --owner=bob --jq"))` is
  `Words{Tab: "underway", Owner: []string{"--mine", "--owner=bob"}}`. `TestKitFlowsWordsReachAnAppOpenedLater`
  (`screens_test.go`): after `help(t).Taking(w)` and a switch to Repositories, and again after a workspace pick, as
  `TestPickingAGroupRemembersItAndRemakesTheScreens` picks, `m.made[shell.Repositories].(flow.Model).Words()` is `w`.
- [ ] **Steps 2 to 4:** run them (`undefined: Words`); implement, `nil` becoming `flow.Words{}` at `OpenApp`'s other
  callers; run `go test ./internal/flow ./internal/screens ./cmd/kittui -count=1`.
- [ ] **Step 5: Commit** as `#138 kittui: kit flow's words reach every app, each command taking those its catalog
  entry names`.

## Task 5: Platform's toolchains tab

**Files:** Modify `internal/flow/{fields.json,apps.go:34}` and `testdata/fake-kit`. Create `platform_test.go`,
`testdata/platform-toolchains.json`, and `testdata/platform-toolchains.golden`.

The three new entries, Tasks 6's and 7's with this one's. Each is `slurp` with record `.[]`, as its command prints an
object a line; takes nothing; is not staged; has presets Default and Compact; and renders every field as `text`:

| stage | command | key | empty | fields: id (type, path where not `.<id>`; what it can do) | Compact hides |
| --- | --- | --- | --- | --- | --- |
| `platform/toolchains` | `toolchains --json` | `.provider, .role` | no toolchain | `toolchain` (text, `.provider`; sorts), `role` (enum of project, system; sorts, groups, filters), `usable`, `active`, `chosen`, `fills` (each an enum of its own id, from the boolean; groups, filters; dim; fills hidden) | active |
| `platform/tools` | `tools --json` | `.from, .name` | no pinned tool | `tool` (text, `.name`; sorts), `pinned`, `installed` (text; dim), `from` (enum; sorts, groups, filters), `path` (text; hidden) | from |
| `trust/policy` | `policy --rules --json` | `.rule` | no rule | `rule` (text), `pattern` (text; wraps), `from` (enum of default, kit.yml; groups, filters), `engine` (enum; groups) | engine |

`apps` names the stage where it named `planned/<tab>`. The fixture is the nine lines kit prints in this repository,
as `{"provider":"brew","role":"system","usable":true,"active":false,"chosen":true,"fills":false}`. fake-kit answers a
command by its words before it looks for `flow`: the fixture, or a refusal, exit 1, under `FAKE_FLOW_REFUSE=<stage>`.

- [ ] **Step 1: Write the failing test** `TestPlatformsToolchainsTabDrawsKitsRows`: Platform, opened with
  `Words{Tab: "toolchains"}` and loaded at 100×20, calls `toolchains --json` and no `--stages`; `valueOf(t, m,
  "platform/toolchains", "brew system", "chosen")` is `chosen`, `brew project`'s is empty, and the count is `9`.
- [ ] **Steps 2 to 4:** run it (the tab says it has no data source yet); add the entry, the fixture, and fake-kit's
  case; pin `raw(m)` with `testkit.Golden`, written with `-update` and read; run `go test ./internal/flow -count=1`.
- [ ] **Step 5: Commit** as `#138 kittui: Platform's toolchains tab reads kit toolchains`.

## Task 6: Platform's tools tab

**Files:** as Task 5's, with `testdata/platform-tools.json` and its golden, and `apps.go:34`'s `planned/tools`.

- [ ] **Step 1: Write the failing tests**: `TestPlatformsToolsTabDrawsKitsRows`, as Task 5's, over a fixture of
  lines as `{"name":"bash","pinned":null,"installed":"5.3.20","from":"brew"}`, whose `pinned` is empty at key
  `brew bash`; and `TestAToolsRefusalLeavesToolchainsDrawn`: under `FAKE_FLOW_REFUSE=platform/tools`, which prints
  kit's five lines from `❌ several toolchains are usable` to `KIT_TOOLCHAIN=<name>`, tools' count is `!`, `view(m)`
  holds `kit toolchains choose mise`, and toolchains' count is `9`.
- [ ] **Steps 2 to 5:** as Task 5's; commit as `#138 kittui: Platform's tools tab reads kit tools`.

## Task 7: `kit policy --rules`, and Trust's policy tab

**Files:** Modify `kit/bin/kit-policy:4,17-18,41`, `kit/modules/bash/policy.sh:68`, `tests/policy.bats`,
`docs/policy.md`, `internal/syntax/testdata/policy.json`; then as Task 5's, with `testdata/trust-policy.json`.

**Interfaces:** Produces `kit policy [--input | --facts=<file> | --rules [--json | --jq]]`. `policy:listed` prints
`<rule>\t<pattern>\t<from>` for `branch`, `commit_subject`, and `change_title`, in that order, `<from>` being
`kit.yml` where the `policy:` map sets the rule, else `default`. `--rules` prints them as a table, and with `--json`
an object a line, `{rule, pattern, from, engine}`, `engine` as `policy:engine` says. It reads no facts and no forge.

- [ ] **Step 1: Write the failing test** in `tests/policy.bats`:

```bash
@test "kit policy --rules --json lists each rule, its pattern, where it is set, and the engine" {
  printf "policy:\n  branch: '^feature/[a-z-]+\$'\n  change_title: ''\n" > kit.yml
  run "${kit_root}/bin/kit" policy --rules --json
  [[ "$status" -eq 0 && "${#lines[@]}" -eq 3 && "$(jq --raw-output .from <<<"${lines[1]}")" == default ]]
  [ "${lines[0]}" = '{"rule":"branch","pattern":"^feature/[a-z-]+$","from":"kit.yml","engine":"patterns"}' ]
  [ "${lines[2]}" = '{"rule":"change_title","pattern":"","from":"kit.yml","engine":"patterns"}' ]
}
```

- [ ] **Step 2: Run** `KIT_TOOLCHAIN=mise kit/bin/kit tools run bats tests/policy.bats`. Expected: the new test
  fails, as `kit policy` refuses `--rules` with 2.
- [ ] **Step 3: Implement** `policy:listed`, and the branch in `kit-policy` before `--input`'s; describe `--rules`,
  `--json`, and `--jq` in the header and in `docs/policy.md`; refresh the syntax fixture.
- [ ] **Step 4: Run** `tests/policy.bats`, `tests/syntax.bats`, and `tests/help.bats`. Expected: pass.
- [ ] **Step 5: Wire the tab** as Task 5's; `TestTrustsPolicyTabDrawsKitsRules` finds 3 rows, `branch` from `kit.yml`.
- [ ] **Step 6: Commit** the kit paths as `#138 kit: kit policy --rules lists each rule, its pattern, and where it
  is set`, then `internal/flow` as `#138 kittui: Trust's policy tab reads kit policy --rules`.

## Task 8: Add to Pull Request… asks kit for the caller's pull requests

**Files:** Modify `internal/flow/act.go:152-189,264-271`, `flow.go` (beside `case trackersMsg`), and `act_test.go`.

`mine()` read the `changes` stage, which Issues has not had since #138, so the line was never enabled
(Apps 7). It goes. Choosing the line reads `kit flow changes --json --mine` where the issues' member is
checked out, as New… reads the trackers, into `type pullsMsg struct{ pulls []pull; err error }`, by `Runner.Rows` and
`decoding[pull]`; `acting` enables it whenever issues of one member are selected. `pullsMsg` opens the ask as before;
with none the notice is `❌ you have no open pull request to add to`, and kit's line when the read fails.

- [ ] **Step 1: Re-write** `TestAddToPullRequestPicksOneOfTheCallersOwn` over Issues' two stages: a backlog row
  selected, the line is enabled; chosen, the calls hold `flow changes --json --mine`, and the ask lists each of
  `testdata/changes.json`'s by 🔀, number, and title; under `FAKE_FLOW_REFUSE=changes` the notice is kit's line.
- [ ] **Steps 2 to 4:** run it (the line is dim), implement, and run `go test ./internal/flow -count=1`.
- [ ] **Step 5: Commit** as `#138 kittui: Add to Pull Request… reads the caller's pull requests from kit when chosen`.

## Task 9: A tab or an app with nothing says so, and offers only what can apply

**Files:** Modify `internal/flow/{view.go:423,menu.go:47-58,121-142,watch.go:36-61,host.go:44,legend.go:214,apps.go}`,
`internal/shell/shell.go:132-143`, `apps_test.go`, `menu_test.go`, and `internal/shell/shell_test.go`.

| Rule | Its test, new in `apps_test.go` but the last, `shell`'s |
| --- | --- |
| A planned tab's line, dim, is `no kit command lists <tab> yet`. Its menu holds Maximize or Restore alone: Refresh, Watch, the folds, the selection, and Open Selected can never apply (Apps 19). `m` and a click on the tab work as on any tab. | `TestAPlannedTabSaysWhatItWaitsForAndItsMenuHoldsOnlyTheSize`: on Trust's first tab `view(m)` holds `no kit command lists evidence yet`, and `m.tabItems()` is Maximize alone. |
| A tab whose command is not staged has no mark and is never stale: its menu omits Watch, and `watching` counts staged stages alone. The app's menu holds Refresh All where a tab is sourced, and its two Watch lines where one is staged. | `TestATabThatIsNeverStaleOffersNoWatch`: `platform/toolchains` has no Watch, and Platform's `ownItems()` is Refresh All alone. |
| `Name()` and the legend's title give `label(name)`: `policy legend`, not `trust/policy legend` (Apps 20). | `TestDialogsNameTheTab`: on Trust's policy, `m.Name() == "policy"`. |
| `func Hide(s Screen)` in `shell`: `Menu.Items` passes a hidden screen by. `flow`'s `init` hides each app none of whose stages is sourced, and `func Shown(app shell.Screen) bool` says which are not. | `TestAnAppWithNothingIsLeftOutOfTheMenu`: `shell.Menu{}.Items()` names Trust and Platform and no People, and `Shown(shell.People)` is false. `TestAHiddenScreenIsNotInTheMenu`, which shows the screen again in its cleanup. |

- [ ] **Step 1: Write the failing tests** of the table.
- [ ] **Steps 2 to 4:** run them, implement, update `TestAPlannedTabLoadsNothingAndSaysSo` and the tests that find
  People in the menu, and run `go test ./internal/flow ./internal/shell ./internal/screens -count=1`.
- [ ] **Step 5: Commit** as `#138 kittui: a planned tab says what it waits for, a menu leaves out what can never
  apply, and an app with nothing to show is left out of the app's menu`.

## Task 10: `kit tui --app <app> [<tab>]`, and the README

**Files:** Modify `kit/bin/kit-tui:4-11`, `tests/tui.bats`, `internal/syntax/testdata/tui.json`, `docs/features.md`
(§ The terminal UI), `cmd/kittui/main.go:83-98`, `main_test.go`, `internal/flow/apps.go`, and `README.md:6-33`.

**Interfaces:** kit-tui's second usage line is `kit tui --app <app> [<tab>]`, `<app>` described as issues, changes,
repositories, delivery, trust, or platform; kit passes the words on. In `flow`: `func AppNamed(name string)
(shell.Screen, bool)`, an app that is `Shown`. In `main.go`, `open`'s `case "--app"` is `OpenApp(r, app,
flow.Words{Tab: tab})`; another name, or none, is `&kit.ExitError{Code: 2, Stderr: "kittui has no app '<name>'"}`.

- [ ] **Step 1: Write the failing tests.** `tests/tui.bats`: `run terminal "'${kit_root}/bin/kit' tui --app platform
  toolchains"` prints `kittui native-native --app platform toolchains`. `main_test.go`,
  `TestAnAppsWordOpensItOnTheTabNamed`: `open(r, "--app", []string{"platform", "tools"})` gives `shell.Platform` and
  a `flow.Model` maximized on tools, its title `kit platform`; `[]string{"people"}` and `nil` give the refusal.
- [ ] **Steps 2 to 4:** run them, implement, refresh the syntax fixture, and run `tests/tui.bats`,
  `tests/syntax.bats`, and `go test ./cmd/kittui ./internal/flow -count=1`.
- [ ] **Step 5: Rewrite `README.md:6-33`** for the screens there are, the palette, a verb's form, the welcome, the
  tutorials, and the seven apps with their tabs and which read kit; a tab as a `fields.json` entry and its keys; and
  the three words that open an app. `docs/features.md:209` names the tabs still planned.
- [ ] **Step 6: Commit** the kit paths as `#138 kit: kit tui --app opens an app on a tab`, then `projects/tui` as
  `#138 kittui: --app opens any app on a tab, and the README describes the seven apps`.

## Tabs that need a kit stage first

These are not tasks: each needs a read kit does not have. Once it exists, the tab is one catalog entry.

| Tab | The read, and its row | What stands in the way |
| --- | --- | --- |
| Changes' checks | `kit flow checks --json`: `{change, name, verdict, age, took, member}` | The snapshot holds each change's runs (`modules/jq/forge.jq:56`); a stage needs its grammar, text, and mark. |
| Changes' reviews, merged | `kit flow reviews --json`: `{change, reviewer, state, unresolved, age, url}`; `kit flow merged --json`: a change's row and `mergedAt` | The forge read holds only whether each thread is resolved, and no merged change. |
| Issues' milestones, iterations | `kit flow milestones --json`: `{title, due, open, closed, url}` | `modules/bash/forge.sh` has no milestone call; iterations wait for the spec's mappings. |
| Platform's services, settings | `kit platform --json`, `kit config --json` | The spec's order 4. Delivery's other three, People's three, and Trust's other four wait for their seams. |

## Not in this plan

- The seven builders that stay Go: their detail lines, typed renderers, and links built from `origin`
  (Apps 8) need kit to print each row's page and reference, the spec's first decision.
- A new tab's glyph, color, and legend, which are `kit/modules/jq/glyphs.json`'s; and Apps 13 (an app
  opens without blocking on `--stages`), 28 (`--workspace <host/path>`), and 2 (a document printed with exit 1).
