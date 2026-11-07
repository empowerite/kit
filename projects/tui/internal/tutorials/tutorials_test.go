package tutorials

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/empowerite/kit/projects/tui/internal/kit"
	"github.com/empowerite/kit/projects/tui/internal/shell"
	"github.com/empowerite/kit/projects/tui/internal/style"
	"github.com/empowerite/kit/projects/tui/internal/testkit"
)

// The arrows move the cursor in reading order, left and right along a row and on to the next, up and down a column,
// down to the last card from a row below which it has none; up from the first row gives the bar the keyboard, and down
// from the bar gives it back to the cards.
func TestArrowsMoveBetweenCardsAndRows(t *testing.T) {
	m, _ := withEight(t, at120)
	if m.columns() != 3 || !m.inPane || m.cursor != 0 {
		t.Fatalf("at 120: %d columns, in the pane %v, cursor %d", m.columns(), m.inPane, m.cursor)
	}
	for _, step := range []struct {
		key    tea.KeyPressMsg
		cursor int
	}{
		{left, 0}, {right, 1}, {right, 2}, {right, 3}, {down, 6}, {right, 7}, {right, 7}, {down, 7}, {up, 4},
		{left, 3}, {left, 2}, {down, 5}, {down, 7}, {up, 4}, {up, 1},
	} {
		if m = drive(m, step.key); m.cursor != step.cursor || !m.inPane {
			t.Fatalf("%s: cursor %d, want %d, in the pane %v", step.key, m.cursor, step.cursor, m.inPane)
		}
	}
	if m = drive(m, up); m.inPane {
		t.Fatal("up from the first row kept the cards")
	}
	if strings.Contains(m.View().Content, "\x1b[1;96m╭") {
		t.Error("a card is drawn under the cursor while the bar has the keyboard")
	}
	if m = drive(m, down); !m.inPane || m.cursor != 1 {
		t.Errorf("down from the bar: in the pane %v, cursor %d", m.inPane, m.cursor)
	}
	if m = drive(m, esc); m.inPane {
		t.Error("esc kept the cards")
	}
	if m = drive(m, enter); !m.inPane {
		t.Error("enter on the bar did not give the cards the keyboard")
	}
	// at 60, one card a row
	if m = drive(m, at60); m.columns() != 1 {
		t.Errorf("at 60: %d columns", m.columns())
	}
	if m = drive(m, tea.WindowSizeMsg{Width: 20, Height: 20}); m.columns() != 1 {
		t.Errorf("at 20: %d columns", m.columns())
	}
}

// At 60×24, eight cards stand one a row, two rows at a time; the pane scrolls so the card under the cursor is always
// drawn whole, and says how many cards it hides above and below.
func TestTheCursorScrollsThePane(t *testing.T) {
	m, _ := withEight(t, tea.WindowSizeMsg{Width: 60, Height: 24})
	whole := func(k int) {
		t.Helper()
		ls := lines(m)
		at := slices.IndexFunc(ls, func(l string) bool { return strings.Contains(l, "Tutorial "+string(rune('1'+k))+" ") })
		if at < 1 || at+5 >= len(ls)-1 || !strings.HasPrefix(ls[at-1], "  ╭") || !strings.HasPrefix(ls[at+5], "  ╰") {
			t.Fatalf("the cursor's card %d is not whole:\n%s", k+1, view(m))
		}
	}
	whole(0)
	if !strings.Contains(view(m), "… 6 more") || strings.Contains(view(m), "above") {
		t.Errorf("at the top:\n%s", view(m))
	}
	for k := 1; k < 8; k++ {
		m = drive(m, down)
		whole(k)
	}
	if !strings.Contains(view(m), "… 6 above") || strings.Contains(view(m), "more") {
		t.Errorf("at the bottom:\n%s", view(m))
	}
	for k := 6; k >= 0; k-- {
		m = drive(m, left)
		whole(k)
	}
	if m.offset != 0 {
		t.Errorf("back at the first card, the pane is scrolled %d rows", m.offset)
	}
	// a terminal too short for a card still draws the cursor's card from its top
	m = drive(m, tea.WindowSizeMsg{Width: 60, Height: 8}, down, down)
	if ls := lines(m); !strings.Contains(strings.Join(ls, "\n"), "Tutorial 3") {
		t.Errorf("at 60×8:\n%s", view(m))
	}
	// at 60×20, a card at a time
	if m = drive(m, at60); m.rows() != 1 || !strings.Contains(view(m), "… 2 above") ||
		!strings.Contains(view(m), "… 5 more") {
		t.Errorf("at 60×20, %d rows:\n%s", m.rows(), view(m))
	}
}

