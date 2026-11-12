// A field's card filters its rows by the values checked, pins how it is drawn, and shows or hides it.
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

// texts gives the card's lines as they read.
func texts(p Popover, h Host) []string {
	var out []string
	for _, l := range p.Lines(h) {
		out = append(out, ansi.Strip(l.Text))
	}
	return out
}

// key gives the press of a key.
func key(name string) tea.KeyPressMsg {
	switch name {
	case "up":
		return tea.KeyPressMsg{Code: tea.KeyUp}
	case "down":
		return tea.KeyPressMsg{Code: tea.KeyDown}
	case "esc":
		return tea.KeyPressMsg{Code: tea.KeyEscape}
	}
	return tea.KeyPressMsg{Code: tea.KeySpace, Text: " "}
}

func TestAPopoverFiltersByTheValuesChecked(t *testing.T) {
	h := screen()
	p := OpenPopover(h, "state", 10, 3, true)
	want := []string{"filtering", grid.Picked + " include  " + grid.Unpicked + " exclude", grid.Unticked + " 🟢 open",
		grid.Unticked + " 🔴 closed", grid.Unticked + " ❔ merged", grid.Unticked + " " + grid.NoValue, "", "rendering",
		grid.Picked + " auto glyph+text", grid.Unpicked + " glyph+text", grid.Unpicked + " text", grid.Unpicked + " glyph", "", grid.Ticked + " " + ShowIt}
	if got := texts(p, h); !slices.Equal(got, want) || p.At != 1 || p.AtFilter(h).At != 1 ||
		OpenPopover(h, "title", 0, 0, false).AtFilter(h).At != 1 {
		t.Fatalf("the lines, at %d:\n%q", p.At, got)
	}
	// space on a value checks it; on the mode, it excludes what is checked
	p.Key(h, key("down"))
	p.Key(h, key("space"))
	if f := h.panel.Field("state").Filter; !slices.Equal(f.Values, []string{"open"}) || f.Exclude || p.At != 2 {
		t.Fatalf("open checked: %+v, at %d", f, p.At)
	}
	p.Key(h, key("up"))
	p.Key(h, key("space"))
	if f := h.panel.Field("state").Filter; !f.Exclude {
		t.Errorf("exclude: %+v", f)
	}
	if got := texts(p, h)[1]; got != grid.Unpicked+" include  "+grid.Picked+" exclude" {
		t.Errorf("the mode: %q", got)
	}
	// space on open again unchecks it
	p.Key(h, key("down"))
	p.Key(h, key("space"))
	if f := h.panel.Field("state").Filter; len(f.Values) != 0 {
		t.Errorf("open unchecked: %+v", f)
	}
	// a value the filter names that the rows no longer hold is still offered, last
	h.panel = h.panel.FilteredBy("state", grid.Filter{Values: []string{"gone"}})
	if got := FilterValues(h, state); got[len(got)-1] != "gone" || !slices.Contains(texts(p, h), grid.Ticked+" ❔ gone") {
		t.Errorf("a value the rows lost: %q", got)
	}
}

func TestAPopoverPinsARenderingAndKnowsTheCaller(t *testing.T) {
	h := screen()
	h.me = "ada"
	p := OpenPopover(h, "who", 10, 3, false)
	if got := texts(p, h); !slices.Contains(got, grid.Unticked+" "+grid.Caller+" ada") || got[3] != grid.Unticked+" ada" ||
		got[4] != grid.Unticked+" bob" {
		t.Errorf("a person's values: %q", got)
	}
	if got := texts(OpenPopover(h, "tags", 0, 0, false), h); !strings.Contains(got[2], "bug") {
		t.Errorf("a label's values: %q", got)
	}
	// the rendering: a radio, a choice taken again changes nothing
	p = OpenPopover(h, "title", 10, 3, false)
	if got := texts(p, h); !slices.Equal(got, []string{"rendering", grid.Picked + " auto ", grid.Unpicked + " text"}) {
		t.Fatalf("title's lines: %q", got)
	}
	if p.Acts(h) || p.At != 1 {
		t.Errorf("auto, taken, acts: at %d", p.At)
	}
	if keys := (shell.Path{p.Keys(h)}).String(); !strings.Contains(keys, "move") || strings.Contains(keys, "choose") {
		t.Errorf("auto's keys: %q", keys)
	}
	p.Key(h, key("down"))
	if keys := (shell.Path{p.Keys(h)}).String(); !strings.Contains(keys, "choose") {
		t.Errorf("text's keys: %q", keys)
	}
	p.Key(h, key("space"))
	if h.panel.Field("title").Rendering != "text" || h.sets != 1 {
		t.Errorf("text pinned: %+v", h.panel.Field("title"))
	}
	if p.Act(h, 0); h.sets != 1 {
		t.Error("a heading acted")
	}
	if p.Key(h, key("esc")); p.Open {
		t.Error("esc left it open")
	}
	if (Popover{Field: "nonesuch"}).Lines(h) != nil || (Popover{At: 9}).Acts(h) {
		t.Error("a field the table does not know")
	}
	if keys := (shell.Path{OpenPopover(h, "state", 0, 0, false).Keys(h)}).String(); !strings.Contains(keys, "toggle") {
		t.Errorf("a box's keys: %q", keys)
	}
}

