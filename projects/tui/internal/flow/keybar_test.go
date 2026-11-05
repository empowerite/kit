// Every key the keys bar names does something where the keyboard is: the bar is made from the path to the keyboard,
// and a key it names that changes nothing on the screen is a key it should not have named.
package flow

import (
	"fmt"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/empowerite/kit/projects/tui/internal/testkit"
)

func TestEveryKeyTheBarNamesDoesSomething(t *testing.T) {
	for _, c := range []struct {
		where string
		msgs  []tea.Msg
	}{
		{"on the bar", nil},
		{"on the bar, the sidebar shown", []tea.Msg{press('?')}},
		{"on a row that opens", []tea.Msg{right, right, right, down}},
		{"on a row open", []tea.Msg{right, right, right, down, right}},
		{"in the sidebar's search", []tea.Msg{press(tea.KeyTab)}},
		{"in the sidebar's search, typed", append([]tea.Msg{press(tea.KeyTab)}, typed("o")...)},
		{"on the sidebar's first column", []tea.Msg{press(tea.KeyTab), down}},
		{"on the sidebar's second column", []tea.Msg{press(tea.KeyTab), down, down}},
		{"in the search, the sidebar hidden", typed("/")},
		{"in the rows, then the sidebar", []tea.Msg{down, press(tea.KeyTab), down}},
	} {
		t.Run(c.where, func(t *testing.T) {
			t.Parallel()
			m := loaded(t, options(), c.msgs...)
			before := view(m)
			for k, segment := range m.keys().Shown() {
				for _, b := range segment {
					if b.Key == "" {
						continue // what is typed, which the search's own tests type
					}
					acts := false
					for _, p := range b.Presses {
						// a model of its own for each press, as a copy shares its stages with the one it is copied from
						fresh := loaded(t, options(), c.msgs...)
						if view(drive(fresh, testkit.Press(p))) != before {
							acts = true
							break
						}
					}
					if !acts {
						t.Errorf("control %d of %q names %s, which changes nothing:\n%s", k,
							m.keys().String(), fmt.Sprintf("%q %q", b.Key, b.Does), before)
					}
				}
			}
		})
	}
}
