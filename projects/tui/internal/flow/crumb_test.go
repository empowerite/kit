package flow

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"github.com/empowerite/kit/projects/tui/internal/shell"
	"github.com/empowerite/kit/projects/tui/internal/tabs"
	"github.com/empowerite/kit/projects/tui/internal/testkit"
)

// Where the picture stands, as a crumb keeps it, is put back as it was: the tab, maximized, what has the keyboard, the
// table's search and its cursor's row, though the picture moved on since; the name is the app's and the tab's.
func TestASpotPutsThePictureBackWhereItStood(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	o := options()
	o.App = shell.Changes
	m := loaded(t, o)
	i := len(m.stages) - 1
	m.choose(tabs.Bar{Selected: i})
	m = intoRows(m)
	if len(m.rows(i)) < 2 {
		t.Fatalf("%s has %d rows", m.stages[i].name, len(m.rows(i)))
	}
	m.down(1)
	m.maximized = true
	under := m.rows(i)[m.stages[i].Cursor].Key
	// the view its rows were grouped by as they loaded is kept too, as a copy of its own
	grouped := m.current(i)
	m.stages[i].Grouped = &grouped
	name, spot := m.Spot()
	if want := "changes › " + label(m.stages[i].name); name != want {
		t.Errorf("the name is %q, not %q", name, want)
	}
	m.choose(tabs.Bar{Selected: 0})
	m.maximized, m.stages[i].Cursor = false, 0
	p := m.panel(i)
	p.Query = "nonesuch"
	m.stages[i].View = p
	m.rebuild()
	m = m.Restore(spot).(Model)
	if m.stage != i || !m.maximized || !m.inRows || m.panel(i).Query != "" ||
		m.rows(i)[m.stages[i].Cursor].Key != under {
		t.Errorf("put back on %d, maximized %v, in rows %v, searching %q, under %q", m.stage, m.maximized, m.inRows,
			m.panel(i).Query, m.rows(i)[m.stages[i].Cursor].Key)
	}
	if g := m.stages[i].Grouped; g == nil || g == &grouped {
		t.Errorf("the view grouped as it loaded is put back as %p, the stage's own %p", g, &grouped)
	}
	if m.Restore("nothing").(Model).stage != i {
		t.Error("what is no spot moved the picture")
	}
	// a picture of no app's is flow's
	if name, _ := loaded(t, Options{Stages: []string{"issues"}}).Spot(); name != "flow › issues" {
		t.Errorf("the name of no app's picture is %q", name)
	}
}

// The crumb the picture is told stands last on its title; a plain click on it, or ⌫, is the way back, but while a
// menu is open over the title, or the search has the keyboard.
func TestThePicturesCrumbIsClickedOrKeyedBack(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	m := loaded(t, options())
	if m.Backs(testkit.Press("backspace")) {
		t.Error("⌫ goes back with no crumb")
	}
	m = m.Crumbed("issues › milestones").(Model)
	at := testkit.Column(m.View().Content, []rune(shell.Back)[0])
	if at < 0 || !m.OnCrumb(testkit.Click(at, shell.Top)) || !m.Backs(testkit.Press("backspace")) {
		t.Fatalf("the crumb at %d is not the way back", at)
	}
	if m.OnCrumb(testkit.Click(at, shell.Top+1)) || m.Backs(testkit.Press("x")) {
		t.Error("a click off the crumb, or another key, goes back")
	}
	open := m
	open.appMenu = open.appMenu.Opened()
	if open.OnCrumb(testkit.Click(at, shell.Top)) || open.Backs(testkit.Press("backspace")) {
		t.Error("the way back goes under the open app menu")
	}
	searching := m
	searching.inPanel = true
	if searching.Backs(testkit.Press("backspace")) {
		t.Error("⌫ goes back while the search has the keyboard")
	}
}

// The legend says what the crumb's glyph does, while the title shows one.
func TestTheLegendSaysTheCrumb(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	m := loaded(t, options())
	m.toggleLegend()
	if strings.Contains(m.View().Content, "back: click or ⌫") {
		t.Error("the legend names a crumb the title does not show")
	}
	m = m.Crumbed("issues").(Model)
	if !strings.Contains(m.View().Content, "back: click or ⌫") {
		t.Errorf("the legend does not name the crumb:\n%s", ansi.Strip(m.View().Content))
	}
}
