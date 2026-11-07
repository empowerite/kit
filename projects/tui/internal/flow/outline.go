// The outline: what a stage's rows open and close. A group's heading, a row, and the header are each a node of it, a
// stage's open holds which stand open, and every way to open and close them goes through it alike. Right opens the node
// under the cursor, or, on a heading already open, steps into its first row; left closes it, or, on a node closed or one
// that hides nothing, steps out to the heading over it; with option, each opens or closes every node under it too. A
// click on a heading, or on a row's ▸ or ▾, opens or closes it, and with option everything under it. The tab's menu
// expands and collapses everything, or, while rows are selected, the selection: the selected rows expanded with the
// groups they stand in, or collapsed, or every other row collapsed and every group without a selected row folded. A
// selection reaches into a group: space on its heading selects every row in it, or none once all are, and its mark says
// whether all or some are. What stands open lasts the session, and is no part of a preset.
package flow

import (
	"maps"
	"slices"

	"github.com/empowerite/kit/projects/tui/internal/grid"
	"github.com/empowerite/kit/projects/tui/internal/shell"
)

// headerKey is the header's key in a stage's outline: open, every line of names that take more than one shows.
const headerKey = "\x00header"

// every gives a stage's whole outline: its rows shown, each closed group's rows after its heading.
func (m Model) every(i int) []row { return grid.Every(m.rows(i)) }

// opens says whether the node at j of the outline hides something to open: a group's heading always, a row while its
// lines do not all show, laid out as the selected stage lays its rows out.
func (m Model) opens(every []row, j int) bool {
	return grid.Heads(every[j]) || grid.Opens(every, m.names(m.stage), j, m.gridWidth(m.stage))
}

// reoutline gives the selected stage the outline next, and builds its rows again: the cursor stays on its row, or, once
// that is folded away, on the nearest heading over it that shows.
func (m *Model) reoutline(next grid.Outline) {
	s := &m.stages[m.stage]
	rows := m.rows(m.stage)
	var chain []string
	for k := s.cursor; k >= 0 && k < len(rows); k = grid.Parent(rows, k) {
		chain = append(chain, rows[k].Key)
	}
	s.open = next
	m.rebuild()
	rows = m.rows(m.stage)
	for _, key := range chain {
		if k := slices.IndexFunc(rows, func(r row) bool { return r.Key == key }); k >= 0 {
			s.cursor = k
			break
		}
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
		if m.opens(every, j) && next.Open(every[j]) != open {
			next[every[j].Key], changed = open, true
		}
	}
	if changed {
		m.reoutline(next)
	}
	return changed
}

// expand takes right, as open says, or left, and with option either, as subtree says: the node under the cursor opened
// or closed, as disclose says; or, where that changes nothing and option is not held, right on a heading steps into its
// first row, and left steps out to the heading over the node.
func (m *Model) expand(open, subtree bool) {
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
	case !open && grid.Parent(rows, s.cursor) >= 0:
		s.cursor = grid.Parent(rows, s.cursor)
	default:
		return
	}
	m.fit()
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
// one: Expand All opens every node, the header too where it hides names; Collapse All closes every one, down to the
// outermost headings; Expand Selected opens the selected rows and every heading over them, so they show; Collapse
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
	switch _, hides := m.header(); name {
	case "Expand All":
		next[headerKey] = hides
	case "Collapse All":
		next[headerKey] = false
	}
	return next, true
}

// changes says whether an outline stands the selected stage's nodes otherwise than they stand now.
func (m Model) changes(next grid.Outline) bool {
	s, every := m.stages[m.stage], m.every(m.stage)
	for j, r := range every {
		if m.opens(every, j) && next.Open(r) != s.open.Open(r) {
			return true
		}
	}
	return next[headerKey] != s.open[headerKey]
}
