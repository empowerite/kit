package tabs

import (
	"slices"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/empowerite/kit/projects/tui/internal/style"
	"github.com/empowerite/kit/projects/tui/internal/testkit"
)

// stages is a bar like kit flow's: eight tabs with glyphs and counts.
func stages(selected int, width int) Bar {
	return Bar{Selected: selected, Focused: true, Width: width, Row: 1, Tabs: []Tab{
		{"🎫", "backlog", "15"}, {"🛫", "underway", "58"}, {"🛬", "incoming", "2"}, {"💻", "desk", "1"},
		{"🌿", "worktrees", "4"}, {"🔀", "changes", "3"}, {"🏭", "builds", "4"}, {"📦", "releases", "3"},
	}}
}

// plain gives the bar's lines without their escapes.
func plain(b Bar) string { return ansi.Strip(b.View()) }

// row gives one of the bar's lines without its escapes: 0 over the tabs, 1 the tabs, 2 under them.
func row(b Bar, k int) string { return strings.Split(plain(b), "\n")[k] }

func TestLabelIsTheSelectedTabAsTheBarDrawsIt(t *testing.T) {
	if got := (Bar{}).Label(); got != "" {
		t.Errorf("no tabs: %q", got)
	}
	b := stages(1, 120)
	b.Color = "33"
	mark, _ := style.In("33")
	// one space after the name, the stale dot's cell not kept, as the bar keeps it
	if got := b.Label(); ansi.Strip(got) != "🛫 underway 58 "+Menu || !strings.Contains(got, mark("underway")) ||
		!strings.Contains(b.View(), mark("underway")) {
		t.Errorf("the label: %q", got)
	}
	b.MenuOpen = true
	if got := ansi.Strip(b.Label()); got != "🛫 underway    "+Menu {
		t.Errorf("the menu open, the count blank: %q", got)
	}
	b.MenuOpen, b.Stale = false, []bool{false, true}
	if got := ansi.Strip(b.Label()); got != "🛫 underway"+Stale+" 58 "+Menu {
		t.Errorf("stale, the dot after the name: %q", got)
	}
	// the numbered label's dot, as a click on it needs it, after the number, a space, and the name
	if got := b.LabelStaleAt(); got != ansi.StringWidth(Marker(1, true)+" underway") {
		t.Errorf("the numbered label's dot at %d: %q", got, ansi.Strip(b.NumberedLabel()))
	}
	b.Stale = nil
	if got := b.LabelStaleAt(); got != -1 {
		t.Errorf("a label without a dot has it at %d", got)
	}
	b.MenuOpen, b.Tabs[1].Count = false, ""
	if got := ansi.Strip(b.Label()); got != "🛫 underway "+Menu {
		t.Errorf("no count: %q", got)
	}
}

func TestTheMarkersAreNumberedToTenThenNone(t *testing.T) {
	cases := []struct {
		i        int
		selected bool
		want     string
	}{{0, false, "➀"}, {0, true, "➊"}, {8, false, "➈"}, {8, true, "➒"}, {9, false, "➉"}, {9, true, "➓"}}
	for _, c := range cases {
		if got := Marker(c.i, c.selected); got != c.want || ansi.StringWidth(got) != 1 {
			t.Errorf("Marker(%d, %v) = %q, %d cells; want %q, one cell", c.i, c.selected, got, ansi.StringWidth(got),
				c.want)
		}
	}
	if Marker(10, false) != "" || Marker(12, true) != "" {
		t.Errorf("a tab past the tenth has a number: %q %q", Marker(10, false), Marker(12, true))
	}
}

// Past the tenth, a tab's number cell is its border: ─ over it, or ╭ when selected; 0 selects the tenth.
func TestPastTheTenthATabsBorderRunsThroughItsNumbersCell(t *testing.T) {
	var many []Tab
	for range 12 {
		many = append(many, Tab{"🎫", "t", ""})
	}
	b := Bar{Tabs: many, Selected: 10, Focused: true, Width: 200}
	over := row(b, 0)
	if !strings.Contains(over, "➉") || !strings.Contains(over, "─╴╭") ||
		strings.Count(over, "╮") != 11 || !strings.HasSuffix(over, "╮") {
		t.Errorf("the line over twelve tabs: %q", over)
	}
	if got, ok := b.Update(tea.KeyPressMsg{Code: '0', Text: "0"}); !ok || got.Selected != 9 || !strings.Contains(row(got, 0), "➓") {
		t.Errorf("0 with twelve tabs: tab %d, taken %v", got.Selected, ok)
	}
}

