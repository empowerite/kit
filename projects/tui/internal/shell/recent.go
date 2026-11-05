package shell

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
)

// Recall is the glyph after the search while it has the keyboard and there are searches to recall: a click on it, or
// ctrl+r, drops them down.
const Recall = "▾"

// RecallKeys is the keys bar while the search has the keyboard and there are searches to recall.
const RecallKeys = SearchKeys + " · ⌃R recent"

// recentMost is how many searches kittui remembers.
const recentMost = 20

// Recent is the searches kittui remembers, the newest first, every screen's alike, as their file keeps them, and the
// dropdown that recalls one: whether it is open, and the search under the keyboard. A search is remembered when it
// gives up the keyboard with something typed; one remembered again moves to the top. With no file, the searches are
// remembered for the session alone.
type Recent struct {
	Path  string
	Items []string
	Open  bool
	At    int
}

// RecentFile gives the file kittui keeps its searches in: kit/search-history.json under XDG_CONFIG_HOME, else under
// ~/.config, beside the views; nothing when neither is known.
func RecentFile() string {
	dir := os.Getenv("XDG_CONFIG_HOME")
	if dir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return ""
		}
		dir = filepath.Join(home, ".config")
	}
	return filepath.Join(dir, "kit", "search-history.json")
}

// Loaded gives the searches as the file keeps them now, which another screen may have added to; as they were when
// there is no file, or it does not read.
func (r Recent) Loaded() Recent {
	if r.Path == "" {
		return r
	}
	var items []string
	if data, err := os.ReadFile(r.Path); err == nil && json.Unmarshal(data, &items) == nil {
		r.Items = items
	}
	return r
}

// Remember puts a search at the top, once, and keeps the newest recentMost in the file; an empty one is not
// remembered. A file that cannot be written leaves the searches remembered for the session.
func (r Recent) Remember(query string) Recent {
	query = strings.TrimSpace(query)
	if query == "" {
		return r
	}
	r = r.Loaded()
	items := append([]string{query}, slices.DeleteFunc(slices.Clone(r.Items), func(q string) bool { return q == query })...)
	r.Items = items[:min(len(items), recentMost)]
	if r.Path != "" {
		if data, err := json.Marshal(r.Items); err == nil && os.MkdirAll(filepath.Dir(r.Path), 0o755) == nil {
			_ = os.WriteFile(r.Path, append(data, '\n'), 0o644)
		}
	}
	return r
}

// Opened gives the dropdown open, the newest search under the keyboard, when there is one to recall.
func (r Recent) Opened() Recent {
	r = r.Loaded()
	r.Open, r.At = len(r.Items) > 0, 0
	return r
}

// items gives the searches as a menu's items.
func (r Recent) items() []Item {
	out := make([]Item, len(r.Items))
	for k, q := range r.Items {
		out[k] = Item{Name: q}
	}
	return out
}

// Key takes a key while the dropdown is open: up and down move, enter and space choose, which closes it and gives
// the search chosen, and esc closes it. Whatever else is pressed leaves it as it is.
func (r Recent) Key(k tea.KeyPressMsg) (Recent, string) {
	every := func(int) bool { return true }
	switch k.String() {
	case "up":
		r.At = Move(r.items(), every, r.At, -1)
	case "down":
		r.At = Move(r.items(), every, r.At, 1)
	case "enter", "space":
		r.Open = false
		return r, r.Items[r.At]
	case "esc", "ctrl+r":
		r.Open = false
	}
	return r, ""
}

// Box gives the dropdown's lines, hanging from the Recall glyph at the column glyph of the title line, no wider than
// room, and the column its left edge is drawn at.
func (r Recent) Box(glyph, room int) ([]string, int) {
	return Box(r.items(), func(int) bool { return true }, r.At, Recall, glyph, room)
}

// Overlay draws the open dropdown over the screen's lines, from the title line, hanging from the glyph's column.
func (r Recent) Overlay(lines []string, glyph, room int) []string {
	if !r.Open {
		return lines
	}
	box, left := r.Box(glyph, room)
	return Over(lines, box, 0, left)
}

// Click takes a click while the dropdown is open: on a search, it chooses it, which closes the dropdown and gives
// it; anywhere else, it closes the dropdown.
func (r Recent) Click(x, y, glyph, room int) (Recent, string) {
	box, left := r.Box(glyph, room)
	r.Open = false
	if k := y - 1; x > left && x < left+ansi.StringWidth(box[0])-1 && k >= 0 && k < len(r.Items) {
		return r, r.Items[k]
	}
	return r, ""
}

// RecallLine draws a search as SearchLine does, and, while it has the keyboard and there are searches to recall,
// Recall after it.
func RecallLine(query string, focused, recall bool) string {
	if focused && recall {
		return SearchLine(query, true) + " " + Recall
	}
	return SearchLine(query, focused)
}

// RecallAt gives the screen column of the Recall glyph after a search that starts at the column at and has the
// keyboard.
func RecallAt(at int, query string) int {
	return at + ansi.StringWidth(SearchLine(query, true)) + 1
}
