// A json field: its value read whole at its path, and drawn compact on a closed row and pretty on an open one.
package table

import (
	"slices"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"github.com/empowerite/kit/projects/tui/internal/style"
)

func TestAJSONFieldReadsItsValueWholeAsJqPrintsIt(t *testing.T) {
	f := Field{ID: "value", Type: "json", Path: ".value", Null: "none", Empty: "empty", Renderings: []string{"json"}}
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
		{`{"value":""}`, nil, nil, "empty"},
		{`{}`, nil, nil, ""},
	} {
		v := f.Read([]byte(c.record))
		var pretty []string
		texts, _ := v.Of.([]jsonText)
		for _, p := range texts {
			pretty = append(pretty, p.Pretty)
		}
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
	plain := func(lines []string) []string {
		var out []string
		for _, l := range lines {
			out = append(out, ansi.Strip(l))
		}
		return out
	}
	if ansi.Strip(object.Summary) != `{"read":["README.md"]}` ||
		!slices.Equal(plain(object.Values), []string{"{", `  "read": [`, `    "README.md"`, "  ]", "}"}) {
		t.Errorf("an object: summary %q, values %q", object.Summary, object.Values)
	}
	if scalar.Summary != "" || !slices.Equal(plain(scalar.Values), []string{"bin/ci"}) ||
		!slices.Equal(scalar.Raw, []string{"bin/ci"}) {
		t.Errorf("a string: summary %q, values %q, raw %q", scalar.Summary, scalar.Values, scalar.Raw)
	}
	if len(none.Values) != 0 || none.Summary != "" {
		t.Errorf("no value: %q, %q", none.Values, none.Summary)
	}
	// a value a screen made, with no pretty form beside its raw one, draws its raw value as it is
	if made := f.Draw([]Value{One("bin/ci")}, "")[0]; !slices.Equal(plain(made.Values), []string{"bin/ci"}) || made.Summary != "" {
		t.Errorf("a value without its pretty form: %q, %q", made.Values, made.Summary)
	}
	if !Registered("json") || Types["json"].Filters {
		t.Error("json is a registered renderer and a type no filter chooses among")
	}
}

// A value draws colored so it can be scanned: its keys in the accent color, its strings green, its numbers yellow, its
// true and false magenta, and its null, its marks, and so an empty array and object, dim; a string alone draws bare,
// green, and every line of a pretty-printed one indents the same.
func TestAJSONFieldDrawsEachKindOfValueInItsOwnColor(t *testing.T) {
	f := Field{ID: "value", Type: "json", Path: ".value", Renderings: []string{"json"}}
	cell := f.Draw([]Value{f.Read([]byte(
		`{"value":{"on":true,"off":false,"n":-1.5e3,"s":"a \"q\": b","none":null,"list":[],"map":{},"bin":["x"]}}`))}, "")[0]
	want := []string{
		style.Off("{"),
		"  " + style.Tint(`"bin"`) + style.Off(":") + " " + style.Off("["),
		"    " + style.Text(`"x"`),
		"  " + style.Off("],"),
		"  " + style.Tint(`"list"`) + style.Off(":") + " " + style.Off("[],"),
		"  " + style.Tint(`"map"`) + style.Off(":") + " " + style.Off("{},"),
		"  " + style.Tint(`"n"`) + style.Off(":") + " " + style.Warn("-1.5e3") + style.Off(","),
		"  " + style.Tint(`"none"`) + style.Off(":") + " " + style.Off("null") + style.Off(","),
		"  " + style.Tint(`"off"`) + style.Off(":") + " " + style.Literal("false") + style.Off(","),
		"  " + style.Tint(`"on"`) + style.Off(":") + " " + style.Literal("true") + style.Off(","),
		"  " + style.Tint(`"s"`) + style.Off(":") + " " + style.Text(`"a \"q\": b"`),
		style.Off("}"),
	}
	if !slices.Equal(cell.Values, want) {
		t.Errorf("an object drew\n%s\nwant\n%s", strings.Join(cell.Values, "\n"), strings.Join(want, "\n"))
	}
	if ansi.Strip(cell.Summary) == "" || cell.Summary == ansi.Strip(cell.Summary) {
		t.Errorf("a closed row's compact value is not colored: %q", cell.Summary)
	}
	bare := f.Draw([]Value{f.Read([]byte(`{"value":"line\n\nlast"}`))}, "")[0]
	if !slices.Equal(bare.Values, []string{style.Text("line"), "", style.Text("last")}) {
		t.Errorf("a string draws %q", bare.Values)
	}
	// a line a pretty form cut inside a string still draws, as one string
	if got := colorJSON(`"cut`, false); got != style.Text(`"cut`) {
		t.Errorf("a string cut off draws %q", got)
	}
}
