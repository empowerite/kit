// The live form is driven key by key through Update, against the runner's fake kit, as Bubble Tea would drive it.
package form

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/empowerite/kit/projects/tui/internal/syntax"
	"github.com/empowerite/kit/projects/tui/internal/testkit"
)

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

// drive sends the messages to the form from its start, as the program would, and gives the model and every message
// the commands produced; a BackMsg ends the form, so it is produced and not fed back.
func drive(t *testing.T, m Model, msgs ...tea.Msg) (Model, []tea.Msg) {
	t.Helper()
	back := func(msg tea.Msg) bool { _, ok := msg.(BackMsg); return ok }
	return testkit.Drive(m, back, testkit.Start(m, msgs...)...)
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
	return New(spec, fields, testkit.Runner(t, "kit"), back)
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
	m, _ = drive(t, m, testkit.Typed("#1 tui: x")...)
	if got := strings.Join(m.CommandLine(), "\x00"); got != "--message\x00#1 tui: x" {
		t.Errorf("a line being typed counts: %q", got)
	}
}

func TestOneOfArrows(t *testing.T) {
	m := newForm(t, "packages", Answers{}, false)
	m, _ = drive(t, m, keys("right", "right")...)
	if m.fields[0].Chosen != 1 || len(m.fields) != 3 || m.fields[1].Key != "select | add | remove" || m.fields[2].Key != "<package>" {
		t.Fatalf("right twice: %+v", m.fields)
	}
	// kit packages needs one of its lines, so backspace leaves the choice where it is.
	m, _ = drive(t, m, keys("backspace")...)
	if m.fields[0].Chosen != 1 || len(m.fields) != 3 {
		t.Errorf("backspace: %+v", m.fields)
	}
	// kit flow's owner row, under the picture's line, is optional, so backspace clears it.
	m = newForm(t, "flow", Answers{Chosen: map[int]int{0: 0}}, false)
	m, _ = drive(t, m, keys("down", "down", "down", "right")...)
	if m.fields[3].Chosen != 0 {
		t.Fatalf("right on the owner row: %+v", m.fields[3])
	}
	m, _ = drive(t, m, keys("backspace")...)
	if m.fields[3].Chosen != -1 {
		t.Errorf("backspace on an optional row: %+v", m.fields[3])
	}
}

func TestAnyOfSpace(t *testing.T) {
	// flow's owner row, under the picture's line, its second alternative chosen, opens an any-of row of --mine and
	// --others
	m := newForm(t, "flow", Answers{Chosen: map[int]int{0: 0}}, false)
	m, _ = drive(t, m, keys("down", "down", "down", "right", "right", "down", "space", "right", "space")...)
	if got := strings.Join(m.CommandLine(), " "); got != "--mine --others" {
		t.Errorf("toggled: %q", got)
	}
	m, _ = drive(t, m, keys("space")...)
	if got := strings.Join(m.CommandLine(), " "); got != "--mine" {
		t.Errorf("toggled back: %q", got)
	}
}

func TestJSONAndJQAreOneOrTheOther(t *testing.T) {
	// the picture's output row is --text | --json | --jq: right moves the choice along it, so one flag stands at a time
	m := newForm(t, "flow", Answers{Chosen: map[int]int{0: 0}}, false)
	m, _ = drive(t, m, keys("down", "down", "right", "right")...)
	if got := strings.Join(m.CommandLine(), " "); got != "--json" {
		t.Errorf("the output row on --json: %q", got)
	}
	m, _ = drive(t, m, keys("right")...)
	if got := strings.Join(m.CommandLine(), " "); got != "--jq" {
		t.Errorf("the output row on --jq: %q", got)
	}
}

