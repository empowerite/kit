// The selected tab's menu, driven by = and by clicks: it drops down under the menu glyph, and opens or closes every
// row of the stage.
package flow

import (
	"slices"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

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

func TestEqualsDropsTheMenuDownAndItOpensOrClosesEveryRow(t *testing.T) {
	m := loaded(t, options(), right, right, right, press('='))
	if !m.menu || lines(m)[23] != " "+quitKey+" · "+menuKeys || !strings.Contains(view(m), "│ "+Settings+" Settings…    │") ||
		!strings.Contains(view(m), "│ "+Expand+" Expand All   │") || !strings.Contains(view(m), "│ ─────────────── │") {
		t.Fatalf("= did not drop the menu down:\n%s", view(m))
	}
	// the menu's left edge is two cells before the menu glyph, its top edge on the bar's middle line, through the
	// glyph, half a line either side of it, all in the menu's color; the keyboard starts on expand all, collapse all,
	// with no row open, is dim, and the settings, last, do nothing yet
	glyph, cells := m.bar().MenuAt(), ansi.StringWidth(tabs.Menu)
	if box, left := m.menuBox(); left != glyph-2 || len(box) != len(menuItems)+2 || m.menuAt != 0 ||
		ansi.Cut(lines(m)[menuTop], glyph-1, glyph+cells+1) != "╴"+tabs.Menu+"╶" ||
		!strings.Contains(box[2], style.Off(" "+Collapse+" Collapse All ")) || box[0] != style.Menu(ansi.Strip(box[0])) ||
		!strings.Contains(box[4], style.Off(" "+Settings+" Settings…    ")) {
		t.Errorf("the menu's left edge %d, the glyph at %d, the keyboard on %d:\n%s", left, glyph, m.menuAt, view(m))
	}
	// a letter moves to the next enabled item that starts with it; one no enabled item starts with moves nothing; down
	// passes collapse all by while it is dim, and the rule and the settings
	for _, step := range []struct {
		letter string
		want   int
	}{{"c", 0}, {"s", 0}, {"E", 0}, {"z", 0}} {
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
	// now expand all is dim and the keyboard starts on collapse all, which space chooses
	if m = drive(m, press('=')); m.menuAt != 1 || m.enabled(0) {
		t.Errorf("with every row expanded the keyboard is on %d", m.menuAt)
	}
	if m = drive(m, up, down, press(tea.KeySpace)); openRows(m) != 0 || m.menu {
		t.Errorf("collapse all: open %d", openRows(m))
	}
	// esc and = close it, choosing nothing
	m = drive(m, press('='), press(tea.KeyEnter))
	for _, closing := range []tea.Msg{esc, press('=')} {
		if m = drive(m, press('='), closing); m.menu || openRows(m) != opened {
			t.Errorf("%v did not close the menu alone: menu %v", closing, m.menu)
		}
	}
	// enter on a dim item does nothing, and leaves the menu open
	m = drive(m, press('='))
	if m.pick(3); !m.menu {
		t.Error("the settings closed the menu")
	}
}

// On incoming, where no row hides anything, every item is dim: the keyboard is on none of them, moves to none, and
// enter does nothing.
func TestTheMenuOfAStageWithNothingToOpenHasNothingEnabled(t *testing.T) {
	m := loaded(t, options(), slices.Repeat([]tea.Msg{right}, 6)...)
	if m = drive(m, press('='), down, up, typed("e")[0], press(tea.KeyEnter)); !m.menu || m.menuAt != -1 {
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
	// a click on collapse all while it is dim, on the rule, or on the settings does nothing, and leaves the menu open
	for _, y := range []int{menuTop + 2, menuTop + 3, menuTop + 4} {
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

func TestEqualsInThePanelDropsTheMenuButIsTypedInTheSearch(t *testing.T) {
	m := loaded(t, options(), tab)
	if m = drive(m, typed("=")...); m.menu || m.panel(0).Query != "=" {
		t.Fatalf("= in the search: menu %v, search %q", m.menu, m.panel(0).Query)
	}
	if m = drive(m, down, press('=')); !m.menu || !m.inPanel {
		t.Errorf("= off the search: menu %v, in panel %v", m.menu, m.inPanel)
	}
	// with no stage, = drops nothing, and the menu's box keeps to the screen when the bar draws no glyph
	if m = loaded(t, Options{}, press('=')); m.menu {
		t.Error("= with no stage dropped a menu")
	}
	m = loaded(t, options(), tea.WindowSizeMsg{Width: 20, Height: 24}, press('='))
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
				m = drive(m, press('='))
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
			got := splice(line, "XY", at)
			if ansi.StringWidth(got) != ansi.StringWidth(line) || ansi.Cut(got, at, at+2) != "XY" {
				t.Errorf("splice(%q, XY, %d) = %q", line, at, got)
			}
		}
	}
}

// After Expand All, the rows have the keyboard, so left closes the row under the cursor even once the wheel has
// scrolled its first line away; and a click on that first line, pinned atop the pane, closes it too.
func TestAfterExpandAllTheRowsHaveTheKeyboardAndThePinnedLineCloses(t *testing.T) {
	wheel := tea.MouseWheelMsg{X: 10, Y: paneTop + 2, Button: tea.MouseWheelDown}
	size := tea.WindowSizeMsg{Width: 100, Height: 12}
	m := loaded(t, options(), size, right, right, right, press('='), press(tea.KeyEnter), wheel)
	if s := m.stages[m.stage]; !m.inRows || s.offset == 0 || !strings.HasPrefix(lines(m)[paneTop], " "+style.Opened) {
		t.Fatalf("after expand all and the wheel: in rows %v, offset %d:\n%s", m.inRows, s.offset, view(m))
	}
	opened := openRows(m)
	if m = drive(m, left); openRows(m) != opened-1 || m.stages[m.stage].name != "worktrees" {
		t.Errorf("left after expand all: open %d of %d, on %s", openRows(m), opened, m.stages[m.stage].name)
	}
	m = loaded(t, options(), size, right, right, right, press('='), press(tea.KeyEnter), wheel)
	if m = drive(m, testkit.Click(1, paneTop)); openRows(m) != opened-1 {
		t.Errorf("a click on the pinned line: open %d of %d:\n%s", openRows(m), opened, view(m))
	}
}
