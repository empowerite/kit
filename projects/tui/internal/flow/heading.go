// The heading over the groups is internal/table's; it stands on the control line, which flow draws.
package flow

import "github.com/empowerite/kit/projects/tui/internal/table"

// first gives the first of a stage's rows the cursor can stand on, as table.First says.
func (m Model) first(i int) int { return table.First(m.rows(i)) }

// barText gives the heading over the selected stage's groups, as table.BarText says; nothing with no stage.
func (m Model) barText() string {
	if len(m.stages) == 0 {
		return ""
	}
	return table.BarText(m.rows(m.stage))
}
