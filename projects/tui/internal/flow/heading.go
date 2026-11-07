// The heading over the groups names their columns, and a click on a name drops the column's own menu around it, as a
// name in the header does, and so does ⏎ with the cursor on the heading, around its first column's; the heading folds
// by its ▾ or ▸ alone. A click anywhere on a value's heading folds its rows away, or unfolds them, as one on its ▾ or
// ▸ does.
package flow

import (
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
