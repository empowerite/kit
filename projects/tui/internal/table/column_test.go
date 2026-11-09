// The column menu sorts, groups, hides, and moves its column, and opens its field's popover.
package table

import (
	"slices"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/empowerite/kit/projects/tui/internal/grid"
)

// lines gives the column menu's lines' names.
func names(c ColumnMenu, h Host) []string {
	var out []string
	for _, it := range c.Items(h) {
		out = append(out, it.Name)
	}
	return out
}

func TestTheColumnMenuSortsGroupsAndHidesItsColumn(t *testing.T) {
	h := listing()
	c := OpenColumnMenu(h, "state", false)
	if got := names(c, h); !slices.Equal(got, []string{"▲ Ascending", "▼ Descending", Unsorted, "", GroupIt,
		UngroupIt, "", Hide, More}) || c.At != 0 {
		t.Fatalf("state's menu, at %d: %q", c.At, got)
	}
	if !strings.Contains(c.Items(h)[c.LineOf(h, ActUnsorted)].Glyph, grid.Picked) {
		t.Errorf("Unsorted is not picked on a column that sorts nothing: %+v", c.Items(h))
	}
	if c.Enabled(h, c.LineOf(h, ActUngroup)) || c.Enabled(h, 3) ||
		c.Enabled(h, -1) || !c.Enabled(h, c.LineOf(h, ActHide)) {
		t.Error("what acts with nothing sorted or grouped")
	}
	// who neither sorts nor groups: its menu has neither section
	if got := names(OpenColumnMenu(h, "title", false), h); !slices.Equal(got, []string{Hide, More}) {
		t.Errorf("title's menu: %q", got)
	}
	// Ascending sorts; a key after another carries ⁿ; Unsorted takes the column out of the keys, the others staying
	if _, ok := c.Act(h, c.LineOf(h, ActUp)); ok || c.Open || h.panel.Field("state").Sort != grid.Ascending {
		t.Fatalf("ascending: %+v", h.panel.Field("state"))
	}
	c = OpenColumnMenu(h, "state", false)
	c.Act(h, c.LineOf(h, ActDown))
	if h.panel.Field("state").Sort != grid.Descending {
		t.Errorf("descending: %+v", h.panel.Field("state"))
	}
	c = OpenColumnMenu(h, "state", false)
	c.Act(h, c.LineOf(h, ActGroup))
	if !slices.Equal(h.panel.Groups(), []string{"state"}) {
		t.Errorf("grouped: %q", h.panel.Groups())
	}
	c = OpenColumnMenu(h, "state", false)
	c.Act(h, c.LineOf(h, ActUngroup))
	// with another key sorting the rows, state's sort carries ⁿ and ₙ
	h.panel.Sortable["title"] = true
	h.panel = h.panel.SortedAs("title", grid.Ascending)
	if got := names(OpenColumnMenu(h, "state", false), h)[1]; got != "▼ₙ Descending" {
		t.Errorf("a key with another: %q", got)
	}
	c = OpenColumnMenu(h, "state", false)
	c.Act(h, c.LineOf(h, ActUnsorted))
	if keys := h.panel.ColumnKeys(); len(keys) != 1 || keys[0].Column != "title" {
		t.Errorf("state unsorted: %v", keys)
	}
	c = OpenColumnMenu(h, "state", false)
	c.Act(h, c.LineOf(h, ActHide))
	if h.panel.Field("state").Placement != grid.Hidden {
		t.Errorf("hidden: %+v", h.panel.Field("state"))
	}
	// a line that does not act closes the menu, and does nothing
	c = OpenColumnMenu(h, "who", false)
	sets := h.sets
	if _, ok := c.Act(h, c.LineOf(h, ActUngroup)); ok || c.Open || h.sets != sets {
		t.Error("a dim line acted")
	}
}

