package table

import (
	"fmt"
	"runtime"
	"testing"

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

// A turn of the mouse wheel and its drawing copy no row: a wheel spun sends dozens of turns, and the screen is drawn
// after each, so a turn that copied every row of a table of ten thousand, a few hundred bytes each, several times over,
// would allocate tens of megabytes and leave the collector the rest of the frame. What a turn allocates is counted, not
// timed, so the machine's load weighs on it not at all: over ten thousand rows a turn may allocate no more than 64
// bytes a row past what it does over a thousand, the few a row a slice of each row's text takes.
func TestATurnOfTheWheelCopiesNoRow(t *testing.T) {
	allocated := func(rows int) uint64 {
		s := NewSheet(things, "things", "75", "")
		s.Place(4, 36, 120, 40)
		records := make([]Record, rows)
		for i := range records {
			records[i] = thing(fmt.Sprintf("thing %05d", i), []string{"fruit", "vegetable"}[i%2],
				"a note that says what the thing is, at some length")
		}
		s.SetRecords(records)
		_ = s.View()
		turn := tea.MouseWheelMsg{Button: tea.MouseWheelDown, X: 5, Y: s.top + s.height/2}
		var before, after runtime.MemStats
		runtime.ReadMemStats(&before)
		for range turns {
			s.Wheel(turn)
			_ = s.View()
		}
		runtime.ReadMemStats(&after)
		if s.tbl.Offset == 0 {
			t.Errorf("the wheel did not scroll %d rows", rows)
		}
		return (after.TotalAlloc - before.TotalAlloc) / turns
	}
	few, many := allocated(1000), allocated(10000)
	if many > few+9000*64 {
		t.Errorf("a turn over ten thousand rows allocates %d bytes, over a thousand %d: it copies the rows", many, few)
	}
}

// turns is how many turns of the wheel TestATurnOfTheWheelCopiesNoRow counts the allocations of.
const turns = 20
