// The heading menu: a click on a group's heading, off its ▾ or ▸, drops it around the heading's text, and so does ⏎
// with the cursor on the heading. Many terminals keep option, ctrl, and the right button with a click for their own
// selection and menu, so the menu holds what those would do on a heading, and what space does there: everything under
// the heading expanded or collapsed, the group's rows selected or unselected, and the rows no longer grouped by the
// heading's column.
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

// headingMenu is the heading menu while it is open: the key of the heading it acts on, the screen line the heading
// stood on when it opened, and the line under the keyboard.
type headingMenu struct {
	open bool
	key  string
	y    int
	at   int
}

// The heading menu's lines: everything under the heading opened or closed, the group's rows selected or, once every
// one is, unselected, and, before the column's label, the rows no longer grouped by it.
const (
	ExpandUnder   = "Expand All Under It"
	CollapseUnder = "Collapse All Under It"
	SelectGroup   = "Select Group"
	UnselectGroup = "Unselect Group"
	Ungroup       = "Ungroup by "
)

// headingAt gives the index of the heading the menu acts on among the selected stage's rows, -1 once it shows no more.
func (m Model) headingAt() int {
	return slices.IndexFunc(m.rows(m.stage), func(r row) bool { return r.Key == m.headMenu.key })
}

// headingItems gives the heading menu's lines: expand and collapse, a rule, the group selected or unselected, a rule,
// and the ungrouping, which names the column, or stands alone on the heading over every row, which no column the
// person chose makes. A line that cannot act stays, dim.
func (m Model) headingItems() []shell.Item {
	selecting, ungroup := SelectGroup, strings.TrimSpace(Ungroup)
	rows, s := m.rows(m.stage), m.stages[m.stage]
	if k := m.headingAt(); k >= 0 {
		every := m.every(m.stage)
		keys := groupRows(every, slices.IndexFunc(every, func(r row) bool { return r.Key == rows[k].Key }))
		if len(keys) > 0 && !slices.ContainsFunc(keys, func(key string) bool { return !s.picked[key] }) {
			selecting = UnselectGroup
		}
		if column := m.panel(m.stage).Heading(rows[k]); column != "" {
			ungroup = Ungroup + cmp.Or(m.panel(m.stage).Labels[column], column)
		}
	}
	return []shell.Item{{Glyph: Expand, Name: ExpandUnder}, {Glyph: Collapse, Name: CollapseUnder}, {},
		{Glyph: Every, Name: selecting}, {}, {Name: ungroup}}
}

// headingEnabled says whether the heading menu's line k acts: expand while a node under the heading, or the heading
// itself, is closed, collapse while one is open; selecting always; ungrouping on a heading of a column the person
// chose; the rules never.
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
		return m.panel(m.stage).Heading(m.rows(m.stage)[at]) != ""
	}
	return false
}

// openHeadingMenu drops the heading menu around the heading at the screen line y, the cursor on the heading, the
// first line that acts under the keyboard.
func (m *Model) openHeadingMenu(k, y int) {
	m.stages[m.stage].cursor, m.inRows = k, true
	m.headMenu = headingMenu{open: true, key: m.rows(m.stage)[k].Key, y: y, at: -1}
	m.headMenu.at = shell.Move(m.headingItems(), m.headingEnabled, -1, 1)
}

// headingKey takes a key while the heading menu is open: up and down move, a letter to the next line it starts, enter
// and space act, and esc closes it.
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
	default:
		m.headMenu.at = shell.NextItem(items, m.headingEnabled, m.headMenu.at, k.Text)
	}
	return m, nil
}

// headingAct closes the heading menu and does what its line k says, when it acts, the cursor on the heading.
func (m *Model) headingAct(k int) {
	enabled, at := m.headingEnabled(k), m.headingAt()
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
		m.setPanel(m.panel(m.stage).Grouped(m.panel(m.stage).Heading(m.rows(m.stage)[at]), true))
	}
	m.fit()
}

// headingText gives the screen column a heading's text starts on, past its ▾ or ▸, and the text, as the rows draw it:
// the gutter's cell, two cells a level below the outermost, and the triangle and a space.
func headingText(r row) (int, string) {
	text := strings.TrimLeft(ansi.Strip(r.Cells[0].Values[0]), " ")
	_, text, _ = strings.Cut(text, " ")
	return 1 + 2*(grid.Level(r)-1) + 2, text
}

// headingBox gives the heading menu's lines, the box around the heading's text, which its top edge draws where the
// heading does; and the screen line and column it is drawn from. Once a reload takes the heading away, there are none.
func (m Model) headingBox() (lines []string, top, left int) {
	k := m.headingAt()
	if k < 0 {
		return nil, 0, 0
	}
	x, text := headingText(m.rows(m.stage)[k])
	lines, left = shell.Box(m.headingItems(), m.headingEnabled, m.headMenu.at, text, x, m.paneWidth())
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
