// Every key the keys bar names does something where the keyboard is: the bar is made from the path to the keyboard,
// and a key it names that changes nothing on the screen, its styles too, as the line under a menu's keyboard, is a key
// it should not have named.
package flow

import (
	"fmt"
	"slices"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/empowerite/kit/projects/tui/internal/testkit"
)

func TestEveryKeyTheBarNamesDoesSomething(t *testing.T) {
	shiftUp := testkit.Press("shift+up")
	for _, c := range []struct {
		where   string
		msgs    []tea.Msg
		presets bool // whether the tables have their preset line
	}{
		{"on the bar", nil, false},
		{"on the bar, the sidebar shown", []tea.Msg{press('?')}, false},
		{"on a row that opens", []tea.Msg{right, right, right, down}, false},
		{"on a row open", []tea.Msg{right, right, right, down, right}, false},
		{"in the sidebar's search", []tea.Msg{press(tea.KeyTab)}, false},
		{"in the sidebar's search, typed", append([]tea.Msg{press(tea.KeyTab)}, typed("o")...), false},
		{"on the sidebar's first column", []tea.Msg{press(tea.KeyTab), down}, false},
		{"on the sidebar's second column", []tea.Msg{press(tea.KeyTab), down, down}, false},
		{"on a group in the sidebar's editor", []tea.Msg{press(tea.KeyTab), down, down, shiftUp, shiftUp}, false},
		{"on the last column in the sidebar's editor", append([]tea.Msg{press(tea.KeyTab)},
			slices.Repeat([]tea.Msg{down}, 6)...), false},
		{"on a hidden field in the sidebar's editor", []tea.Msg{press(tea.KeyTab), down, down, down,
			press(tea.KeySpace)}, false},
		{"on the sidebar's box for folding", append([]tea.Msg{press(tea.KeyTab)}, slices.Repeat([]tea.Msg{down}, 7)...),
			false},
		{"in a field's popover, on a sort taken", []tea.Msg{press(tea.KeyTab), down, down, down, press(tea.KeyEnter)},
			false},
		{"in a field's popover, on a value", []tea.Msg{press(tea.KeyTab), down, down, down, press(tea.KeyEnter), down,
			down, down, down}, false},
		{"in a field's popover, on a rendering not taken", append([]tea.Msg{press(tea.KeyTab), down, down, down,
			press(tea.KeyEnter)}, slices.Repeat([]tea.Msg{down}, 20)...), false},
		{"in a field's popover, on auto, a rendering pinned", append(append([]tea.Msg{press(tea.KeyTab), down, down, down,
			press(tea.KeyEnter)}, slices.Repeat([]tea.Msg{down}, 20)...), press(tea.KeySpace), up, up), false},
		{"in the search, the sidebar hidden", typed("/"), false},
		{"in the rows, then the sidebar", []tea.Msg{down, press(tea.KeyTab), down}, false},
		{"on the bar, over a preset line", nil, true},
		{"on the preset line", []tea.Msg{down}, true},
		{"on the preset line, the table changed", []tea.Msg{down, press(tea.KeyTab), down, press(tea.KeySpace), esc},
			true},
		{"renaming on the preset line", append([]tea.Msg{down}, typed("Mine")...), true},
		{"with the presets dropped down", []tea.Msg{down, press(tea.KeyEnter)}, true},
		{"in the rows, under a preset line", []tea.Msg{down, down}, true},
		{"in the sidebar, from the preset line", []tea.Msg{down, press(tea.KeyTab)}, true},
	} {
		t.Run(c.where, func(t *testing.T) {
			t.Parallel()
			o := options()
			if c.presets {
				o = viewing("")
			}
			m := loaded(t, o, c.msgs...)
			before := m.View().Content
			for k, segment := range m.keys().Shown() {
				for _, b := range segment {
					if b.Key == "" {
						continue // what is typed, which the search's own tests type
					}
					acts := false
					for _, p := range b.Presses {
						// a model of its own for each press, as a copy shares its stages with the one it is copied from
						fresh := loaded(t, o, c.msgs...)
						if drive(fresh, testkit.Press(p)).View().Content != before {
							acts = true
							break
						}
					}
					if !acts {
						t.Errorf("control %d of %q names %s, which changes nothing:\n%s", k,
							m.keys().String(), fmt.Sprintf("%q %q", b.Key, b.Does), view(m))
					}
				}
			}
		})
	}
}