func TestTheColumnMenuOpensItsFieldsPopoverAndMovesItsColumn(t *testing.T) {
	h := listing()
	c := OpenColumnMenu(h, "who", false)
	if pop, ok := c.Act(h, c.LineOf(h, ActMore)); !ok || pop.Field != "who" || pop.X != 12 || pop.Y != 6 || !pop.Header {
		t.Errorf("options… from the header: %+v", pop)
	}
	h.panel = h.panel.Grouped("who", true)
	c = OpenColumnMenu(h, "who", true)
	if pop, ok := c.Act(h, c.LineOf(h, ActMore)); !ok || pop.X != 4 || pop.Y != 4 {
		t.Errorf("options… from the heading: %+v", pop)
	}
	// shift with right moves a column among those the header names, never past title, which it does not name
	h = listing()
	c = OpenColumnMenu(h, "state", false)
	if keys := ShiftKeys(h, "state"); len(keys) != 1 {
		t.Errorf("state's shift keys: %+v", keys)
	}
	c.Key(h, tea.KeyPressMsg{Code: tea.KeyRight, Mod: tea.ModShift})
	if got := h.panel.Arrange(h.panel.Columns); !slices.Equal(got[:2], []string{"who", "state"}) {
		t.Errorf("shift+right: %q", got)
	}
	if _, ok := Shifted(h, "state", 1); ok || ShiftKeys(h, "nonesuch") != nil {
		t.Error("moved past the last the header names")
	}
	h.panel = h.panel.Placed("title", grid.Hidden)
	if _, ok := Shifted(h, "title", -1); ok {
		t.Error("a hidden column moved")
	}
	// a group moves among the groups
	h.panel = h.panel.Grouped("state", true).Grouped("who", true)
	if p, ok := Shifted(h, "who", -1); !ok || !slices.Equal(p.Groups(), []string{"who", "state"}) {
		t.Errorf("a group outward: %v", ok)
	}
	if _, ok := ShiftStep("left"); ok {
		t.Error("left is not shift with left")
	}
}

func TestTheColumnMenuHangsFromItsNameAndTakesKeysAndClicks(t *testing.T) {
	h := listing()
	c := OpenColumnMenu(h, "state", false)
	box, top, left := c.Box(h)
	if top != 6 || len(box) != len(c.Items(h))+2 {
		t.Fatalf("from the header: %d, %d, %d lines", top, left, len(box))
	}
	// a key moves through the menu; enter on Descending sorts
	c.Key(h, key("down"))
	if _, ok := c.Key(h, tea.KeyPressMsg{Code: tea.KeyEnter}); ok || h.panel.Field("state").Sort != grid.Descending {
		t.Errorf("enter on descending: %+v", h.panel.Field("state"))
	}
	c = OpenColumnMenu(h, "state", false)
	if c.Key(h, key("esc")); c.Open {
		t.Error("esc left it open")
	}
	// a click on Options… opens the popover; a click on a rule does nothing; one off it closes it
	c = OpenColumnMenu(h, "state", false)
	box, top, left = c.Box(h)
	if pop, ok := c.Click(h, left+3, top+1+c.LineOf(h, ActMore)); !ok || pop.Field != "state" {
		t.Errorf("a click on options…: %+v", pop)
	}
	c = OpenColumnMenu(h, "state", false)
	if c.Click(h, left+3, top+1+c.LineOf(h, ActRule)); !c.Open {
		t.Error("a click on a rule closed it")
	}
	if c.Click(h, 0, 0); c.Open {
		t.Error("a click off it left it open")
	}
	// from the heading, while the column groups the rows; none once it no longer does, and a click closes it
	h.panel = h.panel.Grouped("state", true)
	c = OpenColumnMenu(h, "state", true)
	if _, top, _ := c.Box(h); top != 4 {
		t.Errorf("from the heading: %d", top)
	}
	h.panel = h.panel.Grouped("state", true)
	if box, _, _ := c.Box(h); box != nil {
		t.Error("a box once the heading no longer names it")
	}
	if c.Click(h, 4, 4); c.Open {
		t.Error("a click with no box left it open")
	}
}
