// A cell with a summary, as an issue's labels by their count: closed, its column shows the summary; open, its values,
// one to a line, down the same column, so two lists in a row each keep their own. The column is as wide as its widest
// value.
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

func TestACellWithASummaryShowsItClosedAndItsValuesDownItsColumnOpen(t *testing.T) {
	closed := plain(Draw(labeled(), nil, 40, 0, -1, func(Row) bool { return false }, nil))
	if want := []string{"  ▸ #1  3                 a title", "  ▸ #2  1                 two",
		"    #3                    three"}; !slices.Equal(closed, want) {
		t.Errorf("closed:\n%s\nwant\n%s", strings.Join(closed, "\n"), strings.Join(want, "\n"))
	}
	open := plain(Draw(labeled(), nil, 40, 0, -1, func(Row) bool { return true }, nil))
	want := []string{
		"  ▾ #1  bug               a title",
		"        priority high",
		"        good first issue",
		"  ▾ #2  docs              two",
		"    #3                    three",
	}
	if !slices.Equal(open, want) {
		t.Errorf("open:\n%s\nwant\n%s", strings.Join(open, "\n"), strings.Join(want, "\n"))
	}
	if n := Need([]Cell{labeled()[0].Cells[1]}, ""); n != len("good first issue") {
		t.Errorf("the summary's column needs %d cells", n)
	}
}

// Two lists in a row, open, each stand down their own column.
func TestTwoListsInAnOpenRowEachKeepTheirColumn(t *testing.T) {
	list := func(values ...string) Cell { return Cell{Values: values, Raw: values, Summary: "2"} }
	rows := []Row{{Key: "a", Cells: []Cell{{Values: []string{"a"}}, list("remote", "primary"), list("x y", "z"),
		{Values: []string{"meaning"}, Wrap: true}}}}
	got := plain(Draw(rows, nil, 60, 0, -1, func(Row) bool { return true }, nil))
	want := []string{"  ▾ a  remote   x y  meaning", "       primary  z"}
	if !slices.Equal(got, want) {
		t.Errorf("open:\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
}
