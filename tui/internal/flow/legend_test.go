// Every glyph the picture draws is in kit help flow's legend, the header of kit/bin/kit-flow, so bash and Go
// draw the same picture; and ? hangs each stage's glyphs under its tree, the glyphs in one column.
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
)

// legendText reads the Legend of kit-flow's header, from its heading to the exit codes.
func legendText(t *testing.T) string {
	t.Helper()
	page, err := os.ReadFile(filepath.Join("..", "..", "..", "kit", "bin", "kit-flow"))
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
		m = drive(m, down, enter)
	}
	return m
}

func TestEveryGlyphDrawnIsInTheHeaderLegendAndInItsStagesTree(t *testing.T) {
	text := legendText(t)
	for i, name := range eight {
		m := opened(t, i)
		for g := range glyphsOf(view(m)) {
			if !strings.Contains(text, g) {
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
			// the desk and the worktrees draw a pull request's state as changes does, and changes' tree names it
			if _, state := stateGlyphs[stateOf(g)]; !own[g] && !(own[Change] && state) {
				t.Errorf("%s draws %s, which its tree's legend lacks", name, g)
			}
		}
	}
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

func TestEachTreeHangsItsLegendWithTheGlyphsInOneColumn(t *testing.T) {
	for _, width := range []int{80, 120, 200} {
		m := loaded(t, options(), tea.WindowSizeMsg{Width: width, Height: 60}, press('?'))
		for i, name := range eight {
			if i > 0 {
				m = drive(m, right)
			}
			marker := m.center(i) - m.scroll + 1
			want := map[string]int{"legend": marker + 2}
			for _, e := range legends[name] {
				want[e.glyph+"  "+e.meaning] = marker + 4
			}
			if name == "changes" {
				want["state"] = marker + 4
				for _, e := range states {
					want[e.glyph+"  "+e.meaning] = marker + 6
				}
			}
			for text, column := range want {
				found := false
				for _, l := range lines(m) {
					found = found || slices.Contains(cellsOf(l, text), column)
				}
				if !found {
					t.Errorf("at %d, %s's tree lacks %q at cell %d:\n%s", width, name, text, column, view(m))
				}
			}
		}
	}
}
