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

	"github.com/empowerite/kit/projects/tui/internal/pane"
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

// Panel is the controls beside a grid's rows, top to bottom: the search field; the field editor, each field with a name
// in one of three sections, the groups, the columns, and the hidden, as Lines draws them; whether the panel folds; and
// whether it shows the emoji every row shares.
// Columns names the columns by their fields' ids, in the stage's order, and Labels gives the label the editor lists
// each by, its id where it gives none. Fields holds what the panel says of each field, by id, as Field says; a field it
// does not name is a column, unsorted and unfiltered. The panel never changes the map it was given, but gives a new
// one. Fold says whether a value every row of a group shares is said once, in the group's heading, rather than in each
// row: the rows then group on their own, as group says, by each groupable column whose value is the same on every row
// of a group. Glyphs says whether a glyph every row shown shares is drawn still, where the screen would drop it as
// saying nothing; Auto names the columns that do so now, as Filtered gave them, and AutoAt the level of the headings each
// is said at, from 1, as Said gives it. Sortable and Groupable name the columns that can sort and group, which the
// stage says. Owner is the field that names the people the rows belong to, none when they belong to nobody; a filter's
// Caller is the caller's login, Me, which matches no one while it is unknown. Cursor is the control under the
// keyboard, Focused whether the panel has it. Recall says whether there are searches to recall, which the search, with
// the keyboard, offers with its Recall glyph. Outline says which groups' headings stand open, as FoldAway folds the
// rows of those that do not; the panel only reads it. Fill is the cells the control under the keyboard is drawn inverse
// across, at least.
type Panel struct {
	Query     string
	Columns   []string
	Labels    map[string]string
	Fields    map[string]Field
	Fold      bool
	Glyphs    bool
	Sortable  map[string]bool
	Groupable map[string]bool
	Auto      []string
	AutoAt    map[string]int
	Recall    bool
	Outline   Outline
	Fill      int
	Owner     string
	Me        string
	Cursor    int
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

// Sorted gives the panel with the column sorting the rows alone, as a click on its name asks, or, with also, as one
// more key after the others, as an option-click does: a column not sorting sorts ascending, then descending, then not at
// all. A column that cannot sort changes nothing.
func (p Panel) Sorted(column string, also bool) Panel {
	if !p.Sortable[column] {
		return p
	}
	sorts := p.SortKeys()
	k := slices.IndexFunc(sorts, func(s SortKey) bool { return s.Column == column })
	keys := slices.Clone(sorts)
	if !also {
		keys = nil
		if k >= 0 {
			keys = []SortKey{sorts[k]}
		}
		k = min(k, 0)
	}
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

// Mark gives what stands after a column's name where it sorts the rows: ▲ ascending or ▼ descending, and its place
// among the keys, as a subscript, when there are several; nothing where it does not.
func (p Panel) Mark(column string) string {
	sorts := p.SortKeys()
	for k, s := range sorts {
		if s.Column != column {
			continue
		}
		mark := "▲"
		if s.Desc {
			mark = "▼"
		}
		if len(sorts) > 1 {
			mark += subscript(k + 1)
		}
		return mark
	}
	return ""
}

// grouping gives how many columns the rows group by: those the person chose, and those they group by on their own.
func (p Panel) grouping() int {
	groups := p.Groups()
	n := len(groups)
	for _, c := range p.Auto {
		if !slices.Contains(groups, c) {
			n++
		}
	}
	return n
}

// subscript gives a number in subscript digits, as ₁₂, which mark a column's place among the keys or the groups.
func subscript(n int) string {
	return strings.Map(func(r rune) rune { return '₀' + r - '0' }, strconv.Itoa(n))
}

// Grouped gives the panel grouping the rows by the column alone, as a ctrl-click on its name asks, or, with nest, as a
// ctrl-option-click asks, within the groups it has, innermost; a column it groups by already, it no longer does, alone or among the others, and it is a column
// again. A column that cannot group changes nothing.
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
// one a column the person groups by, and one the heading over every row, when the rows group on their own and the
// person groups by nothing, so the rows under it stand in as they do under a heading the person chose.
func Levels(rows []Row) int {
	n := 0
	for _, r := range rows {
		if Heads(r) {
			n = max(n, strings.Count(r.Key, "\x00"))
		}
	}
	return n
}

// Said gives the level of the headings, from 1, that say each column the rows group by on their own, the shallowest
// when it is said at several: the heading over every row is the first level, as is a heading of the outermost column
// the person chose.
func Said(rows []Row) map[string]int {
	out := map[string]int{}
	for _, r := range rows {
		if !Heads(r) {
			continue
		}
		level := max(1, strings.Count(r.Key, "\x00"))
		for _, c := range r.Says {
			if at, ok := out[c]; !ok || level < at {
				out[c] = level
			}
		}
	}
	return out
}

// overAll is the key of the heading over every row.
const overAll = "group\x00"

// Heads says whether a row is the heading of a group, as group draws one.
func Heads(r Row) bool { return r.Aside && strings.HasPrefix(r.Key, "group\x00") }

// Heading gives the column whose group a row heads, as group draws the headings, or nothing for a row that heads no
// group, or the heading over every row, which no column the person chose makes.
func (p Panel) Heading(r Row) string {
	if !Heads(r) {
		return ""
	}
	if groups, depth := p.Groups(), strings.Count(r.Key, "\x00")-1; depth < len(groups) {
		return groups[depth]
	}
	return ""
}

// group gives the rows under a heading per value of each column the panel groups by, the outermost first, the groups
// in their values' order, the larger first where the column sorts descending, and the rows in each as they came: a
// heading, an aside, is the value, as the rows draw it or as their cell's Head says it, and the column's name, dim,
// indented by its depth, and how many rows it heads, dim. A row groups by a cell's sort key, or under none when the
// cell has none. A row of one cell, as a heading, has no value: those follow the groups.
//
// While the panel folds, under each heading, the walk goes on along the columns that can group, in the panel's order:
// one whose value is the same on every row the heading heads, as their sort key says, one value on each and none
// blank, groups them on its own. Its value joins the heading, after a comma, and its cell is blank in those rows, so
// the value is said once; a heading under it says it no more. Without a column the person chose, the rows group on
// their own under one heading over every row, their values alone. A heading over one row is not worth it: a group of
// one row, or a stage of one, does not group on its own. A column hidden never does. It gives the columns that group
// on their own anywhere.
func (p Panel) group(rows []Row, columns []string) ([]Row, []string) {
	var at []int
	var desc []bool
	for _, name := range p.Groups() {
		if j := slices.Index(columns, name); j >= 0 {
			at, desc = append(at, j), append(desc, p.Fields[name].Sort == Descending)
		}
	}
	value := func(r Row, j int) string {
		if j < len(r.Cells) && r.Cells[j].key() != "" {
			return r.Cells[j].key()
		}
		return "none"
	}
	// same says whether two rows share their values down to a depth
	same := func(a, b Row, depth int) bool {
		for _, j := range at[:depth+1] {
			if value(a, j) != value(b, j) {
				return false
			}
		}
		return true
	}
	var grouped, rest []Row
	for _, r := range rows {
		if len(r.Cells) > 1 {
			grouped = append(grouped, r)
		} else {
			rest = append(rest, r)
		}
	}
	walk := p.walk(columns, at)
	var took []int
	if len(at) == 0 {
		took = constant(grouped, walk)
		if len(took) == 0 {
			return rows, nil
		}
		heading := p.fold(overAll) + " " + joined(grouped[0], columns, took) + "  " +
			style.Off(strconv.Itoa(len(grouped)))
		out := []Row{{Key: overAll, Cells: []Cell{{Values: []string{heading}, Wrap: true}}, Aside: true,
			Says: names(columns, took)}}
		for _, r := range grouped {
			out = append(out, blank(r, took))
		}
		return append(out, rest...), names(columns, took)
	}
	slices.SortStableFunc(grouped, func(a, b Row) int {
		for k, j := range at {
			if c := natural(value(a, j), value(b, j)); c != 0 {
				if desc[k] {
					return -c
				}
				return c
			}
		}
		return 0
	})
	var out []Row
	// the columns each heading open at its depth groups on its own, which the headings under it leave out
	auto := make([][]int, len(at))
	for i, r := range grouped {
		for depth, j := range at {
			if i > 0 && same(grouped[i-1], r, depth) {
				continue
			}
			n, key := 0, "group"
			for _, s := range grouped[i:] {
				if !same(s, r, depth) {
					break
				}
				n++
			}
			for _, x := range at[:depth+1] {
				key += "\x00" + value(r, x)
			}
			auto[depth] = constant(grouped[i:i+n], slices.DeleteFunc(slices.Clone(walk), func(x int) bool {
				return slices.ContainsFunc(auto[:depth], func(above []int) bool { return slices.Contains(above, x) })
			}))
			took = append(took, auto[depth]...)
			said := "none"
			if j < len(r.Cells) {
				said = r.Cells[j].heading()
			}
			heading := strings.Repeat("  ", depth) + p.fold(key) + " " + said + " " + style.Off(columns[j])
			if len(auto[depth]) > 0 {
				heading += style.Off(",") + " " + joined(r, columns, auto[depth])
			}
			heading += "  " + style.Off(strconv.Itoa(n))
			out = append(out, Row{Key: key, Cells: []Cell{{Values: []string{heading}, Wrap: true}}, Aside: true,
				Says: names(columns, auto[depth])})
		}
		out = append(out, blank(r, slices.Concat(auto...)))
	}
	return append(out, rest...), names(columns, took)
}

// Unfolded and Folded are the glyphs before a node of the outline that hides something, a group's heading or a row:
// what it hides shown, or hidden. Folded is drawn dim, as is a heading's text; Unfolded is not.
const (
	Unfolded = "▾"
	Folded   = "▸"
)

// Outline is which nodes of the rows' outline stand open, by key: a group's heading, whose rows show under it, a row,
// whose lines and detail show, and anything else a screen keys in it, as its header. A key it does not hold stands as
// it opens: a heading open, everything else closed.
type Outline map[string]bool

// Open says whether a row stands open: a group's heading unless it was closed, any other row only once opened.
func (o Outline) Open(r Row) bool {
	if open, ok := o[r.Key]; ok {
		return open
	}
	return Heads(r)
}

// fold gives the glyph before the heading of a key, styled: Folded, dim, while its group is closed, Unfolded while it is
// open.
func (p Panel) fold(key string) string {
	if !p.Outline.Open(Row{Key: key, Aside: true}) {
		return style.Off(Folded)
	}
	return Unfolded
}

// FoldAway gives the rows as group draws them with every closed group folded away: under a closed heading, the rows and
// the headings deeper than it, up to the next heading as shallow, or the rows of one cell that follow the groups. The
// heading stays, its count the rows it hides, and holds them in its Folded, which lay the columns out still.
func (p Panel) FoldAway(rows []Row) []Row {
	var out []Row
	closed := 0 // the level of the closed heading the rows stand under, or none
	for _, r := range rows {
		level := Level(r)
		if closed > 0 && (level > closed || level == 0 && len(r.Cells) > 1) {
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

// walk gives the columns the rows may group by on their own, as indexes into them, in the panel's order: while the
// panel folds, each column that can group and has a name, but those the person groups by or hides; none while it does
// not.
func (p Panel) walk(columns []string, chosen []int) []int {
	if !p.Fold {
		return nil
	}
	var out []int
	for _, j := range p.arrangement(columns) {
		name := columns[j]
		if name != "" && p.Groupable[name] && p.Shows(name) && !slices.Contains(chosen, j) {
			out = append(out, j)
		}
	}
	return out
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

// joined gives each column a row groups by on its own, at the indexes, as a heading says it: the row's value in it,
// as its cell's heading says it, and the column's name, as a heading of a column the person chose says them, and a comma between
// columns.
func joined(r Row, names []string, columns []int) string {
	var values []string
	for _, j := range columns {
		values = append(values, r.Cells[j].heading()+" "+style.Off(names[j]))
	}
	return strings.Join(values, style.Off(",")+" ")
}

// blank gives the row with its cells in the columns blank, their values said by the heading over it.
func blank(r Row, columns []int) Row {
	if len(columns) == 0 {
		return r
	}
	r.Cells = slices.Clone(r.Cells)
	for _, j := range columns {
		r.Cells[j] = Cell{Right: r.Cells[j].Right}
	}
	return r
}

// names gives the columns' names, each once, in the stage's order.
func names(columns []string, at []int) []string {
	var out []string
	for j, name := range columns {
		if slices.Contains(at, j) {
			out = append(out, name)
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

// control is one of the panel's controls: the search, or a field's line in the editor.
type control struct {
	search bool
	field  string
}

// sections are the editor's sections, top to bottom, each a placement, and the dim heading over each, by placement.
var (
	sections = []Placement{Group, Column, Hidden}
	headings = map[Placement]string{Group: "group by", Column: "columns", Hidden: "hidden"}
)

// label gives the label a field is listed by: the one the panel was given, else its id.
func (p Panel) label(id string) string {
	if l := p.Labels[id]; l != "" {
		return l
	}
	return id
}

// offers says whether the editor lists a field: one with a name among the columns, or one the rows group by.
func (p Panel) offers(id string) bool {
	return id != "" && (slices.Contains(p.Columns, id) || p.Fields[id].Placement == Group)
}

// section gives the fields the editor lists under a placement, in their order: the groups by depth, the outermost
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

// controls gives the panel's controls, in order: the search, and, while the stage has fields with a name, each field in
// its section, as section gives them.
func (p Panel) controls() []control {
	out := []control{{search: true}}
	if !slices.ContainsFunc(p.Columns, func(id string) bool { return id != "" }) {
		return out
	}
	for _, at := range sections {
		for _, id := range p.section(at) {
			out = append(out, control{field: id})
		}
	}
	return out
}

// Filtered is the mark after a field's label in the editor while a filter on it holds rows back.
const Filtered = "⊘"

// drawnLine is one line of the panel, as layout draws it: its text; the control it draws, or -1; and for a section's
// heading, the section, which a field dragged onto it moves to the top of.
type drawnLine struct {
	text    string
	control int
	heading Placement
	heads   bool
}

// layout draws the panel's lines, as Lines says.
func (p Panel) layout() []drawnLine {
	var out []drawnLine
	add := func(text string, control int) { out = append(out, drawnLine{text: text, control: control}) }
	controls := p.controls()
	draw := func(k int, line string) {
		if p.Focused && p.Cursor == k && k > 0 {
			line = style.Cursor(pane.Pad(ansi.Strip(line), p.Fill, false))
		}
		add(line, k)
	}
	add(shell.RecallLine(p.Query, p.Focused && p.Cursor == 0, p.Recall), 0)
	if len(controls) == 1 {
		return out
	}
	add("", -1)
	k := 1
	for _, at := range sections {
		// every section's heading, an empty one's too, so a field can be moved to it
		out = append(out, drawnLine{text: style.Off(headings[at]), control: -1, heading: at, heads: true})
		for range p.section(at) {
			draw(k, p.fieldLine(controls[k].field))
			k++
		}
	}
	if p.Focused {
		add("", -1)
		for _, h := range p.hints() {
			add(style.Off(h), -1)
		}
	}
	return out
}

// fieldLine draws a field's line in the editor: its box, ticked in the accent and bold for a column or a group, dim and
// unticked for a field hidden; its label; ▲ or ▼ while it sorts the rows, with its place among the keys when there are
// several; ▤ while the rows group by it, with its depth while they group by more than one, or dim while they group by
// it on their own, at the level of the headings that say it; and ⊘ while a filter on it holds rows back.
func (p Panel) fieldLine(id string) string {
	line := style.Mark(Ticked + " " + p.label(id))
	if p.Fields[id].Placement == Hidden {
		line = style.Off(Unticked) + " " + p.label(id)
	}
	if mark := p.Mark(id); mark != "" {
		line += " " + style.Tint(mark)
	}
	if k := slices.Index(p.Groups(), id); k >= 0 {
		mark := Grouping
		if p.grouping() > 1 {
			mark += subscript(k + 1)
		}
		line += " " + style.Tint(mark)
	} else if slices.Contains(p.Auto, id) {
		mark := Grouping
		if at := p.AutoAt[id]; at > 0 && p.grouping() > 1 {
			mark += subscript(at)
		}
		line += " " + style.Off(mark)
	}
	if len(p.Fields[id].Filter.Values) > 0 {
		line += " " + style.Tint(Filtered)
	}
	return line
}

// Lines draws the panel's controls, one a line: the search; under it, after a gap, the field editor, one list in three
// sections, group by, columns, and hidden, each under a dim heading, a field's line as fieldLine draws it; after a gap,
// whether the panel folds, a box ticked or not; and, while the panel has the keyboard, after a gap, dim hints of what
// moves a field. The control under the keyboard is inverse while the panel has it. Beside each line is the control it
// draws, or -1 for a heading, a hint, or a gap.
func (p Panel) Lines() (lines []string, at []int) {
	for _, l := range p.layout() {
		lines, at = append(lines, l.text), append(at, l.control)
	}
	return lines, at
}

// hints gives the dim lines under the editor, while the panel has the keyboard, that say how a field moves, and what its
// popover sets; then what a click on a column's name in the header does.
func (p Panel) hints() []string {
	return append([]string{"⇧↑↓  move, or drag", "⏎    filter, draw", ""}, Clicks...)
}

// Clicks say what a click on a column's name in the header does: it drops down the column's menu, which sorts by it,
// groups by it, and opens its field's popover.
var Clicks = []string{"click a name  its menu"}

// Update takes a key while the panel has the keyboard, and says whether it took it: up and down move between the
// controls, stopping at the ends; on the search, a printable key is typed, / too, and backspace takes one back; off it,
// / goes to the search; on a field, space hides it, or shows a hidden one as a column, and shift with up or down moves
// it, as Step says, the keyboard with it; on whether the panel folds, or shows the emoji every row shares, space or
// enter toggles it.
func (p Panel) Update(k tea.KeyPressMsg) (Panel, bool) {
	controls := p.controls()
	p.Cursor = min(max(p.Cursor, 0), len(controls)-1)
	c := controls[p.Cursor]
	switch key := k.String(); {
	case key == "up":
		p.Cursor = max(p.Cursor-1, 0)
	case key == "down":
		p.Cursor = min(p.Cursor+1, len(controls)-1)
	case c.search && (key == "backspace" || k.Text != ""):
		p.Query, _ = shell.Type(p.Query, k)
	case k.Text == "/":
		p.Cursor = 0
	case c.field != "" && (key == "shift+up" || key == "shift+down"):
		step := 1
		if key == "shift+up" {
			step = -1
		}
		p, _ = p.Step(c.field, step)
	case c.field != "" && key == "space":
		p = p.act(c)
	default:
		return p, false
	}
	return p, true
}

// Under gives the field whose line is under the keyboard, or nothing on another control.
func (p Panel) Under() string {
	controls := p.controls()
	return controls[min(max(p.Cursor, 0), len(controls)-1)].field
}

// Keys gives the keys bar's segment of the control under the keyboard: the search types, and recalls a search when
// there are some; a field's line hides it, or shows it, with space, moves it with shift and up or down, where it can
// move that way, and opens its popover with enter, which the screen draws.
func (p Panel) Keys(recalls bool) shell.Segment {
	controls := p.controls()
	c := controls[min(max(p.Cursor, 0), len(controls)-1)]
	if c.search {
		return shell.SearchKeys(recalls)
	}
	toggle := "hide"
	if p.Fields[c.field].Placement == Hidden {
		toggle = "show"
	}
	s := shell.Segment{Takes: []string{"shift+up", "shift+down"},
		Bindings: []shell.Binding{shell.Bind("␣", toggle, "space")}}
	var moves []string
	if _, ok := p.Step(c.field, -1); ok {
		moves = append(moves, "shift+up")
	}
	if _, ok := p.Step(c.field, 1); ok {
		moves = append(moves, "shift+down")
	}
	if len(moves) > 0 {
		s.Bindings = append(s.Bindings, shell.Bind("⇧↑↓", "move", moves...))
	}
	return s.With(shell.Bind("⏎", "options", "enter"))
}

// Click takes a click on a line of the panel, as Lines numbers them, and says whether it fell on a control: the
// control is then under the keyboard, and a field's box or whether the panel folds is acted on as space does.
func (p Panel) Click(line int) (Panel, bool) {
	p, ok := p.Point(line)
	if !ok {
		return p, false
	}
	if c := p.controls()[p.Cursor]; !c.search {
		p = p.act(c)
	}
	return p, true
}

// FieldAt gives the id of the field whose line a line of the panel, as Lines numbers them, draws, or nothing.
func (p Panel) FieldAt(line int) string {
	_, at := p.Lines()
	if line < 0 || line >= len(at) || at[line] < 0 {
		return ""
	}
	return p.controls()[at[line]].field
}

// Point puts the keyboard on the control a line of the panel draws, acting on nothing, and says whether one is there.
func (p Panel) Point(line int) (Panel, bool) {
	_, at := p.Lines()
	if line < 0 || line >= len(at) || at[line] < 0 {
		return p, false
	}
	p.Cursor = at[line]
	return p, true
}

// Find puts the search under the keyboard.
func (p Panel) Find() Panel {
	p.Cursor, p.Focused = 0, true
	return p
}

// act toggles a field's box, hiding a column or a group, which then leaves the groups, and showing a hidden field as a
// column where it stood.
func (p Panel) act(c control) Panel {
	if p.Fields[c.field].Placement == Hidden {
		return p.Placed(c.field, Column).follow(c.field)
	}
	return p.Placed(c.field, Hidden).follow(c.field)
}

// follow gives the panel with the keyboard on a field's line, wherever it stands now.
func (p Panel) follow(id string) Panel {
	if k := slices.IndexFunc(p.controls(), func(c control) bool { return c.field == id }); k >= 0 {
		p.Cursor = k
	}
	return p
}

// Step gives the panel with a field moved a place, up for a negative step, down for a positive one, the keyboard
// with it, and says whether it moved: within its section, a group moves a depth and a column a position; past its
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

// Moved gives the panel with a field moved into a section at an index among the fields there, the keyboard with it,
// and says whether anything changed: into the groups, at that depth, where the field can group; into the columns, at
// that position, which every field the panel offers then has, a field hidden or grouped keeping its own among them;
// hidden, where it keeps its position. A field the editor does not list never moves.
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
	p = p.follow(id)
	after := [][]string{p.section(Group), p.section(Column), p.section(Hidden)}
	return p, !slices.EqualFunc(before, after, slices.Equal)
}

// Dropped gives the panel with a field moved, as a drag of its line drops it on a line of the panel, as Lines numbers
// them: onto another field's line, to where that field stands, it and those after it moving a place; onto a section's
// heading, to the top of that section. It says whether anything changed.
func (p Panel) Dropped(id string, line int) (Panel, bool) {
	drawn := p.layout()
	if line < 0 || line >= len(drawn) {
		return p, false
	}
	l := drawn[line]
	if l.heads {
		return p.Moved(id, l.heading, 0)
	}
	target := ""
	if l.control >= 0 {
		target = p.controls()[l.control].field
	}
	if target == "" || target == id {
		return p, false
	}
	at := p.Fields[target].Placement
	return p.Moved(id, at, slices.Index(p.section(at), target))
}

// Width gives the cells the panel's widest line takes, with the keyboard or without it.
func (p Panel) Width() int {
	p.Focused = true
	lines, _ := p.Lines()
	widest := 0
	for _, l := range lines {
		widest = max(widest, ansi.StringWidth(l))
	}
	return widest
}

// Filter gives the rows the panel lets through, as Filtered does.
func (p Panel) Filter(rows []Row, columns []string) []Row {
	out, _ := p.Filtered(rows, columns)
	return out
}

// Filtered gives the rows the panel lets through: those whose text has the search, without regard to case, and whose
// raw values pass the filter on each field, sorted by its keys, grouped as group says, each with its cells in its
// order and without those of the columns it does not show. columns names each cell's column, in the stage's order.
// While the search or a filter narrows the rows, an aside that heads none of them is left out with them. It gives the
// columns the rows group by on their own too, for the panel's Auto.
func (p Panel) Filtered(rows []Row, columns []string) ([]Row, []string) {
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
	out, auto := p.group(p.order(out, columns), columns)
	for k, r := range out {
		if !r.Aside {
			out[k] = p.hide(r, columns)
		}
	}
	return out, auto
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
