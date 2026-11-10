// kit workspace's stages, the picture of what git holds across the workspace: remotes, branches, worktrees, commits,
// and tags. Their rows are drawn from the catalog alone: kit prints each row's values by the field's id, and where a
// value leads, so a stage needs no Go of its own. kittui knows each by its name after workspace/, so its fields and
// the person's presets stay apart from kit flow's stage of the same name, as worktrees.
package flow

import (
	"encoding/json"
	"os"
	"strconv"
	"strings"

	"github.com/empowerite/kit/projects/tui/internal/style"
	"github.com/empowerite/kit/projects/tui/internal/table"
)

// workspaceRow is a row of a kit workspace stage as kit prints it: its key, its stage's own; its member, over a group;
// and where each field's value leads, by the field's id: a page, or row:<stage>/<key>, a row of another of its stages.
// Its values are the row's other keys, by the field's id.
type workspaceRow struct {
	Key    string             `json:"key"`
	Member string             `json:"member"`
	Links  map[string]*string `json:"links"`
}

// workspaceRows gives a kit workspace stage's records: each row's value in each of the stage's fields, as rawValue reads
// it, its member's as every stage's, and its links, a row of another stage after the member, as keyOf names it.
func (m Model) workspaceRows(s stage) []record {
	fields := fieldsOf(s.name)
	var out []record
	for _, raw := range s.rows {
		var r workspaceRow
		var values map[string]json.RawMessage
		if json.Unmarshal(raw, &r) != nil || json.Unmarshal(raw, &values) != nil {
			continue
		}
		rec := record{row{Key: m.keyOf(r.Member, r.Key)}, map[string]table.Value{}}
		for _, f := range fields {
			if f.ID == "member" {
				rec.values[f.ID] = m.memberValue(r.Member)
				continue
			}
			v := rawValue(values[f.ID])
			if link := r.Links[f.ID]; link != nil {
				v.Link = m.workspaceLink(*link, r.Member)
			}
			rec.values[f.ID] = v
		}
		out = append(out, rec)
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
