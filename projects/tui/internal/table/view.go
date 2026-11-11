// A table's view: the lines its pane shows, the control line over them, the headings frozen over its header, the
// header and its rule, and the rows' window, which scrolls; the cursor's moves through the rows, and the selection.
package table

import (
	"slices"

	"github.com/empowerite/kit/projects/tui/internal/grid"
	"github.com/empowerite/kit/projects/tui/internal/pane"
)

// Line is one line of a table's pane: the row it belongs to, or -1 for a line about the table, and its text.
type Line struct {
	Row  int
	Text string
}

// Place is a line of the pane under the control line: the index of the table's line it draws, as the host's Lines
// gives them, -1 for a line no row draws, one that says how many are hidden or the blank line over the header, or
// -2-h for the header's line h.
type Place = int

// HeaderPlace gives the place of the header's line h.
func HeaderPlace(h int) Place { return -2 - h }

// Frozen gives how many of a table's rows stand frozen over its header, past the heading over the groups: the headings
// of the outermost levels of the groups, each of which has one value alone, so one heading stands under the one
// before. They are the context of every row under them, so they never scroll; the first level of more than one
// value, and everything under it, scrolls under the header.
func Frozen(h Host) int {
	rows := h.Rows()
	first := First(rows)
	if first == 0 {
		return 0
	}
	values := map[int]int{} // how many headings each level has, those folded away too
	for _, r := range grid.Every(rows) {
		if grid.Heads(r) {
			values[grid.Level(r)]++
		}
	}
	// each heading deeper than the one before, which may hold the levels between on its line
	k, over := first, grid.Level(rows[0])
	for k < len(rows) && grid.Heads(rows[k]) && grid.Level(rows[k]) > over && values[grid.Level(rows[k])] == 1 {
		over = grid.Level(rows[k])
		k++
	}
	return k
}

// FrozenLines gives how many of the table's lines the frozen rows take: as many as they draw, the headings alone,
// never the rows under them, so a table of many lines costs no more to ask.
func FrozenLines(h Host) int {
	k := Frozen(h)
	if k == 0 {
		return 0
	}
	return len(grid.Draw(h.Rows()[:k], h.Names(), h.GridWidth(), h.Table().Shift, -1, h.Table().Open.Open, func(grid.Row) string { return "" }))
}

// Gap gives how many blank lines set a header of those lines apart from the frozen lines over it, as many as there
// are: one where both are, so the header stands apart from the context over it as it does from the control line; none
// where either is not, as the control line's blank line, or the pane's top, sets it apart already.
func Gap(frozen int, header []string) int {
	if frozen > 0 && len(header) > 0 {
		return 1
	}
	return 0
}

// RowsTop gives the screen line the rows' window starts on: the pane's first, or the one under the control line and
// the blank line that sets it apart from the rows.
func RowsTop(h Host) int {
	top, _ := h.Pane()
	if h.Controls() {
		return top + 2
	}
	return top
}

// HeightUnder gives the lines the rows have under a header of those lines, the pane's, less the control line's, the
// frozen lines over the header, and the blank line between them.
func HeightUnder(h Host, header []string) int {
	top, _ := h.Pane()
	frozen := FrozenLines(h)
	return max(1, h.PaneHeight()-(RowsTop(h)-top)-frozen-Gap(frozen, header)-len(header))
}

// RowsHeight gives the lines the rows have, which scroll: the pane's, less the control line's and the header's, which
// do not.
func RowsHeight(h Host) int {
	header, _ := h.Header()
	return HeightUnder(h, header)
}

// ScrollLines gives the table's lines that scroll, in the rows' window: those the host draws, past the frozen lines.
func ScrollLines(h Host) []Line { return h.Lines()[FrozenLines(h):] }

// Laid gives the pane's lines under the control line, as many as the frozen lines, the header, and the rows' window
// take, and the place of each: the frozen lines, which never scroll, as a pane freezes in a spreadsheet; a blank line
// under them, as Gap gives it; the header's lines and its rule, over the first item of the first level of the groups
// with more than one value, or over the first row; and the rows' window under them, as pane.Lines draws it.
func Laid(h Host) (texts []string, places []Place) {
	all, t := h.Lines(), h.Table()
	skip := FrozenLines(h)
	lines := all[skip:]
	header, _ := h.Header()
	height := HeightUnder(h, header)
	for k := range skip {
		texts, places = append(texts, all[k].Text), append(places, k)
	}
	for range Gap(skip, header) {
		texts, places = append(texts, ""), append(places, -1)
	}
	for k := range header {
		texts, places = append(texts, header[k]), append(places, HeaderPlace(k))
	}
	window := make([]string, len(lines))
	for i, l := range lines {
		window[i] = l.Text
	}
	head := Head(h, lines, height)
	window = pane.Lines(window, t.Offset, height, head, "   ")
	for r := range window {
		k := pane.At(len(lines), height, t.Offset, r)
		if r == 0 && head != "" {
			k = slices.IndexFunc(lines, func(l Line) bool { return l.Row == t.Cursor })
		}
		if k >= 0 {
			k += skip
		}
		places = append(places, k)
	}
	return append(texts, window...), places
}

