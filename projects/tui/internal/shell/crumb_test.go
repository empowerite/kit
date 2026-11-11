package shell

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

// The crumb stands last on the title line, after the workspace menu and a maximized tab, a dot between, as ↩ and the
// name of the place it goes back to; a click on it is found where it is drawn, as much of it as shows, and what the
// view stands on is found where it was.
func TestTheCrumbStandsLastOnTheTitleLine(t *testing.T) {
	b := MenuBar{Screen: "kit repositories", Workspace: "- o/r", On: "➋ remotes 1 :", Crumb: "changes",
		Search: SearchLine("", false)}
	line := ansi.Strip(b.Line(90))
	if !strings.HasPrefix(line, "  "+App+" kit repositories · - o/r · ➋ remotes 1 : · "+Back+" changes  ") {
		t.Errorf("the title at 90: %q", line)
	}
	from, until, whole := b.CrumbSpan(90)
	if !whole || ansi.Strip(ansi.Cut(b.Line(90), from, until)) != Back+" changes" {
		t.Errorf("the crumb spans %d to %d, whole %v", from, until, whole)
	}
	if !b.OnCrumb(from, Top, 90) || !b.OnCrumb(until-1, Top, 90) || b.OnCrumb(until, Top, 90) ||
		b.OnCrumb(from-1, Top, 90) || b.OnCrumb(from, Top+1, 90) {
		t.Error("a click on the crumb is not found where it is drawn")
	}
	if on, _, _ := b.OnSpan(90); ansi.Strip(ansi.Cut(b.Line(90), on, on+1)) != "➋" {
		t.Errorf("what the view stands on moved, to %d", on)
	}
	// not maximized, it follows the workspace menu
	b.On = ""
	if line = ansi.Strip(b.Line(90)); !strings.HasPrefix(line, "  "+App+" kit repositories · - o/r · "+Back+" changes") {
		t.Errorf("the title, not maximized: %q", line)
	}
	// cut by the search field, as much of it as shows is clicked; none of it, nothing is
	if _, _, whole = b.CrumbSpan(40); whole {
		t.Error("at 40 the crumb is cut, not whole")
	}
	if from, _, _ = b.CrumbSpan(20); from != -1 || b.OnCrumb(0, Top, 20) {
		t.Errorf("at 20 the crumb does not show, yet starts at %d", from)
	}
	if b.Crumb = ""; ansi.Strip(b.Line(90)) == line || strings.Contains(ansi.Strip(b.Line(90)), Back) {
		t.Errorf("with no crumb: %q", ansi.Strip(b.Line(90)))
	}
	if from, _, _ = b.CrumbSpan(90); from != -1 {
		t.Error("no crumb has a span")
	}
}

// ⌫ goes back while a crumb shows, and the keys bar names it after the screen's own keys; with none, it names nothing.
func TestTheKeysBarNamesTheWayBackWhileACrumbShows(t *testing.T) {
	if got := CrumbOf("changes", Bind("?", "legend")); len(got) != 2 || got[1].Key != "⌫" ||
		got[1].Presses[0] != BackPress {
		t.Errorf("with a crumb: %+v", got)
	}
	if got := CrumbOf("", Bind("?", "legend")); len(got) != 1 {
		t.Errorf("with none: %+v", got)
	}
}

// A screen with no tab bar under its menu bar draws the rule there across its width, one cell a glyph.
func TestTheRuleUnderTheMenuBarSpansTheScreen(t *testing.T) {
	if ansi.StringWidth(Rule) != 1 {
		t.Fatalf("the rule's glyph is not one cell: %q", Rule)
	}
	for _, width := range []int{0, 1, 80} {
		line := RuleLine(width, "")
		if got := ansi.Strip(line); got != strings.Repeat(Rule, width) {
			t.Errorf("at %d: %q", width, got)
		}
	}
}
