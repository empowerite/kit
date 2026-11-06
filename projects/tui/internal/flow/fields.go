// The field catalog, fields.json: for each stage with columns, where its records are in kit flow <stage> --json, its
// fields, and its own presets. A field is declared once: a stable id, which presets and the code use, and a label,
// which only the header shows; the type of value it holds; whether its rows can sort, group, and be filtered by it; an
// enum's known values, each with its glyph; its renderings, each a renderer by name, in order of preference; the record
// path its value is read from; and the placement it takes where a preset says none. The catalog is the data's
// description; the value types and the renderers, its behavior, are Go's, registered here by name, and a test fails on
// a name the catalog gives that is not registered.
package flow

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"slices"
	"strconv"

	"github.com/empowerite/kit/projects/tui/internal/grid"
	"github.com/empowerite/kit/projects/tui/internal/style"
)

//go:embed fields.json
var fieldsJSON []byte

// catalog is fields.json, each stage's fields and presets by the stage's name. A stage it does not name, as incoming,
// whose rows are commits and branches, has no columns.
var catalog = readCatalog(fieldsJSON)

// stageFields is one stage's part of the catalog: the stage; the record its rows are read from, as jq names it in kit
// flow <stage> --json; its fields, in the order its columns stand; and its own presets, in the order its preset line
// lists them.
type stageFields struct {
	Stage   string   `json:"stage"`
	Record  string   `json:"record"`
	Fields  []field  `json:"fields"`
	Presets []preset `json:"presets"`
}

// field is one field of a stage. ID is what presets and the code name it by, and Label what the header draws over its
// column, none for a column without a name, as builds' leading glyph. Type is the kind of value it holds, one of types.
// Path is where its value is read from in the stage's record, as jq names it: several, comma-separated, for a value
// made of several, and one after a stage's name and a colon for a value read from another stage's records, as a
// checkout's pull request from changes. Sort, Group, and Filter say whether the rows can sort, group, and be filtered
// by it, and Owns that the rows belong to the people it names. Values are an enum's known values, and Other the glyph of
// one it does not know, if any. Glyph is the glyph drawn before each value of a field that is not an enum. Renderings
// names the renderers that can draw it, the first preferred. Dim, Right, and Wrap say how its column is drawn: dim,
// right-aligned, or as the one column that takes the width left and wraps. Placement is where a preset that names
// none puts it: a column, a group, or hidden.
type field struct {
	ID         string      `json:"id"`
	Label      string      `json:"label"`
	Type       string      `json:"type"`
	Path       string      `json:"path"`
	Sort       bool        `json:"sort"`
	Group      bool        `json:"group"`
	Filter     bool        `json:"filter"`
	Owns       bool        `json:"owns"`
	Values     []enumValue `json:"values"`
	Other      string      `json:"other"`
	Glyph      string      `json:"glyph"`
	Renderings []string    `json:"renderings"`
	Dim        bool        `json:"dim"`
	Right      bool        `json:"right"`
	Wrap       bool        `json:"wrap"`
	Placement  string      `json:"placement"`
}

// enumValue is one of an enum's known values, and its glyph, if it has one.
type enumValue struct {
	Value string `json:"value"`
	Glyph string `json:"glyph"`
}

// preset is a stage's panel as a preset keeps it, one of the stage's own or one the person saved: its name; whether the
// header shows, or, when it says nothing, as the screen opens; whether it folds, a value every row of a group shares
// said once, in the group's heading, or, when it says nothing, as it does; and what it says of each field it names, by
// id. A field it does not name takes the catalog's placement, and no sort, filter, or rendering.
type preset struct {
	Name    string             `json:"name"`
	Headers *bool              `json:"headers,omitempty"`
	Fold    *bool              `json:"fold,omitempty"`
	Fields  map[string]setting `json:"fields,omitempty"`
}

