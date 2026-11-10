// kittui's apps: each is the picture over its own stages, kit flow's and kit workspace's together, and a tab kit has no
// data source for yet says so and loads nothing; kit flow's and kit workspace's words open the app with the tab named.
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
			if !ofPlanned(name) && !known[name] {
				t.Errorf("%s draws %s, which kittui does not know", a.screen, name)
			}
		}
	}
	for _, s := range []shell.Screen{shell.Issues, shell.Changes, shell.Repositories, shell.Delivery, shell.People,
		shell.Trust, shell.Platform} {
		if !IsApp(s) {
			t.Errorf("%s is not an app", s)
		}
	}
	if IsApp(shell.Welcome) || IsApp(shell.Help) || IsApp(shell.Tutorials) {
		t.Error("a guide is an app")
	}
}

func TestKitFlowsAndKitWorkspacesWordsOpenTheAppWithTheTabNamed(t *testing.T) {
	for _, c := range []struct {
		verb, words string
		app         shell.Screen
		taken       string
	}{
		{"flow", "", shell.Issues, ""},
		{"flow", "--mine", shell.Issues, "--mine"},
		{"flow", "underway --mine", shell.Issues, "underway --mine"},
		{"flow", "changes", shell.Changes, "changes"},
		{"flow", "builds", shell.Delivery, "builds"},
		{"flow", "incoming --no-fetch", shell.Repositories, "incoming --no-fetch"},
		{"flow", "desk", shell.Repositories, "worktrees"},
		{"flow", "worktrees", shell.Repositories, "worktrees"},
		{"workspace", "", shell.Repositories, ""},
		{"workspace", "branches", shell.Repositories, "branches"},
	} {
		app, words := AppOf(c.verb, strings.Fields(c.words))
		if app != c.app || strings.Join(words, " ") != c.taken {
			t.Errorf("kit %s %s: %s, %q", c.verb, c.words, app, words)
		}
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
