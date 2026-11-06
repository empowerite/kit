// The panel's controls, and the rows they let through.
package grid

import (
	"maps"
	"slices"
	"strconv"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/empowerite/kit/projects/tui/internal/pane"
	"github.com/empowerite/kit/projects/tui/internal/shell"
	"github.com/empowerite/kit/projects/tui/internal/style"
)

// Show is which rows a panel lets through by whom they belong to: every one, the caller's, or everyone else's.
type Show int

// The choices of Show, in the panel's order.
const (
	All Show = iota
	Mine
	Others
)

// shows names each choice of Show, as the panel draws it.
var shows = []string{"all", "mine", "others"}

// The glyphs the panel draws: a column the rows are grouped by, the search, a checkbox off and on, and a radio button
// off and on. Each is one cell wide in every terminal, as Unicode fixes its width, but the search, which is two.
const (
	Grouping = "▤"
	Search   = shell.Find
	Unticked = "☐"
	Ticked   = "☑"
	Unpicked = "⦾"
	Picked   = "⦿"
)

// Panel is the controls beside a grid's rows, top to bottom: the search field; a checkbox per column that has a name;
// and, when the rows belong to people, a radio button per choice of Show. Mine and others need the caller's login,
// Me; while it is unknown they are dim, and cannot be chosen. Cursor is the control under the keyboard, Focused
// whether the panel has it. Hidden holds the names of the columns hidden; the panel never changes the map it was
// given, but gives a new one. Order is the order the named columns stand in, the rows' and the panel's checkboxes
// alike, as Arrange says; none is the stage's own. Sort is the columns the rows sort by, the first first, and Group
// the columns they group by, the outermost first; Sortable and Groupable name the columns that can, which the stage
// says. Beside the groups the person chose, the rows group on their own by each groupable column whose value is the
// same on every row of a group, as group says; Auto names the columns that do so now, as Filtered gave them, AutoAt
// the level of the headings each is said at, from 1, as Said gives it, and Kept
// those the person keeps as columns, which never do, a new map as Hidden is. Recall says whether there are searches to
// recall, which the search, with the keyboard, offers with its Recall glyph. Views says whether the panel offers a
// dropdown of named views under its search: View is the name it shows, none once the panel differs from its view,
// Naming whether the person is typing a new one, Changed whether the panel differs from the view or the name was
// changed, which shows Save, and Savable whether there is a name to save it under, without which Save is dim. Closed
// holds the keys of the groups' headings whose rows are folded away under them, as Fold says, a new map as Hidden is.
// Fill is the cells the control under the keyboard is drawn inverse across, at least.
type Panel struct {
	Query     string
	Columns   []string
	Hidden    map[string]bool
	Order     []string
	Sort      []SortKey
	Sortable  map[string]bool
	Group     []string
	Groupable map[string]bool
	Auto      []string
	AutoAt    map[string]int
	Recall    bool
	Views     bool
	View      string
	Naming    bool
	Changed   bool
	Savable   bool
	Closed    map[string]bool
	Kept      map[string]bool
	Fill      int
	Show      Show
	Owned     bool
	Me        string
	Cursor    int
	Focused   bool
}

// SortKey is a column the rows sort by, and whether the larger come first.
type SortKey struct {
	Column string
	Desc   bool
}

// Sorted gives the panel with the column sorting the rows alone, as a click on its name asks, or, with also, as one
// more key after the others, as a shift-click does: a column not sorting sorts ascending, then descending, then not at
// all. A column that cannot sort changes nothing.
func (p Panel) Sorted(column string, also bool) Panel {
	if !p.Sortable[column] {
		return p
	}
	k := slices.IndexFunc(p.Sort, func(s SortKey) bool { return s.Column == column })
	keys := slices.Clone(p.Sort)
	if !also {
		keys = nil
		if k >= 0 {
			keys = []SortKey{p.Sort[k]}
		}
		k = min(k, 0)
	}
	switch {
	case k < 0:
		keys = append(keys, SortKey{Column: column})
	case !keys[k].Desc:
		keys[k].Desc = true
	default:
		keys = slices.Delete(keys, k, k+1)
	}
	p.Sort = keys
	return p
}

