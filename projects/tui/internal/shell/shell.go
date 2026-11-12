// Package shell is what every screen of kittui shares around its own: the title line, after =, which drops the app's
// menu down; the app's menu, the same on every screen, which moves between kit's screens and quits; the box every
// menu is drawn in, hanging from the glyph that drops it down.
package shell

import (
	"image"
	"math"
	"slices"
	"strings"
	"unicode/utf8"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/empowerite/kit/projects/tui/internal/pane"
	"github.com/empowerite/kit/projects/tui/internal/style"
)

// App is the glyph before every screen's title, which drops the app's menu down: =, the key that drops it down too.
const App = "="

// Top is the screen line the title is on, and the app's menu's box starts on; At is App's column on it, after the
// gutter every line of a screen keeps before it, as the tab bar does.
const (
	Top = 0
	At  = 2
)

// Screen names one of kit's screens, which the app's menu moves between.
type Screen string

// kit's screens: the guides, the welcome's guide, the help's palette of verbs, and the tutorials' cards; and the
// apps: flow, the picture of where the work stands, every stage kit flow draws, and those built around one entity
// each: issues, changes, repositories, delivery's releases, people, trust's evidence, the platform's services, and the
// packages kit delivers.
const (
	Welcome      Screen = "welcome"
	Help         Screen = "help"
	Tutorials    Screen = "tutorials"
	Flow         Screen = "flow"
	Issues       Screen = "issues"
	Changes      Screen = "changes"
	Repositories Screen = "repositories"
	Delivery     Screen = "delivery"
	People       Screen = "people"
	Trust        Screen = "trust"
	Platform     Screen = "platform"
	Packages     Screen = "packages"
	// Manage is the workspaces' screen, which the workspace menu's Manage… opens, and no line of the app's menu does.
	Manage Screen = "workspaces"
)

// The glyphs beside the app's menu's items: the welcome, the help, the tutorials; the flow's compass, and each other
// app's; the settings, and goodbye. A repository is one book, as a workspace, 📚, is many.
const (
	Home      = "🏠"
	Asking    = "❓"
	Learn     = "🎓"
	Compass   = "🧭"
	Ticket    = "🎫"
	Shuffle   = "🔀"
	Book      = "📘"
	Truck     = "🚚"
	Busts     = "👥"
	Sealed    = "🔏"
	Toolbox   = "🧰"
	Puzzle    = "🧩"
	Configure = "🔧"
	Leave     = "👋"
)

// Item is one line of a menu: a glyph, which may be none; the name it is chosen by; and the keys that do the same with
// the menu closed, as the keys bar names keys, none when no key does: the menu shows them at its right, dim, where the
// keys bar would have to name one key apiece, a letter in upper case, R as ⌃A, as a macOS menu draws it, though the
// press is the letter alone. A line with no name is the rule between groups of items.
type Item struct{ Glyph, Name, Keys string }

// screenItem is one of the app's menu's screens, with the item it is chosen by.
type screenItem struct {
	item   Item
	screen Screen
}

// sections are the app's menu's screens, in its order, a rule between each group: the guides, which come with kit
// whatever the work is, then the apps: the flow, the picture of the work from backlog to releases, and each other a
// picture of the work around one entity.
var sections = [][]screenItem{
	{{Item{Glyph: Home, Name: "Welcome"}, Welcome}, {Item{Glyph: Learn, Name: "Tutorials"}, Tutorials},
		{Item{Glyph: Asking, Name: "Help"}, Help}},
	{{Item{Glyph: Compass, Name: "Flow"}, Flow}, {Item{Glyph: Ticket, Name: "Issues"}, Issues},
		{Item{Glyph: Shuffle, Name: "Changes"}, Changes}, {Item{Glyph: Book, Name: "Repositories"}, Repositories},
		{Item{Glyph: Truck, Name: "Delivery"}, Delivery},
		{Item{Glyph: Busts, Name: "People"}, People}, {Item{Glyph: Sealed, Name: "Trust"}, Trust},
		{Item{Glyph: Toolbox, Name: "Platform"}, Platform}, {Item{Glyph: Puzzle, Name: "Packages"}, Packages}},
}

