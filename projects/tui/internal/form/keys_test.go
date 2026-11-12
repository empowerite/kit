package form

import (
	"slices"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/empowerite/kit/projects/tui/internal/kit"
	"github.com/empowerite/kit/projects/tui/internal/syntax"
	"github.com/empowerite/kit/projects/tui/internal/testkit"
)

// press gives the key press of a key code, with its modifiers.
func press(code rune, mod tea.KeyMod) tea.KeyPressMsg { return tea.KeyPressMsg{Code: code, Mod: mod} }

var (
	tab      = press(tea.KeyTab, 0)
	shiftTab = press(tea.KeyTab, tea.ModShift)
	pgdown   = press(tea.KeyPgDown, 0)
	pgup     = press(tea.KeyPgUp, 0)
	home     = press(tea.KeyHome, 0)
	end      = press(tea.KeyEnd, 0)
)

// send drives the form with the messages, without running Init again.
func send(m Model, msgs ...tea.Msg) Model {
	m, _ = testkit.Drive(m, nil, msgs...)
	return m
}

// focusOn puts the keyboard on the field, as moving there would, and brings its list's rows.
func focusOn(m Model, i int) Model {
	m.focus = i
	if cmd := m.enter(); cmd != nil {
		m = send(m, cmd())
	}
	return m
}

// A flag on its own flips on the space, and the arrows, which fold and unfold, leave it; Enter, down, and tab move on;
// up and shift-tab move back, never past the first field.
func TestAToggleFlipsOnTheSpace(t *testing.T) {
	m := focusOn(newForm(t, "commit", Answers{}, true), 0)
	on := func(m Model) bool { return slices.Contains(m.CommandLine(), "--all") }
	if m = send(m, key("space")); !on(m) {
		t.Fatalf("space did not turn --all on: %q", m.CommandLine())
	}
	if m = send(m, key("left"), key("right")); !on(m) {
		t.Errorf("an arrow turned --all off: %q", m.CommandLine())
	}
	if m = send(m, key("space")); on(m) {
		t.Errorf("space did not turn --all off: %q", m.CommandLine())
	}
	m = send(m, key("space"))
	for _, msg := range []tea.Msg{key("enter"), key("down"), tab} {
		m.focus = 0
		if m = send(m, msg); m.focus != 1 {
			t.Errorf("%s moved to %d, not 1", msg, m.focus)
		}
	}
	for _, msg := range []tea.Msg{key("up"), shiftTab} {
		m.focus = 0
		if m = send(m, msg); m.focus != 0 {
			t.Errorf("%s from the first field moved to %d", msg, m.focus)
		}
	}
	// a flag the verb requires stays on whatever is pressed
	m.fields[0].Required, m.fields[0].Values, m.focus = true, []string{"1"}, 0
	if m = send(m, key("space"), key("left")); !on(m) {
		t.Errorf("a required flag turned off: %q", m.CommandLine())
	}
}

// A text field is what is typed: backspace erases its last rune, the value shows as it is typed, and coming back to
// the field brings the text back to edit; the arrows and tab move between fields from it.
func TestATextFieldIsWhatIsTypedAndBackspaceErasesIt(t *testing.T) {
	m := newForm(t, "generate", Answers{}, true)
	m = send(m, key("down"))
	m = send(m, testkit.Typed("12")...)
	m = send(m, key("backspace"))
	if line := strings.Join(m.CommandLine(), " "); line != "--iterations=1" {
		t.Fatalf("after 12 and a backspace: %q", line)
	}
	m = send(m, key("enter"), key("up"))
	if m.focus != 1 || m.input != "1" {
		t.Errorf("back on --iterations: focus %d, input %q", m.focus, m.input)
	}
	m = send(m, testkit.Typed("x")...)
	m = send(m, key("backspace"), key("backspace"), key("backspace"))
	if m.input != "" || slices.Contains(m.CommandLine(), "--iterations=") || len(m.fields[1].Values) != 0 {
		t.Errorf("backspace past the text: input %q, line %q", m.input, m.CommandLine())
	}
	for _, c := range []struct {
		msg  tea.Msg
		want int
	}{{shiftTab, 0}, {tab, 2}, {key("up"), 0}} {
		m.focus = 1
		m.enter()
		if m = send(m, c.msg); m.focus != c.want {
			t.Errorf("%s from --iterations moved to %d, not %d", c.msg, m.focus, c.want)
		}
	}
}