// Mark gives what stands after a column's name where it sorts the rows: ▲ ascending or ▼ descending, and its place
// among the keys, as a subscript, when there are several; nothing where it does not.
func (p Panel) Mark(column string) string {
	for k, s := range p.Sort {
		if s.Column != column {
			continue
		}
		mark := "▲"
		if s.Desc {
			mark = "▼"
		}
		if len(p.Sort) > 1 {
			mark += subscript(k + 1)
		}
		return mark
	}
	return ""
}

// grouping gives how many columns the rows group by: those the person chose, and those they group by on their own.
func (p Panel) grouping() int {
	n := len(p.Group)
	for _, c := range p.Auto {
		if !slices.Contains(p.Group, c) {
			n++
		}
	}
	return n
}

// subscript gives a number in subscript digits, as ₁₂, which mark a column's place among the keys or the groups.
func subscript(n int) string {
	return strings.Map(func(r rune) rune { return '₀' + r - '0' }, strconv.Itoa(n))
}

// Grouped gives the panel grouping the rows by the column alone, as g asks, or, with nest, as G asks, within the groups
// it has, innermost; a column it groups by already, it no longer does, alone or among the others. A column that cannot
// group changes nothing.
func (p Panel) Grouped(column string, nest bool) Panel {
	if !p.Groupable[column] {
		return p
	}
	k := slices.Index(p.Group, column)
	switch {
	case k >= 0 && nest:
		p.Group = slices.Delete(slices.Clone(p.Group), k, k+1)
	case k >= 0 && len(p.Group) == 1:
		p.Group = nil
	case nest:
		p.Group = append(slices.Clone(p.Group), column)
	default:
		p.Group = []string{column}
	}
	return p
}

// Keep gives the panel keeping the column as a column, so the rows no longer group by it on their own, for as long as
// the panel lasts.
func (p Panel) Keep(column string) Panel {
	kept := maps.Clone(p.Kept)
	if kept == nil {
		kept = map[string]bool{}
	}
	kept[column] = true
	p.Kept = kept
	p.Auto = slices.DeleteFunc(slices.Clone(p.Auto), func(c string) bool { return c == column })
	return p
}

// Levels gives how many levels of headings the rows stand under, as group draws them, which a screen indents them by:
// one a column the person groups by, and one the heading over every row, when the rows group on their own and the
// person groups by nothing, so the rows under it stand in as they do under a heading the person chose.
func Levels(rows []Row) int {
	n := 0
	for _, r := range rows {
		if Heads(r) {
			n = max(n, strings.Count(r.Key, "\x00"))
		}
	}
	return n
}

// Said gives the level of the headings, from 1, that say each column the rows group by on their own, the shallowest
// when it is said at several: the heading over every row is the first level, as is a heading of the outermost column
// the person chose.
func Said(rows []Row) map[string]int {
	out := map[string]int{}
	for _, r := range rows {
		if !Heads(r) {
			continue
		}
		level := max(1, strings.Count(r.Key, "\x00"))
		for _, c := range r.Says {
			if at, ok := out[c]; !ok || level < at {
				out[c] = level
			}
		}
	}
	return out
}

// overAll is the key of the heading over every row.
const overAll = "group\x00"

// Heads says whether a row is the heading of a group, as group draws one.
func Heads(r Row) bool { return r.Aside && strings.HasPrefix(r.Key, "group\x00") }

// Heading gives the column whose group a row heads, as group draws the headings, or nothing for a row that heads no
// group, or the heading over every row, which no column the person chose makes.
func (p Panel) Heading(r Row) string {
	if !Heads(r) {
		return ""
	}
	if depth := strings.Count(r.Key, "\x00") - 1; depth < len(p.Group) {
		return p.Group[depth]
	}
	return ""
}

