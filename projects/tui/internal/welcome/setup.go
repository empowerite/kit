// The setup step's folders as a table, internal/table's sheet, as every other screen draws one: a row a folder, the
// desk first, as kit gives them, its name, its branch, the kit its kit.yml declares, and the state of each of its checks,
// a column each; the machine's checks first, a row each, in a group of their own; the members of a group with no
// checkout last. Under a row that is open, its failed checks and its offers, each with its fix as any failed check has
// one, and its buttons while the row is under the cursor. A row that has something to say stands open until it is
// closed. A kit that names no folder draws the step as any other step with checks.
package welcome

import (
	_ "embed"
	"path/filepath"
	"slices"
	"strings"

	"github.com/empowerite/kit/projects/tui/internal/fix"
	"github.com/empowerite/kit/projects/tui/internal/grid"
	"github.com/empowerite/kit/projects/tui/internal/kit"
	"github.com/empowerite/kit/projects/tui/internal/style"
	"github.com/empowerite/kit/projects/tui/internal/table"
)

//go:embed folders.json
var foldersJSON []byte

// foldersTable is the name of the folders' table in the catalog, which keys the person's presets of it too.
const foldersTable = "welcome/folders"

// foldersCatalog is folders.json, read.
var foldersCatalog = readCatalog(foldersJSON)

// setupID is the id of the setup step, whose checks are drawn as a table where kit names their folders.
const setupID = "setup"

// machineGroup is the group of the machine's checks, which name no repository.
const machineGroup = "This machine"

// The glyphs of a folder's name: a scratch worktree, one left in a temporary folder, besides the desk's and a
// worktree's, members.go's.
const scratchGlyph = "🧹"

// scratchID is the id of the check that a worktree stands in a temporary folder, outside its desk.
const scratchID = "worktree-scratch"

// column is one of a folder row's columns of checks: its field, and whether a check's id is one it draws.
type column struct {
	field string
	holds func(id string) bool
}

// is gives the test of a column that draws the one of the ids.
func is(ids ...string) func(string) bool {
	return func(other string) bool { return slices.Contains(ids, other) }
}

// columns are a folder row's columns of checks, in order. tools draws every check the others do not: lefthook, the
// dependency providers, the linters, and any a later kit adds.
var columns = []column{
	{"git", is("repo-git", scratchID)}, {"installed", is("kit-installed")}, {"hydrated", is("kit-hydrated")},
	{"hooks", is("hooks-installed")}, {"toolchain", is("toolchain-project")}, {"tools", nil},
	{"check", is("kit-check")},
}

// columnOf gives the index of the column that draws a check's id.
func columnOf(id string) int {
	tools := 0
	for k, c := range columns {
		if c.holds == nil {
			tools = k
		} else if c.holds(id) {
			return k
		}
	}
	return tools
}

// foldered says whether the setup step's checks name their folders, as a kit that judges each folder does, so the step
// draws them as a table.
func (m Model) foldered(step int) bool {
	if m.tabs[step].id != setupID {
		return false
	}
	return slices.ContainsFunc(m.checks, func(c kit.Check) bool { return c.Folder != "" && m.stepOf(c) == step })
}

// folderRow is one folder of the table: the first of its checks, which says the folder, and the indexes of all its
// checks, in kit's order.
type folderRow struct {
	head   kit.Check
	checks []int
}

// repoGroup is one repository of the table, and its folders in kit's order, the desk first.
type repoGroup struct {
	repository string
	folders    []folderRow
}

// groups gives the setup step's folders by repository, each in the order kit first names it.
func (m Model) groups(step int) []repoGroup {
	var out []repoGroup
	for i, c := range m.checks {
		if c.Folder == "" || m.stepOf(c) != step {
			continue
		}
		g := slices.IndexFunc(out, func(r repoGroup) bool { return r.repository == c.Repository })
		if g < 0 {
			out = append(out, repoGroup{repository: c.Repository})
			g = len(out) - 1
		}
		f := slices.IndexFunc(out[g].folders, func(r folderRow) bool { return r.head.Folder == c.Folder })
		if f < 0 {
			out[g].folders = append(out[g].folders, folderRow{head: c})
			f = len(out[g].folders) - 1
		}
		out[g].folders[f].checks = append(out[g].folders[f].checks, i)
	}
	return out
}

// folderNames gives each folder of a repository its name: its folder's, and, where another folder of the repository
// has the same, as scratch worktrees under different parents do, its parent's before it, and as many parents as it
// takes to tell them apart. A folder is by its path, so two never share a name for long.
func folderNames(folders []folderRow) map[string]string {
	depth := make([]int, len(folders))
	segments := make([][]string, len(folders))
	for k, f := range folders {
		segments[k] = strings.Split(strings.Trim(filepath.ToSlash(f.head.Folder), "/"), "/")
		depth[k] = 1
	}
	name := func(k int) string { return strings.Join(segments[k][len(segments[k])-depth[k]:], "/") }
	for changed := true; changed; {
		changed = false
		now := make([]string, len(folders))
		for k := range folders {
			now[k] = name(k)
		}
		var clashes []int
		for k := range folders {
			if n := slices.Index(now, now[k]); n != k || slices.Index(now[k+1:], now[k]) >= 0 {
				clashes = append(clashes, k)
			}
		}
		for _, k := range clashes {
			if depth[k] < len(segments[k]) {
				depth[k]++
				changed = true
			}
		}
	}
	out := map[string]string{}
	for k, f := range folders {
		out[f.head.Folder] = name(k)
	}
	return out
}

