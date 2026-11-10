// A field that holds a list, as an issue's assignees or labels, never sorts while a row holds all its values; its
// column's menu flattens it, a row per value, which sorts, and gathers the values onto one row again.
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
	if got := onesOf("backlog"); got["assignees"] != "assignee" || got["labels"] != "label" || len(got) != 2 {
		t.Errorf("backlog's lists: %v", got)
	}
}

// A click on the name of a column that holds a list drops its menu, whose ⇊ line flattens it: the header names one
// value, a row stands once per value it holds, and once where it holds none, every count still the rows', and the rule
// under the name sorts by it; the preset the panel is read back as says it is flat. Its ⇈ line gathers the values
// onto one row again, and its sort no longer orders them.
func TestTheColumnMenuFlattensAListAndItsRowsSort(t *testing.T) {
	m := loaded(t, viewing(""), tea.WindowSizeMsg{Width: 120, Height: 30})
	choose := func(m Model, column string, act table.ColAct) Model {
		x, _ := m.columnName(column)
		m = drive(m, testkit.Click(x, m.headerTop()))
		_, top, left := m.colMenu.Box(&m)
		return drive(m, testkit.Click(left+3, top+1+m.colMenu.LineOf(&m, act)))
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
	m = choose(m, "assignees", table.ActFlatten)
	header := lines(m)[m.headerTop()]
	if !m.panel(m.stage).Field("assignees").Flat || !strings.Contains(header, "assignee ") ||
		strings.Contains(header, "assignees") || rows(m, "#7") != 2 || rows(m, "#45") != 1 {
		t.Fatalf("assignees flat: #7 %d rows, #45 %d\n%s", rows(m, "#7"), rows(m, "#45"), raw(m))
	}
	for _, want := range []string{"backlog  3", "source 3", "github 3"} {
		if !strings.Contains(raw(m), want) {
			t.Errorf("a count is no longer the rows': no %q\n%s", want, raw(m))
		}
	}
	// the sort's target stands after the flatten mark, under the name's first cell
	x, _ := m.columnName("assignees")
	if m = drive(m, testkit.Click(x+1, ruleY(m))); m.panel(m.stage).Mark("assignees") != "▲" {
		t.Errorf("a click under assignee: %q", m.panel(m.stage).Mark("assignees"))
	}
	if pr := table.PresetOf(m.panel(m.stage), fieldsOf("backlog")); !pr.Fields["assignees"].Flat ||
		pr.Fields["assignees"].Sort != "asc" {
		t.Errorf("the preset: %+v", pr.Fields["assignees"])
	}
	// labels too, the rows a copy per pair of values
	if m = choose(m, "labels", table.ActFlatten); rows(m, "#45") != 3 || !strings.Contains(lines(m)[m.headerTop()], "label ") {
		t.Errorf("labels flat too: #45 %d rows", rows(m, "#45"))
	}
	m = choose(choose(m, "labels", table.ActGather), "assignees", table.ActGather)
	if p := m.panel(m.stage); p.Field("assignees").Flat || len(p.SortKeys()) != 0 || rows(m, "#7") != 1 ||
		!strings.Contains(lines(m)[m.headerTop()], "assignees") || slices.Contains(p.Groups(), "assignees") {
		t.Errorf("gathered: keys %v, #7 %d rows", p.SortKeys(), rows(m, "#7"))
	}
}
