package table

import (
	"slices"
	"strings"
	"testing"

	"github.com/empowerite/kit/projects/tui/internal/grid"
	"github.com/empowerite/kit/projects/tui/internal/shell"
	"github.com/empowerite/kit/projects/tui/internal/testkit"
)

// fruity gives the sheet of the five things whose fruit alone can be selected.
func fruity() Sheet {
	s := sheet("")
	s.SetPicks(func(key string) bool { return slices.Contains([]string{"pear", "apple", "fig"}, key) })
	s.FocusRows()
	return s
}

// A sheet selects no row until SetPicks says which can be; its keys then name selecting, and draw the rows that can
// be with an empty box, those that cannot with none.
func TestASheetSelectsOnlyTheRowsItsPicksAllow(t *testing.T) {
	s := sheet("")
	s.FocusRows()
	if took := pressed(&s, "down", "space"); took[1] || len(s.Selected()) != 0 || s.Selects() {
		t.Fatalf("a sheet without picks took space: %v, selected %q", took, s.Selected())
	}
	s = fruity()
	if seg, _ := s.Keys(); !strings.Contains(shell.Path{seg}.String(), "␣·⇧↑↓ select ⌃A all") {
		t.Errorf("the keys do not name selecting: %q", shell.Path{seg}.String())
	}
	screen := strings.Join(screenOf(s), "\n")
	for _, want := range []string{"□ ▾ fruit", "  □   apple", "  ▾ vegetable", "    ▸ kale", "      leek"} {
		if !strings.Contains(screen, want) {
			t.Errorf("no %q:\n%s", want, screen)
		}
	}
	// ctrl+a selects the fruit, in the order of the records, and again none
	pressed(&s, "ctrl+a")
	if got := s.Selected(); !slices.Equal(got, []string{"pear", "apple", "fig"}) {
		t.Errorf("ctrl+a selected %q", got)
	}
	if pressed(&s, "ctrl+a"); len(s.Selected()) != 0 {
		t.Errorf("ctrl+a again left %q", s.Selected())
	}
	// space on a vegetable selects nothing; on the vegetables' heading, nothing; on the fruit's, every fruit
	for s.Rows()[s.tbl.Cursor].Key != "kale" {
		pressed(&s, "down")
	}
	if pressed(&s, "space"); len(s.Selected()) != 0 {
		t.Errorf("space on kale selected %q", s.Selected())
	}
	pressed(&s, "up", "space")
	if !grid.Heads(s.Rows()[s.tbl.Cursor]) || len(s.Selected()) != 0 {
		t.Errorf("space on the vegetables' heading selected %q", s.Selected())
	}
	pressed(&s, "home", "space")
	if got := s.Selected(); len(got) != 3 {
		t.Errorf("space on the fruit's heading selected %q", got)
	}
	// records without a row selected unselect it
	s.SetRecords(stock()[:2])
	if got := s.Selected(); !slices.Equal(got, []string{"pear"}) {
		t.Errorf("after the records changed: %q", got)
	}
	if s.Unselect(); len(s.Selected()) != 0 {
		t.Errorf("Unselect left %q", s.Selected())
	}
}

// A plain click on a row's box selects it, or unselects it, and on a group's heading's box every row of the group that
// can be; a click on the box of a row that cannot be selected, or on the rest of a row, selects nothing.
func TestAClickOnABoxSelects(t *testing.T) {
	s := fruity()
	x, y := spot(s, "□   apple")
	if !s.Click(testkit.Click(x, y)) || !slices.Equal(s.Selected(), []string{"apple"}) ||
		s.Rows()[s.tbl.Cursor].Key != "apple" {
		t.Fatalf("a click on apple's box selected %q", s.Selected())
	}
	if !strings.Contains(strings.Join(screenOf(s), "\n"), "◩ ▾ fruit") {
		t.Errorf("the fruit's heading is not partly selected:\n%s", strings.Join(screenOf(s), "\n"))
	}
	if s.Click(testkit.Click(x, y)); len(s.Selected()) != 0 {
		t.Errorf("a second click left %q", s.Selected())
	}
	if _, y := spot(s, "kale"); s.Click(testkit.Click(x, y)) && len(s.Selected()) != 0 {
		t.Errorf("a click on kale's gutter selected %q", s.Selected())
	}
	if _, y := spot(s, "apple"); s.Click(testkit.Click(x+8, y)) && len(s.Selected()) != 0 {
		t.Errorf("a click on apple's name selected %q", s.Selected())
	}
	hx, hy := spot(s, "□ ▾ fruit")
	if s.Click(testkit.Click(hx, hy)); len(s.Selected()) != 3 {
		t.Errorf("a click on the fruit's heading's box selected %q", s.Selected())
	}
}
