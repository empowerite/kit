// What a stage's rows draw from what kit printed: where the work of an issue underway is, and an answer the screen
// has no stage for.
package flow

import (
	"maps"
	"slices"
	"testing"

	"github.com/empowerite/kit/projects/tui/internal/kit"
	"github.com/empowerite/kit/projects/tui/internal/shell"
	"github.com/empowerite/kit/projects/tui/internal/source"
	"github.com/empowerite/kit/projects/tui/internal/table"
)

// Issues draw an issue in the backlog and one underway in one stage: each its status, milestone, iteration, swimlane,
// and branch, as kit prints them; one underway its pull request beside it, its number, state, and where the work is,
// and one in the backlog none of those, nor a branch.
func TestIssuesDrawTheBacklogAndWhatIsUnderwayTogether(t *testing.T) {
	m := loaded(t, options())
	values := map[string]map[string][]string{}
	for _, r := range m.records(*m.find(issuesStage)) {
		values[r.Key] = map[string][]string{}
		for id, v := range r.values {
			values[r.Key][id] = v.Raw
		}
	}
	for key, want := range map[string]map[string][]string{
		"#7": {"status": {"backlog"}, "milestone": {"v1"}, "iteration": nil, "swimlane": nil, "branch": nil,
			"change": nil, "state": nil, "where": nil, "pushed": nil},
		"#46": {"status": {"backlog"}, "milestone": {"v0.2.0"}, "iteration": {"Sprint 4"}, "swimlane": {"tui"},
			"branch": nil, "change": nil, "where": nil},
		"#1 #2": {"status": {"underway"}, "milestone": {"v0.1.0"}, "iteration": {"Sprint 3"}, "swimlane": {"release"},
			"branch": {"1-the-first-release-v0-1-0"}, "change": {"#2"}, "state": {"ready"}, "where": {"here"}},
		"#8 #11": {"status": {"underway"}, "change": {"#11"}, "state": {"draft"}, "where": {"local"}},
	} {
		got, ok := values[key]
		if !ok {
			t.Errorf("no row %s: %q", key, slices.Collect(maps.Keys(values)))
			continue
		}
		for id, raw := range want {
			if !slices.Equal(got[id], raw) {
				t.Errorf("%s's %s: %q, want %q", key, id, got[id], raw)
			}
		}
	}
	if len(values) != 7 {
		t.Errorf("%d rows, not 7: %q", len(values), slices.Collect(maps.Keys(values)))
	}
}

// An issue underway that no pull request closes, as kit prints one a branch is named for, has no change: where draws
// the branch after a branch's glyph, dim, on the primary; here, or local, where this desk has it, as for a pull
// request's.
func TestAnIssueABranchNamesIsUnderwayOnIt(t *testing.T) {
	m := loaded(t, options())
	s := stage{name: issuesStage}
	if err := s.decode([]byte(`[
	  {"number": 9, "title": "Nine", "status": "underway", "branch": "9-other", "pr": null, "here": false, "local": false},
	  {"number": 10, "title": "Ten", "status": "underway", "branch": "10-ten", "pr": null, "here": false, "local": true},
	  {"number": 11, "title": "Eleven", "status": "backlog", "branch": null, "pr": null, "here": null, "local": null}]`),
		false); err != nil {
		t.Fatal(err)
	}
	where := map[string]table.Value{}
	for _, r := range m.records(s) {
		where[r.Key] = r.values["where"]
		if r.values["change"].Raw != nil {
			t.Errorf("%s has a change: %q", r.Key, r.values["change"].Raw)
		}
	}
	if v := where["#9"]; v.Of != "branch" || !slices.Equal(v.Raw, []string{"9-other"}) {
		t.Errorf("#9's where: %+v", v)
	}
	if v := where["#10"]; v.Of != "local" || where["#11"].Raw != nil {
		t.Errorf("#10's where: %+v, #11's: %+v", v, where["#11"])
	}
	if got := whereCells(table.Field{}, []table.Value{where["#9"]}); !slices.Equal(got[0].Values, dim(Branch+" 9-other").Values) {
		t.Errorf("the branch drawn: %+v", got)
	}
}

func TestTheDeskIsHereInIssues(t *testing.T) {
	p := "/home/kit/worktrees/1-the-first-release-v0-1-0"
	o := options()
	o.App, o.Stages = shell.Issues, []string{"issues"}
	m := loaded(t, o, shell.WorkspaceMsg{Workspace: kit.Workspace{Kind: "repository", Checkout: p}})
	where := func(m Model) (any, string) {
		for _, r := range m.records(*m.find("issues")) {
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
