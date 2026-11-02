// The screen drawn: the title, the line of steps with each step's tab hanging from it, the selected step's pane, and
// the keys bar on the last line. Widths are terminal cells, so a two-cell glyph never pushes a line past the edge. The
// line and the tabs are one canvas: when it is wider than the screen, the view scrolls sideways to the selected step,
// as kit flow's line does.
package welcome

import (
	"strconv"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/empowerite/kit/tui/internal/style"
)

// The glyphs the screen draws besides the tabs'.
const (
	Repository = "📚" // the repository, owner/name, in the title
	Passed     = "✅" // a check that passed
	Failed     = "❌" // a check that failed, and a refusal
)

// The keys bar: on the line of a step with checks, on a check's row, on a row that offers now and later, on a row
// whose fix kit only shows, and on a page.
const (
	stepKeys   = "↑↓ move · ←→ step · n now · l later · ⏎ choose · esc quit"
	rowKeys    = "↑↓ move · ←→ step · n now · l later · ⏎ choose · esc back"
	choiceKeys = "↑↓ move · ←→ now or later · n now · l later · ⏎ choose · esc back"
	showKeys   = "↑↓ move · ←→ step · l later · esc back"
	pageKeys   = "↑↓ scroll · ←→ step · esc quit"
)

// fixed is how many lines the title, the line, the tabs, the gaps, and the keys take; the pane has the rest.
const fixed = 6

// gap is the cells before the first marker, and between one tab and the next marker.
const gap = 2

// room is the cells the view keeps before the selected marker when it scrolls: the dotted end and a little line.
const room = 5

// dotted is what a hidden end of the line is drawn with.
const dotted = "┄"

