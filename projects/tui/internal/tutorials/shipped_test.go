// A tutorial that ships with the kit is told from one online on its card, by its glyph on the card's top edge and by
// the where field the cards are grouped and filtered by; it opens here, its page in place of the cards, and esc, or a
// click on the line over it, goes back to the cards.
package tutorials

import (
	"slices"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/empowerite/kit/projects/tui/internal/grid"
	"github.com/empowerite/kit/projects/tui/internal/kit"
	"github.com/empowerite/kit/projects/tui/internal/shell"
	"github.com/empowerite/kit/projects/tui/internal/testkit"
)

// shipped opens the screen at 120×30 over testdata/shipped.json, two tutorials online and then one in the kit, whose
// page is testdata/applications.md, runs Init, then sends the messages.
func shipped(t *testing.T, msgs ...tea.Msg) (Model, *browser) {
	t.Helper()
	catalogueFile(t, "shipped.json")
	return loaded(t, msgs...)
}

func TestACardSaysWhetherItsTutorialIsInTheKitOrOnline(t *testing.T) {
	m, _ := shipped(t)
	top := lines(m)[paneTop+m.top()]
	if strings.Count(top, " "+Online+" ─╮") != 2 || strings.Count(top, " "+Shipped+" ─╮") != 1 ||
		!strings.HasSuffix(top, " "+Shipped+" ─╮") {
		t.Errorf("the cards' top edges: %q", top)
	}
	// each top edge is a card's width, as the rest of it is
	for _, edge := range strings.SplitAfter(strings.TrimSpace(top), "╮") {
		if edge = strings.TrimSpace(edge); edge != "" && ansi.StringWidth(edge) != cardWidth {
			t.Errorf("%q is %d cells", edge, ansi.StringWidth(edge))
		}
	}
	// where is a field the cards are filtered by
	m.sheet.SetPanel(m.sheet.Panel().FilteredBy(whereField, grid.Filter{Values: []string{inTheKit}}))
	if cards := m.cards(); len(cards) != 1 || cards[0].ID != "applications" {
		t.Errorf("filtered to the kit's: %v", cards)
	}
	m.sheet.SetPanel(m.sheet.Panel().FilteredBy(whereField, grid.Filter{Values: []string{online}}))
	if cards := m.cards(); len(cards) != 2 || cards[0].ID != "kit-tutorials" {
		t.Errorf("filtered to the online: %v", cards)
	}
}

// The preset In the kit or online groups the cards by where they are.
func TestThePresetGroupsTheCardsByWhereTheyAre(t *testing.T) {
	m, _ := shipped(t, up, tabKey, enter, down, down, enter, down)
	if !slices.Equal(m.sheet.Panel().Groups(), []string{whereField}) {
		t.Fatalf("groups %q\n%s", m.sheet.Panel().Groups(), view(m))
	}
	v := view(m)
	if !strings.Contains(v, pillText(inTheKit)) || !strings.Contains(v, pillText(online)) {
		t.Errorf("no heading for each place:\n%s", v)
	}
}

