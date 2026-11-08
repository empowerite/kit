# kittui components: design

Status: proposed, 2026-11-07.

## Intent

kittui is a proof of concept that will be read back into a spec and rebuilt piece by piece. The pieces are easiest to
lift, and to reuse on the next screen, when each is a component with a narrow contract and a home of its own. Today
most of them live inside one screen. This design names the components, gives each a package, and orders the moves so
every step keeps the tests green.

## What the code is now

The packages are layered without a cycle: `style`, then `pane`, then `shell`, then `grid` and `tabs`, then the
screens (`flow`, `form`, `palette`, `welcome`, and `tutorials`), then `screens` and `app`, which switch between them.
The layering is sound. What it lacks is a home for the parts every screen rebuilds.

| Package | Files | Lines, without tests |
| --- | --- | --- |
| `flow` | 20 | 6,769 |
| `form` | 4 | 1,934 |
| `grid` | 2 | 1,832 |
| `welcome` | 4 | 1,436 |
| `shell` | 5 | 1,020 |
| `tutorials` | 2 | 677 |
| `palette` | 1 | 531 |
| the other ten, `cmd/kittui` among them | 11 | 2,606 |
| all of kittui | 49 | 16,805 |

`flow` is two fifths of kittui, and its `Model` has 61 fields. Five findings follow from reading it beside the
other screens.

1. Menus are written five times. The tab menu, the column menu, the heading menu, the presets' menu, and the app
   menu's keys each keep their own open flag, cursor, and anchor, and each has a key handler of the same shape: up
   and down through `shell.Move`, enter, esc, and a typed letter through `shell.NextItem`. `columnKey` in
   `column.go` and `headingKey` in `heading.go` differ only in their names. `shell` holds the drawing, `shell.Box`,
   and the cursor's steps, but no menu that holds its own state.
2. Focus is decided in more than one place. `key` in `flow.go` asks each open control in turn, from the ask dialog
   down to the rows, in a chain of `if`s; `keys` in `view.go` repeats the order to build the keys bar's path; the
   clicks and the overlays repeat it again. A new popup has to be added to each, in the same place.
3. Each screen builds the same frame. `flow`, `welcome`, `tutorials`, and `palette` each draw the title line with the
   app menu's ≡, a tab bar, the pane, the overlays, and a last line that is the keys bar or a notice. `flow` and
   `tutorials` each keep a notice of their own, and `flow` alone makes it fade.
4. The table is split across two packages. `grid` lays the rows out and keeps a panel's per-field state; `flow` holds
   the field catalog, `fields.json`, the value types and renderers, the presets and their storage, the automatic
   renderings, the fields dialog, the column and heading menus, the field popover, and the outline. None of that is
   about stages, but a second screen with a table could use none of it.
5. Reading kit is part of the screen. A stage's load, with its generation, its context, and its deadline, the kept
   picture, the marks and their minute, and the rate limits and theirs all live on `flow.Model`. `welcome`,
   `tutorials`, and `palette` each load from kit in their own way. A timer is a `tea.Tick` inside the model, which
   `testkit.Drive` runs in place: a fake kit that answered `--marks` by accident once hung two packages' tests for
   ten minutes.

## Goals

- Each component below lives in a package of its own, with a contract a second screen can use as it stands.
- Focus, key routing, click routing, overlay order, and the keys bar come from one ordered list of open controls.
- A screen states what it shows, its pane, its controls, and its sources, and the frame draws the rest.
- Reading kit, with caching, staleness, polling, and Watch, is one component the screens share.
- Every step of the move keeps the tests, the goldens among them, passing, and changes no behavior.

## Non-goals

- A declarative layout for whole screens. Issue #85 leaves that to a follow-up, and this design leaves room for it.
- A new look. The components draw what the screens draw now.
- Moving the field catalog into kit. It stays in kittui, as #85 decided.

## Anti-goals

- A framework. A component is a value with methods, in the Bubble Tea manner, not a registry, an event bus, or an
  inheritance tree. If a contract needs more than four methods, it is two components.
- A big-bang rewrite. A step that cannot land as one commit, green, while other sessions work in the same worktree,
  is too big.

## The components

Each row names the component, its package, what it owns, and where it is today.

