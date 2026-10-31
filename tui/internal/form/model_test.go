// The live form is driven key by key through Update, against the runner's fake kit, as Bubble Tea would drive it.
package form

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/empowerite/kit/tui/internal/kit"
	"github.com/empowerite/kit/tui/internal/syntax"
)

// runner is a kit.Runner over the runner package's fake kit.
func runner(t *testing.T) kit.Runner {
	t.Helper()
	path, err := filepath.Abs(filepath.Join("..", "kit", "testdata", "fake-kit"))
	if err != nil {
		t.Fatal(err)
	}
	return kit.Runner{Kit: path, Dir: t.TempDir()}
}

// key gives the key press a name stands for, as Bubble Tea names them.
func key(name string) tea.KeyPressMsg {
	switch name {
	case "enter":
		return tea.KeyPressMsg{Code: tea.KeyEnter}
	case "shift+enter":
		return tea.KeyPressMsg{Code: tea.KeyEnter, Mod: tea.ModShift}
	case "space":
		return tea.KeyPressMsg{Code: tea.KeySpace, Text: " "}
	case "esc":
		return tea.KeyPressMsg{Code: tea.KeyEscape}
	case "backspace":
		return tea.KeyPressMsg{Code: tea.KeyBackspace}
	case "up":
		return tea.KeyPressMsg{Code: tea.KeyUp}
	case "down":
		return tea.KeyPressMsg{Code: tea.KeyDown}
	case "left":
		return tea.KeyPressMsg{Code: tea.KeyLeft}
	case "right":
		return tea.KeyPressMsg{Code: tea.KeyRight}
	case "ctrl+c":
		return tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl}
	case "ctrl+n":
		return tea.KeyPressMsg{Code: 'n', Mod: tea.ModCtrl}
	}
	return tea.KeyPressMsg{}
}

// typed gives the key presses of a text.
func typed(text string) []tea.Msg {
	var msgs []tea.Msg
	for _, r := range text {
		msgs = append(msgs, tea.KeyPressMsg{Code: r, Text: string(r)})
	}
	return msgs
}

// drive sends the messages to the model, running every command each update returns and feeding its message back,
// as the program would; it gives the model and every message the commands produced.
func drive(t *testing.T, m Model, msgs ...tea.Msg) (Model, []tea.Msg) {
	t.Helper()
	var produced []tea.Msg
	queue := append([]tea.Msg{}, msgs...)
	if cmd := m.Init(); cmd != nil {
		queue = append([]tea.Msg{cmd()}, queue...)
	}
	for len(queue) > 0 {
		msg := queue[0]
		queue = queue[1:]
		next, cmd := m.Update(msg)
		m = next.(Model)
		if cmd != nil {
			out := cmd()
			produced = append(produced, out)
			if _, quit := out.(tea.QuitMsg); !quit {
				if _, back := out.(BackMsg); !back {
					queue = append([]tea.Msg{out}, queue...)
				}
			}
		}
	}
	return m, produced
}

// keys turns names into key presses.
func keys(names ...string) []tea.Msg {
	msgs := make([]tea.Msg, 0, len(names))
	for _, name := range names {
		msgs = append(msgs, key(name))
	}
	return msgs
}

func newForm(t *testing.T, verb string, a Answers, back bool) Model {
	t.Helper()
	spec := specs(t)[verb]
	fields, err := Fields(spec, a)
	if err != nil {
		t.Fatal(err)
	}
	return New(spec, fields, runner(t), back)
}

func TestEnterWalksToCommandLineAndRuns(t *testing.T) {
	m := newForm(t, "commit", FromMatch(syntax.Match{Options: map[string][]string{"--message": {"hi"}},
		Values: map[string][]string{"--message": {"hi"}}}), false)
	m, _ = drive(t, m, keys("enter", "enter", "enter", "enter")...)
	if !strings.Contains(stripANSI(m.View().Content), "kit commit --message hi") {
		t.Errorf("the command line:\n%s", stripANSI(m.View().Content))
	}
	m, produced := drive(t, m, key("enter"))
	run, args := m.Done()
	if !run || strings.Join(args, " ") != "--message hi" {
		t.Fatalf("Done = %v %q", run, args)
	}
	if _, quit := produced[len(produced)-1].(tea.QuitMsg); !quit {
		t.Errorf("no quit: %v", produced)
	}
}

func TestKeyboardLandsOnTheFirstEmptyRequiredField(t *testing.T) {
	m := newForm(t, "commit", Answers{}, false)
	if m.focus != 1 {
		t.Errorf("focus = %d", m.focus)
	}
	m, _ = drive(t, m, keys("right")...)
	m, _ = drive(t, m, keys("down")...)
	m, _ = drive(t, m, typed("#1 tui: x")...)
	if got := strings.Join(m.CommandLine(), "\x00"); got != "--message\x00#1 tui: x" {
		t.Errorf("a line being typed counts: %q", got)
	}
}

func TestOneOfArrows(t *testing.T) {
	m := newForm(t, "package", Answers{}, false)
	m, _ = drive(t, m, keys("right", "right")...)
	if m.fields[0].Chosen != 1 || len(m.fields) != 2 || m.fields[1].Key != "<package>" {
		t.Fatalf("right twice: %+v", m.fields)
	}
	m, _ = drive(t, m, keys("backspace")...)
	if m.fields[0].Chosen != -1 || len(m.fields) != 1 {
		t.Errorf("backspace: %+v", m.fields)
	}
}

