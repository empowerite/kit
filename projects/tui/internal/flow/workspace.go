// kit workspace's stages, the picture of what git holds across the workspace: remotes, branches, worktrees, commits,
// and tags. Their rows are drawn from the catalog alone: kit prints each row's values by the field's id, and where a
// value leads, so a stage needs no Go of its own. kittui knows each by its name after workspace/, so its fields and
// the person's presets stay apart from any stage of another verb's of the same name.
package flow

import (
	"bytes"
	"encoding/json"
	"maps"
	"os"
	"slices"
	"strconv"
	"strings"

	"github.com/empowerite/kit/projects/tui/internal/style"
	"github.com/empowerite/kit/projects/tui/internal/table"
)

// workspaceRow is a row of a kit workspace stage as kit prints it: its key, its stage's own; its member, over a group;
// and where each field's value leads, by the field's id: a page, or row:<stage>/<key>, a row of another of its stages,
// or, for a list whose values lead apart, as a release's assets, a list of them, one a value. Its values are the row's
// other keys, by the field's id.
type workspaceRow struct {
	Key    string                     `json:"key"`
	Member string                     `json:"member"`
	Links  map[string]json.RawMessage `json:"links"`
}

// workspaceRows gives a kit workspace stage's records, as rowRecords reads them.
func (m Model) workspaceRows(s stage) []record { return m.rowRecords(s.name, s.rows, "") }

// rowRecords gives the records of a stage whose rows kit prints flat, as kit workspace's stages and incoming's: each
// row's value in each of the stage's fields, by the field's id, as rawValue reads it; its member's as every stage's, the
// row's own, else member, as incoming's rows carry none of their own; its links, a row of another stage after the
// member, as keyOf names it; and, in a field whose path is *, every value of the row no other field names, as
// otherValues gives them, so a row can say more than its catalog announces.
func (m Model) rowRecords(stage string, raws []json.RawMessage, member string) []record {
	fields := fieldsOf(stage)
	named := map[string]bool{"key": true, "member": true, "links": true}
	for _, f := range fields {
		named[f.ID] = true
	}
	var out []record
	for _, raw := range raws {
		var r workspaceRow
		var values map[string]json.RawMessage
		if json.Unmarshal(raw, &r) != nil || json.Unmarshal(raw, &values) != nil {
			continue
		}
		if r.Member == "" {
			r.Member = member
		}
		rec := record{row{Key: m.keyOf(r.Member, r.Key)}, map[string]table.Value{}}
		for _, f := range fields {
			switch {
			case f.ID == "member":
				rec.values[f.ID] = m.memberValue(r.Member)
				continue
			case f.Path == "*":
				rec.values[f.ID] = otherValues(values, named)
				continue
			}
			v := rawValue(values[f.ID])
			m.linkValue(&v, r.Links[f.ID], r.Member)
			rec.values[f.ID] = v
		}
		out = append(out, rec)
	}
	return out
}

// otherValues gives a row's values no field names, one a value, each as its key, a colon, and its value as JSON prints
// it, compact, the keys in order; a value that is null, an empty list, or an empty object says nothing, and is left out.
func otherValues(values map[string]json.RawMessage, named map[string]bool) table.Value {
	var out table.Value
	for _, k := range slices.Sorted(maps.Keys(values)) {
		var compact bytes.Buffer
		if named[k] || json.Compact(&compact, values[k]) != nil {
			continue
		}
		if text := compact.String(); text != "null" && text != "[]" && text != "{}" {
			out.Raw = append(out.Raw, k+": "+strings.Trim(text, `"`))
		}
	}
	return out
}

// rawValue reads a value as kit workspace prints it: none for null; one text, or a number's digits, for one; and one
// each for a list of them.
func rawValue(data json.RawMessage) table.Value {
	var one any
	if json.Unmarshal(data, &one) != nil {
		return table.Value{}
	}
	text := func(v any) string {
		switch v := v.(type) {
		case string:
			return v
		case float64:
			return strconv.FormatFloat(v, 'f', -1, 64)
		case bool:
			return strconv.FormatBool(v)
		}
		return ""
	}
	switch v := one.(type) {
	case nil:
		return table.Value{}
	case []any:
		var out table.Value
		for _, each := range v {
			out.Raw = append(out.Raw, text(each))
		}
		return out
	}
	return table.Some(text(one))
}

// linkValue gives a value where it leads, as a row's links say for its field: one link, the value's; a list, one a raw
// value, each where that value leads, and the value's own where it holds one alone; null, or none, nowhere.
func (m Model) linkValue(v *table.Value, link json.RawMessage, member string) {
	var one *string
	if json.Unmarshal(link, &one) == nil {
		if one != nil {
			v.Link = m.workspaceLink(*one, member)
		}
		return
	}
	var each []string
	if json.Unmarshal(link, &each) != nil {
		return
	}
	for _, l := range each {
		v.Links = append(v.Links, m.workspaceLink(l, member))
	}
	if len(v.Links) == 1 {
		v.Link = v.Links[0]
	}
}

// workspaceLink gives where a row's value leads: a page as it is, and row:<stage>/<key> as the link to that row of the
// workspace's stage, its key after the row's member, over a group.
func (m Model) workspaceLink(link, member string) string {
	to, ok := strings.CutPrefix(link, "row:")
	if !ok {
		return link
	}
	name, key, _ := strings.Cut(to, "/")
	return jumpTo(workspaceVerb+"/"+name, m.keyOf(member, key))
}

// The renderers of kit workspace's own values, registered with the table's by the names fields.json gives them.
func init() {
	table.Register("track", table.Each(trackText), false)
	table.Register("tree", treeCells, true)
}

// trackText draws what a branch is ahead and behind another, as kit gives them, [ahead, behind]: ↑3 ↓1, either alone,
// or = where it is neither.
func trackText(f table.Field, v table.Value) []string {
	if len(v.Raw) != 2 {
		return nil
	}
	var parts []string
	if v.Raw[0] != "0" {
		parts = append(parts, "↑"+v.Raw[0])
	}
	if v.Raw[1] != "0" {
		parts = append(parts, "↓"+v.Raw[1])
	}
	if len(parts) == 0 {
		return []string{"="}
	}
	return []string{strings.Join(parts, " ")}
}

// treeCells draws the worktrees a value names, by their paths, each after the field's glyph, ~ for the home folder: in
// a closed row, where a path is deep, … and its last two folders, as a clone's path is drawn; open, whole.
func treeCells(f table.Field, vs []table.Value) []cell {
	home, _ := os.UserHomeDir()
	out := make([]cell, len(vs))
	for k, v := range vs {
		for _, path := range v.Raw {
			out[k].Values = append(out[k].Values, f.Glyph+" "+tilde(path, home))
		}
		if len(v.Raw) == 1 {
			if parts := strings.Split(tilde(v.Raw[0], home), "/"); len(parts) > 3 {
				out[k].Summary = f.Glyph + " …/" + strings.Join(parts[len(parts)-2:], "/")
				if f.Dim {
					out[k].Summary = style.Off(out[k].Summary)
				}
			}
		}
	}
	return out
}
