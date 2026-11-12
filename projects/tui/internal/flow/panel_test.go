// The panel beside the rows, driven as the person drives it: tab and / give it the keyboard, typing narrows the rows,
// space hides a field, shift with an arrow or a drag moves one, and a click acts on a control. A click on a row's
// gutter opens it.
package flow

import (
	"reflect"
	"slices"
	"strconv"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/empowerite/kit/projects/tui/internal/grid"
	"github.com/empowerite/kit/projects/tui/internal/shell"
	"github.com/empowerite/kit/projects/tui/internal/table"
	"github.com/empowerite/kit/projects/tui/internal/testkit"
)

// searchKeys is the keys bar while the search has the keyboard: esc or enter leaves it, then the search's own, which
// hides every key of the screen's and the bar's it types.
const searchKeys = "⏎ rows type to search"

// menuKeys is the keys bar while a menu is open.
var menuKeys = shell.Path{shell.MenuKeys()}.String()

// typed gives a key press for each rune of the text, as a terminal sends them.
func typed(text string) []tea.Msg {
	var out []tea.Msg
	for _, r := range text {
		out = append(out, tea.KeyPressMsg{Code: r, Text: string(r)})
	}
	return out
}

// keys gives issues' rows' keys, as the panel lets them through, without the headings of their groups.
func keys(m Model) []string {
	var out []string
	for _, r := range m.rows(m.stage) {
		if !grid.Heads(r) {
			out = append(out, r.Key)
		}
	}
	return out
}

// drawn gives the stage's rows as the pane draws them, without the side.
func drawn(m Model) string {
	var out []string
	for _, l := range m.lines() {
		out = append(out, l.Text)
	}
	return strings.Join(out, "\n")
}

func TestSlashGivesTheSearchTheKeyboardAndTypingNarrowsTheRowsAndTheCount(t *testing.T) {
	m := loaded(t, options(), typed("/")...)
	if !m.inPanel || m.bar().Focused || lines(m)[23] != " "+searchKeys {
		t.Fatalf("/: in panel %v, keys %q", m.inPanel, lines(m)[23])
	}
	if !strings.Contains(raw(m), grid.Search+" ▏") {
		t.Errorf("the search under the keyboard has no caret:\n%s", view(m))
	}
	m = drive(m, typed("NOUN")...)
	// the tab says how many rows loaded, however few the search lets through
	if got := keys(m); !slices.Equal(got, []string{"#46"}) || !shows(m, stageGlyphs["issues"], "7") ||
		strings.Contains(barNames(m), "1/7") {
		t.Errorf("typing noun: rows %v\n%s", got, view(m))
	}
	// a number types into the search, and r does not refresh
	m = drive(m, typed("2r")...)
	if m.stage != 0 || m.stages[0].src.Loading || !strings.Contains(raw(m), grid.Search+" NOUN2r▏") {
		t.Errorf("a number or r in the search: stage %d\n%s", m.stage, view(m))
	}
	if !strings.Contains(view(m), "no row matches the panel") {
		t.Errorf("a search no row has:\n%s", view(m))
	}
	for range "NOUN2r" {
		m = drive(m, press(tea.KeyBackspace))
	}
	if len(keys(m)) != 7 || !shows(m, stageGlyphs["issues"], "7") {
		t.Errorf("the search emptied: rows %v", keys(m))
	}
	// a key neither the search nor the bar takes keeps the keyboard in the search
	if m = drive(m, press(tea.KeyPgDown)); !m.inPanel {
		t.Error("pgdown left the search")
	}
	// ⌃U clears what is typed, and the hint names it while there is some
	if m = drive(m, typed("NO")...); !strings.Contains(lines(m)[23], "⌃U clear") {
		t.Errorf("⌃U is not named: %q", lines(m)[23])
	}
	if m = drive(m, tea.KeyPressMsg{Code: 'u', Mod: tea.ModCtrl}); m.panel(m.stage).Query != "" || !m.inPanel {
		t.Errorf("⌃U: %q, in panel %v", m.panel(m.stage).Query, m.inPanel)
	}
	// esc, enter, down, and tab each give the rows focus, the filter kept
	m = drive(m, esc)
	for _, k := range []tea.KeyPressMsg{esc, press(tea.KeyEnter), down, press(tea.KeyTab)} {
		m = drive(m, typed("/")[0], typed("N")[0])
		if m = drive(m, k); m.inPanel || !m.inRows || m.panel(m.stage).Query == "" {
			t.Errorf("%s: in panel %v, in rows %v, %q", k, m.inPanel, m.inRows, m.panel(m.stage).Query)
		}
		m = drive(m, typed("/")[0], tea.KeyPressMsg{Code: 'u', Mod: tea.ModCtrl}, esc)
	}
}

