// The outline: what a stage's rows open and close. A group's heading, a row, and the header are each a node of it, a
// table's open holds which stand open, and every way to open and close them goes through it alike, by two primitives.
// Show and Hide open or close one node and nothing under it, so Hide then Show puts back what showed: right shows the
// node under the cursor, or, on a heading already open, steps into its first row; left hides it, or, on a node hidden
// or one that hides nothing, steps out to the heading over it; a click on the ▸ or ▾ of a heading or a row does
// either. More and Less work a level at a time: More shows the shallowest level where something hides, all of it,
// and hides nothing; Less hides the deepest level open. > and < and the tab's menu take them over the whole table, and
// option with right or left, or with a click on a ▸ or ▾, over the node, where the terminal sends it. The tab's menu
// also shows the selected rows, every heading over them first, and hides them, and nothing over them. A
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
// one for each, as a heading that holds the headings under it on its line, across, has one fold, before its first
// value, and a separator before each value after it.
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

// Disclose shows the node under the cursor, as open says, or hides it, as Show and Hide ask, and says whether that
// changed anything; a node that hides nothing stays as it is.
func Disclose(h Host, open bool) bool {
	if !h.InRows() {
		return false
	}
	s, rows := h.Table(), h.Rows()
	if s.Cursor >= len(rows) {
		return false
	}
	every := every(h)
	at := slices.IndexFunc(every, func(r grid.Row) bool { return r.Key == rows[s.Cursor].Key })
	if at < 0 || !Opens(h, every, at) || s.Open.Unfolds(every[at]) == open {
		return false
	}
	next := maps.Clone(s.Open)
	if open {
		show(next, every, at)
	} else {
		next[every[at].Key] = false
	}
	Reoutline(h, next)
	return true
}

// show opens the node at j of the outline in next, and changes nothing else that shows, so that Hide then Show puts
// back what showed: under it, every node stands as it stood. Where the heading over the groups folds a level over
// it, as LevelKey says, the level opens again, and every heading of the level it folds that stood open, hidden with
// it, closes, so it stays hidden, but the node and the headings over it; what the heading over the groups holds
// shows that way as a level of closed headings.
func show(next grid.Outline, every []grid.Row, j int) {
	r := every[j]
	next[r.Key] = true
	if !grid.Heads(r) {
		return
	}
	chain := map[string]bool{}
	for p := j; p >= 0; p = grid.Parent(every, p) {
		chain[every[p].Key] = true
	}
	for d := 0; d < grid.Level(r); d++ {
		if open, ok := next[grid.LevelKey(d)]; !ok || open {
			continue
		}
		// a level closed folds what the headings of the level over it hold; the heading over the groups, at level 1,
		// never closes, so under it the headings it shows close instead
		folds := max(d+1, 2)
		for _, o := range every {
			if grid.Heads(o) && !grid.OverGroups(o) && grid.Level(o) == folds && !chain[o.Key] && next.Open(o) {
				next[o.Key] = false
			}
		}
		next[grid.LevelKey(d)] = true
	}
}

// unfolds says whether a node shows what it holds, as Unfolds says, but the heading over the groups only while the
// outermost level shows, as it holds that level.
func unfolds(o grid.Outline, r grid.Row) bool {
	return o.Unfolds(r) && (!grid.OverGroups(r) || o.Shows(grid.Level(r)-1))
}

