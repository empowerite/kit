// Every glyph the picture draws is in kit help flow's legend, the header of kit/bin/kit-flow, so bash and Go
// draw the same picture; and ? shows each stage's glyphs beside its rows, the glyphs in one column.
package flow

import (
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

	tea "charm.land/bubbletea/v2"

	"github.com/empowerite/kit/projects/tui/internal/grid"
	"github.com/empowerite/kit/projects/tui/internal/tabs"
	"github.com/empowerite/kit/projects/tui/internal/testkit"
)

// legendText reads the Legend of kit-flow's header, from its heading to the exit codes.
func legendText(t *testing.T) string {
	t.Helper()
	page, err := os.ReadFile(filepath.Join("..", "..", "..", "..", "kit", "bin", "kit-flow"))
	if err != nil {
		t.Fatal(err)
	}
	_, after, ok := strings.Cut(string(page), "# Legend")
	if !ok {
		t.Fatal("kit-flow has no Legend")
	}
	before, _, _ := strings.Cut(after, "# Exits")
	return before
}

// glyphsOf gives the glyphs in a text: the pictographs and symbols, and the media symbols as 🛑, not the arrows, the
// line art, the ellipsis, or ⏎. A glyph's variation selector is left out, so 🏭 is 🏭.
func glyphsOf(text string) map[string]bool {
	out := map[string]bool{}
	for _, r := range text {
		if r >= 0x1F000 || r >= 0x2600 && r <= 0x27BF || r >= 0x2B00 && r <= 0x2BFF || r >= 0x23E9 && r <= 0x23FA {
			out[string(r)] = true
		}
	}
	return out
}

