// The screen is driven key by key through Update, against testdata/fake-kit, which answers each stage from the
// fixtures beside it, as Bubble Tea would drive it.
package status

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/empowerite/kit/tui/internal/kit"
	"github.com/empowerite/kit/tui/internal/style"
	"github.com/empowerite/kit/tui/internal/testkit"
)

// five is the picture's default order.
var five = []string{"backlog", "incoming", "worktrees", "changes", "releases"}

// runner gives a kit.Runner over testdata/fake-kit.
func runner(t *testing.T) kit.Runner {
	t.Helper()
	return testkit.Runner(t, "status")
}

// options gives the five stages and origin as a test knows them.
func options() Options {
	return Options{Stages: five, Origin: Origin{Host: "github.com", Repo: "empowerite/kit"}}
}

// drive sends the messages, running every command each update returns and feeding its messages back, a batch's
// each in turn, as the program would. Quit is noted and dropped. Init is loaded's to send, once.
func drive(m Model, msgs ...tea.Msg) Model {
	m, _ = testkit.Drive(m, nil, msgs...)
	return m
}

// loaded opens the screen at 80×24 and loads every stage, then sends the messages.
func loaded(t *testing.T, o Options, msgs ...tea.Msg) Model {
	t.Helper()
	m := New(runner(t), o)
	first := []tea.Msg{tea.WindowSizeMsg{Width: 80, Height: 24}}
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
	enter = press(tea.KeyEnter)
)

// view gives the screen without its escapes.
func view(m Model) string { return ansi.Strip(m.View().Content) }

