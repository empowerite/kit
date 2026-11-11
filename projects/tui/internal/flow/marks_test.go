// Stale stages: a stage that loaded takes kit's next mark as its own; a mark that then differs makes it stale, a dot
// on its tab, until it loads again; a stage without a mark never is.
package flow

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/empowerite/kit/projects/tui/internal/tabs"
)

// marking writes the marks kit gives, as FAKE_FLOW_MARKS names them, and gives a function that writes them again.
func marking(t *testing.T, marks string) func(string) {
	t.Helper()
	file := filepath.Join(t.TempDir(), "marks.json")
	write := func(marks string) {
		if err := os.WriteFile(file, []byte(marks), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	write(marks)
	t.Setenv("FAKE_FLOW_MARKS", file)
	return write
}

func TestAStageWhoseMarkMovesSinceItLoadedIsStaleUntilItLoadsAgain(t *testing.T) {
	write := marking(t, `{"backlog": "b1", "changes": "c1", "builds": "u1", "releases": null}`)
	m := loaded(t, options(), tea.WindowSizeMsg{Width: 160, Height: 24})
	if s := m.find("backlog"); s.src.Mark() != "b1" || s.src.Stale {
		t.Fatalf("after the load, backlog's mark %q, stale %v", s.src.Mark(), s.src.Stale)
	}
	if len(clockOf(m).Timers()) == 0 || m.marks.Asking() || m.marks.Off() {
		t.Errorf("after the read: waits %v, asking %v, off %v", clockOf(m).Timers(), m.marks.Asking(), m.marks.Off())
	}
	// the minute's read: backlog's mark moved, changes' did not, builds' is gone, releases never had one
	write(`{"backlog": "b2", "changes": "c1", "releases": null}`)
	m = drive(m, clockOf(m).Fire()...)
	for name, want := range map[string]bool{"backlog": true, "changes": false, "builds": false, "releases": false} {
		if got := m.find(name).src.Stale; got != want {
			t.Errorf("%s stale %v, want %v", name, got, want)
		}
	}
	if names := barNames(m); !strings.Contains(names, "backlog"+tabs.Stale) || strings.Contains(names, "changes"+tabs.Stale) {
		t.Errorf("the dots: %q", names)
	}
	// stale is stale: a later read keeps it so, though the mark moves on, or back
	write(`{"backlog": "b1"}`)
	if m = drive(m, clockOf(m).Fire()...); !m.find("backlog").src.Stale {
		t.Error("a later read cleared the dot")
	}
	// Refresh All loads it again, and it takes the next mark as its own
	write(`{"backlog": "b3", "changes": "c1"}`)
	older := clockOf(m).Fire()
	m = drive(m, refreshAll...)
	if s := m.find("backlog"); s.src.Stale || s.src.Mark() != "b3" || strings.Contains(barNames(m), "backlog"+tabs.Stale) {
		t.Errorf("after Refresh All: stale %v, mark %q", s.src.Stale, s.src.Mark())
	}
	// the minute an older read waited from reads nothing
	for _, wait := range older {
		if _, cmd := m.Update(wait); cmd != nil {
			t.Error("an older minute reads the marks again")
		}
	}
}

// A stage that loaded since the last read takes the next mark as its own, whatever the one before it.
func TestAStageThatLoadedTakesTheNextMarkAsItsOwn(t *testing.T) {
	write := marking(t, `{"builds": "u1"}`)
	m := loaded(t, options(), tea.WindowSizeMsg{Width: 160, Height: 24})
	write(`{"builds": "u2"}`)
	m = drive(m, m.load("builds", false)())
	if s := m.find("builds"); s.src.Stale || s.src.Mark() != "u2" {
		t.Errorf("builds after its own load and the read after it: stale %v, mark %q", s.src.Stale, s.src.Mark())
	}
}

// A kit that cannot read the marks makes nothing stale, and is asked again only on Refresh All.
func TestAKitThatCannotReadTheMarksIsAskedAgainOnRefreshAll(t *testing.T) {
	record := filepath.Join(t.TempDir(), "calls")
	t.Setenv("FAKE_FLOW_CALLS", record)
	m := loaded(t, options(), tea.WindowSizeMsg{Width: 160, Height: 24})
	if !m.marks.Off() || m.marks.Asking() || m.marks.Got {
		t.Fatalf("without the marks: off %v, asking %v, read %v", m.marks.Off(), m.marks.Asking(), m.marks.Got)
	}
	asked := func() int {
		got, _ := os.ReadFile(record)
		return strings.Count(string(got), "flow --marks\n")
	}
	if n := asked(); n != 1 {
		t.Errorf("the stages' loads asked %d times", n)
	}
	drive(m, refreshAll...)
	if n := asked(); n != 2 {
		t.Errorf("after Refresh All, asked %d times", n)
	}
}

// A stage whose mark kit cannot read, null as kit flow --marks exits 1 when the forge cannot answer, is not known: no
// dot, and, watched, no load, read after read; the marks kit read beside it still make their stages stale.
func TestAStageWhoseMarkIsNullIsNeitherStaleNorLoadedAgain(t *testing.T) {
	record := filepath.Join(t.TempDir(), "calls")
	write := marking(t, `{"backlog": "b1", "changes": "c1"}`)
	m := loaded(t, options(), tea.WindowSizeMsg{Width: 160, Height: 24})
	m.find("backlog").src.Watch = true
	t.Setenv("FAKE_FLOW_CALLS", record)
	write(`{"backlog": null, "changes": "c2"}`)
	for range 3 {
		m = drive(m, clockOf(m).Fire()...)
	}
	if s := m.find("backlog"); s.src.Stale || strings.Contains(barNames(m), "backlog"+tabs.Stale) {
		t.Errorf("backlog of a null mark: stale %v, the dots %q", s.src.Stale, barNames(m))
	}
	if got, _ := os.ReadFile(record); strings.Contains(string(got), "backlog") {
		t.Errorf("a null mark loaded backlog again: %q", got)
	}
	if !m.find("changes").src.Stale || !strings.Contains(barNames(m), "changes"+tabs.Stale) {
		t.Errorf("changes, whose mark moved beside the null: the dots %q", barNames(m))
	}
	loads := func() int {
		got, _ := os.ReadFile(record)
		return strings.Count(string(got), "flow backlog --json")
	}
	// the forge answers again with the mark backlog took: nothing moved, so nothing loads
	write(`{"backlog": "b1", "changes": "c2"}`)
	for range 2 {
		m = drive(m, clockOf(m).Fire()...)
	}
	if n := loads(); n != 0 || m.find("backlog").src.Stale {
		t.Errorf("back to its own mark: %d loads, stale %v", n, m.find("backlog").src.Stale)
	}
	// null again, then a mark that moved: one load, read after read
	write(`{"backlog": null, "changes": "c2"}`)
	m = drive(m, clockOf(m).Fire()...)
	write(`{"backlog": "b2", "changes": "c2"}`)
	for range 3 {
		m = drive(m, clockOf(m).Fire()...)
	}
	if n := loads(); n != 1 {
		t.Errorf("a mark that moved past a null loaded backlog %d times", n)
	}
	// a mark that holds the word null is a mark: kit reads it with exit 0
	write(`{"backlog": "nullable", "changes": "c2"}`)
	if err := exec.Command(runner(t).Kit, "flow", "--marks").Run(); err != nil {
		t.Errorf("a mark holding null: %v", err)
	}
}
