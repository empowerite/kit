// An issue's or a change's labels, as its forge names and colors them, each drawn as a pill, as style draws one: its
// text on the label's color between half circles, a label that is a key and a value, as priority=high, two segments.
package flow

import "github.com/empowerite/kit/projects/tui/internal/table"

// labelNamed gives the label of a name, with its color, as one of the rows the selected stage loaded holds it in the
// field, by id, as table.LabelNamed finds it; a name no row holds is a label without a color.
func (m Model) labelNamed(id, name string) table.Label {
	return table.LabelNamed(entries(m.records(m.stages[m.stage])), id, name)
}
