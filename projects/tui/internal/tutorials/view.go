// The screen drawn: the title, the audiences' tab bar, the selected tab's cards, and the keys bar on the last line.
// Widths are terminal cells, so a two-cell glyph never pushes a line past the edge.
package tutorials

import (
	"strconv"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/empowerite/kit/projects/tui/internal/kit"
	"github.com/empowerite/kit/projects/tui/internal/pane"
	"github.com/empowerite/kit/projects/tui/internal/shell"
	"github.com/empowerite/kit/projects/tui/internal/style"
	"github.com/empowerite/kit/projects/tui/internal/tabs"
)

// Failed is the glyph before kit's refusal.
const Failed = "❌"

// A card is cardWidth cells wide and cardHeight lines tall, its border with it, and its text stands a cell in from
// either side; the cards stand gap cells apart, after the gutter every line keeps, as the tab bar does.
const (
	cardWidth  = 38
	cardHeight = 6
	textWidth  = cardWidth - 4
	gap        = 1
	gutter     = tabs.Gutter
)

// summaryLines is how many lines a card gives its summary.
const summaryLines = 3

// fixed is how many lines the title, the bar, the gap, and the keys take; the pane has the rest, from paneTop on,
// right under the bar.
const (
	fixed   = 3 + tabs.Height
	paneTop = 1 + tabs.Height
)

// lead is what starts a line of the pane that is no card's.
const lead = "  "

// View draws the screen on the alternate screen, with the mouse on for the bar and the cards.
func (m Model) View() tea.View {
	title := pane.Clip(shell.Title(style.Label("kit tutorials"), m.appMenu.Open), m.width-1)
	lines := append([]string{title}, strings.Split(m.bar().View(), "\n")...)
	lines = append(lines, m.pane()...)
	lines = m.appMenu.Overlay(append(lines, ""), m.width)
	v := tea.NewView(strings.Join(style.Bottom(lines, m.height, m.bottom()), "\n"))
	v.AltScreen = true
	v.MouseMode = tea.MouseModeCellMotion
	return v
}

// bottom gives the screen's last line: the notice while there is one, else the keys bar. The link to open by hand
// drops its words before the link, when the line is too narrow for them.
func (m Model) bottom() string {
	room := m.width - 1
	switch {
	case m.notice == "":
		return m.keys().Line(m.width)
	case !m.noted && ansi.StringWidth(m.notice) > room:
		_, url, _ := strings.Cut(strings.TrimSuffix(m.notice, " in a browser"), "Open ")
		return " " + style.Warn(pane.Clip(url, room))
	case !m.noted:
		return " " + style.Warn(m.notice)
	}
	return " " + pane.Clip(m.notice, room)
}

// bar gives the tab bar, from the screen's second line: All, then each audience, each with how many tutorials it has,
// … while kit is asked, and ! in red when kit refused; the keyboard on it unless the cards have it, and its view where
// it was left.
func (m Model) bar() tabs.Bar {
	b := tabs.Bar{Selected: m.tab, Focused: !m.focused(), Scroll: m.barView, Width: m.width, Row: 1}
	names := []string{"All"}
	for _, a := range m.catalog.Audiences {
		names = append(names, a.Label)
	}
	for i, name := range names {
		count := strconv.Itoa(len(m.cardsOf(i)))
		switch {
		case !m.loaded:
			count = "…"
		case m.refusal != "":
			count = "!"
		}
		b.Tabs = append(b.Tabs, tabs.Tab{Name: name, Count: count})
		b.Alert = append(b.Alert, count == "!")
	}
	return b
}

