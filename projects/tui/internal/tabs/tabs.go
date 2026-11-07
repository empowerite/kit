// Package tabs is the tab bar kit flow and kit welcome share: three lines, as tabs on folders. Each tab's part starts
// with a cell of its own, which holds its number on the line above; then a space, its glyph, a space, and its name on
// the middle line, and its count right-aligned at the tab's end, the menu's glyph after it on the selected tab of a
// bar with menus; then a cell between it and the next. Each line over a tab starts with a half line, ╶, apart
// from the number before it. Each tab but the selected one has a dim border: a line from its number over it to ╮ in
// that cell, │ under that, and ╰ under that, out to the right over the line under the tab, or a half line, ╵, on the
// last tab; the tab before the selected
// one has no such edge: its line over it ends in a half line, ╴, short of the selected tab's number, and the line under
// it runs on to the selected tab's curve. The selected tab has a line over it, and is open below to the pane under the
// bar. The selected tab's edges stand in its first cell and in the cell after it: its number in the corner and ╮, each
// over │, and under that a curve, ╯ and ╰, to the line under the tab beside it, or a half line, ╵, at either end of the
// bar. A number stays where it is, selected or not. When every tab fits whole, they share the line in proportion to
// their widths; when they do not, the bar is wider than the screen, and the view scrolls sideways, with ◃ in the
// gutter before the bar for the tabs it hides on the left and ▹ after those on the right, a tab at either edge cut
// there. The selected tab always shows whole, with its count and its menu's glyph: when it would not, the view centers
// it, as near as the bar's ends let. A triangle scrolls the view to show whole the nearest tab it hides, never
// selecting it; one that would cut the selected tab is hidden, its cell left blank, and moves nothing. The first ten tabs are numbered ➀ to ➉, the selected one ➊ to ➓; past the tenth a tab has no number, and its
// border runs through the number's cell. The selected tab, its lines, and the line under every tab are in the accent,
// wherever the keyboard is; while the pane under the bar has it, the other tabs are dim, all but a count that wants
// attention. Widths are terminal cells, so a two-cell glyph never pushes the line past the edge.
//
// The bar holds no state of its own: a screen builds it from its own, each time, and reads back what Update chose. Left
// and right move along it while it has the keyboard, and stop at its ends; a number, 1 to 9 or 0 for the tenth, typed,
// on the keypad, with alt, or with super, which a terminal that reports it sends for command, selects that tab from
// anywhere, as a click on the bar does, and gives the bar the keyboard; a click on a triangle scrolls the view, and
// leaves the selected tab and the keyboard as they were.
package tabs

import (
	"cmp"
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/empowerite/kit/projects/tui/internal/pane"
	"github.com/empowerite/kit/projects/tui/internal/shell"
	"github.com/empowerite/kit/projects/tui/internal/style"
)

// The triangles that say the view hides tabs: before the first shown, and after the last. They are white, apart from
// the black ones the screens draw for the cursor and the legend.
const (
	Before = "◃"
	After  = "▹"
)

// Stale is the dot after a tab's name while what it shows is stale, as kept from before, or since the source changed:
// yellow, wherever the keyboard is.
const Stale = "•"

// Menu is the glyph after the selected tab's count, which will open the tab's menu: a vertical ellipsis, a kebab, as a
// menu of actions on what it stands beside is drawn, narrow in every terminal.
const Menu = "⋮"

// Height is the lines the bar takes.
const Height = 3

// Tab is one tab: its glyph, its name, and its count, which may be empty.
type Tab struct {
	Glyph, Name, Count string
}