// group gives the rows under a heading per value of each column the panel groups by, the outermost first, the groups
// in their values' order and the rows in each as they came: a heading, an aside, is the value, as the rows draw it,
// and the column's name, dim, indented by its depth, and how many rows it heads, dim. A row groups by a cell's first value, or under none when the
// cell has none. A row of one cell, as a heading, has no value: those follow the groups.
//
// Under each heading, the walk goes on along the columns that can group, in the panel's order: one whose value is the
// same on every row the heading heads, as their sort key says, one value on each and none blank, groups them on its
// own. Its value joins the heading, after a comma, and its cell is blank in those rows, so the value is said once; a
// heading under it says it no more. Without a column the person chose, the rows group on their own under one heading
// over every row, their values alone. A heading over one row is not worth it: a group of one row, or a stage of one,
// does not group on its own. A column hidden, or kept as a column, never does. It gives the columns that group on
// their own anywhere.
func (p Panel) group(rows []Row, columns []string) ([]Row, []string) {
	var at []int
	for _, name := range p.Group {
		if j := slices.Index(columns, name); j >= 0 {
			at = append(at, j)
		}
	}
	value := func(r Row, j int) string {
		if j < len(r.Cells) && r.Cells[j].key() != "" {
			return r.Cells[j].key()
		}
		return "none"
	}
	// same says whether two rows share their values down to a depth
	same := func(a, b Row, depth int) bool {
		for _, j := range at[:depth+1] {
			if value(a, j) != value(b, j) {
				return false
			}
		}
		return true
	}
	var grouped, rest []Row
	for _, r := range rows {
		if len(r.Cells) > 1 {
			grouped = append(grouped, r)
		} else {
			rest = append(rest, r)
		}
	}
	walk := p.walk(columns, at)
	var took []int
	if len(at) == 0 {
		took = constant(grouped, walk)
		if len(took) == 0 {
			return rows, nil
		}
		heading := style.Off(p.fold(overAll)) + " " + joined(grouped[0], columns, took) + "  " +
			style.Off(strconv.Itoa(len(grouped)))
		out := []Row{{Key: overAll, Cells: []Cell{{Values: []string{heading}, Wrap: true}}, Aside: true,
			Says: names(columns, took)}}
		for _, r := range grouped {
			out = append(out, blank(r, took))
		}
		return append(out, rest...), names(columns, took)
	}
	slices.SortStableFunc(grouped, func(a, b Row) int {
		for _, j := range at {
			if c := natural(value(a, j), value(b, j)); c != 0 {
				return c
			}
		}
		return 0
	})
	var out []Row
	// the columns each heading open at its depth groups on its own, which the headings under it leave out
	auto := make([][]int, len(at))
	for i, r := range grouped {
		for depth, j := range at {
			if i > 0 && same(grouped[i-1], r, depth) {
				continue
			}
			n, key := 0, "group"
			for _, s := range grouped[i:] {
				if !same(s, r, depth) {
					break
				}
				n++
			}
			for _, x := range at[:depth+1] {
				key += "\x00" + value(r, x)
			}
			auto[depth] = constant(grouped[i:i+n], slices.DeleteFunc(slices.Clone(walk), func(x int) bool {
				return slices.ContainsFunc(auto[:depth], func(above []int) bool { return slices.Contains(above, x) })
			}))
			took = append(took, auto[depth]...)
			heading := strings.Repeat("  ", depth) + style.Off(p.fold(key)) + " " + value(r, j) + " " +
				style.Off(columns[j])
			if len(auto[depth]) > 0 {
				heading += style.Off(",") + " " + joined(r, columns, auto[depth])
			}
			heading += "  " + style.Off(strconv.Itoa(n))
			out = append(out, Row{Key: key, Cells: []Cell{{Values: []string{heading}, Wrap: true}}, Aside: true,
				Says: names(columns, auto[depth])})
		}
		out = append(out, blank(r, slices.Concat(auto...)))
	}
	return append(out, rest...), names(columns, took)
}

// Unfolded and Folded are the glyphs before a group's heading: its rows shown under it, or folded away.
const (
	Unfolded = "▾"
	Folded   = "▸"
)

// fold gives the glyph before the heading of a key: Folded while its group is closed, Unfolded while it is open.
func (p Panel) fold(key string) string {
	if p.Closed[key] {
		return Folded
	}
	return Unfolded
}

