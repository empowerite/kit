package flow

import (
	"slices"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/empowerite/kit/projects/tui/internal/grid"
	"github.com/empowerite/kit/projects/tui/internal/shell"
	"github.com/empowerite/kit/projects/tui/internal/table"
	"github.com/empowerite/kit/projects/tui/internal/testkit"
)

// A right click on a column's name drops the column menu down under it, as a terminal that keeps ctrl and option with
// a click for itself still sends the right button, titled by the name alone: its sort, a radio, and Clear Sort; for
// a list, flattened here so it sorts, its rows gathered again; the rows grouped by it, innermost, or no longer;
// table.Hide; and its options. A line the column cannot act on stays, dim.
func TestARightClickOnAColumnsNameDropsItsMenu(t *testing.T) {
	m := loaded(t, viewing(""), tea.WindowSizeMsg{Width: 140, Height: 40})
	m.setPanel(m.panel(m.stage).Flattened("assignees", true))
	at := func(name string) int { return strings.Index(lines(m)[m.headerTop()], name) }
	rightClick := func(name string) tea.MouseClickMsg {
		return tea.MouseClickMsg{X: at(name), Y: m.headerTop(), Button: tea.MouseRight}
	}
	group := func() []string { return m.panel(m.stage).Groups() }
	sorts := func() int { return len(m.panel(m.stage).SortKeys()) }
	if m = drive(m, rightClick("assignee")); !m.colMenu.Open || m.colMenu.Column != "assignees" ||
		m.colMenu.At != m.colMenu.LineOf(&m, table.ActUp) {
		t.Fatalf("a right click on assignee: %+v\n%s", m.colMenu, raw(m))
	}
	var names, glyphs []string
	for _, it := range m.colMenu.Items(&m) {
		names, glyphs = append(names, it.Name), append(glyphs, it.Glyph)
	}
	want := []string{"▲ Ascending", "▼ Descending", table.Unsorted, "", table.GroupIt, table.UngroupIt, "",
		table.FlattenIt("assignee"), table.GatherIt("assignees"), "", table.Hide, table.More}
	if !slices.Equal(names, want) || !strings.Contains(raw(m), table.GroupGlyph+"  "+table.GroupIt) ||
		slices.Contains(names, "assignee") || !strings.Contains(keysOf(m), "esc close") {
		t.Fatalf("the menu: %q, %q", names, keysOf(m))
	}
	if !slices.Equal(glyphs[3:], []string{"", table.GroupGlyph, table.UngroupGlyph, "", table.FlattenGlyph,
		table.GatherGlyph, "", table.HideGlyph, table.MoreGlyph}) {
		t.Errorf("the glyphs: %q", glyphs)
	}
	// the radio has Unsorted picked, and Ungroup is dim on a column not grouping; enter on ▲ Ascending sorts by it
	if !strings.Contains(m.colMenu.Items(&m)[m.colMenu.LineOf(&m, table.ActUnsorted)].Glyph, grid.Picked) ||
		m.colMenu.Enabled(&m, m.colMenu.LineOf(&m, table.ActUngroup)) || m.colMenu.Enabled(&m, -1) {
		t.Errorf("Unsorted is not picked, or Ungroup acts")
	}
	if m = drive(m, press(tea.KeyEnter)); m.colMenu.Open || sorts() != 1 || m.panel(m.stage).Mark("assignees") != "▲" {
		t.Fatalf("Ascending: %+v", m.panel(m.stage).SortKeys())
	}
	// another column's Ascending, chosen by a click, comes after it, its triangles ₙ
	m = drive(m, rightClick("age"))
	if m.colMenu.Items(&m)[0].Name != "▲ⁿ Ascending" {
		t.Errorf("age, with assignees sorting: %q", m.colMenu.Items(&m)[0].Name)
	}
	box, top, left := m.colMenu.Box(&m)
	if m = drive(m, testkit.Click(left+3, top+1+m.colMenu.LineOf(&m, table.ActUp))); m.colMenu.Open || sorts() != 2 ||
		m.panel(m.stage).SortKeys()[1].Column != "age" {
		t.Fatalf("Ascending on age: %+v\n%v", m.panel(m.stage).SortKeys(), box)
	}
	// Group groups within the groups, innermost, and Ungroup takes it out wherever it stands; a letter moves to the
	// next line it starts
	m.setPanel(regroup(m.panel(m.stage), "source"))
	// Group stands right after the sort, before the flattening
	m = drive(m, rightClick("assignee"), down, down, down)
	if m.colMenu.At != m.colMenu.LineOf(&m, table.ActGroup) {
		t.Fatalf("down to Group: %d", m.colMenu.At)
	}
	if m = drive(m, press(tea.KeySpace)); !slices.Equal(group(), []string{"source", "assignees"}) {
		t.Fatalf("Group assignees: %q", group())
	}
	m.colMenu = table.ColumnMenu{Dropdown: shell.Dropdown{Open: true}, Column: "source"}
	if m.colMenu.Enabled(&m, m.colMenu.LineOf(&m, table.ActGroup)) || !m.colMenu.Enabled(&m, m.colMenu.LineOf(&m, table.ActUngroup)) {
		t.Errorf("source, grouping: group %v", m.colMenu.Enabled(&m, m.colMenu.LineOf(&m, table.ActGroup)))
	}
	if m.columnAct(m.colMenu.LineOf(&m, table.ActUngroup)); !slices.Equal(group(), []string{"assignees"}) {
		t.Fatalf("Ungroup source: %q", group())
	}
	m = drive(m, rightClick("title"))
	if m = drive(m, typed("u")...); m.colMenu.At == m.colMenu.LineOf(&m, table.ActUngroup) {
		t.Errorf("u moved to Ungroup, which title cannot do")
	}
	// title sorts and does not group: Group is dim, and does nothing chosen; Unsorted takes title out of the keys
	if m.colMenu.Enabled(&m, m.colMenu.LineOf(&m, table.ActGroup)) || !m.colMenu.Enabled(&m, m.colMenu.LineOf(&m, table.ActUp)) ||
		!m.colMenu.Enabled(&m, m.colMenu.LineOf(&m, table.ActHide)) ||
		!m.colMenu.Enabled(&m, m.colMenu.LineOf(&m, table.ActUnsorted)) {
		t.Errorf("title's lines")
	}
	if m.columnAct(m.colMenu.LineOf(&m, table.ActGroup)); len(group()) != 1 || m.colMenu.Open {
		t.Errorf("Group title: %q", group())
	}
	m = drive(m, rightClick("title"))
	m.columnAct(m.colMenu.LineOf(&m, table.ActUp))
	m = drive(m, rightClick("title"))
	if m.columnAct(m.colMenu.LineOf(&m, table.ActUnsorted)); slices.ContainsFunc(m.panel(m.stage).SortKeys(),
		func(k grid.SortKey) bool { return k.Column == "title" }) || len(m.panel(m.stage).SortKeys()) == 0 {
		t.Errorf("Unsorted on title: %+v", m.panel(m.stage).SortKeys())
	}
	// Options… opens the field's popover; esc closes the menu; a click off it closes it too
	m = drive(m, append([]tea.Msg{rightClick("title")}, typed("o")...)...)
	if m = drive(m, press(tea.KeyEnter)); !m.pop.Open || m.pop.Field != "title" {
		t.Fatalf("Options…: %+v", m.pop)
	}
	m = drive(m, esc, rightClick("age"), esc)
	if m.colMenu.Open || m.pop.Open {
		t.Errorf("esc: menu %v, popover %v", m.colMenu.Open, m.pop.Open)
	}
	if m = drive(m, rightClick("age"), testkit.Click(1, 30)); m.colMenu.Open {
		t.Errorf("a click off the menu: open %v", m.colMenu.Open)
	}
	// table.Hide takes the column out of the header and the rows
	m = drive(m, rightClick("age"))
	if m.columnAct(m.colMenu.LineOf(&m, table.ActHide)); slices.Contains(m.shownColumns(m.stage), "age") || m.colMenu.Open {
		t.Errorf("Hide age: %q", m.shownColumns(m.stage))
	}
	// a right click on the header's gutter, off every name, opens nothing
	if m = drive(m, tea.MouseClickMsg{X: 1, Y: m.headerTop(), Button: tea.MouseRight}); m.colMenu.Open {
		t.Errorf("a right click off the names opened the menu")
	}
	// nor does one under the rows, or a click on the rule of a table with no preset line
	if m = drive(m, tea.MouseClickMsg{X: 10, Y: 30, Button: tea.MouseRight}); m.colMenu.Open {
		t.Errorf("a right click under the rows opened the menu")
	}
	m.noPresets = true
	if m = drive(m, testkit.Click(10, ruleY(m))); m.colMenu.Open || m.pop.Open {
		t.Errorf("a click on the rule opened a menu")
	}
}

