// A sheet, one table on a screen: built from its records, drawn in its pane with the control line over it, its keys
// and clicks, its layers, and the person's presets of it kept in the file beside every other table's.
package table

import (
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/empowerite/kit/projects/tui/internal/grid"
	"github.com/empowerite/kit/projects/tui/internal/shell"
	"github.com/empowerite/kit/projects/tui/internal/testkit"
)

// things is a catalog's table of things: a name that sorts, a note that wraps, a kind they group and filter by, and
// tags, a list, which Default hides; Default groups them by kind, sorted by name, and Compact hides the note.
var things = &Schema{Stage: "things", Fields: []Field{
	{ID: "name", Label: "name", Type: "text", Sort: true, Renderings: []string{"text"}, Placement: PlaceColumn},
	{ID: "note", Label: "note", Type: "text", Renderings: []string{"text"}, Wrap: true, Placement: PlaceColumn},
	{ID: "kind", Label: "kind", Type: "enum", Sort: true, Group: true, Filter: true, Renderings: []string{"text"},
		Dim: true, Placement: PlaceColumn},
	{ID: "tags", Label: "tags", Type: "text", One: "tag", Renderings: []string{"text"}, Placement: PlaceColumn},
}, Presets: []Preset{
	{Name: Default, Fields: map[string]Setting{"kind": {Placement: PlaceGroup, Depth: 1},
		"name": {Sort: "asc", Priority: 1}, "tags": {Placement: PlaceHidden}}},
	{Name: "Compact", Fields: map[string]Setting{"note": {Placement: PlaceHidden}}},
}}

// thing gives a record of a thing, its tags a list.
func thing(name, kind, note string, tags ...string) Record {
	return Record{Row: grid.Row{Key: name}, Values: map[string]Value{"name": One(name), "kind": One(kind),
		"note": One(note), "tags": {Raw: tags}}}
}

// stock is five things of two kinds, kale's note long enough to wrap when its row opens.
func stock() []Record {
	return []Record{thing("pear", "fruit", "a pear, sweet and soft", "sweet"),
		thing("leek", "vegetable", "a leek"), thing("apple", "fruit", "an apple, crisp", "sweet", "crisp"),
		thing("kale", "vegetable", "a kale, curly and green, and long enough to wrap over the end of its line"),
		thing("fig", "fruit", "a fig")}
}

// sheet gives a sheet of the five things, placed from line 4, 16 lines tall, on a screen 80 by 24, its presets kept
// in the file at path.
func sheet(path string) Sheet {
	s := NewSheet(things, "things", "75", path)
	s.Place(4, 16, 80, 24)
	s.SetRecords(stock())
	return s
}

// screenOf gives the sheet's pane without its escapes, one line each, from the screen's line 0, its layers drawn
// over it.
func screenOf(s Sheet) []string {
	lines := append(make([]string, s.top), s.View()...)
	lines = s.Over(append(lines, make([]string, 4)...))
	for k, l := range lines {
		lines[k] = strings.TrimRight(ansi.Strip(l), " ")
	}
	return lines
}

// spot gives the cell and the line the text first starts on in the sheet's screen, or -1, -1.
func spot(s Sheet, text string) (x, y int) {
	for y, l := range screenOf(s) {
		if k := strings.Index(l, text); k >= 0 {
			return ansi.StringWidth(l[:k]), y
		}
	}
	return -1, -1
}

// pressed sends the key presses, by name, and gives whether the sheet took each.
func pressed(s *Sheet, names ...string) []bool {
	var took []bool
	for _, n := range names {
		took = append(took, s.Key(testkit.Press(n)))
	}
	return took
}

// clickOn clicks the text's first cell, past it by dx, and says whether the sheet took the click.
func clickOn(t *testing.T, s *Sheet, text string, dx int) bool {
	t.Helper()
	x, y := spot(*s, text)
	if y < 0 {
		t.Fatalf("no %q on the screen:\n%s", text, strings.Join(screenOf(*s), "\n"))
	}
	return s.Click(testkit.Click(x+dx, y))
}

func TestASheetDrawsItsTableUnderTheControlLine(t *testing.T) {
	s := sheet("")
	got := strings.Join(screenOf(s), "\n")
	for _, want := range []string{"▾ kind ▾ 5", "Default ▼", "name   note", "▾ fruit 3", "apple  an apple, crisp",
		"▾ vegetable 2", "leek   a leek"} {
		if !strings.Contains(got, want) {
			t.Errorf("no %q in:\n%s", want, got)
		}
	}
	if _, y := spot(s, "apple"); y >= 0 {
		if _, fig := spot(s, "fig"); fig != y+1 {
			t.Errorf("the rows sort by name: apple on %d, fig on %d", y, fig)
		}
	}
	if !s.Controls() || !s.Columnar() || !s.HasPresets() || s.Color() != "75" || s.Name() != "things" || s.Me() != "" {
		t.Error("what the sheet says of itself")
	}
	if w, h := s.Size(); w != 80 || h != 24 || s.PaneHeight() != 16 || len(s.Fields()) != 4 ||
		len(s.Values("name")) != 5 {
		t.Error("its size and its fields")
	}
	if top, w := s.Pane(); top != 4 || w != 79 || s.HeaderTop() != 6 || s.Drawn("name") != "text" {
		t.Errorf("its pane: %d %d, header from %d", top, w, s.HeaderTop())
	}
	if x, y := s.HeadingName("kind"); y != 4 || x != 4 {
		t.Errorf("the heading's kind at %d, %d", x, y)
	}
	if x, name := s.HeaderName("note"); name != "note" || x != 13 {
		t.Errorf("the header's note at %d: %q", x, name)
	}
	if l := s.Label("tags", "sweet"); l.Name != "sweet" || l.Color != "" {
		t.Errorf("a label: %+v", l)
	}
	if s.Focused() || s.InRows() || s.Layered() {
		t.Error("a sheet the screen has not given the keyboard")
	}
	if seg, layer := s.Keys(); layer || len(seg.Bindings) != 0 {
		t.Errorf("keys without the keyboard: %+v", seg)
	}
}

func TestASheetsSearchAndSettingsNarrowAndFoldItsRows(t *testing.T) {
	s := sheet("")
	if s.Search("pea"); len(s.Rows()) != 3 || !strings.Contains(strings.Join(screenOf(s), "\n"), "pear") {
		t.Errorf("searched: %d rows", len(s.Rows()))
	}
	s.Search("pea")
	if s.Search("nothing"); !slices.Contains(screenOf(s), "    no row matches the search") {
		t.Errorf("none searched:\n%s", strings.Join(screenOf(s), "\n"))
	}
	if s.Settings(shell.Settings{SharedEmoji: true}); s.Panel().Fold || !s.Panel().Glyphs {
		t.Error("the settings")
	}
	empty := NewSheet(things, "things", "75", "")
	empty.SetRecords(nil)
	if got := ansi.Strip(strings.Join(empty.View(), "\n")); !strings.Contains(got, "    no rows") {
		t.Errorf("no rows: %q", got)
	}
	if empty.FocusRows(); !empty.Focused() || empty.InRows() {
		t.Error("a table without a row gives its control line the keyboard")
	}
}

