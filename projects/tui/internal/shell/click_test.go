package shell

import (
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
)

// A second left click on the same cell within DoubleClick of the first is a double click; later, on another cell, or
// after a click of another button, it is a click of its own. A double click starts over: the third click is the first
// of the next pair. The clock is the test's.
func TestASecondClickOnTheSameCellSoonIsADoubleClick(t *testing.T) {
	start := time.Date(2026, 11, 5, 12, 0, 0, 0, time.UTC)
	now := start
	c := Clicks{Now: func() time.Time { return now }}
	left := func(x, y int) tea.MouseClickMsg { return tea.MouseClickMsg{X: x, Y: y, Button: tea.MouseLeft} }
	click := func(msg tea.MouseClickMsg, after time.Duration) bool {
		now = now.Add(after)
		var double bool
		c, double = c.Click(msg)
		return double
	}
	for k, step := range []struct {
		msg   tea.MouseClickMsg
		after time.Duration
		want  bool
	}{
		{left(3, 4), 0, false},
		{left(3, 4), DoubleClick, true},
		{left(3, 4), time.Millisecond, false}, // a third click starts the next pair
		{left(3, 4), 100 * time.Millisecond, true},
		{left(3, 4), time.Second, false},
		{left(3, 4), DoubleClick + time.Millisecond, false}, // too late: a click of its own, the first of a pair
		{left(4, 4), 10 * time.Millisecond, false},          // another cell
		{tea.MouseClickMsg{X: 4, Y: 4, Button: tea.MouseRight}, 10 * time.Millisecond, false},
		{left(4, 4), 10 * time.Millisecond, false}, // the right click forgot the left one before it
		{left(4, 4), 10 * time.Millisecond, true},
		{tea.MouseClickMsg{X: 4, Y: 4, Button: tea.MouseLeft, Mod: tea.ModShift}, 10 * time.Millisecond, false},
		{tea.MouseClickMsg{X: 4, Y: 4, Button: tea.MouseLeft, Mod: tea.ModShift}, 10 * time.Millisecond, true},
	} {
		if got := click(step.msg, step.after); got != step.want {
			t.Errorf("click %d, %v after the one before: double %v", k, step.after, got)
		}
	}
	// a clock that went back is no double click
	now = start
	c = Clicks{Now: func() time.Time { return now }}
	click(left(1, 1), 0)
	if click(left(1, 1), -time.Millisecond) {
		t.Error("a click before the one before it is a double click")
	}
	// without a clock, it reads the time
	var real Clicks
	real, _ = real.Click(left(1, 1))
	if _, double := real.Click(left(1, 1)); !double {
		t.Error("two clicks at once, on the real clock, are not a double click")
	}
}
