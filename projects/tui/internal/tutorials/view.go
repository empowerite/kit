// The screen drawn: the title, the table's control line, the cards in their groups, and the keys bar on the last line.
// Widths are terminal cells, so a two-cell glyph never pushes a line past the edge.
package tutorials

import (
	"slices"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/empowerite/kit/projects/tui/internal/grid"
	"github.com/empowerite/kit/projects/tui/internal/kit"
	"github.com/empowerite/kit/projects/tui/internal/pane"
	"github.com/empowerite/kit/projects/tui/internal/shell"
	"github.com/empowerite/kit/projects/tui/internal/style"
	"github.com/empowerite/kit/projects/tui/internal/table"
)

// appLabel is the screen's name on the title line, after the app menu's =; a click on either drops the menu down.
const appLabel = "kit tutorials"

// Failed is the glyph before kit's refusal.
const Failed = "❌"

// A card is cardWidth cells wide and cardHeight lines tall, its border with it, and its text stands a cell in from
// either side; the cards stand gap cells apart, after the gutter every line keeps, as the control line does.
const (
	cardWidth  = 38
	cardHeight = 8
	textWidth  = cardWidth - 4
	gap        = 1
	gutter     = 2
)

// summaryLines is how many lines a card gives its summary.
const summaryLines = 3

// controlTop is the screen line of the table's control line, under the menu bar; the pane has the lines from paneTop
// on, right under it, but the gap and the keys, which fixed counts with the menu bar and the control line.
const (
	controlTop = 1
	paneTop    = controlTop + 1
	fixed      = 4
)

// lead is what starts a line of the pane that is no card's.
const lead = "  "

// View draws the screen on the alternate screen, with the mouse on for the control line and the cards, the table's
// layers over them.
func (m Model) View() tea.View {
	lines := append([]string{m.menuBar().Line(m.width), m.control()}, m.pane()...)
	lines = m.sheet.Over(lines)
	lines = m.recent.Overlay(append(lines, ""), m.recallAt(), m.width)
	lines = m.appMenu.Overlay(lines, m.width)
	lines = m.books.Overlay(lines, m.booksAt(), m.width)
	return shell.View(lines, m.height, m.bottom())
}

// control gives the table's control line, as the table draws it: the heading over the groups and the preset line,
// once there are tutorials, and nothing before.
func (m Model) control() string {
	if len(m.catalog.Tutorials) == 0 {
		return ""
	}
	return pane.Clip(m.sheet.View()[0], m.width-1)
}

// menuBar gives the menu bar, as internal/shell draws every screen's: the kit menu's = before kit tutorials; the
// workspace menu, which names the workspace; and the search field.
func (m Model) menuBar() shell.MenuBar {
	return shell.MenuBar{Screen: appLabel, Open: m.appMenu.Open, Workspace: m.books.Label(), Search: m.searchLine()}
}

// bottom gives the screen's last line: the notice while there is one, else what the table did, or why not, while it
// says it, else the keys bar. The link to open by hand drops its words before the link, when the line is too narrow
// for them.
func (m Model) bottom() string {
	room := m.width - 1
	switch n := m.sheet.Notice(); {
	case m.note.Text == "" && n.Text != "" && n.Done:
		return " " + pane.Clip(n.Text, room)
	case m.note.Text == "" && n.Text != "":
		return " " + style.Refusal(pane.Clip(n.Text, room))
	case m.note.Text == "":
		return m.keys().Line(m.width)
	case !m.note.Done && ansi.StringWidth(m.note.Text) > room:
		_, url, _ := strings.Cut(strings.TrimSuffix(m.note.Text, " in a browser"), "Open ")
		return " " + style.Warn(pane.Clip(url, room))
	case !m.note.Done:
		return " " + style.Warn(m.note.Text)
	}
	return " " + pane.Clip(m.note.Text, room)
}

