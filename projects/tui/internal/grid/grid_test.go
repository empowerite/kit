package grid

import (
	"maps"
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
		{Key: "#1", Cells: []Cell{{Values: []string{"#1"}, Right: true}, {Values: []string{"👤 ada", "👤 bob"}},
			{Values: []string{"2h"}, Right: true}, {Values: []string{"a title long enough to wrap in its column"}, Wrap: true}},
			Owners: []string{"ada", "bob"}},
		{Key: "#22", Cells: []Cell{{Values: []string{"#22"}, Right: true}, {Values: []string{"👤 carol"}},
			{Values: []string{"40d"}, Right: true}, {Values: []string{"short"}, Wrap: true}},
			Owners: []string{"carol"}},
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
		" ▸  #1  👤 ada +1   2h  a title long en…",
		"   #22  👤 carol   40d  short",
		" ▸  #3              1d  leaf",
	}
	if !slices.Equal(got, want) {
		t.Errorf("closed:\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
}

func TestAnOpenRowShowsEveryValueAndWrapsItsTextInItsColumnThenItsDetail(t *testing.T) {
	got := plain(Draw(issues(), nil, 40, -1, func(Row) bool { return true }, nil))
	want := []string{
		" ▾  #1  👤 ada      2h  a title long",
		"        👤 bob          enough to wrap",
		"                        in its column",
		"   #22  👤 carol   40d  short",
		" ▾  #3              1d  leaf",
		"      https://example/3",
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

func TestLinkAtFindsTheLinkedGlyphUnderAColumn(t *testing.T) {
	rows := issues()
	rows[0].Cells[3].Link = "https://example/1"
	rows[0].Cells[0].Link = "https://example/number"
	rows = append(rows, Row{Key: "s", Cells: []Cell{{Values: []string{"🔗 a span"}, Link: "https://example/span"}}})
	// the title starts after the gutter, #1, the assignees, and the age, each with the gap after it
	title := len(lead) + 3 + gap + ansi.StringWidth("👤 ada +1") + gap + 3 + gap
	for x, want := range map[int]string{title: "https://example/1", title + 1: "https://example/1", title + 2: "",
		len(lead) + 1: "https://example/number", len(lead): "", 0: ""} {
		if got := LinkAt(rows, nil, 0, 40, x); got != want {
			t.Errorf("at %d: %q, want %q", x, got, want)
		}
	}
	if got := LinkAt(rows, nil, 3, 40, len(lead)); got != "https://example/span" {
		t.Errorf("a span's glyph: %q", got)
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
	if !slices.Equal(closed, []string{" ▸ a heading far to…"}) || len(open) != 2 || open[1] != "   long for the pane" {
		t.Errorf("closed %q, open %q", closed, open)
	}
}

func TestThePanelDrawsItsControlsOneALine(t *testing.T) {
	p := Panel{Columns: []string{"number", "who", "", "title"}, Hidden: map[string]bool{"who": true}, Owned: true}
	lines, at := p.Lines()
	want := []string{"🔎 search", "", "columns", "☑ number", "☐ who", "☑ title", "", "show", "⦿ all", "⦾ mine", "⦾ others"}
	got := make([]string, len(lines))
	for i, l := range lines {
		got[i] = ansi.Strip(l)
	}
	if !slices.Equal(got, want) || !slices.Equal(at, []int{0, -1, -1, 1, 2, 3, -1, -1, 4, 5, 6}) {
		t.Errorf("the panel:\n%s\n%v", strings.Join(got, "\n"), at)
	}
	// what is selected is bold in the accent, a ticked column as the picked choice; the others' boxes are dim
	if lines[3] != style.Mark("☑ number") || lines[4] != style.Off("☐")+" who" || lines[8] != style.Mark("⦿ all") {
		t.Errorf("the selected controls are not marked: %q", lines[3:9])
	}
	// without the caller's login, mine and others are dim, and cannot be chosen
	if !strings.Contains(lines[9], style.Off("⦾ mine")) {
		t.Errorf("mine is not dim without a login: %q", lines[9])
	}
	if q, ok := p.Click(9); !ok || q.Show != All {
		t.Errorf("mine was chosen without a login: %v, %v", q.Show, ok)
	}
	// the width keeps room for the hints the panel shows with the keyboard
	if p.Width() != ansi.StringWidth("⇧↑↓  order, or drag") {
		t.Errorf("width %d", p.Width())
	}
	// with the keyboard, dim hints under the columns say what a column's keys do, the sort's and the groups' only
	// where a column can sort or group
	p.Focused, p.Cursor = true, 0
	for _, c := range []struct {
		sortable, groupable map[string]bool
		want                []string
	}{
		{nil, nil, []string{"", "⇧↑↓  order, or drag"}},
		{map[string]bool{"number": true}, map[string]bool{"who": true},
			[]string{"", "s/S  sort, then by", "g/G  group, within", "⇧↑↓  order, or drag"}},
	} {
		p.Sortable, p.Groupable = c.sortable, c.groupable
		lines, at := p.Lines()
		got := make([]string, len(c.want))
		for i := range got {
			got[i] = ansi.Strip(lines[6+i])
		}
		if !slices.Equal(got, c.want) || at[6] != -1 || ansi.Strip(lines[7+len(c.want)]) != "show" {
			t.Errorf("the hints: %q", got)
		}
	}
}

func TestThePanelTakesKeysAndClicks(t *testing.T) {
	p := Panel{Columns: []string{"number", "who"}, Owned: true, Me: "ada"}.Find()
	key := func(code rune, text string) tea.KeyPressMsg { return tea.KeyPressMsg{Code: code, Text: text} }
	for _, r := range "ad" {
		p, _ = p.Update(key(r, string(r)))
	}
	if p, _ = p.Update(key(tea.KeyBackspace, "")); p.Query != "a" {
		t.Fatalf("typing and backspace: %q", p.Query)
	}
	if lines, _ := p.Lines(); !strings.Contains(lines[0], "🔎 a▏") {
		t.Errorf("the search under the keyboard has no caret: %q", lines[0])
	}
	// down to who, space hides it, again shows it; the panel's map is never the one it was given
	given := map[string]bool{}
	p.Hidden = given
	p, _ = p.Update(key(tea.KeyDown, ""))
	p, _ = p.Update(key(tea.KeyDown, ""))
	if p, _ = p.Update(key(tea.KeySpace, " ")); !p.Hidden["who"] || given["who"] {
		t.Errorf("space did not hide who, or changed the given map: %v %v", p.Hidden, given)
	}
	// up stops at the search, down at mine and others' last
	for range 10 {
		p, _ = p.Update(key(tea.KeyDown, ""))
	}
	if p, _ = p.Update(key(tea.KeyEnter, "")); p.Show != Others {
		t.Errorf("enter on the last control: show %v", p.Show)
	}
	if q, ok := p.Click(1); ok || q.Cursor != p.Cursor {
		t.Errorf("a click on a gap was taken")
	}
	// with the keyboard, the hints under the columns push the show choices down: a gap, the order's hint, a gap
	if q, ok := p.Click(9); !ok || q.Show != All || q.Cursor != 3 {
		t.Errorf("a click on all: show %v, cursor %d", q.Show, q.Cursor)
	}
	if _, ok := p.Update(key('x', "x")); ok {
		t.Error("a letter off the search was taken")
	}
	// / off the search goes to it; on it, / is typed
	if p, _ = p.Update(key('/', "/")); p.Cursor != 0 {
		t.Errorf("/ off the search: cursor %d", p.Cursor)
	}
	if p, _ = p.Update(key('/', "/")); !strings.HasSuffix(p.Query, "/") {
		t.Errorf("/ on the search: %q", p.Query)
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
	p := Panel{Me: "ada"}
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
	if p.Show = Mine; !slices.Equal(keys(p.Filter(issues(), columns)), []string{"#1"}) {
		t.Errorf("mine: %v", keys(p.Filter(issues(), columns)))
	}
	if p.Show = Others; !slices.Equal(keys(p.Filter(issues(), columns)), []string{"#22", "#3"}) {
		t.Errorf("others: %v", keys(p.Filter(issues(), columns)))
	}
	// a hidden column is left out of every row
	p = Panel{Hidden: map[string]bool{"who": true}}
	got := plain(Draw(p.Filter(issues(), columns), nil, 40, -1, func(Row) bool { return false }, nil))
	if strings.Contains(strings.Join(got, "\n"), "👤") || !strings.HasPrefix(got[1], "   #22  40d  short") {
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
	if !hides || len(closed) != 1 || plain[0] != " ▸  id  changes  the title of the issue" || strings.Contains(plain[0], "fork") {
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
	if plain := strip(open); len(open) != 2 || !strings.HasPrefix(plain[0], " ▾  id") || !strings.HasPrefix(plain[1], "     ") ||
		!strings.HasSuffix(plain[1], "it closes") {
		t.Errorf("the open header:\n%s", strings.Join(plain, "\n"))
	}
	if got, hides := Header(rows, names[:3], 40, false); hides || strings.HasPrefix(ansi.Strip(got[0]), " ▸") {
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
	if got != "   #1  short        ready" {
		t.Errorf("a short title: %q", got)
	}
	rows[1].Cells[1].Values[0] = "a title of thirty cells, wide."
	if got := ansi.Strip(Draw(rows, nil, 80, -1, func(Row) bool { return false }, nil)[0].Text); got != "   #1  short"+strings.Repeat(" ", 27)+"ready" {
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
// follow; Move takes a column to another's place; shift with up or down moves the column under the keyboard, the
// keyboard with it, and stops at the columns' ends.
func TestThePanelOrdersTheColumns(t *testing.T) {
	columns := []string{"", "a", "b", "c"}
	p := Panel{Columns: columns, Order: []string{"c", "a"}}
	if got := p.Arrange(columns); !slices.Equal(got, []string{"", "c", "a", "b"}) {
		t.Errorf("arranged: %q", got)
	}
	r := p.hide(Row{Cells: []Cell{one("0"), one("A"), one("B"), one("C")}}, columns)
	if got := r.Cells[1].first() + r.Cells[2].first() + r.Cells[3].first(); r.Cells[0].first() != "0" || got != "CAB" {
		t.Errorf("the row's cells: %q", got)
	}
	if got := (Panel{Columns: columns}).Move("c", "a").Arrange(columns); !slices.Equal(got, []string{"", "c", "a", "b"}) {
		t.Errorf("c moved to a: %q", got)
	}
	if got := (Panel{Columns: columns}).Move("a", "nonesuch"); got.Order != nil {
		t.Errorf("a move to no column: %q", got.Order)
	}
	shift := func(code rune) tea.KeyPressMsg { return tea.KeyPressMsg{Code: code, Mod: tea.ModShift} }
	q := Panel{Columns: columns, Focused: true, Cursor: 1}
	if q, _ = q.Update(shift(tea.KeyDown)); q.Cursor != 2 || !slices.Equal(q.Arrange(columns), []string{"", "b", "a", "c"}) {
		t.Errorf("shift-down: cursor %d, %q", q.Cursor, q.Arrange(columns))
	}
	q, _ = q.Update(shift(tea.KeyUp))
	if q, _ = q.Update(shift(tea.KeyUp)); q.Cursor != 1 || q.Arrange(columns)[1] != "a" {
		t.Errorf("shift-up past the first column: cursor %d, %q", q.Cursor, q.Arrange(columns))
	}
	// on a line that draws no column there is none to hold
	if p.ColumnAt(0) != "" || p.ColumnAt(99) != "" {
		t.Error("a column where none is")
	}
	if _, ok := p.Point(-1); ok {
		t.Error("Point on no line")
	}
}

// Rows sort by the panel's keys, each in turn: by a cell's sort value when it has one, else its text, naturally, so
// #7 comes before #31; descending reverses a key; rows alike keep their order, and a row of one cell its place. A click
// sorts by a column alone, ▲, ▼, then not at all; a shift-click adds it after the others; a column that cannot sort
// changes nothing.
func TestThePanelSortsTheRows(t *testing.T) {
	columns := []string{"id", "age"}
	row := func(id, age, secs string) Row {
		return Row{Key: id, Cells: []Cell{one(id), {Values: []string{age}, Sort: secs}}}
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
	if got := keys(p.Sorted("id", false).order(rows, columns)); got != "#7 #31 heading #100" {
		t.Errorf("by id: %s", got)
	}
	if got := keys(p.Sorted("id", false).Sorted("id", false).order(rows, columns)); got != "#100 #31 heading #7" {
		t.Errorf("by id, descending: %s", got)
	}
	// by age, then id: the two of 60m by id
	byAge := p.Sorted("age", false).Sorted("id", true)
	if got := keys(byAge.order(rows, columns)); got != "#7 #100 heading #31" || byAge.Mark("age") != "▲₁" || byAge.Mark("id") != "▲₂" {
		t.Errorf("by age, then id: %s, marks %q %q", got, byAge.Mark("age"), byAge.Mark("id"))
	}
	// a click on a key among several sorts by it alone, as it was; a third click on a column stops it
	if got := byAge.Sorted("id", false); len(got.Sort) != 1 || got.Sort[0] != (SortKey{"id", true}) || got.Mark("id") != "▼" {
		t.Errorf("a click on id among two keys: %+v", got.Sort)
	}
	if got := p.Sorted("id", false).Sorted("id", false).Sorted("id", false); len(got.Sort) != 0 || got.Mark("id") != "" {
		t.Errorf("three clicks: %+v", got.Sort)
	}
	if got := byAge.Sorted("id", true).Sorted("id", true); len(got.Sort) != 1 || got.Sort[0].Column != "age" {
		t.Errorf("two shift-clicks on id: %+v", got.Sort)
	}
	if got := p.Sorted("title", false); got.Sort != nil {
		t.Errorf("a column that cannot sort: %+v", got.Sort)
	}
	// s and S in the panel, on a column's checkbox, do what a click and a shift-click do
	q := Panel{Columns: columns, Sortable: p.Sortable, Focused: true, Cursor: 1}
	q, _ = q.Update(tea.KeyPressMsg{Code: 's', Text: "s"})
	q, _ = q.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	q, _ = q.Update(tea.KeyPressMsg{Code: 'S', Text: "S"})
	if lines, _ := q.Lines(); len(q.Sort) != 2 || !strings.HasSuffix(ansi.Strip(lines[3]), "id ▲₁") ||
		!strings.HasSuffix(ansi.Strip(lines[4]), "age ▲₂") {
		t.Errorf("s then S: %+v\n%s", q.Sort, strings.Join(lines, "\n"))
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

// Grouping puts the rows under a dim heading per value, the groups in their values' order, the rows in each as they
// sorted; a second column nests under the first; a cell with no value groups under none; a row of one cell follows
// the groups. g on a groupable column's line groups by it, ▤ beside its name, and again stops; a column that cannot
// group changes nothing.
func TestThePanelGroupsTheRows(t *testing.T) {
	columns := []string{"id", "state", "who"}
	row := func(id, state, who string) Row { return Row{Key: id, Cells: []Cell{one(id), one(state), one(who)}} }
	rows := []Row{row("#3", "ready", "ada"), row("#1", "draft", "bob"), row("#2", "ready", ""),
		{Key: "gone", Cells: []Cell{one("gone: x")}}}
	p := Panel{Groupable: map[string]bool{"state": true, "who": true}}.Grouped("state", false).Grouped("who", true)
	var got []string
	grouped, _ := p.group(rows, columns)
	for _, r := range grouped {
		text := r.Key
		if r.Aside {
			text = ansi.Strip(r.Cells[0].first())
		}
		got = append(got, text)
	}
	want := []string{"state · draft  1", "  who · bob  1", "#1", "state · ready  2", "  who · ada  1", "#3",
		"  who · none  1", "#2", "gone"}
	if !slices.Equal(got, want) {
		t.Errorf("grouped:\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
	if q := p.Grouped("state", true); !slices.Equal(q.Group, []string{"who"}) || p.Grouped("id", false).Group[0] != "state" {
		t.Errorf("ungrouping, or a column that cannot group: %q", q.Group)
	}
	if got, _ := (Panel{}).group(rows, columns); len(got) != len(rows) {
		t.Error("grouping by nothing changed the rows")
	}
	q := Panel{Columns: columns, Groupable: p.Groupable, Focused: true, Cursor: 2}
	q, _ = q.Update(tea.KeyPressMsg{Code: 'g', Text: "g"})
	if lines, _ := q.Lines(); !slices.Equal(q.Group, []string{"state"}) || !strings.HasSuffix(ansi.Strip(lines[4]), "state "+Grouping) {
		t.Errorf("g on state: %q\n%s", q.Group, strings.Join(lines, "\n"))
	}
}

// A column that can group, whose value is the same on every row, groups them on its own: without a group the person
// chose, one heading over every row says the values, joined in the panel's order, and their cells are blank. Under a
// group the person chose, the walk goes on in each group, the values joining its heading. A column blank on some rows,
// one with several values, a group of one row, a hidden column, and one kept a column do not group on their own; g
// on such a column in the panel keeps it, and the panel marks one grouping on its own with a dim ▤.
func TestTheRowsGroupOnTheirOwnByAColumnTheSameOnEveryRow(t *testing.T) {
	columns := []string{"id", "source", "state", "where", "who"}
	row := func(id, state, where, who string) Row {
		return Row{Key: id, Cells: []Cell{one(id), one("github"), one(state), one(where), one(who)}}
	}
	groupable := map[string]bool{"source": true, "state": true, "where": true, "who": true}
	texts := func(rows []Row) []string {
		var out []string
		for _, r := range rows {
			text := r.Key
			if r.Aside {
				text = ansi.Strip(r.Cells[0].first())
			}
			for _, c := range r.Cells[1:] {
				if !r.Aside {
					text += "|" + c.first()
				}
			}
			out = append(out, text)
		}
		return out
	}
	rows := []Row{row("#1", "draft", "here", "ada"), row("#2", "draft", "here", ""), row("#3", "ready", "here", "bob")}
	p := Panel{Columns: columns, Groupable: groupable}
	got, auto := p.group(rows, columns)
	want := []string{"source · github, where · here  3", "#1||draft||ada", "#2||draft||", "#3||ready||bob"}
	if !slices.Equal(texts(got), want) || !slices.Equal(auto, []string{"source", "where"}) || Levels(got) != 0 {
		t.Errorf("over every row: %q, auto %q, levels %d", texts(got), auto, Levels(got))
	}
	// the panel's order joins the values; a hidden column and one kept do not group
	if got, _ := (Panel{Groupable: groupable, Order: []string{"where", "source"}}).group(rows, columns); texts(got)[0] != "where · here, source · github  3" {
		t.Errorf("in the panel's order: %q", texts(got)[0])
	}
	if got, auto := (Panel{Groupable: groupable, Hidden: map[string]bool{"source": true}, Kept: map[string]bool{"where": true}}).group(rows, columns); len(got) != 3 || auto != nil {
		t.Errorf("hidden and kept: %q, auto %q", texts(got), auto)
	}
	// under the groups the person chose, each group alone; one of one row does not group
	p.Group = []string{"state"}
	got, auto = p.group(rows, columns)
	// the column the person groups by stays in the cells, which the panel's hide then leaves out
	want = []string{"state · draft, source · github, where · here  2", "#1||draft||ada", "#2||draft||", "state · ready  1",
		"#3|github|ready|here|bob"}
	if !slices.Equal(texts(got), want) || !slices.Equal(auto, []string{"source", "where"}) || Levels(got) != 1 {
		t.Errorf("under state: %q, auto %q, levels %d", texts(got), auto, Levels(got))
	}
	// two values in a cell, or one row alone, do not group
	rows[0].Cells[3].Values = []string{"here", "there"}
	if got, _ := (Panel{Groupable: groupable}).group(rows, columns); texts(got)[0] != "source · github  3" {
		t.Errorf("several values: %q", texts(got)[0])
	}
	if got, auto := (Panel{Groupable: groupable}).group(rows[:1], columns); len(got) != 1 || auto != nil {
		t.Errorf("one row: %q", texts(got))
	}
	// the panel marks a column grouping on its own dim, and g on it keeps it a column
	q := Panel{Columns: columns, Groupable: groupable, Auto: []string{"source"}, Cursor: 2}
	if lines, _ := q.Lines(); !strings.HasSuffix(lines[4], " "+style.Off(Grouping)) {
		t.Errorf("source grouping on its own: %q", lines[4])
	}
	// the headings say which columns they group by on their own, at which level; with more than one column grouping,
	// the panel numbers each by its level, the person's and the rows' own alike
	got, _ = p.group(rows, columns)
	if said := Said(got); !maps.Equal(said, map[string]int{"source": 1}) {
		t.Errorf("said: %v", said)
	}
	if said := Said([]Row{{Key: overAll, Aside: true, Says: []string{"where"}},
		{Key: "group\x00a\x00b", Aside: true, Says: []string{"where", "source"}}}); !maps.Equal(said,
		map[string]int{"where": 1, "source": 2}) {
		t.Errorf("said at two levels: %v", said)
	}
	n := Panel{Columns: columns, Groupable: groupable, Group: []string{"state"}, Auto: []string{"source"},
		AutoAt: map[string]int{"source": 1}}
	if lines, _ := n.Lines(); !strings.HasSuffix(lines[4], " "+style.Off(Grouping+"₁")) ||
		!strings.HasSuffix(lines[5], " "+style.Tint(Grouping+"₁")) {
		t.Errorf("numbered by level: %q %q", lines[4], lines[5])
	}
	q.Focused = true
	if q, _ = q.Update(tea.KeyPressMsg{Code: 'g', Text: "g"}); !q.Kept["source"] || len(q.Group) > 0 || len(q.Auto) > 0 {
		t.Errorf("g on source: kept %v, group %q, auto %q", q.Kept, q.Group, q.Auto)
	}
	// a column shown again is kept a column
	q = Panel{Columns: columns, Hidden: map[string]bool{"who": true}}
	if q, _ = q.Click(len(columns) + 2); q.Hidden["who"] || !q.Kept["who"] {
		t.Errorf("who shown: hidden %v, kept %v", q.Hidden, q.Kept)
	}
}

// Heading names the column whose group a row heads, at its depth, and nothing for a row that heads none.
func TestAHeadingNamesTheColumnItGroups(t *testing.T) {
	p := Panel{Group: []string{"source", "who"}, Groupable: map[string]bool{"source": true, "who": true}}
	rows, _ := p.group([]Row{{Key: "#1", Cells: []Cell{{Values: []string{"github"}}, {Values: []string{"ada"}}}}},
		[]string{"source", "who"})
	var got []string
	for _, r := range rows {
		got = append(got, p.Heading(r))
	}
	if !slices.Equal(got, []string{"source", "who", ""}) {
		t.Errorf("the headings: %q", got)
	}
	if h := (Panel{Group: []string{"source"}}).Heading(rows[1]); h != "" {
		t.Errorf("a heading deeper than the groups: %q", h)
	}
}

// The views' dropdown stands under the search, a gap either side of it, before the columns: the view's name and ▾,
// a caret while it is renamed under the keyboard, inverse there, and Save at the line's right while the panel differs.
func TestTheViewsDropdownStandsUnderTheSearch(t *testing.T) {
	p := Panel{Columns: []string{"id", "title"}, Views: true, View: "Default", Fill: 24}
	lines, at := p.Lines()
	plain := make([]string, len(lines))
	for k, l := range lines {
		plain[k] = strings.TrimRight(ansi.Strip(l), " ")
	}
	if !slices.Equal(plain[:5], []string{"🔎 search", "", ViewGlyph + " Default ▾", "", "columns"}) ||
		!slices.Equal(at[:5], []int{0, -1, 1, -1, -1}) || !p.ViewAt(2) || p.ViewAt(4) || p.OnView() {
		t.Errorf("the dropdown: %q %v", plain[:5], at[:5])
	}
	p.Focused, p.Cursor, p.Naming, p.Changed, p.View = true, 1, true, true, "Mine"
	lines, _ = p.Lines()
	if !p.OnView() || !strings.HasPrefix(lines[2], style.Cursor(ViewGlyph + " Mine▏ ▾")[:8]) ||
		!strings.HasSuffix(lines[2], Save) || ansi.StringWidth(lines[2]) != 24 {
		t.Errorf("renamed under the keyboard, changed: %q", lines[2])
	}
	// the dropdown's keys are its screen's, but up and down
	if _, ok := p.Update(tea.KeyPressMsg{Code: tea.KeyEnter}); ok {
		t.Error("the panel took enter on the dropdown")
	}
	if q, ok := p.Update(tea.KeyPressMsg{Code: tea.KeyDown}); !ok || q.Cursor != 2 {
		t.Errorf("down from the dropdown: %d", q.Cursor)
	}
	if q, ok := p.Click(2); !ok || q.Cursor != 1 || q.Show != p.Show {
		t.Errorf("a click on the dropdown: %d", q.Cursor)
	}
}
