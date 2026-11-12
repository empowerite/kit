# kittui

`kittui` is the kit's interactive terminal UI. It is a Go binary, and `kit tui`, `kit flow`, `kit workspace`, the app
verbs below, `kit welcome`, and `kit tutorials` exec it. It runs `kit` for everything it knows, so it holds no verb by
name.

It has six screens:

- **The palette.** `kit help` opens it, focus in the rows, so the arrows move at once. It is a table of every verb, its
  group, command, and description, grouped by group and sorted by command; / gives the search focus, and typing
  filters the rows; ⏎ or ↓ gives the rows focus again, → opens a row to the verb's help page, and ⏎ opens the verb's
  form. [ and ] move to the group before and after, { and } to the first and the last.
- **A verb's form.** It opens when a verb at a terminal is short of its arguments. It finishes the command line.
- **The flow picture.** `kit flow` at a terminal draws its stages here. Over a group, `kit --workspace=<group> flow`
  or one picked from the 📚 workspace menu, every tab gathers every cloned member's rows, grouped by member as they
  load. An action on rows runs in each row's member's checkout.
- **The workspace picture.** `kit workspace` at a terminal draws what git holds across the workspace here: the 📚 repos
  tab first, a row per member of the workspace, then a tab for each of git's entities, remotes, branches, worktrees,
  commits, and tags. It is the flow picture's screen on kit workspace's stages, which kittui knows after `workspace/`,
  so their fields and presets stay apart from another verb's stage of the same name. A stage's rows are drawn from the
  catalog alone: kit prints each row's values by the field's id, and where a value leads, a page on the forge or a row
  of another tab, so a stage needs no Go of its own. It reads no forge, so it spends no rate limit, and a tab is stale,
  its dot lit, when `kit workspace --marks` moves. The other apps are the same screen; see The apps, below.
- **The welcome guide.** `kit welcome` at a terminal walks its steps here, and bare `kit` opens it too.
- **The tutorials.** `kit tutorials` at a terminal shows a card for each tutorial here, tabbed by audience.

Search works the same on every screen. `/` or a click on the search field at the menu bar's right gives it focus, and
typing filters as it goes. ⏎, ↓, esc, or tab gives the view focus and keeps the filter, ⌃U clears it, and ⌃R recalls
a past search.

`kit <verb> <argument>` opens the verb's screen on the argument. `kit flow issues` opens the picture with the issues
stage maximized, and both `kit flow backlog` and `kit flow underway` open it the same way, issues set as its Backlog
or Underway preset. `kit tutorials screens` opens the tutorials with the cursor on that card. `kit help commit` opens
the palette on the commit verb's page, and `kit tui commit` opens its form.

## The apps

Flow and the workspace's picture are two of nine apps, each this screen over its own tabs, one stage of kit for each,
drawn from the catalog alone. `kit <verb>` at a terminal opens the app on its first tab, and `kit <verb> <tab>` opens it
on that tab, maximized; a tab of kit flow's, as `kit issues issues`, opens the same way, and the app that holds it
opens. `kit issues backlog` and `kit issues underway` open Issues on its issues tab, set as that preset. Each tab reads
one kit command with `--json`, the entry of that stage in `internal/flow/fields.json`:

| App | Verb | Tab and the command it reads |
| --- | --- | --- |
| Flow | `kit flow` | each stage of the `flow` setting that kittui draws, as `kit flow <stage> --json` |
| Issues | `kit issues` | issues, `kit flow issues`, the backlog and underway; milestones and iterations, `kit issues <tab>` |
| Changes | `kit changes` | changes, `kit flow changes`; reviews, checks, and merged, `kit changes <tab>` |
| Repositories | `kit workspace` | repos, worktrees, branches, commits, tags, remotes: `kit workspace <tab>`; incoming: `kit flow` |
| Delivery | `kit delivery` | releases and builds, `kit flow`'s; artifacts, environments, deployments, `kit delivery <tab>` |
| People | `kit people` | people, teams, and agents, `kit people <tab>` |
| Trust | `kit audit` | evidence, findings, controls, waivers, policy, and enforcement, `kit audit <tab>` |
| Platform | `kit platform` | services, toolchains, tools, and settings, `kit platform <tab>` |
| Packages | `kit packages` | packages, `kit packages packages` |

