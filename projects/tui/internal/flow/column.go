// The column menu: a click on a column's name in the header drops it under the name, as does a right click where the
// terminal sends one. Many terminals keep option, ctrl, and the right button with a click for their own selection and
// menu, so the menu holds what those would do: the column's sort, a radio that shows it, None, Ascending, or
// Descending, a column that did not sort coming after the keys there are, one that did keeping its place, and Clear
// Sort, which no column sorts after; the rows grouped by it too, innermost, or no longer; the column hidden; and the
// field's popover, which leaves the sort to the menu. The menu's title is the column's name, so no line names it again. Shift with left or
// right moves the column a place among those the header shows, the menu with it.
package flow

import (
	"slices"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/empowerite/kit/projects/tui/internal/grid"
	"github.com/empowerite/kit/projects/tui/internal/shell"
	"github.com/empowerite/kit/projects/tui/internal/style"
)

// columnMenu is the column menu while it is open: the column it acts on, and the line under the keyboard.
type columnMenu struct {
	open   bool
	column string
	at     int
}

// colAct is what a line of the column menu does.
type colAct int

// What the column menu's lines do: none, the rules; the radio of the column's sort, None, Ascending and Descending;
// every column's sort cleared; the rows grouped by the column, innermost, or no longer; the column hidden; and More,
// the field's popover.
const (
	actRule colAct = iota
	actNone
	actUp
	actDown
	actClear
	actGroup
	actUngroup
	actHide
	actMore
)

// The words of the column menu's lines, and of the heading menu's ungrouping, and their glyphs: grouping's is the mark
// a column grouping the rows has, hiding's the box of a hidden column in the columns dialog.
const (
	ClearSort = "Clear Sort"
	GroupIt   = "Group"
	UngroupIt = "Ungroup"
	Hide      = "Hide"
	More      = "Options…"

	GroupGlyph   = grid.Grouping
	UngroupGlyph = "▢"
	HideGlyph    = grid.Unticked
	MoreGlyph    = "⋯"
)

// colLine is a line of the column menu: what it draws, and what it does.
type colLine struct {
	item shell.Item
	act  colAct
}

// columnLines gives the column menu's lines. The radio's choice picked is the column's sort, None where it does not
// sort. Where something else orders the rows, a group or another key, its triangles carry ₙ, as the header's mark
// carries the place the column takes. Clear Sort, as the tab menu's Clear Selection, sweeps. A line that cannot act on
// the column stays, dim.
func (m Model) columnLines() []colLine {
	p, column := m.panel(m.stage), m.colMenu.column
	d, n := p.Field(column).Sort, ""
	if place, total := p.Place(column); total-min(place, 1) > 0 {
		n = "ₙ"
	}
	radio := func(picked bool, name string, act colAct) colLine {
		if picked {
			return colLine{shell.Item{Glyph: style.Tint(grid.Picked), Name: name}, act}
		}
		return colLine{shell.Item{Glyph: style.Off(grid.Unpicked), Name: name}, act}
	}
	return []colLine{radio(d == grid.Unsorted, "None", actNone), radio(d == grid.Ascending, "▲"+n+" Ascending", actUp),
		radio(d == grid.Descending, "▼"+n+" Descending", actDown), colLine{shell.Item{Glyph: Sweep, Name: ClearSort}, actClear},
		colLine{act: actRule},
		colLine{shell.Item{Glyph: GroupGlyph, Name: GroupIt}, actGroup},
		colLine{shell.Item{Glyph: UngroupGlyph, Name: UngroupIt}, actUngroup}, colLine{act: actRule},
		colLine{shell.Item{Glyph: HideGlyph, Name: Hide}, actHide}, colLine{shell.Item{Glyph: MoreGlyph, Name: More}, actMore}}
}

// columnItems gives the column menu's lines as the menu draws them.
func (m Model) columnItems() []shell.Item {
	var items []shell.Item
	for _, l := range m.columnLines() {
		items = append(items, l.item)
	}
	return items
}

// columnLine gives the index of the column menu's line that does act, -1 where none does.
func (m Model) columnLine(act colAct) int {
	return slices.IndexFunc(m.columnLines(), func(l colLine) bool { return l.act == act })
}

