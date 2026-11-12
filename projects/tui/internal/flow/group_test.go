// Over a group, every stage's document is kit's {rows, errors}: each member's rows, every row tagged with its member,
// and the members whose read failed. The screen draws every member's rows, keyed apart by member, each linking where
// its own member's do, and a failed member as one line among them.
package flow

import (
	"slices"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/empowerite/kit/projects/tui/internal/kit"
	"github.com/empowerite/kit/projects/tui/internal/shell"
	"github.com/empowerite/kit/projects/tui/internal/testkit"
)

// group is the workspace the tests' screens are told of: the group w of github.com/o/a and github.com/o/b, a cloned
// at /code/a and b at /code/b.
var group = kit.Workspace{Kind: "group", Name: "w", Members: []kit.WorkspaceMember{
	{Repository: "github.com/o/a", Checkout: "/code/a"}, {Repository: "github.com/o/b", Checkout: "/code/b"}}}

// over gives the screen over the group w, at 120×24, told of its members, its stages as given, none loaded: a test
// answers each itself.
func over(t *testing.T, stages ...string) Model {
	t.Helper()
	r := runner(t)
	r.Group = "w"
	o := Options{App: shell.Flow, Stages: stages, NoPresets: true, Clock: &testkit.Clock{}}
	if slices.ContainsFunc(stages, ofWorkspace) {
		o.App = shell.Repositories
	}
	m := New(r, o)
	return drive(m, tea.WindowSizeMsg{Width: 120, Height: 24}, shell.WorkspaceMsg{Workspace: group})
}

// ofWorkspace says whether a stage, as kittui knows it, is kit workspace's.
func ofWorkspace(name string) bool { return strings.HasPrefix(name, workspaceVerb+"/") }

// recordKeys gives the keys of a stage's records, in their order.
func recordKeys(m Model, name string) []string {
	var out []string
	for _, r := range m.records(*m.find(name)) {
		out = append(out, r.Key)
	}
	return out
}

func TestOverAGroupEachStageDrawsEveryMembersRows(t *testing.T) {
	m := over(t, "issues")
	m = drive(m, answer("issues", `{"rows": [{"number": 1, "title": "a1", "member": "github.com/o/a"},
		{"number": 1, "title": "b1", "member": "github.com/o/b"}], "errors": []}`))
	if s := m.find("issues"); s.refusal != "" {
		t.Fatalf("refused: %s", s.refusal)
	}
	if got := recordKeys(m, "issues"); strings.Join(got, "|") != "github.com/o/a #1|github.com/o/b #1" {
		t.Errorf("the keys: %q", got)
	}
	records := m.records(*m.find("issues"))
	if got := records[1].values["title"].Link; got != "https://github.com/o/b/issues/1" {
		t.Errorf("b's issue links to %q", got)
	}
	if v := view(m); !strings.Contains(v, "a1") || !strings.Contains(v, "b1") {
		t.Errorf("both rows are not drawn:\n%s", v)
	}
}

func TestAMemberThatFailedIsItsLineAndTheOthersDraw(t *testing.T) {
	m := over(t, "changes")
	m = drive(m, answer("changes", `{"rows": [{"number": 3, "title": "a's change", "member": "github.com/o/a"}],
		"errors": [{"member": "github.com/o/b", "error": "gh cannot read o/b"}]}`))
	s := m.find("changes")
	if s.refusal != "" {
		t.Fatalf("the whole stage refused: %s", s.refusal)
	}
	v := view(m)
	if !strings.Contains(v, "a's change") || !strings.Contains(v, Failed+" b: gh cannot read o/b") {
		t.Errorf("a's row and b's line:\n%s", v)
	}
}

func TestAPullRequestsLinkLandsOnItsMembersRow(t *testing.T) {
	m := over(t, "issues", "changes")
	m = drive(m, answer("changes", `{"rows": [{"number": 7, "title": "a7", "member": "github.com/o/a"},
		{"number": 7, "title": "b7", "member": "github.com/o/b"}], "errors": []}`))
	m = drive(m, answer("issues", `{"rows": [
		{"number": 1, "title": "a1", "pr": {"number": 7}, "member": "github.com/o/a"},
		{"number": 1, "title": "b1", "pr": {"number": 7}, "member": "github.com/o/b"}], "errors": []}`))
	records := m.records(*m.find("issues"))
	link := records[1].values["change"].Link
	if link != jumpTo("changes", "github.com/o/b #7") {
		t.Fatalf("b's change links to %q", link)
	}
	m.jump(strings.TrimPrefix(link, jumps))
	if m.stages[m.stage].name != "changes" {
		t.Fatalf("the jump selected %s", m.stages[m.stage].name)
	}
	rows := m.rows(m.stage)
	if k := m.stages[m.stage].Cursor; k >= len(rows) || rows[k].Key != "github.com/o/b #7" {
		t.Errorf("the cursor is not on b's change: %d of %v", k, slices.Collect(func(yield func(string) bool) {
			for _, r := range rows {
				yield(r.Key)
			}
		}))
	}
}