// Enter on a card, or a click on one, asks the browser for its link, which carries the kit's version, and says so in
// place of the keys until its time is up or a key; a stale fade leaves a newer notice.
func TestEnterAndAClickOpenTheCardsURL(t *testing.T) {
	m, b := loaded(t, enter)
	want := "https://empowerite.github.io/kit/tutorials/main/kit-tutorials/?kit=unreleased"
	if len(b.urls) != 1 || b.urls[0] != want {
		t.Fatalf("enter opened %q", b.urls)
	}
	last := lines(m)[29]
	if last != " 🎓 Opened Reading kit's tutorials in the browser" || !m.Noticing() {
		t.Errorf("the notice: %q", last)
	}
	if m = drive(m, fadedMsg{m.fades}); m.Noticing() {
		t.Error("the notice outlived its time")
	}
	m = drive(m, enter)
	if m = drive(m, fadedMsg{m.fades - 1}); !m.Noticing() {
		t.Error("an older notice's time took the newer one")
	}
	if m = drive(m, right); m.Noticing() || m.cursor != 1 {
		t.Errorf("a key kept the notice: %q", lines(m)[29])
	}
	// a click on the second card, on its summary, opens it
	m = drive(m, testkit.Click(2+cardWidth+gap+5, paneTop+1+2))
	if len(b.urls) != 3 || !strings.HasSuffix(b.urls[2], "/kittui-tour/?kit=unreleased") || m.cursor != 1 || !m.inPane {
		t.Errorf("a click on the second card opened %q, cursor %d", b.urls, m.cursor)
	}
	// a click between cards, past them, under them, or above the pane opens nothing
	for _, at := range [][2]int{{2 + cardWidth, paneTop + 2}, {2 + 2*(cardWidth+gap) + 3, paneTop + 2},
		{5, paneTop + 1 + cardHeight + 1}, {5, paneTop}, {0, paneTop + 2}} {
		m = drive(m, testkit.Click(at[0], at[1]))
	}
	if len(b.urls) != 3 {
		t.Errorf("a click off the cards opened %q", b.urls[3:])
	}
}

// A browser that fails leaves the link in place of the keys, to open by hand, until a key; on a screen too narrow for
// the line, the link alone.
func TestABrowserThatFailsShowsTheLink(t *testing.T) {
	m, b := loaded(t)
	b.fail = errors.New("no opener")
	m = drive(m, right, enter)
	want := " Open https://empowerite.github.io/kit/tutorials/main/kittui-tour/?kit=unreleased in a browser"
	if last := lines(m)[29]; last != want || !m.Noticing() {
		t.Fatalf("the notice: %q", last)
	}
	if m = drive(m, fadedMsg{m.fades}); !m.Noticing() {
		t.Error("the link went with time")
	}
	if m = drive(m, at60); lines(m)[19] != " https://empowerite.github.io/kit/tutorials/main/kittui-tou…" {
		t.Errorf("at 60: %q", lines(m)[19])
	}
	if m = drive(m, tea.WindowSizeMsg{Width: 100, Height: 20}); lines(m)[19] != want {
		t.Errorf("at 100: %q", lines(m)[19])
	}
	if m = drive(m, left); m.Noticing() {
		t.Error("a key kept the link")
	}
}

// A tag from the command line selects its audience's tab once the catalogue is read, the first tag that names one;
// a tag that names none leaves All. kit is asked for every tutorial, so All holds them all.
func TestATagFromTheCommandLineSelectsItsTab(t *testing.T) {
	calls := filepath.Join(t.TempDir(), "calls")
	t.Setenv("FAKE_TUTORIALS_CALLS", calls)
	for _, c := range []struct {
		tags []string
		tab  int
	}{
		{[]string{"ui"}, 2}, {[]string{"nonesuch"}, 0}, {[]string{"nonesuch", "maintainer"}, 3}, {nil, 0},
	} {
		m, _ := fake(t, c.tags...)
		m = drive(m, testkit.Start(m, at120)...)
		if m.tab != c.tab {
			t.Errorf("--tag %q: tab %d, want %d", c.tags, m.tab, c.tab)
		}
	}
	if got, _ := os.ReadFile(calls); string(got) != strings.Repeat("tutorials --json\n", 4) {
		t.Errorf("kit ran as %q", got)
	}
	m, _ := fake(t, "ui")
	m = drive(m, testkit.Start(m, at120)...)
	if v := view(m); !strings.Contains(v, "Finding your way around kittui") || strings.Contains(v, "Reading kit's") {
		t.Errorf("the UI tab:\n%s", v)
	}
	if m = drive(m, press('4')); !strings.Contains(view(m), "No tutorials for Maintainer yet") {
		t.Errorf("a tab with no tutorial:\n%s", view(m))
	}
	// the tag selects once: a catalogue read again keeps the tab the person chose, or the last it still has
	if m = drive(m, press('1'), catalogueMsg{catalogue: eight()}); m.tab != 0 {
		t.Errorf("read again: tab %d", m.tab)
	}
	fewer := eight()
	fewer.Audiences = fewer.Audiences[:1]
	if m = drive(m, press('3'), catalogueMsg{catalogue: fewer}); m.tab != 1 {
		t.Errorf("read again with one audience: tab %d", m.tab)
	}
}

