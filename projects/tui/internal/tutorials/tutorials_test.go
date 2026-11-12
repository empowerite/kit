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

	"github.com/empowerite/kit/projects/tui/internal/grid"
	"github.com/empowerite/kit/projects/tui/internal/kit"
	"github.com/empowerite/kit/projects/tui/internal/pane"
	"github.com/empowerite/kit/projects/tui/internal/shell"
	"github.com/empowerite/kit/projects/tui/internal/style"
	"github.com/empowerite/kit/projects/tui/internal/testkit"
)

func TestATutorialsPageOpensWithItsTitleLine(t *testing.T) {
	m, _ := shipped(t, tabKey, tabKey, enter)
	page := m.pageLines()
	end := slices.Index(page[1:], "") + 1
	title := lead + style.Label("kit's") + " " + style.Label("applications") + " — "
	if page[0] != "" || end < 2 || !strings.HasPrefix(page[1], title) {
		t.Fatalf("the page opens with a blank, then its title bold, a dash, and its summary:\n%s", strings.Join(page, "\n"))
	}
	var words []string
	for _, l := range page[1:end] {
		words = append(words, strings.Fields(ansi.Strip(l))...)
	}
	want := "kit's applications — kit has an application for each side of the work, from Flow, the trip of the work " +
		"from an issue to a release, to Packages"
	if got := strings.Join(words, " "); got != want {
		t.Errorf("the title line, its summary without its period: got %q\nwant %q", got, want)
	}
	if !strings.Contains(page[end+1], "Each application shows one side of the work in this repository.") {
		t.Errorf("a blank, then the page: %q", page[end+1])
	}
	if got := titleOf(kit.Tutorial{Title: "Gone"}); got != "# Gone" {
		t.Errorf("a tutorial with no summary is titled by its title alone: %q", got)
	}
}

