package shell

import (
	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/empowerite/kit/projects/tui/internal/style"
)

// Find is the search's glyph, before what is typed.
const Find = "🔎"

// Search is the search every screen of kittui has, the same on each: on the title line's right, 🔎 and what is
// typed, which the screen narrows what it shows to. / or a click on it gives it the keyboard; then a printable key is
// typed, / and = too, and backspace takes one back; esc, tab, and enter leave it, what is typed kept.
type Search struct {
	Query   string
	Focused bool
}

// Key takes a key while the search has the keyboard, and says whether it was the search's: what types and erases,
// and what leaves it. Any other key, as the arrows, is the screen's.
func (s Search) Key(k tea.KeyPressMsg) (Search, bool) {
	switch k.String() {
	case "esc", "tab", "enter":
		s.Focused = false
		return s, true
	}
	q, ok := Type(s.Query, k)
	s.Query = q
	return s, ok
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
		return Find + " " + style.Off("search")
	}
	return Find + " " + query
}

// SearchAt gives the column a search, drawn as line, starts at on the title line of a screen so wide: right-aligned a
// cell short of the edge, and as wide as an idle one at least, so it stays where it stands as it takes the keyboard,
// and grows leftward only once what is typed needs the room.
func SearchAt(width int, line string) int {
	return max(0, width-1-max(ansi.StringWidth(line), ansi.StringWidth(Find+" search")))
}
