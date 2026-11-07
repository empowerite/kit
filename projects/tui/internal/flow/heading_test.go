// The heading menu, which a plain click on a group's heading's text drops, or ⏎ on the heading, so what option, ctrl,
// and the right button do on a heading is there in a terminal that keeps those clicks.
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

// menuNames gives the heading menu's lines, a rule as "".
func menuNames(m Model) []string {
	var out []string
	for _, i := range m.headingItems() {
		out = append(out, i.Name)
	}
	return out
}

// nameOf gives the screen column a heading of the selected stage, its row k, draws its first column's name on.
func nameOf(m Model, k int) int { return m.headingNames(m.rows(m.stage)[k])[0].from }

// A click on a column's name in a heading drops the menu around the name, titled by it alone, the first line that acts
// under the keyboard: everything under the heading collapsed or expanded, the group selected or unselected, and the
// rows ungrouped by the column. A line that cannot act stays, dim; esc, or a click off the menu, closes it. A click
// elsewhere on the heading, on its value, folds its rows away, and another unfolds them.
func TestAClickOnAColumnsNameInAHeadingDropsItsMenu(t *testing.T) {
	m := nested(t)
	all := len(m.rows(0))
	x := nameOf(m, 0)
	if text := ansi.Strip(lines(m)[m.rowsTop()]); ansi.Cut(text, x, x+len("source")) != "source" {
		t.Fatalf("source does not stand at %d: %q", x, text)
	}
	if m = drive(m, testkit.Click(x-3, m.rowsTop())); m.headMenu.open || len(m.rows(0)) != 1 {
		t.Fatalf("a click on the heading's value: menu %v, %d rows", m.headMenu.open, len(m.rows(0)))
	}
	if m = drive(m, testkit.Click(x-3, m.rowsTop())); len(m.rows(0)) != all {
		t.Fatalf("a second click on the value: %d rows of %d", len(m.rows(0)), all)
	}
	if m = drive(m, testkit.Click(x+2, m.rowsTop())); !m.headMenu.open || m.stages[0].cursor != 0 ||
		m.headMenu.column != "source" {
		t.Fatalf("a click on source: menu %+v, cursor %d", m.headMenu, m.stages[0].cursor)
	}
	want := []string{ExpandGroup, CollapseGroup, "", SelectGroup, "", UngroupIt}
	// the rows under it are closed, so both expand and collapse act
	if got := menuNames(m); !slices.Equal(got, want) || !m.headingEnabled(0) || !m.headingEnabled(1) ||
		m.headMenu.at != 0 || !m.headingEnabled(5) {
		t.Fatalf("the menu: %q, at %d", got, m.headMenu.at)
	}
	box, top, _ := m.headingBox()
	if edge := ansi.Strip(box[0]); top != m.rowsTop() || !strings.Contains(edge, "╴source╶") || strings.Contains(edge, "github") ||
		!strings.Contains(raw(m), CollapseGroup) || !strings.Contains(lines(m)[len(lines(m))-1], "esc") {
		t.Errorf("the box hangs at %d, %q:\n%s", top, edge, raw(m))
	}
	// down and enter collapse everything under the heading, and close the menu
	if m = drive(m, down, press(tea.KeyEnter)); m.headMenu.open || len(m.rows(0)) != 1 {
		t.Fatalf("collapse all under it: menu %v, %d rows", m.headMenu.open, len(m.rows(0)))
	}
	// ⏎ on the heading drops it too, around its column's name; e finds expand, which opens everything again
	if last := lines(m)[len(lines(m))-1]; !strings.Contains(last, "⏎ menu") {
		t.Errorf("the keys on a heading: %q", last)
	}
	if m = drive(m, press(tea.KeyEnter)); m.headMenu.column != "source" || m.headMenu.x != x {
		t.Errorf("⏎ on the heading: %+v", m.headMenu)
	}
	if m = drive(m, typed("e")[0], press(tea.KeyEnter)); len(m.rows(0)) != all {
		t.Fatalf("expand all under it: %d rows of %d", len(m.rows(0)), all)
	}
	// a click on Select Group selects every row of the group; it then reads Unselect Group, and unselects them
	m = drive(m, testkit.Click(x, m.rowsTop()))
	box, top, left := m.headingBox()
	if m = drive(m, testkit.Click(left+3, top+1+3)); m.selected() != 3 || m.headMenu.open {
		t.Fatalf("select group: %d selected\n%s", m.selected(), strings.Join(box, "\n"))
	}
	if m = drive(m, testkit.Click(x, m.rowsTop())); menuNames(m)[3] != UnselectGroup {
		t.Fatalf("with the group selected: %q", menuNames(m))
	}
	m.headMenu.at = 3
	if m = drive(m, press(tea.KeySpace)); m.selected() != 0 {
		t.Fatalf("unselect group: %d selected", m.selected())
	}
	// esc, and a click off the menu, close it, acting on nothing
	if m = drive(m, testkit.Click(x, m.rowsTop()), esc); m.headMenu.open {
		t.Error("esc did not close the menu")
	}
	if m = drive(m, testkit.Click(x, m.rowsTop()), testkit.Click(x, m.rowsTop()+8)); m.headMenu.open ||
		len(m.rows(0)) != all || m.selected() != 0 {
		t.Error("a click off the menu did something, or left it open")
	}
	// with everything open, Expand Group is dim: up passes over it, and a click on it does nothing but close
	m = drive(m, tea.KeyPressMsg{Code: tea.KeyRight, Mod: tea.ModAlt}, testkit.Click(x, m.rowsTop()), up)
	if m.headingEnabled(0) || m.headMenu.at != 1 {
		t.Errorf("up with everything open: at %d, expand enabled %v", m.headMenu.at, m.headingEnabled(0))
	}
	_, top, left = m.headingBox()
	if m = drive(m, testkit.Click(left+3, top+1)); m.headMenu.open || len(m.rows(0)) != all {
		t.Errorf("a click on the dim expand: menu %v, %d rows", m.headMenu.open, len(m.rows(0)))
	}
	if y := m.rowLine(len(m.rows(0)) + 5); y != -1 {
		t.Errorf("a row the pane does not show stands on line %d", y)
	}
	// on an inner heading, a click on assignees, and Ungroup leaves the rows grouped by source alone
	m = drive(m, testkit.Click(nameOf(m, 1), m.rowsTop()+1))
	m.headMenu.at = 5
	if m = drive(m, press(tea.KeyEnter)); !slices.Equal(m.panel(0).Groups(), []string{"source"}) {
		t.Errorf("ungroup by assignees: %q", m.panel(0).Groups())
	}
}

