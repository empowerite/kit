// kit workspace's picture: the screen opens on kit workspace's stages, reads each from kit workspace <stage> --json,
// and draws its rows from the catalog alone, its links leading to its own rows and to the forge's pages.
package flow

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

	tea "charm.land/bubbletea/v2"

	"github.com/empowerite/kit/projects/tui/internal/shell"
	"github.com/empowerite/kit/projects/tui/internal/table"
	"github.com/empowerite/kit/projects/tui/internal/testkit"
)

// workspaceStages are kit workspace's stages, as kittui knows them.
var workspaceStages = []string{"workspace/repos", "workspace/remotes", "workspace/branches", "workspace/worktrees", "workspace/commits",
	"workspace/tags"}

// workspaceOptions gives kit workspace's stages, as a test knows them, the tables without their preset line.
func workspaceOptions() Options {
	o := options()
	o.App, o.Stages = shell.Repositories, workspaceStages
	return o
}

// stageOf gives the stage of the model by the name kittui knows it by.
func stageOf(m Model, name string) stage {
	return m.stages[slices.IndexFunc(m.stages, func(s stage) bool { return s.name == name })]
}

// valueOf gives the text a stage's row draws in a field, without its styles, by the row's key and the field's id.
func valueOf(t *testing.T, m Model, name, key, id string) string {
	t.Helper()
	s := stageOf(m, name)
	fields := fieldsOf(name)
	j := slices.IndexFunc(fields, func(f table.Field) bool { return f.ID == id })
	for _, r := range laid(name, m.records(s), func(string) string { return "" }) {
		if r.Key == key {
			return ansi.Strip(strings.Join(r.Cells[j].Values, " "))
		}
	}
	t.Fatalf("%s has no row %q", name, key)
	return ""
}

// Repositories reads kit workspace's stages and kit flow's, names kit workspace's after workspace/, keeps them in its
// own order, incoming last, and opens on the one the words name, maximized; the title, the tabs, and the app's menu
// say the app's names, not the workspace/ kittui knows.
func TestRepositoriesReadsKitWorkspacesStagesAndIncoming(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	record := filepath.Join(t.TempDir(), "calls")
	t.Setenv("FAKE_FLOW_CALLS", record)
	m, err := OpenApp(runner(t), shell.Repositories, Words{})
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, s := range m.stages {
		names = append(names, s.name)
	}
	want := []string{"workspace/repos", "workspace/worktrees", "workspace/branches", "workspace/commits",
		"workspace/tags", "workspace/remotes", "incoming"}
	if !slices.Equal(names, want) || m.appMenu.Here != shell.Repositories {
		t.Errorf("the stages %q, the menu on %q", names, m.appMenu.Here)
	}
	if got := calls(t, record); strings.Join(got, "\n") != "workspace --stages\nflow --stages" {
		t.Errorf("OpenApp ran %q", got)
	}
	m = drive(m, tea.WindowSizeMsg{Width: 120, Height: 24})
	if top := lines(m)[0]; !strings.Contains(top, "kit repositories") {
		t.Errorf("the title: %q", top)
	}
	for _, s := range m.bar().Tabs {
		if strings.Contains(s.Name, "/") {
			t.Errorf("a tab named %q", s.Name)
		}
	}
	if m, err = OpenApp(runner(t), shell.Repositories, ParseWords([]string{"branches"})); err != nil {
		t.Fatal(err)
	}
	if m.stages[m.stage].name != "workspace/branches" || !m.maximized {
		t.Errorf("OpenApp branches: on %q, maximized %v", m.stages[m.stage].name, m.maximized)
	}
}

// Each stage loads from kit workspace <stage> --json, asking nothing kept and no rate limits; its rows are kit's, each
// a value per field: a branch's ahead and behind, a worktree's path, a commit's issue, and a tag's kind.
func TestTheWorkspacesStagesDrawKitsRows(t *testing.T) {
	record := filepath.Join(t.TempDir(), "calls")
	t.Setenv("FAKE_FLOW_CALLS", record)
	m := loaded(t, workspaceOptions())
	got := calls(t, record)
	for _, want := range []string{"workspace remotes --json", "workspace branches --json", "workspace worktrees --json",
		"workspace commits --json", "workspace tags --json"} {
		if !slices.Contains(got, want) {
			t.Errorf("no %q in %q", want, got)
		}
	}
	for _, call := range got {
		if strings.Contains(call, "--cached") || strings.Contains(call, "--quota") || strings.HasPrefix(call, "flow") {
			t.Errorf("the workspace ran %q", call)
		}
	}
	for _, c := range []struct{ stage, key, id, want string }{
		{"workspace/remotes", "origin", "remote", Remote + " origin"},
		{"workspace/remotes", "origin", "default", Default + " main"},
		{"workspace/branches", "origin/7-fix", "upstream", "↑1"},
		{"workspace/branches", "origin/main", "upstream", "="},
		{"workspace/branches", "origin/main", "default", Default},
		{"workspace/branches", "origin/old", "kind", Gone},
		{"workspace/branches", "scratch", "kind", Sprout},
		{"workspace/worktrees", "/src/r/worktrees/7-fix", "base", "↑2"},
		{"workspace/worktrees", "/src/r/worktrees/7-fix", "kind", Worktree},
		{"workspace/worktrees", "/src/r", "kind", Desk},
		{"workspace/tags", "v1", "kind", "annotated"},
	} {
		if got := valueOf(t, m, c.stage, c.key, c.id); got != c.want {
			t.Errorf("%s %s %s: %q, want %q", c.stage, c.key, c.id, got, c.want)
		}
	}
	commits := stageOf(m, "workspace/commits")
	issues := 0
	for _, r := range m.records(commits) {
		if v := r.values["issue"]; len(v.Raw) > 0 {
			issues++
			if v.Raw[0] != "#7" || v.Link != "https://github.com/o/r/issues/7" {
				t.Errorf("the issue: %+v", v)
			}
		}
	}
	if issues != 1 {
		t.Errorf("%d commits name an issue", issues)
	}
	if !strings.Contains(view(m), "remotes") || m.count(1) != "1" {
		t.Errorf("the remotes tab: count %q\n%s", m.count(1), view(m))
	}
}

