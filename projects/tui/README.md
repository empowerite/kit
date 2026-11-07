# kittui

`kittui` is the kit's interactive terminal UI. It is a Go binary, and `kit tui`, `kit flow`, `kit welcome`, and
`kit tutorials` exec it. It runs `kit` for everything it knows, so it holds no verb by name.

It has five screens:

- **The palette.** Bare `kit` opens it. It lists every verb under its group, filtered as you type.
- **A verb's form.** It opens when a verb at a terminal is short of its arguments. It finishes the command line.
- **The flow picture.** `kit flow` at a terminal draws its stages here.
- **The welcome guide.** `kit welcome` at a terminal walks its steps here.
- **The tutorials.** `kit tutorials` at a terminal shows a card for each tutorial here, tabbed by audience.

Each screen reads what it shows from kit as JSON. A form reads `kit syntax <verb>`, and the picture reads
`kit flow <stage> --json`. `kittui` never parses kit's text.

## Where kit finds it

`kit tui` looks for `kittui` in this order:

1. In a checkout of the kit, with `go` on `PATH`, it builds `kittui` from this directory. It builds once for each
   change to these sources or to `go`, into the cache.
2. In a release, it runs the binary for this platform from `libexec/`. `bin/release` builds those for linux and
   darwin on amd64 and arm64.
3. Otherwise, it runs a `kittui` on `PATH`.

[kit/modules/bash/tui.sh](../../kit/modules/bash/tui.sh) holds that lookup.

## Layout

- `cmd/kittui/` is the binary. It picks a screen from its arguments.
- `internal/app/` is the root model: the palette, a verb's form, and the slide between them.
- `internal/palette/` is the list of verbs that bare `kit` opens.
- `internal/form/` turns a verb's grammar into form fields, and the fields back into a command line.
- `internal/syntax/` decodes the grammar `kit syntax` prints.
- `internal/flow/` draws `kit flow`: the stages as tabs, and each stage's rows. `fields.json` is the catalog. It
  declares each stage's fields, each by a stable id, with its label, its type, what it can sort, group, and filter
  by, and its renderings in order, and the stage's own presets, Default and Compact. The renderers are Go functions,
  registered by name. A preset says, per field by id, its placement, its sort, its filter, and the rendering it pins.
  The person's presets are kept in `~/.config/kit/presets.json`, version 2. Where there is none, kittui reads the
  `views.json` beside it once, maps each column's label to its field's id, notes a label the catalog no longer knows,
  and leaves the old file as it is.
  - The column names always stand over the rows. The rule under them is the preset line, which takes no line of the
    rows. At its right end, after how many rows are selected and a chip per filter, it names the preset the table is set
    as and ▼, a pill on the tab's color, as a label is, on the menus' cyan while the line has the keyboard; the name is
    in italics once the table differs from every preset. ↓ from the tabs reaches it, and ⏎ or a click on it drops down
    the presets' menu, in the tab's color, as the pill is, titled Presets at its left as a dialog is, from the header's
    first line, its right edge over the ▼: 📇 Fields…, the columns dialog, then Save, Save As…, Rename…, Delete, and the
    presets. Save As… takes a name in the selector's place, the person's preset the line was set to typed already, and
    ⏎ saves; Rename… takes a new name the same way. Save, Rename…, and Delete act on the person's preset the line was
    set to, and are dim on Default and Compact. What was done is said in place of the keys, for five seconds or until a
    key; why something was not done stays until a key.
  - The columns dialog, in `columns.go`, titled by the stage's fields, opens from Fields… in the presets' menu, the
    header drawing no icon of its own: every field a line, those the rows group by first, then the columns as they
    stand, then the hidden, each with whether it shows, its sort and its place among the keys, and its group's depth.
    What it sets applies at once: space shows or hides, s turns the sort, g groups, ⇧↑↓ moves a field, into the groups
    or out, and ⏎ opens its Options…; a click on a line's box, sort, or group does as space, s, or g.
  - Maximize, first in the tab's menu, gives the rows the bar's lines but one, for the session: the title names the
    tab, its count, and its ⋮, which drops the tab's menu as the bar's does, a line in the tab's color runs under it,
    and the header follows; Restore, in its place, puts the bar back as it was, and a tab's number selects a tab
    either way.
  - The search stands at the title line's right end, and / or a click on it gives it the keyboard. There is no sidebar:
    the rows take the screen's width. While the person groups by no column, a value every row shares moves into one
    heading over them, the column grouping on its own; a group the person chooses overrides that, and the rows then
    group as chosen alone. Auto-group, which turns that on or off, and Repeated emoji, which keeps an emoji every row
    shows instead of dropping it, are kittui's settings, in = Settings…, every view's alike.
  - The legend, in `legend.go`, floats over the rows while F1 or ? shows it: what each glyph the selected stage's rows
    draw now means, and nothing of a glyph they do not, each glyph a click leads somewhere from marked. It opens in
    the bottom right corner; the arrows on its top edge, ↖ ↗ ↙ ↘, move it to another, the one it stands in dim. It
    takes no other key or click, so the rows under it go on as they were.
  - ⏎ on a field in the columns dialog opens its popover: the checklist of values its filter lets through or holds back,
    for an enum or a person, and its rendering; the sort is the column menu's. What it sets applies at once. A click on
    a column's name drops down its menu, around the name, as ≡'s and ⋮'s hang around theirs, titled by the name, which
    no line repeats: its sort, a radio that shows it, None, Ascending, or Descending, and 🧹 Clear Sort; ▤ Group and ▢
    Ungroup; ☐ Hide; and ⋯ Options…, its popover, titled over the name. A column's menu, and a group heading's, which
    has Ungroup too, take shift with ← or →, which moves the column a place among those the header names, or a depth
    among the groups, the menu with it. Sorting always adds to the keys: a column that did not sort comes after the
    others, and one that did turns in its place; grouping always adds the column innermost, and the columns dialog
    orders both. The groups order the rows before any key, the outermost 1, so a column's ▲ or ▼ carries its place after
    them, raised after ▲ and lowered after ▼, as each points, ▲² and ▼₂, wherever more than one thing orders the rows;
    the menu's triangles carry ⁿ and ₙ there. The line under the keyboard in a menu or a dialog stands on the menu's
    bright cyan, in black, and a dialog's keys are the keys bar's alone. A click on the cell right after a column's
    name, or on the cell after that unless it stands against the next name, sorts by the column, ascending, after the
    others, and on a column that sorts already turns its sort around; that is where its ▲ or ▼ stands, and a shift-click
    on the mark stops the sort. A terminal such as iTerm keeps ⌥, ⌃, shift, and the right button with a click for its
    own selection and menu, so the menu is the one way that works everywhere. Where the terminal sends them, ⌥ with a
    click turns a sort, ⌃ groups or ungroups, and the right button drops the menu too. The presets' menu hangs around
    the preset line's selector the same way.
  - A field whose rendering is auto drops a glyph every row shown shares, and, while the columns are short of width,
    steps down to its next, shorter rendering, from the rightmost column leftward. A glyph that links, to a page or to
    another stage's row, is a button, so it stays, and a field drawn without its glyph links from nothing. The title
    never steps, and a rendering pinned always wins.
  - The rows are an outline, in `outline.go`: a group's heading, a row, and a header of more than one line each open
    and close, ▾ or ▸ in the gutter, and one map per stage holds which stand open, a heading open and a row closed
    until changed. → opens the node under the cursor, or steps into an open heading's first row; ← closes it, or
    steps out to the heading over it; ⌥ with either, or with a click, does the same to everything under the node. The
    tab's menu has Expand, Collapse, and Collapse Others, which act on everything, or on the selection while rows are
    selected: Expand Selected opens the groups over them too, and Collapse Others folds every group without one.
    Space on a heading selects every row of its group, or none once all are, and the heading's mark, ▣ or ◩, says
    whether all are or some. What stands open lasts the session; a preset does not keep it. Grouped rows stand under
    one heading that names the grouped columns, `▾ ▾ source › ▾ assignees`, and a heading per value under it. A click
    on a column's name in that heading, or ⏎ on it, drops the column's own menu, in `heading.go`, as a name in the
    header does; its first ▾ or ▸ folds it all, and the one before each name shows the groups down to that level and
    nothing under it, collapsing or expanding to it. A click anywhere on a value's heading opens or closes it. A field of
    several values, as assignees or labels, groups too: a row stands under each of its values, the same row, its
    selection one.
  - Auto-group, a setting, groups a view's rows as the view loads, as the screen opens or the preset line sets it,
    when the view groups by nothing: by each groupable column shown whose value is the same on every row, left to
    right. From then on they are groups like any other, to move, ungroup, or save; a reload changes nothing, and
    the preset line still names the view.
  - An open row breaks a value too wide for its column across lines, and one a field draws with a glyph hangs: the
    lines after its first start under the text, not under the glyph.