func TestSlashFindsAndANumberWithAltLeavesTheSearchForItsStage(t *testing.T) {
	// past jig's #7 and kit's heading, to kit's first row
	m := drive(intoRows(loaded(t, options())), down, down)
	m = drive(m, typed("/")...)
	if !m.inPanel || !m.inRows {
		t.Fatalf("/: in panel %v, in rows %v", m.inPanel, m.inRows)
	}
	// the band stays on the row under the cursor while the search has the keyboard
	if !strings.HasPrefix(cursor(m), "#1") {
		t.Errorf("the cursor's row lost its band: %q", cursor(m))
	}
	m = drive(m, tea.KeyPressMsg{Code: '2', Mod: tea.ModAlt})
	if m.inPanel || m.stage != 1 || !m.bar().Focused {
		t.Errorf("alt-2 in the search: in panel %v, stage %d", m.inPanel, m.stage)
	}
	m = drive(m, typed("/x")...)
	if p := m.panel(m.stage); !m.inPanel || p.Query != "x" {
		t.Errorf("/ then x: in panel %v, search %q", m.inPanel, p.Query)
	}
	m = drive(m, esc)
	// each stage keeps its own search: issues' is empty
	if m.panel(1).Query != "x" || m.panel(0).Query != "" {
		t.Errorf("the searches: %q, %q", m.panel(1).Query, m.panel(0).Query)
	}
}

// glyphAt gives the column of the search's glyph on the screen's first line.
func glyphAt(m Model) int {
	top := strings.Split(raw(m), "\n")[0]
	return ansi.StringWidth(top[:strings.Index(top, grid.Search)])
}

// The search stands on the screen's top right, right-aligned one cell short of the edge, and stays there as / or a
// click on it gives it the keyboard; down then has nothing to move to. What is typed past its room grows it leftward.
func TestTheSearchStandsOnTheTopRight(t *testing.T) {
	m := loaded(t, options(), down)
	idle := m.width - 1 - ansi.StringWidth(grid.Search+" "+shell.Idle)
	if top := strings.Split(raw(m), "\n")[0]; !strings.HasSuffix(top, grid.Search+" "+shell.Idle) ||
		ansi.StringWidth(top) != m.width-1 || glyphAt(m) != idle || m.bar().Width != 120 {
		t.Fatalf("the search is not right-aligned, or the bar is %d wide: %q", m.bar().Width, top)
	}
	for _, enter := range []tea.Msg{typed("/")[0], testkit.Click(idle, 0)} {
		m = drive(m, esc, enter)
		if !m.inPanel || lines(m)[23] != " "+searchKeys {
			t.Fatalf("%v: in panel %v, keys %q", enter, m.inPanel, lines(m)[23])
		}
		if at := glyphAt(m); at != idle {
			t.Errorf("%v: the search moved to %d from %d", enter, at, idle)
		}
		if m = drive(m, press(tea.KeyPgDown)); !m.inPanel {
			t.Errorf("pgdown in the search left it")
		}
	}
	if m = drive(m, typed("noun")...); !slices.Equal(keys(m), []string{"#46"}) || glyphAt(m) != idle ||
		!strings.Contains(strings.Split(raw(m), "\n")[0], grid.Search+" noun") {
		t.Errorf("typing in the search: %v\n%s", keys(m), view(m))
	}
	// a query longer than its room grows it leftward, the title clipped before it; left empty, it is back at the edge
	long := " the noun that names it"
	if m = drive(m, typed(long)...); glyphAt(m) != m.width-1-ansi.StringWidth(grid.Search+" noun"+long+"▏") {
		t.Errorf("a long query: the search at %d", glyphAt(m))
	}
	for range "noun" + long {
		m = drive(m, press(tea.KeyBackspace))
	}
	if m = drive(m, esc); glyphAt(m) != idle {
		t.Errorf("the empty search stayed at %d", glyphAt(m))
	}
}

// A click on the ▸ of a row's first line opens it, from the search too; again, on its ▾, closes it; a click on the
// gutter elsewhere, as on the selection's cell, opens nothing.
func TestAClickOnAGutterOpensItsRow(t *testing.T) {
	m := loaded(t, options(), tea.WindowSizeMsg{Width: 80, Height: 60}, typed("/")[0])
	// #45 stands under its repository's heading, two cells in
	y, at := rowY(m, "#45"), slices.IndexFunc(m.rows(0), func(r row) bool { return r.Key == "#45" })
	if m = drive(m, testkit.Click(m.indent(0), y)); m.stages[0].Open["#45"] {
		t.Fatalf("a click on #45's selection cell opened it")
	}
	gutter := m.indent(0) + table.FoldCell
	if m = drive(m, testkit.Click(gutter, y)); m.inPanel || !m.inRows || m.stages[0].Cursor != at ||
		!m.stages[0].Open["#45"] {
		t.Errorf("a click on #45's gutter: in panel %v, cursor %d\n%s", m.inPanel, m.stages[0].Cursor, view(m))
	}
	if m = drive(m, testkit.Click(gutter, y)); m.stages[0].Open["#45"] {
		t.Errorf("a second click on the gutter did not close the row")
	}
}

func TestThePanelWithoutStagesDoesNothing(t *testing.T) {
	m := loaded(t, Options{}, typed("/")[0], testkit.Click(78, 0))
	if m.inPanel {
		t.Error("the search took the keyboard without a stage")
	}
	if m = drive(m, press('?')); m.legend.open {
		t.Error("? showed a legend without a stage")
	}
}

