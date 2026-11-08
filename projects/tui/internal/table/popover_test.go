// A field's popover filters its rows by the values checked, and pins how it is drawn.
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

// texts gives the popover's lines as they read.
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
	want := []string{"filter", grid.Picked + " include  " + grid.Unpicked + " exclude", grid.Unticked + " 🟢 open",
		grid.Unticked + " 🔴 closed", grid.Unticked + " ❔ merged", grid.Unticked + " " + grid.NoValue, "rendering",
		grid.Picked + " auto glyph+text", grid.Unpicked + " glyph+text", grid.Unpicked + " text", grid.Unpicked + " glyph"}
	if got := texts(p, h); !slices.Equal(got, want) || p.At != 1 {
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
	if _, top, left := beside.Box(h); left+ansi.StringWidth(box[0]) != h.w || top+len(box) > h.h-1 {
		t.Errorf("beside: %d, %d", top, left)
	}
	// a click on a line acts on it; on the edge, nothing; off the box, it closes
	p.Click(h, left+2, top+3)
	if f := h.panel.Field("state").Filter; !slices.Equal(f.Values, []string{"open"}) {
		t.Errorf("a click on open: %+v", f)
	}
	if p.Click(h, left, top); !p.Open {
		t.Error("a click on the edge closed it")
	}
	if p.Click(h, 0, 0); p.Open {
		t.Error("a click off the box left it open")
	}
	if labelOf(h, "nonesuch") != "nonesuch" {
		t.Error("an id's own label")
	}
}
