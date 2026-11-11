// Every step's screen at 120×30, the design width, but make it yours's, which yours_test.go pins with its others, is
// pinned as a golden file without its escapes; so are the intro with the cursor on its second link, the first step
// with checks, as its first link opens it, with its failed check left for later, and the terms searched for kit flow,
// the rows with the keyboard.
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
		"toolchains.golden":      {tabKey, tabKey, down},
		"setup.golden":           {tabKey, tabKey, tabKey, down},
		"lay-of-the-land.golden": {tabKey, tabKey, tabKey, tabKey},
		"terms.golden":           {press('6')},
		"terms-search.golden":    searched("kit flow"),
	} {
		m, _ := loaded(t, keys...)
		testkit.Golden(t, filepath.Join("testdata", file), view(m))
	}
}

// searched gives the keys that select the terms and search them for the text, then give their rows the keyboard.
func searched(text string) []tea.Msg {
	return append([]tea.Msg{press('6'), press('/')}, append(testkit.Typed(text), enter)...)
}
