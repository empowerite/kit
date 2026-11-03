package pane

import (
	"slices"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

func TestTheWindowCountsWhatItHides(t *testing.T) {
	for _, c := range []struct {
		n, h, o, start, end int
		above, below        bool
	}{
		{5, 10, 3, 0, 5, false, false},
		{20, 10, 0, 0, 9, false, true},
		{20, 10, 5, 5, 13, true, true},
		{20, 10, 50, 11, 20, true, false},
		{20, 1, 5, 5, 6, false, false},
		{20, 2, 50, 18, 20, false, false},
		{20, 0, 5, 5, 5, false, false},
	} {
		start, end, above, below := Window(c.n, c.h, c.o)
		if start != c.start || end != c.end || above != c.above || below != c.below {
			t.Errorf("Window(%d, %d, %d) = %d %d %v %v", c.n, c.h, c.o, start, end, above, below)
		}
	}
}

// Every height from none to a dozen lines, every offset in and past the lines: the window never ends before it
// starts, never takes more lines than the pane has, and fills the pane when the lines do not fit.
func TestTheWindowFitsEveryHeight(t *testing.T) {
	for n := 0; n <= 30; n++ {
		for h := 0; h <= 12; h++ {
			for o := -2; o <= n+2; o++ {
				start, end, above, below := Window(n, h, o)
				taken := end - start
				if above {
					taken++
				}
				if below {
					taken++
				}
				switch {
				case start < 0 || end < start || end > n:
					t.Errorf("Window(%d, %d, %d) = %d %d", n, h, o, start, end)
				case taken > max(h, n) || n > h && taken != h:
					t.Errorf("Window(%d, %d, %d) takes %d lines", n, h, o, taken)
				case h < 3 && (above || below):
					t.Errorf("Window(%d, %d, %d) says lines are hidden in a pane too small for it", n, h, o)
				}
				all := make([]string, n)
				if got := Lines(all, o, h, "", "   "); n > h && len(got) != h {
					t.Errorf("Lines(%d, %d, %d) gives %d lines", n, h, o, len(got))
				}
			}
		}
	}
}

func TestLinesSayWhatIsHiddenOrPinTheHead(t *testing.T) {
	all := []string{"a", "b", "c", "d", "e", "f", "g", "h"}
	got := Lines(all, 3, 4, "", "  ")
	if len(got) != 4 || !strings.Contains(got[0], "… 3 above") || got[1] != "d" || got[2] != "e" ||
		!strings.Contains(got[3], "… 3 more") || !strings.HasPrefix(got[0], "  ") {
		t.Errorf("Lines from 3 in 4: %q", got)
	}
	if got := Lines(all, 3, 4, "head", "  "); got[0] != "head" {
		t.Errorf("a head does not take the place of the … above: %q", got)
	}
	if got := Lines(all[:2], 0, 4, "head", "  "); !slices.Equal(got, all[:2]) {
		t.Errorf("lines that fit are not shown as they are: %q", got)
	}
}

func TestAFrameHoldsTheRowAndItsLinesNeverGrowTheWindow(t *testing.T) {
	for n := 0; n <= 30; n++ {
		for room := 1; room <= 12; room++ {
			for at := 0; at < max(n, 1); at++ {
				for _, want := range []int{0, at - 4, at} {
					s := Frame(n, room, want, at, at)
					lines := s.End - s.Start
					if s.Above {
						lines++
					}
					if s.Below {
						lines++
					}
					switch {
					case n <= room && (s.Start != 0 || s.End != n || s.Above || s.Below):
						t.Errorf("n %d room %d: a list that fits is cut: %+v", n, room, s)
					case n > 0 && (at < s.Start || at >= s.End):
						t.Errorf("n %d room %d at %d want %d: the row is not shown: %+v", n, room, at, want, s)
					case lines > room:
						t.Errorf("n %d room %d at %d want %d: %d lines: %+v", n, room, at, want, lines, s)
					case room >= 3 && n > room && lines != room:
						t.Errorf("n %d room %d at %d want %d: %d lines, the window not full: %+v", n, room, at, want,
							lines, s)
					case s.Above != (s.Start > 0 && room >= 3) || s.Below != (s.End < n && room >= 3):
						t.Errorf("n %d room %d at %d want %d: rows hidden and not said: %+v", n, room, at, want, s)
					case s.Above && s.Start < 2 || s.Below && n-s.End < 2:
						t.Errorf("n %d room %d at %d want %d: a line hides one row: %+v", n, room, at, want, s)
					}
				}
			}
		}
	}
	if s := Frame(5, 0, 0, 0, 0); s.Start != 0 || s.End != 5 {
		t.Errorf("a room of zero does not show every row: %+v", s)
	}
}

func TestTheHiddenLinesAreCounted(t *testing.T) {
	if Above(3) != "… 3 above" || More(2) != "… 2 more" {
		t.Errorf("Above and More: %q %q", Above(3), More(2))
	}
}

func TestClipCutsInCellsWithAnEllipsis(t *testing.T) {
	for _, c := range []struct {
		in   string
		n    int
		want string
	}{
		{"abcdef", 4, "abc…"},
		{"abc", 4, "abc"},
		{"abc", 0, ""},
		{"🏭🏭🏭", 4, "🏭…"},
	} {
		if got := Clip(c.in, c.n); got != c.want {
			t.Errorf("Clip(%q, %d) = %q, want %q", c.in, c.n, got, c.want)
		}
	}
}

func TestWrapBreaksAtSpacesAndKeepsALongWordWhole(t *testing.T) {
	got := Wrap("one two three four five six seven", 12)
	if !slices.Equal(got, []string{"one two", "three four", "five six", "seven"}) {
		t.Errorf("Wrap at 12: %q", got)
	}
	if got := Wrap("a supercalifragilistic word", 5); !slices.Contains(got, "supercalifragilistic") {
		t.Errorf("a long word is broken, or the width went under ten: %q", got)
	}
	if got := Wrap("   ", 20); got != nil {
		t.Errorf("a text without a word gives lines: %q", got)
	}
	for _, l := range Wrap(strings.Repeat("🏭 ", 30), 11) {
		if ansi.StringWidth(l) > 11 {
			t.Errorf("a line is wider than the width in cells: %q", l)
		}
	}
}

func TestTailKeepsTheEndInCells(t *testing.T) {
	for _, c := range []struct {
		in   string
		n    int
		want string
	}{
		{"abcdef", 4, "…def"},
		{"abc", 4, "abc"},
		{"abc", 0, ""},
		{"ab🏭🏭", 3, "…🏭"},
	} {
		if got := Tail(c.in, c.n); got != c.want || ansi.StringWidth(got) > max(c.n, 0) {
			t.Errorf("Tail(%q, %d) = %q, want %q", c.in, c.n, got, c.want)
		}
	}
}

func TestBreakKeepsEveryCellOfALongWord(t *testing.T) {
	word := strings.Repeat("9", 35)
	got := Break("--iterations="+word+" x", 12)
	if strings.Join(got, "") != "--iterations="+word+"x" {
		t.Errorf("Break lost text: %q", got)
	}
	for _, l := range got {
		if ansi.StringWidth(l) > 12 {
			t.Errorf("a line wider than 12: %q", l)
		}
	}
}

func TestPadFillsToTheWidthOnEitherSide(t *testing.T) {
	if got := Pad("ab", 4, false); got != "ab  " {
		t.Errorf("left: %q", got)
	}
	if got := Pad("ab", 4, true); got != "  ab" {
		t.Errorf("right: %q", got)
	}
	if got := Pad("abcde", 4, true); got != "abcde" {
		t.Errorf("a wider text is changed: %q", got)
	}
	if got := Width([]string{"ab", "🏭🏭", "a"}); got != 4 {
		t.Errorf("Width = %d", got)
	}
}
