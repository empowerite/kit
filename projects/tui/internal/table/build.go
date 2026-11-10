// A table's rows built from its records: each record's values drawn in the table's fields, the rows the panel lets
// through, and the renderings Automatic chooses for them at the width they have.
package table

import (
	"slices"

	"github.com/empowerite/kit/projects/tui/internal/grid"
)

// Lead is what stands before the rule under the header: the gutter the rows' marks and folds take, as internal/grid
// lays it out, two cells for the selection and two for the ▸ or ▾.
const Lead = "    "

// Record is a row as a screen's builder gives it: the row, without its cells, and its value in each of the table's
// fields, by the field's id. A row of one cell, which spans the columns, as a heading, has its cell and no values.
type Record struct {
	grid.Row
	Values map[string]Value
}

// Lay gives a table's rows from its records: each record's values drawn in the fields, in the order their columns
// stand, as each field's renderer draws its column, so a renderer can line a column up, or the one pinned gives it by
// id; a row of one cell as it is.
func Lay(fields []Field, records []Record, pinned func(id string) string) []grid.Row {
	out := make([]grid.Row, len(records))
	var at []int
	for k, r := range records {
		out[k] = r.Row
		if r.Values != nil {
			at = append(at, k)
		}
	}
	for _, f := range fields {
		vs := make([]Value, len(at))
		for n, k := range at {
			vs[n] = records[k].Values[f.ID]
		}
		for n, c := range f.Draw(vs, pinned(f.ID)) {
			out[at[n]].Cells = append(out[at[n]].Cells, c)
		}
	}
	return out
}

// Shown is a table's rows as its panel lets them through, the columns its panel offers, how many levels of headings
// they stand under, how many rows but the asides it holds, the panel apart, and the rendering each field is drawn by,
// by id, as Automatic chose it.
type Shown struct {
	Rows   []grid.Row
	Names  []string
	Levels int
	All    int
	Drawn  map[string]string
}

// Build gives a table's rows from its records, as the panel lets them through, and what the panel shows of them, the
// header standing heads over the columns, in a pane of the width. The rows are laid out twice: first as the panel
// pins each field's rendering, or by its first, which says the rows the panel lets through; then by the renderings
// Automatic chooses for those rows and the width, when they differ. The second time lets through the rows the first
// did, so a search keeps the rows whose text had it, though a field now draws less of it.
func Build(s *Schema, records []Record, p grid.Panel, heads []string, width int) Shown {
	names, fields := s.Columns(), s.Of()
	all := Lay(fields, records, func(id string) string { return p.Field(id).Rendering })
	rows := p.Filter(all, names)
	placed := Placements(fields, names, p, heads)
	drawn := Automatic(placed, ShownValues(records, rows, p), width-2*grid.Indents(rows)-len(Lead), p.Glyphs)
	if slices.ContainsFunc(placed, func(f Placed) bool { return drawn[f.ID] != f.DrawnBy(f.Pinned) }) {
		passed := map[string]bool{}
		for _, r := range rows {
			passed[r.Key] = true
		}
		all = Lay(fields, records, func(id string) string { return drawn[id] })
		q := p
		q.Query = ""
		rows = q.Filter(slices.DeleteFunc(slices.Clone(all), func(r grid.Row) bool { return !passed[r.Key] }), names)
	}
	return Shown{p.FoldAway(rows), Filled(all, names), grid.Indents(rows), Counted(all), drawn}
}

// Placements gives a table's fields as its panel places them, for Automatic: its columns, in the order they stand,
// each with the name the header stands over it while it shows, of heads, then the fields grouped or hidden. columns
// names each field's column, as the panel knows it.
func Placements(fields []Field, columns []string, p grid.Panel, heads []string) []Placed {
	var out, rest []Placed
	for k, id := range p.Arrange(columns) {
		f := fields[k]
		if id != "" {
			f = fields[slices.IndexFunc(fields, func(f Field) bool { return f.ID == id })]
		}
		pin := p.Field(f.ID).Rendering
		if id != "" && !p.Shows(id) {
			rest = append(rest, Placed{Field: f, Pinned: pin})
			continue
		}
		name := ""
		if len(out) < len(heads) {
			name = heads[len(out)]
		}
		out = append(out, Placed{Field: f, Pinned: pin, Column: true, Name: name})
	}
	return append(out, rest...)
}

// ShownValues gives the values the records the rows let through hold in each field, by id, one a record; a list the
// panel flattens or groups by, one a value, as the rows stand once per value of it.
func ShownValues(records []Record, rows []grid.Row, p grid.Panel) map[string][]Value {
	passed := map[string]bool{}
	for _, r := range rows {
		if !r.Aside {
			passed[r.Key] = true
		}
	}
	out := map[string][]Value{}
	for _, r := range records {
		if r.Values == nil || !passed[r.Key] {
			continue
		}
		for id, v := range r.Values {
			if f := p.Field(id); !p.Lists(id) || !f.Flat && f.Placement != grid.Group || len(v.Raw) == 0 {
				out[id] = append(out[id], v)
				continue
			}
			for _, raw := range v.Raw {
				one := v
				one.Raw = []string{raw}
				out[id] = append(out[id], one)
			}
		}
	}
	return out
}

// Filled gives a table's columns as its panel offers them: each named column that one of the rows has a value in, and
// no name for the rest, as a branch while no row has one, since a line there would hide nothing.
func Filled(rows []grid.Row, names []string) []string {
	out := make([]string, len(names))
	for _, r := range rows {
		if len(r.Cells) < 2 {
			continue
		}
		for j, c := range r.Cells {
			if j < len(names) && len(c.Values) > 0 && c.Values[0] != "" {
				out[j] = names[j]
			}
		}
	}
	return out
}

// LabelNamed gives the label of a name, with its color, as one of the records holds it in the field, by id; a name no
// record holds is a label without a color.
func LabelNamed(records []Record, id, name string) Label {
	for _, r := range records {
		for _, l := range LabelsIn(r.Values[id]) {
			if l.Name == name {
				return l
			}
		}
	}
	return Label{Name: name}
}
