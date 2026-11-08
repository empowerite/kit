// Named views: a stage's panel as it is set, each field's placement, sort, filter, and rendering, and whether it folds,
// kept under a name as a preset. Every stage with columns has its own views, the presets the field catalog gives it,
// which never change: Default, the stage as it opens, and Compact, the few columns that say what a row is. The person
// saves others, per stage, in a file of their own, presets.json under ~/.config/kit/, so their views follow them to
// every repository; one they named as a stage's own was named before the stage had it, and stays theirs. The preset
// line, the rule under the stage's header, ends in the name of the view the table is set as and ▼, the name in italics
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
	"github.com/charmbracelet/x/ansi"

	"github.com/empowerite/kit/projects/tui/internal/grid"
	"github.com/empowerite/kit/projects/tui/internal/shell"
	"github.com/empowerite/kit/projects/tui/internal/table"
)

// Opening is the name of the view a stage opens on, one of its own.
const Opening = table.Opening

// Presets ends the preset line, after the name of the view the table is set as, and a click on it drops the presets
// down. FieldsIt is the presets' menu's first item, after its glyph, Fields, and a rule; SaveIt, SaveAs,
// RenameIt, and DeleteIt follow, Save and Delete after their glyphs, Save and Deleted; those, and Renamed, start what each did says. SavingAs and Renaming stand, dim, before the name
// typed for Save As… and Rename….
const (
	Presets  = "▼"
	FieldsIt = "Fields…"
	SaveIt   = "Save"
	SaveAs   = "Save As…"
	RenameIt = "Rename…"
	DeleteIt = "Delete"
	SavingAs = "save as"
	Renaming = "rename"
	Save     = "💾"
	Renamed  = "📝"
	Deleted  = "🚮"
)

