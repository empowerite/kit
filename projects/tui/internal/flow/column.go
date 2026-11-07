// The column menu: a click on a column's name in the header drops it under the name, as does a right click where the
// terminal sends one. Many terminals keep option, ctrl, and the right button with a click for their own selection and
// menu, so the menu holds what those would do: the column's sort, a radio that shows it, None, Ascending, or
// Descending, a column that did not sort coming after the keys there are, one that did keeping its place, and Clear
// Sort, which no column sorts after; the rows grouped by it too, innermost, or no longer; the column hidden; and the
// field's popover, which leaves the sort to the menu. The menu's title is the column's name, so no line names it again. Shift with left or
// right moves the column a place among those the header shows, the menu with it.
package flow

import (
	"slices"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/empowerite/kit/projects/tui/internal/grid"
	"github.com/empowerite/kit/projects/tui/internal/shell"
	"github.com/empowerite/kit/projects/tui/internal/style"
)

// columnMenu is the column menu while it is open: a dropdown, the column it acts on, and whether it hangs from the
// column's name in the heading over the groups rather than in the header.
type columnMenu struct {
	shell.Dropdown
	column  string
	heading bool
}

// colAct is what a line of the column menu does.
type colAct int

// What the column menu's lines do: none, the rules; the radio of the column's sort, None, Ascending and Descending;
// every column's sort cleared; the rows grouped by the column, innermost, or no longer; the column hidden; and More,
// the field's popover.
const (
	actRule colAct = iota
	actNone
	actUp
	actDown
	actClear
	actGroup
	actUngroup
	actHide
	actMore
)

// The words of the column menu's lines, and their glyphs: grouping's is the mark
// a column grouping the rows has, hiding's the box of a hidden column in the columns dialog.
const (
	ClearSort = "Clear Sort"
	GroupIt   = "Group"
	UngroupIt = "Ungroup"
	Hide      = "Hide"
	More      = "Options…"

	GroupGlyph   = grid.Grouping
	UngroupGlyph = "▢"
	HideGlyph    = grid.Unticked
	MoreGlyph    = "⋯"
)

// colLine is a line of the column menu: what it draws, and what it does.
type colLine struct {
	item shell.Item
	act  colAct
}

// columnLines gives the column menu's lines. The radio's choice picked is the column's sort, None where it does not
// sort. Where something else orders the rows, a group or another key, its triangles carry ⁿ and ₙ, as the header's
// mark carries the place the column takes, raised after ▲ and lowered after ▼. Clear Sort, as the tab menu's Clear Selection, sweeps. A line that cannot act on
// the column stays, dim.
func (m Model) columnLines() []colLine {
	p, column := m.panel(m.stage), m.colMenu.column
	d, up, down := p.Field(column).Sort, "", ""
	if place, total := p.Place(column); total-min(place, 1) > 0 {
		up, down = "ⁿ", "ₙ"
	}
	radio := func(picked bool, name string, act colAct) colLine {
		if picked {
			return colLine{shell.Item{Glyph: style.Tint(grid.Picked), Name: name}, act}
		}
		return colLine{shell.Item{Glyph: style.Off(grid.Unpicked), Name: name}, act}
	}
	return []colLine{radio(d == grid.Unsorted, "None", actNone), radio(d == grid.Ascending, "▲"+up+" Ascending", actUp),
		radio(d == grid.Descending, "▼"+down+" Descending", actDown), colLine{shell.Item{Glyph: Sweep, Name: ClearSort}, actClear},
		colLine{act: actRule},
		colLine{shell.Item{Glyph: GroupGlyph, Name: GroupIt}, actGroup},
		colLine{shell.Item{Glyph: UngroupGlyph, Name: UngroupIt}, actUngroup}, colLine{act: actRule},
		colLine{shell.Item{Glyph: HideGlyph, Name: Hide}, actHide}, colLine{shell.Item{Glyph: MoreGlyph, Name: More}, actMore}}
}

// columnItems gives the column menu's lines as the menu draws them.
func (m Model) columnItems() []shell.Item {
	var items []shell.Item
	for _, l := range m.columnLines() {
		items = append(items, l.item)
	}
	return items
}