Every tab has a source, so none is drawn empty. A tab is stale, its dot lit, when its verb's `--marks` moves for it, and
Watch reads it again then. A tab kit marks null, as kit flow's now are, has its Watch dim, and one the catalog says is
not `marked`, as People's three, Trust's controls and waivers, and Issues' iterations, offers no Watch at all; each is
read again on Refresh only. A calendar day, as a due, start, or end, draws as its UTC day. A tab of a stage that reads
the forge reads the rate limits again as it loads; any tab reads them as it loads while the app has none, so every app
draws them, Platform's too. A tab kit read only part of, as a capped list, counts what it drew with a +, `50+`; a
feature the forge has off here, as secret scanning, says so, dim, under the rows, and is no failure.
A link to a row of a tab the app does not have opens the app built around that tab's entity.

Each screen reads what it shows from kit as JSON. A form reads `kit syntax <verb>`, the picture reads
`kit flow <stage> --json`, and the workspace's reads `kit workspace <stage> --json`. `kittui` never parses kit's text.

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
- `internal/palette/` is the table of verbs that `kit help` opens.
- `internal/form/` turns a verb's grammar into form fields, and the fields back into a command line.
- `internal/syntax/` decodes the grammar `kit syntax` prints.
- `internal/flow/` draws `kit flow`: the stages as tabs, and each stage's rows. `fields.json` is the catalog. It
  declares each stage's fields, each by a stable id, with its label, its type, what it can sort, group, and filter by,
  and its renderings in order, and the stage's own presets, the first the one it opens on: Default and Compact, or for
  issues All Issues, Backlog, Underway, By Milestone, By Iteration, By Swimlane, and Compact; a table with no presets
  of its own has Default. A field that holds a list, as assignees
  or labels, names one of its values, assignee or label, and sorts by size while a row holds all of them, how many
  each holds, and by value once flattened. Where a row
  holds more than one, its column says how many, dim, as labels' 3, or, for a list with a glyph for many, after it, as
  assignees' 👥 2 in place of 👤 and a name; the row open lists them all under its lines. The renderers
  are Go functions, registered by name with `internal/table`, which has those every table shares. A preset says, per
  field by id, its placement, its sort, its filter, and the rendering it pins. The person's presets are kept in
  `~/.config/kit/presets.json`, version 2. Where there is none, kittui reads the `views.json` beside it once, maps each
  column's label to its field's id, notes a label the catalog no longer knows, and leaves the old file as it is.
  - The tab's view starts with the control line: the heading over the groups at its left, which, where the rows group by
    nothing, is the rows' ▾ and their count alone, and the preset line at its right end. After how many rows are
    selected, a chip per filter, which a click opens its field's card under, on its filter, and 📇 Fields, which opens
    the columns dialog, as f on the line does, it names the preset the table is set as and ▼, a pill on the tab's color,
    as a label is, on the menus' cyan while the line has the keyboard; the name is struck through and italic once the
    table differs from every preset. ↓ from the tabs reaches the line, on the heading over the groups' first stop; ⇥ and
    ⇧⇥ move between the groups' names, the rows' ▾ and count after them, and the pill; → opens the level of the stop
    under the keyboard and ← closes it, as ␣ turns it. ↓ from the line enters the rows on the outline's first line, the
    first group's heading where the rows group, as ↑ from there comes back. On a name ⏎ opens the column's card; on the
    rows' ⏎ opens or closes them. ⏎ or a click on the pill drops down the presets' menu, in the tab's color, as the pill
    is, titled Presets at its left as a dialog is, from the control line, its right edge over the ▼: Save, Save As…,
    Manage…, and the presets, in sections under dim headings: the person's favorites, in their order, then the team's,
    team, then the stage's own, built in, then the person's, mine, each preset once. Each has ★ at its right where the
    person starred it, ☆ where not, and `*` on its line, or a click on the star, stars it or no longer, the menu left
    open; the presets file keeps the favorites beside the presets, and a favorite follows its preset's rename, and goes
    when it is deleted. Save As… takes a name in the selector's place, the person's preset the line was set to typed
    already, and ⏎ saves. Save acts on the person's preset the line was set to, or the team's, and is dim on the stage's
    own. Manage…, in `manage.go`, opens the presets dialog, on the preset the line was set to: every preset a line, its
    star, its name, and team, built in, or mine, the favorites first, a rule under them; ↑↓ move, ␣ or `*` stars the
    preset, ⇧↑↓ moves a favorite among the favorites, and, on one of the person's own, r renames it, its new name typed
    in the selector's place as Save As… takes one, and d deletes it, the table set as the one it opens on where it was
    set as that preset; Rename… and Delete under the presets do the same by a click, and a stage's own preset has
    neither. The team's presets are checked into the workspace, in `team.go` and kit flow's own: a file each, named by
    its preset's slug, under `.kit/presets/<stage>` at the clone's root, read only where kit trust --check says kit
    trusts the clone and its kit.yml declares a kit version, so a clone without the kit gets no `.kit/`. In the dialog,
    s, Share with team, moves one of the person's own there, refused where the team has its slug already, and dim, as
    it does nothing, where the clone keeps no team's; c, Copy to mine, copies one of the team's into the person's,
    named copy after it; Save, r, and d on one of the team's change its file. Each is a change in the working tree,
    which kit commit carries. Each stage
    opens on its panel as the person last left it, unsaved changes and all, which the presets file keeps as last, a
    stage's own each time the panel changes: set as the view it was, its name struck through where the panel has changed
    from it, or as the stage's opening view where that one is gone; a preset named on the command line wins. What was
    done is said in place of the keys, for five seconds or until a key; why something was not done stays until a key.
  - The columns dialog, in `columns.go`, titled by the stage's fields, opens from 📇 Fields on the control line, the
    header drawing no icon of its own: every field a line, those the rows group by first, then the columns as they
    stand, then the hidden, a dim rule between each two of those it has, each with whether it shows, its sort and its
    place among the keys, and its group's depth.
    What it sets applies at once: space shows or hides, s turns the sort, g groups, ⇧↑↓ moves a field, into the groups
    or out, and → or ⏎ gives the card the keyboard, which ← gives back; a click on a line's box, sort, or group does as
    space, s, or g, and one on its label gives its card the keyboard. At the list's right stands the card of the field
    under the keyboard, as wide as the widest field's, so the dialog stands still as the keyboard moves.
  - Maximize, first in the tab's menu, gives the rows the bar's lines but one, for the session: the title names the
    tab, its count, and its :, and a click anywhere on them drops the tab's menu as the bar's does; a line in the tab's
    color runs under it, and the header follows; Restore, in its place, puts the bar back as it was, and a tab's
    number selects a tab either way.
  - A link to another app leaves a crumb, last on the title line of the app it opens, after the workspace menu and a
    maximized tab: ↩ and the app and tab it was followed from. A plain click on it, or ⌫, goes back there, the view put
    back as `Spot` kept it, with that place's own crumb, so crumbs chain, each one deep; the root, in
    `internal/screens/crumb.go`, keeps the chain for the session. Only the app menu shows an app with no crumb.
  - A menu line's key stands at its right, dim, a letter in upper case, R as ⌃A, as a macOS menu draws it, and the
    letter alone does the same with the menu closed as choosing the line does with it open: r Refresh, w Watch, m
    Maximize or Restore, o Open Selected, ⌃A Select All, and on issues n New… and a Assign to Me. With a menu open, a
    line's key chooses it before a letter moves to the next line starting with it.
    [ and ] move the cursor to the group heading before and after it, { and } to the first and the last, as they move
    between the palette's groups.
  - The search stands at the menu bar's right end, every screen's, and ⏎ or esc there gives the rows focus. No sidebar:
    the rows take the screen's width. While the person groups by no column, a value every row shares moves into one
    heading over them, the column grouping on its own; a group the person chooses overrides that, and the rows then
    group as chosen alone. Auto-group, which turns that on or off, Auto-hide, which hides a column no row has a value
    in, and Repeated emoji, which shows an emoji every row has instead of dropping it, are kittui's settings, in
    = Settings…, every view's alike. The Settings dialog opens over the menu, which stays open, its own Settings…
    lying over the menu's line; esc closes it back to the menu.
  - The legend, in `legend.go`, floats over the rows while F1 or ? shows it: what each glyph on the screen under it
    means, the menu bar's, the tab bar's, the control line's, the rule's, the rows', and the status bar's, in the order
    each first appears, reading from the top left, and nothing of a glyph the screen does not show, each glyph a click
    leads somewhere from marked. A glyph means what the line it stands on says: a plain · between items, in the menu bar
    or a heading, is not explained, only the · on the rule, a column that can sort and is not sorted; ▼ is the presets'
    menu on the control line and says it is not a sort, a sort being a raised ¹ or a lowered ₁ and never a triangle. A
    legend taller than the room under the tab bar stands in columns. It opens in the bottom right corner; the arrows on
    its top edge, ↖ ↗ ↙ ↘, move it to another, the one it stands in dim. It takes no other key or click, so the rows
    under it go on as they were.
  - A field's card, in `popover.go`, holds every setting of one field. A click on a column's name opens it under the
    name, its title over the name, as ='s and :'s menus hang around theirs, and a second click on the name closes it:
    its sort, a radio, Ascending, Descending, or Unsorted, which takes it out of the keys, as a shift-click on its sort
    does; its rows, ☐ Group, which groups the rows by it, innermost, or no longer, its depth after it while it groups,
    and, for a list, One row per assignee, which flattens it, or All assignees on one row, which gathers it again; its
    filter, the checklist of values it lets through or holds back, for an enum, a person, a label, a time, or a number;
    its rendering; and ☑ Show. What it sets applies at once, and the card stays open; a section the field cannot use is
    left out. A card opened from a name or a chip has all ▸ at the right of its top edge, which a click, or ⇥, widens
    into the columns dialog, on its field. Beside the dialog's list the card leaves out the sort, the grouping, and
    Show, which the list's line shows already. Flattened, a row stands once per value it holds, and once where it holds
    none, the header names one value, and the column sorts; a preset keeps it flat. Grouping by a list places a row
    under each of its values as flattening does, so a list the rows group by sorts its groups. Every count, the tab's
    and each heading's, counts a row once, however many copies of it stand. A card opened from a column's name, in the
    header or the heading over the groups, takes shift with ← or →, which moves the column a place among those the
    header names, or a depth among the groups, the card with it.
    Sorting always adds to the keys: a column that did not sort comes after the others, and one that did turns in its
    place; grouping always adds the column innermost, and the columns dialog orders both. The groups order the rows
    before any key, the outermost 1, so a column's sort is its place after them, always a small number, raised for
    ascending and lowered for descending, as each points, ¹ and ₁ for the first, ² and ₂ for the second, as
    grid.DirectionMark gives it to the rule, the columns dialog, the card and the legend, never a full-size ▲ or ▼; the
    card's lines carry ¹ and ₁ alone, ⁿ and ₙ where other things order the rows. The line under the keyboard in a menu
    or a dialog stands on the menu's bright cyan, in black, and a dialog's keys are the keys bar's alone. A column's
    sort mark stands on the rule under the first cell of its name, wherever the name aligns. A dim · there says the
    column can sort and does not; a column that cannot sort has the rule alone.
    A list has its mark under that first cell, what a click on it does: ×, dim, while a row holds all its values, which
    flattens it, and ÷ once flattened, which gathers it. Its sort's mark stands after it, which sorts by size while the
    list is whole.
    Every column's card stands in one order: its sort, by size for a list a row holds whole, then its rows, grouping
    before flattening, then its filter, its rendering, and Show.
    A plain click on the rule under the name turns the sort a step: ascending, after the other keys; descending, in its
    place; then off on the last key, and ascending again on any other, so a turn never moves the keys after it. A
    shift-click turns it off wherever it stands, as Unsorted on the card does. A terminal such as iTerm keeps ⌥, ⌃, and
    the right button with a click for its own selection and menu, and others keep shift, so a plain click and the card
    are the ways that work everywhere. Where the terminal sends them, ⌥ with a click turns a sort, ⌃ groups or ungroups,
    and the right button opens the card too. The presets' menu hangs around the preset line's selector the same way.
  - A field whose rendering is auto drops a glyph every row shown shares, and, while the columns are short of width,
    steps down to its next, shorter rendering, from the rightmost column leftward. A glyph that links, to a page or to
    another stage's row, is a button, so it stays, and a field drawn without its glyph links from nothing. The title
    never steps, and a rendering pinned always wins.
  - The rows are an outline, in `outline.go`: a group's heading, a row, and a header of more than one line each open and
    close by its ▾ or ▸, and a click opens or closes one only there; a click elsewhere on it puts the cursor on it, so
    space selects it without folding anything. One map per stage holds which stand open, a heading open and a row closed
    until changed. Two primitives fold it. Show and Hide open or close one node and nothing under it, so Hide then
    Show puts back exactly what showed: → shows the node under the cursor, or steps into an open heading's first row;
    ← hides it, or steps out to the heading over it; a click on its ▸ or ▾ does either. Where the heading over the
    groups folds a level, Show on one heading of it shows that heading's alone, the others hidden still. More and Less
    work a level at a time: More shows the shallowest level where something hides, every node of it, and hides
    nothing, so what stood open under them comes back as it stood; Less hides the deepest level open, down to the
    outermost headings. > and < do them over the whole table, as the tab's menu's Show More and Show Less do; ⌥ with →
    or ←, or with a click on a ▸ or ▾, does them under the one node, where the terminal sends ⌥. The key hints name ⌥
    with an arrow only where the terminal answered the kitty keyboard protocol: a macOS terminal with its default
    settings, iTerm among them, sends it as another key, or as the arrow alone, which is why > and < stand for it. The
    tab's menu has Show Selected, which shows the selected rows and every heading over them, and Hide Selected, which
    hides the selected rows and nothing over them, both dim while none are selected. Space on a heading selects every
    row of its group, or none once all are, and the heading's mark, ▣ or ◩, says whether all are or some. What stands
    open lasts the session; a preset does not keep it. Grouped rows stand under one heading that names the grouped
    columns, `▾ source ▾ assignees ▾ 3`, at the control line's left, apart from the rows, a blank line under it, and a
    heading per value under the line. A heading whose one heading under it heads the same rows takes it onto its line,
    across, `▾ jeffrey-aguilera · github · #2 53`, and so on down: one node, with one fold, before its first value,
    which opens and closes it, and a dim · before each value after it; what stands under the line steps in once from it,
    not once a level it took. In the heading over the groups, a level whose every level over it has one value, its
    headings frozen over the header, folds as the first does, so a dim · stands in its fold's place, and where every
    level has one value the rows' fold is not drawn: `▾ source · assignees ▾ 3`, `▾ source 21`. Otherwise the ▾ or ▸
    before each name opens or closes that level, and the one after the last name the rows under the groups: a level
    closed hides its headings and everything under them, and the levels under it show ▸ too; nothing else changes, so
    each level and each heading keeps its own state, and opening a level again shows what was under it as it was left.
    With the first level closed, nothing shows. A ▸ opens its level and every level over it; a value's heading shown
    shows what it holds alone, as Show does. A click on a column's name there drops the column's own menu, in
    `heading.go`, as a name in the header does. A field of several values, as assignees or labels, groups too: a row
    stands under each of its values, the same row, its selection one.
  - The column names and the rule under them, with each sort's mark, start at the first column. They stand under the
    headings of the outermost levels that have one value alone, the rows' context, and over the first item of the
    first level of more than one value, or over the first row, a blank line apart from those headings, as from the
    control line. Those headings and the header are frozen, as a pane freezes in a spreadsheet: everything under them
    scrolls, and they stay. While no row shows, every one folded away, there is no header. A stage with no control
    line says how many rows are selected at the rule's right end. The band under the cursor runs the pane's width.
  - Auto-group, a setting, groups a view's rows as the view loads, as the screen opens or the preset line sets it,
    when the view groups by nothing: by each groupable column shown whose value is the same on every row, left to
    right. From then on they are groups like any other, to move, ungroup, or save; a reload changes nothing, and
    the preset line still names the view.
  - Auto-hide, a setting, keeps a column no row loaded holds a value in from showing, its header with it: a header
    over nothing says nothing. It places nothing, so the view, the columns dialog, and a preset saved are as they
    were, and the column shows again once a load gives a row a value in it. Two columns always stay, as a row of one
    cell reads as a heading.
  - An open row breaks a value too wide for its column across lines, and one a field draws with a glyph hangs: the
    lines after its first start under the text, not under the glyph.
