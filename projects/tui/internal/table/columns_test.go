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
	case "right":
		return tea.KeyPressMsg{Code: tea.KeyRight}
	case "left":
		return tea.KeyPressMsg{Code: tea.KeyLeft}
	case "tab":
		return tea.KeyPressMsg{Code: tea.KeyTab}
	case "shift+tab":
		return tea.KeyPressMsg{Code: tea.KeyTab, Mod: tea.ModShift}
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
	card, ctop, cleft := c.CardBox(h)
	if top != 4 || left+ansi.StringWidth(box[0]) != cleft || cleft+ansi.StringWidth(card[0]) != h.w-1 || ctop != top ||
		!strings.Contains(ansi.Strip(box[0]), "things fields") || !strings.Contains(ansi.Strip(card[0]), "state") {
		t.Errorf("the box at %d, %d: %q, the card at %d", top, left, ansi.Strip(box[0]), cleft)
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
	// enter gives the field's card, beside the list, the keyboard; esc gives it back, and again closes the dialog
	if c.Key(h, press("enter")); !c.InCard || c.Card(h).Field != c.Listed(h)[c.At] {
		t.Errorf("enter: %+v", c)
	}
	if c.Key(h, press("esc")); c.InCard || !c.Open {
		t.Error("esc in the card left the dialog")
	}
	if c.Key(h, press("esc")); c.Open {
		t.Error("esc left it open")
	}
}

// The card of the field under the keyboard stands beside the list, as wide as the widest field's, so the dialog stands
// still as the keyboard moves; it has what the list's line does not, the filter and the rendering. Right, enter, or tab
// gives it the keyboard, and left, esc, or shift-tab gives it back; a click on one of its lines acts on it. A field the
// catalog does not know has no card, and the keys that would reach one do nothing.
func TestTheColumnsDialogHoldsTheFieldsCardBesideIt(t *testing.T) {
	h := listing()
	c := Columns{Open: true}
	card, _, cleft := c.CardBox(h)
	for c.At = range c.Listed(h) {
		if _, _, left := c.CardBox(h); left != cleft {
			t.Errorf("%s's card at %d, not %d", c.Listed(h)[c.At], left, cleft)
		}
	}
	c.At = 0
	if text := ansi.Strip(strings.Join(card, "\n")); strings.Contains(text, "sort") || strings.Contains(text, ShowIt) ||
		!strings.Contains(text, "filtering") || !strings.Contains(text, "rendering") {
		t.Errorf("state's card beside the list:\n%s", text)
	}
	// right gives the card the keyboard, on its filter's mode; down and space check open; left gives it back
	if c.Key(h, press("right")); !c.InCard || !c.Card(h).Lines(h)[c.CardAt].Mode ||
		!strings.Contains((shell.Path{c.Keys(h)}).String(), "← fields") {
		t.Fatalf("right: %+v, %q", c, (shell.Path{c.Keys(h)}).String())
	}
	c.Key(h, key("down"))
	c.Key(h, key("space"))
	if f := h.panel.Field("state").Filter; !slices.Equal(f.Values, []string{"open"}) ||
		!strings.Contains(ansi.Strip(c.Line(h, "state")), grid.Filtered) {
		t.Errorf("open checked in the card: %+v", f)
	}
	if c.Key(h, press("left")); c.InCard {
		t.Error("left in the card")
	}
	if c.Key(h, press("tab")); !c.InCard {
		t.Error("tab")
	}
	c.Key(h, press("shift+tab"))
	// a click on the card's open unchecks it, and gives the card the keyboard; one on its edge does nothing
	card, ctop, cleft := c.CardBox(h)
	if c.Click(h, cleft+3, ctop+3, false); len(h.panel.Field("state").Filter.Values) != 0 || !c.InCard || c.CardAt != 2 {
		t.Errorf("a click on the card's open: %+v, %+v", h.panel.Field("state").Filter, c)
	}
	if c.Click(h, cleft, ctop+3, false); !c.Open || len(h.panel.Field("state").Filter.Values) != 0 {
		t.Error("a click on the card's edge")
	}
	heading := slices.Index(texts(c.Card(h), h), "rendering")
	if c.Click(h, cleft+3, ctop+1+heading, false); c.CardAt != 2 || len(card) == 0 {
		t.Errorf("a click on the card's heading moved the keyboard: %d", c.CardAt)
	}
	// a field the catalog does not know has no card: right does nothing, nor does a click on its label
	h.panel.Columns = append(h.panel.Columns, "extra")
	c = Columns{Open: true, At: slices.Index(h.panel.Listed(), "extra")}
	_, top, left := c.Box(h)
	if card, _, _ := c.CardBox(h); card != nil {
		t.Error("a card for a field the catalog does not know")
	}
	if c.Key(h, press("right")); c.InCard {
		t.Error("right into no card")
	}
	if c.Click(h, left+6, top+2+c.At, false); c.InCard {
		t.Error("a click on extra's label into no card")
	}
	if (Columns{At: 99}).Card(h).Open {
		t.Error("a card past the list")
	}
	// OpenColumns opens the dialog on a field, its card with the keyboard; on the first for a field it does not list
	if o := OpenColumns(h, "who"); !o.Open || o.Listed(h)[o.At] != "who" || !o.InCard || o.CardAt < 0 {
		t.Errorf("open on who: %+v", o)
	}
	if o := OpenColumns(h, "nonesuch"); o.At != 0 {
		t.Errorf("open on no field: %+v", o)
	}
	h.fields, h.panel.Columns = nil, []string{"extra"}
	if (Columns{}).cardInner(h) != 0 {
		t.Error("a card's width with no card")
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
	if c.Click(h, left+6, y, false); c.At != 1 || !c.InCard {
		t.Errorf("a click on a label: %+v", c)
	}
	if c.Click(h, left+3, top+2, false); c.InCard {
		t.Error("a click on a box left the card the keyboard")
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
