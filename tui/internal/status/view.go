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

// timeline draws the line and the labels under it: a marker per stage, evenly spaced across the width, the selected
// one ● and bold, the others ○, and each stage's name and count centered under its marker, clipped to its share of
// the width.
func (m Model) timeline() (string, string) {
	n, w := len(m.stages), m.width-1
	if n == 0 || w < 3*n {
		return "", ""
	}
	var line, labels strings.Builder
	line.WriteString(" ")
	labels.WriteString(" ")
	at, labelAt := 0, 0
	for i, s := range m.stages {
		center := (w * (2*i + 1)) / (2 * n)
		line.WriteString(strings.Repeat("─", center-at))
		at = center + 1
		marker := "○"
		if i == m.stage {
			marker = style.Label("●")
		}
		line.WriteString(marker)
		label := clip(strings.TrimSpace(s.name+" "+m.count(i)), w/n)
		width := ansi.StringWidth(label)
		start := min(max(center-width/2, labelAt), w-width)
		labels.WriteString(strings.Repeat(" ", max(0, start-labelAt)))
		if i == m.stage {
			label = style.Label(label)
		}
		labels.WriteString(label)
		labelAt = start + width
	}
	line.WriteString(strings.Repeat("─", w-at))
	return line.String(), labels.String()
}

// pane gives the pane's lines, as many as it has: the legend's from its offset on when it shows, else the selected
// stage's from its offset on.
func (m Model) pane() []string {
	h := m.paneHeight()
	var out []string
	switch {
	case m.legend:
		all := m.legendLines()
		start := min(max(m.legendOffset, 0), max(0, len(all)-h))
		out = all[start:min(len(all), start+h)]
	case len(m.stages) == 0:
		out = append(out, "   "+style.Off("no stage is configured"))
	default:
		lines := m.lines(m.stage)
		s := m.stages[m.stage]
		start := min(max(s.offset, 0), max(0, len(lines)-h))
		for i := start; i < len(lines) && i-start < h; i++ {
			out = append(out, lines[i].text)
		}
	}
	for len(out) < h {
		out = append(out, "")
	}
	return out
}

// legendLines draws the legend: each group's name, dim, in a column of its own, and beside it the group's glyphs, each
// with its meaning, joined by · and wrapped to the width; an entry wider than the room wraps on its own lines.
func (m Model) legendLines() []string {
	column := 0
	for _, e := range legend {
		column = max(column, ansi.StringWidth(e.group)+2)
	}
	room := max(m.width-3-column, 10)
	var out []string
	for i := 0; i < len(legend); {
		group := legend[i].group
		var wrapped []string
		current := ""
		for ; i < len(legend) && legend[i].group == group; i++ {
			e := legend[i].glyph + " " + legend[i].meaning
			switch {
			case current != "" && ansi.StringWidth(current)+3+ansi.StringWidth(e) <= room:
				current += " · " + e
			default:
				if current != "" {
					wrapped = append(wrapped, current)
				}
				parts := wrap(e, room)
				wrapped = append(wrapped, parts[:len(parts)-1]...)
				current = parts[len(parts)-1]
			}
		}
		wrapped = append(wrapped, current)
		for k, l := range wrapped {
			name := ""
			if k == 0 {
				name = group
			}
			out = append(out, "   "+style.Off(pad(name, column, false))+l)
		}
	}
	return out
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
