package grid

import (
	"fmt"
	"maps"
	"slices"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/empowerite/kit/projects/tui/internal/shell"
	"github.com/empowerite/kit/projects/tui/internal/style"
)

// first gives the first of two values, as the rows of group.
func first(rows []Row, _ []string) []Row { return rows }

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
	if !slices.Equal(closed, []string{" ▸ a heading far to…"}) || len(open) != 2 || open[1] != "   long for the pane" {
		t.Errorf("closed %q, open %q", closed, open)
	}
}

// plainLines gives the panel's lines without their escapes, and what each draws.
func plainLines(p Panel) ([]string, []int) {
	lines, at := p.Lines()
	for i, l := range lines {
		lines[i] = strings.TrimRight(ansi.Strip(l), " ")
	}
	return lines, at
}

// Under the search, the field editor lists each field with a label, by it, once, in its section: the groups by depth,
// the columns by position, the hidden in the stage's order, each section under its dim heading, an empty one's too. A
// field's line marks its sort, its group, and a filter on it; whether the panel folds is a box of its own after them.
func TestThePanelDrawsItsControlsOneALine(t *testing.T) {
	p := Panel{Columns: []string{"number", "who", "", "state", "title"}, Labels: map[string]string{"number": "#"},
		Groupable: map[string]bool{"state": true, "who": true}, Sortable: map[string]bool{"number": true},
		Fields: map[string]Field{"who": {Placement: Hidden, Filter: Filter{Values: []string{Caller}}},
			"title": {Position: 1}, "number": {Sort: Descending, Priority: 1}}, Owner: "who"}
	got, at := plainLines(p)
	want := []string{"🔎 search", "", "group by", "columns", "☑ title", "☑ # ▼", "☑ state", "hidden", "☐ who ⊘", "",
		"☐ " + Folds}
	if !slices.Equal(got, want) || !slices.Equal(at, []int{0, -1, -1, -1, 1, 2, 3, -1, 4, -1, 5}) {
		t.Errorf("the panel:\n%s\n%v", strings.Join(got, "\n"), at)
	}
	// a column and a group are ticked, bold in the accent; a hidden field's box is dim; the marks are in the accent
	lines, _ := p.Lines()
	if lines[4] != style.Mark("☑ title") || lines[8] != style.Off("☐")+" who "+style.Tint(Filtered) ||
		lines[2] != style.Off("group by") {
		t.Errorf("the controls are not marked: %q", lines[2:9])
	}
	q := p.Grouped("state", false)
	if got, _ := plainLines(q); !slices.Equal(got[2:5], []string{"group by", "☑ state ▤", "columns"}) {
		t.Errorf("state grouped: %q", got)
	}
	// with the keyboard, dim hints under the editor say how a field moves, and what enter on it opens, then what the
	// clicks on a column's name do; the width keeps room for them
	p.Focused = true
	hints := append([]string{"⇧↑↓  move, or drag", "⏎    sort, filter, draw", ""}, Clicks...)
	if got, _ := plainLines(p); !slices.Equal(got[len(got)-len(hints):], hints) ||
		p.Width() != ansi.StringWidth("⏎    sort, filter, draw") {
		t.Errorf("the hints: %q, width %d", got[len(got)-len(hints):], p.Width())
	}
	// a stage with no field that has a label has the search alone
	if got, _ := plainLines(Panel{Columns: []string{""}}); len(got) != 1 {
		t.Errorf("no field: %q", got)
	}
}

