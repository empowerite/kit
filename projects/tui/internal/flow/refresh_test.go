package flow

import (
	"context"
	"errors"
	"path/filepath"
	"slices"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/empowerite/kit/projects/tui/internal/shell"
	"github.com/empowerite/kit/projects/tui/internal/source"
)

// to selects the stage by name, on the bar.
func to(t *testing.T, m Model, name string) Model {
	t.Helper()
	for range m.stages {
		if m.stages[m.stage].name == name {
			return m
		}
		m = drive(m, tab)
	}
	t.Fatalf("no stage %s", name)
	return m
}

// Refreshing a stage again while it loads supersedes the load before: its command is ended, and its answer, should it
// still come, is dropped, so the newer load is the one the stage shows.
func TestANewerRefreshSupersedesTheOneBefore(t *testing.T) {
	t.Setenv("FAKE_FLOW_SLOW", "releases")
	m := to(t, loaded(t, options()), "releases")
	t.Setenv("FAKE_FLOW_SLOW", "")
	older := m.restart("releases")
	t.Setenv("FAKE_FLOW_SLOW", "releases")
	answers := make(chan tea.Msg, 1)
	go func() { answers <- older() }()
	newer := m.restart("releases")
	t.Setenv("FAKE_FLOW_SLOW", "")
	// the older load is told it was superseded, and the answer it gives is not taken
	var dropped tea.Msg
	select {
	case dropped = <-answers:
	case <-time.After(10 * time.Second):
		t.Fatal("the older load ran on after a newer one superseded it")
	}
	m = drive(m, dropped)
	if s := m.find("releases"); !s.src.Stale || len(s.rows) == 0 {
		t.Errorf("the superseded answer was taken: stale %v, refusal %q", s.src.Stale, s.refusal)
	}
	m = drive(m, newer())
	if s := m.find("releases"); s.src.Loading || s.src.Stale || s.refusal != "" || len(s.rows) == 0 {
		t.Errorf("the newer answer was not taken: loading %v, refusal %q, %d releases", s.src.Loading, s.refusal,
			len(s.rows))
	}
	if err := dropped.(source.Answer[[]byte]).Err; !errors.Is(err, context.Canceled) {
		t.Errorf("the older load ended with %v, not its cancel", err)
	}
}

// Refresh All reads the forge afresh for issues and changes, past kit's shared snapshot of the open issues
// and pull requests; the stages that never share it are not told.
func TestRefreshAllReadsTheForgeAfreshWhereTheStagesShareItsRead(t *testing.T) {
	record := filepath.Join(t.TempDir(), "calls")
	m := loaded(t, options())
	t.Setenv("FAKE_FLOW_CALLS", record)
	drive(m, refreshAll...)
	got := calls(t, record)
	slices.Sort(got)
	want := []string{"flow builds --json", "flow changes --json --fresh", "flow incoming --json",
		"flow issues --json --fresh", "flow releases --json"}
	if !slices.Equal(got, want) {
		t.Errorf("Refresh All ran %q, not %q", got, want)
	}
}

// The tab's menu's Refresh runs its stage again, reading the forge afresh; on issues or changes, which share kit's one
// read of the open issues and pull requests, it runs both afresh, so they stay one snapshot.
func TestRefreshInTheTabsMenuRunsItsStageAndTheStagesThatShareItsRead(t *testing.T) {
	for _, c := range []struct {
		stage string
		want  []string
	}{
		{"changes", []string{"flow changes --json --fresh", "flow issues --json --fresh"}},
		{"builds", []string{"flow builds --json"}},
	} {
		record := filepath.Join(t.TempDir(), "calls")
		m := to(t, loaded(t, options()), c.stage)
		t.Setenv("FAKE_FLOW_CALLS", record)
		m = choose(t, m, "Refresh")
		got := calls(t, record)
		slices.Sort(got)
		if !slices.Equal(got, c.want) || m.tabMenu.Open {
			t.Errorf("refresh on %s ran %q, not %q", c.stage, got, c.want)
		}
	}
}

// On Changes, Refresh on reviews or checks reads the forge afresh, as changes does, and runs the three again, as they
// share kit's one read of the open changes, so a review given within the minute shows; merged reads no snapshot and
// runs alone.
func TestRefreshOnReviewsOrChecksReadsTheOpenChangesAfresh(t *testing.T) {
	shared := []string{"changes checks --json --fresh", "changes reviews --json --fresh", "flow changes --json --fresh"}
	for _, c := range []struct {
		stage string
		want  []string
	}{
		{"changes/reviews", shared},
		{"changes/checks", shared},
		{"changes/merged", []string{"changes merged --json"}},
	} {
		record := filepath.Join(t.TempDir(), "calls")
		m := to(t, appOf(t, shell.Changes, tea.WindowSizeMsg{Width: 160, Height: 30}), c.stage)
		t.Setenv("FAKE_FLOW_CALLS", record)
		m = choose(t, m, "Refresh")
		got := calls(t, record)
		slices.Sort(got)
		if !slices.Equal(got, c.want) {
			t.Errorf("refresh on %s ran %q, not %q", c.stage, got, c.want)
		}
	}
}
