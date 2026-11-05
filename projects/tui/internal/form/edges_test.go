package form

import (
	"slices"
	"strings"
	"testing"

	"github.com/empowerite/kit/projects/tui/internal/style"
	"github.com/empowerite/kit/projects/tui/internal/syntax"
	"github.com/empowerite/kit/projects/tui/internal/testkit"
)

// anyOf opens flow's form with the picture's line and the owner's second alternative chosen, which lays out an any-of
// row of --mine and --others, and gives the form and the row's index.
func anyOf(t *testing.T) (Model, int) {
	t.Helper()
	m := newForm(t, "flow", Answers{Chosen: map[int]int{0: 0}}, false)
	m = send(m, key("down"), key("down"), key("down"), key("right"), key("right"))
	i := slices.IndexFunc(m.fields, func(f Field) bool { return f.Kind == AnyOf && slices.Contains(f.Options, "--others") })
	if i < 0 {
		t.Fatalf("no any-of row: %s", kinds(m.fields))
	}
	return m, i
}

// On an any-of row, the keys bar names the arrows across its flags and the space that turns one on; the flag under the
// cursor is drawn inverse, and a flag on is drawn chosen, live and on a plain form alike.
func TestAnAnyOfRowNamesItsKeysAndDrawsTheCursorAndWhatIsOn(t *testing.T) {
	m, i := anyOf(t)
	m = focusOn(m, i)
	if bar := m.keys().String(); !strings.Contains(bar, "←→ option · ␣ toggle") {
		t.Errorf("the any-of row's keys: %q", bar)
	}
	m = send(m, key("right"), key("space"))
	if v := m.View().Content; !strings.Contains(v, style.Cursor(style.Chosen("--others"))) {
		t.Errorf("--others, on and under the cursor, is not drawn inverse and chosen")
	}
	plain := Render(m.spec, m.Fields(), Options{Width: 80, Focus: i, Cursor: 1, Styles: Plain})
	if !strings.Contains(plain, "*--others*") {
		t.Errorf("the plain form does not mark --others on:\n%s", plain)
	}
}

// A field the verb would refuse carries kit's line under it, on the live form and on a plain one.
func TestAFieldsRefusalShowsUnderIt(t *testing.T) {
	m := newForm(t, "commit", Answers{}, true)
	fields := m.Fields()
	fields[0].Refusal = "--all takes nothing staged here"
	if got := RenderStatic(m.spec, fields, 80); !strings.Contains(got, "❌ --all takes nothing staged here") {
		t.Errorf("the plain form lacks the refusal:\n%s", got)
	}
}

// A list of one value takes the row under the cursor on Enter and moves on; a closed list says Enter takes it.
func TestAListOfOneTakesTheRowOnEnter(t *testing.T) {
	m := newForm(t, "new", FromMatch(syntax.Match{Literals: []string{"file"}}), true)
	i := slices.IndexFunc(m.fields, func(f Field) bool { return f.Key == "--rule" })
	m = focusOn(m, i)
	if v := stripANSI(m.View().Content); !strings.Contains(v, "⏎ takes it · type to filter") {
		t.Errorf("the closed list's hint:\n%s", v)
	}
	m = send(m, key("down"), key("enter"))
	if got := m.fields[i].Values; len(got) != 1 || got[0] != "b" || m.focus != i+1 {
		t.Errorf("Enter on the second row: %q, focus %d", got, m.focus)
	}
}

