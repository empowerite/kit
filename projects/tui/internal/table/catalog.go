// Package table is what a screen's table is made of, whatever its rows hold: a catalog of fields, each with its type,
// its renderings, and where a preset that names none puts it, read from the screen's JSON; the values a row holds in
// each field; the renderers that draw a field down its column, registered by name, so the catalog stays data; the
// automatic choice among a field's renderings; and the presets that set a table's panel, and are read back from it.
package table

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
)

// Schema is one stage's part of the catalog: the stage; the kit command that prints its rows, kit's words after kit,
// as ["toolchains", "--json"], and the words it takes beside them, as Takes names them; whether it is staged, a stage
// its verb's --stages and --marks name, which its command's second word is; whether its verb ever marks it, false for
// a stage whose mark --marks always prints null, as kit people's, none for one it marks; whether the command reads
// the forge, and so spends its rate limits; whether the command prints a document a
// line, read as jq --slurp reads them; the record its rows are read from, as Records reads them; the paths of each
// record's key, as KeyOf reads them; the line an empty table shows; its fields, in the order its columns stand; and its
// own presets, in the order its preset line lists them. A schema without a command is a screen's own, which reads its
// rows itself, and names its record as it likes.
type Schema struct {
	Stage   string   `json:"stage"`
	Command []string `json:"command"`
	Takes   []string `json:"takes"`
	Staged  bool     `json:"staged"`
	Marked  *bool    `json:"marked"`
	Forge   bool     `json:"forge"`
	Slurp   bool     `json:"slurp"`
	Record  string   `json:"record"`
	Key     string   `json:"key"`
	Empty   string   `json:"empty"`
	Fields  []Field  `json:"fields"`
	Presets []Preset `json:"presets"`
}

// Takes are the words a stage's command can take beside its own, as a Schema's Takes names them: the owner's flags,
// --all, --mine, --others, and --owner=<login>; --no-fetch; --fresh, which reads the forge afresh; --cached, which
// prints what kit kept of the stage's last read; and kit's --workspace=<group>, which reads every member of a group.
var Takes = []string{"owner", "no-fetch", "fresh", "cached", "workspace"}

// Field is one Field of a stage. ID is what presets and the code name it by, and Label what the header draws over its
// column, none for a column without a name, as builds' leading glyph. Type is the kind of value it holds, one of types.
// Path is where its value is read from in the stage's record, as Read reads it: several, comma-separated, for a value
// made of several; * for the record's keys no other field names, as a screen gathers them. Sort, Group, and Filter say
// whether the rows can sort, group, and be filtered by it, and Owns that the rows belong to the people it names. One
// names one of its values, as assignee, for a field that holds a list, as assignees: such a field sorts by size while a
// row holds all its values, how many each holds, and flattened, the rows standing once per value of it, by value, the
// header naming it so. Values are an enum's known values, none for one whose values only the rows know, as an issue's
// repository, and Other the glyph of one it does not know, if any. Glyph is the glyph drawn before each value of a
// Field that is not an enum; a row that holds more than one value of a list draws its first, and +N, dim, how many
// more. Renderings names the renderers that can draw it, the first preferred. Dim, Right, and Wrap say how its column
// is drawn: dim, right-aligned, or as the one column that takes the width left and wraps. Width says how its column
// gives way to the one that wraps: none, the column gives up the cells past its share as soon as the wrapping one is
// short; WidthWhole, it keeps its widest value while the wrapping one keeps a few cells, and gives way only after it,
// as a glossary's term does. Null is what a row whose value at the path is null draws, dim, where null means more
// than nothing, as an environment's branches, any, or a policy rule's branch merged across them all, all; and Empty
// what a row whose every value there is an empty string draws, dim, as a rule's pattern that is off. Placement is where a preset that names none puts it: a column, a
// group, or hidden; and Depth, for a group, how deep, from 1, the outermost, as a stage whose rows belong to a
// repository groups them by source, organization, and repository in each of its own views. Order
// says how its groups stand: none, sorted; OrderGiven, in the order the data gives them, an enum's known values first,
// as the catalog lists them, then every other value as the records first hold it, as kit lists a repository's primary
// remote first.
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
	Width      string      `json:"width"`
	Null       string      `json:"null"`
	Empty      string      `json:"empty"`
	Placement  string      `json:"placement"`
	Depth      int         `json:"depth"`
	Order      string      `json:"order"`
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