// Sorting always adds to the keys: the menu's Ascending or Descending on a column not sorting puts it after the others,
// and on one sorting turns it in its place; None takes it out. The groups order the rows before any key, the outermost
// first, those that sort, so the rule numbers the keys after them; a group that does not sort, as a group does not by
// default, takes no number. Each mark stands on the rule under its column's name, so the names stand where they stood.
// A mark stands under its name's first cell, as a name over numbers does too. A plain click on a column's ▲ turns it
// to ▼, and one on ▼ stops the sort; the menu stays shut.
func TestTheSortMarkTurnsAndTheMenuSorts(t *testing.T) {
	m := loaded(t, viewing(""), tea.WindowSizeMsg{Width: 140, Height: 40})
	header := func() string { return lines(m)[m.headerTop()] }
	at := func(text string) int { return ansi.StringWidth(header()[:strings.Index(header(), text)]) }
	rule := func() string { return lines(m)[ruleY(m)] }
	onRule := func(text string) int { return ansi.StringWidth(rule()[:strings.Index(rule(), text)]) }
	unsorted := header()
	sorts := func() []grid.SortKey { return m.panel(m.stage).SortKeys() }
	menu := func(name string, act table.ColAct) {
		m = drive(m, testkit.Click(at(name), m.headerTop()))
		m.columnAct(m.colMenu.LineOf(&m, act))
	}
	if m.colMenu.Column = "age"; m.colMenu.Items(&m)[0].Name != "▲ Ascending" {
		t.Errorf("age, nothing sorting: %q", m.colMenu.Items(&m)[0].Name)
	}
	m.colMenu = table.ColumnMenu{}
	menu("age", table.ActUp)
	menu("id", table.ActUp)
	if m.colMenu.Column = "id"; m.colMenu.Items(&m)[0].Name != "▲ⁿ Ascending" || !strings.Contains(m.colMenu.Items(&m)[0].Glyph, grid.Picked) {
		t.Errorf("id, the second key: %+v", m.colMenu.Items(&m)[:3])
	}
	m.colMenu = table.ColumnMenu{}
	if !slices.Equal(sorts(), []grid.SortKey{{Column: "age"}, {Column: "id"}}) || !strings.Contains(rule(), "▲¹") ||
		onRule("▲¹") != at("age") || onRule("▲²") != at("id") || header() != unsorted {
		t.Fatalf("then by age, then by id: %+v\n%s\n%s", sorts(), header(), rule())
	}
	// the ▲ of age turns, and age keeps its place among the keys; the menu stays shut
	if m = drive(m, testkit.Click(onRule("▲¹"), ruleY(m))); m.colMenu.Open ||
		!slices.Equal(sorts(), []grid.SortKey{{Column: "age", Desc: true}, {Column: "id"}}) {
		t.Fatalf("a click on age's ▲: %+v, menu %v\n%s", sorts(), m.colMenu.Open, header())
	}
	// age is not the last key: a click on its ▼, its ₁ too, turns it ascending again, in its place
	if m = drive(m, testkit.Click(onRule("▼₁")+1, ruleY(m))); !slices.Equal(sorts(), []grid.SortKey{{Column: "age"},
		{Column: "id"}}) {
		t.Fatalf("a click on age's ₁: %+v\n%s", sorts(), rule())
	}
	// a shift-click on age's ▲ stops age's sort, and id is the one key, with no number
	shift := testkit.Click(onRule("▲¹"), ruleY(m))
	shift.Mod = tea.ModShift
	if m = drive(m, shift); !slices.Equal(sorts(), []grid.SortKey{{Column: "id"}}) {
		t.Fatalf("a shift-click on age's ▲: %+v\n%s", sorts(), rule())
	}
	// id's ▲ turns to ▼, then off, a plain click each; then age sorts again, the one key
	for _, want := range [][]grid.SortKey{{{Column: "id", Desc: true}}, nil} {
		mark := "▲"
		if len(sorts()) == 1 && sorts()[0].Desc {
			mark = "▼"
		}
		if m = drive(m, testkit.Click(onRule(mark), ruleY(m))); !slices.Equal(sorts(), want) || m.colMenu.Open {
			t.Fatalf("a click on id's %s: %+v\n%s", mark, sorts(), rule())
		}
	}
	if m = drive(m, testkit.Click(at("age"), ruleY(m))); !slices.Equal(sorts(), []grid.SortKey{{Column: "age"}}) {
		t.Fatalf("a click under age: %+v\n%s", sorts(), rule())
	}
	// ▼ on id, not a key, comes after age; Descending is picked; ▲ on age turns it in its place
	if menu("id", table.ActDown); !slices.Equal(sorts(), []grid.SortKey{{Column: "age"}, {Column: "id", Desc: true}}) {
		t.Fatalf("▼ on id: %+v", sorts())
	}
	if m.colMenu.Column = "id"; !strings.Contains(m.colMenu.Items(&m)[1].Glyph, grid.Picked) {
		t.Errorf("▼ is not picked: %+v", m.colMenu.Items(&m)[1])
	}
	m.colMenu = table.ColumnMenu{}
	if menu("age", table.ActDown); !slices.Equal(sorts(), []grid.SortKey{{Column: "age", Desc: true}, {Column: "id", Desc: true}}) {
		t.Fatalf("▼ on age: %+v", sorts())
	}
	if menu("id", table.ActUnsorted); !slices.Equal(sorts(), []grid.SortKey{{Column: "age", Desc: true}}) {
		t.Errorf("Unsorted on id: %+v", sorts())
	}
	// grouped by source, which does not sort, age, the one key, has no number; once source sorts, it is 1, and age ▼₂
	m.setPanel(regroup(m.panel(m.stage), "source"))
	if !strings.Contains(rule(), "▼") || strings.Contains(rule(), "▼₁") || strings.Contains(rule(), "▼₂") {
		t.Errorf("age under a group that does not sort:\n%s", rule())
	}
	m.setPanel(m.panel(m.stage).SortedAs("source", grid.Descending))
	if !strings.Contains(rule(), "▼₂") || onRule("▼₂") != at("age") {
		t.Errorf("age under a group that sorts:\n%s\n%s", header(), rule())
	}
	if m.colMenu.Column = "id"; m.colMenu.Items(&m)[0].Name != "▲ⁿ Ascending" {
		t.Errorf("id under a group: %q", m.colMenu.Items(&m)[0].Name)
	}
	m.colMenu = table.ColumnMenu{}
}

