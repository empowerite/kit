// The screen's crumb: the way back to where a link led from.
package welcome

import (
	"strings"
	"testing"

	"github.com/empowerite/kit/projects/tui/internal/shell"
	"github.com/empowerite/kit/projects/tui/internal/testkit"
)

// The crumb the screen is told stands last on its title; a plain click on it, or ⌫, is the way back, but while a menu
// is open over the title, or the search has the keyboard.
func TestTheCrumbIsClickedOrKeyedBack(t *testing.T) {
	m := func() Model { m, _ := loaded(t); return m }()
	if m.Backs(testkit.Press("backspace")) {
		t.Error("⌫ goes back with no crumb")
	}
	m = m.Crumbed("flow").(Model)
	keys := m.keys().String() // before the search, whose state the copies share
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
	s := searching.search()
	s.Focused = true
	searching.setSearch(s)
	if searching.Backs(testkit.Press("backspace")) {
		t.Error("⌫ goes back while the search has the keyboard")
	}
	if !strings.Contains(keys, "⌫ back") {
		t.Errorf("the keys do not name the way back: %q", keys)
	}
}

// The screen says where it stands as flow's picture does, by its step and what is searched for in it, a term's search
// without the slash it ends in; it keeps its place as it was left.
func TestWelcomeSaysWhereItStands(t *testing.T) {
	m, _ := loaded(t)
	name, spot := m.Spot()
	if !strings.HasPrefix(name, "welcome › ") || spot != nil || strings.Count(name, "›") != 1 {
		t.Errorf("the spot %q, %v", name, spot)
	}
	m.jump("terms/pull%20request/")
	if name, _ := m.Spot(); !strings.HasSuffix(name, "› terms › pull request") {
		t.Errorf("after a jump to a term: %q", name)
	}
	if got := m.Restore(nil); got.(Model).step != m.step {
		t.Error("restoring moved the screen")
	}
}
