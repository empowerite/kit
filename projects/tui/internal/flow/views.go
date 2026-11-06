// Named views: a stage's panel as it is set, its columns shown and their order, its sort and groups, whose rows it
// shows, and whether its headers stand over the rows, kept under a name. Every stage with columns has its own views,
// the presets the field catalog gives it, which never change: Default, the stage as it opens, and Compact, the few
// columns that say what a row is. The person saves others, per stage, in a file of their own, under ~/.config/kit/, so
// their views follow them to every repository; one they named as a stage's own was named before the stage had it, and
// stays theirs. The panel's dropdown, under its search, names the view; enter, or a click on it, drops down the views
// to choose from; typing renames it. Once the panel differs from the view, the dropdown names none, as the panel is no
// longer the view, and 💾 Save stands at its right, dim until a name is typed; it stands there too once the name was
// changed. ctrl+s or a click on it saves the panel under the name, a view of the person's own overwritten, a stage's own
// refused for a name of their own, and no name refused.
package flow

import (
	"cmp"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/empowerite/kit/projects/tui/internal/grid"
	"github.com/empowerite/kit/projects/tui/internal/shell"
)

// Opening is the name of the view a stage opens on, one of its own.
const Opening = "Default"

// named is a stage's panel as a view keeps it: the columns hidden, their order, the sort's keys, the groups, the
// columns kept from grouping on their own, whose rows it shows, and whether the headers show.
type named struct {
	Name    string    `json:"name"`
	Hidden  []string  `json:"hidden,omitempty"`
	Order   []string  `json:"order,omitempty"`
	Sort    []sortKey `json:"sort,omitempty"`
	Group   []string  `json:"group,omitempty"`
	Kept    []string  `json:"kept,omitempty"`
	Show    string    `json:"show,omitempty"`
	Headers bool      `json:"headers"`
}

// sortKey is one of a view's sort keys: a column, and whether the larger come first.
type sortKey struct {
	Column string `json:"column"`
	Desc   bool   `json:"desc,omitempty"`
}

// viewsFile gives the file the person's views are kept in: kit/views.json under XDG_CONFIG_HOME, else under
// ~/.config; nothing when neither is known.
func viewsFile() string {
	dir := os.Getenv("XDG_CONFIG_HOME")
	if dir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return ""
		}
		dir = filepath.Join(home, ".config")
	}
	return filepath.Join(dir, "kit", "views.json")
}

// readViews gives the views the file keeps, by stage; none when there is no file. A file that does not read as views
// is the error.
func readViews(path string) (map[string][]named, error) {
	out := map[string][]named{}
	if path == "" {
		return out, nil
	}
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return out, nil
	}
	if err == nil {
		err = json.Unmarshal(data, &out)
	}
	return out, err
}