// setting is what a preset says of one field, three things apart: its placement, a column at a position, from 1, a
// group at a depth, from 1, the outermost first, or hidden, or none for the catalog's; its sort, asc or desc at a
// priority, from 1, or none; and, for a field a filter can narrow, the values to include or to exclude. A person's
// filter also knows @me, the caller's login, and none. Rendering pins one of the field's renderings.
type setting struct {
	Placement string       `json:"placement,omitempty"`
	Position  int          `json:"position,omitempty"`
	Depth     int          `json:"depth,omitempty"`
	Sort      string       `json:"sort,omitempty"`
	Priority  int          `json:"priority,omitempty"`
	Filter    *valueFilter `json:"filter,omitempty"`
	Rendering string       `json:"rendering,omitempty"`
}

// valueFilter is a preset's filter on a field: the values its rows must hold one of, or those they must hold none of.
type valueFilter struct {
	Include []string `json:"include,omitempty"`
	Exclude []string `json:"exclude,omitempty"`
}

// The placements a field can take, and the person a person's filter names by the caller's login.
const (
	placeColumn = "column"
	placeGroup  = "group"
	placeHidden = "hidden"
	caller      = grid.Caller
)

// readCatalog reads the catalog, every key known; one that does not read is kittui's own fault, which the tests catch
// before it ships.
func readCatalog(data []byte) map[string]*stageFields {
	var file struct {
		Stages []*stageFields `json:"stages"`
	}
	d := json.NewDecoder(bytes.NewReader(data))
	d.DisallowUnknownFields()
	if err := d.Decode(&file); err != nil {
		panic(fmt.Sprintf("fields.json does not read: %v", err))
	}
	out := map[string]*stageFields{}
	for _, s := range file.Stages {
		out[s.Stage] = s
	}
	return out
}

// fieldsOf gives a stage's fields, in the order its columns stand; none for a stage without columns.
func fieldsOf(stage string) []field {
	if s := catalog[stage]; s != nil {
		return s.Fields
	}
	return nil
}

// columnsOf gives the names the panel knows a stage's columns by, in the order they stand: each field's id, and none
// for a field without a label, which the panel offers no checkbox for, never moves, and never sorts or groups by.
func columnsOf(stage string) []string {
	var out []string
	for _, f := range fieldsOf(stage) {
		if f.Label == "" {
			out = append(out, "")
			continue
		}
		out = append(out, f.ID)
	}
	return out
}

// labelOf gives the label the header draws over a stage's column, by the field's id; an id the stage does not know
// is its own label.
func labelOf(stage, id string) string {
	if k := slices.IndexFunc(fieldsOf(stage), func(f field) bool { return f.ID == id }); k >= 0 {
		return fieldsOf(stage)[k].Label
	}
	return id
}

// able gives the ids of a stage's fields that can, as can says, as a set.
func able(stage string, can func(field) bool) map[string]bool {
	out := map[string]bool{}
	for _, f := range fieldsOf(stage) {
		if f.Label != "" && can(f) {
			out[f.ID] = true
		}
	}
	return out
}

// sortableOf names a stage's columns its rows can sort by: a value that orders.
func sortableOf(stage string) map[string]bool {
	return able(stage, func(f field) bool { return f.Sort })
}

// groupableOf names a stage's columns its rows can group by: a value many rows share.
func groupableOf(stage string) map[string]bool {
	return able(stage, func(f field) bool { return f.Group })
}

// ownerOf gives the id of the person field that names the people a stage's rows belong to, whose filter the panel
// sets as every row, the caller's, or everyone else's; none for a stage whose rows belong to nobody.
func ownerOf(stage string) string {
	for _, f := range fieldsOf(stage) {
		if f.Owns {
			return f.ID
		}
	}
	return ""
}

// valueType is a kind of value a field holds, and whether a filter chooses among its values, as among an enum's known
// values and among people.
type valueType struct {
	filters bool
}

// types gives each kind of value a field can hold, by the name the catalog gives it: a reference, as an issue's #45; a
// text; an enum, one of known values; a person's login; a time, in seconds; and a number.
var types = map[string]valueType{
	"ref":    {},
	"text":   {},
	"enum":   {filters: true},
	"person": {filters: true},
	"time":   {},
	"number": {},
}