// A value that leads to another of the workspace's stages, as a branch's worktree does, jumps there, onto the row; over
// a group, the row's key is after the member, as every row's is.
func TestAWorkspaceLinkLeadsToItsRow(t *testing.T) {
	m := loaded(t, workspaceOptions())
	branches := stageOf(m, "workspace/branches")
	var link string
	for _, r := range m.records(branches) {
		if r.Key == "origin/7-fix" {
			link = r.values["worktree"].Link
		}
	}
	if link != jumpTo("workspace/worktrees", "/src/r/worktrees/7-fix") {
		t.Fatalf("the worktree's link: %q", link)
	}
	m.jump(strings.TrimPrefix(link, jumps))
	if m.stages[m.stage].name != "workspace/worktrees" || m.rows(m.stage)[m.stages[m.stage].Cursor].Key != "/src/r/worktrees/7-fix" {
		t.Errorf("the jump: on %q, row %d", m.stages[m.stage].name, m.stages[m.stage].Cursor)
	}
	if got := m.rowLink(workspaceVerb, "row:worktrees//src/r", "github.com/o/r"); got != jumpTo("workspace/worktrees",
		"github.com/o/r /src/r") {
		t.Errorf("a link over a group: %q", got)
	}
	if got := m.rowLink(workspaceVerb, "https://github.com/o/r", "github.com/o/r"); got != "https://github.com/o/r" {
		t.Errorf("a page: %q", got)
	}
}

// The workspace reads kit workspace --marks, by the names kittui knows its stages by: a stage whose mark moves is
// stale, a dot on its tab.
func TestAWorkspaceStageWhoseMarkMovesIsStale(t *testing.T) {
	marks := filepath.Join(t.TempDir(), "marks")
	write := func(tags string) {
		if err := os.WriteFile(marks, []byte(`{"remotes":"r","branches":"b","worktrees":"w","commits":"c","tags":"`+tags+`"}`), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("one")
	t.Setenv("FAKE_WORKSPACE_MARKS", marks)
	m := loaded(t, workspaceOptions())
	m = drive(m, m.marks.Ask(m.env())())
	write("two")
	m = drive(m, m.marks.Ask(m.env())())
	if !stageOf(m, "workspace/tags").src.Stale || stageOf(m, "workspace/branches").src.Stale {
		t.Errorf("tags stale %v, branches stale %v", stageOf(m, "workspace/tags").src.Stale,
			stageOf(m, "workspace/branches").src.Stale)
	}
}

// A stage kit workspace refuses shows kit's line, naming kit workspace's command, not kit flow's.
func TestARefusedWorkspaceStageNamesItsCommand(t *testing.T) {
	t.Setenv("FAKE_FLOW_REFUSE", "workspace/tags")
	m := loaded(t, workspaceOptions())
	if s := stageOf(m, "workspace/tags"); !strings.Contains(s.refusal, "kit workspace tags refuses") {
		t.Errorf("the refusal: %q", s.refusal)
	}
	if got := refusalOf("workspace/tags", &testError{}); got != "kit workspace tags --json was killed" {
		t.Errorf("a stage killed: %q", got)
	}
}

// testError is an error that says nothing, as a kit killed leaves.
type testError struct{}

func (testError) Error() string { return "" }

// The picture of the workspace's branches, its tabs, and its rows, as the screen draws them.
func TestGoldenWorkspace(t *testing.T) {
	m := loaded(t, workspaceOptions(), tea.WindowSizeMsg{Width: 120, Height: 16}, testkit.Press("tab"), testkit.Press("tab"))
	testkit.Golden(t, "testdata/workspace-branches.golden", view(m))
}
