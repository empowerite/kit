// The menus: the app's, which = or a click on ≡ before the title drops down, and the selected tab's, which : or ; or a
// click on the tab's ⋮ drops down. Each hangs from its glyph, which takes a cell of the box's top edge, two in from
// its left corner, or from its right one when the box would run off the screen. While one is open it has the
// keyboard: up and down move, a letter moves to the next item that starts with it, enter and space choose, and esc or
// the key that opened it closes it; a click on an item chooses it, and a click anywhere else closes it. The tab's
// choices act on the selected stage's rows, the app's on all of kittui; an item that would change nothing is dim, and
// the keyboard passes it by.
package flow

import (
	"slices"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/empowerite/kit/projects/tui/internal/grid"
	"github.com/empowerite/kit/projects/tui/internal/pane"
	"github.com/empowerite/kit/projects/tui/internal/style"
	"github.com/empowerite/kit/projects/tui/internal/tabs"
)

// App is the glyph before the title, which drops the app's menu down: three bars, a hamburger, as the tab's menu is
// its ⋮.
const App = "≡"

// The glyphs beside the menus' items: every row expanded, and every row collapsed, all the way down and up as ▾ and
// ▸ are one row's; every stage loaded again; the settings; and the way out.
const (
	Expand   = "⏬"
	Collapse = "⏫"
	Reload   = "🔄"
	Settings = "🔧"
	Leave    = "🚪"
)

// menuItem is one line of a menu: a glyph, which may be none, and the name it is chosen by; a line with no name is the
// rule between groups of items.
type menuItem struct{ glyph, name string }

// tabItems are the tab's menu's lines, top to bottom: every row that hides something expanded, and every row
// collapsed. appItems are the app's: every stage loaded again, reading the forge afresh, a rule, the settings, which
// kittui does not have yet, a rule, and quit.
var (
	tabItems = []menuItem{{Expand, "Expand All"}, {Collapse, "Collapse All"}}
	appItems = []menuItem{{Reload, "Refresh All"}, {}, {Settings, "Settings…"}, {}, {Leave, "Quit"}}
)

// menuKeys is the keys bar while a menu is open.
const menuKeys = "↑↓ move · ⏎ choose · esc close"

// items gives the open menu's lines: the app's, or the tab's.
func (m Model) items() []menuItem {
	if m.app {
		return appItems
	}
	return tabItems
}

// openMenu drops a menu down, the app's or the tab's, its first item that does something under the keyboard.
func (m *Model) openMenu(app bool) {
	if len(m.stages) > 0 {
		m.menu, m.app = true, app
		m.menuAt = m.move(-1, 1)
	}
}

// menuKey takes a key while a menu is open.
func (m Model) menuKey(k tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch key := k.String(); {
	case key == "up":
		m.menuAt = m.move(m.menuAt, -1)
	case key == "down":
		m.menuAt = m.move(m.menuAt, 1)
	case key == "enter" || key == "space":
		return m, m.pick(m.menuAt)
	case key == "esc", m.app && key == "=", !m.app && (key == ":" || key == ";"):
		m.menu = false
	default:
		m.menuAt = m.nextItem(m.menuAt, k.Text)
	}
	return m, nil
}

