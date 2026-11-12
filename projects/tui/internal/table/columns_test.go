// The columns dialog lists every field, groups first, then the columns, then the hidden, and shows, hides, sorts,
// groups, and moves the one under the keyboard.
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

// listing gives a host whose panel lists state, who, and title as its columns, state sorting and grouping, who
// grouping alone, and title neither.
func listing() *host {
	h := screen()
	h.panel = grid.Panel{Columns: []string{"state", "who", "title"},
		Labels:    map[string]string{"state": "state", "who": "who", "title": "title"},
		Sortable:  map[string]bool{"state": true},
		Groupable: map[string]bool{"state": true, "who": true},
		Fields:    map[string]grid.Field{}}
	return h
}

// press gives the press of a key by its name.
func press(name string) tea.KeyPressMsg {
	switch name {
	case "enter":
		return tea.KeyPressMsg{Code: tea.KeyEnter}
	case "shift+up":
		return tea.KeyPressMsg{Code: tea.KeyUp, Mod: tea.ModShift}
	case "s", "g":
		return tea.KeyPressMsg{Code: rune(name[0]), Text: name}
	}
	return key(name)
}

func TestTheColumnsDialogShowsSortsGroupsAndMovesAField(t *testing.T) {
	h := listing()
	c := Columns{Open: true}
	if got := c.Listed(h); !slices.Equal(got, []string{"state", "who", "title"}) {
		t.Fatalf("listed: %q", got)
	}
	box, top, left := c.Box(h)
	if top != 4 || left+ansi.StringWidth(box[0]) != h.w-1 || !strings.Contains(ansi.Strip(box[0]), "things fields") {
		t.Errorf("the box at %d, %d: %q", top, left, ansi.Strip(box[0]))
	}
	// space hides state, and again shows it; s sorts it, g groups by it
	c.Key(h, press("space"))
	if h.panel.Field("state").Placement != grid.Hidden || !strings.HasPrefix(ansi.Strip(c.Line(h, "state")), grid.Unticked) {
		t.Errorf("hidden: %q", ansi.Strip(c.Line(h, "state")))
	}
	c.At = slices.Index(c.Listed(h), "state")
	c.Key(h, press("space"))
	c.Key(h, press("s"))
	if h.panel.Field("state").Sort != grid.Ascending {
		t.Errorf("sorted: %+v", h.panel.Field("state"))
	}
	c.Key(h, press("g"))
	if !slices.Equal(h.panel.Groups(), []string{"state"}) || c.Listed(h)[c.At] != "state" {
		t.Errorf("grouped: %q, at %d", h.panel.Groups(), c.At)
	}
	if got := ansi.Strip(c.Line(h, "state")); !strings.HasSuffix(got, grid.GroupMark(1)) {
		t.Errorf("a group's line: %q", got)
	}
	// title neither sorts nor groups: s does nothing, and its line has no dots
	c.At = slices.Index(c.Listed(h), "title")
	if c.Key(h, press("s")); h.panel.Field("title").Sort != grid.Unsorted {
		t.Error("title sorted")
	}
	if got := ansi.Strip(c.Line(h, "title")); strings.Contains(got, "·") {
		t.Errorf("title's line: %q", got)
	}
	// shift with up moves it a place; down and up move the keyboard
	c.Key(h, press("shift+up"))
	if got := c.Listed(h); got[c.At] != "title" || c.At != 1 {
		t.Errorf("shift+up: %q, at %d", got, c.At)
	}
	c.Key(h, press("down"))
	c.Key(h, press("up"))
	if c.At != 1 {
		t.Errorf("down, up: at %d", c.At)
	}
	// a filter on a field marks its line
	h.panel = h.panel.FilteredBy("who", grid.Filter{Values: []string{"ada"}})
	if got := ansi.Strip(c.Line(h, "who")); !strings.Contains(got, grid.Filtered) {
		t.Errorf("a filtered line: %q", got)
	}
	// enter gives the field's card to open, beside the dialog
	if pop, ok := c.Key(h, press("enter")); !ok || pop.Field != c.Listed(h)[c.At] || pop.Header {
		t.Errorf("enter: %+v %v", pop, ok)
	}
	if c.Key(h, press("esc")); c.Open {
		t.Error("esc left it open")
	}
}

