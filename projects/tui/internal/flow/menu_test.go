// The selected tab's menu, driven by = and by clicks: it drops down under the menu glyph, and opens or closes every
// row of the stage.
package flow

import (
	"path/filepath"
	"slices"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/empowerite/kit/projects/tui/internal/grid"
	"github.com/empowerite/kit/projects/tui/internal/shell"
	"github.com/empowerite/kit/projects/tui/internal/style"
	"github.com/empowerite/kit/projects/tui/internal/tabs"
	"github.com/empowerite/kit/projects/tui/internal/testkit"
)

// opened gives how many of the selected stage's rows are open.
func openRows(m Model) int {
	n := 0
	for _, open := range m.stages[m.stage].open {
		if open {
			n++
		}
	}
	return n
}

func TestColonDropsTheTabsMenuDownAndItOpensOrClosesEveryRow(t *testing.T) {
	m := loaded(t, options(), right, right, right, press(':'))
	if !m.menu || m.appMenu.Open || lines(m)[23] != " "+shell.Keys ||
		!strings.Contains(view(m), "│ "+Expand+" Expand All   │") || strings.Contains(view(m), "Settings") {
		t.Fatalf(": did not drop the tab's menu down:\n%s", view(m))
	}
	// the menu's left edge is two cells before the menu glyph, its top edge on the bar's middle line, through the
	// glyph, half a line either side of it, all in the menu's color; the keyboard starts on expand all, and collapse
	// all, with no row open, is dim
	glyph, cells := m.bar().MenuAt(), ansi.StringWidth(tabs.Menu)
	if box, left := m.menuBox(); left != glyph-2 || len(box) != len(m.tabItems())+2 || m.menuAt != 0 ||
		ansi.Cut(lines(m)[menuTop], glyph-1, glyph+cells+1) != "╴"+tabs.Menu+"╶" ||
		!strings.Contains(box[2], style.Off(" "+Collapse+" Collapse All ")) || box[0] != style.Menu(ansi.Strip(box[0])) {
		t.Errorf("the menu's left edge %d, the glyph at %d, the keyboard on %d:\n%s", left, glyph, m.menuAt, view(m))
	}
	// a letter moves to the next enabled item that starts with it; one no enabled item starts with moves nothing; down
	// passes collapse all by while it is dim
	for _, step := range []struct {
		letter string
		want   int
	}{{"c", 0}, {"E", 0}, {"z", 0}} {
		if m = drive(m, tea.KeyPressMsg{Code: rune(step.letter[0]), Text: step.letter}); m.menuAt != step.want {
			t.Errorf("%s: item %d, want %d", step.letter, m.menuAt, step.want)
		}
	}
	if m = drive(m, down, up); m.menuAt != 0 {
		t.Errorf("up and down moved to a dim item: %d", m.menuAt)
	}
	// enter on expand all expands every worktree row that hides something; the menu closes
	if m = drive(m, press(tea.KeyEnter)); m.menu || openRows(m) == 0 || strings.Contains(view(m), "Expand All") {
		t.Fatalf("expand all: menu %v, open %d", m.menu, openRows(m))
	}
	opened := openRows(m)
	// now expand all is dim and the keyboard starts on collapse all, which space chooses; ; drops it down too
	if m = drive(m, press(';')); !m.menu || m.menuAt != 1 || m.enabled(0) {
		t.Errorf("with every row expanded the keyboard is on %d", m.menuAt)
	}
	// up passes the dim expand all by, and down the rule, to show headers
	if m = drive(m, up, down); m.menuAt != 3 {
		t.Errorf("down from collapse all: item %d", m.menuAt)
	}
	if m = drive(m, up, press(tea.KeySpace)); openRows(m) != 0 || m.menu {
		t.Errorf("collapse all: open %d", openRows(m))
	}
	// esc, :, and ; close it, choosing nothing, and = does not
	m = drive(m, press(':'), press(tea.KeyEnter))
	for _, closing := range []tea.Msg{esc, press(':'), press(';')} {
		if m = drive(m, press(':'), closing); m.menu || openRows(m) != opened {
			t.Errorf("%v did not close the menu alone: menu %v", closing, m.menu)
		}
	}
	if m = drive(m, press(':'), press('=')); !m.menu || m.appMenu.Open {
		t.Errorf("= in the tab's menu: menu %v, app %v", m.menu, m.appMenu.Open)
	}
}

