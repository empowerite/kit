// kittui's apps: Flow is the picture kit flow draws, every stage of the flow setting in its order; each other app is
// the picture over its own stages, kit flow's and kit workspace's together, and a tab kit has no data source for yet
// says so and loads nothing; kit flow opens Flow, and kit workspace's words open the app with the tab named.
package flow

import (
	"path/filepath"
	"slices"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/empowerite/kit/projects/tui/internal/shell"
)

func TestEveryAppInTheMenuHasTabsAndNoStageIsInTwo(t *testing.T) {
	seen := map[string]shell.Screen{}
	for _, a := range apps {
		if len(a.stages) == 0 {
			t.Errorf("%s has no tabs", a.screen)
		}
		for _, name := range a.stages {
			if other, ok := seen[label(name)]; ok {
				t.Errorf("%s is a tab of %s and of %s", label(name), other, a.screen)
			}
			seen[label(name)] = a.screen
			if !ofPlanned(name) && !sourced(name) {
				t.Errorf("%s draws %s, which kittui does not know", a.screen, name)
			}
		}
	}
	for _, s := range []shell.Screen{shell.Flow, shell.Issues, shell.Changes, shell.Repositories, shell.Delivery,
		shell.People, shell.Trust, shell.Platform, shell.Packages} {
		if !IsApp(s) {
			t.Errorf("%s is not an app", s)
		}
	}
	if IsApp(shell.Welcome) || IsApp(shell.Help) || IsApp(shell.Tutorials) {
		t.Error("a guide is an app")
	}
}

func TestKitFlowOpensFlowAndKitWorkspacesWordsTheAppWithTheTabNamed(t *testing.T) {
	for _, c := range []struct {
		verb, words string
		app         shell.Screen
		taken       string
	}{
		{"flow", "", shell.Flow, ""},
		{"flow", "--mine", shell.Flow, "--mine"},
		{"flow", "underway --mine", shell.Flow, "underway --mine"},
		{"flow", "changes", shell.Flow, "changes"},
		{"flow", "builds", shell.Flow, "builds"},
		{"flow", "incoming --no-fetch", shell.Flow, "incoming --no-fetch"},
		{"workspace", "", shell.Repositories, ""},
		{"workspace", "branches", shell.Repositories, "branches"},
		{"packages", "", shell.Packages, ""},
		{"packages", "packages", shell.Packages, "packages"},
	} {
		app, words := AppOf(c.verb, strings.Fields(c.words))
		if app != c.app || strings.Join(words, " ") != c.taken {
			t.Errorf("kit %s %s: %s, %q", c.verb, c.words, app, words)
		}
	}
}

// Flow draws every stage kit flow --stages prints that kittui draws, in its order, each its own tab, and plans none,
// nor a stage kittui no longer draws, as desk; its title is kit flow's, and a link to a stage it does not draw leads to
// the app built around that stage's entity.
func TestFlowDrawsEveryStageKitFlowPrintsInItsOrder(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("FAKE_FLOW_STAGES", "releases desk builds backlog worktrees")
	m, err := OpenApp(runner(t), shell.Flow, []string{"builds"})
	if err != nil {
		t.Fatal(err)
	}
	defer m.Close()
	var names []string
	for _, s := range m.stages {
		names = append(names, s.name)
	}
	if got := strings.Join(names, " "); got != "releases builds backlog" {
		t.Errorf("Flow's tabs: %q", got)
	}
	if m.stages[m.stage].name != "builds" {
		t.Errorf("kit flow builds opens on %q", m.stages[m.stage].name)
	}
	m = drive(m, tea.WindowSizeMsg{Width: 100, Height: 20})
	if !strings.Contains(lines(m)[0], "kit flow") {
		t.Errorf("Flow's title: %q", lines(m)[0])
	}
	if !m.reaches("workspace/repos") || m.reaches("nowhere") {
		t.Error("Flow's links do not lead to the entity apps' tabs")
	}
	if got := appWith("changes/7"); got != shell.Changes {
		t.Errorf("a link to a change leads to %q", got)
	}
}

