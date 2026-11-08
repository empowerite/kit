// A host for the controls' tests: a table of its own fields and rows, its panel kept as the controls set it.
package table

import (
	"slices"

	"github.com/empowerite/kit/projects/tui/internal/grid"
)

// host is a screen of one table: its fields, its panel, the values its rows hold, by field, its caller, the renderings
// drawn, and its size.
type host struct {
	fields []Field
	panel  grid.Panel
	values map[string][]Value
	me     string
	drawn  map[string]string
	w, h   int
	sets   int // how many times the panel was set
	all    []grid.Row
	tbl    Table
	keys   bool // whether the rows have the keyboard
	hides  bool
	built  int // how many times the rows were built again
	settle int // how many times the cursor was kept on a row
	views  Views
	loaded bool // whether the table has columns, loaded
	paneH  int  // the pane's height
	ctl    bool // whether a control line stands over the rows
	left   int  // how many times the rows gave up the keyboard
}

// Lines draws the rows as the grid does, the one under the cursor chosen while the rows have the keyboard.
func (h *host) Lines() []Line {
	rows, chosen := h.Rows(), -1
	if h.keys && h.tbl.Cursor < len(rows) {
		chosen = h.tbl.Cursor
	}
	var out []Line
	for _, l := range grid.Draw(rows, h.Names(), h.GridWidth(), chosen, h.tbl.Open.Open, func(grid.Row) string { return "" }) {
		out = append(out, Line{Row: l.Row, Text: l.Text})
	}
	return out
}

// Header gives the header's names and a rule under them, none while no row shows.
func (h *host) Header() ([]string, bool) {
	rows := h.Rows()
	if !slices.ContainsFunc(rows, func(r grid.Row) bool { return !r.Aside }) {
		return nil, false
	}
	lines, hides := grid.Header(rows, h.Names(), h.GridWidth(), h.tbl.Open[HeaderKey])
	return append(lines, "────"), hides
}

func (h *host) PaneHeight() int { return h.paneH }
func (h *host) Controls() bool  { return h.ctl }
func (h *host) LeaveRows()      { h.keys, h.left = false, h.left+1 }
func (h *host) EnterRows()      { h.keys = true }

func (h *host) Rows() []grid.Row {
	p := h.panel
	p.Outline = h.tbl.Open
	return p.FoldAway(p.Filter(h.all, h.panel.Columns))
}
func (h *host) Names() []string   { return h.panel.Columns }
func (h *host) GridWidth() int    { return h.w }
func (h *host) Table() *Table     { return &h.tbl }
func (h *host) Rebuild()          { h.built++ }
func (h *host) Settle()           { h.settle++ }
func (h *host) InRows() bool      { return h.keys }
func (h *host) HeaderHides() bool { return h.hides }
func (h *host) Views() Views      { return h.views }
func (h *host) Columnar() bool    { return h.loaded }
func (h *host) Color() string     { return "75" }
func (h *host) Selector() (int, int) {
	return 60, 70
}

func (h *host) Fields() []Field          { return h.fields }
func (h *host) Panel() grid.Panel        { return h.panel }
func (h *host) SetPanel(p grid.Panel)    { h.panel, h.sets = p, h.sets+1 }
func (h *host) Values(id string) []Value { return h.values[id] }
func (h *host) Me() string               { return h.me }
func (h *host) Drawn(id string) string   { return h.drawn[id] }
func (h *host) Size() (int, int)         { return h.w, h.h }
func (h *host) Name() string             { return "things" }
func (h *host) Pane() (int, int)         { return 4, h.w - 1 }
func (h *host) ShownColumns() []string   { return h.panel.Columns }
func (h *host) HeaderTop() int           { return 6 }

// HeaderName gives where a column's name stands in the header: each column eight cells after the one before, from 4,
// its label its first line; none for title, which the header does not name.
func (h *host) HeaderName(column string) (int, string) {
	k := slices.Index(h.panel.Columns, column)
	if k < 0 || column == "title" {
		return 0, ""
	}
	return 4 + 8*k, column
}

// HeadingName gives where a column's name stands in the heading over the groups, on line 4, while it groups the rows.
func (h *host) HeadingName(column string) (int, int) {
	if k := slices.Index(h.panel.Groups(), column); k >= 0 {
		return 4 + 8*k, 4
	}
	return 0, -1
}
func (h *host) Label(id, name string) Label {
	for _, v := range h.values[id] {
		for _, l := range LabelsIn(v) {
			if l.Name == name {
				return l
			}
		}
	}
	return Label{Name: name}
}

// who is a person field whose filter offers @me.
var who = Field{ID: "who", Label: "who", Type: "person", Renderings: []string{"text"}, Filter: true}

// tags is a field of labels.
var tags = Field{ID: "tags", Label: "tags", Type: "label", Renderings: []string{"pills"}, Filter: true}

// screen gives a host of state, who, tags, and title, the rows holding a few values of each.
func screen() *host {
	return &host{fields: []Field{state, who, tags, title}, w: 80, h: 24,
		values: map[string][]Value{
			"state": {One("open"), One("merged")},
			"who":   {One("bob"), One("ada"), {}},
			"tags":  {Labelled([]Label{{"bug", "#ff0000"}})},
		},
		drawn: map[string]string{"state": "glyph+text"}}
}
