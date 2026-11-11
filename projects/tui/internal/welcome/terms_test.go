// The terms step as a table: the page read as terms, what each is run with, the table drawn under the lead, and its
// keys, clicks, search, presets, and settings, as the screen hands them to it; a page without terms stays a page.
package welcome

import (
	"reflect"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/empowerite/kit/projects/tui/internal/shell"
	"github.com/empowerite/kit/projects/tui/internal/table"
	"github.com/empowerite/kit/projects/tui/internal/testkit"
)

func TestAPageIsReadAsItsTerms(t *testing.T) {
	md := "<!-- kit:generated -- the terms -->\r\n# Terms\r\n\r\nThe words.\r\n\r\n## desk\r\n\r\nYour root " +
		"checkout, as [the guide](../1st.md) and [GitHub](https://github.com) say. It is **yours**.\n\nIts second " +
		"paragraph names `kit work`.\n\n`kit flow desk`\n\n`kit flow worktrees`\n\n## `bare` term, plain, ##\n\nNo command " +
		"here.\n\n```sh\necho hi\n```\n\n### a heading under it\n\n<!--\n## not a term\n-->\n## kind\n\n`kit kinds`\n"
	lead, terms := parseTerms(md)
	if lead != "# Terms\n\nThe words." {
		t.Errorf("the lead: %q", lead)
	}
	if len(terms) != 3 {
		t.Fatalf("the terms: %+v", terms)
	}
	desk := term{names: []string{"desk"}, examples: []string{"kit flow desk", "kit flow worktrees"},
		meaning: "Your root checkout, as the guide and GitHub (https://github.com) say. It is yours. Its second " +
			"paragraph names kit work."}
	if !reflect.DeepEqual(terms[0], desk) {
		t.Errorf("desk:\n%+v\n%+v", terms[0], desk)
	}
	if !reflect.DeepEqual(terms[1], term{names: []string{"bare term", "plain"},
		meaning: "No command here. echo hi ### a heading under it"}) {
		t.Errorf("an entry of two terms without an example: %+v", terms[1])
	}
	if !reflect.DeepEqual(terms[2], term{names: []string{"kind"}, examples: []string{"kit kinds"}}) {
		t.Errorf("an entry without a meaning: %+v", terms[2])
	}
	if lead, terms := parseTerms("# Ours\n\n- **deploy**: a release.\n"); terms != nil || lead == "" {
		t.Errorf("a page without terms: %q %+v", lead, terms)
	}
	for command, want := range map[string]string{"kit flow --stages": "kit flow", "git remote --verbose": "git remote",
		"kit gate": "kit gate", "ls": "ls", "kit --version": "kit", "": ""} {
		if got := verbOf(command); got != want {
			t.Errorf("verbOf(%q) = %q, want %q", command, got, want)
		}
	}
	rs := termRecords(terms)
	if rs[0].Key != "0 desk" || rs[0].Values["command"].Raw[0] != "kit flow" || rs[1].Key != "1 bare term, plain" ||
		rs[1].Values["command"].Raw != nil || rs[1].Values["examples"].Raw != nil ||
		!reflect.DeepEqual(rs[1].Values["terms"].Raw, []string{"bare term", "plain"}) ||
		!reflect.DeepEqual(rs[0].Values["examples"].Raw, desk.examples) || rs[2].Values["command"].Raw[0] != "kit kinds" {
		t.Errorf("the records: %+v", rs)
	}
}

func TestTheTermsAreATableOfCommandsTermsMeaningsAndExamples(t *testing.T) {
	m, _ := loaded(t, press('5'))
	if !m.gridded() || keysAt(m) != "= menu - workspace / search │ ⇥·⇧⇥ tab ↓ terms 1-6 jump" {
		t.Fatalf("the terms step: gridded %v, keys %q", m.gridded(), keysAt(m))
	}
	got := view(m)
	// the command, the terms, each a pill and two a count, the meaning, and the examples, a second one counted
	for _, want := range []string{"   Terms\n", "The words kit uses", "▾ 12 ", "Default ▼", "command       terms",
		"meaning", "examples", "▸ git remote    2   ", "▸ kit flow      " + pill("desk"),
		"kit flow worktrees        +1"} {
		if !strings.Contains(got, want) {
			t.Errorf("no %q in:\n%s", want, got)
		}
	}
	// down gives the table the keyboard, on the count over the rows, and down again the rows
	if m = drive(m, down); !m.inPane || !strings.Contains(keysAt(m), "↓ rows") {
		t.Fatalf("down: in the pane %v, keys %q", m.inPane, keysAt(m))
	}
	if m = drive(m, down); !termsSheet(m).InRows() || !strings.Contains(keysAt(m), "␣·⇧↑↓ select") {
		t.Errorf("down into the rows: keys %q", keysAt(m))
	}
	// a key the table does not take, as a step's number, is the screen's still
	if m = drive(m, tea.KeyPressMsg{Code: '4', Text: "4"}); m.step != land || m.inPane || termsSheet(m).Focused() {
		t.Errorf("4 from the terms' rows: step %d, in the pane %v", m.step, m.inPane)
	}
	// esc in the rows gives the bar the keyboard
	if m = drive(m, press('5'), enter, down, down, esc); m.inPane || termsSheet(m).Focused() {
		t.Errorf("esc in the rows: in the pane %v", m.inPane)
	}
	// tab from the bar moves to the next step, and the table holds no keyboard there
	if m = drive(m, tabKey); m.step != yours || termsSheet(m).Focused() {
		t.Errorf("tab: step %d", m.step)
	}
	if m = drive(m, backTab, up); m.inPane {
		t.Error("up on the bar")
	}
}

