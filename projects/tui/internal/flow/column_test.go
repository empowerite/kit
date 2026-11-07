package flow

import (
	"slices"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/empowerite/kit/projects/tui/internal/grid"
	"github.com/empowerite/kit/projects/tui/internal/testkit"
)

// A right click on a column's name drops the column menu down under it, as a terminal that keeps ctrl and option with
// a click for itself still sends the right button: sort by the column alone or after the other keys, group by it
// alone or within the groups, and its options; a line the column cannot act on stays, dim.
func TestARightClickOnAColumnsNameDropsItsMenu(t *testing.T) {
	m := loaded(t, viewing(""), tea.WindowSizeMsg{Width: 140, Height: 40})
	at := func(name string) int { return strings.Index(lines(m)[paneTop], name) }
	rightClick := func(name string) tea.MouseClickMsg {
		return tea.MouseClickMsg{X: at(name), Y: paneTop, Button: tea.MouseRight}
	}
	group := func() []string { return m.panel(m.stage).Groups() }
	sorts := func() int { return len(m.panel(m.stage).SortKeys()) }
	if m = drive(m, rightClick("assignees")); !m.colMenu.open || m.colMenu.column != "assignees" ||
		m.colMenu.at != lineOff {
		t.Fatalf("a right click on assignees: %+v\n%s", m.colMenu, raw(m))
	}
	var names []string
	for _, it := range m.columnItems() {
		names = append(names, it.Name)
	}
	want := []string{"sort", "off", "▲ ascending", "▼ descending", ThenSort + "assignees", "", GroupBy + "assignees",
		ThenGroup + "assignees", "", More}
	if !slices.Equal(names, want) || !strings.Contains(raw(m), ThenGroup+"assignees") ||
		!strings.Contains(keysOf(m), "esc close") {
		t.Fatalf("the menu: %q, %q", names, keysOf(m))
	}
	// the radio has off picked; enter on ▲ ascending sorts by it alone; Then sort by, chosen by a click, adds age after it
	if !strings.Contains(m.columnItems()[lineOff].Glyph, grid.Picked) {
		t.Errorf("off is not picked: %+v", m.columnItems()[lineOff])
	}
	if m = drive(m, down, press(tea.KeyEnter)); m.colMenu.open || sorts() != 1 || m.panel(m.stage).Mark("assignees") != "▲" {
		t.Fatalf("Sort by: %+v", m.panel(m.stage).SortKeys())
	}
	m = drive(m, rightClick("age"))
	box, top, left := m.columnBox()
	if m = drive(m, testkit.Click(left+3, top+1+lineThen)); m.colMenu.open || sorts() != 2 {
		t.Fatalf("Then sort by age: %+v\n%v", m.panel(m.stage).SortKeys(), box)
	}
	// Then group by groups within the groups; a letter moves to the next line it starts, and Group by groups by it
	// alone
	m.setPanel(m.panel(m.stage).Grouped("source", false))
	// off, ▲, ▼, past Then sort by, dim on a key, and the rule, to Group by, then Then group by
	m = drive(m, rightClick("assignees"), down, down, down, up, down, down)
	if m.colMenu.at != lineThenGroup {
		t.Fatalf("down to Then group by: %d", m.colMenu.at)
	}
	if m = drive(m, press(tea.KeySpace)); !slices.Equal(group(), []string{"source", "assignees"}) {
		t.Fatalf("Then group by assignees: %q", group())
	}
	m.setPanel(m.panel(m.stage).Grouped("assignees", true))
	m = drive(m, rightClick("assignees"))
	if m = drive(m, typed("g")...); m.colMenu.at != lineGroup {
		t.Errorf("g in the menu: %d", m.colMenu.at)
	}
	if m = drive(m, press(tea.KeyEnter)); !slices.Equal(group(), []string{"assignees"}) {
		t.Fatalf("Group by assignees: %q", group())
	}
	// title sorts and does not group: its groupings are dim, and do nothing chosen
	m = drive(m, rightClick("title"))
	if m.columnEnabled(lineGroup) || m.columnEnabled(lineThenGroup) || !m.columnEnabled(lineUp) || m.columnEnabled(5) ||
		m.columnEnabled(lineSort) {
		t.Errorf("title's lines: %v %v %v", m.columnEnabled(lineUp), m.columnEnabled(lineGroup), m.columnEnabled(5))
	}
	if m.columnAct(lineGroup); len(group()) != 1 || m.colMenu.open {
		t.Errorf("Group by title: %q", group())
	}
	// Options… opens the field's popover; esc closes the menu; a click off it closes it too
	m = drive(m, append([]tea.Msg{rightClick("title")}, typed("o")...)...)
	if m = drive(m, press(tea.KeyEnter)); !m.pop.open || m.pop.field != "title" {
		t.Fatalf("Options…: %+v", m.pop)
	}
	m = drive(m, esc, rightClick("age"), esc)
	if m.colMenu.open || m.pop.open {
		t.Errorf("esc: menu %v, popover %v", m.colMenu.open, m.pop.open)
	}
	if m = drive(m, rightClick("age"), testkit.Click(1, 30)); m.colMenu.open || sorts() != 2 {
		t.Errorf("a click off the menu: open %v, sorts %d", m.colMenu.open, sorts())
	}
	// a right click on the header's gutter, off every name, opens nothing
	if m = drive(m, tea.MouseClickMsg{X: 1, Y: paneTop, Button: tea.MouseRight}); m.colMenu.open {
		t.Errorf("a right click off the names opened the menu")
	}
	// nor does one under the rows, or a click on the rule of a table with no preset line
	if m = drive(m, tea.MouseClickMsg{X: 10, Y: 30, Button: tea.MouseRight}); m.colMenu.open {
		t.Errorf("a right click under the rows opened the menu")
	}
	m.noPresets = true
	if m = drive(m, testkit.Click(10, paneTop+1)); m.colMenu.open || m.pop.open {
		t.Errorf("a click on the rule opened a menu")
	}
}