| Component | Package | Owns | Today |
| --- | --- | --- | --- |
| Control | `shell` | the contract every interactive piece meets | implicit, per screen |
| Layers | `shell` | the ordered open controls: focus, keys, clicks, overlays | done: `flow`'s eight popups, in `layers.go` |
| Menu | `shell`, as `Dropdown` | open or not, the cursor, its keys and its clicks | done: the app's, the tab's, the column, and the presets' menus |
| Dialog | `shell`, as `Placed` and `Step` | where a box stands and a click falls on it, and the cursor's step | done: the four dialogs |
| Frame | `shell`, as `View` and `Notice` | the screen's end on its last line, and the notice with its fade | done: four screens' views, two screens' notices |
| Table | `table` | catalog, types, renderers, presets, auto renderings, fields dialog, column and heading menus, popover, outline | `flow` and `grid` |
| Grid | `grid` | laying rows out in columns, and the panel's per-field state | `grid`, as now |
| Source | `source` | a read of kit: generation, cancel, deadline, kept copy, mark, staleness, Watch, polling | `flow.Model`, and each screen's loads |
| Clock | `source` | when a tick fires, which a test sets | `tea.Tick` in each model |
| Pill, glyph | `style` | drawing a label as a pill, glyph widths | `style.Pill`, done |

### Control and Layers

A control is anything that takes keys or clicks while it is open: a menu, a dialog, the search, the preset line, the
rows. Its contract, in `shell`:

```go
type Control interface {
    Update(msg tea.Msg) (Control, tea.Cmd, bool) // handled says the message stops here
    Keys() Segment                               // what the keys bar names for it
    Box() (lines []string, top, left int)        // what it draws over the screen, none for an inline control
}
```

`shell.Layers` is the screen's open controls, the topmost first. A key goes to each in turn until one handles it; a
click goes to the first whose box holds it, and a click outside every box closes the topmost, as a click off a menu
does now; `Path` gives the keys bar from the same order; `Overlay` draws the boxes bottom up. A screen opens a control
by pushing it, and a control that closes pops itself. Focus is then one list, and a new popup is one push.

### Menu

`shell.Dropdown` holds what each copy in `flow` holds: whether it is open, and the line under the keyboard. Its lines
and which of them act are its owner's, given to each call, since they follow what the screen shows. It answers up,
down, enter, space, esc, a key its owner names, a typed letter, and a click on a line of its box, and gives back the
line chosen. A menu that does more, as the column menu moves its column on ⇧←→, takes those keys before it hands the
rest to the dropdown. It lives in `shell`, not a package of its own, because the app's menu, `shell.Menu`, embeds it,
and `shell` cannot import a package that imports `shell`. The app's menu, the tab menu, the column menu, and the
presets' menu are each on it; the heading menu is gone. A click on a line that cannot act does nothing on every one of
them. Disabled lines stay, dim, as kittui's menus show them.

### Dialog

What the four dialogs share is narrower than a dialog type: `shell.Placed`, a box as `shell.Dialog` draws it, where it
stands, which draws itself and says whether a click falls off it, on its edges, or on a body line; and `shell.Step`,
the line under the keyboard moving to the next line it can stand on and stopping at the ends. The field popover, the
fields dialog, the Settings dialog, and what a backlog action asks each use both, and keep what they show, how they mark
the line under the keyboard, where they stand, and their own keys, as the fields dialog's s, g, and ⇧↑↓, and the ask
dialog's typing. A type that drew their lines too would hold four looks for four dialogs, so it stops here.

### Frame

What the screens share of their frame is two parts of `shell`. `shell.View` ends a screen: its lines over the last
line, on the alternate screen, with the mouse on; `flow`, `tutorials`, `welcome`, and `palette` end in it, and the
form, which takes no click, keeps its own. `shell.Notice` is what stands in the keys bar's place: what was done, which
fades after a while unless a key or a later notice takes its place, or why something was not done, which stays until
a key; `flow` and `tutorials` keep one each. The tab bar is `tabs`' already. The title line and its search stay each
screen's own: each lays the search out in its own room, and one shared line would move it on some of them.

### Table

`table` is `grid` with what `flow` adds to it, made generic over a catalog. A table is given a catalog, the fields
with their types, renderers, and default placements, read from a screen's JSON, and records, each a value per field
id. It owns the presets and their storage, the automatic renderings, the header and the preset line, the fields
dialog, the column and heading menus, the field popover, and the outline, each as a control on the layers. Renderers
and value types stay registered by name, as now, so the catalog stays data. `flow` keeps what is about stages: the
JSON of `kit flow <stage>`, the record builders in `rows.go`, the links in `origin.go`, the actions in `act.go`, and
the glyph legend.

