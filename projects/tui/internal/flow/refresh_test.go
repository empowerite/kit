package flow

import (
	"context"
	"errors"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
)

// to selects the stage by name, on the bar.
func to(t *testing.T, m Model, name string) Model {
	t.Helper()
	for range m.stages {
		if m.stages[m.stage].name == name {
			return m
		}
		m = drive(m, right)
	}
	t.Fatalf("no stage %s", name)
	return m
}

// Pressing r again while a stage loads supersedes the load before: its command is ended, and its answer, should it
// still come, is dropped, so the newer load is the one the stage shows.
func TestANewerRefreshSupersedesTheOneBefore(t *testing.T) {
	t.Setenv("FAKE_FLOW_SLOW", "releases")
	m := to(t, loaded(t, options()), "releases")
	t.Setenv("FAKE_FLOW_SLOW", "")
	next, older := m.Update(press('r'))
	m = next.(Model)
	t.Setenv("FAKE_FLOW_SLOW", "releases")
	answers := make(chan tea.Msg, 1)
	go func() { answers <- older() }()
	next, newer := m.Update(press('r'))
	m = next.(Model)
	t.Setenv("FAKE_FLOW_SLOW", "")
	// the older load is told it was superseded, and the answer it gives is not taken
	var dropped tea.Msg
	select {
	case dropped = <-answers:
	case <-time.After(10 * time.Second):
		t.Fatal("the older load ran on after a newer one superseded it")
	}
	m = drive(m, dropped)
	if s := m.find("releases"); !s.loading {
		t.Errorf("the superseded answer was taken: loading %v, refusal %q", s.loading, s.refusal)
	}
	m = drive(m, newer())
	if s := m.find("releases"); s.loading || s.refusal != "" || len(s.rels) == 0 {
		t.Errorf("the newer answer was not taken: loading %v, refusal %q, %d releases", s.loading, s.refusal,
			len(s.rels))
	}
	if err := dropped.(loadedMsg).err; !errors.Is(err, context.Canceled) {
		t.Errorf("the older load ended with %v, not its cancel", err)
	}
}

// r on incoming fetches, and what the fetch brings moves the desk's and the worktrees' counts and the issues underway,
// so those stages load again once incoming answers; the others do not.
func TestRefreshingIncomingReloadsTheStagesTheFetchMoves(t *testing.T) {
	record := filepath.Join(t.TempDir(), "calls")
	m := to(t, loaded(t, options()), "incoming")
	t.Setenv("FAKE_FLOW_CALLS", record)
	m = drive(m, press('r'))
	var ran []string
	for _, call := range calls(t, record) {
		ran = append(ran, strings.Fields(call)[1])
	}
	slices.Sort(ran)
	if want := []string{"desk", "incoming", "underway", "worktrees"}; !slices.Equal(ran, want) {
		t.Errorf("r on incoming ran %v, not %v", ran, want)
	}
	for _, name := range []string{"desk", "underway", "worktrees"} {
		if s := m.find(name); s.loading || s.refusal != "" {
			t.Errorf("%s did not load again: loading %v, refusal %q", name, s.loading, s.refusal)
		}
	}
}
