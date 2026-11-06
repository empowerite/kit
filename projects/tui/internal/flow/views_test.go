// The panel's dropdown of views: each stage's own two, the person's saved in their file, a change to the panel
// showing 💾, and a save that keeps the panel under the name typed, never over a stage's own.
package flow

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/empowerite/kit/projects/tui/internal/grid"
	"github.com/empowerite/kit/projects/tui/internal/shell"
	"github.com/empowerite/kit/projects/tui/internal/testkit"
)

// onDropdown gives backlog with the sidebar shown and the keyboard on the panel's dropdown, its views kept in file.
func onDropdown(t *testing.T, file string) Model {
	t.Helper()
	m := loaded(t, viewing(file), tea.WindowSizeMsg{Width: 140, Height: 40}, press('?'), tab, down)
	if !m.inPanel || !m.panel(m.stage).OnView() {
		t.Fatalf("the keyboard is not on the dropdown: cursor %d\n%s", m.panel(m.stage).Cursor, view(m))
	}
	return m
}

// sideText gives the panel's lines without their escapes.
func sideText(m Model) []string {
	lines, _ := m.panel(m.stage).Lines()
	out := make([]string, len(lines))
	for k, l := range lines {
		out[k] = strings.TrimRight(ansi.Strip(l), " ")
	}
	return out
}

// shown gives the columns a stage's panel shows.
func shownNow(m Model) []string { return m.shownColumns(m.stage) }

func TestTheDropdownStandsUnderTheSearchAndNamesTheView(t *testing.T) {
	m := onDropdown(t, "")
	got := sideText(m)
	if len(got) < 6 || got[1] != "" || got[2] != "" || !strings.HasPrefix(got[3], "Default ▾") || got[4] != "" ||
		got[5] != "columns" {
		t.Errorf("the panel's first lines: %q", got[:6])
	}
	if last := lines(m)[len(lines(m))-1]; !strings.Contains(last, "⏎ views") || !strings.Contains(last, "⌃S save") {
		t.Errorf("the keys on the dropdown: %q", last)
	}
	// a stage with no columns has no dropdown
	m = drive(m, esc, press('7'))
	if m.stages[m.stage].name != "incoming" || m.panel(m.stage).Views {
		t.Errorf("%s offers views", m.stages[m.stage].name)
	}
}

func TestChoosingAViewSetsThePanelAndAChangeShowsSave(t *testing.T) {
	m := onDropdown(t, "")
	m = drive(m, press(tea.KeyEnter))
	if !m.viewMenu {
		t.Fatal("enter did not drop the views down")
	}
	// the menu is drawn, over the sidebar, whole
	if drawn := raw(m); !strings.Contains(drawn, "Default") || !strings.Contains(drawn, "Compact") {
		t.Fatalf("the views are not drawn:\n%s", drawn)
	}
	names := []string{}
	for _, i := range m.viewItems() {
		names = append(names, i.Name)
	}
	if !slices.Equal(names, []string{"Default", "Compact"}) || m.viewItems()[0].Glyph != Checked {
		t.Fatalf("the views: %q", names)
	}
	if last := lines(m)[len(lines(m))-1]; !strings.Contains(last, "esc close") {
		t.Errorf("the keys with the menu open: %q", last)
	}
	m = drive(m, down, press(tea.KeyEnter))
	if m.viewMenu || m.stages[0].viewName != "Compact" || !slices.Equal(shownNow(m), []string{"id", "title"}) ||
		m.panel(0).Changed {
		t.Fatalf("Compact: %q, changed %v", shownNow(m), m.panel(0).Changed)
	}
	// a column shown again differs from Compact: the dropdown names no view, and 💾 Save, dim, stands at its right
	m.setPanel(m.panel(0).Placed("age", grid.Column))
	if q := m.panel(0); !q.Changed || q.Savable || q.View != "" || !strings.HasPrefix(sideText(m)[3], grid.Unnamed+" ▾") ||
		!strings.HasSuffix(sideText(m)[3], grid.Saving) {
		t.Errorf("a change does not show save, or names a view: %q", sideText(m)[3])
	}
	if at := m.width - m.sideWidth() + len(grid.Unnamed) + 1; m.dropAt() != at {
		t.Errorf("▾ after unnamed at %d, not %d", m.dropAt(), at)
	}
	// saving with no name is refused, as is saving over a stage's own view; the change stays
	m = drive(m, tea.KeyPressMsg{Code: 's', Mod: tea.ModCtrl})
	if last := lines(m)[len(lines(m))-1]; !strings.Contains(last, "a view needs a name") || !m.panel(0).Changed {
		t.Errorf("save with no name: %q", last)
	}
	m = drive(m, typed("Compact")...)
	if m.panel(0).Savable {
		t.Error("a stage's own name can be saved")
	}
	m = drive(m, tea.KeyPressMsg{Code: 's', Mod: tea.ModCtrl})
	if last := lines(m)[len(lines(m))-1]; !strings.Contains(last, "Compact is the stage's own and does not change") ||
		!m.panel(0).Changed {
		t.Errorf("save over Compact: %q", last)
	}
	m = drive(m, esc)
	// the menu's keys: up, a letter to the next view it starts, and esc
	m = drive(m, press(tea.KeyEnter))
	if m = drive(m, up, typed("c")[0]); m.viewAt != 1 {
		t.Errorf("c in the menu: %d", m.viewAt)
	}
	m = drive(m, esc, press(tea.KeyEnter), up)
	// Default shows every column again
	if m = drive(m, press(tea.KeyEnter)); len(shownNow(m)) != len(fieldsOf("backlog")) || m.panel(0).Changed {
		t.Errorf("Default: %q", shownNow(m))
	}
}

