// Named views: a stage's panel as it is set, each field's placement, sort, filter, and rendering, and whether it folds,
// kept under a name as a preset. Every stage with columns has its own views, the presets the field catalog gives it,
// which never change: Default, the stage as it opens, and Compact, the few columns that say what a row is. The person
// saves others, per stage, in a file of their own, presets.json under ~/.config/kit/, so their views follow them to
// every repository; one they named as a stage's own was named before the stage had it, and stays theirs. The preset
// line, the rule under the stage's header, ends in the name of the view the table is set as and ▼, the name struck through
// once the table differs from every view; enter, or a click on it, drops down what can be done with views, then the views to
// choose from. Save keeps the panel under the person's view it was set to; Save As… takes a name typed in the
// selector's place, the person's view the line was set to typed already, and enter saves the panel under it, a view of
// the person's own overwritten, a stage's own refused, and no name refused; Rename… takes a new name for the person's
// view, as Save As… takes one, and Delete drops it, the stage back to Default. A stage's own views are never saved
// over, renamed, or deleted: those lines stay, dim, while the line names one.
package flow

import (
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/empowerite/kit/projects/tui/internal/grid"
	"github.com/empowerite/kit/projects/tui/internal/table"
)

// Opening is the name of the view a stage opens on, one of its own.
const Opening = table.Opening

// The preset line's words and glyphs, internal/table's: Presets ends its selector; the presets' menu's lines; and
// what each did starts with.
const (
	Presets  = table.Presets
	FieldsIt = table.FieldsIt
	SaveIt   = table.SaveIt
	SaveAs   = table.SaveAs
	RenameIt = table.RenameIt
	DeleteIt = table.DeleteIt
	SavingAs = table.SavingAsWord
	Renaming = table.RenamingWord
	Save     = table.Save
	Renamed  = table.Renamed
	Deleted  = table.Deleted
)

// named is a stage's panel as views.json, the views file before presets.json, keeps it, by the columns' labels: the
// columns hidden, their order, the sort's keys, the groups, the columns kept from grouping on their own, and whose rows
// it shows. It said whether the headers showed, which they now always do, and whether it folded, which kittui's
// settings now say for every view.
type named struct {
	Name   string    `json:"name"`
	Hidden []string  `json:"hidden,omitempty"`
	Order  []string  `json:"order,omitempty"`
	Sort   []sortKey `json:"sort,omitempty"`
	Group  []string  `json:"group,omitempty"`
	Kept   []string  `json:"kept,omitempty"`
	Show   string    `json:"show,omitempty"`
}

// sortKey is one of a view's sort keys: a column, and whether the larger come first.
type sortKey struct {
	Column string `json:"column"`
	Desc   bool   `json:"desc,omitempty"`
}

// readPresets gives the presets the file keeps, by stage; none when there is no file. With no presets.json, it reads
// views.json beside it, the file before it, once: each view a preset, its labels the fields' ids, which it then keeps
// in presets.json, leaving views.json as it is. It says what it noted, the labels the catalog no longer knows, which
// are dropped, or none. A file that does not read, or is of another version, is the error, and nothing is written.
func readPresets(path string) (map[string][]table.Preset, string, error) {
	out, err := table.ReadPresets(path)
	if errors.Is(err, os.ErrNotExist) {
		return migrate(path)
	}
	return out, "", err
}

// migrate reads views.json beside the presets file, when there is one, as readPresets says, and writes the presets
// file from it. Its note names each label dropped, after its stage and its view.
func migrate(path string) (map[string][]table.Preset, string, error) {
	out := map[string][]table.Preset{}
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
	if err := table.WritePresets(path, out); err != nil {
		return out, "", fmt.Errorf("%s could not be written from %s: %w", path, old, err)
	}
	return out, note, nil
}

// builtins gives a stage's own views, its presets in the catalog, in the catalog's order; a stage without columns has
// Default alone, as it opens.
func (m Model) builtins(i int) []table.Preset {
	if s := catalog[m.stages[i].name]; s != nil {
		return slices.Clone(s.Presets)
	}
	return []table.Preset{{Name: Opening}}
}

