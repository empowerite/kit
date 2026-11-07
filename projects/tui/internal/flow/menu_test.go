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
	for key, open := range m.stages[m.stage].open {
		if open && key != headerKey && !strings.HasPrefix(key, "group\x00") {
			n++
		}
	}
	return n
}

func TestColonDropsTheTabsMenuDownAndItOpensOrClosesEveryRow(t *testing.T) {
	m := loaded(t, options(), right, right, right, press(':'))
	if !m.tabMenu.Open || m.appMenu.Open || lines(m)[23] != " "+menuKeys ||
		!strings.Contains(view(m), "│ "+Expand+" Expand All ") || strings.Contains(view(m), "Settings") {
		t.Fatalf(": did not drop the tab's menu down:\n%s", view(m))
	}
	// the menu's left edge is two cells before the menu glyph, its top edge on the bar's middle line, through the
	// glyph, half a line either side of it, all in the menu's color; the keyboard starts on refresh, watch and maximize
	// after it, and collapse all, with no row open, is dim; Fields… is the presets' menu's, not this one's
	glyph, cells := m.bar().MenuAt(), ansi.StringWidth(tabs.Menu)
	if box, left := m.menuBox(); left != glyph-2 || len(box) != len(m.tabItems())+2 || m.tabMenu.At != 0 ||
		ansi.Cut(lines(m)[m.menuTop()], glyph-1, glyph+cells+1) != "╴"+tabs.Menu+"╶" ||
		!strings.Contains(box[1], " "+Reload+" Refresh") || !strings.Contains(box[2], " "+Unwatched+" Watch") ||
		!strings.Contains(box[3], " "+Maximize+"  Maximize") ||
		strings.Contains(view(m), FieldsIt) ||
		!strings.Contains(box[6], strings.TrimSuffix(style.Off("x"), "x\x1b[m")+" "+Collapse+" Collapse All ") ||
		box[0] != style.Menu(ansi.Strip(box[0])) {
		t.Errorf("the menu's left edge %d, the glyph at %d, the keyboard on %d:\n%s", left, glyph, m.tabMenu.At, view(m))
	}
	// a letter moves to the next enabled item that starts with it; one no enabled item starts with moves nothing, as c
	// does while collapse all, collapse others, and clear selection are dim; down passes the rule, and collapse all
	// while it is dim
	for _, step := range []struct {
		letter string
		want   int
	}{{"c", 0}, {"E", 4}, {"z", 4}} {
		if m = drive(m, tea.KeyPressMsg{Code: rune(step.letter[0]), Text: step.letter}); m.tabMenu.At != step.want {
			t.Errorf("%s: item %d, want %d", step.letter, m.tabMenu.At, step.want)
		}
	}
	if m = drive(m, down, up); m.tabMenu.At != 4 {
		t.Errorf("up and down moved to a dim item: %d", m.tabMenu.At)
	}
	// enter on expand all expands every worktree row that hides something; the menu closes
	if m = drive(m, press(tea.KeyEnter)); m.tabMenu.Open || openRows(m) == 0 || strings.Contains(view(m), "Expand All") {
		t.Fatalf("expand all: menu %v, open %d", m.tabMenu.Open, openRows(m))
	}
	opened := openRows(m)
	// now expand all is dim, and c finds collapse all, which space chooses; ; drops the menu down too
	if m = drive(m, press(';'), typed("c")[0]); !m.tabMenu.Open || m.tabMenu.At != 5 || m.enabled(4) {
		t.Errorf("with every row expanded, c is on %d", m.tabMenu.At)
	}
	// down passes the dim collapse others and the rule to select all; up passes them, the dim expand all, and the
	// rule, to watch
	if m = drive(m, down); m.tabMenu.At != 8 {
		t.Errorf("down from collapse all: item %d", m.tabMenu.At)
	}
	if m = drive(m, up, up); m.tabMenu.At != 2 {
		t.Errorf("up past expand all: item %d", m.tabMenu.At)
	}
	if m = drive(m, down, press(tea.KeySpace)); openRows(m) != 0 || m.tabMenu.Open {
		t.Errorf("collapse all: open %d", openRows(m))
	}
	m = drive(m, press(':'), typed("e")[0], press(tea.KeyEnter))
	// esc, :, and ; close it, choosing nothing, and = does not
	for _, closing := range []tea.Msg{esc, press(':'), press(';')} {
		if m = drive(m, press(':'), closing); m.tabMenu.Open || openRows(m) != opened {
			t.Errorf("%v did not close the menu alone: menu %v", closing, m.tabMenu.Open)
		}
	}
	if m = drive(m, press(':'), press('=')); !m.tabMenu.Open || m.appMenu.Open {
		t.Errorf("= in the tab's menu: menu %v, app %v", m.tabMenu.Open, m.appMenu.Open)
	}
}