// The bar selects a tab by its number, by a click, or, while it has the keyboard, by the arrows; each tab counts its
// tutorials, and down enters the cards only where there are some.
func TestTheBarSelectsATabAndCountsItsTutorials(t *testing.T) {
	m, _ := loaded(t)
	if bar := lines(m)[2]; !strings.Contains(bar, "All 2") || !strings.Contains(bar, "Just starting 2") ||
		!strings.Contains(bar, "UI 1") || !strings.Contains(bar, "Maintainer 0") {
		t.Errorf("the counts: %q", bar)
	}
	if m = drive(m, up, right); m.tab != 1 || m.inPane {
		t.Fatalf("right on the bar: tab %d, in the pane %v", m.tab, m.inPane)
	}
	if keys := m.keys().String(); keys != "= menu │ ←→ tab ↓ cards 1-4 jump" {
		t.Errorf("the bar's keys: %q", keys)
	}
	x := strings.Index(lines(m)[2], "Maintainer")
	if m = drive(m, testkit.Click(x, 2)); m.tab != 3 {
		t.Fatalf("a click on Maintainer: tab %d", m.tab)
	}
	if m = drive(m, down, enter); m.inPane {
		t.Error("down entered a tab with no tutorials")
	}
	if keys := m.keys().String(); keys != "= menu │ ←→ tab 1-4 jump" {
		t.Errorf("the keys of an empty tab: %q", keys)
	}
	if m = drive(m, press('2'), down); !m.inPane || m.tab != 1 {
		t.Errorf("2 and down: tab %d, in the pane %v", m.tab, m.inPane)
	}
	if keys := m.keys().String(); keys != "= menu │ 1-4 jump │ esc Just starting ←→↑↓ move ⏎ open" {
		t.Errorf("the cards' keys: %q", keys)
	}
}

// Before kit answers, the pane says it is reading, and the bar counts nothing; kit's refusal is drawn in the pane.
func TestThePaneSaysWhatKitDid(t *testing.T) {
	m, _ := fake(t)
	m = drive(m, at120)
	if v := view(m); !strings.Contains(v, "reading…") || !strings.Contains(lines(m)[2], "All …") {
		t.Errorf("before kit answers:\n%s", v)
	}
	if m = drive(m, enter, down, right); m.cursor != 0 {
		t.Errorf("keys with no cards moved the cursor to %d", m.cursor)
	}
	t.Setenv("FAKE_TUTORIALS_REFUSE", "1")
	m = drive(m, m.Init()())
	if v := view(m); !strings.Contains(v, "❌ kit tutorials refuses for the test; run: kit help tutorials") {
		t.Errorf("kit's refusal:\n%s", v)
	}
	for err, want := range map[error]string{
		errors.New(""):  "kit tutorials --json failed",
		errors.New("x"): "x",
	} {
		if got := refusalOf(err); got != want {
			t.Errorf("refusalOf(%v): %q", err, got)
		}
	}
}