// lines gives the screen's lines without their escapes.
func lines(m Model) []string { return strings.Split(view(m), "\n") }

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
	t.Setenv("FAKE_STATUS_CALLS", record)
	o := options()
	o.NoFetch, o.Filters = true, []string{"--mine", "--owner=bob"}
	m := New(runner(t), o)
	msg := m.Init()()
	if got := calls(t, record); len(got) != 1 || got[0] != "status incoming --json --no-fetch" {
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
	if !ok || len(batch) != 4 {
		t.Fatalf("incoming's answer did not start the other four: %T", cmd())
	}
	m = drive(m, batch)
	want := []string{"status incoming --json --no-fetch", "status backlog --json --mine --owner=bob",
		"status worktrees --json", "status changes --json --mine --owner=bob", "status releases --json"}
	if got := calls(t, record); strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Errorf("the calls:\n%s\nwant:\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
	for i := range m.stages {
		if m.stages[i].loading {
			t.Errorf("%s still loads", m.stages[i].name)
		}
	}
	// a second answer from incoming starts nothing again
	if _, cmd := m.Update(loadedMsg{stage: "incoming", data: []byte(`{"fetched": false}`)}); cmd != nil {
		t.Errorf("incoming's second answer started something")
	}
}

func TestWithoutIncomingEveryStageLoadsAtOnce(t *testing.T) {
	record := filepath.Join(t.TempDir(), "calls")
	t.Setenv("FAKE_STATUS_CALLS", record)
	m := New(runner(t), Options{Stages: []string{"changes", "worktrees"}})
	batch, ok := m.Init()().(tea.BatchMsg)
	if !ok || len(batch) != 2 {
		t.Fatalf("Init did not start both: %T", m.Init()())
	}
	m = drive(m, batch)
	if got := calls(t, record); strings.Join(got, "\n") != "status changes --json\nstatus worktrees --json" {
		t.Errorf("the calls: %q", got)
	}
	if first := lines(m)[0]; first != " kit status" {
		t.Errorf("the title without incoming: %q", first)
	}
	if labels := lines(m)[3]; !strings.Contains(labels, "changes 3") || !strings.Contains(labels, "worktrees 5") ||
		strings.Index(labels, "changes") > strings.Index(labels, "worktrees") {
		t.Errorf("the labels follow kit's order: %q", labels)
	}
}

func TestLeftAndRightMoveAlongTheLineEvenInsideTheRows(t *testing.T) {
	m := loaded(t, options())
	if m.stage != 0 || !strings.Contains(m.View().Content, style.Mark("●")) {
		t.Fatalf("the first stage is not selected:\n%s", view(m))
	}
	// left at the first stage goes round to the last, and right there back to the first
	m = drive(m, left)
	if m.stage != 4 || m.moving {
		t.Errorf("left at the first stage: stage %d, moving %v", m.stage, m.moving)
	}
	m = drive(m, right)
	if m.stage != 0 {
		t.Errorf("right at the last stage: stage %d", m.stage)
	}
	m = drive(m, right, right)
	if m.stage != 2 || !strings.Contains(m.View().Content, Branch+" "+style.Mark("worktrees")+" "+style.Off("5")) {
		t.Errorf("right twice: stage %d\n%s", m.stage, view(m))
	}
	m = drive(m, down, right)
	if m.stage != 3 || !m.inRows {
		t.Errorf("right inside the rows: stage %d, in rows %v", m.stage, m.inRows)
	}
	if !strings.Contains(view(m), style.Focus+" "+Change+" #2") {
		t.Errorf("the next stage's first row is not under the cursor:\n%s", view(m))
	}
	// from changes, three to the right goes round the end: releases, backlog, incoming
	m = drive(m, right, right, right)
	if m.stage != 1 {
		t.Errorf("right round the end of the line: %d", m.stage)
	}
}

func TestDownEntersTheRowsAndUpOnTheFirstRowReturnsToTheLine(t *testing.T) {
	m := loaded(t, options(), right, right)
	if m.inRows || strings.Contains(view(m), style.Focus) {
		t.Fatalf("on the line, a row has the focus:\n%s", view(m))
	}
	m = drive(m, down)
	if !m.inRows || !strings.Contains(view(m), " "+style.Focus+" "+Desk+" main") {
		t.Fatalf("down did not enter the rows:\n%s", view(m))
	}
	m = drive(m, down, down)
	if !strings.Contains(view(m), style.Focus+" "+Branch+" 42-settings") {
		t.Errorf("down twice more:\n%s", view(m))
	}
	m = drive(m, down, down, down)
	if m.stages[2].cursor != 4 {
		t.Errorf("down past the last row: %d", m.stages[2].cursor)
	}
	m = drive(m, up, up, up, up)
	if !m.inRows || m.stages[2].cursor != 0 {
		t.Errorf("up to the first row: in rows %v, cursor %d", m.inRows, m.stages[2].cursor)
	}
	m = drive(m, up)
	if m.inRows || strings.Contains(view(m), style.Focus) {
		t.Errorf("up on the first row did not return to the line:\n%s", view(m))
	}
	m = drive(m, up)
	if m.inRows {
		t.Errorf("up on the line entered the rows")
	}
}

func TestEnterOpensARowAndAgainClosesIt(t *testing.T) {
	m := loaded(t, options(), right, right, down, down, enter)
	v := view(m)
	for _, want := range []string{"f6f51bc #1 materialize", "… and 263 more", " M kit/bin/kit-status", "?? docs/status.md"} {
		if !strings.Contains(v, want) {
			t.Errorf("the open worktree lacks %q:\n%s", want, v)
		}
	}
	if !m.stages[2].open["/home/kit/worktrees/1-the-first-release-v0-1-0"] {
		t.Errorf("the row is not open by its key: %v", m.stages[2].open)
	}
	m = drive(m, enter)
	if v := view(m); strings.Contains(v, "f6f51bc") {
		t.Errorf("enter again did not close the row:\n%s", v)
	}
	// enter on the line opens nothing
	m = drive(m, up, up, up, enter)
	if strings.Contains(view(m), "f6f51bc") {
		t.Errorf("enter on the line opened a row")
	}
	// an incoming row is a leaf
	m = drive(m, left, down, enter)
	if n := len(lines(m)); n != 24 || len(m.stages[1].open) != 0 {
		t.Errorf("a leaf opened: %d lines, open %v", n, m.stages[1].open)
	}
}

func TestRefreshRunsTheStageAgain(t *testing.T) {
	record := filepath.Join(t.TempDir(), "calls")
	t.Setenv("FAKE_STATUS_CALLS", record)
	m := loaded(t, options(), right, right, right, right)
	next, cmd := m.Update(press('r'))
	m = next.(Model)
	if cmd == nil || !strings.Contains(view(m), "loading") || !strings.Contains(view(m), "releases …") {
		t.Fatalf("r did not run releases again:\n%s", view(m))
	}
	m = drive(m, cmd())
	if got := calls(t, record); len(got) != 6 || got[5] != "status releases --json" {
		t.Errorf("the calls: %q", got)
	}
	if !strings.Contains(view(m), "releases 3") {
		t.Errorf("releases did not come back:\n%s", view(m))
	}
}

func TestARefusedStageShowsKitsLineAndRefreshTriesAgain(t *testing.T) {
	t.Setenv("FAKE_STATUS_REFUSE", "changes")
	m := loaded(t, options(), right, right, right)
	v := view(m)
	if !strings.Contains(v, "changes !") || !strings.Contains(v, Failed+" kit status changes: the forge cannot be read; run: kit help status") {
		t.Fatalf("the refusal is not shown:\n%s", v)
	}
	if !strings.Contains(m.View().Content, style.Refusal(Failed+" kit status changes: the forge cannot be read; run: kit help status")) {
		t.Errorf("the refusal is not red")
	}
	// with no pull requests known, the worktrees show none
	m = drive(m, left)
	if v := strings.Join(lines(m)[5:], "\n"); strings.Contains(v, "no pull request yet") || strings.Contains(v, Change) {
		t.Errorf("the worktrees claim pull requests after changes failed:\n%s", v)
	}
	t.Setenv("FAKE_STATUS_REFUSE", "")
	m = drive(m, right, press('r'))
	if v := view(m); !strings.Contains(v, "changes 3") || strings.Contains(v, "cannot be read") {
		t.Errorf("r did not try again:\n%s", v)
	}
}

func TestAStageThatDiesSilentlyIsARefusalNotAnEmptyStage(t *testing.T) {
	t.Setenv("FAKE_STATUS_SILENT", "backlog")
	m := loaded(t, options())
	v := view(m)
	if !strings.Contains(v, "backlog !") || strings.Contains(v, "no open issue") ||
		!strings.Contains(v, Failed+" kit status backlog exited 1 with nothing on stderr") {
		t.Errorf("the silent failure:\n%s", v)
	}
	// a kit killed by a signal has no code; one with a line keeps it
	if got := refusalOf("backlog", &kit.ExitError{Code: -1}); got != "kit status backlog was killed" {
		t.Errorf("killed: %q", got)
	}
	if got := refusalOf("backlog", &kit.ExitError{Code: 1, Stderr: "❌ the line\n"}); got != "the line" {
		t.Errorf("with a line: %q", got)
	}
}

func TestADocumentOfAnotherShapeIsOneRefusalLine(t *testing.T) {
	t.Setenv("FAKE_STATUS_ABOVE", "incoming")
	m := loaded(t, options(), right)
	v := view(m)
	// the line wraps at the pane's width, as every refusal does
	want := "kit status incoming --json printed an array where an object was expected; run kit status inside a repository"
	if m.stages[1].refusal != want || !strings.Contains(v, "incoming !") ||
		!strings.Contains(v, Failed+" kit status incoming --json printed an array where an object was expected;") {
		t.Errorf("an array for incoming: %q\n%s", m.stages[1].refusal, v)
	}
	t.Setenv("FAKE_STATUS_ABOVE", "backlog")
	m = loaded(t, options())
	v = view(m)
	want = `kit status backlog --json printed an array of objects without "number" where an array of issues was ` +
		"expected; run kit status inside a repository"
	if m.stages[0].refusal != want || !strings.Contains(v, "backlog !") || strings.Contains(v, "#0") ||
		!strings.Contains(v, Failed+` kit status backlog --json printed an array of objects without "number"`) {
		t.Errorf("desks for backlog: %q\n%s", m.stages[0].refusal, v)
	}
	for name, data := range map[string]string{"changes": "{}", "worktrees": "[]", "releases": `[{"version": "1"}, {}]`} {
		if shape(name, []byte(data)) == nil {
			t.Errorf("%s took %s", name, data)
		}
	}
	for name, data := range map[string]string{"backlog": "[]", "incoming": `{"fetched": false}`, "changes": "null",
		"releases": `[{"version": "1"}]`, "worktrees": " {} "} {
		if err := shape(name, []byte(data)); err != nil {
			t.Errorf("%s refused %s: %v", name, data, err)
		}
	}
}

func TestQuitCodes(t *testing.T) {
	m := loaded(t, options())
	next, cmd := m.Update(press(tea.KeyEscape))
	if m = next.(Model); cmd == nil || cmd() != tea.Quit() || m.Interrupted() || m.ctx.Err() == nil {
		t.Errorf("esc on the line: cmd %v, interrupted %v, context %v", cmd, m.Interrupted(), m.ctx.Err())
	}
	// q is no key of the screen's
	m = loaded(t, options())
	if next, cmd = m.Update(press('q')); cmd != nil || next.(Model).ctx.Err() != nil {
		t.Errorf("q quit the screen")
	}
	// ctrl-c quits from anywhere: the rows, and over the legend
	for _, at := range [][]tea.Msg{nil, {down}, {down, press('?')}} {
		m = loaded(t, options(), at...)
		next, cmd = m.Update(tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl})
		if m = next.(Model); cmd == nil || cmd() != tea.Quit() || !m.Interrupted() || m.ctx.Err() == nil {
			t.Errorf("ctrl-c after %v: cmd %v, interrupted %v, context %v", at, cmd, m.Interrupted(), m.ctx.Err())
		}
	}
}

