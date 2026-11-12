// A table's state, as a screen keeps it for each table, while another is shown too.
package table

import "github.com/empowerite/kit/projects/tui/internal/grid"

// Table is one table's state: the row under the cursor and the first line its pane shows; which of its headings,
// rows, and its header stand open, by key; the rows selected, by key, several at once; the preset its preset line was
// last set to, the name typed for Save As… or Rename… while the person names one, and the person's preset Rename…
// renames; its panel, its search, what it says of each field, and its cursor; and whether the view it was last set to
// still waits for rows to group by, as its rows load, and the view as that grouping left it, which the preset line
// names as the view still; and the columns its view has scrolled past, after the first, as grid.Scroll says, where
// they are wider than the pane; and the columns no row it loaded last holds a value in, as grid.Blanks gives them,
// which Auto-hide keeps from showing.
type Table struct {
	Cursor   int
	Offset   int
	Open     grid.Outline
	Picked   map[string]bool
	ViewName string
	Naming   string
	SavingAs bool
	Renaming string
	View     grid.Panel
	Grouping bool
	Grouped  *Preset
	Shift    int
	Blank    []string
}
