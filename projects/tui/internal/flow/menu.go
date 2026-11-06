// The menus: the app's, which = or a click on ≡ before the title drops down, internal/shell's, the same on every screen
// with refresh all first; and the selected tab's, which : or ; or a click on the tab's ⋮ drops down. Each hangs from
// its glyph, which takes a cell of the box's top edge, two in from its left corner, or from its right one when the box
// would run off the screen. While one is open it has the keyboard: up and down move, a letter moves to the next item
// that starts with it, enter and space choose, and esc or the key that opened it closes it; a click on an item chooses
// it, and a click anywhere else closes it. The tab's choices act on the selected stage's rows; an item that would
// change nothing is dim, and the keyboard passes it by.
package flow

import (
	"slices"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/empowerite/kit/projects/tui/internal/grid"
	"github.com/empowerite/kit/projects/tui/internal/shell"
	"github.com/empowerite/kit/projects/tui/internal/tabs"
)

// The glyphs beside the tab's menu's items, the stage loaded again, every row expanded and every row collapsed, all
// the way down and up as ▾ and ▸ are one row's, the headers shown, two cells wide as the others, the selected rows'
// pages opened, every row selected, and the selection swept away; and beside the app's menu's own, every stage loaded
// again.
const (
	Expand   = "⏬"
	Collapse = "⏫"
	Checked  = "✅"
	Reload   = "🔄"
	Browse   = "🌐"
	Every    = "📋"
	Sweep    = "🧹"
)

// ownItems are the picture's own in the app's menu: every stage loaded again, reading the forge afresh.
var ownItems = []shell.Item{{Glyph: Reload, Name: "Refresh All"}}

