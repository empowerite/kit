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
	want := "  │ 🎫 backlog 15 │  🛫 underway 58 │  🛬 incoming 2 │  💻 desk 1 │  🌿 worktrees 4 │  🔀 changes 3 │  " +
		"🏭 builds 4 │  📦 releases 3 │"
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
		"  ➀╶──────────────╮➁╶───────────────╮➂╶──────────────╴➍╶──────────╮➄╶───────────────╮➅╶─────────────╮" +
			"➆╶────────────╮➇╶──────────────╮",
		"    🎫 backlog 15 │  🛫 underway 58 │  🛬 incoming 2  │ 💻 desk 1 │  🌿 worktrees 4 │  🔀 changes 3 │  " +
			"🏭 builds 4 │  📦 releases 3 │",
		"  ────────────────╰─────────────────╰─────────────────╯           ╰─────────────────╰───────────────╰" +
			"──────────────╰────────────────╵",
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
		!strings.Contains(v, style.Mark("╰")) || !strings.Contains(v, style.Mark(strings.Repeat("─", 15))) ||
		!strings.Contains(v, style.Off("╰")) {
		t.Errorf("the edges are not in the accent: %q", v)
	}
}

// A screen too narrow for every tab scrolls to center the selected one, with ◃ in the gutter where tabs are hidden
// before it and ▹ where they are hidden after it, a tab at either edge cut there; a triangle that would cut the
// selected tab is hidden, the cut tab alone saying more stand past the edge.
func TestANarrowScreenScrollsToCenterTheSelectedTab(t *testing.T) {
	if got, want := row(stages(5, 40), 1), " ◃rktrees 4  │ 🔀 changes 3 │  🏭 buil ▹"; got != want {
		t.Errorf("at 40, tab 5:\n got %q\nwant %q", got, want)
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
	// from tab 0 at 40, ▹ shows whole the tab it cut, and the next, until the selected tab would be cut
	b := stages(0, 40)
	b, _ = b.Update(testkit.Click(39, 2))
	if got := row(b, 1); !strings.Contains(got, "🛫 underway 58 │") || !strings.Contains(got, "🎫 backlog 15") {
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
	b, _ = b.Update(tea.KeyPressMsg{Code: tea.KeyLeft})
	if b.Selected != 2 || b.layout().scroll != l.scroll {
		t.Errorf("left to a tab the view shows: tab %d, scroll %d, was %d", b.Selected, b.layout().scroll, l.scroll)
	}
	b.Selected = 7
	if got := b.layout(); !got.shows(got.scroll, b.screen(), 7) {
		t.Errorf("the last tab, selected, is not whole: %q", row(b, 1))
	}
}

func TestLeftAndRightMoveOnlyWhileTheBarHasTheKeyboard(t *testing.T) {
	b := stages(0, 80)
	if got, ok := b.Update(tea.KeyPressMsg{Code: tea.KeyLeft}); !ok || got.Selected != 0 {
		t.Errorf("left at the first tab: %d, %v", got.Selected, ok)
	}
	b.Selected = 7
	if got, ok := b.Update(tea.KeyPressMsg{Code: tea.KeyRight}); !ok || got.Selected != 7 {
		t.Errorf("right at the last tab: %d, %v", got.Selected, ok)
	}
	b.Selected = 0
	b.Focused = false
	if got, ok := b.Update(tea.KeyPressMsg{Code: tea.KeyRight}); ok || got.Selected != 0 {
		t.Errorf("right without the keyboard: %d, %v", got.Selected, ok)
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
	if strings.Contains(b.View(), dim+" 🛬 incoming 2") {
		t.Errorf("an other tab is dim while the bar has the keyboard: %q", b.View())
	}
	b.Focused = false
	got := b.View()
	if !strings.Contains(got, dim+" 🛬 incoming 2") || !strings.Contains(got, dim+"➂") || !strings.Contains(got, dim+"╶─") ||
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
	if got := plain(bare); !strings.HasPrefix(got, "  ➀╶") || !strings.HasPrefix(row(bare, 1), "    nonesuch !  │") {
		t.Errorf("a tab without a glyph: %q", got)
	}
}

// A reserve lays every count out as wide as it, at least, and a bar with menus keeps the room for the menu's glyph on
// every tab, drawing it on the selected one alone: a count that grows, or a selection that moves, moves no tab.
func TestTheReserveAndTheMenuKeepEveryTabWhereItIs(t *testing.T) {
	bar := func(selected int, count string) Bar {
		return Bar{Selected: selected, Width: 60, Reserve: 7, Menus: true,
			Tabs: []Tab{{"🎫", "backlog", count}, {"🛫", "underway", "4"}, {"💻", "desk", ""}}}
	}
	at := func(b Bar) []int { return b.layout().at }
	base := at(bar(0, "3"))
	for _, b := range []Bar{bar(0, "12/345"), bar(1, "3"), bar(2, "123/456")} {
		if got := at(b); !slices.Equal(got, base) {
			t.Errorf("the tabs moved: %v, want %v", got, base)
		}
	}
	if got := row(bar(1, "3"), 1); strings.Count(got, Menu) != 1 || !strings.Contains(got, "4 "+Menu+" │") {
		t.Errorf("the menu's glyph is not on the selected tab alone, after its count: %q", got)
	}
	// the glyph is bold only while the menu is open, and the count is then blank, the glyph in its column
	open := bar(1, "3")
	closed := open.View()
	if strings.Contains(closed, style.Menu(Menu)) || !strings.Contains(closed, style.Tint(Menu)) {
		t.Errorf("the glyph of a closed menu is bold: %q", closed)
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
			b.Menus = true
			want := -1
			if line := strings.Split(ansi.Strip(b.View()), "\n")[1]; strings.Contains(line, Menu) {
				want = ansi.StringWidth(line[:strings.Index(line, Menu)])
			}
			if got := b.MenuAt(); got != want {
				t.Errorf("at %d, tab %d: MenuAt %d, drawn at %d", width, selected, got, want)
			}
		}
	}
	if (Bar{Tabs: []Tab{{"", "a", ""}}, Width: 40}).MenuAt() != -1 || (Bar{Menus: true}).MenuAt() != -1 {
		t.Error("a bar without menus, or without tabs, has a menu glyph")
	}
}

// A joined bar's last line turns down in its last column into a line under the bar: the line under the tabs as ┐, the
// last tab's edge as ┤ where the line under it runs up to it, and as │ under the selected tab, open to the pane.
func TestAJoinedBarTurnsDownInItsLastColumn(t *testing.T) {
	for _, c := range []struct {
		selected, width int
		want            string
	}{{0, natural(), "┤"}, {7, natural(), "│"}, {0, 200, "┐"}, {7, 200, "┐"}, {0, 80, "┐"}, {3, 60, "┐"},
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
			b.Reserve, b.Menus = len("xxx/yyy"), true
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
// with a small reserve stays narrow.
func TestATabsOwnReserveKeepsItsCountsRoomAndNoMore(t *testing.T) {
	b := Bar{Tabs: []Tab{{"🎫", "backlog", "3"}, {"🛫", "underway", "12"}}, Width: 200, Reserves: []int{3, 5}}
	if got := []int{b.wants(0), b.wants(1)}; !slices.Equal(got, []int{ansi.StringWidth("🎫 backlog") + 1 + 3 + 1,
		ansi.StringWidth("🛫 underway") + 1 + 5 + 1}) {
		t.Errorf("wants: %v", got)
	}
	before := row(b, 0)
	b.Tabs[0].Count, b.Tabs[1].Count = "2/3", "9/12"
	if after := row(b, 0); before != after {
		t.Errorf("a count within its reserve moved the tabs: %v, then %v", before, after)
	}
}

// The bar's keys: while it has the keyboard, left and right, and down where it says what down enters; a tab's number
// from anywhere, nine at most; and : for the tab's menu where the bar has menus. A bar of no tabs has none.
func TestTheBarNamesItsKeys(t *testing.T) {
	keys := func(b Bar, down string) string {
		var out []string
		for _, k := range b.Keys(down).Bindings {
			out = append(out, k.Key+" "+k.Does)
		}
		return strings.Join(out, ", ")
	}
	b := stages(0, 120)
	b.Focused, b.Menus = true, true
	if got := keys(b, "preset"); got != "←→ tab, ↓ preset, 1-8 jump, : tab menu" {
		t.Errorf("focused, with menus: %q", got)
	}
	b.Focused, b.Menus = false, false
	if got := keys(b, "preset"); got != "1-8 jump" {
		t.Errorf("not focused: %q", got)
	}
	if got := keys(Bar{}, ""); got != "" {
		t.Errorf("no tabs: %q", got)
	}
}
