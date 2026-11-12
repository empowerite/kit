// The field catalog holds together: each stage's ids are its own, every type and renderer it names is registered, its
// presets name only its fields, and the stage's rows give a value in each of its fields.
package flow

import (
	"maps"
	"reflect"
	"slices"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/empowerite/kit/projects/tui/internal/grid"
	"github.com/empowerite/kit/projects/tui/internal/shell"
	"github.com/empowerite/kit/projects/tui/internal/table"
)

func TestTheCatalogNamesOnlyWhatIsRegistered(t *testing.T) {
	// every stage, incoming among them, kit workspace's, its repos tab among them, kit packages', and every other app's
	every := map[string]bool{}
	for _, name := range five {
		every[name] = true
	}
	for _, a := range apps {
		for _, name := range a.stages {
			every[name] = true
		}
	}
	if len(catalog) != len(every) || catalog["incoming"] == nil || catalog["workspace/repos"] == nil ||
		catalog[packagesStage] == nil || len(every) != len(five)+len(workspaceStages)+1+20 {
		t.Errorf("the catalog's stages: %d of %d, incoming %v", len(catalog), len(every), catalog["incoming"] != nil)
	}
	// each names the command that prints it, a stage of its verb's, and each stage Go builds is the catalog's
	for name := range catalog {
		if verb, stage := verbOf(name); !sourced(name) || verb == "" || label(name) != stage {
			t.Errorf("%s reads %q", name, commandOf(name))
		}
	}
	for name := range builders {
		if catalog[name] == nil {
			t.Errorf("Go builds %s, which the catalog does not name", name)
		}
	}
	var glyphs []string
	for _, l := range legends {
		for _, e := range l {
			glyphs = append(glyphs, e.glyph)
		}
	}
	for _, e := range states {
		glyphs = append(glyphs, e.glyph)
	}
	for name, s := range catalog {
		if s.Stage != name || s.Record == "" || len(s.Fields) == 0 {
			t.Errorf("%s: stage %q, record %q, %d fields", name, s.Stage, s.Record, len(s.Fields))
		}
		ids := map[string]bool{}
		for _, f := range s.Fields {
			if f.ID == "" || ids[f.ID] {
				t.Errorf("%s: the id %q is empty or not unique", name, f.ID)
			}
			ids[f.ID] = true
			kind, ok := table.Types[f.Type]
			if !ok {
				t.Errorf("%s.%s: the type %q is not registered", name, f.ID, f.Type)
			}
			if f.Filter != kind.Filters {
				t.Errorf("%s.%s: filter is %v for a %s", name, f.ID, f.Filter, f.Type)
			}
			if f.Path == "" {
				t.Errorf("%s.%s: no path", name, f.ID)
			}
			if len(f.Renderings) == 0 {
				t.Errorf("%s.%s: no rendering", name, f.ID)
			}
			for _, r := range f.Renderings {
				if !table.Registered(r) {
					t.Errorf("%s.%s: the renderer %q is not registered", name, f.ID, r)
				}
			}
			if !slices.Contains([]string{table.PlaceColumn, table.PlaceGroup, table.PlaceHidden}, f.Placement) {
				t.Errorf("%s.%s: the placement %q", name, f.ID, f.Placement)
			}
			if f.Type != "enum" && len(f.Values) > 0 {
				t.Errorf("%s.%s: a %s with %d values", name, f.ID, f.Type, len(f.Values))
			}
			if f.Order != "" && (f.Order != table.OrderGiven || !f.Group) {
				t.Errorf("%s.%s: the order %q, for a field that groups %v", name, f.ID, f.Order, f.Group)
			}
			if f.Owns && f.Type != "person" {
				t.Errorf("%s.%s: a %s owns the rows", name, f.ID, f.Type)
			}
			for _, g := range append([]string{f.Glyph, f.Other}, valueGlyphs(f)...) {
				if g != "" && !slices.Contains(glyphs, g) {
					t.Errorf("%s.%s: the glyph %q is in no legend", name, f.ID, g)
				}
			}
		}
		presets := map[string]bool{}
		for _, p := range s.Presets {
			if p.Name == "" || presets[p.Name] {
				t.Errorf("%s: the preset %q is unnamed or not unique", name, p.Name)
			}
			presets[p.Name] = true
			for id, set := range p.Fields {
				k := slices.IndexFunc(s.Fields, func(f table.Field) bool { return f.ID == id })
				if k < 0 {
					t.Errorf("%s: the preset %s names %q, not a field of the stage", name, p.Name, id)
					continue
				}
				checkSetting(t, name+" "+p.Name+" "+id, s.Fields[k], set)
			}
		}
		want := []string{table.Default, "Compact"}
		if name == issuesStage {
			want = []string{"All Issues", "Backlog", "Underway", "By Milestone", "By Iteration", "By Swimlane", "Compact"}
		}
		if got := slices.Collect(func(yield func(string) bool) {
			for _, p := range s.Presets {
				yield(p.Name)
			}
		}); !slices.Equal(got, want) {
			t.Errorf("%s: the presets %v", name, got)
		}
	}
	// a catalog that does not read is kittui's own fault, and stops it as it starts
	func() {
		defer func() {
			if r, _ := recover().(string); !strings.HasPrefix(r, "fields.json does not read: ") {
				t.Errorf("a catalog that does not read: %q", r)
			}
		}()
		readCatalog([]byte("{"))
	}()
}

