// The 📚 repos tab: first, a row per member of the workspace, with its forge, remotes, checkout, branch, worktrees, and
// the branches no checkout has.
package flow

import (
	"path/filepath"
	"slices"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/empowerite/kit/projects/tui/internal/style"
	"github.com/empowerite/kit/projects/tui/internal/table"
	"github.com/empowerite/kit/projects/tui/internal/testkit"
)

// reposOf is the repos stage's document over the group w: a cloned at /code/a, b at /code/b, or neither cloned.
func reposOf(cloned bool) string {
	a, b := `null`, `null`
	if cloned {
		a, b = `"/code/a"`, `"/code/b"`
	}
	return `{"rows": [
		{"repository": "github.com/o/a", "forge": "github", "remotes": [], "checkout": ` + a + `, "by": "fixed"},
		{"repository": "github.com/o/b", "forge": "github", "remotes": [], "checkout": ` + b + `, "by": "fixed"}],
		"errors": []}`
}

func TestReposDrawsAMemberARow(t *testing.T) {
	m := loaded(t, Options{Stages: []string{"repos", "backlog", "desk"}, NoPresets: true, Clock: &testkit.Clock{}},
		tea.WindowSizeMsg{Width: 120, Height: 24})
	if s := m.find("repos"); s.refusal != "" || len(s.repos) != 2 {
		t.Fatalf("repos: %q, %d rows", s.refusal, len(s.repos))
	}
	testkit.Golden(t, filepath.Join("testdata", "repos.golden"), raw(m))
}

func TestOpenPutsReposFirst(t *testing.T) {
	for given, want := range map[string]string{"backlog desk": "repos backlog desk", "repos backlog": "repos backlog"} {
		t.Setenv("FAKE_FLOW_STAGES", given)
		m, err := Open(runner(t), nil)
		if err != nil {
			t.Fatal(err)
		}
		var names []string
		for _, s := range m.stages {
			names = append(names, s.name)
		}
		if got := strings.Join(names, " "); got != want {
			t.Errorf("kit's %q opens %q", given, got)
		}
		m.Close()
	}
}

func TestARepoWhoseDeskFailedSaysSo(t *testing.T) {
	m := drive(over(t, "repos"), answer("repos", `{"rows": [{"repository": "github.com/o/a", "checkout": "/code/a",
		"remotes": [{"name": "origin"}], "branch": "main", "ahead": 2, "behind": 1, "by": "fixed",
		"error": "git cannot read /code/a"}], "errors": []}`))
	r := m.records(*m.find("repos"))[0]
	if got := r.values["checkout"].Raw; !slices.Equal(got, []string{"/code/a"}) {
		t.Errorf("the checkout: %q", got)
	}
	if len(r.Detail) != 1 || !strings.Contains(r.Detail[0][0].Values[0], "git cannot read /code/a") {
		t.Errorf("the detail: %+v", r.Detail)
	}
	if c := r.values["ahead"].Of.(checkout); c.Ahead != 2 || c.Behind != 1 {
		t.Errorf("ahead and behind: %+v", c)
	}
}

func TestIncomingDrawsEachMembersRemotesUnderItsName(t *testing.T) {
	m := drive(over(t, "repos", "incoming"), answer("repos", reposOf(true)), answer("incoming", `{"rows": [
		{"fetched": true, "primary": "origin", "member": "github.com/o/a", "remotes": [{"name": "origin",
		 "tags": ["v1"]}]},
		{"fetched": true, "primary": "origin", "member": "github.com/o/b", "remotes": [{"name": "origin"}]}],
		"errors": []}`))
	m = drive(m, tab)
	if v := view(m); !strings.Contains(v, "a · "+Remote+" origin") || !strings.Contains(v, "b · "+Remote+" origin") {
		t.Errorf("each member's remotes under its name:\n%s", v)
	}
}

func TestAnEmptyReposTabSaysSo(t *testing.T) {
	if got := empty(stage{name: "repos"}); got != "no repository" {
		t.Errorf("an empty repos tab says %q", got)
	}
}

func TestTheReposTabCountsNoOtherTab(t *testing.T) {
	for _, f := range fieldsOf("repos") {
		if f.ID == "backlog" || f.ID == "releases" {
			t.Errorf("the repos tab has a field %s", f.ID)
		}
	}
}

// checkoutCell draws a member's checkout as the repos tab does, with HOME at /home/u.
func checkoutCell(t *testing.T, path string, failed bool) cell {
	t.Helper()
	t.Setenv("HOME", "/home/u")
	m := drive(over(t, "repos"))
	r := repo{Repository: "github.com/o/a", Checkout: &path}
	if failed {
		line := "git cannot read it"
		r.Error = &line
	}
	m.find("repos").repos = []repo{r}
	v := m.records(*m.find("repos"))[0].values["checkout"]
	return fieldIn("repos", "checkout").Draw([]table.Value{v}, "")[0]
}

func TestACheckoutUnderHomeReadsTildeAndClosedItsLastTwoFolders(t *testing.T) {
	c := checkoutCell(t, "/home/u/Code/github.com/empowerite/kit", false)
	if !slices.Equal(c.Values, []string{style.Off("~/Code/github.com/empowerite/kit")}) {
		t.Errorf("open, the checkout reads %q", c.Values)
	}
	if c.Summary != style.Off("…/empowerite/kit") {
		t.Errorf("closed, the checkout reads %q", c.Summary)
	}
	if !slices.Equal(c.Raw, []string{"/home/u/Code/github.com/empowerite/kit"}) {
		t.Errorf("the checkout sorts by %q", c.Raw)
	}
}

