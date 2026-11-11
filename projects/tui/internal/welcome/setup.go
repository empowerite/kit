// The setup step as one table: the machine's checks, its own group first, each a row as any step's check is; then a
// group per repository, a row per folder, the desk first, as kit gives them: its name, its branch, the kit its kit.yml
// declares, and the state of each of its checks, a column each; and under a folder's row, its failed checks, each
// offering its fix as any failed check does. A kit that names no folder draws the step as any other step with checks.
package welcome

import (
	"slices"
	"strings"

	"github.com/charmbracelet/x/ansi"

	"github.com/empowerite/kit/projects/tui/internal/kit"
	"github.com/empowerite/kit/projects/tui/internal/pane"
	"github.com/empowerite/kit/projects/tui/internal/style"
)

// setupID is the id of the setup step, whose checks are drawn as a table where kit names their folders.
const setupID = "setup"

// column is one of a folder row's columns of checks: its heading, and whether a check's id is one it draws.
type column struct {
	head  string
	holds func(id string) bool
}

// is gives the test of a column that draws the one id.
func is(id string) func(string) bool { return func(other string) bool { return other == id } }

// columns are a folder row's columns of checks, in order. tools draws every check the others do not: lefthook, the
// dependency providers, the linters, and any a later kit adds.
var columns = []column{
	{"git", is("repo-git")}, {"kit", is("kit-installed")}, {".kit", is("kit-hydrated")}, {"hooks", is("hooks-installed")},
	{"toolchain", is("toolchain-project")}, {"tools", nil}, {"check", is("kit-check")},
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

// The widths of a folder row's columns before its checks', in cells, at most: a folder's name, its branch, its kit.
const (
	nameMost   = 28
	branchMost = 22
	kitMost    = 12
)

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

// foldered says whether the setup step's checks name their folders, as a kit that judges each folder does, so the step
// draws them as a table.
func (m Model) foldered(step int) bool {
	if m.tabs[step].id != setupID {
		return false
	}
	return slices.ContainsFunc(m.checks, func(c kit.Check) bool { return c.Folder != "" && m.stepOf(c) == step })
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

// widths gives the widths of the name, branch, and kit columns: as wide as the widest of each, or its heading, and no
// wider than its most.
func widths(groups []repoGroup) (name, branch, version int) {
	name, branch, version = len("folder"), len("branch"), len("kit")
	for _, g := range groups {
		for _, f := range g.folders {
			name = max(name, ansi.StringWidth(folderName(f.head)))
			branch = max(branch, ansi.StringWidth(f.head.Branch))
			version = max(version, ansi.StringWidth(f.head.Kit))
		}
	}
	return min(name, nameMost), min(branch, branchMost), min(version, kitMost)
}

// cell gives a text fitted to a column of the width: cut with … where it is wider, padded where it is narrower, and a
// gap after it.
func cell(text string, width int) string {
	text = ansi.Truncate(text, width, "…")
	return text + strings.Repeat(" ", max(0, width-ansi.StringWidth(text))) + "  "
}

// states gives the cells of a folder's checks, a column each, with no gap after the last: ❌ where one it draws failed,
// ✅ where every one it draws holds, and a dim · where kit judged none there, as a folder with no kit.yml has no .kit/
// to check.
func (m Model) states(f folderRow) string {
	failed, held := make([]bool, len(columns)), make([]bool, len(columns))
	for _, i := range f.checks {
		k := columnOf(m.checks[i].ID)
		if m.checks[i].OK {
			held[k] = true
		} else {
			failed[k] = true
		}
	}
	var b strings.Builder
	for k, c := range columns {
		w := max(ansi.StringWidth(c.head), 2)
		glyph := style.Off("·") + " "
		switch {
		case failed[k]:
			glyph = Failed
		case held[k]:
			glyph = Passed
		}
		b.WriteString(glyph + strings.Repeat(" ", w-2) + "  ")
	}
	return strings.TrimRight(b.String(), " ")
}

// headings gives the table's heading line: folder, branch, kit, and each column of checks, dim.
func headings(name, branch, version int) string {
	text := cell("folder", name) + cell("branch", branch) + cell("kit", version)
	for _, c := range columns {
		text += cell(c.head, max(ansi.StringWidth(c.head), 2))
	}
	return style.Off(strings.TrimRight(text, " "))
}

// shows says whether the table draws a folder's row under the search: with nothing typed; where its repository, path,
// or branch holds what is typed; or where one of its checks is among the step's rows.
func (m Model) shows(f folderRow, rows []int, query string) bool {
	if query == "" || matches(f.head.Repository, query) || matches(f.head.Folder, query) ||
		matches(f.head.Branch, query) {
		return true
	}
	return slices.ContainsFunc(f.checks, func(i int) bool { return slices.Contains(rows, i) })
}

// setupLines gives the setup step's lines as a table: its page, kit's refusal, the machine's checks, then each
// repository's folders, each row followed by its failed checks, and the checks left for later.
func (m Model) setupLines() []line {
	w := m.width - 1
	out := m.stepHead()
	add := func(row int, text string) { out = append(out, line{row, pane.Clip(text, w)}) }
	query := m.search().Query
	rows := m.rows(m.step)
	var machine []int
	for _, i := range rows {
		if m.checks[i].Folder == "" {
			machine = append(machine, i)
		}
	}
	if len(machine) > 0 {
		add(-1, lead+style.Label("This machine"))
		for _, i := range machine {
			out = append(out, m.checkLines(i, m.checks[i].Key(), "")...)
		}
		add(-1, "")
	}
	groups := m.groups(m.step)
	name, branch, version := widths(groups)
	headed := false
	for _, g := range groups {
		var shown []folderRow
		for _, f := range g.folders {
			if m.shows(f, rows, query) {
				shown = append(shown, f)
			}
		}
		if len(shown) == 0 {
			continue
		}
		if !headed {
			add(-1, lead+"  "+headings(name, branch, version))
			headed = true
		}
		add(-1, lead+style.Repo(g.repository))
		for _, f := range shown {
			kitText := f.head.Kit
			if kitText == "" {
				kitText = style.Off("none")
			}
			add(-1, lead+"  "+cell(folderName(f.head), name)+cell(f.head.Branch, branch)+cell(kitText, version)+
				m.states(f))
			for _, i := range f.checks {
				if slices.Contains(rows, i) {
					out = append(out, m.checkLines(i, m.checks[i].Key(), "    ")...)
				}
			}
		}
	}
	if !headed && len(machine) == 0 {
		switch {
		case m.checking.Running():
			add(-1, lead+style.Off("checking…"))
		case query != "":
			add(-1, lead+style.Off("no check matches the search"))
		}
		return out
	}
	return append(out, m.stepTail(rows)...)
}
