// The screen drawn: the title, the steps' tab bar, the selected step's pane, and the keys bar on the last line. Widths
// are terminal cells, so a two-cell glyph never pushes a line past the edge.
package welcome

import (
	"path/filepath"
	"strconv"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/empowerite/kit/projects/tui/internal/pane"
	"github.com/empowerite/kit/projects/tui/internal/shell"
	"github.com/empowerite/kit/projects/tui/internal/style"
	"github.com/empowerite/kit/projects/tui/internal/tabs"
)

// appLabel is the screen's name on the title line, after the app menu's =; a click on either drops the menu down.
const appLabel = "kit welcome"

// The glyphs the screen draws besides the tabs'.
const (
	Passed = "✅" // a check that passed
	Failed = "❌" // a check that failed, and a refusal
)

// fixed is how many lines the title, the bar, the gap, and the keys take; the pane has the rest, from paneTop on,
// right under the bar, which is open there to the selected tab.
const (
	fixed   = 3 + tabs.Height
	paneTop = 1 + tabs.Height
)

// lead is what starts a pane's line, and focus what starts the line under the cursor.
const (
	lead  = "   "
	focus = " " + style.Focus + " "
)

// under is what starts a line under a check's: as far in as the check's text.
const under = lead + "   "

// line is one line of the pane: the check it belongs to, or -1, and its text.
type line struct {
	row  int
	text string
}

// View draws the screen on the alternate screen, with the mouse on for the bar, the selected step's table's layers
// over its pane, and what the table did, or why not, in place of the keys while it says it.
func (m Model) View() tea.View {
	lines := append([]string{m.menuBar().Line(m.width)}, strings.Split(m.bar().View(), "\n")...)
	lines = append(lines, m.pane()...)
	if m.gridded() {
		lines = m.sheet().Over(lines)
	}
	lines = m.recent.Overlay(append(lines, ""), m.recallAt(), m.width)
	lines = m.appMenu.Overlay(lines, m.width)
	lines = m.books.Overlay(lines, m.booksAt(), m.width)
	bottom := m.keys().Line(m.width)
	if notice := m.sheetNotice(); notice != "" {
		bottom = notice
	}
	return shell.View(lines, m.height, bottom)
}

// bar gives the steps' tab bar, from the screen's second line: each step's glyph, name, and count, the count in red
// when a check failed or kit refused them, the keyboard on it unless the pane has it, and its view where it was left.
func (m Model) bar() tabs.Bar {
	b := tabs.Bar{Selected: m.step, Focused: !m.inPane, Scroll: m.barView, Width: m.width, Row: 1}
	for i, t := range m.tabs {
		count := m.count(i)
		b.Tabs = append(b.Tabs, tabs.Tab{Glyph: t.glyph, Name: t.name, Count: count})
		b.Alert = append(b.Alert, count != "" && count != "…")
	}
	return b
}

// paneHeight gives the lines the pane has.
func (m Model) paneHeight() int { return max(1, m.height-fixed) }

// menuBar gives the menu bar, as internal/shell draws every screen's: the kit menu's = before kit welcome; the
// workspace menu, which names the workspace; and the search field.
func (m Model) menuBar() shell.MenuBar {
	return shell.MenuBar{Screen: appLabel, Open: m.appMenu.Open, Workspace: m.books.Label(), Search: m.searchLine()}
}

// booksAt gives the column the menu bar's workspace menu starts at.
func (m Model) booksAt() int { return m.menuBar().WorkspaceAt() }

