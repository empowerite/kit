// Issues of more than one tracker: each by its reference, under its tracker where there is more than one, its page
// the one kit gave where the forge's is not.
package flow

import (
	"slices"
	"testing"

	"github.com/empowerite/kit/projects/tui/internal/table"
)

// trackers is the issues stage of a repository with an Azure DevOps tracker beside the forge's own: #7 on GitHub, and
// two work items, AB#4521 with its page and AB#9 without one.
const trackers = `[
  {"number": 7, "ref": "#7", "tracker": "#", "title": "Seven", "source": "github", "status": "backlog"},
  {"number": 4521, "ref": "AB#4521", "tracker": "AB#", "title": "Claim form", "source": "ado", "status": "backlog",
   "url": "https://dev.azure.com/example/Claims/_workitems/edit/4521"},
  {"number": 9, "ref": "AB#9", "tracker": "AB#", "title": "Nine", "source": "ado", "status": "backlog"}]`

// issuesOf gives issues alone, loaded with the document.
func issuesOf(t *testing.T, doc string) Model {
	t.Helper()
	o := options()
	o.Stages = []string{"issues"}
	return drive(New(runner(t), o), answer("issues", doc))
}

// An issue of another tracker is drawn by its reference, its title leading to the page kit gave, or nowhere, as kit
// knows no page of a tracker it did not name one for; the forge's own leads to its page on the forge, as ever. Its
// tracker is a value of its own.
func TestAnIssueOfAnotherTrackerIsItsReference(t *testing.T) {
	m := issuesOf(t, trackers)
	got := map[string][2]string{}
	for _, r := range m.records(*m.find(issuesStage)) {
		got[r.values["id"].Raw[0]] = [2]string{r.values["tracker"].Raw[0], r.values["title"].Link}
	}
	want := map[string][2]string{
		"#7":      {"#", "https://github.com/empowerite/kit/issues/7"},
		"AB#4521": {"AB#", "https://dev.azure.com/example/Claims/_workitems/edit/4521"},
		"AB#9":    {"AB#", ""},
	}
	for ref, w := range want {
		if got[ref] != w {
			t.Errorf("%s: %q, want %q", ref, got[ref], w)
		}
	}
}

// Issues of two trackers stand under each first, outermost, as they load; those of one tracker alone stand as ever,
// under no heading of it, and so do those of a view that names the tracker.
func TestIssuesStandUnderTheirTrackerWhereThereAreSeveral(t *testing.T) {
	if g := issuesOf(t, trackers).panel(0).Groups(); len(g) == 0 || g[0] != "tracker" {
		t.Errorf("two trackers group by %q", g)
	}
	one := `[{"number": 7, "ref": "#7", "tracker": "#", "title": "Seven", "source": "github", "status": "backlog"},
	  {"number": 8, "ref": "#8", "tracker": "#", "title": "Eight", "source": "github", "status": "backlog"}]`
	if g := issuesOf(t, one).panel(0).Groups(); slices.Contains(g, "tracker") {
		t.Errorf("one tracker groups by %q", g)
	}
	if several([]record{{}, {values: map[string]table.Value{"tracker": {Raw: []string{"#"}}}}}, "tracker") {
		t.Error("a record with no tracker counts as one")
	}
}

// A row's key holds its issue's reference, whatever its prefix, after the member over a group, and the pull request's
// after it for an issue underway.
func TestARowsKeyGivesTheReferenceOfAnyTracker(t *testing.T) {
	for key, want := range map[string][2]string{
		"AB#4521":                   {"", "AB#4521"},
		"AB#4521 #3":                {"", "AB#4521"},
		"github.com/o/r AB#4521 #3": {"github.com/o/r", "AB#4521"},
		"github.com/o/r #45":        {"github.com/o/r", "#45"},
	} {
		if member, ref := refOf(key); member != want[0] || ref != want[1] {
			t.Errorf("%q: %q %q, want %q", key, member, ref, want)
		}
	}
}
