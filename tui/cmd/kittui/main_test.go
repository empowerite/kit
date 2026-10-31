// The binary refuses without a terminal, and ends as the form ended: silent on esc and ctrl-c, with the transcript
// before the verb on run.
package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/empowerite/kit/tui/internal/app"
	"github.com/empowerite/kit/tui/internal/kit"
)

func runner(t *testing.T) kit.Runner {
	t.Helper()
	path, err := filepath.Abs(filepath.Join("..", "..", "internal", "kit", "testdata", "fake-kit"))
	if err != nil {
		t.Fatal(err)
	}
	return kit.Runner{Kit: path, Dir: t.TempDir()}
}

func drive(m app.Model, msgs ...tea.Msg) app.Model {
	queue := append([]tea.Msg{}, msgs...)
	if cmd := m.Init(); cmd != nil {
		queue = append([]tea.Msg{cmd()}, queue...)
	}
	for len(queue) > 0 {
		next, cmd := m.Update(queue[0])
		queue = queue[1:]
		m = next.(app.Model)
		if cmd != nil {
			if out := cmd(); out != nil {
				if _, quit := out.(tea.QuitMsg); !quit {
					queue = append([]tea.Msg{out}, queue...)
				}
			}
		}
	}
	return m
}

func TestNoTerminalRefuses(t *testing.T) {
	binary := filepath.Join(t.TempDir(), "kittui")
	if out, err := exec.Command("go", "build", "-o", binary, ".").CombinedOutput(); err != nil {
		t.Fatalf("build: %v\n%s", err, out)
	}
	cmd := exec.Command(binary, "--", "commit")
	cmd.Stdin = nil
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	err := cmd.Run()
	exit, ok := err.(*exec.ExitError)
	if !ok || exit.ExitCode() != 2 || stderr.String() != "❌ kit tui needs an interactive terminal; run: kit help tui\n" {
		t.Fatalf("err %v, stderr %q", err, stderr.String())
	}
}

func TestCtrlCIsSilent(t *testing.T) {
	for _, verb := range []string{"", "commit"} {
		m, err := app.Open(runner(t), verb, nil)
		if err != nil {
			t.Fatal(err)
		}
		m = drive(m, tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl})
		var stderr bytes.Buffer
		if code := finish(m, &stderr, nil); code != 130 || stderr.Len() != 0 {
			t.Errorf("%q: exit %d, stderr %q", verb, code, stderr.String())
		}
	}
}

func TestEscIsSilent(t *testing.T) {
	m, _ := app.Open(runner(t), "commit", nil)
	m = drive(m, tea.KeyPressMsg{Code: tea.KeyEscape})
	var stderr bytes.Buffer
	if code := finish(m, &stderr, nil); code != 1 || stderr.Len() != 0 {
		t.Errorf("exit %d, stderr %q", code, stderr.String())
	}
}

func TestRunPrintsTheTranscriptThenExecs(t *testing.T) {
	m, _ := app.Open(runner(t), "commit", []string{"--message", "hi"})
	m = drive(m, tea.KeyPressMsg{Code: tea.KeyEnter, Mod: tea.ModShift})
	var stderr bytes.Buffer
	var ran []string
	code := finish(m, &stderr, func(verb string, args []string) error {
		ran = append([]string{verb}, args...)
		return nil
	})
	if code != 0 || len(ran) == 0 || ran[0] != "commit" {
		t.Fatalf("exit %d, ran %q", code, ran)
	}
	if !strings.Contains(stderr.String(), "✓ --all") || !strings.Contains(stderr.String(), "  kit commit ") {
		t.Errorf("transcript %q", stderr.String())
	}
}

func TestWordsAfterADoubleDash(t *testing.T) {
	verb, words := parse([]string{"--", "commit", "--all"})
	if verb != "commit" || strings.Join(words, " ") != "--all" {
		t.Errorf("parse = %q %q", verb, words)
	}
	verb, words = parse(nil)
	if verb != "" || words != nil {
		t.Errorf("parse(nil) = %q %q", verb, words)
	}
	_ = os.Stdin
}
