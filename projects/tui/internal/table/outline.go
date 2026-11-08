// The outline: what a stage's rows open and close. A group's heading, a row, and the header are each a node of it, a
// table's open holds which stand open, and every way to open and close them goes through it alike. Right opens the node
// under the cursor, or, on a heading already open, steps into its first row; left closes it, or, on a node closed or one
// that hides nothing, steps out to the heading over it; with option, right opens one level more under it, as grow
// says, and left closes the deepest open level under it, as peel says, a level each press. A click on the ▸ or ▾ of a
// heading or a row opens or closes it, and with option a level under it, as option with right or left does. The tab's menu
// expands and collapses everything, or, while rows are selected, the selection: the selected rows expanded with the
// groups they stand in, or collapsed, or every other row collapsed and every group without a selected row folded. A
// selection reaches into a group: space on its heading selects every row in it, or none once all are, and its mark says
// whether all or some are. The levels of the groups open and close too, by the heading over the groups on the control
// line, as the heading over the groups says, each keeping its own state, as each heading does. What stands open lasts the session, and
// is no part of a preset.
package table

import (
	"maps"
	"slices"
	"strings"

	"github.com/charmbracelet/x/ansi"
	"github.com/empowerite/kit/projects/tui/internal/grid"
)

// FoldCell is the cell of the gutter a row's ▾ or ▸ stands on, past its selection's two, and the header's, which alone
// fold them.
const FoldCell = 2

// FoldAt gives the screen column, past the rows' indent for a row, the ▾ or ▸ of a row or a heading stands on, as
// internal/grid draws it, which alone opens or closes it: a row's in the gutter, a heading's past its own indent too.
func FoldAt(r grid.Row) int { return FoldsAt(r)[0] }

// FoldsAt gives the screen columns of every ▾ or ▸ that opens or closes a row or a heading, as FoldAt places the first:
// a heading that holds the headings under it on its line, across, has one before each value, each of which opens and
// closes it alike.
func FoldsAt(r grid.Row) []int {
	if !grid.Heads(r) {
		return []int{FoldCell}
	}
	var out []int
	x, before := FoldCell, " "
	for _, c := range strings.Split(ansi.Strip(r.Cells[0].Values[0]), "") {
		if before == " " && (c == grid.Unfolded || c == grid.Folded) {
			out = append(out, x)
		}
		x, before = x+ansi.StringWidth(c), c
	}
	return out
}

// HeaderKey is the header's key in a table's outline: open, every line of names that take more than one shows.
const HeaderKey = "\x00header"

// every gives the table's whole outline: its rows shown, each closed group's rows after its heading.
func every(h Host) []grid.Row { return grid.Every(h.Rows()) }

// Opens says whether the node at j of the outline hides something to open: a group's heading always, but the one over
// the groups, which never closes, its levels folding under it; a row while its lines do not all show, laid out as the
// table lays its rows out.
func Opens(h Host, every []grid.Row, j int) bool {
	if grid.OverGroups(every[j]) {
		return false
	}
	return grid.Heads(every[j]) || grid.Opens(every, h.Names(), j, h.GridWidth())
}

// Reoutline gives the table the outline next, and builds its rows again: the cursor stays on its row, under
// the same headings, as a row grouped under several values stands under each; or, once that is folded away, on the
// nearest heading over it that shows.
func Reoutline(h Host, next grid.Outline) {
	s := h.Table()
	// chain gives the keys of the node at k and of each heading over it, the node's first
	chain := func(rows []grid.Row, k int) []string {
		var out []string
		for ; k >= 0 && k < len(rows); k = grid.Parent(rows, k) {
			out = append(out, rows[k].Key)
		}
		return out
	}
	was := chain(h.Rows(), s.Cursor)
	s.Open = next
	h.Rebuild()
	rows := h.Rows()
	// the cursor's row under the same headings, else the nearest heading over it that shows
	found := func() int {
		for at := range was {
			for k := range rows {
				if rows[k].Key == was[at] && slices.Equal(chain(rows, k), was[at:]) {
					return k
				}
			}
		}
		return -1
	}
	if k := found(); k >= 0 {
		s.Cursor = k
	}
	h.Settle()
}

