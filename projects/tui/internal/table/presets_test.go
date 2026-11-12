// The preset line names the view the table is set as, and its menu lists what can be done with views, then the views.
package table

import (
	"regexp"
	"slices"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/empowerite/kit/projects/tui/internal/grid"
)

// presetting gives a host whose table has the views of views, set as Default, its columns loaded.
func presetting() *host {
	h := listing()
	h.views = views()
	h.tbl = Table{ViewName: Default, View: h.views.Chosen(Default).Panel(h.views.Fields, grid.Panel{})}
	h.fields = h.views.Fields
	h.loaded = true
	return h
}

// itemNames gives the presets' menu's lines' names.
func itemNames(h Host) []string {
	var out []string
	for _, it := range PresetItems(h) {
		out = append(out, it.Name)
	}
	return out
}

func TestThePresetsMenuListsWhatCanBeDoneWithViewsThenTheViews(t *testing.T) {
	h := presetting()
	want := []string{FieldsIt, "", SaveIt, SaveAs, RenameIt, DeleteIt, "", Default, "Grouped", "", "Mine"}
	if got := itemNames(h); !slices.Equal(got, want) || PresetItems(h)[7].Glyph != Checked {
		t.Fatalf("the menu: %q", got)
	}
	// set as Default, a table's own: Save, Rename…, and Delete are dim; Fields… and Save As… act
	for k, acts := range map[int]bool{ItemFields: true, 1: false, ItemSave: false, ItemSaveAs: true, ItemRename: false,
		ItemDelete: false, 7: true, -1: false, 99: false} {
		if got := PresetEnabled(h, k); got != acts {
			t.Errorf("line %d acts %v", k, got)
		}
	}
	// set as Mine, then changed: Save, Rename…, and Delete act
	h.tbl.ViewName = "Mine"
	h.tbl.View = h.tbl.View.Placed("state", grid.Hidden)
	if !PresetEnabled(h, ItemSave) || !PresetEnabled(h, ItemRename) || !PresetEnabled(h, ItemDelete) {
		t.Error("Mine changed")
	}
	h.loaded = false
	if PresetEnabled(h, ItemFields) {
		t.Error("Fields… with no columns loaded")
	}
}

func TestThePresetsMenuDropsFromTheSelectorAndTakesKeysAndClicks(t *testing.T) {
	h := presetting()
	pm := PresetMenu{}.Drop(h)
	if !pm.Open || pm.At != 7 {
		t.Fatalf("dropped at %d", pm.At)
	}
	// set as none of them, it drops at Save As…
	h.tbl.View = h.tbl.View.Placed("state", grid.Hidden).Placed("title", grid.Hidden)
	if pm := (PresetMenu{}).Drop(h); pm.At != ItemSaveAs {
		t.Errorf("as none: at %d", pm.At)
	}
	box, top, left := pm.Box(h)
	if top != 4 || left+ansi.StringWidth(box[0]) != 70 || !strings.Contains(ansi.Strip(box[0]), PresetsTitle) {
		t.Errorf("the box at %d, %d: %q", top, left, ansi.Strip(box[0]))
	}
	if chosen := pm.Key(h, key("down")); chosen != -1 || pm.At != 8 {
		t.Errorf("down: %d, at %d", chosen, pm.At)
	}
	if chosen := pm.Key(h, tea.KeyPressMsg{Code: tea.KeyEnter}); chosen != 8 {
		t.Errorf("enter: %d", chosen)
	}
	pm = PresetMenu{}.Drop(h)
	if chosen := pm.Click(h, left+3, top+1+ItemSaveAs); chosen != ItemSaveAs {
		t.Errorf("a click on Save As…: %d", chosen)
	}
	if chosen := pm.Click(h, 0, 0); chosen != -1 || pm.Open {
		t.Errorf("a click off it: %d, open %v", chosen, pm.Open)
	}
}

// A preset the table has changed from is struck through and italic, every letter of its name, as a terminal has no
// bolder strikethrough; one it is set as is neither, and Presets after the name never is.
func TestAChangedPresetIsStruckThroughAndItalic(t *testing.T) {
	changed, set := SelectorPill("Grouped", "75", true), SelectorPill("Grouped", "75", false)
	for _, letter := range "Grouped" {
		if got := sgrBefore(changed, string(letter)); !got["3"] || !got["9"] {
			t.Errorf("%c is not struck through and italic: %q", letter, changed)
		}
		if got := sgrBefore(set, string(letter)); got["3"] || got["9"] {
			t.Errorf("set as, %c is struck or italic: %q", letter, set)
		}
	}
	if got := sgrBefore(changed, " "+Presets); got["3"] || got["9"] {
		t.Errorf("Presets struck or italic: %q", changed)
	}
}

// sgrBefore gives the parameters of the SGR sequence just before the first text given in s.
func sgrBefore(s, text string) map[string]bool {
	m := regexp.MustCompile(`\x1b\[([0-9;]*)m` + regexp.QuoteMeta(text)).FindStringSubmatch(s)
	out := map[string]bool{}
	if m != nil {
		for _, p := range strings.Split(m[1], ";") {
			out[p] = true
		}
	}
	return out
}

func TestThePresetLineNamesTheViewTheTableIsSetAs(t *testing.T) {
	h := presetting()
	if got := PresetLine(h, false); got != SelectorPill(Default, "75", false) {
		t.Errorf("as Default: %q", ansi.Strip(got))
	}
	if got := PresetLine(h, true); got != SelectorPill(Default, SelectorFocus, false) {
		t.Errorf("with the keyboard: %q", ansi.Strip(got))
	}
	// changed from every view, the one it was set to, struck through
	h.tbl.View = h.tbl.View.Placed("state", grid.Hidden).Placed("title", grid.Hidden)
	if got := PresetLine(h, false); got != SelectorPill(Default, "75", true) {
		t.Errorf("changed: %q", ansi.Strip(got))
	}
	// a name typed for Save As… or Rename…, once the line has the keyboard
	h.tbl.SavingAs, h.tbl.Naming = true, "Triage"
	if got := ansi.Strip(PresetLine(h, true)); got != SavingAsWord+" Triage▏" {
		t.Errorf("saving as: %q", got)
	}
	h.tbl.Renaming = "Mine"
	if got := ansi.Strip(PresetLine(h, true)); got != RenamingWord+" Triage▏" {
		t.Errorf("renaming: %q", got)
	}
}

func TestAChipSaysWhatAFilterHoldsBack(t *testing.T) {
	h := presetting()
	h.tbl.View = h.tbl.View.FilteredBy("state", grid.Filter{Values: []string{"open"}})
	if got := Chips(h); len(got) != 1 || ansi.Strip(got[0]) != "state: open" {
		t.Errorf("include: %q", got)
	}
	h.tbl.View = h.tbl.View.FilteredBy("state", grid.Filter{Values: []string{"open"}, Exclude: true})
	if got := Chips(h); ansi.Strip(got[0]) != "state: ¬open" {
		t.Errorf("exclude: %q", got)
	}
}
