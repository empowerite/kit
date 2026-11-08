// The picture drawn: the title, the stages' tab bar, and the selected stage's rows in a pane that scrolls, beside its
// panel and legend when ? shows them, and the keys under them all. Widths are terminal cells, so a two-cell glyph never pushes a row past
// the edge.
package flow

import (
	"cmp"
	"slices"
	"strconv"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/empowerite/kit/projects/tui/internal/grid"
	"github.com/empowerite/kit/projects/tui/internal/pane"
	"github.com/empowerite/kit/projects/tui/internal/shell"
	"github.com/empowerite/kit/projects/tui/internal/style"
	"github.com/empowerite/kit/projects/tui/internal/tabs"
)

// top gives the screen line the pane starts on: right under the bar, which is open there to the selected tab; or,
// maximized, under the title and the banner, the line under it in the tab's color.
func (m Model) top() int {
	if m.maximized {
		return 2
	}
	return 1 + tabs.Height
}

// banner gives the line under the title of a maximized tab: Banner, as wide as the pane, in the color the bar draws the
// selected tab in.
func (m Model) banner() string {
	mark, _ := style.In(m.bar().Color)
	return mark(strings.Repeat(Banner, m.paneWidth()))
}

// line is one line of the pane: the row it belongs to, or -1 for a line about the stage, and its text.
type line struct {
	row  int
	text string
}

// View draws the title and, on its right, the search; the bar, or, maximized, the banner; the selected stage's pane, the
// rows the width of the screen; and a gap, over the keys bar on the last line; the legend over them while it shows; on
// the alternate screen, with the mouse on for the bar; and, at the keys bar's right end, watching while the selected
// stage is watched, then each forge's rate limit. A screen too short for all of them keeps the bar, and cuts the rest
// from the bottom.
func (m Model) View() tea.View {
	lines := []string{m.titled()}
	if m.maximized {
		lines = append(lines, m.banner())
	} else {
		lines = append(lines, strings.Split(m.bar().View(), "\n")...)
	}
	lines = append(lines, m.pane()...)
	lines = m.layers().Draw(m, m.overLegend(lines))
	right := quotaText(m.quotas, time.Now())
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
	return shell.View(lines, m.height, bottom)
}

// bar gives the stages' tab bar, from the screen's second line, as wide as barWidth says: each stage's glyph, name,
// and count, the room for the widest count of n/m its rows allow and the menu's glyph kept for each, the keyboard on it unless the rows or the
// panel have it, its view where it was left, the selected stage's group's color, and a dot on each stage that is stale.
func (m Model) bar() tabs.Bar {
	b := tabs.Bar{Selected: m.stage, Focused: !m.inRows && !m.inPanel && !m.onPreset, Scroll: m.barView,
		Width: m.width, Row: 1, Menus: true, MenuOpen: m.tabMenu.Open}
	if len(m.stages) > 0 {
		b.Color = stageColors[m.stages[m.stage].name]
	}
	for i, s := range m.stages {
		b.Tabs = append(b.Tabs, tabs.Tab{Glyph: stageGlyphs[s.name], Name: s.name, Count: m.count(i)})
		b.Reserves = append(b.Reserves, m.reserve(i))
		b.Stale = append(b.Stale, s.stale)
	}
	return b
}

// paneHeight gives the lines the pane has: the screen's from top on, but the keys bar under it, which follows the
// pane's last line at once.
func (m Model) paneHeight() int { return max(1, m.height-m.top()-1) }

// titleWidth gives the cells the title has: all up to two spaces before the search, which is always there.
func (m Model) titleWidth() int { return max(0, m.searchAt()-2) }

// searchAt gives the screen column the search starts at, on the screen's first line, where internal/shell puts every
// screen's.
func (m Model) searchAt() int {
	if len(m.stages) == 0 {
		return m.width
	}
	return shell.SearchAt(m.width, m.searchLine())
}