// keys gives the path to the keyboard, which the key hints are drawn from: the screen, whose = drops the app's menu
// down and / gives the search focus; then the search while it has focus, the control line while it has the keyboard,
// as the table names its keys, but up and esc, which leave it where it is, and down, which enters the cards; or the
// cards while they have it: esc gives the control line the keyboard, up and down move between the rows of cards, tab
// and shift-tab between the cards, and enter opens. An open menu, or a layer of the table's, takes every press, and is
// the path alone.
func (m Model) keys() shell.Path {
	if m.appMenu.Open || m.recent.Open || m.books.Open {
		return shell.Path{shell.MenuKeys()}
	}
	seg, layer := m.sheet.Keys()
	if layer {
		return shell.Path{seg}
	}
	path := shell.Path{shell.WorkspaceScreenKeys()}
	switch {
	case m.search.Focused:
		return append(path, shell.SearchKeys("cards", m.search.Query, len(m.recent.Items) > 0))
	case m.controlled():
		seg.Bindings = slices.DeleteFunc(slices.Clone(seg.Bindings), func(b shell.Binding) bool {
			return b.Does == table.TabsHint
		})
		for k, b := range seg.Bindings {
			if b.Does == "rows" {
				seg.Bindings[k].Does = "cards"
			}
		}
		return append(path, seg)
	case m.focused():
		return append(path, shell.Segment{Bindings: []shell.Binding{shell.Bind("esc", "controls"),
			shell.Bind("↑↓", "move", "up", "down"),
			shell.Bind("⇥·⇧⇥", "card", "tab", "shift+tab"), shell.Bind("⏎", "open", "enter")}})
	}
	return path
}

// paneHeight gives the lines the pane has.
func (m Model) paneHeight() int { return max(1, m.height-fixed) }

// top gives the lines the pane keeps over the cards: one, which says how many cards are scrolled past, where the pane
// is taller than a card, and none where it is not, so the card shows from its top.
func (m Model) top() int {
	if m.paneHeight() > cardHeight {
		return 1
	}
	return 0
}

// columns gives how many cards stand side by side: as many as fit short of the screen's last cell, and at least one.
func (m Model) columns() int { return max(1, (m.width-1-gutter+gap)/(cardWidth+gap)) }

// band is a stretch of the pane: a group's heading, its line as the table draws it, or a row of cards, the indexes of
// their tutorials among the cards.
type band struct {
	heading string
	cards   []int
}

// height gives the lines a band takes: a heading's one, a row of cards' cardHeight.
func (b band) height() int {
	if b.cards == nil {
		return 1
	}
	return cardHeight
}

// bands gives the pane's bands, as the table's rows stand: each group's heading, then its cards, as many to a row as
// stand side by side; the heading over the groups stands on the control line, so it is none.
func (m Model) bands() []band {
	rows, cols := m.sheet.Rows(), m.columns()
	texts := map[int]string{}
	for _, l := range m.sheet.Lines() {
		if _, ok := texts[l.Row]; !ok {
			texts[l.Row] = l.Text
		}
	}
	var out []band
	card := 0
	for k, r := range rows {
		switch {
		case grid.OverGroups(r):
		case grid.Heads(r):
			out = append(out, band{heading: texts[k]})
		case r.Aside:
		default:
			if n := len(out); n == 0 || out[n-1].cards == nil || len(out[n-1].cards) == cols {
				out = append(out, band{cards: []int{}})
			}
			out[len(out)-1].cards = append(out[len(out)-1].cards, card)
			card++
		}
	}
	return out
}

// bandOf gives the band a card stands in and its column there; the first band for a card in none.
func (m Model) bandOf(bands []band, card int) (b, col int) {
	for k, x := range bands {
		if i := slices.Index(x.cards, card); i >= 0 {
			return k, i
		}
	}
	return 0, 0
}

// room gives the lines the pane has for its bands, under the line over them.
func (m Model) room() int { return m.paneHeight() - m.top() }

// shownUntil gives the band past the last the pane shows from the band from: as many as fit, and at least one.
func (m Model) shownUntil(bands []band, from int) int {
	used, k := 0, from
	for ; k < len(bands); k++ {
		if used+bands[k].height() > m.room() && k > from {
			break
		}
		used += bands[k].height()
	}
	return k
}

