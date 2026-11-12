// The introduction's failed checks, drawn as a table, internal/table's sheet, as the terms and the parts that customize
// kit are: one row a failed check, its fields in checks.json, the catalog. It groups by fix at first, so the checks one
// command fixes, as kit hydrate in every folder, stand under one heading; its presets group by folder or by step, or
// not at all. A row whose fix kit runs can be selected, as one of several, and Run Selected Fixes, on the line over the
// table, runs each fix selected in turn, once in a folder however many of its checks name it, then the checks again.
// The row under the cursor offers its own: r runs its fix, c copies it, and enter goes to its check, on its step, as a
// click on its step's glyph does.
package welcome

import (
	_ "embed"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/empowerite/kit/projects/tui/internal/grid"
	"github.com/empowerite/kit/projects/tui/internal/kit"
	"github.com/empowerite/kit/projects/tui/internal/pane"
	"github.com/empowerite/kit/projects/tui/internal/shell"
	"github.com/empowerite/kit/projects/tui/internal/style"
	"github.com/empowerite/kit/projects/tui/internal/table"
)

//go:embed checks.json
var checksJSON []byte

// checksTable is the name of the failed checks' table in the catalog, which keys the person's presets of it too.
const checksTable = "welcome/checks"

// checksCatalog is checks.json, read.
var checksCatalog = readCatalog(checksJSON)

// The words the line over the failed checks draws: Run Selected Fixes, which runs the fixes selected, and what the
// keys bar names it by, R; and copied, with the fix after it, once c copied a row's fix, until the next key or click.
const (
	runSelected    = "Run Selected Fixes"
	runSelectedKey = "R"
	copiedWord     = "copied"
)

// noFix is what a failed check with no fix draws in its fix's column, and its group's heading.
const noFix = "no fix"

// checkLink is how a row's step leads to its check, the check's key after it: a click on the step's glyph goes there,
// on the screen, as enter does.
const checkLink = "check:"

// newChecksSheet gives the failed checks' table, empty, the person's presets of it read from the file at path, and
// where a click on a row's step asks to go, which the screen then follows.
func newChecksSheet(path string, settings shell.Settings) (*table.Sheet, *string) {
	s := table.NewSheet(checksCatalog[checksTable], "checks", sheetColor, path)
	s.Settings(settings.FoldShared, settings.SharedEmoji)
	jump := new(string)
	s.SetBrowse(func(link string) error {
		*jump = link
		return table.Followed
	})
	return &s, jump
}

// folderValue gives what a failed check's folder column draws, and groups by: the machine's, for a check of the
// machine; else its repository's name, its folder's glyph and the last two parts of its path, so two worktrees named
// alike stand apart, and the branch it is on.
func folderValue(c kit.Check) string {
	if c.Folder == "" {
		return "💻 this machine"
	}
	glyph := worktreeGlyph
	if c.Desk {
		glyph = deskGlyph
	}
	text := short(c.Repository, 1) + " " + glyph + " " + short(filepath.ToSlash(c.Folder), 2)
	if c.Branch != "" {
		text += " on " + c.Branch
	}
	return text
}

// checkRecords gives the failed checks as the table's records, in the order of their steps, each keyed by its check's
// key: its place; its line; its fix, or none; its folder, as folderValue draws it; its step, by its glyph and name; and
// its note, later where it was left for later, and what its fix's run said.
func (m Model) checkRecords() []table.Record {
	var out []table.Record
	for n, i := range m.failed() {
		c := m.checks[i]
		t := m.tabs[m.stepOf(c)]
		fix := table.Some(c.Fix)
		if c.Fix == "" {
			fix.Else = noFix
		}
		var notes []string
		if m.deferred[c.Key()] {
			notes = append(notes, "later")
		}
		if nt, ok := m.notes[c.Key()]; ok {
			notes = append(notes, nt.text)
		}
		out = append(out, table.Record{Row: grid.Row{Key: c.Key()},
			Values: map[string]table.Value{"order": table.One(strconv.Itoa(n + 1)), "check": table.Some(c.Line),
				"fix": fix, "folder": table.One(folderValue(c)), "step": {Raw: []string{t.glyph + " " + t.name}, Link: checkLink + c.Key()},
				"note": table.Some(strings.Join(notes, " · "))}})
	}
	return out
}

// failed gives the indexes of the failed checks, in the order of their steps.
func (m Model) failed() []int {
	var out []int
	for i, c := range m.checks {
		if !c.OK {
			out = append(out, i)
		}
	}
	slices.SortStableFunc(out, func(a, b int) int { return m.stepOf(m.checks[a]) - m.stepOf(m.checks[b]) })
	return out
}

// checkAt gives the index of the check with the key, or -1.
func (m Model) checkAt(key string) int {
	return slices.IndexFunc(m.checks, func(c kit.Check) bool { return c.Key() == key })
}