// Close gives the panel with the group a heading's key names closed, its rows folded away under the heading, or open
// again.
func (p Panel) Close(key string, closed bool) Panel {
	c := maps.Clone(p.Closed)
	if c == nil {
		c = map[string]bool{}
	}
	if closed {
		c[key] = true
	} else {
		delete(c, key)
	}
	p.Closed = c
	return p
}

// Fold gives the rows as group draws them with every closed group folded away: under a closed heading, the rows and
// the headings deeper than it, up to the next heading as shallow, or the rows of one cell that follow the groups. The
// heading stays, its count the rows it hides, and holds them in its Folded, which lay the columns out still.
func (p Panel) Fold(rows []Row) []Row {
	if len(p.Closed) == 0 {
		return rows
	}
	var out []Row
	closed := 0 // the level of the closed heading the rows stand under, or none
	for _, r := range rows {
		level := 0
		if Heads(r) {
			level = strings.Count(r.Key, "\x00")
		}
		if closed > 0 && (level > closed || level == 0 && len(r.Cells) > 1) {
			out[len(out)-1].Folded = append(out[len(out)-1].Folded, r)
			continue
		}
		closed = 0
		out = append(out, r)
		if level > 0 && p.Closed[r.Key] {
			closed = level
		}
	}
	return out
}

// walk gives the columns the rows may group by on their own, as indexes into them, in the panel's order: each that can
// group and has a name, but those the person groups by, hides, or keeps as a column.
func (p Panel) walk(columns []string, chosen []int) []int {
	var out []int
	for _, j := range p.arrangement(columns) {
		name := columns[j]
		if name != "" && p.Groupable[name] && !p.Hidden[name] && !p.Kept[name] && !slices.Contains(chosen, j) {
			out = append(out, j)
		}
	}
	return out
}

// constant gives those of the columns whose value is the same on every one of the rows, as their sort key says, each
// cell holding one value and none blank; none for fewer than two rows.
func constant(rows []Row, columns []int) []int {
	if len(rows) < 2 {
		return nil
	}
	var out []int
	for _, j := range columns {
		one := func(r Row) bool { return j < len(r.Cells) && len(r.Cells[j].Values) == 1 && r.Cells[j].key() != "" }
		if !one(rows[0]) {
			continue
		}
		if !slices.ContainsFunc(rows[1:], func(r Row) bool { return !one(r) || r.Cells[j].key() != rows[0].Cells[j].key() }) {
			out = append(out, j)
		}
	}
	return out
}

// joined gives each column a row groups by on its own, at the indexes, as a heading says it: the row's value in it,
// without its style, and the column's name, as a heading of a column the person chose says them, and a comma between
// columns.
func joined(r Row, names []string, columns []int) string {
	var values []string
	for _, j := range columns {
		values = append(values, ansi.Strip(r.Cells[j].first())+" "+style.Off(names[j]))
	}
	return strings.Join(values, style.Off(",")+" ")
}

// blank gives the row with its cells in the columns blank, their values said by the heading over it.
func blank(r Row, columns []int) Row {
	if len(columns) == 0 {
		return r
	}
	r.Cells = slices.Clone(r.Cells)
	for _, j := range columns {
		r.Cells[j] = Cell{Right: r.Cells[j].Right}
	}
	return r
}

// names gives the columns' names, each once, in the stage's order.
func names(columns []string, at []int) []string {
	var out []string
	for j, name := range columns {
		if slices.Contains(at, j) {
			out = append(out, name)
		}
	}
	return out
}

// order sorts the rows by the panel's keys, each in turn, the rows alike on every key keeping their order; a row of
// one cell, as a heading, stays where it is, the rows around it sorting in the places they hold.
func (p Panel) order(rows []Row, columns []string) []Row {
	if len(p.Sort) == 0 {
		return rows
	}
	var places []int
	var sorted []Row
	for i, r := range rows {
		if len(r.Cells) > 1 {
			places, sorted = append(places, i), append(sorted, r)
		}
	}
	slices.SortStableFunc(sorted, func(a, b Row) int {
		for _, s := range p.Sort {
			j := slices.Index(columns, s.Column)
			if j < 0 || j >= len(a.Cells) || j >= len(b.Cells) {
				continue
			}
			if c := natural(a.Cells[j].key(), b.Cells[j].key()); c != 0 {
				if s.Desc {
					return -c
				}
				return c
			}
		}
		return 0
	})
	out := slices.Clone(rows)
	for k, i := range places {
		out[i] = sorted[k]
	}
	return out
}

