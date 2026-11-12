// A table's layers, as a screen opens them over itself, in the order shell.Layers follows, the topmost first: a field's
// card, which moves its column too where it hangs from the column's name, and widens into the fields dialog; the fields
// dialog, its field's card beside its list; the presets dialog; and the presets' menu. A screen reaches each through its model, so the
// layers are the same on every screen that has a table.
package table

import (
	tea "charm.land/bubbletea/v2"

	"github.com/empowerite/kit/projects/tui/internal/shell"
)

// Overlays are the functions of a screen's model M that reach its table's layers: the host of the table selected;
// the card, the fields dialog, and the presets' menu it keeps; what the screen does once a field's card shows, as asking
// kit for the caller's login, which @me names; what it does with the line of the presets' menu chosen; what it does
// with the line whose view the person stars, or no longer; the presets dialog it keeps; and what it does with what that
// asks.
type Overlays[M any] struct {
	Host    func(*M) Host
	Pop     func(*M) *Popover
	Cols    func(*M) *Columns
	Presets func(*M) *PresetMenu
	Popped  func(*M) tea.Cmd
	Choose  func(*M, int)
	Star    func(*M, int)
	Manage  func(*M) *Manage
	Managed func(*M, ManageDo)
}

// Layers gives the table's layers, the topmost first.
func (o Overlays[M]) Layers() shell.Layers[M] {
	return shell.Layers[M]{
		{ // a field's card
			Open: func(m M) bool { return o.Pop(&m).Open },
			Key: func(m M, k tea.KeyPressMsg) (M, tea.Cmd) {
				// tab widens a card that hangs from what was clicked into the fields dialog
				if id := o.Pop(&m).Key(o.Host(&m), k); id != "" {
					*o.Cols(&m) = OpenColumns(o.Host(&m), id)
				}
				return m, nil
			},
			Click: func(m M, c tea.MouseClickMsg) (M, tea.Cmd, bool) {
				// as does a click on its all ▸
				if id := o.Pop(&m).Click(o.Host(&m), c.X, c.Y); id != "" {
					*o.Cols(&m) = OpenColumns(o.Host(&m), id)
				}
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
				o.Cols(&m).Key(o.Host(&m), k)
				return m, o.Popped(&m)
			},
			Click: func(m M, c tea.MouseClickMsg) (M, tea.Cmd, bool) {
				o.Cols(&m).Click(o.Host(&m), c.X, c.Y, c.Mod&tea.ModShift != 0)
				return m, o.Popped(&m), true
			},
			Keys: func(m M) shell.Segment { return o.Cols(&m).Keys(o.Host(&m)) },
			Draw: func(m M, lines []string) []string {
				box, top, left := o.Cols(&m).Box(o.Host(&m))
				lines = shell.Over(lines, box, top, left)
				card, top, left := o.Cols(&m).CardBox(o.Host(&m))
				return shell.Over(lines, card, top, left)
			},
		},
		{ // the presets dialog
			Open: func(m M) bool { return o.Manage(&m).Open },
			Key: func(m M, k tea.KeyPressMsg) (M, tea.Cmd) {
				o.Managed(&m, o.Manage(&m).Key(o.Host(&m), k))
				return m, nil
			},
			Click: func(m M, c tea.MouseClickMsg) (M, tea.Cmd, bool) {
				o.Managed(&m, o.Manage(&m).Click(o.Host(&m), c.X, c.Y))
				return m, nil, true
			},
			Keys: func(m M) shell.Segment { return o.Manage(&m).Keys(o.Host(&m)) },
			Draw: func(m M, lines []string) []string {
				box, top, left := o.Manage(&m).Box(o.Host(&m))
				return shell.Over(lines, box, top, left)
			},
		},
		{ // the presets' menu, whose * and a click on a star star a view, or no longer
			Open: func(m M) bool { return o.Presets(&m).Open },
			Key: func(m M, k tea.KeyPressMsg) (M, tea.Cmd) {
				chosen, star := o.Presets(&m).Key(o.Host(&m), k)
				o.act(&m, chosen, star)
				return m, nil
			},
			Click: func(m M, c tea.MouseClickMsg) (M, tea.Cmd, bool) {
				chosen, star := o.Presets(&m).Click(o.Host(&m), c.X, c.Y)
				o.act(&m, chosen, star)
				return m, nil, true
			},
			Keys: func(m M) shell.Segment {
				keys := shell.MenuKeys()
				if _, ok := PresetView(o.Host(&m), o.Presets(&m).At); ok {
					keys.Bindings = append(keys.Bindings, shell.Bind("*", "star"))
				}
				return keys
			},
			Draw: func(m M, lines []string) []string {
				box, top, left := o.Presets(&m).Box(o.Host(&m))
				return shell.Over(lines, box, top, left)
			},
		},
	}
}

// act does what the presets' menu asked: the line chosen, as the screen's Choose does, or the view starred, as its
// Star does; -1 asks for neither.
func (o Overlays[M]) act(m *M, chosen, star int) {
	switch {
	case chosen >= 0:
		o.Choose(m, chosen)
	case star >= 0:
		o.Star(m, star)
	}
}