// runnable says whether a check's fix is one kit runs.
func runnable(c kit.Check) bool { return !c.OK && c.Fix != "" && c.Runs }

// takeChecks gives the failed checks' table the checks as they now stand, and puts it on the introduction while some
// failed, kit's refusal of a run since over it: none while they first run, or when every one passed, where the
// introduction draws what they found as a page does. Only a row whose fix kit runs can be selected.
func (m *Model) takeChecks() {
	i := indexOf(m.tabs, introductionID)
	if i < 0 {
		return
	}
	if m.checksSheet == nil {
		m.checksSheet, m.jump = newChecksSheet(m.presets, m.settings)
	}
	failing := !m.checking.Loading && len(m.failed()) > 0
	if !failing {
		if m.tabs[i].sheet != nil && m.step == i {
			m.inPane = false
		}
		m.tabs[i].sheet = nil
		return
	}
	runs := map[string]bool{}
	for _, c := range m.checks {
		if runnable(c) {
			runs[c.Key()] = true
		}
	}
	m.checksSheet.SetRecords(m.checkRecords())
	m.checksSheet.SetPicks(func(key string) bool { return runs[key] })
	m.checksSheet.Search(m.searches[introductionID].Query)
	m.tabs[i].sheet = m.checksSheet
	m.placeChecks()
}

// introGridded says whether the selected step is the introduction drawn with its failed checks' table.
func (m Model) introGridded() bool { return m.isIntroduction(m.step) && m.gridded() }

// introHead gives the lines the introduction draws over its failed checks' table: its page's title and the blank line
// under it, kit's refusal of the checks' last run, in red, and a blank line, where kit refused it, then how many checks
// failed, with Run Selected Fixes after it, and a rule under that.
func (m Model) introHead() []string {
	page := m.pageLines(indexOf(m.tabs, introductionID))
	out := slices.Clone(page[:min(2, len(page))])
	if refusal := m.refusalLines(); len(refusal) > 0 {
		out = append(append(out, refusal...), "")
	}
	n := len(m.failed())
	head := strconv.Itoa(n) + " checks need your attention"
	if n == 1 {
		head = "1 check needs your attention"
	}
	line := lead + style.Label(head)
	if button := m.runButton(); button != "" {
		line += "   " + button
	}
	switch i := m.checkAt(m.copied); {
	case m.copied != "" && i >= 0:
		line += "  " + style.Off(copiedWord+": "+m.checks[i].Fix)
	case m.copied != "":
		line += "  " + style.Off(copiedWord)
	case m.checking.Running():
		line += "  " + style.Off("checking…")
	}
	out = append(out, line, lead+style.Off(strings.Repeat("─", ansi.StringWidth(head))))
	for k := range out {
		out[k] = pane.Clip(out[k], m.width-1)
	}
	return out
}

// runButton gives Run Selected Fixes as the line over the table draws it, a cell either side: in the accent, with how
// many are selected, while some are; dim while none is, as a choice not open now; and nothing where no failed check's
// fix is one kit runs.
func (m Model) runButton() string {
	if m.checksSheet == nil || !slices.ContainsFunc(m.checks, runnable) {
		return ""
	}
	n := len(m.checksSheet.Selected())
	if n == 0 {
		return style.Off(" " + runSelected + " ")
	}
	return style.Tint(" " + runSelected + " (" + strconv.Itoa(n) + ") ")
}

// onRunButton says whether the screen cell x, y is on Run Selected Fixes.
func (m Model) onRunButton(x, y int) bool {
	head := m.introHead()
	k := y - paneTop
	if k < 0 || k >= len(head) || m.runButton() == "" {
		return false
	}
	return onButton(ansi.Strip(head[k]), runSelected, x) || onSelected(ansi.Strip(head[k]), x)
}

// onSelected says whether the cell x of the line is on Run Selected Fixes with its count, as runButton draws it.
func onSelected(line string, x int) bool {
	i := strings.Index(line, " "+runSelected+" (")
	if i < 0 {
		return false
	}
	at := ansi.StringWidth(line[:i])
	end := strings.Index(line[i:], ") ")
	return end >= 0 && x >= at && x < at+ansi.StringWidth(line[i:i+end+2])
}

// placeChecks places the failed checks' table under the introduction's head, while it shows there, as tall as its
// lines, or the pane under the head when they are more, so the page's words follow it where they fit.
func (m *Model) placeChecks() {
	if i := indexOf(m.tabs, introductionID); i < 0 || m.checksSheet == nil || m.tabs[i].sheet != m.checksSheet {
		return
	}
	top, room := paneTop+len(m.introHead()), m.paneHeight()-len(m.introHead())
	m.checksSheet.Place(top, max(1, room), m.width, m.height)
	view := m.checksSheet.View()
	need := len(view)
	for need > 1 && view[need-1] == "" {
		need--
	}
	if need < room {
		m.checksSheet.Place(top, need, m.width, m.height)
	}
}