// pane gives the pane's lines, as many as it has: reading… until kit answers, kit's refusal, that there is no
// tutorial, that the search or the filter lets none through, or the cards.
func (m Model) pane() []string {
	h := m.paneHeight()
	var out []string
	switch bands := m.bands(); {
	case m.reading.Loading:
		out = []string{"", lead + style.Off("reading…")}
	case m.refusal != "":
		out = []string{"", lead + style.Refusal(Failed+" "+m.refusal)}
	case len(m.catalog.Tutorials) == 0:
		out = []string{"", lead + style.Off("No tutorials yet")}
	case len(bands) == 0:
		out = []string{"", lead + style.Off("No tutorial matches the search or the filter")}
	default:
		out = m.grid(bands)
	}
	out = out[:min(len(out), h)]
	for k := range out {
		out[k] = pane.Clip(out[k], m.width-1)
	}
	for len(out) < h {
		out = append(out, "")
	}
	return out
}

// grid gives the bands' lines: the line over them, which says how many cards are scrolled past above, then the bands
// the pane shows from its offset, then, where there is room, how many cards are below.
func (m Model) grid(bands []band) []string {
	cards := m.cards()
	var out []string
	count := func(bands []band) int {
		n := 0
		for _, b := range bands {
			n += len(b.cards)
		}
		return n
	}
	if m.top() > 0 {
		above := ""
		if n := count(bands[:m.offset]); n > 0 {
			above = lead + style.Off(pane.Above(n))
		}
		out = append(out, above)
	}
	end := m.shownUntil(bands, m.offset)
	for _, b := range bands[m.offset:end] {
		if b.cards == nil {
			out = append(out, b.heading)
			continue
		}
		var drawn [][]string
		for _, i := range b.cards {
			drawn = append(drawn, m.card(cards[i], m.focused() && i == m.cursor))
		}
		for k := range cardHeight {
			var parts []string
			for _, c := range drawn {
				parts = append(parts, c[k])
			}
			out = append(out, strings.Repeat(" ", gutter)+strings.Join(parts, strings.Repeat(" ", gap)))
		}
	}
	if below := count(bands[end:]); below > 0 {
		out = append(out, lead+style.Off(pane.More(below)))
	}
	return out
}

// card gives a tutorial's card, its lines each cardWidth cells: a rounded border, dim, or in the menus' bright cyan
// under the cursor; the glyph and the title in bold; a blank line; the summary on three lines; another blank line, so
// the summary stands clear of the pills; and a pill for each of its audiences on the bottom edge.
func (m Model) card(t kit.Tutorial, cursor bool) []string {
	edge, inner := style.Off, cardWidth-2
	if cursor {
		edge = style.Menu
	}
	line := func(s string) string {
		return edge("│") + " " + pane.Pad(pane.Clip(s, textWidth), textWidth, false) + " " + edge("│")
	}
	title := t.Title
	if t.Glyph != "" {
		title = t.Glyph + " " + title
	}
	out := []string{edge("╭" + strings.Repeat("─", inner) + "╮"), line(style.Label(title)), line("")}
	for _, l := range summary(t.Summary) {
		out = append(out, line(l))
	}
	out = append(out, line(""))
	// the pills stand a cell in from the corner's ─ and leave a ─ of the edge before the other corner
	drawn := pills(m.audiences(t), inner-4, edge)
	if drawn == "" {
		return append(out, edge("╰"+strings.Repeat("─", inner)+"╯"))
	}
	fill := strings.Repeat("─", inner-3-ansi.StringWidth(drawn))
	return append(out, edge("╰─ ")+drawn+edge(" "+fill+"╯"))
}

