// The heading over the groups names their columns, and a click on a name drops the column's own menu around it, as a
// name in the header does, and so does ⏎ with the cursor on the heading, around its first column's; the heading folds
// by its first ▾ or ▸ alone, and each level of the groups by the ▾ or ▸ before its column's name, as foldLevel says.
// A click anywhere on a value's heading folds its rows away, or unfolds them, as one on its ▾ or ▸ does.
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

// headingNames gives the columns the heading over the groups names, each where the rows draw its name, before a ›,
// the count, or the heading's end; any other row names none. The heading's text stands after the gutter's two cells,
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
				strings.HasPrefix(text[end:], " ›") {
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
// screen column x falls on, and whether one does.
func (m Model) levelAt(r row, x int) (int, bool) {
	for _, n := range m.headingNames(r) {
		if x == n.from-2 {
			return slices.Index(m.panel(m.stage).Groups(), n.column), true
		}
	}
	return 0, false
}

// foldLevel closes every heading of a level of the groups, from 0 the outermost, while every one stands open, as a
// click on its ▾ asks; else, as a click on its ▸ asks, it opens them and every heading over them, so the rows stand
// open down to the level. The cursor stays on its row, as reoutline says.
func (m *Model) foldLevel(depth int) {
	s, every, level := m.stages[m.stage], m.every(m.stage), depth+2
	open := !slices.ContainsFunc(every, func(r row) bool { return grid.Level(r) == level && !s.open.Open(r) })
	next := maps.Clone(s.open)
	for _, r := range every {
		switch l := grid.Level(r); {
		case l == level:
			next[r.Key] = !open
		case !open && l >= 1 && l < level:
			next[r.Key] = true
		}
	}
	m.reoutline(next)
}
