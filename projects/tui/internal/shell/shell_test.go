// The app's menu is the same on every screen: kit's screens, the guides then the apps, the one it is on dim, the
// settings, and quit; its keys and clicks choose, and the box hangs from its glyph, whatever is under it kept in its
// columns.
package shell

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/empowerite/kit/projects/tui/internal/style"
)

// key gives the press of a key, with its text when it types one.
func key(s string) tea.KeyPressMsg {
	switch s {
	case "up":
		return tea.KeyPressMsg{Code: tea.KeyUp}
	case "down":
		return tea.KeyPressMsg{Code: tea.KeyDown}
	case "enter":
		return tea.KeyPressMsg{Code: tea.KeyEnter}
	case "esc":
		return tea.KeyPressMsg{Code: tea.KeyEscape}
	case "space":
		return tea.KeyPressMsg{Code: tea.KeySpace, Text: " "}
	}
	return tea.KeyPressMsg{Code: rune(s[0]), Text: s}
}

func TestTheMenuHasItsScreensOwnItemsKitsScreensTheSettingsAndQuit(t *testing.T) {
	m := Menu{Here: Welcome, Own: []Item{{Glyph: "🔄", Name: "Refresh All"}}}.Opened()
	var names []string
	for _, item := range m.Items() {
		names = append(names, item.Name)
	}
	if got := strings.Join(names, "|"); got != "Welcome|Tutorials|Help||Issues|Changes|Repositories|Delivery|People|Trust|Platform||Refresh All|Settings…||Quit" {
		t.Fatalf("the items: %q", got)
	}
	// its own, another screen, the settings, and quit do something; the screen it is on and the rules do not
	for k, want := range []bool{false, true, true, false, true, true, true, true, true, true, true, false, true, true, false, true} {
		if m.enabled(k) != want {
			t.Errorf("item %d (%s) enabled %v", k, names[k], !want)
		}
	}
	if (Menu{}).Items()[12].Name != "Settings…" || m.At != 1 || !m.Open {
		t.Errorf("a menu with no items of its screen's, or the keyboard on its first that acts: %+v", m)
	}
}

func TestTheMenusKeysMoveChooseAndClose(t *testing.T) {
	m := Menu{Here: Issues}.Opened()
	for _, step := range []struct {
		key  string
		want int
	}{{"down", 1}, {"down", 2}, {"down", 5}, {"down", 6}, {"down", 7}, {"down", 8}, {"down", 9}, {"down", 10},
		{"down", 12}, {"down", 14}, {"down", 14}, {"up", 12}, {"w", 0}, {"q", 14}, {"z", 14}} {
		if m, _ = m.Key(key(step.key)); m.At != step.want {
			t.Errorf("%s: item %d, want %d", step.key, m.At, step.want)
		}
	}
	if m, c := m.Key(key("enter")); m.Open || c != (Choice{Quit: true}) {
		t.Errorf("quit: open %v, %+v", m.Open, c)
	}
	if m, c := m.Opened().Key(key("h")); !m.Open || c != (Choice{}) {
		t.Errorf("h: open %v, %+v", m.Open, c)
	} else if m, c = m.Key(key("enter")); c != (Choice{Screen: Help}) || m.Open {
		t.Errorf("help: open %v, %+v", m.Open, c)
	}
	if m, c := m.Opened().Key(key("s")); !m.Open || c != (Choice{}) {
		t.Errorf("s: open %v, %+v", m.Open, c)
	} else if _, c = m.Key(key("enter")); c != (Choice{Screen: SettingsScreen}) {
		t.Errorf("settings: %+v", c)
	}
	own, _ := Menu{Here: Help, Own: []Item{{Name: "Mine"}}}.Opened().Key(key("m"))
	if _, c := own.Key(key("enter")); c != (Choice{Own: "Mine"}) {
		t.Errorf("its own: %+v", c)
	}
	// enter on nothing chosen does nothing; esc and = close
	if m, c := (Menu{Here: Issues, Dropdown: Dropdown{Open: true, At: -1}}).Key(key("enter")); !m.Open || c != (Choice{}) {
		t.Errorf("enter on nothing: %v %+v", m.Open, c)
	}
	for _, k := range []string{"esc", "="} {
		if m, _ := m.Opened().Key(key(k)); m.Open {
			t.Errorf("%s did not close the menu", k)
		}
	}
	if Switch(Welcome)() != (SwitchMsg{Welcome}) {
		t.Error("Switch")
	}
}

