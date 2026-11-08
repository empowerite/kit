// A table's outline opens and closes its groups' headings and its rows, a level at a time too, keeps the cursor on its
// row, and marks a heading whose rows are picked; the heading over the groups opens and closes their levels.
package table

import (
	"slices"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"github.com/empowerite/kit/projects/tui/internal/grid"
)

// outlined gives a host of four rows, grouped by state then by who, the rows having the keyboard: open #1 by ada and
// #2 by bob, closed #3 by ada, and open #4 by ada, whose title runs on past the width, so it opens.
func outlined() *host {
	h := listing()
	h.panel.Columns = []string{"state", "who", "n", "title"}
	h.panel.Labels["n"] = "n"
	row := func(key, state, who, title string) grid.Row {
		return grid.Row{Key: key, Cells: []grid.Cell{{Values: []string{state}, Raw: []string{state}},
			{Values: []string{who}, Raw: []string{who}}, {Values: []string{key}}, {Values: []string{title}, Wrap: true}}}
	}
	h.all = []grid.Row{row("#1", "open", "ada", "one"), row("#2", "open", "bob", "two"), row("#3", "closed", "ada", "three"),
		row("#4", "open", "ada", strings.Repeat("a long title ", 20))}
	h.panel = h.panel.Grouped("state", true).Grouped("who", true)
	h.tbl = Table{Open: grid.Outline{}, Picked: map[string]bool{}}
	h.keys = true
	return h
}

// at gives the index of the row of the key among the host's rows.
func at(h *host, key string) int {
	return slices.IndexFunc(h.Rows(), func(r grid.Row) bool { return r.Key == key })
}

// heading gives the key of the heading of state's value, and of who's under it.
func heading(values ...string) string { return "group\x00\x01\x00" + strings.Join(values, "\x00") }

func TestTheOutlineOpensAndClosesTheNodeUnderTheCursor(t *testing.T) {
	h := outlined()
	rows := h.Rows()
	if First(rows) != 1 || First(nil) != 0 || !strings.Contains(ansi.Strip(BarText(rows)), "state") || BarText(nil) != "" {
		t.Fatalf("first %d, bar %q", First(rows), ansi.Strip(BarText(rows)))
	}
	// left on open's heading closes it, and the cursor stays on it
	h.tbl.Cursor = at(h, heading("open"))
	Expand(h, false, false)
	if h.tbl.Open[heading("open")] || at(h, "#1") >= 0 || h.Rows()[h.tbl.Cursor].Key != heading("open") {
		t.Fatalf("open closed: %v, cursor on %q", h.tbl.Open, h.Rows()[h.tbl.Cursor].Key)
	}
	// left again, on a closed heading, steps out: none over it but the heading over the groups, so it stays
	if Expand(h, false, false); h.Rows()[h.tbl.Cursor].Key != heading("open") {
		t.Errorf("left on a closed outermost heading moved the cursor")
	}
	// right opens it, and right again steps into its first row
	Expand(h, true, false)
	Expand(h, true, false)
	if got := h.Rows()[h.tbl.Cursor].Key; got != heading("open", "ada") {
		t.Errorf("right twice: on %q", got)
	}
	// on #4, right opens the row, whose title runs on; left on a row that hides nothing steps out to its heading
	h.tbl.Cursor = at(h, "#4")
	if Expand(h, true, false); !h.tbl.Open["#4"] {
		t.Error("right did not open #4")
	}
	h.tbl.Cursor = at(h, "#1")
	if Expand(h, false, false); h.Rows()[h.tbl.Cursor].Key != heading("open", "ada") {
		t.Errorf("left on #1: on %q", h.Rows()[h.tbl.Cursor].Key)
	}
	// on a heading, the subtree under it closes with it
	h.tbl.Open["#4"] = true
	h.tbl.Cursor = at(h, heading("open"))
	if !Disclose(h, false, true) || h.tbl.Open["#4"] || h.tbl.Open[heading("open", "ada")] {
		t.Errorf("the subtree closed: %v", h.tbl.Open)
	}
	// a cursor on no row stays where it was, for the host to keep on a row
	h.tbl.Open = grid.Outline{}
	h.tbl.Cursor = 99
	settled := h.settle
	if Reoutline(h, grid.Outline{grid.LevelKey(0): false}); h.tbl.Cursor != 99 || h.settle != settled+1 {
		t.Errorf("a cursor whose row is gone: %d", h.tbl.Cursor)
	}
	// without the keyboard, nothing
	h.keys = false
	if Disclose(h, false, false) {
		t.Error("closed without the keyboard")
	}
	if Expand(h, true, false); Grow(h) || Peel(h) {
		t.Error("a level without the keyboard")
	}
}

