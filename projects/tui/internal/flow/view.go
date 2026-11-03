// The picture drawn: the title, the stages' tab bar, the selected stage's rows in a pane that scrolls, beside its
// panel and legend when ? shows them, and the keys. Widths are terminal cells, so a two-cell glyph never pushes a row past the
// edge.
package flow

import (
	"slices"
	"strconv"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/empowerite/kit/projects/tui/internal/grid"
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

// rule is what stands between the rows and the side beside them, the panel and the legend, a space either side of it,
// drawn dim; while the side is hidden, a space and the rule end every line of the pane, at the screen's last column. On
// the pane's first line the rule is a triangle, which a click turns as ? does: pointing left to show the side, right
// to hide it.
const (
	rule = "┊"
	show = "◂"
	hide = "▸"
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
	lines = append(lines, m.pane()...)
	lines = append(lines, "")
	lines = style.Bottom(lines, m.height, style.Bar(m.keys(), m.width))
	v := tea.NewView(strings.Join(lines, "\n"))
	v.AltScreen = true
	v.MouseMode = tea.MouseModeCellMotion
	return v
}

// bar gives the stages' tab bar, from the screen's second line: each stage's glyph, name, and count, and the keyboard
// on it unless the rows or the panel have it.
func (m Model) bar() tabs.Bar {
	b := tabs.Bar{Selected: m.stage, Focused: !m.inRows && !m.inPanel, Width: m.width, Row: 1}
	for i, s := range m.stages {
		b.Tabs = append(b.Tabs, tabs.Tab{Glyph: stageGlyphs[s.name], Name: s.name, Count: m.count(i)})
	}
	return b
}

// paneHeight gives the lines the pane has.
func (m Model) paneHeight() int { return max(1, m.height-fixed) }

// legendWidth gives the cells the side takes beside the rows, the rule before it apart: the widest stage's panel or
// legend, so moving along the bar never lays the rows out again, and never more than two fifths of the screen, which
// clips it; nothing while ? does not show it.
func (m Model) legendWidth() int {
	if !m.legend {
		return 0
	}
	return min(m.widest, (m.width-3)*2/5)
}

// paneWidth gives the cells the rows have: the screen's, less a space and the rule, and a space and the side while
// ? shows it.
func (m Model) paneWidth() int {
	if w := m.legendWidth(); w > 0 {
		return m.width - 3 - w
	}
	return m.width - 2
}

// keys gives the keys bar for where the keyboard is, naming only the keys that act there: in the panel, its own; on
// the bar, left and right, down into the rows, and the numbers; in the rows, up and down and a page at a time; on a
// row that opens, right to open it or left to close it; tab into the panel; r on a stage kittui loads; esc going back
// from the rows and from the side, and ctrl-c quitting from the bar.
func (m Model) keys() string {
	if m.inPanel {
		return "↑↓ move · space toggle · type to search · esc back"
	}
	var keys []string
	if m.inRows {
		keys = append(keys, "↑↓ move", "pgup/pgdn page")
		s, rows := m.stages[m.stage], m.rows(m.stage)
		if s.cursor < len(rows) && grid.Opens(rows, s.cursor, m.paneWidth()) {
			if s.open[rows[s.cursor].Key] {
				keys = append(keys, "← close")
			} else {
				keys = append(keys, "→ open")
			}
		}
	} else {
		keys = append(keys, "←→ view", "↓ rows", "1-9 jump")
	}
	if len(m.stages) > 0 {
		keys = append(keys, "⇥ panel")
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

// count gives a stage's count for its tab: … while it loads, ! when its command failed, nothing for an incoming
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
		if !r.Aside {
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

// side gives the lines beside the selected stage's rows while ? shows them: its panel, then, after a gap, its legend.
func (m Model) side() []string {
	panel, _ := m.panel(m.stage).Lines()
	if legend := legendLines(m.stages[m.stage].name); len(legend) > 0 {
		return append(append(panel, ""), legend...)
	}
	return panel
}

// pane gives the pane's lines, as many as it has: the selected stage's from its offset on, then the rule, its triangle
// on the first line, and beside it, while ? shows it, the stage's panel and legend, a side taller than the pane ending
// in a dim … N more.
func (m Model) pane() []string {
	h := m.paneHeight()
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
	var side []string
	if lw > 0 {
		side, triangle = m.side(), hide
	}
	if len(side) > h {
		side = append(side[:h-1], style.Off(pane.More(len(side)-h+1)))
	}
	pw := m.paneWidth()
	for k := range out {
		line := style.Off(rule)
		if k == 0 {
			line = triangle
		}
		// the rule's last space only before a side line, so a line with none ends on the rule
		edge := " " + line
		if k < len(side) {
			edge += " " + pane.Clip(side[k], lw)
		}
		out[k] = pane.Pad(pane.Clip(out[k], pw), pw, false) + edge
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

// lines gives a stage's lines: loading, kit's line when it failed, the dim line of an empty stage, or of one whose
// panel lets no row through, or its rows as the grid draws them, the one under the cursor on a band while the rows have the keyboard.
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
	switch p := m.panel(i); {
	case len(rows) > 0:
	case p.Query != "" || p.Show != grid.All:
		return []line{{-1, lead + style.Off("no row matches the panel")}}
	default:
		return []line{{-1, lead + style.Off(empty(s))}}
	}
	chosen := -1
	if m.inRows && i == m.stage && s.cursor < len(rows) {
		chosen = s.cursor
	}
	drawn := grid.Draw(rows, m.paneWidth(), chosen, func(r row) bool { return s.open[r.Key] })
	out := make([]line, len(drawn))
	for k, l := range drawn {
		out[k] = line{l.Row, l.Text}
	}
	return out
}
