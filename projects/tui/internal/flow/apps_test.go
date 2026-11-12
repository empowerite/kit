// kittui's apps: Flow is the picture kit flow draws, every stage of the flow setting in its order; each other app is
// the picture over its own stages, kit flow's and its own verb's together, every tab a stage of kit's; kit flow opens
// Flow, and an app verb's words open the app with the tab named.
package flow

import (
	"encoding/json"
	"maps"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/empowerite/kit/projects/tui/internal/shell"
	"github.com/empowerite/kit/projects/tui/internal/source"
	"github.com/empowerite/kit/projects/tui/internal/style"
	"github.com/empowerite/kit/projects/tui/internal/table"
	"github.com/empowerite/kit/projects/tui/internal/testkit"
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
			if !sourced(name) {
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
		{"flow", "issues --mine", shell.Flow, "issues --mine"},
		// backlog and underway are no stages of kit flow's: Flow opens on issues, as ParseWords reads them
		{"flow", "backlog --mine", shell.Flow, "backlog --mine"},
		{"flow", "underway", shell.Flow, "underway"},
		{"issues", "backlog", shell.Issues, "backlog"},
		{"issues", "underway --mine", shell.Issues, "underway --mine"},
		{"issues", "milestones", shell.Issues, "milestones"},
		{"flow", "changes", shell.Flow, "changes"},
		{"flow", "builds", shell.Flow, "builds"},
		{"flow", "incoming --no-fetch", shell.Flow, "incoming --no-fetch"},
		// desk and worktrees are no stages of kit flow: Repositories' worktrees draws them, the desk its first row
		{"flow", "desk", shell.Repositories, "worktrees"},
		{"flow", "worktrees --no-fetch", shell.Repositories, "worktrees --no-fetch"},
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
	t.Setenv("FAKE_FLOW_STAGES", "releases desk builds issues worktrees")
	m, err := OpenApp(runner(t), shell.Flow, ParseWords([]string{"builds"}))
	if err != nil {
		t.Fatal(err)
	}
	defer m.Close()
	var names []string
	for _, s := range m.stages {
		names = append(names, s.name)
	}
	if got := strings.Join(names, " "); got != "releases builds issues" {
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

// appOf gives an app built around an entity, every tab its own, loaded from fake-kit's fixtures, the tables without
// their preset line, its waits on the test's clock.
func appOf(t *testing.T, app shell.Screen, msgs ...tea.Msg) Model {
	t.Helper()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	m := loaded(t, Options{App: app, Stages: stagesIn(app), Origin: Origin{Host: "github.com", Repo: "o/r"},
		NoPresets: true, Clock: &testkit.Clock{}}, msgs...)
	t.Cleanup(m.Close)
	return m
}

// Every tab of every app reads a stage of kit's: its catalog entry's command, <verb> <stage> --json, staged, so its
// verb's --stages and --marks name it, over the workspace; no tab is planned. Each app opens on the tabs its verbs'
// --stages print, and each tab draws its rows, a count on its tab, and no refusal.
func TestEveryTabReadsAKitStage(t *testing.T) {
	for _, a := range apps {
		for _, name := range a.stages {
			verb, stage := verbOf(name)
			if !sourced(name) || !staged(name) || verb == "" || label(name) != stage || !takes(name, "workspace") {
				t.Errorf("%s's %s reads %q, staged %v", a.screen, name, commandOf(name), staged(name))
			}
		}
	}
	for _, c := range []struct {
		app    shell.Screen
		stages string
	}{
		{shell.Issues, "issues issues/milestones issues/iterations"},
		{shell.Changes, "changes changes/reviews changes/checks changes/merged"},
		{shell.Delivery, "releases builds delivery/artifacts delivery/environments delivery/deployments"},
		{shell.People, "people/people people/teams people/agents"},
		{shell.Trust, "audit/evidence audit/findings audit/controls audit/waivers audit/policy"},
		{shell.Platform, "platform/services platform/toolchains platform/tools platform/settings"},
	} {
		t.Setenv("XDG_CONFIG_HOME", t.TempDir())
		opened, err := OpenApp(runner(t), c.app, Words{})
		if err != nil {
			t.Fatalf("%s: %v", c.app, err)
		}
		opened.Close()
		var names []string
		for _, s := range opened.stages {
			names = append(names, s.name)
		}
		if got := strings.Join(names, " "); got != c.stages {
			t.Errorf("%s's tabs: %q", c.app, got)
		}
		m := appOf(t, c.app, tea.WindowSizeMsg{Width: 160, Height: 30})
		for i, s := range m.stages {
			if s.src.Loading || s.refusal != "" || m.count(i) == "" || m.count(i) == "0" {
				t.Errorf("%s's %s: loading %v, refusal %q, count %q", c.app, s.name, s.src.Loading, s.refusal,
					m.count(i))
			}
		}
		if v := view(m); strings.Contains(v, "no data source") || strings.Contains(v, "does not draw") {
			t.Errorf("%s:\n%s", c.app, v)
		}
	}
}

// A tab whose every error is a feature the forge has off here draws its rows, or its empty line, then, dim, that the
// feature is off here, and no failure: no ❌, and its count is its rows'. A real error over a group is still the
// member's failure.
func TestATabOffHereSaysSo(t *testing.T) {
	m := appOf(t, shell.Trust, tea.WindowSizeMsg{Width: 160, Height: 30})
	m = drive(m, tab)
	if m.stages[m.stage].name != "audit/findings" {
		t.Fatalf("the second tab is %s", m.stages[m.stage].name)
	}
	v := view(m)
	if !strings.Contains(v, "js/xss") || !strings.Contains(v, "secret-scanning is off here") ||
		strings.Contains(v, Failed) || m.count(m.stage) != "2" {
		t.Errorf("findings with secret scanning off, count %q:\n%s", m.count(m.stage), v)
	}
	m = drive(m, reanswer(m, "audit/findings", `{"rows": [], "errors": [{"feature": "secret-scanning", "off": true}]}`))
	got := m.lines()
	if len(got) != 2 || !strings.Contains(got[0].Text, "no alert") ||
		got[1].Text != lead+style.Off("secret-scanning is off here") || strings.Contains(view(m), Failed) ||
		m.count(m.stage) != "0" {
		t.Errorf("findings with nothing but secret scanning off, count %q: %+v", m.count(m.stage), got)
	}
	// over a group, the member it is off for is named, and a member that failed is a failure, as ever
	s := m.find("audit/findings")
	if err := s.decode([]byte(`{"rows": [], "errors": [{"member": "github.com/o/a", "feature": "secret-scanning",
		"off": true}, {"member": "github.com/o/b", "error": "gh cannot read"}]}`), true); err != nil {
		t.Fatal(err)
	}
	var texts []string
	for _, r := range m.records(*s) {
		texts = append(texts, ansi.Strip(r.Cells[0].Values[0]))
	}
	if !slices.Equal(texts, []string{Failed + " github.com/o/b: gh cannot read",
		"secret-scanning is off here in github.com/o/a"}) {
		t.Errorf("over a group: %q", texts)
	}
	// the waivers draw what kit read beside a scanner off here, and beside a part the forge refused, its line a failure
	m = drive(m, tab, tab)
	if v := view(m); m.stages[m.stage].name != "audit/waivers" || !strings.Contains(v, "tolerable_risk") ||
		!strings.Contains(v, "secret-scanning is off here") || strings.Contains(v, Failed) || m.count(m.stage) != "3" {
		t.Errorf("waivers with secret scanning off, count %q:\n%s", m.count(m.stage), v)
	}
	m = drive(m, reanswer(m, "audit/waivers", `{"rows": [{"key": "dependabot 1", "waives": "dependabot 1",
		"actor": null, "kind": "dismissal", "reason": "tolerable_risk", "by": "ada", "until": null, "hidden": null, "links": {}}],
		"errors": [{"feature": "bypasses", "error": "the bypasses could not be read"},
		{"feature": "secret-scanning", "off": true}]}`))
	if v := view(m); !strings.Contains(v, "tolerable_risk") ||
		!strings.Contains(v, Failed+" the bypasses could not be read") ||
		!strings.Contains(v, "secret-scanning is off here") || m.count(m.stage) != "1" {
		t.Errorf("waivers with the bypasses refused, count %q:\n%s", m.count(m.stage), v)
	}
}

// Checks on a forge whose provider lacks check-runs, as one written to a contract before 4, says so, dim, under its
// empty line, as a feature off here is, and no failure; over a group, the member it is lacked for is named.
func TestATabWhoseFeatureTheProviderLacksSaysSo(t *testing.T) {
	m := to(t, appOf(t, shell.Changes, tea.WindowSizeMsg{Width: 160, Height: 30}), "changes/checks")
	m = drive(m, reanswer(m, "changes/checks",
		`{"rows": [], "errors": [{"feature": "check-runs", "off": true, "lacked": true}]}`))
	got := m.lines()
	if len(got) != 2 || !strings.Contains(got[0].Text, "no check") ||
		got[1].Text != lead+style.Off("check-runs is lacked by the provider of the forge here") ||
		strings.Contains(view(m), Failed) || m.count(m.stage) != "0" {
		t.Errorf("checks with check-runs lacked, count %q: %+v", m.count(m.stage), got)
	}
	if got := m.offText(failure{Member: "github.com/o/a", Feature: "check-runs", Off: true, Lacked: true}); got !=
		"check-runs is lacked by the provider of the forge here in github.com/o/a" {
		t.Errorf("over a group: %q", got)
	}
}

// A check's start is drawn by its day and minute here, so two runs of a day are told apart, and by its day alone where
// the pane has not the room; its took stays right-aligned under its name, clear of the started column.
func TestACheckStartsAtItsMinute(t *testing.T) {
	const doc = `{"errors": [], "rows": [
	  {"key": "1 a", "change": 1, "title": "The first release", "name": "a", "verdict": "passed", "at": 1798675200, "took": 1154},
	  {"key": "1 b", "change": 1, "title": "The first release", "name": "b", "verdict": "passed", "at": 1798675200, "took": 0}]}`
	minute := time.Unix(1798675200, 0).Format("2006-01-02 15:04")
	m := to(t, appOf(t, shell.Changes, tea.WindowSizeMsg{Width: 120, Height: 20}), "changes/checks")
	m = drive(m, reanswer(m, "changes/checks", doc))
	v := view(m)
	if !strings.Contains(v, minute+"  19m 14s  The first release") || !strings.Contains(v, minute+"       0s  The first release") {
		t.Errorf("the checks do not start at %s, took right-aligned:\n%s", minute, v)
	}
	m = drive(m, tea.WindowSizeMsg{Width: 56, Height: 20})
	if v := view(m); strings.Contains(v, minute) || !strings.Contains(v, time.Unix(1798675200, 0).Format(time.DateOnly)) {
		t.Errorf("a narrow pane does not fall back to the day:\n%s", v)
	}
}

// A document kit capped, limited at N, makes its tab's count N+: kit read N of more.
func TestACappedTabCountsWithAPlus(t *testing.T) {
	m := appOf(t, shell.Delivery)
	rows := make([]map[string]any, 50)
	for k := range rows {
		rows[k] = map[string]any{"key": k + 1, "id": k + 1, "environment": "production", "ref": "main",
			"state": "success", "actor": "ada", "age": 60 * k, "links": map[string]any{}}
	}
	doc := func(extra map[string]any) string {
		d := map[string]any{"rows": rows, "errors": []any{}}
		maps.Copy(d, extra)
		data, err := json.Marshal(d)
		if err != nil {
			t.Fatal(err)
		}
		return string(data)
	}
	i := slices.IndexFunc(m.stages, func(s stage) bool { return s.name == "delivery/deployments" })
	m = drive(m, reanswer(m, "delivery/deployments", doc(map[string]any{"limited": 50})))
	if got := m.count(i); got != "50+" || !shows(m, "deployments", "50+") {
		t.Errorf("deployments limited at 50 counts %q", got)
	}
	m = drive(m, reanswer(m, "delivery/deployments", doc(nil)))
	if got := m.count(i); got != "50" {
		t.Errorf("deployments not limited counts %q", got)
	}
}

// The stages whose verb never marks them, as their catalog entries' marked false says, are those whose --marks prints
// null always: kit people's three, audit's controls and waivers, and issues' iterations.
func TestTheStagesNeverMarkedAreTheCatalogs(t *testing.T) {
	var never []string
	for name := range catalog {
		if !everMarked(name) {
			never = append(never, name)
		}
	}
	slices.Sort(never)
	if want := []string{"audit/controls", "audit/waivers", "issues/iterations", "people/agents", "people/people",
		"people/teams"}; !slices.Equal(never, want) {
		t.Errorf("never marked: %q", never)
	}
	if !everMarked("issues") || !everMarked("nonesuch") {
		t.Error("a stage the catalog says nothing of is not marked")
	}
}

// A tab its verb never marks is never stale, and its tab's menu has no Watch, which Watch All passes by; a tab whose
// mark kit's last read gave null, as kit flow's when the forge cannot answer, has Watch dim, its key doing nothing,
// until a mark comes; a tab with a mark offers it.
func TestATabWithANullMarkIsNeverStale(t *testing.T) {
	file := filepath.Join(t.TempDir(), "marks.json")
	if err := os.WriteFile(file, []byte(`{"people": null, "teams": null, "agents": null}`), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("FAKE_PEOPLE_MARKS", file)
	m := appOf(t, shell.People, tea.WindowSizeMsg{Width: 160, Height: 30})
	m = drive(m, clockOf(m).Fire()...)
	for range m.stages {
		if s := m.stages[m.stage]; item(m, WatchIt) >= 0 || item(m, StopIt) >= 0 || s.src.Stale {
			t.Errorf("%s offers Watch, or is stale %v", s.name, s.src.Stale)
		}
		if m = drive(m, typed("w")...); m.stages[m.stage].src.Watch {
			t.Errorf("w watches %s", m.stages[m.stage].name)
		}
		m = drive(m, tab)
	}
	if _, on := own(m, WatchAll); on {
		t.Error("Watch All acts on People, whose every tab is never marked")
	}
	m.watchAll(true)
	if slices.ContainsFunc(m.stages, func(s stage) bool { return s.src.Watch }) {
		t.Error("Watch All watches a tab its verb never marks")
	}
	// Issues: issues' mark null now, milestones marked, iterations never
	flowMarks := filepath.Join(t.TempDir(), "flow.json")
	issuesMarks := filepath.Join(t.TempDir(), "issues.json")
	for path, marks := range map[string]string{flowMarks: `{"issues": null}`,
		issuesMarks: `{"milestones": "m1", "iterations": null}`} {
		if err := os.WriteFile(path, []byte(marks), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("FAKE_FLOW_MARKS", flowMarks)
	t.Setenv("FAKE_ISSUES_MARKS", issuesMarks)
	i := appOf(t, shell.Issues, tea.WindowSizeMsg{Width: 160, Height: 30})
	i = drive(i, clockOf(i).Fire()...)
	for _, c := range []struct {
		name          string
		offered, acts bool
	}{{"issues", true, false}, {"issues/milestones", true, true},
		{"issues/iterations", false, false}} {
		i.stage = slices.IndexFunc(i.stages, func(s stage) bool { return s.name == c.name })
		k := item(i, WatchIt)
		if k >= 0 != c.offered || k >= 0 && i.enabled(k) != c.acts {
			t.Errorf("%s: Watch at %d, acts %v", c.name, k, k >= 0 && i.enabled(k))
		}
	}
	i.watchAll(true)
	for name, want := range map[string]bool{"issues": false, "issues/milestones": true,
		"issues/iterations": false} {
		if got := i.find(name).src.Watch; got != want {
			t.Errorf("Watch All watches %s: %v", name, got)
		}
	}
}

// kittui asks no verb for its marks whose every tab its catalog entry says kit never marks: People's --marks is never
// run, as it prints null for every tab; Issues', whose milestones kit marks, is.
func TestAVerbThatNeverMarksItsTabsIsNotAskedForTheirMarks(t *testing.T) {
	if got := verbsOf([]string{"people/people", "people/teams", "people/agents"}); len(got) != 0 {
		t.Errorf("People's marks are read of %q", got)
	}
	record := filepath.Join(t.TempDir(), "calls")
	t.Setenv("FAKE_FLOW_CALLS", record)
	m := appOf(t, shell.People, tea.WindowSizeMsg{Width: 160, Height: 30})
	drive(m, clockOf(m).Fire()...)
	if got := calls(t, record); slices.Contains(got, "people --marks") {
		t.Errorf("People ran %q", got)
	}
	i := appOf(t, shell.Issues, tea.WindowSizeMsg{Width: 160, Height: 30})
	drive(i, clockOf(i).Fire()...)
	if got := calls(t, record); !slices.Contains(got, "issues --marks") {
		t.Errorf("Issues ran %q", got)
	}
}

// A load of a stage that reads the forge, as its catalog entry's forge says, reads the rate limits again; one that
// reads none, as Platform's, reads them only while the screen has none, so every app draws them, those whose stages
// read no forge too, and a load that spends none does not read them again.
func TestAForgeStagesLoadReadsTheLimitsAgain(t *testing.T) {
	for name, forge := range map[string]bool{"issues": true, "people/teams": true, "audit/findings": true,
		"audit/policy": false, "platform/tools": false, "workspace/repos": false, "incoming": false} {
		if catalog[name].Forge != forge {
			t.Errorf("%s reads the forge: %v", name, catalog[name].Forge)
		}
	}
	for app, name := range map[shell.Screen]string{shell.People: "people/teams", shell.Platform: "platform/tools"} {
		for _, got := range []bool{false, true} {
			m := New(runner(t), Options{App: app, Stages: stagesIn(app), Clock: &testkit.Clock{}})
			m.quota.Got = got
			next, _ := m.Update(source.Answer[[]byte]{Name: name, Gen: m.find(name).src.Gen(),
				Value: []byte(`{"rows": [], "errors": []}`)})
			if asked, want := next.(Model).quota.Asking(), catalog[name].Forge || !got; asked != want {
				t.Errorf("a load of %s, the limits read already %v, reads them: %v", name, got, asked)
			}
			m.Close()
		}
	}
}

// Every app draws the rate limits at its keys bar's right end once its stages load, Platform, whose stages read no
// forge, as well as Issues.
func TestEveryAppDrawsTheLimits(t *testing.T) {
	t.Setenv("FAKE_FLOW_QUOTA", quotas)
	for _, a := range apps {
		m := appOf(t, a.screen, tea.WindowSizeMsg{Width: 200, Height: 30})
		if last := lines(m)[len(lines(m))-1]; !strings.Contains(last, GitHub+" 4871/5000") {
			t.Errorf("%s's keys bar: %q", a.screen, last)
		}
	}
}

// In an app, a link to a stage another app draws leads there: an issue's pull request to its row on Changes, and its
// where, for work in a checkout here, to the worktree's row on Repositories. Following it asks kittui for that app at
// that row; a stage of no app's, or one this app should draw and does not, leads nowhere.
func TestALinkToAnotherAppsTabAsksKittuiForThatAppAtTheRow(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	o := options()
	o.App, o.Stages = shell.Issues, []string{"issues"}
	m := loaded(t, o)
	values := map[string]map[string]string{}
	for _, r := range m.records(*m.find("issues")) {
		values[r.Key] = map[string]string{"change": r.values["change"].Link, "where": r.values["where"].Link}
	}
	if got := values["#1 #2"]["change"]; got != jumpTo("changes", "#2") {
		t.Errorf("the change leads to %q", got)
	}
	// a change changes does not hold is !<number> on GitLab and Azure DevOps, #<number> elsewhere
	gitlab, ado, github := m.pullKey("", "gitlab", 5), m.pullKey("", "ado", 5), m.pullKey("", "github", 5)
	if gitlab != "!5" || ado != "!5" || github != "#5" {
		t.Errorf("a change changes does not hold: %q on GitLab, %q on Azure DevOps, %q on GitHub", gitlab, ado, github)
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
	if m.jump("nowhere/x") != nil || m.jump("issues/iterations/x") != nil {
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

// kit flow's words reach each command whose catalog entry takes them, and no other: --no-fetch incoming, the owner's
// flags the forge's issues and pull requests, --fresh each stage that takes it on its first read; a word kittui does
// not take, as --jq, is no tab and no filter.
func TestTheWordsReachEachCommandThatTakesThem(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	ran := func(app shell.Screen, w Words) []string {
		record := filepath.Join(t.TempDir(), "calls")
		t.Setenv("FAKE_FLOW_CALLS", record)
		m, err := OpenApp(runner(t), app, w)
		if err != nil {
			t.Fatal(err)
		}
		defer m.Close()
		if !reflect.DeepEqual(m.Words(), w) {
			t.Errorf("%s keeps the words %+v, not %+v", app, m.Words(), w)
		}
		drive(m, m.Init()())
		return calls(t, record)
	}
	got := ran(shell.Repositories, Words{NoFetch: true, Owner: []string{"--mine"}})
	if !slices.Contains(got, "flow incoming --json --no-fetch") ||
		slices.ContainsFunc(got, func(c string) bool { return strings.HasPrefix(c, "workspace") && strings.Contains(c, "--mine") }) {
		t.Errorf("Repositories ran %q", got)
	}
	if got := ran(shell.Changes, Words{NoFetch: true, Fresh: true, Owner: []string{"--mine"}}); !slices.Contains(got,
		"flow changes --json --mine --fresh") {
		t.Errorf("Changes ran %q", got)
	}
	want := Words{Tab: "issues", Owner: []string{"--mine", "--owner=bob"}}
	if got := ParseWords(strings.Fields("issues --mine --owner=bob --jq")); !reflect.DeepEqual(got, want) {
		t.Errorf("the words: %+v", got)
	}
	// backlog and underway are issues, set as the preset of the word's name
	for word, preset := range map[string]string{"backlog": "Backlog", "underway": "Underway"} {
		w := ParseWords([]string{word, "--mine"})
		if !reflect.DeepEqual(w, Words{Tab: "issues", Preset: preset, Owner: []string{"--mine"}}) ||
			!reflect.DeepEqual(w.Rest(), Words{Owner: []string{"--mine"}}) {
			t.Errorf("%s: %+v, the rest %+v", word, w, w.Rest())
		}
	}
	all := ParseWords(strings.Fields("--all --others --no-fetch --fresh incoming changes"))
	if !reflect.DeepEqual(all, Words{Tab: "incoming", NoFetch: true, Fresh: true, Owner: []string{"--all", "--others"}}) ||
		!reflect.DeepEqual(all.Rest(), Words{NoFetch: true, Fresh: true, Owner: []string{"--all", "--others"}}) {
		t.Errorf("every word: %+v, the rest %+v", all, all.Rest())
	}
}

// A row's link to another stage of its verb's, row:<stage>/<key>, leads to that stage as kittui names it: a
// deployment's environment to its row of delivery/environments, a waiver to its control's row of audit/controls;
// following it lands there, on the stage the link names, not on another whose name it also starts with, as changes is
// of changes/reviews.
func TestARowOfAnAppVerbLeadsToItsVerbsStage(t *testing.T) {
	m := appOf(t, shell.Delivery, tea.WindowSizeMsg{Width: 160, Height: 30})
	links := map[string]string{}
	for _, r := range m.records(*m.find("delivery/deployments")) {
		links[r.Key] = r.values["environment"].Link
	}
	if links["9"] != jumpTo("delivery/environments", "production") {
		t.Errorf("deployment 9's environment leads to %q", links["9"])
	}
	m.jump("delivery/environments/staging")
	if s := m.stages[m.stage]; s.name != "delivery/environments" || m.rows(m.stage)[s.Cursor].Key != "staging" {
		t.Errorf("the jump: on %s, row %d", s.name, s.Cursor)
	}
	trust := appOf(t, shell.Trust)
	for _, r := range trust.records(*trust.find("audit/waivers")) {
		if r.Key == "7 release" && r.values["waives"].Link != jumpTo("audit/controls", "7") {
			t.Errorf("the waiver leads to %q", r.values["waives"].Link)
		}
	}
	if stageNamed("flow", "changes") != "changes" || stageNamed("nonesuch", "x") != "nonesuch/x" {
		t.Errorf("kit flow's changes is %q", stageNamed("flow", "changes"))
	}
	c := appOf(t, shell.Changes)
	if got := c.stages[c.stageOf("changes/reviews/3 bob")].name; got != "changes/reviews" {
		t.Errorf("changes/reviews/3 bob is %s's", got)
	}
	if got := c.stages[c.stageOf("changes/#2")].name; got != "changes" || c.stageOf("nowhere/x") >= 0 {
		t.Errorf("changes/#2 is %s's", got)
	}
	// a load of changes leaves a jump to changes/reviews waiting for its own stage
	c.landing = "changes/reviews/3 bob"
	c.land("changes", true)
	if c.landing == "" {
		t.Error("a load of changes ended a jump to changes/reviews")
	}
}

// An instant kit gives since the epoch is drawn by its day here, and a size in bytes in the unit that keeps it short.
func TestADateIsItsDayAndASizeItsUnit(t *testing.T) {
	day := time.Unix(1798675200, 0).Format(time.DateOnly)
	if got := dates(table.Field{}, table.One("1798675200")); !slices.Equal(got, []string{day}) {
		t.Errorf("the date: %q", got)
	}
	if got, want := moments(table.Field{}, table.One("1798675200")), time.Unix(1798675200, 0).Format("2006-01-02 15:04"); !slices.Equal(got, []string{want}) || len(want) != 16 {
		t.Errorf("the moment: %q, want %q", got, want)
	}
	for raw, want := range map[string]string{"512": "512 B", "2048": "2.0 KiB", "1572864": "1.5 MiB",
		"3221225472": "3.0 GiB", "1099511627776": "1.0 TiB", "2251799813685248": "2048.0 TiB"} {
		if got := sizes(table.Field{}, table.One(raw)); !slices.Equal(got, []string{want}) {
			t.Errorf("%s bytes: %q", raw, got)
		}
	}
}

// Each app verb opens its own app, as kit audit opens Trust, on the tab its words name; a verb that is none opens Flow.
func TestEachAppVerbOpensItsApp(t *testing.T) {
	for _, c := range []struct {
		verb, words string
		app         shell.Screen
	}{
		{"issues", "", shell.Issues}, {"issues", "milestones", shell.Issues}, {"changes", "merged", shell.Changes},
		{"delivery", "", shell.Delivery}, {"people", "teams", shell.People}, {"audit", "", shell.Trust},
		{"platform", "settings", shell.Platform}, {"tutorials", "", shell.Flow},
	} {
		app, _ := AppOf(c.verb, strings.Fields(c.words))
		if app != c.app || AppVerb(c.verb) != (c.verb != "tutorials") {
			t.Errorf("kit %s %s opens %s", c.verb, c.words, app)
		}
	}
}

// A calendar day kit gives as its midnight in UTC, a milestone's due or an iteration's start, or as its last second, an
// iteration's end, is that day wherever the screen is, west of UTC or east of it; an instant, as a change's merge, is
// its day here.
func TestADueDateIsItsOwnDayAnywhere(t *testing.T) {
	local := time.Local
	t.Cleanup(func() { time.Local = local })
	// 2026-12-31 00:00 UTC, and 2027-01-14 23:59:59 UTC
	start, end := "1798675200", strconv.Itoa(1798675200+15*86400-1)
	for _, zone := range []*time.Location{time.FixedZone("west", -8*3600), time.FixedZone("east", 9*3600)} {
		time.Local = zone
		if got := days(table.Field{}, table.Value{Raw: []string{start, end}}); !slices.Equal(got,
			[]string{"2026-12-31", "2027-01-14"}) {
			t.Errorf("%s: the days %q", zone, got)
		}
		here := time.Unix(1798675200, 0).In(zone).Format(time.DateOnly)
		if got := dates(table.Field{}, table.One(start)); got[0] != here {
			t.Errorf("%s: the instant %q", zone, got)
		}
	}
	for stage, ids := range map[string][]string{"issues/milestones": {"due"}, "issues/iterations": {"start", "end"}} {
		for _, f := range fieldsOf(stage) {
			if slices.Contains(ids, f.ID) && !slices.Equal(f.Renderings, []string{"day"}) {
				t.Errorf("%s's %s draws by %q", stage, f.ID, f.Renderings)
			}
		}
	}
}

// An environment's branches, null, are any branch: the column says any, dim; an empty list says nothing.
func TestAnEnvironmentOfAnyBranchSaysAny(t *testing.T) {
	m := appOf(t, shell.Delivery, tea.WindowSizeMsg{Width: 160, Height: 30})
	values := map[string]table.Value{}
	for _, r := range m.records(*m.find("delivery/environments")) {
		values[r.Key] = r.values["branches"]
	}
	if v := values["staging"]; len(v.Raw) != 0 || v.Else != "any" {
		t.Errorf("staging's branches: %+v", v)
	}
	if v := values["production"]; !slices.Equal(v.Raw, []string{"main"}) || v.Else != "" {
		t.Errorf("production's branches: %+v", v)
	}
	m = drive(m, tab, tab, tab)
	if s := m.stages[m.stage]; s.name != "delivery/environments" || !strings.Contains(raw(m), "any") {
		t.Errorf("on %s:\n%s", s.name, view(m))
	}
	if !slices.ContainsFunc(m.lines(), func(l table.Line) bool {
		return strings.Contains(l.Text, style.Off("any"))
	}) {
		t.Error("any is not dim")
	}
}
