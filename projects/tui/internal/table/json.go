// A value of any shape, as JSON holds it, drawn as jq draws it: on a closed row compact, as jq -c prints it, and on an
// open row pretty, as jq . prints it, a line each. A string draws bare, as jq -r prints it.
package table

import (
	"bytes"
	"encoding/json"
	"strings"

	"github.com/empowerite/kit/projects/tui/internal/grid"
)

// readJSON gives a json field's value in a record decoded: each value at its paths, an array or an object whole, its
// Raw the value compact, which the rows sort, group, and search by, and its Of the value pretty, one a raw value. Null,
// or no value at any path, is no value, and draws the field's Null where the record holds null.
func (f Field) readJSON(v any) Value {
	var out Value
	var pretty []string
	null := false
	for _, path := range strings.Split(f.Path, ",") {
		for _, each := range walk(v, path) {
			if each == nil {
				null = true
				continue
			}
			compact, indented := jqText(each)
			out.Raw, pretty = append(out.Raw, compact), append(pretty, indented)
		}
	}
	if len(out.Raw) > 0 {
		out.Of = pretty
	} else if null {
		out.Else = f.Null
	}
	return out
}

// jqText gives a value decoded as jq -c and jq . print it: a string bare, as jq -r does; anything else as JSON, an
// object's keys sorted, as jq -S sorts them, since the record decoded keeps no order; two spaces an indent; and no
// character escaped that JSON lets stand, as jq leaves < and &.
func jqText(v any) (compact, pretty string) {
	if s, ok := v.(string); ok {
		return s, s
	}
	var b bytes.Buffer
	e := json.NewEncoder(&b)
	e.SetEscapeHTML(false)
	_ = e.Encode(v) // a value decoded from JSON encodes again
	compact = strings.TrimSuffix(b.String(), "\n")
	b.Reset()
	_ = json.Indent(&b, []byte(compact), "", "  ") // compact is JSON the encoder just wrote
	return compact, b.String()
}

// drawJSON draws a json field: a value of one line as it is; a value pretty-printed onto several, its lines down the
// column while the row is open, and the value compact in their place while it is closed.
func drawJSON(f Field, vs []Value) []grid.Cell {
	out := make([]grid.Cell, len(vs))
	for k, v := range vs {
		pretty, _ := v.Of.([]string)
		if len(pretty) != len(v.Raw) {
			pretty = v.Raw
		}
		var lines []string
		for _, p := range pretty {
			lines = append(lines, strings.Split(p, "\n")...)
		}
		out[k].Values = lines
		if len(lines) > len(v.Raw) {
			out[k].Summary = strings.Join(v.Raw, " ")
		}
	}
	return out
}