func TestTheBarIsThreeLinesWithinTheScreenAndKeepsEveryNumberWhenItFits(t *testing.T) {
	for width := 12; width <= 160; width++ {
		for selected := range 8 {
			b := stages(selected, width)
			lines := strings.Split(plain(b), "\n")
			if len(lines) != Height {
				t.Fatalf("at %d: %d lines", width, len(lines))
			}
			for _, l := range lines {
				if n := ansi.StringWidth(l); n > width {
					t.Fatalf("at %d, tab %d: a line of %d cells: %q", width, selected, n, l)
				}
			}
			if b.layout().width > b.screen() {
				continue
			}
			for i := range 8 {
				if !strings.Contains(lines[0], Marker(i, i == selected)) {
					t.Errorf("at %d, tab %d: no %s: %q", width, selected, Marker(i, i == selected), lines[0])
				}
			}
		}
	}
}

// natural is the screen's width the eight stages' tabs just fill, each at its own width, the gutter before them.
func natural() int { return stages(0, 1000).layout().width + Gutter }

func TestEveryTabIsWholeAtItsOwnWidthAndTheLineUnderRunsOnPastThem(t *testing.T) {
	want := "  │ 🎫 backlog  15 : │  🛫 underway  58   │  🛬 incoming  2   │  💻 desk  1   │  🌿 worktrees  4   │  " +
		"🔀 changes  3   │  🏭 builds  4   │  📦 releases  3   │"
	// a wider screen draws them as wide, and the line under them runs on to its edge
	for _, width := range []int{natural(), 170} {
		if got := row(stages(0, width), 1); got != want {
			t.Errorf("at %d:\n got %q\nwant %q", width, got, want)
		}
	}
	if under := row(stages(0, 170), 2); ansi.StringWidth(under) != 170 || !strings.HasSuffix(under, "╰"+
		strings.Repeat("─", 170-natural())) {
		t.Errorf("the line under at 170: %q", under)
	}
	// narrower, no name is cut: the bar scrolls instead
	for width := 20; width < 170; width++ {
		if got := row(stages(0, width), 1); strings.Contains(got, "…") {
			t.Fatalf("at %d, a name is cut: %q", width, got)
		}
	}
}

