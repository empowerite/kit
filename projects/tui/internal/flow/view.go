// The picture drawn: the title, the stages' tab bar, and the selected stage's rows in a pane that scrolls, beside its
// panel and legend when ? shows them, and the keys under them all. Widths are terminal cells, so a two-cell glyph never pushes a row past
// the edge.
package flow

import (
	"slices"
	"strconv"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

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

// rule is what stands between the rows and the sidebar beside them, a space either side of it: a dim solid line from
// the line under the bar down to the keys bar; while the sidebar is hidden, a space and the rule end each of those
// lines, at the screen's last column. Beside the title and the bar there is none. On the rule's second line, at
// triangleAt, is a triangle, which a click turns as ? does: pointing left to show the sidebar, right to hide it. Under
// the triangle the rule starts again halfway down its cell, as under, so the space below the triangle looks as wide
// as the space above it.
const (
	rule       = "│"
	under      = "╷"
	show       = "◂"
	hide       = "▸"
	triangleAt = paneTop + 1
)

// line is one line of the pane: the row it belongs to, or -1 for a line about the stage, and its text.
type line struct {
	row  int
	text string
}

// View draws the title, the bar, the selected stage's pane, and a gap, and on the screen's right, from its top down to
// the keys bar on the last line, the panel's search, the rule under the bar, and the sidebar while ? shows it; on the
// alternate screen, with the mouse on for the bar. A screen too short for all of them keeps the bar, and cuts the rest from the bottom.
func (m Model) View() tea.View {
	lines := append([]string{" " + pane.Clip(m.title(), m.titleWidth()-1)}, strings.Split(m.bar().View(), "\n")...)
	lines = append(lines, m.pane()...)
	lines = m.side(m.overlay(append(lines, "")))
	bottom := style.Bar(m.keys(), m.width)
	if m.notice != "" {
		bottom = " " + style.Refusal(pane.Clip(m.notice, m.width-1))
	}
	lines = style.Bottom(lines, m.height, bottom)
	v := tea.NewView(strings.Join(lines, "\n"))
	v.AltScreen = true
	v.MouseMode = tea.MouseModeCellMotion
	return v
}

// bar gives the stages' tab bar, from the screen's second line, as wide as barWidth says: each stage's glyph, name,
// and count, the room for a count of n/m and the menu's glyph kept for each, and the keyboard on it unless the rows or
// the panel have it.
func (m Model) bar() tabs.Bar {
	b := tabs.Bar{Selected: m.stage, Focused: !m.inRows && !m.inPanel, Width: m.barWidth(), Row: 1,
		Reserve: len("xxx/yyy"), Menus: true, MenuOpen: m.menu}
	for i, s := range m.stages {
		b.Tabs = append(b.Tabs, tabs.Tab{Glyph: stageGlyphs[s.name], Name: s.name, Count: m.count(i)})
	}
	return b
}

// paneHeight gives the lines the pane has.
func (m Model) paneHeight() int { return max(1, m.height-fixed) }

// sideHeight gives the lines the rule and the sidebar have: every line above the keys bar.
func (m Model) sideHeight() int { return m.paneHeight() + fixed - 1 }

// sideWidth gives the cells the panel takes on the screen's right, shown or not: the widest stage's panel or legend,
// so moving along the bar never lays anything out again, and never more than two fifths of the screen, which clips it.
func (m Model) sideWidth() int { return min(m.widest, (m.width-3)*2/5) }

// sidebarWidth gives the cells the sidebar takes beside the rows, the rule before it apart: the panel's, while ? shows
// it, and nothing while it does not.
func (m Model) sidebarWidth() int {
	if !m.sidebar {
		return 0
	}
	return m.sideWidth()
}

// titleWidth gives the cells the title has: the screen's, less two spaces and the panel's search beside it, which is
// always there.
func (m Model) titleWidth() int { return m.width - 2 - m.sideWidth() }

// barWidth gives the cells the bar has: the screen's while the sidebar is hidden; beside it, the rows' and the cell
// of the rule, which starts under the bar, as the title's.
func (m Model) barWidth() int {
	if m.sidebar {
		return m.titleWidth()
	}
	return m.width
}

// paneWidth gives the cells the rows have: the screen's, less a space and the rule, and a space and the sidebar while
// ? shows it.
func (m Model) paneWidth() int {
	if w := m.sidebarWidth(); w > 0 {
		return m.width - 3 - w
	}
	return m.width - 2
}

// quitKey leads the keys bar wherever the keyboard is, as ctrl-c quits from anywhere: the one key that never moves.
const quitKey = "^C quit"

// keys gives the keys bar for where the keyboard is: ctrl-c first, then only the keys that act there: in the menu, in
// the panel, and in the search alone while the sidebar is hidden, their own; on the bar, left and right, down into the rows, and the numbers; in the rows, up and down
// and a page at a time; on a row that opens, right to open it or left to close it; tab into the panel, and = for the
// menu; r on a stage kittui loads; esc going back from the rows and from the sidebar. On a screen too narrow for them,
// the keys spared names are left out before the bar is cut.
func (m Model) keys() string {
	switch {
	case m.menu:
		return quitKey + " · " + menuKeys
	case m.inPanel && !m.sidebar:
		return quitKey + " · type to search · esc back"
	case m.inPanel:
		return quitKey + " · ↑↓ move · space toggle · type to search · esc back"
	}
	keys := []string{quitKey}
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
		keys = append(keys, "⇥ panel", "= menu")
	}
	if len(m.stages) > 0 && known[m.stages[m.stage].name] {
		keys = append(keys, "r refresh")
	}
	keys = append(keys, "? sidebar")
	if m.inRows || m.sidebar {
		keys = append(keys, "esc back")
	}
	return fitKeys(keys, m.width-1)
}

