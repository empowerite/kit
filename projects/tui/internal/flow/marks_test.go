// Stale stages: a stage that loaded takes kit's next mark as its own; a mark that then differs makes it stale, a dot
// on its tab, until it loads again; a stage without a mark never is.
package flow

import (
	"os"
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
	write := marking(t, `{"backlog": "b1", "changes": "c1", "desk": "d1", "releases": null}`)
	m := loaded(t, options(), tea.WindowSizeMsg{Width: 160, Height: 24})
	if s := m.find("backlog"); s.mark != "b1" || s.stale || s.remark {
		t.Fatalf("after the load, backlog's mark %q, stale %v, remark %v", s.mark, s.stale, s.remark)
	}
	if m.marksSerial == 0 || m.marking || m.marksOff {
		t.Errorf("after the read: serial %d, marking %v, off %v", m.marksSerial, m.marking, m.marksOff)
	}
	// the minute's read: backlog's mark moved, changes' did not, desk's is gone, releases never had one
	write(`{"backlog": "b2", "changes": "c1", "releases": null}`)
	m = drive(m, quotaTickMsg{}, marksTickMsg{m.marksSerial})
	for name, want := range map[string]bool{"backlog": true, "changes": false, "desk": false, "releases": false} {
		if got := m.find(name).stale; got != want {
			t.Errorf("%s stale %v, want %v", name, got, want)
		}
	}
	if names := barNames(m); !strings.Contains(names, "backlog"+tabs.Stale) || strings.Contains(names, "changes"+tabs.Stale) {
		t.Errorf("the dots: %q", names)
	}
	// stale is stale: a later read keeps it so, though the mark moves on, or back
	write(`{"backlog": "b1"}`)
	if m = drive(m, marksTickMsg{m.marksSerial}); !m.find("backlog").stale {
		t.Error("a later read cleared the dot")
	}
	// Refresh All loads it again, and it takes the next mark as its own
	write(`{"backlog": "b3", "changes": "c1"}`)
	m = drive(m, refreshAll...)
	if s := m.find("backlog"); s.stale || s.mark != "b3" || strings.Contains(barNames(m), "backlog"+tabs.Stale) {
		t.Errorf("after Refresh All: stale %v, mark %q", s.stale, s.mark)
	}
	// a tick an older read waits on does nothing
	if _, cmd := m.Update(marksTickMsg{m.marksSerial - 1}); cmd != nil {
		t.Error("a stale tick reads the marks again")
	}
}

// A stage that loaded since the last read takes the next mark as its own, whatever the one before it.
func TestAStageThatLoadedTakesTheNextMarkAsItsOwn(t *testing.T) {
	write := marking(t, `{"desk": "d1"}`)
	m := loaded(t, options(), tea.WindowSizeMsg{Width: 160, Height: 24})
	write(`{"desk": "d2"}`)
	m = drive(m, m.load("desk", false)())
	if s := m.find("desk"); s.stale || s.mark != "d2" {
		t.Errorf("desk after its own load and the read after it: stale %v, mark %q", s.stale, s.mark)
	}
}

// A kit that cannot read the marks makes nothing stale, and is asked again only on Refresh All.
func TestAKitThatCannotReadTheMarksIsAskedAgainOnRefreshAll(t *testing.T) {
	record := filepath.Join(t.TempDir(), "calls")
	t.Setenv("FAKE_FLOW_CALLS", record)
	m := loaded(t, options(), tea.WindowSizeMsg{Width: 160, Height: 24})
	if !m.marksOff || m.marking || m.marked {
		t.Fatalf("without the marks: off %v, marking %v, marked %v", m.marksOff, m.marking, m.marked)
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