// = or a click on ≡ before the title drops the app's menu down, internal/shell's, with the picture's own first:
// refresh all, which runs every stage again, the forge read afresh. The flow's own screen is dim; another screen asks
// kittui to show it in the picture's place, and quit ends the picture with 0.
func TestEqualsDropsTheAppsMenuDownToRefreshAnotherScreenOrQuit(t *testing.T) {
	m := loaded(t, options(), press('='))
	if !m.appMenu.Open || m.menu || !strings.HasPrefix(lines(m)[shell.Top], "╭╴"+shell.App+"╶") ||
		!strings.Contains(view(m), "│ "+Reload+" Refresh All │") || !strings.Contains(view(m), "Welcome") ||
		lines(m)[23] != " "+shell.Keys {
		t.Fatalf("= did not drop the app's menu down:\n%s", view(m))
	}
	if m = drive(m, esc); m.appMenu.Open {
		t.Error("esc did not close the app's menu")
	}
	// a click on ≡ drops it down; one on refresh all runs every stage again, the shared ones afresh
	record := filepath.Join(t.TempDir(), "calls")
	t.Setenv("FAKE_FLOW_CALLS", record)
	if m = drive(m, testkit.Click(shell.At, shell.Top)); !m.appMenu.Open {
		t.Fatalf("a click on %s:\n%s", shell.App, view(m))
	}
	m = drive(m, testkit.Click(3, shell.Top+1))
	got := calls(t, record)
	if m.appMenu.Open || !slices.Contains(got, "flow underway --json --fresh") || !slices.Contains(got, "flow builds --json") ||
		len(got) < len(eight) {
		t.Errorf("refresh all ran %q", got)
	}
	// welcome asks for the welcome, the picture kept as it is; quit ends it, not interrupted
	_, msgs := testkit.Drive(m, func(msg tea.Msg) bool { _, ok := msg.(shell.SwitchMsg); return ok },
		press('='), typed("w")[0], press(tea.KeyEnter))
	if !slices.Contains(msgs, tea.Msg(shell.SwitchMsg{Screen: shell.Welcome})) {
		t.Errorf("welcome asked %v", msgs)
	}
	after, msgs := testkit.Drive(m, nil, press('='), typed("q")[0], press(tea.KeyEnter))
	if !slices.ContainsFunc(msgs, func(msg tea.Msg) bool { _, ok := msg.(tea.QuitMsg); return ok }) || after.interrupted {
		t.Errorf("quit did not end the picture: %v", msgs)
	}
}

// On incoming, where no row hides anything, every item is dim: the keyboard is on none of them, moves to none, and
// enter does nothing.
func TestTheMenuOfAStageWithNothingToOpenHasNothingEnabled(t *testing.T) {
	m := loaded(t, options(), slices.Repeat([]tea.Msg{right}, 6)...)
	if m = drive(m, press(':'), down, up, typed("e")[0], press(tea.KeyEnter)); !m.menu || m.menuAt != -1 {
		t.Errorf("on %s the keyboard is on %d:\n%s", m.stages[m.stage].name, m.menuAt, view(m))
	}
	// nothing is drawn under the keyboard, even when the keyboard was left on an item the rows no longer let do
	// anything
	for _, at := range []int{-1, 0} {
		m.menuAt = at
		if box, _ := m.menuBox(); strings.Contains(strings.Join(box, "\n"), style.Cursor(" ")[:4]) {
			t.Errorf("with the keyboard on %d, a dim item is drawn under it: %q", at, box)
		}
	}
}