// On the heading over every row, which no column the person chose makes, Ungroup, which the rows group by
// on their own, stands dim; once a reload takes the
// heading the menu hangs from away, it draws nothing, and a click closes it.
func TestTheHeadingMenuOverEveryRowAndAfterItsHeadingGoes(t *testing.T) {
	m := loaded(t, options(), tea.WindowSizeMsg{Width: 120, Height: 30})
	x := nameOf(m, 0)
	m = drive(m, testkit.Click(x, m.rowsTop()))
	if got := menuNames(m); !m.headMenu.open || got[5] != UngroupIt || m.headingEnabled(5) {
		t.Fatalf("the menu over every row: %q, ungroup enabled %v", got, m.headingEnabled(5))
	}
	m.headMenu.key = "a heading a reload took away"
	if box, _, _ := m.headingBox(); box != nil || m.headingEnabled(0) || strings.Contains(raw(m), CollapseGroup) {
		t.Error("the menu of a heading gone still draws")
	}
	if m = drive(m, testkit.Click(x, m.rowsTop())); m.headMenu.open {
		t.Error("a click did not close the menu of a heading gone")
	}
}

// A heading's column names are found after their values: a value that holds a column's name in it is passed over, as
// is a name the heading does not say.
func TestAHeadingsNamesStandAfterTheirValues(t *testing.T) {
	m := loaded(t, options())
	heading := row{Key: "group\x00x", Aside: true, Says: []string{"source", "nowhere"},
		Cells: []grid.Cell{{Values: []string{"▾ source of truth source  3"}}}}
	names := m.headingNames(heading)
	if len(names) != 1 || names[0].column != "source" || names[0].from != 1+ansi.StringWidth("▾ source of truth ") {
		t.Errorf("the names: %+v", names)
	}
	// several columns the rows group by on their own, at one level, each its own name, as the heading says them
	text := "▾ 🐙 github source, empowerite organization, kit repository  12"
	heading = row{Key: "group\x00", Aside: true, Says: []string{"source", "organization", "repository"},
		Cells: []grid.Cell{{Values: []string{text}}}}
	var got []string
	for _, n := range m.headingNames(heading) {
		got = append(got, fmt.Sprintf("%s@%d", n.column, n.from))
		if cut := ansi.Cut(" "+text, n.from, n.until); cut != n.column {
			t.Errorf("%s stands at %d, where the heading draws %q", n.column, n.from, cut)
		}
	}
	if len(got) != 3 {
		t.Errorf("the names of three columns at one level: %q", got)
	}
	// a click on repository drops the menu titled by it alone, Ungroup dim, as the rows group by it on
	// their own
	m.headMenu = headingMenu{open: true, key: heading.Key, column: "repository"}
	if items := m.headingItems(); items[5].Name != UngroupIt || m.headingEnabled(5) {
		t.Errorf("repository's menu: %+v", items)
	}
}
