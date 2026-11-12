// A click does on the screen what the keys do: on a group's heading among the cards it opens nothing; on the control
// line, what the table's control line does, the rows' ▾ folding the cards away and the selector dropping the presets
// down, a layer that takes the next click; the searches remembered drop down from the search's Recall glyph, and a
// click on one, or a key, searches it again; and the workspace menu's lines pick.
package tutorials

import (
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

// cell gives the cell where the text first stands on the screen, on the line from or under it, or -1, -1.
func cell(m Model, text string, from int) (x, y int) {
	for k, l := range lines(m) {
		if i := strings.Index(l, text); k >= from && i >= 0 {
			return ansi.StringWidth(l[:i]), k
		}
	}
	return -1, -1
}

// byAudience gives the screen over the eight tutorials at the size, By audience chosen on the control line, the
// cards given the keyboard back.
func byAudience(t *testing.T, size tea.WindowSizeMsg) (Model, *browser) {
	t.Helper()
	m, b := withEight(t, size, up, tabKey, enter, down, enter, down)
	if !slices.Equal(m.sheet.Panel().Groups(), []string{audienceField}) || !m.focused() {
		t.Fatalf("By audience: groups %q, cards focused %v\n%s", m.sheet.Panel().Groups(), m.focused(), view(m))
	}
	return m, b
}

func TestAClickOnAHeadingOpensNothingAndTheControlLineFolds(t *testing.T) {
	m, b := byAudience(t, at120)
	x, y := cell(m, "▾ "+pillText("Just starting"), paneTop)
	if y < 0 || m.cardAt(x+2, y) >= 0 {
		t.Fatalf("no heading among the cards:\n%s", view(m))
	}
	if m = drive(m, testkit.Click(x+2, y)); len(b.urls) != 0 || m.cursor != 0 || !m.focused() {
		t.Errorf("a click on a heading: opened %q, cursor %d, cards focused %v", b.urls, m.cursor, m.focused())
	}
	// the rows' ▾ on the control line folds every card away under the headings, which stand alone, ▸: the control line
	// has the keyboard
	x, y = cell(m, "▾ 8", controlTop)
	if m = drive(m, testkit.Click(x, y)); len(m.cards()) != 0 || m.focused() || !m.controlled() || m.offset != 0 {
		t.Fatalf("a click on the rows' ▾: %d cards, cards focused %v\n%s", len(m.cards()), m.focused(), view(m))
	}
	if v := view(m); !strings.Contains(v, "▸ "+pillText("Just starting")+" 4") || strings.Contains(v, "╭") {
		t.Errorf("the cards folded:\n%s", v)
	}
	// its ▸ unfolds them, the control line keeping the keyboard
	if m = drive(m, testkit.Click(x, y)); len(m.cards()) != 8 || !m.controlled() {
		t.Errorf("a click on the rows' ▸: %d cards, controls %v\n%s", len(m.cards()), m.controlled(), view(m))
	}
	// the selector drops the presets down, a layer that takes the click on Default
	x, y = cell(m, "By audience ▼", controlTop)
	if m = drive(m, testkit.Click(x, y)); !m.sheet.Layered() {
		t.Fatalf("a click on the selector:\n%s", view(m))
	}
	x, y = cell(m, "Default", controlTop+1)
	if m = drive(m, testkit.Click(x, y)); m.sheet.Layered() || len(m.sheet.Panel().Groups()) != 0 {
		t.Errorf("a click on Default: layered %v, groups %q\n%s", m.sheet.Layered(), m.sheet.Panel().Groups(), view(m))
	}
}

// Moving up onto a group's first row of cards scrolls its heading back over it, where both fit.
func TestUpOntoAGroupsFirstRowShowsItsHeading(t *testing.T) {
	m, _ := byAudience(t, tea.WindowSizeMsg{Width: 120, Height: 20})
	m = drive(m, down, down, down)
	if m.cards()[m.cursor].ID != "t8" || strings.Contains(view(m), pillText("UI")+" 4") {
		t.Fatalf("down to the last card: %s\n%s", m.cards()[m.cursor].ID, view(m))
	}
	if m = drive(m, up); m.cards()[m.cursor].ID != "t2" || !strings.Contains(view(m), "▾ "+pillText("UI")+" 4") {
		t.Errorf("up onto the UI group's first row: %s\n%s", m.cards()[m.cursor].ID, view(m))
	}
}

// A search is remembered when it gives the cards focus; in the search, ctrl+r or a click on its Recall glyph drops the
// searches down, and a key or a click on one searches it again.
func TestTheTutorialsSearchIsRememberedAndRecalled(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	m := Open(testkit.Runner(t, "tutorials"), nil)
	m.browse = func(string) error { return nil }
	m = drive(m, append(append(testkit.Start(m, at120), press('/')), append(testkit.Typed("tour"), enter)...)...)
	if !slices.Equal(m.recent.Items, []string{"tour"}) || m.search.Focused {
		t.Fatalf("remembered: %q, search focused %v", m.recent.Items, m.search.Focused)
	}
	ctrlU, ctrlR := tea.KeyPressMsg{Code: 'u', Mod: tea.ModCtrl}, tea.KeyPressMsg{Code: 'r', Mod: tea.ModCtrl}
	if m = drive(m, press('/'), ctrlU, ctrlR); !m.recent.Open || !strings.Contains(m.keys().String(), "esc close") {
		t.Fatalf("ctrl+r: open %v, keys %q", m.recent.Open, m.keys().String())
	}
	if m = drive(m, enter); m.recent.Open || m.search.Query != "tour" || len(m.cards()) != 1 {
		t.Errorf("enter on tour: open %v, %q, %d cards", m.recent.Open, m.search.Query, len(m.cards()))
	}
	m = drive(m, ctrlU, testkit.Click(m.recallAt(), shell.Top))
	if !m.recent.Open {
		t.Fatalf("a click on the Recall glyph did not drop the searches down:\n%s", view(m))
	}
	_, left := m.recent.Box(m.recallAt(), m.width)
	if m = drive(m, testkit.Click(left+3, shell.Top+1)); m.recent.Open || m.search.Query != "tour" {
		t.Errorf("a click on tour: open %v, %q", m.recent.Open, m.search.Query)
	}
}

// A click on the workspace menu's Open Repository… asks kittui for the Open Repository… dialog, and closes the menu.
func TestTheWorkspaceMenuPicksByAClick(t *testing.T) {
	m, _ := loaded(t, shell.WorkspaceMsg{Workspace: kit.Workspace{Kind: "repository", Member: "github.com/o/r"}})
	m = drive(m, testkit.Click(m.booksAt(), shell.Top))
	x, y := cell(m, "Open Repository…", shell.Top+1)
	if m, produced := testkit.Drive(m, nil, testkit.Click(x, y)); m.books.Open ||
		!slices.Contains(produced, tea.Msg(shell.OpenMsg{})) {
		t.Errorf("a click on Open Repository… at %d, %d: open %v, %v", x, y, m.books.Open, produced)
	}
}

// What the table did, as saving a preset, stands in place of the keys until its time is up; why it did not, in red,
// until a key.
func TestTheTablesNoticeStandsInPlaceOfTheKeys(t *testing.T) {
	m, _ := loaded(t, up, tabKey, enter, testkit.Press("s"), enter, enter)
	if last := lines(m)[29]; !strings.Contains(last, "a preset needs a name") ||
		!strings.Contains(m.View().Content, style.Refusal(m.sheet.Notice().Text)) {
		t.Fatalf("Save As… with no name: %q", last)
	}
	m = drive(m, testkit.Typed("mine")...)
	next, cmd := m.Update(enter)
	if m = next.(Model); !strings.Contains(lines(m)[29], "saved the preset mine") || cmd == nil {
		t.Fatalf("saved: %q", lines(m)[29])
	}
	if m = drive(m, m.sheet.Notice().FadeMsg()); strings.Contains(lines(m)[29], "saved") {
		t.Errorf("the notice outlived its time: %q", lines(m)[29])
	}
}

// What kittui's settings say reaches the table: whether a value every row shares folds, and its glyphs.
func TestTheSettingsReachTheTutorialsTable(t *testing.T) {
	m, _ := loaded(t, shell.SettingsMsg{Settings: shell.Settings{FoldShared: false, SharedEmoji: true}})
	if p := m.sheet.Panel(); p.Fold || !p.Glyphs {
		t.Errorf("the settings did not reach the table: fold %v, glyphs %v", p.Fold, p.Glyphs)
	}
}
