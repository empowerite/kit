// The picture drawn: the title, the line with a marker and a label per stage, the selected stage's rows in a pane
// that scrolls, and the keys. Widths are terminal cells, so a two-cell glyph never pushes a row past the edge.
package status

import (
	"strconv"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/empowerite/kit/tui/internal/style"
)

// The keys bar, the screen's last line, in each state: on the line, in the rows, and over the legend.
const (
	lineKeys   = "↑↓ move · ←→ stage · r refresh · ⏎ open · ? legend · esc quit"
	rowKeys    = "↑↓ move · ←→ stage · r refresh · ⏎ open · ? legend · esc back"
	legendKeys = "↑↓ move · ? legend · esc back"
)

// fixed is how many lines the title, the line, the gaps, and the keys take; the pane has the rest.
const fixed = 7

// minFill is the fewest cells a fill cell keeps when the other columns leave it less; the row is then clipped.
const minFill = 12

// line is one line of the pane: the row it belongs to, or -1 for a line about the stage, and its text.
type line struct {
	row  int
	text string
}

// View draws the title, the line with its labels, the selected stage's pane or the legend, and the keys bar on the
// last line, on the alternate screen.
func (m Model) View() tea.View {
	timeline, labels := m.timeline()
	lines := []string{" " + clip(m.title(), m.width-1), "", timeline, labels, ""}
	lines = append(lines, m.pane()...)
	lines = append(lines, "", style.Bar(m.keys(), m.width))
	v := tea.NewView(strings.Join(lines, "\n"))
	v.AltScreen = true
	return v
}

// paneHeight gives the lines the pane has.
func (m Model) paneHeight() int { return max(1, m.height-fixed) }

// keys gives the keys bar for the state the screen is in: the legend's, the rows', or the line's.
func (m Model) keys() string {
	switch {
	case m.legend:
		return legendKeys
	case m.inRows:
		return rowKeys
	}
	return lineKeys
}

// title gives the title line: kit status, and origin's owner/repo when known.
func (m Model) title() string {
	if m.origin.Repo == "" {
		return style.Label("kit status")
	}
	return style.Label("kit status") + " · " + m.origin.Repo
}

// count gives a stage's count for its label: … while it loads, ! when its command failed, nothing for an incoming
// that did not fetch, else its rows.
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
	return strconv.Itoa(len(m.rows(i)))
}

// timeline draws the line and the labels under it: a marker ○ per stage, evenly spaced across the width, the dot ● in
// the accent color on the selected one, or on its way there, and each stage's name and count centered under its
// marker, clipped to its share of the width.
func (m Model) timeline() (string, string) {
	n, w := len(m.stages), m.width-1
	if n == 0 || w < 3*n {
		return "", ""
	}
	var line, labels strings.Builder
	line.WriteString(" ")
	labels.WriteString(" ")
	labelAt := 0
	cells := make([]string, w)
	for k := range cells {
		cells[k] = "─"
	}
	for i := range m.stages {
		cells[m.center(i)] = "○"
	}
	dot := m.center(m.stage)
	if m.moving {
		dot = m.dot.at
	}
	cells[dot] = style.Mark("●")
	line.WriteString(strings.Join(cells, ""))
	for i := range m.stages {
		center := m.center(i)
		label, width := m.label(i, w/n)
		start := min(max(center-width/2, labelAt), w-width)
		labels.WriteString(strings.Repeat(" ", max(0, start-labelAt)))
		labels.WriteString(label)
		labelAt = start + width
	}
	return line.String(), labels.String()
}

// center gives the cell of a stage's marker on the line.
func (m Model) center(i int) int {
	n, w := len(m.stages), m.width-1
	return (w * (2*i + 1)) / (2 * n)
}

