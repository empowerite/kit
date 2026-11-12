// Package grid is the rows a screen lays out in columns: kit flow's stages draw theirs through it. A row has cells, one
// a column, and the lines it opens to. A cell holds one value, or several, as an issue's assignees, or a text that
// wraps, as a title. A closed row shows a cell's first value, with +N, right-aligned in the cell, when it holds more,
// and its text cut with … where its column ends. An open row shows every value, one a line in its column, the text
// wrapped in its column, then the lines it opens to. A cell with a summary, as an issue's labels by their count, shows
// the summary in its column, closed or open, and an open row's values in it follow the row's lines, as many to a line
// as the width holds. A row that hides anything, closed, has ▸ in the gutter before it,
// and ▾ open; the row under the cursor is drawn on a faint band, its first cell bold in the accent, a pill as it is;
// the band marks the row, so the cell is not underlined as a chosen name is. A glyph, or a first word, whose link
// opens a page is underlined with dots. A header can stand the columns' names over the rows, widening each column to
// its name; a click on a name sorts by it. A row selected, one of several, has ✓ before its ▸. A panel beside the rows narrows them: a
// search, which keeps the rows whose text has it, and a field editor, each field in one of three sections, the groups,
// the columns, and the hidden, moved within its section or across into the next. It keeps, per field, its placement, as
// a column at a position, a group at a depth, or hidden, its sort, and its filter, each group under a heading; while it
// folds, a column whose value is the same on every row of a group groups it on its own, its value said once in the
// heading.
package grid

import (
	"cmp"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"
	"unicode"
	"unicode/utf8"

	"github.com/charmbracelet/x/ansi"

	"github.com/empowerite/kit/projects/tui/internal/pane"
	"github.com/empowerite/kit/projects/tui/internal/style"
)

// Cell is one column of a row: its values, styled already when they are dim, each a line when the row is open; Right is
// a number, right-aligned in its column; Wrap is the one cell of a row that takes the width left, cut while the row is
// closed and wrapped when it is open. Link is where the glyph its text starts with leads, as an issue's page, or its
// first word, where it draws no glyph: a click on it opens it. Raw is what the cell sorts, groups, and filters by, its
// raw values, unstyled, as an age by its seconds and assignees by their logins; it sorts and groups by the first, and
// with none by its first value's text, naturally, so #7 comes before #31. Head is what a group's heading says of the
// value, for a cell that draws more than it groups by, as checks their counts beside their state; none says the cell's
// first value. Hang says its values start with a glyph, so a line an open row breaks one onto starts under the text
// after the glyph, not under the glyph: a hanging indent. Summary is what its column shows in place of its values while
// the row is closed, as how many labels an issue has; open, the row shows the values down the same column, one to a
// line, so two lists in a row each keep their own. Whole says its column keeps the
// width of its widest value while the title can keep minFill cells, and gives way only after the title, as fit says.
// Links, for a list whose values lead apart, as a release's assets each to its download, is where each raw value
// leads, one a value: flattened, each row's copy leads where its own value does. The text that wraps may hold several
// lines, as a release's notes: closed, the row shows the first; open, each wraps on lines of its own.
type Cell struct {
	Values  []string
	Right   bool
	Wrap    bool
	Link    string
	Links   []string
	Raw     []string
	Head    string
	Hang    bool
	Summary string
	Whole   bool
}

// key gives what a cell sorts and groups by: its first raw value, else its first value's text without its styles.
func (c Cell) key() string {
	if len(c.Raw) > 0 && c.Raw[0] != "" {
		return c.Raw[0]
	}
	return ansi.Strip(c.first())
}

// heading gives the cell's value as a group's heading says it: its Head, else its first value without its styles, as
// the rows draw it; none for a cell without a value.
func (c Cell) heading() string {
	switch {
	case c.key() == "":
		return "none"
	case c.Head != "":
		return c.Head
	}
	return ansi.Strip(c.first())
}

// groups gives the values a row groups by in the cell, raw, each once, and each as a group's heading says it: for a
// cell of one value, its key, as heading says it; for a cell of several, each raw value with the value the row draws
// for it, without its styles, or with them where the cell's Head says it so, as a label's pill; for a cell without a
// value, the empty value, which no value is, headed none, so it never meets a value that reads none.
func (c Cell) groups() (values, heads []string) {
	if len(c.Raw) < 2 {
		return []string{c.key()}, []string{c.heading()}
	}
	for k, v := range c.Raw {
		if v == "" || slices.Contains(values, v) {
			continue
		}
		head := v
		if len(c.Values) == len(c.Raw) {
			if head = c.Values[k]; c.Head == "" {
				head = ansi.Strip(head)
			}
		}
		values, heads = append(values, v), append(heads, head)
	}
	if len(values) == 0 {
		return []string{""}, []string{"none"}
	}
	return values, heads
}