// = or a click on ≡ drops the app's menu down, its Tutorials dim: a screen asks kittui to show it in this one's place,
// quit ends the screen, and alt-esc quits as well, not interrupted; Close ends what the screen started.
func TestTheAppMenuSwitchesAndQuits(t *testing.T) {
	m, _ := loaded(t)
	m = drive(m, tea.KeyPressMsg{Code: '=', Text: "="})
	if v := view(m); !m.appMenu.Open || !strings.Contains(v, "Tutorials") ||
		!strings.HasSuffix(v, shell.Path{shell.MenuKeys()}.String()) {
		t.Fatalf("= did not drop the app's menu down:\n%s", v)
	}
	switched := func(msg tea.Msg) bool { _, ok := msg.(shell.SwitchMsg); return ok }
	_, msgs := testkit.Drive(m, switched, tea.KeyPressMsg{Code: 'h', Text: "h"}, enter)
	if !slices.Contains(msgs, tea.Msg(shell.SwitchMsg{Screen: shell.Help})) {
		t.Errorf("help asked %v", msgs)
	}
	m = drive(m, esc, testkit.Click(shell.At, shell.Top))
	if !m.appMenu.Open {
		t.Fatal("a click on ≡ did not drop the menu down")
	}
	_, msgs = testkit.Drive(m, switched, testkit.Click(4, shell.Top+3))
	if !slices.Contains(msgs, tea.Msg(shell.SwitchMsg{Screen: shell.Flow})) {
		t.Errorf("a click on Flow asked %v", msgs)
	}
	if _, msgs := testkit.Drive(m, nil, testkit.Click(4, shell.Top+8)); !slices.Contains(msgs, tea.Msg(tea.QuitMsg{})) {
		t.Errorf("a click on quit: %v", msgs)
	}
	if m.chose(shell.Choice{}) != nil {
		t.Error("no choice did something")
	}
	next, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEscape, Mod: tea.ModAlt})
	if cmd == nil || cmd() != tea.Quit() || next.(Model).Interrupted() {
		t.Error("alt-esc did not quit with 0")
	}
	m.Close()
	if m.ctx.Err() == nil {
		t.Error("Close did not end what the screen started")
	}
	if v := Open(m.runner, nil).View(); v.MouseMode != tea.MouseModeCellMotion || !v.AltScreen {
		t.Error("the screen is not on the alternate screen with the mouse")
	}
}

func TestCtrlCIsInterrupted(t *testing.T) {
	m, _ := loaded(t)
	next, cmd := m.Update(tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl})
	if cmd == nil || cmd() != tea.Quit() || !next.(Model).Interrupted() {
		t.Errorf("ctrl-c did not quit as interrupted")
	}
}

