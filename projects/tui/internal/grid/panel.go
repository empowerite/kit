// The panel's controls, and the rows they let through.
package grid

import (
	"cmp"
	"maps"
	"slices"
	"strconv"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/empowerite/kit/projects/tui/internal/shell"
	"github.com/empowerite/kit/projects/tui/internal/style"
)

// Show is a choice of whose rows a panel lets through: every one, the caller's, or everyone else's. It is the filter
// on the field that owns the rows: none, the caller's login alone, or anyone's but the caller's, as views.json, the
// presets' file before presets.json, said it.
type Show int

// The choices of Show, in the panel's order.
const (
	All Show = iota
	Mine
	Others
)

// The glyphs the panel and a field's popover draw: a field the rows are grouped by, the search, a checkbox off and on,
// and a radio button off and on. Each is one cell wide in every terminal, as Unicode fixes its width, but the search,
// which is two.
const (
	Grouping = "▤"
	Search   = shell.Find
	Unticked = "☐"
	Ticked   = "☑"
	Unpicked = "⦾"
	Picked   = "⦿"
)

// Placement is where a field stands: a column, a group the rows stand under, or hidden.
type Placement int

// The placements, a column the one a field takes where the panel says none.
const (
	Column Placement = iota
	Group
	Hidden
)

// Direction is how a field sorts the rows: not at all, the smaller first, or the larger first.
type Direction int

// The directions, none the one a field takes where the panel says none.
const (
	Unsorted Direction = iota
	Ascending
	Descending
)

// Field is what a panel says of one field, three things apart. Its placement: a column, a group at a Depth, from 1, the
// outermost first, or hidden. Position is where it stands among the named columns, from 1, those with none after them
// in the stage's order; a field grouped or hidden keeps its own, so it stands where it was once it is a column again.
// Its sort: none, or a direction at a Priority, from 1, the first key first; sorting a grouped field orders its groups.
// Its filter, which a row's raw values in it must pass. And the rendering it pins, which draws it, none for the
// one the rows and the width choose.
type Field struct {
	Placement Placement
	Position  int
	Depth     int
	Sort      Direction
	Priority  int
	Filter    Filter
	Rendering string
}

// Filter is a field's filter: the values a row must hold one of, or, with Exclude, none of; with no values, it lets
// every row through. Among them, Caller stands for the caller's login, and NoValue for a row that holds none.
type Filter struct {
	Values  []string
	Exclude bool
}

// Caller and NoValue are the values a person's filter names the caller and no one by.
const (
	Caller  = "@me"
	NoValue = "none"
)

// Panel is what a screen sets of a grid's rows: the search, Query, and what it says of each field, which the columns
// dialog lists in three sections, the groups, the columns, and the hidden, as Listed gives them.
// Columns names the columns by their fields' ids, in the stage's order, and Labels gives the label each is listed by,
// its id where it gives none. Fields holds what the panel says of each field, by id, as Field says; a field it
// does not name is a column, unsorted and unfiltered. The panel never changes the map it was given, but gives a new
// one. Fold says whether a view, as it loads, groups the rows by each column whose value is the same on every row, as
// AutoGrouped says, where it groups by none. Glyphs says whether a glyph every row shown shares is drawn still, where
// the screen would drop it as saying nothing. Sortable and Groupable name the columns that can sort and group, which the
// stage says. Owner is the field that names the people the rows belong to, none when they belong to nobody; a filter's
// Caller is the caller's login, Me, which matches no one while it is unknown. Focused says whether the search has the
// keyboard. Recall says whether there are searches to recall, which the search, with the keyboard, offers with its
// Recall glyph. Outline says which groups' headings stand open, as FoldAway folds the rows of those that do not; the
// panel only reads it.
type Panel struct {
	Query     string
	Columns   []string
	Labels    map[string]string
	Fields    map[string]Field
	Fold      bool
	Glyphs    bool
	Sortable  map[string]bool
	Groupable map[string]bool
	Recall    bool
	Outline   Outline
	Owner     string
	Me        string
	Focused   bool
}

// SortKey is a column the rows sort by, and whether the larger come first.
type SortKey struct {
	Column string
	Desc   bool
}

// Field gives what the panel says of a field, by id.
func (p Panel) Field(id string) Field { return p.Fields[id] }

// with gives the panel with what it says of the fields changed as change says, in a map of its own.
func (p Panel) with(change func(fields map[string]Field)) Panel {
	fields := maps.Clone(p.Fields)
	if fields == nil {
		fields = map[string]Field{}
	}
	change(fields)
	p.Fields = fields
	return p
}

// ranked gives the ids of the fields rank gives a rank to, from 1, by it, the lower first.
func (p Panel) ranked(rank func(Field) int) []string {
	var out []string
	for id, f := range p.Fields {
		if rank(f) > 0 {
			out = append(out, id)
		}
	}
	slices.SortFunc(out, func(a, b string) int {
		return cmp.Or(cmp.Compare(rank(p.Fields[a]), rank(p.Fields[b])), strings.Compare(a, b))
	})
	return out
}