// each gives the cell once for each value a row groups by in it, as groups gives them, each copy holding that value
// alone: its raw value, the value drawn for it, and no count of them; where the cell's Head says its value as it draws
// it, as a label's pill, the copy's says its own. A cell of one value or none is given as it is.
func (c Cell) each() []Cell {
	values, heads := c.groups()
	if len(c.Raw) < 2 || len(values) == 1 && values[0] == "" {
		return []Cell{c}
	}
	out := make([]Cell, len(values))
	for k, v := range values {
		one := c
		one.Raw, one.Values, one.Summary, one.Links = []string{v}, []string{v}, "", nil
		if len(c.Values) == len(c.Raw) {
			one.Values = []string{c.Values[slices.Index(c.Raw, v)]}
		}
		if len(c.Links) == len(c.Raw) {
			one.Link = c.Links[slices.Index(c.Raw, v)]
		}
		if c.Head != "" {
			one.Head = heads[k]
		}
		out[k] = one
	}
	return out
}

// natural compares two texts as a person reads them: each run of digits by its number, the rest without regard to
// case, so 7 comes before 31 and a2 before a10.
func natural(a, b string) int {
	for a != "" && b != "" {
		da, db := digits(a), digits(b)
		switch {
		case da > 0 && db > 0:
			x, _ := strconv.ParseUint(a[:da], 10, 64)
			y, _ := strconv.ParseUint(b[:db], 10, 64)
			if x != y {
				return cmp.Compare(x, y)
			}
			a, b = a[da:], b[db:]
		default:
			ra, na := utf8.DecodeRuneInString(a)
			rb, nb := utf8.DecodeRuneInString(b)
			if c := cmp.Compare(unicode.ToLower(ra), unicode.ToLower(rb)); c != 0 {
				return c
			}
			a, b = a[na:], b[nb:]
		}
	}
	return cmp.Compare(len(a), len(b))
}

// digits gives how many bytes of digits a text starts with.
func digits(s string) int {
	n := 0
	for n < len(s) && s[n] >= '0' && s[n] <= '9' {
		n++
	}
	return n
}

// Row is one row: its key, which survives a refresh; its cells; the lines it opens to, each a row of cells again,
// drawn closed; whether it is an aside, as a heading, which a screen does not count, and whether that aside stands for
// itself, as a line saying what could not be read, which a filter keeps though it heads no row; and, for a heading
// whose group is closed, the rows it folds away, which are not drawn but still lay the columns and the header out, so
// closing a group moves no column and leaves the header standing. Rows once drawn are not changed: other rows are a
// new slice, since the columns are measured once for a slice of rows, as columnWidths says.
type Row struct {
	Key    string
	Cells  []Cell
	Detail [][]Cell
	Aside  bool
	Lone   bool
	Folded []Row
}

// shapeOf gives the cells of the first row of two cells or more, among the rows and those they fold away: the columns
// the header names; nothing when there is none.
func shapeOf(rows []Row) []Cell {
	for _, r := range rows {
		if len(r.Cells) >= 2 {
			return r.Cells
		}
		if shape := shapeOf(r.Folded); shape != nil {
			return shape
		}
	}
	return nil
}

// Line is one line the grid draws: the row it belongs to, an index into the rows drawn, and its text.
type Line struct {
	Row  int
	Text string
}

// Chosen marks a row selected, in the gutter before its ▸ on its first line: a filled box, centered on the line as the
// triangles are, solid enough to show without bold; a row not selected has nothing there, so the selection scans.
const Chosen = "▣"

// Partly marks a group's heading some of whose rows are selected, but not all, where Chosen marks one whose rows all
// are: a square half filled, as Chosen is filled.
const Partly = "◩"

// Unchosen marks, dim, a row that could be selected and is not, where a table lets only some of its rows be: an empty
// square, as Chosen is filled, so the rows that can be stand apart from those that cannot, and a click on it selects.
const Unchosen = "□"

// gap is the cells between two columns. The cell that wraps, the title, is the last to give up room: while it has less
// than its widest text needs, every other column wider than capWidth gives up the cells past it, the widest first, a
// column of Whole cells apart; while it has fewer than minFill, they give up more, down to floorWidth, and then the
// Whole columns do too; only then does the title have less, never fewer than floorWidth, the row then clipped.
const (
	gap        = 2
	minFill    = 12
	capWidth   = 16
	floorWidth = 6
)

// lead is the gutter before a row's lines: two cells for its selection, its mark and a space, so the mark never
// stands against the ▸ or ▾ in the two after; a line the row opens to starts under its first cell, and three cells in.
const (
	lead  = "    "
	under = lead + "   "
)

// first gives a cell's first value, or nothing.
func (c Cell) first() string {
	if len(c.Values) == 0 {
		return ""
	}
	return c.Values[0]
}

