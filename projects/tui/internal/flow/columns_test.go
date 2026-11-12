// The columns dialog: what the rows show, in what order, sorted and grouped how, all in one place.
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

// columnsAt gives the screen line and column of a field's line in the open columns dialog, its box's cell.
func columnsAt(t *testing.T, m Model, id string) (x, y int) {
	t.Helper()
	k := slices.Index(m.cols.Listed(&m), id)
	_, top, left := m.cols.Box(&m)
	for y := top; y < top+30; y++ {
		if m.cols.Row(&m, y) == k {
			return left + 2, y
		}
	}
	t.Fatalf("no line for %s", id)
	return -1, -1
}

// 📇 Fields stands on the control line, before the preset line's selector, and a click on it opens the dialog under the
// header's right end, the header drawing no icon of its own: the fields the rows group by first, a rule under them, then the columns
// as the rows stand them, then the hidden, each with whether it shows, its sort, and its group's depth.
func TestTheColumnsDialogListsTheGroupsFirstThenTheColumns(t *testing.T) {
	m := loaded(t, viewing(""), tea.WindowSizeMsg{Width: 110, Height: 30})
	if strings.Contains(lines(m)[m.headerTop()], "◫") || strings.Contains(lines(m)[m.headerTop()], Fields) {
		t.Errorf("an icon on the header: %q", lines(m)[m.headerTop()])
	}
	// organization, which the view groups by, hidden, to stand among the hidden, after the columns
	m.setPanel(regroup(m.panel(0).SortedAs("age", grid.Descending), "assignees").Placed("organization", grid.Hidden))
	control := lines(m)[m.top()]
	at := ansi.StringWidth(control[:strings.Index(control, table.FieldsButton)])
	if at+ansi.StringWidth(table.FieldsButton)+2 != m.dropStart() || !strings.HasPrefix(table.FieldsButton, Fields) {
		t.Fatalf("📇 Fields at %d, the selector at %d: %q", at, m.dropStart(), control)
	}
	if m = drive(m, testkit.Click(at+1, m.top())); m.presetsMenu.Open || !m.cols.Open || m.cols.At != 0 {
		t.Fatalf("📇 Fields did not open the dialog: menu %v, dialog %+v", m.presetsMenu.Open, m.cols)
	}
	if got := m.cols.Listed(&m); got[0] != "assignees" || slices.Index(got, "organization") < slices.Index(got, "title") {
		t.Errorf("the fields: %q", got)
	}
	box, top, left := m.cols.Box(&m)
	card, _, cleft := m.cols.CardBox(&m)
	text := ansi.Strip(strings.Join(box, "\n"))
	if top != m.top() || left+ansi.StringWidth(box[0]) != cleft || cleft+ansi.StringWidth(card[0]) != m.paneWidth() ||
		!strings.Contains(text, "issues fields") ||
		!strings.Contains(text, "field") || !strings.Contains(text, grid.Ticked+" assignees") ||
		!strings.Contains(text, "▼") || !strings.Contains(text, grid.Unticked+" organization") ||
		!strings.Contains(keysOf(m), "s sort") {
		t.Errorf("the dialog at %d, %d:\n%s\n%q", top, left, text, keysOf(m))
	}
	if m.cols.Row(&m, top) != -1 || m.cols.Row(&m, top+1) != -1 || m.cols.Row(&m, top+2) != 0 || m.cols.Row(&m, top+3) != -1 ||
		m.cols.Row(&m, top+4) != 1 || m.cols.Row(&m, top+40) != -1 {
		t.Errorf("the rows: edge %d, headings %d, assignees %d, rule %d, id %d", m.cols.Row(&m, top), m.cols.Row(&m, top+1),
			m.cols.Row(&m, top+2), m.cols.Row(&m, top+3), m.cols.Row(&m, top+4))
	}
}

