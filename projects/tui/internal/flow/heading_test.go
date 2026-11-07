// The heading over the groups, whose names drop their columns' menus, as the header's names do; and the headings of
// the values under it, which name nothing and fold.
package flow

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/empowerite/kit/projects/tui/internal/grid"
	"github.com/empowerite/kit/projects/tui/internal/testkit"
)

// The heading over the groups the person chose names their columns, and a click on a name drops that column's menu
// around it, as the header's names do, the cursor on the heading; ⏎ on the heading drops its first column's. Ungroup
// there leaves the rows grouped by the others. A click on a value's heading, which names no column, only folds its rows
// away, and another unfolds them; ⏎ there drops no menu.
func TestANameInTheHeadingOverTheGroupsDropsItsColumnMenu(t *testing.T) {
	m := nested(t)
	all := len(m.rows(0))
	spots := m.headingNames(m.rows(0)[0])
	if len(spots) != 2 || spots[0].column != "source" || spots[1].column != "assignees" {
		t.Fatalf("the heading over the groups names %+v", spots)
	}
	text := ansi.Strip(lines(m)[m.rowsTop()])
	for _, n := range spots {
		if ansi.Cut(text, n.from, n.until) != n.column {
			t.Errorf("%s does not stand at %d: %q", n.column, n.from, text)
		}
	}
	m.stages[0].cursor = 2
	if m = drive(m, testkit.Click(spots[1].from+1, m.rowsTop())); !m.colMenu.Open || !m.colMenu.heading ||
		m.colMenu.column != "assignees" || m.stages[0].cursor != 0 || len(m.rows(0)) != all {
		t.Fatalf("a click on assignees: menu %+v, cursor %d, %d rows of %d", m.colMenu, m.stages[0].cursor,
			len(m.rows(0)), all)
	}
	box, top, _ := m.columnBox()
	if edge := ansi.Strip(box[0]); top != m.rowsTop() || !strings.Contains(edge, "╴assignees╶") ||
		!m.columnEnabled(m.columnLine(actUngroup)) || m.columnEnabled(m.columnLine(actGroup)) {
		t.Errorf("the box hangs at %d, %q:\n%s", top, edge, raw(m))
	}
	// Ungroup leaves the rows grouped by source alone, and closes the menu
	m.colMenu.At = m.columnLine(actUngroup)
	if m = drive(m, press(tea.KeyEnter)); m.colMenu.Open || !slices.Equal(m.panel(0).Groups(), []string{"source"}) {
		t.Fatalf("ungroup assignees: menu %v, groups %q", m.colMenu.Open, m.panel(0).Groups())
	}
	// ⏎ on the heading drops source's menu; esc closes it
	m.stages[0].cursor = 0
	if m = drive(m, press(tea.KeyEnter)); !m.colMenu.Open || m.colMenu.column != "source" {
		t.Fatalf("⏎ on the heading over the groups: %+v", m.colMenu)
	}
	if m = drive(m, esc); m.colMenu.Open {
		t.Error("esc did not close the menu")
	}
	// a click on a value's heading folds it, menu or none; ⏎ there drops none, nor does the keys bar offer one
	all = len(m.rows(0))
	if m = drive(m, testkit.Click(6, m.rowsTop()+1)); m.colMenu.Open || len(m.rows(0)) != 2 {
		t.Fatalf("a click on a value's heading: menu %v, %d rows", m.colMenu.Open, len(m.rows(0)))
	}
	if last := lines(m)[len(lines(m))-1]; strings.Contains(last, "⏎ menu") {
		t.Errorf("the keys on a value's heading: %q", last)
	}
	if m = drive(m, press(tea.KeyEnter)); m.colMenu.Open {
		t.Error("⏎ on a value's heading dropped a menu")
	}
	if m = drive(m, testkit.Click(6, m.rowsTop()+1)); len(m.rows(0)) != all {
		t.Errorf("a second click on the value's heading: %d rows of %d", len(m.rows(0)), all)
	}
	// Options… there opens the field's popover, hanging from its name in the heading; a column not grouped has no
	// name there
	m.stages[0].cursor = 0
	m = drive(m, press(tea.KeyEnter))
	x, y := m.fieldName("source")
	m.columnAct(m.columnLine(actMore))
	if !m.pop.open || m.pop.field != "source" ||
		m.pop.x != x || m.pop.y != y || y != m.rowsTop() {
		t.Errorf("Options… from the heading: %+v, source at %d, %d", m.pop, x, y)
	}
	m.pop = popover{}
	if _, y := m.fieldName("age"); y != -1 {
		t.Errorf("age, not grouped, stands in the heading on line %d", y)
	}
	// while the stage loads, the rule has no sort targets
	m.stages[0].loading = true
	if spots := m.sortSpots(); spots != nil {
		t.Errorf("the sort targets while loading: %+v", spots)
	}
	m.stages[0].loading = false
	// once the heading no longer shows, the menu draws nothing, and a click closes it
	m.stages[0].cursor = 0
	m = drive(m, press(tea.KeyEnter))
	m.setPanel(m.panel(0).Grouped("source", true))
	if box, _, _ := m.columnBox(); box != nil {
		t.Error("the menu of a heading gone still draws")
	}
	if m = drive(m, testkit.Click(1, m.rowsTop())); m.colMenu.Open {
		t.Error("a click did not close the menu of a heading gone")
	}
}