// The catalog draws a pull request's state, a build's, and a release's kind as the legend does: each known value with
// the glyph the legend gives it, and Other, or a release's own, for one it does not know.
func TestTheCatalogsGlyphsAreTheLegends(t *testing.T) {
	drawn := func(stage, id string, raw ...string) []string {
		k := slices.IndexFunc(fieldsOf(stage), func(f table.Field) bool { return f.ID == id })
		return fieldsOf(stage)[k].Glyphed(raw, false)
	}
	for _, stage := range []string{"issues", "changes"} {
		for state, glyph := range stateGlyphs {
			if got := drawn(stage, "state", state); got[0] != glyph {
				t.Errorf("%s draws %s as %s, not %s", stage, state, got[0], glyph)
			}
		}
		if got := drawn(stage, "state", "nonesuch"); got[0] != Other {
			t.Errorf("%s draws a state it does not know as %s", stage, got[0])
		}
	}
	for _, state := range []string{"running", "passed", "failed", "cancelled", "queued"} {
		if got := drawn("builds", "status", state); got[0] != runGlyph(state) {
			t.Errorf("builds draws %s as %s, not %s", state, got[0], runGlyph(state))
		}
	}
	if got := drawn("releases", "kind", "draft", "pre-release", "release", "other"); !slices.Equal(got,
		[]string{Draft, Pre, Release, Release}) {
		t.Errorf("the releases' kinds: %q", got)
	}
}

// valueGlyphs gives the glyphs of an enum's known values.
func valueGlyphs(f table.Field) []string {
	var out []string
	for _, v := range f.Values {
		out = append(out, v.Glyph)
	}
	return out
}

// checkSetting fails on what a preset cannot say of a field: a placement it does not know, or a column's, a group's, or
// a sort's place that is not from 1; a group or a sort the field cannot do, or a filter it cannot take, with values
// that are none of the field's; a rendering not the field's.
func checkSetting(t *testing.T, where string, f table.Field, s table.Setting) {
	t.Helper()
	switch s.Placement {
	case "", table.PlaceHidden:
	case table.PlaceColumn:
		if s.Position < 0 {
			t.Errorf("%s: the position %d", where, s.Position)
		}
	case table.PlaceGroup:
		if !f.Group || s.Depth < 1 {
			t.Errorf("%s: grouped at %d, the field groups %v", where, s.Depth, f.Group)
		}
	default:
		t.Errorf("%s: the placement %q", where, s.Placement)
	}
	if s.Sort != "" && (!f.Sort || s.Sort != "asc" && s.Sort != "desc" || s.Priority < 1) {
		t.Errorf("%s: sorted %q at %d, the field sorts %v", where, s.Sort, s.Priority, f.Sort)
	}
	if s.Filter != nil {
		if !f.Filter {
			t.Errorf("%s: a filter on a field that takes none", where)
		}
		for _, v := range slices.Concat(s.Filter.Include, s.Filter.Exclude) {
			known := slices.ContainsFunc(f.Values, func(e table.EnumValue) bool { return e.Value == v })
			// a person's filter names logins, @me among them, and none; a label's, any label's name, and none
			if !known && (f.Type != "person" && f.Type != "label" || v == "") {
				t.Errorf("%s: the filter's value %q", where, v)
			}
		}
	}
	if s.Rendering != "" && !slices.Contains(f.Renderings, s.Rendering) {
		t.Errorf("%s: the rendering %q is not the field's", where, s.Rendering)
	}
}

