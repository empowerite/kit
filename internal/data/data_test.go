package data

import (
	"errors"
	"reflect"
	"strings"
	"testing"
	"testing/fstest"
)

// thing is what the tests' schema describes: a name, and a count that must be a whole number of at least one.
type thing struct {
	Name  string `yaml:"name"`
	Count int    `yaml:"count"`
}

const thingSchema = `{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "type": "object",
  "properties": {
    "name": {"type": "string"},
    "count": {"type": "integer", "minimum": 1}
  },
  "required": ["name"],
  "additionalProperties": false
}`

// store gives a store of the tests' own files: the schema above, and a file of kit's own that holds to it.
func store() Store {
	return Store{fstest.MapFS{
		"files/thing.schema.json": {Data: []byte(thingSchema)},
		"files/thing.yml":         {Data: []byte("name: a thing\ncount: 2\n")},
		"files/notes.txt":         {Data: []byte("not a data file\n")},
	}}
}

func TestDecodeGivesTheValue(t *testing.T) {
	var got thing
	if err := store().Decode(Source{"x.yml", []byte("name: a\ncount: 3\n")}, "thing", &got); err != nil {
		t.Fatalf("Decode gave the error %v", err)
	}
	if want := (thing{"a", 3}); got != want {
		t.Errorf("Decode gave %+v; want %+v", got, want)
	}
}

func TestDecodeTakesJSON(t *testing.T) {
	var got thing
	if err := store().Decode(Source{"x.json", []byte(`{"name": "a", "count": 3}`)}, "thing", &got); err != nil {
		t.Fatalf("Decode gave the error %v", err)
	}
	if want := (thing{"a", 3}); got != want {
		t.Errorf("Decode gave %+v; want %+v", got, want)
	}
}

func TestDecodeNamesTheLineOfAValueTheSchemaRefuses(t *testing.T) {
	for _, c := range []struct{ name, text, want string }{
		{"a value of the wrong kind", "# a thing\nname: a\ncount: many\n", "x.yml:3: count: "},
		{"a value under the minimum", "name: a\n\ncount: 0\n", "x.yml:3: count: "},
		{"a key the schema does not know", "name: a\nsize: 3\n", "x.yml:2: additional properties 'size'"},
		{"a key the schema requires, missing", "count: 3\n", "x.yml:1: "},
	} {
		var got thing
		err := store().Decode(Source{"x.yml", []byte(c.text)}, "thing", &got)
		var at *Error
		if !errors.As(err, &at) || !strings.HasPrefix(err.Error(), c.want) {
			t.Errorf("%s: Decode gave the error %v; want an *Error that begins %q", c.name, err, c.want)
		}
	}
}

func TestDecodeNamesTheLineOfYAMLThatDoesNotParse(t *testing.T) {
	var got thing
	err := store().Decode(Source{"x.yml", []byte("name: a\ncount: [1, 2\nother: 3\n")}, "thing", &got)
	var at *Error
	if !errors.As(err, &at) || at.File != "x.yml" || at.Line != 3 {
		t.Errorf("Decode gave the error %v; want an *Error at x.yml:3", err)
	}
}

func TestDecodeRefusesAnEmptyFile(t *testing.T) {
	var got thing
	err := store().Decode(Source{"x.yml", nil}, "thing", &got)
	if err == nil || err.Error() != "x.yml:1: the file is empty" {
		t.Errorf("Decode gave the error %v; want x.yml:1: the file is empty", err)
	}
}

func TestDecodeRefusesASchemaItDoesNotHold(t *testing.T) {
	var got thing
	err := store().Decode(Source{"x.yml", []byte("name: a\n")}, "other", &got)
	if err == nil || !strings.Contains(err.Error(), "no schema named other") {
		t.Errorf("Decode gave the error %v; want one that says there is no schema named other", err)
	}
}

func TestDecodeRefusesASchemaThatDoesNotCompile(t *testing.T) {
	s := Store{fstest.MapFS{"files/bad.schema.json": {Data: []byte(`{"type": 7}`)}}}
	var got thing
	if err := s.Decode(Source{"x.yml", []byte("name: a\n")}, "bad", &got); err == nil {
		t.Error("Decode gave no error for a schema that does not compile")
	}
	s = Store{fstest.MapFS{"files/bad.schema.json": {Data: []byte(`{`)}}}
	if err := s.Decode(Source{"x.yml", []byte("name: a\n")}, "bad", &got); err == nil {
		t.Error("Decode gave no error for a schema that is not JSON")
	}
}

func TestDecodeRefusesAValueGoCannotHold(t *testing.T) {
	// The schema lets a count be any whole number; Go's int8 cannot hold 300.
	var got struct {
		Name  string `yaml:"name"`
		Count int8   `yaml:"count"`
	}
	err := store().Decode(Source{"x.yml", []byte("name: a\ncount: 300\n")}, "thing", &got)
	if err == nil || !strings.HasPrefix(err.Error(), "x.yml:") {
		t.Errorf("Decode gave the error %v; want one that names x.yml", err)
	}
}