- `internal/welcome/` draws `kit welcome`: one tab per page under `.kit/welcome/`. A page of `##` sections can stand
  as a table, a `Sheet` of `internal/table`'s, as the flow picture's stages stand: `sections.go` reads a page's
  sections, each its heading, its text, and its lines of one backticked command, and draws the step's table under the
  text before the first heading; each step drawn so is a catalog, its fields and presets, and the few lines that make
  its records. A page with no such heading, as a house package's may be, is drawn as a page. The terms' page, in
  `terms.go`, has its fields in `terms.json`: each `## <topic>` heading is a topic, and each `### <term>, <term>`
  heading under it a row, its columns its command, what its first example is run with, its tool and its verb, as
  `kit flow`; its terms, the heading's comma-separated names, a list drawn as pills, as an issue's labels are; its
  meaning, the paragraphs under it as one line; and its examples, each line of one backticked command, a list too. A
  page with no `###` heading has a row to a `##` heading, under one topic, its title. Default groups the rows by
  topic, and stands both in the page's order, the topics by the catalog's `"order": "given"`; By command flattens the
  terms, a row a term, sorted, and groups them by command; Compact flattens them too, and hides the command and the
  examples. The command, the terms, and the examples keep their whole width, `"width": "whole"` in the catalog, and
  the meaning gives way first. The customize page, in `customize.go`, has its fields in `customize.json`: each
  `## <part>` heading is a row, its columns its part, the heading, a title; its settings, the `kit.yml` keys its text
  names, each backticked with its colon, as `packages:`, a list drawn as pills; what it is about, its paragraphs and
  list items as one line; and its commands. Its command, what its first command is run with, is hidden, and so is its
  order, the section's place on the page. Default stands the rows in the page's order; By command groups them by
  command; Compact hides the settings and the commands. The text before the first heading stands over the table,
  where the pane has room for it.
