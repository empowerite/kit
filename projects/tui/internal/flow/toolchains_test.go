// The providers table of the platform app's toolchains tab: a yes in a column is the mark kit toolchains prints, not
// the column's own name, and each heading says in plain words what it marks.
package flow

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/empowerite/kit/projects/tui/internal/shell"
)

func TestTheProvidersTableMarksEachYesAndNamesItsColumnsInPlainWords(t *testing.T) {
	m := appOf(t, shell.Platform, tea.WindowSizeMsg{Width: 120, Height: 30})
	m = drive(m, tab)
	var text []string
	for _, l := range lines(m) {
		text = append(text, ansi.Strip(l))
	}
	screen := strings.Join(text, "\n")
	for _, want := range []string{"usable here", "inside now", "kit uses", "adds the rest", "✅", "⚡", "📌", "➕"} {
		if !strings.Contains(screen, want) {
			t.Errorf("the providers table lacks %q:\n%s", want, screen)
		}
	}
	for _, line := range text {
		if strings.Contains(line, "mise") {
			for _, bare := range []string{"usable ", "active", "chosen", "fills"} {
				if strings.Contains(line, bare) {
					t.Errorf("a row draws the column's own name %q: %q", bare, line)
				}
			}
		}
	}
}