// A title's glyph leads to the issue's page, so it stays though every row shares it; drawn without it, as a preset may
// pin, its words lead nowhere.
func TestAClickOnATitlesGlyphOpensTheIssuesPage(t *testing.T) {
	m := loaded(t, options())
	// the first issue stands under source's heading, the view grouped by it as it loaded, and the header
	y := rowY(m, "#45")
	first := lines(m)[y]
	x := ansi.StringWidth(first[:strings.Index(first, Issue+" kit flow is")])
	browsed = nil
	if m = drive(m, testkit.Click(x, y)); len(browsed) != 1 ||
		browsed[0] != "https://github.com/empowerite/kit/issues/45" || !m.inRows || m.stages[0].Open["#45"] {
		t.Fatalf("a click on the glyph: %v, in rows %v", browsed, m.inRows)
	}
	// a page the browser cannot open says why in place of the keys, until the next key
	of[issue](*m.find("issues"))[6].URL = "https://refuse.example/45"
	m.rebuild()
	m = drive(m, testkit.Click(x, y))
	if last := lines(m)[23]; !strings.Contains(last, "could not open https://refuse.example/45") || !m.Noticing() {
		t.Errorf("a page that did not open: %q", last)
	}
	if m = drive(m, down); m.note.Text != "" || strings.Contains(lines(m)[23], "could not open") {
		t.Errorf("the next key kept the notice: %q", lines(m)[23])
	}
	// pinned to its text, the title's first word is no button
	m = drive(m, up)
	m.setPanel(m.panel(0).Rendered("title", "text"))
	first = lines(m)[y]
	browsed = nil
	if m = drive(m, testkit.Click(ansi.StringWidth(first[:strings.Index(first, "kit flow is")]), y)); len(browsed) != 0 {
		t.Errorf("a click on the title's first word, drawn without its glyph: %v", browsed)
	}
}

func TestAColumnNoRowFillsHasNoCheckbox(t *testing.T) {
	m := loaded(t, options())
	m.apply(m.viewsOf(0).Chosen("Backlog"))
	m.cols = table.Columns{Open: true}
	if !strings.Contains(view(m), grid.Ticked+" branch") {
		t.Fatalf("the backlog's branch has no checkbox while an issue has one:\n%s", view(m))
	}
	for k := range of[issue](*m.find("issues")) {
		of[issue](*m.find("issues"))[k].Branch = ""
	}
	m.rebuild()
	if strings.Contains(view(m), grid.Ticked+" branch") || !strings.Contains(view(m), grid.Ticked+" title") {
		t.Errorf("branch, which no issue has, keeps its checkbox:\n%s", view(m))
	}
}

func TestTheWheelScrollsTheRowsAndLeavesTheCursor(t *testing.T) {
	m := loaded(t, options(), tea.WindowSizeMsg{Width: 120, Height: 9 + titleHeight}, tab, tab, down)
	turn := func(b tea.MouseButton) tea.Msg { return tea.MouseWheelMsg{X: 10, Y: m.headerTop() + 1, Button: b} }
	if m = drive(m, turn(tea.MouseWheelDown)); m.find("builds").Offset == 0 || m.find("builds").Cursor != 0 {
		t.Fatalf("the wheel down: offset %d, cursor %d\n%s", m.find("builds").Offset, m.find("builds").Cursor, view(m))
	}
	if m = drive(m, turn(tea.MouseWheelUp), turn(tea.MouseWheelUp)); m.find("builds").Offset != 0 {
		t.Errorf("the wheel up past the top: offset %d", m.find("builds").Offset)
	}
	// on the bar, or with no stage, the wheel scrolls nothing
	if m = drive(m, tea.MouseWheelMsg{X: 10, Y: 1, Button: tea.MouseWheelDown}); m.find("builds").Offset != 0 {
		t.Errorf("the wheel on the bar scrolled the rows")
	}
	loaded(t, Options{}, turn(tea.MouseWheelDown))
}

