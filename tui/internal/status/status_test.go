// The screen is driven key by key through Update, against testdata/fake-kit, which answers each stage from the
// fixtures beside it, as Bubble Tea would drive it.
package status

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/empowerite/kit/tui/internal/kit"
	"github.com/empowerite/kit/tui/internal/style"
	"github.com/empowerite/kit/tui/internal/testkit"
)

// eight is the picture's default order.
var eight = []string{"backlog", "underway", "incoming", "desk", "worktrees", "changes", "builds", "releases"}

// runner gives a kit.Runner over testdata/fake-kit.
func runner(t *testing.T) kit.Runner {
	t.Helper()
	return testkit.Runner(t, "status")
}

// options gives the eight stages and origin as a test knows them.
func options() Options {
	return Options{Stages: eight, Origin: Origin{Host: "github.com", Repo: "empowerite/kit"}}
}

// drive sends the messages, running every command each update returns and feeding its messages back, a batch's
// each in turn, as the program would. Quit is noted and dropped. Init is loaded's to send, once.
func drive(m Model, msgs ...tea.Msg) Model {
	m, _ = testkit.Drive(m, nil, msgs...)
	return m
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
	if !ok || len(batch) != 7 {
		t.Fatalf("incoming's answer did not start the other seven: %T", cmd())
	}
	m = drive(m, batch)
	want := []string{"status incoming --json --no-fetch", "status backlog --json --mine --owner=bob",
		"status underway --json --mine --owner=bob", "status desk --json", "status worktrees --json",
		"status changes --json --mine --owner=bob",
		"status builds --json", "status releases --json"}
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
	if labels := lines(m)[2]; !strings.Contains(labels, "changes 3") || !strings.Contains(labels, "worktrees 4") ||
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
	if m.stage != 7 || m.moving {
		t.Errorf("left at the first stage: stage %d, moving %v", m.stage, m.moving)
	}
	m = drive(m, right)
	if m.stage != 0 {
		t.Errorf("right at the last stage: stage %d", m.stage)
	}
	m = drive(m, right, right, right)
	if m.stage != 3 || !strings.Contains(m.View().Content, Desk+" "+style.Mark("1")) {
		t.Errorf("right three times: stage %d\n%s", m.stage, view(m))
	}
	m = drive(m, right, down, right)
	if m.stage != 5 || !m.inRows {
		t.Errorf("right inside the rows: stage %d, in rows %v", m.stage, m.inRows)
	}
	if !strings.Contains(view(m), style.Focus+" "+Change+" #2") {
		t.Errorf("the next stage's first row is not under the cursor:\n%s", view(m))
	}
	// from changes, four to the right goes round the end: builds, releases, backlog, underway
	m = drive(m, right, right, right, right)
	if m.stage != 1 {
		t.Errorf("right round the end of the line: %d", m.stage)
	}
}

