// The picture's layers: what it opens over itself, in the one order the keys, the clicks, the keys bar, and the drawing
// follow, as shell.Layers says, the topmost first: what a backlog action asks, which takes every key and click until it
// is answered; a field's popover; the fields dialog, which the popover opens over; the presets' menu; the column menu;
// the searches remembered under the search's ▾; the app's menu; the 📚 control's dropdown; and the tab's menu. The two menus that hang from the
// bar take a left click, and leave the others to the screen, as a right click under them groups.
package flow

import (
	tea "charm.land/bubbletea/v2"

	"github.com/empowerite/kit/projects/tui/internal/shell"
	"github.com/empowerite/kit/projects/tui/internal/style"
	"github.com/empowerite/kit/projects/tui/internal/table"
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
			Open: func(m Model) bool { return m.pop.Open },
			Key: func(m Model, k tea.KeyPressMsg) (Model, tea.Cmd) {
				m.pop.Key(&m, k)
				return m, nil
			},
			Click: func(m Model, c tea.MouseClickMsg) (Model, tea.Cmd, bool) {
				m.pop.Click(&m, c.X, c.Y)
				return m, nil, true
			},
			Keys: func(m Model) shell.Segment { return m.pop.Keys(&m) },
			Draw: func(m Model, lines []string) []string {
				box, top, left := m.pop.Box(&m)
				return shell.Over(lines, box, top, left)
			},
		},
		{ // the fields dialog
			Open: func(m Model) bool { return m.cols.Open },
			Key: func(m Model, k tea.KeyPressMsg) (Model, tea.Cmd) {
				// enter opens the field's popover over it, which asks kit for the caller's login
				if pop, ok := m.cols.Key(&m, k); ok {
					m.pop = pop
					return m, m.askMe()
				}
				return m, nil
			},
			Click: func(m Model, c tea.MouseClickMsg) (Model, tea.Cmd, bool) {
				m.cols.Click(&m, c.X, c.Y)
				return m, nil, true
			},
			Keys: func(m Model) shell.Segment { return m.cols.Keys(&m) },
			Draw: func(m Model, lines []string) []string {
				box, top, left := m.cols.Box(&m)
				return shell.Over(lines, box, top, left)
			},
		},
		{ // the presets' menu
			Open: func(m Model) bool { return m.presetsMenu.Open },
			Key: func(m Model, k tea.KeyPressMsg) (Model, tea.Cmd) {
				if chosen := m.presetsMenu.Key(&m, k); chosen >= 0 {
					m.choosePreset(chosen)
				}
				return m, nil
			},
			Click: func(m Model, c tea.MouseClickMsg) (Model, tea.Cmd, bool) {
				if chosen := m.presetsMenu.Click(&m, c.X, c.Y); chosen >= 0 {
					m.choosePreset(chosen)
				}
				return m, nil, true
			},
			Keys: menu,
			Draw: func(m Model, lines []string) []string {
				box, top, left := m.presetsMenu.Box(&m)
				return shell.Over(lines, box, top, left)
			},
		},
		{ // the column menu, which moves its column too
			Open: func(m Model) bool { return m.colMenu.Open },
			Key: func(m Model, k tea.KeyPressMsg) (Model, tea.Cmd) {
				// Options… opens the field's popover, which asks kit for the caller's login
				if pop, ok := m.colMenu.Key(&m, k); ok {
					m.pop = pop
					return m, m.askMe()
				}
				return m, nil
			},
			Click: func(m Model, c tea.MouseClickMsg) (Model, tea.Cmd, bool) {
				if pop, ok := m.colMenu.Click(&m, c.X, c.Y); ok {
					m.pop = pop
					return m, m.askMe(), true
				}
				return m, nil, true
			},
			Keys: func(m Model) shell.Segment {
				keys := shell.MenuKeys()
				keys.Bindings = append(keys.Bindings, table.ShiftKeys(&m, m.colMenu.Column)...)
				return keys
			},
			Draw: func(m Model, lines []string) []string {
				if box, top, left := m.colMenu.Box(&m); len(box) > 0 {
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
		{ // the 📚 control's dropdown, a pick asking kittui to switch
			Open: func(m Model) bool { return m.books.Open },
			Key: func(m Model, k tea.KeyPressMsg) (Model, tea.Cmd) {
				var picked string
				m.books, picked = m.books.Key(k)
				return m, pick(picked)
			},
			Click: func(m Model, c tea.MouseClickMsg) (Model, tea.Cmd, bool) {
				if c.Button != tea.MouseLeft {
					return m, nil, false
				}
				var picked string
				m.books, picked = m.books.Click(c.X, c.Y, m.booksAt(), m.paneWidth())
				return m, pick(picked), true
			},
			Keys: menu,
			Draw: func(m Model, lines []string) []string { return m.books.Overlay(lines, m.booksAt(), m.paneWidth()) },
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

// pick asks kittui to switch to the workspace picked from the 📚 control, when one was.
func pick(workspace string) tea.Cmd {
	if workspace == "" {
		return nil
	}
	return shell.Pick(workspace)
}

// booksAt gives the column the title's 📚 control starts at.
func (m Model) booksAt() int { return shell.BooksAt(shell.Title(style.Label("kit flow"), false)) }

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
