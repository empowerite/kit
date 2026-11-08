package grid

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/empowerite/kit/projects/tui/internal/style"
)

// one gives a cell of one value.
func one(s string) Cell { return Cell{Values: []string{s}} }

// issues are three rows like backlog's: number, assignees, age, and a title that wraps.
func issues() []Row {
	return []Row{
		{Key: "#1", Cells: []Cell{{Values: []string{"#1"}, Right: true},
			{Values: []string{"👤 ada", "👤 bob"}, Raw: []string{"ada", "bob"}},
			{Values: []string{"2h"}, Right: true}, {Values: []string{"a title long enough to wrap in its column"}, Wrap: true}}},
		{Key: "#22", Cells: []Cell{{Values: []string{"#22"}, Right: true}, {Values: []string{"👤 carol"}, Raw: []string{"carol"}},
			{Values: []string{"40d"}, Right: true}, {Values: []string{"short"}, Wrap: true}}},
		{Key: "#3", Cells: []Cell{{Values: []string{"#3"}, Right: true}, {}, {Values: []string{"1d"}, Right: true},
			{Values: []string{"leaf"}, Wrap: true}}, Detail: [][]Cell{{one("https://example/3")}}},
	}
}

// plain gives the lines' texts without their escapes, their trailing spaces trimmed.
func plain(lines []Line) []string {
	out := make([]string, len(lines))
	for i, l := range lines {
		out[i] = strings.TrimRight(ansi.Strip(l.Text), " ")
	}
	return out
}