func TestByCommandStandsOncePerTermGroupedByItsCommand(t *testing.T) {
	m, _ := loaded(t, press('5'))
	m = drive(m, clickOn(t, m, "Default ▼"))
	m = drive(m, clickOn(t, m, "By command"))
	got := view(m)
	for _, want := range []string{"▾ command ▾ 12", "By command ▼", "    term    ", "▾ git remote 1",
		"▸ " + pill("primary") + "     A remote is", "▸ " + pill("remote") + "      A remote is", "▾ kit flow 4",
		"▸ " + pill("code forge") + "  The service"} {
		if !strings.Contains(got, want) {
			t.Errorf("no %q in:\n%s", want, got)
		}
	}
	if strings.Contains(got, "▸ kit flow ") {
		t.Errorf("the command stands as a group, not a column:\n%s", got)
	}
}

// termsSheet gives the terms step's table.
func termsSheet(m Model) *table.Sheet { return m.tabs[indexOf(m.tabs, "terms")].sheet }

// pill gives a term as the test's view draws its pill: the text between the half circles, their colors stripped.
func pill(text string) string { return "\ue0b6" + text + "\ue0b4" }

// clickOn gives a click on the first place the screen draws text.
func clickOn(t *testing.T, m Model, text string) tea.MouseClickMsg {
	t.Helper()
	for y, l := range strings.Split(view(m), "\n") {
		if k := strings.Index(l, text); k >= 0 {
			return click(ansi.StringWidth(l[:k]), y)
		}
	}
	t.Fatalf("no %q on the screen", text)
	return tea.MouseClickMsg{}
}

func TestTheSearchNarrowsTheTerms(t *testing.T) {
	m, _ := loaded(t, press('5'), press('/'))
	m = drive(m, testkit.Typed("worktree")...)
	got := view(m)
	if !strings.Contains(got, pill("worktree")) || strings.Contains(got, pill("desk")) || !strings.Contains(got, "▾ 1 ") {
		t.Errorf("searched:\n%s", got)
	}
	// enter gives the rows the keyboard, the search kept
	if m = drive(m, enter); !termsSheet(m).InRows() || !m.inPane || !strings.Contains(view(m), pill("worktree")) {
		t.Errorf("enter from the search: in the rows %v", termsSheet(m).InRows())
	}
	m = drive(m, press('/'), tea.KeyPressMsg{Code: 'u', Mod: tea.ModCtrl})
	m = drive(m, testkit.Typed("nothing like it")...)
	if !strings.Contains(view(m), "no row matches the search") {
		t.Errorf("none searched:\n%s", view(m))
	}
	if m = drive(m, enter); !m.inPane || termsSheet(m).InRows() {
		t.Errorf("enter from a search no term matches: in the pane %v, in the rows %v", m.inPane,
			termsSheet(m).InRows())
	}
}

func TestAClickReachesTheTermsTable(t *testing.T) {
	fades = time.Millisecond
	t.Cleanup(func() { fades = table.Fades })
	m, _ := loaded(t, press('5'))
	lines := strings.Split(view(m), "\n")
	at := func(text string) (int, int) {
		for y, l := range lines {
			if k := strings.Index(l, text); k >= 0 {
				return ansi.StringWidth(l[:k]), y
			}
		}
		t.Fatalf("no %q on the screen", text)
		return 0, 0
	}
	// a click on a term puts the cursor there, the table with the keyboard
	x, y := at(pill("worktree"))
	if m = drive(m, click(x, y)); !m.inPane || !termsSheet(m).InRows() {
		t.Fatalf("a click on worktree: in the pane %v", m.inPane)
	}
	// a click on the selector drops the presets down, a layer that takes the next click, on Compact
	x, y = at("Default ▼")
	if m = drive(m, click(x, y)); !termsSheet(m).Layered() {
		t.Fatal("a click on the selector")
	}
	lines = strings.Split(view(m), "\n")
	x, y = at("Compact")
	if m = drive(m, click(x, y)); termsSheet(m).Layered() || termsSheet(m).Table().ViewName != "Compact" {
		t.Fatalf("a click on Compact: %q", termsSheet(m).Table().ViewName)
	}
	header := strings.Split(view(m), "\n")[paneTop+6]
	if !strings.Contains(header, "term ") || !strings.Contains(header, "meaning") ||
		strings.Contains(header, "command") || strings.Contains(header, "examples") {
		t.Errorf("Compact hides the command and the examples, a row a term:\n%s", view(m))
	}
	// Save As… saves the table as a preset of the person's, which the keys bar says, until it fades
	termsSheet(m).Focus()
	// on the picker, enter drops the presets down, s moves past Save, dim on Compact, to Save As…, and enter takes it
	m = drive(m, testkit.Press("tab"), enter, testkit.Press("s"), enter)
	m = drive(m, testkit.Typed("mine")...)
	next, cmd := m.Update(enter)
	m = next.(Model)
	if bottom := lastLine(m); !strings.Contains(bottom, "saved the preset mine") || cmd == nil {
		t.Fatalf("saved: %q", bottom)
	}
	if m = drive(m, cmd()); strings.Contains(lastLine(m), "saved") {
		t.Errorf("the notice did not fade: %q", lastLine(m))
	}
	// a click off the table, on the lead, is no one's
	if m = drive(m, click(5, paneTop)); termsSheet(m).Layered() {
		t.Error("a click on the lead")
	}
}

