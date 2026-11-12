// The dialog names the clone and what it asks kit to run, and ends as the person chose: Trust runs kit trust --yes on
// the clone, Not now leaves it as it was, by a plain click on either button or by the keyboard; every test runs it
// against testdata/fake-kit.
package trust

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/empowerite/kit/projects/tui/internal/testkit"
)

// opened gives the dialog over the fake, at 80 columns and 30 lines, with the calls to kit trust --yes recorded.
func opened(t *testing.T) (Model, string) {
	t.Helper()
	record := filepath.Join(t.TempDir(), "record")
	t.Setenv("FAKE_TRUST_RECORD", record)
	m := Open(testkit.Runner(t, "trust"), "/the/clone")
	next, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 30})
	return next.(Model), record
}

// shown gives the dialog's view without its colors.
func shown(m Model) string { return ansi.Strip(m.View().Content) }

// said gives the dialog's words on one line, as its prose reads unwrapped.
func said(m Model) string { return strings.Join(strings.Fields(shown(m)), " ") }

// drive sends the messages to the dialog, as the program would, and gives it and whether it quit.
func drive(m Model, msgs ...tea.Msg) (Model, bool) {
	m, produced := testkit.Drive(m, nil, msgs...)
	for _, p := range produced {
		if _, ok := p.(tea.QuitMsg); ok {
			return m, true
		}
	}
	return m, false
}

func TestTheDialogNamesTheCloneAndWhatItAsks(t *testing.T) {
	m, _ := opened(t)
	testkit.Golden(t, "testdata/dialog.golden", shown(m))
	if v := m.View(); !v.AltScreen || v.MouseMode != tea.MouseModeCellMotion {
		t.Error("the dialog is on the alternate screen, with the mouse on")
	}
	if m.Init() != nil {
		t.Error("the dialog starts nothing")
	}
}

// A clone that asks for nothing says so, and one whose asks kit cannot read says why, in place of the list.
func TestWhatTheCloneAsksReadsNothingOrARefusal(t *testing.T) {
	t.Setenv("FAKE_TRUST_ASKS", "none")
	m, _ := opened(t)
	if !strings.Contains(shown(m), "• run nothing beyond the kit itself") {
		t.Errorf("no asks:\n%s", shown(m))
	}
	t.Setenv("FAKE_TRUST_ASKS", "refuse")
	m, _ = opened(t)
	if !strings.Contains(said(m), "what it asks could not be read: kit trust: kit.yml cannot be read for the test") {
		t.Errorf("a refusal:\n%s", shown(m))
	}
}

// The prose wraps to the screen, at most 100 columns, and 80 before the screen's width is known.
func TestTheProseWrapsToTheScreen(t *testing.T) {
	m := Open(testkit.Runner(t, "trust"), "/the/clone")
	for width, want := range map[int]int{0: 80, 40: 40, 200: 100} {
		next, _ := m.Update(tea.WindowSizeMsg{Width: width, Height: 40})
		for _, line := range strings.Split(shown(next.(Model)), "\n") {
			if w := ansi.StringWidth(line); w > want {
				t.Errorf("at %d columns, a line of %d: %q", width, w, line)
			}
		}
	}
	if got := wrap("short averyveryverylongword", 6); len(got) != 2 || got[1] != "  averyveryverylongword" {
		t.Errorf("a word longer than the line: %q", got)
	}
}

// Trust, by enter on it, by t, or by a plain click on it, runs kit trust --yes on the clone and ends the dialog.
func TestTrustRunsKitTrustAndEnds(t *testing.T) {
	m, _ := opened(t)
	_, row := m.lines()
	for name, msgs := range map[string][]tea.Msg{
		"enter": {testkit.Press("enter")},
		"space": {testkit.Press("space")},
		"t":     {testkit.Press("t")},
		"click": {testkit.Click(3, row)},
		"back":  {testkit.Press("right"), testkit.Press("left"), testkit.Press("enter")},
	} {
		m, record := opened(t)
		m, quit := drive(m, msgs...)
		said, err := os.ReadFile(record)
		if !quit || !m.Trusted() || m.Interrupted() || err != nil || string(said) != "trust --yes /the/clone\n" {
			t.Errorf("%s: quit %v, trusted %v, kit ran %q (%v)", name, quit, m.Trusted(), said, err)
		}
	}
}

