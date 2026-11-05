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
	got := plain(Draw(issues(), 40, -1, func(Row) bool { return false }))
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
	got := plain(Draw(issues(), 40, -1, func(Row) bool { return true }))
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
	lines := Draw(issues(), 40, -1, func(Row) bool { return true })
	if rows := []int{lines[0].Row, lines[2].Row, lines[3].Row, lines[5].Row}; !slices.Equal(rows, []int{0, 0, 1, 2}) {
		t.Errorf("the lines' rows: %v", rows)
	}
}

func TestOpensSaysWhichRowsHideSomethingAtTheWidth(t *testing.T) {
	rows := issues()
	if !Opens(rows, 0, 40) || Opens(rows, 1, 40) || !Opens(rows, 2, 40) {
		t.Errorf("at 40: %v %v %v", Opens(rows, 0, 40), Opens(rows, 1, 40), Opens(rows, 2, 40))
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
		if got := LinkAt(rows, 0, 40, x); got != want {
			t.Errorf("at %d: %q, want %q", x, got, want)
		}
	}
	if got := LinkAt(rows, 3, 40, len(lead)); got != "https://example/span" {
		t.Errorf("a span's glyph: %q", got)
	}
}

// The band marks the cursor's row, so its first cell is bold in the accent and not underlined.
func TestTheCursorsRowIsOnABandWithItsFirstCellChosen(t *testing.T) {
	lines := Draw(issues(), 40, 1, func(Row) bool { return false })
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
	if got := Draw(rows, 20, 0, func(Row) bool { return false })[0].Text; !strings.Contains(got, chosen+"a") ||
		strings.Contains(got, dim) {
		t.Errorf("a dim first cell under the cursor: %q", got)
	}
}

func TestARowOfOneCellSpansTheWidthAndOpensWhenItsTextIsCut(t *testing.T) {
	rows := []Row{{Key: "h", Cells: []Cell{{Values: []string{"a heading far too long for the pane"}, Wrap: true}}}}
	closed := plain(Draw(rows, 20, -1, func(Row) bool { return false }))
	open := plain(Draw(rows, 20, -1, func(Row) bool { return true }))
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
	got := plain(Draw(p.Filter(issues(), columns), 40, -1, func(Row) bool { return false }))
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

// The header stands each name in its column, as Draw lays the rows out: over a number right-aligned, over the text
// that wraps at its start, wrapped in its column when wider, and left out over a column no row has text in; rows of
// one cell have none.
func TestTheHeaderStandsEachNameOverItsColumn(t *testing.T) {
	rows := []Row{
		{Key: "a", Cells: []Cell{{Values: []string{"#1"}, Right: true}, one(""), one("🔀 #2"), {Values: []string{"a title"}, Wrap: true}}},
		{Key: "b", Cells: []Cell{{Values: []string{"#31"}, Right: true}, one(""), one("🔀 #12"), {Values: []string{"another"}, Wrap: true}}},
	}
	header := Header(rows, []string{"id", "fork", "changes", "title"}, 40)
	plain := make([]string, len(header))
	for k, l := range header {
		plain[k] = ansi.Strip(l)
	}
	drawn := ansi.Strip(Draw(rows, 40, -1, func(Row) bool { return false })[1].Text)
	// "changes" is wider than its column, "🔀 #12", and breaks in it; "fork" is over no text, and left out
	if len(header) != 2 || plain[0] != "    id  change  title" || plain[1] != "        s" || strings.Contains(plain[0], "fork") {
		t.Fatalf("the header:\n%s", strings.Join(plain, "\n"))
	}
	for _, name := range []string{"id", "change", "title"} {
		at := strings.Index(plain[0], name)
		under := map[string]string{"id": "#31", "change": "🔀", "title": "another"}[name]
		if name == "id" {
			at += len(name) - len(under)
		}
		if ansi.StringWidth(plain[0][:at]) != ansi.StringWidth(drawn[:strings.Index(drawn, under)]) {
			t.Errorf("%s does not stand over its column:\n%s\n%s", name, plain[0], drawn)
		}
	}
	if !strings.Contains(header[0], style.Label("id")) {
		t.Errorf("the names are not bold: %q", header[0])
	}
	if got := Header([]Row{{Cells: []Cell{one("a heading")}}}, []string{"x"}, 40); got != nil {
		t.Errorf("rows of one cell have a header: %q", got)
	}
}
