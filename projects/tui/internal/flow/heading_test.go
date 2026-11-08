// The heading over the groups, on the control line, whose names drop their columns' menus, as the header's names do,
// and whose folds open and close the levels of the groups; and the headings of the values under it, which name nothing
// and fold.
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

// The heading over the groups the person chose stands on the control line, apart from the rows, and names their
// columns; a click on a name drops that column's menu around it, as the header's names do, the control line under
// the keyboard. Ungroup there leaves the rows grouped by the others. A click on a value's heading, which names no
// column, puts the cursor on it, and one on its ▾ folds its rows away, and another unfolds them; ⏎ there drops no menu.
func TestANameInTheHeadingOverTheGroupsDropsItsColumnMenu(t *testing.T) {
	m := nested(t)
	all := len(m.rows(0))
	spots := m.barNames()
	if len(spots) != 2 || spots[0].column != "source" || spots[1].column != "assignees" {
		t.Fatalf("the heading over the groups names %+v", spots)
	}
	text := ansi.Strip(lines(m)[m.top()])
	for _, n := range spots {
		if ansi.Cut(text, n.from, n.until) != n.column {
			t.Errorf("%s does not stand at %d: %q", n.column, n.from, text)
		}
	}
	// the control line is no row: the first row the cursor reaches is the first source's heading, under it
	if m.rowsTop() != m.top()+1 || !strings.Contains(lines(m)[m.rowsTop()], "github") {
		t.Fatalf("the rows do not start under the control line:\n%s", view(m))
	}
	// the rule under the header starts at the first column, past the rows' indent and their gutter
	if rule := ansi.Strip(m.rule()); !strings.HasPrefix(rule, strings.Repeat(" ", m.indent(0)+len(lead))+"─") {
		t.Errorf("the rule does not start at the first column: %q", rule)
	}
	m.stages[0].cursor = 2
	if m = drive(m, testkit.Click(spots[1].from+1, m.top())); !m.colMenu.Open || !m.colMenu.heading ||
		m.colMenu.column != "assignees" || m.inRows || len(m.rows(0)) != all {
		t.Fatalf("a click on assignees: menu %+v, in rows %v, %d rows of %d", m.colMenu, m.inRows, len(m.rows(0)), all)
	}
	box, top, _ := m.columnBox()
	if edge := ansi.Strip(box[0]); top != m.top() || !strings.Contains(edge, "╴assignees╶") ||
		!m.columnEnabled(m.columnLine(actUngroup)) || m.columnEnabled(m.columnLine(actGroup)) {
		t.Errorf("the box hangs at %d, %q:\n%s", top, edge, raw(m))
	}
	// Ungroup leaves the rows grouped by source alone, and closes the menu
	m.colMenu.At = m.columnLine(actUngroup)
	if m = drive(m, press(tea.KeyEnter)); m.colMenu.Open || !slices.Equal(m.panel(0).Groups(), []string{"source"}) {
		t.Fatalf("ungroup assignees: menu %v, groups %q", m.colMenu.Open, m.panel(0).Groups())
	}
	// a click on a value's heading puts the cursor on it, folding nothing; one on its ▾ folds it, menu or none; ⏎ there
	// drops none, nor does the keys bar offer one
	all = len(m.rows(0))
	y := rowY(m, m.rows(0)[1].Key)
	if m = drive(m, testkit.Click(8, y)); m.colMenu.Open || len(m.rows(0)) != all || m.stages[0].cursor != 1 {
		t.Fatalf("a click on a value's heading: menu %v, %d rows of %d, cursor %d", m.colMenu.Open, len(m.rows(0)), all,
			m.stages[0].cursor)
	}
	fold := foldAt(m.rows(0)[1])
	if m = drive(m, testkit.Click(fold, y)); m.colMenu.Open || len(m.rows(0)) != 2 {
		t.Fatalf("a click on a value's ▾: menu %v, %d rows", m.colMenu.Open, len(m.rows(0)))
	}
	if last := lines(m)[len(lines(m))-1]; strings.Contains(last, "⏎ menu") {
		t.Errorf("the keys on a value's heading: %q", last)
	}
	if m = drive(m, press(tea.KeyEnter)); m.colMenu.Open {
		t.Error("⏎ on a value's heading dropped a menu")
	}
	if m = drive(m, testkit.Click(fold, y)); len(m.rows(0)) != all {
		t.Errorf("a click on the value's ▸: %d rows of %d", len(m.rows(0)), all)
	}
	// Options… there opens the field's popover, hanging from its name in the heading; a column not grouped has no
	// name there
	m = drive(m, testkit.Click(m.barNames()[0].from, m.top()))
	x, y := m.fieldName("source")
	m.columnAct(m.columnLine(actMore))
	if !m.pop.open || m.pop.field != "source" || m.pop.x != x || m.pop.y != y || y != m.top() {
		t.Errorf("Options… from the heading: %+v, source at %d, %d", m.pop, x, y)
	}
	m.pop = popover{}
	m.colMenu = columnMenu{}
	// a click on the control line off its names and folds does nothing where the stage has no presets
	if m = drive(m, testkit.Click(m.paneWidth()/2, m.top())); m.colMenu.Open || m.onPreset {
		t.Errorf("a click on the control line without presets: menu %v, on the preset line %v", m.colMenu.Open, m.onPreset)
	}
	if _, y := m.fieldName("age"); y != -1 {
		t.Errorf("age, not grouped, stands in the heading on line %d", y)
	}
	// while the stage loads, the rule has no sort targets
	m.stages[0].loading = true
	if spots := m.sortSpots(); spots != nil {
		t.Errorf("the sort targets while loading: %+v", spots)
	}
	m.stages[0].loading = false
	// once the heading no longer names the column, the menu draws nothing, and a click closes it
	m = drive(m, testkit.Click(m.barNames()[0].from, m.top()))
	m.setPanel(m.panel(0).Grouped("source", true))
	if box, _, _ := m.columnBox(); box != nil {
		t.Error("the menu of a heading gone still draws")
	}
	if m = drive(m, testkit.Click(1, m.rowsTop())); m.colMenu.Open {
		t.Error("a click did not close the menu of a heading gone")
	}
}