// The menu's radio shows the column's sort: ▲ or ▼ on a column not sorting sorts by it alone, and on one among the keys
// turns it in its place; off takes it out of the keys. Then sort by adds it after the others, ascending, and is dim once
// it is a key; the heading says its place among the keys while there are several. A click on a column's ▲ turns
// it to ▼ and back, and a shift-click on it, where the terminal sends shift, stops the sort; the menu stays shut.
func TestTheSortMarkTurnsAndTheMenuSorts(t *testing.T) {
	m := loaded(t, viewing(""), tea.WindowSizeMsg{Width: 140, Height: 40})
	header := func() string { return lines(m)[paneTop] }
	at := func(text string) int { return ansi.StringWidth(header()[:strings.Index(header(), text)]) }
	sorts := func() []grid.SortKey { return m.panel(m.stage).SortKeys() }
	menu := func(name string, line int) {
		m = drive(m, testkit.Click(at(name), paneTop))
		m.columnAct(line)
	}
	if m.colMenu.column = "age"; !m.columnEnabled(lineThen) || !m.columnEnabled(lineOff) {
		t.Errorf("age, not a key: then %v, off %v", m.columnEnabled(lineThen), m.columnEnabled(lineOff))
	}
	m.colMenu = columnMenu{}
	menu("age", lineThen)
	menu("id", lineThen)
	if m.colMenu.column = "id"; m.columnItems()[lineSort].Name != "sort, key 2 of 2" || m.columnEnabled(lineThen) {
		t.Errorf("id, the second key: %q, then %v", m.columnItems()[lineSort].Name, m.columnEnabled(lineThen))
	}
	m.colMenu = columnMenu{}
	if !slices.Equal(sorts(), []grid.SortKey{{Column: "age"}, {Column: "id"}}) || !strings.Contains(header(), "age ▲₁") {
		t.Fatalf("then by age, then by id: %+v\n%s", sorts(), header())
	}
	// the ▲ of age turns, and age keeps its place among the keys; the menu stays shut
	if m = drive(m, testkit.Click(at("▲₁"), paneTop)); m.colMenu.open ||
		!slices.Equal(sorts(), []grid.SortKey{{Column: "age", Desc: true}, {Column: "id"}}) {
		t.Fatalf("a click on age's ▲: %+v, menu %v\n%s", sorts(), m.colMenu.open, header())
	}
	if m = drive(m, testkit.Click(at("▼₁")+1, paneTop)); !slices.Equal(sorts(), []grid.SortKey{{Column: "age"}, {Column: "id"}}) {
		t.Fatalf("a click on age's ₁: %+v\n%s", sorts(), header())
	}
	shift := tea.MouseClickMsg{X: at("▲₂") - 1, Y: paneTop, Button: tea.MouseLeft, Mod: tea.ModShift}
	if m = drive(m, shift); !slices.Equal(sorts(), []grid.SortKey{{Column: "age"}}) || m.colMenu.open {
		t.Fatalf("a shift-click before id's ▲: %+v\n%s", sorts(), header())
	}
	// ▼ on id, not a key, sorts by it alone; ▲ on it then turns it; off on it leaves nothing sorting
	menu("id", lineDown)
	if !slices.Equal(sorts(), []grid.SortKey{{Column: "id", Desc: true}}) {
		t.Fatalf("▼ on id: %+v", sorts())
	}
	if m.colMenu.column = "id"; !strings.Contains(m.columnItems()[lineDown].Glyph, grid.Picked) {
		t.Errorf("▼ is not picked: %+v", m.columnItems()[lineDown])
	}
	m.colMenu = columnMenu{}
	menu("age", lineThen)
	if menu("id", lineUp); !slices.Equal(sorts(), []grid.SortKey{{Column: "id"}, {Column: "age"}}) {
		t.Fatalf("▲ on id, the first of two keys: %+v", sorts())
	}
	if menu("id", lineOff); !slices.Equal(sorts(), []grid.SortKey{{Column: "age"}}) {
		t.Errorf("off on id: %+v", sorts())
	}
}