- `internal/tutorials/` draws `kit tutorials`: a card for each tutorial, one tab for each audience.
- `internal/tabs/` is the tab bar `kit flow`, `kit welcome`, and `kit tutorials` share: three lines of numbered tabs,
  and the keys and the clicks that select them.
- `internal/table/` is what a table is made of, whatever its rows hold: the catalog's fields and presets, read from
  a screen's JSON; the values a row holds; the renderers that draw a field down its column, registered by name; the
  automatic choice among a field's renderings; a preset set on a panel, and read back from it; a table's views, and
  the rules of which it is set as and what the person's can be saved, renamed, and deleted as; the file the person's
  presets are kept in; and a table's state, as a screen keeps one for each table. A table's controls take a `Host`,
  which a screen implements once, for the table selected: the field's card, the columns dialog, the
  outline, the heading over the groups, the preset line and its menu, the pane's layout, the cursor's moves, and the
  selection. It also builds a table's rows from its records, draws their lines, the header, the rule with each sort's
  mark, and the control line, names the keys of the rows, the preset line, and the heading over the groups, takes
  their keys and clicks, and gives a screen the table's layers in their order. `flow` is the host of each of its
  stages' tables. A `Sheet`, in `sheet.go`, is one table on a screen, whole, the host of its own controls: a screen
  gives it a catalog's table and its records, places it in a pane, hands it the keys and the clicks the pane takes,
  and draws its lines and its layers; it keeps the person's presets of it in the presets file under its catalog's
  name for it, beside every other table's. Each screen saves a table's presets with `KeepPresets`, which reads the
  file again first, so what another screen saved since stays.
