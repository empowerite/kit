// Layers: the topmost open layer takes the keys and the clicks, the keys bar names its keys alone, and the open
// layers are drawn bottom up.
package shell

import (
	"slices"
	"testing"

	tea "charm.land/bubbletea/v2"
)

// screen is a screen with two layers, a menu over a dialog that takes no click, and what reached each.
type screen struct {
	menu, dialog bool
	got          []string
}

// note gives the screen with what reached it noted.
func (s screen) note(what string) screen {
	s.got = append(slices.Clone(s.got), what)
	return s
}

// layered gives the screen's layers: the menu, topmost, then the dialog.
func layered() Layers[screen] {
	return Layers[screen]{
		{
			Open:  func(s screen) bool { return s.menu },
			Key:   func(s screen, k tea.KeyPressMsg) (screen, tea.Cmd) { return s.note("menu " + k.String()), nil },
			Click: func(s screen, x, y int) (screen, tea.Cmd) { return s.note("menu click"), nil },
			Keys:  func(screen) Segment { return MenuKeys() },
			Draw:  func(s screen, lines []string) []string { return append(lines, "menu") },
		},
		{
			Open: func(s screen) bool { return s.dialog },
			Key:  func(s screen, k tea.KeyPressMsg) (screen, tea.Cmd) { return s.note("dialog " + k.String()), nil },
			Keys: func(screen) Segment { return DialogKeys("esc close") },
			Draw: func(s screen, lines []string) []string { return append(lines, "dialog") },
		},
	}
}

func TestNoLayerOpenLeavesTheScreenItsKeysClicksAndBar(t *testing.T) {
	ls := layered()
	if _, _, ok := ls.Key(screen{}, key("x")); ok {
		t.Error("a key went to a layer while none was open")
	}
	if _, _, ok := ls.Click(screen{}, 1, 1); ok {
		t.Error("a click went to a layer while none was open")
	}
	if _, ok := ls.Keys(screen{}); ok {
		t.Error("the keys bar named a layer's keys while none was open")
	}
	if got := ls.Draw(screen{}, []string{"rows"}); !slices.Equal(got, []string{"rows"}) {
		t.Errorf("drawn: %q", got)
	}
}

func TestTheTopmostOpenLayerTakesTheKeysAndTheClicks(t *testing.T) {
	ls := layered()
	both := screen{menu: true, dialog: true}
	if s, _, ok := ls.Key(both, key("x")); !ok || !slices.Equal(s.got, []string{"menu x"}) {
		t.Errorf("a key with both open: %v %q", ok, s.got)
	}
	if s, _, ok := ls.Click(both, 1, 1); !ok || !slices.Equal(s.got, []string{"menu click"}) {
		t.Errorf("a click with both open: %v %q", ok, s.got)
	}
	if path, ok := ls.Keys(both); !ok || path.String() != (Path{MenuKeys()}).String() {
		t.Errorf("the keys bar with both open: %q", path.String())
	}
	// the dialog alone: it takes the keys, and a click is taken, doing nothing, as its Click is nil
	dialog := screen{dialog: true}
	if s, _, ok := ls.Key(dialog, key("esc")); !ok || !slices.Equal(s.got, []string{"dialog esc"}) {
		t.Errorf("a key to the dialog: %v %q", ok, s.got)
	}
	if s, _, ok := ls.Click(dialog, 1, 1); !ok || len(s.got) != 0 {
		t.Errorf("a click with the dialog open: %v %q", ok, s.got)
	}
	if path, _ := ls.Keys(dialog); path.String() != (Path{DialogKeys("esc close")}).String() {
		t.Errorf("the keys bar with the dialog open: %q", path.String())
	}
}

func TestTheOpenLayersAreDrawnBottomUp(t *testing.T) {
	got := layered().Draw(screen{menu: true, dialog: true}, []string{"rows"})
	if want := []string{"rows", "dialog", "menu"}; !slices.Equal(got, want) {
		t.Errorf("drawn: %q, want %q", got, want)
	}
}
