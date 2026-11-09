// Package grid is the rows a screen lays out in columns: kit flow's stages draw theirs through it. A row has cells, one
// a column, and the lines it opens to. A cell holds one value, or several, as an issue's assignees, or a text that
// wraps, as a title. A closed row shows a cell's first value, with +N, right-aligned in the cell, when it holds more,
// and its text cut with … where its column ends. An open row shows every value, one a line in its column, the text
// wrapped in its column, then the lines it opens to. A cell with a summary, as an issue's labels by their count, shows
// the summary in its column, closed or open, and an open row's values in it follow the row's lines, as many to a line
// as the width holds. A row that hides anything, closed, has ▸ in the gutter before it,
// and ▾ open; the row under the cursor is drawn on a faint band, its first cell bold in the accent; the band marks the
// row, so the cell is not underlined as a chosen name is. A glyph, or a first word, whose link opens a page is
// underlined with dots. A header can stand the columns' names over the rows, widening each column to its name; a click
// on a name sorts by it. A row selected, one of several, has ✓ before its ▸. A panel beside the rows narrows them: a
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
// after the glyph, not under the glyph: a hanging indent. Summary is what its column shows in place of its values, as
// how many labels an issue has: the column is as wide as the summary, and an open row draws the values after its own
// lines, side by side across the width, so values too wide for a column stand whole.
type Cell struct {
	Values  []string
	Right   bool
	Wrap    bool
	Link    string
	Raw     []string
	Head    string
	Hang    bool
	Summary string
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
		one.Raw, one.Values, one.Summary = []string{v}, []string{v}, ""
		if len(c.Values) == len(c.Raw) {
			one.Values = []string{c.Values[slices.Index(c.Raw, v)]}
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
// closing a group moves no column and leaves the header standing.
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

// gap is the cells between two columns. The cell that wraps, the title, is the last to give up room: while it has less
// than its widest text needs, every other column wider than capWidth gives up the cells past it, the widest first;
// while it has fewer than minFill, they give up more, down to floorWidth; only then does the title have less, never
// fewer than floorWidth, the row then clipped.
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
// draws it apart, over the rows, and it lays the columns out still with what it holds.
func Draw(rows []Row, head []string, width, cursor int, open func(Row) bool, picked func(Row) string) []Line {
	widths := headed(rows, head)
	cells := width - len(lead)
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
			lines = openLines(r.Cells, fitted, left, cells, k == cursor)
		} else {
			lines = []string{closedLine(r.Cells, fitted, left, cells, k == cursor)}
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
			for _, c := range r.Cells {
				if c.Summary == "" {
					continue
				}
				for _, l := range flowed(c.Values, cells-len(under)+len(lead)) {
					out = append(out, Line{k, under + l})
				}
			}
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

// Header draws the names over the rows' columns, as Draw lays the rows out under them for a pane of so many cells:
// each name in its column, bold, right-aligned over a number, and wrapped in its column when wider, as only the text
// that wraps can be, since a name widens any other column to fit it, a word wider than the column broken in it. Like a
// row, it shows its first line alone, with ▸ in the gutter before it, when it is closed and its names take more; open,
// every line, with ▾. A column no row has text in is left out, as Draw leaves it. Rows of one cell alone, which span
// the width, have no header. It says whether it hides lines while it is closed.
func Header(rows []Row, names []string, width int, open bool) (lines []string, hides bool) {
	shape := shapeOf(rows)
	if shape == nil {
		return nil, false
	}
	cells := width - len(lead)
	widths, left := fit(shape, headed(rows, names), cells)
	type column struct {
		lines []string
		width int
		right bool
	}
	var columns []column
	height := 0
	for j, c := range shape {
		w := left
		if !c.Wrap {
			if j >= len(widths) || widths[j] == 0 {
				continue
			}
			w = widths[j]
		}
		name := ""
		if j < len(names) {
			name = names[j]
		}
		var lines []string
		if name != "" {
			lines = strings.Split(ansi.Wrap(name, max(w, 1), ""), "\n")
		}
		columns = append(columns, column{lines, w, c.Right})
		height = max(height, len(lines))
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
		var parts []string
		for _, c := range columns {
			v := ""
			if k < len(c.lines) {
				v = style.Label(c.lines[k])
			}
			parts = append(parts, pane.Pad(v, c.width, c.right))
		}
		g := lead
		if k == 0 {
			g = "  " + mark + " "
		}
		out[k] = g + pane.Clip(strings.TrimRight(strings.Join(parts, strings.Repeat(" ", gap)), " "), cells)
	}
	return out, hides
}

// HeaderAt gives the column whose name the header draws at the column x of the screen, as an index into the rows'
// cells, as Header lays the names out for a pane of so many cells; -1 off every name, or with no header.
func HeaderAt(rows []Row, names []string, width, x int) int {
	shape := shapeOf(rows)
	widths, left := fit(shape, headed(rows, names), width-len(lead))
	at := len(lead)
	for j, c := range shape {
		w := left
		if !c.Wrap {
			if j >= len(widths) || widths[j] == 0 {
				continue
			}
			w = widths[j]
		}
		if x >= at && x < at+w && j < len(names) && names[j] != "" {
			return j
		}
		at += w + gap
	}
	return -1
}

// HeaderName gives where the header draws the name of the column j, an index into the rows' cells, as Header lays the
// names out for a pane of so many cells: the column of the screen its first line starts on, from the header's own
// first, and that line, without its styles; -1 and none where the header draws no name for it.
func HeaderName(rows []Row, names []string, width, j int) (int, string) {
	shape := shapeOf(rows)
	widths, left := fit(shape, headed(rows, names), width-len(lead))
	at := len(lead)
	for k, c := range shape {
		w := left
		if !c.Wrap {
			if k >= len(widths) || widths[k] == 0 {
				continue
			}
			w = widths[k]
		}
		if k == j {
			if j >= len(names) || names[j] == "" {
				break
			}
			first := ansi.Strip(strings.Split(ansi.Wrap(names[j], max(w, 1), ""), "\n")[0])
			if c.Right {
				return at + w - ansi.StringWidth(first), first
			}
			return at, first
		}
		at += w + gap
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
		case len(c.Values) > 1, c.Summary != "" && len(c.Values) > 0, c.Wrap && ansi.StringWidth(c.first()) > left:
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
// column x, as Draw lays the rows out for a pane of so many cells under the header's names; nothing when no linked
// word is there. A cell that draws no glyph, as a reference, leads from its first word.
func LinkAt(rows []Row, head []string, k, width, x int) string {
	r := rows[k].Cells
	widths, left := fit(r, headed(rows, head), width-len(lead))
	at := len(lead)
	for j, c := range r {
		w := left
		switch {
		case len(r) == 1:
			w = width - len(lead)
		case c.Wrap:
		case j >= len(widths) || widths[j] == 0:
			continue
		default:
			w = widths[j]
		}
		start := at
		if c.Right {
			start += w - ansi.StringWidth(c.face()+c.more())
		}
		word, _, _ := strings.Cut(ansi.Strip(c.face()), " ")
		if c.Link != "" && x >= start && x < start+min(ansi.StringWidth(word), w) {
			return c.Link
		}
		at += w + gap
	}
	return ""
}

// table lays rows out closed, as the lines a row opens to are: one line each, clipped to the width.
func table(rows [][]Cell, width int) []string {
	widths := columnWidths(wrap(rows))
	lines := make([]string, len(rows))
	for i, r := range rows {
		fitted, left := fit(r, widths, width)
		lines[i] = closedLine(r, fitted, left, width, false)
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

// columnWidths gives each column's width over the rows of two cells or more, those a closed heading folds away too:
// the most any of its cells needs, as need says.
func columnWidths(rows []Row) []int {
	var widths []int
	for _, r := range rows {
		for j, w := range columnWidths(r.Folded) {
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
// wraps, the widest it is on one line, which fit takes as the most it needs; for a cell with a summary, the summary,
// as its values stand under the row.
func (c Cell) need() int {
	if c.Summary != "" {
		return ansi.StringWidth(c.Summary)
	}
	n := 0
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
// needs, the other columns give up room, as capWidth and floorWidth say. It gives the columns' widths and the title's
// cells. A row of one cell spans the width.
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
	shrink(r, widths, room-need, capWidth)
	if room-used(r, widths) < minFill {
		shrink(r, widths, room-minFill, floorWidth)
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
// no more than budget, or every one is down to least.
func shrink(r []Cell, widths []int, budget, least int) {
	for used(r, widths) > budget {
		widest := -1
		for j, c := range r {
			if !c.Wrap && j < len(widths) && widths[j] > least && (widest < 0 || widths[j] > widths[widest]) {
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
// number right-aligned, the text that wraps cut to the width left; a column with no text is left out. The chosen row's
// first cell is drawn as choose says, but for a row of one cell, which spans the columns and only the band marks.
func closedLine(r []Cell, widths []int, left, width int, chosen bool) string {
	if len(r) == 1 {
		return pane.Clip(r[0].first(), width)
	}
	var parts []string
	for j, c := range r {
		switch {
		case c.Wrap:
			parts = append(parts, pane.Pad(dotted(pane.Clip(c.first(), left), c), left, false))
		case j >= len(widths) || widths[j] == 0:
		default:
			more := c.more()
			w := widths[j] - ansi.StringWidth(more)
			v := dotted(pane.Clip(c.face(), w), c)
			if len(parts) == 0 {
				v = choose(v, chosen)
			}
			parts = append(parts, pane.Pad(v, w, c.Right)+more)
		}
	}
	return pane.Clip(strings.TrimRight(strings.Join(parts, strings.Repeat(" ", gap)), " "), width)
}

// openLines draws an open row: as many lines as its tallest cell, each value on a line of its own in its column, a value
// wider than its column broken across lines in it, and the text that wraps wrapped in its; the chosen row's first cell
// is drawn as choose says, as a closed row's is. Every line is clipped to the width, a row of one cell's too: the wrap
// keeps lines ten cells wide where the width is less, and a word wider than the width stands alone.
func openLines(r []Cell, widths []int, left, width int, chosen bool) []string {
	if len(r) == 1 {
		lines := wrapped(r[0], width)
		for k, l := range lines {
			lines[k] = pane.Clip(l, width)
		}
		return lines
	}
	columns := make([][]string, len(r))
	height := 1
	for j, c := range r {
		switch {
		case c.Wrap:
			columns[j] = wrapped(c, left)
		case j < len(widths) && widths[j] > 0 && c.Summary != "":
			columns[j] = []string{pane.Clip(c.Summary, widths[j])}
		case j < len(widths) && widths[j] > 0:
			for _, v := range c.Values {
				columns[j] = append(columns[j], hung(v, c.Hang, widths[j], broken)...)
			}
		}
		height = max(height, len(columns[j]))
	}
	out := make([]string, height)
	for k := range height {
		var parts []string
		for j, c := range r {
			w := left
			if !c.Wrap {
				if j >= len(widths) || widths[j] == 0 {
					continue
				}
				w = widths[j]
			}
			v := ""
			if k < len(columns[j]) {
				v = columns[j][k]
			}
			if k == 0 {
				v = dotted(v, c)
			}
			if k == 0 && len(parts) == 0 {
				v = choose(v, chosen)
			}
			parts = append(parts, pane.Pad(v, w, c.Right))
		}
		out[k] = pane.Clip(strings.TrimRight(strings.Join(parts, strings.Repeat(" ", gap)), " "), width)
	}
	return out
}

// flowed lays values out side by side, a space between each two, in lines of so many cells: a line takes each value
// that still fits on it whole, and a value wider than a line has one of its own, cut where it ends.
func flowed(values []string, width int) []string {
	var out []string
	line, used := "", 0
	for _, v := range values {
		w := ansi.StringWidth(v)
		if line != "" && used+1+w > width {
			out, line, used = append(out, line), "", 0
		}
		if line != "" {
			line, used = line+" ", used+1
		}
		line, used = line+pane.Clip(v, width), used+min(w, width)
	}
	if line != "" {
		out = append(out, line)
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

// wrapped gives a cell's text broken into lines of so many cells, every value after the first on lines of its own.
func wrapped(c Cell, width int) []string {
	var out []string
	for _, v := range c.Values {
		out = append(out, hung(v, c.Hang, width, func(v string, width int) []string {
			return pane.Wrap(v, max(width, 1))
		})...)
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
// cannot draw over them.
func choose(s string, chosen bool) string {
	if chosen && s != "" {
		return style.Mark(ansi.Strip(s))
	}
	return s
}
