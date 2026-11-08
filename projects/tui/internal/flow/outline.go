// The outline: what a stage's rows open and close. A group's heading, a row, and the header are each a node of it, a
// stage's open holds which stand open, and every way to open and close them goes through it alike. Right opens the node
// under the cursor, or, on a heading already open, steps into its first row; left closes it, or, on a node closed or one
// that hides nothing, steps out to the heading over it; with option, right opens one level more under it, as grow
// says, and left closes the deepest open level under it, as peel says, a level each press. A click on the ▸ or ▾ of a
// heading or a row opens or closes it, and with option a level under it, as option with right or left does. The tab's menu
// expands and collapses everything, or, while rows are selected, the selection: the selected rows expanded with the
// groups they stand in, or collapsed, or every other row collapsed and every group without a selected row folded. A
// selection reaches into a group: space on its heading selects every row in it, or none once all are, and its mark says
// whether all or some are. The levels of the groups open and close too, by the heading over the groups on the control
// line, as heading.go says, each keeping its own state, as each heading does. What stands open lasts the session, and
// is no part of a preset.
package flow

import (
	"maps"
	"slices"
	"strings"

	"github.com/charmbracelet/x/ansi"

	"github.com/empowerite/kit/projects/tui/internal/grid"
	"github.com/empowerite/kit/projects/tui/internal/shell"
)

// foldCell is the cell of the gutter a row's ▾ or ▸ stands on, past its selection's two, and the header's, which alone
// fold them.
const foldCell = 2

// foldAt gives the screen column, past the rows' indent for a row, the ▾ or ▸ of a row or a heading stands on, as
// internal/grid draws it, which alone opens or closes it: a row's in the gutter, a heading's past its own indent too.
func foldAt(r row) int { return foldsAt(r)[0] }

// foldsAt gives the screen columns of every ▾ or ▸ that opens or closes a row or a heading, as foldAt places the first:
// a heading that holds the headings under it on its line, across, has one before each value, each of which opens and
// closes it alike.
func foldsAt(r row) []int {
	if !grid.Heads(r) {
		return []int{foldCell}
	}
	var out []int
	x, before := foldCell, " "
	for _, c := range strings.Split(ansi.Strip(r.Cells[0].Values[0]), "") {
		if before == " " && (c == grid.Unfolded || c == grid.Folded) {
			out = append(out, x)
		}
		x, before = x+ansi.StringWidth(c), c
	}
	return out
}

// headerKey is the header's key in a stage's outline: open, every line of names that take more than one shows.
const headerKey = "\x00header"

// every gives a stage's whole outline: its rows shown, each closed group's rows after its heading.
func (m Model) every(i int) []row { return grid.Every(m.rows(i)) }

// opens says whether the node at j of the outline hides something to open: a group's heading always, but the one over
// the groups, which never closes, its levels folding under it; a row while its lines do not all show, laid out as the
// selected stage lays its rows out.
func (m Model) opens(every []row, j int) bool {
	if grid.OverGroups(every[j]) {
		return false
	}
	return grid.Heads(every[j]) || grid.Opens(every, m.names(m.stage), j, m.gridWidth(m.stage))
}

// reoutline gives the selected stage the outline next, and builds its rows again: the cursor stays on its row, under
// the same headings, as a row grouped under several values stands under each; or, once that is folded away, on the
// nearest heading over it that shows.
func (m *Model) reoutline(next grid.Outline) {
	s := &m.stages[m.stage]
	// chain gives the keys of the node at k and of each heading over it, the node's first
	chain := func(rows []row, k int) []string {
		var out []string
		for ; k >= 0 && k < len(rows); k = grid.Parent(rows, k) {
			out = append(out, rows[k].Key)
		}
		return out
	}
	was := chain(m.rows(m.stage), s.cursor)
	s.open = next
	m.rebuild()
	rows := m.rows(m.stage)
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
		s.cursor = k
	}
	m.clamp(m.stage)
	m.fit()
}