// A click on the rule under a column's name sorts by it, ascending, after the other keys, where it does not sort yet,
// turns it descending where it sorts ascending, and stops it where it sorts descending, a step each click; the names stand where they stood, as the marks do not widen
// them. The cells after a name on the names' line sort nothing, and the rule off every name sorts nothing.
func TestAClickOnTheRuleUnderAColumnsNameSortsByIt(t *testing.T) {
	m := loaded(t, options(), tea.WindowSizeMsg{Width: 120, Height: 30})
	// under gives the screen column of the first cell of a column's name, which the rule's target starts under
	under := func(m Model, column string) int {
		x, _ := m.columnName(column)
		return x
	}
	after := func(m Model, column string) int {
		x, name := m.columnName(column)
		return x + ansi.StringWidth(name)
	}
	keys := func(m Model) int { return len(m.panel(m.stage).SortKeys()) }
	mark := func(m Model, column string) string { return m.panel(m.stage).Mark(column) }
	header := lines(m)[m.headerTop()]
	if m = drive(m, testkit.Click(after(m, "age"), m.headerTop())); keys(m) != 0 || m.colMenu.Open {
		t.Fatalf("a click right after age's name: %d keys, menu %v", keys(m), m.colMenu.Open)
	}
	if m = drive(m, testkit.Click(1, ruleY(m))); keys(m) != 0 {
		t.Fatalf("a click on the rule's gutter: %d keys", keys(m))
	}
	if m = drive(m, testkit.Click(under(m, "age"), ruleY(m))); keys(m) != 1 || mark(m, "age") != "▲" || m.colMenu.Open {
		t.Fatalf("a click under age: %d keys, age %q, menu %v", keys(m), mark(m, "age"), m.colMenu.Open)
	}
	// again turns it descending, again off, and again ascending, its last cell as its first
	for _, want := range []string{"▼", "", "▲"} {
		if m = drive(m, testkit.Click(after(m, "age")-1, ruleY(m))); keys(m) != len([]rune(want)) || mark(m, "age") != want {
			t.Fatalf("another click under age: %d keys, age %q, want %q", keys(m), mark(m, "age"), want)
		}
	}
	if m = drive(m, testkit.Click(under(m, "change"), ruleY(m))); keys(m) != 2 || mark(m, "change") != "▲²" {
		t.Errorf("a click under change: %d keys, change %q", keys(m), mark(m, "change"))
	}
	// a list sorts by size while a row holds all its values: the rule has the dim flatten mark under its name's first
	// cell, and the sort's after it, which a click sorts by how many values each row holds
	marks := func(m Model, column string, cells int) string {
		return ansi.Strip(ansi.Cut(m.rule(), under(m, column), under(m, column)+cells))
	}
	if got := marks(m, "assignees", 2); got != table.FlattenGlyph+Sortable {
		t.Errorf("the rule under assignees: %q", got)
	}
	if m = drive(m, testkit.Click(under(m, "assignees")+1, ruleY(m))); keys(m) != 3 || mark(m, "assignees") != "▲³" ||
		m.panel(m.stage).Field("assignees").Flat || !m.panel(m.stage).Counts("assignees") {
		t.Errorf("a click past assignees' flatten mark: %d keys, %q, flat %v", keys(m), mark(m, "assignees"),
			m.panel(m.stage).Field("assignees").Flat)
	}
	// a click on it flattens the list, a row per value, which keeps its sort, by value now; the mark is then the
	// gather's, which a click gathers
	if m = drive(m, testkit.Click(under(m, "assignees"), ruleY(m))); !m.panel(m.stage).Field("assignees").Flat ||
		marks(m, "assignees", 3) != table.GatherGlyph+"▲³" || m.panel(m.stage).Counts("assignees") {
		t.Fatalf("a click on assignees' flatten mark: flat %v, %q", m.panel(m.stage).Field("assignees").Flat,
			marks(m, "assignees", 3))
	}
	// and a click on it again gathers the values back on one row, sorted by size again
	if m = drive(m, testkit.Click(under(m, "assignees"), ruleY(m))); m.panel(m.stage).Field("assignees").Flat ||
		keys(m) != 3 || marks(m, "assignees", 3) != table.FlattenGlyph+"▲³" {
		t.Errorf("a click on the flatten mark, flat: %d keys, %q", keys(m), marks(m, "assignees", 3))
	}
	// a click after it turns the sort off, for the clicks below
	m = drive(m, testkit.Click(under(m, "assignees")+1, ruleY(m)), testkit.Click(under(m, "assignees")+1, ruleY(m)))
	if got := lines(m)[m.headerTop()]; got != header {
		t.Errorf("sorting moved the names:\n%s\n%s", header, got)
	}
	// the name itself drops the column menu down, and sorts nothing
	if m = drive(m, testkit.Click(after(m, "change")-2, m.headerTop())); !m.colMenu.Open || keys(m) != 2 {
		t.Errorf("a click on change's name: menu %v, %d keys", m.colMenu.Open, keys(m))
	}
}

