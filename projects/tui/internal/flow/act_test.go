// The backlog's actions run kit's verbs on the issues selected there: at once, or after a dialog asks for an answer
// or a yes; the keys bar says what kit did, or why it refused.
package flow

import (
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/empowerite/kit/projects/tui/internal/shell"
	"github.com/empowerite/kit/projects/tui/internal/style"
	"github.com/empowerite/kit/projects/tui/internal/testkit"
)

// item gives the line of the tab's menu named, or -1.
func item(m Model, name string) int {
	return slices.IndexFunc(m.tabItems(), func(i shell.Item) bool { return i.Name == name })
}

// choose opens the tab's menu and chooses the item named, as enter on it does.
func choose(t *testing.T, m Model, name string) Model {
	t.Helper()
	m = drive(m, press(':'))
	k := item(m, name)
	if k < 0 || !m.enabled(k) {
		t.Fatalf("%s is not in the menu, or dim: %d", name, k)
	}
	m.tabMenu.At = k
	return drive(m, press(tea.KeyEnter))
}

// acted gives the calls of the verbs that act, as fake-kit recorded them, the stages' loads left out.
func acted(t *testing.T, record string) []string {
	t.Helper()
	var out []string
	for _, c := range calls(t, record) {
		if !strings.HasPrefix(c, "flow ") {
			out = append(out, c)
		}
	}
	return out
}

// selecting gives the picture loaded with backlog's first two issues selected, recording kit's calls.
func selecting(t *testing.T) (Model, string) {
	t.Helper()
	record := filepath.Join(t.TempDir(), "calls")
	t.Setenv("FAKE_FLOW_CALLS", record)
	m := loaded(t, options(), tea.WindowSizeMsg{Width: 120, Height: 40}, down)
	m.stages[0].picked["#45"], m.stages[0].picked["#46"] = true, true
	if got := m.refs(); !slices.Equal(got, []string{"#45", "#46"}) {
		t.Fatalf("selected: %q", got)
	}
	return m, record
}

func TestTheBacklogsActionsAreInItsMenuDimUntilIssuesAreSelected(t *testing.T) {
	m := loaded(t, options(), down)
	for _, name := range []string{"Start Local Work for Each", "Assign to Me", "Close as Not Planned", "Copy Links"} {
		if k := item(m, name); k < 0 || m.enabled(k) {
			t.Errorf("%s with nothing selected: line %d, not dim", name, k)
		}
	}
	// New… files an issue, and needs none selected
	if k := item(m, "New…"); k < 0 || !m.enabled(k) {
		t.Errorf("New… with nothing selected: line %d, dim", k)
	}
	m, _ = selecting(t)
	names := []string{"New…", "Start Local Work for Each", "Start Local Work Together…", "Add to Pull Request…", "Assign to Me",
		"Assign to…", "Unassign", "Gather Under a Parent…", "Close as Not Planned", "Close as Duplicate of…",
		"Copy Links", "Open Selected"}
	if got := m.tabItems()[len(m.tabItems())-len(names):]; !slices.EqualFunc(got, names,
		func(i shell.Item, n string) bool { return i.Name == n }) || m.tabItems()[len(m.tabItems())-len(names)-1].Name != "" {
		t.Errorf("the last section: %+v", got)
	}
	// another tab offers none of them, even with rows selected there
	m = drive(m, esc, right, down, press(tea.KeySpace))
	if item(m, "Assign to Me") >= 0 {
		t.Error("underway offers the backlog's actions")
	}
}

func TestAssignToMeRunsKitAndTheKeysBarSaysWhatItDid(t *testing.T) {
	m, record := selecting(t)
	m = choose(t, m, "Assign to Me")
	if got := acted(t, record); !slices.Equal(got, []string{"assign #45 #46"}) {
		t.Fatalf("the calls: %q", got)
	}
	last := lines(m)[len(lines(m))-1]
	if !strings.Contains(last, "✅ kit assign #45 #46") || m.selected() != 0 || strings.Contains(m.View().Content,
		style.Refusal("✅")) {
		t.Errorf("after assign: %q, %d selected", last, m.selected())
	}
	// what reads the forge's issues read them again
	if got := calls(t, record); !slices.ContainsFunc(got, func(c string) bool {
		return strings.HasPrefix(c, "flow backlog --json") && strings.Contains(c, "--fresh")
	}) {
		t.Errorf("backlog was not read afresh: %q", got)
	}
}

func TestStartForEachRunsKitWorkOncePerIssue(t *testing.T) {
	m, record := selecting(t)
	choose(t, m, "Start Local Work for Each")
	if got := acted(t, record); !slices.Equal(got, []string{"work #45", "work #46"}) {
		t.Errorf("the calls: %q", got)
	}
}