// WidthWhole is the width of a field whose column keeps its widest value, as Field's Width says.
const WidthWhole = "whole"

// OrderGiven is the order of a field whose groups stand as the data gives them, as Field's Order says.
const OrderGiven = "given"

// The placements a field can take.
const (
	PlaceColumn = "column"
	PlaceGroup  = "group"
	PlaceHidden = "hidden"
)

// Catalog is a screen's tables' fields and presets, by each table's name.
type Catalog map[string]*Schema

// ReadCatalog reads a catalog, every key known: a list of tables under "stages", each its own Schema, as check holds
// it. A catalog a table cannot index by, as one that declares a stage twice, is the error.
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
	var errs []error
	for _, s := range file.Stages {
		if out[s.Stage] != nil {
			errs = append(errs, fmt.Errorf("stage %q is declared twice", s.Stage))
		}
		errs = append(errs, s.check())
		out[s.Stage] = s
	}
	if err := errors.Join(errs...); err != nil {
		return nil, err
	}
	return out, nil
}

// check refuses a schema a table cannot index by: one that names no stage; a field declared twice, or without a
// rendering; a field placed in a group that cannot group or names no depth, one that names a depth but is not placed
// in a group, and two grouped at one depth; fields without a Default preset; and, for a schema with a command, a record Records does not read, a word
// it takes that Takes does not name, and staged with no stage after the verb, a command of three words at least, as
// flow backlog --json, and marked on a stage that is not staged. Words taken, staged, or the forge read without a
// command are refused too.
func (s *Schema) check() error {
	var errs []error
	if s.Stage == "" {
		errs = append(errs, errors.New("a table in the catalog names no stage"))
	}
	ids, depths := map[string]bool{}, map[int]string{}
	for _, f := range s.Fields {
		if ids[f.ID] {
			errs = append(errs, fmt.Errorf("field %q of %q is declared twice", f.ID, s.Stage))
		}
		ids[f.ID] = true
		if len(f.Renderings) == 0 {
			errs = append(errs, fmt.Errorf("field %q of %q has no rendering", f.ID, s.Stage))
		}
		if f.Placement == PlaceGroup && (!f.Group || f.Depth < 1) {
			errs = append(errs, fmt.Errorf("field %q of %q is placed in a group, but cannot group or names no depth",
				f.ID, s.Stage))
		}
		if f.Depth > 0 && f.Placement != PlaceGroup {
			errs = append(errs, fmt.Errorf("field %q of %q names a depth, but is not placed in a group", f.ID, s.Stage))
		}
		if f.Depth > 0 && depths[f.Depth] != "" {
			errs = append(errs, fmt.Errorf("fields %q and %q of %q are grouped at one depth", depths[f.Depth], f.ID,
				s.Stage))
		}
		depths[f.Depth] = f.ID
	}
	if len(s.Fields) > 0 && !slices.ContainsFunc(s.Presets, func(p Preset) bool { return p.Name == Opening }) {
		errs = append(errs, fmt.Errorf("stage %q has no preset named %s", s.Stage, Opening))
	}
	if len(s.Command) == 0 {
		if len(s.Takes) > 0 || s.Staged {
			errs = append(errs, fmt.Errorf("stage %q takes, but no command takes them", s.Stage))
		}
		if s.Forge {
			errs = append(errs, fmt.Errorf("stage %q reads the forge, but names no command that reads it", s.Stage))
		}
		return errors.Join(errs...)
	}
	if !recordForms.MatchString(s.Record) {
		errs = append(errs, fmt.Errorf("stage %q reads record %q; want ., .[], or .<key>[]", s.Stage, s.Record))
	}
	for _, word := range s.Takes {
		if !slices.Contains(Takes, word) {
			errs = append(errs, fmt.Errorf("stage %q takes %q; want one of %s", s.Stage, word,
				strings.Join(Takes, ", ")))
		}
	}
	if s.Marked != nil && !s.Staged {
		errs = append(errs, fmt.Errorf("stage %q says whether it is marked, but is not staged", s.Stage))
	}
	if s.Staged && len(s.Command) < 3 {
		errs = append(errs, fmt.Errorf("stage %q is staged, but %q names no stage after its verb", s.Stage,
			strings.Join(s.Command, " ")))
	}
	return errors.Join(errs...)
}
