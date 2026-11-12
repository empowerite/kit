// A block cut at the pane's edge, a page's code block or a check's fix and what more it says, scrolls across on its
// own, under the pointer, by a turn of the wheel across or a click on its ◃ or ▹; the prose and the checks' rows stay.
package welcome

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/empowerite/kit/projects/tui/internal/kit"
	"github.com/empowerite/kit/projects/tui/internal/pane"
	"github.com/empowerite/kit/projects/tui/internal/testkit"
)

// lineAt gives the screen line the text stands on, and the line.
func lineAt(t *testing.T, m Model, text string) (int, string) {
	t.Helper()
	for y, l := range strings.Split(view(m), "\n") {
		if strings.Contains(l, text) {
			return y, l
		}
	}
	t.Fatalf("no %q:\n%s", text, view(m))
	return -1, ""
}

// across gives a turn of the wheel across at the screen line y, right or left.
func across(y int, right bool) tea.MouseWheelMsg {
	b := tea.MouseWheelLeft
	if right {
		b = tea.MouseWheelRight
	}
	return tea.MouseWheelMsg{X: 10, Y: y, Button: b}
}

func TestATurnAcrossScrollsAPagesCodeBlockAlone(t *testing.T) {
	m, _ := loaded(t, tea.WindowSizeMsg{Width: 40, Height: 30})
	m.tabs[introduction].page = "Prose.\n\n```\nkit " + strings.Repeat("abcdefghij", 6) + " end\n```\n\nMore prose."
	m = drive(m, stepKey(introduction))
	y, l := lineAt(t, m, "kit abc")
	if !strings.HasSuffix(strings.TrimRight(l, " "), pane.After) {
		t.Fatalf("the code block cut at the edge: %q", l)
	}
	m = drive(m, across(y, true))
	if l := strings.Split(view(m), "\n")[y]; !strings.Contains(l, lead+pane.Before+" abcdefghij") {
		t.Errorf("a turn right scrolls the block, the lead put:\n%s", view(m))
	}
	if _, l := lineAt(t, m, "Prose."); strings.Contains(l, pane.Before) {
		t.Errorf("the prose stays: %q", l)
	}
	for range 30 {
		m = drive(m, across(y, true))
	}
	if _, l := lineAt(t, m, "hij end"); strings.Contains(l, pane.After) {
		t.Errorf("never past the block's end: %q", l)
	}
	for range 30 {
		m = drive(m, tea.MouseWheelMsg{X: 10, Y: y, Button: tea.MouseWheelUp, Mod: tea.ModShift})
	}
	if _, l := lineAt(t, m, "kit abc"); strings.Contains(l, pane.Before) || m.offsets[introduction] != 0 {
		t.Errorf("shift with turns up, back to the start: %q, offset %d", l, m.offsets[introduction])
	}
}

// A check's fix, too long for the pane, and what more it says scroll across as one block, under the row, which stays;
// a click on the block's ▹ scrolls it half the pane, and on its ◃ back, the cursor on the check.
func TestACheckFixCutAtTheEdgeScrollsAcrossAndItsRowStays(t *testing.T) {
	m, _ := loaded(t, tea.WindowSizeMsg{Width: 40, Height: 30})
	m.checks = []kit.Check{{ID: "tool-jq", Step: "toolchains", Line: "jq is not installed",
		Fix: "brew install jq --with-every-option-there-is --and-more", More: "see https://jqlang.org/download/ for more",
		Runs: true}}
	m = drive(m, stepKey(chains))
	y, l := lineAt(t, m, "brew install")
	if !strings.HasSuffix(strings.TrimRight(l, " "), pane.After) {
		t.Fatalf("the fix cut at the edge: %q\n%s", l, view(m))
	}
	row, _ := lineAt(t, m, "jq is not installed")
	if m = drive(m, across(row, true)); strings.Contains(view(m), pane.Before) {
		t.Errorf("a turn across the check's row scrolled:\n%s", view(m))
	}
	if m = drive(m, across(y, true)); !strings.Contains(strings.Split(view(m), "\n")[y], pane.Before+"install jq") {
		t.Errorf("a turn right on the fix:\n%s", view(m))
	}
	if _, more := lineAt(t, m, "jqlang"); !strings.Contains(more, pane.Before) {
		t.Errorf("what more it says scrolls with the fix: %q", more)
	}
	if m = drive(m, across(y, false)); strings.Contains(view(m), pane.Before) {
		t.Errorf("a turn left, back:\n%s", view(m))
	}
	// a plain click on ▹, the pane's last cell, then on ◃, where the block starts under the row
	if m = drive(m, testkit.Click(m.width-2, y)); m.across["check/"+m.checks[0].Key()] != (m.width-1-len(under))/2 {
		t.Fatalf("a click on ▹: %v\n%s", m.across, view(m))
	}
	if m = drive(m, testkit.Click(len(under), y)); m.across["check/"+m.checks[0].Key()] != 0 || m.cursor != 0 {
		t.Errorf("a click on ◃: %v, cursor %d", m.across, m.cursor)
	}
}
