// A table drawn: the columns its header names, its rows' lines, the header over them, the rule under the header with
// each sort's mark on it, where each column's name stands, and the control line over it all.
package table

import (
	"slices"
	"strconv"
	"strings"

	"github.com/charmbracelet/x/ansi"

	"github.com/empowerite/kit/projects/tui/internal/grid"
	"github.com/empowerite/kit/projects/tui/internal/style"
)

// ShownColumns gives the ids of the columns the panel shows, of columns, in its order, one a cell of its rows, and none
// for a column without a label.
func ShownColumns(p grid.Panel, columns []string) []string {
	var names []string
	for _, name := range p.Arrange(columns) {
		if name == "" || p.Shows(name) {
			names = append(names, name)
		}
	}
	return names
}

// HeaderNames gives the labels of the columns the panel shows, of columns, in its order, as the header stands them
// over the rows, which widen the columns. A sort's mark stands on the rule under the name, not after it, so sorting
// never widens a column.
func HeaderNames(p grid.Panel, columns []string) []string {
	names := ShownColumns(p, columns)
	for k, name := range names {
		names[k] = p.Label(name)
	}
	return names
}

// Drawing is what a table's rows are drawn with, besides the rows: the names its header stands over them, the cells
// they are laid out in, the pane's width, the cells they stand in by under the headings of their groups, which stand
// open, which are selected, by key, and the table's color, which marks them.
type Drawing struct {
	Names  []string
	Width  int
	Pane   int
	Indent int
	Open   grid.Outline
	Picked map[string]bool
	Color  string
}

// Draw draws a table's rows as the grid does, the one under the cursor, chosen, on a band the pane's width; -1 for
// none. A selected row has the grid's mark in the table's color, and a group's heading the mark of how many of its
// rows are, all or some; every line but an aside's stands in by the indent.
func (d Drawing) Draw(rows []grid.Row, chosen int) []Line {
	marked, _ := style.In(d.Color)
	groups := GroupMarks(grid.Every(rows), d.Picked)
	drawn := grid.Draw(rows, d.Names, d.Width, chosen, d.Open.Open,
		func(r grid.Row) string {
			switch {
			case grid.Heads(r) && groups[r.Key] != "":
				return marked(groups[r.Key])
			case d.Picked[r.Key] && !r.Aside:
				return marked(grid.Chosen)
			}
			return ""
		})
	out := make([]Line, len(drawn))
	indent := strings.Repeat(" ", d.Indent)
	for k, l := range drawn {
		text := l.Text
		if !rows[l.Row].Aside {
			text = indent + text
		}
		if l.Row == chosen {
			// the band runs the pane's width, over the indent and past a heading's end, as the grid lays it out
			// narrower
			text = style.Band(text, d.Pane)
		}
		out[k] = Line{Row: l.Row, Text: text}
	}
	return out
}

// Headed gives the names of the columns the header stands over the rows, as internal/grid lays them out, open or
// closed as open says, each line in by the indent, and whether they hide lines while closed; none for a table whose
// rows have no columns.
func Headed(rows []grid.Row, names []string, width, indent int, open bool) (lines []string, hides bool) {
	lines, hides = grid.Header(rows, names, width, open)
	for k := range lines {
		lines[k] = strings.Repeat(" ", indent) + lines[k]
	}
	return lines, hides
}

// indent gives the cells the host's rows stand in by under the headings of their groups: the pane's, past the grid's.
func indent(h Host) int {
	_, width := h.Pane()
	return width - h.GridWidth()
}

// Rule gives the line under the header, as wide as the pane: blank up to the first column, past the gutter and the
// rows' indent, then a thin dim line, each sort's mark on it under its column's name, as the spots place them, where
// the line reaches; and, where there is no control line to say it, how many rows are selected at its right end, in
// the accent, when some are.
func Rule(h Host, spots []SortSpot) string {
	_, width := h.Pane()
	right := ""
	if n := len(SelectedRows(h)); n > 0 && !h.Controls() {
		right = style.Tint(" "+strconv.Itoa(n)+" selected ") + style.Off("─")
	}
	from := indent(h) + len(Lead)
	fill := max(1, width-from-ansi.StringWidth(right))
	return ansi.Truncate(strings.Repeat(" ", from)+RuleFill(spots, from, fill)+right, width, "")
}

// Sortable is the dim mark on the rule under the name of a column that can sort and does not, so the rule says where a
// click sorts; a column that cannot sort has the rule alone.
const Sortable = "·"

