// A table's layers, as a screen opens them over itself, in the order shell.Layers follows, the topmost first: a field's
// card, which moves its column too where it hangs from the column's name; the fields dialog, which the card opens over;
// and the presets' menu. A screen reaches each through its model, so the layers are the same on every screen that has a table.
package table

import (
	tea "charm.land/bubbletea/v2"

	"github.com/empowerite/kit/projects/tui/internal/shell"
)

// Overlays are the functions of a screen's model M that reach its table's layers: the host of the table selected;
// the card, the fields dialog, and the presets' menu it keeps; what the screen does once a field's card opens, as asking kit for the caller's login, which @me names; and what it does with the line of the presets'
// menu chosen.
type Overlays[M any] struct {
	Host    func(*M) Host
	Pop     func(*M) *Popover
	Cols    func(*M) *Columns
	Presets func(*M) *PresetMenu
	Popped  func(*M) tea.Cmd
	Choose  func(*M, int)
}

// Layers gives the table's layers, the topmost first.
func (o Overlays[M]) Layers() shell.Layers[M] {
	menu := func(M) shell.Segment { return shell.MenuKeys() }
	return shell.Layers[M]{
		{ // a field's card
			Open: func(m M) bool { return o.Pop(&m).Open },
			Key: func(m M, k tea.KeyPressMsg) (M, tea.Cmd) {
				o.Pop(&m).Key(o.Host(&m), k)
				return m, nil
			},
			Click: func(m M, c tea.MouseClickMsg) (M, tea.Cmd, bool) {
				o.Pop(&m).Click(o.Host(&m), c.X, c.Y)
				return m, nil, true
			},
			Keys: func(m M) shell.Segment { return o.Pop(&m).Keys(o.Host(&m)) },
			Draw: func(m M, lines []string) []string {
				box, top, left := o.Pop(&m).Box(o.Host(&m))
				return shell.Over(lines, box, top, left)
			},
		},
		{ // the fields dialog
			Open: func(m M) bool { return o.Cols(&m).Open },
			Key: func(m M, k tea.KeyPressMsg) (M, tea.Cmd) {
				// enter opens the field's card over it
				if pop, ok := o.Cols(&m).Key(o.Host(&m), k); ok {
					*o.Pop(&m) = pop
					return m, o.Popped(&m)
				}
				return m, nil
			},
			Click: func(m M, c tea.MouseClickMsg) (M, tea.Cmd, bool) {
				// a click on a field's label opens its card over it
				if pop, ok := o.Cols(&m).Click(o.Host(&m), c.X, c.Y, c.Mod&tea.ModShift != 0); ok {
					*o.Pop(&m) = pop
					return m, o.Popped(&m), true
				}
				return m, nil, true
			},
			Keys: func(m M) shell.Segment { return o.Cols(&m).Keys(o.Host(&m)) },
			Draw: func(m M, lines []string) []string {
				box, top, left := o.Cols(&m).Box(o.Host(&m))
				return shell.Over(lines, box, top, left)
			},
		},
		{ // the presets' menu
			Open: func(m M) bool { return o.Presets(&m).Open },
			Key: func(m M, k tea.KeyPressMsg) (M, tea.Cmd) {
				if chosen := o.Presets(&m).Key(o.Host(&m), k); chosen >= 0 {
					o.Choose(&m, chosen)
				}
				return m, nil
			},
			Click: func(m M, c tea.MouseClickMsg) (M, tea.Cmd, bool) {
				if chosen := o.Presets(&m).Click(o.Host(&m), c.X, c.Y); chosen >= 0 {
					o.Choose(&m, chosen)
				}
				return m, nil, true
			},
			Keys: menu,
			Draw: func(m M, lines []string) []string {
				box, top, left := o.Presets(&m).Box(o.Host(&m))
				return shell.Over(lines, box, top, left)
			},
		},
	}
}
