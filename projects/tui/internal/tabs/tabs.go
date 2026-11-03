// Package tabs is the tab bar kit flow and kit welcome share: three lines, as tabs on folders. Each tab's part starts
// with a cell of its own, which holds its number on the line above; then a space, its glyph, a space, its name, and its
// count on the middle line; then a cell between it and the next. Each line over a tab starts with a half line, ╶, apart
// from the number before it. Each tab but the selected one has a dim border: a line from its number over it to ╮ in
// that cell, │ under that, and ╰ under that, out to the right over the line under the tab; the tab before the selected
// one has no such edge: its line over it ends in a half line, ╴, short of the selected tab's number, and the line under
// it runs on to the selected tab's curve. The selected tab has a line over it, and is open below to the pane under the
// bar. The selected tab's edges stand in its first cell and in the cell after it: its number in the corner and ╮, each
// over │, and under that a curve, ╯ and ╰, to the line under the tab beside it, or a half line, ╵, at either end of the
// bar. A number stays where it is, selected or not. When every tab fits whole, they share the line in proportion to
// their widths; when they do not, the bar is wider than the screen, and the view scrolls sideways to center the
// selected tab, with ◃ before the tabs it hides on the left and ▹ after those on the right, a tab at either edge cut
// there. The first ten tabs are numbered ➀ to ➉, the selected one ➊ to ➓; past the tenth a tab has no number, and its
// border runs through the number's cell. The selected tab, its lines, and the line under every tab are in the accent,
// wherever the keyboard is; while the pane under the bar has it, the other tabs are dim. Widths are terminal cells, so
// a two-cell glyph never pushes the line past the edge.
//
// The bar holds no state of its own: a screen builds it from its own, each time, and reads back what Update chose. Left
// and right move along it while it has the keyboard, and stop at its ends; a number, 1 to 9 or 0 for the tenth, typed,
// on the keypad, with alt, or with super, which a terminal that reports it sends for command, selects that tab from
// anywhere, as a click on the bar does, and gives the bar the keyboard; a click on a triangle selects the nearest tab
// it hides, which then comes into view.
package tabs

import (
	"cmp"
	"slices"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/empowerite/kit/projects/tui/internal/pane"
	"github.com/empowerite/kit/projects/tui/internal/style"
)

// The triangles that say the view hides tabs: before the first shown, and after the last. They are white, apart from
// the black ones the screens draw for the cursor and the legend.
const (
	Before = "◃"
	After  = "▹"
)

// Height is the lines the bar takes.
const Height = 3

// Tab is one tab: its glyph, its name, and its count, which may be empty.
type Tab struct {
	Glyph, Name, Count string
}

// Bar is the tab bar as a screen has it now: the tabs, the selected one, whether the bar has the keyboard, the
// screen's width, and the screen line it starts on, which a click is matched against.
type Bar struct {
	Tabs     []Tab
	Selected int
	Focused  bool
	Width    int
	Row      int
}

// Update takes a message the bar acts on, and says whether it did: left and right while the bar has the keyboard, at an
// end too, where they move nothing; a number that names a tab; and a left click on any of the bar's lines. A number
// and a click give the bar the keyboard.
func (b Bar) Update(msg tea.Msg) (Bar, bool) {
	n := len(b.Tabs)
	if n == 0 {
		return b, false
	}
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch k := msg.String(); {
		case b.Focused && k == "left":
			b.Selected = max(b.Selected-1, 0)
			return b, true
		case b.Focused && k == "right":
			b.Selected = min(b.Selected+1, n-1)
			return b, true
		}
		if i, ok := number(msg); ok && i < n {
			b.Selected, b.Focused = i, true
			return b, true
		}
	case tea.MouseClickMsg:
		mouse := msg.Mouse()
		if mouse.Button != tea.MouseLeft || mouse.Y < b.Row || mouse.Y >= b.Row+Height {
			return b, false
		}
		if i := b.hit(mouse.X); i >= 0 {
			b.Selected, b.Focused = i, true
			return b, true
		}
	}
	return b, false
}