// Each stage's records give a value in every one of its fields, by id, and none in a field it does not have; its rows
// then give their cells in the order of its fields, one a field.
func TestTheRowsGiveAValueInEachFieldAndCellsInItsOrder(t *testing.T) {
	m := loaded(t, options())
	for _, s := range m.stages {
		fields := fieldsOf(s.name)
		if fields == nil {
			continue
		}
		var ids []string
		for _, f := range fields {
			ids = append(ids, f.ID)
		}
		records := m.records(s)
		if len(records) == 0 {
			t.Errorf("%s: no rows to check", s.name)
		}
		for _, r := range records {
			if r.values == nil {
				continue
			}
			if got := slices.Sorted(maps.Keys(r.values)); !slices.Equal(got, slices.Sorted(slices.Values(ids))) {
				t.Errorf("%s: the row %s gives %q, the stage's fields are %q", s.name, r.Key, got, ids)
			}
		}
		for _, r := range laid(s.name, records, func(string) string { return "" }) {
			if len(r.Cells) > 1 && len(r.Cells) != len(fields) {
				t.Errorf("%s: the row %s has %d cells, the stage %d fields", s.name, r.Key, len(r.Cells), len(fields))
			}
		}
	}
}

// A preset says, per field, its placement, its sort, and its filter, and whether a list is flat, as assignees, which
// sorts while it is; the panel is set as it says, and gives back a preset that sets it alike.
func TestAPresetSetsThePanel(t *testing.T) {
	fields := fieldsOf("issues")
	p := table.Preset{Name: "Triage", Fields: map[string]table.Setting{
		"title":     {Placement: table.PlaceColumn, Position: 1},
		"id":        {Placement: table.PlaceColumn, Position: 2},
		"source":    {Placement: table.PlaceGroup, Depth: 1},
		"branch":    {Placement: table.PlaceHidden, Rendering: "text"},
		"age":       {Sort: "desc", Priority: 1},
		"assignees": {Flat: true, Sort: "asc", Priority: 2, Filter: &table.ValueFilter{Exclude: []string{caller}}},
	}}
	g := p.Panel(fields, grid.Panel{Owner: ownerOf("issues"), Ones: onesOf("issues")})
	if got := g.Arrange(columnsOf("issues")); !slices.Equal(got, []string{"title", "id", "member", "source",
		"organization", "repository", "status", "assignees", "age", "milestone", "iteration", "swimlane", "branch",
		"change", "state", "where", "fork", "pushed", "by", "labels"}) ||
		!slices.Equal(g.Groups(), []string{"source", "organization", "repository"}) ||
		g.Field("branch").Placement != grid.Hidden ||
		!slices.Equal(g.SortKeys(), []grid.SortKey{{Column: "age", Desc: true}, {Column: "assignees"}}) ||
		g.Showing() != grid.Others || g.Field("branch").Rendering != "text" {
		t.Errorf("the panel: %q %+v", got, g.Fields)
	}
	back := table.PresetOf(g, fields)
	if !reflect.DeepEqual(table.PresetOf(back.Panel(fields, grid.Panel{}), fields), back) ||
		back.Fields["source"] != (table.Setting{}) || back.Fields["title"].Position != 1 {
		t.Errorf("the preset again: %+v", back)
	}
	// source grouped outermost, as the catalog groups it, says nothing read back; organization and repository, which
	// Triage names not, group under it as the catalog says
	// the stage's own: Default shows every column but member, which only a group's rows hold, and groups by source,
	// organization, and repository, as every stage whose rows belong to a repository does; Compact, the few that say
	// what a row is
	builds := catalog["builds"]
	if d := builds.Presets[0].Panel(builds.Fields, grid.Panel{}); builds.Presets[0].Name != table.Default ||
		!slices.Equal(hiddenOf(grid.Panel{Fields: d.Fields, Columns: columnsOf("builds")}), []string{"member"}) ||
		!slices.Equal(d.Groups(), []string{"source", "organization", "repository"}) {
		t.Errorf("Default: %+v", d.Fields)
	}
	if c := builds.Presets[1].Panel(builds.Fields, grid.Panel{Columns: columnsOf("builds")}); builds.Presets[1].Name != "Compact" ||
		!slices.Equal(hiddenOf(c), []string{"member", "run", "event", "branch", "commit", "took"}) ||
		!slices.Equal(c.Groups(), []string{"source", "organization", "repository"}) {
		t.Errorf("Compact: %+v", c.Fields)
	}
	// a rendering pinned draws the field
	title := fields[slices.IndexFunc(fields, func(f table.Field) bool { return f.ID == "title" })]
	if c := title.Draw([]table.Value{table.One("a")}, "text"); c[0].Values[0] != "a" {
		t.Errorf("title pinned to text: %q", c[0].Values)
	}
	if c := title.Draw([]table.Value{table.One("a")}, "age"); c[0].Values[0] != Issue+" a" {
		t.Errorf("title pinned to a rendering not its own: %q", c[0].Values)
	}
}