// Head gives the line the pane pins on top, in place of its … above: the first line of the row under the cursor, once
// paging through that open row has scrolled it away while the rest of the row still shows; else nothing.
func Head(h Host, lines []Line, height int) string {
	if !h.InRows() {
		return ""
	}
	t := h.Table()
	start, _, above, _ := pane.Window(len(lines), height, t.Offset)
	first := slices.IndexFunc(lines, func(l Line) bool { return l.Row == t.Cursor })
	if !above || first < 0 || first >= start || lines[start].Row != t.Cursor {
		return ""
	}
	return lines[first].Text
}

// PlaceAt gives the place of the screen line y in the pane under the control line, as Laid gives it, and whether y
// falls on one.
func PlaceAt(h Host, y int) (Place, bool) {
	_, places := Laid(h)
	if r := y - RowsTop(h); r >= 0 && r < len(places) {
		return places[r], true
	}
	return -1, false
}

// HeaderTop gives the screen line the header's first line stands on, as Laid places it, or -1 where there is no
// header.
func HeaderTop(h Host) int {
	_, places := Laid(h)
	if r := slices.Index(places, HeaderPlace(0)); r >= 0 {
		return RowsTop(h) + r
	}
	return -1
}

// HeaderLine gives the line of the header the screen line y falls on, from 0, and whether it falls on one: the rule
// its last.
func HeaderLine(h Host, y int) (int, bool) {
	if k, ok := PlaceAt(h, y); ok && k <= HeaderPlace(0) {
		return HeaderPlace(0) - k, true
	}
	return 0, false
}

// OnRule says whether the screen line y is the rule under the header.
func OnRule(h Host, y int) bool {
	header, _ := h.Header()
	k, ok := HeaderLine(h, y)
	return ok && k == len(header)-1
}

// Up moves the cursor n rows up; past the first row, the rows give up the keyboard, and the pane goes to its top.
func Up(h Host, n int) {
	if !h.InRows() {
		return
	}
	if t := h.Table(); t.Cursor-n < First(h.Rows()) {
		t.Cursor, t.Offset = 0, 0
		h.LeaveRows()
	} else {
		t.Cursor -= n
	}
	Fit(h)
}

// Enter gives the rows the keyboard from the line over them, the control line, the cursor on the outline's first line:
// the first group's heading where the rows group, else the first row, as up from it leads back to the line; a table
// without a row stays as it is.
func Enter(h Host) {
	t, all := h.Table(), h.Rows()
	if len(all) <= First(all) {
		return
	}
	h.EnterRows()
	t.Cursor = First(all)
	Fit(h)
}

// Down moves the cursor n rows down; without the keyboard, the rows take it, when the table has any, past the
// headings of groups over the row under the cursor, so it lands on a row.
func Down(h Host, n int) {
	t, all := h.Table(), h.Rows()
	first, rows := First(all), len(all)
	if rows <= first {
		return
	}
	t.Cursor = max(t.Cursor, first)
	if !h.InRows() {
		h.EnterRows()
		n--
		for t.Cursor+n < rows-1 && grid.Heads(all[t.Cursor+n]) {
			n++
		}
	}
	t.Cursor = min(t.Cursor+n, rows-1)
	Fit(h)
}

// Page moves a page down or up. Inside a row taller than the pane, as an open pull request with many issues is, it
// first scrolls through that row a window at a time, so each line it hides shows at least once: down, the next window
// starts at the line under the last one shown, and up, the window before ends at the line over the first, the … lines
// in the pane taking their places; windows may overlap at a row's top and the table's end. At the row's end it moves
// the cursor a page of rows. A page never leaves the rows: up stops at the first row, and down at the last; without
// the keyboard it does nothing.
func Page(h Host, d int) {
	if !h.InRows() {
		return
	}
	height := RowsHeight(h)
	step := max(1, height-1)
	t, all := h.Table(), ScrollLines(h)
	first, last := rowLines(all, t.Cursor)
	start, end, _, _ := pane.Window(len(all), height, t.Offset)
	switch {
	case first >= 0 && d > 0 && last >= end:
		t.Offset, _, _, _ = pane.Window(len(all), height, max(end, start+1)) // a pane with no room still moves on
	case first >= 0 && d < 0 && first < start:
		t.Offset = max(first, before(len(all), height, start))
	case d > 0:
		Down(h, step)
	default:
		t.Cursor = max(First(h.Rows()), t.Cursor-step)
		Fit(h)
	}
}

// End moves the cursor to the first row, d below 0, as Home does, or to the last, as End does, wherever ⇞ and ⇟ page;
// without the keyboard it does nothing. At the first row the pane scrolls to its top, over the headings too.
func End(h Host, d int) {
	if !h.InRows() {
		return
	}
	t, all := h.Table(), h.Rows()
	if first := First(all); d < 0 {
		t.Cursor, t.Offset = first, 0
	} else if len(all) > first {
		t.Cursor = len(all) - 1
	}
	Fit(h)
}