// number gives the tab a key names, from 0: 1 to 9, and 0 for the tenth, typed or on the keypad, alone or with alt or
// super.
func number(k tea.KeyPressMsg) (int, bool) {
	key := strings.TrimPrefix(strings.TrimPrefix(k.String(), "alt+"), "super+")
	key = strings.TrimPrefix(key, "kp")
	switch {
	case key == "0":
		return 9, true
	case len(key) == 1 && key[0] >= '1' && key[0] <= '9':
		return int(key[0] - '1'), true
	}
	return 0, false
}

// Marker gives a tab's number: ➀ to ➉, ➊ to ➓ when selected, and past the tenth nothing, its cell then the tab's
// border. The dingbats' sans-serif digits are one cell in every terminal, as Unicode fixes their width, and the two
// sets come from one font; ① and ❶ are of ambiguous width, and terminals draw them from fonts of different sizes.
func Marker(i int, selected bool) string {
	switch {
	case i >= 10:
		return ""
	case selected:
		return string(rune('➊' + i))
	}
	return string(rune('➀' + i))
}

// label gives a tab's label on the middle line: its glyph and a space, when it has one; its name, in the accent when
// selected; its count, dim; and a space after them. Its number is on the line above.
func (b Bar) label(i int) string {
	t, label := b.Tabs[i], ""
	if t.Glyph != "" {
		label = t.Glyph + " "
	}
	if i == b.Selected {
		label += style.Mark(t.Name)
	} else {
		label += t.Name
	}
	if t.Count != "" {
		label += " " + style.Off(t.Count)
	}
	return label + " "
}

// screen gives the cells of the screen the line takes: all but the margin before it.
func (b Bar) screen() int { return max(1, b.Width-1) }

// layout is the bar laid out on its canvas: where each tab's part starts, its cells, the cell between it and the next
// among them, the canvas's width, and the cells scrolled past on its left.
type layout struct {
	at, parts []int
	width     int
	scroll    int
}

// layout lays each tab out whole, its number's cell and a space before it and the cell after it besides. When they
// fit the screen, they share it in proportion to their widths; when they do not, the canvas is as wide as they are,
// and the view scrolls so the selected tab is in its middle, or as near as the ends of the canvas let.
func (b Bar) layout() layout {
	n, w := len(b.Tabs), b.screen()
	whole := make([]int, n)
	wanted := 0
	for i := range n {
		whole[i] = 2 + ansi.StringWidth(b.label(i)) + 1 // its number's cell and a space, and the cell after it
		wanted += whole[i]
	}
	parts := whole
	if wanted <= w {
		parts = share(w, whole)
	}
	l := layout{at: make([]int, n), parts: parts}
	for i, part := range parts {
		l.at[i], l.width = l.width, l.width+part
	}
	if l.width > w {
		l.scroll = min(max(l.at[b.Selected]+l.parts[b.Selected]/2-w/2, 0), l.width-w)
	}
	return l
}

// share splits so many cells between parts in proportion to what each wants; the cells the rounding leaves go to the
// largest remainders, the earlier part of two alike first.
func share(cells int, wants []int) []int {
	total := 0
	for _, want := range wants {
		total += want
	}
	out, order, given := make([]int, len(wants)), make([]int, len(wants)), 0
	for i, want := range wants {
		out[i], order[i] = cells*want/total, i
		given += out[i]
	}
	slices.SortStableFunc(order, func(a, b int) int { return cells*wants[b]%total - cells*wants[a]%total })
	for _, i := range order[:cells-given] {
		out[i]++
	}
	return out
}

