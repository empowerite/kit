// The menus: the app's, which = or a click on ≡ before the title drops down, internal/shell's, the same on every screen
// with refresh all first; and the selected tab's, which : or ; or a click on the tab's ⋮ drops down. Each hangs from
// its glyph, which takes a cell of the box's top edge, two in from its left corner, or from its right one when the box
// would run off the screen. While one is open it has the keyboard: up and down move, a letter moves to the next item
// that starts with it, enter and space choose, and esc or the key that opened it closes it; a click on an item chooses
// it, and a click anywhere else closes it. The tab's choices act on the selected stage's rows; an item that would
// change nothing is dim, and the keyboard passes it by.
package flow

import (
	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/empowerite/kit/projects/tui/internal/grid"
	"github.com/empowerite/kit/projects/tui/internal/shell"
	"github.com/empowerite/kit/projects/tui/internal/tabs"
)

// The glyphs beside the tab's menu's items, every row expanded and every row collapsed, all the way down and up as ▾
// and ▸ are one row's, and the headers shown; and beside the app's menu's own, every stage loaded again.
const (
	Expand   = "⏬"
	Collapse = "⏫"
	Checked  = "✓"
	Reload   = "🔄"
)

// ownItems are the picture's own in the app's menu: every stage loaded again, reading the forge afresh.
var ownItems = []shell.Item{{Glyph: Reload, Name: "Refresh All"}}

// tabItems gives the tab's menu's lines, top to bottom: every row that hides something expanded, every row collapsed,
// a rule, and the headers, checked while the selected stage shows them.
func (m Model) tabItems() []shell.Item {
	headers := ""
	if len(m.stages) > 0 && m.stages[m.stage].headers {
		headers = Checked
	}
	return []shell.Item{{Glyph: Expand, Name: "Expand All"}, {Glyph: Collapse, Name: "Collapse All"}, {},
		{Glyph: headers, Name: "Show Headers"}}
}

// openMenu drops the tab's menu down, its first item that does something under the keyboard.
func (m *Model) openMenu() {
	if len(m.stages) > 0 {
		m.menu, m.menuAt = true, shell.Move(m.tabItems(), m.enabled, -1, 1)
	}
}

// menuKey takes a key while the tab's menu is open.
func (m Model) menuKey(k tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch key := k.String(); key {
	case "up":
		m.menuAt = shell.Move(m.tabItems(), m.enabled, m.menuAt, -1)
	case "down":
		m.menuAt = shell.Move(m.tabItems(), m.enabled, m.menuAt, 1)
	case "enter", "space":
		m.pick(m.menuAt)
	case "esc", ":", ";":
		m.menu = false
	default:
		m.menuAt = shell.NextItem(m.tabItems(), m.enabled, m.menuAt, k.Text)
	}
	return m, nil
}

// appKey takes a key while the app's menu is open, and does what a choice from it asks.
func (m Model) appKey(k tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	var c shell.Choice
	m.appMenu, c = m.appMenu.Key(k)
	return m, m.chose(c)
}

// chose does what a choice from the app's menu asks: refresh all runs every stage kittui draws again, reading the
// forge afresh; a screen asks kittui to show it in the picture's place, which stays as it is; and quit ends the
// screen.
func (m *Model) chose(c shell.Choice) tea.Cmd {
	switch {
	case c.Quit:
		return m.quit()
	case c.Screen != "":
		return shell.Switch(c.Screen)
	case c.Own == "Refresh All":
		var cmds []tea.Cmd
		for _, s := range m.stages {
			if known[s.name] {
				cmds = append(cmds, m.restart(s.name))
			}
		}
		return tea.Batch(cmds...)
	}
	return nil
}

// enabled says whether the tab's menu's line k does something to the selected stage's rows now: expand all while a
// row that hides something is collapsed, collapse all while one of its rows is expanded, and show headers on a stage
// whose rows have columns; the rule never does.
func (m Model) enabled(k int) bool {
	s, rows := m.stages[m.stage], m.rows(m.stage)
	switch m.tabItems()[k].Name {
	case "Show Headers":
		return len(columns[s.name]) > 0
	case "":
		return false
	}
	for j, r := range rows {
		switch m.tabItems()[k].Name {
		case "Expand All":
			if !s.open[r.Key] && grid.Opens(rows, j, m.paneWidth()) {
				return true
			}
		case "Collapse All":
			if s.open[r.Key] {
				return true
			}
		}
	}
	return false
}

// pick does what the tab's menu's item at k says to the selected stage's rows, closes the menu, and gives the rows
// the keyboard, so left and right go on from there; show headers shows them over the rows, or hides them, and leaves
// the keyboard where it was. An item that is not enabled does nothing, and leaves the menu open.
func (m *Model) pick(k int) {
	if k < 0 || !m.enabled(k) {
		return
	}
	s, rows := &m.stages[m.stage], m.rows(m.stage)
	m.menu = false
	if m.tabItems()[k].Name == "Show Headers" {
		s.headers = !s.headers
		m.fit()
		return
	}
	m.inRows, m.inPanel = true, false
	switch m.tabItems()[k].Name {
	case "Expand All":
		for j, r := range rows {
			if grid.Opens(rows, j, m.paneWidth()) {
				s.open[r.Key] = true
			}
		}
	case "Collapse All":
		clear(s.open)
	}
	m.fit()
}

// menuBox gives the tab's menu's lines, as internal/shell draws a menu, hanging from the tab's ⋮, and the column its
// left edge is drawn at. The rows, not the bar, bound it, as the lines under the bar end where the rows do.
func (m Model) menuBox() (lines []string, left int) {
	return shell.Box(m.tabItems(), m.enabled, m.menuAt, tabs.Menu, m.bar().MenuAt(), m.paneWidth())
}

// menuTop is the screen line the tab's menu's box starts on: the bar's middle one, its top edge through the tab's
// glyph.
const menuTop = paneTop - 2

// overlay draws the open menu over the screen's lines: the tab's from menuTop, the app's from the title.
func (m Model) overlay(lines []string) []string {
	if m.menu {
		box, left := m.menuBox()
		lines = shell.Over(lines, box, menuTop, left)
	}
	return m.appMenu.Overlay(lines, m.paneWidth())
}

// menuClick takes a left click while the tab's menu is open: on an item, it chooses it, and on one not enabled it
// does nothing; anywhere else, it closes the menu.
func (m *Model) menuClick(x, y int) {
	box, left := m.menuBox()
	k := y - menuTop - 1
	if x > left && x < left+ansi.StringWidth(box[0])-1 && k >= 0 && k < len(m.tabItems()) {
		m.pick(k)
		return
	}
	m.menu = false
}
