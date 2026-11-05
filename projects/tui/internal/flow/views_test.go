// The panel's dropdown of views: each stage's own three, the person's saved in their file, a change to the panel
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
	if len(got) < 5 || got[1] != "" || !strings.HasPrefix(got[2], grid.ViewGlyph+" Default ▾") || got[3] != "" ||
		got[4] != "columns" {
		t.Errorf("the panel's first lines: %q", got[:5])
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
	names := []string{}
	for _, i := range m.viewItems() {
		names = append(names, i.Name)
	}
	if !slices.Equal(names, []string{"Everything", "Default", "Minimal"}) || m.viewItems()[1].Glyph != Checked {
		t.Fatalf("the views: %q", names)
	}
	if last := lines(m)[len(lines(m))-1]; !strings.Contains(last, "esc close") {
		t.Errorf("the keys with the menu open: %q", last)
	}
	m = drive(m, down, press(tea.KeyEnter))
	if m.viewMenu || m.stages[0].viewName != "Minimal" || !slices.Equal(shownNow(m), []string{"id", "title"}) ||
		m.panel(0).Changed {
		t.Fatalf("Minimal: %q, changed %v", shownNow(m), m.panel(0).Changed)
	}
	// a column shown again differs from Minimal: 💾 at the dropdown's right
	m.setPanel(m.panel(0).Keep("age"))
	p := m.panel(0)
	p.Hidden = map[string]bool{"source": true, "assignees": true, "branch": true}
	m.setPanel(p)
	if !m.panel(0).Changed || !strings.HasSuffix(sideText(m)[2], grid.Save) {
		t.Errorf("a change does not show save: %q", sideText(m)[2])
	}
	// saving over a stage's own view is refused; the change stays
	m = drive(m, tea.KeyPressMsg{Code: 's', Mod: tea.ModCtrl})
	if last := lines(m)[len(lines(m))-1]; !strings.Contains(last, "Minimal is the stage's own and does not change") ||
		!m.panel(0).Changed {
		t.Errorf("save over Minimal: %q", last)
	}
	// the menu's keys: up, a letter to the next view it starts, and esc
	m = drive(m, press(tea.KeyEnter))
	if m = drive(m, up, up, typed("m")[0]); m.viewAt != 2 {
		t.Errorf("m in the menu: %d", m.viewAt)
	}
	m = drive(m, esc, press(tea.KeyEnter), up, up)
	// Everything shows every column again
	if m = drive(m, press(tea.KeyEnter)); len(shownNow(m)) != len(columns["backlog"]) || m.panel(0).Changed {
		t.Errorf("Everything: %q", shownNow(m))
	}
}

func TestATypedNameSavesThePanelToTheFileAndTheViewComesBack(t *testing.T) {
	file := filepath.Join(t.TempDir(), "kit", "views.json")
	m := onDropdown(t, file)
	p := m.panel(0)
	p.Hidden, p.Sort, p.Show = map[string]bool{"branch": true}, []grid.SortKey{{Column: "age", Desc: true}}, grid.Others
	p.Group, p.Kept = []string{"assignees"}, map[string]bool{"source": true}
	m.setPanel(p)
	m.stages[0].headers = false
	// typing renames from the view's name; backspace takes one back; esc stops the rename
	m = drive(m, typed("x")...)
	if !m.stages[0].renaming || m.stages[0].naming != "Defaultx" || !strings.Contains(sideText(m)[2], "Defaultx▏") {
		t.Fatalf("typing: %q", sideText(m)[2])
	}
	if m = drive(m, esc); m.stages[0].renaming || !m.inPanel {
		t.Fatalf("esc did not stop the rename, or left the panel")
	}
	m = drive(m, slices.Repeat([]tea.Msg{press(tea.KeyBackspace)}, 7)...)
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
		v.Headers || len(v.Sort) != 1 || !v.Sort[0].Desc {
		t.Errorf("the view kept: %+v", v)
	}
	// the menu lists it after a rule
	m = drive(m, press(tea.KeyEnter))
	if items := m.viewItems(); len(items) != 5 || items[3].Name != "" || items[4].Name != "Triage" ||
		items[4].Glyph != Checked || m.viewAt != 4 {
		t.Errorf("the menu: %+v, at %d", items, m.viewAt)
	}
	m = drive(m, esc)
	// changed and saved again under its own name, it is overwritten
	p = m.panel(0)
	p.Hidden = nil
	m.setPanel(p)
	m = drive(m, tea.KeyPressMsg{Code: 's', Mod: tea.ModCtrl})
	if data, _ = os.ReadFile(file); strings.Count(string(data), `"name"`) != 1 || strings.Contains(string(data), "branch") {
		t.Errorf("overwritten: %s", data)
	}
	// another session reads it, and choosing it sets the panel
	n := onDropdown(t, file)
	n = drive(n, press(tea.KeyEnter))
	n.viewAt = 4
	if n = drive(n, press(tea.KeyEnter)); n.stages[0].viewName != "Triage" || n.panel(0).Show != grid.Others ||
		n.stages[0].headers || len(n.panel(0).Sort) != 1 || !slices.Equal(n.panel(0).Group, []string{"assignees"}) ||
		!n.panel(0).Kept["source"] {
		t.Errorf("Triage in another session: %+v", n.panel(0))
	}
}

func TestClicksOpenTheViewsChooseOneAndSave(t *testing.T) {
	file := filepath.Join(t.TempDir(), "views.json")
	m := onDropdown(t, file)
	y := m.viewRow()
	if m = drive(m, testkit.Click(m.width-m.sideWidth()+4, y)); !m.viewMenu {
		t.Fatalf("a click on the dropdown did not open it:\n%s", view(m))
	}
	box, top, left := m.viewBox()
	if m = drive(m, testkit.Click(left+3, top+3)); m.viewMenu || m.stages[0].viewName != "Minimal" {
		t.Fatalf("a click on Minimal: menu %v, view %q\n%v", m.viewMenu, m.stages[0].viewName, box)
	}
	// a click anywhere else closes the menu
	m = drive(m, testkit.Click(m.width-m.sideWidth()+4, y))
	if m = drive(m, testkit.Click(1, 20)); m.viewMenu || m.stages[0].viewName != "Minimal" {
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
	if !strings.Contains(m.notice, "does not read as views") || len(m.views(0)) != 3 {
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
