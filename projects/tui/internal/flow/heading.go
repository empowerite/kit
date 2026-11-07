// The heading over the groups names their columns, and a click on a name drops the column's own menu around it, as a
// name in the header does, and so does ⏎ with the cursor on the heading, around its first column's; the ▾ or ▸ before
// a column's name shows the groups down to its level, and the one after the last name the rows under them, as
// foldLevel says. The heading itself never closes. A value's heading folds by its own ▾ or ▸, as any row does.
package flow

import (
	"maps"
	"slices"
	"strings"

	"github.com/charmbracelet/x/ansi"

	"github.com/empowerite/kit/projects/tui/internal/grid"
)

// nameSpot is a column's name in a heading, and the screen columns it stands on, from its first to the one past it.
type nameSpot struct {
	column      string
	from, until int
}

// headingNames gives the columns the heading over the groups names, each where the rows draw its name, before the
// next level's fold, the count, or the heading's end; any other row names none. The heading's text stands after the gutter's two cells,
// its selection's.
func (m Model) headingNames(r row) []nameSpot {
	if !grid.OverGroups(r) {
		return nil
	}
	text := ansi.Strip(r.Cells[0].Values[0])
	var out []nameSpot
	for _, column := range m.panel(m.stage).Groups() {
		for from := 0; ; {
			k := strings.Index(text[from:], " "+column)
			if k < 0 {
				break
			}
			at := from + k + 1
			if end := at + len(column); end == len(text) || strings.HasPrefix(text[end:], "  ") ||
				strings.HasPrefix(text[end:], " "+grid.Unfolded) || strings.HasPrefix(text[end:], " "+grid.Folded) {
				x := 2 + ansi.StringWidth(text[:at])
				out = append(out, nameSpot{column, x, x + ansi.StringWidth(column)})
				break
			}
			from = at
		}
	}
	return out
}

// nameAt gives the column's name in a heading that the screen column x falls on, and whether one does.
func (m Model) nameAt(r row, x int) (nameSpot, bool) {
	for _, n := range m.headingNames(r) {
		if x >= n.from && x < n.until {
			return n, true
		}
	}
	return nameSpot{}, false
}

// levelAt gives the level of the groups, from 0 the outermost, whose ▾ or ▸ before its column's name in a heading the
// screen column x falls on, or the rows' level, under the last group, whose ▾ or ▸ follows the last name; and whether
// one does.
func (m Model) levelAt(r row, x int) (int, bool) {
	names, groups := m.headingNames(r), m.panel(m.stage).Groups()
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

// foldLevel shows the groups down to a level, from 0 the outermost, and nothing under it, as a click on its ▾ or ▸
// asks: every heading over the level opens, and every heading of it closes, so its values stand listed, their rows
// folded away. From ▾, the rows open under the level, it collapses to it; from ▸, it expands to it. The cursor stays
// on its row, or the nearest heading over it that shows, as reoutline says.
func (m *Model) foldLevel(depth int) {
	s, every, level := m.stages[m.stage], m.every(m.stage), depth+2
	next := maps.Clone(s.open)
	for _, r := range every {
		switch l := grid.Level(r); {
		case l == level:
			next[r.Key] = false
		case l >= 1 && l < level:
			next[r.Key] = true
		}
	}
	m.reoutline(next)
}