// firstLine gives a text's first line that is not blank, or nothing, where it has none: a closed line is one line.
func firstLine(s string) string {
	for _, line := range strings.Split(s, "\n") {
		if strings.TrimSpace(line) != "" {
			return line
		}
	}
	return ""
}

// face gives what a cell's column shows of it: its summary, when it has one, else its first value.
func (c Cell) face() string {
	if c.Summary != "" {
		return c.Summary
	}
	return c.first()
}

// more gives what a closed cell shows after its first value when it holds more: +N, dim, a space before it; nothing
// for a cell with a summary, which says it already.
func (c Cell) more() string {
	if len(c.Values) < 2 || c.Summary != "" {
		return ""
	}
	return " " + style.Off("+"+strconv.Itoa(len(c.Values)-1))
}

// Draw lays the rows out for a pane of so many cells, under the header's names, which widen the columns they stand
// over: each row's lines, closed, or open when open says so, the cursor's row, an index into the rows, on a band, and
// each row selected, as one of several, with what picked gives for it, drawn, in the gutter's first cell of its first
// line: Chosen in a color, or nothing for a row not selected. picked may be nil. A row opens only when it hides
// something. A group's heading stands two cells further out than a row, its ▾ or ▸ where a row's own stands, so the
// rows under it stand in from it, and what picked gives for it, as Chosen or Partly, in the two cells before, after
// the heading's own indent, so its mark stands in with it. The heading over the groups is drawn as nothing: a screen
// draws it apart, over the rows, and it lays the columns out still with what it holds. The columns stand scrolled by
// shift, as Scroll says, so a row shows the columns the header names over it.
func Draw(rows []Row, head []string, width, shift, cursor int, open func(Row) bool, picked func(Row) string) []Line {
	widths := headed(rows, head)
	cells := width - len(lead)
	shift, _, _ = Scroll(rows, head, width, shift)
	var out []Line
	for k, r := range rows {
		if OverGroups(r) {
			continue
		}
		fitted, left := fit(r.Cells, widths, cells)
		hides := Hides(r, fitted, left)
		opened := hides && open(r)
		var lines []string
		if opened {
			lines = openLines(r.Cells, fitted, left, cells, shift, k == cursor)
		} else {
			lines = []string{closedLine(r.Cells, fitted, left, cells, shift, k == cursor)}
		}
		mark := " "
		switch {
		case opened:
			mark = "▾"
		case hides:
			mark = style.Off("▸")
		}
		selected := " "
		if picked != nil && picked(r) != "" {
			selected = picked(r)
		}
		for n, l := range lines {
			g := lead
			switch {
			case Heads(r) && n == 0:
				// the heading's indent, its depth's, stands before its mark
				plain := ansi.Strip(l)
				in := len(plain) - len(strings.TrimLeft(plain, " "))
				g, l = strings.Repeat(" ", in)+selected+" ", ansi.TruncateLeft(l, in, "")
			case Heads(r):
				g = "  "
			case n == 0:
				g = selected + " " + mark + " "
			}
			out = append(out, Line{k, g + l})
		}
		if opened {
			for _, d := range table(r.Detail, cells-len(under)+len(lead)) {
				out = append(out, Line{k, under + d})
			}
		}
	}
	for i := range out {
		if out[i].Row == cursor {
			out[i].Text = style.Band(out[i].Text, width)
		}
	}
	return out
}

// Header draws the names over the rows' columns, as Draw lays the rows out under them for a pane of so many cells,
// scrolled by shift: each name in its column, bold, right-aligned over a number, and wrapped in its column when wider,
// as only the text that wraps can be, since a name widens any other column to fit it, a word wider than the column
// broken in it. Like a row, it shows its first line alone, with ▸ in the gutter before it, when it is closed and its
// names take more; open, every line, with ▾. A column no row has text in is left out, as Draw leaves it. Where the
// view hides columns, as Scroll says, its first line has Before after the first column for those scrolled past, and
// After at its right end for those past the edge. Rows of one cell alone, which span the width, have no header. It
// says whether it hides lines while it is closed.
func Header(rows []Row, names []string, width, shift int, open bool) (lines []string, hides bool) {
	shape := shapeOf(rows)
	if shape == nil {
		return nil, false
	}
	cells := width - len(lead)
	widths, left := fit(shape, headed(rows, names), cells)
	shown, room, to, after := view(columnsOf(shape, widths, left), cells, shift)
	named := make([][]string, len(shown))
	height := 0
	for k, col := range shown {
		switch {
		case col.j >= len(names) || names[col.j] == "":
		case col.cut:
			named[k] = []string{pane.Clip(names[col.j], col.w)}
		default:
			named[k] = strings.Split(ansi.Wrap(names[col.j], max(col.w, 1), ""), "\n")
		}
		height = max(height, len(named[k]))
	}
	hides = height > 1
	mark := " "
	switch {
	case hides && open:
		mark = "▾"
	case hides:
		mark, height = style.Off("▸"), 1
	}
	out := make([]string, height)
	for k := range height {
		parts := make([]string, len(shown))
		for n, col := range shown {
			v := ""
			if k < len(named[n]) {
				v = style.Label(named[n][k])
			}
			parts[n] = pane.Pad(v, col.w, shape[col.j].Right)
		}
		line := lay(shown, parts)
		if k == 0 && to > 0 {
			// Before stands between the first column and those after it, a space either side
			first := ansi.StringWidth(parts[0])
			line = parts[0] + " " + Before + " " + ansi.TruncateLeft(line, first+3, "")
		}
		text := pane.Clip(strings.TrimRight(line, " "), room)
		if k == 0 && after {
			text = pane.Pad(text, room, false) + " " + After
		}
		g := lead
		if k == 0 {
			g = "  " + mark + " "
		}
		out[k] = g + text
	}
	return out, hides
}