func TestATutorialInTheKitOpensHereAndEscGoesBack(t *testing.T) {
	m, b := shipped(t, tabKey, tabKey, enter)
	if len(b.urls) != 0 || !m.read.open {
		t.Fatalf("enter on the kit's tutorial: opened %q, reading %v", b.urls, m.read.open)
	}
	v := view(m)
	if !strings.Contains(v, "\n  Each application shows one side of the work in this repository.") ||
		strings.Contains(v, "title:") || strings.Contains(v, "╭") {
		t.Errorf("the page, without its front matter, in place of the cards:\n%s", v)
	}
	if got := lines(m)[controlTop]; !strings.HasPrefix(got, "  "+Back+" tutorials") ||
		!strings.Contains(got, "📚 kit's applications") {
		t.Errorf("the line over the page: %q", got)
	}
	if keys := m.keys().String(); keys != "= menu / search │ esc cards ↑↓ scroll ⇞⇟ page" {
		t.Errorf("keys %q", keys)
	}
	if m = drive(m, down, down); m.read.offset != 2 {
		t.Errorf("down twice: offset %d", m.read.offset)
	}
	if m = drive(m, up, up, up); m.read.offset != 0 {
		t.Errorf("up past the top: offset %d", m.read.offset)
	}
	// page down and page up scroll by the pane's lines less the two that count the hidden ones, on a pane shorter than
	// the page
	short := drive(m, tea.WindowSizeMsg{Width: 120, Height: 16})
	if short = drive(short, press(tea.KeyPgDown)); short.read.offset != short.paneHeight()-2 {
		t.Errorf("page down: offset %d, not %d", short.read.offset, short.paneHeight()-2)
	}
	if short = drive(short, press(tea.KeyPgUp)); short.read.offset != 0 {
		t.Errorf("page up: offset %d", short.read.offset)
	}
	// = drops the app's menu down over the page
	if menu := drive(m, press('=')); !menu.appMenu.Open || !menu.read.open {
		t.Errorf("=: the app menu open %v, reading %v", menu.appMenu.Open, menu.read.open)
	}
	if m = drive(m, press(tea.KeyEnd)); m.read.offset == 0 || !strings.Contains(view(m), "kit platform") {
		t.Errorf("end: offset %d\n%s", m.read.offset, view(m))
	}
	end := m.read.offset
	if m = drive(m, down); m.read.offset != end {
		t.Errorf("down past the end: offset %d, not %d", m.read.offset, end)
	}
	if home := drive(m, press(tea.KeyHome)); home.read.offset != 0 {
		t.Errorf("home: offset %d", home.read.offset)
	}
	if m = drive(m, esc); m.read.open || m.cursor != 2 || !m.focused() || !strings.Contains(view(m), "╭") {
		t.Errorf("esc: reading %v, cursor %d, cards focused %v\n%s", m.read.open, m.cursor, m.focused(), view(m))
	}
}

func TestAClickOpensATutorialInTheKitAndAClickOverItGoesBack(t *testing.T) {
	m, b := shipped(t)
	x, y := cell(m, "kit's applications", paneTop)
	if m = drive(m, testkit.Click(x, y)); len(b.urls) != 0 || !m.read.open || m.cursor != 2 {
		t.Fatalf("a click on the kit's card: opened %q, reading %v, cursor %d", b.urls, m.read.open, m.cursor)
	}
	// a click on the page, off its links, does nothing
	if m = drive(m, testkit.Click(10, paneTop+3)); !m.read.open {
		t.Fatal("a click on the page closed it")
	}
	x, y = cell(m, Back, controlTop)
	if m = drive(m, testkit.Click(x, y)); m.read.open || !m.focused() || m.cursor != 2 {
		t.Errorf("a click on the line over the page: reading %v, cards focused %v", m.read.open, m.focused())
	}
	// / while reading gives the search focus, the page kept under it
	if m = drive(m, enter, press('/')); !m.read.open || !m.search.Focused {
		t.Errorf("/ while reading: reading %v, search %v", m.read.open, m.search.Focused)
	}
}

