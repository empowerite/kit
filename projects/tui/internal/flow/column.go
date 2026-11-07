// The column menu: a click on a column's name in the header drops it under the name, as does a right click where the
// terminal sends one. Many terminals keep option, ctrl, and the right button with a click for their own selection and
// menu, so the menu holds what those would do: sort by the column alone or after the other keys, group by it alone or
// within the groups, and the field's popover.
package flow

import (
	"cmp"
	"slices"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/empowerite/kit/projects/tui/internal/grid"
	"github.com/empowerite/kit/projects/tui/internal/shell"
)

// columnMenu is the column menu while it is open: the column it acts on, and the line under the keyboard.
type columnMenu struct {
	open   bool
	column string
	at     int
}

// The column menu's lines: SortBy, before the column's label, sorts by it alone, ThenSort after the other keys, and
// NoSort stops sorting by it; GroupBy and ThenGroup group by it alone or within the groups; and More opens the field's
// popover.
const (
	SortBy    = "Sort by "
	ThenSort  = "Then sort by "
	NoSort    = "Don't sort by "
	GroupBy   = "Group by "
	ThenGroup = "Then group by "
	More      = "Options…"
)

// columnItems gives the column menu's lines: the three sorts, the two groupings, a rule, and Options…. A line that
// cannot act on the column stays, dim.
func (m Model) columnItems() []shell.Item {
	label := cmp.Or(m.panel(m.stage).Labels[m.colMenu.column], m.colMenu.column)
	return []shell.Item{{Name: SortBy + label}, {Name: ThenSort + label}, {Name: NoSort + label},
		{Name: GroupBy + label}, {Name: ThenGroup + label}, {}, {Name: More}}
}

// columnEnabled says whether the column menu's line k acts: Sort by on a column that sorts; Then sort by on one that
// sorts and is not among the keys yet, and Don't sort by on one that is; a grouping on one that groups; and Options…
// always; the rule never does.
func (m Model) columnEnabled(k int) bool {
	p, column := m.panel(m.stage), m.colMenu.column
	keyed := slices.ContainsFunc(p.SortKeys(), func(s grid.SortKey) bool { return s.Column == column })
	switch k {
	case 0:
		return p.Sortable[column]
	case 1:
		return p.Sortable[column] && !keyed
	case 2:
		return keyed
	case 3, 4:
		return p.Groupable[column]
	}
	return k == 6
}

// sortedAlone gives the panel sorting the rows by the column alone: in its direction, when it is among the keys
// already, else ascending.
func sortedAlone(p grid.Panel, column string) grid.Panel {
	for _, s := range p.SortKeys() {
		if s.Column != column {
			p = p.SortedAs(s.Column, grid.Unsorted)
		}
	}
	if p.Mark(column) == "" {
		p = p.SortedAs(column, grid.Ascending)
	}
	return p
}

// openColumnMenu drops the column menu down from a column's name, the first line that acts under the keyboard.
func (m *Model) openColumnMenu(column string) {
	m.colMenu = columnMenu{open: true, column: column, at: -1}
	m.colMenu.at = shell.Move(m.columnItems(), m.columnEnabled, -1, 1)
}

// columnKey takes a key while the column menu is open: up and down move, a letter to the next line it starts, enter
// and space act, and esc closes it.
func (m Model) columnKey(k tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	items := m.columnItems()
	switch k.String() {
	case "up":
		m.colMenu.at = shell.Move(items, m.columnEnabled, m.colMenu.at, -1)
	case "down":
		m.colMenu.at = shell.Move(items, m.columnEnabled, m.colMenu.at, 1)
	case "enter", "space":
		cmd := m.columnAct(m.colMenu.at) // before m is returned, which a return would copy first
		return m, cmd
	case "esc":
		m.colMenu = columnMenu{}
	default:
		m.colMenu.at = shell.NextItem(items, m.columnEnabled, m.colMenu.at, k.Text)
	}
	return m, nil
}

// columnAct closes the column menu and does what its line k says, when it acts.
func (m *Model) columnAct(k int) tea.Cmd {
	c, enabled := m.colMenu, m.columnEnabled(k)
	m.colMenu = columnMenu{}
	if !enabled {
		return nil
	}
	p := m.panel(m.stage)
	switch k {
	case 0:
		p = sortedAlone(p, c.column)
	case 1:
		p = p.SortedAs(c.column, grid.Ascending)
	case 2:
		p = p.SortedAs(c.column, grid.Unsorted)
	case 3, 4:
		p = p.Grouped(c.column, k == 4)
	default:
		x, _ := m.columnName(c.column)
		return m.openPopover(c.column, x, paneTop, true)
	}
	m.setPanel(p)
	return nil
}

