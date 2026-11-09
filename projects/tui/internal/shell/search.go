package shell

import (
	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/empowerite/kit/projects/tui/internal/style"
)

// Find is the search's glyph, before what is typed.
const Find = "🔎"

// Idle is what an idle, empty search reads, dim: /search, its key before its name.
const Idle = "/search"

// Search is the search every screen of kittui has, the same on each: on the menu bar's right, 🔎 and what is typed,
// which the screen narrows its view to. / or a click on it gives it focus, and a screen may open with focus there; then
// a printable key is typed, / and = too, as the filter goes, backspace takes one back, and ⌃U clears it; ⏎, ↓, esc,
// and tab give the view focus, the filter kept; and ⌃R recalls the past searches.
type Search struct {
	Query   string
	Focused bool
}

// SearchDid says what a key did to a search with focus.
type SearchDid int

// What a key does to a search with focus: nothing, the key the screen's, as the arrows but ↓ are; it typed, erased, or
// cleared; it gave the view focus, the filter kept; or it asks for the past searches.
const (
	NotSearch SearchDid = iota
	Typed
	Left
	Recalled
)

// Key takes a key while the search has focus, as SearchKey says, and loses focus when the key gives the view focus.
func (s Search) Key(k tea.KeyPressMsg) (Search, SearchDid) {
	q, did := SearchKey(s.Query, k)
	s.Query = q
	if did == Left {
		s.Focused = false
	}
	return s, did
}

// SearchKey gives what is typed after a key pressed in a search with focus, and what the key did: ⏎, ↓, esc, and tab
// leave for the view, ⌃U clears, ⌃R recalls, and the rest types or erases as Type does. Any other key is the screen's.
func SearchKey(query string, k tea.KeyPressMsg) (string, SearchDid) {
	switch k.String() {
	case "enter", "down", "esc", "tab":
		return query, Left
	case "ctrl+u":
		return "", Typed
	case "ctrl+r":
		return query, Recalled
	}
	if q, ok := Type(query, k); ok {
		return q, Typed
	}
	return query, NotSearch
}

// Type gives what is typed after a key: a printable key added, backspace taking the last one back; and whether the key
// typed or erased.
func Type(query string, k tea.KeyPressMsg) (string, bool) {
	switch {
	case k.String() == "backspace":
		if r := []rune(query); len(r) > 0 {
			query = string(r[:len(r)-1])
		}
		return query, true
	case k.Text != "":
		return query + k.Text, true
	}
	return query, false
}

// SearchLine draws a search: 🔎 and what is typed, a cursor after it while the search has the keyboard; or, idle and
// empty, 🔎 and search, dim.
func SearchLine(query string, focused bool) string {
	switch {
	case focused:
		return Find + " " + query + "▏"
	case query == "":
		return Find + " " + style.Off(Idle)
	}
	return Find + " " + query
}

// SearchAt gives the column a search, drawn as line, starts at on the title line of a screen so wide: right-aligned a
// cell short of the edge, and as wide as an idle one at least, so it stays where it stands as it takes the keyboard,
// and grows leftward only once what is typed needs the room.
func SearchAt(width int, line string) int {
	return max(0, width-1-max(ansi.StringWidth(line), ansi.StringWidth(Find+" "+Idle)))
}
