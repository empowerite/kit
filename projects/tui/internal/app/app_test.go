// The root model opens the palette or a form, slides from one to the other, and reports what ran.
package app

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/empowerite/kit/projects/tui/internal/kit"
	"github.com/empowerite/kit/projects/tui/internal/palette"
	"github.com/empowerite/kit/projects/tui/internal/style"
	"github.com/empowerite/kit/projects/tui/internal/testkit"
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

func TestAResizeWhileTheFormIsOpenReachesThePaletteUnderIt(t *testing.T) {
	m, err := Open(testkit.Runner(t, "kit"), "", nil)
	if err != nil {
		t.Fatal(err)
	}
	m = drive(m, tea.WindowSizeMsg{Width: 120, Height: 30}, press(tea.KeyDown), press(tea.KeyEnter),
		tea.WindowSizeMsg{Width: 60, Height: 12}, press(tea.KeyEscape))
	lines := strings.Split(m.View().Content, "\n")
	if m.InForm() || len(lines) != 12 {
		t.Fatalf("back in the palette, %d lines, not 12:\n%s", len(lines), m.View().Content)
	}
	for _, l := range lines {
		if ansi.StringWidth(l) > 60 {
			t.Errorf("a line of %d cells at 60 columns: %q", ansi.StringWidth(l), ansi.Strip(l))
		}
	}
}

// Opening a verb from the palette asks kit for its grammar off the update loop, so a slow kit never freezes the
// screen; the form opens when the grammar comes.
func TestOpeningAVerbAsksKitOffTheUpdateLoop(t *testing.T) {
	m, err := Open(testkit.Runner(t, "kit"), "", nil)
	if err != nil {
		t.Fatal(err)
	}
	m = drive(m, tea.WindowSizeMsg{Width: 120, Height: 30})
	t.Setenv("FAKE_KIT_HANG", "1")
	start := time.Now()
	next, ask := m.Update(palette.OpenMsg{Verb: "commit"})
	m = next.(Model)
	if took := time.Since(start); took > 300*time.Millisecond || ask == nil || m.InForm() {
		t.Fatalf("opening took %s, asked %v, in the form %v", took, ask != nil, m.InForm())
	}
	answer := ask()
	t.Setenv("FAKE_KIT_HANG", "")
	if m = drive(m, answer); !m.InForm() || !strings.Contains(m.View().Content, "kit commit") {
		t.Errorf("the grammar came and the form did not open:\n%s", m.View().Content)
	}
}

func TestAFormOpenedLaterKnowsWhatTheTerminalSaidOfItsKeyboard(t *testing.T) {
	m, err := Open(testkit.Runner(t, "kit"), "", nil)
	if err != nil {
		t.Fatal(err)
	}
	m = drive(m, tea.KeyboardEnhancementsMsg{Flags: ansi.KittyDisambiguateEscapeCodes}, press(tea.KeyDown),
		press(tea.KeyEnter))
	if !m.InForm() || !strings.Contains(m.View().Content, "⇧⏎ run") {
		t.Errorf("the form does not name shift-Enter:\n%s", m.View().Content)
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
	line, code, ok := Refusal(err)
	if !ok || code != 2 || !strings.Contains(line, "no kit verb 'nonesuch'") || strings.HasPrefix(line, "❌") {
		t.Fatalf("Refusal(%v) = %q %d %v", err, line, code, ok)
	}
	if _, _, ok := Refusal(errors.New("not kit's")); ok {
		t.Error("an error not kit's is taken for a refusal")
	}
}

func TestWordsTheVerbRefusesAreKitsRefusal(t *testing.T) {
	t.Setenv("FAKE_KIT_MATCH_REFUSE", "commit")
	_, err := Open(testkit.Runner(t, "kit"), "commit", []string{"--bogus"})
	if line, code, ok := Refusal(err); !ok || code != 2 || !strings.Contains(line, "refuses these words") {
		t.Fatalf("Refusal(%v) = %q %d %v", err, line, code, ok)
	}
}

func TestAUsageThatDoesNotParseOpensNoForm(t *testing.T) {
	t.Setenv("FAKE_KIT_UNPARSED", "commit")
	if _, err := Open(testkit.Runner(t, "kit"), "commit", nil); err == nil || !strings.Contains(err.Error(), "parse") {
		t.Fatalf("a hand-off to a verb whose usage does not parse: err = %v", err)
	}
	// from the palette, the line shows under the filter and the palette stays
	m, err := Open(testkit.Runner(t, "kit"), "", nil)
	if err != nil {
		t.Fatal(err)
	}
	m = drive(m, press(tea.KeyEnter))
	if m.InForm() || !strings.Contains(m.View().Content, "usage does not parse") {
		t.Errorf("the palette does not say why commit did not open:\n%s", m.View().Content)
	}
}

func TestAKitThatCannotListOpensNoPalette(t *testing.T) {
	r := kit.Runner{Kit: filepath.Join(t.TempDir(), "no-kit"), Dir: t.TempDir()}
	if _, err := Open(r, "", nil); err == nil {
		t.Error("a palette opened over a kit that does not run")
	}
	if _, err := OpenHelp(r, "commit"); err == nil {
		t.Error("kit help opened a palette over a kit that does not run")
	}
}

func TestHelpOpensThePaletteOnTheVerbsPage(t *testing.T) {
	m, err := OpenHelp(testkit.Runner(t, "kit"), "submit")
	if err != nil {
		t.Fatal(err)
	}
	m = drive(m, tea.WindowSizeMsg{Width: 120, Height: 30})
	if v := ansi.Strip(m.View().Content); !strings.Contains(v, "kit submit - a purpose for the test") ||
		!strings.Contains(v, style.Focus+" submit") {
		t.Errorf("kit help submit is not on submit's page:\n%s", v)
	}
	// with no name, the palette opens at the top, purposes shown
	m, err = OpenHelp(testkit.Runner(t, "kit"), "")
	if err != nil {
		t.Fatal(err)
	}
	if v := ansi.Strip(drive(m).View().Content); strings.Contains(v, "a purpose for the test") ||
		!strings.Contains(v, style.Focus+" commit") {
		t.Errorf("kit help with no name:\n%s", v)
	}
}

func TestCtrlCIsAnInterruptInThePaletteAndInTheForm(t *testing.T) {
	m, err := Open(testkit.Runner(t, "kit"), "", nil)
	if err != nil {
		t.Fatal(err)
	}
	if drive(m).Interrupted() {
		t.Fatal("interrupted before ctrl-c")
	}
	if !drive(m, tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl}).Interrupted() {
		t.Error("ctrl-c in the palette is not an interrupt")
	}
	if !drive(m, press(tea.KeyDown), press(tea.KeyEnter), tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl}).Interrupted() {
		t.Error("ctrl-c in the form is not an interrupt")
	}
}