// scratch says whether a folder's checks say it is a scratch worktree: the check of that, failed.
func (m Model) scratch(f folderRow) bool {
	return slices.ContainsFunc(f.checks, func(i int) bool { return m.checks[i].ID == scratchID && !m.checks[i].OK })
}

// glyphOf gives the glyph a folder's name starts with: the desk's, a worktree's, or a scratch worktree's.
func (m Model) glyphOf(f folderRow) string {
	switch {
	case m.scratch(f):
		return scratchGlyph
	case f.head.Desk:
		return deskGlyph
	}
	return worktreeGlyph
}

// state gives the value of a column from its checks: failed where one it draws failed, else offered where one is an
// offer, else held where every one holds; none where kit judged none there, as a folder with no kit.yml has no .kit/
// to check.
func (m Model) state(checks []int, k int) table.Value {
	state := ""
	for _, i := range checks {
		if columnOf(m.checks[i].ID) != k {
			continue
		}
		switch c := m.checks[i]; {
		case !c.OK:
			state = "failed"
		case c.Offer() && state != "failed":
			state = "offered"
		case state == "":
			state = "held"
		}
	}
	return table.Some(state)
}

// current gives the check whose buttons the table draws, or -1: the one the cursor is on, while the rows have the
// keyboard.
func (m Model) current() int {
	if m.gridded() && m.sheet().InRows() && m.cursor < len(m.checks) {
		return m.cursor
	}
	return -1
}

// details gives the lines under a check's row: ✅, 💡 for an offer, or ❌, and its line, its findings, its fix, where to
// read more, and what the screen said of its fix, each line under the first indented; the buttons last on the fix, or
// on the line, while the check is the current one.
func (m Model) details(i int) [][]grid.Cell {
	c := m.checks[i]
	glyph := Passed
	if !c.OK {
		glyph = Failed
	} else if c.Offer() {
		glyph = Offered
	}
	end := ""
	if i == m.current() {
		end = m.rowButtons(c, c.Key())
	}
	lines := []string{glyph + " " + c.Line}
	for _, f := range c.Findings {
		lines = append(lines, "   "+f)
	}
	if c.Fix != "" {
		shown := fix.In(c.Folder, c.Fix)
		if c.Interactive {
			shown += " · it needs you at the keyboard"
		}
		for _, l := range strings.Split(shown, "\n") {
			lines = append(lines, "   "+style.Off(l))
		}
		lines[len(lines)-1] += end
		end = ""
	}
	if !c.OK && c.More != "" {
		lines = append(lines, "   "+style.Off(c.More))
	}
	if n, ok := m.notes[c.Key()]; ok {
		draw := style.Warn
		if n.refusal {
			draw = style.Refusal
		}
		lines = append(lines, "   "+draw(n.text))
	}
	lines[0] += end
	out := make([][]grid.Cell, len(lines))
	for k, l := range lines {
		out[k] = []grid.Cell{{Values: []string{l}, Wrap: true}}
	}
	return out
}

// setupRecords gives the table's records: the machine's checks, each a row, then each folder, then the members of the
// workspace's group with no checkout.
func (m Model) setupRecords(step int) []table.Record {
	var out []table.Record
	for i, c := range m.checks {
		if c.Folder != "" || m.stepOf(c) != step || folded(c) {
			continue
		}
		values := map[string]table.Value{"repository": table.One(machineGroup), "folder": table.One(c.ID)}
		k := columnOf(c.ID)
		for j, col := range columns {
			values[col.field] = table.Value{}
			if j == k {
				values[col.field] = table.One(map[bool]string{true: "held", false: "failed"}[c.OK])
				if c.Offer() {
					values[col.field] = table.One("offered")
				}
			}
		}
		out = append(out, table.Record{Row: grid.Row{Key: c.Key(), Detail: m.details(i)}, Values: values})
	}
	for _, g := range m.groups(step) {
		names := folderNames(g.folders)
		for _, f := range g.folders {
			var detail [][]grid.Cell
			for _, i := range f.checks {
				if asks(m.checks[i]) {
					detail = append(detail, m.details(i)...)
				}
			}
			branch := table.Value{Else: "none"}
			if f.head.Branch != "" {
				branch = table.One(f.head.Branch)
			}
			version := table.Value{Else: "none"}
			if f.head.Kit != "" {
				version = table.One(f.head.Kit)
			}
			values := map[string]table.Value{"repository": table.One(g.repository),
				"folder": table.One(m.glyphOf(f) + " " + names[f.head.Folder]), "branch": branch, "kit": version}
			for k, col := range columns {
				values[col.field] = m.state(f.checks, k)
				if len(values[col.field].Raw) == 0 {
					values[col.field] = table.Value{Else: "·"}
				}
			}
			out = append(out, table.Record{Row: grid.Row{Key: f.head.Folder, Detail: detail}, Values: values})
		}
	}
	for _, member := range m.workspace.Members {
		if member.Checkout == "" && m.foldered(step) {
			values := map[string]table.Value{"repository": table.One(member.Repository),
				"folder": {Else: "not cloned"}}
			out = append(out, table.Record{Row: grid.Row{Key: "uncloned " + member.Repository}, Values: values})
		}
	}
	return out
}
