// A field that holds a list, as an issue's assignees or labels, sorts by size while a row holds all its values; its
// column's card flattens it, a row per value, which sorts by value, and gathers the values onto one row again.
package flow

import (
	"slices"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/empowerite/kit/projects/tui/internal/table"
	"github.com/empowerite/kit/projects/tui/internal/testkit"
)

// Every field the rows hold a list in, a field read by one path whose rows hold several values in it, names one of its
// values, and declares no sort; a field that names one is a list.
func TestAFieldThatHoldsAListNamesOneValueAndNeverSorts(t *testing.T) {
	m := loaded(t, options())
	for _, s := range m.stages {
		for _, f := range fieldsOf(s.name) {
			if f.One != "" && f.Sort {
				t.Errorf("%s's %s holds a list and sorts", s.name, f.ID)
			}
			if f.One != "" || strings.Contains(f.Path, ",") {
				continue
			}
			for _, r := range m.records(s) {
				if len(r.values[f.ID].Raw) > 1 {
					t.Errorf("%s's %s holds a list, %q in %s, and names none of its values", s.name, f.ID, r.values[f.ID].Raw, r.Key)
					break
				}
			}
		}
	}
	if got := onesOf("issues"); got["assignees"] != "assignee" || got["labels"] != "label" || len(got) != 2 {
		t.Errorf("issues' lists: %v", got)
	}
}

// A click on the name of a column that holds a list opens its card, whose One row per line flattens it: the header
// names one value, a row stands once per value it holds, and once where it holds none, every count still the rows',
// and the rule under the name sorts by it; the preset the panel is read back as says it is flat. Its All … on one row
// line gathers the values onto one row again, and its sort no longer orders them.
func TestTheCardFlattensAListAndItsRowsSort(t *testing.T) {
	m := loaded(t, viewing(""), tea.WindowSizeMsg{Width: 120, Height: 30})
	// choose clicks a column's name, then its card's line that reads as line, then closes the card
	choose := func(m Model, column, line string) Model {
		x, _ := m.columnName(column)
		m = drive(m, testkit.Click(x, m.headerTop()))
		_, top, left := m.pop.Box(&m)
		return drive(m, testkit.Click(left+3, top+1+cardLine(m, line)), esc)
	}
	rows := func(m Model, key string) int {
		n := 0
		for _, r := range m.rows(m.stage) {
			if r.Key == key {
				n++
			}
		}
		return n
	}
	m = choose(m, "assignees", table.FlattenIt("assignee"))
	header := lines(m)[m.headerTop()]
	if !m.panel(m.stage).Field("assignees").Flat || !strings.Contains(header, "assignee ") ||
		strings.Contains(header, "assignees") || rows(m, "#7") != 2 || rows(m, "#45") != 1 {
		t.Fatalf("assignees flat: #7 %d rows, #45 %d\n%s", rows(m, "#7"), rows(m, "#45"), raw(m))
	}
	// jig's #7 stands on two lines, one row
	for _, want := range []string{"issues  7", "repository ▾ 7", "github · empowerite 7", "jig 1"} {
		if !strings.Contains(raw(m), want) {
			t.Errorf("a count is no longer the rows': no %q\n%s", want, raw(m))
		}
	}
	// the sort's target stands after the flatten mark, under the name's first cell
	x, _ := m.columnName("assignees")
	if m = drive(m, testkit.Click(x+1, ruleY(m))); m.panel(m.stage).Mark("assignees") != "▲" {
		t.Errorf("a click under assignee: %q", m.panel(m.stage).Mark("assignees"))
	}
	if pr := table.PresetOf(m.panel(m.stage), fieldsOf("issues")); !pr.Fields["assignees"].Flat ||
		pr.Fields["assignees"].Sort != "asc" {
		t.Errorf("the preset: %+v", pr.Fields["assignees"])
	}
	// labels too, the rows a copy per pair of values
	if m = choose(m, "labels", table.FlattenIt("label")); rows(m, "#45") != 3 || !strings.Contains(lines(m)[m.headerTop()], "label ") {
		t.Errorf("labels flat too: #45 %d rows", rows(m, "#45"))
	}
	// gathered, its sort orders the rows by size, how many assignees each holds
	m = choose(choose(m, "labels", table.GatherIt("labels")), "assignees", table.GatherIt("assignees"))
	if p := m.panel(m.stage); p.Field("assignees").Flat || len(p.SortKeys()) != 1 || !p.Counts("assignees") ||
		rows(m, "#7") != 1 ||
		!strings.Contains(lines(m)[m.headerTop()], "assignees") || slices.Contains(p.Groups(), "assignees") {
		t.Errorf("gathered: keys %v, #7 %d rows", p.SortKeys(), rows(m, "#7"))
	}
}
