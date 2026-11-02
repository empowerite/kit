// The screen is driven key by key through Update, against testdata/fake-kit, which answers the checks from
// testdata/checks.json and the pages from testdata/release, as Bubble Tea would drive it.
package welcome

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/empowerite/kit/tui/internal/testkit"
)

// at120 is the design size.
var at120 = tea.WindowSizeMsg{Width: 120, Height: 30}

func press(code rune) tea.KeyPressMsg { return tea.KeyPressMsg{Code: code} }

var (
	up     = press(tea.KeyUp)
	down   = press(tea.KeyDown)
	left   = press(tea.KeyLeft)
	right  = press(tea.KeyRight)
	enter  = press(tea.KeyEnter)
	esc    = press(tea.KeyEscape)
	pgdown = press(tea.KeyPgDown)
	pgup   = press(tea.KeyPgUp)
)

// ran is what a fix handed the terminal ran: its words after the kit, one call each.
type ran struct{ calls [][]string }

// fake gives the screen over testdata/fake-kit, with its state in a file of its own, so a fixed check stays fixed for
// this test alone. A fix runs at once, its words kept in ran, its output dropped, as the terminal would take it.
func fake(t *testing.T) (Model, *ran) {
	t.Helper()
	t.Setenv("FAKE_WELCOME_STATE", filepath.Join(t.TempDir(), "state"))
	m := New(testkit.Runner(t, "welcome"), "empowerite/kit")
	r := &ran{}
	m.exec = func(cmd *exec.Cmd, done tea.ExecCallback) tea.Cmd {
		r.calls = append(r.calls, cmd.Args[1:])
		return func() tea.Msg { return done(cmd.Run()) }
	}
	return m, r
}

// loaded opens the screen at 120×30, runs Init, then sends the messages.
func loaded(t *testing.T, msgs ...tea.Msg) (Model, *ran) {
	t.Helper()
	m, r := fake(t)
	return drive(m, append(testkit.Start(m, at120), msgs...)...), r
}

// drive sends the messages, running every command each update returns and feeding its messages back.
func drive(m Model, msgs ...tea.Msg) Model {
	m, _ = testkit.Drive(m, nil, msgs...)
	return m
}

// view gives the screen without its escapes.
func view(m Model) string { return ansi.Strip(m.View().Content) }

// calls reads the welcome calls the fake recorded.
func calls(t *testing.T, path string) []string {
	t.Helper()
	got, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	return strings.Split(strings.TrimSpace(string(got)), "\n")
}
