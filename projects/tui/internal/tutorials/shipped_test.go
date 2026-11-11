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
	if keys := m.keys().String(); keys != "= menu - workspace / search │ esc cards ↑↓ scroll ⇞⇟ page" {
		t.Errorf("keys %q", keys)
	}
	if m = drive(m, down, down); m.read.offset != 2 {
		t.Errorf("down twice: offset %d", m.read.offset)
	}
	if m = drive(m, up, up, up); m.read.offset != 0 {
		t.Errorf("up past the top: offset %d", m.read.offset)
	}
	if m = drive(m, press(tea.KeyEnd)); m.read.offset == 0 || !strings.Contains(view(m), "kit platform") {
		t.Errorf("end: offset %d\n%s", m.read.offset, view(m))
	}
	end := m.read.offset
	if m = drive(m, down); m.read.offset != end {
		t.Errorf("down past the end: offset %d, not %d", m.read.offset, end)
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
	// a click on the page itself does nothing
	if m = drive(m, testkit.Click(10, paneTop+3)); !m.read.open {
		t.Fatal("a click on the page closed it")
	}
	x, y = cell(m, Back, controlTop)
	if m = drive(m, testkit.Click(x, y)); m.read.open || !m.focused() || m.cursor != 2 {
		t.Errorf("a click on the line over the page: reading %v, cards focused %v", m.read.open, m.focused())
	}
	// / while reading gives the search focus, over the cards
	if m = drive(m, enter, press('/')); m.read.open || !m.search.Focused {
		t.Errorf("/ while reading: reading %v, search %v", m.read.open, m.search.Focused)
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
