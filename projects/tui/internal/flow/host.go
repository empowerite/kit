// The picture is the host of its tables' controls, as internal/table asks of one: for the selected stage, its fields,
// its panel, the values its rows hold, its labels, the caller's login, the renderings it draws by, and the screen's size.
package flow

import (
	"github.com/empowerite/kit/projects/tui/internal/grid"
	"github.com/empowerite/kit/projects/tui/internal/table"
)

var _ table.Host = (*Model)(nil)

// Fields gives the selected stage's fields.
func (m *Model) Fields() []table.Field { return fieldsOf(m.stages[m.stage].name) }

// Panel gives the selected stage's panel, as its rows are let through.
func (m *Model) Panel() grid.Panel { return m.panel(m.stage) }

// SetPanel sets the selected stage's panel, as setPanel does.
func (m *Model) SetPanel(p grid.Panel) { m.setPanel(p) }

// Values gives the values the selected stage's rows hold in a field, one a row.
func (m *Model) Values(id string) []table.Value {
	var out []table.Value
	for _, r := range m.records(m.stages[m.stage]) {
		out = append(out, r.values[id])
	}
	return out
}

// Label gives the label of a name, as labelNamed finds it.
func (m *Model) Label(id, name string) table.Label { return m.labelNamed(id, name) }

// Me gives the caller's login, once kit has said it.
func (m *Model) Me() string { return m.me }

// Drawn gives the rendering the selected stage draws a field by now.
func (m *Model) Drawn(id string) string { return m.shown(m.stage).drawn[id] }

// Size gives the screen's width and height.
func (m *Model) Size() (width, height int) { return m.width, m.height }

// Name gives the selected stage's label, the tab's alone, which titles its table's dialogs.
func (m *Model) Name() string { return label(m.stages[m.stage].name) }

// Pane gives the screen line the pane starts on, under the bar, and the pane's width.
func (m *Model) Pane() (top, width int) { return m.top(), m.paneWidth() }

// ShownColumns gives the columns the selected stage's header shows, in their order.
func (m *Model) ShownColumns() []string { return m.shownColumns(m.stage) }

// HeaderName gives where a column's name starts in the header, and its first line, as columnName finds it.
func (m *Model) HeaderName(column string) (int, string) { return m.columnName(column) }

// HeaderTop gives the screen line the header starts on.
func (m *Model) HeaderTop() int { return m.headerTop() }

// HeadingName gives where a column's name stands in the heading over the groups, as fieldName finds it.
func (m *Model) HeadingName(column string) (x, y int) { return m.fieldName(column) }

// Rows gives the selected stage's rows as its table lays them out.
func (m *Model) Rows() []grid.Row { return m.rows(m.stage) }

// Names gives the names the selected stage's header stands over its rows.
func (m *Model) Names() []string { return m.names(m.stage) }

// GridWidth gives the width the selected stage's rows are laid out in.
func (m *Model) GridWidth() int { return m.gridWidth(m.stage) }

// Table gives the selected stage's table's state.
func (m *Model) Table() *table.Table { return &m.stages[m.stage].Table }

// Rebuild builds the stages' rows again, once what stands open changed.
func (m *Model) Rebuild() { m.rebuild() }

// Settle keeps the selected stage's cursor on a row, and in the pane's view.
func (m *Model) Settle() {
	m.clamp(m.stage)
	m.fit()
}

// InRows says whether the rows have the keyboard, a stage there to take it.
func (m *Model) InRows() bool { return m.inRows && len(m.stages) > 0 }

// Views gives the selected stage's views, its own and the person's.
func (m *Model) Views() table.Views { return m.viewsOf(m.stage) }

// Color gives the selected stage's color, its tab's.
func (m *Model) Color() string { return stageColors[m.stages[m.stage].name] }

// Selector gives the screen columns the preset line's selector stands on, from its first to the one past it.
func (m *Model) Selector() (from, until int) { return m.dropStart(), m.dropEnd() }

// Lines gives the lines the selected stage's pane draws, as allLines draws them.
func (m *Model) Lines() []table.Line { return m.allLines(m.stage) }

// Header gives the selected stage's header's lines and its rule, as header gives them.
func (m *Model) Header() ([]string, bool) { return m.header() }

// PaneHeight gives the pane's height.
func (m *Model) PaneHeight() int { return m.paneHeight() }

// Controls says whether the control line stands over the selected stage's rows.
func (m *Model) Controls() bool { return m.controls() }

// LeaveRows gives up the rows' keyboard, past their first row: to the control line, on its first stop, or the bar
// where the stage has none.
func (m *Model) LeaveRows() {
	if m.inRows = false; m.presets(m.stage) || len(m.stops()) > 0 {
		m.toControls()
	}
}

// EnterRows gives the rows the keyboard.
func (m *Model) EnterRows() { m.inRows = true }

var _ table.Keyboard = (*Model)(nil)

// Stop gives the stop of the heading over the groups the keyboard stands on, -1 the preset picker.
func (m *Model) Stop() int { return m.stop }

// SetStop puts the keyboard on a stop of the heading over the groups, -1 the preset picker.
func (m *Model) SetStop(k int) { m.stop = k }

// LeaveControls gives up the control line's keyboard, to the bar.
func (m *Model) LeaveControls() { m.onPreset = false }

// OpenFieldMenu opens a column's card around its name in the heading over the groups, as openFieldMenu does; headingKey
// asks kit for the caller's login.
func (m *Model) OpenFieldMenu(column string) { m.openFieldMenu(column) }

// HasPresets says whether the selected stage's table has a preset line.
func (m *Model) HasPresets() bool { return m.presets(m.stage) }
