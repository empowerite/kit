// The app's menu is the same on every screen: its own items first, kit's screens, the one it is on dim, the
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
	m := Menu{Here: Welcome, Own: []Item{{"🔄", "Refresh All"}}}.Opened()
	var names []string
	for _, item := range m.Items() {
		names = append(names, item.Name)
	}
	if got := strings.Join(names, "|"); got != "Welcome|Tutorials|Flow|Help||Refresh All|Settings…||Quit" {
		t.Fatalf("the items: %q", got)
	}
	// its own, another screen, the settings, and quit do something; the screen it is on, the tutorials, and the rules
	// do not
	for k, want := range []bool{false, false, true, true, false, true, true, false, true} {
		if m.enabled(k) != want {
			t.Errorf("item %d (%s) enabled %v", k, names[k], !want)
		}
	}
	if (Menu{}).Items()[5].Name != "Settings…" || m.At != 2 || !m.Open {
		t.Errorf("a menu with no items of its screen's, or the keyboard on its first that acts: %+v", m)
	}
}

func TestTheMenusKeysMoveChooseAndClose(t *testing.T) {
	m := Menu{Here: Flow}.Opened()
	for _, step := range []struct {
		key  string
		want int
	}{{"down", 3}, {"down", 5}, {"down", 7}, {"down", 7}, {"up", 5}, {"w", 0}, {"q", 7}, {"z", 7}} {
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
	own, _ := Menu{Here: Help, Own: []Item{{"", "Mine"}}}.Opened().Key(key("m"))
	if _, c := own.Key(key("enter")); c != (Choice{Own: "Mine"}) {
		t.Errorf("its own: %+v", c)
	}
	// enter on nothing chosen does nothing; esc and = close
	if m, c := (Menu{Here: Flow, Open: true, At: -1}).Key(key("enter")); !m.Open || c != (Choice{}) {
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
	m := Menu{Here: Flow}.Opened()
	screen := strings.Split(strings.Repeat(strings.Repeat("x", 60)+"\n", 20), "\n")
	drawn := m.Overlay(screen, 60)
	if !strings.HasPrefix(ansi.Strip(drawn[Top]), "╭╴"+App+"╶") || !strings.HasPrefix(ansi.Strip(drawn[1]), "│ "+Home+" Welcome") ||
		ansi.StringWidth(drawn[3]) != 60 {
		t.Errorf("the menu over the screen:\n%s", ansi.Strip(strings.Join(drawn, "\n")))
	}
	if got := (Menu{}).Overlay([]string{"a"}, 60); got[0] != "a" {
		t.Errorf("a closed menu drew: %q", got)
	}
	// a click on welcome chooses it; on flow, where it is, nothing, the menu open; off the box, it closes
	if m, c := m.Click(3, Top+1, 60); c != (Choice{Screen: Welcome}) || m.Open {
		t.Errorf("a click on welcome: %v %+v", m.Open, c)
	}
	if m, c := m.Click(3, Top+3, 60); c != (Choice{}) || !m.Open {
		t.Errorf("a click on flow: %v %+v", m.Open, c)
	}
	if m, c := m.Click(50, 15, 60); c != (Choice{}) || m.Open {
		t.Errorf("a click off the menu: %v %+v", m.Open, c)
	}
	if !OnApp(At, Top) || !OnApp(0, Top) || OnApp(At, Top+1) || OnApp(5, Top) {
		t.Error("OnApp")
	}
	if Title("kit", false) != "  "+style.Tint(App)+" kit" || Title("kit", true) != "  "+style.Menu(App)+" kit" {
		t.Errorf("the title: %q %q", Title("kit", false), Title("kit", true))
	}
}

// A box without its glyph keeps to room's right; one whose glyph is near the right turns its corner there; the rows
// past the screen's last are not drawn.
func TestABoxKeepsToTheRoomItHas(t *testing.T) {
	items := []Item{{"", "One"}}
	all := func(int) bool { return true }
	if box, left := Box(items, all, 0, "⋮", -1, 40); left+ansi.StringWidth(box[0]) != 40 {
		t.Errorf("no glyph: left %d, width %d", left, ansi.StringWidth(box[0]))
	}
	if box, left := Box(items, all, 0, "⋮", 38, 40); left+ansi.StringWidth(box[0]) != 40 ||
		!strings.HasSuffix(ansi.Strip(box[0]), "╴⋮╮") {
		t.Errorf("a glyph near the right: left %d, %q", left, ansi.Strip(box[0]))
	}
	if got := Over([]string{"a"}, []string{"x", "y"}, 0, 0); len(got) != 1 {
		t.Errorf("a box past the last line: %q", got)
	}
}

// The search types what is pressed, erases with backspace, and leaves on esc, tab, or enter, what is typed kept; it
// stands right-aligned while idle, and in use over the tab bar's rightmost number that leaves room to type.
func TestTheSearchIsTheSameOnEveryScreen(t *testing.T) {
	s := Search{Focused: true}
	for _, k := range []string{"c", "o", "=", "/"} {
		s, _ = s.Key(key(k))
	}
	if s, ok := s.Key(tea.KeyPressMsg{Code: tea.KeyBackspace}); !ok || s.Query != "co=" {
		t.Errorf("typing and backspace: %q", s.Query)
	}
	if _, ok := s.Key(key("up")); ok {
		t.Error("up was the search's")
	}
	for _, k := range []tea.KeyPressMsg{key("esc"), {Code: tea.KeyTab}, key("enter")} {
		if left, ok := s.Key(k); !ok || left.Focused || left.Query != s.Query {
			t.Errorf("%s did not leave the search, what is typed kept: %+v", k, left)
		}
	}
	if q, _ := Type("", tea.KeyPressMsg{Code: tea.KeyBackspace}); q != "" {
		t.Error("backspace on nothing")
	}
	if SearchLine("co", true) != Find+" co▏" || SearchLine("co", false) != Find+" co" ||
		SearchLine("", false) != Find+" "+style.Off("search") {
		t.Error("the search's line")
	}
	// right-aligned a cell short of the edge, idle or with the keyboard, until what is typed needs more room; then it
	// grows leftward, as far as the screen's first cell
	idle := 80 - 1 - ansi.StringWidth(Find+" search")
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