func TestAClosingActionListsTheIssuesAndRunsOnlyOnYes(t *testing.T) {
	m, record := selecting(t)
	m = choose(t, m, "Close as Not Planned")
	v := view(m)
	if m.ask == nil || !strings.Contains(v, "Close these issues as not planned?") || !strings.Contains(v, "#45 ") ||
		!strings.Contains(v, "#46 ") || !strings.Contains(lines(m)[len(lines(m))-1], "esc no ⏎ yes") {
		t.Fatalf("the dialog:\n%s", v)
	}
	// a click does nothing while it asks; esc runs nothing
	if m = drive(m, testkit.Click(10, 10), esc); m.ask != nil || len(acted(t, record)) != 0 || m.selected() != 2 {
		t.Fatalf("esc: ask %v, calls %q", m.ask, acted(t, record))
	}
	m = choose(t, m, "Close as Not Planned")
	m = drive(m, press(tea.KeyEnter))
	if got := acted(t, record); !slices.Equal(got, []string{"close --not-planned #45 #46"}) {
		t.Errorf("the calls: %q", got)
	}
	if m = drive(m, typed("x")...); strings.Contains(lines(m)[len(lines(m))-1], "kit close") {
		t.Error("a key did not take the notice away")
	}
}

func TestAnActionThatAsksTakesTheTypedAnswer(t *testing.T) {
	for _, c := range []struct{ name, answer, want string }{
		{"Start Local Work Together…", "The plan", "work --title The plan #45 #46"},
		{"Gather Under a Parent…", "The plan", "gather --title The plan #45 #46"},
		{"Assign to…", "bob", "assign --to bob #45 #46"},
		{"Close as Duplicate of…", "#7", "close --duplicate-of #7 #45 #46"},
	} {
		m, record := selecting(t)
		m = choose(t, m, c.name)
		if m.ask == nil || !m.ask.typing {
			t.Fatalf("%s does not ask", c.name)
		}
		// enter with nothing typed waits
		if m = drive(m, press(tea.KeyEnter)); m.ask == nil {
			t.Fatalf("%s ran with nothing typed", c.name)
		}
		m = drive(m, typed(c.answer+"x")...)
		m = drive(m, press(tea.KeyBackspace))
		if !strings.Contains(view(m), c.answer+"▏") {
			t.Errorf("%s: the typed line:\n%s", c.name, view(m))
		}
		m = drive(m, press(tea.KeyEnter))
		if got := acted(t, record); !slices.Equal(got, []string{c.want}) {
			t.Errorf("%s: the calls %q", c.name, got)
		}
	}
}

func TestAddToPullRequestPicksOneOfTheCallersOwn(t *testing.T) {
	m, record := selecting(t)
	m.me = "ada"
	numbers, _ := m.mine()
	if len(numbers) == 0 {
		t.Fatal("the fixture has no pull request of ada's")
	}
	m = choose(t, m, "Add to Pull Request…")
	if m.ask == nil || m.ask.pick == nil || !strings.Contains(view(m), "Add to which of your pull requests?") {
		t.Fatalf("the dialog:\n%s", view(m))
	}
	m = drive(m, up, down, up, press(tea.KeyEnter))
	if got := acted(t, record); !slices.Equal(got, []string{"work --pr " + strconv.Itoa(numbers[0]) + " #45 #46"}) {
		t.Errorf("the calls: %q", got)
	}
	// someone with no open pull request has nothing to add to: the item is dim
	m.me = "nobody"
	m.stages[0].picked["#7"] = true
	m = drive(m, press(':'))
	if k := item(m, "Add to Pull Request…"); k < 0 || m.enabled(k) {
		t.Errorf("add to pull request for a caller with none: %d", k)
	}
}

func TestARefusalStaysInTheKeysBarAndKeepsTheSelection(t *testing.T) {
	m, _ := selecting(t)
	t.Setenv("FAKE_FLOW_REFUSE", "unassign")
	m = choose(t, m, "Unassign")
	m = drive(m, press(tea.KeyEnter))
	last := lines(m)[len(lines(m))-1]
	if !strings.Contains(last, Failed+" kit unassign refuses for the test") || m.noted || m.selected() != 2 {
		t.Errorf("the refusal: %q, %d selected", lines(m)[len(lines(m))-1], m.selected())
	}
}

