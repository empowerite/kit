// The panel beside the rows, driven as the person drives it: tab and / give it the keyboard, typing narrows the rows,
// space hides a field, shift with an arrow or a drag moves one, and a click acts on a control. A click on a row's
// gutter opens it.
package flow

import (
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/empowerite/kit/projects/tui/internal/grid"
	"github.com/empowerite/kit/projects/tui/internal/pane"
	"github.com/empowerite/kit/projects/tui/internal/shell"
	"github.com/empowerite/kit/projects/tui/internal/style"
	"github.com/empowerite/kit/projects/tui/internal/testkit"
)

// panelKeys is the keys bar while the panel's search has the keyboard, as it has when tab gives the panel the keyboard
// from the bar: the sidebar's keys, then the search's, which hides every key of the screen's and the bar's it types.
const panelKeys = "⇥·esc backlog ↑↓ move ⇞⇟ scroll │ type to search"

// searchKeys is the keys bar while the search has the keyboard, the sidebar hidden, from the bar.
const searchKeys = "⇥·esc backlog │ type to search"

// menuKeys is the keys bar while a menu is open.
var menuKeys = shell.Path{shell.MenuKeys()}.String()

// typed gives a key press for each rune of the text, as a terminal sends shell.Typing.
func typed(text string) []tea.Msg {
	var out []tea.Msg
	for _, r := range text {
		out = append(out, tea.KeyPressMsg{Code: r, Text: string(r)})
	}
	return out
}

// keys gives the backlog's rows' keys, as the panel lets them through, without the headings of their groups.
func keys(m Model) []string {
	var out []string
	for _, r := range m.rows(m.stage) {
		if !grid.Heads(r) {
			out = append(out, r.Key)
		}
	}
	return out
}

// drawn gives the stage's rows as the pane draws them, without the side.
func drawn(m Model) string {
	var out []string
	for _, l := range m.lines(m.stage) {
		out = append(out, l.text)
	}
	return strings.Join(out, "\n")
}

var tab = press(tea.KeyTab)

func TestTabGivesThePanelTheKeyboardAndTypingNarrowsTheRowsAndTheCount(t *testing.T) {
	m := loaded(t, options(), tab)
	if !m.inPanel || !m.sidebar || m.bar().Focused || lines(m)[23] != " "+panelKeys {
		t.Fatalf("tab: in panel %v, side %v, keys %q", m.inPanel, m.sidebar, lines(m)[23])
	}
	if !strings.Contains(view(m), grid.Search+" ▏") {
		t.Errorf("the search under the keyboard has no caret:\n%s", view(m))
	}
	m = drive(m, typed("NOUN")...)
	// the tab says how many of the rows loaded the panel lets through
	if got := keys(m); !slices.Equal(got, []string{"#46"}) || !shows(m, stageGlyphs["backlog"], "1/3") {
		t.Errorf("shell.Typing noun: rows %v\n%s", got, view(m))
	}
	// a number types into the search, and r does not refresh
	m = drive(m, typed("2r")...)
	if m.stage != 0 || m.stages[0].loading || !strings.Contains(view(m), grid.Search+" NOUN2r▏") {
		t.Errorf("a number or r in the search: stage %d\n%s", m.stage, view(m))
	}
	if !strings.Contains(view(m), "no row matches the panel") {
		t.Errorf("a search no row has:\n%s", view(m))
	}
	for range "NOUN2r" {
		m = drive(m, press(tea.KeyBackspace))
	}
	if len(keys(m)) != 3 || !shows(m, stageGlyphs["backlog"], "3") {
		t.Errorf("the search emptied: rows %v", keys(m))
	}
	// a key neither the panel nor the bar takes keeps the keyboard in the panel
	if m = drive(m, press(tea.KeyPgDown)); !m.inPanel {
		t.Error("pgdown left the panel")
	}
	// esc gives the keyboard back to the bar, the side still showing; tab again takes it, and esc from the rows
	// returns to them
	if m = drive(m, esc); m.inPanel || !m.sidebar || !m.bar().Focused {
		t.Errorf("esc: in panel %v, side %v", m.inPanel, m.sidebar)
	}
	m = drive(m, down, tab, tab)
	if m.inPanel || !m.inRows {
		t.Errorf("tab from the rows and back: in panel %v, in rows %v", m.inPanel, m.inRows)
	}
}

func TestSlashFindsAndANumberWithAltLeavesThePanelForItsStage(t *testing.T) {
	m := loaded(t, options(), down, down)
	m = drive(m, typed("/")...)
	if !m.inPanel || m.panel(0).Cursor != 0 || !m.inRows {
		t.Fatalf("/: in panel %v, cursor %d", m.inPanel, m.panel(0).Cursor)
	}
	// the band stays on the row under the cursor while the panel has the keyboard
	if !strings.HasPrefix(cursor(m), "#46") {
		t.Errorf("the cursor's row lost its band: %q", cursor(m))
	}
	m = drive(m, tea.KeyPressMsg{Code: '3', Mod: tea.ModAlt})
	if m.inPanel || m.stage != 2 || !m.bar().Focused {
		t.Errorf("alt-3 in the panel: in panel %v, stage %d", m.inPanel, m.stage)
	}
	// / from another control of the panel goes to the search, and what is typed next lands there
	m = drive(m, typed("2")...)
	m = drive(m, tab, down, down)
	m = drive(m, typed("/x")...)
	if p := m.panel(m.stage); !m.inPanel || p.Cursor != 0 || p.Query != "x" {
		t.Errorf("/ from a checkbox: in panel %v, cursor %d, search %q", m.inPanel, p.Cursor, p.Query)
	}
	m = drive(m, esc)
	// each stage keeps its own panel: the desk's search is empty
	if m.panel(2).Query != "" || m.panel(0).Query != "" {
		t.Errorf("the panels: %q, %q", m.panel(2).Query, m.panel(0).Query)
	}
}

// glyphAt gives the column of the search's glyph on the screen's first line.
func glyphAt(m Model) int {
	top := strings.Split(raw(m), "\n")[0]
	return ansi.StringWidth(top[:strings.Index(top, grid.Search)])
}

