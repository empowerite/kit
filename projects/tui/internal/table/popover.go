// A field's popover: a small box over the screen that sets, for one field of the table, which of its values the rows
// must hold, and how it is drawn; how they sort is the column menu's, and the header's. It has the keyboard while it is
// open: up and down move between its lines, space or enter acts on the one under the keyboard, and esc closes it; a
// click on a line acts on it, and a click off the box closes it. What it sets applies at once, so the rows, the
// columns dialog, and the preset line's chips follow it.
package table

import (
	"slices"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/empowerite/kit/projects/tui/internal/grid"
	"github.com/empowerite/kit/projects/tui/internal/shell"
	"github.com/empowerite/kit/projects/tui/internal/style"
)

// Popover is a field's popover while it is open: the field, by id; its line under the keyboard; and the screen cell it
// hangs from, the columns dialog's corner, whose right edge it keeps, or its name in the header, which its title
// covers.
type Popover struct {
	Open   bool
	Field  string
	At     int
	X, Y   int
	Header bool
}

// PopLine is one line of a popover: what it draws, and what space does on it, nil for a heading or a gap. Picked says
// a choice is the one taken, which space on it again leaves as it is; a value's box toggles, picked or not.
type PopLine struct {
	Text   string
	Act    func(grid.Panel) grid.Panel
	Picked bool
	Box    bool
}

// OpenPopover gives a field's popover, open, hanging from the screen cell x, y, the first line that acts under the
// keyboard; from the header, its title covers the name there, else it opens beside the cell.
func OpenPopover(h Host, id string, x, y int, header bool) Popover {
	p := Popover{Open: true, Field: id, X: x, Y: y, Header: header, At: -1}
	p.At = p.Move(h, -1, 1)
	return p
}

// Lines gives the popover's lines for its field, under two dim headings, each where the field has it, a blank line
// between them. filtering, for an
// enum, a person, or a label: whether the values checked are let through or held back, include or exclude, one line
// that toggles, then a box per value, as FilterValues lists them, an enum's known values with their glyphs, a label's
// names as its pills draw them, and @me with the caller's login once known. rendering: auto, with the rendering it
// chooses now, dim, then each of the field's renderings, a radio.
func (p Popover) Lines(h Host) []PopLine {
	f, ok := field(h, p.Field)
	if !ok {
		return nil
	}
	id, g := f.ID, h.Panel().Field(f.ID)
	var out []PopLine
	heading := func(text string) { out = append(out, PopLine{Text: style.Off(text)}) }
	radio := func(text string, picked bool, act func(grid.Panel) grid.Panel) {
		glyph := style.Off(grid.Unpicked)
		if picked {
			glyph = style.Tint(grid.Picked)
		}
		out = append(out, PopLine{Text: glyph + " " + text, Act: act, Picked: picked})
	}
	if f.Filter && Types[f.Type].Filters {
		heading("filtering")
		mode := style.Tint(grid.Picked) + " include  " + style.Off(grid.Unpicked) + " exclude"
		if g.Filter.Exclude {
			mode = style.Off(grid.Unpicked) + " include  " + style.Tint(grid.Picked) + " exclude"
		}
		out = append(out, PopLine{Text: mode, Box: true, Act: func(p grid.Panel) grid.Panel {
			f := p.Field(id).Filter
			return p.FilteredBy(id, grid.Filter{Values: f.Values, Exclude: !f.Exclude})
		}})
		for _, v := range FilterValues(h, f) {
			name := v
			switch {
			case f.Type == "enum" && v != grid.NoValue && f.GlyphOf(v) != "":
				name = f.GlyphOf(v) + " " + v
			case f.Type == "label" && v != grid.NoValue:
				name = Pill(h.Label(f.ID, v))
			case v == grid.Caller && h.Me() != "":
				name += " " + style.Off(h.Me())
			}
			box := style.Off(grid.Unticked)
			if slices.Contains(g.Filter.Values, v) {
				box = style.Tint(grid.Ticked)
			}
			out = append(out, PopLine{Text: box + " " + name, Box: true, Act: func(p grid.Panel) grid.Panel {
				filter := p.Field(id).Filter
				if k := slices.Index(filter.Values, v); k >= 0 {
					filter.Values = slices.Delete(slices.Clone(filter.Values), k, k+1)
				} else {
					filter.Values = append(slices.Clone(filter.Values), v)
				}
				return p.FilteredBy(id, filter)
			}})
		}
	}
	if len(out) > 0 {
		out = append(out, PopLine{})
	}
	heading("rendering")
	for _, r := range append([]string{""}, f.Renderings...) {
		name := r
		if r == "" {
			name = "auto " + style.Off(h.Drawn(id))
		}
		radio(name, g.Rendering == r, func(p grid.Panel) grid.Panel { return p.Rendered(id, r) })
	}
	return out
}

