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
	"github.com/empowerite/kit/projects/tui/internal/style"
	"github.com/empowerite/kit/projects/tui/internal/table"
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

// presetText gives the preset line as the screen draws it, at the control line's right end, without its escapes:
// what follows the heading over the groups.
func presetText(m Model) string {
	line, _ := m.control()
	return strings.TrimSpace(strings.TrimPrefix(ansi.Strip(line), strings.Repeat(" ", table.BarLeft)+ansi.Strip(m.barText())))
}

// selector gives the preset line's selector as the screen draws it, without its escapes: the name and Presets between
// the pill's half circles.
func selector(name string) string { return style.PillLeft + name + " " + Presets + style.PillRight }

// shown gives the columns a stage's panel shows.
func shownNow(m Model) []string { return m.shownColumns(m.stage) }

// keysOf gives the keys bar as the screen draws it, without its escapes.
func keysOf(m Model) string { return lines(m)[len(lines(m))-1] }

// The preset line stands at the right end of the control line, the pane's first, over the rows, the heading over the
// groups at its left: it names the view, and the rows start under it, the names of the columns over the first row.
func TestThePresetLineStandsOnTheControlLineAndNamesTheView(t *testing.T) {
	m := loaded(t, viewing(""), tea.WindowSizeMsg{Width: 140, Height: 40}, press('?'))
	if got := presetText(m); got != selector(Opening) || m.presetTop() != m.top() || m.rowsTop() != m.top()+2 ||
		!strings.Contains(lines(m)[m.headerTop()], "assignees") || !strings.Contains(lines(m)[m.top()], "source") {
		t.Errorf("the preset line: %q, on line %d, rows from %d", got, m.presetTop(), m.rowsTop())
	}
	// the selector stands at its right end, after how many rows are selected, which the rule no longer says
	m.stages[0].Picked[m.rows(0)[slices.IndexFunc(m.rows(0), func(r row) bool { return !r.Aside })].Key] = true
	if got := ansi.Truncate(ansi.Strip(lines(m)[m.presetTop()]), m.paneWidth(), ""); !strings.HasPrefix(got, "  ▾ source") ||
		!strings.HasSuffix(got, " 1 selected  "+selector(Opening)) || strings.Contains(m.rule(), "selected") {
		t.Errorf("one row selected: %q", got)
	}
	m.stages[0].Picked = map[string]bool{}
	if !strings.Contains(keysOf(m), "↓ preset") {
		t.Errorf("down on the bar does not say it reaches the preset line: %q", keysOf(m))
	}
	m = drive(m, down)
	keys := keysOf(m)
	for _, want := range []string{"↑·esc tabs", "↓ rows", "⏎ presets"} {
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
	// up past source's heading, over the first row, the heading over the groups standing on the control line
	if m = drive(m, up, up); !m.onPreset || m.inRows {
		t.Fatalf("up from the first row: in rows %v, on the line %v", m.inRows, m.onPreset)
	}
	if m = drive(m, up); m.onPreset || !m.bar().Focused {
		t.Fatalf("up from the line: on the line %v", m.onPreset)
	}
	if m = drive(m, down, esc); m.onPreset || !m.bar().Focused {
		t.Fatalf("esc from the line: on the line %v", m.onPreset)
	}
	// / from the line gives the search the keyboard, and esc gives it back to the line
	m = drive(m, down, typed("/")[0])
	if !m.inPanel || !strings.Contains(keysOf(m), "esc·⏎ back") {
		t.Fatalf("/ from the line: in panel %v, %q", m.inPanel, keysOf(m))
	}
	if m = drive(m, esc); m.inPanel || !m.onPreset {
		t.Errorf("esc from the search: in panel %v, on the line %v", m.inPanel, m.onPreset)
	}
	// naming for Save As…, a number is typed, as a name may hold one; a click on a row leaves the line
	m.choosePreset(table.ItemSaveAs)
	if m = drive(m, typed("2")...); !m.onPreset || m.stage != 0 || m.stages[0].Naming != "2" {
		t.Errorf("a number on the line: on the line %v, stage %d, %q", m.onPreset, m.stage, m.stages[0].Naming)
	}
	if m = drive(m, testkit.Click(6, rowY(m, "#45"))); m.onPreset || !m.inRows {
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
	if got := presetText(m); !strings.HasPrefix(got, "assignees: @me") || !strings.HasSuffix(got, "@me  "+selector(Opening)) {
		t.Errorf("mine: %q", got)
	}
	m.setPanel(m.panel(0).Shown(grid.Others))
	if got := presetText(m); !strings.HasPrefix(got, "assignees: ¬@me") {
		t.Errorf("others: %q", got)
	}
	// too narrow, the chips are clipped before the selector is
	m = drive(m, tea.WindowSizeMsg{Width: 30, Height: 40})
	if got := presetText(m); !strings.HasSuffix(got, " "+Presets+style.PillRight) || ansi.StringWidth(got) > m.paneWidth() {
		t.Errorf("narrow: %q", got)
	}
}

func TestChoosingAViewSetsThePanelAndAChangeStrikesTheNameThrough(t *testing.T) {
	m := onDropdown(t, "")
	m = drive(m, press(tea.KeyEnter))
	if !m.presetsMenu.Open {
		t.Fatal("enter did not drop the presets down")
	}
	// the menu is titled Presets at its left, as a dialog is, not by the preset the table is set as; it stands on the
	// control line, as the columns dialog does, its right edge on the selector's ▼
	if box, top, left := m.presetsMenu.Box(&m); !strings.HasPrefix(ansi.Strip(box[0]), "╭─ "+table.PresetsTitle+" ─") ||
		strings.Contains(box[0], Opening) || top != m.top() || left+ansi.StringWidth(box[0]) != m.dropEnd() {
		t.Errorf("the menu's title, at %d, %d: %q", top, left, ansi.Strip(box[0]))
	}
	// it is in its tab's color, as the selector is, not the menus' own
	if box, _, _ := m.presetsMenu.Box(&m); box[0] != shell.TitledColored(table.PresetsTitle, ansi.StringWidth(box[0]),
		stageColors["backlog"]) || strings.Contains(strings.Join(box, ""), style.Menu("│")) {
		t.Errorf("the menu is not in backlog's color: %q", box)
	}
	// the menu is drawn, under the line, whole
	if drawn := raw(m); !strings.Contains(drawn, SaveAs) || !strings.Contains(drawn, "Compact") {
		t.Fatalf("the presets are not drawn:\n%s", drawn)
	}
	names := []string{}
	for _, i := range table.PresetItems(&m) {
		names = append(names, i.Name)
	}
	if !slices.Equal(names, []string{FieldsIt, "", SaveIt, SaveAs, RenameIt, DeleteIt, "", "Default", "Compact"}) ||
		table.PresetItems(&m)[7].Glyph != Checked || m.presetsMenu.At != 7 {
		t.Fatalf("the presets: %q, at %d", names, m.presetsMenu.At)
	}
	if !strings.Contains(keysOf(m), "esc close") {
		t.Errorf("the keys with the menu open: %q", keysOf(m))
	}
	m = drive(m, down, press(tea.KeyEnter))
	if m.presetsMenu.Open || m.stages[0].ViewName != "Compact" || !slices.Equal(shownNow(m), []string{"id", "title"}) ||
		presetText(m) != selector("Compact") {
		t.Fatalf("Compact: %q, %q", shownNow(m), presetText(m))
	}
	// a column shown again differs from every view: the line names the one it was set to, struck through
	m.setPanel(m.panel(0).Placed("age", grid.Column))
	color := stageColors["backlog"]
	if got := presetText(m); m.is(0) != "" || got != selector("Compact") ||
		!strings.Contains(controlLine(m), table.SelectorPill("Compact", table.SelectorFocus, true)) {
		t.Errorf("a change does not strike the view through: %q", got)
	}
	if m.onPreset = false; !strings.Contains(controlLine(m), table.SelectorPill("Compact", color, true)) {
		t.Errorf("off the line, the view is not struck through, on the tab's color: %q", controlLine(m))
	}
	m.onPreset = true
	if m.dropEnd() != m.dropStart()+ansi.StringWidth(selector("Compact")) || m.dropStart() < m.paneWidth()/2 {
		t.Errorf("▼ from %d to %d", m.dropStart(), m.dropEnd())
	}
	// a stage's own view is never saved over, renamed, or deleted: those lines are dim
	if table.PresetEnabled(&m, table.ItemSave) || table.PresetEnabled(&m, table.ItemRename) || table.PresetEnabled(&m, table.ItemDelete) {
		t.Error("Save, Rename…, or Delete acts on a stage's own view")
	}
	// the menu then opens on Save As…, no name typed for a stage's own view; saving with no name is refused, as is
	// saving over a stage's own view; the change stays
	if m = drive(m, press(tea.KeyEnter)); m.presetsMenu.At != table.ItemSaveAs {
		t.Errorf("the menu on a changed table opens at %d", m.presetsMenu.At)
	}
	m = drive(m, press(tea.KeyEnter))
	if !m.stages[0].SavingAs || m.presetsMenu.Open || !strings.HasSuffix(presetText(m), SavingAs+" ▏") ||
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
	if m = drive(m, esc); m.stages[0].SavingAs || !m.onPreset || presetText(m) != selector("Compact") {
		t.Errorf("esc did not stop Save As…, or left the line: %q", presetText(m))
	}
	// the table set as another view names that one
	m.setPanel(m.builtins(0)[0].Panel(fieldsOf("backlog"), m.panel(0)))
	if got := presetText(m); got != selector(Opening) {
		t.Errorf("the table as Default: %q", got)
	}
	// the menu's keys: up, a letter to the next item it starts, and esc
	m = drive(m, press(tea.KeyEnter))
	if m = drive(m, typed("c")[0]); m.presetsMenu.At != 8 {
		t.Errorf("c in the menu: %d", m.presetsMenu.At)
	}
	m = drive(m, up, esc)
	if m.presetsMenu.Open || !m.onPreset {
		t.Errorf("esc in the menu: open %v", m.presetsMenu.Open)
	}
	// Default shows every column the catalog places, again
	m = drive(m, press(tea.KeyEnter), down, press(tea.KeyEnter), press(tea.KeyEnter), up)
	columns := slices.DeleteFunc(slices.Clone(fieldsOf("backlog")), func(f table.Field) bool { return f.Placement != table.PlaceColumn })
	// less source, which the view groups the rows by as it loads
	if m = drive(m, press(tea.KeyEnter)); len(shownNow(m)) != len(columns)-1 || m.is(0) != "Default" ||
		!slices.Equal(m.panel(0).Groups(), []string{"source"}) {
		t.Errorf("Default: %q", shownNow(m))
	}
}

func TestSaveAsKeepsThePanelInTheFileAndTheViewComesBack(t *testing.T) {
	file := filepath.Join(t.TempDir(), "kit", "presets.json")
	m := onDropdown(t, file)
	p := m.panel(0).Placed("branch", grid.Hidden).Sorted("age").Sorted("age").Shown(grid.Others)
	p = p.Grouped("assignees", false)
	m.setPanel(p)
	// off Save As…, what is typed names nothing
	if m = drive(m, typed("x")...); m.stages[0].SavingAs || m.stages[0].Naming != "" {
		t.Fatalf("typing off Save As…: %q", m.stages[0].Naming)
	}
	// Save As… from the menu, then a name, backspace, and enter
	m = drive(m, press(tea.KeyEnter), press(tea.KeyEnter))
	m = drive(m, typed("Triagex")...)
	if m = drive(m, press(tea.KeyBackspace)); m.stages[0].Naming != "Triage" ||
		!strings.HasSuffix(presetText(m), SavingAs+" Triage▏") {
		t.Fatalf("typing: %q", presetText(m))
	}
	m = drive(m, press(tea.KeyEnter))
	if last := keysOf(m); !strings.Contains(last, "saved the preset Triage") || m.stages[0].ViewName != "Triage" ||
		m.stages[0].SavingAs || !strings.HasSuffix(presetText(m), "assignees: ¬@me  "+selector("Triage")) {
		t.Fatalf("save: %q, view %q, %q", last, m.stages[0].ViewName, presetText(m))
	}
	data, err := os.ReadFile(file)
	var kept struct {
		Version int                       `json:"version"`
		Stages  map[string][]table.Preset `json:"stages"`
	}
	if err != nil || json.Unmarshal(data, &kept) != nil || kept.Version != 2 || len(kept.Stages["backlog"]) != 1 {
		t.Fatalf("the file: %s, %v", data, err)
	}
	// by field id: branch hidden, assignees grouped and filtered to everyone but the caller, age sorted descending
	want := map[string]table.Setting{"branch": {Placement: table.PlaceHidden}, "age": {Sort: "desc", Priority: 1},
		"assignees": {Placement: table.PlaceGroup, Depth: 1, Filter: &table.ValueFilter{Exclude: []string{caller}}}}
	// whether shared values fold is kittui's setting, so the preset does not keep it
	if v := kept.Stages["backlog"][0]; v.Name != "Triage" || !reflect.DeepEqual(v.Fields, want) ||
		strings.Contains(string(data), "fold") {
		t.Errorf("the view kept: %s", data)
	}
	// the menu lists it after a rule
	m = drive(m, press(tea.KeyEnter))
	if items := table.PresetItems(&m); len(items) != 11 || items[9].Name != "" || items[10].Name != "Triage" ||
		items[10].Glyph != Checked || m.presetsMenu.At != 10 {
		t.Errorf("the menu: %+v, at %d", items, m.presetsMenu.At)
	}
	m = drive(m, esc)
	// changed, then Save As… again, its name typed already, it is overwritten; up while naming stops, and leaves the
	// line
	m.setPanel(m.panel(0).Placed("branch", grid.Column))
	m.choosePreset(table.ItemSaveAs)
	if !m.stages[0].SavingAs || m.stages[0].Naming != "Triage" || !strings.HasSuffix(presetText(m), SavingAs+" Triage▏") {
		t.Fatalf("Save As… on Triage: %q", presetText(m))
	}
	m = drive(m, press(tea.KeyEnter))
	if data, _ = os.ReadFile(file); strings.Count(string(data), `"name"`) != 1 || strings.Contains(string(data), "branch") {
		t.Errorf("overwritten: %s", data)
	}
	m.choosePreset(table.ItemSaveAs)
	if m = drive(m, up); m.stages[0].SavingAs || m.onPreset {
		t.Errorf("up while naming: naming %v, on the line %v", m.stages[0].SavingAs, m.onPreset)
	}
	// another session reads it, and choosing it sets the panel
	n := onDropdown(t, file)
	n = drive(n, press(tea.KeyEnter))
	n.presetsMenu.At = 10
	if n = drive(n, press(tea.KeyEnter)); n.stages[0].ViewName != "Triage" || n.panel(0).Showing() != grid.Others ||
		len(n.panel(0).SortKeys()) != 1 || !slices.Equal(n.panel(0).Groups(), []string{"assignees"}) ||
		!n.panel(0).Fold {
		t.Errorf("Triage in another session: %+v", n.panel(0))
	}
}

func TestClicksOpenThePresetsAndChooseOne(t *testing.T) {
	file := filepath.Join(t.TempDir(), "presets.json")
	m := loaded(t, viewing(file), tea.WindowSizeMsg{Width: 140, Height: 40})
	// a click on the line before the name gives it the keyboard, and opens nothing
	if m = drive(m, testkit.Click(m.dropStart()-4, m.presetTop())); m.presetsMenu.Open || !m.onPreset {
		t.Fatalf("a click past the name: menu %v, on the line %v", m.presetsMenu.Open, m.onPreset)
	}
	m = drive(m, esc)
	// a click on the name drops the presets down, as one on ▼ does
	if m = drive(m, testkit.Click(m.dropStart()+1, m.presetTop())); !m.presetsMenu.Open || !m.onPreset {
		t.Fatalf("a click on the name: menu %v, on the line %v\n%s", m.presetsMenu.Open, m.onPreset, raw(m))
	}
	box, top, left := m.presetsMenu.Box(&m)
	if m = drive(m, testkit.Click(left+3, top+9)); m.presetsMenu.Open || m.stages[0].ViewName != "Compact" {
		t.Fatalf("a click on Compact: menu %v, view %q\n%v", m.presetsMenu.Open, m.stages[0].ViewName, box)
	}
	// a click anywhere else closes the menu
	m = drive(m, testkit.Click(m.dropEnd()-1, m.presetTop()))
	if !m.presetsMenu.Open {
		t.Fatalf("a click on ▼ did not open it:\n%s", raw(m))
	}
	if m = drive(m, testkit.Click(1, 20)); m.presetsMenu.Open || m.stages[0].ViewName != "Compact" {
		t.Errorf("a click off the menu: menu %v", m.presetsMenu.Open)
	}
	// a click on Save, dim on a stage's own view, does nothing, the menu left open, as on every dropdown; a click on
	// Save As… takes a name, and enter saves under it
	m = drive(m, testkit.Click(m.dropStart(), m.presetTop()))
	_, top, left = m.presetsMenu.Box(&m)
	if m = drive(m, testkit.Click(left+3, top+3)); m.stages[0].SavingAs || !m.presetsMenu.Open {
		t.Errorf("a click on Save, dim: naming %v, menu %v", m.stages[0].SavingAs, m.presetsMenu.Open)
	}
	m = drive(m, testkit.Click(left+3, top+4))
	m = drive(m, typed("Mine")...)
	if m = drive(m, press(tea.KeyEnter)); m.stages[0].ViewName != "Mine" {
		t.Errorf("Save As… by clicks: view %q, %q", m.stages[0].ViewName, keysOf(m))
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
		if !strings.Contains(m.note.Text, says) || m.note.Done || len(m.views(0)) != 2 {
			t.Errorf("a bad file: %q", m.note.Text)
		}
		if data, _ := os.ReadFile(file); string(data) != content {
			t.Errorf("the bad file was changed: %s", data)
		}
	}
	m := loaded(t, viewing(file), tea.WindowSizeMsg{Width: 140, Height: 40})
	// an empty name saves nothing; a file that cannot be written is said
	m = drive(m, down)
	m.stages[0].SavingAs, m.stages[0].Naming = true, "  "
	if m = drive(m, press(tea.KeyEnter)); !strings.Contains(lines(m)[len(lines(m))-1], "needs a name") {
		t.Errorf("an empty name: %q", lines(m)[len(lines(m))-1])
	}
	m.presetsPath = filepath.Join(file, "under-a-file", "presets.json")
	m.stages[0].SavingAs, m.stages[0].Naming = true, "Mine"
	if m = drive(m, press(tea.KeyEnter)); !strings.Contains(lines(m)[len(lines(m))-1], "could not be saved") {
		t.Errorf("an unwritable file: %q", lines(m)[len(lines(m))-1])
	}
	if got := table.PresetsPath(); !strings.HasSuffix(got, filepath.Join("kit", "presets.json")) {
		t.Errorf("the presets file: %q", got)
	}
	t.Setenv("XDG_CONFIG_HOME", "/somewhere")
	if got := table.PresetsPath(); got != "/somewhere/kit/presets.json" {
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
	m.choosePreset(table.ItemSaveAs)
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
	if !m.note.Done || m.note.Text != Save+" views.json is now presets.json, without the columns kittui no longer knows: "+
		"milestone in backlog Mine; estimate in backlog All; milestone in changes Authors" {
		t.Errorf("the notice: %q", m.note.Text)
	}
	if data, _ := os.ReadFile(filepath.Join(dir, "views.json")); string(data) != string(v1) {
		t.Errorf("views.json was changed: %s", data)
	}
	data, err := os.ReadFile(file)
	var kept struct {
		Version int                       `json:"version"`
		Stages  map[string][]table.Preset `json:"stages"`
	}
	if err != nil || json.Unmarshal(data, &kept) != nil || kept.Version != 2 {
		t.Fatalf("presets.json: %s, %v", data, err)
	}
	want := map[string][]table.Preset{
		"backlog": {
			{Name: "Mine", Fields: map[string]table.Setting{
				"title":     {Placement: table.PlaceColumn, Position: 1},
				"id":        {Placement: table.PlaceColumn, Position: 2},
				"source":    {Placement: table.PlaceColumn, Position: 3},
				"assignees": {Placement: table.PlaceColumn, Position: 4, Filter: &table.ValueFilter{Include: []string{caller}}},
				"age":       {Placement: table.PlaceColumn, Position: 5, Sort: "desc", Priority: 1},
				"branch":    {Placement: table.PlaceHidden},
				// a field the version 1 file could not name, as labels, stands after the columns it named
				"labels": {Placement: table.PlaceColumn, Position: 6},
			}},
			{Name: "Others", Fields: map[string]table.Setting{
				"assignees": {Placement: table.PlaceGroup, Depth: 1, Filter: &table.ValueFilter{Exclude: []string{caller}}},
			}},
			{Name: "All"},
		},
		"changes": {
			{Name: "Authors", Fields: map[string]table.Setting{"author": {Placement: table.PlaceGroup, Depth: 1}}},
		},
	}
	if !reflect.DeepEqual(kept.Stages, want) || !reflect.DeepEqual(m.saved, want) {
		t.Errorf("presets.json:\n%s", data)
	}
	// each show is the owner's filter: mine, others, and all
	for k, show := range []grid.Show{grid.Mine, grid.Others, grid.All} {
		if got := m.saved["backlog"][k].Panel(fieldsOf("backlog"), grid.Panel{Owner: "assignees"}).Showing(); got != show {
			t.Errorf("%s shows %v, not %v", m.saved["backlog"][k].Name, got, show)
		}
	}
	// read again, presets.json is what is read, and nothing is noted
	if n := loaded(t, viewing(file)); n.note.Text != "" || !reflect.DeepEqual(n.saved, want) {
		t.Errorf("read again: %q, %+v", n.note.Text, n.saved)
	}
	// a stage's fields a preset does not name take the catalog's placement
	if g := want["changes"][0].Panel(fieldsOf("changes"), grid.Panel{}); g.Field("branch").Placement != grid.Column ||
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
	m.stages[0].ViewName = "Gone"
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
		m.choosePreset(table.ItemSaveAs)
		m.stages[0].Naming = n
		return drive(m, press(tea.KeyEnter))
	}
	m.setPanel(m.panel(0).Placed("branch", grid.Hidden))
	m = name("Other")
	m.setPanel(m.panel(0).Placed("age", grid.Hidden))
	m = name("Triage")
	if table.PresetEnabled(&m, table.ItemSave) || !table.PresetEnabled(&m, table.ItemRename) || !table.PresetEnabled(&m, table.ItemDelete) {
		t.Fatalf("on Triage as it is: save %v, rename %v, delete %v", table.PresetEnabled(&m, table.ItemSave),
			table.PresetEnabled(&m, table.ItemRename), table.PresetEnabled(&m, table.ItemDelete))
	}
	// changed, Save overwrites it
	m.setPanel(m.panel(0).Placed("age", grid.Column))
	if !table.PresetEnabled(&m, table.ItemSave) {
		t.Fatal("Save is dim on a changed Triage")
	}
	m.choosePreset(table.ItemSave)
	if v, _ := m.own(); v.Name != "Triage" || !m.same(0, v, m.current(0)) || !strings.Contains(keysOf(m), Save+" saved the preset Triage") {
		t.Errorf("Save: %q", keysOf(m))
	}
	// Rename… types its name already; a stage's own name and another of the person's are refused
	m.choosePreset(table.ItemRename)
	if !strings.HasSuffix(presetText(m), Renaming+" Triage▏") || !strings.Contains(m.keys().String(), "⏎ rename") {
		t.Fatalf("Rename…: %q, %q", presetText(m), m.keys().String())
	}
	for _, refused := range []string{"Compact", "Other"} {
		m.stages[0].Naming = refused
		if m = drive(m, press(tea.KeyEnter)); !strings.Contains(m.note.Text, refused+" is ") || m.stages[0].ViewName != "Triage" {
			t.Errorf("rename to %s: %q", refused, m.note.Text)
		}
	}
	m.stages[0].Naming = "Mine"
	m = drive(m, press(tea.KeyEnter))
	if data, _ := os.ReadFile(file); m.stages[0].ViewName != "Mine" || strings.Contains(string(data), "Triage") ||
		!strings.Contains(string(data), `"Mine"`) || !strings.Contains(m.note.Text, Renamed+" renamed the preset Triage to Mine") {
		t.Errorf("renamed: view %q, %q, %s", m.stages[0].ViewName, m.note.Text, data)
	}
	// Delete drops it, and the stage is Default
	m.choosePreset(table.ItemDelete)
	if data, _ := os.ReadFile(file); strings.Contains(string(data), "Mine") || m.is(0) != Opening ||
		!strings.Contains(m.note.Text, Deleted+" deleted the preset Mine") {
		t.Errorf("deleted: %q, is %q, %s", m.note.Text, m.is(0), data)
	}
	if table.PresetEnabled(&m, table.ItemRename) || table.PresetEnabled(&m, table.ItemDelete) {
		t.Error("Rename… or Delete acts on Default")
	}
}

// A notice that says what was done goes once its time is up, short of a key; one that says why not stays, and a
// time up for an earlier notice leaves a later one.
func TestANoticeOfWhatWasDoneFades(t *testing.T) {
	m := onDropdown(t, filepath.Join(t.TempDir(), "presets.json"))
	m.choosePreset(table.ItemSaveAs)
	m.stages[0].Naming = "Mine"
	m = drive(m, press(tea.KeyEnter))
	first := m.note.FadeMsg()
	if !strings.Contains(keysOf(m), "saved the preset Mine") {
		t.Fatalf("the notice: %q", keysOf(m))
	}
	m.done(Save + " another")
	if m = drive(m, first); m.note.Text == "" {
		t.Error("a time up for an earlier notice took a later one")
	}
	if m = drive(m, m.note.FadeMsg()); m.note.Text != "" {
		t.Errorf("the notice stayed: %q", m.note.Text)
	}
	m.choosePreset(table.ItemSaveAs)
	m.stages[0].Naming = ""
	m = drive(m, press(tea.KeyEnter))
	if m = drive(m, m.note.FadeMsg()); !strings.Contains(m.note.Text, "needs a name") {
		t.Errorf("a refusal faded: %q", m.note.Text)
	}
}

// Rename… with no name is refused, and with the same name stops, the view as it was; Rename… and Delete that cannot
// write the file say so and change nothing, as does a file that cannot be written beside or replaced; Delete on a
// stage's own view does nothing.
func TestRenameAndDeleteRefuseWhatTheyCannotDo(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "presets.json")
	m := onDropdown(t, file)
	if m.choosePreset(table.ItemDelete); m.is(0) != Opening || m.note.Text != "" {
		t.Errorf("Delete on Default: %q", m.note.Text)
	}
	m.choosePreset(table.ItemSaveAs)
	m.stages[0].Naming = "Mine"
	m = drive(m, press(tea.KeyEnter))
	rename := func(name string) Model {
		m.choosePreset(table.ItemRename)
		m.stages[0].Naming = name
		return drive(m, press(tea.KeyEnter))
	}
	if m = rename(" "); !strings.Contains(m.note.Text, "needs a name") || !m.stages[0].SavingAs {
		t.Errorf("rename to nothing: %q", m.note.Text)
	}
	m = drive(m, esc)
	if m = rename("Mine"); m.stages[0].SavingAs || m.stages[0].ViewName != "Mine" || strings.Contains(m.note.Text, "renamed") {
		t.Errorf("rename to the same name: naming %v, %q", m.stages[0].SavingAs, m.note.Text)
	}
	m.presetsPath = filepath.Join(file, "under-a-file", "presets.json")
	if m = rename("Yours"); !strings.Contains(m.note.Text, "could not be renamed") || m.stages[0].ViewName != "Mine" {
		t.Errorf("rename, unwritable: %q", m.note.Text)
	}
	m = drive(m, esc)
	if m.choosePreset(table.ItemDelete); !strings.Contains(m.note.Text, "could not be deleted") || m.stages[0].ViewName != "Mine" {
		t.Errorf("delete, unwritable: %q", m.note.Text)
	}
	// the file cannot be written beside, as a directory stands where it would be; or replaced, as it is a directory
	blocked := filepath.Join(dir, "blocked.json")
	if err := os.MkdirAll(filepath.Join(blocked+".new", "x"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := table.WritePresets(blocked, nil); err == nil {
		t.Error("a file written where a directory stands")
	}
	if err := os.MkdirAll(filepath.Join(dir, "taken.json", "x"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := table.WritePresets(filepath.Join(dir, "taken.json"), nil); err == nil {
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
	if line, _ := m.control(); !m.presets(0) || !strings.Contains(line, table.SelectorPill(Opening, stageColors["backlog"], false)) {
		t.Errorf("backlog's preset line: %q", line)
	}
}

// controlLine gives the control line as the screen draws it, with its escapes.
func controlLine(m Model) string {
	line, _ := m.control()
	return line
}