func TestTheKeyboardMovesThroughTheControlLineAndTheRows(t *testing.T) {
	s := sheet("")
	s.Focus()
	if !s.Focused() || s.Stop() != 0 {
		t.Fatalf("focused on stop %d", s.Stop())
	}
	if seg, _ := s.Keys(); !strings.Contains(shellLine(seg), "⏎ settings") {
		t.Errorf("on the kind: %s", shellLine(seg))
	}
	// tab to the rows' stop, then the preset picker, whose keys are the preset line's; shift-tab back
	pressed(&s, "tab", "tab")
	if s.Stop() != -1 || !strings.Contains(ansi.Strip(s.View()[0]), "Default ▼") {
		t.Errorf("past the stops: %d", s.Stop())
	}
	if seg, _ := s.Keys(); !strings.Contains(shellLine(seg), "⏎ presets") || !strings.Contains(shellLine(seg), "f fields") {
		t.Errorf("on the picker: %s", shellLine(seg))
	}
	// f there opens the columns dialog, which esc closes
	if pressed(&s, "f"); !s.cols.Open {
		t.Error("f on the picker")
	}
	pressed(&s, "esc")
	if pressed(&s, "shift+tab"); s.Stop() != 1 {
		t.Errorf("shift-tab: %d", s.Stop())
	}
	// enter on the kind opens its card, which esc closes
	pressed(&s, "shift+tab", "enter")
	if !s.Layered() || !s.pop.Heading || s.pop.Field != "kind" {
		t.Fatal("enter on the kind's stop")
	}
	if seg, layer := s.Keys(); !layer || !strings.Contains(shellLine(seg), "esc") {
		t.Errorf("the card's keys: %s", shellLine(seg))
	}
	if pressed(&s, "esc"); s.Layered() {
		t.Error("esc closes the card")
	}
	// down enters the rows on the first group's heading; down again on apple
	pressed(&s, "down", "down")
	if !s.InRows() || s.Rows()[s.tbl.Cursor].Key != "apple" {
		t.Fatalf("in the rows on %q", s.Rows()[s.tbl.Cursor].Key)
	}
	// a sheet selects nothing, since nothing acts on a selection there: its keys name no selecting, and space, shift
	// with up or down, and ctrl+a select no row
	if seg, _ := s.Keys(); strings.Contains(shellLine(seg), "select") || strings.Contains(shellLine(seg), "all") {
		t.Errorf("the rows' keys: %s", shellLine(seg))
	}
	pressed(&s, "space", "shift+down", "shift+up", "ctrl+a")
	if n := len(SelectedRows(&s)); n != 0 || strings.Contains(ansi.Strip(s.View()[0]), "selected") {
		t.Errorf("selected %d", n)
	}
	if s.Rows()[s.tbl.Cursor].Key != "apple" {
		t.Errorf("shift with down and up moves back to apple, on %q", s.Rows()[s.tbl.Cursor].Key)
	}
	pressed(&s, "]", "}", "[", "{", "end", "home", "pgdown", "pgup")
	// kale opens and closes
	pressed(&s, "end", "up")
	if s.Rows()[s.tbl.Cursor].Key != "kale" {
		t.Fatalf("on %q", s.Rows()[s.tbl.Cursor].Key)
	}
	if pressed(&s, "right"); !s.tbl.Open.Unfolds(s.Rows()[s.tbl.Cursor]) {
		t.Error("right opens kale")
	}
	pressed(&s, "left", "alt+right", "alt+left")
	if got := pressed(&s, "x"); got[0] {
		t.Error("the rows took x")
	}
	// up past the first row reaches the control line, and esc there gives the screen the keyboard
	pressed(&s, "home", "up")
	if s.InRows() || !s.Focused() {
		t.Fatalf("up from the first: in the rows %v", s.InRows())
	}
	if pressed(&s, "esc"); s.Focused() {
		t.Error("esc from the control line")
	}
	if got := pressed(&s, "down"); got[0] {
		t.Error("a sheet without the keyboard took down")
	}
	// esc in the rows gives the screen the keyboard, the window back at its top
	s.FocusRows()
	if pressed(&s, "esc"); s.Focused() || s.tbl.Offset != 0 {
		t.Error("esc in the rows")
	}
	s.FocusRows()
	if s.Blur(); s.Focused() {
		t.Error("blur")
	}
}

// shellLine gives a segment's keys as the keys bar names them.
func shellLine(seg shell.Segment) string { return shell.Path{seg}.String() }

func TestGreaterAndLessShowALevelMoreAndLessOfTheWholeSheet(t *testing.T) {
	s := sheet("")
	s.Focus()
	pressed(&s, "down")
	open := len(s.Rows())
	// < closes a level of the whole table, from the rows; > opens it again
	if took := pressed(&s, "<"); !took[0] || len(s.Rows()) >= open {
		t.Fatalf("< in the rows: took %v, %d rows of %d", took, len(s.Rows()), open)
	}
	if took := pressed(&s, ">"); !took[0] || len(s.Rows()) != open {
		t.Errorf("> in the rows: took %v, %d rows of %d", took, len(s.Rows()), open)
	}
	// the control line takes them too
	pressed(&s, "esc")
	s.Focus()
	if pressed(&s, "<"); len(s.Rows()) >= open {
		t.Errorf("< on the control line: %d rows of %d", len(s.Rows()), open)
	}
	pressed(&s, ">")
	// while a preset's name is typed, < folds nothing
	s.tbl.SavingAs = true
	if pressed(&s, "<"); len(s.Rows()) != open {
		t.Errorf("< while a name is typed: %d rows of %d", len(s.Rows()), open)
	}
}