// preset gives a view of views.json as a preset of the stage's: each column's label, or its id, as the field's id,
// the catalog's; the columns hidden, those it groups by, outermost first, and those it sorts by, first first, as their
// placements and sorts; the columns' order as their positions; whose rows it shows as the filter on the field that
// owns them: mine as the caller alone, others as anyone but the caller; and whether it folds, the headers aside. The
// columns it kept from grouping on their own are dropped, as the person no longer keeps a column so; it folds, or does
// not, as a whole. It gives the labels the catalog does not know, each once, which are dropped.
func (v named) preset(stage string) (table.Preset, []string) {
	fields := fieldsOf(stage)
	var dropped []string
	id := func(label string) string {
		k := slices.IndexFunc(fields, func(f table.Field) bool { return f.Label != "" && f.Label == label })
		if k < 0 {
			k = slices.IndexFunc(fields, func(f table.Field) bool { return f.Label != "" && f.ID == label })
		}
		if k < 0 {
			if !slices.Contains(dropped, label) {
				dropped = append(dropped, label)
			}
			return ""
		}
		return fields[k].ID
	}
	p := grid.Panel{Groupable: groupableOf(stage), Sortable: sortableOf(stage), Ones: onesOf(stage), Owner: ownerOf(stage),
		Fields: map[string]grid.Field{}}
	// the columns the order names stand first, in its order, the rest after them, in the stage's
	for _, label := range v.Order {
		if c := id(label); c != "" && p.Fields[c].Position == 0 {
			p.Fields[c] = grid.Field{Position: len(p.Fields) + 1}
		}
	}
	// a field the catalog hides, which views.json knew nothing of, stays hidden, as are those it hid
	for _, f := range fields {
		if f.Placement == table.PlaceHidden {
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
			p = p.Sorted(c)
			if s.Desc {
				p = p.Sorted(c)
			}
		}
	}
	switch v.Show {
	case "mine":
		p = p.Shown(grid.Mine)
	case "others":
		p = p.Shown(grid.Others)
	}
	out := table.PresetOf(p, fields)
	out.Name = v.Name
	return out, dropped
}

// viewsOf gives a stage's views, as table.Views keeps them: its fields, its own views, and the person's.
func (m Model) viewsOf(i int) table.Views {
	name := m.stages[i].name
	return table.Views{Fields: fieldsOf(name), Own: m.builtins(i), Saved: m.saved[name]}
}

// views gives a stage's views, its own first, then the person's.
func (m Model) views(i int) []table.Preset { return m.viewsOf(i).All() }

// builtin says whether a name is one of a stage's own views.
func (m Model) builtin(i int, name string) bool { return m.viewsOf(i).Builtin(name) }

// chosen gives the view a stage's preset line was last set to, or Default when that view is gone.
func (m Model) chosen(i int) table.Preset { return m.viewsOf(i).Chosen(m.stages[i].ViewName) }

// current gives a stage's panel as a view would keep it.
func (m Model) current(i int) table.Preset { return m.viewsOf(i).Current(m.stages[i].View) }

// same says whether two views set a stage's panel alike, their names apart.
func (m Model) same(i int, a, b table.Preset) bool { return m.viewsOf(i).Same(a, b) }

// is gives the name of the view a stage's panel is set as, as table.Views says; none when it differs from every one.
func (m Model) is(i int) string { return m.viewsOf(i).Is(m.stages[i].Table) }

// apply sets the selected stage's panel as the view keeps it, its search, cursor, and groups closed as they are, and
// names the view, which then groups the rows as it loads, as autoGroup says.
func (m *Model) apply(v table.Preset) {
	s := &m.stages[m.stage]
	p := v.Panel(fieldsOf(s.name), s.View)
	s.ViewName, s.SavingAs, s.Naming = v.Name, false, ""
	s.Grouping, s.Grouped = true, nil
	m.setPanel(p)
	m.autoGroup(s.name)
	m.clamp(m.stage)
	m.fit()
}

// presetKey takes a key while the keyboard is on the preset line, and says whether it did, as table.PresetKey says;
// it saves under the name typed, or renames, or drops the presets down, where the key asks it to.
func (m *Model) presetKey(k tea.KeyPressMsg) (tea.Cmd, bool) {
	do, took := table.PresetKey(m, k)
	switch do {
	case table.DoSave:
		return m.savePreset(), true
	case table.DoRename:
		m.renamePreset()
	case table.DoDrop:
		m.presetsMenu = m.presetsMenu.Drop(m)
	}
	return nil, took
}

// own gives the person's view the selected stage's preset line was set to, and whether it is one: a stage's own is
// not, nor is a view since deleted.
func (m Model) own() (table.Preset, bool) { return m.viewsOf(m.stage).Mine(m.stages[m.stage].ViewName) }

// keep writes the person's views of the selected stage as they are given, and says whether it could; when it could
// not, the notice says why, and nothing changes.
func (m *Model) keep(saved []table.Preset, what string) bool {
	all := map[string][]table.Preset{}
	for stage, vs := range m.saved {
		all[stage] = vs
	}
	all[m.stages[m.stage].name] = saved
	if err := table.WritePresets(m.presetsPath, all); err != nil {
		m.note.Text, m.note.Done = fmt.Sprintf("%s the preset could not be %s: %v", Failed, what, err), false
		return false
	}
	m.saved = all
	return true
}

