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
	}
	return tea.KeyPressMsg{Code: rune(s[0]), Text: s}
}

func TestTheMenuHasItsScreensOwnItemsKitsScreensTheSettingsAndQuit(t *testing.T) {
	m := Menu{Here: Welcome, Own: []Item{{"🔄", "Refresh All"}}}.Opened()
	var names []string
	for _, item := range m.Items() {
		names = append(names, item.Name)
	}
	if got := strings.Join(names, "|"); got != "Refresh All||Flow|Welcome|Help|Tutorials||Settings…||Quit" {
		t.Fatalf("the items: %q", got)
	}
	// its own, another screen, and quit do something; the screen it is on, the tutorials, the settings, and the
	// rules do not
	for k, want := range []bool{true, false, true, false, true, false, false, false, false, true} {
		if m.enabled(k) != want {
			t.Errorf("item %d (%s) enabled %v", k, names[k], !want)
		}
	}
	if (Menu{}).Items()[0].Name != "Flow" || m.At != 0 || !m.Open {
		t.Errorf("a menu with no items of its screen's, or the keyboard: %+v", m)
	}
}

func TestTheMenusKeysMoveChooseAndClose(t *testing.T) {
	m := Menu{Here: Flow}.Opened()
	for _, step := range []struct {
		key  string
		want int
	}{{"down", 2}, {"down", 7}, {"down", 7}, {"up", 2}, {"w", 1}, {"q", 7}, {"z", 7}} {
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
	own := Menu{Here: Help, Own: []Item{{"", "Mine"}}}.Opened()
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
	if !strings.HasPrefix(ansi.Strip(drawn[Top]), "╭╴"+App+"╶") || !strings.HasPrefix(ansi.Strip(drawn[2]), "│ "+Home+" Welcome") ||
		ansi.StringWidth(drawn[3]) != 60 {
		t.Errorf("the menu over the screen:\n%s", ansi.Strip(strings.Join(drawn, "\n")))
	}
	if got := (Menu{}).Overlay([]string{"a"}, 60); got[0] != "a" {
		t.Errorf("a closed menu drew: %q", got)
	}
	// a click on welcome chooses it; on flow, where it is, nothing, the menu open; off the box, it closes
	if m, c := m.Click(3, Top+2, 60); c != (Choice{Screen: Welcome}) || m.Open {
		t.Errorf("a click on welcome: %v %+v", m.Open, c)
	}
	if m, c := m.Click(3, Top+1, 60); c != (Choice{}) || !m.Open {
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
		if left, ok := s.Key(k); !ok || left.Focused || left.Query != s.Query || !left.InUse() {
			t.Errorf("%s did not leave the search, what is typed kept: %+v", k, left)
		}
	}
	if q, _ := Type("", tea.KeyPressMsg{Code: tea.KeyBackspace}); q != "" || (Search{}).InUse() {
		t.Error("backspace on nothing, or an idle search in use")
	}
	if SearchLine("co", true) != Find+" co▏" || SearchLine("co", false) != Find+" co" ||
		SearchLine("", false) != Find+" "+style.Off("search") {
		t.Error("the search's line")
	}
	// idle, a cell short of the edge; in use, over the rightmost number with room, else as far right as has it
	if at := SearchAt(80, nil, false); at+ansi.StringWidth(Find+" search") != 79 {
		t.Errorf("idle at %d", at)
	}
	if at := SearchAt(80, []int{1, 20, 40, 70}, true); at != 40 {
		t.Errorf("in use at %d", at)
	}
	if at := SearchAt(80, nil, true); at != 80-1-Typing-3 || SearchAt(5, nil, true) != 0 {
		t.Errorf("in use with no number at %d", at)
	}
}