// screens are the app's menu's screens, in its order, without the rules.
var screens = slices.Concat(sections...)

// Menu is the app's menu on a screen: the screen it is on, whose item is dim; the screen's own items, which come
// after kit's screens, and the names of those that do nothing now, dim; the name of the screen's crumb, the place a
// link led from, which the title draws after the workspace, the way back the menu's screens are not, none without one;
// and, as a dropdown, whether it is open, and then has the keyboard, and the item under the keyboard.
type Menu struct {
	Here  Screen
	Own   []Item
	Dim   []string
	Crumb string
	Dropdown
}

// Choice is what a choice from the app's menu asks of its screen: a screen to open in its place, kittui to quit, or
// one of the screen's own items, by name; and, for Settings…, the screen cell the menu drew its name from, which the
// Settings dialog's title then stands on. The zero Choice asks nothing.
type Choice struct {
	Screen Screen
	Quit   bool
	Own    string
	From   *image.Point
}

// Switch gives the command that asks for the screen the choice names, from where the menu drew it.
func (c Choice) Switch() tea.Cmd {
	return func() tea.Msg { return SwitchMsg{Screen: c.Screen, From: c.From} }
}

// SwitchMsg asks kittui to show another of kit's screens in place of the one shown, each kept as it was left: from the
// app's menu, with no crumb; with Crumb, as a link leads, the one shown left as the crumb of the one asked for, as
// crumb.go says. From is the screen cell the menu drew the line chosen from, for the Settings dialog, none else.
type SwitchMsg struct {
	Screen Screen
	Crumb  bool
	From   *image.Point
}

// JumpMsg asks kittui to show an app with a row of one of its tabs under the cursor, as a link in another app leads:
// the app, and where in it, the tab's stage as the app knows it, a slash, and the row's key. kittui shows the app, as
// SwitchMsg asks, then gives it the message, and the app selects the tab and the row, once the row has loaded.
type JumpMsg struct {
	Screen Screen
	To     string
}

// Quit is the key that quits kittui on every screen, the app's menu open or not, as its Quit line says; ctrl-c quits
// too, as an interrupt. No modifier on esc quits: a macOS terminal with its default settings sends ⌥esc, ⌃esc, and
// ⇧esc as a plain esc, which only cancels or goes back, and keeps ⌘ for itself. Raw mode turns flow control off, so
// ctrl-q reaches kittui.
const Quit = "ctrl+q"

// Items gives the menu's lines, top to bottom: kit's screens, a rule after each section of them; the screen's own
// items, when it has any, and the settings; a rule; and quit, which ⌃Q does on every screen.
func (m Menu) Items() []Item {
	var items []Item
	for _, section := range sections {
		for _, s := range section {
			items = append(items, s.item)
		}
		items = append(items, Item{})
	}
	items = append(items, m.Own...)
	return append(items, settingsItem, Item{}, Item{Glyph: Leave, Name: "Quit", Keys: "⌃Q"})
}

// settingsItem is the menu's line that opens the Settings dialog.
var settingsItem = Item{Glyph: Configure, Name: "Settings…"}

// enabled says whether the menu's line k does something: the screen's own items but those it dims, a screen other
// than the one it is on, the settings, and quit; the rules never do.
func (m Menu) enabled(k int) bool {
	item := m.Items()[k]
	if slices.Contains(m.Own, item) && item.Name != "" {
		return !slices.Contains(m.Dim, item.Name)
	}
	if item.Name == "Quit" || item == settingsItem {
		return true
	}
	for _, s := range screens {
		if s.item == item {
			return s.screen != m.Here
		}
	}
	return false
}

// choice gives what the menu's line k asks, the zero Choice for one not enabled, the screen's rows as wide as room.
func (m Menu) choice(k, room int) Choice {
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
	if item == settingsItem {
		// the line's name starts after the box's edge, a cell, the glyph's two cells, and a cell
		_, left := m.box(room)
		return Choice{Screen: SettingsScreen, From: &image.Point{X: left + 5, Y: Top + 1 + k}}
	}
	return Choice{Own: item.Name}
}