func TestAClickOnTheMenuGlyphDropsItDownAndAClickChoosesOrCloses(t *testing.T) {
	m := loaded(t, options(), right, right, right)
	glyph := testkit.Click(m.bar().MenuAt(), m.bar().Row+1)
	if m = drive(m, glyph); !m.menu {
		t.Fatalf("a click on the menu glyph:\n%s", view(m))
	}
	_, left := m.menuBox()
	// a click on collapse all while it is dim does nothing, and leaves the menu open
	for _, y := range []int{menuTop + 2} {
		if m = drive(m, testkit.Click(left+2, y)); !m.menu || openRows(m) != 0 {
			t.Errorf("a click on line %d: menu %v, open %d", y, m.menu, openRows(m))
		}
	}
	if m = drive(m, testkit.Click(left+2, menuTop+1)); m.menu || openRows(m) == 0 {
		t.Errorf("a click on expand all: menu %v, open %d", m.menu, openRows(m))
	}
	if m = drive(m, glyph, testkit.Click(left+2, menuTop+2)); m.menu || openRows(m) != 0 {
		t.Errorf("a click on collapse all: menu %v, open %d", m.menu, openRows(m))
	}
	// a click off the menu closes it, and does nothing else
	before := m.stages[m.stage].cursor
	if m = drive(m, glyph, testkit.Click(2, paneTop+8)); m.menu || m.stages[m.stage].cursor != before {
		t.Errorf("a click off the menu: menu %v", m.menu)
	}
}

func TestTheMenusKeysInThePanelDropThemButAreTypedInTheSearch(t *testing.T) {
	m := loaded(t, options(), tab)
	if m = drive(m, typed("=:;")...); m.menu || m.panel(0).Query != "=:;" {
		t.Fatalf("the menus' keys in the search: menu %v, search %q", m.menu, m.panel(0).Query)
	}
	for _, c := range []struct {
		key rune
		app bool
	}{{'=', true}, {':', false}, {';', false}} {
		if m = drive(m, down, press(c.key)); m.menu == c.app || m.appMenu.Open != c.app || !m.inPanel {
			t.Errorf("%c off the search: menu %v, app %v, in panel %v", c.key, m.menu, m.appMenu.Open, m.inPanel)
		}
		m = drive(m, esc)
	}
	// with no stage, neither drops, and no glyph stands before the title; the tab's menu's box keeps to the screen
	// when the bar draws no glyph
	if m = loaded(t, Options{}, press(':')); m.menu || !strings.Contains(view(m), shell.App) {
		t.Error("a menu with no stage")
	}
	m = loaded(t, options(), tea.WindowSizeMsg{Width: 20, Height: 24}, press(':'))
	if _, left := m.menuBox(); m.bar().MenuAt() >= 0 || left < 0 || !m.menu {
		t.Errorf("a bar too narrow for the glyph: glyph %d, left %d", m.bar().MenuAt(), left)
	}
}

// On the rightmost tab, a box from the glyph would run past the rows: the glyph takes the top edge's cell two before
// the right corner instead, or a nearer one when the rows end there; at every width, beside the sidebar or not, the
// whole box shows, the edge half a line either side of the glyph.
// What the box does not cover stays in its columns, a two-cell glyph it cuts in half giving way to a space.
func TestTheMenuOnTheRightmostTabEndsUnderTheGlyphAndLeavesTheRestInPlace(t *testing.T) {
	for _, width := range []int{60, 80, 97, 110, 131, 174, 200} {
		for _, sidebar := range []bool{false, true} {
			m := loaded(t, options(), tea.WindowSizeMsg{Width: width, Height: 24})
			if sidebar {
				m = drive(m, press('?'))
			}
			for i := range m.stages {
				m = drive(m, press(':'))
				box, left := m.menuBox()
				glyph, end := m.bar().MenuAt(), left+ansi.StringWidth(box[0])
				drawn, cells := lines(m), ansi.StringWidth(tabs.Menu)
				after := ansi.Cut(drawn[menuTop], glyph+cells, glyph+cells+1)
				if end > m.paneWidth() || glyph-left < 2 || ansi.Cut(drawn[menuTop], glyph-1, glyph+cells) != "╴"+tabs.Menu ||
					!(after == "╶" || glyph+cells == end-1 && after == "╮" || glyph+cells == end) ||
					ansi.Cut(drawn[menuTop+len(box)-1], left, end) != ansi.Strip(box[len(box)-1]) {
					t.Errorf("at %d, sidebar %v, tab %d: the box spans %d to %d, the glyph at %d:\n%s", width, sidebar, i,
						left, end-1, glyph, view(m))
				}
				m = drive(m, esc, right)
			}
		}
	}
	for _, line := range []string{"ab🎫cd", "🎫🎫🎫🎫"} {
		for at := range 4 {
			got := shell.Splice(line, "XY", at)
			if ansi.StringWidth(got) != ansi.StringWidth(line) || ansi.Cut(got, at, at+2) != "XY" {
				t.Errorf("Splice(%q, XY, %d) = %q", line, at, got)
			}
		}
	}
}