As built so far, `table` holds the catalog's fields and presets, read from a screen's JSON by `ReadCatalog`; the
values a row holds; the renderers every table has, and `Register` for a screen's own, by name; `Automatic`, the
choice among a field's renderings; labels as pills; a preset set on a panel and read back with `PresetOf`; `Views`,
the rules of which view a table is set as and what the person's can be saved, renamed, and deleted as; the presets
file; and `Table`, a table's state, which `flow` embeds in each stage. A table's controls take a `Host`, which the
screen implements once, for the table selected: its fields, its panel and setting it, the values its rows hold, a
label's color, the caller's login, the rendering a field draws by now, the screen's size, the table's name, its pane,
the columns its header shows, and where a column's name stands in the header and in the heading over the groups. On
it are `Popover`, `Columns`, the fields dialog, and `ColumnMenu`; each gives back the popover it opens, so the screen
asks kit for the caller's login as it did.

Still in `flow` are the parts that the rows' own keys and drawing hold together: the heading over the groups on the
control line, the outline, its folds and its selection marks, the preset line and its menu, the header and its rule,
and the rows' cursor, paging, and scrolling. They move together, as the table's view, a control that draws the
control line, the header, and the rows' window from the rows the host gives it, and takes the rows' keys.

### Source

`source` reads kit for a screen, in two shapes, each generic over its answer.

- `source.Load` is a read the screen shows, as each of `flow`'s stages is, and `welcome`'s checks and `tutorials`'
  catalogue. It holds the read's generation and context: a read started again with `Again` supersedes the one
  before, which it ends, and whose answer `Take` drops. It holds whether nothing shows yet, `Loading`; whether the
  latest read is still out, `Running`, while what showed before stays; the copy kit kept, which stands in, stale,
  while the first read runs, for a command that keeps one; the mark it took after it last answered, which `Marked`
  compares with the next read of the marks; and `Watch`, which reads it again once its mark moves. `flow` keeps a
  stage's decoding, its refusal, and what a fetch reloads. `kit welcome` and `kit tutorials` keep no copy and have no
  marks, so those screens gain the superseding and the end of a run still out, not the kept copy or the staleness.
- `source.Poll` is a read asked again a while after each answer, as `flow` reads the rate limits and the marks. A read
  that fails keeps the last answer; the first to fail before any answered stops it until `Reset`, as Refresh All does.

The answers are `source.Answer` values, named by the source, so a screen routes them with a type switch and a test
builds one. The palette reads its verbs before the screen opens, so it has no source.

Every wait goes through `source.Clock`, `After(d, msg) tea.Cmd`. The program's clock is `tea.Tick`; `testkit.Clock`
keeps each wait and brings none until the test fires it, so no command a test drives can block on one. An env's limit
of 0 is no deadline but the screen's own, as `welcome` and `tutorials` read.

## The order of the moves

Each step is one commit, keeps every test and golden as it is, and lands while `flow` is quiet, as the sessions in
the worktree agree.

1. `shell.Dropdown`: the menu type, and every menu on it, one commit each. Done.
2. `shell.Layers`: done for `flow`'s popups, its menus and dialogs, in `flow/layers.go`; `key`, `keys`, the clicks,
   and `View` each make one call to it. A layer is a table of functions on the screen's model, not a `Control` value,
   as the popups are the model's methods; the search, the preset line, and the rows are the screen's still.
3. `shell.Placed` and `shell.Step`: the popover, the fields dialog, the Settings dialog, and the ask dialog on them.
   Done.
4. `shell.View` and `shell.Notice`: the screens' ends and the two notices on them. Done.
5. `source` and its clock: the marks and the rate limits on `source.Poll`, then the stages' loads and the kept copy
   on `source.Load`, then `welcome`'s and `tutorials`' loads. Done.
6. `table`: the catalog, the renderers, and the presets out of `flow`, then the dialogs and menus that act on a table.
   It is the largest move, and the steps before it have made its parts controls and its loads sources. Done as far
   as the controls that stand apart from the rows: the catalog, the renderers, the automatic renderings, the presets,
   the views and their file, the table's state, the host, the popover, the fields dialog, and the column menu.
7. The table's view: the heading over the groups, the outline, the preset line, the header, and the rows' cursor,
   paging, and scrolling, out of `flow` together, onto the host.

Watch, issue #88's third step, is built now as designed: it reads like `source`'s, and step 5 moves it with the marks.

## Decided

- A table reads each record as a map of values by field id, as it does now. An interface a table asks for its values
  is not needed yet.

## Left to the builder

- Whether `Control.Update` returns the control, as Bubble Tea's models do, or works on a pointer, where a control is
  part of a larger value.
- How the layers carry a control's identity, so a screen can ask whether its column menu is open.