func TestTheMenuHangsFromTheTitlesGlyphAndAClickChooses(t *testing.T) {
	m := Menu{Here: Issues}.Opened()
	screen := strings.Split(strings.Repeat(strings.Repeat("x", 60)+"\n", 20), "\n")
	drawn := m.Overlay(screen, 60)
	if !strings.HasPrefix(ansi.Strip(drawn[Top]), "╭╴"+App+"╶") || !strings.HasPrefix(ansi.Strip(drawn[1]), "│ "+Home+" Welcome") ||
		ansi.StringWidth(drawn[3]) != 60 {
		t.Errorf("the menu over the screen:\n%s", ansi.Strip(strings.Join(drawn, "\n")))
	}
	if got := (Menu{}).Overlay([]string{"a"}, 60); got[0] != "a" {
		t.Errorf("a closed menu drew: %q", got)
	}
	// a click on welcome chooses it; on issues, where it is, nothing, the menu open; off the box, it closes
	if m, c := m.Click(3, Top+1, 60); c != (Choice{Screen: Welcome}) || m.Open {
		t.Errorf("a click on welcome: %v %+v", m.Open, c)
	}
	if m, c := m.Click(3, Top+5, 60); c != (Choice{}) || !m.Open {
		t.Errorf("a click on issues: %v %+v", m.Open, c)
	}
	if m, c := m.Click(50, 15, 60); c != (Choice{}) || m.Open {
		t.Errorf("a click off the menu: %v %+v", m.Open, c)
	}
	// = and the name after it, "kit", take the click, on the title line alone
	if !OnApp(At, Top, "kit") || !OnApp(0, Top, "kit") || !OnApp(At+4, Top, "kit") || OnApp(At+5, Top, "kit") ||
		OnApp(At, Top+1, "kit") {
		t.Error("OnApp")
	}
	if Title("kit", false) != "  "+style.Tint(App)+" kit" || Title("kit", true) != "  "+style.Menu(App)+" kit" {
		t.Errorf("the title: %q %q", Title("kit", false), Title("kit", true))
	}
}

// A box without its glyph keeps to room's right; one whose glyph is near the right turns its corner there; the rows
// past the screen's last are not drawn.
func TestABoxKeepsToTheRoomItHas(t *testing.T) {
	items := []Item{{Name: "One"}}
	all := func(int) bool { return true }
	if box, left := Box(items, all, 0, ":", -1, 40); left+ansi.StringWidth(box[0]) != 40 {
		t.Errorf("no glyph: left %d, width %d", left, ansi.StringWidth(box[0]))
	}
	if box, left := Box(items, all, 0, ":", 38, 40); left+ansi.StringWidth(box[0]) != 40 ||
		!strings.HasSuffix(ansi.Strip(box[0]), "╴:╮") {
		t.Errorf("a glyph near the right: left %d, %q", left, ansi.Strip(box[0]))
	}
	if got := Over([]string{"a"}, []string{"x", "y"}, 0, 0); len(got) != 1 {
		t.Errorf("a box past the last line: %q", got)
	}
}