// lastLine gives the screen's last line, the keys bar, without its escapes.
func lastLine(m Model) string {
	lines := strings.Split(view(m), "\n")
	return lines[len(lines)-1]
}

func TestTheTermsTableFollowsTheSettingsAndTheScreensSize(t *testing.T) {
	m, _ := loaded(t, press('5'))
	m = drive(m, shell.SettingsMsg{Settings: shell.Settings{FoldShared: false, SharedEmoji: true}})
	if p := termsSheet(m).Panel(); p.Fold || !p.Glyphs {
		t.Error("the settings")
	}
	// a short screen has no room for the lead
	if m = drive(m, tea.WindowSizeMsg{Width: 120, Height: 14}); strings.Contains(view(m), "The words kit uses") ||
		!strings.Contains(view(m), "▾ 12 ") {
		t.Errorf("a short screen:\n%s", view(m))
	}
	// settings told before the pages are read hold for the table once they are
	m, _ = fake(t)
	m = drive(m, shell.SettingsMsg{Settings: shell.Settings{FoldShared: false}})
	m = drive(m, append(testkit.Start(m, at120), press('5'))...)
	if termsSheet(m).Panel().Fold {
		t.Error("the settings told first")
	}
}

func TestATermsPageWithoutTermsIsAPage(t *testing.T) {
	r := repoWith(t, map[string]string{"5-terms.md": "# Our words\n\nWe say *ship* for a release.\n"})
	m := New(r, "")
	m = drive(m, append(testkit.Start(m, at120), press('5'))...)
	if m.gridded() || !strings.Contains(view(m), "We say *ship* for a release.") {
		t.Errorf("a page without terms:\n%s", view(m))
	}
	if !strings.Contains(keysAt(m), "↓ page") {
		t.Errorf("its keys: %q", keysAt(m))
	}
}

func TestTheEdgesOfTheTermsStep(t *testing.T) {
	// a catalog that does not read is kittui's own fault, which panics
	func() {
		defer func() {
			if recover() == nil {
				t.Error("a catalog that does not read")
			}
		}()
		readCatalog([]byte("{"))
	}()
	// terms with no lead have the table at the pane's top; a refusal stands in the keys bar's place, in red
	r := repoWith(t, map[string]string{"5-terms.md": "## desk\n\nYour root checkout.\n\n`kit flow desk`\n"})
	m := New(r, "")
	m = drive(m, append(testkit.Start(m, at120), press('5'), down, testkit.Press("tab"), enter, testkit.Press("s"),
		enter, enter)...)
	if lines := strings.Split(view(m), "\n"); !strings.Contains(lines[paneTop], "▾ 1 ") ||
		!strings.Contains(lastLine(m), "a preset needs a name") || !strings.Contains(m.sheetNotice(), "\x1b[") {
		t.Errorf("no lead, and a refusal:\n%s", view(m))
	}
	// a click under the pane of a page is no one's
	if m, _ = loaded(t, press('4'), click(5, 29)); m.inPane {
		t.Error("a click under a page's pane")
	}
}

func TestTheCommandTermsAndExamplesStandWholeWhileTheMeaningGivesWay(t *testing.T) {
	// at 80 columns no command, term, or example is cut: the meaning gives way
	m, _ := loaded(t, tea.WindowSizeMsg{Width: 80, Height: 40}, press('5'))
	got := view(m)
	for _, whole := range []string{"▸ git remote    2             A remote is anot…  git remote --verbose",
		"▸ kit flow      " + pill("code forge") + "  The service that…  kit flow incoming --no-fetch"} {
		if !strings.Contains(got, whole) {
			t.Errorf("at 80, no %q in:\n%s", whole, got)
		}
	}
	// at 60, the terms are drawn as text, not pills; the meaning keeps a few cells, and the widest of the others,
	// the examples, give way after it
	m = drive(m, tea.WindowSizeMsg{Width: 60, Height: 40})
	got = view(m)
	for _, want := range []string{"▸ git remote    2           A remote is…  git remote --v…",
		"▸ kit flow      worktree    A checkout …  kit flow wo… +1"} {
		if !strings.Contains(got, want) {
			t.Errorf("at 60, no %q in:\n%s", want, got)
		}
	}
}
