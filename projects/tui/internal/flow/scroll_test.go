// A stage whose columns are wider than the screen hides those past its edge behind ▹ on its header, and tab, shift-tab,
// and a click on a triangle scroll them a column at a time, the id put.
package flow

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/empowerite/kit/projects/tui/internal/grid"
	"github.com/empowerite/kit/projects/tui/internal/shell"
	"github.com/empowerite/kit/projects/tui/internal/testkit"
)

// narrow gives issues at 40 cells, too narrow for its columns, its rows with the keyboard.
func narrow(t *testing.T) Model {
	t.Helper()
	return intoRows(loaded(t, options(), tea.WindowSizeMsg{Width: 40, Height: 16}))
}

// headerOf gives the screen line the header's first line stands on, and its text, without its escapes.
func headerOf(t *testing.T, m Model) (int, string) {
	t.Helper()
	y := m.headerTop()
	if y < 0 {
		t.Fatalf("no header:\n%s", view(m))
	}
	return y, ansi.Strip(lines(m)[y])
}

func TestTabScrollsTheColumnsOfAStageWiderThanTheScreen(t *testing.T) {
	m := narrow(t)
	if _, head := headerOf(t, m); !strings.HasSuffix(strings.TrimRight(head, " "), grid.After) ||
		strings.Contains(head, grid.Before) || strings.Contains(view(m), "Fix the") {
		t.Fatalf("the columns past the edge hide behind ▹:\n%s", view(m))
	}
	if keys := (shell.Path{m.rowKeys()}).String(); !strings.Contains(keys, "⇥·⇧⇥ columns") {
		t.Errorf("the rows' keys of a stage cut: %s", keys)
	}
	m = drive(m, tab)
	if _, head := headerOf(t, m); !strings.Contains(head, "id  "+grid.Before) || !strings.Contains(view(m), "#7") ||
		!strings.Contains(view(m), "🎫") || !m.inRows || m.stage != 0 {
		t.Errorf("tab scrolls a column, the id put, the keyboard in the rows:\n%s", view(m))
	}
	if m = drive(m, backTab); m.stages[0].Shift != 0 || strings.Contains(view(m), grid.Before) {
		t.Errorf("shift+tab scrolls back:\n%s", view(m))
	}
}

func TestAClickOnAHeadersTriangleScrollsItsColumns(t *testing.T) {
	m := narrow(t)
	y, head := headerOf(t, m)
	x := ansi.StringWidth(head[:strings.Index(head, grid.After)])
	if m = drive(m, testkit.Click(x, y)); m.stages[0].Shift != 1 {
		t.Fatalf("a click on ▹: shift %d\n%s", m.stages[0].Shift, view(m))
	}
	y, head = headerOf(t, m)
	x = ansi.StringWidth(head[:strings.Index(head, grid.Before)])
	if m = drive(m, testkit.Click(x, y)); m.stages[0].Shift != 0 {
		t.Errorf("a click on ◃: shift %d\n%s", m.stages[0].Shift, view(m))
	}
}

func TestAStageThatFitsDrawsNoTriangleAndTabScrollsNothing(t *testing.T) {
	m := intoRows(loaded(t, options()))
	if keys := (shell.Path{m.rowKeys()}).String(); strings.Contains(keys, "columns") {
		t.Errorf("the rows' keys of a stage that fits: %s", keys)
	}
	m = drive(m, tab)
	if _, head := headerOf(t, m); m.stages[0].Shift != 0 || strings.ContainsAny(head, grid.Before+grid.After) {
		t.Errorf("tab in a stage that fits:\n%s", view(m))
	}
}

func TestTheLegendExplainsAHeadersTriangles(t *testing.T) {
	m := drive(narrow(t), press('?'))
	if got := strings.Join(m.legendLines(), "\n"); !strings.Contains(ansi.Strip(got), "more columns: click to scroll") {
		t.Errorf("the legend of a stage cut:\n%s", ansi.Strip(got))
	}
}