// columnEnabled says whether the column menu's line k acts: the radio on a column that sorts; Clear Sort while a column
// sorts the rows; Group on one that can group and does not yet, Ungroup on one that does; and Hide and Options…
// always; the rules never do.
func (m Model) columnEnabled(k int) bool {
	lines := m.columnLines()
	if k < 0 || k >= len(lines) {
		return false
	}
	p, column := m.panel(m.stage), m.colMenu.column
	grouped := slices.Contains(p.Groups(), column)
	switch lines[k].act {
	case actNone, actUp, actDown:
		return p.Sortable[column]
	case actClear:
		return len(p.ColumnKeys()) > 0
	case actGroup:
		return p.Groupable[column] && !grouped
	case actUngroup:
		return grouped
	case actHide, actMore:
		return true
	}
	return false
}

// shifted gives the panel with a column moved a place, left for a negative step, right for a positive one, among those
// it stands with: a column among the columns the header names, past those no row fills and those the rows group by on
// their own; a group among the groups, a
// depth outward or inward. It says whether it moved; at either end it does not, nor does it cross into the groups or the
// hidden.
func (m Model) shifted(column string, d int) (grid.Panel, bool) {
	p := m.panel(m.stage)
	placement := p.Field(column).Placement
	order := func(q grid.Panel) []string { return q.Groups() }
	switch placement {
	case grid.Column:
		// those the header names: not one the rows group by on their own, whose value their heading says
		shown := slices.DeleteFunc(slices.Clone(m.shownColumns(m.stage)), func(id string) bool {
			_, name := m.columnName(id)
			return name == ""
		})
		order = func(q grid.Panel) []string {
			return slices.DeleteFunc(q.Listed(), func(id string) bool {
				return q.Field(id).Placement != grid.Column || !slices.Contains(shown, id)
			})
		}
	case grid.Hidden:
		return p, false
	}
	before := order(p)
	for q := p; ; {
		var ok bool
		if q, ok = q.Step(column, d); !ok || q.Field(column).Placement != placement {
			return p, false
		}
		if !slices.Equal(order(q), before) {
			return q, true
		}
	}
}

// shiftStep gives the step shift with left or right asks for, and whether the key is one.
func shiftStep(key string) (int, bool) {
	switch key {
	case "shift+left":
		return -1, true
	case "shift+right":
		return 1, true
	}
	return 0, false
}

// shiftKeys gives the binding of shift with left and right on a menu that moves a column, each where it moves it.
func (m Model) shiftKeys(column string) []shell.Binding {
	var presses []string
	for _, key := range []string{"shift+left", "shift+right"} {
		if d, _ := shiftStep(key); func() bool { _, ok := m.shifted(column, d); return ok }() {
			presses = append(presses, key)
		}
	}
	if len(presses) == 0 {
		return nil
	}
	return []shell.Binding{shell.Bind("⇧←→", "move", presses...)}
}

// openColumnMenu drops the column menu down from a column's name, the first line that acts under the keyboard.
func (m *Model) openColumnMenu(column string) {
	m.colMenu = columnMenu{open: true, column: column, at: -1}
	m.colMenu.at = shell.Move(m.columnItems(), m.columnEnabled, -1, 1)
}

// columnKey takes a key while the column menu is open: up and down move, a letter to the next line it starts, enter
// and space act, shift with left or right moves the column, and esc closes it.
func (m Model) columnKey(k tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	items := m.columnItems()
	switch k.String() {
	case "up":
		m.colMenu.at = shell.Move(items, m.columnEnabled, m.colMenu.at, -1)
	case "down":
		m.colMenu.at = shell.Move(items, m.columnEnabled, m.colMenu.at, 1)
	case "enter", "space":
		cmd := m.columnAct(m.colMenu.at) // before m is returned, which a return would copy first
		return m, cmd
	case "esc":
		m.colMenu = columnMenu{}
	case "shift+left", "shift+right":
		// the column moves, the menu with it, the keyboard on the same line
		d, _ := shiftStep(k.String())
		if p, ok := m.shifted(m.colMenu.column, d); ok {
			m.setPanel(p)
		}
	default:
		m.colMenu.at = shell.NextItem(items, m.columnEnabled, m.colMenu.at, k.Text)
	}
	return m, nil
}

