// The mouse wheel scrolls what it turns over: a page's lines, a step's checks, or a step's table, never past their
// ends, the keyboard and the cursor left where they are; a turn off the pane, or while a menu is open, scrolls nothing.
package welcome

import (
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

func TestTheWheelScrollsAPage(t *testing.T) {
	m, _ := house(t, tea.WindowSizeMsg{Width: 120, Height: 12}, esc, stepKey(ourTeam))
	if m.step != ourTeam || m.inPane {
		t.Fatalf("step %d, in the pane %v", m.step, m.inPane)
	}
	over := paneTop + 2
	if m = drive(m, turn(10, over, true)); m.offsets[ourTeam] != wheel || m.inPane {
		t.Errorf("a turn down: offset %d, in the pane %v", m.offsets[ourTeam], m.inPane)
	}
	if m = drive(m, turn(10, over, true), turn(10, over, false)); m.offsets[ourTeam] != wheel {
		t.Errorf("down, then up: offset %d", m.offsets[ourTeam])
	}
	if m = drive(m, turn(10, over, false), turn(10, over, false)); m.offsets[ourTeam] != 0 {
		t.Errorf("up past the top: offset %d", m.offsets[ourTeam])
	}
	// never past the page's end
	for range 50 {
		m = drive(m, turn(10, over, true))
	}
	end := m.offsets[ourTeam]
	if m = drive(m, turn(10, over, true)); end == 0 || m.offsets[ourTeam] != end {
		t.Errorf("past the end: offset %d, then %d", end, m.offsets[ourTeam])
	}
	// off the pane, the title and the tab bar, and with the app's menu open, a turn scrolls nothing
	for _, y := range []int{0, 1, m.height - 1} {
		if n := drive(m, turn(10, y, false)); n.offsets[ourTeam] != end {
			t.Errorf("a turn on line %d scrolled the page to %d", y, n.offsets[ourTeam])
		}
	}
	if n := drive(m, press('='), turn(10, over, false)); n.offsets[ourTeam] != end {
		t.Errorf("a turn under the app's menu scrolled the page to %d", n.offsets[ourTeam])
	}
}

func TestTheWheelScrollsAStepsChecksAndLeavesTheCursor(t *testing.T) {
	// on a short screen the pane has scrolled down to the cursor's row; a turn up scrolls back past it
	m, _ := atChains(t, tea.WindowSizeMsg{Width: 120, Height: 10})
	at, from := m.cursor, m.offsets[chains]
	m = drive(m, turn(10, paneTop+1, false))
	if from < wheel || m.offsets[chains] != from-wheel || m.cursor != at || !m.inPane {
		t.Errorf("a turn up from %d: offset %d, cursor %d, in the pane %v", from, m.offsets[chains], m.cursor, m.inPane)
	}
}

func TestTheWheelScrollsAStepsTable(t *testing.T) {
	m, _ := loaded(t, tea.WindowSizeMsg{Width: 120, Height: 16}, stepKey(terms))
	before := view(m)
	if m = drive(m, turn(10, m.height-4, true)); view(m) == before {
		t.Errorf("a turn over the terms' rows scrolled nothing:\n%s", view(m))
	}
}