func TestDownEntersTheRowsAndUpOnTheFirstRowReturnsToTheLine(t *testing.T) {
	m := loaded(t, options(), right, right, right, right)
	if m.inRows || strings.Contains(view(m), style.Focus) {
		t.Fatalf("on the line, a row has the focus:\n%s", view(m))
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
	if m.stages[4].cursor != 3 {
		t.Errorf("down past the last row: %d", m.stages[4].cursor)
	}
	m = drive(m, up, up, up)
	if !m.inRows || m.stages[4].cursor != 0 {
		t.Errorf("up to the first row: in rows %v, cursor %d", m.inRows, m.stages[4].cursor)
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
	m := loaded(t, options(), right, right, right, right, down, enter)
	v := view(m)
	for _, want := range []string{"f6f51bc #1 materialize", "… and 263 more", " M kit/bin/kit-status",
		"?? docs/status.md"} {
		if !strings.Contains(v, want) {
			t.Errorf("the open worktree lacks %q:\n%s", want, v)
		}
	}
	if !m.stages[4].open["/home/kit/worktrees/1-the-first-release-v0-1-0"] {
		t.Errorf("the row is not open by its key: %v", m.stages[4].open)
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
	m = drive(m, left, left, down, enter)
	if n := len(lines(m)); n != 24 || len(m.stages[2].open) != 0 {
		t.Errorf("a leaf opened: %d lines, open %v", n, m.stages[2].open)
	}
}

func TestRefreshRunsTheStageAgain(t *testing.T) {
	record := filepath.Join(t.TempDir(), "calls")
	t.Setenv("FAKE_STATUS_CALLS", record)
	m := loaded(t, options(), right, right, right, right, right, right, right)
	next, cmd := m.Update(press('r'))
	m = next.(Model)
	if cmd == nil || !strings.Contains(view(m), "loading") || !strings.Contains(view(m), Release+" …") {
		t.Fatalf("r did not run releases again:\n%s", view(m))
	}
	m = drive(m, cmd())
	if got := calls(t, record); len(got) != 9 || got[8] != "status releases --json" {
		t.Errorf("the calls: %q", got)
	}
	if !strings.Contains(view(m), Release+" 3") {
		t.Errorf("releases did not come back:\n%s", view(m))
	}
}

func TestARefusedStageShowsKitsLineAndRefreshTriesAgain(t *testing.T) {
	t.Setenv("FAKE_STATUS_REFUSE", "changes")
	m := loaded(t, options(), right, right, right, right, right)
	v := view(m)
	refusal := Failed + " kit status changes: the forge cannot be read; run: kit help status"
	if !strings.Contains(v, Change+" !") || !strings.Contains(v, refusal) {
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
	t.Setenv("FAKE_STATUS_REFUSE", "")
	m = drive(m, right, press('r'))
	if v := view(m); !strings.Contains(v, Change+" 3") || strings.Contains(v, "cannot be read") {
		t.Errorf("r did not try again:\n%s", v)
	}
}

func TestAStageThatDiesSilentlyIsARefusalNotAnEmptyStage(t *testing.T) {
	t.Setenv("FAKE_STATUS_SILENT", "backlog")
	m := loaded(t, options())
	v := view(m)
	if !strings.Contains(v, Issue+" !") || strings.Contains(v, "no open issue") ||
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
	m := loaded(t, options(), right, right)
	v := view(m)
	// the line wraps at the pane's width, as every refusal does
	want := "kit status incoming --json printed an array where an object of remotes was expected; run kit status " +
		"inside a repository"
	if m.stages[2].refusal != want || !strings.Contains(v, Fetched+" !") ||
		!strings.Contains(v, Failed+" kit status incoming --json printed an array where an object of remotes was") {
		t.Errorf("an array for incoming: %q\n%s", m.stages[2].refusal, v)
	}
	t.Setenv("FAKE_STATUS_ABOVE", "backlog")
	m = loaded(t, options())
	v = view(m)
	want = `kit status backlog --json printed an array of objects without "number" where an array of issues was ` +
		"expected; run kit status inside a repository"
	if m.stages[0].refusal != want || !strings.Contains(v, Issue+" !") || strings.Contains(v, "#0") ||
		!strings.Contains(v, Failed+` kit status backlog --json printed an array of objects without "number"`) {
		t.Errorf("desks for backlog: %q\n%s", m.stages[0].refusal, v)
	}
	t.Setenv("FAKE_STATUS_ABOVE", "desk")
	m = loaded(t, options(), right, right, right)
	want = "kit status desk --json printed an array where an object was expected; run kit status inside a repository"
	if m.stages[3].refusal != want || !strings.Contains(view(m), Desk+" !") || strings.Contains(view(m), Desk+" main") {
		t.Errorf("desks above desks: %q\n%s", m.stages[3].refusal, view(m))
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
	m := loaded(t, options(), right, right, right, right, down, down, press('?'))
	if !m.legend || !strings.Contains(view(m), "└ legend") {
		t.Fatalf("? did not hang the legend:\n%s", view(m))
	}
	next, cmd := m.Update(press(tea.KeyEscape))
	if m = next.(Model); cmd != nil || m.legend || !m.inRows || m.stages[4].cursor != 1 {
		t.Fatalf("esc with the legend on: legend %v, in rows %v, cursor %d", m.legend, m.inRows, m.stages[4].cursor)
	}
	if v := view(m); !strings.Contains(v, style.Focus+" "+Branch+" 42-settings") || strings.Contains(v, "└ legend") {
		t.Errorf("esc with the legend on did not take it down and keep the rows as they were:\n%s", v)
	}
	next, cmd = m.Update(press(tea.KeyEscape))
	if m = next.(Model); cmd != nil || m.inRows || m.stage != 4 || strings.Contains(view(m), style.Focus) {
		t.Fatalf("esc in the rows: in rows %v, stage %d\n%s", m.inRows, m.stage, view(m))
	}
	next, cmd = m.Update(press(tea.KeyEscape))
	if m = next.(Model); cmd == nil || cmd() != tea.Quit() || m.Interrupted() || m.ctx.Err() == nil {
		t.Errorf("esc on the line did not quit: cmd %v", cmd)
	}
	// on the line, esc takes the legend down before it quits
	m = loaded(t, options(), press('?'), press(tea.KeyEscape))
	if m.legend || m.ctx.Err() != nil {
		t.Errorf("esc on the line with the legend on: legend %v, context %v", m.legend, m.ctx.Err())
	}
}

func TestTheKeysBarIsTheLastLineInEveryState(t *testing.T) {
	for _, c := range []struct {
		msgs []tea.Msg
		keys string
	}{
		{nil, lineKeys},
		{[]tea.Msg{down}, backKeys},
		{[]tea.Msg{right, right, right, right, down, enter}, backKeys},
		{[]tea.Msg{press('?')}, backKeys},
		{[]tea.Msg{press('?'), press(tea.KeyPgDown)}, backKeys},
	} {
		for _, width := range []int{80, 120, 200} {
			for _, height := range []int{24, 12} {
				m := loaded(t, options(), append([]tea.Msg{tea.WindowSizeMsg{Width: width, Height: height}}, c.msgs...)...)
				got := lines(m)
				if len(got) != height || got[height-1] != " "+c.keys {
					t.Errorf("after %v at %d×%d: %d lines, the last %q", c.msgs, width, height, len(got), got[len(got)-1])
				}
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
	m = drive(m, tea.WindowSizeMsg{Width: 120, Height: 24}, right, right, right, right, right, right, right)
	start := time.Now()
	msg := m.load("releases")()
	// the fake sleeps five seconds: the answer comes at the limit only when the sleep is killed with the kit
	if took := time.Since(start); took > time.Second {
		t.Fatalf("the stage did not end at the limit: %s", took)
	}
	m = drive(m, msg)
	v := view(m)
	if !strings.Contains(v, Release+" !") ||
		!strings.Contains(v, Failed+" kit status releases took longer than 2 minutes; r runs it again") {
		t.Fatalf("the stage past its limit:\n%s", v)
	}
	t.Setenv("FAKE_STATUS_SLOW", "")
	m = drive(m, press('r'))
	if v := view(m); !strings.Contains(v, Release+" 3") || strings.Contains(v, "took longer") {
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
	if first := lines(m)[0]; first != " kit status · "+Repository+" empowerite/kit" {
		t.Errorf("the title: %q", first)
	}
	if !strings.Contains(m.View().Content, Repository+" "+style.Repo("empowerite/kit")) {
		t.Errorf("the repository is not in its own color: %q", lines(m)[0])
	}
	// the line of stages comes right under the title
	// at 120 the eight labels take the count form
	if !strings.Contains(lines(m)[1], "●") || !strings.Contains(lines(m)[2], Fetched+" 6") {
		t.Errorf("the line and the trees: %q %q", lines(m)[1], lines(m)[2])
	}
	// while incoming loads, its label says so
	m = New(runner(t), options())
	m = drive(m, tea.WindowSizeMsg{Width: 120, Height: 24})
	if labels := lines(m)[2]; !strings.Contains(labels, Fetched+" …") {
		t.Errorf("incoming loading: %q", labels)
	}
	o := options()
	o.NoFetch, o.Origin = true, Origin{}
	m = loaded(t, o, right, right)
	if first := lines(m)[0]; first != " kit status" {
		t.Errorf("the title with --no-fetch: %q", first)
	}
	// without the fetch, each remote's heading says not fetched, and the label has no count
	pane := lines(m)[4:10]
	for i, want := range []string{Remote + " origin · github · github.com", "not fetched",
		Remote + " upstream · gitlab · git.example.org", "not fetched", Remote + " backup · no forge · nas.local",
		"not fetched"} {
		if strings.TrimSpace(pane[i]) != want {
			t.Errorf("the incoming pane with --no-fetch, line %d is not %q:\n%s", i, want, view(m))
		}
	}
	if strings.Contains(view(m), Fetched+" 0") || !strings.Contains(lines(m)[2], "└ "+Fetched+"   ") {
		t.Errorf("the incoming label with --no-fetch: %q", lines(m)[2])
	}
	if !strings.Contains(m.View().Content, style.Off("not fetched")) {
		t.Errorf("not fetched is not dim")
	}
	// with no remote at all, the pane says why it is empty
	m.find("incoming").in.Remotes = nil
	if !strings.Contains(view(m), "   not fetched: kit status --no-fetch skips the fetch") {
		t.Errorf("no remote with --no-fetch:\n%s", view(m))
	}
	m = loaded(t, Options{Stages: []string{"releases"}})
	if first := lines(m)[0]; first != " kit status" {
		t.Errorf("the title alone: %q", first)
	}
}

func TestIncomingGroupsItsRowsByRemoteThePrimaryFirst(t *testing.T) {
	m := loaded(t, options(), right, right)
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
	if !strings.Contains(lines(m)[2], Fetched+" "+"6") || m.count(2) != "6" {
		t.Errorf("incoming does not count its rows alone: %q", lines(m)[2])
	}
	// every row is a leaf, the headings too
	m = drive(m, down, enter, down, enter)
	if len(m.stages[2].open) != 0 {
		t.Errorf("an incoming row opened: %v", m.stages[2].open)
	}
	// a remote whose fetch failed shows git's line, in red, and is no row of the count
	m.find("incoming").in.Remotes[0].Error = "fatal: could not read from remote repository"
	m.find("incoming").in.Remotes[0].Branches.Gone, m.find("incoming").in.Remotes[0].Tags = nil, nil
	failed := Failed + " fatal: could not read from remote repository"
	if v := view(m); !strings.Contains(v, Remote+" upstream · gitlab · git.example.org\n   "+failed+"\n") ||
		!strings.Contains(m.View().Content, style.Refusal(failed)) || m.count(2) != "4" {
		t.Errorf("a failed fetch: count %s\n%s", m.count(2), v)
	}
	m.find("incoming").in.Remotes[0].Error = ""
	// with every remote unchanged, each says so and the count is zero
	for i := range m.find("incoming").in.Remotes {
		r := &m.find("incoming").in.Remotes[i]
		r.Commits, r.Branches.New, r.Branches.Updated, r.Branches.Gone, r.Tags = nil, nil, nil, nil, nil
	}
	if v := view(m); strings.Count(v, "unchanged") != 3 || m.count(2) != "0" {
		t.Errorf("three remotes unchanged: count %s\n%s", m.count(2), v)
	}
	// with no remote at all, the pane says so
	m.find("incoming").in.Remotes = nil
	if !strings.Contains(view(m), "   no remote to fetch") {
		t.Errorf("no remote:\n%s", view(m))
	}
}

func TestTheGoneBranchesAreOneRowThatOpensToEach(t *testing.T) {
	m := loaded(t, options(), right, right, right, right, down, down, down, down, enter)
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

func TestChangesCarryTheirChecksThreadsAndIssues(t *testing.T) {
	// a change's row is wide: at 80 columns its right end is clipped, so the screen is wider here
	m := loaded(t, options(), tea.WindowSizeMsg{Width: 160, Height: 24}, right, right, right, right, right, down, enter)
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
	m := loaded(t, options(), tea.WindowSizeMsg{Width: 80, Height: 24}, down, enter)
	v := view(m)
	for _, want := range []string{"#45", "#46  👤 jeffrey-aguilera   2d  " + Branch + " 46-a-noun-lists  " + Issue,
		"#7  👤 ada bob           40d", "interactive on a terminal, and\n", "the listing verbs fold into it",
		"https://github.com/empowerite/kit/issues/45", "kit issue 45 starts it"} {
		if !strings.Contains(v, want) {
			t.Errorf("the backlog lacks %q:\n%s", want, v)
		}
	}
	m = loaded(t, options(), right, right, right, right, right, right, right, down, down, enter)
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
	m := loaded(t, options(), tea.WindowSizeMsg{Width: 80, Height: 9}, right, right, right, right, down, down, down)
	got := lines(m)
	if len(got) != 9 {
		t.Fatalf("%d lines on a screen of 9:\n%s", len(got), view(m))
	}
	if !strings.HasPrefix(got[0], " kit status") || !strings.Contains(got[2], Branch+" 4") || got[8] != " "+backKeys {
		t.Errorf("the title, the line, or the keys moved:\n%s", view(m))
	}
	// three lines of pane: … 2 above, the cursor's row, and the last row, which needs no … line
	if !strings.Contains(got[4], "… 2 above") || !strings.Contains(got[5], style.Focus+" "+Branch+" worktrees/spike") ||
		!strings.Contains(got[6], Gone+" gone:") || strings.Contains(view(m), "1-the-first-release") {
		t.Errorf("the pane did not scroll to the cursor:\n%s", view(m))
	}
	// a pane of three lines holds the … lines and the row; one more line shows what the row opens to
	m = drive(m, enter, tea.WindowSizeMsg{Width: 80, Height: 10})
	if !strings.Contains(view(m), "?? notes.md") || !strings.Contains(view(m), style.Focus) {
		t.Errorf("the open row's detail is not shown with its row:\n%s", view(m))
	}
	m = drive(m, tea.WindowSizeMsg{Width: 80, Height: 9})
	// a page is the pane less one row: two rows here
	m = drive(m, press(tea.KeyPgUp))
	if !m.inRows || m.stages[4].cursor != 0 {
		t.Errorf("page up: in rows %v, cursor %d", m.inRows, m.stages[4].cursor)
	}
	m = drive(m, press(tea.KeyPgUp))
	if m.inRows || !strings.Contains(view(m), Branch+" 1-the-first-release") {
		t.Errorf("page up past the first row did not return to the line and the top:\n%s", view(m))
	}
	m = drive(m, press(tea.KeyPgDown))
	if !m.inRows || m.stages[4].cursor != 1 {
		t.Errorf("page down from the line: in rows %v, cursor %d", m.inRows, m.stages[4].cursor)
	}
	m = drive(m, press(tea.KeyPgDown), press(tea.KeyPgDown))
	if m.stages[4].cursor != 3 {
		t.Errorf("page down past the last row: cursor %d", m.stages[4].cursor)
	}
}

func TestEveryLineFitsTheWidthInCells(t *testing.T) {
	for _, width := range []int{40, 50, 80, 120, 200} {
		for _, legend := range []bool{false, true} {
			// one load for the eight: each stage in turn, its first row open, then back to the line
			m := loaded(t, options(), tea.WindowSizeMsg{Width: width, Height: 40})
			if legend {
				m = drive(m, press('?'))
			}
			for i := range eight {
				if i > 0 {
					m = drive(m, up, right)
				}
				m = drive(m, down, enter)
				clipped := false
				for _, l := range strings.Split(m.View().Content, "\n") {
					if n := ansi.StringWidth(l); n > width {
						t.Errorf("%s at %d, legend %v: a line of %d cells: %q", eight[i], width, legend, n,
							ansi.Strip(l))
					}
					clipped = clipped || strings.HasSuffix(ansi.Strip(l), "…")
				}
				long := eight[i] == "changes" || eight[i] == "backlog" || eight[i] == "underway"
				if width <= 80 && long && !clipped {
					t.Errorf("%s at %d: no long title was clipped:\n%s", eight[i], width, view(m))
				}
			}
		}
	}
}

func TestAnUnknownStageIsRefusedWithoutRunning(t *testing.T) {
	record := filepath.Join(t.TempDir(), "calls")
	t.Setenv("FAKE_STATUS_CALLS", record)
	m := loaded(t, Options{Stages: []string{"worktrees", "nonesuch"}}, right)
	if v := view(m); !strings.Contains(v, "nonesuch !") ||
		!strings.Contains(v, "kittui does not draw the stage 'nonesuch'") {
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
	if strings.Join(names, " ") != strings.Join(eight, " ") || !m.noFetch || strings.Join(m.filters, " ") != "--mine" {
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
	m := loaded(t, options(), tea.WindowSizeMsg{Width: 80, Height: 10}, right, right, right, right, down, down, enter)
	got := lines(m)
	if len(got) != 10 || got[9] != " "+backKeys {
		t.Fatalf("the screen is not 10 lines with the keys last:\n%s", view(m))
	}
	v := view(m)
	if !strings.Contains(v, "… 1 above") || !strings.Contains(v, "… 4 more") ||
		!strings.Contains(v, style.Focus+" "+Branch+" 42-settings") {
		t.Errorf("the open row is not shown between its … lines:\n%s", v)
	}
	// a pane that holds every line says nothing of the kind
	m = loaded(t, options(), right, right, right, right)
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
	// one key, no frames yet: the stage is chosen, and the dot has left the first marker but not reached the next
	next, cmd := m.Update(press(tea.KeyRight))
	if m := next.(Model); m.stage != 1 || !m.moving || cmd == nil {
		t.Fatalf("right: stage %d, moving %v", m.stage, m.moving)
	}
	m, canvas, screen := frameAll(t, m, right)
	for k := 1; k < len(canvas); k++ {
		if canvas[k] <= canvas[k-1] || screen[k] != canvas[k]-m.scroll+1 {
			t.Fatalf("the dot did not go right every frame: %v on the screen at %v", canvas, screen)
		}
	}
	if canvas[len(canvas)-1] != m.center(1) {
		t.Errorf("the dot did not stop on underway: %v", canvas)
	}
	// left from the first stage goes left, off the start of the line and in from its end
	m, canvas, _ = frameAll(t, loaded(t, options()), left)
	wrapped := 0
	for k := 1; k < len(canvas); k++ {
		if canvas[k] > canvas[k-1] {
			wrapped++
		}
	}
	if m.stage != 7 || wrapped != 1 || canvas[len(canvas)-1] != m.center(7) {
		t.Errorf("left at the first stage did not go left round the start to releases: %v", canvas)
	}
}

func TestQuestionMarkHangsTheLegendInEveryTreeAndThePaneStays(t *testing.T) {
	// the eight trees with their legends fit 240 columns
	m := loaded(t, options(), tea.WindowSizeMsg{Width: 240, Height: 24}, right, right, right, right)
	before := lines(m)
	m = drive(m, press('?'))
	got := lines(m)
	if len(got) != 24 || got[0] != before[0] || got[23] != " "+backKeys {
		t.Errorf("the title or the keys moved under the legend:\n%s", view(m))
	}
	if n := strings.Count(view(m), "└ legend"); n != 8 {
		t.Errorf("%d trees hang a legend, not eight:\n%s", n, view(m))
	}
	if !strings.Contains(view(m), Branch+" 1-the-first-release") {
		t.Errorf("the pane did not stay under the trees:\n%s", view(m))
	}
	// the keys move in the rows while the legend is on
	m = drive(m, down, down)
	if !m.inRows || m.stages[4].cursor != 1 || !strings.Contains(view(m), style.Focus+" "+Branch+" 42-settings") {
		t.Errorf("down with the legend on: in rows %v, cursor %d\n%s", m.inRows, m.stages[4].cursor, view(m))
	}
	m = drive(m, up, up, press('?'))
	if m.legend || strings.Contains(view(m), "└ legend") || lines(m)[23] != " "+lineKeys ||
		strings.Join(lines(m), "\n") != strings.Join(before, "\n") {
		t.Errorf("? again did not take the legend down:\n%s", view(m))
	}
	// shift-/ comes as ? with its text
	m = drive(m, tea.KeyPressMsg{Code: '/', Mod: tea.ModShift, Text: "?"})
	if !m.legend {
		t.Errorf("shift-/ did not hang the legend")
	}
}

func TestWorktreesShowTheirPullRequestsAndTheirUncommittedCountButNotTheDesk(t *testing.T) {
	m := loaded(t, options(), right, right, right, right)
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
	if !strings.Contains(lines(m)[2], Branch+" 4") {
		t.Errorf("the worktrees count the desk: %q", lines(m)[2])
	}
	// the numbers line up
	if strings.Index(rows[0], "↓") != strings.Index(rows[1], "↓") {
		t.Errorf("the counts do not line up:\n%s", strings.Join(rows, "\n"))
	}
}

func TestTheDeskIsOneRowThatOpensToItsPathsThenItsCommits(t *testing.T) {
	m := loaded(t, options(), right, right, right)
	pane := lines(m)[4:]
	for _, text := range []string{Desk + " main", "↑12 ↓1", "2 uncommitted"} {
		if !strings.Contains(pane[0], text) {
			t.Errorf("the desk's row lacks %q: %q", text, pane[0])
		}
	}
	if strings.Contains(pane[0], "no pull request") || strings.TrimSpace(pane[1]) != "" ||
		!strings.Contains(lines(m)[2], Desk+" 1") {
		t.Errorf("the desk is not one row without a pull request:\n%s", view(m))
	}
	if !strings.Contains(m.View().Content, style.Warn("2 uncommitted")) {
		t.Errorf("the uncommitted count is not yellow")
	}
	m = drive(m, down, enter)
	v := view(m)
	order := []string{style.Focus + " " + Desk + " main", " M kit.yml", "?? scratch.txt", "c0ffee1 #45 status: the desk",
		"c0ffeea #45 status: the goldens", "… and 2 more"}
	at := -1
	for _, want := range order {
		k := strings.Index(v, want)
		if k <= at {
			t.Errorf("the open desk lacks %q after the line before it:\n%s", want, v)
		}
		at = k
	}
	if strings.Contains(v, "c0ffeeb") {
		t.Errorf("the open desk lists more than ten commits:\n%s", v)
	}
	if !strings.Contains(m.View().Content, style.Warn(" M")+" kit.yml") {
		t.Errorf("an uncommitted path's status is not yellow")
	}
	// a desk on a branch with a pull request shows it, as a worktree does
	m.find("desk").desk.Branch = "1-the-first-release-v0-1-0"
	if !strings.Contains(view(m), Change+" #2 🛬 ready") {
		t.Errorf("the desk does not show its pull request:\n%s", view(m))
	}
	// a clean desk says so, dim
	m.find("desk").desk.Uncommitted = 0
	if !strings.Contains(m.View().Content, style.Off("clean")) {
		t.Errorf("a clean desk does not say clean")
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

// dotAt gives the cell of the dot on the screen's line, or -1; the line holds one-cell runes only.
func dotAt(m Model) int {
	for k, r := range []rune(ansi.Strip(lines(m)[1])) {
		if r == '●' {
			return k
		}
	}
	return -1
}

// frameAll steps the dot to its stage, frame by frame, and gives the dot's cell on the canvas and on the screen after
// the key and after every frame.
func frameAll(t *testing.T, m Model, key tea.Msg) (Model, []int, []int) {
	t.Helper()
	next, _ := m.Update(key)
	m = next.(Model)
	canvas, screen := []int{m.dot.at}, []int{dotAt(m)}
	for n := 0; m.moving; n++ {
		if n > frames {
			t.Fatalf("the dot took more than %d frames", frames)
		}
		next, _ = m.Update(frameMsg{})
		m = next.(Model)
		at := m.dot.at
		if !m.moving {
			at = m.center(m.stage)
		}
		canvas, screen = append(canvas, at), append(screen, dotAt(m))
	}
	return m, canvas, screen
}

func TestTheViewFollowsTheDotAcrossAWideLine(t *testing.T) {
	// at 80 with the legend on, the eight trees are wider than the screen
	m := loaded(t, options(), tea.WindowSizeMsg{Width: 80, Height: 40}, press('?'))
	if m.layout().width <= 79 {
		t.Fatalf("the canvas fits 80 columns: %d", m.layout().width)
	}
	scroll := m.scroll
	for stage := 1; stage < len(eight); stage++ {
		var canvas, screen []int
		m, canvas, screen = frameAll(t, m, right)
		for k := range canvas {
			// every frame, the dot is on the screen, clear of the dotted ends
			if screen[k] < 3 || screen[k] > 76 {
				t.Errorf("to %s, frame %d: the dot is at cell %d of the screen: %v", eight[stage], k, screen[k], screen)
			}
		}
		if m.scroll < scroll {
			t.Errorf("to %s, the view went back: %d after %d", eight[stage], m.scroll, scroll)
		}
		scroll = m.scroll
	}
	if want := m.layout().width - 79; m.scroll != want {
		t.Errorf("at releases the view is not at the end: %d, want %d", m.scroll, want)
	}
	// round the end, the view follows the dot back to the start
	m, _, screen := frameAll(t, m, right)
	if m.stage != 0 || m.scroll != 0 || screen[len(screen)-1] != m.center(0)+1 {
		t.Errorf("round the end: stage %d, scroll %d, the dot at %v", m.stage, m.scroll, screen)
	}
}

func TestTheLabelsShortenBeforeTheLineScrolls(t *testing.T) {
	// with these fixtures and eight stages, the names fit from 132 columns, the counts from 67, and the glyphs alone
	// from 51; at 120 the line shows the counts
	for _, c := range []struct {
		width  int
		label  string // backlog's, the selected stage
		other  string // incoming's
		scroll bool
	}{
		{200, Issue + " backlog 3", Fetched + " incoming 6", false},
		{132, Issue + " backlog 3", Fetched + " incoming 6", false},
		{131, Issue + " 3", Fetched + " 6", false},
		{120, Issue + " 3", Fetched + " 6", false},
		{80, Issue + " 3", Fetched + " 6", false},
		{67, Issue + " 3", Fetched + " 6", false},
		{66, Issue, Fetched, false},
		{51, Issue, Fetched, false},
		{50, Issue, Fetched, true},
		{40, Issue, Fetched, true},
	} {
		m := loaded(t, options(), tea.WindowSizeMsg{Width: c.width, Height: 24})
		trees, line := lines(m)[2], lines(m)[1]
		if !strings.Contains(trees, "└ "+c.label+" ") && !strings.HasSuffix(trees, "└ "+c.label) ||
			!strings.Contains(trees, "└ "+c.other) {
			t.Errorf("at %d the labels are not %q and %q: %q", c.width, c.label, c.other, trees)
		}
		// one form for every stage at once
		if n := strings.Count(trees, "└ "); n != 8 && !c.scroll {
			t.Errorf("at %d, %d labels show: %q", c.width, n, trees)
		}
		if strings.Contains(line, dotted) != c.scroll {
			t.Errorf("at %d the line scrolls %v: %q", c.width, !c.scroll, line)
		}
	}
	// in the count form, the selected stage's count is drawn as its name would be
	m := loaded(t, options(), tea.WindowSizeMsg{Width: 80, Height: 24})
	if !strings.Contains(m.View().Content, Issue+" "+style.Mark("3")) {
		t.Errorf("the selected count is not marked: %q", lines(m)[2])
	}
	// the legend nodes are not shortened, and the labels keep the form the line without them takes
	m = drive(m, press('?'))
	if v := view(m); !strings.Contains(v, "└ legend") || !strings.Contains(v, Owner+"  its assignees") ||
		!strings.Contains(lines(m)[2], "├ "+Issue+" 3") {
		t.Errorf("the legend at 80:\n%s", v)
	}
}

func TestAWideLineScrollsWithDottedEndsAndKeepsTheSelectedTreeWhole(t *testing.T) {
	m := loaded(t, options(), tea.WindowSizeMsg{Width: 80, Height: 32}, press('?'))
	for i, name := range eight {
		if i > 0 {
			m = drive(m, right)
		}
		got := lines(m)
		line := got[1]
		hiddenStart, hiddenEnd := m.scroll > 0, m.scroll+79 < m.layout().width
		if strings.HasPrefix(line, " "+strings.Repeat(dotted, 3)) != hiddenStart ||
			strings.HasSuffix(line, strings.Repeat(dotted, 3)) != hiddenEnd {
			t.Errorf("at %s, scroll %d: the dotted ends are wrong: %q", name, m.scroll, line)
		}
		// the selected tree shows whole, each line under the marker
		marker := m.center(i) - m.scroll + 1
		for k, want := range m.layout().trees[i] {
			plain := ansi.Strip(want)
			row := got[2+k]
			if !slices.Contains(cellsOf(row, plain), marker) {
				t.Errorf("at %s, the tree's line %q is not whole under the marker at %d: %q", name, plain, marker, row)
			}
		}
		// at 80 the view starts at backlog, scrolls through the middle five, and ends at builds and releases
		switch name {
		case "backlog":
			if hiddenStart || !hiddenEnd {
				t.Errorf("at backlog the view is not at the start: %q", line)
			}
		case "underway", "incoming", "desk", "worktrees", "changes":
			if !hiddenStart || !hiddenEnd {
				t.Errorf("at %s the view is not in the middle: %q", name, line)
			}
		case "builds", "releases":
			if !hiddenStart || hiddenEnd {
				t.Errorf("at %s the view is not at the end: %q", name, line)
			}
		}
		// no tree shows without its marker: the screen's trees start at its markers
		markers := cellsOf(line, "○")
		markers = append(markers, cellsOf(line, "●")...)
		for _, row := range got[2:5] {
			for _, at := range cellsOf(row, "└ ") {
				if !slices.Contains(markers, at) {
					t.Errorf("at %s a tree shows at %d without its marker: %q", name, at, row)
				}
			}
		}
	}
}

func TestPageDownScrollsThroughAnOpenRowTallerThanThePaneBeforeMovingOn(t *testing.T) {
	// worktrees is the fifth stage; its first row, opened, is taller than a pane on a short screen
	m := loaded(t, options(), tea.WindowSizeMsg{Width: 120, Height: 12}, right, right, right, right, down, enter)
	if m.stage != 4 || m.stages[4].cursor != 0 || !strings.Contains(view(m), " more") {
		t.Fatalf("the open row does not overflow the pane:\n%s", view(m))
	}
	seen := map[string]bool{}
	for k := 0; k < 20 && m.stages[4].cursor == 0; k++ {
		for _, l := range lines(m) {
			seen[strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(ansi.Strip(l)), style.Focus))] = true
		}
		m = drive(m, press(tea.KeyPgDown))
	}
	// every line the open row holds was shown on the way down, and only then did the cursor move on
	for _, l := range m.lines(4) {
		if l.row == 0 && !seen[strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(ansi.Strip(l.text)), style.Focus))] {
			t.Errorf("page down never showed %q", ansi.Strip(l.text))
		}
	}
	if m.stages[4].cursor == 0 {
		t.Fatalf("page down never left the open row")
	}
	// page up comes back through the same row, to its first line
	m = drive(m, press(tea.KeyPgUp))
	for k := 0; k < 20 && strings.Contains(view(m), " above"); k++ {
		m = drive(m, press(tea.KeyPgUp))
	}
	if m.stages[4].cursor != 0 || !strings.Contains(view(m), style.Focus+" "+Branch+" 1-the-first-release") {
		t.Errorf("page up did not return to the open row's first line:\n%s", view(m))
	}
}

func TestBuildsCarryTheirStateAndOpenToTheirJobsFailedFirstThenTheirURL(t *testing.T) {
	// builds is the seventh stage, between changes and releases
	m := loaded(t, options(), right, right, right, right, right, right)
	if m.stages[m.stage].name != "builds" || !strings.Contains(lines(m)[2], Build+" 4") {
		t.Fatalf("builds is not the seventh stage:\n%s", view(m))
	}
	rows := lines(m)[4:8]
	for i, want := range [][]string{
		{Running + "  release  release/0.1.0", "5m", "…"},
		{Failed + "  kit      main", "2h", "4m 02s"},
		{Passed + "  kit      main", "24h", "3m 36s"},
		{Cancelled + "  nightly  main", "2d", "12s"},
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
	// the failed build opens to its jobs, the failed first, then its URL, dim
	m = drive(m, down, down, enter)
	v := view(m)
	order := []string{style.Focus + " " + Failed + "  kit", Failed + "  test", Passed + "  gate", Passed + "  lint",
		Cancelled + "  docs", "https://github.com/empowerite/kit/actions/runs/41309409467", Passed + "  kit"}
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
	// a build without jobs opens to its URL alone
	m = drive(m, enter, down, down, enter)
	got := lines(m)
	k := slices.IndexFunc(got, func(l string) bool { return strings.Contains(l, style.Focus+" "+Cancelled) })
	if k < 0 || strings.TrimSpace(got[k+1]) != "https://github.com/empowerite/kit/actions/runs/41309409322" ||
		strings.TrimSpace(got[k+2]) != "" {
		t.Errorf("a build without jobs does not open to its URL alone:\n%s", view(m))
	}
	// an empty stage says how far it looked
	m.find("builds").runs = nil
	if !strings.Contains(view(m), "   no build since the last 20 runs") || !strings.Contains(lines(m)[2], Build+" 0") {
		t.Errorf("the empty builds:\n%s", view(m))
	}
}

func TestBuildsRefuseADocumentWithoutIDs(t *testing.T) {
	t.Setenv("FAKE_STATUS_ABOVE", "builds")
	m := loaded(t, options(), right, right, right, right, right, right)
	want := `kit status builds --json printed an array of objects without "id" where an array of builds was ` +
		"expected; run kit status inside a repository"
	if m.stages[6].refusal != want || !strings.Contains(view(m), Build+" !") {
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
	if m.stages[m.stage].name != "underway" || !strings.Contains(lines(m)[2], InFlight+" 4") {
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
	m = drive(m, down, enter)
	v := view(m)
	order := []string{style.Focus + "  #1",
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
	m = drive(m, enter, down, enter)
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
	if v := view(m); !strings.Contains(v, "/-/issues/8") || !strings.Contains(v, "/-/merge_requests/4") {
		t.Errorf("kit's URLs do not win:\n%s", v)
	}
	// an empty stage says no issue is underway
	s.under = nil
	if !strings.Contains(view(m), "   no issue is underway") || !strings.Contains(lines(m)[2], InFlight+" 0") {
		t.Errorf("the empty underway:\n%s", view(m))
	}
}

func TestUnderwayRefusesADocumentWithoutPullRequests(t *testing.T) {
	t.Setenv("FAKE_STATUS_ABOVE", "underway")
	m := loaded(t, options(), right)
	want := `kit status underway --json printed an array of objects without "number" where an array of issues ` +
		"underway was expected; run kit status inside a repository"
	if m.stages[1].refusal != want || !strings.Contains(view(m), InFlight+" !") {
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
