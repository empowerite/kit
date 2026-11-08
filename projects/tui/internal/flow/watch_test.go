// Watch: a stage watched loads again once its mark moves, keeping its rows and its dot until the new ones come. Watch in
// the tab's menu watches one stage, and reads Stop Watching while it is; Watch All in the app's menu watches every one,
// and Stop Watching All stops them, each dim when it would do nothing. The keys bar's right end says watching while the
// selected stage is watched. Nothing is watched at the start.
package flow

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/empowerite/kit/projects/tui/internal/shell"
	"github.com/empowerite/kit/projects/tui/internal/style"
	"github.com/empowerite/kit/projects/tui/internal/testkit"
)

// own gives the app's menu's own line by name, and whether it acts.
func own(m Model, name string) (shell.Item, bool) {
	items := m.appMenu.Items()
	k := slices.IndexFunc(items, func(it shell.Item) bool { return it.Name == name })
	return items[k], !slices.Contains(m.appMenu.Dim, name)
}

func TestNothingIsWatchedAtTheStart(t *testing.T) {
	m := loaded(t, options(), tea.WindowSizeMsg{Width: 160, Height: 24})
	for _, s := range m.stages {
		if s.watch {
			t.Errorf("%s is watched at the start", s.name)
		}
	}
	if got := m.watchItem(m.stage); got != (shell.Item{Glyph: Unwatched, Name: WatchIt}) {
		t.Errorf("the tab's Watch: %+v", got)
	}
	// Watch All acts and Stop Watching All is dim, both under Refresh All
	if all, on := own(m, WatchAll); all.Glyph != Unwatched || !on {
		t.Errorf("Watch All: %+v, acts %v", all, on)
	}
	if stop, on := own(m, StopAll); stop.Glyph != StopWatched || on {
		t.Errorf("Stop Watching All: %+v, acts %v", stop, on)
	}
	items := m.appMenu.Items()
	at := slices.IndexFunc(items, func(it shell.Item) bool { return it.Name == RefreshAll })
	if at < 0 || items[at+1].Name != WatchAll || items[at+2].Name != StopAll {
		t.Errorf("the app's menu: %+v", items)
	}
	// the keys bar's right end says nothing of Watch
	if last := lines(m)[len(lines(m))-1]; strings.Contains(last, "watching") || m.watchedText() != "" {
		t.Errorf("the keys bar with nothing watched: %q", last)
	}
	// a picture of no stage kittui draws has nothing to watch, nor to stop
	none := New(runner(t), Options{Stages: []string{"nonesuch"}})
	if _, on := own(none, WatchAll); on {
		t.Error("Watch All acts with no stage to watch")
	}
	if _, on := own(none, StopAll); on {
		t.Error("Stop Watching All acts with no stage to watch")
	}
	// Watch All passes over a stage kittui does not draw
	some := New(runner(t), Options{Stages: []string{"nonesuch", "desk"}})
	if some.watchAll(true); some.stages[0].watch || !some.stages[1].watch {
		t.Errorf("Watch All with a stage kittui does not draw: %v %v", some.stages[0].watch, some.stages[1].watch)
	}
}

func TestWatchInTheTabsMenuWatchesTheStageThenReadsStopWatching(t *testing.T) {
	m := loaded(t, options(), tea.WindowSizeMsg{Width: 160, Height: 24}, press(':'))
	k := item(m, WatchIt)
	if k < 0 || !m.enabled(k) {
		t.Fatalf("Watch is not in the tab's menu, or dim: %d", k)
	}
	m.tabMenu.At = k
	if m = drive(m, press(tea.KeyEnter)); !m.stages[m.stage].watch || m.tabMenu.Open ||
		m.watchItem(m.stage) != (shell.Item{Glyph: StopWatched, Name: StopIt}) {
		t.Fatalf("Watch: watched %v, menu %v, item %+v", m.stages[m.stage].watch, m.tabMenu.Open, m.watchItem(m.stage))
	}
	// the keys bar's right end says the stage is watched, in the accent; another stage, not watched, says nothing
	if last := lines(m)[len(lines(m))-1]; !strings.Contains(last, Unwatched+" watching") ||
		m.watchedText() != style.Tint(Unwatched+" watching") {
		t.Errorf("the keys bar while the stage is watched: %q", last)
	}
	if n := drive(m, right); strings.Contains(lines(n)[len(lines(n))-1], "watching") {
		t.Errorf("the keys bar on a stage not watched: %q", lines(n)[len(lines(n))-1])
	}
	// one stage watched: both of the app's Watch lines act
	if _, on := own(m, WatchAll); !on {
		t.Error("Watch All is dim with one stage watched")
	}
	if _, on := own(m, StopAll); !on {
		t.Error("Stop Watching All is dim with one stage watched")
	}
	m = drive(m, press(':'))
	if m.tabMenu.At = item(m, StopIt); m.tabMenu.At < 0 || !m.enabled(m.tabMenu.At) {
		t.Fatalf("Stop Watching is not in the tab's menu, or dim: %d", m.tabMenu.At)
	}
	if m = drive(m, press(tea.KeyEnter)); m.stages[m.stage].watch || item(m, WatchIt) < 0 {
		t.Error("Stop Watching did not stop")
	}
}