// An app whose every tab is planned asks kit for no stages and loads nothing; each tab has no count, and its pane says
// it has no data source yet, dim, with no header over it.
func TestAPlannedTabLoadsNothingAndSaysSo(t *testing.T) {
	record := filepath.Join(t.TempDir(), "calls")
	t.Setenv("FAKE_FLOW_CALLS", record)
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	m, err := OpenApp(runner(t), shell.Trust, nil)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, s := range m.stages {
		names = append(names, label(s.name))
	}
	if got := strings.Join(names, " "); got != "evidence findings controls waivers policy" {
		t.Errorf("Trust's tabs: %q", got)
	}
	if cmds := m.loads(); len(cmds) != 0 {
		t.Errorf("Trust loads %d stages", len(cmds))
	}
	m = drive(m, tea.WindowSizeMsg{Width: 100, Height: 20})
	if got := calls(t, record); slices.ContainsFunc(got, func(c string) bool { return strings.Contains(c, "--stages") }) {
		t.Errorf("Trust asked kit %q", got)
	}
	v := view(m)
	if !strings.Contains(v, "evidence has no data source yet") || strings.Contains(v, "loading") ||
		!strings.Contains(lines(m)[0], "kit trust") {
		t.Errorf("Trust:\n%s", v)
	}
	if m.count(0) != "" || m.stages[0].src.Loading || m.stages[0].refusal != "" {
		t.Errorf("a planned tab: count %q, loading %v, refusal %q", m.count(0), m.stages[0].src.Loading,
			m.stages[0].refusal)
	}
}

// In an app, a link to a stage another app draws leads there: underway's pull request to its row on Changes, and its
// where, for work in a checkout here, to the worktree's row on Repositories. Following it asks kittui for that app at
// that row; a stage of no app's, or one this app should draw and does not, leads nowhere.
func TestALinkToAnotherAppsTabAsksKittuiForThatAppAtTheRow(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	o := options()
	o.App, o.Stages = shell.Issues, []string{"backlog", "underway"}
	m := loaded(t, o)
	values := map[string]map[string]string{}
	for _, r := range m.records(*m.find("underway")) {
		values[r.Key] = map[string]string{"change": r.values["change"].Link, "where": r.values["where"].Link}
	}
	if got := values["#1 #2"]["change"]; got != jumpTo("changes", "#2") {
		t.Errorf("the change leads to %q", got)
	}
	here := jumpTo("workspace/worktrees", "/home/kit/worktrees/1-the-first-release-v0-1-0")
	if got := values["#1 #2"]["where"]; got != here {
		t.Errorf("here leads to %q", got)
	}
	cmd := m.jump("changes/#2")
	if cmd == nil {
		t.Fatal("a jump to Changes asked for nothing")
	}
	if got := cmd(); got != (shell.JumpMsg{Screen: shell.Changes, To: "changes/#2"}) {
		t.Errorf("a jump to Changes asked %+v", got)
	}
	if m.jump("nowhere/x") != nil || m.jump("planned/iterations/x") != nil {
		t.Error("a jump to a stage of no app's, or of Issues' own, asked for an app")
	}
}

// A jump that comes before its stage's rows waits for them, and the row is under the cursor once they load; a jump to
// a row the stage does not have is done once it loads.
func TestAJumpLandsOnTheRowOnceItsStageLoads(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	o := options()
	o.App, o.Stages = shell.Changes, []string{"changes"}
	m := New(runner(t), o)
	m = drive(m, tea.WindowSizeMsg{Width: 120, Height: 24}, shell.JumpMsg{Screen: shell.Changes, To: "changes/#4"})
	if m.landing != "changes/#4" {
		t.Fatalf("the jump waits for %q", m.landing)
	}
	m = drive(m, m.Init()())
	rows := m.rows(0)
	k := slices.IndexFunc(rows, func(r row) bool { return r.Key == "#4" })
	if k <= 0 || m.stages[0].Cursor != k || m.landing != "" {
		t.Errorf("the cursor on %d, #4 at %d, waiting for %q", m.stages[0].Cursor, k, m.landing)
	}
	m = drive(m, shell.JumpMsg{Screen: shell.Changes, To: "changes/#99"})
	if m.landing != "" || m.stages[0].Cursor != k {
		t.Errorf("a jump to no row: the cursor on %d, waiting for %q", m.stages[0].Cursor, m.landing)
	}
}