func TestEscWalksBackFromTheLegendToTheRowsToTheLineAndQuits(t *testing.T) {
	m := loaded(t, options(), right, right, down, down, press('?'))
	if !m.legend || !strings.Contains(view(m), groupEvery) {
		t.Fatalf("? did not show the legend:\n%s", view(m))
	}
	next, cmd := m.Update(press(tea.KeyEscape))
	if m = next.(Model); cmd != nil || m.legend || !m.inRows || m.stages[2].cursor != 1 {
		t.Fatalf("esc over the legend: legend %v, in rows %v, cursor %d", m.legend, m.inRows, m.stages[2].cursor)
	}
	if !strings.Contains(view(m), style.Focus+" "+Branch+" 1-the-first-release") {
		t.Errorf("esc over the legend did not bring the rows back as they were:\n%s", view(m))
	}
	next, cmd = m.Update(press(tea.KeyEscape))
	if m = next.(Model); cmd != nil || m.inRows || m.stage != 2 || strings.Contains(view(m), style.Focus) {
		t.Fatalf("esc in the rows: in rows %v, stage %d\n%s", m.inRows, m.stage, view(m))
	}
	next, cmd = m.Update(press(tea.KeyEscape))
	if m = next.(Model); cmd == nil || cmd() != tea.Quit() || m.Interrupted() || m.ctx.Err() == nil {
		t.Errorf("esc on the line did not quit: cmd %v", cmd)
	}
}

