// The panel beside the rows, driven as the person drives it: tab and / give it the keyboard, typing narrows the rows,
// space hides a column or picks whose rows show, and a click acts on a control. A click on a row's gutter opens it.
package flow

import (
	"path/filepath"
	"slices"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/empowerite/kit/projects/tui/internal/grid"
	"github.com/empowerite/kit/projects/tui/internal/style"
	"github.com/empowerite/kit/projects/tui/internal/testkit"
)

// panelKeys is the keys bar while the panel has the keyboard.
const panelKeys = "↑↓ move · space toggle · type to search · esc back"

// typed gives a key press for each rune of the text, as a terminal sends typing.
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
	if !m.inPanel || !m.legend || m.bar().Focused || lines(m)[23] != " "+panelKeys {
		t.Fatalf("tab: in panel %v, side %v, keys %q", m.inPanel, m.legend, lines(m)[23])
	}
	if !strings.Contains(view(m), grid.Search+" ▏") {
		t.Errorf("the search under the keyboard has no caret:\n%s", view(m))
	}
	m = drive(m, typed("NOUN")...)
	if got := keys(m); !slices.Equal(got, []string{"#46"}) || !shows(m, stageGlyphs["backlog"], "1") {
		t.Errorf("typing noun: rows %v\n%s", got, view(m))
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
	if m = drive(m, esc); m.inPanel || !m.legend || !m.bar().Focused {
		t.Errorf("esc: in panel %v, side %v", m.inPanel, m.legend)
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
	// each stage keeps its own panel: the desk's search is empty
	if m.panel(2).Query != "" || m.panel(0).Query != "" {
		t.Errorf("the panels: %q, %q", m.panel(2).Query, m.panel(0).Query)
	}
}

func TestThePanelHidesAColumnAndShowsTheCallersRowsOrEveryoneElses(t *testing.T) {
	record := filepath.Join(t.TempDir(), "calls")
	t.Setenv("FAKE_FLOW_CALLS", record)
	m := loaded(t, options(), tab, down, down, press(tea.KeySpace))
	if m.me != "ada" {
		t.Fatalf("the caller's login: %q", m.me)
	}
	if strings.Contains(drawn(m), Owner) || !strings.Contains(view(m), grid.Unticked+" assignees") {
		t.Errorf("assignees hidden:\n%s", view(m))
	}
	// the controls: the search, five columns, then all, mine, and others
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
	// underway's seven named columns
	m = drive(m, esc, right)
	m = drive(m, append([]tea.Msg{tab}, slices.Repeat([]tea.Msg{down}, 9)...)...)
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
	// the side's lines: the search, a gap, columns, then number and assignees
	if m = drive(m, testkit.Click(x, paneTop+4)); !m.inPanel || !m.panel(0).Hidden["assignees"] ||
		strings.Contains(drawn(m), Owner) {
		t.Fatalf("a click on assignees: in panel %v\n%s", m.inPanel, view(m))
	}
	if m = drive(m, testkit.Click(x, paneTop+1)); !m.inPanel || m.panel(0).Cursor != 2 {
		t.Errorf("a click on a gap moved the panel's cursor: %d", m.panel(0).Cursor)
	}
	// a click on the gutter of a row's first line opens it, from the panel; again closes it
	if m = drive(m, testkit.Click(1, paneTop+1)); m.inPanel || !m.inRows || m.stages[0].cursor != 1 ||
		!m.stages[0].open["#46"] {
		t.Errorf("a click on #46's gutter: in panel %v, cursor %d\n%s", m.inPanel, m.stages[0].cursor, view(m))
	}
	if m = drive(m, testkit.Click(1, paneTop+1)); m.stages[0].open["#46"] {
		t.Errorf("a second click on the gutter did not close the row")
	}
	// a click past the panel's lines, or on the side while it is hidden, acts on nothing
	before := view(m)
	if m = drive(m, testkit.Click(x, paneTop+m.paneHeight()-1)); view(m) != before || m.inPanel {
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
	if side := m.side(); len(side) != 1 || !strings.HasPrefix(side[0], grid.Search) {
		t.Errorf("the side of a stage kittui does not draw: %q", side)
	}
}