// savePreset keeps the selected stage's panel under the name typed for Save As…: the person's view of that name is
// overwritten, a new one is added after theirs, and a stage's own name is refused, as those never change. What it
// did, or why not, is in place of the keys.
func (m *Model) savePreset() tea.Cmd {
	m.saveAs(strings.TrimSpace(m.stages[m.stage].Naming))
	return nil
}

// saveAs keeps the selected stage's panel under a name, as table.Views saves it; the line then names it, and the
// notice says so, for a while. A name refused, or a file that could not be written, says why.
func (m *Model) saveAs(name string) {
	s := &m.stages[m.stage]
	saved, err := m.viewsOf(m.stage).SaveAs(name, s.View)
	switch {
	case errors.Is(err, table.ErrUnnamed):
		m.note.Text, m.note.Done = Failed+" a preset needs a name: type one, then ⏎", false
		return
	case errors.Is(err, table.ErrBuiltin):
		m.note.Text, m.note.Done = Failed+" "+name+" is the stage's own and does not change: type a new name, then ⏎", false
		return
	case !m.keep(saved, "saved"):
		return
	}
	s.ViewName, s.SavingAs, s.Naming, s.Renaming = name, false, "", ""
	m.done(Save + " saved the preset " + name)
}

// renamePreset gives the person's view being renamed, which stays while the line has the keyboard, the name typed for
// Rename…, as table.Views renames it: none, a stage's own, and another of theirs are refused, and the same name leaves
// it as it is. The line names it by its new name.
func (m *Model) renamePreset() {
	s := &m.stages[m.stage]
	from, name := s.Renaming, strings.TrimSpace(s.Naming)
	saved, err := m.viewsOf(m.stage).Rename(from, name)
	switch {
	case errors.Is(err, table.ErrUnnamed):
		m.note.Text, m.note.Done = Failed+" a preset needs a name: type one, then ⏎", false
		return
	case errors.Is(err, table.ErrBuiltin):
		m.note.Text, m.note.Done = Failed+" "+name+" is the stage's own: type a new name, then ⏎", false
		return
	case errors.Is(err, table.ErrTaken):
		m.note.Text, m.note.Done = Failed+" "+name+" is a preset already: type a new name, then ⏎", false
		return
	case name == from:
		s.SavingAs, s.Naming, s.Renaming = false, "", ""
		return
	case !m.keep(saved, "renamed"):
		return
	}
	if s.ViewName == from {
		s.ViewName = name
	}
	s.SavingAs, s.Naming, s.Renaming = false, "", ""
	m.done(Renamed + " renamed the preset " + from + " to " + name)
}

// deletePreset drops the person's view the line was set to, as choosePreset has it only while there is one, and sets
// the stage as Default.
func (m *Model) deletePreset() {
	v, _ := m.own()
	if !m.keep(m.viewsOf(m.stage).Delete(v.Name), "deleted") {
		return
	}
	m.apply(m.viewsOf(m.stage).Chosen(Opening))
	m.done(Deleted + " deleted the preset " + v.Name)
}

// choosePreset closes the menu and takes its line k: Fields… opens the columns dialog; Save keeps the panel under the person's view the line was set to;
// Save As… gives the preset line the keyboard, for a name, that view's typed already; Rename… does too, for its new
// name; Delete drops it; a view applies. A line that does not act now does nothing.
func (m *Model) choosePreset(k int) {
	m.presetsMenu.Open = false
	if !table.PresetEnabled(m, k) {
		return
	}
	s := &m.stages[m.stage]
	v, _ := m.own()
	switch k {
	case table.ItemFields:
		m.cols = table.Columns{Open: true}
	case table.ItemSave:
		m.saveAs(v.Name)
	case table.ItemSaveAs, table.ItemRename:
		s.SavingAs, s.Naming, s.Renaming = true, v.Name, ""
		if k == table.ItemRename {
			s.Renaming = v.Name
		}
		m.onPreset, m.inRows, m.inPanel, m.stop = true, false, false, -1
	case table.ItemDelete:
		m.deletePreset()
	default:
		name := table.PresetItems(m)[k].Name
		vs := m.views(m.stage)
		m.apply(vs[slices.IndexFunc(vs, func(v table.Preset) bool { return v.Name == name })])
	}
}

// presetClick takes a click on the preset line, which then has the keyboard: on the selector, its name or ▼, it drops
// the presets down.
func (m *Model) presetClick(x int) tea.Cmd {
	m.onPreset, m.inRows, m.inPanel, m.stop = true, false, false, -1
	if x >= m.dropStart() && x < m.dropEnd() {
		m.presetsMenu = m.presetsMenu.Drop(m)
	}
	return nil
}
