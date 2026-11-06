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
	"github.com/empowerite/kit/projects/tui/internal/shell"
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
// lines, at the screen's last column. Beside the title and the bar there is none. On the rule's middle line, at
// triangleAt, is a triangle, pointing left to show the sidebar and right to hide it, as a click anywhere on the rule
// does, as ? does. Under the triangle the rule starts again halfway down its cell, as under, so the space below the
// triangle looks as wide as the space above it.
const (
	rule  = "│"
	under = "╷"
	show  = "◂"
	hide  = "▸"
)

// triangleAt gives the screen line of the rule's triangle: the middle of the rule, from under the bar down to the keys
// bar.
func (m Model) triangleAt() int { return paneTop + (m.sideHeight()-paneTop)/2 }

// line is one line of the pane: the row it belongs to, or -1 for a line about the stage, and its text.
type line struct {
	row  int
	text string
}

// View draws the title, the bar, the selected stage's pane, and a gap, and on the screen's right, from its top down to
// the keys bar on the last line, the panel's search, the rule under the bar, and the sidebar while ? shows it; on the
// alternate screen, with the mouse on for the bar. A screen too short for all of them keeps the bar, and cuts the rest from the bottom.
func (m Model) View() tea.View {
	lines := append([]string{pane.Clip(m.title(), m.titleWidth())}, strings.Split(m.bar().View(), "\n")...)
	lines = append(lines, m.pane()...)
	lines = m.float(m.side(m.overlay(append(lines, ""))))
	bottom := m.keys().Line(m.width)
	switch {
	case m.notice != "" && m.noted:
		bottom = " " + pane.Clip(m.notice, m.width-1)
	case m.notice != "":
		bottom = " " + style.Refusal(pane.Clip(m.notice, m.width-1))
	}
	lines = style.Bottom(lines, m.height, bottom)
	v := tea.NewView(strings.Join(lines, "\n"))
	v.AltScreen = true
	v.MouseMode = tea.MouseModeCellMotion
	return v
}

