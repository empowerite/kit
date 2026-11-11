// A table read from what its command prints: the records, as the schema's record names them in the document, each
// field's value, as its path names it in a record, and each record's key. Every read is fresh: a record shares no
// bytes with the document it came from, nor with another read's, so a value one read held is never another's.
package table

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"regexp"
	"slices"
	"strings"
)

// ShapeError is a document of another shape than the record expects: what was Printed, as nothing, null, an array, or
// an object without "rows", and what was Wanted, an object, an array, or an object with "<key>".
type ShapeError struct{ Printed, Wanted string }

func (e *ShapeError) Error() string {
	return "printed " + e.Printed + " where " + e.Wanted + " was expected"
}

// recordForms are the forms a record takes: . the document, an object; .[] each element of the document, an array; and
// .<key>[] each element of the array the document, an object, holds under <key>.
var recordForms = regexp.MustCompile(`^\.([A-Za-z_][A-Za-z0-9_]*\[\])?$|^\.\[\]$`)

// wanted says what document a record expects, as a ShapeError words it.
func wanted(record string) string {
	switch record {
	case ".":
		return "an object"
	case ".[]":
		return "an array"
	}
	return fmt.Sprintf("an object with %q", strings.TrimSuffix(strings.TrimPrefix(record, "."), "[]"))
}

// kindOf words what a JSON value is, as a ShapeError says what was printed.
func kindOf(raw json.RawMessage) string {
	switch raw[0] {
	case '{':
		return "an object"
	case '[':
		return "an array"
	case '"':
		return "a string"
	case 'n':
		return "null"
	case 't', 'f':
		return "a boolean"
	}
	return "a number"
}

// Records gives the records of a document, as record names them: fresh, each its own bytes. With slurp the data is a
// document a line, as a command that prints one a line gives them, read as jq --slurp reads them, into one array, so
// no line is no record; without it, the data is one document, and nothing printed is a ShapeError, as is null or a
// document of another shape. JSON that does not decode is its own error.
func Records(data []byte, record string, slurp bool) ([]json.RawMessage, error) {
	doc, err := document(data, slurp)
	if err != nil {
		return nil, err
	}
	refuse := func(printed string) error { return &ShapeError{Printed: printed, Wanted: wanted(record)} }
	if doc == nil {
		return nil, refuse("nothing")
	}
	switch kind := kindOf(doc); {
	case record == "." && kind == "an object":
		return []json.RawMessage{doc}, nil
	case record == ".[]" && kind == "an array":
		return elements(doc)
	case record == "." || record == ".[]" || kind != "an object":
		return nil, refuse(kind)
	}
	key := strings.TrimSuffix(strings.TrimPrefix(record, "."), "[]")
	var object map[string]json.RawMessage
	_ = json.Unmarshal(doc, &object) // the document is an object, which decodes as one
	held, ok := object[key]
	switch {
	case !ok:
		return nil, refuse(fmt.Sprintf("an object without %q", key))
	case kindOf(held) != "an array":
		return nil, refuse(fmt.Sprintf("%s %q", kindOf(held), key))
	}
	return elements(held)
}

// document gives the one document the data holds, or, with slurp, the array of the documents it holds, one a line;
// none for nothing printed, without slurp.
func document(data []byte, slurp bool) (json.RawMessage, error) {
	d := json.NewDecoder(bytes.NewReader(data))
	var docs []json.RawMessage
	for {
		var doc json.RawMessage
		err := d.Decode(&doc)
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, err
		}
		docs = append(docs, doc)
	}
	switch {
	case slurp:
		return json.Marshal(append([]json.RawMessage{}, docs...))
	case len(docs) > 1:
		return nil, fmt.Errorf("printed %d documents where one was expected", len(docs))
	case len(docs) == 0:
		return nil, nil
	}
	return docs[0], nil
}

// elements gives each element of an array, each its own bytes.
func elements(array json.RawMessage) ([]json.RawMessage, error) {
	out := []json.RawMessage{}
	return out, json.Unmarshal(array, &out)
}

