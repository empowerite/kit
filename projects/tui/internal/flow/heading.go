// The heading over the groups is internal/table's; it stands on the control line, which flow draws.
package flow

import (
	tea "charm.land/bubbletea/v2"

	"github.com/empowerite/kit/projects/tui/internal/shell"
	"github.com/empowerite/kit/projects/tui/internal/table"
)

// first gives the first of a stage's rows the cursor can stand on, as table.First says.
func (m Model) first(i int) int { return table.First(m.rows(i)) }

// barText gives the heading over the selected stage's groups, as table.BarText says; nothing with no stage.
func (m Model) barText() string {
	if len(m.stages) == 0 {
		return ""
	}
	return table.BarText(m.rows(m.stage))
}

// stops gives the stops of the heading over the selected stage's groups, as table.Stops says; none with no stage.
func (m *Model) stops() []table.Stop {
	if len(m.stages) == 0 {
		return nil
	}
	return table.Stops(m)
}

// atStop gives the stop of the heading over the groups the keyboard stands on, and whether it stands on one: while
// the control line has the keyboard, on a stop, not the preset picker.
func (m *Model) atStop() (table.Stop, bool) {
	stops := m.stops()
	if !m.onPreset || m.inPanel || m.stop < 0 || m.stop >= len(stops) {
		return table.Stop{}, false
	}
	return stops[m.stop], true
}

// toControls gives the control line the keyboard, on its first stop: the first name of the heading over the groups,
// or the preset picker where the rows group by nothing.
func (m *Model) toControls() {
	m.onPreset, m.inRows, m.inPanel, m.stop = true, false, false, -1
	if len(m.stops()) > 0 {
		m.stop = 0
	}
}

// headingKey takes a key while the keyboard stands on a stop of the heading over the groups, as table.HeadingKey says.
func (m *Model) headingKey(k tea.KeyPressMsg) (tea.Cmd, bool) { return nil, table.HeadingKey(m, k) }

// headingKeys gives the segment of a stop of the heading over the groups while it has the keyboard, as
// table.HeadingKeys says.
func (m *Model) headingKeys(stop table.Stop) shell.Segment {
	return table.HeadingKeys(m, stop, m.presets(m.stage))
}

// focusStop draws the stop the keyboard stands on in the control line's left part, as table.FocusStop draws it; the
// rest as it is.
func (m *Model) focusStop(left string) string {
	stop, ok := m.atStop()
	if !ok {
		return left
	}
	return table.FocusStop(left, stop)
}
