// A table read from what a command prints: its records, each field's path, a record's key, and the catalog refusing a
// schema a table cannot index by.
package table

import (
	"encoding/json"
	"errors"
	"slices"
	"strings"
	"testing"
)

func TestRecordsReadsEachShapeFreshAndRefusesTheRest(t *testing.T) {
	for _, c := range []struct {
		data, record string
		slurp        bool
		n            int
		printed      string
	}{
		{`[{"n":1},{"n":2}]`, ".[]", false, 2, ""}, {`[]`, ".[]", false, 0, ""}, {`{"n":1}`, ".", false, 1, ""},
		{`{"rows":[{"n":1}],"errors":[]}`, ".rows[]", false, 1, ""}, {"{\"n\":1}\n{\"n\":2}\n", ".[]", true, 2, ""},
		{"", ".[]", true, 0, ""}, {"", ".[]", false, 0, "nothing"}, {"null", ".", false, 0, "null"},
		{`{"n":1}`, ".[]", false, 0, "an object"}, {`[]`, ".", false, 0, "an array"},
		{`{"errors":[]}`, ".rows[]", false, 0, `an object without "rows"`}, {`{"r":null}`, ".r[]", false, 0, `null "r"`},
		{`[1]`, ".r[]", false, 0, "an array"}, {`"x"`, ".", false, 0, "a string"}, {` 3 `, ".[]", false, 0, "a number"},
		{`{"r":{}}`, ".r[]", false, 0, `an object "r"`}, {"null", ".[]", false, 0, "null"},
		{"true", ".", false, 0, "a boolean"},
	} {
		got, err := Records([]byte(c.data), c.record, c.slurp)
		var shape *ShapeError
		switch {
		case c.printed == "" && (err != nil || len(got) != c.n):
			t.Errorf("%q by %s: %d records, %v", c.data, c.record, len(got), err)
		case c.printed != "" && (!errors.As(err, &shape) || shape.Printed != c.printed || got != nil):
			t.Errorf("%q by %s: %v, %d records; want printed %s", c.data, c.record, err, len(got), c.printed)
		}
	}
	// each call's records are its own: changing one leaves the data, and the next call's, as they were
	data := []byte(`[{"n":1}]`)
	first, _ := Records(data, ".[]", false)
	first[0][2] = 'X'
	if again, _ := Records(data, ".[]", false); string(again[0]) != `{"n":1}` || string(data) != `[{"n":1}]` {
		t.Errorf("records share their bytes: %s, %s", again[0], data)
	}
	err := (&ShapeError{Printed: "nothing", Wanted: `an object with "rows"`}).Error()
	if err != `printed nothing where an object with "rows" was expected` {
		t.Errorf("the shape's error: %q", err)
	}
	for _, bad := range []struct{ data, record string }{{`{`, "."}, {`[1] [2]`, ".[]"}, {"{\n", ".[]"}} {
		var shape *ShapeError
		if _, err := Records([]byte(bad.data), bad.record, bad.data == "{\n"); err == nil || errors.As(err, &shape) {
			t.Errorf("%q: %v, a shape's error or none", bad.data, err)
		}
	}
}

