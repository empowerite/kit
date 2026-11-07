// The legend: a box that floats over the screen, F1 or ? showing it and hiding it, which says what each glyph the
// selected stage's rows draw now means, and marks those a click on leads somewhere. It stands in a corner of the rows,
// the bottom right as it opens; the corners' arrows on its top edge move it to another, its own dim. It takes no key
// but F1 and ?, and no click but its arrows', so the rows under it go on as they were.
package flow

import (
	"strings"

	"github.com/charmbracelet/x/ansi"

	"github.com/empowerite/kit/projects/tui/internal/pane"
	"github.com/empowerite/kit/projects/tui/internal/shell"
	"github.com/empowerite/kit/projects/tui/internal/style"
)

// corner is where the legend stands: one of the rows' corners.
type corner int

// The corners, in the order their arrows stand on the legend's top edge.
const (
	topLeft corner = iota
	topRight
	bottomLeft
	bottomRight
)

// arrows are the corners' arrows on the legend's top edge, in corner's order.
var arrows = []string{"↖", "↗", "↙", "↘"}

// legend is the legend while it shows, and the corner it stands in.
type legend struct {
	open bool
	at   corner
}

// toggleLegend shows the legend, or hides it, where it stands kept.
func (m *Model) toggleLegend() {
	if len(m.stages) > 0 {
		m.legend.open = !m.legend.open
	}
}

// drawn gives the text the selected stage's rows draw now, the lines the pane shows, to find the glyphs in.
func (m Model) drawn() string {
	var out []string
	for _, l := range m.lines(m.stage) {
		out = append(out, ansi.Strip(l.text))
	}
	return strings.Join(out, "\n")
}

// legendLines gives the selected stage's legend: each glyph its rows draw now, two cells wide, a space, what it
// means, and its mark when a click on it leads somewhere; changes' states the rows draw, after a gap under state,
// marked; or, where the rows draw none of them, a dim line that says so.
func (m Model) legendLines() []string {
	name, drawn := m.stages[m.stage].name, m.drawn()
	lines := func(entries []entry, mark map[string]string) []string {
		var out []string
		for _, e := range entries {
			if strings.Contains(drawn, e.glyph) {
				out = append(out, pane.Pad(e.glyph, 2, false)+" "+e.meaning+marked(mark[e.glyph]))
			}
		}
		return out
	}
	out := lines(legends[name], marks[name])
	if name == "changes" {
		if shown := lines(states, nil); len(shown) > 0 {
			if len(out) > 0 {
				out = append(out, "")
			}
			out = append(append(out, style.Off("state")+marked(ToPage)), shown...)
		}
	}
	if len(out) == 0 {
		return []string{style.Off("no glyph on the screen")}
	}
	return out
}

// marked gives a legend's mark after its meaning, dim after a space, or nothing for none.
func marked(mark string) string {
	if mark == "" {
		return ""
	}
	return " " + style.Off(mark)
}

// legendBox gives the legend's lines, a dialog titled by the stage, the corners' arrows at its top edge's right end,
// the one it stands in dim; and the screen line and column it is drawn from, in its corner of the rows, under the bar
// and over the keys bar.
func (m Model) legendBox() (lines []string, top, left int) {
	title := m.stages[m.stage].name + " legend"
	// a blank line, dropped once drawn, wide enough that the title, a dash, and the arrows fit on the top edge
	room := strings.Repeat(" ", ansi.StringWidth(title)+len(arrows)*2+3)
	lines = shell.Dialog(title, append(m.legendLines(), room), m.width)
	width := ansi.StringWidth(lines[0])
	// the arrows, a space before each, on the top edge, a cell short of its right corner
	var buttons []string
	for c, a := range arrows {
		if corner(c) == m.legend.at {
			a = style.Off(a)
		}
		buttons = append(buttons, a)
	}
	lines[0] = shell.Splice(lines[0], " "+strings.Join(buttons, " ")+" ", width-len(arrows)*2-2)
	lines = append(lines[:len(lines)-2], lines[len(lines)-1])
	top, left = paneTop, 0
	if m.legend.at == topRight || m.legend.at == bottomRight {
		left = max(0, m.width-width)
	}
	if m.legend.at == bottomLeft || m.legend.at == bottomRight {
		top = max(paneTop, m.height-1-len(lines))
	}
	return lines, top, left
}

// overLegend draws the legend over the screen's lines while it shows, under the menus and dialogs drawn after it.
func (m Model) overLegend(lines []string) []string {
	if !m.legend.open {
		return lines
	}
	box, top, left := m.legendBox()
	return shell.Over(lines, box, top, left)
}

// legendClick takes a click while the legend shows, and says whether it fell on it: on a corner's arrow, the legend
// moves to that corner; anywhere else on it, nothing. A click off it is the screen's.
func (m *Model) legendClick(x, y int) bool {
	box, top, left := m.legendBox()
	width := ansi.StringWidth(box[0])
	if x < left || x >= left+width || y < top || y >= top+len(box) {
		return false
	}
	if at := x - (left + width - len(arrows)*2 - 1); y == top && at >= 0 && at%2 == 0 && at/2 < len(arrows) {
		m.legend.at = corner(at / 2)
	}
	return true
}
