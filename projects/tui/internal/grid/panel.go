// The panel's controls, and the rows they let through.
package grid

import (
	"maps"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/empowerite/kit/projects/tui/internal/style"
)

// Show is which rows a panel lets through by whom they belong to: every one, the caller's, or everyone else's.
type Show int

// The choices of Show, in the panel's order.
const (
	All Show = iota
	Mine
	Others
)

// shows names each choice of Show, as the panel draws it.
var shows = []string{"all", "mine", "others"}

// The glyphs the panel draws: the search, a checkbox off and on, and a radio button off and on. Each is one cell wide
// in every terminal, as Unicode fixes its width, but the search, which is two.
const (
	Search   = "🔎"
	Unticked = "☐"
	Ticked   = "☑"
	Unpicked = "⦾"
	Picked   = "⦿"
)

// Panel is the controls beside a grid's rows, top to bottom: the search field; a checkbox per column that has a name;
// and, when the rows belong to people, a radio button per choice of Show. Mine and others need the caller's login,
// Me; while it is unknown they are dim, and cannot be chosen. Cursor is the control under the keyboard, Focused
// whether the panel has it. Hidden holds the names of the columns hidden; the panel never changes the map it was
// given, but gives a new one.
type Panel struct {
	Query   string
	Columns []string
	Hidden  map[string]bool
	Show    Show
	Owned   bool
	Me      string
	Cursor  int
	Focused bool
}

// control is one of the panel's controls: the search, a column's checkbox, or a choice of Show.
type control struct {
	search bool
	column string
	show   Show
}

// controls gives the panel's controls, in order.
func (p Panel) controls() []control {
	out := []control{{search: true}}
	for _, c := range p.Columns {
		if c != "" {
			out = append(out, control{column: c})
		}
	}
	if p.Owned {
		for s := range shows {
			out = append(out, control{show: Show(s)})
		}
	}
	return out
}

// Lines draws the panel's controls, one a line, the headings dim, the control under the keyboard inverse while the
// panel has it; what is selected, a column ticked and the choice of Show picked, is bold in the accent color, the
// others' boxes and buttons dim, and all of a choice that cannot be picked. Beside each line is the control it draws, or -1 for a heading or a gap.
func (p Panel) Lines() (lines []string, at []int) {
	add := func(line string, control int) {
		lines, at = append(lines, line), append(at, control)
	}
	for k, c := range p.controls() {
		var line string
		switch {
		case c.search:
			line = Search + " " + p.Query
			if p.Focused && p.Cursor == k {
				line += "▏"
			} else if p.Query == "" {
				line = Search + " " + style.Off("search")
			}
		case c.column != "":
			if k == 1 {
				add("", -1)
				add(style.Off("columns"), -1)
			}
			line = style.Mark(Ticked + " " + c.column)
			if p.Hidden[c.column] {
				line = style.Off(Unticked) + " " + c.column
			}
		default:
			if c.show == All {
				add("", -1)
				add(style.Off("show"), -1)
			}
			switch {
			case p.Show == c.show:
				line = style.Mark(Picked + " " + shows[c.show])
			case c.show != All && p.Me == "":
				line = style.Off(Unpicked + " " + shows[c.show])
			default:
				line = style.Off(Unpicked) + " " + shows[c.show]
			}
		}
		if p.Focused && p.Cursor == k && !c.search {
			line = style.Cursor(ansi.Strip(line))
		}
		add(line, k)
	}
	return lines, at
}

// Update takes a key while the panel has the keyboard, and says whether it changed anything: up and down move between
// the controls, stopping at the ends; on the search, a printable key is typed, / too, and backspace takes one back;
// off it, / goes to the search, and space and enter toggle a checkbox and pick a choice of Show.
func (p Panel) Update(k tea.KeyPressMsg) (Panel, bool) {
	controls := p.controls()
	p.Cursor = min(max(p.Cursor, 0), len(controls)-1)
	c := controls[p.Cursor]
	switch key := k.String(); {
	case key == "up":
		p.Cursor = max(p.Cursor-1, 0)
	case key == "down":
		p.Cursor = min(p.Cursor+1, len(controls)-1)
	case c.search && key == "backspace":
		if r := []rune(p.Query); len(r) > 0 {
			p.Query = string(r[:len(r)-1])
		}
	case c.search && k.Text != "":
		p.Query += k.Text
	case k.Text == "/":
		p.Cursor = 0
	case !c.search && (key == "space" || key == "enter"):
		p = p.act(c)
	default:
		return p, false
	}
	return p, true
}