// Opened gives the menu dropped down, its first item that does something under the keyboard.
func (m Menu) Opened() Menu {
	m.Dropdown = m.Dropdown.Opened(m.Items(), m.enabled)
	return m
}

// Key takes a key while the menu is open, as a dropdown does, = closing it as esc does; it gives the menu, closed once
// a choice is made, and the choice. The menu stands as wide as it is, as no screen is too narrow for it.
func (m Menu) Key(k tea.KeyPressMsg) (Menu, Choice) {
	var at int
	m.Dropdown, at = m.Dropdown.Key(k, m.Items(), m.enabled, "=")
	return m.chose(at, math.MaxInt)
}

// Click takes a left click while the menu is open, the screen's rows as wide as room, as a dropdown does; it gives the
// menu and the choice.
func (m Menu) Click(x, y, room int) (Menu, Choice) {
	box, left := m.box(room)
	var at int
	m.Dropdown, at = m.Dropdown.Click(x, y, Top, left, ansi.StringWidth(box[0]), m.Items(), m.enabled)
	return m.chose(at, room)
}

// chose gives the menu and what its line k asks, the screen's rows as wide as room: Settings… leaves the menu open
// with the line under the keyboard, as the Settings dialog opens over it and esc closes the dialog back to it.
func (m Menu) chose(k, room int) (Menu, Choice) {
	c := m.choice(k, room)
	if c.Screen == SettingsScreen {
		m.Open, m.At = true, k
	}
	return m, c
}

// OnApp says whether a click at x, y is on App before the title, the cells either side of it, or the screen's name after
// it, the label Title is given, so a click anywhere on "= kit flow" drops the app's menu down.
func OnApp(x, y int, label string) bool {
	return y == Top && x < At+ansi.StringWidth(App)+1+ansi.StringWidth(label)
}

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

// Press gives the key press a menu line's keys stand for, as Bubble Tea names it: ⌃ and a letter is ctrl with the
// letter, and a single character is itself, a letter in lower case; "" for keys that are neither.
func Press(keys string) string {
	if letter, ok := strings.CutPrefix(keys, "⌃"); ok && utf8.RuneCountInString(letter) == 1 {
		return "ctrl+" + strings.ToLower(letter)
	}
	if utf8.RuneCountInString(keys) == 1 {
		return strings.ToLower(keys)
	}
	return ""
}

// Shortcut gives the line that acts whose keys a press stands for, or -1. A line's keys do the same with its menu open
// as with it closed, so a menu takes them before a letter moves to the next line that starts with it.
func Shortcut(items []Item, enabled func(int) bool, press string) int {
	for k, item := range items {
		if item.Keys != "" && Press(item.Keys) == press && enabled(k) {
			return k
		}
	}
	return -1
}

// Box gives a menu's lines, a rounded box in the menu's color around its items, each after the room for a glyph, the
// one under the keyboard, at, inverse and the ones not enabled dim, a dim rule between groups, a cell short of either
// side; and the screen column its left edge is drawn at. The box hangs from its glyph, mark, drawn at the column
// glyph: two cells before it, or the screen's first, so the glyph starts at the top edge's third cell; when the box
// would run past room that way, the column that puts the glyph two cells before the right corner instead, or nearer
// when room ends there; and against room's right when no glyph is drawn, glyph -1.
func Box(items []Item, enabled func(int) bool, at int, mark string, glyph, room int) (lines []string, left int) {
	lines, left, _ = BoxIn(items, enabled, at, mark, glyph, room, len(items)+2)
	return lines, left
}

// BoxIn gives a menu's lines as Box does, in no more than rows lines, and the first item they show. A menu taller than
// that shows the items around the one under the keyboard, as many as fit, and a dim ▴ on its top edge while items
// are above them, and a ▾ on its bottom edge while items are below.
func BoxIn(items []Item, enabled func(int) bool, at int, mark string, glyph, room, rows int) (lines []string, left,
	first int) {
	return boxIn(items, enabled, at, mark, glyph, room, rows, "")
}

// BoxColored gives a menu's lines as Box does, its edges and the line under the keyboard in color, a 256-color number,
// as style.MenuIn draws them, for a menu that belongs to one group.
func BoxColored(items []Item, enabled func(int) bool, at int, mark string, glyph, room int, color string) (
	lines []string, left int) {
	lines, left, _ = boxIn(items, enabled, at, mark, glyph, room, len(items)+2, color)
	return lines, left
}

