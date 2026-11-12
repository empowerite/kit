// The frame every screen ends in: its lines, cut or padded to the screen's height but one, and the last line under
// them, the keys bar or what a notice says in its place; on the alternate screen, with the mouse on for clicks.
package shell

import (
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/empowerite/kit/projects/tui/internal/style"
)

// View gives a screen's view: its lines over the last line, as style.Bottom puts them on a screen so high, on the
// alternate screen, with the mouse on.
func View(lines []string, height int, last string) tea.View {
	v := tea.NewView(strings.Join(style.Bottom(lines, height, last), "\n"))
	v.AltScreen = true
	v.MouseMode = tea.MouseModeCellMotion
	return v
}
