// The screen's part in the crumbs, as internal/shell's crumb.go says them: the crumb it is told, which the title draws
// last, and whether a click or a key is the way back.
package tutorials

import (
	tea "charm.land/bubbletea/v2"

	"github.com/empowerite/kit/projects/tui/internal/shell"
)

// Crumbed gives the screen told its crumb, by the name of where it goes back to, none for "".
func (m Model) Crumbed(name string) tea.Model {
	m.appMenu.Crumb = name
	return m
}

// covered says whether anything is open over the screen: a menu, or a layer of the table.
func (m Model) covered() bool {
	return m.appMenu.Open || m.recent.Open || m.sheet.Layered()
}

// OnCrumb says whether a click is on the crumb, a left one, with nothing open over the title.
func (m Model) OnCrumb(c tea.MouseClickMsg) bool {
	return c.Button == tea.MouseLeft && !m.covered() && m.menuBar().OnCrumb(c.X, c.Y, m.width)
}

// Backs says whether a key goes back to the crumb: ⌫, while a crumb shows, with nothing open and the search without
// the keyboard.
func (m Model) Backs(k tea.KeyPressMsg) bool {
	return k.String() == shell.BackPress && m.appMenu.Crumb != "" && !m.covered() && !m.search.Focused
}