func TestOptionOpensOrClosesALevelAtATime(t *testing.T) {
	h := outlined()
	h.tbl.Cursor = at(h, heading("open"))
	// option with left closes the deepest open level under it: first the rows, of which #4 alone opens, so the
	// headings of who, then open itself
	h.tbl.Open["#4"] = true
	for _, want := range []string{"#4", heading("open", "ada"), heading("open")} {
		if !Peel(h) || h.tbl.Open[want] {
			t.Fatalf("peel: %q still open: %v", want, h.tbl.Open)
		}
	}
	if Peel(h) {
		t.Error("peeled with everything closed")
	}
	// option with right opens a level each press, the node itself first
	for _, want := range []string{heading("open"), heading("open", "ada"), "#4"} {
		if !Grow(h) || !h.tbl.Open.Unfolds(h.Rows()[at(h, heading("open"))]) || want == "#4" && !h.tbl.Open["#4"] {
			t.Fatalf("grow: %q not open: %v", want, h.tbl.Open)
		}
	}
	if Grow(h) {
		t.Error("grew with everything open")
	}
	if Expand(h, false, true); h.tbl.Open["#4"] {
		t.Error("option with left left #4 open")
	}
	if Expand(h, true, true); !h.tbl.Open["#4"] {
		t.Error("option with right left #4 closed")
	}
	// a cursor past the rows stands on none
	h.tbl.Cursor = 99
	if Disclose(h, true, false) || Grow(h) {
		t.Error("a node past the rows")
	}
	if Expand(h, true, false); h.tbl.Cursor != 99 {
		t.Error("a cursor past the rows moved")
	}
}

func TestTheFoldsActOnEverythingOrTheSelection(t *testing.T) {
	h := outlined()
	h.hides = true
	if _, ok := Folding(h, "Nonesuch"); ok {
		t.Error("a name that folds nothing")
	}
	next, _ := Folding(h, "Collapse All")
	if next[heading("open")] || next[HeaderKey] || !Changes(h, next) {
		t.Errorf("collapse all: %v", next)
	}
	Reoutline(h, next)
	if got := len(h.Rows()); got != 3 {
		t.Errorf("collapsed, %d rows", got)
	}
	next, _ = Folding(h, "Expand All")
	if !next[heading("open")] || !next["#4"] || !next[HeaderKey] {
		t.Errorf("expand all: %v", next)
	}
	h.tbl.Open = grid.Outline{grid.LevelKey(0): false}
	if next, _ := Folding(h, "Expand All"); !next.Shows(1) || !Changes(h, next) {
		t.Errorf("expand all shows every level: %v", next)
	}
	// with #4 picked: expanded, collapsed, and the others collapsed
	h.tbl = Table{Open: grid.Outline{}, Picked: map[string]bool{"#4": true}}
	Reoutline(h, grid.Outline{heading("open"): false})
	next, _ = Folding(h, "Expand Selected")
	if !next[heading("open")] || !next[heading("open", "ada")] || !next["#4"] {
		t.Errorf("expand selected: %v", next)
	}
	h.tbl.Open = next
	if next, _ := Folding(h, "Collapse Selected"); next["#4"] {
		t.Errorf("collapse selected: %v", next)
	}
	if next, _ := Folding(h, "Collapse Others"); next[heading("closed")] || !next[heading("open")] || next[heading("open", "bob")] {
		t.Errorf("collapse others: %v", next)
	}
	if Changes(h, h.tbl.Open) {
		t.Error("the same outline changes it")
	}
	if Changes(h, grid.Outline{grid.LevelKey(1): false, heading("open"): true, heading("open", "ada"): true,
		heading("open", "bob"): true, heading("closed"): true, "#4": true}) != true {
		t.Error("a level closed does not change it")
	}
	// with every heading closed, a level closed changes only the level
	h.tbl.Open = grid.Outline{heading("open"): false, heading("closed"): false, heading("open", "ada"): false,
		heading("open", "bob"): false, heading("closed", "ada"): false}
	if !Changes(h, mapWith(h.tbl.Open, grid.LevelKey(2), false)) {
		t.Error("a level alone does not change it")
	}
	if !Changes(h, mapWith(h.tbl.Open, HeaderKey, true)) {
		t.Error("the header opened does not change it")
	}
}

