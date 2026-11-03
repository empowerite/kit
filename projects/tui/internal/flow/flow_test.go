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

	"github.com/empowerite/kit/projects/tui/internal/kit"
	"github.com/empowerite/kit/projects/tui/internal/style"
	"github.com/empowerite/kit/projects/tui/internal/tabs"
	"github.com/empowerite/kit/projects/tui/internal/testkit"
)

// eight is the picture's default order.
var eight = []string{"backlog", "underway", "desk", "worktrees", "changes", "incoming", "builds", "releases"}

// runner gives a kit.Runner over testdata/fake-kit.
func runner(t *testing.T) kit.Runner {
	t.Helper()
	return testkit.Runner(t, "flow")
}

// options gives the eight stages and origin as a test knows them.
func options() Options {
	return Options{Stages: eight, Origin: Origin{Host: "github.com", Repo: "empowerite/kit"}}
}

// drive sends the messages, running every command each update returns and feeding its messages back, a batch's
// each in turn, as the program would. Quit is noted and dropped. Init is loaded's to send, once.
func drive(m Model, msgs ...tea.Msg) Model {
	for _, msg := range msgs {
		if _, ok := msg.(toggleMsg); ok {
			msg = m.toggleKey()
		}
		m, _ = testkit.Drive(m, nil, msg)
	}
	return m
}

// toggleMsg stands, in drive, for the key that opens the row under the cursor or closes it: toggle.
type toggleMsg struct{}

// toggle opens the row under the cursor when it is closed, with right, and closes it when it is open, with left.
var toggle tea.Msg = toggleMsg{}