// label gives a stage's label under its marker, and its width in cells: its glyph, its name, bold when selected, and
// its count, dim. A label wider than its share is clipped, plain.
func (m Model) label(i, share int) (string, int) {
	s := m.stages[i]
	glyph, count := stageGlyphs[s.name], m.count(i)
	plain := strings.TrimSpace(strings.TrimSpace(glyph+" "+s.name) + " " + count)
	if ansi.StringWidth(plain) > share {
		plain = clip(plain, share)
		return plain, ansi.StringWidth(plain)
	}
	name := s.name
	if i == m.stage {
		name = style.Mark(name)
	}
	label := strings.TrimSpace(glyph + " " + name)
	if count != "" {
		label += " " + style.Off(count)
	}
	return label, ansi.StringWidth(plain)
}

// pane gives the pane's lines, as many as it has: the legend's from its offset on when it shows, else the selected
// stage's from its offset on.
func (m Model) pane() []string {
	h := m.paneHeight()
	var out []string
	switch {
	case m.legend:
		out = windowed(m.legendLines(), m.legendOffset, h)
	case len(m.stages) == 0:
		out = append(out, "   "+style.Off("no stage is configured"))
	default:
		lines := m.lines(m.stage)
		texts := make([]string, len(lines))
		for i, l := range lines {
			texts[i] = l.text
		}
		out = windowed(texts, m.stages[m.stage].offset, h)
	}
	for len(out) < h {
		out = append(out, "")
	}
	return out
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
		out = append(out, "   "+style.Off("… "+strconv.Itoa(start)+" above"))
	}
	out = append(out, all[start:end]...)
	if below {
		out = append(out, "   "+style.Off("… "+strconv.Itoa(len(all)-end)+" more"))
	}
	return out
}

// legendLines gives the legend: each group's name, dim, over its glyphs, one per line, every glyph in a column the eye
// can run down. When every stage's own glyphs fit the width the line of stages gives it, the glyphs every stage
// draws come first, and each stage's own sit under its name on the line; otherwise the groups flow into as many
// columns as the width holds, a group never split.
func (m Model) legendLines() []string {
	var names []string
	blocks := map[string][]string{}
	for _, e := range legend {
		if _, ok := blocks[e.group]; !ok {
			names = append(names, e.group)
			heading := e.group
			if g := stageGlyphs[strings.SplitN(e.group, ":", 2)[0]]; g != "" {
				heading = g + " " + e.group
			}
			blocks[e.group] = []string{style.Off(heading)}
		}
		blocks[e.group] = append(blocks[e.group], "  "+pad(e.glyph, 2, false)+"  "+e.meaning)
	}
	if out, ok := m.legendUnderStages(names, blocks); ok {
		return out
	}
	groups := make([][]string, 0, len(names))
	for _, name := range names {
		groups = append(groups, blocks[name])
	}
	return flow(groups, max(m.width-3, 10))
}

// legendUnderStages lays the legend out under the line of stages: the groups every stage draws, then a row of
// columns, each stage's groups starting where the stage's share of the line starts. It says no when a group belongs
// to no stage on the line, or a stage's glyphs are wider than its share.
func (m Model) legendUnderStages(names []string, blocks map[string][]string) ([]string, bool) {
	n, w := len(m.stages), m.width-1
	if n == 0 || w < 3*n {
		return nil, false
	}
	share := w / n
	columns := make([][]string, n)
	var shared []string
	for _, name := range names {
		if name == groupEvery {
			shared = blocks[name]
			continue
		}
		at := -1
		for i, s := range m.stages {
			if name == s.name || strings.HasPrefix(name, s.name+":") {
				at = i
			}
		}
		if at < 0 || width(blocks[name]) > share-1 {
			return nil, false
		}
		block := blocks[name]
		if name == m.stages[at].name {
			block = block[1:]
		} else {
			block = append([]string{style.Off(strings.TrimPrefix(name, m.stages[at].name+": "))}, block[1:]...)
		}
		if len(columns[at]) > 0 {
			columns[at] = append(columns[at], "")
		}
		columns[at] = append(columns[at], block...)
	}
	out := make([]string, 0, len(shared)+16)
	for _, l := range shared {
		out = append(out, "   "+l)
	}
	out = append(out, "")
	height := 0
	for _, c := range columns {
		height = max(height, len(c))
	}
	for row := 0; row < height; row++ {
		var b strings.Builder
		b.WriteString(" ")
		at := 1
		for i, c := range columns {
			if row >= len(c) {
				continue
			}
			center := (w * (2*i + 1)) / (2 * n)
			start := 1 + max(i*w/n, center-width(c)/2)
			b.WriteString(strings.Repeat(" ", max(0, start-at)))
			b.WriteString(c[row])
			at = start + ansi.StringWidth(c[row])
		}
		out = append(out, strings.TrimRight(b.String(), " "))
	}
	return out, true
}