func TestThePresetsAreChosenSavedRenamedAndDeleted(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "presets.json")
	// another table's presets are in the file already, and stay
	if err := WritePresets(path, map[string][]Preset{"other": {{Name: "theirs"}}}); err != nil {
		t.Fatal(err)
	}
	s := sheet(path)
	s.Focus()
	pressed(&s, "tab", "tab", "enter")
	if !s.Layered() {
		t.Fatal("enter on the picker drops the presets")
	}
	// c moves to Compact, which enter applies
	if pressed(&s, "c", "enter"); s.tbl.ViewName != "Compact" || slices.Contains(s.ShownColumns(), "note") {
		t.Fatalf("Compact: %q, %q", s.tbl.ViewName, s.ShownColumns())
	}
	// Save As… takes a name, typed, and enter saves it
	s.choose(ItemSaveAs)
	if !s.tbl.SavingAs || !strings.Contains(ansi.Strip(s.View()[0]), "save as") {
		t.Fatal("Save As…")
	}
	if seg, _ := s.Keys(); !strings.Contains(shellLine(seg), "save") {
		t.Errorf("naming's keys: %s", shellLine(seg))
	}
	pressed(&s, "m", "i", "n", "x", "backspace", "e", "enter")
	if s.tbl.ViewName != "mine" || !strings.Contains(s.Notice().Text, "saved the preset mine") {
		t.Fatalf("saved: %q, %q", s.tbl.ViewName, s.Notice().Text)
	}
	all, err := ReadPresets(path)
	if err != nil || len(all["things"]) != 1 || all["other"][0].Name != "theirs" {
		t.Fatalf("the file: %v %+v", err, all)
	}
	s.Faded(s.Notice().FadeMsg())
	if s.Notice().Text != "" {
		t.Error("the notice fades")
	}
	// a change, saved under the preset the line was set to
	s.SetPanel(s.Panel().Sorted("kind"))
	s.choose(ItemSave)
	if all, _ := ReadPresets(path); all["things"][0].Fields["kind"].Sort != "asc" {
		t.Errorf("saved again: %+v", all["things"][0])
	}
	// a name that is none, or the table's own, is refused, and esc stops naming
	for name, want := range map[string]string{"": "needs a name", "Default": "the table's own"} {
		s.choose(ItemSaveAs)
		s.tbl.Naming = name
		if pressed(&s, "enter"); !strings.Contains(s.Notice().Text, want) {
			t.Errorf("save as %q: %q", name, s.Notice().Text)
		}
		pressed(&s, "esc")
	}
	// Rename… to a name taken, to the same name, and to a new one
	s.choose(ItemSaveAs)
	s.tbl.Naming = "theirs too"
	pressed(&s, "enter")
	s.choose(ItemRename)
	s.tbl.Naming = "mine"
	if pressed(&s, "enter"); !strings.Contains(s.Notice().Text, "a preset already") {
		t.Errorf("renamed to one taken: %q", s.Notice().Text)
	}
	s.tbl.Naming = "theirs too"
	if pressed(&s, "enter"); s.tbl.SavingAs {
		t.Error("renamed to its own name")
	}
	s.choose(ItemRename)
	s.tbl.Naming = "ours"
	if pressed(&s, "enter"); s.tbl.ViewName != "ours" || !strings.Contains(s.Notice().Text, "renamed the preset") {
		t.Errorf("renamed: %q %q", s.tbl.ViewName, s.Notice().Text)
	}
	// up while naming stops it and moves on
	s.choose(ItemSaveAs)
	if pressed(&s, "up"); s.tbl.SavingAs || s.Focused() {
		t.Error("up while naming")
	}
	// Delete drops the preset, and the table is set as Default
	s.choose(ItemDelete)
	if s.tbl.ViewName != Default || !strings.Contains(s.Notice().Text, "deleted the preset ours") {
		t.Errorf("deleted: %q %q", s.tbl.ViewName, s.Notice().Text)
	}
	// on a table's own preset, Save does nothing
	s.choose(ItemSave)
	if strings.Contains(s.Notice().Text, "saved") {
		t.Error("Save on Default")
	}
	// 📇 Fields opens the fields dialog, whose enter gives a field's card the keyboard, which esc gives back
	if clickOn(t, &s, FieldsButton, 0); !s.Layered() || !s.cols.Open {
		t.Fatal("📇 Fields")
	}
	pressed(&s, "space")
	pressed(&s, "enter")
	if !s.cols.InCard {
		t.Fatal("enter in the dialog gives the card the keyboard")
	}
	if seg, layer := s.Keys(); !layer || len(seg.Bindings) == 0 {
		t.Error("the card's keys")
	}
	if box := strings.Join(screenOf(s), "\n"); !strings.Contains(box, "rendering") {
		t.Error("the card drawn beside the list")
	}
	pressed(&s, "esc", "esc")
	if s.Layered() {
		t.Error("esc gives the list the keyboard, then closes the dialog")
	}
}

func TestAPresetThatCannotBeKeptSaysWhy(t *testing.T) {
	dir := t.TempDir()
	broken := filepath.Join(dir, "presets.json")
	if err := os.WriteFile(broken, []byte("{"), 0o644); err != nil {
		t.Fatal(err)
	}
	s := sheet(broken)
	if !strings.Contains(s.Notice().Text, "does not read as presets") {
		t.Errorf("a file that does not read: %q", s.Notice().Text)
	}
	s.saveAs("mine")
	if !strings.Contains(s.Notice().Text, "could not be saved") || s.tbl.ViewName == "mine" {
		t.Errorf("saved into it: %q", s.Notice().Text)
	}
	// a file under a file cannot be written
	blocked := filepath.Join(broken, "presets.json")
	s = sheet(blocked)
	if !strings.Contains(s.Notice().Text, "not a directory") {
		t.Errorf("a file under a file: %q", s.Notice().Text)
	}
	s.saveAs("mine")
	if !strings.Contains(s.Notice().Text, "could not be saved") {
		t.Errorf("written under a file: %q", s.Notice().Text)
	}
	s.saved = []Preset{{Name: "mine"}}
	s.tbl.ViewName, s.tbl.Renaming, s.tbl.Naming = "mine", "mine", "yours"
	if s.rename(); !strings.Contains(s.Notice().Text, "could not be renamed") {
		t.Errorf("renamed under a file: %q", s.Notice().Text)
	}
	s.choose(ItemDelete)
	if !strings.Contains(s.Notice().Text, "could not be deleted") {
		t.Errorf("deleted under a file: %q", s.Notice().Text)
	}
}