// searchLine gives the search as the title line draws it.
func (m Model) searchLine() string {
	p := m.panel(m.stage)
	return shell.RecallLine(p.Query, p.Focused, len(m.recent.Items) > 0)
}

// titled gives the title line: the title, clipped, and two spaces after it the search, from searchAt; none with no
// stage.
func (m Model) titled() string {
	title := pane.Clip(m.title(), m.titleWidth())
	if len(m.stages) == 0 {
		return title
	}
	return pane.Pad(title, m.titleWidth(), false) + "  " + pane.Clip(m.searchLine(), m.width-1-m.searchAt())
}

// paneWidth gives the cells the rows have: the screen's, but its last cell.
func (m Model) paneWidth() int { return max(1, m.width-1) }

// keys gives the path to the keyboard, which the keys bar is drawn from: the screen, whose ? and F1 show or hide the
// legend; the stages' bar, whose down reaches the preset line, or the rows where the stage has none, and which,
// maximized, has no left or right; and the part the keyboard is in, the preset line, the rows, or the search.
// An open menu, dialog, or field's popover takes every press, and is the path alone.
func (m Model) keys() shell.Path {
	// an open layer, the topmost, names its keys alone, as layers.go orders them
	if path, ok := m.layers().Keys(m); ok {
		return path
	}
	legend := shell.Bind("?", "legend", "?", "f1")
	if len(m.stages) == 0 {
		return shell.Path{shell.ScreenKeys(legend)}
	}
	down := "rows"
	if m.presets(m.stage) {
		down = "preset"
	}
	bar := m.bar().Keys(down)
	if m.maximized {
		// the bar is not there to move along
		bar.Bindings = slices.DeleteFunc(bar.Bindings, func(b shell.Binding) bool { return b.Key == "←→" })
	}
	path := shell.Path{shell.ScreenKeys(legend), bar}
	switch {
	case m.inPanel:
		return append(path, shell.Segment{Bindings: []shell.Binding{shell.Bind("esc", "back"), shell.Bind("⏎", "back", "enter")}},
			shell.SearchKeys(len(m.recent.Items) > 0))
	case m.onPreset:
		return append(path, m.presetKeys())
	case m.inRows:
		return append(path, m.rowKeys())
	}
	return path
}

// presetKeys gives the segment of the preset line while it has the keyboard: up or esc gives the bar the keyboard,
// named by the stage it has selected, and down the rows, when the stage has some; enter or space drops the presets
// down. While a name is typed for Save As… or Rename…, what is typed is the name, = : and ; apart, which the menus
// take; enter saves under it, or renames, and esc stops.
func (m Model) presetKeys() shell.Segment {
	if st := m.stages[m.stage]; st.savingAs {
		s := shell.Segment{Types: true, Takes: []string{"backspace"}, Passes: []string{"=", ":", ";"}}
		does := "save"
		if st.renaming != "" {
			does = "rename"
		}
		s.Bindings = append(s.Bindings, shell.Bind("⏎", does, "enter"), shell.Bind("esc", "cancel"))
		return s.With(shell.Bind("", "type the preset's name"))
	}
	var s shell.Segment
	name := m.stages[m.stage].name
	s.Bindings = append(s.Bindings, shell.Bind("↑", name, "up"), shell.Bind("esc", name))
	if len(m.rows(m.stage)) > 0 {
		s.Bindings = append(s.Bindings, shell.Bind("↓", "rows", "down"))
	}
	return s.With(shell.Bind("⏎", "presets", "enter", "space"))
}