func TestDecodeRefusesYAMLThatJSONCannotHold(t *testing.T) {
	// A key that is a list has no form in JSON, so no schema can judge it.
	var got thing
	err := store().Decode(Source{"x.yml", []byte("? [a, b]\n: 1\n")}, "thing", &got)
	if err == nil || !strings.HasPrefix(err.Error(), "x.yml:1: ") {
		t.Errorf("Decode gave the error %v; want one at x.yml:1", err)
	}
}

func TestDecodeRefusesANumberJSONCannotHold(t *testing.T) {
	var got thing
	err := store().Decode(Source{"x.yml", []byte("name: a\ncount: .inf\n")}, "thing", &got)
	if err == nil || !strings.HasPrefix(err.Error(), "x.yml:1: it holds what JSON cannot: ") {
		t.Errorf("Decode gave the error %v; want one that says JSON cannot hold it", err)
	}
}

func TestDecodeNamesTheLineOfAnItemOfAList(t *testing.T) {
	s := Store{fstest.MapFS{"files/list.schema.json": {Data: []byte(`{"type": "object",
		"properties": {"tags": {"type": "array", "items": {"type": "string"}}}}`)}}}
	var got any
	err := s.Decode(Source{"x.yml", []byte("tags:\n  - a\n  - 3\n")}, "list", &got)
	if err == nil || !strings.HasPrefix(err.Error(), "x.yml:3: tags.1: ") {
		t.Errorf("Decode gave the error %v; want one at x.yml:3 that names tags.1", err)
	}
}

func TestDecodeNamesTheNearestLineOfAValueAMergeBrought(t *testing.T) {
	// A merge key brings count in from elsewhere, so the top of the file holds no key of that name: the line given
	// is the nearest node that the value's location reaches.
	s := Store{fstest.MapFS{"files/open.schema.json": {Data: []byte(`{"type": "object",
		"properties": {"count": {"type": "integer", "minimum": 1}}}`)}}}
	var got any
	err := s.Decode(Source{"x.yml", []byte("defaults: &d\n  count: 0\n<<: *d\n")}, "open", &got)
	if err == nil || !strings.HasPrefix(err.Error(), "x.yml:1: count: ") {
		t.Errorf("Decode gave the error %v; want one at x.yml:1 that names count", err)
	}
}

func TestDecodeSaysWhatYAMLSaysWhereItNamesNoLine(t *testing.T) {
	// yaml names no line for a character it does not allow in a file at all: the error is put at the first.
	var got thing
	err := store().Decode(Source{"x.yml", []byte("name: a\x01\n")}, "thing", &got)
	if err == nil || !strings.HasPrefix(err.Error(), "x.yml:1: ") {
		t.Errorf("Decode gave the error %v; want one at x.yml:1", err)
	}
}

func TestShowGivesKitsOwnFile(t *testing.T) {
	got, err := store().Show("thing")
	if err != nil || string(got) != "name: a thing\ncount: 2\n" {
		t.Errorf("Show(thing) = %q, %v; want the file's text", got, err)
	}
}

func TestShowRefusesANameItDoesNotHoldAndListsTheNames(t *testing.T) {
	_, err := store().Show("other")
	if err == nil || err.Error() != "data: no file named other: the files are thing" {
		t.Errorf("Show(other) gave the error %v; want one that lists thing", err)
	}
}

func TestNamesGivesKitsOwnFilesSorted(t *testing.T) {
	s := store()
	s.files.(fstest.MapFS)["files/another.yml"] = &fstest.MapFile{Data: []byte("x: 1\n")}
	if got, want := s.Names(), []string{"another", "thing"}; !reflect.DeepEqual(got, want) {
		t.Errorf("Names = %q; want %q", got, want)
	}
}

func TestOverLetsTheLaterLayerWin(t *testing.T) {
	base := map[string]any{"a": 1, "b": map[string]any{"x": 1}}
	over := map[string]any{"b": map[string]any{"y": 2}, "c": 3}
	got := Over(base, over)
	want := map[string]any{"a": 1, "b": map[string]any{"y": 2}, "c": 3}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Over = %v; want %v", got, want)
	}
	if !reflect.DeepEqual(base, map[string]any{"a": 1, "b": map[string]any{"x": 1}}) {
		t.Errorf("Over changed its base: %v", base)
	}
}

func TestEveryDefaultHoldsToItsSchema(t *testing.T) {
	// Each of kit's own files is checked against its schema of the same name, as kit checks it when it loads it.
	for _, name := range Names() {
		text, err := Show(name)
		if err != nil {
			t.Fatalf("Show(%s) gave the error %v", name, err)
		}
		var value any
		if err := Decode(Source{name + ".yml", text}, name, &value); err != nil {
			t.Errorf("%s.yml does not hold to %s.schema.json: %v", name, name, err)
		}
	}
	if len(Names()) == 0 {
		t.Error("kit holds no data files of its own")
	}
}