// The search stands on the screen's top right whether the sidebar shows or not: while it is hidden, right-aligned one
// cell short of the edge, and in use, its glyph over the rightmost tab's number that leaves ten cells to type in, until
// it is left empty. / or a click on it gives it the keyboard without showing the sidebar; down then has nothing to move
// to. With the sidebar shown, the search is the sidebar's, and down moves to the panel's first control.
func TestTheSearchStandsAloneWhileTheSidebarIsHidden(t *testing.T) {
	m := loaded(t, options(), down)
	idle := m.width - 1 - ansi.StringWidth(grid.Search+" search")
	if top := strings.Split(raw(m), "\n")[0]; !strings.HasSuffix(top, grid.Search+" search") ||
		ansi.StringWidth(top) != m.width-1 || glyphAt(m) != idle || m.bar().Width != 120 {
		t.Fatalf("the search is not right-aligned, or the bar is %d wide: %q", m.bar().Width, top)
	}
	for _, enter := range []tea.Msg{typed("/")[0], testkit.Click(idle, 0)} {
		m = drive(m, esc, enter)
		if !m.inPanel || m.sidebar || m.panel(0).Cursor != 0 || lines(m)[23] != " "+searchKeys {
			t.Fatalf("%v: in panel %v, sidebar %v, keys %q", enter, m.inPanel, m.sidebar, lines(m)[23])
		}
		numbers := m.bar().Numbers()
		at := glyphAt(m)
		if !slices.Contains(numbers, at) || m.width-1-(at+3) < shell.Typing ||
			slices.ContainsFunc(numbers, func(x int) bool { return x > at && m.width-1-(x+3) >= shell.Typing }) {
			t.Errorf("%v: the search's glyph is at %d, over none of %v that leaves room", enter, at, numbers)
		}
		if m = drive(m, down, press(tea.KeyPgDown)); m.panel(0).Cursor != 0 || m.sideOffset != 0 || m.sidebar {
			t.Errorf("down in the search alone: cursor %d, offset %d", m.panel(0).Cursor, m.sideOffset)
		}
	}
	at := glyphAt(m)
	if m = drive(m, typed("noun")...); !slices.Equal(keys(m), []string{"#46"}) ||
		!strings.Contains(strings.Split(raw(m), "\n")[0], grid.Search+" noun") {
		t.Errorf("shell.Typing in the search alone: %v\n%s", keys(m), view(m))
	}
	// a query left behind keeps the search where it is typed in; an empty one goes back to the edge
	if m = drive(m, esc); glyphAt(m) != at {
		t.Errorf("the search with a query moved to %d", glyphAt(m))
	}
	if m = drive(m, typed("/")[0], press(tea.KeyBackspace), press(tea.KeyBackspace), press(tea.KeyBackspace),
		press(tea.KeyBackspace), esc); glyphAt(m) != idle {
		t.Errorf("the empty search stayed at %d", glyphAt(m))
	}
	// on a screen too narrow for any number to leave the room, the search leaves it as far right as it can
	m = drive(m, tea.WindowSizeMsg{Width: 20, Height: 24}, typed("/")[0])
	if at := glyphAt(m); at != 20-1-shell.Typing-3 {
		t.Errorf("at 20 columns, the search's glyph is at %d", at)
	}
	m = drive(m, esc, tea.WindowSizeMsg{Width: 120, Height: 24})
	if m = drive(m, press('?'), typed("/")[0], down); m.panel(0).Cursor != 1 || !m.sidebar ||
		glyphAt(m) != m.width-m.sideWidth() {
		t.Errorf("down from the search with the sidebar shown: cursor %d, glyph at %d", m.panel(0).Cursor, glyphAt(m))
	}
}

// Space on a field's line in the editor hides it, and it stands under hidden, the keyboard with it; again shows it
// where it stood. Kit is asked for the caller's login once, whichever way the sidebar shows.
func TestSpaceInTheEditorHidesAFieldAndShowsItAgain(t *testing.T) {
	record := filepath.Join(t.TempDir(), "calls")
	t.Setenv("FAKE_FLOW_CALLS", record)
	// the search, then id, source, and assignees, under columns
	m := loaded(t, options(), tab, down, down, down, press(tea.KeySpace))
	if m.me != "ada" {
		t.Fatalf("the caller's login: %q", m.me)
	}
	side, _ := m.panel(0).Lines()
	at := slices.Index(side, style.Cursor(pane.Pad(grid.Unticked+" assignees", m.sideWidth()-1, false)))
	if strings.Contains(drawn(m), "jeffrey-agui") || at < 0 || ansi.Strip(side[at-1]) != grid.Unticked+" repository" || m.panel(0).Under() != "assignees" {
		t.Errorf("assignees hidden:\n%s", view(m))
	}
	if keys := lines(m)[23]; !strings.Contains(keys, "␣ show ⇧↑↓ move") {
		t.Errorf("the keys on a hidden field: %q", keys)
	}
	if m = drive(m, press(tea.KeySpace)); !strings.Contains(drawn(m), "jeffrey-agui") || shownNow(m)[2] != "assignees" {
		t.Errorf("assignees shown again: %q\n%s", shownNow(m), view(m))
	}
	m = drive(m, press('?'), press('?'), esc, tab)
	if n := strings.Count(strings.Join(calls(t, record), "\n"), "flow --me"); n != 1 {
		t.Errorf("kit flow --me ran %d times", n)
	}
}

func TestAClickOnThePanelActsAndAClickOnAGutterOpensItsRow(t *testing.T) {
	// tall enough that the panel and the legend end above the side's last line
	m := loaded(t, options(), tea.WindowSizeMsg{Width: 80, Height: 60}, press('?'))
	x := m.paneWidth() + 3
	// the side's lines: the search, a gap, group by, columns, then id, source, and assignees
	if m = drive(m, testkit.Click(x, 6), testkit.Release(x, 6)); !m.inPanel || m.panel(0).Field("assignees").Placement != grid.Hidden ||
		strings.Contains(drawn(m), Owner) {
		t.Fatalf("a click on assignees: in panel %v\n%s", m.inPanel, view(m))
	}
	if m = drive(m, testkit.Click(x, 1)); !m.inPanel || m.panel(0).Under() != "assignees" {
		t.Errorf("a click on a gap moved the panel's cursor: %d", m.panel(0).Cursor)
	}
	// a click on the gutter of a row's first line opens it, from the panel; again closes it. #45 stands under the
	// heading over every row, two cells in
	gutter := m.indent(0) + 1
	if m = drive(m, testkit.Click(gutter, m.rowsTop()+1)); m.inPanel || !m.inRows || m.stages[0].cursor != 1 ||
		!m.stages[0].open["#45"] {
		t.Errorf("a click on #45's gutter: in panel %v, cursor %d\n%s", m.inPanel, m.stages[0].cursor, view(m))
	}
	if m = drive(m, testkit.Click(gutter, m.rowsTop()+1)); m.stages[0].open["#45"] {
		t.Errorf("a second click on the gutter did not close the row")
	}
	// a click past the panel's lines, or on the side while it is hidden, acts on nothing
	before := view(m)
	if m = drive(m, testkit.Click(x, m.sideHeight()-1)); view(m) != before || m.inPanel {
		t.Errorf("a click under the side changed the screen")
	}
}

