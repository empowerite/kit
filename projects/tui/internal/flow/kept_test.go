// The last picture: a stage is drawn from what kit kept of its last load while its own load runs, a dot on its tab,
// and its load replaces it and clears the dot; a load that lands first leaves what was kept unread.
package flow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/empowerite/kit/projects/tui/internal/tabs"
)

// keeping gives a directory of what kit kept, as FAKE_FLOW_KEPT names it: backlog as it was before #45 and #46 were
// filed, #7 alone.
func keeping(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	kept := `[{"number": 7, "title": "Fix the thing, as kept", "assignees": ["ada"], "age": 3456000, "source": "github"}]`
	if err := os.WriteFile(filepath.Join(dir, "backlog.json"), []byte(kept), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("FAKE_FLOW_KEPT", dir)
}

// barNames gives the tab bar's middle line, the stages' names, without its escapes.
func barNames(m Model) string { return strings.Split(ansi.Strip(m.bar().View()), "\n")[1] }

func TestAStageIsDrawnStaleFromWhatWasKeptUntilItsLoadLands(t *testing.T) {
	keeping(t)
	m := drive(New(runner(t), options()), tea.WindowSizeMsg{Width: 160, Height: 24})
	m = drive(m, m.kept("backlog")(), m.kept("changes")())
	s := m.find("backlog")
	if s.src.Loading || !s.src.Stale || len(s.issues) != 1 || !strings.Contains(raw(m), "Fix the thing, as kept") {
		t.Fatalf("the kept backlog: loading %v, stale %v, %d issues\n%s", s.src.Loading, s.src.Stale, len(s.issues), raw(m))
	}
	if names := barNames(m); !strings.Contains(names, "backlog"+tabs.Stale) || strings.Contains(names, "changes"+tabs.Stale) {
		t.Errorf("the dots: %q", names)
	}
	// changes kept nothing: it goes on loading
	if d := m.find("changes"); !d.src.Loading || d.src.Stale {
		t.Errorf("changes without a kept picture: loading %v, stale %v", d.src.Loading, d.src.Stale)
	}
	m = drive(m, m.load("backlog", false)())
	if s := m.find("backlog"); s.src.Stale || len(s.issues) != 3 || strings.Contains(raw(m), "as kept") ||
		strings.Contains(barNames(m), "backlog"+tabs.Stale) {
		t.Errorf("after its load: stale %v, %d issues\n%s", s.src.Stale, len(s.issues), raw(m))
	}
}

func TestALoadThatLandsFirstLeavesWhatWasKeptUnread(t *testing.T) {
	keeping(t)
	m := drive(New(runner(t), options()), tea.WindowSizeMsg{Width: 160, Height: 24})
	m = drive(m, m.load("backlog", false)(), m.kept("backlog")())
	if s := m.find("backlog"); s.src.Stale || len(s.issues) != 3 || strings.Contains(raw(m), "as kept") {
		t.Errorf("kept after the load: stale %v, %d issues", s.src.Stale, len(s.issues))
	}
	// what a load that began before the one now running kept is not drawn either
	n := drive(New(runner(t), options()), tea.WindowSizeMsg{Width: 160, Height: 24})
	old := n.kept("backlog")()
	n.find("backlog").src.Again(n.env(), false)
	if n = drive(n, old); n.find("backlog").src.Stale {
		t.Error("what an older load kept is drawn")
	}
}

// The start reads what was kept of every stage at once, before their loads, and their loads win.
func TestTheStartReadsWhatWasKeptOfEveryStage(t *testing.T) {
	keeping(t)
	record := filepath.Join(t.TempDir(), "calls")
	t.Setenv("FAKE_FLOW_CALLS", record)
	m := loaded(t, options())
	got, _ := os.ReadFile(record)
	for _, s := range m.stages {
		if known[s.name] && !strings.Contains(string(got), "flow "+s.name+" --json") {
			t.Errorf("%s was not read", s.name)
		}
	}
	if n := strings.Count(string(got), " --cached\n"); n != len(m.stages) {
		t.Errorf("%d stages' kept pictures read, of %d:\n%s", n, len(m.stages), got)
	}
	if s := m.find("backlog"); s.src.Stale || len(s.issues) != 3 {
		t.Errorf("backlog after the start: stale %v, %d issues", s.src.Stale, len(s.issues))
	}
}

// Over a group, a member kit kept nothing of is no failure: what was kept is not drawn, and the stage goes on loading.
func TestOverAGroupWhatWasKeptOfOnlySomeMembersIsNotDrawn(t *testing.T) {
	dir := t.TempDir()
	kept := `{"rows": [{"number": 7, "title": "kept of a", "member": "github.com/o/a"}], "errors": [{"member":
		"github.com/o/b", "error": "kit flow backlog has printed nothing here yet; run it without --cached"}]}`
	if err := os.WriteFile(filepath.Join(dir, "backlog.json"), []byte(kept), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("FAKE_FLOW_KEPT", dir)
	m := over(t, "backlog")
	m = drive(m, m.kept("backlog")())
	if s := m.find("backlog"); !s.src.Loading || s.src.Stale {
		t.Errorf("backlog: loading %v, stale %v", s.src.Loading, s.src.Stale)
	}
	if v := view(m); strings.Contains(v, "printed nothing here yet") {
		t.Errorf("a kept failure is drawn:\n%s", v)
	}
}
