// Package testkit holds what the packages' tests share: a runner over a package's fake kit, a driver that sends
// messages to a model as the program would, a clock whose timers fire when the test says, the key presses of a text, a
// click and the cell a rune is drawn at, and a golden file compared or rewritten.
// Tests alone import it; -update is its flag, and the test binary that imports it takes the flag.
package testkit

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/empowerite/kit/projects/tui/internal/kit"
)

var update = flag.Bool("update", false, "rewrite the golden files")

// T is what the helpers report through: a test, as *testing.T is, or, in this package's own tests, a recorder.
type T interface {
	Helper()
	TempDir() string
	Fatal(args ...any)
	Fatalf(format string, args ...any)
	Errorf(format string, args ...any)
}

// Runner gives a kit.Runner over internal/<pkg>/testdata/fake-kit, run in a directory of its own.
func Runner(t T, pkg string) kit.Runner {
	t.Helper()
	root, err := moduleRoot()
	if err != nil {
		t.Fatal(err)
	}
	return kit.Runner{Kit: filepath.Join(root, "internal", pkg, "testdata", "fake-kit"), Dir: t.TempDir()}
}

// moduleRoot gives the directory of the go.mod above the working directory, which a test runs in its package's.
func moduleRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("no go.mod above %s", dir)
		}
		dir = parent
	}
}

// Drive sends the messages to the model, running every command each update returns and feeding its message back, a
// batch's each in turn, as the program would. It gives the model and every message the commands produced. A
// tea.QuitMsg, or a message ends says leaves the model, is produced but not fed back; ends may be nil.
func Drive[M tea.Model](m M, ends func(tea.Msg) bool, msgs ...tea.Msg) (M, []tea.Msg) {
	var produced []tea.Msg
	queue := append([]tea.Msg{}, msgs...)
	for len(queue) > 0 {
		msg := queue[0]
		queue = queue[1:]
		if batch, ok := msg.(tea.BatchMsg); ok {
			var out []tea.Msg
			for _, cmd := range batch {
				if cmd == nil {
					continue
				}
				msg := cmd()
				switch {
				case msg == nil:
				case ends != nil && ends(msg):
					produced = append(produced, msg)
				default:
					out = append(out, msg)
				}
			}
			queue = append(out, queue...)
			continue
		}
		next, cmd := m.Update(msg)
		m = next.(M)
		if cmd == nil {
			continue
		}
		out := cmd()
		if out == nil {
			continue
		}
		produced = append(produced, out)
		if _, quit := out.(tea.QuitMsg); quit || ends != nil && ends(out) {
			continue
		}
		queue = append([]tea.Msg{out}, queue...)
	}
	return m, produced
}

// Start gives Init's message, when Init has one, before the messages, for a model driven from its start.
func Start(m tea.Model, msgs ...tea.Msg) []tea.Msg {
	cmd := m.Init()
	if cmd == nil {
		return msgs
	}
	first := cmd()
	if first == nil {
		return msgs
	}
	return append([]tea.Msg{first}, msgs...)
}

// Typed gives the key presses of a text, one per rune.
func Typed(text string) []tea.Msg {
	var msgs []tea.Msg
	for _, r := range text {
		msgs = append(msgs, tea.KeyPressMsg{Code: r, Text: string(r)})
	}
	return msgs
}

// Press gives the key press Bubble Tea names so, as a keys bar's binding names its presses: a named key, with ctrl+ or
// shift+ before it, or a character, typed.
func Press(name string) tea.KeyPressMsg {
	var mod tea.KeyMod
	for prefix, m := range map[string]tea.KeyMod{"ctrl+": tea.ModCtrl, "shift+": tea.ModShift} {
		if rest, ok := strings.CutPrefix(name, prefix); ok && rest != "" {
			name, mod = rest, mod|m
		}
	}
	codes := map[string]rune{"up": tea.KeyUp, "down": tea.KeyDown, "left": tea.KeyLeft, "right": tea.KeyRight,
		"enter": tea.KeyEnter, "esc": tea.KeyEscape, "tab": tea.KeyTab, "space": tea.KeySpace,
		"backspace": tea.KeyBackspace, "pgup": tea.KeyPgUp, "pgdown": tea.KeyPgDown, "home": tea.KeyHome,
		"end": tea.KeyEnd}
	if code, ok := codes[name]; ok {
		k := tea.KeyPressMsg{Code: code, Mod: mod}
		if code == tea.KeySpace {
			k.Text = " "
		}
		return k
	}
	r := []rune(name)[0]
	if mod != 0 {
		return tea.KeyPressMsg{Code: r, Mod: mod}
	}
	return tea.KeyPressMsg{Code: r, Text: name}
}

// Click gives a left click on a cell of the screen.
func Click(x, y int) tea.MouseClickMsg { return tea.MouseClickMsg{X: x, Y: y, Button: tea.MouseLeft} }

// Release gives the release of the left button over a cell of the screen, which ends a click or a drag.
func Release(x, y int) tea.MouseReleaseMsg {
	return tea.MouseReleaseMsg{X: x, Y: y, Button: tea.MouseLeft}
}

// Drag gives the mouse moving over a cell of the screen with the left button held.
func Drag(x, y int) tea.MouseMotionMsg { return tea.MouseMotionMsg{X: x, Y: y, Button: tea.MouseLeft} }

// Column gives the cell of a line where the rune first starts, or -1 when the line lacks it.
func Column(line string, r rune) int {
	k := strings.IndexRune(line, r)
	if k < 0 {
		return -1
	}
	return ansi.StringWidth(line[:k])
}

// Golden compares got with the golden file at path; with -update it rewrites the file instead and says nothing.
func Golden(t T, path, got string) {
	t.Helper()
	if *update {
		if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("%s: %v; run go test -update", path, err)
	}
	if string(want) != got {
		t.Errorf("%s differs:\n%s", path, got)
	}
}
