package grid

import (
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
	got := plain(Draw(issues(), nil, 40, -1, func(Row) bool { return false }))
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
	got := plain(Draw(issues(), nil, 40, -1, func(Row) bool { return true }))
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
	lines := Draw(issues(), nil, 40, -1, func(Row) bool { return true })
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
	lines := Draw(issues(), nil, 40, 1, func(Row) bool { return false })
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
	if got := Draw(rows, nil, 20, 0, func(Row) bool { return false })[0].Text; !strings.Contains(got, chosen+"a") ||
		strings.Contains(got, dim) {
		t.Errorf("a dim first cell under the cursor: %q", got)
	}
}

func TestARowOfOneCellSpansTheWidthAndOpensWhenItsTextIsCut(t *testing.T) {
	rows := []Row{{Key: "h", Cells: []Cell{{Values: []string{"a heading far too long for the pane"}, Wrap: true}}}}
	closed := plain(Draw(rows, nil, 20, -1, func(Row) bool { return false }))
	open := plain(Draw(rows, nil, 20, -1, func(Row) bool { return true }))
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
	if p.Width() != ansi.StringWidth("🔎 search") {
		t.Errorf("width %d", p.Width())
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
	if q, ok := p.Click(7); !ok || q.Show != All || q.Cursor != 3 {
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
	got := plain(Draw(p.Filter(issues(), columns), nil, 40, -1, func(Row) bool { return false }))
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
	drawn := ansi.Strip(Draw(rows, names, 40, -1, func(Row) bool { return false })[1].Text)
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
	got := ansi.Strip(Draw(rows, nil, 80, -1, func(Row) bool { return false })[0].Text)
	// the title column is as wide as the widest title, shorter one, even under minFill; the state stands after it
	if got != "   #1  short        ready" {
		t.Errorf("a short title: %q", got)
	}
	rows[1].Cells[1].Values[0] = "a title of thirty cells, wide."
	if got := ansi.Strip(Draw(rows, nil, 80, -1, func(Row) bool { return false })[0].Text); got != "   #1  short"+strings.Repeat(" ", 27)+"ready" {
		t.Errorf("a title as wide as the longest: %q", got)
	}
}
