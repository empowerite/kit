package shell

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/empowerite/kit/projects/tui/internal/style"
)

// Find is the search's glyph, before what is typed.
const Find = "🔎"

// Idle is what an idle, empty search reads, dim: /search, its key before its name.
const Idle = "/search"

// Search is the search every screen of kittui has, the same on each: on the menu bar's right, 🔎 and what is typed,
// which a table narrows its rows to, and a page's text keeps every line of, each match drawn as Found draws it. / or a
// click on it gives it focus, and a screen may open with focus there; then a printable key is typed, / and = too, as
// the filter goes, backspace takes one back, and ⌃U clears it; ⏎, ↓, esc, and tab give the view focus, the filter
// kept; and ⌃R recalls the past searches.
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

// The escapes Found draws a match between: reversed, and back, which leaves the rest of the text's style alone.
const (
	reverse   = "\x1b[7m"
	unreverse = "\x1b[27m"
)

// Found gives a text with each stretch of it that holds what is typed in a search reversed, as less draws what it
// finds, without regard to case; nothing typed, or nothing found, gives the text as it is. The text's own escapes are
// kept, and the reversing taken up again after each inside a stretch, so a style or link ending in it does not end it.
func Found(text, query string) string {
	if query == "" {
		return text
	}
	// each grapheme and escape of the text, a grapheme with its place in the plain text, lowered; an escape at -1
	type piece struct {
		s          string
		start, end int
	}
	var pieces []piece
	var plain strings.Builder
	var state byte
	for rest := text; rest != ""; {
		seq, _, n, next := ansi.DecodeSequence(rest, state, nil)
		state, rest = next, rest[n:]
		if seq[0] < 0x20 || seq[0] == 0x7f {
			pieces = append(pieces, piece{seq, -1, -1})
			continue
		}
		start := plain.Len()
		plain.WriteString(strings.ToLower(seq))
		pieces = append(pieces, piece{seq, start, plain.Len()})
	}
	low, p := strings.ToLower(query), plain.String()
	marked := make([]bool, len(pieces))
	for from := 0; ; {
		i := strings.Index(p[from:], low)
		if i < 0 {
			break
		}
		a, b := from+i, from+i+len(low)
		for k, pc := range pieces {
			marked[k] = marked[k] || pc.start >= 0 && pc.start < b && pc.end > a
		}
		from = b
	}
	var out strings.Builder
	on := false
	for k, pc := range pieces {
		switch {
		case pc.start < 0:
			out.WriteString(pc.s)
			if on {
				out.WriteString(reverse)
			}
			continue
		case marked[k] && !on:
			out.WriteString(reverse)
			on = true
		case !marked[k] && on:
			out.WriteString(unreverse)
			on = false
		}
		out.WriteString(pc.s)
	}
	if on {
		out.WriteString(unreverse)
	}
	return out.String()
}

// Holds says whether a text holds what is typed in a search, without regard to case or escapes; nothing typed holds
// in no text, as Found marks nothing.
func Holds(text, query string) bool {
	return query != "" && strings.Contains(strings.ToLower(ansi.Strip(text)), strings.ToLower(query))
}

// NextFound gives the line of a page that holds what is typed after the line cur, or before it for a dir below 0, as n and
// N take less from one match to the next: it wraps from the last to the first and back, and gives cur itself when it
// is the only one; -1 for nothing typed or no line that holds it. A cur of -1 is before the first line.
func NextFound(lines []string, query string, cur, dir int) int {
	n := len(lines)
	if n == 0 {
		return -1
	}
	for k := 1; k <= n; k++ {
		i := ((cur+dir*k)%n + n) % n
		if cur < 0 && dir < 0 {
			i = n - k
		}
		if Holds(lines[i], query) {
			return i
		}
	}
	return -1
}
