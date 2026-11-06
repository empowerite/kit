// Automatic renderings: a field whose preset pins no rendering is drawn by one of its own that the rows shown and the
// pane's width choose, its renderings taken in the catalog's order.
package flow

import (
	"maps"
	"slices"

	"github.com/empowerite/kit/projects/tui/internal/grid"
)

// placed is a field as a stage's table places it, for automatic: the field; the rendering its preset pins, none for
// auto; whether it stands as a column; and the name the header stands over it, none while the header is hidden.
type placed struct {
	field
	pinned string
	column bool
	name   string
}

// automatic gives the rendering each field is drawn by, by id, for the values the rows shown hold in it, by id, one a
// row, and the cells the rows are laid out in; the fields that stand as columns come in the order they stand. A field
// whose preset pins one of its renderings is drawn by it. Else its renderings are taken in order, and two rules choose
// among them. A glyph every row shares says nothing: where each value the rows hold in the field draws the same
// glyph, the field drops its renderings that draw one, unless every one does. And a table short of width steps down:
// while the columns do not fit, as grid.Fits says, each column, from the rightmost leftward and a step each in turn,
// takes its next rendering that is narrower, until they fit or none can step. The column that wraps takes what is left,
// as it always does, and never steps.
func automatic(fields []placed, vs map[string][]value, width int) map[string]string {
	out, ladders := map[string]string{}, map[string][]string{}
	for _, f := range fields {
		if slices.Contains(f.Renderings, f.pinned) {
			out[f.ID] = f.pinned
			continue
		}
		ladder := f.Renderings
		if plain := slices.DeleteFunc(slices.Clone(ladder), func(r string) bool { return glyphed[r] }); len(plain) > 0 &&
			f.sharesGlyph(vs[f.ID]) {
			ladder = plain
		}
		out[f.ID], ladders[f.ID] = ladder[0], ladder[1:]
	}
	needs := map[string]int{}
	need := func(f placed, rendering string) int { return grid.Need(f.draw(vs[f.ID], rendering), f.name) }
	wraps := false
	for _, f := range fields {
		switch {
		case !f.column:
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
			if !f.column || f.Wrap {
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

// sharesGlyph says whether every value the rows hold in a field draws the same glyph, so the glyph says nothing the
// value does not: a row that holds none draws none, and does not count, and with no value there is none to share.
func (f field) sharesGlyph(vs []value) bool {
	seen := ""
	for _, v := range vs {
		for _, r := range v.Raw {
			g := f.glyphOf(r)
			if g == "" || seen != "" && g != seen {
				return false
			}
			seen = g
		}
	}
	return seen != ""
}
