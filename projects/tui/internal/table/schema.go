// What a table's catalog says of its fields, as its panel knows them: the names of its columns, their labels, which
// sort, which group, which hold a list, and which names the people its rows belong to. Each holds for no table too, a
// nil Schema, which has no fields.
package table

import "github.com/empowerite/kit/projects/tui/internal/grid"

// Of gives the table's fields, in the order its columns stand; none for no table.
func (s *Schema) Of() []Field {
	if s == nil {
		return nil
	}
	return s.Fields
}

// Columns gives the names the panel knows the table's columns by, in the order they stand: each field's id, and none
// for a field without a label, which the panel's editor never lists nor moves, and never sorts or groups by.
func (s *Schema) Columns() []string {
	var out []string
	for _, f := range s.Of() {
		if f.Label == "" {
			out = append(out, "")
			continue
		}
		out = append(out, f.ID)
	}
	return out
}

// Labels gives the label of each of the table's fields that has one, by its id, as the panel lists them.
func (s *Schema) Labels() map[string]string {
	out := map[string]string{}
	for _, f := range s.Of() {
		if f.Label != "" {
			out[f.ID] = f.Label
		}
	}
	return out
}

// able gives the ids of the table's fields that can, as can says, as a set.
func (s *Schema) able(can func(Field) bool) map[string]bool {
	out := map[string]bool{}
	for _, f := range s.Of() {
		if f.Label != "" && can(f) {
			out[f.ID] = true
		}
	}
	return out
}

// Sortable names the table's columns its rows can sort by: a value that orders.
func (s *Schema) Sortable() map[string]bool { return s.able(func(f Field) bool { return f.Sort }) }

// Groupable names the table's columns its rows can group by: a value many rows share.
func (s *Schema) Groupable() map[string]bool { return s.able(func(f Field) bool { return f.Group }) }

// Ones names one value of each of the table's fields that holds a list, by its id: a field the rows can flatten.
func (s *Schema) Ones() map[string]string {
	out := map[string]string{}
	for _, f := range s.Of() {
		if f.Label != "" && f.One != "" {
			out[f.ID] = f.One
		}
	}
	return out
}

// Owner gives the id of the person field that names the people the table's rows belong to, whose filter the panel
// sets as every row, the caller's, or everyone else's; none for a table whose rows belong to nobody.
func (s *Schema) Owner() string {
	for _, f := range s.Of() {
		if f.Owns {
			return f.ID
		}
	}
	return ""
}

// Ready gives a panel with what the catalog says of the table's fields: whose rows they are, the columns that sort and
// group, each field's label, and the one value of each list.
func (s *Schema) Ready(p grid.Panel) grid.Panel {
	p.Owner, p.Sortable, p.Groupable, p.Labels, p.Ones = s.Owner(), s.Sortable(), s.Groupable(), s.Labels(), s.Ones()
	return p
}