func TestThePanelTakesKeysAndClicks(t *testing.T) {
	p := Panel{Columns: []string{"number", "who"}, Groupable: map[string]bool{"who": true}}.Find()
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
	given := map[string]Field{}
	p.Fields = given
	p, _ = p.Update(key(tea.KeyDown, ""))
	p, _ = p.Update(key(tea.KeyDown, ""))
	if p.Under() != "who" {
		t.Fatalf("down twice: %q", p.Under())
	}
	if p, _ = p.Update(key(tea.KeySpace, " ")); p.Field("who").Placement != Hidden || len(given) > 0 || p.Under() != "who" {
		t.Errorf("space did not hide who, or changed the given map: %v %v", p.Fields, given)
	}
	if p, _ = p.Update(key(tea.KeySpace, " ")); p.Field("who").Placement != Column {
		t.Errorf("space did not show who: %v", p.Fields)
	}
	// enter on a field is not the panel's; down stops at whether it folds, and space or enter toggles it
	if _, ok := p.Update(key(tea.KeyEnter, "")); ok {
		t.Error("enter on a field was taken")
	}
	for range 10 {
		p, _ = p.Update(key(tea.KeyDown, ""))
	}
	if p, _ = p.Update(key(tea.KeyEnter, "")); !p.Fold || p.Under() != "" {
		t.Errorf("enter on the last control: fold %v", p.Fold)
	}
	if q, ok := p.Click(1); ok || q.Cursor != p.Cursor {
		t.Errorf("a click on a gap was taken")
	}
	// a click on a field acts as space does: number's line, under the headings of the groups and the columns
	if q, ok := p.Click(4); !ok || q.Field("number").Placement != Hidden || q.Under() != "number" {
		t.Errorf("a click on number: %+v, under %q", q.Field("number"), q.Under())
	}
	if _, ok := p.Update(key('x', "x")); ok {
		t.Error("a letter off the search was taken")
	}
	// s and g are no longer the panel's
	p.Cursor = 1
	if _, ok := p.Update(key('g', "g")); ok {
		t.Error("g was taken")
	}
	// / off the search goes to it; on it, / is typed
	if p, _ = p.Update(key('/', "/")); p.Cursor != 0 {
		t.Errorf("/ off the search: cursor %d", p.Cursor)
	}
	if p, _ = p.Update(key('/', "/")); !strings.HasSuffix(p.Query, "/") {
		t.Errorf("/ on the search: %q", p.Query)
	}
}

// Shift with up or down moves the field under the keyboard a place, the keyboard with it: within its section, a column
// changes position and a group depth; across a heading, its placement, a column up into the groups' last depth where
// it can group, and down into the hidden, a group down into the first column, a hidden field up into the last column.
// The keys bar names the move only where the field can move.
func TestShiftWithAnArrowMovesAFieldAcrossTheSections(t *testing.T) {
	columns := []string{"id", "state", "who", "title"}
	p := Panel{Columns: columns, Groupable: map[string]bool{"state": true, "who": true}, Focused: true}
	shift := func(p Panel, code rune) Panel {
		p, _ = p.Update(tea.KeyPressMsg{Code: code, Mod: tea.ModShift})
		return p
	}
	sections := func(p Panel) string {
		return fmt.Sprintf("%v %v %v", p.section(Group), p.section(Column), p.section(Hidden))
	}
	p.Cursor = 2 // state
	if p = shift(p, tea.KeyUp); sections(p) != "[] [state id who title] []" || p.Under() != "state" {
		t.Fatalf("shift-up on state: %s, under %q", sections(p), p.Under())
	}
	if p = shift(p, tea.KeyUp); sections(p) != "[state] [id who title] []" || p.Under() != "state" {
		t.Fatalf("shift-up on the first column: %s", sections(p))
	}
	if q := shift(p, tea.KeyUp); sections(q) != sections(p) {
		t.Errorf("shift-up on the first group: %s", sections(q))
	}
	p.Cursor = 3 // who, the second column, up into the groups' last depth
	p = shift(shift(p, tea.KeyUp), tea.KeyUp)
	if sections(p) != "[state who] [id title] []" || p.Field("who").Depth != 2 {
		t.Fatalf("who up twice: %s", sections(p))
	}
	if p = shift(p, tea.KeyUp); sections(p) != "[who state] [id title] []" || p.Field("state").Depth != 2 {
		t.Fatalf("who up a depth: %s", sections(p))
	}
	p.Cursor = 2 // state, the last group, down into the first column
	if p = shift(p, tea.KeyDown); sections(p) != "[who] [state id title] []" || p.Field("state").Placement != Column {
		t.Fatalf("state down: %s", sections(p))
	}
	// id cannot group: up from the first column it stays
	p.Cursor = 3
	if p = shift(p, tea.KeyUp); sections(p) != "[who] [id state title] []" {
		t.Fatalf("id up: %s", sections(p))
	}
	if q := shift(p, tea.KeyUp); sections(q) != sections(p) || slices.ContainsFunc(q.Keys(false).Bindings,
		func(b shell.Binding) bool { return slices.Contains(b.Presses, "shift+up") }) {
		t.Errorf("id, which cannot group, up from the first column: %s, keys %+v", sections(q), q.Keys(false))
	}
	// the last column down is hidden, and keeps its position; a hidden field up is the last column
	p.Cursor = 4
	if p = shift(p, tea.KeyDown); sections(p) != "[who] [id state] [title]" || p.Under() != "title" {
		t.Fatalf("title down: %s", sections(p))
	}
	if keys := p.Keys(false); keys.Bindings[0].Does != "show" || !slices.Equal(keys.Bindings[1].Presses, []string{"shift+up"}) {
		t.Errorf("the keys of a hidden field: %+v", keys)
	}
	if p = shift(p, tea.KeyUp); sections(p) != "[who] [id state title] []" {
		t.Fatalf("title up: %s", sections(p))
	}
	// a field no row fills, which the editor does not list, never moves
	if _, ok := p.Moved("nonesuch", Column, 0); ok {
		t.Error("a field not listed moved")
	}
}

