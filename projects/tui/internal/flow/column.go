// The column menu: a right click on a column's name in the header drops it under the name, with what a click and its
// modifiers do there, as many terminals keep option and ctrl with a click for their own selection and menu, and never
// send them: sort by the column alone or after the other keys, group by it alone or within the groups, and the
// field's popover.
package flow

import (
	"cmp"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/empowerite/kit/projects/tui/internal/grid"
	"github.com/empowerite/kit/projects/tui/internal/shell"
)

// columnMenu is the column menu while it is open: the column it acts on, the line under the keyboard, and the screen
// column and line of the click that dropped it.
type columnMenu struct {
	open   bool
	column string
	at     int
	x, y   int
}

// The column menu's lines: SortBy and ThenSort, before the column's label, sort by it alone or after the other keys;
// GroupBy and ThenGroup group by it alone or within the groups; and More opens the field's popover.
const (
	SortBy    = "Sort by "
	ThenSort  = "Then sort by "
	GroupBy   = "Group by "
	ThenGroup = "Then group by "
	More      = "Options…"
)

// columnItems gives the column menu's lines: the two sorts, the two groupings, a rule, and Options…. A line that cannot
// act on the column, which does not sort or does not group, stays, dim.
func (m Model) columnItems() []shell.Item {
	label := cmp.Or(m.panel(m.stage).Labels[m.colMenu.column], m.colMenu.column)
	return []shell.Item{{Name: SortBy + label}, {Name: ThenSort + label}, {Name: GroupBy + label},
		{Name: ThenGroup + label}, {}, {Name: More}}
}

// columnEnabled says whether the column menu's line k acts: a sort on a column that sorts, a grouping on one that
// groups, and Options… always; the rule never does.
func (m Model) columnEnabled(k int) bool {
	p := m.panel(m.stage)
	switch k {
	case 0, 1:
		return p.Sortable[m.colMenu.column]
	case 2, 3:
		return p.Groupable[m.colMenu.column]
	}
	return k == 5
}

// openColumnMenu drops the column menu down under the click at x, y on a column's name, the first line that acts under
// the keyboard.
func (m *Model) openColumnMenu(column string, x, y int) {
	m.colMenu = columnMenu{open: true, column: column, x: x, y: y, at: -1}
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
	case 0, 1:
		p = p.Sorted(c.column, k == 1)
	case 2, 3:
		p = p.Grouped(c.column, k == 3)
	default:
		return m.openPopover(c.column, c.x, c.y, true)
	}
	m.setPanel(p)
	return nil
}

// columnBox gives the column menu's lines, under the click, its left edge a cell before it, or nearer the screen's left
// when that would run past its edge, and the screen line and column it is drawn from.
func (m Model) columnBox() (lines []string, top, left int) {
	lines, _ = shell.Box(m.columnItems(), m.columnEnabled, m.colMenu.at, "", -1, m.width)
	return lines, m.colMenu.y + 1, max(0, min(m.colMenu.x-1, m.width-ansi.StringWidth(lines[0])))
}

// columnClick takes a click while the column menu is open: on a line that acts, it does what the line says; anywhere
// else, it closes the menu.
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