func TestPickManyFilterAndCount(t *testing.T) {
	calls := filepath.Join(t.TempDir(), "calls")
	t.Setenv("FAKE_KIT_CALLS", calls)
	t.Setenv("FAKE_KIT_ROWS", "3000")
	m := newForm(t, "lint", Answers{}, false)
	m, _ = drive(t, m, keys("down")...)
	m, _ = drive(t, m, testkit.Typed("docs/")...)
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
	m := newForm(t, "packages", Answers{Literals: []string{"select"}, Chosen: map[int]int{0: 1}}, false)
	m, _ = drive(t, m)
	if m.focus != 2 || strings.Join(m.fields[2].Values, " ") != "a b" {
		t.Fatalf("seeded with the active rows: focus %d, %+v", m.focus, m.fields[2])
	}
	m, _ = drive(t, m, keys("down", "space")...)
	if strings.Join(m.fields[2].Values, " ") != "a b" {
		t.Errorf("the flip stood: %+v", m.fields[2])
	}
	if !strings.Contains(stripANSI(m.View().Content), "b: kept, c depend(s) on it") {
		t.Errorf("no note:\n%s", stripANSI(m.View().Content))
	}
	m, _ = drive(t, m, keys("up", "space")...)
	if strings.Join(m.fields[2].Values, " ") != "b" || !strings.Contains(stripANSI(m.View().Content), "➖") {
		t.Errorf("a flipped off:\n%s", stripANSI(m.View().Content))
	}
	if !strings.Contains(stripANSI(m.View().Content), "➕ 0 to add · ➖ 1 to remove · ✅ active · ␣ flips") {
		t.Errorf("the hint does not name the marks drawn:\n%s", stripANSI(m.View().Content))
	}
}

func TestEscCancelsOrGoesBack(t *testing.T) {
	m := newForm(t, "flow", Answers{}, false)
	m, produced := drive(t, m, key("esc"))
	if run, args := m.Done(); run || args != nil || len(produced) == 0 {
		t.Errorf("esc without a way back: %v %v %v", run, args, produced)
	}
	m = newForm(t, "flow", Answers{}, true)
	_, produced = drive(t, m, key("esc"))
	if _, back := produced[len(produced)-1].(BackMsg); !back {
		t.Errorf("esc with a way back: %v", produced)
	}
	m = newForm(t, "flow", Answers{}, false)
	m, _ = drive(t, m, key("ctrl+c"))
	if !m.Interrupted() {
		t.Error("ctrl-c was not an interrupt")
	}
}

