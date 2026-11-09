// Maximize, the first item of the tab's menu, gives the rows the bar's lines but one: the title names the tab, its
// count, and its :, a line in the tab's color runs under it, and the rows' header follows. Restore puts the bar back.
package flow

import (
	"path/filepath"
	"slices"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/empowerite/kit/projects/tui/internal/shell"
	"github.com/empowerite/kit/projects/tui/internal/style"
	"github.com/empowerite/kit/projects/tui/internal/tabs"
	"github.com/empowerite/kit/projects/tui/internal/testkit"
)

// maximize gives the screen with the tab's menu's Maximize chosen, or Restore once maximized: the keyboard moves down
// to it from Refresh, where the menu opens, past Watch.
func maximize(t *testing.T, m Model) Model {
	t.Helper()
	m = drive(m, press(':'))
	for range 3 {
		if name := m.tabItems()[m.tabMenu.At].Name; name == "Maximize" || name == "Restore" {
			return drive(m, press(tea.KeyEnter))
		}
		m = drive(m, press(tea.KeyDown))
	}
	t.Fatalf("no Maximize or Restore in the tab's menu's first section: %+v", m.tabItems()[:4])
	return m
}

func TestTheTabsMenusFirstSectionIsRefreshWatchAndMaximize(t *testing.T) {
	m := drive(loaded(t, viewing("")), press(':'))
	if got := m.tabItems()[:4]; got[0].Name != "Refresh" || got[1].Name != WatchIt || got[2].Glyph != Maximize ||
		got[2].Name != "Maximize" || got[3].Name != "" || !m.enabled(2) || m.tabMenu.At != 0 {
		t.Fatalf("the tab's menu's first section: %+v, the keyboard on %d", got, m.tabMenu.At)
	}
	if ansi.StringWidth(Maximize) != 1 || ansi.StringWidth(Restore) != 1 || ansi.StringWidth(Banner) != 1 {
		t.Errorf("a glyph is not one cell: %q %q %q", Maximize, Restore, Banner)
	}
	if m = maximize(t, loaded(t, viewing(""))); !m.maximized || m.tabMenu.Open {
		t.Fatalf("Maximize: maximized %v, menu %v", m.maximized, m.tabMenu.Open)
	}
	if m = drive(m, press(':')); m.tabItems()[2].Glyph != Restore || m.tabItems()[2].Name != "Restore" {
		t.Errorf("maximized, the tab's menu's third item is %+v", m.tabItems()[2])
	}
}

func TestMaximizeGivesTheRowsTwoLines(t *testing.T) {
	// twelve lines tall, the backlog's heading and its three rows overflow the pane
	size := tea.WindowSizeMsg{Width: 120, Height: 12}
	normal := loaded(t, viewing(""), size)
	m := maximize(t, normal)
	if m.rowsHeight() != normal.rowsHeight()+2 || m.rowsTop() != normal.rowsTop()-2 || m.top() != 2 ||
		normal.top() != 1+tabs.Height {
		t.Fatalf("rows %d from %d, maximized %d from %d", normal.rowsHeight(), normal.rowsTop(), m.rowsHeight(), m.rowsTop())
	}
	// the lines under the control line that are not blank: the frozen heading, the header, and the rows
	rowLines := func(m Model) int {
		n := 0
		texts, _ := m.laid()
		for _, l := range texts {
			if strings.TrimSpace(ansi.Strip(l)) != "" {
				n++
			}
		}
		return n
	}
	if len(lines(m)) != len(lines(normal)) || rowLines(m) != rowLines(normal)+2 {
		t.Errorf("%d lines of rows, %d maximized; %d lines, %d maximized:\n%s\n%s", rowLines(normal), rowLines(m),
			len(lines(normal)), len(lines(m)), view(normal), view(m))
	}
	// the control line follows the line under the title
	if !strings.Contains(lines(m)[2], Presets) || !strings.Contains(lines(m)[2], "source") {
		t.Errorf("the header is not under the line:\n%s", view(m))
	}
}

