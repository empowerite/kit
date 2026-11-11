// The keys a table's controls answer while they have the keyboard: the rows', the preset line's, and those of a stop of
// the heading over the groups, as the keys bar names them; and what the control line does with a key.
package table

import (
	"slices"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/empowerite/kit/projects/tui/internal/grid"
	"github.com/empowerite/kit/projects/tui/internal/shell"
)

// TabsHint is what the key hints say esc does where it gives the tab bar the keyboard, from the rows or the control
// line: it goes back to the tabs, whichever is selected.
const TabsHint = "tabs"

// GroupHeadings gives how many headings of groups the cursor can stand on in the host's rows.
func GroupHeadings(h Host) int {
	n, all := 0, h.Rows()
	for _, r := range all[min(First(all), len(all)):] {
		if grid.Heads(r) {
			n++
		}
	}
	return n
}

// RowKeys gives the segment of the host's rows while they have the keyboard: esc gives the bar the keyboard; up and
// down move, a page at a time too while the rows are more than the pane shows, and home and end to the first and the
// last, unnamed until a test shows the keys arrive in Terminal.app and iTerm; tab and shift-tab scroll the columns
// while the view hides some, as grid.Scroll says; [ and ] move between the groups' headings where there are more than
// one; space selects a row, shift with up or down selects as it moves, and ctrl+a
// every row; right shows what the node under the cursor holds, a row's lines or a group's rows, and left hides it,
// the one that acts named. ⌥ with an arrow is named only where alt says the terminal tells it apart, as the kitty
// keyboard protocol does: a macOS terminal with its default settings sends it as another key, or as the arrow alone.
func RowKeys(h Host, alt bool) shell.Segment {
	seg := shell.Segment{Takes: []string{"left", "right", "alt+left", "alt+right", "home", "end"},
		Bindings: []shell.Binding{shell.Bind("esc", TabsHint), shell.Bind("↑↓", "move", "up", "down")}}
	if len(ScrollLines(h)) > RowsHeight(h) {
		seg.Bindings = append(seg.Bindings, shell.Bind("⇞⇟", "page", "pgup", "pgdown"))
	}
	if _, before, after := Scrolled(h); before || after {
		seg.Bindings = append(seg.Bindings, shell.Bind("⇥·⇧⇥", "columns", "tab", "shift+tab"))
	}
	if GroupHeadings(h) > 1 {
		// [ and ] move between the groups' headings, { and } to the first and the last, unnamed
		seg.Bindings = append(seg.Bindings, shell.Bind("[]", "group", "[", "]"))
		seg.Takes = append(seg.Takes, "{", "}")
	}
	seg.Bindings = append(seg.Bindings, shell.Bind("␣", "select", "space"),
		shell.Bind("⇧↑↓", "select", "shift+up", "shift+down"), shell.Bind("⌃A", "all", "ctrl+a"))
	t, rows := h.Table(), h.Rows()
	if t.Cursor >= len(rows) {
		return seg
	}
	every := grid.Every(rows)
	at := slices.IndexFunc(every, func(r grid.Row) bool { return r.Key == rows[t.Cursor].Key })
	switch {
	case Opens(h, every, at) && t.Open.Unfolds(rows[t.Cursor]):
		seg.Bindings = append(seg.Bindings, shell.Bind("←", "hide", "left"))
	case Opens(h, every, at):
		seg.Bindings = append(seg.Bindings, shell.Bind("→", "show", "right"))
	case grid.Parent(rows, t.Cursor) >= First(rows):
		seg.Bindings = append(seg.Bindings, shell.Bind("←", "group", "left"))
	}
	if grid.Heads(rows[t.Cursor]) && alt {
		seg.Bindings = append(seg.Bindings, shell.Bind("⌥←→", "a level", "alt+left", "alt+right"))
	}
	return seg
}

// PresetKeys gives the segment of the preset line while it has the keyboard: up or esc gives the bar the keyboard, and
// down the rows, when the table has some; shift-tab goes back to the heading over the groups, where the rows group;
// enter or space drops the presets down. While a name is typed for Save As… or Rename…, what is typed is the name,
// = : and ; apart, which the menus take; enter saves under it, or renames, and esc stops.
func PresetKeys(h Host) shell.Segment {
	if t := h.Table(); t.SavingAs {
		s := shell.Segment{Types: true, Takes: []string{"backspace"}, Passes: []string{"=", ":", ";"}}
		does := "save"
		if t.Renaming != "" {
			does = "rename"
		}
		s.Bindings = append(s.Bindings, shell.Bind("⏎", does, "enter"), shell.Bind("esc", "cancel"))
		return s.With(shell.Bind("", "type the preset's name"))
	}
	// up and esc give the tab bar the keyboard, named as the components spec names it
	var s shell.Segment
	s.Bindings = append(s.Bindings, shell.Bind("↑", TabsHint, "up"), shell.Bind("esc", TabsHint))
	if len(h.Rows()) > 0 {
		s.Bindings = append(s.Bindings, shell.Bind("↓", "rows", "down"))
	}
	if len(Stops(h)) > 0 {
		s.Bindings = append(s.Bindings, shell.Bind("⇧⇥", "groups", "shift+tab"))
	}
	return s.With(shell.Bind("⏎", "presets", "enter", "space"))
}

