// A json field: its value read whole at its path, and drawn compact on a closed row and pretty on an open one.
package table

import (
	"slices"
	"testing"
)

func TestAJSONFieldReadsItsValueWholeAsJqPrintsIt(t *testing.T) {
	f := Field{ID: "value", Type: "json", Path: ".value", Null: "none", Renderings: []string{"json"}}
	for _, c := range []struct {
		record   string
		raw      []string
		pretty   []string
		fallback string
	}{
		{`{"value":"bin/ci"}`, []string{"bin/ci"}, []string{"bin/ci"}, ""},
		{`{"value":14}`, []string{"14"}, []string{"14"}, ""},
		{`{"value":1e400}`, []string{"1e400"}, []string{"1e400"}, ""},
		{`{"value":true}`, []string{"true"}, []string{"true"}, ""},
		{`{"value":false}`, []string{"false"}, []string{"false"}, ""},
		{`{"value":[]}`, []string{"[]"}, []string{"[]"}, ""},
		{`{"value":{"b":["<x>"],"a":{}}}`, []string{`{"a":{},"b":["<x>"]}`},
			[]string{"{\n  \"a\": {},\n  \"b\": [\n    \"<x>\"\n  ]\n}"}, ""},
		{`{"value":null}`, nil, nil, "none"},
		{`{}`, nil, nil, ""},
	} {
		v := f.Read([]byte(c.record))
		pretty, _ := v.Of.([]string)
		if !slices.Equal(v.Raw, c.raw) || !slices.Equal(pretty, c.pretty) || v.Else != c.fallback {
			t.Errorf("%s: raw %q, pretty %q, else %q", c.record, v.Raw, pretty, v.Else)
		}
	}
}

func TestAJSONFieldDrawsCompactClosedAndPrettyOpen(t *testing.T) {
	f := Field{ID: "value", Type: "json", Path: ".value", Renderings: []string{"json"}}
	cells := f.Draw([]Value{
		f.Read([]byte(`{"value":{"read":["README.md"]}}`)), f.Read([]byte(`{"value":"bin/ci"}`)), {},
	}, "")
	object, scalar, none := cells[0], cells[1], cells[2]
	if object.Summary != `{"read":["README.md"]}` ||
		!slices.Equal(object.Values, []string{"{", `  "read": [`, `    "README.md"`, "  ]", "}"}) {
		t.Errorf("an object: summary %q, values %q", object.Summary, object.Values)
	}
	if scalar.Summary != "" || !slices.Equal(scalar.Values, []string{"bin/ci"}) ||
		!slices.Equal(scalar.Raw, []string{"bin/ci"}) {
		t.Errorf("a string: summary %q, values %q, raw %q", scalar.Summary, scalar.Values, scalar.Raw)
	}
	if len(none.Values) != 0 || none.Summary != "" {
		t.Errorf("no value: %q, %q", none.Values, none.Summary)
	}
	if !Registered("json") || Types["json"].Filters {
		t.Error("json is a registered renderer and a type no filter chooses among")
	}
}