// Before and After are the triangles the header's first line has where the view hides columns, as the tab bar's hide
// tabs: Before after the first column, for those scrolled past, and After at the right end, for those past the edge.
// A click on one scrolls the columns that way, as tab and shift-tab do.
const (
	Before = "◃"
	After  = "▹"
)

// column is one column a row lays out: its index into the row's cells, its width, and the cell it starts on, as view
// places it, past the gutter; and whether the view cuts it at its edge, narrower than its width, so its values and its
// name are clipped there, not wrapped.
type column struct {
	j, w, x int
	cut     bool
}

// columnsOf gives the columns a row of several cells lays out, in their order, as fit gave their widths and the
// title's cells: each column with a width, and the title.
func columnsOf(r []Cell, widths []int, left int) []column {
	var out []column
	for j, c := range r {
		switch {
		case c.Wrap:
			out = append(out, column{j: j, w: left})
		case j < len(widths) && widths[j] > 0:
			out = append(out, column{j: j, w: widths[j]})
		}
	}
	return out
}

// view places the columns in so many cells, scrolled by shift as the tab bar's view scrolls, but a column at a time:
// the first column, which says which row it is, stays put; after it, the columns from the shift'th past it on, each
// that stands whole in the cells, then the next cut at the edge, as the tab bar cuts a tab there, where that leaves it
// floorWidth cells, or where it is the first after the first column. It gives the columns it shows, each at the cell
// it starts on; the cells their line is cut at; the shift it stands at, no further than the one that shows the last
// column whole, or alone after the first; and whether the view hides or cuts columns past the edge that a further
// shift shows. A view scrolled sets the columns after the first a cell further from it, for Before between; one that
// hides columns after cuts the line two cells short, for a space and After. A shift below 0 places every column whole,
// the line cut at the cells, as the lines a row opens to are.
func view(cols []column, cells, shift int) (shown []column, room, to int, after bool) {
	if shift < 0 || len(cols) < 2 {
		x := 0
		for _, c := range cols {
			c.x, x = x, x+c.w+gap
			shown = append(shown, c)
		}
		return shown, cells, 0, false
	}
	for to = 0; ; to++ {
		shown, room, after = place(cols, cells, to)
		if to >= shift || !after {
			return shown, room, to, after
		}
	}
}

// place places the columns as view does, scrolled by exactly shift columns past the first, of at least two, and no
// more than leaves one after the first.
func place(cols []column, cells, shift int) (shown []column, room int, after bool) {
	try := func(room int) ([]column, bool) {
		first := cols[0]
		first.x = 0
		shown := []column{first}
		x := first.w + gap
		if shift > 0 {
			x++
		}
		for k, c := range cols[1+shift:] {
			if x+c.w <= room {
				c.x = x
				shown = append(shown, c)
				x += c.w + gap
				continue
			}
			if left := room - x; left >= floorWidth || k == 0 {
				c.x, c.w, c.cut = x, max(left, 0), true
				shown = append(shown, c)
			}
			return shown, true
		}
		return shown, false
	}
	// the last column alone after the first leaves nothing to scroll to: it is cut at the edge, with no After
	if shown, after = try(cells); !after || shift >= len(cols)-2 {
		return shown, cells, false
	}
	shown, _ = try(cells - 2)
	return shown, cells - 2, true
}

// Scroll gives how the columns stand scrolled by shift, as Draw and Header lay them out for a pane of so many cells
// under the header's names: the shift they stand at, no further than the one that shows the last column; and whether
// the view hides columns before those it shows, past the first, and after them, past the edge. Nothing scrolls where
// every column stands whole.
func Scroll(rows []Row, names []string, width, shift int) (to int, before, after bool) {
	shape := shapeOf(rows)
	if shape == nil {
		return 0, false, false
	}
	widths, left := fit(shape, headed(rows, names), width-len(lead))
	_, _, to, after = view(columnsOf(shape, widths, left), width-len(lead), max(shift, 0))
	return to, to > 0, after
}

