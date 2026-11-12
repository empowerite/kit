// The picture drawn: the title, the stages' tab bar, and the selected stage's rows in a pane that scrolls, beside its
// panel and legend when ? shows them, and the keys under them all. Widths are terminal cells, so a two-cell glyph never pushes a row past
// the edge.
package flow

import (
	"fmt"
	"maps"
	"slices"
	"strconv"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/empowerite/kit/projects/tui/internal/markdown"
	"github.com/empowerite/kit/projects/tui/internal/pane"
	"github.com/empowerite/kit/projects/tui/internal/shell"
	"github.com/empowerite/kit/projects/tui/internal/style"
	"github.com/empowerite/kit/projects/tui/internal/table"
	"github.com/empowerite/kit/projects/tui/internal/tabs"
)

// appLabel gives the screen's name on the title line, after the app menu's =: kit and its app's name, kit flow for
// Flow; a click on either drops the menu down.
func (m Model) appLabel() string { return "kit " + string(m.app) }

// top gives the screen line the pane starts on: under the bar, which is open there to the selected tab, its title
// line, and a blank; or, maximized, under the title and the banner, the line under it in the tab's color.
func (m Model) top() int {
	if m.maximized {
		return 2
	}
	return 1 + tabs.Height + titleHeight
}

// titleHeight is the lines the selected tab's title line takes under the bar: the line, and the blank under it.
const titleHeight = 2

// title gives the selected tab's title line, under the bar: its name bold, a dash, and what its catalog entry says it
// holds, as internal/markdown draws a page's title, clipped to the pane; its name alone for a stage the catalog does
// not describe.
func (m Model) title() string {
	if len(m.stages) == 0 {
		return ""
	}
	name := label(m.stages[m.stage].name)
	heading := "# " + strings.ToUpper(name[:1]) + name[1:]
	if c := catalog[m.stages[m.stage].name]; c != nil && c.Summary != "" {
		heading += " — " + c.Summary
	}
	return pane.Clip("  "+markdown.Render(heading, 1<<16)[0], m.paneWidth())
}

// banner gives the line under the title of a maximized tab: Banner, as wide as the pane, in the color the bar draws the
// selected tab in, the line every screen with no tab bar under its menu bar has, as shell.RuleLine draws it.
func (m Model) banner() string { return shell.RuleLine(m.paneWidth(), m.bar().Color) }

// View draws the title and, on its right, the search; the bar, or, maximized, the banner; the selected stage's pane, the
// rows the width of the screen; and a gap, over the keys bar on the last line; the legend over them while it shows; on
// the alternate screen, with the mouse on for the bar; and, at the keys bar's right end, watching while the selected
// stage is watched, then each forge's rate limit. A screen too short for all of them keeps the bar, and cuts the rest
// from the bottom.
func (m Model) View() tea.View {
	return shell.View(m.layers().Draw(m, m.overLegend(m.frame())), m.height, m.bottom())
}

// frame gives the screen's lines over its last, before the legend and the layers are drawn over them: the title, the
// tab bar and the selected tab's title line, or the banner of a maximized tab, and the pane.
func (m Model) frame() []string {
	lines := []string{m.menuBar().Line(m.width)}
	if m.maximized {
		lines = append(lines, m.banner())
	} else {
		lines = append(lines, strings.Split(m.bar().View(), "\n")...)
		lines = append(lines, m.title(), "")
	}
	return append(lines, m.pane()...)
}

// bottom gives the screen's last line: the keys bar, each forge's rate limit and whether the stage is watched at its
// right end; or a notice in its place.
func (m Model) bottom() string {
	right := quotaText(m.quota.Value, time.Now())
	if watched := m.watchedText(); watched != "" {
		right = strings.TrimSpace(watched + "  " + right)
	}
	bottom := m.keys().LineWith(m.width, right)
	switch {
	case m.note.Text != "" && m.note.Done:
		bottom = " " + pane.Clip(m.note.Text, m.width-1)
	case m.note.Text != "":
		bottom = " " + style.Refusal(pane.Clip(m.note.Text, m.width-1))
	}
	return bottom
}

