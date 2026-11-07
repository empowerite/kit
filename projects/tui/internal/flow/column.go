// The column menu: a click on a column's name in the header drops it under the name, as does a right click where the
// terminal sends one. Many terminals keep option, ctrl, and the right button with a click for their own selection and
// menu, so the menu holds what those would do: the column's sort, off, ascending, or descending, a radio that shows
// it, or the column after the other keys; group by it alone or within the groups; and the field's popover, which
// leaves the sort to the menu.
package flow

import (
	"cmp"
	"fmt"
	"slices"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/empowerite/kit/projects/tui/internal/grid"
	"github.com/empowerite/kit/projects/tui/internal/shell"
	"github.com/empowerite/kit/projects/tui/internal/style"
)

// columnMenu is the column menu while it is open: the column it acts on, and the line under the keyboard.
type columnMenu struct {
	open   bool
	column string
	at     int
}

// The column menu's lines, by index: a dim heading, sort, saying the column's place among the keys while there are
// several; the radio of its sort, off, ▲ ascending, and ▼ descending; ThenSort, before the column's label, adding it
// after the other keys; a rule; GroupBy and ThenGroup, grouping by it alone or within the groups; a rule; and More,
// the field's popover.
const (
	lineSort = iota
	lineOff
	lineUp
	lineDown
	lineThen
	_
	lineGroup
	lineThenGroup
	_
	lineMore
)

// The words of the column menu's lines.
const (
	ThenSort  = "Then sort by "
	GroupBy   = "Group by "
	ThenGroup = "Then group by "
	More      = "Options…"
)

// columnItems gives the column menu's lines, as their indexes say. A line that cannot act on the column stays, dim.
func (m Model) columnItems() []shell.Item {
	p, column := m.panel(m.stage), m.colMenu.column
	label := cmp.Or(p.Labels[column], column)
	title := "sort"
	if keys := p.SortKeys(); len(keys) > 1 && p.Mark(column) != "" {
		title += fmt.Sprintf(", key %d of %d", slices.IndexFunc(keys, func(s grid.SortKey) bool { return s.Column == column })+1,
			len(keys))
	}
	radio := func(d grid.Direction, name string) shell.Item {
		if p.Field(column).Sort == d {
			return shell.Item{Glyph: style.Tint(grid.Picked), Name: name}
		}
		return shell.Item{Glyph: style.Off(grid.Unpicked), Name: name}
	}
	return []shell.Item{{Name: title}, radio(grid.Unsorted, "off"), radio(grid.Ascending, "▲ ascending"),
		radio(grid.Descending, "▼ descending"), {Name: ThenSort + label}, {}, {Name: GroupBy + label},
		{Name: ThenGroup + label}, {}, {Name: More}}
}

// columnEnabled says whether the column menu's line k acts: the radio on a column that sorts; Then sort by on one that
// sorts and is not among the keys yet; a grouping on one that groups; and Options… always; the heading and the rules
// never do.
func (m Model) columnEnabled(k int) bool {
	p, column := m.panel(m.stage), m.colMenu.column
	switch k {
	case lineOff, lineUp, lineDown:
		return p.Sortable[column]
	case lineThen:
		return p.Sortable[column] && p.Mark(column) == ""
	case lineGroup, lineThenGroup:
		return p.Groupable[column]
	}
	return k == lineMore
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
	case lineOff:
		p = p.SortedAs(c.column, grid.Unsorted)
	case lineUp, lineDown:
		// a column among the keys turns in its place; one that is not sorts the rows alone
		d := map[int]grid.Direction{lineUp: grid.Ascending, lineDown: grid.Descending}[k]
		if p.Mark(c.column) == "" {
			p = sortedAlone(p, c.column)
		}
		p = p.SortedAs(c.column, d)
	case lineThen:
		p = p.SortedAs(c.column, grid.Ascending)
	case lineGroup, lineThenGroup:
		p = p.Grouped(c.column, k == lineThenGroup)
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
