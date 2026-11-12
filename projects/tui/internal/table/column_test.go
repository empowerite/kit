// A column's card sorts, groups, flattens, hides, and moves its column, hanging from its name.
package table

import (
	"slices"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/empowerite/kit/projects/tui/internal/grid"
	"github.com/empowerite/kit/projects/tui/internal/shell"
)

// lineOf gives the index of the card's first line that reads as text, its glyph apart, -1 for none.
func lineOf(p Popover, h Host, text string) int {
	return slices.IndexFunc(texts(p, h), func(l string) bool {
		_, rest, _ := strings.Cut(l, " ")
		return l == text || rest == text
	})
}

// act acts on the card's line that reads as text, as a click on it does.
func act(t *testing.T, p *Popover, h Host, text string) {
	t.Helper()
	k := lineOf(*p, h, text)
	if k < 0 {
		t.Fatalf("no line %q in %q", text, texts(*p, h))
	}
	p.Act(h, k)
}

func TestTheCardSortsGroupsAndHidesItsColumn(t *testing.T) {
	h := listing()
	p := OpenCard(h, "state", false)
	got := texts(p, h)
	if !slices.Equal(got[:7], []string{"sort", grid.Unpicked + " ▲ Ascending", grid.Unpicked + " ▼ Descending",
		grid.Picked + " " + Unsorted, "", "rows", grid.Unticked + " " + GroupIt}) || got[8] != "filtering" ||
		got[len(got)-1] != grid.Ticked+" "+ShowIt || p.At != 1 || !p.Header || p.X != 4 || p.Y != 6 {
		t.Fatalf("state's card, at %d from %d, %d: %q", p.At, p.X, p.Y, got)
	}
	// title neither sorts nor groups: its card has neither section
	if got := texts(OpenCard(h, "title", false), h); got[0] != "rendering" {
		t.Errorf("title's card: %q", got)
	}
	// Ascending sorts, and the card stays open; Descending turns it
	act(t, &p, h, "▲ Ascending")
	if !p.Open || h.panel.Field("state").Sort != grid.Ascending || p.Acts(h) {
		t.Fatalf("ascending: %+v", h.panel.Field("state"))
	}
	act(t, &p, h, "▼ Descending")
	if h.panel.Field("state").Sort != grid.Descending {
		t.Errorf("descending: %+v", h.panel.Field("state"))
	}
	// Group groups the rows by it, its depth after it; again, no longer
	act(t, &p, h, GroupIt)
	grouped := GroupIt + " " + grid.GroupMark(h.panel.Field("state").Depth)
	if !slices.Equal(h.panel.Groups(), []string{"state"}) || lineOf(p, h, grouped) < 0 ||
		texts(p, h)[lineOf(p, h, grouped)] != grid.Ticked+" "+grouped {
		t.Fatalf("grouped: %q, %q", h.panel.Groups(), texts(p, h))
	}
	act(t, &p, h, grouped)
	if len(h.panel.Groups()) != 0 {
		t.Errorf("ungrouped: %q", h.panel.Groups())
	}
	// with another key sorting the rows, state's sort carries ⁿ and ₙ; Unsorted takes it out of the keys
	h.panel.Sortable["title"] = true
	h.panel = h.panel.SortedAs("title", grid.Ascending)
	if lineOf(p, h, "▼ₙ Descending") < 0 {
		t.Errorf("a key with another: %q", texts(p, h))
	}
	act(t, &p, h, Unsorted)
	if keys := h.panel.ColumnKeys(); len(keys) != 1 || keys[0].Column != "title" {
		t.Errorf("state unsorted: %v", keys)
	}
	// Show hides it, and shows it again
	act(t, &p, h, ShowIt)
	if h.panel.Field("state").Placement != grid.Hidden || texts(p, h)[lineOf(p, h, ShowIt)] != grid.Unticked+" "+ShowIt {
		t.Errorf("hidden: %+v", h.panel.Field("state"))
	}
	act(t, &p, h, ShowIt)
	if h.panel.Field("state").Placement != grid.Column {
		t.Errorf("shown: %+v", h.panel.Field("state"))
	}
}

