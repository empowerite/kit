// The root model opens the palette or a form, slides from one to the other, and reports what ran.
package app

import (
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/empowerite/kit/tui/internal/kit"
)

func runner(t *testing.T) kit.Runner {
	t.Helper()
	path, err := filepath.Abs(filepath.Join("..", "kit", "testdata", "fake-kit"))
	if err != nil {
		t.Fatal(err)
	}
	return kit.Runner{Kit: path, Dir: t.TempDir()}
}

func drive(m Model, msgs ...tea.Msg) Model {
	queue := append([]tea.Msg{}, msgs...)
	if cmd := m.Init(); cmd != nil {
		queue = append([]tea.Msg{cmd()}, queue...)
	}
	for len(queue) > 0 {
		next, cmd := m.Update(queue[0])
		queue = queue[1:]
		m = next.(Model)
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

func press(code rune) tea.KeyPressMsg { return tea.KeyPressMsg{Code: code} }

func TestPaletteOpensAFormAndTakesItBack(t *testing.T) {
	m, err := Open(runner(t), "", nil)
	if err != nil {
		t.Fatal(err)
	}
	m = drive(m, press(tea.KeyDown), press(tea.KeyEnter))
	if !m.InForm() || !strings.Contains(m.View().Content, "‹ verbs") {
		t.Fatalf("not in the form:\n%s", m.View().Content)
	}
	m = drive(m, press(tea.KeyEscape))
	if m.InForm() {
		t.Fatal("esc did not go back to the palette")
	}
	m = drive(m, press(tea.KeyEscape))
	if run, _, _ := m.Result(); run {
		t.Error("esc in the palette ran something")
	}
}

func TestAHandOffOpensTheFormAloneFilledFromTheWords(t *testing.T) {
	m, err := Open(runner(t), "commit", []string{"--message", "hi"})
	if err != nil {
		t.Fatal(err)
	}
	view := m.View().Content
	if !m.InForm() || strings.Contains(view, "‹ verbs") {
		t.Fatalf("a hand-off has no way back:\n%s", view)
	}
	m = drive(m, tea.KeyPressMsg{Code: tea.KeyEnter, Mod: tea.ModShift})
	run, verb, args := m.Result()
	if !run || verb != "commit" || strings.Join(args, " ") == "" {
		t.Errorf("Result = %v %q %q", run, verb, args)
	}
}

func TestAnUnknownVerbIsKitsRefusal(t *testing.T) {
	_, err := Open(runner(t), "nonesuch", nil)
	var refusal *kit.ExitError
	if err == nil || !strings.Contains(err.Error(), "no kit verb 'nonesuch'") {
		t.Fatalf("err = %v", err)
	}
	_ = refusal
}
