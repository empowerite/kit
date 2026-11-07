// What every dialog shares: where a click falls on its box, and how the line under the keyboard moves.
package shell

import (
	"slices"
	"testing"
)

func TestAPlacedBoxSaysWhereAClickFalls(t *testing.T) {
	// a box 10 cells wide, drawn from line 4 and column 2: its body lines are lines 5 to 6, between columns 3 and 10
	p := Placed{Lines: Dialog("t", []string{"one", "two"}, 10), Top: 4, Left: 2}
	for _, c := range []struct {
		name string
		x, y int
		line int
		on   bool
	}{
		{"on the first body line", 3, 5, 0, true},
		{"on the second", 6, 6, 1, true},
		{"on the left edge", 2, 5, -1, true},
		{"on the top edge", 4, 4, -1, true},
		{"on the bottom edge", 4, 7, -1, true},
		{"left of the box", 1, 5, -1, false},
		{"under the box", 4, 8, -1, false},
	} {
		if line, on := p.Hit(c.x, c.y); line != c.line || on != c.on {
			t.Errorf("%s: line %d, on %v; want %d, %v", c.name, line, on, c.line, c.on)
		}
	}
	if line, on := (Placed{}).Hit(0, 0); line != -1 || on {
		t.Errorf("no box: line %d, on %v", line, on)
	}
	if got := p.Draw([]string{"", "", "", "", "", "", "", "", ""}); got[5] == "" {
		t.Errorf("the box was not drawn: %q", got)
	}
}

func TestStepMovesToTheNextLineThatStandsAndStopsAtTheEnds(t *testing.T) {
	odd := func(k int) bool { return k%2 == 1 }
	for _, c := range []struct{ at, step, want int }{{-1, 1, 1}, {1, 1, 3}, {3, 1, 3}, {3, -1, 1}, {1, -1, 1}} {
		if got := Step(c.at, 5, c.step, odd); got != c.want {
			t.Errorf("from %d by %d: %d, want %d", c.at, c.step, got, c.want)
		}
	}
	var got []int
	for at, k := 0, 0; k < 5; k++ {
		at = Step(at, 3, 1, Every)
		got = append(got, at)
	}
	if !slices.Equal(got, []int{1, 2, 2, 2, 2}) {
		t.Errorf("down every line: %v", got)
	}
}