// A column that holds a list sorts by size while a row holds all its values, how many each holds; its card flattens
// it, a row per value, and gathers them again, a radio; flattened, it sorts by value, and the columns dialog lists it
// by one value's name; gathered, its sort orders the rows by size again.
func TestTheCardFlattensAListAndGathersItAgain(t *testing.T) {
	h := listing()
	h.panel.Ones = map[string]string{"who": "person"}
	p := OpenCard(h, "who", false)
	flatten, gather := FlattenIt("person"), GatherIt("who")
	if lineOf(p, h, "▲ Ascending"+BySize) < 0 || texts(p, h)[lineOf(p, h, gather)] != grid.Picked+" "+gather ||
		texts(p, h)[lineOf(p, h, flatten)] != grid.Unpicked+" "+flatten {
		t.Fatalf("who's card: %q", texts(p, h))
	}
	act(t, &p, h, flatten)
	if !h.panel.Field("who").Flat || lineOf(p, h, "▲ Ascending") < 0 || !strings.Contains(Columns{}.Line(h, "who"), "person") {
		t.Fatalf("who flat: %q, its line %q", texts(p, h), Columns{}.Line(h, "who"))
	}
	act(t, &p, h, "▼ Descending")
	act(t, &p, h, gather)
	if f := h.panel.Field("who"); f.Flat || !slices.Equal(h.panel.SortKeys(), []grid.SortKey{{Column: "who", Desc: true}}) ||
		lineOf(p, h, "▼ Descending"+BySize) < 0 {
		t.Errorf("who gathered: %+v, keys %v", f, h.panel.SortKeys())
	}
}

// A card opened from a column's name hangs from it, in the header or the heading over the groups, its title over the
// name, and follows it as shift with left or right moves the column; where the name no longer shows, it stays.
func TestTheCardHangsFromItsNameAndMovesItsColumn(t *testing.T) {
	h := listing()
	p := OpenCard(h, "who", false)
	box, top, left := p.Box(h)
	if p.X != 12 || p.Y != 6 || top != 6 || left != 9 || !strings.Contains(ansi.Strip(box[0]), "who") {
		t.Errorf("from the header: %d, %d at %d, %d", p.X, p.Y, top, left)
	}
	if title := OpenCard(h, "title", false); title.X != 0 || title.Y != 0 {
		t.Errorf("a name the header does not draw: %d, %d", title.X, title.Y)
	}
	// from the heading, while the column groups the rows; it stays once the column no longer does
	h.panel = h.panel.Grouped("who", true)
	p = OpenCard(h, "who", true)
	if p.X != 4 || p.Y != 4 || !p.Heading || p.Header {
		t.Errorf("from the heading: %+v", p)
	}
	act(t, &p, h, GroupIt+" "+grid.GroupMark(h.panel.Field("who").Depth))
	if _, top, _ := p.Box(h); len(h.panel.Groups()) != 0 || p.X != 4 || top != 4 || !p.Open {
		t.Errorf("no longer grouping: %+v at %d", p, top)
	}
	// shift with right moves a column among those the header names, the card with it, never past title
	h = listing()
	p = OpenCard(h, "state", false)
	if keys := (shell.Path{p.Keys(h)}).String(); !strings.Contains(keys, "⇧←→") {
		t.Errorf("state's keys: %q", keys)
	}
	p.Key(h, tea.KeyPressMsg{Code: tea.KeyRight, Mod: tea.ModShift})
	if got := h.panel.Arrange(h.panel.Columns); !slices.Equal(got[:2], []string{"who", "state"}) || p.X != 12 {
		t.Errorf("shift+right: %q, the card at %d", got, p.X)
	}
	if p.Key(h, tea.KeyPressMsg{Code: tea.KeyRight, Mod: tea.ModShift}); p.X != 12 || !p.Open {
		t.Errorf("past the last the header names: at %d", p.X)
	}
	if _, ok := Shifted(h, "state", 1); ok || ShiftKeys(h, "nonesuch") != nil {
		t.Error("moved past the last the header names")
	}
	// a card beside the columns dialog moves nothing, and names no such key
	beside := OpenPopover(h, "state", 0, 0, false)
	if beside.Key(h, tea.KeyPressMsg{Code: tea.KeyLeft, Mod: tea.ModShift}); h.panel.Arrange(h.panel.Columns)[1] != "state" ||
		strings.Contains((shell.Path{beside.Keys(h)}).String(), "⇧←→") {
		t.Error("a card beside the dialog moved its column")
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