// value is what a row holds in one field. Raw is what the row sorts, groups, and filters by, unstyled, one a value, as
// a person field's logins, or none: a time's seconds, a number's digits, a reference as its tracker gives it. Of is the
// record, or the part of it, a renderer draws more from than the value, as the checks' counts beside their state. Link
// is where the glyph the field draws leads; Else is what the field draws, dim, while it holds no value, as … while
// changes loads.
type value struct {
	Raw  []string
	Of   any
	Link string
	Else string
}

// one gives a value of one raw text.
func one(raw string) value { return value{Raw: []string{raw}} }

// some gives a value of one raw text, or none when it is empty.
func some(raw string) value {
	if raw == "" {
		return value{}
	}
	return one(raw)
}

// seconds gives the value of a time, in seconds, never fewer than none; nothing for a time that is missing.
func seconds(secs *int64) value {
	if secs == nil {
		return value{}
	}
	return one(strconv.FormatInt(max(*secs, 0), 10))
}

// renderer draws a field down its column: from the values the rows hold in it, one a row, the cell each row shows.
// It draws nothing for a value that holds none, styles a value its drawing gives a style of its own, and leaves the
// rest, the field's dim, alignment, link, and sort, to draw.
type renderer func(f field, vs []value) []cell

// renderers gives each renderer by the name the catalog gives it.
var renderers = map[string]renderer{
	"text":        each(func(f field, v value) []string { return slices.Clone(v.Raw) }),
	"glyph+text":  each(func(f field, v value) []string { return f.glyphed(v.Raw, true) }),
	"glyph":       each(func(f field, v value) []string { return f.glyphed(v.Raw, false) }),
	"age":         each(func(f field, v value) []string { return mapped(v.Raw, age) }),
	"took":        each(func(f field, v value) []string { return mapped(v.Raw, func(s int64) string { return took(&s) }) }),
	"glyph+age":   pushedAges,
	"checkout":    each(checkoutName),
	"counts":      aheadBehind,
	"diffstat":    each(func(f field, v value) []string { return []string{diffstat(v.Of.(checkout))} }),
	"uncommitted": uncommittedCells,
	"checks":      checksCells,
	"threads":     each(threadsText),
	"where":       whereCells,
}

// each gives a renderer that draws every row's value alone, as texts, without regard to the others.
func each(draw func(f field, v value) []string) renderer {
	return func(f field, vs []value) []cell {
		out := make([]cell, len(vs))
		for k, v := range vs {
			if len(v.Raw) > 0 {
				out[k].Values = draw(f, v)
			}
		}
		return out
	}
}

// mapped gives each raw text read as seconds, drawn.
func mapped(raw []string, draw func(int64) string) []string {
	var out []string
	for _, r := range raw {
		secs, _ := strconv.ParseInt(r, 10, 64)
		out = append(out, draw(secs))
	}
	return out
}

// glyphOf gives the glyph of one of a field's values: an enum's known value's own, else the enum's for a value it does
// not know; a field that is not an enum draws its own before every value.
func (f field) glyphOf(raw string) string {
	if f.Type != "enum" {
		return f.Glyph
	}
	if k := slices.IndexFunc(f.Values, func(e enumValue) bool { return e.Value == raw }); k >= 0 {
		return f.Values[k].Glyph
	}
	return f.Other
}

// glyphed gives each raw text after its glyph, with the text when text says so, or the glyph alone; a text without a
// glyph stands alone.
func (f field) glyphed(raw []string, text bool) []string {
	var out []string
	for _, r := range raw {
		g := f.glyphOf(r)
		switch {
		case g == "":
			out = append(out, r)
		case text:
			out = append(out, g+" "+r)
		default:
			out = append(out, g)
		}
	}
	return out
}

