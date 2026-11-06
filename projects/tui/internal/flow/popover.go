// A field's popover: a small box over the screen that sets, for one field of the selected stage, how its rows sort,
// which of its values they must hold, and how it is drawn. Enter on the field's line in the sidebar's editor opens it,
// and so does a double click on its name in the header. It has the keyboard while it is open: up and down move between
// its lines, space or enter acts on the one under the keyboard, and esc closes it; a click on a line acts on it, and
// a click off the box closes it. What it sets applies at once, so the rows, the editor, and the preset line's mark and
// chips follow it.
package flow

import (
	"fmt"
	"slices"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/empowerite/kit/projects/tui/internal/grid"
	"github.com/empowerite/kit/projects/tui/internal/pane"
	"github.com/empowerite/kit/projects/tui/internal/shell"
	"github.com/empowerite/kit/projects/tui/internal/style"
)

// popover is a field's popover while it is open: the field, by id; its line under the keyboard; and the screen cell
// it hangs from, the field's line in the sidebar, whose right edge it keeps, or its name in the header, under which it
// opens.
type popover struct {
	open   bool
	field  string
	at     int
	x, y   int
	header bool
}

// popLine is one line of a popover: what it draws, and what space does on it, nil for a heading or a gap. Picked says
// a choice is the one taken, which space on it again leaves as it is; a value's box toggles, picked or not.
type popLine struct {
	text   string
	act    func(grid.Panel) grid.Panel
	picked bool
	box    bool
}

// popLines gives the popover's lines for the field, under three dim headings, each where the field has it. sort: off,
// ▲ ascending, and ▼ descending, a radio, the heading saying the field's place among the keys while it sorts. filter,
// for an enum or a person: whether the values checked are let through or held back, include or exclude, one line that
// toggles, then a box per value: an enum's known values with their glyphs, a person's logins the rows loaded hold,
// after @me, the caller, and the values the catalog does not know, then none, a row with no value, and any the filter
// names that are not there. rendering: auto, with the rendering it chooses now, dim, then each of the field's
// renderings, a radio.
func (m Model) popLines(id string) []popLine {
	s, p := m.stages[m.stage], m.panel(m.stage)
	fields := fieldsOf(s.name)
	k := slices.IndexFunc(fields, func(f field) bool { return f.ID == id })
	if k < 0 {
		return nil
	}
	f, g := fields[k], p.Field(id)
	var out []popLine
	heading := func(text string) { out = append(out, popLine{text: style.Off(text)}) }
	radio := func(text string, picked bool, act func(grid.Panel) grid.Panel) {
		glyph := style.Off(grid.Unpicked)
		if picked {
			glyph = style.Tint(grid.Picked)
		}
		out = append(out, popLine{text: glyph + " " + text, act: act, picked: picked})
	}
	if f.Sort {
		title := "sort"
		if keys := p.SortKeys(); g.Sort != grid.Unsorted {
			title += fmt.Sprintf(", key %d of %d", slices.IndexFunc(keys, func(s grid.SortKey) bool { return s.Column == id })+1,
				len(keys))
		}
		heading(title)
		for _, d := range []struct {
			dir  grid.Direction
			name string
		}{{grid.Unsorted, "off"}, {grid.Ascending, "▲ ascending"}, {grid.Descending, "▼ descending"}} {
			radio(d.name, g.Sort == d.dir, func(p grid.Panel) grid.Panel { return p.SortedAs(id, d.dir) })
		}
	}
	if f.Filter && types[f.Type].filters {
		heading("filter")
		mode := style.Tint(grid.Picked) + " include  " + style.Off(grid.Unpicked) + " exclude"
		if g.Filter.Exclude {
			mode = style.Off(grid.Unpicked) + " include  " + style.Tint(grid.Picked) + " exclude"
		}
		out = append(out, popLine{text: mode, box: true, act: func(p grid.Panel) grid.Panel {
			f := p.Field(id).Filter
			return p.FilteredBy(id, grid.Filter{Values: f.Values, Exclude: !f.Exclude})
		}})
		for _, v := range m.filterValues(f) {
			name := v
			switch {
			case f.Type == "enum" && v != grid.NoValue && f.glyphOf(v) != "":
				name = f.glyphOf(v) + " " + v
			case v == caller && m.me != "":
				name += " " + style.Off(m.me)
			}
			box := style.Off(grid.Unticked)
			if slices.Contains(g.Filter.Values, v) {
				box = style.Tint(grid.Ticked)
			}
			out = append(out, popLine{text: box + " " + name, box: true, act: func(p grid.Panel) grid.Panel {
				filter := p.Field(id).Filter
				if k := slices.Index(filter.Values, v); k >= 0 {
					filter.Values = slices.Delete(slices.Clone(filter.Values), k, k+1)
				} else {
					filter.Values = append(slices.Clone(filter.Values), v)
				}
				return p.FilteredBy(id, filter)
			}})
		}
	}
	heading("rendering")
	for _, r := range append([]string{""}, f.Renderings...) {
		name := r
		if r == "" {
			name = "auto " + style.Off(m.shown(m.stage).drawn[id])
		}
		radio(name, g.Rendering == r, func(p grid.Panel) grid.Panel { return p.Rendered(id, r) })
	}
	return out
}

// filterValues gives the values a field's filter offers, as popLines lists them.
func (m Model) filterValues(f field) []string {
	var out, seen []string
	if f.Type == "person" {
		out = append(out, caller)
	}
	for _, e := range f.Values {
		out = append(out, e.Value)
	}
	for _, r := range m.records(m.stages[m.stage]) {
		for _, v := range r.values[f.ID].Raw {
			if v != "" && !slices.Contains(out, v) && !slices.Contains(seen, v) {
				seen = append(seen, v)
			}
		}
	}
	slices.Sort(seen)
	out = append(append(out, seen...), grid.NoValue)
	for _, v := range m.stages[m.stage].view.Field(f.ID).Filter.Values {
		if !slices.Contains(out, v) {
			out = append(out, v)
		}
	}
	return out
}