func TestAPlainClickReachesEveryControlOfTheSheet(t *testing.T) {
	s := sheet("")
	// the kind's name in the heading opens its card; a click off it closes it
	if !clickOn(t, &s, "kind ▾", 0) || !s.Layered() || !s.pop.Heading {
		t.Fatal("a click on the kind")
	}
	if !s.Click(testkit.Click(70, 22)) || s.Layered() {
		t.Error("a click off the card closes it")
	}
	// the ▾ before kind closes the level, and again opens it
	clickOn(t, &s, "▾ kind", 0)
	if len(s.Rows()) > 1 {
		t.Errorf("the level closed: %d rows", len(s.Rows()))
	}
	clickOn(t, &s, "▸ kind", 0)
	// the selector drops the presets down; a click on Compact applies it
	if clickOn(t, &s, "Default ▼", 0); !s.presets.Open {
		t.Fatal("a click on the selector")
	}
	if clickOn(t, &s, "Compact", 0); s.tbl.ViewName != "Compact" {
		t.Fatalf("a click on Compact: %q", s.tbl.ViewName)
	}
	// elsewhere on the control line gives it the keyboard, on the picker
	if s.Click(testkit.Click(40, 4)); !s.Focused() || s.Stop() != -1 || s.Layered() {
		t.Error("a click on the control line")
	}
	// a name in the header opens its card under it; its Show hides it, and esc closes the card
	if clickOn(t, &s, "tags", 0); !s.Layered() || !s.pop.Header || s.pop.Field != "tags" {
		t.Fatal("a click on the header's tags")
	}
	if clickOn(t, &s, ShowIt, 0); slices.Contains(s.ShownColumns(), "tags") {
		t.Error("Show")
	}
	pressed(&s, "esc")
	// the rule under name turns its sort, a step a click, and a shift-click turns it off
	_, rule := spot(s, "·─")
	x, _ := spot(s, "name")
	s.Click(testkit.Click(x, rule))
	s.Click(testkit.Click(x, rule))
	if s.Panel().Field("name").Sort != grid.Descending {
		t.Errorf("a click on the rule: %v", s.Panel().Field("name").Sort)
	}
	s.Click(tea.MouseClickMsg{X: x, Y: rule, Button: tea.MouseLeft, Mod: tea.ModShift})
	if s.Panel().Field("name").Sort != grid.Unsorted {
		t.Error("a shift-click on the rule")
	}
	// a list's × on the rule flattens it, and the ÷ that stands there then gathers it
	s.apply(s.Views().Chosen("Compact"))
	clickOn(t, &s, FlattenGlyph, 0)
	if !s.Panel().Field("tags").Flat {
		t.Fatal("a click on ×")
	}
	clickOn(t, &s, GatherGlyph, 0)
	if s.Panel().Field("tags").Flat {
		t.Error("a click on ÷")
	}
	// a click on a row puts the cursor there; on apple's ▸ it opens apple
	if clickOn(t, &s, "leek", 0); !s.InRows() || s.Rows()[s.tbl.Cursor].Key != "leek" {
		t.Error("a click on leek")
	}
	clickOn(t, &s, "▸ apple", 0)
	if !s.tbl.Open["apple"] {
		t.Error("a click on apple's ▸")
	}
	// under the rows, off the pane, and with another button
	if !s.Click(testkit.Click(10, 19)) || s.Click(testkit.Click(79, 10)) || s.Click(testkit.Click(10, 2)) ||
		s.Click(tea.MouseClickMsg{X: 10, Y: 10, Button: tea.MouseRight}) {
		t.Error("clicks the pane does not take")
	}
}

func TestAClickOnTheFoldOfAHeaderThatHidesNamesOpensIt(t *testing.T) {
	s := NewSheet(&Schema{Fields: []Field{
		{ID: "a", Label: "a name too long for its column", Type: "text", Renderings: []string{"text"}},
		{ID: "b", Label: "b", Type: "text", Renderings: []string{"text"}, Wrap: true}},
		Presets: []Preset{{Name: Default}}}, "narrow", "75", "")
	s.Place(0, 12, 24, 14)
	s.SetRecords([]Record{{Row: grid.Row{Key: "1"}, Values: map[string]Value{"a": One("x"), "b": One("y")}}})
	if _, hides := s.Header(); !hides {
		t.Skipf("the header does not hide names at this width:\n%s", strings.Join(screenOf(s), "\n"))
	}
	s.Click(testkit.Click(s.paneWidth()-s.GridWidth()+FoldCell, s.HeaderTop()))
	if !s.tbl.Open[HeaderKey] {
		t.Error("a click on the header's fold")
	}
}

// A table opens on the first of its own presets, whatever its name, and a preset the person deleted gives way to it.
func TestASheetOpensOnTheFirstOfItsOwnPresets(t *testing.T) {
	grouped := Preset{Name: "By kind", Fields: map[string]Setting{"kind": {Placement: PlaceGroup, Depth: 1}}}
	s := NewSheet(&Schema{Fields: things.Fields, Presets: []Preset{grouped, {Name: "Flat"}}}, "first", "75", "")
	s.SetRecords([]Record{thing("a", "fruit", "x"), thing("b", "nut", "y")})
	if s.tbl.ViewName != "By kind" || s.Views().Is(s.tbl) != "By kind" ||
		!slices.Equal(s.Panel().Groups(), []string{"kind"}) {
		t.Errorf("opened on %q, is %q, grouped %q", s.tbl.ViewName, s.Views().Is(s.tbl), s.Panel().Groups())
	}
	if names := s.Views().All(); len(names) != 2 || s.Views().Builtin(Default) {
		t.Errorf("its views: %+v", names)
	}
	s.saved = []Preset{{Name: "ours"}}
	s.apply(s.Views().Chosen("ours"))
	if s.choose(ItemDelete); s.tbl.ViewName != "By kind" {
		t.Errorf("after a delete: on %q", s.tbl.ViewName)
	}
}

func TestASheetGroupsAsItLoadsAndKnowsItsDefault(t *testing.T) {
	// a table whose catalog names no Default has one, as it opens, which groups what every row shares
	s := NewSheet(&Schema{Fields: things.Fields}, "bare", "75", "")
	s.SetRecords([]Record{thing("a", "fruit", "x"), thing("b", "fruit", "y")})
	if names := s.Views().All(); len(names) != 1 || names[0].Name != Default {
		t.Errorf("its views: %+v", names)
	}
	if !slices.Contains(s.Panel().Groups(), "kind") || s.Views().Is(s.tbl) != Default {
		t.Errorf("grouped as it loads: %q, is %q", s.Panel().Groups(), s.Views().Is(s.tbl))
	}
}

// bare is a sheet with no control line over its rows, as a table without a preset line draws them.
type bare struct{ *Sheet }

func (b bare) Controls() bool { return false }

func TestTheRuleAndTheControlLineSayWhatIsSelectedAndFiltered(t *testing.T) {
	s := sheet("")
	s.FocusRows()
	// a sheet selects no row itself; a row selected as a host that selects selects one, the drawing is the same
	pressed(&s, "down")
	SelectRow(&s, false)
	if rule := ansi.Strip(Rule(bare{&s}, SortSpots(&s))); !strings.HasSuffix(rule, " 1 selected ─") {
		t.Errorf("the rule without a control line: %q", rule)
	}
	if rule := ansi.Strip(Rule(&s, SortSpots(&s))); strings.Contains(rule, "selected") {
		t.Errorf("the rule under a control line: %q", rule)
	}
	s.SetPanel(s.Panel().FilteredBy("kind", grid.Filter{Values: []string{"fruit"}}))
	if line := ansi.Strip(s.View()[0]); !strings.Contains(line, "1 selected  kind: fruit  "+FieldsButton+"  ") ||
		!strings.Contains(line, "Default ▼") {
		t.Errorf("the control line: %q", line)
	}
	// a click on the chip opens kind's card under it, on its filter
	if clickOn(t, &s, "kind: fruit", 0); !s.pop.Open || !s.pop.Chip || s.pop.Field != "kind" {
		t.Errorf("a click on kind's chip: %+v", s.pop)
	}
	s.pop = Popover{}
	if line, at := Control(&s, "  left", ""); at != -1 || !strings.HasPrefix(ansi.Strip(line), "  left") {
		t.Errorf("a control line without a preset line: %q %d", line, at)
	}
	// a spot the rule cannot hold is left off it
	spots := []SortSpot{{Column: "a", Mark: "▲", At: 2, Until: 4}, {Column: "b", Mark: "▼", At: 40, Until: 41}}
	if fill := ansi.Strip(RuleFill(spots, 4, 10)); fill != strings.Repeat("─", 10) {
		t.Errorf("spots off the rule: %q", fill)
	}
	if c, _ := SortAt("  ──", spots, 9); c != "" {
		t.Error("past the rule's end")
	}
	if c, _ := SortAt("  ──", spots, 0); c != "" {
		t.Error("on the rule's blank start")
	}
	if c, _ := SortAt("  ────", spots, 2); c != "a" {
		t.Error("on a sort's mark")
	}
	if c, _ := SortAt("  ────", spots, 5); c != "" {
		t.Error("on the rule between marks")
	}
}