// Sort by, in the menu a click on a sortable column's name drops, sorts the rows by it, ▲ on the rule under the name; a
// click on ▲ turns it ▼;
// an option-click adds a column after it; the menu of a name that does not sort, as builds' commit, dims its sorts.
func TestSortByInAColumnsMenuSortsTheRows(t *testing.T) {
	m := loaded(t, options(), tea.WindowSizeMsg{Width: 160, Height: 24})
	m.apply(m.viewsOf(0).Chosen("Backlog"))
	unfold(&m) // the source, one tracker's on every row, a column
	at := func(name string) int { return strings.Index(lines(m)[m.headerTop()], name) }
	order := func() string { return strings.Join(keys(m), " ") }
	// under gives the mark the rule draws under a column's name, from its first cell, past the rule and the dim mark of
	// a column that does not sort
	under := func(name string) string {
		x := ansi.StringWidth(lines(m)[m.headerTop()][:at(name)])
		return strings.Trim(ansi.Strip(ansi.Cut(lines(m)[ruleY(m)], x, x+ansi.StringWidth(name)+1)), "─"+Sortable)
	}
	if m = drive(m, testkit.Click(at("age"), m.headerTop()), press(tea.KeyEnter), esc); under("age") != "▲" {
		t.Fatalf("a click on age: %s\n%s", order(), view(m))
	}
	if got := order(); got != "#45 #46 #7" {
		t.Errorf("by age, the youngest first: %s", got)
	}
	if m = drive(m, testkit.Click(at("age"), ruleY(m))); under("age") != "▼" || order() != "#7 #46 #45" {
		t.Errorf("a click on its ▲, the oldest first: %s\n%s", order(), view(m))
	}
	option := tea.MouseClickMsg{X: at("source"), Y: m.headerTop(), Button: tea.MouseLeft, Mod: tea.ModAlt}
	if m = drive(m, option); under("source") != "▲²" || under("age") != "▼₁" {
		t.Errorf("an option-click on source:\n%s", view(m))
	}
	// shift, which a terminal often keeps for its own selection, adds nothing: the click opens the card, as ever
	shift := tea.MouseClickMsg{X: at("source"), Y: m.headerTop(), Button: tea.MouseLeft, Mod: tea.ModShift}
	if m = drive(m, shift); len(m.panel(m.stage).SortKeys()) != 2 || !m.pop.Open {
		t.Errorf("a shift-click on source: %+v", m.panel(m.stage).SortKeys())
	}
	// builds' commit does not sort
	m = drive(m, esc)
	m = drive(m, tabsTo("builds")...)
	if m = drive(m, testkit.Click(at("commit"), m.headerTop())); !m.pop.Open || cardLine(m, "sort") >= 0 ||
		m.stages[m.stage].name != "builds" {
		t.Errorf("commit's card sorts: %q", cardLines(m))
	}
}

// g in the columns dialog groups the rows by a field, at the last depth, within the groups there are; a grouped column
// leaves the header and the rows, whose heading names its value, and the rows and the header stand in two cells a
// level, a click on a name finding its column there.
func TestGroupingNestsAndIndentsTheRows(t *testing.T) {
	m := loaded(t, options(), tea.WindowSizeMsg{Width: 160, Height: 30})
	m.setPanel(regroup(m.panel(m.stage)))
	m.cols = table.Columns{Open: true}
	for _, id := range []string{"source", "assignees"} {
		m.cols.At = slices.Index(m.cols.Listed(&m), id)
		m = drive(m, typed("g")...)
	}
	if got := m.panel(m.stage).Groups(); !slices.Equal(got, []string{"source", "assignees"}) {
		t.Fatalf("source, then assignees, into the groups: %q", got)
	}
	names, _ := m.header()
	header := ansi.Strip(names[0])
	if strings.Contains(header, "source") || strings.Contains(header, "assignees") || !strings.HasPrefix(header, strings.Repeat(" ", 4+3)) {
		t.Errorf("the header of rows grouped two deep:\n%s", view(m))
	}
	first := slices.IndexFunc(lines(m), func(l string) bool { return strings.Contains(l, "#1") })
	if !strings.HasPrefix(lines(m)[first], "    ") || strings.Contains(lines(m)[first], GitHub) {
		t.Errorf("a row grouped two deep is not indented four cells, or keeps its source:\n%s", view(m))
	}
	// a click on a name in the indented header opens its card, whose ▲ Ascending sorts by it, ▲ on the rule under it
	if m = drive(m, esc, testkit.Click(strings.Index(lines(m)[m.headerTop()], "age"), m.headerTop()), press(tea.KeyEnter), esc); m.panel(m.stage).Mark("age") != "▲" ||
		ansi.Cut(lines(m)[ruleY(m)], strings.Index(lines(m)[m.headerTop()], "age"), strings.Index(lines(m)[m.headerTop()], "age")+1) != "▲" {
		t.Errorf("a click on age in the indented header:\n%s", view(m))
	}
	// g on a column grouping alone stops it
	if p := m.panel(m.stage).Grouped("assignees", false).Grouped("assignees", false); len(p.Groups()) != 0 {
		t.Errorf("g twice: %q", p.Groups())
	}
}

