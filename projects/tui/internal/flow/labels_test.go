// A forge's labels: each a pill, a key and a value two segments; a row's column counts them, its open row draws them,
// and a filter keeps the rows that hold one of the labels it names, or none of them.
package flow

import (
	"slices"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"github.com/empowerite/kit/projects/tui/internal/grid"
)

func TestALabelSplitsIntoAKeyAndAValueAtItsLastSeparator(t *testing.T) {
	for name, want := range map[string][3]string{
		"bug":                {"", "bug", ""},
		"priority=high":      {"priority", "high", "split"},
		"kind:fix":           {"kind", "fix", "split"},
		"area::tui":          {"area", "tui", "split"},
		"team::ux::research": {"team::ux", "research", "split"},
		"a:b=c":              {"a:b", "c", "split"},
		"a=b:c":              {"a=b", "c", "split"},
		"area::ui:dark":      {"area::ui", "dark", "split"},
		"area:ui::dark":      {"area:ui", "dark", "split"},
		":wip":               {"", ":wip", ""},
		"todo:":              {"", "todo:", ""},
		"::":                 {"", "::", ""},
		"area::":             {"", "area::", ""},
	} {
		key, val, ok := splitLabel(name)
		if got := [3]string{key, val, map[bool]string{true: "split"}[ok]}; got != want {
			t.Errorf("%q splits as %q, want %q", name, got, want)
		}
	}
}

// A plain label is its name between two half circles; a key and a value, the key, the round edge where the colors
// meet, then the value, the : or = dropped. Each half circle takes one cell.
func TestAPillIsItsTextBetweenHalfCircles(t *testing.T) {
	if got := ansi.Strip(pill(label{Name: "bug", Color: "#d73a4a"})); got != capLeft+"bug"+capRight {
		t.Errorf("a plain pill: %q", got)
	}
	if got := ansi.Strip(pill(label{Name: "priority=high"})); got != capLeft+"priority"+capRight+"high"+capRight {
		t.Errorf("a key and a value: %q", got)
	}
	if w := ansi.StringWidth(pill(label{Name: "kind:fix"})); w != len("kindfix")+3 {
		t.Errorf("the pill takes %d cells", w)
	}
}

// The text is black on a light color and white on a dark one; the key's color is its own, the same for every label
// with that key, and never its value's.
func TestAPillsTextReadsOnItsColorAndAKeyKeepsItsColor(t *testing.T) {
	light, dark := ink("#fbca04"), ink("#1d4ed8")
	if r, _, _, _ := light.RGBA(); r != 0 {
		t.Errorf("text on yellow is not black: %v", light)
	}
	if r, _, _, _ := dark.RGBA(); r == 0 {
		t.Errorf("text on blue is not white: %v", dark)
	}
	if keyColor("priority", "#ffffff") != keyColor("priority", "#000000") {
		t.Error("a key's color depends on its value's")
	}
	for _, c := range pastel {
		if keyColor("priority", c) == c {
			t.Errorf("the key priority is drawn on its value's color %s", c)
		}
	}
	if labelColor(label{Name: "x", Color: "#123456"}) != "#123456" || labelColor(label{Name: "x"}) == "" {
		t.Error("a label's color is not its forge's, or none without one")
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
	want := []string{" ▾ #1  2  a", "      " + capLeft + "bug" + capRight + " " + capLeft + "priority" + capRight + "high" + capRight}
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
