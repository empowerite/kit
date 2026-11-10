// The 📚 repos tab, first in kit workspace's picture: a row per member of the workspace, as kit workspace repos --json
// gives it: where it is cloned, its branch, its worktrees, and the branches no checkout has.
package flow

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/empowerite/kit/projects/tui/internal/style"
	"github.com/empowerite/kit/projects/tui/internal/table"
)

// repo is one member of the workspace, as kit flow repos --json gives it: its host and path, its primary remote's
// forge, its remotes, its clone or none, the source that made it a member, and its desk's branch, with the commits it is
// ahead and behind, its linked worktrees, each on its branch or none, and its local branches no checkout has, none for
// a member not cloned; Error is the desk's line where it failed.
type repo struct {
	Repository string  `json:"repository"`
	Forge      *string `json:"forge"`
	Remotes    []struct {
		Name  string `json:"name"`
		URL   string `json:"url"`
		Forge string `json:"forge"`
	} `json:"remotes"`
	Checkout  *string `json:"checkout"`
	By        string  `json:"by"`
	Branch    *string `json:"branch"`
	Ahead     *int    `json:"ahead"`
	Behind    *int    `json:"behind"`
	Worktrees []struct {
		Branch *string `json:"branch"`
		Path   string  `json:"path"`
	} `json:"worktrees"`
	Branches []string `json:"branches"`
	Error    *string  `json:"error"`
}

// repoRows gives each member: its short name, forge, remotes as pills, clone, or dim not cloned, branch with what it is
// ahead and behind, its worktrees by branch, and the branches no checkout has, and the source that made it a member. A
// member whose desk could not be read shows ❌ before its clone, and kit's line when it is opened.
func (m Model) repoRows(s stage) []record {
	var out []record
	for _, r := range s.repos {
		var labels []table.Label
		for _, remote := range r.Remotes {
			labels = append(labels, table.Label{Name: remote.Name})
		}
		where := table.Value{Else: "not cloned"}
		if r.Checkout != nil {
			where = table.Value{Raw: []string{*r.Checkout}, Of: r.Error != nil}
		}
		var trees table.Value
		for _, w := range r.Worktrees {
			if w.Branch != nil {
				trees.Raw = append(trees.Raw, *w.Branch)
			} else {
				trees.Raw = append(trees.Raw, filepath.Base(w.Path))
			}
			detached, _ := trees.Of.([]bool)
			trees.Of = append(detached, w.Branch == nil)
		}
		var branch, ahead table.Value
		if r.Branch != nil {
			c := checkout{Branch: *r.Branch}
			if r.Ahead != nil {
				c.Ahead = *r.Ahead
			}
			if r.Behind != nil {
				c.Behind = *r.Behind
			}
			branch = table.Value{Raw: []string{c.Branch}, Of: c}
			ahead = table.Value{Raw: []string{strconv.Itoa(c.Ahead)}, Of: c}
		}
		forge := ""
		if r.Forge != nil {
			forge = *r.Forge
		}
		values := map[string]table.Value{
			"repository": {Raw: []string{r.Repository}, Of: m.short(r.Repository)}, "forge": table.Some(forge),
			"remotes": table.Labelled(labels), "checkout": where, "branch": branch, "ahead": ahead,
			"worktrees": trees, "branches": {Raw: r.Branches}, "by": table.Some(r.By),
		}
		rec := record{row{Key: r.Repository}, values}
		if r.Error != nil {
			rec.Detail = [][]cell{span(style.Refusal(Failed + " " + *r.Error))}
		}
		out = append(out, rec)
	}
	return out
}

// pathCells draws a clone's path, ~ for the home folder, after ❌ where its desk failed: whole in an open row, and in a
// closed one, where it has more, … and its last two folders, so a deep clone keeps the columns beside it on the screen.
func pathCells(f table.Field, vs []table.Value) []cell {
	home, _ := os.UserHomeDir()
	out := make([]cell, len(vs))
	for k, v := range vs {
		if len(v.Raw) == 0 {
			continue
		}
		whole, lead := tilde(v.Raw[0], home), ""
		if failed, _ := v.Of.(bool); failed {
			lead = Failed + " "
		}
		out[k].Values = []string{lead + whole}
		if parts := strings.Split(whole, "/"); len(parts) > 3 {
			out[k].Summary = lead + "…/" + strings.Join(parts[len(parts)-2:], "/")
			if f.Dim {
				out[k].Summary = style.Off(out[k].Summary)
			}
		}
	}
	return out
}

// worktreeNames draws each worktree after the field's glyph, by its branch, or by its folder after ❗ where it is on
// none.
func worktreeNames(f table.Field, v table.Value) []string {
	detached, _ := v.Of.([]bool)
	out := make([]string, len(v.Raw))
	for k, name := range v.Raw {
		out[k] = f.Glyph + " " + name
		if k < len(detached) && detached[k] {
			out[k] += " " + Detached
		}
	}
	return out
}

// tilde gives a path with ~ for the home folder it is in, or is.
func tilde(path, home string) string {
	switch {
	case home == "" || home == "/":
		return path
	case path == home:
		return "~"
	case strings.HasPrefix(path, home+"/"):
		return "~" + strings.TrimPrefix(path, home)
	}
	return path
}
