// The picture's layers: what it opens over itself, in the one order the keys, the clicks, the keys bar, and the drawing
// follow, as shell.Layers says, the topmost first: what a backlog action asks, which takes every key and click until it
// is answered; a field's popover; the fields dialog, which the popover opens over; the presets' menu; the column menu;
// the searches remembered under the search's ▾; the app's menu; and the tab's menu. The two menus that hang from the
// bar take a left click, and leave the others to the screen, as a right click under them groups.
package flow

import (
	tea "charm.land/bubbletea/v2"

	"github.com/empowerite/kit/projects/tui/internal/shell"
)

// layers gives the picture's layers, the topmost first.
func (m Model) layers() shell.Layers[Model] {
	menu := func(Model) shell.Segment { return shell.MenuKeys() }
	return shell.Layers[Model]{
		{ // what a backlog action asks
			Open: func(m Model) bool { return m.ask != nil },
			Key:  model(Model.askKey),
			Keys: func(m Model) shell.Segment {
				_, keys := m.askKeys()
				return shell.DialogKeys(keys)
			},
			Draw: func(m Model, lines []string) []string {
				box, left := m.askBox()
				return shell.Over(lines, box, m.top()+1, left)
			},
		},
		{ // a field's popover
			Open: func(m Model) bool { return m.pop.open },
			Key:  model(Model.popKey),
			Click: func(m Model, c tea.MouseClickMsg) (Model, tea.Cmd, bool) {
				m.popClick(c.X, c.Y)
				return m, nil, true
			},
			Keys: Model.popKeys,
			Draw: func(m Model, lines []string) []string {
				box, top, left := m.popBox()
				return shell.Over(lines, box, top, left)
			},
		},
		{ // the fields dialog
			Open: func(m Model) bool { return m.cols.open },
			Key:  model(Model.columnsKey),
			Click: func(m Model, c tea.MouseClickMsg) (Model, tea.Cmd, bool) {
				m.columnsClick(c.X, c.Y)
				return m, nil, true
			},
			Keys: Model.columnsKeys,
			Draw: func(m Model, lines []string) []string {
				box, top, left := m.columnsBox()
				return shell.Over(lines, box, top, left)
			},
		},
		{ // the presets' menu
			Open: func(m Model) bool { return m.presetsMenu.Open },
			Key:  model(Model.presetMenuKey),
			Click: func(m Model, c tea.MouseClickMsg) (Model, tea.Cmd, bool) {
				m.presetMenuClick(c.X, c.Y)
				return m, nil, true
			},
			Keys: menu,
			Draw: func(m Model, lines []string) []string {
				box, top, left := m.presetBox()
				return shell.Over(lines, box, top, left)
			},
		},
		{ // the column menu, which moves its column too
			Open: func(m Model) bool { return m.colMenu.Open },
			Key:  model(Model.columnKey),
			Click: func(m Model, c tea.MouseClickMsg) (Model, tea.Cmd, bool) {
				cmd := m.columnClick(c.X, c.Y)
				return m, cmd, true
			},
			Keys: func(m Model) shell.Segment {
				keys := shell.MenuKeys()
				keys.Bindings = append(keys.Bindings, m.shiftKeys(m.colMenu.column)...)
				return keys
			},
			Draw: func(m Model, lines []string) []string {
				if box, top, left := m.columnBox(); len(box) > 0 {
					return shell.Over(lines, box, top, left)
				}
				return lines
			},
		},
		{ // the searches remembered, one chosen searched again
			Open: func(m Model) bool { return m.recent.Open },
			Key: func(m Model, k tea.KeyPressMsg) (Model, tea.Cmd) {
				var query string
				m.recent, query = m.recent.Key(k)
				m.searchAgain(query)
				return m, nil
			},
			Click: func(m Model, c tea.MouseClickMsg) (Model, tea.Cmd, bool) {
				var query string
				m.recent, query = m.recent.Click(c.X, c.Y, m.recallAt(), m.width)
				m.searchAgain(query)
				return m, nil, true
			},
			Keys: menu,
			Draw: func(m Model, lines []string) []string { return m.recent.Overlay(lines, m.recallAt(), m.width) },
		},
		{ // the app's menu
			Open: func(m Model) bool { return m.appMenu.Open },
			Key:  model(Model.appKey),
			Click: func(m Model, c tea.MouseClickMsg) (Model, tea.Cmd, bool) {
				if c.Button != tea.MouseLeft {
					return m, nil, false
				}
				var choice shell.Choice
				m.appMenu, choice = m.appMenu.Click(c.X, c.Y, m.paneWidth())
				cmd := m.chose(choice)
				return m, cmd, true
			},
			Keys: menu,
			Draw: func(m Model, lines []string) []string { return m.appMenu.Overlay(lines, m.paneWidth()) },
		},
		{ // the tab's menu
			Open: func(m Model) bool { return m.tabMenu.Open },
			Key:  model(Model.menuKey),
			Click: func(m Model, c tea.MouseClickMsg) (Model, tea.Cmd, bool) {
				if c.Button != tea.MouseLeft {
					return m, nil, false
				}
				cmd := m.menuClick(c.X, c.Y)
				return m, cmd, true
			},
			Keys: menu,
			Draw: func(m Model, lines []string) []string {
				box, left := m.menuBox()
				return shell.Over(lines, box, m.menuTop(), left)
			},
		},
	}
}

// model gives a layer's key from one of the picture's own, which gives the picture as a tea.Model.
func model(key func(Model, tea.KeyPressMsg) (tea.Model, tea.Cmd)) func(Model, tea.KeyPressMsg) (Model, tea.Cmd) {
	return func(m Model, k tea.KeyPressMsg) (Model, tea.Cmd) {
		n, cmd := key(m, k)
		return n.(Model), cmd
	}
}

// searchAgain searches the rows for a search remembered and chosen again; none chosen leaves the search as it is.
func (m *Model) searchAgain(query string) {
	if query == "" {
		return
	}
	p := m.panel(m.stage)
	p.Query = query
	m.setPanel(p)
}