// bar gives the stages' tab bar, from the screen's second line, as wide as barWidth says: each stage's glyph, name,
// and count, the room for its count and the menu's glyph kept for each, the keyboard on it unless the rows or the
// panel have it, its view where it was left, the selected stage's group's color, and a dot on each stage that is stale.
func (m Model) bar() tabs.Bar {
	b := tabs.Bar{Selected: m.stage, Focused: !m.inRows && !m.inPanel && !m.onPreset, Scroll: m.barView,
		Width: m.width, Row: 1, MenuOpen: m.tabMenu.Open}
	if len(m.stages) > 0 {
		b.Color = stageColors[m.stages[m.stage].name]
	}
	for i, s := range m.stages {
		b.Tabs = append(b.Tabs, tabs.Tab{Glyph: stageGlyphs[s.name], Name: label(s.name), Count: m.count(i)})
		b.Reserves = append(b.Reserves, m.reserve(i))
		b.Stale = append(b.Stale, s.src.Stale)
	}
	return b
}

// paneHeight gives the lines the pane has: the screen's from top on, but the keys bar under it, which follows the
// pane's last line at once.
func (m Model) paneHeight() int { return max(1, m.height-m.top()-1) }

// searchAt gives the screen column the search starts at, on the menu bar, where internal/shell puts every screen's.
func (m Model) searchAt() int { return m.menuBar().SearchAt(m.width) }

// searchLine gives the search as the menu bar draws it.
func (m Model) searchLine() string {
	p := m.panel(m.stage)
	return shell.RecallLine(p.Query, p.Focused, len(m.recent.Items) > 0)
}

// paneWidth gives the cells the rows have: the screen's, but its last cell.
func (m Model) paneWidth() int { return max(1, m.width-1) }

// keys gives the path to the keyboard, which the keys bar is drawn from: the screen, whose ? and F1 show or hide the
// legend, and whose ⌫ goes back while a crumb shows; the stages' bar, whose down reaches the preset line, or the rows
// where the stage has none, and which, maximized, has no tab or shift-tab; and the part the keyboard is in, the preset
// line, the rows, or the search. An open menu, dialog, or field's popover takes every press, and is the path alone.
func (m Model) keys() shell.Path {
	// an open layer, the topmost, names its keys alone, as layers.go orders them
	if path, ok := m.layers().Keys(m); ok {
		return path
	}
	legend := shell.Bind("?", "legend", "?", "f1")
	screen := shell.WorkspaceScreenKeys(shell.CrumbOf(m.appMenu.Crumb, legend)...)
	if len(m.stages) == 0 {
		return shell.Path{screen}
	}
	// down reaches the control line's first stop: the heading over the groups, or the preset picker
	down := "rows"
	switch {
	case len(m.stops()) > 0:
		down = "groups"
	case m.presets(m.stage):
		down = "preset"
	}
	bar := m.bar().Keys(down)
	if m.maximized {
		// the bar is not there to move along
		bar.Bindings = slices.DeleteFunc(bar.Bindings, func(b shell.Binding) bool { return b.Key == "⇥·⇧⇥" })
	}
	path := shell.Path{screen, bar}
	switch {
	case m.inPanel:
		return append(path, shell.SearchKeys("rows", m.panel(m.stage).Query, len(m.recent.Items) > 0))
	case m.onPreset:
		if stop, on := m.atStop(); on {
			return append(path, m.headingKeys(stop))
		}
		return append(path, m.presetKeys())
	case m.onRemedy():
		return append(path, m.remedyKeys())
	case m.inRows:
		return append(path, m.rowKeys())
	}
	return path
}

// presetKeys gives the segment of the preset line while it has the keyboard, as table.PresetKeys says.
func (m Model) presetKeys() shell.Segment { return table.PresetKeys(&m) }