// The names in the heading over the groups are found where the heading says them, each before a ›, the count, or
// the end; a value that holds a column's name in it is passed over, and any other row names nothing.
func TestTheHeadingOverTheGroupsNamesItsColumnsWhereItSaysThem(t *testing.T) {
	m := loaded(t, options())
	m.setPanel(regroup(m.panel(0), "source", "assignees"))
	text := "▾ source › assignees  3"
	heading := row{Key: m.rows(0)[0].Key, Aside: true, Cells: []grid.Cell{{Values: []string{text}}}}
	var got []string
	for _, n := range m.headingNames(heading) {
		got = append(got, fmt.Sprintf("%s@%d", n.column, n.from))
		if cut := ansi.Cut("  "+text, n.from, n.until); cut != n.column {
			t.Errorf("%s stands at %d, where the heading draws %q", n.column, n.from, cut)
		}
	}
	if len(got) != 2 {
		t.Errorf("the names of two columns: %q", got)
	}
	tricky := row{Key: heading.Key, Aside: true, Cells: []grid.Cell{{Values: []string{"▾ sourcery source › assignees  3"}}}}
	if names := m.headingNames(tricky); len(names) != 2 || names[0].from != 2+ansi.StringWidth("▾ sourcery ") {
		t.Errorf("past a value that holds source: %+v", names)
	}
	if names := m.headingNames(m.rows(0)[1]); names != nil {
		t.Errorf("a value's heading names %+v", names)
	}
}

// A click on the ▾ or ▸ before a column's name in the heading over the groups shows the groups down to its level and
// nothing under it: every heading over the level opens, every heading of it closes, which then shows ▸. From
// everything open, it collapses to the level; from the outermost alone, it expands to it.
func TestALevelsFoldInTheHeadingOverTheGroupsShowsTheGroupsDownToIt(t *testing.T) {
	m := nested(t)
	all := len(m.rows(0))
	spots := m.headingNames(m.rows(0)[0])
	glyph := func(column string) string {
		for _, n := range m.headingNames(m.rows(0)[0]) {
			if n.column == column {
				return ansi.Strip(ansi.Cut(lines(m)[m.rowsTop()], n.from-2, n.from-1))
			}
		}
		return ""
	}
	// levels gives how many headings of each level show, the sources' and the assignees', and how many rows
	levels := func() (sources, assignees, rows int) {
		for _, r := range m.rows(0) {
			switch {
			case grid.Level(r) == 2:
				sources++
			case grid.Level(r) == 3:
				assignees++
			case !r.Aside:
				rows++
			}
		}
		return
	}
	if glyph("assignees") != grid.Unfolded {
		t.Fatalf("assignees' fold: %q in %q", glyph("assignees"), ansi.Strip(lines(m)[m.rowsTop()]))
	}
	_, everyAssignee, _ := levels()
	// from everything open, assignees' collapses to the assignees: each listed, no row under them
	m = drive(m, testkit.Click(spots[1].from-2, m.rowsTop()))
	if s, a, r := levels(); s == 0 || a != everyAssignee || r != 0 || m.colMenu.Open || glyph("assignees") != grid.Folded ||
		glyph("source") != grid.Unfolded {
		t.Fatalf("collapsed to the assignees: %d sources, %d assignees, %d rows, folds %q %q\n%s", s, a, r,
			glyph("source"), glyph("assignees"), view(m))
	}
	// source's collapses to the sources alone; assignees' then expands to the assignees, and no further
	if m = drive(m, testkit.Click(spots[0].from-2, m.rowsTop())); glyph("source") != grid.Folded {
		t.Errorf("source's level: %q", glyph("source"))
	}
	if s, a, r := levels(); s == 0 || a != 0 || r != 0 {
		t.Errorf("collapsed to the sources: %d sources, %d assignees, %d rows", s, a, r)
	}
	m = drive(m, testkit.Click(spots[1].from-2, m.rowsTop()))
	if s, a, r := levels(); s == 0 || a != everyAssignee || r != 0 || glyph("source") != grid.Unfolded ||
		glyph("assignees") != grid.Folded {
		t.Errorf("expanded to the assignees: %d sources, %d assignees, %d rows\n%s", s, a, r, view(m))
	}
	if len(m.rows(0)) >= all {
		t.Errorf("rows show under the assignees: %d of %d", len(m.rows(0)), all)
	}
}
