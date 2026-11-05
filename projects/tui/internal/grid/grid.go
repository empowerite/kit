// Package grid is the rows a screen lays out in columns: kit flow's stages draw theirs through it. A row has cells, one
// a column, and the lines it opens to. A cell holds one value, or several, as an issue's assignees, or a text that
// wraps, as a title. A closed row shows a cell's first value, with +N, right-aligned in the cell, when it holds more,
// and its text cut with … where its column ends. An open row shows every value, one a line in its column, the text
// wrapped in its column, then the lines it opens to. A row that hides anything, closed, has ▸ in the gutter before it,
// and ▾ open; the row under the cursor is drawn on a faint band, its first cell bold in the accent; the band marks the
// row, so the cell is not underlined as a chosen name is. A glyph whose link opens a page is underlined with dots. A
// header can stand the columns' names over the rows, widening each column to its name; a click on a name sorts by it. A
// row selected, one of several, has ✓ before its ▸.  A panel beside the rows narrows them: a search, which keeps the
// rows whose text has it; a checkbox per column, which shows or hides it; and, for rows that belong to people, a choice
// of every row, the caller's, or everyone else's. It also keeps the columns' order, the columns the rows sort by, and
// those they group by, each group under a heading; a column whose value is the same on every row of a group groups it
// on its own, its value said once in the heading.
package grid

import (
	"cmp"
	"slices"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/charmbracelet/x/ansi"

	"github.com/empowerite/kit/projects/tui/internal/pane"
	"github.com/empowerite/kit/projects/tui/internal/style"
)

// Cell is one column of a row: its values, styled already when they are dim, each a line when the row is open; Right
// is a number, right-aligned in its column; Wrap is the one cell of a row that takes the width left, cut while the
// row is closed and wrapped when it is open. Link is where the glyph its text starts with leads, as an issue's page:
// a click on the glyph opens it. Sort is what the cell sorts by, as an age by its seconds; none sorts by its first
// value's text, naturally, so #7 comes before #31.
type Cell struct {
	Values []string
	Right  bool
	Wrap   bool
	Link   string
	Sort   string
}

