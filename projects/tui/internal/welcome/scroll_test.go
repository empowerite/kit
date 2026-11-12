// The introduction with its failed checks' table scrolls as one page: the wheel and the page keys scroll it, the
// cursor staying where it is, down past the table's last row goes on into the page's words, and a key that moves the
// table's cursor, or gives it the keyboard, brings what has it back into view.
package welcome

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/empowerite/kit/projects/tui/internal/testkit"
)

// shortScreen is a screen too short for the introduction's table and its words.
var shortScreen = tea.WindowSizeMsg{Width: 120, Height: 16}

// spin is a turn of the mouse wheel over the pane, down or up.
func spin(down bool) tea.Msg { return turn(10, paneTop+2, down) }

func TestTheWheelScrollsTheIntroductionAndItsTable(t *testing.T) {
	m, _ := loaded(t, shortScreen)
	m, _ = onCheck(m, "jq")
	m = drive(m, spin(true), spin(true), spin(true))
	if m.offsets[introduction] != 9 || m.checks[m.checkUnder()].ID != "jq" ||
		!strings.HasPrefix(strings.Split(view(m), "\n")[paneTop], "  □   jq is not installed") {
		t.Fatalf("offset %d, under the cursor %d: %q", m.offsets[introduction], m.checkUnder(),
			strings.Split(view(m), "\n")[paneTop])
	}
	// the table's rows follow it: a click lands on the row drawn there
	drawn := strings.Split(view(m), "\n")
	y := slicesIndex(drawn, "gh is not logged in")
	if m = drive(m, testkit.Click(10, y)); m.checks[m.checkUnder()].ID != "gh-auth" {
		t.Errorf("a click on gh-auth's row, scrolled: under the cursor %d", m.checkUnder())
	}
	// never past the top
	if m = drive(m, spin(false), spin(false), spin(false), spin(false)); m.offsets[introduction] != 0 {
		t.Errorf("the wheel up past the top: offset %d", m.offsets[introduction])
	}
	// nor past the end, which a taller screen then shows from
	for range 30 {
		m = drive(m, spin(true))
	}
	end := m.offsets[introduction]
	if lines := m.introLines(); end != len(lines)-m.paneHeight() || !strings.Contains(view(m), "kit welcome --checks") {
		t.Errorf("the wheel down past the end: offset %d of %d:\n%s", end, len(lines), view(m))
	}
	if m = drive(m, tea.WindowSizeMsg{Width: 120, Height: 30}); m.offsets[introduction] >= end {
		t.Errorf("a taller screen kept offset %d", m.offsets[introduction])
	}
}

func TestThePageKeysScrollTheIntroductionAndAKeyBringsTheCursorBack(t *testing.T) {
	m, _ := loaded(t, shortScreen)
	m, _ = onCheck(m, "jq")
	h := m.paneHeight()
	if m = drive(m, pgdown); m.offsets[introduction] != h-1 || m.checks[m.checkUnder()].ID != "jq" {
		t.Errorf("pgdown: offset %d, under the cursor %d", m.offsets[introduction], m.checkUnder())
	}
	if m = drive(m, pgup); m.offsets[introduction] != 0 {
		t.Errorf("pgup: offset %d", m.offsets[introduction])
	}
	if m = drive(m, press(tea.KeyEnd)); m.offsets[introduction] != len(m.introLines())-h {
		t.Errorf("end: offset %d", m.offsets[introduction])
	}
	// up brings jq's neighbor into view, from below it
	if m = drive(m, up); !strings.Contains(view(m), "brew install jq 1") {
		t.Errorf("up after end does not show the cursor:\n%s", view(m))
	}
	if m = drive(m, press(tea.KeyHome)); m.offsets[introduction] != 0 {
		t.Errorf("home: offset %d", m.offsets[introduction])
	}
	// down through the rows keeps the cursor in view; on the last, down goes on into the page's words
	m, _ = onCheck(m, "kit-check")
	if v := view(m); !strings.Contains(v, "kit check finds 2 differences") {
		t.Fatalf("the last row is not in view:\n%s", v)
	}
	before := m.offsets[introduction]
	if m = drive(m, down, down); m.offsets[introduction] != before+2 || m.checks[m.checkUnder()].ID != "kit-check" {
		t.Errorf("down on the last row: offset %d from %d, under the cursor %d", m.offsets[introduction], before,
			m.checkUnder())
	}
	// esc, then down from the bar, gives the table's control line the keyboard, in view
	m = drive(m, spin(true), spin(true), esc, down)
	if !strings.Contains(view(m), "Default ▼") {
		t.Errorf("the control line is not in view:\n%s", view(m))
	}
}

// A click past the pane is no click on the table, though its lines, scrolled, reach there.
func TestAClickPastThePaneIsNotTheTables(t *testing.T) {
	m, _ := loaded(t, shortScreen)
	m, _ = onCheck(m, "jq")
	if m = drive(m, testkit.Click(10, paneTop+m.paneHeight())); m.checks[m.checkUnder()].ID != "jq" {
		t.Errorf("a click on the keys bar moved the cursor to %d", m.checkUnder())
	}
}

// slicesIndex gives the index of the first line that holds the text, or -1.
func slicesIndex(lines []string, text string) int {
	for k, l := range lines {
		if strings.Contains(l, text) {
			return k
		}
	}
	return -1
}