// HeadingKeys gives the segment of a stop of the heading over the groups while it has the keyboard: up or esc gives
// the bar the keyboard, down the rows, tab and shift-tab move where there is another stop, or the preset picker, as
// presets says, to move to, left and right close and open the stop's level, and enter drops a name's column menu.
func HeadingKeys(h Host, stop Stop, presets bool) shell.Segment {
	s := shell.Segment{Bindings: []shell.Binding{shell.Bind("↑", TabsHint, "up"), shell.Bind("esc", TabsHint)}}
	if len(h.Rows()) > 0 {
		s.Bindings = append(s.Bindings, shell.Bind("↓", "rows", "down"))
	}
	if len(Stops(h)) > 1 || presets {
		s.Bindings = append(s.Bindings, shell.Bind("⇥·⇧⇥", "move", "tab", "shift+tab"))
	}
	s.Bindings = append(s.Bindings, shell.Bind("←→", "fold", "left", "right"))
	if stop.Column != "" {
		return s.With(shell.Bind("⏎", "menu", "enter"))
	}
	return s
}

// Keyboard is a host whose control line takes the keyboard: the stop of the heading over the groups it stands on, -1
// the preset picker, and setting it; the line giving the keyboard up, to the bar; the column menu dropped around a
// column's name in the heading; and whether the table has a preset line.
type Keyboard interface {
	Host
	Stop() int
	SetStop(int)
	LeaveControls()
	OpenFieldMenu(column string)
	HasPresets() bool
}

// HeadingKey takes a key while the keyboard stands on a stop of the heading over the groups, and says whether it took
// it: tab and shift-tab move between the stops, and tab past the last to the preset picker; right opens the stop's
// level and left closes it; enter on a name drops its column's menu, and on the rows' opens or closes their level, as
// space does on any stop; up or esc gives the bar the keyboard, and down the rows, when the table has some.
func HeadingKey(h Keyboard, k tea.KeyPressMsg) bool {
	stops := Stops(h)
	var stop Stop
	if at := h.Stop(); at >= 0 && at < len(stops) {
		stop = stops[at]
	}
	switch k.String() {
	case "shift+tab":
		h.SetStop(max(0, h.Stop()-1))
	case "tab":
		if h.Stop() < len(stops)-1 {
			h.SetStop(h.Stop() + 1)
		} else if h.HasPresets() {
			h.SetStop(-1)
		}
	case "right", "left":
		// right opens a level closed, and left closes one open; either leaves it as it is otherwise
		if open := h.Table().Open.Shows(stop.Level); open == (k.String() == "left") {
			ToggleLevel(h, stop.Level)
		}
	case "enter":
		if stop.Column != "" {
			h.OpenFieldMenu(stop.Column)
			return true
		}
		ToggleLevel(h, stop.Level)
	case "space":
		ToggleLevel(h, stop.Level)
	case "esc", "up":
		h.LeaveControls()
	case "down":
		if len(h.Rows()) > 0 {
			h.LeaveControls()
			Enter(h)
		}
	default:
		return false
	}
	return true
}

// PresetDo is what a key on the preset line asks of the screen, beyond what PresetKey does itself: nothing, to save
// the panel under the name typed, to rename the preset to it, or to drop the presets down.
type PresetDo int

// What a key on the preset line asks.
const (
	DoNothing PresetDo = iota
	DoSave
	DoRename
	DoDrop
)

// PresetKey takes a key while the keyboard is on the preset line, and says whether it took it, and what it asks of the
// screen: up and esc give the bar the keyboard, down the rows, when the table has some, and shift-tab the heading
// over the groups' last stop; enter and space drop the presets down. While a name is typed for Save As… or Rename…,
// what is typed is the name, = : and ; apart, which the menus take; enter saves under it, or renames, esc stops, and
// up and down stop too, and move as ever.
func PresetKey(h Keyboard, k tea.KeyPressMsg) (PresetDo, bool) {
	t, key := h.Table(), k.String()
	if t.SavingAs {
		switch {
		case key == "up" || key == "down":
			t.SavingAs, t.Naming, t.Renaming = false, "", ""
		case key == "esc":
			t.SavingAs, t.Naming, t.Renaming = false, "", ""
			return DoNothing, true
		case key == "enter" && t.Renaming != "":
			return DoRename, true
		case key == "enter":
			return DoSave, true
		case key == "backspace" || k.Text != "" && !strings.ContainsAny(k.Text, "=:;"):
			t.Naming, _ = shell.Type(t.Naming, k)
			return DoNothing, true
		default:
			return DoNothing, false
		}
	}
	switch key {
	case "shift+tab":
		n := len(Stops(h))
		if n == 0 {
			return DoNothing, false
		}
		h.SetStop(n - 1)
	case "esc", "up":
		h.LeaveControls()
	case "down":
		if len(h.Rows()) > 0 {
			h.LeaveControls()
			Enter(h)
		}
	case "enter", "space":
		return DoDrop, true
	default:
		return DoNothing, false
	}
	return DoNothing, true
}