func TestACheckoutOfTwoFoldersOrFewerReadsWhole(t *testing.T) {
	for path, want := range map[string]string{"/home/u/kit": "~/kit", "/srv/kit": "/srv/kit", "/home/u": "~"} {
		if c := checkoutCell(t, path, false); c.Summary != "" || !slices.Equal(c.Values, []string{style.Off(want)}) {
			t.Errorf("%s reads %q, closed %q", path, c.Values, c.Summary)
		}
	}
	if c := checkoutCell(t, "/home/user/kit/a", false); !slices.Equal(c.Values, []string{style.Off("/home/user/kit/a")}) {
		t.Errorf("a folder beside home reads %q", c.Values)
	}
}

func TestACheckoutWhoseDeskFailedReadsAFailureOpenAndClosed(t *testing.T) {
	c := checkoutCell(t, "/home/u/Code/github.com/o/a", true)
	if !slices.Equal(c.Values, []string{style.Off(Failed + " ~/Code/github.com/o/a")}) || c.Summary != style.Off(Failed+" …/o/a") {
		t.Errorf("the checkout reads %q, closed %q", c.Values, c.Summary)
	}
}

func TestAClosedCheckoutKeepsTheColumnsNarrow(t *testing.T) {
	t.Setenv("HOME", "/home/u")
	m := drive(over(t, "repos"), answer("repos", `{"rows": [{"repository": "github.com/o/a", "remotes": [],
		"checkout": "/home/u/Code/github.com/deeply/nested/folders/all/the/way/down/o/a", "by": "fixed",
		"branch": "main", "ahead": 0, "behind": 0}], "errors": []}`))
	m = drive(m, tea.WindowSizeMsg{Width: 120, Height: 24})
	if v := view(m); !strings.Contains(v, "…/o/a") || strings.Contains(v, "deeply") {
		t.Errorf("the closed row:\n%s", v)
	}
	m.find("repos").Open["github.com/o/a"] = true
	if v := view(m); !strings.Contains(v, "~/Code/github.com/deeply/nested") {
		t.Errorf("the open row:\n%s", v)
	}
}

func TestAPathWithNoHomeReadsAsItIs(t *testing.T) {
	for _, home := range []string{"", "/"} {
		if got := tilde("/srv/kit", home); got != "/srv/kit" {
			t.Errorf("with home %q, /srv/kit reads %q", home, got)
		}
	}
}

func TestARepoListsItsWorktreesByBranchAndItsBranchesNoCheckoutHas(t *testing.T) {
	m := drive(over(t, "repos"), answer("repos", `{"rows": [{"repository": "github.com/o/a", "remotes": [],
		"checkout": "/code/a", "by": "fixed", "branch": "main", "ahead": 0, "behind": 0,
		"worktrees": [{"branch": null, "path": "/code/a/worktrees/loose"}, {"branch": "topic", "path": "/code/a/worktrees/topic"}],
		"branches": ["idle", "old"]}], "errors": []}`))
	r := m.records(*m.find("repos"))[0]
	if got := r.values["worktrees"].Raw; !slices.Equal(got, []string{"loose", "topic"}) {
		t.Errorf("the worktrees: %q", got)
	}
	if got := r.values["branches"].Raw; !slices.Equal(got, []string{"idle", "old"}) {
		t.Errorf("the branches: %q", got)
	}
	c := fieldIn("repos", "worktrees").Draw([]table.Value{r.values["worktrees"]}, "")[0]
	if !slices.Equal(c.Values, []string{Branch + " loose " + Detached, Branch + " topic"}) || ansi.Strip(c.Summary) != "2" {
		t.Errorf("the worktrees draw %q, closed %q", c.Values, c.Summary)
	}
	c = fieldIn("repos", "branches").Draw([]table.Value{r.values["branches"]}, "")[0]
	if !slices.Equal(c.Values, []string{Sprout + " idle", Sprout + " old"}) {
		t.Errorf("the branches draw %q", c.Values)
	}
	if f := fieldIn("repos", "worktrees"); f.One != "worktree" || fieldIn("repos", "branches").One != "branch" {
		t.Errorf("the lists name one %q and %q", f.One, fieldIn("repos", "branches").One)
	}
}

func TestARepoNotClonedListsNoWorktreeAndNoBranch(t *testing.T) {
	m := drive(over(t, "repos"), answer("repos", reposOf(false)))
	r := m.records(*m.find("repos"))[0]
	if len(r.values["worktrees"].Raw) != 0 || len(r.values["branches"].Raw) != 0 {
		t.Errorf("not cloned, it lists %q and %q", r.values["worktrees"].Raw, r.values["branches"].Raw)
	}
	if got := r.values["checkout"].Else; got != "not cloned" {
		t.Errorf("not cloned, the checkout reads %q", got)
	}
}

func TestARepoBranchIsABranchsGlyph(t *testing.T) {
	m := drive(over(t, "repos"), answer("repos", reposOf(true)))
	v := m.records(*m.find("repos"))[0].values["branch"]
	if v.Raw == nil {
		v = table.Value{Raw: []string{"main"}, Of: checkout{Branch: "main"}}
	}
	if c := fieldIn("repos", "branch").Draw([]table.Value{v}, ""); !strings.HasPrefix(c[0].Values[0], Branch+" ") {
		t.Errorf("the branch draws %q", c[0].Values[0])
	}
}