// The selected tab has the line over it and none under; the others a dim border, from the number over them to ╮,
// down │, and out to the right with ╰ over the line under them, which is in the accent; the tab before the selected
// one has no edge after it: its line over it ends in a half line short of the selected number, and the line under it
// runs on.
func TestTheSelectedTabIsOpenBelowAndItsEdgesCurve(t *testing.T) {
	want := []string{
		"  ➀╶─────────────────╮➁╶──────────────────╮➂╶─────────────────╴➍╶─────────────╮➄╶──────────────────╮" +
			"➅╶────────────────╮➆╶───────────────╮➇╶─────────────────╮",
		"    🎫 backlog  15   │  🛫 underway  58   │  🛬 incoming  2    │ 💻 desk  1 : │  🌿 worktrees  4   │" +
			"  🔀 changes  3   │  🏭 builds  4   │  📦 releases  3   │",
		"  ───────────────────╰────────────────────╰────────────────────╯              ╰────────────────────╰" +
			"──────────────────╰─────────────────╰───────────────────╵",
	}
	b := stages(3, natural())
	if got := strings.Split(plain(b), "\n"); !slices.Equal(got, want) {
		t.Errorf("the bar:\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
	// the first tab and the last have the same edges, the first's after the gutter, each over a half line where no tab
	// is beside it
	first, last := strings.Split(plain(stages(0, natural())), "\n"), strings.Split(plain(stages(7, natural())), "\n")
	for k, edge := range []string{"➊", "│", "╵"} {
		if !strings.HasPrefix(first[k], "  "+edge) {
			t.Errorf("the first tab's line %d does not start with %s: %q", k, edge, first[k])
		}
	}
	for k, edge := range []string{"╮", "│", "╵"} {
		if !strings.HasSuffix(last[k], edge) {
			t.Errorf("the last tab's line %d does not end with %s: %q", k, edge, last[k])
		}
	}
	// short of the screen's edge, the last tab's edge curves out to the line that runs on
	if under := row(stages(7, 170), 2); !strings.Contains(under, "╰─") || strings.Contains(under, "╵") {
		t.Errorf("the selected last tab short of the edge: %q", under)
	}
	// the selected tab's lines are in the accent, and so is the line under every tab; the others' hooks are dim
	if v := b.View(); !strings.Contains(v, style.Mark("➍")) || !strings.Contains(v, style.Mark("╯")) ||
		!strings.Contains(v, style.Mark("╰")) || !strings.Contains(v, style.Mark(strings.Repeat("─", 17))) ||
		!strings.Contains(v, style.Off("╰")) {
		t.Errorf("the edges are not in the accent: %q", v)
	}
}

// A screen too narrow for every tab scrolls to center the selected one, with ◃ in the gutter where tabs are hidden
// before it and ▹ where they are hidden after it, a tab at either edge cut there; a triangle that would cut the
// selected tab is hidden, the cut tab alone saying more stand past the edge.
func TestANarrowScreenScrollsToCenterTheSelectedTab(t *testing.T) {
	if got, want := row(stages(5, 46), 1), " ◃ktrees  4    │ 🔀 changes  3 : │  🏭 build ▹"; got != want {
		t.Errorf("at 46, tab 5:\n got %q\nwant %q", got, want)
	}
	if got := row(stages(0, 40), 1); strings.Contains(got, Before) || strings.Contains(got, After) {
		t.Errorf("at 40, tab 0, a triangle that would cut backlog shows: %q", got)
	}
	if got := row(stages(7, 40), 1); strings.Contains(got, Before) || strings.Contains(got, After) {
		t.Errorf("at 40, tab 7, a triangle that would cut releases shows: %q", got)
	}
	// a click on a triangle scrolls the view and never selects a tab; see TestATriangleScrollsOneTabAndNeverSelects
	// the margin without a triangle is no tab's
	if _, ok := stages(3, 170).Update(testkit.Click(0, 2)); ok {
		t.Error("a click on the margin was taken")
	}
}

// A triangle, clicked, scrolls the view to show whole the nearest tab it hides, and leaves the selected tab and the
// keyboard where they were; the selected tab shows whole wherever the view goes, and a triangle that would cut it is
// dim, and moves nothing. Clicked over and over, ◃ and ▹ each stop, dim or gone, and the view is where it stopped.
func TestATriangleScrollsOneTabAndNeverSelects(t *testing.T) {
	for _, width := range []int{30, 40, 60, 80, 100, 120} {
		for s := range 8 {
			for _, focused := range []bool{true, false} {
				for _, right := range []bool{false, true} {
					b := stages(s, width)
					b.Focused = focused
					x := 0
					if right {
						x = width - 1
					}
					for range 10 {
						l, w := b.layout(), b.screen()
						to, ok := l.before(w, s)
						shown := l.scroll > 0
						if right {
							to, ok = l.after(w, s)
							shown = l.scroll+w < l.width
						}
						got, taken := b.Update(testkit.Click(x, 2))
						after := got.layout()
						switch {
						case !shown && taken && !right:
							t.Fatalf("at %d, tab %d, no ◃, but a click on the margin was taken", width, s)
						case !shown:
						case !taken || got.Selected != s || got.Focused != focused:
							t.Fatalf("at %d, a click on the triangle at %d: tab %d (was %d), focused %v, taken %v",
								width, x, got.Selected, s, got.Focused, taken)
						case !after.shows(after.scroll, w, s):
							t.Fatalf("at %d, tab %d, the triangle at %d cut the selected tab:\n%s", width, s, x,
								plain(got))
						case ok && after.scroll != to, !ok && after.scroll != l.scroll:
							t.Fatalf("at %d, tab %d, the triangle at %d: scroll %d, was %d, want %d (moves %v)", width,
								s, x, after.scroll, l.scroll, to, ok)
						}
						glyph := Before
						if right {
							glyph = After
						}
						if drawn := strings.Contains(plain(b), glyph); shown && drawn != ok {
							t.Fatalf("at %d, tab %d, the triangle at %d is drawn %v and moves %v:\n%s", width, s, x, drawn,
								ok, plain(b))
						}
						if !shown || !ok {
							break
						}
						b = got
					}
				}
			}
		}
	}
	// from tab 0 at 46, ▹ shows whole the tab it cut, and the next, until the selected tab would be cut
	b := stages(0, 46)
	b, _ = b.Update(testkit.Click(45, 2))
	if got := row(b, 1); !strings.Contains(got, "🛫 underway  58   │") || !strings.Contains(got, "🎫 backlog  15") {
		t.Errorf("▹ from tab 0: %q", got)
	}
	if got := row(b, 1); strings.Contains(got, After) {
		t.Errorf("▹ would cut backlog, so it is hidden: %q", got)
	}
}

// The view stays where the screen keeps it while the selected tab shows whole there; a tab selected that it cuts, or
// does not show, is centered.
func TestTheViewStaysWhileTheSelectedTabShowsWhole(t *testing.T) {
	b := stages(3, 60)
	l := b.layout()
	b.Scroll = l.scroll
	b, _ = b.Update(tea.KeyPressMsg{Code: tea.KeyTab, Mod: tea.ModShift})
	if b.Selected != 2 || b.layout().scroll != l.scroll {
		t.Errorf("shift-tab to a tab the view shows: tab %d, scroll %d, was %d", b.Selected, b.layout().scroll, l.scroll)
	}
	b.Selected = 7
	if got := b.layout(); !got.shows(got.scroll, b.screen(), 7) {
		t.Errorf("the last tab, selected, is not whole: %q", row(b, 1))
	}
}

func TestTabAndShiftTabMoveOnlyWhileTheBarHasTheKeyboard(t *testing.T) {
	b := stages(0, 80)
	if got, ok := b.Update(tea.KeyPressMsg{Code: tea.KeyTab, Mod: tea.ModShift}); !ok || got.Selected != 0 {
		t.Errorf("shift-tab at the first tab: %d, %v", got.Selected, ok)
	}
	if got, ok := b.Update(tea.KeyPressMsg{Code: tea.KeyTab}); !ok || got.Selected != 1 {
		t.Errorf("tab from the first tab: %d, %v", got.Selected, ok)
	}
	b.Selected = 7
	if got, ok := b.Update(tea.KeyPressMsg{Code: tea.KeyTab}); !ok || got.Selected != 7 {
		t.Errorf("tab at the last tab: %d, %v", got.Selected, ok)
	}
	b.Selected = 0
	b.Focused = false
	if got, ok := b.Update(tea.KeyPressMsg{Code: tea.KeyTab}); ok || got.Selected != 0 {
		t.Errorf("tab without the keyboard: %d, %v", got.Selected, ok)
	}
}

// Left and right fold and unfold, and nothing folds on the bar: they move nothing on it.
func TestLeftAndRightAreNotTheBars(t *testing.T) {
	b := stages(3, 80)
	for _, k := range []tea.KeyPressMsg{{Code: tea.KeyLeft}, {Code: tea.KeyRight}} {
		if got, ok := b.Update(k); ok || got.Selected != 3 {
			t.Errorf("%s on the bar: %d, %v", k, got.Selected, ok)
		}
	}
}

func TestANumberSelectsItsTabFromAnywhere(t *testing.T) {
	keys := []tea.KeyPressMsg{
		{Code: '3', Text: "3"},
		{Code: tea.KeyKp3},
		{Code: '3', Mod: tea.ModAlt},
		{Code: '3', Mod: tea.ModSuper},
	}
	for _, k := range keys {
		b := stages(0, 80)
		b.Focused = false
		if got, ok := b.Update(k); !ok || got.Selected != 2 || !got.Focused {
			t.Errorf("%s: tab %d, taken %v, focused %v", k, got.Selected, ok, got.Focused)
		}
	}
	short := Bar{Tabs: []Tab{{"👋", "intro", ""}, {"🧱", "basics", ""}}, Width: 80}
	if got, ok := short.Update(tea.KeyPressMsg{Code: '3', Text: "3"}); ok || got.Selected != 0 {
		t.Errorf("3 with two tabs: tab %d, taken %v", got.Selected, ok)
	}
	if _, ok := short.Update(tea.KeyPressMsg{Code: '0', Text: "0"}); ok {
		t.Error("0 was taken")
	}
}

func TestAClickOnTheBarSelectsTheTabUnderIt(t *testing.T) {
	b := stages(0, 80)
	b.Focused = false
	x, click := testkit.Column(row(b, 0), '➃'), testkit.Click
	if got, ok := b.Update(click(x, 1)); !ok || got.Selected != 3 || !got.Focused {
		t.Errorf("a click on ➃: tab %d, taken %v, focused %v", got.Selected, ok, got.Focused)
	}
	if got, ok := b.Update(click(x+4, 1)); !ok || got.Selected != 3 {
		t.Errorf("a click on ➃'s rule: tab %d, taken %v", got.Selected, ok)
	}
	for _, y := range []int{2, 3} {
		if got, ok := b.Update(click(x, y)); !ok || got.Selected != 3 {
			t.Errorf("a click on ➃'s line %d: tab %d, taken %v", y, got.Selected, ok)
		}
	}
	for _, y := range []int{0, 4} {
		if _, ok := b.Update(click(x, y)); ok {
			t.Errorf("a click off the bar, on line %d, was taken", y)
		}
	}
	if _, ok := b.Update(tea.MouseClickMsg{X: x, Y: 1, Button: tea.MouseRight}); ok {
		t.Error("a right click was taken")
	}
}

func TestTheOtherTabsAreDimWhileThePaneHasTheKeyboard(t *testing.T) {
	b := stages(0, 80)
	dim, _, _ := strings.Cut(style.Off("x"), "x")
	if strings.Contains(b.View(), dim+" 🛬 incoming") {
		t.Errorf("an other tab is dim while the bar has the keyboard: %q", b.View())
	}
	b.Focused = false
	got := b.View()
	if !strings.Contains(got, dim+" 🛬 incoming") || !strings.Contains(got, dim+"➂") || !strings.Contains(got, dim+"╶─") ||
		!strings.Contains(got, style.Mark("backlog")) {
		t.Errorf("the other tabs and their lines are not dim beside the selected one: %q", got)
	}
}

// A stale tab has a yellow dot after its name, which stays yellow while the other tabs dim; it moves no tab, and every
// line but the middle one's dot is as it was.
func TestAStaleTabHasADotAfterItsNameAndMovesNothing(t *testing.T) {
	for _, focused := range []bool{true, false} {
		b := stages(0, 120)
		b.Focused = focused
		fresh := strings.Split(plain(b), "\n")
		b.Stale = []bool{true, false, true}
		got := b.View()
		lines := strings.Split(ansi.Strip(got), "\n")
		if !strings.Contains(lines[1], "backlog"+Stale) || !strings.Contains(lines[1], "incoming"+Stale) ||
			strings.Contains(lines[1], "underway"+Stale) {
			t.Errorf("focused %v: the stale tabs' middle line: %q", focused, lines[1])
		}
		if !strings.Contains(got, style.Warn(Stale)) {
			t.Errorf("focused %v: the dot is not yellow: %q", focused, got)
		}
		if lines[0] != fresh[0] || lines[2] != fresh[2] ||
			strings.ReplaceAll(lines[1], Stale, " ") != fresh[1] {
			t.Errorf("focused %v: the dot moved the tabs:\n%s\nwas\n%s", focused, strings.Join(lines, "\n"),
				strings.Join(fresh, "\n"))
		}
	}
}

// StaleAt says which tab's stale dot is drawn at a cell of the screen, as a click needs it: each dot the bar draws,
// on its middle line, right after its name, or what the view shows of it, and no other cell, whatever the view
// scrolls to; a tab that is not stale has none.
func TestStaleAtIsWhereEachDotIsDrawn(t *testing.T) {
	for _, width := range []int{40, 80, 170} {
		for selected := range 8 {
			b := stages(selected, width)
			b.Stale = []bool{true, false, true, true, false, true, true, true}
			lines := strings.Split(ansi.Strip(b.View()), "\n")
			for y := range Height {
				x := 0
				for _, r := range lines[y] {
					got, cell := b.StaleAt(x, b.Row+y), string(r)
					before := strings.Fields(ansi.Cut(lines[y], 0, x))
					switch {
					case cell == Stale && (got < 0 || !b.Stale[got] || len(before) == 0 ||
						!strings.HasSuffix(b.Tabs[got].Name, strings.TrimLeft(before[len(before)-1], Before))):
						t.Errorf("at %d, tab %d: the dot at %d, %d is tab %d's", width, selected, x, y, got)
					case cell != Stale && got >= 0:
						t.Errorf("at %d, tab %d: no dot at %d, %d, but tab %d", width, selected, x, y, got)
					}
					x += ansi.StringWidth(cell)
				}
			}
		}
	}
	if (Bar{}).StaleAt(Gutter, 1) != -1 {
		t.Error("a bar without tabs has a dot")
	}
}

func TestTheKeyboardChangesTheColorsAloneAndNeverTheSelectedTabs(t *testing.T) {
	b := stages(0, 80)
	focused := b.View()
	b.Focused = false
	if focused == b.View() {
		t.Error("the bar draws the same with and without the keyboard")
	}
	if plain(b) != ansi.Strip(focused) {
		t.Error("the keyboard changed the bar's text, not only its color")
	}
}

func TestTheBarsEdges(t *testing.T) {
	// no tabs: nothing drawn, nothing taken
	var empty Bar
	if _, ok := empty.Update(tea.KeyPressMsg{Code: '1', Text: "1"}); ok || empty.View() != "" {
		t.Error("an empty bar drew or took a key")
	}
	// a key the bar has no use for
	if _, ok := stages(0, 80).Update(tea.KeyPressMsg{Code: 'x', Text: "x"}); ok {
		t.Error("x was taken")
	}
	// a tab without a glyph has its name under its number
	bare := Bar{Selected: 1, Width: 40, Tabs: []Tab{{"", "nonesuch", "!"}, {"🎫", "backlog", "3"}}}
	if got := plain(bare); !strings.HasPrefix(got, "  ➀╶") || !strings.HasPrefix(row(bare, 1), "    nonesuch  !    │") {
		t.Errorf("a tab without a glyph: %q", got)
	}
}

// A reserve lays every count out as wide as it, at least, and every bar keeps the room for the menu's glyph on every
// tab, drawing it on the selected one alone: a count that grows, or a selection that moves, moves no tab.
func TestTheReserveAndTheMenuKeepEveryTabWhereItIs(t *testing.T) {
	bar := func(selected int, count string) Bar {
		return Bar{Selected: selected, Width: 60, Reserve: 7,
			Tabs: []Tab{{"🎫", "backlog", count}, {"🛫", "underway", "4"}, {"💻", "desk", ""}}}
	}
	at := func(b Bar) []int { return b.layout().at }
	base := at(bar(0, "3"))
	for _, b := range []Bar{bar(0, "12/345"), bar(1, "3"), bar(2, "123/456")} {
		if got := at(b); !slices.Equal(got, base) {
			t.Errorf("the tabs moved: %v, want %v", got, base)
		}
	}
	if got := row(bar(1, "3"), 1); strings.Count(got, Menu) != 1 || !strings.Contains(got, "underway  4 ") ||
		!strings.Contains(got, " "+Menu+" │") {
		t.Errorf("the menu's glyph is not on the selected tab alone, at its end: %q", got)
	}
	// the glyph is bold in the tab's color while the menu is closed, as the tab's name is, and in the menu's own
	// while it is open, the count then blank, the glyph in its column
	open := bar(1, "3")
	closed := open.View()
	if strings.Contains(closed, style.Menu(Menu)) || !strings.Contains(closed, style.Mark(Menu)) {
		t.Errorf("the glyph of a closed menu is not bold in the tab's color: %q", closed)
	}
	if open.MenuOpen = true; !strings.Contains(open.View(), style.Menu(Menu)) {
		t.Errorf("the glyph of an open menu is not bold")
	}
	if got := row(open, 1); !strings.Contains(got, "  "+Menu+" │") || strings.Contains(got, "4 "+Menu) ||
		strings.Index(got, Menu) != strings.Index(row(bar(1, "3"), 1), Menu) {
		t.Errorf("the count beside an open menu: %q", got)
	}
}

// MenuAt says where the selected tab's menu glyph is drawn, as a click needs it, and -1 when it is not drawn.
func TestMenuAtIsWhereTheGlyphIsDrawn(t *testing.T) {
	for _, width := range []int{40, 80, 170} {
		for selected := range 8 {
			b := stages(selected, width)
			want := -1
			if line := strings.Split(ansi.Strip(b.View()), "\n")[1]; strings.Contains(line, Menu) {
				want = ansi.StringWidth(line[:strings.Index(line, Menu)])
			}
			if got := b.MenuAt(); got != want {
				t.Errorf("at %d, tab %d: MenuAt %d, drawn at %d", width, selected, got, want)
			}
		}
	}
	if (Bar{}).MenuAt() != -1 {
		t.Error("a bar without tabs has a menu glyph")
	}
}

// OnSelected takes every cell of the selected tab, on each of the bar's lines, and none of the tabs either side of it,
// none off the bar, and none on a bar without tabs.
func TestOnSelectedIsTheSelectedTabWhole(t *testing.T) {
	b := stages(2, 170)
	b.Row = 4
	l := b.layout()
	first, last := Gutter+l.at[2]-l.scroll, Gutter+l.at[2]+l.parts[2]-1-l.scroll
	for y := b.Row; y < b.Row+Height; y++ {
		for _, x := range []int{first, b.MenuAt(), last} {
			if !b.OnSelected(x, y) {
				t.Errorf("%d, %d is not on the selected tab, %d to %d", x, y, first, last)
			}
		}
	}
	if b.OnSelected(first-1, b.Row+1) || b.OnSelected(last+1, b.Row+1) || b.OnSelected(first, b.Row-1) ||
		b.OnSelected(first, b.Row+Height) {
		t.Errorf("a cell off the selected tab, %d to %d, is on it", first, last)
	}
	if (Bar{}).OnSelected(Gutter, 0) {
		t.Error("a bar without tabs takes the click")
	}
}

// A joined bar's last line turns down in its last column into a line under the bar: the line under the tabs as ┐, the
// last tab's edge as ┤ where the line under it runs up to it, and as │ under the selected tab, open to the pane.
func TestAJoinedBarTurnsDownInItsLastColumn(t *testing.T) {
	for _, c := range []struct {
		selected, width int
		want            string
	}{{0, natural(), "┤"}, {7, natural(), "│"}, {0, 200, "┐"}, {7, 200, "┐"}, {0, 80, "┐"}, {3, 64, "┐"},
		{1, 12, "╷"}} {
		b := stages(c.selected, c.width)
		b.Joined = true
		under := row(b, 2)
		if got := ansi.Cut(under, c.width-1, c.width); got != c.want || ansi.StringWidth(under) != c.width {
			t.Errorf("selected %d at %d: the last cell is %q, not %q: %q", c.selected, c.width, got, c.want, under)
		}
	}
	// a line that ends in no line keeps it
	if got := joinDown(style.Off("x"), 1); got != style.Off("x") {
		t.Errorf("x joined: %q", got)
	}
}

// However the view scrolls, the middle line keeps every cell in its column: each tab's edge, ╮ over it, stands over │,
// short of the triangles at either end, and MenuAt is where the menu's glyph is drawn. A cut through a two-cell glyph
// at the view's left edge would move the rest of the line a cell right.
func TestEveryLineOfAScrolledBarKeepsItsColumns(t *testing.T) {
	for w := 20; w < 260; w++ {
		for s := range 8 {
			b := stages(s, w)
			b.Reserve = len("xxx/yyy")
			lines := strings.Split(plain(b), "\n")
			for x := 2; x < w-2; x++ {
				if ansi.Cut(lines[0], x, x+1) == "╮" && ansi.Cut(lines[1], x, x+1) != "│" {
					t.Fatalf("at %d, tab %d selected, column %d is not under its edge:\n%s\n%s", w, s, x, lines[0],
						lines[1])
				}
			}
			if at := b.MenuAt(); at >= 0 && ansi.Cut(lines[1], at, at+ansi.StringWidth(Menu)) != Menu {
				t.Fatalf("at %d, tab %d selected, MenuAt says %d: %q", w, s, at, lines[1])
			}
		}
	}
}

// A count that wants attention is bold and red, not dim, on any tab, and stays so while the pane has the keyboard and
// the other tabs are dim; the tabs lie where they lie without it.
func TestACountThatWantsAttentionStandsOut(t *testing.T) {
	for _, focused := range []bool{true, false} {
		b := stages(0, 120)
		b.Focused, b.Alert = focused, []bool{false, false, true}
		got := b.View()
		if !strings.Contains(got, style.Loss("2")) || strings.Contains(got, style.Off("2")) {
			t.Errorf("focused %v: incoming's count is not red: %q", focused, got)
		}
		without := stages(0, 120)
		without.Focused = focused
		if plain(b) != plain(without) {
			t.Errorf("focused %v: the alert moved the tabs:\n%s", focused, plain(b))
		}
	}
	// the selected tab's count wants attention too
	b := stages(2, 120)
	b.Alert = []bool{false, false, true}
	if !strings.Contains(b.View(), style.Loss("2")) {
		t.Errorf("the selected tab's count is not red: %q", b.View())
	}
}

// Each tab's reserve keeps room for the count it may grow to, so a count that grows within it moves no tab, and a tab
// with a small reserve stays narrow; each keeps room for the menu's glyph too.
func TestATabsOwnReserveKeepsItsCountsRoomAndNoMore(t *testing.T) {
	b := Bar{Tabs: []Tab{{"🎫", "backlog", "3"}, {"🛫", "underway", "12"}}, Width: 200, Reserves: []int{3, 5}}
	if got := []int{b.wants(0), b.wants(1)}; !slices.Equal(got, []int{ansi.StringWidth("🎫 backlog") + 2 + 3 + 2 + 1,
		ansi.StringWidth("🛫 underway") + 2 + 5 + 2 + 1}) {
		t.Errorf("wants: %v", got)
	}
	before := row(b, 0)
	b.Tabs[0].Count, b.Tabs[1].Count = "2/3", "9/12"
	if after := row(b, 0); before != after {
		t.Errorf("a count within its reserve moved the tabs: %v, then %v", before, after)
	}
}

// The bar's keys: while it has the keyboard, left and right, and down where it says what down enters; a tab's number
// from anywhere, nine at most; and : for the tab's menu, which every bar has. A bar of no tabs has none.
func TestTheBarNamesItsKeys(t *testing.T) {
	keys := func(b Bar, down string) string {
		var out []string
		for _, k := range b.Keys(down).Bindings {
			out = append(out, k.Key+" "+k.Does)
		}
		return strings.Join(out, ", ")
	}
	b := stages(0, 120)
	b.Focused = true
	if got := keys(b, "preset"); got != "⇥·⇧⇥ tab, ↓ preset, 1-8 jump, : tab menu" {
		t.Errorf("focused: %q", got)
	}
	b.Focused = false
	if got := keys(b, "preset"); got != "1-8 jump, : tab menu" {
		t.Errorf("not focused: %q", got)
	}
	if got := keys(Bar{}, ""); got != "" {
		t.Errorf("no tabs: %q", got)
	}
}

// A tab's count follows its name after two cells, the first the stale dot's, blank on a tab that is not stale; the
// room a reserve keeps, or a wider tab spares, comes after the count, and the menu's glyph ends the selected tab.
func TestACountFollowsItsNameAfterTheDotsCellAndASpace(t *testing.T) {
	b := Bar{Width: 200, Reserves: []int{4, 1}, Stale: []bool{false, true},
		Tabs: []Tab{{"🎫", "backlog", "3"}, {"🛫", "underway", "12"}}}
	got := row(b, 1)
	for _, want := range []string{"🎫 backlog  3    " + Menu + " │", "🛫 underway" + Stale + " 12   │"} {
		if !strings.Contains(got, want) {
			t.Errorf("the tabs lack %q: %q", want, got)
		}
	}
	if w := b.wants(0); w != ansi.StringWidth("🎫 backlog")+2+4+1+ansi.StringWidth(Menu)+1 {
		t.Errorf("backlog wants %d", w)
	}
}

// A maximized tab's label, which a screen draws in its title, shows the tab's number, solid, in the tab's color, in
// place of its glyph, for a tab numbered 1 to 9; any other keeps its glyph.
func TestANumberedLabelShowsTheTabsNumberInPlaceOfItsGlyph(t *testing.T) {
	b := stages(1, 120)
	b.Color = "33"
	mark, _ := style.In("33")
	if got := b.NumberedLabel(); ansi.Strip(got) != "➋ underway 58 "+Menu || !strings.Contains(got, mark("➋")) {
		t.Errorf("underway's: %q", got)
	}
	ten := Bar{Selected: 9, Tabs: slices.Repeat([]Tab{{"🎫", "backlog", "3"}}, 10)}
	if got := ansi.Strip(ten.NumberedLabel()); got != "🎫 backlog 3 "+Menu {
		t.Errorf("the tenth's: %q", got)
	}
	if got := (Bar{}).NumberedLabel(); got != "" {
		t.Errorf("no tabs: %q", got)
	}
}