func TestAFieldReadsItsPath(t *testing.T) {
	rec := json.RawMessage(`{"a":{"b":"x"},"n":3,"t":true,"f":false,"z":null,"l":[{"name":"p"},{"name":"q"}],` +
		`"u":[1,2],"big":12345678901234567890,"ll":[[1],[2,"w"]]}`)
	for path, want := range map[string][]string{".a.b": {"x"}, ".n": {"3"}, ".t": {"id"}, ".f": nil, ".z": nil,
		".nope": nil, ".a": nil, ".l[].name": {"p", "q"}, ".u": {"1", "2"}, ".u[]": {"1", "2"}, ".n, .a.b": {"3", "x"},
		".big": {"12345678901234567890"}, ".ll": {"1", "2", "w"}, ".n.deeper": nil, "": nil, ".": nil, ".a..b": nil} {
		if got := (Field{ID: "id", Path: path}).Read(rec).Raw; !slices.Equal(got, want) {
			t.Errorf("%q: %q, want %q", path, got, want)
		}
	}
	if got := (Field{ID: "id", Path: ".a"}).Read(json.RawMessage(`not json`)).Raw; got != nil {
		t.Errorf("a record that does not decode: %q", got)
	}
	for _, c := range []struct {
		schema Schema
		rec    string
		key    string
		ok     bool
	}{
		{Schema{Key: ".n, .a.b"}, string(rec), "3 x", true}, {Schema{}, `{"key":"v"}`, "v", true},
		{Schema{Key: ".z"}, string(rec), "", false}, {Schema{Key: ".n, .nope"}, string(rec), "", false},
		{Schema{}, `{"k":"v"}`, "", false}, {Schema{Key: ".t"}, string(rec), "true", true},
		{Schema{Key: ".l"}, string(rec), "", false}, {Schema{Key: ".a"}, string(rec), "", false},
		{Schema{Key: ".l[].name"}, string(rec), "p q", true}, {Schema{Key: ".u"}, string(rec), "1 2", true},
		{Schema{}, `[1]`, "", false},
	} {
		if key, ok := c.schema.KeyOf(json.RawMessage(c.rec)); key != c.key || ok != c.ok {
			t.Errorf("the key %q of %s: %q %v, want %q %v", c.schema.Key, c.rec, key, ok, c.key, c.ok)
		}
	}
}

func TestTheCatalogRefusesWhatATableCannotIndexBy(t *testing.T) {
	field := `{"id": "x", "label": "x", "type": "text", "renderings": ["text"]}`
	stage := func(name, extra, fields, presets string) string {
		return `{"stage": "` + name + `"` + extra + `, "fields": [` + fields + `], "presets": [` + presets + `]}`
	}
	ok := stage("a", `, "record": ".rows[]", "command": ["flow", "a", "--json"], "takes": ["owner", "no-fetch",
		"fresh", "cached", "workspace"], "staged": true, "forge": true, "slurp": true, "key": ".x",
		"empty": "no a"`, field,
		`{"name": "Default"}`)
	c, err := ReadCatalog([]byte(`{"stages": [` + ok + `]}`))
	if a := c["a"]; err != nil || !slices.Equal(a.Command, []string{"flow", "a", "--json"}) || len(a.Takes) != 5 ||
		!a.Staged || !a.Forge || !a.Slurp || a.Key != ".x" || a.Empty != "no a" {
		t.Fatalf("the schema's keys: %v %+v", err, c["a"])
	}
	// a schema without a command, as a screen's own, names its record as it likes
	if _, err := ReadCatalog([]byte(`{"stages": [` + stage("b", `, "record": "## <part>"`, field,
		`{"name": "Default"}`) + `]}`)); err != nil {
		t.Errorf("a record without a command: %v", err)
	}
	def := `{"name": "Default"}`
	plain := func(extra string) string { return stage("a", extra, field, def) }
	for want, stages := range map[string]string{
		"declared twice":                     ok + "," + ok,
		`field "x" of "a" is declared twice`: stage("a", "", field+","+field, def),
		"no preset named Default":            stage("a", "", field, `{"name": "Compact"}`),
		"has no rendering":                   stage("a", "", `{"id": "x", "label": "x", "type": "text"}`, def),
		`record "..x"`:                       plain(`, "record": "..x", "command": ["t", "--json"]`),
		`record ""`:                          plain(`, "command": ["t", "--json"]`),
		`takes "bogus"`:                      plain(`, "record": ".", "command": ["t"], "takes": ["bogus"]`),
		"names no stage":                     stage("", "", field, def),
		"staged":                             plain(`, "record": ".", "command": ["t", "--json"], "staged": true`),
		"takes, but no command":              plain(`, "takes": ["owner"]`),
		"names no command that reads it":     plain(`, "forge": true`),
	} {
		if _, err := ReadCatalog([]byte(`{"stages": [` + stages + `]}`)); err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("%s: %v", want, err)
		}
	}
	// a stage of no fields needs no preset
	if _, err := ReadCatalog([]byte(`{"stages": [` + stage("c", "", "", "") + `]}`)); err != nil {
		t.Errorf("a stage of no fields: %v", err)
	}
}
