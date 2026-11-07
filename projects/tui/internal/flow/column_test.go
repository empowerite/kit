package flow

import (
	"slices"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

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
	if m = drive(m, rightClick("assignees")); !m.colMenu.open || m.colMenu.column != "assignees" || m.colMenu.at != 0 {
		t.Fatalf("a right click on assignees: %+v\n%s", m.colMenu, raw(m))
	}
	var names []string
	for _, it := range m.columnItems() {
		names = append(names, it.Name)
	}
	want := []string{SortBy + "assignees", ThenSort + "assignees", GroupBy + "assignees", ThenGroup + "assignees", "",
		More}
	if !slices.Equal(names, want) || !strings.Contains(raw(m), ThenGroup+"assignees") ||
		!strings.Contains(keysOf(m), "esc close") {
		t.Fatalf("the menu: %q, %q", names, keysOf(m))
	}
	// enter on Sort by sorts by it alone; Then sort by, chosen by a click, adds age after it
	if m = drive(m, press(tea.KeyEnter)); m.colMenu.open || sorts() != 1 || m.panel(m.stage).Mark("assignees") == "" {
		t.Fatalf("Sort by: %+v", m.panel(m.stage).SortKeys())
	}
	m = drive(m, rightClick("age"))
	box, top, left := m.columnBox()
	if m = drive(m, testkit.Click(left+3, top+2)); m.colMenu.open || sorts() != 2 {
		t.Fatalf("Then sort by age: %+v\n%v", m.panel(m.stage).SortKeys(), box)
	}
	// Then group by groups within the groups; a letter moves to the next line it starts, and Group by groups by it
	// alone
	m.setPanel(m.panel(m.stage).Grouped("source", false))
	m = drive(m, rightClick("assignees"), down, down, down, up, down)
	if m = drive(m, press(tea.KeySpace)); !slices.Equal(group(), []string{"source", "assignees"}) {
		t.Fatalf("Then group by assignees: %q", group())
	}
	m.setPanel(m.panel(m.stage).Grouped("assignees", true))
	m = drive(m, rightClick("assignees"))
	if m = drive(m, typed("g")...); m.colMenu.at != 2 {
		t.Errorf("g in the menu: %d", m.colMenu.at)
	}
	if m = drive(m, press(tea.KeyEnter)); !slices.Equal(group(), []string{"assignees"}) {
		t.Fatalf("Group by assignees: %q", group())
	}
	// title sorts and does not group: its groupings are dim, and do nothing chosen
	m = drive(m, rightClick("title"))
	if m.columnEnabled(2) || m.columnEnabled(3) || !m.columnEnabled(0) || m.columnEnabled(4) {
		t.Errorf("title's lines: %v %v %v", m.columnEnabled(0), m.columnEnabled(2), m.columnEnabled(4))
	}
	if m.columnAct(2); len(group()) != 1 || m.colMenu.open {
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
}