func TestEveryGlyphIsInTheLegend(t *testing.T) {
	text := legendText(t)
	all := []entry{title}
	for _, entries := range legends {
		all = append(all, entries...)
	}
	all = append(all, states...)
	for _, g := range stageGlyphs {
		all = append(all, entry{g, "a stage"})
	}
	for _, e := range all {
		if !strings.Contains(text, e.glyph) {
			t.Errorf("the legend has no %s (%s)", e.glyph, e.meaning)
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

// opened gives the screen at 200×80 on stage i, the legend on and every row open.
func opened(t *testing.T, i int) Model {
	t.Helper()
	m := loaded(t, options(), tea.WindowSizeMsg{Width: 200, Height: 80}, press('?'))
	for range i {
		m = drive(m, right)
	}
	for range m.rows(i) {
		m = drive(m, down, toggle)
	}
	return m
}

func TestEveryGlyphDrawnIsInTheHeaderLegendAndInItsStagesLegend(t *testing.T) {
	text := legendText(t)
	for i, name := range eight {
		m := opened(t, i)
		for g := range glyphsOf(raw(m)) {
			// the bar's numbers are the tabs', and the panel's glyphs its controls', not the picture's
			if !strings.Contains(text, g) && !isMarker(g) && !isControl(g) {
				t.Errorf("%s draws %s, which kit help flow's legend lacks", name, g)
			}
		}
		// the stage's own glyphs: its rows', and the glyph beside its name
		own := map[string]bool{}
		for _, e := range legends[name] {
			maps.Copy(own, glyphsOf(e.glyph))
		}
		if name == "changes" {
			for _, e := range states {
				maps.Copy(own, glyphsOf(e.glyph))
			}
		}
		var pane []string
		for _, l := range m.lines(i) {
			pane = append(pane, ansi.Strip(l.text))
		}
		for g := range glyphsOf(strings.Join(pane, "\n") + stageGlyphs[name]) {
			// the desk and the worktrees draw a pull request's state as changes does, and changes' legend names it
			if _, state := stateGlyphs[stateOf(g)]; !own[g] && !(own[Change] && state) {
				t.Errorf("%s draws %s, which its legend lacks", name, g)
			}
		}
	}
}

// isControl says whether a glyph is one the panel draws for its controls.
func isControl(g string) bool {
	return slices.Contains([]string{grid.Search, grid.Ticked, grid.Unticked, grid.Picked, grid.Unpicked}, g)
}

// isMarker says whether a glyph is one of the tab bar's numbers.
func isMarker(g string) bool {
	for i := range 10 {
		if g == tabs.Marker(i, false) || g == tabs.Marker(i, true) {
			return true
		}
	}
	return false
}

// stateOf gives the state word a glyph draws, or nothing.
func stateOf(glyph string) string {
	for state, g := range stateGlyphs {
		if g == glyph {
			return state
		}
	}
	return ""
}

func TestTheLegendStandsBesideTheRowsWithTheGlyphsInOneColumn(t *testing.T) {
	for _, width := range []int{120, 200} {
		m := loaded(t, options(), tea.WindowSizeMsg{Width: width, Height: 60}, press('?'))
		for i, name := range eight {
			if i > 0 {
				m = drive(m, right)
			}
			column := m.paneWidth() + 3 // a space, the rule, and a space
			want := map[string]int{"legend": column}
			for _, e := range legends[name] {
				want[e.glyph+"  "+e.meaning] = column
			}
			if name == "changes" {
				want["state"] = column
				for _, e := range states {
					want[e.glyph+"  "+e.meaning] = column
				}
			}
			for text, at := range want {
				found := false
				for _, l := range lines(m)[3:] {
					found = found || slices.Contains(cellsOf(l, text), at)
				}
				if !found {
					t.Errorf("at %d, %s's legend lacks %q at cell %d:\n%s", width, name, text, at, view(m))
				}
			}
		}
	}
}

// The legend takes no more than two fifths of the screen, the rows have the rest, and no line passes the edge; esc
// takes it away, and the rows have the width again.
func TestTheLegendIsNarrowAndEscTakesItAway(t *testing.T) {
	for _, width := range []int{40, 80, 120} {
		m := loaded(t, options(), tea.WindowSizeMsg{Width: width, Height: 30}, press('?'))
		if lw := m.legendWidth(); lw == 0 || lw > (width-3)*2/5 || m.paneWidth()+lw+3 != width {
			t.Errorf("at %d, the legend is %d cells beside rows of %d", width, lw, m.paneWidth())
		}
		for _, l := range lines(m) {
			if n := ansi.StringWidth(l); n > width {
				t.Errorf("at %d, a line of %d cells: %q", width, n, l)
			}
		}
		if m = drive(m, press(tea.KeyEscape)); m.legend || m.paneWidth() != width-2 || strings.Contains(view(m), hide+" "+grid.Search) {
			t.Errorf("at %d, esc left the legend: %v, rows of %d", width, m.legend, m.paneWidth())
		}
	}
}

// While the legend is hidden, its rule ends every line of the pane, at the screen's last column, with ◂ on the first;
// a click on the triangle shows the legend, the triangle then ▸ beside its title, and again hides it.
func TestTheTriangleOnTheRuleShowsAndHidesTheLegend(t *testing.T) {
	m := loaded(t, options(), tea.WindowSizeMsg{Width: 80, Height: 12})
	drawn := strings.Split(raw(m), "\n")
	for k := paneTop; k < paneTop+m.paneHeight(); k++ {
		end := "┊"
		if k == paneTop {
			end = show
		}
		if !strings.HasSuffix(drawn[k], end) || ansi.StringWidth(drawn[k]) != 80 {
			t.Errorf("line %d does not end on %s at the last column: %q", k, end, drawn[k])
		}
	}
	triangle := testkit.Click(m.paneWidth()+1, paneTop)
	if m = drive(m, triangle); !m.legend || !strings.Contains(raw(m), hide+" "+grid.Search) {
		t.Fatalf("a click on %s did not show the legend:\n%s", show, raw(m))
	}
	triangle.X = m.paneWidth() + 1
	if m = drive(m, triangle); m.legend || !strings.HasSuffix(strings.Split(raw(m), "\n")[paneTop], show) {
		t.Errorf("a click on %s did not hide the legend:\n%s", hide, raw(m))
	}
}