// columnAct closes the column menu and does what its line k says, when it acts.
func (m *Model) columnAct(k int) tea.Cmd {
	c, enabled := m.colMenu, m.columnEnabled(k)
	if !enabled {
		m.colMenu = columnMenu{}
		return nil
	}
	act := m.columnLines()[k].act
	m.colMenu = columnMenu{}
	p := m.panel(m.stage)
	switch act {
	case actNone, actUp, actDown:
		// a column that did not sort comes after the keys there are; one that did turns in its place
		p = p.SortedAs(c.column, map[colAct]grid.Direction{actNone: grid.Unsorted, actUp: grid.Ascending,
			actDown: grid.Descending}[act])
	case actClear:
		p = p.Unsorted()
	case actHide:
		p = p.Placed(c.column, grid.Hidden)
	case actGroup, actUngroup:
		// a column joins the groups innermost, or leaves them from wherever it stands
		p = p.Grouped(c.column, true)
	default:
		x, _ := m.columnName(c.column)
		return m.openPopover(c.column, x, paneTop, true)
	}
	m.setPanel(p)
	return nil
}

// columnBox gives the column menu's lines, the box around the column's name, which its top edge draws where the header
// does, as the app's menu hangs from ≡; and the screen line and column it is drawn from.
func (m Model) columnBox() (lines []string, top, left int) {
	x, name := m.columnName(m.colMenu.column)
	lines, left = shell.Box(m.columnItems(), m.columnEnabled, m.colMenu.at, name, x, m.width)
	return lines, paneTop, left
}

// columnName gives the screen column a column's name starts on in the header, and its first line, as the header draws
// it.
func (m Model) columnName(column string) (int, string) {
	j := slices.Index(m.shownColumns(m.stage), column)
	x, name := grid.HeaderName(m.rows(m.stage), m.names(m.stage), m.gridWidth(m.stage), j)
	return x + m.indent(m.stage), name
}

// columnClick takes a click while the column menu is open: on a line that acts, it does what the line says; anywhere
// else, on the name it hangs from too, it closes the menu.
func (m *Model) columnClick(x, y int) tea.Cmd {
	box, top, left := m.columnBox()
	if k := y - top - 1; x > left && x < left+ansi.StringWidth(box[0])-1 && k >= 0 && k < len(box)-2 {
		return m.columnAct(k)
	}
	m.colMenu = columnMenu{}
	return nil
}

// headerColumn gives the column whose name a click at x, y in the header falls on, none off every name.
func (m Model) headerColumn(x, y int) string {
	lines, _ := m.header()
	if y < paneTop || y >= paneTop+len(lines)-1 {
		return ""
	}
	j := grid.HeaderAt(m.rows(m.stage), m.names(m.stage), m.gridWidth(m.stage), x-m.indent(m.stage))
	if j < 0 {
		return ""
	}
	return m.shownColumns(m.stage)[j]
}

// sortAt gives the column whose sort a click at x, y on the header turns, one that can sort: on its sort mark, as onMark
// says; or, on the names' first line, on the cell right after its name, or on the cell after that, unless that one
// stands against the next column's name. Where a column does not sort yet, those cells are blank, and still sort it.
// Nothing elsewhere.
func (m Model) sortAt(x, y int) string {
	rows, names, width, in := m.rows(m.stage), m.names(m.stage), m.gridWidth(m.stage), m.indent(m.stage)
	shown, p := m.shownColumns(m.stage), m.panel(m.stage)
	if m.onMark(x, y) {
		if j := grid.HeaderAt(rows, names, width, x-in); j >= 0 && p.Sortable[shown[j]] {
			return shown[j]
		}
	}
	if y != paneTop {
		return ""
	}
	starts, ends := map[int]bool{}, map[int]string{}
	for j, column := range shown {
		start, first := grid.HeaderName(rows, names, width, j)
		if start < 0 {
			continue
		}
		starts[in+start] = true
		if p.Sortable[column] {
			ends[in+start+ansi.StringWidth(strings.TrimSuffix(first, " "+p.Mark(column)))] = column
		}
	}
	if column, ok := ends[x]; ok {
		return column
	}
	if column, ok := ends[x-1]; ok && !starts[x+1] {
		return column
	}
	return ""
}

// onMark says whether a click at x, y on a line of the header's names falls on a column's sort mark, its ▲ or ▼ and
// the number after it, or the cell before the triangle.
func (m Model) onMark(x, y int) bool {
	lines, _ := m.header()
	line := lines[y-paneTop]
	cell := func(at int) string { return ansi.Strip(ansi.Cut(line, at, at+1)) }
	triangle := func(c string) bool { return c == "▲" || c == "▼" }
	subscript := func(c string) bool { return c != "" && strings.Trim(c, "₀₁₂₃₄₅₆₇₈₉") == "" }
	if triangle(cell(x)) || triangle(cell(x+1)) && cell(x) == " " {
		return true
	}
	for at := x; subscript(cell(at)); at-- {
		if triangle(cell(at - 1)) {
			return true
		}
	}
	return false
}