func TestWatchAllWatchesEveryStageAndStopWatchingAllStopsThemAll(t *testing.T) {
	m := loaded(t, options())
	m.stages[2].watch = true
	m.syncOwn()
	if cmd := m.chose(shell.Choice{Own: WatchAll}); cmd != nil {
		m = drive(m, cmd())
	}
	for _, s := range m.stages {
		if !s.watch {
			t.Errorf("%s is not watched after Watch All", s.name)
		}
	}
	// every stage watched: Watch All is dim, Stop Watching All acts, and each tab's line reads Stop Watching
	if _, on := own(m, WatchAll); on {
		t.Error("Watch All acts while every stage is watched")
	}
	if _, on := own(m, StopAll); !on || m.watchItem(4).Name != StopIt {
		t.Errorf("Stop Watching All acts %v, a tab's line %+v", on, m.watchItem(4))
	}
	m.chose(shell.Choice{Own: StopAll})
	if slices.ContainsFunc(m.stages, func(s stage) bool { return s.watch }) {
		t.Error("Stop Watching All left a stage watched")
	}
	// each chosen from the app's menu by a click; a click on one that is dim does nothing
	click := func(m Model, name string) Model {
		at := slices.IndexFunc(m.appMenu.Items(), func(it shell.Item) bool { return it.Name == name })
		return drive(m, press('='), testkit.Click(3, shell.Top+1+at))
	}
	if m = click(m, StopAll); !m.appMenu.Open {
		t.Error("a click on the dim Stop Watching All closed the menu")
	}
	if m = drive(m, esc); click(m, WatchAll).find("desk").watch != true {
		t.Error("a click on Watch All did not watch")
	}
	m = click(m, WatchAll)
	if m = click(m, StopAll); m.find("desk").watch || m.appMenu.Open {
		t.Errorf("a click on Stop Watching All: desk watched %v, menu open %v", m.find("desk").watch, m.appMenu.Open)
	}
}

// A watched stage whose mark moves loads again at once, reading the forge afresh, its rows and its dot staying until
// the new ones come; a stage not watched only goes stale.
func TestAWatchedStageLoadsAgainOnceItsMarkMoves(t *testing.T) {
	record := filepath.Join(t.TempDir(), "calls")
	t.Setenv("FAKE_FLOW_CALLS", record)
	write := marking(t, `{"backlog": "b1", "releases": "r1"}`)
	m := loaded(t, options(), tea.WindowSizeMsg{Width: 160, Height: 24})
	m.find("backlog").watch = true
	write(`{"backlog": "b2", "releases": "r2"}`)
	// the minute passes, and the read of the marks it asks answers
	next, ask := m.Update(latest(m))
	next, cmd := next.(Model).Update(ask())
	m = next.(Model)
	b := m.find("backlog")
	if !b.stale || b.loading || len(b.issues) == 0 || cmd == nil {
		t.Fatalf("backlog while it loads again: stale %v, loading %v, %d issues", b.stale, b.loading, len(b.issues))
	}
	if !strings.Contains(raw(m), "Fix the thing") {
		t.Errorf("backlog's rows did not stay:\n%s", raw(m))
	}
	m = drive(m, cmd())
	got, _ := os.ReadFile(record)
	if !strings.Contains(string(got), "flow backlog --json --fresh\n") {
		t.Errorf("backlog did not load again afresh:\n%s", got)
	}
	if b := m.find("backlog"); b.stale || len(b.issues) != 3 {
		t.Errorf("backlog after it loaded again: stale %v, %d issues", b.stale, len(b.issues))
	}
	if r := m.find("releases"); !r.stale || strings.Contains(string(got), "flow releases --json --fresh") {
		t.Errorf("releases, not watched: stale %v", r.stale)
	}
}

// Watching a stage already stale loads it again at once; Watch All does the same for each stale stage. A stage still
// loading is not loaded again for it.
func TestWatchingAStaleStageLoadsItAtOnce(t *testing.T) {
	m := loaded(t, options())
	m.stages[m.stage].stale = true
	cmd := m.toggleWatch(m.stage)
	if cmd == nil {
		t.Fatal("watching a stale stage loaded nothing")
	}
	if m = drive(m, cmd()); m.stages[m.stage].stale {
		t.Error("the stale stage is still stale after it loaded again")
	}
	m.toggleWatch(m.stage)
	m.find("releases").stale = true
	m.find("desk").stale, m.find("desk").loading = true, true
	cmd = m.watchAll(true)
	if cmd == nil {
		t.Fatal("Watch All loaded no stale stage")
	}
	if m = drive(m, cmd()); m.find("releases").stale {
		t.Error("releases is still stale after Watch All")
	}
	// a stage loading is not loaded again: watching it alone starts nothing
	m.watchAll(false)
	l := m.find("worktrees")
	l.stale, l.loading = true, true
	if cmd := m.toggleWatch(slices.IndexFunc(m.stages, func(s stage) bool { return s.name == "worktrees" })); cmd != nil {
		t.Error("watching a stage still loading loaded it again")
	}
}
