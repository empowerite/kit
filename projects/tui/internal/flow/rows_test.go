// What a stage's rows draw from what kit printed: where the work of an issue underway is, and an answer the screen
// has no stage for.
package flow

import (
	"slices"
	"testing"

	"github.com/empowerite/kit/projects/tui/internal/kit"
	"github.com/empowerite/kit/projects/tui/internal/shell"
	"github.com/empowerite/kit/projects/tui/internal/source"
	"github.com/empowerite/kit/projects/tui/internal/table"
)

func TestTheDeskIsHereInIssues(t *testing.T) {
	p := "/home/kit/worktrees/1-the-first-release-v0-1-0"
	o := options()
	o.App, o.Stages = shell.Issues, []string{"backlog", "underway"}
	m := loaded(t, o, shell.WorkspaceMsg{Workspace: kit.Workspace{Kind: "repository", Checkout: p}})
	where := func(m Model) (any, string) {
		for _, r := range m.records(*m.find("underway")) {
			if r.Key == "#1 #2" {
				return r.values["where"].Of, r.values["where"].Link
			}
		}
		return nil, ""
	}
	if of, link := where(m); of != "desk" || link != jumpTo("workspace/worktrees", p) {
		t.Errorf("the desk's checkout: %v, leading to %q", of, link)
	}
	// a checkout that is not the desk is a worktree here, leading to the same tab
	m = drive(m, shell.WorkspaceMsg{Workspace: kit.Workspace{Kind: "repository", Checkout: "/home/kit"}})
	if of, link := where(m); of != "here" || link != jumpTo("workspace/worktrees", p) {
		t.Errorf("a worktree's checkout: %v, leading to %q", of, link)
	}
	// over a group, a member's desk is its checkout
	if m = drive(m, shell.WorkspaceMsg{Workspace: group}); m.deskOf("github.com/o/b") != "/code/b" ||
		m.deskOf("github.com/o/c") != "" {
		t.Errorf("the members' desks: %q, %q", m.deskOf("github.com/o/b"), m.deskOf("github.com/o/c"))
	}
}

func TestAnAnswerForAStageTheScreenDoesNotHaveIsDropped(t *testing.T) {
	m := loaded(t, options())
	before := view(m)
	for _, kept := range []bool{false, true} {
		if m = drive(m, source.Answer[[]byte]{Name: "nonesuch", Value: []byte(`[]`), Kept: kept}); view(m) != before {
			t.Errorf("an answer for no stage, kept %v, changed the screen:\n%s", kept, view(m))
		}
	}
}

// A field a stage's builder leaves unset, as where a build is from, is read from the record kit printed by the field's
// path, as the catalog draws a stage with no builder; a value the builder set stands, and a builder that built another
// number of records than kit printed is left as it built them.
func TestAFieldABuilderLeavesUnsetIsReadFromTheRecord(t *testing.T) {
	s := stage{name: "builds", parsed: []parsed{{value: map[string]any{"organization": "o", "repository": "r",
		"source": "github", "workflow": "kit"}}}}
	out := []record{{row{Key: "1"}, map[string]table.Value{"workflow": table.One("built")}}}
	unbuilt(s, out)
	for id, want := range map[string]string{"organization": "o", "repository": "r", "source": "github", "workflow": "built"} {
		if got := out[0].values[id].Raw; !slices.Equal(got, []string{want}) {
			t.Errorf("%s: %q, want %q", id, got, want)
		}
	}
	if _, ok := out[0].values["member"]; ok {
		t.Errorf("member is read from the record: %+v", out[0].values["member"])
	}
	two := []record{{row{Key: "1"}, map[string]table.Value{}}, {row{Key: "2"}, map[string]table.Value{}}}
	if unbuilt(s, two); len(two[0].values) != 0 || len(two[1].values) != 0 {
		t.Errorf("two records of one kit printed: %+v", two)
	}
}
