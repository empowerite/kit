// Every step's screen at 120×30, the design width, is pinned as a golden file without its escapes; so is Setup with
// its first failed check left for later.
package welcome

import (
	"path/filepath"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/empowerite/kit/tui/internal/testkit"
)

func TestGoldenViews(t *testing.T) {
	for file, keys := range map[string][]tea.Msg{
		"setup.golden":           nil,
		"setup-later.golden":     {press('l')},
		"lay-of-the-land.golden": {esc, right},
		"terms.golden":           {esc, right, right},
		"make-it-yours.golden":   {esc, left},
	} {
		m, _ := loaded(t, keys...)
		testkit.Golden(t, filepath.Join("testdata", file), view(m))
	}
}
