// The column menu: a click on a column's name in the header drops it under the name, as does a right click where the
// terminal sends one. Many terminals keep option, ctrl, and the right button with a click for their own selection and
// menu, so the menu holds what those would do: the column's sort, a radio that shows it, None, by the column alone or
// first, Ascending or Descending, or after the other keys, Ascending Next or Descending Next; group by it alone or
// within the groups; the column hidden; and the field's popover, which leaves the sort to the menu.
package flow

import (
	"cmp"
	"fmt"
	"slices"
	"strconv"
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
// several; the radio of its sort: None; Ascending and Descending, by the column alone or first; and Ascending Next and
// Descending Next, after the other keys, each glyph with the place the column takes among them; a rule; GroupBy and
// ThenGroup, grouping by it alone or within the groups; a rule; Hide, before the column's label; and More, the field's
// popover.
const (
	lineSort = iota
	lineNone
	lineUp
	lineDown
	lineUpNext
	lineDownNext
	_
	lineGroup
	lineThenGroup
	_
	lineHide
	lineMore
)

// The words of the column menu's lines.
const (
	GroupBy   = "Group by "
	ThenGroup = "Then group by "
	Hide      = "Hide "
	More      = "Options…"
)

// keyAt gives where a column stands among the sort's keys, from 1, or 0 where it does not sort the rows, and how many
// keys there are but it.
func keyAt(p grid.Panel, column string) (place, others int) {
	keys := p.SortKeys()
	place = slices.IndexFunc(keys, func(s grid.SortKey) bool { return s.Column == column }) + 1
	if place > 0 {
		return place, len(keys) - 1
	}
	return 0, len(keys)
}

// columnItems gives the column menu's lines, as their indexes say. The radio's choice picked is the column's sort:
// None where it does not sort, Ascending or Descending where it is the first key, and a Next where it is a later one,
// whose glyph's subscript is its place; on a column not among the keys, a Next's subscript is the place it would take.
// A line that cannot act on the column stays, dim.
func (m Model) columnItems() []shell.Item {
	p, column := m.panel(m.stage), m.colMenu.column
	label := cmp.Or(p.Labels[column], column)
	place, others := keyAt(p, column)
	title := "sort"
	if place > 0 && others > 0 {
		title += fmt.Sprintf(", key %d of %d", place, others+1)
	}
	next := place
	if next < 2 {
		next = others + 1
	}
	d := p.Field(column).Sort
	radio := func(picked bool, name string) shell.Item {
		if picked {
			return shell.Item{Glyph: style.Tint(grid.Picked), Name: name}
		}
		return shell.Item{Glyph: style.Off(grid.Unpicked), Name: name}
	}
	return []shell.Item{{Name: title}, radio(place == 0, "None"),
		radio(place == 1 && d == grid.Ascending, "▲ Ascending"), radio(place == 1 && d == grid.Descending, "▼ Descending"),
		radio(place > 1 && d == grid.Ascending, "▲"+subscript(next)+" Ascending Next"),
		radio(place > 1 && d == grid.Descending, "▼"+subscript(next)+" Descending Next"), {},
		{Name: GroupBy + label}, {Name: ThenGroup + label}, {}, {Name: Hide + label}, {Name: More}}
}

// subscript gives a number in subscript digits, as the sort's mark gives a key's place: ₂.
func subscript(n int) string {
	return strings.Map(func(r rune) rune { return '₀' + r - '0' }, strconv.Itoa(n))
}

// columnEnabled says whether the column menu's line k acts: the radio on a column that sorts, a Next only where there
// is another key for it to come after; a grouping on one that groups; and Hide and Options… always; the heading and the
// rules never do.
func (m Model) columnEnabled(k int) bool {
	p, column := m.panel(m.stage), m.colMenu.column
	_, others := keyAt(p, column)
	switch k {
	case lineNone, lineUp, lineDown:
		return p.Sortable[column]
	case lineUpNext, lineDownNext:
		return p.Sortable[column] && others > 0
	case lineGroup, lineThenGroup:
		return p.Groupable[column]
	}
	return k == lineHide || k == lineMore
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
	switch place, _ := keyAt(p, c.column); k {
	case lineNone:
		p = p.SortedAs(c.column, grid.Unsorted)
	case lineUp, lineDown:
		// by the column alone, in that direction
		p = sortedAlone(p, c.column).SortedAs(c.column, map[int]grid.Direction{lineUp: grid.Ascending,
			lineDown: grid.Descending}[k])
	case lineUpNext, lineDownNext:
		// a later key turns in its place; the first, or one not among the keys, goes after the others
		d := map[int]grid.Direction{lineUpNext: grid.Ascending, lineDownNext: grid.Descending}[k]
		if place == 1 {
			p = p.SortedAs(c.column, grid.Unsorted)
		}
		p = p.SortedAs(c.column, d)
	case lineHide:
		p = p.Placed(c.column, grid.Hidden)
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