// Bar is the tab bar as a screen has it now: the tabs, the selected one, whether the bar has the keyboard, the
// screen's width, and the screen line it starts on, which a click is matched against. Reserve is the cells a tab's
// count is laid out for at least, so a count that grows, as one a filter narrows to n/m, does not move the tabs;
// Reserves is the same for each tab, by index, beside it.
// Menus says whether the selected tab draws Menu after its count; every tab keeps the room for it, so moving along
// the bar moves nothing. MenuOpen says whether the tab's menu is open: the glyph is then in the menu's color, and the
// count blank. Joined says a line runs
// down from the bar's last column, as a rule beside what is under the bar: the last cell of the bar's last line turns
// down into it. Color is what the selected tab, and the line under the others, are drawn in, a 256-color number, as a
// stage's group has one; none is the accent. Alert says, by index, which tabs' counts want attention, as a step's
// failed checks do. Stale says, by index, which tabs show what is stale: each has Stale after its name, in a cell of
// the room before its count, so the dot moves no tab; a tab with no room to spare draws none. Scroll is the cells of
// the bar the view has scrolled past on its left, as the screen keeps it from
// what Update chose, so a triangle's scroll stays; the view goes on from there while the selected tab shows whole.
type Bar struct {
	Tabs     []Tab
	Selected int
	Focused  bool
	Scroll   int
	Width    int
	Row      int
	Reserve  int
	Reserves []int
	Menus    bool
	MenuOpen bool
	Joined   bool
	Color    string
	Alert    []bool
	Stale    []bool
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
			b.Scroll, b.Selected = b.layout().scroll, max(b.Selected-1, 0)
			return b, true
		case b.Focused && k == "right":
			b.Scroll, b.Selected = b.layout().scroll, min(b.Selected+1, n-1)
			return b, true
		}
		if i, ok := number(msg); ok && i < n {
			b.Scroll, b.Selected, b.Focused = b.layout().scroll, i, true
			return b, true
		}
	case tea.MouseClickMsg:
		mouse := msg.Mouse()
		if mouse.Button != tea.MouseLeft || mouse.Y < b.Row || mouse.Y >= b.Row+Height {
			return b, false
		}
		l, w := b.layout(), b.screen()
		switch {
		case mouse.X < Gutter && l.scroll > 0:
			// a triangle scrolls the view, and leaves the selected tab and the keyboard where they are; hidden, its
			// cell is taken, and moves nothing
			b.Scroll = l.scroll
			if to, ok := l.before(w, b.Selected); ok {
				b.Scroll = to
			}
			return b, true
		case mouse.X == Gutter+w-1 && l.scroll+w < l.width:
			b.Scroll = l.scroll
			if to, ok := l.after(w, b.Selected); ok {
				b.Scroll = to
			}
			return b, true
		}
		if i := b.hit(mouse.X); i >= 0 {
			b.Selected, b.Focused, b.Scroll = i, true, l.scroll
			return b, true
		}
	}
	return b, false
}

