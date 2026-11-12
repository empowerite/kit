// The Packages app: kit packages opens it on its one tab, a row per package in reach, drawn from the catalog, and its
// tab's menu adds, removes, and selects the packages selected, each by running kit packages, asking first where files
// come out.
package flow

import (
	"path/filepath"
	"slices"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/empowerite/kit/projects/tui/internal/kit"
	"github.com/empowerite/kit/projects/tui/internal/shell"
	"github.com/empowerite/kit/projects/tui/internal/source"
)

// packaged gives the Packages app loaded, as kit packages opens it, recording kit's calls, with the packages named
// selected.
func packaged(t *testing.T, names ...string) (Model, string) {
	t.Helper()
	record := filepath.Join(t.TempDir(), "calls")
	t.Setenv("FAKE_FLOW_CALLS", record)
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	app, words := AppOf(packagesVerb, nil)
	m, err := OpenApp(runner(t), app, ParseWords(words))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(m.Close)
	m.origin, m.noPresets = Origin{Host: "github.com", Repo: "empowerite/kit"}, true
	first := []tea.Msg{tea.WindowSizeMsg{Width: 140, Height: 30}}
	if cmd := m.Init(); cmd != nil {
		first = append(first, cmd())
	}
	m = drive(m, first...)
	for _, name := range names {
		m.stages[m.stage].Picked[name] = true
	}
	return m, record
}

// actedOn gives the calls of kit packages that change the active set, as fake-kit recorded them.
func actedOn(t *testing.T, record string) []string {
	t.Helper()
	return slices.DeleteFunc(calls(t, record), func(c string) bool {
		return !strings.HasPrefix(c, "packages add") && !strings.HasPrefix(c, "packages remove") &&
			!strings.HasPrefix(c, "packages select")
	})
}

func TestKitPackagesOpensPackagesOnItsOneTabWhichDrawsEveryPackageInReach(t *testing.T) {
	for _, words := range [][]string{nil, {"packages"}} {
		if app, _ := AppOf(packagesVerb, words); app != shell.Packages {
			t.Errorf("kit packages %q opens %s", words, app)
		}
	}
	if app, _ := AppOf(workspaceVerb, nil); app != shell.Repositories {
		t.Errorf("kit workspace opens %s", app)
	}
	if !IsApp(shell.Packages) || command(packagesStage) != "kit packages packages --json" {
		t.Errorf("Packages is an app: %v; its tab reads %q", IsApp(shell.Packages), command(packagesStage))
	}
	m, record := packaged(t)
	if len(m.stages) != 1 || m.stages[0].name != packagesStage {
		t.Fatalf("Packages' tabs: %+v", m.stages)
	}
	if got := len(m.records(m.stages[0])); got != 8 {
		t.Errorf("Packages draws %d rows", got)
	}
	v := view(m)
	for _, want := range []string{"agents/claude", "🟢 active", "⛔ excluded", "repository", "This repository's own"} {
		if !strings.Contains(v, want) {
			t.Errorf("Packages does not show %q:\n%s", want, v)
		}
	}
	if got := calls(t, record); !slices.Contains(got, "packages packages --json") ||
		slices.ContainsFunc(got, func(c string) bool { return strings.Contains(c, "--cached") }) {
		t.Errorf("Packages' calls: %q", got)
	}
}

func TestPackagesActionsAreInItsMenuDimUntilTheyApply(t *testing.T) {
	m, _ := packaged(t)
	names := []string{AddPackages, RemovePackages, OnlyPackages}
	for _, name := range names {
		if k := item(m, name); k < 0 || m.enabled(k) {
			t.Errorf("%s with nothing selected: line %d, not dim", name, k)
		}
	}
	if item(m, "Assign to Me") >= 0 {
		t.Error("Packages offers the issues' actions")
	}
	m, _ = packaged(t, "agents/cursor")
	if !m.enabled(item(m, AddPackages)) || m.enabled(item(m, RemovePackages)) || !m.enabled(item(m, OnlyPackages)) {
		t.Error("an inactive package selected: Add and Make Only These Active apply, Remove does not")
	}
	m, _ = packaged(t, "kit")
	if m.enabled(item(m, AddPackages)) || !m.enabled(item(m, RemovePackages)) {
		t.Error("an active package selected: Remove applies, Add does not")
	}
}

func TestAddRunsKitPackagesAddOnTheInactivePackagesSelectedAndReadsTheStageAgain(t *testing.T) {
	m, record := packaged(t, "agents/cursor", "style/yaml", "kit")
	m = choose(t, m, AddPackages)
	if got := actedOn(t, record); !slices.Equal(got, []string{"packages add agents/cursor style/yaml"}) {
		t.Fatalf("the calls: %q", got)
	}
	last := lines(m)[len(lines(m))-1]
	if !strings.Contains(last, "✅ kit packages add agents/cursor style/yaml") || m.selected() != 0 {
		t.Errorf("after add: %q, %d selected", last, m.selected())
	}
	n := 0
	for _, c := range calls(t, record) {
		if c == "packages packages --json" {
			n++
		}
	}
	if n != 2 {
		t.Errorf("the stage was read %d times", n)
	}
}