// Disclose opens the node under the cursor, or closes it, and with subtree every node under it as well, and says
// whether that changed anything; a node that hides nothing stays as it is.
func Disclose(h Host, open, subtree bool) bool {
	if !h.InRows() {
		return false
	}
	s, rows := h.Table(), h.Rows()
	if s.Cursor >= len(rows) {
		return false
	}
	every := every(h)
	at := slices.IndexFunc(every, func(r grid.Row) bool { return r.Key == rows[s.Cursor].Key })
	end := at + 1
	if subtree {
		end = grid.Span(every, at)
	}
	next, changed := maps.Clone(s.Open), false
	for j := at; j < end; j++ {
		if Opens(h, every, j) && next.Unfolds(every[j]) != open {
			UnfoldNode(next, every[j], open)
			changed = true
		}
	}
	if changed {
		Reoutline(h, next)
	}
	return changed
}

// UnfoldNode opens or closes a node in an outline, as its ▸ or ▾ shows it: a heading opened shows the level under it too,
// and every level over that, so what it holds shows, though that shows the level under every other heading open too.
func UnfoldNode(next grid.Outline, r grid.Row, open bool) {
	next[r.Key] = open
	if open && grid.Heads(r) {
		for d := 0; d < grid.Level(r); d++ {
			delete(next, grid.LevelKey(d))
		}
	}
}

// Expand takes right, as open says, or left, and with option either, as subtree says: the node under the cursor opened
// or closed, as Disclose says; with option, a level more opened under it, as Grow says, or the deepest open level
// under it closed, as Peel says; or, where that changes nothing and option is not held, right on a heading steps into
// its first row, and left steps out to the heading over the node.
func Expand(h Host, open, subtree bool) {
	switch {
	case subtree && open:
		Grow(h)
		return
	case subtree:
		Peel(h)
		return
	}
	if Disclose(h, open, subtree) || !h.InRows() {
		return
	}
	s, rows := h.Table(), h.Rows()
	if s.Cursor >= len(rows) {
		return
	}
	switch next := s.Cursor + 1; {
	case open && next < len(rows) && grid.Parent(rows, next) == s.Cursor:
		s.Cursor = next
	case !open && grid.Parent(rows, s.Cursor) >= First(rows):
		s.Cursor = grid.Parent(rows, s.Cursor)
	default:
		return
	}
	h.Settle()
}

// under gives the outline, the index of the node under the cursor in it, and how far under that node the node at j
// stands, with whether every node between them stands open, so that it shows; at is -1 with no node under the cursor.
func under(h Host) (all []grid.Row, at int, depth func(j int) (int, bool)) {
	if !h.InRows() {
		return nil, -1, nil
	}
	s, rows := h.Table(), h.Rows()
	if s.Cursor >= len(rows) {
		return nil, -1, nil
	}
	all = every(h)
	at = slices.IndexFunc(all, func(r grid.Row) bool { return r.Key == rows[s.Cursor].Key })
	depth = func(j int) (int, bool) {
		n, shows := 0, true
		for p := grid.Parent(all, j); p >= at; p = grid.Parent(all, p) {
			n++
			shows = shows && s.Open.Unfolds(all[p])
		}
		return n, shows
	}
	return all, at, depth
}

// Grow opens one level more under the node under the cursor, as option with right asks: the deepest open node under
// it that shows, the node itself counting, sets the level, and every node down to the level under that opens, so
// each press shows one level more, every branch alike; with none open, the node itself opens. It says whether that
// changed anything.
func Grow(h Host) bool {
	every, at, depth := under(h)
	if at < 0 {
		return false
	}
	s := h.Table()
	deepest := -1
	for j := at; j < grid.Span(every, at); j++ {
		// the heading over the groups counts as open, as it always stands so
		if d, shows := depth(j); shows && s.Open.Unfolds(every[j]) && (Opens(h, every, j) || grid.OverGroups(every[j])) {
			deepest = max(deepest, d)
		}
	}
	next, changed := maps.Clone(s.Open), false
	for j := at; j < grid.Span(every, at); j++ {
		if d, _ := depth(j); d <= deepest+1 && Opens(h, every, j) && !next.Unfolds(every[j]) {
			UnfoldNode(next, every[j], true)
			changed = true
		}
	}
	if changed {
		Reoutline(h, next)
	}
	return changed
}