// key gives what a cell sorts and groups by: its sort value, else its first value's text without its styles.
func (c Cell) key() string {
	if c.Sort != "" {
		return c.Sort
	}
	return ansi.Strip(c.first())
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
// drawn closed; whether it is an aside, as a heading, which a screen does not count; and the logins it belongs to,
// which Show matches.
type Row struct {
	Key    string
	Cells  []Cell
	Detail [][]Cell
	Aside  bool
	Owners []string
}

// Line is one line the grid draws: the row it belongs to, an index into the rows drawn, and its text.
type Line struct {
	Row  int
	Text string
}

// Chosen marks a row selected, in the gutter before its ▸ on its first line: a filled box, centered on the line as the
// triangles are, solid enough to show without bold; a row not selected has nothing there, so the selection scans.
const Chosen = "▣"

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

// lead is the gutter before a row's lines; a line the row opens to starts under its first cell, and three cells in.
const (
	lead  = "   "
	under = lead + "   "
)

// first gives a cell's first value, or nothing.
func (c Cell) first() string {
	if len(c.Values) == 0 {
		return ""
	}
	return c.Values[0]
}

// more gives what a closed cell shows after its first value when it holds more: +N, dim, a space before it.
func (c Cell) more() string {
	if len(c.Values) < 2 {
		return ""
	}
	return " " + style.Off("+"+strconv.Itoa(len(c.Values)-1))
}

// Draw lays the rows out for a pane of so many cells, under the header's names, which widen the columns they stand
// over: each row's lines, closed, or open when open says so, the cursor's row, an index into the rows, on a band, and
// each row selected, as one of several, with what picked gives for it, drawn, in the gutter's first cell of its first
// line: Chosen in a color, or nothing for a row not selected. picked may be nil. A row opens only when it hides
// something.
func Draw(rows []Row, head []string, width, cursor int, open func(Row) bool, picked func(Row) string) []Line {
	widths := headed(rows, head)
	cells := width - len(lead)
	var out []Line
	for k, r := range rows {
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
			g := "   "
			if n == 0 {
				g = selected + mark + " "
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

// Header draws the names over the rows' columns, as Draw lays the rows out under them for a pane of so many cells:
// each name in its column, bold, right-aligned over a number, and wrapped in its column when wider, as only the text
// that wraps can be, since a name widens any other column to fit it, a word wider than the column broken in it. Like a
// row, it shows its first line alone, with ▸ in the gutter before it, when it is closed and its names take more; open,
// every line, with ▾. A column no row has text in is left out, as Draw leaves it. Rows of one cell alone, which span
// the width, have no header. It says whether it hides lines while it is closed.
func Header(rows []Row, names []string, width int, open bool) (lines []string, hides bool) {
	var shape []Cell
	for _, r := range rows {
		if len(r.Cells) >= 2 {
			shape = r.Cells
			break
		}
	}
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
			g = " " + mark + " "
		}
		out[k] = g + pane.Clip(strings.TrimRight(strings.Join(parts, strings.Repeat(" ", gap)), " "), cells)
	}
	return out, hides
}

// HeaderAt gives the column whose name the header draws at the column x of the screen, as an index into the rows'
// cells, as Header lays the names out for a pane of so many cells; -1 off every name, or with no header.
func HeaderAt(rows []Row, names []string, width, x int) int {
	var shape []Cell
	for _, r := range rows {
		if len(r.Cells) >= 2 {
			shape = r.Cells
			break
		}
	}
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

// Hides says whether a row hides something while it is closed: lines it opens to, a cell with more than one value, or
// a text wider than the cells its column leaves it, the title's or one cut to give the title room.
func Hides(r Row, widths []int, left int) bool {
	if len(r.Detail) > 0 {
		return true
	}
	for j, c := range r.Cells {
		switch {
		case len(c.Values) > 1, c.Wrap && ansi.StringWidth(c.first()) > left:
			return true
		case !c.Wrap && len(r.Cells) > 1 && j < len(widths) && widths[j] > 0 && ansi.StringWidth(c.first()) > widths[j]:
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

// LinkAt gives the link of the cell whose glyph, the first two cells of its text, a row's first line draws at the
// column x, as Draw lays the rows out for a pane of so many cells under the header's names; nothing when no linked
// glyph is there.
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
			start += w - ansi.StringWidth(c.first()+c.more())
		}
		if c.Link != "" && x >= start && x < start+2 {
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

// columnWidths gives each column's width over the rows of two cells or more: its widest value, or first value and
// +N; for the cell that wraps, the widest it is on one line, which fit takes as the most it needs.
func columnWidths(rows []Row) []int {
	var widths []int
	for _, r := range rows {
		if len(r.Cells) < 2 {
			continue
		}
		for j, c := range r.Cells {
			for len(widths) <= j {
				widths = append(widths, 0)
			}
			if c.Wrap {
				// what the text that wraps needs, at most, to stand on one line
				for _, v := range c.Values {
					widths[j] = max(widths[j], ansi.StringWidth(v))
				}
				continue
			}
			widths[j] = max(widths[j], ansi.StringWidth(c.first()+c.more()))
			for _, v := range c.Values {
				widths[j] = max(widths[j], ansi.StringWidth(v))
			}
		}
	}
	return widths
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
			v := dotted(pane.Clip(c.first(), w), c)
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
// is drawn as choose says, as a closed row's is.
func openLines(r []Cell, widths []int, left, width int, chosen bool) []string {
	if len(r) == 1 {
		return wrapped(r[0], width)
	}
	columns := make([][]string, len(r))
	height := 1
	for j, c := range r {
		switch {
		case c.Wrap:
			columns[j] = wrapped(c, left)
		case j < len(widths) && widths[j] > 0:
			for _, v := range c.Values {
				columns[j] = append(columns[j], broken(v, widths[j])...)
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

// broken gives a value in lines of so many cells: broken at spaces, after a dash or a slash, and inside a word only
// where one is wider. The dash or the slash a line breaks after may stand past the limit ansi.Wordwrap is given, so it
// is given one cell less.
func broken(v string, width int) []string {
	if ansi.StringWidth(v) <= width {
		return []string{v}
	}
	var out []string
	for _, l := range strings.Split(ansi.Wordwrap(v, max(width-1, 1), "-/"), "\n") {
		out = append(out, strings.Split(ansi.Hardwrap(l, max(width, 1), true), "\n")...)
	}
	return out
}

// wrapped gives a cell's text broken into lines of so many cells, every value after the first on lines of its own.
func wrapped(c Cell, width int) []string {
	var out []string
	for _, v := range c.Values {
		out = append(out, pane.Wrap(v, max(width, 1))...)
	}
	if len(out) == 0 {
		out = []string{""}
	}
	return out
}

// dotted underlines a cell's glyph, its text up to the first space, with a dotted line when the cell's link opens a
// page, a URL, as a terminal that draws underline styles shows it; another draws a plain underline, or none. A link to
// a row of another stage is no URL, and its glyph stays as it is; so does a text that opens with a style of its own.
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
