// Where a column's name stands, in the header and in the heading over the groups, which the column menu, internal/
// table's, hangs from; and the sorts' marks on the rule under the header, a click on which turns a column's sort.
package flow

import (
	"slices"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/empowerite/kit/projects/tui/internal/grid"
	"github.com/empowerite/kit/projects/tui/internal/style"
	"github.com/empowerite/kit/projects/tui/internal/table"
)

// openFieldMenu drops the column menu around a column's name in the heading over the groups, on the control line,
// which then has the keyboard, the first line that acts under it.
func (m *Model) openFieldMenu(column string) {
	m.onPreset, m.inRows, m.inPanel = m.presets(m.stage), false, false
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

// fieldName gives the screen column and line a column's name stands on in the heading over the groups, on the control
// line; a line of -1 where the heading no longer names it.
func (m Model) fieldName(column string) (x, y int) {
	for _, n := range table.BarNames(&m) {
		if n.Column == column {
			return n.From, m.top()
		}
	}
	return 0, -1
}

// columnName gives the screen column a column's name starts on in the header, and its first line, as the header draws
// it.
func (m Model) columnName(column string) (int, string) {
	j := slices.Index(m.shownColumns(m.stage), column)
	x, name := grid.HeaderName(m.rows(m.stage), m.names(m.stage), m.gridWidth(m.stage), j)
	return x + m.indent(m.stage), name
}

// headerColumn gives the column whose name a click at x, y in the header falls on, none off every name.
func (m Model) headerColumn(x, y int) string {
	if h, ok := m.headerLine(y); !ok || m.onRule(y) || h < 0 {
		return ""
	}
	j := grid.HeaderAt(m.rows(m.stage), m.names(m.stage), m.gridWidth(m.stage), x-m.indent(m.stage))
	if j < 0 {
		return ""
	}
	return m.shownColumns(m.stage)[j]
}

// Sortable is the dim mark on the rule under the name of a column that can sort and does not, so the rule says where a
// click sorts; a column that cannot sort has the rule alone.
const Sortable = "·"

// sortSpot is a column's sort target on the rule under the header: the screen columns under its name, from its first
// to the one past it, as wide as the name's first line or its mark, whichever is wider; the mark drawn there, its sort's
// where it sorts, and Sortable where it does not; whether it sorts; and the screen column the mark starts on.
type sortSpot struct {
	column, mark    string
	sorts           bool
	at, from, until int
}

// sortSpots gives the sort target of each column the header names that can sort, under its name on the rule, the mark
// at the name's aligned edge: under its first cell, or under its last where the column aligns right, over numbers;
// none while the header draws no names.
func (m Model) sortSpots() []sortSpot {
	if len(m.stages) == 0 || m.noHeaders || m.stages[m.stage].src.Loading || m.stages[m.stage].refusal != "" {
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
		sorts := mark != ""
		if !sorts {
			mark = Sortable
		}
		x, at := in+start, in+start
		if grid.HeaderRight(rows, j) {
			at = max(x, x+ansi.StringWidth(first)-ansi.StringWidth(mark))
		}
		out = append(out, sortSpot{column, mark, sorts, at, x, x + max(ansi.StringWidth(first), ansi.StringWidth(mark))})
	}
	return out
}

// ruleFill gives the rule's dim line, fill cells from the screen column from on, each sort's mark on it under its
// column's name, as sortSpots places them, not dim, where the whole mark fits on the line, and Sortable, dim, under a
// column that can sort and does not.
func (m Model) ruleFill(from, fill int) string {
	cells := slices.Repeat([]string{"─"}, fill)
	marked := make([]bool, fill)
	for _, s := range m.sortSpots() {
		at, glyphs := s.at-from, strings.Split(s.mark, "")
		if at < 0 || at+len(glyphs) > fill {
			continue
		}
		for k, g := range glyphs {
			cells[at+k], marked[at+k] = g, s.sorts
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
// may stand. Where a column does not sort yet, its target is the rule's line, and still sorts it.
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

// sortBy turns a column's sort, as a plain click on its target on the rule asks, a step each click: ascending where it
// does not sort yet, after the other keys; descending where it sorts ascending, in its place; and off where it sorts
// descending.
func (m *Model) sortBy(column string) {
	p, d := m.panel(m.stage), grid.Descending
	switch {
	case p.Mark(column) == "":
		d = grid.Ascending
	case strings.Contains(p.Mark(column), "▼"):
		d = grid.Unsorted
	}
	m.setPanel(p.SortedAs(column, d))
}