// disclose opens the node under the cursor, or closes it, and with subtree every node under it as well, and says
// whether that changed anything; a node that hides nothing stays as it is.
func (m *Model) disclose(open, subtree bool) bool {
	if !m.inRows || len(m.stages) == 0 {
		return false
	}
	s, rows := &m.stages[m.stage], m.rows(m.stage)
	if s.cursor >= len(rows) {
		return false
	}
	every := m.every(m.stage)
	at := slices.IndexFunc(every, func(r row) bool { return r.Key == rows[s.cursor].Key })
	end := at + 1
	if subtree {
		end = grid.Span(every, at)
	}
	next, changed := maps.Clone(s.open), false
	for j := at; j < end; j++ {
		if m.opens(every, j) && next.Unfolds(every[j]) != open {
			unfoldNode(next, every[j], open)
			changed = true
		}
	}
	if changed {
		m.reoutline(next)
	}
	return changed
}

// unfoldNode opens or closes a node in an outline, as its ▸ or ▾ shows it: a heading opened shows the level under it too,
// and every level over that, so what it holds shows, though that shows the level under every other heading open too.
func unfoldNode(next grid.Outline, r row, open bool) {
	next[r.Key] = open
	if open && grid.Heads(r) {
		for d := 0; d < grid.Level(r); d++ {
			delete(next, grid.LevelKey(d))
		}
	}
}

// expand takes right, as open says, or left, and with option either, as subtree says: the node under the cursor opened
// or closed, as disclose says; with option, a level more opened under it, as grow says, or the deepest open level
// under it closed, as peel says; or, where that changes nothing and option is not held, right on a heading steps into
// its first row, and left steps out to the heading over the node.
func (m *Model) expand(open, subtree bool) {
	switch {
	case subtree && open:
		m.grow()
		return
	case subtree:
		m.peel()
		return
	}
	if m.disclose(open, subtree) || subtree || !m.inRows || len(m.stages) == 0 {
		return
	}
	s, rows := &m.stages[m.stage], m.rows(m.stage)
	if s.cursor >= len(rows) {
		return
	}
	switch next := s.cursor + 1; {
	case open && next < len(rows) && grid.Parent(rows, next) == s.cursor:
		s.cursor = next
	case !open && grid.Parent(rows, s.cursor) >= m.first(m.stage):
		s.cursor = grid.Parent(rows, s.cursor)
	default:
		return
	}
	m.fit()
}

// under gives the outline, the index of the node under the cursor in it, and how far under that node the node at j
// stands, with whether every node between them stands open, so that it shows; at is -1 with no node under the cursor.
func (m Model) under() (every []row, at int, depth func(j int) (int, bool)) {
	if !m.inRows || len(m.stages) == 0 {
		return nil, -1, nil
	}
	s, rows := m.stages[m.stage], m.rows(m.stage)
	if s.cursor >= len(rows) {
		return nil, -1, nil
	}
	every = m.every(m.stage)
	at = slices.IndexFunc(every, func(r row) bool { return r.Key == rows[s.cursor].Key })
	depth = func(j int) (int, bool) {
		n, shows := 0, true
		for p := grid.Parent(every, j); p >= at; p = grid.Parent(every, p) {
			n++
			shows = shows && s.open.Unfolds(every[p])
		}
		return n, shows
	}
	return every, at, depth
}

// grow opens one level more under the node under the cursor, as option with right asks: the deepest open node under
// it that shows, the node itself counting, sets the level, and every node down to the level under that opens, so
// each press shows one level more, every branch alike; with none open, the node itself opens. It says whether that
// changed anything.
func (m *Model) grow() bool {
	every, at, depth := m.under()
	if at < 0 {
		return false
	}
	s := &m.stages[m.stage]
	deepest := -1
	for j := at; j < grid.Span(every, at); j++ {
		// the heading over the groups counts as open, as it always stands so
		if d, shows := depth(j); shows && s.open.Unfolds(every[j]) && (m.opens(every, j) || grid.OverGroups(every[j])) {
			deepest = max(deepest, d)
		}
	}
	next, changed := maps.Clone(s.open), false
	for j := at; j < grid.Span(every, at); j++ {
		if d, _ := depth(j); d <= deepest+1 && m.opens(every, j) && !next.Unfolds(every[j]) {
			unfoldNode(next, every[j], true)
			changed = true
		}
	}
	if changed {
		m.reoutline(next)
	}
	return changed
}