// Group moves the cursor to a group's heading: [ to the one before it, ] to the one after it, { to the first, and } to
// the last, as the palette's keys move between its groups. A table with no heading the cursor can stand on, or
// without the keyboard, stays as it is.
func Group(h Host, press string) {
	if !h.InRows() {
		return
	}
	t, all := h.Table(), h.Rows()
	var heads []int
	for i := First(all); i < len(all); i++ {
		if grid.Heads(all[i]) {
			heads = append(heads, i)
		}
	}
	if len(heads) == 0 {
		return
	}
	switch press {
	case "{":
		t.Cursor = heads[0]
	case "}":
		t.Cursor = heads[len(heads)-1]
	case "[":
		if k := slices.IndexFunc(heads, func(i int) bool { return i >= t.Cursor }); k > 0 {
			t.Cursor = heads[k-1]
		} else if k < 0 {
			t.Cursor = heads[len(heads)-1]
		}
	case "]":
		if k := slices.IndexFunc(heads, func(i int) bool { return i > t.Cursor }); k >= 0 {
			t.Cursor = heads[k]
		}
	}
	Fit(h)
}

// before gives the offset of the window of a pane of h lines, out of n, that ends at the line over start, its last
// line shown the one before start: the window end, which grows with the offset, first reaches start there.
func before(n, h, start int) int {
	o := start
	for o > 0 {
		if _, end, _, _ := pane.Window(n, h, o-1); end < start {
			break
		}
		o--
	}
	return o
}

// rowLines gives the first and the last of the lines the row draws, -1 for a row that draws none.
func rowLines(lines []Line, row int) (first, last int) {
	first, last = -1, -1
	for k, l := range lines {
		if l.Row == row {
			if first < 0 {
				first = k
			}
			last = k
		}
	}
	return first, last
}

// Fit scrolls the pane so the row under the cursor shows, with what it opens to when that fits, and never past the
// last line.
func Fit(h Host) {
	t, lines, height := h.Table(), ScrollLines(h), RowsHeight(h)
	t.Offset, _, _, _ = pane.Window(len(lines), height, t.Offset)
	if !h.InRows() {
		return
	}
	first, last := rowLines(lines, t.Cursor)
	if first < 0 {
		return
	}
	// Scroll until the cursor's row shows whole, or its first line and as much as the pane holds; a … line takes the
	// place of a row at either edge, so the window is asked rather than counted.
	for {
		start, end, _, _ := pane.Window(len(lines), height, t.Offset)
		switch {
		case first < start:
			t.Offset = first
		case last >= end && first > start:
			t.Offset++
		default:
			return
		}
	}
}

// SelectRow selects the row under the cursor, as one of several, or unselects it, while the rows have the keyboard;
// with keep, it selects it whether it was or not, as shift with up or down selects as it moves. On a group's heading it
// selects every row of the group, those folded away too, or, once every one is, none; another aside selects nothing.
func SelectRow(h Host, keep bool) {
	if !h.InRows() {
		return
	}
	t, rows := h.Table(), h.Rows()
	if t.Cursor >= len(rows) {
		return
	}
	keys := []string{rows[t.Cursor].Key}
	switch {
	case grid.Heads(rows[t.Cursor]):
		all := grid.Every(rows)
		keys = GroupRows(all, slices.IndexFunc(all, func(r grid.Row) bool { return r.Key == rows[t.Cursor].Key }))
	case rows[t.Cursor].Aside:
		return
	}
	every := !slices.ContainsFunc(keys, func(k string) bool { return !t.Picked[k] })
	for _, key := range keys {
		if keep || !every {
			t.Picked[key] = true
		} else {
			delete(t.Picked, key)
		}
	}
}

// SelectAll selects every row the table shows, those its closed groups fold away too, or, when every one is, none.
func SelectAll(h Host) {
	t, rows := h.Table(), grid.Every(h.Rows())
	all := len(SelectedRows(h)) == Counted(rows)
	clear(t.Picked)
	if all {
		return
	}
	for _, r := range rows {
		if !r.Aside {
			t.Picked[r.Key] = true
		}
	}
}

// SelectedRows gives the table's rows that are selected, of those it shows, its closed groups' too, each once, though
// grouping by a column of several values stands it under each.
func SelectedRows(h Host) []grid.Row {
	var out []grid.Row
	seen, t := map[string]bool{}, h.Table()
	for _, r := range grid.Every(h.Rows()) {
		if t.Picked[r.Key] && !r.Aside && !seen[r.Key] {
			seen[r.Key] = true
			out = append(out, r)
		}
	}
	return out
}

// Counted gives how many of the rows are not asides, each once, though grouping by a column of several values stands
// it under each.
func Counted(rows []grid.Row) int {
	seen := map[string]bool{}
	for _, r := range rows {
		if !r.Aside {
			seen[r.Key] = true
		}
	}
	return len(seen)
}