// An issue's organization and repository, as kit prints them, are fields of issues, in the backlog and underway alike,
// which group the rows under them: the source and the organization every issue shares as one heading, each repository's
// under it; a filter on them offers the values the rows hold.
func TestAnIssuesOrganizationAndRepositoryAreFields(t *testing.T) {
	m := loaded(t, options(), tea.WindowSizeMsg{Width: 160, Height: 24})
	for id, depth := range map[string]int{"source": 1, "organization": 2, "repository": 3} {
		if f := fieldIn(issuesStage, id); f.Placement != table.PlaceGroup || f.Depth != depth || !f.Filter || !f.Group {
			t.Errorf("issues.%s: %+v", id, f)
		}
	}
	for _, heading := range []string{"▾ github · empowerite 7", "▾ jig 1", "▾ kit 6"} {
		if !strings.Contains(raw(m), heading) {
			t.Errorf("no heading %q:\n%s", heading, raw(m))
		}
	}
	if got := table.FilterValues(&m, fieldIn("issues", "repository")); !slices.Equal(got, []string{"jig", "kit", grid.NoValue}) {
		t.Errorf("the repository's filter offers %q", got)
	}
	m.setPanel(m.panel(0).FilteredBy("repository", grid.Filter{Values: []string{"jig"}}))
	if n := table.Counted(m.every(0)); n != 1 {
		t.Errorf("a filter on jig lets %d rows through", n)
	}
	if got := m.records(m.stages[0])[1].values["organization"].Raw; !slices.Equal(got, []string{"empowerite"}) {
		t.Errorf("the organization of an issue underway: %q", got)
	}
}

