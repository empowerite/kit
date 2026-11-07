# kittui

`kittui` is the kit's interactive terminal UI. It is a Go binary, and `kit tui`, `kit flow`, and `kit welcome` exec
it. It runs `kit` for everything it knows, so it holds no verb by name.

It has four screens:

- **The palette.** Bare `kit` opens it. It lists every verb under its group, filtered as you type.
- **A verb's form.** It opens when a verb at a terminal is short of its arguments. It finishes the command line.
- **The flow picture.** `kit flow` at a terminal draws its stages here.
- **The welcome guide.** `kit welcome` at a terminal walks its steps here.

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
    rows. At its right end, after how many rows are selected and a chip per filter, it names the preset the table is
    set as and ▼, or ▼ alone once the table differs from every preset. ↓ from the tabs reaches it, and ⏎ or a click on
    it drops down Save As… and the presets. Save As… takes a name in the selector's place, and ⏎ saves.
  - The sidebar is the field editor: under the search, each field in one of three sections, group by, columns, and
    hidden. ⇧↑↓ or a drag moves a field within its section, or across a heading into the next, and space hides a field
    or shows it again. Under them, two boxes: fold shared values, and show shared emoji, which a preset keeps.
  - ⏎ on a field opens its popover: its sort, the checklist of values its filter lets through or holds back, for an
    enum or a person, and its rendering. What it sets applies at once. A double click on a column's name opens the
    same popover, and puts back the sort the first click of it changed. A click on the name sorts by it alone, and
    with ⌥ adds it after the other sort keys. With ⌃, or the right button, it groups by it alone, and with ⌥⌃ nests
    it within the groups. Shift is not used, because many terminals keep it for their own selection.
  - A field whose rendering is auto drops a glyph every row shown shares, and, while the columns are short of width,
    steps down to its next, shorter rendering, from the rightmost column leftward. A glyph that links, to a page or to
    another stage's row, is a button, so it stays, and a field drawn without its glyph links from nothing. The title
    never steps, and a rendering pinned always wins.
- `internal/welcome/` draws `kit welcome`: one tab per page under `.kit/welcome/`.
- `internal/tabs/` is the tab bar `kit flow` and `kit welcome` share: three lines of numbered tabs, and the keys and
  the clicks that select them.
- `internal/kit/` runs the `kit` CLI for the other packages.
- `internal/transcript/` writes the command line that `kittui` ran into the scrollback.
- `internal/shell/` is what every screen shares: the title line and the app's menu, the search, the Settings dialog
  and kittui's settings file, the double clicks Bubble Tea does not report, and the keys bar. A screen never writes its
  keys bar. It gives the path from the screen to the control that has the keyboard, and each control on the path names
  the keys it answers. A deeper control hides an ancestor's key for a press it takes. The bar draws each control's keys
  between rules, and drops whole controls from the screen's end when the line is too narrow.
- `internal/style/` is the look: one accent color, the glyphs, and the keys bar's styles. It honors `NO_COLOR`.
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
