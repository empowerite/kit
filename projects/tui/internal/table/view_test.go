// A table's view lays its pane out: the headings of one value frozen over the header, a blank line between, the header
// and its rule, and the rows' window; the cursor moves through the rows a row or a page at a time, keeping its row in
// view; and the rows are selected one at a time, a group at a time, or all at once.
package table

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"github.com/empowerite/kit/projects/tui/internal/grid"
	"github.com/empowerite/kit/projects/tui/internal/pane"
)

// viewing gives the outlined host in a pane of 12 lines, 40 wide, under a control line, #4 open, its title over three
// lines.
func viewing() *host {
	h := outlined()
	h.paneH, h.ctl, h.w = 12, true, 40
	h.tbl.Open["#4"] = true
	return h
}

// strip gives texts without their escapes.
func strip(texts []string) []string {
	out := make([]string, len(texts))
	for k, t := range texts {
		out[k] = ansi.Strip(t)
	}
	return out
}

func TestThePaneLaysTheHeaderOverTheRowsWindow(t *testing.T) {
	h := viewing()
	texts, places := Laid(h)
	if Frozen(h) != 1 || FrozenLines(h) != 0 || RowsTop(h) != 6 || RowsHeight(h) != 8 || HeaderTop(h) != 6 {
		t.Fatalf("frozen %d, %d lines; rows at %d, %d high; header at %d", Frozen(h), FrozenLines(h), RowsTop(h),
			RowsHeight(h), HeaderTop(h))
	}
	if !slices.Equal(places[:3], []Place{HeaderPlace(0), HeaderPlace(1), 0}) || places[len(places)-1] != -1 ||
		!strings.HasPrefix(strip(texts)[len(texts)-1], "   … ") {
		t.Errorf("the places %v:\n%q", places, strip(texts))
	}
	if line, ok := HeaderLine(h, 7); !ok || line != 1 || !OnRule(h, 7) || OnRule(h, 6) {
		t.Errorf("the rule: %d %v", line, ok)
	}
	if _, ok := HeaderLine(h, 8); ok {
		t.Error("a row is a header's line")
	}
	if _, ok := PlaceAt(h, 2); ok {
		t.Error("a line over the rows' window is in it")
	}
	// without a control line, the rows start at the pane's top
	h.ctl = false
	if RowsTop(h) != 4 || RowsHeight(h) != 10 {
		t.Errorf("no control line: at %d, %d high", RowsTop(h), RowsHeight(h))
	}
	// with no row to show, no header, and none of its lines
	h.panel.Query = "nonesuch"
	if HeaderTop(h) != -1 {
		t.Errorf("no header at %d", HeaderTop(h))
	}
}

func TestTheHeadingsOfOneValueStandFrozenOverTheHeader(t *testing.T) {
	h := viewing()
	h.panel = h.panel.FilteredBy("state", grid.Filter{Values: []string{"open"}})
	texts, places := Laid(h)
	if Frozen(h) != 2 || FrozenLines(h) != 1 || HeaderTop(h) != 8 || Gap(1, []string{"x"}) != 1 || Gap(0, []string{"x"}) != 0 {
		t.Fatalf("frozen %d, %d lines, header at %d", Frozen(h), FrozenLines(h), HeaderTop(h))
	}
	if got := strip(texts)[:2]; !strings.Contains(got[0], "open 3") || got[1] != "" || places[1] != -1 {
		t.Errorf("the frozen heading and the gap: %q", got)
	}
	if lines := ScrollLines(h); lines[0].Row != at(h, heading("open", "ada")) {
		t.Errorf("the first line that scrolls: %+v", lines[0])
	}
}

func TestTheCursorMovesThroughTheRowsAndTheWindowFollows(t *testing.T) {
	h := viewing()
	h.keys = false
	// down from outside the rows enters them, past the headings, on #3
	Down(h, 1)
	if !h.keys || h.Rows()[h.tbl.Cursor].Key != "#3" {
		t.Fatalf("entered on %q", h.Rows()[h.tbl.Cursor].Key)
	}
	// down to the last row, never past it; the window follows
	Down(h, 99)
	if h.Rows()[h.tbl.Cursor].Key != "#2" || h.tbl.Offset == 0 {
		t.Errorf("down to the last: %q, offset %d", h.Rows()[h.tbl.Cursor].Key, h.tbl.Offset)
	}
	// up to the first, and past it the rows give up the keyboard, the pane at its top
	Up(h, 1)
	Up(h, 99)
	if h.keys || h.left != 1 || h.tbl.Cursor != 0 || h.tbl.Offset != 0 {
		t.Errorf("up past the first: keys %v, cursor %d, offset %d", h.keys, h.tbl.Cursor, h.tbl.Offset)
	}
	if Up(h, 1); h.left != 1 {
		t.Error("up without the keyboard")
	}
	// a table of the heading over the groups alone has no row to enter
	empty := viewing()
	empty.all, empty.keys = nil, false
	if Down(empty, 1); empty.keys {
		t.Error("entered a table with no row")
	}
}

