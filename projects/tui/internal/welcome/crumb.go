// The screen's part in the crumbs, as internal/shell's crumb.go says them: the crumb it is told, which the title draws
// last, and whether a click or a key is the way back.
package welcome

import (
	"strings"

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
	return m.appMenu.Open || m.books.Open || m.recent.Open || m.tabMenu.Open || (m.gridded() && m.sheet().Layered())
}

// OnCrumb says whether a click is on the crumb, a left one, with nothing open over the title.
func (m Model) OnCrumb(c tea.MouseClickMsg) bool {
	return c.Button == tea.MouseLeft && !m.covered() && m.menuBar().OnCrumb(c.X, c.Y, m.width)
}

// Backs says whether a key goes back to the crumb: ⌫, while a crumb shows, with nothing open and the search without
// the keyboard.
func (m Model) Backs(k tea.KeyPressMsg) bool {
	return k.String() == shell.BackPress && m.appMenu.Crumb != "" && !m.covered() && !m.search().Focused
}

// Spot gives the name of where the screen stands, as the crumb of a screen a link led from says it: welcome, the step
// selected, and what is searched for in it; the screen keeps its place as it was left, so no spot.
func (m Model) Spot() (string, any) {
	name := "welcome"
	if m.step < len(m.tabs) {
		name += " › " + m.tabs[m.step].id
	}
	if q := strings.TrimSuffix(m.search().Query, "/"); q != "" { // a term's search ends in a slash
		name += " › " + q
	}
	return name, nil
}

// Restore leaves the screen as it was left.
func (m Model) Restore(any) tea.Model { return m }
