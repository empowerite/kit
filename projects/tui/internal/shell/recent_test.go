package shell

import (
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
)

// The searches remembered are the newest first, each once, at most recentMost, kept in their file for every screen;
// with no file they are the session's alone.
func TestRecentRemembersTheNewestFirstOnceAndKeepsThemInTheFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "kit", "search-history.json")
	r := Recent{Path: path}.Remember("a").Remember("b").Remember("  ").Remember("a")
	if !slices.Equal(r.Items, []string{"a", "b"}) {
		t.Errorf("remembered: %q", r.Items)
	}
	// another screen reads them, and adds to them
	other := Recent{Path: path}.Loaded().Remember("c")
	if r = r.Loaded(); !slices.Equal(r.Items, []string{"c", "a", "b"}) || !slices.Equal(other.Items, r.Items) {
		t.Errorf("shared: %q, %q", r.Items, other.Items)
	}
	for k := range recentMost + 5 {
		r = r.Remember("q" + strconv.Itoa(k))
	}
	if len(r.Items) != recentMost || r.Items[0] != "q24" {
		t.Errorf("the newest %d: %d, %q", recentMost, len(r.Items), r.Items[0])
	}
	// a file that does not read leaves them as they were; none keeps them for the session
	if err := os.WriteFile(path, []byte("{"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := r.Loaded(); len(got.Items) != recentMost {
		t.Errorf("a bad file: %q", got.Items)
	}
	if s := (Recent{}).Remember("x"); !slices.Equal(s.Items, []string{"x"}) {
		t.Errorf("the session's: %q", s.Items)
	}
	if !strings.HasSuffix(RecentFile(), filepath.Join("kit", "search-history.json")) {
		t.Errorf("the file: %q", RecentFile())
	}
	t.Setenv("XDG_CONFIG_HOME", "/somewhere")
	if RecentFile() != "/somewhere/kit/search-history.json" {
		t.Errorf("under XDG_CONFIG_HOME: %q", RecentFile())
	}
	// with no XDG_CONFIG_HOME, under ~/.config, beside the views
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("HOME", "/home/ada")
	if RecentFile() != "/home/ada/.config/kit/search-history.json" {
		t.Errorf("under ~/.config: %q", RecentFile())
	}
}

// The dropdown opens on the newest, moves, chooses, and closes; it hangs from the Recall glyph against the search's 🔎,
// before what is typed, which shows only while the search has the keyboard and there are searches to recall.
func TestTheDropdownRecallsASearch(t *testing.T) {
	r := Recent{Items: []string{"one", "two"}}
	if (Recent{}).Opened().Open {
		t.Error("an empty dropdown opened")
	}
	r = r.Opened()
	key := func(code rune) tea.KeyPressMsg { return tea.KeyPressMsg{Code: code} }
	r, _ = r.Key(key(tea.KeyDown))
	r, _ = r.Key(key('x'))
	if r, got := r.Key(key(tea.KeyEnter)); got != "two" || r.Open {
		t.Errorf("enter: %q, open %v", got, r.Open)
	}
	r = r.Opened()
	if r, _ = r.Key(key(tea.KeyUp)); r.At != 0 {
		t.Errorf("up at the top: %d", r.At)
	}
	if r, got := r.Key(key(tea.KeyEscape)); got != "" || r.Open {
		t.Errorf("esc: %q, open %v", got, r.Open)
	}
	if ansi.Strip(RecallLine("ab", true, true)) != Find+Recall+" ab▏" || RecallLine("ab", false, true) != SearchLine("ab", false) ||
		RecallLine("ab", true, false) != SearchLine("ab", true) {
		t.Error("the recall glyph shows when it should not, or not when it should")
	}
	at := RecallAt(40)
	if line := strings.Repeat(" ", 40) + RecallLine("ab", true, true); ansi.Cut(line, at, at+1) != Recall {
		t.Errorf("the glyph is not at %d: %q", at, line)
	}
	// it hangs from the glyph; a click on a search chooses it, anywhere else closes it
	lines := r.Opened().Overlay(make([]string, 6), at, 80)
	if !strings.Contains(ansi.Strip(lines[0]), Recall) || !strings.Contains(ansi.Strip(lines[1]), "one") {
		t.Errorf("the overlay: %q", lines[:3])
	}
	if got := (Recent{Items: r.Items}).Overlay([]string{"x"}, at, 80); got[0] != "x" {
		t.Error("a closed dropdown drew")
	}
	box, left := r.Opened().Box(at, 80)
	if r, got := r.Opened().Click(left+3, 2, at, 80); got != "two" || r.Open {
		t.Errorf("a click on two: %q %v", got, box)
	}
	if r, got := r.Opened().Click(0, 5, at, 80); got != "" || r.Open {
		t.Errorf("a click off it: %q", got)
	}
}
