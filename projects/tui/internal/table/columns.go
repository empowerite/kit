// The columns dialog: Fields… in the presets' menu opens it under the header's right end, titled by the table's
// fields. It lists every field of the table, those the rows group by first, the outermost first, then the columns
// as the rows stand them, then the hidden, a dim rule between each two of those it has, each a line: whether it shows, its label, its sort, ▲ or ▼ and its place
// among the keys, and its group's depth. What it sets applies at once. Up and down move between the lines; shift with
// either moves the field a place, across into the groups or out of them, or into the hidden; space shows or hides it,
// s turns its sort, as the panel's Sorted says: ascending, as a key after the others, descending, then none on the
// last key and ascending again on any other; and g groups by it, within the groups there are, or no longer; enter
// opens its Options…, and esc closes the dialog. A click on a line's box, its sort, or its group does what space, s,
// or g does, and a shift-click on its sort turns the sort off; one off the dialog closes it.
package table

import (
	"slices"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/empowerite/kit/projects/tui/internal/grid"
	"github.com/empowerite/kit/projects/tui/internal/pane"
	"github.com/empowerite/kit/projects/tui/internal/shell"
	"github.com/empowerite/kit/projects/tui/internal/style"
)

// Columns is the columns dialog while it is open, and the line under the keyboard, an index into the fields it lists.
type Columns struct {
	Open bool
	At   int
}

// The cells of a line of the columns dialog, after its box and a space: the label, then the sort, then the group.
const (
	LabelCells = 16
	SortCells  = 6
)

// Listed gives the fields the columns dialog lists, as the table's panel lists them.
func (c Columns) Listed(h Host) []string { return h.Panel().Listed() }

// Line draws a field's line in the columns dialog: its box, ticked where it shows, as a column or a group; its
// label, dim where it is hidden, ⊘ after it while a filter on it holds rows back; its sort, ▲ or ▼, and its place among the keys while there are several, or a dim dot;
// and its group's depth, or a dim dot, or nothing where it cannot group.
func (c Columns) Line(h Host, id string) string {
	p := h.Panel()
	f := p.Field(id)
	box, label := style.Mark(grid.Ticked), p.Label(id)
	if f.Placement == grid.Hidden {
		box, label = style.Off(grid.Unticked), style.Off(label)
	}
	if len(f.Filter.Values) > 0 {
		label += " " + style.Tint(grid.Filtered)
	}
	sort := style.Off("·")
	if mark := p.Mark(id); mark != "" {
		sort = style.Tint(mark)
	} else if !p.Sorts(id) {
		sort = " "
	}
	group := style.Off("·")
	switch {
	case f.Placement == grid.Group:
		group = style.Tint(grid.GroupMark(f.Depth))
	case !p.Groupable[id]:
		group = " "
	}
	return box + " " + pane.Pad(label, LabelCells, false) + pane.Pad(sort, SortCells, false) + group
}

// rules gives the fields of the columns dialog a dim rule stands before, as indexes into those it lists: the first
// column, under the groups, where there are both; and the first hidden field, under those that show, where there are
// both, one rule where the two stand together.
func (c Columns) rules(h Host) []int {
	p, listed := h.Panel(), c.Listed(h)
	var out []int
	if groups := len(p.Groups()); groups > 0 && groups < len(listed) {
		out = append(out, groups)
	}
	if hidden := slices.IndexFunc(listed, func(id string) bool { return p.Field(id).Placement == grid.Hidden }); hidden > 0 &&
		!slices.Contains(out, hidden) {
		out = append(out, hidden)
	}
	return out
}

// Box gives the columns dialog's lines, titled by the table's fields, as it lists the groups and the hidden
// too, the headings of its cells dim over its fields, a rule under the groups and one over the hidden, as rules says;
// the line under the keyboard on the
// menu's color; and the screen line and column it is drawn from: under the header's right end, its right edge at the
// pane's.
func (c Columns) Box(h Host) (lines []string, top, left int) {
	body := []string{style.Off("  " + pane.Pad("field", LabelCells, false) + pane.Pad("sort", SortCells, false) + "group")}
	rules := c.rules(h)
	for k, id := range c.Listed(h) {
		if slices.Contains(rules, k) {
			body = append(body, style.Off(strings.Repeat("─", 2+LabelCells+SortCells+5)))
		}
		line := c.Line(h, id)
		if k == c.At {
			line = style.Selected(pane.Pad(ansi.Strip(line), 2+LabelCells+SortCells+5, false))
		}
		body = append(body, line)
	}
	top, width := h.Pane()
	lines = shell.Dialog(h.Name()+" fields", body, width)
	return lines, top, max(0, width-ansi.StringWidth(lines[0]))
}