// A click on a column's name with ctrl held, where the terminal sends it, groups the rows by it alone; with option too,
// within the groups there are. The same click, or a right click, on a group's heading stops grouping by its column
// alone, the other groups kept; on a row, it does nothing.
func TestACtrlClickGroupsAndOnAHeadingUngroups(t *testing.T) {
	m := loaded(t, options(), tea.WindowSizeMsg{Width: 160, Height: 30})
	unfold(&m) // the source, one tracker's on every row, a column
	at := func(name string) int { return strings.Index(lines(m)[m.headerTop()], name) }
	heading := func(text string) int {
		return slices.IndexFunc(lines(m), func(l string) bool {
			return strings.Contains(l, grid.Unfolded) && strings.Contains(l, text)
		})
	}
	group := func() []string { return m.panel(m.stage).Groups() }
	if m = drive(m, tea.MouseClickMsg{X: at("source"), Y: m.headerTop(), Button: tea.MouseLeft, Mod: tea.ModCtrl}); !slices.Equal(group(), []string{"source"}) {
		t.Fatalf("a ctrl-click on source: %q\n%s", group(), view(m))
	}
	nest := tea.MouseClickMsg{X: at("assignees"), Y: m.headerTop(), Button: tea.MouseLeft, Mod: tea.ModAlt | tea.ModCtrl}
	if m = drive(m, nest); !slices.Equal(group(), []string{"source", "assignees"}) {
		t.Fatalf("a ctrl-option-click on assignees: %q\n%s", group(), view(m))
	}
	before := group()
	if m = drive(m, tea.MouseClickMsg{X: 10, Y: heading("#1"), Button: tea.MouseRight}); !slices.Equal(group(), before) {
		t.Errorf("a right-click on a row: %q", group())
	}
	if m = drive(m, tea.MouseClickMsg{X: 10, Y: heading("jeffrey-aguilera"), Button: tea.MouseRight}); !slices.Equal(group(), []string{"source"}) {
		t.Fatalf("a right-click on an assignees heading: %q\n%s", group(), view(m))
	}
	if m = drive(m, tea.MouseClickMsg{X: 10, Y: heading("▾ github"), Button: tea.MouseLeft, Mod: tea.ModCtrl}); len(group()) != 0 {
		t.Errorf("a ctrl-click on a source heading: %q\n%s", group(), view(m))
	}
	// a plain click on a name opens its card, and groups nothing
	if m = drive(m, testkit.Click(at("age"), m.headerTop())); len(group()) != 0 || !m.pop.Open {
		t.Errorf("a click on age: group %q, sort %+v", group(), m.panel(m.stage).SortKeys())
	}
}

// Issues group by change: the issues one pull request carries stand under its heading, and those without one, in the
// backlog, under none.
func TestIssuesGroupByChange(t *testing.T) {
	m := loaded(t, options(), tea.WindowSizeMsg{Width: 160, Height: 30})
	m.setPanel(regroup(m.panel(m.stage), "change"))
	if got := m.panel(m.stage).Groups(); !slices.Equal(got, []string{"change"}) {
		t.Fatalf("grouped by change: %q", got)
	}
	headings := 0
	for _, r := range m.rows(m.stage) {
		if r.Aside && m.panel(m.stage).Heading(r) == "change" {
			headings++
		}
	}
	if headings == 0 || !strings.Contains(view(m), grid.Unfolded+" change "+grid.Unfolded+" ") {
		t.Errorf("no heading per change:\n%s", view(m))
	}
}

// A view, as it loads, groups the rows by each column whose value is the same on every row, where it groups by nothing:
// as a view of the issues underway that has their source, organization, and repository as columns, while one tracker
// and one repository hold every issue: a group as any other, under the heading over the groups, ▤₁ in the columns
// dialog, and the preset line still names the view. Once grouped, it is the person's: a search that leaves one row, or
// the rows loaded again, change nothing, and Ungroup ungroups it for good. Setting the view again groups again; with
// kittui's setting, Auto-group, off, a view set groups nothing. Underway groups by the three as the catalog says.
func TestAViewAsItLoadsGroupsByAColumnTheSameOnEveryRow(t *testing.T) {
	m := loaded(t, options(), tea.WindowSizeMsg{Width: 160, Height: 30})
	m.apply(m.viewsOf(0).Chosen("Underway"))
	heading := func(m Model) string {
		if rows := m.rows(m.stage); len(rows) > 1 && grid.OverGroups(rows[0]) {
			return ansi.Strip(rows[0].Cells[0].Values[0])
		}
		return ""
	}
	// three levels, source, organization, and repository, each of one value: the first's fold is the only one, the
	// others' dim separators, and the rows' would repeat it
	both := []string{"source", "organization", "repository"}
	want := grid.Unfolded + " source " + grid.Joined + " organization " + grid.Joined + " repository 4"
	if got := heading(m); got != want || !slices.Equal(m.panel(m.stage).Groups(), both) || m.is(m.stage) != "Underway" {
		t.Fatalf("Underway's heading %q, groups %q, view %q:\n%s", got, m.panel(m.stage).Groups(), m.is(m.stage), view(m))
	}
	column := table.Setting{Placement: table.PlaceColumn}
	flat := table.Preset{Name: "Flat", Fields: map[string]table.Setting{"source": column, "organization": column,
		"repository": column, "status": {Placement: table.PlaceHidden,
			Filter: &table.ValueFilter{Include: []string{"underway"}}}}}
	m.saved["issues"] = []table.Preset{flat}
	if m.apply(flat); heading(m) != want || !slices.Equal(m.panel(m.stage).Groups(), both) || m.is(m.stage) != "Flat" ||
		!strings.Contains(m.cols.Line(&m, "source"), grid.GroupMark(1)) {
		t.Fatalf("Flat's heading %q, groups %q, view %q:\n%s", heading(m), m.panel(m.stage).Groups(), m.is(m.stage),
			view(m))
	}
	// a search that leaves one row, and the rows loaded again, leave the group as it is
	p := m.panel(m.stage)
	p.Query = "kit commit"
	if m.setPanel(p); !slices.Equal(m.panel(m.stage).Groups(), both) {
		t.Errorf("one row left: %q", m.panel(m.stage).Groups())
	}
	p.Query = ""
	m.setPanel(p)
	m = drive(m, m.again("issues", false)())
	if !slices.Equal(m.panel(m.stage).Groups(), both) {
		t.Errorf("loaded again: %q", m.panel(m.stage).Groups())
	}
	// ungrouped, it stays so through a load; the preset line names the view, as the panel is as it says
	m.setPanel(regroup(m.panel(m.stage)))
	if m = drive(m, m.again("issues", false)()); len(m.panel(m.stage).Groups()) > 0 || !rowsAlone(heading(m)) ||
		!strings.Contains(lines(m)[m.headerTop()], "source") {
		t.Errorf("ungrouped, then loaded again: %q\n%s", m.panel(m.stage).Groups(), view(m))
	}
	// the view set again groups again; with Auto-group off, it groups nothing
	if m.apply(m.chosen(m.stage)); !slices.Equal(m.panel(m.stage).Groups(), both) {
		t.Errorf("the view set again: %q", m.panel(m.stage).Groups())
	}
	if unfold(&m); !rowsAlone(heading(m)) || m.panel(m.stage).Fold || len(m.panel(m.stage).Groups()) > 0 ||
		!strings.Contains(lines(m)[m.headerTop()], "source") {
		t.Errorf("Auto-group off: heading %q, groups %q\n%s", heading(m), m.panel(m.stage).Groups(), view(m))
	}
}

