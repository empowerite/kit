// Package pane fits text into a space on the screen, for every screen alike: the window a pane of so many lines
// shows, the lines that say how many it hides, and text clipped, wrapped, or padded to a width. Widths are terminal
// cells, so a two-cell glyph counts as two.
package pane

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/x/ansi"

	"github.com/empowerite/kit/projects/tui/internal/style"
)

// Window says which of n lines a pane of h lines shows from offset o: the offset, clamped, the end, and whether a
// line says how many are hidden above, and how many below. Each such line takes the place of a shown one. A pane of
// fewer than three lines has no room for those lines, so it shows lines alone, from the offset.
func Window(n, h, o int) (start, end int, above, below bool) {
	if n <= h {
		return 0, n, false, false
	}
	if h < 3 {
		start = min(max(o, 0), n-max(h, 0))
		return start, start + max(h, 0), false, false
	}
	start = min(max(o, 0), n-h+1)
	above = start > 0
	end = start + h
	if above {
		end--
	}
	if end < n {
		below = true
		end--
	}
	return start, min(end, n), above, below
}

// At gives which of n lines a pane of h lines shows from offset o draws on its line r, from 0: its index, or -1 for a
// line that says how many are hidden, a line past the last, and a line off the pane. It is Window read backwards, as
// a click on the pane needs it.
func At(n, h, o, r int) int {
	if r < 0 || r >= h {
		return -1
	}
	start, end, above, _ := Window(n, h, o)
	if above {
		r--
	}
	if k := start + r; r >= 0 && k < end {
		return k
	}
	return -1
}

// Lines gives the lines a pane of h lines shows from offset o, with a dim … N above and … N more after lead where
// lines are hidden. A head, when given, takes the place of the … above.
func Lines(all []string, o, h int, head, lead string) []string {
	start, end, above, below := Window(len(all), h, o)
	out := make([]string, 0, max(h, 0))
	switch {
	case above && head != "":
		out = append(out, head)
	case above:
		out = append(out, lead+style.Off(Above(start)))
	}
	out = append(out, all[start:end]...)
	if below {
		out = append(out, lead+style.Off(More(len(all)-end)))
	}
	return out
}

// A Span is the rows a window shows out of all of them: from Start to End, under a line that says how many rows it
// hides above when Above, and over one that says how many it hides below when Below.
type Span struct {
	Start, End   int
	Above, Below bool
}

// Frame gives the rows a window of room lines shows out of n. A line that says how many rows the window hides takes
// a row's place, above it or below it, so the window never grows, and it hides two rows at least, never one it could
// show instead. The window starts at want, or as near it as holds the rows from first to last; a range longer than the
// window shows from first. A window that reaches the last row starts where it shows the most. A room under three
// lines has no place for those lines, and shows the rows from first. A room of zero or less shows every row.
func Frame(n, room, want, first, last int) Span {
	if room <= 0 || n <= room {
		return Span{End: n}
	}
	if room < 3 {
		start := min(max(first, 0), n-room)
		return Span{Start: start, End: start + room}
	}
	end := func(start int) int { // the end of the rows shown from start, less the lines that say what is hidden
		shown := room
		if start > 0 {
			shown--
		}
		if start+shown >= n {
			return n
		}
		return start + shown - 1
	}
	first, last = min(max(first, 0), n-1), min(max(last, first), n-1)
	start := min(max(want, 0), n-1)
	if end(start) <= last {
		start = max(start, min(last-room+3, n-room+1))
	}
	start = min(start, first)
	if start == 1 {
		start = 0 // the line above would hide one row, which it could show; from 0 the window ends at the same row
	}
	if end(start) == n {
		start = min(start, n-room+1)
	}
	return Span{Start: start, End: end(start), Above: start > 0, Below: end(start) < n}
}

// Above says how many rows a window hides above it.
func Above(n int) string { return fmt.Sprintf("… %d above", n) }

// More says how many rows a window hides below it.
func More(n int) string { return fmt.Sprintf("… %d more", n) }

// Clip shortens a text to n cells, with … at the cut.
func Clip(s string, n int) string {
	if n < 1 {
		return ""
	}
	return ansi.Truncate(s, n, "…")
}

// Tail shortens a text to its last n cells, with … in front at the cut, so the end of what was typed stays in sight.
func Tail(s string, n int) string {
	w := ansi.StringWidth(s)
	if w <= n {
		return s
	}
	if n < 1 {
		return ""
	}
	return "…" + Cells(s, w-n+1, w)
}

// Cells gives the cells of a line from the column from up to to, exactly so many, with every escape sequence in it, so
// its colors hold; a two-cell glyph either edge cuts in half is a space for its cell inside. ansi.Cut keeps such a
// glyph whole, a cell more than asked for, which moves everything after it.
func Cells(line string, from, to int) string {
	var b strings.Builder
	col, state := 0, byte(0)
	for line != "" {
		seq, width, n, next := ansi.DecodeSequence(line, state, nil)
		start, end := col, col+width
		switch {
		case width == 0:
			b.WriteString(seq)
		case start >= from && end <= to:
			b.WriteString(seq)
		case end > from && start < to:
			b.WriteString(strings.Repeat(" ", min(end, to)-max(start, from)))
		}
		col, state, line = end, next, line[n:]
	}
	return b.String()
}

// Break wraps a text as Wrap does, then breaks a word wider than the width across lines, so no line is wider and none
// of the text is lost.
func Break(text string, width int) []string {
	width = max(width, 10)
	var out []string
	for _, l := range Wrap(text, width) {
		out = append(out, strings.Split(ansi.Hardwrap(l, width, false), "\n")...)
	}
	return out
}

// Wrap breaks a text at spaces into lines no wider than the width, in cells, and never narrower than ten. A word wider
// than the width stands alone. A text without a word gives no line.
func Wrap(text string, width int) []string {
	width = max(width, 10)
	var lines []string
	current := ""
	for _, word := range strings.Fields(text) {
		switch {
		case current == "":
			current = word
		case ansi.StringWidth(current)+1+ansi.StringWidth(word) <= width:
			current += " " + word
		default:
			lines = append(lines, current)
			current = word
		}
	}
	if current != "" {
		lines = append(lines, current)
	}
	return lines
}

// Pad fills a text to the width with spaces, before it when right is set. A text as wide or wider is left as it is.
func Pad(text string, width int, right bool) string {
	n := width - ansi.StringWidth(text)
	if n <= 0 {
		return text
	}
	if right {
		return strings.Repeat(" ", n) + text
	}
	return text + strings.Repeat(" ", n)
}

// Width gives the widest of some lines, in cells.
func Width(lines []string) int {
	widest := 0
	for _, l := range lines {
		widest = max(widest, ansi.StringWidth(l))
	}
	return widest
}
