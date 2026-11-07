// The heading menu, which a plain click on a group's heading's text drops, or ⏎ on the heading, so what option, ctrl,
// and the right button do on a heading is there in a terminal that keeps those clicks.
package flow

import (
	"slices"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/empowerite/kit/projects/tui/internal/testkit"
)

// names gives the heading menu's lines, a rule as "".
func headingNames(m Model) []string {
	var out []string
	for _, i := range m.headingItems() {
		out = append(out, i.Name)
	}
	return out
}

// A click on a heading's text drops the menu around it, the first line that acts under the keyboard: everything under
// it collapsed or expanded, the group selected or unselected, and the rows ungrouped by its column. A line that cannot
// act stays, dim; esc, or a click off the menu, closes it.
func TestAClickOnAHeadingsTextDropsItsMenu(t *testing.T) {
	m := nested(t)
	all := len(m.rows(0))
	x, text := headingText(m.rows(0)[0])
	if m = drive(m, testkit.Click(x+1, m.rowsTop())); !m.headMenu.open || m.stages[0].cursor != 0 {
		t.Fatalf("a click on the heading's text: menu %v, cursor %d", m.headMenu.open, m.stages[0].cursor)
	}
	want := []string{ExpandUnder, CollapseUnder, "", SelectGroup, "", Ungroup + "source"}
	// the rows under it are closed, so both expand and collapse act
	if got := headingNames(m); !slices.Equal(got, want) || !m.headingEnabled(0) || !m.headingEnabled(1) ||
		m.headMenu.at != 0 {
		t.Fatalf("the menu: %q, at %d", got, m.headMenu.at)
	}
	box, top, _ := m.headingBox()
	if top != m.rowsTop() || !strings.Contains(ansi.Strip(box[0]), text) || !strings.Contains(raw(m), CollapseUnder) ||
		!strings.Contains(lines(m)[len(lines(m))-1], "esc") {
		t.Errorf("the box hangs at %d, %q:\n%s", top, ansi.Strip(box[0]), raw(m))
	}
	// down and enter collapse everything under the heading, and close the menu
	if m = drive(m, down, press(tea.KeyEnter)); m.headMenu.open || len(m.rows(0)) != 1 {
		t.Fatalf("collapse all under it: menu %v, %d rows", m.headMenu.open, len(m.rows(0)))
	}
	// ⏎ on the heading drops it too; e finds expand, which opens everything again
	if last := lines(m)[len(lines(m))-1]; !strings.Contains(last, "⏎ menu") {
		t.Errorf("the keys on a heading: %q", last)
	}
	if m = drive(m, press(tea.KeyEnter), typed("e")[0], press(tea.KeyEnter)); len(m.rows(0)) != all {
		t.Fatalf("expand all under it: %d rows of %d", len(m.rows(0)), all)
	}
	// a click on Select Group selects every row of the group; it then reads Unselect Group, and unselects them
	m = drive(m, testkit.Click(x+1, m.rowsTop()))
	box, top, left := m.headingBox()
	if m = drive(m, testkit.Click(left+3, top+1+3)); m.selected() != 3 || m.headMenu.open {
		t.Fatalf("select group: %d selected\n%s", m.selected(), strings.Join(box, "\n"))
	}
	if m = drive(m, testkit.Click(x+1, m.rowsTop())); headingNames(m)[3] != UnselectGroup {
		t.Fatalf("with the group selected: %q", headingNames(m))
	}
	m.headMenu.at = 3
	if m = drive(m, press(tea.KeySpace)); m.selected() != 0 {
		t.Fatalf("unselect group: %d selected", m.selected())
	}
	// esc, and a click off the menu, close it, acting on nothing
	if m = drive(m, testkit.Click(x+1, m.rowsTop()), esc); m.headMenu.open {
		t.Error("esc did not close the menu")
	}
	if m = drive(m, testkit.Click(x+1, m.rowsTop()), testkit.Click(x+1, m.rowsTop()+8)); m.headMenu.open ||
		len(m.rows(0)) != all || m.selected() != 0 {
		t.Error("a click off the menu did something, or left it open")
	}
	// with everything open, Expand All Under It is dim: up passes over it, and a click on it does nothing but close
	m = drive(m, tea.KeyPressMsg{Code: tea.KeyRight, Mod: tea.ModAlt}, testkit.Click(x+1, m.rowsTop()), up)
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
	// on an inner heading, Ungroup by assignees leaves the rows grouped by source alone
	ix, _ := headingText(m.rows(0)[1])
	m = drive(m, testkit.Click(ix+1, m.rowsTop()+1))
	m.headMenu.at = 5
	if m = drive(m, press(tea.KeyEnter)); !slices.Equal(m.panel(0).Groups(), []string{"source"}) {
		t.Errorf("ungroup by assignees: %q", m.panel(0).Groups())
	}
}

// On the heading over every row, which no column the person chose makes, Ungroup stands dim; once a reload takes the
// heading the menu hangs from away, it draws nothing, and a click closes it.
func TestTheHeadingMenuOverEveryRowAndAfterItsHeadingGoes(t *testing.T) {
	m := loaded(t, options(), tea.WindowSizeMsg{Width: 120, Height: 30})
	x, _ := headingText(m.rows(0)[0])
	m = drive(m, testkit.Click(x+1, m.rowsTop()))
	if got := headingNames(m); !m.headMenu.open || got[5] != strings.TrimSpace(Ungroup) || m.headingEnabled(5) {
		t.Fatalf("the menu over every row: %q, ungroup enabled %v", got, m.headingEnabled(5))
	}
	m.headMenu.key = "a heading a reload took away"
	if box, _, _ := m.headingBox(); box != nil || m.headingEnabled(0) || strings.Contains(raw(m), CollapseUnder) {
		t.Error("the menu of a heading gone still draws")
	}
	if m = drive(m, testkit.Click(x+1, m.rowsTop())); m.headMenu.open {
		t.Error("a click did not close the menu of a heading gone")
	}
}