// keys gives the path to the keyboard, which the keys bar is drawn from: the screen, whose = drops the app's menu down;
// the bar, whose down enters the cards where the tab has some; and the cards while they have the keyboard: esc gives
// the bar the keyboard, named by the tab it has selected, the arrows move, and enter opens. An open menu takes every
// press, and is the path alone.
func (m Model) keys() shell.Path {
	if m.appMenu.Open {
		return shell.Path{shell.MenuKeys()}
	}
	down := ""
	if len(m.cards()) > 0 {
		down = "cards"
	}
	path := shell.Path{{Bindings: []shell.Binding{shell.Bind("=", "menu")}}, m.bar().Keys(down)}
	if m.focused() {
		path = append(path, shell.Segment{Bindings: []shell.Binding{shell.Bind("esc", m.bar().Tabs[m.tab].Name),
			shell.Bind("←→↑↓", "move", "left", "right", "up", "down"), shell.Bind("⏎", "open", "enter")}})
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

// rows gives how many rows of cards the pane shows at once, and at least one.
func (m Model) rows() int { return max(1, (m.paneHeight()-m.top())/cardHeight) }

// pane gives the pane's lines, as many as it has: reading… until kit answers, kit's refusal, that the tab has no
// tutorial, or its cards.
func (m Model) pane() []string {
	h := m.paneHeight()
	var out []string
	switch cards := m.cards(); {
	case !m.loaded:
		out = []string{"", lead + style.Off("reading…")}
	case m.refusal != "":
		out = []string{"", lead + style.Refusal(Failed+" "+m.refusal)}
	case len(cards) == 0 && m.tab == 0:
		out = []string{"", lead + style.Off("No tutorials yet")}
	case len(cards) == 0:
		out = []string{"", lead + style.Off("No tutorials for "+m.catalog.Audiences[m.tab-1].Label+" yet")}
	default:
		out = m.grid(cards)
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

// grid gives the cards' lines: the line over them, which says how many cards are scrolled past above, then the rows
// the pane shows from its offset, then, where there is room, how many cards are below.
func (m Model) grid(cards []kit.Tutorial) []string {
	cols, n := m.columns(), len(cards)
	var out []string
	if m.top() > 0 {
		above := ""
		if m.offset > 0 {
			above = lead + style.Off(pane.Above(m.offset*cols))
		}
		out = append(out, above)
	}
	end := min((n+cols-1)/cols, m.offset+m.rows())
	for r := m.offset; r < end; r++ {
		var drawn [][]string
		for i := r * cols; i < min((r+1)*cols, n); i++ {
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
	if below := n - end*cols; below > 0 {
		out = append(out, lead+style.Off(pane.More(below)))
	}
	return out
}

// card gives a tutorial's card, its lines each cardWidth cells: a rounded border, dim, or in the menus' bright cyan
// under the cursor; the glyph and the title in bold; the summary on three lines; and the labels of its audiences,
// dim, on the bottom edge.
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
	out := []string{edge("╭" + strings.Repeat("─", inner) + "╮"), line(style.Label(title))}
	for _, l := range summary(t.Summary) {
		out = append(out, line(l))
	}
	labels := pane.Clip(m.labels(t), inner-4)
	if labels == "" {
		return append(out, edge("╰"+strings.Repeat("─", inner)+"╯"))
	}
	fill := strings.Repeat("─", inner-3-ansi.StringWidth(labels))
	return append(out, edge("╰─ ")+style.Off(labels)+edge(" "+fill+"╯"))
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

// labels gives the labels of the audiences a tutorial is for, in its order, a dot between them; an audience the
// catalogue does not list stands as its id.
func (m Model) labels(t kit.Tutorial) string {
	var out []string
	for _, id := range t.Audience {
		label := id
		for _, a := range m.catalog.Audiences {
			if a.ID == id {
				label = a.Label
			}
		}
		out = append(out, label)
	}
	return strings.Join(out, " · ")
}

// cardAt gives the index of the card at a cell of the screen, in the selected tab's tutorials, or -1 where there is
// none: on the line over the cards, between them, past them, or under the rows the pane shows.
func (m Model) cardAt(x, y int) int {
	r, c, cols := y-paneTop-m.top(), x-gutter, m.columns()
	if r < 0 || c < 0 || r/cardHeight >= m.rows() || c%(cardWidth+gap) >= cardWidth || c/(cardWidth+gap) >= cols {
		return -1
	}
	if i := (m.offset+r/cardHeight)*cols + c/(cardWidth+gap); i < len(m.cards()) {
		return i
	}
	return -1
}

// fit scrolls the pane a row of cards at a time, so the row under the cursor shows, and never past the last row.
func (m *Model) fit() {
	cols, shown := m.columns(), m.rows()
	row, rows := m.cursor/cols, (len(m.cards())+cols-1)/cols
	if row < m.offset {
		m.offset = row
	}
	if row >= m.offset+shown {
		m.offset = row - shown + 1
	}
	m.offset = max(0, min(m.offset, rows-shown))
}
