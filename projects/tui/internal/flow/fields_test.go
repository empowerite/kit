// The field catalog holds together: each stage's ids are its own, every type and renderer it names is registered, its
// presets name only its fields, and its fields stand in the order the stage's rows give their cells.
package flow

import (
	"maps"
	"slices"
	"testing"
)

func TestTheCatalogNamesOnlyWhatIsRegistered(t *testing.T) {
	if len(catalog) != len(eight)-1 || catalog["incoming"] != nil {
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
			kind, ok := types[f.Type]
			if !ok {
				t.Errorf("%s.%s: the type %q is not registered", name, f.ID, f.Type)
			}
			if f.Filter != kind.filters {
				t.Errorf("%s.%s: filter is %v for a %s", name, f.ID, f.Filter, f.Type)
			}
			if f.Path == "" {
				t.Errorf("%s.%s: no path", name, f.ID)
			}
			if len(f.Renderings) == 0 {
				t.Errorf("%s.%s: no rendering", name, f.ID)
			}
			for _, r := range f.Renderings {
				if renderers[r] == nil {
					t.Errorf("%s.%s: the renderer %q is not registered", name, f.ID, r)
				}
			}
			if !slices.Contains([]string{placeColumn, placeGroup, placeHidden}, f.Placement) {
				t.Errorf("%s.%s: the placement %q", name, f.ID, f.Placement)
			}
			if (f.Type == "enum") != (len(f.Values) > 0) {
				t.Errorf("%s.%s: a %s with %d values", name, f.ID, f.Type, len(f.Values))
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
				k := slices.IndexFunc(s.Fields, func(f field) bool { return f.ID == id })
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

// valueGlyphs gives the glyphs of an enum's known values.
func valueGlyphs(f field) []string {
	var out []string
	for _, v := range f.Values {
		out = append(out, v.Glyph)
	}
	return out
}

// checkSetting fails on what a preset cannot say of a field: a placement it does not know, or a column's, a group's, or
// a sort's place that is not from 1; a group or a sort the field cannot do, or a filter it cannot take, with values
// that are none of the field's; a rendering not the field's.
func checkSetting(t *testing.T, where string, f field, s setting) {
	t.Helper()
	switch s.Placement {
	case "", placeHidden:
	case placeColumn:
		if s.Position < 0 {
			t.Errorf("%s: the position %d", where, s.Position)
		}
	case placeGroup:
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
			known := slices.ContainsFunc(f.Values, func(e enumValue) bool { return e.Value == v })
			// a person's filter names logins, @me among them, and none
			if !known && (f.Type != "person" || v == "") {
				t.Errorf("%s: the filter's value %q", where, v)
			}
		}
	}
	if s.Rendering != "" && !slices.Contains(f.Renderings, s.Rendering) {
		t.Errorf("%s: the rendering %q is not the field's", where, s.Rendering)
	}
}

// Each stage's rows give their cells in the order of its fields, one a field.
func TestTheFieldsStandInTheOrderTheRowsGiveThem(t *testing.T) {
	m := loaded(t, options())
	for i, s := range m.stages {
		fields := fieldsOf(s.name)
		if fields == nil {
			continue
		}
		rows := m.unfiltered(i)
		if len(rows) == 0 {
			t.Errorf("%s: no rows to check", s.name)
		}
		for _, r := range rows {
			if len(r.Cells) > 1 && len(r.Cells) != len(fields) {
				t.Errorf("%s: the row %s has %d cells, the stage %d fields", s.name, r.Key, len(r.Cells), len(fields))
			}
		}
	}
}

// A preset says, per field, its placement, its sort, and its filter; the panel keeps them as a view does.
func TestAPresetIsKeptAsAView(t *testing.T) {
	fields := fieldsOf("backlog")
	yes := true
	p := preset{Name: "Triage", Headers: &yes, Fields: map[string]setting{
		"title":     {Placement: placeColumn, Position: 1},
		"id":        {Placement: placeColumn, Position: 2},
		"source":    {Placement: placeGroup, Depth: 1},
		"branch":    {Placement: placeHidden},
		"age":       {Sort: "desc", Priority: 1},
		"assignees": {Sort: "asc", Priority: 2, Filter: &valueFilter{Exclude: []string{caller}}},
	}}
	v := p.view(fields, false)
	if v.Name != "Triage" || !v.Headers || !slices.Equal(v.Order, []string{"title", "id"}) ||
		!slices.Equal(v.Group, []string{"source"}) || !slices.Equal(v.Hidden, []string{"branch"}) ||
		!slices.Equal(v.Sort, []sortKey{{"age", true}, {"assignees", false}}) || v.Show != "others" {
		t.Errorf("the view: %+v", v)
	}
	// the stage's own: Default shows every column, with the headers as the screen opens; Compact, the few that say
	// what a row is
	builds := catalog["builds"]
	if d := builds.Presets[0].view(builds.Fields, false); d.Name != Opening || d.Headers || d.Hidden != nil {
		t.Errorf("Default: %+v", d)
	}
	if c := builds.Presets[1].view(builds.Fields, false); c.Name != "Compact" || !c.Headers ||
		!slices.Equal(c.Hidden, []string{"run", "event", "branch", "commit", "took"}) {
		t.Errorf("Compact: %+v", c)
	}
}
