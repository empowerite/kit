// Every step's screen at 120×30, the design width, is pinned as a golden file without its escapes; so are the intro
// with the cursor on its second link, and the first step with checks, as its first link opens it, with its failed
// check left for later.
package welcome

import (
	"path/filepath"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/empowerite/kit/projects/tui/internal/testkit"
)

func TestGoldenViews(t *testing.T) {
	for file, keys := range map[string][]tea.Msg{
		"intro.golden":           nil,
		"intro-links.golden":     {down, down},
		"basics.golden":          {down, enter},
		"basics-later.golden":    {down, enter, press('l')},
		"toolchains.golden":      {right, right, down},
		"setup.golden":           {right, right, right, down},
		"lay-of-the-land.golden": {right, right, right, right},
		"terms.golden":           {press('6')},
		"make-it-yours.golden":   {press('7')},
	} {
		m, _ := loaded(t, keys...)
		testkit.Golden(t, filepath.Join("testdata", file), view(m))
	}
}