func TestThePanelWithoutStagesDoesNothing(t *testing.T) {
	m := loaded(t, Options{}, tab, testkit.Click(78, paneTop))
	if m.inPanel {
		t.Error("the panel took the keyboard without a stage")
	}
	if m = drive(m, press('?'), testkit.Click(m.paneWidth()+3, paneTop)); m.inPanel {
		t.Error("a click on the side took the keyboard without a stage")
	}
}

func TestAStageWithoutALegendShowsItsPanelAlone(t *testing.T) {
	m := loaded(t, Options{Stages: []string{"nonesuch"}}, press('?'))
	if side := m.sideLines(); len(side) != 1 || !strings.HasPrefix(side[0], grid.Search) {
		t.Errorf("the side of a stage kittui does not draw: %q", side)
	}
}

func TestTheSidebarScrollsSoTheLegendUnderThePanelCanBeRead(t *testing.T) {
	m := loaded(t, options(), tea.WindowSizeMsg{Width: 120, Height: 10}, press('?'))
	x, h := m.paneWidth()+3, m.sideHeight()
	if len(m.sideLines()) <= h || !strings.Contains(view(m), "… ") {
		t.Fatalf("the sidebar fits %d lines: %d", h, len(m.sideLines()))
	}
	wheel := func(b tea.MouseButton, x int) tea.Msg { return tea.MouseWheelMsg{X: x, Y: paneTop + 2, Button: b} }
	if m = drive(m, wheel(tea.MouseWheelDown, x)); m.sideOffset == 0 {
		t.Error("the wheel did not scroll the sidebar down")
	}
	if m = drive(m, wheel(tea.MouseWheelUp, x), wheel(tea.MouseWheelDown, 4)); m.sideOffset != 0 {
		t.Errorf("the wheel up, or over the rows: offset %d", m.sideOffset)
	}
	// a click on … more pages down, and another to the legend's end; one on … above, under the search, pages back up
	if m = drive(m, testkit.Click(x, h-1)); m.sideOffset == 0 || strings.Contains(view(m), Failed+" failed") {
		t.Errorf("a click on … more: offset %d\n%s", m.sideOffset, view(m))
	}
	for range 2 {
		m = drive(m, testkit.Click(x, h-1))
	}
	if !strings.Contains(view(m), Failed+" failed") {
		t.Errorf("a click on … more: offset %d\n%s", m.sideOffset, view(m))
	}
	if m = drive(m, testkit.Click(x, 1), testkit.Click(x, 1), testkit.Click(x, 1)); m.sideOffset != 0 {
		t.Errorf("a click on … above: offset %d", m.sideOffset)
	}
	// with the panel's keyboard, page down and up scroll it; down past the last control scrolls on, and up brings
	// the control under the keyboard back
	if m = drive(m, tab, press(tea.KeyPgDown)); m.sideOffset == 0 || !m.inPanel {
		t.Errorf("page down in the panel: offset %d", m.sideOffset)
	}
	if m = drive(m, press(tea.KeyPgUp)); m.sideOffset != 0 {
		t.Errorf("page up in the panel: offset %d", m.sideOffset)
	}
	m = drive(m, slices.Repeat([]tea.Msg{down}, 11)...)
	at := m.sideOffset
	if m = drive(m, down); m.sideOffset != at+1 {
		t.Errorf("down past the last control: offset %d, was %d", m.sideOffset, at)
	}
	if m = drive(m, slices.Repeat([]tea.Msg{up}, 11)...); m.sideOffset != 0 {
		t.Errorf("up to the search: offset %d", m.sideOffset)
	}
	// another stage starts at the sidebar's top
	m = drive(m, esc, press(tea.KeyPgDown), right)
	if m.sideOffset != 0 {
		t.Errorf("another stage: offset %d", m.sideOffset)
	}
}

// A title's glyph leads to the issue's page, so it stays though every row shares it; drawn without it, as a preset may
// pin, its words lead nowhere.
func TestAClickOnATitlesGlyphOpensTheIssuesPage(t *testing.T) {
	m := loaded(t, options())
	// the first issue stands under the heading over every row, the source said once
	y := m.rowsTop() + 1
	first := lines(m)[y]
	x := ansi.StringWidth(first[:strings.Index(first, Issue+" kit flow is")])
	browsed = nil
	if m = drive(m, testkit.Click(x, y)); len(browsed) != 1 ||
		browsed[0] != "https://github.com/empowerite/kit/issues/45" || !m.inRows || m.stages[0].open["#45"] {
		t.Fatalf("a click on the glyph: %v, in rows %v", browsed, m.inRows)
	}
	// a page the browser cannot open says why in place of the keys, until the next key
	m.find("backlog").issues[0].URL = "https://refuse.example/45"
	m.rebuild()
	m = drive(m, testkit.Click(x, y))
	if last := lines(m)[23]; !strings.Contains(last, "could not open https://refuse.example/45") || !m.Noticing() {
		t.Errorf("a page that did not open: %q", last)
	}
	if m = drive(m, down); m.notice != "" || strings.Contains(lines(m)[23], "could not open") {
		t.Errorf("the next key kept the notice: %q", lines(m)[23])
	}
	// pinned to its text, the title's first word is no button
	m = drive(m, up)
	m.setPanel(m.panel(0).Rendered("title", "text"))
	first = lines(m)[y]
	browsed = nil
	if m = drive(m, testkit.Click(ansi.StringWidth(first[:strings.Index(first, "kit flow is")]), y)); len(browsed) != 0 {
		t.Errorf("a click on the title's first word, drawn without its glyph: %v", browsed)
	}
}