// What the dialog sets applies at once: space hides a field or shows it, s turns its sort, none, ascending, descending,
// none, g groups by it within the groups or no longer, and shift with up or down moves it, the keyboard with it, into
// the groups or out of them. A click on a line's box, its sort, or its group does as space, s, or g; a click on its
// label, or enter, gives its card beside the list the keyboard, which esc gives back; esc, or a click off the dialog,
// closes it.
func TestTheColumnsDialogAppliesAtOnce(t *testing.T) {
	m := loaded(t, viewing(""), tea.WindowSizeMsg{Width: 110, Height: 30})
	// the organization and the repository, columns between source and assignees once ungrouped, have tests of their own
	m.setPanel(regroup(m.panel(0)).Placed("organization", grid.Hidden).Placed("repository", grid.Hidden))
	m.cols = table.Columns{Open: true}
	at := func(id string) { m.cols.At = slices.Index(m.cols.Listed(&m), id) }
	at("change")
	if m = drive(m, press(tea.KeySpace)); m.panel(0).Field("change").Placement != grid.Hidden ||
		m.cols.Listed(&m)[m.cols.At] != "change" {
		t.Fatalf("space on change: %+v, at %q", m.panel(0).Field("change"), m.cols.Listed(&m)[m.cols.At])
	}
	if m = drive(m, press(tea.KeySpace)); m.panel(0).Field("change").Placement != grid.Column {
		t.Fatalf("space on hidden change: %+v", m.panel(0).Field("change"))
	}
	at("age")
	for _, want := range []string{"▲", "▼", ""} {
		if m = drive(m, typed("s")...); m.panel(0).Mark("age") != want {
			t.Fatalf("s on age: %q, want %q", m.panel(0).Mark("age"), want)
		}
	}
	at("source")
	if m = drive(m, typed("g")...); !slices.Equal(m.panel(0).Groups(), []string{"source"}) || m.cols.Listed(&m)[0] != "source" ||
		m.cols.At != 0 {
		t.Fatalf("g on source: %q, at %d", m.panel(0).Groups(), m.cols.At)
	}
	// shift up moves assignees a place, past status, then past id, the first column, into the groups; id, which cannot
	// group, stays
	at("assignees")
	m = drive(m, testkit.Press("shift+up"), testkit.Press("shift+up"), testkit.Press("shift+up"))
	if groups := m.panel(0).Groups(); !slices.Equal(groups, []string{"source", "assignees"}) || m.cols.At != 1 {
		t.Errorf("shift+up into the groups: %q, at %d", groups, m.cols.At)
	}
	// a group's cell is the grouping glyph and its depth as a subscript, as a sort's mark carries its place
	for id, want := range map[string]string{"source": grid.Grouping + "₁", "assignees": grid.Grouping + "₂"} {
		if got := strings.TrimSpace(ansi.Strip(m.cols.Line(&m, id))[len("☑ ")+table.LabelCells+table.TypeCells+table.SortCells:]); got != want {
			t.Errorf("%s's group cell: %q, want %q", id, got, want)
		}
	}
	m = drive(m, typed("g")...)
	at("id")
	if m = drive(m, testkit.Press("shift+up")); len(m.panel(0).Groups()) != 1 {
		t.Errorf("shift+up on id: %q", m.panel(0).Groups())
	}
	// clicks on a line's box, sort, and group
	x, y := columnsAt(t, m, "title")
	if m = drive(m, testkit.Click(x, y)); m.panel(0).Field("title").Placement != grid.Hidden {
		t.Errorf("a click on title's box: %+v", m.panel(0).Field("title"))
	}
	x, y = columnsAt(t, m, "age")
	if m = drive(m, testkit.Click(x+2+table.LabelCells+table.TypeCells, y)); !strings.HasPrefix(m.panel(0).Mark("age"), "▲") {
		t.Errorf("a click on age's sort: %q", m.panel(0).Mark("age"))
	}
	x, y = columnsAt(t, m, "assignees")
	if m = drive(m, testkit.Click(x+2+table.LabelCells+table.TypeCells+table.SortCells, y)); !slices.Contains(m.panel(0).Groups(), "assignees") {
		t.Errorf("a click on assignees' group: %q", m.panel(0).Groups())
	}
	x, y = columnsAt(t, m, "assignees")
	if m = drive(m, testkit.Click(x+4, y)); m.cols.Listed(&m)[m.cols.At] != "assignees" || !m.cols.Open || !m.cols.InCard {
		t.Errorf("a click on assignees' label: %+v", m.cols)
	}
	m = drive(m, esc)
	_, top, _ := m.cols.Box(&m)
	if m = drive(m, testkit.Click(x, top+1)); !m.cols.Open {
		t.Error("a click on the headings closed the dialog")
	}
	// enter gives the field's card the keyboard; esc gives it back, then closes the dialog
	if m = drive(m, press(tea.KeyEnter)); !m.cols.InCard || m.cols.Card(&m).Field != "assignees" || !m.cols.Open {
		t.Fatalf("enter: %+v", m.cols)
	}
	if m = drive(m, esc, esc); m.cols.InCard || m.cols.Open {
		t.Errorf("esc twice: in the card %v, dialog %v", m.cols.InCard, m.cols.Open)
	}
	m.cols = table.Columns{Open: true}
	if m = drive(m, testkit.Click(1, m.rowsTop()+3)); m.cols.Open {
		t.Error("a click off the dialog left it open")
	}
	// up stops at the first line, down at the last; s on a field that cannot sort does nothing
	m.cols = table.Columns{Open: true}
	m = drive(m, press(tea.KeyUp))
	for range 20 {
		m = drive(m, press(tea.KeyDown))
	}
	if m.cols.At != len(m.cols.Listed(&m))-1 {
		t.Errorf("down to the last: %d", m.cols.At)
	}
	// builds' commit does not sort: its sort cell is blank, and s changes nothing
	m = drive(m, esc)
	m = drive(m, tabsTo("builds")...)
	m.cols = table.Columns{Open: true}
	at("commit")
	if m.stages[m.stage].name != "builds" || strings.TrimSpace(ansi.Strip(m.cols.Line(&m, "commit"))[len("☑ ")+table.LabelCells+table.TypeCells:]) != "" {
		t.Errorf("commit's line on %s: %q", m.stages[m.stage].name, ansi.Strip(m.cols.Line(&m, "commit")))
	}
	if m = drive(m, typed("s")...); len(m.panel(m.stage).SortKeys()) != 0 {
		t.Errorf("s on commit sorted: %+v", m.panel(m.stage).SortKeys())
	}
}