// The workspace kittui reads can cross repositories, so every stage whose rows belong to one groups them, in each of
// its own views, by source, organization, and repository, the outermost first, a view the person saves keeping the
// groups they chose: repos, whose rows are repositories, by source and organization alone; remotes, whose
// organization and repository are those a remote names, by none of them. Audit policy's rows merge up to a scope, and
// a rule set for a whole organization stands under its repository's null, all, as its column draws it.
func TestEveryStageGroupsItsOwnViewsBySourceOrganizationAndRepository(t *testing.T) {
	want := []string{"source", "organization", "repository"}
	for name, s := range catalog {
		switch name {
		case "workspace/repos":
			want := want[:2]
			for _, pr := range s.Presets {
				if got := pr.Panel(s.Fields, grid.Panel{}).Groups(); !slices.Equal(got, want) {
					t.Errorf("%s %s groups by %q", name, pr.Name, got)
				}
			}
			continue
		case "workspace/remotes":
			for _, f := range s.Fields {
				if f.Placement == table.PlaceGroup {
					t.Errorf("%s groups by %s", name, f.ID)
				}
			}
			continue
		}
		for _, pr := range s.Presets {
			// issues' By… presets group by their field within the repository, at depth 4
			by, _ := strings.CutPrefix(strings.ToLower(pr.Name), "by ")
			// platform's tools are a checkout's, so they group by the worktree and its branch too
			if got := pr.Panel(s.Fields, grid.Panel{}).Groups(); !slices.Equal(got, want) &&
				!(name == issuesStage && slices.Equal(got, append(slices.Clone(want), by))) &&
				!(name == "platform/tools" && slices.Equal(got, append(slices.Clone(want), "worktree", "branch"))) {
				t.Errorf("%s %s groups by %q", name, pr.Name, got)
			}
		}
	}
	m := appOf(t, shell.Trust, tea.WindowSizeMsg{Width: 160, Height: 30})
	for m.stages[m.stage].name != "audit/policy" {
		m = drive(m, tab)
	}
	for _, heading := range []string{"▾ github · o 2", "▾ all 1", "▾ r 1"} {
		if !strings.Contains(raw(m), heading) {
			t.Errorf("policy has no heading %q:\n%s", heading, raw(m))
		}
	}
	// a clone with no remote on a forge has no source and no organization, "", which stand under none, not all
	m = drive(m, reanswer(m, "audit/policy", `{"rows": [{"key": "branch   r", "source": "", "organization": "",
		"repository": "r", "branch": null, "scope": "repository", "rule": "branch", "pattern": "^x-", "from": "default",
		"engine": "patterns", "links": {}}], "errors": []}`))
	if !strings.Contains(raw(m), "▾ none · none · r 1") || strings.Contains(raw(m), "▾ all") {
		t.Errorf("a clone with no remote's policy:\n%s", raw(m))
	}
}

// A ref names a thing, an issue's id or a run's number, and is no quantity to compare: its column is left-aligned, so
// its # stands in one column on every row. A time, a quantity, is right-aligned.
func TestARefIsLeftAlignedAndATimeRight(t *testing.T) {
	for name, s := range catalog {
		for _, f := range s.Fields {
			if f.Type == "ref" && f.Right || f.Type == "time" && !f.Right {
				t.Errorf("%s: %s, a %s, right-aligned %v", name, f.ID, f.Type, f.Right)
			}
		}
	}
}

// Every stage whose command reads a group's members has member, hidden, which groups and filters: over a group, the
// rows group by it.
func TestEveryStageHasAMemberField(t *testing.T) {
	n := 0
	for stage := range catalog {
		// the repos tab lists the members themselves
		if !takes(stage, "workspace") || stage == "workspace/repos" {
			continue
		}
		n++
		if f := fieldIn(stage, "member"); f.Placement != table.PlaceHidden || !f.Filter || !f.Group || !f.Sort {
			t.Errorf("%s.member: %+v", stage, f)
		}
	}
	if n != len(catalog)-1 {
		t.Errorf("%d stages of %d take workspace", n, len(catalog)-1)
	}
}

// A time and a number filter by ranges: issues' age offers a time's, and <1d keeps the issues of 2h and 3h, ≥30d the
// one of 40d.
func TestTheIssuesAgeFiltersByRanges(t *testing.T) {
	m := loaded(t, options(), tea.WindowSizeMsg{Width: 160, Height: 24})
	age := fieldIn("issues", "age")
	if !age.Filter || !table.Ranged(age) {
		t.Fatalf("issues' age: %+v", age)
	}
	if got := table.FilterValues(&m, age); !slices.Contains(got, "<1d") || !slices.Contains(got, "≥30d") {
		t.Errorf("the age's filter offers %q", got)
	}
	for token, want := range map[string]int{"<1d": 2, "≥30d": 1, "<1h": 0} {
		m.setPanel(m.panel(0).FilteredBy("age", grid.Filter{Values: []string{token}}))
		if n := table.Counted(m.every(0)); n != want {
			t.Errorf("%s lets %d rows through, not %d", token, n, want)
		}
	}
	for _, s := range catalog {
		for _, f := range s.Fields {
			if table.Ranged(f) && !f.Filter {
				t.Errorf("%s.%s, a %s, does not filter", s.Stage, f.ID, f.Type)
			}
		}
	}
}
