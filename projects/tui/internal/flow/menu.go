// The menus: the app's, which = or a click on = before the title drops down, internal/shell's, the same on every screen
// with refresh all first; and the selected tab's, which : or ; or a click on the tab's : drops down. Each hangs from
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

	"github.com/empowerite/kit/projects/tui/internal/shell"
	"github.com/empowerite/kit/projects/tui/internal/table"
	"github.com/empowerite/kit/projects/tui/internal/tabs"
)

// The glyphs beside the tab's menu's items, the tab maximized and restored, one cell each, the stage loaded again, every
// row expanded and every row collapsed, all the way down and up as ▾ and ▸ are one row's, the selected rows' pages
// opened, every row selected, and the selection swept away; beside the presets' menu's, the one the table stands as,
// two cells wide as the others; beside the app's menu's own, every stage loaded again; and the line under the title of
// a maximized tab, in the tab's color, a cell of it.
const (
	Maximize = "⤢"
	Restore  = "⤡"
	Banner   = "━"
	Expand   = "⏬"
	Collapse = "⏫"
	Checked  = table.Checked
	Reload   = "🔄"
	Browse   = "🌐"
	Every    = "📋"
	Sweep    = table.Sweep
)

// tabItems gives the tab's menu's lines, top to bottom: the stage loaded again; watched, or no longer, as watch.go
// says; the tab maximized, or restored once it is; a rule; the outline expanded, collapsed, and collapsed but for the
// selection, as folds names them; a rule; every row selected and none; a rule; and what acts on the rows selected: on
// backlog, the actions there, then each one's page opened.
func (m Model) tabItems() []shell.Item {
	size := shell.Item{Glyph: Maximize, Name: "Maximize"}
	if m.maximized {
		size = shell.Item{Glyph: Restore, Name: "Restore"}
	}
	items := append([]shell.Item{{Glyph: Reload, Name: "Refresh"}, m.watchItem(m.stage), size, {}}, m.folds()...)
	items = append(items, shell.Item{},
		shell.Item{Glyph: Every, Name: "Select All", Keys: "⌃A"}, shell.Item{Glyph: Sweep, Name: "Clear Selection"},
		shell.Item{})
	return append(append(items, m.actions()...), shell.Item{Glyph: Browse, Name: "Open Selected"})
}

// openMenu drops the tab's menu down, its first item that does something under the keyboard.
func (m *Model) openMenu() {
	if len(m.stages) > 0 {
		m.tabMenu = m.tabMenu.Opened(m.tabItems(), m.enabled)
	}
}