func TestNarrowWidthGoesVertical(t *testing.T) {
	// the picture's output row draws its names side by side at 80 columns, and one per line at 40, where its names
	// would not fit; the line's own name, the usage line whole, is the one line that may still say --json and --jq
	m := newForm(t, "flow", Answers{Chosen: map[int]int{0: 0}}, false)
	if view := stripANSI(m.View().Content); !strings.Contains(view, "--text   --json   --jq") {
		t.Fatalf("not one row at 80 columns:\n%s", view)
	}
	m, _ = drive(t, m, tea.WindowSizeMsg{Width: 40, Height: 0})
	view := stripANSI(m.View().Content)
	if strings.Contains(view, "--text   --json") {
		t.Errorf("one row at 40 columns:\n%s", view)
	}
	for _, line := range strings.Split(view, "\n") {
		// wrap keeps a word wider than the room beside the column whole, alone on its line, as the flow package's
		// wrap does: the one line that may pass the edge, here "uncommitted" beside a column of 30
		if n := len([]rune(line)); n > 40 && strings.Contains(strings.TrimSpace(line), " ") {
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
	if !strings.HasPrefix(labels[0], "file | kind |") || strings.Join(labels[1:], " ") != "<path> --rule --variant «title» «purpose»" {
		t.Fatalf("labels: %q", labels)
	}
	m.focus = 4
	m.enter()
	m, _ = drive(t, m, testkit.Typed("kit tui")...)
	if got := Quote(m.CommandLine()); got != "file docs/x.md --rule=markdown --title='kit tui'" {
		t.Errorf("the command line: %q", got)
	}
}

func TestShiftEnterRunsFromAnywhere(t *testing.T) {
	m := newForm(t, "flow", Answers{}, false)
	m, _ = drive(t, m, key("shift+enter"))
	if run, _ := m.Done(); !run {
		t.Error("shift-enter did not run")
	}
}

func TestTheKeysBarNamesShiftEnterOnlyWhereTheTerminalTellsItApart(t *testing.T) {
	m := newForm(t, "flow", Answers{}, true)
	m, _ = drive(t, m, tea.WindowSizeMsg{Width: 80, Height: 30})
	if bar := screen(m)[29]; strings.Contains(bar, "⇧⏎") || !strings.HasPrefix(bar, " esc back · ↑↓ move · ⏎ next │ ") {
		t.Errorf("a terminal that said nothing of its keyboard, as Terminal.app: %q", bar)
	}
	m, _ = drive(t, m, tea.KeyboardEnhancementsMsg{})
	if bar := screen(m)[29]; strings.Contains(bar, "⇧⏎") {
		t.Errorf("a terminal that answered with no flags: %q", bar)
	}
	m, _ = drive(t, m, disambiguates)
	if bar := screen(m)[29]; !strings.HasPrefix(bar, " esc back · ↑↓ move · ⏎ next · ⇧⏎ run │ ") {
		t.Errorf("a terminal that tells them apart: %q", bar)
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

func TestAddStartsEmptyAndDrawsEachRowsStateBesideItsBox(t *testing.T) {
	m := newForm(t, "packages", Answers{Literals: []string{"add"}, Chosen: map[int]int{0: 1}}, false)
	m, _ = drive(t, m)
	if len(m.fields[2].Values) != 0 || strings.Join(m.CommandLine(), " ") != "add" {
		t.Fatalf("add was seeded: %+v, %q", m.fields[2], m.CommandLine())
	}
	view := stripANSI(m.View().Content)
	if !strings.Contains(view, "◻ ✅ a") || strings.Contains(view, "➖") ||
		!strings.Contains(view, "0 of 3 picked · ✅ active · ␣ pick") {
		t.Errorf("add's list:\n%s", view)
	}
}

func TestHintsWrapAtFiftyColumns(t *testing.T) {
	for _, c := range []struct {
		verb string
		a    Answers
		keys []string
	}{
		{"lint", Answers{}, []string{"down"}},
		{"commit", Answers{}, []string{"right", "down"}},
		{"packages", Answers{Literals: []string{"select"}, Chosen: map[int]int{0: 1}}, nil},
	} {
		m := newForm(t, c.verb, c.a, false)
		m, _ = drive(t, m, tea.WindowSizeMsg{Width: 50, Height: 40})
		m, _ = drive(t, m, keys(c.keys...)...)
		for _, line := range strings.Split(stripANSI(m.View().Content), "\n") {
			if n := len([]rune(line)); n > 50 {
				t.Errorf("%s: a line of %d columns: %q", c.verb, n, line)
			}
		}
	}
}

func TestAFailedChoicesCallIsShown(t *testing.T) {
	m := newForm(t, "lint", Answers{}, false)
	m, _ = drive(t, m, key("down"), choicesMsg{key: "<path>", err: errors.New("kit lint has no choices for '<path>'")})
	if !strings.Contains(stripANSI(m.View().Content), "kit lint has no choices for '<path>'") {
		t.Errorf("the error is not shown:\n%s", stripANSI(m.View().Content))
	}
}

func TestAListKitCannotGiveShowsItsLineAndTakesWhatIsTyped(t *testing.T) {
	t.Setenv("FAKE_KIT_REFUSE", "lint")
	m := newForm(t, "lint", Answers{}, false)
	m, _ = drive(t, m, key("down"))
	view := stripANSI(m.View().Content)
	if !strings.Contains(view, "❌ kit lint refuses for the test; run: kit help lint") || strings.Contains(view, "❌ ❌") {
		t.Fatalf("kit's line is not the note, once:\n%s", view)
	}
	if !strings.Contains(view, "⏎ takes what is typed") {
		t.Errorf("the field does not say it takes free text:\n%s", view)
	}
	m, _ = drive(t, m, testkit.Typed("docs/x.md")...)
	m, _ = drive(t, m, key("enter"))
	if got := strings.Join(m.CommandLine(), " "); got != "docs/x.md" {
		t.Errorf("the command line: %q", got)
	}
	if view = stripANSI(m.View().Content); !strings.Contains(view, "◼ docs/x.md") || m.focus != 1 {
		t.Errorf("the value taken is not shown under the field, with the keyboard still on it:\n%s", view)
	}
}

func TestAFailedExtensionIsTheFocusedFieldsNote(t *testing.T) {
	t.Setenv("FAKE_KIT_REFUSE", "new")
	m := newForm(t, "new", FromMatch(syntax.Match{Literals: []string{"file"}}), false)
	m, _ = drive(t, m)
	line := "❌ kit new refuses for the test; run: kit help new"
	view := stripANSI(m.View().Content)
	path, note, rule := strings.Index(view, "<path>"), strings.Index(view, line), strings.Index(view, "--rule")
	if m.focus != 1 || path < 0 || note < path || rule < note {
		t.Fatalf("the line is not under <path>, the focused field:\n%s", view)
	}
	m, _ = drive(t, m, key("down"))
	if !strings.Contains(stripANSI(m.View().Content), line) {
		t.Errorf("the next field, asked again, shows no line:\n%s", stripANSI(m.View().Content))
	}
}

// screen gives the form's lines without their escapes.
func screen(m Model) []string { return strings.Split(stripANSI(m.View().Content), "\n") }

// disambiguates is a terminal's answer that it tells shift-Enter from Enter, as kitty, iTerm2, and VS Code's do.
var disambiguates = tea.KeyboardEnhancementsMsg{Flags: ansi.KittyDisambiguateEscapeCodes}

func TestTheWayBackIsOnTopTheKeysLastAndThePurposeNamesTheCommand(t *testing.T) {
	m := newForm(t, "flow", Answers{}, true)
	m, _ = drive(t, m, disambiguates)
	lines := screen(m)
	bar := lines[len(lines)-1]
	if strings.TrimSpace(lines[0]) != "‹ verbs" || !strings.HasPrefix(bar, " esc back · ↑↓ move · ⏎ next · ⇧⏎ run │ ") {
		t.Errorf("with the way back, the first line %q, the last %q", lines[0], bar)
	}
	if !strings.Contains(stripANSI(m.View().Content), "\n kit flow - "+m.spec.Purpose[:20]) {
		t.Errorf("the purpose does not name the command:\n%s", stripANSI(m.View().Content))
	}
	m = newForm(t, "flow", Answers{}, false)
	lines = screen(m)
	if !strings.HasPrefix(lines[0], " kit flow - ") || !strings.HasPrefix(lines[len(lines)-1], " esc cancel · ") {
		t.Errorf("without the way back, the first line %q, the last %q", lines[0], lines[len(lines)-1])
	}
}

func TestTheKeysBarIsTheLastLineWhereverTheKeyboardIs(t *testing.T) {
	// the form's keys, then the field's; too narrow for both, the field's alone
	const text = " esc back · ↑↓ move · ⏎ next · ⇧⏎ run │ ⌫ erase"
	const many = " … │ ⇞⇟ page · ␣ toggle · ⌫ erase"
	const command = " esc back │ ↑ move · ⏎ run"
	long := strings.Repeat("kit commit refuses for the test, at length; ", 6) + "run: kit help commit"
	for _, height := range []int{40, 12, 7, 3, 2, 1} {
		for _, c := range []struct {
			verb    string
			a       Answers
			keys    []string
			command bool   // the keyboard on the command line, with a long refusal under it
			want    string // the bar, at 50 columns
		}{
			{"commit", Answers{}, []string{"right", "down"}, false, text}, // --message's lines, with their hint
			{"lint", Answers{}, []string{"down"}, false, many},            // a list of 3000 rows, with its count
			{"packages", Answers{Literals: []string{"select"}, Chosen: map[int]int{0: 1}}, nil, false, many},
			{"flow", Answers{}, nil, true, command},
		} {
			t.Setenv("FAKE_KIT_ROWS", "")
			if c.verb == "lint" {
				t.Setenv("FAKE_KIT_ROWS", "3000")
			}
			m := newForm(t, c.verb, c.a, true)
			m, _ = drive(t, m, tea.WindowSizeMsg{Width: 50, Height: height}, disambiguates)
			m, _ = drive(t, m, keys(c.keys...)...)
			if c.command {
				m.focus, m.refusal = len(m.fields), long
				m.enter()
			}
			if lines := screen(m); len(lines) != height || lines[height-1] != c.want {
				t.Errorf("%s %v at %d: %d lines, the last %q", c.verb, c.keys, height, len(lines), lines[len(lines)-1])
			}
		}
	}
	// at 80 columns the bar is whole; a height not yet known draws the form, a gap, and the bar
	m := newForm(t, "commit", Answers{}, true)
	m, _ = drive(t, m, tea.WindowSizeMsg{Width: 80, Height: 0}, disambiguates)
	m, _ = drive(t, m, keys("right", "down")...)
	lines := screen(m)
	if lines[len(lines)-1] != text || lines[len(lines)-2] != "" || !strings.Contains(lines[len(lines)-3], "kit commit") {
		t.Errorf("with no height, the last lines: %q", lines[len(lines)-3:])
	}
	m, _ = drive(t, m, tea.WindowSizeMsg{Width: 80, Height: 30})
	if lines = screen(m); len(lines) != 30 || lines[29] != text {
		t.Errorf("at 80×30: %d lines, the last %q", len(lines), lines[len(lines)-1])
	}
}

// On every field of every verb, the keys bar names the keys that field takes, and no other: the arrows across only
// where they move across options, the space where it toggles, backspace where it erases or clears, and the page keys
// only on a list.
func TestTheKeysBarNamesOnlyTheKeysTheFieldTakes(t *testing.T) {
	for verb, spec := range specs(t) {
		fields, err := Fields(spec, Answers{})
		if err != nil {
			continue
		}
		m := New(spec, fields, testkit.Runner(t, "kit"), true)
		for i, f := range m.fields {
			m.focus = i
			bar := m.keys().String()
			across := f.Kind == AnyOf || f.Kind == OneOf
			space := f.Kind == AnyOf || f.Kind == PickMany || f.Kind == Toggle && !f.Required
			erase := f.Kind == Text || f.Kind == Lines || f.Kind == Wildcard || f.Kind == Pick || f.Kind == PickMany ||
				f.Kind == OneOf && !f.Required
			page := f.Kind == Pick || f.Kind == PickMany
			for _, c := range []struct {
				key  string
				want bool
			}{{"←→", across}, {"␣", space}, {"⌫", erase}, {"⇞⇟", page}} {
				if strings.Contains(bar, c.key) != c.want {
					t.Errorf("%s %s (kind %d, required %v): %q names %s: %v", verb, f.Key, f.Kind, f.Required, bar,
						c.key, !c.want)
				}
			}
		}
		m.focus = len(m.fields)
		if bar := m.keys().String(); bar != "esc back │ ↑ move · ⏎ run" {
			t.Errorf("%s on the command line: %q", verb, bar)
		}
	}
}

func TestQIsTypedNotAQuit(t *testing.T) {
	m := newForm(t, "commit", Answers{}, true)
	m, _ = drive(t, m, keys("right", "down")...)
	m, produced := drive(t, m, testkit.Typed("q")...)
	if len(produced) != 0 || m.input != "q" {
		t.Errorf("q: %v, input %q", produced, m.input)
	}
}

func TestAShortTerminalKeepsTheKeysAndTheFocusedField(t *testing.T) {
	m := newForm(t, "new", FromMatch(syntax.Match{Literals: []string{"file"}}), true)
	m, _ = drive(t, m, tea.WindowSizeMsg{Width: 80, Height: 9})
	whole := strings.Count(strings.TrimRight(stripANSI(m.View().Content), "\n"), "\n") + 1
	if whole != 9 {
		t.Fatalf("%d lines on a screen of 9:\n%s", whole, stripANSI(m.View().Content))
	}
	for _, msgs := range [][]tea.Msg{nil, keys("down"), keys("down"), keys("down", "down", "down")} {
		m, _ = drive(t, m, msgs...)
		lines := screen(m)
		if len(lines) != 9 {
			t.Errorf("focus %d: %d lines on a screen of 9", m.focus, len(lines))
		}
		if strings.TrimSpace(lines[0]) != "‹ verbs" || !strings.Contains(lines[8], "↑↓ move") {
			t.Errorf("focus %d: the way back or the keys moved: %q, %q", m.focus, lines[0], lines[8])
		}
		focused := m.focus >= len(m.fields) && strings.Contains(lines[len(lines)-3], "kit new")
		for _, line := range lines {
			if strings.Contains(line, "▸") {
				focused = true
			}
		}
		if !focused {
			t.Errorf("focus %d: the focused field is off the screen:\n%s", m.focus, strings.Join(lines, "\n"))
		}
	}
	// a tall terminal clips nothing
	m, _ = drive(t, m, tea.WindowSizeMsg{Width: 80, Height: 60})
	if n := strings.Count(stripANSI(m.View().Content), "\n"); n <= 9 {
		t.Errorf("a tall terminal shows %d lines", n)
	}
}

// rows40 opens kit lint's form on a list of 40 paths, docs/guide.md then src/file-1.go to src/file-39.go, with the
// keyboard on the list, at 80 columns and the height.
func rows40(t *testing.T, height int) Model {
	t.Helper()
	t.Setenv("FAKE_KIT_ROWS", "40")
	m := newForm(t, "lint", Answers{}, true)
	m, _ = drive(t, m, tea.WindowSizeMsg{Width: 80, Height: height})
	m, _ = drive(t, m, key("down"))
	if n := len(m.filtered()); n != 40 {
		t.Fatalf("%d rows", n)
	}
	return m
}

// path gives the value of the list's row i.
func path(i int) string {
	if i == 0 {
		return "docs/guide.md"
	}
	return fmt.Sprintf("src/file-%d.go", i)
}

// hidden gives the lines that say how many rows a window hides, as they read.
func hidden(lines []string) []string {
	var out []string
	for _, line := range lines {
		if marker.MatchString(line) {
			out = append(out, strings.TrimSpace(line))
		}
	}
	return out
}

var marker = regexp.MustCompile(`^ *… \d+ (above|more)$`)

func TestPageKeysMoveTheListsCursorAPageAndTheWindowFollows(t *testing.T) {
	m := rows40(t, 40)
	for _, c := range []struct {
		key    string
		cursor int
	}{
		{"pgdown", 9}, {"pgdown", 18}, {"pgdown", 27}, {"pgdown", 36}, {"pgdown", 39}, {"pgdown", 39},
		{"pgup", 30}, {"pgup", 21}, {"pgup", 12}, {"pgup", 3}, {"pgup", 0}, {"pgup", 0},
		{"end", 39}, {"end", 39}, {"home", 0}, {"home", 0},
	} {
		m, _ = drive(t, m, tea.KeyPressMsg{Code: map[string]rune{"pgdown": tea.KeyPgDown, "pgup": tea.KeyPgUp,
			"home": tea.KeyHome, "end": tea.KeyEnd}[c.key]})
		if m.cursor != c.cursor || m.focus != 1 {
			t.Fatalf("%s: the cursor on %d, the focus on %d; want %d on 1", c.key, m.cursor, m.focus, c.cursor)
		}
		if view := stripANSI(m.View().Content); !strings.Contains(view, "▸ ◻ "+path(c.cursor)) {
			t.Errorf("%s: the cursor's row, %s, is not drawn under ▸:\n%s", c.key, path(c.cursor), view)
		}
	}
}

func TestAListsWindowSaysHowManyRowsItHides(t *testing.T) {
	for _, c := range []struct {
		keys   []string
		hidden string
		first  int // the first row shown
		shown  int
	}{
		{nil, "… 31 more", 0, 9},
		{[]string{"pgdown", "pgdown"}, "… 14 above|… 18 more", 14, 8},
		{[]string{"end"}, "… 31 above", 31, 9},
	} {
		m := rows40(t, 40)
		for _, k := range c.keys {
			m, _ = drive(t, m, tea.KeyPressMsg{Code: map[string]rune{"pgdown": tea.KeyPgDown, "end": tea.KeyEnd}[k]})
		}
		lines := screen(m)
		if got := strings.Join(hidden(lines), "|"); got != c.hidden {
			t.Errorf("%v: the lines about hidden rows %q, want %q", c.keys, got, c.hidden)
		}
		var drawn []string
		for _, line := range lines {
			if f := strings.Fields(line); len(f) > 0 && (strings.HasPrefix(f[len(f)-1], "src/") ||
				strings.HasPrefix(f[len(f)-1], "docs/")) {
				drawn = append(drawn, f[len(f)-1])
			}
		}
		if len(drawn) != c.shown || drawn[0] != path(c.first) {
			t.Errorf("%v: rows %q, want %d from %s", c.keys, drawn, c.shown, path(c.first))
		}
		// the window takes ten lines whatever it shows, so the screen keeps its height and the bar its line
		if len(lines) != 40 || !strings.Contains(lines[39], "│ ⇞⇟ page · ") {
			t.Errorf("%v: %d lines, the last %q", c.keys, len(lines), lines[len(lines)-1])
		}
	}
	// a list that fits shows neither line
	t.Setenv("FAKE_KIT_ROWS", "")
	m := newForm(t, "packages", Answers{Literals: []string{"select"}, Chosen: map[int]int{0: 1}}, false)
	m, _ = drive(t, m, tea.WindowSizeMsg{Width: 80, Height: 40})
	if got := hidden(screen(m)); len(got) != 0 || m.focus != 2 {
		t.Errorf("a list of 3 rows: %q", got)
	}
}

func TestAShortScreenSaysHowManyLinesItHidesAndKeepsTheListsCursor(t *testing.T) {
	m := rows40(t, 0)
	m, _ = drive(t, m, tea.KeyPressMsg{Code: tea.KeyPgDown})
	for _, height := range []int{24, 16, 12, 9} {
		m, _ = drive(t, m, tea.WindowSizeMsg{Width: 80, Height: height})
		for _, k := range []rune{0, tea.KeyPgDown, tea.KeyEnd, tea.KeyHome} {
			if k != 0 {
				m, _ = drive(t, m, tea.KeyPressMsg{Code: k})
			}
			lines := screen(m)
			if len(lines) != height || !strings.Contains(lines[height-1], "│ ⇞⇟ page · ") ||
				strings.TrimSpace(lines[0]) != "‹ verbs" {
				t.Fatalf("at %d: %d lines, the first %q, the last %q", height, len(lines), lines[0],
					lines[len(lines)-1])
			}
			cursor := false
			for _, line := range lines {
				cursor = cursor || strings.Contains(line, "▸ ◻ "+path(m.cursor))
			}
			if !cursor {
				t.Errorf("at %d, the cursor on %d: its row is off the screen:\n%s", height, m.cursor,
					strings.Join(lines, "\n"))
			}
		}
	}
	// the counts are lines of the form: the lines shown and the lines said hidden make the form drawn whole
	m, _ = drive(t, m, tea.WindowSizeMsg{Width: 80, Height: 0}, tea.KeyPressMsg{Code: tea.KeyHome})
	whole := screen(m)
	whole = whole[:len(whole)-2] // the gap and the bar under a form drawn whole
	m, _ = drive(t, m, tea.WindowSizeMsg{Width: 80, Height: 12})
	lines := screen(m)
	lines = lines[:len(lines)-2] // the gap and the bar
	counted := 0
	for _, line := range lines {
		if marker.MatchString(line) && !strings.HasPrefix(line, "     ") {
			var n int
			fmt.Sscanf(strings.TrimSpace(line), "… %d", &n)
			counted += n
		} else {
			counted++
		}
	}
	if got := hidden(lines); counted != len(whole) || len(got) == 0 {
		t.Errorf("%d lines shown and said hidden, %d drawn whole; the lines about them %q:\n%s", counted, len(whole),
			got, strings.Join(lines, "\n"))
	}
}

// Text typed longer than the screen keeps its caret in sight: the field shows the text's end after a …, and the command
// line breaks the long word, so no line runs past the edge and none of the word is lost.
func TestTextLongerThanTheScreenKeepsItsCaretInSight(t *testing.T) {
	typed := strings.Repeat("9", 150)
	for _, width := range []int{80, 40} {
		m := newForm(t, "generate", Answers{}, false)
		m, _ = drive(t, m, tea.WindowSizeMsg{Width: width, Height: 30}, key("down"))
		m, _ = drive(t, m, testkit.Typed(typed)...)
		lines := screen(m)
		caret, command := false, ""
		for _, l := range lines {
			if ansi.StringWidth(l) > width {
				t.Errorf("at %d columns, a line of %d cells: %.60q…", width, ansi.StringWidth(l), l)
			}
			if strings.HasSuffix(l, "9▏") && strings.Contains(l, "…9") {
				caret = true
			}
			if strings.Count(l, "9") > 0 && !strings.Contains(l, "▏") {
				command += strings.TrimLeft(strings.TrimSpace(l), "kit generate-ions=")
			}
		}
		if !caret {
			t.Errorf("at %d columns, the caret is not at the text's shown end:\n%s", width, strings.Join(lines, "\n"))
		}
		if strings.Count(command, "9") != len(typed) {
			t.Errorf("at %d columns, the command line holds %d of the %d nines:\n%s", width, strings.Count(command,
				"9"), len(typed), strings.Join(lines, "\n"))
		}
	}
}

// Enter on the command line asks kit off the update loop, so a kit slow to judge it never freezes the screen; the
// answer, when it comes, runs the line. An answer about a line that changed since it was asked is dropped.
func TestEnterOnTheCommandLineAsksKitOffTheUpdateLoop(t *testing.T) {
	m := newForm(t, "commit", FromMatch(syntax.Match{Values: map[string][]string{"--message": {"hi"}}}), false)
	m.focus = len(m.fields)
	m.enter()
	t.Setenv("FAKE_KIT_HANG", "1")
	start := time.Now()
	m, ask := step(m, key("enter"))
	if took := time.Since(start); took > 300*time.Millisecond || ask == nil || m.done {
		t.Fatalf("Enter took %s, asked %v, done %v", took, ask != nil, m.done)
	}
	answer := ask()
	t.Setenv("FAKE_KIT_HANG", "")
	if done, _ := step(m, answer); !done.run {
		t.Errorf("kit's answer did not run the line")
	}
	// the person changed the line while kit judged it: the answer is about another line
	m.fields[0].Values = []string{"1"}
	if stale, _ := step(m, answer); stale.run || stale.done {
		t.Errorf("an answer about a line since changed ran it")
	}
}

// step sends one message and gives the model and its command, unrun, so a test can hold an answer back.
func step(m Model, msg tea.Msg) (Model, tea.Cmd) {
	next, cmd := m.Update(msg)
	return next.(Model), cmd
}

// A list's rows asked for before another answer changed, and arriving after, came from answers that no longer hold:
// the list asks again when the keyboard comes back to it.
func TestRowsAskedBeforeAnAnswerChangedAreAskedAgain(t *testing.T) {
	for _, late := range []bool{false, true} {
		m := newForm(t, "generate", Answers{}, false)
		m.focus = 1 // --iterations
		m.initCmd = m.enter()
		m, fetch := step(m, key("down")) // onto <path>: its rows are asked for
		if fetch == nil {
			t.Fatal("going onto <path> asked for no rows")
		}
		answer := fetch()
		if !late {
			m, _ = step(m, answer)
		}
		m, _ = step(m, key("up"))
		m, _ = step(m, tea.KeyPressMsg{Code: '3', Text: "3"})
		if late {
			m, _ = step(m, answer)
		}
		if _, again := step(m, key("down")); again == nil {
			t.Errorf("rows arriving after --iterations changed (late %v) are not asked for again", late)
		}
	}
}
