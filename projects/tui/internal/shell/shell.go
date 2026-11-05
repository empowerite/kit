// Package shell is what every screen of kittui shares around its own: the title line, after ≡, which drops the app's
// menu down; the app's menu, the same on every screen, which moves between kit's screens and quits; and the box every
// menu is drawn in, hanging from the glyph that drops it down.
package shell

import (
	"slices"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/empowerite/kit/projects/tui/internal/pane"
	"github.com/empowerite/kit/projects/tui/internal/style"
)

// App is the glyph before every screen's title, which drops the app's menu down: three bars, a hamburger.
const App = "≡"

// Top is the screen line the title is on, and the app's menu's box starts on; At is App's column on it, after the
// gutter every line of a screen keeps before it, as the tab bar does.
const (
	Top = 0
	At  = 2
)

// Screen names one of kit's screens, which the app's menu moves between.
type Screen string

// kit's screens: the picture of where the work stands, the welcome's guide, the help's palette of verbs, and the
// tutorials, which kit does not have yet.
const (
	Flow      Screen = "flow"
	Welcome   Screen = "welcome"
	Help      Screen = "help"
	Tutorials Screen = "tutorials"
)

// The glyphs beside the app's menu's items: the picture, the welcome, the help, the tutorials, the settings, and
// goodbye.
const (
	Map      = "🧭"
	Home     = "🏠"
	Asking   = "❓"
	Learn    = "🎓"
	Settings = "🔧"
	Leave    = "👋"
)

// Item is one line of a menu: a glyph, which may be none, and the name it is chosen by; a line with no name is the
// rule between groups of items.
type Item struct{ Glyph, Name string }

// screens are the app's menu's screens, in its order, with the item each is chosen by.
var screens = []struct {
	item   Item
	screen Screen
}{
	{Item{Home, "Welcome"}, Welcome}, {Item{Learn, "Tutorials"}, Tutorials}, {Item{Asking, "Help"}, Help},
	{Item{Map, "Flow"}, Flow},
}

// Menu is the app's menu on a screen: the screen it is on, whose item is dim; the screen's own items, which come
// first; whether it is open, and then has the keyboard; and the item under the keyboard.
type Menu struct {
	Here Screen
	Own  []Item
	Open bool
	At   int
}

// Choice is what a choice from the app's menu asks of its screen: a screen to open in its place, kittui to quit, or
// one of the screen's own items, by name. The zero Choice asks nothing.
type Choice struct {
	Screen Screen
	Quit   bool
	Own    string
}

// SwitchMsg asks kittui to show another of kit's screens in place of the one shown, each kept as it was left.
type SwitchMsg struct{ Screen Screen }

// Switch gives the command that asks for the screen.
func Switch(to Screen) tea.Cmd { return func() tea.Msg { return SwitchMsg{to} } }

// Keys is the keys bar while a menu is open.
const Keys = "↑↓ move · ⏎ choose · esc close"

// Items gives the menu's lines, top to bottom: the screen's own items and a rule, when it has any; kit's screens; a
// rule; the settings, which kittui does not have yet; a rule; and quit.
func (m Menu) Items() []Item {
	var items []Item
	if len(m.Own) > 0 {
		items = append(slices.Clone(m.Own), Item{})
	}
	for _, s := range screens {
		items = append(items, s.item)
	}
	return append(items, Item{}, Item{Settings, "Settings…"}, Item{}, Item{Leave, "Quit"})
}

// enabled says whether the menu's line k does something: the screen's own items, a screen other than the one it is
// on but the tutorials, and quit; the settings and the rules never do.
func (m Menu) enabled(k int) bool {
	item := m.Items()[k]
	if item.Name == "Quit" || slices.Contains(m.Own, item) && item.Name != "" {
		return true
	}
	for _, s := range screens {
		if s.item == item {
			return s.screen != m.Here && s.screen != Tutorials
		}
	}
	return false
}

// choice gives what the menu's line k asks, the zero Choice for one not enabled.
func (m Menu) choice(k int) Choice {
	if k < 0 || !m.enabled(k) {
		return Choice{}
	}
	item := m.Items()[k]
	for _, s := range screens {
		if s.item == item {
			return Choice{Screen: s.screen}
		}
	}
	if item.Name == "Quit" {
		return Choice{Quit: true}
	}
	return Choice{Own: item.Name}
}

// Opened gives the menu dropped down, its first item that does something under the keyboard.
func (m Menu) Opened() Menu {
	m.Open, m.At = true, Move(m.Items(), m.enabled, -1, 1)
	return m
}

// Key takes a key while the menu is open: up and down move, a letter moves to the next item that starts with it, enter
// and space choose, and esc or = closes it; it gives the menu, closed once a choice is made, and the choice.
func (m Menu) Key(k tea.KeyPressMsg) (Menu, Choice) {
	items := m.Items()
	switch k.String() {
	case "up":
		m.At = Move(items, m.enabled, m.At, -1)
	case "down":
		m.At = Move(items, m.enabled, m.At, 1)
	case "enter", "space":
		if c := m.choice(m.At); c != (Choice{}) {
			m.Open = false
			return m, c
		}
	case "esc", "=":
		m.Open = false
	default:
		m.At = NextItem(items, m.enabled, m.At, k.Text)
	}
	return m, Choice{}
}

