// The heading menu: a click on a column's name in a group's heading drops it around the name, and so does ⏎ with the
// cursor on the heading, around its first column's; a click anywhere else on the heading folds its rows away, or
// unfolds them, as one on its ▾ or ▸ does. Many terminals keep option, ctrl, and the right button with a click for their own
// selection and menu, so the menu holds what those would do on a heading, and what space does there: everything under
// the heading expanded or collapsed, the group's rows selected or unselected, and the rows no longer grouped by the
// heading's column, which titles the menu. Shift with left or right moves the column a depth outward or inward among
// the groups, the menu with its heading.
package flow

import (
	"slices"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/empowerite/kit/projects/tui/internal/grid"
	"github.com/empowerite/kit/projects/tui/internal/shell"
)

// headingMenu is the heading menu while it is open: the key of the heading it acts on, the column whose name it hangs
// from, the screen line and column the name stood on when it opened, and the line under the keyboard.
type headingMenu struct {
	open   bool
	key    string
	column string
	x, y   int
	at     int
}

// nameSpot is a column's name in a heading, and the screen columns it stands on, from its first to the one past it.
type nameSpot struct {
	column      string
	from, until int
}

// headingNames gives the columns a group's heading names, each where the rows draw its name: the column the person
// grouped by, then those the rows group by on their own, each name after its value, before a comma, the count, or the
// heading's end. The heading stands after the gutter's one cell.
func (m Model) headingNames(r row) []nameSpot {
	text := ansi.Strip(r.Cells[0].Values[0])
	var columns []string
	if c := m.panel(m.stage).Heading(r); c != "" {
		columns = append(columns, c)
	}
	var out []nameSpot
	for _, column := range append(columns, r.Says...) {
		for from := 0; ; {
			k := strings.Index(text[from:], " "+column)
			if k < 0 {
				break
			}
			at := from + k + 1
			if end := at + len(column); end == len(text) || text[end] == ',' || strings.HasPrefix(text[end:], "  ") {
				x := 1 + ansi.StringWidth(text[:at])
				out = append(out, nameSpot{column, x, x + ansi.StringWidth(column)})
				break
			}
			from = at
		}
	}
	return out
}

// The heading menu's lines: everything under the heading opened or closed, the group's rows selected or, once every
// one is, unselected, and the rows no longer grouped by its column, as the column menu's Ungroup says.
const (
	ExpandUnder   = "Expand All Under It"
	CollapseUnder = "Collapse All Under It"
	SelectGroup   = "Select Group"
	UnselectGroup = "Unselect Group"
)

// headingAt gives the index of the heading the menu acts on among the selected stage's rows, -1 once it shows no more.
func (m Model) headingAt() int {
	return slices.IndexFunc(m.rows(m.stage), func(r row) bool { return r.Key == m.headMenu.key })
}

// nameAt gives the column's name in a heading that the screen column x falls on, and whether one does.
func (m Model) nameAt(r row, x int) (nameSpot, bool) {
	for _, n := range m.headingNames(r) {
		if x >= n.from && x < n.until {
			return n, true
		}
	}
	return nameSpot{}, false
}

// headingItems gives the heading menu's lines: expand and collapse, a rule, the group selected or unselected, a rule,
// and the rows ungrouped by the menu's column. A line that cannot act stays, dim.
func (m Model) headingItems() []shell.Item {
	selecting := SelectGroup
	rows, s := m.rows(m.stage), m.stages[m.stage]
	if k := m.headingAt(); k >= 0 {
		every := m.every(m.stage)
		keys := groupRows(every, slices.IndexFunc(every, func(r row) bool { return r.Key == rows[k].Key }))
		if len(keys) > 0 && !slices.ContainsFunc(keys, func(key string) bool { return !s.picked[key] }) {
			selecting = UnselectGroup
		}
	}
	return []shell.Item{{Glyph: Expand, Name: ExpandUnder}, {Glyph: Collapse, Name: CollapseUnder}, {},
		{Glyph: Every, Name: selecting}, {}, {Glyph: UngroupGlyph, Name: UngroupIt}}
}

// headingEnabled says whether the heading menu's line k acts: expand while a node under the heading, or the heading
// itself, is closed, collapse while one is open; selecting always; ungrouping on a column the person groups by, not one
// the rows group by on their own; the rules never.
func (m Model) headingEnabled(k int) bool {
	at := m.headingAt()
	if at < 0 {
		return false
	}
	switch k {
	case 0, 1:
		s, every := m.stages[m.stage], m.every(m.stage)
		j := slices.IndexFunc(every, func(r row) bool { return r.Key == m.headMenu.key })
		for i := j; i < grid.Span(every, j); i++ {
			if m.opens(every, i) && s.open.Open(every[i]) == (k == 1) {
				return true
			}
		}
		return false
	case 3:
		return true
	case 5:
		return slices.Contains(m.panel(m.stage).Groups(), m.headMenu.column)
	}
	return false
}