func TestAClosedRowShowsTheFirstValueWithItsCountRightAlignedAndTheTextCut(t *testing.T) {
	got := plain(Draw(issues(), nil, 40, -1, func(Row) bool { return false }, nil))
	want := []string{
		"  ▸  #1  👤 ada +1   2h  a title long e…",
		"    #22  👤 carol   40d  short",
		"  ▸  #3              1d  leaf",
	}
	if !slices.Equal(got, want) {
		t.Errorf("closed:\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
}

func TestAnOpenRowShowsEveryValueAndWrapsItsTextInItsColumnThenItsDetail(t *testing.T) {
	got := plain(Draw(issues(), nil, 40, -1, func(Row) bool { return true }, nil))
	want := []string{
		"  ▾  #1  👤 ada      2h  a title long",
		"         👤 bob          enough to wrap",
		"                         in its column",
		"    #22  👤 carol   40d  short",
		"  ▾  #3              1d  leaf",
		"       https://example/3",
	}
	if !slices.Equal(got, want) {
		t.Errorf("open:\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
	// a row that hides nothing never opens, and every line says which row it is of
	lines := Draw(issues(), nil, 40, -1, func(Row) bool { return true }, nil)
	if rows := []int{lines[0].Row, lines[2].Row, lines[3].Row, lines[5].Row}; !slices.Equal(rows, []int{0, 0, 1, 2}) {
		t.Errorf("the lines' rows: %v", rows)
	}
}

func TestOpensSaysWhichRowsHideSomethingAtTheWidth(t *testing.T) {
	rows := issues()
	if !Opens(rows, nil, 0, 40) || Opens(rows, nil, 1, 40) || !Opens(rows, nil, 2, 40) {
		t.Errorf("at 40: %v %v %v", Opens(rows, nil, 0, 40), Opens(rows, nil, 1, 40), Opens(rows, nil, 2, 40))
	}
	// a cell with nothing in it opens to one empty line
	if got := wrapped(Cell{Wrap: true}, 10); len(got) != 1 || got[0] != "" {
		t.Errorf("an empty cell wrapped: %q", got)
	}
}

// A column needs its widest value, or its first and +N, widened to its name while it has text; columns fit a width
// with a gap between each two and, when one wraps, minFill left for it.
func TestNeedAndFitsMeasureTheColumnsAsDrawLaysThemOut(t *testing.T) {
	cells := []Cell{{Values: []string{"👤 ada", "👤 bob"}}, {Values: []string{"👤 carol"}}}
	if n := Need(cells[1:], ""); n != ansi.StringWidth("👤 carol") {
		t.Errorf("one assignee needs %d", n)
	}
	if n := Need(cells, ""); n != ansi.StringWidth("👤 ada +1") {
		t.Errorf("two assignees and one need %d", n)
	}
	if n := Need(cells, "the assignees"); n != len("the assignees") {
		t.Errorf("under a wide name, %d", n)
	}
	if n := Need([]Cell{{}, {}}, "assignees"); n != 0 {
		t.Errorf("a column without text needs %d", n)
	}
	for _, c := range []struct {
		needs []int
		wraps bool
		width int
		fits  bool
	}{
		{[]int{3, 5}, false, 3 + gap + 5, true},
		{[]int{3, 5}, false, 3 + gap + 5 - 1, false},
		{[]int{3, 0, 5}, false, 3 + gap + 5, true},
		{[]int{3, 5}, true, 3 + gap + 5 + gap + minFill, true},
		{[]int{3, 5}, true, 3 + gap + 5 + gap + minFill - 1, false},
		{nil, true, minFill, true},
	} {
		if got := Fits(c.needs, c.wraps, c.width); got != c.fits {
			t.Errorf("%v, wraps %v, at %d: %v", c.needs, c.wraps, c.width, got)
		}
	}
}

// A link leads from a cell's first word: its glyph, as a span's, or, drawn without one, its text's, as the title's a.
func TestLinkAtFindsTheLinkedGlyphOrFirstWordUnderAColumn(t *testing.T) {
	rows := issues()
	rows[0].Cells[3].Link = "https://example/1"
	rows[0].Cells[0].Link = "https://example/number"
	rows = append(rows, Row{Key: "s", Cells: []Cell{{Values: []string{"🔗 a span"}, Link: "https://example/span"}}})
	// the title starts after the gutter, #1, the assignees, and the age, each with the gap after it
	title := len(lead) + 3 + gap + ansi.StringWidth("👤 ada +1") + gap + 3 + gap
	for x, want := range map[int]string{title: "https://example/1", title + 1: "", title + 2: "",
		len(lead) + 1: "https://example/number", len(lead): "", 0: ""} {
		if got := LinkAt(rows, nil, 0, 40, x); got != want {
			t.Errorf("at %d: %q, want %q", x, got, want)
		}
	}
	if got := LinkAt(rows, nil, 3, 40, len(lead)+1); got != "https://example/span" {
		t.Errorf("a span's glyph: %q", got)
	}
	if got := LinkAt(rows, nil, 3, 40, len(lead)+2); got != "" {
		t.Errorf("after a span's glyph: %q", got)
	}
}

// The band marks the cursor's row, so its first cell is bold in the accent and not underlined.
func TestTheCursorsRowIsOnABandWithItsFirstCellChosen(t *testing.T) {
	lines := Draw(issues(), nil, 40, 1, func(Row) bool { return false }, nil)
	band, _, _ := strings.Cut(style.Band("x", 1), "x")
	chosen, _, _ := strings.Cut(style.Mark("#"), "#")
	underlined, _, _ := strings.Cut(style.Chosen("#"), "#")
	if !strings.HasPrefix(lines[1].Text, band) || !strings.Contains(lines[1].Text, chosen+"#") ||
		strings.Contains(lines[1].Text, underlined) {
		t.Errorf("the cursor's row is not on a band, its first cell bold in the accent alone: %q", lines[1].Text)
	}
	if ansi.StringWidth(lines[1].Text) != 40 || strings.Contains(lines[0].Text, chosen) || strings.HasPrefix(lines[0].Text, band) {
		t.Errorf("the band does not fill the width, or another row is chosen: %q", lines[0].Text)
	}
}

func TestAChosenCellThatIsDimIsDrawnChosenAlone(t *testing.T) {
	rows := []Row{{Key: "a", Cells: []Cell{{Values: []string{style.Off("abc")}}, one("x")}}}
	chosen, _, _ := strings.Cut(style.Mark("a"), "a")
	dim, _, _ := strings.Cut(style.Off("a"), "a")
	if got := Draw(rows, nil, 20, 0, func(Row) bool { return false }, nil)[0].Text; !strings.Contains(got, chosen+"a") ||
		strings.Contains(got, dim) {
		t.Errorf("a dim first cell under the cursor: %q", got)
	}
}

func TestARowOfOneCellSpansTheWidthAndOpensWhenItsTextIsCut(t *testing.T) {
	rows := []Row{{Key: "h", Cells: []Cell{{Values: []string{"a heading far too long for the pane"}, Wrap: true}}}}
	closed := plain(Draw(rows, nil, 20, -1, func(Row) bool { return false }, nil))
	open := plain(Draw(rows, nil, 20, -1, func(Row) bool { return true }, nil))
	if !slices.Equal(closed, []string{"  ▸ a heading far t…"}) || len(open) != 3 || open[1] != "    too long for the" {
		t.Errorf("closed %q, open %q", closed, open)
	}
}

// Step moves a field a place: within its section, a column changes position and a group depth; past its section's end,
// its placement, a column up into the groups' last depth where it can group, and down into the hidden, a group down
// into the first column, a hidden field up into the last column. A field that cannot move that way stays, and says so.
func TestStepMovesAFieldAcrossTheSections(t *testing.T) {
	columns := []string{"id", "state", "who", "title"}
	p := Panel{Columns: columns, Groupable: map[string]bool{"state": true, "who": true}}
	step := func(p Panel, id string, d int) Panel {
		p, _ = p.Step(id, d)
		return p
	}
	sections := func(p Panel) string {
		return fmt.Sprintf("%v %v %v", p.section(Group), p.section(Column), p.section(Hidden))
	}
	if p = step(p, "state", -1); sections(p) != "[] [state id who title] []" {
		t.Fatalf("state up: %s", sections(p))
	}
	if p = step(p, "state", -1); sections(p) != "[state] [id who title] []" {
		t.Fatalf("state up from the first column: %s", sections(p))
	}
	if q, ok := p.Step("state", -1); ok || sections(q) != sections(p) {
		t.Errorf("the first group up: %s", sections(q))
	}
	// who, the second column, up into the groups' last depth, then up a depth
	if p = step(step(p, "who", -1), "who", -1); sections(p) != "[state who] [id title] []" || p.Field("who").Depth != 2 {
		t.Fatalf("who up twice: %s", sections(p))
	}
	if p = step(p, "who", -1); sections(p) != "[who state] [id title] []" || p.Field("state").Depth != 2 {
		t.Fatalf("who up a depth: %s", sections(p))
	}
	// state, the last group, down into the first column
	if p = step(p, "state", 1); sections(p) != "[who] [state id title] []" || p.Field("state").Placement != Column {
		t.Fatalf("state down: %s", sections(p))
	}
	// id cannot group: up from the first column it stays
	if p = step(p, "id", -1); sections(p) != "[who] [id state title] []" {
		t.Fatalf("id up: %s", sections(p))
	}
	if q, ok := p.Step("id", -1); ok || sections(q) != sections(p) {
		t.Errorf("id, which cannot group, up from the first column: %s", sections(q))
	}
	// the last column down is hidden, and keeps its position; a hidden field up is the last column; down, it stays
	if p = step(p, "title", 1); sections(p) != "[who] [id state] [title]" {
		t.Fatalf("title down: %s", sections(p))
	}
	if _, ok := p.Step("title", 1); ok {
		t.Error("a hidden field moved down")
	}
	if p = step(p, "title", -1); sections(p) != "[who] [id state title] []" {
		t.Fatalf("title up: %s", sections(p))
	}
	// a field no row fills, which the columns dialog does not list, never moves
	if _, ok := p.Moved("nonesuch", Column, 0); ok {
		t.Error("a field not listed moved")
	}
	if _, ok := p.Step("nonesuch", 1); ok {
		t.Error("a field not listed stepped")
	}
}

func TestTheFilterKeepsWhatTheSearchAndShowLetThroughAndDropsHiddenColumns(t *testing.T) {
	columns := []string{"number", "who", "age", "title"}
	keys := func(rows []Row) []string {
		var out []string
		for _, r := range rows {
			out = append(out, r.Key)
		}
		return out
	}
	p := Panel{Me: "ada", Owner: "who"}
	if got := keys(p.Filter(issues(), columns)); !slices.Equal(got, []string{"#1", "#22", "#3"}) {
		t.Errorf("no filter: %v", got)
	}
	// the search ignores case and escapes, and reads the lines a row opens to
	for query, want := range map[string][]string{"WRAP": {"#1"}, "example/3": {"#3"}, "carol": {"#22"}, "zz": nil} {
		p.Query = query
		if got := keys(p.Filter(issues(), columns)); !slices.Equal(got, want) {
			t.Errorf("search %q: %v", query, got)
		}
	}
	p.Query = ""
	if p = p.Shown(Mine); !slices.Equal(keys(p.Filter(issues(), columns)), []string{"#1"}) {
		t.Errorf("mine: %v", keys(p.Filter(issues(), columns)))
	}
	if p = p.Shown(Others); !slices.Equal(keys(p.Filter(issues(), columns)), []string{"#22", "#3"}) {
		t.Errorf("others: %v", keys(p.Filter(issues(), columns)))
	}
	// while the caller's login is unknown, the caller is no one
	if p.Me = ""; !slices.Equal(keys(p.Shown(Mine).Filter(issues(), columns)), nil) {
		t.Errorf("mine without a login: %v", keys(p.Shown(Mine).Filter(issues(), columns)))
	}
	// a filter includes a row holding one of its values, none a row holding no value, and an exclusion the others
	for _, c := range []struct {
		filter Filter
		want   []string
	}{
		{Filter{Values: []string{"carol", "bob"}}, []string{"#1", "#22"}},
		{Filter{Values: []string{NoValue}}, []string{"#3"}},
		{Filter{Values: []string{"bob", NoValue}, Exclude: true}, []string{"#22"}},
	} {
		q := Panel{Fields: map[string]Field{"who": {Filter: c.filter}}}
		if got := keys(q.Filter(issues(), columns)); !slices.Equal(got, c.want) || !q.Filtering() || q.Showing() != All {
			t.Errorf("the filter %+v: %v", c.filter, got)
		}
		if q.Owner = "who"; q.Showing() != -1 {
			t.Errorf("the filter %+v says a choice of show: %v", c.filter, q.Showing())
		}
	}
	// a hidden column is left out of every row
	p = Panel{Fields: map[string]Field{"who": {Placement: Hidden}}}
	got := plain(Draw(p.Filter(issues(), columns), nil, 40, -1, func(Row) bool { return false }, nil))
	if strings.Contains(strings.Join(got, "\n"), "👤") || !strings.HasPrefix(got[1], "    #22  40d  short") {
		t.Errorf("who hidden:\n%s", strings.Join(got, "\n"))
	}
	// an aside heading no row the search keeps goes with them; without a search, every aside stays
	headed := []Row{{Key: "a", Aside: true, Cells: []Cell{one("a")}}, {Key: "x", Cells: []Cell{one("x")}},
		{Key: "b", Aside: true, Cells: []Cell{one("b")}}, {Key: "y", Cells: []Cell{one("y")}},
		{Key: "c", Aside: true, Cells: []Cell{one("unchanged")}}}
	if got := keys(Panel{Query: "y"}.Filter(headed, nil)); !slices.Equal(got, []string{"b", "y"}) {
		t.Errorf("asides with a search: %v", got)
	}
	if got := keys(Panel{}.Filter(headed, nil)); len(got) != 5 {
		t.Errorf("asides without one: %v", got)
	}
}

// The header stands each name in its column, as Draw lays the rows out under it: over a number right-aligned, over
// the text that wraps at its start, and left out over a column no row has text in. A name widens its column, which
// the rows then take too; one over the text that wraps, which takes the width left, wraps in it, and the header shows
// its first line alone, with ▸ before it, until it is open, with ▾. Rows of one cell have none.
func TestTheHeaderStandsEachNameOverItsColumn(t *testing.T) {
	rows := []Row{
		{Key: "a", Cells: []Cell{{Values: []string{"#1"}, Right: true}, one(""), one("🔀 #2"), {Values: []string{"a title"}, Wrap: true}}},
		{Key: "b", Cells: []Cell{{Values: []string{"#31"}, Right: true}, one(""), one("🔀 #12"), {Values: []string{"another"}, Wrap: true}}},
	}
	names := []string{"id", "fork", "changes", "the title of the issue it closes"}
	strip := func(lines []string) []string {
		out := make([]string, len(lines))
		for k, l := range lines {
			out[k] = ansi.Strip(l)
		}
		return out
	}
	closed, hides := Header(rows, names, 40, false)
	plain := strip(closed)
	drawn := ansi.Strip(Draw(rows, names, 40, -1, func(Row) bool { return false }, nil)[1].Text)
	// changes widens its column; fork is over no text, and left out; the title's name wraps, the header closed
	if !hides || len(closed) != 1 || plain[0] != "  ▸  id  changes  the title of the issue" || strings.Contains(plain[0], "fork") {
		t.Fatalf("the closed header:\n%s", strings.Join(plain, "\n"))
	}
	for name, under := range map[string]string{"changes": "🔀", "the": "another"} {
		if ansi.StringWidth(plain[0][:strings.Index(plain[0], name)]) != ansi.StringWidth(drawn[:strings.Index(drawn, under)]) {
			t.Errorf("%s does not stand over its column:\n%s\n%s", name, plain[0], drawn)
		}
	}
	if !strings.HasSuffix(plain[0][:strings.Index(plain[0], "  changes")], "id") ||
		ansi.StringWidth(plain[0][:strings.Index(plain[0], "id")])+2 != ansi.StringWidth(drawn[:strings.Index(drawn, "#31")])+3 {
		t.Errorf("id is not right-aligned over #31:\n%s\n%s", plain[0], drawn)
	}
	if !strings.Contains(closed[0], style.Label("id")) || !strings.Contains(closed[0], style.Off("▸")) {
		t.Errorf("the names are not bold, or ▸ not dim: %q", closed[0])
	}
	open, _ := Header(rows, names, 40, true)
	if plain := strip(open); len(open) != 2 || !strings.HasPrefix(plain[0], "  ▾  id") || !strings.HasPrefix(plain[1], "     ") ||
		!strings.HasSuffix(plain[1], "it closes") {
		t.Errorf("the open header:\n%s", strings.Join(plain, "\n"))
	}
	if got, hides := Header(rows, names[:3], 40, false); hides || strings.Contains(ansi.Strip(got[0]), "▸") {
		t.Errorf("a header of one line: %q", got)
	}
	if got, _ := Header([]Row{{Cells: []Cell{one("a heading")}}}, []string{"x"}, 40, false); got != nil {
		t.Errorf("rows of one cell have a header: %q", got)
	}
}

// A glyph whose link opens a page is underlined with dots; one that leads to a row of another stage is not.
func TestALinkToAPageIsDotted(t *testing.T) {
	page := Cell{Values: []string{"🎫 a title"}, Link: "https://example/1", Wrap: true}
	row := Cell{Values: []string{"🔀 #2"}, Link: "kittui:changes/#2"}
	got := closedLine([]Cell{one("x"), row, page}, []int{1, 5, 0}, 12, 40, false)
	if !strings.Contains(got, "\x1b[4:4m🎫\x1b[4:0m a title") || strings.Contains(got, "\x1b[4:4m🔀") {
		t.Errorf("the links: %q", got)
	}
	if dotted("\x1b[2m🎫\x1b[m", page) != "\x1b[2m🎫\x1b[m" || dotted("🎫", page) != "\x1b[4:4m🎫\x1b[4:0m" {
		t.Error("a styled text, or a glyph alone")
	}
}

// The text that wraps takes no more than its widest needs, minFill being the floor only of a text squeezed: a short
// title leaves the room after the last column, so a column after it stands right after the title, not across the row.
func TestAShortTitleLeavesTheRoomAfterTheLastColumn(t *testing.T) {
	rows := []Row{
		{Key: "a", Cells: []Cell{one("#1"), {Values: []string{"short"}, Wrap: true}, one("ready")}},
		{Key: "b", Cells: []Cell{one("#2"), {Values: []string{"shorter one"}, Wrap: true}, one("draft")}},
	}
	got := ansi.Strip(Draw(rows, nil, 80, -1, func(Row) bool { return false }, nil)[0].Text)
	// the title column is as wide as the widest title, shorter one, even under minFill; the state stands after it
	if got != "    #1  short        ready" {
		t.Errorf("a short title: %q", got)
	}
	rows[1].Cells[1].Values[0] = "a title of thirty cells, wide."
	if got := ansi.Strip(Draw(rows, nil, 80, -1, func(Row) bool { return false }, nil)[0].Text); got != "    #1  short"+strings.Repeat(" ", 27)+"ready" {
		t.Errorf("a title as wide as the longest: %q", got)
	}
}

// The title is the last column to give up room: while it has less than its text needs, a column wider than capWidth
// is cut at it, the widest first; while it has fewer than minFill, the columns are cut further, down to floorWidth; a
// column cut hides its text, so the row opens to it, broken across lines in its column.
func TestTheTitleIsTheLastToGiveUpRoom(t *testing.T) {
	branch := strings.Repeat("b", 30)
	cells := []Cell{one("#1"), one(branch), {Values: []string{strings.Repeat("t", 40)}, Wrap: true}}
	widths := columnWidths([]Row{{Cells: cells}})
	// room for everything: nothing is cut
	if got, left := fit(cells, widths, 2+2+30+2+40); !slices.Equal(got, widths) || left != 40 {
		t.Errorf("room for all: %v, title %d", got, left)
	}
	// the title short of its need: the branch gives up what is past the cap first
	if got, left := fit(cells, widths, 2+2+30+2+30); got[1] != 30-10 || left != 40 {
		t.Errorf("ten cells short: %v, title %d", got, left)
	}
	if got, left := fit(cells, widths, 2+2+30+2+20); got[1] != capWidth || left != 20+30-capWidth {
		t.Errorf("past the cap: %v, title %d", got, left)
	}
	// short of minFill even so: the branch gives up more, down to the floor, and only then the title
	if got, left := fit(cells, widths, 2+2+capWidth+2+minFill-4); got[1] != capWidth-4 || left != minFill {
		t.Errorf("under minFill: %v, title %d", got, left)
	}
	if got, left := fit(cells, widths, 2+2+floorWidth+2+4); got[1] != floorWidth || left != floorWidth {
		t.Errorf("past the floor: %v, title %d", got, left)
	}
	rows := []Row{{Key: "a", Cells: cells}}
	width := len(lead) + 2 + 2 + capWidth + 2 + 40
	closed := ansi.Strip(Draw(rows, nil, width, -1, func(Row) bool { return false }, nil)[0].Text)
	if !strings.Contains(closed, strings.Repeat("b", capWidth-1)+"…  "+strings.Repeat("t", 40)) || !Opens(rows, nil, 0, width) {
		t.Errorf("the branch cut at the cap: %q", closed)
	}
	open := Draw(rows, nil, width, -1, func(Row) bool { return true }, nil)
	if len(open) != 2 || !strings.Contains(open[1].Text, strings.Repeat("b", 30-capWidth)) {
		t.Errorf("the row open does not break the branch across lines: %v", open)
	}
}

// The panel orders the named columns as Order says, a column without a name keeping its place, and the rows' cells
// follow; Moved takes a column to another's place; shift with up or down moves the column under the keyboard, the
// keyboard with it, and stops at the columns' ends.
func TestThePanelOrdersTheColumns(t *testing.T) {
	columns := []string{"", "a", "b", "c"}
	p := Panel{Columns: columns, Fields: map[string]Field{"c": {Position: 1}, "a": {Position: 2}}}
	if got := p.Arrange(columns); !slices.Equal(got, []string{"", "c", "a", "b"}) {
		t.Errorf("arranged: %q", got)
	}
	r := p.hide(Row{Cells: []Cell{one("0"), one("A"), one("B"), one("C")}}, columns)
	if got := r.Cells[1].first() + r.Cells[2].first() + r.Cells[3].first(); r.Cells[0].first() != "0" || got != "CAB" {
		t.Errorf("the row's cells: %q", got)
	}
	if got, _ := (Panel{Columns: columns}).Moved("c", Column, 0); !slices.Equal(got.Arrange(columns), []string{"", "c", "a", "b"}) {
		t.Errorf("c moved to a: %q", got.Arrange(columns))
	}
	if got, ok := (Panel{Columns: columns}).Moved("nonesuch", Column, 0); ok || got.Fields != nil {
		t.Errorf("a move of no column: %v", got.Fields)
	}
	// a hidden column keeps its place, and stands there shown again
	moved, _ := (Panel{Columns: columns}).Moved("c", Column, 0)
	if got := moved.Placed("a", Hidden).Placed("a", Column).Arrange(columns); !slices.Equal(got, []string{"", "c", "a", "b"}) {
		t.Errorf("a hidden and shown again: %q", got)
	}
	q := Panel{Columns: columns}
	if q, _ = q.Step("a", 1); !slices.Equal(q.Arrange(columns), []string{"", "b", "a", "c"}) {
		t.Errorf("a down: %q", q.Arrange(columns))
	}
	q, _ = q.Step("a", -1)
	if q, _ = q.Step("a", -1); q.Arrange(columns)[1] != "a" {
		t.Errorf("a up past the first column: %q", q.Arrange(columns))
	}
}

// Rows sort by the panel's keys, each in turn: by a cell's sort value when it has one, else its text, naturally, so
// #7 comes before #31; descending reverses a key; rows alike keep their order, and a row of one cell its place. A click
// sorts by a column alone, ▲, ▼, then not at all; a shift-click adds it after the others; a column that cannot sort
// changes nothing.
func TestThePanelSortsTheRows(t *testing.T) {
	columns := []string{"id", "age"}
	row := func(id, age, secs string) Row {
		return Row{Key: id, Cells: []Cell{one(id), {Values: []string{age}, Raw: []string{secs}}}}
	}
	rows := []Row{row("#31", "2d", "172800"), row("#7", "60m", "3600"), {Key: "heading", Cells: []Cell{one("heading")}},
		row("#100", "60m", "3600")}
	keys := func(rows []Row) string {
		var out []string
		for _, r := range rows {
			out = append(out, r.Key)
		}
		return strings.Join(out, " ")
	}
	p := Panel{Sortable: map[string]bool{"id": true, "age": true}}
	if got := keys(p.Sorted("id").order(rows, columns)); got != "#7 #31 heading #100" {
		t.Errorf("by id: %s", got)
	}
	if got := keys(p.Sorted("id").Sorted("id").order(rows, columns)); got != "#100 #31 heading #7" {
		t.Errorf("by id, descending: %s", got)
	}
	// by age, then id: the two of 60m by id
	byAge := p.Sorted("age").Sorted("id")
	if got := keys(byAge.order(rows, columns)); got != "#7 #100 heading #31" || byAge.Mark("age") != "▲¹" || byAge.Mark("id") != "▲²" {
		t.Errorf("by age, then id: %s, marks %q %q", got, byAge.Mark("age"), byAge.Mark("id"))
	}
	// a key among several turns in its place; a third time on a column stops it; Unsorted clears every key
	if got := byAge.Sorted("id").SortKeys(); len(got) != 2 || got[1] != (SortKey{"id", true}) ||
		byAge.Sorted("id").Mark("id") != "▼₂" {
		t.Errorf("id turned among two keys: %+v", got)
	}
	if got := p.Sorted("id").Sorted("id").Sorted("id"); len(got.SortKeys()) != 0 || got.Mark("id") != "" {
		t.Errorf("three times: %+v", got.SortKeys())
	}
	if got := byAge.Sorted("id").Sorted("id").SortKeys(); len(got) != 1 || got[0].Column != "age" {
		t.Errorf("id twice more: %+v", got)
	}
	if got := byAge.Unsorted().SortKeys(); len(got) != 0 {
		t.Errorf("unsorted: %+v", got)
	}
	// grouped, the groups that sort order the rows first: the outermost of those is 1, and a key after them numbers
	// on; a group that does not sort, as a group does not by default, takes no number, so a key under it alone has
	// none; a group keeps how it sorts through Unsorted, and is marked only where it sorts
	g := Panel{Sortable: map[string]bool{"id": true, "age": true}, Groupable: map[string]bool{"age": true}}.
		Grouped("age", true).Sorted("id")
	if place, total := g.Place("age"); g.Mark("id") != "▲" || g.Mark("age") != "" || place != 0 || total != 1 {
		t.Errorf("under a group that does not sort: id %q, age %q at %d of %d", g.Mark("id"), g.Mark("age"), place,
			total)
	}
	if s := g.SortedAs("age", Descending); s.Mark("age") != "▼₁" || s.Mark("id") != "▲²" {
		t.Errorf("under a group that sorts: id %q, age %q", s.Mark("id"), s.Mark("age"))
	}
	if g = g.SortedAs("age", Descending).Unsorted(); g.Mark("age") != "▼" || len(g.ColumnKeys()) != 0 {
		t.Errorf("a group sorting, unsorted: %q %+v", g.Mark("age"), g.SortKeys())
	}
	if place, _ := g.Place("id"); place != 0 {
		t.Errorf("id, not sorting, at %d", place)
	}
	// each key is the field's sort, at its priority
	if f := byAge.Field("id"); f.Sort != Ascending || f.Priority != 2 || byAge.Field("age").Priority != 1 {
		t.Errorf("the fields' sorts: %+v", byAge.Fields)
	}
	if got := p.Sorted("title"); got.SortKeys() != nil {
		t.Errorf("a column that cannot sort: %+v", got.SortKeys())
	}
	if natural("a2", "a10") >= 0 || natural("B", "a") <= 0 || natural("ab", "abc") >= 0 || natural("x", "x") != 0 {
		t.Error("natural")
	}
}

// HeaderAt finds the column whose name a cell of the header falls on, and none off the names.
func TestHeaderAtFindsTheColumnUnderAName(t *testing.T) {
	rows := []Row{{Cells: []Cell{one("#31"), one(""), one("ready"), {Values: []string{"a title"}, Wrap: true}}}}
	names := []string{"id", "fork", "state", "title"}
	header, _ := Header(rows, names, 40, false)
	plain := ansi.Strip(header[0])
	for name, want := range map[string]int{"id": 0, "state": 2, "title": 3} {
		if got := HeaderAt(rows, names, 40, strings.Index(plain, name)); got != want {
			t.Errorf("%s: column %d, want %d", name, got, want)
		}
	}
	if HeaderAt(rows, names, 40, 1) != -1 || HeaderAt(rows, names, 40, 39) != -1 || HeaderAt(nil, names, 40, 5) != -1 {
		t.Error("a column off the names")
	}
}

// HeaderName says where the header draws a column's name, and its first line: from the cell's left, or against its
// right for a column that aligns right; none for a column with no name, or none drawn.
func TestHeaderNameSaysWhereANameIsDrawn(t *testing.T) {
	rows := []Row{{Cells: []Cell{one("#31"), one(""), {Values: []string{"40d"}, Right: true},
		{Values: []string{"a title"}, Wrap: true}}}}
	names := []string{"id", "", "the age", "title"}
	header, _ := Header(rows, names, 40, false)
	plain := ansi.Strip(header[0])
	for j, want := range map[int]string{0: "id", 2: "the age", 3: "title"} {
		x, first := HeaderName(rows, names, 40, j)
		if first != want || !strings.HasPrefix(plain[x:], want) {
			t.Errorf("column %d: %q at %d in %q", j, first, x, plain)
		}
	}
	if x, first := HeaderName(rows, names, 40, 1); x != -1 || first != "" {
		t.Errorf("a column with no name: %q at %d", first, x)
	}
	if x, _ := HeaderName(rows, names, 40, 9); x != -1 {
		t.Errorf("a column past the rows': %d", x)
	}
}

// Grouping puts the rows under a dim heading per value, the groups in their values' order, the rows in each as they
// sorted; a second column nests under the first; a cell with no value groups under none; a row of one cell follows
// the groups. A groupable column moved into the groups groups them, ▤ beside its name; a column that cannot group
// changes nothing.
func TestThePanelGroupsTheRows(t *testing.T) {
	columns := []string{"id", "state", "who"}
	row := func(id, state, who string) Row { return Row{Key: id, Cells: []Cell{one(id), one(state), one(who)}} }
	rows := []Row{row("#3", "ready", "ada"), row("#1", "draft", "bob"), row("#2", "ready", ""),
		{Key: "gone", Cells: []Cell{one("gone: x")}}}
	p := Panel{Groupable: map[string]bool{"state": true, "who": true}}.Grouped("state", false).Grouped("who", true)
	var got []string
	grouped := p.group(rows, columns)
	for _, r := range grouped {
		text := r.Key
		if r.Aside {
			text = ansi.Strip(r.Cells[0].first())
		}
		got = append(got, text)
	}
	// draft's one heading, bob's, heads the same row, so it stands on draft's line, across
	want := []string{"▾ state ▾ who ▾ 3", "▾ draft ▾ bob 1", "#1", "▾ ready 2", "  ▾ ada 1", "#3", "  ▾ none 1", "#2",
		"gone"}
	if !slices.Equal(got, want) {
		t.Errorf("grouped:\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
	// the heading over the groups names their columns, a level over their values, which say no column; the levels a
	// screen indents by are the columns', the heading over them apart
	if !OverGroups(grouped[0]) || OverGroups(grouped[1]) || Level(grouped[0]) != 1 || Level(grouped[3]) != 2 ||
		Level(grouped[4]) != 3 || Levels(grouped) != 2 {
		t.Errorf("the levels: %d %d %d of %d", Level(grouped[0]), Level(grouped[3]), Level(grouped[4]), Levels(grouped))
	}
	if q := p.Grouped("state", true); !slices.Equal(q.Groups(), []string{"who"}) || q.Field("state").Placement != Column ||
		q.Field("who").Depth != 1 || p.Grouped("id", false).Groups()[0] != "state" {
		t.Errorf("ungrouping, or a column that cannot group: %q", q.Groups())
	}
	// a group hidden leaves the groups, those under it moving out a level
	if q := p.Placed("state", Hidden); !slices.Equal(q.Groups(), []string{"who"}) || q.Field("who").Depth != 1 {
		t.Errorf("state hidden: %q %+v", q.Groups(), q.Fields)
	}
	// sorting a grouped field orders its groups, the larger first descending
	desc := p.with(func(fields map[string]Field) {
		fields["state"] = Field{Placement: Group, Depth: 1, Sort: Descending, Priority: 1}
	})
	if got := desc.group(rows, columns); ansi.Strip(got[1].Cells[0].first()) != "▾ ready 2" {
		t.Errorf("the groups descending: %q", ansi.Strip(got[1].Cells[0].first()))
	}
	if got := (Panel{}).group(rows, columns); len(got) != len(rows) {
		t.Error("grouping by nothing changed the rows")
	}
	q := Panel{Columns: columns, Groupable: p.Groupable}
	q, _ = q.Step("state", -1)
	if q, _ = q.Step("state", -1); !slices.Equal(q.Groups(), []string{"state"}) {
		t.Errorf("state moved up into the groups: %q", q.Groups())
	}
}

// The rows group by a cell's sort key, its raw value, and the heading says the value as the rows draw it, or as the
// cell's Head says it where the rows draw more than they group by.
func TestAGroupIsTheRawValueAndItsHeadingTheDrawnOne(t *testing.T) {
	columns := []string{"id", "state", "checks"}
	row := func(id, state, checks string) Row {
		return Row{Key: id, Cells: []Cell{one(id), {Values: []string{"🏁 " + state}, Raw: []string{state}},
			{Values: []string{"✅ " + checks}, Raw: []string{"SUCCESS"}, Head: "✅ SUCCESS"}}}
	}
	rows := []Row{row("#1", "ready", "1 / 1"), row("#2", "draft", "2 / 2"), row("#3", "ready", "3 / 3")}
	headings := func(p Panel) []string {
		got := p.group(rows, columns)
		var out []string
		for _, r := range got {
			if r.Aside {
				out = append(out, ansi.Strip(r.Cells[0].first()))
			}
		}
		return out
	}
	p := Panel{Groupable: map[string]bool{"state": true, "checks": true}, Fold: true}
	if got := headings(p.Grouped("state", true)); !slices.Equal(got, []string{"▾ state ▾ 3", "▾ 🏁 draft 1", "▾ 🏁 ready 2"}) {
		t.Errorf("the headings by state: %q", got)
	}
	// grouped by checks, the rows group by its raw value, every row's SUCCESS, the heading saying its Head
	if got := headings(p.Grouped("checks", false)); !slices.Equal(got, []string{"▾ checks ▾ 3", "▾ ✅ SUCCESS 3"}) {
		t.Errorf("the heading by checks: %q", got)
	}
}

// While the panel folds and groups by nothing, a view as it loads groups the rows by each column that can group whose
// value is the same on every row it lets through, left to right in the panel's order: groups as any other, which the
// panel then moves and ungroups as it would. A column blank on some rows, one with several values, a hidden column,
// a column that cannot group, and the rows a filter or the search holds back count for nothing; one row alone, a panel
// that groups already, or one that does not fold, is given back as it is.
func TestAViewAsItLoadsGroupsByEachColumnTheSameOnEveryRow(t *testing.T) {
	columns := []string{"id", "source", "state", "where", "who"}
	row := func(id, state, where, who string) Row {
		return Row{Key: id, Cells: []Cell{one(id), one("github"), one(state), one(where), one(who)}}
	}
	groupable := map[string]bool{"source": true, "state": true, "where": true, "who": true}
	rows := []Row{row("#1", "draft", "here", "ada"), row("#2", "draft", "here", ""), row("#3", "ready", "here", "bob"),
		{Key: "gone", Cells: []Cell{one("gone: x")}}}
	p := Panel{Columns: columns, Groupable: groupable, Fold: true}
	if got := p.AutoGrouped(rows, columns).Groups(); !slices.Equal(got, []string{"source", "where"}) {
		t.Fatalf("grouped as it loads: %q", got)
	}
	// they are groups as any other: drawn so, ungrouped and moved as any other
	q := p.AutoGrouped(rows, columns)
	var headings []string
	for _, r := range q.group(rows, columns) {
		if r.Aside {
			headings = append(headings, ansi.Strip(r.Cells[0].first()))
		}
	}
	if want := []string{"▾ source ▾ where ▾ 3", "▾ github ▾ here 3"}; !slices.Equal(headings, want) {
		t.Errorf("the headings: %q", headings)
	}
	if got := q.Grouped("source", true).Groups(); !slices.Equal(got, []string{"where"}) {
		t.Errorf("source ungrouped: %q", got)
	}
	// left to right, as the panel orders the columns
	if moved, _ := p.Moved("where", Column, 0); !slices.Equal(moved.AutoGrouped(rows, columns).Groups(),
		[]string{"where", "source"}) {
		t.Errorf("where moved first: %q", moved.AutoGrouped(rows, columns).Groups())
	}
	// a hidden column, or one that cannot group, does not group
	if got := p.Placed("source", Hidden).AutoGrouped(rows, columns).Groups(); !slices.Equal(got, []string{"where"}) {
		t.Errorf("source hidden: %q", got)
	}
	r := p
	r.Groupable = map[string]bool{"source": true}
	if got := r.AutoGrouped(rows, columns).Groups(); !slices.Equal(got, []string{"source"}) {
		t.Errorf("where cannot group: %q", got)
	}
	// the rows the search or a filter holds back count for nothing: with the drafts alone, state is the same on every
	// row too; with one row alone, nothing groups
	f := p
	f.Query = "draft"
	if got := f.AutoGrouped(rows, columns).Groups(); !slices.Equal(got, []string{"source", "state", "where"}) {
		t.Errorf("the drafts alone: %q", got)
	}
	if f.Query = "ready"; len(f.AutoGrouped(rows, columns).Groups()) != 0 {
		t.Errorf("one row through the search: %q", f.AutoGrouped(rows, columns).Groups())
	}
	// two values in a cell, or one row alone, do not group; nor does a panel that groups already, or does not fold
	several := slices.Clone(rows)
	several[0].Cells = slices.Clone(several[0].Cells)
	several[0].Cells[3] = Cell{Values: []string{"here", "there"}}
	if got := p.AutoGrouped(several, columns).Groups(); !slices.Equal(got, []string{"source"}) {
		t.Errorf("several values: %q", got)
	}
	if got := p.AutoGrouped(rows[:1], columns).Groups(); len(got) != 0 {
		t.Errorf("one row: %q", got)
	}
	if got := p.Grouped("state", false).AutoGrouped(rows, columns).Groups(); !slices.Equal(got, []string{"state"}) {
		t.Errorf("a panel that groups: %q", got)
	}
	if got := (Panel{Groupable: groupable}).AutoGrouped(rows, columns).Groups(); len(got) != 0 {
		t.Errorf("not folding: %q", got)
	}
	// grouping by nothing, the rows stand as they came
	if got := p.group(rows, columns); len(got) != len(rows) || Levels(got) != 0 {
		t.Errorf("grouped by nothing: %d rows, %d levels", len(got), Levels(got))
	}
}

// Heading names the column whose group a row heads, at its depth, and nothing for a row that heads none, nor for the
// heading over the groups.
func TestAHeadingNamesTheColumnItGroups(t *testing.T) {
	p := Panel{Groupable: map[string]bool{"source": true, "who": true}}.Grouped("source", false).Grouped("who", true)
	rows := p.group([]Row{{Key: "#1", Cells: []Cell{{Values: []string{"github"}}, {Values: []string{"ada"}}}},
		{Key: "#2", Cells: []Cell{{Values: []string{"github"}}, {Values: []string{"bob"}}}}}, []string{"source", "who"})
	var got []string
	for _, r := range rows {
		got = append(got, p.Heading(r))
	}
	// the heading over the groups heads no one column: it names them all
	if !slices.Equal(got, []string{"", "source", "who", "", "who", ""}) {
		t.Errorf("the headings: %q", got)
	}
	if h := p.Grouped("who", true).Heading(rows[2]); h != "" {
		t.Errorf("a heading deeper than the groups: %q", h)
	}
}

// A group's heading folds its rows away, and the headings under it, up to the next heading as shallow; its ▾ turns ▸,
// and its count stays the rows it heads. The rows of one cell after the groups never fold.
func TestAGroupsHeadingFoldsItsRowsAway(t *testing.T) {
	columns := []string{"id", "state", "who"}
	row := func(id, state, who string) Row { return Row{Key: id, Cells: []Cell{one(id), one(state), one(who)}} }
	rows := []Row{row("#3", "ready", "ada"), row("#1", "draft", "bob"), row("#2", "ready", ""),
		{Key: "gone", Cells: []Cell{one("gone: x")}}}
	p := Panel{Groupable: map[string]bool{"state": true, "who": true}}.Grouped("state", false).Grouped("who", true)
	texts := func(p Panel) []string {
		grouped := p.group(rows, columns)
		var out []string
		for _, r := range p.FoldAway(grouped) {
			text := r.Key
			if r.Aside {
				text = ansi.Strip(r.Cells[0].first())
			}
			out = append(out, text)
		}
		return out
	}
	ready := byFields + "\x00ready"
	// shut gives the panel with the groups of the keys closed in its outline
	shut := func(p Panel, keys ...string) Panel {
		p.Outline = Outline{}
		for _, k := range keys {
			p.Outline[k] = false
		}
		return p
	}
	want := []string{"▾ state ▾ who ▾ 3", "▾ draft ▾ bob 1", "#1", "▸ ready 2", "gone"}
	if got := texts(shut(p, ready)); !slices.Equal(got, want) {
		t.Errorf("ready folded:\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
	// a nested heading folds only its own rows
	want = []string{"▾ state ▾ who ▾ 3", "▾ draft ▾ bob 1", "#1", "▾ ready 2", "  ▸ ada 1", "  ▾ none 1", "#2",
		"gone"}
	if got := texts(shut(p, ready+"\x00ada")); !slices.Equal(got, want) {
		t.Errorf("ada folded:\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
	// a closed heading holds the rows it folds away, which still lay out the header and the columns
	every := Panel{Groupable: p.Groupable}.Grouped("state", false)
	grouped := every.group(rows, columns)
	names := []string{"id", "state", "who"}
	open, _ := Header(grouped, names, 60, false)
	folded := shut(every, byFields+"\x00draft", ready).FoldAway(grouped)
	if closed, _ := Header(folded, names, 60, false); len(folded) != 4 || len(folded[2].Folded) != 2 ||
		!slices.Equal(closed, open) || !slices.Equal(headed(folded, names), headed(grouped, names)) {
		t.Errorf("every group closed: %d rows, %d folded under ready, header %q, was %q", len(folded),
			len(folded[2].Folded), closed, open)
	}
	if HeaderAt(folded, names, 60, 3) != HeaderAt(grouped, names, 60, 3) {
		t.Error("a click on the header of closed groups finds another column")
	}
	// opened again, every row is back
	if q := shut(p); len(texts(q)) != 9 {
		t.Errorf("unfolded: %q", texts(q))
	}
	// the whole outline puts each closed group's rows back in their places
	grouped = p.group(rows, columns)
	if got := Every(shut(p, ready).FoldAway(grouped)); len(got) != len(grouped) ||
		slices.ContainsFunc(got, func(r Row) bool { return r.Folded != nil }) {
		t.Errorf("every row of the outline: %d of %d", len(got), len(grouped))
	}
}

// The outline's tree: a heading's level, its span over the rows and headings under it, and each node's parent, the
// heading it stands under; the rows of one cell after the groups stand under none. A heading stands open until it is
// closed, a row closed until it is opened; a heading's ▸ is dim, as a row's is.
func TestTheOutlinesTreeAndWhatStandsOpen(t *testing.T) {
	columns := []string{"id", "state", "who"}
	row := func(id, state, who string) Row { return Row{Key: id, Cells: []Cell{one(id), one(state), one(who)}} }
	rows := []Row{row("#3", "ready", "ada"), row("#1", "draft", "bob"), row("#2", "ready", ""),
		{Key: "gone", Cells: []Cell{one("gone: x")}}}
	p := Panel{Groupable: map[string]bool{"state": true, "who": true}}.Grouped("state", false).Grouped("who", true)
	grouped := p.group(rows, columns)
	// state and who, draft with bob on its line, #1, ready, ada, #3, none, #2, gone
	var levels, spans, parents []int
	for k, r := range grouped {
		levels, spans, parents = append(levels, Level(r)), append(spans, Span(grouped, k)), append(parents,
			Parent(grouped, k))
	}
	if !slices.Equal(levels, []int{1, 2, 0, 2, 3, 0, 3, 0, 0}) ||
		!slices.Equal(spans, []int{8, 3, 3, 8, 6, 6, 8, 8, 9}) ||
		!slices.Equal(parents, []int{-1, 0, 1, 0, 3, 4, 3, 6, -1}) {
		t.Errorf("levels %v, spans %v, parents %v", levels, spans, parents)
	}
	o := Outline{"#1": true, grouped[3].Key: false}
	if !o.Open(grouped[0]) || o.Open(grouped[3]) || !o.Open(grouped[2]) || o.Open(grouped[5]) {
		t.Error("what stands open")
	}
	p.Outline = o
	if got := p.fold(grouped[3].Key); got != style.Off(Folded) || p.fold(grouped[0].Key) != Unfolded {
		t.Errorf("the glyphs: %q", got)
	}
}

// A field's popover sets its sort, its filter, and its rendering apart: SortedAs keeps a key's place among the keys in
// a new direction, adds one after the others, and takes one away; a field that cannot sort changes nothing. The
// panel's map is never the one it was given.
func TestAFieldsPopoverSetsItsSortFilterAndRendering(t *testing.T) {
	p := Panel{Sortable: map[string]bool{"id": true, "age": true}}.Sorted("age").Sorted("id")
	given := p.Fields
	if q := p.SortedAs("age", Descending); q.Mark("age") != "▼₁" || q.Mark("id") != "▲²" || given["age"].Sort != Ascending {
		t.Errorf("age descending: %q %q", q.Mark("age"), q.Mark("id"))
	}
	if q := p.SortedAs("age", Unsorted); len(q.SortKeys()) != 1 || q.Mark("id") != "▲" {
		t.Errorf("age off: %+v", q.SortKeys())
	}
	if q := p.SortedAs("age", Unsorted).SortedAs("age", Ascending); q.Mark("age") != "▲²" {
		t.Errorf("age back on: %q", q.Mark("age"))
	}
	if q := p.SortedAs("title", Ascending); len(q.SortKeys()) != 2 {
		t.Errorf("title, which cannot sort: %+v", q.SortKeys())
	}
	values := []string{Caller}
	q := p.FilteredBy("who", Filter{Values: values, Exclude: true}).Rendered("who", "text")
	values[0] = "bob"
	if f := q.Field("who"); !slices.Equal(f.Filter.Values, []string{Caller}) || !f.Filter.Exclude ||
		f.Rendering != "text" || !q.Filtering() || p.Filtering() {
		t.Errorf("who's filter and rendering: %+v", f)
	}
}

// Shown and Showing agree: each choice of Show sets the owner's filter so that Showing says the same choice again.
func TestShowingSaysTheChoiceShownSet(t *testing.T) {
	p := Panel{Owner: "who"}
	for _, s := range []Show{Mine, Others, All} {
		if got := p.Shown(s).Showing(); got != s {
			t.Errorf("shown %v, showing %v", s, got)
		}
	}
}

// An open row breaks a value that starts with a glyph with a hanging indent: the lines after its first start under the
// text, not under the glyph, and a dim value stays dim on each. A first word with a letter or a digit is no glyph, a
// cell that says nothing of glyphs hangs nothing, and a column too narrow for ten cells after the glyph breaks as ever.
func TestAnOpenRowHangsTheLinesAfterAGlyph(t *testing.T) {
	plain := func(lines []string) []string {
		out := make([]string, len(lines))
		for k, l := range lines {
			out[k] = ansi.Strip(l)
		}
		return out
	}
	title := "🎫 Every tracker in the picture, grouped by tracker"
	if got := plain(wrapped(Cell{Values: []string{title}, Hang: true}, 24)); !slices.Equal(got,
		[]string{"🎫 Every tracker in the", "   picture, grouped by", "   tracker"}) {
		t.Errorf("a title hung: %q", got)
	}
	if got := plain(wrapped(Cell{Values: []string{title}}, 24)); !slices.Equal(got,
		[]string{"🎫 Every tracker in the", "picture, grouped by", "tracker"}) {
		t.Errorf("a cell that hangs nothing: %q", got)
	}
	if got := plain(wrapped(Cell{Values: []string{"#45 Every tracker in the picture"}, Hang: true}, 16)); strings.HasPrefix(got[1], " ") {
		t.Errorf("a first word with a digit: %q", got)
	}
	if got := plain(wrapped(Cell{Values: []string{title}, Hang: true}, 12)); strings.HasPrefix(got[1], " ") {
		t.Errorf("too narrow to hang: %q", got)
	}
	if got := wrapped(Cell{Values: []string{"🎫 short"}, Hang: true}, 24); !slices.Equal(got, []string{"🎫 short"}) {
		t.Errorf("a value that fits: %q", got)
	}
	dim := wrapped(Cell{Values: []string{style.Off(title)}, Hang: true}, 24)
	for _, l := range dim[1:] {
		if !strings.HasPrefix(l, "   ") || !strings.Contains(l, "\x1b[") {
			t.Errorf("a dim line, unhung or undimmed: %q", l)
		}
	}
	// a value in a column of its own width breaks the same way, at a dash or a slash too
	if got := plain(hung("🌿 46-a-noun-lists-a-noun-with-a-verb", true, 16, broken)); len(got) < 2 ||
		!strings.HasPrefix(got[0], "🌿 46-") || !strings.HasPrefix(got[1], "   ") {
		t.Errorf("a branch hung: %q", got)
	}
	rows := []Row{{Key: "a", Cells: []Cell{one("#1"), {Values: []string{"👤 jeffrey-aguilera-the-long"}, Hang: true},
		{Values: []string{title}, Wrap: true, Hang: true}}}}
	if got := plain(openLines(rows[0].Cells, []int{2, 14}, 24, 44, false)); len(got) < 2 || !strings.Contains(got[1], "   ") {
		t.Errorf("an open row: %q", got)
	}
}

// The columns dialog lists the groups, the outermost first, then the columns as they stand, then the hidden; the
// search takes what is typed and backspace, and leaves any other key.
func TestThePanelListsItsFieldsAndTypesInItsSearch(t *testing.T) {
	columns := []string{"id", "state", "who", "title"}
	p := Panel{Columns: columns, Groupable: map[string]bool{"state": true}, Fields: map[string]Field{
		"state": {Placement: Group, Depth: 1}, "who": {Placement: Hidden}, "title": {Placement: Column, Position: 1}}}
	if got := p.Listed(); !slices.Equal(got, []string{"state", "title", "id", "who"}) {
		t.Errorf("listed: %q", got)
	}
	if p = p.Find(); !p.Focused {
		t.Error("Find left the search without the keyboard")
	}
	p, _ = p.Update(tea.KeyPressMsg{Code: 'a', Text: "a"})
	p, _ = p.Update(tea.KeyPressMsg{Code: 'b', Text: "b"})
	if p, _ = p.Update(tea.KeyPressMsg{Code: tea.KeyBackspace}); p.Query != "a" {
		t.Errorf("typed ab and backspace: %q", p.Query)
	}
	if _, ok := p.Update(tea.KeyPressMsg{Code: tea.KeyDown}); ok {
		t.Error("the search took down")
	}
}

// A row's selection takes two cells of the gutter, its mark and a space, so the mark never stands against the ▸ or ▾
// after it; a heading's mark stands in with the heading, after its indent. A row stands where the screen indents it,
// two cells in from the innermost heading, as Levels says.
func TestASelectionMarkStandsApartFromTheFoldAndInWithItsHeading(t *testing.T) {
	columns := []string{"id", "state", "who"}
	row := func(id, state, who string) Row { return Row{Key: id, Cells: []Cell{one(id), one(state), one(who)}} }
	p := Panel{Groupable: map[string]bool{"state": true, "who": true}}.Grouped("state", false).Grouped("who", true)
	grouped := p.group([]Row{row("#1", "ready", "ada"), row("#2", "ready", "bob")}, columns)
	picked := func(r Row) string {
		if Heads(r) {
			return Partly
		}
		return Chosen
	}
	// the heading over the groups is drawn as nothing, as a screen draws it apart
	got := plain(Draw(grouped, nil, 60, -1, func(r Row) bool { return Heads(r) }, picked))
	want := []string{Partly + " ▾ ready 2", "  " + Partly + " ▾ ada 1", Chosen + "   #1  ready  ada"}
	for k, w := range want {
		if !strings.HasPrefix(got[k], w) {
			t.Errorf("line %d: %q, want %q", k, got[k], w)
		}
	}
}

// The none kittui makes up for a row with no value follows every value, the group sorting up or down, of one value or
// several, a cell of several empty values among them; a value that reads none is a value, and sorts as its text; a
// value given twice groups the row once.
func TestNoneFollowsEveryValue(t *testing.T) {
	columns := []string{"id", "who"}
	row := func(id string, who ...string) Row {
		return Row{Key: id, Cells: []Cell{one(id), {Values: slices.Clone(who), Raw: slices.Clone(who)}}}
	}
	rows := []Row{row("#1", "zed"), row("#2"), row("#3", "ada"), row("#4", "none"), row("#5", "yan", "zed"),
		row("#6", "", ""), row("#7", "ada", "ada")}
	texts := func(p Panel) []string {
		var out []string
		for _, r := range p.group(rows, columns) {
			if r.Aside && !OverGroups(r) {
				out = append(out, ansi.Strip(r.Cells[0].first()))
			}
		}
		return out
	}
	p := Panel{Groupable: map[string]bool{"who": true}}.Grouped("who", false)
	if got, want := texts(p), []string{"▾ ada 2", "▾ none 1", "▾ yan 1", "▾ zed 2", "▾ none 2"}; !slices.Equal(got, want) {
		t.Errorf("up: %q, want %q", got, want)
	}
	p.Fields["who"] = Field{Placement: p.Fields["who"].Placement, Depth: p.Fields["who"].Depth, Sort: Descending}
	if got, want := texts(p), []string{"▾ zed 2", "▾ yan 1", "▾ none 1", "▾ ada 2", "▾ none 2"}; !slices.Equal(got, want) {
		t.Errorf("down: %q, want %q", got, want)
	}
}

// A column of several values groups a row under each of its values, the same row under every heading it belongs to,
// a heading saying its value as the row draws it; a row with none stands under none. The heading over the groups
// counts each row once, each value's heading the rows under it. Nested, a row stands under each pair of its values.
func TestARowOfSeveralValuesStandsUnderEachOfThem(t *testing.T) {
	columns := []string{"id", "who", "state"}
	row := func(id, state string, who ...string) Row {
		return Row{Key: id, Cells: []Cell{one(id), {Values: slices.Clone(who), Raw: slices.Clone(who)}, one(state)}}
	}
	rows := []Row{row("#1", "ready", "ada", "bob"), row("#2", "draft", "bob"), row("#3", "ready")}
	texts := func(p Panel) []string {
		var out []string
		for _, r := range p.group(rows, columns) {
			text := r.Key
			if r.Aside {
				text = ansi.Strip(r.Cells[0].first())
			}
			out = append(out, text)
		}
		return out
	}
	p := Panel{Groupable: map[string]bool{"who": true, "state": true}}.Grouped("who", false)
	want := []string{"▾ who ▾ 3", "▾ ada 1", "#1", "▾ bob 2", "#1", "#2", "▾ none 1", "#3"}
	if got := texts(p); !slices.Equal(got, want) {
		t.Errorf("by who:\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
	want = []string{"▾ state ▾ who ▾ 3", "▾ draft ▾ bob 1", "#2", "▾ ready 2", "  ▾ ada 1", "#1", "  ▾ bob 1", "#1",
		"  ▾ none 1", "#3"}
	if got := texts(p.Grouped("state", false).Grouped("who", true)); !slices.Equal(got, want) {
		t.Errorf("by state, then who:\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
	// a heading says the value as the row draws it, its styles kept where the cell's Head says it so, as a label's pill
	rows[0].Cells[1] = Cell{Values: []string{"👤 ada", "👤 bob"}, Raw: []string{"ada", "bob"}}
	if got := texts(p)[1]; got != "▾ 👤 ada 1" {
		t.Errorf("ada's heading: %q", got)
	}
}

// The heading over the groups says, before each column's name, whether that level shows, as the outline's Shows says:
// ▾ while it and every level over it stand open, ▸ while one does not, never dim, as a level's fold always acts; and
// after the last name, whether the rows show. A heading closed by hand changes none of them.
func TestTheHeadingOverTheGroupsFoldsEachLevel(t *testing.T) {
	columns := []string{"id", "state", "who"}
	row := func(id, state, who string) Row { return Row{Key: id, Cells: []Cell{one(id), one(state), one(who)}} }
	rows := []Row{row("#3", "ready", "ada"), row("#1", "draft", "bob"), row("#2", "ready", "")}
	p := Panel{Groupable: map[string]bool{"state": true, "who": true}}.Grouped("state", false).Grouped("who", true)
	bar := func(p Panel) string { return ansi.Strip(p.group(rows, columns)[0].Cells[0].first()) }
	if got := bar(p); got != "▾ state ▾ who ▾ 3" {
		t.Errorf("every level open: %q", got)
	}
	p.Outline = Outline{byFields + "\x00ready\x00ada": false}
	if got := bar(p); got != "▾ state ▾ who ▾ 3" {
		t.Errorf("one of who's headings closed: %q", got)
	}
	p.Outline = Outline{LevelKey(1): false}
	if got := bar(p); got != "▾ state ▸ who ▸ 3" {
		t.Errorf("who's level closed: %q", got)
	}
	if got := p.group(rows, columns)[0].Cells[0].first(); strings.Contains(got, style.Off(Folded)) {
		t.Errorf("a closed level's ▸ is dim, as if it could not act: %q", got)
	}
	// each column's name is bold, as the header draws it
	if got := p.group(rows, columns)[0].Cells[0].first(); !strings.Contains(got, style.Label("state")) ||
		!strings.Contains(got, style.Label("who")) {
		t.Errorf("the names are not bold: %q", got)
	}
	// a level closed shows every level under it closed too, each keeping its own state
	p.Outline = Outline{LevelKey(0): false}
	if got := bar(p); got != "▸ state ▸ who ▸ 3" || !p.Outline.Open(Row{Key: byFields + "\x00ready", Aside: true}) {
		t.Errorf("state's level closed: %q", got)
	}
	p.Outline = Outline{LevelKey(2): false}
	if got := bar(p); got != "▾ state ▾ who ▸ 3" {
		t.Errorf("the rows closed: %q", got)
	}
}

// A level of the groups that does not show folds away its headings and everything under them, the rows' level its
// rows, each heading over them standing, ▸ as it shows nothing; the row before them holds them, the heading over the
// groups where nothing else shows, so the columns lay out as they did. A heading keeps its own state: a level opened
// again shows each as it was.
func TestALevelThatDoesNotShowFoldsAway(t *testing.T) {
	columns := []string{"id", "state", "who"}
	row := func(id, state, who string) Row { return Row{Key: id, Cells: []Cell{one(id), one(state), one(who)}} }
	rows := []Row{row("#3", "ready", "ada"), row("#1", "draft", "bob"), row("#2", "ready", ""),
		{Key: "gone", Cells: []Cell{one("gone: x")}}}
	p := Panel{Groupable: map[string]bool{"state": true, "who": true}}.Grouped("state", false).Grouped("who", true)
	grouped := p.group(rows, columns)
	texts := func(p Panel) []string {
		var out []string
		for _, r := range p.FoldAway(p.group(rows, columns)) {
			text := r.Key
			if r.Aside {
				text = ansi.Strip(r.Cells[0].first())
			}
			out = append(out, text)
		}
		return out
	}
	p.Outline = Outline{LevelKey(2): false}
	if got, want := texts(p), []string{"▾ state ▾ who ▸ 3", "▸ draft ▸ bob 1", "▾ ready 2", "  ▸ ada 1", "  ▸ none 1",
		"gone"}; !slices.Equal(got, want) {
		t.Errorf("the rows closed:\n%s", strings.Join(got, "\n"))
	}
	p.Outline = Outline{LevelKey(1): false, byFields + "\x00ready\x00ada": false}
	if got, want := texts(p), []string{"▾ state ▸ who ▸ 3", "▸ draft 1", "▸ ready 2", "gone"}; !slices.Equal(got, want) {
		t.Errorf("who's level closed:\n%s", strings.Join(got, "\n"))
	}
	p.Outline = Outline{LevelKey(0): false}
	folded := p.FoldAway(grouped)
	if got := texts(p); !slices.Equal(got, []string{"▸ state ▸ who ▸ 3", "gone"}) || len(Every(folded)) != len(grouped) ||
		!slices.Equal(headed(folded, columns), headed(grouped, columns)) {
		t.Errorf("state's level closed: %q", got)
	}
	// opened again, ada's heading is closed still, as it was left
	p.Outline = Outline{byFields + "\x00ready\x00ada": false}
	if got := texts(p); !slices.Contains(got, "  ▸ ada 1") || slices.Contains(got, "#3") {
		t.Errorf("opened again: %q", got)
	}
}

// A heading whose one heading under it heads the same rows takes it onto its line, across, as the heading over the
// groups names its levels, and so on down: one node, the outermost's, whose every fold stands as it does. A heading
// with more than one under it, though each heads as many rows as it does, as a row of several values may, keeps them
// apart; so does one whose next level does not show.
func TestHeadingsOfTheSameRowsStandAcrossOnOneLine(t *testing.T) {
	columns := []string{"id", "state", "where", "who"}
	row := func(id, state, where string, who ...string) Row {
		return Row{Key: id, Cells: []Cell{one(id), one(state), one(where), {Values: who, Raw: who}}}
	}
	rows := []Row{row("#1", "ready", "here", "ada", "bob"), row("#2", "draft", "there", "cy"),
		row("#3", "draft", "there", "cy")}
	p := Panel{Groupable: map[string]bool{"state": true, "where": true, "who": true}}.Grouped("state", false).
		Grouped("where", true).Grouped("who", true)
	texts := func(p Panel) []string {
		var out []string
		for _, r := range p.FoldAway(p.group(rows, columns)) {
			text := r.Key
			if r.Aside {
				text = ansi.Strip(r.Cells[0].first())
			}
			out = append(out, text)
		}
		return out
	}
	want := []string{"▾ state ▾ where ▾ who ▾ 3", "▾ draft ▾ there ▾ cy 2", "#2", "#3", "▾ ready ▾ here 1", "    ▾ ada 1",
		"#1", "    ▾ bob 1", "#1"}
	if got := texts(p); !slices.Equal(got, want) {
		t.Errorf("across:\n%s", strings.Join(got, "\n"))
	}
	// the line is one node, draft's: closed, every fold on it is ▸, and it folds everything under it
	p.Outline = Outline{byFields + "\x00draft": false}
	if got := texts(p); got[1] != "▸ draft ▸ there ▸ cy 2" || got[2] != "▾ ready ▾ here 1" {
		t.Errorf("draft closed: %q", got)
	}
	// where's level closed, nothing joins the line from it
	p.Outline = Outline{LevelKey(1): false}
	if got := texts(p); got[1] != "▸ draft 2" {
		t.Errorf("where closed: %q", got)
	}
}