// keys gives the path to the keyboard, which the keys bar is drawn from: the screen; the steps' bar, whose down enters
// the step's checks, its table, which the table's name names, or its page; and the pane while it has the keyboard, the
// part of the step's table that has it, or the search, which takes every press while it has it. An open menu takes
// every press, and is the path alone, as a layer of the step's table is.
func (m Model) keys() shell.Path {
	if m.appMenu.Open || m.recent.Open || m.books.Open {
		return shell.Path{shell.MenuKeys()}
	}
	if m.gridded() {
		if seg, layer := m.sheet().Keys(); layer {
			return shell.Path{seg}
		}
	}
	down := "page"
	switch {
	case m.gridded():
		down = m.sheet().Name()
	case m.hasRows():
		down = "checks"
	}
	path := shell.Path{shell.WorkspaceScreenKeys(), m.bar().Keys(down)}
	switch {
	case m.search().Focused:
		search := shell.SearchKeys(down, m.search().Query, m.recalls())
		search.Owns = true
		return append(path, search)
	case m.inPane && m.gridded():
		seg, _ := m.sheet().Keys()
		return append(path, seg)
	case m.inPane:
		return append(path, m.paneKeys())
	}
	return path
}

// paneKeys gives the segment of the pane while it has the keyboard: esc gives the bar the keyboard, named by the step
// it has selected; up and down move between the rows, or scroll a page with none, a page at a time too, and home and
// end to the first and the last, unnamed until a test shows the keys arrive in Terminal.app and iTerm; enter on a link
// goes to its check; and on a check whose fix kit runs, tab and shift-tab choose now or later, n and l take one at
// once, and enter the one chosen; on one whose fix kit only shows, l takes later.
func (m Model) paneKeys() shell.Segment {
	move := "move"
	if !m.hasRows() {
		move = "scroll"
	}
	s := shell.Segment{Takes: []string{"home", "end"}, Bindings: []shell.Binding{shell.Bind("esc", "tabs"),
		shell.Bind("↑↓", move, "up", "down"), shell.Bind("⇞⇟", "page", "pgup", "pgdown")}}
	switch {
	case m.onLink():
		s.Bindings = append(s.Bindings, shell.Bind("⏎", "go to the check", "enter"))
	case m.choosing():
		s.Bindings = append(s.Bindings, shell.Bind("⇥·⇧⇥", "now or later", "tab", "shift+tab"), shell.Bind("n", "now"),
			shell.Bind("l", "later"), shell.Bind("⏎", "choose", "enter"))
	case m.keeping():
		s.Bindings = append(s.Bindings, shell.Bind("⇥·⇧⇥", "later or keep", "tab", "shift+tab"),
			shell.Bind("l", "later"), shell.Bind("k", "keep"), shell.Bind("⏎", "choose", "enter"))
	case m.showOnly():
		s.Bindings = append(s.Bindings, shell.Bind("l", "later"))
	}
	return s
}

// count gives the count of a step with checks: … while the checks first run, ! when kit refused them, else how many
// of its checks failed, or nothing when none did. The pages have none.
func (m Model) count(i int) string {
	switch {
	case !m.tabs[i].checked:
		return ""
	case m.checking.Loading:
		return "…"
	case m.refusal != "":
		return "!"
	}
	n := 0
	for _, k := range m.rows(i) {
		if !m.checks[k].OK {
			n++
		}
	}
	if n == 0 {
		return ""
	}
	return strconv.Itoa(n)
}

// pane gives the pane's lines, as many as it has: the selected step's from its offset on, or, on a step drawn as a
// table, its lead and its table.
func (m Model) pane() []string {
	h := m.paneHeight()
	if m.gridded() {
		// the table fills the pane under the lead
		return m.sheetPane()
	}
	var texts []string
	for _, l := range m.lines() {
		texts = append(texts, l.text)
	}
	out := pane.Lines(texts, m.offsets[m.step], h, "", lead)
	for k := range out {
		out[k] = pane.Clip(out[k], m.width-1)
	}
	for len(out) < h {
		out = append(out, "")
	}
	return out
}

