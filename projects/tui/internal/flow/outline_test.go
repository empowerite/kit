// The outline: a group's heading, a row, and the header open and close alike, by the keys, a click, and the tab's
// menu, whose fold items act on everything or on the selection.
package flow

import (
	"maps"
	"slices"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

	tea "charm.land/bubbletea/v2"

	"github.com/empowerite/kit/projects/tui/internal/grid"
	"github.com/empowerite/kit/projects/tui/internal/table"
	"github.com/empowerite/kit/projects/tui/internal/testkit"
)

// nested gives backlog grouped by hand, by source, then by assignees within it, at 120 columns: the heading over the
// groups, on the control line, a heading per source, a heading per assignee under each, and their rows.
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
func at(m Model) row { return m.rows(m.stage)[m.stages[m.stage].Cursor] }

// Right opens the node under the cursor, or on a heading already open steps into its first row; left closes it, or on
// a node closed, or one that hides nothing, steps out to the heading over it, as a tree does. With option, either
// opens or closes everything under the node too, as an option-click on a heading does.
func TestTheOutlineOpensClosesAndStepsAlongItsTree(t *testing.T) {
	m := nested(t)
	all := len(m.rows(0))
	// down lands past the headings on a row; left steps out to its heading, and again closes that heading
	m = intoRows(m)
	if grid.Heads(at(m)) {
		t.Fatalf("down landed on a heading")
	}
	if m = drive(m, left); grid.Level(at(m)) != 3 {
		t.Fatalf("left on a row closed did not step out to its heading: %q", at(m).Key)
	}
	inner := at(m).Key
	if m = drive(m, left); len(m.rows(0)) >= all || at(m).Key != inner || m.stages[0].Open.Open(at(m)) {
		t.Fatalf("left on an open heading: %d rows of %d, cursor on %q", len(m.rows(0)), all, at(m).Key)
	}
	// left on the closed heading steps out to its source's, which a second left closes; a third has nowhere to step
	// out to, the heading over the groups standing on the control line, which the cursor never reaches
	if m = drive(m, left); grid.Level(at(m)) != 2 || !m.stages[0].Open.Open(at(m)) {
		t.Fatalf("left on a closed heading did not step out to the one over it: %q", at(m).Key)
	}
	if m = drive(m, left, left); m.stages[0].Cursor != 1 || len(m.rows(0)) != 2 {
		t.Fatalf("left on the outermost heading closed: cursor %d, %d rows", m.stages[0].Cursor, len(m.rows(0)))
	}
	if m = drive(m, right, right); m.stages[0].Cursor != 2 {
		t.Fatalf("right twice on the outer heading: cursor %d", m.stages[0].Cursor)
	}
	// from every heading open, option-left on the outermost heading closes the deepest open level under it, a level
	// each press, and option-right opens one level more, every branch alike
	m.stages[0].Open, m.stages[0].Cursor = grid.Outline{}, 1
	m.rebuild()
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
	optioned := func(code rune) tea.KeyPressMsg { return tea.KeyPressMsg{Code: code, Mod: tea.ModAlt} }
	s0, a0, r0 := levels()
	if m = drive(m, optioned(tea.KeyLeft)); func() bool { s, a, r := levels(); return s != s0 || a != a0 || r != 0 }() {
		s, a, r := levels()
		t.Fatalf("option-left from every heading open: %d %d %d, were %d %d %d", s, a, r, s0, a0, r0)
	}
	if m = drive(m, optioned(tea.KeyLeft)); func() bool { s, a, r := levels(); return s != s0 || a != 0 || r != 0 }() {
		s, a, r := levels()
		t.Fatalf("option-left again: %d sources, %d assignees, %d rows", s, a, r)
	}
	if m = drive(m, optioned(tea.KeyLeft)); len(m.rows(0)) != 1+s0 {
		t.Errorf("option-left a third time left the heading open: %d rows", len(m.rows(0)))
	}
	// ⌥ with an arrow is named only where the terminal said it tells it apart, as the kitty keyboard protocol does
	if last := lines(m)[len(lines(m))-1]; strings.Contains(last, "← hide") || strings.Contains(last, "⌥←→") {
		t.Errorf("the keys on the heading over the groups, the terminal silent: %q", last)
	}
	m = drive(m, tea.KeyboardEnhancementsMsg{Flags: ansi.KittyDisambiguateEscapeCodes})
	if last := lines(m)[len(lines(m))-1]; strings.Contains(last, "← hide") || !strings.Contains(last, "⌥←→ a level") {
		t.Errorf("the keys on the heading over the groups, the protocol answered: %q", last)
	}
	if m = drive(m, optioned(tea.KeyRight)); func() bool { s, a, r := levels(); return s != s0 || a != a0 || r != 0 }() {
		s, a, r := levels()
		t.Fatalf("option-right: %d sources, %d assignees, %d rows", s, a, r)
	}
	if m = drive(m, optioned(tea.KeyRight)); len(m.rows(0)) != all || openRows(m) != 0 {
		t.Fatalf("option-right again: %d rows of %d, %d open", len(m.rows(0)), all, openRows(m))
	}
	if m = drive(m, optioned(tea.KeyRight)); openRows(m) == 0 {
		t.Fatalf("option-right a third time opened no row")
	}
	// left closes the outermost heading, and again has nowhere to step out to
	if m = drive(m, left, left); m.stages[0].Cursor != 1 || len(m.rows(0)) != 2 {
		t.Errorf("left on the outermost heading twice: cursor %d, %d rows", m.stages[0].Cursor, len(m.rows(0)))
	}
	// a click on the heading over the groups, off its folds and names, folds nothing
	names, before := table.BarNames(&m), len(m.rows(0))
	if m = drive(m, testkit.Click(names[len(names)-1].Until+3, m.top())); len(m.rows(0)) != before || m.colMenu.Open {
		t.Fatalf("a click on the heading over the groups, off its ▾ and names: %d rows of %d, menu %v",
			len(m.rows(0)), before, m.colMenu.Open)
	}
	alt := testkit.Click(table.FoldAt(m.rows(0)[1]), rowY(m, m.rows(0)[1].Key))
	alt.Mod = tea.ModAlt
	// source's heading closed, as left left it, everything under it open: an option-click on its ▸ opens it, a level,
	// so all shows again; each option-click on its ▾ then closes the deepest open level under it, the rows first, then
	// the assignees' headings, then the heading itself
	if m = drive(m, alt); len(m.rows(0)) != all || openRows(m) == 0 {
		t.Fatalf("an option-click on source's ▸: %d rows of %d, %d open", len(m.rows(0)), all, openRows(m))
	}
	if m = drive(m, alt); len(m.rows(0)) != all || openRows(m) != 0 {
		t.Fatalf("an option-click on its ▾: %d rows of %d, %d open", len(m.rows(0)), all, openRows(m))
	}
	if m = drive(m, alt); len(m.rows(0)) >= all || !m.stages[0].Open.Open(m.every(0)[1]) {
		t.Fatalf("a second option-click: %d rows of %d", len(m.rows(0)), all)
	}
	if m = drive(m, alt); len(m.rows(0)) != 2 || m.stages[0].Open.Open(m.every(0)[1]) {
		t.Errorf("a third option-click: %d rows", len(m.rows(0)))
	}
	// an option-click on a level's fold in the heading over the groups does as a click there: source's closes its level,
	// nothing under the heading showing
	level := testkit.Click(table.BarNames(&m)[0].From-2, m.top())
	level.Mod = tea.ModAlt
	if m = drive(m, level); len(m.rows(0)) != 1 {
		t.Errorf("an option-click on source's fold: %d rows", len(m.rows(0)))
	}
}

