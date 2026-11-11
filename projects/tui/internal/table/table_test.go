// A table's parts on fields of their own: the catalog read, values, the renderers and their registry, the glyphs a
// field draws, times, labels, presets and back, and the automatic choice among a field's renderings.
package table

import (
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"github.com/empowerite/kit/projects/tui/internal/grid"
	"github.com/empowerite/kit/projects/tui/internal/style"
)

// state is an enum of two known values, an unknown one's glyph, and both a glyph and a plain rendering.
var state = Field{ID: "state", Label: "state", Type: "enum", Values: []EnumValue{{"open", "🟢"}, {"closed", "🔴"}},
	Other: "❔", Renderings: []string{"glyph+text", "text", "glyph"}, Sort: true, Group: true, Filter: true}

// title is a text that wraps, and dims.
var title = Field{ID: "title", Label: "title", Type: "text", Renderings: []string{"text"}, Wrap: true}

func TestACatalogReadsEveryTableByItsName(t *testing.T) {
	c, err := ReadCatalog([]byte(`{"stages": [{"stage": "a", "record": ".[]", "fields": [{"id": "x", "label": "x",
		"type": "text", "renderings": ["text"]}], "presets": [{"name": "Default"}]}]}`))
	if err != nil || c["a"] == nil || c["a"].Fields[0].ID != "x" || c["a"].Presets[0].Name != "Default" {
		t.Fatalf("the catalog: %v %+v", err, c)
	}
	if _, err := ReadCatalog([]byte(`{"stages": [{"stage": "a", "unknown": 1}]}`)); err == nil {
		t.Error("a key no table knows was read")
	}
}

func TestValuesHoldWhatARowSortsGroupsAndFiltersBy(t *testing.T) {
	secs := int64(-5)
	if !slices.Equal(One("x").Raw, []string{"x"}) || Some("").Raw != nil || Some("y").Raw[0] != "y" ||
		Seconds(nil).Raw != nil || Seconds(&secs).Raw[0] != "0" {
		t.Error("the values")
	}
	if !Types["enum"].Filters || Types["text"].Filters {
		t.Error("the types' filters")
	}
}

func TestAFieldDrawsItsGlyphs(t *testing.T) {
	if state.GlyphOf("open") != "🟢" || state.GlyphOf("merged") != "❔" || (Field{Glyph: "📦"}).GlyphOf("x") != "📦" {
		t.Error("the glyphs of values")
	}
	if got := state.Glyphed([]string{"open", "closed"}, true); !slices.Equal(got, []string{"🟢 open", "🔴 closed"}) {
		t.Errorf("glyph and text: %q", got)
	}
	if got := state.Glyphed([]string{"open"}, false); got[0] != "🟢" {
		t.Errorf("the glyph alone: %q", got)
	}
	if got := title.Glyphed([]string{"t"}, true); got[0] != "t" {
		t.Errorf("a text without a glyph: %q", got)
	}
	if state.DrawnBy("text") != "text" || state.DrawnBy("age") != "glyph+text" {
		t.Error("the rendering drawn by")
	}
}

