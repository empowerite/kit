// The control line's stops: down from the bar reaches the heading over the groups, each group's name a stop, then the
// rows', then the preset picker; left and right move between them, enter and space act on the one the keyboard is on.
package flow

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/empowerite/kit/projects/tui/internal/style"
	"github.com/empowerite/kit/projects/tui/internal/table"
)

// onStops gives backlog, grouped by source, with the keyboard on the heading over the groups' first stop.
func onStops(t *testing.T) Model {
	t.Helper()
	m := drive(loaded(t, viewing(""), tea.WindowSizeMsg{Width: 140, Height: 40}), down)
	if stop, on := m.atStop(); !on || stop.Column != "source" || m.bar().Focused || m.inRows {
		t.Fatalf("down from the bar is not on source's name: %+v, on %v\n%s", stop, on, view(m))
	}
	return m
}

func TestDownFromTheBarReachesTheGroupsNamesAndRightThePresetPicker(t *testing.T) {
	m := onStops(t)
	// the stop is drawn as the line under the keyboard in a menu is, the preset picker not
	control, _ := m.control()
	if !strings.Contains(control, style.Selected(grid0(m))) || m.presetLine() != table.PresetLine(&m, false) {
		t.Errorf("source's stop is not drawn under the keyboard: %q", control)
	}
	for _, want := range []string{"←→ move", "⏎ menu", "␣ fold", "↓ rows"} {
		if !strings.Contains(keysOf(m), want) {
			t.Errorf("the keys on a name lack %q: %q", want, keysOf(m))
		}
	}
	// right reaches the rows' stop, then the preset picker; left goes back, and stops at the first
	if m = drive(m, right); !onRows(m) || !strings.Contains(keysOf(m), "⏎ fold") {
		t.Errorf("right from source: %q", keysOf(m))
	}
	if m = drive(m, right); !m.onPreset || isOnStop(m) || m.presetLine() != table.PresetLine(&m, true) ||
		!strings.Contains(keysOf(m), "⏎ presets") {
		t.Errorf("right from the rows' stop: on the line %v, on a stop %v", m.onPreset, isOnStop(m))
	}
	if m = drive(m, right); !m.onPreset || isOnStop(m) {
		t.Errorf("right past the picker left it")
	}
	if m = drive(m, left); !onRows(m) {
		t.Errorf("left from the picker is not on the rows' stop")
	}
	if m = drive(m, left, left); !isOnStop(m) || m.stop != 0 {
		t.Errorf("left past the first stop: %d", m.stop)
	}
}

func TestEnterAndSpaceActOnTheStop(t *testing.T) {
	m := onStops(t)
	// space on source folds its level, and again opens it; enter on the rows' stop folds theirs
	if m = drive(m, press(tea.KeySpace)); m.stages[0].Open.Shows(0) || !isOnStop(m) {
		t.Errorf("space on source left its level open: %v", m.stages[0].Open)
	}
	if m = drive(m, press(tea.KeySpace)); !m.stages[0].Open.Shows(0) {
		t.Errorf("space again left source's level closed")
	}
	if m = drive(m, right, press(tea.KeyEnter)); m.stages[0].Open.Shows(1) || !onRows(m) {
		t.Errorf("enter on the rows' stop left them open: %v", m.stages[0].Open)
	}
	m = drive(m, press(tea.KeyEnter))
	// enter on a name drops its column's menu; esc closes it, the keyboard back on the name
	if m = drive(m, left, press(tea.KeyEnter)); !m.colMenu.Open || m.colMenu.Column != "source" {
		t.Fatalf("enter on source: menu %v, %q", m.colMenu.Open, m.colMenu.Column)
	}
	if m = drive(m, esc); m.colMenu.Open || !isOnStop(m) || m.stop != 0 {
		t.Errorf("esc in source's menu: open %v, stop %d", m.colMenu.Open, m.stop)
	}
}

func TestTheStopsLeadToTheRowsAndBackAndToTheBar(t *testing.T) {
	m := onStops(t)
	if m = drive(m, down); !m.inRows || m.onPreset {
		t.Fatalf("down from a stop: in rows %v", m.inRows)
	}
	// up past the first row lands on the first stop again
	if m = drive(m, up, up); !isOnStop(m) || m.stop != 0 {
		t.Errorf("up from the first row: on a stop %v, %d", isOnStop(m), m.stop)
	}
	if m = drive(m, up); m.onPreset || !m.bar().Focused {
		t.Errorf("up from a stop: on the line %v", m.onPreset)
	}
	if m = drive(m, down, esc); m.onPreset || !m.bar().Focused {
		t.Errorf("esc from a stop: on the line %v", m.onPreset)
	}
	// a key no stop takes passes on, as / to the search
	if m = drive(m, down, press('/')); !m.inPanel {
		t.Errorf("/ from a stop did not reach the search")
	}
	// a click on a name in the heading over the groups puts the keyboard on its stop
	m = drive(onStops(t), right, right)
	stops := m.stops()
	if m = drive(m, leftAt(stops[0].From+3, m.top()), esc); !isOnStop(m) || m.stop != 0 {
		t.Errorf("a click on source's name: on a stop %v, %d", isOnStop(m), m.stop)
	}
}

// isOnStop says whether the keyboard is on a stop of the heading over the groups.
func isOnStop(m Model) bool {
	_, on := m.atStop()
	return on
}

// onRows says whether the keyboard is on the rows' stop, past the groups' names.
func onRows(m Model) bool {
	stop, on := m.atStop()
	return on && stop.Column == ""
}

// grid0 gives the cells of the first stop as the control line draws them, plain.
func grid0(m Model) string {
	stop := m.stops()[0]
	line, _ := m.control()
	return ansi.Strip(ansi.Cut(line, stop.From, stop.Until))
}
