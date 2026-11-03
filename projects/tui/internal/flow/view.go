// The picture drawn: the title, the line with a marker and a tree per stage, the selected stage's rows in a pane that
// scrolls, and the keys. Widths are terminal cells, so a two-cell glyph never pushes a row past the edge. The line and
// the trees are one canvas: when it is wider than the screen, the view scrolls sideways to the selected stage.
package flow

import (
	"slices"
	"strconv"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/empowerite/kit/projects/tui/internal/pane"
	"github.com/empowerite/kit/projects/tui/internal/style"
)

// The keys bar, the screen's last line: on the line, where esc quits, and over the legend or on a row that does not
// open, where esc goes back. A row that opens adds ⏎, as open or close.
const (
	lineKeys = "↑↓ move · ←→ stage · r refresh · ? legend · esc quit"
	backKeys = "↑↓ move · ←→ stage · r refresh · ? legend · esc back"
)

// fixed is how many lines the title, the line, the gaps, and the keys take; the trees and the pane have the rest.
const fixed = 5

// minPane is the fewest lines the pane keeps when the trees are taller than the screen holds; the trees are cut.
const minPane = 3

// gap is the cells before the first marker, and between one tree and the next marker.
const gap = 2

// room is the cells the view keeps before the selected marker when it scrolls: the dotted end and a little line.
const room = 5

// dotted is what a hidden end of the line is drawn with.
const dotted = "┄"

// minFill is the fewest cells a fill cell keeps when the other columns leave it less; the row is then clipped.
const minFill = 12

// The label forms, widest first: glyph, name, and count; glyph and count; the glyph alone.
const (
	fullLabel = iota
	countLabel
	glyphLabel
)

// line is one line of the pane: the row it belongs to, or -1 for a line about the stage, and its text.
type line struct {
	row  int
	text string
}

// layout is the line of stages laid out on its canvas: each stage's marker, each stage's tree, and the canvas's
// width, never less than the screen's.
type layout struct {
	at    []int
	trees [][]string
	width int
}

// View draws the title, the line, the trees under it, the selected stage's pane, and the keys bar on the last line,
// on the alternate screen. A screen too short for all of them keeps the bar, and cuts the rest from the bottom.
func (m Model) View() tea.View {
	l := m.layout()
	rows := m.treeRows(l)
	lines := []string{" " + pane.Clip(m.title(), m.width-1), m.line(l)}
	lines = append(lines, m.treeLines(l, rows)...)
	lines = append(lines, "")
	lines = append(lines, m.pane(m.paneOf(rows))...)
	lines = append(lines, "")
	lines = style.Bottom(lines, m.height, style.Bar(m.keys(), m.width))
	v := tea.NewView(strings.Join(lines, "\n"))
	v.AltScreen = true
	return v
}

// paneHeight gives the lines the pane has.
func (m Model) paneHeight() int { return m.paneOf(m.treeRows(m.layout())) }

// paneOf gives the lines the pane has under trees of so many rows.
func (m Model) paneOf(rows int) int { return max(1, m.height-fixed-rows) }

// keys gives the keys bar for the state the screen is in, naming only the keys that act there: Enter on a row that
// opens, as open or close; r on a stage kittui loads; esc going back from the rows and from the legend.
func (m Model) keys() string {
	keys := []string{"↑↓ move", "←→ stage"}
	if len(m.stages) > 0 && known[m.stages[m.stage].name] {
		keys = append(keys, "r refresh")
	}
	if m.inRows {
		s, rows := m.stages[m.stage], m.rows(m.stage)
		if s.cursor < len(rows) && len(rows[s.cursor].detail) > 0 {
			if s.open[rows[s.cursor].key] {
				keys = append(keys, "⏎ close")
			} else {
				keys = append(keys, "⏎ open")
			}
		}
	}
	keys = append(keys, "? legend")
	if m.inRows || m.legend {
		return strings.Join(append(keys, "esc back"), " · ")
	}
	return strings.Join(append(keys, "esc quit"), " · ")
}

// title gives the title line: kit flow, and origin's owner/repo after its glyph, in its own color, when known.
func (m Model) title() string {
	if m.origin.Repo == "" {
		return style.Label("kit flow")
	}
	return style.Label("kit flow") + " · " + Repository + " " + style.Repo(m.origin.Repo)
}