// Selects says the picture acts on the rows selected: its tab menu runs its verbs over them.
func (m *Model) Selects() bool { return true }

// rowKeys gives the segment of the rows while they have the keyboard, as table.RowKeys says, ⌥ with an arrow named
// where the terminal answered the kitty keyboard protocol.
func (m Model) rowKeys() shell.Segment { return table.RowKeys(&m, m.altArrows) }

// menuBar gives the menu bar, as internal/shell draws every screen's: the kit menu's = before kit flow; the workspace
// menu, which names the workspace; maximized, the selected tab, as the bar draws it but its number in place of its
// glyph, its : last; the crumb, while a link led here; and the search field, with a stage to search.
func (m Model) menuBar() shell.MenuBar {
	b := shell.MenuBar{Screen: m.appLabel(), Open: m.appMenu.Open, Workspace: m.books.Label(), Crumb: m.appMenu.Crumb}
	if m.maximized {
		b.On = m.bar().NumberedLabel()
	}
	if len(m.stages) > 0 {
		b.Search = m.searchLine()
	}
	return b
}

// count gives a stage's count for its tab: … while it loads, ! when its command failed, else the rows it loaded but
// the asides, however few the panel lets through, and, for incoming, but its remotes' own rows, so a fetch that brought
// nothing counts none; after a +, where kit read those of more the forge holds, as the document's limited says.
func (m Model) count(i int) string {
	s := m.stages[i]
	switch {
	case s.src.Loading:
		return "…"
	case s.refusal != "":
		return "!"
	case s.name == "incoming":
		// from the records as decode read them, which the bar asks for on every draw: never built again here
		n := 0
		for _, p := range s.parsed {
			if kind := incomingKind.ReadFrom(p.value).Raw; len(kind) > 0 && kind[0] != "remote" {
				n++
			}
		}
		return strconv.Itoa(n)
	}
	return strconv.Itoa(m.shown(i).all) + m.capped(i)
}

// capped gives what a stage's count ends with: + where kit read only some of the rows the forge holds, as the
// document's limited says, so the tab counts those it drew and more; nothing else.
func (m Model) capped(i int) string {
	if m.stages[i].limited > 0 {
		return "+"
	}
	return ""
}

// incomingKind is the field incoming's count reads its rows' kinds from.
var incomingKind = table.Field{ID: "kind", Path: ".kind"}

// reserve gives the cells a stage's count is laid out for: the count's, its + among them, or one while it loads or
// after a refusal.
func (m Model) reserve(i int) int {
	if m.stages[i].src.Loading || m.stages[i].refusal != "" {
		return 1
	}
	return len(strconv.Itoa(m.shown(i).all) + m.capped(i))
}

// names gives the labels of a stage's columns that its panel shows, in its order, as table.HeaderNames gives them;
// none while the headers are hidden, as a test of the rows alone asks.
func (m Model) names(i int) []string {
	if m.noHeaders {
		return nil
	}
	return table.HeaderNames(m.filter(i), columnsOf(m.stages[i].name))
}

// shownColumns gives the ids of a stage's columns its panel shows, in its order, as table.ShownColumns gives them.
func (m Model) shownColumns(i int) []string {
	return table.ShownColumns(m.filter(i), columnsOf(m.stages[i].name))
}

// header gives the lines that stand over the selected stage's rows, as laid places them, and whether they hide lines
// while closed: the names of its columns the panel shows, as internal/grid lays them over the rows, open or closed as
// the stage's header is, then the rule under them, as rule draws it. While it loads, after a refusal, when its rows
// have no columns, or while no row shows, every one folded away under a heading, there are none.
func (m Model) header() (lines []string, hides bool) {
	if len(m.stages) == 0 || !slices.ContainsFunc(m.rows(m.stage), func(r row) bool { return !r.Aside }) {
		return nil, false
	}
	return m.headerLines()
}