// browse runs the opener with the link, and does not wait for it.
func TestBrowseRunsTheOpener(t *testing.T) {
	dir := t.TempDir()
	record := filepath.Join(dir, "record")
	for _, opener := range []string{"open", "xdg-open"} {
		script := "#!/bin/sh\nprintf '%s\\n' \"$1\" > " + record + "\n"
		if err := os.WriteFile(filepath.Join(dir, opener), []byte(script), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("PATH", dir)
	cmd, err := start("https://example.test/?kit=1")
	if err != nil {
		t.Fatal(err)
	}
	if err := cmd.Wait(); err != nil {
		t.Fatal(err)
	}
	if got, _ := os.ReadFile(record); string(got) != "https://example.test/?kit=1\n" {
		t.Errorf("the opener was given %q", got)
	}
	t.Setenv("PATH", t.TempDir())
	if err := browse("https://example.test/"); err == nil {
		t.Error("no opener on PATH did not fail")
	}
}

// A card keeps to its cells whatever its tutorial holds: a title too long is cut with …, a summary that runs over ends
// its third line with … after a whole word, or inside a word wider than the card, a tutorial with no glyph starts with
// its title, and one for no audience has a plain bottom edge; an audience the catalogue lacks shows as its id, after
// those it lists. Pills that do not fit end with …, and one too wide for the edge alone has its text cut.
func TestACardKeepsToItsCells(t *testing.T) {
	m, _ := fake(t)
	m.catalog.Audiences = append(eight().Audiences, kit.Audience{ID: "long", Label: strings.Repeat("Long ", 8)},
		kit.Audience{ID: "wide", Label: "🎓 Twenty-nine cells and wide"},
		kit.Audience{ID: "edge", Label: strings.Repeat("a", 28) + "🎓b"})
	long := strings.Repeat("word ", 30)
	for _, c := range []struct {
		tutorial kit.Tutorial
		want     []string
	}{
		{kit.Tutorial{Title: strings.Repeat("A long title ", 4), Summary: long},
			[]string{"│ A long title A long title A long … │", "│ word word word word word word…     │",
				"╰────────────────────────────────────╯"}},
		{kit.Tutorial{Title: "T", Glyph: "📘", Summary: strings.Repeat("x", 120), Audience: []string{"ops", "ui"}},
			[]string{"│ 📘 T                               │", "│ xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx… │",
				bottomEdge(pillText("UI") + " " + pillText("ops"))}},
		{kit.Tutorial{Title: "T", Audience: []string{"ui", "newcomer", "wide"}},
			[]string{bottomEdge(pillText("Just starting") + " " + pillText("UI") + " …")}},
		{kit.Tutorial{Title: "T", Audience: []string{"wide"}},
			[]string{bottomEdge(pillText("🎓 Twenty-nine cells and wide"))}},
		{kit.Tutorial{Title: "T", Audience: []string{"long"}},
			[]string{bottomEdge(pillText("Long Long Long Long Long Long…"))}},
		{kit.Tutorial{Title: "T", Audience: []string{"wide", "long"}},
			[]string{bottomEdge(pillText("Long Long Long Long Long Lo…") + " …")}},
		{kit.Tutorial{Title: "T", Audience: []string{"edge"}},
			[]string{bottomEdge(pillText(strings.Repeat("a", 28) + "…"))}},
	} {
		got := m.card(c.tutorial, false)
		if len(got) != cardHeight {
			t.Fatalf("%d lines: %q", len(got), got)
		}
		for k, l := range got {
			if w := ansi.StringWidth(l); w != cardWidth {
				t.Errorf("line %d is %d cells: %q", k, w, ansi.Strip(l))
			}
		}
		plain := ansi.Strip(strings.Join(got, "\n"))
		for _, want := range c.want {
			if !strings.Contains(plain, want) {
				t.Errorf("no %q in\n%s", want, plain)
			}
		}
	}
	// a right click does nothing
	m, b := loaded(t)
	if m = drive(m, tea.MouseClickMsg{X: 5, Y: paneTop + 2, Button: tea.MouseRight}); len(b.urls) != 0 {
		t.Errorf("a right click opened %q", b.urls)
	}
}

// A card reads its top edge, the glyph and the title, a blank line, the summary on three lines, and its bottom edge.
func TestTheTitleIsFollowedByABlankLine(t *testing.T) {
	m, _ := loaded(t)
	got := m.card(m.catalog.Tutorials[0], false)
	want := []string{"╭" + strings.Repeat("─", cardWidth-2) + "╮", "│ 🎓 Reading kit's tutorials         │",
		"│" + strings.Repeat(" ", cardWidth-2) + "│", "│ How the tutorials are laid out,    │",
		"│ who each one is for, and how a     │", "│ page tells you that your kit is…   │",
		bottomEdge(pillText("Just starting"))}
	if plain := strings.Split(ansi.Strip(strings.Join(got, "\n")), "\n"); !slices.Equal(plain, want) {
		t.Errorf("the card:\n%s\nwant\n%s", strings.Join(plain, "\n"), strings.Join(want, "\n"))
	}
	if ls := lines(m); ls[paneTop+1+2] != "  "+want[2]+" "+want[2] {
		t.Errorf("the screen's line under the titles: %q", ls[paneTop+1+2])
	}
}

// Each audience is a pill on the card's bottom edge, its label on its color between the half circles, a cell apart,
// in the catalogue's order whatever the tutorial's; an audience the catalogue gives no color is drawn on the pastel its
// label picks, as a forge's label without one is. The pills keep their colors under the cursor, where the edge is cyan.
func TestAudiencesArePillsInTheirColors(t *testing.T) {
	m, _ := loaded(t)
	tour := m.catalog.Tutorials[1]
	if strings.Join(tour.Audience, " ") != "ui newcomer" {
		t.Fatalf("the fixture's tour is for %q", tour.Audience)
	}
	for _, cursor := range []bool{false, true} {
		bottom := m.card(tour, cursor)[cardHeight-1]
		pills := style.Pill("Just starting", "#15803d") + " " + style.Pill("UI", "#1d4ed8")
		if !strings.Contains(bottom, pills) {
			t.Errorf("cursor %v: no pills in their colors, in the catalogue's order, in %q", cursor, bottom)
		}
		if ansi.Strip(bottom) != bottomEdge(pillText("Just starting")+" "+pillText("UI")) {
			t.Errorf("cursor %v: %q", cursor, ansi.Strip(bottom))
		}
		// the edge's own escapes end before the first half circle and start after the last
		before, after, _ := strings.Cut(bottom, pills)
		if !strings.HasSuffix(before, "\x1b[m") || !strings.HasPrefix(after, "\x1b[") {
			t.Errorf("cursor %v: the edge runs into the pills: %q · %q", cursor, before, after)
		}
	}
	if !strings.Contains(m.View().Content, style.Pill("UI", "#1d4ed8")) {
		t.Error("the screen draws no UI pill")
	}
	m.catalog.Audiences[1].Color = ""
	if bottom := m.card(tour, false)[cardHeight-1]; !strings.Contains(bottom, style.Pill("UI", "")) {
		t.Errorf("an audience with no color: %q", bottom)
	}
}