// count gives a stage's count for its label: … while it loads, ! when its command failed, nothing for an incoming
// that did not fetch, else its rows but the asides, as incoming's headings.
func (m Model) count(i int) string {
	s := m.stages[i]
	switch {
	case s.loading:
		return "…"
	case s.refusal != "":
		return "!"
	case s.name == "incoming" && s.in != nil && !s.in.Fetched:
		return ""
	}
	n := 0
	for _, r := range m.rows(i) {
		if !r.aside {
			n++
		}
	}
	return strconv.Itoa(n)
}

// layout lays the stages out: each takes its tree's width and a gap, and when they all fit the screen, the width left
// spreads evenly between them; when they do not, the canvas is wider than the screen. When the trees with their legends
// fit the screen, each takes that width, legend on or off, so ? never moves a marker.
func (m Model) layout() layout {
	n, w := len(m.stages), m.width-1
	l := layout{at: make([]int, n), trees: make([][]string, n)}
	forms := m.forms()
	shown, legend := make([]int, n), make([]int, n)
	total, full := gap, gap
	for i := range m.stages {
		l.trees[i] = m.tree(i, forms[i], m.legend)
		shown[i], legend[i] = pane.Width(l.trees[i]), pane.Width(m.tree(i, forms[i], true))
		total += shown[i] + gap
		full += legend[i] + gap
	}
	widths := shown
	if full <= w {
		widths, total = legend, full
	}
	l.width = max(total, w, 1)
	if n == 0 {
		return l
	}
	spare, x := max(0, w-total), gap
	for i := range m.stages {
		l.at[i] = x
		x += widths[i] + gap + spare/n
		if i < spare%n {
			x++
		}
	}
	return l
}