- `internal/kit/` runs the `kit` CLI for the other packages.
- `internal/source/` reads kit for a screen. A load is a read the screen shows, as each of `kit flow`'s stages is: a
  read started again supersedes and ends the one before, the copy kit kept stands in, stale, while the first runs,
  and the marks say when what it shows has changed, a watched load read again at once. A stale tab has a yellow • after
  its name, on the bar or a maximized tab's title, and a plain click on it reads the stage again, as its menu's Refresh
  does. A poll is a read asked again a
  while after each answer, as `kit flow` reads the rate limits and the stages' marks each minute. Every wait goes
  through a clock: the program's is a `tea.Tick`, and a test's fires none until the test says.
- `internal/transcript/` writes the command line that `kittui` ran into the scrollback.
- `internal/shell/` is what every screen shares: the menu bar, with the kit menu, the workspace menu, the crumb, and
  the search field, laid out alike on every screen, a verb's form included; the Settings dialog and kittui's settings
  file; and the key hints on the status bar. A screen never writes its key hints. It gives the path from the screen to
  the control that has the keyboard, and each control on the path names the keys it answers. A deeper control hides an
  ancestor's key for a press it takes. The bar draws each control's keys between rules. When the line is too narrow,
  it keeps the screen's keys and drops the controls between them and the one with the keyboard, then clips that one's
  end.
- `internal/style/` is the look: one accent color, the glyphs, the keys bar's styles, and the pill a label or an
  audience is drawn as, its text on its color between half circles. It honors `NO_COLOR`.
- `internal/testkit/` holds what the tests share, the test's clock among them. Only tests import it.

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

`internal/syntax/testdata/*.json` are copies of what `kit syntax <verb>` prints, one for every verb, and
`internal/form/testdata/*.golden` the form drawn from each. `tests/syntax.bats` fails when one is stale or a verb has
none, and `bin/contracts` at the root writes them afresh; see [testing.md](../../docs/testing.md#contracts).

Tests that run kit run a fake one, a `testdata/fake-kit` script, never the real one. The flow legend tests read the
real [kit/bin/kit-flow](../../kit/bin/kit-flow), and each app verb's header, as kit-issues, so the picture and the
help agree on every glyph.
