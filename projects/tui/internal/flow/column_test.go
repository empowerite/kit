package flow

import (
	"slices"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/empowerite/kit/projects/tui/internal/grid"
	"github.com/empowerite/kit/projects/tui/internal/table"
	"github.com/empowerite/kit/projects/tui/internal/testkit"
)

// cardLines gives the open card's lines as they read.
func cardLines(m Model) []string {
	var out []string
	for _, l := range m.pop.Lines(&m) {
		out = append(out, ansi.Strip(l.Text))
	}
	return out
}

// cardLine gives the index of the open card's line that reads as text, its glyph apart, -1 for none.
func cardLine(m Model, text string) int {
	return slices.IndexFunc(cardLines(m), func(l string) bool {
		_, rest, _ := strings.Cut(l, " ")
		return l == text || rest == text
	})
}

// cardAct acts on the open card's line that reads as text, as a click on it does.
func cardAct(t *testing.T, m *Model, text string) {
	t.Helper()
	k := cardLine(*m, text)
	if k < 0 {
		t.Fatalf("no line %q in %q", text, cardLines(*m))
	}
	m.pop.Act(m, k)
}

// A right click on a column's name opens its card under it, as a terminal that keeps ctrl and option with a click for
// itself still sends the right button, titled by its label: its sort, a radio; for a list, flattened here so it sorts,
// its rows gathered again, a radio; the rows grouped by it, innermost, or no longer, a box; its filter and rendering;
// and whether it shows. What it sets applies at once, and the card stays open; a section the column cannot use is not
// there.
func TestARightClickOnAColumnsNameOpensItsCard(t *testing.T) {
	m := loaded(t, viewing(""), tea.WindowSizeMsg{Width: 140, Height: 40})
	m.setPanel(m.panel(m.stage).Flattened("assignees", true))
	at := func(name string) int { return strings.Index(lines(m)[m.headerTop()], name) }
	rightClick := func(name string) tea.MouseClickMsg {
		return tea.MouseClickMsg{X: at(name), Y: m.headerTop(), Button: tea.MouseRight}
	}
	group := func() []string { return m.panel(m.stage).Groups() }
	sorts := func() int { return len(m.panel(m.stage).SortKeys()) }
	if m = drive(m, rightClick("assignee")); !m.pop.Open || m.pop.Field != "assignees" || !m.pop.Header || m.pop.At != 1 {
		t.Fatalf("a right click on assignee: %+v\n%s", m.pop, raw(m))
	}
	got := cardLines(m)
	if !slices.Equal(got[:10], []string{"sort", grid.Unpicked + " ¹ Ascending", grid.Unpicked + " ₁ Descending",
		grid.Picked + " " + table.Unsorted, "", "rows", grid.Unticked + " " + table.GroupIt,
		grid.Picked + " " + table.FlattenIt("assignee"), grid.Unpicked + " " + table.GatherIt("assignees"), ""}) ||
		got[len(got)-1] != grid.Ticked+" "+table.ShowIt || !strings.Contains(keysOf(m), "esc close") {
		t.Fatalf("the card: %q, %q", got, keysOf(m))
	}
	// enter on ¹ Ascending sorts by it, and the card stays open
	if m = drive(m, press(tea.KeyEnter)); !m.pop.Open || sorts() != 1 || m.panel(m.stage).Mark("assignees") != "¹" {
		t.Fatalf("Ascending: %+v", m.panel(m.stage).SortKeys())
	}
	// another column's Ascending, chosen by a click, comes after it, its triangles ⁿ
	m = drive(m, esc)
	m = drive(m, rightClick("age"))
	if cardLine(m, "ⁿ Ascending") != 1 {
		t.Errorf("age, with assignees sorting: %q", cardLines(m))
	}
	box, top, left := m.pop.Box(&m)
	if m = drive(m, testkit.Click(left+3, top+1+cardLine(m, "ⁿ Ascending"))); !m.pop.Open || sorts() != 2 ||
		m.panel(m.stage).SortKeys()[1].Column != "age" {
		t.Fatalf("Ascending on age: %+v\n%v", m.panel(m.stage).SortKeys(), box)
	}
	// Group groups within the groups, innermost, and again takes it out wherever it stands
	m.setPanel(regroup(m.panel(m.stage), "source"))
	m = drive(m, esc)
	m = drive(m, rightClick("assignee"))
	cardAct(t, &m, table.GroupIt)
	if !slices.Equal(group(), []string{"source", "assignees"}) {
		t.Fatalf("Group assignees: %q", group())
	}
	m.pop = table.OpenCard(&m, "source", true)
	cardAct(t, &m, table.GroupIt+" "+grid.GroupMark(m.panel(m.stage).Field("source").Depth))
	if !slices.Equal(group(), []string{"assignees"}) {
		t.Fatalf("Group source again: %q", group())
	}
	// title sorts and does not group: its card has no rows; Unsorted takes title out of the keys
	m = drive(m, esc)
	m = drive(m, rightClick("title"))
	if cardLine(m, "rows") >= 0 || cardLine(m, table.GroupIt) >= 0 {
		t.Errorf("title's card: %q", cardLines(m))
	}
	cardAct(t, &m, "ⁿ Ascending")
	cardAct(t, &m, table.Unsorted)
	if slices.ContainsFunc(m.panel(m.stage).SortKeys(), func(k grid.SortKey) bool { return k.Column == "title" }) ||
		len(m.panel(m.stage).SortKeys()) == 0 {
		t.Errorf("Unsorted on title: %+v", m.panel(m.stage).SortKeys())
	}
	// esc closes the card; a click off it closes it too
	if m = drive(m, esc); m.pop.Open {
		t.Error("esc left the card open")
	}
	if m = drive(m, rightClick("age"), testkit.Click(1, 39)); m.pop.Open {
		t.Error("a click off the card left it open")
	}
	// Show takes the column out of the header and the rows
	m = drive(m, rightClick("age"))
	if cardAct(t, &m, table.ShowIt); slices.Contains(m.shownColumns(m.stage), "age") || !m.pop.Open {
		t.Errorf("Show age: %q", m.shownColumns(m.stage))
	}
	m = drive(m, esc)
	// a right click on the header's gutter, off every name, opens nothing
	if m = drive(m, tea.MouseClickMsg{X: 1, Y: m.headerTop(), Button: tea.MouseRight}); m.pop.Open {
		t.Errorf("a right click off the names opened a card")
	}
	// nor does one under the rows, or a click on the rule of a table with no preset line
	if m = drive(m, tea.MouseClickMsg{X: 10, Y: 30, Button: tea.MouseRight}); m.pop.Open {
		t.Errorf("a right click under the rows opened a card")
	}
	m.noPresets = true
	if m = drive(m, testkit.Click(10, ruleY(m))); m.pop.Open {
		t.Errorf("a click on the rule opened a card")
	}
}

