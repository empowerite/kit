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
	"regexp"
	"slices"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/empowerite/kit/projects/tui/internal/grid"
	"github.com/empowerite/kit/projects/tui/internal/shell"
	"github.com/empowerite/kit/projects/tui/internal/style"
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

// presetText gives the preset line as the screen draws it, the rule under the header, without its escapes, its run of
// ─ and the sidebar after it apart.
func presetText(m Model) string {
	line := ansi.Strip(m.rule())
	return strings.TrimRight(regexp.MustCompile(` ?─+ ?`).ReplaceAllString(line, " "), " ")
}

// shown gives the columns a stage's panel shows.
func shownNow(m Model) []string { return m.shownColumns(m.stage) }

// keysOf gives the keys bar as the screen draws it, without its escapes.
func keysOf(m Model) string { return lines(m)[len(lines(m))-1] }

// The preset line is the rule under the header, so it takes no line of the rows: the names of the columns stand on
// the pane's first line, the line under them names the view, and the rows start under it.
func TestThePresetLineIsTheHeadersRuleAndNamesTheView(t *testing.T) {
	m := loaded(t, viewing(""), tea.WindowSizeMsg{Width: 140, Height: 40}, press('?'))
	if got := presetText(m); got != "   Default "+Presets || m.presetTop() != paneTop+1 || m.rowsTop() != paneTop+2 ||
		!strings.Contains(lines(m)[paneTop], "assignees") {
		t.Errorf("the preset line: %q, on line %d, rows from %d", got, m.presetTop(), m.rowsTop())
	}
	// the selector stands at its right end, after how many rows are selected
	m.stages[0].picked[m.rows(0)[1].Key] = true
	if got := ansi.Truncate(ansi.Strip(lines(m)[m.presetTop()]), m.paneWidth(), ""); !strings.HasPrefix(got, "   ─") ||
		!strings.HasSuffix(got, "─ 1 selected ─ Default ▼") {
		t.Errorf("one row selected: %q", got)
	}
	m.stages[0].picked = map[string]bool{}
	if !strings.Contains(keysOf(m), "↓ preset") {
		t.Errorf("down on the bar does not say it reaches the preset line: %q", keysOf(m))
	}
	// the sidebar under the search starts with the field editor
	side, _ := m.panel(0).Lines()
	if got := ansi.Strip(side[2]); got != "group by" || ansi.Strip(side[1]) != "" {
		t.Errorf("the sidebar's first lines: %q", side[:3])
	}
	m = drive(m, down)
	keys := keysOf(m)
	for _, want := range []string{"↑·esc backlog", "↓ rows", "⏎ presets"} {
		if !strings.Contains(keys, want) {
			t.Errorf("the keys on the preset line lack %q: %q", want, keys)
		}
	}
	if strings.Contains(keys, "save") || strings.Contains(keys, "rename") {
		t.Errorf("the line names a save or a rename before Save As…: %q", keys)
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
	if !m.inPanel || !strings.Contains(keysOf(m), "⇥·esc preset") {
		t.Fatalf("tab from the line: in panel %v, %q", m.inPanel, keysOf(m))
	}
	if m = drive(m, esc); m.inPanel || !m.onPreset {
		t.Errorf("esc from the sidebar: in panel %v, on the line %v", m.inPanel, m.onPreset)
	}
	// naming for Save As…, a number is typed, as a name may hold one; a click on a row leaves the line
	m.choosePreset(itemSaveAs)
	if m = drive(m, typed("2")...); !m.onPreset || m.stage != 0 || m.stages[0].naming != "2" {
		t.Errorf("a number on the line: on the line %v, stage %d, %q", m.onPreset, m.stage, m.stages[0].naming)
	}
	if m = drive(m, testkit.Click(6, m.rowsTop()+1)); m.onPreset || !m.inRows {
		t.Errorf("a click on a row: on the line %v", m.onPreset)
	}
	// off Save As…, a number on the line selects the stage it numbers, as on the bar
	m = drive(m, up, up)
	if m = drive(m, typed("2")...); m.onPreset || m.stage != 1 {
		t.Errorf("a number on the line, not naming: on the line %v, stage %d", m.onPreset, m.stage)
	}
}

func TestAChipStandsForEachFilter(t *testing.T) {
	m := onDropdown(t, "")
	m.setPanel(m.panel(0).Shown(grid.Mine))
	if got := presetText(m); !strings.Contains(got, "  assignees: @me") || !strings.HasSuffix(got, "@me  Default "+Presets) {
		t.Errorf("mine: %q", got)
	}
	m.setPanel(m.panel(0).Shown(grid.Others))
	if got := presetText(m); !strings.Contains(got, "  assignees: ¬@me") {
		t.Errorf("others: %q", got)
	}
	// too narrow, the chips are clipped before the selector is
	m = drive(m, tea.WindowSizeMsg{Width: 30, Height: 40})
	if got := presetText(m); !strings.HasSuffix(got, " "+Presets) || ansi.StringWidth(got) > m.paneWidth() {
		t.Errorf("narrow: %q", got)
	}
}

func TestChoosingAViewSetsThePanelAndAChangeItalicizesTheName(t *testing.T) {
	m := onDropdown(t, "")
	m = drive(m, press(tea.KeyEnter))
	if !m.presetMenu {
		t.Fatal("enter did not drop the presets down")
	}
	// the menu is titled Presets, not by the preset the table is set as
	if box, _, _ := m.presetBox(); !strings.Contains(ansi.Strip(box[0]), "╴"+PresetsTitle+"╶") ||
		strings.Contains(box[0], Opening) {
		t.Errorf("the menu's title: %q", ansi.Strip(box[0]))
	}
	// the menu is drawn, under the line, whole
	if drawn := raw(m); !strings.Contains(drawn, SaveAs) || !strings.Contains(drawn, "Compact") {
		t.Fatalf("the presets are not drawn:\n%s", drawn)
	}
	names := []string{}
	for _, i := range m.presetItems() {
		names = append(names, i.Name)
	}
	if !slices.Equal(names, []string{SaveIt, SaveAs, RenameIt, DeleteIt, "", "Default", "Compact"}) ||
		m.presetItems()[5].Glyph != Checked || m.presetAt != 5 {
		t.Fatalf("the presets: %q, at %d", names, m.presetAt)
	}
	if !strings.Contains(keysOf(m), "esc close") {
		t.Errorf("the keys with the menu open: %q", keysOf(m))
	}
	m = drive(m, down, press(tea.KeyEnter))
	if m.presetMenu || m.stages[0].viewName != "Compact" || !slices.Equal(shownNow(m), []string{"id", "title"}) ||
		presetText(m) != "   Compact "+Presets {
		t.Fatalf("Compact: %q, %q", shownNow(m), presetText(m))
	}
	// a column shown again differs from every view: the line names the one it was set to, in italics
	m.setPanel(m.panel(0).Placed("age", grid.Column))
	color := stageColors["backlog"]
	if got := presetText(m); m.is(0) != "" || got != "   Compact "+Presets ||
		!strings.Contains(m.rule(), style.Named("Compact", color, true, true)) {
		t.Errorf("a change does not italicize the view: %q", got)
	}
	if m.onPreset = false; !strings.Contains(m.rule(), style.Named("Compact", color, true, false)) {
		t.Errorf("off the line, the view is not in italics, in the tab's color: %q", m.rule())
	}
	m.onPreset = true
	if m.dropEnd() != m.dropStart()+len("Compact ")+1 || m.dropStart() < m.paneWidth()/2 {
		t.Errorf("▼ from %d to %d", m.dropStart(), m.dropEnd())
	}
	// a stage's own view is never saved over, renamed, or deleted: those lines are dim
	if m.presetEnabled(itemSave) || m.presetEnabled(itemRename) || m.presetEnabled(itemDelete) {
		t.Error("Save, Rename…, or Delete acts on a stage's own view")
	}
	// the menu then opens on Save As…, no name typed for a stage's own view; saving with no name is refused, as is
	// saving over a stage's own view; the change stays
	if m = drive(m, press(tea.KeyEnter)); m.presetAt != itemSaveAs {
		t.Errorf("the menu on a changed table opens at %d", m.presetAt)
	}
	m = drive(m, press(tea.KeyEnter))
	if !m.stages[0].savingAs || m.presetMenu || !strings.HasSuffix(presetText(m), SavingAs+" ▏") ||
		!strings.Contains(keysOf(m), "⏎ save") {
		t.Fatalf("Save As…: %q, %q", presetText(m), keysOf(m))
	}
	m = drive(m, press(tea.KeyEnter))
	if last := keysOf(m); !strings.Contains(last, "a preset needs a name") || m.is(0) != "" {
		t.Errorf("save with no name: %q", last)
	}
	m = drive(m, typed("Compact")...)
	m = drive(m, press(tea.KeyEnter))
	if last := keysOf(m); !strings.Contains(last, "Compact is the stage's own and does not change") || m.is(0) != "" {
		t.Errorf("save over Compact: %q", last)
	}
	if m = drive(m, esc); m.stages[0].savingAs || !m.onPreset || presetText(m) != "   Compact "+Presets {
		t.Errorf("esc did not stop Save As…, or left the line: %q", presetText(m))
	}
	// the table set as another view names that one
	m.setPanel(m.builtins(0)[0].panel(fieldsOf("backlog"), m.panel(0)))
	if got := presetText(m); got != "   Default "+Presets {
		t.Errorf("the table as Default: %q", got)
	}
	// the menu's keys: up, a letter to the next item it starts, and esc
	m = drive(m, press(tea.KeyEnter))
	if m = drive(m, typed("c")[0]); m.presetAt != 6 {
		t.Errorf("c in the menu: %d", m.presetAt)
	}
	m = drive(m, up, esc)
	if m.presetMenu || !m.onPreset {
		t.Errorf("esc in the menu: open %v", m.presetMenu)
	}
	// Default shows every column the catalog places, again
	m = drive(m, press(tea.KeyEnter), down, press(tea.KeyEnter), press(tea.KeyEnter), up)
	columns := slices.DeleteFunc(slices.Clone(fieldsOf("backlog")), func(f field) bool { return f.Placement != placeColumn })
	if m = drive(m, press(tea.KeyEnter)); len(shownNow(m)) != len(columns) || m.is(0) != "Default" {
		t.Errorf("Default: %q", shownNow(m))
	}
}

func TestSaveAsKeepsThePanelInTheFileAndTheViewComesBack(t *testing.T) {
	file := filepath.Join(t.TempDir(), "kit", "presets.json")
	m := onDropdown(t, file)
	p := m.panel(0).Placed("branch", grid.Hidden).Sorted("age", false).Sorted("age", false).Shown(grid.Others)
	p = p.Grouped("assignees", false)
	p.Fold = false
	m.setPanel(p)
	// off Save As…, what is typed names nothing
	if m = drive(m, typed("x")...); m.stages[0].savingAs || m.stages[0].naming != "" {
		t.Fatalf("typing off Save As…: %q", m.stages[0].naming)
	}
	// Save As… from the menu, then a name, backspace, and enter
	m = drive(m, press(tea.KeyEnter), press(tea.KeyEnter))
	m = drive(m, typed("Triagex")...)
	if m = drive(m, press(tea.KeyBackspace)); m.stages[0].naming != "Triage" ||
		!strings.HasSuffix(presetText(m), SavingAs+" Triage▏") {
		t.Fatalf("typing: %q", presetText(m))
	}
	m = drive(m, press(tea.KeyEnter))
	if last := keysOf(m); !strings.Contains(last, "saved the preset Triage") || m.stages[0].viewName != "Triage" ||
		m.stages[0].savingAs || !strings.HasSuffix(presetText(m), "assignees: ¬@me  Triage "+Presets) {
		t.Fatalf("save: %q, view %q, %q", last, m.stages[0].viewName, presetText(m))
	}
	data, err := os.ReadFile(file)
	var kept presets
	if err != nil || json.Unmarshal(data, &kept) != nil || kept.Version != 2 || len(kept.Stages["backlog"]) != 1 {
		t.Fatalf("the file: %s, %v", data, err)
	}
	// by field id: branch hidden, assignees grouped and filtered to everyone but the caller, age sorted descending
	want := map[string]setting{"branch": {Placement: placeHidden}, "age": {Sort: "desc", Priority: 1},
		"assignees": {Placement: placeGroup, Depth: 1, Filter: &valueFilter{Exclude: []string{caller}}}}
	if v := kept.Stages["backlog"][0]; v.Name != "Triage" || !reflect.DeepEqual(v.Fields, want) || *v.Fold {
		t.Errorf("the view kept: %s", data)
	}
	// the menu lists it after a rule
	m = drive(m, press(tea.KeyEnter))
	if items := m.presetItems(); len(items) != 9 || items[7].Name != "" || items[8].Name != "Triage" ||
		items[8].Glyph != Checked || m.presetAt != 8 {
		t.Errorf("the menu: %+v, at %d", items, m.presetAt)
	}
	m = drive(m, esc)
	// changed, then Save As… again, its name typed already, it is overwritten; up while naming stops, and leaves the
	// line
	m.setPanel(m.panel(0).Placed("branch", grid.Column))
	m.choosePreset(itemSaveAs)
	if !m.stages[0].savingAs || m.stages[0].naming != "Triage" || !strings.HasSuffix(presetText(m), SavingAs+" Triage▏") {
		t.Fatalf("Save As… on Triage: %q", presetText(m))
	}
	m = drive(m, press(tea.KeyEnter))
	if data, _ = os.ReadFile(file); strings.Count(string(data), `"name"`) != 1 || strings.Contains(string(data), "branch") {
		t.Errorf("overwritten: %s", data)
	}
	m.choosePreset(itemSaveAs)
	if m = drive(m, up); m.stages[0].savingAs || m.onPreset {
		t.Errorf("up while naming: naming %v, on the line %v", m.stages[0].savingAs, m.onPreset)
	}
	// another session reads it, and choosing it sets the panel
	n := onDropdown(t, file)
	n = drive(n, press(tea.KeyEnter))
	n.presetAt = 8
	if n = drive(n, press(tea.KeyEnter)); n.stages[0].viewName != "Triage" || n.panel(0).Showing() != grid.Others ||
		len(n.panel(0).SortKeys()) != 1 || !slices.Equal(n.panel(0).Groups(), []string{"assignees"}) ||
		n.panel(0).Fold {
		t.Errorf("Triage in another session: %+v", n.panel(0))
	}
}

func TestClicksOpenThePresetsAndChooseOne(t *testing.T) {
	file := filepath.Join(t.TempDir(), "presets.json")
	m := loaded(t, viewing(file), tea.WindowSizeMsg{Width: 140, Height: 40})
	// a click on the line before the name gives it the keyboard, and opens nothing
	if m = drive(m, testkit.Click(m.dropStart()-4, m.presetTop())); m.presetMenu || !m.onPreset {
		t.Fatalf("a click past the name: menu %v, on the line %v", m.presetMenu, m.onPreset)
	}
	m = drive(m, esc)
	// a click on the name drops the presets down, as one on ▼ does
	if m = drive(m, testkit.Click(m.dropStart()+1, m.presetTop())); !m.presetMenu || !m.onPreset {
		t.Fatalf("a click on the name: menu %v, on the line %v\n%s", m.presetMenu, m.onPreset, raw(m))
	}
	box, top, left := m.presetBox()
	if m = drive(m, testkit.Click(left+3, top+7)); m.presetMenu || m.stages[0].viewName != "Compact" {
		t.Fatalf("a click on Compact: menu %v, view %q\n%v", m.presetMenu, m.stages[0].viewName, box)
	}
	// a click anywhere else closes the menu
	m = drive(m, testkit.Click(m.dropEnd()-1, m.presetTop()))
	if !m.presetMenu {
		t.Fatalf("a click on ▼ did not open it:\n%s", raw(m))
	}
	if m = drive(m, testkit.Click(1, 20)); m.presetMenu || m.stages[0].viewName != "Compact" {
		t.Errorf("a click off the menu: menu %v", m.presetMenu)
	}
	// a click on Save, dim on a stage's own view, does nothing; a click on Save As… takes a name, and enter saves
	// under it
	m = drive(m, testkit.Click(m.dropStart(), m.presetTop()))
	_, top, left = m.presetBox()
	if m = drive(m, testkit.Click(left+3, top+1)); m.stages[0].savingAs || m.presetMenu {
		t.Errorf("a click on Save, dim: naming %v, menu %v", m.stages[0].savingAs, m.presetMenu)
	}
	m = drive(m, testkit.Click(m.dropStart(), m.presetTop()))
	m = drive(m, testkit.Click(left+3, top+2))
	m = drive(m, typed("Mine")...)
	if m = drive(m, press(tea.KeyEnter)); m.stages[0].viewName != "Mine" {
		t.Errorf("Save As… by clicks: view %q, %q", m.stages[0].viewName, keysOf(m))
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
	m.stages[0].savingAs, m.stages[0].naming = true, "  "
	if m = drive(m, press(tea.KeyEnter)); !strings.Contains(lines(m)[len(lines(m))-1], "needs a name") {
		t.Errorf("an empty name: %q", lines(m)[len(lines(m))-1])
	}
	m.presetsPath = filepath.Join(file, "under-a-file", "presets.json")
	m.stages[0].savingAs, m.stages[0].naming = true, "Mine"
	if m = drive(m, press(tea.KeyEnter)); !strings.Contains(lines(m)[len(lines(m))-1], "could not be saved") {
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
	m.choosePreset(itemSaveAs)
	m = drive(m, typed("Minimal")...)
	m = drive(m, press(tea.KeyEnter))
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
	yes := true
	want := map[string][]preset{
		"backlog": {
			{Name: "Mine", Fold: &yes, Fields: map[string]setting{
				"title":     {Placement: placeColumn, Position: 1},
				"id":        {Placement: placeColumn, Position: 2},
				"source":    {Placement: placeColumn, Position: 3},
				"assignees": {Placement: placeColumn, Position: 4, Filter: &valueFilter{Include: []string{caller}}},
				"age":       {Placement: placeColumn, Position: 5, Sort: "desc", Priority: 1},
				"branch":    {Placement: placeHidden},
			}},
			{Name: "Others", Fold: &yes, Fields: map[string]setting{
				"assignees": {Placement: placeGroup, Depth: 1, Filter: &valueFilter{Exclude: []string{caller}}},
			}},
			{Name: "All", Fold: &yes},
		},
		"changes": {
			{Name: "Authors", Fold: &yes, Fields: map[string]setting{"author": {Placement: placeGroup, Depth: 1}}},
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

// A stage without columns has Default alone, and a view the line was set to that is gone gives way to Default.
func TestAStageWithoutColumnsHasDefaultAloneAndAViewGoneGivesWayToIt(t *testing.T) {
	m := loaded(t, options())
	incoming := slices.Index(eight, "incoming")
	if vs := m.builtins(incoming); len(vs) != 1 || vs[0].Name != Opening {
		t.Errorf("incoming's views: %+v", vs)
	}
	m.stages[0].viewName = "Gone"
	if v := m.chosen(0); v.Name != Opening {
		t.Errorf("a view gone: %q", v.Name)
	}
}

// Save keeps the panel under the person's view the line was set to, once it differs; Rename… takes a new name for
// it, its own typed already, and refuses a stage's own and another of theirs; Delete drops it, and the stage is
// Default again. Each says what it did.
func TestSaveRenameAndDeleteActOnThePersonsView(t *testing.T) {
	file := filepath.Join(t.TempDir(), "kit", "presets.json")
	m := onDropdown(t, file)
	name := func(n string) Model {
		m.choosePreset(itemSaveAs)
		m.stages[0].naming = n
		return drive(m, press(tea.KeyEnter))
	}
	m.setPanel(m.panel(0).Placed("branch", grid.Hidden))
	m = name("Other")
	m.setPanel(m.panel(0).Placed("age", grid.Hidden))
	m = name("Triage")
	if m.presetEnabled(itemSave) || !m.presetEnabled(itemRename) || !m.presetEnabled(itemDelete) {
		t.Fatalf("on Triage as it is: save %v, rename %v, delete %v", m.presetEnabled(itemSave),
			m.presetEnabled(itemRename), m.presetEnabled(itemDelete))
	}
	// changed, Save overwrites it
	m.setPanel(m.panel(0).Placed("age", grid.Column))
	if !m.presetEnabled(itemSave) {
		t.Fatal("Save is dim on a changed Triage")
	}
	m.choosePreset(itemSave)
	if v, _ := m.own(); v.Name != "Triage" || !m.same(0, v, m.current(0)) || !strings.Contains(keysOf(m), Save+" saved the preset Triage") {
		t.Errorf("Save: %q", keysOf(m))
	}
	// Rename… types its name already; a stage's own name and another of the person's are refused
	m.choosePreset(itemRename)
	if !strings.HasSuffix(presetText(m), Renaming+" Triage▏") || !strings.Contains(m.keys().String(), "⏎ rename") {
		t.Fatalf("Rename…: %q, %q", presetText(m), m.keys().String())
	}
	for _, refused := range []string{"Compact", "Other"} {
		m.stages[0].naming = refused
		if m = drive(m, press(tea.KeyEnter)); !strings.Contains(m.notice, refused+" is ") || m.stages[0].viewName != "Triage" {
			t.Errorf("rename to %s: %q", refused, m.notice)
		}
	}
	m.stages[0].naming = "Mine"
	m = drive(m, press(tea.KeyEnter))
	if data, _ := os.ReadFile(file); m.stages[0].viewName != "Mine" || strings.Contains(string(data), "Triage") ||
		!strings.Contains(string(data), `"Mine"`) || !strings.Contains(m.notice, Renamed+" renamed the preset Triage to Mine") {
		t.Errorf("renamed: view %q, %q, %s", m.stages[0].viewName, m.notice, data)
	}
	// Delete drops it, and the stage is Default
	m.choosePreset(itemDelete)
	if data, _ := os.ReadFile(file); strings.Contains(string(data), "Mine") || m.is(0) != Opening ||
		!strings.Contains(m.notice, Deleted+" deleted the preset Mine") {
		t.Errorf("deleted: %q, is %q, %s", m.notice, m.is(0), data)
	}
	if m.presetEnabled(itemRename) || m.presetEnabled(itemDelete) {
		t.Error("Rename… or Delete acts on Default")
	}
}

// A notice that says what was done goes once its time is up, short of a key; one that says why not stays, and a
// time up for an earlier notice leaves a later one.
func TestANoticeOfWhatWasDoneFades(t *testing.T) {
	m := onDropdown(t, filepath.Join(t.TempDir(), "presets.json"))
	m.choosePreset(itemSaveAs)
	m.stages[0].naming = "Mine"
	m = drive(m, press(tea.KeyEnter))
	first := m.fades
	if !strings.Contains(keysOf(m), "saved the preset Mine") {
		t.Fatalf("the notice: %q", keysOf(m))
	}
	m.done(Save + " another")
	if m = drive(m, fadedMsg{first}); m.notice == "" {
		t.Error("a time up for an earlier notice took a later one")
	}
	if m = drive(m, fadedMsg{m.fades}); m.notice != "" {
		t.Errorf("the notice stayed: %q", m.notice)
	}
	m.choosePreset(itemSaveAs)
	m.stages[0].naming = ""
	m = drive(m, press(tea.KeyEnter))
	if m = drive(m, fadedMsg{m.fades}); !strings.Contains(m.notice, "needs a name") {
		t.Errorf("a refusal faded: %q", m.notice)
	}
}

// Rename… with no name is refused, and with the same name stops, the view as it was; Rename… and Delete that cannot
// write the file say so and change nothing, as does a file that cannot be written beside or replaced; Delete on a
// stage's own view does nothing.
func TestRenameAndDeleteRefuseWhatTheyCannotDo(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "presets.json")
	m := onDropdown(t, file)
	if m.choosePreset(itemDelete); m.is(0) != Opening || m.notice != "" {
		t.Errorf("Delete on Default: %q", m.notice)
	}
	m.choosePreset(itemSaveAs)
	m.stages[0].naming = "Mine"
	m = drive(m, press(tea.KeyEnter))
	rename := func(name string) Model {
		m.choosePreset(itemRename)
		m.stages[0].naming = name
		return drive(m, press(tea.KeyEnter))
	}
	if m = rename(" "); !strings.Contains(m.notice, "needs a name") || !m.stages[0].savingAs {
		t.Errorf("rename to nothing: %q", m.notice)
	}
	m = drive(m, esc)
	if m = rename("Mine"); m.stages[0].savingAs || m.stages[0].viewName != "Mine" || strings.Contains(m.notice, "renamed") {
		t.Errorf("rename to the same name: naming %v, %q", m.stages[0].savingAs, m.notice)
	}
	m.presetsPath = filepath.Join(file, "under-a-file", "presets.json")
	if m = rename("Yours"); !strings.Contains(m.notice, "could not be renamed") || m.stages[0].viewName != "Mine" {
		t.Errorf("rename, unwritable: %q", m.notice)
	}
	m = drive(m, esc)
	if m.choosePreset(itemDelete); !strings.Contains(m.notice, "could not be deleted") || m.stages[0].viewName != "Mine" {
		t.Errorf("delete, unwritable: %q", m.notice)
	}
	// the file cannot be written beside, as a directory stands where it would be; or replaced, as it is a directory
	blocked := filepath.Join(dir, "blocked.json")
	if err := os.MkdirAll(filepath.Join(blocked+".new", "x"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := writePresets(blocked, nil); err == nil {
		t.Error("a file written where a directory stands")
	}
	if err := os.MkdirAll(filepath.Join(dir, "taken.json", "x"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := writePresets(filepath.Join(dir, "taken.json"), nil); err == nil {
		t.Error("a directory replaced by the file")
	}
}

// The preset line names the preset in its tab's color, as the tab is named, the preset being the tab's; while a stage
// loads, or once it refused, there is no header, and neither the rule nor the preset line stands.
func TestThePresetLineIsTheTabsColorAndWaitsForTheRows(t *testing.T) {
	m := New(runner(t), viewing(""))
	if lines, _ := m.header(); lines != nil || m.presets(0) || strings.Contains(view(m), Presets) {
		t.Errorf("while backlog loads: header %q, preset line %v\n%s", lines, m.presets(0), view(m))
	}
	m = loaded(t, viewing(""))
	if !m.presets(0) || !strings.Contains(m.rule(), style.Named(Opening, stageColors["backlog"], false, false)+" "+Presets) {
		t.Errorf("backlog's preset line: %q", m.rule())
	}
}
