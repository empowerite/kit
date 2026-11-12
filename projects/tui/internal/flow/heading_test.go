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
	"github.com/empowerite/kit/projects/tui/internal/pane"
	"github.com/empowerite/kit/projects/tui/internal/style"
	"github.com/empowerite/kit/projects/tui/internal/table"
	"github.com/empowerite/kit/projects/tui/internal/testkit"
)

// The heading over the groups the person chose stands on the control line, apart from the rows, and names their
// columns; a click on a name opens that column's card around it, as the header's names do, the control line under
// the keyboard. Its Group box, unticked, leaves the rows grouped by the others. A click on a value's heading, which
// names no column, puts the cursor on it, and one on its ▾ folds its rows away, and another unfolds them; ⏎ there opens
// no card.
func TestANameInTheHeadingOverTheGroupsOpensItsCard(t *testing.T) {
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
	// the control line is no row: the first row the cursor reaches is the first source's heading, under it and the rule
	// under the heading over the groups, as wide as the heading
	if m.rowsTop() != m.top()+2 || strings.TrimRight(ansi.Strip(lines(m)[m.top()+1]), " ") != ansi.Strip(table.BarRule(&m)) ||
		!strings.Contains(lines(m)[m.rowsTop()], "github") {
		t.Fatalf("the rows do not start under the control line:\n%s", view(m))
	}
	// the rule under the header starts at the first column, past the rows' indent and their gutter, with its line or,
	// under a name that can sort, the name's mark
	gap := strings.Repeat(" ", m.indent(0)+len(lead))
	if rule := ansi.Strip(m.rule()); !strings.HasPrefix(rule, gap+"─") && !strings.HasPrefix(rule, gap+Sortable) {
		t.Errorf("the rule does not start at the first column: %q", rule)
	}
	m.stages[0].Cursor = 2
	if m = drive(m, testkit.Click(spots[1].From+1, m.top())); !m.pop.Open || !m.pop.Heading ||
		m.pop.Field != "assignees" || m.inRows || len(m.rows(0)) != all {
		t.Fatalf("a click on assignees: card %+v, in rows %v, %d rows of %d", m.pop, m.inRows, len(m.rows(0)), all)
	}
	grouped := table.GroupIt + " " + grid.GroupMark(m.panel(0).Field("assignees").Depth)
	box, top, left := m.pop.Box(&m)
	if edge := ansi.Strip(box[0]); top > m.top() || left+3 != spots[1].From || !strings.Contains(edge, " assignees ") ||
		cardLine(m, grouped) < 0 {
		t.Errorf("the box hangs at %d, %d, %q:\n%s", top, left, edge, raw(m))
	}
	// unticking Group leaves the rows grouped by source alone, and the card stays
	m.pop.At = cardLine(m, grouped)
	if m = drive(m, press(tea.KeyEnter)); !m.pop.Open || !slices.Equal(m.panel(0).Groups(), []string{"source"}) {
		t.Fatalf("ungroup assignees: card %v, groups %q", m.pop.Open, m.panel(0).Groups())
	}
	m = drive(m, esc)
	// a click on a value's heading puts the cursor on it, folding nothing; one on its ▾ folds it, menu or none; ⏎ there
	// drops none, nor does the keys bar offer one
	all = len(m.rows(0))
	y := rowY(m, m.rows(0)[1].Key)
	if m = drive(m, testkit.Click(8, y)); m.pop.Open || len(m.rows(0)) != all || m.stages[0].Cursor != 1 {
		t.Fatalf("a click on a value's heading: card %v, %d rows of %d, cursor %d", m.pop.Open, len(m.rows(0)), all,
			m.stages[0].Cursor)
	}
	fold := table.FoldAt(m.rows(0)[1])
	if m = drive(m, testkit.Click(fold, y)); m.pop.Open || len(m.rows(0)) != 2 {
		t.Fatalf("a click on a value's ▾: card %v, %d rows", m.pop.Open, len(m.rows(0)))
	}
	if last := lines(m)[len(lines(m))-1]; strings.Contains(last, "⏎ settings") {
		t.Errorf("the keys on a value's heading: %q", last)
	}
	if m = drive(m, press(tea.KeyEnter)); m.pop.Open {
		t.Error("⏎ on a value's heading opened a card")
	}
	if m = drive(m, testkit.Click(fold, y)); len(m.rows(0)) != all {
		t.Errorf("a click on the value's ▸: %d rows of %d", len(m.rows(0)), all)
	}
	// source's card hangs from its name in the heading; a column not grouped has no name there
	m = drive(m, testkit.Click(table.BarNames(&m)[0].From, m.top()))
	x, y := m.fieldName("source")
	if !m.pop.Open || m.pop.Field != "source" || m.pop.X != x || m.pop.Y != y || y != m.top() {
		t.Errorf("source's card from the heading: %+v, source at %d, %d", m.pop, x, y)
	}
	m.pop = table.Popover{}
	// a click on the control line off its names and folds does nothing where the stage has no presets
	if m = drive(m, testkit.Click(m.paneWidth()/2, m.top())); m.pop.Open || m.onPreset {
		t.Errorf("a click on the control line without presets: card %v, on the preset line %v", m.pop.Open, m.onPreset)
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
	// once the heading no longer names the column, the card stays where it was, and a click off it closes it
	m = drive(m, testkit.Click(table.BarNames(&m)[0].From, m.top()))
	x = m.pop.X
	m.setPanel(m.panel(0).Grouped("source", true))
	if box, _, _ := m.pop.Box(&m); box == nil || m.pop.X != x {
		t.Error("the card of a heading gone moved")
	}
	if m = drive(m, testkit.Click(m.paneWidth()-1, m.rowsTop()+20)); m.pop.Open {
		t.Error("a click did not close the card of a heading gone")
	}
	// a click on the rule under the heading, on a group's sort mark, sorts the groups by it
	m = nested(t)
	rule := ansi.Strip(lines(m)[m.top()+1])
	x = ansi.StringWidth(rule[:strings.Index(rule, Sortable)])
	if m = drive(m, testkit.Click(x, m.top()+1)); m.panel(0).Mark("source") == "" {
		t.Errorf("a click on source's mark on the rule %q leaves it unsorted", rule)
	}
}

// The names in the heading over the groups are found where it says them, each before the next level's fold, the
// count, or the end; a value that holds a column's name in it is passed over.
func TestTheHeadingOverTheGroupsNamesItsColumnsWhereItSaysThem(t *testing.T) {
	groups, same := []string{"source", "assignees"}, func(id string) string { return id }
	text := "▾ source ▾ assignees  3"
	var got []string
	for _, n := range table.NamesIn(text, groups, same) {
		got = append(got, fmt.Sprintf("%s@%d", n.Column, n.From))
		if cut := ansi.Cut("  "+text, n.From, n.Until); cut != n.Column {
			t.Errorf("%s stands at %d, where the heading draws %q", n.Column, n.From, cut)
		}
	}
	if len(got) != 2 {
		t.Errorf("the names of two columns: %q", got)
	}
	if names := table.NamesIn("▾ sourcery source ▾ assignees  3", groups, same); len(names) != 2 ||
		names[0].From != table.BarLeft+ansi.StringWidth("▾ sourcery ") {
		t.Errorf("past a value that holds source: %+v", names)
	}
	if names := table.NamesIn("", groups, same); names != nil {
		t.Errorf("no heading names %+v", names)
	}
}

// levels gives how many headings of each level show in the nested issues, the sources' and the assignees', and how
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
	if m = drive(m, testkit.Click(spots[1].From-2, m.top())); len(m.rows(0)) != all || m.pop.Open {
		t.Fatalf("a click on assignees' ·: %d rows of %d, card %v", len(m.rows(0)), all, m.pop.Open)
	}
	// from the keyboard, assignees' stop folds as source's does, and the rows' stop stands where its fold is drawn
	if stops := m.stops(); len(stops) != 3 || stops[1].Level != 0 || stops[2].Level != 2 {
		t.Fatalf("the stops: %+v", stops)
	}
	// the levels of issues set as Underway, source, organization, and repository, each of one value: a stop each, and
	// none for the rows
	single := loaded(t, options(), tea.WindowSizeMsg{Width: 120, Height: 30})
	if single.apply(single.viewsOf(0).Chosen("Underway")); len(single.stops()) != 3 ||
		slices.ContainsFunc(single.stops(), func(s table.Stop) bool { return s.Column == "" }) {
		t.Errorf("three levels of one value, ▾ source · organization · repository 4: stops %+v", single.stops())
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
// every heading stands, each ▸, and no header, as no row shows; one heading shown shows its rows alone.
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
	// a value's heading opened shows its rows alone, as Show shows one node: every other heading stays folded
	k := slices.IndexFunc(m.rows(0), func(r row) bool { return grid.Level(r) == 3 })
	key := m.rows(0)[k].Key
	m = drive(m, testkit.Click(table.FoldAt(m.rows(0)[k]), rowY(m, key)))
	folded := 0
	for _, r := range m.rows(0) {
		if grid.Level(r) == 3 && r.Key != key && !m.stages[0].Open.Unfolds(r) {
			folded++
		}
	}
	if _, _, r := levels(m); r == 0 || folded == 0 || len(m.rows(0)) >= all {
		t.Fatalf("a heading opened under the rows closed: %d rows of %d, %d others folded", len(m.rows(0)), all, folded)
	}
	// Show More shows the level, every heading of it
	m.stages[0].Cursor, m.inRows = 1, true
	next, _ := table.Folding(&m, "Show More")
	if table.Reoutline(&m, next); !next.Shows(len(m.panel(0).Groups())) || len(m.rows(0)) != all {
		t.Errorf("show more leaves rows hidden: %d rows of %d", len(m.rows(0)), all)
	}
}

// The headings of the outermost levels that have one value alone are frozen over the header, as a pane freezes in a
// spreadsheet: they never scroll, so the rows under them keep their context; the header stands over the first item of
// the first level of more than one value, which scrolls under it with everything after it. Where the outermost level
// has more than one value, nothing is frozen, and the header stands over its first heading. A blank line sets the
// header apart from the frozen headings over it. A click on a frozen heading's ▾ folds it; with no row to stand over,
// the header goes.
func TestTheHeadingsOfOneValueAreFrozenOverTheHeader(t *testing.T) {
	m := loaded(t, options(), tea.WindowSizeMsg{Width: 120, Height: 14 + titleHeight})
	m.setPanel(regroup(m.panel(m.stage), "source", "assignees"))
	github := lines(m)[m.rowsTop()]
	if table.Frozen(&m) != 2 || !strings.Contains(github, "github") || m.headerTop() != m.rowsTop()+2 ||
		lines(m)[m.rowsTop()+1] != "" || !strings.Contains(lines(m)[m.headerTop()+2], "ada") {
		t.Fatalf("frozen %d, the header at %d:\n%s", table.Frozen(&m), m.headerTop(), view(m))
	}
	// down to the last row: the rows scroll under the header, which stays, under github's heading
	m = drive(m, down, down, down, down, down, down, down, down, down)
	if s := m.stages[0]; s.Offset == 0 || lines(m)[m.rowsTop()] != github || m.headerTop() != m.rowsTop()+2 ||
		!strings.Contains(lines(m)[m.headerTop()+2], pane.Up) {
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
	k := slices.IndexFunc(m.rows(0), func(r row) bool { return strings.HasSuffix(r.Key, "\x00jeffrey-aguilera") })
	folds, text := table.FoldsAt(m.rows(0)[k]), ansi.Strip(m.rows(0)[k].Cells[0].Values[0])
	if len(folds) != 1 || !strings.HasPrefix(text, grid.Unfolded+" jeffrey-aguilera "+grid.Joined+" ") {
		t.Fatalf("jeffrey-aguilera's line: %q, folds at %v", text, folds)
	}
	all, y := len(m.rows(0)), rowY(m, m.rows(0)[k].Key)
	joined := folds[0] + ansi.StringWidth(grid.Unfolded+" jeffrey-aguilera ")
	if m = drive(m, testkit.Click(joined, y)); len(m.rows(0)) != all {
		t.Fatalf("a click on the · on jeffrey-aguilera's line folded it: %d rows of %d", len(m.rows(0)), all)
	}
	if m = drive(m, testkit.Click(folds[0], y)); len(m.rows(0)) != all-2 || m.stages[0].Open.Open(m.rows(0)[k]) {
		t.Fatalf("a click on jeffrey-aguilera's fold: %d rows of %d", len(m.rows(0)), all)
	}
	if m = drive(m, testkit.Click(folds[0], y)); len(m.rows(0)) != all {
		t.Errorf("a click on jeffrey-aguilera's fold again: %d rows of %d", len(m.rows(0)), all)
	}
}