// mapWith gives an outline with one key set.
func mapWith(o grid.Outline, key string, open bool) grid.Outline {
	next := grid.Outline{}
	for k, v := range o {
		next[k] = v
	}
	next[key] = open
	return next
}

func TestAHeadingIsMarkedAsItsRowsArePicked(t *testing.T) {
	h := outlined()
	all := grid.Every(h.Rows())
	if marks := GroupMarks(all, map[string]bool{"#1": true, "#4": true}); marks[heading("open", "ada")] != grid.Chosen ||
		marks[heading("open")] != grid.Partly || marks[heading("closed")] != "" {
		t.Errorf("marks: %v", marks)
	}
	if got := GroupRows(all, slices.IndexFunc(all, func(r grid.Row) bool { return r.Key == heading("open") })); len(got) != 3 {
		t.Errorf("open's rows: %q", got)
	}
}

func TestTheFoldsOfARowAndAHeadingStandWhereTheyAreDrawn(t *testing.T) {
	h := outlined()
	if FoldAt(h.all[0]) != FoldCell {
		t.Error("a row's fold")
	}
	across := grid.Row{Key: heading("open"), Aside: true, Cells: []grid.Cell{{Values: []string{
		"  " + grid.Unfolded + " open " + grid.Unfolded + " ada 2"}}}}
	if got := FoldsAt(across); !slices.Equal(got, []int{FoldCell + 2, FoldCell + 9}) {
		t.Errorf("a heading across: %v", got)
	}
}

func TestTheHeadingOverTheGroupsOpensAndClosesTheirLevels(t *testing.T) {
	h := outlined()
	names := BarNames(h)
	if len(names) != 2 || names[0].Column != "state" || names[1].Column != "who" {
		t.Fatalf("the names: %+v", names)
	}
	if spot, ok := NameAt(h, names[1].From); !ok || spot.Column != "who" {
		t.Errorf("a click on who: %+v", spot)
	}
	if _, ok := NameAt(h, 0); ok {
		t.Error("a name off every name")
	}
	if level, ok := LevelAt(h, names[1].From-2); !ok || level != 1 {
		t.Errorf("who's fold: %d", level)
	}
	if level, ok := LevelAt(h, names[1].Until+1); !ok || level != 2 {
		t.Errorf("the rows' fold: %d", level)
	}
	if _, ok := LevelAt(h, 0); ok {
		t.Error("a level off every fold")
	}
	// who's level closes, hiding the rows under it, and opens again, as it was
	ToggleLevel(h, 1)
	if h.tbl.Open.Shows(1) || at(h, "#1") >= 0 {
		t.Errorf("who's level closed: %v", h.tbl.Open)
	}
	ToggleLevel(h, 1)
	if !h.tbl.Open.Shows(1) || at(h, "#1") < 0 {
		t.Errorf("who's level opened: %v", h.tbl.Open)
	}
	// a value that holds a column's name is passed over, and a column the heading does not name has no spot
	text := " " + grid.Unfolded + " who-ish " + grid.Unfolded + " who " + grid.Unfolded + " 2"
	if got := NamesIn(text, []string{"who", "nonesuch"}); len(got) != 1 || got[0].From != BarLeft+13 {
		t.Errorf("names in: %+v", got)
	}
}
