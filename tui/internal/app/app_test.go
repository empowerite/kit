// The root model opens the palette or a form, slides from one to the other, and reports what ran.
package app

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/empowerite/kit/tui/internal/kit"
	"github.com/empowerite/kit/tui/internal/testkit"
)

// drive sends the messages to the root model from its start, as the program would.
func drive(m Model, msgs ...tea.Msg) Model {
	m, _ = testkit.Drive(m, nil, testkit.Start(m, msgs...)...)
	return m
}

func press(code rune) tea.KeyPressMsg { return tea.KeyPressMsg{Code: code} }

func TestPaletteOpensAFormAndTakesItBack(t *testing.T) {
	m, err := Open(testkit.Runner(t, "kit"), "", nil)
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
	m, err := Open(testkit.Runner(t, "kit"), "commit", []string{"--message", "hi"})
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

func TestAVerbWhoseFormCannotOpenLeavesKitsLineInThePalette(t *testing.T) {
	t.Setenv("FAKE_KIT_REFUSE", "submit")
	m, err := Open(testkit.Runner(t, "kit"), "", nil)
	if err != nil {
		t.Fatal(err)
	}
	m = drive(m, press(tea.KeyDown), press(tea.KeyEnter))
	view := m.View().Content
	if m.InForm() || !strings.Contains(view, "❌ kit submit refuses for the test; run: kit help submit") {
		t.Fatalf("the palette does not show why submit did not open:\n%s", view)
	}
	// another verb opens as ever, and takes the line with it
	m = drive(m, press(tea.KeyUp), press(tea.KeyEnter))
	if !m.InForm() {
		t.Fatalf("commit did not open after the refusal:\n%s", m.View().Content)
	}
	m = drive(m, press(tea.KeyEscape))
	if strings.Contains(m.View().Content, "refuses for the test") {
		t.Errorf("the line outlived the verb that caused it:\n%s", m.View().Content)
	}
}

func TestAnUnknownVerbIsKitsRefusal(t *testing.T) {
	_, err := Open(testkit.Runner(t, "kit"), "nonesuch", nil)
	var refusal *kit.ExitError
	if err == nil || !strings.Contains(err.Error(), "no kit verb 'nonesuch'") {
		t.Fatalf("err = %v", err)
	}
	_ = refusal
}
