// The selected step's menu, as every screen with tabs has one: : or ; or a click anywhere on the selected tab drops it
// down from the tab's :, on the bar, or, maximized, at the title's end. While it is open it has the keyboard: up and
// down move, a letter moves to the next item that starts with it, enter and space choose, and esc or the key that
// opened it closes it; a click on an item chooses it, and a click anywhere else closes it. An item that would change
// nothing now is dim, and the keyboard passes it by; a line with a key shows it at its right, and the key does the same
// with the menu closed.
//
// Refresh runs the checks again, as after a fix. Maximize gives the pane the bar's lines but one, the bar no longer
// drawn, the selected tab on the title line instead, and Restore takes them back, for the session, whatever step is
// selected. A step drawn as a table adds its folds, as > and < and table.Folding name them.
package welcome

import (
	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/empowerite/kit/projects/tui/internal/shell"
	"github.com/empowerite/kit/projects/tui/internal/table"
	"github.com/empowerite/kit/projects/tui/internal/tabs"
)

// refreshName is the tab's menu's item that runs the checks again. It has no key: r runs a check's fix here.
const refreshName = "Refresh"

// tabItems gives the tab's menu's lines, top to bottom: Refresh, then Maximize, or Restore once maximized; and on a
// step drawn as a table, a rule and its folds.
func (m Model) tabItems() []shell.Item {
	items := []shell.Item{{Glyph: tabs.Reload, Name: refreshName}, tabs.SizeItem(m.maximized)}
	if !m.gridded() {
		return items
	}
	return append(append(items, shell.Item{}), tabs.FoldItems()...)
}

// enabled says whether the tab's menu's line k does something now: Refresh, Maximize, and Restore always; a fold while
// it would show or hide a node of the table's outline, as table.Changes says. The rule never does.
func (m Model) enabled(k int) bool {
	switch name := m.tabItems()[k].Name; name {
	case refreshName, "Maximize", "Restore":
		return true
	case "":
		return false
	default:
		next, _ := table.Folding(m.sheet(), name)
		return table.Changes(m.sheet(), next)
	}
}

// pick does what the tab's menu's line k says, and closes the menu: Refresh runs the checks again; Maximize and Restore
// give the pane the bar's lines or take them back; and a fold shows or hides the table's outline, the table's rows then
// with the keyboard. The line is always one enabled: the menu's keys and clicks, and a shortcut, choose no other.
func (m *Model) pick(k int) tea.Cmd {
	m.tabMenu.Open = false
	switch name := m.tabItems()[k].Name; name {
	case refreshName:
		return m.again()
	case "Maximize", "Restore":
		m.maximize(!m.maximized)
		return nil
	default:
		next, _ := table.Folding(m.sheet(), name)
		m.sheet().FocusRows()
		m.inPane = m.sheet().Focused()
		table.Reoutline(m.sheet(), next)
		return nil
	}
}

// shortcut gives the tab's menu's line whose key the press is, when it acts now, or -1; with the menu closed, the key
// does what choosing the line does.
func (m Model) shortcut(press string) int { return shell.Shortcut(m.tabItems(), m.enabled, press) }

// openMenu drops the tab's menu down, its first line that does something under the keyboard.
func (m *Model) openMenu() { m.tabMenu = m.tabMenu.Opened(m.tabItems(), m.enabled) }

// menuKey takes a key while the tab's menu is open, as a dropdown does, : and ; closing it as esc does, and does what
// the line chosen says.
func (m Model) menuKey(k tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	var chosen int
	if m.tabMenu, chosen = m.tabMenu.Key(k, m.tabItems(), m.enabled, ":", ";"); chosen >= 0 {
		return m, m.pick(chosen)
	}
	return m, nil
}

// menuClick takes a left click while the tab's menu is open: on a line it shows, it chooses it, and on one not enabled
// it does nothing; anywhere else, it closes the menu.
func (m *Model) menuClick(x, y int) tea.Cmd {
	box, left, first := m.menuBoxIn()
	var chosen int
	m.tabMenu, chosen = m.tabMenu.ClickIn(x, y, m.menuTop(), left, ansi.StringWidth(box[0]), first, len(box)-2,
		m.tabItems(), m.enabled)
	if chosen < 0 {
		return nil
	}
	return m.pick(chosen)
}

// menuBoxIn gives the tab's menu's lines, as internal/shell draws a menu, hanging from the tab's :, as tabMenuAt says,
// the column its left edge is drawn at, and the first line they show: the box ends above the keys bar, and a menu
// taller than the room scrolls with the keyboard.
func (m Model) menuBoxIn() (lines []string, left, first int) {
	return shell.BoxIn(m.tabItems(), m.enabled, m.tabMenu.At, tabs.Menu, m.tabMenuAt(), m.width,
		max(3, m.height-1-m.menuTop()))
}

// menuTop gives the screen line the tab's menu's box starts on, its top edge through the tab's :: the bar's middle
// line, or, maximized, the title's.
func (m Model) menuTop() int {
	if m.maximized {
		return shell.Top
	}
	return m.bar().Row + 1
}

// tabMenuAt gives the screen column the selected tab's : is drawn at: on the bar, as its MenuAt says, or, maximized,
// at the title's end, -1 where the title is clipped short of it.
func (m Model) tabMenuAt() int {
	if !m.maximized {
		return m.bar().MenuAt()
	}
	if _, until, whole := m.menuBar().OnSpan(m.width); whole {
		return until - ansi.StringWidth(tabs.Menu)
	}
	return -1
}

// onMenu says whether a click at x, y is on the selected tab, which drops its menu down, as its : does: anywhere on its
// part of the bar, on any of the bar's lines; or, maximized, anywhere on its label at the title's end, as much of it as
// the title shows.
func (m Model) onMenu(x, y int) bool {
	if !m.maximized {
		return m.bar().OnSelected(x, y)
	}
	from, until, _ := m.menuBar().OnSpan(m.width)
	return y == shell.Top && from >= 0 && x >= from && x < until
}

// barTakes gives what the bar makes of a message, as its Update says, and whether it took it; while maximized, the bar
// is not drawn, so a click is never on it, and tab and shift-tab have no tabs to move along: it takes a tab's number
// alone.
func (m Model) barTakes(msg tea.Msg) (tabs.Bar, bool) {
	if k, ok := msg.(tea.KeyPressMsg); m.maximized && (!ok || k.String() == "tab" || k.String() == "shift+tab") {
		return tabs.Bar{}, false
	}
	return m.bar().Update(msg)
}

// maximize gives the pane the bar's lines but one, the bar no longer drawn, or, restored, takes them back, and places
// the steps' tables in the pane as it now stands.
func (m *Model) maximize(on bool) {
	m.maximized = on
	m.placeSheets()
	m.fit()
}

// paneTop gives the screen line the pane starts on: under the bar, which is open there to the selected tab; or,
// maximized, under the title and the banner.
func (m Model) paneTop() int {
	if m.maximized {
		return 2
	}
	return 1 + tabs.Height
}

// banner gives the line under the title of a maximized tab: shell.Rule, as wide as the screen, in the accent the bar
// draws the selected tab in, as every screen with no tab bar under its menu bar draws it.
func (m Model) banner() string { return shell.RuleLine(m.width, m.bar().Color) }
