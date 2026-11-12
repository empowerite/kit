// A code block on a tutorial's page wider than the screen scrolls across on its own, under the pointer, by a turn of
// the wheel across or a click on its ◃ or ▹, its lines together; the prose around it stays.
package tutorials

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/empowerite/kit/projects/tui/internal/kit"
	"github.com/empowerite/kit/projects/tui/internal/pane"
	"github.com/empowerite/kit/projects/tui/internal/testkit"
)

// wide opens, at 40 cells, a tutorial in the kit whose page has prose, then a code block twice the screen's width.
func wide(t *testing.T) (Model, int) {
	t.Helper()
	m, _ := fake(t)
	code := "kit " + strings.Repeat("abcdefghij", 7) + " end"
	page := "Some prose.\n\n```\n" + code + "\nshort\n```\n\nMore prose."
	m = drive(m, tea.WindowSizeMsg{Width: 40, Height: 20},
		readMsg{tutorial: kit.Tutorial{ID: "w", Title: "Wide", Where: kit.InTheKit}, page: page})
	return m, rowOf(t, m, "kit abc")
}

// rowOf gives the screen line the text stands on.
func rowOf(t *testing.T, m Model, text string) int {
	t.Helper()
	for y, l := range lines(m) {
		if strings.Contains(l, text) {
			return y
		}
	}
	t.Fatalf("no %q:\n%s", text, view(m))
	return -1
}

func TestATurnAcrossScrollsTheCodeBlockUnderThePointer(t *testing.T) {
	m, y := wide(t)
	across := func(right bool) tea.Msg {
		b := tea.MouseWheelLeft
		if right {
			b = tea.MouseWheelRight
		}
		return tea.MouseWheelMsg{X: 10, Y: y, Button: b}
	}
	if l := lines(m)[y]; !strings.HasSuffix(strings.TrimRight(l, " "), pane.After) || strings.Contains(l, pane.Before) {
		t.Fatalf("the block cut at the edge:\n%s", view(m))
	}
	m = drive(m, across(true))
	if l := lines(m)[y]; !strings.Contains(l, "  "+pane.Before+" abcdefghij") {
		t.Errorf("a turn right scrolls the block %d cells, the lead put: %q", pane.TurnCells, l)
	}
	if l := lines(m)[y+1]; !strings.Contains(l, pane.Before) {
		t.Errorf("the block's short line scrolls with it: %q", l)
	}
	if !strings.Contains(view(m), "Some prose.") || !strings.Contains(view(m), "More prose.") {
		t.Errorf("the prose stays:\n%s", view(m))
	}
	for range 30 {
		m = drive(m, across(true))
	}
	if l := strings.TrimRight(lines(m)[y], " "); !strings.HasSuffix(l, "hij end") || strings.Contains(l, pane.After) {
		t.Errorf("never past the block's end: %q", l)
	}
	// shift with a turn up goes left; a turn across the prose, or off the pane, scrolls nothing
	for range 30 {
		m = drive(m, tea.MouseWheelMsg{X: 10, Y: y, Button: tea.MouseWheelUp, Mod: tea.ModShift})
	}
	if l := lines(m)[y]; strings.Contains(l, pane.Before) || m.read.offset != 0 {
		t.Errorf("shift with turns up, back to the start: %q, offset %d", l, m.read.offset)
	}
	prose := rowOf(t, m, "Some prose.")
	if m = drive(m, tea.MouseWheelMsg{X: 10, Y: prose, Button: tea.MouseWheelRight},
		tea.MouseWheelMsg{X: 10, Y: 1, Button: tea.MouseWheelRight}); strings.Contains(lines(m)[y], pane.Before) {
		t.Errorf("a turn across the prose, or off the pane, scrolled the block: %q", lines(m)[y])
	}
}

// A click on the block's ▹ scrolls it half the page's width, and on its ◃ back; the crumb's spot keeps where it
// stood, as later turns leave it.
func TestAClickOnACodeBlocksTriangleScrollsIt(t *testing.T) {
	m, y := wide(t)
	x := len(lead) + m.pageWidth() - 1
	if m = drive(m, testkit.Click(x, y)); m.read.across["0"] != m.pageWidth()/2 {
		t.Fatalf("a click on ▹: %v\n%s", m.read.across, view(m))
	}
	_, spot := m.Spot()
	if m = drive(m, testkit.Click(len(lead), y)); m.read.across["0"] != 0 {
		t.Errorf("a click on ◃: %v", m.read.across)
	}
	if back := m.Restore(spot).(Model); back.read.across["0"] != m.pageWidth()/2 {
		t.Errorf("the spot put back: %v", back.read.across)
	}
	if m = drive(m, testkit.Click(10, y)); m.read.across["0"] != 0 || !m.read.open {
		t.Errorf("a click on the block's text: %v, reading %v", m.read.across, m.read.open)
	}
}
