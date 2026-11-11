// A preset sets a table's panel, and a table's panel is read back as a preset.
package table

import (
	"cmp"
	"slices"

	"github.com/empowerite/kit/projects/tui/internal/grid"
)

// Panel gives a stage's Panel set as a preset says, of the stage's fields, the rest of it as p has it: each field placed,
// sorted, filtered, and drawn as the preset says, a field it does not name placed as the catalog says, and folding
// unless the preset says it does not. What a field cannot do it is not set to do: a field without a label is never
// hidden, nor grouped, sorted, or filtered unless it can be, nor drawn by a rendering not its own, nor flat unless it
// holds a list; a list keeps its sort, which sorts the rows while it is flat, or grouped by.
func (pr Preset) Panel(fields []Field, p grid.Panel) grid.Panel {
	p.Fields = map[string]grid.Field{}
	for _, f := range fields {
		s := pr.Fields[f.ID]
		var g grid.Field
		switch cmp.Or(s.Placement, f.Placement) {
		case PlaceHidden:
			if f.Label != "" {
				g.Placement = grid.Hidden
			}
		case PlaceGroup:
			if f.Group {
				g.Placement, g.Depth = grid.Group, cmp.Or(s.Depth, f.Depth)
			}
		default:
			g.Position = s.Position
		}
		if (f.Sort || f.One != "") && (s.Sort == "asc" || s.Sort == "desc") {
			g.Sort, g.Priority = grid.Ascending, s.Priority
			if s.Sort == "desc" {
				g.Sort = grid.Descending
			}
		}
		if s.Filter != nil && f.Filter {
			switch {
			case len(s.Filter.Include) > 0:
				g.Filter = grid.Filter{Values: slices.Clone(s.Filter.Include)}
			case len(s.Filter.Exclude) > 0:
				g.Filter = grid.Filter{Values: slices.Clone(s.Filter.Exclude), Exclude: true}
			}
		}
		if slices.Contains(f.Renderings, s.Rendering) {
			g.Rendering = s.Rendering
		}
		g.Flat = s.Flat && f.One != ""
		p.Fields[f.ID] = g
	}
	return p
}

// PresetOf gives a stage's panel as a preset keeps it, of the stage's fields, unnamed: of each field, what differs from
// what a preset that names none of them says. A column's position is said only while the columns stand in another order
// than the stage's, and then for every column, from 1; a group's depth and a sort's priority are numbered from 1, the
// outermost and the first first.
func PresetOf(p grid.Panel, fields []Field) Preset {
	var out Preset
	var ids []string
	for _, f := range fields {
		if f.Label != "" {
			ids = append(ids, f.ID)
		}
	}
	columns := slices.DeleteFunc(p.Arrange(ids), func(id string) bool { return !p.Shows(id) })
	positioned := !slices.Equal(columns, slices.DeleteFunc(slices.Clone(ids), func(id string) bool { return !p.Shows(id) }))
	groups, sorts := p.Groups(), p.SortKeys()
	for _, f := range fields {
		g := p.Field(f.ID)
		var s Setting
		switch k := slices.Index(columns, f.ID); {
		case g.Placement == grid.Hidden && f.Label != "":
			if f.Placement != PlaceHidden {
				s.Placement = PlaceHidden
			}
		case g.Placement == grid.Group && slices.Contains(groups, f.ID):
			if depth := slices.Index(groups, f.ID) + 1; f.Placement != PlaceGroup || f.Depth != depth {
				s.Placement, s.Depth = PlaceGroup, depth
			}
		case positioned && k >= 0:
			s.Placement, s.Position = PlaceColumn, k+1
		case cmp.Or(f.Placement, PlaceColumn) != PlaceColumn:
			s.Placement = PlaceColumn
		}
		if k := slices.IndexFunc(sorts, func(s grid.SortKey) bool { return s.Column == f.ID }); k >= 0 {
			s.Sort, s.Priority = "asc", k+1
			if sorts[k].Desc {
				s.Sort = "desc"
			}
		}
		if len(g.Filter.Values) > 0 {
			s.Filter = &ValueFilter{Include: slices.Clone(g.Filter.Values)}
			if g.Filter.Exclude {
				s.Filter = &ValueFilter{Exclude: slices.Clone(g.Filter.Values)}
			}
		}
		s.Rendering, s.Flat = g.Rendering, g.Flat
		if s != (Setting{}) {
			if out.Fields == nil {
				out.Fields = map[string]Setting{}
			}
			out.Fields[f.ID] = s
		}
	}
	return out
}