// A list kit refused takes what is typed: a list of one on Enter, moving on; a list of many keeps each value, with ◼,
// and says an empty line ends it. Before the rows answer, a list shows that it waits.
func TestAListKitRefusedTakesWhatIsTyped(t *testing.T) {
	t.Setenv("FAKE_KIT_REFUSE", "new")
	m := newForm(t, "new", FromMatch(syntax.Match{Literals: []string{"file"}}), true)
	i := slices.IndexFunc(m.fields, func(f Field) bool { return f.Key == "--rule" })
	m = focusOn(m, i)
	if v := stripANSI(m.View().Content); !strings.Contains(v, "⏎ takes what is typed") {
		t.Errorf("a refused list of one does not say it takes what is typed:\n%s", v)
	}
	m = send(m, testkit.Typed("markdown")...)
	m = send(m, key("enter"))
	if got := m.fields[i].Values; len(got) != 1 || got[0] != "markdown" || m.focus != i+1 {
		t.Errorf("a refused list of one took %q, focus %d", got, m.focus)
	}
	t.Setenv("FAKE_KIT_REFUSE", "commit")
	m = focusOn(newForm(t, "commit", Answers{}, true), 2)
	m = send(m, testkit.Typed("a.md")...)
	m = send(m, key("enter"))
	if v := stripANSI(m.View().Content); !strings.Contains(v, style.Picked+" a.md") ||
		!strings.Contains(v, "⏎ takes what is typed · an empty line ends the list") || m.focus != 2 {
		t.Errorf("a refused list of many:\n%s", v)
	}
	// rows not yet answered: the list waits, and the space flips nothing
	m = newForm(t, "commit", Answers{}, true)
	m.focus = 2
	m.enter()
	if v := stripANSI(m.View().Content); !strings.Contains(v, "  …\n") {
		t.Errorf("a list waiting for its rows:\n%s", v)
	}
	if _, cmd := step(m, key("space")); cmd != nil {
		t.Error("the space on a list with no rows asked for something")
	}
}

// In a list kit calls a set, the hint counts what the command line adds and what it removes: an active row turned off
// is one to remove.
func TestASetCountsWhatTheLineAddsAndRemoves(t *testing.T) {
	m := newForm(t, "packages", Answers{Literals: []string{"select"}, Chosen: map[int]int{0: 1}}, true)
	i := slices.IndexFunc(m.fields, func(f Field) bool { return f.Kind == PickMany })
	m = focusOn(m, i)
	if !m.sets[m.fields[i].Key] {
		t.Fatalf("the list is not a set")
	}
	m = send(m, key("space"))
	if v := stripANSI(m.View().Content); !strings.Contains(v, style.Remove+" 1 to remove") {
		t.Errorf("an active row turned off is not counted to remove:\n%s", v)
	}
}

// Rows for a field the form no longer has are dropped; a grammar that stops laying out keeps the fields, with why as
// the note; a focus past the fields, after they shrink, is the command line.
func TestTheFormKeepsItsFieldsWhenWhatComesNoLongerFits(t *testing.T) {
	m := newForm(t, "commit", Answers{}, true)
	before := len(m.fields)
	if m = send(m, choicesMsg{key: "<gone>"}); len(m.fields) != before {
		t.Errorf("rows for a field that is gone changed the fields")
	}
	m.spec.Error = "a test"
	m.rebuild(Collect(m.fields))
	if len(m.fields) != before || !strings.Contains(m.note, "usage does not parse") {
		t.Errorf("a grammar that does not lay out: %d fields, note %q", len(m.fields), m.note)
	}
	m = newForm(t, "commit", Answers{}, true)
	m.focus = len(m.fields) + 3
	m.rebuild(Collect(m.fields))
	if m.focus != len(m.fields) {
		t.Errorf("a focus past the fields is %d, not the command line", m.focus)
	}
	if m.filtered() != nil {
		t.Error("the command line has a list's rows")
	}
	if text, at := m.expand(0, "  "); text != "" || at != -1 {
		t.Errorf("a field without the keyboard drew %q", text)
	}
}

// A usage that does not parse lays out no field; a word with a quote in it is quoted for the shell.
func TestAUsageThatDoesNotParseHasNoFieldsAndAQuoteIsQuoted(t *testing.T) {
	if _, err := Fields(syntax.Spec{Verb: "x", Error: "a test"}, Answers{}); err == nil ||
		!strings.Contains(err.Error(), "kit x: usage does not parse: a test") {
		t.Errorf("Fields of a broken usage: %v", err)
	}
	if got := Quote([]string{"--message", "it's done", "plain"}); got != `--message 'it'\''s done' plain` {
		t.Errorf("Quote = %s", got)
	}
}

// The plain form draws a list of many's values each after ◼.
func TestThePlainFormDrawsPickedValues(t *testing.T) {
	m := newForm(t, "commit", Answers{}, true)
	fields := m.Fields()
	fields[2].Values = []string{"a.md", "b.md"}
	if got := RenderStatic(m.spec, fields, 80); !strings.Contains(got, "◼ a.md") || !strings.Contains(got, "◼ b.md") {
		t.Errorf("the picked values:\n%s", got)
	}
}
