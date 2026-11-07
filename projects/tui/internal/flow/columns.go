// The columns dialog: Fields… in the presets' menu opens it under the header's right end, titled by the stage's
// fields. It lists every field of the stage, those the rows group by first, the outermost first, then the columns
// as the rows stand them, then the hidden, each a line: whether it shows, its label, its sort, ▲ or ▼ and its place
// among the keys, and its group's depth. What it sets applies at once. Up and down move between the lines; shift with
// either moves the field a place, across into the groups or out of them, or into the hidden; space shows or hides it,
// s turns its sort, none, ascending, descending, none again, a key after the others, and g groups by it, within the
// groups there are, or no longer; enter opens its Options…, and esc closes the dialog. A click on a line's box, its
// sort, or its group does what space, s, or g does; one off the dialog closes it.
package flow

import (
	"cmp"
	"slices"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/empowerite/kit/projects/tui/internal/grid"
	"github.com/empowerite/kit/projects/tui/internal/pane"
	"github.com/empowerite/kit/projects/tui/internal/shell"
	"github.com/empowerite/kit/projects/tui/internal/style"
)

// Fields is the glyph of Fields…, the presets' menu's line that opens the columns dialog: a card index.
const Fields = "📇"

// columnsDialog is the columns dialog while it is open, and the line under the keyboard, an index into the fields it
// lists.
type columnsDialog struct {
	open bool
	at   int
}

// The cells of a line of the columns dialog, after its box and a space: the label, then the sort, then the group.
const (
	labelCells = 16
	sortCells  = 6
)

// colsListed gives the fields the columns dialog lists, as the selected stage's panel lists them.
func (m Model) colsListed() []string { return m.panel(m.stage).Listed() }

// openColumns opens the columns dialog, the keyboard on its first line.
func (m *Model) openColumns() {
	m.cols = columnsDialog{open: true}
}

// columnsLine draws a field's line in the columns dialog: its box, ticked where it shows, as a column or a group; its
// label, dim where it is hidden, ⊘ after it while a filter on it holds rows back; its sort, ▲ or ▼, and its place among the keys while there are several, or a dim dot;
// and its group's depth, or a dim dot, or nothing where it cannot group.
func (m Model) columnsLine(id string) string {
	p := m.panel(m.stage)
	f := p.Field(id)
	box, label := style.Mark(grid.Ticked), cmp.Or(p.Labels[id], id)
	if f.Placement == grid.Hidden {
		box, label = style.Off(grid.Unticked), style.Off(label)
	}
	if len(f.Filter.Values) > 0 {
		label += " " + style.Tint(grid.Filtered)
	}
	sort := style.Off("·")
	if mark := p.Mark(id); mark != "" {
		sort = style.Tint(mark)
	} else if !p.Sortable[id] {
		sort = " "
	}
	group := style.Off("·")
	switch {
	case f.Placement == grid.Group:
		group = style.Tint(grid.GroupMark(f.Depth))
	case !p.Groupable[id]:
		group = " "
	}
	return box + " " + pane.Pad(label, labelCells, false) + pane.Pad(sort, sortCells, false) + group
}

// columnsBox gives the columns dialog's lines, titled by the stage's fields, as it lists the groups and the hidden
// too, the headings of its cells dim over its fields, a rule under the groups; the line under the keyboard on the
// menu's color; and the screen line and column it is drawn from: under the header's right end, its right edge at the
// pane's.
func (m Model) columnsBox() (lines []string, top, left int) {
	p := m.panel(m.stage)
	body := []string{style.Off("  " + pane.Pad("field", labelCells, false) + pane.Pad("sort", sortCells, false) + "group")}
	groups := len(p.Groups())
	for k, id := range m.colsListed() {
		if k == groups && groups > 0 {
			body = append(body, style.Off(strings.Repeat("─", 2+labelCells+sortCells+5)))
		}
		line := m.columnsLine(id)
		if k == m.cols.at {
			line = style.Selected(pane.Pad(ansi.Strip(line), 2+labelCells+sortCells+5, false))
		}
		body = append(body, line)
	}
	lines = shell.Dialog(m.stages[m.stage].name+" fields", body, m.paneWidth())
	return lines, paneTop, max(0, m.paneWidth()-ansi.StringWidth(lines[0]))
}

// columnsRow gives the field a screen line of the columns dialog draws, as an index into the fields it lists, or -1 for
// its edges, its headings, and its rule.
func (m Model) columnsRow(y int) int {
	_, top, _ := m.columnsBox()
	k := y - top - 2 // the top edge and the headings
	if groups := len(m.panel(m.stage).Groups()); groups > 0 && k >= groups {
		if k == groups {
			return -1
		}
		k--
	}
	if k < 0 || k >= len(m.colsListed()) {
		return -1
	}
	return k
}

