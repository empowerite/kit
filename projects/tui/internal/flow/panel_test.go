// The panel beside the rows, driven as the person drives it: tab and / give it the keyboard, shell.Typing narrows the rows,
// space hides a column or picks whose rows show, and a click acts on a control. A click on a row's gutter opens it.
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
	"github.com/empowerite/kit/projects/tui/internal/testkit"
)

// panelKeys is the keys bar while the panel's search has the keyboard, as it has when tab gives the panel the keyboard.
const panelKeys = shell.SearchKeys

// typed gives a key press for each rune of the text, as a terminal sends shell.Typing.
func typed(text string) []tea.Msg {
	var out []tea.Msg
	for _, r := range text {
		out = append(out, tea.KeyPressMsg{Code: r, Text: string(r)})
	}
	return out
}

// keys gives the backlog's rows' keys, as the panel lets them through.
func keys(m Model) []string {
	var out []string
	for _, r := range m.rows(m.stage) {
		out = append(out, r.Key)
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
		if !m.inPanel || m.sidebar || m.panel(0).Cursor != 0 || lines(m)[23] != " "+shell.SearchKeys {
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
	m = drive(m, tea.WindowSizeMsg{Width: 24, Height: 24}, typed("/")[0])
	if at := glyphAt(m); at != 24-1-shell.Typing-3 {
		t.Errorf("at 24 columns, the search's glyph is at %d", at)
	}
	m = drive(m, esc, tea.WindowSizeMsg{Width: 120, Height: 24})
	if m = drive(m, press('?'), typed("/")[0], down); m.panel(0).Cursor != 1 || !m.sidebar ||
		glyphAt(m) != m.width-m.sideWidth() {
		t.Errorf("down from the search with the sidebar shown: cursor %d, glyph at %d", m.panel(0).Cursor, glyphAt(m))
	}
}

func TestThePanelHidesAColumnAndShowsTheCallersRowsOrEveryoneElses(t *testing.T) {
	record := filepath.Join(t.TempDir(), "calls")
	t.Setenv("FAKE_FLOW_CALLS", record)
	m := loaded(t, options(), tab, down, down, down, press(tea.KeySpace))
	if m.me != "ada" {
		t.Fatalf("the caller's login: %q", m.me)
	}
	if strings.Contains(drawn(m), Owner) || !strings.Contains(view(m), grid.Unticked+" assignees") {
		t.Errorf("assignees hidden:\n%s", view(m))
	}
	// the controls: the search, six columns, then all, mine, and others
	for range 5 {
		m = drive(m, down)
	}
	if m = drive(m, press(tea.KeySpace)); !slices.Equal(keys(m), []string{"#7"}) {
		t.Errorf("mine: %v\n%s", keys(m), view(m))
	}
	if m = drive(m, down, press(tea.KeyEnter)); !slices.Equal(keys(m), []string{"#45", "#46"}) {
		t.Errorf("others: %v", keys(m))
	}
	// underway's rows belong to the pull request's author too, so ada's pull request on #8 is mine; mine is past
	// underway's eleven named columns
	m = drive(m, esc, right)
	m = drive(m, append([]tea.Msg{tab}, slices.Repeat([]tea.Msg{down}, 13)...)...)
	if m = drive(m, press(tea.KeySpace)); !slices.Equal(keys(m), []string{"#8 #11", "#31 #12"}) {
		t.Errorf("underway, mine: %v\n%s", keys(m), view(m))
	}
	// kit is asked once, whichever way the side shows
	m = drive(m, press('?'), press('?'), esc, tab)
	if n := strings.Count(strings.Join(calls(t, record), "\n"), "flow --me"); n != 1 {
		t.Errorf("kit flow --me ran %d times", n)
	}
}

func TestMineAndOthersStayDimWhenKitCannotSayWhoTheCallerIs(t *testing.T) {
	t.Setenv("FAKE_FLOW_REFUSE", "me")
	m := loaded(t, options(), tab)
	if m.me != "" || !strings.Contains(m.View().Content, style.Off(grid.Unpicked+" mine")) {
		t.Fatalf("mine without a login: %q\n%s", m.me, view(m))
	}
	m = drive(m, slices.Repeat([]tea.Msg{down}, 7)...)
	if m = drive(m, press(tea.KeySpace)); len(keys(m)) != 3 {
		t.Errorf("mine was picked without a login: %v", keys(m))
	}
}

func TestAClickOnThePanelActsAndAClickOnAGutterOpensItsRow(t *testing.T) {
	m := loaded(t, options(), press('?'))
	x := m.paneWidth() + 3
	// the side's lines: the search, a gap, columns, then number, source, and assignees
	if m = drive(m, testkit.Click(x, 5), testkit.Release(x, 5)); !m.inPanel || !m.panel(0).Hidden["assignees"] ||
		strings.Contains(drawn(m), Owner) {
		t.Fatalf("a click on assignees: in panel %v\n%s", m.inPanel, view(m))
	}
	if m = drive(m, testkit.Click(x, 1)); !m.inPanel || m.panel(0).Cursor != 3 {
		t.Errorf("a click on a gap moved the panel's cursor: %d", m.panel(0).Cursor)
	}
	// a click on the gutter of a row's first line opens it, from the panel; again closes it
	if m = drive(m, testkit.Click(1, m.rowsTop())); m.inPanel || !m.inRows || m.stages[0].cursor != 0 ||
		!m.stages[0].open["#45"] {
		t.Errorf("a click on #45's gutter: in panel %v, cursor %d\n%s", m.inPanel, m.stages[0].cursor, view(m))
	}
	if m = drive(m, testkit.Click(1, m.rowsTop())); m.stages[0].open["#45"] {
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
	m = drive(m, slices.Repeat([]tea.Msg{down}, 9)...)
	at := m.sideOffset
	if m = drive(m, down); m.sideOffset != at+1 {
		t.Errorf("down past the last control: offset %d, was %d", m.sideOffset, at)
	}
	if m = drive(m, slices.Repeat([]tea.Msg{up}, 9)...); m.sideOffset != 0 {
		t.Errorf("up to the search: offset %d", m.sideOffset)
	}
	// another stage starts at the sidebar's top
	m = drive(m, esc, press(tea.KeyPgDown), right)
	if m.sideOffset != 0 {
		t.Errorf("another stage: offset %d", m.sideOffset)
	}
}

func TestAClickOnATitlesGlyphOpensTheIssuesPage(t *testing.T) {
	m := loaded(t, options())
	first := lines(m)[m.rowsTop()]
	x := ansi.StringWidth(first[:strings.Index(first, Issue)])
	browsed = nil
	if m = drive(m, testkit.Click(x, m.rowsTop())); len(browsed) != 1 ||
		browsed[0] != "https://github.com/empowerite/kit/issues/45" || !m.inRows || m.stages[0].open["#45"] {
		t.Fatalf("a click on the glyph: %v, in rows %v", browsed, m.inRows)
	}
	// a page the browser cannot open says why in place of the keys, until the next key
	m.find("backlog").issues[0].URL = "https://refuse.example/45"
	m.rebuild()
	m = drive(m, testkit.Click(x, m.rowsTop()))
	if last := lines(m)[23]; !strings.Contains(last, "could not open https://refuse.example/45") {
		t.Errorf("a page that did not open: %q", last)
	}
	if m = drive(m, down); m.notice != "" || strings.Contains(lines(m)[23], "could not open") {
		t.Errorf("the next key kept the notice: %q", lines(m)[23])
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
		!strings.Contains(l[m.rowsTop()], "↑266 ↓0  🌿 1-the-first-release") {
		t.Fatalf("shift-down on branch:\n%s", view(m))
	}
	// a drag of state's line onto lines' puts state where lines was, toggling nothing
	from, to := line("state"), line("lines")
	m = drive(m, testkit.Click(x, from), testkit.Drag(x, to), testkit.Release(x, to))
	if line("state") != to || len(m.panel(m.stage).Hidden) != 0 || !strings.Contains(lines(m)[paneTop], "state") ||
		strings.Index(lines(m)[paneTop], "state") > strings.Index(lines(m)[paneTop], "lines") {
		t.Errorf("a drag of state onto lines:\n%s", view(m))
	}
	// a press and a release on the same line toggle; a press released elsewhere does not
	at := line("ahead")
	if m = drive(m, testkit.Click(x, at)); m.panel(m.stage).Hidden["ahead"] {
		t.Error("the press alone toggled ahead")
	}
	if m = drive(m, testkit.Release(x, at)); !m.panel(m.stage).Hidden["ahead"] {
		t.Error("the release did not toggle ahead")
	}
	if m = drive(m, testkit.Click(x, at), testkit.Release(x, at+5)); !m.panel(m.stage).Hidden["ahead"] {
		t.Error("a release off its line toggled ahead")
	}
	// a drag that never held a column, or over a line that draws none, moves nothing
	before := m.panel(m.stage).Order
	if m = drive(m, testkit.Drag(x, to), testkit.Release(x, to)); !slices.Equal(m.panel(m.stage).Order, before) {
		t.Error("a drag with nothing held moved a column")
	}
	m = drive(m, testkit.Click(x, line("state")), testkit.Drag(x, 0), testkit.Drag(2, 10), testkit.Release(2, 10))
	if !slices.Equal(m.panel(m.stage).Order, before) {
		t.Error("a drag over no column's line moved one")
	}
}

// A click on a sortable column's name in the header sorts the rows by it, ▲ beside the name, and again ▼; a shift-click
// adds a column after it; a click on a name that does not sort, as builds' commit, changes nothing.
func TestAClickOnAColumnsNameSortsTheRows(t *testing.T) {
	m := loaded(t, options(), tea.WindowSizeMsg{Width: 160, Height: 24})
	at := func(name string) int { return strings.Index(lines(m)[paneTop], name) }
	order := func() string { return strings.Join(keys(m), " ") }
	if m = drive(m, testkit.Click(at("age"), paneTop)); !strings.Contains(lines(m)[paneTop], "age ▲") {
		t.Fatalf("a click on age: %s\n%s", order(), view(m))
	}
	if got := order(); got != "#45 #46 #7" {
		t.Errorf("by age, the youngest first: %s", got)
	}
	if m = drive(m, testkit.Click(at("age"), paneTop)); !strings.Contains(lines(m)[paneTop], "age ▼") || order() != "#7 #46 #45" {
		t.Errorf("again, the oldest first: %s\n%s", order(), view(m))
	}
	shift := tea.MouseClickMsg{X: at("source"), Y: paneTop, Button: tea.MouseLeft, Mod: tea.ModShift}
	if m = drive(m, shift); !strings.Contains(lines(m)[paneTop], "source ▲₂") || !strings.Contains(lines(m)[paneTop], "age ▼₁") {
		t.Errorf("a shift-click on source:\n%s", view(m))
	}
	// builds' commit does not sort
	m = drive(m, slices.Repeat([]tea.Msg{right}, 5)...)
	if m = drive(m, testkit.Click(at("commit"), paneTop)); len(m.panel(m.stage).Sort) != 0 || m.stages[m.stage].name != "builds" {
		t.Errorf("a click on commit sorted builds: %+v", m.panel(m.stage).Sort)
	}
}

// g in the side panel, on a groupable column's line, groups the stage's rows by it under dim headings, which its tab
// does not count; on a column that cannot group, as id, it does nothing.
func TestGInThePanelGroupsTheRows(t *testing.T) {
	m := loaded(t, options(), tea.WindowSizeMsg{Width: 160, Height: 30}, right, press('?'), tab)
	count := m.count(m.stage)
	// down to state, the sixth column: id, source, assignees, age, change, state
	m = drive(m, append(slices.Repeat([]tea.Msg{down}, 6), typed("g")...)...)
	if v := view(m); !slices.Equal(m.panel(m.stage).Group, []string{"state"}) || !strings.Contains(v, "state · 🏁 ready  1") ||
		!strings.Contains(v, "state · 🔨 draft  1") || m.count(m.stage) != count {
		t.Fatalf("g on state: %q, count %s\n%s", m.panel(m.stage).Group, m.count(m.stage), v)
	}
	if !strings.Contains(m.View().Content, style.Off("state · 🏁 ready  1")) {
		t.Error("a group's heading is not dim")
	}
	m = drive(m, slices.Repeat([]tea.Msg{up}, 5)...)
	if m = drive(m, typed("g")...); len(m.panel(m.stage).Group) != 1 {
		t.Errorf("g on id: %q", m.panel(m.stage).Group)
	}
}

// In the side panel a column toggles only by space, or a click on its checkbox: a click on its name takes the keyboard,
// to sort or group by it, and toggles nothing. The control under the keyboard is inverse across the panel but a cell.
func TestAClickOnAColumnsNameTogglesNothing(t *testing.T) {
	m := loaded(t, options(), tea.WindowSizeMsg{Width: 140, Height: 30}, press('?'))
	box := m.paneWidth() + 3
	m = drive(m, testkit.Click(box+3, 4), testkit.Release(box+3, 4))
	if p := m.panel(0); len(p.Hidden) != 0 || !m.inPanel || p.ColumnAt(4) != "source" {
		t.Errorf("a click on source's name: hidden %v, in panel %v", p.Hidden, m.inPanel)
	}
	lines, _ := m.panel(0).Lines()
	if got := ansi.StringWidth(lines[4]); got != m.sideWidth()-1 || !strings.HasPrefix(lines[4], style.Cursor(grid.Ticked)[:4]) {
		t.Errorf("the control under the keyboard is %d cells inverse, not %d: %q", got, m.sideWidth()-1, lines[4])
	}
	if m = drive(m, testkit.Click(box, 4), testkit.Release(box, 4)); !m.panel(0).Hidden["source"] {
		t.Error("a click on source's checkbox did not toggle it")
	}
}

// g groups by a column alone, G within the groups there are; a grouped column leaves the header and the rows, whose
// heading names its value, and the rows and the header stand in two cells a level, a click on a name finding its
// column there.
func TestGroupingNestsWithShiftAndIndentsTheRows(t *testing.T) {
	m := loaded(t, options(), tea.WindowSizeMsg{Width: 160, Height: 30}, right, press('?'), tab)
	// down to source, then to assignees: g on each groups by it alone; G nests
	m = drive(m, down, down, typed("g")[0], down, typed("g")[0])
	if got := m.panel(m.stage).Group; !slices.Equal(got, []string{"assignees"}) {
		t.Fatalf("g on source, then g on assignees: %q", got)
	}
	m = drive(m, up, typed("G")[0])
	if got := m.panel(m.stage).Group; !slices.Equal(got, []string{"assignees", "source"}) {
		t.Fatalf("G on source: %q", got)
	}
	header := ansi.Cut(lines(m)[paneTop], 0, m.paneWidth())
	if strings.Contains(header, "source") || strings.Contains(header, "assignees") || !strings.HasPrefix(header, strings.Repeat(" ", 4+3)) {
		t.Errorf("the header of rows grouped two deep:\n%s", view(m))
	}
	first := slices.IndexFunc(lines(m), func(l string) bool { return strings.Contains(l, "#1") })
	if !strings.HasPrefix(lines(m)[first], "    ") || strings.Contains(lines(m)[first], GitHub) {
		t.Errorf("a row grouped two deep is not indented four cells, or keeps its source:\n%s", view(m))
	}
	// a click on a name in the indented header sorts by it
	if m = drive(m, esc, testkit.Click(strings.Index(lines(m)[paneTop], "age"), paneTop)); !strings.Contains(lines(m)[paneTop], "age ▲") {
		t.Errorf("a click on age in the indented header:\n%s", view(m))
	}
	// g on a column grouping alone stops it
	if p := m.panel(m.stage).Grouped("assignees", false).Grouped("assignees", false); len(p.Group) != 0 {
		t.Errorf("g twice: %q", p.Group)
	}
}

// A click on a column's name with ctrl or option held, or with the right button, as macOS may send a ctrl-click,
// groups the rows by it alone; with shift too, within the groups there are. The same click on a group's heading stops
// grouping by its column alone, the other groups kept; on a row, it does nothing.
func TestACtrlClickGroupsAndOnAHeadingUngroups(t *testing.T) {
	m := loaded(t, options(), tea.WindowSizeMsg{Width: 160, Height: 30}, right)
	at := func(name string) int { return strings.Index(lines(m)[paneTop], name) }
	heading := func(text string) int {
		return slices.IndexFunc(lines(m), func(l string) bool { return strings.Contains(l, text) })
	}
	group := func() []string { return m.panel(m.stage).Group }
	if m = drive(m, tea.MouseClickMsg{X: at("source"), Y: paneTop, Button: tea.MouseLeft, Mod: tea.ModCtrl}); !slices.Equal(group(), []string{"source"}) {
		t.Fatalf("a ctrl-click on source: %q\n%s", group(), view(m))
	}
	nest := tea.MouseClickMsg{X: at("assignees"), Y: paneTop, Button: tea.MouseLeft, Mod: tea.ModAlt | tea.ModShift}
	if m = drive(m, nest); !slices.Equal(group(), []string{"source", "assignees"}) {
		t.Fatalf("a shift-option-click on assignees: %q\n%s", group(), view(m))
	}
	before := group()
	if m = drive(m, tea.MouseClickMsg{X: 10, Y: heading("#1"), Button: tea.MouseRight}); !slices.Equal(group(), before) {
		t.Errorf("a right-click on a row: %q", group())
	}
	if m = drive(m, tea.MouseClickMsg{X: 10, Y: heading("assignees · "), Button: tea.MouseRight}); !slices.Equal(group(), []string{"source"}) {
		t.Fatalf("a right-click on an assignees heading: %q\n%s", group(), view(m))
	}
	if m = drive(m, tea.MouseClickMsg{X: 10, Y: heading("source · "), Button: tea.MouseLeft, Mod: tea.ModCtrl}); len(group()) != 0 {
		t.Errorf("a ctrl-click on a source heading: %q\n%s", group(), view(m))
	}
	// a plain click on a name still sorts, and groups nothing
	if m = drive(m, testkit.Click(at("age"), paneTop)); len(group()) != 0 || len(m.panel(m.stage).Sort) != 1 {
		t.Errorf("a click on age: group %q, sort %+v", group(), m.panel(m.stage).Sort)
	}
}

// Underway groups by change: the issues one pull request carries stand under its heading, and those without one under
// none.
func TestUnderwayGroupsByChange(t *testing.T) {
	m := loaded(t, options(), tea.WindowSizeMsg{Width: 160, Height: 30}, right)
	m.setPanel(m.panel(m.stage).Grouped("change", false))
	if got := m.panel(m.stage).Group; !slices.Equal(got, []string{"change"}) {
		t.Fatalf("grouped by change: %q", got)
	}
	headings := 0
	for _, r := range m.rows(m.stage) {
		if r.Aside && m.panel(m.stage).Heading(r) == "change" {
			headings++
		}
	}
	if headings == 0 || !strings.Contains(view(m), "change · ") {
		t.Errorf("no heading per change:\n%s", view(m))
	}
}
