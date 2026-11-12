// The screen drawn: the title, the steps' tab bar, the selected step's pane, and the keys bar on the last line. Widths
// are terminal cells, so a two-cell glyph never pushes a line past the edge.
package welcome

import (
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/empowerite/kit/projects/tui/internal/fix"
	"github.com/empowerite/kit/projects/tui/internal/kit"
	"github.com/empowerite/kit/projects/tui/internal/pane"
	"github.com/empowerite/kit/projects/tui/internal/shell"
	"github.com/empowerite/kit/projects/tui/internal/style"
	"github.com/empowerite/kit/projects/tui/internal/tabs"
)

// appLabel is the screen's name on the title line, after the app menu's =; a click on either drops the menu down.
const appLabel = "kit welcome"

// The glyphs the screen draws besides the tabs'.
const (
	Passed  = "✅" // a check that passed
	Offered = "💡" // a check that passed and offers a fix all the same
	Failed  = "❌" // a check that failed, and a refusal
)

// lead is what starts a pane's line, and focus what starts the line under the cursor.
const (
	lead  = "   "
	focus = " " + style.Focus + " "
)

// under is what starts a line under a check's: as far in as the check's text.
const under = lead + "   "

// line is one line of the pane: the check it belongs to, or -1, and its text; and the block it scrolls across with, as
// pane.Across keeps them, "" for a line that stays, its first lead cells staying as it does.
type line struct {
	row   int
	text  string
	block string
	lead  int
}

// blocksOf gives the blocks the lines stand in, each the lines that follow one another with its name.
func blocksOf(lines []line) []pane.Block {
	var out []pane.Block
	for k, l := range lines {
		if l.block == "" {
			continue
		}
		wide := ansi.StringWidth(l.text) - l.lead
		if n := len(out); n > 0 && out[n-1].ID == l.block && out[n-1].To == k {
			out[n-1].To, out[n-1].Wide = k+1, max(out[n-1].Wide, wide)
			continue
		}
		out = append(out, pane.Block{ID: l.block, From: k, To: k + 1, Lead: l.lead, Wide: wide})
	}
	return out
}

// texts gives the lines' texts, each block's whole.
func texts(lines []line) []string {
	out := make([]string, len(lines))
	for k, l := range lines {
		out[k] = l.text
	}
	return out
}

// slid gives the lines' texts, each block's scrolled across as far as it is.
func (m Model) slid(lines []line) []string {
	return m.across.Slid(texts(lines), blocksOf(lines), m.width-1)
}

