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
	"github.com/empowerite/kit/projects/tui/internal/table"
)

func TestTheCatalogNamesOnlyWhatIsRegistered(t *testing.T) {
	// every stage, incoming among them, kit workspace's, its repos tab among them, and kit packages'
	if len(catalog) != len(eight)+len(workspaceStages)+1 || catalog["incoming"] == nil ||
		catalog["workspace/repos"] == nil || catalog[packagesStage] == nil {
		t.Errorf("the catalog's stages: %d, incoming %v", len(catalog), catalog["incoming"] != nil)
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
		if !slices.Equal(slices.Sorted(maps.Keys(presets)), []string{"Compact", Opening}) {
			t.Errorf("%s: the presets %v", name, presets)
		}
	}
}

// The catalog draws a pull request's state, a build's, and a release's kind as the legend does: each known value with
// the glyph the legend gives it, and Other, or a release's own, for one it does not know.
func TestTheCatalogsGlyphsAreTheLegends(t *testing.T) {
	drawn := func(stage, id string, raw ...string) []string {
		k := slices.IndexFunc(fieldsOf(stage), func(f table.Field) bool { return f.ID == id })
		return fieldsOf(stage)[k].Glyphed(raw, false)
	}
	for _, stage := range []string{"underway", "desk", "worktrees", "changes"} {
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
	fields := fieldsOf("backlog")
	p := table.Preset{Name: "Triage", Fields: map[string]table.Setting{
		"title":     {Placement: table.PlaceColumn, Position: 1},
		"id":        {Placement: table.PlaceColumn, Position: 2},
		"source":    {Placement: table.PlaceGroup, Depth: 1},
		"branch":    {Placement: table.PlaceHidden, Rendering: "text"},
		"age":       {Sort: "desc", Priority: 1},
		"assignees": {Flat: true, Sort: "asc", Priority: 2, Filter: &table.ValueFilter{Exclude: []string{caller}}},
	}}
	g := p.Panel(fields, grid.Panel{Owner: ownerOf("backlog"), Ones: onesOf("backlog")})
	if got := g.Arrange(columnsOf("backlog")); !slices.Equal(got, []string{"title", "id", "member", "source",
		"organization", "repository", "assignees", "age", "branch", "labels"}) ||
		!slices.Equal(g.Groups(), []string{"source"}) || g.Field("branch").Placement != grid.Hidden ||
		!slices.Equal(g.SortKeys(), []grid.SortKey{{Column: "age", Desc: true}, {Column: "assignees"}}) ||
		g.Showing() != grid.Others || g.Field("branch").Rendering != "text" {
		t.Errorf("the panel: %q %+v", got, g.Fields)
	}
	back := table.PresetOf(g, fields)
	if !reflect.DeepEqual(table.PresetOf(back.Panel(fields, grid.Panel{}), fields), back) ||
		back.Fields["source"] != (table.Setting{Placement: table.PlaceGroup, Depth: 1}) || back.Fields["title"].Position != 1 {
		t.Errorf("the preset again: %+v", back)
	}
	// the stage's own: Default shows every column but member, which only a group's rows hold; Compact, the few that say
	// what a row is
	builds := catalog["builds"]
	if d := builds.Presets[0].Panel(builds.Fields, grid.Panel{}); builds.Presets[0].Name != Opening ||
		!slices.Equal(hiddenOf(grid.Panel{Fields: d.Fields, Columns: columnsOf("builds")}), []string{"member"}) {
		t.Errorf("Default: %+v", d.Fields)
	}
	if c := builds.Presets[1].Panel(builds.Fields, grid.Panel{Columns: columnsOf("builds")}); builds.Presets[1].Name != "Compact" ||
		!slices.Equal(hiddenOf(c), []string{"member", "run", "event", "branch", "commit", "took"}) {
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

// An issue's organization and repository, as kit prints them, are fields of backlog and of underway, hidden until the
// person shows them; shown, they draw each row's, and a filter on them offers the values the rows hold.
func TestAnIssuesOrganizationAndRepositoryAreFields(t *testing.T) {
	m := loaded(t, options(), tea.WindowSizeMsg{Width: 160, Height: 24})
	for _, stage := range []string{"backlog", "underway"} {
		for _, id := range []string{"organization", "repository"} {
			if f := fieldIn(stage, id); f.Placement != table.PlaceHidden || !f.Filter || !f.Group {
				t.Errorf("%s.%s: %+v", stage, id, f)
			}
		}
	}
	if strings.Contains(raw(m), "jig") {
		t.Errorf("the repository is drawn while hidden:\n%s", raw(m))
	}
	m.setPanel(m.panel(0).Placed("repository", grid.Column))
	if !strings.Contains(raw(m), "jig") || !strings.Contains(raw(m), "kit") {
		t.Errorf("the repository shown:\n%s", raw(m))
	}
	if got := table.FilterValues(&m, fieldIn("backlog", "repository")); !slices.Equal(got, []string{"jig", "kit", grid.NoValue}) {
		t.Errorf("the repository's filter offers %q", got)
	}
	m.setPanel(m.panel(0).FilteredBy("repository", grid.Filter{Values: []string{"jig"}}))
	if n := table.Counted(m.every(0)); n != 1 {
		t.Errorf("a filter on jig lets %d rows through", n)
	}
	m = drive(m, tab)
	if got := m.records(m.stages[m.stage])[0].values["organization"].Raw; !slices.Equal(got, []string{"empowerite"}) {
		t.Errorf("underway's organization: %q", got)
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

// Every stage drawn from records has member, hidden, which groups and filters: over a group, the rows group by it.
func TestEveryStageHasAMemberField(t *testing.T) {
	// the repos tab lists the members themselves
	for _, stage := range append([]string{"backlog", "underway", "desk", "worktrees", "changes", "builds", "releases"},
		workspaceStages[1:]...) {
		if f := fieldIn(stage, "member"); f.Placement != table.PlaceHidden || !f.Filter || !f.Group || !f.Sort {
			t.Errorf("%s.member: %+v", stage, f)
		}
	}
}

// A time and a number filter by ranges: the backlog's age offers a time's, and <1d keeps the issue of 2h, ≥30d the one
// of 40d.
func TestTheBacklogsAgeFiltersByRanges(t *testing.T) {
	m := loaded(t, options(), tea.WindowSizeMsg{Width: 160, Height: 24})
	age := fieldIn("backlog", "age")
	if !age.Filter || !table.Ranged(age) {
		t.Fatalf("the backlog's age: %+v", age)
	}
	if got := table.FilterValues(&m, age); !slices.Contains(got, "<1d") || !slices.Contains(got, "≥30d") {
		t.Errorf("the age's filter offers %q", got)
	}
	for token, want := range map[string]int{"<1d": 1, "≥30d": 1, "<1h": 0} {
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