func TestAColumnNoRowFillsHasNoCheckbox(t *testing.T) {
	m := loaded(t, options(), press('?'))
	if !strings.Contains(view(m), grid.Ticked+" branch") {
		t.Fatalf("backlog's branch has no checkbox while an issue has one:\n%s", view(m))
	}
	for k := range m.find("backlog").issues {
		m.find("backlog").issues[k].Branch = ""
	}
	m.rebuild()
	if strings.Contains(view(m), grid.Ticked+" branch") || !strings.Contains(view(m), grid.Ticked+" title") {
		t.Errorf("branch, which no issue has, keeps its checkbox:\n%s", view(m))
	}
}

func TestTheWheelScrollsTheRowsAndLeavesTheCursor(t *testing.T) {
	m := loaded(t, options(), tea.WindowSizeMsg{Width: 120, Height: 9}, right, right, right, down)
	turn := func(b tea.MouseButton) tea.Msg { return tea.MouseWheelMsg{X: 10, Y: paneTop + 1, Button: b} }
	if m = drive(m, turn(tea.MouseWheelDown)); m.find("worktrees").offset == 0 || m.find("worktrees").cursor != 0 {
		t.Fatalf("the wheel down: offset %d, cursor %d\n%s", m.find("worktrees").offset, m.find("worktrees").cursor, view(m))
	}
	if m = drive(m, turn(tea.MouseWheelUp), turn(tea.MouseWheelUp)); m.find("worktrees").offset != 0 {
		t.Errorf("the wheel up past the top: offset %d", m.find("worktrees").offset)
	}
	// on the bar, or with no stage, the wheel scrolls nothing
	if m = drive(m, tea.MouseWheelMsg{X: 10, Y: 1, Button: tea.MouseWheelDown}); m.find("worktrees").offset != 0 {
		t.Errorf("the wheel on the bar scrolled the rows")
	}
	loaded(t, Options{}, turn(tea.MouseWheelDown))
}

// In the panel, shift with up or down moves a column, and a drag of its line moves it as far as the mouse goes; the
// header and the rows follow. A press on a column's line toggles nothing until the release, and a drag never does.
func TestThePanelOrdersTheColumnsByKeyOrByDrag(t *testing.T) {
	m := loaded(t, options(), tea.WindowSizeMsg{Width: 140, Height: 30}, right, right, right, press('?'))
	x := m.paneWidth() + 3
	line := func(name string) int {
		for y, l := range lines(m) {
			if strings.HasSuffix(strings.TrimRight(l, " "), grid.Ticked+" "+name) {
				return y
			}
		}
		t.Fatalf("no %s in the panel:\n%s", name, view(m))
		return -1
	}
	// branch, the first column, moves down a place: ahead first, in the header and in the rows
	m = drive(m, tab, down, tea.KeyPressMsg{Code: tea.KeyDown, Mod: tea.ModShift})
	if l := lines(m); !strings.HasPrefix(strings.TrimSpace(l[paneTop]), "ahead") || line("branch") <= line("ahead") ||
		!strings.Contains(l[m.rowsTop()], "↑266 ↓0  1-the-first-release") {
		t.Fatalf("shift-down on branch:\n%s", view(m))
	}
	// a drag of state's line onto lines' puts state where lines was, toggling nothing
	from, to := line("state"), line("lines")
	m = drive(m, testkit.Click(x, from), testkit.Drag(x, to), testkit.Release(x, to))
	if line("state") != to || len(hiddenOf(m.panel(m.stage))) != 0 || !strings.Contains(lines(m)[paneTop], "state") ||
		strings.Index(lines(m)[paneTop], "state") > strings.Index(lines(m)[paneTop], "lines") {
		t.Errorf("a drag of state onto lines:\n%s", view(m))
	}
	// a press and a release on the same line toggle; a press released elsewhere does not
	at := line("ahead")
	hidden := func() bool { return m.panel(m.stage).Field("ahead").Placement == grid.Hidden }
	if m = drive(m, testkit.Click(x, at)); hidden() {
		t.Error("the press alone toggled ahead")
	}
	if m = drive(m, testkit.Release(x, at)); !hidden() {
		t.Error("the release did not toggle ahead")
	}
	if m = drive(m, testkit.Click(x, at), testkit.Release(x, at+5)); !hidden() {
		t.Error("a release off its line toggled ahead")
	}
	// a drag that never held a column, or over a line that draws none, moves nothing
	order := func() []string { return m.panel(m.stage).Arrange(m.panel(m.stage).Columns) }
	before := order()
	if m = drive(m, testkit.Drag(x, to), testkit.Release(x, to)); !slices.Equal(order(), before) {
		t.Error("a drag with nothing held moved a column")
	}
	m = drive(m, testkit.Click(x, line("state")), testkit.Drag(x, 0), testkit.Drag(2, 10), testkit.Release(2, 10))
	if !slices.Equal(order(), before) {
		t.Error("a drag over no column's line moved one")
	}
}

