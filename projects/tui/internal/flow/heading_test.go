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
	"github.com/empowerite/kit/projects/tui/internal/style"
	"github.com/empowerite/kit/projects/tui/internal/table"
	"github.com/empowerite/kit/projects/tui/internal/testkit"
)

// The heading over the groups the person chose stands on the control line, apart from the rows, and names their
// columns; a click on a name drops that column's menu around it, as the header's names do, the control line under
// the keyboard. Ungroup there leaves the rows grouped by the others. A click on a value's heading, which names no
// column, puts the cursor on it, and one on its ▾ folds its rows away, and another unfolds them; ⏎ there drops no menu.
func TestANameInTheHeadingOverTheGroupsDropsItsColumnMenu(t *testing.T) {
	m := nested(t)
	all := len(m.rows(0))
	spots := table.BarNames(&m)
	if len(spots) != 2 || spots[0].Column != "source" || spots[1].Column != "assignees" {
		t.Fatalf("the heading over the groups names %+v", spots)
	}
	text := ansi.Strip(lines(m)[m.top()])
	for _, n := range spots {
		if ansi.Cut(text, n.From, n.Until) != n.Column {
			t.Errorf("%s does not stand at %d: %q", n.Column, n.From, text)
		}
	}
	// the control line is no row: the first row the cursor reaches is the first source's heading, under it and a blank
	// line
	if m.rowsTop() != m.top()+2 || lines(m)[m.top()+1] != "" || !strings.Contains(lines(m)[m.rowsTop()], "github") {
		t.Fatalf("the rows do not start under the control line:\n%s", view(m))
	}
	// the rule under the header starts at the first column, past the rows' indent and their gutter, with its line or,
	// under a name that can sort, the name's mark
	gap := strings.Repeat(" ", m.indent(0)+len(lead))
	if rule := ansi.Strip(m.rule()); !strings.HasPrefix(rule, gap+"─") && !strings.HasPrefix(rule, gap+Sortable) {
		t.Errorf("the rule does not start at the first column: %q", rule)
	}
	m.stages[0].Cursor = 2
	if m = drive(m, testkit.Click(spots[1].From+1, m.top())); !m.colMenu.Open || !m.colMenu.Heading ||
		m.colMenu.Column != "assignees" || m.inRows || len(m.rows(0)) != all {
		t.Fatalf("a click on assignees: menu %+v, in rows %v, %d rows of %d", m.colMenu, m.inRows, len(m.rows(0)), all)
	}
	box, top, _ := m.colMenu.Box(&m)
	if edge := ansi.Strip(box[0]); top != m.top() || !strings.Contains(edge, "╴assignees╶") ||
		!m.colMenu.Enabled(&m, m.colMenu.LineOf(&m, table.ActUngroup)) || m.colMenu.Enabled(&m, m.colMenu.LineOf(&m, table.ActGroup)) {
		t.Errorf("the box hangs at %d, %q:\n%s", top, edge, raw(m))
	}
	// Ungroup leaves the rows grouped by source alone, and closes the menu
	m.colMenu.At = m.colMenu.LineOf(&m, table.ActUngroup)
	if m = drive(m, press(tea.KeyEnter)); m.colMenu.Open || !slices.Equal(m.panel(0).Groups(), []string{"source"}) {
		t.Fatalf("ungroup assignees: menu %v, groups %q", m.colMenu.Open, m.panel(0).Groups())
	}
	// a click on a value's heading puts the cursor on it, folding nothing; one on its ▾ folds it, menu or none; ⏎ there
	// drops none, nor does the keys bar offer one
	all = len(m.rows(0))
	y := rowY(m, m.rows(0)[1].Key)
	if m = drive(m, testkit.Click(8, y)); m.colMenu.Open || len(m.rows(0)) != all || m.stages[0].Cursor != 1 {
		t.Fatalf("a click on a value's heading: menu %v, %d rows of %d, cursor %d", m.colMenu.Open, len(m.rows(0)), all,
			m.stages[0].Cursor)
	}
	fold := table.FoldAt(m.rows(0)[1])
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
	m = drive(m, testkit.Click(table.BarNames(&m)[0].From, m.top()))
	x, y := m.fieldName("source")
	m.columnAct(m.colMenu.LineOf(&m, table.ActMore))
	if !m.pop.Open || m.pop.Field != "source" || m.pop.X != x || m.pop.Y != y || y != m.top() {
		t.Errorf("Options… from the heading: %+v, source at %d, %d", m.pop, x, y)
	}
	m.pop = table.Popover{}
	m.colMenu = table.ColumnMenu{}
	// a click on the control line off its names and folds does nothing where the stage has no presets
	if m = drive(m, testkit.Click(m.paneWidth()/2, m.top())); m.colMenu.Open || m.onPreset {
		t.Errorf("a click on the control line without presets: menu %v, on the preset line %v", m.colMenu.Open, m.onPreset)
	}
	if _, y := m.fieldName("age"); y != -1 {
		t.Errorf("age, not grouped, stands in the heading on line %d", y)
	}
	// while the stage loads, the rule has no sort targets
	m.stages[0].src.Loading = true
	if spots := m.sortSpots(); spots != nil {
		t.Errorf("the sort targets while loading: %+v", spots)
	}
	m.stages[0].src.Loading = false
	// once the heading no longer names the column, the menu draws nothing, and a click closes it
	m = drive(m, testkit.Click(table.BarNames(&m)[0].From, m.top()))
	m.setPanel(m.panel(0).Grouped("source", true))
	if box, _, _ := m.colMenu.Box(&m); box != nil {
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
	for _, n := range table.NamesIn(text, groups) {
		got = append(got, fmt.Sprintf("%s@%d", n.Column, n.From))
		if cut := ansi.Cut("  "+text, n.From, n.Until); cut != n.Column {
			t.Errorf("%s stands at %d, where the heading draws %q", n.Column, n.From, cut)
		}
	}
	if len(got) != 2 {
		t.Errorf("the names of two columns: %q", got)
	}
	if names := table.NamesIn("▾ sourcery source ▾ assignees  3", groups); len(names) != 2 ||
		names[0].From != table.BarLeft+ansi.StringWidth("▾ sourcery ") {
		t.Errorf("past a value that holds source: %+v", names)
	}
	if names := table.NamesIn("", groups); names != nil {
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
	names := table.BarNames(&m)
	x := names[len(names)-1].Until + 1
	for _, n := range names {
		if n.Column == column {
			x = n.From - 2
		}
	}
	return ansi.Strip(ansi.Cut(lines(m)[m.top()], x, x+1))
}

// A click on the ▾ before a column's name in the heading over the groups closes its level: its headings, and
// everything under them, no longer show. A level whose every level over it has one value, its headings frozen over
// the header, folds as the first level does, so it shows a dim · where its fold would stand, which a click does
// nothing on; the rows' fold stands after the last name while a level has more than one value. With the outermost
// level closed, nothing shows under the heading over the groups, and a click on the rows' ▸ shows every level again,
// each heading open or closed as it was left.
func TestALevelsFoldInTheHeadingOverTheGroupsOpensAndClosesIt(t *testing.T) {
	m := nested(t)
	all := len(m.rows(0))
	spots := table.BarNames(&m)
	s0, a0, r0 := levels(m)
	// source has one value, github, so the assignees fold as the sources do
	if glyph(m, "source") != grid.Unfolded || glyph(m, "assignees") != grid.Joined || glyph(m, "") != grid.Unfolded {
		t.Fatalf("the folds: %q", ansi.Strip(lines(m)[m.top()]))
	}
	if m = drive(m, testkit.Click(spots[1].From-2, m.top())); len(m.rows(0)) != all || m.colMenu.Open {
		t.Fatalf("a click on assignees' ·: %d rows of %d, menu %v", len(m.rows(0)), all, m.colMenu.Open)
	}
	// one assignee's heading closed by hand, to be found closed again
	ada := slices.IndexFunc(m.rows(0), func(r row) bool { return strings.HasSuffix(r.Key, "\x00ada") })
	m = drive(m, testkit.Click(table.FoldAt(m.rows(0)[ada]), rowY(m, m.rows(0)[ada].Key)))
	_, _, r1 := levels(m)
	// the rows' fold closes the rows alone: the headings stand, each assignee ▸
	if m = drive(m, testkit.Click(spots[1].Until+1, m.top())); func() bool { s, a, r := levels(m); return s != s0 || a != a0 || r != 0 }() ||
		glyph(m, "") != grid.Folded {
		s, a, r := levels(m)
		t.Fatalf("the rows closed: %d sources, %d assignees, %d rows\n%s", s, a, r, view(m))
	}
	// source's closes the sources: nothing shows under the heading over the groups, nor the header, over no row
	if m = drive(m, testkit.Click(spots[0].From-2, m.top())); len(m.rows(0)) != 1 || glyph(m, "source") != grid.Folded ||
		m.headerTop() != -1 {
		t.Fatalf("source closed: %d rows, header at %d\n%s", len(m.rows(0)), m.headerTop(), view(m))
	}
	// with the outermost closed, a click on the rows' ▸ opens every level over them, and they show as they were
	m = drive(m, testkit.Click(spots[1].Until+1, m.top()))
	if s, a, r := levels(m); s != s0 || a != a0 || r != r1 || glyph(m, "source") != grid.Unfolded {
		t.Fatalf("the rows' ▸ under every level closed: %d sources, %d assignees, %d rows", s, a, r)
	}
	if r0 <= r1 || len(m.rows(0)) >= all {
		t.Errorf("ada's heading hid no row: %d rows, %d with it closed", r0, r1)
	}
}

// The ▾ or ▸ after the last name in the heading over the groups opens and closes the rows under the groups: closed,
// every heading stands, each ▸, and no header, as no row shows; open again, the rows show as they were.
func TestTheFoldAfterTheLastNameOpensAndClosesTheRows(t *testing.T) {
	m := nested(t)
	all := len(m.rows(0))
	s0, a0, _ := levels(m)
	rows := table.BarNames(&m)[len(table.BarNames(&m))-1].Until + 1
	if m = drive(m, testkit.Click(rows, m.top())); func() bool { s, a, r := levels(m); return s != s0 || a != a0 || r != 0 }() ||
		glyph(m, "") != grid.Folded || glyph(m, "assignees") != grid.Joined || m.headerTop() != -1 {
		s, a, r := levels(m)
		t.Fatalf("the rows closed: %d sources, %d assignees, %d rows, the header at %d\n%s", s, a, r, m.headerTop(),
			view(m))
	}
	// a value's heading opened shows its rows, and every other heading's with them: the level under it opens
	k := slices.IndexFunc(m.rows(0), func(r row) bool { return grid.Level(r) == 3 })
	if m = drive(m, testkit.Click(table.FoldAt(m.rows(0)[k]), rowY(m, m.rows(0)[k].Key))); len(m.rows(0)) != all ||
		glyph(m, "") != grid.Unfolded {
		t.Fatalf("a heading opened under the rows closed: %d rows of %d", len(m.rows(0)), all)
	}
	// Expand All opens every level
	m = drive(m, testkit.Click(rows, m.top()))
	m.stages[0].Cursor, m.inRows = 1, true
	if next, _ := table.Folding(&m, "Expand All"); !next.Shows(len(m.panel(0).Groups())) {
		t.Errorf("expand all leaves the rows closed: %v", next)
	}
}

// The headings of the outermost levels that have one value alone are frozen over the header, as a pane freezes in a
// spreadsheet: they never scroll, so the rows under them keep their context; the header stands over the first item of
// the first level of more than one value, which scrolls under it with everything after it. Where the outermost level
// has more than one value, nothing is frozen, and the header stands over its first heading. A blank line sets the
// header apart from the frozen headings over it. A click on a frozen heading's ▾ folds it; with no row to stand over,
// the header goes.
func TestTheHeadingsOfOneValueAreFrozenOverTheHeader(t *testing.T) {
	m := loaded(t, options(), tea.WindowSizeMsg{Width: 120, Height: 14})
	m.setPanel(regroup(m.panel(m.stage), "source", "assignees"))
	github := lines(m)[m.rowsTop()]
	if table.Frozen(&m) != 2 || !strings.Contains(github, "github") || m.headerTop() != m.rowsTop()+2 ||
		lines(m)[m.rowsTop()+1] != "" || !strings.Contains(lines(m)[m.headerTop()+2], "ada") {
		t.Fatalf("frozen %d, the header at %d:\n%s", table.Frozen(&m), m.headerTop(), view(m))
	}
	// down to the last row: the rows scroll under the header, which stays, under github's heading
	m = drive(m, down, down, down, down, down, down, down, down, down)
	if s := m.stages[0]; s.Offset == 0 || lines(m)[m.rowsTop()] != github || m.headerTop() != m.rowsTop()+2 ||
		!strings.Contains(lines(m)[m.headerTop()+2], "above") {
		t.Fatalf("scrolled to the last row, offset %d:\n%s", s.Offset, view(m))
	}
	// a click on github's ▾, frozen, folds it, and the header goes, as no row shows
	if m = drive(m, testkit.Click(table.FoldAt(m.rows(0)[1]), m.rowsTop())); len(m.rows(0)) != 2 || m.headerTop() != -1 {
		t.Fatalf("a click on the frozen heading's ▾: %d rows, the header at %d\n%s", len(m.rows(0)), m.headerTop(), view(m))
	}
	// grouped by assignees first, of several values, nothing is frozen: the header stands over ada's heading
	m = drive(m, testkit.Click(table.FoldAt(m.rows(0)[1]), m.rowsTop()))
	m.setPanel(regroup(m.panel(m.stage), "assignees", "source"))
	if table.Frozen(&m) != 1 || m.headerTop() != m.rowsTop() || !strings.Contains(lines(m)[m.rowsTop()+2], "ada") {
		t.Errorf("grouped by assignees: frozen %d, the header at %d:\n%s", table.Frozen(&m), m.headerTop(), view(m))
	}
}

// The band under the cursor runs the pane's width, from its first cell to its last, on a heading as on a row under
// its indent.
func TestTheCursorsBandRunsThePanesWidth(t *testing.T) {
	m := nested(t)
	open, _, _ := strings.Cut(style.Band("x", 1), "x")
	for _, k := range []int{1, 3} {
		m.stages[0].Cursor, m.inRows = k, true
		m.rebuild()
		l := m.allLines(0)[slices.IndexFunc(m.allLines(0), func(l table.Line) bool { return l.Row == k })]
		if open == "" || !strings.HasPrefix(l.Text, open) || ansi.StringWidth(l.Text) != m.paneWidth() {
			t.Errorf("row %d's band, %d cells of %d: %q", k, ansi.StringWidth(l.Text), m.paneWidth(), l.Text)
		}
	}
}

// Grouped by assignees, then source, each assignee's one source heads the same rows, so it stands on the assignee's
// line, across: one node, with one fold, before the assignee, which a click opens and closes; a dim · stands before the
// source, and a click on it folds nothing.
func TestAHeadingAcrossHasOneFold(t *testing.T) {
	m := loaded(t, options(), tea.WindowSizeMsg{Width: 120, Height: 30})
	m.setPanel(regroup(m.panel(m.stage), "assignees", "source"))
	k := slices.IndexFunc(m.rows(0), func(r row) bool { return strings.HasSuffix(r.Key, "\x00ada") })
	folds, text := table.FoldsAt(m.rows(0)[k]), ansi.Strip(m.rows(0)[k].Cells[0].Values[0])
	if len(folds) != 1 || !strings.HasPrefix(text, grid.Unfolded+" ada "+grid.Joined+" ") {
		t.Fatalf("ada's line: %q, folds at %v", text, folds)
	}
	all, y := len(m.rows(0)), rowY(m, m.rows(0)[k].Key)
	joined := folds[0] + ansi.StringWidth(grid.Unfolded+" ada ")
	if m = drive(m, testkit.Click(joined, y)); len(m.rows(0)) != all {
		t.Fatalf("a click on the · on ada's line folded it: %d rows of %d", len(m.rows(0)), all)
	}
	if m = drive(m, testkit.Click(folds[0], y)); len(m.rows(0)) != all-1 || m.stages[0].Open.Open(m.rows(0)[k]) {
		t.Fatalf("a click on ada's fold: %d rows of %d", len(m.rows(0)), all)
	}
	if m = drive(m, testkit.Click(folds[0], y)); len(m.rows(0)) != all {
		t.Errorf("a click on ada's fold again: %d rows of %d", len(m.rows(0)), all)
	}
}