// Groups gives the fields the rows group by, the outermost first.
func (p Panel) Groups() []string {
	return p.ranked(func(f Field) int {
		if f.Placement == Group {
			return max(f.Depth, 1)
		}
		return 0
	})
}

// SortKeys gives the fields the rows sort by, the first first.
func (p Panel) SortKeys() []SortKey {
	var out []SortKey
	for _, id := range p.ranked(func(f Field) int {
		if f.Sort != Unsorted {
			return max(f.Priority, 1)
		}
		return 0
	}) {
		out = append(out, SortKey{id, p.Fields[id].Sort == Descending})
	}
	return out
}

// sortedBy gives the panel sorting the rows by the keys alone, in their order.
func (p Panel) sortedBy(keys []SortKey) Panel {
	return p.with(func(fields map[string]Field) {
		for id, f := range fields {
			f.Sort, f.Priority = Unsorted, 0
			fields[id] = f
		}
		for k, s := range keys {
			f := fields[s.Column]
			f.Sort, f.Priority = Ascending, k+1
			if s.Desc {
				f.Sort = Descending
			}
			fields[s.Column] = f
		}
	})
}

// groupedBy gives the panel grouping the rows by the fields alone, the outermost first; a field it no longer groups
// by is a column again.
func (p Panel) groupedBy(ids []string) Panel {
	return p.with(func(fields map[string]Field) {
		for id, f := range fields {
			if f.Placement == Group {
				f.Placement, f.Depth = Column, 0
				fields[id] = f
			}
		}
		for k, id := range ids {
			f := fields[id]
			f.Placement, f.Depth = Group, k+1
			fields[id] = f
		}
	})
}

// Placed gives the panel with a field a column or hidden; a group it leaves, and those under it move out a level.
func (p Panel) Placed(id string, at Placement) Panel {
	p = p.groupedBy(slices.DeleteFunc(p.Groups(), func(g string) bool { return g == id }))
	return p.with(func(fields map[string]Field) {
		f := fields[id]
		f.Placement = at
		fields[id] = f
	})
}

// Showing gives the choice of Show the filter on the field that owns the rows says: none lets every row through, the
// caller alone the caller's, and anyone but the caller everyone else's; -1 for another filter, which no choice says.
func (p Panel) Showing() Show {
	f := p.Fields[p.Owner].Filter
	switch {
	case len(f.Values) == 0:
		return All
	case !slices.Equal(f.Values, []string{Caller}):
		return -1
	case f.Exclude:
		return Others
	}
	return Mine
}

// Shown gives the panel letting through the rows a choice of Show says, by the filter on the field that owns them.
func (p Panel) Shown(s Show) Panel {
	if p.Owner == "" {
		return p
	}
	var filter Filter
	if s != All {
		filter = Filter{Values: []string{Caller}, Exclude: s == Others}
	}
	return p.with(func(fields map[string]Field) {
		f := fields[p.Owner]
		f.Filter = filter
		fields[p.Owner] = f
	})
}

// Filtering says whether a filter on any field narrows the rows.
func (p Panel) Filtering() bool {
	for _, f := range p.Fields {
		if len(f.Filter.Values) > 0 {
			return true
		}
	}
	return false
}

// Sorted gives the panel with the column's sort turned, as an option-click on its name asks: a column not sorting sorts
// ascending, as one more key after the others, then descending in its place, then not at all. A column that cannot
// sort changes nothing.
func (p Panel) Sorted(column string) Panel {
	if !p.Sortable[column] {
		return p
	}
	keys := p.SortKeys()
	k := slices.IndexFunc(keys, func(s SortKey) bool { return s.Column == column })
	switch {
	case k < 0:
		keys = append(keys, SortKey{Column: column})
	case !keys[k].Desc:
		keys[k].Desc = true
	default:
		keys = slices.Delete(keys, k, k+1)
	}
	return p.sortedBy(keys)
}

// SortedAs gives the panel with a field sorting the rows as d says, as its popover asks: a field already among the keys
// keeps its place among them, in the new direction, or leaves them for Unsorted; one not among them comes after the
// others. A field that cannot sort changes nothing.
func (p Panel) SortedAs(id string, d Direction) Panel {
	if !p.Sortable[id] {
		return p
	}
	keys := p.SortKeys()
	k := slices.IndexFunc(keys, func(s SortKey) bool { return s.Column == id })
	switch {
	case d == Unsorted && k >= 0:
		keys = slices.Delete(keys, k, k+1)
	case d == Unsorted:
	case k >= 0:
		keys[k].Desc = d == Descending
	default:
		keys = append(keys, SortKey{Column: id, Desc: d == Descending})
	}
	return p.sortedBy(keys)
}

