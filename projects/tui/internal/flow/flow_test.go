// The screen is driven key by key through Update, against testdata/fake-kit, which answers each stage from the
// fixtures beside it, as Bubble Tea would drive it.
package flow

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/empowerite/kit/projects/tui/internal/grid"
	"github.com/empowerite/kit/projects/tui/internal/kit"
	"github.com/empowerite/kit/projects/tui/internal/pane"
	"github.com/empowerite/kit/projects/tui/internal/shell"
	"github.com/empowerite/kit/projects/tui/internal/source"
	"github.com/empowerite/kit/projects/tui/internal/style"
	"github.com/empowerite/kit/projects/tui/internal/table"
	"github.com/empowerite/kit/projects/tui/internal/tabs"
	"github.com/empowerite/kit/projects/tui/internal/testkit"
)

// eight is the picture's default order.
var eight = []string{"backlog", "underway", "desk", "worktrees", "changes", "builds", "incoming", "releases"}

// runner gives a kit.Runner over testdata/fake-kit.
func runner(t *testing.T) kit.Runner {
	t.Helper()
	return testkit.Runner(t, "flow")
}

// bare gives the options, every stage's headers hidden, for a test of how the rows alone take the pane.
func bare() Options {
	o := options()
	o.NoHeaders = true
	return o
}

// options gives the eight stages and origin as a test knows them, and the tables without their preset line, which
// views_test.go tests, so down from the bar enters the rows.
func options() Options {
	return Options{Stages: eight, Origin: Origin{Host: "github.com", Repo: "empowerite/kit"}, NoPresets: true,
		Clock: &testkit.Clock{}}
}

// answer gives a stage's first load's answer, the JSON its command printed.
func answer(stage, data string) source.Answer[[]byte] {
	return source.Answer[[]byte]{Name: stage, Value: []byte(data)}
}

// clockOf gives the test's clock the screen's waits go through, whose timers fire when the test says.
func clockOf(m Model) *testkit.Clock { return m.clock.(*testkit.Clock) }

// latest gives what the latest wait on the test's clock brings, and forgets them all: a read's every answer waits
// anew, and only the wait from the latest asks again.
func latest(m Model) tea.Msg {
	waits := clockOf(m).Fire()
	return waits[len(waits)-1]
}

// viewing gives the options with the tables' preset line, their presets kept in the file named, or for the test alone
// when it is empty.
func viewing(file string) Options {
	o := options()
	o.NoPresets, o.Presets = false, file
	return o
}

// drive sends the messages, running every command each update returns and feeding its messages back, a batch's
// each in turn, as the program would. Quit is noted and dropped. Init is loaded's to send, once.
func drive(m Model, msgs ...tea.Msg) Model {
	for _, msg := range msgs {
		if _, ok := msg.(toggleMsg); ok {
			msg = m.toggleKey()
		}
		m, _ = testkit.Drive(m, faded, msg)
	}
	return m
}

// faded says a notice's time is up, which drive never feeds back, so a test sees what a notice says; a test of the
// fade sends it itself. The rate limits' and the marks' minutes wait on the test's clock, which fires none until the
// test says.
func faded(msg tea.Msg) bool {
	_, ok := msg.(shell.FadedMsg)
	return ok
}

// TestMain waits no time for a notice to fade, as drive runs the wait's command in place.
func TestMain(m *testing.M) {
	fades = 0
	os.Exit(m.Run())
}

// refreshAll drops the app's menu down and chooses Refresh All, the item r moves to.
var refreshAll = []tea.Msg{press('='), typed("r")[0], press(tea.KeyEnter)}

// toggleMsg stands, in drive, for the key that opens the row under the cursor or closes it: toggle.
type toggleMsg struct{}

// toggle opens the row under the cursor when it is closed, with right, and closes it when it is open, with left.
var toggle tea.Msg = toggleMsg{}

// toggleKey gives the key toggle sends for the row under the cursor.
func (m Model) toggleKey() tea.KeyPressMsg {
	s, rows := m.stages[m.stage], m.rows(m.stage)
	if s.Cursor < len(rows) && s.Open[rows[s.Cursor].Key] {
		return left
	}
	return right
}

// loaded opens the screen at 120×24, the common width, and loads every stage, then sends the messages.
func loaded(t *testing.T, o Options, msgs ...tea.Msg) Model {
	t.Helper()
	m := New(runner(t), o)
	first := []tea.Msg{tea.WindowSizeMsg{Width: 120, Height: 24}}
	if cmd := m.Init(); cmd != nil {
		first = append(first, cmd())
	}
	return drive(m, append(first, msgs...)...)
}

func press(code rune) tea.KeyPressMsg { return tea.KeyPressMsg{Code: code} }

// unfold turns Auto-group off, as kittui's settings do, and sets each stage's view again, so its rows, loaded already,
// stand as a view set while Auto-group is off leaves them: grouped by nothing, the source, one tracker's on every row,
// a column.
func unfold(m *Model) {
	folding(m, false)
	at := m.stage
	for i := range m.stages {
		m.stage = i
		m.apply(m.chosen(i))
	}
	m.stage = at
}

// regroup gives the panel grouping the rows by the columns alone, the outermost first, whatever it grouped by before,
// as the view grouped them as it loaded.
func regroup(p grid.Panel, ids ...string) grid.Panel {
	for _, g := range p.Groups() {
		p = p.Grouped(g, true)
	}
	for _, id := range ids {
		p = p.Grouped(id, true)
	}
	return p
}

// folding has the tables fold, or not, as kittui's settings say.
func folding(m *Model, on bool) {
	n, _ := m.Update(shell.SettingsMsg{Settings: shell.Settings{KeysBar: true, FoldShared: on}})
	*m = n.(Model)
}

// hiddenOf gives the fields a panel hides, in their order.
func hiddenOf(p grid.Panel) []string {
	var out []string
	for _, id := range p.Arrange(p.Columns) {
		if id != "" && p.Field(id).Placement == grid.Hidden {
			out = append(out, id)
		}
	}
	return out
}

var (
	up    = press(tea.KeyUp)
	down  = press(tea.KeyDown)
	left  = press(tea.KeyLeft)
	right = press(tea.KeyRight)
	esc   = press(tea.KeyEscape)
)

// raw gives the screen without its escapes, as it is drawn.
func raw(m Model) string { return ansi.Strip(m.View().Content) }

// view gives the screen as lines gives it, one string.
func view(m Model) string { return strings.Join(lines(m), "\n") }

// shows says whether the bar draws a stage's glyph and its count, or no count when it is empty, with the stage's name
// between them, and the menu's glyph after the selected stage's.
func shows(m Model, glyph, count string) bool {
	re := regexp.QuoteMeta(glyph) + `( \S+)?`
	if count != "" {
		re += " +" + regexp.QuoteMeta(count)
	}
	re += `( ` + regexp.QuoteMeta(tabs.Menu) + `)?`
	// on a bar wide enough for every tab, so a stage the screen scrolls past still answers
	b := m.bar()
	b.Width = 1000
	return regexp.MustCompile(re + ` +( |│|$)`).MatchString(strings.Split(ansi.Strip(b.View()), "\n")[1])
}

// lines gives the screen's lines without their escapes, the title's without the search beside it, and the rest above
// the keys bar without the spaces they end in: the title, the bar, and the rows as they are drawn.
func lines(m Model) []string {
	out := strings.Split(raw(m), "\n")
	for k, l := range out {
		switch {
		case k == len(out)-1:
		case k == 0:
			out[k] = strings.TrimRight(ansi.Cut(l, 0, m.titleWidth()), " ")
		default:
			out[k] = strings.TrimRight(l, " ")
		}
	}
	return out
}

// band is what opens the faint band the grid draws the cursor's row on.
var band, _, _ = strings.Cut(style.Band("x", 1), "x")

// cursor gives the first line the cursor's row is drawn on, without its escapes and its gutter, or nothing when no row
// is under the cursor.
func cursor(m Model) string {
	for _, l := range strings.Split(m.View().Content, "\n") {
		// under the headings of groups, the rows stand in
		if strings.HasPrefix(strings.TrimLeft(l, " "), band) {
			return strings.TrimSpace(strings.TrimLeft(strings.TrimSpace(ansi.Strip(l)), style.Focus+style.Opened))
		}
	}
	return ""
}

// squeeze gives a line with each run of spaces one space, as a tab's count right-aligned in it reads.
func squeeze(s string) string { return regexp.MustCompile(` +`).ReplaceAllString(s, " ") }

// calls reads the words of every kit call the fake recorded but the rate limits' and the marks' reads, which each load
// brings, and the reads of what kit kept of each stage, which the start brings; quota_test, marks_test, and kept_test
// count those.
func calls(t *testing.T, path string) []string {
	t.Helper()
	got, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	return slices.DeleteFunc(strings.Split(strings.TrimSpace(string(got)), "\n"), func(c string) bool {
		return c == "flow --quota" || c == "flow --marks" || strings.HasSuffix(c, " --cached")
	})
}