// A search while a tutorial's page is open keeps the page, highlights what it finds, and scrolls to the first line
// that holds it; n and N step through the lines, ⏎ gives the page the keyboard back, and esc goes back to the cards,
// narrowed as the search narrows them.
func TestASearchOnATutorialsPageHighlightsWhatItFinds(t *testing.T) {
	m, _ := shipped(t, tea.WindowSizeMsg{Width: 120, Height: 16}, tabKey, tabKey, enter)
	m = drive(m, testkit.Click(m.searchAt()+1, shell.Top))
	if !m.read.open || !m.search.Focused {
		t.Fatalf("a click on the search closed the page: reading %v, search %v", m.read.open, m.search.Focused)
	}
	if !strings.Contains(m.keys().String(), "⏎ page") {
		t.Errorf("the search's keys over a page: %q", m.keys().String())
	}
	m = drive(m, testkit.Typed("kit")...)
	first := m.read.hit
	if first <= 0 || m.read.offset == 0 || !strings.Contains(m.View().Content, "\x1b[7mkit\x1b[27m") || !m.read.open {
		t.Fatalf("typed kit: hit %d, offset %d, reading %v\n%s", first, m.read.offset, m.read.open, view(m))
	}
	m = drive(m, enter)
	if !m.read.open || m.search.Focused || !strings.Contains(m.keys().String(), "n·N match") {
		t.Fatalf("enter: reading %v, search %v, keys %q", m.read.open, m.search.Focused, m.keys().String())
	}
	if m = drive(m, testkit.Typed("n")...); m.read.hit == first {
		t.Errorf("n stayed on line %d", first)
	}
	if m = drive(m, testkit.Typed("N")...); m.read.hit != first {
		t.Errorf("N went to line %d, not back to %d", m.read.hit, first)
	}
	// a search no line holds goes to the page's top; cleared, and with the page back, n and N leave it be, unnamed
	if m = drive(m, append([]tea.Msg{press('/')}, testkit.Typed("qqqq")...)...); m.read.offset != 0 || m.read.hit != -1 {
		t.Errorf("qqqq: hit %d, offset %d", m.read.hit, m.read.offset)
	}
	if m = drive(m, tea.KeyPressMsg{Code: 'u', Mod: tea.ModCtrl}, enter, press('n')); m.read.hit != -1 ||
		strings.Contains(m.keys().String(), "n·N") {
		t.Errorf("n with nothing typed: hit %d, keys %q", m.read.hit, m.keys().String())
	}
	// a page opened with a search already typed is scrolled to its first line
	m = drive(m, esc, press('/'))
	m = drive(m, append(testkit.Typed("kit"), enter, tabKey, tabKey, enter)...)
	if !m.read.open || m.read.hit < 0 {
		t.Errorf("opened over a search: reading %v, hit %d\n%s", m.read.open, m.read.hit, view(m))
	}
}

func TestTheWheelScrollsATutorialInTheKit(t *testing.T) {
	m, _ := shipped(t, tea.WindowSizeMsg{Width: 120, Height: 16}, tabKey, tabKey, enter)
	if m = drive(m, turn(10, paneTop+2, true)); m.read.offset != wheelLines {
		t.Errorf("a turn down: offset %d", m.read.offset)
	}
	if m = drive(m, turn(10, paneTop+2, false), turn(10, paneTop+2, false)); m.read.offset != 0 {
		t.Errorf("up past the top: offset %d", m.read.offset)
	}
	if m = drive(m, press('='), turn(10, paneTop+2, true)); m.read.offset != 0 {
		t.Errorf("a turn under the app's menu: offset %d", m.read.offset)
	}
}

func TestATutorialInTheKitWithNoPageSaysWhereItWasLookedFor(t *testing.T) {
	m, _ := fake(t)
	c := kit.Catalogue{Tutorials: []kit.Tutorial{{ID: "gone", Title: "Gone", Where: kit.InTheKit,
		File: "/nowhere/gone.md"}}}
	m = drive(m, at120, catalogue(c), enter)
	if !m.read.open || !strings.Contains(view(m), Failed+" no page at /nowhere/gone.md") {
		t.Errorf("reading %v:\n%s", m.read.open, view(m))
	}
}

// A link on a tutorial's page to one of kittui's screens opens it, with the page as its crumb: the screen by its name,
// or one of its tabs by its stage; a click anywhere else on the page asks for nothing.
func TestALinkOnATutorialsPageOpensTheScreenItLeadsTo(t *testing.T) {
	m, _ := shipped(t, tabKey, tabKey, enter)
	asked := func(x, y int) tea.Msg {
		t.Helper()
		_, cmd := m.Update(testkit.Click(x, y))
		if cmd == nil {
			return nil
		}
		return cmd()
	}
	// the link's line, under the title line, whose summary names Flow too
	_, at := cell(m, "Open", paneTop)
	x, y := cell(m, "Flow,", at)
	if got := asked(x, y); got != (shell.SwitchMsg{Screen: shell.Flow, Crumb: true}) {
		t.Errorf("a click on Flow asked %#v", got)
	}
	x, y = cell(m, "Issues", paneTop)
	if got := asked(x+5, y); got != (shell.JumpMsg{Screen: shell.Flow, To: "issues/"}) {
		t.Errorf("a click on Issues' last letter asked %#v", got)
	}
	if got := asked(x+6, y); got != nil {
		t.Errorf("a click past Issues asked %#v", got)
	}
	if got := asked(x-1, y+1); got != nil {
		t.Errorf("a click under the link asked %#v", got)
	}
	// the page is the crumb's name, and the crumb puts the page back where it was scrolled to; with no page open, the
	// crumb is the cards'
	if name, _ := m.Spot(); name != "tutorials › kit's applications" {
		t.Errorf("the crumb's name %q", name)
	}
	if name, _ := drive(m, esc).Spot(); name != "tutorials" {
		t.Errorf("the cards' crumb's name %q", name)
	}
	_, spot := drive(m, down).Spot()
	back := drive(m, esc).Restore(spot).(Model)
	if !back.read.open || back.read.offset != 1 {
		t.Errorf("put back: reading %v, offset %d", back.read.open, back.read.offset)
	}
}

