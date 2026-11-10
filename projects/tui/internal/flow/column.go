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

// sortSpot is a column's targets on the rule under the header, from the screen column under its name's first cell: a
// column that holds a list has the flatten mark there, dim while a row holds all its values, not once flattened, and a
// click on it flattens or gathers; after it, or from that first cell for another column, the sort's, its mark where it
// sorts, Sortable where it can and does not, and none where it cannot. The sort's target runs to the end of the
// name's first line, or of the marks, whichever is wider.
type sortSpot struct {
	column, mark, flat string
	sorts, flattened   bool
	at, until          int
}

// sortFrom gives the screen column the sort's mark and target start on: after the flatten mark, where there is one.
func (s sortSpot) sortFrom() int { return s.at + ansi.StringWidth(s.flat) }

// sortSpots gives the targets of each column the header names that can sort or flatten, under its name on the rule,
// wherever the name aligns; none while the header draws no names.
func (m Model) sortSpots() []sortSpot {
	if len(m.stages) == 0 || m.noHeaders || m.stages[m.stage].src.Loading || m.stages[m.stage].refusal != "" {
		return nil
	}
	rows, names, width, in := m.rows(m.stage), m.names(m.stage), m.gridWidth(m.stage), m.indent(m.stage)
	p := m.panel(m.stage)
	var out []sortSpot
	for j, column := range m.shownColumns(m.stage) {
		start, first := grid.HeaderName(rows, names, width, j)
		if start < 0 || !p.Sorts(column) && !p.Lists(column) {
			continue
		}
		s := sortSpot{column: column, at: in + start}
		if p.Lists(column) {
			s.flat, s.flattened = table.FlattenGlyph, p.Field(column).Flat
		}
		if p.Sorts(column) {
			s.mark = p.Mark(column)
			if s.sorts = s.mark != ""; !s.sorts {
				s.mark = Sortable
			}
		}
		s.until = s.at + max(ansi.StringWidth(first), ansi.StringWidth(s.flat+s.mark))
		out = append(out, s)
	}
	return out
}

// ruleFill gives the rule's dim line, fill cells from the screen column from on, each column's marks on it under its
// name, as sortSpots places them, where the whole of them fits on the line: a sort's not dim, nor a flattened list's;
// Sortable, and the flatten mark of a list not flat, dim.
func (m Model) ruleFill(from, fill int) string {
	cells := slices.Repeat([]string{"─"}, fill)
	marked := make([]bool, fill)
	for _, s := range m.sortSpots() {
		at := s.at - from
		if at < 0 || at+ansi.StringWidth(s.flat+s.mark) > fill {
			continue
		}
		for k, g := range strings.Split(s.flat, "") {
			cells[at+k], marked[at+k] = g, s.flattened
		}
		at = s.sortFrom() - from
		for k, g := range strings.Split(s.mark, "") {
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

// sortAt gives the column whose target a click at the screen column x on the rule under the header falls on, as
// sortSpots gives them, short of the rule's right end, where the selected count may stand, and whether it is the
// flatten mark, which flattens or gathers, rather than the sort's, which turns it. Where a column does not sort yet,
// its target is the rule's line, and still sorts it; a list not flat has the flatten mark alone. Nothing elsewhere.
func (m Model) sortAt(x int) (column string, flatten bool) {
	line := ansi.Strip(m.rule())
	if x >= ansi.StringWidth(line) || ansi.Cut(line, x, x+1) == " " {
		return "", false
	}
	for _, s := range m.sortSpots() {
		switch {
		case s.flat != "" && x >= s.at && x < s.sortFrom():
			return s.column, true
		case s.mark != "" && x >= s.sortFrom() && x < s.until:
			return s.column, false
		}
	}
	return "", false
}

// sortBy turns a column's sort, as a click on its target on the rule asks, a step each click, as the panel's Sorted
// says: ascending where it does not sort yet, after the other keys; descending where it sorts ascending, in its place;
// then off on the last key, and ascending again on any other. A shift-click turns it off, wherever it stands.
func (m *Model) sortBy(column string, shift bool) {
	p := m.panel(m.stage)
	if shift {
		m.setPanel(p.SortedAs(column, grid.Unsorted))
		return
	}
	m.setPanel(p.Sorted(column))
}
