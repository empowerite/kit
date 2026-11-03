// Every step's screen at 120×30, the design width, is pinned as a golden file without its escapes; so is the first
// step with checks, with its failed check left for later.
package welcome

import (
	"path/filepath"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/empowerite/kit/projects/tui/internal/testkit"
)

func TestGoldenViews(t *testing.T) {
	for file, keys := range map[string][]tea.Msg{
		"intro.golden":           {esc, left},
		"basics.golden":          nil,
		"basics-later.golden":    {press('l')},
		"toolchains.golden":      {esc, right, down},
		"setup.golden":           {esc, right, right, down},
		"lay-of-the-land.golden": {esc, right, right, right},
		"terms.golden":           {esc, press('6')},
		"make-it-yours.golden":   {esc, press('7')},
	} {
		m, _ := loaded(t, keys...)
		testkit.Golden(t, filepath.Join("testdata", file), view(m))
	}
}
