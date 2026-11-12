// The settings tab of the platform app: each value in its value column.
package flow

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/empowerite/kit/projects/tui/internal/shell"
	"github.com/empowerite/kit/projects/tui/internal/style"
)

// The settings tab's value column draws each value as JSON colored to be scanned, pretty-printed on an open row and
// compact on a closed one, and a value that is null or empty as a dim word in place of a blank.
func TestTheSettingsTabDrawsEachValueColoredAndAnEmptyOneAsAWord(t *testing.T) {
	m := appOf(t, shell.Platform, tea.WindowSizeMsg{Width: 120, Height: 40})
	m = drive(m, tab, tab, tab)
	drawn := func() string {
		var out []string
		for _, l := range m.lines() {
			out = append(out, l.Text)
		}
		return strings.Join(out, "\n")
	}
	closed := drawn()
	for _, want := range []string{
		style.Text("upstream"), style.Warn("14"), style.Off("none"), style.Off("empty"),
		style.Off("{") + style.Tint(`"commit_subject"`) + style.Off(":") + style.Text(`""`) + style.Off("}"),
		style.Off("[]"), style.Off("{}"),
	} {
		if !strings.Contains(closed, want) {
			t.Errorf("the settings tab lacks %q:\n%s", want, closed)
		}
	}
	for range m.rows(m.stage) {
		m = drive(m, down, toggle)
	}
	if open := ansi.Strip(drawn()); !strings.Contains(open, "\"commit_subject\": \"\"\n") {
		t.Errorf("an open row does not pretty-print its object:\n%s", open)
	}
}