// A click on the cell right after a column's name sorts by it alone, ascending, where it does not sort yet, and turns
// its sort around where it does, never off; the cell after that does the same, unless it stands against the next
// column's name, where a click does nothing. The name itself still drops the column menu down.
func TestAClickJustAfterAColumnsNameSortsByIt(t *testing.T) {
	m := loaded(t, options(), tea.WindowSizeMsg{Width: 120, Height: 30})
	// after gives the screen column of the cell right after a column's name, its sort mark apart
	after := func(m Model, column string) int {
		j := slices.Index(m.shownColumns(m.stage), column)
		start, first := grid.HeaderName(m.rows(m.stage), m.names(m.stage), m.gridWidth(m.stage), j)
		return m.indent(m.stage) + start + ansi.StringWidth(strings.TrimSuffix(first, " "+m.panel(m.stage).Mark(column)))
	}
	keys := func(m Model) int { return len(m.panel(m.stage).SortKeys()) }
	mark := func(m Model, column string) string { return m.panel(m.stage).Mark(column) }
	// age stands two cells before branch: the cell after it sorts, the next, against branch's name, does nothing
	if m = drive(m, testkit.Click(after(m, "age")+1, paneTop)); keys(m) != 0 || m.colMenu.open {
		t.Fatalf("a click against branch's name: %d keys, menu %v", keys(m), m.colMenu.open)
	}
	if m = drive(m, testkit.Click(after(m, "age"), paneTop)); keys(m) != 1 || mark(m, "age") != "▲" || m.colMenu.open {
		t.Fatalf("a click right after age: %d keys, age %q, menu %v", keys(m), mark(m, "age"), m.colMenu.open)
	}
	// again turns it around, and again back, never off
	for _, want := range []string{"▼", "▲"} {
		if m = drive(m, testkit.Click(after(m, "age"), paneTop)); keys(m) != 1 || mark(m, "age") != want {
			t.Fatalf("another click after age: %d keys, age %q", keys(m), mark(m, "age"))
		}
	}
	// assignees has room after its name: the second cell sorts by it alone too
	if m = drive(m, testkit.Click(after(m, "assignees")+1, paneTop)); keys(m) != 1 || mark(m, "assignees") != "▲" {
		t.Errorf("a click two cells after assignees: %d keys, assignees %q", keys(m), mark(m, "assignees"))
	}
	// off the names' line, as on the rule under them, the cells after a name sort nothing
	if column := m.sortAt(after(m, "assignees"), paneTop+1); column != "" {
		t.Errorf("the rule under assignees sorts %q", column)
	}
	// the name itself drops the column menu down, and sorts nothing
	if m = drive(m, testkit.Click(after(m, "branch")-2, paneTop)); !m.colMenu.open || keys(m) != 1 {
		t.Errorf("a click on branch's name: menu %v, %d keys", m.colMenu.open, keys(m))
	}
}
