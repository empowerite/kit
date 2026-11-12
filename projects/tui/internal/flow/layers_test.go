// The picture's layers: one order, the topmost first, for the keys, the clicks, the keys bar, and the drawing.
package flow

import (
	"testing"

	tea "charm.land/bubbletea/v2"
)

// The menus that hang from the title and the bar take a left click, and leave a right one to the screen, which leaves
// them open.
func TestTheMenusLeaveARightClickToTheScreen(t *testing.T) {
	right := tea.MouseClickMsg{X: 3, Y: 12, Button: tea.MouseRight}
	m := loaded(t, options(), press('='))
	if _, _, ok := m.layers().Click(m, right); ok || !m.appMenu.Open {
		t.Errorf("the app's menu took a right click: %v, open %v", ok, m.appMenu.Open)
	}
	if m = drive(m, right); !m.appMenu.Open {
		t.Error("a right click closed the app's menu")
	}
	m = loaded(t, options(), press(':'))
	if _, _, ok := m.layers().Click(m, right); ok || !m.tabMenu.Open {
		t.Errorf("the tab's menu took a right click: %v, open %v", ok, m.tabMenu.Open)
	}
}