func TestEveryStageLoadsAtOnceWithItsWords(t *testing.T) {
	record := filepath.Join(t.TempDir(), "calls")
	t.Setenv("FAKE_FLOW_CALLS", record)
	o := options()
	o.NoFetch, o.Filters = true, []string{"--mine", "--owner=bob"}
	m := New(runner(t), o)
	// the eight read what kit kept of them, then load
	batch, ok := m.Init()().(tea.BatchMsg)
	if !ok || len(batch) != 16 {
		t.Fatalf("Init did not start the eight at once: %T of %d", batch, len(batch))
	}
	for i := range m.stages {
		if !m.stages[i].src.Loading || m.count(i) != "…" {
			t.Errorf("%s is not loading before it answers", m.stages[i].name)
		}
	}
	if !strings.Contains(view(m), "loading") {
		t.Errorf("the pane does not say loading:\n%s", view(m))
	}
	m = drive(m, batch)
	want := []string{"flow backlog --json --mine --owner=bob", "flow underway --json --mine --owner=bob",
		"flow desk --json", "flow worktrees --json", "flow changes --json --mine --owner=bob",
		"flow builds --json", "flow incoming --json --no-fetch", "flow releases --json"}
	if got := calls(t, record); strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Errorf("the calls:\n%s\nwant:\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
	for i := range m.stages {
		if m.stages[i].src.Loading {
			t.Errorf("%s still loads", m.stages[i].name)
		}
	}
	// an incoming that did not fetch moves nothing, so nothing loads again
	if _, cmd := m.Update(answer("incoming", `{"fetched": false, "remotes": []}`)); cmd != nil {
		t.Errorf("incoming's answer without a fetch started something")
	}
}

// Incoming's fetch, at the start, moves what the backlog, the desk, the worktrees, and underway read: they load again
// once it has fetched, keeping their rows on the screen until the new ones come.
func TestTheFetchReloadsTheStagesItMovesKeepingTheirRows(t *testing.T) {
	record := filepath.Join(t.TempDir(), "calls")
	t.Setenv("FAKE_FLOW_CALLS", record)
	m := New(runner(t), options())
	m = drive(m, tea.WindowSizeMsg{Width: 120, Height: 24})
	batch := m.Init()().(tea.BatchMsg)
	var fetched tea.Msg
	for _, cmd := range batch {
		msg := cmd()
		if l, ok := msg.(source.Answer[[]byte]); ok && l.Name == "incoming" {
			fetched = msg
		} else {
			m = drive(m, msg)
		}
	}
	next, cmd := m.Update(fetched)
	m = next.(Model)
	if cmd == nil || m.find("desk").src.Loading || m.find("backlog").issues == nil || m.count(0) != "3" {
		t.Fatalf("the fetch blanked the stages it moves: %v %q", m.find("desk").src.Loading, m.count(0))
	}
	m = drive(m, cmd())
	var again []string
	for _, call := range calls(t, record)[8:] {
		again = append(again, strings.Fields(call)[1])
	}
	slices.Sort(again)
	if want := []string{"backlog", "desk", "underway", "worktrees"}; !slices.Equal(again, want) {
		t.Errorf("after the fetch, %v loaded again, not %v", again, want)
	}
}

func TestWithoutIncomingEveryStageLoadsAtOnce(t *testing.T) {
	record := filepath.Join(t.TempDir(), "calls")
	t.Setenv("FAKE_FLOW_CALLS", record)
	m := New(runner(t), Options{Stages: []string{"changes", "worktrees"}})
	// each reads what kit kept of it, then loads
	batch, ok := m.Init()().(tea.BatchMsg)
	if !ok || len(batch) != 4 {
		t.Fatalf("Init did not start both: %T", m.Init()())
	}
	m = drive(m, batch)
	if got := calls(t, record); strings.Join(got, "\n") != "flow changes --json\nflow worktrees --json" {
		t.Errorf("the calls: %q", got)
	}
	if first := lines(m)[0]; first != "  "+shell.App+" kit flow · "+shell.Books {
		t.Errorf("the title without incoming: %q", first)
	}
	if labels := squeeze(lines(m)[2]); !strings.Contains(labels, "changes 3") || !strings.Contains(labels, "worktrees 4") ||
		strings.Index(labels, "changes") > strings.Index(labels, "worktrees") {
		t.Errorf("the labels follow kit's order: %q", labels)
	}
}

func TestLeftAndRightMoveAlongTheBarOnlyWhileItHasTheKeyboard(t *testing.T) {
	m := loaded(t, options())
	if mark, _ := style.In(stageColors["backlog"]); m.stage != 0 || !strings.Contains(m.View().Content, mark("➊")) {
		t.Fatalf("the first stage is not selected:\n%s", view(m))
	}
	// left at the first stage stays there, and right at the last
	m = drive(m, left)
	if m.stage != 0 {
		t.Errorf("left at the first stage: stage %d", m.stage)
	}
	if last := drive(m, right, right, right, right, right, right, right, right); last.stage != 7 {
		t.Errorf("right at the last stage: stage %d", last.stage)
	}
	m = drive(m, right, right)
	if mark, _ := style.In(stageColors["desk"]); m.stage != 2 || !strings.Contains(m.View().Content, Desk+" "+mark("desk")) {
		t.Errorf("right twice: stage %d\n%s", m.stage, view(m))
	}
	// inside the rows, left and right are the rows' own, and move nothing
	m = drive(m, right, down, right, left)
	if m.stage != 3 || !m.inRows {
		t.Errorf("left and right inside the rows: stage %d, in rows %v", m.stage, m.inRows)
	}
	// esc gives the bar the keyboard back, and right moves again, onto the next stage's first row
	m = drive(m, esc, right, down)
	if m.stage != 4 || !strings.HasPrefix(cursor(m), "🔀 #2") {
		t.Errorf("the next stage's first row is not under the cursor:\n%s", view(m))
	}
	// from changes, five to the right stops at the end: builds, incoming, releases, and there
	m = drive(m, esc, right, right, right, right, right)
	if m.stage != 7 {
		t.Errorf("right past the end of the bar: %d", m.stage)
	}
}

// A click on a row puts the cursor there and gives the rows the keyboard, and opens nothing, so space then selects it;
// a click on its ▸ opens it, and on its ▾ closes it. A click on a … line, on the legend, or off the pane does nothing.
func TestAClickOnARowSelectsItAndAgainOpensOrClosesIt(t *testing.T) {
	click := testkit.Click
	m := loaded(t, bare(), right, right, right)
	// the screen line the row is drawn on, as the person sees it
	if y := slices.IndexFunc(strings.Split(raw(m), "\n"), func(l string) bool { return strings.Contains(l, "42-settings") }); y != m.rowsTop()+1 {
		t.Fatalf("the second row is drawn on line %d, not %d", y, m.rowsTop()+1)
	}
	if m = drive(m, click(6, m.rowsTop()+1)); !m.inRows || m.find("worktrees").Cursor != 1 {
		t.Fatalf("a click on the second row: in rows %v, cursor %d", m.inRows, m.find("worktrees").Cursor)
	}
	if m = drive(m, click(6, m.rowsTop()+1)); strings.Contains(view(m), style.Opened) {
		t.Errorf("a click on the row under the cursor opened it:\n%s", view(m))
	}
	if m = drive(m, click(table.FoldCell, m.rowsTop()+1)); !strings.Contains(view(m), style.Opened+" 42-settings") {
		t.Errorf("a click on the row's ▸ did not open it:\n%s", view(m))
	}
	if m = drive(m, click(table.FoldCell, m.rowsTop()+1)); strings.Contains(view(m), style.Opened) {
		t.Errorf("a click on its ▾ did not close it:\n%s", view(m))
	}
	if m = drive(m, click(6, m.top())); m.find("worktrees").Cursor != 0 || m.stage != 3 {
		t.Errorf("a click on the first row: cursor %d, stage %d", m.find("worktrees").Cursor, m.stage)
	}
	before := view(m)
	m = drive(m, press('?'))
	for _, c := range []tea.Msg{click(m.paneWidth()+4, 1), click(6, 0), click(6, 23),
		tea.MouseClickMsg{X: 6, Y: m.top() + 1, Button: tea.MouseRight}} {
		if m = drive(m, c); m.find("worktrees").Cursor != 0 || strings.Contains(view(m), style.Opened) {
			t.Errorf("%v moved the cursor or opened a row:\n%s", c, view(m))
		}
	}
	if m = drive(m, press('?')); view(m) != before {
		t.Errorf("the clicks changed the screen:\n%s", view(m))
	}
	// on a short pane, the … line above is no row's
	m = loaded(t, bare(), tea.WindowSizeMsg{Width: 80, Height: 8}, right, right, right, down, down, down)
	if m.find("worktrees").Cursor != 2 || !strings.Contains(lines(m)[m.top()], "above") {
		t.Fatalf("the short pane: cursor %d\n%s", m.find("worktrees").Cursor, view(m))
	}
	if m = drive(m, click(6, m.top())); m.find("worktrees").Cursor != 2 {
		t.Errorf("a click on … above moved the cursor to %d:\n%s", m.find("worktrees").Cursor, view(m))
	}
}

func TestANumberOrAClickSelectsAStageFromAnywhere(t *testing.T) {
	m := loaded(t, options(), down)
	if m = drive(m, tea.KeyPressMsg{Code: '5', Text: "5"}); m.stage != 4 || m.inRows {
		t.Errorf("5 in the rows: stage %d, in rows %v", m.stage, m.inRows)
	}
	for _, k := range []tea.KeyPressMsg{{Code: tea.KeyKp8}, {Code: '8', Mod: tea.ModAlt}, {Code: '8', Mod: tea.ModSuper}} {
		if m = drive(loaded(t, options(), down), k); m.stage != 7 || m.inRows {
			t.Errorf("%s: stage %d, in rows %v", k, m.stage, m.inRows)
		}
	}
	if m = drive(m, tea.KeyPressMsg{Code: '9', Text: "9"}); m.stage != 7 {
		t.Errorf("9 with eight stages moved to %d", m.stage)
	}
	// wide enough that ➂ shows with the last stage selected
	m = drive(m, tea.WindowSizeMsg{Width: 200, Height: 24})
	x := testkit.Column(lines(m)[1], '➂')
	if m = drive(m, down, testkit.Click(x, 1)); m.stage != 2 || m.inRows {
		t.Errorf("a click on ➂: stage %d, in rows %v\n%s", m.stage, m.inRows, view(m))
	}
	if m.View().MouseMode != tea.MouseModeCellMotion {
		t.Error("the mouse is off")
	}
}

func TestDownEntersTheRowsAndUpOnTheFirstRowReturnsToTheBar(t *testing.T) {
	m := loaded(t, options(), right, right, right)
	if m.inRows || cursor(m) != "" {
		t.Fatalf("on the bar, a row has the focus:\n%s", view(m))
	}
	m = drive(m, down)
	if !m.inRows || !strings.HasPrefix(cursor(m), "1-the-first-release") {
		t.Fatalf("down did not enter the rows:\n%s", view(m))
	}
	m = drive(m, down)
	if !strings.HasPrefix(cursor(m), "42-settings") {
		t.Errorf("down once more:\n%s", view(m))
	}
	m = drive(m, down, down, down)
	if m.find("worktrees").Cursor != 3 {
		t.Errorf("down past the last row: %d", m.find("worktrees").Cursor)
	}
	m = drive(m, up, up, up)
	if !m.inRows || m.find("worktrees").Cursor != 0 {
		t.Errorf("up to the first row: in rows %v, cursor %d", m.inRows, m.find("worktrees").Cursor)
	}
	m = drive(m, up)
	if m.inRows || cursor(m) != "" {
		t.Errorf("up on the first row did not return to the bar:\n%s", view(m))
	}
	m = drive(m, up)
	if m.inRows {
		t.Errorf("up on the bar entered the rows")
	}
}

func TestRightOpensARowAndLeftClosesIt(t *testing.T) {
	m := loaded(t, options(), right, right, right, down, right)
	v := view(m)
	for _, want := range []string{"f6f51bc #1 materialize", "… and 263 more", " M kit/bin/kit-flow",
		"?? docs/flow.md"} {
		if !strings.Contains(v, want) {
			t.Errorf("the open worktree lacks %q:\n%s", want, v)
		}
	}
	if !m.find("worktrees").Open["/home/kit/worktrees/1-the-first-release-v0-1-0"] {
		t.Errorf("the row is not open by its key: %v", m.find("worktrees").Open)
	}
	// right on an open row and enter keep it open; left closes it, and again keeps it closed
	if m = drive(m, right, press(tea.KeyEnter)); !strings.Contains(view(m), "f6f51bc") {
		t.Errorf("right or enter closed the row:\n%s", view(m))
	}
	m = drive(m, left, left)
	if v := view(m); strings.Contains(v, "f6f51bc") || m.stage != 3 {
		t.Errorf("left did not close the row, or moved the stage:\n%s", v)
	}
	// enter on the bar opens nothing
	m = drive(m, up, up, up, press(tea.KeyEnter))
	if strings.Contains(view(m), "f6f51bc") {
		t.Errorf("enter on the bar opened a row")
	}
	// an incoming row is a leaf
	m = drive(m, right, right, down, toggle)
	if n := len(lines(m)); n != 24 || len(m.find("incoming").Open) != 0 {
		t.Errorf("a leaf opened: %d lines, open %v", n, m.find("incoming").Open)
	}
}

func TestRefreshAllRunsEveryStageAgain(t *testing.T) {
	record := filepath.Join(t.TempDir(), "calls")
	t.Setenv("FAKE_FLOW_CALLS", record)
	m := loaded(t, options(), right, right, right, right, right, right, right, press('='), typed("r")[0])
	next, cmd := m.Update(press(tea.KeyEnter))
	m = next.(Model)
	// releases keeps its rows, stale, its dot on its tab, until the answer comes
	if cmd == nil || strings.Contains(view(m), "loading") || !shows(m, Release, "3") || !m.find("releases").src.Stale {
		t.Fatalf("Refresh All did not run releases again:\n%s", view(m))
	}
	m = drive(m, cmd())
	// the eight and the four the fetch moved, then the eight again and the four again
	if got := calls(t, record); len(got) != 24 || slices.Index(got[12:], "flow releases --json") < 0 {
		t.Errorf("the calls: %q", got)
	}
	if !shows(m, Release, "3") || m.find("releases").src.Stale {
		t.Errorf("releases did not come back:\n%s", view(m))
	}
}

func TestARefusedStageShowsKitsLineAndRefreshTriesAgain(t *testing.T) {
	t.Setenv("FAKE_FLOW_REFUSE", "changes")
	m := loaded(t, options(), right, right, right, right)
	v := view(m)
	refusal := Failed + " kit flow changes: the forge cannot be read; run: kit help flow"
	if !shows(m, Change, "!") || !strings.Contains(v, refusal) {
		t.Fatalf("the refusal is not shown:\n%s", v)
	}
	if !strings.Contains(m.View().Content, style.Refusal(refusal)) {
		t.Errorf("the refusal is not red")
	}
	// with no pull requests known, the worktrees show none
	m = drive(m, left)
	if v := strings.Join(lines(m)[4:], "\n"); strings.Contains(v, "no pull request yet") || strings.Contains(v, Change) {
		t.Errorf("the worktrees claim pull requests after changes failed:\n%s", v)
	}
	t.Setenv("FAKE_FLOW_REFUSE", "")
	m = drive(m, append([]tea.Msg{right}, refreshAll...)...)
	if v := view(m); !shows(m, Change, "3") || strings.Contains(v, "cannot be read") {
		t.Errorf("Refresh All did not try again:\n%s", v)
	}
}

func TestAStageThatDiesSilentlyIsARefusalNotAnEmptyStage(t *testing.T) {
	t.Setenv("FAKE_FLOW_SILENT", "backlog")
	m := loaded(t, options())
	v := view(m)
	if !shows(m, Issue, "!") || strings.Contains(v, "no open issue") ||
		!strings.Contains(v, Failed+" kit flow backlog exited 1 with nothing on stderr") {
		t.Errorf("the silent failure:\n%s", v)
	}
	// a kit killed by a signal has no code; one with a line keeps it
	if got := refusalOf("backlog", &kit.ExitError{Code: -1}); got != "kit flow backlog was killed" {
		t.Errorf("killed: %q", got)
	}
	if got := refusalOf("backlog", &kit.ExitError{Code: 1, Stderr: "❌ the line\n"}); got != "the line" {
		t.Errorf("with a line: %q", got)
	}
}

func TestADocumentOfAnotherShapeIsOneRefusalLine(t *testing.T) {
	t.Setenv("FAKE_FLOW_ABOVE", "incoming")
	m := loaded(t, options(), right, right, right, right, right, right)
	v := view(m)
	// the line wraps at the pane's width, as every refusal does
	want := "kit flow incoming --json printed an array where an object of remotes was expected; run kit flow " +
		"inside a repository"
	if m.find("incoming").refusal != want || !shows(m, Fetched, "!") ||
		!strings.Contains(v, Failed+" kit flow incoming --json printed an array where an object of remotes was") {
		t.Errorf("an array for incoming: %q\n%s", m.find("incoming").refusal, v)
	}
	t.Setenv("FAKE_FLOW_ABOVE", "backlog")
	m = loaded(t, options())
	v = view(m)
	want = `kit flow backlog --json printed an array of objects without "number" where an array of issues was ` +
		"expected; run kit flow inside a repository"
	if m.stages[0].refusal != want || !shows(m, Issue, "!") || strings.Contains(v, "#0") ||
		!strings.Contains(v, Failed+` kit flow backlog --json printed an array of objects without "number"`) {
		t.Errorf("desks for backlog: %q\n%s", m.stages[0].refusal, v)
	}
	t.Setenv("FAKE_FLOW_ABOVE", "desk")
	m = loaded(t, options(), right, right)
	want = "kit flow desk --json printed an array where an object was expected; run kit flow inside a repository"
	if m.find("desk").refusal != want || !shows(m, Desk, "!") || strings.Contains(view(m), Desk+" main") {
		t.Errorf("desks above desks: %q\n%s", m.find("desk").refusal, view(m))
	}
	for name, data := range map[string]string{"changes": "{}", "worktrees": "[]", "desk": "[]",
		"releases": `[{"version": "1"}, {}]`, "builds": `{"id": 1}`, "incoming": `{"fetched": false}`,
		"underway": `[{"number": 1}]`} {
		if shape(name, []byte(data)) == nil {
			t.Errorf("%s took %s", name, data)
		}
	}
	for name, data := range map[string]string{"backlog": "[]", "incoming": `{"fetched": false, "remotes": []}`,
		"underway": `[{"number": 1, "pr": {}}]`, "changes": "null",
		"releases": `[{"version": "1"}]`, "worktrees": " {} ", "desk": "{}", "builds": `[{"id": 1}]`} {
		if err := shape(name, []byte(data)); err != nil {
			t.Errorf("%s refused %s: %v", name, data, err)
		}
	}
}

func TestQuitCodes(t *testing.T) {
	// alt-esc and super-esc quit from anywhere with 0
	for _, mod := range []tea.KeyMod{tea.ModAlt, tea.ModSuper} {
		for _, at := range [][]tea.Msg{nil, {down}} {
			m := loaded(t, options(), at...)
			next, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEscape, Mod: mod})
			if m = next.(Model); cmd == nil || cmd() != tea.Quit() || m.Interrupted() || m.ctx.Err() == nil {
				t.Errorf("%v esc after %v: cmd %v, interrupted %v, context %v", mod, at, cmd, m.Interrupted(), m.ctx.Err())
			}
		}
	}
	// esc on the bar and q are no way out
	m := loaded(t, options())
	for _, k := range []tea.KeyPressMsg{press(tea.KeyEscape), press('q')} {
		if next, cmd := m.Update(k); cmd != nil || next.(Model).ctx.Err() != nil {
			t.Errorf("%s quit the screen", k)
		}
	}
	// ctrl-c quits from anywhere: the rows, and over the legend
	for _, at := range [][]tea.Msg{nil, {down}, {down, press('?')}} {
		m = loaded(t, options(), at...)
		next, cmd := m.Update(tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl})
		if m = next.(Model); cmd == nil || cmd() != tea.Quit() || !m.Interrupted() || m.ctx.Err() == nil {
			t.Errorf("ctrl-c after %v: cmd %v, interrupted %v, context %v", at, cmd, m.Interrupted(), m.ctx.Err())
		}
	}
}