// View draws the bar's three lines as the screen shows them, each after a margin of one cell, as the package says.
func (b Bar) View() string {
	if len(b.Tabs) == 0 {
		return ""
	}
	l, w, n := b.layout(), b.screen(), len(b.Tabs)
	var over, tabs, under strings.Builder
	cell := func(top, middle, bottom string) {
		over.WriteString(top)
		tabs.WriteString(middle)
		under.WriteString(bottom)
	}
	// curve draws a curve under the selected tab's edge, or a half line where no tab is beside it
	curve := func(glyph string, end bool) string {
		if end {
			glyph = "╵"
		}
		return style.Mark(glyph)
	}
	for i := range b.Tabs {
		// the tab's first cell holds its number, and the selected tab's corner, edge, and curve under it
		cells := l.parts[i] - 1
		label := pane.Pad(" "+b.label(i), cells-1, false)
		if i == b.Selected {
			cell(style.Mark(cmp.Or(Marker(i, true), "╭")), style.Mark("│"), curve("╯", i == 0))
			cell(style.Mark(top(cells-1)), label, strings.Repeat(" ", cells-1))
			cell(style.Mark("╮"), style.Mark("│"), curve("╰", i == n-1))
			continue
		}
		// the others' borders are dim, and the line under them in the accent; the tab before the selected one has no
		// edge after it
		number := b.dim(Marker(i, false))
		if number == "" {
			number = style.Off("─")
		}
		cell(number, " ", style.Mark(strings.Repeat("─", cells)))
		cell(style.Off(top(cells-1)), b.dim(label), "")
		if i+1 == b.Selected {
			cell(style.Off("╴"), " ", style.Mark("─"))
		} else {
			cell(style.Off("╮"), style.Off("│"), style.Off("╰"))
		}
	}
	// shown gives a line as the view shows it, after the margin, the triangles on the middle line
	shown := func(line string, middle bool) string {
		margin, rest := " ", pane.Pad(ansi.Cut(line, l.scroll, l.scroll+w), w, false)
		if middle && l.scroll > 0 {
			margin, rest = Before, " "+pane.Pad(ansi.Cut(rest, 1, w), w-1, false)
		}
		if middle && l.scroll+w < l.width {
			rest = pane.Pad(ansi.Cut(rest, 0, w-2), w-2, false) + " " + After
		}
		// a cut through a two-cell glyph can leave a cell more than the screen has; the spaces it ends with go, and the
		// colors of what the cut left empty with them
		out := ansi.Truncate(margin+rest, w+1, "")
		return ansi.Truncate(out, ansi.StringWidth(strings.TrimRight(ansi.Strip(out), " ")), "")
	}
	return shown(over.String(), false) + "\n" + shown(tabs.String(), true) + "\n" + shown(under.String(), false)
}

// top gives the line over a tab, so many cells of it, never fewer than two, as no tab is narrower than four: it starts
// with a half line, ╶, apart from the number before it.
func top(cells int) string { return "╶" + strings.Repeat("─", cells-1) }

// dim draws what a tab other than the selected one draws dim, its own colors stripped, while the pane under the bar has
// the keyboard; as it is while the bar has it.
func (b Bar) dim(s string) string {
	if b.Focused || strings.TrimSpace(s) == "" {
		return s
	}
	// the spaces after it stay out of the dim, so a line can end without them
	text := strings.TrimRight(ansi.Strip(s), " ")
	return pane.Pad(style.Off(text), ansi.StringWidth(s), false)
}

// hit gives the tab a column of the screen falls on: the one whose part covers it; on a triangle that says tabs are
// hidden, the one that covers the cell just past the view, which selected comes into view. The margin without a
// triangle is no tab's: -1.
func (b Bar) hit(x int) int {
	l, w := b.layout(), b.screen()
	var at int
	switch {
	case x == 0 && l.scroll > 0:
		at = l.scroll - 1
	case x == w && l.scroll+w < l.width:
		at = l.scroll + w
	case x < 1 || x > w:
		return -1
	default:
		at = x - 1 + l.scroll
	}
	hit := 0
	for i, start := range l.at {
		if start <= at {
			hit = i
		}
	}
	return hit
}
