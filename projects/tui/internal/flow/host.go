// The picture is the host of its tables' controls, as internal/table asks of one: for the selected stage, its fields,
// its panel, the values its rows hold, its labels, the caller's login, the renderings it draws by, and the screen's size.
package flow

import (
	tea "charm.land/bubbletea/v2"
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

// Name gives the selected stage's name, which titles its table's dialogs.
func (m *Model) Name() string { return m.stages[m.stage].name }

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

// openPopover opens a field's popover, as table.OpenPopover does, and asks kit for the caller's login the first time,
// which @me names.
func (m *Model) openPopover(id string, x, y int, header bool) tea.Cmd {
	m.pop = table.OpenPopover(m, id, x, y, header)
	return m.askMe()
}