// Not now, by esc, by n, by enter on it, or by a plain click on it, ends the dialog and runs nothing.
func TestNotNowEndsAndRunsNothing(t *testing.T) {
	m, _ := opened(t)
	_, row := m.lines()
	x := strings.Index(strings.Split(shown(m), "\n")[row], "[ "+NotNowButton)
	for name, msgs := range map[string][]tea.Msg{
		"esc":   {testkit.Press("esc")},
		"n":     {testkit.Press("n")},
		"tab":   {testkit.Press("tab"), testkit.Press("enter")},
		"click": {testkit.Click(x+1, row)},
	} {
		m, record := opened(t)
		m, quit := drive(m, msgs...)
		if _, err := os.Stat(record); !quit || m.Trusted() || m.Interrupted() || err == nil {
			t.Errorf("%s: quit %v, trusted %v, kit ran (%v)", name, quit, m.Trusted(), err)
		}
	}
}

// A click elsewhere, on the button line between the buttons or off it, or with another button than the left, and a key
// the dialog does not take, change nothing.
func TestAClickOffTheButtonsAndAnotherKeyChangeNothing(t *testing.T) {
	m, record := opened(t)
	_, row := m.lines()
	for _, msg := range []tea.Msg{testkit.Click(0, row), testkit.Click(12, row), testkit.Click(70, row),
		testkit.Click(3, row-1), tea.MouseClickMsg{X: 3, Y: row, Button: tea.MouseRight}, testkit.Press("x")} {
		if next, quit := drive(m, msg); quit || next.Trusted() || shown(next) != shown(m) {
			t.Errorf("%v changed the dialog", msg)
		}
	}
	if _, err := os.Stat(record); err == nil {
		t.Error("kit ran")
	}
}

// ctrl-c ends the dialog as interrupted, trusting nothing.
func TestCtrlCInterrupts(t *testing.T) {
	m, _ := opened(t)
	m, quit := drive(m, tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl})
	if !quit || !m.Interrupted() || m.Trusted() {
		t.Errorf("quit %v, interrupted %v, trusted %v", quit, m.Interrupted(), m.Trusted())
	}
}

// While kit trust --yes runs, the dialog says so and a second press runs nothing more; a refusal stays on the dialog,
// said under the buttons, and Trust can be pressed again.
func TestARefusalOfTrustStaysOnTheDialog(t *testing.T) {
	m, _ := opened(t)
	next, cmd := m.Update(testkit.Press("enter"))
	m = next.(Model)
	if cmd == nil || !strings.Contains(shown(m), "trusting /the/clone…") {
		t.Fatalf("no run, or not said:\n%s", shown(m))
	}
	if _, again := m.Update(testkit.Press("enter")); again != nil {
		t.Error("a second press ran kit again")
	}
	next, cmd = m.Update(trustedMsg{errors.New("the trust file cannot be written for the test")})
	m = next.(Model)
	if cmd != nil || m.Trusted() || !strings.Contains(said(m), "❌ the trust file cannot be written for the test") ||
		strings.Contains(shown(m), "trusting") {
		t.Errorf("the refusal:\n%s", shown(m))
	}
	t.Setenv("FAKE_TRUST_REFUSE", "1")
	m, quit := drive(m, testkit.Press("enter"))
	if quit || m.Trusted() || !strings.Contains(said(m), "the trust file cannot be written for the test") {
		t.Errorf("kit's refusal:\n%s", shown(m))
	}
}

// Not now leaves kittui with kit's own line, its remedy run from where kittui runs.
func TestUntrustedIsKitsLine(t *testing.T) {
	got := Untrusted("/the/clone")
	if !strings.HasPrefix(got, "kit runs nothing in /the/clone until you trust it; see what it asks kit to run, and "+
		"trust it: ") || !strings.HasSuffix(got, "kit trust") {
		t.Errorf("%q", got)
	}
}
