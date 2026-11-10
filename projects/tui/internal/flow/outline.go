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

// folds gives the tab's menu's items that show and hide the outline: a level more of the whole table and a level
// less, as > and < do, and the selected rows shown, every heading over them too, and hidden, dim while none are.
func (m Model) folds() []shell.Item {
	return []shell.Item{{Glyph: More, Name: "Show More", Keys: ">"}, {Glyph: Less, Name: "Show Less", Keys: "<"},
		{Glyph: Expand, Name: "Show Selected"}, {Glyph: Collapse, Name: "Hide Selected"}}
}
