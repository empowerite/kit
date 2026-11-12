// The clones a walk found, as the Open… dialog shows them: a table, as every one of kittui's is, a sheet of the
// catalog's clones.json. Each repository is a heading, its desk and its linked worktrees under it, which its ▸ or ▾
// opens and closes, > and < a level at a time, as the table's outline does; whether a clone's repository is a member
// of the workspace shown is a column of its own. Space selects the clone under the cursor, or a heading every clone
// under it, ⇧↑↓ as the cursor moves, ⌃A every clone, and a plain click on a row's mark does as space does on it.
//
// Under the table stand three lines, each a plain click, dim while it cannot act: Add, + with the keyboard, drops down
// the groups kit keeps, and the one chosen takes the selected clones as members, or every clone found while none is
// selected; Remove, -, takes the selected out of the group shown; and Open, enter, opens the clone under the cursor.
// Each runs kit workspaces with the clones' folders, which kit reads as their repositories; kit's line, when it
// refuses, stands in the dialog, and what it did, when it did, with the selection cleared. Esc goes back to the
// folders.
package opener

import (
	_ "embed"
	"path/filepath"
	"slices"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/empowerite/kit/projects/tui/internal/grid"
	"github.com/empowerite/kit/projects/tui/internal/kit"
	"github.com/empowerite/kit/projects/tui/internal/shell"
	"github.com/empowerite/kit/projects/tui/internal/style"
	"github.com/empowerite/kit/projects/tui/internal/table"
)

//go:embed clones.json
var catalogJSON []byte

// clonesTable is the clones' table in the catalog.
const clonesTable = "opener/clones"

// catalog is clones.json, read; a catalog that does not read is a bug, which the package's tests catch.
var catalog, _ = table.ReadCatalog(catalogJSON)

// clones is the clones' table: its sheet; the walk it shows; the home folder, which writes a desk outside the folder
// walked; and where it was last placed, its pane's first screen line, its height, its width, and the screen's height.
type clones struct {
	*table.Sheet
	found                      ClonesMsg
	home                       string
	top, height, width, screen int
}

// Selects says the clones' rows are selected, as a table that acts on its selection does.
func (c *clones) Selects() bool { return true }

// newClones gives the table of the clones a walk found, its rows having the keyboard, each marked when its repository
// is a member of the workspace shown.
func newClones(found ClonesMsg, home string, ws kit.Workspace) *clones {
	s := table.NewSheet(catalog[clonesTable], "clones", "", "")
	c := &clones{Sheet: &s, found: found, home: home}
	c.mark(ws)
	return c
}

// mark gives the table its records again, each clone's repository marked when it is a member of the workspace shown,
// a group whose members' checkouts hold its desk.
func (c *clones) mark(ws kit.Workspace) {
	members := map[string]bool{}
	if ws.Kind == "group" {
		for _, m := range ws.Members {
			if m.Checkout != "" {
				members[real(m.Checkout)] = true
			}
		}
	}
	records := make([]table.Record, 0, len(c.found.Clones))
	for k, rel := range c.found.Clones {
		desk := c.found.Desks[k]
		member := table.Value{}
		if members[real(desk)] {
			member = table.One(Group)
		}
		records = append(records, table.Record{Row: grid.Row{Key: filepath.Join(c.found.Under, rel)},
			Values: map[string]table.Value{"repository": table.One(c.repository(desk)), "clone": table.One(rel),
				"member": member}})
	}
	c.SetRecords(records)
	if !c.InRows() {
		c.FocusRows()
	}
}

// real gives a folder as the disk names it, its links followed, or as it is where it cannot.
func real(path string) string {
	if p, err := filepath.EvalSymlinks(path); err == nil {
		return p
	}
	return filepath.Clean(path)
}

// repository gives the name a desk's heading draws: its folder from the one walked, the walked one's own name for
// itself, and one outside it whole, ~ in place of home.
func (c *clones) repository(desk string) string {
	rel, err := filepath.Rel(c.found.Under, desk)
	switch {
	case err != nil || rel == ".." || len(rel) > 2 && rel[:3] == "../":
		return tilde(desk, c.home)
	case rel == ".":
		return filepath.Base(c.found.Under)
	}
	return rel
}