// SortSpot is a column's targets on the rule under the header, from the screen column under its name's first cell: a
// column that holds a list has the flatten mark there, dim while a row holds all its values, not once flattened, and a
// click on it flattens or gathers; after it, or from that first cell for another column, the sort's, its mark where it
// sorts, Sortable where it can and does not, and none where it cannot. The sort's target runs to the end of the
// name's first line, or of the marks, whichever is wider.
type SortSpot struct {
	Column, Mark, Flat string
	Sorts, Flattened   bool
	At, Until          int
}

// SortFrom gives the screen column the sort's mark and target start on: after the flatten mark, where there is one.
func (s SortSpot) SortFrom() int { return s.At + ansi.StringWidth(s.Flat) }

// SortSpots gives the targets of each column the host's header names that can sort or flatten, under its name on the
// rule, wherever the name aligns.
func SortSpots(h Host) []SortSpot {
	rows, names, width, in := h.Rows(), h.Names(), h.GridWidth(), indent(h)
	p := h.Panel()
	var out []SortSpot
	for j, column := range h.ShownColumns() {
		start, first := grid.HeaderName(rows, names, width, j)
		if start < 0 || !p.Sorts(column) && !p.Lists(column) {
			continue
		}
		s := SortSpot{Column: column, At: in + start}
		if p.Lists(column) {
			s.Flat, s.Flattened = FlattenGlyph, p.Field(column).Flat
		}
		if p.Sorts(column) {
			s.Mark = p.Mark(column)
			if s.Sorts = s.Mark != ""; !s.Sorts {
				s.Mark = Sortable
			}
		}
		s.Until = s.At + max(ansi.StringWidth(first), ansi.StringWidth(s.Flat+s.Mark))
		out = append(out, s)
	}
	return out
}

// RuleFill gives the rule's dim line, fill cells from the screen column from on, each column's marks on it under its
// name, as the spots place them, where the whole of them fits on the line: a sort's not dim, nor a flattened list's;
// Sortable, and the flatten mark of a list not flat, dim.
func RuleFill(spots []SortSpot, from, fill int) string {
	cells := slices.Repeat([]string{"─"}, fill)
	marked := make([]bool, fill)
	for _, s := range spots {
		at := s.At - from
		if at < 0 || at+ansi.StringWidth(s.Flat+s.Mark) > fill {
			continue
		}
		for k, g := range strings.Split(s.Flat, "") {
			cells[at+k], marked[at+k] = g, s.Flattened
		}
		at = s.SortFrom() - from
		for k, g := range strings.Split(s.Mark, "") {
			cells[at+k], marked[at+k] = g, s.Sorts
		}
	}
	var out strings.Builder
	for k := 0; k < fill; {
		end := k
		for end < fill && marked[end] == marked[k] {
			end++
		}
		run := strings.Join(cells[k:end], "")
		if !marked[k] {
			run = style.Off(run)
		}
		out.WriteString(run)
		k = end
	}
	return out.String()
}

// SortAt gives the column whose target a click at the screen column x on the rule falls on, as the spots give them,
// short of the rule's right end, where the selected count may stand, and whether it is the flatten mark, which
// flattens or gathers, rather than the sort's, which turns it. Where a column does not sort yet, its target is the
// rule's line, and still sorts it; a list not flat has the flatten mark alone. Nothing elsewhere.
func SortAt(rule string, spots []SortSpot, x int) (column string, flatten bool) {
	line := ansi.Strip(rule)
	if x >= ansi.StringWidth(line) || ansi.Cut(line, x, x+1) == " " {
		return "", false
	}
	for _, s := range spots {
		switch {
		case s.Flat != "" && x >= s.At && x < s.SortFrom():
			return s.Column, true
		case s.Mark != "" && x >= s.SortFrom() && x < s.Until:
			return s.Column, false
		}
	}
	return "", false
}

// SortBy turns a column's sort, as a click on its target on the rule asks, a step each click, as the panel's Sorted
// says: ascending where it does not sort yet, after the other keys; descending where it sorts ascending, in its place;
// then off on the last key, and ascending again on any other. With shift, the click turns it off, wherever it stands.
func SortBy(h Host, column string, shift bool) {
	p := h.Panel()
	if shift {
		h.SetPanel(p.SortedAs(column, grid.Unsorted))
		return
	}
	h.SetPanel(p.Sorted(column))
}

// ColumnName gives the screen column a column's name starts on in the host's header, and its first line, as the header
// draws it.
func ColumnName(h Host, column string) (int, string) {
	j := slices.Index(h.ShownColumns(), column)
	x, name := grid.HeaderName(h.Rows(), h.Names(), h.GridWidth(), j)
	return x + indent(h), name
}