// = or a click on ≡ before the title drops the app's menu down, internal/shell's, with the picture's own first:
// refresh all, which runs every stage again, the forge read afresh. The flow's own screen is dim; another screen asks
// kittui to show it in the picture's place, and quit ends the picture with 0.
func TestEqualsDropsTheAppsMenuDownToRefreshAnotherScreenOrQuit(t *testing.T) {
	m := loaded(t, options(), press('='))
	if !m.appMenu.Open || m.tabMenu.Open || !strings.HasPrefix(lines(m)[shell.Top], "╭╴"+shell.App+"╶") ||
		!strings.Contains(view(m), "│ "+Reload+" Refresh All ") || !strings.Contains(view(m), "Welcome") ||
		lines(m)[23] != " "+menuKeys {
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
	m = drive(m, testkit.Click(3, shell.Top+1+slices.IndexFunc(m.appMenu.Items(), func(it shell.Item) bool {
		return it.Name == "Refresh All"
	})))
	got := calls(t, record)
	if m.appMenu.Open || !slices.Contains(got, "flow underway --json --fresh") || !slices.Contains(got, "flow builds --json") ||
		len(got) < len(eight) {
		t.Errorf("refresh all ran %q", got)
	}
	// welcome asks for the welcome, the picture kept as it is; quit ends it, not interrupted
	welcome := shell.Top + 1 + slices.IndexFunc(m.appMenu.Items(), func(it shell.Item) bool { return it.Name == "Welcome" })
	_, msgs := testkit.Drive(m, func(msg tea.Msg) bool { _, ok := msg.(shell.SwitchMsg); return ok },
		testkit.Click(shell.At, shell.Top), testkit.Click(3, welcome))
	if !slices.Contains(msgs, tea.Msg(shell.SwitchMsg{Screen: shell.Welcome})) {
		t.Errorf("welcome asked %v", msgs)
	}
	after, msgs := testkit.Drive(m, nil, press('='), typed("q")[0], press(tea.KeyEnter))
	if !slices.ContainsFunc(msgs, func(msg tea.Msg) bool { _, ok := msg.(tea.QuitMsg); return ok }) || after.interrupted {
		t.Errorf("quit did not end the picture: %v", msgs)
	}
}

// On incoming, where no row hides anything, the keyboard starts on maximize, as everywhere, and expand all and collapse
// all, dim, are never under it.
func TestTheMenuOfAStageWithNothingToOpenHasRefreshAlone(t *testing.T) {
	m := loaded(t, options(), slices.Repeat([]tea.Msg{right}, 6)...)
	if m = drive(m, press(':'), down, up, typed("e")[0]); !m.tabMenu.Open || m.tabMenu.At != 0 || m.enabled(4) || m.enabled(5) {
		t.Errorf("on %s the keyboard is on %d:\n%s", m.stages[m.stage].name, m.tabMenu.At, view(m))
	}
	// nothing is drawn under the keyboard on a dim item, even when the keyboard was left on one the rows no longer
	// let do anything
	for _, at := range []int{-1, 4} {
		m.tabMenu.At = at
		if box, _ := m.menuBox(); strings.Contains(strings.Join(box, "\n"), style.Cursor(" ")[:4]) {
			t.Errorf("with the keyboard on %d, a dim item is drawn under it: %q", at, box)
		}
	}
}

func TestAClickOnTheMenuGlyphDropsItDownAndAClickChoosesOrCloses(t *testing.T) {
	m := loaded(t, options(), right, right, right)
	glyph := testkit.Click(m.bar().MenuAt(), m.bar().Row+1)
	if m = drive(m, glyph); !m.tabMenu.Open {
		t.Fatalf("a click on the menu glyph:\n%s", view(m))
	}
	_, left := m.menuBox()
	// a click on the rule, or on collapse all while it is dim, does nothing, and leaves the menu open
	for _, y := range []int{m.menuTop() + 4, m.menuTop() + 6} {
		if m = drive(m, testkit.Click(left+2, y)); !m.tabMenu.Open || openRows(m) != 0 {
			t.Errorf("a click on line %d: menu %v, open %d", y, m.tabMenu.Open, openRows(m))
		}
	}
	if m = drive(m, testkit.Click(left+2, m.menuTop()+5)); m.tabMenu.Open || openRows(m) == 0 {
		t.Errorf("a click on expand all: menu %v, open %d", m.tabMenu.Open, openRows(m))
	}
	if m = drive(m, glyph, testkit.Click(left+2, m.menuTop()+6)); m.tabMenu.Open || openRows(m) != 0 {
		t.Errorf("a click on collapse all: menu %v, open %d", m.tabMenu.Open, openRows(m))
	}
	// a click off the menu closes it, and does nothing else
	before := m.stages[m.stage].cursor
	if m = drive(m, glyph, testkit.Click(2, m.top()+8)); m.tabMenu.Open || m.stages[m.stage].cursor != before {
		t.Errorf("a click off the menu: menu %v", m.tabMenu.Open)
	}
}

func TestTheMenusKeysInThePanelDropThemButAreTypedInTheSearch(t *testing.T) {
	m := loaded(t, options(), typed("/")[0])
	if m = drive(m, typed("=:;")...); m.tabMenu.Open || m.panel(0).Query != "=:;" {
		t.Fatalf("the menus' keys in the search: menu %v, search %q", m.tabMenu.Open, m.panel(0).Query)
	}
	for _, c := range []struct {
		key rune
		app bool
	}{{'=', true}, {':', false}, {';', false}} {
		if m = drive(m, typed("/")[0], esc, press(c.key)); m.tabMenu.Open == c.app || m.appMenu.Open != c.app || m.inPanel {
			t.Errorf("%c after the search: menu %v, app %v, in panel %v", c.key, m.tabMenu.Open, m.appMenu.Open, m.inPanel)
		}
		m = drive(m, esc)
	}
	// with no stage, neither drops, and no glyph stands before the title; the tab's menu's box keeps to the screen
	// when the bar draws no glyph
	if m = loaded(t, Options{}, press(':')); m.tabMenu.Open || !strings.Contains(view(m), shell.App) {
		t.Error("a menu with no stage")
	}
	m = loaded(t, options(), tea.WindowSizeMsg{Width: 20, Height: 24}, press(':'))
	if _, left := m.menuBox(); m.bar().MenuAt() >= 0 || left < 0 || !m.tabMenu.Open {
		t.Errorf("a bar too narrow for the glyph: glyph %d, left %d", m.bar().MenuAt(), left)
	}
}

// On the rightmost tab, a box from the glyph would run past the rows: the glyph takes the top edge's cell two before
// the right corner instead, or a nearer one when the rows end there; at every width, over the legend or not, the
// whole box shows, the edge half a line either side of the glyph.
// What the box does not cover stays in its columns, a two-cell glyph it cuts in half giving way to a space.
func TestTheMenuOnTheRightmostTabEndsUnderTheGlyphAndLeavesTheRestInPlace(t *testing.T) {
	for _, width := range []int{60, 80, 97, 110, 131, 174, 200} {
		for _, legend := range []bool{false, true} {
			m := loaded(t, options(), tea.WindowSizeMsg{Width: width, Height: 24})
			if legend {
				m = drive(m, press('?'))
			}
			for i := range m.stages {
				m = drive(m, press(':'))
				box, left := m.menuBox()
				glyph, end := m.bar().MenuAt(), left+ansi.StringWidth(box[0])
				drawn, cells := lines(m), ansi.StringWidth(tabs.Menu)
				after := ansi.Cut(drawn[m.menuTop()], glyph+cells, glyph+cells+1)
				if end > m.paneWidth() || glyph-left < 2 || ansi.Cut(drawn[m.menuTop()], glyph-1, glyph+cells) != "╴"+tabs.Menu ||
					!(after == "╶" || glyph+cells == end-1 && after == "╮" || glyph+cells == end) ||
					ansi.Cut(drawn[m.menuTop()+len(box)-1], left, end) != ansi.Strip(box[len(box)-1]) {
					t.Errorf("at %d, legend %v, tab %d: the box spans %d to %d, the glyph at %d:\n%s", width, legend, i,
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
	wheel := tea.MouseWheelMsg{X: 10, Y: Model{}.top() + 2, Button: tea.MouseWheelDown}
	size := tea.WindowSizeMsg{Width: 100, Height: 12}
	m := loaded(t, options(), size, right, right, right, press(':'), typed("e")[0], press(tea.KeyEnter), wheel)
	if s := m.stages[m.stage]; !m.inRows || s.offset == 0 || !strings.HasPrefix(lines(m)[m.rowsTop()], "  "+style.Opened) {
		t.Fatalf("after expand all and the wheel: in rows %v, offset %d:\n%s", m.inRows, s.offset, view(m))
	}
	opened := openRows(m)
	if m = drive(m, left); openRows(m) != opened-1 || m.stages[m.stage].name != "worktrees" {
		t.Errorf("left after expand all: open %d of %d, on %s", openRows(m), opened, m.stages[m.stage].name)
	}
	m = loaded(t, options(), size, right, right, right, press(':'), typed("e")[0], press(tea.KeyEnter), wheel)
	if m = drive(m, testkit.Click(1, m.rowsTop())); openRows(m) != opened-1 {
		t.Errorf("a click on the pinned line: open %d of %d:\n%s", openRows(m), opened, view(m))
	}
}

// The names of the stage's columns stand over its rows, as every stage with columns draws them, and the tab's menu
// has no item to hide them: a header that never scrolls, a thin dim line under it, and no row of its own, so a click
// on it chooses nothing. Incoming, whose rows are no columns, has none.
func TestTheHeaderStandsTheColumnsNamesOverTheRows(t *testing.T) {
	m := loaded(t, options(), tea.WindowSizeMsg{Width: 120, Height: 10}, right, right, right)
	if slices.ContainsFunc(m.tabItems(), func(i shell.Item) bool { return strings.Contains(i.Name, "Header") }) {
		t.Errorf("the tab's menu offers the headers: %+v", m.tabItems())
	}
	drawn := lines(m)
	if !strings.Contains(drawn[m.top()], "branch") || !strings.Contains(drawn[m.top()], "uncommitted") ||
		!strings.HasPrefix(drawn[m.rowsTop()-1], "    ───") ||
		!strings.Contains(m.View().Content, strings.TrimSuffix(style.Off("─"), "─\x1b[m")+"───") || m.rowsTop() <= m.top() {
		t.Fatalf("the headers:\n%s", view(m))
	}
	// the header stays while the rows scroll under it
	header := drawn[m.top():m.rowsTop()]
	m = drive(m, down, down, down, down, down, down)
	if got := lines(m)[m.top():m.rowsTop()]; !slices.Equal(got, header) || m.stages[m.stage].offset == 0 {
		t.Errorf("the header scrolled with the rows, offset %d:\n%s", m.stages[m.stage].offset, view(m))
	}
	// a click on the header chooses no row
	before := m.stages[m.stage].cursor
	if m = drive(m, testkit.Click(5, m.top())); m.stages[m.stage].cursor != before {
		t.Errorf("a click on the header moved the cursor to %d", m.stages[m.stage].cursor)
	}
	// a column the panel hides has no name over it
	m.setPanel(m.panel(m.stage).Placed("uncommitted", grid.Hidden))
	if strings.Contains(lines(m)[m.top()], "uncommitted") {
		t.Errorf("a hidden column's name:\n%s", view(m))
	}
	m = loaded(t, options(), slices.Repeat([]tea.Msg{right}, 6)...)
	if m.stages[m.stage].name != "incoming" || m.rowsTop() != m.top() {
		t.Errorf("%s has a header:\n%s", m.stages[m.stage].name, view(m))
	}
}

// A header whose names take more than a line shows its first alone, with ▸ before it, until it is opened: by a click
// on its gutter, as a row's, or by Expand All, which Collapse All undoes; the rows then start under every line of it.
func TestAHeaderTallerThanALineOpensAsARowDoes(t *testing.T) {
	backlog := catalog["backlog"]
	defer func(was []field) { backlog.Fields = was }(backlog.Fields)
	backlog.Fields = slices.Clone(backlog.Fields)
	backlog.Fields[len(backlog.Fields)-1].Label = "the title of the issue, as its tracker gives it, in full"
	m := loaded(t, options(), tea.WindowSizeMsg{Width: 60, Height: 24})
	// the header stands in as the rows do, under the headings the view grouped them by as it loaded
	closed, in := m.rowsTop(), strings.Repeat(" ", m.indent(0))
	if lines, hides := m.header(); !hides || !strings.HasPrefix(lines[0], in+"  "+style.Off("▸")) || closed != m.top()+2 {
		t.Fatalf("the closed header:\n%s", view(m))
	}
	if m = drive(m, testkit.Click(len(in)+2, m.top())); !m.stages[0].open[headerKey] || m.rowsTop() <= closed ||
		!strings.HasPrefix(lines(m)[m.top()], in+"  ▾") || m.inRows {
		t.Fatalf("a click on its gutter did not open it:\n%s", view(m))
	}
	if m = drive(m, testkit.Click(len(in)+2, m.top())); m.stages[0].open[headerKey] {
		t.Error("a second click did not close it")
	}
	// Expand All opens it with the rows, and Collapse All closes it, the group over the rows folded with them
	m = drive(m, press(':'))
	if !m.enabled(item(m, "Expand All")) {
		t.Errorf("expand all %v over a closed header", m.enabled(item(m, "Expand All")))
	}
	if m = drive(m, typed("e")[0], press(tea.KeyEnter)); !m.stages[0].open[headerKey] {
		t.Error("expand all left the header closed")
	}
	if m = drive(m, press(':')); !m.enabled(item(m, "Collapse All")) {
		t.Error("collapse all is dim over an open header")
	}
	if m = drive(m, typed("c")[0], typed("c")[0], press(tea.KeyEnter)); m.stages[0].open[headerKey] || m.rowsTop() != closed {
		t.Errorf("collapse all left the header open:\n%s", view(m))
	}
	// a click on the header off its gutter, or on its rule, does nothing; the group open again, down has a row to go to
	m.stages[0].open = grid.Outline{}
	m.rebuild()
	before := m.stages[0].cursor
	if m = drive(m, down, testkit.Click(12, m.top()), testkit.Click(len(in)+1, m.top()+1)); m.stages[0].open[headerKey] ||
		m.stages[0].cursor != before+1 {
		t.Error("a click on the header's names or rule did something")
	}
}

// In the rows, space selects the row under the cursor, ▣ in its gutter in the tab's group's color, and again unselects
// it; shift with up or down
// selects as it moves; ctrl+a selects every row shown, and again none. A click puts the cursor on a row and selects
// nothing, and a heading is no row to select. The keys bar counts the rows selected; the tab's menu opens each
// selected row's page, and clears the selection.
func TestRowsAreSelectedSeveralAtOnceForTheTabsMenuToActOn(t *testing.T) {
	// the panel not folding, the rows stand under no heading
	m := loaded(t, options(), tea.WindowSizeMsg{Width: 140, Height: 24})
	unfold(&m)
	m = drive(m, down, press(tea.KeySpace))
	gutter := func(k int) string { return string([]rune(lines(m)[m.rowsTop()+k])[0]) }
	if m.selected() != 1 || gutter(0) != grid.Chosen || !strings.Contains(lines(m)[m.rowsTop()-1], " 1 selected ") {
		t.Fatalf("space on the first row:\n%s", view(m))
	}
	if marked, _ := style.In(stageColors["backlog"]); !strings.Contains(m.View().Content, marked(grid.Chosen)) {
		t.Error("the selection's mark is not in the tab's group's color")
	}
	if m = drive(m, press(tea.KeySpace)); m.selected() != 0 || gutter(0) != " " {
		t.Errorf("space again: %d selected", m.selected())
	}
	m = drive(m, tea.KeyPressMsg{Code: tea.KeyDown, Mod: tea.ModShift})
	if m.selected() != 2 || m.stages[0].cursor != 1 || gutter(1) != grid.Chosen {
		t.Errorf("shift-down: %d selected, cursor %d\n%s", m.selected(), m.stages[0].cursor, view(m))
	}
	if m = drive(m, tea.KeyPressMsg{Code: tea.KeyUp, Mod: tea.ModShift}); m.selected() != 2 || m.stages[0].cursor != 0 {
		t.Errorf("shift-up: %d selected, cursor %d", m.selected(), m.stages[0].cursor)
	}
	ctrlA := tea.KeyPressMsg{Code: 'a', Mod: tea.ModCtrl}
	if m = drive(m, ctrlA); m.selected() != 3 {
		t.Errorf("ctrl+a: %d selected", m.selected())
	}
	if m = drive(m, ctrlA); m.selected() != 0 {
		t.Errorf("ctrl+a again: %d selected", m.selected())
	}
	// a click moves the cursor and selects nothing
	if m = drive(m, testkit.Click(10, m.rowsTop()+2)); m.selected() != 0 || m.stages[0].cursor != 2 {
		t.Errorf("a click: %d selected, cursor %d", m.selected(), m.stages[0].cursor)
	}
	// the tab's menu: open selected and clear selection are dim with nothing selected
	on := func(name string) bool {
		return m.enabled(slices.IndexFunc(m.tabItems(), func(i shell.Item) bool { return i.Name == name }))
	}
	if m = drive(m, press(':')); on("Open Selected") || !on("Select All") || on("Clear Selection") {
		t.Errorf("open %v, select all %v, clear %v, with nothing selected", on("Open Selected"), on("Select All"),
			on("Clear Selection"))
	}
	// select all selects every row, and is then dim
	if m = drive(m, typed("s")[0], typed("s")[0], press(tea.KeyEnter)); m.selected() != 3 || on("Select All") {
		t.Errorf("select all: %d selected", m.selected())
	}
	m = drive(m, press(':'))
	m.tabMenu.At = item(m, "Clear Selection")
	m = drive(m, press(tea.KeyEnter), press(tea.KeySpace), up, press(tea.KeySpace))
	browsed = nil
	m = drive(m, press(':'), typed("o")[0], press(tea.KeyEnter))
	if len(browsed) != 2 || !strings.HasSuffix(browsed[0], "/issues/46") && !strings.HasSuffix(browsed[1], "/issues/46") {
		t.Errorf("open selected opened %q", browsed)
	}
	m = drive(m, press(':'))
	m.tabMenu.At = item(m, "Clear Selection")
	if m = drive(m, press(tea.KeyEnter)); m.selected() != 0 || m.tabMenu.Open {
		t.Errorf("clear selection: %d selected", m.selected())
	}
	// space on a group's heading selects every row of the group, those folded away too, its heading marked ▣, and
	// again none; with some selected, ◩ marks it. Off the rows, space selects nothing
	m.setPanel(grid.Panel{Groupable: groupableOf("backlog")}.Grouped("source", false))
	m.stages[0].cursor = 0
	m = drive(m, left)
	if m = drive(m, press(tea.KeySpace)); m.selected() != 3 || m.groupMarks(0)[m.rows(0)[0].Key] != grid.Chosen ||
		!strings.HasPrefix(ansi.Strip(lines(m)[m.rowsTop()]), grid.Chosen+" "+grid.Folded) {
		t.Errorf("space on a folded heading selected %d:\n%s", m.selected(), view(m))
	}
	m = drive(m, right, down, down, press(tea.KeySpace), up, up)
	if m.selected() != 2 || m.groupMarks(0)[m.rows(0)[0].Key] != grid.Partly {
		t.Errorf("one row unselected: %d selected, the mark %q", m.selected(), m.groupMarks(0)[m.rows(0)[0].Key])
	}
	if m = drive(m, press(tea.KeySpace), press(tea.KeySpace)); m.selected() != 0 || len(m.groupMarks(0)) != 0 {
		t.Errorf("space on the heading twice: %d selected", m.selected())
	}
	if m = drive(m, esc, press(tea.KeySpace), ctrlA); m.selected() != 3 || m.inRows {
		t.Errorf("off the rows: %d selected", m.selected())
	}
}

// With rows selected, the tab's menu expands and collapses the selected rows alone, and collapses the others.
func TestExpandAndCollapseActOnTheSelectedRowsWhenThereAreAny(t *testing.T) {
	m := loaded(t, options(), right, right, right, down, press(tea.KeySpace), press(':'))
	if m.tabItems()[4].Name != "Expand Selected" || m.tabItems()[5].Name != "Collapse Selected" || m.enabled(5) {
		t.Fatalf("the items with a row selected: %+v", m.tabItems()[4:6])
	}
	if m = drive(m, typed("e")[0], press(tea.KeyEnter)); openRows(m) != 1 || !m.stages[m.stage].open[m.rows(m.stage)[0].Key] {
		t.Errorf("expand selected opened %d", openRows(m))
	}
	if m = drive(m, press(':')); m.enabled(4) || !m.enabled(5) {
		t.Errorf("with the selected row open: expand %v, collapse %v", m.enabled(4), m.enabled(5))
	}
	if m = drive(m, typed("c")[0], press(tea.KeyEnter)); openRows(m) != 0 {
		t.Errorf("collapse selected left %d open", openRows(m))
	}
	// collapse others closes the rows not selected, the selected ones left open; with none of those open, it is dim
	s, rows := &m.stages[m.stage], m.rows(m.stage)
	s.open[rows[0].Key], s.open[rows[1].Key] = true, true
	if m = drive(m, press(':')); m.tabItems()[6].Name != "Collapse Others" || !m.enabled(6) {
		t.Fatalf("collapse others with a row not selected open: %+v, %v", m.tabItems()[6], m.enabled(6))
	}
	if m = drive(m, typed("c")[0], typed("c")[0], press(tea.KeyEnter)); openRows(m) != 1 ||
		!m.stages[m.stage].open[rows[0].Key] || m.tabMenu.Open {
		t.Errorf("collapse others left %d open, the selected one %v", openRows(m), m.stages[m.stage].open[rows[0].Key])
	}
	if m = drive(m, press(':')); m.enabled(item(m, "Collapse Others")) {
		t.Error("collapse others with no row not selected open is not dim")
	}
}
