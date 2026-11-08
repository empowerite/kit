// What a table's controls ask of the screen that shows the table: the screen implements Host once, for the table
// selected, and every control takes it.
package table

import "github.com/empowerite/kit/projects/tui/internal/grid"

// Host is the screen a table's controls act in, for the table selected: its fields; its panel, as its rows are let
// through, and setting it, which the rows follow at once; the values the rows loaded hold in a field, one a row; a
// label of a name, with its color, as a row holds it; the caller's login, once known; the rendering a field is drawn
// by now, as Automatic chose it; the screen's width and height; the table's name, which titles its dialogs; and the
// screen line its pane starts on, and the pane's width; the columns the header shows, in their order; where a column's
// name starts in the header, and its first line, none where the header does not name it; the screen line the header
// starts on; where a column's name stands in the heading over the groups, a line of -1 where it does not; the rows as
// the table lays them out, the names its header stands over them, and the width it lays them in; the table's state;
// building its rows again once what stands open changed, and keeping its cursor on a row and in its pane's view after;
// whether the rows have the keyboard; whether the header, closed, hides names; the table's views; whether it has
// columns the columns dialog lists, loaded; its color; and the screen columns the preset line's selector stands on,
// from its first to the one past it.
type Host interface {
	Fields() []Field
	Panel() grid.Panel
	SetPanel(grid.Panel)
	Values(id string) []Value
	Label(id, name string) Label
	Me() string
	Drawn(id string) string
	Size() (width, height int)
	Name() string
	Pane() (top, width int)
	ShownColumns() []string
	HeaderName(column string) (x int, name string)
	HeaderTop() int
	HeadingName(column string) (x, y int)
	Rows() []grid.Row
	Names() []string
	GridWidth() int
	Table() *Table
	Rebuild()
	Settle()
	InRows() bool
	HeaderHides() bool
	Views() Views
	Columnar() bool
	Color() string
	Selector() (from, until int)
}

// field gives the host's field of the id, and whether it has one.
func field(h Host, id string) (Field, bool) {
	for _, f := range h.Fields() {
		if f.ID == id {
			return f, true
		}
	}
	return Field{}, false
}

// labelOf gives the label the header draws over the host's field of the id; an id it does not know is its own label.
func labelOf(h Host, id string) string {
	if f, ok := field(h, id); ok {
		return f.Label
	}
	return id
}