// Sorting always adds to the keys: the card's Ascending or Descending on a column not sorting puts it after the others,
// and on one sorting turns it in its place; None takes it out. The groups order the rows before any key, the outermost
// first, those that sort, so the rule numbers the keys after them; a group that does not sort, as a group does not by
// default, takes no number. Each mark stands on the rule under its column's name, so the names stand where they stood.
// A mark stands under its name's first cell, as a name over numbers does too. A plain click on a column's ¹ turns it
// to ₁, and one on ₁ stops the sort; no card opens.
func TestTheSortMarkTurnsAndTheCardSorts(t *testing.T) {
	m := loaded(t, viewing(""), tea.WindowSizeMsg{Width: 140, Height: 40})
	header := func() string { return lines(m)[m.headerTop()] }
	at := func(text string) int { return ansi.StringWidth(header()[:strings.Index(header(), text)]) }
	rule := func() string { return lines(m)[ruleY(m)] }
	onRule := func(text string) int { return ansi.StringWidth(rule()[:strings.Index(rule(), text)]) }
	unsorted := header()
	sorts := func() []grid.SortKey { return m.panel(m.stage).SortKeys() }
	// card acts on the line of a column's card that starts as line does, the card opened by a click on its name
	card := func(name, line string) {
		m = drive(m, testkit.Click(at(name), m.headerTop()))
		k := slices.IndexFunc(cardLines(m), func(l string) bool { return strings.HasPrefix(l[strings.Index(l, " ")+1:], line) })
		if k < 0 {
			t.Fatalf("no %q on %s's card: %q", line, name, cardLines(m))
		}
		m.pop.Act(&m, k)
		m = drive(m, esc)
	}
	// first gives the first line of a column's card that acts, as it reads
	first := func(column string) string { return table.OpenCard(&m, column, false).Lines(&m)[1].Text }
	if got := ansi.Strip(first("age")); got != grid.Unpicked+" ¹ Ascending" {
		t.Errorf("age, nothing sorting: %q", got)
	}
	card("age", "¹")
	card("id", "ⁿ")
	if got := ansi.Strip(first("id")); got != grid.Picked+" ⁿ Ascending" {
		t.Errorf("id, the second key: %q", got)
	}
	if !slices.Equal(sorts(), []grid.SortKey{{Column: "age"}, {Column: "id"}}) || !strings.Contains(rule(), "¹") ||
		onRule("¹") != at("age") || onRule("²") != at("id") || header() != unsorted {
		t.Fatalf("then by age, then by id: %+v\n%s\n%s", sorts(), header(), rule())
	}
	// the ¹ of age turns, and age keeps its place among the keys; no card opens
	if m = drive(m, testkit.Click(onRule("¹"), ruleY(m))); m.pop.Open ||
		!slices.Equal(sorts(), []grid.SortKey{{Column: "age", Desc: true}, {Column: "id"}}) {
		t.Fatalf("a click on age's ¹: %+v, card %v\n%s", sorts(), m.pop.Open, header())
	}
	// age is not the last key: a click on its ₁ turns it ascending again, in its place
	if m = drive(m, testkit.Click(onRule("₁")+1, ruleY(m))); !slices.Equal(sorts(), []grid.SortKey{{Column: "age"},
		{Column: "id"}}) {
		t.Fatalf("a click on age's ₁: %+v\n%s", sorts(), rule())
	}
	// a shift-click on age's ¹ stops age's sort, and id is the one key, with no number
	shift := testkit.Click(onRule("¹"), ruleY(m))
	shift.Mod = tea.ModShift
	if m = drive(m, shift); !slices.Equal(sorts(), []grid.SortKey{{Column: "id"}}) {
		t.Fatalf("a shift-click on age's ¹: %+v\n%s", sorts(), rule())
	}
	// id's ¹ turns to ₁, then off, a plain click each; then age sorts again, the one key
	for _, want := range [][]grid.SortKey{{{Column: "id", Desc: true}}, nil} {
		mark := "¹"
		if len(sorts()) == 1 && sorts()[0].Desc {
			mark = "₁"
		}
		if m = drive(m, testkit.Click(onRule(mark), ruleY(m))); !slices.Equal(sorts(), want) || m.pop.Open {
			t.Fatalf("a click on id's %s: %+v\n%s", mark, sorts(), rule())
		}
	}
	if m = drive(m, testkit.Click(at("age"), ruleY(m))); !slices.Equal(sorts(), []grid.SortKey{{Column: "age"}}) {
		t.Fatalf("a click under age: %+v\n%s", sorts(), rule())
	}
	// ₁ on id, not a key, comes after age; Descending is picked; ₁ on age turns it in its place
	if card("id", "ₙ"); !slices.Equal(sorts(), []grid.SortKey{{Column: "age"}, {Column: "id", Desc: true}}) {
		t.Fatalf("₁ on id: %+v", sorts())
	}
	if got := ansi.Strip(table.OpenCard(&m, "id", false).Lines(&m)[2].Text); got != grid.Picked+" ₙ Descending" {
		t.Errorf("₁ is not picked: %q", got)
	}
	if card("age", "ₙ"); !slices.Equal(sorts(), []grid.SortKey{{Column: "age", Desc: true}, {Column: "id", Desc: true}}) {
		t.Fatalf("₁ on age: %+v", sorts())
	}
	if card("id", table.Unsorted); !slices.Equal(sorts(), []grid.SortKey{{Column: "age", Desc: true}}) {
		t.Errorf("Unsorted on id: %+v", sorts())
	}
	// grouped by source, which does not sort, age, the one key, is ₁; once source sorts, it is 1, and age ₂
	m.setPanel(regroup(m.panel(m.stage), "source"))
	if !strings.Contains(rule(), "₁") || strings.Contains(rule(), "₂") {
		t.Errorf("age under a group that does not sort:\n%s", rule())
	}
	m.setPanel(m.panel(m.stage).SortedAs("source", grid.Descending))
	if !strings.Contains(rule(), "₂") || onRule("₂") != at("age") {
		t.Errorf("age under a group that sorts:\n%s\n%s", header(), rule())
	}
	if got := ansi.Strip(first("id")); got != grid.Unpicked+" ⁿ Ascending" {
		t.Errorf("id under a group: %q", got)
	}
}