func TestADrawingStylesWhatTheFieldSays(t *testing.T) {
	dim := state
	dim.Dim, dim.Right = true, true
	cells := dim.Draw([]Value{{Raw: []string{"open"}, Link: "https://x"}, {Else: "…"}}, "")
	if cells[0].Values[0] != style.Off("🟢 open") || !cells[0].Right || cells[0].Link != "https://x" || !cells[0].Hang {
		t.Errorf("a dim, right, linked value: %+v", cells[0])
	}
	if cells[1].Values[0] != style.Off("…") {
		t.Errorf("a field holding no value: %+v", cells[1])
	}
	// drawn without its glyph, which another of its renderings draws, the text is no link
	if c := state.Draw([]Value{{Raw: []string{"open"}, Link: "https://x"}}, "text"); c[0].Link != "" || c[0].Hang {
		t.Errorf("the text alone: %+v", c[0])
	}
	labels := Field{ID: "labels", Type: "label", Renderings: []string{"pills"}}
	c := labels.Draw([]Value{Labelled([]Label{{"bug", "#ff0000"}, {"ui", ""}})}, "")[0]
	if len(c.Values) != 2 || c.Head != c.Values[0] || ansi.Strip(c.Summary) != "2" {
		t.Errorf("labels: %+v", c)
	}
	// people, as labels: more than one is their glyph for many and how many, dim; one is drawn, its glyph its own
	people := Field{ID: "assignees", Type: "person", One: "assignee", Glyph: "👤", Many: "👥", Dim: true,
		Renderings: []string{"glyph+text", "text"}}
	cs := people.Draw([]Value{{Raw: []string{"ada", "bo"}}, One("ada")}, "")
	if cs[0].Summary != style.Off("👥 2") || len(cs[0].Values) != 2 || cs[0].Head != "" {
		t.Errorf("two people: %+v", cs[0])
	}
	if cs[1].Summary != "" || cs[1].Values[0] != style.Off("👤 ada") {
		t.Errorf("one person: %+v", cs[1])
	}
	// drawn without its glyph, the count stands alone
	if c := people.Draw([]Value{{Raw: []string{"ada", "bo"}}}, "text")[0]; c.Summary != style.Off("2") {
		t.Errorf("two people as text: %+v", c)
	}
	// a list whose values lead apart, as assets each to its download, hands each value's link to its cell, from the
	// glyph; drawn without it, none
	assets := Field{ID: "assets", Type: "text", One: "asset", Glyph: "📎", Many: "📎",
		Renderings: []string{"glyph+text", "text"}}
	v := Value{Raw: []string{"a", "b"}, Links: []string{"https://x/a", "https://x/b"}}
	if c := assets.Draw([]Value{v}, "")[0]; !slices.Equal(c.Links, v.Links) || c.Link != "" {
		t.Errorf("assets: %+v", c)
	}
	if c := assets.Draw([]Value{v}, "text")[0]; c.Links != nil {
		t.Errorf("assets as text: %+v", c)
	}
}

func TestAScreenRegistersItsOwnRenderers(t *testing.T) {
	Register("test-shout", Each(func(f Field, v Value) []string { return []string{strings.ToUpper(v.Raw[0])} }), true)
	if !Registered("test-shout") || !DrawsGlyph("test-shout") || DrawsGlyph("text") || Registered("nonesuch") {
		t.Fatal("the registry")
	}
	f := Field{ID: "x", Renderings: []string{"test-shout"}}
	if got := f.Draw([]Value{One("a"), {}}, ""); got[0].Values[0] != "A" || got[1].Values != nil {
		t.Errorf("the registered renderer: %+v", got)
	}
	defer func() {
		if recover() == nil {
			t.Error("a name registered twice did not panic")
		}
	}()
	Register("text", nil, false)
}

func TestTimesAreDrawnInTheirCoarsestUnit(t *testing.T) {
	for secs, want := range map[int64]string{-1: "0s", 45: "45s", 720: "12m", 10800: "3h", 4060800: "47d"} {
		if got := Age(secs); got != want {
			t.Errorf("Age(%d) = %q, want %q", secs, got, want)
		}
	}
	for _, c := range []struct {
		secs *int64
		want string
	}{{nil, "…"}, {ptr(45), "45s"}, {ptr(216), "3m 36s"}, {ptr(3720), "1h 02m"}} {
		if got := Took(c.secs); got != c.want {
			t.Errorf("Took = %q, want %q", got, c.want)
		}
	}
	when := Field{ID: "when", Renderings: []string{"age", "took"}}
	if got := when.Draw([]Value{One("720")}, "age"); got[0].Values[0] != "12m" {
		t.Errorf("drawn by age: %+v", got)
	}
	if got := when.Draw([]Value{One("216")}, "took"); got[0].Values[0] != "3m 36s" {
		t.Errorf("drawn by took: %+v", got)
	}
	if got := Mapped([]string{"45"}, Age); got[0] != "45s" {
		t.Errorf("mapped: %q", got)
	}
}

