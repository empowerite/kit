// A field's card: a small box over the screen that holds every setting of one field of the table: how the rows sort
// by it, whether they group by it, whether a list it holds stands a row per value, which of its values the rows must
// hold, how it is drawn, and whether it shows. A click on a column's name opens it under the name, as the keyboard does
// on a name in the heading over the groups, and shift with left or right moves the column a place, the card with it; a
// click on a filter's chip on the control line opens it under the chip, on the filter; a click on a field in the
// columns dialog opens it beside the dialog. It has the keyboard while it is open: up and down
// move between its lines, space or enter acts on the one under the keyboard, and esc closes it; a click on a line acts
// on it, and a click off the box closes it. What it sets applies at once, so the rows, the header, the columns dialog,
// and the preset line's chips follow it.
package table

import (
	"cmp"
	"slices"
	"strconv"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/empowerite/kit/projects/tui/internal/grid"
	"github.com/empowerite/kit/projects/tui/internal/pane"
	"github.com/empowerite/kit/projects/tui/internal/shell"
	"github.com/empowerite/kit/projects/tui/internal/style"
)

// Popover is a field's card while it is open: the field, by id; its line under the keyboard; and the screen cell it
// hangs from, the columns dialog's corner, whose right edge it keeps, or its column's name in the header, or, as
// Heading says, in the heading over the groups, which its title covers, and which it follows as the column moves; or,
// as Chip says, its filter's chip on the control line, which its title covers.
type Popover struct {
	Open    bool
	Field   string
	At      int
	X, Y    int
	Header  bool
	Heading bool
	Chip    bool
	Typed   string
}

// hangs says whether the card hangs from what was clicked, its title over it, rather than beside the columns dialog.
func (p Popover) hangs() bool { return p.Header || p.Heading || p.Chip }

// AllFields stands on the top edge of a card that hangs from what was clicked, at its right: a click on it, or tab,
// widens the card into the columns dialog, on its field.
const AllFields = "all ▸"

// allCells is how many cells AllFields takes on the top edge, between ╴ and ╶, the rule and the corner after it.
var allCells = 1 + ansi.StringWidth(AllFields) + 3

// PopLine is one line of a card: what it draws, and what space does on it, nil for a heading or a gap. Picked says
// a choice is the one taken, which space on it again leaves as it is; a box toggles, picked or not. Input is the line a
// range is typed on, which takes the keys typed while it is under the keyboard. Mode is the filter's include or exclude.
type PopLine struct {
	Text   string
	Act    func(grid.Panel) grid.Panel
	Picked bool
	Box    bool
	Input  bool
	Mode   bool
}

// OpenPopover gives a field's card, open, hanging from the screen cell x, y, the first line that acts under the
// keyboard; from the header, its title covers the name there, else it opens beside the cell.
func OpenPopover(h Host, id string, x, y int, header bool) Popover {
	p := Popover{Open: true, Field: id, X: x, Y: y, Header: header, At: -1}
	p.At = p.Move(h, -1, 1)
	return p
}

// OpenCard gives a column's card, open, hanging from its name in the header or, as heading says, in the heading over
// the groups, the first line that acts under the keyboard.
func OpenCard(h Host, column string, heading bool) Popover {
	p := Popover{Open: true, Field: column, Header: !heading, Heading: heading, At: -1}
	p.follow(h)
	p.At = p.Move(h, -1, 1)
	return p
}

// OpenChip gives the card of the field a chip on the control line filters by, open, hanging from the chip, its title
// over it, the keyboard on the filter's mode.
func OpenChip(h Host, chip ControlSpot) Popover {
	top, _ := h.Pane()
	p := Popover{Open: true, Field: chip.Field, X: chip.From, Y: top, Chip: true}
	return p.AtFilter(h)
}

// AtFilter gives the card with the keyboard on its filter's mode, where its field filters, else as it was.
func (p Popover) AtFilter(h Host) Popover {
	if k := slices.IndexFunc(p.Lines(h), func(l PopLine) bool { return l.Mode }); k >= 0 {
		p.At = k
	}
	return p
}

// follow puts a card that hangs from its column's name where the name now stands, wherever the column has moved; one
// whose name no longer shows, hidden or no longer grouping, stays where it was.
func (p *Popover) follow(h Host) {
	switch {
	case p.Heading:
		if x, y := h.HeadingName(p.Field); y >= 0 {
			p.X, p.Y = x, y
		}
	case p.Header:
		if x, name := h.HeaderName(p.Field); name != "" && x >= 0 {
			p.X, p.Y = x, h.HeaderTop()
		}
	}
}