// A field of lines takes a line on Enter, another on the next, and moves on at an empty one; backspace with nothing
// typed takes the last line back; leaving the field takes the line being typed.
func TestAFieldOfLinesTakesALineOnEnterAndBackspaceTakesItBack(t *testing.T) {
	m := newForm(t, "template", Answers{}, true)
	if m.fields[0].Kind != Lines {
		t.Fatalf("template's first field is %d, not lines", m.fields[0].Kind)
	}
	m = send(m, testkit.Typed("a.md")...)
	m = send(m, key("enter"))
	m = send(m, testkit.Typed("b.md")...)
	m = send(m, key("enter"))
	if got := m.fields[0].Values; !slices.Equal(got, []string{"a.md", "b.md"}) || m.focus != 0 {
		t.Fatalf("two lines: %q, focus %d", got, m.focus)
	}
	m = send(m, key("backspace"))
	if got := m.fields[0].Values; !slices.Equal(got, []string{"a.md"}) {
		t.Errorf("backspace with nothing typed: %q", got)
	}
	m = send(m, testkit.Typed("c.md")...)
	m = send(m, key("down"))
	if got := m.fields[0].Values; !slices.Equal(got, []string{"a.md", "c.md"}) || m.focus != len(m.fields) {
		t.Errorf("leaving with a line typed: %q, focus %d", got, m.focus)
	}
	m = send(m, key("up"), key("enter"))
	if m.focus != len(m.fields) {
		t.Errorf("Enter at an empty line did not move on: focus %d", m.focus)
	}
	if !strings.Contains(stripANSI(send(m, key("up")).View().Content), "an empty line ends the list") {
		t.Errorf("the field of lines does not say how it ends")
	}
}

// A list moves its cursor a page at a time and to either end, leaves for the field before from its first row and
// for the next from its last, and with tab and shift-tab; typing filters it, and backspace widens it again.
func TestAListMovesByPagesAndEndsAndLeavesAtItsEdges(t *testing.T) {
	t.Setenv("FAKE_KIT_ROWS", "30")
	m := focusOn(newForm(t, "commit", Answers{}, true), 2)
	if rows := len(m.filtered()); rows != 30 {
		t.Fatalf("%d rows, not 30", rows)
	}
	for _, c := range []struct {
		msg  tea.Msg
		want int
	}{{pgdown, listWindow - 1}, {pgdown, 2 * (listWindow - 1)}, {pgup, listWindow - 1}, {end, 29}, {home, 0}} {
		if m = send(m, c.msg); m.cursor != c.want {
			t.Errorf("%s: cursor %d, not %d", c.msg, m.cursor, c.want)
		}
	}
	if m = send(m, key("up")); m.focus != 1 {
		t.Errorf("up from the first row moved to %d, not the field before", m.focus)
	}
	m = send(focusOn(m, 2), end, key("down"))
	if m.focus != len(m.fields) {
		t.Errorf("down from the last row moved to %d, not the command line", m.focus)
	}
	for _, c := range []struct {
		msg  tea.Msg
		want int
	}{{tab, len(m.fields)}, {shiftTab, 1}} {
		m = send(focusOn(m, 2), c.msg)
		if m.focus != c.want {
			t.Errorf("%s from the list moved to %d, not %d", c.msg, m.focus, c.want)
		}
	}
	m = focusOn(m, 2)
	m = send(m, testkit.Typed("FILE-2")...)
	if rows := m.filtered(); len(rows) != 11 || rows[0].Value != "src/file-2.go" {
		t.Errorf("filtered by FILE-2, ignoring case: %d rows, %v", len(rows), rows)
	}
	m = send(m, testkit.Typed("x")...)
	if v := stripANSI(m.View().Content); len(m.filtered()) != 0 || !strings.Contains(v, "nothing matches") {
		t.Errorf("a filter that matches nothing:\n%s", v)
	}
	m = send(m, key("backspace"), key("backspace"))
	if m.input != "FILE-" || len(m.filtered()) != 29 {
		t.Errorf("backspace widened the filter to %q, %d rows", m.input, len(m.filtered()))
	}
}