- `internal/welcome/` draws `kit welcome`: one tab per page under `.kit/welcome/`.
- `internal/tutorials/` draws `kit tutorials`: a card for each tutorial, one tab for each audience.
- `internal/tabs/` is the tab bar `kit flow`, `kit welcome`, and `kit tutorials` share: three lines of numbered tabs,
  and the keys and the clicks that select them.
- `internal/kit/` runs the `kit` CLI for the other packages.
- `internal/transcript/` writes the command line that `kittui` ran into the scrollback.
- `internal/shell/` is what every screen shares: the title line and the app's menu, the search, the Settings dialog
  and kittui's settings file, and the keys bar. A screen never writes its
  keys bar. It gives the path from the screen to the control that has the keyboard, and each control on the path names
  the keys it answers. A deeper control hides an ancestor's key for a press it takes. The bar draws each control's keys
  between rules, and drops whole controls from the screen's end when the line is too narrow.
- `internal/style/` is the look: one accent color, the glyphs, the keys bar's styles, and the pill a label or an
  audience is drawn as, its text on its color between half circles. It honors `NO_COLOR`.
- `internal/testkit/` holds what the tests share. Only tests import it.

## Testing

Run these from this directory. `bin/ci` runs the same three:

```sh
test -z "$(gofmt -l .)"
go vet ./...
go test ./...
```

The flow, form, and welcome screens are checked against golden files in their `testdata/`. After a change you mean
to make, rewrite them and review the diff:

```sh
go test ./internal/flow ./internal/form ./internal/welcome -update
```

Name the packages. `./...` fails, because packages that do not import `internal/testkit` have no `-update` flag.

`internal/syntax/testdata/*.json` are copies of what `kit syntax <verb>` prints. `tests/syntax.bats` fails when one
is stale, and it prints the command that refreshes it.

Tests that run kit run a fake one, a `testdata/fake-kit` script, never the real one. The flow legend tests read the
real [kit/bin/kit-flow](../../kit/bin/kit-flow), so the picture and the help agree on every glyph.