// control is one of the panel's controls: the search, a column's checkbox, or a choice of Show.
type control struct {
	search bool
	view   bool
	column string
	show   Show
}

// controls gives the panel's controls, in order.
func (p Panel) controls() []control {
	out := []control{{search: true}}
	if p.Views {
		out = append(out, control{view: true})
	}
	for _, c := range p.Arrange(p.Columns) {
		if c != "" {
			out = append(out, control{column: c})
		}
	}
	if p.Owned {
		for s := range shows {
			out = append(out, control{show: Show(s)})
		}
	}
	return out
}

// Lines draws the panel's controls, one a line, the headings dim, the control under the keyboard inverse while the
// panel has it; what is selected, a column ticked and the choice of Show picked, is bold in the accent color, the
// others' boxes and buttons dim, and all of a choice that cannot be picked. Beside each line is the control it draws, or -1 for a heading or a gap.
func (p Panel) Lines() (lines []string, at []int) {
	add := func(line string, control int) {
		lines, at = append(lines, line), append(at, control)
	}
	controls := p.controls()
	for k, c := range controls {
		var line string
		switch {
		case c.search:
			line = shell.RecallLine(p.Query, p.Focused && p.Cursor == k, p.Recall)
		case c.view:
			add("", -1)
			add("", -1)
			line = p.viewLine(p.Focused && p.Cursor == k)
			add(line, k)
			continue
		case c.column != "":
			if controls[k-1].column == "" {
				add("", -1)
				add(style.Off("columns"), -1)
			}
			line = style.Mark(Ticked + " " + c.column)
			if p.Hidden[c.column] {
				line = style.Off(Unticked) + " " + c.column
			}
			if mark := p.Mark(c.column); mark != "" {
				line += " " + style.Tint(mark)
			}
			if k := slices.Index(p.Group, c.column); k >= 0 {
				mark := Grouping
				if p.grouping() > 1 {
					mark += subscript(k + 1)
				}
				line += " " + style.Tint(mark)
			} else if slices.Contains(p.Auto, c.column) {
				// a column the rows group by on their own, apart from one the person chose, numbered by the level of
				// the headings that say it
				mark := Grouping
				if at := p.AutoAt[c.column]; at > 0 && p.grouping() > 1 {
					mark += subscript(at)
				}
				line += " " + style.Off(mark)
			}
		default:
			if c.show == All {
				add("", -1)
				add(style.Off("show"), -1)
			}
			switch {
			case p.Show == c.show:
				line = style.Mark(Picked + " " + shows[c.show])
			case c.show != All && p.Me == "":
				line = style.Off(Unpicked + " " + shows[c.show])
			default:
				line = style.Off(Unpicked) + " " + shows[c.show]
			}
		}
		if p.Focused && p.Cursor == k && !c.search {
			line = style.Cursor(pane.Pad(ansi.Strip(line), p.Fill, false))
		}
		add(line, k)
		if p.Focused && c.column != "" && (k+1 == len(controls) || controls[k+1].column == "") {
			add("", -1)
			for _, h := range p.hints() {
				add(style.Off(h), -1)
			}
		}
	}
	return lines, at
}

// hints gives the dim lines under the columns, while the panel has the keyboard, that say what a column's keys do: the
// sort's and the groups' only where a column can sort or group, and the order's always.
func (p Panel) hints() []string {
	var out []string
	if len(p.Sortable) > 0 {
		out = append(out, "s/S  sort, then by")
	}
	if len(p.Groupable) > 0 {
		out = append(out, "g/G  group, within")
	}
	return append(out, "⇧↑↓  order, or drag")
}