// Unsorted gives the panel with no column sorting the rows within their groups: the keys of the columns gone, the
// groups keeping how they sort.
func (p Panel) Unsorted() Panel {
	groups := p.Groups()
	return p.sortedBy(slices.DeleteFunc(p.SortKeys(), func(s SortKey) bool { return !slices.Contains(groups, s.Column) }))
}

// FilteredBy gives the panel with the filter on a field, by id, as f says.
func (p Panel) FilteredBy(id string, f Filter) Panel {
	return p.with(func(fields map[string]Field) {
		g := fields[id]
		g.Filter = Filter{Values: slices.Clone(f.Values), Exclude: f.Exclude}
		fields[id] = g
	})
}

// Rendered gives the panel with a field drawn by the rendering named, or, for none, by the one the rows and the width
// choose.
func (p Panel) Rendered(id, rendering string) Panel {
	return p.with(func(fields map[string]Field) {
		g := fields[id]
		g.Rendering = rendering
		fields[id] = g
	})
}

// Place gives where a field stands among what sorts the rows, from 1, or 0 where it sorts nothing, and how many
// things sort them: the groups that sort first, as the rows stand in their groups before any key sorts them, the
// outermost 1; then the keys of the columns the rows do not group by, the first first. A group that does not sort, as
// a group does not by default, takes no place, so the keys number on from the groups that do.
func (p Panel) Place(id string) (place, total int) {
	groups := slices.DeleteFunc(p.Groups(), func(g string) bool { return p.Fields[g].Sort == Unsorted })
	keys := p.ColumnKeys()
	total = len(groups) + len(keys)
	if k := slices.Index(groups, id); k >= 0 {
		return k + 1, total
	}
	if k := slices.IndexFunc(keys, func(s SortKey) bool { return s.Column == id }); k >= 0 {
		return len(groups) + k + 1, total
	}
	return 0, total
}

// ColumnKeys gives the keys that sort the rows within their groups: those of the fields the rows do not group by, the
// first first.
func (p Panel) ColumnKeys() []SortKey {
	groups := p.Groups()
	return slices.DeleteFunc(p.SortKeys(), func(s SortKey) bool { return slices.Contains(groups, s.Column) })
}

// Mark gives what stands after a field's name where it sorts the rows: ▲ ascending or ▼ descending, and its place
// among what orders them, as Place gives it, when more than one thing does, raised after ▲ and lowered after ▼, as
// each points, ▲² and ▼₂; nothing where it does not sort. A group sorts its rows' values as its field's sort says,
// ascending where it says none, and is marked only where it says one.
func (p Panel) Mark(id string) string {
	f := p.Fields[id]
	place, total := p.Place(id)
	if f.Sort == Unsorted || place == 0 {
		return ""
	}
	mark, digits := "▲", superscript
	if f.Sort == Descending {
		mark, digits = "▼", subscript
	}
	if total > 1 {
		mark += digits(place)
	}
	return mark
}

// GroupMark gives what stands for a field the rows are grouped by at a depth: the grouping glyph and the depth as a
// subscript, ▤₁ for the outermost, as Mark carries a sort's place.
func GroupMark(depth int) string { return Grouping + subscript(depth) }

// subscript gives a number in subscript digits, as ₁₂, which mark a group's depth and a descending
// column's place among the keys.
func subscript(n int) string {
	return strings.Map(func(r rune) rune { return '₀' + r - '0' }, strconv.Itoa(n))
}

// superscript gives a number in superscript digits, as ¹², which mark an ascending column's place among the keys.
// Their code points are not in a run, as subscript's are: ¹ ² ³ stand in Latin-1.
func superscript(n int) string {
	digits := []rune("⁰¹²³⁴⁵⁶⁷⁸⁹")
	return strings.Map(func(r rune) rune { return digits[r-'0'] }, strconv.Itoa(n))
}

// Grouped gives the panel grouping the rows by the column alone, or, with nest, as the column menu's Group and a
// ctrl-click on a name ask, within the groups it has, innermost; a column it groups by already, it no longer does,
// alone or among the others, and it is a column again. A column that cannot group changes nothing.
func (p Panel) Grouped(column string, nest bool) Panel {
	if !p.Groupable[column] {
		return p
	}
	groups := p.Groups()
	k := slices.Index(groups, column)
	switch {
	case k >= 0 && nest:
		groups = slices.Delete(groups, k, k+1)
	case k >= 0 && len(groups) == 1:
		groups = nil
	case nest:
		groups = append(groups, column)
	default:
		groups = []string{column}
	}
	return p.groupedBy(groups)
}

// Levels gives how many levels of headings the rows stand under, as group draws them, which a screen indents them by:
// one each column they group by, the heading over the groups apart, which a screen draws apart from them; none where
// nothing groups them.
func Levels(rows []Row) int {
	n := 0
	for _, r := range rows {
		if Heads(r) {
			n = max(n, strings.Count(r.Key, "\x00")-1)
		}
	}
	return n
}

