# kittui components: design

Status: built, 2026-11-08: every move done, as each says. The frame, its terms, and the rules for keys decided,
2026-11-09 (#94); #95 to #106 build them.

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

Every screen has the same frame, in three parts, in the same places (#96):

- The menu bar, the first line: the kit menu `=`, the screen's name, a separator, the workspace menu `-` and the
  workspace's name, a separator and the tab menu while a tab is maximized, space, and the search field at the right
  end. Another menu may come to stand right of the search field.
- The status bar, the last line: the key hints, space, and the quota meter. A notice or an alert stands over the key
  hints.
- The view, between them, which is each screen's own: a tab bar, a control line, a table, cards, a form.

What the screens share of it is in `shell`. `shell.View` ends a screen: its lines over the last line, on the
alternate screen, with the mouse on; `flow`, `tutorials`, `welcome`, and `palette` end in it, and the form keeps its
own until #96 brings it into the frame. `shell.Notice` is what stands over the key hints: a notice, what was done,
which goes away after a while unless a key or a later notice takes its place; or an alert, why something was not done,
which stays until esc. `flow` and `tutorials` keep one each. The tab bar is `tabs`'. The menu bar is each screen's
own today, each laying the search field out in its own room; #96 makes it one line every screen draws alike.

### Terms

These are the names the code, the docs, the key hints, and the tutorials use for what the person sees. A name
used in the code today that differs is in the last column, to be renamed as the code it is in is touched.

| Where | What it is | Term | Today |
| --- | --- | --- | --- |
| first line | the whole line | menu bar | title line |
| | `=` | kit menu | app's menu |
| | `kit flow` | screen name | title |
| | `·` | separator | dot |
| | `-` and `empowerite/kit` | workspace menu, workspace name | the workspace menu |
| | `🎫 backlog :` while a tab is maximized | tab menu | tab's menu |
| | `🔎 /search` | search field | the search |
| last line | the whole line | status bar | keys bar |
| | the keys, at the left | key hints | keys bar |
| | the rate limits, at the right | quota meter | quota |
| | a message that goes away on its own | notice | notice, done |
| | a message that stays until esc | alert | notice, not done |
| between | everything else | view | pane, screen |
| view | `➊ 🎫 backlog 3 :` | tab bar, tab number, tab menu | same |
| | `◃` and `▹` | scroll arrows | triangles |
| | `▾ source ▾ 3 … Default ▼` | control line | same |
| | `▾ source ▾ 3` | group path | heading over the groups |
| | `source: ¬gitlab` | filter chip | chip |
| | `Default ▼` | preset picker | preset line, pill, selector |
| | the column names and the rule under them | header | same |
| | `▾ github 3` | group heading | heading |
| | `▸` and `▾` | disclosure triangle | fold |
| | the band the arrows move, and the rows space picks | cursor, selection | same, picked |
| over the view | a list hung from the glyph or name that opens it | menu | menu, dropdown |
| | a titled box that takes focus until esc | dialog | same |
| | a dialog hung from the control it sets | popover | same |
| | a box that floats and takes no focus | legend | same |
| | a dialog that asks one question | prompt | ask |
| everywhere | where the keys go | focus | has the keyboard |

A screen is one of kittui's four: Flow, Welcome, Tutorials, and Help, which the kit menu moves between. They are
screens of one program, not applications.

### Keys

These rules hold on every screen, in every control.

- **esc only cancels or goes back** (#95). It closes what is open, or moves focus back one level. At the top level
  it does nothing. It never quits.
- **⌃C and ⌃Q quit** (#95), on every screen. Raw mode turns off flow control, so ⌃Q reaches kittui. No modifier on
  esc quits: macOS terminals with default settings send ⌥esc, ⌃esc, and ⇧esc as a plain esc, and keep ⌘ keys for
  themselves.
- **Search works the same everywhere** (#97). `/` or a click on the search field gives it focus; typing filters as it
  goes; ⏎ or ↓ gives focus to the view and keeps the filter; esc does too; ⌃U clears the field; ⌃R recalls the past
  searches.
- **⇞ and ⇟ come with Home and End** (#98). Wherever ⇞ and ⇟ page, Home goes to the first item and End to the last.
- **Groups move by brackets** (#102). `[` and `]` move to the previous and the next group, `{` and `}` to the first and
  the last.
- **A shortcut works where it is shown** (#102). A menu line shows its key at its right, dim, to teach it, and the key
  does the same with the menu open as with it closed: with a menu open, a line's shortcut chooses that line, and any
  other letter jumps to the next line starting with it. A shortcut is the line's first letter where it can be.
- **No key or click is shown that a terminal may not send** (#103). A key or click is shown when a test shows it
  arrives in Terminal.app and iTerm with their default settings, or when kittui asked the terminal and it answered, as
  ⇧⏎ is shown only when the kitty keyboard protocol answers. Every action has a plain key and a plain click; a
  modifier or right-button click may stay as an unnamed extra.
- **The status bar keeps the screen's keys** (#99). When the key hints do not fit, they drop the controls between the
  screen and the one with focus, never the screen's own.

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

The table's view is `table`'s too, on the same host, which now also gives the rows as the table lays them out, the
header's names and the width, the lines it draws and its header, the pane's height, whether a control line stands
over the rows, the table's views, its color, and where its preset line's selector stands, and takes and gives up the
rows' keyboard. On it are the outline (`Expand`, `Grow`, `Peel`, `Folding`, and the marks of a heading whose rows are
picked); the heading over the groups (its names, and its folds, which open and close the levels); the preset line's
selector, its menu, and the chips of the filters; and the pane's layout (the headings of one value frozen over the
header, the header, the rows' window and the line it pins on top), the cursor's moves a row or a page at a time, and
the selection.

`flow` keeps what draws a stage or is about one: it draws the lines, the control line, the header and the rule with
the sorts' marks, and caches the lines it draws; it builds the rows from kit's records; it says what a preset saved,
renamed, or deleted did, in its own words, and writes the person's presets; and it holds the keyboard's focus, which
a control hands back by what it returns or by the host's calls. Each move keeps the name `flow` gave it, a call to
`table`'s.

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
   paging, and scrolling, out of `flow` together, onto the host. Done, its drawing apart, which `flow` keeps, as it
   caches the lines and says what a stage holds.

Watch, issue #88's third step, is built now as designed: it reads like `source`'s, and step 5 moves it with the marks.

## Decided

- A table reads each record as a map of values by field id, as it does now. An interface a table asks for its values
  is not needed yet.
- One frame for every screen, its terms, and the rules for keys, as the Frame section says (2026-11-09, #94). They
  replace an earlier decision that each screen keeps its own title line.

## Left to the builder

- Whether `Control.Update` returns the control, as Bubble Tea's models do, or works on a pointer, where a control is
  part of a larger value.
- How the layers carry a control's identity, so a screen can ask whether its column menu is open.
