// The help's crumb: its palette draws it, and a form opened from it does not.
package app

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/empowerite/kit/projects/tui/internal/shell"
	"github.com/empowerite/kit/projects/tui/internal/testkit"
)

func TestTheHelpDrawsItsCrumbOnItsPaletteAlone(t *testing.T) {
	m, err := OpenHelp(testkit.Runner(t, "kit"), "")
	if err != nil {
		t.Fatal(err)
	}
	m = drive(m)
	if m.Backs(testkit.Press("backspace")) {
		t.Error("⌫ goes back with no crumb")
	}
	m = m.Crumbed("tutorials").(Model)
	at := testkit.Column(m.View().Content, []rune(shell.Back)[0])
	if at < 0 || !m.OnCrumb(testkit.Click(at, shell.Top)) || !m.Backs(testkit.Press("backspace")) {
		t.Fatalf("the crumb at %d is not the way back:\n%s", at, m.View().Content)
	}
	if name, spot := m.Spot(); name != "help" || spot != nil || m.Restore(nil).(Model).InForm() {
		t.Errorf("the spot %q, %v", name, spot)
	}
	m = drive(m, press(tea.KeyDown), press(tea.KeyEnter))
	if !m.InForm() || strings.Contains(m.View().Content, shell.Back) || m.OnCrumb(testkit.Click(at, shell.Top)) ||
		m.Backs(testkit.Press("backspace")) {
		t.Errorf("a form draws or takes the crumb:\n%s", m.View().Content)
	}
	var bare Model
	if bare.Crumbed("x").(Model).hasPalette || bare.OnCrumb(testkit.Click(0, 0)) {
		t.Error("a help with no palette has a crumb")
	}
}