func TestRemoveListsTheActivePackagesSelectedAndRunsOnlyOnYes(t *testing.T) {
	m, record := packaged(t, "style/bash", "agents/cursor")
	m = choose(t, m, RemovePackages)
	v := view(m)
	if !strings.Contains(v, "Remove these packages, and the files they delivered?") || !strings.Contains(v,
		Minus+" style/bash") || strings.Contains(v, Minus+" agents/cursor") {
		t.Fatalf("the dialog:\n%s", v)
	}
	m = drive(m, esc)
	if got := actedOn(t, record); len(got) != 0 || m.ask != nil {
		t.Fatalf("esc ran %q", got)
	}
	m = choose(t, m, RemovePackages)
	drive(m, press(tea.KeyEnter))
	if got := actedOn(t, record); !slices.Equal(got, []string{"packages remove style/bash"}) {
		t.Errorf("the calls: %q", got)
	}
}

func TestMakeOnlyTheseActiveListsWhatItKeepsAddsAndTakesOutThenRunsSelect(t *testing.T) {
	m, record := packaged(t, "kit", "agents/cursor")
	m = choose(t, m, OnlyPackages)
	v := view(m)
	for _, want := range []string{Checked + " kit", Plus + " agents/cursor", Minus + " house", Minus + " style/bash"} {
		if !strings.Contains(v, want) {
			t.Errorf("the dialog does not say %q:\n%s", want, v)
		}
	}
	if strings.Contains(v, Minus+" kit") || strings.Contains(v, Minus+" style/yaml") {
		t.Errorf("the dialog takes out what is kept, or what is not active:\n%s", v)
	}
	drive(m, press(tea.KeyEnter))
	if got := actedOn(t, record); !slices.Equal(got, []string{"packages select kit agents/cursor"}) {
		t.Errorf("the calls: %q", got)
	}
}

func TestAPackagesRefusalIsTheKeysBarsLineAndKeepsTheSelection(t *testing.T) {
	t.Setenv("FAKE_FLOW_REFUSE", "packages add")
	m, _ := packaged(t, "agents/cursor")
	m = choose(t, m, AddPackages)
	last := lines(m)[len(lines(m))-1]
	if !strings.Contains(last, "kit packages add refuses for the test") || m.selected() != 1 {
		t.Errorf("after a refusal: %q, %d selected", last, m.selected())
	}
}

func TestOverAGroupAPackagesActionRunsInEachMembersCheckout(t *testing.T) {
	m, _ := packaged(t)
	m.runner.Group = "w"
	m.stages[0].rows = nil
	m = drive(m, answer(packagesStage, `{"rows": [
		{"key": "kit", "member": "github.com/o/a", "state": "active", "links": {}},
		{"key": "style/bash", "member": "github.com/o/b", "state": "inactive", "links": {}}], "errors": []}`))
	m.stages[0].Picked["github.com/o/a kit"], m.stages[0].Picked["github.com/o/b style/bash"] = true, true
	if k := item(m, OnlyPackages); m.enabled(k) {
		t.Error("Make Only These Active applies to packages of two members")
	}
	// kit has not said where the members are checked out, so nothing runs
	m = choose(t, m, AddPackages)
	if last := lines(m)[len(lines(m))-1]; !strings.Contains(last, "kit has not said where github.com/o/a is") {
		t.Errorf("a member checked out nowhere kit said: %q", last)
	}
	dirs := filepath.Join(t.TempDir(), "dirs")
	t.Setenv("FAKE_FLOW_DIRS", dirs)
	a, _ := filepath.EvalSymlinks(t.TempDir())
	b, _ := filepath.EvalSymlinks(t.TempDir())
	m.ws.Members = []kit.WorkspaceMember{{Repository: "github.com/o/a", Checkout: a},
		{Repository: "github.com/o/b", Checkout: b}}
	m = choose(t, m, AddPackages)
	// the add cleared the selection, and the stage read the rows again, as the fake prints them
	m = drive(m, source.Answer[[]byte]{Name: packagesStage, Gen: m.find(packagesStage).src.Gen(), Value: []byte(
		`{"rows": [{"key": "kit", "member": "github.com/o/a", "state": "active", "links": {}}], "errors": []}`)})
	m.stages[0].Picked["github.com/o/a kit"] = true
	m = choose(t, m, RemovePackages)
	drive(m, press(tea.KeyEnter))
	got := slices.DeleteFunc(ranIn(t, dirs), func(c string) bool {
		return strings.HasSuffix(c, "\tpackages packages --json")
	})
	if !slices.Equal(got, []string{b + "\tpackages add style/bash",
		a + "\tpackages remove kit"}) {
		t.Errorf("the calls, in their checkouts: %q", got)
	}
}

// A tab's dialogs are titled by the tab alone, not by its app and the tab: packages fields, not packages/packages
// fields, and the legend and presets likewise.
func TestADialogIsTitledByTheTabAlone(t *testing.T) {
	m, _ := packaged(t)
	if m.Name() != "packages" {
		t.Errorf("the table's name: %q", m.Name())
	}
	box, _, _ := m.cols.Box(&m)
	if title := ansi.Strip(box[0]); !strings.Contains(title, "─ packages fields ─") {
		t.Errorf("the fields dialog's title: %q", title)
	}
	m.legend.open = true
	box, _, _ = m.legendBox()
	if title := ansi.Strip(box[0]); !strings.Contains(title, "─ packages legend ") {
		t.Errorf("the legend's title: %q", title)
	}
}