func TestWhereAColumnsNameStandsIsWhereAClickFindsIt(t *testing.T) {
	s := sheet("")
	x, y := spot(s, "note")
	if HeaderColumn(&s, x, y) != "note" || HeaderColumn(&s, x, y+1) != "" || HeaderColumn(&s, 1, y) != "" ||
		HeaderColumn(&s, x, 2) != "" {
		t.Error("the header's columns")
	}
	if on, _, _ := HeaderHit(&s, x, 2); on {
		t.Error("a click off the header")
	}
	if _, y := FieldName(&s, "note"); y != -1 {
		t.Error("a column the heading over the groups does not name")
	}
	s.tbl.Open["x"] = true
	if ToggleHeader(&s); !s.tbl.Open[HeaderKey] || !s.tbl.Open["x"] {
		t.Errorf("the header opened, the rest as it stood: %v", s.tbl.Open)
	}
	// with every column hidden, there is no header
	p := s.Panel()
	for _, id := range []string{"name", "note"} {
		p = p.Placed(id, grid.Hidden)
	}
	if s.SetPanel(p); func() bool { h, _ := s.Header(); return h != nil }() {
		t.Error("a header over no columns")
	}
}

func TestTheRowsKeysNameWhatActsUnderTheCursor(t *testing.T) {
	s := sheet("")
	s.Place(4, 7, 80, 24)
	s.FocusRows()
	line := func() string { return shellLine(RowKeys(&s, true)) }
	if !strings.Contains(line(), "⇞⇟ page") || !strings.Contains(line(), "[] group") ||
		!strings.Contains(line(), "← hide") || !strings.Contains(line(), "⌥←→ a level") {
		t.Errorf("on the first heading: %s", line())
	}
	pressed(&s, "down")
	if !strings.Contains(line(), "← group") {
		t.Errorf("on a row: %s", line())
	}
	pressed(&s, "end", "up")
	if !strings.Contains(line(), "→ show") {
		t.Errorf("on kale: %s", line())
	}
	s.tbl.Cursor = 99
	if strings.Contains(line(), "show") {
		t.Errorf("past the rows: %s", line())
	}
	// a host that acts on the rows selected names space, shift with up or down, and ctrl+a
	if line := shellLine(RowKeys(selects{&s}, false)); !strings.Contains(line, "␣·⇧↑↓ select") ||
		!strings.Contains(line, "⌃A all") {
		t.Errorf("a host that selects: %s", line)
	}
}

// selects is a sheet that acts on the rows selected, as a screen that does says.
type selects struct{ *Sheet }

func (selects) Selects() bool { return true }

func TestTheControlLinesKeysMoveFoldAndDrop(t *testing.T) {
	s := sheet("")
	s.Focus()
	// left closes the kind's level, right opens it, and space and enter on the rows' stop turn theirs
	if pressed(&s, "left"); s.tbl.Open.Shows(1) {
		t.Error("left on the kind")
	}
	pressed(&s, "right", "tab", "space")
	if s.tbl.Open.Shows(2) {
		t.Error("space on the rows")
	}
	pressed(&s, "enter")
	if !s.tbl.Open.Shows(2) {
		t.Error("enter on the rows")
	}
	if got := pressed(&s, "x"); got[0] {
		t.Error("a stop took x")
	}
	if seg := HeadingKeys(&s, Stop{}, false); strings.Contains(shellLine(seg), "menu") {
		t.Errorf("a stop with no column: %s", shellLine(seg))
	}
	pressed(&s, "down")
	if !s.InRows() {
		t.Error("down from a stop")
	}
	// the picker: down enters the rows, shift-tab where nothing groups goes nowhere, and x is not its
	s.apply(s.Views().Chosen("Compact"))
	s.Focus()
	if got := pressed(&s, "tab", "x"); got[1] || s.Stop() != -1 {
		t.Errorf("the picker took %v", got)
	}
	if pressed(&s, "down"); !s.InRows() {
		t.Error("down from the picker")
	}
	// while naming, a key it does not type is not its, and esc cancels
	s.choose(ItemSaveAs)
	if got := pressed(&s, "ctrl+x"); got[0] {
		t.Error("naming took ctrl+x")
	}
	if seg, _ := s.Keys(); !strings.Contains(shellLine(seg), "save") {
		t.Errorf("naming's keys: %s", shellLine(seg))
	}
	s.tbl.Renaming = "x"
	if seg, _ := s.Keys(); !strings.Contains(shellLine(seg), "rename") {
		t.Errorf("renaming's keys: %s", shellLine(seg))
	}
	pressed(&s, "esc")
	if s.tbl.SavingAs {
		t.Error("esc while naming")
	}
	// a table without rows names no down
	empty := NewSheet(things, "things", "75", "")
	empty.Focus()
	if seg, _ := empty.Keys(); strings.Contains(shellLine(seg), "rows") {
		t.Errorf("no rows: %s", shellLine(seg))
	}
	if seg := HeadingKeys(&empty, Stop{Column: "kind"}, false); strings.Contains(shellLine(seg), "rows") ||
		strings.Contains(shellLine(seg), "move") {
		t.Errorf("a lone stop over no rows: %s", shellLine(seg))
	}
	if got := pressed(&empty, "down"); !got[0] || empty.InRows() {
		t.Error("down from the picker of a table without rows")
	}
	// a table without fields has no control line, and its rows take the keyboard first
	plain := NewSheet(&Schema{Presets: []Preset{{Name: Default}}}, "plain", "75", "")
	plain.SetRecords([]Record{{Row: grid.Row{Key: "a", Cells: []grid.Cell{{Values: []string{"a"}}}}}})
	if plain.Focus(); !plain.InRows() {
		t.Error("a table without fields")
	}
}

