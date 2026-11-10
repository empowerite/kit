// A table's rows built from its records: what the catalog says of the fields, the rows laid out and let through, the
// renderings chosen for the width, the values shown, the columns offered, and a label found by its name.
package table

import (
	"slices"
	"testing"

	"github.com/empowerite/kit/projects/tui/internal/grid"
)

// list is a field of names that flattens, one name a row.
var list = Field{ID: "names", Label: "names", Type: "text", One: "name", Renderings: []string{"text"}, Sort: true}

// owner is a person field whose people the rows belong to.
var owner = Field{ID: "owner", Label: "owner", Type: "person", Owns: true, Renderings: []string{"text"}}

// gutter is a field without a label, which the header names not.
var gutter = Field{ID: "gutter", Type: "text", Renderings: []string{"text"}}

func TestASchemaSaysWhatItsFieldsCanDo(t *testing.T) {
	s := &Schema{Fields: []Field{gutter, state, list, owner, title}}
	if got := s.Columns(); !slices.Equal(got, []string{"", "state", "names", "owner", "title"}) {
		t.Errorf("the columns: %q", got)
	}
	if got := s.Labels(); len(got) != 4 || got["state"] != "state" || got["gutter"] != "" {
		t.Errorf("the labels: %v", got)
	}
	if got := s.Sortable(); !got["state"] || !got["names"] || got["title"] || len(got) != 2 {
		t.Errorf("the sortable: %v", got)
	}
	if got := s.Groupable(); !got["state"] || len(got) != 1 {
		t.Errorf("the groupable: %v", got)
	}
	if got := s.Ones(); got["names"] != "name" || len(got) != 1 {
		t.Errorf("the lists: %v", got)
	}
	if s.Owner() != "owner" || (&Schema{Fields: []Field{state}}).Owner() != "" {
		t.Error("the owner")
	}
	p := s.Ready(grid.Panel{Query: "x"})
	if p.Query != "x" || p.Owner != "owner" || !p.Sortable["state"] || !p.Groupable["state"] ||
		p.Ones["names"] != "name" || p.Labels["title"] != "title" {
		t.Errorf("the panel made ready: %+v", p)
	}
	// no table has no fields, and says nothing of them
	var none *Schema
	if none.Of() != nil || none.Columns() != nil || len(none.Labels()) != 0 || none.Labels() == nil ||
		none.Owner() != "" {
		t.Error("no table")
	}
}

// records gives three records of state, names, and title, and a heading of one cell, which holds no values.
func records() []Record {
	row := func(key string) grid.Row { return grid.Row{Key: key} }
	return []Record{
		{Row: grid.Row{Key: "head", Aside: true, Lone: true, Cells: []grid.Cell{{Values: []string{"a heading"}}}}},
		{Row: row("1"), Values: map[string]Value{"state": One("open"), "names": {Raw: []string{"ada", "bo"}},
			"title": One("the first title, long enough to wrap")}},
		{Row: row("2"), Values: map[string]Value{"state": One("closed"), "names": One("ada"), "title": One("second")}},
		{Row: row("3"), Values: map[string]Value{"state": One("open"), "title": One("third"),
			"tags": Labelled([]Label{{"bug", "#ff0000"}})}},
	}
}

func TestRecordsAreLaidOutInTheirFieldsAndBuilt(t *testing.T) {
	s := &Schema{Fields: []Field{state, list, title}, Presets: []Preset{{Name: Opening}}}
	rs := records()
	all := Lay(s.Fields, rs, func(id string) string { return "text" })
	if len(all) != 4 || len(all[0].Cells) != 1 || len(all[1].Cells) != 3 || all[1].Cells[0].Values[0] != "open" {
		t.Fatalf("laid out: %+v", all)
	}
	p := s.Ready(s.Presets[0].Panel(s.Fields, grid.Panel{}))
	b := Build(s, rs, p, []string{"state", "names", "title"}, 80)
	// the heading over the groups stands first, the rows' ▾ and count
	if len(b.Rows) != 5 || b.All != 3 || b.Levels != 0 || !slices.Equal(b.Names, []string{"state", "names", "title"}) {
		t.Errorf("built: %d rows, %d of them, %d levels, offered %q", len(b.Rows), b.All, b.Levels, b.Names)
	}
	if b.Drawn["state"] != "glyph+text" || b.Drawn["title"] != "text" {
		t.Errorf("drawn by: %v", b.Drawn)
	}
	// narrow, the state steps down to its glyph, and the search keeps the rows its text had
	p.Query = "open"
	b = Build(s, rs, p, []string{"state", "names", "title"}, 24)
	if b.Drawn["state"] == "glyph+text" || b.All != 3 || len(b.Rows) != 4 {
		t.Errorf("narrow and searched: drawn %v, %d rows", b.Drawn, len(b.Rows))
	}
	// grouped by the state, the rows stand a level in
	p.Query = ""
	b = Build(s, rs, p.Grouped("state", false), nil, 80)
	if b.Levels != 1 {
		t.Errorf("grouped: %d levels", b.Levels)
	}
}

func TestTheFieldsArePlacedAsThePanelPlacesThem(t *testing.T) {
	fields := []Field{gutter, state, list, title}
	columns := (&Schema{Fields: fields}).Columns()
	p := (Preset{Name: "x", Fields: map[string]Setting{"names": {Placement: PlaceHidden},
		"title": {Rendering: "text"}}}).Panel(fields, grid.Panel{})
	got := Placements(fields, columns, p, []string{"", "state"})
	var ids []string
	for _, f := range got {
		ids = append(ids, f.ID)
	}
	if !slices.Equal(ids, []string{"gutter", "state", "title", "names"}) || got[1].Name != "state" ||
		got[2].Name != "" || !got[2].Column || got[3].Column || got[2].Pinned != "text" {
		t.Errorf("placed: %+v", got)
	}
}

func TestTheValuesShownAreThoseOfTheRowsLetThrough(t *testing.T) {
	rs := records()
	p := (&Schema{Fields: []Field{state, list, title}}).Ready(grid.Panel{})
	rows := []grid.Row{{Key: "1"}, {Key: "3"}, {Key: "head", Aside: true}}
	got := ShownValues(rs, rows, p)
	if len(got["state"]) != 2 || len(got["names"]) != 1 || len(got["names"][0].Raw) != 2 || len(got["tags"]) != 1 {
		t.Errorf("the values: %v", got)
	}
	// flattened, a list stands once per value it holds
	p = p.Flattened("names", true)
	if got := ShownValues(rs, rows, p); len(got["names"]) != 2 || got["names"][1].Raw[0] != "bo" {
		t.Errorf("flattened: %v", got["names"])
	}
	laid := Lay([]Field{state, list}, rs, func(string) string { return "" })
	if got := Filled(laid, []string{"state", "names", "x"}); !slices.Equal(got, []string{"state", "names", ""}) {
		t.Errorf("the columns offered: %q", got)
	}
}

func TestALabelIsFoundByItsName(t *testing.T) {
	if l := LabelNamed(records(), "tags", "bug"); l.Color != "#ff0000" {
		t.Errorf("a label: %+v", l)
	}
	if l := LabelNamed(records(), "tags", "ui"); l.Name != "ui" || l.Color != "" {
		t.Errorf("a label no row holds: %+v", l)
	}
}