// A group's heading folds its rows away and unfolds them: left on it folds, right unfolds, the cursor staying on the
// heading, and a click on its ▾ or ▸, and there alone, does either; ▾ turns ▸, the keys say which, and the tab's count
// stays the rows the panel lets through, folded or not. The heading over the groups never folds.
func TestAGroupsHeadingFoldsAndUnfoldsItsRows(t *testing.T) {
	m := loaded(t, options(), tea.WindowSizeMsg{Width: 160, Height: 30})
	count, rows := m.count(m.stage), len(m.rows(m.stage))
	// left on the heading over the groups folds nothing
	m.stages[m.stage].Cursor, m.inRows = 0, true
	if m = drive(m, left); len(m.rows(m.stage)) != rows {
		t.Fatalf("left on the heading over the groups: %d rows of %d", len(m.rows(m.stage)), rows)
	}
	// the cursor on source's heading, github's
	m.stages[m.stage].Cursor = 1
	if last := lines(m)[len(lines(m))-1]; !strings.Contains(last, "← hide") {
		t.Errorf("the keys on an open heading: %q", last)
	}
	header, _ := m.headerLines()
	m = drive(m, left)
	// with every row folded away there is nothing to stand over, so the header goes, every column where it was still
	if got, _ := m.headerLines(); m.headerTop() != -1 || !slices.Equal(got, header) {
		t.Errorf("the header with every row folded: at %d, %q, was %q", m.headerTop(), got, header)
	}
	heading := ansi.Strip(m.rows(m.stage)[1].Cells[0].Values[0])
	if len(m.rows(m.stage)) != 2 || !strings.HasPrefix(heading, grid.Folded+" ") || m.count(m.stage) != count ||
		m.stages[m.stage].Cursor != 1 {
		t.Fatalf("left on the heading: %d rows, %q, count %s", len(m.rows(m.stage)), heading, m.count(m.stage))
	}
	if last := lines(m)[len(lines(m))-1]; !strings.Contains(last, "→ show") {
		t.Errorf("the keys on a folded heading: %q", last)
	}
	if m = drive(m, right); len(m.rows(m.stage)) != rows {
		t.Fatalf("right on the heading: %d rows", len(m.rows(m.stage)))
	}
	// a click on the heading's ▾ folds it, and another, on its ▸, unfolds it; one on its value folds nothing
	fold := table.FoldAt(m.rows(m.stage)[1])
	if m = drive(m, testkit.Click(fold+3, m.rowsTop())); len(m.rows(m.stage)) != rows {
		t.Fatalf("a click on the heading's value: %d rows of %d", len(m.rows(m.stage)), rows)
	}
	if m = drive(m, testkit.Click(fold, m.rowsTop())); len(m.rows(m.stage)) != 2 || m.pop.Open {
		t.Fatalf("a click on the heading's ▾: %d rows, card %v\n%s", len(m.rows(m.stage)), m.pop.Open, view(m))
	}
	if m = drive(m, testkit.Click(fold, m.rowsTop())); len(m.rows(m.stage)) != rows {
		t.Fatalf("a click on the heading's ▸: %d rows", len(m.rows(m.stage)))
	}
}