func TestTheLayersTakeKeysAndClicks(t *testing.T) {
	s := sheet("")
	// a column's card takes keys, and names them
	s.pop = OpenCard(&s, "kind", false)
	at := s.pop.At
	if pressed(&s, "down"); s.pop.At == at {
		t.Fatal("down in the card")
	}
	if seg, _ := s.Keys(); len(seg.Bindings) == 0 {
		t.Error("the card's keys")
	}
	s.Click(testkit.Click(0, 0))
	s.pop = Popover{}
	// the fields dialog takes a click, a click on a field's label opening its card over it, and the presets' menu a
	// click off it
	s.cols = Columns{Open: true}
	if x, y := spot(s, "☑"); !s.Click(testkit.Click(x, y)) || !s.cols.Open {
		t.Error("a click in the fields dialog")
	}
	if x, y := spot(s, "☑"); !s.Click(testkit.Click(x+4, y)) || !s.cols.InCard {
		t.Errorf("a click on a field's label: %+v", s.cols)
	}
	// a card opened alone widens into the dialog, by tab or a click on its all ▸
	s.cols = Columns{}
	s.pop = OpenCard(&s, "kind", false)
	if pressed(&s, "tab"); s.pop.Open || !s.cols.Open || s.cols.Listed(&s)[s.cols.At] != "kind" {
		t.Errorf("tab on kind's card: %+v", s.cols)
	}
	s.cols = Columns{}
	s.pop = OpenCard(&s, "kind", false)
	if clickOn(t, &s, AllFields, 1); s.pop.Open || !s.cols.Open {
		t.Errorf("a click on all ▸: %+v", s.cols)
	}
	s.pop = Popover{}
	s.cols = Columns{}
	s.presets = s.presets.Drop(&s)
	if !s.Click(testkit.Click(1, 22)) || s.presets.Open {
		t.Error("a click off the presets' menu")
	}
}

func TestTheEdgesOfAnEmptyTableAndAMenuThatLostItsName(t *testing.T) {
	// a file not there yet reads as no presets, and says nothing
	if s := sheet(filepath.Join(t.TempDir(), "presets.json")); s.Notice().Text != "" || len(s.saved) != 0 {
		t.Errorf("no file: %q", s.Notice().Text)
	}
	// the picker of a table without rows: shift-tab has no stop to go to, down no row, and x is not its
	empty := NewSheet(things, "things", "75", "")
	empty.Focus()
	if got := pressed(&empty, "shift+tab", "down", "x"); got[0] || !got[1] || got[2] || empty.InRows() {
		t.Errorf("the picker took %v", got)
	}
	// a card hung from a name the heading over the groups no longer shows stays where it was
	s := sheet("")
	s.OpenFieldMenu("kind")
	x, y := s.pop.X, s.pop.Y
	s.SetPanel(s.Panel().Placed("kind", grid.Column))
	if got := s.Over(make([]string, 24)); !strings.Contains(ansi.Strip(got[y-2]), "kind") || s.pop.X != x || s.pop.Y != y {
		t.Errorf("a card without its name moved to %d, %d", s.pop.X, s.pop.Y)
	}
}

