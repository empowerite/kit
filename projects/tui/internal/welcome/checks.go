// The introduction's failed checks, drawn as a table, internal/table's sheet, as the terms and the parts that customize
// kit are: one row a failed check, its fields in checks.json, the catalog. It groups by fix at first, so the checks one
// command fixes, as kit hydrate in every folder, stand under one heading; its presets group by folder or by step, or
// not at all. A row whose fix kit runs can be selected, as one of several, and Run Selected Fixes, on the line over the
// table, runs each fix selected in turn, once in a folder however many of its checks name it, then the checks again.
// The row under the cursor offers its own fix, as internal/fix draws the pair at its first line's end: ⧉, or c, copies
// it, copied in its place until the next key or click, and ▶️, or r, runs it where kit runs it; enter goes to its
// check, on its step, as a click on its step's glyph does.
package welcome

import (
	_ "embed"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/empowerite/kit/projects/tui/internal/fix"
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
// keys bar names it by, R.
const (
	runSelected    = "Run Selected Fixes"
	runSelectedKey = "R"
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
	s.Settings(settings)
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
// its note, what its fix's run said.
func (m Model) checkRecords() []table.Record {
	var out []table.Record
	for n, i := range m.failed() {
		c := m.checks[i]
		t := m.tabs[m.stepOf(c)]
		fix := table.Some(c.Fix)
		if c.Fix == "" {
			fix.Else = noFix
		}
		var note string
		if nt, ok := m.notes[c.Key()]; ok {
			note = nt.text
		}
		out = append(out, table.Record{Row: grid.Row{Key: c.Key()},
			Values: map[string]table.Value{"order": table.One(strconv.Itoa(n + 1)), "check": table.Some(c.Line),
				"fix": fix, "folder": table.One(folderValue(c)), "step": {Raw: []string{t.glyph + " " + t.name}, Link: checkLink + c.Key()},
				"note": table.Some(note)}})
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
	if m.checking.Running() {
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

// onRunButton says whether the screen cell x, y is on Run Selected Fixes, its count with it, as runButton draws it.
func (m Model) onRunButton(x, y int) bool {
	head := m.introHead()
	k := y - paneTop + m.offsets[m.step]
	if y < paneTop || k >= len(head) || m.runButton() == "" {
		return false
	}
	return onButton(ansi.Strip(head[k]), strings.TrimSpace(ansi.Strip(m.runButton())), x)
}

// placeChecks places the failed checks' table in the introduction, while it shows there, as tall as all its lines, so
// it never scrolls itself: under the introduction's head, scrolled as the introduction is, which may put its top above
// the pane. An introduction scrolled past its end, as when rows went, comes back to it.
func (m *Model) placeChecks() {
	i := indexOf(m.tabs, introductionID)
	if i < 0 || m.checksSheet == nil || m.tabs[i].sheet != m.checksSheet {
		return
	}
	s := m.checksSheet
	top := paneTop + len(m.introHead()) - m.offsets[i]
	s.Place(top, max(1, s.PaneHeight()), m.width, m.height)
	frozen := table.FrozenLines(s)
	header, _ := s.Header()
	need := table.RowsTop(s) - top + frozen + table.Gap(frozen, header) + len(header) + len(table.ScrollLines(s))
	s.Place(top, need, m.width, m.height)
	if last := max(0, len(m.introLines())-m.paneHeight()); m.offsets[i] > last {
		m.offsets[i] = last
		s.Place(paneTop+len(m.introHead())-last, need, m.width, m.height)
	}
}

// introLines gives every line of the introduction while its failed checks' table shows: its head, the table, all of
// it, the row under the cursor ending with its fix's pair, as fixLine places it, then a blank line, the folders not
// cloned, the folders the checks are on while they run again, and the rest of the page.
func (m Model) introLines() []string {
	out := m.introHead()
	c, at, buttons := m.fixLine()
	for k, l := range m.checksSheet.View() {
		if buttons && k == at {
			l = fix.After(l, m.width-1, runnable(c), m.copied == c.Key())
		}
		out = append(out, pane.Clip(l, m.width-1))
	}
	out = append(out, "")
	if gone := uncloned(m.workspace); len(gone) > 0 {
		out = append(out, lead+style.Off("not cloned: "+strings.Join(gone, ", ")), "")
	}
	if m.checking.Running() {
		m.addChecking(func(_ int, text string) { out = append(out, pane.Clip(text, m.width-1)) })
		out = append(out, "")
	}
	page := m.pageLines(indexOf(m.tabs, introductionID))
	for _, l := range page[min(2, len(page)):] {
		out = append(out, pane.Clip(l, m.width-1))
	}
	return out
}

// introPane gives the introduction's pane while its failed checks' table shows: its lines from where it is scrolled to,
// as many as the pane holds.
func (m Model) introPane() []string {
	lines := m.introLines()
	o := min(m.offsets[m.step], max(0, len(lines)-m.paneHeight()))
	return lines[o:min(len(lines), o+m.paneHeight())]
}

// scrollIntro scrolls the introduction n lines, never past its ends: its first line at the pane's top, or its last
// at the bottom.
func (m *Model) scrollIntro(n int) {
	last := max(0, len(m.introLines())-m.paneHeight())
	m.offsets[m.step] = min(max(0, m.offsets[m.step]+n), last)
	m.placeChecks()
}

// showFocus scrolls the introduction so the part of its table with the keyboard shows: its control line, or the lines
// of the row under its cursor, its first where they are more than the pane holds.
func (m *Model) showFocus() {
	s := m.sheet()
	head := len(m.introHead())
	first, last := head, head
	if s.InRows() {
		_, places := table.Laid(s)
		top, _ := s.Pane()
		lines, at, found := s.Lines(), table.RowsTop(s)-top, false
		for k, p := range places {
			if p >= 0 && lines[p].Row == s.Table().Cursor {
				if !found {
					first, found = head+at+k, true
				}
				last = head + at + k
			}
		}
	}
	o, h := &m.offsets[m.step], m.paneHeight()
	switch {
	case first < *o:
		*o = first
	case last >= *o+h:
		*o = min(first, last-h+1)
	}
	m.placeChecks()
}

// introScrollKey takes a key that scrolls the introduction while its table's rows have the keyboard, and says whether
// it took it: a page up or down, home and end to its ends, and down on the table's last row, on into the page's words.
// The cursor stays where it is, until a key moves it and the introduction follows it again.
func (m *Model) introScrollKey(key string) bool {
	s := m.sheet()
	if !m.inPane || !s.InRows() || s.Layered() {
		return false
	}
	switch h := m.paneHeight(); key {
	case "pgup":
		m.scrollIntro(-max(1, h-1))
	case "pgdown":
		m.scrollIntro(max(1, h-1))
	case "home":
		m.scrollIntro(-len(m.introLines()))
	case "end":
		m.scrollIntro(len(m.introLines()))
	case "down":
		if s.Table().Cursor < len(s.Rows())-1 {
			return false
		}
		m.scrollIntro(1)
	default:
		return false
	}
	return true
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

// fixLine gives the check under the table's cursor, while its rows have the keyboard, where it has a fix, and the line
// of the table's that ends with the fix's pair: its row's first.
func (m Model) fixLine() (c kit.Check, at int, ok bool) {
	if !m.introGridded() || !m.inPane {
		return kit.Check{}, 0, false
	}
	i := m.checkUnder()
	if i < 0 || m.checks[i].Fix == "" {
		return kit.Check{}, 0, false
	}
	s := m.checksSheet
	_, places := table.Laid(s)
	top, _ := s.Pane()
	lines, rows := s.Lines(), table.RowsTop(s)-top
	for k, p := range places {
		if p >= 0 && lines[p].Row == s.Table().Cursor {
			return m.checks[i], rows + k, true
		}
	}
	return kit.Check{}, 0, false
}

// fixClick takes a left click at the screen cell x, y on the pair the row under the table's cursor ends with, as it was
// drawn with copied as it was, and says whether it took it: ⧉ copies the row's fix, ▶️ runs it where kit runs it.
func (m *Model) fixClick(x, y int, copied string) (tea.Cmd, bool) {
	c, at, ok := m.fixLine()
	k := y - paneTop + m.offsets[m.step]
	if !ok || y < paneTop || y >= paneTop+m.paneHeight() || k != len(m.introHead())+at {
		return nil, false
	}
	m.copied = copied
	line := m.introLines()[k]
	m.copied = ""
	m.cursor = m.checkAt(c.Key())
	switch fix.On(line, x) {
	case fix.Copy:
		return m.copy(), true
	case fix.Run:
		if runnable(c) {
			return m.runFix(), true
		}
	}
	return nil, false
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
// row, c copies its fix, r runs it, where kit runs it, and enter goes to the check, on its step. It says whether it
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
	case fix.CopyKey:
		if c.Fix != "" {
			return m.copy(), true
		}
	case fix.RunKey:
		if runnable(c) {
			return m.runFix(), true
		}
	case "enter":
		m.follow()
		return nil, true
	}
	return nil, false
}

// introKeys gives the keys the introduction adds to its table's rows' while they have the keyboard: enter, c, and r
// on a check's row, as introKey says, and R while some row is selected.
func (m Model) introKeys(seg shell.Segment) shell.Segment {
	if i := m.checkUnder(); i >= 0 {
		seg.Bindings = append(seg.Bindings, shell.Bind("⏎", "go to the check", "enter"))
		if c := m.checks[i]; c.Fix != "" {
			seg.Bindings = append(seg.Bindings, fix.Bindings(runnable(c))...)
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
	selected, seen := map[string]bool{}, map[string]bool{}
	for _, key := range m.checksSheet.Selected() {
		selected[key] = true
	}
	m.queue = nil
	for _, i := range m.failed() {
		c := m.checks[i]
		if where := c.Folder + "\n" + c.Fix; selected[c.Key()] && runnable(c) && !seen[where] {
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
	id, key := c.ID, c.Key()
	return m.exec(m.over().WelcomeFix(c), func(err error) tea.Msg { return fixedMsg{id: id, key: key, err: err} })
}