// ptr gives a pointer to n.
func ptr(n int64) *int64 { return &n }

func TestLabelsAreDrawnAsPills(t *testing.T) {
	if Labelled(nil).Raw != nil {
		t.Error("no labels hold a value")
	}
	v := Labelled([]Label{{"bug", "#ff0000"}})
	if !slices.Equal(v.Raw, []string{"bug"}) || LabelsIn(v)[0].Color != "#ff0000" {
		t.Errorf("a labelled value: %+v", v)
	}
	if got := LabelsIn(One("ui")); len(got) != 1 || got[0] != (Label{Name: "ui"}) {
		t.Errorf("a value of names alone: %+v", got)
	}
	if got := Pills(Field{}, v); len(got) != 1 || got[0] != Pill(Label{"bug", "#ff0000"}) {
		t.Errorf("the pills: %q", got)
	}
}

func TestAPresetSetsAPanelAndIsReadBackFromIt(t *testing.T) {
	note := Field{ID: "note", Label: "note", Type: "text", Renderings: []string{"text"}, Placement: PlaceHidden}
	lead := Field{ID: "lead", Type: "text", Renderings: []string{"text"}}
	fields := []Field{lead, state, title, note}
	// what a field cannot do it is not set to do: title neither sorts nor filters, and lead, without a label, never hides
	pr := Preset{Fields: map[string]Setting{
		"state": {Placement: PlaceGroup, Depth: 1, Sort: "desc", Priority: 1, Filter: &ValueFilter{Exclude: []string{"closed"}},
			Rendering: "text"},
		"title": {Sort: "asc", Priority: 2, Filter: &ValueFilter{Include: []string{"x"}}, Rendering: "glyph"},
		"lead":  {Placement: PlaceHidden},
	}}
	p := pr.Panel(fields, grid.Panel{})
	want := map[string]Setting{"state": {Placement: PlaceGroup, Depth: 1, Sort: "desc", Priority: 1,
		Filter: &ValueFilter{Exclude: []string{"closed"}}, Rendering: "text"}}
	if got := PresetOf(p, fields); !reflect.DeepEqual(got.Fields, want) {
		t.Errorf("read back: %+v", got.Fields)
	}
	// the columns in another order than the table's say each one's position; a field the catalog hides, shown, says so;
	// a filter of the values to hold, and a sort ascending
	pr = Preset{Fields: map[string]Setting{"title": {Position: 1}, "state": {Position: 2, Sort: "asc", Priority: 1,
		Filter: &ValueFilter{Include: []string{"open"}}}, "note": {Placement: PlaceColumn, Position: 3}}}
	want = map[string]Setting{"title": {Placement: PlaceColumn, Position: 1}, "state": {Placement: PlaceColumn,
		Position: 2, Sort: "asc", Priority: 1, Filter: &ValueFilter{Include: []string{"open"}}},
		"note": {Placement: PlaceColumn, Position: 3}}
	if got := PresetOf(pr.Panel(fields, grid.Panel{}), fields); !reflect.DeepEqual(got.Fields, want) {
		t.Errorf("columns moved: %+v", got.Fields)
	}
	// a field hidden that the catalog shows says so; none set says nothing
	if got := PresetOf((Preset{Fields: map[string]Setting{"title": {Placement: PlaceHidden}}}).Panel(fields, grid.Panel{}),
		fields); !reflect.DeepEqual(got.Fields, map[string]Setting{"title": {Placement: PlaceHidden}}) {
		t.Errorf("title hidden: %+v", got.Fields)
	}
	// a field the catalog hides, shown where it stands, says so without a position
	if got := PresetOf((Preset{Fields: map[string]Setting{"note": {Placement: PlaceColumn}}}).Panel(fields, grid.Panel{}),
		fields); !reflect.DeepEqual(got.Fields, map[string]Setting{"note": {Placement: PlaceColumn}}) {
		t.Errorf("note shown: %+v", got.Fields)
	}
	if got := PresetOf((Preset{}).Panel(fields, grid.Panel{}), fields); got.Fields != nil {
		t.Errorf("the catalog's own: %+v", got.Fields)
	}
	// a field that holds a list is flat as a preset says, and sorts while it is; one that holds none is never flat, and
	// a list gathered keeps no sort
	people := Field{ID: "who", Label: "who", One: "person", Type: "person", Renderings: []string{"text"}}
	fields, lists := []Field{people, title}, grid.Panel{Ones: map[string]string{"who": "person"}}
	pr = Preset{Fields: map[string]Setting{"who": {Flat: true, Sort: "desc", Priority: 1}, "title": {Flat: true}}}
	want = map[string]Setting{"who": {Flat: true, Sort: "desc", Priority: 1}}
	if got := PresetOf(pr.Panel(fields, lists), fields); !reflect.DeepEqual(got.Fields, want) {
		t.Errorf("who flat, sorting: %+v", got.Fields)
	}
	pr = Preset{Fields: map[string]Setting{"who": {Sort: "desc", Priority: 1}}}
	if got := PresetOf(pr.Panel(fields, lists), fields); got.Fields != nil {
		t.Errorf("who gathered, sorting: %+v", got.Fields)
	}
}

