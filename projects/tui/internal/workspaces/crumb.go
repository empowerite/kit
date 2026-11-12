// The screen's part in the crumbs, as internal/shell's crumb.go says them: the crumb it is told, which the title draws
// last, and whether a click or a key is the way back. What the screen shows is kept as it was left, so kittui's root
// keeps nothing more of it.
package workspaces

import (
	tea "charm.land/bubbletea/v2"

	"github.com/empowerite/kit/projects/tui/internal/shell"
)

// Crumbed gives the screen told its crumb, by the name of where it goes back to, none for "".
func (m Model) Crumbed(name string) tea.Model {
	m.appMenu.Crumb = name
	return m
}

// open says whether anything is open over the screen: a prompt, a menu, or the table's own layer.
func (m Model) open() bool {
	return m.prompt.open || m.appMenu.Open || m.books.Open || m.sheet.Layered()
}

// OnCrumb says whether a click is on the crumb, a left one, with nothing open over the title.
func (m Model) OnCrumb(c tea.MouseClickMsg) bool {
	return c.Button == tea.MouseLeft && !m.open() && m.menuBar().OnCrumb(c.X, c.Y, m.width)
}

// Backs says whether a key goes back to the crumb: ⌫, while a crumb shows, with nothing open and the search without
// the keyboard.
func (m Model) Backs(k tea.KeyPressMsg) bool {
	return k.String() == shell.BackPress && m.appMenu.Crumb != "" && !m.open() && !m.search.Focused
}