func TestTheColumnsDialogTakesClicksAndNamesItsKeys(t *testing.T) {
	h := listing()
	c := Columns{Open: true}
	c.Key(h, press("g")) // state grouped: a rule stands under the groups
	_, top, left := c.Box(h)
	if c.Row(h, top) != -1 || c.Row(h, top+2) != 0 || c.Row(h, top+3) != -1 || c.Row(h, top+4) != 1 || c.Row(h, top+40) != -1 {
		t.Errorf("rows: %d %d %d %d", c.Row(h, top), c.Row(h, top+2), c.Row(h, top+3), c.Row(h, top+4))
	}
	// on who's line: its sort cell does nothing, its group cell groups by it, its box hides it, its label opens its card
	y := top + 4
	c.Click(h, left+4+LabelCells+TypeCells+SortCells, y, false)
	if !slices.Equal(h.panel.Groups(), []string{"state", "who"}) {
		t.Errorf("a click on who's group: %q", h.panel.Groups())
	}
	c.Click(h, left+4+LabelCells+TypeCells, top+2, false)
	if h.panel.Field("state").Sort != grid.Ascending {
		t.Errorf("a click on state's sort: %+v", h.panel.Field("state"))
	}
	c.Click(h, left+3, top+2, false)
	if h.panel.Field("state").Placement != grid.Hidden {
		t.Errorf("a click on state's box: %+v", h.panel.Field("state"))
	}
	if pop, ok := c.Click(h, left+6, y, false); !ok || c.At != 1 || pop.Field != c.Listed(h)[1] || pop.Header || pop.Y != top {
		t.Errorf("a click on a label: at %d, %+v", c.At, pop)
	}
	if _, ok := c.Click(h, left+3, top+2, false); ok {
		t.Error("a click on a box opened a card")
	}
	if c.Click(h, left, top+1, false); !c.Open {
		t.Error("a click on the headings closed it")
	}
	keys := (shell.Path{c.Keys(h)}).String()
	for _, want := range []string{"close", "move", "show/hide", "settings"} {
		if !strings.Contains(keys, want) {
			t.Errorf("the keys %q have no %s", keys, want)
		}
	}
	c.At = slices.Index(c.Listed(h), "state")
	if keys := (shell.Path{c.Keys(h)}).String(); !strings.Contains(keys, "sort") || !strings.Contains(keys, "group") ||
		!strings.Contains(keys, "order") {
		t.Errorf("state's keys: %q", keys)
	}
	if c.Click(h, 0, 0, false); c.Open {
		t.Error("a click off it left it open")
	}
}

// A click on a key's sort before the last turns it between ascending and descending, and a shift-click turns it off;
// on the last key, a click turns it off after descending.
func TestAClickTurnsASortOffOnlyOnTheLastKeyAndAShiftClickAnywhere(t *testing.T) {
	h := listing()
	h.panel.Sortable["who"] = true
	c := Columns{Open: true}
	_, top, left := c.Box(h)
	state, who := top+2, top+3
	sort := left + 4 + LabelCells + TypeCells
	c.Click(h, sort, state, false)
	c.Click(h, sort, who, false)
	c.Click(h, sort, state, false)
	if c.Click(h, sort, state, false); h.panel.Field("state").Sort != grid.Ascending {
		t.Errorf("state, before who, clicked twice more: %+v", h.panel.Field("state"))
	}
	if c.Click(h, sort, state, true); h.panel.Field("state").Sort != grid.Unsorted ||
		h.panel.Field("who").Sort != grid.Ascending {
		t.Errorf("state shift-clicked: %+v, who %+v", h.panel.Field("state"), h.panel.Field("who"))
	}
	c.Click(h, sort, who, false)
	if c.Click(h, sort, who, false); h.panel.Field("who").Sort != grid.Unsorted {
		t.Errorf("who, the last, clicked twice more: %+v", h.panel.Field("who"))
	}
}

// A dim rule stands between the fields that show and the hidden ones, where there is at least one of each, as one
// stands under the groups; a click on it chooses no field.
func TestTheColumnsDialogSetsTheHiddenFieldsApart(t *testing.T) {
	h := listing()
	c := Columns{Open: true}
	rule := func() []int {
		box, _, _ := c.Box(h)
		var at []int
		for k, l := range box {
			if strings.Contains(ansi.Strip(l), "───") && !strings.HasPrefix(ansi.Strip(l), "╭") &&
				!strings.HasPrefix(ansi.Strip(l), "╰") {
				at = append(at, k)
			}
		}
		return at
	}
	if got := rule(); len(got) != 0 {
		t.Fatalf("a rule with nothing hidden and nothing grouped: %v", got)
	}
	c.At = slices.Index(c.Listed(h), "title")
	c.Key(h, press("space")) // title hidden: the rule stands over it
	_, top, _ := c.Box(h)
	if got := rule(); len(got) != 1 || c.Row(h, top+got[0]) != -1 || c.Listed(h)[c.Row(h, top+got[0]+1)] != "title" {
		t.Fatalf("title hidden: rules at %v", got)
	}
	c.At = 0
	c.Key(h, press("g")) // state grouped too: a rule under the groups, and one over the hidden
	if got := rule(); len(got) != 2 {
		t.Errorf("grouped and hidden: rules at %v", got)
	}
}

// The columns dialog names each field's type after the field, dim, under the heading type, between field and sort.
func TestTheColumnsDialogNamesEachFieldsType(t *testing.T) {
	h := listing()
	box, _, _ := Columns{Open: true}.Box(h)
	heads := strings.Fields(strings.Trim(ansi.Strip(box[1]), "│ "))
	if !slices.Equal(heads, []string{"field", "type", "sort", "group"}) {
		t.Errorf("the headings: %q", heads)
	}
	// n, a column of the outlined host's own, is no field of its catalog, so it has no type
	for _, c := range []struct {
		h        *host
		id, want string
	}{{h, "who", "person"}, {h, "title", title.Type}, {outlined(), "n", ""}} {
		line := ansi.Strip(Columns{}.Line(c.h, c.id))
		if got := strings.TrimSpace(string([]rune(line)[2+LabelCells : 2+LabelCells+TypeCells])); got != c.want {
			t.Errorf("%s's type: %q in %q", c.id, got, line)
		}
	}
}