func TestTwoWritersOfThePresetsFileKeepEachOthersPresets(t *testing.T) {
	path := filepath.Join(t.TempDir(), "presets.json")
	// two tables open on one file, each read before the other saved
	a := sheet(path)
	b := NewSheet(&Schema{Stage: "others", Fields: things.Fields, Presets: things.Presets}, "others", "75", path)
	b.Place(4, 16, 80, 24)
	b.SetRecords(stock())
	a.saveAs("ours")
	b.saveAs("theirs")
	a.saveAs("ours too")
	all, err := ReadPresets(path)
	if err != nil || len(all["things"]) != 2 || len(all["others"]) != 1 || all["others"][0].Name != "theirs" {
		t.Fatalf("both writers' presets: %v %+v", err, all)
	}
	// a writer that keeps its own presets keeps what is there, and a file that does not read is not written over
	if got, err := KeepPresets(path, "third", []Preset{{Name: "x"}}, nil); err != nil || len(got) != 3 {
		t.Errorf("a third writer: %v %+v", err, got)
	}
	// favorites are kept beside the presets, each table's, and saving a table's presets keeps another's
	if err := KeepFavorites(path, "third", []string{"x"}); err != nil || !slices.Equal(ReadFavorites(path)["third"], []string{"x"}) {
		t.Errorf("third's favorites: %v %q", err, ReadFavorites(path))
	}
	a.saveAs("ours again")
	if all, _ := ReadPresets(path); len(all["third"]) != 1 || !slices.Equal(ReadFavorites(path)["third"], []string{"x"}) {
		t.Errorf("another table's save dropped third's favorites: %q", ReadFavorites(path))
	}
	if err := WritePresets(path, map[string][]Preset{"third": {{Name: "x"}}}); err != nil ||
		len(ReadFavorites(path)["third"]) != 1 {
		t.Errorf("writing the presets dropped the favorites: %v", err)
	}
	if _, err := KeepPresets(path, "third", nil, nil); err != nil || len(ReadFavorites(path)) != 0 {
		t.Errorf("no favorites left: %v %q", err, ReadFavorites(path))
	}
	if err := os.WriteFile(path, []byte("{"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := KeepPresets(path, "third", nil, nil); err == nil {
		t.Error("kept over a file that does not read")
	}
	if KeepFavorites(path, "third", []string{"x"}) == nil || ReadFavorites(path) == nil {
		t.Error("favorites kept over a file that does not read")
	}
	if data, _ := os.ReadFile(path); string(data) != "{" {
		t.Errorf("the file was written over: %s", data)
	}
	// a file not there yet in a directory that cannot be written to is the error
	locked := t.TempDir()
	if err := os.Chmod(locked, 0o500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(locked, 0o700) })
	if _, err := KeepPresets(filepath.Join(locked, "presets.json"), "third", nil, nil); err == nil {
		t.Error("kept in a directory that cannot be written to")
	}
}

// A view starred in the presets' menu stands first, in favorites, and the file keeps it; starred again it no longer
// is. A favorite follows its view's rename, and is dropped once the view is deleted. Where the file cannot be written,
// starring says so, and stars nothing.
func TestAStarredViewIsAFavoriteTheFileKeeps(t *testing.T) {
	path := filepath.Join(t.TempDir(), "presets.json")
	s := sheet(path)
	compact := slices.IndexFunc(PresetItems(&s), func(it shell.Item) bool { return it.Name == "Compact" })
	s.star(compact)
	if !slices.Equal(ReadFavorites(path)["things"], []string{"Compact"}) || PresetItems(&s)[ItemViews+1].Name != FavoritesHeading ||
		!strings.Contains(s.Notice().Text, "starred the preset Compact") {
		t.Fatalf("Compact starred: %q, %q", ReadFavorites(path), s.Notice().Text)
	}
	// * in the menu, on Compact, now the favorite under the keyboard, stars it no longer; the keys bar names it
	s.presets = s.presets.Drop(&s)
	s.presets.At = ItemViews + 2
	if seg, _ := s.Keys(); !strings.Contains(shellLine(seg), "* star") {
		t.Errorf("the menu's keys: %s", shellLine(seg))
	}
	if !s.Key(tea.KeyPressMsg{Code: '*', Text: "*"}) || len(ReadFavorites(path)["things"]) != 0 || !s.presets.Open ||
		!strings.Contains(s.Notice().Text, "unstarred") {
		t.Errorf("Compact unstarred: %q", ReadFavorites(path))
	}
	s.presets.At = ItemSave
	if seg, _ := s.Keys(); strings.Contains(shellLine(seg), "* star") {
		t.Errorf("the menu's keys off a view: %s", shellLine(seg))
	}
	s.presets = PresetMenu{}
	s.star(ItemSave)
	// a favorite of the person's own follows its rename, and goes with it
	s.saveAs("Mine")
	s.star(slices.IndexFunc(PresetItems(&s), func(it shell.Item) bool { return it.Name == "Mine" }))
	s.tbl.SavingAs, s.tbl.Renaming, s.tbl.Naming = true, "Mine", "Ours"
	s.rename()
	if !slices.Equal(ReadFavorites(path)["things"], []string{"Ours"}) || !slices.Equal(s.starred, []string{"Ours"}) {
		t.Errorf("renamed: %q", ReadFavorites(path))
	}
	s.keep(s.Views().Delete("Ours"), "deleted")
	if len(ReadFavorites(path)["things"]) != 0 || len(s.starred) != 0 {
		t.Errorf("deleted: %q", ReadFavorites(path))
	}
	// a file that does not read takes no star
	if err := os.WriteFile(path, []byte("{"), 0o644); err != nil {
		t.Fatal(err)
	}
	s.star(compact)
	if len(s.starred) != 0 || !strings.Contains(s.Notice().Text, "could not be starred") {
		t.Errorf("starred over a file that does not read: %q", s.Notice().Text)
	}
}

// A turn of the mouse wheel over the pane scrolls the rows' window WheelLines a turn, never past its ends, the cursor
// left where it is; a turn off the pane, a turn that is no up or down, or one while a layer is open scrolls nothing.
func TestTheWheelScrollsTheRowsAndLeavesTheCursor(t *testing.T) {
	s := sheet("")
	s.Place(4, 7, 80, 24)
	s.FocusRows()
	pressed(&s, "down")
	cursor := s.tbl.Cursor
	turn := func(y int, b tea.MouseButton) bool { return s.Wheel(tea.MouseWheelMsg{X: 5, Y: y, Button: b}) }
	if !turn(8, tea.MouseWheelDown) || s.tbl.Offset != WheelLines || s.tbl.Cursor != cursor {
		t.Errorf("a turn down: offset %d, cursor %d, want %d and %d", s.tbl.Offset, s.tbl.Cursor, WheelLines, cursor)
	}
	for range 10 {
		turn(8, tea.MouseWheelDown)
	}
	end := s.tbl.Offset
	if turn(8, tea.MouseWheelDown); s.tbl.Offset != end || end == 0 {
		t.Errorf("past the end: offset %d, then %d", end, s.tbl.Offset)
	}
	if turn(8, tea.MouseWheelUp); s.tbl.Offset != max(end-WheelLines, 0) {
		t.Errorf("a turn back up from %d: offset %d", end, s.tbl.Offset)
	}
	at := s.tbl.Offset
	if turn(2, tea.MouseWheelDown) || turn(11, tea.MouseWheelDown) || turn(8, tea.MouseLeft) || s.tbl.Offset != at {
		t.Errorf("a turn off the pane, or no turn, scrolled: offset %d, want %d", s.tbl.Offset, at)
	}
	if s.Wheel(tea.MouseWheelMsg{X: 79, Y: 8, Button: tea.MouseWheelDown}) {
		t.Error("a turn on the screen's last cell, past the pane, was taken")
	}
	s.cols = Columns{Open: true}
	if turn(8, tea.MouseWheelUp) || s.tbl.Offset != at {
		t.Errorf("a turn under the columns dialog scrolled: offset %d", s.tbl.Offset)
	}
}

// broad is a sheet of rows wider than a screen 40 cells wide: an id, then three columns of twelve cells.
func broad(width int) Sheet {
	field := func(id string) Field {
		return Field{ID: id, Label: id, Type: "text", Renderings: []string{"text"}, Placement: PlaceColumn}
	}
	s := NewSheet(&Schema{Fields: []Field{field("id"), field("aa"), field("bb"), field("cc")},
		Presets: []Preset{{Name: Default}}}, "broad", "75", "")
	s.Place(4, 10, width, 24)
	var records []Record
	for _, id := range []string{"1", "2"} {
		records = append(records, Record{Row: grid.Row{Key: id}, Values: map[string]Value{"id": One("#" + id),
			"aa": One("aaaaaaaaaa" + id), "bb": One("bbbbbbbbbb" + id), "cc": One("cccccccccc" + id)}})
	}
	s.SetRecords(records)
	return s
}

func TestTabAndATriangleScrollTheColumnsOfATableWiderThanTheScreen(t *testing.T) {
	s := broad(40)
	screen := func() string { return strings.Join(screenOf(s), "\n") }
	if got := screen(); !strings.Contains(got, grid.After) || strings.Contains(got, grid.Before) ||
		strings.Contains(got, "cccccccccc1") || strings.Contains(got, "…") {
		t.Fatalf("the columns past the edge hide behind ▹, uncut:\n%s", got)
	}
	s.FocusRows()
	if line := shellLine(RowKeys(&s, false)); !strings.Contains(line, "⇥·⇧⇥ columns") {
		t.Errorf("the keys of a table cut: %s", line)
	}
	if took := pressed(&s, "tab"); !took[0] {
		t.Fatal("tab in the rows of a table cut")
	}
	if got := screen(); !strings.Contains(got, "#1 ◃") && !strings.Contains(got, grid.Before) ||
		!strings.Contains(got, "cccccccccc1") || strings.Contains(got, "aaaaaaaaaa1") || !strings.Contains(got, "#1") {
		t.Errorf("tab scrolls a column, the id put:\n%s", got)
	}
	if pressed(&s, "tab"); s.tbl.Shift != 1 {
		t.Errorf("tab past the last column: shift %d", s.tbl.Shift)
	}
	if pressed(&s, "shift+tab"); s.tbl.Shift != 0 || !strings.Contains(screen(), "aaaaaaaaaa1") {
		t.Errorf("shift+tab scrolls back: shift %d\n%s", s.tbl.Shift, screen())
	}
	// a plain click on ▹ scrolls right, and on ◃ left
	if !clickOn(t, &s, grid.After, 0) || s.tbl.Shift != 1 {
		t.Errorf("a click on ▹: shift %d", s.tbl.Shift)
	}
	if !clickOn(t, &s, grid.Before, 0) || s.tbl.Shift != 0 {
		t.Errorf("a click on ◃: shift %d", s.tbl.Shift)
	}
	if !s.InRows() {
		t.Error("a triangle leaves the keyboard where it was")
	}
}

func TestATableThatFitsTakesNoTab(t *testing.T) {
	s := broad(80)
	s.FocusRows()
	if line := shellLine(RowKeys(&s, false)); strings.Contains(line, "columns") {
		t.Errorf("the keys of a table that fits: %s", line)
	}
	if took := pressed(&s, "tab", "shift+tab"); took[0] || took[1] {
		t.Errorf("tab and shift+tab in a table that fits: %v", took)
	}
	if got := strings.Join(screenOf(s), "\n"); strings.ContainsAny(got, grid.Before+grid.After) {
		t.Errorf("a table that fits draws no triangle:\n%s", got)
	}
}

func TestAColumnMovedPastTheEdgeScrollsIntoView(t *testing.T) {
	s := broad(40)
	if clickOn(t, &s, "bb ", 0); !s.Layered() {
		t.Fatal("a click on the header's bb drops its menu")
	}
	pressed(&s, "shift+right")
	if x, name := s.HeaderName("bb"); name != "bb" || x < 0 || s.tbl.Shift != 1 {
		t.Errorf("bb moved past the edge: at %d %q, shift %d\n%s", x, name, s.tbl.Shift,
			strings.Join(screenOf(s), "\n"))
	}
}

func TestOnlyTheHeadersFirstLineHasTriangles(t *testing.T) {
	s := broad(40)
	x, _ := spot(s, grid.After)
	_, row := spot(s, "#1")
	if d := ArrowAt(&s, x, row); d != 0 {
		t.Errorf("ArrowAt on a row, under ▹ = %d, want 0", d)
	}
}

// Under the heading over the groups stands a rule as wide as the heading, each group's marks under its name, as the
// rule under the header places a column's: a click on the sort's sorts the groups, and on a list's × it stands as a
// column again, flattened, a row per value.
func TestARuleUnderTheGroupsSortsThemAndFlattensAList(t *testing.T) {
	s := sheet("")
	bar := BarLeft + ansi.StringWidth(ansi.Strip(BarText(s.Rows())))
	rule := screenOf(s)[s.top+1]
	x, _ := spot(s, "kind")
	// it starts at the first group's name, past its fold, as the header's rule starts at a column's name
	first := BarNames(&s)[0].From
	if ansi.StringWidth(rule) != bar || ansi.Cut(rule, x, x+1) != Sortable ||
		strings.TrimSpace(ansi.Cut(rule, 0, first)) != "" ||
		strings.Trim(ansi.Cut(rule, first, ansi.StringWidth(rule)), "─"+Sortable) != "" {
		t.Fatalf("the rule under a heading %d wide: %q", bar, rule)
	}
	s.click(x, s.top+1, false)
	if s.Panel().Mark("kind") == "" || !strings.Contains(screenOf(s)[s.top+1], "▲") {
		t.Errorf("a click on the sort's mark: %q, %q", s.Panel().Mark("kind"), screenOf(s)[s.top+1])
	}
	// a list grouped by: its × makes it a column again, flattened
	grouping := *things
	grouping.Fields = slices.Clone(things.Fields)
	grouping.Fields[3].Group = true
	g := NewSheet(&grouping, "things", "75", "")
	g.Place(4, 16, 80, 24)
	g.SetRecords(stock())
	g.SetPanel(g.Panel().Grouped("tags", true))
	x, _ = spot(g, "tags")
	if r := screenOf(g)[g.top+1]; ansi.Cut(r, x, x+1) != FlattenGlyph {
		t.Fatalf("a list's rule: %q", r)
	}
	g.click(x, g.top+1, false)
	if p := g.Panel(); slices.Contains(p.Groups(), "tags") || !p.Field("tags").Flat {
		t.Errorf("tags after ×: groups %v, flat %v", p.Groups(), p.Field("tags").Flat)
	}
	// a group whose column neither sorts nor holds a list has the rule alone, and a click on the rule off every mark
	// is not taken
	h := outlined()
	if spots := BarSpots(h); len(spots) != 1 || spots[0].Column != "state" {
		t.Errorf("the marks under state and who: %+v", spots)
	}
	if BarRuleClick(h, 0, false) || h.sets != 0 {
		t.Error("a click off every mark was taken")
	}
}

// A click on a value that leads somewhere, a page or a file, opens it, as the person's opener does; where none can be
// asked, the link stands in the notice instead, to open by hand. A click elsewhere on the row opens nothing.
func TestAClickOnALinkOpensIt(t *testing.T) {
	linked := &Schema{Stage: "linked", Fields: []Field{
		{ID: "name", Label: "name", Type: "text", Renderings: []string{"text"}, Placement: PlaceColumn},
		{ID: "where", Label: "where", Type: "text", Renderings: []string{"text"}, Placement: PlaceColumn},
	}}
	s := NewSheet(linked, "linked", "75", "")
	s.Place(4, 16, 80, 24)
	s.SetRecords([]Record{{Row: grid.Row{Key: "a"}, Values: map[string]Value{"name": One("alpha"),
		"where": {Raw: []string{"kit.yml"}, Link: "/repo/kit.yml"}}}})
	var opened []string
	s.SetBrowse(func(link string) error { opened = append(opened, link); return nil })
	x, y := spot(s, "kit.yml")
	s.click(x, y, false)
	if !slices.Equal(opened, []string{"/repo/kit.yml"}) {
		t.Fatalf("a click on kit.yml opened %q", opened)
	}
	x, y = spot(s, "alpha")
	if s.click(x, y, false); len(opened) != 1 {
		t.Errorf("a click on alpha opened %q", opened)
	}
	s.SetBrowse(func(string) error { return shell.ErrNoDisplay })
	x, y = spot(s, "kit.yml")
	if s.click(x, y, false); !strings.Contains(s.Notice().Text, "/repo/kit.yml") {
		t.Errorf("no opener: %q", s.Notice().Text)
	}
}

// While kittui's settings hide them, a column no row loaded holds a value in does not show, its header with it, and
// shows again once a load gives a row a value in it; nothing is placed, so the columns dialog and the view are as they
// were. Two columns stay shown all the same. With Auto-hide off, the column stands, its header over nothing.
func TestAColumnNoRowHoldsAValueInHidesUntilOneDoes(t *testing.T) {
	bare := func() []Record {
		var out []Record
		for _, r := range stock() {
			r.Values["note"] = One("")
			out = append(out, r)
		}
		return out
	}
	s := sheet("")
	s.SetPanel(s.Panel().Placed("tags", grid.Column))
	view := s.tbl.View
	if s.SetRecords(bare()); s.Panel().Shows("note") || !s.Panel().Shows("tags") || slices.Contains(s.Names(), "note") ||
		s.Panel().Field("note").Placement != grid.Column || !reflect.DeepEqual(s.tbl.View.Fields, view.Fields) {
		t.Errorf("no note: shows %v, names %q", s.Panel().Shows("note"), s.Names())
	}
	if s.SetRecords(stock()); !s.Panel().Shows("note") || !slices.Contains(s.Names(), "note") {
		t.Errorf("noted again: names %q", s.Names())
	}
	two := sheet("")
	if two.SetRecords(bare()); !two.Panel().Shows("note") {
		t.Errorf("two columns: names %q", two.Names())
	}
	s.Settings(shell.Settings{FoldShared: true})
	if s.SetRecords(bare()); !s.Panel().Shows("note") || !slices.Contains(s.Names(), "note") {
		t.Errorf("Auto-hide off: names %q", s.Names())
	}
}