// Sort by, in the menu a click on a sortable column's name drops, sorts the rows by it, ▲ beside the name; a click on ▲
// turns it ▼;
// an option-click adds a column after it; the menu of a name that does not sort, as builds' commit, dims its sorts.
func TestSortByInAColumnsMenuSortsTheRows(t *testing.T) {
	m := loaded(t, options(), tea.WindowSizeMsg{Width: 160, Height: 24})
	unfold(&m) // the source, one tracker's on every row, a column
	at := func(name string) int { return strings.Index(lines(m)[paneTop], name) }
	order := func() string { return strings.Join(keys(m), " ") }
	if m = drive(m, testkit.Click(at("age"), paneTop), down, press(tea.KeyEnter)); !strings.Contains(lines(m)[paneTop], "age ▲") {
		t.Fatalf("a click on age: %s\n%s", order(), view(m))
	}
	if got := order(); got != "#45 #46 #7" {
		t.Errorf("by age, the youngest first: %s", got)
	}
	if m = drive(m, testkit.Click(at("age")+len("age "), paneTop)); !strings.Contains(lines(m)[paneTop], "age ▼") || order() != "#7 #46 #45" {
		t.Errorf("a click on its ▲, the oldest first: %s\n%s", order(), view(m))
	}
	option := tea.MouseClickMsg{X: at("source"), Y: paneTop, Button: tea.MouseLeft, Mod: tea.ModAlt}
	if m = drive(m, option); !strings.Contains(lines(m)[paneTop], "source ▲₂") || !strings.Contains(lines(m)[paneTop], "age ▼₁") {
		t.Errorf("an option-click on source:\n%s", view(m))
	}
	// shift, which a terminal often keeps for its own selection, adds nothing: the click drops the menu, as ever
	shift := tea.MouseClickMsg{X: at("source"), Y: paneTop, Button: tea.MouseLeft, Mod: tea.ModShift}
	if m = drive(m, shift); len(m.panel(m.stage).SortKeys()) != 2 || !m.colMenu.open {
		t.Errorf("a shift-click on source: %+v", m.panel(m.stage).SortKeys())
	}
	// builds' commit does not sort
	m = drive(m, esc)
	m = drive(m, slices.Repeat([]tea.Msg{right}, 5)...)
	if m = drive(m, testkit.Click(at("commit"), paneTop)); m.columnEnabled(lineUp) || m.colMenu.at <= lineDownNext ||
		m.stages[m.stage].name != "builds" {
		t.Errorf("commit's menu sorts: %+v", m.colMenu)
	}
}

// A groupable field moved up from the first column, by shift and up in the editor, groups the stage's rows by it under
// headings, which its tab does not count, each its value as the rows draw it and the field's name and the count dim; a
// field that cannot group, as id, stays the first column.
func TestAFieldMovedIntoTheGroupsGroupsTheRows(t *testing.T) {
	m := loaded(t, options(), tea.WindowSizeMsg{Width: 160, Height: 30}, right, press('?'), tab)
	count := m.count(m.stage)
	shiftUp := tea.KeyPressMsg{Code: tea.KeyUp, Mod: tea.ModShift}
	// down to state, the sixth column: id, source, assignees, age, change, state; up past the five before it
	m = drive(m, append(slices.Repeat([]tea.Msg{down}, 6), slices.Repeat([]tea.Msg{shiftUp}, 6)...)...)
	if v := view(m); !slices.Equal(m.panel(m.stage).Groups(), []string{"state"}) || !strings.Contains(v, "🏁 ready state  1") ||
		!strings.Contains(v, "🔨 draft state  1") || m.count(m.stage) != count || m.panel(m.stage).Under() != "state" {
		t.Fatalf("state up into the groups: %q, count %s\n%s", m.panel(m.stage).Groups(), m.count(m.stage), v)
	}
	if want := grid.Unfolded + " 🏁 ready " + style.Off("state") + "  " + style.Off("1"); !strings.Contains(
		m.View().Content, want) {
		t.Error("a group's heading does not draw its ▾ and value plain, and its column and count dim")
	}
	m = drive(m, down)
	if m = drive(m, shiftUp); m.panel(m.stage).Under() != "id" || len(m.panel(m.stage).Groups()) != 1 {
		t.Errorf("shift-up on id: %q", m.panel(m.stage).Groups())
	}
}

// In the side panel a field toggles only by space, or a click on its box: a click on its label takes the keyboard, and
// toggles nothing. The control under the keyboard is inverse across the panel but a cell.
func TestAClickOnAColumnsNameTogglesNothing(t *testing.T) {
	m := loaded(t, options(), tea.WindowSizeMsg{Width: 140, Height: 30}, press('?'))
	box := m.paneWidth() + 3
	m = drive(m, testkit.Click(box+3, 5), testkit.Release(box+3, 5))
	if p := m.panel(0); !slices.Equal(hiddenOf(p), []string{"organization", "repository"}) || !m.inPanel ||
		p.FieldAt(5) != "source" {
		t.Errorf("a click on source's name: hidden %v, in panel %v", hiddenOf(p), m.inPanel)
	}
	lines, _ := m.panel(0).Lines()
	if got := ansi.StringWidth(lines[5]); got != m.sideWidth()-1 || !strings.HasPrefix(lines[5], style.Cursor(grid.Ticked)[:4]) {
		t.Errorf("the control under the keyboard is %d cells inverse, not %d: %q", got, m.sideWidth()-1, lines[5])
	}
	if m = drive(m, testkit.Click(box, 5), testkit.Release(box, 5)); m.panel(0).Field("source").Placement != grid.Hidden {
		t.Error("a click on source's checkbox did not toggle it")
	}
}

// A field moved into the groups groups the rows by it, at the last depth, within the groups there are; a grouped
// column leaves the header and the rows, whose heading names its value, and the rows and the header stand in two cells
// a level, a click on a name finding its column there.
func TestGroupingNestsAndIndentsTheRows(t *testing.T) {
	m := loaded(t, options(), tea.WindowSizeMsg{Width: 160, Height: 30}, right, press('?'), tab)
	shiftUp := tea.KeyPressMsg{Code: tea.KeyUp, Mod: tea.ModShift}
	// down to source, up into the groups; then down to assignees, under it and id, and up after it
	m = drive(m, down, down, shiftUp, shiftUp)
	if got := m.panel(m.stage).Groups(); !slices.Equal(got, []string{"source"}) {
		t.Fatalf("source up into the groups: %q", got)
	}
	m = drive(m, down, down, shiftUp, shiftUp)
	if got := m.panel(m.stage).Groups(); !slices.Equal(got, []string{"source", "assignees"}) {
		t.Fatalf("assignees up into the groups: %q", got)
	}
	header := ansi.Cut(lines(m)[paneTop], 0, m.paneWidth())
	if strings.Contains(header, "source") || strings.Contains(header, "assignees") || !strings.HasPrefix(header, strings.Repeat(" ", 4+3)) {
		t.Errorf("the header of rows grouped two deep:\n%s", view(m))
	}
	first := slices.IndexFunc(lines(m), func(l string) bool { return strings.Contains(l, "#1") })
	if !strings.HasPrefix(lines(m)[first], "    ") || strings.Contains(lines(m)[first], GitHub) {
		t.Errorf("a row grouped two deep is not indented four cells, or keeps its source:\n%s", view(m))
	}
	// a click on a name in the indented header drops its menu, whose ▲ Ascending sorts by it
	if m = drive(m, esc, testkit.Click(strings.Index(lines(m)[paneTop], "age"), paneTop), down, press(tea.KeyEnter)); !strings.Contains(lines(m)[paneTop], "age ▲") {
		t.Errorf("a click on age in the indented header:\n%s", view(m))
	}
	// g on a column grouping alone stops it
	if p := m.panel(m.stage).Grouped("assignees", false).Grouped("assignees", false); len(p.Groups()) != 0 {
		t.Errorf("g twice: %q", p.Groups())
	}
}