// Update takes a key while the panel has the keyboard, and says whether it changed anything: up and down move between
// the controls, stopping at the ends; on the search, a printable key is typed, / too, and backspace takes one back;
// off it, / goes to the search, and space and enter toggle a checkbox and pick a choice of Show; on a column's
// checkbox, shift with up or down moves the column a place, the keyboard with it, among the columns alone, s sorts by
// the column alone and S adds it to the sort, as Sorted says, and g groups by it alone and G within the groups, as
// Grouped says; on a column the rows group by on their own, g or G keeps it a column instead, as Keep says.
func (p Panel) Update(k tea.KeyPressMsg) (Panel, bool) {
	controls := p.controls()
	p.Cursor = min(max(p.Cursor, 0), len(controls)-1)
	c := controls[p.Cursor]
	if key := k.String(); c.view && key != "up" && key != "down" {
		// the dropdown's keys are its screen's to take: it opens the views, names one, and saves it
		return p, false
	}
	switch key := k.String(); {
	case key == "up":
		p.Cursor = max(p.Cursor-1, 0)
	case key == "down":
		p.Cursor = min(p.Cursor+1, len(controls)-1)
	case c.search && (key == "backspace" || k.Text != ""):
		p.Query, _ = shell.Type(p.Query, k)
	case k.Text == "/":
		p.Cursor = 0
	case c.column != "" && (k.Text == "s" || k.Text == "S"):
		p = p.Sorted(c.column, k.Text == "S")
	case c.column != "" && (k.Text == "g" || k.Text == "G"):
		if slices.Contains(p.Auto, c.column) && !slices.Contains(p.Group, c.column) {
			p = p.Keep(c.column)
		} else {
			p = p.Grouped(c.column, k.Text == "G")
		}
	case c.column != "" && (key == "shift+up" || key == "shift+down"):
		step := 1
		if key == "shift+up" {
			step = -1
		}
		if next := p.Cursor + step; next >= 1 && next < len(controls) && controls[next].column != "" {
			p = p.Move(c.column, controls[next].column)
			p.Cursor = next
		}
	case !c.search && (key == "space" || key == "enter"):
		p = p.act(c)
	default:
		return p, false
	}
	return p, true
}

// Keys gives the keys bar's segment of the control under the keyboard: the search types, and recalls a search when
// there are some; a column's line shows or hides it with space or enter, sorts by it with s, groups by it with g where
// the column can group, or keeps it a column where the rows group by it on their own, and orders it among the others
// with shift and up or down; a choice of whose rows show is taken with space or enter.
// The views' dropdown takes keys its screen says.
func (p Panel) Keys(recalls bool) shell.Segment {
	controls := p.controls()
	switch c := controls[min(max(p.Cursor, 0), len(controls)-1)]; {
	case c.search:
		return shell.SearchKeys(recalls)
	case c.view:
		return shell.Segment{}
	case c.column != "":
		toggle := "hide"
		if p.Hidden[c.column] {
			toggle = "show"
		}
		s := shell.Segment{Bindings: []shell.Binding{shell.Bind("␣", toggle, "space", "enter"),
			shell.Bind("s", "sort", "s", "S")}}
		switch {
		case slices.Contains(p.Auto, c.column) && !slices.Contains(p.Group, c.column):
			s.Bindings = append(s.Bindings, shell.Bind("g", "ungroup", "g", "G"))
		case p.Groupable[c.column]:
			s.Bindings = append(s.Bindings, shell.Bind("g", "group", "g", "G"))
		}
		return s.With(shell.Bind("⇧↑↓", "order", "shift+up", "shift+down"))
	}
	return shell.Segment{Bindings: []shell.Binding{shell.Bind("␣", "choose", "space", "enter")}}
}

// Click takes a click on a line of the panel, as Lines numbers them, and says whether it fell on a control: the
// control is then under the keyboard, and a checkbox or a choice is acted on as space does.
func (p Panel) Click(line int) (Panel, bool) {
	_, at := p.Lines()
	if line < 0 || line >= len(at) || at[line] < 0 {
		return p, false
	}
	p.Cursor = at[line]
	if c := p.controls()[p.Cursor]; !c.search && !c.view {
		p = p.act(c)
	}
	return p, true
}