func TestTheRenderingIsChosenAutomatically(t *testing.T) {
	column := func(f Field) Placed { return Placed{Field: f, Column: true, Name: f.Label} }
	open := []Value{One("open"), One("open")}
	// pinned, a field is drawn as pinned
	pinned := column(state)
	pinned.Pinned = "glyph"
	if got := Automatic([]Placed{pinned}, map[string][]Value{"state": open}, 100, false); got["state"] != "glyph" {
		t.Errorf("pinned: %v", got)
	}
	// a glyph every row shares says nothing, unless the panel shows the emoji every row shares
	if got := Automatic([]Placed{column(state)}, map[string][]Value{"state": open}, 100, false); got["state"] != "text" {
		t.Errorf("a shared glyph: %v", got)
	}
	if got := Automatic([]Placed{column(state)}, map[string][]Value{"state": open}, 100, true); got["state"] != "glyph+text" {
		t.Errorf("a shared glyph, shown: %v", got)
	}
	// a glyph that leads somewhere is a button
	linked := []Value{{Raw: []string{"open"}, Link: "https://x"}, One("open")}
	if got := Automatic([]Placed{column(state)}, map[string][]Value{"state": linked}, 100, false); got["state"] != "glyph+text" {
		t.Errorf("a linked glyph: %v", got)
	}
	// short of width, a column steps down to a narrower rendering; the one that wraps never does
	mixed := []Value{One("open"), One("closed")}
	vs := map[string][]Value{"state": mixed, "title": {One("a title"), One("another")}}
	if got := Automatic([]Placed{column(state), column(title)}, vs, 100, false); got["state"] != "glyph+text" {
		t.Errorf("room enough: %v", got)
	}
	if got := Automatic([]Placed{column(state), column(title), {Field: note()}}, vs, 12, false); got["state"] != "glyph" ||
		got["title"] != "text" {
		t.Errorf("short of width: %v", got)
	}
	// a glyph a row alone draws says what the row is
	if state.SharesGlyph([]Value{One("open"), {}}) || !state.SharesGlyph(open) || state.SharesGlyph(mixed) {
		t.Error("shares glyph")
	}
	// a row of several people draws their glyph for many, so one of one and one of two share nothing
	people := Field{ID: "assignees", Type: "person", Glyph: "👤", Many: "👥"}
	ones := []Value{One("ada"), One("bo")}
	if !people.SharesGlyph(ones) || people.SharesGlyph(append(ones, Value{Raw: []string{"ada", "bo"}})) ||
		!people.SharesGlyph([]Value{{Raw: []string{"ada", "bo"}}, {Raw: []string{"cy", "di"}}}) {
		t.Error("people share a glyph")
	}
}

// note gives a field that is not a column.
func note() Field { return Field{ID: "note", Label: "note", Renderings: []string{"text"}} }