// spared names the keys a narrow keys bar leaves out, the first first, until the rest fit: the numbers, which the
// tabs show, a page at a time, and r.
var spared = []string{"1-9 jump", "pgup/pgdn page", "r refresh"}

// fitKeys joins the keys, leaving out what spared names, in its order, until they fit so many cells.
func fitKeys(keys []string, width int) string {
	line := strings.Join(keys, " · ")
	for _, key := range spared {
		if ansi.StringWidth(line) <= width {
			break
		}
		keys = slices.DeleteFunc(keys, func(k string) bool { return k == key })
		line = strings.Join(keys, " · ")
	}
	return line
}

// title gives the title line: kit flow, and origin's owner/repo after its glyph, in its own color, when known.
func (m Model) title() string {
	if m.origin.Repo == "" {
		return style.Label("kit flow")
	}
	return style.Label("kit flow") + " · " + Repository + " " + style.Repo(m.origin.Repo)
}

// count gives a stage's count for its tab: … while it loads, ! when its command failed, nothing for an incoming
// that did not fetch, else its rows but the asides, as incoming's headings; n/m, the rows the panel lets through of
// the rows loaded, when it holds some back.
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
	n, all := counted(m.rows(i)), m.shown(i).all
	if n != all {
		return strconv.Itoa(n) + "/" + strconv.Itoa(all)
	}
	return strconv.Itoa(n)
}

// counted gives how many of the rows are not asides.
func counted(rows []row) int {
	n := 0
	for _, r := range rows {
		if !r.Aside {
			n++
		}
	}
	return n
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

// sideLines gives the sidebar's lines while ? shows it: the selected stage's panel, then, after a gap, its legend.
func (m Model) sideLines() []string {
	panel, _ := m.panel(m.stage).Lines()
	if legend := legendLines(m.stages[m.stage].name); len(legend) > 0 {
		return append(append(panel, ""), legend...)
	}
	return panel
}

// pane gives the pane's lines, as many as it has, each as wide as the pane: the selected stage's from its offset on.
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
	return out
}

// side ends each of the lines, from the screen's top, with the lines on its right: beside the title, and the bar while
// the sidebar shows, two spaces and those of sideColumn; under the bar, a space and the rule, its triangle at triangleAt, and beside it,
// while ? shows it, a space and the rest of them.
func (m Model) side(lines []string) []string {
	side, sw, pw, bw := m.sideColumn(), m.sideWidth(), m.paneWidth(), m.barWidth()
	triangle := show
	if m.sidebar {
		triangle = hide
	}
	for k, l := range lines {
		// the spaces before a side line only when there is one, so a line with none ends on the bar, or the rule
		width, edge, gap := bw, "", "  "
		switch {
		case k == 0:
			width = m.titleWidth()
		case k >= paneTop:
			width, edge, gap = pw, " "+style.Off(rule), " "
			switch k {
			case triangleAt:
				edge = " " + triangle
			case triangleAt + 1:
				edge = " " + style.Off(under)
			}
		}
		if k < len(side) && side[k] != "" {
			edge += gap + pane.Clip(side[k], sw)
		}
		lines[k] = pane.Pad(pane.Clip(l, width), width, false) + edge
	}
	return lines
}

// sideColumn gives the lines on the screen's right, from its top: the selected stage's search, always; under it, while
// ? shows the sidebar, the rest of its panel and its legend from the sidebar's offset, a dim … N above and … N more
// where the screen hides some of them.
func (m Model) sideColumn() []string {
	if len(m.stages) == 0 || m.sideWidth() == 0 {
		return nil
	}
	all := m.sideLines()
	if !m.sidebar {
		return all[:1]
	}
	return append(all[:1:1], pane.Lines(all[1:], m.sideOffset, m.sideHeight()-1, "", "")...)
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
// panel lets no row through, or its rows as the grid draws them, the one under the cursor on a band while the rows have
// the keyboard.
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