// columnLine gives the index of the column menu's line that does act, -1 where none does.
func (m Model) columnLine(act colAct) int {
	return slices.IndexFunc(m.columnLines(), func(l colLine) bool { return l.act == act })
}

// columnEnabled says whether the column menu's line k acts: the radio on a column that sorts; Clear Sort while a column
// sorts the rows; Group on one that can group and does not yet, Ungroup on one that does; and Hide and Options…
// always; the rules never do.
func (m Model) columnEnabled(k int) bool {
	lines := m.columnLines()
	if k < 0 || k >= len(lines) {
		return false
	}
	p, column := m.panel(m.stage), m.colMenu.column
	grouped := slices.Contains(p.Groups(), column)
	switch lines[k].act {
	case actNone, actUp, actDown:
		return p.Sortable[column]
	case actClear:
		return len(p.ColumnKeys()) > 0
	case actGroup:
		return p.Groupable[column] && !grouped
	case actUngroup:
		return grouped
	case actHide, actMore:
		return true
	}
	return false
}

// shifted gives the panel with a column moved a place, left for a negative step, right for a positive one, among those
// it stands with: a column among the columns the header names, past those no row fills and those the rows group by on
// their own; a group among the groups, a
// depth outward or inward. It says whether it moved; at either end it does not, nor does it cross into the groups or the
// hidden.
func (m Model) shifted(column string, d int) (grid.Panel, bool) {
	p := m.panel(m.stage)
	placement := p.Field(column).Placement
	order := func(q grid.Panel) []string { return q.Groups() }
	switch placement {
	case grid.Column:
		// those the header names: not one the rows group by on their own, whose value their heading says
		shown := slices.DeleteFunc(slices.Clone(m.shownColumns(m.stage)), func(id string) bool {
			_, name := m.columnName(id)
			return name == ""
		})
		order = func(q grid.Panel) []string {
			return slices.DeleteFunc(q.Listed(), func(id string) bool {
				return q.Field(id).Placement != grid.Column || !slices.Contains(shown, id)
			})
		}
	case grid.Hidden:
		return p, false
	}
	before := order(p)
	for q := p; ; {
		var ok bool
		if q, ok = q.Step(column, d); !ok || q.Field(column).Placement != placement {
			return p, false
		}
		if !slices.Equal(order(q), before) {
			return q, true
		}
	}
}

// shiftStep gives the step shift with left or right asks for, and whether the key is one.
func shiftStep(key string) (int, bool) {
	switch key {
	case "shift+left":
		return -1, true
	case "shift+right":
		return 1, true
	}
	return 0, false
}

// shiftKeys gives the binding of shift with left and right on a menu that moves a column, each where it moves it.
func (m Model) shiftKeys(column string) []shell.Binding {
	var presses []string
	for _, key := range []string{"shift+left", "shift+right"} {
		if d, _ := shiftStep(key); func() bool { _, ok := m.shifted(column, d); return ok }() {
			presses = append(presses, key)
		}
	}
	if len(presses) == 0 {
		return nil
	}
	return []shell.Binding{shell.Bind("⇧←→", "move", presses...)}
}

// openColumnMenu drops the column menu down from a column's name, the first line that acts under the keyboard.
func (m *Model) openColumnMenu(column string) {
	m.colMenu = columnMenu{column: column}
	m.colMenu.Dropdown = m.colMenu.Opened(m.columnItems(), m.columnEnabled)
}

// openFieldMenu drops the column menu around a column's name in the heading over the groups, its row k, the cursor on
// the heading, the first line that acts under the keyboard.
func (m *Model) openFieldMenu(k int, column string) {
	m.stages[m.stage].cursor, m.inRows = k, true
	m.colMenu = columnMenu{column: column, heading: true}
	m.colMenu.Dropdown = m.colMenu.Opened(m.columnItems(), m.columnEnabled)
}