func TestMaximizeTitleNamesTheTabAndItsCount(t *testing.T) {
	m := maximize(t, loaded(t, viewing("")))
	want := "  " + shell.App + " kit flow · " + shell.Books + " empowerite/kit · 🎫 backlog 3 " + tabs.Menu
	if got := lines(m)[0]; got != want {
		t.Errorf("the title:\n%q\nwant\n%q", got, want)
	}
	// the tab's name in its color, as the bar draws it
	mark, _ := style.In(stageColors["backlog"])
	if !strings.Contains(strings.Split(m.View().Content, "\n")[0], mark("backlog")) {
		t.Errorf("the tab's name is not in its color: %q", strings.Split(m.View().Content, "\n")[0])
	}
	// a number selects a tab, still maximized, and the title names it
	if m = drive(m, press('2')); !m.maximized || !strings.HasSuffix(lines(m)[0], " · 🛫 underway 4 "+tabs.Menu) {
		t.Errorf("after 2, the title: %q", lines(m)[0])
	}
	// the search stays at its right
	if !strings.Contains(raw(m), "🔎 search") {
		t.Errorf("the search went:\n%s", raw(m))
	}
}

func TestTheRuleIsTheTabsColor(t *testing.T) {
	m := maximize(t, loaded(t, viewing("")))
	rule := func(stage string) string {
		mark, _ := style.In(stageColors[stage])
		return mark(strings.Repeat(Banner, m.paneWidth()))
	}
	if got := strings.Split(m.View().Content, "\n")[1]; got != rule("backlog") {
		t.Errorf("line 2:\n%q\nwant\n%q", got, rule("backlog"))
	}
	if m = drive(m, press('3')); strings.Split(m.View().Content, "\n")[1] != rule("desk") {
		t.Errorf("desk's line: %q", strings.Split(m.View().Content, "\n")[1])
	}
}

func TestRestorePutsTheBarBack(t *testing.T) {
	// the rows scrolled, the cursor on the last row, one row selected
	m := loaded(t, viewing(""), tea.WindowSizeMsg{Width: 120, Height: 10}, down, down, press(tea.KeySpace), down,
		down)
	before, cursor, offset := m.View().Content, m.stages[m.stage].Cursor, m.stages[m.stage].Offset
	if offset == 0 || m.selected() != 1 {
		t.Fatalf("the rows are not scrolled (%d) with one selected (%d):\n%s", offset, m.selected(), raw(m))
	}
	if m = maximize(t, m); m.View().Content == before || m.stages[m.stage].Cursor != cursor || !m.inRows {
		t.Fatalf("maximized, the cursor %d, rows %v:\n%s", m.stages[m.stage].Cursor, m.inRows, raw(m))
	}
	if m = maximize(t, m); m.maximized || m.View().Content != before || m.stages[m.stage].Cursor != cursor ||
		m.stages[m.stage].Offset != offset {
		t.Errorf("restored:\n%s\nbefore:\n%s", raw(m), ansi.Strip(before))
	}
}

func TestClicksFollowTheTop(t *testing.T) {
	m := maximize(t, loaded(t, viewing("")))
	header := lines(m)[m.headerTop()]
	// a click on a column's name drops its menu
	if m = drive(m, testkit.Click(strings.Index(header, "age"), m.headerTop())); !m.colMenu.Open || m.colMenu.Column != "age" {
		t.Fatalf("a click on age: %+v\n%s", m.colMenu, raw(m))
	}
	m = drive(m, esc)
	// a click on a row puts the cursor on it
	y := slices.IndexFunc(lines(m), func(l string) bool { return strings.Contains(l, "#46") })
	if m = drive(m, testkit.Click(10, y)); !m.inRows || m.rowLine(m.stages[m.stage].Cursor) != y {
		t.Fatalf("a click on #46, line %d: rows %v, cursor %d\n%s", y, m.inRows, m.stages[m.stage].Cursor, raw(m))
	}
	// a click on the line under the title selects no tab, as the bar there would
	if m = drive(m, testkit.Click(25, 1)); m.stage != 0 || !m.inRows {
		t.Errorf("a click on the line: stage %d, rows %v", m.stage, m.inRows)
	}
	// a click on the title's : drops the tab's menu, hanging from it, and a click on its first item restores
	glyph := m.tabMenuAt()
	if glyph < 0 || ansi.Cut(lines(m)[0], glyph, glyph+1) != tabs.Menu {
		t.Fatalf("the title's : at %d: %q", glyph, lines(m)[0])
	}
	// so does a click on the tab's first cell in the title
	if m := drive(m, testkit.Click(ansi.StringWidth(m.title())-ansi.StringWidth(m.bar().Label()), 0)); !m.tabMenu.Open {
		t.Errorf("a click on the title's tab:\n%s", raw(m))
	}
	if m = drive(m, testkit.Click(glyph, 0)); !m.tabMenu.Open {
		t.Fatalf("a click on the title's ::\n%s", raw(m))
	}
	if box, left := m.menuBox(); left != glyph-2 || m.menuTop() != 0 ||
		ansi.Cut(lines(m)[0], glyph-1, glyph+2) != "╴"+tabs.Menu+"╶" ||
		!strings.Contains(ansi.Strip(box[3]), Restore+"  Restore") {
		t.Errorf("the menu hangs at %d from %d:\n%s", left, glyph, raw(m))
	}
	_, left := m.menuBox()
	// Restore is the menu's third line, after Refresh and Watch, under its top edge on the title's line
	if m = drive(m, testkit.Click(left+3, 3)); m.maximized || m.tabMenu.Open {
		t.Errorf("a click on Restore: maximized %v, menu %v", m.maximized, m.tabMenu.Open)
	}
}

