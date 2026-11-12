package pane

import (
	"strings"

	"github.com/charmbracelet/x/ansi"

	"github.com/empowerite/kit/projects/tui/internal/style"
)

// Before and After are the triangles a line that scrolls across has where it is cut: Before in place of its first
// cell, for what is scrolled past, and After in place of its last, for what is past the edge, as a table's header has
// them for its columns.
const (
	Before = "◃"
	After  = "▹"
)

// TurnCells is the cells a turn of the mouse wheel scrolls a block across.
const TurnCells = 4

// A Block is lines of a pane that scroll across together, so a code block's stay aligned: the lines from From up to
// To, their first Lead cells, an indent, standing still, the rest Wide cells at the widest; and its ID, which keeps how
// far it is scrolled while the lines are drawn again.
type Block struct {
	ID             string
	From, To, Lead int
	Wide           int
}

// room gives the cells a block's lines have after their lead in lines so many cells wide.
func (b Block) room(width int) int { return width - b.Lead }

// Across is how far each block of a pane is scrolled across, in cells, by its ID; a block not in it stands at 0.
type Across map[string]int

// Under gives the block line k of a pane stands in, in lines so many cells wide, that is wider than its room, so it
// scrolls; false when the line is in none.
func Under(blocks []Block, k, width int) (Block, bool) {
	for _, b := range blocks {
		if k >= b.From && k < b.To && b.Wide > b.room(width) {
			return b, true
		}
	}
	return Block{}, false
}

// Of gives how far a block is scrolled in lines so many cells wide, never past where its widest line ends at the edge.
func (a Across) Of(b Block, width int) int { return min(max(a[b.ID], 0), max(b.Wide-b.room(width), 0)) }

// Turn scrolls the block line k stands in so many cells across, right for more than 0, in lines so many cells wide,
// never past either end, and says whether it moved.
func (a Across) Turn(blocks []Block, k, width, cells int) bool {
	b, ok := Under(blocks, k, width)
	if !ok {
		return false
	}
	was := a.Of(b, width)
	a[b.ID] = was + cells
	a[b.ID] = a.Of(b, width)
	return a[b.ID] != was
}

// Slid gives the lines, so many cells wide, with each block's lead where it is and the rest from how far the block is
// scrolled, as Slide shows it; a line in no block, or in one that fits, is as it is.
func (a Across) Slid(lines []string, blocks []Block, width int) []string {
	out := append([]string(nil), lines...)
	for _, b := range blocks {
		if b.Wide <= b.room(width) {
			continue
		}
		for k := max(b.From, 0); k < min(b.To, len(out)); k++ {
			out[k] = a.slid(out[k], b, width)
		}
	}
	return out
}

// slid gives a line of a block as Slid shows it.
func (a Across) slid(line string, b Block, width int) string {
	return Cells(line, 0, b.Lead) + Slide(Cells(line, b.Lead, b.Lead+b.Wide), a.Of(b, width), b.room(width))
}

// Slide gives the cells of a line from x, n of them, with Before, dim, in place of the first where it hides text
// before them, and After in place of the last where it hides text after; a line with no text left of the edge shows
// Before alone.
func Slide(line string, x, n int) string {
	if n < 1 {
		return ""
	}
	wide := ansi.StringWidth(strings.TrimRight(ansi.Strip(line), " "))
	before, after := x > 0 && wide > 0, wide > x+n
	shown := Cells(line, x, min(x+n, max(wide, x)))
	if after {
		shown = Cells(shown, 0, n-1) + style.Off(After)
	}
	if before {
		shown = style.Off(Before) + Cells(shown, 1, n)
	}
	return shown
}

// Arrow gives which way a click on a pane's line k at cell x, in lines so many cells wide, scrolls the block it stands
// in: 1 on the After its line has, -1 on its Before, and 0 elsewhere.
func (a Across) Arrow(lines []string, blocks []Block, k, x, width int) int {
	b, ok := Under(blocks, k, width)
	if !ok || k >= len(lines) {
		return 0
	}
	shown := ansi.Strip(Slide(Cells(lines[k], b.Lead, b.Lead+b.Wide), a.Of(b, width), b.room(width)))
	switch x -= b.Lead; {
	case x == 0 && strings.HasPrefix(shown, Before):
		return -1
	case x == b.room(width)-1 && strings.HasSuffix(shown, After):
		return 1
	}
	return 0
}

// Click takes a plain click on a pane's line k at cell x, in lines so many cells wide: on its block's Before or After,
// as Arrow says, it scrolls the block half its room that way, and says it took the click.
func (a Across) Click(lines []string, blocks []Block, k, x, width int) bool {
	b, _ := Under(blocks, k, width)
	d := a.Arrow(lines, blocks, k, x, width)
	return d != 0 && a.Turn(blocks, k, width, d*max(b.room(width)/2, TurnCells))
}
