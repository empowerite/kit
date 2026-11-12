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
	// the presets dropped down from the preset line, up past the view the line is set to, Save As…, and the dim Save,
	// to Fields…, chosen: the columns dialog; from the bar, down to the preset line first
	fromLine := []tea.Msg{press(tea.KeyEnter), up, up, press(tea.KeyEnter)}
	// down reaches source's name in the heading over the groups; tab four times, past organization's, repository's,
	// and the rows', the preset picker
	line := []tea.Msg{down, tab, tab, tab, tab}
	columns := append(slices.Clone(line), fromLine...)
	for _, c := range []struct {
		where   string
		msgs    []tea.Msg
		presets bool // whether the tables have their preset line
	}{
		{"on the bar", nil, false},
		{"on the bar, the legend shown", []tea.Msg{press('?')}, false},
		{"on a row that opens", []tea.Msg{tab, tab, down}, false},
		{"on a row open", []tea.Msg{tab, tab, down, right}, false},
		{"in the search", typed("/"), false},
		{"in the search, typed", typed("/o"), false},
		{"in the rows, then the search", append([]tea.Msg{down}, typed("/")...), false},
		{"in the columns dialog", columns, true},
		{"on a group in the columns dialog", append(slices.Clone(columns), shiftUp), true},
		{"on the last field in the columns dialog", append(slices.Clone(columns),
			slices.Repeat([]tea.Msg{down}, 9)...), true},
		{"on a hidden field in the columns dialog", append(slices.Clone(columns), down, down, press(tea.KeySpace)),
			true},
		{"in a field's popover", append(slices.Clone(columns), down, down, press(tea.KeyEnter)), true},
		{"in a field's popover, on a value", append(slices.Clone(columns), down, down, press(tea.KeyEnter), down,
			down, down, down), true},
		{"in a field's popover, on a rendering not taken", append(append(slices.Clone(columns), down, down,
			press(tea.KeyEnter)), slices.Repeat([]tea.Msg{down}, 20)...), true},
		{"in a field's popover, on auto, a rendering pinned", append(append(append(slices.Clone(columns), down, down,
			press(tea.KeyEnter)), slices.Repeat([]tea.Msg{down}, 20)...), press(tea.KeySpace), up, up), true},
		{"on the bar, over a preset line", nil, true},
		{"on the heading over the groups, on a name", []tea.Msg{down}, true},
		{"on the heading over the groups, on the rows'", []tea.Msg{down, tab, tab, tab}, true},
		{"on the preset line", line, true},
		{"on the preset line, the table changed", append(append(slices.Clone(line), fromLine...), down,
			press(tea.KeySpace), esc), true},
		{"naming for Save As… on the preset line", append(append(slices.Clone(line), press(tea.KeyEnter), up, press(tea.KeyEnter)),
			typed("Mine")...), true},
		{"with the presets dropped down", append(slices.Clone(line), press(tea.KeyEnter)), true},
		{"in the rows, under a preset line", []tea.Msg{down, down}, true},
		{"in the search, from the preset line", append(slices.Clone(line), typed("/")...), true},
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