// A click on a column's name with ctrl held, where the terminal sends it, groups the rows by it alone; with option too,
// within the groups there are. The same click, or a right click, on a group's heading stops grouping by its column
// alone, the other groups kept; on a row, it does nothing.
func TestACtrlClickGroupsAndOnAHeadingUngroups(t *testing.T) {
	m := loaded(t, options(), tea.WindowSizeMsg{Width: 160, Height: 30}, right)
	unfold(&m) // the source, one tracker's on every row, a column
	at := func(name string) int { return strings.Index(lines(m)[paneTop], name) }
	heading := func(text string) int {
		return slices.IndexFunc(lines(m), func(l string) bool {
			return strings.Contains(l, grid.Unfolded) && strings.Contains(l, text)
		})
	}
	group := func() []string { return m.panel(m.stage).Groups() }
	if m = drive(m, tea.MouseClickMsg{X: at("source"), Y: paneTop, Button: tea.MouseLeft, Mod: tea.ModCtrl}); !slices.Equal(group(), []string{"source"}) {
		t.Fatalf("a ctrl-click on source: %q\n%s", group(), view(m))
	}
	nest := tea.MouseClickMsg{X: at("assignees"), Y: paneTop, Button: tea.MouseLeft, Mod: tea.ModAlt | tea.ModCtrl}
	if m = drive(m, nest); !slices.Equal(group(), []string{"source", "assignees"}) {
		t.Fatalf("a ctrl-option-click on assignees: %q\n%s", group(), view(m))
	}
	before := group()
	if m = drive(m, tea.MouseClickMsg{X: 10, Y: heading("#1"), Button: tea.MouseRight}); !slices.Equal(group(), before) {
		t.Errorf("a right-click on a row: %q", group())
	}
	if m = drive(m, tea.MouseClickMsg{X: 10, Y: heading(" assignees"), Button: tea.MouseRight}); !slices.Equal(group(), []string{"source"}) {
		t.Fatalf("a right-click on an assignees heading: %q\n%s", group(), view(m))
	}
	if m = drive(m, tea.MouseClickMsg{X: 10, Y: heading(" source"), Button: tea.MouseLeft, Mod: tea.ModCtrl}); len(group()) != 0 {
		t.Errorf("a ctrl-click on a source heading: %q\n%s", group(), view(m))
	}
	// a plain click on a name drops its menu, and groups nothing
	if m = drive(m, testkit.Click(at("age"), paneTop)); len(group()) != 0 || !m.colMenu.open {
		t.Errorf("a click on age: group %q, sort %+v", group(), m.panel(m.stage).SortKeys())
	}
}

// Underway groups by change: the issues one pull request carries stand under its heading, and those without one under
// none.
func TestUnderwayGroupsByChange(t *testing.T) {
	m := loaded(t, options(), tea.WindowSizeMsg{Width: 160, Height: 30}, right)
	m.setPanel(m.panel(m.stage).Grouped("change", false))
	if got := m.panel(m.stage).Groups(); !slices.Equal(got, []string{"change"}) {
		t.Fatalf("grouped by change: %q", got)
	}
	headings := 0
	for _, r := range m.rows(m.stage) {
		if r.Aside && m.panel(m.stage).Heading(r) == "change" {
			headings++
		}
	}
	if headings == 0 || !strings.Contains(view(m), " change  ") {
		t.Errorf("no heading per change:\n%s", view(m))
	}
}

// A column whose value is the same on every row shown groups the rows on its own, as underway's source while one
// tracker holds every issue: the value is said once, in a heading over the rows, and the panel marks the column with a
// dim ▤. It follows the rows: a search that leaves one row ungroups it. The panel's box for folding, the last of its
// controls, stops it, so it is a column again.
func TestAColumnTheSameOnEveryRowGroupsTheRowsOnItsOwn(t *testing.T) {
	m := loaded(t, options(), tea.WindowSizeMsg{Width: 160, Height: 30}, right)
	heading := func(m Model) string {
		if rows := m.rows(m.stage); len(rows) > 0 && grid.Heads(rows[0]) {
			return ansi.Strip(rows[0].Cells[0].Values[0])
		}
		return ""
	}
	if got := heading(m); got != grid.Unfolded+" github source  4" || !slices.Equal(m.panel(m.stage).Auto, []string{"source"}) {
		t.Fatalf("underway's heading %q, auto %q:\n%s", got, m.panel(m.stage).Auto, view(m))
	}
	if lines, _ := m.panel(m.stage).Lines(); !slices.Contains(lines, style.Mark(grid.Ticked+" source")+" "+style.Off(grid.Grouping)) {
		t.Errorf("the panel does not mark source dim:\n%s", strings.Join(lines, "\n"))
	}
	// a search that leaves one row: the source is a column again
	p := m.panel(m.stage)
	p.Query = "kit commit"
	if m.setPanel(p); heading(m) != "" || len(m.panel(m.stage).Auto) > 0 || !strings.Contains(lines(m)[paneTop], "source") {
		t.Errorf("one row left still groups:\n%s", view(m))
	}
	p.Query = ""
	m.setPanel(p)
	// space on the box for folding stops it, through a refresh of the rows: the search, thirteen fields, then the box
	if m = drive(m, append([]tea.Msg{press(tea.KeyTab)}, slices.Repeat([]tea.Msg{down}, 14)...)...); m.panel(m.stage).Under() != "" {
		t.Fatalf("the panel's cursor: %d", m.panel(m.stage).Cursor)
	}
	if m = drive(m, press(tea.KeySpace)); heading(m) != "" || m.panel(m.stage).Fold ||
		len(m.panel(m.stage).Groups()) > 0 || !strings.Contains(lines(m)[paneTop], "source") {
		t.Errorf("space on the box: heading %q, group %q\n%s", heading(m), m.panel(m.stage).Groups(), view(m))
	}
	if m.rebuild(); heading(m) != "" {
		t.Errorf("the rows built again group by source")
	}
}