// Arrows gives the columns of the screen Before and After stand on in the header's first line, from the header's own
// first, as Header draws it scrolled by shift for a pane of so many cells; -1 for one it does not draw.
func Arrows(rows []Row, names []string, width, shift int) (before, after int) {
	_, back, on := Scroll(rows, names, width, shift)
	before, after = -1, -1
	if back {
		first := at(rows, names, width, shift)[0]
		before = first.x + first.w + 1
	}
	if on {
		after = width - 1
	}
	return before, after
}

// at gives the columns the header shows of the rows, as Header places them scrolled by shift for a pane of so many
// cells, each at the column of the screen it starts on, from the header's own first.
func at(rows []Row, names []string, width, shift int) []column {
	shape := shapeOf(rows)
	if shape == nil {
		return nil
	}
	widths, left := fit(shape, headed(rows, names), width-len(lead))
	shown, _, _, _ := view(columnsOf(shape, widths, left), width-len(lead), shift)
	for k := range shown {
		shown[k].x += len(lead)
	}
	return shown
}

// HeaderAt gives the column whose name the header draws at the column x of the screen, as an index into the rows'
// cells, as Header lays the names out for a pane of so many cells, scrolled by shift; -1 off every name, or with no
// header.
func HeaderAt(rows []Row, names []string, width, shift, x int) int {
	for _, c := range at(rows, names, width, shift) {
		if x >= c.x && x < c.x+c.w && c.j < len(names) && names[c.j] != "" {
			return c.j
		}
	}
	return -1
}

// HeaderName gives where the header draws the name of the column j, an index into the rows' cells, as Header lays the
// names out for a pane of so many cells, scrolled by shift: the column of the screen its first line starts on, from
// the header's own first, and that line, without its styles; -1 and none where the header names it not, as a column
// no row fills, and -1 with its name where the view hides it, scrolled past or past the edge.
func HeaderName(rows []Row, names []string, width, shift, j int) (int, string) {
	if j < 0 || j >= len(names) || names[j] == "" {
		return -1, ""
	}
	shape := shapeOf(rows)
	first := func(w int, cut bool) string {
		if cut {
			return ansi.Strip(pane.Clip(names[j], w))
		}
		return ansi.Strip(strings.Split(ansi.Wrap(names[j], max(w, 1), ""), "\n")[0])
	}
	for _, s := range at(rows, names, width, shift) {
		if s.j != j {
			continue
		}
		line := first(s.w, s.cut)
		if shape[j].Right {
			return s.x + s.w - ansi.StringWidth(line), line
		}
		return s.x, line
	}
	widths, left := fit(shape, headed(rows, names), width-len(lead))
	for _, c := range columnsOf(shape, widths, left) {
		if c.j == j {
			return -1, first(c.w, false)
		}
	}
	return -1, ""
}

// Hides says whether a row hides something while it is closed: lines it opens to, a cell with more than one value, or
// with a summary in their place, or a text wider than the cells its column leaves it, the title's or one cut to give
// the title room.
func Hides(r Row, widths []int, left int) bool {
	if len(r.Detail) > 0 {
		return true
	}
	for j, c := range r.Cells {
		switch {
		case len(c.Values) > 1, c.Summary != "" && len(c.Values) > 0, c.Wrap && ansi.StringWidth(c.first()) > left,
			c.Wrap && strings.Contains(c.first(), "\n"):
			return true
		case !c.Wrap && len(r.Cells) > 1 && j < len(widths) && widths[j] > 0 && ansi.StringWidth(c.face()) > widths[j]:
			return true
		}
	}
	return false
}

// Opens says whether the row at k hides anything while it is closed, as Draw lays the rows out for a pane of so many
// cells under the header's names, and so has ▸ before it and can open.
func Opens(rows []Row, head []string, k, width int) bool {
	widths, left := fit(rows[k].Cells, headed(rows, head), width-len(lead))
	return Hides(rows[k], widths, left)
}

// LinkAt gives the link of the cell whose first word, its glyph where it draws one, a row's first line draws at the
// column x, as Draw lays the rows out for a pane of so many cells under the header's names, scrolled by shift; nothing
// when no linked word is there. A cell that draws no glyph, as a reference, leads from its first word.
func LinkAt(rows []Row, head []string, k, width, shift, x int) string {
	r := rows[k].Cells
	if len(r) == 0 {
		return ""
	}
	widths, left := fit(r, headed(rows, head), width-len(lead))
	cols := []column{{w: width - len(lead)}}
	if len(r) > 1 {
		shift, _, _ = Scroll(rows, head, width, shift)
		cols, _, _, _ = view(columnsOf(r, widths, left), width-len(lead), shift)
	}
	for _, col := range cols {
		c := r[col.j]
		start := len(lead) + col.x
		if c.Right {
			start += col.w - ansi.StringWidth(c.face()+c.more())
		}
		word, _, _ := strings.Cut(ansi.Strip(c.face()), " ")
		if c.Link != "" && x >= start && x < start+min(ansi.StringWidth(word), col.w) {
			return c.Link
		}
	}
	return ""
}