// The rows under a group's heading stand in from it, whether the view grouped them as it loaded or the person did: the
// heading's ▾ two cells left of its rows' ▸, and a heading nested under another two cells in from it, the headings of
// the values in from the heading over the groups, which names their columns.
func TestTheRowsUnderAHeadingStandInFromIt(t *testing.T) {
	m := loaded(t, options(), tea.WindowSizeMsg{Width: 120, Height: 30})
	// column gives the cell a line of the pane draws its first glyph of the kind at
	column := func(m Model, has, glyph string) int {
		k := slices.IndexFunc(lines(m), func(l string) bool { return strings.Contains(l, has) })
		if k < 0 {
			t.Fatalf("no line has %q:\n%s", has, view(m))
		}
		before, _, _ := strings.Cut(lines(m)[k], glyph)
		return ansi.StringWidth(before)
	}
	if h, r := column(m, "▾ kit", grid.Unfolded), column(m, "#45", "▸"); r != h+2 {
		t.Errorf("grouped as the view loaded, the heading's ▾ at %d, a row's ▸ at %d:\n%s", h, r, view(m))
	}
	unfold(&m)
	m.setPanel(regroup(m.panel(m.stage), "source", "assignees"))
	over := column(m, "source "+grid.Joined+" assignees", grid.Unfolded)
	outer, inner := column(m, "▾ github", grid.Unfolded), column(m, "jeffrey-aguilera", grid.Unfolded)
	if r := column(m, "#45", "▸"); outer != over || inner != outer+2 || r != inner+2 {
		t.Errorf("grouped by hand, the headings' ▾ at %d, %d and %d, a row's ▸ at %d:\n%s", over, outer, inner, r, view(m))
	}
}

// popLineAt gives the screen line of the card's line that has the text, after its glyph, and a column inside it: the
// card beside the columns dialog's list while it has the keyboard, else the card open.
func popLineAt(t *testing.T, m Model, text string) (x, y int) {
	t.Helper()
	box, top, left := m.pop.Box(&m)
	if m.cols.InCard {
		box, top, left = m.cols.CardBox(&m)
	}
	for k, l := range box {
		if strings.Contains(ansi.Strip(l), " "+text+" ") {
			return left + 3, top + k
		}
	}
	t.Fatalf("the popover has no %q:\n%s", text, view(m))
	return -1, -1
}

// popoverOf gives a field's card beside the columns dialog's list the keyboard, enter on its line, on its filter's mode.
func popoverOf(m Model, id string) Model {
	m.cols = table.Columns{Open: true}
	m.cols.At = slices.Index(m.cols.Listed(&m), id)
	m = drive(m, press(tea.KeyEnter))
	m.cols.CardAt = m.cols.Card(&m).AtFilter(&m).At
	return m
}

// Enter on a field's line in the columns dialog gives its card, beside the list, the keyboard: up and down move between
// the lines that act, past the headings; space takes a rendering, and toggles a value of the filter, or its mode; each
// applies at once, the rows and the preset line's chips following it. The list's line shows the sort, so the card
// beside it has none. Esc gives the list the keyboard again. A click on a line acts on it, and a click off the dialog
// closes it.
func TestAFieldsPopoverFiltersAndDrawsIt(t *testing.T) {
	m := loaded(t, viewing(""))
	m.apply(m.viewsOf(0).Chosen("Backlog"))
	m = popoverOf(m, "assignees")
	if !m.cols.InCard || m.cols.Card(&m).Field != "assignees" || !strings.Contains(view(m), "╭─ assignees ") {
		t.Fatalf("enter on assignees:\n%s", view(m))
	}
	// the keyboard on the filter's mode; there is no sort to take
	if v := view(m); strings.Contains(v, "ascending") || !strings.Contains(m.keys().String(), "␣ toggle") {
		t.Errorf("the card's first line: %q\n%s", m.keys().String(), v)
	}
	// @me, the caller, is ada; checked, only her issue shows, and the preset line has its chip
	m = drive(m, down)
	if keys := m.keys().String(); keys != "← fields ↑↓ move ␣ toggle" || !strings.Contains(view(m), "@me ada") {
		t.Errorf("the card's keys on @me: %q", keys)
	}
	if m = drive(m, press(tea.KeySpace)); !slices.Equal(keys(m), []string{"#7"}) ||
		!strings.Contains(presetText(m), "assignees: @me") {
		t.Errorf("@me: %v, %q", keys(m), presetText(m))
	}
	// the mode, above it, holds back the values checked
	if m = drive(m, up, press(tea.KeySpace)); !slices.Equal(keys(m), []string{"#46", "#45"}) ||
		!strings.Contains(presetText(m), "assignees: ¬@me") {
		t.Errorf("exclude: %v, %q", keys(m), presetText(m))
	}
	// auto keeps the glyph of the one row with a login, as no other row shares it, and says so; text, pinned, drops it
	if !strings.Contains(drawn(m), Owner) || !strings.Contains(view(m), "auto glyph+text") {
		t.Errorf("assignees drawn by auto:\n%s", view(m))
	}
	x, y := popLineAt(t, m, "text")
	if m = drive(m, testkit.Click(x, y)); m.panel(0).Field("assignees").Rendering != "text" || strings.Contains(drawn(m), Owner) {
		t.Errorf("a click on text: %+v\n%s", m.panel(0).Field("assignees"), view(m))
	}
	// esc gives the list the keyboard, which marks the field filtered
	if m = drive(m, esc); m.cols.InCard || !m.cols.Open || !strings.Contains(view(m), "assignees ⊘") {
		t.Errorf("esc: in the card %v, dialog %v\n%s", m.cols.InCard, m.cols.Open, view(m))
	}
	// a click on the preset line's chip, the dialog closed, opens the card under it, on its filter
	m = drive(m, esc)
	control := lines(m)[m.top()]
	chip := ansi.StringWidth(control[:strings.Index(control, "assignees: ¬@me")])
	if m = drive(m, testkit.Click(chip+2, m.top())); !m.pop.Open || !m.pop.Chip || m.pop.Field != "assignees" ||
		!m.pop.Lines(&m)[m.pop.At].Mode {
		t.Errorf("a click on the chip: %+v", m.pop)
	}
	m = drive(m, esc)
	m.cols = table.Columns{Open: true}
	// a click off the dialog closes it, acting on nothing
	m = drive(m, press(tea.KeyEnter))
	before := m.panel(0).Fields
	if m = drive(m, testkit.Click(2, m.rowsTop())); m.cols.Open || !reflect.DeepEqual(m.panel(0).Fields, before) {
		t.Errorf("a click off the dialog: open %v", m.cols.Open)
	}
	// an enum's values are the catalog's, with their glyphs, then none
	m = popoverOf(drive(m, esc, tab), "state")
	if v := view(m); m.cols.Card(&m).Field != "state" || !strings.Contains(v, grid.Unticked+" 🔨 draft") ||
		!strings.Contains(v, grid.Unticked+" 🏁 ready") || !strings.Contains(v, grid.Unticked+" none") {
		t.Errorf("state's popover:\n%s", v)
	}
}