// [ and ] move the cursor to the groups' heading before it and after it, [ past the last heading to the last, and
// neither past the first or the last; { and } to the first and the last; home and end to the first row and the last.
// Without the keyboard, or with no heading to stand on, the cursor stays where it is.
func TestTheCursorMovesBetweenTheGroupsHeadingsAndToTheEnds(t *testing.T) {
	h := viewing()
	closed, open, ada, bob := at(h, heading("closed")), at(h, heading("open")), at(h, heading("open", "ada")),
		at(h, heading("open", "bob"))
	last := len(h.Rows()) - 1
	for _, c := range []struct {
		from  int
		press string
		want  int
	}{
		{last, "[", bob}, {bob, "[", ada}, {ada, "[", open}, {closed, "[", closed},
		{ada, "]", bob}, {bob, "]", bob}, {last, "{", closed}, {closed, "}", bob},
	} {
		h.tbl.Cursor = c.from
		if Group(h, c.press); h.tbl.Cursor != c.want {
			t.Errorf("%s from %d: on %d, want %d", c.press, c.from, h.tbl.Cursor, c.want)
		}
	}
	if End(h, 1); h.tbl.Cursor != last {
		t.Errorf("end: on %d, want %d", h.tbl.Cursor, last)
	}
	if End(h, -1); h.tbl.Cursor != First(h.Rows()) || h.tbl.Offset != 0 {
		t.Errorf("home: on %d, offset %d", h.tbl.Cursor, h.tbl.Offset)
	}
	h.keys = false
	if Group(h, "}"); h.tbl.Cursor != closed {
		t.Errorf("} without the keyboard: on %d", h.tbl.Cursor)
	}
	if End(h, 1); h.tbl.Cursor != closed {
		t.Errorf("end without the keyboard: on %d", h.tbl.Cursor)
	}
	// rows that group by nothing have no heading to stand on but the one over the groups
	bare := listing()
	bare.panel.Columns, bare.all, bare.keys = h.panel.Columns, h.all, true
	bare.tbl.Cursor = 2
	if Group(bare, "}"); bare.tbl.Cursor != 2 {
		t.Errorf("} with no heading: on %d", bare.tbl.Cursor)
	}
}

// Entering the rows from the line over them, the control line, puts the cursor on the outline's first line, the first
// group's heading, wherever it stood before; a table without a row stays as it is.
func TestEnteringTheRowsLandsOnTheOutlinesFirstLine(t *testing.T) {
	h := viewing()
	h.keys, h.tbl.Cursor = false, at(h, "#2")
	Enter(h)
	if first := h.Rows()[h.tbl.Cursor]; !h.keys || h.tbl.Cursor != First(h.Rows()) || !grid.Heads(first) {
		t.Errorf("entered on %q, at %d", first.Key, h.tbl.Cursor)
	}
	empty := viewing()
	empty.all, empty.keys = nil, false
	if Enter(empty); empty.keys {
		t.Error("entered a table with no row")
	}
}

func TestAPageScrollsThroughATallRowBeforeMovingOn(t *testing.T) {
	h := viewing()
	h.all[3].Cells[3].Values = []string{strings.Repeat("a long title ", 60)}
	h.paneH, h.keys = 8, true // the rows have 4 lines; #4 takes many more
	h.tbl.Cursor = at(h, "#4")
	Fit(h)
	lines := ScrollLines(h)
	first, last := rowLines(lines, h.tbl.Cursor)
	if last-first < 6 || h.tbl.Offset != first {
		t.Fatalf("#4 from %d to %d, the window from %d", first, last, h.tbl.Offset)
	}
	// down scrolls inside #4 first, then moves on past its end
	Page(h, 1)
	if h.tbl.Cursor != at(h, "#4") || h.tbl.Offset <= first {
		t.Fatalf("a page down inside #4: cursor %d, offset %d", h.tbl.Cursor, h.tbl.Offset)
	}
	// the head pins #4's first line while its rest shows, scrolled past
	if texts, places := Laid(h); !strings.Contains(ansi.Strip(texts[2]), "#4") || places[2] != first {
		t.Errorf("the pinned line: %v %q", places, strip(texts))
	}
	for k := 0; k < 20 && h.tbl.Cursor == at(h, "#4"); k++ {
		Page(h, 1)
	}
	if h.tbl.Cursor <= at(h, "#4") {
		t.Fatal("a page down never left #4")
	}
	// up, from inside #4's end, scrolls back up through it, then moves on
	h.tbl.Cursor, h.tbl.Offset = at(h, "#4"), last-1
	Page(h, -1)
	if h.tbl.Cursor != at(h, "#4") || h.tbl.Offset >= last-1 {
		t.Errorf("a page up inside #4: cursor %d, offset %d", h.tbl.Cursor, h.tbl.Offset)
	}
	for k := 0; k < 20 && h.tbl.Cursor == at(h, "#4"); k++ {
		Page(h, -1)
	}
	if h.tbl.Cursor >= at(h, "#4") {
		t.Error("a page up never left #4")
	}
	// without the keyboard, no head, and a page does nothing
	h.keys = false
	if Head(h, lines, RowsHeight(h)) != "" {
		t.Error("a head without the keyboard")
	}
	was := h.tbl.Cursor
	if Page(h, 1); h.tbl.Cursor != was {
		t.Error("a page without the keyboard")
	}
	if first, last := rowLines(lines, 99); first != -1 || last != -1 {
		t.Error("lines of no row")
	}
	// a cursor on no row leaves the window where it is
	h.keys, h.tbl.Cursor, h.tbl.Offset = true, 99, 0
	if Fit(h); h.tbl.Offset != 0 {
		t.Errorf("fit to no row: %d", h.tbl.Offset)
	}
}

