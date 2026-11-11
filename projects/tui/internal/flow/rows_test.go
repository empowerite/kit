// What a stage's rows draw from what kit printed: where the work of an issue underway is, and an answer the screen
// has no stage for.
package flow

import (
	"testing"

	"github.com/empowerite/kit/projects/tui/internal/kit"
	"github.com/empowerite/kit/projects/tui/internal/shell"
	"github.com/empowerite/kit/projects/tui/internal/source"
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