// Row gives the field a screen line of the columns dialog draws, as an index into the fields it lists, or -1 for
// its edges, its headings, and its rules.
func (c Columns) Row(h Host, y int) int {
	_, top, _ := c.Box(h)
	line, rules := y-top-2, c.rules(h) // past the top edge and the headings
	for k := range c.Listed(h) {
		if slices.Contains(rules, k) {
			if line == 0 {
				return -1
			}
			line--
		}
		if line == 0 {
			return k
		}
		line--
	}
	return -1
}

// Act does to the field under the keyboard what a key asks: space shows it or hides it; s turns its sort; g
// groups by it, or no longer; shift with up or down moves it a place, the keyboard with it.
func (c *Columns) Act(h Host, key string) {
	id, p := c.Listed(h)[c.At], h.Panel()
	switch key {
	case "space":
		if p.Field(id).Placement == grid.Hidden {
			p = p.Placed(id, grid.Column)
		} else {
			p = p.Placed(id, grid.Hidden)
		}
	case "s":
		if !p.Sorts(id) {
			return
		}
		p = p.Sorted(id)
	case "g":
		p = p.Grouped(id, true)
	case "shift+up", "shift+down":
		step := 1
		if key == "shift+up" {
			step = -1
		}
		p, _ = p.Step(id, step)
	}
	h.SetPanel(p)
	// the keyboard stays on the field, wherever it stands now
	if k := slices.Index(c.Listed(h), id); k >= 0 {
		c.At = k
	}
}

// Key takes a key while the columns dialog is open: up and down move between its lines; space, s, g, and shift with up
// or down act on the field under the keyboard, as Act says; enter gives the field's popover to open, over the dialog,
// and whether it does; and esc closes the dialog.
func (c *Columns) Key(h Host, k tea.KeyPressMsg) (Popover, bool) {
	switch key := k.String(); key {
	case "up", "down":
		step := map[string]int{"up": -1, "down": 1}[key]
		c.At = shell.Step(c.At, len(c.Listed(h)), step, shell.Every)
	case "esc":
		*c = Columns{}
	case "enter":
		_, top, left := c.Box(h)
		return OpenPopover(h, c.Listed(h)[c.At], left, top, false), true
	case "space", "s", "g", "shift+up", "shift+down":
		c.Act(h, key)
	}
	return Popover{}, false
}

// Click takes a click while the columns dialog is open, shift held or not: on a field's box, its sort, or its group,
// it acts as space, s, or g does, the keyboard on the field, but that a shift-click on its sort turns the sort off;
// on its label, the keyboard goes there; on the dialog elsewhere, its edges among it, nothing; off it, the dialog
// closes.
func (c *Columns) Click(h Host, x, y int, shift bool) {
	box, top, left := c.Box(h)
	line, on := (shell.Placed{Lines: box, Top: top, Left: left}).Hit(x, y)
	if !on {
		*c = Columns{}
		return
	}
	k := c.Row(h, y)
	if line < 0 || k < 0 {
		return
	}
	c.At = k
	switch cell := x - left - 2; {
	case cell < 2:
		c.Act(h, "space")
	case cell >= 2+LabelCells+SortCells:
		c.Act(h, "g")
	case cell >= 2+LabelCells && shift:
		h.SetPanel(h.Panel().SortedAs(c.Listed(h)[k], grid.Unsorted))
	case cell >= 2+LabelCells:
		c.Act(h, "s")
	}
}

// Keys gives the keys bar's segment while the columns dialog is open, each key where it acts on the field under
// the keyboard: up and down where there is a line that way, shift with either where the field moves that way, s where
// it sorts, g where it groups, and enter where it has Options….
func (c Columns) Keys(h Host) shell.Segment {
	s := shell.Segment{Owns: true, Bindings: []shell.Binding{shell.Bind("esc", "close")}}
	id, p := c.Listed(h)[c.At], h.Panel()
	var moves, orders []string
	for _, step := range []struct {
		key  string
		by   int
		more bool
	}{{"up", -1, c.At > 0}, {"down", 1, c.At < len(c.Listed(h))-1}} {
		if step.more {
			moves = append(moves, step.key)
		}
		if _, ok := p.Step(id, step.by); ok {
			orders = append(orders, "shift+"+step.key)
		}
	}
	if len(moves) > 0 {
		s.Bindings = append(s.Bindings, shell.Bind("↑↓", "move", moves...))
	}
	if len(orders) > 0 {
		s.Bindings = append(s.Bindings, shell.Bind("⇧↑↓", "order", orders...))
	}
	s.Bindings = append(s.Bindings, shell.Bind("␣", "show/hide", "space"))
	if p.Sorts(id) {
		s.Bindings = append(s.Bindings, shell.Bind("s", "sort"))
	}
	if p.Groupable[id] {
		s.Bindings = append(s.Bindings, shell.Bind("g", "group"))
	}
	if len((Popover{Field: id}).Lines(h)) > 0 {
		s.Bindings = append(s.Bindings, shell.Bind("⏎", "options", "enter"))
	}
	return s
}