// Esc walks back from the rows to the bar and leaves the legend as it is: only F1 or ? takes it away.
func TestEscWalksBackFromTheRowsToTheBarAndKeepsTheLegend(t *testing.T) {
	m := loaded(t, options(), right, right, right, down, down, press('?'))
	if !m.legend.open || !strings.Contains(view(m), "worktrees legend") {
		t.Fatalf("? did not show the legend:\n%s", view(m))
	}
	next, cmd := m.Update(press(tea.KeyEscape))
	if m = next.(Model); cmd != nil || !m.legend.open || m.inRows || m.stage != 3 || cursor(m) != "" {
		t.Fatalf("esc in the rows: legend %v, in rows %v, stage %d\n%s", m.legend.open, m.inRows, m.stage, view(m))
	}
	// on the bar, esc does nothing
	next, _ = m.Update(press(tea.KeyEscape))
	if m = next.(Model); !m.legend.open || m.ctx.Err() != nil {
		t.Errorf("esc on the bar: legend %v, context %v", m.legend.open, m.ctx.Err())
	}
	if m = drive(m, press('?')); m.legend.open {
		t.Error("? did not take the legend away")
	}
	if m = drive(m, press(tea.KeyF1)); !m.legend.open {
		t.Error("F1 did not show the legend")
	}
}

// at80 gives the keys bar as a screen of 80 columns draws it, from the bar as wide as it takes.
func at80(keys string) string { return fit(keys, 79) }