// Keys gives the bar's segment of the keys bar: while it has the keyboard, left and right move along it, and down, when
// it says what down enters, enters it; a tab's number selects it from anywhere; and where the bar has menus, : or ;
// drops the selected tab's down, as the screen takes it.
func (b Bar) Keys(down string) shell.Segment {
	var s shell.Segment
	if len(b.Tabs) == 0 {
		return s
	}
	if b.Focused {
		s.Bindings = append(s.Bindings, shell.Bind("←→", "tab", "left", "right"))
		if down != "" {
			s.Bindings = append(s.Bindings, shell.Bind("↓", down, "down"))
		}
	}
	s.Bindings = append(s.Bindings, shell.Bind(fmt.Sprintf("1-%d", min(len(b.Tabs), 9)), "jump", shell.Digits...))
	if b.Menus {
		s.Bindings = append(s.Bindings, shell.Bind(":", "tab menu", ":", ";"))
	}
	return s
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

// label gives a tab's label on the middle line, as two ends: its glyph and a space, when it has one, and its name, in
// the accent when selected; then, right-aligned in the tab, its count, dim, or bold and red when it wants attention,
// and the menu's glyph after a space on the selected tab, when the bar has menus, its count blank while the menu is
// open. Its number is on the line above.
func (b Bar) label(i int) (name, end string) {
	t := b.Tabs[i]
	if t.Glyph != "" {
		name = t.Glyph + " "
	}
	mark, tint := style.In(b.Color)
	if i == b.Selected {
		name += mark(t.Name)
	} else {
		name += t.Name
	}
	end = style.Off(t.Count)
	if b.alert(i) {
		end = style.Loss(t.Count)
	}
	if b.Menus && i == b.Selected {
		glyph := tint(Menu)
		if b.MenuOpen {
			// the open menu hangs from the glyph, and the count would show beside its edge: its cells stay, blank
			end, glyph = strings.Repeat(" ", ansi.StringWidth(t.Count)), style.Menu(Menu)
		}
		end += " " + glyph
	}
	return name, end
}

// Label gives the selected tab's label as the bar draws it on its middle line, on one line with a space between its
// parts: its glyph, its name in the bar's color, its count, and the menu's glyph, as label says; a screen that draws the
// tab without the bar draws it so. No tabs give nothing.
func (b Bar) Label() string {
	if len(b.Tabs) == 0 {
		return ""
	}
	name, end := b.label(b.Selected)
	if ansi.StringWidth(b.Tabs[b.Selected].Count) > 0 {
		name += " "
	}
	return name + end
}

// wants gives the cells a tab's label takes at least: its name, a space and its count, or the bar's reserve or the
// tab's own in Reserves when the count is narrower, room for a space and the menu's glyph on every tab when the bar
// has menus, and a space after them.
func (b Bar) wants(i int) int {
	name, _ := b.label(i)
	n := ansi.StringWidth(name)
	reserve := b.Reserve
	if i < len(b.Reserves) {
		reserve = max(reserve, b.Reserves[i])
	}
	if count := ansi.StringWidth(b.Tabs[i].Count); count > 0 || reserve > 0 {
		n += 1 + max(count, reserve)
	}
	if b.Menus {
		n += 1 + ansi.StringWidth(Menu)
	}
	return n + 1
}

// Gutter is the cells before the bar on each of its lines, which ◃ stands in the second of when tabs are hidden on
// the left, as a row's ▸ stands under it.
const Gutter = 2

// screen gives the cells of the screen the line takes: all but the gutter before it.
func (b Bar) screen() int { return max(1, b.Width-Gutter) }

// layout is the bar laid out on its canvas: where each tab's part starts, its cells, the cell between it and the next
// among them, the canvas's width, and the cells scrolled past on its left.
type layout struct {
	at, parts []int
	width     int
	scroll    int
}

// layout lays each tab out whole, at its own width, its number's cell and a space before it and the cell after it
// besides; the canvas is as wide as they are. When it is narrower than the screen, the line under the bar runs on to
// the screen's edge; when it is wider, the view stays where Scroll has it while the selected tab shows whole there, and
// else scrolls so the selected tab is in its middle, or as near as the ends of the canvas let, and whole where the
// screen is wide enough for it.
func (b Bar) layout() layout {
	n, w := len(b.Tabs), b.screen()
	whole := make([]int, n)
	for i := range n {
		whole[i] = 2 + b.wants(i) + 1 // its number's cell and a space, and the cell after it
	}
	l := layout{at: make([]int, n), parts: whole}
	for i, part := range whole {
		l.at[i], l.width = l.width, l.width+part
	}
	if l.width <= w {
		return l
	}
	l.scroll = l.clamp(b.Scroll, w)
	if !l.shows(l.scroll, w, b.Selected) {
		l.scroll = l.clamp(l.at[b.Selected]+l.parts[b.Selected]/2-w/2, w)
		if l.parts[b.Selected] <= w {
			// centered, a tab the view can show whole may still be cut by ▹, or at the canvas's end
			l.scroll = l.reveal(l.scroll, w, b.Selected)
		}
	}
	return l
}

// clamp gives a scroll the canvas allows on a screen so wide: none short of its left end, none past its right.
func (l layout) clamp(scroll, w int) int { return min(max(scroll, 0), max(l.width-w, 0)) }

// room gives the cells the view shows of the canvas from a scroll: the screen's, less the two ▹ takes at its end
// while it hides tabs on the right.
func (l layout) room(scroll, w int) int {
	if scroll+w < l.width {
		return w - 2
	}
	return w
}

// shows says whether the view from a scroll shows a tab whole: its number's cell to its edge, short of ▹.
func (l layout) shows(scroll, w, i int) bool {
	return l.at[i] >= scroll && l.at[i]+l.parts[i] <= scroll+l.room(scroll, w)
}

// reveal gives the scroll nearest the one given from which the view shows a tab whole, as near as the canvas's ends
// let: its first cell at the view's left edge when it is cut there, its last before ▹, or at the screen's edge where
// it is the canvas's last, when it is cut on the right.
func (l layout) reveal(scroll, w, i int) int {
	end := l.at[i] + l.parts[i]
	switch {
	case l.at[i] < scroll:
		return l.clamp(l.at[i], w)
	case end > scroll+l.room(scroll, w):
		return l.clamp(min(end-(w-2), l.width-w), w)
	}
	return scroll
}

// before gives the scroll ◃ moves the view to, which shows whole the nearest tab it hides, or cuts, on its left, and
// whether the selected tab still shows whole from there; when it does not, ◃ is dim, and moves nothing.
func (l layout) before(w, selected int) (int, bool) {
	j := -1
	for i, at := range l.at {
		if at < l.scroll {
			j = i
		}
	}
	if j < 0 {
		return l.scroll, false
	}
	to := l.reveal(l.scroll, w, j)
	return to, l.shows(to, w, selected)
}

// after gives the scroll ▹ moves the view to, which shows whole the nearest tab it hides, or cuts, on its right, and
// whether the selected tab still shows whole from there; when it does not, ▹ is dim, and moves nothing.
func (l layout) after(w, selected int) (int, bool) {
	for i := range l.at {
		if l.at[i]+l.parts[i] > l.scroll+l.room(l.scroll, w) {
			to := l.reveal(l.scroll, w, i)
			return to, l.shows(to, w, selected)
		}
	}
	return l.scroll, false
}

// View draws the bar's three lines as the screen shows them, each after the gutter, as the package says.
func (b Bar) View() string {
	if len(b.Tabs) == 0 {
		return ""
	}
	l, w, n := b.layout(), b.screen(), len(b.Tabs)
	// whether the last tab reaches the screen's edge; short of it, the line under the bar runs on past it
	edge := l.width >= w
	var over, tabs, under strings.Builder
	cell := func(top, middle, bottom string) {
		over.WriteString(top)
		tabs.WriteString(middle)
		under.WriteString(bottom)
	}
	// curve draws a curve under the selected tab's edge, or a half line where no tab is beside it
	mark, _ := style.In(b.Color)
	curve := func(glyph string, end bool) string {
		if end {
			glyph = "╵"
		}
		return mark(glyph)
	}
	for i := range b.Tabs {
		// the tab's first cell holds its number, and the selected tab's corner, edge, and curve under it
		cells := l.parts[i] - 1
		name, end := b.label(i)
		room := cells - 2 - ansi.StringWidth(end)
		head := pane.Pad(" "+name, room, false)
		label := head + end + " "
		dimmed, dimHead := b.dim(label), b.dim(head)
		if lead := " " + name; b.stale(i) && ansi.StringWidth(lead)+1 <= room {
			// the dot stays yellow while the rest of the tab dims
			gap := strings.Repeat(" ", room-ansi.StringWidth(lead)-1)
			head, dimHead = lead+style.Warn(Stale)+gap, b.dim(lead)+style.Warn(Stale)+gap
			label, dimmed = head+end+" ", dimHead+b.dim(end)+" "
		}
		if i == b.Selected {
			cell(mark(cmp.Or(Marker(i, true), "╭")), mark("│"), curve("╯", i == 0))
			cell(mark(top(cells-1)), label, strings.Repeat(" ", cells-1))
			cell(mark("╮"), mark("│"), curve("╰", i == n-1 && edge))
			continue
		}
		// the others' borders are dim, and the line under them in the selected tab's color; the tab before the selected
		// one has no edge after it
		number := b.dim(Marker(i, false))
		if number == "" {
			number = style.Off("─")
		}
		cell(number, " ", mark(strings.Repeat("─", cells)))
		if b.alert(i) {
			// a count that wants attention stays bold and red, wherever the keyboard is
			dimmed = dimHead + end + " "
		}
		cell(style.Off(top(cells-1)), dimmed, "")
		switch {
		case i+1 == b.Selected:
			cell(style.Off("╴"), " ", mark("─"))
		case i == n-1 && edge:
			// the last tab at the edge has no line after it to curve out to: its edge ends in a half line
			cell(style.Off("╮"), style.Off("│"), style.Off("╵"))
		default:
			cell(style.Off("╮"), style.Off("│"), style.Off("╰"))
		}
	}
	// past the last tab, the line under the bar runs on to the screen's edge
	if l.width < w {
		under.WriteString(mark(strings.Repeat("─", w-l.width)))
	}
	// shown gives a line as the view shows it, after the gutter, the triangles on the middle line, each hidden, a
	// blank in its cell, when moving the view would cut the selected tab
	triangle := func(glyph string, ok bool) string {
		if ok {
			return glyph
		}
		return " "
	}
	_, back := l.before(w, b.Selected)
	_, on := l.after(w, b.Selected)
	shown := func(line string, middle bool) string {
		margin, rest := strings.Repeat(" ", Gutter), pane.Pad(pane.Cells(line, l.scroll, l.scroll+w), w, false)
		if middle && l.scroll > 0 {
			margin = strings.Repeat(" ", Gutter-1) + triangle(Before, back)
		}
		if middle && l.scroll+w < l.width {
			rest = pane.Pad(pane.Cells(rest, 0, w-2), w-2, false) + " " + triangle(After, on)
		}
		// the spaces it ends with go, and the colors of what the cut left empty with them
		out := margin + rest
		return ansi.Truncate(out, ansi.StringWidth(strings.TrimRight(ansi.Strip(out), " ")), "")
	}
	bottom := shown(under.String(), false)
	if b.Joined {
		bottom = joinDown(bottom, w+Gutter)
	}
	return shown(over.String(), false) + "\n" + shown(tabs.String(), true) + "\n" + bottom
}

// joinDown turns the last cell of the line, at the column width less one, down into a line under it, in that cell's
// colors: the line under the tabs turns down, ┐; a tab's edge, which goes up from there, runs on down, joining the line
// under that tab, ┤, when one runs up to it, and alone, │, when, as under the selected tab, open to the pane, none
// does. A line that stops short of the column gets a half line, ╷, there, dim.
func joinDown(line string, width int) string {
	plain := []rune(ansi.Strip(line))
	if ansi.StringWidth(string(plain)) < width {
		return pane.Pad(line, width-1, false) + style.Off("╷")
	}
	last, turned := plain[len(plain)-1], '│'
	switch {
	case last == '─':
		turned = '┐'
	case !strings.ContainsRune("╵╰╯│", last):
		return line
	case len(plain) > 1 && plain[len(plain)-2] == '─':
		turned = '┤'
	}
	at := strings.LastIndex(line, string(last))
	return line[:at] + string(turned) + line[at+len(string(last)):]
}

// top gives the line over a tab, so many cells of it, never fewer than two, as no tab is narrower than four: it starts
// with a half line, ╶, apart from the number before it.
func top(cells int) string { return "╶" + strings.Repeat("─", cells-1) }

// alert says whether a tab's count wants attention.
func (b Bar) alert(i int) bool { return i < len(b.Alert) && b.Alert[i] }

// stale says whether the tab at i shows what is stale.
func (b Bar) stale(i int) bool { return i < len(b.Stale) && b.Stale[i] }

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

// MenuAt gives the screen column the selected tab's menu glyph is drawn at on the bar's middle line, or -1 when the bar
// has no menus or the view hides the glyph, as past a triangle.
func (b Bar) MenuAt() int {
	if !b.Menus || len(b.Tabs) == 0 {
		return -1
	}
	l, w := b.layout(), b.screen()
	// the glyph ends the label but its trailing space, after the tab's number cell
	x := Gutter + l.at[b.Selected] + l.parts[b.Selected] - 2 - ansi.StringWidth(Menu) - l.scroll
	first, last := Gutter, Gutter+w-1
	if l.scroll+w < l.width {
		last -= 2
	}
	if x < first || x+ansi.StringWidth(Menu)-1 > last {
		return -1
	}
	return x
}

// hit gives the tab a column of the screen falls on: the one whose part covers it. A triangle, which Update takes
// first, and the margin without one are no tab's: -1.
func (b Bar) hit(x int) int {
	l, w := b.layout(), b.screen()
	if x < Gutter || x >= Gutter+w || x == Gutter+w-1 && l.scroll+w < l.width {
		return -1
	}
	at := x - Gutter + l.scroll
	hit := 0
	for i, start := range l.at {
		if start <= at {
			hit = i
		}
	}
	return hit
}