// menuKey takes a key while the tab's menu is open, as a dropdown does, : and ; closing it as esc does, and does what
// the item chosen says.
func (m Model) menuKey(k tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	var chosen int
	if m.tabMenu, chosen = m.tabMenu.Key(k, m.tabItems(), m.enabled, ":", ";"); chosen >= 0 {
		return m, m.pick(chosen)
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
// forge afresh; watch all watches every stage, and stop watching all stops them, as watch.go says; a screen asks kittui to show it in the picture's place, which stays as it is; and quit ends the
// screen.
func (m *Model) chose(c shell.Choice) tea.Cmd {
	switch {
	case c.Quit:
		return m.quit()
	case c.Screen != "":
		return shell.Switch(c.Screen)
	case c.Own == WatchAll, c.Own == StopAll:
		return m.watchAll(c.Own == WatchAll)
	case c.Own == RefreshAll:
		m.quota.Reset()
		m.marks.Reset()
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

// enabled says whether the tab's menu's line k does something to the selected stage's rows now: a fold item while it
// would open or close a node of the outline, as folding says; the rule never does.
func (m Model) enabled(k int) bool {
	s := m.stages[m.stage]
	if action, on := m.acting(m.tabItems()[k].Name); action {
		return on
	}
	if next, fold := table.Folding(&m, m.tabItems()[k].Name); fold {
		// Collapse Others, with nothing selected, would collapse everything, as Collapse All does
		return (m.selected() > 0 || m.tabItems()[k].Name != "Collapse Others") && table.Changes(&m, next)
	}
	switch m.tabItems()[k].Name {
	case "Maximize", "Restore":
		return true
	case "Refresh", WatchIt, StopIt:
		return known[s.name]
	case "Open Selected":
		return len(m.pages()) > 0
	case "Select All":
		return m.selected() < counted(m.every(m.stage))
	case "Clear Selection":
		return m.selected() > 0
	}
	return false
}

// pick does what the tab's menu's item at k says to the selected stage's rows, closes the menu, and gives the rows the
// keyboard, so left and right go on from there; maximize and restore give the rows the bar's lines but one, or take them
// back, and refresh runs the stage again, as refresh says, each leaving the keyboard where it was. An item that is not
// enabled does nothing, and leaves the menu open. It gives what refresh runs.
func (m *Model) pick(k int) tea.Cmd {
	if k < 0 || !m.enabled(k) {
		return nil
	}
	s := &m.stages[m.stage]
	m.tabMenu.Open = false
	if action, _ := m.acting(m.tabItems()[k].Name); action {
		return m.act(m.tabItems()[k].Name)
	}
	if next, fold := table.Folding(m, m.tabItems()[k].Name); fold {
		m.inRows, m.inPanel, m.onPreset = true, false, false
		table.Reoutline(m, next)
		return nil
	}
	switch m.tabItems()[k].Name {
	case "Maximize", "Restore":
		m.maximize(!m.maximized)
		return nil
	case "Refresh":
		return m.refresh(s.name)
	case WatchIt, StopIt:
		return m.toggleWatch(m.stage)
	case "Open Selected":
		return m.openPages(m.pages())
	case "Select All":
		clear(s.Picked)
		m.selectAll()
		return nil
	}
	// Clear Selection, the one item left that acts
	clear(s.Picked)
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

// menuBox gives the tab's menu's lines, as internal/shell draws a menu, hanging from the tab's :, on the bar or the
// title as tabMenuAt says, and the column its left edge is drawn at. The rows, not the bar, bound it, as the lines under
// the bar end where the rows do.
func (m Model) menuBox() (lines []string, left int) {
	lines, left, _ = m.menuBoxIn()
	return lines, left
}

// menuBoxIn gives the tab's menu's lines as menuBox does, and the first item they show: the box ends above the keys
// bar, and a menu taller than the room scrolls with the keyboard.
func (m Model) menuBoxIn() (lines []string, left, first int) {
	return shell.BoxIn(m.tabItems(), m.enabled, m.tabMenu.At, tabs.Menu, m.tabMenuAt(), m.paneWidth(),
		max(3, m.height-1-m.menuTop()))
}

// menuTop gives the screen line the tab's menu's box starts on, its top edge through the tab's glyph: the bar's middle
// one, or, maximized, the title's.
func (m Model) menuTop() int {
	if m.maximized {
		return shell.Top
	}
	return m.top() - 2
}

// tabMenuAt gives the screen column the selected tab's : is drawn at, which drops its menu down: on the bar, as its MenuAt
// says, or, maximized, at the title's end, -1 where the title is clipped short of it.
func (m Model) tabMenuAt() int {
	if !m.maximized {
		return m.bar().MenuAt()
	}
	end := ansi.StringWidth(m.title())
	if end > m.titleWidth() {
		return -1
	}
	return end - ansi.StringWidth(tabs.Menu)
}

// maximize gives the rows the bar's lines but one, the bar no longer drawn, or, restored, takes them back, the bar as it
// was: the cursor, the rows' scroll, and the selection stay as they are, and once the rows have fewer lines, the cursor
// is scrolled to as it moves there.
func (m *Model) maximize(on bool) {
	if m.maximized = on; !on {
		m.fit()
	}
}

// menuClick takes a left click while the tab's menu is open: on an item it shows, it chooses it, and on one not
// enabled it does nothing; anywhere else, it closes the menu.
func (m *Model) menuClick(x, y int) tea.Cmd {
	box, left, first := m.menuBoxIn()
	// the lines between the box's edges show the items from first on
	var chosen int
	m.tabMenu, chosen = m.tabMenu.ClickIn(x, y, m.menuTop(), left, ansi.StringWidth(box[0]), first, len(box)-2,
		m.tabItems(), m.enabled)
	if chosen < 0 {
		return nil
	}
	return m.pick(chosen)
}