// lines gives the selected step's lines: a step with checks has its page and its checks, the introduction its page and
// what the checks found, and any other step its page.
func (m Model) lines() []line {
	switch {
	case m.foldered(m.step):
		return m.setupLines()
	case m.checked():
		return m.stepLines()
	case m.isIntroduction(m.step):
		return m.introductionLines()
	}
	var out []line
	for _, l := range m.pageLines(m.step) {
		out = append(out, line{-1, l})
	}
	return out
}

// introductionLines gives the introduction's lines: its page's title and the blank line under it, then what the checks
// found, so it shows before the page's words however long they run, then a blank line and the rest of the page.
func (m Model) introductionLines() []line {
	page := m.pageLines(m.step)
	k := min(2, len(page))
	var out []line
	for _, l := range page[:k] {
		out = append(out, line{-1, l})
	}
	out = append(append(out, m.findings()...), line{-1, ""})
	for _, l := range page[k:] {
		out = append(out, line{-1, l})
	}
	return out
}

// findings gives what the checks found, as the introduction draws it. That is checking… until they first answer; then ✅
// when every check passed, or how many failed and a link to each, in the order of their steps: ❌, its line, and the
// glyph and name of its step, a folder's links under a dim line that names the folder, its repository, and its branch,
// the cursor's link drawn as chosen, and the search narrowing them as it does a step's checks. Kit's refusal, in red,
// comes first when kit refused them, and checking… last while they run again.
func (m Model) findings() []line {
	w := m.width - 1
	var out []line
	add := func(row int, text string) { out = append(out, line{row, pane.Clip(text, w)}) }
	if m.refusal != "" {
		for _, l := range wrapWords(strings.Fields(Failed+" "+m.refusal), w-len(lead), "", "") {
			add(-1, lead+style.Refusal(l))
		}
		if !m.checking.Loading {
			add(-1, "")
		}
	}
	if m.checking.Loading {
		if m.checking.Running() && m.refusal == "" {
			m.addChecking(add)
		}
		return out
	}
	failed := 0
	for _, c := range m.checks {
		if !c.OK {
			failed++
		}
	}
	if failed == 0 {
		add(-1, lead+Passed+" "+style.Label("all checks pass"))
	} else {
		head := strconv.Itoa(failed) + " checks need your attention"
		if failed == 1 {
			head = "1 check needs your attention"
		}
		add(-1, lead+style.Label(head))
		add(-1, lead+style.Off(strings.Repeat("─", ansi.StringWidth(head))))
		rows := m.rows(m.step)
		folder := ""
		for _, i := range rows {
			c := m.checks[i]
			t := m.tabs[m.stepOf(c)]
			// a folder's links stand under a line that names it, the first time one comes
			indent := ""
			if c.Folder != "" {
				indent = "  "
				if c.Folder != folder {
					add(-1, lead+style.Off(folderLine(c)))
				}
			}
			folder = c.Folder
			start, text := indent+lead, c.Line
			if m.inPane && i == m.cursor {
				start, text = indent+focus, style.Chosen(c.Line)
			}
			if m.deferred[c.Key()] {
				text += style.Off(" · later")
			}
			add(i, start+Failed+" "+text+" "+style.Tint("→ "+t.glyph+" "+t.name))
		}
		if len(rows) == 0 {
			add(-1, lead+style.Off("no check matches the search"))
		}
	}
	if gone := uncloned(m.workspace); len(gone) > 0 {
		add(-1, lead+style.Off("not cloned: "+strings.Join(gone, ", ")))
	}
	if m.checking.Running() {
		add(-1, "")
		m.addChecking(add)
	}
	return out
}