// pushedAges draws when a branch was last pushed: the field's glyph and the age, right-aligned to the widest age in
// the column, so the glyphs stand in a line.
func pushedAges(f field, vs []value) []cell {
	widest := 0
	for _, v := range vs {
		for _, a := range mapped(v.Raw, age) {
			widest = max(widest, len(a))
		}
	}
	return each(func(f field, v value) []string {
		return mapped(v.Raw, func(s int64) string { return f.Glyph + " " + fmt.Sprintf("%*s", widest, age(s)) })
	})(f, vs)
}

// checkoutName draws a checkout's name after the field's glyph, and, for a checkout on no branch, that it is detached.
func checkoutName(f field, v value) []string {
	label := f.Glyph + " " + v.Raw[0]
	if c, ok := v.Of.(checkout); ok && c.Detached {
		label += " " + Detached + " detached"
	}
	return []string{label}
}

// aheadBehind draws a checkout's commits ahead and behind, as counts does, ahead right-aligned to the widest in the
// column.
func aheadBehind(f field, vs []value) []cell {
	widest := 0
	for _, v := range vs {
		if c, ok := v.Of.(checkout); ok {
			widest = max(widest, len("↑"+strconv.Itoa(c.Ahead)))
		}
	}
	return each(func(f field, v value) []string { return []string{counts(v.Of.(checkout), widest)} })(f, vs)
}

// uncommittedCells draws a checkout's uncommitted paths, as uncommitted does: the count, or clean, dim.
func uncommittedCells(f field, vs []value) []cell {
	out := make([]cell, len(vs))
	for k, v := range vs {
		if len(v.Raw) > 0 {
			n, _ := strconv.Atoi(v.Raw[0])
			out[k] = uncommitted(n)
		}
	}
	return out
}

// checksCells draws a pull request's checks, as their cell does; a heading over rows grouped by them says their state
// after its glyph, as the counts and the time differ from row to row.
func checksCells(f field, vs []value) []cell {
	out := make([]cell, len(vs))
	for k, v := range vs {
		c, ok := v.Of.(*checks)
		if len(v.Raw) == 0 || !ok {
			continue
		}
		out[k] = text(c.cell())
		out[k].Head = f.glyphed(v.Raw, true)[0]
	}
	return out
}

// threadsText draws a pull request's review threads unresolved after the field's glyph, or nothing for none.
func threadsText(f field, v value) []string {
	if n, _ := strconv.Atoi(v.Raw[0]); n > 0 {
		return []string{fmt.Sprintf("%s %d unresolved", f.Glyph, n)}
	}
	return nil
}

// whereCells draws where the work of an issue underway is, as its value's Of says: here, after the desk's glyph or a
// worktree's, for a checkout on this desk; local for a branch this desk has without a checkout; else the pull
// request's author, dim.
func whereCells(f field, vs []value) []cell {
	out := make([]cell, len(vs))
	for k, v := range vs {
		if len(v.Raw) == 0 {
			continue
		}
		switch v.Of {
		case "desk":
			out[k] = text(Desk + " here")
		case "here":
			out[k] = text(Branch + " here")
		case "local":
			out[k] = text(Sprout + " local")
		default:
			out[k] = dim(Owner + " " + v.Raw[0])
		}
	}
	return out
}

// draw gives a field's cells from the values the rows hold in it, as the rendering pinned draws them, when it is one of
// the field's, else its first: each dim where the field is, the field's alignment, each row's link, and what each
// sorts, groups, and filters by, its raw values; what a field holding no value draws instead, dim.
func (f field) draw(vs []value, pinned string) []cell {
	rendering := f.Renderings[0]
	if slices.Contains(f.Renderings, pinned) {
		rendering = pinned
	}
	out := renderers[rendering](f, vs)
	for k, v := range vs {
		c := &out[k]
		if f.Dim {
			for j, s := range c.Values {
				if s != "" {
					c.Values[j] = style.Off(s)
				}
			}
		}
		if len(v.Raw) == 0 && v.Else != "" {
			c.Values = []string{style.Off(v.Else)}
		}
		c.Right, c.Wrap, c.Link, c.Raw = f.Right, f.Wrap, v.Link, v.Raw
	}
	return out
}