// Peel closes the deepest open nodes under the one under the cursor, of those that show, one level of them, as option
// with left asks: from everything open, the rows close first, then the innermost groups' headings, and so outward, a
// level each press, the node itself last. It says whether that changed anything.
func Peel(h Host) bool {
	every, at, depth := under(h)
	if at < 0 {
		return false
	}
	s := h.Table()
	deepest, closing := -1, []int(nil)
	for j := at; j < grid.Span(every, at); j++ {
		d, shows := depth(j)
		if !shows || !Opens(h, every, j) || !s.Open.Unfolds(every[j]) || d < deepest {
			continue
		}
		if d > deepest {
			deepest, closing = d, nil
		}
		closing = append(closing, j)
	}
	if len(closing) == 0 {
		return false
	}
	next := maps.Clone(s.Open)
	for _, j := range closing {
		next[every[j].Key] = false
	}
	Reoutline(h, next)
	return true
}

// GroupRows gives the keys of the rows a group's heading at j of the outline stands over, its headings apart.
func GroupRows(every []grid.Row, j int) []string {
	var keys []string
	for _, r := range every[j+1 : grid.Span(every, j)] {
		if !r.Aside {
			keys = append(keys, r.Key)
		}
	}
	return keys
}

// GroupMarks gives the mark of each group's heading of a table's whole outline, by key, while some of its rows are
// picked: Chosen while all are, Partly while only some.
func GroupMarks(every []grid.Row, picked map[string]bool) map[string]string {
	marks := map[string]string{}
	for j, r := range every {
		if !grid.Heads(r) {
			continue
		}
		keys := GroupRows(every, j)
		switch n := len(slices.DeleteFunc(slices.Clone(keys), func(k string) bool { return !picked[k] })); {
		case n > 0 && n == len(keys):
			marks[r.Key] = grid.Chosen
		case n > 0:
			marks[r.Key] = grid.Partly
		}
	}
	return marks
}

// Folding gives the outline one of the fold items leaves the table with, and whether the name is
// one: Expand All opens every node and every level of the groups, the header too where it hides names; Collapse All closes every one, down to the
// outermost headings; Expand Selected opens the selected rows, every heading over them, and every level, so they show;
// Collapse
// Selected closes the selected rows; and Collapse Others closes every row not selected, and every heading over none.
func Folding(h Host, name string) (grid.Outline, bool) {
	if !slices.Contains([]string{"Expand All", "Collapse All", "Expand Selected", "Collapse Selected", "Collapse Others"},
		name) {
		return nil, false
	}
	s, every := h.Table(), every(h)
	next := maps.Clone(s.Open)
	over := map[string]bool{} // the headings over a selected row
	for j, r := range every {
		if s.Picked[r.Key] && !r.Aside {
			for p := grid.Parent(every, j); p >= 0; p = grid.Parent(every, p) {
				over[every[p].Key] = true
			}
		}
	}
	if name == "Expand All" || name == "Expand Selected" {
		ShowLevels(next, len(h.Panel().Groups()))
	}
	for j, r := range every {
		if !Opens(h, every, j) {
			continue
		}
		picked := s.Picked[r.Key] && !r.Aside
		switch name {
		case "Expand All":
			next[r.Key] = true
		case "Collapse All":
			next[r.Key] = false
		case "Expand Selected":
			if picked || over[r.Key] {
				next[r.Key] = true
			}
		case "Collapse Selected":
			if picked {
				next[r.Key] = false
			}
		case "Collapse Others":
			if grid.Heads(r) || !picked {
				next[r.Key] = over[r.Key]
			}
		}
	}
	switch name {
	case "Expand All":
		next[HeaderKey] = h.HeaderHides()
	case "Collapse All":
		next[HeaderKey] = false
	}
	return next, true
}

// Changes says whether an outline stands the table's nodes, or the levels of its groups, otherwise than they
// stand now.
func Changes(h Host, next grid.Outline) bool {
	s, every := h.Table(), every(h)
	for j, r := range every {
		if Opens(h, every, j) && next.Unfolds(r) != s.Open.Unfolds(r) {
			return true
		}
	}
	for d := 0; d <= len(h.Panel().Groups()); d++ {
		if next.Shows(d) != s.Open.Shows(d) {
			return true
		}
	}
	return next[HeaderKey] != s.Open[HeaderKey]
}

// ShowLevels opens every level of the groups in an outline, of so many groups, the rows' too, as Expand asks, so
// nothing a level hides stays hidden.
func ShowLevels(next grid.Outline, groups int) {
	for d := 0; d <= groups; d++ {
		delete(next, grid.LevelKey(d))
	}
}

// First gives the first of a table's rows the cursor can stand on: past the heading over the groups, which the control
// line draws, not the rows.
func First(rows []grid.Row) int {
	if len(rows) > 0 && grid.OverGroups(rows[0]) {
		return 1
	}
	return 0
}