// rowKeys gives the segment of the rows while they have the keyboard: esc gives the bar the keyboard, named by the
// stage it has selected; up and down move,
// a page at a time too while the rows are more than the pane shows; space selects a row, shift with up or down selects as it moves, and ctrl+a every row; right
// opens the row under the cursor, or unfolds a group's heading, and left closes or folds it, the one that acts named.
func (m Model) rowKeys() shell.Segment {
	seg := shell.Segment{Takes: []string{"left", "right", "alt+left", "alt+right"}, Bindings: []shell.Binding{
		shell.Bind("esc", m.stages[m.stage].name), shell.Bind("↑↓", "move", "up", "down")}}
	if len(m.lines(m.stage)) > m.rowsHeight() {
		seg.Bindings = append(seg.Bindings, shell.Bind("⇞⇟", "page", "pgup", "pgdown"))
	}
	seg.Bindings = append(seg.Bindings, shell.Bind("␣", "select", "space"),
		shell.Bind("⇧↑↓", "select", "shift+up", "shift+down"), shell.Bind("⌃A", "all", "ctrl+a"))
	s, rows := m.stages[m.stage], m.rows(m.stage)
	if s.cursor >= len(rows) {
		return seg
	}
	every := m.every(m.stage)
	at := slices.IndexFunc(every, func(r row) bool { return r.Key == rows[s.cursor].Key })
	switch {
	case m.opens(every, at) && s.open.Unfolds(rows[s.cursor]):
		seg.Bindings = append(seg.Bindings, shell.Bind("←", "collapse", "left"))
	case m.opens(every, at):
		seg.Bindings = append(seg.Bindings, shell.Bind("→", "expand", "right"))
	case grid.Parent(rows, s.cursor) >= m.first(m.stage):
		seg.Bindings = append(seg.Bindings, shell.Bind("←", "group", "left"))
	}
	if grid.Heads(rows[s.cursor]) {
		seg.Bindings = append(seg.Bindings, shell.Bind("⌥←→", "a level", "alt+left", "alt+right"))
	}
	return seg
}