// writeViews keeps the views in the file, written whole beside it first, so a reader never sees half of it.
func writeViews(path string, views map[string][]named) error {
	if path == "" {
		return nil
	}
	data, err := json.MarshalIndent(views, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp := path + ".new"
	if err := os.WriteFile(tmp, append(data, '\n'), 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// builtins gives a stage's own views, its presets in the catalog, in the catalog's order; a stage without columns has
// Default alone, as it opens.
func (m Model) builtins(i int) []named {
	s := catalog[m.stages[i].name]
	if s == nil {
		return []named{{Name: Opening, Headers: !m.noHeaders}}
	}
	var out []named
	for _, p := range s.Presets {
		out = append(out, p.view(s.Fields, !m.noHeaders))
	}
	return out
}

// view gives a preset as a view keeps a stage's panel, of the stage's fields: the columns hidden, those it places at a
// position in their order, the groups by depth, the sort's keys by priority, whose rows it shows, from the filter on the
// field that owns them, and the headers, as the screen opens them where the preset says nothing. A field it does not
// name takes the catalog's placement. A field without a label, which the panel offers no checkbox for, is never hidden.
func (p preset) view(fields []field, headers bool) named {
	v := named{Name: p.Name, Headers: headers}
	if p.Headers != nil {
		v.Headers = *p.Headers
	}
	type at struct {
		id   string
		n    int
		desc bool
	}
	var order, groups, sorts []at
	for _, f := range fields {
		s := p.Fields[f.ID]
		placement := cmp.Or(s.Placement, f.Placement)
		switch {
		case placement == placeHidden && f.Label != "":
			v.Hidden = append(v.Hidden, f.ID)
		case placement == placeGroup:
			groups = append(groups, at{id: f.ID, n: s.Depth})
		case placement == placeColumn && s.Position > 0:
			order = append(order, at{id: f.ID, n: s.Position})
		}
		if s.Sort != "" {
			sorts = append(sorts, at{f.ID, s.Priority, s.Sort == "desc"})
		}
		if f.Owns && s.Filter != nil {
			switch {
			case slices.Equal(s.Filter.Include, []string{caller}):
				v.Show = "mine"
			case slices.Equal(s.Filter.Exclude, []string{caller}):
				v.Show = "others"
			}
		}
	}
	byN := func(a, b at) int { return cmp.Compare(a.n, b.n) }
	slices.SortStableFunc(order, byN)
	slices.SortStableFunc(groups, byN)
	slices.SortStableFunc(sorts, byN)
	for _, a := range order {
		v.Order = append(v.Order, a.id)
	}
	for _, a := range groups {
		v.Group = append(v.Group, a.id)
	}
	for _, a := range sorts {
		v.Sort = append(v.Sort, sortKey{a.id, a.desc})
	}
	return v
}

// views gives a stage's views, its own first, then the person's.
func (m Model) views(i int) []named {
	return append(m.builtins(i), m.saved[m.stages[i].name]...)
}

// builtin says whether a name is one of a stage's own views.
func (m Model) builtin(i int, name string) bool {
	return slices.ContainsFunc(m.builtins(i), func(v named) bool { return v.Name == name })
}

// chosen gives the view a stage's dropdown names, or Default when that view is gone.
func (m Model) chosen(i int) named {
	vs := m.views(i)
	if k := slices.IndexFunc(vs, func(v named) bool { return v.Name == m.stages[i].viewName }); k >= 0 {
		return vs[k]
	}
	return vs[slices.IndexFunc(vs, func(v named) bool { return v.Name == Opening })]
}

// current gives a stage's panel as a view would keep it.
func (m Model) current(i int) named {
	s := m.stages[i]
	p := s.view
	v := named{Order: p.Order, Group: p.Group, Headers: s.headers}
	for c, hidden := range p.Hidden {
		if hidden {
			v.Hidden = append(v.Hidden, c)
		}
	}
	for c, kept := range p.Kept {
		if kept {
			v.Kept = append(v.Kept, c)
		}
	}
	for _, k := range p.Sort {
		v.Sort = append(v.Sort, sortKey{k.Column, k.Desc})
	}
	switch p.Show {
	case grid.Mine:
		v.Show = "mine"
	case grid.Others:
		v.Show = "others"
	}
	return v
}

// same says whether two views set a stage's panel alike, their names apart: the same columns hidden, in the same
// order, sorted and grouped alike, keeping the same columns, showing the same rows, and the headers alike.
func (m Model) same(i int, a, b named) bool {
	cols := columnsOf(m.stages[i].name)
	normal := func(v named) named {
		v.Name = ""
		v.Order = grid.Panel{Order: v.Order}.Arrange(cols)
		v.Hidden = slices.DeleteFunc(slices.Sorted(slices.Values(v.Hidden)), func(c string) bool {
			return !slices.Contains(cols, c)
		})
		v.Kept = slices.Sorted(slices.Values(v.Kept))
		return v
	}
	a, b = normal(a), normal(b)
	return slices.Equal(a.Hidden, b.Hidden) && slices.Equal(a.Order, b.Order) && slices.Equal(a.Sort, b.Sort) &&
		slices.Equal(a.Group, b.Group) && slices.Equal(a.Kept, b.Kept) && a.Show == b.Show && a.Headers == b.Headers
}

// changed says whether the selected stage's panel differs from its view, or the view's name was changed: then 💾
// shows.
func (m Model) changed(i int) bool {
	s := m.stages[i]
	if s.renaming && strings.TrimSpace(s.naming) != s.viewName {
		return true
	}
	return !m.same(i, m.current(i), m.chosen(i))
}

// viewShown gives the name a stage's dropdown shows: as typed while the person renames the view; none once the panel
// differs from its view; else the view's.
func (m Model) viewShown(i int) string {
	s := m.stages[i]
	switch {
	case s.renaming:
		return s.naming
	case !m.same(i, m.current(i), m.chosen(i)):
		return ""
	}
	return s.viewName
}

// apply sets the selected stage's panel as the view keeps it, its search and cursor as they are, and names the view.
func (m *Model) apply(v named) {
	s := &m.stages[m.stage]
	p := s.view
	p.Hidden, p.Kept, p.Sort, p.Show = nil, nil, nil, grid.All
	p.Order, p.Group = slices.Clone(v.Order), slices.Clone(v.Group)
	for _, c := range v.Hidden {
		if p.Hidden == nil {
			p.Hidden = map[string]bool{}
		}
		p.Hidden[c] = true
	}
	for _, c := range v.Kept {
		if p.Kept == nil {
			p.Kept = map[string]bool{}
		}
		p.Kept[c] = true
	}
	for _, k := range v.Sort {
		p.Sort = append(p.Sort, grid.SortKey{Column: k.Column, Desc: k.Desc})
	}
	switch v.Show {
	case "mine":
		p.Show = grid.Mine
	case "others":
		p.Show = grid.Others
	}
	s.headers, s.viewName, s.renaming, s.naming = v.Headers, v.Name, false, ""
	m.setPanel(p)
}

// viewKey takes a key while the keyboard is on the dropdown, and says whether it did: enter and space drop the views
// down; ctrl+s saves; esc stops a rename; and what is typed renames the view: a letter first typed takes the name's
// place, as in a field whose text is selected, and backspace first takes the last letter of the name shown back.
func (m *Model) viewKey(k tea.KeyPressMsg) (tea.Cmd, bool) {
	s := &m.stages[m.stage]
	switch key := k.String(); {
	case key == "enter" || key == "space":
		vs := m.views(m.stage)
		m.viewMenu = true
		m.viewAt = slices.IndexFunc(vs, func(v named) bool { return v.Name == m.chosen(m.stage).Name })
		if m.viewAt >= len(m.builtins(m.stage)) {
			m.viewAt++ // past the rule between the stage's own and the person's
		}
		return nil, true
	case key == "ctrl+s":
		return m.saveView(), true
	case key == "esc" && s.renaming:
		s.renaming, s.naming = false, ""
		return nil, true
	case key == "backspace" || k.Text != "" && !strings.ContainsAny(k.Text, "=:;"):
		if !s.renaming {
			s.renaming, s.naming = true, ""
			if key == "backspace" {
				s.naming = m.viewShown(m.stage)
			}
		}
		s.naming, _ = shell.Type(s.naming, k)
		return nil, true
	}
	return nil, false
}

// saveView keeps the selected stage's panel under the dropdown's name: the person's view of that name is overwritten,
// a new one is added after theirs, and a stage's own name is refused, as those never change. What it did, or why not,
// is in place of the keys.
func (m *Model) saveView() tea.Cmd {
	s := &m.stages[m.stage]
	name := strings.TrimSpace(m.viewShown(m.stage))
	switch {
	case name == "":
		m.notice, m.noted = Failed+" a view needs a name: type one, then save", false
		return nil
	case m.builtin(m.stage, name):
		m.notice, m.noted = Failed+" "+name+" is the stage's own and does not change: type a new name, then save", false
		return nil
	}
	v := m.current(m.stage)
	v.Name = name
	saved := slices.Clone(m.saved[s.name])
	if k := slices.IndexFunc(saved, func(w named) bool { return w.Name == name }); k >= 0 {
		saved[k] = v
	} else {
		saved = append(saved, v)
	}
	all := map[string][]named{}
	for stage, vs := range m.saved {
		all[stage] = vs
	}
	all[s.name] = saved
	if err := writeViews(m.viewsPath, all); err != nil {
		m.notice, m.noted = fmt.Sprintf("%s the view could not be saved: %v", Failed, err), false
		return nil
	}
	m.saved = all
	s.viewName, s.renaming, s.naming = name, false, ""
	m.notice, m.noted = grid.Save+" saved the view "+name, true
	return nil
}

// viewItems gives the dropdown's menu: the stage's own views, a rule, and the person's, the view chosen checked.
func (m Model) viewItems() []shell.Item {
	var items []shell.Item
	chosen := m.chosen(m.stage).Name
	for k, v := range m.views(m.stage) {
		if k == len(m.builtins(m.stage)) {
			items = append(items, shell.Item{})
		}
		glyph := ""
		if v.Name == chosen {
			glyph = Checked
		}
		items = append(items, shell.Item{Glyph: glyph, Name: v.Name})
	}
	return items
}

// viewEnabled says whether a line of the dropdown's menu is a view, which the rule is not.
func (m Model) viewEnabled(k int) bool { return m.viewItems()[k].Name != "" }

// viewMenuKey takes a key while the dropdown's menu is open: up and down move, a letter to the next view it starts,
// enter and space choose, and esc closes it.
func (m Model) viewMenuKey(k tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	items := m.viewItems()
	switch k.String() {
	case "up":
		m.viewAt = shell.Move(items, m.viewEnabled, m.viewAt, -1)
	case "down":
		m.viewAt = shell.Move(items, m.viewEnabled, m.viewAt, 1)
	case "enter", "space":
		m.chooseView(m.viewAt)
	case "esc":
		m.viewMenu = false
	default:
		m.viewAt = shell.NextItem(items, m.viewEnabled, m.viewAt, k.Text)
	}
	return m, nil
}

// chooseView applies the view on the menu's line k, and closes the menu.
func (m *Model) chooseView(k int) {
	m.viewMenu = false
	if name := m.viewItems()[k].Name; name != "" {
		vs := m.views(m.stage)
		m.apply(vs[slices.IndexFunc(vs, func(v named) bool { return v.Name == name })])
	}
}

// viewBox gives the dropdown's menu's lines, under the dropdown's line from its first column, or nearer the screen's
// left when that would run past its edge, and the screen line and column it is drawn from.
func (m Model) viewBox() (lines []string, top, left int) {
	lines, left = shell.Box(m.viewItems(), m.viewEnabled, m.viewAt, "", -1, m.width)
	return lines, m.viewRow() + 1, min(m.width-m.sideWidth(), left)
}

// viewRow gives the screen line the dropdown is drawn on, as the sidebar is scrolled, or -1 when it is scrolled away.
func (m Model) viewRow() int {
	p := m.panel(m.stage)
	for y := 1; y < m.sideHeight(); y++ {
		if p.ViewAt(m.panelLine(m.width-1, y)) {
			return y
		}
	}
	return -1
}

// viewMenuClick takes a click while the dropdown's menu is open: on a view, it chooses it; anywhere else, it closes
// the menu.
func (m *Model) viewMenuClick(x, y int) {
	box, top, left := m.viewBox()
	if k := y - top - 1; x > left && x < left+ansi.StringWidth(box[0])-1 && k >= 0 && k < len(box)-2 &&
		m.viewEnabled(k) {
		m.chooseView(k)
		return
	}
	m.viewMenu = false
}

// viewClick takes a click on the dropdown's line: on 💾 Save, at the line's right end, it saves, or says why not; on ▾, or past it, the
// dropdown takes the keyboard and drops the views down; on the name, the dropdown takes the keyboard, so what is typed
// renames the view.
func (m *Model) viewClick(x int) tea.Cmd {
	p := m.panel(m.stage)
	if p.Changed && x >= m.width-1-ansi.StringWidth(grid.Saving) {
		return m.saveView()
	}
	// the dropdown is the control after the search
	m.inPanel, m.stages[m.stage].view.Cursor = true, 1
	if x < m.dropAt() {
		return nil
	}
	cmd, _ := m.viewKey(tea.KeyPressMsg{Code: tea.KeyEnter})
	return cmd
}

// dropAt gives the screen column of the dropdown's ▾, after the name as the panel draws it.
func (m Model) dropAt() int {
	p := m.panel(m.stage)
	name := p.View
	switch {
	case p.Naming:
		name += "▏"
	case name == "":
		name = grid.Unnamed
	}
	return m.width - m.sideWidth() + ansi.StringWidth(name) + 1
}
