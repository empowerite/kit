package tabs

import (
	"slices"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/empowerite/kit/projects/tui/internal/style"
)

// stages is a bar like kit flow's: eight tabs with glyphs and counts.
func stages(selected int, width int) Bar {
	return Bar{Selected: selected, Focused: true, Width: width, Row: 1, Tabs: []Tab{
		{"🎫", "backlog", "15"}, {"🛫", "underway", "58"}, {"📥", "incoming", "2"}, {"💻", "desk", "1"},
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
	if !strings.Contains(over, "➉") || strings.ContainsAny(over, "○●") || !strings.Contains(over, "─╴╭") ||
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

func TestEveryTabIsWholeAndTheyShareTheLineInProportionWhenTheyFit(t *testing.T) {
	want := " │ 🎫 backlog 15      │  🛫 underway 58      │  📥 incoming 2      │  💻 desk 1     │  🌿 worktrees 4      │  " +
		"🔀 changes 3      │  🏭 builds 4     │  📦 releases 3      │"
	if got := row(stages(0, 170), 1); got != want {
		t.Errorf("at 170:\n got %q\nwant %q", got, want)
	}
	// narrower, no name is cut: the bar scrolls instead
	for width := 20; width < 170; width++ {
		if got := row(stages(0, width), 1); strings.Contains(got, "…") {
			t.Fatalf("at %d, a name is cut: %q", width, got)
		}
	}
}

func TestShareGivesEachItsProportionAndNoneUnderItsFloor(t *testing.T) {
	if got := share(10, []int{1, 1, 2}, []int{0, 0, 0}); !slices.Equal(got, []int{3, 2, 5}) {
		t.Errorf("share(10, 1:1:2) = %v", got)
	}
	if got := share(20, []int{1, 9}, []int{6, 0}); !slices.Equal(got, []int{6, 14}) {
		t.Errorf("share(20, 1:9, floor 6) = %v", got)
	}
	if got := share(4, []int{3}, []int{4}); !slices.Equal(got, []int{4}) {
		t.Errorf("share(4, 3, floor 4) = %v", got)
	}
}

// The selected tab has the line over it and none under; the others a dim border, from the number over them to ╮,
// down │, and out to the right with ╰ over the line under them, which is in the accent; the tab before the selected
// one has no edge after it: its line over it ends in a half line short of the selected number, and the line under it
// runs on.
func TestTheSelectedTabIsOpenBelowAndItsEdgesCurve(t *testing.T) {
	want := []string{
		" ➀╶───────────────────╮➁╶────────────────────╮➂╶───────────────────╴➍╶──────────────╮➄╶────────────────────╮" +
			"➅╶──────────────────╮➆╶────────────────╮➇╶───────────────────╮",
		"   🎫 backlog 15      │  🛫 underway 58      │  📥 incoming 2       │ 💻 desk 1     │  🌿 worktrees 4      │  " +
			"🔀 changes 3      │  🏭 builds 4     │  📦 releases 3      │",
		" ─────────────────────╰──────────────────────╰──────────────────────╯               ╰──────────────────────╰" +
			"────────────────────╰──────────────────╰─────────────────────╰",
	}
	b := stages(3, 170)
	if got := strings.Split(plain(b), "\n"); !slices.Equal(got, want) {
		t.Errorf("the bar:\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
	// the first tab and the last have the same edges, the first's before it, in the margin, each over a half line where
	// no tab is beside it
	first, last := strings.Split(plain(stages(0, 170)), "\n"), strings.Split(plain(stages(7, 170)), "\n")
	for k, edge := range []string{"➊", "│", "╵"} {
		if !strings.HasPrefix(first[k], " "+edge) {
			t.Errorf("the first tab's line %d does not start with %s: %q", k, edge, first[k])
		}
	}
	for k, edge := range []string{"╮", "│", "╵"} {
		if !strings.HasSuffix(last[k], edge) {
			t.Errorf("the last tab's line %d does not end with %s: %q", k, edge, last[k])
		}
	}
	// the selected tab's lines are in the accent, and so is the line under every tab; the others' hooks are dim
	if v := b.View(); !strings.Contains(v, style.Mark("➍")) || !strings.Contains(v, style.Mark("╯")) ||
		!strings.Contains(v, style.Mark("╰")) || !strings.Contains(v, style.Mark(strings.Repeat("─", 21))) ||
		!strings.Contains(v, style.Off("╰")) {
		t.Errorf("the edges are not in the accent: %q", v)
	}
}

// A screen too narrow for every tab scrolls to center the selected one, with ◃ where tabs are hidden before it and ▹
// where they are hidden after it, a tab at either edge cut there.
func TestANarrowScreenScrollsToCenterTheSelectedTab(t *testing.T) {
	if got, want := row(stages(5, 40), 1), "◃ ktrees 4  │ 🔀 changes 3 │  🏭 build ▹"; got != want {
		t.Errorf("at 40, tab 5:\n got %q\nwant %q", got, want)
	}
	if got := row(stages(0, 40), 1); strings.HasPrefix(got, Before) || !strings.HasSuffix(got, After) {
		t.Errorf("at 40, tab 0: %q", got)
	}
	if got := row(stages(7, 40), 1); !strings.HasPrefix(got, Before) || strings.HasSuffix(got, After) {
		t.Errorf("at 40, tab 7: %q", got)
	}
	// a click on a triangle selects the nearest tab it hides, cut at the edge or wholly hidden, which then shows
	b := stages(5, 40)
	for _, c := range []struct{ x, want int }{{0, 4}, {39, 6}} {
		got, ok := b.Update(tea.MouseClickMsg{X: c.x, Y: 2, Button: tea.MouseLeft})
		if !ok || got.Selected != c.want || !strings.Contains(row(got, 0), Marker(c.want, true)) {
			t.Errorf("a click on the triangle at %d: tab %d, taken %v", c.x, got.Selected, ok)
		}
	}
	// the margin without a triangle is no tab's
	if _, ok := stages(3, 170).Update(tea.MouseClickMsg{X: 0, Y: 2, Button: tea.MouseLeft}); ok {
		t.Error("a click on the margin was taken")
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
	line := []rune(row(b, 0))
	x := 0
	for _, r := range line {
		if r == '➃' {
			break
		}
		x += ansi.StringWidth(string(r))
	}
	click := func(x, y int) tea.MouseClickMsg { return tea.MouseClickMsg{X: x, Y: y, Button: tea.MouseLeft} }
	if got, ok := b.Update(click(x, 1)); !ok || got.Selected != 3 || !got.Focused {
		t.Errorf("a click on ➃: tab %d, taken %v, focused %v", got.Selected, ok, got.Focused)
	}
	if got, ok := b.Update(click(x+4, 1)); !ok || got.Selected != 3 {
		t.Errorf("a click on ➃'s rule: tab %d, taken %v", got.Selected, ok)
	}
	for _, y := range []int{2, 3} {
		if got, ok := b.Update(click(x, y)); !ok || got.Selected != 3 {
			t.Errorf("a click on ④'s line %d: tab %d, taken %v", y, got.Selected, ok)
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
	if strings.Contains(b.View(), dim+" 📥 incoming 2") {
		t.Errorf("an other tab is dim while the bar has the keyboard: %q", b.View())
	}
	b.Focused = false
	got := b.View()
	if !strings.Contains(got, dim+" 📥 incoming 2") || !strings.Contains(got, dim+"➂") || !strings.Contains(got, dim+"╶─") ||
		!strings.Contains(got, style.Mark("backlog")) {
		t.Errorf("the other tabs and their lines are not dim beside the selected one: %q", got)
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
	// a click on the margin is no tab's
	b := stages(0, 80)
	if _, ok := b.Update(tea.MouseClickMsg{X: 0, Y: 1, Button: tea.MouseLeft}); ok {
		t.Error("a click on the margin was taken")
	}
	// a key the bar has no use for
	if _, ok := b.Update(tea.KeyPressMsg{Code: 'x', Text: "x"}); ok {
		t.Error("x was taken")
	}
	// a tab without a glyph has its name under its number
	bare := Bar{Selected: 1, Width: 40, Tabs: []Tab{{"", "nonesuch", "!"}, {"🎫", "backlog", "3"}}}
	if got := plain(bare); !strings.HasPrefix(got, " ➀╶") || !strings.HasPrefix(row(bare, 1), "   nonesuch !      │") {
		t.Errorf("a tab without a glyph: %q", got)
	}
}