// table lays rows out closed, as the lines a row opens to are: one line each, clipped to the width.
func table(rows [][]Cell, width int) []string {
	widths := measure(wrap(rows))
	lines := make([]string, len(rows))
	for i, r := range rows {
		fitted, left := fit(r, widths, width)
		lines[i] = closedLine(r, fitted, left, width, -1, false)
	}
	return lines
}

// wrap gives the rows as rows of cells, for columnWidths.
func wrap(rows [][]Cell) []Row {
	out := make([]Row, len(rows))
	for i, r := range rows {
		out[i].Cells = r
	}
	return out
}

// headed gives the columns' widths, as columnWidths does, each named column with text in it widened to its name.
func headed(rows []Row, head []string) []int {
	widths := columnWidths(rows)
	for j, name := range head {
		if j < len(widths) && widths[j] > 0 {
			widths[j] = max(widths[j], ansi.StringWidth(name))
		}
	}
	return widths
}

// columnWidths gives each column's width over the rows, as measure does, measured once for the rows: every call
// after the first for the same rows reads the widths measured, so a header, a click, or a turn of the wheel, which lay
// out the columns again, costs what the columns cost, not what the rows do. The rows are the same while their slice
// is: rows are never changed once built, and a cached slice is held, so its memory is never another's.
func columnWidths(rows []Row) []int {
	if len(rows) == 0 {
		return nil
	}
	key := measured{&rows[0], len(rows)}
	widthsCache.Lock()
	defer widthsCache.Unlock()
	for _, e := range widthsCache.entries {
		if e.key == key {
			return slices.Clone(e.widths)
		}
	}
	widths := measure(rows)
	widthsCache.entries[widthsCache.next] = measuredWidths{key, widths}
	widthsCache.next = (widthsCache.next + 1) % len(widthsCache.entries)
	return slices.Clone(widths)
}

// measured names the rows widths were measured over: the first row's place and how many there are.
type measured struct {
	first *Row
	n     int
}

// measuredWidths is the widths measured over the rows the key names.
type measuredWidths struct {
	key    measured
	widths []int
}

// widthsCache holds the widths measured over the rows drawn of late, for columnWidths: enough for every stage, the
// rows above a frozen line, and a dialog's table over them, the oldest replaced first.
var widthsCache struct {
	sync.Mutex
	entries [32]measuredWidths
	next    int
}

// measure gives each column's width over the rows of two cells or more, those a closed heading folds away too: the
// most any of its cells needs, as need says.
func measure(rows []Row) []int {
	var widths []int
	for _, r := range rows {
		for j, w := range measure(r.Folded) {
			for len(widths) <= j {
				widths = append(widths, 0)
			}
			widths[j] = max(widths[j], w)
		}
		if len(r.Cells) < 2 {
			continue
		}
		for j, c := range r.Cells {
			for len(widths) <= j {
				widths = append(widths, 0)
			}
			widths[j] = max(widths[j], c.need())
		}
	}
	return widths
}

// need gives the cells a cell takes in its column: its widest value, or its first value and +N; for the cell that
// wraps, the widest it is on one line, which fit takes as the most it needs; for a cell with a summary, its widest
// value or the summary, as its values stand in its column when the row is open.
func (c Cell) need() int {
	n := 0
	if c.Summary != "" {
		n = ansi.StringWidth(c.Summary)
	}
	if !c.Wrap {
		n = ansi.StringWidth(c.first() + c.more())
	}
	for _, v := range c.Values {
		n = max(n, ansi.StringWidth(v))
	}
	return n
}

// Need gives the cells a column of these cells takes, as Draw lays it out: its widest value, or first value and +N,
// widened to the name the header stands over it; none for a column no cell has text in, which Draw leaves out.
func Need(cells []Cell, name string) int {
	n := 0
	for _, c := range cells {
		n = max(n, c.need())
	}
	if n > 0 {
		n = max(n, ansi.StringWidth(name))
	}
	return n
}

// Fits says whether columns that need so many cells each stand in a pane of so many cells, a gap between each two,
// with, when one of them wraps, minFill cells left for it: as fit lays them out without giving up room. A column that
// needs none is left out, as Draw leaves it.
func Fits(needs []int, wraps bool, width int) bool {
	n, used := 0, 0
	for _, w := range needs {
		if w > 0 {
			n, used = n+1, used+w
		}
	}
	if wraps {
		n, used = n+1, used+minFill
	}
	return used+max(n-1, 0)*gap <= width
}

