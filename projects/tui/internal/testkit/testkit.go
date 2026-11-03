// Package testkit holds what the packages' tests share: a runner over a package's fake kit, a driver that sends
// messages to a model as the program would, the key presses of a text, and a golden file compared or rewritten.
// Tests alone import it; -update is its flag, and the test binary that imports it takes the flag.
package testkit

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/empowerite/kit/projects/tui/internal/kit"
)

var update = flag.Bool("update", false, "rewrite the golden files")

// Runner gives a kit.Runner over internal/<pkg>/testdata/fake-kit, run in a directory of its own.
func Runner(t *testing.T, pkg string) kit.Runner {
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
				if msg := cmd(); msg != nil {
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

// Golden compares got with the golden file at path; with -update it rewrites the file instead and says nothing.
func Golden(t *testing.T, path, got string) {
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