// Save is the glyph of the button at the views' dropdown's right while the panel differs from its view, SaveText the
// word after it, and Saving the two, as the button draws them.
const (
	Save     = "💾"
	SaveText = "Save"
	Saving   = Save + " " + SaveText
)

// Unnamed stands, dim, in place of the view's name while the dropdown names none: the panel differs from its view, and
// nothing is typed.
const Unnamed = "unnamed"

// viewLine draws the views' dropdown: the view's name, as typed while it is renamed, with a caret while the keyboard is
// on it, or Unnamed, dim, while it has none, and ▾; and the Saving button at the line's right end, at Fill's last
// cells, while the panel differs from the view or the name was changed, its word dim while there is no name to save
// it under: an emoji draws in its own colors, which no dimming changes. Under the keyboard, the line is inverse up to
// the button.
func (p Panel) viewLine(under bool) string {
	name := p.View
	switch {
	case under && p.Naming:
		name += "▏"
	case name == "":
		name = style.Off(Unnamed)
	}
	text := name + " ▾"
	button := ansi.StringWidth(Saving)
	if under {
		text = style.Cursor(pane.Pad(ansi.Strip(text), max(ansi.StringWidth(text), p.Fill-button-1), false))
	}
	if !p.Changed {
		return text
	}
	word := SaveText
	if !p.Savable {
		word = style.Off(word)
	}
	return pane.Pad(text, max(ansi.StringWidth(text)+1, p.Fill-button), false) + Save + " " + word
}

// OnView says whether the views' dropdown is under the keyboard.
func (p Panel) OnView() bool {
	controls := p.controls()
	return p.Cursor >= 0 && p.Cursor < len(controls) && controls[p.Cursor].view
}

// ViewAt says whether a line of the panel, as Lines numbers them, is the views' dropdown's.
func (p Panel) ViewAt(line int) bool {
	_, at := p.Lines()
	return line >= 0 && line < len(at) && at[line] >= 0 && p.controls()[at[line]].view
}

// ColumnAt gives the name of the column whose checkbox a line of the panel, as Lines numbers them, draws, or nothing.
func (p Panel) ColumnAt(line int) string {
	_, at := p.Lines()
	if line < 0 || line >= len(at) || at[line] < 0 {
		return ""
	}
	return p.controls()[at[line]].column
}

// Point puts the keyboard on the control a line of the panel draws, acting on nothing, and says whether one is there.
func (p Panel) Point(line int) (Panel, bool) {
	_, at := p.Lines()
	if line < 0 || line >= len(at) || at[line] < 0 {
		return p, false
	}
	p.Cursor = at[line]
	return p, true
}

// Find puts the search under the keyboard.
func (p Panel) Find() Panel {
	p.Cursor, p.Focused = 0, true
	return p
}

// act toggles a column's checkbox, or picks a choice of Show; mine and others only while the caller's login is known.
// A column the person shows is kept a column, as Keep says.
func (p Panel) act(c control) Panel {
	if c.column != "" {
		hidden := maps.Clone(p.Hidden)
		if hidden == nil {
			hidden = map[string]bool{}
		}
		hidden[c.column] = !hidden[c.column]
		p.Hidden = hidden
		if !hidden[c.column] {
			p = p.Keep(c.column)
		}
		return p
	}
	if c.show == All || p.Me != "" {
		p.Show = c.show
	}
	return p
}

// Width gives the cells the panel's widest line takes, with the keyboard or without it.
func (p Panel) Width() int {
	p.Focused = true
	lines, _ := p.Lines()
	widest := 0
	for _, l := range lines {
		widest = max(widest, ansi.StringWidth(l))
	}
	return widest
}

// Filter gives the rows the panel lets through, as Filtered does.
func (p Panel) Filter(rows []Row, columns []string) []Row {
	out, _ := p.Filtered(rows, columns)
	return out
}