func TestCopyLinksCopiesEachIssueAsMarkdown(t *testing.T) {
	m, _ := selecting(t)
	if got := m.links(); !strings.HasPrefix(got, "- [#45 ") || !strings.Contains(got, "](https://") ||
		strings.Count(got, "\n") != 1 {
		t.Errorf("the links: %q", got)
	}
	m = drive(m, press(':'))
	m.tabMenu.At = item(m, "Copy Links")
	next, cmd := m.Update(press(tea.KeyEnter))
	if m = next.(Model); cmd == nil || !strings.Contains(ansi.Strip(lines(m)[len(lines(m))-1]), "copied 2 links") {
		t.Errorf("copy links: %q", lines(m)[len(lines(m))-1])
	}
}

// A menu taller than the screen's room scrolls with the keyboard: it ends above the keys bar, marks the items above
// and below it, and a click picks the item it shows there.
func TestATallMenuScrollsWithTheKeyboardAndAClickPicksWhatItShows(t *testing.T) {
	m := loaded(t, options(), tea.WindowSizeMsg{Width: 120, Height: 16}, down)
	m.stages[0].picked["#45"] = true
	m = drive(m, press(':'))
	box, _, first := m.menuBoxIn()
	if len(box) != 16-1-m.menuTop() || first != 0 || !strings.Contains(ansi.Strip(box[len(box)-1]), "▾") ||
		strings.Contains(ansi.Strip(box[0]), "▴") {
		t.Fatalf("the menu at the top: %d lines, first %d\n%s", len(box), first, view(m))
	}
	m.tabMenu.At = item(m, "Open Selected")
	box, left, first := m.menuBoxIn()
	if first == 0 || !strings.Contains(ansi.Strip(box[0]), "▴") || strings.Contains(ansi.Strip(box[len(box)-1]), "▾") {
		t.Errorf("the menu at the end: first %d\n%s", first, view(m))
	}
	// a click on the line that shows Assign to Me picks it
	k := item(m, "Assign to Me")
	record := filepath.Join(t.TempDir(), "calls")
	t.Setenv("FAKE_FLOW_CALLS", record)
	if m = drive(m, testkit.Click(left+3, m.menuTop()+1+k-first)); !slices.Equal(acted(t, record), []string{"assign #45"}) {
		t.Errorf("a click on Assign to Me: %q", acted(t, record))
	}
	// a click on the box's bottom edge picks nothing
	m = drive(m, press(':'))
	box, left, _ = m.menuBoxIn()
	if m = drive(m, testkit.Click(left+3, m.menuTop()+len(box)-1)); m.tabMenu.Open || len(acted(t, record)) != 1 {
		t.Errorf("a click on the edge: menu %v, calls %q", m.tabMenu.Open, acted(t, record))
	}
}

// New… asks kit for the trackers, and offers those that give a page, each its prefix and where, the forge's own under
// the keyboard; enter opens the page picked. A tracker with no page is left
// out, and kit's refusal to list them is in place of the keys.
func TestNewOffersTheTrackersAndOpensThePagePicked(t *testing.T) {
	m := loaded(t, options(), down)
	var opened []string
	m.browse = func(url string) error { opened = append(opened, url); return nil }
	m = choose(t, m, "New…")
	if m.ask == nil || m.ask.title != "New issue in which tracker?" || m.ask.at != 0 ||
		!slices.Equal(m.ask.lines, []string{"#  github empowerite/kit", "AB#  ado Claims Team"}) {
		t.Fatalf("the dialog: %+v", m.ask)
	}
	if last := lines(m)[len(lines(m))-1]; !strings.Contains(last, "⏎ open") || !strings.Contains(raw(m), "AB#  ado Claims Team") {
		t.Errorf("the dialog's keys: %q\n%s", last, raw(m))
	}
	if m = drive(m, down, press(tea.KeyEnter)); m.ask != nil ||
		!slices.Equal(opened, []string{"https://dev.azure.com/example/Claims%20Team/_workitems/create/Task"}) {
		t.Errorf("enter on AB#: opened %q, dialog %+v", opened, m.ask)
	}
	// no tracker with a page, or kit's refusal, is said in place of the keys
	trackers, err := m.runner.Trackers(m.ctx)
	if err != nil || len(trackers) != 3 {
		t.Fatalf("the trackers: %+v, %v", trackers, err)
	}
	if m.newIssue(trackersMsg{trackers: trackers[2:]}); m.ask != nil || !strings.Contains(m.notice, "the BZ# tracker has no new page") {
		t.Errorf("no tracker with a page: %q", m.notice)
	}
	t.Setenv("FAKE_FLOW_REFUSE", "new")
	if m = choose(t, m, "New…"); m.ask != nil || !strings.Contains(lines(m)[len(lines(m))-1], "the forge cannot be read") {
		t.Errorf("a refusal: %+v, %q", m.ask, lines(m)[len(lines(m))-1])
	}
}
