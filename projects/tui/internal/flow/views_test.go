// The preset line on top of each stage's table: each stage's own two presets, the person's saved in their file, a
// change to the table showing a mark and 💾, a chip per filter, and a save that keeps the panel under the name typed,
// never over a stage's own; the keyboard's way to it and from it; and the views file before presets.json, read once
// into it.
package flow

import (
	"encoding/json"
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

// onDropdown gives backlog with the keyboard on its preset line, its views kept in file.
func onDropdown(t *testing.T, file string) Model {
	t.Helper()
	m := loaded(t, viewing(file), tea.WindowSizeMsg{Width: 140, Height: 40}, down)
	if !m.onPreset || m.inRows || m.bar().Focused {
		t.Fatalf("the keyboard is not on the preset line\n%s", view(m))
	}
	return m
}

// presetText gives the preset line as the screen draws it, without its escapes, the rule and the sidebar after it
// apart.
func presetText(m Model) string {
	return strings.TrimRight(ansi.Truncate(ansi.Strip(lines(m)[paneTop]), m.paneWidth(), ""), " ")
}

// shown gives the columns a stage's panel shows.
func shownNow(m Model) []string { return m.shownColumns(m.stage) }

// keysOf gives the keys bar as the screen draws it, without its escapes.
func keysOf(m Model) string { return lines(m)[len(lines(m))-1] }

func TestThePresetLineStandsOnTopOfTheTableAndNamesTheView(t *testing.T) {
	m := loaded(t, viewing(""), tea.WindowSizeMsg{Width: 140, Height: 40}, press('?'))
	if got := presetText(m); got != "   Default ▾" || m.headTop() != paneTop+1 {
		t.Errorf("the preset line: %q", got)
	}
	if !strings.Contains(keysOf(m), "↓ preset") {
		t.Errorf("down on the bar does not say it reaches the preset line: %q", keysOf(m))
	}
	// the sidebar under the search starts with the columns
	side, _ := m.panel(0).Lines()
	if got := ansi.Strip(side[2]); got != "columns" || ansi.Strip(side[1]) != "" {
		t.Errorf("the sidebar's first lines: %q", side[:3])
	}
	m = drive(m, down)
	keys := keysOf(m)
	for _, want := range []string{"↑/esc backlog", "↓ rows", "⏎ presets", "type to rename"} {
		if !strings.Contains(keys, want) {
			t.Errorf("the keys on the preset line lack %q: %q", want, keys)
		}
	}
	if strings.Contains(keys, "⌃S save") {
		t.Errorf("save is named while the table is its view: %q", keys)
	}
	// the column names hidden, the line stays, the table no longer its view
	m.stages[0].headers = false
	if got := presetText(m); !strings.HasPrefix(got, "   "+Unnamed+" ▾ "+Differs) || m.rowsTop() != paneTop+1 {
		t.Errorf("the preset line without the headers: %q", got)
	}
	// a stage with no columns has no preset line, and down on the bar says it reaches the rows
	m = drive(m, esc, press('7'))
	if m.stages[m.stage].name != "incoming" || m.presets(m.stage) || !strings.Contains(keysOf(m), "↓ rows") {
		t.Errorf("%s has a preset line: %q", m.stages[m.stage].name, keysOf(m))
	}
}

func TestDownAndUpPassThroughThePresetLine(t *testing.T) {
	m := onDropdown(t, "")
	if m = drive(m, down); !m.inRows || m.onPreset {
		t.Fatalf("down from the line: in rows %v, on the line %v", m.inRows, m.onPreset)
	}
	// up past the heading over the first row
	if m = drive(m, up, up); !m.onPreset || m.inRows {
		t.Fatalf("up from the first row: in rows %v, on the line %v", m.inRows, m.onPreset)
	}
	if m = drive(m, up); m.onPreset || !m.bar().Focused {
		t.Fatalf("up from the line: on the line %v", m.onPreset)
	}
	if m = drive(m, down, esc); m.onPreset || !m.bar().Focused {
		t.Fatalf("esc from the line: on the line %v", m.onPreset)
	}
	// tab from the line gives the sidebar the keyboard, and esc gives it back to the line
	m = drive(m, down, tab)
	if !m.inPanel || !strings.Contains(keysOf(m), "⇥/esc preset") {
		t.Fatalf("tab from the line: in panel %v, %q", m.inPanel, keysOf(m))
	}
	if m = drive(m, esc); m.inPanel || !m.onPreset {
		t.Errorf("esc from the sidebar: in panel %v, on the line %v", m.inPanel, m.onPreset)
	}
	// on the line, a number is typed, as a name may hold one; a click on a row leaves the line
	if m = drive(m, typed("2")...); !m.onPreset || m.stage != 0 || m.stages[0].naming != "2" {
		t.Errorf("a number on the line: on the line %v, stage %d, %q", m.onPreset, m.stage, m.stages[0].naming)
	}
	if m = drive(m, testkit.Click(6, m.rowsTop()+1)); m.onPreset || !m.inRows {
		t.Errorf("a click on a row: on the line %v", m.onPreset)
	}
}

func TestAChipStandsForEachFilter(t *testing.T) {
	m := onDropdown(t, "")
	m.setPanel(m.panel(0).Shown(grid.Mine))
	if got := presetText(m); !strings.Contains(got, "  assignees: @me") || !strings.HasSuffix(got, Saving) {
		t.Errorf("mine: %q", got)
	}
	m.setPanel(m.panel(0).Shown(grid.Others))
	if got := presetText(m); !strings.Contains(got, "  assignees: ¬@me") {
		t.Errorf("others: %q", got)
	}
	// too narrow, the chips are clipped before Save is
	m = drive(m, tea.WindowSizeMsg{Width: 30, Height: 40})
	if got := presetText(m); !strings.HasSuffix(got, Saving) || ansi.StringWidth(got) > m.paneWidth() {
		t.Errorf("narrow: %q", got)
	}
}

func TestChoosingAViewSetsThePanelAndAChangeShowsSave(t *testing.T) {
	m := onDropdown(t, "")
	m = drive(m, press(tea.KeyEnter))
	if !m.presetMenu {
		t.Fatal("enter did not drop the presets down")
	}
	// the menu is drawn, under the line, whole
	if drawn := raw(m); !strings.Contains(drawn, "Default") || !strings.Contains(drawn, "Compact") {
		t.Fatalf("the presets are not drawn:\n%s", drawn)
	}
	names := []string{}
	for _, i := range m.presetItems() {
		names = append(names, i.Name)
	}
	if !slices.Equal(names, []string{"Default", "Compact"}) || m.presetItems()[0].Glyph != Checked {
		t.Fatalf("the presets: %q", names)
	}
	if !strings.Contains(keysOf(m), "esc close") {
		t.Errorf("the keys with the menu open: %q", keysOf(m))
	}
	m = drive(m, down, press(tea.KeyEnter))
	if m.presetMenu || m.stages[0].viewName != "Compact" || !slices.Equal(shownNow(m), []string{"id", "title"}) ||
		m.changed(0) || presetText(m) != "   Compact ▾" {
		t.Fatalf("Compact: %q, changed %v, %q", shownNow(m), m.changed(0), presetText(m))
	}
	// a column shown again differs from every view: the line names none, the mark says so, and 💾 Save, dim, stands
	// at its right
	m.setPanel(m.panel(0).Placed("age", grid.Column))
	if got := presetText(m); !m.changed(0) || m.savable(0) || m.viewShown(0) != "" ||
		!strings.HasPrefix(got, "   "+Unnamed+" ▾ "+Differs) || !strings.HasSuffix(got, Saving) ||
		!strings.Contains(keysOf(m), "⌃S save") {
		t.Errorf("a change does not show save, or names a view: %q", got)
	}
	if at := len(lead) + len(Unnamed) + 2; m.dropEnd() != at {
		t.Errorf("▾ ends at %d, not %d", m.dropEnd(), at)
	}
	// saving with no name is refused, as is saving over a stage's own view; the change stays
	m = drive(m, tea.KeyPressMsg{Code: 's', Mod: tea.ModCtrl})
	if last := keysOf(m); !strings.Contains(last, "a preset needs a name") || !m.changed(0) {
		t.Errorf("save with no name: %q", last)
	}
	m = drive(m, typed("Compact")...)
	if m.savable(0) {
		t.Error("a stage's own name can be saved")
	}
	m = drive(m, tea.KeyPressMsg{Code: 's', Mod: tea.ModCtrl})
	if last := keysOf(m); !strings.Contains(last, "Compact is the stage's own and does not change") || !m.changed(0) {
		t.Errorf("save over Compact: %q", last)
	}
	m = drive(m, esc)
	// the table set as another view names that one
	m.setPanel(m.builtins(0)[0].panel(fieldsOf("backlog"), m.panel(0)))
	if got := presetText(m); got != "   Default ▾" || m.changed(0) {
		t.Errorf("the table as Default: %q", got)
	}
	// the menu's keys: up, a letter to the next view it starts, and esc
	m = drive(m, press(tea.KeyEnter))
	if m = drive(m, typed("c")[0]); m.presetAt != 1 {
		t.Errorf("c in the menu: %d", m.presetAt)
	}
	m = drive(m, up, esc)
	if m.presetMenu || !m.onPreset {
		t.Errorf("esc in the menu: open %v", m.presetMenu)
	}
	// Default shows every column again
	m = drive(m, press(tea.KeyEnter), down, press(tea.KeyEnter), press(tea.KeyEnter), up)
	if m = drive(m, press(tea.KeyEnter)); len(shownNow(m)) != len(fieldsOf("backlog")) || m.changed(0) {
		t.Errorf("Default: %q", shownNow(m))
	}
}

func TestATypedNameSavesThePanelToTheFileAndTheViewComesBack(t *testing.T) {
	file := filepath.Join(t.TempDir(), "kit", "presets.json")
	m := onDropdown(t, file)
	p := m.panel(0).Placed("branch", grid.Hidden).Sorted("age", false).Sorted("age", false).Shown(grid.Others)
	p = p.Grouped("assignees", false)
	p.Fold = false
	m.setPanel(p)
	m.stages[0].headers = false
	// a letter first typed takes the name's place; backspace first takes the name's last letter; esc stops the rename
	m = drive(m, typed("x")...)
	if !m.stages[0].renaming || m.stages[0].naming != "x" || !strings.HasPrefix(presetText(m), "   x▏ ▾") ||
		!m.savable(0) {
		t.Fatalf("typing: %q", presetText(m))
	}
	if m = drive(m, esc); m.stages[0].renaming || !m.onPreset {
		t.Fatalf("esc did not stop the rename, or left the line")
	}
	// the table differs from every view, so the line names none, and backspace first has nothing to take back
	if m = drive(m, press(tea.KeyBackspace)); m.stages[0].naming != "" || m.savable(0) {
		t.Fatalf("backspace first: %q", m.stages[0].naming)
	}
	m = drive(m, esc)
	m = drive(m, typed("Triage")...)
	m = drive(m, tea.KeyPressMsg{Code: 's', Mod: tea.ModCtrl})
	if last := keysOf(m); !strings.Contains(last, "saved the preset Triage") || m.stages[0].viewName != "Triage" ||
		m.changed(0) || !strings.HasPrefix(presetText(m), "   Triage ▾  assignees") {
		t.Fatalf("save: %q, view %q, changed %v, %q", last, m.stages[0].viewName, m.changed(0), presetText(m))
	}
	data, err := os.ReadFile(file)
	var kept presets
	if err != nil || json.Unmarshal(data, &kept) != nil || kept.Version != 2 || len(kept.Stages["backlog"]) != 1 {
		t.Fatalf("the file: %s, %v", data, err)
	}
	// by field id: branch hidden, assignees grouped and filtered to everyone but the caller, age sorted descending
	want := map[string]setting{"branch": {Placement: placeHidden}, "age": {Sort: "desc", Priority: 1},
		"assignees": {Placement: placeGroup, Depth: 1, Filter: &valueFilter{Exclude: []string{caller}}}}
	if v := kept.Stages["backlog"][0]; v.Name != "Triage" || !reflect.DeepEqual(v.Fields, want) || *v.Headers || *v.Fold {
		t.Errorf("the view kept: %s", data)
	}
	// the menu lists it after a rule
	m = drive(m, press(tea.KeyEnter))
	if items := m.presetItems(); len(items) != 4 || items[2].Name != "" || items[3].Name != "Triage" ||
		items[3].Glyph != Checked || m.presetAt != 3 {
		t.Errorf("the menu: %+v, at %d", items, m.presetAt)
	}
	m = drive(m, esc)
	// changed, the line names no view, and nothing is saved until a name is typed; saved again under its own name, it
	// is overwritten
	m.setPanel(m.panel(0).Placed("branch", grid.Column))
	m = drive(m, tea.KeyPressMsg{Code: 's', Mod: tea.ModCtrl})
	if data, _ = os.ReadFile(file); !strings.Contains(string(data), "branch") {
		t.Error("a change was saved with no name")
	}
	m = drive(m, typed("Triage")...)
	m = drive(m, tea.KeyPressMsg{Code: 's', Mod: tea.ModCtrl})
	if data, _ = os.ReadFile(file); strings.Count(string(data), `"name"`) != 1 || strings.Contains(string(data), "branch") {
		t.Errorf("overwritten: %s", data)
	}
	// another session reads it, and choosing it sets the panel
	n := onDropdown(t, file)
	n = drive(n, press(tea.KeyEnter))
	n.presetAt = 3
	if n = drive(n, press(tea.KeyEnter)); n.stages[0].viewName != "Triage" || n.panel(0).Showing() != grid.Others ||
		n.stages[0].headers || len(n.panel(0).SortKeys()) != 1 || !slices.Equal(n.panel(0).Groups(), []string{"assignees"}) ||
		n.panel(0).Fold {
		t.Errorf("Triage in another session: %+v", n.panel(0))
	}
}

func TestClicksOpenThePresetsChooseOneAndSave(t *testing.T) {
	file := filepath.Join(t.TempDir(), "presets.json")
	m := loaded(t, viewing(file), tea.WindowSizeMsg{Width: 140, Height: 40})
	// a click on the line past the name gives it the keyboard, and opens nothing
	if m = drive(m, testkit.Click(m.dropEnd()+4, paneTop)); m.presetMenu || !m.onPreset {
		t.Fatalf("a click past ▾: menu %v, on the line %v", m.presetMenu, m.onPreset)
	}
	m = drive(m, esc)
	// a click on the name drops the presets down, as one on ▾ does
	if m = drive(m, testkit.Click(len(lead)+1, paneTop)); !m.presetMenu || !m.onPreset {
		t.Fatalf("a click on the name: menu %v, on the line %v\n%s", m.presetMenu, m.onPreset, raw(m))
	}
	box, top, left := m.presetBox()
	if m = drive(m, testkit.Click(left+3, top+2)); m.presetMenu || m.stages[0].viewName != "Compact" {
		t.Fatalf("a click on Compact: menu %v, view %q\n%v", m.presetMenu, m.stages[0].viewName, box)
	}
	// a click anywhere else closes the menu
	m = drive(m, testkit.Click(m.dropEnd()-1, paneTop))
	if !m.presetMenu {
		t.Fatalf("a click on ▾ did not open it:\n%s", raw(m))
	}
	if m = drive(m, testkit.Click(1, 20)); m.presetMenu || m.stages[0].viewName != "Compact" {
		t.Errorf("a click off the menu: menu %v", m.presetMenu)
	}
	// with a name of its own, a click on 💾 saves
	m.stages[0].renaming, m.stages[0].naming = true, "Mine"
	if m = drive(m, testkit.Click(m.paneWidth()-2, paneTop)); m.stages[0].viewName != "Mine" {
		t.Errorf("a click on save: view %q, %q", m.stages[0].viewName, keysOf(m))
	}
	if _, err := os.Stat(file); err != nil {
		t.Errorf("nothing saved: %v", err)
	}
}

func TestAPresetsFileThatDoesNotReadIsSaidAndLeftAsItIs(t *testing.T) {
	file := filepath.Join(t.TempDir(), "presets.json")
	for content, says := range map[string]string{"{not json": "does not read as presets",
		`{"version": 3, "stages": {}}`: "version 3, not 2"} {
		if err := os.WriteFile(file, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
		m := loaded(t, viewing(file), tea.WindowSizeMsg{Width: 140, Height: 40})
		if !strings.Contains(m.notice, says) || m.noted || len(m.views(0)) != 2 {
			t.Errorf("a bad file: %q", m.notice)
		}
		if data, _ := os.ReadFile(file); string(data) != content {
			t.Errorf("the bad file was changed: %s", data)
		}
	}
	m := loaded(t, viewing(file), tea.WindowSizeMsg{Width: 140, Height: 40})
	// an empty name saves nothing; a file that cannot be written is said
	m = drive(m, down)
	m.stages[0].renaming, m.stages[0].naming = true, "  "
	if m = drive(m, tea.KeyPressMsg{Code: 's', Mod: tea.ModCtrl}); !strings.Contains(lines(m)[len(lines(m))-1], "needs a name") {
		t.Errorf("an empty name: %q", lines(m)[len(lines(m))-1])
	}
	m.presetsPath = filepath.Join(file, "under-a-file", "presets.json")
	m.stages[0].renaming, m.stages[0].naming = true, "Mine"
	if m = drive(m, tea.KeyPressMsg{Code: 's', Mod: tea.ModCtrl}); !strings.Contains(lines(m)[len(lines(m))-1], "could not be saved") {
		t.Errorf("an unwritable file: %q", lines(m)[len(lines(m))-1])
	}
	if got := presetsFile(); !strings.HasSuffix(got, filepath.Join("kit", "presets.json")) {
		t.Errorf("the presets file: %q", got)
	}
	t.Setenv("XDG_CONFIG_HOME", "/somewhere")
	if got := presetsFile(); got != "/somewhere/kit/presets.json" {
		t.Errorf("under XDG_CONFIG_HOME: %q", got)
	}
}

// A view the person saved as Everything or Minimal, which the stages once had of their own, is the person's: it is
// listed after the stage's own, and saving over it overwrites it.
func TestAViewNamedAsAStageOnceNamedItsOwnIsThePersons(t *testing.T) {
	file := filepath.Join(t.TempDir(), "presets.json")
	kept := `{"version": 2, "stages": {"backlog": [{"name": "Everything", "headers": true},
		{"name": "Minimal", "fields": {"age": {"placement": "hidden"}}, "headers": true}]}}`
	if err := os.WriteFile(file, []byte(kept), 0o644); err != nil {
		t.Fatal(err)
	}
	m := onDropdown(t, file)
	var names []string
	for _, v := range m.views(0) {
		names = append(names, v.Name)
	}
	if !slices.Equal(names, []string{"Default", "Compact", "Everything", "Minimal"}) || m.builtin(0, "Minimal") {
		t.Fatalf("the views: %q", names)
	}
	m.setPanel(m.panel(0).Placed("branch", grid.Hidden))
	m = drive(m, typed("Minimal")...)
	m = drive(m, tea.KeyPressMsg{Code: 's', Mod: tea.ModCtrl})
	if data, _ := os.ReadFile(file); !strings.Contains(string(data), `"branch"`) || strings.Contains(string(data), `"age"`) {
		t.Errorf("Minimal was not overwritten: %s", data)
	}
}

// A search that gives up the keyboard with something typed is remembered, in the file every screen shares; with the
// keyboard again, ctrl+r or a click on ▾ drops the searches down, and choosing one puts it in the search.
func TestASearchIsRememberedAndRecalled(t *testing.T) {
	file := filepath.Join(t.TempDir(), "search-history.json")
	o := options()
	o.History = file
	m := loaded(t, o, tea.WindowSizeMsg{Width: 140, Height: 30})
	m = drive(m, append(typed("/fix"), esc)...)
	if data, err := os.ReadFile(file); err != nil || !strings.Contains(string(data), `"fix"`) {
		t.Fatalf("the search was not remembered: %s, %v", data, err)
	}
	m = drive(m, append(typed("/"), press(tea.KeyBackspace), press(tea.KeyBackspace), press(tea.KeyBackspace))...)
	if top := strings.Split(raw(m), "\n")[0]; !strings.Contains(top, shell.Recall) ||
		!strings.Contains(lines(m)[len(lines(m))-1], "⌃R recent") {
		t.Fatalf("no recall with the keyboard on the search: %q", top)
	}
	m = drive(m, tea.KeyPressMsg{Code: 'r', Mod: tea.ModCtrl})
	if top := strings.Split(raw(m), "\n"); !m.recent.Open || !strings.Contains(top[1], "fix") || !strings.Contains(lines(m)[len(lines(m))-1], "esc close") {
		t.Fatalf("ctrl+r did not drop the searches down:\n%s", view(m))
	}
	if m = drive(m, press(tea.KeyEnter)); m.recent.Open || m.panel(m.stage).Query != "fix" {
		t.Errorf("choosing fix: %q", m.panel(m.stage).Query)
	}
	// a click on ▾ opens it; a click on a search chooses it; one elsewhere closes it
	m = drive(m, press(tea.KeyBackspace))
	m = drive(m, testkit.Click(m.recallAt(), 0))
	if !m.recent.Open {
		t.Fatalf("a click on ▾ did not open the searches:\n%s", view(m))
	}
	box, left := m.recent.Box(m.recallAt(), m.width)
	if m = drive(m, testkit.Click(left+3, 1)); m.recent.Open || m.panel(m.stage).Query != "fix" {
		t.Errorf("a click on fix: %q %v", m.panel(m.stage).Query, box)
	}
	m = drive(m, testkit.Click(m.recallAt(), 0), testkit.Click(1, 20))
	if m.recent.Open {
		t.Error("a click off the searches left them open")
	}
}

// With no presets.json, the views.json beside it is read once into presets.json, version 2, and left as it is: each
// column's label its field's id, the order the columns' positions, mine the filter on the assignees of the caller
// alone, others of anyone but the caller, the columns kept from grouping on their own dropped, and each label the
// catalog does not know dropped, which the notice line says, once.
func TestViewsJSONBecomesPresetsJSONOnce(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "kit")
	v1, err := os.ReadFile(filepath.Join("testdata", "views-v1.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil || os.WriteFile(filepath.Join(dir, "views.json"), v1, 0o644) != nil {
		t.Fatal(err)
	}
	file := filepath.Join(dir, "presets.json")
	m := loaded(t, viewing(file))
	if !m.noted || m.notice != Save+" views.json is now presets.json, without the columns kittui no longer knows: "+
		"milestone in backlog Mine; estimate in backlog All; milestone in changes Authors" {
		t.Errorf("the notice: %q", m.notice)
	}
	if data, _ := os.ReadFile(filepath.Join(dir, "views.json")); string(data) != string(v1) {
		t.Errorf("views.json was changed: %s", data)
	}
	data, err := os.ReadFile(file)
	var kept presets
	if err != nil || json.Unmarshal(data, &kept) != nil || kept.Version != 2 {
		t.Fatalf("presets.json: %s, %v", data, err)
	}
	yes, no := true, false
	want := map[string][]preset{
		"backlog": {
			{Name: "Mine", Headers: &yes, Fold: &yes, Fields: map[string]setting{
				"title":     {Placement: placeColumn, Position: 1},
				"id":        {Placement: placeColumn, Position: 2},
				"source":    {Placement: placeColumn, Position: 3},
				"assignees": {Placement: placeColumn, Position: 4, Filter: &valueFilter{Include: []string{caller}}},
				"age":       {Placement: placeColumn, Position: 5, Sort: "desc", Priority: 1},
				"branch":    {Placement: placeHidden},
			}},
			{Name: "Others", Headers: &no, Fold: &yes, Fields: map[string]setting{
				"assignees": {Placement: placeGroup, Depth: 1, Filter: &valueFilter{Exclude: []string{caller}}},
			}},
			{Name: "All", Headers: &yes, Fold: &yes},
		},
		"changes": {
			{Name: "Authors", Headers: &yes, Fold: &yes, Fields: map[string]setting{"author": {Placement: placeGroup, Depth: 1}}},
		},
	}
	if !reflect.DeepEqual(kept.Stages, want) || !reflect.DeepEqual(m.saved, want) {
		t.Errorf("presets.json:\n%s", data)
	}
	// each show is the owner's filter: mine, others, and all
	for k, show := range []grid.Show{grid.Mine, grid.Others, grid.All} {
		if got := m.saved["backlog"][k].panel(fieldsOf("backlog"), grid.Panel{Owner: "assignees"}).Showing(); got != show {
			t.Errorf("%s shows %v, not %v", m.saved["backlog"][k].Name, got, show)
		}
	}
	// read again, presets.json is what is read, and nothing is noted
	if n := loaded(t, viewing(file)); n.notice != "" || !reflect.DeepEqual(n.saved, want) {
		t.Errorf("read again: %q, %+v", n.notice, n.saved)
	}
	// a stage's fields a preset does not name take the catalog's placement
	if g := want["changes"][0].panel(fieldsOf("changes"), grid.Panel{}); g.Field("branch").Placement != grid.Column ||
		!g.Shows("title") {
		t.Errorf("Authors' other fields: %+v", g.Fields)
	}
}
