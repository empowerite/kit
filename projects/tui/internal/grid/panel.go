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
// says. Fill is the cells the control under the keyboard is drawn inverse across, at least.
type Panel struct {
	Query     string
	Columns   []string
	Hidden    map[string]bool
	Order     []string
	Sort      []SortKey
	Sortable  map[string]bool
	Group     []string
	Groupable map[string]bool
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

// group gives the rows under a heading per value of each column the panel groups by, the outermost first, the groups
// in their values' order and the rows in each as they came: a heading, an aside, is the column's name and the value,
// dim, indented by its depth, and how many rows it heads. A row groups by a cell's first value, or under none when the
// cell has none. A row of one cell, as a heading, has no value: those follow the groups.
func (p Panel) group(rows []Row, columns []string) []Row {
	var at []int
	for _, name := range p.Group {
		if j := slices.Index(columns, name); j >= 0 {
			at = append(at, j)
		}
	}
	if len(at) == 0 {
		return rows
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
	slices.SortStableFunc(grouped, func(a, b Row) int {
		for _, j := range at {
			if c := natural(value(a, j), value(b, j)); c != 0 {
				return c
			}
		}
		return 0
	})
	var out []Row
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
			heading := strings.Repeat("  ", depth) + columns[j] + " · " + value(r, j) + "  " + strconv.Itoa(n)
			out = append(out, Row{Key: key, Cells: []Cell{{Values: []string{style.Off(heading)}, Wrap: true}}, Aside: true})
		}
		out = append(out, r)
	}
	return append(out, rest...)
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
	column string
	show   Show
}

// controls gives the panel's controls, in order.
func (p Panel) controls() []control {
	out := []control{{search: true}}
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
	for k, c := range p.controls() {
		var line string
		switch {
		case c.search:
			line = shell.SearchLine(p.Query, p.Focused && p.Cursor == k)
		case c.column != "":
			if k == 1 {
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
				if len(p.Group) > 1 {
					mark += subscript(k + 1)
				}
				line += " " + style.Tint(mark)
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
	}
	return lines, at
}

// Update takes a key while the panel has the keyboard, and says whether it changed anything: up and down move between
// the controls, stopping at the ends; on the search, a printable key is typed, / too, and backspace takes one back;
// off it, / goes to the search, and space and enter toggle a checkbox and pick a choice of Show; on a column's
// checkbox, shift with up or down moves the column a place, the keyboard with it, among the columns alone, s sorts by
// the column alone and S adds it to the sort, as Sorted says, and g groups by it alone and G within the groups, as
// Grouped says.
func (p Panel) Update(k tea.KeyPressMsg) (Panel, bool) {
	controls := p.controls()
	p.Cursor = min(max(p.Cursor, 0), len(controls)-1)
	c := controls[p.Cursor]
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
		p = p.Grouped(c.column, k.Text == "G")
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

// Click takes a click on a line of the panel, as Lines numbers them, and says whether it fell on a control: the
// control is then under the keyboard, and a checkbox or a choice is acted on as space does.
func (p Panel) Click(line int) (Panel, bool) {
	_, at := p.Lines()
	if line < 0 || line >= len(at) || at[line] < 0 {
		return p, false
	}
	p.Cursor = at[line]
	if c := p.controls()[p.Cursor]; !c.search {
		p = p.act(c)
	}
	return p, true
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
func (p Panel) act(c control) Panel {
	if c.column != "" {
		hidden := maps.Clone(p.Hidden)
		if hidden == nil {
			hidden = map[string]bool{}
		}
		hidden[c.column] = !hidden[c.column]
		p.Hidden = hidden
		return p
	}
	if c.show == All || p.Me != "" {
		p.Show = c.show
	}
	return p
}

// Width gives the cells the panel's widest line takes.
func (p Panel) Width() int {
	lines, _ := p.Lines()
	widest := 0
	for _, l := range lines {
		widest = max(widest, ansi.StringWidth(l))
	}
	return widest
}

// Filter gives the rows the panel lets through: those whose text has the search, without regard to case, and that
// belong to the caller or not, as Show says, sorted by its keys, each with its cells in its order and without the
// hidden columns'. columns names each cell's column, in the stage's order. While the search or Show narrows the rows,
// an aside that heads none of them is left out with them.
func (p Panel) Filter(rows []Row, columns []string) []Row {
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
	out = p.group(p.order(out, columns), columns)
	for k, r := range out {
		if !r.Aside {
			out[k] = p.hide(r, columns)
		}
	}
	return out
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