// introPane gives the introduction's pane while its failed checks' table shows: its head, the table, and, where the
// pane has room after it, a blank line, the folders not cloned, the folders the checks are on while they run again,
// and the rest of the page.
func (m Model) introPane() []string {
	out := m.introHead()
	for _, l := range m.sheet().View() {
		out = append(out, pane.Clip(l, m.width-1))
	}
	tail := []string{""}
	if gone := uncloned(m.workspace); len(gone) > 0 {
		tail = append(tail, lead+style.Off("not cloned: "+strings.Join(gone, ", ")), "")
	}
	if m.checking.Running() {
		m.addChecking(func(_ int, text string) { tail = append(tail, text) })
		tail = append(tail, "")
	}
	page := m.pageLines(m.step)
	tail = append(tail, page[min(2, len(page)):]...)
	for _, l := range tail {
		if len(out) >= m.paneHeight() {
			break
		}
		out = append(out, pane.Clip(l, m.width-1))
	}
	return out
}

// checkUnder gives the index of the check whose row is under the table's cursor while its rows have the keyboard, or
// -1 on a group's heading, or with the keyboard elsewhere.
func (m Model) checkUnder() int {
	s := m.sheet()
	rows := s.Rows()
	if !s.InRows() || s.Table().Cursor >= len(rows) {
		return -1
	}
	r := rows[s.Table().Cursor]
	if r.Aside || grid.Heads(r) {
		return -1
	}
	return m.checkAt(r.Key)
}

// jumped goes to the check whose row's step a click on the table asked for, if one did.
func (m *Model) jumped() {
	if m.jump == nil || *m.jump == "" {
		return
	}
	key := strings.TrimPrefix(*m.jump, checkLink)
	*m.jump = ""
	if i := m.checkAt(key); i >= 0 {
		m.cursor = i
		m.follow()
	}
}

// introKey takes a key the introduction's failed checks' table did not: R runs the fixes selected; and on a check's
// row, r runs its fix, where kit runs it, c copies it, and enter goes to the check, on its step. It says whether it
// took the key.
func (m *Model) introKey(k tea.KeyPressMsg) (tea.Cmd, bool) {
	if k.String() == runSelectedKey {
		return m.runSelected(), true
	}
	i := m.checkUnder()
	if !m.inPane || i < 0 {
		return nil, false
	}
	m.cursor = i
	switch c := m.checks[i]; k.String() {
	case "r":
		if runnable(c) {
			return m.now(), true
		}
	case "c":
		if c.Fix != "" {
			return m.copy(), true
		}
	case "enter":
		m.follow()
		return nil, true
	}
	return nil, false
}

// introKeys gives the keys the introduction adds to its table's rows' while they have the keyboard: enter, r, and c
// on a check's row, as introKey says, and R while some row is selected.
func (m Model) introKeys(seg shell.Segment) shell.Segment {
	if i := m.checkUnder(); i >= 0 {
		seg.Bindings = append(seg.Bindings, shell.Bind("⏎", "go to the check", "enter"))
		if runnable(m.checks[i]) {
			seg.Bindings = append(seg.Bindings, shell.Bind("r", "run"))
		}
		if m.checks[i].Fix != "" {
			seg.Bindings = append(seg.Bindings, shell.Bind("c", "copy"))
		}
	}
	if len(m.checksSheet.Selected()) > 0 {
		seg.Bindings = append(seg.Bindings, shell.Bind(runSelectedKey, "run selected"))
	}
	return seg
}

// runSelected runs the fixes of the checks selected, each in turn, once in a folder however many of its checks name
// it, the selection then cleared; when the last ends, the checks run again, as after one fix. None selected runs none.
func (m *Model) runSelected() tea.Cmd {
	seen := map[string]bool{}
	m.queue = nil
	for _, key := range m.checksSheet.Selected() {
		i := m.checkAt(key)
		if i < 0 || !runnable(m.checks[i]) {
			continue
		}
		c := m.checks[i]
		if where := c.Folder + "\n" + c.Fix; !seen[where] {
			seen[where] = true
			m.queue = append(m.queue, c)
		}
	}
	m.checksSheet.Unselect()
	return m.runNext()
}

// runNext hands the terminal to the next fix of those runSelected queued, none when none is left.
func (m *Model) runNext() tea.Cmd {
	if len(m.queue) == 0 {
		return nil
	}
	c := m.queue[0]
	m.queue = m.queue[1:]
	delete(m.deferred, c.Key())
	id, key := c.ID, c.Key()
	return m.exec(m.over().WelcomeFix(c), func(err error) tea.Msg { return fixedMsg{id: id, key: key, err: err} })
}
