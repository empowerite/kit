// A value of any shape, as JSON holds it, drawn as jq draws it: on a closed row compact, as jq -c prints it, and on an
// open row pretty, as jq . prints it, a line each. A string draws bare, as jq -r prints it. Each draws colored to be
// scanned: a key in the accent color, a string green, a number yellow, a true or a false magenta, and a null, the
// marks between them, and so an empty array or object, dim.
package table

import (
	"bytes"
	"encoding/json"
	"slices"
	"strings"

	"github.com/empowerite/kit/projects/tui/internal/grid"
	"github.com/empowerite/kit/projects/tui/internal/style"
)

// jsonText is one value of a json field, as jq . prints it, and whether it is a string, which draws bare.
type jsonText struct {
	Pretty string
	Bare   bool
}

// readJSON gives a json field's value in a record decoded: each value at its paths, an array or an object whole, its
// Raw the value compact, which the rows sort, group, and search by, and its Of the value pretty, one a raw value. Null,
// or no value at any path, is no value, and draws the field's Null where the record holds null; a string of no
// characters is none too, and draws the field's Empty.
func (f Field) readJSON(v any) Value {
	var out Value
	var pretty []jsonText
	null := false
	for _, path := range strings.Split(f.Path, ",") {
		for _, each := range walk(v, path) {
			if each == nil {
				null = true
				continue
			}
			compact, indented := jqText(each)
			_, bare := each.(string)
			out.Raw, pretty = append(out.Raw, compact), append(pretty, jsonText{indented, bare})
		}
	}
	switch {
	case f.Empty != "" && len(out.Raw) > 0 && !slices.ContainsFunc(out.Raw, func(s string) bool { return s != "" }):
		out.Raw, out.Else = nil, f.Empty
	case len(out.Raw) > 0:
		out.Of = pretty
	case null:
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
// column while the row is open, and the value compact in their place while it is closed; each colored as colorJSON says.
func drawJSON(f Field, vs []Value) []grid.Cell {
	out := make([]grid.Cell, len(vs))
	for k, v := range vs {
		pretty, _ := v.Of.([]jsonText)
		if len(pretty) != len(v.Raw) {
			for _, r := range v.Raw {
				pretty = append(pretty, jsonText{r, true})
			}
		}
		var lines, compact []string
		for j, p := range pretty {
			for _, line := range strings.Split(p.Pretty, "\n") {
				lines = append(lines, colorJSON(line, p.Bare))
			}
			compact = append(compact, colorJSON(v.Raw[j], p.Bare))
		}
		out[k].Values = lines
		if len(lines) > len(v.Raw) {
			out[k].Summary = strings.Join(compact, " ")
		}
	}
	return out
}

// colorJSON draws a line of JSON, or a string drawn bare, colored: a key, a string, a number, a true or a false, and
// a null, each in its own; the braces, brackets, commas, and colons dim, so an empty array or object is dim too.
func colorJSON(line string, bare bool) string {
	if bare {
		if line == "" {
			return ""
		}
		return style.Text(line)
	}
	var b strings.Builder
	for i := 0; i < len(line); {
		switch c := line[i]; {
		case c == '"':
			end := stringEnd(line, i)
			word, rest := line[i:end], strings.TrimLeft(line[end:], " ")
			if strings.HasPrefix(rest, ":") {
				b.WriteString(style.Tint(word))
			} else {
				b.WriteString(style.Text(word))
			}
			i = end
		case c == '-' || c >= '0' && c <= '9':
			end := i
			for end < len(line) && strings.IndexByte("-+.eE0123456789", line[end]) >= 0 {
				end++
			}
			b.WriteString(style.Warn(line[i:end]))
			i = end
		case c >= 'a' && c <= 'z':
			end := i
			for end < len(line) && line[end] >= 'a' && line[end] <= 'z' {
				end++
			}
			if word := line[i:end]; word == "null" {
				b.WriteString(style.Off(word))
			} else {
				b.WriteString(style.Literal(word))
			}
			i = end
		case c == ' ':
			b.WriteByte(c)
			i++
		default:
			end := i
			for end < len(line) && strings.IndexByte("{}[],:", line[end]) >= 0 {
				end++
			}
			b.WriteString(style.Off(line[i:max(end, i+1)]))
			i = max(end, i+1)
		}
	}
	return b.String()
}

// stringEnd gives where the string that starts at i, at its opening quote, ends: just after its closing quote, or at
// the line's end for one a line cuts off. A backslash escapes the character after it.
func stringEnd(line string, i int) int {
	for j := i + 1; j < len(line); j++ {
		switch line[j] {
		case '\\':
			j++
		case '"':
			return j + 1
		}
	}
	return len(line)
}
