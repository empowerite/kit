// The outline: a group's heading, a row, and the header open and close alike, by the keys, a click, and the tab's
// menu, whose fold items act on everything or on the selection.
package flow

import (
	"slices"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/empowerite/kit/projects/tui/internal/grid"
	"github.com/empowerite/kit/projects/tui/internal/testkit"
)

// nested gives backlog grouped by hand, by source, then by assignees within it, at 120 columns: the heading over the
// groups, a heading per source under it, a heading per assignee under each, and their rows.
func nested(t *testing.T) Model {
	t.Helper()
	m := loaded(t, options(), tea.WindowSizeMsg{Width: 120, Height: 30})
	m.setPanel(regroup(m.panel(m.stage), "source", "assignees"))
	if rows := m.rows(0); !grid.OverGroups(rows[0]) || grid.Level(rows[1]) != 2 || grid.Level(rows[2]) != 3 {
		t.Fatalf("backlog does not group two deep:\n%s", view(m))
	}
	return m
}

// at gives the row under the selected stage's cursor.
func at(m Model) row { return m.rows(m.stage)[m.stages[m.stage].cursor] }

// Right opens the node under the cursor, or on a heading already open steps into its first row; left closes it, or on
// a node closed, or one that hides nothing, steps out to the heading over it, as a tree does. With option, either
// opens or closes everything under the node too, as an option-click on a heading does.
func TestTheOutlineOpensClosesAndStepsAlongItsTree(t *testing.T) {
	m := nested(t)
	all := len(m.rows(0))
	// down lands past the headings on a row; left steps out to its heading, and again closes that heading
	m = drive(m, down)
	if grid.Heads(at(m)) {
		t.Fatalf("down landed on a heading")
	}
	if m = drive(m, left); grid.Level(at(m)) != 3 {
		t.Fatalf("left on a row closed did not step out to its heading: %q", at(m).Key)
	}
	inner := at(m).Key
	if m = drive(m, left); len(m.rows(0)) >= all || at(m).Key != inner || m.stages[0].open.Open(at(m)) {
		t.Fatalf("left on an open heading: %d rows of %d, cursor on %q", len(m.rows(0)), all, at(m).Key)
	}
	// left on the closed heading steps out to its source's, which a second left closes and a third steps out of, to the
	// heading over the groups
	if m = drive(m, left); grid.Level(at(m)) != 2 || !m.stages[0].open.Open(at(m)) {
		t.Fatalf("left on a closed heading did not step out to the one over it: %q", at(m).Key)
	}
	if m = drive(m, left, left); m.stages[0].cursor != 0 {
		t.Fatalf("left on a closed heading: cursor %d", m.stages[0].cursor)
	}
	if m = drive(m, left, left); m.stages[0].cursor != 0 || len(m.rows(0)) != 1 {
		t.Fatalf("left on the outer heading, then again: cursor %d, %d rows", m.stages[0].cursor, len(m.rows(0)))
	}
	if m = drive(m, right, right); m.stages[0].cursor != 1 {
		t.Fatalf("right twice on the outer heading: cursor %d", m.stages[0].cursor)
	}
	// option-left on the outer heading closes everything under it; option-right opens everything, rows too
	m = drive(m, up, tea.KeyPressMsg{Code: tea.KeyLeft, Mod: tea.ModAlt})
	if len(m.rows(0)) != 1 || m.stages[0].open[m.every(0)[1].Key] {
		t.Fatalf("option-left: %d rows, the inner heading open %v", len(m.rows(0)), m.stages[0].open[m.every(0)[1].Key])
	}
	if last := lines(m)[len(lines(m))-1]; !strings.Contains(last, "→ expand") || !strings.Contains(last, "⌥←→ all under it") {
		t.Errorf("the keys on a closed heading: %q", last)
	}
	if m = drive(m, tea.KeyPressMsg{Code: tea.KeyRight, Mod: tea.ModAlt}); len(m.rows(0)) != all || openRows(m) == 0 {
		t.Fatalf("option-right: %d rows of %d, %d rows open", len(m.rows(0)), all, openRows(m))
	}
	// a row that hides nothing under no heading has nowhere to step out to
	if m = drive(m, down, left, left); m.stages[0].cursor != 0 {
		t.Errorf("left out of an inner row twice: cursor %d", m.stages[0].cursor)
	}
	// an option-click on a heading's ▾ closes everything under it, and another opens it all again; elsewhere on the
	// heading over the groups, off its names, a click folds nothing
	names, before := m.headingNames(m.rows(0)[0]), len(m.rows(0))
	if m = drive(m, testkit.Click(names[len(names)-1].until+1, m.rowsTop())); len(m.rows(0)) != before || m.colMenu.Open {
		t.Fatalf("a click on the heading over the groups, off its ▾ and names: %d rows of %d, menu %v",
			len(m.rows(0)), before, m.colMenu.Open)
	}
	alt := testkit.Click(1, m.rowsTop())
	alt.Mod = tea.ModAlt
	if m = drive(m, alt); len(m.rows(0)) != 1 || m.stages[0].open[m.every(0)[1].Key] {
		t.Fatalf("an option-click on the outer heading: %d rows", len(m.rows(0)))
	}
	if m = drive(m, alt); len(m.rows(0)) != all {
		t.Errorf("a second option-click: %d rows of %d", len(m.rows(0)), all)
	}
}

