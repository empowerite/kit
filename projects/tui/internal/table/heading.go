// The heading over the groups stands on the control line, the pane's first, apart from the rows: it names the columns
// the rows group by, and a click on a name drops the column's own menu around it, as a name in the header does. The ▾
// or ▸ before a column's name opens or closes its level, and the one after the last name the rows under the groups, as
// ToggleLevel says: a level closed hides everything under it, and each level and each heading keeps its own state, so
// opening it again shows what was there as it was. A value's heading folds by its own ▾ or ▸, as any row does.
package table

import (
	"maps"
	"slices"
	"strings"

	"github.com/charmbracelet/x/ansi"
	"github.com/empowerite/kit/projects/tui/internal/grid"
)

// BarLeft is the screen column the heading over the groups starts on, on the control line: past the gutter's two
// cells, as a heading's mark takes them.
const BarLeft = 2

// BarText gives the heading over a table's groups, as internal/grid draws it among the rows, with its styles; nothing
// where the rows group by nothing.
func BarText(rows []grid.Row) string {
	if len(rows) > 0 && grid.OverGroups(rows[0]) {
		return rows[0].Cells[0].Values[0]
	}
	return ""
}

// NameSpot is a column's name in the heading over the groups, and the screen columns it stands on, from its first to
// the one past it.
type NameSpot struct {
	Column      string
	From, Until int
}

// BarNames gives the columns the heading over the groups names, each where the control line draws its name, as
// NamesIn finds them.
func BarNames(h Host) []NameSpot { return NamesIn(ansi.Strip(BarText(h.Rows())), h.Panel().Groups()) }

// NamesIn gives where the heading over the groups, its text plain, names each of the groups' columns on the control
// line, from BarLeft: before the next level's fold, the count, or the heading's end, so a value that holds a column's
// name is passed over.
func NamesIn(text string, groups []string) []NameSpot {
	var out []NameSpot
	for _, column := range groups {
		for from := 0; ; {
			k := strings.Index(text[from:], " "+column)
			if k < 0 {
				break
			}
			at := from + k + 1
			if end := at + len(column); end == len(text) || strings.HasPrefix(text[end:], "  ") ||
				strings.HasPrefix(text[end:], " "+grid.Unfolded) || strings.HasPrefix(text[end:], " "+grid.Folded) {
				x := BarLeft + ansi.StringWidth(text[:at])
				out = append(out, NameSpot{column, x, x + ansi.StringWidth(column)})
				break
			}
			from = at
		}
	}
	return out
}

// Stop is where the keyboard stands on the heading over the groups: a group's name, with the fold before it, which
// drops its column's menu; or the rows', the fold after the last name and the count after it. Level is the level its
// fold opens or closes, from 0 the outermost, the rows' under the last group; From and Until the screen columns it
// takes, from its first to the one past it.
type Stop struct {
	Column      string // the group's column, "" for the rows
	Level       int
	From, Until int
}

// Stops gives the stops of the heading over the groups, left to right: each group's name, then the rows'; none where
// the rows group by nothing.
func Stops(h Host) []Stop {
	names, groups := BarNames(h), h.Panel().Groups()
	if len(names) == 0 {
		return nil
	}
	var out []Stop
	for _, n := range names {
		out = append(out, Stop{n.Column, slices.Index(groups, n.Column), n.From - 2, n.Until})
	}
	last, end := names[len(names)-1], BarLeft+ansi.StringWidth(ansi.Strip(BarText(h.Rows())))
	return append(out, Stop{"", len(groups), last.Until + 1, end})
}

// NameAt gives the column's name in the heading over the groups that the screen column x falls on, and whether one
// does.
func NameAt(h Host, x int) (NameSpot, bool) {
	for _, n := range BarNames(h) {
		if x >= n.From && x < n.Until {
			return n, true
		}
	}
	return NameSpot{}, false
}

// LevelAt gives the level of the groups, from 0 the outermost, whose ▾ or ▸ before its column's name in the heading
// over the groups the screen column x falls on, or the rows' level, under the last group, whose ▾ or ▸ follows the
// last name; and whether one does.
func LevelAt(h Host, x int) (int, bool) {
	names, groups := BarNames(h), h.Panel().Groups()
	for _, n := range names {
		if x == n.From-2 {
			return slices.Index(groups, n.Column), true
		}
	}
	if len(names) > 0 && x == names[len(names)-1].Until+1 {
		return len(groups), true
	}
	return 0, false
}

// ToggleLevel opens or closes a level of the groups, from 0 the outermost, the rows' under the last, as a click on its
// ▾ or ▸ asks. From ▾ it closes the level alone, which hides everything under it; from ▸, shown where it or a level
// over it is closed, it opens the level and every level over it, so it shows. Nothing else changes: each heading
// stays open or closed as it was, so a level shows again as it was left. The cursor stays on its row, or the nearest
// heading over it that shows, as Reoutline says.
func ToggleLevel(h Host, depth int) {
	s := h.Table()
	next := maps.Clone(s.Open)
	if s.Open.Shows(depth) {
		next[grid.LevelKey(depth)] = false
	} else {
		for d := 0; d <= depth; d++ {
			delete(next, grid.LevelKey(d))
		}
	}
	Reoutline(h, next)
}