// tabItems gives the tab's menu's lines, top to bottom: the stage loaded again, a rule, every row that hides something
// expanded and every row collapsed, or, while rows are selected, the selected rows expanded and collapsed and the rows
// not selected collapsed, a rule, the headers, checked while the selected stage shows them, a rule, every row selected
// and none, a rule, and what acts on the rows selected: on backlog, the actions there, then each one's page opened.
func (m Model) tabItems() []shell.Item {
	headers := ""
	if len(m.stages) > 0 && m.stages[m.stage].headers {
		headers = Checked
	}
	folds := []shell.Item{{Glyph: Expand, Name: "Expand All"}, {Glyph: Collapse, Name: "Collapse All"}}
	if m.selected() > 0 {
		folds = []shell.Item{{Glyph: Expand, Name: "Expand Selected"}, {Glyph: Collapse, Name: "Collapse Selected"},
			{Glyph: Collapse, Name: "Collapse Others"}}
	}
	items := append([]shell.Item{{Glyph: Reload, Name: "Refresh"}, {}}, folds...)
	items = append(items, shell.Item{}, shell.Item{Glyph: headers, Name: "Show Headers"}, shell.Item{},
		shell.Item{Glyph: Every, Name: "Select All"}, shell.Item{Glyph: Sweep, Name: "Clear Selection"}, shell.Item{})
	return append(append(items, m.actions()...), shell.Item{Glyph: Browse, Name: "Open Selected"})
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
	if action, on := m.acting(m.tabItems()[k].Name); action {
		return on
	}
	switch m.tabItems()[k].Name {
	case "Refresh":
		return known[s.name]
	case "Show Headers":
		return len(fieldsOf(s.name)) > 0
	case "Open Selected":
		return len(m.pages()) > 0
	case "Select All":
		return m.selected() < counted(m.rows(m.stage))
	case "Clear Selection":
		return m.selected() > 0
	case "":
		return false
	case "Collapse Others":
		return slices.ContainsFunc(rows, func(r row) bool { return s.open[r.Key] && !s.picked[r.Key] })
	case "Expand Selected", "Collapse Selected":
		for j, r := range rows {
			opens := grid.Opens(rows, m.names(m.stage), j, m.gridWidth(m.stage))
			if s.picked[r.Key] && opens && s.open[r.Key] == (m.tabItems()[k].Name == "Collapse Selected") {
				return true
			}
		}
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
			if !s.open[r.Key] && grid.Opens(rows, m.names(m.stage), j, m.gridWidth(m.stage)) {
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
	if action, _ := m.acting(m.tabItems()[k].Name); action {
		return m.act(m.tabItems()[k].Name)
	}
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
	case "Select All":
		for _, r := range rows {
			if !r.Aside {
				s.picked[r.Key] = true
			}
		}
		return nil
	case "Clear Selection":
		clear(s.picked)
		return nil
	case "Expand Selected", "Collapse Selected":
		for j, r := range rows {
			if s.picked[r.Key] && grid.Opens(rows, m.names(m.stage), j, m.gridWidth(m.stage)) {
				s.open[r.Key] = m.tabItems()[k].Name == "Expand Selected"
			}
		}
		m.inRows, m.inPanel, m.onPreset = true, false, false
		m.fit()
		return nil
	case "Collapse Others":
		for _, r := range rows {
			if !s.picked[r.Key] {
				delete(s.open, r.Key)
			}
		}
		m.inRows, m.inPanel, m.onPreset = true, false, false
		m.fit()
		return nil
	}
	m.inRows, m.inPanel, m.onPreset = true, false, false
	switch m.tabItems()[k].Name {
	case "Expand All":
		for j, r := range rows {
			if grid.Opens(rows, m.names(m.stage), j, m.gridWidth(m.stage)) {
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
	lines, left, _ = m.menuBoxIn()
	return lines, left
}

// menuBoxIn gives the tab's menu's lines as menuBox does, and the first item they show: the box ends above the keys
// bar, and a menu taller than the room scrolls with the keyboard.
func (m Model) menuBoxIn() (lines []string, left, first int) {
	return shell.BoxIn(m.tabItems(), m.enabled, m.menuAt, tabs.Menu, m.bar().MenuAt(), m.paneWidth(),
		max(3, m.height-1-menuTop))
}

// menuTop is the screen line the tab's menu's box starts on: the bar's middle one, its top edge through the tab's
// glyph.
const menuTop = paneTop - 2

// overlay draws the open menu over the screen's lines: the tab's from menuTop, the app's from the title; and what a
// backlog action asks, over the rows' top. The menus that hang over the sidebar are drawn after it, by float.
func (m Model) overlay(lines []string) []string {
	if m.menu {
		box, left := m.menuBox()
		lines = shell.Over(lines, box, menuTop, left)
	}
	if m.ask != nil {
		box, left := m.askBox()
		lines = shell.Over(lines, box, paneTop+1, left)
	}
	return m.appMenu.Overlay(lines, m.paneWidth())
}

// menuClick takes a left click while the tab's menu is open: on an item it shows, it chooses it, and on one not
// enabled it does nothing; anywhere else, it closes the menu.
func (m *Model) menuClick(x, y int) tea.Cmd {
	box, left, first := m.menuBoxIn()
	// the lines between the box's edges show the items from first on
	if k := y - menuTop - 1; x > left && x < left+ansi.StringWidth(box[0])-1 && k >= 0 && k < len(box)-2 {
		return m.pick(first + k)
	}
	m.menu = false
	return nil
}

// float draws the menus that hang over the sidebar, once the sidebar is laid out beside the rows, which would clip
// them: the presets' menu under the preset line, a field's popover, and the searches remembered under the search's ▾.
func (m Model) float(lines []string) []string {
	if m.presetMenu {
		box, top, left := m.presetBox()
		lines = shell.Over(lines, box, top, left)
	}
	if m.pop.open {
		box, top, left := m.popBox()
		lines = shell.Over(lines, box, top, left)
	}
	return m.recent.Overlay(lines, m.recallAt(), m.width)
}
