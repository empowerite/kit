// A dropdown keeps whether it is open and the line under the keyboard; the lines, and which act, are its owner's.
package shell

import (
	"testing"

	tea "charm.land/bubbletea/v2"
)

// lines is a dropdown's lines: Alpha, a rule, Beta, which does not act, Bravo, and Charlie.
var lines = []Item{
	{Glyph: "🅰", Name: "Alpha"}, {}, {Glyph: "🅱", Name: "Beta"}, {Glyph: "🅱", Name: "Bravo"}, {Glyph: "©", Name: "Charlie"},
}

// acts says which of lines act: all but the rule and Beta.
func acts(k int) bool { return k != 1 && k != 2 }

func TestADropdownOpensOnItsFirstLineThatActsAndPassesTheOthersBy(t *testing.T) {
	d := Dropdown{}.Opened(lines, func(k int) bool { return k >= 3 })
	if !d.Open || d.At != 3 {
		t.Fatalf("opened: %+v", d)
	}
	d = Dropdown{}.Opened(lines, acts)
	steps := []struct {
		key  string
		want int
	}{{"down", 3}, {"down", 4}, {"down", 4}, {"up", 3}, {"up", 0}, {"up", 0}, {"b", 3}, {"c", 4}, {"z", 4}}
	for _, s := range steps {
		var chosen int
		if d, chosen = d.Key(key(s.key), lines, acts); d.At != s.want || chosen != -1 || !d.Open {
			t.Errorf("%s: at %d, chosen %d, open %v; want at %d", s.key, d.At, chosen, d.Open, s.want)
		}
	}
}

func TestADropdownChoosesTheLineUnderTheKeyboardAndClosesOnEscOrItsOwnersKey(t *testing.T) {
	for _, k := range []string{"enter", "space"} {
		d := Dropdown{Open: true, At: 3}
		if d, chosen := d.Key(key(k), lines, acts); chosen != 3 || d.Open {
			t.Errorf("%s: chosen %d, open %v", k, chosen, d.Open)
		}
	}
	// the keyboard on a line that does not act chooses nothing, and the dropdown stays open
	if d, chosen := (Dropdown{Open: true, At: 2}).Key(key("enter"), lines, acts); chosen != -1 || !d.Open {
		t.Errorf("enter on Beta: chosen %d, open %v", chosen, d.Open)
	}
	if d, chosen := (Dropdown{Open: true, At: -1}).Key(key("enter"), lines, acts); chosen != -1 || !d.Open {
		t.Errorf("enter with no line under the keyboard: chosen %d, open %v", chosen, d.Open)
	}
	for _, k := range []string{"esc", ":"} {
		if d, chosen := (Dropdown{Open: true, At: 3}).Key(key(k), lines, acts, ":"); chosen != -1 || d.Open {
			t.Errorf("%s: chosen %d, open %v", k, chosen, d.Open)
		}
	}
	// a key its owner does not name is no closer
	if d, _ := (Dropdown{Open: true, At: 3}).Key(key(":"), lines, acts); !d.Open {
		t.Error(": closed a dropdown whose owner names no closer")
	}
}

// The box is drawn from line 5, its left edge in column 10, 12 cells wide, so its lines are lines 6 to 10, between
// columns 11 and 20.
func TestADropdownTakesAClickOnALineOrOffItsBox(t *testing.T) {
	for _, c := range []struct {
		name   string
		x, y   int
		chosen int
		open   bool
	}{
		{"on Bravo", 12, 9, 3, false},
		{"on Charlie's last cell", 20, 10, 4, false},
		{"on the rule", 12, 7, -1, true},
		{"on Beta, which does not act", 12, 8, -1, true},
		{"on the left edge", 10, 9, -1, false},
		{"on the right edge", 21, 9, -1, false},
		{"on the top edge", 12, 5, -1, false},
		{"under the box", 12, 11, -1, false},
	} {
		d, chosen := Dropdown{Open: true, At: 0}.Click(c.x, c.y, 5, 10, 12, lines, acts)
		if chosen != c.chosen || d.Open != c.open {
			t.Errorf("%s: chosen %d, open %v; want %d, %v", c.name, chosen, d.Open, c.chosen, c.open)
		}
	}
	// scrolled to show two lines from Bravo on, the box's first line is Bravo, and a line past the two is off it
	if d, chosen := (Dropdown{Open: true}).ClickIn(12, 6, 5, 10, 12, 3, 2, lines, acts); chosen != 3 || d.Open {
		t.Errorf("a click on a scrolled box's first line: chosen %d, open %v", chosen, d.Open)
	}
	if d, chosen := (Dropdown{Open: true}).ClickIn(12, 8, 5, 10, 12, 3, 2, lines, acts); chosen != -1 || d.Open {
		t.Errorf("a click past a scrolled box's lines: chosen %d, open %v", chosen, d.Open)
	}
}

// A line's keys stand for a press, as Bubble Tea names it; with the dropdown open, the press chooses the line, when it
// acts, before a letter moves to the next line that starts with it.
func TestALinesKeysChooseItWithTheDropdownOpen(t *testing.T) {
	for keys, want := range map[string]string{"⌃A": "ctrl+a", "⌃Q": "ctrl+q", "r": "r", "M": "m", "⇧⌃F": "", "": ""} {
		if got := Press(keys); got != want {
			t.Errorf("Press(%q): %q, want %q", keys, got, want)
		}
	}
	items := []Item{{Name: "Refresh", Keys: "R"}, {Name: "Rename"}, {Name: "Off", Keys: "O"}, {Name: "Select All", Keys: "⌃A"}}
	acting := func(k int) bool { return k != 2 }
	d := Dropdown{}.Opened(items, acting)
	if _, chosen := d.Key(key("r"), items, acting); chosen != 0 {
		t.Errorf("r chose %d, not Refresh", chosen)
	}
	if d, chosen := d.Key(key("o"), items, acting); chosen != -1 || !d.Open {
		t.Errorf("o, the key of a line that does not act, chose %d", chosen)
	}
	if _, chosen := d.Key(tea.KeyPressMsg{Code: 'a', Mod: tea.ModCtrl}, items, acting); chosen != 3 {
		t.Errorf("ctrl+a chose %d, not Select All", chosen)
	}
	if d, chosen := d.Key(key("s"), items, acting); chosen != -1 || d.At != 3 {
		t.Errorf("s, no line's key, chose %d and moved to %d, not to Select All", chosen, d.At)
	}
}

// A key that types nothing, as tab, leaves the keyboard on the line it was on.
func TestADropdownKeyThatTypesNothingStays(t *testing.T) {
	tab := tea.KeyPressMsg{Code: tea.KeyTab}
	if d, chosen := (Dropdown{Open: true, At: 3}).Key(tab, lines, acts); d.At != 3 || chosen != -1 || !d.Open {
		t.Errorf("tab: at %d, chosen %d, open %v", d.At, chosen, d.Open)
	}
}