func TestATypedNameSavesThePanelToTheFileAndTheViewComesBack(t *testing.T) {
	file := filepath.Join(t.TempDir(), "kit", "views.json")
	m := onDropdown(t, file)
	p := m.panel(0).Placed("branch", grid.Hidden).Sorted("age", false).Sorted("age", false).Shown(grid.Others)
	p = p.Grouped("assignees", false)
	p.Fold = false
	m.setPanel(p)
	m.stages[0].headers = false
	// a letter first typed takes the name's place; backspace first takes the name's last letter; esc stops the rename
	m = drive(m, typed("x")...)
	if !m.stages[0].renaming || m.stages[0].naming != "x" || !strings.Contains(sideText(m)[3], "x▏") ||
		!m.panel(0).Savable {
		t.Fatalf("typing: %q", sideText(m)[3])
	}
	if m = drive(m, esc); m.stages[0].renaming || !m.inPanel {
		t.Fatalf("esc did not stop the rename, or left the panel")
	}
	// the panel differs from Default, so the dropdown names none, and backspace first has nothing to take back
	if m = drive(m, press(tea.KeyBackspace)); m.stages[0].naming != "" || m.panel(0).Savable {
		t.Fatalf("backspace first: %q", m.stages[0].naming)
	}
	m = drive(m, esc)
	m = drive(m, typed("Triage")...)
	m = drive(m, tea.KeyPressMsg{Code: 's', Mod: tea.ModCtrl})
	if last := lines(m)[len(lines(m))-1]; !strings.Contains(last, "saved the view Triage") || m.stages[0].viewName != "Triage" ||
		m.panel(0).Changed {
		t.Fatalf("save: %q, view %q, changed %v", last, m.stages[0].viewName, m.panel(0).Changed)
	}
	data, err := os.ReadFile(file)
	var kept map[string][]named
	if err != nil || json.Unmarshal(data, &kept) != nil || len(kept["backlog"]) != 1 {
		t.Fatalf("the file: %s, %v", data, err)
	}
	if v := kept["backlog"][0]; v.Name != "Triage" || !slices.Equal(v.Hidden, []string{"branch"}) || v.Show != "others" ||
		v.Headers || len(v.Sort) != 1 || !v.Sort[0].Desc || v.Fold == nil || *v.Fold {
		t.Errorf("the view kept: %+v", v)
	}
	// the menu lists it after a rule
	m = drive(m, press(tea.KeyEnter))
	if items := m.viewItems(); len(items) != 4 || items[2].Name != "" || items[3].Name != "Triage" ||
		items[3].Glyph != Checked || m.viewAt != 3 {
		t.Errorf("the menu: %+v, at %d", items, m.viewAt)
	}
	m = drive(m, esc)
	// changed, the dropdown names no view, and nothing is saved until a name is typed; saved again under its own name,
	// it is overwritten
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
	n.viewAt = 3
	if n = drive(n, press(tea.KeyEnter)); n.stages[0].viewName != "Triage" || n.panel(0).Showing() != grid.Others ||
		n.stages[0].headers || len(n.panel(0).SortKeys()) != 1 || !slices.Equal(n.panel(0).Groups(), []string{"assignees"}) ||
		n.panel(0).Fold {
		t.Errorf("Triage in another session: %+v", n.panel(0))
	}
}