// headerLines gives the header's lines and whether they hide lines while closed, as header says, whether a row shows or
// not: as table.Headed gives them, then the rule under them, as rule draws it. It is asked only on a stage, as header
// asks it.
func (m Model) headerLines() (lines []string, hides bool) {
	if s := m.stages[m.stage]; !m.noHeaders && !s.src.Loading && s.refusal == "" {
		lines, hides = table.Headed(m.rows(m.stage), m.names(m.stage), m.gridWidth(m.stage), s.Shift, m.indent(m.stage),
			s.Open[table.HeaderKey])
	}
	if len(lines) == 0 {
		return nil, false
	}
	return append(lines, m.rule()), hides
}

// rule gives the line under the header, as table.Rule draws it, each sort's mark on it as sortSpots places them.
func (m Model) rule() string { return table.Rule(&m, m.sortSpots()) }

// controls says whether the selected stage's pane starts with its control line: where its rows group, or it has a
// preset line.
func (m Model) controls() bool {
	return len(m.stages) > 0 && (m.presets(m.stage) || m.barText() != "")
}

// control gives the control line, the pane's first, as table.Control draws it, and the screen column the preset's
// selector starts on, -1 where the stage has no preset line: the heading over the groups, from table.BarLeft, where the
// rows group, the stop the keyboard stands on drawn as focusStop says; and, where the stage has a preset line, the
// selector, as presetLine draws it.
func (m Model) control() (string, int) {
	left, tail := m.controlParts()
	return table.Control(&m, left, tail)
}

// controlParts gives the control line's left part, the heading over the groups, the stop under the keyboard drawn as
// focusStop draws it, and its tail, the preset line's selector, where the stage has one.
func (m Model) controlParts() (left, tail string) {
	left = m.focusStop(strings.Repeat(" ", table.BarLeft) + m.barText())
	if m.presets(m.stage) {
		tail = m.presetLine()
	}
	return left, tail
}

// spots gives what a click on the control line's right part reaches, as table.ControlSpots says.
func (m Model) spots() []table.ControlSpot {
	left, tail := m.controlParts()
	return table.ControlSpots(&m, left, tail)
}

// presetTop gives the screen line the preset line is drawn on, the control line.
func (m Model) presetTop() int { return m.top() }

// pane gives the pane's lines, as many as it has, each as wide as the pane: the control line, where there is one, and
// the rule under the heading over the groups, as table.BarRule draws it, then the selected stage's rows from its offset on, with the header and its rule among them, as laid places them.
func (m Model) pane() []string {
	var out []string
	switch {
	case len(m.stages) == 0:
		out = append(out, "   "+style.Off("no stage is configured"))
	default:
		if m.controls() {
			line, _ := m.control()
			out = append(out, line, table.BarRule(&m))
		}
		texts, _ := m.laid()
		out = append(out, texts...)
	}
	for len(out) < m.paneHeight() {
		out = append(out, "")
	}
	return out
}

// allLines gives a stage's lines: loading, kit's line when it failed, the dim line of an empty stage, then, dim, each
// feature the forge has off here, where kit said so, or the dim line of one whose panel lets no row through, or its
// rows as the grid draws them, the one under the cursor on a band while the rows have the keyboard.
func (m Model) allLines(i int) []table.Line {
	s := m.stages[i]
	switch {
	case s.src.Loading:
		return []table.Line{{Row: -1, Text: lead + style.Off("loading")}}
	case s.refusal != "":
		return m.refusalLines(i)
	}
	rows := m.rows(i)
	switch p := m.panel(i); {
	case len(s.parsed) == 0 && len(s.failed) > 0 && !s.failing():
		// nothing but features off here: the empty line, and what is off under it, in place of the asides' rows
		out := []table.Line{{Row: -1, Text: lead + style.Off(empty(s))}}
		for _, f := range s.failed {
			out = append(out, table.Line{Row: -1, Text: lead + style.Off(m.offText(f))})
		}
		return out
	case len(rows) > 0:
	case p.Query != "" || p.Filtering():
		return []table.Line{{Row: -1, Text: lead + style.Off("no row matches the panel")}}
	default:
		return []table.Line{{Row: -1, Text: lead + style.Off(empty(s))}}
	}
	chosen := -1
	if m.inRows && i == m.stage && s.Cursor < len(rows) {
		chosen = s.Cursor
	}
	key := m.drawnFor(i, rows, chosen)
	if d, ok := m.built.lines[i]; ok && d.key == key {
		return d.lines
	}
	out := m.draw(i, rows, chosen)
	m.built.lines[i] = drawnLines{key, out}
	return out
}

