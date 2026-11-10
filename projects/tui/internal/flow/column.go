// Where a column's name stands, in the header and in the heading over the groups, which the column menu, internal/
// table's, hangs from; and the sorts' marks on the rule under the header, a click on which turns a column's sort.
package flow

import (
	"slices"

	tea "charm.land/bubbletea/v2"

	"github.com/empowerite/kit/projects/tui/internal/table"
)

// openFieldMenu drops the column menu around a column's name in the heading over the groups, on the control line,
// which then has the keyboard, the first line that acts under it.
func (m *Model) openFieldMenu(column string) {
	m.onPreset, m.inRows, m.inPanel = m.presets(m.stage), false, false
	m.stop = slices.IndexFunc(m.stops(), func(s table.Stop) bool { return s.Column == column })
	m.colMenu = table.OpenColumnMenu(m, column, true)
}

// openColumnMenu drops the column menu down from a column's name in the header, the first line that acts under the
// keyboard.
func (m *Model) openColumnMenu(column string) { m.colMenu = table.OpenColumnMenu(m, column, false) }

// columnAct does what the column menu's line k says, as table.ColumnMenu does, and opens the field's popover where
// it says to, which asks kit for the caller's login.
func (m *Model) columnAct(k int) tea.Cmd {
	if pop, ok := m.colMenu.Act(m, k); ok {
		m.pop = pop
		return m.askMe()
	}
	return nil
}

// fieldName gives the screen column and line a column's name stands on in the heading over the groups, as
// table.FieldName says.
func (m Model) fieldName(column string) (x, y int) { return table.FieldName(&m, column) }

// columnName gives the screen column a column's name starts on in the header, and its first line, as table.ColumnName
// says.
func (m Model) columnName(column string) (int, string) { return table.ColumnName(&m, column) }

// headerColumn gives the column whose name a click at x, y in the header falls on, as table.HeaderColumn says.
func (m Model) headerColumn(x, y int) string { return table.HeaderColumn(&m, x, y) }

// Sortable is the dim mark on the rule under the name of a column that can sort and does not, as table.Sortable says.
const Sortable = table.Sortable

// sortSpot is a column's targets on the rule under the header, as table.SortSpot says.
type sortSpot = table.SortSpot

// sortSpots gives the targets of each column the header names that can sort or flatten, as table.SortSpots gives
// them; none while the header draws no names.
func (m Model) sortSpots() []sortSpot {
	if len(m.stages) == 0 || m.noHeaders || m.stages[m.stage].src.Loading || m.stages[m.stage].refusal != "" {
		return nil
	}
	return table.SortSpots(&m)
}

// sortAt gives the column whose target a click at the screen column x on the rule falls on, and whether it is the
// flatten mark, as table.SortAt says.
func (m Model) sortAt(x int) (column string, flatten bool) {
	return table.SortAt(m.rule(), m.sortSpots(), x)
}

// sortBy turns a column's sort, as a click on its target on the rule asks, as table.SortBy says.
func (m *Model) sortBy(column string, shift bool) { table.SortBy(m, column, shift) }
