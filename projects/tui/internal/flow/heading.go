// The heading over the groups stands on the control line, the pane's first, apart from the rows: it names the columns
// the rows group by, and a click on a name drops the column's own menu around it, as a name in the header does. The ▾
// or ▸ before a column's name opens or closes its level, and the one after the last name the rows under the groups, as
// toggleLevel says: a level closed hides everything under it, and each level and each heading keeps its own state, so
// opening it again shows what was there as it was. A value's heading folds by its own ▾ or ▸, as any row does.
package flow

import (
	"maps"
	"slices"
	"strings"

	"github.com/charmbracelet/x/ansi"

	"github.com/empowerite/kit/projects/tui/internal/grid"
)

// barLeft is the screen column the heading over the groups starts on, on the control line: past the gutter's two
// cells, as a heading's mark takes them.
const barLeft = 2

// first gives the first of a stage's rows the cursor can stand on: past the heading over the groups, which the control
// line draws, not the rows.
func (m Model) first(i int) int {
	if rows := m.rows(i); len(rows) > 0 && grid.OverGroups(rows[0]) {
		return 1
	}
	return 0
}

// barText gives the heading over the selected stage's groups, as internal/grid draws it, plain of its styles or not;
// nothing where the rows group by nothing.
func (m Model) barText() string {
	if len(m.stages) == 0 {
		return ""
	}
	if rows := m.rows(m.stage); len(rows) > 0 && grid.OverGroups(rows[0]) {
		return rows[0].Cells[0].Values[0]
	}
	return ""
}

// nameSpot is a column's name in the heading over the groups, and the screen columns it stands on, from its first to
// the one past it.
type nameSpot struct {
	column      string
	from, until int
}

// barNames gives the columns the heading over the groups names, each where the control line draws its name, as
// namesIn finds them.
func (m Model) barNames() []nameSpot {
	return namesIn(ansi.Strip(m.barText()), m.panel(m.stage).Groups())
}

// namesIn gives where the heading over the groups, its text plain, names each of the groups' columns on the control
// line, from barLeft: before the next level's fold, the count, or the heading's end, so a value that holds a column's
// name is passed over.
func namesIn(text string, groups []string) []nameSpot {
	var out []nameSpot
	for _, column := range groups {
		for from := 0; ; {
			k := strings.Index(text[from:], " "+column)
			if k < 0 {
				break
			}
			at := from + k + 1
			if end := at + len(column); end == len(text) || strings.HasPrefix(text[end:], "  ") ||
				strings.HasPrefix(text[end:], " "+grid.Unfolded) || strings.HasPrefix(text[end:], " "+grid.Folded) {
				x := barLeft + ansi.StringWidth(text[:at])
				out = append(out, nameSpot{column, x, x + ansi.StringWidth(column)})
				break
			}
			from = at
		}
	}
	return out
}

// nameAt gives the column's name in the heading over the groups that the screen column x falls on, and whether one
// does.
func (m Model) nameAt(x int) (nameSpot, bool) {
	for _, n := range m.barNames() {
		if x >= n.from && x < n.until {
			return n, true
		}
	}
	return nameSpot{}, false
}

// levelAt gives the level of the groups, from 0 the outermost, whose ▾ or ▸ before its column's name in the heading
// over the groups the screen column x falls on, or the rows' level, under the last group, whose ▾ or ▸ follows the
// last name; and whether one does.
func (m Model) levelAt(x int) (int, bool) {
	names, groups := m.barNames(), m.panel(m.stage).Groups()
	for _, n := range names {
		if x == n.from-2 {
			return slices.Index(groups, n.column), true
		}
	}
	if len(names) > 0 && x == names[len(names)-1].until+1 {
		return len(groups), true
	}
	return 0, false
}

// toggleLevel opens or closes a level of the groups, from 0 the outermost, the rows' under the last, as a click on its
// ▾ or ▸ asks. From ▾ it closes the level alone, which hides everything under it; from ▸, shown where it or a level
// over it is closed, it opens the level and every level over it, so it shows. Nothing else changes: each heading
// stays open or closed as it was, so a level shows again as it was left. The cursor stays on its row, or the nearest
// heading over it that shows, as reoutline says.
func (m *Model) toggleLevel(depth int) {
	s := m.stages[m.stage]
	next := maps.Clone(s.open)
	if s.open.Shows(depth) {
		next[grid.LevelKey(depth)] = false
	} else {
		for d := 0; d <= depth; d++ {
			delete(next, grid.LevelKey(d))
		}
	}
	m.reoutline(next)
}

// showLevels opens every level of the groups in an outline, the rows' too, as Expand asks, so nothing a level hides
// stays hidden.
func (m Model) showLevels(next grid.Outline) {
	for d := 0; d <= len(m.panel(m.stage).Groups()); d++ {
		delete(next, grid.LevelKey(d))
	}
}