func TestAPopoverHangsWhereItWasOpenedAndTakesClicks(t *testing.T) {
	h := screen()
	// from the header, its title over the name; from the dialog, against the screen's right edge
	p := OpenPopover(h, "state", 20, 3, true)
	box, top, left := p.Box(h)
	if top != 3 || left != 17 || !strings.Contains(ansi.Strip(box[0]), "state") {
		t.Errorf("from the header: %d, %d, %q", top, left, ansi.Strip(box[0]))
	}
	beside := OpenPopover(h, "state", 20, 30, false)
	if box, top, left := beside.Box(h); left+ansi.StringWidth(box[0]) != h.w || top+len(box) > h.h-1 {
		t.Errorf("beside: %d, %d", top, left)
	}
	// a click on a line acts on it; on the edge, nothing; on the title over the name, or off the box, it closes
	p.Click(h, left+2, top+3)
	if f := h.panel.Field("state").Filter; !slices.Equal(f.Values, []string{"open"}) {
		t.Errorf("a click on open: %+v", f)
	}
	if p.Click(h, left, top+1); !p.Open {
		t.Error("a click on the edge closed it")
	}
	_, btop, bleft := beside.Box(h)
	if beside.Click(h, bleft+1, btop); !beside.Open {
		t.Error("a click on the top edge of a card beside the dialog closed it")
	}
	if p.Click(h, left+4, top); p.Open {
		t.Error("a click on the title, over the name, left the card open")
	}
	p = OpenPopover(h, "state", 20, 3, true)
	if p.Click(h, 0, 0); p.Open {
		t.Error("a click off the box left it open")
	}
	if labelOf(h, "nonesuch") != "nonesuch" {
		t.Error("an id's own label")
	}
}

// A time's filter offers ranges, fixed: under a minute, an hour, a day, a week, thirty days, and thirty days or more; a
// number's the ranges its rows' numbers fall in, of 0, 1, 2-5, 6-20, 21-100, and more than 100. A line under them
// takes a range typed, <2h, >=3, 2-5, which enter adds to the filter, >= and <= as ≥ and ≤; one that reads as no range
// stays as typed, and adds nothing.
func TestATimeOrANumberFiltersByRanges(t *testing.T) {
	age := Field{ID: "age", Label: "age", Type: "time", Filter: true, Renderings: []string{"age"}}
	lines := Field{ID: "lines", Label: "lines", Type: "number", Filter: true, Renderings: []string{"text"}}
	h := &host{fields: []Field{age, lines, title}, w: 80, h: 24, values: map[string][]Value{
		"age":   {One("30"), One("90000")},
		"lines": {One("0"), One("3"), One("250")},
	}, drawn: map[string]string{"age": "age", "lines": "text"}}
	if got := FilterValues(h, age); !slices.Equal(got, []string{"<1m", "<1h", "<1d", "<1w", "<30d", "≥30d", grid.NoValue}) {
		t.Errorf("a time's ranges: %q", got)
	}
	if got := FilterValues(h, lines); !slices.Equal(got, []string{"0", "2-5", ">100", grid.NoValue}) {
		t.Errorf("a number's ranges: %q", got)
	}
	// <1m checked keeps the row of 30 seconds
	p := OpenPopover(h, "age", 10, 3, true)
	p.Key(h, key("down"))
	p.Key(h, key("space"))
	if f := h.Panel().Field("age").Filter; !slices.Equal(f.Values, []string{"<1m"}) {
		t.Fatalf("down to <1m and space: %+v, at %d %q", f, p.At, texts(p, h))
	}
	// the typed line, after none: >=2h typed, enter adds ≥2h
	for p.Move(h, p.At, 1) != p.At {
		p.Key(h, key("down"))
		if p.Lines(h)[p.At].Input {
			break
		}
	}
	if !p.Lines(h)[p.At].Input {
		t.Fatalf("no typed line: %q", texts(p, h))
	}
	if keys := shellLine(p.Keys(h)); !strings.Contains(keys, "⏎ add") {
		t.Errorf("the typed line's keys: %s", keys)
	}
	for _, r := range "oops" {
		p.Key(h, tea.KeyPressMsg{Code: r, Text: string(r)})
	}
	p.Key(h, tea.KeyPressMsg{Code: tea.KeyEnter})
	if p.Typed != "oops" || len(h.Panel().Field("age").Filter.Values) != 1 {
		t.Errorf("no range typed: %q, %+v", p.Typed, h.Panel().Field("age").Filter)
	}
	p.Typed = ""
	for _, r := range ">=2h" {
		p.Key(h, tea.KeyPressMsg{Code: r, Text: string(r)})
	}
	p.Key(h, tea.KeyPressMsg{Code: tea.KeyEnter})
	if f := h.Panel().Field("age").Filter; !slices.Equal(f.Values, []string{"<1m", "≥2h"}) || p.Typed != "" {
		t.Errorf("≥2h typed: %+v, still typed %q", f, p.Typed)
	}
	if got := texts(p, h); !slices.Contains(got, grid.Ticked+" ≥2h") {
		t.Errorf("the range typed is a box of its own: %q", got)
	}
}
