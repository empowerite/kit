// A column's card, which a click on its name opens, takes what the column menu did: the column's sort, a radio,
// Ascending or Descending, a column that did not sort coming after the keys there are, one that did keeping its place,
// or Unsorted, which takes it out of the keys, as a shift-click on its sort does; for a column that holds a list, the
// rows standing once per value of it, or once with all of them; the rows grouped by it too, innermost, or no longer;
// and the column hidden. Many terminals keep option, ctrl, and the right button with a click for their own selection
// and menu, so the card holds what those would do. Shift with left or right moves the column a place among those the
// header shows, the card with it, as Shifted says. Here are the words of those lines and how a column moves.
package table

import (
	"slices"

	"github.com/empowerite/kit/projects/tui/internal/grid"
	"github.com/empowerite/kit/projects/tui/internal/shell"
)

// The words of the card's lines that came from the column menu, and the glyphs the header and the legend draw for
// a column's flattening: ×, a row crossed with its values, a row a pair, as a cross product is, and ÷, the rows
// divided back into one; and a sweep, which a menu line that clears draws.
const (
	Unsorted = "Unsorted"
	BySize   = " by size"
	GroupIt  = "Group"
	ShowIt   = "Show"

	Sweep        = "🧹"
	FlattenGlyph = "×"
	GatherGlyph  = "÷"
)

// FlattenIt gives the words of the line that flattens a column that holds a list, by the name of one of its values.
func FlattenIt(one string) string { return "One row per " + one }

// GatherIt gives the words of the line that gathers a column's values onto one row again, by its label.
func GatherIt(label string) string { return "All " + label + " on one row" }

// Shifted gives the panel with a column moved a place, left for a negative step, right for a positive one, among those
// it stands with: a column among the columns the header names, past those no row fills and those the rows group by on
// their own; a group among the groups, a depth outward or inward. It says whether it moved; at either end it does not,
// nor does it cross into the groups or the hidden.
func Shifted(h Host, column string, d int) (grid.Panel, bool) {
	p := h.Panel()
	placement := p.Field(column).Placement
	order := func(q grid.Panel) []string { return q.Groups() }
	switch placement {
	case grid.Column:
		// those the header names: not one the rows group by on their own, whose value their heading says
		shown := slices.DeleteFunc(slices.Clone(h.ShownColumns()), func(id string) bool {
			_, name := h.HeaderName(id)
			return name == ""
		})
		order = func(q grid.Panel) []string {
			return slices.DeleteFunc(q.Listed(), func(id string) bool {
				return q.Field(id).Placement != grid.Column || !slices.Contains(shown, id)
			})
		}
	case grid.Hidden:
		return p, false
	}
	before := order(p)
	for q := p; ; {
		var ok bool
		if q, ok = q.Step(column, d); !ok || q.Field(column).Placement != placement {
			return p, false
		}
		if !slices.Equal(order(q), before) {
			return q, true
		}
	}
}

// ShiftStep gives the step shift with left or right asks for, and whether the key is one.
func ShiftStep(key string) (int, bool) {
	switch key {
	case "shift+left":
		return -1, true
	case "shift+right":
		return 1, true
	}
	return 0, false
}

// ShiftKeys gives the binding of shift with left and right on a menu that moves a column, each where it moves it.
func ShiftKeys(h Host, column string) []shell.Binding {
	var presses []string
	for _, key := range []string{"shift+left", "shift+right"} {
		if d, _ := ShiftStep(key); func() bool { _, ok := Shifted(h, column, d); return ok }() {
			presses = append(presses, key)
		}
	}
	if len(presses) == 0 {
		return nil
	}
	return []shell.Binding{shell.Bind("⇧←→", "move", presses...)}
}
