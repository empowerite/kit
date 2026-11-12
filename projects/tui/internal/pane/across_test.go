package pane

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"github.com/empowerite/kit/projects/tui/internal/style"
)

// Slide shows n cells of a line from x, ◃ in place of the first where text is scrolled past, ▹ in place of the last
// where text is past the edge, and ◃ alone where all of it is scrolled past; a blank line shows no triangle.
func TestSlideShowsTheCellsFromWhereTheBlockIsScrolled(t *testing.T) {
	for _, c := range []struct {
		line string
		x, n int
		want string
	}{
		{"abcdefghij", 0, 10, "abcdefghij"},
		{"abcdefghij", 0, 6, "abcde▹"},
		{"abcdefghij", 2, 6, "◃defg▹"},
		{"abcdefghij", 4, 6, "◃fghij"},
		{"abc", 4, 6, "◃"},
		{"abc   ", 0, 4, "abc"},
		{"    ", 2, 4, ""},
		{"", 2, 4, ""},
		{"abcdefghij", 2, 1, "◃"},
		{"abcdefghij", 0, 1, "▹"},
		{"abcdefghij", 2, 0, ""},
	} {
		if got := ansi.Strip(Slide(c.line, c.x, c.n)); got != c.want {
			t.Errorf("Slide(%q, %d, %d) = %q, want %q", c.line, c.x, c.n, got, c.want)
		}
	}
	// the line's color holds on after ◃, which is dim
	slid := Slide(style.Mark("abcdefghij"), 2, 6)
	if !strings.HasPrefix(slid, style.Off(Before)) || !strings.Contains(slid, style.Mark("defg")[:4]) {
		t.Errorf("a styled line, slid: %q", slid)
	}
}

// A turn scrolls the block under the line, so many cells, never past either end, and says whether it moved; a line in
// no block, or in one that fits, takes no turn. Slid draws each block from where it is scrolled, and the rest as it is.
func TestATurnScrollsTheBlockUnderTheLineAlone(t *testing.T) {
	lines := []string{"prose", "0123456789ab", "0123456", "prose", "short"}
	blocks := []Block{{ID: "code", From: 1, To: 3, Wide: 12}, {ID: "fits", From: 4, To: 5, Wide: 5}}
	a := Across{}
	if !a.Turn(blocks, 2, 8, TurnCells) || a["code"] != TurnCells {
		t.Fatalf("a turn right on the block's second line: %v", a)
	}
	if a.Turn(blocks, 1, 8, TurnCells) || a["code"] != 4 {
		t.Errorf("a turn right past the end, where its widest line ends at the edge: %v", a)
	}
	if a.Turn(blocks, 0, 8, TurnCells) || a.Turn(blocks, 4, 8, TurnCells) || a.Turn(blocks, 9, 8, 1) {
		t.Errorf("a turn on prose, on a block that fits, or past the lines moved: %v", a)
	}
	got := a.Slid(lines, blocks, 8)
	want := []string{"prose", "◃56789ab", "◃56", "prose", "short"}
	for k := range want {
		if ansi.Strip(got[k]) != want[k] {
			t.Errorf("line %d slid: %q, want %q", k, ansi.Strip(got[k]), want[k])
		}
	}
	if lines[1] != "0123456789ab" {
		t.Error("Slid changed the lines it was given")
	}
	if !a.Turn(blocks, 1, 8, -2*TurnCells) || a["code"] != 0 || a.Turn(blocks, 1, 8, -1) {
		t.Errorf("turns left, to the start and past it: %v", a)
	}
	// a block scrolled further than a wider pane lets it is drawn from as far as it can be
	a["code"] = 9
	if b, _ := Under(blocks, 1, 8); a.Of(b, 10) != 2 {
		t.Errorf("a block scrolled past its end in a wider pane: %d", a.Of(b, 10))
	}
	// a block's lines past the pane's are not drawn
	if got := (Across{}).Slid(lines[:2], blocks, 8); len(got) != 2 || ansi.Strip(got[1]) != "0123456▹" {
		t.Errorf("a block past the lines: %q", got)
	}
}

// Arrow says which way a click on a slid line's ◃ or ▹ scrolls its block, and 0 off them, on a line in no block, or
// past the lines.
func TestArrowGivesWhichWayATriangleScrolls(t *testing.T) {
	lines := []string{"0123456789ab", "0123456", "prose"}
	blocks := []Block{{ID: "code", From: 0, To: 2, Wide: 12}, {ID: "gone", From: 3, To: 5, Wide: 20}}
	a := Across{"code": 2}
	for _, c := range []struct{ k, x, want int }{
		{0, 0, -1}, {0, 7, 1}, {0, 3, 0}, {1, 0, -1}, {1, 7, 0}, {2, 0, 0}, {3, 0, 0},
	} {
		if got := a.Arrow(lines, blocks, c.k, c.x, 8); got != c.want {
			t.Errorf("a click on line %d at %d: %d, want %d", c.k, c.x, got, c.want)
		}
	}
	if got := (Across{}).Arrow(lines, blocks, 0, 0, 8); got != 0 {
		t.Errorf("a click on the first cell of a block not scrolled: %d", got)
	}
}

// A lead's cells stay where they are while the block after them scrolls, and a click on a triangle counts its cell from
// the line's start; a click on ◃ or ▹ scrolls the block half the cells it is shown in, and on neither, nothing.
func TestALeadStaysAndAClickOnATriangleScrollsHalfTheWidth(t *testing.T) {
	lines := []string{"  " + strings.Repeat("0123456789", 3)}
	blocks := []Block{{ID: "code", From: 0, To: 1, Lead: 2, Wide: 30}}
	a := Across{}
	if got := ansi.Strip(a.Slid(lines, blocks, 12)[0]); got != "  012345678▹" {
		t.Errorf("not scrolled, after the lead: %q", got)
	}
	if a.Click(lines, blocks, 0, 5, 12) || a.Click(lines, blocks, 0, 2, 12) || a["code"] != 0 {
		t.Errorf("a click off the triangles, or on ◃ where there is none, scrolled: %v", a)
	}
	if !a.Click(lines, blocks, 0, 11, 12) || a["code"] != 5 {
		t.Fatalf("a click on ▹: %v", a)
	}
	if got := ansi.Strip(a.Slid(lines, blocks, 12)[0]); got != "  ◃67890123▹" {
		t.Errorf("scrolled half the width, the lead put: %q", got)
	}
	if !a.Click(lines, blocks, 0, 2, 12) || a["code"] != 0 {
		t.Errorf("a click on ◃: %v", a)
	}
}