// fit gives the keys bar in the room a screen has for it, from the bar as wide as it takes: whole controls dropped from
// the screen's end, … in their place, until it fits, and the last alone clipped.
func fit(keys string, room int) string {
	segments := strings.Split(keys, " │ ")
	for k := 0; k < len(segments); k++ {
		line := strings.Join(segments[k:], " │ ")
		if k > 0 {
			line = "… │ " + line
		}
		if ansi.StringWidth(line) <= room || k == len(segments)-1 {
			return ansi.Truncate(line, room, "…")
		}
	}
	return ""
}

// The keys bar on the bar, over the legend there, and in the rows, on a row that opens, closed and open: the screen's
// keys, the stages' bar's, then the rows', each control's between rules; ? and F1 show the legend from anywhere.
const (
	screenKeys = "= menu / search ? legend │ "
	barKeys    = screenKeys + "←→ tab ↓ rows 1-8 jump : tab menu"
	inRows     = screenKeys + "1-8 jump : tab menu │ esc STAGE ↑↓ move ␣·⇧↑↓ select ⌃A all"
	opens      = inRows + " → expand"
	closes     = inRows + " ← collapse"
)

// keysFor gives the keys bar for the screen as it is: esc in the rows names the stage selected, as STAGE stands for; and in rows more than the pane shows, the page keys move a page at a time.
func keysFor(m Model, keys string) string {
	keys = strings.Replace(keys, "STAGE", m.stages[m.stage].name, 1)
	if m.inRows && len(m.lines(m.stage)) > m.rowsHeight() {
		keys = strings.Replace(keys, "↑↓ move ", "↑↓ move ⇞⇟ page ", 1)
	}
	return keys
}

func TestTheKeysBarIsTheLastLineInEveryState(t *testing.T) {
	for _, c := range []struct {
		msgs []tea.Msg
		keys string
	}{
		{nil, barKeys},
		{[]tea.Msg{right, right, right, down}, opens},
		{[]tea.Msg{down, down, down, down}, opens},
		{[]tea.Msg{right, right, right, down, right}, closes},
		{[]tea.Msg{press('?')}, barKeys},
		{[]tea.Msg{press('?'), press(tea.KeyPgDown)}, barKeys},
	} {
		for _, width := range []int{80, 120, 200} {
			// each load is a model of its own, so the sizes load side by side
			t.Run(fmt.Sprintf("%v at %d", c.msgs, width), func(t *testing.T) {
				t.Parallel()
				for _, height := range []int{24, 12, 8, 7, 6, 5, 4, 3, 2, 1} {
					m := loaded(t, options(), append([]tea.Msg{tea.WindowSizeMsg{Width: width, Height: height}},
						c.msgs...)...)
					// a bar too narrow for every key drops the screen's, then the next control's, … in their place
					want := fit(keysFor(m, c.keys), width-1)
					got := lines(m)
					if len(got) != height || got[height-1] != " "+want {
						t.Errorf("after %v at %d×%d: %d lines, the last %q", c.msgs, width, height, len(got),
							got[len(got)-1])
					}
				}
			})
		}
	}
	// at a width too narrow for it, the bar keeps the stages' bar's keys, after …, clipped, never wrapped
	m := loaded(t, options(), tea.WindowSizeMsg{Width: 30, Height: 12})
	if got := lines(m); len(got) != 12 || !strings.HasPrefix(got[11], " … │ ←→ tab ↓ rows ") ||
		ansi.StringWidth(got[11]) > 30 || !strings.HasSuffix(got[11], "…") {
		t.Errorf("the bar at 30 columns: %q", got[len(got)-1])
	}
}

func TestAStageThatOutlastsTheLimitIsARefusalAndRRunsItAgain(t *testing.T) {
	t.Setenv("FAKE_FLOW_SLOW", "releases")
	m := New(runner(t), options())
	m.limit = 100 * time.Millisecond
	m = drive(m, tea.WindowSizeMsg{Width: 120, Height: 24}, right, right, right, right, right, right, right)
	start := time.Now()
	msg := m.load("releases", false)()
	// the fake sleeps five seconds: the answer comes at the limit only when the sleep is killed with the kit
	if took := time.Since(start); took > time.Second {
		t.Fatalf("the stage did not end at the limit: %s", took)
	}
	m = drive(m, msg)
	v := view(m)
	if !shows(m, Release, "!") ||
		!strings.Contains(v, Failed+" kit flow releases took longer than 2 minutes; the menu's Refresh All runs it again") {
		t.Fatalf("the stage past its limit:\n%s", v)
	}
	t.Setenv("FAKE_FLOW_SLOW", "")
	m = drive(m, refreshAll...)
	if v := view(m); !shows(m, Release, "3") || strings.Contains(v, "took longer") {
		t.Errorf("Refresh All did not run it again:\n%s", v)
	}
}