// A click on the rule under a column's name sorts by it, ascending, after the other keys, where it does not sort yet,
// turns it descending where it sorts ascending, and stops it where it sorts descending, a step each click; the names stand where they stood, as the marks do not widen
// them. The cells after a name on the names' line sort nothing, and the rule off every name sorts nothing.
func TestAClickOnTheRuleUnderAColumnsNameSortsByIt(t *testing.T) {
	m := loaded(t, options(), tea.WindowSizeMsg{Width: 120, Height: 30})
	// under gives the screen column of the first cell of a column's name, which the rule's target starts under
	under := func(m Model, column string) int {
		x, _ := m.columnName(column)
		return x
	}
	after := func(m Model, column string) int {
		x, name := m.columnName(column)
		return x + ansi.StringWidth(name)
	}
	keys := func(m Model) int { return len(m.panel(m.stage).SortKeys()) }
	mark := func(m Model, column string) string { return m.panel(m.stage).Mark(column) }
	header := lines(m)[m.headerTop()]
	if m = drive(m, testkit.Click(after(m, "age"), m.headerTop())); keys(m) != 0 || m.pop.Open {
		t.Fatalf("a click right after age's name: %d keys, card %v", keys(m), m.pop.Open)
	}
	if m = drive(m, testkit.Click(1, ruleY(m))); keys(m) != 0 {
		t.Fatalf("a click on the rule's gutter: %d keys", keys(m))
	}
	if m = drive(m, testkit.Click(under(m, "age"), ruleY(m))); keys(m) != 1 || mark(m, "age") != "¹" || m.pop.Open {
		t.Fatalf("a click under age: %d keys, age %q, card %v", keys(m), mark(m, "age"), m.pop.Open)
	}
	// again turns it descending, again off, and again ascending, its last cell as its first
	for _, want := range []string{"₁", "", "¹"} {
		if m = drive(m, testkit.Click(after(m, "age")-1, ruleY(m))); keys(m) != len([]rune(want)) || mark(m, "age") != want {
			t.Fatalf("another click under age: %d keys, age %q, want %q", keys(m), mark(m, "age"), want)
		}
	}
	if m = drive(m, testkit.Click(under(m, "change"), ruleY(m))); keys(m) != 2 || mark(m, "change") != "²" {
		t.Errorf("a click under change: %d keys, change %q", keys(m), mark(m, "change"))
	}
	// a list sorts by size while a row holds all its values: the rule has the dim flatten mark under its name's first
	// cell, and the sort's after it, which a click sorts by how many values each row holds
	marks := func(m Model, column string, cells int) string {
		return ansi.Strip(ansi.Cut(m.rule(), under(m, column), under(m, column)+cells))
	}
	if got := marks(m, "assignees", 2); got != table.FlattenGlyph+Sortable {
		t.Errorf("the rule under assignees: %q", got)
	}
	if m = drive(m, testkit.Click(under(m, "assignees")+1, ruleY(m))); keys(m) != 3 || mark(m, "assignees") != "³" ||
		m.panel(m.stage).Field("assignees").Flat || !m.panel(m.stage).Counts("assignees") {
		t.Errorf("a click past assignees' flatten mark: %d keys, %q, flat %v", keys(m), mark(m, "assignees"),
			m.panel(m.stage).Field("assignees").Flat)
	}
	// a click on it flattens the list, a row per value, which keeps its sort, by value now; the mark is then the
	// gather's, which a click gathers
	if m = drive(m, testkit.Click(under(m, "assignees"), ruleY(m))); !m.panel(m.stage).Field("assignees").Flat ||
		marks(m, "assignees", 2) != table.GatherGlyph+"³" || m.panel(m.stage).Counts("assignees") {
		t.Fatalf("a click on assignees' flatten mark: flat %v, %q", m.panel(m.stage).Field("assignees").Flat,
			marks(m, "assignees", 2))
	}
	// and a click on it again gathers the values back on one row, sorted by size again
	if m = drive(m, testkit.Click(under(m, "assignees"), ruleY(m))); m.panel(m.stage).Field("assignees").Flat ||
		keys(m) != 3 || marks(m, "assignees", 2) != table.FlattenGlyph+"³" {
		t.Errorf("a click on the flatten mark, flat: %d keys, %q", keys(m), marks(m, "assignees", 2))
	}
	// a click after it turns the sort off, for the clicks below
	m = drive(m, testkit.Click(under(m, "assignees")+1, ruleY(m)), testkit.Click(under(m, "assignees")+1, ruleY(m)))
	if got := lines(m)[m.headerTop()]; got != header {
		t.Errorf("sorting moved the names:\n%s\n%s", header, got)
	}
	// the name itself opens the column's card under it, and sorts nothing
	if m = drive(m, testkit.Click(after(m, "change")-2, m.headerTop())); !m.pop.Open || m.pop.Field != "change" || keys(m) != 2 {
		t.Errorf("a click on change's name: card %+v, %d keys", m.pop, keys(m))
	}
}