// byFields is the key of the heading over the groups, which names their columns, and which every heading of a value
// keys under.
const byFields = "group\x00\x01"

// OverGroups says whether a row is the heading over the groups, which names their columns.
func OverGroups(r Row) bool { return r.Key == byFields }

// Heads says whether a row is the heading of a group, as group draws one.
func Heads(r Row) bool { return r.Aside && strings.HasPrefix(r.Key, "group\x00") }

// Heading gives the column whose group a row heads, as group draws the headings, a value of it; or nothing for a row
// that heads no group, or the heading over the groups, which names them all.
func (p Panel) Heading(r Row) string {
	if !Heads(r) || !strings.HasPrefix(r.Key, byFields+"\x00") {
		return ""
	}
	if groups, depth := p.Groups(), Level(r)-2; depth < len(groups) {
		return groups[depth]
	}
	return ""
}

// group gives the rows under one heading that names the columns the panel groups by, bold as the header names them, the
// outermost first, each after its level's fold, ▾ while the level shows, as the outline's Shows says, ▸ while it does
// not, never dim, as it always acts; after the last, the rows' own, ▾ while they show; and how many rows they group,
// dim. A screen draws that heading apart from the rows, which draw it as nothing, and it holds what a closed level
// hides. Under it stand a heading per value of each column, the groups in their values' order, the larger first where
// the column sorts descending, and the rows in each as they came: a heading of a value, an aside, is the value, as the
// rows draw it or as their cell's Head says it, indented by its depth, the outermost not at all, and how many rows it
// heads, dim. A row groups by a cell's sort key, or under none when the cell has none; a cell of several values groups
// the row under each, the same row under every heading it belongs to, nested under each pair of its values, each
// heading over it counting it once. A row of one cell, as a heading, has no value: those follow the groups. Where the
// panel groups by nothing, the rows stand as they came.
func (p Panel) group(rows []Row, columns []string) []Row {
	var at []int
	var desc []bool
	for _, name := range p.Groups() {
		if j := slices.Index(columns, name); j >= 0 {
			at, desc = append(at, j), append(desc, p.Fields[name].Sort == Descending)
		}
	}
	var grouped, rest []Row
	for _, r := range rows {
		if len(r.Cells) > 1 {
			grouped = append(grouped, r)
		} else {
			rest = append(rest, r)
		}
	}
	if len(at) == 0 {
		return rows
	}
	// an entry is a row under one value of each column, as many entries as the pairs of its values
	type entry struct {
		row    Row
		values []string // the raw values, a depth each
		heads  []string // as the headings say them
	}
	var entries []entry
	for _, r := range grouped {
		partial := []entry{{row: r}}
		for _, j := range at {
			var c Cell
			if j < len(r.Cells) {
				c = r.Cells[j]
			}
			values, heads := c.groups()
			var next []entry
			for _, e := range partial {
				for k := range values {
					next = append(next, entry{r, append(slices.Clone(e.values), values[k]),
						append(slices.Clone(e.heads), heads[k])})
				}
			}
			partial = next
		}
		entries = append(entries, partial...)
	}
	slices.SortStableFunc(entries, func(a, b entry) int {
		for k := range at {
			if c := natural(a.values[k], b.values[k]); c != 0 {
				if desc[k] {
					return -c
				}
				return c
			}
		}
		return 0
	})
	// same says whether two entries share their values down to a depth
	same := func(a, b entry, depth int) bool { return slices.Equal(a.values[:depth+1], b.values[:depth+1]) }
	var out []Row
	if len(entries) > 0 {
		out = append(out, Row{Key: byFields, Aside: true}) // its text says each level's fold, once they are known
	}
	// a value's heading, by its index in out: its depth, how many rows it heads, and its value as it says it
	type head struct {
		depth, n int
		text     string
	}
	heads := map[int]head{}
	for i, e := range entries {
		for depth := range at {
			if i > 0 && same(entries[i-1], e, depth) {
				continue
			}
			// the rows under it, each once, though it stands under several of the values deeper
			under := map[string]bool{}
			for _, f := range entries[i:] {
				if !same(f, e, depth) {
					break
				}
				under[f.row.Key] = true
			}
			n := len(under)
			key := byFields
			for _, v := range e.values[:depth+1] {
				key += "\x00" + v
			}
			heads[len(out)] = head{depth, n, e.heads[depth]}
			out = append(out, Row{Key: key, Aside: true})
		}
		out = append(out, e.row)
	}
	// a heading whose one heading under it heads the same rows takes it into its line, as the heading over the groups
	// names its levels across, and so on down while the next does too, of a level that shows; the headings taken in
	// stand as one node, the outermost's, which every fold on the line opens and closes
	var combined []Row
	for i := 0; i < len(out); i++ {
		h, ok := heads[i]
		if !ok {
			combined = append(combined, out[i])
			continue
		}
		start, texts, last := i, []string{h.text}, h
		for {
			next, ok := heads[i+1]
			if !ok || next.depth != last.depth+1 || next.n != h.n || !p.Outline.Shows(next.depth) || !alone(len(out), i+1, func(k int) (int, bool) {
				d, ok := heads[k]
				return d.depth, ok
			}) {
				break
			}
			texts, last = append(texts, next.text), next
			i++
		}
		r := out[start]
		// the line shows what it holds while it stands open and the level under its last value shows
		fold := Unfolded
		if !p.Outline.Open(r) || !p.Outline.Shows(last.depth+1) {
			fold = style.Off(Folded)
		}
		for k := range texts {
			texts[k] = fold + " " + texts[k]
		}
		text := strings.Repeat("  ", h.depth) + strings.Join(texts, " ") + " " + style.Off(strconv.Itoa(h.n))
		r.Cells = []Cell{{Values: []string{text}, Wrap: true}}
		combined = append(combined, r)
	}
	out = combined
	if len(out) > 0 {
		// fold gives a level's glyph, from 0 the outermost, the rows' the last: never dim, as it always acts
		fold := func(depth int) string {
			if p.Outline.Shows(depth) {
				return Unfolded
			}
			return Folded
		}
		var titles []string
		for depth, j := range at {
			titles = append(titles, fold(depth)+" "+style.Label(columns[j]))
		}
		heading := strings.Join(titles, " ") + " " + fold(len(at)) + " " + style.Off(strconv.Itoa(len(grouped)))
		out[0].Cells = []Cell{{Values: []string{heading}, Wrap: true}}
	}
	return append(out, rest...)
}