// fit lays a row's columns out in a width: each column but the title takes its widest need, and the title, the cell
// that wraps, the width they and the gaps between them leave, never more than its widest text needs, so a narrow text
// leaves the room after the last column, not in the middle of the row. While that leaves the title less than its text
// needs, the other columns give up room, as capWidth and floorWidth say, a column of Whole cells after every other. It
// gives the columns' widths and the title's cells. A row of one cell spans the width.
func fit(r []Cell, widths []int, width int) ([]int, int) {
	if len(r) == 1 {
		return widths, width
	}
	gaps, need, wraps := -1, 0, false
	for j, c := range r {
		switch {
		case c.Wrap:
			gaps, wraps = gaps+1, true
			if j < len(widths) {
				need = widths[j]
			}
		case j < len(widths) && widths[j] > 0:
			gaps++
		}
	}
	room := width - gaps*gap
	if !wraps || need == 0 || room-used(r, widths) >= need {
		return widths, title(room-used(r, widths), need, minFill)
	}
	widths = slices.Clone(widths)
	shrink(r, widths, room-need, capWidth, false)
	if room-used(r, widths) < minFill {
		shrink(r, widths, room-minFill, floorWidth, false)
		shrink(r, widths, room-minFill, floorWidth, true)
	}
	return widths, title(room-used(r, widths), need, floorWidth)
}

// title gives the cells the title takes of those left: no more than it needs, when it needs any, and no fewer than
// least, the row then clipped.
func title(left, need, least int) int {
	fill := max(left, least)
	if need > 0 {
		fill = min(fill, need)
	}
	return fill
}

// used gives the cells a row's columns but the title take, the gaps apart.
func used(r []Cell, widths []int) int {
	n := 0
	for j, c := range r {
		if !c.Wrap && j < len(widths) {
			n += widths[j]
		}
	}
	return n
}

// shrink narrows a row's columns but the title, the widest first, each to no fewer cells than least, until they take
// no more than budget, or every one is down to least; a column of a Whole cell only with whole.
func shrink(r []Cell, widths []int, budget, least int, whole bool) {
	for used(r, widths) > budget {
		widest := -1
		for j, c := range r {
			if !c.Wrap && (whole || !c.Whole) && j < len(widths) && widths[j] > least &&
				(widest < 0 || widths[j] > widths[widest]) {
				widest = j
			}
		}
		if widest < 0 {
			return
		}
		widths[widest]--
	}
}

// closedLine draws a closed row: each cell's first value, padded to its column, +N after it when it holds more, the
// number right-aligned, the text that wraps cut to the width left; a column with no text is left out, and so is one
// the view hides, scrolled by shift as view says. The chosen row's first cell is drawn as choose says, but for a row
// of one cell, which spans the columns and only the band marks.
func closedLine(r []Cell, widths []int, left, width, shift int, chosen bool) string {
	if len(r) == 1 {
		return pane.Clip(r[0].first(), width)
	}
	shown, room, _, _ := view(columnsOf(r, widths, left), width, shift)
	parts := make([]string, len(shown))
	for k, col := range shown {
		c := r[col.j]
		if c.Wrap {
			parts[k] = pane.Pad(dotted(pane.Clip(firstLine(c.first()), col.w), c), col.w, false)
			continue
		}
		more := c.more()
		w := col.w - ansi.StringWidth(more)
		v := dotted(pane.Clip(c.face(), w), c)
		if k == 0 {
			v = choose(v, chosen)
		}
		parts[k] = pane.Pad(v, w, c.Right) + more
	}
	return pane.Clip(strings.TrimRight(lay(shown, parts), " "), room)
}

// lay gives the columns' parts of a line, the first first, then each after the cells view sets between it and the
// column before it, a gap, or a cell more after the first where the view is scrolled.
func lay(cols []column, parts []string) string {
	var line strings.Builder
	for k, col := range cols {
		if k > 0 {
			line.WriteString(strings.Repeat(" ", col.x-cols[k-1].x-cols[k-1].w))
		}
		line.WriteString(parts[k])
	}
	return line.String()
}

// openLines draws an open row: as many lines as its tallest cell the view shows, scrolled by shift as view says, each
// value on a line of its own in its column, a value wider than its column broken across lines in it, and the text that
// wraps wrapped in its; the chosen row's first cell is drawn as choose says, as a closed row's is. Every line is
// clipped to the width, a row of one cell's too: the wrap keeps lines ten cells wide where the width is less, and a
// word wider than the width stands alone.
func openLines(r []Cell, widths []int, left, width, shift int, chosen bool) []string {
	if len(r) == 1 {
		lines := wrapped(r[0], width)
		for k, l := range lines {
			lines[k] = pane.Clip(l, width)
		}
		return lines
	}
	shown, room, _, _ := view(columnsOf(r, widths, left), width, shift)
	columns := make([][]string, len(shown))
	height := 1
	for k, col := range shown {
		c := r[col.j]
		switch {
		case c.Wrap:
			columns[k] = wrapped(c, col.w)
		case col.cut:
			// a column cut at the edge keeps a value a line, clipped there, as the closed row's are
			for _, v := range c.Values {
				columns[k] = append(columns[k], pane.Clip(v, col.w))
			}
		default:
			for _, v := range c.Values {
				columns[k] = append(columns[k], hung(v, c.Hang, col.w, broken)...)
			}
		}
		height = max(height, len(columns[k]))
	}
	out := make([]string, height)
	for n := range height {
		parts := make([]string, len(shown))
		for k, col := range shown {
			c := r[col.j]
			v := ""
			if n < len(columns[k]) {
				v = columns[k][n]
			}
			if n == 0 {
				v = dotted(v, c)
			}
			if n == 0 && k == 0 {
				v = choose(v, chosen)
			}
			parts[k] = pane.Pad(v, col.w, c.Right)
		}
		out[n] = pane.Clip(strings.TrimRight(lay(shown, parts), " "), room)
	}
	return out
}