// enabled says whether the open menu's line k does something now: expand all while a row of the selected stage that
// hides something is collapsed, collapse all while one of its rows is expanded, refresh all and quit always; the
// settings and the rules never do.
func (m Model) enabled(k int) bool {
	switch m.items()[k].name {
	case "Refresh All", "Quit":
		return true
	case "Expand All", "Collapse All":
	default:
		return false
	}
	s, rows := m.stages[m.stage], m.rows(m.stage)
	for j, r := range rows {
		switch m.items()[k].name {
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

// move gives the item from k on, by step, that is enabled, stopping at the first and the last; k itself when none
// is.
func (m Model) move(k, step int) int {
	for j := k + step; j >= 0 && j < len(m.items()); j += step {
		if m.enabled(j) {
			return j
		}
	}
	return k
}

// nextItem gives the enabled item after the one at k, round from the last to the first, that starts with the letter
// typed, regardless of case; k itself when none does, or nothing was typed.
func (m Model) nextItem(k int, typed string) int {
	if typed == "" {
		return k
	}
	items := m.items()
	for step := 1; step <= len(items); step++ {
		j := (k + step) % len(items)
		if m.enabled(j) && strings.HasPrefix(strings.ToLower(items[j].name), strings.ToLower(typed)) {
			return j
		}
	}
	return k
}

// pick does what the open menu's item at k says, closes the menu, and gives the command it runs: the tab's choices
// change the selected stage's rows and give them the keyboard, so left and right go on from there; refresh all runs
// every stage again, reading the forge afresh, and quit ends the screen. An item that is not enabled does nothing,
// and leaves the menu open.
func (m *Model) pick(k int) tea.Cmd {
	if k < 0 || !m.enabled(k) {
		return nil
	}
	m.menu = false
	switch m.items()[k].name {
	case "Refresh All":
		var cmds []tea.Cmd
		for _, s := range m.stages {
			if known[s.name] {
				cmds = append(cmds, m.restart(s.name))
			}
		}
		return tea.Batch(cmds...)
	case "Quit":
		return m.quit()
	}
	m.inRows, m.inPanel = true, false
	s, rows := &m.stages[m.stage], m.rows(m.stage)
	switch m.items()[k].name {
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
	return nil
}

// menuBox gives the open menu's lines, a rounded box in the menu's color around its items, each after the room for a
// glyph, the one under the keyboard inverse and the ones not enabled dim, a dim rule between groups, a cell short of
// either side; and the screen column its left edge is drawn at: two before its glyph's, or the screen's first, so the
// glyph starts at the top edge's third cell; when the box would run past the rows that way, as on the rightmost tab,
// the column that puts the glyph two cells before the right corner instead, or nearer when the rows end there; and
// against the rows' right when the bar does not draw the glyph. The rows, not the bar, bound it, as the lines under the
// bar end where the rows do.
func (m Model) menuBox() (lines []string, left int) {
	names := make([]string, len(m.items()))
	for k, item := range m.items() {
		names[k] = item.name
	}
	inner := pane.Width(names) + 5
	mark, glyph := tabs.Menu, m.bar().MenuAt()
	if m.app {
		mark, glyph = App, appAt
	}
	width, room, cells := inner+2, m.paneWidth(), ansi.StringWidth(mark)
	switch {
	case glyph < 0:
		left = room - width
	case glyph-2+width > room:
		left = min(glyph+cells+2, room) - width
	default:
		left = glyph - 2
	}
	left = max(0, left)
	top := []rune("╭" + strings.Repeat("─", inner) + "╮")
	if c := glyph - left; glyph >= 0 && c >= 0 && c+cells <= width {
		// the glyph takes its cells of the top edge, the edge either side of it drawn half its length; every cell of
		// the edge is a rune of its own
		if c > 1 {
			top[c-1] = '╴'
		}
		if c+cells < width-1 {
			top[c+cells] = '╶'
		}
		top = slices.Concat(top[:c], []rune(mark), top[c+cells:])
	}
	lines = append(lines, style.Menu(string(top)))
	for k, item := range m.items() {
		text := pane.Pad(" "+pane.Pad(item.glyph, 2, false)+" "+item.name+" ", inner, false)
		switch {
		case item.name == "":
			lines = append(lines, style.Menu("│")+" "+style.Off(strings.Repeat("─", inner-2))+" "+style.Menu("│"))
			continue
		case !m.enabled(k):
			// dim, and never under the keyboard, even when the rows changed under an open menu
			text = style.Off(text)
		case k == m.menuAt:
			text = style.Cursor(text)
		}
		lines = append(lines, style.Menu("│")+text+style.Menu("│"))
	}
	lines = append(lines, style.Menu("╰"+strings.Repeat("─", inner)+"╯"))
	return lines, left
}

// menuTop is the screen line the tab's menu's box starts on: the bar's middle one, its top edge through the tab's
// glyph. The app's starts on the title's, through App, which stands at appAt.
const (
	menuTop = paneTop - 2
	appTop  = 0
	appAt   = 1
)

// boxTop gives the screen line the open menu's box starts on.
func (m Model) boxTop() int {
	if m.app {
		return appTop
	}
	return menuTop
}

// overlay draws the open menu over the screen's lines, from its top; what the box covers is hidden, and what is
// beside it stays in its columns, a two-cell glyph the box cuts in half giving way to a space.
func (m Model) overlay(lines []string) []string {
	if !m.menu {
		return lines
	}
	box, left := m.menuBox()
	for k, b := range box {
		if i := m.boxTop() + k; i < len(lines) {
			lines[i] = splice(lines[i], b, left)
		}
	}
	return lines
}

// splice draws box over line from the column left, keeping every cell of line outside it in its column: a two-cell
// glyph the box's edge cuts in half gives way to a space.
func splice(line, box string, left int) string {
	right := left + ansi.StringWidth(box)
	line = pane.Pad(line, right, false)
	return pane.Cells(line, 0, left) + box + pane.Cells(line, right, ansi.StringWidth(line))
}

// menuClick takes a left click while a menu is open: on an item, it chooses it, and on one not enabled, or a rule, it
// does nothing; anywhere else, it closes the menu.
func (m *Model) menuClick(x, y int) tea.Cmd {
	box, left := m.menuBox()
	k := y - m.boxTop() - 1
	if x > left && x < left+ansi.StringWidth(box[0])-1 && k >= 0 && k < len(m.items()) {
		return m.pick(k)
	}
	m.menu = false
	return nil
}