// The arrows move the cursor in reading order, left and right along a row and on to the next, up and down a column,
// down to the last card from a row below which it has none; up from the first row gives the control line the
// keyboard, and down from it gives it back to the cards. The control line is the top: up and esc leave it there.
func TestArrowsMoveBetweenCardsAndRows(t *testing.T) {
	m, _ := withEight(t, at120)
	if m.columns() != 3 || !m.inPane || m.cursor != 0 {
		t.Fatalf("at 120: %d columns, in the pane %v, cursor %d", m.columns(), m.inPane, m.cursor)
	}
	for _, step := range []struct {
		key    tea.KeyPressMsg
		cursor int
	}{
		{backTab, 0}, {tabKey, 1}, {tabKey, 2}, {tabKey, 3}, {down, 6}, {tabKey, 7}, {tabKey, 7}, {down, 7}, {up, 4},
		{backTab, 3}, {backTab, 2}, {down, 5}, {down, 7}, {up, 4}, {up, 1},
	} {
		if m = drive(m, step.key); m.cursor != step.cursor || !m.inPane {
			t.Fatalf("%s: cursor %d, want %d, in the pane %v", step.key, m.cursor, step.cursor, m.inPane)
		}
	}
	if m = drive(m, up); m.inPane || !m.controlled() {
		t.Fatal("up from the first row did not give the control line the keyboard")
	}
	if strings.Contains(m.View().Content, "\x1b[1;96m╭") {
		t.Error("a card is drawn under the cursor while the control line has the keyboard")
	}
	if m = drive(m, up, esc); !m.controlled() {
		t.Error("up and esc left the control line")
	}
	if m = drive(m, down); !m.inPane || m.controlled() || m.cursor != 1 {
		t.Errorf("down from the control line: in the pane %v, cursor %d", m.inPane, m.cursor)
	}
	if m = drive(m, esc); m.inPane || !m.controlled() {
		t.Error("esc did not give the control line the keyboard")
	}
	if m = drive(m, down); !m.inPane {
		t.Error("down on the control line did not give the cards the keyboard")
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
		end := at + m.cardHeight() - 2 // the card's bottom edge, from its title's line
		if at < 1 || end >= len(ls)-1 || !strings.HasPrefix(ls[at-1], "  ╭") || !strings.HasPrefix(ls[end], "  ╰") {
			t.Fatalf("the cursor's card %d is not whole:\n%s", k+1, view(m))
		}
	}
	whole(0)
	if !strings.Contains(view(m), "⬇⁶") || strings.Contains(view(m), pane.Up) {
		t.Errorf("at the top:\n%s", view(m))
	}
	for k := 1; k < 8; k++ {
		m = drive(m, down)
		whole(k)
	}
	if !strings.Contains(view(m), "⬆₆") || strings.Contains(view(m), pane.Down) {
		t.Errorf("at the bottom:\n%s", view(m))
	}
	for k := 6; k >= 0; k-- {
		m = drive(m, backTab)
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
	if m = drive(m, at60); m.shownUntil(m.bands(), m.offset)-m.offset != 1 || !strings.Contains(view(m), "⬆₂") ||
		!strings.Contains(view(m), "⬇⁵") {
		t.Errorf("at 60×20, from band %d:\n%s", m.offset, view(m))
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
	older := m.note.FadeMsg()
	if m = drive(m, older); m.Noticing() {
		t.Error("the notice outlived its time")
	}
	m = drive(m, enter)
	if m = drive(m, older); !m.Noticing() {
		t.Error("an older notice's time took the newer one")
	}
	if m = drive(m, tabKey); m.Noticing() || m.cursor != 1 {
		t.Errorf("a key kept the notice: %q", lines(m)[29])
	}
	// a click on the second card, on its summary, opens it
	m = drive(m, testkit.Click(2+cardWidth+gap+5, paneTop+1+2))
	if len(b.urls) != 3 || !strings.HasSuffix(b.urls[2], "/kittui-tour/?kit=unreleased") || m.cursor != 1 || !m.inPane {
		t.Errorf("a click on the second card opened %q, cursor %d", b.urls, m.cursor)
	}
	// a click between cards, past them, under them, or above the pane opens nothing
	for _, at := range [][2]int{{2 + cardWidth, paneTop + 2}, {2 + 2*(cardWidth+gap) + 3, paneTop + 2},
		{5, paneTop + 1 + m.cardHeight() + 1}, {5, paneTop}, {0, paneTop + 2}} {
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
	m = drive(m, tabKey, enter)
	want := " Open https://empowerite.github.io/kit/tutorials/main/kittui-tour/?kit=unreleased in a browser"
	if last := lines(m)[29]; last != want || !m.Noticing() {
		t.Fatalf("the notice: %q", last)
	}
	if m = drive(m, m.note.FadeMsg()); !m.Noticing() {
		t.Error("the link went with time")
	}
	if m = drive(m, at60); lines(m)[19] != " https://empowerite.github.io/kit/tutorials/main/kittui-tou…" {
		t.Errorf("at 60: %q", lines(m)[19])
	}
	if m = drive(m, tea.WindowSizeMsg{Width: 100, Height: 20}); lines(m)[19] != want {
		t.Errorf("at 100: %q", lines(m)[19])
	}
	if m = drive(m, backTab); m.Noticing() {
		t.Error("a key kept the link")
	}
}

// The tags from the command line filter the cards by audience once the catalogue is read, as the audience's column
// menu would, to the tutorials for any audience they name; tags that name none filter nothing. kit is asked for every
// tutorial, so the person can drop the filter, and a catalogue read again keeps the filter the person left.
func TestTheTagsFromTheCommandLineFilterTheCards(t *testing.T) {
	calls := filepath.Join(t.TempDir(), "calls")
	t.Setenv("FAKE_TUTORIALS_CALLS", calls)
	ids := func(m Model) string {
		var out []string
		for _, c := range m.cards() {
			out = append(out, c.ID)
		}
		return strings.Join(out, " ")
	}
	for _, c := range []struct {
		tags   []string
		filter []string
		cards  string
	}{
		{[]string{"ui"}, []string{"UI"}, "kittui-tour"}, {[]string{"nonesuch"}, nil, "kit-tutorials kittui-tour"},
		{[]string{"nonesuch", "maintainer"}, []string{"Maintainer"}, ""},
		{[]string{"ui", "newcomer"}, []string{"Just starting", "UI"}, "kit-tutorials kittui-tour"},
		{nil, nil, "kit-tutorials kittui-tour"},
	} {
		m, _ := fake(t, c.tags...)
		m = drive(m, testkit.Start(m, at120)...)
		if f := m.sheet.Panel().Field(audienceField).Filter; !slices.Equal(f.Values, c.filter) || ids(m) != c.cards {
			t.Errorf("--tag %q: filter %q, cards %q; want %q and %q", c.tags, f.Values, ids(m), c.filter, c.cards)
		}
	}
	if got, _ := os.ReadFile(calls); string(got) != strings.Repeat("tutorials --json\n", 5) {
		t.Errorf("kit ran as %q", got)
	}
	m, _ := fake(t, "maintainer")
	m = drive(m, testkit.Start(m, at120)...)
	if v := view(m); !strings.Contains(v, "No tutorial matches the search or the filter") || !m.controlled() {
		t.Errorf("a filter that lets no tutorial through, the control line focused %v:\n%s", m.controlled(), v)
	}
	// the tags filter once: a catalogue read again keeps what the person set
	m.sheet.SetPanel(m.sheet.Panel().FilteredBy(audienceField, grid.Filter{}))
	if m = drive(m, catalogue(eight())); len(m.cards()) != 8 {
		t.Errorf("read again: %d cards", len(m.cards()))
	}
}

// kit tutorials' words are --tag=<tag> words, or --tag pairs, whose values are the tags, and the tutorial named, the
// first other word that is no option; anything else is ignored.
func TestTheWordsAreTheTagsAndTheTutorialNamed(t *testing.T) {
	for words, want := range map[string]struct {
		tags  []string
		focus string
	}{
		"":                           {nil, ""},
		"--tag a --tag b":            {[]string{"a", "b"}, ""},
		"stray --tag a --tag":        {[]string{"a"}, "stray"},
		"--tag a --other --tag c t9": {[]string{"a", "c"}, "t9"},
		"--tag=a t9 --tag=b":         {[]string{"a", "b"}, "t9"},
	} {
		if tags, focus := parse(strings.Fields(words)); !slices.Equal(tags, want.tags) || focus != want.focus {
			t.Errorf("%q: tags %q and %q, want %q and %q", words, tags, focus, want.tags, want.focus)
		}
	}
}

// A tutorial named on the command line, as kit tutorials <tutorial> names it, has the cursor on its card once the
// catalogue is read, the cards with the keyboard: among the cards the tags let through when they hold it, else among
// them all, the tags' filter dropped.
func TestATutorialNamedHasTheCursor(t *testing.T) {
	for _, c := range []struct {
		words    string
		filtered bool
		cursor   int
	}{
		{"kittui-tour", false, 1}, {"--tag=ui kittui-tour", true, 0}, {"--tag=newcomer kittui-tour", true, 1},
		{"--tag=ui kit-tutorials", false, 0}, {"nonesuch", false, 0}, {"--tag=ui nonesuch", true, 0},
	} {
		m := New(testkit.Runner(t, "tutorials"), nil)
		m.tags, m.focus = parse(strings.Fields(c.words))
		m = drive(m, testkit.Start(m, at120)...)
		if m.sheet.Panel().Filtering() != c.filtered || m.cursor != c.cursor || !m.focused() {
			t.Errorf("%q: filtered %v, cursor %d, cards focused %v; want %v and %d", c.words,
				m.sheet.Panel().Filtering(), m.cursor, m.focused(), c.filtered, c.cursor)
		}
	}
}

// The search is every screen's: / gives it focus, typing narrows the cards to those whose title, audiences, summary, or
// id holds it, ⌃U clears it, and enter gives the cards focus, the filter kept, or the control line where it lets no
// card through. A click on it gives it focus too.
func TestTheSearchNarrowsTheCards(t *testing.T) {
	m, _ := fake(t)
	m = drive(m, testkit.Start(m, at120)...)
	m = drive(m, press('/'))
	if !m.search.Focused || !strings.Contains(m.keys().String(), "⏎ cards type to search") {
		t.Fatalf("/: focused %v, keys %q", m.search.Focused, m.keys().String())
	}
	m = drive(m, testkit.Typed("TOUR")...)
	if cards := m.cards(); len(cards) != 1 || cards[0].ID != "kittui-tour" {
		t.Errorf("tour: %v\n%s", cards, view(m))
	}
	if m = drive(m, tea.KeyPressMsg{Code: 'u', Mod: tea.ModCtrl}); m.search.Query != "" || len(m.cards()) != 2 {
		t.Errorf("⌃U: %q, %d cards", m.search.Query, len(m.cards()))
	}
	m = drive(m, append(testkit.Typed("nothing"), enter)...)
	if m.search.Focused || m.search.Query != "nothing" || len(m.cards()) != 0 || !m.controlled() {
		t.Errorf("a search no card holds, then enter: %+v, %d cards", m.search, len(m.cards()))
	}
	if m = drive(m, append([]tea.Msg{press('/'), tea.KeyPressMsg{Code: 'u', Mod: tea.ModCtrl}},
		testkit.Typed("kit-tutorials")...)...); len(m.cards()) != 1 {
		t.Errorf("a search by id: %d cards", len(m.cards()))
	}
	m = drive(m, testkit.Click(m.searchAt()+1, 0), tea.KeyPressMsg{Code: 'u', Mod: tea.ModCtrl}, testkit.Typed("k")[0],
		enter)
	if m.search.Focused || !m.focused() || len(m.cards()) != 2 {
		t.Errorf("a click, k, and enter: %+v, cards focused %v", m.search, m.focused())
	}
}

// The table's control line stands where the tabs stood, and does what they did: By audience groups the cards under
// a heading for each audience, the table's own, its count with it, a tutorial for two under both, each heading a band
// of the pane; the audience's card filters by an audience, and a closed group folds its cards away. Its keys
// are the table's, but up and esc, which leave it where it is, and down enters the cards.
func TestTheControlLineGroupsAndFiltersTheCards(t *testing.T) {
	m, _ := loaded(t)
	if ls := lines(m); !strings.Contains(ls[controlTop], "▾ 2") || !strings.Contains(ls[controlTop], "Default ▼") {
		t.Fatalf("the control line: %q", ls[controlTop])
	}
	if m = drive(m, up); !m.controlled() {
		t.Fatal("up did not give the control line the keyboard")
	}
	if keys := m.keys().String(); keys != "= menu - workspace / search │ ↓ cards ⇥·⇧⇥ move ←→ fold" {
		t.Errorf("the control line's keys, on the rows' fold: %q", keys)
	}
	if m = drive(m, tabKey); m.keys().String() != "= menu - workspace / search │ ↓ cards ⇧⇥ groups ⏎ presets f fields" {
		t.Errorf("the control line's keys, on the presets: %q", m.keys().String())
	}
	if m = drive(m, enter); !m.sheet.Layered() || !strings.Contains(view(m), "By audience") {
		t.Fatalf("enter did not drop the presets down:\n%s", view(m))
	}
	m = drive(m, down, enter)
	if g := m.sheet.Panel().Groups(); !slices.Equal(g, []string{audienceField}) {
		t.Fatalf("By audience groups by %q:\n%s", g, view(m))
	}
	v, ls := view(m), lines(m)
	if !strings.Contains(ls[controlTop], "▾ audience") || !strings.Contains(ls[controlTop], "By audience ▼") {
		t.Errorf("the control line, grouped: %q", ls[controlTop])
	}
	starting := slices.IndexFunc(ls, func(l string) bool { return strings.Contains(l, "▾ "+pillText("Just starting")+" 2") })
	ui := slices.IndexFunc(ls, func(l string) bool { return strings.Contains(l, "▾ "+pillText("UI")+" 1") })
	if starting != paneTop+1 || ui != starting+1+m.cardHeight() || strings.Count(v, "Finding your way around kittui") != 2 {
		t.Errorf("the groups, at %d and %d:\n%s", starting, ui, v)
	}
	if len(m.cards()) != 3 || m.cards()[2].ID != "kittui-tour" {
		t.Errorf("the cards, grouped: %v", m.cards())
	}
	// down enters the cards, and down again passes the UI heading to its card
	if m = drive(m, down, down); !m.focused() || m.cursor != 2 {
		t.Errorf("down past a heading: cursor %d", m.cursor)
	}
	// a filter by an audience, as its card sets it, leaves the tutorials for it, under each of their groups, the
	// cursor kept among them
	m.sheet.SetPanel(m.sheet.Panel().FilteredBy(audienceField, grid.Filter{Values: []string{"UI"}}))
	m.settle()
	if len(m.cards()) != 2 || m.cards()[0].ID != "kittui-tour" || m.cursor != 1 ||
		strings.Contains(view(m), "Reading kit's") {
		t.Errorf("filtered by UI: %v, cursor %d\n%s", m.cards(), m.cursor, view(m))
	}
}

// Before kit answers, the pane says it is reading, and the control line is blank; kit's refusal is drawn in the pane.
func TestThePaneSaysWhatKitDid(t *testing.T) {
	m, _ := fake(t)
	m = drive(m, at120)
	if v := view(m); !strings.Contains(v, "reading…") || lines(m)[controlTop] != "" {
		t.Errorf("before kit answers:\n%s", v)
	}
	if m = drive(m, enter, down, tabKey); m.cursor != 0 {
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

// = or a click on the = drops the app's menu down, its Tutorials dim: a screen asks kittui to show it in this one's place,
// quit ends the screen, and ctrl-q quits as well, not interrupted; Close ends what the screen started.
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
	m = drive(m, esc, testkit.Click(shell.At+ansi.StringWidth(shell.App)+len(appLabel), shell.Top))
	if !m.appMenu.Open {
		t.Fatal("a click on the screen's name did not drop the menu down")
	}
	_, msgs = testkit.Drive(m, switched, testkit.Click(4, shell.Top+6))
	if !slices.Contains(msgs, tea.Msg(shell.SwitchMsg{Screen: shell.Issues})) {
		t.Errorf("a click on Issues asked %v", msgs)
	}
	if _, msgs := testkit.Drive(m, nil, testkit.Click(4, shell.Top+17)); !slices.Contains(msgs, tea.Msg(tea.QuitMsg{})) {
		t.Errorf("a click on quit: %v", msgs)
	}
	if m.chose(shell.Choice{}) != nil {
		t.Error("no choice did something")
	}
	next, cmd := m.Update(tea.KeyPressMsg{Code: 'q', Mod: tea.ModCtrl})
	if cmd == nil || cmd() != tea.Quit() || next.(Model).Interrupted() {
		t.Error("ctrl-q did not quit with 0")
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

// The screen gives the link, until a key, when the person is away from the desk.
func TestAwayFromTheDeskTheScreenGivesTheLink(t *testing.T) {
	m, _ := loaded(t)
	m.browse = func(string) error { return shell.ErrNoDisplay }
	m = drive(m, tabKey, enter)
	want := " Open https://empowerite.github.io/kit/tutorials/main/kittui-tour/?kit=unreleased in a browser"
	if last := lines(m)[29]; last != want || !m.Noticing() {
		t.Errorf("the notice: %q", last)
	}
}

// A card keeps to its cells whatever its tutorial holds: a title too long is cut with …, a summary is never cut, the
// card as tall as it needs, a word wider than the card broken across lines, a tutorial with no glyph starts with its
// title, and one for no audience has a plain bottom edge; an audience the catalogue lacks shows as its id, after those
// it lists. Pills that do not fit end with …, and one too wide for the edge alone has its text cut.
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
			[]string{"│ A long title A long title A long … │", "│ word word word word word word word │",
				"╰────────────────────────────────────╯"}},
		{kit.Tutorial{Title: "T", Glyph: "📘", Summary: strings.Repeat("x", 120), Audience: []string{"ops", "ui"}},
			[]string{"│ 📘 T                               │", "│ xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx │",
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
		m.catalog.Tutorials = []kit.Tutorial{c.tutorial}
		got := m.card(c.tutorial, false)
		if len(got) != m.cardHeight() {
			t.Fatalf("%d lines, not %d: %q", len(got), m.cardHeight(), got)
		}
		// the summary stands whole: its lines, read in order, are its text
		var text []string
		for _, l := range got[3 : 3+m.summaryHeight()] {
			text = append(text, strings.TrimSpace(strings.Trim(ansi.Strip(l), "│")))
		}
		if strings.Join(strings.Fields(strings.Join(text, " ")), " ") != strings.Join(strings.Fields(c.tutorial.Summary), " ") &&
			strings.Join(text, "") != c.tutorial.Summary {
			t.Errorf("the summary is not whole: %q", text)
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

// A card reads its top edge, the glyph of where the tutorial is at its right, the glyph and the title, a blank line, the
// whole summary, another blank line, so the summary stands clear of the pills, and its bottom edge; every card is as
// tall as the longest summary needs, a shorter one padded with blank lines.
func TestTheTitleIsFollowedByABlankLine(t *testing.T) {
	m, _ := loaded(t)
	got := m.card(m.catalog.Tutorials[0], false)
	want := []string{"╭" + strings.Repeat("─", cardWidth-7) + " " + Online + " ─╮", "│ 🎓 Reading kit's tutorials         │",
		"│" + strings.Repeat(" ", cardWidth-2) + "│", "│ How the tutorials are laid out,    │",
		"│ who each one is for, and how a     │", "│ page tells you that your kit is    │",
		"│ too old for a passage.             │",
		"│" + strings.Repeat(" ", cardWidth-2) + "│", bottomEdge(pillText("Just starting"))}
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
		bottom := m.card(tour, cursor)[m.cardHeight()-1]
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
	if bottom := m.card(tour, false)[m.cardHeight()-1]; !strings.Contains(bottom, style.Pill("UI", "")) {
		t.Errorf("an audience with no color: %q", bottom)
	}
}

// An answer that is not the latest read's is dropped: the cards stay as they were.
func TestAnAnswerThatIsNotTheLatestReadsIsDropped(t *testing.T) {
	m, _ := loaded(t)
	before := len(m.catalog.Tutorials)
	stray := catalogue(kit.Catalogue{})
	stray.Gen = m.reading.Gen() + 1
	if m = drive(m, stray); before == 0 || len(m.catalog.Tutorials) != before {
		t.Errorf("a stray answer replaced the catalogue: %d tutorials, %d before", len(m.catalog.Tutorials), before)
	}
}

// The menu bar names the workspace kittui says, as every screen's does, and - or a click on it drops its menu down.
func TestTheTutorialsMenuBarHasTheWorkspaceMenu(t *testing.T) {
	m, _ := fake(t)
	m = drive(m, append(testkit.Start(m, at120),
		shell.WorkspaceMsg{Workspace: kit.Workspace{Kind: "repository", Member: "github.com/o/r"}})...)
	if first := lines(m)[0]; !strings.HasPrefix(first, "  "+shell.App+" kit tutorials · "+shell.Workspaces+" o/r ") {
		t.Errorf("the menu bar: %q", first)
	}
	if m = drive(m, press('-')); !m.books.Open || !strings.Contains(m.keys().String(), "esc close") {
		t.Errorf("-: open %v, keys %q", m.books.Open, m.keys().String())
	}
	if m = drive(m, esc, testkit.Click(m.booksAt(), shell.Top)); !m.books.Open {
		t.Error("a click on the workspace menu did not drop it down")
	}
}

// The screen's catalog reads; one that does not read is a bug in the build, which panics.
func TestTheCatalogReads(t *testing.T) {
	if len(readCatalog(catalogJSON)) == 0 {
		t.Error("tutorials.json reads to no table")
	}
	defer func() {
		if recover() == nil {
			t.Error("a catalog that does not read does not panic")
		}
	}()
	readCatalog([]byte("{"))
}