// fieldName gives the screen column and line a column's name stands on in the heading over the groups, as the pane is
// scrolled; a line of -1 once the heading, or the name in it, no longer shows.
func (m Model) fieldName(column string) (x, y int) {
	rows := m.rows(m.stage)
	k := slices.IndexFunc(rows, grid.OverGroups)
	if k < 0 {
		return 0, -1
	}
	for _, n := range m.headingNames(rows[k]) {
		if n.column == column {
			return n.from, m.rowLine(k)
		}
	}
	return 0, -1
}

// columnKey takes a key while the column menu is open: shift with left or right moves the column, the menu with it,
// the keyboard on the same line; the rest go to the dropdown, and the line chosen does what it says.
func (m Model) columnKey(k tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if key := k.String(); key == "shift+left" || key == "shift+right" {
		d, _ := shiftStep(key)
		if p, ok := m.shifted(m.colMenu.column, d); ok {
			m.setPanel(p)
		}
		return m, nil
	}
	d, chosen := m.colMenu.Key(k, m.columnItems(), m.columnEnabled)
	return m, m.columnChose(d, chosen)
}

// columnChose takes what the dropdown did: a line chosen does what it says; a dropdown closed without one closes the
// menu; one still open goes on.
func (m *Model) columnChose(d shell.Dropdown, chosen int) tea.Cmd {
	switch {
	case chosen >= 0:
		return m.columnAct(chosen)
	case !d.Open:
		m.colMenu = columnMenu{}
	default:
		m.colMenu.Dropdown = d
	}
	return nil
}

// columnAct closes the column menu and does what its line k says, when it acts.
func (m *Model) columnAct(k int) tea.Cmd {
	c, enabled := m.colMenu, m.columnEnabled(k)
	if !enabled {
		m.colMenu = columnMenu{}
		return nil
	}
	act := m.columnLines()[k].act
	m.colMenu = columnMenu{}
	p := m.panel(m.stage)
	switch act {
	case actNone, actUp, actDown:
		// a column that did not sort comes after the keys there are; one that did turns in its place
		p = p.SortedAs(c.column, map[colAct]grid.Direction{actNone: grid.Unsorted, actUp: grid.Ascending,
			actDown: grid.Descending}[act])
	case actClear:
		p = p.Unsorted()
	case actHide:
		p = p.Placed(c.column, grid.Hidden)
	case actGroup, actUngroup:
		// a column joins the groups innermost, or leaves them from wherever it stands
		p = p.Grouped(c.column, true)
	default:
		x, _ := m.columnName(c.column)
		y := m.top()
		if c.heading {
			x, y = m.fieldName(c.column)
		}
		return m.openPopover(c.column, x, y, true)
	}
	m.setPanel(p)
	return nil
}

// columnBox gives the column menu's lines, the box around the column's name, which its top edge draws where the header
// does, as the app's menu hangs from ≡, or where the heading over the groups does; and the screen line and column it
// is drawn from. Once the heading no longer shows, there are none.
func (m Model) columnBox() (lines []string, top, left int) {
	if m.colMenu.heading {
		x, y := m.fieldName(m.colMenu.column)
		if y < 0 {
			return nil, 0, 0
		}
		lines, left = shell.Box(m.columnItems(), m.columnEnabled, m.colMenu.At, m.colMenu.column, x, m.paneWidth())
		return lines, y, left
	}
	x, name := m.columnName(m.colMenu.column)
	lines, left = shell.Box(m.columnItems(), m.columnEnabled, m.colMenu.At, name, x, m.width)
	return lines, m.top(), left
}

// columnName gives the screen column a column's name starts on in the header, and its first line, as the header draws
// it.
func (m Model) columnName(column string) (int, string) {
	j := slices.Index(m.shownColumns(m.stage), column)
	x, name := grid.HeaderName(m.rows(m.stage), m.names(m.stage), m.gridWidth(m.stage), j)
	return x + m.indent(m.stage), name
}

// columnClick takes a click while the column menu is open, as a dropdown does: on a line that acts, it does what the
// line says; on another line of the box, nothing; anywhere else, on the name it hangs from too, it closes the menu,
// as it does when its box no longer shows.
func (m *Model) columnClick(x, y int) tea.Cmd {
	box, top, left := m.columnBox()
	if len(box) == 0 {
		m.colMenu = columnMenu{}
		return nil
	}
	d, chosen := m.colMenu.Click(x, y, top, left, ansi.StringWidth(box[0]), m.columnItems(), m.columnEnabled)
	return m.columnChose(d, chosen)
}

