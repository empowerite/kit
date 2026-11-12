// The preset line: at the right end of the control line, a selector that names the view the table is set as, and drops
// the presets' menu down: what can be done with views, and the views to choose from. 📇 Fields stands before it, which
// opens the columns dialog, and chips before that say what the panel's filters hold back, each opening its field's card
// at its filter. What a line of the menu does is the screen's, which says what it did, and keeps the person's views
// where it keeps them.
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
// down. FieldsButton, after its glyph, Fields, stands before the selector, and opens the columns dialog. The presets'
// menu's items are SaveIt, after its glyph, Save, SaveAs, and ManageIt, which opens the presets dialog, whose RenameIt
// and DeleteIt stand after Renamed and Deleted; Save, Renamed, and Deleted start what each did says. SavingAsWord and RenamingWord stand, dim, before the name typed for
// Save As… and Rename…. Checked marks the view the table is set as. Starred stands at the right of a view the person
// starred, Unstarred at the right of the others, and Starring starts what starring one says.
const (
	Starred      = "★"
	Unstarred    = "☆"
	Starring     = "⭐"
	Presets      = "▼"
	FieldsButton = Fields + " Fields"
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

// The presets' menu's lines: those that act on views, before a rule and the views themselves.
const (
	ItemSave = iota
	ItemSaveAs
	ItemManage
	ItemViews // the rule after them; the views follow
)

// PresetsTitle is what the presets' menu's top edge says, over the preset line's selector it hangs from.
const PresetsTitle = "Presets"

// SelectorFocus is the color the preset line's selector is drawn on while the line has the keyboard: the menus' bright
// cyan.
const SelectorFocus = "14"

// PresetMenu is the presets' menu while it is open: a dropdown.
type PresetMenu struct{ shell.Dropdown }

// The headings of the presets' menu's sections, dim: the person's favorites, the team's views, the table's own, and
// the person's.
const (
	FavoritesHeading = "favorites"
	TeamHeading      = "team"
	BuiltinHeading   = "built in"
	MineHeading      = "mine"
)

// presetLine is a line of the presets' menu: what it draws, and the view it names, none for an action, a rule, or a
// heading.
type presetLine struct {
	item shell.Item
	view string
}

// presetLines gives the presets' menu's lines: Save after its glyph, Save As…, and Manage…, a rule, then the views in sections, each under its heading where it has a view: the person's favorites, in
// their order; the team's; the table's own views; and the person's. A view stands once, in favorites where it is one; the one the
// table is set as is checked; and each has Starred or Unstarred at its right, as the person starred it or not.
func presetLines(h Host) []presetLine {
	lines := []presetLine{{item: shell.Item{Glyph: Save, Name: SaveIt}}, {item: shell.Item{Name: SaveAs}},
		{item: shell.Item{Name: ManageIt}}, {}}
	v := h.Views()
	is, favorites := v.Is(*h.Table()), v.Favorites()
	section := func(heading string, names []string) {
		if len(names) == 0 {
			return
		}
		lines = append(lines, presetLine{item: shell.Item{Name: heading}})
		for _, name := range names {
			glyph, star := "", Unstarred
			if name == is {
				glyph = Checked
			}
			if slices.Contains(favorites, name) {
				star = Starred
			}
			lines = append(lines, presetLine{item: shell.Item{Glyph: glyph, Name: name, Keys: star}, view: name})
		}
	}
	others := func(ps []Preset) []string {
		var out []string
		for _, p := range ps {
			if !slices.Contains(favorites, p.Name) {
				out = append(out, p.Name)
			}
		}
		return out
	}
	section(FavoritesHeading, favorites)
	section(TeamHeading, others(v.Team))
	section(BuiltinHeading, others(v.Own))
	section(MineHeading, others(v.Saved))
	return lines
}

// PresetItems gives the presets' menu's lines as the menu draws them, as presetLines says.
func PresetItems(h Host) []shell.Item {
	var items []shell.Item
	for _, l := range presetLines(h) {
		items = append(items, l.item)
	}
	return items
}

// PresetView gives the view the presets' menu's line k names, and whether it names one.
func PresetView(h Host, k int) (string, bool) {
	lines := presetLines(h)
	if k < 0 || k >= len(lines) || lines[k].view == "" {
		return "", false
	}
	return lines[k].view, true
}

// PresetEnabled says whether a line of the presets' menu acts: Save while the line was set to a view of the person's
// own, or of the team's, that the panel no longer is; Save As…, Manage…, and every view always; a rule or a heading
// never.
func PresetEnabled(h Host, k int) bool {
	v := h.Views()
	mine, own := v.Mine(h.Table().ViewName)
	if team, ok := v.Shared(h.Table().ViewName); !own {
		mine, own = team, ok
	}
	switch k {
	case ItemSave:
		return own && !v.Same(v.Current(h.Table().View), mine)
	case ItemSaveAs, ItemManage:
		return true
	}
	_, ok := PresetView(h, k)
	return ok
}

// Drop gives the presets' menu dropped down, the view the table is set as under the keyboard, else Save As….
func (pm PresetMenu) Drop(h Host) PresetMenu {
	at := slices.IndexFunc(PresetItems(h), func(it shell.Item) bool { return it.Glyph == Checked })
	if at < 0 {
		at = ItemSaveAs
	}
	return PresetMenu{shell.Dropdown{Open: true, At: at}}
}

// Key takes a key while the menu is open, as a dropdown does, and gives the line chosen, -1 for none; and the line whose
// view * stars, or no longer, -1 for none.
func (pm *PresetMenu) Key(h Host, k tea.KeyPressMsg) (chosen, star int) {
	if k.String() == "*" {
		if _, ok := PresetView(h, pm.At); ok {
			return -1, pm.At
		}
		return -1, -1
	}
	pm.Dropdown, chosen = pm.Dropdown.Key(k, PresetItems(h), func(k int) bool { return PresetEnabled(h, k) })
	return chosen, -1
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

// Click takes a click while the menu is open, as a dropdown does, and gives the line chosen, -1 for none: on a view's
// star, the line whose view it stars, or no longer, the menu left open; on a line that acts, it; on another line of the
// box, nothing; anywhere else, the menu closes.
func (pm *PresetMenu) Click(h Host, x, y int) (chosen, star int) {
	box, top, left := pm.Box(h)
	if k := y - top - 1; k >= 0 && k < len(box)-2 {
		if _, ok := PresetView(h, k); ok {
			plain := ansi.Strip(box[k+1])
			if at := strings.LastIndexAny(plain, Starred+Unstarred); at >= 0 && x == left+ansi.StringWidth(plain[:at]) {
				pm.At = k
				return -1, k
			}
		}
	}
	pm.Dropdown, chosen = pm.Dropdown.Click(x, y, top, left, ansi.StringWidth(box[0]), PresetItems(h),
		func(k int) bool { return PresetEnabled(h, k) })
	return chosen, -1
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

// SelectorPill draws the preset line's selector: the preset's name, struck through and italic once the table has
// changed from it, as a terminal has no bolder strikethrough, and Presets, bold black on back, between half circles
// in back.
func SelectorPill(name, back string, changed bool) string {
	text := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("0")).Background(lipgloss.Color(back))
	return style.PillEnd(style.PillLeft, back) + text.Strikethrough(changed).Italic(changed).Render(name) +
		text.Render(" "+Presets) + style.PillEnd(style.PillRight, back)
}

// Chips gives a chip per field the table's panel filters by, in the catalog's order: its label, a colon, and the
// values it lets through, or, ¬ before each, those it holds back, in the accent.
func Chips(h Host) []string {
	_, chips := chipsOf(h)
	return chips
}

// chipsOf gives the fields the table's panel filters by, by id, and their chips, as Chips draws them.
func chipsOf(h Host) (ids, out []string) {
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
		ids, out = append(ids, f.ID), append(out, style.Tint(cmp.Or(f.Label, f.ID)+": "+strings.Join(values, ", ")))
	}
	return ids, out
}
