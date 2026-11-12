package shell

import tea "charm.land/bubbletea/v2"

// Turn says which way a turn of the mouse wheel goes, a step across, dx, or a step down, dy, each 1 or -1, the other
// 0: right or down 1, left or up -1. A sideways turn, as a trackpad's swipe across or a tilted wheel sends it, goes
// across, and so does a turn up or down with shift held, as a wheel that only turns up and down scrolls across. A
// button that is no turn goes nowhere.
func Turn(w tea.MouseWheelMsg) (dx, dy int) {
	across := w.Mod.Contains(tea.ModShift)
	switch w.Button {
	case tea.MouseWheelLeft:
		return -1, 0
	case tea.MouseWheelRight:
		return 1, 0
	case tea.MouseWheelUp:
		dy = -1
	case tea.MouseWheelDown:
		dy = 1
	}
	if across {
		return dy, 0
	}
	return 0, dy
}
