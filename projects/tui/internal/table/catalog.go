// Package table is what a screen's table is made of, whatever its rows hold: a catalog of fields, each with its type,
// its renderings, and where a preset that names none puts it, read from the screen's JSON; the values a row holds in
// each field; the renderers that draw a field down its column, registered by name, so the catalog stays data; the
// automatic choice among a field's renderings; and the presets that set a table's panel, and are read back from it.
package table

import (
	"bytes"
	"encoding/json"
)

// Schema is one stage's part of the catalog: the stage; the record its rows are read from, as jq names it in kit
// flow <stage> --json; its fields, in the order its columns stand; and its own presets, in the order its preset line
// lists them.
type Schema struct {
	Stage   string   `json:"stage"`
	Record  string   `json:"record"`
	Fields  []Field  `json:"fields"`
	Presets []Preset `json:"presets"`
}

// Field is one Field of a stage. ID is what presets and the code name it by, and Label what the header draws over its
// column, none for a column without a name, as builds' leading glyph. Type is the kind of value it holds, one of types.
// Path is where its value is read from in the stage's record, as jq names it: several, comma-separated, for a value
// made of several, and one after a stage's name and a colon for a value read from another stage's records, as a
// checkout's pull request from changes. Sort, Group, and Filter say whether the rows can sort, group, and be filtered
// by it, and Owns that the rows belong to the people it names. One names one of its values, as assignee, for a field
// that holds a list, as assignees: such a field never sorts while a row holds all its values, but flattened, the rows
// standing once per value of it, it sorts, and the header names it so. Values are an enum's known values, none for one whose
// values only the rows know, as an issue's repository, and Other the glyph of one it does not know, if any. Glyph is
// the glyph drawn before each value of a Field that is not an enum. Renderings names the renderers that can draw it,
// the first preferred. Dim, Right, and Wrap say how its column is drawn: dim, right-aligned, or as the one column that
// takes the width left and wraps. Placement is where a preset that names none puts it: a column, a group, or hidden.
type Field struct {
	ID         string      `json:"id"`
	Label      string      `json:"label"`
	Type       string      `json:"type"`
	Path       string      `json:"path"`
	Sort       bool        `json:"sort"`
	Group      bool        `json:"group"`
	Filter     bool        `json:"filter"`
	Owns       bool        `json:"owns"`
	One        string      `json:"one"`
	Values     []EnumValue `json:"values"`
	Other      string      `json:"other"`
	Glyph      string      `json:"glyph"`
	Renderings []string    `json:"renderings"`
	Dim        bool        `json:"dim"`
	Right      bool        `json:"right"`
	Wrap       bool        `json:"wrap"`
	Placement  string      `json:"placement"`
}

// EnumValue is one of an enum's known values, and its glyph, if it has one.
type EnumValue struct {
	Value string `json:"value"`
	Glyph string `json:"glyph"`
}

// Preset is a stage's panel as a Preset keeps it, one of the stage's own or one the person saved: its name, and what it
// says of each field it names, by id. A field it does not name takes the catalog's placement, and no sort, filter, or
// rendering. Whether shared values fold and shared emoji show are kittui's settings, every view's alike; a file whose
// presets still say them is read as ever, and what they say is passed over.
type Preset struct {
	Name   string             `json:"name"`
	Fields map[string]Setting `json:"fields,omitempty"`
}

// Setting is what a preset says of one field, three things apart: its placement, a column at a position, from 1, a
// group at a depth, from 1, the outermost first, or hidden, or none for the catalog's; its sort, asc or desc at a
// priority, from 1, or none; and, for a field a filter can narrow, the values to include or to exclude. A person's
// filter also knows @me, the caller's login, and none. Rendering pins one of the field's renderings, and Flat says
// that a field that holds a list stands once per value.
type Setting struct {
	Placement string       `json:"placement,omitempty"`
	Position  int          `json:"position,omitempty"`
	Depth     int          `json:"depth,omitempty"`
	Sort      string       `json:"sort,omitempty"`
	Priority  int          `json:"priority,omitempty"`
	Filter    *ValueFilter `json:"filter,omitempty"`
	Rendering string       `json:"rendering,omitempty"`
	Flat      bool         `json:"flat,omitempty"`
}

// ValueFilter is a preset's filter on a field: the values its rows must hold one of, or those they must hold none of.
type ValueFilter struct {
	Include []string `json:"include,omitempty"`
	Exclude []string `json:"exclude,omitempty"`
}

// The placements a field can take.
const (
	PlaceColumn = "column"
	PlaceGroup  = "group"
	PlaceHidden = "hidden"
)

// Catalog is a screen's tables' fields and presets, by each table's name.
type Catalog map[string]*Schema

// ReadCatalog reads a catalog, every key known: a list of tables under "stages", each its own Schema.
func ReadCatalog(data []byte) (Catalog, error) {
	var file struct {
		Stages []*Schema `json:"stages"`
	}
	d := json.NewDecoder(bytes.NewReader(data))
	d.DisallowUnknownFields()
	if err := d.Decode(&file); err != nil {
		return nil, err
	}
	out := Catalog{}
	for _, s := range file.Stages {
		out[s.Stage] = s
	}
	return out, nil
}
