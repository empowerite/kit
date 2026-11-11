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
	"unicode"

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
			// a name ends at the line's end, the gap after it, the fold or the separator after it, or, where the rows'
			// fold is not drawn, the count
			if end := at + len(column); end == len(text) || strings.HasPrefix(text[end:], "  ") ||
				strings.HasPrefix(text[end:], " "+grid.Unfolded) || strings.HasPrefix(text[end:], " "+grid.Folded) ||
				strings.HasPrefix(text[end:], " "+grid.Joined) || len(text) > end+1 && text[end] == ' ' &&
				unicode.IsDigit(rune(text[end+1])) {
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

// Stops gives the stops of the heading over the groups, left to right: each group's name, then the rows' where their
// fold is drawn; the rows' alone, their fold and count, where the rows group by nothing; none where there is no
// heading, as over no rows. A name whose fold is a dim separator, as its level folds as the first does, folds the
// first level.
func Stops(h Host) []Stop {
	text := ansi.Strip(BarText(h.Rows()))
	if text == "" {
		return nil
	}
	names, groups := BarNames(h), h.Panel().Groups()
	if len(names) == 0 {
		return []Stop{{"", len(groups), BarLeft, BarLeft + ansi.StringWidth(text)}}
	}
	at := func(x int) string { return ansi.Cut(text, x-BarLeft, x-BarLeft+1) }
	var out []Stop
	for _, n := range names {
		level := slices.Index(groups, n.Column)
		if at(n.From-2) == grid.Joined {
			level = 0
		}
		out = append(out, Stop{n.Column, level, n.From - 2, n.Until})
	}
	last, end := names[len(names)-1], BarLeft+ansi.StringWidth(text)
	if g := at(last.Until + 1); g != grid.Unfolded && g != grid.Folded {
		return out
	}
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
// last name; and whether one does. A dim separator in a fold's place, where the level folds as the one before does,
// is no fold, and neither is the room of a rows' fold the heading does not draw.
func LevelAt(h Host, x int) (int, bool) {
	names, groups := BarNames(h), h.Panel().Groups()
	if g := ansi.Cut(ansi.Strip(BarText(h.Rows())), x-BarLeft, x-BarLeft+1); g != grid.Unfolded && g != grid.Folded {
		return 0, false
	}
	for _, n := range names {
		if x == n.From-2 {
			return slices.Index(groups, n.Column), true
		}
	}
	if len(names) > 0 && x == names[len(names)-1].Until+1 || len(names) == 0 && x == BarLeft {
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

// BarSpots gives the targets under each name of the heading over the groups, on the rule under it, as SortSpots gives
// a column's under the header: a list's FlattenGlyph, dim, which makes it a column again, flattened, a row per value;
// after it, or from the name's first cell, the sort's mark, which orders the groups, or Sortable where they can sort
// and do not. A name whose column neither sorts nor holds a list has the rule alone.
func BarSpots(h Host) []SortSpot {
	p := h.Panel()
	var out []SortSpot
	for _, name := range BarNames(h) {
		if !p.Sorts(name.Column) && !p.Lists(name.Column) {
			continue
		}
		s := SortSpot{Column: name.Column, At: name.From}
		if p.Lists(name.Column) {
			s.Flat = FlattenGlyph
		}
		if p.Sorts(name.Column) {
			if s.Mark = p.Mark(name.Column); s.Mark != "" {
				s.Sorts = true
			} else {
				s.Mark = Sortable
			}
		}
		s.Until = max(name.Until, s.At+ansi.StringWidth(s.Flat+s.Mark))
		out = append(out, s)
	}
	return out
}

// BarRule gives the rule under the heading over the groups, on the line under the control line: blank to the first
// group's name, past its fold, as the rule under the header starts at a column's name, or to BarLeft where the heading
// names no group; then a thin dim line to the heading's end, the marks of BarSpots on it; nothing where the rows group
// by nothing.
func BarRule(h Host) string {
	text := ansi.Strip(BarText(h.Rows()))
	if text == "" {
		return ""
	}
	from, end := BarLeft, BarLeft+ansi.StringWidth(text)
	if names := BarNames(h); len(names) > 0 {
		from = names[0].From
	}
	return strings.Repeat(" ", from) + RuleFill(BarSpots(h), from, end-from)
}

// BarRuleClick takes a plain click at the screen column x on the rule under the heading over the groups, and says
// whether it took it: on a sort's mark it turns the groups' sort, as SortBy does, a shift-click off; on a list's × the
// list leaves the groups for a column, flattened. Nothing elsewhere.
func BarRuleClick(h Host, x int, shift bool) bool {
	switch column, flatten := SortAt(BarRule(h), BarSpots(h), x); {
	case flatten:
		h.SetPanel(h.Panel().Grouped(column, true).Flattened(column, true))
	case column != "":
		SortBy(h, column, shift)
	default:
		return false
	}
	return true
}
