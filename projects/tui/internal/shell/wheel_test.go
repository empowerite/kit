package shell

import (
	"testing"

	tea "charm.land/bubbletea/v2"
)

// A turn goes across for left and right, and for up and down with shift; down for down and up; nowhere for a button
// that is no turn.
func TestATurnGoesAcrossOrDown(t *testing.T) {
	for _, c := range []struct {
		button tea.MouseButton
		mod    tea.KeyMod
		dx, dy int
	}{
		{tea.MouseWheelUp, 0, 0, -1},
		{tea.MouseWheelDown, 0, 0, 1},
		{tea.MouseWheelLeft, 0, -1, 0},
		{tea.MouseWheelRight, 0, 1, 0},
		{tea.MouseWheelUp, tea.ModShift, -1, 0},
		{tea.MouseWheelDown, tea.ModShift, 1, 0},
		{tea.MouseWheelLeft, tea.ModShift, -1, 0},
		{tea.MouseLeft, 0, 0, 0},
		{tea.MouseLeft, tea.ModShift, 0, 0},
	} {
		if dx, dy := Turn(tea.MouseWheelMsg{Button: c.button, Mod: c.mod}); dx != c.dx || dy != c.dy {
			t.Errorf("%v with %v: %d, %d, want %d, %d", c.button, c.mod, dx, dy, c.dx, c.dy)
		}
	}
}