// Paging through an open row three or four panes tall shows each of its lines once on the way down, none skipped and
// none twice, and every one of them again on the way back up, though a … line above and one below take two of the
// pane's lines.
func TestAPageThroughATallRowShowsEachOfItsLinesOnce(t *testing.T) {
	h := viewing()
	h.all[3].Cells[3].Values = []string{strings.Repeat("a long title ", 60)}
	for k := 5; k < 15; k++ { // rows under #4, so no page meets the table's end, where a window can only overlap
		h.all = append(h.all, grid.Row{Key: fmt.Sprintf("#%d", k), Cells: h.all[1].Cells})
	}
	h.tbl.Cursor = at(h, "#4")
	Fit(h)
	lines, height := ScrollLines(h), RowsHeight(h)
	first, last := rowLines(lines, h.tbl.Cursor)
	if rows := last - first + 1; rows < 3*height || rows > 5*height {
		t.Fatalf("#4 takes %d lines in a pane of %d", rows, height)
	}
	shown := func(seen map[int]int) {
		start, end, _, _ := paneWindow(h)
		for k := max(start, first); k < min(end, last+1); k++ {
			seen[k]++
		}
	}
	down, up := map[int]int{}, map[int]int{}
	for k := 0; k < 40 && h.tbl.Cursor == at(h, "#4"); k++ {
		shown(down)
		Page(h, 1)
	}
	h.tbl.Cursor, h.tbl.Offset = at(h, "#4"), last // up from a window that shows the row's last line
	for k := 0; k < 40 && h.tbl.Cursor == at(h, "#4"); k++ {
		shown(up)
		Page(h, -1)
	}
	// at the row's top, up moves the cursor a page of rows, as it does outside a tall row
	if want := max(First(h.Rows()), at(h, "#4")-(height-1)); h.tbl.Cursor != want {
		t.Errorf("a page up past #4's top: cursor %d, not %d", h.tbl.Cursor, want)
	}
	for k := first; k <= last; k++ {
		if down[k] != 1 || up[k] == 0 || up[k] > 2 {
			t.Errorf("line %d of #4 (%d to %d) shown %d times down, %d up", k, first, last, down[k], up[k])
		}
	}
}

func TestRowsAreSelectedAloneByGroupOrAll(t *testing.T) {
	h := viewing()
	h.keys = true
	h.tbl.Cursor = at(h, "#1")
	if SelectRow(h, false); !h.tbl.Picked["#1"] || len(SelectedRows(h)) != 1 {
		t.Fatalf("#1: %v", h.tbl.Picked)
	}
	if SelectRow(h, true); !h.tbl.Picked["#1"] {
		t.Error("keep unselected #1")
	}
	if SelectRow(h, false); h.tbl.Picked["#1"] {
		t.Error("#1 again stayed selected")
	}
	// on a group's heading, every row of the group, then none
	h.tbl.Cursor = at(h, heading("open"))
	if SelectRow(h, false); len(SelectedRows(h)) != 3 {
		t.Errorf("open's rows: %v", h.tbl.Picked)
	}
	if SelectRow(h, false); len(SelectedRows(h)) != 0 {
		t.Errorf("open's rows again: %v", h.tbl.Picked)
	}
	// an aside that heads no group selects nothing
	h.all = append(h.all, grid.Row{Key: "note", Aside: true, Cells: []grid.Cell{{Values: []string{"a note"}}}})
	h.tbl.Cursor = at(h, "note")
	if SelectRow(h, false); len(h.tbl.Picked) != 0 {
		t.Errorf("on an aside: %v", h.tbl.Picked)
	}
	// a cursor on no row selects nothing, nor does one without the keyboard
	h.tbl.Cursor = 99
	if SelectRow(h, false); len(h.tbl.Picked) != 0 {
		t.Errorf("on no row: %v", h.tbl.Picked)
	}
	h.keys, h.tbl.Cursor = false, at(h, "#1")
	if SelectRow(h, false); len(h.tbl.Picked) != 0 {
		t.Error("selected without the keyboard")
	}
	// all, then none
	if SelectAll(h); len(SelectedRows(h)) != 4 || Counted(grid.Every(h.Rows())) != 4 {
		t.Errorf("all: %v", h.tbl.Picked)
	}
	if SelectAll(h); len(h.tbl.Picked) != 0 {
		t.Errorf("all again: %v", h.tbl.Picked)
	}
}

// paneWindow gives the window of the lines that scroll the host's pane shows, as pane.Window gives it.
func paneWindow(h *host) (start, end int, above, below bool) {
	return pane.Window(len(ScrollLines(h)), RowsHeight(h), h.tbl.Offset)
}