// columnBox gives the column menu's lines, the box around the column's name, which its top edge draws where the header
// does, as the app's menu hangs from ≡; and the screen line and column it is drawn from.
func (m Model) columnBox() (lines []string, top, left int) {
	x, name := m.columnName(m.colMenu.column)
	lines, left = shell.Box(m.columnItems(), m.columnEnabled, m.colMenu.at, name, x, m.width)
	return lines, paneTop, left
}

// columnName gives the screen column a column's name starts on in the header, and its first line, as the header draws
// it.
func (m Model) columnName(column string) (int, string) {
	j := slices.Index(m.shownColumns(m.stage), column)
	x, name := grid.HeaderName(m.rows(m.stage), m.names(m.stage), m.gridWidth(m.stage), j)
	return x + m.indent(m.stage), name
}

// columnClick takes a click while the column menu is open: on a line that acts, it does what the line says; anywhere
// else, on the name it hangs from too, it closes the menu.
func (m *Model) columnClick(x, y int) tea.Cmd {
	box, top, left := m.columnBox()
	if k := y - top - 1; x > left && x < left+ansi.StringWidth(box[0])-1 && k >= 0 && k < len(box)-2 {
		return m.columnAct(k)
	}
	m.colMenu = columnMenu{}
	return nil
}

// headerColumn gives the column whose name a click at x, y in the header falls on, none off every name.
func (m Model) headerColumn(x, y int) string {
	lines, _ := m.header()
	if y < paneTop || y >= paneTop+len(lines)-1 {
		return ""
	}
	j := grid.HeaderAt(m.rows(m.stage), m.names(m.stage), m.gridWidth(m.stage), x-m.indent(m.stage))
	if j < 0 {
		return ""
	}
	return m.shownColumns(m.stage)[j]
}

// sortAt gives the column whose sort a click at x, y on the header turns, one that can sort: on its sort mark, as onMark
// says; or, on the names' first line, on the cell right after its name, or on the cell after that, unless that one
// stands against the next column's name. Where a column does not sort yet, those cells are blank, and still sort it.
// Nothing elsewhere.
func (m Model) sortAt(x, y int) string {
	rows, names, width, in := m.rows(m.stage), m.names(m.stage), m.gridWidth(m.stage), m.indent(m.stage)
	shown, p := m.shownColumns(m.stage), m.panel(m.stage)
	if m.onMark(x, y) {
		if j := grid.HeaderAt(rows, names, width, x-in); j >= 0 && p.Sortable[shown[j]] {
			return shown[j]
		}
	}
	if y != paneTop {
		return ""
	}
	starts, ends := map[int]bool{}, map[int]string{}
	for j, column := range shown {
		start, first := grid.HeaderName(rows, names, width, j)
		if start < 0 {
			continue
		}
		starts[in+start] = true
		if p.Sortable[column] {
			ends[in+start+ansi.StringWidth(strings.TrimSuffix(first, " "+p.Mark(column)))] = column
		}
	}
	if column, ok := ends[x]; ok {
		return column
	}
	if column, ok := ends[x-1]; ok && !starts[x+1] {
		return column
	}
	return ""
}

// onMark says whether a click at x, y on a line of the header's names falls on a column's sort mark, its ▲ or ▼ and
// the number after it, or the cell before the triangle.
func (m Model) onMark(x, y int) bool {
	lines, _ := m.header()
	line := lines[y-paneTop]
	cell := func(at int) string { return ansi.Strip(ansi.Cut(line, at, at+1)) }
	triangle := func(c string) bool { return c == "▲" || c == "▼" }
	subscript := func(c string) bool { return c != "" && strings.Trim(c, "₀₁₂₃₄₅₆₇₈₉") == "" }
	if triangle(cell(x)) || triangle(cell(x+1)) && cell(x) == " " {
		return true
	}
	for at := x; subscript(cell(at)); at-- {
		if triangle(cell(at - 1)) {
			return true
		}
	}
	return false
}