// A link on a tutorial's page leads where its target says: a tutorial, tutorials/ and its id, opens in place of the
// page, as a card does, one in the kit by its page and one online in the browser; a web address opens the browser; a
// screen's tab with what it keeps in view is a jump to it; a tutorial the catalogue does not list leads nowhere.
func TestALinkOnATutorialsPageOpensTheTutorialOrTheAddressItLeadsTo(t *testing.T) {
	m, b := shipped(t, tabKey, tabKey, enter)
	if got, ok := m.follow("tutorials/applications")().(readMsg); !ok || got.tutorial.ID != "applications" {
		t.Errorf("a link to a tutorial in the kit gave %#v", got)
	}
	if m.follow("tutorials/applications") == nil || m.follow("tutorials/nowhere") != nil || m.follow("") != nil {
		t.Error("a link to a tutorial the catalogue lacks, or none, led somewhere")
	}
	if cmd := m.follow("tutorials/kittui-tour"); cmd == nil {
		t.Error("a link to an online tutorial led nowhere")
	} else if cmd(); !slices.Equal(b.urls, []string{m.catalog.Tutorials[1].URL}) {
		t.Errorf("an online tutorial opened %q", b.urls)
	}
	b.urls = nil
	if got := m.follow("https://example.com/docs")(); !slices.Equal(b.urls, []string{"https://example.com/docs"}) ||
		got.(browsedMsg).err != nil {
		t.Errorf("a web address opened %q: %#v", b.urls, got)
	}
	if got := m.follow("welcome/terms/workspace")(); got != (shell.JumpMsg{Screen: shell.Welcome, To: "terms/workspace"}) {
		t.Errorf("a term asked %#v", got)
	}
	if got := m.follow("welcome")(); got != (shell.SwitchMsg{Screen: shell.Welcome, Crumb: true}) {
		t.Errorf("a screen asked %#v", got)
	}
	// a click on a link in the page follows it: opening a tutorial in place of the page
	m = drive(m, m.follow("tutorials/applications")())
	if !m.read.open || m.read.tutorial.ID != "applications" {
		t.Errorf("reading %v: %q", m.read.open, m.read.tutorial.ID)
	}
}

// Another screen's link to a tutorial, a JumpMsg, opens it in place of the cards, once the catalogue has come where it
// had not; a tutorial the catalogue does not list opens nothing, and is forgotten.
func TestAnotherScreensLinkOpensATutorial(t *testing.T) {
	m, _ := shipped(t, shell.JumpMsg{Screen: shell.Tutorials, To: "applications/"})
	if !m.read.open || m.read.tutorial.ID != "applications" || m.landing != "" {
		t.Errorf("a jump after the catalogue: reading %v, %q, landing %q", m.read.open, m.read.tutorial.ID, m.landing)
	}
	catalogueFile(t, "shipped.json")
	early, _ := fake(t)
	early = drive(early, shell.JumpMsg{Screen: shell.Tutorials, To: "applications/"})
	if early.read.open || early.landing != "applications" {
		t.Fatalf("a jump before the catalogue: reading %v, landing %q", early.read.open, early.landing)
	}
	early = drive(early, testkit.Start(early, at120)...)
	if !early.read.open || early.read.tutorial.ID != "applications" || early.landing != "" {
		t.Errorf("the catalogue came: reading %v, %q, landing %q", early.read.open, early.read.tutorial.ID, early.landing)
	}
	if m = drive(m, esc, shell.JumpMsg{Screen: shell.Tutorials, To: "nowhere/"}); m.read.open || m.landing != "" {
		t.Errorf("a tutorial not listed: reading %v, landing %q", m.read.open, m.landing)
	}
}
