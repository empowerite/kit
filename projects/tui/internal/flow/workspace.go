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

// catalogRows gives the records of a stage whose rows the catalog draws, as kit prints them flat, from each record as
// decode read it, which none of this decodes again: each row's key, after its member over a group, as keyOf names it;
// its value in each of the stage's fields, as the field's path reads it, and where it leads, as the row's links say;
// its member's as every stage's; and, in a field whose path is *, every value of the row no other field names, as
// otherValues gives them, so a row can say more than its catalog announces.
func (m Model) catalogRows(s stage) []record {
	fields := catalog[s.name].Of()
	named := map[string]bool{"key": true, "member": true, "links": true}
	for _, f := range fields {
		named[f.ID] = true
	}
	out := make([]record, 0, len(s.parsed))
	for _, p := range s.parsed {
		rec := record{row{Key: m.keyOf(p.member, p.key)}, make(map[string]table.Value, len(fields))}
		for _, f := range fields {
			switch {
			case f.ID == "member":
				rec.values[f.ID] = m.memberValue(p.member)
				continue
			case f.Path == "*":
				object, _ := p.value.(map[string]any)
				rec.values[f.ID] = otherValues(object, named)
				continue
			}
			v := f.ReadFrom(p.value)
			m.linkValue(&v, p.links[f.ID], p.member)
			rec.values[f.ID] = v
		}
		out = append(out, rec)
	}
	return out
}

// otherValues gives a row's values no field names, one a value, each as its key, a colon, and its value as JSON prints
// it, compact, the keys in order; a value that is null, an empty list, or an empty object says nothing, and is left out.
func otherValues(values map[string]any, named map[string]bool) table.Value {
	var out table.Value
	for _, k := range slices.Sorted(maps.Keys(values)) {
		if named[k] {
			continue
		}
		var compact bytes.Buffer
		e := json.NewEncoder(&compact)
		e.SetEscapeHTML(false)
		_ = e.Encode(values[k]) // a value decoded from JSON encodes again
		if text := strings.TrimSpace(compact.String()); text != "null" && text != "[]" && text != "{}" {
			out.Raw = append(out.Raw, k+": "+strings.Trim(text, `"`))
		}
	}
	return out
}

// linkValue gives a value where it leads, as a row's links say for its field, as decode read them: one link, the
// value's; a list, one a raw value, each where that value leads, and the value's own where it holds one alone; null,
// or none, nowhere.
func (m Model) linkValue(v *table.Value, link any, member string) {
	switch link := link.(type) {
	case string:
		v.Link = m.workspaceLink(link, member)
	case []any:
		for _, l := range link {
			v.Links = append(v.Links, m.workspaceLink(l.(string), member))
		}
		if len(v.Links) == 1 {
			v.Link = v.Links[0]
		}
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
