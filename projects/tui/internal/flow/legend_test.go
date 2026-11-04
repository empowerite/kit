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
				want[e.glyph+" "+e.meaning] = column
			}
			if name == "changes" {
				want["state"] = column
				for _, e := range states {
					want[e.glyph+" "+e.meaning] = column
				}
			}
			for text, at := range want {
				found := false
				for _, l := range lines(m) {
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
		if lw := m.sidebarWidth(); lw == 0 || lw > (width-3)*2/5 || m.paneWidth()+lw+3 != width {
			t.Errorf("at %d, the legend is %d cells beside rows of %d", width, lw, m.paneWidth())
		}
		for _, l := range lines(m) {
			if n := ansi.StringWidth(l); n > width {
				t.Errorf("at %d, a line of %d cells: %q", width, n, l)
			}
		}
		if m = drive(m, press(tea.KeyEscape)); m.sidebar || m.paneWidth() != width-2 || strings.Contains(view(m), "legend") {
			t.Errorf("at %d, esc left the legend: %v, rows of %d", width, m.sidebar, m.paneWidth())
		}
	}
}

// While the sidebar is hidden, the rule ends every line from under the bar down to the keys bar, at the screen's last
// column, with ◂ on its middle line; a click on the triangle shows the sidebar, the triangle then ▸, and a click
// anywhere else on the rule again hides it.
func TestTheTriangleOnTheRuleShowsAndHidesTheLegend(t *testing.T) {
	m := loaded(t, options(), tea.WindowSizeMsg{Width: 80, Height: 12})
	drawn := strings.Split(raw(m), "\n")
	for k := paneTop; k < m.sideHeight(); k++ {
		end := rule
		switch k {
		case m.triangleAt():
			end = show
		case m.triangleAt() + 1:
			end = under
		}
		if !strings.HasSuffix(drawn[k], end) || ansi.StringWidth(drawn[k]) != 80 {
			t.Errorf("line %d does not end on %s at the last column: %q", k, end, drawn[k])
		}
	}
	if m.triangleAt() != paneTop+(m.sideHeight()-paneTop)/2 || m.triangleAt() <= paneTop+1 {
		t.Errorf("the triangle is on line %d, not the rule's middle", m.triangleAt())
	}
	triangle := testkit.Click(m.paneWidth()+1, m.triangleAt())
	if m = drive(m, triangle); !m.sidebar || !strings.Contains(raw(m), "columns") {
		t.Fatalf("a click on %s did not show the legend:\n%s", show, raw(m))
	}
	triangle.X, triangle.Y = m.paneWidth()+1, paneTop
	if m = drive(m, triangle); m.sidebar || !strings.HasSuffix(strings.Split(raw(m), "\n")[m.triangleAt()], show) {
		t.Errorf("a click on %s did not hide the legend:\n%s", hide, raw(m))
	}
}

// Each glyph a stage's rows lead somewhere from is marked in its legend, † for a row under another stage and 🔗 for
// a page, and each mark is on a glyph that leads there; a pull request's state, marked under changes' state heading,
// leads to its page wherever it is drawn.
func TestTheLegendMarksEveryGlyphAClickLeadsSomewhereFrom(t *testing.T) {
	for i, name := range eight {
		m := opened(t, i)
		if name == "desk" {
			// the desk, on main, has no pull request; on a branch with one, it leads to it
			m.find("desk").desk.Branch = "1-the-first-release-v0-1-0"
			m.rebuild()
		}
		if name == "underway" {
			// a row whose work is on the desk leads to the desk, as one in a worktree leads to the worktree
			s := m.find("underway")
			s.under[1].Here, s.under[1].Worktree = true, m.find("desk").desk.Path
			m.rebuild()
		}
		drawn := map[string]string{}
		for _, r := range m.rows(i) {
			for _, c := range r.Cells {
				if c.Link == "" {
					continue
				}
				glyph, _, _ := strings.Cut(ansi.Strip(c.Values[0]), " ")
				mark := ToPage
				if strings.HasPrefix(c.Link, jumps) {
					mark = ToRow
				}
				if _, state := stateGlyphs[stateOf(glyph)]; state || glyph == Other {
					if mark != ToPage {
						t.Errorf("%s: a state's glyph %s leads to %q", name, glyph, c.Link)
					}
					continue
				}
				drawn[glyph] = mark
				if marks[name][glyph] != mark {
					t.Errorf("%s: %s leads to %q, but its legend marks it %q", name, glyph, c.Link, marks[name][glyph])
				}
			}
		}
		for glyph, mark := range marks[name] {
			if drawn[glyph] != mark {
				t.Errorf("%s's legend marks %s %s, which no row leads from", name, glyph, mark)
			}
		}
	}
}