// A group's heading folds its rows away and unfolds them: left on it folds, right unfolds, the cursor staying on the
// heading, and a click anywhere on it does either; ▾ turns ▸, the keys say which, and the tab's count stays the rows
// the panel lets through, folded or not.
func TestAGroupsHeadingFoldsAndUnfoldsItsRows(t *testing.T) {
	m := loaded(t, options(), tea.WindowSizeMsg{Width: 160, Height: 30}, right)
	count, rows := m.count(m.stage), len(m.rows(m.stage))
	// down lands past the heading on a row; up puts the cursor on the heading
	m = drive(m, down, up)
	if m.stages[m.stage].cursor != 0 || !grid.Heads(m.rows(m.stage)[0]) {
		t.Fatalf("the cursor is not on the heading: %d", m.stages[m.stage].cursor)
	}
	if last := lines(m)[len(lines(m))-1]; !strings.Contains(last, "← collapse") {
		t.Errorf("the keys on an open heading: %q", last)
	}
	header := lines(m)[paneTop]
	m = drive(m, left)
	// the header stands as it was over the rows folded away, every column where it was
	if got := lines(m)[paneTop]; got != header || !strings.Contains(got, "title") {
		t.Errorf("the header with every row folded: %q, was %q", got, header)
	}
	heading := ansi.Strip(m.rows(m.stage)[0].Cells[0].Values[0])
	if len(m.rows(m.stage)) != 1 || !strings.HasPrefix(heading, grid.Folded+" ") || m.count(m.stage) != count ||
		m.stages[m.stage].cursor != 0 {
		t.Fatalf("left on the heading: %d rows, %q, count %s", len(m.rows(m.stage)), heading, m.count(m.stage))
	}
	if last := lines(m)[len(lines(m))-1]; !strings.Contains(last, "→ expand") {
		t.Errorf("the keys on a folded heading: %q", last)
	}
	if m = drive(m, right); len(m.rows(m.stage)) != rows {
		t.Fatalf("right on the heading: %d rows", len(m.rows(m.stage)))
	}
	// a click on the heading's ▾ folds it, and another, on its ▸, unfolds it
	if m = drive(m, testkit.Click(1, m.rowsTop())); len(m.rows(m.stage)) != 1 || m.headMenu.open {
		t.Fatalf("a click on the heading's ▾: %d rows, menu %v\n%s", len(m.rows(m.stage)), m.headMenu.open, view(m))
	}
	if m = drive(m, testkit.Click(1, m.rowsTop())); len(m.rows(m.stage)) != rows {
		t.Fatalf("a click on the heading's ▸: %d rows", len(m.rows(m.stage)))
	}
}

// The rows under a group's heading stand in from it, whether the rows group on their own or by a column the person
// chose: the heading's ▾ two cells left of its rows' ▸, and a heading nested under another two cells in from it.
func TestTheRowsUnderAHeadingStandInFromIt(t *testing.T) {
	m := loaded(t, options(), tea.WindowSizeMsg{Width: 120, Height: 30})
	// column gives the cell a line of the pane draws its first glyph of the kind at
	column := func(m Model, has, glyph string) int {
		k := slices.IndexFunc(lines(m), func(l string) bool { return strings.Contains(l, has) })
		if k < 0 {
			t.Fatalf("no line has %q:\n%s", has, view(m))
		}
		before, _, _ := strings.Cut(lines(m)[k], glyph)
		return ansi.StringWidth(before)
	}
	if h, r := column(m, "github source", grid.Unfolded), column(m, "#45", "▸"); r != h+2 {
		t.Errorf("grouped on its own, the heading's ▾ at %d, a row's ▸ at %d:\n%s", h, r, view(m))
	}
	unfold(&m)
	m.setPanel(m.panel(m.stage).Grouped("source", false).Grouped("assignees", true))
	outer, inner := column(m, "github source", grid.Unfolded), column(m, "jeffrey-aguilera assignees", grid.Unfolded)
	if r := column(m, "#45", "▸"); inner != outer+2 || r != inner+2 {
		t.Errorf("grouped by hand, the headings' ▾ at %d and %d, a row's ▸ at %d:\n%s", outer, inner, r, view(m))
	}
}

// popLineAt gives the screen line of the popover's line that has the text, after its glyph, and a column inside it.
func popLineAt(t *testing.T, m Model, text string) (x, y int) {
	t.Helper()
	box, top, left := m.popBox()
	for k, l := range box {
		if strings.Contains(ansi.Strip(l), " "+text+" ") {
			return left + 3, top + k
		}
	}
	t.Fatalf("the popover has no %q:\n%s", text, view(m))
	return -1, -1
}

