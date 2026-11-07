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
| Layers | `shell` | the ordered open controls: focus, keys, clicks, overlays | `key`, `keys`, and the clicks in `flow` |
| Menu | `shell`, as `Dropdown` | open or not, the cursor, its keys and its clicks | `shell.Menu`, done; four copies in `flow` |
| Dialog | `dialog` | a titled box of acting lines, radios, and boxes | `popover.go`, `columns.go`, `shell.SettingsDialog`, the ask dialog |
| Frame | `frame` | title, app menu, search, tab bar, pane, notice with fade, keys bar and its right end | each screen's `View` |
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
and `shell` cannot import a package that imports `shell`. The app's menu is on it already; the tab menu, the column
menu, the heading menu, and the presets' menu each move onto it in a commit of their own. Disabled lines stay, dim,
as kittui's menus show them.

### Dialog

`dialog.Dialog` is a titled box of lines, each a heading, a gap, an acting line, a radio, or a box that toggles: what
the field popover, the fields dialog, the settings dialog, and the ask dialog draw. It answers the keys they share and
gives each acting line's function the line's value. The fields dialog's own keys, s, g, and ⇧↑↓, are extra bindings,
as a menu's are.

### Frame

`frame.Frame` draws a screen: the title line with ≡ and the search at its right end; the tab bar, when the screen has
tabs; the pane the screen gives it; the layers' boxes; and the last line, a notice while one stands, else the keys bar
with its right end, as the rate limits use it. It owns the notice: what was done fades after `Fades`, and why
something was not done stays until a key. A screen gives its title, its tabs, its pane, its layers, and its right end,
and gets the lines back. Maximize, under way for #85, makes the pane's top the frame's to say, which this keeps.

### Table

`table` is `grid` with what `flow` adds to it, made generic over a catalog. A table is given a catalog, the fields
with their types, renderers, and default placements, read from a screen's JSON, and records, each a value per field
id. It owns the presets and their storage, the automatic renderings, the header and the preset line, the fields
dialog, the column and heading menus, the field popover, and the outline, each as a control on the layers. Renderers
and value types stay registered by name, as now, so the catalog stays data. `flow` keeps what is about stages: the
JSON of `kit flow <stage>`, the record builders in `rows.go`, the links in `origin.go`, the actions in `act.go`, and
the glyph legend.

### Source

`source.Source` is one read of kit that a screen shows: a command, its arguments, and the decode of its answer. It
owns what `flow` spreads over its model and its stages: the load's generation and context, its deadline, the kept
copy shown while it loads, the mark that says it is stale, Watch, which loads it again once its mark moves, and the
minute's polling. The rate limits and the marks are sources too, read for the whole screen. `welcome`'s checks,
`tutorials`' catalogue, and `palette`'s verbs become sources, and gain the kept copy and the staleness for free.

Every timer goes through `source.Clock`, `After(d, msg) tea.Cmd`. The program's clock is `tea.Tick`; `testkit`'s
records each timer and fires none until the test says so, so no test can hang on one.

## The order of the moves

Each step is one commit, keeps every test and golden as it is, and lands while `flow` is quiet, as the sessions in
the worktree agree.

1. `shell.Dropdown`: the menu type and the app's menu on it, done; then the four menus in `flow` moved onto it, one
   commit each, smallest first. It removes the most copied code at the least risk.
2. `shell.Layers` and `Control`: the menus first, then the dialogs, the search, the preset line, and the rows. `key`,
   `keys`, and the clicks in `flow` shrink to one call each.
3. `dialog`: the popover, the fields dialog, the settings dialog, and the ask dialog moved onto it.
4. `frame`, with the notice: `tutorials` first, as the newest and smallest, then `welcome`, `palette`, and `flow`.
5. `source` and its clock: the marks, the rate limits, and the kept copy first, as they are already apart in
   `marks.go`, `quota.go`, and `kept.go`; then the stages' loads; then the other screens' loads.
6. `table`: the catalog, the renderers, and the presets out of `flow`, then the dialogs and menus that act on a table.
   It is the largest move, and the steps before it have made its parts controls and its loads sources.

Watch, issue #88's third step, is built now as designed: it reads like `source`'s, and step 5 moves it with the marks.

## Decided

- A table reads each record as a map of values by field id, as it does now. An interface a table asks for its values
  is not needed yet.

## Left to the builder

- The names of the packages, and whether `dialog` folds into `shell` beside `Dropdown`.
- Whether `Control.Update` returns the control, as Bubble Tea's models do, or works on a pointer, where a control is
  part of a larger value.
- How the layers carry a control's identity, so a screen can ask whether its column menu is open.
