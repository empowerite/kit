// Named views: a stage's panel as it is set, each field's placement, sort, filter, and rendering, whether it folds, and
// whether its headers stand over the rows, kept under a name as a preset. Every stage with columns has its own views,
// the presets the field catalog gives it, which never change: Default, the stage as it opens, and Compact, the few
// columns that say what a row is. The person saves others, per stage, in a file of their own, presets.json under
// ~/.config/kit/, so their views follow them to every repository; one they named as a stage's own was named before the
// stage had it, and stays theirs. The panel's dropdown, under its search, names the view; enter, or a click on it, drops down the views
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
	"maps"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/empowerite/kit/projects/tui/internal/grid"
	"github.com/empowerite/kit/projects/tui/internal/shell"
)

// Opening is the name of the view a stage opens on, one of its own.
const Opening = "Default"

// named is a stage's panel as views.json, the views file before presets.json, keeps it, by the columns' labels: the
// columns hidden, their order, the sort's keys, the groups, the columns kept from grouping on their own, whose rows it
// shows, whether the headers show, and whether it folds, when it does not.
type named struct {
	Name    string    `json:"name"`
	Hidden  []string  `json:"hidden,omitempty"`
	Order   []string  `json:"order,omitempty"`
	Sort    []sortKey `json:"sort,omitempty"`
	Group   []string  `json:"group,omitempty"`
	Kept    []string  `json:"kept,omitempty"`
	Show    string    `json:"show,omitempty"`
	Headers bool      `json:"headers"`
	Fold    *bool     `json:"fold,omitempty"`
}

// sortKey is one of a view's sort keys: a column, and whether the larger come first.
type sortKey struct {
	Column string `json:"column"`
	Desc   bool   `json:"desc,omitempty"`
}

// presetsFile gives the file the person's presets are kept in: kit/presets.json under XDG_CONFIG_HOME, else under
// ~/.config; nothing when neither is known.
func presetsFile() string {
	dir := os.Getenv("XDG_CONFIG_HOME")
	if dir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return ""
		}
		dir = filepath.Join(home, ".config")
	}
	return filepath.Join(dir, "kit", "presets.json")
}

// presetsVersion is the version of presets.json; views.json, the file before it, was the first.
const presetsVersion = 2

// presets is presets.json: its version, and the person's presets, by stage, each in the catalog's vocabulary, its
// fields by id.
type presets struct {
	Version int                 `json:"version"`
	Stages  map[string][]preset `json:"stages"`
}

// readPresets gives the presets the file keeps, by stage; none when there is no file. With no presets.json, it reads
// views.json beside it, the file before it, once: each view a preset, its labels the fields' ids, which it then keeps
// in presets.json, leaving views.json as it is. It says what it noted, the labels the catalog no longer knows, which
// are dropped, or none. A file that does not read, or is of another version, is the error, and nothing is written.
func readPresets(path string) (map[string][]preset, string, error) {
	out := map[string][]preset{}
	if path == "" {
		return out, "", nil
	}
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return migrate(path)
	}
	var file presets
	if err == nil {
		err = json.Unmarshal(data, &file)
	}
	if err == nil && file.Version != presetsVersion {
		err = fmt.Errorf("version %d, not %d", file.Version, presetsVersion)
	}
	if err != nil {
		return out, "", fmt.Errorf("%s does not read as presets: %w", path, err)
	}
	for stage, ps := range file.Stages {
		out[stage] = ps
	}
	return out, "", nil
}

// migrate reads views.json beside the presets file, when there is one, as readPresets says, and writes the presets
// file from it. Its note names each label dropped, after its stage and its view.
func migrate(path string) (map[string][]preset, string, error) {
	out := map[string][]preset{}
	old := filepath.Join(filepath.Dir(path), "views.json")
	data, err := os.ReadFile(old)
	if errors.Is(err, os.ErrNotExist) {
		return out, "", nil
	}
	var kept map[string][]named
	if err == nil {
		err = json.Unmarshal(data, &kept)
	}
	if err != nil {
		return out, "", fmt.Errorf("%s does not read as views: %w", old, err)
	}
	var dropped []string
	for _, stage := range slices.Sorted(maps.Keys(kept)) {
		for _, v := range kept[stage] {
			p, labels := v.preset(stage)
			out[stage] = append(out[stage], p)
			if len(labels) > 0 {
				dropped = append(dropped, fmt.Sprintf("%s in %s %s", strings.Join(labels, ", "), stage, v.Name))
			}
		}
	}
	note := ""
	if len(dropped) > 0 {
		note = grid.Save + " views.json is now presets.json, without the columns kittui no longer knows: " +
			strings.Join(dropped, "; ")
	}
	if err := writePresets(path, out); err != nil {
		return out, "", fmt.Errorf("%s could not be written from %s: %w", path, old, err)
	}
	return out, note, nil
}