func TestQuestionMarkShowsTheLegendInPlaceOfThePaneAndAgainHidesIt(t *testing.T) {
	m := loaded(t, options(), right, right)
	before := lines(m)
	m = drive(m, press('?'))
	got := lines(m)
	if len(got) != 24 || got[0] != before[0] || got[2] != before[2] || got[3] != before[3] {
		t.Errorf("the title or the line moved under the legend:\n%s", view(m))
	}
	if strings.Contains(view(m), Desk+" main") || !strings.Contains(view(m), groupEvery) ||
		got[23] != " "+legendKeys {
		t.Errorf("the legend is not in place of the pane, with its keys:\n%s", view(m))
	}
	// the arrows scroll the legend, not the line, on a screen too short for all of it
	m = drive(m, tea.WindowSizeMsg{Width: 80, Height: 12}, right, down)
	if m.stage != 2 || m.legendOffset != 1 || m.inRows || strings.Contains(view(m), "   "+groupEvery) {
		t.Errorf("keys over the legend: stage %d, offset %d, in rows %v\n%s", m.stage, m.legendOffset, m.inRows, view(m))
	}
	m = drive(m, tea.WindowSizeMsg{Width: 80, Height: 24}, press('?'))
	if m.legend || !strings.Contains(view(m), Desk+" main") || lines(m)[23] != " "+lineKeys {
		t.Errorf("? again did not hide the legend:\n%s", view(m))
	}
	// shift-/ comes as ? with its text
	m = drive(m, tea.KeyPressMsg{Code: '/', Mod: tea.ModShift, Text: "?"})
	if !m.legend {
		t.Errorf("shift-/ did not show the legend")
	}
}

