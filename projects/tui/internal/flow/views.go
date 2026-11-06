// Named views: a stage's panel as it is set, each field's placement, sort, filter, and rendering, and whether it folds,
// kept under a name as a preset. Every stage with columns has its own views, the presets the field catalog gives it,
// which never change: Default, the stage as it opens, and Compact, the few columns that say what a row is. The person
// saves others, per stage, in a file of their own, presets.json under ~/.config/kit/, so their views follow them to
// every repository; one they named as a stage's own was named before the stage had it, and stays theirs. The preset
// line, the rule under the stage's header, names the view the table is set as; enter, or a click on the name, drops
// down the views to choose from; typing renames it. Once the table differs from every view, the line names none, and 💾
// Save stands at its right, dim until a name is typed; it stands there too once the name was changed. ctrl+s or a click
// on it saves the panel under the name, a view of the person's own overwritten, a stage's own refused for a name of
// their own, and no name refused.
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

// Save is the glyph of the button at the preset line's right while the table differs from its view, SaveText the word
// after it, and Saving the two, as the button draws them. Unnamed stands, dim, in place of the view's name while the
// line names none: the table differs from every view, and nothing is typed.
const (
	Save     = "💾"
	SaveText = "Save"
	Saving   = Save + " " + SaveText
	Unnamed  = "unnamed"
)