// The presets' menu's lines: Fields… and a rule, then those that act on views, before a rule and the views themselves.
const (
	itemFields = iota
	_          // the rule after it
	itemSave
	itemSaveAs
	itemRename
	itemDelete
	itemViews // the rule after them; the views follow
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
	p := grid.Panel{Groupable: groupableOf(stage), Sortable: sortableOf(stage), Owner: ownerOf(stage),
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

// presetKey takes a key while the keyboard is on the preset line, and says whether it did: up and esc give the bar the
// keyboard, and down the rows, when the stage has some; enter and space drop the presets down. While a name is typed
// for Save As… or Rename…, what is typed is the name, = : and ; apart, which the menus take; enter saves under it, or
// renames, esc stops, and up and down stop too, and move as ever.
func (m *Model) presetKey(k tea.KeyPressMsg) (tea.Cmd, bool) {
	s := &m.stages[m.stage]
	key := k.String()
	if s.SavingAs {
		switch {
		case key == "up" || key == "down":
			s.SavingAs, s.Naming, s.Renaming = false, "", ""
		case key == "esc":
			s.SavingAs, s.Naming, s.Renaming = false, "", ""
			return nil, true
		case key == "enter" && s.Renaming != "":
			m.renamePreset()
			return nil, true
		case key == "enter":
			return m.savePreset(), true
		case key == "backspace" || k.Text != "" && !strings.ContainsAny(k.Text, "=:;"):
			s.Naming, _ = shell.Type(s.Naming, k)
			return nil, true
		default:
			return nil, false
		}
	}
	switch key {
	case "esc", "up":
		m.onPreset = false
	case "down":
		if len(m.rows(m.stage)) > 0 {
			m.onPreset = false
			m.down(1)
		}
	case "enter", "space":
		m.dropPresets()
	default:
		return nil, false
	}
	return nil, true
}

// dropPresets drops the selected stage's presets down, the one the line names under the keyboard, else Save As….
func (m *Model) dropPresets() {
	at := slices.IndexFunc(m.presetItems(), func(it shell.Item) bool { return it.Glyph == Checked })
	if at < 0 {
		at = itemSaveAs
	}
	m.presetsMenu = shell.Dropdown{Open: true, At: at}
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

// presetItems gives the presets' menu: Fields… after its glyph, a rule, Save and Delete after their
// glyphs, Save As… and Rename… between them without, a rule, the stage's own views, a rule, and the person's, the one the line names checked.
func (m Model) presetItems() []shell.Item {
	items := []shell.Item{{Glyph: Fields, Name: FieldsIt}, {}, {Glyph: Save, Name: SaveIt}, {Name: SaveAs},
		{Name: RenameIt}, {Glyph: Deleted, Name: DeleteIt}, {}}
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

// presetEnabled says whether a line of the presets' menu acts: Fields… while the stage has fields to list; Save while the line was set to a view of the person's
// own that the panel no longer is; Rename… and Delete while it was set to one; Save As… and every view always; a rule
// never.
func (m Model) presetEnabled(k int) bool {
	v, own := m.own()
	switch k {
	case itemFields:
		return m.columnar(m.stage)
	case itemSave:
		return own && !m.same(m.stage, m.current(m.stage), v)
	case itemRename, itemDelete:
		return own
	}
	return m.presetItems()[k].Name != ""
}

// presetMenuKey takes a key while the presets' menu is open, as a dropdown does, and takes the line chosen.
func (m Model) presetMenuKey(k tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	var chosen int
	if m.presetsMenu, chosen = m.presetsMenu.Key(k, m.presetItems(), m.presetEnabled); chosen >= 0 {
		m.choosePreset(chosen)
	}
	return m, nil
}

// choosePreset closes the menu and takes its line k: Fields… opens the columns dialog; Save keeps the panel under the person's view the line was set to;
// Save As… gives the preset line the keyboard, for a name, that view's typed already; Rename… does too, for its new
// name; Delete drops it; a view applies. A line that does not act now does nothing.
func (m *Model) choosePreset(k int) {
	m.presetsMenu.Open = false
	if !m.presetEnabled(k) {
		return
	}
	s := &m.stages[m.stage]
	v, _ := m.own()
	switch k {
	case itemFields:
		m.cols = table.Columns{Open: true}
	case itemSave:
		m.saveAs(v.Name)
	case itemSaveAs, itemRename:
		s.SavingAs, s.Naming, s.Renaming = true, v.Name, ""
		if k == itemRename {
			s.Renaming = v.Name
		}
		m.onPreset, m.inRows, m.inPanel = true, false, false
	case itemDelete:
		m.deletePreset()
	default:
		name := m.presetItems()[k].Name
		vs := m.views(m.stage)
		m.apply(vs[slices.IndexFunc(vs, func(v table.Preset) bool { return v.Name == name })])
	}
}

// PresetsTitle is what the presets' menu's top edge says, over the preset line's selector it hangs from.
const PresetsTitle = "Presets"

// presetBox gives the presets' menu's lines, in its tab's color, as the selector is, titled PresetsTitle at its top
// edge's left, as a dialog is; and the screen line and column it is drawn from: the control line, the selector's, as
// the columns dialog's, its right edge on the selector's last cell, so it covers the ▼.
func (m Model) presetBox() (lines []string, top, left int) {
	color := stageColors[m.stages[m.stage].name]
	lines, left = shell.BoxColored(m.presetItems(), m.presetEnabled, m.presetsMenu.At, "", -1, min(m.dropEnd(), m.width),
		color)
	lines[0] = shell.TitledColored(PresetsTitle, ansi.StringWidth(lines[0]), color)
	return lines, m.top(), left
}

// presetMenuClick takes a click while the presets' menu is open, as a dropdown does: on a line that acts, it chooses
// it; on another line of the box, nothing; anywhere else, it closes the menu.
func (m *Model) presetMenuClick(x, y int) {
	box, top, left := m.presetBox()
	var chosen int
	if m.presetsMenu, chosen = m.presetsMenu.Click(x, y, top, left, ansi.StringWidth(box[0]), m.presetItems(),
		m.presetEnabled); chosen >= 0 {
		m.choosePreset(chosen)
	}
}

// presetClick takes a click on the preset line, which then has the keyboard: on the selector, its name or ▼, it drops
// the presets down.
func (m *Model) presetClick(x int) tea.Cmd {
	m.onPreset, m.inRows, m.inPanel = true, false, false
	if x >= m.dropStart() && x < m.dropEnd() {
		m.dropPresets()
	}
	return nil
}
