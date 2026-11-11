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
	"strings"

	"github.com/empowerite/kit/projects/tui/internal/style"
	"github.com/empowerite/kit/projects/tui/internal/table"
)

// catalogRow is what a row a stage's catalog draws holds beside its values: its member, over a group, and where each
// field's value leads, by the field's id: a page, or row:<stage>/<key>, a row of another of its stages, or, for a list
// whose values lead apart, as a release's assets, a list of them, one a value.
type catalogRow struct {
	Member string                     `json:"member"`
	Links  map[string]json.RawMessage `json:"links"`
}

// catalogRows gives the records of a stage whose rows the catalog draws, as kit prints them flat: each row's key, as
// the entry's key reads it, after its member over a group, as keyOf names it; its value in each of the stage's
// fields, as the field's path reads it, and where it leads, as the row's links say; its member's as every stage's;
// and, in a field whose path is *, every value of the row no other field names, as otherValues gives them, so a row
// can say more than its catalog announces.
func (m Model) catalogRows(s stage) []record {
	c := catalog[s.name]
	named := map[string]bool{"key": true, "member": true, "links": true}
	for _, f := range c.Of() {
		named[f.ID] = true
	}
	var out []record
	for _, raw := range s.rows {
		var r catalogRow
		var values map[string]json.RawMessage
		if json.Unmarshal(raw, &r) != nil || json.Unmarshal(raw, &values) != nil {
			continue
		}
		key, _ := c.KeyOf(raw)
		rec := record{row{Key: m.keyOf(r.Member, key)}, map[string]table.Value{}}
		for _, f := range c.Of() {
			switch {
			case f.ID == "member":
				rec.values[f.ID] = m.memberValue(r.Member)
				continue
			case f.Path == "*":
				rec.values[f.ID] = otherValues(values, named)
				continue
			}
			v := f.Read(raw)
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