// title gives the title line: as internal/shell draws it, the app's menu's glyph before kit flow; origin's owner/repo
// after its glyph, in its own color, when known; and, maximized, the selected tab, as the bar draws it, its ⋮ last.
func (m Model) title() string {
	title := shell.Title(style.Label("kit flow"), m.appMenu.Open)
	if m.origin.Repo != "" {
		title += " · " + Repository + " " + style.Repo(m.origin.Repo)
	}
	if m.maximized {
		title += " · " + m.bar().Label()
	}
	return title
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

// counted gives how many of the rows are not asides, each once, though grouping by a column of several values stands
// it under each.
func counted(rows []row) int {
	seen := map[string]bool{}
	for _, r := range rows {
		if !r.Aside {
			seen[r.Key] = true
		}
	}
	return len(seen)
}

// names gives the labels of a stage's columns that its panel shows, in its order, as its header stands them over its
// rows, which widen its columns; none while the headers are hidden, as a test of the rows alone asks. A sort's mark
// stands on the rule under the name, not after it, so sorting never widens a column.
func (m Model) names(i int) []string {
	if m.noHeaders {
		return nil
	}
	names := m.shownColumns(i)
	for k, name := range names {
		names[k] = labelOf(m.stages[i].name, name)
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

// header gives the lines that stand over the selected stage's rows, as laid places them, and whether they hide lines
// while closed: the names of its columns the panel shows, as internal/grid lays them over the rows, open or closed as
// the stage's header is, then the rule under them, as rule draws it. While it loads, after a refusal, or when its rows
// have no columns, there are none.
func (m Model) header() (lines []string, hides bool) {
	if len(m.stages) == 0 {
		return nil, false
	}
	if s := m.stages[m.stage]; !m.noHeaders && !s.loading && s.refusal == "" {
		lines, hides = grid.Header(m.rows(m.stage), m.names(m.stage), m.gridWidth(m.stage), s.open[headerKey])
		for k := range lines {
			lines[k] = strings.Repeat(" ", m.indent(m.stage)) + lines[k]
		}
	}
	if len(lines) == 0 {
		return nil, false
	}
	return append(lines, m.rule()), hides
}

// rule gives the line under the header, as wide as the pane: blank up to the first column, past the gutter and the
// rows' indent, then a thin dim line, each sort's mark on it under its column's name, as sortSpots places them, where
// the line reaches; and, where there is no control line to say it, how many rows are selected at its right end, in
// the accent, when some are.
func (m Model) rule() string {
	right := ""
	if n := m.selected(); n > 0 && !m.controls() {
		right = style.Tint(" "+strconv.Itoa(n)+" selected ") + style.Off("─")
	}
	from := m.indent(m.stage) + len(lead)
	fill := max(1, m.paneWidth()-from-ansi.StringWidth(right))
	return ansi.Truncate(strings.Repeat(" ", from)+m.ruleFill(from, fill)+right, m.paneWidth(), "")
}

// controls says whether the selected stage's pane starts with its control line: where its rows group, or it has a
// preset line.
func (m Model) controls() bool {
	return len(m.stages) > 0 && (m.presets(m.stage) || m.barText() != "")
}

// control gives the control line, the pane's first, as wide as the pane, and the screen column the preset's selector
// starts on, -1 where the stage has no preset line: the heading over the groups, from barLeft, where the rows group,
// as internal/grid draws it; and at the line's right end, how many rows are selected, in the accent, when some are,
// then, where the stage has a preset line, a chip per filter the panel sets, as chips gives them, and the selector, as
// presetLine draws it. The chips are clipped first where the line is short, then the heading.
func (m Model) control() (string, int) {
	width, right, tail := m.paneWidth(), "", ""
	if n := m.selected(); n > 0 {
		right = style.Tint(strconv.Itoa(n)+" selected") + "  "
	}
	left := strings.Repeat(" ", barLeft) + m.barText()
	if m.presets(m.stage) {
		tail = m.presetLine()
		if chips := m.chips(m.stage); len(chips) > 0 {
			// two spaces either side of the chips
			room := width - ansi.StringWidth(left) - 2 - ansi.StringWidth(right) - 2 - ansi.StringWidth(tail)
			if room > 1 {
				right += ansi.Truncate(strings.Join(chips, "  "), room, "…") + "  "
			}
		}
	}
	room := max(0, width-ansi.StringWidth(right)-ansi.StringWidth(tail)-1)
	left = ansi.Truncate(left, room, "…")
	fill := max(1, width-ansi.StringWidth(left)-ansi.StringWidth(right)-ansi.StringWidth(tail))
	line := ansi.Truncate(left+strings.Repeat(" ", fill)+right+tail, width, "")
	if tail == "" {
		return line, -1
	}
	return line, ansi.StringWidth(left) + fill + ansi.StringWidth(right)
}

// presetTop gives the screen line the preset line is drawn on, the control line.
func (m Model) presetTop() int { return m.top() }

// rowsTop gives the screen line the rows' window starts on: the pane's first, or the one under the control line.
func (m Model) rowsTop() int {
	if m.controls() {
		return m.top() + 1
	}
	return m.top()
}

// rowsHeight gives the lines the rows have, which scroll: the pane's, less the control line's and the header's, which
// do not.
func (m Model) rowsHeight() int {
	lines, _ := m.header()
	return m.heightUnder(lines)
}

// heightUnder gives the lines the rows have under a header of those lines, as rowsHeight says.
func (m Model) heightUnder(header []string) int {
	return max(1, m.paneHeight()-(m.rowsTop()-m.top())-len(header))
}

// place is a line of the pane under the control line: the index of the stage's line it draws, -1 for a line that
// says how many are hidden, or -2-h for the header's line h.
type place = int

// headerPlace gives the place of the header's line h.
func headerPlace(h int) place { return -2 - h }

// laid gives the pane's lines under the control line, as many as the rows' window and the header take, and the place
// of each: the rows' window, as pane.Lines draws it, with the header's lines and its rule over the first line of a row
// it shows, not a group's heading, and over the … above when that line is next over it; or under the window's last
// line where it shows no row.
func (m Model) laid() (texts []string, places []place) {
	lines, s := m.lines(m.stage), m.stages[m.stage]
	header, _ := m.header()
	h := m.heightUnder(header)
	texts = make([]string, len(lines))
	for i, l := range lines {
		texts[i] = l.text
	}
	head := m.head(lines, h)
	texts = pane.Lines(texts, s.offset, h, head, "   ")
	places = make([]place, len(texts))
	for r := range places {
		places[r] = pane.At(len(lines), h, s.offset, r)
	}
	if head != "" {
		places[0] = slices.IndexFunc(lines, func(l line) bool { return l.row == s.cursor })
	}
	if len(header) == 0 {
		return texts, places
	}
	rows := m.rows(m.stage)
	at := slices.IndexFunc(places, func(k place) bool {
		return k >= 0 && lines[k].row >= 0 && !rows[lines[k].row].Aside
	})
	switch {
	case at < 0:
		at = len(places)
	case at == 1 && places[0] == -1:
		at = 0
	}
	marks := make([]place, len(header))
	for k := range marks {
		marks[k] = headerPlace(k)
	}
	return slices.Insert(texts, at, header...), slices.Insert(places, at, marks...)
}

// placeAt gives the place of the screen line y in the pane under the control line, as laid gives it, and whether y
// falls on one.
func (m Model) placeAt(y int) (place, bool) {
	_, places := m.laid()
	if r := y - m.rowsTop(); r >= 0 && r < len(places) {
		return places[r], true
	}
	return -1, false
}

// headerTop gives the screen line the header's first line stands on, as laid places it, or -1 where there is no
// header.
func (m Model) headerTop() int {
	_, places := m.laid()
	if r := slices.Index(places, headerPlace(0)); r >= 0 {
		return m.rowsTop() + r
	}
	return -1
}

// headerLine gives the line of the header the screen line y falls on, from 0, and whether it falls on one: the rule
// its last.
func (m Model) headerLine(y int) (int, bool) {
	if k, ok := m.placeAt(y); ok && k <= headerPlace(0) {
		return headerPlace(0) - k, true
	}
	return 0, false
}

// onRule says whether the screen line y is the rule under the header.
func (m Model) onRule(y int) bool {
	lines, _ := m.header()
	h, ok := m.headerLine(y)
	return ok && h == len(lines)-1
}

// pane gives the pane's lines, as many as it has, each as wide as the pane: the control line, where there is one, then
// the selected stage's rows from its offset on, with the header and its rule among them, as laid places them.
func (m Model) pane() []string {
	var out []string
	switch {
	case len(m.stages) == 0:
		out = append(out, "   "+style.Off("no stage is configured"))
	default:
		if m.controls() {
			line, _ := m.control()
			out = append(out, line)
		}
		texts, _ := m.laid()
		out = append(out, texts...)
	}
	for len(out) < m.paneHeight() {
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

// lines gives a stage's lines: loading, kit's line when it failed, the dim line of an empty stage, or of one whose
// panel lets no row through, or its rows as the grid draws them, the one under the cursor on a band while the rows have
// the keyboard.
func (m Model) lines(i int) []line {
	s := m.stages[i]
	switch {
	case s.loading:
		return []line{{-1, lead + style.Off("loading")}}
	case s.refusal != "":
		var out []line
		for _, l := range pane.Wrap(Failed+" "+s.refusal, m.paneWidth()-len(lead)) {
			out = append(out, line{-1, lead + style.Refusal(l)})
		}
		return out
	}
	rows := m.rows(i)
	switch p := m.panel(i); {
	case len(rows) > 0:
	case p.Query != "" || p.Filtering():
		return []line{{-1, lead + style.Off("no row matches the panel")}}
	default:
		return []line{{-1, lead + style.Off(empty(s))}}
	}
	chosen := -1
	if m.inRows && i == m.stage && s.cursor < len(rows) {
		chosen = s.cursor
	}
	marked, _ := style.In(stageColors[s.name])
	groups := m.groupMarks(i)
	drawn := grid.Draw(rows, m.names(i), m.gridWidth(i), chosen, s.open.Open,
		func(r row) string {
			switch {
			case grid.Heads(r) && groups[r.Key] != "":
				return marked(groups[r.Key])
			case s.picked[r.Key] && !r.Aside:
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
// as internal/grid counts them, the heading over the groups apart, which stands on the control line.
func (m Model) indent(i int) int { return 2 * m.shown(i).levels }

// gridWidth gives the cells a stage's rows are laid out in: the pane's, less their indent.
func (m Model) gridWidth(i int) int { return m.paneWidth() - m.indent(i) }

// Noticing says whether the keys bar's line says what was done, or why not, in place of the keys, until a key: kittui
// keeps it on the screen while the keys bar is hidden.
func (m Model) Noticing() bool { return m.note.Text != "" }

// lead is what stands before the rule under the header: the gutter the rows' marks and folds take, as internal/grid
// lays it out, two cells for the selection and two for the ▸ or ▾.
const lead = "    "

// presets says whether a stage's table has a preset line: one with columns, the views of which it names.
func (m Model) presets(i int) bool { return !m.noPresets && m.columnar(i) }

// columnar says whether a stage's table has columns the columns dialog sets: its fields, loaded, not refused.
func (m Model) columnar(i int) bool {
	return i < len(m.stages) && len(fieldsOf(m.stages[i].name)) > 0 && !m.stages[i].loading && m.stages[i].refusal == ""
}

// dropStart gives the screen column the preset line's selector starts on, at the control line's right end.
func (m Model) dropStart() int {
	_, at := m.control()
	return at
}

// dropEnd gives the screen column after the preset line's selector; a click from dropStart up to it drops the presets
// down.
func (m Model) dropEnd() int { return m.dropStart() + ansi.StringWidth(m.presetLine()) }

// presetLine gives the preset's selector, at the right end of the control line: the name of the
// preset the table is set as, and Presets; once the table differs from every preset, the name of the one it was last
// set to, in italics, as a hint it has left it; or, while a name is typed for Save As… or Rename…, the name with a
// caret after a dim "save as" or "rename", once the line has the keyboard. The selector is a pill, as a label is: bold
// black on its tab's color, the preset being the tab's, between half circles in that color; on the menus' bright cyan
// while the line has the keyboard, as the line under the keyboard in a menu is.
func (m Model) presetLine() string {
	on := m.onPreset && !m.inPanel
	if st := m.stages[m.stage]; on && st.savingAs {
		label := SavingAs
		if st.renaming != "" {
			label = Renaming
		}
		return style.Off(label) + " " + st.naming + "▏"
	}
	back := stageColors[m.stages[m.stage].name]
	if on {
		back = selectorFocus
	}
	if is := m.is(m.stage); is != "" {
		return selectorPill(is, back, false)
	}
	return selectorPill(m.chosen(m.stage).Name, back, true)
}

// selectorPill draws the preset line's selector: the preset's name, in italics or not, and Presets, bold black on
// back, between half circles in back.
func selectorPill(name, back string, italic bool) string {
	text := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("0")).Background(lipgloss.Color(back))
	return style.PillEnd(style.PillLeft, back) + text.Italic(italic).Render(name) + text.Render(" "+Presets) +
		style.PillEnd(style.PillRight, back)
}

// selectorFocus is the color the preset line's selector is drawn on while the line has the keyboard: the menus' bright
// cyan.
const selectorFocus = "14"

// chips gives a chip per field the stage's panel filters by, in the catalog's order: its label, a colon, and the
// values it lets through, or, ¬ before each, those it holds back, in the accent.
func (m Model) chips(i int) []string {
	var out []string
	for _, f := range fieldsOf(m.stages[i].name) {
		filter := m.stages[i].view.Field(f.ID).Filter
		if len(filter.Values) == 0 {
			continue
		}
		values := slices.Clone(filter.Values)
		if filter.Exclude {
			for k := range values {
				values[k] = "¬" + values[k]
			}
		}
		out = append(out, style.Tint(cmp.Or(f.Label, f.ID)+": "+strings.Join(values, ", ")))
	}
	return out
}
