// Every glyph the picture draws is in kit help flow's legend, the header of kit/bin/kit-flow, so bash and Go
// draw the same picture; and ? shows each stage's glyphs beside its rows, the glyphs in one column.
package flow

import (
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

	tea "charm.land/bubbletea/v2"

	"github.com/empowerite/kit/projects/tui/internal/grid"
	"github.com/empowerite/kit/projects/tui/internal/pane"
	"github.com/empowerite/kit/projects/tui/internal/shell"
	"github.com/empowerite/kit/projects/tui/internal/style"
	"github.com/empowerite/kit/projects/tui/internal/table"
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
			pane = append(pane, ansi.Strip(l.Text))
		}
		for g := range glyphsOf(strings.Join(pane, "\n") + stageGlyphs[name]) {
			// the desk and the worktrees draw a pull request's state as changes does, and changes' legend names it
			if _, state := stateGlyphs[stateOf(g)]; !own[g] && !(own[Change] && state) {
				t.Errorf("%s draws %s, which its legend lacks", name, g)
			}
		}
	}
}

// isControl says whether a glyph is one the panel draws for its controls, or the bar's menu glyph.
func isControl(g string) bool {
	return slices.Contains([]string{grid.Search, grid.Ticked, grid.Unticked, grid.Picked, grid.Unpicked, tabs.Menu,
		Save}, g)
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

// F1 or ? floats the legend over the rows, in the bottom right corner as it opens, titled by the stage: it says what
// each glyph the rows draw now means, beside it, and nothing of a glyph they do not draw. An arrow on its top edge moves it to
// that corner, the one it stands in dim; a click elsewhere on it does nothing, and one off it is the screen's.
func TestTheLegendFloatsInACornerWithTheGlyphsOnTheScreen(t *testing.T) {
	for _, width := range []int{80, 120, 200} {
		m := loaded(t, options(), tea.WindowSizeMsg{Width: width, Height: 40}, tea.KeyPressMsg{Code: tea.KeyF1})
		for i, name := range eight {
			if i > 0 {
				m = drive(m, right)
			}
			box, top, left := m.legendBox()
			drawn := m.drawn()
			if !strings.Contains(ansi.Strip(box[0]), name+" legend") || left+ansi.StringWidth(box[0]) != width ||
				top+len(box) != m.height-1 {
				t.Errorf("at %d, %s's legend is not in the bottom right, titled: %d, %d\n%s", width, name, top, left, view(m))
			}
			entries := legends[name]
			if name == "changes" {
				entries = append(slices.Clone(entries), states...)
			}
			for _, e := range entries {
				on, shown := strings.Contains(drawn, e.glyph), false
				for _, l := range box {
					shown = shown || explains(l, e)
				}
				if on != shown {
					t.Errorf("at %d, %s: %s %q drawn %v, in the legend %v:\n%s", width, name, e.glyph, e.meaning, on,
						shown, view(m))
				}
			}
		}
	}
	m := loaded(t, options(), press('?'))
	box, top, left := m.legendBox()
	arrow := func(c corner) int { return left + cellOf(box[0], arrows[c]) }
	for _, c := range []corner{topLeft, topRight, bottomLeft, bottomRight} {
		box, top, left = m.legendBox()
		if m = drive(m, testkit.Click(arrow(c), top)); m.legend.at != c || !m.legend.open {
			t.Errorf("the %s arrow: at %d", arrows[c], m.legend.at)
		}
		box, top, left = m.legendBox()
		if c == topLeft && (top != m.top() || left != 0) || c == bottomRight && left+ansi.StringWidth(box[0]) != m.width {
			t.Errorf("the legend in corner %d: %d, %d", c, top, left)
		}
	}
	// a click on the legend off its arrows does nothing; one off it, the screen takes; F1 hides it
	before := m.stages[m.stage].Cursor
	if m = drive(m, testkit.Click(left+2, top+1)); !m.legend.open || m.legend.at != bottomRight || m.stages[m.stage].Cursor != before {
		t.Errorf("a click on the legend: open %v, at %d", m.legend.open, m.legend.at)
	}
	if m = drive(m, tea.KeyPressMsg{Code: tea.KeyF1}); m.legend.open || strings.Contains(view(m), "backlog legend") {
		t.Errorf("F1 kept the legend:\n%s", view(m))
	}
	// with no stage, there is no legend to show
	if m = loaded(t, Options{}, press('?')); m.legend.open {
		t.Error("a legend with no stage")
	}
}

// explains says whether a legend's line, in one of its columns, says what the entry says: its glyph, then its meaning,
// and after it nothing but a mark, the gap before the next column, or the box's edge.
func explains(line string, e entry) bool {
	plain, want := ansi.Strip(line), pane.Pad(e.glyph, 2, false)+" "+e.meaning
	for at := strings.Index(plain, want); at >= 0; {
		rest := plain[at+len(want):]
		for _, end := range []string{"  ", " │", " " + ToPage, " " + ToRow} {
			if rest == "" || strings.HasPrefix(rest, end) {
				return true
			}
		}
		next := strings.Index(rest, want)
		if next < 0 {
			break
		}
		at += len(want) + next
	}
	return false
}

// cellOf gives the screen cell a text starts at in a line, or -1.
func cellOf(line, text string) int {
	plain := ansi.Strip(line)
	k := strings.Index(plain, text)
	if k < 0 {
		return -1
	}
	return ansi.StringWidth(plain[:k])
}

// Each glyph a stage's rows lead somewhere from is marked in its legend, † for a row under another stage and 🔗 for
// a page, and each mark is on a glyph that leads there; a pull request's state, marked under changes' state heading,
// leads to its page wherever it is drawn.
func TestTheLegendMarksEveryGlyphAClickLeadsSomewhereFrom(t *testing.T) {
	for i, name := range eight {
		m := opened(t, i)
		asCatalog(&m, i)
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

// The glyphs region is what its command makes from the kit's table of glyphs, legend.jq's Go gofmt'd, so a change to
// the table that kit generate has not carried here fails.
func TestTheGlyphsRegionIsWhatTheKitsTableGives(t *testing.T) {
	for _, tool := range []string{"bash", "jq", "gofmt"} {
		if _, err := exec.LookPath(tool); err != nil {
			t.Skipf("no %s", tool)
		}
	}
	source, err := os.ReadFile("glyph.go")
	if err != nil {
		t.Fatal(err)
	}
	_, after, _ := strings.Cut(string(source), "// kit:generate glyphs `")
	command, region, _ := strings.Cut(after, "`\n")
	region, _, ok := strings.Cut(region, "// kit:end glyphs\n")
	if !ok || command == "" {
		t.Fatal("glyph.go has no glyphs region")
	}
	run := exec.Command("bash", "-c", command)
	run.Dir = filepath.Join("..", "..", "..", "..")
	made, err := run.Output()
	if err != nil {
		t.Fatalf("%s: %v", command, err)
	}
	if string(made) != region {
		t.Errorf("the glyphs region is stale; run kit generate:\n%s", made)
	}
}

// Every stage is in one group of the kit's table, the pair that are about one thing, and the bar draws the selected
// tab, and the line under the others, in that group's color: a stage of another group, another color.
func TestTheSelectedTabIsDrawnInItsGroupsColor(t *testing.T) {
	for _, name := range eight {
		if stageColors[name] == "" {
			t.Errorf("%s is in no group", name)
		}
	}
	m := loaded(t, options())
	for i, name := range eight {
		mark, _ := style.In(stageColors[name])
		open, _, _ := strings.Cut(mark("x"), "x")
		bar := strings.Split(m.View().Content, "\n")[1:4]
		if !strings.Contains(bar[1], mark(name)) || !strings.Contains(bar[2], open+"─") {
			t.Errorf("tab %d, %s, is not drawn in its group's color, %s", i, name, stageColors[name])
		}
		m = drive(m, right)
	}
	if stageColors["backlog"] != stageColors["underway"] || stageColors["backlog"] == stageColors["desk"] {
		t.Errorf("the groups' colors: %v", stageColors)
	}
}

// The legend's glyph for the workspace is a group's in the workspace menu, 📚, and it says it names the workspace.
func TestTheTitlesGlyphIsTheWorkspaces(t *testing.T) {
	if title != (entry{shell.Books, "the workspace"}) {
		t.Errorf("the title's entry: %+v", title)
	}
}

// The legend explains every glyph on the screen, the frame's and the controls' too, each where it means what it says,
// in the order each first appears, reading from the top left: the kit menu's =, the workspace menu's -, the search,
// the tabs' numbers, a stage's glyph, the presets' ▼ on the control line, and a sortable column's · on the rule.
func TestTheLegendExplainsEveryGlyphOnTheScreenInOrder(t *testing.T) {
	m := loaded(t, viewing(""), tea.WindowSizeMsg{Width: 120, Height: 40}, press('?'))
	var order []string
	for _, l := range m.legendLines() {
		order = append(order, ansi.Strip(l))
	}
	want := []string{shell.App + "  the kit menu", shell.Workspaces + "  the workspace menu",
		grid.Search + " the search: / or a click", tabs.Marker(0, true) + "  the selected tab's number",
		tabs.Marker(1, false) + "  a tab's number, which selects it", Issue + " the backlog stage",
		table.Presets + "  the presets' menu", Sortable + "  sortable: a click sorts"}
	at := -1
	for _, w := range want {
		k := slices.Index(order, w)
		if k < 0 || k < at {
			t.Errorf("%q is not in the legend after the line before it:\n%s", w, strings.Join(order, "\n"))
		}
		at = k
	}
	// a glyph the screen does not show is not listed: no tab is stale, and nothing sorts
	for _, absent := range []string{tabs.Stale, "▲", "▼  descending"} {
		if slices.ContainsFunc(order, func(l string) bool { return strings.HasPrefix(l, absent) }) {
			t.Errorf("the legend lists %q, which the screen does not show", absent)
		}
	}
}