// A field's line dragged onto another's moves to where it stands, as shift with an arrow does, across the sections too;
// onto a section's heading, to the top of that section, an empty one's too.
func TestADraggedFieldMovesWhereItIsDropped(t *testing.T) {
	columns := []string{"id", "state", "who"}
	p := Panel{Columns: columns, Groupable: map[string]bool{"state": true, "who": true}}
	line := func(p Panel, text string) int {
		got, _ := plainLines(p)
		return slices.IndexFunc(got, func(l string) bool { return strings.HasSuffix(l, text) })
	}
	q, ok := p.Dropped("who", line(p, "group by"))
	if !ok || !slices.Equal(q.Groups(), []string{"who"}) || q.Under() != "who" {
		t.Fatalf("who onto group by: %q", q.Groups())
	}
	if q, _ = q.Dropped("state", line(q, "☑ who ▤")); !slices.Equal(q.Groups(), []string{"state", "who"}) {
		t.Errorf("state onto who: %q", q.Groups())
	}
	if q, _ = q.Dropped("id", line(q, "hidden")); q.Field("id").Placement != Hidden {
		t.Errorf("id onto hidden: %+v", q.Field("id"))
	}
	if q, _ = q.Dropped("who", line(q, "☐ id")); q.Field("who").Placement != Hidden || len(q.Groups()) != 1 {
		t.Errorf("who onto id, hidden: %+v %q", q.Field("who"), q.Groups())
	}
	if _, ok := q.Dropped("state", line(q, "☑ state ▤")); ok {
		t.Error("state onto itself moved")
	}
	if _, ok := q.Dropped("state", 0); ok {
		t.Error("state onto the search moved")
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
	if p.FieldAt(0) != "" || p.FieldAt(99) != "" {
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
	if got := byAge.Sorted("id", false).SortKeys(); len(got) != 1 || got[0] != (SortKey{"id", true}) ||
		byAge.Sorted("id", false).Mark("id") != "▼" {
		t.Errorf("a click on id among two keys: %+v", got)
	}
	if got := p.Sorted("id", false).Sorted("id", false).Sorted("id", false); len(got.SortKeys()) != 0 || got.Mark("id") != "" {
		t.Errorf("three clicks: %+v", got.SortKeys())
	}
	if got := byAge.Sorted("id", true).Sorted("id", true).SortKeys(); len(got) != 1 || got[0].Column != "age" {
		t.Errorf("two shift-clicks on id: %+v", got)
	}
	// each key is the field's sort, at its priority
	if f := byAge.Field("id"); f.Sort != Ascending || f.Priority != 2 || byAge.Field("age").Priority != 1 {
		t.Errorf("the fields' sorts: %+v", byAge.Fields)
	}
	if got := p.Sorted("title", false); got.SortKeys() != nil {
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
	grouped, _ := p.group(rows, columns)
	for _, r := range grouped {
		text := r.Key
		if r.Aside {
			text = ansi.Strip(r.Cells[0].first())
		}
		got = append(got, text)
	}
	want := []string{"▾ draft state  1", "  ▾ bob who  1", "#1", "▾ ready state  2", "  ▾ ada who  1", "#3",
		"  ▾ none who  1", "#2", "gone"}
	if !slices.Equal(got, want) {
		t.Errorf("grouped:\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
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
	if got, _ := desc.group(rows, columns); ansi.Strip(got[0].Cells[0].first()) != "▾ ready state  2" {
		t.Errorf("the groups descending: %q", ansi.Strip(got[0].Cells[0].first()))
	}
	if got, _ := (Panel{}).group(rows, columns); len(got) != len(rows) {
		t.Error("grouping by nothing changed the rows")
	}
	q := Panel{Columns: columns, Groupable: p.Groupable, Focused: true, Cursor: 2}
	q, _ = q.Update(tea.KeyPressMsg{Code: tea.KeyUp, Mod: tea.ModShift})
	q, _ = q.Update(tea.KeyPressMsg{Code: tea.KeyUp, Mod: tea.ModShift})
	if lines, _ := q.Lines(); !slices.Equal(q.Groups(), []string{"state"}) || !strings.HasSuffix(ansi.Strip(lines[3]), "state "+Grouping) {
		t.Errorf("state moved up into the groups: %q\n%s", q.Groups(), strings.Join(lines, "\n"))
	}
}

// While the panel folds, a column that can group, whose value is the same on every row, groups them on its own: without
// a group the person chose, one heading over every row says the values, joined in the panel's order, and their cells
// are blank. Under a group the person chose, the walk goes on in each group, the values joining its heading. A column
// blank on some rows, one with several values, a group of one row, and a hidden column do not group on their own, nor
// does any while the panel does not fold; the panel's box for folding stops it, and the panel marks a column grouping on
// its own with a dim ▤.
// The rows group by a cell's sort key, its raw value, and the heading says the value as the rows draw it, or as the
// cell's Head says it where the rows draw more than they group by.
func TestAGroupIsTheRawValueAndItsHeadingTheDrawnOne(t *testing.T) {
	columns := []string{"id", "state", "checks"}
	row := func(id, state, checks string) Row {
		return Row{Key: id, Cells: []Cell{one(id), {Values: []string{"🏁 " + state}, Raw: []string{state}},
			{Values: []string{"✅ " + checks}, Raw: []string{"SUCCESS"}, Head: "✅ SUCCESS"}}}
	}
	rows := []Row{row("#1", "ready", "1 / 1"), row("#2", "draft", "2 / 2"), row("#3", "ready", "3 / 3")}
	p := Panel{Groupable: map[string]bool{"state": true, "checks": true}, Fold: true,
		Fields: map[string]Field{"state": {Placement: Group, Depth: 1}}}
	got, _ := p.group(rows, columns)
	var headings []string
	for _, r := range got {
		if r.Aside {
			headings = append(headings, ansi.Strip(r.Cells[0].first()))
		}
	}
	if want := []string{"▾ 🏁 draft state  1", "▾ 🏁 ready state, ✅ SUCCESS checks  2"}; !slices.Equal(headings, want) {
		t.Errorf("the headings: %q", headings)
	}
}

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
	p := Panel{Columns: columns, Groupable: groupable, Fold: true}
	got, auto := p.group(rows, columns)
	want := []string{"▾ github source, here where  3", "#1||draft||ada", "#2||draft||", "#3||ready||bob"}
	if !slices.Equal(texts(got), want) || !slices.Equal(auto, []string{"source", "where"}) || Levels(got) != 1 {
		t.Errorf("over every row: %q, auto %q, levels %d", texts(got), auto, Levels(got))
	}
	// the panel's order joins the values; a hidden column does not group, nor does any while the panel does not fold
	if moved, _ := p.Moved("where", Column, 0); texts(first(moved.group(rows, columns)))[0] != "▾ here where, github source  3" {
		t.Errorf("in the panel's order: %q", texts(first(moved.group(rows, columns)))[0])
	}
	if got, auto := p.Placed("source", Hidden).group(rows, columns); texts(got)[0] != "▾ here where  3" ||
		!slices.Equal(auto, []string{"where"}) {
		t.Errorf("source hidden: %q, auto %q", texts(got), auto)
	}
	if got, auto := (Panel{Groupable: groupable}).group(rows, columns); len(got) != 3 || auto != nil {
		t.Errorf("not folding: %q, auto %q", texts(got), auto)
	}
	// under the groups the person chose, each group alone; one of one row does not group
	p = p.Grouped("state", false)
	got, auto = p.group(rows, columns)
	// the column the person groups by stays in the cells, which the panel's hide then leaves out
	want = []string{"▾ draft state, github source, here where  2", "#1||draft||ada", "#2||draft||", "▾ ready state  1",
		"#3|github|ready|here|bob"}
	if !slices.Equal(texts(got), want) || !slices.Equal(auto, []string{"source", "where"}) || Levels(got) != 1 {
		t.Errorf("under state: %q, auto %q, levels %d", texts(got), auto, Levels(got))
	}
	// two values in a cell, or one row alone, do not group
	rows[0].Cells[3].Values = []string{"here", "there"}
	if got, _ := (Panel{Groupable: groupable, Fold: true}).group(rows, columns); texts(got)[0] != "▾ github source  3" {
		t.Errorf("several values: %q", texts(got)[0])
	}
	if got, auto := (Panel{Groupable: groupable, Fold: true}).group(rows[:1], columns); len(got) != 1 || auto != nil {
		t.Errorf("one row: %q", texts(got))
	}
	// the panel marks a column grouping on its own dim, and its box for folding stops the folding
	q := Panel{Columns: columns, Groupable: groupable, Fold: true, Auto: []string{"source"}}
	if lines, _ := q.Lines(); !strings.HasSuffix(lines[5], " "+style.Off(Grouping)) {
		t.Errorf("source grouping on its own: %q", lines[5])
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
	n := Panel{Columns: columns, Groupable: groupable, Fields: map[string]Field{"state": {Placement: Group, Depth: 1}},
		Auto: []string{"source"}, AutoAt: map[string]int{"source": 1}}
	if lines, _ := n.Lines(); !strings.HasSuffix(lines[6], " "+style.Off(Grouping+"₁")) ||
		!strings.HasSuffix(lines[3], " "+style.Tint(Grouping+"₁")) {
		t.Errorf("numbered by level: %q %q", lines[6], lines[3])
	}
	q.Focused, q.Cursor = true, len(q.controls())-1
	if q.Keys(false).Bindings[0].Does != "unfold" {
		t.Errorf("the keys on the box for folding: %+v", q.Keys(false))
	}
	if q, _ = q.Update(tea.KeyPressMsg{Code: tea.KeySpace, Text: " "}); q.Fold || len(q.Groups()) > 0 || len(q.Auto) > 0 {
		t.Errorf("space on the box for folding: fold %v, group %q, auto %q", q.Fold, q.Groups(), q.Auto)
	}
	// a column shown again is a column
	q = Panel{Columns: columns, Fields: map[string]Field{"who": {Placement: Hidden}}}
	if q, _ = q.Click(len(columns) + 4); q.Field("who").Placement != Column {
		t.Errorf("who shown: %+v", q.Field("who"))
	}
}

// Heading names the column whose group a row heads, at its depth, and nothing for a row that heads none.
func TestAHeadingNamesTheColumnItGroups(t *testing.T) {
	p := Panel{Groupable: map[string]bool{"source": true, "who": true}}.Grouped("source", false).Grouped("who", true)
	rows, _ := p.group([]Row{{Key: "#1", Cells: []Cell{{Values: []string{"github"}}, {Values: []string{"ada"}}}}},
		[]string{"source", "who"})
	var got []string
	for _, r := range rows {
		got = append(got, p.Heading(r))
	}
	if !slices.Equal(got, []string{"source", "who", ""}) {
		t.Errorf("the headings: %q", got)
	}
	if h := p.Grouped("who", true).Heading(rows[1]); h != "" {
		t.Errorf("a heading deeper than the groups: %q", h)
	}
}

// Under the search, after a line of gap, the field editor starts: the panel has no control between them.
func TestTheColumnsStandUnderTheSearch(t *testing.T) {
	p := Panel{Columns: []string{"id", "title"}, Fill: 24}
	lines, at := p.Lines()
	plain := make([]string, len(lines))
	for k, l := range lines {
		plain[k] = strings.TrimRight(ansi.Strip(l), " ")
	}
	if !slices.Equal(plain[:5], []string{"🔎 search", "", "group by", "columns", Ticked + " id"}) ||
		!slices.Equal(at[:5], []int{0, -1, -1, -1, 1}) {
		t.Errorf("the panel's first lines: %q %v", plain[:5], at[:5])
	}
	if q, ok := p.Update(tea.KeyPressMsg{Code: tea.KeyDown}); !ok || q.FieldAt(4) != "id" || q.Cursor != 1 {
		t.Errorf("down from the search: %d", q.Cursor)
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
		grouped, _ := p.group(rows, columns)
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
	ready := "group\x00ready"
	want := []string{"▾ draft state  1", "  ▾ bob who  1", "#1", "▸ ready state  2", "gone"}
	if got := texts(p.Close(ready, true)); !slices.Equal(got, want) {
		t.Errorf("ready folded:\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
	// a nested heading folds only its own rows
	want = []string{"▾ draft state  1", "  ▾ bob who  1", "#1", "▾ ready state  2", "  ▸ ada who  1",
		"  ▾ none who  1", "#2", "gone"}
	if got := texts(p.Close(ready+"\x00ada", true)); !slices.Equal(got, want) {
		t.Errorf("ada folded:\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
	// a closed heading holds the rows it folds away, which still lay out the header and the columns
	every := Panel{Groupable: p.Groupable}.Grouped("state", false)
	grouped, _ := every.group(rows, columns)
	names := []string{"id", "state", "who"}
	open, _ := Header(grouped, names, 60, false)
	folded := every.Close("group\x00draft", true).Close(ready, true).FoldAway(grouped)
	if shut, _ := Header(folded, names, 60, false); len(folded) != 3 || len(folded[1].Folded) != 2 ||
		!slices.Equal(shut, open) || !slices.Equal(headed(folded, names), headed(grouped, names)) {
		t.Errorf("every group closed: %d rows, %d folded under ready, header %q, was %q", len(folded),
			len(folded[1].Folded), shut, open)
	}
	if HeaderAt(folded, names, 60, 3) != HeaderAt(grouped, names, 60, 3) {
		t.Error("a click on the header of closed groups finds another column")
	}
	// unfolded again, every row is back, and the panel's map is never the one it was given
	closed := p.Close(ready, true)
	if q := closed.Close(ready, false); len(texts(q)) != 9 || !closed.Closed[ready] {
		t.Errorf("unfolded: %q, the closed panel's map %v", texts(q), closed.Closed)
	}
}

// A field's popover sets its sort, its filter, and its rendering apart: SortedAs keeps a key's place among the keys in
// a new direction, adds one after the others, and takes one away; a field that cannot sort changes nothing. The
// panel's map is never the one it was given.
func TestAFieldsPopoverSetsItsSortFilterAndRendering(t *testing.T) {
	p := Panel{Sortable: map[string]bool{"id": true, "age": true}}.Sorted("age", false).Sorted("id", true)
	given := p.Fields
	if q := p.SortedAs("age", Descending); q.Mark("age") != "▼₁" || q.Mark("id") != "▲₂" || given["age"].Sort != Ascending {
		t.Errorf("age descending: %q %q", q.Mark("age"), q.Mark("id"))
	}
	if q := p.SortedAs("age", Unsorted); len(q.SortKeys()) != 1 || q.Mark("id") != "▲" {
		t.Errorf("age off: %+v", q.SortKeys())
	}
	if q := p.SortedAs("age", Unsorted).SortedAs("age", Ascending); q.Mark("age") != "▲₂" {
		t.Errorf("age back on: %q", q.Mark("age"))
	}
	if q := p.SortedAs("title", Ascending); len(q.SortKeys()) != 2 {
		t.Errorf("title, which cannot sort: %+v", q.SortKeys())
	}
	if q := p.SortedBy(nil); len(q.SortKeys()) != 0 {
		t.Errorf("sorted by nothing: %+v", q.SortKeys())
	}
	values := []string{Caller}
	q := p.FilteredBy("who", Filter{Values: values, Exclude: true}).Rendered("who", "text")
	values[0] = "bob"
	if f := q.Field("who"); !slices.Equal(f.Filter.Values, []string{Caller}) || !f.Filter.Exclude ||
		f.Rendering != "text" || !q.Filtering() || p.Filtering() {
		t.Errorf("who's filter and rendering: %+v", f)
	}
}