// pageLines gives a page's lines: the page drawn to the width, a dim line while the pages are read, or the dim lines
// saying where the page was looked for, the path broken across lines so all of it shows. The pane clips what a narrow
// screen cannot hold, since a page keeps ten cells for its words.
func (m Model) pageLines(i int) []string {
	switch {
	case m.tabs[i].missing != "":
		out := []string{lead + style.Off("no page at")}
		path := ansi.Hardwrap(m.tabs[i].missing, max(m.width-len(lead)-1, 1), false)
		for _, piece := range strings.Split(path, "\n") {
			out = append(out, lead+style.Off(piece))
		}
		return out
	case m.tabs[i].page == "":
		return []string{lead + style.Off("reading…")}
	}
	var out []string
	query := m.searches[m.tabs[i].id].Query
	for _, l := range render(m.tabs[i].page, m.width-len(lead)-1) {
		if matches(l, query) {
			out = append(out, strings.TrimRight(lead+l, " "))
		}
	}
	if len(out) == 0 {
		return []string{lead + style.Off("no line matches the search")}
	}
	return out
}

// stepLines gives the lines of a step with checks: its page, kit's refusal, its checks, each failed one's fix and
// where to read more under it, and the list of its checks left for later, each with its fix and, under it, what more
// says, as the toolchains check's other choices.
func (m Model) stepLines() []line {
	w := m.width - 1
	out := m.stepHead()
	add := func(row int, text string) { out = append(out, line{row, pane.Clip(text, w)}) }
	query := m.search().Query
	rows := m.rows(m.step)
	if len(rows) == 0 {
		switch {
		case m.checking.Running():
			m.addChecking(add)
		case query != "":
			add(-1, lead+style.Off("no check matches the search"))
		}
		return out
	}
	for _, i := range rows {
		out = append(out, m.checkLines(i, m.checks[i].Key(), "")...)
	}
	return append(out, m.stepTail(rows)...)
}

// stepHead gives the lines a step with checks starts with: its page, the setup step's without its heading, which the
// tab names, the lines the search lets through, and kit's refusal.
func (m Model) stepHead() []line {
	w := m.width - 1
	var out []line
	add := func(row int, text string) { out = append(out, line{row, pane.Clip(text, w)}) }
	query := m.search().Query
	if t := m.tabs[m.step]; t.missing == "" && t.page != "" {
		page := t.page
		if t.id == setupID {
			page = withoutHeading(page)
		}
		shown := 0
		for _, l := range render(page, w-len(lead)) {
			if matches(l, query) {
				add(-1, strings.TrimRight(lead+l, " "))
				shown++
			}
		}
		if shown > 0 {
			add(-1, "")
		}
	}
	if m.refusal != "" {
		for _, l := range wrapWords(strings.Fields(Failed+" "+m.refusal), w-len(lead), "", "") {
			add(-1, lead+style.Refusal(l))
		}
		add(-1, "")
	}
	return out
}

// withoutHeading gives a page without the heading it opens with, after its comments, if it opens with one.
func withoutHeading(page string) string {
	lines := strings.SplitAfter(page, "\n")
	for k, l := range lines {
		text := strings.TrimSpace(l)
		switch {
		case text == "" || strings.HasPrefix(text, "<!--"):
			continue
		case strings.HasPrefix(text, "# "):
			return strings.Join(lines[:k], "") + strings.Join(lines[k+1:], "")
		}
		break
	}
	return page
}

// addChecking adds the lines that say the checks run: checking…, then, dim under it, the check each folder is on, and
// the machine's, as kit says it starts each: <check> in <folder's name>.
func (m Model) addChecking(add func(row int, text string)) {
	add(-1, lead+style.Off("checking…"))
	for _, p := range m.running {
		text := p.Check
		if p.Folder != "" {
			text += " in " + filepath.Base(p.Folder)
		}
		add(-1, lead+"  "+style.Off(text))
	}
}