// Expand takes right, as open says, or left, and with option either, as subtree says: the node under the cursor shown
// or hidden, as Disclose says; with option, a level more under it, as Grow says, or a level less, as Peel says; or,
// where that changes nothing and option is not held, right on a heading steps into its first row, and left steps out
// to the heading over the node.
func Expand(h Host, open, subtree bool) {
	switch {
	case subtree && open:
		Grow(h)
		return
	case subtree:
		Peel(h)
		return
	}
	if Disclose(h, open) || !h.InRows() {
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

// scope gives the outline and the nodes ShowMore and ShowLess work over, from up to to: the whole table's, as whole
// says, or the node under the cursor's, it and every node under it, none with no node under the cursor; and how deep
// under them the node at j stands, the heading over the groups not counted, with whether every node over it among them
// shows what it holds, so that it shows.
func scope(h Host, whole bool) (all []grid.Row, from, to int, depth func(j int) (int, bool)) {
	s := h.Table()
	all = every(h)
	if whole {
		to = len(all)
	} else {
		rows := h.Rows()
		if !h.InRows() || s.Cursor >= len(rows) {
			return nil, 0, 0, nil
		}
		// every row shown stands in the whole outline, as grid.Every gives it
		from = slices.IndexFunc(all, func(r grid.Row) bool { return r.Key == rows[s.Cursor].Key })
		to = grid.Span(all, from)
	}
	depth = func(j int) (int, bool) {
		n, shows := 0, true
		for p := grid.Parent(all, j); p >= from; p = grid.Parent(all, p) {
			// the heading over the groups, on the control line, is no level of what the rows show
			if !grid.OverGroups(all[p]) {
				n++
			}
			shows = shows && unfolds(s.Open, all[p])
		}
		return n, shows
	}
	return all, from, to, depth
}

// ShowMore gives the outline with a level more shown, of the whole table's nodes, as whole says, or the node under the
// cursor's, and whether there is one: the shallowest level where a node that shows hides something, every such node
// of it shown, as Show shows one, and nothing hidden, so a node under them stands as it stood. The heading over the
// groups counts, while the outermost level is folded, so More shows it again, though Less never folds it.
func ShowMore(h Host, whole bool) (grid.Outline, bool) {
	all, from, to, depth := scope(h, whole)
	s := h.Table()
	shallowest, showing := -1, []int(nil)
	for j := from; j < to; j++ {
		d, shows := depth(j)
		opens := Opens(h, all, j) || grid.OverGroups(all[j])
		if !shows || !opens || unfolds(s.Open, all[j]) || shallowest >= 0 && d > shallowest {
			continue
		}
		if shallowest < 0 || d < shallowest {
			shallowest, showing = d, nil
		}
		showing = append(showing, j)
	}
	if len(showing) == 0 {
		return nil, false
	}
	next := maps.Clone(s.Open)
	for _, j := range showing {
		show(next, all, j)
	}
	return next, true
}

// ShowLess gives the outline with a level less shown, of the whole table's nodes, as whole says, or the node under the
// cursor's, and whether there is one: the deepest level where a node that shows stands open, every such node of it
// hidden. From everything open, the rows hide first, then the innermost groups' headings, and so outward.
func ShowLess(h Host, whole bool) (grid.Outline, bool) {
	all, from, to, depth := scope(h, whole)
	s := h.Table()
	deepest, hiding := -1, []int(nil)
	for j := from; j < to; j++ {
		d, shows := depth(j)
		if !shows || !Opens(h, all, j) || !s.Open.Unfolds(all[j]) || d < deepest {
			continue
		}
		if d > deepest {
			deepest, hiding = d, nil
		}
		hiding = append(hiding, j)
	}
	if len(hiding) == 0 {
		return nil, false
	}
	next := maps.Clone(s.Open)
	for _, j := range hiding {
		next[all[j].Key] = false
	}
	return next, true
}

// Grow shows a level more under the node under the cursor, as ShowMore says, as option with right asks, the node itself
// first, and says whether that changed anything.
func Grow(h Host) bool {
	next, ok := ShowMore(h, false)
	if ok {
		Reoutline(h, next)
	}
	return ok
}

// Peel shows a level less under the node under the cursor, as ShowLess says, as option with left asks, the node itself
// last, and says whether that changed anything.
func Peel(h Host) bool {
	next, ok := ShowLess(h, false)
	if ok {
		Reoutline(h, next)
	}
	return ok
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

// Fold takes > and <, a level more and a level less of the whole table, as Show More and Show Less do, and says
// whether the key is one of them. A key that would open or close nothing is taken all the same, and does nothing.
func Fold(h Host, key string) bool {
	name := ""
	switch key {
	case ">":
		name = "Show More"
	case "<":
		name = "Show Less"
	default:
		return false
	}
	if next, _ := Folding(h, name); Changes(h, next) {
		Reoutline(h, next)
	}
	return true
}

// Folding gives the outline one of the fold items of the tab's menu leaves the table with, and whether the name is
// one: Show More and Show Less a level more and a level less of the whole table, as ShowMore and ShowLess say, as > and <
// do; Show Selected shows the selected rows, every heading over them first, as Show shows a node; and Hide Selected
// hides the selected rows, and nothing over them, so the rows beside them stay as they stand.
func Folding(h Host, name string) (grid.Outline, bool) {
	s := h.Table()
	switch name {
	case "Show More", "Show Less":
		level := ShowMore
		if name == "Show Less" {
			level = ShowLess
		}
		if next, ok := level(h, true); ok {
			return next, true
		}
		return maps.Clone(s.Open), true
	case "Show Selected", "Hide Selected":
	default:
		return nil, false
	}
	every, next := every(h), maps.Clone(s.Open)
	over := map[string]bool{} // the headings over a selected row
	for j, r := range every {
		if s.Picked[r.Key] && !r.Aside {
			for p := grid.Parent(every, j); p >= 0; p = grid.Parent(every, p) {
				over[every[p].Key] = true
			}
		}
	}
	// the outline runs outermost first, so a heading shows before the headings and rows under it
	for j, r := range every {
		picked := s.Picked[r.Key] && !r.Aside
		switch {
		case !Opens(h, every, j):
		case name == "Hide Selected" && picked:
			next[r.Key] = false
		case name == "Show Selected" && (picked || over[r.Key]) && !next.Unfolds(r):
			show(next, every, j)
		}
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

// First gives the first of a table's rows the cursor can stand on: past the heading over the groups, which the control
// line draws, not the rows.
func First(rows []grid.Row) int {
	if len(rows) > 0 && grid.OverGroups(rows[0]) {
		return 1
	}
	return 0
}