// tableWidth gives the width the clones' table is drawn in on a screen so wide, which the dialog's box takes.
func (d Dialog) tableWidth(width int) int { return max(20, min(100, width-8)) }

// tableHeight gives the lines the clones' table takes over a screen of so many lines.
func tableHeight(height int) int { return max(4, min(22, height-12)) }

// fit places the table where the dialog's box draws it over a screen of so many lines, so wide: under the box's top
// edge and its head, as wide as tableWidth says, and as tall as the lines it draws, up to the most tableHeight lets;
// a table already there is left as it is.
func (c *clones) fit(d Dialog, height, width int) {
	most, w := tableHeight(height), d.tableWidth(width)
	place := func(h int) {
		lines := len(d.head()) + h + 1 + 3 + 2
		if c.found.Capped {
			lines++
		}
		top := boxTop(height, lines) + 1 + len(d.head())
		if top != c.top || h != c.height || w+1 != c.width || height != c.screen {
			c.top, c.height, c.width, c.screen = top, h, w+1, height
			c.Place(top, h, w+1, height)
		}
	}
	place(most)
	lines := c.View()
	for len(lines) > 1 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	place(len(lines))
}

// onMark says whether a column of the table, from its left, falls on the mark of a row: before a heading's ▾ or ▸, or
// before the gutter's cell of a row's.
func (c *clones) onMark(row grid.Row, x int) bool {
	if grid.Heads(row) {
		return x < table.FoldAt(row)
	}
	_, width := c.Pane()
	return x < width-c.GridWidth()+table.FoldCell
}

// paths gives the folders of the clones selected, each once; with none selected, every clone found.
func (c *clones) paths() []string {
	var out []string
	for _, r := range table.SelectedRows(c) {
		out = append(out, r.Key)
	}
	if len(out) > 0 {
		return out
	}
	for _, rel := range c.found.Clones {
		out = append(out, filepath.Join(c.found.Under, rel))
	}
	return out
}

// selected gives how many clones are selected.
func (c *clones) selected() int { return len(table.SelectedRows(c)) }

// under gives the folder of the clone under the cursor, nothing on a heading or with no row.
func (c *clones) under() string {
	rows, at := c.Rows(), c.Table().Cursor
	if !c.InRows() || at < 0 || at >= len(rows) || grid.Heads(rows[at]) || rows[at].Aside {
		return ""
	}
	return rows[at].Key
}

// groups gives the dropdown's lines: each group kit keeps, by name.
func (d Dialog) groups() []shell.Item {
	var out []shell.Item
	for _, g := range d.ws.Workspaces {
		out = append(out, shell.Item{Glyph: Group, Name: g.Name})
	}
	return out
}

// every says a line of the dropdown acts: every group does.
func every(int) bool { return true }

// canAdd says whether Add acts: there are clones and groups, and kit is not asked.
func (d Dialog) canAdd() bool { return !d.Busy && len(d.found.Clones) > 0 && len(d.ws.Workspaces) > 0 }

// canRemove says whether Remove acts: the workspace shown is a group, some clones are selected, and kit is not asked.
func (d Dialog) canRemove() bool { return !d.Busy && d.ws.Kind == "group" && d.table.selected() > 0 }

// canOpen says whether Open acts: a clone is under the cursor, and kit is not asked.
func (d Dialog) canOpen() bool { return !d.Busy && d.table.under() != "" }

// dropped gives the dialog with the groups dropped down from Add, when it acts.
func (d Dialog) dropped() Dialog {
	if d.canAdd() {
		d.adding = d.adding.Opened(d.groups(), every)
	}
	return d
}

// add gives the dialog asking kit to add the clones selected, or every one, to the group named.
func (d Dialog) add(group string) (Dialog, Ask) {
	return d.run(append([]string{"workspaces", "--add=" + group}, d.table.paths()...)...)
}