// broken gives a value in lines of so many cells: broken at spaces, after a dash or a slash, and inside a word only
// where one is wider. The dash or the slash a line breaks after may stand past the limit ansi.Wordwrap is given, so it
// is given one cell less. A style that runs across a break goes on, as pane.Restyle carries it.
func broken(v string, width int) []string {
	if ansi.StringWidth(v) <= width {
		return []string{v}
	}
	var out []string
	for _, l := range strings.Split(ansi.Wordwrap(v, max(width-1, 1), "-/"), "\n") {
		out = append(out, strings.Split(ansi.Hardwrap(l, max(width, 1), true), "\n")...)
	}
	return pane.Restyle(out)
}

// wrapped gives a cell's text broken into lines of so many cells, every value after the first on lines of its own,
// and every line of a value of several too, the blank lines between them kept as one.
func wrapped(c Cell, width int) []string {
	var out []string
	for _, v := range c.Values {
		blank := false
		for k, line := range strings.Split(v, "\n") {
			if strings.TrimSpace(line) == "" {
				blank = k > 0
				continue
			}
			if blank {
				out, blank = append(out, ""), false
			}
			out = append(out, hung(line, c.Hang, width, func(v string, width int) []string {
				return pane.Wrap(v, max(width, 1))
			})...)
		}
	}
	if len(out) == 0 {
		out = []string{""}
	}
	return out
}

// hung breaks a value into lines of so many cells as wrap does, with a hanging indent when hang says the value starts
// with a glyph: the text after the glyph and its space is broken narrower by their cells, the glyph stands before its
// first line, and the lines after start under the text. A value whose first word is no glyph, as it has a letter or a
// digit, or a column too narrow to leave ten cells after the glyph, breaks as wrap breaks it.
func hung(v string, hang bool, width int, wrap func(string, int) []string) []string {
	glyph, text, ok := strings.Cut(v, " ")
	cells := ansi.StringWidth(glyph) + 1
	if !hang || !ok || cells == 1 || width-cells < 10 || strings.ContainsFunc(ansi.Strip(glyph), func(r rune) bool {
		return unicode.IsLetter(r) || unicode.IsDigit(r)
	}) || ansi.StringWidth(v) <= width {
		return wrap(v, width)
	}
	// the text goes on in the styles the glyph started, as Restyle carries a style across a break
	lines := wrap(strings.Join(sgr.FindAllString(glyph, -1), "")+text, width-cells)
	out := make([]string, len(lines))
	for k, l := range lines {
		lead := strings.Repeat(" ", cells)
		if k == 0 {
			lead = glyph + " "
		}
		out[k] = lead + l
	}
	return out
}

// sgr is a sequence that sets how text is drawn: its color, bold, dim, and the like.
var sgr = regexp.MustCompile(`\x1b\[[0-9;:]*m`)

// dotted underlines a cell's glyph, or first word, its text up to the first space, with a dotted line when the cell's
// link opens a page, a URL, as a terminal that draws underline styles shows it; another draws a plain underline, or
// none. A link to a row of another stage is no URL, and its glyph stays as it is; so does a text that opens with a
// style of its own.
func dotted(v string, c Cell) string {
	if !strings.Contains(c.Link, "://") || v == "" || strings.HasPrefix(v, "\x1b") {
		return v
	}
	glyph, rest, spaced := strings.Cut(v, " ")
	if spaced {
		rest = " " + rest
	}
	return "\x1b[4:4m" + glyph + "\x1b[4:0m" + rest
}

// choose draws the chosen row's first cell: bold in the accent, its own styles, as dim, giving way, since the accent
// cannot draw over them. A pill keeps its own: its colors are what it is, and its half circles, stripped of them,
// would stand bare in the accent; the band marks its row.
func choose(s string, chosen bool) string {
	if chosen && s != "" && !strings.Contains(s, style.PillLeft) {
		return style.Mark(ansi.Strip(s))
	}
	return s
}
