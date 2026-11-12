package table

import (
	"fmt"
	"math"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
)

// A sheet's lines are drawn once for what it shows: a turn of the wheel, and drawing the pane, reuse them. Moving the
// cursor, selecting a row, opening one, or saying anew which rows can be, draws them anew.
func TestASheetsLinesAreDrawnOnceForWhatItShows(t *testing.T) {
	s := fruity()
	first := s.Lines()
	_ = s.View()
	s.Wheel(tea.MouseWheelMsg{Button: tea.MouseWheelDown, X: 5, Y: s.top + 2})
	_ = s.View()
	if again := s.Lines(); &again[0] != &first[0] {
		t.Error("the wheel and a drawing drew the lines again for the same rows")
	}
	for _, change := range []struct {
		name string
		do   func()
	}{
		{"moving the cursor", func() { pressed(&s, "down") }},
		{"selecting the row", func() { pressed(&s, "space") }},
		{"moving onto kale", func() { pressed(&s, "end", "up") }},
		{"opening kale", func() { pressed(&s, "right") }},
		{"saying anew which rows can be selected", func() { s.SetPicks(func(string) bool { return true }) }},
	} {
		before := s.Lines()
		change.do()
		if again := s.Lines(); &again[0] == &before[0] {
			t.Errorf("%s kept the lines drawn before", change.name)
		}
	}
}

// With ten thousand rows, a turn of the mouse wheel over them and its drawing take well under the 16 ms a frame lasts
// at 60 frames a second: a wheel spun sends dozens of turns, and the screen is drawn after each, so a turn costs what
// the columns and the lines on the screen do, not every row, and the rows keep up with the wheel. The race detector
// slows everything about tenfold, so its bound is ten times as wide.
func TestATurnOfTheWheelOverTenThousandRowsIsWellUnderAFramesTime(t *testing.T) {
	budget := 16 * time.Millisecond / 2
	if raceEnabled {
		budget *= 10
	}
	s := NewSheet(things, "things", "75", "")
	s.Place(4, 36, 120, 40)
	records := make([]Record, 10000)
	for i := range records {
		records[i] = thing(fmt.Sprintf("thing %05d", i), []string{"fruit", "vegetable"}[i%2],
			"a note that says what the thing is, at some length")
	}
	s.SetRecords(records)
	_ = s.View()
	turn := tea.MouseWheelMsg{Button: tea.MouseWheelDown, X: 5, Y: s.top + s.height/2}
	if per := best(func() { s.Wheel(turn); _ = s.View() }); per > budget {
		t.Errorf("a turn of the wheel and its drawing take %v, over %v", per, budget)
	}
	if s.tbl.Offset == 0 {
		t.Error("the wheel did not scroll the rows")
	}
}

// best gives how long do takes, the least of three batches of ten, so a moment the machine is busy elsewhere, or a
// slower core, does not count against it.
func best(do func()) time.Duration {
	const batches, n = 3, 10
	least := time.Duration(math.MaxInt64)
	for range batches {
		start := time.Now()
		for range n {
			do()
		}
		least = min(least, time.Since(start)/n)
	}
	return least
}