// columnsAct does to the field under the keyboard what a key asks: space shows it or hides it; s turns its sort; g
// groups by it, or no longer; shift with up or down moves it a place, the keyboard with it.
func (m *Model) columnsAct(key string) {
	id, p := m.colsListed()[m.cols.at], m.panel(m.stage)
	switch key {
	case "space":
		if p.Field(id).Placement == grid.Hidden {
			p = p.Placed(id, grid.Column)
		} else {
			p = p.Placed(id, grid.Hidden)
		}
	case "s":
		if !p.Sortable[id] {
			return
		}
		p = p.SortedAs(id, map[grid.Direction]grid.Direction{grid.Unsorted: grid.Ascending, grid.Ascending: grid.Descending,
			grid.Descending: grid.Unsorted}[p.Field(id).Sort])
	case "g":
		p = p.Grouped(id, true)
	case "shift+up", "shift+down":
		step := 1
		if key == "shift+up" {
			step = -1
		}
		p, _ = p.Step(id, step)
	}
	m.setPanel(p)
	// the keyboard stays on the field, wherever it stands now
	if k := slices.Index(m.colsListed(), id); k >= 0 {
		m.cols.at = k
	}
}

// columnsKey takes a key while the columns dialog is open: up and down move between its lines; space, s, g, and shift
// with up or down act on the field under the keyboard, as columnsAct says; enter opens the field's Options…, over the
// dialog; and esc closes it.
func (m Model) columnsKey(k tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch key := k.String(); key {
	case "up":
		m.cols.at = max(m.cols.at-1, 0)
	case "down":
		m.cols.at = min(m.cols.at+1, len(m.colsListed())-1)
	case "esc":
		m.cols = columnsDialog{}
	case "enter":
		_, top, left := m.columnsBox()
		return m, m.openPopover(m.colsListed()[m.cols.at], left, top, false)
	case "space", "s", "g", "shift+up", "shift+down":
		m.columnsAct(key)
	}
	return m, nil
}

// columnsClick takes a click while the columns dialog is open: on a field's box, its sort, or its group, it acts as
// space, s, or g does, the keyboard on the field; on its label, the keyboard goes there; on the dialog elsewhere,
// nothing; off it, the dialog closes.
func (m *Model) columnsClick(x, y int) {
	box, top, left := m.columnsBox()
	if x < left || x >= left+ansi.StringWidth(box[0]) || y < top || y >= top+len(box) {
		m.cols = columnsDialog{}
		return
	}
	k := m.columnsRow(y)
	if k < 0 {
		return
	}
	m.cols.at = k
	switch cell := x - left - 2; {
	case cell < 2:
		m.columnsAct("space")
	case cell >= 2+labelCells+sortCells:
		m.columnsAct("g")
	case cell >= 2+labelCells:
		m.columnsAct("s")
	}
}

// columnsKeys gives the keys bar's segment while the columns dialog is open, each key where it acts on the field under
// the keyboard: up and down where there is a line that way, shift with either where the field moves that way, s where
// it sorts, g where it groups, and enter where it has Options….
func (m Model) columnsKeys() shell.Segment {
	s := shell.Segment{Owns: true, Bindings: []shell.Binding{shell.Bind("esc", "close")}}
	id, p := m.colsListed()[m.cols.at], m.panel(m.stage)
	var moves, orders []string
	for _, step := range []struct {
		key  string
		by   int
		more bool
	}{{"up", -1, m.cols.at > 0}, {"down", 1, m.cols.at < len(m.colsListed())-1}} {
		if step.more {
			moves = append(moves, step.key)
		}
		if _, ok := p.Step(id, step.by); ok {
			orders = append(orders, "shift+"+step.key)
		}
	}
	if len(moves) > 0 {
		s.Bindings = append(s.Bindings, shell.Bind("↑↓", "move", moves...))
	}
	if len(orders) > 0 {
		s.Bindings = append(s.Bindings, shell.Bind("⇧↑↓", "order", orders...))
	}
	s.Bindings = append(s.Bindings, shell.Bind("␣", "show/hide", "space"))
	if p.Sortable[id] {
		s.Bindings = append(s.Bindings, shell.Bind("s", "sort"))
	}
	if p.Groupable[id] {
		s.Bindings = append(s.Bindings, shell.Bind("g", "group"))
	}
	if len(m.popLines(id)) > 0 {
		s.Bindings = append(s.Bindings, shell.Bind("⏎", "options", "enter"))
	}
	return s
}
