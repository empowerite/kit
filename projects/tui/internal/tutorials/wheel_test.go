// The mouse wheel scrolls the cards a row at a time, or a heading, never past their ends, the keyboard and the cursor
// left where they are, until a key moves the cursor and the pane follows it again; a turn off the cards, or while a
// menu is open, scrolls nothing.
package tutorials

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

// turn gives a turn of the mouse wheel at the screen cell x, y, down or up.
func turn(x, y int, down bool) tea.MouseWheelMsg {
	b := tea.MouseWheelUp
	if down {
		b = tea.MouseWheelDown
	}
	return tea.MouseWheelMsg{X: x, Y: y, Button: b}
}

func TestTheWheelScrollsTheCardsAndLeavesTheCursor(t *testing.T) {
	m, _ := withEight(t, tea.WindowSizeMsg{Width: 60, Height: 24})
	over := paneTop + 3
	if m = drive(m, turn(10, over, true)); m.offset != 1 || m.cursor != 0 || !m.focused() ||
		!strings.Contains(view(m), "… 1 above") {
		t.Errorf("a turn down: offset %d, cursor %d, cards focused %v\n%s", m.offset, m.cursor, m.focused(), view(m))
	}
	if m = drive(m, turn(10, over, true), turn(10, over, false)); m.offset != 1 {
		t.Errorf("down, then up: offset %d", m.offset)
	}
	if m = drive(m, turn(10, over, false), turn(10, over, false)); m.offset != 0 {
		t.Errorf("up past the top: offset %d", m.offset)
	}
	// never past the last card
	for range 20 {
		m = drive(m, turn(10, over, true))
	}
	end := m.offset
	if m = drive(m, turn(10, over, true)); end == 0 || m.offset != end || m.cursor != 0 ||
		!strings.Contains(view(m), "Tutorial 8") || strings.Contains(view(m), "more") {
		t.Errorf("past the end: offset %d, then %d, cursor %d\n%s", end, m.offset, m.cursor, view(m))
	}
	// off the cards, and with the app's menu open, a turn scrolls nothing
	for _, y := range []int{0, controlTop, m.height - 1} {
		if n := drive(m, turn(10, y, false)); n.offset != end {
			t.Errorf("a turn on line %d scrolled the cards to %d", y, n.offset)
		}
	}
	if n := drive(m, press('='), turn(10, over, false)); n.offset != end {
		t.Errorf("a turn under the app's menu scrolled the cards to %d", n.offset)
	}
	// a key moves the cursor, and the pane follows it again
	if m = drive(m, down); m.cursor != 1 || m.offset != 1 {
		t.Errorf("down after the turns: cursor %d, offset %d\n%s", m.cursor, m.offset, view(m))
	}
}
