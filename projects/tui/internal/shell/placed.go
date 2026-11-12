// What every dialog shares, whatever it shows: a box, as Dialog draws one, where it stands on the screen, which says
// where a click falls on it; and the line under the keyboard, which moves to the next line it can stand on and stops at
// the ends. What a dialog shows, and what its lines do, are its own.
package shell

import "github.com/charmbracelet/x/ansi"

// Placed is a box where it stands: its lines, the screen line of its top edge, and the column of its left edge.
type Placed struct {
	Lines     []string
	Top, Left int
}

// Draw draws the box over the screen's lines.
func (p Placed) Draw(lines []string) []string { return Over(lines, p.Lines, p.Top, p.Left) }

// Hit says where a click at the screen cell x, y falls: off the box, on, false; on it, true, with the body line under
// it, from 0 for the line under the top edge, or -1 on the edges.
func (p Placed) Hit(x, y int) (line int, on bool) {
	if len(p.Lines) == 0 {
		return -1, false
	}
	width := ansi.StringWidth(p.Lines[0])
	if x < p.Left || x >= p.Left+width || y < p.Top || y >= p.Top+len(p.Lines) {
		return -1, false
	}
	if x == p.Left || x == p.Left+width-1 || y == p.Top || y == p.Top+len(p.Lines)-1 {
		return -1, true
	}
	return y - p.Top - 1, true
}

// Step gives the line under the keyboard moved from at by step, one line up or down, to the next of n lines it can
// stand on, as stands says; at itself where none is that way.
func Step(at, n, step int, stands func(int) bool) int {
	for k := at + step; k >= 0 && k < n; k += step {
		if stands(k) {
			return k
		}
	}
	return at
}

// Every says a dialog's cursor can stand on every line.
func Every(int) bool { return true }