func TestAnyOfSpace(t *testing.T) {
	m := newForm(t, "status", Answers{}, false)
	m, _ = drive(t, m, keys("space", "right", "space")...)
	if got := strings.Join(m.CommandLine(), " "); got != "--no-fetch --json" {
		t.Errorf("toggled: %q", got)
	}
	m, _ = drive(t, m, keys("space")...)
	if got := strings.Join(m.CommandLine(), " "); got != "--no-fetch" {
		t.Errorf("toggled back: %q", got)
	}
}

func TestPickManyFilterAndCount(t *testing.T) {
	calls := filepath.Join(t.TempDir(), "calls")
	t.Setenv("FAKE_KIT_CALLS", calls)
	t.Setenv("FAKE_KIT_ROWS", "3000")
	m := newForm(t, "lint", Answers{}, false)
	m, _ = drive(t, m, keys("down")...)
	m, _ = drive(t, m, typed("docs/")...)
	m, _ = drive(t, m, keys("space")...)
	view := stripANSI(m.View().Content)
	if !strings.Contains(view, "1 of 3000 picked") || strings.Contains(view, "src/file-1.go") {
		t.Errorf("filtered and picked:\n%s", view)
	}
	if got := strings.Join(m.CommandLine(), " "); got != "docs/guide.md" {
		t.Errorf("the command line: %q", got)
	}
	recorded, _ := os.ReadFile(calls)
	if n := strings.Count(string(recorded), "choices lint <path>"); n != 1 {
		t.Errorf("%d choices calls:\n%s", n, recorded)
	}
}

func TestStatefulPickIsSeededAndANoteUndoesTheFlip(t *testing.T) {
	m := newForm(t, "package", FromMatch(syntax.Match{Literals: []string{"select"}}), false)
	m, _ = drive(t, m)
	if m.focus != 1 || strings.Join(m.fields[1].Values, " ") != "a b" {
		t.Fatalf("seeded with the active rows: focus %d, %+v", m.focus, m.fields[1])
	}
	m, _ = drive(t, m, keys("down", "space")...)
	if strings.Join(m.fields[1].Values, " ") != "a b" {
		t.Errorf("the flip stood: %+v", m.fields[1])
	}
	if !strings.Contains(stripANSI(m.View().Content), "b: kept, c depend(s) on it") {
		t.Errorf("no note:\n%s", stripANSI(m.View().Content))
	}
	m, _ = drive(t, m, keys("up", "space")...)
	if strings.Join(m.fields[1].Values, " ") != "b" || !strings.Contains(stripANSI(m.View().Content), "➖") {
		t.Errorf("a flipped off:\n%s", stripANSI(m.View().Content))
	}
}

func TestEscCancelsOrGoesBack(t *testing.T) {
	m := newForm(t, "status", Answers{}, false)
	m, produced := drive(t, m, key("esc"))
	if run, args := m.Done(); run || args != nil || len(produced) == 0 {
		t.Errorf("esc without a way back: %v %v %v", run, args, produced)
	}
	m = newForm(t, "status", Answers{}, true)
	_, produced = drive(t, m, key("esc"))
	if _, back := produced[len(produced)-1].(BackMsg); !back {
		t.Errorf("esc with a way back: %v", produced)
	}
	m = newForm(t, "status", Answers{}, false)
	m, _ = drive(t, m, key("ctrl+c"))
	if !m.Interrupted() {
		t.Error("ctrl-c was not an interrupt")
	}
}

func TestNarrowWidthGoesVertical(t *testing.T) {
	m := newForm(t, "status", Answers{}, false)
	m, _ = drive(t, m, tea.WindowSizeMsg{Width: 40, Height: 30})
	view := stripANSI(m.View().Content)
	for _, line := range strings.Split(view, "\n") {
		if strings.Contains(line, "--no-fetch") && strings.Contains(line, "--json") {
			t.Errorf("one row at 40 columns:\n%s", view)
		}
		if n := len([]rune(stripANSI(line))); n > 40 {
			t.Errorf("a line of %d columns: %q", n, line)
		}
	}
}

func TestExtendedFieldsReplaceTheWildcard(t *testing.T) {
	m := newForm(t, "new", FromMatch(syntax.Match{Literals: []string{"file"}, Operands: []string{"docs/x.md"},
		Options: map[string][]string{"--rule": {"markdown"}}, Values: map[string][]string{"<path>": {"docs/x.md"}, "--rule": {"markdown"}}}), false)
	m, _ = drive(t, m)
	var labels []string
	for _, f := range m.fields {
		labels = append(labels, f.Label)
	}
	if strings.Join(labels, " ") != "file <path> --rule --variant «title» «purpose»" {
		t.Fatalf("labels: %q", labels)
	}
	m.focus = 3
	m.enter()
	m, _ = drive(t, m, typed("kit tui")...)
	if got := Quote(m.CommandLine()); got != "file docs/x.md --rule=markdown --title='kit tui'" {
		t.Errorf("the command line: %q", got)
	}
}

func TestShiftEnterRunsFromAnywhere(t *testing.T) {
	m := newForm(t, "status", Answers{}, false)
	m, _ = drive(t, m, key("shift+enter"))
	if run, _ := m.Done(); !run {
		t.Error("shift-enter did not run")
	}
}

// stripANSI drops the escape sequences of styled text, for measuring it.
func stripANSI(s string) string {
	var b strings.Builder
	escape := false
	for _, r := range s {
		switch {
		case r == '\x1b':
			escape = true
		case escape && (r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z'):
			escape = false
		case !escape:
			b.WriteRune(r)
		}
	}
	return b.String()
}