// toggleKey gives the key toggle sends for the row under the cursor.
func (m Model) toggleKey() tea.KeyPressMsg {
	s, rows := m.stages[m.stage], m.rows(m.stage)
	if s.cursor < len(rows) && s.open[rows[s.cursor].key] {
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
// between them.
func shows(m Model, glyph, count string) bool {
	re := regexp.QuoteMeta(glyph) + `( \S+)?`
	if count != "" {
		re += " " + regexp.QuoteMeta(count)
	}
	// on a bar wide enough for every tab, so a stage the screen scrolls past still answers
	b := m.bar()
	b.Width = 1000
	return regexp.MustCompile(re + ` ( |│|$)`).MatchString(strings.Split(ansi.Strip(b.View()), "\n")[1])
}

// lines gives the screen's lines without their escapes, and without the rule that ends each of the pane's while the
// legend is hidden: the rows as they are drawn.
func lines(m Model) []string {
	out := strings.Split(raw(m), "\n")
	for k, l := range out {
		if k >= paneTop && k < paneTop+m.paneHeight() && !m.legend {
			for _, end := range []string{show, rule} {
				if cut, ok := strings.CutSuffix(l, end); ok {
					out[k] = strings.TrimRight(cut, " ")
				}
			}
		}
	}
	return out
}

// calls reads the words of every kit call the fake recorded.
func calls(t *testing.T, path string) []string {
	t.Helper()
	got, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	return strings.Split(strings.TrimSpace(string(got)), "\n")
}

func TestIncomingLoadsFirstAndTheRestAfterItWithTheirWords(t *testing.T) {
	record := filepath.Join(t.TempDir(), "calls")
	t.Setenv("FAKE_FLOW_CALLS", record)
	o := options()
	o.NoFetch, o.Filters = true, []string{"--mine", "--owner=bob"}
	m := New(runner(t), o)
	msg := m.Init()()
	if got := calls(t, record); len(got) != 1 || got[0] != "flow incoming --json --no-fetch" {
		t.Fatalf("Init ran %q", got)
	}
	for i := range m.stages {
		if !m.stages[i].loading || m.count(i) != "…" {
			t.Errorf("%s is not loading before incoming answers", m.stages[i].name)
		}
	}
	if !strings.Contains(view(m), "loading") {
		t.Errorf("the pane does not say loading:\n%s", view(m))
	}
	next, cmd := m.Update(msg)
	m = next.(Model)
	batch, ok := cmd().(tea.BatchMsg)
	if !ok || len(batch) != 7 {
		t.Fatalf("incoming's answer did not start the other seven: %T", cmd())
	}
	m = drive(m, batch)
	want := []string{"flow incoming --json --no-fetch", "flow backlog --json --mine --owner=bob",
		"flow underway --json --mine --owner=bob", "flow desk --json", "flow worktrees --json",
		"flow changes --json --mine --owner=bob",
		"flow builds --json", "flow releases --json"}
	if got := calls(t, record); strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Errorf("the calls:\n%s\nwant:\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
	for i := range m.stages {
		if m.stages[i].loading {
			t.Errorf("%s still loads", m.stages[i].name)
		}
	}
	// a second answer from incoming starts nothing again
	if _, cmd := m.Update(loadedMsg{stage: "incoming", data: []byte(`{"fetched": false, "remotes": []}`)}); cmd != nil {
		t.Errorf("incoming's second answer started something")
	}
}

func TestWithoutIncomingEveryStageLoadsAtOnce(t *testing.T) {
	record := filepath.Join(t.TempDir(), "calls")
	t.Setenv("FAKE_FLOW_CALLS", record)
	m := New(runner(t), Options{Stages: []string{"changes", "worktrees"}})
	batch, ok := m.Init()().(tea.BatchMsg)
	if !ok || len(batch) != 2 {
		t.Fatalf("Init did not start both: %T", m.Init()())
	}
	m = drive(m, batch)
	if got := calls(t, record); strings.Join(got, "\n") != "flow changes --json\nflow worktrees --json" {
		t.Errorf("the calls: %q", got)
	}
	if first := lines(m)[0]; first != " kit flow" {
		t.Errorf("the title without incoming: %q", first)
	}
	if labels := lines(m)[2]; !strings.Contains(labels, "changes 3") || !strings.Contains(labels, "worktrees 4") ||
		strings.Index(labels, "changes") > strings.Index(labels, "worktrees") {
		t.Errorf("the labels follow kit's order: %q", labels)
	}
}

func TestLeftAndRightMoveAlongTheBarOnlyWhileItHasTheKeyboard(t *testing.T) {
	m := loaded(t, options())
	if m.stage != 0 || !strings.Contains(m.View().Content, style.Mark("➊")) {
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
	if m.stage != 2 || !strings.Contains(m.View().Content, Desk+" "+style.Mark("desk")) {
		t.Errorf("right twice: stage %d\n%s", m.stage, view(m))
	}
	// inside the rows, left and right are the rows' own, and move nothing
	m = drive(m, right, down, right, left)
	if m.stage != 3 || !m.inRows {
		t.Errorf("left and right inside the rows: stage %d, in rows %v", m.stage, m.inRows)
	}
	// esc gives the bar the keyboard back, and right moves again, onto the next stage's first row
	m = drive(m, esc, right, down)
	if m.stage != 4 || !strings.Contains(view(m), style.Focus+" "+Change+" #2") {
		t.Errorf("the next stage's first row is not under the cursor:\n%s", view(m))
	}
	// from changes, five to the right stops at the end: incoming, builds, releases, and there
	m = drive(m, esc, right, right, right, right, right)
	if m.stage != 7 {
		t.Errorf("right past the end of the bar: %d", m.stage)
	}
}

// A click on a row puts the cursor there and gives the rows the keyboard; a click on the row under the cursor opens it,
// and again closes it. A click on a … line, on the legend, or off the pane does nothing.
func TestAClickOnARowSelectsItAndAgainOpensOrClosesIt(t *testing.T) {
	click := testkit.Click
	m := loaded(t, options(), right, right, right)
	// the screen line the row is drawn on, as the person sees it
	if y := slices.IndexFunc(strings.Split(raw(m), "\n"), func(l string) bool { return strings.Contains(l, "42-settings") }); y != paneTop+1 {
		t.Fatalf("the second row is drawn on line %d, not %d", y, paneTop+1)
	}
	if m = drive(m, click(6, paneTop+1)); !m.inRows || m.find("worktrees").cursor != 1 {
		t.Fatalf("a click on the second row: in rows %v, cursor %d", m.inRows, m.find("worktrees").cursor)
	}
	if m = drive(m, click(6, paneTop+1)); !strings.Contains(view(m), style.Opened+" "+Branch+" 42-settings") {
		t.Errorf("a click on the row under the cursor did not open it:\n%s", view(m))
	}
	if m = drive(m, click(6, paneTop+1)); strings.Contains(view(m), style.Opened) {
		t.Errorf("a second click did not close it:\n%s", view(m))
	}
	if m = drive(m, click(6, paneTop)); m.find("worktrees").cursor != 0 || m.stage != 3 {
		t.Errorf("a click on the first row: cursor %d, stage %d", m.find("worktrees").cursor, m.stage)
	}
	before := view(m)
	m = drive(m, press('?'))
	for _, c := range []tea.Msg{click(m.paneWidth()+4, paneTop+1), click(6, 0), click(6, 23),
		tea.MouseClickMsg{X: 6, Y: paneTop + 1, Button: tea.MouseRight}} {
		if m = drive(m, c); m.find("worktrees").cursor != 0 || strings.Contains(view(m), style.Opened) {
			t.Errorf("%v moved the cursor or opened a row:\n%s", c, view(m))
		}
	}
	if m = drive(m, press('?')); view(m) != before {
		t.Errorf("the clicks changed the screen:\n%s", view(m))
	}
	// on a short pane, the … line above is no row's
	m = loaded(t, options(), tea.WindowSizeMsg{Width: 80, Height: 9}, right, right, right, down, down, down)
	if m.find("worktrees").cursor != 2 || !strings.Contains(lines(m)[paneTop], "above") {
		t.Fatalf("the short pane: cursor %d\n%s", m.find("worktrees").cursor, view(m))
	}
	if m = drive(m, click(6, paneTop)); m.find("worktrees").cursor != 2 {
		t.Errorf("a click on … above moved the cursor to %d:\n%s", m.find("worktrees").cursor, view(m))
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
	if m.inRows || strings.Contains(view(m), style.Focus) {
		t.Fatalf("on the bar, a row has the focus:\n%s", view(m))
	}
	m = drive(m, down)
	if !m.inRows || !strings.Contains(view(m), " "+style.Focus+" "+Branch+" 1-the-first-release") {
		t.Fatalf("down did not enter the rows:\n%s", view(m))
	}
	m = drive(m, down)
	if !strings.Contains(view(m), style.Focus+" "+Branch+" 42-settings") {
		t.Errorf("down once more:\n%s", view(m))
	}
	m = drive(m, down, down, down)
	if m.find("worktrees").cursor != 3 {
		t.Errorf("down past the last row: %d", m.find("worktrees").cursor)
	}
	m = drive(m, up, up, up)
	if !m.inRows || m.find("worktrees").cursor != 0 {
		t.Errorf("up to the first row: in rows %v, cursor %d", m.inRows, m.find("worktrees").cursor)
	}
	m = drive(m, up)
	if m.inRows || strings.Contains(view(m), style.Focus) {
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
	if !m.find("worktrees").open["/home/kit/worktrees/1-the-first-release-v0-1-0"] {
		t.Errorf("the row is not open by its key: %v", m.find("worktrees").open)
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
	if n := len(lines(m)); n != 24 || len(m.find("incoming").open) != 0 {
		t.Errorf("a leaf opened: %d lines, open %v", n, m.find("incoming").open)
	}
}

func TestRefreshRunsTheStageAgain(t *testing.T) {
	record := filepath.Join(t.TempDir(), "calls")
	t.Setenv("FAKE_FLOW_CALLS", record)
	m := loaded(t, options(), right, right, right, right, right, right, right)
	next, cmd := m.Update(press('r'))
	m = next.(Model)
	if cmd == nil || !strings.Contains(view(m), "loading") || !shows(m, Release, "…") {
		t.Fatalf("r did not run releases again:\n%s", view(m))
	}
	m = drive(m, cmd())
	if got := calls(t, record); len(got) != 9 || got[8] != "flow releases --json" {
		t.Errorf("the calls: %q", got)
	}
	if !shows(m, Release, "3") {
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
	m = drive(m, right, press('r'))
	if v := view(m); !shows(m, Change, "3") || strings.Contains(v, "cannot be read") {
		t.Errorf("r did not try again:\n%s", v)
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
	m := loaded(t, options(), right, right, right, right, right)
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

func TestEscWalksBackFromTheLegendToTheRowsToTheBar(t *testing.T) {
	m := loaded(t, options(), right, right, right, down, down, press('?'))
	if !m.legend || !strings.Contains(view(m), hide+" legend") {
		t.Fatalf("? did not show the legend:\n%s", view(m))
	}
	next, cmd := m.Update(press(tea.KeyEscape))
	if m = next.(Model); cmd != nil || m.legend || !m.inRows || m.find("worktrees").cursor != 1 {
		t.Fatalf("esc with the legend on: legend %v, in rows %v, cursor %d", m.legend, m.inRows, m.find("worktrees").cursor)
	}
	if v := view(m); !strings.Contains(v, style.Focus+" "+Branch+" 42-settings") || strings.Contains(v, hide+" legend") {
		t.Errorf("esc with the legend on did not take it away and keep the rows as they were:\n%s", v)
	}
	next, cmd = m.Update(press(tea.KeyEscape))
	if m = next.(Model); cmd != nil || m.inRows || m.stage != 3 || strings.Contains(view(m), style.Focus) {
		t.Fatalf("esc in the rows: in rows %v, stage %d\n%s", m.inRows, m.stage, view(m))
	}
	// on the bar, esc takes the legend away, and then does nothing
	m = loaded(t, options(), press('?'), press(tea.KeyEscape))
	if m.legend || m.ctx.Err() != nil {
		t.Errorf("esc on the bar with the legend on: legend %v, context %v", m.legend, m.ctx.Err())
	}
}

// The keys bar on the bar, over the legend there, and in the rows, on a row that opens, closed and open.
const (
	barKeys    = "←→ tab · ↓ rows · 1-9 jump · r refresh · ? legend · ctrl+c quit"
	legendKeys = "←→ tab · ↓ rows · 1-9 jump · r refresh · ? legend · esc back"
	opens      = "↑↓ move · pgup/pgdn page · → open · r refresh · ? legend · esc back"
	closes     = "↑↓ move · pgup/pgdn page · ← close · r refresh · ? legend · esc back"
)

func TestTheKeysBarIsTheLastLineInEveryState(t *testing.T) {
	for _, c := range []struct {
		msgs []tea.Msg
		keys string
	}{
		{nil, barKeys},
		{[]tea.Msg{down}, opens},
		{[]tea.Msg{down, down, down, down}, opens},
		{[]tea.Msg{right, right, right, down, right}, closes},
		{[]tea.Msg{press('?')}, legendKeys},
		{[]tea.Msg{press('?'), press(tea.KeyPgDown)}, legendKeys},
	} {
		for _, width := range []int{80, 120, 200} {
			// each load is a model of its own, so the sizes load side by side
			t.Run(fmt.Sprintf("%v at %d", c.msgs, width), func(t *testing.T) {
				t.Parallel()
				for _, height := range []int{24, 12, 8, 7, 6, 5, 4, 3, 2, 1} {
					m := loaded(t, options(), append([]tea.Msg{tea.WindowSizeMsg{Width: width, Height: height}},
						c.msgs...)...)
					got := lines(m)
					if len(got) != height || got[height-1] != " "+c.keys {
						t.Errorf("after %v at %d×%d: %d lines, the last %q", c.msgs, width, height, len(got),
							got[len(got)-1])
					}
				}
			})
		}
	}
	// at a width too narrow for it, the bar is clipped, never wrapped
	m := loaded(t, options(), tea.WindowSizeMsg{Width: 30, Height: 12})
	if got := lines(m); len(got) != 12 || !strings.HasSuffix(got[11], "…") || ansi.StringWidth(got[11]) != 30 {
		t.Errorf("the bar at 30 columns: %q", got[len(got)-1])
	}
}

func TestAStageThatOutlastsTheLimitIsARefusalAndRRunsItAgain(t *testing.T) {
	t.Setenv("FAKE_FLOW_SLOW", "releases")
	m := New(runner(t), options())
	m.limit = 100 * time.Millisecond
	m = drive(m, tea.WindowSizeMsg{Width: 120, Height: 24}, right, right, right, right, right, right, right)
	start := time.Now()
	msg := m.load("releases")()
	// the fake sleeps five seconds: the answer comes at the limit only when the sleep is killed with the kit
	if took := time.Since(start); took > time.Second {
		t.Fatalf("the stage did not end at the limit: %s", took)
	}
	m = drive(m, msg)
	v := view(m)
	if !shows(m, Release, "!") ||
		!strings.Contains(v, Failed+" kit flow releases took longer than 2 minutes; r runs it again") {
		t.Fatalf("the stage past its limit:\n%s", v)
	}
	t.Setenv("FAKE_FLOW_SLOW", "")
	m = drive(m, press('r'))
	if v := view(m); !shows(m, Release, "3") || strings.Contains(v, "took longer") {
		t.Errorf("r did not run it again:\n%s", v)
	}
}

func TestQuitKillsTheStagesStillRunning(t *testing.T) {
	record := filepath.Join(t.TempDir(), "calls")
	t.Setenv("FAKE_FLOW_CALLS", record)
	t.Setenv("FAKE_FLOW_SLOW", "releases")
	m := New(runner(t), options())
	answers := make(chan tea.Msg, 1)
	load := m.load("releases")
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
		if err := msg.(loadedMsg).err; !errors.Is(err, context.Canceled) {
			t.Errorf("the stage ended with %v, not the quit", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("the stage outlived the screen")
	}
}

func TestTheTitleCarriesOriginAndTheFetchIsTheIncomingStage(t *testing.T) {
	m := loaded(t, options())
	if first := lines(m)[0]; first != " kit flow · "+Repository+" empowerite/kit" {
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
	m = loaded(t, o, right, right, right, right, right)
	if first := lines(m)[0]; first != " kit flow" {
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
	if first := lines(m)[0]; first != " kit flow" {
		t.Errorf("the title alone: %q", first)
	}
}

func TestIncomingGroupsItsRowsByRemoteThePrimaryFirst(t *testing.T) {
	m := loaded(t, options(), right, right, right, right, right)
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
	if !shows(m, Fetched, "6") || m.count(5) != "6" {
		t.Errorf("incoming does not count its rows alone: %q", lines(m)[2])
	}
	// every row is a leaf, the headings too
	m = drive(m, down, toggle, down, toggle)
	if len(m.find("incoming").open) != 0 {
		t.Errorf("an incoming row opened: %v", m.find("incoming").open)
	}
	// a remote whose fetch failed shows git's line, in red, and is no row of the count
	m.find("incoming").in.Remotes[0].Error = "fatal: could not read from remote repository"
	m.find("incoming").in.Remotes[0].Branches.Gone, m.find("incoming").in.Remotes[0].Tags = nil, nil
	m.rebuild()
	failed := Failed + " fatal: could not read from remote repository"
	if v := view(m); !strings.Contains(v, Remote+" upstream · gitlab · git.example.org\n   "+failed+"\n") ||
		!strings.Contains(m.View().Content, style.Refusal(failed)) || m.count(5) != "4" {
		t.Errorf("a failed fetch: count %s\n%s", m.count(5), v)
	}
	m.find("incoming").in.Remotes[0].Error = ""
	m.rebuild()
	// with every remote unchanged, each says so and the count is zero
	for i := range m.find("incoming").in.Remotes {
		r := &m.find("incoming").in.Remotes[i]
		r.Commits, r.Branches.New, r.Branches.Updated, r.Branches.Gone, r.Tags = nil, nil, nil, nil, nil
	}
	m.rebuild()
	if v := view(m); strings.Count(v, "unchanged") != 3 || m.count(5) != "0" {
		t.Errorf("three remotes unchanged: count %s\n%s", m.count(5), v)
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
	for _, want := range []string{Change + " #2", "👤 jeffrey-aguilera", "🛬 ready", Passed + " 1 / 1 3m",
		"🔨 draft", Running + " 0 / 2", Threads + " 2 unresolved", "🔥 failing:gate", Failed + " lint failing 1 / 2 95s",
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
	m := loaded(t, options(), tea.WindowSizeMsg{Width: 80, Height: 24}, down, toggle)
	v := view(m)
	for _, want := range []string{"#45", "#46  👤 jeffrey-aguilera   2d  " + Branch + " 46-a-noun-lists  " + Issue,
		"#7  👤 ada bob           40d", "interactive on a terminal, and\n", "the listing verbs fold into it",
		"https://github.com/empowerite/kit/issues/45", "kit issue 45 starts it"} {
		if !strings.Contains(v, want) {
			t.Errorf("the backlog lacks %q:\n%s", want, v)
		}
	}
	m = loaded(t, options(), right, right, right, right, right, right, right, down, down, toggle)
	v = view(m)
	for _, want := range []string{"0.2.0       " + Draft + " draft", "0.1.0       " + Release + " release",
		Latest + " latest",
		"0.1.0-rc.7  " + Pre + " pre-release  28h", "kit 0.1.0", "release/0.1.0", "2026-10-31T20:00:00Z",
		"https://github.com/empowerite/kit/releases/tag/release/0.1.0"} {
		if !strings.Contains(v, want) {
			t.Errorf("the releases lack %q:\n%s", want, v)
		}
	}
}

func TestThePaneScrollsToKeepTheCursorWhileTheRestStays(t *testing.T) {
	m := loaded(t, options(), tea.WindowSizeMsg{Width: 80, Height: 9}, right, right, right, down, down, down)
	got := lines(m)
	if len(got) != 9 {
		t.Fatalf("%d lines on a screen of 9:\n%s", len(got), view(m))
	}
	if !strings.HasPrefix(got[0], " kit flow") || !shows(m, Branch, "4") || got[8] != " "+opens {
		t.Errorf("the title, the bar, or the keys moved:\n%s", view(m))
	}
	// three lines of pane: … 2 above, the cursor's row, and the last row, which needs no … line
	if !strings.Contains(got[4], "… 2 above") || !strings.Contains(got[5], style.Focus+" "+Branch+" worktrees/spike") ||
		!strings.Contains(got[6], Gone+" gone:") || strings.Contains(view(m), "1-the-first-release") {
		t.Errorf("the pane did not scroll to the cursor:\n%s", view(m))
	}
	// a pane of three lines holds the … lines and the row; one more line shows what the row opens to
	m = drive(m, toggle, tea.WindowSizeMsg{Width: 80, Height: 10})
	if !strings.Contains(view(m), "?? notes.md") || !strings.Contains(view(m), style.Opened) {
		t.Errorf("the open row's detail is not shown with its row:\n%s", view(m))
	}
	m = drive(m, tea.WindowSizeMsg{Width: 80, Height: 9})
	// a page is the pane less one row: two rows here
	m = drive(m, press(tea.KeyPgUp))
	if !m.inRows || m.find("worktrees").cursor != 0 {
		t.Errorf("page up: in rows %v, cursor %d", m.inRows, m.find("worktrees").cursor)
	}
	// a page never leaves the rows: up past the first row stays on it, at the top
	m = drive(m, press(tea.KeyPgUp))
	if !m.inRows || m.find("worktrees").cursor != 0 || !strings.Contains(view(m), Branch+" 1-the-first-release") {
		t.Errorf("page up past the first row left it, or the top:\n%s", view(m))
	}
	// on the bar, a page does nothing
	if b := drive(m, up, press(tea.KeyPgDown), press(tea.KeyPgUp)); b.inRows || b.find("worktrees").cursor != 0 {
		t.Errorf("a page on the bar: in rows %v, cursor %d", b.inRows, b.find("worktrees").cursor)
	}
	m = drive(m, press(tea.KeyPgDown))
	if !m.inRows || m.find("worktrees").cursor != 2 {
		t.Errorf("page down: in rows %v, cursor %d", m.inRows, m.find("worktrees").cursor)
	}
	m = drive(m, press(tea.KeyPgDown), press(tea.KeyPgDown))
	if m.find("worktrees").cursor != 3 {
		t.Errorf("page down past the last row: cursor %d", m.find("worktrees").cursor)
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
			// a clipped row ends in …, before the rule
			clipped = clipped || strings.Contains(ansi.Strip(l), "… ┊") || strings.Contains(ansi.Strip(l), "… "+show) ||
				strings.Contains(ansi.Strip(l), "… "+hide)
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
	if v := view(m); !strings.Contains(v, "nonesuch !") ||
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
	if !strings.Contains(m.View().Content, " "+style.Focus+" "+style.Chosen("#45")) {
		t.Errorf("the row under the cursor is not drawn as chosen:\n%s", m.View().Content)
	}
}

func TestAgeIsKitStatuss(t *testing.T) {
	for secs, want := range map[int64]string{0: "0s", 45: "45s", 119: "119s", 120: "2m", 7199: "119m", 7200: "2h",
		172799: "47h", 172800: "2d", 3456000: "40d", -5: "0s"} {
		if got := age(secs); got != want {
			t.Errorf("age(%d) = %q, want %q", secs, got, want)
		}
	}
}

func TestAPaneTooShortSaysHowManyLinesAreAboveAndBelow(t *testing.T) {
	m := loaded(t, options(), tea.WindowSizeMsg{Width: 80, Height: 10}, right, right, right, down, down, toggle)
	got := lines(m)
	if len(got) != 10 || got[9] != " "+closes {
		t.Fatalf("the screen is not 10 lines with the keys last:\n%s", view(m))
	}
	v := view(m)
	if !strings.Contains(v, "… 1 above") || !strings.Contains(v, "… 4 more") ||
		!strings.Contains(v, style.Opened+" "+Branch+" 42-settings") {
		t.Errorf("the open row is not shown between its … lines:\n%s", v)
	}
	// a pane that holds every line says nothing of the kind
	m = loaded(t, options(), right, right, right)
	if strings.Contains(view(m), " above") || strings.Contains(view(m), " more") {
		t.Errorf("a pane that fits says lines are hidden:\n%s", view(m))
	}
}

func TestQuestionMarkShowsTheLegendBesideTheRowsAndTheBarStays(t *testing.T) {
	m := loaded(t, options(), tea.WindowSizeMsg{Width: 120, Height: 24}, right, right, right)
	before := lines(m)
	m = drive(m, press('?'))
	got := lines(m)
	if len(got) != 24 || got[0] != before[0] || got[23] != " "+legendKeys {
		t.Errorf("the title or the keys moved under the legend:\n%s", view(m))
	}
	if got[1] != before[1] {
		t.Errorf("the bar moved under the legend:\n%s\n%s", before[1], got[1])
	}
	if !strings.Contains(view(m), hide+" legend") || !strings.Contains(view(m), Branch+" 1-the-first-release") {
		t.Errorf("the legend is not beside the rows:\n%s", view(m))
	}
	// the keys move in the rows while the legend is on
	m = drive(m, down, down)
	if !m.inRows || m.find("worktrees").cursor != 1 || !strings.Contains(view(m), style.Focus+" "+Branch+" 42-settings") {
		t.Errorf("down with the legend on: in rows %v, cursor %d\n%s", m.inRows, m.find("worktrees").cursor, view(m))
	}
	m = drive(m, up, up, press('?'))
	if m.legend || strings.Contains(view(m), hide+" legend") || lines(m)[23] != " "+barKeys ||
		strings.Join(lines(m), "\n") != strings.Join(before, "\n") {
		t.Errorf("? again did not take the legend away:\n%s", view(m))
	}
	// shift-/ comes as ? with its text
	m = drive(m, tea.KeyPressMsg{Code: '/', Mod: tea.ModShift, Text: "?"})
	if !m.legend {
		t.Errorf("shift-/ did not show the legend")
	}
}

func TestWorktreesShowTheirPullRequestsAndTheirUncommittedCountButNotTheDesk(t *testing.T) {
	m := loaded(t, options(), right, right, right)
	rows := lines(m)[4:7]
	for i, want := range [][]string{
		{Branch + " 1-the-first-release-v0-1-0", "↑266 ↓0", Change + " #2 🛬 ready", "3 uncommitted"},
		{Branch + " 42-settings", "↑3 ↓1", "no pull request yet", "clean"},
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
	m := loaded(t, options(), right, right)
	pane := lines(m)[4:]
	for _, text := range []string{Desk + " main", "↑12 ↓1", "2 uncommitted"} {
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
	if !strings.Contains(m.View().Content, style.Warn(" M")+" kit.yml") {
		t.Errorf("an uncommitted path's status is not yellow")
	}
	// a desk on a branch with a pull request shows it, as a worktree does
	m.find("desk").desk.Branch = "1-the-first-release-v0-1-0"
	m.rebuild()
	if !strings.Contains(view(m), Change+" #2 🛬 ready") {
		t.Errorf("the desk does not show its pull request:\n%s", view(m))
	}
	// a clean desk says so, dim
	m.find("desk").desk.Uncommitted = 0
	m.rebuild()
	if !strings.Contains(m.View().Content, style.Off("clean")) {
		t.Errorf("a clean desk does not say clean")
	}
}

func TestAnOpenCheckoutFillsThePaneAndPagesThroughEveryCommit(t *testing.T) {
	m := loaded(t, options(), right, right, down, toggle, tea.WindowSizeMsg{Width: 80, Height: 12})
	got := lines(m)
	// the pane runs to the gap above the keys bar, its last line counting what is below
	if len(got) != 12 || !strings.Contains(got[9], "more") || strings.TrimSpace(got[10]) != "" {
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
		m.find("desk").cursor != 0 || !m.inRows {
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
	if !strings.Contains(m.View().Content, Issue+" "+style.Mark("backlog")) {
		t.Errorf("the selected name is not marked: %q", lines(m)[2])
	}
	m = drive(m, right, right, right, right, right, down)
	dim, _, _ := strings.Cut(style.Off("x"), "x")
	if bar := strings.Split(m.View().Content, "\n")[2]; !strings.Contains(bar, style.Mark("incoming")) ||
		!strings.Contains(bar, dim+" 🎫 backlog 3") {
		t.Errorf("with the rows' keyboard, the selected stage left the accent, or the others are not dim: %q", bar)
	}
	for _, width := range []int{120, 80} {
		m = drive(m, tea.WindowSizeMsg{Width: width, Height: 24})
		bar := lines(m)[2]
		if !strings.Contains(bar, "│ "+Fetched+" incoming 6 │") || !strings.HasPrefix(bar, tabs.Before) ||
			strings.Contains(bar, "…") {
			t.Errorf("at %d, the bar does not scroll to incoming whole: %q", width, bar)
		}
	}
}

func TestPageDownScrollsThroughAnOpenRowTallerThanThePaneBeforeMovingOn(t *testing.T) {
	// worktrees is the fourth stage; its first row, opened, is taller than a pane on a short screen
	m := loaded(t, options(), tea.WindowSizeMsg{Width: 120, Height: 12}, right, right, right, down, toggle)
	if m.stage != 3 || m.find("worktrees").cursor != 0 || !strings.Contains(view(m), " more") {
		t.Fatalf("the open row does not overflow the pane:\n%s", view(m))
	}
	bare := func(s string) string {
		return strings.TrimSpace(strings.TrimLeft(strings.TrimSpace(ansi.Strip(s)), style.Focus+style.Opened))
	}
	head := style.Opened + " " + Branch + " 1-the-first-release"
	seen := map[string]bool{}
	for k := 0; k < 20 && m.find("worktrees").cursor == 0; k++ {
		for _, l := range lines(m) {
			seen[bare(l)] = true
		}
		// scrolled into the row, its own line stays on the pane's first line, where … above would be
		if pane := lines(m)[4]; m.find("worktrees").offset > 0 && (!strings.Contains(pane, head) || strings.Contains(pane, "above")) {
			t.Errorf("the open row's line is not pinned atop the pane:\n%s", view(m))
		}
		m = drive(m, press(tea.KeyPgDown))
	}
	// every line the open row holds was shown on the way down, and only then did the cursor move on
	for _, l := range m.lines(3) {
		if l.row == 0 && !seen[bare(l.text)] {
			t.Errorf("page down never showed %q", ansi.Strip(l.text))
		}
	}
	if m.find("worktrees").cursor == 0 {
		t.Fatalf("page down never left the open row")
	}
	// past the row, the pane says … above again
	if !strings.Contains(lines(m)[4], " above") {
		t.Errorf("past the open row, the pane's first line is not … above:\n%s", view(m))
	}
	// page up comes back through the same row, to the top of the pane
	for k := 0; k < 20 && (m.find("worktrees").cursor != 0 || m.find("worktrees").offset > 0); k++ {
		m = drive(m, press(tea.KeyPgUp))
	}
	if m.find("worktrees").cursor != 0 || m.find("worktrees").offset != 0 || !strings.Contains(lines(m)[4], head) {
		t.Errorf("page up did not return to the open row's first line:\n%s", view(m))
	}
}

func TestBuildsCarryTheirStateAndOpenToTheirCommitAttemptJobsFailedFirstThenTheirURL(t *testing.T) {
	// builds is the seventh stage, between incoming and releases
	m := loaded(t, options(), right, right, right, right, right, right)
	if m.stages[m.stage].name != "builds" || !shows(m, Build, "4") {
		t.Fatalf("builds is not the seventh stage:\n%s", view(m))
	}
	rows := lines(m)[4:8]
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
	// the failed build opens to its commit and title in full, its attempt, its jobs, the failed first with the step
	// that failed under it, then its URL, dim
	m = drive(m, down, down, toggle)
	v := view(m)
	order := []string{style.Opened + " " + Failed + "  kit",
		"8e817bf #45 status: the builds stage tells which commit each run built", "attempt 2", Failed + "  test",
		"failed at bats tests/status.bats", Passed + "  gate", Passed + "  lint", Cancelled + "  docs",
		"https://github.com/empowerite/kit/actions/runs/41309409467", Passed + "  kit"}
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
	if !strings.Contains(m.View().Content, style.Off("https://github.com/empowerite/kit/actions/runs/41309409467")) {
		t.Errorf("the URL is not dim")
	}
	// a first attempt says nothing of it
	if m = drive(m, toggle, down, toggle); strings.Contains(view(m), "attempt") {
		t.Errorf("a first attempt is drawn:\n%s", view(m))
	}
	// a build without jobs, whose title repeats its workflow, opens to its URL alone
	m = drive(m, toggle, down, toggle)
	got := lines(m)
	k := slices.IndexFunc(got, func(l string) bool { return strings.Contains(l, style.Opened+" "+Cancelled) })
	if k < 0 || strings.TrimSpace(got[k+1]) != "https://github.com/empowerite/kit/actions/runs/41309409322" ||
		strings.TrimSpace(got[k+2]) != "" {
		t.Errorf("a build without jobs does not open to its URL alone:\n%s", view(m))
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
	m := loaded(t, options(), right, right, right, right, right, right)
	want := `kit flow builds --json printed an array of objects without "id" where an array of builds was ` +
		"expected; run kit flow inside a repository"
	if m.stages[6].refusal != want || !shows(m, Build, "!") {
		t.Errorf("desks for builds: %q\n%s", m.stages[6].refusal, view(m))
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
		if got := took(c.secs); got != c.want {
			t.Errorf("took(%v) = %q, want %q", c.secs, got, c.want)
		}
	}
}

func TestUnderwayDrawsEachIssueBesideThePullRequestThatClosesIt(t *testing.T) {
	// underway is the second stage, between backlog and incoming
	m := loaded(t, options(), right)
	if m.stages[m.stage].name != "underway" || !shows(m, InFlight, "4") {
		t.Fatalf("underway is not the second stage:\n%s", view(m))
	}
	rows := lines(m)[4:8]
	for i, want := range [][]string{
		{"#1  " + Owner + " jeffrey-aguilera", "8d", Change + " #2 ", "🛬 ready", Desk + " here",
			Issue + " The first release, v0.1.0"},
		{"#8", "2d", Change + " #4 ", "🔥 failing:gate", Owner + " bob", Issue + " Lint the modules"},
		{"#8", "2d", Change + " #11", "🔨 draft", Owner + " ada", Issue + " Lint the modules"},
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
	for _, text := range []string{Change, Owner + " ada"} {
		if slices.Max(cellsOf(rows[2], text)) != slices.Max(cellsOf(rows[3], text)) {
			t.Errorf("%q does not line up:\n%s", text, strings.Join(rows, "\n"))
		}
	}
	// here is plain; another's desk and the assignees are dim; a long title is clipped
	c := m.View().Content
	if !strings.Contains(c, style.Off(Owner+" bob")) || strings.Contains(c, style.Off(Desk+" here")) ||
		!strings.Contains(c, style.Off(Owner+" jeffrey-aguilera")) || !strings.HasSuffix(rows[3], "…") {
		t.Errorf("the where cells or the title:\n%s", strings.Join(rows, "\n"))
	}
	// the row here opens to the issue's title in full, the pull request's, the worktree, and both URLs, dim
	m = drive(m, down, toggle)
	v := view(m)
	order := []string{style.Opened + "  #1",
		"The first release, v0.1.0: the kit verbs, the picture, and the forge seam on GitHub",
		Change + " #2 The first release, v0.1.0", Desk + " /home/kit/worktrees/1-the-first-release-v0-1-0",
		"https://github.com/empowerite/kit/issues/1", "https://github.com/empowerite/kit/pull/2", "#8"}
	at := -1
	for _, want := range order {
		k := strings.Index(v, want)
		if k <= at {
			t.Errorf("the open row lacks %q after the line before it:\n%s", want, v)
		}
		at = k
	}
	for _, url := range []string{"https://github.com/empowerite/kit/issues/1",
		"https://github.com/empowerite/kit/pull/2"} {
		if !strings.Contains(m.View().Content, style.Off(url)) {
			t.Errorf("%s is not dim", url)
		}
	}
	// a row elsewhere opens to no worktree; the issue's other pull request stays closed
	m = drive(m, toggle, down, toggle)
	v = view(m)
	if !strings.Contains(v, Change+" #4 Lint every module") || !strings.Contains(v, "/kit/pull/4") ||
		strings.Contains(v, "/kit/pull/11") || strings.Contains(v, Desk+" /") {
		t.Errorf("the open row elsewhere:\n%s", v)
	}
	if s := m.find("underway"); !s.open["#8 #4"] || s.open["#8 #11"] {
		t.Errorf("the two rows of #8 are not apart: %v", s.open)
	}
	// the URLs kit prints win, as a merge request's on GitLab
	s := m.find("underway")
	s.under[1].URL = "https://git.example.org/g/kit/-/issues/8"
	s.under[1].PR.URL = "https://git.example.org/g/kit/-/merge_requests/4"
	m.rebuild()
	if v := view(m); !strings.Contains(v, "/-/issues/8") || !strings.Contains(v, "/-/merge_requests/4") {
		t.Errorf("kit's URLs do not win:\n%s", v)
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
