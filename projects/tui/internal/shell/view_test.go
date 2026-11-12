// The frame every screen ends in.
package shell

import (
	"testing"

	tea "charm.land/bubbletea/v2"
)

func TestAScreensViewEndsInItsLastLineOnTheAlternateScreenWithTheMouse(t *testing.T) {
	v := View([]string{"title", "rows"}, 4, "keys")
	if v.Content != "title\nrows\n\nkeys" || !v.AltScreen || v.MouseMode != tea.MouseModeCellMotion {
		t.Errorf("the view: %q, alternate %v, mouse %v", v.Content, v.AltScreen, v.MouseMode)
	}
}
