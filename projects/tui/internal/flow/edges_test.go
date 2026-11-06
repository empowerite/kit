package flow

import (
	"path/filepath"
	"slices"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/empowerite/kit/projects/tui/internal/tabs"
)

// With no stage in the picture, the keys that move, open, page, and drop the tab's menu down do nothing, and the pane
// says so; = still drops the app's menu down.
func TestAPictureWithNoStageTakesEveryKeyQuietly(t *testing.T) {
	m := New(runner(t), Options{})
	for _, msg := range []tea.Msg{tea.WindowSizeMsg{Width: 120, Height: 24}, left, right, down, up, press(tea.KeyEnter),
		press(tea.KeyPgDown), press(':')} {
		next, cmd := m.Update(msg)
		m = next.(Model)
		if cmd != nil {
			t.Errorf("%v with no stage started something", msg)
		}
	}
	if !strings.Contains(view(m), "no stage is configured") || m.inRows {
		t.Errorf("no stage:\n%s", view(m))
	}
	// the app's menu still drops down, to another screen or out
	if next, _ := m.Update(press('=')); !next.(Model).appMenu.Open {
		t.Error("= with no stage did not drop the app's menu down")
	}
	// Close ends what the picture still runs, as quitting does
	if m.Close(); m.ctx.Err() == nil {
		t.Error("Close did not end the stages' commands")
	}
}

// Refresh All runs no stage kittui does not draw, and down into a stage with no rows stays on the bar.
func TestRefreshAndDownDoNothingWhereThereIsNothingToLoadOrEnter(t *testing.T) {
	m := loaded(t, Options{Stages: []string{"nonesuch", "builds"}})
	m = drive(m, right, loadedMsg{stage: "builds", data: []byte("[]")}, down)
	if m.inRows || !strings.Contains(view(m), "no build since the last 20 runs") {
		t.Errorf("down into an empty stage: in rows %v\n%s", m.inRows, view(m))
	}
	record := filepath.Join(t.TempDir(), "calls")
	t.Setenv("FAKE_FLOW_CALLS", record)
	if drive(m, refreshAll...); !slices.Equal(calls(t, record), []string{"flow builds --json"}) {
		t.Errorf("Refresh All ran %q", calls(t, record))
	}
}

// A document that is empty, that does not parse, or whose fields are of the wrong type is a refusal naming the stage.
func TestADocumentKitFlowCannotDecodeIsARefusal(t *testing.T) {
	for _, c := range []struct {
		stage, data string
	}{
		{"releases", ""},
		{"releases", `[{"version": 5}]`},
		{"releases", `[{`},
		{"desk", `{"path": 7}`},
		{"desk", `{`},
		{"builds", `[{"id": "x"}]`},
	} {
		m := loaded(t, Options{Stages: []string{c.stage}}, loadedMsg{stage: c.stage, data: []byte(c.data)})
		if s := m.find(c.stage); !strings.HasPrefix(s.refusal, "kit flow "+c.stage+" --json: ") {
			t.Errorf("%s %q: refusal %q", c.stage, c.data, s.refusal)
		}
	}
}

// Page up inside an open row taller than the pane scrolls back through that row, the cursor staying on it.
func TestPageUpInsideATallOpenRowScrollsBackThroughIt(t *testing.T) {
	m := loaded(t, options(), tea.WindowSizeMsg{Width: 120, Height: 12}, right, right, right, down, toggle,
		press(tea.KeyPgDown))
	s := m.find("worktrees")
	if s.cursor != 0 || s.offset == 0 {
		t.Fatalf("page down did not scroll into the open row: cursor %d, offset %d", s.cursor, s.offset)
	}
	before := s.offset
	m = drive(m, press(tea.KeyPgUp))
	if s := m.find("worktrees"); s.cursor != 0 || s.offset >= before {
		t.Errorf("page up inside the row: cursor %d, offset %d from %d", s.cursor, s.offset, before)
	}
}

// Refresh All while the cursor is in a stage's rows shows the stage loading, the cursor kept, and the rows come back
// with the answer.
func TestRefreshInTheRowsKeepsTheCursorThroughTheLoad(t *testing.T) {
	m := loaded(t, options(), down, press('='))
	next, cmd := m.Update(press(tea.KeyEnter))
	m = next.(Model)
	if !m.inRows || !strings.Contains(view(m), "loading") {
		t.Fatalf("Refresh All in the rows: in rows %v\n%s", m.inRows, view(m))
	}
	if m = drive(m, cmd()); !m.inRows || len(m.rows(m.stage)) == 0 || strings.Contains(view(m), "loading") {
		t.Errorf("the answer did not bring the rows back:\n%s", view(m))
	}
}

// The small drawings: a build state #48 does not name, a release kit gave a URL for, an age kit did not give, checks
// that never reported or report a state kittui does not know, a checkout with only its path, a pull request with no
// reference of kit's, and a run started by hand.
func TestTheCellsDrawWhatIsMissingOrUnknown(t *testing.T) {
	if runGlyph("queued") != Other {
		t.Errorf("an unknown state draws %q", runGlyph("queued"))
	}
	if url := (Origin{Host: "github.com", Repo: "a/b"}).release("v1", "https://x/r"); url != "https://x/r" {
		t.Errorf("a release's own URL: %q", url)
	}
	if ageOf(nil) != "" {
		t.Errorf("a missing age draws %q", ageOf(nil))
	}
	var none *checks
	if none.cell() != "" || (&checks{State: "SKIPPED"}).cell() != "" {
		t.Errorf("checks with nothing to say draw %q, %q", none.cell(), (&checks{State: "SKIPPED"}).cell())
	}
	m := New(runner(t), options())
	if r := laid("worktrees", []record{m.checkoutRow(checkout{Path: "/src/kit"}, false)}, func(string) string { return "" }); !strings.Contains(r[0].Cells[0].Values[0], "/src/kit") {
		t.Errorf("a checkout with only its path: %+v", r[0].Cells[0])
	}
	if got := ref(pull{Number: 7}); got != "#7" {
		t.Errorf("a pull request's reference: %q", got)
	}
	if got := eventOf(run{Event: "workflow_dispatch"}); got != "manual" {
		t.Errorf("a run started by hand: %q", got)
	}
}

// Every stage that is empty says what it lacks, in its own words.
func TestEveryEmptyStageSaysWhatItLacks(t *testing.T) {
	for name, want := range map[string]string{
		"backlog":   "no open issue without a pull request",
		"underway":  "no issue is underway",
		"incoming":  "no remote to fetch",
		"desk":      "no desk",
		"worktrees": "no linked worktree",
		"changes":   "no open pull request",
		"builds":    "no build since the last 20 runs",
		"releases":  "no release",
		"nonesuch":  "nothing",
	} {
		if got := empty(stage{name: name, in: &incoming{Fetched: true}}); got != want {
			t.Errorf("an empty %s says %q, not %q", name, got, want)
		}
	}
}

// On a screen too narrow for every stage, the bar scrolls, the selected stage whole.
func TestOnANarrowScreenTheBarScrolls(t *testing.T) {
	m := loaded(t, options(), tea.WindowSizeMsg{Width: 30, Height: 24})
	bar := lines(m)[2]
	if !strings.Contains(squeeze(bar), "│ 🎫 backlog 3 "+tabs.Menu+" │") || !strings.HasSuffix(bar, tabs.After) {
		t.Errorf("at 30 columns, the bar does not scroll with backlog whole: %q", bar)
	}
}
