package shell

import (
	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/empowerite/kit/projects/tui/internal/style"
)

// Find is the search's glyph, before what is typed.
const Find = "🔎"

// Typing is the cells the search keeps at least for what is typed, while it is in use.
const Typing = 10

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

// SearchAt gives the column a search starts at on the title line of a screen so wide: idle and empty, right-aligned a
// cell short of the edge; in use, with the keyboard or a query, its glyph over the rightmost of numbers, the columns
// of the tab bar's numbers under it, that leaves Typing cells after it, or as far right as leaves them when none
// does.
func SearchAt(width int, numbers []int, inUse bool) int {
	if !inUse {
		return max(0, width-1-ansi.StringWidth(Find+" search"))
	}
	room := width - 1 - Typing - ansi.StringWidth(Find+" ")
	at := room
	for _, x := range numbers {
		if x <= room {
			at = x
		}
	}
	return max(0, at)
}

// InUse says whether the search is in use: it has the keyboard, or something typed.
func (s Search) InUse() bool { return s.Focused || s.Query != "" }