// alone says whether the heading at k of n rows is the only one of its depth under the heading over it: none of its
// depth follows it before one shallower than it, or the end. depth gives the depth of the heading at an index, and
// whether the row there is a heading.
func alone(n, k int, depth func(int) (int, bool)) bool {
	d, _ := depth(k)
	for j := k + 1; j < n; j++ {
		if dj, ok := depth(j); ok && dj <= d {
			return dj < d
		}
	}
	return true
}

// Unfolded and Folded are the glyphs before a node of the outline that hides something, a group's heading or a row:
// what it hides shown, or hidden. Folded is drawn dim, as is a heading's text; Unfolded is not.
const (
	Unfolded = "▾"
	Folded   = "▸"
)

// Outline is which nodes of the rows' outline stand open, by key: a group's heading, whose rows show under it, a row,
// whose lines and detail show, and anything else a screen keys in it, as its header; and which levels of the groups
// show, by LevelKey. A key it does not hold stands as it opens: a heading open, a level showing, everything else
// closed.
type Outline map[string]bool

// Open says whether a row stands open: the heading over the groups always, as each level folds under it; any other
// group's heading unless it was closed; any other row only once opened.
func (o Outline) Open(r Row) bool {
	if OverGroups(r) {
		return true
	}
	if open, ok := o[r.Key]; ok {
		return open
	}
	return Heads(r)
}

// LevelKey gives the key that says in an outline whether a level of the groups shows, from 0 the outermost, as the
// fold before its column's name in the heading over the groups opens and closes it; the level under the last group's
// is the rows'.
func LevelKey(depth int) string { return "group\x01" + strconv.Itoa(depth) }

// Shows says whether a level of the groups shows, from 0 the outermost: open, and every level over it open too, as a
// level closed hides everything under it. Each keeps its own state, so opening the levels over it again shows it as
// it was.
func (o Outline) Shows(depth int) bool {
	for d := 0; d <= depth; d++ {
		if open, ok := o[LevelKey(d)]; ok && !open {
			return false
		}
	}
	return true
}

// Unfolds says whether a row shows what it holds: open, as Open says, and, for a group's heading, the level under it
// showing, as Shows says.
func (o Outline) Unfolds(r Row) bool {
	if !o.Open(r) {
		return false
	}
	return !Heads(r) || OverGroups(r) || o.Shows(Level(r)-1)
}

// fold gives the glyph before the heading of a key, styled: Folded, dim, while its group shows nothing under it, as
// Unfolds says, Unfolded while it does.
func (p Panel) fold(key string) string {
	if !p.Outline.Unfolds(Row{Key: key, Aside: true}) {
		return style.Off(Folded)
	}
	return Unfolded
}