// The tab's menu's fold items stand always, named for what they act on. With nothing selected, Collapse All folds every
// group to the outermost heading, the cursor on what shows of its row, Expand All opens every group and every row that
// hides something, and Collapse Others is dim. With rows selected, Expand Selected opens them and the groups they stand
// in, and Collapse Others folds every group without a selected row and closes every row not selected.
func TestTheTabsMenuFoldsTheGroupsAndTheSelection(t *testing.T) {
	m := nested(t)
	all := len(m.rows(0))
	choose := func(m Model, name string) Model {
		m = drive(m, press(':'))
		if m.tabMenu.At = item(m, name); m.tabMenu.At < 0 || !m.enabled(m.tabMenu.At) {
			t.Fatalf("%s is not in the menu, or dim: %d", name, m.tabMenu.At)
		}
		return drive(m, press(tea.KeyEnter))
	}
	if k := item(m, "Collapse Others"); k < 0 || m.enabled(k) {
		t.Errorf("collapse others with nothing selected: at %d, enabled %v", k, k >= 0 && m.enabled(k))
	}
	m = drive(m, down)
	if m = choose(m, "Collapse All"); len(m.rows(0)) != 1 || m.stages[0].cursor != 0 {
		t.Fatalf("collapse all: %d rows, cursor %d", len(m.rows(0)), m.stages[0].cursor)
	}
	if m = choose(m, "Expand All"); len(m.rows(0)) != all || openRows(m) == 0 {
		t.Fatalf("expand all: %d rows of %d, %d open", len(m.rows(0)), all, openRows(m))
	}
	// #46, selected while its group is folded away, shows and opens with Expand Selected; the other groups stay folded
	m = choose(m, "Collapse All")
	m.stages[0].picked["#46"] = true
	if m = drive(m, press(':')); m.tabItems()[4].Name != "Expand Selected" || !m.enabled(item(m, "Collapse Others")) {
		t.Fatalf("the items with a row selected: %+v", m.tabItems()[4:7])
	}
	m = drive(m, esc)
	keys := func(m Model) []string {
		var out []string
		for _, r := range m.rows(0) {
			if !r.Aside {
				out = append(out, r.Key)
			}
		}
		return out
	}
	if m = choose(m, "Expand Selected"); !slices.Equal(keys(m), []string{"#46"}) {
		t.Fatalf("expand selected shows %q", keys(m))
	}
	// collapse others, from everything open, leaves #46's group alone open
	clear(m.stages[0].picked)
	m = choose(m, "Expand All")
	m.stages[0].picked["#46"] = true
	if m = choose(m, "Collapse Others"); !slices.Equal(keys(m), []string{"#46"}) || openRows(m) > 1 {
		t.Errorf("collapse others shows %q, %d open", keys(m), openRows(m))
	}
}

// Where the cursor has no row under it, as when a search leaves none, the outline's keys and space do nothing; space
// on an aside that heads no group, as incoming's remotes, selects nothing; and with no stage, nothing is selected.
func TestTheOutlinesKeysWithNothingToActOn(t *testing.T) {
	m := nested(t)
	p := m.panel(0)
	p.Query = "matches no row at all"
	m.setPanel(p)
	m.inRows = true
	if m = drive(m, right, left, press(tea.KeySpace)); len(m.rows(0)) != 0 || m.selected() != 0 {
		t.Errorf("keys over no rows: %d rows, %d selected", len(m.rows(0)), m.selected())
	}
	m = loaded(t, options(), slices.Repeat([]tea.Msg{right}, 6)...)
	if m.stages[m.stage].name != "incoming" || !m.rows(m.stage)[0].Aside || grid.Heads(m.rows(m.stage)[0]) {
		t.Fatalf("incoming does not start with a remote's aside")
	}
	m.inRows = true
	if m = drive(m, press(tea.KeySpace)); m.selected() != 0 {
		t.Errorf("space on a remote's aside selected %d", m.selected())
	}
	none := New(runner(t), Options{})
	if none.selectAll(); none.selected() != 0 {
		t.Errorf("with no stage, %d selected", none.selected())
	}
}

// Grouped by labels, a field of several values, an issue stands under each of its labels, the same row under every
// heading: selecting it under one selects it under all, as the selection is the row's, and the tab counts it once.
// Opening it, the cursor stays under the heading it was opened under.
func TestARowGroupedUnderSeveralValuesIsOneRow(t *testing.T) {
	m := loaded(t, options(), tea.WindowSizeMsg{Width: 120, Height: 40})
	count := m.count(0)
	m.setPanel(regroup(m.panel(0), "labels"))
	rows := m.rows(0)
	var at []int
	for k, r := range rows {
		if r.Key == "#45" {
			at = append(at, k)
		}
	}
	if len(at) < 2 || m.count(0) != count {
		t.Fatalf("#45 stands under %d headings, the count %s, was %s:\n%s", len(at), m.count(0), count, view(m))
	}
	// selected under its last label, it is selected under every one, each heading over it marked
	m.stages[0].cursor, m.inRows = at[len(at)-1], true
	m.selectRow(false)
	marks := m.groupMarks(0)
	for _, k := range at {
		if p := grid.Parent(rows, k); marks[rows[p].Key] == "" {
			t.Errorf("the heading over #45 at %d is not marked", k)
		}
	}
	if m.selected() != 1 {
		t.Errorf("%d selected, want one row", m.selected())
	}
	// opened under its last label, the cursor stays there, not on the first
	m.stages[0].cursor = at[len(at)-1]
	if m = drive(m, right); m.stages[0].cursor != at[len(at)-1] || !m.stages[0].open["#45"] {
		t.Errorf("opened under its last label: cursor %d, want %d", m.stages[0].cursor, at[len(at)-1])
	}
}
