// Automatic renderings: a field whose preset pins no rendering is drawn by one of its own that the rows shown and the
// pane's width choose, its renderings taken in the catalog's order.
package table

import (
	"maps"
	"slices"

	"github.com/empowerite/kit/projects/tui/internal/grid"
)

// Placed is a field as a stage's table places it, for automatic: the field; the rendering its preset pins, none for
// auto; whether it stands as a column; and the name the header stands over it, none while the header is hidden.
type Placed struct {
	Field
	Pinned string
	Column bool
	Name   string
}

// Automatic gives the rendering each field is drawn by, by id, for the values the rows shown hold in it, by id, one a
// row, and the cells the rows are laid out in; the fields that stand as columns come in the order they stand. A field
// whose preset pins one of its renderings is drawn by it. Else its renderings are taken in order, and three rules
// choose among them. A glyph that leads somewhere is a button: where a value the rows hold in the field links, the
// field keeps to its renderings that draw a glyph, when it has one. Else a glyph every row shares says nothing: where
// each value the rows hold in the field draws the same glyph, the field drops its renderings that draw one, unless
// every one does, or the panel shows the emoji every row shares, as shared says. And a table short of width steps down:
// while the columns do not fit, as grid.Fits says, each column, from the rightmost leftward and a step each in turn,
// takes its next rendering that is narrower, until they fit or none can step. The column that wraps takes what is left,
// as it always does, and never steps.
func Automatic(fields []Placed, vs map[string][]Value, width int, shared bool) map[string]string {
	out, ladders := map[string]string{}, map[string][]string{}
	for _, f := range fields {
		if slices.Contains(f.Renderings, f.Pinned) {
			out[f.ID] = f.Pinned
			continue
		}
		ladder := f.Renderings
		drawing := slices.DeleteFunc(slices.Clone(ladder), func(r string) bool { return !glyphed[r] })
		plain := slices.DeleteFunc(slices.Clone(ladder), func(r string) bool { return glyphed[r] })
		switch {
		case len(drawing) > 0 && Links(vs[f.ID]):
			ladder = drawing
		case len(plain) > 0 && !shared && f.SharesGlyph(vs[f.ID]):
			ladder = plain
		}
		out[f.ID], ladders[f.ID] = ladder[0], ladder[1:]
	}
	needs := map[string]int{}
	need := func(f Placed, rendering string) int { return grid.Need(f.Draw(vs[f.ID], rendering), f.Name) }
	wraps := false
	for _, f := range fields {
		switch {
		case !f.Column:
		case f.Wrap:
			wraps = wraps || need(f, out[f.ID]) > 0
		default:
			needs[f.ID] = need(f, out[f.ID])
		}
	}
	fits := func() bool { return grid.Fits(slices.Collect(maps.Values(needs)), wraps, width) }
	for stepped := true; stepped && !fits(); {
		stepped = false
		for k := len(fields) - 1; k >= 0 && !fits(); k-- {
			f := fields[k]
			if !f.Column || f.Wrap {
				continue
			}
			for len(ladders[f.ID]) > 0 {
				next := ladders[f.ID][0]
				ladders[f.ID] = ladders[f.ID][1:]
				if n := need(f, next); n < needs[f.ID] {
					out[f.ID], needs[f.ID], stepped = next, n, true
					break
				}
			}
		}
	}
	return out
}

// SharesGlyph says whether every value the rows hold in a field draws the same glyph, so the glyph says nothing the
// value does not: a row that holds none draws none, and does not count. Two rows at least must share it: the glyph of
// a row alone, as the desk's, says what the row is, which nothing beside it says. A row that holds more than one
// value of a list with a glyph for many draws that glyph alone, as its summary does.
func (f Field) SharesGlyph(vs []Value) bool {
	seen, rows := "", 0
	for _, v := range vs {
		if len(v.Raw) > 0 {
			rows++
		}
		glyphs := make([]string, len(v.Raw))
		for k, r := range v.Raw {
			glyphs[k] = f.GlyphOf(r)
		}
		if f.Many != "" && len(v.Raw) > 1 {
			glyphs = []string{f.Many}
		}
		for _, g := range glyphs {
			if g == "" || seen != "" && g != seen {
				return false
			}
			seen = g
		}
	}
	return seen != "" && rows > 1
}

// Links says whether a value the rows hold leads somewhere, a page or another stage's row, from its glyph.
func Links(vs []Value) bool {
	return slices.ContainsFunc(vs, func(v Value) bool { return v.Link != "" })
}