// The names in the heading over the groups are found where it says them, each before the next level's fold, the
// count, or the end; a value that holds a column's name in it is passed over.
func TestTheHeadingOverTheGroupsNamesItsColumnsWhereItSaysThem(t *testing.T) {
	groups := []string{"source", "assignees"}
	text := "▾ source ▾ assignees  3"
	var got []string
	for _, n := range namesIn(text, groups) {
		got = append(got, fmt.Sprintf("%s@%d", n.column, n.from))
		if cut := ansi.Cut("  "+text, n.from, n.until); cut != n.column {
			t.Errorf("%s stands at %d, where the heading draws %q", n.column, n.from, cut)
		}
	}
	if len(got) != 2 {
		t.Errorf("the names of two columns: %q", got)
	}
	if names := namesIn("▾ sourcery source ▾ assignees  3", groups); len(names) != 2 ||
		names[0].from != barLeft+ansi.StringWidth("▾ sourcery ") {
		t.Errorf("past a value that holds source: %+v", names)
	}
	if names := namesIn("", groups); names != nil {
		t.Errorf("no heading names %+v", names)
	}
}

// levels gives how many headings of each level show in the nested backlog, the sources' and the assignees', and how
// many rows.
func levels(m Model) (sources, assignees, rows int) {
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

// glyph gives the ▾ or ▸ the control line draws before a column's name, or, for none, after the last.
func glyph(m Model, column string) string {
	names := m.barNames()
	x := names[len(names)-1].until + 1
	for _, n := range names {
		if n.column == column {
			x = n.from - 2
		}
	}
	return ansi.Strip(ansi.Cut(lines(m)[m.top()], x, x+1))
}

// A click on the ▾ before a column's name in the heading over the groups closes its level: its headings, and
// everything under them, no longer show, and the levels under it show ▸ too; nothing else changes, so a click on its
// ▸ shows the level again as it was, each heading open or closed as it was left. With the outermost level closed,
// nothing shows under the heading over the groups; and the header stands under the last line, where no row shows.
func TestALevelsFoldInTheHeadingOverTheGroupsOpensAndClosesIt(t *testing.T) {
	m := nested(t)
	all := len(m.rows(0))
	spots := m.barNames()
	s0, a0, r0 := levels(m)
	if glyph(m, "source") != grid.Unfolded || glyph(m, "assignees") != grid.Unfolded || glyph(m, "") != grid.Unfolded {
		t.Fatalf("the folds: %q", ansi.Strip(lines(m)[m.top()]))
	}
	// one assignee's heading closed by hand, to be found closed again
	ada := slices.IndexFunc(m.rows(0), func(r row) bool { return strings.HasSuffix(r.Key, "\x00ada") })
	m = drive(m, testkit.Click(foldAt(m.rows(0)[ada]), rowY(m, m.rows(0)[ada].Key)))
	_, _, r1 := levels(m)
	// assignees' closes the assignees: the sources stand alone, each ▸, as nothing shows under it
	m = drive(m, testkit.Click(spots[1].from-2, m.top()))
	if s, a, r := levels(m); s != s0 || a != 0 || r != 0 || m.colMenu.Open || glyph(m, "assignees") != grid.Folded ||
		glyph(m, "") != grid.Folded || glyph(m, "source") != grid.Unfolded ||
		!strings.Contains(lines(m)[m.rowsTop()], grid.Folded+" github") {
		t.Fatalf("assignees closed: %d sources, %d assignees, %d rows\n%s", s, a, r, view(m))
	}
	// source's closes the sources: nothing shows under the heading over the groups, and the header stands under it
	if m = drive(m, testkit.Click(spots[0].from-2, m.top())); len(m.rows(0)) != 1 || glyph(m, "source") != grid.Folded ||
		m.headerTop() != m.rowsTop() {
		t.Fatalf("source closed: %d rows, header at %d\n%s", len(m.rows(0)), m.headerTop(), view(m))
	}
	// source's again shows the sources, the assignees still closed, as they were
	if m = drive(m, testkit.Click(spots[0].from-2, m.top())); func() bool { s, a, r := levels(m); return s != s0 || a != 0 || r != 0 }() ||
		glyph(m, "assignees") != grid.Folded {
		s, a, r := levels(m)
		t.Fatalf("source opened again: %d sources, %d assignees, %d rows\n%s", s, a, r, view(m))
	}
	// assignees' again shows everything as it was: ada's heading still closed
	if m = drive(m, testkit.Click(spots[1].from-2, m.top())); func() bool { s, a, r := levels(m); return s != s0 || a != a0 || r != r1 }() ||
		m.stages[0].open.Open(m.rows(0)[ada]) {
		s, a, r := levels(m)
		t.Fatalf("assignees opened again: %d sources, %d assignees, %d rows, were %d %d %d\n%s", s, a, r, s0, a0, r1,
			view(m))
	}
	// with the outermost closed, a click on the rows' ▸ opens every level over them, and they show as they were
	m = drive(m, testkit.Click(spots[0].from-2, m.top()), testkit.Click(spots[1].until+1, m.top()))
	if s, a, r := levels(m); s != s0 || a != a0 || r != r1 || glyph(m, "source") != grid.Unfolded {
		t.Fatalf("the rows' ▸ under every level closed: %d sources, %d assignees, %d rows", s, a, r)
	}
	if r0 <= r1 || len(m.rows(0)) >= all {
		t.Errorf("ada's heading hid no row: %d rows, %d with it closed", r0, r1)
	}
}

// The ▾ or ▸ after the last name in the heading over the groups opens and closes the rows under the groups: closed,
// every heading stands, each ▸, the header under the last of them; open again, the rows show as they were.
func TestTheFoldAfterTheLastNameOpensAndClosesTheRows(t *testing.T) {
	m := nested(t)
	all := len(m.rows(0))
	s0, a0, _ := levels(m)
	rows := m.barNames()[len(m.barNames())-1].until + 1
	if m = drive(m, testkit.Click(rows, m.top())); func() bool { s, a, r := levels(m); return s != s0 || a != a0 || r != 0 }() ||
		glyph(m, "") != grid.Folded || glyph(m, "assignees") != grid.Unfolded || m.headerTop() != m.rowsTop()+s0+a0 {
		s, a, r := levels(m)
		t.Fatalf("the rows closed: %d sources, %d assignees, %d rows, the header at %d\n%s", s, a, r, m.headerTop(),
			view(m))
	}
	// a value's heading opened shows its rows, and every other heading's with them: the level under it opens
	k := slices.IndexFunc(m.rows(0), func(r row) bool { return grid.Level(r) == 3 })
	if m = drive(m, testkit.Click(foldAt(m.rows(0)[k]), rowY(m, m.rows(0)[k].Key))); len(m.rows(0)) != all ||
		glyph(m, "") != grid.Unfolded {
		t.Fatalf("a heading opened under the rows closed: %d rows of %d", len(m.rows(0)), all)
	}
	// Expand All opens every level
	m = drive(m, testkit.Click(rows, m.top()))
	m.stages[0].cursor, m.inRows = 1, true
	if next, _ := m.folding("Expand All"); !next.Shows(len(m.panel(0).Groups())) {
		t.Errorf("expand all leaves the rows closed: %v", next)
	}
}
