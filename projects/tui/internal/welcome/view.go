// The screen drawn: the title, the steps' tab bar, the selected step's pane, and the keys bar on the last line. Widths
// are terminal cells, so a two-cell glyph never pushes a line past the edge.
package welcome

import (
	"strconv"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/empowerite/kit/projects/tui/internal/pane"
	"github.com/empowerite/kit/projects/tui/internal/style"
	"github.com/empowerite/kit/projects/tui/internal/tabs"
)

// The glyphs the screen draws besides the tabs'.
const (
	Repository = "📚" // the repository, owner/name, in the title
	Passed     = "✅" // a check that passed
	Failed     = "❌" // a check that failed, and a refusal
)

// The keys bar names only the keys that act where the keyboard is: on the bar over a step with checks, and over a
// page; on a check's row that offers no fix, on a row that offers now and later, on a row whose fix kit only shows;
// and in a page.
const (
	stepKeys   = "←→ view · ↓ checks · 1-9 jump · ctrl+c quit"
	pageKeys   = "←→ view · ↓ page · 1-9 jump · ctrl+c quit"
	rowKeys    = "↑↓ move · pgup/pgdn page · esc back"
	choiceKeys = "↑↓ move · pgup/pgdn page · ←→ now or later · n now · l later · ⏎ choose · esc back"
	showKeys   = "↑↓ move · pgup/pgdn page · l later · esc back"
	scrollKeys = "↑↓ scroll · pgup/pgdn page · esc back"
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

// View draws the screen on the alternate screen, with the mouse on for the bar.
func (m Model) View() tea.View {
	lines := append([]string{" " + pane.Clip(m.title(), m.width-1)}, strings.Split(m.bar().View(), "\n")...)
	lines = append(lines, m.pane()...)
	lines = append(lines, "")
	v := tea.NewView(strings.Join(style.Bottom(lines, m.height, style.Bar(m.keys(), m.width)), "\n"))
	v.AltScreen = true
	v.MouseMode = tea.MouseModeCellMotion
	return v
}

// bar gives the steps' tab bar, from the screen's second line: each step's glyph, name, and count, and the keyboard on
// it unless the pane has it.
func (m Model) bar() tabs.Bar {
	b := tabs.Bar{Selected: m.step, Focused: !m.inPane, Width: m.width, Row: 1}
	for i, t := range m.tabs {
		b.Tabs = append(b.Tabs, tabs.Tab{Glyph: t.glyph, Name: t.name, Count: m.count(i)})
	}
	return b
}

// paneHeight gives the lines the pane has.
func (m Model) paneHeight() int { return max(1, m.height-fixed) }

// title gives the title line: kit welcome, and origin's owner/repo after its glyph, in its own color, when known.
func (m Model) title() string {
	if m.repo == "" {
		return style.Label("kit welcome")
	}
	return style.Label("kit welcome") + " · " + Repository + " " + style.Repo(m.repo)
}

// keys gives the keys bar for where the keyboard is.
func (m Model) keys() string {
	switch {
	case !m.inPane && m.hasRows():
		return stepKeys
	case !m.inPane:
		return pageKeys
	case !m.hasRows():
		return scrollKeys
	case m.choosing():
		return choiceKeys
	case m.showOnly():
		return showKeys
	}
	return rowKeys
}

// count gives the count of a step with checks: … while the checks first run, ! when kit refused them, else how many
// of its checks failed, or nothing when none did. The pages have none.
func (m Model) count(i int) string {
	switch {
	case !m.tabs[i].checked:
		return ""
	case !m.loaded && m.checking:
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

// pane gives the pane's lines, as many as it has: the selected step's from its offset on.
func (m Model) pane() []string {
	h := m.paneHeight()
	var texts []string
	if m.hasRows() {
		for _, l := range m.stepLines() {
			texts = append(texts, l.text)
		}
	} else {
		texts = m.pageLines(m.step)
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
	for _, l := range render(m.tabs[i].page, m.width-len(lead)-1) {
		out = append(out, strings.TrimRight(lead+l, " "))
	}
	return out
}

// stepLines gives the lines of a step with checks: its page, kit's refusal, its checks, each failed one's fix and
// where to read more under it, and the list of its checks left for later, each with its fix and, under it, what more
// says, as the toolchains check's other choices.
func (m Model) stepLines() []line {
	w := m.width - 1
	var out []line
	add := func(row int, text string) { out = append(out, line{row, pane.Clip(text, w)}) }
	if t := m.tabs[m.step]; t.missing == "" && t.page != "" {
		for _, l := range render(t.page, w-len(lead)) {
			add(-1, strings.TrimRight(lead+l, " "))
		}
		add(-1, "")
	}
	if m.refusal != "" {
		for _, l := range wrapWords(strings.Fields(Failed+" "+m.refusal), w-len(lead), "", "") {
			add(-1, lead+style.Refusal(l))
		}
		add(-1, "")
	}
	rows := m.rows(m.step)
	if len(rows) == 0 {
		if m.checking {
			add(-1, lead+style.Off("checking…"))
		}
		return out
	}
	for _, i := range rows {
		out = append(out, m.checkLines(i, m.checks[i].ID)...)
	}
	if m.checking {
		add(-1, "")
		add(-1, lead+style.Off("checking…"))
	}
	var left []line
	for _, i := range rows {
		if c := m.checks[i]; m.deferred[c.ID] && !c.OK {
			text := lead + "• " + c.Line + ": " + style.Mark(c.Fix)
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

// checkLines gives a check's lines: ✅ or ❌ and its line, the cursor's row drawn as chosen with now and later on it
// when it offers them; under a failed check, its fix, dim, then where to read more; and the note its fix left.
func (m Model) checkLines(i int, id string) []line {
	c, w := m.checks[i], m.width-1
	focused := m.hasRows() && m.inPane && i == m.cursor
	start, text, glyph := lead, c.Line, Passed
	if focused {
		start, text = focus, style.Chosen(c.Line)
	}
	if !c.OK {
		glyph = Failed
	}
	row := start + glyph + " " + text
	if m.deferred[id] && !c.OK {
		row += style.Off(" · later")
	}
	if focused && !c.OK && c.Fix != "" && c.Runs {
		buttons := "   " + button("now", m.choice == now) + " " + button("later", m.choice == later)
		row = pane.Clip(row, w-ansi.StringWidth(buttons)) + buttons
	}
	out := []line{{i, pane.Clip(row, w)}}
	if !c.OK && c.Fix != "" {
		fix := c.Fix
		if c.Interactive {
			fix += " · it needs you at the keyboard"
		}
		out = append(out, line{i, pane.Clip(under+style.Off(fix), w)})
	}
	if !c.OK && c.More != "" {
		out = append(out, line{i, pane.Clip(under+style.Off(c.More), w)})
	}
	if n, ok := m.notes[id]; ok {
		draw := style.Warn
		if n.refusal {
			draw = style.Refusal
		}
		out = append(out, line{i, pane.Clip(under+draw(n.text), w)})
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

// fit keeps the pane's offset within its lines, and in a step with checks scrolls so the row under the cursor shows,
// whole when it fits.
func (m *Model) fit() {
	if !m.hasRows() {
		m.scroll(0)
		return
	}
	lines, h := m.stepLines(), m.paneHeight()
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