// The search types what is pressed, erases with backspace, clears with ⌃U, recalls with ⌃R, and gives the view focus
// on enter, down, esc, or tab, what is typed kept; it stands right-aligned while idle, and in use over the tab bar's
// rightmost number that leaves room to type.
func TestTheSearchIsTheSameOnEveryScreen(t *testing.T) {
	s := Search{Focused: true}
	for _, k := range []string{"c", "o", "=", "/"} {
		s, _ = s.Key(key(k))
	}
	if s, did := s.Key(tea.KeyPressMsg{Code: tea.KeyBackspace}); did != Typed || s.Query != "co=" {
		t.Errorf("typing and backspace: %q", s.Query)
	}
	if _, did := s.Key(key("up")); did != NotSearch {
		t.Error("up was the search's")
	}
	for _, k := range []tea.KeyPressMsg{key("esc"), {Code: tea.KeyTab}, key("enter"), key("down")} {
		if left, did := s.Key(k); did != Left || left.Focused || left.Query != s.Query {
			t.Errorf("%s did not give the view focus, what is typed kept: %+v", k, left)
		}
	}
	if cleared, did := s.Key(tea.KeyPressMsg{Code: 'u', Mod: tea.ModCtrl}); did != Typed || cleared.Query != "" ||
		!cleared.Focused {
		t.Errorf("⌃U: %+v", cleared)
	}
	if recalled, did := s.Key(tea.KeyPressMsg{Code: 'r', Mod: tea.ModCtrl}); did != Recalled || recalled.Query != s.Query {
		t.Errorf("⌃R: %+v", recalled)
	}
	if q, _ := Type("", tea.KeyPressMsg{Code: tea.KeyBackspace}); q != "" {
		t.Error("backspace on nothing")
	}
	if SearchLine("co", true) != Find+" co▏" || SearchLine("co", false) != Find+" co" ||
		SearchLine("", false) != Find+" "+style.Off(Idle) {
		t.Error("the search's line")
	}
	// right-aligned a cell short of the edge, idle or with the keyboard, until what is typed needs more room; then it
	// grows leftward, as far as the screen's first cell
	idle := 80 - 1 - ansi.StringWidth(Find+" "+Idle)
	for _, line := range []string{SearchLine("", false), SearchLine("", true), SearchLine("co", true)} {
		if at := SearchAt(80, line); at != idle {
			t.Errorf("%q at %d", line, at)
		}
	}
	if at := SearchAt(80, SearchLine("a longer query", true)); at != 80-1-ansi.StringWidth(Find+" a longer query▏") {
		t.Errorf("a long query at %d", at)
	}
	if SearchAt(5, SearchLine("", false)) != 0 {
		t.Error("too narrow a screen")
	}
}

// A box and a title in a group's color say what the menu's own do, the edges and the line under the keyboard drawn in
// that color, as style.MenuIn draws them; no color is the menu's own.
func TestABoxInAGroupsColorSaysWhatTheMenusOwnDoes(t *testing.T) {
	items := []Item{{Glyph: "💾", Name: "Save"}, {}, {Name: "Default"}}
	enabled := func(k int) bool { return items[k].Name != "" }
	own, left := Box(items, enabled, 2, "", -1, 40)
	blue, at := BoxColored(items, enabled, 2, "", -1, 40, "75")
	if plain, _ := BoxColored(items, enabled, 2, "", -1, 40, ""); strings.Join(plain, "\n") != strings.Join(own, "\n") {
		t.Errorf("no color is not the menu's own:\n%s", strings.Join(plain, "\n"))
	}
	edge, under := style.MenuIn("75")
	if at != left || len(blue) != len(own) || blue[0] != edge(ansi.Strip(blue[0])) ||
		!strings.Contains(blue[3], "48;5;75") || blue[3] == own[3] || under("x") == style.Selected("x") || strings.Contains(strings.Join(blue, ""), style.Menu("│")) {
		t.Errorf("the box in 75, at %d:\n%q", at, blue)
	}
	for k := range own {
		if ansi.Strip(blue[k]) != ansi.Strip(own[k]) {
			t.Errorf("line %d says %q, not %q", k, ansi.Strip(blue[k]), ansi.Strip(own[k]))
		}
	}
	if got := TitledColored("Presets", 20, "75"); ansi.Strip(got) != ansi.Strip(Titled("Presets", 20)) ||
		got != edge(ansi.Strip(got)) || TitledColored("Presets", 20, "") != Titled("Presets", 20) {
		t.Errorf("the title in 75: %q", got)
	}
}

