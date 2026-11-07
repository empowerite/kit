// Watch: a stage watched loads again once its mark moves, keeping its rows and its dot until the new ones come. Watch in
// the tab's menu watches one stage; Watch All in the app's menu every one, or none once every one is. Nothing is
// watched at the start.
package flow

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/empowerite/kit/projects/tui/internal/shell"
	"github.com/empowerite/kit/projects/tui/internal/testkit"
)

// watchAllItem gives the app's menu's Watch All.
func watchAllItem(m Model) shell.Item {
	items := m.appMenu.Items()
	return items[slices.IndexFunc(items, func(it shell.Item) bool { return it.Name == WatchAll })]
}

func TestNothingIsWatchedAtTheStart(t *testing.T) {
	m := loaded(t, options())
	for _, s := range m.stages {
		if s.watch {
			t.Errorf("%s is watched at the start", s.name)
		}
	}
	if got := m.watchItem(m.stage); got != (shell.Item{Glyph: Unwatched, Name: WatchIt}) {
		t.Errorf("the tab's Watch: %+v", got)
	}
	if got := watchAllItem(m); got.Glyph != Unwatched || m.watchingAll() {
		t.Errorf("Watch All: %+v, watching all %v", got, m.watchingAll())
	}
	// Watch All stands under Refresh All
	items := m.appMenu.Items()
	at := slices.IndexFunc(items, func(it shell.Item) bool { return it.Name == RefreshAll })
	if at < 0 || items[at+1].Name != WatchAll {
		t.Errorf("the app's menu: %+v", items)
	}
	// a picture of no stage kittui draws watches none of them all
	if none := New(runner(t), Options{Stages: []string{"nonesuch"}}); none.watchingAll() {
		t.Error("a picture without a stage to watch watches them all")
	}
	// Watch All passes over a stage kittui does not draw
	some := New(runner(t), Options{Stages: []string{"nonesuch", "desk"}})
	if some.watchAll(); some.stages[0].watch || !some.stages[1].watch || !some.watchingAll() {
		t.Errorf("Watch All with a stage kittui does not draw: %v %v", some.stages[0].watch, some.stages[1].watch)
	}
}

func TestWatchInTheTabsMenuWatchesTheStageOrStops(t *testing.T) {
	m := loaded(t, options(), press(':'))
	k := item(m, WatchIt)
	if k < 0 || !m.enabled(k) {
		t.Fatalf("Watch is not in the tab's menu, or dim: %d", k)
	}
	m.tabMenu.At = k
	if m = drive(m, press(tea.KeyEnter)); !m.stages[m.stage].watch || m.tabMenu.Open ||
		m.watchItem(m.stage) != (shell.Item{Glyph: Checked, Name: WatchIt}) {
		t.Fatalf("Watch: watched %v, menu %v, item %+v", m.stages[m.stage].watch, m.tabMenu.Open, m.watchItem(m.stage))
	}
	// one stage watched is not every one
	if watchAllItem(m).Glyph != Unwatched {
		t.Errorf("Watch All with one stage watched: %+v", watchAllItem(m))
	}
	m = drive(m, press(':'))
	m.tabMenu.At = item(m, WatchIt)
	if m = drive(m, press(tea.KeyEnter)); m.stages[m.stage].watch {
		t.Error("Watch again did not stop")
	}
}

func TestWatchAllWatchesEveryStageOrStopsThemAll(t *testing.T) {
	m := loaded(t, options())
	m.stages[2].watch = true
	m.syncOwn()
	// with some watched, Watch All watches every one
	if cmd := m.chose(shell.Choice{Own: WatchAll}); cmd != nil {
		m = drive(m, cmd())
	}
	for _, s := range m.stages {
		if !s.watch {
			t.Errorf("%s is not watched after Watch All", s.name)
		}
	}
	if watchAllItem(m).Glyph != Checked || !m.watchingAll() || m.watchItem(4).Glyph != Checked {
		t.Errorf("Watch All while every stage is watched: %+v", watchAllItem(m))
	}
	// with every one watched, it stops them all
	m.chose(shell.Choice{Own: WatchAll})
	if slices.ContainsFunc(m.stages, func(s stage) bool { return s.watch }) || watchAllItem(m).Glyph != Unwatched {
		t.Errorf("Watch All again: %+v", watchAllItem(m))
	}
	// chosen from the app's menu by a click
	at := slices.IndexFunc(m.appMenu.Items(), func(it shell.Item) bool { return it.Name == WatchAll })
	if m = drive(m, press('='), testkit.Click(3, shell.Top+1+at)); !m.watchingAll() || m.appMenu.Open {
		t.Errorf("a click on Watch All: watching all %v, menu open %v", m.watchingAll(), m.appMenu.Open)
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
	next, cmd := m.Update(marksMsg{marks: map[string]string{"backlog": "b2", "releases": "r2"}})
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
	cmd = m.watchAll()
	if cmd == nil {
		t.Fatal("Watch All loaded no stale stage")
	}
	if m = drive(m, cmd()); m.find("releases").stale {
		t.Error("releases is still stale after Watch All")
	}
	// a stage loading is not loaded again: watching it alone starts nothing
	m.watchAll()
	l := m.find("worktrees")
	l.stale, l.loading = true, true
	if cmd := m.toggleWatch(slices.IndexFunc(m.stages, func(s stage) bool { return s.name == "worktrees" })); cmd != nil {
		t.Error("watching a stage still loading loaded it again")
	}
}