// named is a stage's panel as views.json, the views file before presets.json, keeps it, by the columns' labels: the
// columns hidden, their order, the sort's keys, the groups, the columns kept from grouping on their own, whose rows it
// shows, and whether it folds, when it does not. It said whether the headers showed, which they now always do.
type named struct {
	Name   string    `json:"name"`
	Hidden []string  `json:"hidden,omitempty"`
	Order  []string  `json:"order,omitempty"`
	Sort   []sortKey `json:"sort,omitempty"`
	Group  []string  `json:"group,omitempty"`
	Kept   []string  `json:"kept,omitempty"`
	Show   string    `json:"show,omitempty"`
	Fold   *bool     `json:"fold,omitempty"`
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
		note = Save + " views.json is now presets.json, without the columns kittui no longer knows: " +
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

// builtins gives a stage's own views, its presets in the catalog, in the catalog's order; a stage without columns has
// Default alone, as it opens.
func (m Model) builtins(i int) []preset {
	if s := catalog[m.stages[i].name]; s != nil {
		return slices.Clone(s.Presets)
	}
	return []preset{{Name: Opening}}
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
	p.Fold, p.Glyphs = pr.Fold == nil || *pr.Fold, pr.Glyphs
	return p
}

// presetOf gives a stage's panel as a preset keeps it, of the stage's fields, unnamed: of each field, what differs from
// what a preset that names none of them says. A column's position is said only while the columns stand in another order
// than the stage's, and then for every column, from 1; a group's depth and a sort's priority are numbered from 1, the
// outermost and the first first.
func presetOf(p grid.Panel, fields []field) preset {
	fold := p.Fold
	out := preset{Fold: &fold, Glyphs: p.Glyphs}
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
// owns them: mine as the caller alone, others as anyone but the caller; and whether it folds, the headers aside. The
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
	// a field the catalog hides, which views.json knew nothing of, stays hidden, as are those it hid
	for _, f := range fields {
		if f.Placement == placeHidden {
			p = p.Placed(f.ID, grid.Hidden)
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
	out := presetOf(p, fields)
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

// chosen gives the view a stage's preset line was last set to, or Default when that view is gone.
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
	return presetOf(s.view, fieldsOf(s.name))
}

// same says whether two views set a stage's panel alike, their names apart: each field placed, sorted, filtered, and
// drawn alike, and folding alike.
func (m Model) same(i int, a, b preset) bool {
	fields := fieldsOf(m.stages[i].name)
	normal := func(v preset) preset { return presetOf(v.panel(fields, grid.Panel{}), fields) }
	return reflect.DeepEqual(normal(a), normal(b))
}

// is gives the name of the view a stage's panel is set as: the view chosen, when the panel is as it says; else the
// first of the stage's views it is as; none when it differs from every one.
func (m Model) is(i int) string {
	now := m.current(i)
	if m.same(i, now, m.chosen(i)) {
		return m.chosen(i).Name
	}
	for _, v := range m.views(i) {
		if m.same(i, now, v) {
			return v.Name
		}
	}
	return ""
}

// standing gives where a stage's panel stands among its views: the view it is set as, as is gives it; whether it
// differs from every view, or the name was changed, when 💾 shows; and whether there is a name to save it under then,
// one shown and not a stage's own.
func (m Model) standing(i int) (is string, changed, savable bool) {
	s := m.stages[i]
	is = m.is(i)
	name := is
	if s.renaming {
		name = strings.TrimSpace(s.naming)
	}
	changed = is == "" || s.renaming && name != is
	return is, changed, changed && name != "" && !m.builtin(i, name)
}

// changed says whether a stage's panel differs from every view, or the name was changed: then 💾 shows.
func (m Model) changed(i int) bool {
	_, changed, _ := m.standing(i)
	return changed
}

// savable says whether there is a name to save a stage's panel under, while 💾 shows: one typed, not a stage's own.
func (m Model) savable(i int) bool {
	_, _, savable := m.standing(i)
	return savable
}

// viewShown gives the name a stage's preset line shows: as typed while the person renames the view; else the view the
// panel is set as, or none once it differs from every one.
func (m Model) viewShown(i int) string {
	if s := m.stages[i]; s.renaming {
		return s.naming
	}
	return m.is(i)
}

// apply sets the selected stage's panel as the view keeps it, its search, cursor, and groups closed as they are, and
// names the view.
func (m *Model) apply(v preset) {
	s := &m.stages[m.stage]
	p := v.panel(fieldsOf(s.name), s.view)
	s.viewName, s.renaming, s.naming = v.Name, false, ""
	m.setPanel(p)
}

// presetKey takes a key while the keyboard is on the preset line, and says whether it did: up and esc give the bar the
// keyboard, and down the rows, when the stage has some; enter and space drop the presets down; ctrl+s saves, while 💾
// shows; esc stops a rename; and what is typed renames the view, = : and ; apart, which the menus take: a letter first
// typed takes the name's place, as in a field whose text is selected, and backspace first takes the last letter of the
// name shown back.
func (m *Model) presetKey(k tea.KeyPressMsg) (tea.Cmd, bool) {
	s := &m.stages[m.stage]
	switch key := k.String(); {
	case key == "esc" && s.renaming:
		s.renaming, s.naming = false, ""
	case key == "esc" || key == "up":
		m.onPreset = false
	case key == "down":
		if len(m.rows(m.stage)) > 0 {
			m.onPreset = false
			m.down(1)
		}
	case key == "enter" || key == "space":
		m.dropPresets()
	case key == "ctrl+s" && m.changed(m.stage):
		return m.savePreset(), true
	case key == "backspace" || k.Text != "" && !strings.ContainsAny(k.Text, "=:;"):
		if !s.renaming {
			s.renaming, s.naming = true, ""
			if key == "backspace" {
				s.naming = m.viewShown(m.stage)
			}
		}
		s.naming, _ = shell.Type(s.naming, k)
	default:
		return nil, false
	}
	return nil, true
}

// dropPresets drops the selected stage's presets down, the one the line names under the keyboard, else the one it was
// last set to.
func (m *Model) dropPresets() {
	vs, name := m.views(m.stage), cmp.Or(m.is(m.stage), m.chosen(m.stage).Name)
	m.presetMenu = true
	m.presetAt = slices.IndexFunc(vs, func(v preset) bool { return v.Name == name })
	if m.presetAt >= len(m.builtins(m.stage)) {
		m.presetAt++ // past the rule between the stage's own and the person's
	}
}

// savePreset keeps the selected stage's panel under the name the preset line shows: the person's view of that name is
// overwritten, a new one is added after theirs, and a stage's own name is refused, as those never change. What it
// did, or why not, is in place of the keys.
func (m *Model) savePreset() tea.Cmd {
	s := &m.stages[m.stage]
	name := strings.TrimSpace(m.viewShown(m.stage))
	switch {
	case name == "":
		m.notice, m.noted = Failed+" a preset needs a name: type one, then save", false
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
		m.notice, m.noted = fmt.Sprintf("%s the preset could not be saved: %v", Failed, err), false
		return nil
	}
	m.saved = all
	s.viewName, s.renaming, s.naming = name, false, ""
	m.notice, m.noted = Save+" saved the preset "+name, true
	return nil
}

// presetItems gives the presets' menu: the stage's own views, a rule, and the person's, the one the line names
// checked.
func (m Model) presetItems() []shell.Item {
	var items []shell.Item
	is := m.is(m.stage)
	for k, v := range m.views(m.stage) {
		if k == len(m.builtins(m.stage)) {
			items = append(items, shell.Item{})
		}
		glyph := ""
		if v.Name == is {
			glyph = Checked
		}
		items = append(items, shell.Item{Glyph: glyph, Name: v.Name})
	}
	return items
}

// presetEnabled says whether a line of the presets' menu is a view, which the rule is not.
func (m Model) presetEnabled(k int) bool { return m.presetItems()[k].Name != "" }

// presetMenuKey takes a key while the presets' menu is open: up and down move, a letter to the next view it starts,
// enter and space choose, and esc closes it.
func (m Model) presetMenuKey(k tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	items := m.presetItems()
	switch k.String() {
	case "up":
		m.presetAt = shell.Move(items, m.presetEnabled, m.presetAt, -1)
	case "down":
		m.presetAt = shell.Move(items, m.presetEnabled, m.presetAt, 1)
	case "enter", "space":
		m.choosePreset(m.presetAt)
	case "esc":
		m.presetMenu = false
	default:
		m.presetAt = shell.NextItem(items, m.presetEnabled, m.presetAt, k.Text)
	}
	return m, nil
}

// choosePreset applies the view on the menu's line k, and closes the menu.
func (m *Model) choosePreset(k int) {
	m.presetMenu = false
	if name := m.presetItems()[k].Name; name != "" {
		vs := m.views(m.stage)
		m.apply(vs[slices.IndexFunc(vs, func(v preset) bool { return v.Name == name })])
	}
}

// presetBox gives the presets' menu's lines, under the preset line, its left edge a cell before the name's, or nearer
// the screen's left when that would run past its edge, and the screen line and column it is drawn from.
func (m Model) presetBox() (lines []string, top, left int) {
	lines, _ = shell.Box(m.presetItems(), m.presetEnabled, m.presetAt, "", -1, m.width)
	return lines, m.presetTop() + 1, max(0, min(m.dropStart()-1, m.width-ansi.StringWidth(lines[0])))
}

// presetMenuClick takes a click while the presets' menu is open: on a view, it chooses it; anywhere else, it closes the
// menu.
func (m *Model) presetMenuClick(x, y int) {
	box, top, left := m.presetBox()
	if k := y - top - 1; x > left && x < left+ansi.StringWidth(box[0])-1 && k >= 0 && k < len(box)-2 &&
		m.presetEnabled(k) {
		m.choosePreset(k)
		return
	}
	m.presetMenu = false
}

// presetClick takes a click on the preset line, which then has the keyboard: on 💾 Save, at the line's right end, it
// saves, or says why not; on the name or ▾, it drops the presets down.
func (m *Model) presetClick(x int) tea.Cmd {
	m.onPreset, m.inRows, m.inPanel = true, false, false
	switch {
	case m.changed(m.stage) && x >= m.paneWidth()-ansi.StringWidth(Saving) && x < m.paneWidth():
		return m.savePreset()
	case x >= m.dropStart() && x < m.dropEnd():
		m.dropPresets()
	}
	return nil
}