// flow lays groups out in as many columns as the room holds, each as wide as the widest line, filled top to bottom
// and a group never split across two.
func flow(groups [][]string, room int) []string {
	widest := 0
	total := 0
	for _, g := range groups {
		widest = max(widest, width(g))
		total += len(g) + 1
	}
	cell := min(widest+4, room)
	columns := max(1, min(len(groups), room/cell))
	target := (total + columns - 1) / columns
	var cols [][]string
	var current []string
	for _, g := range groups {
		if len(current) > 0 && len(current)+len(g) > target && len(cols) < columns-1 {
			cols = append(cols, current)
			current = nil
		}
		if len(current) > 0 {
			current = append(current, "")
		}
		current = append(current, g...)
	}
	cols = append(cols, current)
	height := 0
	for _, c := range cols {
		height = max(height, len(c))
	}
	out := make([]string, 0, height)
	for row := 0; row < height; row++ {
		var b strings.Builder
		for n, c := range cols {
			text := ""
			if row < len(c) {
				text = clip(c[row], cell)
			}
			if n < len(cols)-1 {
				text = pad(text, cell, false)
			}
			b.WriteString(text)
		}
		out = append(out, "   "+strings.TrimRight(b.String(), " "))
	}
	return out
}

// width gives the widest of some lines, in cells.
func width(lines []string) int {
	widest := 0
	for _, l := range lines {
		widest = max(widest, ansi.StringWidth(l))
	}
	return widest
}

// lines gives a stage's lines: loading, kit's line when it failed, the dim line of an empty stage, or its rows, the
// one under the cursor led by the focus glyph, each open row's detail indented under it.
func (m Model) lines(i int) []line {
	s := m.stages[i]
	lead := "   "
	switch {
	case s.loading:
		return []line{{-1, lead + style.Off("loading")}}
	case s.refusal != "":
		var out []line
		for _, l := range wrap(Failed+" "+s.refusal, m.width-3) {
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
		if k == chosen {
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
			lines[i] = clip(r[0].text, width)
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
				parts = append(parts, pad(clip(c.text, left), left, false))
			case widths[j] == 0:
			default:
				text := c.text
				if i == chosen && len(parts) == 0 {
					text = style.Chosen(text)
				}
				parts = append(parts, pad(text, widths[j], c.right))
			}
		}
		lines[i] = clip(strings.TrimRight(strings.Join(parts, "  "), " "), width)
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

// pad fills a text to the width with spaces, before it when right is set.
func pad(text string, width int, right bool) string {
	n := width - ansi.StringWidth(text)
	if n <= 0 {
		return text
	}
	if right {
		return strings.Repeat(" ", n) + text
	}
	return text + strings.Repeat(" ", n)
}

// clip shortens a text to n cells, with … at the cut.
func clip(s string, n int) string {
	if n < 1 {
		return ""
	}
	return ansi.Truncate(s, n, "…")
}

// wrap breaks a text at spaces into lines no wider than the width, in cells; a word wider than the width stands alone.
func wrap(text string, width int) []string {
	width = max(width, 10)
	var lines []string
	current := ""
	for _, word := range strings.Fields(text) {
		switch {
		case current == "":
			current = word
		case ansi.StringWidth(current)+1+ansi.StringWidth(word) <= width:
			current += " " + word
		default:
			lines = append(lines, current)
			current = word
		}
	}
	if current != "" {
		lines = append(lines, current)
	}
	return lines
}
