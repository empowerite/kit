package flow

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/empowerite/kit/projects/tui/internal/tabs"
	"github.com/empowerite/kit/projects/tui/internal/testkit"
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
	m = drive(m, tab, answer("builds", "[]"), down)
	if m.inRows || !strings.Contains(view(m), "no build since the last 20 runs") {
		t.Errorf("down into an empty stage: in rows %v\n%s", m.inRows, view(m))
	}
	record := filepath.Join(t.TempDir(), "calls")
	t.Setenv("FAKE_FLOW_CALLS", record)
	if drive(m, refreshAll...); !slices.Equal(calls(t, record), []string{"flow builds --json"}) {
		t.Errorf("Refresh All ran %q", calls(t, record))
	}
}

// A document that does not parse, or whose fields are of the wrong type, is a refusal naming the stage; nothing is
// one too, as TestNothingOrNullOnStdoutIsARefusalNotAnEmptyStage says.
func TestADocumentKitFlowCannotDecodeIsARefusal(t *testing.T) {
	for _, c := range []struct {
		stage, data string
	}{
		{"releases", `[{"key": "v", "version": 5}]`},
		{"releases", `[{`},
		{"backlog", `[{"number": "x"}]`},
		{"incoming", `{`},
		{"builds", `[{"id": "x"}]`},
	} {
		m := loaded(t, Options{Stages: []string{c.stage}}, answer(c.stage, c.data))
		if s := m.find(c.stage); !strings.HasPrefix(s.refusal, "kit flow "+c.stage+" --json: ") {
			t.Errorf("%s %q: refusal %q", c.stage, c.data, s.refusal)
		}
	}
}

// Page up inside an open row taller than the pane scrolls back through that row, the cursor staying on it.
func TestPageUpInsideATallOpenRowScrollsBackThroughIt(t *testing.T) {
	m := loaded(t, options(), tea.WindowSizeMsg{Width: 120, Height: 12})
	m = drive(m, tallBuild(m, 5), tab, tab, tab, down, down, toggle, press(tea.KeyPgDown))
	s := m.find("builds")
	if s.Cursor != 1 || s.Offset == 0 {
		t.Fatalf("page down did not scroll into the open row: cursor %d, offset %d", s.Cursor, s.Offset)
	}
	before := s.Offset
	m = drive(m, press(tea.KeyPgUp))
	if s := m.find("builds"); s.Cursor != 1 || s.Offset >= before {
		t.Errorf("page up inside the row: cursor %d, offset %d from %d", s.Cursor, s.Offset, before)
	}
}

// Refresh All while the cursor is in a stage's rows keeps the rows on the screen, stale, the cursor kept, and the
// answer replaces them.
func TestRefreshInTheRowsKeepsTheCursorThroughTheLoad(t *testing.T) {
	m := drive(intoRows(loaded(t, options())), press('='), typed("r")[0], typed("r")[0])
	next, cmd := m.Update(press(tea.KeyEnter))
	m = next.(Model)
	if !m.inRows || strings.Contains(view(m), "loading") || len(m.rows(m.stage)) == 0 || !m.stages[m.stage].src.Stale {
		t.Fatalf("Refresh All in the rows: in rows %v, stale %v\n%s", m.inRows, m.stages[m.stage].src.Stale, view(m))
	}
	if m = drive(m, cmd()); !m.inRows || len(m.rows(m.stage)) == 0 || strings.Contains(view(m), "loading") {
		t.Errorf("the answer did not bring the rows back:\n%s", view(m))
	}
}

// The small drawings: a build state #48 does not name, an age kit did not give, checks
// that never reported or report a state kittui does not know, a pull request with no reference of kit's, and a run
// started by hand.
func TestTheCellsDrawWhatIsMissingOrUnknown(t *testing.T) {
	if runGlyph("queued") != Other {
		t.Errorf("an unknown state draws %q", runGlyph("queued"))
	}
	if ageOf(nil) != "" {
		t.Errorf("a missing age draws %q", ageOf(nil))
	}
	var none *checks
	if none.cell() != "" || (&checks{State: "SKIPPED"}).cell() != "" {
		t.Errorf("checks with nothing to say draw %q, %q", none.cell(), (&checks{State: "SKIPPED"}).cell())
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
		"backlog":           "no open issue without a pull request",
		"underway":          "no issue is underway",
		"incoming":          "no remote to fetch",
		"changes":           "no open pull request",
		"builds":            "no build since the last 20 runs",
		"releases":          "no release",
		"workspace/repos":   "no repository",
		"workspace/remotes": "nothing",
		"packages/packages": "no package in reach",
		"nonesuch":          "nothing",
	} {
		if got := empty(stage{name: name}); got != want {
			t.Errorf("an empty %s says %q, not %q", name, got, want)
		}
	}
}