// Shift with left or right, while a column's card is open, moves the column a place among those the header shows, the
// card with it, and names itself on the keys bar where it moves the column; at the header's end it moves nothing. From
// a name in the heading over the groups, it moves the column a depth among them, the card on its name where it now
// stands.
func TestShiftWithLeftOrRightMovesTheColumnOfAnOpenCard(t *testing.T) {
	m := loaded(t, viewing(""), tea.WindowSizeMsg{Width: 140, Height: 40})
	// the columns the header names, apart from source, which the view grouped the rows by as it loaded
	shown := func() []string {
		return slices.DeleteFunc(slices.Clone(m.shownColumns(m.stage)), func(id string) bool {
			_, name := m.columnName(id)
			return name == ""
		})
	}
	first := shown()[0]
	column := shown()[1]
	m.openColumnMenu(column)
	x0, _ := m.columnName(column)
	if !strings.Contains(keysOf(m), "⇧←→ move") {
		t.Errorf("the keys bar: %q", keysOf(m))
	}
	if m = drive(m, testkit.Press("shift+left")); shown()[0] != column || !m.pop.Open {
		t.Fatalf("shift+left on %s: %q", column, shown())
	}
	if _, left := table.Shifted(&m, column, -1); left || !strings.Contains(keysOf(m), "⇧←→ move") {
		t.Errorf("%s, first, moved further left", column)
	}
	if x, _ := m.columnName(column); x >= x0 || m.pop.X != x {
		t.Errorf("the card stayed behind: %d, at %d", x, m.pop.X)
	}
	if m = drive(m, testkit.Press("shift+right")); shown()[0] != first || shown()[1] != column {
		t.Errorf("shift+right: %q", shown())
	}
	if _, ok := table.ShiftStep("left"); ok {
		t.Error("left alone is a shift")
	}
	m = drive(m, esc)
	// a hidden column does not move
	if _, ok := table.Shifted(&m, "branch", 1); ok && m.panel(m.stage).Field("branch").Placement == grid.Hidden {
		t.Error("a hidden column moved")
	}
	// grouped by source, then assignees: shift+left in assignees' card, from its name in the heading over the groups,
	// makes it the outer group, the card on its name still, now first, which shift+left then cannot move; shift+right
	// moves it back in
	m.setPanel(regroup(m.panel(m.stage), "source", "assignees"))
	m.openFieldMenu("assignees")
	x0, _ = m.fieldName("assignees")
	if m = drive(m, testkit.Press("shift+left")); !slices.Equal(m.panel(m.stage).Groups(), []string{"assignees", "source"}) ||
		!m.pop.Open || !m.pop.Heading {
		t.Fatalf("shift+left on assignees' name: %q, card %+v", m.panel(m.stage).Groups(), m.pop)
	}
	if box, top, _ := m.pop.Box(&m); top != m.top() || len(box) == 0 {
		t.Errorf("the card stands at %d", top)
	}
	if x, _ := m.fieldName("assignees"); x >= x0 || m.pop.X != x {
		t.Errorf("the card stayed behind: %d, at %d", x, m.pop.X)
	}
	if m = drive(m, testkit.Press("shift+left")); m.panel(m.stage).Groups()[0] != "assignees" || !m.pop.Open {
		t.Errorf("shift+left on the outer group: %q", m.panel(m.stage).Groups())
	}
	if m = drive(m, testkit.Press("shift+right")); !slices.Equal(m.panel(m.stage).Groups(), []string{"source", "assignees"}) {
		t.Errorf("shift+right: %q", m.panel(m.stage).Groups())
	}
}