func TestTheKeysBarIsTheLastLineInEveryState(t *testing.T) {
	for _, c := range []struct {
		msgs []tea.Msg
		keys string
	}{
		{nil, lineKeys},
		{[]tea.Msg{down}, rowKeys},
		{[]tea.Msg{right, right, down, enter}, rowKeys},
		{[]tea.Msg{press('?')}, legendKeys},
		{[]tea.Msg{press('?'), press(tea.KeyPgDown)}, legendKeys},
	} {
		for _, height := range []int{24, 12} {
			m := loaded(t, options(), append([]tea.Msg{tea.WindowSizeMsg{Width: 80, Height: height}}, c.msgs...)...)
			got := lines(m)
			if len(got) != height || got[height-1] != " "+c.keys {
				t.Errorf("after %v at %d: %d lines, the last %q", c.msgs, height, len(got), got[len(got)-1])
			}
		}
	}
	// at a width too narrow for it, the bar is clipped, never wrapped
	m := loaded(t, options(), tea.WindowSizeMsg{Width: 30, Height: 12})
	if got := lines(m); len(got) != 12 || !strings.HasSuffix(got[11], "…") || ansi.StringWidth(got[11]) != 30 {
		t.Errorf("the bar at 30 columns: %q", got[len(got)-1])
	}
}

func TestAStageThatOutlastsTheLimitIsARefusalAndRRunsItAgain(t *testing.T) {
	t.Setenv("FAKE_STATUS_SLOW", "releases")
	m := New(runner(t), options())
	m.limit = 100 * time.Millisecond
	m = drive(m, tea.WindowSizeMsg{Width: 80, Height: 24}, right, right, right, right)
	start := time.Now()
	msg := m.load("releases")()
	// the fake sleeps five seconds: the answer comes at the limit only when the sleep is killed with the kit
	if took := time.Since(start); took > time.Second {
		t.Fatalf("the stage did not end at the limit: %s", took)
	}
	m = drive(m, msg)
	v := view(m)
	if !strings.Contains(v, "releases !") ||
		!strings.Contains(v, Failed+" kit status releases took longer than 2 minutes; r runs it again") {
		t.Fatalf("the stage past its limit:\n%s", v)
	}
	t.Setenv("FAKE_STATUS_SLOW", "")
	m = drive(m, press('r'))
	if v := view(m); !strings.Contains(v, "releases 3") || strings.Contains(v, "took longer") {
		t.Errorf("r did not run it again:\n%s", v)
	}
}

func TestQuitKillsTheStagesStillRunning(t *testing.T) {
	record := filepath.Join(t.TempDir(), "calls")
	t.Setenv("FAKE_STATUS_CALLS", record)
	t.Setenv("FAKE_STATUS_SLOW", "releases")
	m := New(runner(t), options())
	answers := make(chan tea.Msg, 1)
	load := m.load("releases")
	go func() { answers <- load() }()
	// the fake records its call before it sleeps: esc comes once it runs
	for start := time.Now(); calls(t, record) == nil; time.Sleep(10 * time.Millisecond) {
		if time.Since(start) > 2*time.Second {
			t.Fatal("the stage did not start")
		}
	}
	m = drive(m, press(tea.KeyEscape))
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
	if first := lines(m)[0]; first != " kit status · empowerite/kit" {
		t.Errorf("the title: %q", first)
	}
	if !strings.Contains(lines(m)[3], "incoming 6") {
		t.Errorf("incoming's count: %q", lines(m)[3])
	}
	// while incoming loads, its label says so
	m = New(runner(t), options())
	if labels := lines(m)[3]; !strings.Contains(labels, "incoming …") {
		t.Errorf("incoming loading: %q", labels)
	}
	o := options()
	o.NoFetch, o.Origin = true, Origin{}
	m = loaded(t, o, right)
	if first := lines(m)[0]; first != " kit status" {
		t.Errorf("the title with --no-fetch: %q", first)
	}
	v := view(m)
	if !strings.Contains(v, "   not fetched: kit status --no-fetch skips the fetch") ||
		strings.Contains(v, "incoming 0") || !strings.Contains(lines(m)[3], "incoming ") {
		t.Errorf("the incoming pane with --no-fetch:\n%s", v)
	}
	m = loaded(t, Options{Stages: []string{"releases"}})
	if first := lines(m)[0]; first != " kit status" {
		t.Errorf("the title alone: %q", first)
	}
}