// Decoded gives a record as Go holds it, its numbers as their digits, as ReadFrom and KeyFrom read it; none for one
// that does not decode. A screen that reads a record's fields decodes it once, here, and reads each from what this
// gives.
func Decoded(record json.RawMessage) any {
	d := json.NewDecoder(bytes.NewReader(record))
	d.UseNumber()
	var v any
	if d.Decode(&v) != nil {
		return nil
	}
	return v
}

// walk gives what a path names in a value, as Read reads it: .a.b, each key a step, a [] after one allowed and ignored,
// a step through an array taken in each of its elements; nothing for a step a value does not hold, or a path that is
// not one. Each value it gives is there, and may be null.
func walk(v any, path string) []any {
	path = strings.TrimSpace(path)
	if !strings.HasPrefix(path, ".") {
		return nil
	}
	values := []any{v}
	for _, step := range strings.Split(path[1:], ".") {
		if step = strings.TrimSuffix(step, "[]"); step == "" {
			return nil
		}
		var next []any
		for _, v := range flat(values) {
			if object, ok := v.(map[string]any); ok {
				if held, ok := object[step]; ok {
					next = append(next, held)
				}
			}
		}
		values = next
	}
	return values
}

// flat gives the values, an array's elements in its place, at every depth.
func flat(values []any) []any {
	var out []any
	for _, v := range values {
		if each, ok := v.([]any); ok {
			out = append(out, flat(each)...)
			continue
		}
		out = append(out, v)
	}
	return out
}

// Read gives the field's value in a record, as ReadFrom reads it from the record decoded.
func (f Field) Read(record json.RawMessage) Value { return f.ReadFrom(Decoded(record)) }

// ReadFrom gives the field's value in a record decoded, as Decoded gives it, as its path names it: one path, or
// several after commas, each .a.b, a [] after a key allowed and ignored. An array gives the value an element each,
// and a path through one maps over it; a string is itself, a number its digits, true the field's id, and false, null,
// or an object no value. A field that says what null draws, as its Null, draws it for a record that holds null at a
// path, and no value at any.
func (f Field) ReadFrom(v any) Value {
	var out Value
	null := false
	for _, path := range strings.Split(f.Path, ",") {
		held := walk(v, path)
		null = null || len(held) > 0 && !slices.ContainsFunc(held, func(h any) bool { return h != nil })
		for _, each := range flat(held) {
			switch each := each.(type) {
			case string:
				out.Raw = append(out.Raw, each)
			case json.Number:
				out.Raw = append(out.Raw, each.String())
			case bool:
				if each {
					out.Raw = append(out.Raw, f.ID)
				}
			}
		}
	}
	if null && len(out.Raw) == 0 {
		out.Else = f.Null
	}
	return out
}

// KeyOf gives a record's key, as KeyFrom reads it from the record decoded.
func (s Schema) KeyOf(record json.RawMessage) (key string, ok bool) {
	return s.KeyFrom(Decoded(record))
}

// KeyFrom gives a record's key, from the record decoded, as Decoded gives it, as the schema's key names it, its paths
// after commas, or .key where it names none: each scalar's text, a space between, and ok when each path gives the text
// of a scalar, a string, a number, or a boolean, at least. A path that is not there, is null, or gives only objects,
// and a record that is no object, have none.
func (s Schema) KeyFrom(v any) (key string, ok bool) {
	paths := s.Key
	if _, object := v.(map[string]any); !object {
		return "", false
	}
	if paths == "" {
		paths = ".key"
	}
	var texts []string
	for _, path := range strings.Split(paths, ",") {
		n := len(texts)
		for _, each := range flat(walk(v, path)) {
			switch each := each.(type) {
			case string:
				texts = append(texts, each)
			case json.Number:
				texts = append(texts, each.String())
			case bool:
				texts = append(texts, fmt.Sprint(each))
			}
		}
		if len(texts) == n {
			return "", false
		}
	}
	return strings.Join(texts, " "), true
}