// Lines gives the card's lines for its field, in sections under dim headings, each where the field has it, a blank
// line between them. sort, for a field that sorts: a radio, Ascending, Descending, or Unsorted, a list
// sorting by size, and ⁿ and ₙ on the triangles where something else orders the rows too. rows: a box that
// groups the rows by it, innermost, or no longer, its depth after it while it groups, for a field that can; and for a
// field that holds a list, a radio, a row per value or all on one row. filtering, for an enum, a person, a label, a
// time, or a number: whether the values checked are let through or held back, include or exclude, one line that
// toggles, then a box per value, as FilterValues lists them, an enum's known values with their glyphs, a label's names
// as its pills draw them, and @me with the caller's login once known, and a line a range is typed on for a time or a
// number. rendering: auto, with the rendering it chooses now, dim, then each of the field's renderings, a radio. Last,
// after a blank line, a box that shows the field, or hides it. Beside the columns dialog, whose lines show a field's
// sort, its grouping, and whether it shows, the card leaves those out, so nothing stands twice.
func (p Popover) Lines(h Host) []PopLine {
	f, ok := field(h, p.Field)
	if !ok {
		return nil
	}
	panel := h.Panel()
	id, g := f.ID, panel.Field(f.ID)
	var out []PopLine
	heading := func(text string) {
		if len(out) > 0 {
			out = append(out, PopLine{})
		}
		out = append(out, PopLine{Text: style.Off(text)})
	}
	radio := func(text string, picked bool, act func(grid.Panel) grid.Panel) {
		glyph := style.Off(grid.Unpicked)
		if picked {
			glyph = style.Tint(grid.Picked)
		}
		out = append(out, PopLine{Text: glyph + " " + text, Act: act, Picked: picked})
	}
	box := func(text string, ticked bool, act func(grid.Panel) grid.Panel) {
		glyph := style.Off(grid.Unticked)
		if ticked {
			glyph = style.Tint(grid.Ticked)
		}
		out = append(out, PopLine{Text: glyph + " " + text, Box: true, Act: act})
	}
	alone := p.hangs()
	if alone && panel.Sorts(id) {
		heading("sort")
		up, down, by := grid.DirectionMark(grid.Ascending, 1), grid.DirectionMark(grid.Descending, 1), ""
		if place, total := panel.Place(id); total-min(place, 1) > 0 {
			up, down = grid.AscendingPlace, grid.DescendingPlace
		}
		if panel.Counts(id) {
			by = BySize
		}
		sorted := func(d grid.Direction) func(grid.Panel) grid.Panel {
			return func(q grid.Panel) grid.Panel { return q.SortedAs(id, d) }
		}
		radio(up+" Ascending"+by, g.Sort == grid.Ascending, sorted(grid.Ascending))
		radio(down+" Descending"+by, g.Sort == grid.Descending, sorted(grid.Descending))
		radio(Unsorted, g.Sort == grid.Unsorted, sorted(grid.Unsorted))
	}
	if alone && panel.Groupable[id] || panel.Lists(id) {
		heading("rows")
	}
	if alone && panel.Groupable[id] {
		text := GroupIt
		if g.Placement == grid.Group {
			text += " " + style.Tint(grid.GroupMark(g.Depth))
		}
		box(text, g.Placement == grid.Group, func(q grid.Panel) grid.Panel { return q.Grouped(id, true) })
	}
	if panel.Lists(id) {
		radio(FlattenIt(panel.Ones[id]), g.Flat, func(q grid.Panel) grid.Panel { return q.Flattened(id, true) })
		radio(GatherIt(cmp.Or(panel.Labels[id], id)), !g.Flat,
			func(q grid.Panel) grid.Panel { return q.Flattened(id, false) })
	}
	if f.Filter && Types[f.Type].Filters {
		heading("filtering")
		mode := style.Tint(grid.Picked) + " include  " + style.Off(grid.Unpicked) + " exclude"
		if g.Filter.Exclude {
			mode = style.Off(grid.Unpicked) + " include  " + style.Tint(grid.Picked) + " exclude"
		}
		out = append(out, PopLine{Text: mode, Box: true, Mode: true, Act: func(p grid.Panel) grid.Panel {
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
			box(name, slices.Contains(g.Filter.Values, v), func(p grid.Panel) grid.Panel {
				filter := p.Field(id).Filter
				if k := slices.Index(filter.Values, v); k >= 0 {
					filter.Values = slices.Delete(slices.Clone(filter.Values), k, k+1)
				} else {
					filter.Values = append(slices.Clone(filter.Values), v)
				}
				return p.FilteredBy(id, filter)
			})
		}
		if Ranged(f) {
			out = append(out, p.inputLine(id))
		}
	}
	heading("rendering")
	for _, r := range append([]string{""}, f.Renderings...) {
		name := r
		if r == "" {
			name = "auto " + style.Off(h.Drawn(id))
		}
		radio(name, g.Rendering == r, func(p grid.Panel) grid.Panel { return p.Rendered(id, r) })
	}
	if !alone {
		return out
	}
	out = append(out, PopLine{})
	box(ShowIt, g.Placement != grid.Hidden, func(q grid.Panel) grid.Panel {
		if q.Field(id).Placement == grid.Hidden {
			return q.Placed(id, grid.Column)
		}
		return q.Placed(id, grid.Hidden)
	})
	return out
}

// FilterValues gives the values a field's filter offers, as the card lists them: @me for a person, an enum's known
// values, those the rows hold that are not known, sorted, none, a row with no value, then any the filter names that
// are none of those.
func FilterValues(h Host, f Field) []string {
	if Ranged(f) {
		return withFiltered(h, f, append(ranges(h, f), grid.NoValue))
	}
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
	return withFiltered(h, f, append(append(out, seen...), grid.NoValue))
}

// withFiltered gives the values a filter offers, then any the field's filter names that are none of them, as a range
// typed.
func withFiltered(h Host, f Field, out []string) []string {
	for _, v := range h.Panel().Field(f.ID).Filter.Values {
		if !slices.Contains(out, v) {
			out = append(out, v)
		}
	}
	return out
}

// Ranged says whether a field's filter chooses among ranges rather than values: a time's, or a number's.
func Ranged(f Field) bool { return f.Type == "time" || f.Type == "number" }

// timeRanges are the ranges a time's filter offers, whatever its rows hold; numberRanges those a number's offers where
// one of its rows' numbers falls in them.
var (
	timeRanges   = []string{"<1m", "<1h", "<1d", "<1w", "<30d", "≥30d"}
	numberRanges = []string{"0", "1", "2-5", "6-20", "21-100", ">100"}
)

// ranges gives the ranges a time's or a number's filter offers, as timeRanges and numberRanges say.
func ranges(h Host, f Field) []string {
	if f.Type == "time" {
		return slices.Clone(timeRanges)
	}
	var out []string
	for _, r := range numberRanges {
		lo, hi, ok := grid.Range(r)
		if !ok {
			lo, hi = 0, 0
			if n, err := strconv.ParseFloat(r, 64); err == nil {
				lo, hi = n, n
			}
		}
		if slices.ContainsFunc(h.Values(f.ID), func(v Value) bool {
			return slices.ContainsFunc(v.Raw, func(raw string) bool {
				n, err := strconv.ParseFloat(raw, 64)
				return err == nil && n >= lo && n <= hi
			})
		}) {
			out = append(out, r)
		}
	}
	return out
}

// inputLine gives the line a range is typed on: what is typed, a cursor after it, or, with nothing typed, what it
// takes, dim; one that reads as no range drawn as a refusal. Enter, or a click, adds what is typed to the filter where
// it reads as a range, >= and <= as ≥ and ≤.
func (p Popover) inputLine(id string) PopLine {
	text := "› " + p.Typed + "▏"
	switch typed := Typed(p.Typed); {
	case p.Typed == "":
		text = style.Off("› type a range: <2h, >=3, 2-5")
	case !isRange(typed):
		text = style.Refusal(text)
	}
	return PopLine{Text: text, Input: true, Act: func(panel grid.Panel) grid.Panel {
		typed := Typed(p.Typed)
		filter := panel.Field(id).Filter
		if !isRange(typed) || slices.Contains(filter.Values, typed) {
			return panel
		}
		filter.Values = append(slices.Clone(filter.Values), typed)
		return panel.FilteredBy(id, filter)
	}}
}

// Typed gives a range as typed, as the filter keeps it: without spaces, >= and <= as ≥ and ≤.
func Typed(s string) string {
	s = strings.ReplaceAll(s, " ", "")
	return strings.NewReplacer(">=", "≥", "<=", "≤").Replace(s)
}

// isRange says whether a range typed reads as one.
func isRange(s string) bool {
	_, _, ok := grid.Range(s)
	return ok
}

// Move gives the line of the card from k on, by step, that acts, stopping at the first and the last; k itself when
// none does.
func (p Popover) Move(h Host, k, step int) int {
	lines := p.Lines(h)
	return shell.Step(k, len(lines), step, func(j int) bool { return lines[j].Act != nil })
}

// Acts says whether space on the card's line under the keyboard changes anything: a box and the filter's
// mode always do, and a choice not taken.
func (p Popover) Acts(h Host) bool {
	lines := p.Lines(h)
	if p.At < 0 || p.At >= len(lines) {
		return false
	}
	l := lines[p.At]
	return l.Act != nil && (l.Box || !l.Picked)
}

// Keys gives the card's keys, on the keys bar, which it owns: esc closes it; up and down move, where there is a line
// that acts that way; shift with left or right moves the column, on a card that hangs from its name, where it moves
// that way; and space or enter toggles a box, or takes a choice not taken, or, on the line a range is typed on, enter
// adds it.
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
	if p.Header || p.Heading {
		s.Bindings = append(s.Bindings, ShiftKeys(h, p.Field)...)
	}
	if p.hangs() {
		s.Bindings = append(s.Bindings, shell.Bind("⇥", "all fields", "tab"))
	}
	if lines := p.Lines(h); p.At >= 0 && p.At < len(lines) && lines[p.At].Input {
		return s.With(shell.Bind("⏎", "add", "enter"))
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

// Act acts on the card's line k: it is then under the keyboard, and the table's panel changes as the line says, at
// once, the card following its column's name wherever that now stands.
func (p *Popover) Act(h Host, k int) {
	lines := p.Lines(h)
	if k < 0 || k >= len(lines) || lines[k].Act == nil {
		return
	}
	p.At = k
	h.SetPanel(lines[k].Act(h.Panel()))
	p.follow(h)
	if lines[k].Input && isRange(Typed(p.Typed)) {
		p.Typed = ""
	}
}

// Key takes a key while the card is open, as the card's comment says: shift with left or right moves a column whose
// name it hangs from a place, as Shifted says, the card with it; tab, on a card that hangs from what was clicked,
// closes it and gives its field, to widen into the columns dialog; every other key does nothing. It gives the field to
// widen into, or nothing.
func (p *Popover) Key(h Host, k tea.KeyPressMsg) string {
	if d, ok := ShiftStep(k.String()); ok && (p.Header || p.Heading) {
		if q, ok := Shifted(h, p.Field, d); ok {
			h.SetPanel(q)
			Reveal(h, p.Field)
			p.follow(h)
		}
		return ""
	}
	if k.String() == "tab" && p.hangs() {
		id := p.Field
		*p = Popover{}
		return id
	}
	lines := p.Lines(h)
	if on := p.At >= 0 && p.At < len(lines) && lines[p.At].Input; on && k.String() == "enter" {
		p.Act(h, p.At)
		return ""
	} else if on && k.String() != "space" {
		if typed, ok := shell.Type(p.Typed, k); ok {
			p.Typed = typed
			return ""
		}
	}
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
	return ""
}

// Box gives the card's lines, a dialog titled by the field's label, its keys on the keys bar alone, and the screen
// line and column it is drawn from: from a column's name or a chip, its title over it, or nearer the screen's left
// where it would run past the edge, AllFields on its top edge at the right; from the columns dialog, its right edge at
// the screen's, its top on the dialog's, or higher where it would run past the keys bar.
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
	label := labelOf(h, p.Field)
	if p.hangs() && len(body) > 0 {
		// room on the top edge for the title, then AllFields: the corner, a rule, a space, the title, a space
		body[0] = pane.Pad(body[0], ansi.StringWidth(label)+4+allCells-2, false)
	}
	box = shell.Dialog(label, body, width)
	w := ansi.StringWidth(box[0])
	if p.hangs() {
		box[0] = ansi.Cut(box[0], 0, w-allCells) + style.Menu("╴") + style.Tint(AllFields) + style.Menu("╶─╮")
		// the title on its top edge over the name, where the header draws it, after the corner, a rule, and a space
		top, left = p.Y, min(p.X-3, width-w)
	} else {
		top, left = p.Y, width-w
	}
	return box, max(1, min(top, height-1-len(box))), max(0, left)
}

// Click takes a click while the card is open: on one of its lines that acts, it acts on it, as space does; on
// AllFields, it closes the card and gives its field, to widen into the columns dialog; off the box, or elsewhere on the
// top edge of a card that hangs from a column's name or a chip, which the edge's title covers, it closes it, as a
// second click on a name closes what the first opened; on the box elsewhere, nothing. It gives the field to widen into,
// or nothing.
func (p *Popover) Click(h Host, x, y int) string {
	box, top, left := p.Box(h)
	w := ansi.StringWidth(box[0])
	switch line, on := (shell.Placed{Lines: box, Top: top, Left: left}).Hit(x, y); {
	case on && p.hangs() && y == top && x >= left+w-allCells && x < left+w-2:
		id := p.Field
		*p = Popover{}
		return id
	case !on, p.hangs() && y == top:
		*p = Popover{}
	case line >= 0:
		p.Act(h, line)
	}
	return ""
}