func TestQuitKillsTheStagesStillRunning(t *testing.T) {
	record := filepath.Join(t.TempDir(), "calls")
	t.Setenv("FAKE_FLOW_CALLS", record)
	t.Setenv("FAKE_FLOW_SLOW", "releases")
	m := New(runner(t), options())
	answers := make(chan tea.Msg, 1)
	load := m.load("releases", false)
	go func() { answers <- load() }()
	// the fake records its call before it sleeps: alt-esc comes once it runs
	for start := time.Now(); calls(t, record) == nil; time.Sleep(10 * time.Millisecond) {
		if time.Since(start) > 2*time.Second {
			t.Fatal("the stage did not start")
		}
	}
	m = drive(m, tea.KeyPressMsg{Code: tea.KeyEscape, Mod: tea.ModAlt})
	select {
	case msg := <-answers:
		if err := msg.(source.Answer[[]byte]).Err; !errors.Is(err, context.Canceled) {
			t.Errorf("the stage ended with %v, not the quit", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("the stage outlived the screen")
	}
}

func TestTheTitleCarriesOriginAndTheFetchIsTheIncomingStage(t *testing.T) {
	m := loaded(t, options())
	if first := lines(m)[0]; first != "  "+shell.App+" kit flow · "+Repository+" empowerite/kit" {
		t.Errorf("the title: %q", first)
	}
	if !strings.Contains(m.View().Content, Repository+" "+style.Repo("empowerite/kit")) {
		t.Errorf("the repository is not in its own color: %q", lines(m)[0])
	}
	// the bar comes right under the title
	if !strings.Contains(lines(m)[1], "➊") || !shows(m, Fetched, "6") {
		t.Errorf("the bar: %q", lines(m)[2])
	}
	// while incoming loads, its tab says so
	m = New(runner(t), options())
	m = drive(m, tea.WindowSizeMsg{Width: 120, Height: 24})
	if labels := lines(m)[2]; !shows(m, Fetched, "…") {
		t.Errorf("incoming loading: %q", labels)
	}
	o := options()
	o.NoFetch, o.Origin = true, Origin{}
	m = loaded(t, o, right, right, right, right, right, right)
	if first := lines(m)[0]; first != "  "+shell.App+" kit flow · "+shell.Books {
		t.Errorf("the title with --no-fetch: %q", first)
	}
	// without the fetch, each remote's heading says not fetched, and its tab has no count
	pane := lines(m)[4:10]
	for i, want := range []string{Remote + " origin · github · github.com", "not fetched",
		Remote + " upstream · gitlab · git.example.org", "not fetched", Remote + " backup · no forge · nas.local",
		"not fetched"} {
		if strings.TrimSpace(pane[i]) != want {
			t.Errorf("the incoming pane with --no-fetch, line %d is not %q:\n%s", i, want, view(m))
		}
	}
	if shows(m, Fetched, "0") || !shows(m, Fetched, "") {
		t.Errorf("the incoming label with --no-fetch: %q", lines(m)[2])
	}
	if !strings.Contains(m.View().Content, style.Off("not fetched")) {
		t.Errorf("not fetched is not dim")
	}
	// with no remote at all, the pane says why it is empty
	m.find("incoming").in.Remotes = nil
	m.rebuild()
	if !strings.Contains(view(m), "   not fetched: kit flow --no-fetch skips the fetch") {
		t.Errorf("no remote with --no-fetch:\n%s", view(m))
	}
	m = loaded(t, Options{Stages: []string{"releases"}})
	if first := lines(m)[0]; first != "  "+shell.App+" kit flow · "+shell.Books {
		t.Errorf("the title alone: %q", first)
	}
}

func TestIncomingGroupsItsRowsByRemoteThePrimaryFirst(t *testing.T) {
	m := loaded(t, options(), right, right, right, right, right, right)
	want := []string{
		Remote + " origin · github · github.com",
		"a1b2c3d  #1 docs: the docs say what the code does",
		"e4f5a6b  #1 yaml: yq reads every YAML file",
		Branch + "       46-a-noun-lists updated",
		Branch + "       42-settings new branch",
		Remote + " upstream · gitlab · git.example.org",
		Gone + "       old-experiment upstream gone",
		Tag + "       release/0.2.0 new tag",
		Remote + " backup · no forge · nas.local",
		"unchanged",
	}
	rows := lines(m)[4 : 4+len(want)+1]
	for i, text := range want {
		if !strings.HasPrefix(strings.TrimSpace(rows[i]), text) {
			t.Errorf("row %d is not %q: %q", i, text, rows[i])
		}
	}
	if strings.TrimSpace(rows[len(want)]) != "" {
		t.Errorf("a row past the last remote: %q", rows[len(want)])
	}
	// the headings and unchanged are dim, and neither counts: six rows across the three remotes
	for _, text := range []string{Remote + " upstream · gitlab · git.example.org", "unchanged"} {
		if !strings.Contains(m.View().Content, style.Off(text)) {
			t.Errorf("%q is not dim", text)
		}
	}
	if !shows(m, Fetched, "6") || m.count(6) != "6" {
		t.Errorf("incoming does not count its rows alone: %q", lines(m)[2])
	}
	// every row is a leaf, the headings too
	m = drive(m, down, toggle, down, toggle)
	if len(m.find("incoming").Open) != 0 {
		t.Errorf("an incoming row opened: %v", m.find("incoming").Open)
	}
	// a remote whose fetch failed shows git's line, in red, and is no row of the count
	m.find("incoming").in.Remotes[0].Error = "fatal: could not read from remote repository"
	m.find("incoming").in.Remotes[0].Branches.Gone, m.find("incoming").in.Remotes[0].Tags = nil, nil
	m.rebuild()
	failed := Failed + " fatal: could not read from remote repository"
	if v := view(m); !strings.Contains(v, Remote+" upstream · gitlab · git.example.org\n    "+failed+"\n") ||
		!strings.Contains(m.View().Content, style.Refusal(failed)) || m.count(6) != "4" {
		t.Errorf("a failed fetch: count %s\n%s", m.count(6), v)
	}
	m.find("incoming").in.Remotes[0].Error = ""
	m.rebuild()
	// with every remote unchanged, each says so and the count is zero
	for i := range m.find("incoming").in.Remotes {
		r := &m.find("incoming").in.Remotes[i]
		r.Commits, r.Branches.New, r.Branches.Updated, r.Branches.Gone, r.Tags = nil, nil, nil, nil, nil
	}
	m.rebuild()
	if v := view(m); strings.Count(v, "unchanged") != 3 || m.count(6) != "0" {
		t.Errorf("three remotes unchanged: count %s\n%s", m.count(6), v)
	}
	// with no remote at all, the pane says so
	m.find("incoming").in.Remotes = nil
	m.rebuild()
	if !strings.Contains(view(m), "   no remote to fetch") {
		t.Errorf("no remote:\n%s", view(m))
	}
}

func TestTheGoneBranchesAreOneRowThatOpensToEach(t *testing.T) {
	m := loaded(t, options(), right, right, right, down, down, down, down, toggle)
	v := view(m)
	for _, want := range []string{style.Opened + " " + Gone + " gone: old-experiment, 44-a-spike",
		"      " + Gone + " old-experiment upstream gone · kit tidy removes it",
		"      " + Gone + " 44-a-spike upstream gone"} {
		if !strings.Contains(v, want) {
			t.Errorf("the gone branches lack %q:\n%s", want, v)
		}
	}
	if !strings.Contains(m.View().Content, style.Off(Gone+" gone: old-experiment, 44-a-spike")) {
		t.Errorf("the gone row is not dim")
	}
}

func TestChangesCarryTheirChecksThreadsAndIssues(t *testing.T) {
	// a change's row is wide: at 80 columns its right end is clipped, so the screen is wider here
	m := loaded(t, options(), tea.WindowSizeMsg{Width: 160, Height: 24}, right, right, right, right, down, toggle)
	v := view(m)
	// the checks of the failing one are cut at the cap, to give the title room
	for _, want := range []string{style.Opened + " 🔀 #2", "jeffrey-aguilera  🏁 ready", Passed + " 1 / 1 3m",
		"🔨 draft", Running + " 0 / 2", Threads + " 2 unresolved", "🔥 failing:gate", Failed + " lint failing…",
		"#1  👤 jeffrey-aguilera  " + InFlight + " in flight  8d  " + Issue + " The first release, v0.1.0"} {
		if !strings.Contains(v, want) {
			t.Errorf("the changes lack %q:\n%s", want, v)
		}
	}
	m = drive(m, down, toggle)
	if !strings.Contains(view(m), "closes no issue") {
		t.Errorf("a pull request that closes nothing says nothing:\n%s", view(m))
	}
}

func TestBacklogAndReleasesOpenToTheirDetail(t *testing.T) {
	m := loaded(t, options(), tea.WindowSizeMsg{Width: 80, Height: 24})
	// the source and labels columns have tests of their own
	m.setPanel(m.panel(m.stage).Placed("source", grid.Hidden).Placed("labels", grid.Hidden))
	m = drive(m, down, toggle)
	v := view(m)
	// the title has what the other columns leave, with its glyph, which leads to the issue's page, and the branch, which
	// one row alone has, with its own
	for _, want := range []string{"#45", "#46  jeffrey-aguilera   2d  🌿 46-a-noun-li…  " + Issue + " A noun lists",
		"#7  ada           +1  40d", "picture: five stages,", "interactive on a terminal,"} {
		if !strings.Contains(v, want) {
			t.Errorf("the backlog lacks %q:\n%s", want, v)
		}
	}
	// the issue's page is its title's glyph's, not a line of its own, and nothing says how to start it
	if strings.Contains(v, "https://") || strings.Contains(v, "kit work 45") ||
		last(m.rows(0)[0].Cells).Link != "https://github.com/empowerite/kit/issues/45" {
		t.Errorf("the backlog's links:\n%s", v)
	}
	m = loaded(t, options(), right, right, right, right, right, right, right, down, down, toggle)
	v = view(m)
	for _, want := range []string{"0.2.0       " + Draft + " draft", "0.1.0       " + Release + " release",
		Latest + " latest",
		"0.1.0-rc.7  " + Pre + " pre-release  28h", "kit 0.1.0", "release/0.1.0", "2026-10-31T20:00:00Z"} {
		if !strings.Contains(v, want) {
			t.Errorf("the releases lack %q:\n%s", want, v)
		}
	}
	if rows := m.rows(7); strings.Contains(v, "https://") ||
		rows[1].Cells[1].Link != "https://github.com/empowerite/kit/releases/tag/release/0.1.0" {
		t.Errorf("the release's page is not its kind's glyph's:\n%s", v)
	}
}

func TestThePaneScrollsToKeepTheCursorWhileTheRestStays(t *testing.T) {
	m := loaded(t, bare(), tea.WindowSizeMsg{Width: 80, Height: 8}, right, right, right, down, down, down)
	got := lines(m)
	if len(got) != 8 {
		t.Fatalf("%d lines on a screen of 8:\n%s", len(got), view(m))
	}
	if !strings.HasPrefix(got[0], "  "+shell.App+" kit flow") || !shows(m, Branch, "4") || got[7] != " "+at80(keysFor(m, opens)) {
		t.Errorf("the title, the bar, or the keys moved:\n%s", view(m))
	}
	// three lines of pane: … 2 above, the cursor's row, and the last row, which needs no … line
	if !strings.Contains(got[4], "… 2 above") || !strings.Contains(got[5], "worktrees/spike") || !strings.HasPrefix(cursor(m), "worktrees/spike") ||
		!strings.Contains(got[6], Gone+" gone:") || strings.Contains(view(m), "1-the-first-release") {
		t.Errorf("the pane did not scroll to the cursor:\n%s", view(m))
	}
	// a pane of three lines holds the … lines and the row; one more line shows what the row opens to
	m = drive(m, toggle, tea.WindowSizeMsg{Width: 80, Height: 10})
	if !strings.Contains(view(m), "?? notes.md") || !strings.Contains(view(m), style.Opened) {
		t.Errorf("the open row's detail is not shown with its row:\n%s", view(m))
	}
	m = drive(m, tea.WindowSizeMsg{Width: 80, Height: 8})
	// a page is the pane less one row: two rows here
	m = drive(m, press(tea.KeyPgUp))
	if !m.inRows || m.find("worktrees").Cursor != 0 {
		t.Errorf("page up: in rows %v, cursor %d", m.inRows, m.find("worktrees").Cursor)
	}
	// a page never leaves the rows: up past the first row stays on it, at the top
	m = drive(m, press(tea.KeyPgUp))
	if !m.inRows || m.find("worktrees").Cursor != 0 || !strings.Contains(view(m), "1-the-first-release") {
		t.Errorf("page up past the first row left it, or the top:\n%s", view(m))
	}
	// on the bar, a page does nothing
	if b := drive(m, up, press(tea.KeyPgDown), press(tea.KeyPgUp)); b.inRows || b.find("worktrees").Cursor != 0 {
		t.Errorf("a page on the bar: in rows %v, cursor %d", b.inRows, b.find("worktrees").Cursor)
	}
	m = drive(m, press(tea.KeyPgDown))
	if !m.inRows || m.find("worktrees").Cursor != 2 {
		t.Errorf("page down: in rows %v, cursor %d", m.inRows, m.find("worktrees").Cursor)
	}
	m = drive(m, press(tea.KeyPgDown), press(tea.KeyPgDown))
	if m.find("worktrees").Cursor != 3 {
		t.Errorf("page down past the last row: cursor %d", m.find("worktrees").Cursor)
	}
}

func TestEveryLineFitsTheWidthInCells(t *testing.T) {
	for _, width := range []int{12, 24, 40, 50, 80, 120, 200} {
		for _, legend := range []bool{false, true} {
			t.Run(fmt.Sprintf("%d legend %v", width, legend), func(t *testing.T) {
				t.Parallel()
				everyLineFits(t, width, legend)
			})
		}
	}
}

// everyLineFits opens each stage's first row in turn, at the width, with the legend or without, and holds every line to
// the width; at 80 columns or fewer, a long title is clipped.
func everyLineFits(t *testing.T, width int, legend bool) {
	// one load for the eight: each stage in turn, its first row open, then back to the bar
	m := loaded(t, options(), tea.WindowSizeMsg{Width: width, Height: 40})
	if legend {
		m = drive(m, press('?'))
	}
	for i := range eight {
		if i > 0 {
			m = drive(m, up, right)
		}
		m = drive(m, down, toggle)
		clipped := false
		for _, l := range strings.Split(m.View().Content, "\n") {
			if n := ansi.StringWidth(l); n > width {
				t.Errorf("%s at %d, legend %v: a line of %d cells: %q", eight[i], width, legend, n, ansi.Strip(l))
			}
			// a clipped row ends in …
			clipped = clipped || strings.Contains(strings.TrimRight(ansi.Strip(l), " "), "…")
		}
		long := eight[i] == "changes" || eight[i] == "backlog" || eight[i] == "underway"
		if width <= 80 && long && !clipped {
			t.Errorf("%s at %d: no long title was clipped:\n%s", eight[i], width, view(m))
		}
	}
}

func TestAnUnknownStageIsRefusedWithoutRunning(t *testing.T) {
	record := filepath.Join(t.TempDir(), "calls")
	t.Setenv("FAKE_FLOW_CALLS", record)
	m := loaded(t, Options{Stages: []string{"worktrees", "nonesuch"}}, right)
	if v := view(m); !strings.Contains(squeeze(v), "nonesuch !") ||
		!strings.Contains(v, "kittui does not draw the stage 'nonesuch'") {
		t.Errorf("the unknown stage:\n%s", v)
	}
	if got := calls(t, record); strings.Join(got, "\n") != "flow worktrees --json" {
		t.Errorf("the calls: %q", got)
	}
	m = loaded(t, Options{})
	if v := view(m); !strings.Contains(v, "no stage is configured") || len(lines(m)) != 24 {
		t.Errorf("no stage at all:\n%s", v)
	}
}

func TestOpenReadsTheStagesAndTheWords(t *testing.T) {
	record := filepath.Join(t.TempDir(), "calls")
	t.Setenv("FAKE_FLOW_CALLS", record)
	m, err := Open(runner(t), []string{"--no-fetch", "--mine"})
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, s := range m.stages {
		names = append(names, s.name)
	}
	if strings.Join(names, " ") != strings.Join(eight, " ") || !m.noFetch || strings.Join(m.filters, " ") != "--mine" {
		t.Errorf("Open: %q, no fetch %v, filters %q", names, m.noFetch, m.filters)
	}
	if got := calls(t, record); strings.Join(got, "\n") != "flow --stages" {
		t.Errorf("Open ran %q", got)
	}
	t.Setenv("FAKE_FLOW_REFUSE", "stages")
	_, err = Open(runner(t), nil)
	refusal, ok := err.(*kit.ExitError)
	if !ok || refusal.Code != 2 || refusal.Error() != "kit flow refuses for the test; run: kit help flow" {
		t.Errorf("Open on a refusal: %v", err)
	}
}

func TestTheSelectedRowIsChosen(t *testing.T) {
	m := loaded(t, options(), down)
	chosen, _, _ := strings.Cut(style.Mark("#"), "#")
	if !strings.Contains(m.View().Content, chosen+"#") || !strings.HasPrefix(cursor(m), "#45") {
		t.Errorf("the row under the cursor is not drawn as chosen:\n%s", m.View().Content)
	}
}

func TestAgeIsKitStatuss(t *testing.T) {
	for secs, want := range map[int64]string{0: "0s", 45: "45s", 119: "119s", 120: "2m", 7199: "119m", 7200: "2h",
		172799: "47h", 172800: "2d", 3456000: "40d", -5: "0s"} {
		if got := table.Age(secs); got != want {
			t.Errorf("age(%d) = %q, want %q", secs, got, want)
		}
	}
}

func TestAPaneTooShortSaysHowManyLinesAreAboveAndBelow(t *testing.T) {
	m := loaded(t, bare(), tea.WindowSizeMsg{Width: 80, Height: 9}, right, right, right, down, down, toggle)
	got := lines(m)
	if len(got) != 9 || got[8] != " "+at80(keysFor(m, closes)) {
		t.Fatalf("the screen is not 9 lines with the keys last:\n%s", view(m))
	}
	v := view(m)
	if !strings.Contains(v, "… 1 above") || !strings.Contains(v, "… 4 more") ||
		!strings.Contains(v, style.Opened+" 42-settings") {
		t.Errorf("the open row is not shown between its … lines:\n%s", v)
	}
	// a pane that holds every line says nothing of the kind
	m = loaded(t, options(), right, right, right)
	if strings.Contains(view(m), " above") || strings.Contains(view(m), " more") {
		t.Errorf("a pane that fits says lines are hidden:\n%s", view(m))
	}
}

func TestWorktreesShowTheirPullRequestsAndTheirUncommittedCountButNotTheDesk(t *testing.T) {
	m := loaded(t, bare(), right, right, right)
	rows := lines(m)[m.rowsTop() : m.rowsTop()+3]
	for i, want := range [][]string{
		{" 1-the-first-release-v0-1-0", "↑266 ↓0", "#2  🏁 ready", "3 uncommitted"},
		{" 42-settings", "↑3 ↓1", "no pull request yet", "clean"},
		{"worktrees/spike " + Detached + " detached", "↑0 ↓0", "1 uncommitted"},
	} {
		for _, text := range want {
			if !strings.Contains(rows[i], text) {
				t.Errorf("row %d lacks %q: %q", i, text, rows[i])
			}
		}
	}
	if strings.Contains(rows[2], "no pull request") {
		t.Errorf("a detached checkout wants a pull request: %q", rows)
	}
	if pane := strings.Join(lines(m)[4:], "\n"); strings.Contains(pane, Desk) || strings.Contains(pane, "main") {
		t.Errorf("the worktrees draw the desk:\n%s", pane)
	}
	if !shows(m, Branch, "4") {
		t.Errorf("the worktrees count the desk: %q", lines(m)[2])
	}
	// the numbers line up
	if strings.Index(rows[0], "↓") != strings.Index(rows[1], "↓") {
		t.Errorf("the counts do not line up:\n%s", strings.Join(rows, "\n"))
	}
}

func TestTheDeskIsOneRowThatOpensToItsPathsThenItsCommits(t *testing.T) {
	m := loaded(t, bare(), right, right)
	pane := lines(m)[m.rowsTop():]
	for _, text := range []string{" main", "↑12 ↓1", "2 uncommitted"} {
		if !strings.Contains(pane[0], text) {
			t.Errorf("the desk's row lacks %q: %q", text, pane[0])
		}
	}
	if strings.Contains(pane[0], "no pull request") || strings.TrimSpace(pane[1]) != "" ||
		!shows(m, Desk, "1") {
		t.Errorf("the desk is not one row without a pull request:\n%s", view(m))
	}
	if !strings.Contains(m.View().Content, style.Warn("2 uncommitted")) {
		t.Errorf("the uncommitted count is not yellow")
	}
	m = drive(m, down, toggle)
	v := view(m)
	order := []string{style.Opened + " " + Desk + " main", " M kit.yml", "?? scratch.txt", "c0ffee1 #45 status: the desk",
		"c0ffeea #45 status: the goldens", "c0ffeeb #45 status: the eleventh", "… and 1 more"}
	at := -1
	for _, want := range order {
		k := strings.Index(v, want)
		if k <= at {
			t.Errorf("the open desk lacks %q after the line before it:\n%s", want, v)
		}
		at = k
	}
	if !strings.Contains(m.View().Content, style.Warn(" M")) {
		t.Errorf("an uncommitted path's status is not yellow")
	}
	// a desk on a branch with a pull request shows it, as a worktree does
	m.find("desk").desk.Branch = "1-the-first-release-v0-1-0"
	m.rebuild()
	if !strings.Contains(view(m), "#2  🏁 ready") {
		t.Errorf("the desk does not show its pull request:\n%s", view(m))
	}
	// its pull request's glyph leads to its row under changes, and its state's to its page
	if r := m.rows(m.stage)[0]; r.Cells[3].Link != jumpTo("changes", "#2") ||
		r.Cells[4].Link != "https://github.com/empowerite/kit/pull/2" {
		t.Errorf("the desk's links: %q %q", r.Cells[3].Link, r.Cells[4].Link)
	}
	// a clean desk says so, dim
	m.find("desk").desk.Uncommitted = 0
	m.rebuild()
	if !strings.Contains(m.View().Content, style.Off("clean")) {
		t.Errorf("a clean desk does not say clean")
	}
}

func TestAnOpenCheckoutFillsThePaneAndPagesThroughEveryCommit(t *testing.T) {
	m := loaded(t, options(), right, right, down, toggle, tea.WindowSizeMsg{Width: 80, Height: 11})
	got := lines(m)
	// the pane runs to the keys bar, its last line, right over it, counting what is below
	if len(got) != 11 || !strings.Contains(got[9], "more") {
		t.Fatalf("the open desk does not fill the pane:\n%s", view(m))
	}
	for range 5 {
		if strings.Contains(view(m), "c0ffeeb") {
			break
		}
		m = drive(m, press(tea.KeyPgDown))
	}
	v := view(m)
	if !strings.Contains(v, "c0ffeeb #45 status: the eleventh") || !strings.Contains(v, "… and 1 more") ||
		m.find("desk").Cursor != 0 || !m.inRows {
		t.Errorf("page down did not reach the last commit within the row:\n%s", v)
	}
}

// cellsOf gives every cell a text starts at in a line.
func cellsOf(line, text string) []int {
	var out []int
	for from := 0; ; {
		at := strings.Index(line[from:], text)
		if at < 0 {
			return out
		}
		out = append(out, ansi.StringWidth(line[:from+at]))
		from += at + 1
	}
}

// At 200 columns every name shows. Narrower, the bar scrolls to center the selected stage, no name cut. The selected
// stage's name is in the accent wherever the keyboard is; while the rows have it, the others are dim.
func TestTheBarScrollsToTheSelectedStage(t *testing.T) {
	m := loaded(t, options(), tea.WindowSizeMsg{Width: 200, Height: 24})
	for _, name := range eight {
		if bar := lines(m)[2]; !strings.Contains(bar, name) {
			t.Errorf("at 200, %s is not whole: %q", name, bar)
		}
	}
	if mark, _ := style.In(stageColors["backlog"]); !strings.Contains(m.View().Content, Issue+" "+mark("backlog")) {
		t.Errorf("the selected name is not marked: %q", lines(m)[2])
	}
	m = drive(m, right, right, right, right, right, right, down)
	dim, _, _ := strings.Cut(style.Off("x"), "x")
	mark, _ := style.In(stageColors["incoming"])
	if bar := strings.Split(m.View().Content, "\n")[2]; !strings.Contains(bar, mark("incoming")) ||
		!strings.Contains(bar, dim+" 🛫 underway ") {
		t.Errorf("with the rows' keyboard, the selected stage left its color, or the others are not dim: %q", bar)
	}
	// at 120, ◃ says tabs are hidden on the left; at 80 it is blank, as moving the view back would cut incoming
	for _, c := range []struct {
		width  int
		before string
	}{{120, " " + tabs.Before}, {80, "  "}} {
		m = drive(m, tea.WindowSizeMsg{Width: c.width, Height: 24})
		bar := lines(m)[2]
		if !strings.Contains(squeeze(bar), "│ "+Fetched+" incoming 6 "+tabs.Menu+" │") || !strings.HasPrefix(bar, c.before) ||
			strings.Contains(bar, "…") {
			t.Errorf("at %d, the bar does not scroll to incoming whole: %q", c.width, bar)
		}
	}
}

func TestPageDownScrollsThroughAnOpenRowTallerThanThePaneBeforeMovingOn(t *testing.T) {
	// worktrees is the fourth stage; its first row, opened, is taller than a pane on a short screen
	m := loaded(t, bare(), tea.WindowSizeMsg{Width: 120, Height: 12}, right, right, right, down, toggle)
	if m.stage != 3 || m.find("worktrees").Cursor != 0 || !strings.Contains(view(m), " more") {
		t.Fatalf("the open row does not overflow the pane:\n%s", view(m))
	}
	bare := func(s string) string {
		return strings.TrimSpace(strings.TrimLeft(strings.TrimSpace(ansi.Strip(s)), style.Focus+style.Opened))
	}
	head := style.Opened + " 1-the-first-release"
	seen := map[string]bool{}
	for k := 0; k < 20 && m.find("worktrees").Cursor == 0; k++ {
		for _, l := range lines(m) {
			seen[bare(l)] = true
		}
		// scrolled into the row, its own line stays on the pane's first line, where … above would be
		if pane := lines(m)[4]; m.find("worktrees").Offset > 0 && (!strings.Contains(pane, head) || strings.Contains(pane, "above")) {
			t.Errorf("the open row's line is not pinned atop the pane:\n%s", view(m))
		}
		m = drive(m, press(tea.KeyPgDown))
	}
	// every line the open row holds was shown on the way down, and only then did the cursor move on
	for _, l := range m.lines(3) {
		if l.Row == 0 && !seen[bare(l.Text)] {
			t.Errorf("page down never showed %q", ansi.Strip(l.Text))
		}
	}
	if m.find("worktrees").Cursor == 0 {
		t.Fatalf("page down never left the open row")
	}
	// past the row, the pane says … above again
	if !strings.Contains(lines(m)[4], " above") {
		t.Errorf("past the open row, the pane's first line is not … above:\n%s", view(m))
	}
	// page up comes back through the same row, to the top of the pane
	for k := 0; k < 20 && (m.find("worktrees").Cursor != 0 || m.find("worktrees").Offset > 0); k++ {
		m = drive(m, press(tea.KeyPgUp))
	}
	if m.find("worktrees").Cursor != 0 || m.find("worktrees").Offset != 0 || !strings.Contains(lines(m)[4], head) {
		t.Errorf("page up did not return to the open row's first line:\n%s", view(m))
	}
}

func TestBuildsCarryTheirStateAndOpenToTheirCommitAttemptJobsFailedFirstThenTheirURL(t *testing.T) {
	// builds is the seventh stage, between incoming and releases
	m := loaded(t, bare(), right, right, right, right, right)
	if m.stages[m.stage].name != "builds" || !shows(m, Build, "4") {
		t.Fatalf("builds is not the seventh stage:\n%s", view(m))
	}
	rows := lines(m)[m.rowsTop() : m.rowsTop()+4]
	for i, want := range [][]string{
		{Running + "  release    #7  tag       release/0.1.0  1a2b3c4", "5m", "…"},
		{Failed + "  kit      #160  push      main           8e817bf  #45 status: the builds", "2h", "4m 02s"},
		{Passed + "  kit      #159  push      main           5d376dd  #49 status: an underway stage", "24h", "3m 36s"},
		{Cancelled + "  nightly   #12  schedule  main           5d376dd", "2d", "12s"},
	} {
		for _, text := range want {
			if !strings.Contains(rows[i], text) {
				t.Errorf("row %d lacks %q: %q", i, text, rows[i])
			}
		}
	}
	// the age and how long it took are right-aligned, and a running build's is a dim …
	if !slices.Contains(cellsOf(rows[2], "24h"), cellsOf(rows[0], "5m")[0]-1) {
		t.Errorf("the ages do not line up on the right:\n%s", strings.Join(rows, "\n"))
	}
	if ansi.StringWidth(rows[0]) != ansi.StringWidth(rows[1]) || !strings.HasSuffix(rows[0], "…") {
		t.Errorf("the took column is not right-aligned:\n%s", strings.Join(rows, "\n"))
	}
	if !strings.Contains(m.View().Content, style.Off("…")) {
		t.Errorf("a running build's … is not dim")
	}
	// a title that only repeats the workflow is left out, and the number, how it began, and the commit are dim
	if strings.Contains(rows[0], "1a2b3c4  release") || strings.Contains(rows[3], "5d376dd  nightly") {
		t.Errorf("a title that repeats the workflow is drawn:\n%s", strings.Join(rows, "\n"))
	}
	for _, text := range []string{"#160", "schedule", "8e817bf"} {
		if !strings.Contains(m.View().Content, style.Off(text)) {
			t.Errorf("%q is not dim", text)
		}
	}
	// the failed build opens to its title wrapped in its column, its attempt, its jobs, the failed first with the step
	// that failed under it, then its URL, dim
	m = drive(m, down, down, toggle)
	v := view(m)
	order := []string{style.Opened + " " + Failed + "  kit",
		"8e817bf  #45 status: the builds stage tells which commit", "each run built", "attempt 2", Failed + "  test",
		"failed at bats tests/status.bats", Passed + "  gate", Passed + "  lint", Cancelled + "  docs", Passed + "  kit"}
	at := -1
	for _, want := range order {
		k := strings.Index(v, want)
		if k <= at {
			t.Errorf("the open build lacks %q after the line before it:\n%s", want, v)
		}
		at = k
	}
	if !strings.Contains(v, "test  4m 02s\n") || !strings.Contains(v, "lint     38s\n") {
		t.Errorf("the jobs' durations are not right-aligned:\n%s", v)
	}
	if b := m.find("builds"); strings.Contains(v, "https://") ||
		m.rows(slices.Index(eight, "builds"))[b.Cursor].Cells[0].Link != "https://github.com/empowerite/kit/actions/runs/41309409467" {
		t.Errorf("the run's page is not its state's glyph's:\n%s", v)
	}
	// a first attempt says nothing of it
	if m = drive(m, toggle, down, toggle); strings.Contains(view(m), "attempt") {
		t.Errorf("a first attempt is drawn:\n%s", view(m))
	}
	// a build without jobs, whose title repeats its workflow, has nothing to open to
	m = drive(m, toggle, down, toggle)
	if strings.Contains(view(m), style.Opened+" "+Cancelled) {
		t.Errorf("a build without jobs opens:\n%s", view(m))
	}
	// an empty stage says how far it looked
	m.find("builds").runs = nil
	m.rebuild()
	if !strings.Contains(view(m), "   no build since the last 20 runs") || !shows(m, Build, "0") {
		t.Errorf("the empty builds:\n%s", view(m))
	}
}

func TestBuildsRefuseADocumentWithoutIDs(t *testing.T) {
	t.Setenv("FAKE_FLOW_ABOVE", "builds")
	m := loaded(t, options(), right, right, right, right, right)
	want := `kit flow builds --json printed an array of objects without "id" where an array of builds was ` +
		"expected; run kit flow inside a repository"
	if m.stages[5].refusal != want || !shows(m, Build, "!") {
		t.Errorf("desks for builds: %q\n%s", m.stages[5].refusal, view(m))
	}
	if err := shape("builds", []byte(`{"id": 1}`)); err == nil || !strings.Contains(err.Error(), "printed an object") {
		t.Errorf("builds took an object: %v", err)
	}
}

func TestTookIsToTheSecond(t *testing.T) {
	secs := func(n int64) *int64 { return &n }
	for _, c := range []struct {
		secs *int64
		want string
	}{
		{nil, "…"}, {secs(0), "0s"}, {secs(59), "59s"}, {secs(60), "1m 00s"}, {secs(216), "3m 36s"},
		{secs(3599), "59m 59s"}, {secs(3600), "1h 00m"}, {secs(3725), "1h 02m"}, {secs(-5), "0s"},
	} {
		if got := table.Took(c.secs); got != c.want {
			t.Errorf("took(%v) = %q, want %q", c.secs, got, c.want)
		}
	}
}

func TestUnderwayDrawsEachIssueBesideThePullRequestThatClosesIt(t *testing.T) {
	// underway is the second stage, between backlog and incoming
	m := loaded(t, bare(), right)
	// the source and labels columns have tests of their own
	m.setPanel(m.panel(m.stage).Placed("source", grid.Hidden).Placed("labels", grid.Hidden))
	asCatalog(&m, m.stage) // auto_test has what the screen chooses
	if m.stages[m.stage].name != "underway" || !shows(m, InFlight, "4") {
		t.Fatalf("underway is not the second stage:\n%s", view(m))
	}
	rows := lines(m)[m.rowsTop() : m.rowsTop()+4]
	for i, want := range [][]string{
		{"#1  " + Owner + " jeffrey-agui…", "8d", Change + " #2 ", "🏁 ready", Branch + " here", Pushed + " 60m",
			Issue + " The first release"},
		{"#8", "2d", Change + " #4 ", "🔥 failing:gate", Owner + " bob", Fork + " bob", Pushed + " 25h", Owner + " cy",
			Issue + " Lint the modules"},
		{"#8", "2d", Change + " #11", "🔨 draft", Sprout + " local", Pushed + "  2h", Issue + " Lint the modules"},
		{"#31  " + Owner + " ada", "3h", Change + " #12", "💬 comments:open", Owner + " ada", Issue + " kit commit"},
	} {
		for _, text := range want {
			if !strings.Contains(rows[i], text) {
				t.Errorf("row %d lacks %q: %q", i, text, rows[i])
			}
		}
	}
	// the numbers and the ages are right-aligned, and the pull requests and where line up
	if cellsOf(rows[3], "#31")[0] != cellsOf(rows[0], "#1")[0]-1 ||
		cellsOf(rows[3], "3h")[0] != cellsOf(rows[0], "8d")[0] {
		t.Errorf("the numbers or the ages do not line up on the right:\n%s", strings.Join(rows, "\n"))
	}
	for _, text := range []string{Change, Pushed} {
		if slices.Max(cellsOf(rows[0], text)) != slices.Max(cellsOf(rows[2], text)) {
			t.Errorf("%q does not line up:\n%s", text, strings.Join(rows, "\n"))
		}
	}
	// where lines up, here, local, and the author's alike, and a push's age is right-aligned after its glyph
	if at := cellsOf(rows[0], Branch+" here")[0]; cellsOf(rows[1], Owner+" bob")[0] != at ||
		cellsOf(rows[2], Sprout+" local")[0] != at || slices.Max(cellsOf(rows[3], Owner+" ada")) != at ||
		cellsOf(rows[2], "2h")[0]+1 != cellsOf(rows[0], "60m")[0]+2 {
		t.Errorf("where or the pushes do not line up:\n%s", strings.Join(rows, "\n"))
	}
	// a push by the author names no one; one by another names them; an unknown push draws nothing
	if strings.Contains(rows[0], Owner+" jeffrey-aguilera  "+Pushed) || strings.Count(rows[0], Owner) != 1 ||
		strings.Contains(rows[3], Pushed) {
		t.Errorf("the pushes:\n%s", strings.Join(rows, "\n"))
	}
	// here is plain; another's desk and the assignees are dim; a long title is clipped
	c := m.View().Content
	if !strings.Contains(c, style.Off(Owner+" bob")) || strings.Contains(c, style.Off(Desk+" here")) ||
		!strings.Contains(c, pane.Clip(style.Off(Owner+" jeffrey-aguilera"), 16)) || !strings.HasSuffix(rows[3], "…") {
		t.Errorf("the where cells or the title:\n%s", strings.Join(rows, "\n"))
	}
	// the row here opens to the issue's title wrapped in its column and the worktree; the pull request's title is
	// under changes
	m = drive(m, down, toggle)
	v := view(m)
	order := []string{style.Opened + "  #1",
		"The first release, v0.1.0:", "the kit verbs, the", "picture, and the forge", "seam on GitHub and GitLab",
		"#8"}
	if strings.Contains(v, Change+" #2 The first release") {
		t.Errorf("the open row repeats the pull request:\n%s", v)
	}
	at := -1
	for _, want := range order {
		k := strings.Index(v, want)
		if k <= at {
			t.Errorf("the open row lacks %q after the line before it:\n%s", want, v)
		}
		at = k
	}
	// the pull request's glyph leads to its row under changes, the state's to its page, the title's to the issue's
	links := func(m Model) (string, string) {
		r := m.rows(1)[m.find("underway").Cursor]
		if r.Cells[3].Link != jumpTo("changes", "#2") && r.Cells[3].Link != jumpTo("changes", "#4") {
			t.Errorf("the pull request's glyph leads to %q", r.Cells[3].Link)
		}
		return r.Cells[4].Link, last(r.Cells).Link
	}
	if pr, issue := links(m); strings.Contains(v, "https://") || pr != "https://github.com/empowerite/kit/pull/2" ||
		issue != "https://github.com/empowerite/kit/issues/1" {
		t.Errorf("the links: %q %q\n%s", pr, issue, v)
	}
	// a row elsewhere has no worktree, and with a title that fits, nothing to open
	m = drive(m, toggle, down, toggle)
	v = view(m)
	if pr, _ := links(m); !strings.Contains(v, "Lint the modules") || !strings.HasSuffix(pr, "/kit/pull/4") ||
		strings.Contains(v, Desk+" /") {
		t.Errorf("the open row elsewhere:\n%s", v)
	}
	if s := m.find("underway"); s.Open["#8 #4"] || grid.Opens(m.rows(1), nil, s.Cursor, m.paneWidth()) {
		t.Errorf("a row elsewhere with a short title opens: %v", s.Open)
	}
	// the URLs kit prints win, as a merge request's on GitLab
	s := m.find("underway")
	s.under[1].URL = "https://git.example.org/g/kit/-/issues/8"
	s.under[1].PR.URL = "https://git.example.org/g/kit/-/merge_requests/4"
	m.rebuild()
	if pr, issue := links(m); issue != "https://git.example.org/g/kit/-/issues/8" ||
		pr != "https://git.example.org/g/kit/-/merge_requests/4" {
		t.Errorf("kit's URLs do not win: %q %q", pr, issue)
	}
	// an empty stage says no issue is underway
	s.under = nil
	m.rebuild()
	if !strings.Contains(view(m), "   no issue is underway") || !shows(m, InFlight, "0") {
		t.Errorf("the empty underway:\n%s", view(m))
	}
}

func TestUnderwayRefusesADocumentWithoutPullRequests(t *testing.T) {
	t.Setenv("FAKE_FLOW_ABOVE", "underway")
	m := loaded(t, options(), right)
	want := `kit flow underway --json printed an array of objects without "number" where an array of issues ` +
		"underway was expected; run kit flow inside a repository"
	if m.stages[1].refusal != want || !shows(m, InFlight, "!") {
		t.Errorf("desks for underway: %q\n%s", m.stages[1].refusal, view(m))
	}
	// backlog's issues are not underway's: each needs its pull request
	err := shape("underway", []byte(`[{"number": 1}]`))
	if err == nil || !strings.Contains(err.Error(), `without "pr"`) {
		t.Errorf("underway took issues without pull requests: %v", err)
	}
	// incoming before every remote is refused
	if err := shape("incoming", []byte(`{"fetched": true, "commits": []}`)); err == nil ||
		!strings.Contains(err.Error(), `printed an object without "remotes"`) {
		t.Errorf("incoming took the document before every remote: %v", err)
	}
}

// A click on underway's pull request glyph selects changes, the cursor on that pull request's row; a click on its
// state's glyph opens its page; without changes in the picture, the pull request's glyph opens the page too.
func TestUnderwaysPullRequestLeadsToItsRowUnderChangesAndItsStateToItsPage(t *testing.T) {
	m := loaded(t, bare(), right, down, down)
	at := func(m Model, glyph string) int {
		return cellsOf(lines(m)[m.rowLine(m.find("underway").Cursor)], glyph)[0]
	}
	change, state, y := at(m, "🔀 #4"), at(m, "🔥"), m.rowLine(m.find("underway").Cursor)
	browsed = nil
	if m = drive(m, testkit.Click(state, y)); len(browsed) != 1 || !strings.HasSuffix(browsed[0], "/kit/pull/4") ||
		m.stages[m.stage].name != "underway" {
		t.Fatalf("a click on the state: %v\n%s", browsed, view(m))
	}
	m = drive(m, testkit.Click(change, y))
	if s := m.stages[m.stage]; s.name != "changes" || !m.inRows || m.rows(m.stage)[s.Cursor].Key != "#4" ||
		len(browsed) != 1 {
		t.Fatalf("a click on the pull request: %s, row %d:\n%s", s.name, s.Cursor, view(m))
	}
	// a pull request changes does not hold leaves its cursor where it was
	before := m.stages[m.stage].Cursor
	if m.jump("changes/999"); m.stages[m.stage].Cursor != before {
		t.Errorf("a jump to no row moved the cursor to %d", m.stages[m.stage].Cursor)
	}
	if m.jump("nonesuch/1"); m.stages[m.stage].name != "changes" {
		t.Errorf("a jump to no stage moved to %s", m.stages[m.stage].name)
	}
	// without changes, the pull request's glyph opens its page
	o := options()
	o.Stages = slices.DeleteFunc(slices.Clone(o.Stages), func(s string) bool { return s == "changes" })
	m = loaded(t, o, right)
	// the first row stands under the heading over the groups and source's value, as the view grouped it as it loaded
	first := m.rows(1)[2]
	link := ""
	for _, c := range first.Cells {
		if strings.Contains(c.Link, "/pull/") {
			link = c.Link
		}
	}
	if link != "https://github.com/empowerite/kit/pull/2" {
		t.Errorf("without changes the pull request's glyph leads to %q", link)
	}
}

// browsed is every page a test's screen opened, the newest last; no test opens the browser.
var browsed []string

func init() {
	browser = func(url string) error {
		browsed = append(browsed, url)
		if strings.Contains(url, "refuse") {
			return errors.New("no browser for the test")
		}
		return nil
	}
}

// Backlog and underway say which tracker holds each issue: its glyph and the source kit names, dim; a source kittui
// has no glyph for by its name alone; and none at all leaves the column out, as an empty one. While one tracker holds
// every issue, the view groups the rows by it as it loads, so it is said once, in its group's heading, as any group's
// value, and the column leaves them; with Auto-group off, it stands under the source header, the glyph every row
// shares left out.
func TestBacklogAndUnderwaySayWhichTrackerHoldsEachIssue(t *testing.T) {
	m := loaded(t, options(), tea.WindowSizeMsg{Width: 160, Height: 24})
	for i, name := range []string{"backlog", "underway"} {
		if i > 0 {
			m = drive(m, right)
			folding(&m, true)
			m.apply(m.chosen(m.stage))
		}
		if l := lines(m); strings.Contains(l[m.headerTop()], "source") || !strings.Contains(l[m.rowsTop()], "github") ||
			strings.Count(view(m), "github") != 1 {
			t.Errorf("%s does not say the source once, over the rows:\n%s", name, view(m))
		}
		unfold(&m)
		if l := lines(m); !strings.Contains(l[m.headerTop()], "source") || !strings.Contains(l[m.headerTop()+2], "github") ||
			!strings.Contains(m.View().Content, style.Off("github")) || strings.Contains(view(m), GitHub) {
			t.Errorf("%s not folding does not say the source:\n%s", name, view(m))
		}
	}
	source := fieldsOf("backlog")[slices.IndexFunc(fieldsOf("backlog"), func(f table.Field) bool { return f.ID == "source" })]
	if c := source.Draw([]table.Value{table.Some("gitlab"), table.Some("AB#"), table.Some("")}, ""); c[0].Values[0] != style.Off(GitLab+" gitlab") ||
		c[1].Values[0] != style.Off("AB#") || len(c[2].Values) != 0 {
		t.Errorf("the source cells: %q %q %q", c[0].Values, c[1].Values, c[2].Values)
	}
	s := m.find("underway")
	for k := range s.under {
		s.under[k].Source = ""
	}
	m.rebuild()
	if strings.Contains(lines(m)[m.top()], "source") {
		t.Errorf("a column no issue has a source in has a header:\n%s", view(m))
	}
}

// last gives a row's last cell, the title's, which stands after every other column.
func last(cells []grid.Cell) grid.Cell { return cells[len(cells)-1] }

// rowY gives the screen line the first line of the selected stage's row of the key stands on, -1 where none shows.
func rowY(m Model, key string) int {
	k := slices.IndexFunc(m.rows(m.stage), func(r row) bool { return r.Key == key && !r.Aside || r.Key == key })
	if k < 0 {
		return -1
	}
	return m.rowLine(k)
}

// ruleY gives the screen line the rule under the selected stage's header stands on, as the pane lays it out.
func ruleY(m Model) int {
	lines, _ := m.header()
	return m.headerTop() + len(lines) - 1
}