// stepTail gives the lines a step with checks ends with, after its rows: checking… while the checks run again, and the
// list of its checks left for later.
func (m Model) stepTail(rows []int) []line {
	w := m.width - 1
	var out []line
	add := func(row int, text string) { out = append(out, line{row, pane.Clip(text, w)}) }
	if m.checking.Running() {
		add(-1, "")
		m.addChecking(add)
	}
	var left []line
	for _, i := range rows {
		if c := m.checks[i]; m.deferred[c.Key()] && !c.OK {
			text := lead + "• " + c.Line + folderOf(c) + ": " + style.Mark(c.Fix)
			if !c.Runs && c.Fix != "" {
				text += style.Off(" · kit does not run it")
			}
			left = append(left, line{-1, pane.Clip(text, w)})
			if c.More != "" {
				left = append(left, line{-1, pane.Clip(lead+"  "+style.Mark(c.More), w)})
			}
		}
	}
	if len(left) > 0 {
		add(-1, "")
		add(-1, lead+style.Label("Left for later"))
		out = append(out, left...)
	}
	return out
}

// checkLines gives a check's lines, after the indent: ✅ or ❌ and its line, then, where no folder's row is drawn
// above it, its folder, the cursor's row drawn as chosen with now and later on it when it offers them; under a failed
// check, its fix, dim, then where to read more; and the note its fix left.
func (m Model) checkLines(i int, key, indent string) []line {
	c, w := m.checks[i], m.width-1
	focused := m.hasRows() && m.inPane && i == m.cursor
	start, text, glyph := indent+lead, c.Line, Passed
	if focused {
		start, text = indent+focus, style.Chosen(c.Line)
	}
	if !c.OK {
		glyph = Failed
	}
	row := start + glyph + " " + text
	if !m.foldered(m.step) {
		row += folderOf(c)
	}
	if m.deferred[key] && !c.OK {
		row += style.Off(" · later")
	}
	if focused && !c.OK && c.Fix != "" && c.Runs {
		buttons := "   " + button("now", m.choice == now) + " " + button("later", m.choice == later)
		row = pane.Clip(row, w-ansi.StringWidth(buttons)) + buttons
	}
	if focused && !c.OK && terminalFinding(c) {
		buttons := "   " + button("later", m.choice != keep) + " " + button("keep", m.choice == keep)
		row = pane.Clip(row, w-ansi.StringWidth(buttons)) + buttons
	}
	out := []line{{i, pane.Clip(row, w)}}
	if !c.OK && c.Fix != "" {
		fix := c.Fix
		if c.Interactive {
			fix += " · it needs you at the keyboard"
		}
		out = append(out, line{i, pane.Clip(indent+under+style.Off(fix), w)})
	}
	if !c.OK && c.More != "" {
		out = append(out, line{i, pane.Clip(indent+under+style.Off(c.More), w)})
	}
	if n, ok := m.notes[key]; ok {
		draw := style.Warn
		if n.refusal {
			draw = style.Refusal
		}
		out = append(out, line{i, pane.Clip(indent+under+draw(n.text), w)})
	}
	return out
}

// button draws a choice on a row: inverse when highlighted, dim when not, a space either side.
func button(name string, highlighted bool) string {
	if highlighted {
		return style.Cursor(" " + name + " ")
	}
	return style.Off(" " + name + " ")
}

// fit keeps the pane's offset within its lines, and in a step with rows scrolls so the row under the cursor shows,
// whole when it fits.
func (m *Model) fit() {
	if m.gridded() {
		// a step's table keeps its own window
		return
	}
	if !m.hasRows() {
		m.scroll(0)
		return
	}
	lines, h := m.lines(), m.paneHeight()
	o := &m.offsets[m.step]
	*o, _, _, _ = pane.Window(len(lines), h, *o)
	if !m.inPane {
		return
	}
	first, last := -1, -1
	for k, l := range lines {
		if l.row == m.cursor {
			if first < 0 {
				first = k
			}
			last = k
		}
	}
	if first < 0 {
		return
	}
	// a … line takes the place of a line at either edge, so the window is asked rather than counted
	for {
		start, end, _, _ := pane.Window(len(lines), h, *o)
		switch {
		case first < start:
			*o = first
		case last >= end && first > start:
			*o++
		default:
			return
		}
	}
}