// boxIn gives a menu's lines as BoxIn does, its edges and the line under the keyboard in color, or the menu's own with
// none.
func boxIn(items []Item, enabled func(int) bool, at int, mark string, glyph, room, rows int, color string) (
	lines []string, left, first int) {
	edge, under := style.MenuIn(color)
	shown := len(items)
	if rows-2 < shown {
		shown = max(1, rows-2)
		first = min(max(0, at-shown/2), len(items)-shown)
	}
	names, keys := make([]string, len(items)), make([]string, len(items))
	for k, item := range items {
		names[k], keys[k] = item.Name, item.Keys
	}
	// a line is a cell, the glyph's two, a cell, the name, and a cell; then, where any line has keys, two cells and the
	// widest keys, every line's keys flush right
	named, keyed := pane.Width(names), pane.Width(keys)
	inner := named + 5
	if keyed > 0 {
		inner += 2 + keyed
	}
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
	more := func(line []rune, glyph rune) string {
		// the mark of more items sits two cells before the right corner, dim, in the edge's color either side; an edge
		// is never shorter than seven runes, the glyph's two cells one rune
		k := len(line) - 3
		return edge(string(line[:k])) + style.Off(string(glyph)) + edge(string(line[k+1:]))
	}
	if first > 0 {
		lines = append(lines, more(top, '▴'))
	} else {
		lines = append(lines, edge(string(top)))
	}
	for k, item := range items {
		if k < first || k >= first+shown {
			continue
		}
		text := " " + pane.Pad(item.Glyph, 2, false) + " " + pane.Pad(item.Name, named, false)
		shortcut := ""
		if keyed > 0 {
			shortcut = "  " + pane.Pad(item.Keys, keyed, true)
		}
		switch {
		case item.Name == "":
			lines = append(lines, edge("│")+" "+style.Off(strings.Repeat("─", inner-2))+" "+edge("│"))
			continue
		case !enabled(k):
			// dim, and never under the keyboard, even when what it acts on changed under an open menu
			text = style.Off(text + shortcut + " ")
		case k == at:
			text = under(text + shortcut + " ")
		case item.Keys != "":
			text += style.Off(shortcut) + " "
		default:
			text += shortcut + " "
		}
		lines = append(lines, edge("│")+text+edge("│"))
	}
	bottom := []rune("╰" + strings.Repeat("─", inner) + "╯")
	if first+shown < len(items) {
		lines = append(lines, more(bottom, '▾'))
	} else {
		lines = append(lines, edge(string(bottom)))
	}
	return lines, left, first
}

// Titled gives a box's top edge, so wide, in the menu's color, its title at the left after a dash, clipped to the
// room, as a dialog's is.
func Titled(title string, width int) string { return TitledColored(title, width, "") }

// TitledColored gives a box's top edge as Titled does, in color, a 256-color number, as BoxColored draws its edges.
func TitledColored(title string, width int, color string) string {
	edge, _ := style.MenuIn(color)
	inner := width - 2
	title = ansi.Truncate(title, inner-3, "…")
	return edge("╭─ " + title + " " + strings.Repeat("─", max(0, inner-3-ansi.StringWidth(title))) + "╮")
}

// Dialog gives a dialog's lines: a rounded box in the menu's color, its title on the top edge, around its lines, each a
// cell in from either side; no wider than width, a line too long clipped. Its keys are the keys bar's alone, so the
// bottom edge is a plain one.
func Dialog(title string, body []string, width int) []string {
	inner := max(pane.Width(body)+2, ansi.StringWidth(title)+4)
	inner = max(4, min(inner, width-2))
	lines := []string{Titled(title, inner+2)}
	for _, l := range body {
		lines = append(lines, style.Menu("│")+" "+pane.Pad(ansi.Truncate(l, inner-2, "…"), inner-2, false)+" "+
			style.Menu("│"))
	}
	return append(lines, style.Menu("╰"+strings.Repeat("─", inner)+"╯"))
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