// Where the title is clipped short of the tab's :, : still drops the menu, against the rows' right edge, and so does a
// click on what the title shows of the tab.
func TestANarrowTitleHidesTheKebabAndColonStillDropsTheMenu(t *testing.T) {
	m := maximize(t, loaded(t, viewing(""), tea.WindowSizeMsg{Width: 60, Height: 24}))
	if m.tabMenuAt() != -1 || strings.Contains(lines(m)[0], tabs.Menu) {
		t.Fatalf("the : at %d: %q", m.tabMenuAt(), lines(m)[0])
	}
	if start := ansi.StringWidth(m.title()) - ansi.StringWidth(m.bar().Label()); start >= m.titleWidth()-1 {
		t.Fatalf("the tab starts at %d, past the title's %d", start, m.titleWidth())
	}
	if m := drive(m, testkit.Click(m.titleWidth()-1, 0)); !m.tabMenu.Open {
		t.Errorf("a click on what the clipped title shows of the tab dropped no menu")
	}
	if m := drive(m, testkit.Click(m.titleWidth(), 0)); m.tabMenu.Open {
		t.Errorf("a click past the clipped title dropped the menu")
	}
	if m = drive(m, press(':')); !m.tabMenu.Open {
		t.Fatalf(": did not drop the menu")
	}
	if box, left := m.menuBox(); left+ansi.StringWidth(box[0]) != m.paneWidth() {
		t.Errorf("the menu at %d, %d wide, the rows %d", left, ansi.StringWidth(box[0]), m.paneWidth())
	}
}

func TestMaximizedTheKeysDropTheBars(t *testing.T) {
	m := maximize(t, loaded(t, viewing("")))
	if bar := lines(m)[len(lines(m))-1]; strings.Contains(bar, "←→") || !strings.Contains(bar, "1-8") ||
		!strings.Contains(bar, ": tab menu") || !strings.Contains(bar, "↓ preset") {
		t.Errorf("the keys bar: %q", bar)
	}
	// left and right move along no bar, and change nothing
	before := raw(m)
	if m = drive(m, right, right, left); m.stage != 0 || raw(m) != before {
		t.Errorf("right and left moved to stage %d:\n%s", m.stage, raw(m))
	}
	// down reaches the preset line, and up from it leaves it
	if m = drive(m, down); !m.onPreset {
		t.Errorf("down did not reach the preset line")
	}
	if m = drive(m, up); m.onPreset || m.inRows {
		t.Errorf("up from the preset line: preset %v, rows %v", m.onPreset, m.inRows)
	}
}

func TestGoldenMaximized(t *testing.T) {
	m := maximize(t, loaded(t, viewing("")))
	testkit.Golden(t, filepath.Join("testdata", "backlog-max.golden"), raw(m))
	m = drive(m, testkit.Click(m.tabMenuAt(), 0))
	testkit.Golden(t, filepath.Join("testdata", "backlog-max-menu.golden"), raw(m))
}

// A tab's menu line shows its key at its right, and the key chooses the line with the menu closed, as m maximizes and
// restores, and with it open, before a letter moves to the next line that starts with it.
func TestATabMenuLinesKeyChoosesItOpenOrClosed(t *testing.T) {
	m := loaded(t, viewing(""))
	if got := m.tabItems()[:3]; got[0].Keys != "r" || got[1].Keys != "w" || got[2].Keys != "m" {
		t.Fatalf("the tab's menu's keys: %+v", got)
	}
	if m = drive(m, testkit.Typed("m")...); !m.maximized || m.tabMenu.Open {
		t.Fatalf("m with the menu closed: maximized %v", m.maximized)
	}
	if m = drive(m, press(':'), press('m')); m.maximized || m.tabMenu.Open {
		t.Errorf("m with the menu open: maximized %v, menu open %v", m.maximized, m.tabMenu.Open)
	}
}