// bar gives the stages' tab bar, from the screen's second line, as wide as barWidth says: each stage's glyph, name,
// and count, the room for the widest count of n/m its rows allow and the menu's glyph kept for each, the keyboard on it unless the rows or the
// panel have it, and the selected stage's group's color.
func (m Model) bar() tabs.Bar {
	b := tabs.Bar{Selected: m.stage, Focused: !m.inRows && !m.inPanel, Width: m.barWidth(), Row: 1,
		Menus: true, MenuOpen: m.menu, Joined: true}
	if len(m.stages) > 0 {
		b.Color = stageColors[m.stages[m.stage].name]
	}
	for i, s := range m.stages {
		b.Tabs = append(b.Tabs, tabs.Tab{Glyph: stageGlyphs[s.name], Name: s.name, Count: m.count(i)})
		b.Reserves = append(b.Reserves, m.reserve(i))
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

// titleWidth gives the cells the title has: all up to two spaces before the search, which is always there.
func (m Model) titleWidth() int { return max(0, m.searchAt()-2) }

// searchAt gives the screen column the search starts at, on the screen's first line: the sidebar's, while it shows;
// while it is hidden, where internal/shell puts every screen's, over the tab bar's numbers.
func (m Model) searchAt() int {
	if m.sidebar || len(m.stages) == 0 {
		return m.width - m.sideWidth()
	}
	p := m.panel(m.stage)
	return shell.SearchAt(m.width, m.bar().Numbers(), p.Focused && p.Cursor == 0 || p.Query != "")
}

// barWidth gives the cells the bar has: the screen's while the sidebar is hidden; beside it, the rows', a space, and
// the rule's cell, where the bar's last line turns down into the rule under it.
func (m Model) barWidth() int {
	if m.sidebar {
		return m.paneWidth() + 2
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

// keys gives the path to the keyboard, which the keys bar is drawn from: the screen, whose ? shows or hides the sidebar
// and tab gives it the keyboard; the stages' bar; and the part the keyboard is in, the rows or the sidebar and its
// control under the keyboard. An open menu or dialog takes every press, and is the path alone.
func (m Model) keys() shell.Path {
	switch {
	case m.ask != nil:
		_, keys := m.askKeys()
		return shell.Path{shell.DialogKeys(keys)}
	case m.viewMenu, m.recent.Open, m.menu, m.appMenu.Open:
		return shell.Path{shell.MenuKeys()}
	}
	if len(m.stages) == 0 {
		return shell.Path{shell.ScreenKeys(shell.Bind("?", m.sidebarDoes()))}
	}
	path := shell.Path{shell.ScreenKeys(shell.Bind("?", m.sidebarDoes()), shell.Bind("⇥", "sidebar", "tab")),
		m.bar().Keys("rows")}
	switch {
	case m.inPanel:
		return append(path, m.sidebarKeys(), m.controlKeys())
	case m.inRows:
		return append(path, m.rowKeys())
	}
	return path
}

// sidebarDoes says what ? does: shows the sidebar, or hides it.
func (m Model) sidebarDoes() string {
	if m.sidebar {
		return "hide sidebar"
	}
	return "show sidebar"
}

// sidebarKeys gives the segment of the sidebar while it has the keyboard: tab or esc gives the keyboard back to the
// rows or the bar, as they had it, named by the stage the bar has selected; up and down move between its controls, and
// a page at a time scrolls it, while it shows. It takes every other press, but the menus', the search's, and a stage's
// number.
func (m Model) sidebarKeys() shell.Segment {
	back := m.stages[m.stage].name
	if m.inRows {
		back = "rows"
	}
	s := shell.Segment{Owns: true, Passes: append([]string{"=", ":", ";", "/"}, shell.Digits...),
		Bindings: []shell.Binding{shell.Bind("⇥", back, "tab"), shell.Bind("esc", back)}}
	if m.sidebar {
		s.Bindings = append(s.Bindings, shell.Bind("↑↓", "move", "up", "down"))
		if n, h := m.scrolled(); n > h {
			s.Bindings = append(s.Bindings, shell.Bind("⇞⇟", "scroll", "pgup", "pgdown"))
		}
	}
	return s
}

// controlKeys gives the segment of the sidebar's control under the keyboard: the panel's own; on the views' dropdown,
// enter or space drops the views down, ctrl+s saves the panel as the view named, and what is typed renames it, = : and
// ; apart, which the menus take; esc stops a rename.
func (m Model) controlKeys() shell.Segment {
	p := m.panel(m.stage)
	if !p.OnView() {
		return p.Keys(len(m.recent.Items) > 0)
	}
	s := shell.Segment{Types: true, Takes: []string{"backspace"}, Passes: []string{"=", ":", ";"},
		Bindings: []shell.Binding{shell.Bind("⏎", "views", "enter", "space"), shell.Bind("⌃S", "save", "ctrl+s"),
			shell.Bind("", "type to rename")}}
	if m.stages[m.stage].renaming {
		s.Bindings = append(s.Bindings, shell.Bind("esc", "cancel"))
	}
	return s
}

// rowKeys gives the segment of the rows while they have the keyboard: esc gives the bar the keyboard, named by the
// stage it has selected; up and down move,
// a page at a time too while the rows are more than the pane shows; space selects a row, shift with up or down selects as it moves, and ctrl+a every row; right
// opens the row under the cursor, or unfolds a group's heading, and left closes or folds it, the one that acts named.
func (m Model) rowKeys() shell.Segment {
	seg := shell.Segment{Takes: []string{"left", "right"}, Bindings: []shell.Binding{
		shell.Bind("esc", m.stages[m.stage].name), shell.Bind("↑↓", "move", "up", "down")}}
	if len(m.lines(m.stage)) > m.rowsHeight() {
		seg.Bindings = append(seg.Bindings, shell.Bind("⇞⇟", "page", "pgup", "pgdown"))
	}
	seg.Bindings = append(seg.Bindings, shell.Bind("␣", "select", "space"),
		shell.Bind("⇧↑↓", "select", "shift+up", "shift+down"), shell.Bind("⌃A", "all", "ctrl+a"))
	s, rows := m.stages[m.stage], m.rows(m.stage)
	switch {
	case s.cursor >= len(rows):
	case grid.Heads(rows[s.cursor]) && s.view.Closed[rows[s.cursor].Key]:
		seg.Bindings = append(seg.Bindings, shell.Bind("→", "unfold", "right"))
	case grid.Heads(rows[s.cursor]):
		seg.Bindings = append(seg.Bindings, shell.Bind("←", "fold", "left"))
	case !grid.Opens(rows, m.names(m.stage), s.cursor, m.gridWidth(m.stage)):
	case s.open[rows[s.cursor].Key]:
		seg.Bindings = append(seg.Bindings, shell.Bind("←", "close", "left"))
	default:
		seg.Bindings = append(seg.Bindings, shell.Bind("→", "open", "right"))
	}
	return seg
}

// title gives the title line: as internal/shell draws it, the app's menu's glyph before kit flow; and origin's
// owner/repo after its glyph, in its own color, when known.
func (m Model) title() string {
	title := shell.Title(style.Label("kit flow"), m.appMenu.Open)
	if m.origin.Repo == "" {
		return title
	}
	return title + " · " + Repository + " " + style.Repo(m.origin.Repo)
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
	n, all := m.shown(i).passed, m.shown(i).all
	if n != all {
		return strconv.Itoa(n) + "/" + strconv.Itoa(all)
	}
	return strconv.Itoa(n)
}

// reserve gives the cells a stage's count is laid out for: as wide as it can grow while the stage's rows stay what
// they are, n/m with both as wide as the rows loaded, so a search or Show that narrows them moves no tab.
func (m Model) reserve(i int) int {
	if m.stages[i].loading || m.stages[i].refusal != "" {
		return 1
	}
	return 2*len(strconv.Itoa(m.shown(i).all)) + 1
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
// draws, two cells wide, a space, what it means, and its mark when a click on it leads somewhere; changes' states
// after a gap under state, dim and marked; nothing for a stage with no legend.
func legendLines(name string) []string {
	if len(legends[name]) == 0 {
		return nil
	}
	lines := func(entries []entry, mark map[string]string) []string {
		var out []string
		for _, e := range entries {
			out = append(out, pane.Pad(e.glyph, 2, false)+" "+e.meaning+marked(mark[e.glyph]))
		}
		return out
	}
	out := append([]string{style.Off("legend")}, lines(legends[name], marks[name])...)
	if name == "changes" {
		out = append(append(out, "", style.Off("state")+marked(ToPage)), lines(states, nil)...)
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

// sideLines gives the sidebar's lines while ? shows it: the selected stage's panel, then, after a gap, its legend.
func (m Model) sideLines() []string {
	panel, _ := m.panel(m.stage).Lines()
	if legend := legendLines(m.stages[m.stage].name); len(legend) > 0 {
		return append(append(panel, ""), legend...)
	}
	return panel
}

// names gives the labels of a stage's columns that its panel shows, in its order, as its header stands them over its
// rows, which widen its columns, each that sorts the rows marked as the panel marks it; none while its tab's menu hides
// its headers.
func (m Model) names(i int) []string {
	if !m.stages[i].headers {
		return nil
	}
	p, names := m.filter(i), m.shownColumns(i)
	for k, name := range names {
		names[k] = labelOf(m.stages[i].name, name)
		if mark := p.Mark(name); mark != "" {
			names[k] += " " + mark
		}
	}
	return names
}

// shownColumns gives the ids of a stage's columns its panel shows, in its order, one a cell of its rows, and none for a
// column without a label.
func (m Model) shownColumns(i int) []string {
	p := m.filter(i)
	var names []string
	for _, name := range p.Arrange(columnsOf(m.stages[i].name)) {
		if name == "" || p.Shows(name) {
			names = append(names, name)
		}
	}
	return names
}

// header gives the lines that stand over the selected stage's rows while its tab's menu shows its headers, and whether
// they hide lines while closed: the names of its columns the panel shows, as internal/grid lays them over the rows,
// open or closed as the stage's header is, then the rule under them, as rule draws it; nothing while it loads, after a refusal,
// or when it has no rows in columns.
func (m Model) header() (lines []string, hides bool) {
	if len(m.stages) == 0 {
		return nil, false
	}
	s := m.stages[m.stage]
	if !s.headers || s.loading || s.refusal != "" {
		return nil, false
	}
	lines, hides = grid.Header(m.rows(m.stage), m.names(m.stage), m.gridWidth(m.stage), s.headed)
	if len(lines) == 0 {
		return nil, false
	}
	for k := range lines {
		lines[k] = strings.Repeat(" ", m.indent(m.stage)) + lines[k]
	}
	return append(lines, m.rule()), hides
}

// rule gives the thin dim line under the header, across the pane after the gutter, with how many rows are selected,
// in the accent, near its right end, when some are.
func (m Model) rule() string {
	width := max(0, m.paneWidth()-3)
	n := m.selected()
	if n == 0 {
		return "   " + style.Off(strings.Repeat("─", width))
	}
	label := " " + strconv.Itoa(n) + " selected "
	left := max(0, width-ansi.StringWidth(label)-1)
	return "   " + style.Off(strings.Repeat("─", left)) + style.Tint(label) + style.Off("─")
}

// headerLines gives the header's lines alone.
func (m Model) headerLines() []string {
	lines, _ := m.header()
	return lines
}

// rowsTop gives the screen line the rows start on: the pane's first, or the one under the header.
func (m Model) rowsTop() int { return paneTop + len(m.headerLines()) }

// rowsHeight gives the lines the rows have, which scroll: the pane's, less the header's, which do not.
func (m Model) rowsHeight() int { return max(1, m.paneHeight()-len(m.headerLines())) }

// pane gives the pane's lines, as many as it has, each as wide as the pane: the header, when shown, then the selected
// stage's rows from its offset on.
func (m Model) pane() []string {
	h := m.rowsHeight()
	out := m.headerLines()
	switch {
	case len(m.stages) == 0:
		out = append(out, "   "+style.Off("no stage is configured"))
	default:
		lines := m.lines(m.stage)
		texts := make([]string, len(lines))
		for i, l := range lines {
			texts[i] = l.text
		}
		out = append(out, pane.Lines(texts, m.stages[m.stage].offset, h, m.head(lines, h), "   ")...)
	}
	for len(out) < m.paneHeight() {
		out = append(out, "")
	}
	return out
}

// side ends each of the lines, from the screen's top, with the lines on its right: beside the title, two spaces and
// the search, from searchAt; beside the bar while the sidebar shows, a space and the next of sideColumn's lines; under the bar, a space and the rule, its triangle at
// triangleAt, and beside it, while ? shows it, a space and the rest of them.
func (m Model) side(lines []string) []string {
	side, sw, pw, bw := m.sideColumn(), m.sideWidth(), m.paneWidth(), m.barWidth()
	triangle, at := show, m.triangleAt()
	if m.sidebar {
		triangle = hide
	}
	for k, l := range lines {
		// the spaces before a side line only when there is one, so a line with none ends on the bar, or the rule
		width, edge, gap, room := bw, "", " ", sw
		switch {
		case k == 0:
			width, gap, room = m.titleWidth(), "  ", m.width-m.searchAt()
			if !m.sidebar {
				room-- // the search alone ends one cell short of the edge
			}
		case k >= paneTop:
			width, edge, gap = pw, " "+style.Off(rule), " "
			switch k {
			case at:
				edge = " " + triangle
			case at + 1:
				edge = " " + style.Off(under)
			}
		}
		if k < len(side) && side[k] != "" {
			edge += gap + pane.Clip(side[k], room)
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
	marked, _ := style.In(stageColors[s.name])
	drawn := grid.Draw(rows, m.names(i), m.gridWidth(i), chosen, func(r row) bool { return s.open[r.Key] },
		func(r row) string {
			if s.picked[r.Key] {
				return marked(grid.Chosen)
			}
			return ""
		})
	out := make([]line, len(drawn))
	indent := strings.Repeat(" ", m.indent(i))
	for k, l := range drawn {
		if rows[l.Row].Aside {
			out[k] = line{l.Row, l.Text}
		} else {
			out[k] = line{l.Row, indent + l.Text}
		}
	}
	return out
}

// indent gives the cells a stage's rows, and its header, stand in by, under the headings of its groups: two a level,
// as internal/grid counts them.
func (m Model) indent(i int) int { return 2 * m.shown(i).levels }

// gridWidth gives the cells a stage's rows are laid out in: the pane's, less their indent.
func (m Model) gridWidth(i int) int { return m.paneWidth() - m.indent(i) }

// Noticing says whether the keys bar's line says what was done, or why not, in place of the keys, until a key: kittui
// keeps it on the screen while the keys bar is hidden.
func (m Model) Noticing() bool { return m.notice != "" }