// summary gives a summary's three lines, broken at spaces to the card's text; one that runs over ends its third line
// with …, after the last word that leaves room for it.
func summary(text string) []string {
	lines := pane.Break(text, textWidth)
	if len(lines) > summaryLines {
		third := lines[summaryLines-1]
		words := strings.Fields(third)
		for len(words) > 0 && ansi.StringWidth(strings.Join(words, " ")+"…") > textWidth {
			words = words[:len(words)-1]
		}
		if len(words) == 0 {
			// one word the width of the card, broken across lines: cut, to leave the … room
			words = []string{ansi.Truncate(third, textWidth-1, "")}
		}
		lines = append(lines[:summaryLines-1], strings.Join(words, " ")+"…")
	}
	for len(lines) < summaryLines {
		lines = append(lines, "")
	}
	return lines
}

// audiences gives the audiences a tutorial is for: those the catalogue lists, in its order, then any it does not, each
// its id alone, in the tutorial's order.
func (m Model) audiences(t kit.Tutorial) []kit.Audience {
	var out []kit.Audience
	for _, a := range m.catalog.Audiences {
		if slices.Contains(t.Audience, a.ID) {
			out = append(out, a)
		}
	}
	for _, id := range t.Audience {
		if !slices.ContainsFunc(m.catalog.Audiences, func(a kit.Audience) bool { return a.ID == id }) {
			out = append(out, kit.Audience{ID: id, Label: id})
		}
	}
	return out
}

// pills gives a pill for each audience, its label on its color, a cell apart, in room cells. Where they do not all
// fit, the pills that leave room for a cell and … stand whole, then the …, in the edge's style; where not even the
// first does, it stands alone, its label cut with … to fit, so no pill is cut through its half circles.
func pills(audiences []kit.Audience, room int, edge func(string) string) string {
	all := make([]string, len(audiences))
	for k, a := range audiences {
		all[k] = style.Pill(a.Label, a.Color)
	}
	if whole := strings.Join(all, " "); ansi.StringWidth(whole) <= room {
		return whole
	}
	more := " " + edge("…")
	var fit []string
	for _, p := range all {
		if ansi.StringWidth(strings.Join(append(fit, p), " ")+more) > room {
			break
		}
		fit = append(fit, p)
	}
	if len(fit) > 0 {
		return strings.Join(fit, " ") + more
	}
	first := audiences[0]
	if len(audiences) == 1 {
		more = ""
	}
	// the label's cells: the room less the half circles and whatever follows
	cut := ansi.Truncate(first.Label, room-2-ansi.StringWidth(more), "…")
	return style.Pill(cut, first.Color) + more
}

// cardAt gives the index of the card at a cell of the screen, among the cards, or -1 where there is none: on the line
// over the cards, on a heading, between them, past them, or under the bands the pane shows.
func (m Model) cardAt(x, y int) int {
	r, c := y-paneTop-m.top(), x-gutter
	if r < 0 || c < 0 || c%(cardWidth+gap) >= cardWidth {
		return -1
	}
	bands := m.bands()
	for _, b := range bands[min(m.offset, len(bands)):m.shownUntil(bands, m.offset)] {
		if r < b.height() {
			if col := c / (cardWidth + gap); b.cards != nil && col < len(b.cards) && r < cardHeight {
				return b.cards[col]
			}
			return -1
		}
		r -= b.height()
	}
	return -1
}

// fit scrolls the pane a band at a time, so the band under the cursor shows whole, with its group's heading where it is
// its group's first row and both fit, and never past the last band; once the wheel scrolled it, only never past either
// end, until a key.
func (m *Model) fit() {
	bands := m.bands()
	if len(bands) == 0 {
		m.offset = 0
		return
	}
	b, _ := m.bandOf(bands, m.cursor)
	if !m.wheeled && b < m.offset {
		m.offset = b
		if b > 0 && bands[b-1].cards == nil && m.shownUntil(bands, b-1) > b {
			m.offset = b - 1
		}
	}
	for !m.wheeled && m.offset < b && m.shownUntil(bands, m.offset) <= b {
		m.offset++
	}
	last := len(bands) - 1
	for last > 0 && m.shownUntil(bands, last-1) == len(bands) {
		last--
	}
	m.offset = max(0, min(m.offset, last))
}
