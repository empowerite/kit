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
	m := New(r, Options{Stages: stages, NoPresets: true, Clock: &testkit.Clock{}})
	return drive(m, tea.WindowSizeMsg{Width: 120, Height: 24}, shell.WorkspaceMsg{Workspace: group})
}

// recordKeys gives the keys of a stage's records, in their order.
func recordKeys(m Model, name string) []string {
	var out []string
	for _, r := range m.records(*m.find(name)) {
		out = append(out, r.Key)
	}
	return out
}

func TestOverAGroupEachStageDrawsEveryMembersRows(t *testing.T) {
	m := over(t, "backlog")
	m = drive(m, answer("backlog", `{"rows": [{"number": 1, "title": "a1", "member": "github.com/o/a"},
		{"number": 1, "title": "b1", "member": "github.com/o/b"}], "errors": []}`))
	if s := m.find("backlog"); s.refusal != "" {
		t.Fatalf("refused: %s", s.refusal)
	}
	if got := recordKeys(m, "backlog"); strings.Join(got, "|") != "github.com/o/a #1|github.com/o/b #1" {
		t.Errorf("the keys: %q", got)
	}
	records := m.records(*m.find("backlog"))
	if got := records[1].values["title"].Link; got != "https://github.com/o/b/issues/1" {
		t.Errorf("b's issue links to %q", got)
	}
	if v := view(m); !strings.Contains(v, "a1") || !strings.Contains(v, "b1") {
		t.Errorf("both rows are not drawn:\n%s", v)
	}
}

func TestOverAGroupTheDeskIsAPerMemberRow(t *testing.T) {
	m := over(t, "desk")
	m = drive(m, answer("desk", `{"rows": [
		{"path": "/code/a", "branch": "main", "ahead": 1, "member": "github.com/o/a"},
		{"path": "/code/b", "branch": "topic", "member": "github.com/o/b"}], "errors": []}`))
	if got := recordKeys(m, "desk"); strings.Join(got, "|") != "/code/a|/code/b" {
		t.Errorf("the desks: %q", got)
	}
	if v := view(m); !strings.Contains(v, "main") || !strings.Contains(v, "topic") {
		t.Errorf("both desks are not drawn:\n%s", v)
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
	m := over(t, "underway", "changes")
	m = drive(m, answer("changes", `{"rows": [{"number": 7, "title": "a7", "member": "github.com/o/a"},
		{"number": 7, "title": "b7", "member": "github.com/o/b"}], "errors": []}`))
	m = drive(m, answer("underway", `{"rows": [
		{"number": 1, "title": "a1", "pr": {"number": 7}, "member": "github.com/o/a"},
		{"number": 1, "title": "b1", "pr": {"number": 7}, "member": "github.com/o/b"}], "errors": []}`))
	records := m.records(*m.find("underway"))
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
	m := over(t, "backlog")
	m = drive(m, answer("backlog", `[]`))
	want := "kit flow backlog --workspace --json printed an array where {rows, errors} was expected"
	if got := m.find("backlog").refusal; got != want {
		t.Errorf("the refusal: %q", got)
	}
}

func TestOverOneRepositoryTheStagesDecodeAsBefore(t *testing.T) {
	m := loaded(t, options())
	if got := recordKeys(m, "desk"); strings.Join(got, "|") != "/home/kit" {
		t.Errorf("the desk: %q", got)
	}
}
