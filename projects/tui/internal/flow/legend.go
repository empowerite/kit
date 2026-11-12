// The legend: a box that floats over the screen, F1 or ? showing it and hiding it, which says what each glyph the
// screen shows under it means, the frame's, the controls', and the selected stage's rows', in the order each first
// appears, and marks those a click on leads somewhere. It stands in a corner of the rows,
// the bottom right as it opens; the corners' arrows on its top edge move it to another, its own dim, and so do home and
// end, to the top left and the bottom right, and page up and page down, to the corner over and under it. It takes no
// other key but F1 and ?, and no click but its arrows', so the rows under it go on as they were.
package flow

import (
	"cmp"
	"slices"
	"strings"

	"github.com/charmbracelet/x/ansi"

	"github.com/empowerite/kit/projects/tui/internal/grid"
	"github.com/empowerite/kit/projects/tui/internal/pane"
	"github.com/empowerite/kit/projects/tui/internal/shell"
	"github.com/empowerite/kit/projects/tui/internal/style"
	"github.com/empowerite/kit/projects/tui/internal/table"
	"github.com/empowerite/kit/projects/tui/internal/tabs"
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

// legendKey takes a key while the legend shows, and says whether it moved it: home to the top left and end to the
// bottom right, as their arrows do, and page up and page down to the corner over and under it, on its side.
func (m *Model) legendKey(k string) bool {
	if !m.legend.open {
		return false
	}
	right := m.legend.at == topRight || m.legend.at == bottomRight
	switch {
	case k == "home":
		m.legend.at = topLeft
	case k == "end":
		m.legend.at = bottomRight
	case k == "pgup" && right:
		m.legend.at = topRight
	case k == "pgup":
		m.legend.at = topLeft
	case k == "pgdown" && right:
		m.legend.at = bottomRight
	case k == "pgdown":
		m.legend.at = bottomLeft
	default:
		return false
	}
	return true
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
	for _, l := range m.allLines(m.stage) {
		out = append(out, ansi.Strip(l.Text))
	}
	return strings.Join(out, "\n")
}

// part is a glyph the legend explains where the screen shows it: the glyph, or those that mean the same, as the tabs'
// numbers; what it means; its mark, when a click on it leads somewhere; and the screen lines it means that on, as one
// glyph means one thing on the rule and another on the control line.
type part struct {
	glyphs        []string
	meaning, mark string
	on            func(y int) bool
}

// parts gives every glyph the legend can explain on a screen whose last line, the keys bar's, is the line last: the
// frame's and the controls', then the selected stage's rows' and its tab's, each on the lines it stands on.
func (m Model) parts(last int) []part {
	top := m.top()
	menuBar := func(y int) bool { return y == 0 }
	tabBar := func(y int) bool { return !m.maximized && y > 0 && y < top }
	control := func(y int) bool { return m.controls() && y == top }
	// the rule under the header, and the one under the heading over the groups, whose marks mean the same
	rule := func(y int) bool { return y >= top && y < last && (m.onRule(y) || m.controls() && y == top+1) }
	header := func(y int) bool { return y >= top && y < last && m.inHeader(y) }
	rows := func(y int) bool { return y >= top && y < last }
	headings := func(y int) bool { return rows(y) && !rule(y) && !control(y) }
	status := func(y int) bool { return y == last }
	var numbers, selected, places []string
	for i := range 10 {
		numbers, selected = append(numbers, tabs.Marker(i, false)), append(selected, tabs.Marker(i, true))
	}
	for _, d := range strings.Split("¹²³⁴⁵⁶⁷⁸⁹₁₂₃₄₅₆₇₈₉", "") {
		places = append(places, d)
	}
	one := func(glyph, meaning string, on func(int) bool) part { return part{[]string{glyph}, meaning, "", on} }
	out := []part{
		one(shell.App, "kit menu", menuBar),
		one(shell.Workspaces, "workspace menu", menuBar),
		one(shell.Back, "back: click or ⌫", menuBar),
		one("·", "separator", headings), // the menu bar's, between the kit and the workspace menus, is the bar's own
		one(grid.Search, "search: / or click", menuBar),
		one(shell.Recall, "past searches", menuBar),
		one(tabs.Menu, "tab menu", func(y int) bool { return tabBar(y) || m.maximized && menuBar(y) }),
		{selected, "selected tab", "", tabBar},
		{numbers, "tab: click to select", "", tabBar},
		{[]string{tabs.Before, tabs.After}, "more tabs: click to scroll", "", tabBar},
		{[]string{grid.Before, grid.After}, "more columns: click to scroll", "", header},
		one(tabs.Stale, "stale: click to reload", tabBar),
		one(grid.Unfolded, "open: click to close", rows),
		one(grid.Folded, "closed: click to open", rows),
		one("¬", "filtered out", control),
		one(Fields, "fields dialog", control),
		one(table.Presets, "presets menu", control),
		one(Sortable, "sortable: click to sort", rule),
		one("▲", "ascending: click to reverse", rule),
		one("▼", "descending: click to clear", rule),
		{places, "sort order", "", rule},
		one(table.FlattenGlyph, "list: click to flatten", rule),
		one(table.GatherGlyph, "flattened: click to gather", rule),
		one(grid.Chosen, "group selected", rows),
		one(grid.Partly, "group partly selected", rows),
		one(Unwatched, "watched", status),
	}
	for _, g := range []string{GitHub, GitLab} {
		out = append(out, one(g, "forge requests left", status))
	}
	name := m.stages[m.stage].name
	for _, s := range m.stages {
		if g := stageGlyphs[s.name]; g != "" {
			out = append(out, one(g, label(s.name)+" stage", tabBar))
		}
	}
	for _, e := range legends[name] {
		out = append(out, part{[]string{e.glyph}, e.meaning, marks[name][e.glyph], headings})
	}
	if name == "changes" {
		for _, e := range states {
			out = append(out, part{[]string{e.glyph}, e.meaning, ToPage, headings})
		}
	}
	return out
}

// legendLines gives the legend: every glyph the screen shows under it that the legend can explain, in the order each
// first appears, reading from the top left, each two cells wide, a space, what it means, and its mark when a click on
// it leads somewhere; or, where the screen shows none, a dim line that says so.
func (m Model) legendLines() []string {
	screen := m.frame()
	screen = append(screen, m.bottom())
	type found struct {
		line  string
		y, at int
	}
	var shown []found
	for _, p := range m.parts(len(screen) - 1) {
		first := found{y: -1}
		for y, l := range screen {
			if !p.on(y) {
				continue
			}
			plain := ansi.Strip(l)
			for _, g := range p.glyphs {
				if at := strings.Index(plain, g); at >= 0 && (first.y < 0 || y < first.y || y == first.y && at < first.at) {
					first = found{pane.Pad(g, 2, false) + " " + p.meaning + marked(p.mark), y, at}
				}
			}
			if first.y >= 0 {
				break
			}
		}
		if first.y >= 0 {
			shown = append(shown, first)
		}
	}
	slices.SortStableFunc(shown, func(a, b found) int { return cmp.Or(cmp.Compare(a.y, b.y), cmp.Compare(a.at, b.at)) })
	var out []string
	for _, f := range shown {
		out = append(out, f.line)
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

// columned lays a legend's lines out in as few columns as hold them in rows lines, side by side, two cells apart, each
// as wide as its widest line, so a legend taller than the room under the tab bar shows whole.
func columned(lines []string, rows int) []string {
	if rows < 1 || len(lines) <= rows {
		return lines
	}
	columns := (len(lines) + rows - 1) / rows
	rows = (len(lines) + columns - 1) / columns
	out := make([]string, rows)
	for c := range columns {
		part := lines[c*rows : min(len(lines), (c+1)*rows)]
		width := 0
		for _, l := range part {
			width = max(width, ansi.StringWidth(l))
		}
		for k := range rows {
			l := ""
			if k < len(part) {
				l = part[k]
			}
			if c < columns-1 {
				l = pane.Pad(l, width, false) + "  "
			}
			out[k] += l
		}
	}
	return out
}

// legendBox gives the legend's lines, a dialog titled by the stage, the corners' arrows at its top edge's right end,
// the one it stands in dim; and the screen line and column it is drawn from, in its corner of the rows, under the bar
// and over the keys bar.
func (m Model) legendBox() (lines []string, top, left int) {
	title := label(m.stages[m.stage].name) + " legend"
	// a blank line, dropped once drawn, wide enough that the title, a dash, and the arrows fit on the top edge
	room := strings.Repeat(" ", ansi.StringWidth(title)+len(arrows)*2+3)
	lines = shell.Dialog(title, append(columned(m.legendLines(), m.height-1-m.top()-2), room), m.width)
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
	top, left = m.top(), 0
	if m.legend.at == topRight || m.legend.at == bottomRight {
		left = max(0, m.width-width)
	}
	if m.legend.at == bottomLeft || m.legend.at == bottomRight {
		top = max(m.top(), m.height-1-len(lines))
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