// remove gives the dialog asking kit to take the clones selected out of the group shown, when Remove acts.
func (d Dialog) remove() (Dialog, Ask) {
	if !d.canRemove() {
		return d, Ask{}
	}
	return d.run(append([]string{"workspaces", "--remove=" + d.ws.Name}, d.table.paths()...)...)
}

// open gives the dialog asking to open the clone under the cursor, when Open acts.
func (d Dialog) open() (Dialog, Ask) {
	if !d.canOpen() {
		return d, Ask{}
	}
	return d.askFor(d.table.under())
}

// cloneKey takes a key over the clones, and says whether it took it: the dropdown's, while it is open; the table's
// layers', while one is; esc goes back to the folders; space, ⇧↑↓, and ⌃A select; + drops the groups down, - removes,
// and enter on a clone opens it; the rest are the table's, as its rows and its control line take them. A key it does
// not take is the dialog's, as typing is.
func (d Dialog) cloneKey(k tea.KeyPressMsg) (Dialog, Ask, bool) {
	c := d.table
	if d.adding.Open {
		var chosen int
		if d.adding, chosen = d.adding.Key(k, d.groups(), every); chosen >= 0 {
			next, ask := d.add(d.groups()[chosen].Name)
			return next, ask, true
		}
		return d, Ask{}, true
	}
	if c.Layered() {
		c.Key(k)
		return d, Ask{}, true
	}
	d.Did = ""
	switch key := k.String(); key {
	case "esc":
		return d.typing(d.Typed), Ask{}, true
	case "space":
		table.SelectRow(c, false)
		return d, Ask{}, c.InRows()
	case "ctrl+a":
		table.SelectAll(c)
		return d, Ask{}, true
	case "shift+up", "shift+down":
		if c.InRows() {
			table.SelectRow(c, true)
			table.Down(c, map[string]int{"shift+up": -1, "shift+down": 1}[key])
			table.SelectRow(c, true)
		}
		return d, Ask{}, true
	case "+":
		return d.dropped(), Ask{}, true
	case "-":
		next, ask := d.remove()
		return next, ask, true
	case "enter":
		if c.under() != "" {
			next, ask := d.open()
			return next, ask, true
		}
		if rows, at := c.Rows(), c.Table().Cursor; c.InRows() && at < len(rows) && grid.Heads(rows[at]) {
			table.Disclose(c, !c.Table().Open.Unfolds(rows[at]))
			return d, Ask{}, true
		}
	}
	return d, Ask{}, c.Key(k)
}

// cloneKeys gives the keys bar's segment over the clones: the dropdown's while it is open; the table's layer's, or its
// control line's, while they have the keyboard; else the rows', esc going back to the folders, with > and < a level,
// and + adding, - removing, and enter opening.
func (d Dialog) cloneKeys() shell.Segment {
	c := d.table
	if d.adding.Open {
		return shell.Segment{Owns: true, Bindings: []shell.Binding{shell.Bind("esc", "close"),
			shell.Bind("↑↓", "move", "up", "down"), shell.Bind("⏎", "add", "enter")}}
	}
	if seg, layered := c.Keys(); layered || !c.InRows() {
		seg.Owns = true
		return seg
	}
	seg := table.RowKeys(c, false)
	seg.Owns, seg.Bindings[0] = true, shell.Bind("esc", "back")
	return seg.With(shell.Bind("><", "level", ">", "<"), shell.Bind("+", "add", "+"), shell.Bind("-", "remove", "-"),
		shell.Bind("⏎", "open", "enter"))
}