// After Expand All, the rows have the keyboard, so left closes the row under the cursor even once the wheel has
// scrolled its first line away; and a click on that first line, pinned atop the pane, closes it too.
func TestAfterExpandAllTheRowsHaveTheKeyboardAndThePinnedLineCloses(t *testing.T) {
	wheel := tea.MouseWheelMsg{X: 10, Y: paneTop + 2, Button: tea.MouseWheelDown}
	size := tea.WindowSizeMsg{Width: 100, Height: 12}
	m := loaded(t, options(), size, right, right, right, press(':'), press(tea.KeyEnter), wheel)
	if s := m.stages[m.stage]; !m.inRows || s.offset == 0 || !strings.HasPrefix(lines(m)[m.rowsTop()], " "+style.Opened) {
		t.Fatalf("after expand all and the wheel: in rows %v, offset %d:\n%s", m.inRows, s.offset, view(m))
	}
	opened := openRows(m)
	if m = drive(m, left); openRows(m) != opened-1 || m.stages[m.stage].name != "worktrees" {
		t.Errorf("left after expand all: open %d of %d, on %s", openRows(m), opened, m.stages[m.stage].name)
	}
	m = loaded(t, options(), size, right, right, right, press(':'), press(tea.KeyEnter), wheel)
	if m = drive(m, testkit.Click(1, m.rowsTop())); openRows(m) != opened-1 {
		t.Errorf("a click on the pinned line: open %d of %d:\n%s", openRows(m), opened, view(m))
	}
}

// Show Headers, in the tab's menu, stands the names of the stage's columns over its rows, as every stage opens, checked
// while it does: a header that never scrolls, a thin dim line under it, and no row of its own, so a click on it
// chooses nothing. Choosing it again hides them. Each stage keeps its own; incoming, whose rows are no columns, has
// none to show.
func TestShowHeadersStandsTheColumnsNamesOverTheRows(t *testing.T) {
	m := loaded(t, options(), tea.WindowSizeMsg{Width: 120, Height: 11}, right, right, right, press(':'))
	box, _ := m.menuBox()
	if !strings.Contains(ansi.Strip(box[4]), Checked+"  Show Headers") || m.rowsTop() == paneTop {
		t.Fatalf("the menu as the stage opens: %q", ansi.Strip(box[4]))
	}
	m = drive(m, typed("s")[0], press(tea.KeyEnter))
	if box, _ = drive(m, press(':')).menuBox(); m.stages[m.stage].headers || m.rowsTop() != paneTop ||
		!strings.Contains(ansi.Strip(box[4]), "    Show Headers") {
		t.Fatalf("show headers did not hide them:\n%s", view(m))
	}
	m = drive(m, press(':'), typed("s")[0], press(tea.KeyEnter))
	drawn := lines(m)
	if !m.stages[m.stage].headers || m.menu || !strings.Contains(drawn[paneTop], "branch") ||
		!strings.Contains(drawn[paneTop], "uncommitted") || !strings.HasPrefix(drawn[m.rowsTop()-1], "   ───") ||
		!strings.Contains(m.View().Content, strings.TrimSuffix(style.Off("─"), "─\x1b[m")+"───") || m.rowsTop() <= paneTop {
		t.Fatalf("the headers:\n%s", view(m))
	}
	if m = drive(m, press(':')); !strings.Contains(ansi.Strip(m.tabItems()[3].Glyph), Checked) {
		t.Errorf("show headers is not checked: %+v", m.tabItems()[3])
	}
	m = drive(m, esc)
	// the header stays while the rows scroll under it
	header := drawn[paneTop:m.rowsTop()]
	m = drive(m, down, down, down, down, down, down)
	if got := lines(m)[paneTop:m.rowsTop()]; !slices.Equal(got, header) || m.stages[m.stage].offset == 0 {
		t.Errorf("the header scrolled with the rows, offset %d:\n%s", m.stages[m.stage].offset, view(m))
	}
	// a click on the header chooses no row
	before := m.stages[m.stage].cursor
	if m = drive(m, testkit.Click(5, paneTop)); m.stages[m.stage].cursor != before {
		t.Errorf("a click on the header moved the cursor to %d", m.stages[m.stage].cursor)
	}
	// a column the panel hides has no name over it
	m.setPanel(grid.Panel{Hidden: map[string]bool{"uncommitted": true}})
	if strings.Contains(lines(m)[paneTop], "uncommitted") {
		t.Errorf("a hidden column's name:\n%s", view(m))
	}
	// the stage beside keeps its own: hidden here, still shown there
	m = drive(m, esc, press(':'), typed("s")[0], press(tea.KeyEnter))
	if m = drive(m, right); m.rowsTop() == paneTop {
		t.Errorf("changes lost its headers with the worktrees'")
	}
	m = loaded(t, options(), slices.Repeat([]tea.Msg{right}, 6)...)
	if m.stages[m.stage].name != "incoming" || m.enabled(3) {
		t.Errorf("%s offers headers", m.stages[m.stage].name)
	}
}