// FoldAway gives the rows as group draws them with every closed group, and every level that does not show, folded
// away: under a closed heading, the rows and the headings deeper than it, up to the next heading as shallow, or the
// rows of one cell that follow the groups; and a heading, or a row under the groups, whose level does not show, as the
// outline's Shows says. The row before it stays, the heading over the groups when nothing does, and holds them in its
// Folded, which lay the columns out still.
func (p Panel) FoldAway(rows []Row) []Row {
	groups := Levels(rows)
	var out []Row
	closed := 0 // the level of the closed heading the rows stand under, or none
	for _, r := range rows {
		level := Level(r)
		// the level of the groups the row stands at, the rows' under the last, -1 for one no level hides
		depth := -1
		switch {
		case level >= 2:
			depth = level - 2
		case level == 0 && len(r.Cells) > 1 && groups > 0:
			depth = groups
		}
		hidden := depth >= 0 && !p.Outline.Shows(depth)
		if hidden || closed > 0 && (level > closed || level == 0 && len(r.Cells) > 1) {
			out[len(out)-1].Folded = append(out[len(out)-1].Folded, r)
			continue
		}
		closed = 0
		out = append(out, r)
		if level > 0 && !p.Outline.Open(r) {
			closed = level
		}
	}
	return out
}

// Level gives the level of a group's heading, from 1, the outermost; 0 for any other row.
func Level(r Row) int {
	if !Heads(r) {
		return 0
	}
	return strings.Count(r.Key, "\x00")
}

// Every gives the whole outline the rows stand for, as group drew it: each closed heading followed by the rows it folds
// away, in their places, which hold nothing folded.
func Every(rows []Row) []Row {
	var out []Row
	for _, r := range rows {
		folded := r.Folded
		r.Folded = nil
		out = append(append(out, r), Every(folded)...)
	}
	return out
}

// beneath says whether a row stands under a heading of the level in the outline, as one of its rows or a heading deeper
// than it; the rows of one cell that follow the groups stand under none.
func beneath(r Row, level int) bool {
	if Heads(r) {
		return Level(r) > level
	}
	return len(r.Cells) > 1
}

// Span gives where the subtree of the row at k ends in the rows, past its last: a heading's runs over every row and
// heading under it; any other row's is itself.
func Span(rows []Row, k int) int {
	end := k + 1
	if !Heads(rows[k]) {
		return end
	}
	for end < len(rows) && beneath(rows[end], Level(rows[k])) {
		end++
	}
	return end
}

// Parent gives the index of the heading the row at k stands under in the rows, the nearest above it shallower than it,
// or -1 for a row under none: an outermost heading, or a row that follows the groups or stands where nothing groups.
func Parent(rows []Row, k int) int {
	if !Heads(rows[k]) && len(rows[k].Cells) < 2 {
		return -1
	}
	for j := k - 1; j >= 0; j-- {
		if Heads(rows[j]) && (!Heads(rows[k]) || Level(rows[j]) < Level(rows[k])) {
			return j
		}
	}
	return -1
}

// AutoGrouped gives the panel a view takes as it loads, while it folds and groups by nothing: the rows grouped by
// each column whose value is the same on every row it lets through, as constant says, left to right, in the panel's
// order, the outermost first; a column hidden, or one that cannot group, never is. From then on they are groups as
// any other, which the person moves and ungroups as they would. A panel that does not fold, or groups already, or
// whose rows share no value, is given back as it is. columns names each cell's column, in the stage's order.
func (p Panel) AutoGrouped(rows []Row, columns []string) Panel {
	if !p.Fold || len(p.Groups()) > 0 {
		return p
	}
	query := strings.ToLower(p.Query)
	var through []Row
	for _, r := range rows {
		if !r.Aside && len(r.Cells) > 1 && p.matches(r, columns, query) {
			through = append(through, r)
		}
	}
	var walk []int
	for _, j := range p.arrangement(columns) {
		if name := columns[j]; name != "" && p.Groupable[name] && p.Shows(name) {
			walk = append(walk, j)
		}
	}
	var ids []string
	for _, j := range constant(through, walk) {
		ids = append(ids, columns[j])
	}
	if len(ids) == 0 {
		return p
	}
	return p.groupedBy(ids)
}

// constant gives those of the columns whose value is the same on every one of the rows, as their sort key says, each
// cell holding one value and none blank; none for fewer than two rows.
func constant(rows []Row, columns []int) []int {
	if len(rows) < 2 {
		return nil
	}
	var out []int
	for _, j := range columns {
		one := func(r Row) bool { return j < len(r.Cells) && len(r.Cells[j].Values) == 1 && r.Cells[j].key() != "" }
		if !one(rows[0]) {
			continue
		}
		if !slices.ContainsFunc(rows[1:], func(r Row) bool { return !one(r) || r.Cells[j].key() != rows[0].Cells[j].key() }) {
			out = append(out, j)
		}
	}
	return out
}