// openPopover opens a field's popover, hanging from the screen cell x, y, the first line that acts under the keyboard;
// from the header, it opens under the cell, else beside it. It asks kit for the caller's login the first time, which
// @me names.
func (m *Model) openPopover(id string, x, y int, header bool) tea.Cmd {
	m.pop = popover{open: true, field: id, x: x, y: y, header: header, at: -1}
	m.pop.at = m.popMove(-1, 1)
	return m.askMe()
}

// fieldLineAt gives the screen line the sidebar draws the line of the panel's control under the keyboard on, or the
// pane's first when it draws none.
func (m Model) fieldLineAt() int {
	_, at := m.panel(m.stage).Lines()
	line := slices.Index(at, m.panel(m.stage).Cursor)
	for y := 1; y < m.sideHeight(); y++ {
		if m.panelLine(m.width-1, y) == line {
			return y
		}
	}
	return paneTop
}

// popMove gives the line of the popover from k on, by step, that acts, stopping at the first and the last; k itself
// when none does.
func (m Model) popMove(k, step int) int {
	lines := m.popLines(m.pop.field)
	for j := k + step; j >= 0 && j < len(lines); j += step {
		if lines[j].act != nil {
			return j
		}
	}
	return k
}

// popActs says whether space on the popover's line under the keyboard changes anything: a value's box and the filter's
// mode always do, and a choice not taken.
func (m Model) popActs() bool {
	lines := m.popLines(m.pop.field)
	if m.pop.at < 0 || m.pop.at >= len(lines) {
		return false
	}
	l := lines[m.pop.at]
	return l.act != nil && (l.box || !l.picked)
}

// popKeys gives the popover's keys, on its bottom edge and on the keys bar, which it owns: esc closes it; up and down
// move, where there is a line that acts that way; and space or enter toggles a value's box, or takes a choice not
// taken.
func (m Model) popKeys() shell.Segment {
	s := shell.Segment{Owns: true, Bindings: []shell.Binding{shell.Bind("esc", "close")}}
	var moves []string
	if m.popMove(m.pop.at, -1) != m.pop.at {
		moves = append(moves, "up")
	}
	if m.popMove(m.pop.at, 1) != m.pop.at {
		moves = append(moves, "down")
	}
	if len(moves) > 0 {
		s.Bindings = append(s.Bindings, shell.Bind("↑↓", "move", moves...))
	}
	if m.popActs() {
		does := "choose"
		if m.popLines(m.pop.field)[m.pop.at].box {
			does = "toggle"
		}
		s.Bindings = append(s.Bindings, shell.Bind("␣", does, "space", "enter"))
	}
	return s
}

// popAct acts on the popover's line k: it is then under the keyboard, and the selected stage's panel changes as the
// line says, at once.
func (m *Model) popAct(k int) {
	lines := m.popLines(m.pop.field)
	if k < 0 || k >= len(lines) || lines[k].act == nil {
		return
	}
	m.pop.at = k
	m.setPanel(lines[k].act(m.panel(m.stage)))
}

// popKey takes a key while the popover is open, as the popover's comment says; every other key does nothing.
func (m Model) popKey(k tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch k.String() {
	case "esc":
		m.pop = popover{}
	case "up":
		m.pop.at = m.popMove(m.pop.at, -1)
	case "down":
		m.pop.at = m.popMove(m.pop.at, 1)
	case "space", "enter":
		if m.popActs() {
			m.popAct(m.pop.at)
		}
	}
	return m, nil
}

// popWidest is the widest the popover's keys get, on its bottom edge.
const popWidest = "esc close · ↑↓ move · ␣ choose"

// popBox gives the popover's lines, a dialog titled by the field's label with its keys on the bottom edge, and the
// screen line and column it is drawn from: from the header, under the name clicked, from its column, or nearer the
// screen's left where it would run past the edge; from the sidebar, its right edge at the screen's, its top on the
// field's line, or higher where it would run past the keys bar.
func (m Model) popBox() (box []string, top, left int) {
	var body []string
	for k, l := range m.popLines(m.pop.field) {
		text := l.text
		if k == m.pop.at {
			text = style.Cursor(ansi.Strip(text))
		}
		body = append(body, text)
	}
	var keys []string
	for _, b := range m.popKeys().Bindings {
		keys = append(keys, b.Key+" "+b.Does)
	}
	if len(body) > 0 {
		// as wide as every key would make it, so the box keeps its width as the keys come and go
		body[0] = pane.Pad(body[0], ansi.StringWidth(popWidest)+2, false)
	}
	box = shell.Dialog(labelOf(m.stages[m.stage].name, m.pop.field), body, strings.Join(keys, " · "), m.width)
	w := ansi.StringWidth(box[0])
	if m.pop.header {
		top, left = m.pop.y+1, min(m.pop.x, m.width-w)
	} else {
		top, left = m.pop.y, m.width-w
	}
	return box, max(1, min(top, m.height-1-len(box))), max(0, left)
}

// popClick takes a click while the popover is open: on one of its lines that acts, it acts on it, as space does; on the
// box elsewhere, nothing; off the box, it closes it.
func (m *Model) popClick(x, y int) {
	box, top, left := m.popBox()
	if x < left || x >= left+ansi.StringWidth(box[0]) || y < top || y >= top+len(box) {
		m.pop = popover{}
		return
	}
	if x > left && x < left+ansi.StringWidth(box[0])-1 {
		m.popAct(y - top - 1)
	}
}
