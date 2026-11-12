package pane

import (
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"github.com/empowerite/kit/projects/tui/internal/style"
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
// At reads Lines backwards: each line a pane draws is the line At gives for its place, and the lines that say how
// many are hidden, the places past the last line, and the places off the pane are no line's.
func TestAtGivesTheLineLinesDrawsAtEachPlace(t *testing.T) {
	for n := 0; n <= 12; n++ {
		all := make([]string, n)
		for k := range all {
			all[k] = strconv.Itoa(k)
		}
		for h := 0; h <= 6; h++ {
			for o := 0; o <= n; o++ {
				drawn := Lines(all, o, h, "")
				for r := -1; r <= h; r++ {
					want := -1
					if r >= 0 && r < len(drawn) && !strings.HasPrefix(drawn[r], Up) && !strings.HasPrefix(drawn[r], Down) {
						want, _ = strconv.Atoi(drawn[r])
					}
					if got := At(n, h, o, r); got != want {
						t.Fatalf("At(%d, %d, %d, %d) = %d, want %d: %q", n, h, o, r, got, want, drawn)
					}
				}
			}
		}
	}
}

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
				if got := Lines(all, o, h, ""); n > h && len(got) != h {
					t.Errorf("Lines(%d, %d, %d) gives %d lines", n, h, o, len(got))
				}
			}
		}
	}
}

func TestLinesSayWhatIsHiddenOrPinTheHead(t *testing.T) {
	all := []string{"a", "b", "c", "d", "e", "f", "g", "h"}
	got := Lines(all, 3, 4, "")
	if len(got) != 4 || got[0] != Above(3) || got[1] != "d" || got[2] != "e" || got[3] != More(3) {
		t.Errorf("Lines from 3 in 4: %q", got)
	}
	if got := Lines(all, 3, 4, "head"); got[0] != "head" {
		t.Errorf("a head does not take the place of Above: %q", got)
	}
	if got := Lines(all[:2], 0, 4, "head"); !slices.Equal(got, all[:2]) {
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
	if Above(3) != Up+style.Off("₃") || More(12) != Down+style.Off("¹²") {
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
	// a pill cut keeps its right end, in its color, so it still reads as a pill; too narrow for the end and a cell of
	// text, it is cut as any text is
	pill := style.Pill("package", "#1d4ed8")
	got := Clip(pill, 7)
	if ansi.StringWidth(got) != 7 || ansi.Strip(got) != style.PillLeft+"pack…"+style.PillRight ||
		!strings.HasSuffix(got, style.PillEnd(style.PillRight, "#1d4ed8")) {
		t.Errorf("a pill cut: %q", got)
	}
	if got := Clip(pill, 2); ansi.Strip(got) != style.PillLeft+"…" {
		t.Errorf("a pill cut to two cells: %q", ansi.Strip(got))
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
		{"ab🏭🏭", 4, "… 🏭"},
	} {
		if got := Tail(c.in, c.n); got != c.want || ansi.StringWidth(got) != min(c.n, ansi.StringWidth(c.in)) {
			t.Errorf("Tail(%q, %d) = %q, want %q", c.in, c.n, got, c.want)
		}
	}
}

// Cells gives exactly the cells asked for, a two-cell glyph either edge cuts in half a space, and keeps the escapes,
// where ansi.Cut keeps the glyph whole.
func TestCellsCutsExactlyEvenThroughATwoCellGlyph(t *testing.T) {
	line := "a" + style.Off("🏭b") + "🌿"
	for _, c := range []struct {
		from, to int
		want     string
	}{{0, 6, "a🏭b🌿"}, {2, 6, " b🌿"}, {0, 2, "a "}, {1, 3, "🏭"}, {4, 5, " "}} {
		got := Cells(line, c.from, c.to)
		if ansi.Strip(got) != c.want || ansi.StringWidth(got) != c.to-c.from {
			t.Errorf("Cells(%d, %d) = %q, want %q", c.from, c.to, got, c.want)
		}
	}
	if !strings.Contains(Cells(line, 2, 6), "\x1b[") {
		t.Error("Cells dropped the colors")
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

// A style that runs across a break goes on on the next line, turned off where each line ends, so a line drawn apart, as
// a value broken in its column, keeps it; a line after the style ends, and a text with none, are as they were.
func TestAStyleRunsOnAcrossABreak(t *testing.T) {
	const dim, bold, off = "\x1b[2m", "\x1b[1m", "\x1b[m"
	got := Restyle([]string{dim + "jeffrey-", "aguilera" + off + " and", bold + "more", "still" + off, "plain"})
	want := []string{dim + "jeffrey-" + off, dim + "aguilera" + off + " and", bold + "more" + off, bold + "still" + off,
		"plain"}
	for k := range want {
		if got[k] != want[k] {
			t.Errorf("line %d: %q, want %q", k, got[k], want[k])
		}
	}
	if lines := Wrap(dim+"a title long enough to break"+off, 12); lines[1] != dim+"enough to"+off {
		t.Errorf("wrapped: %q", lines)
	}
}
