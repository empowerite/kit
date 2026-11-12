// The help's part in the crumbs, as internal/shell's crumb.go says them: its palette draws the crumb; a form opened from
// it draws none, and has no way back to one.
package app

import (
	tea "charm.land/bubbletea/v2"

	"github.com/empowerite/kit/projects/tui/internal/palette"
)

// Crumbed gives the screen told its crumb, by the name of where it goes back to, none for "", which its palette draws.
func (m Model) Crumbed(name string) tea.Model {
	if m.hasPalette {
		m.palette = m.palette.Crumbed(name).(palette.Model)
	}
	return m
}

// OnCrumb says whether a click is on the crumb, which the palette draws, not under a form.
func (m Model) OnCrumb(c tea.MouseClickMsg) bool {
	return !m.inForm && m.hasPalette && m.palette.OnCrumb(c)
}

// Backs says whether a key goes back to the crumb: as the palette says, not under a form.
func (m Model) Backs(k tea.KeyPressMsg) bool {
	return !m.inForm && m.hasPalette && m.palette.Backs(k)
}

// Spot gives the name of where the screen stands, as the crumb of a screen a link led from says it: help; the screen
// keeps its place as it was left, so no spot.
func (m Model) Spot() (string, any) { return "help", nil }

// Restore leaves the screen as it was left.
func (m Model) Restore(any) tea.Model { return m }
