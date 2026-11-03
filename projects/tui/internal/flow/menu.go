// The selected tab's menu: = or a click on the tab's menu glyph drops it down right under the glyph, its left edge in
// the glyph's column, or its right edge when the box would run off the screen. While it is open it has the keyboard: up and down
// move, a letter moves to the next item that starts with it, enter and space choose, and esc or = closes it; a click
// on an item chooses it, and a click anywhere else closes it. Each choice acts on the selected stage's rows.
package flow

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/empowerite/kit/projects/tui/internal/grid"
	"github.com/empowerite/kit/projects/tui/internal/pane"
	"github.com/empowerite/kit/projects/tui/internal/style"
)

// menuItems are the menu's choices, top to bottom: every row that hides something opened, and every row closed.
var menuItems = []string{"open all", "close all"}

// menuKeys is the keys bar while the menu is open.
const menuKeys = "↑↓ move · ⏎ choose · esc close"

// openMenu drops the menu down, its first item under the keyboard.
func (m *Model) openMenu() {
	if len(m.stages) > 0 {
		m.menu, m.menuAt = true, 0
	}
}

// menuKey takes a key while the menu is open.
func (m Model) menuKey(k tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch k.String() {
	case "up":
		m.menuAt = max(m.menuAt-1, 0)
	case "down":
		m.menuAt = min(m.menuAt+1, len(menuItems)-1)
	case "enter", "space":
		m.pick(m.menuAt)
	case "esc", "=":
		m.menu = false
	default:
		m.menuAt = nextItem(m.menuAt, k.Text)
	}
	return m, nil
}

// nextItem gives the item after the one at k, round from the last to the first, that starts with the letter typed,
// regardless of case; k itself when none does, or nothing was typed.
func nextItem(k int, typed string) int {
	if typed == "" {
		return k
	}
	for step := 1; step <= len(menuItems); step++ {
		j := (k + step) % len(menuItems)
		if strings.HasPrefix(strings.ToLower(menuItems[j]), strings.ToLower(typed)) {
			return j
		}
	}
	return k
}

// pick closes the menu and does what its item at k says to the selected stage's rows.
func (m *Model) pick(k int) {
	m.menu = false
	s, rows := &m.stages[m.stage], m.rows(m.stage)
	switch menuItems[k] {
	case "open all":
		for j, r := range rows {
			if grid.Opens(rows, j, m.paneWidth()) {
				s.open[r.Key] = true
			}
		}
	case "close all":
		clear(s.open)
	}
	m.fit()
}

// menuBox gives the menu's lines, a rounded box in the accent around its items, the one under the keyboard inverse,
// and the screen column its left edge is drawn at: the menu glyph's; when the box would run off the screen that way,
// as on the rightmost tab, the column that puts its right edge under the glyph; and against the screen's right when
// the bar does not draw the glyph.
func (m Model) menuBox() (lines []string, left int) {
	inner := pane.Width(menuItems) + 2
	lines = append(lines, style.Mark("╭"+strings.Repeat("─", inner)+"╮"))
	for k, item := range menuItems {
		text := pane.Pad(" "+item+" ", inner, false)
		if k == m.menuAt {
			text = style.Cursor(text)
		}
		lines = append(lines, style.Mark("│")+text+style.Mark("│"))
	}
	lines = append(lines, style.Mark("╰"+strings.Repeat("─", inner)+"╯"))
	glyph, width := m.bar().MenuAt(), inner+2
	switch {
	case glyph < 0:
		left = m.width - width
	case glyph+width > m.width:
		// the box would run off the screen: its right edge is under the glyph instead
		left = glyph + 1 - width
	default:
		left = glyph
	}
	return lines, max(0, left)
}

// menuTop is the screen line the menu's box starts on: the bar's last, right under the menu glyph on its middle one.
const menuTop = paneTop - 1

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
	return cells(line, 0, left) + box + cells(line, right, ansi.StringWidth(line))
}

// cells gives the cells of line from the column from up to to, with every escape sequence in it, so its colors hold;
// a glyph either edge cuts in half is a space for each of its cells inside.
func cells(line string, from, to int) string {
	var b strings.Builder
	col, state := 0, byte(0)
	for line != "" {
		seq, width, n, next := ansi.DecodeSequence(line, state, nil)
		start, end := col, col+width
		switch {
		case width == 0:
			b.WriteString(seq)
		case start >= from && end <= to:
			b.WriteString(seq)
		case end > from && start < to:
			b.WriteString(strings.Repeat(" ", min(end, to)-max(start, from)))
		}
		col, state, line = end, next, line[n:]
	}
	return b.String()
}

// menuClick takes a left click while the menu is open: on an item, it chooses it; anywhere else, it closes the menu.
func (m *Model) menuClick(x, y int) {
	box, left := m.menuBox()
	k := y - menuTop - 1
	if x > left && x < left+ansi.StringWidth(box[0])-1 && k >= 0 && k < len(menuItems) {
		m.pick(k)
		return
	}
	m.menu = false
}
