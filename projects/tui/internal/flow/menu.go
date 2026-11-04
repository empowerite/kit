// The selected tab's menu: = or a click on the tab's menu glyph drops it down from the glyph, which takes a cell of the
// box's top edge, two in from its left corner, or from its right one when the box would run off the screen. While it is
// open it has the keyboard: up and down move, a letter moves to the next item that starts with it, enter and space
// choose, and esc or = closes it; a click on an item chooses it, and a click anywhere else closes it. Each choice acts
// on the selected stage's rows; an item that would change nothing is dim, and the keyboard passes it by.
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

// The glyphs beside the menu's items: the settings, every row expanded, and every row collapsed, all the way down and
// up as ▾ and ▸ are one row's.
const (
	Settings = "🔧"
	Expand   = "⏬"
	Collapse = "⏫"
)

// menuItem is one line of the menu: a glyph, which may be none, and the name it is chosen by; a line with no name is
// the rule between groups of items.
type menuItem struct{ glyph, name string }

// menuItems are the menu's lines, top to bottom: every row that hides something expanded, every row collapsed, a
// rule, and the settings, which kittui does not have yet.
var menuItems = []menuItem{{Expand, "Expand All"}, {Collapse, "Collapse All"}, {}, {Settings, "Settings…"}}

// menuKeys is the keys bar while the menu is open.
const menuKeys = "↑↓ move · ⏎ choose · esc close"

// openMenu drops the menu down, its first item that does something under the keyboard.
func (m *Model) openMenu() {
	if len(m.stages) > 0 {
		m.menu, m.menuAt = true, m.move(-1, 1)
	}
}

// menuKey takes a key while the menu is open.
func (m Model) menuKey(k tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch k.String() {
	case "up":
		m.menuAt = m.move(m.menuAt, -1)
	case "down":
		m.menuAt = m.move(m.menuAt, 1)
	case "enter", "space":
		m.pick(m.menuAt)
	case "esc", "=":
		m.menu = false
	default:
		m.menuAt = m.nextItem(m.menuAt, k.Text)
	}
	return m, nil
}

// enabled says whether the menu's line k does something to the selected stage's rows now: expand all while a row
// that hides something is collapsed, collapse all while a row is expanded; the settings and the rule never do.
func (m Model) enabled(k int) bool {
	s, rows := m.stages[m.stage], m.rows(m.stage)
	for j, r := range rows {
		switch menuItems[k].name {
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
	for j := k + step; j >= 0 && j < len(menuItems); j += step {
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
	for step := 1; step <= len(menuItems); step++ {
		j := (k + step) % len(menuItems)
		if m.enabled(j) && strings.HasPrefix(strings.ToLower(menuItems[j].name), strings.ToLower(typed)) {
			return j
		}
	}
	return k
}

// pick does what the menu's item at k says to the selected stage's rows, closes the menu, and gives the rows the
// keyboard, so left and right go on from there; an item that is not enabled does nothing, and leaves the menu open.
func (m *Model) pick(k int) {
	if k < 0 || !m.enabled(k) {
		return
	}
	m.menu, m.inRows, m.inPanel = false, true, false
	s, rows := &m.stages[m.stage], m.rows(m.stage)
	switch menuItems[k].name {
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

// menuBox gives the menu's lines, a rounded box in the accent around its items, each after the room for a glyph, the
// one under the keyboard inverse and the ones not enabled dim, a rule between groups, a cell short of either side; and
// the screen column its left edge is drawn at: two before the menu glyph's, so the glyph starts at the top edge's third
// cell; when the box would run past the rows that way, as on the rightmost tab, the column that puts the glyph two
// cells before the right corner instead, or nearer when the rows end there; and against the rows' right when the bar
// does not draw the glyph. The rows, not the bar, bound it, as the lines under the bar end where the rows do.
func (m Model) menuBox() (lines []string, left int) {
	names := make([]string, len(menuItems))
	for k, item := range menuItems {
		names[k] = item.name
	}
	inner := pane.Width(names) + 5
	glyph, width, room, cells := m.bar().MenuAt(), inner+2, m.paneWidth(), ansi.StringWidth(tabs.Menu)
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
		top = slices.Concat(top[:c], []rune(tabs.Menu), top[c+cells:])
	}
	lines = append(lines, style.Menu(string(top)))
	for k, item := range menuItems {
		text := pane.Pad(" "+pane.Pad(item.glyph, 2, false)+" "+item.name+" ", inner, false)
		switch {
		case item.name == "":
			lines = append(lines, style.Menu("│ "+strings.Repeat("─", inner-2)+" │"))
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

// menuTop is the screen line the menu's box starts on: the bar's middle one, its top edge through the menu glyph.
const menuTop = paneTop - 2

// overlay draws the open menu over the screen's lines, from menuTop; what the box covers is hidden, and what is
// beside it stays in its columns, a two-cell glyph the box cuts in half giving way to a space.
func (m Model) overlay(lines []string) []string {
	if !m.menu {
		return lines
	}
	box, left := m.menuBox()
	for k, b := range box {
		if i := menuTop + k; i < len(lines) {
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

// menuClick takes a left click while the menu is open: on an item, it chooses it, and on one not enabled, or the rule,
// it does nothing; anywhere else, it closes the menu.
func (m *Model) menuClick(x, y int) {
	box, left := m.menuBox()
	k := y - menuTop - 1
	if x > left && x < left+ansi.StringWidth(box[0])-1 && k >= 0 && k < len(menuItems) {
		m.pick(k)
		return
	}
	m.menu = false
}
