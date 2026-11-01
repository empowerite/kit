// Every glyph the picture draws is in kit help status's legend, the header of kit/bin/kit-status, so bash and Go
// draw the same picture; and ? draws every glyph of the table with its meaning.
package status

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

	tea "charm.land/bubbletea/v2"
)

// legendText reads the Legend of kit-status's header, from its heading to the exit codes.
func legendText(t *testing.T) string {
	t.Helper()
	page, err := os.ReadFile(filepath.Join("..", "..", "..", "kit", "bin", "kit-status"))
	if err != nil {
		t.Fatal(err)
	}
	_, after, ok := strings.Cut(string(page), "# Legend")
	if !ok {
		t.Fatal("kit-status has no Legend")
	}
	before, _, _ := strings.Cut(after, "# Exits")
	return before
}

func TestEveryGlyphIsInTheLegend(t *testing.T) {
	text := legendText(t)
	for _, e := range legend {
		if !strings.Contains(text, e.glyph) {
			t.Errorf("the legend has no %s", e.glyph)
		}
	}
	for state, glyph := range stateGlyphs {
		found := false
		for _, line := range strings.Split(text, "\n") {
			if strings.Contains(line, glyph) && strings.Contains(line, state) {
				found = true
			}
		}
		if !found {
			t.Errorf("the legend does not draw %s as %s", state, glyph)
		}
	}
	if stateGlyph("nonesuch") != Other || stateGlyph("ready") != "🛬" {
		t.Errorf("stateGlyph: %s %s", stateGlyph("nonesuch"), stateGlyph("ready"))
	}
	if kindGlyph("draft") != Draft || kindGlyph("pre-release") != Pre || kindGlyph("release") != Release {
		t.Errorf("kindGlyph: %s %s %s", kindGlyph("draft"), kindGlyph("pre-release"), kindGlyph("release"))
	}
}

func TestTheDrawnLegendListsEveryGlyphOfTheTable(t *testing.T) {
	for _, width := range []int{80, 50, 200} {
		m := New(runner(t), options())
		m = drive(m, tea.WindowSizeMsg{Width: width, Height: 60}, press('?'))
		v := view(m)
		for _, e := range legend {
			if !strings.Contains(v, e.glyph+"  "+e.meaning) {
				t.Errorf("at %d the drawn legend lacks %s %s:\n%s", width, e.glyph, e.meaning, v)
			}
		}
		// at 200 each stage's own glyphs sit under its name on the line, and the states under "state"
		groups := []string{groupEvery, groupIncoming, groupWorktrees, groupChanges, groupStates, groupReleases}
		if width == 200 {
			groups = []string{groupEvery, "state"}
		}
		for _, group := range groups {
			if !strings.Contains(v, group) {
				t.Errorf("at %d the drawn legend lacks the group %s:\n%s", width, group, v)
			}
		}
	}
}

func TestEveryGlyphOfTheLegendStartsInOneColumn(t *testing.T) {
	m := New(runner(t), options())
	m = drive(m, tea.WindowSizeMsg{Width: 80, Height: 60}, press('?'))
	column := -1
	for _, line := range m.legendLines() {
		plain := ansi.Strip(line)
		for _, e := range legend {
			if at := strings.Index(plain, e.glyph+"  "+e.meaning); at >= 0 {
				cells := ansi.StringWidth(plain[:at])
				if column >= 0 && cells != column {
					t.Errorf("%s starts at cell %d, the others at %d", e.glyph, cells, column)
				}
				column = cells
			}
		}
	}
	if column < 0 {
		t.Fatal("no glyph found")
	}
}