// FilterValues gives the values a field's filter offers, as the popover lists them: @me for a person, an enum's known
// values, those the rows hold that are not known, sorted, none, a row with no value, then any the filter names that
// are none of those.
func FilterValues(h Host, f Field) []string {
	var out, seen []string
	if f.Type == "person" {
		out = append(out, grid.Caller)
	}
	for _, e := range f.Values {
		out = append(out, e.Value)
	}
	for _, v := range h.Values(f.ID) {
		for _, raw := range v.Raw {
			if raw != "" && !slices.Contains(out, raw) && !slices.Contains(seen, raw) {
				seen = append(seen, raw)
			}
		}
	}
	slices.Sort(seen)
	out = append(append(out, seen...), grid.NoValue)
	for _, v := range h.Panel().Field(f.ID).Filter.Values {
		if !slices.Contains(out, v) {
			out = append(out, v)
		}
	}
	return out
}

// Move gives the line of the popover from k on, by step, that acts, stopping at the first and the last; k itself when
// none does.
func (p Popover) Move(h Host, k, step int) int {
	lines := p.Lines(h)
	return shell.Step(k, len(lines), step, func(j int) bool { return lines[j].Act != nil })
}

// Acts says whether space on the popover's line under the keyboard changes anything: a value's box and the filter's
// mode always do, and a choice not taken.
func (p Popover) Acts(h Host) bool {
	lines := p.Lines(h)
	if p.At < 0 || p.At >= len(lines) {
		return false
	}
	l := lines[p.At]
	return l.Act != nil && (l.Box || !l.Picked)
}

// Keys gives the popover's keys, on the keys bar, which it owns: esc closes it; up and down move, where there is a
// line that acts that way; and space or enter toggles a value's box, or takes a choice not taken.
func (p Popover) Keys(h Host) shell.Segment {
	s := shell.Segment{Owns: true, Bindings: []shell.Binding{shell.Bind("esc", "close")}}
	var moves []string
	if p.Move(h, p.At, -1) != p.At {
		moves = append(moves, "up")
	}
	if p.Move(h, p.At, 1) != p.At {
		moves = append(moves, "down")
	}
	if len(moves) > 0 {
		s.Bindings = append(s.Bindings, shell.Bind("↑↓", "move", moves...))
	}
	if p.Acts(h) {
		does := "choose"
		if p.Lines(h)[p.At].Box {
			does = "toggle"
		}
		s.Bindings = append(s.Bindings, shell.Bind("␣", does, "space", "enter"))
	}
	return s
}

// Act acts on the popover's line k: it is then under the keyboard, and the table's panel changes as the line says, at
// once.
func (p *Popover) Act(h Host, k int) {
	lines := p.Lines(h)
	if k < 0 || k >= len(lines) || lines[k].Act == nil {
		return
	}
	p.At = k
	h.SetPanel(lines[k].Act(h.Panel()))
}

// Key takes a key while the popover is open, as the popover's comment says; every other key does nothing.
func (p *Popover) Key(h Host, k tea.KeyPressMsg) {
	switch k.String() {
	case "esc":
		*p = Popover{}
	case "up":
		p.At = p.Move(h, p.At, -1)
	case "down":
		p.At = p.Move(h, p.At, 1)
	case "space", "enter":
		if p.Acts(h) {
			p.Act(h, p.At)
		}
	}
}

// Box gives the popover's lines, a dialog titled by the field's label, its keys on the keys bar alone, and the screen
// line and column it is drawn from: from the header, its title over the name, or nearer the screen's left where it
// would run past the edge; from the columns dialog, its right edge at the screen's, its top on the dialog's, or higher
// where it would run past the keys bar.
func (p Popover) Box(h Host) (box []string, top, left int) {
	var body []string
	for k, l := range p.Lines(h) {
		text := l.Text
		if k == p.At {
			text = style.Selected(text)
		}
		body = append(body, text)
	}
	width, height := h.Size()
	box = shell.Dialog(labelOf(h, p.Field), body, width)
	w := ansi.StringWidth(box[0])
	if p.Header {
		// the title on its top edge over the name, where the header draws it, after the corner, a rule, and a space
		top, left = p.Y, min(p.X-3, width-w)
	} else {
		top, left = p.Y, width-w
	}
	return box, max(1, min(top, height-1-len(box))), max(0, left)
}

// Click takes a click while the popover is open: on one of its lines that acts, it acts on it, as space does; on the
// box elsewhere, nothing; off the box, it closes it.
func (p *Popover) Click(h Host, x, y int) {
	box, top, left := p.Box(h)
	switch line, on := (shell.Placed{Lines: box, Top: top, Left: left}).Hit(x, y); {
	case !on:
		*p = Popover{}
	case line >= 0:
		p.Act(h, line)
	}
}
