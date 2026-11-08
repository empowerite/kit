// The outline is internal/table's: what a stage's rows open and close, as table.Expand, table.Folding, and the
// heading over the groups say. A stage keeps which of its nodes stand open in its table's state; the tab's menu names
// the folds.
package flow

import (
	"github.com/empowerite/kit/projects/tui/internal/grid"
	"github.com/empowerite/kit/projects/tui/internal/shell"
)

// every gives a stage's whole outline: its rows shown, each closed group's rows after its heading.
func (m Model) every(i int) []row { return grid.Every(m.rows(i)) }

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
