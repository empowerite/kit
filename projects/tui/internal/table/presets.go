// The preset line: at the right end of the control line, a selector that names the view the table is set as, and drops
// the presets' menu down: Fields…, which opens the columns dialog, what can be done with views, and the views to choose
// from. Chips before it say what the panel's filters hold back. What a line of the menu does is the screen's, which
// says what it did, and keeps the person's views where it keeps them.
package table

import (
	"cmp"
	"slices"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/empowerite/kit/projects/tui/internal/shell"
	"github.com/empowerite/kit/projects/tui/internal/style"
)

// Presets ends the preset line, after the name of the view the table is set as, and a click on it drops the presets
// down. FieldsIt is the presets' menu's first item, after its glyph, Fields, and a rule; SaveIt, SaveAs, RenameIt,
// and DeleteIt follow, Save and Delete after their glyphs, Save and Deleted; those, and Renamed, start what each did
// says. SavingAsWord and RenamingWord stand, dim, before the name typed for Save As… and Rename…. Checked marks the
// view the table is set as.
const (
	Presets      = "▼"
	FieldsIt     = "Fields…"
	SaveIt       = "Save"
	SaveAs       = "Save As…"
	RenameIt     = "Rename…"
	DeleteIt     = "Delete"
	SavingAsWord = "save as"
	RenamingWord = "rename"
	Fields       = "📇"
	Save         = "💾"
	Renamed      = "📝"
	Deleted      = "🚮"
	Checked      = "✅"
)

// The presets' menu's lines: Fields… and a rule, then those that act on views, before a rule and the views themselves.
const (
	ItemFields = iota
	_          // the rule after it
	ItemSave
	ItemSaveAs
	ItemRename
	ItemDelete
	ItemViews // the rule after them; the views follow
)

// PresetsTitle is what the presets' menu's top edge says, over the preset line's selector it hangs from.
const PresetsTitle = "Presets"

// SelectorFocus is the color the preset line's selector is drawn on while the line has the keyboard: the menus' bright
// cyan.
const SelectorFocus = "14"

// PresetMenu is the presets' menu while it is open: a dropdown.
type PresetMenu struct{ shell.Dropdown }

// PresetItems gives the presets' menu: Fields… after its glyph, a rule, Save and Delete after their glyphs, Save As…
// and Rename… between them without, a rule, the table's own views, a rule, and the person's, the one the table is set
// as checked.
func PresetItems(h Host) []shell.Item {
	items := []shell.Item{{Glyph: Fields, Name: FieldsIt}, {}, {Glyph: Save, Name: SaveIt}, {Name: SaveAs},
		{Name: RenameIt}, {Glyph: Deleted, Name: DeleteIt}, {}}
	v := h.Views()
	is := v.Is(*h.Table())
	for k, p := range v.All() {
		if k == len(v.Own) {
			items = append(items, shell.Item{})
		}
		glyph := ""
		if p.Name == is {
			glyph = Checked
		}
		items = append(items, shell.Item{Glyph: glyph, Name: p.Name})
	}
	return items
}

// PresetEnabled says whether a line of the presets' menu acts: Fields… while the table has fields to list; Save while
// the line was set to a view of the person's own that the panel no longer is; Rename… and Delete while it was set to
// one; Save As… and every view always; a rule never.
func PresetEnabled(h Host, k int) bool {
	v := h.Views()
	mine, own := v.Mine(h.Table().ViewName)
	switch k {
	case ItemFields:
		return h.Columnar()
	case ItemSave:
		return own && !v.Same(v.Current(h.Table().View), mine)
	case ItemRename, ItemDelete:
		return own
	}
	items := PresetItems(h)
	return k >= 0 && k < len(items) && items[k].Name != ""
}

// Drop gives the presets' menu dropped down, the view the table is set as under the keyboard, else Save As….
func (pm PresetMenu) Drop(h Host) PresetMenu {
	at := slices.IndexFunc(PresetItems(h), func(it shell.Item) bool { return it.Glyph == Checked })
	if at < 0 {
		at = ItemSaveAs
	}
	return PresetMenu{shell.Dropdown{Open: true, At: at}}
}