// drawnFor says what a stage's lines are drawn for, beyond what its rows were built for: the rows themselves, the row
// under the cursor, the columns scrolled past, the rows and headings open, and the rows selected. Lines drawn for the
// same are the same, so allLines draws them once and every key, click, and drawing after reuses them, however many
// lines an open row has.
func (m Model) drawnFor(i int, rows []row, chosen int) string {
	s := m.stages[i]
	var b strings.Builder
	fmt.Fprintf(&b, "%p %d %d %d %d", &rows[0], len(rows), chosen, m.gridWidth(i), s.Shift)
	for _, k := range slices.Sorted(maps.Keys(s.Open)) {
		fmt.Fprintf(&b, "\x00o%s=%t", k, s.Open[k])
	}
	for _, k := range slices.Sorted(maps.Keys(s.Picked)) {
		if s.Picked[k] {
			fmt.Fprintf(&b, "\x00p%s", k)
		}
	}
	return b.String()
}

// draw draws a stage's rows as allLines says, the one under the cursor, chosen, on a band; -1 for none: as a
// table.Drawing draws them, in the stage's color.
func (m Model) draw(i int, rows []row, chosen int) []table.Line {
	s := m.stages[i]
	d := table.Drawing{Names: m.names(i), Width: m.gridWidth(i), Shift: s.Shift, Pane: m.paneWidth(), Indent: m.indent(i),
		Open: s.Open, Picked: s.Picked, Color: stageColors[s.name]}
	return d.Draw(rows, chosen)
}

// indent gives the cells a stage's rows, and its header, stand in by, under the headings of its groups: two a level,
// as internal/grid counts them, the heading over the groups apart, which stands on the control line.
func (m Model) indent(i int) int { return 2 * m.shown(i).levels }

// gridWidth gives the cells a stage's rows are laid out in: the pane's, less their indent.
func (m Model) gridWidth(i int) int { return m.paneWidth() - m.indent(i) }

// Noticing says whether the keys bar's line says what was done, or why not, in place of the keys, until a key: kittui
// keeps it on the screen while the keys bar is hidden.
func (m Model) Noticing() bool { return m.note.Text != "" }

// lead is what stands before the rule under the header, as table.Lead says.
const lead = table.Lead

// presets says whether a stage's table has a preset line: one with columns, the views of which it names.
func (m Model) presets(i int) bool { return !m.noPresets && m.columnar(i) }

// columnar says whether a stage's table has columns the columns dialog sets: its fields, loaded, not refused.
func (m Model) columnar(i int) bool {
	return i < len(m.stages) && len(fieldsOf(m.stages[i].name)) > 0 && !m.stages[i].src.Loading && m.stages[i].refusal == ""
}

// dropStart gives the screen column the preset line's selector starts on, at the control line's right end.
func (m Model) dropStart() int {
	_, at := m.control()
	return at
}

// dropEnd gives the screen column after the preset line's selector; a click from dropStart up to it drops the presets
// down.
func (m Model) dropEnd() int { return m.dropStart() + ansi.StringWidth(m.presetLine()) }

// presetLine gives the preset's selector, at the right end of the control line, as table.PresetLine draws it, the
// line having the keyboard while it does and the search does not.
func (m Model) presetLine() string {
	_, on := m.atStop()
	return table.PresetLine(&m, m.onPreset && !m.inPanel && !on)
}
