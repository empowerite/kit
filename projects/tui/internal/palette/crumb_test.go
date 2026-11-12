// The screen's crumb: the way back to where a link led from.
package palette

import (
	"strings"
	"testing"

	"github.com/empowerite/kit/projects/tui/internal/shell"
	"github.com/empowerite/kit/projects/tui/internal/testkit"
)

// The crumb the screen is told stands last on its title; a plain click on it, or ⌫, is the way back, but while a menu
// is open over the title, or the search has the keyboard.
func TestTheCrumbIsClickedOrKeyedBack(t *testing.T) {
	m := listing(t)
	if m.Backs(testkit.Press("backspace")) {
		t.Error("⌫ goes back with no crumb")
	}
	m = m.Crumbed("flow").(Model)
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
	searching.search.Focused = true
	if searching.Backs(testkit.Press("backspace")) {
		t.Error("⌫ goes back while the search has the keyboard")
	}
	if !strings.Contains(m.keys().String(), "⌫ back") {
		t.Errorf("the keys do not name the way back")
	}
}