// The tab's menu's fold items stand always. > and <, as Show More and Show Less, show and hide a level at a time,
// Show Less down to the outermost groups' headings, under the heading over them, which never closes, and Show More
// up to every group and every row that hides something. Show Selected and Hide Selected are dim with nothing
// selected; with #46 selected while its group is folded away, Show Selected shows it and the groups over it, the
// other groups folded still, and Hide Selected hides its lines, and nothing over it.
func TestTheTabsMenuFoldsTheGroupsAndTheSelection(t *testing.T) {
	m := nested(t)
	all := len(m.rows(0))
	for _, name := range []string{"Show Selected", "Hide Selected"} {
		if k := item(m, name); k < 0 || m.enabled(k) {
			t.Errorf("%s with nothing selected: at %d, enabled %v", name, k, k >= 0 && m.enabled(k))
		}
	}
	m = drive(m, down)
	// each press a level, until there is none
	until := func(m Model, key rune) Model {
		for range 10 {
			was := maps.Clone(m.stages[0].Open)
			if m = drive(m, press(key)); maps.Equal(was, m.stages[0].Open) {
				return m
			}
		}
		t.Fatalf("%c never stopped", key)
		return m
	}
	if m = until(m, '<'); len(m.rows(0)) != 2 || m.stages[0].Cursor != 1 {
		t.Fatalf("show less, all the way: %d rows, cursor %d", len(m.rows(0)), m.stages[0].Cursor)
	}
	if m = until(m, '>'); len(m.rows(0)) != all || openRows(m) == 0 {
		t.Fatalf("show more, all the way: %d rows of %d, %d open", len(m.rows(0)), all, openRows(m))
	}
	m = until(m, '<')
	m.stages[0].Picked["#46"] = true
	keys := func(m Model) []string {
		var out []string
		for _, r := range m.rows(0) {
			if !r.Aside {
				out = append(out, r.Key)
			}
		}
		return out
	}
	if m = choose(t, m, "Show Selected"); !slices.Equal(keys(m), []string{"#46"}) {
		t.Fatalf("show selected shows %q", keys(m))
	}
	if m = choose(t, m, "Hide Selected"); !slices.Equal(keys(m), []string{"#46"}) || m.stages[0].Open["#46"] {
		t.Errorf("hide selected shows %q, #46 open %v", keys(m), m.stages[0].Open["#46"])
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
	if m = drive(m, right, left, tea.KeyPressMsg{Code: tea.KeyRight, Mod: tea.ModAlt},
		tea.KeyPressMsg{Code: tea.KeyLeft, Mod: tea.ModAlt}, press(tea.KeySpace)); len(m.rows(0)) != 0 || m.selected() != 0 {
		t.Errorf("keys over no rows: %d rows, %d selected", len(m.rows(0)), m.selected())
	}
	// over a group, a member whose read failed is its line, an aside, which space does not select
	m = drive(over(t, "changes"), answer("changes", `{"rows": [],
		"errors": [{"member": "github.com/o/b", "error": "gh cannot read o/b"}]}`))
	rows := m.rows(m.stage)
	failed := slices.IndexFunc(rows, func(r grid.Row) bool { return r.Aside && !grid.Heads(r) && !grid.OverGroups(r) })
	if failed < 0 {
		t.Fatalf("no failed member's line:\n%s", view(m))
	}
	m.inRows, m.stages[m.stage].Cursor = true, failed
	if m = drive(m, press(tea.KeySpace)); m.selected() != 0 {
		t.Errorf("space on a failed member's line selected %d", m.selected())
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
	m.stages[0].Cursor, m.inRows = at[len(at)-1], true
	m.selectRow(false)
	marks := table.GroupMarks(m.every(0), m.stages[0].Picked)
	for _, k := range at {
		if p := grid.Parent(rows, k); marks[rows[p].Key] == "" {
			t.Errorf("the heading over #45 at %d is not marked", k)
		}
	}
	if m.selected() != 1 {
		t.Errorf("%d selected, want one row", m.selected())
	}
	// opened under its last label, the cursor stays there, not on the first
	m.stages[0].Cursor = at[len(at)-1]
	if m = drive(m, right); m.stages[0].Cursor != at[len(at)-1] || !m.stages[0].Open["#45"] {
		t.Errorf("opened under its last label: cursor %d, want %d", m.stages[0].Cursor, at[len(at)-1])
	}
}

// [ and ] move the cursor to the heading of the group before and after it, { and } to the first and the last, as the
// palette's keys move between its groups; the key hints name [] group while the rows group.
func TestBracketsMoveBetweenTheGroupsHeadings(t *testing.T) {
	m := intoRows(nested(t))
	var heads []int
	for i, r := range m.rows(0) {
		if i >= table.First(m.rows(0)) && grid.Heads(r) {
			heads = append(heads, i)
		}
	}
	if len(heads) < 2 {
		t.Fatalf("backlog grouped two deep has %d headings to move between", len(heads))
	}
	cursor := func() int { return m.stages[m.stage].Cursor }
	if m = drive(m, testkit.Typed("}")...); cursor() != heads[len(heads)-1] {
		t.Errorf("}: cursor %d, want the last heading, %d", cursor(), heads[len(heads)-1])
	}
	if m = drive(m, testkit.Typed("[")...); cursor() != heads[len(heads)-2] {
		t.Errorf("[: cursor %d, want %d", cursor(), heads[len(heads)-2])
	}
	if m = drive(m, testkit.Typed("{")...); cursor() != heads[0] {
		t.Errorf("{: cursor %d, want the first heading, %d", cursor(), heads[0])
	}
	if m = drive(m, testkit.Typed("]")...); cursor() != heads[1] {
		t.Errorf("]: cursor %d, want %d", cursor(), heads[1])
	}
	if last := lines(m)[len(lines(m))-1]; !strings.Contains(last, "[] group") {
		t.Errorf("the key hints in grouped rows: %q", last)
	}
}