// Shift with left or right, while a column's menu is open, moves the column a place among those the header shows, the
// menu with it, and names itself on the keys bar where it moves the column; at the header's end it moves nothing. From
// a name in the heading over the groups, it moves the column a depth among them, the menu on its name where it now
// stands.
func TestShiftWithLeftOrRightMovesTheColumnOfAnOpenMenu(t *testing.T) {
	m := loaded(t, viewing(""), tea.WindowSizeMsg{Width: 140, Height: 40})
	// the columns the header names, apart from source, which the view grouped the rows by as it loaded
	shown := func() []string {
		return slices.DeleteFunc(slices.Clone(m.shownColumns(m.stage)), func(id string) bool {
			_, name := m.columnName(id)
			return name == ""
		})
	}
	first := shown()[0]
	column := shown()[1]
	m.openColumnMenu(column)
	x0, _ := m.columnName(column)
	if !strings.Contains(keysOf(m), "⇧←→ move") {
		t.Errorf("the keys bar: %q", keysOf(m))
	}
	if m = drive(m, testkit.Press("shift+left")); shown()[0] != column || !m.colMenu.Open {
		t.Fatalf("shift+left on %s: %q", column, shown())
	}
	if _, left := table.Shifted(&m, column, -1); left || !strings.Contains(keysOf(m), "⇧←→ move") {
		t.Errorf("%s, first, moved further left", column)
	}
	if x, _ := m.columnName(column); x >= x0 {
		t.Errorf("the menu stayed behind: %d", x)
	}
	if m = drive(m, testkit.Press("shift+right")); shown()[0] != first || shown()[1] != column {
		t.Errorf("shift+right: %q", shown())
	}
	if _, ok := table.ShiftStep("left"); ok {
		t.Error("left alone is a shift")
	}
	m = drive(m, esc)
	// a hidden column does not move
	if _, ok := table.Shifted(&m, "branch", 1); ok && m.panel(m.stage).Field("branch").Placement == grid.Hidden {
		t.Error("a hidden column moved")
	}
	// grouped by source, then assignees: shift+left in assignees' menu, from its name in the heading over the groups,
	// makes it the outer group, the menu on its name still, now first, which shift+left then cannot move; shift+right
	// moves it back in
	m.setPanel(regroup(m.panel(m.stage), "source", "assignees"))
	m.openFieldMenu("assignees")
	x0, _ = m.fieldName("assignees")
	if m = drive(m, testkit.Press("shift+left")); !slices.Equal(m.panel(m.stage).Groups(), []string{"assignees", "source"}) ||
		!m.colMenu.Open || !m.colMenu.Heading {
		t.Fatalf("shift+left on assignees' name: %q, menu %+v", m.panel(m.stage).Groups(), m.colMenu)
	}
	if box, top, _ := m.colMenu.Box(&m); top != m.top() || len(box) == 0 {
		t.Errorf("the column menu stands at %d", top)
	}
	if x, _ := m.fieldName("assignees"); x >= x0 {
		t.Errorf("the menu stayed behind: %d", x)
	}
	if m = drive(m, testkit.Press("shift+left")); m.panel(m.stage).Groups()[0] != "assignees" || !m.colMenu.Open {
		t.Errorf("shift+left on the outer group: %q", m.panel(m.stage).Groups())
	}
	if m = drive(m, testkit.Press("shift+right")); !slices.Equal(m.panel(m.stage).Groups(), []string{"source", "assignees"}) {
		t.Errorf("shift+right: %q", m.panel(m.stage).Groups())
	}
}

