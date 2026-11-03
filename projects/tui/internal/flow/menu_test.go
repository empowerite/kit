// The selected tab's menu, driven by = and by clicks: it drops down under the menu glyph, and opens or closes every
// row of the stage.
package flow

import (
	"slices"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

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
	if !m.menu || lines(m)[23] != " "+menuKeys || !strings.Contains(view(m), "│ open all  │") ||
		!strings.Contains(view(m), "│ close all │") {
		t.Fatalf("= did not drop the menu down:\n%s", view(m))
	}
	// the menu's left edge is under the menu glyph, its top on the bar's last line, right under the glyph
	if box, left := m.menuBox(); left != m.bar().MenuAt() || len(box) != len(menuItems)+2 ||
		!strings.Contains(lines(m)[menuTop], "╭") {
		t.Errorf("the menu's left edge %d, the glyph at %d:\n%s", left, m.bar().MenuAt(), view(m))
	}
	// a letter moves to the next item that starts with it, round to the first; one no item starts with moves nothing
	for _, step := range []struct {
		letter string
		want   int
	}{{"c", 1}, {"o", 0}, {"O", 0}, {"z", 0}} {
		if m = drive(m, tea.KeyPressMsg{Code: rune(step.letter[0]), Text: step.letter}); m.menuAt != step.want {
			t.Errorf("%s: item %d, want %d", step.letter, m.menuAt, step.want)
		}
	}
	// enter on open all opens every worktree row that hides something; the menu closes
	if m = drive(m, press(tea.KeyEnter)); m.menu || openRows(m) == 0 || strings.Contains(view(m), "open all") {
		t.Fatalf("open all: menu %v, open %d", m.menu, openRows(m))
	}
	opened := openRows(m)
	// down past the last item stops there, and space chooses close all
	if m = drive(m, press('='), down, down, up, down, press(tea.KeySpace)); openRows(m) != 0 || m.menu {
		t.Errorf("close all: open %d", openRows(m))
	}
	// esc and = close it, choosing nothing; up past the first stops there
	m = drive(m, press('='), up, press(tea.KeyEnter))
	if openRows(m) != opened {
		t.Errorf("up past the first item: open %d", openRows(m))
	}
	for _, closing := range []tea.Msg{esc, press('=')} {
		if m = drive(m, press('='), closing); m.menu || openRows(m) != opened {
			t.Errorf("%v did not close the menu alone: menu %v", closing, m.menu)
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
	if m = drive(m, testkit.Click(left+2, menuTop+1)); m.menu || openRows(m) == 0 {
		t.Errorf("a click on open all: menu %v, open %d", m.menu, openRows(m))
	}
	if m = drive(m, glyph, testkit.Click(left+2, menuTop+2)); m.menu || openRows(m) != 0 {
		t.Errorf("a click on close all: menu %v, open %d", m.menu, openRows(m))
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

// On the rightmost tab, a box under the glyph would run off the screen: its right edge is under the glyph instead.
// What the box does not cover stays in its columns, a two-cell glyph it cuts in half giving way to a space.
func TestTheMenuOnTheRightmostTabEndsUnderTheGlyphAndLeavesTheRestInPlace(t *testing.T) {
	m := loaded(t, options(), slices.Repeat([]tea.Msg{right}, 7)...)
	m = drive(m, press('='))
	box, left := m.menuBox()
	if glyph := m.bar().MenuAt(); left+ansi.StringWidth(box[0])-1 != glyph || left+ansi.StringWidth(box[0]) > m.width {
		t.Errorf("on the last tab the box spans %d to %d, the glyph at %d", left, left+ansi.StringWidth(box[0])-1, glyph)
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
