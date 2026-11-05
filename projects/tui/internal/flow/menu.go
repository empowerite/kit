// The menus: the app's, which = or a click on ≡ before the title drops down, internal/shell's, the same on every screen
// with refresh all first; and the selected tab's, which : or ; or a click on the tab's ⋮ drops down. Each hangs from
// its glyph, which takes a cell of the box's top edge, two in from its left corner, or from its right one when the box
// would run off the screen. While one is open it has the keyboard: up and down move, a letter moves to the next item
// that starts with it, enter and space choose, and esc or the key that opened it closes it; a click on an item chooses
// it, and a click anywhere else closes it. The tab's choices act on the selected stage's rows; an item that would
// change nothing is dim, and the keyboard passes it by.
package flow

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/empowerite/kit/projects/tui/internal/grid"
	"github.com/empowerite/kit/projects/tui/internal/shell"
	"github.com/empowerite/kit/projects/tui/internal/tabs"
)

// The glyphs beside the tab's menu's items, the stage loaded again, every row expanded and every row collapsed, all
// the way down and up as ▾ and ▸ are one row's, the headers shown, two cells wide as the others, the selected rows'
// pages opened, and the selection swept away; and beside the app's menu's own, every stage loaded again.
const (
	Expand   = "⏬"
	Collapse = "⏫"
	Checked  = "✅"
	Reload   = "🔄"
	Browse   = "🌐"
	Sweep    = "🧹"
)

// ownItems are the picture's own in the app's menu: every stage loaded again, reading the forge afresh.
var ownItems = []shell.Item{{Glyph: Reload, Name: "Refresh All"}}

// tabItems gives the tab's menu's lines, top to bottom: the stage loaded again, a rule, every row that hides something
// expanded, every row collapsed, a rule, the headers, checked while the selected stage shows them, a rule, and the
// rows selected: each one's page opened, and none selected.
func (m Model) tabItems() []shell.Item {
	headers := ""
	if len(m.stages) > 0 && m.stages[m.stage].headers {
		headers = Checked
	}
	return []shell.Item{{Glyph: Reload, Name: "Refresh"}, {}, {Glyph: Expand, Name: "Expand All"},
		{Glyph: Collapse, Name: "Collapse All"}, {}, {Glyph: headers, Name: "Show Headers"}, {},
		{Glyph: Browse, Name: "Open Selected"}, {Glyph: Sweep, Name: "Clear Selection"}}
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
		return m, m.pick(m.menuAt)
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
// row that hides something is collapsed, or the header is, collapse all while one of its rows is expanded, or the
// header is, and show headers on a stage whose rows have columns; the rule never does.
func (m Model) enabled(k int) bool {
	s, rows := m.stages[m.stage], m.rows(m.stage)
	_, hides := m.header()
	switch m.tabItems()[k].Name {
	case "Refresh":
		return known[s.name]
	case "Show Headers":
		return len(columns[s.name]) > 0
	case "Open Selected":
		return len(m.pages()) > 0
	case "Clear Selection":
		return m.selected() > 0
	case "":
		return false
	case "Expand All":
		if hides && !s.headed {
			return true
		}
	case "Collapse All":
		if hides && s.headed {
			return true
		}
	}
	for j, r := range rows {
		switch m.tabItems()[k].Name {
		case "Expand All":
			if !s.open[r.Key] && grid.Opens(rows, m.names(m.stage), j, m.paneWidth()) {
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
// the keyboard, so left and right go on from there; refresh runs the stage again, as refresh says, and show headers
// shows them over the rows, or hides them, each leaving the keyboard where it was. An item that is not enabled does
// nothing, and leaves the menu open. It gives what refresh runs.
func (m *Model) pick(k int) tea.Cmd {
	if k < 0 || !m.enabled(k) {
		return nil
	}
	s, rows := &m.stages[m.stage], m.rows(m.stage)
	m.menu = false
	switch m.tabItems()[k].Name {
	case "Show Headers":
		s.headers = !s.headers
		m.fit()
		return nil
	case "Refresh":
		return m.refresh(s.name)
	case "Open Selected":
		var cmds []tea.Cmd
		for _, page := range m.pages() {
			cmds = append(cmds, m.openPage(page))
		}
		return tea.Batch(cmds...)
	case "Clear Selection":
		clear(s.picked)
		return nil
	}
	m.inRows, m.inPanel = true, false
	switch m.tabItems()[k].Name {
	case "Expand All":
		for j, r := range rows {
			if grid.Opens(rows, m.names(m.stage), j, m.paneWidth()) {
				s.open[r.Key] = true
			}
		}
		_, s.headed = m.header()
	case "Collapse All":
		clear(s.open)
		s.headed = false
	}
	m.fit()
	return nil
}

// pages gives the page each selected row of the selected stage leads to: its first cell's link that opens a page;
// a row with none gives none.
func (m Model) pages() []string {
	var out []string
	for _, r := range m.selectedRows() {
		for _, c := range r.Cells {
			if strings.Contains(c.Link, "://") {
				out = append(out, c.Link)
				break
			}
		}
	}
	return out
}

// refresh runs a stage's command again, reading the forge afresh, and shows it loading until it answers; on a stage
// that shares kit's one read of the open issues and pull requests, every stage in the picture that shares it, so they
// stay one snapshot.
func (m *Model) refresh(name string) tea.Cmd {
	if !shared[name] {
		return m.restart(name)
	}
	var cmds []tea.Cmd
	for _, s := range m.stages {
		if shared[s.name] {
			cmds = append(cmds, m.restart(s.name))
		}
	}
	return tea.Batch(cmds...)
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
func (m *Model) menuClick(x, y int) tea.Cmd {
	box, left := m.menuBox()
	k := y - menuTop - 1
	if x > left && x < left+ansi.StringWidth(box[0])-1 && k >= 0 && k < len(m.tabItems()) {
		return m.pick(k)
	}
	m.menu = false
	return nil
}
