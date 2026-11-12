// The screen is driven key by key through Update, against testdata/fake-kit, which answers each stage from the
// fixtures beside it, as Bubble Tea would drive it.
package flow

import (
	"context"
	"encoding/json"
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

// five is the picture's default order.
var five = []string{"issues", "changes", "builds", "incoming", "releases"}

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

// options gives Flow on the five stages and origin as a test knows them, and the tables without their preset line,
// which views_test.go tests, so down from the bar enters the rows.
func options() Options {
	return Options{App: shell.Flow, Stages: five, Origin: Origin{Host: "github.com", Repo: "empowerite/kit"},
		NoPresets: true, Clock: &testkit.Clock{}}
}

// intoRows gives the selected stage's rows the keyboard from the bar, the cursor on the first row: down, past the
// control line's stops, where the stage has a control line, which lands on the outline's first line, then past the
// groups' headings over the first row, as many times as it takes.
func intoRows(m Model) Model {
	for range 3 {
		if m.inRows {
			break
		}
		m = drive(m, down)
	}
	for range 4 {
		if rows, k := m.rows(m.stage), m.stages[m.stage].Cursor; !m.inRows || k >= len(rows) || !grid.Heads(rows[k]) {
			break
		}
		m = drive(m, down)
	}
	return m
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

// TestMain waits no time for a notice to fade, as drive runs the wait's command in place, and gives every test a home
// of its own, as testkit.Main does, so no preset or tab the person last chose at their desk opens a test's screen on
// it; a test that writes settings still sets its own.
func TestMain(m *testing.M) {
	fades = 0
	os.Exit(testkit.Main(m))
}

// refreshAll drops the app's menu down and chooses Refresh All, the item r moves to after Repositories, or at once on
// Repositories, where Repositories is dim.
var refreshAll = []tea.Msg{press('='), typed("r")[0], typed("r")[0], press(tea.KeyEnter)}

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

// flat opens the screen as loaded does, then has the rows of every stage but issues stand flat, the source,
// organization, and repository hidden, and sends the messages: every row kit prints names the place it is of, which a
// stage's view groups by, and those have tests of their own, so a test of how the rows move, scroll, and open draws
// them under no heading and without those columns. The issues are of two repositories, and stand under their
// headings, as the tests of them say.
func flat(t *testing.T, o Options, msgs ...tea.Msg) Model {
	t.Helper()
	m := loaded(t, o)
	at := m.stage
	for i := range m.stages {
		if m.stages[i].name == "issues" {
			continue
		}
		m.stage = i
		m.setPanel(regroup(m.panel(i)).Placed("source", grid.Hidden).Placed("organization", grid.Hidden).
			Placed("repository", grid.Hidden))
	}
	m.stage = at
	return drive(m, msgs...)
}

func press(code rune) tea.KeyPressMsg { return tea.KeyPressMsg{Code: code} }

// unfold turns Auto-group off, as kittui's settings do, sets each stage's view again, and ungroups it, so its rows,
// loaded already, stand flat: grouped by nothing, not even the source, organization, and repository the view groups
// by, which are columns.
func unfold(m *Model) {
	folding(m, false)
	at := m.stage
	for i := range m.stages {
		m.stage = i
		m.apply(m.chosen(i))
		m.setPanel(regroup(m.panel(i)))
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
			out[k] = strings.TrimRight(ansi.Cut(l, 0, m.menuBar().TitleWidth(m.width)), " ")
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
	// the five read what kit kept of them, then load
	batch, ok := m.Init()().(tea.BatchMsg)
	if !ok || len(batch) != 10 {
		t.Fatalf("Init did not start the five at once: %T of %d", batch, len(batch))
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
	want := []string{"flow issues --json --mine --owner=bob", "flow changes --json --mine --owner=bob",
		"flow builds --json", "flow incoming --json --no-fetch", "flow releases --json"}
	if got := calls(t, record); strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Errorf("the calls:\n%s\nwant:\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
	for i := range m.stages {
		if m.stages[i].src.Loading {
			t.Errorf("%s still loads", m.stages[i].name)
		}
	}
}

func TestLeftAndRightMoveAlongTheBarOnlyWhileItHasTheKeyboard(t *testing.T) {
	m := flat(t, options())
	if mark, _ := style.In(stageColors["issues"]); m.stage != 0 || !strings.Contains(m.View().Content, mark("➊")) {
		t.Fatalf("the first stage is not selected:\n%s", view(m))
	}
	// left at the first stage stays there, and right at the last
	m = drive(m, backTab)
	if m.stage != 0 {
		t.Errorf("left at the first stage: stage %d", m.stage)
	}
	if last := drive(m, tab, tab, tab, tab, tab); last.stage != 4 {
		t.Errorf("right at the last stage: stage %d", last.stage)
	}
	m = drive(m, tab)
	if mark, _ := style.In(stageColors["changes"]); m.stage != 1 || !strings.Contains(m.View().Content, Change+" "+mark("changes")) {
		t.Errorf("right once: stage %d\n%s", m.stage, view(m))
	}
	// inside the rows, left and right are the rows' own, and move nothing
	m = drive(m, down, down, right, left)
	if m.stage != 1 || !m.inRows {
		t.Errorf("left and right inside the rows: stage %d, in rows %v", m.stage, m.inRows)
	}
	// esc gives the bar the keyboard back, and right moves again, onto the next stage's first row
	m = drive(m, esc, tab, down, down)
	if m.stage != 2 || !strings.Contains(cursor(m), "release/0.1.0") {
		t.Errorf("the next stage's first row is not under the cursor:\n%s", view(m))
	}
	// from builds, three to the right stops at the end: incoming, releases, and there
	m = drive(m, esc, tab, tab, tab)
	if m.stage != 4 {
		t.Errorf("right past the end of the bar: %d", m.stage)
	}
}

// A click on a row puts the cursor there and gives the rows the keyboard, and opens nothing, so space then selects it;
// a click on its ▸ opens it, and on its ▾ closes it. A click on a … line, on the legend, or off the pane does nothing.
func TestAClickOnARowSelectsItAndAgainOpensOrClosesIt(t *testing.T) {
	click := testkit.Click
	m := flat(t, bare(), tabsTo("builds")...)
	// the screen line the row is drawn on, as the person sees it
	if y := slices.IndexFunc(strings.Split(raw(m), "\n"), func(l string) bool { return strings.Contains(l, "#160") }); y != m.rowsTop()+1 {
		t.Fatalf("the second row is drawn on line %d, not %d", y, m.rowsTop()+1)
	}
	if m = drive(m, click(6, m.rowsTop()+1)); !m.inRows || m.find("builds").Cursor != 2 {
		t.Fatalf("a click on the second row: in rows %v, cursor %d", m.inRows, m.find("builds").Cursor)
	}
	if m = drive(m, click(6, m.rowsTop()+1)); openRows(m) != 0 {
		t.Errorf("a click on the row under the cursor opened it:\n%s", view(m))
	}
	if m = drive(m, click(table.FoldCell, m.rowsTop()+1)); !strings.Contains(view(m), "bats tests/status.bats") {
		t.Errorf("a click on the row's ▸ did not open it:\n%s", view(m))
	}
	if m = drive(m, click(table.FoldCell, m.rowsTop()+1)); openRows(m) != 0 {
		t.Errorf("a click on its ▾ did not close it:\n%s", view(m))
	}
	if m = drive(m, click(6, m.rowsTop())); m.find("builds").Cursor != 1 || m.stage != 2 {
		t.Errorf("a click on the first row: cursor %d, stage %d", m.find("builds").Cursor, m.stage)
	}
	before := view(m)
	m = drive(m, press('?'))
	for _, c := range []tea.Msg{click(m.paneWidth()+4, 1), click(6, 0), click(6, 23),
		tea.MouseClickMsg{X: 6, Y: m.rowsTop() + 1, Button: tea.MouseRight}} {
		if m = drive(m, c); m.find("builds").Cursor != 1 || openRows(m) != 0 {
			t.Errorf("%v moved the cursor or opened a row:\n%s", c, view(m))
		}
	}
	if m = drive(m, press('?')); view(m) != before {
		t.Errorf("the clicks changed the screen:\n%s", view(m))
	}
	// on a short pane, the … line above is no row's
	m = flat(t, bare(), tea.WindowSizeMsg{Width: 80, Height: 10 + titleHeight}, tab, tab, down, down, down, down)
	if m.find("builds").Cursor != 3 || !strings.Contains(lines(m)[m.rowsTop()], pane.Up) {
		t.Fatalf("the short pane: cursor %d\n%s", m.find("builds").Cursor, view(m))
	}
	if m = drive(m, click(6, m.rowsTop())); m.find("builds").Cursor != 3 {
		t.Errorf("a click on … above moved the cursor to %d:\n%s", m.find("builds").Cursor, view(m))
	}
}

func TestANumberOrAClickSelectsAStageFromAnywhere(t *testing.T) {
	m := loaded(t, options(), down)
	if m = drive(m, tea.KeyPressMsg{Code: '4', Text: "4"}); m.stage != 3 || m.inRows {
		t.Errorf("4 in the rows: stage %d, in rows %v", m.stage, m.inRows)
	}
	for _, k := range []tea.KeyPressMsg{{Code: tea.KeyKp5}, {Code: '5', Mod: tea.ModAlt}, {Code: '5', Mod: tea.ModSuper}} {
		if m = drive(loaded(t, options(), down), k); m.stage != 4 || m.inRows {
			t.Errorf("%s: stage %d, in rows %v", k, m.stage, m.inRows)
		}
	}
	if m = drive(m, tea.KeyPressMsg{Code: '9', Text: "9"}); m.stage != 4 {
		t.Errorf("9 with five stages moved to %d", m.stage)
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
	m := flat(t, options(), tab, tab)
	if m.inRows || cursor(m) != "" {
		t.Fatalf("on the bar, a row has the focus:\n%s", view(m))
	}
	m = intoRows(m)
	if !m.inRows || !strings.Contains(cursor(m), "release/0.1.0") {
		t.Fatalf("down did not enter the rows:\n%s", view(m))
	}
	m = drive(m, down)
	if !strings.Contains(cursor(m), "#160") {
		t.Errorf("down once more:\n%s", view(m))
	}
	m = drive(m, down, down, down)
	if m.find("builds").Cursor != 4 {
		t.Errorf("down past the last row: %d", m.find("builds").Cursor)
	}
	m = drive(m, up, up, up)
	if !m.inRows || m.find("builds").Cursor != 1 {
		t.Errorf("up to the first row: in rows %v, cursor %d", m.inRows, m.find("builds").Cursor)
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
	m := flat(t, options(), tab, tab, down, down, right)
	v := view(m)
	for _, want := range []string{"build", "publish"} {
		if !strings.Contains(v, want) {
			t.Errorf("the open build lacks %q:\n%s", want, v)
		}
	}
	if !m.find("builds").Open["41309409501"] {
		t.Errorf("the row is not open by its key: %v", m.find("builds").Open)
	}
	// right on an open row and enter keep it open; left closes it, and again keeps it closed
	if m = drive(m, right, press(tea.KeyEnter)); !strings.Contains(view(m), "publish") {
		t.Errorf("right or enter closed the row:\n%s", view(m))
	}
	m = drive(m, left, left)
	if v := view(m); strings.Contains(v, "publish") || m.stage != 2 {
		t.Errorf("left did not close the row, or moved the stage:\n%s", v)
	}
	// enter on the bar opens nothing
	m = drive(m, up, up, up, press(tea.KeyEnter))
	if strings.Contains(view(m), "publish") {
		t.Errorf("enter on the bar opened a row")
	}
	// an incoming row is a leaf
	m = drive(m, tab, down, toggle)
	if n := len(lines(m)); n != 24 || len(m.find("incoming").Open) != 0 {
		t.Errorf("a leaf opened: %d lines, open %v", n, m.find("incoming").Open)
	}
}

func TestRefreshAllRunsEveryStageAgain(t *testing.T) {
	record := filepath.Join(t.TempDir(), "calls")
	t.Setenv("FAKE_FLOW_CALLS", record)
	m := loaded(t, options(), tab, tab, tab, tab, press('='), typed("r")[0], typed("r")[0])
	next, cmd := m.Update(press(tea.KeyEnter))
	m = next.(Model)
	// releases keeps its rows, stale, its dot on its tab, until the answer comes
	if cmd == nil || strings.Contains(view(m), "loading") || !shows(m, Release, "3") || !m.find("releases").src.Stale {
		t.Fatalf("Refresh All did not run releases again:\n%s", view(m))
	}
	m = drive(m, cmd())
	// the five, then the five again
	if got := calls(t, record); len(got) != 10 || slices.Index(got[5:], "flow releases --json") < 0 {
		t.Errorf("the calls: %q", got)
	}
	if !shows(m, Release, "3") || m.find("releases").src.Stale {
		t.Errorf("releases did not come back:\n%s", view(m))
	}
}

func TestARefusedStageShowsKitsLineAndRefreshTriesAgain(t *testing.T) {
	t.Setenv("FAKE_FLOW_REFUSE", "changes")
	m := loaded(t, options(), tab)
	v := view(m)
	refusal := Failed + " kit flow changes: the forge cannot be read; run: kit help flow"
	if !shows(m, Change, "!") || !strings.Contains(v, refusal) {
		t.Fatalf("the refusal is not shown:\n%s", v)
	}
	if !strings.Contains(m.View().Content, style.Refusal(refusal)) {
		t.Errorf("the refusal is not red")
	}
	t.Setenv("FAKE_FLOW_REFUSE", "")
	m = drive(m, refreshAll...)
	if v := view(m); !shows(m, Change, "3") || strings.Contains(v, "cannot be read") {
		t.Errorf("Refresh All did not try again:\n%s", v)
	}
}

func TestAStageThatDiesSilentlyIsARefusalNotAnEmptyStage(t *testing.T) {
	t.Setenv("FAKE_FLOW_SILENT", "issues")
	m := loaded(t, options())
	v := view(m)
	if !shows(m, Issue, "!") || strings.Contains(v, "no open issue") ||
		!strings.Contains(v, Failed+" kit flow issues --json exited 1 with nothing on stderr") {
		t.Errorf("the silent failure:\n%s", v)
	}
	// a kit killed by a signal has no code; one with a line keeps it
	if got := refusalOf("issues", &kit.ExitError{Code: -1}); got != "kit flow issues --json was killed" {
		t.Errorf("killed: %q", got)
	}
	if got := refusalOf("issues", &kit.ExitError{Code: 1, Stderr: "❌ the line\n"}); got != "the line" {
		t.Errorf("with a line: %q", got)
	}
}

func TestADocumentOfAnotherShapeIsOneRefusalLine(t *testing.T) {
	t.Setenv("FAKE_FLOW_ABOVE", "incoming")
	m := loaded(t, options(), tab, tab, tab)
	v := view(m)
	// the line wraps at the pane's width, as every refusal does
	want := `kit flow incoming --json printed an array where an object with "rows" was expected`
	if m.find("incoming").refusal != want || !shows(m, Fetched, "!") ||
		!strings.Contains(v, Failed+" kit flow incoming --json printed an array where an object with") {
		t.Errorf("an array for incoming: %q\n%s", m.find("incoming").refusal, v)
	}
	t.Setenv("FAKE_FLOW_ABOVE", "issues")
	m = loaded(t, options())
	v = view(m)
	want = "kit flow issues --json printed a record without .number"
	if m.stages[0].refusal != want || !shows(m, Issue, "!") || strings.Contains(v, "#0") ||
		!strings.Contains(v, Failed+" "+want) {
		t.Errorf("desks for issues: %q\n%s", m.stages[0].refusal, v)
	}
	for name, data := range map[string]string{"changes": "{}",
		"releases": `[{"version": "1"}, {}]`, "builds": `{"id": 1}`, "incoming": `{"fetched": false}`,
		"issues": `[{"pr": {"number": 2}}]`} {
		if (&stage{name: name}).decode([]byte(data), false) == nil {
			t.Errorf("%s took %s", name, data)
		}
	}
	for name, data := range map[string]string{"issues": `[{"number": 1, "pr": {"number": 2}}, {"number": 1}]`,
		"incoming": `{"fetched": false, "rows": []}`, "changes": "[]",
		"releases": `[{"key": "release/1", "version": "1"}]`, "builds": `[{"id": 1}]`} {
		if err := (&stage{name: name}).decode([]byte(data), false); err != nil {
			t.Errorf("%s refused %s: %v", name, data, err)
		}
	}
}

func TestQuitCodes(t *testing.T) {
	// ctrl-q quits from anywhere with 0; esc, with a modifier or none, never quits
	for _, at := range [][]tea.Msg{nil, {down}} {
		m := loaded(t, options(), at...)
		next, cmd := m.Update(tea.KeyPressMsg{Code: 'q', Mod: tea.ModCtrl})
		if m = next.(Model); cmd == nil || cmd() != tea.Quit() || m.Interrupted() || m.ctx.Err() == nil {
			t.Errorf("ctrl-q after %v: cmd %v, interrupted %v, context %v", at, cmd, m.Interrupted(), m.ctx.Err())
		}
		for _, mod := range []tea.KeyMod{tea.ModAlt, tea.ModSuper} {
			if _, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEscape, Mod: mod}); cmd != nil && cmd() == tea.Quit() {
				t.Errorf("%v esc after %v quit", mod, at)
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
	m := loaded(t, options(), tab, tab, down, down, press('?'))
	if !m.legend.open || !strings.Contains(view(m), "builds legend") {
		t.Fatalf("? did not show the legend:\n%s", view(m))
	}
	next, cmd := m.Update(press(tea.KeyEscape))
	if m = next.(Model); cmd != nil || !m.legend.open || m.inRows || m.stage != 2 || cursor(m) != "" {
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

// fit gives the keys bar in the room a screen has for it, from the bar as wide as it takes: the screen's keys kept and
// whole controls dropped after them, … in their place, until it fits, and the last clipped; too narrow for the
// screen's keys, those dropped too, from the screen's end on.
func fit(keys string, room int) string {
	segments := strings.Split(keys, " │ ")
	if len(segments) > 1 && ansi.StringWidth(segments[0]+" │ … │ ") <= room {
		for k := 1; k < len(segments); k++ {
			line := segments[0] + " │ " + strings.Join(segments[k:], " │ ")
			if k > 1 {
				line = segments[0] + " │ … │ " + strings.Join(segments[k:], " │ ")
			}
			if ansi.StringWidth(line) <= room || k == len(segments)-1 {
				return ansi.Truncate(line, room, "…")
			}
		}
	}
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
	screenKeys = "= menu - workspace / search ? legend │ "
	barKeys    = screenKeys + "⇥·⇧⇥ tab ↓ groups 1-5 jump : tab menu"
	inRows     = screenKeys + "1-5 jump : tab menu │ esc tabs ↑↓ move ␣·⇧↑↓ select ⌃A all"
	opens      = inRows + " → show"
	closes     = inRows + " ← hide"
)

// keysFor gives the keys bar for the screen as it is: in rows more than the pane shows, the page keys move a page at a
// time; in rows wider than the pane, tab and shift-tab scroll their columns.
func keysFor(m Model, keys string) string {
	if _, before, after := table.Scrolled(&m); m.inRows && (before || after) {
		keys = strings.Replace(keys, "↑↓ move ", "↑↓ move ⇥·⇧⇥ columns ", 1)
	}
	if m.inRows && len(m.lines()) > m.rowsHeight() {
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
		{[]tea.Msg{tab, tab, down, down}, opens},
		// past the headings, jig's #7, and kit's heading, to #45, which [ and ] step from group to group
		{[]tea.Msg{down, down, down, down, down, down}, strings.Replace(opens, "↑↓ move ", "↑↓ move [] group ", 1)},
		{[]tea.Msg{tab, tab, down, down, right}, closes},
		{[]tea.Msg{press('?')}, barKeys},
		{[]tea.Msg{press('?'), press(tea.KeyPgDown)}, barKeys},
	} {
		for _, width := range []int{80, 120, 200} {
			// each load is a model of its own, so the sizes load side by side
			t.Run(fmt.Sprintf("%v at %d", c.msgs, width), func(t *testing.T) {
				t.Parallel()
				for _, height := range []int{24, 12, 8, 7, 6, 5, 4, 3, 2, 1} {
					m := flat(t, options(), append([]tea.Msg{tea.WindowSizeMsg{Width: width, Height: height}},
						c.msgs...)...)
					// a bar too narrow for every key keeps the screen's and drops the next control's, … in its place
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
	m := flat(t, options(), tea.WindowSizeMsg{Width: 30, Height: 12})
	if got := lines(m); len(got) != 12 || !strings.HasPrefix(got[11], " … │ ⇥·⇧⇥ tab ↓ groups ") ||
		ansi.StringWidth(got[11]) > 30 || !strings.HasSuffix(got[11], "…") {
		t.Errorf("the bar at 30 columns: %q", got[len(got)-1])
	}
}

func TestAStageThatOutlastsTheLimitIsARefusalAndRRunsItAgain(t *testing.T) {
	t.Setenv("FAKE_FLOW_SLOW", "releases")
	m := New(runner(t), options())
	m.limit = 100 * time.Millisecond
	m = drive(m, tea.WindowSizeMsg{Width: 120, Height: 24}, tab, tab, tab, tab)
	start := time.Now()
	msg := m.load("releases", false)()
	// the fake sleeps five seconds: the answer comes at the limit only when the sleep is killed with the kit
	if took := time.Since(start); took > time.Second {
		t.Fatalf("the stage did not end at the limit: %s", took)
	}
	m = drive(m, msg)
	v := view(m)
	if !shows(m, Release, "!") ||
		!strings.Contains(v, Failed+" kit flow releases --json took longer than 2 minutes; the menu's Refresh All runs it "+
			"again") {
		t.Fatalf("the stage past its limit:\n%s", v)
	}
	t.Setenv("FAKE_FLOW_SLOW", "")
	// the run again has the limit a stage has, not the tenth of a second the first had: the fake kit is a bash script,
	// and on a busy machine its start alone outlasts that
	m.limit = deadline
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
	// the fake records its call before it sleeps: ctrl-q comes once it runs
	for start := time.Now(); calls(t, record) == nil; time.Sleep(10 * time.Millisecond) {
		if time.Since(start) > 2*time.Second {
			t.Fatal("the stage did not start")
		}
	}
	m = drive(m, tea.KeyPressMsg{Code: 'q', Mod: tea.ModCtrl})
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
	if first := lines(m)[0]; first != "  "+shell.App+" kit flow · "+shell.Workspaces+" empowerite/kit" {
		t.Errorf("the title: %q", first)
	}
	// the dot between the title's parts is dim, so the menus' glyphs stand out
	if !strings.Contains(m.View().Content, style.Off("·")+" "+shell.Workspaces+" "+style.Repo("empowerite/kit")) {
		t.Errorf("the repository is not in its own color: %q", lines(m)[0])
	}
	// the bar comes right under the title
	if !strings.Contains(lines(m)[1], "➊") || !shows(m, Fetched, "7") {
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
	m = loaded(t, o, tab, tab, tab)
	if first := lines(m)[0]; first != "  "+shell.App+" kit flow · "+shell.Workspaces {
		t.Errorf("the title with --no-fetch: %q", first)
	}
	// without the fetch, each remote's row says not fetched, the local branch behind stands as the last fetch left it,
	// and the tab counts that branch alone
	pane := view(m)
	if strings.Count(pane, "main  not fetched") != 3 || !strings.Contains(pane, Behind+"    origin    main  behind") {
		t.Errorf("the incoming pane with --no-fetch:\n%s", pane)
	}
	if !shows(m, Fetched, "1") {
		t.Errorf("the incoming label with --no-fetch counts the branch behind alone: %q", lines(m)[2])
	}
	if !strings.Contains(m.View().Content, style.Off("not fetched")) {
		t.Errorf("not fetched is not dim")
	}
	// with no remote at all, the pane says so, fetched or not: there is none to fetch
	setRows(t, &m)
	if !strings.Contains(view(m), "   no remote to fetch") {
		t.Errorf("no remote with --no-fetch:\n%s", view(m))
	}
	m = loaded(t, Options{App: shell.Flow, Stages: []string{"releases"}})
	if first := lines(m)[0]; first != "  "+shell.App+" kit flow · "+shell.Workspaces {
		t.Errorf("the title alone: %q", first)
	}
}

func TestIncomingIsATableOfEverythingIncoming(t *testing.T) {
	m := loaded(t, options(), tab, tab, tab)
	// a row a thing incoming, kit's order, the primary's first: each remote itself, each commit, branch, and tag, and
	// each local branch behind its upstream
	want := []string{
		Remote + "    origin    main             fetched",
		Commit + "    origin    main             new        60m                   #1 docs:",
		Commit + "    origin    main             new         2h                   #1 yaml:",
		Branch + "    origin    42-settings      new        30m                   #42 settings:",
		Branch + "    origin    46-a-noun-lists  updated    10m  46-a-noun-lists  #46 lists:",
		Behind + "    origin    main             behind     60m  main             #1 docs:",
		Remote + "    upstream  main             fetched",
		Branch + "    upstream  old-experiment   " + Gone + " gone    10d                   an experiment",
		Tag + "    upstream  release/0.2.0    new        24h                   release 0.2.0",
		Remote + "    backup    main             unchanged",
	}
	at := slices.IndexFunc(lines(m), func(l string) bool { return strings.HasPrefix(strings.TrimSpace(l), Remote) })
	if at < 0 {
		t.Fatalf("no remote's row:\n%s", view(m))
	}
	for i, text := range want {
		// a row that has more to show opens, and says so with its fold
		if got := strings.TrimPrefix(strings.TrimSpace(lines(m)[at+i]), "▸ "); !strings.HasPrefix(got, text) {
			t.Errorf("row %d is not %q: %q", i, text, got)
		}
	}
	// grouped by remote, the groups stand as kit lists the remotes, the primary first, not sorted; grouped by kind, as
	// the catalog lists the kinds
	base := m.panel(m.stage)
	headings := func() []string {
		var out []string
		for _, r := range m.rows(m.stage) {
			if grid.Heads(r) && !grid.OverGroups(r) {
				out = append(out, strings.TrimSpace(ansi.Strip(r.Cells[0].Values[0])))
			}
		}
		return out
	}
	if m.setPanel(base.Grouped("remote", false)); !slices.Equal(headings(), []string{"▾ origin 6", "▾ upstream 3",
		"▾ backup 1"}) {
		t.Errorf("by remote: %q", headings())
	}
	if m.setPanel(base.Grouped("kind", false)); !slices.Equal(headings(), []string{"▾ " + Remote + " 3",
		"▾ " + Commit + " 2", "▾ " + Branch + " 3", "▾ " + Tag + " 1", "▾ " + Behind + " 1"}) {
		t.Errorf("by kind: %q", headings())
	}
	m.setPanel(base)
	// the count leaves out the remotes' own rows: seven things came in
	if !shows(m, Fetched, "7") || m.count(3) != "7" {
		t.Errorf("incoming does not count what came in alone: %q", lines(m)[2])
	}
	// a remote whose fetch failed says so in its row, with git's line
	setRows(t, &m, remoteRow("upstream", "remote", "failed", "fatal: could not read from remote repository"))
	if v := view(m); !strings.Contains(v, Failed+" failed") || !strings.Contains(v, "fatal: could not read from") ||
		m.count(3) != "0" {
		t.Errorf("a failed fetch: count %s\n%s", m.count(3), v)
	}
	// with every remote unchanged, each says so and the count is zero
	setRows(t, &m, remoteRow("origin", "remote", "unchanged", ""), remoteRow("upstream", "remote", "unchanged", ""),
		remoteRow("backup", "remote", "unchanged", ""))
	if v := view(m); strings.Count(v, "unchanged") != 3 || m.count(3) != "0" {
		t.Errorf("three remotes unchanged: count %s\n%s", m.count(3), v)
	}
	// a row's key the catalog does not name stands under more, which Everything a click away shows
	if got := m.records(*m.find("incoming")); len(got) != 3 || got[0].values["more"].Raw != nil {
		t.Errorf("unchanged rows say more: %+v", got)
	}
	setRows(t, &m, map[string]any{"key": "origin x", "remote": "origin", "kind": "commit", "signed": true,
		"note": map[string]any{"by": "ci"}, "empty": []any{}})
	if got := m.records(*m.find("incoming"))[0].values["more"].Raw; !slices.Equal(got,
		[]string{`note: {"by":"ci"}`, "signed: true"}) {
		t.Errorf("more: %q", got)
	}
	// with no remote at all, the pane says so
	setRows(t, &m)
	if !strings.Contains(view(m), "   no remote to fetch") {
		t.Errorf("no remote:\n%s", view(m))
	}
}

// remoteRow gives a row of incoming's: a remote's, of a kind and a change, its subject, as kit prints one.
func remoteRow(remote, kind, change, subject string) map[string]any {
	r := map[string]any{"key": remote + " " + kind, "remote": remote, "kind": kind, "change": change, "ref": "main"}
	if subject != "" {
		r["subject"] = subject
	}
	return r
}

// setRows answers incoming with the rows, as kit prints them, and draws them.
func setRows(t *testing.T, m *Model, rows ...map[string]any) {
	t.Helper()
	if rows == nil {
		rows = []map[string]any{}
	}
	data, err := json.Marshal(map[string]any{"fetched": true, "primary": "origin", "remotes": []any{}, "rows": rows})
	if err != nil {
		t.Fatal(err)
	}
	*m = drive(*m, reanswer(*m, "incoming", string(data)))
}

func TestChangesCarryTheirChecksThreadsAndIssues(t *testing.T) {
	// a change's row is wide: at 80 columns its right end is clipped, so the screen is wider here
	m := flat(t, options(), tea.WindowSizeMsg{Width: 160, Height: 24}, tab, down, down, toggle)
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
	// an issue kit gives no age leaves its age's cell empty, so its column can be left out
	s := m.find("changes")
	of[pull](*s)[0].Closes[0].Age = nil
	if c := m.changeRows(*s)[0].Detail[0][3]; len(c.Values) != 0 {
		t.Errorf("the age of an issue without one: %q", c.Values)
	}
}

func TestTheBacklogOpensToItsDetail(t *testing.T) {
	m := loaded(t, options(), tea.WindowSizeMsg{Width: 80, Height: 24})
	m.apply(m.viewsOf(0).Chosen("Backlog"))
	// the source, organization, repository, and labels have tests of their own, so the rows stand ungrouped
	m.setPanel(m.panel(m.stage).Placed("source", grid.Hidden).Placed("organization", grid.Hidden).Placed("repository",
		grid.Hidden).Placed("labels", grid.Hidden))
	m = drive(intoRows(m), down, down, toggle)
	v := view(m)
	// the title has what the other columns leave, with its glyph, which leads to the issue's page, and no branch, which
	// no issue in the backlog has; two assignees are the first and +1, as any list's are; every row's assignee draws
	// Owner, which then says nothing, so the names stand alone
	for _, want := range []string{"#45", "#46  jeffrey-aguilera   2d  " + Issue + " A noun lists",
		"#7   ada           +1  40d", "picture: five stages,", "interactive on a terminal,"} {
		if !strings.Contains(v, want) {
			t.Errorf("the backlog lacks %q:\n%s", want, v)
		}
	}
	// the issue's page is its title's glyph's, not a line of its own, and nothing says how to start it
	rows := m.rows(0)
	k := slices.IndexFunc(rows, func(r row) bool { return r.Key == "#45" })
	if strings.Contains(v, "https://") || strings.Contains(v, "kit work 45") || k < 0 ||
		last(rows[k].Cells).Link != "https://github.com/empowerite/kit/issues/45" {
		t.Errorf("the backlog's links:\n%s", v)
	}
	// open, the row of two assignees names each down its column, one to a line, as a row of labels does
	if v = view(drive(m, toggle, up, up, toggle)); !strings.Contains(v, "#7   ada") || !strings.Contains(v, "     bob") ||
		strings.Contains(v, "ada bob") || strings.Contains(v, "+1") {
		t.Errorf("#7 open:\n%s", v)
	}
}

// A release is one row of its fields, as kit flow prints it flat, which the catalog draws: its version, its kind's glyph
// leading to its page, its age, the latest, its assets, how many, or the one, leading to where it downloads, and its
// notes' first line; open, the notes line by line and the assets down their column. Flattened, a row stands once per
// asset, each leading to its own download. The name, the tag, the date, the page, and the downloads are fields a
// preset shows, hidden by default.
func TestAReleaseIsARowOfItsFieldsLeadingToItsPageAndItsAssets(t *testing.T) {
	m := intoRows(loaded(t, options(), tea.WindowSizeMsg{Width: 120, Height: 30}, tab, tab, tab, tab))
	v := view(m)
	for _, want := range []string{"0.2.0       " + Draft + " draft", "0.1.0       " + Release + " release",
		Latest + " latest", "📎 kit-0.1.0.tar.gz     +1", "## What's Changed", "0.1.0-rc.7  " + Pre + " pre-release  28h",
		"📎 kit-0.1.0-rc.7.tar.gz", "The last candidate."} {
		if !strings.Contains(v, want) {
			t.Errorf("the releases lack %q:\n%s", want, v)
		}
	}
	if strings.Contains(v, "https://") || strings.Contains(v, "kit 0.1.0") || strings.Contains(v, "2026-10-31") {
		t.Errorf("a hidden field shows:\n%s", v)
	}
	page := "https://github.com/empowerite/kit/releases/"
	download := page + "download/release/"
	links := func(m Model, key string) (out []string) {
		for _, r := range m.rows(m.stage) {
			if r.Key == key {
				for _, c := range r.Cells {
					if c.Link != "" {
						out = append(out, c.Link)
					}
				}
			}
		}
		return out
	}
	if got := links(m, "release/0.1.0"); !slices.Equal(got, []string{page + "tag/release/0.1.0"}) {
		t.Errorf("0.1.0 leads to %q", got)
	}
	if got := links(m, "release/0.1.0-rc.7"); !slices.Equal(got, []string{page + "tag/release/0.1.0-rc.7",
		download + "0.1.0-rc.7/kit-0.1.0-rc.7.tar.gz"}) {
		t.Errorf("0.1.0-rc.7 leads to %q", got)
	}
	// open, the notes line by line, and the assets down their column
	m = drive(m, down, toggle)
	v = view(m)
	for _, want := range []string{"📎 kit-0.1.0.tar.gz", "📎 kit-0.1.0.tar.gz.sha256", "* #42 settings",
		"* #46 lists"} {
		if !strings.Contains(v, want) {
			t.Errorf("0.1.0 open lacks %q:\n%s", want, v)
		}
	}
	// flattened, a row per asset, each to its own download
	m.setPanel(m.panel(m.stage).Flattened("assets", true))
	var got []string
	for _, r := range m.rows(m.stage) {
		if r.Key == "release/0.1.0" {
			for _, c := range r.Cells {
				if strings.HasPrefix(c.Link, download) {
					got = append(got, c.Link)
				}
			}
		}
	}
	slices.Sort(got)
	if !slices.Equal(got, []string{download + "0.1.0/kit-0.1.0.tar.gz", download + "0.1.0/kit-0.1.0.tar.gz.sha256"}) {
		t.Errorf("0.1.0 flattened by its assets leads to %q", got)
	}
	for _, id := range []string{"name", "tag", "at", "url", "downloads"} {
		if f := fieldIn("releases", id); f.Placement != table.PlaceHidden {
			t.Errorf("releases' %s: %+v", id, f)
		}
	}
}

func TestThePaneScrollsToKeepTheCursorWhileTheRestStays(t *testing.T) {
	m := flat(t, bare(), tea.WindowSizeMsg{Width: 80, Height: 10 + titleHeight}, tab, tab, down, down, down, down)
	got := lines(m)
	if len(got) != 10+titleHeight {
		t.Fatalf("%d lines on a screen of %d:\n%s", len(got), 10+titleHeight, view(m))
	}
	if !strings.HasPrefix(got[0], "  "+shell.App+" kit flow") || !shows(m, Build, "4") ||
		got[9+titleHeight] != " "+at80(keysFor(m, opens)) {
		t.Errorf("the title, the bar, or the keys moved:\n%s", view(m))
	}
	// three lines of pane: ⬆₂, the cursor's row, and the last row, which needs no line about hidden rows
	if !strings.Contains(got[6+titleHeight], "⬆₂") || !strings.Contains(got[7+titleHeight], "#159") ||
		!strings.Contains(cursor(m), "#159") || !strings.Contains(got[8+titleHeight], "nightly") ||
		strings.Contains(view(m), "release/0.1.0") {
		t.Errorf("the pane did not scroll to the cursor:\n%s", view(m))
	}
	// a pane of three lines holds the … lines and the row; one more line shows what the row opens to
	m = drive(m, toggle, tea.WindowSizeMsg{Width: 80, Height: 12 + titleHeight})
	if !strings.Contains(view(m), "an underway") || openRows(m) == 0 {
		t.Errorf("the open row's detail is not shown with its row:\n%s", view(m))
	}
	m = drive(m, tea.WindowSizeMsg{Width: 80, Height: 10 + titleHeight})
	// a page is the pane less one row: two rows here
	m = drive(m, press(tea.KeyPgUp))
	if !m.inRows || m.find("builds").Cursor != 1 {
		t.Errorf("page up: in rows %v, cursor %d", m.inRows, m.find("builds").Cursor)
	}
	// a page never leaves the rows: up past the first row stays on it, at the top
	m = drive(m, press(tea.KeyPgUp))
	if !m.inRows || m.find("builds").Cursor != 1 || !strings.Contains(view(m), "release/0.1.0") {
		t.Errorf("page up past the first row left it, or the top:\n%s", view(m))
	}
	// on the bar, a page does nothing
	if b := drive(m, up, press(tea.KeyPgDown), press(tea.KeyPgUp)); b.inRows || b.find("builds").Cursor != 0 {
		t.Errorf("a page on the bar: in rows %v, cursor %d", b.inRows, b.find("builds").Cursor)
	}
	m = drive(m, press(tea.KeyPgDown))
	if !m.inRows || m.find("builds").Cursor != 3 {
		t.Errorf("page down: in rows %v, cursor %d", m.inRows, m.find("builds").Cursor)
	}
	// #159, open, takes four lines, and the pane shows one between its … lines: a page down for each of the three
	// under its first, one onto the last row, and one more, which stays there
	for k, want := range []int{3, 3, 3, 4, 4} {
		if m = drive(m, press(tea.KeyPgDown)); m.find("builds").Cursor != want {
			t.Errorf("page down %d through #159 and past the last row: cursor %d, not %d", k+1,
				m.find("builds").Cursor, want)
		}
	}
	// home goes to the first row, at the top, and end to the last; on the bar neither does anything
	if m = drive(m, press(tea.KeyHome)); !m.inRows || m.find("builds").Cursor != 1 ||
		!strings.Contains(view(m), "release/0.1.0") {
		t.Errorf("home: in rows %v, cursor %d:\n%s", m.inRows, m.find("builds").Cursor, view(m))
	}
	if m = drive(m, press(tea.KeyEnd)); !m.inRows || m.find("builds").Cursor != 4 {
		t.Errorf("end: in rows %v, cursor %d", m.inRows, m.find("builds").Cursor)
	}
	if b := drive(m, press(tea.KeyEscape), press(tea.KeyEnd)); b.inRows {
		t.Errorf("end on the bar entered the rows")
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
// the width; at 80 columns or fewer, a long title is clipped. Esc leaves the rows for the bar, whence right is the next
// stage; up would stop on the control line, and right there opens a level.
func everyLineFits(t *testing.T, width int, legend bool) {
	// one load for the five: each stage in turn, its first row open, then back to the bar
	m := loaded(t, options(), tea.WindowSizeMsg{Width: width, Height: 40})
	if legend {
		m = drive(m, press('?'))
	}
	for i := range five {
		if i > 0 {
			m = drive(m, esc, tab)
		}
		if m.stage != i {
			t.Fatalf("at %d, legend %v: on stage %d, not %s's", width, legend, m.stage, five[i])
		}
		m = drive(intoRows(m), toggle)
		clipped := false
		for _, l := range strings.Split(m.View().Content, "\n") {
			if n := ansi.StringWidth(l); n > width {
				t.Errorf("%s at %d, legend %v: a line of %d cells: %q", five[i], width, legend, n, ansi.Strip(l))
			}
			// a clipped row ends in …
			clipped = clipped || strings.Contains(strings.TrimRight(ansi.Strip(l), " "), "…")
		}
		long := five[i] == "changes" || five[i] == "issues"
		if width <= 80 && long && !clipped {
			t.Errorf("%s at %d: no long title was clipped:\n%s", five[i], width, view(m))
		}
	}
}

func TestAnUnknownStageIsRefusedWithoutRunning(t *testing.T) {
	record := filepath.Join(t.TempDir(), "calls")
	t.Setenv("FAKE_FLOW_CALLS", record)
	m := loaded(t, Options{Stages: []string{"changes", "nonesuch"}}, tab)
	if v := view(m); !strings.Contains(squeeze(v), "nonesuch !") ||
		!strings.Contains(v, "kittui does not draw the stage 'nonesuch'") {
		t.Errorf("the unknown stage:\n%s", v)
	}
	if got := calls(t, record); strings.Join(got, "\n") != "flow changes --json" {
		t.Errorf("the calls: %q", got)
	}
	m = loaded(t, Options{})
	if v := view(m); !strings.Contains(v, "no stage is configured") || len(lines(m)) != 24 {
		t.Errorf("no stage at all:\n%s", v)
	}
}

func TestOpenAppReadsTheStagesAndTheWords(t *testing.T) {
	record := filepath.Join(t.TempDir(), "calls")
	t.Setenv("FAKE_FLOW_CALLS", record)
	m, err := OpenApp(runner(t), shell.Delivery, ParseWords([]string{"--no-fetch", "--mine"}))
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, s := range m.stages {
		names = append(names, s.name)
	}
	want := "releases builds delivery/artifacts delivery/environments delivery/deployments"
	if strings.Join(names, " ") != want || !m.noFetch || strings.Join(m.filters, " ") != "--mine" {
		t.Errorf("OpenApp: %q, no fetch %v, filters %q", names, m.noFetch, m.filters)
	}
	if got := calls(t, record); strings.Join(got, "\n") != "flow --stages\ndelivery --stages\ntrust --check" {
		t.Errorf("OpenApp ran %q", got)
	}
	// a stage of the picture named opens on it, maximized, as kit flow <stage> does; the filters still reach the stages;
	// underway, as backlog, opens on issues, set as the preset of its name, which the other stages are not
	for word, preset := range map[string]string{"issues": "All Issues", "underway": "Underway", "backlog": "Backlog"} {
		if m, err = OpenApp(runner(t), shell.Issues, ParseWords([]string{word, "--mine"})); err != nil {
			t.Fatal(err)
		}
		s := m.stages[m.stage]
		if s.name != "issues" || !m.maximized || strings.Join(m.filters, " ") != "--mine" || s.ViewName != preset ||
			m.is(m.stage) != preset || m.stages[1].ViewName != table.Default {
			t.Errorf("OpenApp %s: on %q set as %q, maximized %v, filters %q", word, s.name, s.ViewName, m.maximized,
				m.filters)
		}
		if !strings.Contains(lines(m)[0], "kit issues · ") || !strings.Contains(lines(m)[0], " · ➊ issues") {
			t.Errorf("OpenApp %s, the title: %q", word, lines(m)[0])
		}
		m.Close()
	}
	t.Setenv("FAKE_FLOW_REFUSE", "stages")
	_, err = OpenApp(runner(t), shell.Issues, Words{})
	refusal, ok := err.(*kit.ExitError)
	if !ok || refusal.Code != 2 || refusal.Error() != "kit flow refuses for the test; run: kit help flow" {
		t.Errorf("Open on a refusal: %v", err)
	}
	// Flow's own stages are kit flow's, and its refusal of --stages is the error as well
	if _, err = OpenApp(runner(t), shell.Flow, Words{}); err == nil || err.Error() != refusal.Error() {
		t.Errorf("Open Flow on a refusal: %v", err)
	}
}

func TestTheSelectedRowIsChosen(t *testing.T) {
	// the first row is jig's #7, under its repository's heading
	m := intoRows(loaded(t, options()))
	chosen, _, _ := strings.Cut(style.Mark("#"), "#")
	if !strings.Contains(m.View().Content, chosen+"#") || !strings.HasPrefix(cursor(m), "#7") {
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
	m := flat(t, bare(), tea.WindowSizeMsg{Width: 80, Height: 11 + titleHeight}, tab, tab, down, down, down, toggle)
	got := lines(m)
	if len(got) != 11+titleHeight || got[10+titleHeight] != " "+at80(keysFor(m, closes)) {
		t.Fatalf("the screen is not %d lines with the keys last:\n%s", 11+titleHeight, view(m))
	}
	v := view(m)
	if !strings.Contains(v, "⬆₁") || !strings.Contains(v, "⬇¹²") ||
		!strings.Contains(v, style.Opened+" "+Failed+"  kit") {
		t.Errorf("the open row is not shown between its … lines:\n%s", v)
	}
	// a pane that holds every line says nothing of the kind
	m = flat(t, options(), tab, tab)
	if strings.Contains(view(m), pane.Up) || strings.Contains(view(m), pane.Down) {
		t.Errorf("a pane that fits says lines are hidden:\n%s", view(m))
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
	for _, name := range five {
		if bar := lines(m)[2]; !strings.Contains(bar, name) {
			t.Errorf("at 200, %s is not whole: %q", name, bar)
		}
	}
	if mark, _ := style.In(stageColors["issues"]); !strings.Contains(m.View().Content, Issue+" "+mark("issues")) {
		t.Errorf("the selected name is not marked: %q", lines(m)[2])
	}
	m = drive(m, tab, tab, tab, down)
	dim, _, _ := strings.Cut(style.Off("x"), "x")
	mark, _ := style.In(stageColors["incoming"])
	if bar := strings.Split(m.View().Content, "\n")[2]; !strings.Contains(bar, mark("incoming")) ||
		!strings.Contains(bar, dim+" 🔀 changes") {
		t.Errorf("with the rows' keyboard, the selected stage left its color, or the others are not dim: %q", bar)
	}
	// at 70, ◃ says tabs are hidden on the left; at 58 it is blank, the view starting on a whole tab
	for _, c := range []struct {
		width  int
		before string
	}{{70, " " + tabs.Before}, {58, "  "}} {
		m = drive(m, tea.WindowSizeMsg{Width: c.width, Height: 24})
		bar := lines(m)[2]
		if !strings.Contains(squeeze(bar), "│ "+Fetched+" incoming 7 "+tabs.Menu+" │") || !strings.HasPrefix(bar, c.before) ||
			strings.Contains(bar, "…") {
			t.Errorf("at %d, the bar does not scroll to incoming whole: %q", c.width, bar)
		}
	}
}

func TestPageDownScrollsThroughAnOpenRowTallerThanThePaneBeforeMovingOn(t *testing.T) {
	// builds is the third stage; its first row, a run of ten jobs, opened, takes about three panes on a short screen
	m := flat(t, bare(), tea.WindowSizeMsg{Width: 120, Height: 12})
	m = drive(m, tallBuild(m, 10), tab, tab, down, down, toggle)
	if m.stage != 2 || m.find("builds").Cursor != 1 || !strings.Contains(view(m), pane.Down) {
		t.Fatalf("the open row does not overflow the pane:\n%s", view(m))
	}
	bare := func(s string) string {
		return strings.TrimSpace(strings.TrimLeft(strings.TrimSpace(ansi.Strip(s)), style.Focus+style.Opened))
	}
	head := "release/0.1.0"
	seen := map[string]bool{}
	for k := 0; k < 20 && m.find("builds").Cursor == 1; k++ {
		for _, l := range lines(m) {
			seen[bare(l)] = true
		}
		// scrolled into the row, its own line stays on the pane's first line, where ⬆ would be
		if first := lines(m)[m.rowsTop()]; m.find("builds").Offset > 0 && (!strings.Contains(first, head) ||
			strings.Contains(first, pane.Up)) {
			t.Errorf("the open row's line is not pinned atop the pane:\n%s", view(m))
		}
		m = drive(m, press(tea.KeyPgDown))
	}
	// every line the open row holds was shown on the way down, and only then did the cursor move on
	for _, l := range m.lines() {
		if l.Row == 1 && !seen[bare(l.Text)] {
			t.Errorf("page down never showed %q", ansi.Strip(l.Text))
		}
	}
	if m.find("builds").Cursor == 1 {
		t.Fatalf("page down never left the open row")
	}
	// past the row, the pane says … above again
	if !strings.Contains(lines(m)[m.rowsTop()], pane.Up) {
		t.Errorf("past the open row, the pane's first line is not … above:\n%s", view(m))
	}
	// page up comes back through the same row, to the top of the pane
	for k := 0; k < 20 && (m.find("builds").Cursor != 1 || m.find("builds").Offset > 0); k++ {
		m = drive(m, press(tea.KeyPgUp))
	}
	if m.find("builds").Cursor != 1 || m.find("builds").Offset != 0 || !strings.Contains(lines(m)[m.rowsTop()], head) {
		t.Errorf("page up did not return to the open row's first line:\n%s", view(m))
	}
}

func TestBuildsCarryTheirStateAndOpenToTheirCommitAttemptJobsFailedFirstThenTheirURL(t *testing.T) {
	// builds is the seventh stage, between incoming and releases
	m := flat(t, bare(), tab, tab)
	if m.stages[m.stage].name != "builds" || !shows(m, Build, "4") {
		t.Fatalf("builds is not the seventh stage:\n%s", view(m))
	}
	rows := lines(m)[m.rowsTop() : m.rowsTop()+4]
	for i, want := range [][]string{
		{Running + "  release  #7    tag       release/0.1.0  1a2b3c4", "5m", "…"},
		{Failed + "  kit      #160  push      main           8e817bf  #45 status: the builds", "2h", "4m 02s"},
		{Passed + "  kit      #159  push      main           5d376dd  #49 status: an underway stage", "24h", "3m 36s"},
		{Cancelled + "  nightly  #12   schedule  main           5d376dd", "2d", "12s"},
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
	m = drive(m, down, down, down, toggle)
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
		m.rows(slices.Index(five, "builds"))[b.Cursor].Cells[0].Link != "https://github.com/empowerite/kit/actions/runs/41309409467" {
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
	m = drive(m, reanswer(m, "builds", "[]"))
	if !strings.Contains(view(m), "   no build since the last 20 runs") || !shows(m, Build, "0") {
		t.Errorf("the empty builds:\n%s", view(m))
	}
}

func TestBuildsRefuseADocumentWithoutIDs(t *testing.T) {
	t.Setenv("FAKE_FLOW_ABOVE", "builds")
	m := loaded(t, options(), tab, tab)
	want := "kit flow builds --json printed a record without .id"
	if m.stages[2].refusal != want || !shows(m, Build, "!") {
		t.Errorf("desks for builds: %q\n%s", m.stages[2].refusal, view(m))
	}
	if err := (&stage{name: "builds"}).decode([]byte(`{"id": 1}`), false); err == nil ||
		!strings.Contains(err.Error(), "printed an object where an array was expected") {
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
	// issues is the first stage, set as Underway: an issue a pull request closes, once beside each
	m := loaded(t, bare())
	m.apply(m.viewsOf(0).Chosen("Underway"))
	// the source, organization, repository, and labels have tests of their own, so the rows stand ungrouped
	m.setPanel(m.panel(m.stage).Placed("source", grid.Hidden).Placed("organization", grid.Hidden).Placed("repository",
		grid.Hidden).Placed("labels", grid.Hidden))
	asCatalog(&m, m.stage) // auto_test has what the screen chooses
	if m.stages[m.stage].name != "issues" || !shows(m, Issue, "7") || table.Counted(m.every(0)) != 4 {
		t.Fatalf("issues set as Underway is not the first stage:\n%s", view(m))
	}
	rows := lines(m)[m.rowsTop() : m.rowsTop()+4]
	for i, want := range [][]string{
		{"#1   " + Owner + " jeffrey-agui…", "8d", Change + " #2 ", "🏁 ready", Branch + " here", Pushed + " 60m",
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
	// the numbers are left-aligned and the ages right-aligned, and the pull requests and where line up
	if cellsOf(rows[3], "#31")[0] != cellsOf(rows[0], "#1")[0] || cellsOf(rows[3], "3h")[0] != cellsOf(rows[0], "8d")[0] {
		t.Errorf("the numbers do not line up on the left, or the ages on the right:\n%s", strings.Join(rows, "\n"))
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
	m = drive(intoRows(m), toggle)
	v := view(m)
	order := []string{style.Opened + " #1",
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
		r := m.rows(0)[m.find("issues").Cursor]
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
	if s := m.find("issues"); s.Open["#8 #4"] || grid.Opens(m.rows(0), nil, s.Cursor, m.paneWidth()) {
		t.Errorf("a row elsewhere with a short title opens: %v", s.Open)
	}
	// the URLs kit prints win, as a merge request's on GitLab
	s := m.find("issues")
	of[issue](*s)[2].URL = "https://git.example.org/g/kit/-/issues/8"
	of[issue](*s)[2].PR.URL = "https://git.example.org/g/kit/-/merge_requests/4"
	m.rebuild()
	if pr, issue := links(m); issue != "https://git.example.org/g/kit/-/issues/8" ||
		pr != "https://git.example.org/g/kit/-/merge_requests/4" {
		t.Errorf("kit's URLs do not win: %q %q", pr, issue)
	}
	// an empty stage, set as Underway, says no row matches
	m = drive(m, reanswer(m, "issues", "[]"))
	if !strings.Contains(view(m), "no row matches the panel") || !shows(m, Issue, "0") {
		t.Errorf("the empty issues:\n%s", view(m))
	}
}

// An issue's row is keyed by its number, and, underway, its pull request's after it: an issue in the backlog is one
// row, one two pull requests close two, and two rows of one key are refused.
func TestIssuesKeyEachRowByTheIssueAndItsPullRequest(t *testing.T) {
	s := stage{name: "issues"}
	if err := s.decode([]byte(`[{"number": 1, "pr": null}, {"number": 2, "pr": {"number": 3}},
		{"number": 2, "pr": {"number": 4}}]`), false); err != nil || len(s.parsed) != 3 || s.parsed[0].key != "1" ||
		s.parsed[1].key != "2 3" || s.parsed[2].key != "2 4" {
		t.Errorf("the keys: %v %+v", err, s.parsed)
	}
	err := (&stage{name: "issues"}).decode([]byte(`[{"number": 1}, {"number": 1, "pr": {}}]`), false)
	if err == nil || !strings.Contains(err.Error(), "printed two records of the key 1") {
		t.Errorf("issues took two rows of one key: %v", err)
	}
	// incoming before its rows is refused
	if err := (&stage{name: "incoming"}).decode([]byte(`{"fetched": true, "commits": []}`), false); err == nil ||
		!strings.Contains(err.Error(), `printed an object without "rows"`) {
		t.Errorf("incoming took the document before its rows: %v", err)
	}
}

// A click on an issue's pull request glyph selects changes, the cursor on that pull request's row; a click on its
// state's glyph opens its page; without changes in the picture, the pull request's glyph opens the page too.
func TestAnIssuesPullRequestLeadsToItsRowUnderChangesAndItsStateToItsPage(t *testing.T) {
	m := loaded(t, bare())
	m.apply(m.viewsOf(0).Chosen("Underway"))
	m = drive(intoRows(m), down)
	at := func(m Model, glyph string) int {
		return cellsOf(lines(m)[m.rowLine(m.find("issues").Cursor)], glyph)[0]
	}
	change, state, y := at(m, "🔀 #4"), at(m, "🔥"), m.rowLine(m.find("issues").Cursor)
	browsed = nil
	if m = drive(m, testkit.Click(state, y)); len(browsed) != 1 || !strings.HasSuffix(browsed[0], "/kit/pull/4") ||
		m.stages[m.stage].name != "issues" {
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
	m = loaded(t, o)
	rows := m.rows(0)
	first := rows[slices.IndexFunc(rows, func(r row) bool { return r.Key == "#1 #2" })]
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

// Issues say which tracker holds each issue, in the backlog and underway: its glyph and the source kit names, dim; a
// source kittui
// has no glyph for by its name alone; and none at all leaves the column out, as an empty one. While one tracker holds
// every issue, the view groups the rows by it as it loads, so it is said once, in its group's heading, as any group's
// value, and the column leaves them; with Auto-group off, it stands under the source header, the glyph every row
// shares left out.
func TestIssuesSayWhichTrackerHoldsEachIssue(t *testing.T) {
	m := loaded(t, options(), tea.WindowSizeMsg{Width: 160, Height: 24})
	for i, name := range []string{"All Issues", "Underway"} {
		if i > 0 {
			folding(&m, true)
			m.apply(m.viewsOf(0).Chosen(name))
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
	source := fieldsOf("issues")[slices.IndexFunc(fieldsOf("issues"), func(f table.Field) bool { return f.ID == "source" })]
	if c := source.Draw([]table.Value{table.Some("gitlab"), table.Some("AB#"), table.Some("")}, ""); c[0].Values[0] != style.Off(GitLab+" gitlab") ||
		c[1].Values[0] != style.Off("AB#") || len(c[2].Values) != 0 {
		t.Errorf("the source cells: %q %q %q", c[0].Values, c[1].Values, c[2].Values)
	}
	s := m.find("issues")
	for k := range of[issue](*s) {
		of[issue](*s)[k].Source = ""
	}
	m.rebuild()
	if strings.Contains(lines(m)[m.top()], "source") {
		t.Errorf("a column no issue has a source in has a header:\n%s", view(m))
	}
}

// last gives a row's last cell, the title's, which stands after every other column.
func last(cells []grid.Cell) grid.Cell { return cells[len(cells)-1] }

// rowLine gives the screen line the first line of the selected stage's row k stands on, as the pane is scrolled, or
// -1 while it is scrolled away.
func (m Model) rowLine(k int) int {
	lines := m.allLines(m.stage)
	_, places := m.laid()
	for r, at := range places {
		if at >= 0 && lines[at].Row == k {
			return m.rowsTop() + r
		}
	}
	return -1
}

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

func TestOverAGroupEveryLoadCarriesTheWorkspace(t *testing.T) {
	record := filepath.Join(t.TempDir(), "calls")
	t.Setenv("FAKE_FLOW_CALLS", record)
	r := runner(t)
	r.Group = "w"
	m := New(r, options())
	m = drive(m, m.Init()())
	got := calls(t, record)
	if len(got) == 0 {
		t.Fatal("nothing loaded")
	}
	for _, c := range got {
		if !strings.HasSuffix(c, " --workspace w") {
			t.Errorf("a call without the workspace: %q", c)
		}
	}
}

// tabsTo gives the tabs from the bar's first stage to the named one, in options()' five.
func tabsTo(name string) []tea.Msg { return slices.Repeat([]tea.Msg{tab}, slices.Index(five, name)) }

// reloaded gives issues alone, answered with the first document, then the second as its latest load's answer: after
// a restart, or, kept, the first is what kit kept and the second its load's.
func reloaded(t *testing.T, kept bool, first, second string) Model {
	t.Helper()
	o := options()
	o.Stages = []string{"issues"}
	a := answer("issues", first)
	a.Kept = kept
	m := drive(New(runner(t), o), a)
	if !kept {
		m.restart("issues")
	}
	return drive(m, source.Answer[[]byte]{Name: "issues", Gen: m.find("issues").src.Gen(), Value: []byte(second)})
}

// A reload whose rows hold null where the rows before held a value shows nothing there: each row is read fresh.
func TestAReloadKeepsNoFieldOfTheRowThatStoodThere(t *testing.T) {
	for _, kept := range []bool{false, true} {
		m := reloaded(t, kept,
			`[{"number":5,"title":"five","branch":"5-five","source":"github"},{"number":7,"title":"seven","branch":null}]`,
			`[{"number":7,"title":"seven","branch":null,"source":null}]`)
		got := m.records(*m.find("issues"))
		if len(got) != 1 || got[0].Key != "#7" || got[0].values["branch"].Raw != nil || got[0].values["source"].Raw != nil {
			t.Errorf("kept %v: the reloaded rows %+v", kept, got)
		}
	}
}

// TestAFreshAnswerKeepsNoFieldOfTheKeptCopy is the kept case of the test above, by the plan's name.
func TestAFreshAnswerKeepsNoFieldOfTheKeptCopy(t *testing.T) {
	m := reloaded(t, true, `[{"number":5,"title":"five","branch":"5-five","source":"github"}]`,
		`[{"number":5,"title":"five","branch":null,"source":null}]`)
	if got := m.records(*m.find("issues")); len(got) != 1 || got[0].values["branch"].Raw != nil ||
		got[0].values["source"].Raw != nil || m.find("issues").src.Stale {
		t.Errorf("the fresh rows over the kept: %+v", got)
	}
}

// Nothing printed, null, or {"rows": null} is a refusal naming the command, and so is a record without its key; an
// empty array is the empty stage.
func TestANullOrEmptyDocumentIsARefusal(t *testing.T) {
	for name, doc := range map[string]string{"issues": "null", "changes": "", "incoming": "null",
		"workspace/tags": `{"rows":null}`, "builds": `[{"number":3}]`} {
		if err := (&stage{name: name}).decode([]byte(doc), false); err == nil ||
			!strings.HasPrefix(err.Error(), command(name)+" printed ") {
			t.Errorf("%s %q: %v", name, doc, err)
		}
	}
	s := stage{name: "issues"}
	if err := s.decode([]byte("[]"), false); err != nil || len(s.rows) != 0 {
		t.Errorf("empty issues: %v, %d rows", err, len(s.rows))
	}
	// a stage the catalog does not name decodes nothing, and its command is its name's
	if err := (&stage{name: "nonesuch"}).decode([]byte("[]"), false); err == nil || command("nonesuch") != "kit nonesuch" {
		t.Errorf("a stage kittui does not draw: %v, %q", err, command("nonesuch"))
	}
	// a row whose member or links are not what kit prints them as is a refusal naming it, as are two rows of one key,
	// and a key of no text; over a group, two members' rows of one key are two rows
	tags := command("workspace/tags")
	unread := " printed the record a, whose member or links cannot be read"
	for doc, want := range map[string]string{
		`{"rows": [{"key": "a", "member": 5}, {"key": "b"}]}`:            tags + unread,
		`{"rows": [{"key": "a", "links": []}]}`:                          tags + unread,
		`{"rows": [{"key": "a", "links": {"tag": 5}}]}`:                  tags + unread,
		`{"rows": [{"key": "a", "member": "m", "links": {"tag": [1]}}]}`: tags + strings.Replace(unread, "a,", "m a,", 1),
		`{"rows": [{"key": "a"}, {"key": "b"}, {"key": "a"}]}`:           tags + " printed two records of the key a",
		`{"rows": [{"key": {"name": "a"}}]}`:                             tags + " printed a record without .key",
	} {
		if err := (&stage{name: "workspace/tags"}).decode([]byte(doc), false); err == nil || err.Error() != want {
			t.Errorf("%s: %v", doc, err)
		}
	}
	s = stage{name: "workspace/tags"}
	err := s.decode([]byte(`{"rows": [{"key": "a", "member": "m", "links": {"tag": ["x", null], "commit": null}}, `+
		`{"key": "a", "member": "n"}]}`), false)
	if err == nil || !strings.Contains(err.Error(), "member or links") {
		t.Errorf("a list of links holding null: %v", err)
	}
	if err := s.decode([]byte(`{"rows": [{"key": "a", "member": "m", "links": {"tag": ["x"], "commit": null}}, `+
		`{"key": "a", "member": "n"}]}`), false); err != nil || len(Model{}.catalogRows(s)) != 2 {
		t.Errorf("two members' rows of one key: %v, %d records", err, len(Model{}.catalogRows(s)))
	}
}

// A row the catalog draws is decoded once, as its answer lands, however many times the screen draws it, its tab's
// count among them.
func TestACatalogRowIsDecodedOnce(t *testing.T) {
	n := 0
	decodeRecord = func(r json.RawMessage) any {
		n++
		return table.Decoded(r)
	}
	defer func() { decodeRecord = table.Decoded }()
	o := options()
	o.Stages = []string{"incoming"}
	m := loaded(t, o)
	rows := len(m.find("incoming").rows)
	for range 3 {
		_ = m.View()
		m.rebuild()
		m = drive(m, down, press(tea.KeySpace), up)
	}
	if rows == 0 || n != rows || m.count(0) != "7" {
		t.Errorf("%d records decoded %d times, the count %s", rows, n, m.count(0))
	}
}

// reanswer gives a stage's answer as its latest load's: the JSON its command printed.
func reanswer(m Model, name, doc string) source.Answer[[]byte] {
	return source.Answer[[]byte]{Name: name, Gen: m.find(name).src.Gen(), Value: []byte(doc)}
}