// writePresets keeps the presets in the file, written whole beside it first, so a reader never sees half of it.
func writePresets(path string, stages map[string][]preset) error {
	if path == "" {
		return nil
	}
	data, err := json.MarshalIndent(presets{Version: presetsVersion, Stages: stages}, "", "  ")
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

// builtins gives a stage's own views, its presets in the catalog, in the catalog's order, each with its headers as the
// screen opens them where it says nothing; a stage without columns has Default alone, as it opens.
func (m Model) builtins(i int) []preset {
	headers := !m.noHeaders
	s := catalog[m.stages[i].name]
	if s == nil {
		return []preset{{Name: Opening, Headers: &headers}}
	}
	var out []preset
	for _, p := range s.Presets {
		if p.Headers == nil {
			p.Headers = &headers
		}
		out = append(out, p)
	}
	return out
}

// panel gives a stage's panel set as a preset says, of the stage's fields, the rest of it as p has it: each field placed,
// sorted, filtered, and drawn as the preset says, a field it does not name placed as the catalog says, and folding
// unless the preset says it does not. What a field cannot do it is not set to do: a field without a label is never
// hidden, nor grouped, sorted, or filtered unless it can be, nor drawn by a rendering not its own.
func (pr preset) panel(fields []field, p grid.Panel) grid.Panel {
	p.Fields = map[string]grid.Field{}
	for _, f := range fields {
		s := pr.Fields[f.ID]
		var g grid.Field
		switch cmp.Or(s.Placement, f.Placement) {
		case placeHidden:
			if f.Label != "" {
				g.Placement = grid.Hidden
			}
		case placeGroup:
			if f.Group {
				g.Placement, g.Depth = grid.Group, s.Depth
			}
		default:
			g.Position = s.Position
		}
		if f.Sort && (s.Sort == "asc" || s.Sort == "desc") {
			g.Sort, g.Priority = grid.Ascending, s.Priority
			if s.Sort == "desc" {
				g.Sort = grid.Descending
			}
		}
		if s.Filter != nil && f.Filter {
			switch {
			case len(s.Filter.Include) > 0:
				g.Filter = grid.Filter{Values: slices.Clone(s.Filter.Include)}
			case len(s.Filter.Exclude) > 0:
				g.Filter = grid.Filter{Values: slices.Clone(s.Filter.Exclude), Exclude: true}
			}
		}
		if slices.Contains(f.Renderings, s.Rendering) {
			g.Rendering = s.Rendering
		}
		p.Fields[f.ID] = g
	}
	p.Fold = pr.Fold == nil || *pr.Fold
	return p
}

// presetOf gives a stage's panel as a preset keeps it, of the stage's fields, unnamed, with the headers as they are: of
// each field, what differs from what a preset that names none of them says. A column's position is said only while
// the columns stand in another order than the stage's, and then for every column, from 1; a group's depth and a sort's
// priority are numbered from 1, the outermost and the first first.
func presetOf(p grid.Panel, fields []field, headers bool) preset {
	fold := p.Fold
	out := preset{Headers: &headers, Fold: &fold}
	var ids []string
	for _, f := range fields {
		if f.Label != "" {
			ids = append(ids, f.ID)
		}
	}
	columns := slices.DeleteFunc(p.Arrange(ids), func(id string) bool { return !p.Shows(id) })
	positioned := !slices.Equal(columns, slices.DeleteFunc(slices.Clone(ids), func(id string) bool { return !p.Shows(id) }))
	groups, sorts := p.Groups(), p.SortKeys()
	for _, f := range fields {
		g := p.Field(f.ID)
		var s setting
		switch k := slices.Index(columns, f.ID); {
		case g.Placement == grid.Hidden && f.Label != "":
			if f.Placement != placeHidden {
				s.Placement = placeHidden
			}
		case g.Placement == grid.Group && slices.Contains(groups, f.ID):
			s.Placement, s.Depth = placeGroup, slices.Index(groups, f.ID)+1
		case positioned && k >= 0:
			s.Placement, s.Position = placeColumn, k+1
		case cmp.Or(f.Placement, placeColumn) != placeColumn:
			s.Placement = placeColumn
		}
		if k := slices.IndexFunc(sorts, func(s grid.SortKey) bool { return s.Column == f.ID }); k >= 0 {
			s.Sort, s.Priority = "asc", k+1
			if sorts[k].Desc {
				s.Sort = "desc"
			}
		}
		if len(g.Filter.Values) > 0 {
			s.Filter = &valueFilter{Include: slices.Clone(g.Filter.Values)}
			if g.Filter.Exclude {
				s.Filter = &valueFilter{Exclude: slices.Clone(g.Filter.Values)}
			}
		}
		s.Rendering = g.Rendering
		if s != (setting{}) {
			if out.Fields == nil {
				out.Fields = map[string]setting{}
			}
			out.Fields[f.ID] = s
		}
	}
	return out
}

// preset gives a view of views.json as a preset of the stage's: each column's label, or its id, as the field's id,
// the catalog's; the columns hidden, those it groups by, outermost first, and those it sorts by, first first, as their
// placements and sorts; the columns' order as their positions; whose rows it shows as the filter on the field that
// owns them: mine as the caller alone, others as anyone but the caller; and the headers, and whether it folds. The
// columns it kept from grouping on their own are dropped, as the person no longer keeps a column so; it folds, or does
// not, as a whole. It gives the labels the catalog does not know, each once, which are dropped.
func (v named) preset(stage string) (preset, []string) {
	fields := fieldsOf(stage)
	var dropped []string
	id := func(label string) string {
		k := slices.IndexFunc(fields, func(f field) bool { return f.Label != "" && f.Label == label })
		if k < 0 {
			k = slices.IndexFunc(fields, func(f field) bool { return f.Label != "" && f.ID == label })
		}
		if k < 0 {
			if !slices.Contains(dropped, label) {
				dropped = append(dropped, label)
			}
			return ""
		}
		return fields[k].ID
	}
	p := grid.Panel{Groupable: groupableOf(stage), Sortable: sortableOf(stage), Owner: ownerOf(stage),
		Fold: v.Fold == nil || *v.Fold, Fields: map[string]grid.Field{}}
	// the columns the order names stand first, in its order, the rest after them, in the stage's
	for _, label := range v.Order {
		if c := id(label); c != "" && p.Fields[c].Position == 0 {
			p.Fields[c] = grid.Field{Position: len(p.Fields) + 1}
		}
	}
	for _, label := range v.Hidden {
		if c := id(label); c != "" {
			p = p.Placed(c, grid.Hidden)
		}
	}
	var groups []string
	for _, label := range v.Group {
		if c := id(label); c != "" {
			groups = append(groups, c)
		}
	}
	for k, c := range groups {
		p = p.Grouped(c, k > 0)
	}
	for _, s := range v.Sort {
		if c := id(s.Column); c != "" {
			p = p.Sorted(c, true)
			if s.Desc {
				p = p.Sorted(c, true)
			}
		}
	}
	switch v.Show {
	case "mine":
		p = p.Shown(grid.Mine)
	case "others":
		p = p.Shown(grid.Others)
	}
	out := presetOf(p, fields, v.Headers)
	out.Name = v.Name
	return out, dropped
}

// views gives a stage's views, its own first, then the person's.
func (m Model) views(i int) []preset {
	return append(m.builtins(i), m.saved[m.stages[i].name]...)
}

// builtin says whether a name is one of a stage's own views.
func (m Model) builtin(i int, name string) bool {
	return slices.ContainsFunc(m.builtins(i), func(v preset) bool { return v.Name == name })
}

// chosen gives the view a stage's dropdown names, or Default when that view is gone.
func (m Model) chosen(i int) preset {
	vs := m.views(i)
	if k := slices.IndexFunc(vs, func(v preset) bool { return v.Name == m.stages[i].viewName }); k >= 0 {
		return vs[k]
	}
	return vs[slices.IndexFunc(vs, func(v preset) bool { return v.Name == Opening })]
}

// current gives a stage's panel as a view would keep it.
func (m Model) current(i int) preset {
	s := m.stages[i]
	return presetOf(s.view, fieldsOf(s.name), s.headers)
}

// same says whether two views set a stage's panel alike, their names apart: each field placed, sorted, filtered, and
// drawn alike, folding alike, and the headers alike, a view that says nothing of them as the screen opens them.
func (m Model) same(i int, a, b preset) bool {
	fields := fieldsOf(m.stages[i].name)
	normal := func(v preset) preset {
		headers := !m.noHeaders
		if v.Headers != nil {
			headers = *v.Headers
		}
		return presetOf(v.panel(fields, grid.Panel{}), fields, headers)
	}
	return reflect.DeepEqual(normal(a), normal(b))
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

// apply sets the selected stage's panel as the view keeps it, its search, cursor, and groups closed as they are, and
// names the view.
func (m *Model) apply(v preset) {
	s := &m.stages[m.stage]
	p := v.panel(fieldsOf(s.name), s.view)
	s.headers = !m.noHeaders
	if v.Headers != nil {
		s.headers = *v.Headers
	}
	s.viewName, s.renaming, s.naming = v.Name, false, ""
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
		m.viewAt = slices.IndexFunc(vs, func(v preset) bool { return v.Name == m.chosen(m.stage).Name })
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
	if k := slices.IndexFunc(saved, func(w preset) bool { return w.Name == name }); k >= 0 {
		saved[k] = v
	} else {
		saved = append(saved, v)
	}
	all := map[string][]preset{}
	for stage, vs := range m.saved {
		all[stage] = vs
	}
	all[s.name] = saved
	if err := writePresets(m.presetsPath, all); err != nil {
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
		m.apply(vs[slices.IndexFunc(vs, func(v preset) bool { return v.Name == name })])
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