// openHeadingMenu drops the heading menu around a column's name in the heading k, at the screen line y, the cursor on
// the heading, the first line that acts under the keyboard.
func (m *Model) openHeadingMenu(k, y int, name nameSpot) {
	m.stages[m.stage].cursor, m.inRows = k, true
	m.headMenu = headingMenu{open: true, key: m.rows(m.stage)[k].Key, column: name.column, x: name.from, y: y, at: -1}
	m.headMenu.at = shell.Move(m.headingItems(), m.headingEnabled, -1, 1)
}

// headingKey takes a key while the heading menu is open: up and down move, a letter to the next line it starts, enter
// and space act, shift with left or right moves the column among the groups, and esc closes it.
func (m Model) headingKey(k tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	items := m.headingItems()
	switch k.String() {
	case "up":
		m.headMenu.at = shell.Move(items, m.headingEnabled, m.headMenu.at, -1)
	case "down":
		m.headMenu.at = shell.Move(items, m.headingEnabled, m.headMenu.at, 1)
	case "enter", "space":
		m.headingAct(m.headMenu.at)
	case "esc":
		m.headMenu = headingMenu{}
	case "shift+left", "shift+right":
		d, _ := shiftStep(k.String())
		m.shiftHeading(d)
	default:
		m.headMenu.at = shell.NextItem(items, m.headingEnabled, m.headMenu.at, k.Text)
	}
	return m, nil
}

// headingAct closes the heading menu and does what its line k says, when it acts, the cursor on the heading.
func (m *Model) headingAct(k int) {
	enabled, at, column := m.headingEnabled(k), m.headingAt(), m.headMenu.column
	m.headMenu = headingMenu{}
	if !enabled {
		return
	}
	m.stages[m.stage].cursor, m.inRows = at, true
	switch k {
	case 0, 1:
		m.disclose(k == 0, true)
	case 3:
		m.selectRow(false)
	default:
		m.setPanel(m.panel(m.stage).Grouped(column, true))
	}
	m.fit()
}

// shiftHeading moves the heading menu's column a depth among the groups, outward for a negative step, and the menu with
// it: onto the heading of that column over the first row the heading headed, where it now stands, the cursor on it, the
// keyboard on the same line. Where the column cannot move that way, nothing changes; where the heading no longer
// shows, the menu closes.
func (m *Model) shiftHeading(d int) {
	p, ok := m.shifted(m.headMenu.column, d)
	every := m.every(m.stage)
	j := slices.IndexFunc(every, func(r row) bool { return r.Key == m.headMenu.key })
	if !ok || j < 0 {
		return
	}
	under := groupRows(every, j)
	if len(under) == 0 {
		return
	}
	m.setPanel(p)
	m.fit()
	every, column := m.every(m.stage), m.headMenu.column
	i := slices.IndexFunc(every, func(r row) bool { return r.Key == under[0] })
	for ; i >= 0 && m.panel(m.stage).Heading(every[i]) != column; i-- {
	}
	rows := m.rows(m.stage)
	k := -1
	if i >= 0 {
		k = slices.IndexFunc(rows, func(r row) bool { return r.Key == every[i].Key })
	}
	if k < 0 {
		m.headMenu = headingMenu{}
		return
	}
	m.stages[m.stage].cursor = k
	m.fit()
	y := m.rowLine(k)
	for _, name := range m.headingNames(rows[k]) {
		if name.column == column && y >= 0 {
			m.headMenu.key, m.headMenu.x, m.headMenu.y = rows[k].Key, name.from, y
			return
		}
	}
	m.headMenu = headingMenu{}
}

// headingBox gives the heading menu's lines, the box around the column's name, which its top edge draws where the
// heading does, as the column menu's draws the name in the header; and the screen line and column it is drawn from.
// Once a reload takes the heading away, there are none.
func (m Model) headingBox() (lines []string, top, left int) {
	if m.headingAt() < 0 {
		return nil, 0, 0
	}
	lines, left = shell.Box(m.headingItems(), m.headingEnabled, m.headMenu.at, m.headMenu.column, m.headMenu.x,
		m.paneWidth())
	return lines, m.headMenu.y, left
}

// headingClick takes a click while the heading menu is open: on a line that acts, it does what the line says; anywhere
// else, on the heading it hangs from too, it closes the menu.
func (m *Model) headingClick(x, y int) {
	box, top, left := m.headingBox()
	if k := y - top - 1; len(box) > 0 && x > left && x < left+ansi.StringWidth(box[0])-1 && k >= 0 && k < len(box)-2 {
		m.headingAct(k)
		return
	}
	m.headMenu = headingMenu{}
}
