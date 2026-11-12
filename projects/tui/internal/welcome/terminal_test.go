// The terminal's checks on the setup step: a terminal's findings that hold folded under its heading, and a failed one
// offering later and keep, as kit welcome --keep= keeps it.
package welcome

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/empowerite/kit/projects/tui/internal/kit"
	"github.com/empowerite/kit/projects/tui/internal/testkit"
)

// terminal opens the screen at 120×30 over testdata/terminal/checks.json: on setup, iTerm2's heading, a finding that
// holds, one that failed, and one kept. It selects setup, gives the pane the keyboard, puts the cursor on the failed
// finding, and sends the messages.
func terminal(t *testing.T, msgs ...tea.Msg) (Model, *ran) {
	t.Helper()
	m, r := fake(t)
	fixture, err := os.ReadFile(filepath.Join("testdata", "terminal", "checks.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(m.runner.Dir, "checks.json"), fixture, 0o644); err != nil {
		t.Fatal(err)
	}
	m = drive(m, append(testkit.Start(m, at120), press('3'), down)...)
	for range 5 {
		if m.checks[m.cursor].ID == "terminal-iterm-option" {
			break
		}
		m = drive(m, down)
	}
	if m.checks[m.cursor].ID != "terminal-iterm-option" {
		t.Fatalf("the cursor is on %q", m.checks[m.cursor].ID)
	}
	return drive(m, msgs...), r
}

func TestAHeldTerminalFindingIsFoldedUnderItsHeading(t *testing.T) {
	m, _ := terminal(t)
	got := view(m)
	for _, want := range []string{"iTerm2, profile Default", "sends ⌥ as a plain key"} {
		if !strings.Contains(got, want) {
			t.Errorf("no %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "runs no trigger") {
		t.Errorf("a finding that holds is drawn:\n%s", got)
	}
}

func TestAKeptTerminalFindingStays(t *testing.T) {
	m, _ := terminal(t)
	if got := view(m); !strings.Contains(got, "kept: iTerm2's profile Default does not report mouse clicks") {
		t.Errorf("the kept finding is not drawn:\n%s", got)
	}
}

func TestAFailedTerminalFindingOffersLaterAndKeep(t *testing.T) {
	m, r := terminal(t, press('n'), enter)
	row := rowOf(t, m, "sends ⌥ as a plain key")
	if !strings.HasSuffix(row, "   later   keep ") || strings.Contains(row, " now ") {
		t.Errorf("the row does not offer later and keep alone: %q", row)
	}
	if want := paneKeys + " ⇥·⇧⇥ later or keep l later k keep ⏎ choose"; keysAt(m) != want {
		t.Errorf("keys %q, want %q", keysAt(m), want)
	}
	if len(r.calls) != 0 {
		t.Errorf("n or enter ran %q", r.calls)
	}
	if !m.deferred["terminal-iterm-option"] {
		t.Errorf("enter on later did not leave it for later")
	}
}

func TestKeepRunsKitWelcomeKeepAndReadsTheChecksAgain(t *testing.T) {
	record := filepath.Join(t.TempDir(), "calls")
	t.Setenv("FAKE_WELCOME_CALLS", record)
	m, _ := terminal(t, press('k'))
	got := calls(t, record)
	if len(got) < 3 || got[len(got)-2] != "welcome --keep=terminal-iterm-option" ||
		!strings.HasPrefix(got[len(got)-1], "welcome --checks --json") {
		t.Fatalf("calls %q", got)
	}
	for _, c := range m.checks {
		if c.ID == "terminal-iterm-option" && !c.OK {
			t.Errorf("the kept check still fails")
		}
	}
}

func TestTabThenEnterKeeps(t *testing.T) {
	record := filepath.Join(t.TempDir(), "calls")
	t.Setenv("FAKE_WELCOME_CALLS", record)
	terminal(t, tabKey, enter)
	if got := calls(t, record); !strings.Contains(strings.Join(got, "\n"), "welcome --keep=terminal-iterm-option") {
		t.Errorf("calls %q", got)
	}
}

func TestAClickOnKeepKeepsAndOnLaterLeavesIt(t *testing.T) {
	record := filepath.Join(t.TempDir(), "calls")
	t.Setenv("FAKE_WELCOME_CALLS", record)
	m, _ := terminal(t)
	x, y := buttonAt(t, m, " keep ")
	drive(m, testkit.Click(x, y))
	if got := calls(t, record); !strings.Contains(strings.Join(got, "\n"), "welcome --keep=terminal-iterm-option") {
		t.Errorf("a click on keep ran %q", got)
	}
	m, _ = terminal(t)
	x, y = buttonAt(t, m, " later ")
	if m = drive(m, testkit.Click(x, y)); !m.deferred["terminal-iterm-option"] {
		t.Errorf("a click on later did not leave it for later")
	}
}

func TestAKeepKitRefusesIsNoted(t *testing.T) {
	t.Setenv("FAKE_WELCOME_REFUSE_KEEP", "terminal-iterm-option")
	m, _ := terminal(t, press('k'))
	if got := view(m); !strings.Contains(got, "kit refused to keep it") {
		t.Errorf("no note of the refusal:\n%s", got)
	}
}

func TestAClickOnKeepInTheRefusalNoteKeepsNothing(t *testing.T) {
	record := filepath.Join(t.TempDir(), "calls")
	t.Setenv("FAKE_WELCOME_CALLS", record)
	t.Setenv("FAKE_WELCOME_REFUSE_KEEP", "terminal-iterm-option")
	m, _ := terminal(t, press('k'))
	before := len(calls(t, record))
	note := rowOf(t, m, "kit refused to keep it")
	i := strings.Index(note, " keep ")
	y := slices.Index(strings.Split(view(m), "\n"), note)
	drive(m, testkit.Click(ansi.StringWidth(note[:i])+1, y))
	if got := calls(t, record); len(got) > before && strings.Contains(strings.Join(got[before:], "\n"), "--keep=") {
		t.Errorf("a click on the note kept it again: %q", got[before:])
	}
}

// rowOf gives the drawn line that holds the text.
func rowOf(t *testing.T, m Model, text string) string {
	t.Helper()
	for _, l := range strings.Split(view(m), "\n") {
		if strings.Contains(l, text) {
			return l
		}
	}
	t.Fatalf("no line holds %q:\n%s", text, view(m))
	return ""
}

// buttonAt gives the screen cell where the button's text starts on the cursor's row.
func buttonAt(t *testing.T, m Model, button string) (int, int) {
	t.Helper()
	for y, l := range strings.Split(view(m), "\n") {
		if strings.Contains(l, "sends ⌥ as a plain key") {
			if i := strings.Index(l, button); i >= 0 {
				return ansi.StringWidth(l[:i]) + 1, y
			}
		}
	}
	t.Fatalf("no %q on the cursor's row:\n%s", button, view(m))
	return 0, 0
}

func TestTerminalFindingIsAFindingOrAProbeAndNotAHeading(t *testing.T) {
	for id, want := range map[string]bool{"terminal-iterm": false, "terminal-iterm-option": true,
		"terminal-probe-color": true, "tool-jq": false} {
		if got := terminalFinding(kit.Check{ID: id}); got != want {
			t.Errorf("%s: %v", id, got)
		}
	}
}
