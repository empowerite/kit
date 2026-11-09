// The 📚 repos tab: first, a row per member of the workspace, with its forge, remotes, checkout, and branch, and a
// count of each other tab's rows of the member, which opens that tab filtered to the member.
package flow

import (
	"path/filepath"
	"slices"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/empowerite/kit/projects/tui/internal/kit"
	"github.com/empowerite/kit/projects/tui/internal/source"
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

// repoCount gives the repos row's count of a stage, by the row's member.
func repoCount(m Model, member, stage string) (shown, link string) {
	for _, r := range m.records(*m.find("repos")) {
		if r.Key == member {
			v := r.values[stage]
			if len(v.Raw) == 0 {
				return v.Else, v.Link
			}
			return v.Raw[0], v.Link
		}
	}
	return "no row", ""
}

// follow follows a link to another stage, as a click on it does.
func follow(m Model, link string) Model {
	m.jump(strings.TrimPrefix(link, jumps))
	return m
}

func TestReposDrawsAMemberARow(t *testing.T) {
	m := loaded(t, Options{Stages: []string{"repos", "backlog", "desk"}, NoPresets: true, Clock: &testkit.Clock{}},
		tea.WindowSizeMsg{Width: 120, Height: 24})
	if s := m.find("repos"); s.refusal != "" || len(s.repos) != 2 {
		t.Fatalf("repos: %q, %d rows", s.refusal, len(s.repos))
	}
	testkit.Golden(t, filepath.Join("testdata", "repos.golden"), raw(m))
}

func TestACountOpensItsTabFilteredToTheMember(t *testing.T) {
	m := drive(over(t, "repos", "backlog"), answer("repos", reposOf(true)), answer("backlog", `{"rows": [
		{"number": 1, "title": "a1", "member": "github.com/o/a"},
		{"number": 2, "title": "a2", "member": "github.com/o/a"},
		{"number": 3, "title": "b3", "member": "github.com/o/b"}], "errors": []}`))
	if n, _ := repoCount(m, "github.com/o/a", "backlog"); n != "2" {
		t.Errorf("a's backlog counts %q", n)
	}
	n, link := repoCount(m, "github.com/o/b", "backlog")
	if n != "1" || link != onlyTo("backlog", "github.com/o/b") {
		t.Fatalf("b's backlog counts %q, linking to %q", n, link)
	}
	m = follow(m, link)
	if m.stages[m.stage].name != "backlog" {
		t.Fatalf("the count selected %s", m.stages[m.stage].name)
	}
	if f := m.panel(m.stage).Field("member").Filter; !slices.Equal(f.Values, []string{"github.com/o/b"}) || f.Exclude {
		t.Errorf("backlog's member filter: %+v", f)
	}
	if v := view(m); !strings.Contains(v, "b3") || strings.Contains(v, "a1") {
		t.Errorf("backlog filtered to b:\n%s", v)
	}
}

func TestAMemberNotClonedCountsADimDashNeverZero(t *testing.T) {
	m := drive(over(t, "repos", "backlog", "desk"), answer("repos", reposOf(false)),
		answer("backlog", `{"rows": [], "errors": []}`), answer("desk", `{"rows": [], "errors": []}`))
	for _, member := range []string{"github.com/o/a", "github.com/o/b"} {
		for _, stage := range []string{"backlog", "desk"} {
			if n, link := repoCount(m, member, stage); n != "-" || link != "" {
				t.Errorf("%s's %s counts %q, linking to %q", member, stage, n, link)
			}
		}
	}
	for _, s := range m.stages {
		if s.refusal != "" {
			t.Errorf("%s refused: %s", s.name, s.refusal)
		}
	}
	m = drive(m, right)
	if v := view(m); !strings.Contains(v, "no open issue without a pull request") {
		t.Errorf("backlog's empty line:\n%s", v)
	}
}

func TestAMemberThatFailedCountsAFailure(t *testing.T) {
	m := drive(over(t, "repos", "changes"), answer("repos", reposOf(true)),
		answer("changes", `{"rows": [{"number": 3, "title": "a's change", "member": "github.com/o/a"}],
			"errors": [{"member": "github.com/o/b", "error": "gh cannot read o/b"}]}`))
	if n, _ := repoCount(m, "github.com/o/a", "changes"); n != "1" {
		t.Errorf("a's changes count %q", n)
	}
	n, link := repoCount(m, "github.com/o/b", "changes")
	if n != Failed || link != onlyTo("changes", "github.com/o/b") {
		t.Fatalf("b's changes count %q, linking to %q", n, link)
	}
	m = follow(m, link)
	if v := view(m); !strings.Contains(v, Failed+" b: gh cannot read o/b") || strings.Contains(v, "a's change") {
		t.Errorf("changes filtered to b:\n%s", v)
	}
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

func TestOverOneRepositoryACountOpensItsTab(t *testing.T) {
	m := loaded(t, Options{Stages: []string{"repos", "desk"}, NoPresets: true, Clock: &testkit.Clock{}})
	n, link := repoCount(m, "github.com/empowerite/kit", "desk")
	if n != "1" || link != onlyTo("desk", "") {
		t.Fatalf("the desk counts %q, linking to %q", n, link)
	}
	m = follow(m, link)
	if m.stages[m.stage].name != "desk" || len(m.panel(m.stage).Field("member").Filter.Values) != 0 {
		t.Errorf("the count selected %s, filtered %+v", m.stages[m.stage].name, m.panel(m.stage).Field("member").Filter)
	}
}

func TestACountSaysWhileItsStageLoadsOrRefusesOrIsNotInThePicture(t *testing.T) {
	m := drive(over(t, "repos", "backlog", "changes"), answer("repos", reposOf(true)))
	if n, link := repoCount(m, "github.com/o/a", "backlog"); n != "…" || link != "" {
		t.Errorf("while backlog loads: %q, %q", n, link)
	}
	if n, _ := repoCount(m, "github.com/o/a", "builds"); n != "" {
		t.Errorf("builds, not in the picture: %q", n)
	}
	m = drive(m, source.Answer[[]byte]{Name: "changes", Err: &kit.ExitError{Code: 2, Stderr: "❌ no workspace w"}})
	n, link := repoCount(m, "github.com/o/b", "changes")
	if n != Failed || link != onlyTo("changes", "github.com/o/b") {
		t.Errorf("changes refused: %q, %q", n, link)
	}
}

func TestIncomingCountsWhatTheFetchBroughtAMember(t *testing.T) {
	m := drive(over(t, "repos", "incoming"), answer("repos", reposOf(true)), answer("incoming", `{"rows": [
		{"fetched": true, "primary": "origin", "member": "github.com/o/a", "remotes": [{"name": "origin",
		 "commits": [{"sha": "c1", "subject": "one"}], "branches": {"new": ["n"], "updated": [], "gone": ["g"]},
		 "tags": ["v1"]}]},
		{"fetched": true, "primary": "origin", "member": "github.com/o/b", "remotes": [{"name": "origin"}]}],
		"errors": []}`))
	if n, _ := repoCount(m, "github.com/o/a", "incoming"); n != "4" {
		t.Errorf("a's incoming counts %q", n)
	}
	if n, _ := repoCount(m, "github.com/o/b", "incoming"); n != "0" {
		t.Errorf("b's incoming counts %q", n)
	}
	m = drive(m, right)
	if v := view(m); !strings.Contains(v, "a · "+Remote+" origin") || !strings.Contains(v, "b · "+Remote+" origin") {
		t.Errorf("each member's remotes under its name:\n%s", v)
	}
}

func TestARepoWhoseDeskFailedSaysSo(t *testing.T) {
	m := drive(over(t, "repos"), answer("repos", `{"rows": [{"repository": "github.com/o/a", "checkout": "/code/a",
		"remotes": [{"name": "origin"}], "branch": "main", "ahead": 2, "behind": 1, "by": "fixed",
		"error": "git cannot read /code/a"}], "errors": []}`))
	r := m.records(*m.find("repos"))[0]
	if got := r.values["checkout"].Raw; !slices.Equal(got, []string{Failed + " /code/a"}) {
		t.Errorf("the checkout: %q", got)
	}
	if len(r.Detail) != 1 || !strings.Contains(r.Detail[0][0].Values[0], "git cannot read /code/a") {
		t.Errorf("the detail: %+v", r.Detail)
	}
	if c := r.values["ahead"].Of.(checkout); c.Ahead != 2 || c.Behind != 1 {
		t.Errorf("ahead and behind: %+v", c)
	}
}

func TestACountsLinkToAStageNotInThePictureStaysPut(t *testing.T) {
	m := drive(over(t, "repos", "backlog"), answer("repos", reposOf(true)))
	if m = follow(m, onlyTo("builds", "github.com/o/a")); m.stages[m.stage].name != "repos" {
		t.Errorf("a link to builds, not in the picture, selected %s", m.stages[m.stage].name)
	}
	if got := empty(*m.find("repos")); got != "no repository" {
		t.Errorf("an empty repos tab says %q", got)
	}
}

func TestAnIncomingCountOpensIncomingAtTheMembersRemotesUnfiltered(t *testing.T) {
	m := drive(over(t, "repos", "incoming"), answer("repos", reposOf(true)), answer("incoming", `{"rows": [
		{"fetched": true, "primary": "origin", "member": "github.com/o/a", "remotes": [{"name": "origin",
		 "tags": ["v1"]}]},
		{"fetched": true, "primary": "origin", "member": "github.com/o/b", "remotes": [{"name": "origin",
		 "tags": ["v2"]}]}], "errors": []}`))
	_, link := repoCount(m, "github.com/o/b", "incoming")
	m = follow(m, link)
	if m.stages[m.stage].name != "incoming" || len(m.panel(m.stage).Field("member").Filter.Values) != 0 {
		t.Fatalf("on %s, filtered %+v", m.stages[m.stage].name, m.panel(m.stage).Field("member").Filter)
	}
	rows := m.rows(m.stage)
	if k := m.stages[m.stage].Cursor; k >= len(rows) || rows[k].Key != "github.com/o/b remote origin" {
		t.Errorf("the cursor is not on b's remote: %d", k)
	}
	if v := view(m); !strings.Contains(v, "v1") || !strings.Contains(v, "v2") {
		t.Errorf("incoming shows every member's:\n%s", v)
	}
}