// Click takes a left click while the menu is open, the screen's rows as wide as room: on an item, it chooses it, and
// on one not enabled, or a rule, it does nothing; anywhere else, it closes the menu.
func (m Menu) Click(x, y, room int) (Menu, Choice) {
	box, left := m.box(room)
	k := y - Top - 1
	if x > left && x < left+ansi.StringWidth(box[0])-1 && k >= 0 && k < len(m.Items()) {
		if c := m.choice(k); c != (Choice{}) {
			m.Open = false
			return m, c
		}
		return m, Choice{}
	}
	m.Open = false
	return m, Choice{}
}

// OnApp says whether a click at x, y is on App before the title, or the cell either side of it.
func OnApp(x, y int) bool { return y == Top && x <= At+ansi.StringWidth(App) }

// box gives the open menu's lines and the column its left edge is drawn at, the screen's rows as wide as room.
func (m Menu) box(room int) ([]string, int) {
	return Box(m.Items(), m.enabled, m.At, App, At, room)
}

// Overlay draws the open menu over the screen's lines, from the title's, the rows as wide as room.
func (m Menu) Overlay(lines []string, room int) []string {
	if !m.Open {
		return lines
	}
	box, left := m.box(room)
	return Over(lines, box, Top, left)
}

// Title gives a screen's title line: the gutter, App, tinted, or in the menu's color while the menu is open, then the
// label.
func Title(label string, open bool) string {
	glyph := style.Tint(App)
	if open {
		glyph = style.Menu(App)
	}
	return strings.Repeat(" ", At) + glyph + " " + label
}

// Move gives the item from k on, by step, that is enabled, stopping at the first and the last; k itself when none
// is.
func Move(items []Item, enabled func(int) bool, k, step int) int {
	for j := k + step; j >= 0 && j < len(items); j += step {
		if enabled(j) {
			return j
		}
	}
	return k
}

// NextItem gives the enabled item after the one at k, round from the last to the first, that starts with the letter
// typed, regardless of case; k itself when none does, or nothing was typed.
func NextItem(items []Item, enabled func(int) bool, k int, typed string) int {
	if typed == "" {
		return k
	}
	for step := 1; step <= len(items); step++ {
		j := (k + step) % len(items)
		if enabled(j) && strings.HasPrefix(strings.ToLower(items[j].Name), strings.ToLower(typed)) {
			return j
		}
	}
	return k
}

// Box gives a menu's lines, a rounded box in the menu's color around its items, each after the room for a glyph, the
// one under the keyboard, at, inverse and the ones not enabled dim, a dim rule between groups, a cell short of either
// side; and the screen column its left edge is drawn at. The box hangs from its glyph, mark, drawn at the column
// glyph: two cells before it, or the screen's first, so the glyph starts at the top edge's third cell; when the box
// would run past room that way, the column that puts the glyph two cells before the right corner instead, or nearer
// when room ends there; and against room's right when no glyph is drawn, glyph -1.
func Box(items []Item, enabled func(int) bool, at int, mark string, glyph, room int) (lines []string, left int) {
	names := make([]string, len(items))
	for k, item := range items {
		names[k] = item.Name
	}
	inner := pane.Width(names) + 5
	width, cells := inner+2, ansi.StringWidth(mark)
	switch {
	case glyph < 0:
		left = room - width
	case glyph-2+width > room:
		left = min(glyph+cells+2, room) - width
	default:
		left = glyph - 2
	}
	left = max(0, left)
	top := []rune("╭" + strings.Repeat("─", inner) + "╮")
	if c := glyph - left; glyph >= 0 && c >= 0 && c+cells <= width {
		// the glyph takes its cells of the top edge, the edge either side of it drawn half its length; every cell of
		// the edge is a rune of its own
		if c > 1 {
			top[c-1] = '╴'
		}
		if c+cells < width-1 {
			top[c+cells] = '╶'
		}
		top = slices.Concat(top[:c], []rune(mark), top[c+cells:])
	}
	lines = append(lines, style.Menu(string(top)))
	for k, item := range items {
		text := pane.Pad(" "+pane.Pad(item.Glyph, 2, false)+" "+item.Name+" ", inner, false)
		switch {
		case item.Name == "":
			lines = append(lines, style.Menu("│")+" "+style.Off(strings.Repeat("─", inner-2))+" "+style.Menu("│"))
			continue
		case !enabled(k):
			// dim, and never under the keyboard, even when what it acts on changed under an open menu
			text = style.Off(text)
		case k == at:
			text = style.Cursor(text)
		}
		lines = append(lines, style.Menu("│")+text+style.Menu("│"))
	}
	lines = append(lines, style.Menu("╰"+strings.Repeat("─", inner)+"╯"))
	return lines, left
}

// Over draws a box over the screen's lines, from the line top and the column left; what the box covers is hidden, and
// what is beside it stays in its columns, a two-cell glyph the box cuts in half giving way to a space.
func Over(lines, box []string, top, left int) []string {
	for k, b := range box {
		if i := top + k; i < len(lines) {
			lines[i] = Splice(lines[i], b, left)
		}
	}
	return lines
}

// Splice draws box over line from the column left, keeping every cell of line outside it in its column: a two-cell
// glyph the box's edge cuts in half gives way to a space.
func Splice(line, box string, left int) string {
	right := left + ansi.StringWidth(box)
	line = pane.Pad(line, right, false)
	return pane.Cells(line, 0, left) + box + pane.Cells(line, right, ansi.StringWidth(line))
}
