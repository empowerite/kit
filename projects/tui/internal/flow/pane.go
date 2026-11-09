// The selected stage's pane is its table's view, internal/table's: these give flow its layout and its moves by the
// names flow has always used, each doing nothing where there is no stage.
package flow

import "github.com/empowerite/kit/projects/tui/internal/table"

// place is a line of the pane under the control line, as table.Place says.
type place = table.Place

// rowsTop gives the screen line the rows' window starts on, as table.RowsTop says.
func (m Model) rowsTop() int { return table.RowsTop(&m) }

// rowsHeight gives the lines the rows have, which scroll, as table.RowsHeight says.
func (m Model) rowsHeight() int { return table.RowsHeight(&m) }

// heightUnder gives the lines the rows have under a header of those lines, as table.HeightUnder says.
func (m Model) heightUnder(header []string) int { return table.HeightUnder(&m, header) }

// laid gives the pane's lines under the control line, and the place of each, as table.Laid says.
func (m Model) laid() ([]string, []place) { return table.Laid(&m) }

// frozenLines gives how many of a stage's lines the frozen rows take: the selected stage's, as table.FrozenLines
// says; none for another, whose pane does not show.
func (m Model) frozenLines(i int) int {
	if len(m.stages) == 0 || i != m.stage {
		return 0
	}
	return table.FrozenLines(&m)
}

// lines gives the lines of a stage that scroll, in the rows' window: those allLines gives, past the frozen lines.
func (m Model) lines(i int) []table.Line { return m.allLines(i)[m.frozenLines(i):] }

// placeAt gives the place of the screen line y in the pane, as table.PlaceAt says.
func (m Model) placeAt(y int) (place, bool) { return table.PlaceAt(&m, y) }

// headerTop gives the screen line the header's first line stands on, as table.HeaderTop says.
func (m Model) headerTop() int { return table.HeaderTop(&m) }

// headerLine gives the line of the header the screen line y falls on, as table.HeaderLine says.
func (m Model) headerLine(y int) (int, bool) { return table.HeaderLine(&m, y) }

// onRule says whether the screen line y is the rule under the header.
func (m Model) onRule(y int) bool { return table.OnRule(&m, y) }

// up moves the cursor n rows up, as table.Up says.
func (m *Model) up(n int) {
	if len(m.stages) > 0 {
		table.Up(m, n)
	}
}

// down moves the cursor n rows down, as table.Down says.
func (m *Model) down(n int) {
	if len(m.stages) > 0 {
		table.Down(m, n)
	}
}

// enter gives the rows the keyboard from the control line, the cursor on the outline's first line, as table.Enter says.
func (m *Model) enter() {
	if len(m.stages) > 0 {
		m.onPreset = false
		table.Enter(m)
	}
}

// page moves a page down or up, as table.Page says.
func (m *Model) page(d int) {
	if len(m.stages) > 0 {
		table.Page(m, d)
	}
}

// end moves the cursor to the first row or the last, as table.End says.
func (m *Model) end(d int) {
	if len(m.stages) > 0 {
		table.End(m, d)
	}
}

// fit scrolls the pane so the row under the cursor shows, as table.Fit says.
func (m *Model) fit() {
	if len(m.stages) > 0 {
		table.Fit(m)
	}
}

// selectRow selects the row under the cursor, or unselects it, as table.SelectRow says.
func (m *Model) selectRow(keep bool) {
	if len(m.stages) > 0 {
		table.SelectRow(m, keep)
	}
}

// selectAll selects every row the stage shows, or none, as table.SelectAll says.
func (m *Model) selectAll() {
	if len(m.stages) > 0 {
		table.SelectAll(m)
	}
}

// selectedRows gives the selected stage's rows that are selected, as table.SelectedRows says.
func (m Model) selectedRows() []row {
	if len(m.stages) == 0 {
		return nil
	}
	return table.SelectedRows(&m)
}

// selected gives how many of the selected stage's rows shown are selected.
func (m Model) selected() int { return len(m.selectedRows()) }