func TestOverAGroupAnArrayIsNoEnvelope(t *testing.T) {
	m := over(t, "issues")
	m = drive(m, answer("issues", `[]`))
	want := `kit flow issues --json printed an array where an object with "rows" was expected`
	if got := m.find("issues").refusal; got != want {
		t.Errorf("the refusal: %q", got)
	}
}

func TestOverOneRepositoryTheStagesDecodeAsBefore(t *testing.T) {
	m := loaded(t, options())
	if got := recordKeys(m, "builds"); strings.Join(got, "|") != "41309409501|41309409467|41309409400|41309409322" {
		t.Errorf("the builds: %q", got)
	}
}

// twoMembers is issues' document over the group w: an issue of a and one of b.
const twoMembers = `{"rows": [{"number": 1, "title": "a1", "member": "github.com/o/a"},
	{"number": 2, "title": "b2", "member": "github.com/o/b"}], "errors": []}`

func TestOverAGroupTheRowsGroupByMemberAsTheyLoad(t *testing.T) {
	m := drive(over(t, "issues"), answer("issues", twoMembers))
	if groups := m.panel(0).Groups(); len(groups) == 0 || groups[0] != "member" {
		t.Fatalf("the groups as issues load: %q", groups)
	}
	member := m.records(*m.find("issues"))[0].values["member"]
	if member.Of != "a" || !slices.Equal(member.Raw, []string{"github.com/o/a"}) {
		t.Errorf("a's member: %+v", member)
	}
	if v := view(m); !strings.Contains(v, "▾ a") || !strings.Contains(v, "▾ b") {
		t.Errorf("the headings name the members short:\n%s", v)
	}
}

// A member's name leads to its row on the repos tab: in kit workspace's picture, its own; in kit flow's, which has no
// repos tab, Repositories'.
func TestAMembersNameLeadsToItsReposRow(t *testing.T) {
	m := drive(over(t, "workspace/repos", "workspace/remotes"), answer("workspace/remotes",
		`{"rows": [{"key": "origin", "remote": "origin", "member": "github.com/o/b"}], "errors": []}`))
	if got := m.records(*m.find("workspace/remotes"))[0].values["member"].Link; got != jumpTo("workspace/repos", "github.com/o/b") {
		t.Errorf("b's member links to %q", got)
	}
	m = drive(over(t, "issues"), answer("issues", twoMembers))
	if got := m.records(*m.find("issues"))[1].values["member"].Link; got != jumpTo("workspace/repos", "github.com/o/b") {
		t.Errorf("in kit flow's picture, b's member links to %q", got)
	}
}

func TestMembersOfTwoOwnersAreNamedInFull(t *testing.T) {
	m := over(t, "issues")
	two := group
	two.Members = []kit.WorkspaceMember{{Repository: "github.com/o/a"}, {Repository: "github.com/p/b"}}
	m = drive(m, shell.WorkspaceMsg{Workspace: two}, answer("issues", `{"rows": [
		{"number": 1, "title": "a1", "member": "github.com/o/a"},
		{"number": 2, "title": "b2", "member": "github.com/p/b"}], "errors": []}`))
	if got := m.records(*m.find("issues"))[1].values["member"].Of; got != "github.com/p/b" {
		t.Errorf("b of another owner is named %q", got)
	}
}

func TestOverOneRepositoryARowHasNoMember(t *testing.T) {
	m := loaded(t, options())
	if v := m.records(*m.find("issues"))[0].values["member"]; len(v.Raw) != 0 || v.Link != "" {
		t.Errorf("a repository's issue has a member: %+v", v)
	}
	if groups := m.panel(0).Groups(); slices.Contains(groups, "member") {
		t.Errorf("a repository's rows group by member: %q", groups)
	}
}

func TestOverAGroupOnlyRowsAndErrorsAreTheStagesDocument(t *testing.T) {
	for data, want := range map[string]string{
		``: `kit flow issues --json printed nothing where an object with "rows" was expected`,
		`{"errors": []}`: `kit flow issues --json printed an object without "rows" where an object with "rows" ` +
			"was expected",
		`{"rows": 1}`: `kit flow issues --json printed a number "rows" where an object with "rows" was expected`,
		`{"rows": [{"number": "1"}]}`: "kit flow issues --json: json: cannot unmarshal string into Go struct " +
			"field issue.number of type int",
		`{bad`: "kit flow issues --json: invalid character 'b' looking for beginning of object key string",
		`nope`: "kit flow issues --json: invalid character 'o' in literal null (expecting 'u')",
	} {
		m := drive(over(t, "issues"), answer("issues", data))
		if got := m.find("issues").refusal; got != want {
			t.Errorf("%q: %q, want %q", data, got, want)
		}
	}
}

func TestAMemberOfNoOwnerIsNamedAsKitNamesIt(t *testing.T) {
	m := over(t, "issues")
	if got := m.short("local"); got != "local" {
		t.Errorf("local is named %q", got)
	}
}
