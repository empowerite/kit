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
		if i, ok := Number(msg); ok && i < n {
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

// Number gives the tab a key names, from 0: 1 to 9, and 0 for the tenth, typed or on the keypad, alone or with alt or
// super.
func Number(k tea.KeyPressMsg) (int, bool) {
	key := k.String()
	for _, mod := range []string{"alt+", "super+"} {
		key = strings.TrimPrefix(key, mod)
	}
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

// paint draws the selected tab's text: in the accent, wherever the keyboard is.
func (b Bar) paint(s string) string { return style.Mark(s) }

// head gives a tab's head on the middle line: its glyph and a space, when it has one. Its number is on the line above.
func (b Bar) head(i int) string {
	if g := b.Tabs[i].Glyph; g != "" {
		return g + " "
	}
	return ""
}

// text gives a tab's name and count and a space after them: the name painted when selected, the count dim.
func (b Bar) text(i int) string {
	t := b.Tabs[i]
	name := t.Name
	if i == b.Selected {
		name = b.paint(name)
	}
	if t.Count != "" {
		name += " " + style.Off(t.Count)
	}
	return name + " "
}

// screen gives the cells of the screen the line takes: all but the margin before it.
func (b Bar) screen() int { return max(1, b.Width-1) }

// layout is the bar laid out on its canvas: where each tab's part starts, its cells, the cell between it and the next
// among them, the canvas's width, never less than the screen's, and the cells scrolled past on its left.
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
		whole[i] = 2 + ansi.StringWidth(b.head(i)+b.text(i)) + 1 // its number's cell and a space, and the cell after it
		wanted += whole[i]
	}
	parts := whole
	if wanted <= w {
		parts = share(w, whole, whole)
	}
	l := layout{at: make([]int, n), parts: parts}
	x := 0
	for i, part := range parts {
		l.at[i], x = x, x+part
	}
	l.width = max(x, w)
	if l.width > w {
		l.scroll = min(max(l.at[b.Selected]+l.parts[b.Selected]/2-w/2, 0), l.width-w)
	}
	return l
}

// share splits so many cells between parts in proportion to what each wants, none less than its floor; the cells the
// rounding leaves go to the largest remainders, the earlier part of two alike first. The floors fit the cells.
func share(cells int, wants, floors []int) []int {
	n := len(wants)
	out, fixed := make([]int, n), make([]bool, n)
	for {
		left, total := cells, 0
		for i := range n {
			if fixed[i] {
				left -= out[i]
			} else {
				total += wants[i]
			}
		}
		if total == 0 {
			return out
		}
		short := false
		for i := range n {
			if !fixed[i] && left*wants[i] < floors[i]*total {
				out[i], fixed[i], short = floors[i], true, true
			}
		}
		if short {
			continue
		}
		var free []int
		given := 0
		for i := range n {
			if !fixed[i] {
				out[i] = left * wants[i] / total
				given += out[i]
				free = append(free, i)
			}
		}
		slices.SortStableFunc(free, func(a, b int) int { return left*wants[b]%total - left*wants[a]%total })
		for _, i := range free[:left-given] {
			out[i]++
		}
		return out
	}
}

// View draws the bar's three lines as the screen shows them, each after a margin of one cell.
func (b Bar) View() string {
	if len(b.Tabs) == 0 {
		return ""
	}
	l, w := b.layout(), b.screen()
	n := len(b.Tabs)
	var over, tabs, under strings.Builder
	cell := func(top, middle, bottom string) {
		over.WriteString(top)
		tabs.WriteString(middle)
		under.WriteString(bottom)
	}
	// where no tab is beside the selected tab's edge, its curve is a half line
	half := func(curve string, end bool) string {
		if end {
			return b.paint("╵")
		}
		return b.paint(curve)
	}
	cell(" ", " ", " ") // the margin
	for i := range b.Tabs {
		// the tab's first cell holds its number, and the selected tab's corner, edge, and curve under it
		cells := l.parts[i] - 1
		text := fill(" "+b.head(i)+b.text(i), cells-1)
		if i == b.Selected {
			cell(b.paint(cmp.Or(Marker(i, true), "╭")), b.paint("│"), half("╯", i == 0))
			cell(b.paint(top(cells-1)), text, strings.Repeat(" ", cells-1))
			// the cell after it: its other edge
			cell(b.paint("╮"), b.paint("│"), half("╰", i == n-1))
			continue
		}
		// the others' borders are dim: from the number over the tab to ╮, down │, and out to the right with ╰, over the
		// line under the tab; the tab before the selected one has no edge after it: its line over it ends in a half
		// line, ╴, short of the selected tab's number, and the line under it runs on to the selected tab's curve
		number := b.dim(Marker(i, false))
		if number == "" {
			number = style.Off("─")
		}
		// the line under it is in the accent, as the selected tab's border is
		cell(number, " ", b.paint(strings.Repeat("─", cells)))
		cell(style.Off(top(cells-1)), b.dim(text), "")
		if i+1 == b.Selected {
			cell(style.Off("╴"), " ", b.paint("─"))
		} else {
			cell(style.Off("╮"), style.Off("│"), style.Off("╰"))
		}
	}
	// the canvas from the margin on: the margin, then the cells the view shows
	shown := func(line string, k int) string {
		margin, rest := ansi.Cut(line, 0, 1), fill(ansi.Cut(line, 1+l.scroll, 1+l.scroll+w), w)
		// a triangle, and a space between it and the tabs, on the middle line
		if l.scroll > 0 {
			margin = " "
			if k == 1 {
				margin, rest = Before, " "+fill(ansi.Cut(rest, 1, w), w-1)
			}
		}
		if l.scroll+w < l.width && k == 1 {
			rest = fill(ansi.Cut(rest, 0, w-2), w-2) + " " + After
		}
		// a cut through a two-cell glyph can leave a cell more than the screen has; the spaces it ends with go, and the
		// colors of what the cut left empty with them
		out := ansi.Truncate(margin+rest, w+1, "")
		return ansi.Truncate(out, ansi.StringWidth(strings.TrimRight(ansi.Strip(out), " ")), "")
	}
	return shown(over.String(), 0) + "\n" + shown(tabs.String(), 1) + "\n" + shown(under.String(), 2)
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
	return style.Off(text) + strings.Repeat(" ", ansi.StringWidth(s)-ansi.StringWidth(text))
}

// fill pads a cut line with spaces to its width, where the cut fell inside a two-cell glyph.
func fill(s string, width int) string {
	return s + strings.Repeat(" ", max(0, width-ansi.StringWidth(s)))
}

// hit gives the tab a column of the screen falls on: the one whose part it is in; on a triangle that says tabs are
// hidden, the nearest of them, which selected comes into view: the last that starts before the view on the left, the
// first that ends after it on the right. The margin without a triangle is no tab's: -1.
func (b Bar) hit(x int) int {
	l, w := b.layout(), b.screen()
	switch {
	case x == 0 && l.scroll > 0:
		hit := 0
		for i, start := range l.at {
			if start < l.scroll {
				hit = i
			}
		}
		return hit
	case x == w && l.scroll+w < l.width:
		for i, start := range l.at {
			if start+l.parts[i] > l.scroll+w {
				return i
			}
		}
	case x < 1 || x > w:
		return -1
	}
	at, hit := x-1+l.scroll, 0
	for i, start := range l.at {
		if start <= at {
			hit = i
		}
	}
	return hit
}
