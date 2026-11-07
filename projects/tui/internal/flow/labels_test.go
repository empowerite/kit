// A forge's labels: each a pill, as style draws it; a row's column counts them, its open row draws them,
// and a filter keeps the rows that hold one of the labels it names, or none of them.
package flow

import (
	"slices"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"github.com/empowerite/kit/projects/tui/internal/grid"
	"github.com/empowerite/kit/projects/tui/internal/style"
)

// A label is drawn as style draws a pill: on its forge's color, or, with none, on the pastel its name picks.
func TestALabelIsDrawnAsStylesPill(t *testing.T) {
	for _, l := range []label{{Name: "bug", Color: "#d73a4a"}, {Name: "priority=high"}} {
		if got := pill(l); got != style.Pill(l.Name, l.Color) {
			t.Errorf("%+v: %q", l, got)
		}
	}
}

// A row's labels column says how many it has, dim; its open row draws each as a pill, under the row's own lines; a
// row with none says nothing there.
func TestTheLabelsColumnCountsAndTheOpenRowDrawsThePills(t *testing.T) {
	f := fieldIn("backlog", "labels")
	cells := f.draw([]value{labelled([]label{{Name: "bug", Color: "#d73a4a"}, {Name: "priority=high"}}), {}}, "")
	if ansi.Strip(cells[0].Summary) != "2" || !slices.Equal(cells[0].Raw, []string{"bug", "priority=high"}) ||
		len(cells[0].Values) != 2 || cells[1].Summary != "" || len(cells[1].Values) != 0 {
		t.Fatalf("the cells: %+v", cells)
	}
	rows := []grid.Row{{Key: "#1", Cells: []grid.Cell{{Values: []string{"#1"}}, cells[0], {Values: []string{"a"}, Wrap: true}}}}
	var got []string
	for _, l := range grid.Draw(rows, nil, 40, -1, func(grid.Row) bool { return true }, nil) {
		got = append(got, strings.TrimRight(ansi.Strip(l.Text), " "))
	}
	want := []string{" ▾ #1  2  a", "      " + style.PillLeft + "bug" + style.PillRight + " " + style.PillLeft + "priority" + style.PillRight + "high" +
		style.PillRight}
	if !slices.Equal(got, want) {
		t.Errorf("open:\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
	// text draws the names alone, for a terminal without the font's half circles, and counts them the same
	if c := f.draw([]value{labelled([]label{{Name: "bug"}})}, "text"); c[0].Values[0] != "bug" || ansi.Strip(c[0].Summary) != "1" {
		t.Errorf("text: %+v", c[0])
	}
}

// A filter on labels keeps the rows that hold one of the names it includes, or, excluding, none of them; none keeps
// the rows without a label.
func TestAFilterOnLabelsKeepsTheRowsThatHoldOne(t *testing.T) {
	f := fieldIn("backlog", "labels")
	vs := []value{labelled([]label{{Name: "bug"}, {Name: "kind=fix"}}), labelled([]label{{Name: "docs"}}), {}}
	var rows []grid.Row
	for k, c := range f.draw(vs, "") {
		rows = append(rows, grid.Row{Key: string(rune('a' + k)), Cells: []grid.Cell{{Values: []string{"x"}}, c}})
	}
	keys := func(filter grid.Filter) string {
		p := grid.Panel{}.FilteredBy("labels", filter)
		var out []string
		for _, r := range p.Filter(rows, []string{"id", "labels"}) {
			out = append(out, r.Key)
		}
		return strings.Join(out, "")
	}
	for _, c := range []struct {
		filter grid.Filter
		want   string
	}{
		{grid.Filter{Values: []string{"kind=fix"}}, "a"},
		{grid.Filter{Values: []string{"docs", "bug"}}, "ab"},
		{grid.Filter{Values: []string{"bug"}, Exclude: true}, "bc"},
		{grid.Filter{Values: []string{grid.NoValue}}, "c"},
	} {
		if got := keys(c.filter); got != c.want {
			t.Errorf("%+v keeps %q, want %q", c.filter, got, c.want)
		}
	}
}