// A menu shows each line's keys at its right, flush with the widest, dim but under the keyboard, where the line is
// drawn as one; a menu with no keys has no room for them; and quit's are ⌃Q, on every screen.
func TestAMenuShowsEachLinesKeysAtItsRight(t *testing.T) {
	items := []Item{{Glyph: "✅", Name: "Select All", Keys: "⌃A"}, {Name: "Find", Keys: "⇧⌃F"}, {Name: "Refresh"}, {},
		{Name: "Off", Keys: "⌃O"}}
	enabled := func(k int) bool { return k < 3 }
	box, _ := Box(items, enabled, 1, "", -1, 40)
	for k, want := range map[int]string{1: "│ ✅ Select All   ⌃A │", 2: "│    Find        ⇧⌃F │",
		3: "│    Refresh         │", 5: "│    Off          ⌃O │"} {
		if got := ansi.Strip(box[k]); got != want {
			t.Errorf("line %d: %q, want %q", k, got, want)
		}
	}
	_, under := style.MenuIn("")
	if !strings.Contains(box[1], style.Off("   ⌃A")) || !strings.Contains(box[2], under("    Find        ⇧⌃F ")) ||
		!strings.Contains(box[5], style.Off("    Off          ⌃O ")) {
		t.Errorf("the keys' styles:\n%q", box)
	}
	if plain, _ := Box([]Item{{Name: "Refresh"}}, Every, 0, "", -1, 40); ansi.Strip(plain[1]) != "│    Refresh │" {
		t.Errorf("no keys: %q", ansi.Strip(plain[1]))
	}
	if quit := (Menu{}).Items(); quit[len(quit)-1].Keys != "⌃Q" {
		t.Errorf("quit: %+v", quit[len(quit)-1])
	}
}

// A dialog is a box with its title on the top edge, a plain bottom edge, its keys being the keys bar's alone, and its
// lines inside, every line as wide as the others; a line too long for the room is clipped.
func TestADialogBoxesItsLinesUnderItsTitle(t *testing.T) {
	lines := Dialog("Close these?", []string{"#45 one", "#46 two"}, 60)
	plain := make([]string, len(lines))
	for k, l := range lines {
		plain[k] = ansi.Strip(l)
	}
	if len(lines) != 4 || !strings.HasPrefix(plain[0], "╭─ Close these? ") || !strings.HasSuffix(plain[0], "╮") ||
		!strings.HasPrefix(plain[1], "│ #45 one ") || strings.Trim(plain[3], "╰─╯") != "" {
		t.Errorf("the dialog:\n%s", strings.Join(plain, "\n"))
	}
	for _, l := range plain {
		if ansi.StringWidth(l) != ansi.StringWidth(plain[0]) {
			t.Errorf("a line of another width: %q", l)
		}
	}
	narrow := Dialog("A title far too long", []string{"a line far too long for the room"}, 16)
	for _, l := range narrow {
		if w := ansi.StringWidth(l); w != 16 {
			t.Errorf("narrow: a line of %d cells: %q", w, ansi.Strip(l))
		}
	}
}

// The menu bar is every screen's: the kit menu and the screen's name, each part given after a separator, and the
// search field at the right end; the left part is clipped two cells short of it, and what the view stands on is found
// where it is drawn, whole or cut.
func TestTheMenuBarIsEveryScreens(t *testing.T) {
	b := MenuBar{Screen: "kit flow", Workspace: "- o/r", On: "➊ backlog 3 :", Search: SearchLine("", false)}
	line := ansi.Strip(b.Line(60))
	if !strings.HasPrefix(line, "  "+App+" kit flow · - o/r · ➊ backlog 3 :") || !strings.HasSuffix(line, Find+" "+Idle) ||
		ansi.StringWidth(line) != 59 {
		t.Errorf("the menu bar at 60: %q", line)
	}
	from, until, whole := b.OnSpan(60)
	if !whole || ansi.Strip(ansi.Cut(b.Line(60), from, until)) != b.On {
		t.Errorf("what the view stands on spans %d to %d, whole %v", from, until, whole)
	}
	if _, _, whole = b.OnSpan(35); whole {
		t.Error("at 35 what the view stands on is cut, not whole")
	}
	if from, _, _ = b.OnSpan(15); from != -1 {
		t.Errorf("at 15 what the view stands on does not show, yet starts at %d", from)
	}
	if bare := (MenuBar{Screen: "kit flow"}); bare.SearchAt(60) != 60 || ansi.Strip(bare.Line(60)) != "  "+App+" kit flow" {
		t.Errorf("with no part and no search: %q", ansi.Strip(bare.Line(60)))
	}
	if at := b.WorkspaceAt(); ansi.Strip(ansi.Cut(b.Line(60), at, at+1)) != Workspaces {
		t.Errorf("the workspace menu is not at %d", at)
	}
}