// peel closes the deepest open nodes under the one under the cursor, of those that show, one level of them, as option
// with left asks: from everything open, the rows close first, then the innermost groups' headings, and so outward, a
// level each press, the node itself last. It says whether that changed anything.
func (m *Model) peel() bool {
	every, at, depth := m.under()
	if at < 0 {
		return false
	}
	s := &m.stages[m.stage]
	deepest, closing := -1, []int(nil)
	for j := at; j < grid.Span(every, at); j++ {
		d, shows := depth(j)
		if !shows || !m.opens(every, j) || !s.open.Unfolds(every[j]) || d < deepest {
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
	next := maps.Clone(s.open)
	for _, j := range closing {
		next[every[j].Key] = false
	}
	m.reoutline(next)
	return true
}

// groupRows gives the keys of the rows a group's heading at j of the outline stands over, its headings apart.
func groupRows(every []row, j int) []string {
	var keys []string
	for _, r := range every[j+1 : grid.Span(every, j)] {
		if !r.Aside {
			keys = append(keys, r.Key)
		}
	}
	return keys
}

// groupMarks gives the mark of each group's heading of a stage, by key, while some of its rows are selected: Chosen
// while all are, Partly while only some.
func (m Model) groupMarks(i int) map[string]string {
	s, every := m.stages[i], m.every(i)
	marks := map[string]string{}
	for j, r := range every {
		if !grid.Heads(r) {
			continue
		}
		keys := groupRows(every, j)
		switch n := len(slices.DeleteFunc(slices.Clone(keys), func(k string) bool { return !s.picked[k] })); {
		case n > 0 && n == len(keys):
			marks[r.Key] = grid.Chosen
		case n > 0:
			marks[r.Key] = grid.Partly
		}
	}
	return marks
}

// folds gives the tab's menu's items that open and close the outline, named for what they act on: everything, or,
// while rows are selected, the selection; Collapse Others stands always, dim while nothing is selected.
func (m Model) folds() []shell.Item {
	scope := "All"
	if m.selected() > 0 {
		scope = "Selected"
	}
	return []shell.Item{{Glyph: Expand, Name: "Expand " + scope}, {Glyph: Collapse, Name: "Collapse " + scope},
		{Glyph: Collapse, Name: "Collapse Others"}}
}

// folding gives the outline one of the tab's menu's fold items leaves the selected stage with, and whether the name is
// one: Expand All opens every node and every level of the groups, the header too where it hides names; Collapse All closes every one, down to the
// outermost headings; Expand Selected opens the selected rows, every heading over them, and every level, so they show;
// Collapse
// Selected closes the selected rows; and Collapse Others closes every row not selected, and every heading over none.
func (m Model) folding(name string) (grid.Outline, bool) {
	if !slices.Contains([]string{"Expand All", "Collapse All", "Expand Selected", "Collapse Selected", "Collapse Others"},
		name) {
		return nil, false
	}
	s, every := m.stages[m.stage], m.every(m.stage)
	next := maps.Clone(s.open)
	over := map[string]bool{} // the headings over a selected row
	for j, r := range every {
		if s.picked[r.Key] && !r.Aside {
			for p := grid.Parent(every, j); p >= 0; p = grid.Parent(every, p) {
				over[every[p].Key] = true
			}
		}
	}
	if name == "Expand All" || name == "Expand Selected" {
		m.showLevels(next)
	}
	for j, r := range every {
		if !m.opens(every, j) {
			continue
		}
		picked := s.picked[r.Key] && !r.Aside
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
	switch _, hides := m.headerLines(); name {
	case "Expand All":
		next[headerKey] = hides
	case "Collapse All":
		next[headerKey] = false
	}
	return next, true
}

// changes says whether an outline stands the selected stage's nodes, or the levels of its groups, otherwise than they
// stand now.
func (m Model) changes(next grid.Outline) bool {
	s, every := m.stages[m.stage], m.every(m.stage)
	for j, r := range every {
		if m.opens(every, j) && next.Unfolds(r) != s.open.Unfolds(r) {
			return true
		}
	}
	for d := 0; d <= len(m.panel(m.stage).Groups()); d++ {
		if next.Shows(d) != s.open.Shows(d) {
			return true
		}
	}
	return next[headerKey] != s.open[headerKey]
}