// The column menu holds the sections a field can use, in every column's order: its sort, by size for a list a row holds
// whole; its grouping; its flattening; then Hide and Options…. One that cannot sort has no sort, its radio and Clear
// Sort gone; one that holds no list cannot flatten; and one that cannot group has no Group or Ungroup; each section's
// rule goes with it. What a field can do but cannot now stays, dim.
func TestTheColumnMenuLeavesOutWhatTheFieldCannotDo(t *testing.T) {
	m := loaded(t, viewing(""), tea.WindowSizeMsg{Width: 140, Height: 40})
	names := func(column string) []string {
		m.colMenu = table.ColumnMenu{Dropdown: shell.Dropdown{Open: true}, Column: column}
		var out []string
		for _, i := range m.colMenu.Items(&m) {
			out = append(out, i.Name)
		}
		return out
	}
	if got := names("title"); !slices.Equal(got, []string{"▲ Ascending", "▼ Descending", table.Unsorted, "", table.Hide,
		table.More}) || m.colMenu.LineOf(&m, table.ActGroup) != -1 || m.colMenu.LineOf(&m, table.ActUngroup) != -1 {
		t.Errorf("title, which sorts and cannot group: %q", got)
	}
	if got := names("labels"); !slices.Equal(got, []string{"▲ Ascending" + table.BySize, "▼ Descending" + table.BySize,
		table.Unsorted, "", table.GroupIt, table.UngroupIt, "", table.FlattenIt("label"), table.GatherIt("labels"), "",
		table.Hide, table.More}) {
		t.Errorf("labels, a list, which sorts by size and groups: %q", got)
	}
	at := m.stage
	m.stage = slices.IndexFunc(m.stages, func(s stage) bool { return s.name == "builds" })
	if got := names("commit"); !slices.Equal(got, []string{table.Hide, table.More}) {
		t.Errorf("builds' commit, which neither sorts nor groups: %q", got)
	}
	m.stage = at
	if got := names("assignees"); !slices.Contains(got, table.GroupIt) || got[0] != "▲ Ascending"+table.BySize ||
		slices.Index(got, table.GroupIt) > slices.Index(got, table.FlattenIt("assignee")) {
		t.Errorf("assignees, a list, which sorts by size, then groups, then flattens: %q", got)
	}
	m.colMenu = table.ColumnMenu{}
}