// An open list takes what is typed on ctrl-n, a value it does not offer; a list of many keeps the keyboard for more.
func TestAnOpenListTakesWhatIsTypedOnCtrlN(t *testing.T) {
	m := focusOn(newForm(t, "commit", Answers{}, true), 2)
	m = send(m, testkit.Typed("notes/new.md")...)
	m = send(m, key("ctrl+n"))
	if got := m.fields[2].Values; !slices.Contains(got, "notes/new.md") || m.focus != 2 || m.input != "" {
		t.Errorf("ctrl-n: values %q, focus %d, input %q", got, m.focus, m.input)
	}
	// with nothing typed, ctrl-n takes nothing
	if m = send(m, key("ctrl+n")); len(m.fields[2].Values) != 1 {
		t.Errorf("ctrl-n with nothing typed took %q", m.fields[2].Values)
	}
	// an open list of one value says that Enter takes the row and ctrl-n what is typed
	spec := syntax.Spec{Verb: "x", Syntax: [][]syntax.Node{{{Kind: "operand", Name: "name"}}},
		Arguments: map[string]syntax.Argument{"<name>": {Choices: "open"}}}
	fields, _ := Fields(spec, Answers{})
	m = send(New(spec, fields, testkit.Runner(t, "kit"), true), choicesMsg{key: "<name>", rows: []kit.Choice{{Value: "a"}}})
	if v := stripANSI(m.View().Content); !strings.Contains(v, "⏎ takes it · ctrl-n takes what is typed · type to filter") {
		t.Errorf("an open list of one's hint:\n%s", v)
	}
}

// On the command line, Enter asks kit to judge it: a line short of a part says what it needs, after the words chosen,
// and takes the keyboard to the first field that needs an answer; a line kit refuses shows kit's line; a kit that
// cannot judge it shows why. Up leaves the command line, and down stays.
func TestTheCommandLineShowsWhatKitSaysOfIt(t *testing.T) {
	t.Setenv("FAKE_KIT_MATCH_MISSING", "template")
	t.Setenv("FAKE_KIT_MATCH_LITERAL", "new")
	m := newForm(t, "template", Answers{}, true)
	m.focus = len(m.fields)
	m = send(m, key("enter"))
	if !strings.Contains(m.refusal, "kit template new needs <path>; run: kit help template") || m.focus != 0 {
		t.Errorf("a missing part: %q, focus %d", m.refusal, m.focus)
	}
	t.Setenv("FAKE_KIT_MATCH_LITERAL", "")
	m = newForm(t, "generate", Answers{}, true)
	t.Setenv("FAKE_KIT_MATCH_MISSING", "generate")
	m.focus = len(m.fields)
	if m = send(m, key("enter")); m.refusal != "kit generate needs <path>; run: kit help generate" ||
		m.focus != len(m.fields) {
		t.Errorf("a missing part with no field required: %q, focus %d", m.refusal, m.focus)
	}
	t.Setenv("FAKE_KIT_MATCH_REFUSE", "generate")
	if m = send(m, key("enter")); !strings.Contains(m.refusal, "refuses these words for the test") {
		t.Errorf("a refusal: %q", m.refusal)
	}
	m.runner = kit.Runner{Kit: "/nonexistent/kit", Dir: t.TempDir()}
	if m = send(m, key("enter")); !strings.Contains(m.refusal, "/nonexistent/kit") {
		t.Errorf("a kit that cannot judge: %q", m.refusal)
	}
	if m = send(m, key("down")); m.focus != len(m.fields) {
		t.Errorf("down on the command line moved to %d", m.focus)
	}
	if m = send(m, key("up")); m.focus != len(m.fields)-1 {
		t.Errorf("up on the command line moved to %d", m.focus)
	}
}

// Fields gives the fields as the form lays them out now, the extension in place.
func TestFieldsAreTheFormsFieldsAsTheyStand(t *testing.T) {
	m := newForm(t, "commit", FromMatch(syntax.Match{Values: map[string][]string{"--message": {"hi"}}}), true)
	got := m.Fields()
	if len(got) != len(m.fields) || got[1].Key != m.fields[1].Key {
		t.Errorf("Fields = %+v", got)
	}
}

// A row's state shows as kit packages draws it, and a set's rows show what the command line makes of them.
func TestARowShowsItsStateAndASetsRowsShowThePlan(t *testing.T) {
	for _, c := range []struct {
		state      string
		picked     bool
		now, after string
	}{
		{"active", true, "✅", "✅"},
		{"active", false, "✅", "➖"},
		{"", true, "  ", "➕"},
		{"excluded", false, "❌", "❌"},
		{"", false, "  ", "  "},
	} {
		row := kit.Choice{Value: "x", State: c.state}
		if got := state(row); got != c.now {
			t.Errorf("state(%q) = %q, not %q", c.state, got, c.now)
		}
		if got := plan(row, c.picked); got != c.after {
			t.Errorf("plan(%q, picked %v) = %q, not %q", c.state, c.picked, got, c.after)
		}
	}
}
