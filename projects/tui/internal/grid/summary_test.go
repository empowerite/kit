// A cell with a summary, as an issue's labels by their count: its column shows the summary, closed or open, and is as
// wide as it; open, the row's values follow its lines, side by side, as many to a line as the width holds.
package grid

import (
	"slices"
	"strings"
	"testing"
)

// labeled gives two rows whose labels stand as a count: the first with three, the second with one, a third with none.
func labeled() []Row {
	tags := func(values ...string) Cell {
		if len(values) == 0 {
			return Cell{}
		}
		return Cell{Values: values, Raw: values, Summary: string(rune('0' + len(values)))}
	}
	return []Row{
		{Key: "#1", Cells: []Cell{{Values: []string{"#1"}, Right: true}, tags("bug", "priority high", "good first issue"),
			{Values: []string{"a title"}, Wrap: true}}},
		{Key: "#2", Cells: []Cell{{Values: []string{"#2"}, Right: true}, tags("docs"), {Values: []string{"two"}, Wrap: true}}},
		{Key: "#3", Cells: []Cell{{Values: []string{"#3"}, Right: true}, tags(), {Values: []string{"three"}, Wrap: true}}},
	}
}

func TestACellWithASummaryShowsItClosedAndItsValuesUnderTheOpenRow(t *testing.T) {
	closed := plain(Draw(labeled(), nil, 40, -1, func(Row) bool { return false }, nil))
	if want := []string{" ▸ #1  3  a title", " ▸ #2  1  two", "   #3     three"}; !slices.Equal(closed, want) {
		t.Errorf("closed:\n%s\nwant\n%s", strings.Join(closed, "\n"), strings.Join(want, "\n"))
	}
	open := plain(Draw(labeled(), nil, 30, -1, func(Row) bool { return true }, nil))
	want := []string{
		" ▾ #1  3  a title",
		"      bug priority high",
		"      good first issue",
		" ▾ #2  1  two",
		"      docs",
		"   #3     three",
	}
	if !slices.Equal(open, want) {
		t.Errorf("open:\n%s\nwant\n%s", strings.Join(open, "\n"), strings.Join(want, "\n"))
	}
	// the column is as wide as its summary, not its widest value, which stands under the row
	if n := Need([]Cell{labeled()[0].Cells[1]}, ""); n != 1 {
		t.Errorf("the summary's column needs %d cells", n)
	}
}

// A value wider than the line has one of its own, cut where it ends, and the values around it keep theirs.
func TestFlowedGivesAValueTooWideALineOfItsOwn(t *testing.T) {
	got := flowed([]string{"a", "much too wide", "b", "c"}, 8)
	if want := []string{"a", "much to…", "b c"}; !slices.Equal(got, want) {
		t.Errorf("flowed: %q, want %q", got, want)
	}
	if got := flowed(nil, 8); got != nil {
		t.Errorf("no values: %q", got)
	}
}