// A column's card holds the sections a field can use, in every column's order: its sort, by size for a list a row holds
// whole; then its rows, grouping before flattening; then its filter and rendering, and whether it shows. One that
// cannot sort has no sort; one that holds no list cannot flatten; and one that cannot group has no Group; one that can
// do neither has no rows.
func TestTheCardLeavesOutWhatTheFieldCannotDo(t *testing.T) {
	m := loaded(t, viewing(""), tea.WindowSizeMsg{Width: 140, Height: 40})
	card := func(column string) []string {
		m.pop = table.OpenCard(&m, column, false)
		return cardLines(m)
	}
	if got := card("title"); got[0] != "sort" || cardLine(m, "rows") >= 0 || got[len(got)-1] != grid.Ticked+" "+table.ShowIt {
		t.Errorf("title, which sorts and cannot group: %q", got)
	}
	if got := card("labels"); cardLine(m, "¹ Ascending"+table.BySize) != 1 || cardLine(m, table.GroupIt) < 0 ||
		cardLine(m, table.FlattenIt("label")) < 0 || cardLine(m, table.GatherIt("labels")) < 0 {
		t.Errorf("labels, a list, which sorts by size and groups: %q", got)
	}
	at := m.stage
	m.stage = slices.IndexFunc(m.stages, func(s stage) bool { return s.name == "builds" })
	if got := card("commit"); cardLine(m, "sort") >= 0 || cardLine(m, "rows") >= 0 {
		t.Errorf("builds' commit, which neither sorts nor groups: %q", got)
	}
	m.stage = at
	if got := card("assignees"); cardLine(m, table.GroupIt) < 0 || cardLine(m, "¹ Ascending"+table.BySize) != 1 ||
		cardLine(m, table.GroupIt) > cardLine(m, table.FlattenIt("assignee")) {
		t.Errorf("assignees, a list, which sorts by size, then groups, then flattens: %q", got)
	}
	m.pop = table.Popover{}
}