// A header whose names take more than a line shows its first alone, with ▸ before it, until it is opened: by a click
// on its gutter, as a row's, or by Expand All, which Collapse All undoes; the rows then start under every line of it.
func TestAHeaderTallerThanALineOpensAsARowDoes(t *testing.T) {
	long := slices.Clone(columns["backlog"])
	long[len(long)-1] = "the title of the issue, as its tracker gives it, in full"
	defer func(was []string) { columns["backlog"] = was }(columns["backlog"])
	columns["backlog"] = long
	m := loaded(t, options(), tea.WindowSizeMsg{Width: 60, Height: 24})
	closed := m.rowsTop()
	if lines, hides := m.header(); !hides || !strings.HasPrefix(lines[0], " "+style.Off("▸")) || closed != paneTop+2 {
		t.Fatalf("the closed header:\n%s", view(m))
	}
	if m = drive(m, testkit.Click(1, paneTop)); !m.stages[0].headed || m.rowsTop() <= closed ||
		!strings.HasPrefix(lines(m)[paneTop], " ▾") || m.inRows {
		t.Fatalf("a click on its gutter did not open it:\n%s", view(m))
	}
	if m = drive(m, testkit.Click(1, paneTop)); m.stages[0].headed {
		t.Error("a second click did not close it")
	}
	// Expand All opens it with the rows, and Collapse All closes it, each enabled for the header alone too
	m = drive(m, press(':'))
	if !m.enabled(0) || m.enabled(1) {
		t.Errorf("expand all %v, collapse all %v, over a closed header", m.enabled(0), m.enabled(1))
	}
	if m = drive(m, press(tea.KeyEnter)); !m.stages[0].headed {
		t.Error("expand all left the header closed")
	}
	if m = drive(m, press(':')); !m.enabled(1) {
		t.Error("collapse all is dim over an open header")
	}
	if m = drive(m, typed("c")[0], press(tea.KeyEnter)); m.stages[0].headed || m.rowsTop() != closed {
		t.Errorf("collapse all left the header open:\n%s", view(m))
	}
	// a click on the header off its gutter, or on its rule, does nothing
	before := m.stages[0].cursor
	if m = drive(m, down, testkit.Click(10, paneTop), testkit.Click(1, paneTop+1)); m.stages[0].headed ||
		m.stages[0].cursor != before+1 {
		t.Error("a click on the header's names or rule did something")
	}
}