// Key takes a key while the menu is open, as a dropdown does, and gives the line chosen, -1 for none.
func (pm *PresetMenu) Key(h Host, k tea.KeyPressMsg) int {
	var chosen int
	pm.Dropdown, chosen = pm.Dropdown.Key(k, PresetItems(h), func(k int) bool { return PresetEnabled(h, k) })
	return chosen
}

// Box gives the presets' menu's lines, in its tab's color, as the selector is, titled PresetsTitle at its top edge's
// left, as a dialog is; and the screen line and column it is drawn from: the control line, the selector's, as the
// columns dialog's, its right edge on the selector's last cell, so it covers the ▼.
func (pm PresetMenu) Box(h Host) (lines []string, top, left int) {
	color := h.Color()
	width, _ := h.Size()
	_, end := h.Selector()
	lines, left = shell.BoxColored(PresetItems(h), func(k int) bool { return PresetEnabled(h, k) }, pm.At, "", -1,
		min(end, width), color)
	lines[0] = shell.TitledColored(PresetsTitle, ansi.StringWidth(lines[0]), color)
	top, _ = h.Pane()
	return lines, top, left
}

// Click takes a click while the menu is open, as a dropdown does, and gives the line chosen, -1 for none: on a line
// that acts, it; on another line of the box, nothing; anywhere else, the menu closes.
func (pm *PresetMenu) Click(h Host, x, y int) int {
	box, top, left := pm.Box(h)
	var chosen int
	pm.Dropdown, chosen = pm.Dropdown.Click(x, y, top, left, ansi.StringWidth(box[0]), PresetItems(h),
		func(k int) bool { return PresetEnabled(h, k) })
	return chosen
}

// PresetLine gives the preset's selector, at the right end of the control line: the name of the preset the table is
// set as, and Presets; once the table differs from every preset, the name of the one it was last set to, struck
// through, as a hint it has left it; or, while a name is typed for Save As… or Rename…, the name with a caret after a dim "save
// as" or "rename", while the line has the keyboard, as focused says. The selector is a pill, as a label is: bold black
// on its tab's color, the preset being the tab's, between half circles in that color; on the menus' bright cyan while
// the line has the keyboard, as the line under the keyboard in a menu is.
func PresetLine(h Host, focused bool) string {
	t := h.Table()
	if focused && t.SavingAs {
		label := SavingAsWord
		if t.Renaming != "" {
			label = RenamingWord
		}
		return style.Off(label) + " " + t.Naming + "▏"
	}
	back := h.Color()
	if focused {
		back = SelectorFocus
	}
	v := h.Views()
	if is := v.Is(*t); is != "" {
		return SelectorPill(is, back, false)
	}
	return SelectorPill(v.Chosen(t.ViewName).Name, back, true)
}

// SelectorPill draws the preset line's selector: the preset's name, struck through once the table has changed from it,
// and Presets, bold black on back, between half circles in back.
func SelectorPill(name, back string, changed bool) string {
	text := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("0")).Background(lipgloss.Color(back))
	return style.PillEnd(style.PillLeft, back) + text.Strikethrough(changed).Render(name) + text.Render(" "+Presets) +
		style.PillEnd(style.PillRight, back)
}

// Chips gives a chip per field the table's panel filters by, in the catalog's order: its label, a colon, and the
// values it lets through, or, ¬ before each, those it holds back, in the accent.
func Chips(h Host) []string {
	var out []string
	for _, f := range h.Fields() {
		filter := h.Table().View.Field(f.ID).Filter
		if len(filter.Values) == 0 {
			continue
		}
		values := slices.Clone(filter.Values)
		if filter.Exclude {
			for k := range values {
				values[k] = "¬" + values[k]
			}
		}
		out = append(out, style.Tint(cmp.Or(f.Label, f.ID)+": "+strings.Join(values, ", ")))
	}
	return out
}
