// A plain click on a link on a step's page follows it: the lead over a step's table too.
package welcome

import (
	"errors"
	"slices"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/empowerite/kit/projects/tui/internal/shell"
	"github.com/empowerite/kit/projects/tui/internal/testkit"
)

// linked opens the screen over pages with links on them: an introduction with a link of each kind, a customize page
// whose lead has one over its table, and a terms page with two terms. Its browser is the recorder's.
func linked(t *testing.T, msgs ...tea.Msg) (Model, *[]string) {
	t.Helper()
	m, _ := fake(t)
	m.runner = repoWith(t, map[string]string{
		"1-introduction.md": "# Introduction — start\n\nSee [Flow](kittui:flow), [the issues](kittui:flow/issues), [a tutorial](kittui:tutorials/search),\n" +
			"[a term](kittui:welcome/terms/desk), [the terms](kittui:welcome/terms), [nowhere](kittui:welcome/nonesuch),\n" +
			"[here](kittui:welcome), and [the docs](https://example.com/docs).\n",
		"4-customize.md": "# Customize — parts\n\nSee [Packages](kittui:packages) over the table.\n\n## Packages\n\nThe list.\n\n`kit packages`\n",
		"5-terms.md":     "# Terms — words\n\n## Where\n\n### workspace\n\nWhat kit looks at.\n\n### desk\n\nYour root.\n",
	})
	var opened []string
	m.browse = func(url string) error { opened = append(opened, url); return nil }
	return drive(m, append(testkit.Start(m, at120), msgs...)...), &opened
}

// at gives the screen cell of the first letter of a word, the line it is on from the top, searching each line from x.
func at(t *testing.T, m Model, word string) (x, y int) {
	t.Helper()
	for y, l := range strings.Split(view(m), "\n") {
		if i := strings.Index(l, word); i >= 0 {
			return ansi.StringWidth(l[:i]), y
		}
	}
	t.Fatalf("no %q on the screen:\n%s", word, view(m))
	return 0, 0
}

// asked gives the message the command a click made produces, none for no command.
func asked(m Model, x, y int) tea.Msg {
	_, cmd := m.Update(testkit.Click(x, y))
	if cmd == nil {
		return nil
	}
	return cmd()
}

func TestAClickOnALinkOnAPageFollowsIt(t *testing.T) {
	m, opened := linked(t, esc)
	x, y := at(t, m, "Flow")
	if got := asked(m, x, y); got != (shell.SwitchMsg{Screen: shell.Flow, Crumb: true}) {
		t.Errorf("a screen: %#v", got)
	}
	x, y = at(t, m, "issues")
	if got := asked(m, x, y); got != (shell.JumpMsg{Screen: shell.Flow, To: "issues/"}) {
		t.Errorf("a tab of a screen: %#v", got)
	}
	x, y = at(t, m, "a tutorial")
	if got := asked(m, x, y); got != (shell.JumpMsg{Screen: shell.Tutorials, To: "search/"}) {
		t.Errorf("a tutorial: %#v", got)
	}
	if got := asked(m, x-1, y); got != nil {
		t.Errorf("a click beside a link: %#v", got)
	}
	// a web address opens the browser; where it cannot be asked, the link stands in place of the keys until a key
	x, y = at(t, m, "docs")
	if got := asked(m, x, y); got != (linkedMsg{url: "https://example.com/docs"}) || !slices.Equal(*opened, []string{"https://example.com/docs"}) {
		t.Errorf("a web address: %#v, opened %q", got, *opened)
	}
	m = drive(m, linkedMsg{url: "https://example.com/docs", err: shell.ErrNoDisplay})
	if !strings.Contains(view(m), "Open https://example.com/docs in a browser") {
		t.Errorf("no browser:\n%s", view(m))
	}
	if m = drive(m, down); strings.Contains(view(m), "in a browser") {
		t.Errorf("the link stands after a key:\n%s", view(m))
	}
	m = drive(m, linkedMsg{url: "https://example.com", err: errors.New("no opener")})
	if !strings.Contains(view(m), Failed+" could not open https://example.com: no opener") {
		t.Errorf("a refusal:\n%s", view(m))
	}
	if m = drive(m, linkedMsg{url: "https://example.com"}, down); strings.Contains(view(m), "could not open") {
		t.Errorf("the link opened:\n%s", view(m))
	}
}

// A link to this screen shows it at once: a tab, as the terms, and a term, which narrows the terms to it; a tab the
// screen has not, and the screen alone, change nothing. Another screen's asking, as a JumpMsg, does the same.
func TestALinkToAPlaceOnThisScreenShowsIt(t *testing.T) {
	m, _ := linked(t, esc)
	x, y := at(t, m, "a term")
	m = drive(m, testkit.Click(x+2, y))
	if id := m.tabs[m.step].id; id != "terms" || m.search().Query != "desk" || strings.Contains(view(m), "workspace  ") {
		t.Errorf("a term: step %s, search %q\n%s", id, m.search().Query, view(m))
	}
	m, _ = linked(t, esc)
	x, y = at(t, m, "the terms")
	if m = drive(m, testkit.Click(x+4, y)); m.tabs[m.step].id != "terms" || m.search().Query != "" {
		t.Errorf("a tab: step %s, search %q", m.tabs[m.step].id, m.search().Query)
	}
	m, _ = linked(t, esc)
	for _, word := range []string{"nowhere", "here"} {
		x, y = at(t, m, word)
		if m = drive(m, testkit.Click(x, y)); m.tabs[m.step].id != "introduction" {
			t.Errorf("a click on %s moved to %s", word, m.tabs[m.step].id)
		}
	}
	m = drive(m, shell.JumpMsg{Screen: shell.Welcome, To: "terms/pull%20request"}, shell.JumpMsg{Screen: shell.Welcome, To: "terms/%zz"})
	if m.tabs[m.step].id != "terms" || m.search().Query != "%zz" {
		t.Errorf("a jump: step %s, search %q", m.tabs[m.step].id, m.search().Query)
	}
}

// A link in the lead over a step's table is followed by a click as one on a page is; a click on the table is its own.
func TestALinkInTheLeadOverATableIsFollowed(t *testing.T) {
	m, _ := linked(t, esc, stepKey(1))
	x, y := at(t, m, "Packages")
	if got := asked(m, x, y); got != (shell.SwitchMsg{Screen: shell.Packages, Crumb: true}) {
		t.Errorf("a link in the lead: %#v", got)
	}
	if got := asked(m, x, y+4); got != nil {
		t.Errorf("a click below the lead: %#v", got)
	}
	if got := m.leadLinkAt(x, y); got != "packages" {
		t.Errorf("leadLinkAt %q", got)
	}
	m = drive(m, tea.WindowSizeMsg{Width: 120, Height: 8})
	if got := m.leadLinkAt(x, y); got != "" {
		t.Errorf("a lead with no room shows its link: %q", got)
	}
}
