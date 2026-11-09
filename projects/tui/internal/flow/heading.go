// The heading over the groups is internal/table's; it stands on the control line, which flow draws.
package flow

import (
	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/empowerite/kit/projects/tui/internal/shell"
	"github.com/empowerite/kit/projects/tui/internal/style"
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

// headingKey takes a key while the keyboard stands on a stop of the heading over the groups: tab and shift-tab move
// between the stops, and tab past the last to the preset picker; right opens the stop's level and left closes it;
// enter on a name drops its column's menu, and on the rows' opens or closes their level, as space does on any stop;
// up or esc gives the bar the keyboard, and down the rows, when the stage has some.
func (m *Model) headingKey(k tea.KeyPressMsg) (tea.Cmd, bool) {
	stop, _ := m.atStop()
	switch k.String() {
	case "shift+tab":
		m.stop = max(0, m.stop-1)
	case "tab":
		if m.stop < len(m.stops())-1 {
			m.stop++
		} else if m.presets(m.stage) {
			m.stop = -1
		}
	case "right", "left":
		// right opens a level closed, and left closes one open; either leaves it as it is otherwise
		if open := m.stages[m.stage].Open.Shows(stop.Level); open == (k.String() == "left") {
			table.ToggleLevel(m, stop.Level)
		}
	case "enter":
		if stop.Column != "" {
			m.openFieldMenu(stop.Column)
			return nil, true
		}
		table.ToggleLevel(m, stop.Level)
	case "space":
		table.ToggleLevel(m, stop.Level)
	case "esc", "up":
		m.onPreset = false
	case "down":
		if len(m.rows(m.stage)) > 0 {
			m.enter()
		}
	default:
		return nil, false
	}
	return nil, true
}

// headingKeys gives the segment of a stop of the heading over the groups while it has the keyboard: up or esc gives
// the bar the keyboard, down the rows, tab and shift-tab move, left and right close and open the stop's level, and
// enter drops a name's column menu.
func (m *Model) headingKeys(stop table.Stop) shell.Segment {
	s := shell.Segment{Bindings: []shell.Binding{shell.Bind("↑", tabsHint, "up"), shell.Bind("esc", tabsHint)}}
	if len(m.rows(m.stage)) > 0 {
		s.Bindings = append(s.Bindings, shell.Bind("↓", "rows", "down"))
	}
	// tab and shift-tab move where there is another stop, or the preset picker, to move to
	if len(m.stops()) > 1 || m.presets(m.stage) {
		s.Bindings = append(s.Bindings, shell.Bind("⇥·⇧⇥", "move", "tab", "shift+tab"))
	}
	s.Bindings = append(s.Bindings, shell.Bind("←→", "fold", "left", "right"))
	if stop.Column != "" {
		return s.With(shell.Bind("⏎", "menu", "enter"))
	}
	return s
}

// focusStop draws the stop the keyboard stands on in the control line's left part, as the line under the keyboard in a
// menu is drawn; the rest as it is.
func (m *Model) focusStop(left string) string {
	stop, ok := m.atStop()
	if !ok {
		return left
	}
	return ansi.Cut(left, 0, stop.From) + style.Selected(ansi.Cut(left, stop.From, stop.Until)) +
		ansi.Cut(left, stop.Until, ansi.StringWidth(left))
}