// headerColumn gives the column whose name a click at x, y in the header falls on, none off every name.
func (m Model) headerColumn(x, y int) string {
	lines, _ := m.header()
	if y < m.top() || y >= m.top()+len(lines)-1 {
		return ""
	}
	j := grid.HeaderAt(m.rows(m.stage), m.names(m.stage), m.gridWidth(m.stage), x-m.indent(m.stage))
	if j < 0 {
		return ""
	}
	return m.shownColumns(m.stage)[j]
}

// sortSpot is a column's sort target on the rule under the header: the screen columns under its name, from its first
// to the one past it, as wide as the name's first line or its mark, whichever is wider, and the mark drawn there,
// none where the column does not sort.
type sortSpot struct {
	column, mark string
	from, until  int
}

// sortSpots gives the sort target of each column the header names that can sort, under its name on the rule, the mark
// starting under the name's first cell; none while the header draws no names.
func (m Model) sortSpots() []sortSpot {
	if len(m.stages) == 0 || m.noHeaders || m.stages[m.stage].loading || m.stages[m.stage].refusal != "" {
		return nil
	}
	rows, names, width, in := m.rows(m.stage), m.names(m.stage), m.gridWidth(m.stage), m.indent(m.stage)
	p := m.panel(m.stage)
	var out []sortSpot
	for j, column := range m.shownColumns(m.stage) {
		start, first := grid.HeaderName(rows, names, width, j)
		if start < 0 || !p.Sortable[column] {
			continue
		}
		mark := p.Mark(column)
		x := in + start
		out = append(out, sortSpot{column, mark, x, x + max(ansi.StringWidth(first), ansi.StringWidth(mark))})
	}
	return out
}

// ruleFill gives the rule's dim line, fill cells from the gutter on, each sort's mark on it under its column's name,
// as sortSpots places them, not dim, where the whole mark fits on the line.
func (m Model) ruleFill(fill int) string {
	cells := slices.Repeat([]string{"─"}, fill)
	marked := make([]bool, fill)
	for _, s := range m.sortSpots() {
		at, glyphs := s.from-len(lead), strings.Split(s.mark, "")
		if s.mark == "" || at < 0 || at+len(glyphs) > fill {
			continue
		}
		for k, g := range glyphs {
			cells[at+k], marked[at+k] = g, true
		}
	}
	var out strings.Builder
	for k := 0; k < fill; {
		end := k
		for end < fill && marked[end] == marked[k] {
			end++
		}
		run := strings.Join(cells[k:end], "")
		if !marked[k] {
			run = style.Off(run)
		}
		out.WriteString(run)
		k = end
	}
	return out.String()
}

// sortAt gives the column whose sort a click at the screen column x on the rule under the header turns: the one whose
// target, under its name, as sortSpots gives them, x falls on, short of the rule's right end, where the selected count
// and the preset line stand. Where a column does not sort yet, its target is the rule's line, and still sorts it.
// Nothing elsewhere.
func (m Model) sortAt(x int) string {
	line := ansi.Strip(m.rule())
	for _, s := range m.sortSpots() {
		if x >= s.from && x < s.until && ansi.Cut(line, x, x+1) != " " && x < ansi.StringWidth(line) {
			return s.column
		}
	}
	return ""
}

// sortBy turns a column's sort, as a click on its target on the rule asks: ascending where it does not sort yet, after
// the other keys; around where it does; or, with drop, as shift asks, off.
func (m *Model) sortBy(column string, drop bool) {
	p, d := m.panel(m.stage), grid.Descending
	switch {
	case p.Mark(column) == "":
		d = grid.Ascending
	case drop:
		d = grid.Unsorted
	case strings.Contains(p.Mark(column), "▼"):
		d = grid.Ascending
	}
	m.setPanel(p.SortedAs(column, d))
}