// order sorts the rows by the panel's keys, each in turn, the rows alike on every key keeping their order; a row of
// one cell, as a heading, stays where it is, the rows around it sorting in the places they hold.
func (p Panel) order(rows []Row, columns []string) []Row {
	sorts := p.SortKeys()
	if len(sorts) == 0 {
		return rows
	}
	var places []int
	var sorted []Row
	for i, r := range rows {
		if len(r.Cells) > 1 {
			places, sorted = append(places, i), append(sorted, r)
		}
	}
	slices.SortStableFunc(sorted, func(a, b Row) int {
		for _, s := range sorts {
			j := slices.Index(columns, s.Column)
			if j < 0 || j >= len(a.Cells) || j >= len(b.Cells) {
				continue
			}
			if c := natural(a.Cells[j].key(), b.Cells[j].key()); c != 0 {
				if s.Desc {
					return -c
				}
				return c
			}
		}
		return 0
	})
	out := slices.Clone(rows)
	for k, i := range places {
		out[i] = sorted[k]
	}
	return out
}

// offers says whether the columns dialog lists a field: one with a name among the columns, or one the rows group by.
func (p Panel) offers(id string) bool {
	return id != "" && (slices.Contains(p.Columns, id) || p.Fields[id].Placement == Group)
}

// section gives the fields the columns dialog lists under a placement, in their order: the groups by depth, the outermost
// first; the columns by position, as the rows stand them; and the hidden in the stage's order.
func (p Panel) section(at Placement) []string {
	var out []string
	switch at {
	case Group:
		return p.Groups()
	case Column:
		for _, id := range p.Arrange(p.Columns) {
			if id != "" && p.Shows(id) {
				out = append(out, id)
			}
		}
	default:
		for _, id := range p.Columns {
			if id != "" && p.Fields[id].Placement == Hidden {
				out = append(out, id)
			}
		}
	}
	return out
}

// Filtered is the mark after a field's label in the columns dialog while a filter on it holds rows back.
const Filtered = "⊘"

// Listed gives the fields the columns dialog lists, in its order: those the rows group by, the outermost first; the
// columns, as the rows stand them; then the hidden, in the stage's order.
func (p Panel) Listed() []string {
	return slices.Concat(p.section(Group), p.section(Column), p.section(Hidden))
}

// Step gives the panel with a field moved a place, up for a negative step, down for a positive one, and says whether it
// moved: within its section, a group moves a depth and a column a position; past its
// section's end, it crosses into the next, a column up into the groups' last depth, where it can group, and down into
// the hidden, a group down into the first column. A hidden field, which stands in the stage's order, moves up alone,
// to the last column.
func (p Panel) Step(id string, d int) (Panel, bool) {
	at := p.Fields[id].Placement
	list := p.section(at)
	i := slices.Index(list, id)
	switch {
	case i < 0:
		return p, false
	case at == Hidden && d < 0:
		return p.Moved(id, Column, len(p.section(Column)))
	case at == Hidden:
		return p, false
	case i+d >= 0 && i+d < len(list):
		return p.Moved(id, at, i+d)
	case d < 0 && at == Column:
		return p.Moved(id, Group, len(p.section(Group)))
	case d > 0 && at == Group:
		return p.Moved(id, Column, 0)
	case d > 0 && at == Column:
		return p.Moved(id, Hidden, 0)
	}
	return p, false
}

// Moved gives the panel with a field moved into a section at an index among the fields there, and says whether
// anything changed: into the groups, at that depth, where the field can group; into the columns, at
// that position, which every field the panel offers then has, a field hidden or grouped keeping its own among them;
// hidden, where it keeps its position. A field the columns dialog does not list never moves.
func (p Panel) Moved(id string, to Placement, k int) (Panel, bool) {
	if !p.offers(id) || to == Group && !p.Groupable[id] {
		return p, false
	}
	before := [][]string{p.section(Group), p.section(Column), p.section(Hidden)}
	without := func(ids []string) []string {
		return slices.DeleteFunc(slices.Clone(ids), func(s string) bool { return s == id })
	}
	switch to {
	case Hidden:
		p = p.Placed(id, Hidden)
	case Group:
		groups := without(p.section(Group))
		p = p.groupedBy(slices.Insert(groups, min(max(k, 0), len(groups)), id))
	default:
		columns := without(p.section(Column))
		k = min(max(k, 0), len(columns))
		order := slices.DeleteFunc(p.Arrange(p.Columns), func(s string) bool { return s == "" })
		at := slices.Index(order, id)
		order = without(order)
		switch {
		case k < len(columns):
			at = slices.Index(order, columns[k])
		case len(columns) > 0:
			at = slices.Index(order, columns[len(columns)-1]) + 1
		}
		if at >= 0 {
			order = slices.Insert(order, min(at, len(order)), id)
		}
		p = p.Placed(id, Column).with(func(fields map[string]Field) {
			for n, c := range order {
				f := fields[c]
				f.Position = n + 1
				fields[c] = f
			}
		})
	}
	after := [][]string{p.section(Group), p.section(Column), p.section(Hidden)}
	return p, !slices.EqualFunc(before, after, slices.Equal)
}

