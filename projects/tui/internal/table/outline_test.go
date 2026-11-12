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

// opened says whether the node of the key stands open in an outline, a heading by default, as Outline.Open says.
func opened(h *host, o grid.Outline, key string) bool {
	every := grid.Every(h.Rows())
	k := slices.IndexFunc(every, func(r grid.Row) bool { return r.Key == key })
	return k >= 0 && o.Open(every[k])
}

// keys gives the keys of the host's rows, in order.
func keys(h *host) []string {
	var out []string
	for _, r := range h.Rows() {
		out = append(out, r.Key)
	}
	return out
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
	// on a heading, Hide then Show puts back what showed: #4 opened under it stays open
	h.tbl.Open["#4"] = true
	h.tbl.Cursor = at(h, heading("open"))
	was := keys(h)
	if !Disclose(h, false) || !h.tbl.Open["#4"] || at(h, "#4") >= 0 {
		t.Errorf("hide: %v", h.tbl.Open)
	}
	if !Disclose(h, true) || !slices.Equal(keys(h), was) || !h.tbl.Open["#4"] {
		t.Errorf("hide then show: %v, want %v", keys(h), was)
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
	if Disclose(h, false) {
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
	if Disclose(h, true) || Grow(h) {
		t.Error("a node past the rows")
	}
	if Expand(h, true, false); h.tbl.Cursor != 99 {
		t.Error("a cursor past the rows moved")
	}
}

// Show on a heading whose level the heading over the groups folds shows what it holds and nothing else: the other
// headings of its level stay as they showed, and Hide then Show puts it back.
func TestShowUnderAFoldedLevelShowsOnlyItsNode(t *testing.T) {
	h := outlined()
	h.tbl.Open = grid.Outline{grid.LevelKey(1): false}
	Reoutline(h, h.tbl.Open)
	if at(h, heading("open", "ada")) >= 0 || at(h, heading("closed", "ada")) >= 0 {
		t.Fatalf("the level folded: %v", keys(h))
	}
	h.tbl.Cursor = at(h, heading("open"))
	if !Disclose(h, true) || at(h, heading("open", "ada")) < 0 || at(h, heading("closed", "ada")) >= 0 {
		t.Fatalf("show under a folded level: %v", keys(h))
	}
	was := keys(h)
	if Disclose(h, false); !Disclose(h, true) || !slices.Equal(keys(h), was) {
		t.Errorf("hide then show: %v, want %v", keys(h), was)
	}
}

// Show More shows the shallowest level where something hides, every node of it, and hides nothing; what stood open
// under a node it shows comes back as it stood. Show Less hides the deepest level open.
func TestShowMoreFillsTheShallowestLevelAndShowLessTrimsTheDeepest(t *testing.T) {
	h := outlined()
	// one branch open all the way to #4, the other closed: the closed heading shows first, the rows stay as they are
	h.tbl.Open = grid.Outline{heading("closed"): false, "#4": true}
	Reoutline(h, h.tbl.Open)
	next, ok := ShowMore(h, true)
	if !ok || !next.Unfolds(h.Rows()[at(h, heading("closed"))]) || !next["#4"] || next["#1"] {
		t.Fatalf("show more: %v", next)
	}
	for _, r := range grid.Every(h.Rows()) {
		if h.tbl.Open.Unfolds(r) && !next.Unfolds(r) {
			t.Errorf("show more hid %q", r.Key)
		}
	}
	// hidden, open keeps #4 open under it, and shows it again when it shows
	h.tbl.Cursor = at(h, heading("open"))
	Disclose(h, false)
	if next, _ := ShowMore(h, true); !opened(h, next, heading("open")) || !next["#4"] {
		t.Errorf("show more after hide: %v", next)
	}
	// show less hides the deepest level that stands open, #4 alone of the rows
	Disclose(h, true)
	if next, _ := ShowLess(h, true); next["#4"] || !opened(h, next, heading("open", "ada")) {
		t.Errorf("show less: %v", next)
	}
}

func TestTheFoldsActOnTheWholeTableOrTheSelection(t *testing.T) {
	h := outlined()
	if _, ok := Folding(h, "Nonesuch"); ok {
		t.Error("a name that folds nothing")
	}
	// from the outset, the headings open and the rows closed: Show Less hides the headings of who, Show More shows them
	next, _ := Folding(h, "Show Less")
	if opened(h, next, heading("open", "ada")) || opened(h, next, heading("closed", "ada")) || !opened(h, next, heading("open")) || !Changes(h, next) {
		t.Errorf("show less: %v", next)
	}
	Reoutline(h, next)
	// closed holds its one heading of who on its line, so #3 shows under it still
	if at(h, heading("open", "ada")) < 0 || at(h, "#1") >= 0 || at(h, "#3") < 0 {
		t.Errorf("a level less: %q", keys(h))
	}
	if next, _ = Folding(h, "Show More"); !opened(h, next, heading("open", "ada")) || next["#4"] {
		t.Errorf("show more: %v", next)
	}
	// with everything shown, Show More changes nothing
	h.tbl.Open = grid.Outline{"#4": true}
	if next, ok := Folding(h, "Show More"); !ok || Changes(h, next) {
		t.Errorf("show more with everything shown: %v", next)
	}
	// with #4 picked and open hidden: Show Selected shows open, ada under it, and #4, and leaves closed as it stood
	h.tbl = Table{Open: grid.Outline{heading("open"): false, heading("closed"): false}, Picked: map[string]bool{"#4": true}}
	Reoutline(h, h.tbl.Open)
	next, _ = Folding(h, "Show Selected")
	if !opened(h, next, heading("open")) || !opened(h, next, heading("open", "ada")) || !next["#4"] || opened(h, next, heading("closed")) {
		t.Errorf("show selected: %v", next)
	}
	h.tbl.Open = next
	if next, _ := Folding(h, "Hide Selected"); next["#4"] || !opened(h, next, heading("open", "ada")) || !opened(h, next, heading("open")) {
		t.Errorf("hide selected: %v", next)
	}
	// under a folded level, Show Selected shows the selected row's heading, and the others stay hidden
	h.tbl.Open = grid.Outline{grid.LevelKey(1): false}
	Reoutline(h, h.tbl.Open)
	next, _ = Folding(h, "Show Selected")
	if !next.Unfolds(h.Rows()[at(h, heading("open"))]) || next.Unfolds(h.Rows()[at(h, heading("closed"))]) {
		t.Errorf("show selected under a folded level: %v", next)
	}
	// with the outermost level folded, nothing but the heading over the groups shows; Show More shows the level again,
	// and Show Selected shows the selected row's headings, the others hidden
	h.tbl.Open = grid.Outline{grid.LevelKey(0): false}
	Reoutline(h, h.tbl.Open)
	if next, _ := Folding(h, "Show More"); !next.Shows(0) || opened(h, next, heading("open")) ||
		opened(h, next, heading("closed")) || !Changes(h, next) {
		t.Errorf("show more with the outermost level folded: %v", next)
	}
	if next, _ := Folding(h, "Show Selected"); !next.Shows(2) || !next["#4"] || opened(h, next, heading("closed")) {
		t.Errorf("show selected with the outermost level folded: %v", next)
	}
	if next, ok := Folding(h, "Show Less"); !ok || Changes(h, next) {
		t.Errorf("show less folded the heading over the groups: %v", next)
	}
	// with nothing picked, the selection's folds change nothing
	h.tbl.Picked = map[string]bool{}
	for _, name := range []string{"Show Selected", "Hide Selected"} {
		if next, _ := Folding(h, name); Changes(h, next) {
			t.Errorf("%s with nothing picked: %v", name, next)
		}
	}
	h.tbl = Table{Open: grid.Outline{heading("open"): true, heading("open", "ada"): true, "#4": false},
		Picked: map[string]bool{}}
	Reoutline(h, h.tbl.Open)
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

// The heading over the groups' stops are each group's name, from its fold, then the rows' fold and count to its end,
// each with the level its fold opens or closes, and where it is clicked; rows that group by nothing have none.
func TestTheHeadingOverTheGroupsHasAStopPerNameThenTheRows(t *testing.T) {
	h := outlined()
	names, stops := BarNames(h), Stops(h)
	want := []Stop{{"state", 0, names[0].From - 2, names[0].Until}, {"who", 1, names[1].From - 2, names[1].Until},
		{"", 2, names[1].Until + 1, BarLeft + ansi.StringWidth(ansi.Strip(BarText(h.Rows())))}}
	if !slices.Equal(stops, want) {
		t.Errorf("the stops: %+v, want %+v", stops, want)
	}
	for _, s := range stops {
		if level, ok := LevelAt(h, s.From); !ok || level != s.Level {
			t.Errorf("%q's fold at %d: level %d", s.Column, s.From, level)
		}
	}
	// under a state of one value, who folds as state does: its fold is a dim separator, which is no fold, and its stop
	// opens or closes state's level
	// a column the heading names by a label of its own, as merged by names by, is found by it, its stop and its fold
	// its own
	labelled := outlined()
	labelled.panel.Labels["who"] = "person"
	spots := BarNames(labelled)
	if len(spots) != 2 || spots[1].Column != "who" ||
		ansi.Cut("  "+ansi.Strip(BarText(labelled.Rows())), spots[1].From, spots[1].Until) != "person" {
		t.Errorf("who, labelled person: %+v", spots)
	}
	if level, ok := LevelAt(labelled, spots[1].From-2); !ok || level != 1 {
		t.Errorf("person's fold: level %d, %v", level, ok)
	}
	single := outlined()
	single.all = single.all[:2]
	under := BarNames(single)
	if got := Stops(single); len(got) != 3 || got[1] != (Stop{"who", 0, under[1].From - 2, under[1].Until}) {
		t.Errorf("who under one state: %+v", got)
	}
	if _, ok := LevelAt(single, under[1].From-2); ok {
		t.Error("the dim separator is a fold")
	}
	// a ▸ a column's name holds is no fold
	held := outlined()
	odd := "who" + grid.Folded
	held.panel.Columns[1], held.panel.Groupable[odd] = odd, true
	held.panel = held.panel.Grouped("who", true).Grouped(odd, true)
	if n := BarNames(held); len(n) != 2 || n[1].Column != odd {
		t.Fatalf("the names: %+v", n)
	} else if _, ok := LevelAt(held, n[1].Until-1); ok {
		t.Errorf("the %s in %s's name is a fold", grid.Folded, odd)
	}
	// rows that group by nothing stand under the rows' fold and count alone, one stop, its fold the rows' level
	bare := listing()
	bare.panel.Columns, bare.panel.Labels["n"], bare.all = h.panel.Columns, "n", h.all
	text := ansi.Strip(BarText(bare.Rows()))
	if got := Stops(bare); !slices.Equal(got, []Stop{{"", 0, BarLeft, BarLeft + ansi.StringWidth(text)}}) {
		t.Errorf("rows that group by nothing: %+v under %q", got, text)
	}
	if level, ok := LevelAt(bare, BarLeft); !ok || level != 0 {
		t.Errorf("the rows' fold over rows that group by nothing: %d, %v", level, ok)
	}
	if _, ok := LevelAt(bare, BarLeft+2); ok {
		t.Error("the count is a fold")
	}
	// no rows, no heading, and no stop
	bare.all = nil
	if Stops(bare) != nil {
		t.Error("no rows have stops")
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
	same := func(id string) string { return id }
	if got := NamesIn(text, []string{"who", "nonesuch"}, same); len(got) != 1 || got[0].From != BarLeft+13 {
		t.Errorf("names in: %+v", got)
	}
}