func TestClicksOpenTheViewsChooseOneAndSave(t *testing.T) {
	file := filepath.Join(t.TempDir(), "views.json")
	m := onDropdown(t, file)
	y := m.viewRow()
	// a click on the name puts the keyboard there, to type a new one, and opens nothing
	m = drive(m, esc)
	if m = drive(m, testkit.Click(m.width-m.sideWidth()+2, y)); m.viewMenu || !m.inPanel || !m.panel(0).OnView() {
		t.Fatalf("a click on the name: menu %v, on the dropdown %v", m.viewMenu, m.panel(0).OnView())
	}
	if m = drive(m, typed("Q")...); m.stages[0].naming != "Q" {
		t.Fatalf("typing after a click on the name: %q", m.stages[0].naming)
	}
	m = drive(m, esc)
	if m = drive(m, testkit.Click(m.dropAt(), y)); !m.viewMenu {
		t.Fatalf("a click on ▾ did not open it:\n%s", raw(m))
	}
	box, top, left := m.viewBox()
	if m = drive(m, testkit.Click(left+3, top+2)); m.viewMenu || m.stages[0].viewName != "Compact" {
		t.Fatalf("a click on Compact: menu %v, view %q\n%v", m.viewMenu, m.stages[0].viewName, box)
	}
	// a click anywhere else closes the menu
	m = drive(m, testkit.Click(m.dropAt(), y))
	if m = drive(m, testkit.Click(1, 20)); m.viewMenu || m.stages[0].viewName != "Compact" {
		t.Errorf("a click off the menu: menu %v", m.viewMenu)
	}
	// with a change and a name of its own, a click on 💾 saves
	m.stages[0].renaming, m.stages[0].naming = true, "Mine"
	if m = drive(m, testkit.Click(m.width-2, y)); m.stages[0].viewName != "Mine" {
		t.Errorf("a click on save: view %q, %q", m.stages[0].viewName, lines(m)[len(lines(m))-1])
	}
	if _, err := os.Stat(file); err != nil {
		t.Errorf("nothing saved: %v", err)
	}
}

func TestAViewsFileThatDoesNotReadIsSaidAndLeftAsItIs(t *testing.T) {
	file := filepath.Join(t.TempDir(), "views.json")
	if err := os.WriteFile(file, []byte("{not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	m := loaded(t, viewing(file), tea.WindowSizeMsg{Width: 140, Height: 40})
	if !strings.Contains(m.notice, "does not read as views") || len(m.views(0)) != 2 {
		t.Errorf("a bad file: %q", m.notice)
	}
	// an empty name saves nothing; a file that cannot be written is said
	m = drive(m, press('?'), tab, down)
	m.stages[0].renaming, m.stages[0].naming = true, "  "
	if m = drive(m, tea.KeyPressMsg{Code: 's', Mod: tea.ModCtrl}); !strings.Contains(lines(m)[len(lines(m))-1], "needs a name") {
		t.Errorf("an empty name: %q", lines(m)[len(lines(m))-1])
	}
	m.viewsPath = filepath.Join(file, "under-a-file", "views.json")
	m.stages[0].renaming, m.stages[0].naming = true, "Mine"
	if m = drive(m, tea.KeyPressMsg{Code: 's', Mod: tea.ModCtrl}); !strings.Contains(lines(m)[len(lines(m))-1], "could not be saved") {
		t.Errorf("an unwritable file: %q", lines(m)[len(lines(m))-1])
	}
	if got := viewsFile(); !strings.HasSuffix(got, filepath.Join("kit", "views.json")) {
		t.Errorf("the views file: %q", got)
	}
	t.Setenv("XDG_CONFIG_HOME", "/somewhere")
	if got := viewsFile(); got != "/somewhere/kit/views.json" {
		t.Errorf("under XDG_CONFIG_HOME: %q", got)
	}
}

// A view the person saved as Everything or Minimal, which the stages once had of their own, is the person's: it is
// listed after the stage's own, and saving over it overwrites it.
func TestAViewNamedAsAStageOnceNamedItsOwnIsThePersons(t *testing.T) {
	file := filepath.Join(t.TempDir(), "views.json")
	kept := `{"backlog": [{"name": "Everything", "headers": true}, {"name": "Minimal", "hidden": ["age"], "headers": true}]}`
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