// Filtered gives the rows the panel lets through: those whose text has the search, without regard to case, and that
// belong to the caller or not, as Show says, sorted by its keys, grouped as group says, each with its cells in its
// order and without the hidden columns'. columns names each cell's column, in the stage's order. While the search or
// Show narrows the rows, an aside that heads none of them is left out with them. It gives the columns the rows group
// by on their own too, for the panel's Auto.
func (p Panel) Filtered(rows []Row, columns []string) ([]Row, []string) {
	query := strings.ToLower(p.Query)
	var out []Row
	for _, r := range rows {
		if r.Aside || p.matches(r, query) {
			out = append(out, r)
		}
	}
	if query != "" || p.Show != All {
		out = dropBare(out)
	}
	out, auto := p.group(p.order(out, columns), columns)
	for k, r := range out {
		if !r.Aside {
			out[k] = p.hide(r, columns)
		}
	}
	return out, auto
}

// dropBare leaves out each aside that heads no row: one the next aside, or the end, follows at once.
func dropBare(rows []Row) []Row {
	var out []Row
	for i, r := range rows {
		if r.Aside && (i+1 == len(rows) || rows[i+1].Aside) {
			continue
		}
		out = append(out, r)
	}
	return out
}

// matches says whether a row has the search in its text, and belongs as Show says.
func (p Panel) matches(r Row, query string) bool {
	switch {
	case p.Show == Mine && !owned(r, p.Me):
		return false
	case p.Show == Others && owned(r, p.Me):
		return false
	case query == "":
		return true
	}
	for _, line := range append([][]Cell{r.Cells}, r.Detail...) {
		for _, c := range line {
			for _, v := range c.Values {
				if strings.Contains(strings.ToLower(ansi.Strip(v)), query) {
					return true
				}
			}
		}
	}
	return false
}

// owned says whether the row belongs to the login.
func owned(r Row, login string) bool {
	for _, o := range r.Owners {
		if o == login {
			return true
		}
	}
	return false
}

// hide gives the row with its cells in the panel's order, without the hidden columns' cells; a row of one cell spans
// the columns, and keeps it.
func (p Panel) hide(r Row, columns []string) Row {
	if len(r.Cells) < 2 || len(p.Hidden) == 0 && len(p.Order) == 0 && len(p.Group) == 0 {
		return r
	}
	var cells []Cell
	for _, j := range p.arrangement(columns) {
		if j < len(r.Cells) && (j >= len(columns) || p.Shows(columns[j])) {
			cells = append(cells, r.Cells[j])
		}
	}
	for j := len(columns); j < len(r.Cells); j++ {
		cells = append(cells, r.Cells[j])
	}
	r.Cells = cells
	return r
}

// Shows says whether the rows show a column: one hidden does not, nor, while the rows are grouped by it, one whose
// value each group's heading names.
func (p Panel) Shows(column string) bool {
	return !p.Hidden[column] && !slices.Contains(p.Group, column)
}

// arrangement gives the order the columns stand in, as indexes into them: a column without a name keeps its place, and
// the named ones fill the other places, those in Order first, in its order, then the rest, in theirs.
func (p Panel) arrangement(columns []string) []int {
	var named []int
	for _, name := range p.Order {
		if j := slices.Index(columns, name); j >= 0 && name != "" && !slices.Contains(named, j) {
			named = append(named, j)
		}
	}
	for j, name := range columns {
		if name != "" && !slices.Contains(named, j) {
			named = append(named, j)
		}
	}
	out := make([]int, len(columns))
	for j, name := range columns {
		if name == "" {
			out[j] = j
			continue
		}
		out[j], named = named[0], named[1:]
	}
	return out
}

// Arrange gives the columns' names in the panel's order, as the rows' cells stand once it lets them through.
func (p Panel) Arrange(columns []string) []string {
	out := make([]string, len(columns))
	for k, j := range p.arrangement(columns) {
		out[k] = columns[j]
	}
	return out
}

// Move gives the panel with the column named moving to where the column named at stands, the columns between shifting
// one place toward where it was; the order then names every column the panel offers.
func (p Panel) Move(name, at string) Panel {
	order := slices.DeleteFunc(p.Arrange(p.Columns), func(c string) bool { return c == "" })
	from, to := slices.Index(order, name), slices.Index(order, at)
	if from < 0 || to < 0 || from == to {
		return p
	}
	order = slices.Delete(order, from, from+1)
	p.Order = slices.Insert(order, to, name)
	return p
}