// forms gives each stage's label form: the widest that lets every stage's tree, without its legend, fit the screen.
// The other stages give up their names first, then their counts; the selected stage keeps its name as long as any
// layout with it fits, and only then shortens with the rest. The glyph alone is the floor; past it the line scrolls.
func (m Model) forms() []int {
	n := len(m.stages)
	fits := func(chosen, others int) bool {
		total := gap
		for i := range m.stages {
			form := others
			if i == m.stage {
				form = chosen
			}
			total += ansi.StringWidth("└ "+m.label(i, form)) + gap
		}
		return total <= m.width-1
	}
	every := func(chosen, others int) []int {
		out := make([]int, n)
		for i := range out {
			out[i] = others
		}
		if m.stage < n {
			out[m.stage] = chosen
		}
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
	return every(glyphLabel, glyphLabel)
}

// label gives a stage's label in a form: its glyph, its name, bold in the accent color when selected, and its count,
// dim, or the accent's when the name is left out. A stage without a glyph keeps its name in every form.
func (m Model) label(i, form int) string {
	s := m.stages[i]
	glyph, count, name := stageGlyphs[s.name], m.count(i), s.name
	chosen := i == m.stage
	if chosen {
		name = style.Mark(name)
	}
	var parts []string
	if glyph != "" {
		parts = append(parts, glyph)
	}
	if form == fullLabel || glyph == "" {
		parts = append(parts, name)
	}
	if form != glyphLabel && count != "" {
		if chosen && form == countLabel {
			parts = append(parts, style.Mark(count))
		} else {
			parts = append(parts, style.Off(count))
		}
	}
	return strings.Join(parts, " ")
}

// node is one node of a tree: its text, and the nodes under it.
type node struct {
	text string
	kids []node
}

// draw adds a node's lines, and its kids', after the prefix that carries the line art of the nodes above it.
func (n node) draw(prefix string, last bool, out *[]string) {
	branch, next := "├ ", prefix+"│ "
	if last {
		branch, next = "└ ", prefix+"  "
	}
	*out = append(*out, prefix+branch+n.text)
	for k, kid := range n.kids {
		kid.draw(next, k == len(n.kids)-1, out)
	}
}

// tree gives the tree hanging from a stage's marker, its lines starting under it: the stage's label, and with the
// legend, legend over every glyph the stage draws, changes' states under state.
func (m Model) tree(i, form int, withLegend bool) []string {
	nodes := []node{{text: m.label(i, form)}}
	if withLegend {
		if legend := legendNode(m.stages[i].name); len(legend.kids) > 0 {
			nodes = append(nodes, legend)
		}
	}
	var out []string
	for k, n := range nodes {
		n.draw("", k == len(nodes)-1, &out)
	}
	return out
}

// legendNode gives a stage's legend as a node: each glyph, two cells wide, two spaces, and what it means.
func legendNode(name string) node {
	glyphs := func(entries []entry) []node {
		var out []node
		for _, e := range entries {
			out = append(out, node{text: pane.Pad(e.glyph, 2, false) + "  " + e.meaning})
		}
		return out
	}
	legend := node{text: style.Off("legend"), kids: glyphs(legends[name])}
	if name == "changes" {
		legend.kids = append(legend.kids, node{text: style.Off("state"), kids: glyphs(states)})
	}
	return legend
}

// treeRows gives the rows the trees take: the tallest tree's, cut so the pane keeps a few lines.
func (m Model) treeRows(l layout) int {
	tallest := 0
	for _, t := range l.trees {
		tallest = max(tallest, len(t))
	}
	return max(1, min(tallest, m.height-fixed-minPane))
}

// center gives a stage's marker on the canvas.
func (m Model) center(i int) int { return m.layout().at[i] }

// follow scrolls the view sideways so the selected stage's tree starts a little room after the left edge, or as near
// it as the end of the canvas lets; while the dot moves, so the dot does, and the view follows it. The trees before
// the selected one are then hidden, under the dotted start.
func (m *Model) follow() {
	l, w := m.layout(), m.width-1
	if len(m.stages) == 0 || l.width <= w {
		m.scroll = 0
		return
	}
	x := l.at[m.stage]
	if m.moving {
		x = m.dot.at
	}
	m.scroll = min(max(x-room, 0), l.width-w)
}

// line draws the line of stages as the view shows it: a marker ○ per stage, the dot ● in the accent color on the
// selected one, or on its way there, and a hidden end drawn dotted.
func (m Model) line(l layout) string {
	w := min(m.width-1, l.width)
	if len(m.stages) == 0 || w < 1 {
		return ""
	}
	cells := make([]string, l.width)
	for k := range cells {
		cells[k] = "─"
	}
	for _, x := range l.at {
		cells[x] = "○"
	}
	dot := l.at[m.stage]
	if m.moving {
		dot = min(max(m.dot.at, 0), l.width-1)
	}
	cells[dot] = style.Mark("●")
	shown := cells[m.scroll : m.scroll+w]
	ends := min(3, w/3)
	if m.scroll > 0 {
		for k := range ends {
			shown[k] = dotted
		}
	}
	if m.scroll+w < l.width {
		for k := range ends {
			shown[w-1-k] = dotted
		}
	}
	return " " + strings.Join(shown, "")
}

// treeLines draws the trees' rows as the view shows them, each tree under its marker and clipped at the right edge; a
// tree whose marker is hidden, or under a dotted start, is hidden with it. A tree taller than the rows ends in a dim
// … N more.
func (m Model) treeLines(l layout, rows int) []string {
	w := min(m.width-1, l.width)
	edge := 0
	if m.scroll > 0 {
		edge = min(3, w/3)
	}
	out := make([]string, rows)
	for r := range rows {
		var b strings.Builder
		at := 0
		for i, t := range l.trees {
			if r >= len(t) {
				continue
			}
			text := t[r]
			if r == rows-1 && len(t) > rows {
				text = style.Off("… " + strconv.Itoa(len(t)-rows+1) + " more")
			}
			x := l.at[i] - m.scroll
			if x < edge || x >= w {
				continue
			}
			if x+ansi.StringWidth(text) > w {
				text = ansi.Truncate(text, w-x, "")
			}
			b.WriteString(strings.Repeat(" ", max(0, x-at)))
			b.WriteString(text)
			at = x + ansi.StringWidth(text)
		}
		out[r] = strings.TrimRight(" "+b.String(), " ")
	}
	return out
}

// pane gives the pane's lines, as many as it has: the selected stage's from its offset on.
func (m Model) pane(h int) []string {
	var out []string
	switch {
	case len(m.stages) == 0:
		out = append(out, "   "+style.Off("no stage is configured"))
	default:
		lines := m.lines(m.stage)
		texts := make([]string, len(lines))
		for i, l := range lines {
			texts[i] = l.text
		}
		out = pane.Lines(texts, m.stages[m.stage].offset, h, m.head(lines, h), "   ")
	}
	for len(out) < h {
		out = append(out, "")
	}
	return out
}

// head gives the line the pane pins on top, in place of its … above: the first line of the row under the cursor, once
// paging through that open row has scrolled it away while the rest of the row still shows; else nothing.
func (m Model) head(lines []line, h int) string {
	if !m.inRows || len(m.stages) == 0 {
		return ""
	}
	s := m.stages[m.stage]
	start, _, above, _ := pane.Window(len(lines), h, s.offset)
	first := slices.IndexFunc(lines, func(l line) bool { return l.row == s.cursor })
	if !above || first < 0 || first >= start || lines[start].row != s.cursor {
		return ""
	}
	return lines[first].text
}

// lines gives a stage's lines: loading, kit's line when it failed, the dim line of an empty stage, or its rows, the
// one under the cursor led by the focus glyph, which turns down when the row is open, each open row's detail
// indented under it.
func (m Model) lines(i int) []line {
	s := m.stages[i]
	lead := "   "
	switch {
	case s.loading:
		return []line{{-1, lead + style.Off("loading")}}
	case s.refusal != "":
		var out []line
		for _, l := range pane.Wrap(Failed+" "+s.refusal, m.width-3) {
			out = append(out, line{-1, lead + style.Refusal(l)})
		}
		return out
	}
	rows := m.rows(i)
	if len(rows) == 0 {
		return []line{{-1, lead + style.Off(empty(s))}}
	}
	closed := make([][]cell, len(rows))
	for k, r := range rows {
		closed[k] = r.cells
	}
	chosen := -1
	if m.inRows && i == m.stage && s.cursor < len(rows) {
		chosen = s.cursor
	}
	drawn := table(closed, m.width-3, chosen)
	var out []line
	for k, r := range rows {
		l := lead
		switch {
		case k == chosen && s.open[r.key] && len(r.detail) > 0:
			l = " " + style.Opened + " "
		case k == chosen:
			l = " " + style.Focus + " "
		}
		out = append(out, line{k, l + drawn[k]})
		if s.open[r.key] {
			for _, d := range table(r.detail, m.width-6, -1) {
				out = append(out, line{k, lead + "   " + d})
			}
		}
	}
	return out
}

// table lays rows out in columns: each column as wide as its widest cell, a number right-aligned, a column with no
// text left out, and a row's fill cell clipped to the width left. A row of one cell spans the width. It gives one
// line per row, clipped to the width and without trailing spaces, the chosen row's first cell drawn as chosen.
func table(rows [][]cell, width, chosen int) []string {
	widths := columnWidths(rows)
	lines := make([]string, len(rows))
	for i, r := range rows {
		if len(r) == 1 {
			lines[i] = pane.Clip(r[0].text, width)
			continue
		}
		used, gaps, hasFill := 0, -1, false
		for j, c := range r {
			switch {
			case c.fill:
				hasFill = true
			case widths[j] > 0:
				used += widths[j]
				gaps++
			}
		}
		if hasFill {
			gaps++
		}
		left := max(width-used-gaps*2, minFill)
		var parts []string
		for j, c := range r {
			switch {
			case c.fill:
				parts = append(parts, pane.Pad(pane.Clip(c.text, left), left, false))
			case widths[j] == 0:
			default:
				text := c.text
				if i == chosen && len(parts) == 0 {
					text = style.Chosen(text)
				}
				parts = append(parts, pane.Pad(text, widths[j], c.right))
			}
		}
		lines[i] = pane.Clip(strings.TrimRight(strings.Join(parts, "  "), " "), width)
	}
	return lines
}

// columnWidths gives each column's width over the rows of two cells or more: its widest cell, a fill cell apart.
func columnWidths(rows [][]cell) []int {
	var widths []int
	for _, r := range rows {
		if len(r) < 2 {
			continue
		}
		for j, c := range r {
			for len(widths) <= j {
				widths = append(widths, 0)
			}
			if !c.fill {
				widths[j] = max(widths[j], ansi.StringWidth(c.text))
			}
		}
	}
	return widths
}