// cloneBody gives the dialog's lines over the clones, the table so wide: its head; the table's lines; a line that
// says the walk stopped at the most it finds, when it did; and Add, Remove, and Open, each dim while it cannot act.
func (d Dialog) cloneBody(int) []openLine {
	lines := d.head()
	for _, l := range d.table.View() {
		lines = append(lines, openLine{text: l, sheet: true})
	}
	if d.found.Capped {
		lines = append(lines, openLine{text: style.Off("stopped at the first " +
			count(len(d.found.Clones), "clone", "clones"))})
	}
	add := Adding + " Add All to a Workspace…"
	if n := d.table.selected(); n > 0 {
		add = Adding + " Add " + count(n, "Clone", "Clones") + " to a Workspace…"
	}
	remove := Removing + " Remove from the Workspace Shown"
	if d.ws.Kind == "group" {
		remove = Removing + " Remove from " + d.ws.Name
	}
	open := Opening + " Open"
	if under := d.table.under(); under != "" {
		rel, _ := filepath.Rel(d.under, under)
		open += " " + rel
	}
	dim := func(text string, acts bool) string {
		if acts {
			return text
		}
		return style.Off(text)
	}
	return append(lines, openLine{}, openLine{text: dim(add, d.canAdd()), adds: true},
		openLine{text: dim(remove, d.canRemove()), removes: true}, openLine{text: dim(open, d.canOpen()), opensAt: true})
}

// line gives the line of the body a line's test picks, -1 for none.
func line(body []openLine, is func(openLine) bool) int { return slices.IndexFunc(body, is) }

// addBox gives the groups' dropdown's lines, hung from Add's glyph in the dialog's box p over a screen so wide, and its
// top line and left column.
func (d Dialog) addBox(p shell.Placed, width int) (lines []string, top, left int) {
	body := d.body(ansi.StringWidth(p.Lines[0]) - 4)
	top = p.Top + 1 + line(body, func(l openLine) bool { return l.adds })
	lines, left = shell.Box(d.groups(), every, d.adding.At, Adding, p.Left+2, width)
	return lines, top, left
}

// cloneOver draws the table's open layers over the screen's lines, as the table places them from the box's inside,
// and the groups' dropdown while it is open.
func (d Dialog) cloneOver(lines []string, p shell.Placed, width int) []string {
	if d.table.Layered() {
		at := p.Left + 2
		inside := make([]string, len(lines))
		for k, l := range lines {
			inside[k] = ansi.Cut(l, at, width)
		}
		inside = d.table.Over(inside)
		for k, l := range lines {
			lines[k] = ansi.Cut(l, 0, at) + inside[k]
		}
	}
	if d.adding.Open {
		box, top, left := d.addBox(p, width)
		lines = shell.Over(lines, box, top, left)
	}
	return lines
}

// cloneClick takes a left click over the clones, in the dialog's box p over a screen of so many lines, so wide, and
// says whether it took it: the dropdown's, while it is open, on a group adding to it; the table's layers', while one
// is; one on a row's mark selects it, or a heading's every clone under it, as space does; the rest of the table's lines
// are the table's; and Add drops the groups down, Remove removes, and Open opens. A click it does not take is the
// dialog's, as one off the box is.
func (d Dialog) cloneClick(msg tea.MouseClickMsg, p shell.Placed, height, width int) (Dialog, Ask, bool) {
	c, x, y := d.table, msg.X, msg.Y
	local := msg
	local.X -= p.Left + 2
	if d.adding.Open {
		box, top, left := d.addBox(p, width)
		var chosen int
		if d.adding, chosen = d.adding.Click(x, y, top, left, ansi.StringWidth(box[0]), d.groups(), every); chosen >= 0 {
			next, ask := d.add(d.groups()[chosen].Name)
			return next, ask, true
		}
		return d, Ask{}, true
	}
	if c.Layered() {
		c.Click(local)
		return d, Ask{}, true
	}
	k, on := p.Hit(x, y)
	body := d.body(ansi.StringWidth(p.Lines[0]) - 4)
	if !on || k < 0 || k >= len(body) {
		return d, Ask{}, false
	}
	d.Did = ""
	switch l := body[k]; {
	case l.sheet:
		if row, _, _, ok := table.RowAt(c, local.X, y); ok && c.onMark(c.Rows()[row], local.X) {
			c.Table().Cursor = row
			c.EnterRows()
			table.SelectRow(c, false)
			return d, Ask{}, true
		}
		c.Click(local)
	case l.adds:
		return d.dropped(), Ask{}, true
	case l.removes:
		next, ask := d.remove()
		return next, ask, true
	case l.opensAt:
		next, ask := d.open()
		return next, ask, true
	default:
		return d, Ask{}, false
	}
	return d, Ask{}, true
}
