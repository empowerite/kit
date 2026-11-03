// The picture drawn: the title, the stages' tab bar, the selected stage's rows in a pane that scrolls, beside its
// legend when ? shows it, and the keys. Widths are terminal cells, so a two-cell glyph never pushes a row past the
// edge.
package flow

import (
	"slices"
	"strconv"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/empowerite/kit/projects/tui/internal/pane"
	"github.com/empowerite/kit/projects/tui/internal/style"
	"github.com/empowerite/kit/projects/tui/internal/tabs"
)

// fixed is how many lines the title, the bar, the gap, and the keys take; the pane has the rest, from paneTop on,
// right under the bar, which is open there to the selected tab.
const (
	fixed   = 3 + tabs.Height
	paneTop = 1 + tabs.Height
)

// minFill is the fewest cells a fill cell keeps when the other columns leave it less; the row is then clipped.
const minFill = 12

// rule is what stands between the rows and the legend beside them, drawn dim; while the legend is hidden, its space
// and its line alone end every line of the pane, at the screen's last column. On the pane's first line the line is a
// triangle, which a click turns as ? does: pointing left to show the legend, right to hide it.
const (
	rule  = " ┊ "
	show  = "◂"
	hide  = "▸"
	ruled = 2 // the space and the line, before the legend's space
)

// line is one line of the pane: the row it belongs to, or -1 for a line about the stage, and its text.
type line struct {
	row  int
	text string
}

// View draws the title, the bar, the selected stage's pane, and the keys bar on the last line, on the alternate
// screen, with the mouse on for the bar. A screen too short for all of them keeps the bar, and cuts the rest from the
// bottom.
func (m Model) View() tea.View {
	lines := append([]string{" " + pane.Clip(m.title(), m.width-1)}, strings.Split(m.bar().View(), "\n")...)
	lines = append(lines, m.pane(m.paneHeight())...)
	lines = append(lines, "")
	lines = style.Bottom(lines, m.height, style.Bar(m.keys(), m.width))
	v := tea.NewView(strings.Join(lines, "\n"))
	v.AltScreen = true
	v.MouseMode = tea.MouseModeCellMotion
	return v
}

// bar gives the stages' tab bar, from the screen's second line: each stage's glyph, name, and count, the keyboard on it
// unless the rows have it, and the glyph alone the floor for the selected stage too.
func (m Model) bar() tabs.Bar {
	b := tabs.Bar{Selected: m.stage, Focused: !m.inRows, Width: m.width, Row: 1}
	for i, s := range m.stages {
		b.Tabs = append(b.Tabs, tabs.Tab{Glyph: stageGlyphs[s.name], Name: s.name, Count: m.count(i)})
	}
	return b
}

// paneHeight gives the lines the pane has.
func (m Model) paneHeight() int { return max(1, m.height-fixed) }

// legendWidth gives the cells the legend takes beside the rows, the rule before it apart: the widest stage's legend,
// so moving along the bar never lays the rows out again, and never more than two fifths of the screen, which clips
// it; nothing while ? does not show it.
func (m Model) legendWidth() int {
	if !m.legend {
		return 0
	}
	widest := 0
	for _, s := range m.stages {
		widest = max(widest, pane.Width(legendLines(s.name)))
	}
	return min(widest, (m.width-ansi.StringWidth(rule))*2/5)
}

// paneWidth gives the cells the rows have: the screen's, less the rule, and the legend while ? shows it.
func (m Model) paneWidth() int {
	if w := m.legendWidth(); w > 0 {
		return m.width - w - ansi.StringWidth(rule)
	}
	return m.width - ruled
}

// keys gives the keys bar for where the keyboard is, naming only the keys that act there: on the bar, left and right,
// down into the rows, and the numbers; in the rows, up and down and a page at a time; on a row that opens, right to
// open it or left to close it; r on a stage kittui
// loads; esc going back from the rows and from the legend, and ctrl-c quitting from the bar.
func (m Model) keys() string {
	var keys []string
	if m.inRows {
		keys = append(keys, "↑↓ move", "pgup/pgdn page")
		s, rows := m.stages[m.stage], m.rows(m.stage)
		if s.cursor < len(rows) && len(rows[s.cursor].detail) > 0 {
			if s.open[rows[s.cursor].key] {
				keys = append(keys, "← close")
			} else {
				keys = append(keys, "→ open")
			}
		}
	} else {
		keys = append(keys, "←→ tab", "↓ rows", "1-9 jump")
	}
	if len(m.stages) > 0 && known[m.stages[m.stage].name] {
		keys = append(keys, "r refresh")
	}
	keys = append(keys, "? legend")
	if m.inRows || m.legend {
		return strings.Join(append(keys, "esc back"), " · ")
	}
	return strings.Join(append(keys, "ctrl+c quit"), " · ")
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

// legendLines gives a stage's legend as the column beside its rows draws it: legend, dim, then each glyph the stage
// draws, two cells wide, two spaces, and what it means; changes' states after a gap under state, dim; nothing for a
// stage with no legend.
func legendLines(name string) []string {
	if len(legends[name]) == 0 {
		return nil
	}
	lines := func(entries []entry) []string {
		var out []string
		for _, e := range entries {
			out = append(out, pane.Pad(e.glyph, 2, false)+"  "+e.meaning)
		}
		return out
	}
	out := append([]string{style.Off("legend")}, lines(legends[name])...)
	if name == "changes" {
		out = append(append(out, "", style.Off("state")), lines(states)...)
	}
	return out
}

// pane gives the pane's lines, as many as it has: the selected stage's from its offset on, then the rule, its triangle
// on the first line, and beside it, while ? shows it, the stage's legend, a legend taller than the pane ending in a dim
// … N more.
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
	lw, triangle := m.legendWidth(), show
	var legend []string
	if lw > 0 && len(m.stages) > 0 {
		legend, triangle = legendLines(m.stages[m.stage].name), hide
	}
	if len(legend) > h {
		legend = append(legend[:h-1], style.Off("… "+strconv.Itoa(len(legend)-h+1)+" more"))
	}
	pw := m.paneWidth()
	for k := range out {
		line := style.Off(strings.TrimSpace(rule))
		if k == 0 {
			line = triangle
		}
		// the rule's last space only before a legend line, so a line with none ends on the rule
		side := " " + line
		if k < len(legend) {
			side += " " + pane.Clip(legend[k], lw)
		}
		out[k] = pane.Pad(pane.Clip(out[k], pw), pw, false) + side
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
		for _, l := range pane.Wrap(Failed+" "+s.refusal, m.paneWidth()-3) {
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
	drawn := table(closed, m.paneWidth()-3, chosen)
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
			for _, d := range table(r.detail, m.paneWidth()-6, -1) {
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