// Enter on a field's line in the editor opens its popover, which has the keyboard: up and down move between the lines
// that act, past the headings; space takes a rendering, and toggles a value of the filter, or its mode; each applies
// at once, the rows, the editor's marks, and the preset line's chips following it. The sort is the column menu's, so
// the popover has none. Esc closes it, and
// the editor has the keyboard again. A click on a line acts on it, and a click off the box closes it.
func TestAFieldsPopoverFiltersAndDrawsIt(t *testing.T) {
	m := loaded(t, viewing(""), tab, down, down, down, press(tea.KeyEnter))
	if !m.pop.open || m.pop.field != "assignees" || !strings.Contains(view(m), "╭─ assignees ") {
		t.Fatalf("enter on assignees:\n%s", view(m))
	}
	// the keyboard starts on the filter's mode; there is no sort to take
	if v := view(m); strings.Contains(v, "ascending") || !strings.Contains(m.keys().String(), "␣ toggle") {
		t.Errorf("the popover's first line: %q\n%s", m.keys().String(), v)
	}
	// @me, the caller, is ada; checked, only her issue shows, and the preset line has its chip
	m = drive(m, down)
	if keys := m.keys().String(); keys != "esc close ↑↓ move ␣ toggle" || !strings.Contains(view(m), "@me ada") {
		t.Errorf("the popover's keys on @me: %q", keys)
	}
	if m = drive(m, press(tea.KeySpace)); !slices.Equal(keys(m), []string{"#7"}) ||
		!strings.Contains(presetText(m), "assignees: @me") {
		t.Errorf("@me: %v, %q", keys(m), presetText(m))
	}
	// the mode, above it, holds back the values checked
	if m = drive(m, up, press(tea.KeySpace)); !slices.Equal(keys(m), []string{"#45", "#46"}) ||
		!strings.Contains(presetText(m), "assignees: ¬@me") {
		t.Errorf("exclude: %v, %q", keys(m), presetText(m))
	}
	// auto keeps the glyph of the one row with a login, as no other row shares it, and says so; text, pinned, drops it
	if !strings.Contains(drawn(m), Owner) || !strings.Contains(view(m), "auto glyph+text") {
		t.Errorf("assignees drawn by auto:\n%s", view(m))
	}
	x, y := popLineAt(t, m, "text")
	if m = drive(m, testkit.Click(x, y)); m.panel(0).Field("assignees").Rendering != "text" || strings.Contains(drawn(m), Owner) {
		t.Errorf("a click on text: %+v\n%s", m.panel(0).Field("assignees"), view(m))
	}
	// esc closes it; the editor has the keyboard, and marks the field filtered
	if m = drive(m, esc); m.pop.open || !m.inPanel || !strings.Contains(view(m), "assignees ⊘") {
		t.Errorf("esc: open %v, in panel %v\n%s", m.pop.open, m.inPanel, view(m))
	}
	// a click off the box closes it, acting on nothing
	m = drive(m, press(tea.KeyEnter))
	before := m.panel(0).Fields
	if m = drive(m, testkit.Click(2, m.rowsTop())); m.pop.open || !reflect.DeepEqual(m.panel(0).Fields, before) {
		t.Errorf("a click off the popover: open %v", m.pop.open)
	}
	// an enum's values are the catalog's, with their glyphs, then none
	m = drive(m, append([]tea.Msg{esc, esc, right, tab}, slices.Repeat([]tea.Msg{down}, 6)...)...)
	m = drive(m, press(tea.KeyEnter))
	if v := view(m); m.pop.field != "state" || !strings.Contains(v, grid.Unticked+" 🔨 draft") ||
		!strings.Contains(v, grid.Unticked+" 🏁 ready") || !strings.Contains(v, grid.Unticked+" none") {
		t.Errorf("state's popover:\n%s", v)
	}
}

// Without the caller's login, @me stands alone, and checked lets no row through.
func TestTheCallerIsNoOneWhileKitCannotSayWhoItIs(t *testing.T) {
	t.Setenv("FAKE_FLOW_REFUSE", "me")
	// the popover opens on its filter's mode; @me is under it
	m := loaded(t, options(), tab, down, down, down, press(tea.KeyEnter), down)
	if m.me != "" || !strings.Contains(view(m), grid.Unticked+" @me ") || strings.Contains(view(m), "@me ada") {
		t.Fatalf("@me without a login: %q\n%s", m.me, view(m))
	}
	if m = drive(m, press(tea.KeySpace)); len(keys(m)) != 0 || !strings.Contains(view(m), "no row matches the panel") {
		t.Errorf("@me checked without a login: %v", keys(m))
	}
}

// A click on a column's name drops its menu down under the name, and sorts nothing; a second click on the name, off
// the menu, closes it. Options… opens the field's popover under the name. Where the terminal sends option with a click,
// the click adds the column after the other sort keys, and drops nothing.
func TestAClickOnAColumnsNameDropsItsMenu(t *testing.T) {
	m := loaded(t, options(), tea.WindowSizeMsg{Width: 160, Height: 30})
	unfold(&m)
	at := func(name string) int {
		l := lines(m)[paneTop]
		return ansi.StringWidth(l[:strings.Index(l, name)])
	}
	click := func(name string, mod tea.KeyMod) {
		m = drive(m, tea.MouseClickMsg{X: at(name), Y: paneTop, Button: tea.MouseLeft, Mod: mod})
	}
	sorts := func() []grid.SortKey { return m.panel(m.stage).SortKeys() }
	if click("age", 0); len(sorts()) != 0 || !m.colMenu.open || m.colMenu.column != "age" {
		t.Fatalf("a click on age: %+v, menu %+v", sorts(), m.colMenu)
	}
	// the box hangs around the name, its top edge drawing it where the header does, as ≡'s menu hangs from ≡
	if box, top, left := m.columnBox(); top != paneTop || !strings.Contains(ansi.Strip(box[0]), "╴age╶") ||
		ansi.StringWidth(ansi.Cut(lines(m)[paneTop], left, at("age"))) != at("age")-left ||
		ansi.Strip(ansi.Cut(lines(m)[paneTop], at("age"), at("age")+3)) != "age" {
		t.Errorf("the menu does not hang from the name: %d, %d\n%s", top, left, view(m))
	}
	if x, name := m.columnName("age"); name != "age" || x != at("age") {
		t.Errorf("age's name in the header: %q at %d, not %d", name, x, at("age"))
	}
	if click("age", 0); m.colMenu.open || len(sorts()) != 0 {
		t.Fatalf("a second click on age: %+v, menu %v", sorts(), m.colMenu.open)
	}
	click("age", 0)
	if m = drive(m, down, press(tea.KeyEnter)); !slices.Equal(sorts(), []grid.SortKey{{Column: "age"}}) {
		t.Fatalf("▲ Ascending on age: %+v", sorts())
	}
	click("age", 0)
	m = drive(m, typed("o")...)
	if m = drive(m, press(tea.KeyEnter)); !m.pop.open || m.pop.field != "age" || !strings.Contains(view(m), "╭─ age ") {
		t.Fatalf("Options… on age: popover %+v\n%s", m.pop, view(m))
	}
	if _, top, left := m.popBox(); top != paneTop || left+3 != at("age") {
		t.Errorf("the popover's title is not over the name: %d, %d", top, left)
	}
	m = drive(m, esc)
	if click("source", tea.ModAlt); len(sorts()) != 2 || m.colMenu.open {
		t.Errorf("an option-click on source: %+v, menu %v", sorts(), m.colMenu.open)
	}
}