// On a screen too narrow for every stage, the bar scrolls, the selected stage whole, and the next cut at the edge; ▹,
// which would cut backlog, is hidden.
func TestOnANarrowScreenTheBarScrolls(t *testing.T) {
	m := loaded(t, options(), tea.WindowSizeMsg{Width: 30, Height: 24})
	bar := lines(m)[2]
	if !strings.Contains(squeeze(bar), "│ 🎫 backlog 3 "+tabs.Menu+" │") || strings.Contains(bar, tabs.After) ||
		!strings.Contains(bar, "🛫 u") {
		t.Errorf("at 30 columns, the bar does not scroll with backlog whole: %q", bar)
	}
}

// Nothing and null are what kit printed when it printed nothing: a refusal, the stage's line, never an empty stage.
func TestNothingOrNullOnStdoutIsARefusalNotAnEmptyStage(t *testing.T) {
	for what, err := range map[string]error{
		"null for backlog":     (&stage{name: "backlog"}).decode([]byte("null"), false),
		"nothing for incoming": (&stage{name: "incoming"}).decode(nil, false),
		"rows of null":         (&stage{name: "backlog"}).decode([]byte(`{"rows": null}`), true),
	} {
		if err == nil || !strings.Contains(err.Error(), "printed n") {
			t.Errorf("%s: %v", what, err)
		}
	}
	m := loaded(t, Options{Stages: []string{"releases"}}, answer("releases", ""))
	if s := m.find("releases"); s.refusal != "kit flow releases --json printed nothing where an array was expected" {
		t.Errorf("releases of nothing: refusal %q", s.refusal)
	}
}

// A stage kit read in part, its document with exit 1, draws what was read and names the part that failed where it
// stands: a remote whose fetch failed as its ❌ line among the remotes, and a member kit could not read as its ❌ line
// among the rows. Neither is the stage's refusal, and the fetch that ran moves the stages it moves.
func TestAStageReadInPartDrawsWhatWasReadAndNamesWhatWasNot(t *testing.T) {
	part := filepath.Join(t.TempDir(), "part.json")
	write := func(doc string) {
		if err := os.WriteFile(part, []byte(doc), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("FAKE_FLOW_PART_JSON", part)
	write(`{"fetched": true, "primary": "origin", "remotes": [
		{"name": "origin", "forge": "github", "host": "github.com", "default": "main",
		 "commits": [{"sha": "a1b2c3d", "subject": "#1 docs: what was fetched"}],
		 "branches": {"new": [], "updated": [], "gone": []}, "tags": []},
		{"name": "fork", "forge": "github", "host": "github.com", "default": null, "error": "fatal: no route to host",
		 "commits": [], "branches": {"new": [], "updated": [], "gone": []}, "tags": []}],
		"rows": [{"key": "origin remote", "remote": "origin", "kind": "remote", "change": "fetched", "ref": "main"},
		 {"key": "origin commit a1b2c3d", "remote": "origin", "kind": "commit", "change": "new", "ref": "main",
		  "sha": "a1b2c3d", "subject": "#1 docs: what was fetched"},
		 {"key": "fork remote", "remote": "fork", "kind": "remote", "change": "failed",
		  "subject": "fatal: no route to host"}]}`)
	t.Setenv("FAKE_FLOW_PART", "incoming")
	m := loaded(t, Options{Stages: []string{"incoming", "changes"}, NoPresets: true, Clock: &testkit.Clock{}})
	s, v := m.find("incoming"), view(m)
	if s.refusal != "" || !strings.Contains(v, Failed+" failed") || !strings.Contains(v, "fatal: no route to host") ||
		!strings.Contains(v, "what was fetched") {
		t.Errorf("incoming read in part: refusal %q\n%s", s.refusal, v)
	}
	write(`{"rows": [{"key": "origin", "remote": "origin", "forge": "github", "repository": "o/a",
		"member": "github.com/o/a", "host": "github.com", "url": "https://github.com/o/a.git", "links": {}}],
		"errors": [{"member": "github.com/o/b", "error": "/code/b is gone; run: kit workspaces --show w"}]}`)
	t.Setenv("FAKE_FLOW_PART", "workspace/remotes")
	m = loaded(t, Options{Stages: []string{"workspace/remotes"}, NoPresets: true, Clock: &testkit.Clock{}})
	if s, v := m.find("workspace/remotes"), view(m); s.refusal != "" || len(s.rows) != 1 ||
		!strings.Contains(v, Failed+" github.com/o/b: /code/b is gone") {
		t.Errorf("remotes read in part: refusal %q, rows %d\n%s", s.refusal, len(s.rows), v)
	}
}