func TestTheFetchsUpdatedBranchesAndNewTagsAreRowsAndTheGoneBranchesOneRow(t *testing.T) {
	m := loaded(t, options(), right)
	rows := lines(m)[5:11]
	for i, want := range []string{"a1b2c3d", "e4f5a6b", Branch + "       46-a-noun-lists updated",
		Branch + "       42-settings new branch", Gone + "       old-experiment upstream gone",
		Tag + "       release/0.2.0 new tag"} {
		if !strings.Contains(rows[i], want) {
			t.Errorf("row %d lacks %q: %q", i, want, rows[i])
		}
	}
	m = drive(m, right, down, down, down, down, down, enter)
	v := view(m)
	for _, want := range []string{style.Focus + " " + Gone + " gone: old-experiment, 44-a-spike",
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

func TestWorktreesShowTheirPullRequestsAndTheirUncommittedCount(t *testing.T) {
	m := loaded(t, options(), right, right)
	rows := lines(m)[5:9]
	for i, want := range [][]string{
		{Desk + " main", "↑0 ↓0", "clean"},
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
	if strings.Contains(rows[0], "no pull request") || strings.Contains(rows[3], "no pull request") {
		t.Errorf("the desk or a detached checkout wants a pull request: %q", rows)
	}
	// the numbers line up
	if strings.Index(rows[0], "↓") != strings.Index(rows[1], "↓") {
		t.Errorf("the counts do not line up:\n%s", strings.Join(rows, "\n"))
	}
}

func TestChangesCarryTheirChecksThreadsAndIssues(t *testing.T) {
	// a change's row is wide: at 80 columns its right end is clipped, so the screen is wider here
	m := loaded(t, options(), tea.WindowSizeMsg{Width: 160, Height: 24}, right, right, right, down, enter)
	v := view(m)
	for _, want := range []string{Change + " #2", "👤 jeffrey-aguilera", "🛬 ready", Passed + " 1 / 1 3m",
		"🔨 draft", Running + " 0 / 2", Threads + " 2 unresolved", "🔥 failing:gate", Failed + " lint failing 1 / 2 95s",
		"#1  👤 jeffrey-aguilera  " + InFlight + " in flight  8d  " + Issue + " The first release, v0.1.0"} {
		if !strings.Contains(v, want) {
			t.Errorf("the changes lack %q:\n%s", want, v)
		}
	}
	m = drive(m, down, enter)
	if !strings.Contains(view(m), "closes no issue") {
		t.Errorf("a pull request that closes nothing says nothing:\n%s", view(m))
	}
}

func TestBacklogAndReleasesOpenToTheirDetail(t *testing.T) {
	m := loaded(t, options(), down, enter)
	v := view(m)
	for _, want := range []string{"#45", "#46  👤 jeffrey-aguilera   2d  " + Branch + " 46-a-noun-lists  " + Issue,
		"#7  👤 ada bob           40d", "interactive on a terminal, and\n", "the listing verbs fold into it",
		"https://github.com/empowerite/kit/issues/45", "kit issue 45 starts it"} {
		if !strings.Contains(v, want) {
			t.Errorf("the backlog lacks %q:\n%s", want, v)
		}
	}
	m = loaded(t, options(), right, right, right, right, down, down, enter)
	v = view(m)
	for _, want := range []string{"0.2.0       " + Draft + " draft", "0.1.0       " + Release + " release", Latest + " latest",
		"0.1.0-rc.7  " + Pre + " pre-release  28h", "kit 0.1.0", "release/0.1.0", "2026-10-31T20:00:00Z",
		"https://github.com/empowerite/kit/releases/tag/release/0.1.0"} {
		if !strings.Contains(v, want) {
			t.Errorf("the releases lack %q:\n%s", want, v)
		}
	}
}

func TestThePaneScrollsToKeepTheCursorWhileTheRestStays(t *testing.T) {
	m := loaded(t, options(), tea.WindowSizeMsg{Width: 80, Height: 10}, right, right, down, down, down, down)
	got := lines(m)
	if len(got) != 10 {
		t.Fatalf("%d lines on a screen of 10:\n%s", len(got), view(m))
	}
	if !strings.HasPrefix(got[0], " kit status") || !strings.Contains(got[3], "worktrees 5") || got[9] != " "+rowKeys {
		t.Errorf("the title, the line, or the keys moved:\n%s", view(m))
	}
	// three lines of pane: … 3 above, the cursor's row, and the last row, which needs no … line
	if !strings.Contains(got[5], "… 3 above") || !strings.Contains(got[6], style.Focus+" "+Branch+" worktrees/spike") ||
		!strings.Contains(got[7], Gone+" gone:") || strings.Contains(view(m), Desk+" main") {
		t.Errorf("the pane did not scroll to the cursor:\n%s", view(m))
	}
	// a pane of three lines holds the … lines and the row; one more line shows what the row opens to
	m = drive(m, enter, tea.WindowSizeMsg{Width: 80, Height: 11})
	if !strings.Contains(view(m), "?? notes.md") || !strings.Contains(view(m), style.Focus) {
		t.Errorf("the open row's detail is not shown with its row:\n%s", view(m))
	}
	m = drive(m, tea.WindowSizeMsg{Width: 80, Height: 10})
	// a page is the pane less one row: two rows here
	m = drive(m, press(tea.KeyPgUp))
	if !m.inRows || m.stages[2].cursor != 1 {
		t.Errorf("page up: in rows %v, cursor %d", m.inRows, m.stages[2].cursor)
	}
	m = drive(m, press(tea.KeyPgUp))
	if m.inRows || !strings.Contains(view(m), Desk+" main") {
		t.Errorf("page up past the first row did not return to the line and the top:\n%s", view(m))
	}
	m = drive(m, press(tea.KeyPgDown))
	if !m.inRows || m.stages[2].cursor != 1 {
		t.Errorf("page down from the line: in rows %v, cursor %d", m.inRows, m.stages[2].cursor)
	}
	m = drive(m, press(tea.KeyPgDown), press(tea.KeyPgDown))
	if m.stages[2].cursor != 4 {
		t.Errorf("page down past the last row: cursor %d", m.stages[2].cursor)
	}
}

func TestEveryLineFitsTheWidthInCells(t *testing.T) {
	for _, width := range []int{80, 50} {
		for i := range five {
			m := loaded(t, options(), tea.WindowSizeMsg{Width: width, Height: 24})
			for range i {
				m = drive(m, right)
			}
			m = drive(m, down, enter)
			clipped := false
			for _, l := range strings.Split(m.View().Content, "\n") {
				if n := ansi.StringWidth(l); n > width {
					t.Errorf("%s at %d: a line of %d cells: %q", five[i], width, n, ansi.Strip(l))
				}
				clipped = clipped || strings.HasSuffix(ansi.Strip(l), "…")
			}
			if (five[i] == "changes" || five[i] == "backlog") && !clipped {
				t.Errorf("%s at %d: no long title was clipped:\n%s", five[i], width, view(m))
			}
		}
	}
}

func TestAnUnknownStageIsRefusedWithoutRunning(t *testing.T) {
	record := filepath.Join(t.TempDir(), "calls")
	t.Setenv("FAKE_STATUS_CALLS", record)
	m := loaded(t, Options{Stages: []string{"worktrees", "nonesuch"}}, right)
	if v := view(m); !strings.Contains(v, "nonesuch !") || !strings.Contains(v, "kittui does not draw the stage 'nonesuch'") {
		t.Errorf("the unknown stage:\n%s", v)
	}
	if got := calls(t, record); strings.Join(got, "\n") != "status worktrees --json" {
		t.Errorf("the calls: %q", got)
	}
	m = loaded(t, Options{})
	if v := view(m); !strings.Contains(v, "no stage is configured") || len(lines(m)) != 24 {
		t.Errorf("no stage at all:\n%s", v)
	}
}

func TestOpenReadsTheStagesAndTheWords(t *testing.T) {
	record := filepath.Join(t.TempDir(), "calls")
	t.Setenv("FAKE_STATUS_CALLS", record)
	m, err := Open(runner(t), []string{"--no-fetch", "--mine"})
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, s := range m.stages {
		names = append(names, s.name)
	}
	if strings.Join(names, " ") != strings.Join(five, " ") || !m.noFetch || strings.Join(m.filters, " ") != "--mine" {
		t.Errorf("Open: %q, no fetch %v, filters %q", names, m.noFetch, m.filters)
	}
	if got := calls(t, record); strings.Join(got, "\n") != "status --stages" {
		t.Errorf("Open ran %q", got)
	}
	t.Setenv("FAKE_STATUS_REFUSE", "stages")
	_, err = Open(runner(t), nil)
	refusal, ok := err.(*kit.ExitError)
	if !ok || refusal.Code != 2 || refusal.Error() != "kit status refuses for the test; run: kit help status" {
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
	m := loaded(t, options(), tea.WindowSizeMsg{Width: 80, Height: 11}, right, right, down, down, press(tea.KeyEnter))
	got := lines(m)
	if len(got) != 11 || got[10] != " "+rowKeys {
		t.Fatalf("the screen is not 11 lines with the keys last:\n%s", view(m))
	}
	v := view(m)
	if !strings.Contains(v, "… 1 above") || !strings.Contains(v, "… 9 more") ||
		!strings.Contains(v, style.Focus+" "+Branch+" 1-the-first-release") {
		t.Errorf("the open row is not shown between its … lines:\n%s", v)
	}
	// a pane that holds every line says nothing of the kind
	m = loaded(t, options(), right, right)
	if strings.Contains(view(m), " above") || strings.Contains(view(m), " more") {
		t.Errorf("a pane that fits says lines are hidden:\n%s", view(m))
	}
}

func TestTheWindowCountsWhatItHides(t *testing.T) {
	for _, c := range []struct {
		n, h, o, start, end int
		above, below        bool
	}{
		{5, 10, 3, 0, 5, false, false},
		{20, 10, 0, 0, 9, false, true},
		{20, 10, 5, 5, 13, true, true},
		{20, 10, 50, 11, 20, true, false},
	} {
		start, end, above, below := window(c.n, c.h, c.o)
		if start != c.start || end != c.end || above != c.above || below != c.below {
			t.Errorf("window(%d, %d, %d) = %d %d %v %v", c.n, c.h, c.o, start, end, above, below)
		}
	}
}

func TestTheDotTravelsTheWayTheArrowPointsAndStopsOnItsStage(t *testing.T) {
	m := loaded(t, options())
	// the line holds one-cell runes only, after one space, so a rune's index is its cell
	at := func(m Model) int {
		for k, r := range []rune(ansi.Strip(lines(m)[2])) {
			if r == '●' {
				return k
			}
		}
		return -1
	}
	// one key, no frames yet: the stage is chosen, and the dot has left the first marker but not reached the next
	next, cmd := m.Update(press(tea.KeyRight))
	m = next.(Model)
	if m.stage != 1 || !m.moving || cmd == nil {
		t.Fatalf("right: stage %d, moving %v", m.stage, m.moving)
	}
	start := at(m)
	trail := []int{start}
	for m.moving {
		next, _ = m.Update(frameMsg{})
		m = next.(Model)
		trail = append(trail, at(m))
	}
	for k := 1; k < len(trail); k++ {
		if trail[k] <= trail[k-1] {
			t.Fatalf("the dot did not go right every frame: %v", trail)
		}
	}
	if len(trail) > frames+1 || trail[len(trail)-1] != m.center(1)+1 {
		t.Errorf("the dot did not stop on incoming in %d frames: %v", frames, trail)
	}
	// left from the first stage goes left, off the start of the line and in from its end
	m = loaded(t, options())
	next, _ = m.Update(press(tea.KeyLeft))
	m = next.(Model)
	next, _ = m.Update(frameMsg{})
	m = next.(Model)
	if m.stage != 4 || at(m) >= m.center(0)+1 && at(m) < m.center(4) {
		t.Errorf("left at the first stage did not go round the start: dot at %d", at(m))
	}
}