// The label forms, widest first: glyph, name, and count; glyph and count; the glyph alone.
const (
	fullLabel = iota
	countLabel
	glyphLabel
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

// layout is the line of steps laid out on its canvas: each tab's marker, each tab's label, the canvas's width, never
// less than the screen's, and the cells scrolled past on its left.
type layout struct {
	at     []int
	labels []string
	width  int
	scroll int
}

// View draws the screen on the alternate screen.
func (m Model) View() tea.View {
	l := m.layout()
	lines := []string{" " + clip(m.title(), m.width-1), m.line(l), m.labelLine(l), ""}
	lines = append(lines, m.pane()...)
	lines = append(lines, "")
	v := tea.NewView(strings.Join(style.Bottom(lines, m.height, style.Bar(m.keys(), m.width)), "\n"))
	v.AltScreen = true
	return v
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

// keys gives the keys bar for where the cursor is.
func (m Model) keys() string {
	switch {
	case !m.hasRows():
		return pageKeys
	case m.choosing():
		return choiceKeys
	case m.showOnly():
		return showKeys
	case m.inRows:
		return rowKeys
	}
	return stepKeys
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

// label gives a step's label in a form, hanging from its marker: its glyph, its name, bold in the accent color when
// selected, and its count, dim, or the accent's when the selected step's name is left out.
func (m Model) label(i, form int) string {
	t, chosen := m.tabs[i], i == m.step
	parts := []string{t.glyph}
	if form == fullLabel {
		name := t.name
		if chosen {
			name = style.Mark(name)
		}
		parts = append(parts, name)
	}
	if count := m.count(i); form != glyphLabel && count != "" {
		if chosen && form == countLabel {
			parts = append(parts, style.Mark(count))
		} else {
			parts = append(parts, style.Off(count))
		}
	}
	return "└ " + strings.Join(parts, " ")
}

// forms gives each step's label form, as kit flow gives its stages': the widest that lets every label fit the
// screen. The other steps give up their names first, then their counts; the selected step keeps its name as long as
// any layout with it fits, and only then shortens with the rest. The floor is the selected step's name among the
// other glyphs alone; past it the line scrolls. Unlike kit flow's, the floor keeps that name: every plain page
// shares one glyph, so the glyph alone would not say which page is open.
func (m Model) forms() []int {
	n := len(m.tabs)
	fits := func(chosen, others int) bool {
		total := gap
		for i := range n {
			form := others
			if i == m.step {
				form = chosen
			}
			total += ansi.StringWidth(m.label(i, form)) + gap
		}
		return total <= m.width-1
	}
	every := func(chosen, others int) []int {
		out := make([]int, n)
		for i := range out {
			out[i] = others
		}
		out[m.step] = chosen
		return out
	}
	for others := fullLabel; others <= glyphLabel; others++ {
		if fits(fullLabel, others) {
			return every(fullLabel, others)
		}
	}
	for form := countLabel; form < glyphLabel; form++ {
		if fits(form, form) {
			return every(form, form)
		}
	}
	return every(fullLabel, glyphLabel)
}

// layout lays the tabs out in their forms: when they fit the screen, the width left spreads evenly between them; when
// they do not, the canvas is wider than the screen, and the view scrolls so the selected tab starts a little room
// after the left edge, or as near it as the end of the canvas lets.
func (m Model) layout() layout {
	n, w := len(m.tabs), max(1, m.width-1)
	l := layout{at: make([]int, n), labels: make([]string, n)}
	total := gap
	for i, form := range m.forms() {
		l.labels[i] = m.label(i, form)
		total += ansi.StringWidth(l.labels[i]) + gap
	}
	l.width = max(total, w)
	spare, x := max(0, w-total), gap
	for i := range n {
		l.at[i] = x
		x += ansi.StringWidth(l.labels[i]) + gap + spare/n
		if i < spare%n {
			x++
		}
	}
	if l.width > w {
		l.scroll = min(max(l.at[m.step]-room, 0), l.width-w)
	}
	return l
}

// shown gives the cells of the canvas the screen shows, and how many cells each hidden end takes.
func (m Model) shown(l layout) (w, ends int) {
	w = min(max(1, m.width-1), l.width)
	return w, min(3, w/3)
}

// line draws the line of steps as the view shows it: a marker ○ per step, the dot ● in the accent color on the
// selected one, and a hidden end drawn dotted.
func (m Model) line(l layout) string {
	w, ends := m.shown(l)
	cells := make([]string, l.width)
	for k := range cells {
		cells[k] = "─"
	}
	for _, x := range l.at {
		cells[x] = "○"
	}
	cells[l.at[m.step]] = style.Mark("●")
	shown := cells[l.scroll : l.scroll+w]
	if l.scroll > 0 {
		for k := range ends {
			shown[k] = dotted
		}
	}
	if l.scroll+w < l.width {
		for k := range ends {
			shown[w-1-k] = dotted
		}
	}
	return " " + strings.Join(shown, "")
}

// labelLine draws each step's label under its marker as the view shows it, clipped at the right edge; a label whose
// marker is hidden, or under a dotted start, is hidden with it.
func (m Model) labelLine(l layout) string {
	w, edge := m.shown(l)
	if l.scroll == 0 {
		edge = 0
	}
	var b strings.Builder
	at := 0
	for i, label := range l.labels {
		x := l.at[i] - l.scroll
		if x < edge || x >= w {
			continue
		}
		if x+ansi.StringWidth(label) > w {
			label = ansi.Truncate(label, w-x, "")
		}
		b.WriteString(strings.Repeat(" ", max(0, x-at)))
		b.WriteString(label)
		at = x + ansi.StringWidth(label)
	}
	return strings.TrimRight(" "+b.String(), " ")
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
	out := windowed(texts, m.offsets[m.step], h)
	for len(out) < h {
		out = append(out, "")
	}
	return out
}

// pageLines gives a page's lines: the page drawn to the width, a dim line while the pages are read, or the dim line
// saying where the page was looked for.
func (m Model) pageLines(i int) []string {
	switch {
	case m.tabs[i].missing != "":
		return []string{lead + style.Off("no page at "+m.tabs[i].missing)}
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
	add := func(row int, text string) { out = append(out, line{row, clip(text, w)}) }
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
			left = append(left, line{-1, clip(text, w)})
			if c.More != "" {
				left = append(left, line{-1, clip(lead+"  "+style.Mark(c.More), w)})
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
	focused := m.hasRows() && m.inRows && i == m.cursor
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
		row = clip(row, w-ansi.StringWidth(buttons)) + buttons
	}
	out := []line{{i, clip(row, w)}}
	if !c.OK && c.Fix != "" {
		fix := c.Fix
		if c.Interactive {
			fix += " · it needs you at the keyboard"
		}
		out = append(out, line{i, clip(under+style.Off(fix), w)})
	}
	if !c.OK && c.More != "" {
		out = append(out, line{i, clip(under+style.Off(c.More), w)})
	}
	if n, ok := m.notes[id]; ok {
		draw := style.Warn
		if n.refusal {
			draw = style.Refusal
		}
		out = append(out, line{i, clip(under+draw(n.text), w)})
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
	*o, _, _, _ = window(len(lines), h, *o)
	if !m.inRows {
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
		start, end, _, _ := window(len(lines), h, *o)
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

// window says which of n lines a pane of h lines shows from offset o: the offset, clamped, the end, and whether a
// line says how many are hidden above, and how many below. Each such line takes the place of a shown one.
func window(n, h, o int) (start, end int, above, below bool) {
	if n <= h {
		return 0, n, false, false
	}
	start = min(max(o, 0), n-h+1)
	above = start > 0
	end = start + h
	if above {
		end--
	}
	if end < n {
		below = true
		end--
	}
	return start, min(end, n), above, below
}

// windowed gives the lines a pane of h lines shows from offset o, with a dim … N above and … N more where lines are
// hidden.
func windowed(all []string, o, h int) []string {
	start, end, above, below := window(len(all), h, o)
	out := make([]string, 0, h)
	if above {
		out = append(out, lead+style.Off("… "+strconv.Itoa(start)+" above"))
	}
	out = append(out, all[start:end]...)
	if below {
		out = append(out, lead+style.Off("… "+strconv.Itoa(len(all)-end)+" more"))
	}
	return out
}
