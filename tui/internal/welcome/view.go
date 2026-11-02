// The screen drawn: the title, the line of steps with each step's tab hanging from it, the selected step's pane, and
// the keys bar on the last line. Widths are terminal cells, so a two-cell glyph never pushes a line past the edge.
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

// The keys bar: on the setup step line, on a check's row, on a row that offers now and later, and on a page.
const (
	setupKeys  = "↑↓ move · ←→ step · n now · l later · ⏎ choose · esc quit"
	rowKeys    = "↑↓ move · ←→ step · n now · l later · ⏎ choose · esc back"
	choiceKeys = "↑↓ move · ←→ now or later · n now · l later · ⏎ choose · esc back"
	pageKeys   = "↑↓ scroll · ←→ step · esc quit"
)

// fixed is how many lines the title, the line, the tabs, the gaps, and the keys take; the pane has the rest.
const fixed = 6

// gap is the cells before the first marker, and between one tab and the next marker.
const gap = 2

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

// layout is the line of steps laid out: each tab's marker, each tab's label, and the line's width.
type layout struct {
	at     []int
	labels []string
	width  int
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
	case m.step != setup:
		return pageKeys
	case m.choosing():
		return choiceKeys
	case m.inRows:
		return rowKeys
	}
	return setupKeys
}

// count gives Setup's count: … while the checks first run, ! when kit refused them, else how many failed, or nothing
// when none did. The pages have none.
func (m Model) count(i int) string {
	switch {
	case i != setup:
		return ""
	case !m.loaded && m.checking:
		return "…"
	case m.refusal != "":
		return "!"
	}
	n := 0
	for _, c := range m.checks {
		if !c.OK {
			n++
		}
	}
	if n == 0 {
		return ""
	}
	return strconv.Itoa(n)
}

// label gives a step's label: its glyph, its name, bold in the accent color when selected, and its count, dim. The
// short form is the glyph alone, with the count.
func (m Model) label(i int, short bool) string {
	t := tabs[i]
	parts := []string{t.glyph}
	if !short {
		name := t.name
		if i == m.step {
			name = style.Mark(name)
		}
		parts = append(parts, name)
	}
	if count := m.count(i); count != "" {
		parts = append(parts, style.Off(count))
	}
	return "└ " + strings.Join(parts, " ")
}

// layout lays the tabs out: every label in full when they fit, else the selected one in full and the rest short,
// else every one short. The width left spreads evenly between them.
func (m Model) layout() layout {
	w := max(1, m.width-1)
	l := layout{at: make([]int, steps), labels: make([]string, steps)}
	forms := []func(i int) bool{
		func(int) bool { return false },         // every label in full
		func(i int) bool { return i != m.step }, // the selected one in full
		func(int) bool { return true },          // every one short
	}
	total := 0
	for _, short := range forms {
		total = gap
		for i := range steps {
			l.labels[i] = m.label(i, short(i))
			total += ansi.StringWidth(l.labels[i]) + gap
		}
		if total <= w {
			break
		}
	}
	l.width = max(total, w)
	spare, x := max(0, w-total), gap
	for i := range steps {
		l.at[i] = x
		x += ansi.StringWidth(l.labels[i]) + gap + spare/steps
		if i < spare%steps {
			x++
		}
	}
	return l
}

// line draws the line of steps: a marker ○ per step and the dot ● in the accent color on the selected one, clipped
// to the screen.
func (m Model) line(l layout) string {
	cells := make([]string, l.width)
	for k := range cells {
		cells[k] = "─"
	}
	for _, x := range l.at {
		cells[x] = "○"
	}
	cells[l.at[m.step]] = style.Mark("●")
	return " " + strings.Join(cells[:min(l.width, max(0, m.width-1))], "")
}

// labelLine draws each step's label under its marker, clipped to the screen.
func (m Model) labelLine(l layout) string {
	var b strings.Builder
	for i, label := range l.labels {
		b.WriteString(strings.Repeat(" ", max(0, l.at[i]-ansi.StringWidth(b.String()))))
		b.WriteString(label)
	}
	return " " + clip(b.String(), m.width-1)
}

// pane gives the pane's lines, as many as it has: the selected step's from its offset on.
func (m Model) pane() []string {
	h := m.paneHeight()
	var texts []string
	if m.step == setup {
		for _, l := range m.setupLines() {
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
	case m.missing[i] != "":
		return []string{lead + style.Off("no page at "+m.missing[i])}
	case m.pages[i] == "":
		return []string{lead + style.Off("reading…")}
	}
	var out []string
	for _, l := range render(m.pages[i], m.width-len(lead)-1) {
		out = append(out, strings.TrimRight(lead+l, " "))
	}
	return out
}

// setupLines gives Setup's lines: its page, kit's refusal, the checks, each failed one's fix or where to read more
// under it, and the list of what is left for later.
func (m Model) setupLines() []line {
	w := m.width - 1
	var out []line
	add := func(row int, text string) { out = append(out, line{row, clip(text, w)}) }
	if m.missing[setup] == "" && m.pages[setup] != "" {
		for _, l := range render(m.pages[setup], w-len(lead)) {
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
	if len(m.checks) == 0 {
		if m.checking {
			add(-1, lead+style.Off("checking…"))
		}
		return out
	}
	for i, c := range m.checks {
		out = append(out, m.checkLines(i, c.ID)...)
	}
	if m.checking {
		add(-1, "")
		add(-1, lead+style.Off("checking…"))
	}
	var left []line
	for _, c := range m.checks {
		if m.deferred[c.ID] && !c.OK {
			left = append(left, line{-1, clip(lead+"• "+c.Line+": "+style.Mark(c.Fix), w)})
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
// when it offers them; under a failed check, its fix, dim, or where to read more; and the note its fix left.
func (m Model) checkLines(i int, id string) []line {
	c, w := m.checks[i], m.width-1
	focused := m.step == setup && m.inRows && i == m.cursor
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
	if focused && !c.OK && c.Fix != "" {
		buttons := "   " + button("now", m.choice == now) + " " + button("later", m.choice == later)
		row = clip(row, w-ansi.StringWidth(buttons)) + buttons
	}
	out := []line{{i, clip(row, w)}}
	switch {
	case c.OK:
	case c.Fix != "":
		fix := c.Fix
		if c.Interactive {
			fix += " · it needs you at the keyboard"
		}
		out = append(out, line{i, clip(under+style.Off(fix), w)})
	case c.More != "":
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

// fit keeps the pane's offset within its lines, and in Setup scrolls so the row under the cursor shows, whole when it
// fits.
func (m *Model) fit() {
	if m.step != setup {
		m.scroll(0)
		return
	}
	lines, h := m.setupLines(), m.paneHeight()
	o := &m.offsets[setup]
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