// Without the caller's login, @me stands alone, and checked lets no row through.
func TestTheCallerIsNoOneWhileKitCannotSayWhoItIs(t *testing.T) {
	t.Setenv("FAKE_FLOW_REFUSE", "me")
	// the popover opens on its filter's mode; @me is under it
	m := drive(popoverOf(loaded(t, options()), "assignees"), down)
	if m.me != "" || !strings.Contains(view(m), grid.Unticked+" @me ") || strings.Contains(view(m), "@me ada") {
		t.Fatalf("@me without a login: %q\n%s", m.me, view(m))
	}
	if m = drive(m, press(tea.KeySpace)); len(keys(m)) != 0 || !strings.Contains(view(m), "no row matches the panel") {
		t.Errorf("@me checked without a login: %v", keys(m))
	}
}

// A click on a column's name opens its card under the name, its title over the name, and sorts nothing; a second click
// on the name, the card's title now, closes it. Enter on its first line sorts by it, ascending. Where the terminal
// sends option with a click, the click adds the column after the other sort keys, and opens nothing.
func TestAClickOnAColumnsNameOpensItsCard(t *testing.T) {
	m := loaded(t, options(), tea.WindowSizeMsg{Width: 160, Height: 30 + titleHeight})
	unfold(&m)
	at := func(name string) int {
		l := lines(m)[m.headerTop()]
		return ansi.StringWidth(l[:strings.Index(l, name)])
	}
	x := at("age")
	click := func(name string, mod tea.KeyMod) {
		m = drive(m, tea.MouseClickMsg{X: at(name), Y: m.headerTop(), Button: tea.MouseLeft, Mod: mod})
	}
	sorts := func() []grid.SortKey { return m.panel(m.stage).SortKeys() }
	if click("age", 0); len(sorts()) != 0 || !m.pop.Open || m.pop.Field != "age" || !m.pop.Header {
		t.Fatalf("a click on age: %+v, card %+v", sorts(), m.pop)
	}
	if box, top, left := m.pop.Box(&m); top != m.headerTop() || !strings.Contains(ansi.Strip(box[0]), "╭─ age ") ||
		left+3 != x || !strings.Contains(view(m), "╭─ age ") {
		t.Errorf("the card's title is not over the name: %d, %d\n%s", top, left, view(m))
	}
	if x, name := m.columnName("age"); name != "age" || x != at("age") {
		t.Errorf("age's name in the header: %q at %d, not %d", name, x, at("age"))
	}
	if m = drive(m, testkit.Click(x, m.headerTop())); m.pop.Open || len(sorts()) != 0 {
		t.Fatalf("a second click on age: %+v, card %v", sorts(), m.pop.Open)
	}
	click("age", 0)
	if m = drive(m, press(tea.KeyEnter), esc); !slices.Equal(sorts(), []grid.SortKey{{Column: "age"}}) || m.pop.Open {
		t.Fatalf("▲ Ascending on age: %+v", sorts())
	}
	if click("source", tea.ModAlt); len(sorts()) != 2 || m.pop.Open {
		t.Errorf("an option-click on source: %+v, card %v", sorts(), m.pop.Open)
	}
}

// rowsAlone says whether the heading over the groups is the rows' alone, as over rows that group by nothing: their
// fold and their count.
func rowsAlone(heading string) bool {
	fold, count, ok := strings.Cut(heading, " ")
	_, err := strconv.Atoi(count)
	return ok && (fold == grid.Unfolded || fold == grid.Folded) && err == nil
}
