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

// presetting gives a host whose table has the views of views, set as Default.
func presetting() *host {
	h := listing()
	h.views = views()
	h.tbl = Table{ViewName: Default, View: h.views.Chosen(Default).Panel(h.views.Fields, grid.Panel{})}
	h.fields = h.views.Fields
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
	want := []string{SaveIt, SaveAs, ManageIt, "", BuiltinHeading, Default, "Grouped", MineHeading, "Mine"}
	if got := itemNames(h); !slices.Equal(got, want) || PresetItems(h)[5].Glyph != Checked ||
		PresetItems(h)[5].Keys != Unstarred {
		t.Fatalf("the menu: %q", got)
	}
	// set as Default, a table's own: Save is dim; Save As… and Manage… act; a heading never does
	for k, acts := range map[int]bool{ItemViews: false, ItemSave: false, ItemSaveAs: true, ItemManage: true,
		4: false, 5: true, -1: false, 99: false} {
		if got := PresetEnabled(h, k); got != acts {
			t.Errorf("line %d acts %v", k, got)
		}
	}
	// set as Mine, then changed: Save, Rename…, and Delete act
	h.tbl.ViewName = "Mine"
	h.tbl.View = h.tbl.View.Placed("state", grid.Hidden)
	if !PresetEnabled(h, ItemSave) {
		t.Error("Mine changed")
	}
}

func TestThePresetsMenuDropsFromTheSelectorAndTakesKeysAndClicks(t *testing.T) {
	h := presetting()
	pm := PresetMenu{}.Drop(h)
	if !pm.Open || pm.At != 5 {
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
	if chosen, _ := pm.Key(h, key("down")); chosen != -1 || pm.At != 6 {
		t.Errorf("down: %d, at %d", chosen, pm.At)
	}
	if chosen, _ := pm.Key(h, tea.KeyPressMsg{Code: tea.KeyEnter}); chosen != 6 {
		t.Errorf("enter: %d", chosen)
	}
	pm = PresetMenu{}.Drop(h)
	if chosen, star := pm.Click(h, left+3, top+1+ItemSaveAs); chosen != ItemSaveAs || star != -1 {
		t.Errorf("a click on Save As…: %d", chosen)
	}
	if chosen, _ := pm.Click(h, 0, 0); chosen != -1 || pm.Open {
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

// A view the person starred stands first, under favorites, in their order, ★ at its right, and in no other section; a
// favorite whose view is gone stays hidden; a section with no view has no heading. * on a view's line, or a click on its
// star, gives the line to star, and leaves the menu open; * elsewhere gives none.
func TestTheViewsThePersonStarredStandFirst(t *testing.T) {
	h := presetting()
	h.views.Starred = []string{"Mine", "gone", "Mine"}
	want := []string{SaveIt, SaveAs, ManageIt, "", FavoritesHeading, "Mine", BuiltinHeading, Default, "Grouped"}
	if got := itemNames(h); !slices.Equal(got, want) || PresetItems(h)[5].Keys != Starred ||
		PresetItems(h)[7].Keys != Unstarred || !slices.Equal(h.views.Favorites(), []string{"Mine"}) {
		t.Fatalf("the menu: %q, favorites %q", got, h.views.Favorites())
	}
	if v := h.views; !slices.Equal(v.Star(Default), []string{"Mine", Default}) || len(v.Star("Mine")) != 0 ||
		!slices.Equal(v.Renamed("Mine", "Ours"), []string{"Ours", "gone", "Ours"}) || !v.Favorite("Mine") ||
		v.Favorite("gone") {
		t.Errorf("starring: %q, %q, %q", v.Star(Default), v.Star("Mine"), v.Renamed("Mine", "Ours"))
	}
	pm := PresetMenu{}.Drop(h)
	if chosen, star := pm.Key(h, tea.KeyPressMsg{Code: '*', Text: "*"}); chosen != -1 || star != 7 {
		t.Errorf("* on Default: %d, %d", chosen, star)
	}
	pm.At = 4
	if chosen, star := pm.Key(h, tea.KeyPressMsg{Code: '*', Text: "*"}); chosen != -1 || star != -1 {
		t.Errorf("* on a heading: %d, %d", chosen, star)
	}
	box, top, left := pm.Box(h)
	plain := ansi.Strip(box[1+5])
	at := left + ansi.StringWidth(plain[:strings.LastIndex(plain, Starred)])
	if chosen, star := pm.Click(h, at, top+1+5); chosen != -1 || star != 5 || !pm.Open || pm.At != 5 {
		t.Errorf("a click on Mine's star: %d, %d", chosen, star)
	}
	if chosen, star := pm.Click(h, at-2, top+1+5); chosen != 5 || star != -1 {
		t.Errorf("a click on Mine's name: %d, %d", chosen, star)
	}
	if _, ok := PresetView(h, 99); ok {
		t.Error("a view past the menu")
	}
}

// The control line's right part, where it has a selector: a chip per filter, then 📇 Fields, then the selector; a click
// on a chip opens its field's card under it, on its filter, and one on 📇 Fields the columns dialog. Where the line is
// short, the chips are clipped first, a chip clipped reaching as far as it shows.
func TestTheControlLinesChipsAndFieldsButtonAreTargets(t *testing.T) {
	h := presetting()
	h.tbl.View = h.tbl.View.FilteredBy("state", grid.Filter{Values: []string{"open"}}).
		FilteredBy("title", grid.Filter{Values: []string{"fix"}})
	tail := SelectorPill(Default, "4", false)
	line, at := Control(h, "  left", tail)
	spots := ControlSpots(h, "  left", tail)
	cut := func(s ControlSpot) string { return ansi.Strip(ansi.Cut(line, s.From, s.Until)) }
	if len(spots) != 3 || spots[0].Field != "state" || spots[1].Field != "title" || spots[2].Field != "" ||
		cut(spots[0]) != "state: open" || cut(spots[1]) != "title: fix" || cut(spots[2]) != FieldsButton ||
		at != spots[2].Until+2 {
		t.Fatalf("the spots: %+v, at %d, in %q", spots, at, ansi.Strip(line))
	}
	if s, ok := SpotAt(spots, spots[1].Until-1); !ok || s.Field != "title" {
		t.Errorf("a click on title's chip: %+v", s)
	}
	if _, ok := SpotAt(spots, spots[0].Until); ok {
		t.Error("a click between the chips reached one")
	}
	// a chip's card hangs from it, its title over it, on the filter's mode, with every setting of its field
	p := OpenChip(h, spots[0])
	if lines := p.Lines(h); !p.Open || !p.Chip || p.X != spots[0].From || p.Y != 4 || !lines[p.At].Mode ||
		lines[len(lines)-1].Act == nil {
		t.Errorf("state's chip's card: %+v", p)
	}
	h.h = 40
	if _, top, left := p.Box(h); top != 4 || left != spots[0].From-3 {
		t.Errorf("the chip's card at %d, %d", top, left)
	}
	// short of width, the chips are clipped: state's shows three cells, title's none, and 📇 Fields stays
	h.w = 36
	line, _ = Control(h, "  left", tail)
	if spots = ControlSpots(h, "  left", tail); len(spots) != 2 || spots[0].Until-spots[0].From != 3 ||
		cut(spots[1]) != FieldsButton {
		t.Errorf("clipped: %+v in %q", spots, ansi.Strip(line))
	}
	// without a selector, there is no right part to click
	if ControlSpots(h, "  left", "") != nil {
		t.Error("spots without a selector")
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