// View draws the screen on the alternate screen, with the mouse on for the bar: the title, the bar, or, maximized, the
// banner, and the pane; the selected step's table's layers over its pane, the tab's menu while it is open, and what
// the table did, or why not, in place of the keys while it says it.
func (m Model) View() tea.View {
	lines := []string{m.menuBar().Line(m.width)}
	if m.maximized {
		lines = append(lines, m.banner())
	} else {
		lines = append(lines, strings.Split(m.bar().View(), "\n")...)
	}
	lines = append(lines, m.pane()...)
	if m.gridded() {
		lines = m.sheet().Over(lines)
	}
	if m.tabMenu.Open {
		box, left, _ := m.menuBoxIn()
		lines = shell.Over(lines, box, m.menuTop(), left)
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
// when a check failed or kit refused them, the keyboard on it unless the pane has it, its view where it was left, and
// the selected tab's :, which drops its menu down.
func (m Model) bar() tabs.Bar {
	b := tabs.Bar{Selected: m.step, Focused: !m.inPane, Scroll: m.barView, Width: m.width, Row: 1,
		MenuOpen: m.tabMenu.Open}
	for i, t := range m.tabs {
		count := m.count(i)
		b.Tabs = append(b.Tabs, tabs.Tab{Glyph: t.glyph, Name: t.name, Count: count})
		b.Alert = append(b.Alert, count != "" && count != "…")
	}
	return b
}

// paneHeight gives the lines the pane has: the screen's, from the pane's top, less the gap and the keys under it.
func (m Model) paneHeight() int { return max(1, m.height-m.paneTop()-2) }

// menuBar gives the menu bar, as internal/shell draws every screen's: the kit menu's = before kit welcome; the
// workspace menu, which names the workspace; maximized, the selected tab, as the bar draws it but its number in place
// of its glyph, its : last; and the search field.
func (m Model) menuBar() shell.MenuBar {
	b := shell.MenuBar{Screen: appLabel, Open: m.appMenu.Open, Workspace: m.books.Label(), Search: m.searchLine()}
	if m.maximized {
		b.On = m.bar().NumberedLabel()
	}
	return b
}

// booksAt gives the column the menu bar's workspace menu starts at.
func (m Model) booksAt() int { return m.menuBar().WorkspaceAt() }

// keys gives the path to the keyboard, which the keys bar is drawn from: the screen; the steps' bar, whose down enters
// the step's checks, its table, which the table's name names, or its page; and the pane while it has the keyboard, the
// part of the step's table that has it, or the search, which takes every press while it has it. An open menu takes
// every press, and is the path alone, as a layer of the step's table is.
func (m Model) keys() shell.Path {
	if m.appMenu.Open || m.recent.Open || m.books.Open || m.tabMenu.Open {
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
	bar := m.bar().Keys(down)
	if m.maximized {
		// the bar is not there to move along
		bar.Bindings = slices.DeleteFunc(bar.Bindings, func(b shell.Binding) bool { return b.Key == "⇥·⇧⇥" })
	}
	path := shell.Path{shell.WorkspaceScreenKeys(), bar}
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
// end to the first and the last, unnamed until a test shows the keys arrive in Terminal.app and iTerm; and on a
// check with a fix, c copies it and r runs it, where kit runs it; on a terminal's failed finding, k keeps it.
func (m Model) paneKeys() shell.Segment {
	move := "move"
	if !m.hasRows() {
		move = "scroll"
	}
	s := shell.Segment{Takes: []string{"home", "end"}, Bindings: []shell.Binding{shell.Bind("esc", "tabs"),
		shell.Bind("↑↓", move, "up", "down"), shell.Bind("⇞⇟", "page", "pgup", "pgdown")}}
	if m.copies() {
		s.Bindings = append(s.Bindings, fix.Bindings(m.runsHere())...)
	}
	if m.keeping() {
		s.Bindings = append(s.Bindings, shell.Bind("k", keepName))
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
	out := pane.Lines(m.slid(m.lines()), m.offsets[m.step], h, "", lead)
	for k := range out {
		out[k] = pane.Clip(out[k], m.width-1)
	}
	for len(out) < h {
		out = append(out, "")
	}
	return out
}

// lines gives the selected step's lines: a step with checks has its page and its checks, setup then the members of the
// workspace's group with no checkout, and any other step its page.
func (m Model) lines() []line {
	switch {
	case m.foldered(m.step):
		return append(m.setupLines(), m.notCloned()...)
	case m.hasRows():
		return append(m.stepLines(), m.notCloned()...)
	}
	return m.pageText(m.step)
}

// notCloned gives the lines setup ends with where the workspace's group has members with no checkout: a blank line,
// then their names, dim. Any other step has none.
func (m Model) notCloned() []line {
	gone := uncloned(m.workspace)
	if m.tabs[m.step].id != setupID || len(gone) == 0 {
		return nil
	}
	return []line{{row: -1}, {row: -1, text: pane.Clip(lead+style.Off("not cloned: "+strings.Join(gone, ", ")),
		m.width-1)}}
}

// pageLines gives a page's lines, as pageText does, their texts alone.
func (m Model) pageLines(i int) []string {
	var out []string
	for _, l := range m.pageText(i) {
		out = append(out, l.text)
	}
	return out
}

// pageText gives a page's lines: the page drawn to the width, what the search finds in it highlighted, each code
// block and table whole, a block of the step's, a dim line while the pages are read, or the dim lines saying where the
// page was looked for, the path broken across lines so all of it shows. The pane clips what a narrow screen cannot
// hold, since a page keeps ten cells for its words.
func (m Model) pageText(i int) []line {
	var out []line
	add := func(text string) { out = append(out, line{row: -1, text: text}) }
	switch {
	case m.tabs[i].missing != "":
		add(lead + style.Off("no page at"))
		path := ansi.Hardwrap(m.tabs[i].missing, max(m.width-len(lead)-1, 1), false)
		for _, piece := range strings.Split(path, "\n") {
			add(lead + style.Off(piece))
		}
		return out
	case m.tabs[i].page == "":
		add(lead + style.Off("reading…"))
		return out
	}
	query := m.searches[m.tabs[i].id].Query
	lines, blocks := wide(m.tabs[i].page, m.width-len(lead)-1)
	for _, l := range lines {
		add(strings.TrimRight(lead+shell.Found(l, query), " "))
	}
	for _, b := range blocks {
		for k := b.From; k < b.To; k++ {
			out[k].block, out[k].lead = m.tabs[i].id+"/"+b.ID, len(lead)
		}
	}
	return out
}

// stepLines gives the lines of a step with checks: its page, kit's refusal, its checks, and each failed one's fix and
// where to read more under it, as the toolchains check's other choices.
func (m Model) stepLines() []line {
	w := m.width - 1
	out := m.stepHead()
	add := func(row int, text string) { out = append(out, line{row: row, text: pane.Clip(text, w)}) }
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

// stepHead gives the lines a step with checks starts with: its page, what the search finds in it highlighted, and kit's
// refusal.
func (m Model) stepHead() []line {
	w := m.width - 1
	var out []line
	add := func(row int, text string) { out = append(out, line{row: row, text: pane.Clip(text, w)}) }
	if t := m.tabs[m.step]; t.missing == "" && t.page != "" {
		out = append(out, m.pageText(m.step)...)
		add(-1, "")
	}
	if m.refusal != "" {
		for _, l := range wrapWords(strings.Fields(Failed+" "+m.refusal), w-len(lead), "", "") {
			add(-1, lead+style.Refusal(l))
		}
		add(-1, "")
	}
	return out
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

// stepTail gives the lines a step with checks ends with, after its rows: checking… while the checks run again.
func (m Model) stepTail(rows []int) []line {
	w := m.width - 1
	var out []line
	add := func(row int, text string) { out = append(out, line{row: row, text: pane.Clip(text, w)}) }
	if m.checking.Running() {
		add(-1, "")
		m.addChecking(add)
	}
	return out
}

// checkLines gives a check's lines, after the indent: ✅, 💡 for an offer, or ❌, and its line, then, where no folder's
// row is drawn above it, its folder, the cursor's row drawn as chosen with its buttons, as rowButtons draws them; under
// it, each of its findings, a line each; under a failed check or an offer, its fix, dim; under a failed check, where to
// read more; and the note its fix left.
func (m Model) checkLines(i int, key, indent string) []line {
	c, w := m.checks[i], m.width-1
	focused := m.hasRows() && m.inPane && i == m.cursor
	start, text, glyph := indent+lead, c.Line, Passed
	if focused {
		start, text = indent+focus, style.Chosen(c.Line)
	}
	if !c.OK {
		glyph = Failed
	} else if c.Offer() {
		glyph = Offered
	}
	row := start + glyph + " " + text
	if !m.foldered(m.step) {
		row += folderOf(c)
	}
	if end := m.rowButtons(c, key); focused && end != "" {
		row = pane.Clip(row, w-ansi.StringWidth(end)) + end
	}
	out := []line{{row: i, text: pane.Clip(row, w)}}
	// the lines under the row, whole, are a block that scrolls across, the indent staying
	add := func(text string) {
		out = append(out, line{row: i, text: indent + under + text, block: "check/" + key, lead: len(indent + under)})
	}
	for _, f := range c.Findings {
		add(f)
	}
	if c.Fix != "" {
		// a folder's fix names its folder, so it runs wherever it is pasted
		shown := fix.In(c.Folder, c.Fix)
		if c.Interactive {
			shown += " · it needs you at the keyboard"
		}
		add(style.Off(shown))
	}
	if !c.OK && c.More != "" {
		add(style.Off(c.More))
	}
	if n, ok := m.notes[key]; ok {
		draw := style.Warn
		if n.refusal {
			draw = style.Refusal
		}
		add(draw(n.text))
	}
	return out
}

// rowButtons gives the buttons the cursor's row ends with: a check with a fix, its pair, as fix.Buttons draws it, ⧉, or
// copied once c copied it, and ▶️ where kit runs it, never on a terminal's finding; then keep, on a terminal's failed
// finding, which keeps it. None where the check offers neither.
func (m Model) rowButtons(c kit.Check, key string) string {
	var end string
	if c.Fix != "" {
		end = fix.Buttons(c.Runs && !terminalFinding(c), m.copied == key)
	}
	if !c.OK && terminalFinding(c) {
		if end == "" {
			end = " "
		}
		end += " " + style.Tint(" "+keepName+" ")
	}
	return end
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