// Update takes a key while the search has the keyboard, and says whether it took it: a printable key is typed, / too,
// and backspace takes one back.
func (p Panel) Update(k tea.KeyPressMsg) (Panel, bool) {
	if k.String() != "backspace" && k.Text == "" {
		return p, false
	}
	p.Query, _ = shell.Type(p.Query, k)
	return p, true
}

// Find puts the search under the keyboard.
func (p Panel) Find() Panel {
	p.Focused = true
	return p
}

// Filter gives the rows the panel lets through: those whose text has the search, without regard to case, and whose
// raw values pass the filter on each field, sorted by its keys, grouped as group says, each with its cells in its
// order and without those of the columns it does not show. columns names each cell's column, in the stage's order.
// While the search or a filter narrows the rows, an aside that heads none of them is left out with them.
func (p Panel) Filter(rows []Row, columns []string) []Row {
	query := strings.ToLower(p.Query)
	var out []Row
	for _, r := range rows {
		if r.Aside || p.matches(r, columns, query) {
			out = append(out, r)
		}
	}
	if query != "" || p.Filtering() {
		out = dropBare(out)
	}
	out = p.group(p.order(out, columns), columns)
	for k, r := range out {
		if !r.Aside {
			out[k] = p.hide(r, columns)
		}
	}
	return out
}

// dropBare leaves out each aside that heads no row: one the next aside, or the end, follows at once.
func dropBare(rows []Row) []Row {
	var out []Row
	for i, r := range rows {
		if r.Aside && (i+1 == len(rows) || rows[i+1].Aside) {
			continue
		}
		out = append(out, r)
	}
	return out
}

// matches says whether a row passes the filter on each field, of the columns named, and has the search in its text.
func (p Panel) matches(r Row, columns []string, query string) bool {
	for id, f := range p.Fields {
		if len(f.Filter.Values) == 0 {
			continue
		}
		var raw []string
		if j := slices.Index(columns, id); j >= 0 && j < len(r.Cells) {
			raw = r.Cells[j].Raw
		}
		if !p.passes(f.Filter, raw) {
			return false
		}
	}
	if query == "" {
		return true
	}
	for _, line := range append([][]Cell{r.Cells}, r.Detail...) {
		for _, c := range line {
			for _, v := range c.Values {
				if strings.Contains(strings.ToLower(ansi.Strip(v)), query) {
					return true
				}
			}
		}
	}
	return false
}

// passes says whether a row's raw values in a field pass its filter: one of them among the filter's values, or, for a
// row that holds none, NoValue among them, and Caller the caller's login, which is no one's while it is unknown; with
// Exclude, none of them.
func (p Panel) passes(f Filter, raw []string) bool {
	in := slices.ContainsFunc(f.Values, func(v string) bool {
		switch v {
		case Caller:
			return p.Me != "" && slices.Contains(raw, p.Me)
		case NoValue:
			return len(raw) == 0
		}
		return slices.Contains(raw, v)
	})
	return in != f.Exclude
}

// hide gives the row with its cells in the panel's order, without the cells of the columns it does not show; a row of
// one cell spans the columns, and keeps it.
func (p Panel) hide(r Row, columns []string) Row {
	if len(r.Cells) < 2 || len(p.Fields) == 0 {
		return r
	}
	var cells []Cell
	for _, j := range p.arrangement(columns) {
		if j < len(r.Cells) && (j >= len(columns) || p.Shows(columns[j])) {
			cells = append(cells, r.Cells[j])
		}
	}
	for j := len(columns); j < len(r.Cells); j++ {
		cells = append(cells, r.Cells[j])
	}
	r.Cells = cells
	return r
}

// Shows says whether the rows show a field as a column: one hidden does not, nor one grouped, whose value each group's
// heading names.
func (p Panel) Shows(column string) bool { return p.Fields[column].Placement == Column }

// arrangement gives the order the columns stand in, as indexes into them: a column without a name keeps its place, and
// the named ones fill the other places, those with a position first, by it, then the rest, in their order.
func (p Panel) arrangement(columns []string) []int {
	var named []int
	for j, name := range columns {
		if name != "" && p.Fields[name].Position > 0 {
			named = append(named, j)
		}
	}
	slices.SortStableFunc(named, func(a, b int) int {
		return cmp.Compare(p.Fields[columns[a]].Position, p.Fields[columns[b]].Position)
	})
	for j, name := range columns {
		if name != "" && p.Fields[name].Position == 0 {
			named = append(named, j)
		}
	}
	out := make([]int, len(columns))
	for j, name := range columns {
		if name == "" {
			out[j] = j
			continue
		}
		out[j], named = named[0], named[1:]
	}
	return out
}

// Arrange gives the columns' names in the panel's order, as the rows' cells stand once it lets them through.
func (p Panel) Arrange(columns []string) []string {
	out := make([]string, len(columns))
	for k, j := range p.arrangement(columns) {
		out[k] = columns[j]
	}
	return out
}