// HeaderColumn gives the column whose name a click at x, y in the host's header falls on, none off every name.
func HeaderColumn(h Host, x, y int) string {
	if k, ok := HeaderLine(h, y); !ok || OnRule(h, y) || k < 0 {
		return ""
	}
	j := grid.HeaderAt(h.Rows(), h.Names(), h.GridWidth(), x-indent(h))
	if j < 0 {
		return ""
	}
	return h.ShownColumns()[j]
}

// FieldName gives the screen column and line a column's name stands on in the heading over the groups, on the control
// line, the pane's first; a line of -1 where the heading no longer names it.
func FieldName(h Host, column string) (x, y int) {
	for _, n := range BarNames(h) {
		if n.Column == column {
			top, _ := h.Pane()
			return n.From, top
		}
	}
	return 0, -1
}

// HeaderHit gives what a click at x, y falls on in the host's header: whether it falls on the header, its rule apart;
// whether on its fold, the gutter of its first line, where the header, closed, hides lines; and the column whose name
// it falls on, none off every name.
func HeaderHit(h Host, x, y int) (on, fold bool, column string) {
	x -= indent(h)
	k, ok := HeaderLine(h, y)
	if !ok || OnRule(h, y) {
		return false, false, ""
	}
	_, hides := h.Header()
	fold = hides && k == 0 && x == FoldCell
	if j := grid.HeaderAt(h.Rows(), h.Names(), h.GridWidth(), x); j >= 0 {
		column = h.ShownColumns()[j]
	}
	return true, fold, column
}

// ToggleHeader opens the host's header, or closes it, as a click on its fold does.
func ToggleHeader(h Host) {
	open := h.Table().Open
	next := grid.Outline{}
	for k, v := range open {
		next[k] = v
	}
	next[HeaderKey] = !open[HeaderKey]
	Reoutline(h, next)
}

// RowAt gives what a click at x, y falls on in the host's rows: the row whose line it falls on, and whether it falls
// on one; the link of a glyph on the row's first line it falls on, none off every link; and whether it falls on the
// ▸ or ▾ of that first line, which opens or closes the row, or the heading.
func RowAt(h Host, x, y int) (row int, link string, fold, ok bool) {
	lines := h.Lines()
	k, on := PlaceAt(h, y)
	if !on || k < 0 || lines[k].Row < 0 {
		return 0, "", false, false
	}
	rows := h.Rows()
	row = lines[k].Row
	if first := k == 0 || lines[k-1].Row != row; first {
		if !rows[row].Aside {
			x -= indent(h)
		}
		link = grid.LinkAt(rows, h.Names(), row, h.GridWidth(), x)
		fold = link == "" && slices.Contains(FoldsAt(rows[row]), x)
	}
	return row, link, fold, true
}

// Control gives the control line, the pane's first, as wide as the pane, and the screen column the preset's selector
// starts on, -1 where there is none: the line's left part, as the heading over the groups draws it; and at its right
// end, how many rows are selected, in the accent, when some are, then, where the tail is the preset line's selector, a
// chip per filter the panel sets, as Chips gives them, and the selector. The chips are clipped first where the line
// is short, then the left part.
func Control(h Host, left, tail string) (string, int) {
	_, width := h.Pane()
	right := ""
	if n := len(SelectedRows(h)); n > 0 {
		right = style.Tint(strconv.Itoa(n)+" selected") + "  "
	}
	if tail != "" {
		if chips := Chips(h); len(chips) > 0 {
			// two spaces either side of the chips
			room := width - ansi.StringWidth(left) - 2 - ansi.StringWidth(right) - 2 - ansi.StringWidth(tail)
			if room > 1 {
				right += ansi.Truncate(strings.Join(chips, "  "), room, "…") + "  "
			}
		}
	}
	room := max(0, width-ansi.StringWidth(right)-ansi.StringWidth(tail)-1)
	left = ansi.Truncate(left, room, "…")
	fill := max(1, width-ansi.StringWidth(left)-ansi.StringWidth(right)-ansi.StringWidth(tail))
	line := ansi.Truncate(left+strings.Repeat(" ", fill)+right+tail, width, "")
	if tail == "" {
		return line, -1
	}
	return line, ansi.StringWidth(left) + fill + ansi.StringWidth(right)
}

// FocusStop draws the stop the keyboard stands on in the control line's left part, as the line under the keyboard in
// a menu is drawn; the rest as it is.
func FocusStop(left string, stop Stop) string {
	return ansi.Cut(left, 0, stop.From) + style.Selected(ansi.Cut(left, stop.From, stop.Until)) +
		ansi.Cut(left, stop.Until, ansi.StringWidth(left))
}