// Click takes a click on a line of the panel, as Lines numbers them, and says whether it fell on a control: the
// control is then under the keyboard, and a checkbox or a choice is acted on as space does.
func (p Panel) Click(line int) (Panel, bool) {
	_, at := p.Lines()
	if line < 0 || line >= len(at) || at[line] < 0 {
		return p, false
	}
	p.Cursor = at[line]
	if c := p.controls()[p.Cursor]; !c.search {
		p = p.act(c)
	}
	return p, true
}

// Find puts the search under the keyboard.
func (p Panel) Find() Panel {
	p.Cursor, p.Focused = 0, true
	return p
}

// act toggles a column's checkbox, or picks a choice of Show; mine and others only while the caller's login is known.
func (p Panel) act(c control) Panel {
	if c.column != "" {
		hidden := maps.Clone(p.Hidden)
		if hidden == nil {
			hidden = map[string]bool{}
		}
		hidden[c.column] = !hidden[c.column]
		p.Hidden = hidden
		return p
	}
	if c.show == All || p.Me != "" {
		p.Show = c.show
	}
	return p
}

// Width gives the cells the panel's widest line takes.
func (p Panel) Width() int {
	lines, _ := p.Lines()
	widest := 0
	for _, l := range lines {
		widest = max(widest, ansi.StringWidth(l))
	}
	return widest
}

// Filter gives the rows the panel lets through: those whose text has the search, without regard to case, and that
// belong to the caller or not, as Show says, each without the hidden columns' cells. columns names each cell's column,
// in order. While the search or Show narrows the rows, an aside that
// heads none of them is left out with them.
func (p Panel) Filter(rows []Row, columns []string) []Row {
	query := strings.ToLower(p.Query)
	var out []Row
	for _, r := range rows {
		switch {
		case r.Aside:
			out = append(out, r)
		case p.matches(r, query):
			out = append(out, p.hide(r, columns))
		}
	}
	if query == "" && p.Show == All {
		return out
	}
	return dropBare(out)
}

// dropBare leaves out each aside that heads no row: one the next aside, or the end, follows at once.
func dropBare(rows []Row) []Row {
	var out []Row
	for i, r := range rows {
		if r.Aside && (i+1 == len(rows) || rows[i+1].Aside) {
			continue
		}
		out = append(out, r)
	}
	return out
}

// matches says whether a row has the search in its text, and belongs as Show says.
func (p Panel) matches(r Row, query string) bool {
	switch {
	case p.Show == Mine && !owned(r, p.Me):
		return false
	case p.Show == Others && owned(r, p.Me):
		return false
	case query == "":
		return true
	}
	for _, line := range append([][]Cell{r.Cells}, r.Detail...) {
		for _, c := range line {
			for _, v := range c.Values {
				if strings.Contains(strings.ToLower(ansi.Strip(v)), query) {
					return true
				}
			}
		}
	}
	return false
}

// owned says whether the row belongs to the login.
func owned(r Row, login string) bool {
	for _, o := range r.Owners {
		if o == login {
			return true
		}
	}
	return false
}

// hide gives the row without the hidden columns' cells; a row of one cell spans the columns, and keeps it.
func (p Panel) hide(r Row, columns []string) Row {
	if len(r.Cells) < 2 || len(p.Hidden) == 0 {
		return r
	}
	var cells []Cell
	for j, c := range r.Cells {
		if j >= len(columns) || !p.Hidden[columns[j]] {
			cells = append(cells, c)
		}
	}
	r.Cells = cells
	return r
}
