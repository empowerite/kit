package shell

import (
	"testing"

	"github.com/charmbracelet/x/ansi"
)

// Found reverses each stretch of a text that holds what is typed, without regard to case, and keeps the text's own
// escapes: a style ending inside a stretch has the reversing taken up again after it, and nothing typed or nothing
// found gives the text as it is.
func TestFoundReversesWhatTheSearchFinds(t *testing.T) {
	bold := "\x1b[1mkit\x1b[m"
	link := "\x1b]8;;kit://flow\x07flow\x1b]8;;\x07"
	for _, c := range []struct{ text, query, want string }{
		{"Run kit now", "", "Run kit now"},
		{"Run kit now", "zzz", "Run kit now"},
		{"Run Kit now", "kit", "Run \x1b[7mKit\x1b[27m now"},
		{"kit and KIT", "Kit", "\x1b[7mkit\x1b[27m and \x1b[7mKIT\x1b[27m"},
		{"Run " + bold + " now", "un kit", "R\x1b[7mun \x1b[1m\x1b[7mkit\x1b[m\x1b[7m\x1b[27m now"},
		{"see " + link, "flow", "see \x1b]8;;kit://flow\x07\x1b[7mflow\x1b]8;;\x07\x1b[7m\x1b[27m"},
		{"a\tb", "b", "a\t\x1b[7mb\x1b[27m"},
		{"Grüße", "GRÜ", "\x1b[7mGrü\x1b[27mße"},
	} {
		got := Found(c.text, c.query)
		if got != c.want {
			t.Errorf("Found(%q, %q) = %q, want %q", c.text, c.query, got, c.want)
		}
		if ansi.Strip(got) != ansi.Strip(c.text) {
			t.Errorf("Found(%q, %q) changed the text: %q", c.text, c.query, ansi.Strip(got))
		}
	}
}

// NextFound steps from one line that holds what is typed to the next, or the one before, as n and N do, wrapping; a
// line is held without regard to case or escapes, and nothing typed, no line, or no match gives -1.
func TestNextFoundStepsFromMatchToMatch(t *testing.T) {
	lines := []string{"one", "\x1b[1mTwo\x1b[m kit", "three", "kit four", "five"}
	for _, c := range []struct {
		query    string
		cur, dir int
		want     int
	}{
		{"kit", -1, 1, 1},
		{"kit", 1, 1, 3},
		{"kit", 3, 1, 1},
		{"kit", 1, -1, 3},
		{"kit", 3, -1, 1},
		{"kit", -1, -1, 3},
		{"two", -1, 1, 1},
		{"four", 3, 1, 3},
		{"zzz", -1, 1, -1},
		{"", -1, 1, -1},
	} {
		if got := NextFound(lines, c.query, c.cur, c.dir); got != c.want {
			t.Errorf("NextFound(%q, cur %d, dir %d) = %d, want %d", c.query, c.cur, c.dir, got, c.want)
		}
	}
	if got := NextFound(nil, "kit", -1, 1); got != -1 {
		t.Errorf("NextFound over no lines = %d", got)
	}
}
